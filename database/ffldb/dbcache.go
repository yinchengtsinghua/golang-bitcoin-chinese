
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package ffldb

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/btcsuite/btcd/database/internal/treap"
	"github.com/btcsuite/goleveldb/leveldb"
	"github.com/btcsuite/goleveldb/leveldb/iterator"
	"github.com/btcsuite/goleveldb/leveldb/util"
)

const (
//DefaultCacheSize是数据库缓存的默认大小。
defaultCacheSize = 100 * 1024 * 1024 //100兆字节

//DefaultFlushsecs是用作
//当缓存大小为
//未超过。
defaultFlushSecs = 300 //5分钟

//ldbbatchheadersize是级别ldb批处理头的大小，该批处理头
//包括序列头和记录计数器。
//
//ldbrecordikeysize是级别ldb内部使用的ikey的大小
//将记录追加到批处理时。
//
//这些用于帮助在一个批次中预先分配所需的空间
//分配而不是让LEVELDB自身不断增长。
//这使得GC上的压力要小得多，因此有助于
//防止GC分配大量多余的空间。
	ldbBatchHeaderSize = 12
	ldbRecordIKeySize  = 8
)

//ldbcacheiter包装Treap迭代器以提供附加功能
//需要满足leveldb iterator.iterator接口。
type ldbCacheIter struct {
	*treap.Iterator
}

//强制LdbCacheIterator实现LevelDB Iterator.Iterator接口。
var _ iterator.Iterator = (*ldbCacheIter)(nil)

//提供的错误仅用于满足迭代器接口，因为没有
//仅此内存结构的错误。
//
//这是leveldb iterator.iterator接口实现的一部分。
func (iter *ldbCacheIter) Error() error {
	return nil
}

//提供setreleaser只是为了满足迭代器接口，因为没有
//需要覆盖它。
//
//这是leveldb iterator.iterator接口实现的一部分。
func (iter *ldbCacheIter) SetReleaser(releaser util.Releaser) {
}

//提供release只是为了满足迭代器接口。
//
//这是leveldb iterator.iterator接口实现的一部分。
func (iter *ldbCacheIter) Release() {
}

//newldbcacheiter针对
//已传递缓存快照的挂起键，并将其包装在
//因此它可以用作LEVELDB迭代器。
func newLdbCacheIter(snap *dbCacheSnapshot, slice *util.Range) *ldbCacheIter {
	iter := snap.pendingKeys.Iterator(slice.Start, slice.Limit)
	return &ldbCacheIter{Iterator: iter}
}

//dbcache迭代器在数据库中的键/值对上定义迭代器
//缓存和基础数据库。
type dbCacheIterator struct {
	cacheSnapshot *dbCacheSnapshot
	dbIter        iterator.Iterator
	cacheIter     iterator.Iterator
	currentIter   iterator.Iterator
	released      bool
}

//强制dbcache迭代器实现LevelDB迭代器.迭代器接口。
var _ iterator.Iterator = (*dbCacheIterator)(nil)

//SkipEndingUpdates跳过当前数据库迭代器位置的任何键
//缓存正在更新。Forwards标志表示
//迭代器移动的方向。
func (iter *dbCacheIterator) skipPendingUpdates(forwards bool) {
	for iter.dbIter.Valid() {
		var skip bool
		key := iter.dbIter.Key()
		if iter.cacheSnapshot.pendingRemove.Has(key) {
			skip = true
		} else if iter.cacheSnapshot.pendingKeys.Has(key) {
			skip = true
		}
		if !skip {
			break
		}

		if forwards {
			iter.dbIter.Next()
		} else {
			iter.dbIter.Prev()
		}
	}
}

