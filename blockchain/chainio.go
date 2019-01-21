
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015-2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package blockchain

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

const (
//blockhdrsize是块头的大小。这只是
//从电线上取常数，仅在此处提供方便，因为
//Wire.MaxBlockHeaderPayLoad相当长。
	blockHdrSize = wire.MaxBlockHeaderPayload

//latestutxosetbucketversion是utxo集的当前版本
//用于跟踪所有未暂停输出的存储桶。
	latestUtxoSetBucketVersion = 2

//LatestSpendJournalBucketVersion是支出的当前版本
//用于跟踪所有已用事务以供使用的日记帐存储桶
//在ReOrgS中。
	latestSpendJournalBucketVersion = 1
)

var (
//BlockIndexBucketname是用于存储到
//阻止标题和上下文信息。
	blockIndexBucketName = []byte("blockheaderidx")

//HashIndexBucketname是用于存储到
//块哈希->块高度索引。
	hashIndexBucketName = []byte("hashidx")

//HeightIndexBucketname是用于存储
//块高度->块哈希索引。
	heightIndexBucketName = []byte("heightidx")

//chainStateKeyName是用于存储最佳
//链状态。
	chainStateKeyName = []byte("chainstate")

//SpendJournalVersionKeyName是用于存储的DB密钥的名称
//数据库中当前支出日记帐的版本。
	spendJournalVersionKeyName = []byte("spendjournalversion")

//SpendjournalBucketname是用于存储的数据库桶的名称
//在每个块中花费的事务输出。
	spendJournalBucketName = []byte("spendjournal")

//utxosetVersionKeyName是用于存储
//
	utxoSetVersionKeyName = []byte("utxosetversion")

//utxosetbacketname是用于存储
//未占用的事务输出集。
	utxoSetBucketName = []byte("utxosetv2")

//字节顺序是用于序列化数字的首选字节顺序
//用于存储在数据库中的字段。
	byteOrder = binary.LittleEndian
)

//errnotinMinain表示块散列或高度不在
//请求主链。
type errNotInMainChain string

//错误实现错误接口。
func (e errNotInMainChain) Error() string {
	return string(e)
}

//IsNotInMainErr返回传递的错误是否为
//errnotinMain错误。
func isNotInMainChainErr(err error) bool {
	_, ok := err.(errNotInMainChain)
	return ok
}

//errDeserialize表示反序列化时遇到问题
//数据。
type errDeserialize string

//错误实现错误接口。
func (e errDeserialize) Error() string {
	return string(e)
}

//IsDeserializer返回传递的错误是否为errDeserialize
//错误。
func isDeserializeErr(err error) bool {
	_, ok := err.(errDeserialize)
	return ok
}

//isdbbucketNotFounderr返回传递的错误是否为
//database.error，错误代码为database.errbacketnotfound。
func isDbBucketNotFoundErr(err error) bool {
	dbErr, ok := err.(database.Error)
	return ok && dbErr.ErrorCode == database.ErrBucketNotFound
}

//dbfetchversion从
//元数据存储桶。它主要用于跟踪实体上的版本，例如
//桶。如果提供的键不存在，则返回零。
func dbFetchVersion(dbTx database.Tx, key []byte) uint32 {
	serialized := dbTx.Metadata().Get(key)
	if serialized == nil {
		return 0
	}

	return byteOrder.Uint32(serialized[:])
}

//DBPutVersion使用现有的数据库事务更新提供的
//在元数据存储桶中键入给定版本。它主要用于
//跟踪实体（如bucket）上的版本。
func dbPutVersion(dbTx database.Tx, key []byte, version uint32) error {
	var serialized [4]byte
	byteOrder.PutUint32(serialized[:], version)
	return dbTx.Metadata().Put(key, serialized[:])
}

//dbfetchorCreateVersion使用现有的数据库事务尝试
//从元数据存储桶中获取所提供的密钥作为版本，在这种情况下
//它不存在，它使用提供的默认版本添加条目，并且
//返回。这在升级过程中很有用，可以自动处理加载
//并根据需要添加版本密钥。
func dbFetchOrCreateVersion(dbTx database.Tx, key []byte, defaultVersion uint32) (uint32, error) {
	version := dbFetchVersion(dbTx, key)
	if version == 0 {
		version = defaultVersion
		err := dbPutVersion(dbTx, key, version)
		if err != nil {
			return 0, err
		}
	}

	return version, nil
}

//————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//事务支出日记帐由连接的每个块的条目组成
//到包含事务输出的主链块开销
//序列化，使其顺序与使用顺序相反。
//
//这是必需的，因为重组链必然需要
//断开块以回到分叉点，这意味着
//取消搁置每个块以前花费的所有事务输出。
//因为根据定义，utxo集只包含未暂停的事务输出，
//必须从某个地方恢复已用事务输出。有
//有多种方法可以做到这一点，但这是最直接的
//不需要具有事务索引且未运行的Forward方法
//块链。
//
//注意：此格式不是自我描述的。其他细节，如
//条目数（事务输入）预期来自
//块本身和utxo集（用于遗留项）。做的理由
//这是为了节省空间。这也是用过的输出
//以相反的顺序序列化，因为后面的事务
//允许在同一块中使用早期输出。
//
//下面的保留字段用于跟踪包含的版本
//当头代码中的高度为非零时的事务，但是
//高度现在总是非零，但保留额外的保留字段允许
//向后兼容。
//
//序列化格式为：
//
//[<header code><reserved><compressed txout>]，…
//
//字段类型大小
//头代码VLQ变量
//保留字节1
//压缩txout
//压缩量VLQ变量
//压缩脚本[]字节变量
//
//序列化的头代码格式为：
//位0-包含事务是一个coinbase
//bits 1-x-包含已用txout的块的高度
//
//例1：
//来自主区块链中的区块170。
//
//130032051DB93E1DCDB8A016B49840F8C53BC1EB68A382E97B1482CAD7B1148A6909A5C
//
//| |                                  |
//预留压缩txout
//标题代码
//
//-标题代码：0x13（CoinBase，高度9）
//-保留：0x00
//-压缩txout 0：
//-0x32:5000000000（50 BTC）的VLQ编码压缩量
//-0x05:特殊脚本类型pay to pubkey
//-0x11…5C:pubkey的x坐标
//
//例2：
//改编自主区块链中的区块100025。
//
//8b99700091f20f006edbc6c4d31bae9f1ccc38538a114bf42de65e868b99700086c64700b2fb57eadf61e06a10a7445a8c3f67898841ec
//<-----><><---------------------------------------------------------------<-><----------------------------------->
//__
//预留压缩txout预留压缩txout
//标题代码标题代码
//
//-上次消耗的输出：
//-标题代码：0x8B9970（非CoinBase，高度100024）
//-保留：0x00
//-压缩txout：
//-0x91F20F:34405000000（344.05 BTC）的VLQ编码压缩量
//-0x00:特殊脚本类型pay to pubkey哈希
//-0x6e…86:公钥哈希
//-第二个至最后一个已用输出：
//-标题代码：0x8B9970（非CoinBase，高度100024）
//-保留：0x00
//-压缩txout：
//-0x86C647:13761000000（137.61 BTC）的VLQ编码压缩量
//-0x00:特殊脚本类型pay to pubkey哈希
//-0xB2…EC:公钥哈希
//————————————————————————————————————————————————————————————————————————————————————————————————————————————————

