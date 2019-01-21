
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package blockchain

import (
	"bytes"
	"container/list"
	"errors"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcd/wire"
)

const (
//blockhdroffset将偏移量定义为v1块索引行
//块标题。
//
//序列化块索引行格式为：
//<blocklocation><blockheader>
	blockHdrOffset = 12
)

//errInterruptRequested指示由于
//用户请求的中断。
var errInterruptRequested = errors.New("interrupt requested")

//当提供的通道关闭时，interruptrequested返回true。
//这稍微简化了早期关闭，因为调用方可以使用if
//语句而不是select。
func interruptRequested(interrupted <-chan struct{}) bool {
	select {
	case <-interrupted:
		return true
	default:
	}

	return false
}

//BlockChainContext表示特定块在块中的位置
//链。这是块索引迁移用来跟踪块元数据的
//将写入磁盘。
type blockChainContext struct {
	parent    *chainhash.Hash
	children  []*chainhash.Hash
	height    int32
	mainChain bool
}

//migrateblockindex从v1块索引桶中迁移所有块条目
//到v2桶。v1存储桶存储由块散列键控的所有块条目，
//而v2存储桶存储的值完全相同，但键入的是
//块高度+哈希。
func migrateBlockIndex(db database.DB) error {
//
//旧升级。
	v1BucketName := []byte("ffldb-blockidx")
	v2BucketName := []byte("blockheaderidx")

	err := db.Update(func(dbTx database.Tx) error {
		v1BlockIdxBucket := dbTx.Metadata().Bucket(v1BucketName)
		if v1BlockIdxBucket == nil {
			return fmt.Errorf("Bucket %s does not exist", v1BucketName)
		}

		log.Info("Re-indexing block information in the database. This might take a while...")

		v2BlockIdxBucket, err :=
			dbTx.Metadata().CreateBucketIfNotExists(v2BucketName)
		if err != nil {
			return err
		}

//得到主链的顶端。
		serializedData := dbTx.Metadata().Get(chainStateKeyName)
		state, err := deserializeBestChainState(serializedData)
		if err != nil {
			return err
		}
		tip := &state.hash

//扫描旧的块索引桶并构造每个块的映射
//到父块和所有子块。
		blocksMap, err := readBlockTree(v1BlockIdxBucket)
		if err != nil {
			return err
		}

//使用方块图计算每个方块的高度。
		err = determineBlockHeights(blocksMap)
		if err != nil {
			return err
		}

//使用方块图和当前提示查找主链上的块。
		determineMainChainBlocks(blocksMap, tip)

//既然我们有了所有块的高度，就可以扫描旧的块索引了。
//将所有行放入新的行中。
		return v1BlockIdxBucket.ForEach(func(hashBytes, blockRow []byte) error {
			endOffset := blockHdrOffset + blockHdrSize
			headerBytes := blockRow[blockHdrOffset:endOffset:endOffset]

			var hash chainhash.Hash
			copy(hash[:], hashBytes[0:chainhash.HashSize])
			chainContext := blocksMap[hash]

			if chainContext.height == -1 {
				return fmt.Errorf("Unable to calculate chain height for "+
					"stored block %s", hash)
			}

//如果块是主链的一部分，则将其标记为有效。
			status := statusDataStored
			if chainContext.mainChain {
				status |= statusValid
			}

//将标题写入v2 bucket
			value := make([]byte, blockHdrSize+1)
			copy(value[0:blockHdrSize], headerBytes)
			value[blockHdrSize] = byte(status)

			key := blockIndexKey(&hash, uint32(chainContext.height))
			err := v2BlockIdxBucket.Put(key, value)
			if err != nil {
				return err
			}

//
			truncatedRow := blockRow[0:blockHdrOffset:blockHdrOffset]
			return v1BlockIdxBucket.Put(hashBytes, truncatedRow)
		})
	})
	if err != nil {
		return err
	}

	log.Infof("Block database migration complete")
	return nil
}

//readBlockTree读取旧的块索引桶并构造
//每个块到其父块和所有子块。此映射表示
//满树的木块。此函数不填充高度或
//返回的BlockChainContext值的主链字段。
func readBlockTree(v1BlockIdxBucket database.Bucket) (map[chainhash.Hash]*blockChainContext, error) {
	blocksMap := make(map[chainhash.Hash]*blockChainContext)
	err := v1BlockIdxBucket.ForEach(func(_, blockRow []byte) error {
		var header wire.BlockHeader
		endOffset := blockHdrOffset + blockHdrSize
		headerBytes := blockRow[blockHdrOffset:endOffset:endOffset]
		err := header.Deserialize(bytes.NewReader(headerBytes))
		if err != nil {
			return err
		}

		blockHash := header.BlockHash()
		prevHash := header.PrevBlock

		if blocksMap[blockHash] == nil {
			blocksMap[blockHash] = &blockChainContext{height: -1}
		}
		if blocksMap[prevHash] == nil {
			blocksMap[prevHash] = &blockChainContext{height: -1}
		}

		blocksMap[blockHash].parent = &prevHash
		blocksMap[prevHash].children =
			append(blocksMap[prevHash].children, &blockHash)
		return nil
	})
	return blocksMap, err
}