//ChooseIterator首先跳过数据库迭代器中
//由缓存更新并将当前迭代器设置为适当的
//迭代器取决于它们的有效性和它们在取时的比较顺序
//考虑到方向标志。向前移动迭代器时
//两个迭代器都有效，选择具有较小键的迭代器，然后
//当迭代器向后移动时，反之亦然。
func (iter *dbCacheIterator) chooseIterator(forwards bool) bool {
//跳过当前数据库迭代器位置的任何键
//正在被缓存更新。
	iter.skipPendingUpdates(forwards)

//当两个迭代器都用完时，迭代器也会用完。
	if !iter.dbIter.Valid() && !iter.cacheIter.Valid() {
		iter.currentIter = nil
		return false
	}

//当缓存迭代器用完时，选择数据库迭代器。
	if !iter.cacheIter.Valid() {
		iter.currentIter = iter.dbIter
		return true
	}

//当数据库迭代器用完时，选择缓存迭代器。
	if !iter.dbIter.Valid() {
		iter.currentIter = iter.cacheIter
		return true
	}

//两个迭代器都是有效的，因此请使用
//较小或较大的键，取决于前进标志。
	compare := bytes.Compare(iter.dbIter.Key(), iter.cacheIter.Key())
	if (forwards && compare > 0) || (!forwards && compare < 0) {
		iter.currentIter = iter.cacheIter
	} else {
		iter.currentIter = iter.dbIter
	}
	return true
}

//首先将迭代器定位在第一个键/值对上，并返回
//或者这对不存在。
//
//这是leveldb iterator.iterator接口实现的一部分。
func (iter *dbCacheIterator) First() bool {
//在数据库和缓存迭代器中查找第一个键
//选择既有效又具有较小键的迭代器。
	iter.dbIter.First()
	iter.cacheIter.First()
	return iter.chooseIterator(true)
}

//Last将迭代器定位在最后一个键/值对上，并返回
//这对不存在。
//
//这是leveldb iterator.iterator接口实现的一部分。
func (iter *dbCacheIterator) Last() bool {
//在数据库和缓存迭代器中查找最后一个键
//选择既有效又具有较大键的迭代器。
	iter.dbIter.Last()
	iter.cacheIter.Last()
	return iter.chooseIterator(false)
}

//下一步将迭代器向前移动一个键/值对，并返回是否
//这对存在。
//
//这是leveldb iterator.iterator接口实现的一部分。
func (iter *dbCacheIterator) Next() bool {
//如果光标用完，则不返回任何内容。
	if iter.currentIter == nil {
		return false
	}

//将当前迭代器移动到下一个条目并选择迭代器
//它既有效又有较小的密钥。
	iter.currentIter.Next()
	return iter.chooseIterator(true)
}

//prev将迭代器向后移动一个键/值对，并返回
//这对不存在。
//
//这是leveldb iterator.iterator接口实现的一部分。
func (iter *dbCacheIterator) Prev() bool {
//如果光标用完，则不返回任何内容。
	if iter.currentIter == nil {
		return false
	}

//将当前迭代器移动到上一个条目并选择
//同时有效且具有较大键的迭代器。
	iter.currentIter.Prev()
	return iter.chooseIterator(false)
}

//seek将迭代器定位在大于
//或等于传递的seek键。如果找不到合适的密钥，则返回false。
//
//这是leveldb iterator.iterator接口实现的一部分。
func (iter *dbCacheIterator) Seek(key []byte) bool {
//在数据库和缓存迭代器中查找提供的键
//然后选择既有效又具有较大键的迭代器。
	iter.dbIter.Seek(key)
	iter.cacheIter.Seek(key)
	return iter.chooseIterator(true)
}

//valid指示迭代器是否定位在有效的键/值对上。
//当新创建或耗尽迭代器时，它将被视为无效。
//
//这是leveldb iterator.iterator接口实现的一部分。
func (iter *dbCacheIterator) Valid() bool {
	return iter.currentIter != nil
}

//键返回迭代器指向的当前键。
//
//这是leveldb iterator.iterator接口实现的一部分。
func (iter *dbCacheIterator) Key() []byte {
//如果迭代器耗尽，则不返回任何内容。
	if iter.currentIter == nil {
		return nil
	}

	return iter.currentIter.Key()
}

//值返回迭代器指向的当前值。
//
//这是leveldb iterator.iterator接口实现的一部分。
func (iter *dbCacheIterator) Value() []byte {
//如果迭代器耗尽，则不返回任何内容。
	if iter.currentIter == nil {
		return nil
	}

	return iter.currentIter.Value()
}

//提供setreleaser只是为了满足迭代器接口，因为没有
//需要覆盖它。
//
//这是leveldb iterator.iterator接口实现的一部分。
func (iter *dbCacheIterator) SetReleaser(releaser util.Releaser) {
}

