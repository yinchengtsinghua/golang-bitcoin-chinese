
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

//此文件是FFLDB包的一部分，而不是作为
//它提供了白盒测试。

package ffldb

import (
	"compress/bzip2"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/goleveldb/leveldb"
	ldberrors "github.com/btcsuite/goleveldb/leveldb/errors"
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
	t            *testing.T
	db           database.DB
	files        map[uint32]*lockableFile
	maxFileSizes map[uint32]int64
	blocks       []*btcutil.Block
}

//testconverterr确保leveldb错误到数据库错误的转换工作正常
//果不其然。
func TestConvertErr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err         error
		wantErrCode database.ErrorCode
	}{
		{&ldberrors.ErrCorrupted{}, database.ErrCorruption},
		{leveldb.ErrClosed, database.ErrDbNotOpen},
		{leveldb.ErrSnapshotReleased, database.ErrTxClosed},
		{leveldb.ErrIterReleased, database.ErrTxClosed},
	}

	for i, test := range tests {
		gotErr := convertErr("test", test.err)
		if gotErr.ErrorCode != test.wantErrCode {
			t.Errorf("convertErr #%d unexpected error - got %v, "+
				"want %v", i, gotErr.ErrorCode, test.wantErrCode)
			continue
		}
	}
}

//TestCornerCases确保打开时可能发生的几个角情况
//数据库和/或块文件按预期工作。
func TestCornerCases(t *testing.T) {
	t.Parallel()

//在DATAPASE路径中创建一个文件以强制下面的打开失败。
	dbPath := filepath.Join(os.TempDir(), "ffldb-errors")
	_ = os.RemoveAll(dbPath)
	fi, err := os.Create(dbPath)
	if err != nil {
		t.Errorf("os.Create: unexpected error: %v", err)
		return
	}
	fi.Close()

//确保当文件存在于
//需要目录。
	testName := "openDB: fail due to file at target location"
	wantErrCode := database.ErrDriverSpecific
	idb, err := openDB(dbPath, blockDataNet, true)
	if !checkDbError(t, testName, err, wantErrCode) {
		if err == nil {
			idb.Close()
		}
		_ = os.RemoveAll(dbPath)
		return
	}

//删除该文件并创建用于运行测试的数据库。它
//这次应该成功。
	_ = os.RemoveAll(dbPath)
	idb, err = openDB(dbPath, blockDataNet, true)
	if err != nil {
		t.Errorf("openDB: unexpected error: %v", err)
		return
	}
	defer os.RemoveAll(dbPath)
	defer idb.Close()

//确保尝试写入无法创建的文件时返回
//预期的错误。
	testName = "writeBlock: open file failure"
	filePath := blockFilePath(dbPath, 0)
	if err := os.Mkdir(filePath, 0755); err != nil {
		t.Errorf("os.Mkdir: unexpected error: %v", err)
		return
	}
	store := idb.(*db).store
	_, err = store.writeBlock([]byte{0x00})
	if !checkDbError(t, testName, err, database.ErrDriverSpecific) {
		return
	}
	_ = os.RemoveAll(filePath)

//从数据库下关闭基础级别数据库。
	ldb := idb.(*db).cache.ldb
	ldb.Close()

//确保基础数据库中的初始化错误作为
//预期。
	testName = "initDB: reinitialization"
	wantErrCode = database.ErrDbNotOpen
	err = initDB(ldb)
	if !checkDbError(t, testName, err, wantErrCode) {
		return
	}

//确保视图处理基础级别数据库中的错误
//适当地。
	testName = "View: underlying leveldb error"
	wantErrCode = database.ErrDbNotOpen
	err = idb.View(func(tx database.Tx) error {
		return nil
	})
	if !checkDbError(t, testName, err, wantErrCode) {
		return
	}

//确保更新处理基础级别数据库中的错误
//适当地。
	testName = "Update: underlying leveldb error"
	err = idb.Update(func(tx database.Tx) error {
		return nil
	})
	if !checkDbError(t, testName, err, wantErrCode) {
		return
	}
}