//spentxout包含一个已用事务输出，可能还有其他
//上下文信息，例如它是否包含在coinbase中
//事务，它所包含的事务的版本，以及
//包含事务的块高度。如上所述
//上面的注释，附加的上下文信息将仅有效
//当这个花费的txout花费包含的最后未使用的输出时
//交易。
type SpentTxOut struct {
//Amount是输出量。
	Amount int64

//pkscipt是输出的公钥脚本。
	PkScript []byte

//height是包含创建tx的块的高度。
	Height int32

//指示创建的Tx是否为CoinBase。
	IsCoinBase bool
}

//fetchPendJournal尝试检索支出日记帐或
//为目标块花费的输出。这提供了所有输出的视图
//一旦目标块连接到
//主链。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) FetchSpendJournal(targetBlock *btcutil.Block) ([]SpentTxOut, error) {
	b.chainLock.RLock()
	defer b.chainLock.RUnlock()

	var spendEntries []SpentTxOut
	err := b.db.View(func(dbTx database.Tx) error {
		var err error

		spendEntries, err = dbFetchSpendJournalEntry(dbTx, targetBlock)
		return err
	})
	if err != nil {
		return nil, err
	}

	return spendEntries, nil
}

//spentxoutheadercode返回计算的头代码，当
//正在序列化提供的stxo项。
func spentTxOutHeaderCode(stxo *SpentTxOut) uint64 {
//如序列化格式注释中所述，头代码
//对移动超过一位的高度和
//最低位。
	headerCode := uint64(stxo.Height) << 1
	if stxo.IsCoinBase {
		headerCode |= 0x01
	}

	return headerCode
}

//spentxoutserializesize返回需要的字节数
//根据上述格式序列化传递的stxo。
func spentTxOutSerializeSize(stxo *SpentTxOut) int {
	size := serializeSizeVLQ(spentTxOutHeaderCode(stxo))
	if stxo.Height > 0 {
//Legacy v1 Spend Journal格式有条件地跟踪了
//包含高度非零时的事务版本，
//所以这是向后兼容所必需的。
		size += serializeSizeVLQ(0)
	}
	return size + compressedTxOutSize(uint64(stxo.Amount), stxo.PkScript)
}

//putspenttxout根据所描述的格式序列化传递的stxo
//直接进入传递的目标字节片。目标字节片必须
//至少大到足以处理
//spentxoutserialize函数，否则将死机。
func putSpentTxOut(target []byte, stxo *SpentTxOut) int {
	headerCode := spentTxOutHeaderCode(stxo)
	offset := putVLQ(target, headerCode)
	if stxo.Height > 0 {
//Legacy v1 Spend Journal格式有条件地跟踪了
//包含高度非零时的事务版本，
//所以这是向后兼容所必需的。
		offset += putVLQ(target[offset:], 0)
	}
	return offset + putCompressedTxOut(target[offset:], uint64(stxo.Amount),
		stxo.PkScript)
}

//decodespentxout解码传递的序列化stxo项，可能后面跟着
//通过其他数据，进入传递的stxo结构。它返回字节数
//读。
func decodeSpentTxOut(serialized []byte, stxo *SpentTxOut) (int, error) {
//确保有要解码的字节。
	if len(serialized) == 0 {
		return 0, errDeserialize("no serialized bytes")
	}

//反序列化头代码。
	code, offset := deserializeVLQ(serialized)
	if offset >= len(serialized) {
		return offset, errDeserialize("unexpected end of data after " +
			"header code")
	}

//解码头代码。
//
//位0表示包含事务是一个coinbase。
//位1-X编码包含事务的高度。
	stxo.IsCoinBase = code&0x01 != 0
	stxo.Height = int32(code >> 1)
	if stxo.Height > 0 {
//Legacy v1 Spend Journal格式有条件地跟踪了
//包含高度非零时的事务版本，
//所以这是向后兼容所必需的。
		_, bytesRead := deserializeVLQ(serialized[offset:])
		offset += bytesRead
		if offset >= len(serialized) {
			return offset, errDeserialize("unexpected end of data " +
				"after reserved")
		}
	}

//解码压缩的txout。
	amount, pkScript, bytesRead, err := decodeCompressedTxOut(
		serialized[offset:])
	offset += bytesRead
	if err != nil {
		return offset, errDeserialize(fmt.Sprintf("unable to decode "+
			"txout: %v", err))
	}
	stxo.Amount = int64(amount)
	stxo.PkScript = pkScript
	return offset, nil
}