//release通过从中移除基础的treap迭代器来释放迭代器
//针对挂起的键treap的活动迭代器列表。
//
//这是leveldb iterator.iterator接口实现的一部分。
func (iter *dbCacheIterator) Release() {
	if !iter.released {
		iter.dbIter.Release()
		iter.cacheIter.Release()
		iter.currentIter = nil
		iter.released = true
	}
}

//提供的错误仅用于满足迭代器接口，因为没有
//仅此内存结构的错误。
//
//这是leveldb iterator.iterator接口实现的一部分。
func (iter *dbCacheIterator) Error() error {
	return nil
}

//dbcachesnapshot定义数据库缓存和底层的快照
//特定时间点的数据库。
type dbCacheSnapshot struct {
	dbSnapshot    *leveldb.Snapshot
	pendingKeys   *treap.Immutable
	pendingRemove *treap.Immutable
}

//has返回传递的键是否存在。
func (snap *dbCacheSnapshot) Has(key []byte) bool {
//首先检查缓存项。
	if snap.pendingRemove.Has(key) {
		return false
	}
	if snap.pendingKeys.Has(key) {
		return true
	}

//查阅数据库。
	hasKey, _ := snap.dbSnapshot.Has(key, nil)
	return hasKey
}

//get返回传递的键的值。当
//密钥不存在。
func (snap *dbCacheSnapshot) Get(key []byte) []byte {
//首先检查缓存项。
	if snap.pendingRemove.Has(key) {
		return nil
	}
	if value := snap.pendingKeys.Get(key); value != nil {
		return value
	}

//查阅数据库。
	value, err := snap.dbSnapshot.Get(key, nil)
	if err != nil {
		return nil
	}
	return value
}

//释放释放快照。
func (snap *dbCacheSnapshot) Release() {
	snap.dbSnapshot.Release()
	snap.pendingKeys = nil
	snap.pendingRemove = nil
}

//NewIterator返回快照的新迭代器。新回来的
//在调用某个方法之前，迭代器没有指向有效项
//以确定位置。
//
//slice参数允许将迭代器限制在键的范围内。
//开始键是包含的，限制键是独占的。或两者兼而有之
//如果不需要功能，则可以为零。
func (snap *dbCacheSnapshot) NewIterator(slice *util.Range) *dbCacheIterator {
	return &dbCacheIterator{
		dbIter:        snap.dbSnapshot.NewIterator(slice, nil),
		cacheIter:     newLdbCacheIter(snap, slice),
		cacheSnapshot: snap,
	}
}

//dbcache提供由基础数据库支持的数据库缓存层。它
//允许指定最大缓存大小和刷新间隔，以便
//当缓存大小超过最大值时，将缓存刷新到数据库
//配置的值，或者自
//最后刷新。这有效地提供了事务批处理，以便调用方
//可以随意提交事务，而不会导致大量性能命中
//频繁的磁盘同步。
type dbCache struct {
//ldb是元数据的底层ldb db。
	ldb *leveldb.DB

//存储用于将块同步到平面文件。
	store *blockStore

//以下字段与将缓存刷新为持久性相关
//存储。注意，所有冲洗都是在机会主义的情况下进行的。
//时尚。这意味着它只在事务或
//当数据库缓存关闭时。
//
//MaxSize是缓存可以增长到的最大大小阈值
//它被冲洗了。
//
//Flushinterval是允许
//在刷新缓存之前通过。
//
//lastflush是缓存上次刷新的时间。它用于
//结合当前时间和刷新间隔。
//
//注意：这些与刷新相关的字段受数据库写入保护。
//锁。
	maxSize       uint64
	flushInterval time.Duration
	lastFlush     time.Time

//以下字段包含需要存储或删除的键
//一旦缓存已满，就有足够的时间
//传递，或在数据库关闭时。注意这些是
//使用不可变的treaps存储以支持o（1）MVCC快照
//缓存的数据。cachelock用于保护并发访问
//用于缓存更新和快照。
	cacheLock    sync.RWMutex
	cachedKeys   *treap.Immutable
	cachedRemove *treap.Immutable
}

