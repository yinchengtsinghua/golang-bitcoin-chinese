
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
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcd/database/internal/treap"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/goleveldb/leveldb"
	"github.com/btcsuite/goleveldb/leveldb/comparer"
	ldberrors "github.com/btcsuite/goleveldb/leveldb/errors"
	"github.com/btcsuite/goleveldb/leveldb/filter"
	"github.com/btcsuite/goleveldb/leveldb/iterator"
	"github.com/btcsuite/goleveldb/leveldb/opt"
	"github.com/btcsuite/goleveldb/leveldb/util"
)

const (
//metadatadbname是用于元数据数据库的名称。
	metadataDbName = "metadata"

//blockhdrsize是块头的大小。这只是
//从电线上取常数，仅在此处提供方便，因为
//Wire.MaxBlockHeaderPayLoad相当长。
	blockHdrSize = wire.MaxBlockHeaderPayload

//blockhdroffset将偏移量定义为
//块标题。
//
//序列化块索引行格式为：
//<blocklocation><blockheader>
	blockHdrOffset = blockLocSize
)

var (
//byte order是通过数据库和
//阻止文件。有时使用big endian来允许有序字节
//可排序整数值。
	byteOrder = binary.LittleEndian

//bucketindexprefix是用于bucket中所有条目的前缀
//索引。
	bucketIndexPrefix = []byte("bidx")

//CurbucketIdKeyName是用于跟踪
//当前Bucket ID计数器。
	curBucketIDKeyName = []byte("bidx-cbid")

//metadata bucket id是顶级元数据桶的ID。
//它是编码为无符号大尾数uint32的值0。
	metadataBucketID = [4]byte{}

//blockIDxBucketid是内部块元数据桶的ID。
//它是被编码为无符号大尾数uint32的值1。
	blockIdxBucketID = [4]byte{0x00, 0x00, 0x00, 0x01}

//blockIDXbucketname是内部用于跟踪块的bucket
//元数据。
	blockIdxBucketName = []byte("ffldb-blockidx")

//WriteLockeyName是用于存储当前写入文件的密钥
//位置。
	writeLocKeyName = []byte("ffldb-writeloc")
)

//常见错误字符串。
const (
//errdbnotopenstr是用于数据库的文本。errdbnotopen
//错误代码。
	errDbNotOpenStr = "database is not open"

//errtxclosedstr是用于数据库的文本。errtxclosedstr错误
//代码。
	errTxClosedStr = "database tx is closed"
)

//BulkFetchData允许指定块位置以及
//请求索引。这反过来又允许批量数据加载
//根据要改进的位置对数据访问进行排序的函数
//同时跟踪数据所针对的结果。
type bulkFetchData struct {
	*blockLocation
	replyIndex int
}

//BulkFetchDataSorter实现Sort.Interface以允许
//要排序的BulkFetchData。尤其是它按文件排序，然后
//偏移，以便对文件进行分组和线性读取。
type bulkFetchDataSorter []bulkFetchData

//len返回切片中的项数。它是
//Sort.Interface实现。
func (s bulkFetchDataSorter) Len() int {
	return len(s)
}