//DeserializespendJournalEntry将传递的序列化字节片解码为
//根据上面详细描述的格式对已用txout进行切片。
//
//因为序列化格式不是自描述的，如
//设置注释格式，此函数还要求使用
//TXOUT。
func deserializeSpendJournalEntry(serialized []byte, txns []*wire.MsgTx) ([]SpentTxOut, error) {
//计算stxos的总数。
	var numStxos int
	for _, tx := range txns {
		numStxos += len(tx.TxIn)
	}

//当一个块没有花费txout时，就没有要序列化的内容。
	if len(serialized) == 0 {
//确保块实际上没有stxos。这不应该
//除非数据库损坏或条目为空，否则将发生
//错误地进入数据库。
		if numStxos != 0 {
			return nil, AssertError(fmt.Sprintf("mismatched spend "+
				"journal serialization - no serialization for "+
				"expected %d stxos", numStxos))
		}

		return nil, nil
	}

//在所有事务中向后循环，以便读取所有内容
//颠倒顺序以匹配序列化顺序。
	stxoIdx := numStxos - 1
	offset := 0
	stxos := make([]SpentTxOut, numStxos)
	for txIdx := len(txns) - 1; txIdx > -1; txIdx-- {
		tx := txns[txIdx]

//在所有事务输入中向后循环并读取
//关联的stxo。
		for txInIdx := len(tx.TxIn) - 1; txInIdx > -1; txInIdx-- {
			txIn := tx.TxIn[txInIdx]
			stxo := &stxos[stxoIdx]
			stxoIdx--

			n, err := decodeSpentTxOut(serialized[offset:], stxo)
			offset += n
			if err != nil {
				return nil, errDeserialize(fmt.Sprintf("unable "+
					"to decode stxo for %v: %v",
					txIn.PreviousOutPoint, err))
			}
		}
	}

	return stxos, nil
}

//SeriesSpendJournalEntry将所有传递的已用txout序列化到
//按照上面详细描述的格式进行单字节切片。
func serializeSpendJournalEntry(stxos []SpentTxOut) []byte {
	if len(stxos) == 0 {
		return nil
	}

//计算序列化整个日记条目所需的大小。
	var size int
	for i := range stxos {
		size += spentTxOutSerializeSize(&stxos[i])
	}
	serialized := make([]byte, size)

//反向将每个单独的stxo直接序列化到切片中
//一个接一个点。
	var offset int
	for i := len(stxos) - 1; i > -1; i-- {
		offset += putSpentTxOut(serialized[offset:], &stxos[i])
	}

	return serialized
}

//dbfetchPendJournalEntry获取已传递块的支出日记条目
//并将其反序列化为已用txout条目的切片。
//
//注意：传统条目将不会设置coinbase标志或高度，除非它
//是包含事务中的最终输出支出。这取决于
//调用方通过在utxo集中查找信息来正确处理此问题。
func dbFetchSpendJournalEntry(dbTx database.Tx, block *btcutil.Block) ([]SpentTxOut, error) {
//排除coinbase事务，因为它不能花费任何东西。
	spendBucket := dbTx.Metadata().Bucket(spendJournalBucketName)
	serialized := spendBucket.Get(block.Hash()[:])
	blockTxns := block.MsgBlock().Transactions[1:]
	stxos, err := deserializeSpendJournalEntry(serialized, blockTxns)
	if err != nil {
//确保将任何反序列化错误作为数据库返回
//损坏错误。
		if isDeserializeErr(err) {
			return nil, database.Error{
				ErrorCode: database.ErrCorruption,
				Description: fmt.Sprintf("corrupt spend "+
					"information for %v: %v", block.Hash(),
					err),
			}
		}

		return nil, err
	}

	return stxos, nil
}

//dbputspendjournalEntry使用现有的数据库事务更新
//使用提供的切片为给定块哈希花费日记条目
//已用完的txout。已用txout切片必须包含每个txout的条目
//块中的事务按花费顺序进行花费。
func dbPutSpendJournalEntry(dbTx database.Tx, blockHash *chainhash.Hash, stxos []SpentTxOut) error {
	spendBucket := dbTx.Metadata().Bucket(spendJournalBucketName)
	serialized := serializeSpendJournalEntry(stxos)
	return spendBucket.Put(blockHash[:], serialized)
}

//DBRemoveSpendJournalEntry使用现有数据库事务删除
//已传递的块哈希的支出日记条目。
func dbRemoveSpendJournalEntry(dbTx database.Tx, blockHash *chainhash.Hash) error {
	spendBucket := dbTx.Metadata().Bucket(spendJournalBucketName)
	return spendBucket.Delete(blockHash[:])
}

//————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//未暂停的事务输出（utxo）集包含每个
//使用优化格式的未暂停输出，以减少使用域的空间
//特定的压缩算法。此格式是稍微修改过的版本
//比特币核心使用的格式。
//
//
//注意，密钥编码使用VLQ，它使用MSB编码，因此
//进行字节比较时，utxos的迭代将在
//秩序。
//
//序列化密钥格式为：
//<hash><output index>
//
//字段类型大小
//哈希chainhash.hash chainhash.hashsize
//输出指数VLQ变量
//
//序列化值格式为：
//
//<header code><compressed txout>
//
//字段类型大小
//头代码VLQ变量
//压缩txout
//压缩量VLQ变量
//压缩脚本[]字节变量
//
//序列化的头代码格式为：
//位0-包含事务是一个coinbase
//bits 1-x-包含未使用txout的块的高度
//
//例1：
//来自主区块链中的Tx：
//黑色1，0e3e2357e806b6cdb1f70b54c3a17b6714ee1f0e68beb44a74b1efd512098:0
//
//03320496B538E853519C726A2C91E61EC1600AE1390813A627C66FB8BE7947BE63C52
//<>
//_
//头代码压缩txout
//
//-标题代码：0x03（CoinBase，高度1）
//-压缩txout：
//-0x32:5000000000（50 BTC）的VLQ编码压缩量
//-0x04:特殊脚本类型pay to pubkey
//-0x96…52:pubkey的x坐标
//
//例2：
//来自主区块链中的Tx：
//黑色113931，4A16969AA4764DD7507FC1DE7F0BAA4850A246DE90C45E59A3207F9A26B5036F:2
//
//8CF3168000900B8025BE1B3EFC63B0AD48E7F9F10E87544528D58
//<-----><-------------------------------------------------->
//_
//头代码压缩txout
//
//-标题代码：0x8CF316（非CoinBase，高度113931）
//-压缩txout：
//-0x8009:15000000（0.15 BTC）的VLQ编码压缩量
//-0x00:特殊脚本类型pay to pubkey哈希
//-0xB8…58:公钥哈希
//
//例3：
//来自主区块链中的Tx：
//BLK 338156、1B02D1C8CFF60A189017B9A420C682CF4A0028175F2F563209E4F61C8C3620:22
//
//A8A258BA5B9E763011DD46A006572D820E448E12D2BB38640BC718E6
//<-----><------------------------------------------------------------>
//_
//头代码压缩txout
//
//-标题代码：0xA8A258（非CoinBase，高度338156）
//-压缩txout：
//-0x8BA5B9E763:366875659的VLQ编码压缩量（3.66875659 BTC）
//-0x01:特殊脚本类型付费脚本哈希
//-0x1D…E6:脚本哈希
//————————————————————————————————————————————————————————————————————————————————————————————————————————————————

