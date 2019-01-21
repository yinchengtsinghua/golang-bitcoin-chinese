
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

package indexers

import (
	"errors"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/gcs"
	"github.com/btcsuite/btcutil/gcs/builder"
)

const (
//
	cfIndexName = "committed filter index"
)

//
//和成对丢弃，并且都由块的哈希索引。此外
//
var (
//
//
	cfIndexParentBucketKey = []byte("cfindexparentbucket")

//
//
	cfIndexKeys = [][]byte{
		[]byte("cf0byhashidx"),
	}

//cfHeaderKeys是一个数据库存储桶名称数组，用于存储
//块散列到CF头。
	cfHeaderKeys = [][]byte{
		[]byte("cf0headerbyhashidx"),
	}

//cfhashkeys是一个数据库存储桶名称数组，用于存储
//block hashes to cf hashes.
	cfHashKeys = [][]byte{
		[]byte("cf0hashbyhashidx"),
	}

	maxFilterType = uint8(len(cfHeaderKeys) - 1)

//zero hash是chainhash。此处定义的所有零字节的哈希值
//为了方便。
	zeroHash chainhash.Hash
)

//从数据库索引数据库中检索一个数据块。
//条目的缺失不被视为错误。
func dbFetchFilterIdxEntry(dbTx database.Tx, key []byte, h *chainhash.Hash) ([]byte, error) {
	idx := dbTx.Metadata().Bucket(cfIndexParentBucketKey).Bucket(key)
	return idx.Get(h[:]), nil
}

//dbstorefilteridxentry将数据blob存储在筛选器索引数据库中。
func dbStoreFilterIdxEntry(dbTx database.Tx, key []byte, h *chainhash.Hash, f []byte) error {
	idx := dbTx.Metadata().Bucket(cfIndexParentBucketKey).Bucket(key)
	return idx.Put(h[:], f)
}

//dbdeletefilteridxentry从筛选器索引数据库中删除数据blob。
func dbDeleteFilterIdxEntry(dbTx database.Tx, key []byte, h *chainhash.Hash) error {
	idx := dbTx.Metadata().Bucket(cfIndexParentBucketKey).Bucket(key)
	return idx.Delete(h[:])
}

//cf index通过哈希索引实现提交的过滤器（cf）。
type CfIndex struct {
	db          database.DB
	chainParams *chaincfg.Params
}

//确保cfindex类型实现索引器接口。
var _ Indexer = (*CfIndex)(nil)

//确保cfindex类型实现NeedsInputser接口。
var _ NeedsInputser = (*CfIndex)(nil)

//NeedsInput表示索引需要按顺序引用输入
//
//
//
func (idx *CfIndex) NeedsInputs() bool {
	return true
}

//
//接口。
func (idx *CfIndex) Init() error {
return nil //无事可做。
}

//
//
func (idx *CfIndex) Key() []byte {
	return cfIndexParentBucketKey
}

//
//
func (idx *CfIndex) Name() string {
	return cfIndexName
}

//
//
//
func (idx *CfIndex) Create(dbTx database.Tx) error {
	meta := dbTx.Metadata()

	cfIndexParentBucket, err := meta.CreateBucket(cfIndexParentBucketKey)
	if err != nil {
		return err
	}

	for _, bucketName := range cfIndexKeys {
		_, err = cfIndexParentBucket.CreateBucket(bucketName)
		if err != nil {
			return err
		}
	}

	for _, bucketName := range cfHeaderKeys {
		_, err = cfIndexParentBucket.CreateBucket(bucketName)
		if err != nil {
			return err
		}
	}

	for _, bucketName := range cfHashKeys {
		_, err = cfIndexParentBucket.CreateBucket(bucketName)
		if err != nil {
			return err
		}
	}

	return nil
}

//
//
func storeFilter(dbTx database.Tx, block *btcutil.Block, f *gcs.Filter,
	filterType wire.FilterType) error {
	if uint8(filterType) > maxFilterType {
		return errors.New("unsupported filter type")
	}

//找出要使用的桶。
	fkey := cfIndexKeys[filterType]
	hkey := cfHeaderKeys[filterType]
	hashkey := cfHashKeys[filterType]

//
	h := block.Hash()
	filterBytes, err := f.NBytes()
	if err != nil {
		return err
	}
	err = dbStoreFilterIdxEntry(dbTx, fkey, h, filterBytes)
	if err != nil {
		return err
	}

//
	filterHash, err := builder.GetFilterHash(f)
	if err != nil {
		return err
	}
	err = dbStoreFilterIdxEntry(dbTx, hashkey, h, filterHash[:])
	if err != nil {
		return err
	}

//
	var prevHeader *chainhash.Hash
	ph := &block.MsgBlock().Header.PrevBlock
	if ph.IsEqual(&zeroHash) {
		prevHeader = &zeroHash
	} else {
		pfh, err := dbFetchFilterIdxEntry(dbTx, hkey, ph)
		if err != nil {
			return err
		}

//构造新块的筛选器头并存储它。
		prevHeader, err = chainhash.NewHash(pfh)
		if err != nil {
			return err
		}
	}

	fh, err := builder.MakeHeaderForFilter(f, *prevHeader)
	if err != nil {
		return err
	}
	return dbStoreFilterIdxEntry(dbTx, hkey, h, fh[:])
}