//交换按通过的指数交换项目。它是
//Sort.Interface实现。
func (s bulkFetchDataSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

//less返回具有索引的项是否应在具有索引的项之前排序
//索引j。它是sort.interface实现的一部分。
func (s bulkFetchDataSorter) Less(i, j int) bool {
	if s[i].blockFileNum < s[j].blockFileNum {
		return true
	}
	if s[i].blockFileNum > s[j].blockFileNum {
		return false
	}

	return s[i].fileOffset < s[j].fileOffset
}

//makedberr创建数据库。给定一组参数时出错。
func makeDbErr(c database.ErrorCode, desc string, err error) database.Error {
	return database.Error{ErrorCode: c, Description: desc, Err: err}
}

//converterr将传递的leveldb错误转换为数据库错误，其中
//等效错误代码和传递的描述。它还设置通过
//作为基础错误的错误。
func convertErr(desc string, ldbErr error) database.Error {
//默认情况下使用驱动程序特定的错误代码。下面的代码将
//如果识别出已转换的错误，则使用该错误进行更新。
	var code = database.ErrDriverSpecific

	switch {
//数据库损坏错误。
	case ldberrors.IsCorrupted(ldbErr):
		code = database.ErrCorruption

//数据库打开/创建错误。
	case ldbErr == leveldb.ErrClosed:
		code = database.ErrDbNotOpen

//事务错误。
	case ldbErr == leveldb.ErrSnapshotReleased:
		code = database.ErrTxClosed
	case ldbErr == leveldb.ErrIterReleased:
		code = database.ErrTxClosed
	}

	return database.Error{ErrorCode: code, Description: desc, Err: ldbErr}
}

//CopySicle返回已传递切片的副本。这主要是用来复制的
//LEVELDB迭代器键和值，因为它们只在迭代器之前有效
//而不是在整个事务期间移动。
func copySlice(slice []byte) []byte {
	ret := make([]byte, len(slice))
	copy(ret, slice)
	return ret
}

//光标是一种内部类型，用于在键/值对上表示光标。
//以及一个bucket的嵌套bucket，实现了database.cursor接口。
type cursor struct {
	bucket      *bucket
	dbIter      iterator.Iterator
	pendingIter iterator.Iterator
	currentIter iterator.Iterator
}

//强制游标实现database.cursor接口。
var _ database.Cursor = (*cursor)(nil)

//bucket返回为其创建光标的bucket。
//
//此函数是database.cursor接口实现的一部分。
func (c *cursor) Bucket() database.Bucket {
//确保事务状态有效。
	if err := c.bucket.tx.checkClosed(); err != nil {
		return nil
	}

	return c.bucket
}

//删除删除光标所在的当前键/值对
//使光标无效。
//
//根据接口约定返回以下错误：
//-如果在光标指向嵌套的
//水桶
//-errtxnotwritable（如果尝试对只读事务执行此操作）
//-如果事务已关闭，则返回errtxclosed
//
//此函数是database.cursor接口实现的一部分。
func (c *cursor) Delete() error {
//确保事务状态有效。
	if err := c.bucket.tx.checkClosed(); err != nil {
		return err
	}

//如果光标用完，则出错。
	if c.currentIter == nil {
		str := "cursor is exhausted"
		return makeDbErr(database.ErrIncompatibleValue, str, nil)
	}

//不允许通过光标删除存储桶。
	key := c.currentIter.Key()
	if bytes.HasPrefix(key, bucketIndexPrefix) {
		str := "buckets may not be deleted from a cursor"
		return makeDbErr(database.ErrIncompatibleValue, str, nil)
	}

	c.bucket.tx.deleteKey(copySlice(key), true)
	return nil
}

//SkipEndingUpdates跳过当前数据库迭代器位置的任何键
//正在由事务更新的。Forwards标志表示
//光标移动的方向。
func (c *cursor) skipPendingUpdates(forwards bool) {
	for c.dbIter.Valid() {
		var skip bool
		key := c.dbIter.Key()
		if c.bucket.tx.pendingRemove.Has(key) {
			skip = true
		} else if c.bucket.tx.pendingKeys.Has(key) {
			skip = true
		}
		if !skip {
			break
		}

		if forwards {
			c.dbIter.Next()
		} else {
			c.dbIter.Prev()
		}
	}
}

//ChooseIterator首先跳过数据库迭代器中
//被事务更新并将当前迭代器设置为
//适当的迭代器取决于它们的有效性和它们比较的顺序
//同时考虑了方向标志。当光标
//向前移动且两个迭代器都有效，具有较小值的迭代器
//当光标向后移动时，选择键，反之亦然。
func (c *cursor) chooseIterator(forwards bool) bool {
//跳过当前数据库迭代器位置的任何键
//正在由事务更新。
	c.skipPendingUpdates(forwards)

//当两个迭代器都用完时，光标也会用完。
	if !c.dbIter.Valid() && !c.pendingIter.Valid() {
		c.currentIter = nil
		return false
	}

//当挂起的键迭代器为
//筋疲力尽的。
	if !c.pendingIter.Valid() {
		c.currentIter = c.dbIter
		return true
	}

//当数据库迭代器为
//筋疲力尽的。
	if !c.dbIter.Valid() {
		c.currentIter = c.pendingIter
		return true
	}

//Both iterators are valid, so choose the iterator with either the
//较小或较大的键，取决于前进标志。
	compare := bytes.Compare(c.dbIter.Key(), c.pendingIter.Key())
	if (forwards && compare > 0) || (!forwards && compare < 0) {
		c.currentIter = c.pendingIter
	} else {
		c.currentIter = c.dbIter
	}
	return true
}

//首先将光标定位在第一个键/值对上，并返回
//这对不存在。
//
//此函数是database.cursor接口实现的一部分。
func (c *cursor) First() bool {
//确保事务状态有效。
	if err := c.bucket.tx.checkClosed(); err != nil {
		return false
	}

//在数据库和挂起的迭代器中查找第一个键
//选择既有效又具有较小键的迭代器。
	c.dbIter.First()
	c.pendingIter.First()
	return c.chooseIterator(true)
}

//Last将光标定位在最后一个键/值对上，并返回
//这对不存在。
//
//此函数是database.cursor接口实现的一部分。
func (c *cursor) Last() bool {
//确保事务状态有效。
	if err := c.bucket.tx.checkClosed(); err != nil {
		return false
	}

//查找数据库和挂起迭代器中的最后一个键
//选择既有效又具有较大键的迭代器。
	c.dbIter.Last()
	c.pendingIter.Last()
	return c.chooseIterator(false)
}

//下一步将光标向前移动一个键/值对，并返回是否
//这对存在。
//
//此函数是database.cursor接口实现的一部分。
func (c *cursor) Next() bool {
//确保事务状态有效。
	if err := c.bucket.tx.checkClosed(); err != nil {
		return false
	}

//如果光标用完，则不返回任何内容。
	if c.currentIter == nil {
		return false
	}

//将当前迭代器移动到下一个条目并选择迭代器
//它既有效又有较小的密钥。
	c.currentIter.Next()
	return c.chooseIterator(true)
}

//prev将光标向后移动一个键/值对，并返回是否
//这对存在。
//
//此函数是database.cursor接口实现的一部分。
func (c *cursor) Prev() bool {
//确保事务状态有效。
	if err := c.bucket.tx.checkClosed(); err != nil {
		return false
	}

//如果光标用完，则不返回任何内容。
	if c.currentIter == nil {
		return false
	}

//将当前迭代器移动到上一个条目并选择
//同时有效且具有较大键的迭代器。
	c.currentIter.Prev()
	return c.chooseIterator(false)
}

//SEEK将光标定位在大于或的第一个键/值对上。
//等于传递的seek键。如果找不到合适的密钥，则返回false。
//
//此函数是database.cursor接口实现的一部分。
func (c *cursor) Seek(seek []byte) bool {
//确保事务状态有效。
	if err := c.bucket.tx.checkClosed(); err != nil {
		return false
	}

//在数据库和挂起的迭代器中查找提供的键
//然后选择既有效又具有较大键的迭代器。
	seekKey := bucketizedKey(c.bucket.id, seek)
	c.dbIter.Seek(seekKey)
	c.pendingIter.Seek(seekKey)
	return c.chooseIterator(true)
}

//rawkey返回光标指向的当前键，而不剥离
//当前的bucket前缀或bucket索引前缀。
func (c *cursor) rawKey() []byte {
//如果光标用完，则不返回任何内容。
	if c.currentIter == nil {
		return nil
	}

	return copySlice(c.currentIter.Key())
}

//键返回光标指向的当前键。
//
//此函数是database.cursor接口实现的一部分。
func (c *cursor) Key() []byte {
//确保事务状态有效。
	if err := c.bucket.tx.checkClosed(); err != nil {
		return nil
	}

//如果光标用完，则不返回任何内容。
	if c.currentIter == nil {
		return nil
	}

//切掉实际的键名并复制，因为它不再是
//迭代到下一项后有效。
//
//当
//光标指向嵌套的bucket。
	key := c.currentIter.Key()
	if bytes.HasPrefix(key, bucketIndexPrefix) {
		key = key[len(bucketIndexPrefix)+4:]
		return copySlice(key)
	}

//当光标指向一个
//正常进入。
	key = key[len(c.bucket.id):]
	return copySlice(key)
}

//rawvalue返回光标指向的当前值
//在不过滤存储桶索引值的情况下进行剥离。
func (c *cursor) rawValue() []byte {
//如果光标用完，则不返回任何内容。
	if c.currentIter == nil {
		return nil
	}

	return copySlice(c.currentIter.Value())
}

//值返回光标指向的当前值。这是零
//用于嵌套存储桶。
//
//此函数是database.cursor接口实现的一部分。
func (c *cursor) Value() []byte {
//确保事务状态有效。
	if err := c.bucket.tx.checkClosed(); err != nil {
		return nil
	}

//如果光标用完，则不返回任何内容。
	if c.currentIter == nil {
		return nil
	}

//当光标指向嵌套的
//桶。
	if bytes.HasPrefix(c.currentIter.Key(), bucketIndexPrefix) {
		return nil
	}

	return copySlice(c.currentIter.Value())
}

//CursorType定义要创建的光标类型。
type cursorType int

//以下常量定义了允许的光标类型。
const (
//ctkeys迭代给定bucket中的所有键。
	ctKeys cursorType = iota

//CTBuckets在给定的
//桶。
	ctBuckets

//ctfull同时迭代键和直接嵌套的bucket
//在给定的桶中。
	ctFull
)

//当对光标进行垃圾收集时调用CursorFinalizer，或者
//手动调用以确保释放底层的游标迭代器。
func cursorFinalizer(c *cursor) {
	c.dbIter.Release()
	c.pendingIter.Release()
}

//new cursor返回给定bucket、bucket id和cursor的新光标
//类型。
//
//注意：调用方负责调用Currror终结器函数。
//返回的光标。
func newCursor(b *bucket, bucketID []byte, cursorTyp cursorType) *cursor {
	var dbIter, pendingIter iterator.Iterator
	switch cursorTyp {
	case ctKeys:
		keyRange := util.BytesPrefix(bucketID)
		dbIter = b.tx.snapshot.NewIterator(keyRange)
		pendingKeyIter := newLdbTreapIter(b.tx, keyRange)
		pendingIter = pendingKeyIter

	case ctBuckets:
//序列化的bucket索引键格式为：
//<bucketindexprefix><parentbucketid><bucketname>

//为数据库和挂起的创建迭代器
//以bucket索引标识符和
//提供的存储桶ID。
		prefix := make([]byte, len(bucketIndexPrefix)+4)
		copy(prefix, bucketIndexPrefix)
		copy(prefix[len(bucketIndexPrefix):], bucketID)
		bucketRange := util.BytesPrefix(prefix)

		dbIter = b.tx.snapshot.NewIterator(bucketRange)
		pendingBucketIter := newLdbTreapIter(b.tx, bucketRange)
		pendingIter = pendingBucketIter

	case ctFull:
		fallthrough
	default:
//序列化的bucket索引键格式为：
//<bucketindexprefix><parentbucketid><bucketname>
		prefix := make([]byte, len(bucketIndexPrefix)+4)
		copy(prefix, bucketIndexPrefix)
		copy(prefix[len(bucketIndexPrefix):], bucketID)
		bucketRange := util.BytesPrefix(prefix)
		keyRange := util.BytesPrefix(bucketID)

//因为数据库中同时需要键和存储桶，
//为每个前缀创建一个单独的迭代器，然后创建
//来自它们的合并迭代器。
		dbKeyIter := b.tx.snapshot.NewIterator(keyRange)
		dbBucketIter := b.tx.snapshot.NewIterator(bucketRange)
		iters := []iterator.Iterator{dbKeyIter, dbBucketIter}
		dbIter = iterator.NewMergedIterator(iters,
			comparer.DefaultComparer, true)

//因为挂起的键需要键和存储桶，
//为每个前缀创建一个单独的迭代器，然后创建
//来自它们的合并迭代器。
		pendingKeyIter := newLdbTreapIter(b.tx, keyRange)
		pendingBucketIter := newLdbTreapIter(b.tx, bucketRange)
		iters = []iterator.Iterator{pendingKeyIter, pendingBucketIter}
		pendingIter = iterator.NewMergedIterator(iters,
			comparer.DefaultComparer, true)
	}

//使用迭代器创建光标。
	return &cursor{bucket: b, dbIter: dbIter, pendingIter: pendingIter}
}

//bucket是一种内部类型，用于表示键/值对的集合
//并实现了database.bucket接口。
type bucket struct {
	tx *transaction
	id [4]byte
}

//强制bucket实现database.bucket接口。
var _ database.Bucket = (*bucket)(nil)

//bucketindexkey返回用于存储和检索
//存储桶索引中的子存储桶。这是必需的，因为
//需要信息来区分具有相同名称的嵌套存储桶。
func bucketIndexKey(parentID [4]byte, key []byte) []byte {
//序列化的bucket索引键格式为：
//<bucketindexprefix><parentbucketid><bucketname>
	indexKey := make([]byte, len(bucketIndexPrefix)+4+len(key))
	copy(indexKey, bucketIndexPrefix)
	copy(indexKey[len(bucketIndexPrefix):], parentID[:])
	copy(indexKey[len(bucketIndexPrefix)+4:], key)
	return indexKey
}

//BucketizedKey返回用于存储和检索密钥的实际密钥
//对于提供的bucket id。这是必需的，因为bucketizing是处理的
//通过对每个桶使用唯一的前缀。
func bucketizedKey(bucketID [4]byte, key []byte) []byte {
//序列化块索引键格式为：
//<密钥> >
	bKey := make([]byte, 4+len(key))
	copy(bKey, bucketID[:])
	copy(bKey[4:], key)
	return bKey
}

//bucket用给定的键检索嵌套的bucket。返回零IF
//桶不存在。
//
//此函数是database.bucket接口实现的一部分。
func (b *bucket) Bucket(key []byte) database.Bucket {
//确保事务状态有效。
	if err := b.tx.checkClosed(); err != nil {
		return nil
	}

//尝试获取子存储桶的ID。水桶没有
//如果bucket索引项不存在，则存在。
	childID := b.tx.fetchKey(bucketIndexKey(b.id, key))
	if childID == nil {
		return nil
	}

	childBucket := &bucket{tx: b.tx}
	copy(childBucket.id[:], childID)
	return childBucket
}

//createBucket创建并返回具有给定键的新嵌套bucket。
//
//根据接口约定返回以下错误：
//-errbacketexists如果存储桶已经存在
//-如果密钥为空，则需要errbacketname
//-errcompatibleValue，如果该键对特定项无效
//实施
//-errtxnotwritable（如果尝试对只读事务执行此操作）
//-如果事务已关闭，则返回errtxclosed
//
//此函数是database.bucket接口实现的一部分。
func (b *bucket) CreateBucket(key []byte) (database.Bucket, error) {
//确保事务状态有效。
	if err := b.tx.checkClosed(); err != nil {
		return nil, err
	}

//确保事务是可写的。
	if !b.tx.writable {
		str := "create bucket requires a writable database transaction"
		return nil, makeDbErr(database.ErrTxNotWritable, str, nil)
	}

//确保提供了密钥。
	if len(key) == 0 {
		str := "create bucket requires a key"
		return nil, makeDbErr(database.ErrBucketNameRequired, str, nil)
	}

//确保bucket不存在。
	bidxKey := bucketIndexKey(b.id, key)
	if b.tx.hasKey(bidxKey) {
		str := "bucket already exists"
		return nil, makeDbErr(database.ErrBucketExists, str, nil)
	}

//找到新bucket要使用的下一个bucket id。在
//特殊的内部块索引的情况下，保持固定的ID。
	var childID [4]byte
	if b.id == metadataBucketID && bytes.Equal(key, blockIdxBucketName) {
		childID = blockIdxBucketID
	} else {
		var err error
		childID, err = b.tx.nextBucketID()
		if err != nil {
			return nil, err
		}
	}

//将新bucket添加到bucket索引。
	if err := b.tx.putKey(bidxKey, childID[:]); err != nil {
		str := fmt.Sprintf("failed to create bucket with key %q", key)
		return nil, convertErr(str, err)
	}
	return &bucket{tx: b.tx, id: childID}, nil
}

//CreateBacketifnotexists创建并返回一个新的嵌套bucket，其中
//给定的键（如果它不存在）。
//
//根据接口约定返回以下错误：
//-如果密钥为空，则需要errbacketname
//-errcompatibleValue，如果该键对特定项无效
//实施
//-errtxnotwritable（如果尝试对只读事务执行此操作）
//-如果事务已关闭，则返回errtxclosed
//
//此函数是database.bucket接口实现的一部分。
func (b *bucket) CreateBucketIfNotExists(key []byte) (database.Bucket, error) {
//确保事务状态有效。
	if err := b.tx.checkClosed(); err != nil {
		return nil, err
	}

//确保事务是可写的。
	if !b.tx.writable {
		str := "create bucket requires a writable database transaction"
		return nil, makeDbErr(database.ErrTxNotWritable, str, nil)
	}

//如果已经存在，则返回现有的bucket，否则创建它。
	if bucket := b.Bucket(key); bucket != nil {
		return bucket, nil
	}
	return b.CreateBucket(key)
}

//删除bucket删除具有给定键的嵌套bucket。
//
//根据接口约定返回以下错误：
//-errbacketnotfound如果指定的存储桶不存在
//-errtxnotwritable（如果尝试对只读事务执行此操作）
//-如果事务已关闭，则返回errtxclosed
//
//此函数是database.bucket接口实现的一部分。
func (b *bucket) DeleteBucket(key []byte) error {
//确保事务状态有效。
	if err := b.tx.checkClosed(); err != nil {
		return err
	}

//确保事务是可写的。
	if !b.tx.writable {
		str := "delete bucket requires a writable database transaction"
		return makeDbErr(database.ErrTxNotWritable, str, nil)
	}

//尝试获取子存储桶的ID。水桶没有
//如果bucket索引项不存在，则存在。在这种情况下
//特殊的内部块索引，保持固定的ID。
	bidxKey := bucketIndexKey(b.id, key)
	childID := b.tx.fetchKey(bidxKey)
	if childID == nil {
		str := fmt.Sprintf("bucket %q does not exist", key)
		return makeDbErr(database.ErrBucketNotFound, str, nil)
	}

//移除所有嵌套存储桶及其键。
	childIDs := [][]byte{childID}
	for len(childIDs) > 0 {
		childID = childIDs[len(childIDs)-1]
		childIDs = childIDs[:len(childIDs)-1]

//删除嵌套存储桶中的所有键。
		keyCursor := newCursor(b, childID, ctKeys)
		for ok := keyCursor.First(); ok; ok = keyCursor.Next() {
			b.tx.deleteKey(keyCursor.rawKey(), false)
		}
		cursorFinalizer(keyCursor)

//遍历所有嵌套存储桶。
		bucketCursor := newCursor(b, childID, ctBuckets)
		for ok := bucketCursor.First(); ok; ok = bucketCursor.Next() {
//将嵌套存储桶的ID推送到堆栈上
//下一次迭代。
			childID := bucketCursor.rawValue()
			childIDs = append(childIDs, childID)

//从bucket索引中移除嵌套bucket。
			b.tx.deleteKey(bucketCursor.rawKey(), false)
		}
		cursorFinalizer(bucketCursor)
	}

//从bucket索引中移除嵌套bucket。任何嵌套的bucket
//它下面已经被移走了。
	b.tx.deleteKey(bidxKey, true)
	return nil
}

//cursor返回一个新的光标，允许在bucket的
//键/值对和嵌套存储桶的前向或后向顺序。
//
//必须使用前面的First、Last或Seek函数查找位置
//调用next、prev、key或value函数。不这样做会
//导致返回值与用尽的光标相同，这对于
//prev和next函数，key和value函数为nil。
//
//此函数是database.bucket接口实现的一部分。
func (b *bucket) Cursor() database.Cursor {
//确保事务状态有效。
	if err := b.tx.checkClosed(); err != nil {
		return &cursor{bucket: b}
	}

//创建光标并设置运行时终结器以确保
//当光标被垃圾收集时，迭代器被释放。
	c := newCursor(b, b.id[:], ctFull)
	runtime.SetFinalizer(c, cursorFinalizer)
	return c
}

//foreach使用bucket中的每个键/值对调用传递的函数。
//这不包括嵌套存储桶或其中的键/值对
//嵌套桶。
//
//警告：使用此方法进行迭代时更改数据是不安全的。
//这样做可能导致基础光标无效并返回
//意外的键和/或值。
//
//根据接口约定返回以下错误：
//-如果事务已关闭，则返回errtxclosed
//
//注意：此函数返回的值仅在
//交易。在事务结束后尝试访问它们将
//可能导致访问冲突。
//
//此函数是database.bucket接口实现的一部分。
func (b *bucket) ForEach(fn func(k, v []byte) error) error {
//确保事务状态有效。
	if err := b.tx.checkClosed(); err != nil {
		return err
	}

//为每个光标项调用回调。返回返回的错误
//从非零的回调。
	c := newCursor(b, b.id[:], ctKeys)
	defer cursorFinalizer(c)
	for ok := c.First(); ok; ok = c.Next() {
		err := fn(c.Key(), c.Value())
		if err != nil {
			return err
		}
	}

	return nil
}

//foreachbucket使用每个嵌套bucket的键调用传递的函数
//在当前存储桶中。这不包括那些
//嵌套桶。
//
//警告：使用此方法进行迭代时更改数据是不安全的。
//这样做可能导致基础光标无效并返回
//意外的键。
//
//根据接口约定返回以下错误：
//-如果事务已关闭，则返回errtxclosed
//
//注意：此函数返回的值仅在
//交易。在事务结束后尝试访问它们将
//可能导致访问冲突。
//
//此函数是database.bucket接口实现的一部分。
func (b *bucket) ForEachBucket(fn func(k []byte) error) error {
//确保事务状态有效。
	if err := b.tx.checkClosed(); err != nil {
		return err
	}

//为每个光标项调用回调。返回返回的错误
//从非零的回调。
	c := newCursor(b, b.id[:], ctBuckets)
	defer cursorFinalizer(c)
	for ok := c.First(); ok; ok = c.Next() {
		err := fn(c.Key())
		if err != nil {
			return err
		}
	}

	return nil
}

//可写返回bucket是否可写。
//
//此函数是database.bucket接口实现的一部分。
func (b *bucket) Writable() bool {
	return b.tx.writable
}

//PUT将指定的键/值对保存到存储桶中。不需要的钥匙
//添加已存在的键，覆盖已存在的键。
//
//根据接口约定返回以下错误：
//-如果密钥为空，则需要errkey
//-errcompatibleValue（如果密钥与现有存储桶相同）
//-errtxnotwritable（如果尝试对只读事务执行此操作）
//-如果事务已关闭，则返回errtxclosed
//
//此函数是database.bucket接口实现的一部分。
func (b *bucket) Put(key, value []byte) error {
//确保事务状态有效。
	if err := b.tx.checkClosed(); err != nil {
		return err
	}

//确保事务是可写的。
	if !b.tx.writable {
		str := "setting a key requires a writable database transaction"
		return makeDbErr(database.ErrTxNotWritable, str, nil)
	}

//确保提供了密钥。
	if len(key) == 0 {
		str := "put requires a key"
		return makeDbErr(database.ErrKeyRequired, str, nil)
	}

	return b.tx.putKey(bucketizedKey(b.id, key), value)
}

//get返回给定键的值。如果键不匹配，则返回nil
//存在于此存储桶中。对于存在但
//没有赋值。
//
//注意：此函数返回的值仅在事务期间有效。
//在事务结束后尝试访问它会导致未定义
//行为。此外，调用方不能修改该值。
//
//此函数是database.bucket接口实现的一部分。
func (b *bucket) Get(key []byte) []byte {
//确保事务状态有效。
	if err := b.tx.checkClosed(); err != nil {
		return nil
	}

//如果没有钥匙，则无需返回。
	if len(key) == 0 {
		return nil
	}

	return b.tx.fetchKey(bucketizedKey(b.id, key))
}

//删除从存储桶中删除指定的键。删除一个键
//不存在不返回错误。
//
//根据接口约定返回以下错误：
//-如果密钥为空，则需要errkey
//-errcompatibleValue（如果密钥与现有存储桶相同）
//-errtxnotwritable（如果尝试对只读事务执行此操作）
//-如果事务已关闭，则返回errtxclosed
//
//此函数是database.bucket接口实现的一部分。
func (b *bucket) Delete(key []byte) error {
//确保事务状态有效。
	if err := b.tx.checkClosed(); err != nil {
		return err
	}

//确保事务是可写的。
	if !b.tx.writable {
		str := "deleting a value requires a writable database transaction"
		return makeDbErr(database.ErrTxNotWritable, str, nil)
	}

//如果没有钥匙，什么都不做。
	if len(key) == 0 {
		return nil
	}

	b.tx.deleteKey(bucketizedKey(b.id, key), true)
	return nil
}

//PendingBlock包含一个块，当数据库
//事务已提交。
type pendingBlock struct {
	hash  *chainhash.Hash
	bytes []byte
}

//事务表示数据库事务。它可以是只读的，也可以是
//读写并实现database.bucket接口。交易
//提供一个根存储桶，所有读取和写入都针对它进行。
type transaction struct {
managed        bool             //交易是否管理？
closed         bool             //交易记录是否已结束？
writable       bool             //事务是否可写？
db             *db              //从中创建Tx的数据库实例。
snapshot       *dbCacheSnapshot //TXN的基础快照。
metaBucket     *bucket          //根元数据存储桶。
blockIdxBucket *bucket          //块索引桶。

//需要在提交时存储的块。悬垂的方块图是
//保持允许按块哈希快速查找挂起的数据。
	pendingBlocks    map[chainhash.Hash]int
	pendingBlockData []pendingBlock

//提交时需要存储或删除的键。
	pendingKeys   *treap.Mutable
	pendingRemove *treap.Mutable

//当挂起的键
//已更新，以便光标可以正确处理
//事务状态。
	activeIterLock sync.RWMutex
	activeIters    []*treap.Iterator
}

//强制事务实现database.tx接口。
var _ database.Tx = (*transaction)(nil)

//removeactiveiter从活动的列表中移除传递的迭代器
//针对挂起的键treap的迭代器。
func (tx *transaction) removeActiveIter(iter *treap.Iterator) {
//循环的索引有意在此处的某个范围内用作范围
//不会在每次迭代中重新评估切片，也不会调整切片
//修改切片的索引。
	tx.activeIterLock.Lock()
	for i := 0; i < len(tx.activeIters); i++ {
		if tx.activeIters[i] == iter {
			copy(tx.activeIters[i:], tx.activeIters[i+1:])
			tx.activeIters[len(tx.activeIters)-1] = nil
			tx.activeIters = tx.activeIters[:len(tx.activeIters)-1]
		}
	}
	tx.activeIterLock.Unlock()
}

//addactiveiter将传递的迭代器添加到的活动迭代器列表中
//挂起的键treap。
func (tx *transaction) addActiveIter(iter *treap.Iterator) {
	tx.activeIterLock.Lock()
	tx.activeIters = append(tx.activeIters, iter)
	tx.activeIterLock.Unlock()
}

//notifyactiveiters通知挂起密钥的所有活动迭代器
//t确认已更新。
func (tx *transaction) notifyActiveIters() {
	tx.activeIterLock.RLock()
	for _, iter := range tx.activeIters {
		iter.ForceReseek()
	}
	tx.activeIterLock.RUnlock()
}

//如果数据库或事务已关闭，则checkclosed返回错误。
func (tx *transaction) checkClosed() error {
//如果事务已关闭，则该事务将不再有效。
	if tx.closed {
		return makeDbErr(database.ErrTxClosed, errTxClosedStr, nil)
	}

	return nil
}

//haskey返回所提供的键在数据库中是否存在，而
//考虑到当前交易状态。
func (tx *transaction) hasKey(key []byte) bool {
//当事务可写时，检查挂起的事务
//先声明状态。
	if tx.writable {
		if tx.pendingRemove.Has(key) {
			return false
		}
		if tx.pendingKeys.Has(key) {
			return true
		}
	}

//请参阅数据库缓存和基础数据库。
	return tx.snapshot.Has(key)
}

//Putkey将提供的密钥添加到要在
//提交事务时的数据库。
//
//注意：只能在可写事务上调用此函数。因为它
//是一个内部助手函数，它不检查。
func (tx *transaction) putKey(key, value []byte) error {
//阻止删除以前计划的密钥
//在事务提交时删除。
	tx.pendingRemove.Delete(key)

//将密钥/值对添加到要写入事务的列表中
//承诺。
	tx.pendingKeys.Put(key, value)
	tx.notifyActiveIters()
	return nil
}

//fetch key尝试从数据库缓存中获取提供的密钥（和
//因此，在考虑当前事务的基础数据库中
//状态。如果键不存在，则返回nil。
func (tx *transaction) fetchKey(key []byte) []byte {
//当事务可写时，检查挂起的事务
//先声明状态。
	if tx.writable {
		if tx.pendingRemove.Has(key) {
			return nil
		}
		if value := tx.pendingKeys.Get(key); value != nil {
			return value
		}
	}

//请参阅数据库缓存和基础数据库。
	return tx.snapshot.Get(key)
}

//DeleteKey将提供的密钥添加到要从中删除的密钥列表
//提交事务时的数据库。通知迭代器标志是
//用于延迟通知迭代器批量删除期间的更改。
//
//注意：只能在可写事务上调用此函数。因为它
//是一个内部助手函数，它不检查。
func (tx *transaction) deleteKey(key []byte, notifyIterators bool) {
//从要写入的挂起密钥列表中删除密钥
//事务提交（如果需要）。
	tx.pendingKeys.Delete(key)

//将密钥添加到事务提交时要删除的列表中。
	tx.pendingRemove.Put(key, nil)

//如果设置了标志，则通知活动迭代器更改。
	if notifyIterators {
		tx.notifyActiveIters()
	}
}

//next bucket id返回用于创建新bucket的下一个bucket id。
//
//注意：只能在可写事务上调用此函数。因为它
//是一个内部助手函数，它不检查。
func (tx *transaction) nextBucketID() ([4]byte, error) {
//加载当前使用的最高bucket id。
	curIDBytes := tx.fetchKey(curBucketIDKeyName)
	curBucketNum := binary.BigEndian.Uint32(curIDBytes)

//递增并更新当前bucket id并返回。
	var nextBucketID [4]byte
	binary.BigEndian.PutUint32(nextBucketID[:], curBucketNum+1)
	if err := tx.putKey(curBucketIDKeyName, nextBucketID[:]); err != nil {
		return [4]byte{}, err
	}
	return nextBucketID, nil
}

//元数据返回所有元数据存储的最高存储桶。
//
//此函数是database.tx接口实现的一部分。
func (tx *transaction) Metadata() database.Bucket {
	return tx.metaBucket
}

//HasBlock返回具有给定哈希的块是否存在。
func (tx *transaction) hasBlock(hash *chainhash.Hash) bool {
//如果块在提交时等待写入，则返回true，因为
//从这个事务的角度来看，它是存在的。
	if _, exists := tx.pendingBlocks[*hash]; exists {
		return true
	}

	return tx.hasKey(bucketizedKey(blockIdxBucketID, hash[:]))
}

//storeblock将提供的块存储到数据库中。没有支票
//要确保块连接到上一个块，包含双倍开销，或
//任何附加功能，如事务索引。它只是储存
//数据库中的块。
//
//根据接口约定返回以下错误：
//-当块哈希已存在时，errblockexists
//-errtxnotwritable（如果尝试对只读事务执行此操作）
//-如果事务已关闭，则返回errtxclosed
//
//此函数是database.tx接口实现的一部分。
func (tx *transaction) StoreBlock(block *btcutil.Block) error {
//确保事务状态有效。
	if err := tx.checkClosed(); err != nil {
		return err
	}

//确保事务是可写的。
	if !tx.writable {
		str := "store block requires a writable database transaction"
		return makeDbErr(database.ErrTxNotWritable, str, nil)
	}

//如果块已经存在，则拒绝它。
	blockHash := block.Hash()
	if tx.hasBlock(blockHash) {
		str := fmt.Sprintf("block %s already exists", blockHash)
		return makeDbErr(database.ErrBlockExists, str, nil)
	}

	blockBytes, err := block.Bytes()
	if err != nil {
		str := fmt.Sprintf("failed to get serialized bytes for block %s",
			blockHash)
		return makeDbErr(database.ErrDriverSpecific, str, err)
	}

//将要存储的块添加到要存储的挂起块列表中
//提交事务时。另外，将其添加到挂起的块中
//映射，以便根据
//块哈希。
	if tx.pendingBlocks == nil {
		tx.pendingBlocks = make(map[chainhash.Hash]int)
	}
	tx.pendingBlocks[*blockHash] = len(tx.pendingBlockData)
	tx.pendingBlockData = append(tx.pendingBlockData, pendingBlock{
		hash:  blockHash,
		bytes: blockBytes,
	})
	log.Tracef("Added block %s to pending blocks", blockHash)

	return nil
}

//HasBlock返回具有给定哈希的块是否存在于
//数据库。
//
//根据接口约定返回以下错误：
//-如果事务已关闭，则返回errtxclosed
//
//此函数是database.tx接口实现的一部分。
func (tx *transaction) HasBlock(hash *chainhash.Hash) (bool, error) {
//确保事务状态有效。
	if err := tx.checkClosed(); err != nil {
		return false, err
	}

	return tx.hasBlock(hash), nil
}

//HasBlocks返回具有提供的哈希值的块
//存在于数据库中。
//
//根据接口约定返回以下错误：
//-如果事务已关闭，则返回errtxclosed
//
//此函数是database.tx接口实现的一部分。
func (tx *transaction) HasBlocks(hashes []chainhash.Hash) ([]bool, error) {
//确保事务状态有效。
	if err := tx.checkClosed(); err != nil {
		return nil, err
	}

	results := make([]bool, len(hashes))
	for i := range hashes {
		results[i] = tx.hasBlock(&hashes[i])
	}

	return results, nil
}

//fetchblockrow获取存储在所提供的块索引中的元数据
//搞砸。如果没有条目，它将返回errblocknotfound。
func (tx *transaction) fetchBlockRow(hash *chainhash.Hash) ([]byte, error) {
	blockRow := tx.blockIdxBucket.Get(hash[:])
	if blockRow == nil {
		str := fmt.Sprintf("block %s does not exist", hash)
		return nil, makeDbErr(database.ErrBlockNotFound, str, nil)
	}

	return blockRow, nil
}

//FetchBlockHeader返回块头的原始序列化字节
//由给定哈希标识。原始字节的格式为
//在Wire.BlockHeader上序列化。
//
//根据接口约定返回以下错误：
//-errblocknotfound如果请求的块哈希不存在
//-如果事务已关闭，则返回errtxclosed
//-errCorrupt如果数据库已损坏
//
//注意：此函数返回的数据仅在
//数据库事务。试图在事务后访问它
//已结束将导致未定义的行为。此约束可防止
//其他数据副本，并允许支持内存映射数据库
//实施。
//
//此函数是database.tx接口实现的一部分。
func (tx *transaction) FetchBlockHeader(hash *chainhash.Hash) ([]byte, error) {
	return tx.FetchBlockRegion(&database.BlockRegion{
		Hash:   hash,
		Offset: 0,
		Len:    blockHdrSize,
	})
}

//fetchblockheaders返回块头的原始序列化字节
//由给定哈希标识。原始字节的格式为
//在Wire.BlockHeader上序列化。
//
//根据接口约定返回以下错误：
//-errblocknotfound如果任何请求的块哈希不存在
//-如果事务已关闭，则返回errtxclosed
//-errCorrupt如果数据库已损坏
//
//注意：此函数返回的数据仅在数据库中有效
//交易。在事务结束后尝试访问它的结果
//在未定义的行为中。此约束防止额外的数据复制和
//允许支持内存映射数据库实现。
//
//此函数是database.tx接口实现的一部分。
func (tx *transaction) FetchBlockHeaders(hashes []chainhash.Hash) ([][]byte, error) {
	regions := make([]database.BlockRegion, len(hashes))
	for i := range hashes {
		regions[i].Hash = &hashes[i]
		regions[i].Offset = 0
		regions[i].Len = blockHdrSize
	}
	return tx.FetchBlockRegions(regions)
}

//fetchblock返回由
//给定哈希值。原始字节的格式是在
//MggBug。
//
//根据接口约定返回以下错误：
//-errblocknotfound如果请求的块哈希不存在
//-如果事务已关闭，则返回errtxclosed
//-errCorrupt如果数据库已损坏
//
//此外，如果在读取
//阻止文件。
//
//注意：此函数返回的数据仅在数据库中有效
//交易。在事务结束后尝试访问它的结果
//在未定义的行为中。此约束防止额外的数据复制和
//允许支持内存映射数据库实现。
//
//此函数是database.tx接口实现的一部分。
func (tx *transaction) FetchBlock(hash *chainhash.Hash) ([]byte, error) {
//确保事务状态有效。
	if err := tx.checkClosed(); err != nil {
		return nil, err
	}

//当块在提交时等待写入时，返回字节
//从那里。
	if idx, exists := tx.pendingBlocks[*hash]; exists {
		return tx.pendingBlockData[idx].bytes, nil
	}

//从块索引中查找文件中块的位置。
	blockRow, err := tx.fetchBlockRow(hash)
	if err != nil {
		return nil, err
	}
	location := deserializeBlockLoc(blockRow)

//从适当的位置读取块。功能也
//对数据执行校验和以检测数据损坏。
	blockBytes, err := tx.db.store.readBlock(hash, location)
	if err != nil {
		return nil, err
	}

	return blockBytes, nil
}

//FetchBlocks返回由
//给定散列。原始字节的格式是在
//MggBug。
//
//根据接口约定返回以下错误：
//-errblocknotfound如果任何请求的哈希块不存在
//-如果事务已关闭，则返回errtxclosed
//-errCorrupt如果数据库已损坏
//
//此外，如果在读取
//阻止文件。
//
//注意：此函数返回的数据仅在数据库中有效
//交易。在事务结束后尝试访问它的结果
//在未定义的行为中。此约束防止额外的数据复制和
//允许支持内存映射数据库实现。
//
//此函数是database.tx接口实现的一部分。
func (tx *transaction) FetchBlocks(hashes []chainhash.Hash) ([][]byte, error) {
//确保事务状态有效。
	if err := tx.checkClosed(); err != nil {
		return nil, err
	}

//注意：这可以在加载前检查是否存在所有块
//但是，在失败的情况下，它们中的任何一个都会更快
//调用方通常不会使用无效的
//值，因此针对常见情况进行优化。

//加载块。
	blocks := make([][]byte, len(hashes))
	for i := range hashes {
		var err error
		blocks[i], err = tx.FetchBlock(&hashes[i])
		if err != nil {
			return nil, err
		}
	}

	return blocks, nil
}

//fetchPendingRegion尝试从任何块获取提供的区域，该块
//等待在提交时写入。它将为字节片返回nil
//当区域引用未挂起的块时。当该地区
//是否引用挂起的块，它是检查边界并返回
//errblockregioninvalid如果无效。
func (tx *transaction) fetchPendingRegion(region *database.BlockRegion) ([]byte, error) {
//如果块在提交时没有等待写入，则不执行任何操作。
	idx, exists := tx.pendingBlocks[*region.Hash]
	if !exists {
		return nil, nil
	}

//确保区域在块的范围内。
	blockBytes := tx.pendingBlockData[idx].bytes
	blockLen := uint32(len(blockBytes))
	endOffset := region.Offset + region.Len
	if endOffset < region.Offset || endOffset > blockLen {
		str := fmt.Sprintf("block %s region offset %d, length %d "+
			"exceeds block length of %d", region.Hash,
			region.Offset, region.Len, blockLen)
		return nil, makeDbErr(database.ErrBlockRegionInvalid, str, nil)
	}

//返回挂起块中的字节。
	return blockBytes[region.Offset:endOffset:endOffset], nil
}

//FetchBlockRegion返回给定块区域的原始序列化字节。
//
//例如，可以直接提取比特币交易和/或
//来自具有此函数的块的脚本。取决于后端
//实现，这可以通过避免
//加载整个块。
//
//原始字节采用wire.msgblock上的serialize返回的格式，并且
//提供的块区域中的偏移字段基于零，并且相对于
//块的开头（字节0）。
//
//根据接口约定返回以下错误：
//-errblocknotfound如果请求的块哈希不存在
//-errblockregioninvalid如果区域超过了关联的
//块
//-如果事务已关闭，则返回errtxclosed
//-errCorrupt如果数据库已损坏
//
//此外，如果在读取
//阻止文件。
//
//注意：此函数返回的数据仅在数据库中有效
//交易。在事务结束后尝试访问它的结果
//在未定义的行为中。此约束防止额外的数据复制和
//允许支持内存映射数据库实现。
//
//此函数是database.tx接口实现的一部分。
func (tx *transaction) FetchBlockRegion(region *database.BlockRegion) ([]byte, error) {
//确保事务状态有效。
	if err := tx.checkClosed(); err != nil {
		return nil, err
	}

//当块在提交时等待写入时，返回字节
//从那里。
	if tx.pendingBlocks != nil {
		regionBytes, err := tx.fetchPendingRegion(region)
		if err != nil {
			return nil, err
		}
		if regionBytes != nil {
			return regionBytes, nil
		}
	}

//从块索引中查找文件中块的位置。
	blockRow, err := tx.fetchBlockRow(region.Hash)
	if err != nil {
		return nil, err
	}
	location := deserializeBlockLoc(blockRow)

//确保区域在块的范围内。
	endOffset := region.Offset + region.Len
	if endOffset < region.Offset || endOffset > location.blockLen {
		str := fmt.Sprintf("block %s region offset %d, length %d "+
			"exceeds block length of %d", region.Hash,
			region.Offset, region.Len, location.blockLen)
		return nil, makeDbErr(database.ErrBlockRegionInvalid, str, nil)

	}

//从相应的磁盘块文件中读取区域。
	regionBytes, err := tx.db.store.readBlockRegion(location, region.Offset,
		region.Len)
	if err != nil {
		return nil, err
	}

	return regionBytes, nil
}

//FetchBlockRegions返回给定块的原始序列化字节
//区域。
//
//例如，可以直接提取比特币交易和/或
//使用此函数的各个块的脚本。取决于后端
//实现，这可以通过避免
//加载整个块。
//
//原始字节采用wire.msgblock上的serialize返回的格式，并且
//提供的块区域中的偏移字段基于零，并且相对于
//块的开头（字节0）。
//
//根据接口约定返回以下错误：
//-errblocknotfound如果任何请求块哈希不存在
//-errblockregioninvalid如果一个或多个区域超出
//关联块
//-如果事务已关闭，则返回errtxclosed
//-errCorrupt如果数据库已损坏
//
//此外，如果在读取
//阻止文件。
//
//注意：此函数返回的数据仅在数据库中有效
//交易。在事务结束后尝试访问它的结果
//在未定义的行为中。此约束防止额外的数据复制和
//允许支持内存映射数据库实现。
//
//此函数是database.tx接口实现的一部分。
func (tx *transaction) FetchBlockRegions(regions []database.BlockRegion) ([][]byte, error) {
//确保事务状态有效。
	if err := tx.checkClosed(); err != nil {
		return nil, err
	}

//注意：这可以检查之前是否存在所有块
//反序列化位置并建立提取列表，其中
//在失败的情况下会更快，但是呼叫者不会
//通常使用无效值调用此函数，因此优化
//对于一般情况。

//注意：这里的一个潜在优化是将相邻的
//减少读取次数的区域。

//为了提高批量数据的加载效率，首先抓取
//所有请求的块哈希和排序的块位置
//reads by filenum:偏移量，以便所有读取都按文件分组
//在每个文件中都是线性的。这会导致相当大的
//性能的提高取决于请求散列的展开方式
//通过减少文件打开/关闭和随机访问的数量
//需要。这个fetchlist被故意分配了一个cap，因为
//某些区域可能从挂起的块中提取，并且
//因此，不需要从磁盘中提取这些数据。
	blockRegions := make([][]byte, len(regions))
	fetchList := make([]bulkFetchData, 0, len(regions))
	for i := range regions {
		region := &regions[i]

//当块在提交时等待写入时，获取
//字节。
		if tx.pendingBlocks != nil {
			regionBytes, err := tx.fetchPendingRegion(region)
			if err != nil {
				return nil, err
			}
			if regionBytes != nil {
				blockRegions[i] = regionBytes
				continue
			}
		}

//从块中查找块在文件中的位置
//索引。
		blockRow, err := tx.fetchBlockRow(region.Hash)
		if err != nil {
			return nil, err
		}
		location := deserializeBlockLoc(blockRow)

//确保区域在块的范围内。
		endOffset := region.Offset + region.Len
		if endOffset < region.Offset || endOffset > location.blockLen {
			str := fmt.Sprintf("block %s region offset %d, length "+
				"%d exceeds block length of %d", region.Hash,
				region.Offset, region.Len, location.blockLen)
			return nil, makeDbErr(database.ErrBlockRegionInvalid, str, nil)
		}

		fetchList = append(fetchList, bulkFetchData{&location, i})
	}
	sort.Sort(bulkFetchDataSorter(fetchList))

//读取提取列表中的所有区域并设置结果。
	for i := range fetchList {
		fetchData := &fetchList[i]
		ri := fetchData.replyIndex
		region := &regions[ri]
		location := fetchData.blockLocation
		regionBytes, err := tx.db.store.readBlockRegion(*location,
			region.Offset, region.Len)
		if err != nil {
			return nil, err
		}
		blockRegions[ri] = regionBytes
	}

	return blockRegions, nil
}

//关闭标记事务已关闭，然后释放任何挂起的数据，
//当
//事务是可写的。
func (tx *transaction) close() {
	tx.closed = true

//清除提交时写入的挂起块。
	tx.pendingBlocks = nil
	tx.pendingBlockData = nil

//清除提交时可能已写入或删除的挂起密钥。
	tx.pendingKeys = nil
	tx.pendingRemove = nil

//释放快照。
	if tx.snapshot != nil {
		tx.snapshot.Release()
		tx.snapshot = nil
	}

	tx.db.closeLock.RUnlock()

//释放可写事务的编写器锁以取消阻止任何
//其他可能正在等待的写入事务。
	if tx.writable {
		tx.db.writeLock.Unlock()
	}
}

//writePendingAndCommit将挂起的块数据写入平面块文件，
//更新元数据及其位置以及新的当前写入
//位置，并将元数据提交到内存数据库缓存。它也
//在出现故障时正确处理回滚。
//
//只有在有挂起的数据要写入时，才能调用此函数。
func (tx *transaction) writePendingAndCommit() error {
//为可能的回滚保存当前块存储写入位置。
//这些变量仅在此函数中更新，并且
//一次只激活一个写事务，因此可以安全地存储
//它们用于潜在的回滚。
	wc := tx.db.store.writeCursor
	wc.RLock()
	oldBlkFileNum := wc.curFileNum
	oldBlkOffset := wc.curOffset
	wc.RUnlock()

//Rollback是一个闭包，用于将所有写入回滚到
//阻止文件。
	rollback := func() {
//如果需要，回滚对块文件所做的任何修改。
		tx.db.store.handleRollback(oldBlkFileNum, oldBlkOffset)
	}

//循环访问所有挂起的块以存储和写入它们。
	for _, blockData := range tx.pendingBlockData {
		log.Tracef("Storing block %s", blockData.hash)
		location, err := tx.db.store.writeBlock(blockData.bytes)
		if err != nil {
			rollback()
			return err
		}

//在块索引中为块添加一条记录。记录
//包括定位块所需的位置信息
//在文件系统和块头上，因为它们是
//如此普遍的需要。
		blockRow := serializeBlockLoc(location)
		err = tx.blockIdxBucket.Put(blockData.hash[:], blockRow)
		if err != nil {
			rollback()
			return err
		}
	}

//更新当前写入文件和偏移量的元数据。
	writeRow := serializeWriteRow(wc.curFileNum, wc.curOffset)
	if err := tx.metaBucket.Put(writeLocKeyName, writeRow); err != nil {
		rollback()
		return convertErr("failed to store write cursor", err)
	}

//自动更新数据库缓存。缓存自动
//处理对基础持久存储数据库的刷新。
	return tx.db.cache.commitTx(tx)
}

//提交提交已对根元数据存储桶进行的所有更改
//以及它的所有子存储桶到定期同步的数据库缓存
//持久存储。此外，它还将所有新块直接提交到
//绕过数据库缓存的持久存储。块可能相当大，所以
//这有助于增加元数据更新可用的缓存量，以及
//是安全的，因为块是不可变的。
//
//此函数是database.tx接口实现的一部分。
func (tx *transaction) Commit() error {
//防止提交托管事务。
	if tx.managed {
		tx.close()
		panic("managed transaction commit not allowed")
	}

//确保事务状态有效。
	if err := tx.checkClosed(); err != nil {
		return err
	}

//无论提交是否成功，事务都将关闭。
//返回。
	defer tx.close()

//确保事务是可写的。
	if !tx.writable {
		str := "Commit requires a writable database transaction"
		return makeDbErr(database.ErrTxNotWritable, str, nil)
	}

//写入挂起的数据。如果出现任何错误，函数将回滚。
	return tx.writePendingAndCommit()
}

//回滚将撤消对根bucket和所有
//它的子桶。
//
//此函数是database.tx接口实现的一部分。
func (tx *transaction) Rollback() error {
//防止对托管事务进行回滚。
	if tx.managed {
		tx.close()
		panic("managed transaction rollback not allowed")
	}

//确保事务状态有效。
	if err := tx.checkClosed(); err != nil {
		return err
	}

	tx.close()
	return nil
}

//db表示持久化和实现的命名空间集合
//database.db接口。所有数据库访问都是通过
//通过特定命名空间获取的事务。
type db struct {
writeLock sync.Mutex   //一次只能写一个事务。
closeLock sync.RWMutex //使数据库在txns活动时关闭块。
closed    bool         //数据库是否已关闭？
store     *blockStore  //将读/写块处理为平面文件。
cache     *dbCache     //包装底层数据库的缓存层。
}

//强制db实现database.db接口。
var _ database.DB = (*db)(nil)

//类型返回当前数据库实例的数据库驱动程序类型
//创建与。
//
//此函数是database.db接口实现的一部分。
func (db *db) Type() string {
	return dbType
}

//begin是begin数据库方法的实现函数。见其
//有关详细信息的文档。
//
//此函数是单独的，因为它返回内部事务
//当数据库方法
//返回接口。
func (db *db) begin(writable bool) (*transaction, error) {
//每当启动新的可写事务时，获取写锁
//确保只有一个写事务可以同时处于活动状态
//时间。在事务处理完成之前，不会释放此锁。
//关闭（通过回滚或提交）。
	if writable {
		db.writeLock.Lock()
	}

//每当启动新事务时，获取
//确保关闭的数据库将等待事务完成。
//在事务关闭之前，不会释放此锁（通过
//回滚或提交）。
	db.closeLock.RLock()
	if db.closed {
		db.closeLock.RUnlock()
		if writable {
			db.writeLock.Unlock()
		}
		return nil, makeDbErr(database.ErrDbNotOpen, errDbNotOpenStr,
			nil)
	}

//获取数据库缓存的快照（反过来，该快照还处理
//基础数据库）。
	snapshot, err := db.cache.Snapshot()
	if err != nil {
		db.closeLock.RUnlock()
		if writable {
			db.writeLock.Unlock()
		}

		return nil, err
	}

//元数据和块索引存储桶只是内部存储桶，因此
//他们定义了ID。
	tx := &transaction{
		writable:      writable,
		db:            db,
		snapshot:      snapshot,
		pendingKeys:   treap.NewMutable(),
		pendingRemove: treap.NewMutable(),
	}
	tx.metaBucket = &bucket{tx: tx, id: metadataBucketID}
	tx.blockIdxBucket = &bucket{tx: tx, id: blockIdxBucketID}
	return tx, nil
}

//BEGIN启动一个只读或读写的事务，具体取决于
//在指定的标志上。可以启动多个只读事务
//同时，在
//时间。当启动读写事务时，调用将阻塞
//已经开放。
//
//注意：当
//它不再需要了。如果不这样做，将导致内存无人认领。
//
//此函数是database.db接口实现的一部分。
func (db *db) Begin(writable bool) (database.Tx, error) {
	return db.begin(writable)
}

//如果调用中的代码
//函数恐慌。这是必需的，因为事务上的互斥体必须
//释放和恐慌在所谓的代码将阻止这种情况的发生。
//
//注意：这只能手动处理托管事务，因为它们
//控制事务的生命周期。文件开始时
//呼出，选择使用手动交易的呼叫者必须确保
//如果事务还需要该功能，那么它将在恐慌时回滚。
//否则数据库将无法关闭，因为读取锁将永远不会
//释放。
func rollbackOnPanic(tx *transaction) {
	if err := recover(); err != nil {
		tx.managed = false
		_ = tx.Rollback()
		panic(err)
	}
}

//视图在托管只读上下文中调用传递的函数
//与命名空间的根bucket的事务。从返回的任何错误
//用户提供的函数将从此函数返回。
//
//此函数是database.db接口实现的一部分。
func (db *db) View(fn func(database.Tx) error) error {
//启动只读事务。
	tx, err := db.begin(false)
	if err != nil {
		return err
	}

//由于用户提供的函数可能会死机，请确保事务
//释放所有互斥体和资源。不能保证打电话的人
//不会用“恢复”继续前进。因此，数据库必须
//由于调用方问题而出现恐慌时处于可用状态。
	defer rollbackOnPanic(tx)

	tx.managed = true
	err = fn(tx)
	tx.managed = false
	if err != nil {
//此处忽略错误，因为尚未写入任何内容
//不管回滚失败如何，Tx现在都关闭了
//不管怎样。
		_ = tx.Rollback()
		return err
	}

	return tx.Rollback()
}

//更新在托管读写上下文中调用传递的函数
//与命名空间的根bucket的事务。从返回的任何错误
//用户提供的函数将导致事务回滚，并且
//从该函数返回。否则，事务被提交
//当用户提供的函数返回nil错误时。
//
//此函数是database.db接口实现的一部分。
func (db *db) Update(fn func(database.Tx) error) error {
//启动读写事务。
	tx, err := db.begin(true)
	if err != nil {
		return err
	}

//由于用户提供的函数可能会死机，请确保事务
//释放所有互斥体和资源。不能保证打电话的人
//不会用“恢复”继续前进。因此，数据库必须
//由于调用方问题而出现恐慌时处于可用状态。
	defer rollbackOnPanic(tx)

	tx.managed = true
	err = fn(tx)
	tx.managed = false
	if err != nil {
//此处忽略错误，因为尚未写入任何内容
//不管回滚失败如何，Tx现在都关闭了
//不管怎样。
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

//CLOSE干净地关闭数据库并同步所有数据。它会阻止
//直到所有数据库事务完成（回滚或
//承诺）。
//
//此函数是database.db接口实现的一部分。
func (db *db) Close() error {
//由于所有事务在此互斥体上都有一个读取锁，因此
//使close等待所有读卡器完成。
	db.closeLock.Lock()
	defer db.closeLock.Unlock()

	if db.closed {
		return makeDbErr(database.ErrDbNotOpen, errDbNotOpenStr, nil)
	}
	db.closed = true

//注意：由于上述锁等待所有事务完成，并且
//防止任何新的启动，可以安全地冲洗
//缓存并清除所有状态，而不使用单独的锁。

//关闭将刷新任何现有项的数据库缓存
//磁盘并关闭底层的LevelDB数据库。保存任何错误
//在清理完之后返回
//即使失败，数据库也将被标记为关闭，因为没有
//无论如何，呼叫方从故障中恢复的好方法。
	closeErr := db.cache.Close()

//关闭所有放置木块的打开的平面文件。
	wc := db.store.writeCursor
	if wc.curFile.file != nil {
		_ = wc.curFile.file.Close()
		wc.curFile.file = nil
	}
	for _, blockFile := range db.store.openBlockFiles {
		_ = blockFile.file.Close()
	}
	db.store.openBlockFiles = nil
	db.store.openBlocksLRU.Init()
	db.store.fileNumToLRUElem = nil

	return closeErr
}

//filesexists报告命名文件或目录是否存在。
func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

//initdb创建包使用的初始存储桶和值。这是
//主要用于测试目的的单独功能。
func initDB(ldb *leveldb.DB) error {
//起始块文件写入光标位置为文件编号0，偏移量
//0。
	batch := new(leveldb.Batch)
	batch.Put(bucketizedKey(metadataBucketID, writeLocKeyName),
		serializeWriteRow(0, 0))

//创建块索引bucket并设置当前bucket id。
//
//注意：由于存储桶是通过使用前缀虚拟化的，
//不需要存储元数据的bucket索引数据
//数据库中的存储桶。但是，要使用的第一个bucket id
//需要考虑它，以确保没有密钥冲突。
	batch.Put(bucketIndexKey(metadataBucketID, blockIdxBucketName),
		blockIdxBucketID[:])
	batch.Put(curBucketIDKeyName, blockIdxBucketID[:])

//把每件事都写成一批。
	if err := ldb.Write(batch, nil); err != nil {
		str := fmt.Sprintf("failed to initialize metadata database: %v",
			err)
		return convertErr(str, err)
	}

	return nil
}

//opendb以提供的路径打开数据库。数据库.errdbdoesnotex列表
//如果数据库不存在且未设置创建标志，则返回。
func openDB(dbPath string, network wire.BitcoinNet, create bool) (database.DB, error) {
//如果数据库不存在且未设置创建标志，则出错。
	metadataDbPath := filepath.Join(dbPath, metadataDbName)
	dbExists := fileExists(metadataDbPath)
	if !create && !dbExists {
		str := fmt.Sprintf("database %q does not exist", metadataDbPath)
		return nil, makeDbErr(database.ErrDbDoesNotExist, str, nil)
	}

//确保数据库的完整路径存在。
	if !dbExists {
//由于调用
//如果目录不能
//创建。
		_ = os.MkdirAll(dbPath, 0700)
	}

//打开元数据数据库（如果需要，将创建它）。
	opts := opt.Options{
		ErrorIfExist: create,
		Strict:       opt.DefaultStrict,
		Compression:  opt.NoCompression,
		Filter:       filter.NewBloomFilter(10),
	}
	ldb, err := leveldb.OpenFile(metadataDbPath, &opts)
	if err != nil {
		return nil, convertErr(err.Error(), err)
	}

//创建包含扫描现有公寓的块存储
//阻止文件以查找当前写入光标位置
//根据磁盘上的实际数据。也创造
//数据库缓存，它包装底层数据库，以提供
//编写缓存。
	store := newBlockStore(dbPath, network)
	cache := newDbCache(ldb, store, defaultCacheSize, defaultFlushSecs)
	pdb := &db{store: store, cache: cache}

//执行块和元数据之间所需的任何协调
//以及数据库初始化（如果需要）。
	return reconcileDB(pdb, create)
}
