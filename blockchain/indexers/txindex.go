
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

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

const (
//txindexname是索引的可读名称。
	txIndexName = "transaction index"
)

var (
//
//把它盖起来。
	txIndexKey = []byte("txbyhashidx")

//idbyhashindexbucketname是用于存储的db bucket的名称
//块ID->块哈希索引。
	idByHashIndexBucketName = []byte("idbyhashidx")

//
//
	hashByIDIndexBucketName = []byte("hashbyididx")

//
//
	errNoBlockIDEntry = errors.New("no entry in the block ID index")
)

//————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//
//链。为了显著优化空间需求，一个单独的
//提供每个块之间的内部映射的索引
//
//
//
//索引。
//
//
//
//每个块的散列到唯一ID，第三个散列将ID映射回
//
//
//
//只要具有相同哈希的上一个事务完全是相同的哈希
//
//
//
//
//无论如何，对给定哈希的want是最新的哈希。
//
//
//< hash > = <ID>
//
//字段类型大小
//哈希链哈希。哈希32字节
//ID uint32 4字节
//
//总数：36字节
//
//
//
//
//字段类型大小
//ID uint32 4字节
//哈希链哈希。哈希32字节
//
//总数：36字节
//
//Tx索引桶中键和值的序列化格式为：
//
//
//
//字段类型大小
//txthash chainhash.hash 32字节
//块ID uint32 4字节
//
//Tx长度uint32 4字节
//-----
//总数：44字节
//————————————————————————————————————————————————————————————————————————————————————————————————————————————————

//DBPutBlockIDindExentry使用现有的数据库事务来更新或添加
//提供的哈希到ID和ID到哈希映射的索引项
//价值观。
func dbPutBlockIDIndexEntry(dbTx database.Tx, hash *chainhash.Hash, id uint32) error {
//序列化高度以在索引项中使用。
	var serializedID [4]byte
	byteOrder.PutUint32(serializedID[:], id)

//
	meta := dbTx.Metadata()
	hashIndex := meta.Bucket(idByHashIndexBucketName)
	if err := hashIndex.Put(hash[:], serializedID[:]); err != nil {
		return err
	}

//
	idIndex := meta.Bucket(hashByIDIndexBucketName)
	return idIndex.Put(serializedID[:], hash[:])
}

//
//
func dbRemoveBlockIDIndexEntry(dbTx database.Tx, hash *chainhash.Hash) error {
//
	meta := dbTx.Metadata()
	hashIndex := meta.Bucket(idByHashIndexBucketName)
	serializedID := hashIndex.Get(hash[:])
	if serializedID == nil {
		return nil
	}
	if err := hashIndex.Delete(hash[:]); err != nil {
		return err
	}

//删除块ID到哈希的映射。
	idIndex := meta.Bucket(hashByIDIndexBucketName)
	return idIndex.Delete(serializedID)
}

//
//
func dbFetchBlockIDByHash(dbTx database.Tx, hash *chainhash.Hash) (uint32, error) {
	hashIndex := dbTx.Metadata().Bucket(idByHashIndexBucketName)
	serializedID := hashIndex.Get(hash[:])
	if serializedID == nil {
		return 0, errNoBlockIDEntry
	}

	return byteOrder.Uint32(serializedID), nil
}

//
//从索引中检索提供的序列化块ID的哈希。
func dbFetchBlockHashBySerializedID(dbTx database.Tx, serializedID []byte) (*chainhash.Hash, error) {
	idIndex := dbTx.Metadata().Bucket(hashByIDIndexBucketName)
	hashBytes := idIndex.Get(serializedID)
	if hashBytes == nil {
		return nil, errNoBlockIDEntry
	}

	var hash chainhash.Hash
	copy(hash[:], hashBytes)
	return &hash, nil
}

//dbfetchblockhashbyid使用现有的数据库事务来检索
//
func dbFetchBlockHashByID(dbTx database.Tx, id uint32) (*chainhash.Hash, error) {
	var serializedID [4]byte
	byteOrder.PutUint32(serializedID[:], id)
	return dbFetchBlockHashBySerializedID(dbTx, serializedID[:])
}