//maxuint32vlqserializesize是最大uint32占用的字节数。
//作为VLQ序列化。
var maxUint32VLQSerializeSize = serializeSizeVLQ(1<<32 - 1)

//OutpointKeyPool定义用于
//为输出点数据库键提供临时缓冲区。
var outpointKeyPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, chainhash.HashSize+maxUint32VLQSerializeSize)
return &b //指向slice的指针，以避免装箱alloc。
	},
}

//outpointkey返回一个适合在utxo集中用作数据库键的键
//同时使用免费列表。如果没有新的缓冲区，则分配新的缓冲区
//已经在免费列表上有了。返回的字节片应为
//当
//调用方已经完成了它，除非切片需要比
//调用方可以计算，例如何时用于写入数据库。
func outpointKey(outpoint wire.OutPoint) *[]byte {
//VLQ采用了MSB编码，因此它们不仅有助于减少
//存储空间的数量，也是这样迭代的utxos时
//按字节进行比较将按顺序生成它们。
	key := outpointKeyPool.Get().(*[]byte)
	idx := uint64(outpoint.Index)
	*key = (*key)[:chainhash.HashSize+serializeSizeVLQ(idx)]
	copy(*key, outpoint.Hash[:])
	putVLQ((*key)[chainhash.HashSize:], idx)
	return key
}

//RecycleOutpointKey放入提供的字节片，该片应该
//通过outpointkey函数获取，返回自由列表。
func recycleOutpointKey(key *[]byte) {
	outpointKeyPool.Put(key)
}

//utxEntryHeaderCode返回计算的头代码，当
//正在序列化提供的utxo项。
func utxoEntryHeaderCode(entry *UtxoEntry) (uint64, error) {
	if entry.IsSpent() {
		return 0, AssertError("attempt to serialize spent utxo header")
	}

//如序列化格式注释中所述，头代码
//对移动超过一位的高度和
//最低位。
	headerCode := uint64(entry.BlockHeight()) << 1
	if entry.IsCoinBase() {
		headerCode |= 0x01
	}

	return headerCode, nil
}

//SerializeUtxEntry返回序列化为适当格式的项
//用于长期储存。格式在上面详细描述。
func serializeUtxoEntry(entry *UtxoEntry) ([]byte, error) {
//已用输出没有序列化。
	if entry.IsSpent() {
		return nil, nil
	}

//对头代码进行编码。
	headerCode, err := utxoEntryHeaderCode(entry)
	if err != nil {
		return nil, err
	}

//计算序列化条目所需的大小。
	size := serializeSizeVLQ(headerCode) +
		compressedTxOutSize(uint64(entry.Amount()), entry.PkScript())

//序列化头代码，然后是压缩的未使用的
//事务输出。
	serialized := make([]byte, size)
	offset := putVLQ(serialized, headerCode)
	offset += putCompressedTxOut(serialized[offset:], uint64(entry.Amount()),
		entry.PkScript())

	return serialized, nil
}

//反序列化eutxoEntry从传递的序列化字节对utxo项进行解码
//使用适合长期使用的格式切片为新的utxoEntry
//存储。格式在上面详细描述。
func deserializeUtxoEntry(serialized []byte) (*UtxoEntry, error) {
//反序列化头代码。
	code, offset := deserializeVLQ(serialized)
	if offset >= len(serialized) {
		return nil, errDeserialize("unexpected end of data after header")
	}

//解码头代码。
//
//位0表示包含的事务是否为coinbase。
//位1-X编码包含事务的高度。
	isCoinBase := code&0x01 != 0
	blockHeight := int32(code >> 1)

//解码压缩的未暂停事务输出。
	amount, pkScript, _, err := decodeCompressedTxOut(serialized[offset:])
	if err != nil {
		return nil, errDeserialize(fmt.Sprintf("unable to decode "+
			"utxo: %v", err))
	}

	entry := &UtxoEntry{
		amount:      int64(amount),
		pkScript:    pkScript,
		blockHeight: blockHeight,
		packedFlags: 0,
	}
	if isCoinBase {
		entry.packedFlags |= tfCoinBase
	}

	return entry, nil
}