//ResetDatabase从与
//测试上下文，包括所有元数据和模拟文件。
func resetDatabase(tc *testContext) bool {
//重置元数据。
	err := tc.db.Update(func(tx database.Tx) error {
//使用光标删除所有键，同时生成
//桶列表。前臂时取下钥匙是不安全的
//在光标期间删除存储桶也不安全
//迭代，所以需要这种双重方法。
		var bucketNames [][]byte
		cursor := tx.Metadata().Cursor()
		for ok := cursor.First(); ok; ok = cursor.Next() {
			if cursor.Value() != nil {
				if err := cursor.Delete(); err != nil {
					return err
				}
			} else {
				bucketNames = append(bucketNames, cursor.Key())
			}
		}

//拆下铲斗。
		for _, k := range bucketNames {
			if err := tx.Metadata().DeleteBucket(k); err != nil {
				return err
			}
		}

		_, err := tx.Metadata().CreateBucket(blockIdxBucketName)
		return err
	})
	if err != nil {
		tc.t.Errorf("Update: unexpected error: %v", err)
		return false
	}

//重置模拟文件。
	store := tc.db.(*db).store
	wc := store.writeCursor
	wc.curFile.Lock()
	if wc.curFile.file != nil {
		wc.curFile.file.Close()
		wc.curFile.file = nil
	}
	wc.curFile.Unlock()
	wc.Lock()
	wc.curFileNum = 0
	wc.curOffset = 0
	wc.Unlock()
	tc.files = make(map[uint32]*lockableFile)
	tc.maxFileSizes = make(map[uint32]int64)
	return true
}

//TestWriteFailures在写入块时测试各种失败路径
//文件夹。
func testWriteFailures(tc *testContext) bool {
	if !resetDatabase(tc) {
		return false
	}

//确保刷新期间发生文件同步错误，返回预期错误。
	store := tc.db.(*db).store
	testName := "flush: file sync failure"
	store.writeCursor.Lock()
	oldFile := store.writeCursor.curFile
	store.writeCursor.curFile = &lockableFile{
		file: &mockFile{forceSyncErr: true, maxSize: -1},
	}
	store.writeCursor.Unlock()
	err := tc.db.(*db).cache.flush()
	if !checkDbError(tc.t, testName, err, database.ErrDriverSpecific) {
		return false
	}
	store.writeCursor.Lock()
	store.writeCursor.curFile = oldFile
	store.writeCursor.Unlock()

//通过使用在写入数据时强制在各种错误路径中出错
//最大大小有限的模拟文件。
	block0Bytes, _ := tc.blocks[0].Bytes()
	tests := []struct {
		fileNum uint32
		maxSize int64
	}{
//写入网络字节时强制出错。
		{fileNum: 0, maxSize: 2},

//写入块大小时强制出错。
		{fileNum: 0, maxSize: 6},

//写入块时强制出错。
		{fileNum: 0, maxSize: 17},

//写入校验和时强制出错。
		{fileNum: 0, maxSize: int64(len(block0Bytes)) + 10},

//在为force multiple写入足够的块后强制出错
//文件夹。
		{fileNum: 15, maxSize: 1},
	}

	for i, test := range tests {
		if !resetDatabase(tc) {
			return false
		}

//确保使用模拟存储指定数量的块
//当事务为
//提交，而不是存储块时。
		tc.maxFileSizes = map[uint32]int64{test.fileNum: test.maxSize}
		err := tc.db.Update(func(tx database.Tx) error {
			for i, block := range tc.blocks {
				err := tx.StoreBlock(block)
				if err != nil {
					tc.t.Errorf("StoreBlock (%d): unexpected "+
						"error: %v", i, err)
					return errSubTestFail
				}
			}

			return nil
		})
		testName := fmt.Sprintf("Force update commit failure - test "+
			"%d, fileNum %d, maxsize %d", i, test.fileNum,
			test.maxSize)
		if !checkDbError(tc.t, testName, err, database.ErrDriverSpecific) {
			tc.t.Errorf("%v", err)
			return false
		}

//确保提交回滚删除了所有额外的文件和数据。
		if len(tc.files) != 1 {
			tc.t.Errorf("Update rollback: new not removed - want "+
				"1 file, got %d", len(tc.files))
			return false
		}
		if _, ok := tc.files[0]; !ok {
			tc.t.Error("Update rollback: file 0 does not exist")
			return false
		}
		file := tc.files[0].file.(*mockFile)
		if len(file.data) != 0 {
			tc.t.Errorf("Update rollback: file did not truncate - "+
				"want len 0, got len %d", len(file.data))
			return false
		}
	}

	return true
}