//
//
//至少大到足以处理由
//t输入常量，否则会恐慌。
func putTxIndexEntry(target []byte, blockID uint32, txLoc wire.TxLoc) {
	byteOrder.PutUint32(target, blockID)
	byteOrder.PutUint32(target[4:], uint32(txLoc.TxStart))
	byteOrder.PutUint32(target[8:], uint32(txLoc.TxLen))
}

//
//
//已序列化putxtindexentry。
func dbPutTxIndexEntry(dbTx database.Tx, txHash *chainhash.Hash, serializedData []byte) error {
	txIndex := dbTx.Metadata().Bucket(txIndexKey)
	return txIndex.Put(txHash[:], serializedData)
}

//dbfetchtxindextentry使用现有的数据库事务来获取块
//事务索引中提供的事务哈希的区域。什么时候？
//提供的哈希没有条目，两者都将返回nil
//区域和错误。
func dbFetchTxIndexEntry(dbTx database.Tx, txHash *chainhash.Hash) (*database.BlockRegion, error) {
//
	txIndex := dbTx.Metadata().Bucket(txIndexKey)
	serializedData := txIndex.Get(txHash[:])
	if len(serializedData) == 0 {
		return nil, nil
	}

//
	if len(serializedData) < 12 {
		return nil, database.Error{
			ErrorCode: database.ErrCorruption,
			Description: fmt.Sprintf("corrupt transaction index "+
				"entry for %s", txHash),
		}
	}

//加载与块ID关联的块哈希。
	hash, err := dbFetchBlockHashBySerializedID(dbTx, serializedData[0:4])
	if err != nil {
		return nil, database.Error{
			ErrorCode: database.ErrCorruption,
			Description: fmt.Sprintf("corrupt transaction index "+
				"entry for %s: %v", txHash, err),
		}
	}

//反序列化最终条目。
	region := database.BlockRegion{Hash: &chainhash.Hash{}}
	copy(region.Hash[:], hash[:])
	region.Offset = byteOrder.Uint32(serializedData[4:8])
	region.Len = byteOrder.Uint32(serializedData[8:12])

	return &region, nil
}

//
//
func dbAddTxIndexEntries(dbTx database.Tx, block *btcutil.Block, blockID uint32) error {
//序列化中事务的偏移量和长度
//块。
	txLocs, err := block.TxLoc()
	if err != nil {
		return err
	}

//
//块的序列化事务索引项和
//
//要写入的数据库子片。这种方法非常重要
//减少所需分配的数量。
	offset := 0
	serializedValues := make([]byte, len(block.Transactions())*txEntrySize)
	for i, tx := range block.Transactions() {
		putTxIndexEntry(serializedValues[offset:], blockID, txLocs[i])
		endOffset := offset + txEntrySize
		err := dbPutTxIndexEntry(dbTx, tx.Hash(),
			serializedValues[offset:endOffset:endOffset])
		if err != nil {
			return err
		}
		offset += txEntrySize
	}

	return nil
}

//
//
func dbRemoveTxIndexEntry(dbTx database.Tx, txHash *chainhash.Hash) error {
	txIndex := dbTx.Metadata().Bucket(txIndexKey)
	serializedData := txIndex.Get(txHash[:])
	if len(serializedData) == 0 {
		return fmt.Errorf("can't remove non-existent transaction %s "+
			"from the transaction index", txHash)
	}

	return txIndex.Delete(txHash[:])
}

//
//
func dbRemoveTxIndexEntries(dbTx database.Tx, block *btcutil.Block) error {
	for _, tx := range block.Transactions() {
		err := dbRemoveTxIndexEntry(dbTx, tx.Hash())
		if err != nil {
			return err
		}
	}

	return nil
}

//
//
type TxIndex struct {
	db         database.DB
	curBlockID uint32
}

//确保TxIndex类型实现索引器接口。
var _ Indexer = (*TxIndex)(nil)