//dbfetchutxoontryhash尝试查找并获取给定哈希的utxo。
//它使用一个光标并试图尽可能高效地执行此操作。
//
//如果提供的哈希没有条目，则将为
//条目和错误。
func dbFetchUtxoEntryByHash(dbTx database.Tx, hash *chainhash.Hash) (*UtxoEntry, error) {
//尝试通过查找哈希和零来查找条目
//索引。由于密钥被序列化为<hash><index>，
//如果索引中有任何项
//哈希值，将找到一个。
	cursor := dbTx.Metadata().Bucket(utxoSetBucketName).Cursor()
	key := outpointKey(wire.OutPoint{Hash: *hash, Index: 0})
	ok := cursor.Seek(*key)
	recycleOutpointKey(key)
	if !ok {
		return nil, nil
	}

//找到了一个条目，但它可能只是下一个条目
//请求的哈希之后的最大哈希，因此请确保哈希
//实际上匹配。
	cursorKey := cursor.Key()
	if len(cursorKey) < chainhash.HashSize {
		return nil, nil
	}
	if !bytes.Equal(hash[:], cursorKey[:chainhash.HashSize]) {
		return nil, nil
	}

	return deserializeUtxoEntry(cursor.Value())
}

//dbfetchutxoEntry使用现有的数据库事务来获取指定的
//来自utxo集的事务输出。
//
//当所提供的输出没有条目时，两者都将返回nil。
//条目和错误。
func dbFetchUtxoEntry(dbTx database.Tx, outpoint wire.OutPoint) (*UtxoEntry, error) {
//获取传递的未暂停事务输出信息
//事务输出。当没有入口时立即返回。
	key := outpointKey(outpoint)
	utxoBucket := dbTx.Metadata().Bucket(utxoSetBucketName)
	serializedUtxo := utxoBucket.Get(*key)
	recycleOutpointKey(key)
	if serializedUtxo == nil {
		return nil, nil
	}

//非零零长度项表示数据库中有一个项
//对于用过的事务输出，这种情况永远不会发生。
	if len(serializedUtxo) == 0 {
		return nil, AssertError(fmt.Sprintf("database contains entry "+
			"for spent tx output %v", outpoint))
	}

//反序列化utxo条目并返回它。
	entry, err := deserializeUtxoEntry(serializedUtxo)
	if err != nil {
//确保将任何反序列化错误作为数据库返回
//损坏错误。
		if isDeserializeErr(err) {
			return nil, database.Error{
				ErrorCode: database.ErrCorruption,
				Description: fmt.Sprintf("corrupt utxo entry "+
					"for %v: %v", outpoint, err),
			}
		}

		return nil, err
	}

	return entry, nil
}

//dbputxoview使用现有的数据库事务更新utxo集
//在数据库中根据提供的utxo查看内容和状态。在
//特别是，只写已标记为修改的条目
//到数据库。
func dbPutUtxoView(dbTx database.Tx, view *UtxoViewpoint) error {
	utxoBucket := dbTx.Metadata().Bucket(utxoSetBucketName)
	for outpoint, entry := range view.entries {
//如果未修改条目，则无需更新数据库。
		if entry == nil || !entry.isModified() {
			continue
		}

//如果utxo项已用完，请将其删除。
		if entry.IsSpent() {
			key := outpointKey(outpoint)
			err := utxoBucket.Delete(*key)
			recycleOutpointKey(key)
			if err != nil {
				return err
			}

			continue
		}

//序列化并存储utxo条目。
		serialized, err := serializeUtxoEntry(entry)
		if err != nil {
			return err
		}
		key := outpointKey(outpoint)
		err = utxoBucket.Put(*key, serialized)
//注意：由于
//数据库接口协定禁止修改。它将
//当数据库处理完毕时，通常会被垃圾收集
//它。
		if err != nil {
			return err
		}
	}

	return nil
}

//
//块索引由两个bucket组成，其中每个bucket都有一个条目。
//主链。一个桶用于哈希到高度映射，另一个桶用于
//用于高度到哈希映射。
//
//哈希到高度存储桶中的值的序列化格式为：
//<高度>
//
//字段类型大小
//高度uint32 4字节
//
//height to hash bucket中值的序列化格式为：
//<散列>
//
//字段类型大小
//哈希chainhash.hash chainhash.hashsize
//————————————————————————————————————————————————————————————————————————————————————————————————————————————————

//DBPutBlockIndex使用现有的数据库事务来更新或添加
//哈希到高度和高度到哈希映射的块索引项
//提供的值。
func dbPutBlockIndex(dbTx database.Tx, hash *chainhash.Hash, height int32) error {
//序列化高度以在索引项中使用。
	var serializedHeight [4]byte
	byteOrder.PutUint32(serializedHeight[:], uint32(height))

//将块哈希到高度映射添加到索引。
	meta := dbTx.Metadata()
	hashIndex := meta.Bucket(hashIndexBucketName)
	if err := hashIndex.Put(hash[:], serializedHeight[:]); err != nil {
		return err
	}

//将块高度添加到哈希映射到索引。
	heightIndex := meta.Bucket(heightIndexBucketName)
	return heightIndex.Put(serializedHeight[:], hash[:])
}

//dbremoveblockindex使用现有的数据库事务remove block index
//从哈希到所提供的高度和高度到哈希映射的条目
//价值观。
func dbRemoveBlockIndex(dbTx database.Tx, hash *chainhash.Hash, height int32) error {
//删除块哈希到高度映射。
	meta := dbTx.Metadata()
	hashIndex := meta.Bucket(hashIndexBucketName)
	if err := hashIndex.Delete(hash[:]); err != nil {
		return err
	}

//删除块高度到哈希的映射。
	var serializedHeight [4]byte
	byteOrder.PutUint32(serializedHeight[:], uint32(height))
	heightIndex := meta.Bucket(heightIndexBucketName)
	return heightIndex.Delete(serializedHeight[:])
}

//dbfetchheightbyhash使用现有的数据库事务来检索
//从索引提供的哈希的高度。
func dbFetchHeightByHash(dbTx database.Tx, hash *chainhash.Hash) (int32, error) {
	meta := dbTx.Metadata()
	hashIndex := meta.Bucket(hashIndexBucketName)
	serializedHeight := hashIndex.Get(hash[:])
	if serializedHeight == nil {
		str := fmt.Sprintf("block %s is not in the main chain", hash)
		return 0, errNotInMainChain(str)
	}

	return int32(byteOrder.Uint32(serializedHeight)), nil
}