//testblockfileerrors确保数据库返回预期的错误，其中
//与文件相关的问题，如关闭和丢失的文件。
func testBlockFileErrors(tc *testContext) bool {
	if !resetDatabase(tc) {
		return false
	}

//当请求无效文件时，确保blockfile和openfile中有错误
//数字。
	store := tc.db.(*db).store
	testName := "blockFile invalid file open"
	_, err := store.blockFile(^uint32(0))
	if !checkDbError(tc.t, testName, err, database.ErrDriverSpecific) {
		return false
	}
	testName = "openFile invalid file open"
	_, err = store.openFile(^uint32(0))
	if !checkDbError(tc.t, testName, err, database.ErrDriverSpecific) {
		return false
	}

//将第一个块插入模拟文件。
	err = tc.db.Update(func(tx database.Tx) error {
		err := tx.StoreBlock(tc.blocks[0])
		if err != nil {
			tc.t.Errorf("StoreBlock: unexpected error: %v", err)
			return errSubTestFail
		}

		return nil
	})
	if err != nil {
		if err != errSubTestFail {
			tc.t.Errorf("Update: unexpected error: %v", err)
		}
		return false
	}

//请求文件时确保readblock和readblockregion中存在错误
//不存在的数字。
	block0Hash := tc.blocks[0].Hash()
	testName = "readBlock invalid file number"
	invalidLoc := blockLocation{
		blockFileNum: ^uint32(0),
		blockLen:     80,
	}
	_, err = store.readBlock(block0Hash, invalidLoc)
	if !checkDbError(tc.t, testName, err, database.ErrDriverSpecific) {
		return false
	}
	testName = "readBlockRegion invalid file number"
	_, err = store.readBlockRegion(invalidLoc, 0, 80)
	if !checkDbError(tc.t, testName, err, database.ErrDriverSpecific) {
		return false
	}

//关闭数据库下的块文件。
	store.writeCursor.curFile.Lock()
	store.writeCursor.curFile.file.Close()
	store.writeCursor.curFile.Unlock()

//确保fetchblock和fetchblockregion中的失败，因为
//他们需要读取的基础文件已关闭。
	err = tc.db.View(func(tx database.Tx) error {
		testName = "FetchBlock closed file"
		wantErrCode := database.ErrDriverSpecific
		_, err := tx.FetchBlock(block0Hash)
		if !checkDbError(tc.t, testName, err, wantErrCode) {
			return errSubTestFail
		}

		testName = "FetchBlockRegion closed file"
		regions := []database.BlockRegion{
			{
				Hash:   block0Hash,
				Len:    80,
				Offset: 0,
			},
		}
		_, err = tx.FetchBlockRegion(&regions[0])
		if !checkDbError(tc.t, testName, err, wantErrCode) {
			return errSubTestFail
		}

		testName = "FetchBlockRegions closed file"
		_, err = tx.FetchBlockRegions(regions)
		if !checkDbError(tc.t, testName, err, wantErrCode) {
			return errSubTestFail
		}

		return nil
	})
	if err != nil {
		if err != errSubTestFail {
			tc.t.Errorf("View: unexpected error: %v", err)
		}
		return false
	}

	return true
}

//testcorruption确保数据库在
//腐败场景。
func testCorruption(tc *testContext) bool {
	if !resetDatabase(tc) {
		return false
	}

//将第一个块插入模拟文件。
	err := tc.db.Update(func(tx database.Tx) error {
		err := tx.StoreBlock(tc.blocks[0])
		if err != nil {
			tc.t.Errorf("StoreBlock: unexpected error: %v", err)
			return errSubTestFail
		}

		return nil
	})
	if err != nil {
		if err != errSubTestFail {
			tc.t.Errorf("Update: unexpected error: %v", err)
		}
		return false
	}

//通过有意修改字节来确保检测到损坏
//存储到模拟文件并读取块。
	block0Bytes, _ := tc.blocks[0].Bytes()
	block0Hash := tc.blocks[0].Hash()
	tests := []struct {
		offset      uint32
		fixChecksum bool
		wantErrCode database.ErrorCode
	}{
//网络字节之一。校验和需要修复，因此
//检测到无效网络。
		{2, true, database.ErrDriverSpecific},

//相同的网络字节，但这次不修复校验和
//以确保检测到损坏。
		{2, false, database.ErrCorruption},

//块长度字节之一。
		{6, false, database.ErrCorruption},

//随机头字节。
		{17, false, database.ErrCorruption},

//随机事务字节。
		{90, false, database.ErrCorruption},

//随机校验和字节。
		{uint32(len(block0Bytes)) + 10, false, database.ErrCorruption},
	}
	err = tc.db.View(func(tx database.Tx) error {
		data := tc.files[0].file.(*mockFile).data
		for i, test := range tests {
//将偏移量处的字节损坏一位。
			data[test.offset] ^= 0x10

//如果要求强制执行其他错误，请修复校验和。
			fileLen := len(data)
			var oldChecksumBytes [4]byte
			copy(oldChecksumBytes[:], data[fileLen-4:])
			if test.fixChecksum {
				toSum := data[:fileLen-4]
				cksum := crc32.Checksum(toSum, castagnoli)
				binary.BigEndian.PutUint32(data[fileLen-4:], cksum)
			}

			testName := fmt.Sprintf("FetchBlock (test #%d): "+
				"corruption", i)
			_, err := tx.FetchBlock(block0Hash)
			if !checkDbError(tc.t, testName, err, test.wantErrCode) {
				return errSubTestFail
			}

//将损坏的数据重置回原始数据。
			data[test.offset] ^= 0x10
			if test.fixChecksum {
				copy(data[fileLen-4:], oldChecksumBytes[:])
			}
		}

		return nil
	})
	if err != nil {
		if err != errSubTestFail {
			tc.t.Errorf("View: unexpected error: %v", err)
		}
		return false
	}

	return true
}

