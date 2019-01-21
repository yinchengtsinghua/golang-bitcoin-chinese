
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

package ffldb_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcd/database/ffldb"
	"github.com/btcsuite/btcutil"
)

//dbtype是此驱动程序的数据库类型名称。
const dbType = "ffldb"

//testcreateopenfail确保与创建和打开
//数据库处理正确。
func TestCreateOpenFail(t *testing.T) {
	t.Parallel()

//确保尝试打开不存在的数据库时返回
//预期的错误。
	wantErrCode := database.ErrDbDoesNotExist
	_, err := database.Open(dbType, "noexist", blockDataNet)
	if !checkDbError(t, "Open", err, wantErrCode) {
		return
	}

//确保尝试打开的数据库的编号不正确
//参数返回预期的错误。
	wantErr := fmt.Errorf("invalid arguments to %s.Open -- expected "+
		"database path and block network", dbType)
	_, err = database.Open(dbType, 1, 2, 3)
	if err.Error() != wantErr.Error() {
		t.Errorf("Open: did not receive expected error - got %v, "+
			"want %v", err, wantErr)
		return
	}

//确保尝试打开的数据库类型无效
//第一个参数返回预期的错误。
	wantErr = fmt.Errorf("first argument to %s.Open is invalid -- "+
		"expected database path string", dbType)
	_, err = database.Open(dbType, 1, blockDataNet)
	if err.Error() != wantErr.Error() {
		t.Errorf("Open: did not receive expected error - got %v, "+
			"want %v", err, wantErr)
		return
	}

//确保尝试打开的数据库类型无效
//第二个参数返回预期的错误。
	wantErr = fmt.Errorf("second argument to %s.Open is invalid -- "+
		"expected block network", dbType)
	_, err = database.Open(dbType, "noexist", "invalid")
	if err.Error() != wantErr.Error() {
		t.Errorf("Open: did not receive expected error - got %v, "+
			"want %v", err, wantErr)
		return
	}

//确保尝试创建的数据库的编号不正确
//参数返回预期的错误。
	wantErr = fmt.Errorf("invalid arguments to %s.Create -- expected "+
		"database path and block network", dbType)
	_, err = database.Create(dbType, 1, 2, 3)
	if err.Error() != wantErr.Error() {
		t.Errorf("Create: did not receive expected error - got %v, "+
			"want %v", err, wantErr)
		return
	}

//确保尝试为创建类型无效的数据库
//第一个参数返回预期的错误。
	wantErr = fmt.Errorf("first argument to %s.Create is invalid -- "+
		"expected database path string", dbType)
	_, err = database.Create(dbType, 1, blockDataNet)
	if err.Error() != wantErr.Error() {
		t.Errorf("Create: did not receive expected error - got %v, "+
			"want %v", err, wantErr)
		return
	}

//确保尝试为创建类型无效的数据库
//第二个参数返回预期的错误。
	wantErr = fmt.Errorf("second argument to %s.Create is invalid -- "+
		"expected block network", dbType)
	_, err = database.Create(dbType, "noexist", "invalid")
	if err.Error() != wantErr.Error() {
		t.Errorf("Create: did not receive expected error - got %v, "+
			"want %v", err, wantErr)
		return
	}

//确保对已关闭数据库的操作返回预期的
//错误。
	dbPath := filepath.Join(os.TempDir(), "ffldb-createfail")
	_ = os.RemoveAll(dbPath)
	db, err := database.Create(dbType, dbPath, blockDataNet)
	if err != nil {
		t.Errorf("Create: unexpected error: %v", err)
		return
	}
	defer os.RemoveAll(dbPath)
	db.Close()

	wantErrCode = database.ErrDbNotOpen
	err = db.View(func(tx database.Tx) error {
		return nil
	})
	if !checkDbError(t, "View", err, wantErrCode) {
		return
	}

	wantErrCode = database.ErrDbNotOpen
	err = db.Update(func(tx database.Tx) error {
		return nil
	})
	if !checkDbError(t, "Update", err, wantErrCode) {
		return
	}

	wantErrCode = database.ErrDbNotOpen
	_, err = db.Begin(false)
	if !checkDbError(t, "Begin(false)", err, wantErrCode) {
		return
	}

	wantErrCode = database.ErrDbNotOpen
	_, err = db.Begin(true)
	if !checkDbError(t, "Begin(true)", err, wantErrCode) {
		return
	}

	wantErrCode = database.ErrDbNotOpen
	err = db.Close()
	if !checkDbError(t, "Close", err, wantErrCode) {
		return
	}
}