//dbfetchhashbyheight使用现有的数据库事务来检索
//从索引提供的高度的哈希。
func dbFetchHashByHeight(dbTx database.Tx, height int32) (*chainhash.Hash, error) {
	var serializedHeight [4]byte
	byteOrder.PutUint32(serializedHeight[:], uint32(height))

	meta := dbTx.Metadata()
	heightIndex := meta.Bucket(heightIndexBucketName)
	hashBytes := heightIndex.Get(serializedHeight[:])
	if hashBytes == nil {
		str := fmt.Sprintf("no block at height %d exists", height)
		return nil, errNotInMainChain(str)
	}

	var hash chainhash.Hash
	copy(hash[:], hashBytes)
	return &hash, nil
}

//————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//最佳链状态由最佳块哈希和高度以及
//达到并包含在最佳块中的事务数，以及
//累计工作总和达到并包括最佳区块。
//
//序列化格式为：
//
//<block hash><block height><total txns><work sum length><work sum>
//
//字段类型大小
//块哈希链哈希。哈希链哈希。哈希大小
//块高度uint32 4字节
//总txns uint64 8字节
//工作和长度uint32 4字节
//工时和大。int工时和长度
//————————————————————————————————————————————————————————————————————————————————————————————————————————————————

//BestChainState表示为当前数据库存储的数据
//最佳链状态。
type bestChainState struct {
	hash      chainhash.Hash
	height    uint32
	totalTxns uint64
	workSum   *big.Int
}

//SerializeBestChainState返回传递的块的最佳序列化
//链状态。这是要存储在链状态存储桶中的数据。
func serializeBestChainState(state bestChainState) []byte {
//计算序列化链状态所需的完整大小。
	workSumBytes := state.workSum.Bytes()
	workSumBytesLen := uint32(len(workSumBytes))
	serializedLen := chainhash.HashSize + 4 + 8 + 4 + workSumBytesLen

//序列化链状态。
	serializedData := make([]byte, serializedLen)
	copy(serializedData[0:chainhash.HashSize], state.hash[:])
	offset := uint32(chainhash.HashSize)
	byteOrder.PutUint32(serializedData[offset:], state.height)
	offset += 4
	byteOrder.PutUint64(serializedData[offset:], state.totalTxns)
	offset += 8
	byteOrder.PutUint32(serializedData[offset:], workSumBytesLen)
	offset += 4
	copy(serializedData[offset:], workSumBytes)
	return serializedData[:]
}

//DeserializeBestChainState反序列化传递的序列化最佳链
//状态。这是存储在链状态存储桶中的数据，并在
//每个区块都与主链相连或断开。
//块。
func deserializeBestChainState(serializedData []byte) (bestChainState, error) {
//确保序列化数据有足够的字节来正确反序列化
//哈希、高度、总事务数和工时和长度。
	if len(serializedData) < chainhash.HashSize+16 {
		return bestChainState{}, database.Error{
			ErrorCode:   database.ErrCorruption,
			Description: "corrupt best chain state",
		}
	}

	state := bestChainState{}
	copy(state.hash[:], serializedData[0:chainhash.HashSize])
	offset := uint32(chainhash.HashSize)
	state.height = byteOrder.Uint32(serializedData[offset : offset+4])
	offset += 4
	state.totalTxns = byteOrder.Uint64(serializedData[offset : offset+8])
	offset += 8
	workSumBytesLen := byteOrder.Uint32(serializedData[offset : offset+4])
	offset += 4

//确保序列化数据有足够的字节来反序列化工作
//和。
	if uint32(len(serializedData[offset:])) < workSumBytesLen {
		return bestChainState{}, database.Error{
			ErrorCode:   database.ErrCorruption,
			Description: "corrupt best chain state",
		}
	}
	workSumBytes := serializedData[offset : offset+workSumBytesLen]
	state.workSum = new(big.Int).SetBytes(workSumBytes)

	return state, nil
}

//dbputbeststate使用现有的数据库事务更新最佳链
//具有给定参数的状态。
func dbPutBestState(dbTx database.Tx, snapshot *BestState, workSum *big.Int) error {
//序列化当前最佳链状态。
	serializedData := serializeBestChainState(bestChainState{
		hash:      snapshot.Hash,
		height:    uint32(snapshot.Height),
		totalTxns: snapshot.TotalTxns,
		workSum:   workSum,
	})

//将当前最佳链状态存储到数据库中。
	return dbTx.Metadata().Put(chainStateKeyName, serializedData)
}