//快照返回位于的数据库缓存和基础数据库的快照
//特定的时间点。
//
//使用后必须通过调用release释放快照。
func (c *dbCache) Snapshot() (*dbCacheSnapshot, error) {
	dbSnapshot, err := c.ldb.GetSnapshot()
	if err != nil {
		str := "failed to open transaction"
		return nil, convertErr(str, err)
	}

//由于要添加和删除的缓存键使用不可变的treap，
//快照只是获取锁下树的根目录。
//用于原子交换根。
	c.cacheLock.RLock()
	cacheSnapshot := &dbCacheSnapshot{
		dbSnapshot:    dbSnapshot,
		pendingKeys:   c.cachedKeys,
		pendingRemove: c.cachedRemove,
	}
	c.cacheLock.RUnlock()
	return cacheSnapshot, nil
}

//updatedb在托管级别db的上下文中调用传递的函数
//交易。从用户提供的函数返回的任何错误都将导致
//要回滚并从此函数返回的事务。
//否则，当用户提供的函数
//返回零错误。
func (c *dbCache) updateDB(fn func(ldbTx *leveldb.Transaction) error) error {
//启动LevelDB事务。
	ldbTx, err := c.ldb.OpenTransaction()
	if err != nil {
		return convertErr("failed to open ldb transaction", err)
	}

	if err := fn(ldbTx); err != nil {
		ldbTx.Discard()
		return err
	}

//提交LEVELDB事务并根据需要转换任何错误。
	if err := ldbTx.Commit(); err != nil {
		return convertErr("failed to commit leveldb transaction", err)
	}
	return nil
}

//treapforeacher是一个接口，允许在升序中迭代treap
//为每个键/值对使用用户提供的回调进行订购。主要是
//存在，因此可变和不变的叛国者都可以原子地致力于
//具有相同功能的数据库。
type TreapForEacher interface {
	ForEach(func(k, v []byte) bool)
}

//committreaps自动提交所有传递的挂起的添加/更新/删除
//对基础数据库的更新。
func (c *dbCache) commitTreaps(pendingKeys, pendingRemove TreapForEacher) error {
//使用原子事务执行所有级别的数据库更新。
	return c.updateDB(func(ldbTx *leveldb.Transaction) error {
		var innerErr error
		pendingKeys.ForEach(func(k, v []byte) bool {
			if dbErr := ldbTx.Put(k, v, nil); dbErr != nil {
				str := fmt.Sprintf("failed to put key %q to "+
					"ldb transaction", k)
				innerErr = convertErr(str, dbErr)
				return false
			}
			return true
		})
		if innerErr != nil {
			return innerErr
		}

		pendingRemove.ForEach(func(k, v []byte) bool {
			if dbErr := ldbTx.Delete(k, nil); dbErr != nil {
				str := fmt.Sprintf("failed to delete "+
					"key %q from ldb transaction",
					k)
				innerErr = convertErr(str, dbErr)
				return false
			}
			return true
		})
		return innerErr
	})
}

//刷新将数据库缓存刷新到持久存储。此调用同步
//块存储并重放已应用于
//缓存到基础数据库。
//
//必须在保持数据库写锁的情况下调用此函数。
func (c *dbCache) flush() error {
	c.lastFlush = time.Now()

//同步与块存储关联的当前写入文件。这是
//在写入元数据之前必须防止
//元数据包含有关实际没有的块的信息
//在意外关闭场景中编写。
	if err := c.store.syncBlocks(); err != nil {
		return err
	}

//由于要添加和删除的缓存键使用不可变的treap，
//快照只是获取锁下树的根目录。
//用于原子交换根。
	c.cacheLock.RLock()
	cachedKeys := c.cachedKeys
	cachedRemove := c.cachedRemove
	c.cacheLock.RUnlock()

//如果没有要刷新的数据，则不执行任何操作。
	if cachedKeys.Len() == 0 && cachedRemove.Len() == 0 {
		return nil
	}

//使用原子事务执行所有级别的数据库更新。
	if err := c.commitTreaps(cachedKeys, cachedRemove); err != nil {
		return err
	}

//清除缓存，因为它已被刷新。
	c.cacheLock.Lock()
	c.cachedKeys = treap.NewImmutable()
	c.cachedRemove = treap.NewImmutable()
	c.cacheLock.Unlock()

	return nil
}