//testfailureScenarios确保几个失败场景，如数据库
//处理损坏、块文件写入失败和回滚失败
//正确地。
func TestFailureScenarios(t *testing.T) {
//创建新数据库以运行测试。
	dbPath := filepath.Join(os.TempDir(), "ffldb-failurescenarios")
	_ = os.RemoveAll(dbPath)
	idb, err := database.Create(dbType, dbPath, blockDataNet)
	if err != nil {
		t.Errorf("Failed to create test database (%s) %v", dbType, err)
		return
	}
	defer os.RemoveAll(dbPath)
	defer idb.Close()

//创建要传递的测试上下文。
	tc := &testContext{
		t:            t,
		db:           idb,
		files:        make(map[uint32]*lockableFile),
		maxFileSizes: make(map[uint32]int64),
	}

//将最大文件大小更改为小值以强制多个平面
//使用测试数据集的文件并替换与文件相关的函数
//利用内存中的模拟文件。这允许注射
//各种与文件相关的错误。
	store := idb.(*db).store
store.maxBlockFileSize = 1024 //1Kib
	store.openWriteFileFunc = func(fileNum uint32) (filer, error) {
		if file, ok := tc.files[fileNum]; ok {
//“重新打开”文件。
			file.Lock()
			mock := file.file.(*mockFile)
			mock.Lock()
			mock.closed = false
			mock.Unlock()
			file.Unlock()
			return mock, nil
		}

//按照测试中的指定限制模拟文件的最大大小
//语境。
		maxSize := int64(-1)
		if maxFileSize, ok := tc.maxFileSizes[fileNum]; ok {
			maxSize = int64(maxFileSize)
		}
		file := &mockFile{maxSize: int64(maxSize)}
		tc.files[fileNum] = &lockableFile{file: file}
		return file, nil
	}
	store.openFileFunc = func(fileNum uint32) (*lockableFile, error) {
//尝试打开max file num时强制出错。
		if fileNum == ^uint32(0) {
			return nil, makeDbErr(database.ErrDriverSpecific,
				"test", nil)
		}
		if file, ok := tc.files[fileNum]; ok {
//“重新打开”文件。
			file.Lock()
			mock := file.file.(*mockFile)
			mock.Lock()
			mock.closed = false
			mock.Unlock()
			file.Unlock()
			return file, nil
		}
		file := &lockableFile{file: &mockFile{}}
		tc.files[fileNum] = file
		return file, nil
	}
	store.deleteFileFunc = func(fileNum uint32) error {
		if file, ok := tc.files[fileNum]; ok {
			file.Lock()
			file.file.Close()
			file.Unlock()
			delete(tc.files, fileNum)
			return nil
		}

		str := fmt.Sprintf("file %d does not exist", fileNum)
		return makeDbErr(database.ErrDriverSpecific, str, nil)
	}

//加载测试块并保存在测试上下文中以便在整个过程中使用
//测试。
	blocks, err := loadBlocks(t, blockDataFile, blockDataNet)
	if err != nil {
		t.Errorf("loadBlocks: Unexpected error: %v", err)
		return
	}
	tc.blocks = blocks

//写入块文件时测试各种故障路径。
	if !testWriteFailures(tc) {
		return
	}

//测试与文件相关的各种问题，如关闭和丢失的文件。
	if !testBlockFileErrors(tc) {
		return
	}

//测试各种损坏情况。
	testCorruption(tc)
}