//
//
//断开块。
//
//这是索引器接口的一部分。
func (idx *TxIndex) Init() error {
//
//
//
//
	err := idx.db.View(func(dbTx database.Tx) error {
//
//
//
		var highestKnown, nextUnknown uint32
		testBlockID := uint32(1)
		increment := uint32(100000)
		for {
			_, err := dbFetchBlockHashByID(dbTx, testBlockID)
			if err != nil {
				nextUnknown = testBlockID
				break
			}

			highestKnown = testBlockID
			testBlockID += increment
		}
		log.Tracef("Forward scan (highest known %d, next unknown %d)",
			highestKnown, nextUnknown)

//由于新数据库，没有使用的块ID。
		if nextUnknown == 1 {
			return nil
		}

//
//
		for {
			testBlockID = (highestKnown + nextUnknown) / 2
			_, err := dbFetchBlockHashByID(dbTx, testBlockID)
			if err != nil {
				nextUnknown = testBlockID
			} else {
				highestKnown = testBlockID
			}
			log.Tracef("Binary scan (highest known %d, next "+
				"unknown %d)", highestKnown, nextUnknown)
			if highestKnown+1 == nextUnknown {
				break
			}
		}

		idx.curBlockID = highestKnown
		return nil
	})
	if err != nil {
		return err
	}

	log.Debugf("Current internal block ID: %d", idx.curBlockID)
	return nil
}

//
//
//这是索引器接口的一部分。
func (idx *TxIndex) Key() []byte {
	return txIndexKey
}

//name返回索引的可读名称。
//
//这是索引器接口的一部分。
func (idx *TxIndex) Name() string {
	return txIndexName
}

//当索引器管理器确定索引需要时调用create
//
//
//
//这是索引器接口的一部分。
func (idx *TxIndex) Create(dbTx database.Tx) error {
	meta := dbTx.Metadata()
	if _, err := meta.CreateBucket(idByHashIndexBucketName); err != nil {
		return err
	}
	if _, err := meta.CreateBucket(hashByIDIndexBucketName); err != nil {
		return err
	}
	_, err := meta.CreateBucket(txIndexKey)
	return err
}

//
//
//
//
//这是索引器接口的一部分。
func (idx *TxIndex) ConnectBlock(dbTx database.Tx, block *btcutil.Block,
	stxos []blockchain.SpentTxOut) error {

//
//
	newBlockID := idx.curBlockID + 1
	if err := dbAddTxIndexEntries(dbTx, block, newBlockID); err != nil {
		return err
	}

//为正在连接的块添加新的块ID索引项，并
//
	err := dbPutBlockIDIndexEntry(dbTx, block.Hash(), newBlockID)
	if err != nil {
		return err
	}
	idx.curBlockID = newBlockID
	return nil
}

//当一个块被
//从主链上断开。此索引器删除
//
//
//这是索引器接口的一部分。
func (idx *TxIndex) DisconnectBlock(dbTx database.Tx, block *btcutil.Block,
	stxos []blockchain.SpentTxOut) error {

//
	if err := dbRemoveTxIndexEntries(dbTx, block); err != nil {
		return err
	}

//
//
	if err := dbRemoveBlockIDIndexEntry(dbTx, block.Hash()); err != nil {
		return err
	}
	idx.curBlockID--
	return nil
}

//
//
//
//
//
//此函数对于并发访问是安全的。
func (idx *TxIndex) TxBlockRegion(hash *chainhash.Hash) (*database.BlockRegion, error) {
	var region *database.BlockRegion
	err := idx.db.View(func(dbTx database.Tx) error {
		var err error
		region, err = dbFetchTxIndexEntry(dbTx, hash)
		return err
	})
	return region, err
}

//
//
//
//
//
//
//
func NewTxIndex(db database.DB) *TxIndex {
	return &TxIndex{db: db}
}

//
func dropBlockIDIndex(db database.DB) error {
	return db.Update(func(dbTx database.Tx) error {
		meta := dbTx.Metadata()
		err := meta.DeleteBucket(idByHashIndexBucketName)
		if err != nil {
			return err
		}

		return meta.DeleteBucket(hashByIDIndexBucketName)
	})
}

//
//
//存在时丢弃。
func DropTxIndex(db database.DB, interrupt <-chan struct{}) error {
	err := dropIndex(db, addrIndexKey, addrIndexName, interrupt)
	if err != nil {
		return err
	}

	return dropIndex(db, txIndexKey, txIndexName, interrupt)
}