//
//并使用它来计算每个块的高度。函数指定
//0到Genesis散列的高度并探索块树
//宽度优先，为每个块分配一个高度，并返回到
//创世纪大厦此函数修改块映射上的高度字段
//条目。
func determineBlockHeights(blocksMap map[chainhash.Hash]*blockChainContext) error {
	queue := list.New()

//Genesis块作为零哈希的子块包含在blocksmap中。
//因为这是Genesis头段中PrevBlock字段的值。
	preGenesisContext, exists := blocksMap[zeroHash]
	if !exists || len(preGenesisContext.children) == 0 {
		return fmt.Errorf("Unable to find genesis block")
	}

	for _, genesisHash := range preGenesisContext.children {
		blocksMap[*genesisHash].height = 0
		queue.PushBack(genesisHash)
	}

	for e := queue.Front(); e != nil; e = queue.Front() {
		queue.Remove(e)
		hash := e.Value.(*chainhash.Hash)
		height := blocksMap[*hash].height

//
//
		for _, childHash := range blocksMap[*hash].children {
			blocksMap[*childHash].height = height + 1
			queue.PushBack(childHash)
		}
	}

	return nil
}

//DetermineMainChainBlocks将块图从顶部向下遍历到
//确定哪些块散列是主链的一部分。这个函数
//修改blocksmap项上的mainchain字段。
func determineMainChainBlocks(blocksMap map[chainhash.Hash]*blockChainContext, tip *chainhash.Hash) {
	for nextHash := tip; *nextHash != zeroHash; nextHash = blocksMap[*nextHash].parent {
		blocksMap[*nextHash].mainChain = true
	}
}

