
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package indexers

import (
	"errors"
	"fmt"
	"sync"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

const (
//
	addrIndexName = "address index"

//
//
//
//
	level0MaxEntries = 8

//
//
	addrKeySize = 1 + 20

//
//
	levelKeySize = addrKeySize + 1

//
	levelOffset = levelKeySize - 1

//
//表示支付到公钥哈希和支付到公钥地址。
//这样做是因为两者在
//
	addrKeyTypePubKeyHash = 0

//AddRkeyTypeScriptHash是地址键中的地址类型，它
//
//
//搞砸。
	addrKeyTypeScriptHash = 1

//
//
//
//
	addrKeyTypeWitnessPubKeyHash = 2

//AddRkeyTypeScriptHash是地址键中的地址类型，它
//
//
//
	addrKeyTypeWitnessScriptHash = 3

//Size of a transaction entry.  It consists of 4 bytes block id + 4
//bytes offset + 4 bytes length.
	txEntrySize = 4 + 4 + 4
)

var (
//addrindexkey是地址索引的键，使用了db bucket
//把它盖起来。
	addrIndexKey = []byte("txbyaddridx")

//
//
	errUnsupportedAddressType = errors.New("address type is not supported " +
		"by the address index")
)

//————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//
//所有涉及该地址的交易。存储事务
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//序列化密钥格式为：
//
//
//
//
//
//
//
//-----
//
//
//序列化值格式为：
//
//
//
//字段类型大小
//块ID uint32 4字节
//起始偏移量uint32 4字节
//
//-----
//
//————————————————————————————————————————————————————————————————————————————————————————————————————————————————

//fetchblockhashfunc定义用于转换
//将块ID序列化为关联的块哈希。
type fetchBlockHashFunc func(serializedID []byte) (*chainhash.Hash, error)

//
//按照上述详细描述的格式定位。
func serializeAddrIndexEntry(blockID uint32, txLoc wire.TxLoc) []byte {
//
	serialized := make([]byte, 12)
	byteOrder.PutUint32(serialized, blockID)
	byteOrder.PutUint32(serialized[4:], uint32(txLoc.TxStart))
	byteOrder.PutUint32(serialized[8:], uint32(txLoc.TxLen))
	return serialized
}

//
//
//
//
func deserializeAddrIndexEntry(serialized []byte, region *database.BlockRegion, fetchBlockHash fetchBlockHashFunc) error {
//
	if len(serialized) < txEntrySize {
		return errDeserialize("unexpected end of data")
	}

	hash, err := fetchBlockHash(serialized[0:4])
	if err != nil {
		return err
	}
	region.Hash = hash
	region.Offset = byteOrder.Uint32(serialized[4:8])
	region.Len = byteOrder.Uint32(serialized[8:12])
	return nil
}

//
//
func keyForLevel(addrKey [addrKeySize]byte, level uint8) [levelKeySize]byte {
	var key [levelKeySize]byte
	copy(key[:], addrKey[:])
	key[levelOffset] = level
	return key
}

//
//根据上面详细描述的基于级别的方案。
func dbPutAddrIndexEntry(bucket internalBucket, addrKey [addrKeySize]byte, blockID uint32, txLoc wire.TxLoc) error {
//
	curLevel := uint8(0)
	maxLevelBytes := level0MaxEntries * txEntrySize

//
//
	newData := serializeAddrIndexEntry(blockID, txLoc)
	level0Key := keyForLevel(addrKey, 0)
	level0Data := bucket.Get(level0Key[:])
	if len(level0Data)+len(newData) <= maxLevelBytes {
		mergedData := newData
		if len(level0Data) > 0 {
			mergedData = make([]byte, len(level0Data)+len(newData))
			copy(mergedData, level0Data)
			copy(mergedData[len(level0Data):], newData)
		}
		return bucket.Put(level0Key[:], mergedData)
	}

//
//
	prevLevelData := level0Data
	for {
//
		curLevel++
		maxLevelBytes *= 2

//
		curLevelKey := keyForLevel(addrKey, curLevel)
		curLevelData := bucket.Get(curLevelKey[:])
		if len(curLevelData) == maxLevelBytes {
			prevLevelData = curLevelData
			continue
		}

//
//
		mergedData := prevLevelData
		if len(curLevelData) > 0 {
			mergedData = make([]byte, len(curLevelData)+
				len(prevLevelData))
			copy(mergedData, curLevelData)
			copy(mergedData[len(curLevelData):], prevLevelData)
		}
		err := bucket.Put(curLevelKey[:], mergedData)
		if err != nil {
			return err
		}

//
		for mergeLevel := curLevel - 1; mergeLevel > 0; mergeLevel-- {
			mergeLevelKey := keyForLevel(addrKey, mergeLevel)
			prevLevelKey := keyForLevel(addrKey, mergeLevel-1)
			prevData := bucket.Get(prevLevelKey[:])
			err := bucket.Put(mergeLevelKey[:], prevData)
			if err != nil {
				return err
			}
		}
		break
	}

//
	return bucket.Put(level0Key[:], newData)
}

//
//
//
//
func dbFetchAddrIndexEntries(bucket internalBucket, addrKey [addrKeySize]byte, numToSkip, numRequested uint32, reverse bool, fetchBlockHash fetchBlockHashFunc) ([]database.BlockRegion, uint32, error) {
//如果未设置反向标志，则需要获取所有级别。
//因为numtoskip和numrequested是从最旧的开始计数的
//事务（最高级别），因此需要总计数。
//但是，当设置了反向标志时，只有足够的记录来满足
//需要请求的金额。
	var level uint8
	var serialized []byte
	for !reverse || len(serialized) < int(numToSkip+numRequested)*txEntrySize {
		curLevelKey := keyForLevel(addrKey, level)
		levelData := bucket.Get(curLevelKey[:])
		if levelData == nil {
//当没有更多级别时停止。
			break
		}

//更高的级别包含较旧的事务，因此请提前准备它们。
		prepended := make([]byte, len(serialized)+len(levelData))
		copy(prepended, levelData)
		copy(prepended[len(levelData):], serialized)
		serialized = prepended
		level++
	}

//当要跳过的请求条目数大于
//可用数字，全部跳过，现在返回实际数字
//跳过。
	numEntries := uint32(len(serialized) / txEntrySize)
	if numToSkip >= numEntries {
		return nil, numEntries, nil
	}

//如果没有请求的条目，则无需执行其他操作。
	if numRequested == 0 {
		return nil, numToSkip, nil
	}

//根据可用条目数限制要加载的数目，
//要跳过的号码和请求的号码。
	numToLoad := numEntries - numToSkip
	if numToLoad > numRequested {
		numToLoad = numRequested
	}

//在所有跳过的条目之后启动偏移量并加载计算的
//号码。
	results := make([]database.BlockRegion, numToLoad)
	for i := uint32(0); i < numToLoad; i++ {
//根据反向标志计算读取偏移量。
		var offset uint32
		if reverse {
			offset = (numEntries - numToSkip - i - 1) * txEntrySize
		} else {
			offset = (numToSkip + i) * txEntrySize
		}

//
		err := deserializeAddrIndexEntry(serialized[offset:],
			&results[i], fetchBlockHash)
		if err != nil {
//确保任何反序列化错误返回为
//
			if isDeserializeErr(err) {
				err = database.Error{
					ErrorCode: database.ErrCorruption,
					Description: fmt.Sprintf("failed to "+
						"deserialized address index "+
						"for key %x: %v", addrKey, err),
				}
			}

			return nil, 0, err
		}
	}

	return results, numToSkip, nil
}

//
//
func minEntriesToReachLevel(level uint8) int {
	maxEntriesForLevel := level0MaxEntries
	minRequired := 1
	for l := uint8(1); l <= level; l++ {
		minRequired += maxEntriesForLevel
		maxEntriesForLevel *= 2
	}
	return minRequired
}

//
//
func maxEntriesForLevel(level uint8) int {
	numEntries := level0MaxEntries
	for l := level; l > 0; l-- {
		numEntries *= 2
	}
	return numEntries
}

//
//
//
func dbRemoveAddrIndexEntries(bucket internalBucket, addrKey [addrKeySize]byte, count int) error {
//
	if count <= 0 {
		return nil
	}

//
//
//
//
	pendingUpdates := make(map[uint8][]byte)
	applyPending := func() error {
		for level, data := range pendingUpdates {
			curLevelKey := keyForLevel(addrKey, level)
			if len(data) == 0 {
				err := bucket.Delete(curLevelKey[:])
				if err != nil {
					return err
				}
				continue
			}
			err := bucket.Put(curLevelKey[:], data)
			if err != nil {
				return err
			}
		}
		return nil
	}

//
//指定的号码已被删除。这可能导致
//
	var highestLoadedLevel uint8
	numRemaining := count
	for level := uint8(0); numRemaining > 0; level++ {
//
		curLevelKey := keyForLevel(addrKey, level)
		curLevelData := bucket.Get(curLevelKey[:])
		if len(curLevelData) == 0 && numRemaining > 0 {
			return AssertError(fmt.Sprintf("dbRemoveAddrIndexEntries "+
				"not enough entries for address key %x to "+
				"delete %d entries", addrKey, count))
		}
		pendingUpdates[level] = curLevelData
		highestLoadedLevel = level

//
		numEntries := len(curLevelData) / txEntrySize
		if numRemaining >= numEntries {
			pendingUpdates[level] = nil
			numRemaining -= numEntries
			continue
		}

//
		offsetEnd := len(curLevelData) - (numRemaining * txEntrySize)
		pendingUpdates[level] = curLevelData[:offsetEnd]
		break
	}

//
//
	if len(pendingUpdates[0]) != 0 {
		return applyPending()
	}

//
//需要回填的级别，当前级别可能
//
//
//
//
//
//
//有效地将当前级别中的所有剩余项压缩到
//
//
//请注意，当前级别之后的级别也可能有条目
//而且不允许有间隙，所以这也保持了最低的
//空级别，以便下面的代码知道在这种情况下要回填多远
//必修的。
	lowestEmptyLevel := uint8(255)
	curLevelData := pendingUpdates[highestLoadedLevel]
	curLevelMaxEntries := maxEntriesForLevel(highestLoadedLevel)
	for level := highestLoadedLevel; level > 0; level-- {
//当当前级别中没有足够的条目时
//对于需要到达的号码，清除
//有效地将它们全部移动到
//下一次迭代的上一级。否则，有
//
//在离开时尽可能多地包含条目
//
		numEntries := len(curLevelData) / txEntrySize
		prevLevelMaxEntries := curLevelMaxEntries / 2
		minPrevRequired := minEntriesToReachLevel(level - 1)
		if numEntries < prevLevelMaxEntries+minPrevRequired {
			lowestEmptyLevel = level
			pendingUpdates[level] = nil
		} else {
//
//
//
			var offset int
			if numEntries-curLevelMaxEntries >= minPrevRequired {
				offset = curLevelMaxEntries * txEntrySize
			} else {
				offset = prevLevelMaxEntries * txEntrySize
			}
			pendingUpdates[level] = curLevelData[:offset]
			curLevelData = curLevelData[offset:]
		}

		curLevelMaxEntries = prevLevelMaxEntries
	}
	pendingUpdates[0] = curLevelData
	if len(curLevelData) == 0 {
		lowestEmptyLevel = 0
	}

//
//
	for len(pendingUpdates[highestLoadedLevel]) == 0 {
//
//
//
//
		level := highestLoadedLevel + 1
		curLevelKey := keyForLevel(addrKey, level)
		levelData := bucket.Get(curLevelKey[:])
		if len(levelData) == 0 {
			break
		}
		pendingUpdates[level] = levelData
		highestLoadedLevel = level

//
//
//
//
//
//
		curLevelMaxEntries := maxEntriesForLevel(level)
		if len(levelData)/txEntrySize != curLevelMaxEntries {
			pendingUpdates[level] = nil
			pendingUpdates[level-1] = levelData
			level--
			curLevelMaxEntries /= 2
		}

//
//
		for level > lowestEmptyLevel {
			offset := (curLevelMaxEntries / 2) * txEntrySize
			pendingUpdates[level] = levelData[:offset]
			levelData = levelData[offset:]
			pendingUpdates[level-1] = levelData
			level--
			curLevelMaxEntries /= 2
		}

//
//水平。
		lowestEmptyLevel = highestLoadedLevel
	}

//
	return applyPending()
}

//
//
func addrToKey(addr btcutil.Address) ([addrKeySize]byte, error) {
	switch addr := addr.(type) {
	case *btcutil.AddressPubKeyHash:
		var result [addrKeySize]byte
		result[0] = addrKeyTypePubKeyHash
		copy(result[1:], addr.Hash160()[:])
		return result, nil

	case *btcutil.AddressScriptHash:
		var result [addrKeySize]byte
		result[0] = addrKeyTypeScriptHash
		copy(result[1:], addr.Hash160()[:])
		return result, nil

	case *btcutil.AddressPubKey:
		var result [addrKeySize]byte
		result[0] = addrKeyTypePubKeyHash
		copy(result[1:], addr.AddressPubKeyHash().Hash160()[:])
		return result, nil

	case *btcutil.AddressWitnessScriptHash:
		var result [addrKeySize]byte
		result[0] = addrKeyTypeWitnessScriptHash

//
//
//
//
//
		copy(result[1:], btcutil.Hash160(addr.ScriptAddress()))
		return result, nil

	case *btcutil.AddressWitnessPubKeyHash:
		var result [addrKeySize]byte
		result[0] = addrKeyTypeWitnessPubKeyHash
		copy(result[1:], addr.Hash160()[:])
		return result, nil
	}

	return [addrKeySize]byte{}, errUnsupportedAddressType
}

//
//
//
//
//
//
//
//
//
type AddrIndex struct {
//以下字段是在创建实例时设置的，不能
//之后再更改，因此无需使用
//单独互斥。
	db          database.DB
	chainParams *chaincfg.Params

//
//
//
//
//
//
//
//
//
//
//
//
//
	unconfirmedLock sync.RWMutex
	txnsByAddr      map[[addrKeySize]byte]map[chainhash.Hash]*btcutil.Tx
	addrsByTx       map[chainhash.Hash]map[[addrKeySize]byte]struct{}
}

//
var _ Indexer = (*AddrIndex)(nil)

//
var _ NeedsInputser = (*AddrIndex)(nil)

//
//
//
//
func (idx *AddrIndex) NeedsInputs() bool {
	return true
}

//
//
//
//
func (idx *AddrIndex) Init() error {
//
	return nil
}

//
//
//
func (idx *AddrIndex) Key() []byte {
	return addrIndexKey
}

//
//
//
func (idx *AddrIndex) Name() string {
	return addrIndexName
}

//
//
//索引。
//
//
func (idx *AddrIndex) Create(dbTx database.Tx) error {
	_, err := dbTx.Metadata().CreateBucket(addrIndexKey)
	return err
}

//
//
//包括地址块。它是按顺序排列的，这样交易就可以
//
type writeIndexData map[[addrKeySize]byte][]int

//
//
//地图。
func (idx *AddrIndex) indexPkScript(data writeIndexData, pkScript []byte, txIdx int) {
//
//
	_, addrs, _, err := txscript.ExtractPkScriptAddrs(pkScript,
		idx.chainParams)
	if err != nil || len(addrs) == 0 {
		return
	}

	for _, addr := range addrs {
		addrKey, err := addrToKey(addr)
		if err != nil {
//
			continue
		}

//
//
//
//
		indexedTxns := data[addrKey]
		numTxns := len(indexedTxns)
		if numTxns > 0 && indexedTxns[numTxns-1] == txIdx {
			continue
		}
		indexedTxns = append(indexedTxns, txIdx)
		data[addrKey] = indexedTxns
	}
}

//
//
//通过的地图。
func (idx *AddrIndex) indexBlock(data writeIndexData, block *btcutil.Block,
	stxos []blockchain.SpentTxOut) {

	stxoIndex := 0
	for txIdx, tx := range block.Transactions() {
//
//
//已经在块中的第一个事务上得到证明
//
		if txIdx != 0 {
			for range tx.MsgTx().TxIn {
//
//在此块中正确花费的事务
//命令获取上一个输入脚本。
				pkScript := stxos[stxoIndex].PkScript
				idx.indexPkScript(data, pkScript, txIdx)

//
//斯克索克特纳
				stxoIndex++
			}
		}

		for _, txOut := range tx.MsgTx().TxOut {
			idx.indexPkScript(data, txOut.PkScript, txIdx)
		}
	}
}

//
//
//
//
//这是索引器接口的一部分。
func (idx *AddrIndex) ConnectBlock(dbTx database.Tx, block *btcutil.Block,
	stxos []blockchain.SpentTxOut) error {

//
//块。
	txLocs, err := block.TxLoc()
	if err != nil {
		return err
	}

//获取与块关联的内部块ID。
	blockID, err := dbFetchBlockIDByHash(dbTx, block.Hash())
	if err != nil {
		return err
	}

//在本地映射中构建所有地址到事务的映射。
	addrsToTxns := make(writeIndexData)
	idx.indexBlock(addrsToTxns, block, stxos)

//为每个地址添加所有索引项。
	addrIdxBucket := dbTx.Metadata().Bucket(addrIndexKey)
	for addrKey, txIdxs := range addrsToTxns {
		for _, txIdx := range txIdxs {
			err := dbPutAddrIndexEntry(addrIdxBucket, addrKey,
				blockID, txLocs[txIdx])
			if err != nil {
				return err
			}
		}
	}

	return nil
}

//当一个块被
//从主链上断开。此索引器删除地址映射
//块中的每个事务都涉及。
//
//这是索引器接口的一部分。
func (idx *AddrIndex) DisconnectBlock(dbTx database.Tx, block *btcutil.Block,
	stxos []blockchain.SpentTxOut) error {

//在本地映射中构建所有地址到事务的映射。
	addrsToTxns := make(writeIndexData)
	idx.indexBlock(addrsToTxns, block, stxos)

//删除每个地址的所有索引项。
	bucket := dbTx.Metadata().Bucket(addrIndexKey)
	for addrKey, txIdxs := range addrsToTxns {
		err := dbRemoveAddrIndexEntries(bucket, addrKey, len(txIdxs))
		if err != nil {
			return err
		}
	}

	return nil
}

//TxRegionsForAddress返回一个块区域切片，每个块区域标识
//
//要跳过的编号、请求的编号以及结果是否应为
//颠倒的。它还返回实际跳过的数字，因为它可能小于
//如果没有足够的条目。
//
//注意：这些结果只包括以块形式确认的交易。见
//
//
//
//此函数对于并发访问是安全的。
func (idx *AddrIndex) TxRegionsForAddress(dbTx database.Tx, addr btcutil.Address, numToSkip, numRequested uint32, reverse bool) ([]database.BlockRegion, uint32, error) {
	addrKey, err := addrToKey(addr)
	if err != nil {
		return nil, 0, err
	}

	var regions []database.BlockRegion
	var skipped uint32
	err = idx.db.View(func(dbTx database.Tx) error {
//
//
		fetchBlockHash := func(id []byte) (*chainhash.Hash, error) {
//
			return dbFetchBlockHashBySerializedID(dbTx, id)
		}

		var err error
		addrIdxBucket := dbTx.Metadata().Bucket(addrIndexKey)
		regions, skipped, err = dbFetchAddrIndexEntries(addrIdxBucket,
			addrKey, numToSkip, numRequested, reverse,
			fetchBlockHash)
		return err
	})

	return regions, skipped, err
}

//
//
//
//
//此函数对于并发访问是安全的。
func (idx *AddrIndex) indexUnconfirmedAddresses(pkScript []byte, tx *btcutil.Tx) {
//
//
//
	_, addresses, _, _ := txscript.ExtractPkScriptAddrs(pkScript,
		idx.chainParams)
	for _, addr := range addresses {
//
		addrKey, err := addrToKey(addr)
		if err != nil {
			continue
		}

//
		idx.unconfirmedLock.Lock()
		addrIndexEntry := idx.txnsByAddr[addrKey]
		if addrIndexEntry == nil {
			addrIndexEntry = make(map[chainhash.Hash]*btcutil.Tx)
			idx.txnsByAddr[addrKey] = addrIndexEntry
		}
		addrIndexEntry[*tx.Hash()] = tx

//
		addrsByTxEntry := idx.addrsByTx[*tx.Hash()]
		if addrsByTxEntry == nil {
			addrsByTxEntry = make(map[[addrKeySize]byte]struct{})
			idx.addrsByTx[*tx.Hash()] = addrsByTxEntry
		}
		addrsByTxEntry[addrKey] = struct{}{}
		idx.unconfirmedLock.Unlock()
	}
}

//
//
//
//
//
//
//
//
//此函数对于并发访问是安全的。
func (idx *AddrIndex) AddUnconfirmedTx(tx *btcutil.Tx, utxoView *blockchain.UtxoViewpoint) {
//
//
//
//
//已知存在。
	for _, txIn := range tx.MsgTx().TxIn {
		entry := utxoView.LookupEntry(txIn.PreviousOutPoint)
		if entry == nil {
//
//
//呼叫所有输入必须可用。
			continue
		}
		idx.indexUnconfirmedAddresses(entry.PkScript(), tx)
	}

//所有创建输出的索引地址。
	for _, txOut := range tx.MsgTx().TxOut {
		idx.indexUnconfirmedAddresses(txOut.PkScript, tx)
	}
}

//removeunconfirmedtx从未确认的事务中删除传递的事务
//（仅内存）地址索引。
//
//此函数对于并发访问是安全的。
func (idx *AddrIndex) RemoveUnconfirmedTx(hash *chainhash.Hash) {
	idx.unconfirmedLock.Lock()
	defer idx.unconfirmedLock.Unlock()

//
//
//引用任何交易。
	for addrKey := range idx.addrsByTx[*hash] {
		delete(idx.txnsByAddr[addrKey], *hash)
		if len(idx.txnsByAddr[addrKey]) == 0 {
			delete(idx.txnsByAddr, addrKey)
		}
	}

//
	delete(idx.addrsByTx, *hash)
}

//
//涉及传递地址的未确认（仅内存）地址索引。
//
//
//此函数对于并发访问是安全的。
func (idx *AddrIndex) UnconfirmedTxnsForAddress(addr btcutil.Address) []*btcutil.Tx {
//
	addrKey, err := addrToKey(addr)
	if err != nil {
		return nil
	}

//保护并发访问。
	idx.unconfirmedLock.RLock()
	defer idx.unconfirmedLock.RUnlock()

//
//安全并发。
	if txns, exists := idx.txnsByAddr[addrKey]; exists {
		addressTxns := make([]*btcutil.Tx, 0, len(txns))
		for _, tx := range txns {
			addressTxns = append(addressTxns, tx)
		}
		return addressTxns
	}

	return nil
}

//
//
//这涉及到他们。
//
//
//
//
func NewAddrIndex(db database.DB, chainParams *chaincfg.Params) *AddrIndex {
	return &AddrIndex{
		db:          db,
		chainParams: chainParams,
		txnsByAddr:  make(map[[addrKeySize]byte]map[chainhash.Hash]*btcutil.Tx),
		addrsByTx:   make(map[chainhash.Hash]map[[addrKeySize]byte]struct{}),
	}
}

//
//存在。
func DropAddrIndex(db database.DB, interrupt <-chan struct{}) error {
	return dropIndex(db, addrIndexKey, addrIndexName, interrupt)
}
