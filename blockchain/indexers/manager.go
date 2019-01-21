
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
	"bytes"
	"fmt"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

var (
//
//
	indexTipsBucketName = []byte("idxtips")
)

//————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//索引管理器通过使用父索引跟踪每个索引的当前提示
//
//
//
//
//
//
//
//
//
//————————————————————————————————————————————————————————————————————————————————————————————————————————————————

//
//
func dbPutIndexerTip(dbTx database.Tx, idxKey []byte, hash *chainhash.Hash, height int32) error {
	serialized := make([]byte, chainhash.HashSize+4)
	copy(serialized, hash[:])
	byteOrder.PutUint32(serialized[chainhash.HashSize:], uint32(height))

	indexesBucket := dbTx.Metadata().Bucket(indexTipsBucketName)
	return indexesBucket.Put(idxKey, serialized)
}

//dbfetchindexertip使用现有的数据库事务来检索
//
func dbFetchIndexerTip(dbTx database.Tx, idxKey []byte) (*chainhash.Hash, int32, error) {
	indexesBucket := dbTx.Metadata().Bucket(indexTipsBucketName)
	serialized := indexesBucket.Get(idxKey)
	if len(serialized) < chainhash.HashSize+4 {
		return nil, 0, database.Error{
			ErrorCode: database.ErrCorruption,
			Description: fmt.Sprintf("unexpected end of data for "+
				"index %q tip", string(idxKey)),
		}
	}

	var hash chainhash.Hash
	copy(hash[:], serialized[:chainhash.HashSize])
	height := int32(byteOrder.Uint32(serialized[chainhash.HashSize:]))
	return &hash, height, nil
}

//
//
//
//
func dbIndexConnectBlock(dbTx database.Tx, indexer Indexer, block *btcutil.Block,
	stxo []blockchain.SpentTxOut) error {

//断言正在连接的块正确连接到
//
	idxKey := indexer.Key()
	curTipHash, _, err := dbFetchIndexerTip(dbTx, idxKey)
	if err != nil {
		return err
	}
	if !curTipHash.IsEqual(&block.MsgBlock().Header.PrevBlock) {
		return AssertError(fmt.Sprintf("dbIndexConnectBlock must be "+
			"called with a block that extends the current index "+
			"tip (%s, tip %s, block %s)", indexer.Name(),
			curTipHash, block.Hash()))
	}

//用连接的块通知索引器，以便索引它。
	if err := indexer.ConnectBlock(dbTx, block, stxo); err != nil {
		return err
	}

//
	return dbPutIndexerTip(dbTx, idxKey, block.Hash(), block.Height())
}

//
//
//因此。如果索引器的当前提示为
//
func dbIndexDisconnectBlock(dbTx database.Tx, indexer Indexer, block *btcutil.Block,
	stxo []blockchain.SpentTxOut) error {

//
//索引。
	idxKey := indexer.Key()
	curTipHash, _, err := dbFetchIndexerTip(dbTx, idxKey)
	if err != nil {
		return err
	}
	if !curTipHash.IsEqual(block.Hash()) {
		return AssertError(fmt.Sprintf("dbIndexDisconnectBlock must "+
			"be called with the block at the current index tip "+
			"(%s, tip %s, block %s)", indexer.Name(),
			curTipHash, block.Hash()))
	}

//
//适当的条目。
	if err := indexer.DisconnectBlock(dbTx, block, stxo); err != nil {
		return err
	}

//
	prevHash := &block.MsgBlock().Header.PrevBlock
	return dbPutIndexerTip(dbTx, idxKey, prevHash, block.Height()-1)
}

//管理器定义一个索引管理器，用于管理多个可选索引和
//
//
type Manager struct {
	db             database.DB
	enabledIndexes []Indexer
}

//
var _ blockchain.IndexManager = (*Manager)(nil)

//indexDropKey返回一个索引的键，该索引指示该索引位于
//被丢弃的过程。
func indexDropKey(idxKey []byte) []byte {
	dropKey := make([]byte, len(idxKey)+1)
	dropKey[0] = 'd'
	copy(dropKey[1:], idxKey)
	return dropKey
}