//CreateChainState将数据库和链状态初始化为
//Genesis区块。这包括创建必要的存储桶和插入
//Genesis块，因此只能在未初始化的数据库上调用它。
func (b *BlockChain) createChainState() error {
//从Genesis块创建一个新节点，并将其设置为最佳节点。
	genesisBlock := btcutil.NewBlock(b.chainParams.GenesisBlock)
	genesisBlock.SetHeight(0)
	header := &genesisBlock.MsgBlock().Header
	node := newBlockNode(header, nil)
	node.status = statusDataStored | statusValid
	b.bestChain.SetTip(node)

//将新节点添加到索引中，该索引用于更快的查找。
	b.index.addNode(node)

//初始化与最佳块相关的状态。既然是
//Genesis块，使用其时间戳作为中间时间。
	numTxns := uint64(len(genesisBlock.MsgBlock().Transactions))
	blockSize := uint64(genesisBlock.MsgBlock().SerializeSize())
	blockWeight := uint64(GetBlockWeight(genesisBlock))
	b.stateSnapshot = newBestState(node, blockSize, blockWeight, numTxns,
		numTxns, time.Unix(node.timestamp, 0))

//创建初始数据库链状态，包括创建
//必要的索引桶和插入Genesis块。
	err := b.db.Update(func(dbTx database.Tx) error {
		meta := dbTx.Metadata()

//创建存储块索引数据的存储桶。
		_, err := meta.CreateBucket(blockIndexBucketName)
		if err != nil {
			return err
		}

//创建存储链块哈希到高度的桶
//索引。
		_, err = meta.CreateBucket(hashIndexBucketName)
		if err != nil {
			return err
		}

//创建包含要散列的链块高度的桶
//索引。
		_, err = meta.CreateBucket(heightIndexBucketName)
		if err != nil {
			return err
		}

//创建存储支出日记帐数据和
//存储其版本。
		_, err = meta.CreateBucket(spendJournalBucketName)
		if err != nil {
			return err
		}
		err = dbPutVersion(dbTx, utxoSetVersionKeyName,
			latestUtxoSetBucketVersion)
		if err != nil {
			return err
		}

//创建存储utxo集的bucket并存储
//版本。请注意，Genesis块CoinBase事务是
//故意不插入此处，因为它不可用于
//共识规则。
		_, err = meta.CreateBucket(utxoSetBucketName)
		if err != nil {
			return err
		}
		err = dbPutVersion(dbTx, spendJournalVersionKeyName,
			latestSpendJournalBucketVersion)
		if err != nil {
			return err
		}

//将Genesis块保存到块索引数据库。
		err = dbStoreBlockNode(dbTx, node)
		if err != nil {
			return err
		}

//将Genesis块散列添加到高度，将高度添加到散列
//映射到索引。
		err = dbPutBlockIndex(dbTx, &node.hash, node.height)
		if err != nil {
			return err
		}

//将当前最佳链状态存储到数据库中。
		err = dbPutBestState(dbTx, b.stateSnapshot, node.workSum)
		if err != nil {
			return err
		}

//将Genesis块存储到数据库中。
		return dbStoreBlock(dbTx, genesisBlock)
	})
	return err
}

//initchainstate尝试从
//数据库。当数据库还不包含任何链状态时，它和
//链状态初始化为Genesis块。
func (b *BlockChain) initChainState() error {
//确定链数据库的状态。我们可能需要初始化
//从零开始或升级某些存储桶。
	var initialized, hasBlockIndex bool
	err := b.db.View(func(dbTx database.Tx) error {
		initialized = dbTx.Metadata().Get(chainStateKeyName) != nil
		hasBlockIndex = dbTx.Metadata().Bucket(blockIndexBucketName) != nil
		return nil
	})
	if err != nil {
		return err
	}

	if !initialized {
//此时数据库尚未初始化，因此
//将它和链状态初始化为Genesis块。
		return b.createChainState()
	}

	if !hasBlockIndex {
		err := migrateBlockIndex(b.db)
		if err != nil {
			return nil
		}
	}

//尝试从数据库加载链状态。
	err = b.db.View(func(dbTx database.Tx) error {
//从数据库元数据中获取存储的链状态。
//当它不存在时，意味着数据库还没有
//已初始化以与链一起使用，因此现在请中断以允许
//在可写的数据库事务下发生。
		serializedData := dbTx.Metadata().Get(chainStateKeyName)
		log.Tracef("Serialized chain state: %x", serializedData)
		state, err := deserializeBestChainState(serializedData)
		if err != nil {
			return err
		}

//从数据中加载所有头文件以获得已知的最佳结果
//链接并相应地构造块索引。自从
//节点数已知，执行单个分配
//对他们来说，对一大堆小的来说
//GC上的压力。
		log.Infof("Loading block index...")

		blockIndexBucket := dbTx.Metadata().Bucket(blockIndexBucketName)

//确定将有多少块加载到索引中，以便
//分配正确的金额。
		var blockCount int32
		cursor := blockIndexBucket.Cursor()
		for ok := cursor.First(); ok; ok = cursor.Next() {
			blockCount++
		}
		blockNodes := make([]blockNode, blockCount)

		var i int32
		var lastNode *blockNode
		cursor = blockIndexBucket.Cursor()
		for ok := cursor.First(); ok; ok = cursor.Next() {
			header, status, err := deserializeBlockRow(cursor.Value())
			if err != nil {
				return err
			}

//确定父块节点。因为我们迭代块头
//按高度顺序，如果块大部分是线性的，则
//很有可能上一个处理的头是父级。
			var parent *blockNode
			if lastNode == nil {
				blockHash := header.BlockHash()
				if !blockHash.IsEqual(b.chainParams.GenesisHash) {
					return AssertError(fmt.Sprintf("initChainState: Expected "+
						"first entry in block index to be genesis block, "+
						"found %s", blockHash))
				}
			} else if header.PrevBlock == lastNode.hash {
//因为我们按照高度的顺序迭代块头，如果
//块大部分是线性的，很有可能
//上一个已处理的头是父级。
				parent = lastNode
			} else {
				parent = b.index.LookupNode(&header.PrevBlock)
				if parent == nil {
					return AssertError(fmt.Sprintf("initChainState: Could "+
						"not find parent for block %s", header.BlockHash()))
				}
			}

//初始化块的块节点，连接它，
//并将其添加到块索引中。
			node := &blockNodes[i]
			initBlockNode(node, header, parent)
			node.status = status
			b.index.addNode(node)

			lastNode = node
			i++
		}

//将最佳链视图设置为存储的最佳状态。
		tip := b.index.LookupNode(&state.hash)
		if tip == nil {
			return AssertError(fmt.Sprintf("initChainState: cannot find "+
				"chain tip %s in block index", state.hash))
		}
		b.bestChain.SetTip(tip)

//加载最佳块的原始块字节。
		blockBytes, err := dbTx.FetchBlock(&state.hash)
		if err != nil {
			return err
		}
		var block wire.MsgBlock
		err = block.Deserialize(bytes.NewReader(blockBytes))
		if err != nil {
			return err
		}

//作为最后的一致性检查，我们将检查所有
//节点是当前链尖端的祖先，并标记
//如果它们尚未标记为有效。这个
//是一个安全的假设，因为当前提示之前的所有块
//根据定义是有效的。
		for iterNode := tip; iterNode != nil; iterNode = iterNode.parent {
//如果索引中尚未将此标记为有效，则
//我们现在就将其标记为有效，以确保一致性
//我们正在运行。
			if !iterNode.status.KnownValid() {
				log.Infof("Block %v (height=%v) ancestor of "+
					"chain tip not marked as valid, "+
					"upgrading to valid for consistency",
					iterNode.hash, iterNode.height)

				b.index.SetStatusFlags(iterNode, statusValid)
			}
		}

//
		blockSize := uint64(len(blockBytes))
		blockWeight := uint64(GetBlockWeight(btcutil.NewBlock(&block)))
		numTxns := uint64(len(block.Transactions))
		b.stateSnapshot = newBestState(tip, blockSize, blockWeight,
			numTxns, state.totalTxns, tip.CalcPastMedianTime())

		return nil
	})
	if err != nil {
		return err
	}

//因为我们可能在加载后更新了索引，所以我们将
//尝试将索引刷新到数据库。这只会导致
//如果元素是脏的，那么它通常是一个noop。
	return b.index.flushToDB()
}