//NeedsFlush返回是否需要将数据库缓存刷新到
//基于当前大小的持久存储，无论是否添加
//传递的数据库事务中的条目将导致它超过
//配置的限制，以及自上次缓存以来经过的时间
//脸红了。
//
//必须在保持数据库写锁的情况下调用此函数。
func (c *dbCache) needsFlush(tx *transaction) bool {
//当经过的时间超过配置的时间时，需要刷新。
//冲洗间隔。
	if time.Since(c.lastFlush) > c.flushInterval {
		return true
	}

//当数据库缓存的大小超过
//指定的最大缓存大小。总计算大小乘以
//1.5此处说明将要
//在刷新期间需要，以及缓存中的旧节点
//由事务使用的快照引用。
	snap := tx.snapshot
	totalSize := snap.pendingKeys.Size() + snap.pendingRemove.Size()
	totalSize = uint64(float64(totalSize) * 1.5)
	return totalSize > c.maxSize
}

//committx自动添加所有挂起的密钥以添加和删除到
//数据库缓存。当添加挂起的密钥时，将导致
//缓存超过最大缓存大小，或上次刷新后的时间超过
//配置的刷新间隔，缓存将刷新到底层
//持久数据库。
//
//这是一个关于缓存的原子操作，其中
//要在事务中添加和删除的挂起密钥将被应用或不应用
//他们会的。
//
//数据库缓存本身可能被刷新到基础持久性
//数据库，即使事务无法应用，但它将仅是
//未应用事务的缓存状态。
//
//必须在数据库写入事务期间调用此函数，其中
//turn表示将保留数据库写锁。
func (c *dbCache) commitTx(tx *transaction) error {
//刷新缓存并将当前事务直接写入
//数据库（如果需要刷新）。
	if c.needsFlush(tx) {
		if err := c.flush(); err != nil {
			return err
		}

//使用原子事务执行所有级别的数据库更新。
		err := c.commitTreaps(tx.pendingKeys, tx.pendingRemove)
		if err != nil {
			return err
		}

//清除提交后的事务条目。
		tx.pendingKeys = nil
		tx.pendingRemove = nil
		return nil
	}

//此时不需要数据库刷新，因此自动提交
//到缓存的事务。

//由于要添加和删除的缓存键使用不可变的treap，
//快照只是获取锁下树的根目录。
//用于原子交换根。
	c.cacheLock.RLock()
	newCachedKeys := c.cachedKeys
	newCachedRemove := c.cachedRemove
	c.cacheLock.RUnlock()

//将数据库事务中要添加的每个键应用到缓存。
	tx.pendingKeys.ForEach(func(k, v []byte) bool {
		newCachedRemove = newCachedRemove.Delete(k)
		newCachedKeys = newCachedKeys.Put(k, v)
		return true
	})
	tx.pendingKeys = nil

//将数据库事务中要删除的每个键应用到缓存。
	tx.pendingRemove.ForEach(func(k, v []byte) bool {
		newCachedKeys = newCachedKeys.Delete(k)
		newCachedRemove = newCachedRemove.Put(k, nil)
		return true
	})
	tx.pendingRemove = nil

//原子化地替换将缓存密钥保存到
//添加和删除。
	c.cacheLock.Lock()
	c.cachedKeys = newCachedKeys
	c.cachedRemove = newCachedRemove
	c.cacheLock.Unlock()
	return nil
}

//CLOSE通过同步所有数据并关闭来完全关闭数据库缓存
//基础级别数据库。
//
//必须在保持数据库写锁的情况下调用此函数。
func (c *dbCache) Close() error {
//将所有未完成的缓存项刷新到磁盘。
	if err := c.flush(); err != nil {
//即使刷新时出现错误，也要尝试关闭
//基础数据库。错误被忽略，因为它会
//屏蔽刷新错误。
		_ = c.ldb.Close()
		return err
	}

//关闭基础级别数据库。
	if err := c.ldb.Close(); err != nil {
		str := "failed to close underlying leveldb database"
		return convertErr(str, err)
	}

	return nil
}

//NeXbCache返回一个新的数据库缓存实例，该实例由提供的
//LevelDB实例。当最大大小为
//超过所提供的值或比提供的间隔长。
//从上次冲洗开始。
func newDbCache(ldb *leveldb.DB, store *blockStore, maxSize uint64, flushIntervalSecs uint32) *dbCache {
	return &dbCache{
		ldb:           ldb,
		store:         store,
		maxSize:       maxSize,
		flushInterval: time.Second * time.Duration(flushIntervalSecs),
		lastFlush:     time.Now(),
		cachedKeys:    treap.NewImmutable(),
		cachedRemove:  treap.NewImmutable(),
	}
}