//TestPersistence确保存储的值在关闭后仍然有效，并且
//重新打开数据库。
func TestPersistence(t *testing.T) {
	t.Parallel()

//创建新数据库以运行测试。
	dbPath := filepath.Join(os.TempDir(), "ffldb-persistencetest")
	_ = os.RemoveAll(dbPath)
	db, err := database.Create(dbType, dbPath, blockDataNet)
	if err != nil {
		t.Errorf("Failed to create test database (%s) %v", dbType, err)
		return
	}
	defer os.RemoveAll(dbPath)
	defer db.Close()

//创建一个bucket，将一些值放入其中，并存储一个块，以便
//可以在重新打开时测试是否存在。
	bucket1Key := []byte("bucket1")
	storeValues := map[string]string{
		"b1key1": "foo1",
		"b1key2": "foo2",
		"b1key3": "foo3",
	}
	genesisBlock := btcutil.NewBlock(chaincfg.MainNetParams.GenesisBlock)
	genesisHash := chaincfg.MainNetParams.GenesisHash
	err = db.Update(func(tx database.Tx) error {
		metadataBucket := tx.Metadata()
		if metadataBucket == nil {
			return fmt.Errorf("Metadata: unexpected nil bucket")
		}

		bucket1, err := metadataBucket.CreateBucket(bucket1Key)
		if err != nil {
			return fmt.Errorf("CreateBucket: unexpected error: %v",
				err)
		}

		for k, v := range storeValues {
			err := bucket1.Put([]byte(k), []byte(v))
			if err != nil {
				return fmt.Errorf("Put: unexpected error: %v",
					err)
			}
		}

		if err := tx.StoreBlock(genesisBlock); err != nil {
			return fmt.Errorf("StoreBlock: unexpected error: %v",
				err)
		}

		return nil
	})
	if err != nil {
		t.Errorf("Update: unexpected error: %v", err)
		return
	}

//关闭并重新打开数据库以确保值保持不变。
	db.Close()
	db, err = database.Open(dbType, dbPath, blockDataNet)
	if err != nil {
		t.Errorf("Failed to open test database (%s) %v", dbType, err)
		return
	}
	defer db.Close()

//确保之前存储在第三个命名空间中的值仍然存在
//并且是正确的。
	err = db.View(func(tx database.Tx) error {
		metadataBucket := tx.Metadata()
		if metadataBucket == nil {
			return fmt.Errorf("Metadata: unexpected nil bucket")
		}

		bucket1 := metadataBucket.Bucket(bucket1Key)
		if bucket1 == nil {
			return fmt.Errorf("Bucket1: unexpected nil bucket")
		}

		for k, v := range storeValues {
			gotVal := bucket1.Get([]byte(k))
			if !reflect.DeepEqual(gotVal, []byte(v)) {
				return fmt.Errorf("Get: key '%s' does not "+
					"match expected value - got %s, want %s",
					k, gotVal, v)
			}
		}

		genesisBlockBytes, _ := genesisBlock.Bytes()
		gotBytes, err := tx.FetchBlock(genesisHash)
		if err != nil {
			return fmt.Errorf("FetchBlock: unexpected error: %v",
				err)
		}
		if !reflect.DeepEqual(gotBytes, genesisBlockBytes) {
			return fmt.Errorf("FetchBlock: stored block mismatch")
		}

		return nil
	})
	if err != nil {
		t.Errorf("View: unexpected error: %v", err)
		return
	}
}

//testinterface执行此数据库驱动程序的所有接口测试。
func TestInterface(t *testing.T) {
	t.Parallel()

//创建新数据库以运行测试。
	dbPath := filepath.Join(os.TempDir(), "ffldb-interfacetest")
	_ = os.RemoveAll(dbPath)
	db, err := database.Create(dbType, dbPath, blockDataNet)
	if err != nil {
		t.Errorf("Failed to create test database (%s) %v", dbType, err)
		return
	}
	defer os.RemoveAll(dbPath)
	defer db.Close()

//确保驱动程序类型为预期值。
	gotDbType := db.Type()
	if gotDbType != dbType {
		t.Errorf("Type: unepxected driver type - got %v, want %v",
			gotDbType, dbType)
		return
	}

//对数据库运行所有接口测试。
	runtime.GOMAXPROCS(runtime.NumCPU())

//将最大文件大小更改为小值以强制多个平面
//包含测试数据集的文件。
	ffldb.TstRunWithMaxBlockFileSize(db, 2048, func() {
		testInterface(t, db)
	})
}