//反序列化BlockRow将块索引桶中的值解析为块
//头和块状态位字段。
func deserializeBlockRow(blockRow []byte) (*wire.BlockHeader, blockStatus, error) {
	buffer := bytes.NewReader(blockRow)

	var header wire.BlockHeader
	err := header.Deserialize(buffer)
	if err != nil {
		return nil, statusNone, err
	}

	statusByte, err := buffer.ReadByte()
	if err != nil {
		return nil, statusNone, err
	}

	return &header, blockStatus(statusByte), nil
}

//dbfetchheaderbyhash使用现有的数据库事务来检索
//提供的哈希的块头。
func dbFetchHeaderByHash(dbTx database.Tx, hash *chainhash.Hash) (*wire.BlockHeader, error) {
	headerBytes, err := dbTx.FetchBlockHeader(hash)
	if err != nil {
		return nil, err
	}

	var header wire.BlockHeader
	err = header.Deserialize(bytes.NewReader(headerBytes))
	if err != nil {
		return nil, err
	}

	return &header, nil
}

//DBFetchHeaderByHeight使用现有的数据库事务来检索
//为提供的高度阻止页眉。
func dbFetchHeaderByHeight(dbTx database.Tx, height int32) (*wire.BlockHeader, error) {
	hash, err := dbFetchHashByHeight(dbTx, height)
	if err != nil {
		return nil, err
	}

	return dbFetchHeaderByHash(dbTx, hash)
}

//dbfetchblockbynode使用现有的数据库事务来检索
//提供的节点的原始块，对其进行反序列化，并返回bcutil.block
//设置了高度。
func dbFetchBlockByNode(dbTx database.Tx, node *blockNode) (*btcutil.Block, error) {
//从数据库加载原始块字节。
	blockBytes, err := dbTx.FetchBlock(&node.hash)
	if err != nil {
		return nil, err
	}

//创建封装块并适当设置高度。
	block, err := btcutil.NewBlockFromBytes(blockBytes)
	if err != nil {
		return nil, err
	}
	block.SetHeight(node.height)

	return block, nil
}

//dbstoreblocknode将块头和验证状态存储到块
//索引桶。这将覆盖当前条目（如果存在）。
func dbStoreBlockNode(dbTx database.Tx, node *blockNode) error {
//序列化要存储的块数据。
	w := bytes.NewBuffer(make([]byte, 0, blockHdrSize+1))
	header := node.Header()
	err := header.Serialize(w)
	if err != nil {
		return err
	}
	err = w.WriteByte(byte(node.status))
	if err != nil {
		return err
	}
	value := w.Bytes()

//将块头数据写入块索引桶。
	blockIndexBucket := dbTx.Metadata().Bucket(blockIndexBucketName)
	key := blockIndexKey(&node.hash, uint32(node.height))
	return blockIndexBucket.Put(key, value)
}

//dbstoreblock将提供的块存储在数据库中（如果尚未存储）
//那里。完整的块数据被写入ffldb。
func dbStoreBlock(dbTx database.Tx, block *btcutil.Block) error {
	hasBlock, err := dbTx.HasBlock(block.Hash())
	if err != nil {
		return err
	}
	if hasBlock {
		return nil
	}
	return dbTx.StoreBlock(block)
}

//block index key为块索引中的条目生成二进制键
//桶。键由编码为big endian的块高度组成。
//32位无符号整数，后跟32字节块哈希。
func blockIndexKey(blockHash *chainhash.Hash, blockHeight uint32) []byte {
	indexKey := make([]byte, chainhash.HashSize+4)
	binary.BigEndian.PutUint32(indexKey[0:4], blockHeight)
	copy(indexKey[4:chainhash.HashSize+4], blockHash[:])
	return indexKey
}

//BlockByHeight返回主链中给定高度的块。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) BlockByHeight(blockHeight int32) (*btcutil.Block, error) {
//在最佳链中查找块高度。
	node := b.bestChain.NodeByHeight(blockHeight)
	if node == nil {
		str := fmt.Sprintf("no block at height %d exists", blockHeight)
		return nil, errNotInMainChain(str)
	}

//从数据库加载块并返回它。
	var block *btcutil.Block
	err := b.db.View(func(dbTx database.Tx) error {
		var err error
		block, err = dbFetchBlockByNode(dbTx, node)
		return err
	})
	return block, err
}

//blockbyhash返回具有给定哈希的主链中的块
//设置适当的链条高度。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) BlockByHash(hash *chainhash.Hash) (*btcutil.Block, error) {
//在块索引中查找块哈希并确保它处于最佳状态
//链。
	node := b.index.LookupNode(hash)
	if node == nil || !b.bestChain.Contains(node) {
		str := fmt.Sprintf("block %s is not in the main chain", hash)
		return nil, errNotInMainChain(str)
	}

//从数据库加载块并返回它。
	var block *btcutil.Block
	err := b.db.View(func(dbTx database.Tx) error {
		var err error
		block, err = dbFetchBlockByNode(dbTx, node)
		return err
	})
	return block, err
}
