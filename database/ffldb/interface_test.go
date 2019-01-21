
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

//此文件将被复制到每个后端驱动程序目录中。各
//驱动程序应该有自己的驱动程序\test.go文件，该文件创建一个数据库和
//调用此文件中的testinterface函数以确保驱动程序正确
//实现接口。
//
//注意：将此文件复制到后端驱动程序文件夹时，包名称
//需要相应更改。

package ffldb_test

import (
	"bytes"
	"compress/bzip2"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

var (
//blockdatanet是测试块数据中的预期网络。
	blockDataNet = wire.MainNet

//blockdatafile是包含前256个块的文件的路径
//关于区块链。
	blockDataFile = filepath.Join("..", "testdata", "blocks1-256.bz2")

//errSubestFail用于指示子测试返回了false。
	errSubTestFail = fmt.Errorf("sub test failure")
)

//加载块加载包含在testdata目录中的块并返回
//一片。
func loadBlocks(t *testing.T, dataFile string, network wire.BitcoinNet) ([]*btcutil.Block, error) {
//打开包含要读取的块的文件。
	fi, err := os.Open(dataFile)
	if err != nil {
		t.Errorf("failed to open file %v, err %v", dataFile, err)
		return nil, err
	}
	defer func() {
		if err := fi.Close(); err != nil {
			t.Errorf("failed to close file %v %v", dataFile,
				err)
		}
	}()
	dr := bzip2.NewReader(fi)

//将第一个区块设置为Genesis区块。
	blocks := make([]*btcutil.Block, 0, 256)
	genesis := btcutil.NewBlock(chaincfg.MainNetParams.GenesisBlock)
	blocks = append(blocks, genesis)

//加载其余块。
	for height := 1; ; height++ {
		var net uint32
		err := binary.Read(dr, binary.LittleEndian, &net)
		if err == io.EOF {
//以预期的偏移量命中文件结尾。没有错误。
			break
		}
		if err != nil {
			t.Errorf("Failed to load network type for block %d: %v",
				height, err)
			return nil, err
		}
		if net != uint32(network) {
			t.Errorf("Block doesn't match network: %v expects %v",
				net, network)
			return nil, err
		}

		var blockLen uint32
		err = binary.Read(dr, binary.LittleEndian, &blockLen)
		if err != nil {
			t.Errorf("Failed to load block size for block %d: %v",
				height, err)
			return nil, err
		}

//读块。
		blockBytes := make([]byte, blockLen)
		_, err = io.ReadFull(dr, blockBytes)
		if err != nil {
			t.Errorf("Failed to load block %d: %v", height, err)
			return nil, err
		}

//反序列化并存储块。
		block, err := btcutil.NewBlockFromBytes(blockBytes)
		if err != nil {
			t.Errorf("Failed to parse block %v: %v", height, err)
			return nil, err
		}
		blocks = append(blocks, block)
	}

	return blocks, nil
}

//checkdberror确保传递的错误是一个数据库。带有错误代码的错误
//与传递的错误代码匹配。
func checkDbError(t *testing.T, testName string, gotErr error, wantErrCode database.ErrorCode) bool {
	dbErr, ok := gotErr.(database.Error)
	if !ok {
		t.Errorf("%s: unexpected error type - got %T, want %T",
			testName, gotErr, database.Error{})
		return false
	}
	if dbErr.ErrorCode != wantErrCode {
		t.Errorf("%s: unexpected error code - got %s (%s), want %s",
			testName, dbErr.ErrorCode, dbErr.Description,
			wantErrCode)
		return false
	}

	return true
}

//test context用于存储有关正在运行的测试的上下文信息，该测试
//传递到helper函数中。
type testContext struct {
	t           *testing.T
	db          database.DB
	bucketDepth int
	isWritable  bool
	blocks      []*btcutil.Block
}

//密钥对包含一个密钥/值对。它在地图上使用，因此可以
//保持。
type keyPair struct {
	key   []byte
	value []byte
}

//lookup key是从
//提供的密钥对切片以及是否找到密钥。
func lookupKey(key []byte, values []keyPair) ([]byte, bool) {
	for _, item := range values {
		if bytes.Equal(item.key, key) {
			return item.value, true
		}
	}

	return nil, false
}

//TogetValues返回一个包含所有nil的所提供密钥对的副本
//值设置为空字节片。这用于确保键设置为
//当检索到nil值而不是nil时，nil值会导致空字节片。
func toGetValues(values []keyPair) []keyPair {
	ret := make([]keyPair, len(values))
	copy(ret, values)
	for i := range ret {
		if ret[i].value == nil {
			ret[i].value = make([]byte, 0)
		}
	}
	return ret
}

//RollbackValues返回所提供的键值对的副本，所有值都设置为
//零。这用于测试值是否正确回滚。
func rollbackValues(values []keyPair) []keyPair {
	ret := make([]keyPair, len(values))
	copy(ret, values)
	for i := range ret {
		ret[i].value = nil
	}
	return ret
}

//TestCursorKeyPair检查提供的键和值是否与预期的匹配
//提供索引处的密钥对。它还确保索引在
//提供了预期的键对切片。
func testCursorKeyPair(tc *testContext, k, v []byte, index int, values []keyPair) bool {
	if index >= len(values) || index < 0 {
		tc.t.Errorf("Cursor: exceeded the expected range of values - "+
			"index %d, num values %d", index, len(values))
		return false
	}

	pair := &values[index]
	if !bytes.Equal(k, pair.key) {
		tc.t.Errorf("Mismatched cursor key: index %d does not match "+
			"the expected key - got %q, want %q", index, k,
			pair.key)
		return false
	}
	if !bytes.Equal(v, pair.value) {
		tc.t.Errorf("Mismatched cursor value: index %d does not match "+
			"the expected value - got %q, want %q", index, v,
			pair.value)
		return false
	}

	return true
}

//testGetValues检查提供的所有键/值对是否可以
//从数据库检索，检索到的值与提供的
//价值观。
func testGetValues(tc *testContext, bucket database.Bucket, values []keyPair) bool {
	for _, item := range values {
		gotValue := bucket.Get(item.key)
		if !reflect.DeepEqual(gotValue, item.value) {
			tc.t.Errorf("Get: unexpected value for %q - got %q, "+
				"want %q", item.key, gotValue, item.value)
			return false
		}
	}

	return true
}

//testputvalues将提供的所有键/值对存储在
//同时检查错误。
func testPutValues(tc *testContext, bucket database.Bucket, values []keyPair) bool {
	for _, item := range values {
		if err := bucket.Put(item.key, item.value); err != nil {
			tc.t.Errorf("Put: unexpected error: %v", err)
			return false
		}
	}

	return true
}

//testDeleteValues从
//提供桶。
func testDeleteValues(tc *testContext, bucket database.Bucket, values []keyPair) bool {
	for _, item := range values {
		if err := bucket.Delete(item.key); err != nil {
			tc.t.Errorf("Delete: unexpected error: %v", err)
			return false
		}
	}

	return true
}

//TestCursorInterface通过
//在传递的桶上执行它的所有功能。
func testCursorInterface(tc *testContext, bucket database.Bucket) bool {
//确保可以为存储桶获取光标。
	cursor := bucket.Cursor()
	if cursor == nil {
		tc.t.Error("Bucket.Cursor: unexpected nil cursor returned")
		return false
	}

//确保光标返回为其创建的相同存储桶。
	if cursor.Bucket() != bucket {
		tc.t.Error("Cursor.Bucket: does not match the bucket it was " +
			"created for")
		return false
	}

	if tc.isWritable {
		unsortedValues := []keyPair{
			{[]byte("cursor"), []byte("val1")},
			{[]byte("abcd"), []byte("val2")},
			{[]byte("bcd"), []byte("val3")},
			{[]byte("defg"), nil},
		}
		sortedValues := []keyPair{
			{[]byte("abcd"), []byte("val2")},
			{[]byte("bcd"), []byte("val3")},
			{[]byte("cursor"), []byte("val1")},
			{[]byte("defg"), nil},
		}

//将光标测试中使用的值存储在未排序的
//订购并确保它们实际存储。
		if !testPutValues(tc, bucket, unsortedValues) {
			return false
		}
		if !testGetValues(tc, bucket, toGetValues(unsortedValues)) {
			return false
		}

//当
//向前迭代。
		curIdx := 0
		for ok := cursor.First(); ok; ok = cursor.Next() {
			k, v := cursor.Key(), cursor.Value()
			if !testCursorKeyPair(tc, k, v, curIdx, sortedValues) {
				return false
			}
			curIdx++
		}
		if curIdx != len(unsortedValues) {
			tc.t.Errorf("Cursor: expected to iterate %d values, "+
				"but only iterated %d", len(unsortedValues),
				curIdx)
			return false
		}

//确保光标返回反向字节排序的所有项目
//反向迭代时的顺序。
		curIdx = len(sortedValues) - 1
		for ok := cursor.Last(); ok; ok = cursor.Prev() {
			k, v := cursor.Key(), cursor.Value()
			if !testCursorKeyPair(tc, k, v, curIdx, sortedValues) {
				return false
			}
			curIdx--
		}
		if curIdx > -1 {
			tc.t.Errorf("Reverse cursor: expected to iterate %d "+
				"values, but only iterated %d",
				len(sortedValues), len(sortedValues)-(curIdx+1))
			return false
		}

//确保前向迭代在查找后按预期工作。
		middleIdx := (len(sortedValues) - 1) / 2
		seekKey := sortedValues[middleIdx].key
		curIdx = middleIdx
		for ok := cursor.Seek(seekKey); ok; ok = cursor.Next() {
			k, v := cursor.Key(), cursor.Value()
			if !testCursorKeyPair(tc, k, v, curIdx, sortedValues) {
				return false
			}
			curIdx++
		}
		if curIdx != len(sortedValues) {
			tc.t.Errorf("Cursor after seek: expected to iterate "+
				"%d values, but only iterated %d",
				len(sortedValues)-middleIdx, curIdx-middleIdx)
			return false
		}

//确保反向迭代在查找后按预期工作。
		curIdx = middleIdx
		for ok := cursor.Seek(seekKey); ok; ok = cursor.Prev() {
			k, v := cursor.Key(), cursor.Value()
			if !testCursorKeyPair(tc, k, v, curIdx, sortedValues) {
				return false
			}
			curIdx--
		}
		if curIdx > -1 {
			tc.t.Errorf("Reverse cursor after seek: expected to "+
				"iterate %d values, but only iterated %d",
				len(sortedValues)-middleIdx, middleIdx-curIdx)
			return false
		}

//确保光标正确删除项目。
		if !cursor.First() {
			tc.t.Errorf("Cursor.First: no value")
			return false
		}
		k := cursor.Key()
		if err := cursor.Delete(); err != nil {
			tc.t.Errorf("Cursor.Delete: unexpected error: %v", err)
			return false
		}
		if val := bucket.Get(k); val != nil {
			tc.t.Errorf("Cursor.Delete: value for key %q was not "+
				"deleted", k)
			return false
		}
	}

	return true
}

//testnestedbucket针对嵌套bucket重新运行testbacketinterface
//用计数器只测试几个水平深度。
func testNestedBucket(tc *testContext, testBucket database.Bucket) bool {
//不要超过2层嵌套深度。
	if tc.bucketDepth > 1 {
		return true
	}

	tc.bucketDepth++
	defer func() {
		tc.bucketDepth--
	}()
	return testBucketInterface(tc, testBucket)
}

//testbacketinterface通过以下方式确保bucket接口正常工作：
//行使其所有职能。这包括
//从存储桶返回的光标。
func testBucketInterface(tc *testContext, bucket database.Bucket) bool {
	if bucket.Writable() != tc.isWritable {
		tc.t.Errorf("Bucket writable state does not match.")
		return false
	}

	if tc.isWritable {
//keyValues保存放置时要使用的键和值
//值进入存储桶。
		keyValues := []keyPair{
			{[]byte("bucketkey1"), []byte("foo1")},
			{[]byte("bucketkey2"), []byte("foo2")},
			{[]byte("bucketkey3"), []byte("foo3")},
			{[]byte("bucketkey4"), nil},
		}
		expectedKeyValues := toGetValues(keyValues)
		if !testPutValues(tc, bucket, keyValues) {
			return false
		}

		if !testGetValues(tc, bucket, expectedKeyValues) {
			return false
		}

//确保从用户提供的foreach返回错误
//返回函数。
		forEachError := fmt.Errorf("example foreach error")
		err := bucket.ForEach(func(k, v []byte) error {
			return forEachError
		})
		if err != forEachError {
			tc.t.Errorf("ForEach: inner function error not "+
				"returned - got %v, want %v", err, forEachError)
			return false
		}

//迭代使用foreach的所有键，同时确保
//存储的值是预期值。
		keysFound := make(map[string]struct{}, len(keyValues))
		err = bucket.ForEach(func(k, v []byte) error {
			wantV, found := lookupKey(k, expectedKeyValues)
			if !found {
				return fmt.Errorf("ForEach: key '%s' should "+
					"exist", k)
			}

			if !reflect.DeepEqual(v, wantV) {
				return fmt.Errorf("ForEach: value for key '%s' "+
					"does not match - got %s, want %s", k,
					v, wantV)
			}

			keysFound[string(k)] = struct{}{}
			return nil
		})
		if err != nil {
			tc.t.Errorf("%v", err)
			return false
		}

//确保所有键都已迭代。
		for _, item := range keyValues {
			if _, ok := keysFound[string(item.key)]; !ok {
				tc.t.Errorf("ForEach: key '%s' was not iterated "+
					"when it should have been", item.key)
				return false
			}
		}

//删除密钥并确保它们已被删除。
		if !testDeleteValues(tc, bucket, keyValues) {
			return false
		}
		if !testGetValues(tc, bucket, rollbackValues(keyValues)) {
			return false
		}

//确保创建新存储桶按预期工作。
		testBucketName := []byte("testbucket")
		testBucket, err := bucket.CreateBucket(testBucketName)
		if err != nil {
			tc.t.Errorf("CreateBucket: unexpected error: %v", err)
			return false
		}
		if !testNestedBucket(tc, testBucket) {
			return false
		}

//确保从用户提供的ForEachBucket返回错误
//返回函数。
		err = bucket.ForEachBucket(func(k []byte) error {
			return forEachError
		})
		if err != forEachError {
			tc.t.Errorf("ForEachBucket: inner function error not "+
				"returned - got %v, want %v", err, forEachError)
			return false
		}

//确保创建已存在的bucket失败
//期望误差。
		wantErrCode := database.ErrBucketExists
		_, err = bucket.CreateBucket(testBucketName)
		if !checkDbError(tc.t, "CreateBucket", err, wantErrCode) {
			return false
		}

//确保CreateBacketifNotexists返回现有Bucket。
		testBucket, err = bucket.CreateBucketIfNotExists(testBucketName)
		if err != nil {
			tc.t.Errorf("CreateBucketIfNotExists: unexpected "+
				"error: %v", err)
			return false
		}
		if !testNestedBucket(tc, testBucket) {
			return false
		}

//确保检索现有存储桶按预期工作。
		testBucket = bucket.Bucket(testBucketName)
		if !testNestedBucket(tc, testBucket) {
			return false
		}

//确保删除存储桶按预期工作。
		if err := bucket.DeleteBucket(testBucketName); err != nil {
			tc.t.Errorf("DeleteBucket: unexpected error: %v", err)
			return false
		}
		if b := bucket.Bucket(testBucketName); b != nil {
			tc.t.Errorf("DeleteBucket: bucket '%s' still exists",
				testBucketName)
			return false
		}

//确保删除不存在的存储桶返回
//期望误差。
		wantErrCode = database.ErrBucketNotFound
		err = bucket.DeleteBucket(testBucketName)
		if !checkDbError(tc.t, "DeleteBucket", err, wantErrCode) {
			return false
		}

//确保CreateBacketifNotexists在以下情况下创建新bucket：
//它还不存在。
		testBucket, err = bucket.CreateBucketIfNotExists(testBucketName)
		if err != nil {
			tc.t.Errorf("CreateBucketIfNotExists: unexpected "+
				"error: %v", err)
			return false
		}
		if !testNestedBucket(tc, testBucket) {
			return false
		}

//确保光标接口按预期工作。
		if !testCursorInterface(tc, testBucket) {
			return false
		}

//删除测试存储桶以避免将来留下它
//电话。
		if err := bucket.DeleteBucket(testBucketName); err != nil {
			tc.t.Errorf("DeleteBucket: unexpected error: %v", err)
			return false
		}
		if b := bucket.Bucket(testBucketName); b != nil {
			tc.t.Errorf("DeleteBucket: bucket '%s' still exists",
				testBucketName)
			return false
		}
	} else {
//Put应该失败，因为Bucket不可写。
		testName := "unwritable tx put"
		wantErrCode := database.ErrTxNotWritable
		failBytes := []byte("fail")
		err := bucket.Put(failBytes, failBytes)
		if !checkDbError(tc.t, testName, err, wantErrCode) {
			return false
		}

//删除应失败，因为存储桶不可写。
		testName = "unwritable tx delete"
		err = bucket.Delete(failBytes)
		if !checkDbError(tc.t, testName, err, wantErrCode) {
			return false
		}

//CreateBack应该失败，因为Bucket不可写。
		testName = "unwritable tx create bucket"
		_, err = bucket.CreateBucket(failBytes)
		if !checkDbError(tc.t, testName, err, wantErrCode) {
			return false
		}

//CreateBacketifnotexists应该失败，因为bucket不是
//可写的。
		testName = "unwritable tx create bucket if not exists"
		_, err = bucket.CreateBucketIfNotExists(failBytes)
		if !checkDbError(tc.t, testName, err, wantErrCode) {
			return false
		}

//DeleteBucket应该失败，因为Bucket不可写。
		testName = "unwritable tx delete bucket"
		err = bucket.DeleteBucket(failBytes)
		if !checkDbError(tc.t, testName, err, wantErrCode) {
			return false
		}

//确保光标接口以只读方式按预期工作
//桶。
		if !testCursorInterface(tc, bucket) {
			return false
		}
	}

	return true
}

//如果调用中的代码
//函数恐慌。这在测试意外恐慌的情况下很有用，
//将使任何手动创建的事务保持数据库互斥锁的状态
//从而导致僵局，掩盖了恐慌的真正原因。它
//还记录一个测试错误和重新绑定，以便跟踪原始死机。
func rollbackOnPanic(t *testing.T, tx database.Tx) {
	if err := recover(); err != nil {
		t.Errorf("Unexpected panic: %v", err)
		_ = tx.Rollback()
		panic(err)
	}
}

//TestMetadataManualTxInterface确保手动事务元数据
//接口按预期工作。
func testMetadataManualTxInterface(tc *testContext) bool {
//填充值的PopulateValues测试按预期工作。
//
//当可写标志为false时，将创建只读转换，
//执行只读事务的标准存储桶测试，以及
//检查commit函数以确保它按预期失败。
//
//否则，将创建读写事务，值为
//读写事务的标准存储桶测试是
//执行，然后提交或回滚事务
//返回取决于标志。
	bucket1Name := []byte("bucket1")
	populateValues := func(writable, rollback bool, putValues []keyPair) bool {
		tx, err := tc.db.Begin(writable)
		if err != nil {
			tc.t.Errorf("Begin: unexpected error %v", err)
			return false
		}
		defer rollbackOnPanic(tc.t, tx)

		metadataBucket := tx.Metadata()
		if metadataBucket == nil {
			tc.t.Errorf("Metadata: unexpected nil bucket")
			_ = tx.Rollback()
			return false
		}

		bucket1 := metadataBucket.Bucket(bucket1Name)
		if bucket1 == nil {
			tc.t.Errorf("Bucket1: unexpected nil bucket")
			return false
		}

		tc.isWritable = writable
		if !testBucketInterface(tc, bucket1) {
			_ = tx.Rollback()
			return false
		}

		if !writable {
//事务不可写，因此应该失败
//提交。
			testName := "unwritable tx commit"
			wantErrCode := database.ErrTxNotWritable
			err := tx.Commit()
			if !checkDbError(tc.t, testName, err, wantErrCode) {
				_ = tx.Rollback()
				return false
			}
		} else {
			if !testPutValues(tc, bucket1, putValues) {
				return false
			}

			if rollback {
//回滚事务。
				if err := tx.Rollback(); err != nil {
					tc.t.Errorf("Rollback: unexpected "+
						"error %v", err)
					return false
				}
			} else {
//承诺应该成功。
				if err := tx.Commit(); err != nil {
					tc.t.Errorf("Commit: unexpected error "+
						"%v", err)
					return false
				}
			}
		}

		return true
	}

//checkvalues启动一个只读事务并检查
//ExpectedValues参数中指定的键/值对匹配
//数据库中有什么。
	checkValues := func(expectedValues []keyPair) bool {
		tx, err := tc.db.Begin(false)
		if err != nil {
			tc.t.Errorf("Begin: unexpected error %v", err)
			return false
		}
		defer rollbackOnPanic(tc.t, tx)

		metadataBucket := tx.Metadata()
		if metadataBucket == nil {
			tc.t.Errorf("Metadata: unexpected nil bucket")
			_ = tx.Rollback()
			return false
		}

		bucket1 := metadataBucket.Bucket(bucket1Name)
		if bucket1 == nil {
			tc.t.Errorf("Bucket1: unexpected nil bucket")
			return false
		}

		if !testGetValues(tc, bucket1, expectedValues) {
			_ = tx.Rollback()
			return false
		}

//回滚只读事务。
		if err := tx.Rollback(); err != nil {
			tc.t.Errorf("Commit: unexpected error %v", err)
			return false
		}

		return true
	}

//DeleteValues启动读写事务并删除键
//在传递的键/值对中。
	deleteValues := func(values []keyPair) bool {
		tx, err := tc.db.Begin(true)
		if err != nil {

		}
		defer rollbackOnPanic(tc.t, tx)

		metadataBucket := tx.Metadata()
		if metadataBucket == nil {
			tc.t.Errorf("Metadata: unexpected nil bucket")
			_ = tx.Rollback()
			return false
		}

		bucket1 := metadataBucket.Bucket(bucket1Name)
		if bucket1 == nil {
			tc.t.Errorf("Bucket1: unexpected nil bucket")
			return false
		}

//删除密钥并确保它们已被删除。
		if !testDeleteValues(tc, bucket1, values) {
			_ = tx.Rollback()
			return false
		}
		if !testGetValues(tc, bucket1, rollbackValues(values)) {
			_ = tx.Rollback()
			return false
		}

//提交更改并确保成功。
		if err := tx.Commit(); err != nil {
			tc.t.Errorf("Commit: unexpected error %v", err)
			return false
		}

		return true
	}

//keyValues保存将值放入
//桶。
	var keyValues = []keyPair{
		{[]byte("umtxkey1"), []byte("foo1")},
		{[]byte("umtxkey2"), []byte("foo2")},
		{[]byte("umtxkey3"), []byte("foo3")},
		{[]byte("umtxkey4"), nil},
	}

//确保尝试使用只读填充值
//事务按预期失败。
	if !populateValues(false, true, keyValues) {
		return false
	}
	if !checkValues(rollbackValues(keyValues)) {
		return false
	}

//确保尝试使用读写填充值
//事务，然后将其回滚，得到预期值。
	if !populateValues(true, true, keyValues) {
		return false
	}
	if !checkValues(rollbackValues(keyValues)) {
		return false
	}

//确保尝试使用读写填充值
//事务，然后提交它来存储期望的值。
	if !populateValues(true, false, keyValues) {
		return false
	}
	if !checkValues(toGetValues(keyValues)) {
		return false
	}

//把钥匙清理干净。
	if !deleteValues(keyValues) {
		return false
	}

	return true
}

//testmanagedtxpanics确保在托管的
//交易恐慌。
func testManagedTxPanics(tc *testContext) bool {
	testPanic := func(fn func()) (paniced bool) {
//设置延迟以捕获预期的恐慌并更新
//返回变量。
		defer func() {
			if err := recover(); err != nil {
				paniced = true
			}
		}()

		fn()
		return false
	}

//确保在托管只读事务上调用commit时出现故障。
	paniced := testPanic(func() {
		tc.db.View(func(tx database.Tx) error {
			tx.Commit()
			return nil
		})
	})
	if !paniced {
		tc.t.Error("Commit called inside View did not panic")
		return false
	}

//确保在托管只读事务上调用rollback时出现紧急情况。
	paniced = testPanic(func() {
		tc.db.View(func(tx database.Tx) error {
			tx.Rollback()
			return nil
		})
	})
	if !paniced {
		tc.t.Error("Rollback called inside View did not panic")
		return false
	}

//确保在托管读写事务上调用commit时出现紧急情况。
	paniced = testPanic(func() {
		tc.db.Update(func(tx database.Tx) error {
			tx.Commit()
			return nil
		})
	})
	if !paniced {
		tc.t.Error("Commit called inside Update did not panic")
		return false
	}

//确保在托管读写事务上调用rollback时出现紧急情况。
	paniced = testPanic(func() {
		tc.db.Update(func(tx database.Tx) error {
			tx.Rollback()
			return nil
		})
	})
	if !paniced {
		tc.t.Error("Rollback called inside Update did not panic")
		return false
	}

	return true
}

//TestMetadataTxInterface测试托管读/写的所有方面，以及
//手动事务元数据接口以及下面的bucket接口
//他们。
func testMetadataTxInterface(tc *testContext) bool {
	if !testManagedTxPanics(tc) {
		return false
	}

	bucket1Name := []byte("bucket1")
	err := tc.db.Update(func(tx database.Tx) error {
		_, err := tx.Metadata().CreateBucket(bucket1Name)
		return err
	})
	if err != nil {
		tc.t.Errorf("Update: unexpected error creating bucket: %v", err)
		return false
	}

	if !testMetadataManualTxInterface(tc) {
		return false
	}

//keyValues保存放置值时要使用的键和值
//变成一个桶。
	keyValues := []keyPair{
		{[]byte("mtxkey1"), []byte("foo1")},
		{[]byte("mtxkey2"), []byte("foo2")},
		{[]byte("mtxkey3"), []byte("foo3")},
		{[]byte("mtxkey4"), nil},
	}

//通过托管只读事务测试bucket接口。
	err = tc.db.View(func(tx database.Tx) error {
		metadataBucket := tx.Metadata()
		if metadataBucket == nil {
			return fmt.Errorf("Metadata: unexpected nil bucket")
		}

		bucket1 := metadataBucket.Bucket(bucket1Name)
		if bucket1 == nil {
			return fmt.Errorf("Bucket1: unexpected nil bucket")
		}

		tc.isWritable = false
		if !testBucketInterface(tc, bucket1) {
			return errSubTestFail
		}

		return nil
	})
	if err != nil {
		if err != errSubTestFail {
			tc.t.Errorf("%v", err)
		}
		return false
	}

//确保从用户提供的视图函数返回的错误为
//返回。
	viewError := fmt.Errorf("example view error")
	err = tc.db.View(func(tx database.Tx) error {
		return viewError
	})
	if err != viewError {
		tc.t.Errorf("View: inner function error not returned - got "+
			"%v, want %v", err, viewError)
		return false
	}

//通过托管读写事务测试bucket接口。
//另外，放置一系列值并强制回滚，因此
//代码可以确保没有存储值。
	forceRollbackError := fmt.Errorf("force rollback")
	err = tc.db.Update(func(tx database.Tx) error {
		metadataBucket := tx.Metadata()
		if metadataBucket == nil {
			return fmt.Errorf("Metadata: unexpected nil bucket")
		}

		bucket1 := metadataBucket.Bucket(bucket1Name)
		if bucket1 == nil {
			return fmt.Errorf("Bucket1: unexpected nil bucket")
		}

		tc.isWritable = true
		if !testBucketInterface(tc, bucket1) {
			return errSubTestFail
		}

		if !testPutValues(tc, bucket1, keyValues) {
			return errSubTestFail
		}

//返回一个错误以强制回滚。
		return forceRollbackError
	})
	if err != forceRollbackError {
		if err == errSubTestFail {
			return false
		}

		tc.t.Errorf("Update: inner function error not returned - got "+
			"%v, want %v", err, forceRollbackError)
		return false
	}

//确保由于强制
//上面的回滚实际上没有存储。
	err = tc.db.View(func(tx database.Tx) error {
		metadataBucket := tx.Metadata()
		if metadataBucket == nil {
			return fmt.Errorf("Metadata: unexpected nil bucket")
		}

		if !testGetValues(tc, metadataBucket, rollbackValues(keyValues)) {
			return errSubTestFail
		}

		return nil
	})
	if err != nil {
		if err != errSubTestFail {
			tc.t.Errorf("%v", err)
		}
		return false
	}

//通过托管读写事务存储一系列值。
	err = tc.db.Update(func(tx database.Tx) error {
		metadataBucket := tx.Metadata()
		if metadataBucket == nil {
			return fmt.Errorf("Metadata: unexpected nil bucket")
		}

		bucket1 := metadataBucket.Bucket(bucket1Name)
		if bucket1 == nil {
			return fmt.Errorf("Bucket1: unexpected nil bucket")
		}

		if !testPutValues(tc, bucket1, keyValues) {
			return errSubTestFail
		}

		return nil
	})
	if err != nil {
		if err != errSubTestFail {
			tc.t.Errorf("%v", err)
		}
		return false
	}

//确保按预期提交以上存储的值。
	err = tc.db.View(func(tx database.Tx) error {
		metadataBucket := tx.Metadata()
		if metadataBucket == nil {
			return fmt.Errorf("Metadata: unexpected nil bucket")
		}

		bucket1 := metadataBucket.Bucket(bucket1Name)
		if bucket1 == nil {
			return fmt.Errorf("Bucket1: unexpected nil bucket")
		}

		if !testGetValues(tc, bucket1, toGetValues(keyValues)) {
			return errSubTestFail
		}

		return nil
	})
	if err != nil {
		if err != errSubTestFail {
			tc.t.Errorf("%v", err)
		}
		return false
	}

//清除托管读写事务中存储在上面的值。
	err = tc.db.Update(func(tx database.Tx) error {
		metadataBucket := tx.Metadata()
		if metadataBucket == nil {
			return fmt.Errorf("Metadata: unexpected nil bucket")
		}

		bucket1 := metadataBucket.Bucket(bucket1Name)
		if bucket1 == nil {
			return fmt.Errorf("Bucket1: unexpected nil bucket")
		}

		if !testDeleteValues(tc, bucket1, keyValues) {
			return errSubTestFail
		}

		return nil
	})
	if err != nil {
		if err != errSubTestFail {
			tc.t.Errorf("%v", err)
		}
		return false
	}

	return true
}

//testfitchblockiomissing确保所有的块检索API函数
//请求不存在的块时按预期工作。
func testFetchBlockIOMissing(tc *testContext, tx database.Tx) bool {
	wantErrCode := database.ErrBlockNotFound

//-----------------
//非批量块IO API
//-----------------

//一次测试一个块API，以确保它们
//返回预期的错误。此外，构建测试
//循环时在下面批量API。
	allBlockHashes := make([]chainhash.Hash, len(tc.blocks))
	allBlockRegions := make([]database.BlockRegion, len(tc.blocks))
	for i, block := range tc.blocks {
		blockHash := block.Hash()
		allBlockHashes[i] = *blockHash

		txLocs, err := block.TxLoc()
		if err != nil {
			tc.t.Errorf("block.TxLoc(%d): unexpected error: %v", i,
				err)
			return false
		}

//确保FetchBlock返回预期错误。
		testName := fmt.Sprintf("FetchBlock #%d on missing block", i)
		_, err = tx.FetchBlock(blockHash)
		if !checkDbError(tc.t, testName, err, wantErrCode) {
			return false
		}

//确保FetchBlockHeader返回预期错误。
		testName = fmt.Sprintf("FetchBlockHeader #%d on missing block",
			i)
		_, err = tx.FetchBlockHeader(blockHash)
		if !checkDbError(tc.t, testName, err, wantErrCode) {
			return false
		}

//确保第一个事务作为块区域从
//数据库返回预期的错误。
		region := database.BlockRegion{
			Hash:   blockHash,
			Offset: uint32(txLocs[0].TxStart),
			Len:    uint32(txLocs[0].TxLen),
		}
		allBlockRegions[i] = region
		_, err = tx.FetchBlockRegion(&region)
		if !checkDbError(tc.t, testName, err, wantErrCode) {
			return false
		}

//确保hasblock返回false。
		hasBlock, err := tx.HasBlock(blockHash)
		if err != nil {
			tc.t.Errorf("HasBlock #%d: unexpected err: %v", i, err)
			return false
		}
		if hasBlock {
			tc.t.Errorf("HasBlock #%d: should not have block", i)
			return false
		}
	}

//------------
//批量块IO API
//------------

//确保FetchBlocks返回预期错误。
	testName := "FetchBlocks on missing blocks"
	_, err := tx.FetchBlocks(allBlockHashes)
	if !checkDbError(tc.t, testName, err, wantErrCode) {
		return false
	}

//确保FETCHAMBESTHOMER返回预期错误。
	testName = "FetchBlockHeaders on missing blocks"
	_, err = tx.FetchBlockHeaders(allBlockHashes)
	if !checkDbError(tc.t, testName, err, wantErrCode) {
		return false
	}

//确保FetchBlockRegions返回预期错误。
	testName = "FetchBlockRegions on missing blocks"
	_, err = tx.FetchBlockRegions(allBlockRegions)
	if !checkDbError(tc.t, testName, err, wantErrCode) {
		return false
	}

//确保所有块的hasblocks返回false。
	hasBlocks, err := tx.HasBlocks(allBlockHashes)
	if err != nil {
		tc.t.Errorf("HasBlocks: unexpected err: %v", err)
	}
	for i, hasBlock := range hasBlocks {
		if hasBlock {
			tc.t.Errorf("HasBlocks #%d: should not have block", i)
			return false
		}
	}

	return true
}

//testfitchblockio确保所有的块检索API函数都可以
//应为提供的块集。块必须已存储在
//数据库，或者至少存储到传递的事务中。它也
//测试几个错误条件，例如确保预期错误
//获取不存在的块、头和区域时返回。
func testFetchBlockIO(tc *testContext, tx database.Tx) bool {
//-----------------
//非批量块IO API
//-----------------

//一次测试一个块API。此外，构建
//循环时测试下面的批量API所需的数据。
	allBlockHashes := make([]chainhash.Hash, len(tc.blocks))
	allBlockBytes := make([][]byte, len(tc.blocks))
	allBlockTxLocs := make([][]wire.TxLoc, len(tc.blocks))
	allBlockRegions := make([]database.BlockRegion, len(tc.blocks))
	for i, block := range tc.blocks {
		blockHash := block.Hash()
		allBlockHashes[i] = *blockHash

		blockBytes, err := block.Bytes()
		if err != nil {
			tc.t.Errorf("block.Bytes(%d): unexpected error: %v", i,
				err)
			return false
		}
		allBlockBytes[i] = blockBytes

		txLocs, err := block.TxLoc()
		if err != nil {
			tc.t.Errorf("block.TxLoc(%d): unexpected error: %v", i,
				err)
			return false
		}
		allBlockTxLocs[i] = txLocs

//确保从数据库中提取的块数据与
//期望字节数。
		gotBlockBytes, err := tx.FetchBlock(blockHash)
		if err != nil {
			tc.t.Errorf("FetchBlock(%s): unexpected error: %v",
				blockHash, err)
			return false
		}
		if !bytes.Equal(gotBlockBytes, blockBytes) {
			tc.t.Errorf("FetchBlock(%s): bytes mismatch: got %x, "+
				"want %x", blockHash, gotBlockBytes, blockBytes)
			return false
		}

//确保从数据库中提取的块头与
//期望字节数。
		wantHeaderBytes := blockBytes[0:wire.MaxBlockHeaderPayload]
		gotHeaderBytes, err := tx.FetchBlockHeader(blockHash)
		if err != nil {
			tc.t.Errorf("FetchBlockHeader(%s): unexpected error: %v",
				blockHash, err)
			return false
		}
		if !bytes.Equal(gotHeaderBytes, wantHeaderBytes) {
			tc.t.Errorf("FetchBlockHeader(%s): bytes mismatch: "+
				"got %x, want %x", blockHash, gotHeaderBytes,
				wantHeaderBytes)
			return false
		}

//确保第一个事务作为块区域从
//数据库与预期的字节匹配。
		region := database.BlockRegion{
			Hash:   blockHash,
			Offset: uint32(txLocs[0].TxStart),
			Len:    uint32(txLocs[0].TxLen),
		}
		allBlockRegions[i] = region
		endRegionOffset := region.Offset + region.Len
		wantRegionBytes := blockBytes[region.Offset:endRegionOffset]
		gotRegionBytes, err := tx.FetchBlockRegion(&region)
		if err != nil {
			tc.t.Errorf("FetchBlockRegion(%s): unexpected error: %v",
				blockHash, err)
			return false
		}
		if !bytes.Equal(gotRegionBytes, wantRegionBytes) {
			tc.t.Errorf("FetchBlockRegion(%s): bytes mismatch: "+
				"got %x, want %x", blockHash, gotRegionBytes,
				wantRegionBytes)
			return false
		}

//确保从数据库中提取的块头与
//期望字节数。
		hasBlock, err := tx.HasBlock(blockHash)
		if err != nil {
			tc.t.Errorf("HasBlock(%s): unexpected error: %v",
				blockHash, err)
			return false
		}
		if !hasBlock {
			tc.t.Errorf("HasBlock(%s): database claims it doesn't "+
				"have the block when it should", blockHash)
			return false
		}

//-----------------
//块/区域无效。
//-----------------

//确保获取不存在的块时返回
//期望误差。
		badBlockHash := &chainhash.Hash{}
		testName := fmt.Sprintf("FetchBlock(%s) invalid block",
			badBlockHash)
		wantErrCode := database.ErrBlockNotFound
		_, err = tx.FetchBlock(badBlockHash)
		if !checkDbError(tc.t, testName, err, wantErrCode) {
			return false
		}

//确保获取不存在的块头返回
//预期的错误。
		testName = fmt.Sprintf("FetchBlockHeader(%s) invalid block",
			badBlockHash)
		_, err = tx.FetchBlockHeader(badBlockHash)
		if !checkDbError(tc.t, testName, err, wantErrCode) {
			return false
		}

//确保获取不存在的块中的块区域
//返回预期的错误。
		testName = fmt.Sprintf("FetchBlockRegion(%s) invalid hash",
			badBlockHash)
		wantErrCode = database.ErrBlockNotFound
		region.Hash = badBlockHash
		region.Offset = ^uint32(0)
		_, err = tx.FetchBlockRegion(&region)
		if !checkDbError(tc.t, testName, err, wantErrCode) {
			return false
		}

//确保获取越界的块区域返回
//预期的错误。
		testName = fmt.Sprintf("FetchBlockRegion(%s) invalid region",
			blockHash)
		wantErrCode = database.ErrBlockRegionInvalid
		region.Hash = blockHash
		region.Offset = ^uint32(0)
		_, err = tx.FetchBlockRegion(&region)
		if !checkDbError(tc.t, testName, err, wantErrCode) {
			return false
		}
	}

//------------
//批量块IO API
//------------

//确保从数据库中获取的大容量块数据与
//期望字节数。
	blockData, err := tx.FetchBlocks(allBlockHashes)
	if err != nil {
		tc.t.Errorf("FetchBlocks: unexpected error: %v", err)
		return false
	}
	if len(blockData) != len(allBlockBytes) {
		tc.t.Errorf("FetchBlocks: unexpected number of results - got "+
			"%d, want %d", len(blockData), len(allBlockBytes))
		return false
	}
	for i := 0; i < len(blockData); i++ {
		blockHash := allBlockHashes[i]
		wantBlockBytes := allBlockBytes[i]
		gotBlockBytes := blockData[i]
		if !bytes.Equal(gotBlockBytes, wantBlockBytes) {
			tc.t.Errorf("FetchBlocks(%s): bytes mismatch: got %x, "+
				"want %x", blockHash, gotBlockBytes,
				wantBlockBytes)
			return false
		}
	}

//确保从数据库中提取的大容量块头与
//期望字节数。
	blockHeaderData, err := tx.FetchBlockHeaders(allBlockHashes)
	if err != nil {
		tc.t.Errorf("FetchBlockHeaders: unexpected error: %v", err)
		return false
	}
	if len(blockHeaderData) != len(allBlockBytes) {
		tc.t.Errorf("FetchBlockHeaders: unexpected number of results "+
			"- got %d, want %d", len(blockHeaderData),
			len(allBlockBytes))
		return false
	}
	for i := 0; i < len(blockHeaderData); i++ {
		blockHash := allBlockHashes[i]
		wantHeaderBytes := allBlockBytes[i][0:wire.MaxBlockHeaderPayload]
		gotHeaderBytes := blockHeaderData[i]
		if !bytes.Equal(gotHeaderBytes, wantHeaderBytes) {
			tc.t.Errorf("FetchBlockHeaders(%s): bytes mismatch: "+
				"got %x, want %x", blockHash, gotHeaderBytes,
				wantHeaderBytes)
			return false
		}
	}

//确保在批量块中获取的每个块的第一个事务
//数据库中的区域与预期的字节匹配。
	allRegionBytes, err := tx.FetchBlockRegions(allBlockRegions)
	if err != nil {
		tc.t.Errorf("FetchBlockRegions: unexpected error: %v", err)
		return false

	}
	if len(allRegionBytes) != len(allBlockRegions) {
		tc.t.Errorf("FetchBlockRegions: unexpected number of results "+
			"- got %d, want %d", len(allRegionBytes),
			len(allBlockRegions))
		return false
	}
	for i, gotRegionBytes := range allRegionBytes {
		region := &allBlockRegions[i]
		endRegionOffset := region.Offset + region.Len
		wantRegionBytes := blockData[i][region.Offset:endRegionOffset]
		if !bytes.Equal(gotRegionBytes, wantRegionBytes) {
			tc.t.Errorf("FetchBlockRegions(%d): bytes mismatch: "+
				"got %x, want %x", i, gotRegionBytes,
				wantRegionBytes)
			return false
		}
	}

//确保批量确定一组块哈希是否在
//对于所有加载的块，数据库返回true。
	hasBlocks, err := tx.HasBlocks(allBlockHashes)
	if err != nil {
		tc.t.Errorf("HasBlocks: unexpected error: %v", err)
		return false
	}
	for i, hasBlock := range hasBlocks {
		if !hasBlock {
			tc.t.Errorf("HasBlocks(%d): should have block", i)
			return false
		}
	}

//-----------------
//块/区域无效。
//-----------------

//确保提取不存在的块时返回
//期望误差。
	testName := "FetchBlocks invalid hash"
	badBlockHashes := make([]chainhash.Hash, len(allBlockHashes)+1)
	copy(badBlockHashes, allBlockHashes)
	badBlockHashes[len(badBlockHashes)-1] = chainhash.Hash{}
	wantErrCode := database.ErrBlockNotFound
	_, err = tx.FetchBlocks(badBlockHashes)
	if !checkDbError(tc.t, testName, err, wantErrCode) {
		return false
	}

//确保获取不存在的块头返回
//期望误差。
	testName = "FetchBlockHeaders invalid hash"
	_, err = tx.FetchBlockHeaders(badBlockHashes)
	if !checkDbError(tc.t, testName, err, wantErrCode) {
		return false
	}

//确保获取其中一个块不存在的块区域
//返回预期错误。
	testName = "FetchBlockRegions invalid hash"
	badBlockRegions := make([]database.BlockRegion, len(allBlockRegions)+1)
	copy(badBlockRegions, allBlockRegions)
	badBlockRegions[len(badBlockRegions)-1].Hash = &chainhash.Hash{}
	wantErrCode = database.ErrBlockNotFound
	_, err = tx.FetchBlockRegions(badBlockRegions)
	if !checkDbError(tc.t, testName, err, wantErrCode) {
		return false
	}

//确保获取超出界限的块区域时返回
//期望误差。
	testName = "FetchBlockRegions invalid regions"
	badBlockRegions = badBlockRegions[:len(badBlockRegions)-1]
	for i := range badBlockRegions {
		badBlockRegions[i].Offset = ^uint32(0)
	}
	wantErrCode = database.ErrBlockRegionInvalid
	_, err = tx.FetchBlockRegions(badBlockRegions)
	return checkDbError(tc.t, testName, err, wantErrCode)
}

//testblockiotxinterface确保块IO接口按预期工作
//对于托管读/写和手动事务。此函数离开
//数据库中的所有存储块。
func testBlockIOTxInterface(tc *testContext) bool {
//确保尝试使用只读事务存储块失败
//出现预期错误。
	err := tc.db.View(func(tx database.Tx) error {
		wantErrCode := database.ErrTxNotWritable
		for i, block := range tc.blocks {
			testName := fmt.Sprintf("StoreBlock(%d) on ro tx", i)
			err := tx.StoreBlock(block)
			if !checkDbError(tc.t, testName, err, wantErrCode) {
				return errSubTestFail
			}
		}

		return nil
	})
	if err != nil {
		if err != errSubTestFail {
			tc.t.Errorf("%v", err)
		}
		return false
	}

//用加载的块填充数据库，并确保所有数据
//在
//提交或回滚。然后，强制回滚，以便下面的代码可以
//确保没有实际存储的数据。
	forceRollbackError := fmt.Errorf("force rollback")
	err = tc.db.Update(func(tx database.Tx) error {
//将所有块存储在同一事务中。
		for i, block := range tc.blocks {
			err := tx.StoreBlock(block)
			if err != nil {
				tc.t.Errorf("StoreBlock #%d: unexpected error: "+
					"%v", i, err)
				return errSubTestFail
			}
		}

//确保在
//事务已提交，返回预期的错误。
		wantErrCode := database.ErrBlockExists
		for i, block := range tc.blocks {
			testName := fmt.Sprintf("duplicate block entry #%d "+
				"(before commit)", i)
			err := tx.StoreBlock(block)
			if !checkDbError(tc.t, testName, err, wantErrCode) {
				return errSubTestFail
			}
		}

//确保之前从存储块中提取所有数据
//事务已按预期提交工作。
		if !testFetchBlockIO(tc, tx) {
			return errSubTestFail
		}

		return forceRollbackError
	})
	if err != forceRollbackError {
		if err == errSubTestFail {
			return false
		}

		tc.t.Errorf("Update: inner function error not returned - got "+
			"%v, want %v", err, forceRollbackError)
		return false
	}

//确保回滚成功
	err = tc.db.View(func(tx database.Tx) error {
		if !testFetchBlockIOMissing(tc, tx) {
			return errSubTestFail
		}
		return nil
	})
	if err != nil {
		if err != errSubTestFail {
			tc.t.Errorf("%v", err)
		}
		return false
	}

//用加载的块填充数据库，并确保所有数据
//获取API工作正常。
	err = tc.db.Update(func(tx database.Tx) error {
//在同一个事务中存储一组块。
		for i, block := range tc.blocks {
			err := tx.StoreBlock(block)
			if err != nil {
				tc.t.Errorf("StoreBlock #%d: unexpected error: "+
					"%v", i, err)
				return errSubTestFail
			}
		}

//确保在
//相同的事务，但在提交之前，返回
//预期的错误。
		for i, block := range tc.blocks {
			testName := fmt.Sprintf("duplicate block entry #%d "+
				"(before commit)", i)
			wantErrCode := database.ErrBlockExists
			err := tx.StoreBlock(block)
			if !checkDbError(tc.t, testName, err, wantErrCode) {
				return errSubTestFail
			}
		}

//确保之前从存储块中提取所有数据
//事务已按预期提交工作。
		if !testFetchBlockIO(tc, tx) {
			return errSubTestFail
		}

		return nil
	})
	if err != nil {
		if err != errSubTestFail {
			tc.t.Errorf("%v", err)
		}
		return false
	}

//确保使用托管的
//成功提交数据后的只读事务
//上面。
	err = tc.db.View(func(tx database.Tx) error {
		if !testFetchBlockIO(tc, tx) {
			return errSubTestFail
		}

		return nil
	})
	if err != nil {
		if err != errSubTestFail {
			tc.t.Errorf("%v", err)
		}
		return false
	}

//确保使用托管的
//成功提交数据后的读写事务
//上面。
	err = tc.db.Update(func(tx database.Tx) error {
		if !testFetchBlockIO(tc, tx) {
			return errSubTestFail
		}

//确保再次尝试存储现有块时返回
//预期错误。请注意，这与
//以前的版本，因为这是在
//已提交块。
		wantErrCode := database.ErrBlockExists
		for i, block := range tc.blocks {
			testName := fmt.Sprintf("duplicate block entry #%d "+
				"(before commit)", i)
			err := tx.StoreBlock(block)
			if !checkDbError(tc.t, testName, err, wantErrCode) {
				return errSubTestFail
			}
		}

		return nil
	})
	if err != nil {
		if err != errSubTestFail {
			tc.t.Errorf("%v", err)
		}
		return false
	}

	return true
}

//testclosedtxinterface确保元数据和块IO API
//当尝试对已关闭的事务执行操作时，函数的行为与预期一致。
func testClosedTxInterface(tc *testContext, tx database.Tx) bool {
	wantErrCode := database.ErrTxClosed
	bucket := tx.Metadata()
	cursor := tx.Metadata().Cursor()
	bucketName := []byte("closedtxbucket")
	keyName := []byte("closedtxkey")

//--------------
//元数据API
//--------------

//确保当
//交易记录已关闭。
	if b := bucket.Bucket(bucketName); b != nil {
		tc.t.Errorf("Bucket: did not return nil on closed tx")
		return false
	}

//确保CreateBack返回预期错误。
	testName := "CreateBucket on closed tx"
	_, err := bucket.CreateBucket(bucketName)
	if !checkDbError(tc.t, testName, err, wantErrCode) {
		return false
	}

//确保CreateBacketifNotexists返回预期错误。
	testName = "CreateBucketIfNotExists on closed tx"
	_, err = bucket.CreateBucketIfNotExists(bucketName)
	if !checkDbError(tc.t, testName, err, wantErrCode) {
		return false
	}

//确保delete返回预期错误。
	testName = "Delete on closed tx"
	err = bucket.Delete(keyName)
	if !checkDbError(tc.t, testName, err, wantErrCode) {
		return false
	}

//确保DeleteBack返回预期错误。
	testName = "DeleteBucket on closed tx"
	err = bucket.DeleteBucket(bucketName)
	if !checkDbError(tc.t, testName, err, wantErrCode) {
		return false
	}

//确保foreach返回预期错误。
	testName = "ForEach on closed tx"
	err = bucket.ForEach(nil)
	if !checkDbError(tc.t, testName, err, wantErrCode) {
		return false
	}

//确保ForeachBucket返回预期错误。
	testName = "ForEachBucket on closed tx"
	err = bucket.ForEachBucket(nil)
	if !checkDbError(tc.t, testName, err, wantErrCode) {
		return false
	}

//确保get返回预期错误。
	testName = "Get on closed tx"
	if k := bucket.Get(keyName); k != nil {
		tc.t.Errorf("Get: did not return nil on closed tx")
		return false
	}

//确保Put返回预期错误。
	testName = "Put on closed tx"
	err = bucket.Put(keyName, []byte("test"))
	if !checkDbError(tc.t, testName, err, wantErrCode) {
		return false
	}

//------------
//元数据光标API
//------------

//确保尝试从关闭的Tx上的光标获取存储桶
//回到零。
	if b := cursor.Bucket(); b != nil {
		tc.t.Error("Cursor.Bucket: returned non-nil on closed tx")
		return false
	}

//确保cursor.delete返回预期错误。
	testName = "Cursor.Delete on closed tx"
	err = cursor.Delete()
	if !checkDbError(tc.t, testName, err, wantErrCode) {
		return false
	}

//确保光标。首先在关闭的Tx上返回false和nil键/值。
	if cursor.First() {
		tc.t.Error("Cursor.First: claims ok on closed tx")
		return false
	}
	if cursor.Key() != nil || cursor.Value() != nil {
		tc.t.Error("Cursor.First: key and/or value are not nil on " +
			"closed tx")
		return false
	}

//确保cursor.last在关闭的tx上返回false和nil键/值。
	if cursor.Last() {
		tc.t.Error("Cursor.Last: claims ok on closed tx")
		return false
	}
	if cursor.Key() != nil || cursor.Value() != nil {
		tc.t.Error("Cursor.Last: key and/or value are not nil on " +
			"closed tx")
		return false
	}

//确保cursor.next在关闭的tx上返回false和nil键/值。
	if cursor.Next() {
		tc.t.Error("Cursor.Next: claims ok on closed tx")
		return false
	}
	if cursor.Key() != nil || cursor.Value() != nil {
		tc.t.Error("Cursor.Next: key and/or value are not nil on " +
			"closed tx")
		return false
	}

//确保cursor.prev在关闭的tx上返回false和nil键/值。
	if cursor.Prev() {
		tc.t.Error("Cursor.Prev: claims ok on closed tx")
		return false
	}
	if cursor.Key() != nil || cursor.Value() != nil {
		tc.t.Error("Cursor.Prev: key and/or value are not nil on " +
			"closed tx")
		return false
	}

//确保cursor.seek在关闭的tx上返回false和nil键/值。
	if cursor.Seek([]byte{}) {
		tc.t.Error("Cursor.Seek: claims ok on closed tx")
		return false
	}
	if cursor.Key() != nil || cursor.Value() != nil {
		tc.t.Error("Cursor.Seek: key and/or value are not nil on " +
			"closed tx")
		return false
	}

//-----------------
//非批量块IO API
//-----------------

//一次测试一个块API，以确保它们
//返回预期的错误。此外，构建测试
//循环时在下面批量API。
	allBlockHashes := make([]chainhash.Hash, len(tc.blocks))
	allBlockRegions := make([]database.BlockRegion, len(tc.blocks))
	for i, block := range tc.blocks {
		blockHash := block.Hash()
		allBlockHashes[i] = *blockHash

		txLocs, err := block.TxLoc()
		if err != nil {
			tc.t.Errorf("block.TxLoc(%d): unexpected error: %v", i,
				err)
			return false
		}

//确保storeblock返回预期错误。
		testName = "StoreBlock on closed tx"
		err = tx.StoreBlock(block)
		if !checkDbError(tc.t, testName, err, wantErrCode) {
			return false
		}

//确保FetchBlock返回预期错误。
		testName = fmt.Sprintf("FetchBlock #%d on closed tx", i)
		_, err = tx.FetchBlock(blockHash)
		if !checkDbError(tc.t, testName, err, wantErrCode) {
			return false
		}

//确保FetchBlockHeader返回预期错误。
		testName = fmt.Sprintf("FetchBlockHeader #%d on closed tx", i)
		_, err = tx.FetchBlockHeader(blockHash)
		if !checkDbError(tc.t, testName, err, wantErrCode) {
			return false
		}

//确保第一个事务作为块区域从
//数据库返回预期的错误。
		region := database.BlockRegion{
			Hash:   blockHash,
			Offset: uint32(txLocs[0].TxStart),
			Len:    uint32(txLocs[0].TxLen),
		}
		allBlockRegions[i] = region
		_, err = tx.FetchBlockRegion(&region)
		if !checkDbError(tc.t, testName, err, wantErrCode) {
			return false
		}

//确保HasBlock返回预期错误。
		testName = fmt.Sprintf("HasBlock #%d on closed tx", i)
		_, err = tx.HasBlock(blockHash)
		if !checkDbError(tc.t, testName, err, wantErrCode) {
			return false
		}
	}

//------------
//批量块IO API
//------------

//确保FetchBlocks返回预期错误。
	testName = "FetchBlocks on closed tx"
	_, err = tx.FetchBlocks(allBlockHashes)
	if !checkDbError(tc.t, testName, err, wantErrCode) {
		return false
	}

//确保FETCHAMBESTHOMER返回预期错误。
	testName = "FetchBlockHeaders on closed tx"
	_, err = tx.FetchBlockHeaders(allBlockHashes)
	if !checkDbError(tc.t, testName, err, wantErrCode) {
		return false
	}

//确保FetchBlockRegions返回预期错误。
	testName = "FetchBlockRegions on closed tx"
	_, err = tx.FetchBlockRegions(allBlockRegions)
	if !checkDbError(tc.t, testName, err, wantErrCode) {
		return false
	}

//确保HasBlocks返回预期错误。
	testName = "HasBlocks on closed tx"
	_, err = tx.HasBlocks(allBlockHashes)
	if !checkDbError(tc.t, testName, err, wantErrCode) {
		return false
	}

//----------------
//提交/回滚
//----------------

//确保尝试回滚或提交的事务
//已关闭返回预期的错误。
	err = tx.Rollback()
	if !checkDbError(tc.t, "closed tx rollback", err, wantErrCode) {
		return false
	}
	err = tx.Commit()
	return checkDbError(tc.t, "closed tx commit", err, wantErrCode)
}

//testxtclosed确保元数据和块IO API函数的行为
//对只读和读写进行尝试时如预期
//交易。
func testTxClosed(tc *testContext) bool {
	bucketName := []byte("closedtxbucket")
	keyName := []byte("closedtxkey")

//启动事务，创建用于测试的bucket和key，以及
//立即对其执行提交操作，以使其关闭。
	tx, err := tc.db.Begin(true)
	if err != nil {
		tc.t.Errorf("Begin(true): unexpected error: %v", err)
		return false
	}
	defer rollbackOnPanic(tc.t, tx)
	if _, err := tx.Metadata().CreateBucket(bucketName); err != nil {
		tc.t.Errorf("CreateBucket: unexpected error: %v", err)
		return false
	}
	if err := tx.Metadata().Put(keyName, []byte("test")); err != nil {
		tc.t.Errorf("Put: unexpected error: %v", err)
		return false
	}
	if err := tx.Commit(); err != nil {
		tc.t.Errorf("Commit: unexpected error: %v", err)
		return false
	}

//确保在关闭的读写上调用所有函数
//事务按预期运行。
	if !testClosedTxInterface(tc, tx) {
		return false
	}

//使用回滚的只读事务重复测试。
	tx, err = tc.db.Begin(false)
	if err != nil {
		tc.t.Errorf("Begin(false): unexpected error: %v", err)
		return false
	}
	defer rollbackOnPanic(tc.t, tx)
	if err := tx.Rollback(); err != nil {
		tc.t.Errorf("Rollback: unexpected error: %v", err)
		return false
	}

//确保在关闭的只读文件上调用所有函数
//事务按预期运行。
	return testClosedTxInterface(tc, tx)
}

//testconcurrency确保数据库正确支持并发读卡器和
//只有一个作家。它还确保视图在当时充当快照。
//它们是后天习得的。
func testConcurrecy(tc *testContext) bool {
//睡眠时间是每一个同时阅读的读者应该睡多久。
//帮助检测数据是否真正被读取
//同时地。它以一个健全的下界开始。
	var sleepTime = time.Millisecond * 250

//确定单块读取需要多长时间。当它的时候
//比默认的最小睡眠时间长，请将睡眠时间调整为
//有助于防止持续时间太短而导致错误
//在速度较慢的系统上测试失败。
	startTime := time.Now()
	err := tc.db.View(func(tx database.Tx) error {
		_, err := tx.FetchBlock(tc.blocks[0].Hash())
		return err
	})
	if err != nil {
		tc.t.Errorf("Unexpected error in view: %v", err)
		return false
	}
	elapsed := time.Since(startTime)
	if sleepTime < elapsed {
		sleepTime = elapsed
	}
	tc.t.Logf("Time to load block 0: %v, using sleep time: %v", elapsed,
		sleepTime)

//读卡器接收要加载的块号，并通过通道返回结果
//的操作。下面使用它来启动多个并发
//读者。
	numReaders := len(tc.blocks)
	resultChan := make(chan bool, numReaders)
	reader := func(blockNum int) {
		err := tc.db.View(func(tx database.Tx) error {
			time.Sleep(sleepTime)
			_, err := tx.FetchBlock(tc.blocks[blockNum].Hash())
			return err
		})
		if err != nil {
			tc.t.Errorf("Unexpected error in concurrent view: %v",
				err)
			resultChan <- false
		}
		resultChan <- true
	}

//为同一块启动多个并发读卡器并等待
//结果。
	startTime = time.Now()
	for i := 0; i < numReaders; i++ {
		go reader(0)
	}
	for i := 0; i < numReaders; i++ {
		if result := <-resultChan; !result {
			return false
		}
	}
	elapsed = time.Since(startTime)
	tc.t.Logf("%d concurrent reads of same block elapsed: %v", numReaders,
		elapsed)

//如果花费的时间超过一半，就认为是失败。
//不带并发性。
	if elapsed > sleepTime*time.Duration(numReaders/2) {
		tc.t.Errorf("Concurrent views for same block did not appear to "+
			"run simultaneously: elapsed %v", elapsed)
		return false
	}

//为不同的块启动多个并发读卡器并等待
//结果。
	startTime = time.Now()
	for i := 0; i < numReaders; i++ {
		go reader(i)
	}
	for i := 0; i < numReaders; i++ {
		if result := <-resultChan; !result {
			return false
		}
	}
	elapsed = time.Since(startTime)
	tc.t.Logf("%d concurrent reads of different blocks elapsed: %v",
		numReaders, elapsed)

//如果花费的时间超过一半，就认为是失败。
//不带并发性。
	if elapsed > sleepTime*time.Duration(numReaders/2) {
		tc.t.Errorf("Concurrent views for different blocks did not "+
			"appear to run simultaneously: elapsed %v", elapsed)
		return false
	}

//启动一些阅读器，等待它们获取视图。各
//读卡器等待编写器发出的信号完成，以确保
//视图看不到由编写器编写的数据，因为
//在设置数据之前启动。
	concurrentKey := []byte("notthere")
	concurrentVal := []byte("someval")
	started := make(chan struct{})
	writeComplete := make(chan struct{})
	reader = func(blockNum int) {
		err := tc.db.View(func(tx database.Tx) error {
			started <- struct{}{}

//等待编写器完成。
			<-writeComplete

//因为这个读卡器是在写入之前创建的
//位置，它添加的数据不应可见。
			val := tx.Metadata().Get(concurrentKey)
			if val != nil {
				return fmt.Errorf("%s should not be visible",
					concurrentKey)
			}
			return nil
		})
		if err != nil {
			tc.t.Errorf("Unexpected error in concurrent view: %v",
				err)
			resultChan <- false
		}
		resultChan <- true
	}
	for i := 0; i < numReaders; i++ {
		go reader(0)
	}
	for i := 0; i < numReaders; i++ {
		<-started
	}

//所有读卡器都已启动并等待编写器完成。
//设置一些读卡器期望找不到的数据，并向
//读卡器通过关闭WriteComplete通道完成写入。
	err = tc.db.Update(func(tx database.Tx) error {
		return tx.Metadata().Put(concurrentKey, concurrentVal)
	})
	if err != nil {
		tc.t.Errorf("Unexpected error in update: %v", err)
		return false
	}
	close(writeComplete)

//等待读卡器结果。
	for i := 0; i < numReaders; i++ {
		if result := <-resultChan; !result {
			return false
		}
	}

//启动一些编写器，并确保总时间至少为
//写入休眠时间*numWriters。这确保只有一个写事务
//可以一次激活。
	writeSleepTime := time.Millisecond * 250
	writer := func() {
		err := tc.db.Update(func(tx database.Tx) error {
			time.Sleep(writeSleepTime)
			return nil
		})
		if err != nil {
			tc.t.Errorf("Unexpected error in concurrent view: %v",
				err)
			resultChan <- false
		}
		resultChan <- true
	}
	numWriters := 3
	startTime = time.Now()
	for i := 0; i < numWriters; i++ {
		go writer()
	}
	for i := 0; i < numWriters; i++ {
		if result := <-resultChan; !result {
			return false
		}
	}
	elapsed = time.Since(startTime)
	tc.t.Logf("%d concurrent writers elapsed using sleep time %v: %v",
		numWriters, writeSleepTime, elapsed)

//总时间必须至少是所有睡眠的总和，如果
//写入被正确阻止。
	if elapsed < writeSleepTime*time.Duration(numWriters) {
		tc.t.Errorf("Concurrent writes appeared to run simultaneously: "+
			"elapsed %v", elapsed)
		return false
	}

	return true
}

//testconcurrentclose确保使用打开的事务关闭数据库
//在事务完成之前阻止。
//
//从该函数返回时，数据库将关闭。
func testConcurrentClose(tc *testContext) bool {
//启动一些阅读器，等待它们获取视图。各
//读卡器等待信号完成以确保事务保持
//打开，直到明确指示关闭。
	var activeReaders int32
	numReaders := 3
	started := make(chan struct{})
	finishReaders := make(chan struct{})
	resultChan := make(chan bool, numReaders+1)
	reader := func() {
		err := tc.db.View(func(tx database.Tx) error {
			atomic.AddInt32(&activeReaders, 1)
			started <- struct{}{}
			<-finishReaders
			atomic.AddInt32(&activeReaders, -1)
			return nil
		})
		if err != nil {
			tc.t.Errorf("Unexpected error in concurrent view: %v",
				err)
			resultChan <- false
		}
		resultChan <- true
	}
	for i := 0; i < numReaders; i++ {
		go reader()
	}
	for i := 0; i < numReaders; i++ {
		<-started
	}

//在单独的goroutine中关闭数据库。这应该阻止到
//交易完成。一旦达成交易，
//关闭dbclosed通道以向下面的主goroutine发出信号。
	dbClosed := make(chan struct{})
	go func() {
		started <- struct{}{}
		err := tc.db.Close()
		if err != nil {
			tc.t.Errorf("Unexpected error in concurrent view: %v",
				err)
			resultChan <- false
		}
		close(dbClosed)
		resultChan <- true
	}()
	<-started

//等待一小段时间，然后向读卡器事务发送信号
//完成。当接收到数据库关闭通道时，确保没有
//活动读卡器打开。
	time.AfterFunc(time.Millisecond*250, func() { close(finishReaders) })
	<-dbClosed
	if nr := atomic.LoadInt32(&activeReaders); nr != 0 {
		tc.t.Errorf("Close did not appear to block with active "+
			"readers: %d active", nr)
		return false
	}

//等待所有结果。
	for i := 0; i < numReaders+1; i++ {
		if result := <-resultChan; !result {
			return false
		}
	}

	return true
}

//TestInterface测试为数据库的各种接口执行测试
//需要给定数据库类型的数据库中的状态的包。
func testInterface(t *testing.T, db database.DB) {
//创建要传递的测试上下文。
	context := testContext{t: t, db: db}

//加载测试块并存储在测试上下文中，以便在整个过程中使用
//测试。
	blocks, err := loadBlocks(t, blockDataFile, blockDataNet)
	if err != nil {
		t.Errorf("loadBlocks: Unexpected error: %v", err)
		return
	}
	context.blocks = blocks

//测试事务元数据接口，包括托管和手动
//交易和存储桶。
	if !testMetadataTxInterface(&context) {
		return
	}

//使用托管和手动测试事务块IO接口
//交易。此函数将所有存储的块保留在
//数据库，因为以后会用到它们。
	if !testBlockIOTxInterface(&context) {
		return
	}

//针对已关闭的
//按预期进行事务处理。
	if !testTxClosed(&context) {
		return
	}

//测试数据库是否支持并发性。
	if !testConcurrecy(&context) {
		return
	}

//测试使用打开的事务块关闭数据库直到
//交易完成。
//
//从该函数返回时将关闭数据库，因此
//一定是最后一件事了。
	testConcurrentClose(&context)
}