//
//
//
func (idx *CfIndex) ConnectBlock(dbTx database.Tx, block *btcutil.Block,
	stxos []blockchain.SpentTxOut) error {

	prevScripts := make([][]byte, len(stxos))
	for i, stxo := range stxos {
		prevScripts[i] = stxo.PkScript
	}

	f, err := builder.BuildBasicFilter(block.MsgBlock(), prevScripts)
	if err != nil {
		return err
	}

	return storeFilter(dbTx, block, f, wire.GCSFilterRegular)
}

//当一个块被
//
//
func (idx *CfIndex) DisconnectBlock(dbTx database.Tx, block *btcutil.Block,
	_ []blockchain.SpentTxOut) error {

	for _, key := range cfIndexKeys {
		err := dbDeleteFilterIdxEntry(dbTx, key, block.Hash())
		if err != nil {
			return err
		}
	}

	for _, key := range cfHeaderKeys {
		err := dbDeleteFilterIdxEntry(dbTx, key, block.Hash())
		if err != nil {
			return err
		}
	}

	for _, key := range cfHashKeys {
		err := dbDeleteFilterIdxEntry(dbTx, key, block.Hash())
		if err != nil {
			return err
		}
	}

	return nil
}

//
//
func (idx *CfIndex) entryByBlockHash(filterTypeKeys [][]byte,
	filterType wire.FilterType, h *chainhash.Hash) ([]byte, error) {

	if uint8(filterType) > maxFilterType {
		return nil, errors.New("unsupported filter type")
	}
	key := filterTypeKeys[filterType]

	var entry []byte
	err := idx.db.View(func(dbTx database.Tx) error {
		var err error
		entry, err = dbFetchFilterIdxEntry(dbTx, key, h)
		return err
	})
	return entry, err
}

//
//
func (idx *CfIndex) entriesByBlockHashes(filterTypeKeys [][]byte,
	filterType wire.FilterType, blockHashes []*chainhash.Hash) ([][]byte, error) {

	if uint8(filterType) > maxFilterType {
		return nil, errors.New("unsupported filter type")
	}
	key := filterTypeKeys[filterType]

	entries := make([][]byte, 0, len(blockHashes))
	err := idx.db.View(func(dbTx database.Tx) error {
		for _, blockHash := range blockHashes {
			entry, err := dbFetchFilterIdxEntry(dbTx, key, blockHash)
			if err != nil {
				return err
			}
			entries = append(entries, entry)
		}
		return nil
	})
	return entries, err
}

//
//
func (idx *CfIndex) FilterByBlockHash(h *chainhash.Hash,
	filterType wire.FilterType) ([]byte, error) {
	return idx.entryByBlockHash(cfIndexKeys, filterType, h)
}

//filtersbyblockhashes返回块的基本或
//
func (idx *CfIndex) FiltersByBlockHashes(blockHashes []*chainhash.Hash,
	filterType wire.FilterType) ([][]byte, error) {
	return idx.entriesByBlockHashes(cfIndexKeys, filterType, blockHashes)
}

//filterHeaderByBlockHash返回块的基本
//
func (idx *CfIndex) FilterHeaderByBlockHash(h *chainhash.Hash,
	filterType wire.FilterType) ([]byte, error) {
	return idx.entryByBlockHash(cfHeaderKeys, filterType, h)
}

//filterheadersbyblockhashes返回块的序列化内容
//
func (idx *CfIndex) FilterHeadersByBlockHashes(blockHashes []*chainhash.Hash,
	filterType wire.FilterType) ([][]byte, error) {
	return idx.entriesByBlockHashes(cfHeaderKeys, filterType, blockHashes)
}

//
//
func (idx *CfIndex) FilterHashByBlockHash(h *chainhash.Hash,
	filterType wire.FilterType) ([]byte, error) {
	return idx.entryByBlockHash(cfHashKeys, filterType, h)
}

//
//
func (idx *CfIndex) FilterHashesByBlockHashes(blockHashes []*chainhash.Hash,
	filterType wire.FilterType) ([][]byte, error) {
	return idx.entriesByBlockHashes(cfHashKeys, filterType, blockHashes)
}

//
//将区块链中所有区块的散列映射到它们各自的散列
//提交的筛选器。
//
//
//
//
func NewCfIndex(db database.DB, chainParams *chaincfg.Params) *CfIndex {
	return &CfIndex{db: db, chainParams: chainParams}
}

//如果存在，DropCfIndex将从提供的数据库中删除CF索引。
func DropCfIndex(db database.DB, interrupt <-chan struct{}) error {
	return dropIndex(db, cfIndexParentBucketKey, cfIndexName, interrupt)
}
