
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

package database_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/database"
	_ "github.com/btcsuite/btcd/database/ffldb"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

//此示例演示如何创建新数据库。
func ExampleCreate() {
//此示例假定导入了ffldb驱动程序。
//
//进口（
//“github.com/btcsuite/btcd/数据库”
//“github.com/btcsuite/btcd/数据库/ffldb”
//）

//创建一个数据库，并安排它在退出时关闭和删除。
//通常情况下，您不希望像这样立即删除数据库
//这个，也不放在temp目录中，但是在这里这样做是为了确保
//该示例将在自身之后进行清理。
	dbPath := filepath.Join(os.TempDir(), "examplecreate")
	db, err := database.Create("ffldb", dbPath, wire.MainNet)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer os.RemoveAll(dbPath)
	defer db.Close()

//输出：
}

//此示例演示如何创建新数据库并使用托管
//用于存储和检索元数据的读写事务。
func Example_basicUsage() {
//此示例假定导入了ffldb驱动程序。
//
//进口（
//“github.com/btcsuite/btcd/数据库”
//“github.com/btcsuite/btcd/数据库/ffldb”
//）

//创建一个数据库，并安排它在退出时关闭和删除。
//通常情况下，您不希望像这样立即删除数据库
//这个，也不放在temp目录中，但是在这里这样做是为了确保
//该示例将在自身之后进行清理。
	dbPath := filepath.Join(os.TempDir(), "exampleusage")
	db, err := database.Create("ffldb", dbPath, wire.MainNet)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer os.RemoveAll(dbPath)
	defer db.Close()

//使用数据库的更新函数执行托管
//读写事务。事务将自动滚动
//返回所提供的内部函数是否返回非零错误。
	err = db.Update(func(tx database.Tx) error {
//将键/值对直接存储在元数据桶中。
//通常，一个嵌套的bucket将用于一个给定的特性，
//但是这个示例直接使用元数据桶
//简单。
		key := []byte("mykey")
		value := []byte("myvalue")
		if err := tx.Metadata().Put(key, value); err != nil {
			return err
		}

//把钥匙读回来，确保它匹配。
		if !bytes.Equal(tx.Metadata().Get(key), value) {
			return fmt.Errorf("unexpected value for key '%s'", key)
		}

//在元数据桶下创建一个新的嵌套桶。
		nestedBucketKey := []byte("mybucket")
		nestedBucket, err := tx.Metadata().CreateBucket(nestedBucketKey)
		if err != nil {
			return err
		}

//上面在元数据存储桶中设置的键
//此新嵌套存储桶中不存在。
		if nestedBucket.Get(key) != nil {
			return fmt.Errorf("key '%s' is not expected nil", key)
		}

		return nil
	})
	if err != nil {
		fmt.Println(err)
		return
	}

//输出：
}

//此示例演示如何使用托管读写创建新数据库
//要存储块的事务，并使用托管只读事务
//取出块。
func Example_blockStorageAndRetrieval() {
//此示例假定导入了ffldb驱动程序。
//
//进口（
//“github.com/btcsuite/btcd/数据库”
//“github.com/btcsuite/btcd/数据库/ffldb”
//）

//创建一个数据库，并安排它在退出时关闭和删除。
//通常情况下，您不希望像这样立即删除数据库
//这个，也不放在temp目录中，但是在这里这样做是为了确保
//该示例将在自身之后进行清理。
	dbPath := filepath.Join(os.TempDir(), "exampleblkstorage")
	db, err := database.Create("ffldb", dbPath, wire.MainNet)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer os.RemoveAll(dbPath)
	defer db.Close()

//使用数据库的更新函数执行托管
//读写事务并将Genesis块存储在数据库中
//并举例说明。
	err = db.Update(func(tx database.Tx) error {
		genesisBlock := chaincfg.MainNetParams.GenesisBlock
		return tx.StoreBlock(btcutil.NewBlock(genesisBlock))
	})
	if err != nil {
		fmt.Println(err)
		return
	}

//使用数据库的View函数执行托管只读
//并获取存储在上面的块。
	var loadedBlockBytes []byte
	err = db.Update(func(tx database.Tx) error {
		genesisHash := chaincfg.MainNetParams.GenesisHash
		blockBytes, err := tx.FetchBlock(genesisHash)
		if err != nil {
			return err
		}

//如文档所述，从数据库中提取的所有数据仅
//在数据库事务期间有效，以支持
//零拷贝后端。因此，复制数据，以便
//可以在事务外部使用。
		loadedBlockBytes = make([]byte, len(blockBytes))
		copy(loadedBlockBytes, blockBytes)
		return nil
	})
	if err != nil {
		fmt.Println(err)
		return
	}

//通常在此时，块可以通过
//Wire.msgBlock.Deserialize函数或以其序列化形式使用
//视需要而定。但是，对于本例，只显示
//显示按预期加载的序列化字节数。
	fmt.Printf("Serialized block size: %d bytes\n", len(loadedBlockBytes))

//输出：
//序列化块大小：285字节
}