//反序列化eutxEntryv0从传递的序列化字节中解码utxo项
//根据旧版本0的格式切片到一个由键控的utxos映射中
//事务中的输出索引。地图是必要的，因为
//以前的格式使用单个
//条目，而新格式分别对每个未暂停的输出进行编码。
//
//
//
//<version><height><header code><unspentness bitmap>[<compressed txouts>，…]
//
//字段类型大小
//版本VLQ变量
//
//头代码VLQ变量
//
//压缩txout
//压缩量VLQ变量
//压缩脚本[]字节变量
//
//序列化的头代码格式为：
//位0-包含事务是一个coinbase
//
//位2-输出1未使用
//bits 3-x—未使用位图中的字节数。当位1和2
//未设置，它编码n-1，因为必须至少有一个未使用的
//输出。
//
//标题代码方案的基本原理如下：
//-只支付单个输出和更改输出的事务是
//非常常见，因此，未使用位图的额外字节可以
//通过将这两个输出编码为低阶位来避免这种情况。
//-假设它被编码为一个VLQ，它可以用一个
//单字节，它留下4位来表示
//未使用的位图，但仍只为
//
//
//这涵盖了绝大多数交易。
//-当位1和2都未设置时，编码n-1字节允许额外的
//在导致头代码需要
//附加字节。
//
//例1：
//来自主区块链中的Tx：
//黑色1，0e3e2357e806b6cdb1f70b54c3a17b6714ee1f0e68beb44a74b1efd512098
//
//010103320496B538E853519C726A2C91E61EC1600AE1390813A627C66FB8BE7947BE63C52
//<><><><-------------------------------------------------------------------
//94
//高度压缩txout 0
//版本标题代码
//
//-版本：1
//-高度：1
//-头代码：0x03（coinbase，输出0未暂停，0字节未暂停）
//-不可用：没有，因为它是零字节
//-压缩txout 0：
//-0x32:5000000000（50 BTC）的VLQ编码压缩量
//-0x04:特殊脚本类型pay to pubkey
//-0x96…52:pubkey的x坐标
//
//例2：
//来自主区块链中的Tx：
//黑色113931，4A16969AA4764DD7507FC1DE7F0BAA4850A246DE90C45E59A3207F9A26B5036F
//
//
//<><----<><><------------------------------------------------->
//|    |  | \-------------------\            |                            |
//版本\------\unspentness压缩txout 2
//高度标题代码压缩txout 0
//
//-版本：1
//-高度：113931
//-头代码：0x0A（输出0个未暂停，1个字节在未暂停位图中）
//-未暂停：[0x01]（设置了位0，因此输出0+2=2未暂停）
//注意：它是+2，因为前两个输出是用头代码编码的
//-压缩txout 0：
//-0x12:20000000（0.2 BTC）的VLQ编码压缩量
//-0x00:特殊脚本类型pay to pubkey哈希
//-0xe2…8a:公钥哈希
//-压缩txout 2：
//-0x8009:15000000（0.15 BTC）的VLQ编码压缩量
//-0x00:特殊脚本类型pay to pubkey哈希
//-0xB8…58:公钥哈希
//
//例3：
//来自主区块链中的Tx：
//BLK 338156、1B02D1C8CFF60A189017B9A420C682CF4A0028175F2F563209E4F61C8C3620
//
//0193D06C100000108BA5B9E763011DD46A006572D820E448E12D2BB38640BC718E6
//<><----<><----<---------------------------------------------->
//|    |  |   \-----------------\            |
//版本\------\unspentness
//高度割台代码压缩txout 22
//
//-版本：1
//-高度：338156
//-头代码：0x10（未使用位图中的2+1=3字节）
//注意：它是+1，因为位1和2都没有设置，所以n-1被编码。
//
//注意：它是+2，因为前两个输出是用头代码编码的
//-压缩txout 22：
//-0x8BA5B9E763:366875659的VLQ编码压缩量（3.66875659 BTC）
//-0x01:特殊脚本类型付费脚本哈希
//-0x1D…E6:脚本哈希
func deserializeUtxoEntryV0(serialized []byte) (map[uint32]*UtxoEntry, error) {
//反序列化版本。
//
//
	_, bytesRead := deserializeVLQ(serialized)
	offset := bytesRead
	if offset >= len(serialized) {
		return nil, errDeserialize("unexpected end of data after version")
	}

//反序列化块高度。
	blockHeight, bytesRead := deserializeVLQ(serialized[offset:])
	offset += bytesRead
	if offset >= len(serialized) {
		return nil, errDeserialize("unexpected end of data after height")
	}

//反序列化头代码。
	code, bytesRead := deserializeVLQ(serialized[offset:])
	offset += bytesRead
	if offset >= len(serialized) {
		return nil, errDeserialize("unexpected end of data after header")
	}

//解码头代码。
//
//位0表示包含的事务是否为coinbase。
//位1表示输出0已松开。
//位2表示输出1已松开。
//bits 3-x编码非零未使用位图字节数
//跟随。当输出0和1都用完时，它编码n-1。
	isCoinBase := code&0x01 != 0
	output0Unspent := code&0x02 != 0
	output1Unspent := code&0x04 != 0
	numBitmapBytes := code >> 3
	if !output0Unspent && !output1Unspent {
		numBitmapBytes++
	}

//确保有足够的字节来反序列化未使用的
//位图。
	if uint64(len(serialized[offset:])) < numBitmapBytes {
		return nil, errDeserialize("unexpected end of data for " +
			"unspentness bitmap")
	}

//根据需要为未暂停的输出0和1添加稀疏输出
//标题代码提供的详细信息。
	var outputIndexes []uint32
	if output0Unspent {
		outputIndexes = append(outputIndexes, 0)
	}
	if output1Unspent {
		outputIndexes = append(outputIndexes, 1)
	}

//解码未使用的位图，为每个未使用的位图添加稀疏输出
//输出。
	for i := uint32(0); i < uint32(numBitmapBytes); i++ {
		unspentBits := serialized[offset]
		for j := uint32(0); j < 8; j++ {
			if unspentBits&0x01 != 0 {
//前2个输出通过
//头代码，因此调整输出编号
//因此。
				outputNum := 2 + i*8 + j
				outputIndexes = append(outputIndexes, outputNum)
			}
			unspentBits >>= 1
		}
		offset++
	}

//
	entries := make(map[uint32]*UtxoEntry)

//所有条目都可能需要标记为CoinBase。
	var packedFlags txoFlags
	if isCoinBase {
		packedFlags |= tfCoinBase
	}

//
	for i, outputIndex := range outputIndexes {
//解码下一个utxo。
		amount, pkScript, bytesRead, err := decodeCompressedTxOut(
			serialized[offset:])
		if err != nil {
			return nil, errDeserialize(fmt.Sprintf("unable to "+
				"decode utxo at index %d: %v", i, err))
		}
		offset += bytesRead

//创建一个新的utxo条目，上面反序列化了详细信息。
		entries[outputIndex] = &UtxoEntry{
			amount:      int64(amount),
			pkScript:    pkScript,
			blockHeight: int32(blockHeight),
			packedFlags: packedFlags,
		}
	}

	return entries, nil
}