//
//
//
//一个大的原子步骤，由于大量的条目。
func (m *Manager) maybeFinishDrops(interrupt <-chan struct{}) error {
	indexNeedsDrop := make([]bool, len(m.enabledIndexes))
	err := m.db.View(func(dbTx database.Tx) error {
//
//尚未创建bucket。
		indexesBucket := dbTx.Metadata().Bucket(indexTipsBucketName)
		if indexesBucket == nil {
			return nil
		}

//
//进展。
		for i, indexer := range m.enabledIndexes {
			dropKey := indexDropKey(indexer.Key())
			if indexesBucket.Get(dropKey) != nil {
				indexNeedsDrop[i] = true
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	if interruptRequested(interrupt) {
		return errInterruptRequested
	}

//
//
	for i, indexer := range m.enabledIndexes {
		if !indexNeedsDrop[i] {
			continue
		}

		log.Infof("Resuming %s drop", indexer.Name())
		err := dropIndex(m.db, indexer.Key(), indexer.Name(), interrupt)
		if err != nil {
			return err
		}
	}

	return nil
}

//maybecreateindexes确定每个启用的索引是否已经
//
func (m *Manager) maybeCreateIndexes(dbTx database.Tx) error {
	indexesBucket := dbTx.Metadata().Bucket(indexTipsBucketName)
	for _, indexer := range m.enabledIndexes {
//
		idxKey := indexer.Key()
		if indexesBucket.Get(idxKey) != nil {
			continue
		}

//索引的提示不存在，因此创建它并
//
//
		if err := indexer.Create(dbTx); err != nil {
			return err
		}

//将索引提示设置为表示
//未初始化索引。
		err := dbPutIndexerTip(dbTx, idxKey, &chainhash.Hash{}, -1)
		if err != nil {
			return err
		}
	}

	return nil
}

//init初始化启用的索引。这是在链期间调用的
//
//
//
//
//
//
//这是区块链.indexManager接口的一部分。
func (m *Manager) Init(chain *blockchain.BlockChain, interrupt <-chan struct{}) error {
//没有启用索引时不执行任何操作。
	if len(m.enabledIndexes) == 0 {
		return nil
	}

	if interruptRequested(interrupt) {
		return errInterruptRequested
	}

//
	if err := m.maybeFinishDrops(interrupt); err != nil {
		return err
	}

//
	err := m.db.Update(func(dbTx database.Tx) error {
//
		meta := dbTx.Metadata()
		_, err := meta.CreateBucketIfNotExists(indexTipsBucketName)
		if err != nil {
			return err
		}

		return m.maybeCreateIndexes(dbTx)
	})
	if err != nil {
		return err
	}

//
	for _, indexer := range m.enabledIndexes {
		if err := indexer.Init(); err != nil {
			return err
		}
	}

//
//
//在索引被禁用时重新组织。这必须在
//
	for i := len(m.enabledIndexes); i > 0; i-- {
		indexer := m.enabledIndexes[i-1]

//
		var height int32
		var hash *chainhash.Hash
		err := m.db.View(func(dbTx database.Tx) error {
			idxKey := indexer.Key()
			hash, height, err = dbFetchIndexerTip(dbTx, idxKey)
			return err
		})
		if err != nil {
			return err
		}

//
		if height == -1 {
			continue
		}

//
		initialHeight := height
		for !chain.MainChainHasBlock(hash) {
//
//直接从数据库中孤立块
//
//
//
//错误。
			var block *btcutil.Block
			err := m.db.View(func(dbTx database.Tx) error {
				blockBytes, err := dbTx.FetchBlock(hash)
				if err != nil {
					return err
				}
				block, err = btcutil.NewBlockFromBytes(blockBytes)
				if err != nil {
					return err
				}
				block.SetHeight(height)
				return err
			})
			if err != nil {
				return err
			}

//我们还将获取一组输出
//
			spentTxos, err := chain.FetchSpendJournal(block)
			if err != nil {
				return err
			}

//
//我们现在可以更新索引本身了。
			err = m.db.Update(func(dbTx database.Tx) error {
//删除所有关联的索引项
//
				err = dbIndexDisconnectBlock(
					dbTx, indexer, block, spentTxos,
				)
				if err != nil {
					return err
				}

//将提示更新到上一个块。
				hash = &block.MsgBlock().Header.PrevBlock
				height--

				return nil
			})
			if err != nil {
				return err
			}

			if interruptRequested(interrupt) {
				return errInterruptRequested
			}
		}

		if initialHeight != height {
			log.Infof("Removed %d orphaned blocks from %s "+
				"(heights %d to %d)", initialHeight-height,
				indexer.Name(), height+1, initialHeight)
		}
	}

//
//最低的一个，所以捕获代码只需要最早开始
//块，并能够跳过连接块的索引
//
	bestHeight := chain.BestSnapshot().Height
	lowestHeight := bestHeight
	indexerHeights := make([]int32, len(m.enabledIndexes))
	err = m.db.View(func(dbTx database.Tx) error {
		for i, indexer := range m.enabledIndexes {
			idxKey := indexer.Key()
			hash, height, err := dbFetchIndexerTip(dbTx, idxKey)
			if err != nil {
				return err
			}

			log.Debugf("Current %s tip (height %d, hash %v)",
				indexer.Name(), height, hash)
			indexerHeights[i] = height
			if height < lowestHeight {
				lowestHeight = height
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

//如果所有索引都被捕获，则没有要索引的内容。
	if lowestHeight == bestHeight {
		return nil
	}

//
	progressLogger := newBlockProgressLogger("Indexed", log)

//
//提示并需要被关注，因此记录详细信息并循环
//
	log.Infof("Catching up indexes from height %d to %d", lowestHeight,
		bestHeight)
	for height := lowestHeight + 1; height <= bestHeight; height++ {
//
//它。
		block, err := chain.BlockByHeight(height)
		if err != nil {
			return err
		}

		if interruptRequested(interrupt) {
			return errInterruptRequested
		}

//
		var spentTxos []blockchain.SpentTxOut
		for i, indexer := range m.enabledIndexes {
//跳过不需要用此更新的索引
//块。
			if indexerHeights[i] >= height {
				continue
			}

//当索引需要所有引用的txout时
//
//从支出日记帐中检索。
			if spentTxos == nil && indexNeedsInputs(indexer) {
				spentTxos, err = chain.FetchSpendJournal(block)
				if err != nil {
					return err
				}
			}

			err := m.db.Update(func(dbTx database.Tx) error {
				return dbIndexConnectBlock(
					dbTx, indexer, block, spentTxos,
				)
			})
			if err != nil {
				return err
			}
			indexerHeights[i] = height
		}

//
		progressLogger.LogBlockHeight(block)

		if interruptRequested(interrupt) {
			return errInterruptRequested
		}
	}

	log.Infof("Indexes caught up to height %d", bestHeight)
	return nil
}

//
//被正在索引的事务输入引用。
func indexNeedsInputs(index Indexer) bool {
	if idx, ok := index.(NeedsInputser); ok {
		return idx.NeedsInputs()
	}

	return false
}

//
//
func dbFetchTx(dbTx database.Tx, hash *chainhash.Hash) (*wire.MsgTx, error) {
//
	blockRegion, err := dbFetchTxIndexEntry(dbTx, hash)
	if err != nil {
		return nil, err
	}
	if blockRegion == nil {
		return nil, fmt.Errorf("transaction %v not found", hash)
	}

//
	txBytes, err := dbTx.FetchBlockRegion(blockRegion)
	if err != nil {
		return nil, err
	}

//
	var msgTx wire.MsgTx
	err = msgTx.Deserialize(bytes.NewReader(txBytes))
	if err != nil {
		return nil, err
	}

	return &msgTx, nil
}

//
//
//检查并调用每个索引器。
//
//这是区块链.indexManager接口的一部分。
func (m *Manager) ConnectBlock(dbTx database.Tx, block *btcutil.Block,
	stxos []blockchain.SpentTxOut) error {

//
//
	for _, index := range m.enabledIndexes {
		err := dbIndexConnectBlock(dbTx, index, block, stxos)
		if err != nil {
			return err
		}
	}
	return nil
}

//
//主链的末端。它跟踪每个索引的状态
//
//
//
//这是区块链.indexManager接口的一部分。
func (m *Manager) DisconnectBlock(dbTx database.Tx, block *btcutil.Block,
	stxo []blockchain.SpentTxOut) error {

//
//断开连接，以便它们可以相应地更新。
	for _, index := range m.enabledIndexes {
		err := dbIndexDisconnectBlock(dbTx, index, block, stxo)
		if err != nil {
			return err
		}
	}
	return nil
}

//NewManager返回启用了所提供索引的新索引管理器。
//
//
//干净地插入正常的区块链处理路径。
func NewManager(db database.DB, enabledIndexes []Indexer) *Manager {
	return &Manager{
		db:             db,
		enabledIndexes: enabledIndexes,
	}
}

//DropIndex从数据库中删除传递的索引。因为索引可以
//大量，它删除多个数据库事务中的索引，以便
//将内存使用保持在合理的水平。它也标志着正在下降
//
//
func dropIndex(db database.DB, idxKey []byte, idxName string, interrupt <-chan struct{}) error {
//如果索引不存在，则不执行任何操作。
	var needsDelete bool
	err := db.View(func(dbTx database.Tx) error {
		indexesBucket := dbTx.Metadata().Bucket(indexTipsBucketName)
		if indexesBucket != nil && indexesBucket.Get(idxKey) != nil {
			needsDelete = true
		}
		return nil
	})
	if err != nil {
		return err
	}
	if !needsDelete {
		log.Infof("Not dropping %s because it does not exist", idxName)
		return nil
	}

//标记索引正在被删除，以便
//
//完成。
	log.Infof("Dropping all %s entries.  This might take a while...",
		idxName)
	err = db.Update(func(dbTx database.Tx) error {
		indexesBucket := dbTx.Metadata().Bucket(indexTipsBucketName)
		return indexesBucket.Put(indexDropKey(idxKey), idxKey)
	})
	if err != nil {
		return err
	}

//
//单个数据库事务中的bucket将导致大量
//由于ulimits，内存使用和可能导致许多系统崩溃。整齐
//
//
//
	const maxDeletions = 2000000
	var totalDeleted uint64

//
//后来删除。
	var subBuckets [][][]byte
	var subBucketClosure func(database.Tx, []byte, [][]byte) error
	subBucketClosure = func(dbTx database.Tx,
		subBucket []byte, tlBucket [][]byte) error {
//获取完整的bucket名称并附加到子bucket以备以后使用
//
		var bucketName [][]byte
		if (tlBucket == nil) || (len(tlBucket) == 0) {
			bucketName = append(bucketName, subBucket)
		} else {
			bucketName = append(tlBucket, subBucket)
		}
		subBuckets = append(subBuckets, bucketName)
//
		bucket := dbTx.Metadata()
		for _, subBucketName := range bucketName {
			bucket = bucket.Bucket(subBucketName)
		}
		return bucket.ForEachBucket(func(k []byte) error {
			return subBucketClosure(dbTx, k, bucketName)
		})
	}

//
	err = db.View(func(dbTx database.Tx) error {
		return subBucketClosure(dbTx, idxKey, nil)
	})
	if err != nil {
		return nil
	}

//
//
	for i := range subBuckets {
		bucketName := subBuckets[len(subBuckets)-1-i]
//一次删除MaxDeletions键/值对。
		for numDeleted := maxDeletions; numDeleted == maxDeletions; {
			numDeleted = 0
			err := db.Update(func(dbTx database.Tx) error {
				subBucket := dbTx.Metadata()
				for _, subBucketName := range bucketName {
					subBucket = subBucket.Bucket(subBucketName)
				}
				cursor := subBucket.Cursor()
				for ok := cursor.First(); ok; ok = cursor.Next() &&
					numDeleted < maxDeletions {

					if err := cursor.Delete(); err != nil {
						return err
					}
					numDeleted++
				}
				return nil
			})
			if err != nil {
				return err
			}

			if numDeleted > 0 {
				totalDeleted += uint64(numDeleted)
				log.Infof("Deleted %d keys (%d total) from %s",
					numDeleted, totalDeleted, idxName)
			}
		}

		if interruptRequested(interrupt) {
			return errInterruptRequested
		}

//
		err = db.Update(func(dbTx database.Tx) error {
			bucket := dbTx.Metadata()
			for j := 0; j < len(bucketName)-1; j++ {
				bucket = bucket.Bucket(bucketName[j])
			}
			return bucket.DeleteBucket(bucketName[len(bucketName)-1])
		})
	}

//
	if idxName == txIndexName {
		if err := dropBlockIDIndex(db); err != nil {
			return err
		}
	}

//立即删除索引提示、索引存储桶和进行中删除标志
//
	err = db.Update(func(dbTx database.Tx) error {
		meta := dbTx.Metadata()
		indexesBucket := meta.Bucket(indexTipsBucketName)
		if err := indexesBucket.Delete(idxKey); err != nil {
			return err
		}

		return indexesBucket.Delete(indexDropKey(idxKey))
	})
	if err != nil {
		return err
	}

	log.Infof("Dropped %s", idxName)
	return nil
}