//upgradeutxosetov2将utxo集条目从版本1迁移到版本2中
//批次。如果返回时没有失败，则保证更新。
func upgradeUtxoSetToV2(db database.DB, interrupt <-chan struct{}) error {
//硬编码的存储桶名称，因此对全局值的更新不会影响
//旧升级。
	var (
		v1BucketName = []byte("utxoset")
		v2BucketName = []byte("utxosetv2")
	)

	log.Infof("Upgrading utxo set to v2.  This will take a while...")
	start := time.Now()

//根据需要创建新的utxo set bucket。
	err := db.Update(func(dbTx database.Tx) error {
		_, err := dbTx.Metadata().CreateBucketIfNotExists(v2BucketName)
		return err
	})
	if err != nil {
		return err
	}

//
//版本1到2分批。这样做是因为utxo集可以
//巨大，因此试图在单个数据库事务中迁移
//会导致大量内存使用，并可能崩溃
//许多系统都是由于ulimits。
//
//它返回处理的utxos数。
	const maxUtxos = 200000
	doBatch := func(dbTx database.Tx) (uint32, error) {
		v1Bucket := dbTx.Metadata().Bucket(v1BucketName)
		v2Bucket := dbTx.Metadata().Bucket(v2BucketName)
		v1Cursor := v1Bucket.Cursor()

//迁移utxos，只要它的最大utxos数
//未超出批处理。
		var numUtxos uint32
		for ok := v1Cursor.First(); ok && numUtxos < maxUtxos; ok =
			v1Cursor.Next() {

//旧密钥是事务哈希。
			oldKey := v1Cursor.Key()
			var txHash chainhash.Hash
			copy(txHash[:], oldKey)

//反序列化包含所有utxo的旧条目
//对于给定的事务。
			utxos, err := deserializeUtxoEntryV0(v1Cursor.Value())
			if err != nil {
				return 0, err
			}

//使用将每个utxo的条目添加到新bucket中
//新格式。
			for txOutIdx, utxo := range utxos {
				reserialized, err := serializeUtxoEntry(utxo)
				if err != nil {
					return 0, err
				}

				key := outpointKey(wire.OutPoint{
					Hash:  txHash,
					Index: txOutIdx,
				})
				err = v2Bucket.Put(*key, reserialized)
//注意：钥匙是故意不回收的
//这里是因为数据库接口契约
//禁止修改。会是垃圾
//数据库完成后正常收集
//有了它。
				if err != nil {
					return 0, err
				}
			}

//删除旧条目。
			err = v1Bucket.Delete(oldKey)
			if err != nil {
				return 0, err
			}

			numUtxos += uint32(len(utxos))

			if interruptRequested(interrupt) {
//这里没有错误，所以数据库事务
//未取消，因此未完成
//
				break
			}
		}

		return numUtxos, nil
	}

//基于上述原因批量迁移所有条目。
	var totalUtxos uint64
	for {
		var numUtxos uint32
		err := db.Update(func(dbTx database.Tx) error {
			var err error
			numUtxos, err = doBatch(dbTx)
			return err
		})
		if err != nil {
			return err
		}

		if interruptRequested(interrupt) {
			return errInterruptRequested
		}

		if numUtxos == 0 {
			break
		}

		totalUtxos += uint64(numUtxos)
		log.Infof("Migrated %d utxos (%d total)", numUtxos, totalUtxos)
	}

//删除旧的bucket，并在它具有
//已完全迁移。
	err = db.Update(func(dbTx database.Tx) error {
		err := dbTx.Metadata().DeleteBucket(v1BucketName)
		if err != nil {
			return err
		}

		return dbPutVersion(dbTx, utxoSetVersionKeyName, 2)
	})
	if err != nil {
		return err
	}

	seconds := int64(time.Since(start) / time.Second)
	log.Infof("Done upgrading utxo set.  Total utxos: %d in %d seconds",
		totalUtxos, seconds)
	return nil
}

//maybeupgradedbuckets检查此使用的buckets的数据库版本
//打包并执行任何所需的升级，以使其达到最新版本。
//
//如果
//
func (b *BlockChain) maybeUpgradeDbBuckets(interrupt <-chan struct{}) error {
//
	var utxoSetVersion uint32
	err := b.db.Update(func(dbTx database.Tx) error {
//从数据库中加载或创建utxo集版本，然后
//如果不存在，则将其初始化为版本1。
		var err error
		utxoSetVersion, err = dbFetchOrCreateVersion(dbTx,
			utxoSetVersionKeyName, 1)
		return err
	})
	if err != nil {
		return err
	}

//如果需要，将utxo集更新为v2。
	if utxoSetVersion < 2 {
		if err := upgradeUtxoSetToV2(b.db, interrupt); err != nil {
			return err
		}
	}

	return nil
}
