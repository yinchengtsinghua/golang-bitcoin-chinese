
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
	"os"
	"path/filepath"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcutil"
)

//BenchmarkBlockHeader基准测试加载主网Genesis需要多长时间
//块标题。
func BenchmarkBlockHeader(b *testing.B) {
//首先创建一个新的数据库，并用mainnet填充它
//创世纪大厦
	dbPath := filepath.Join(os.TempDir(), "ffldb-benchblkhdr")
	_ = os.RemoveAll(dbPath)
	db, err := database.Create("ffldb", dbPath, blockDataNet)
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dbPath)
	defer db.Close()
	err = db.Update(func(tx database.Tx) error {
		block := btcutil.NewBlock(chaincfg.MainNetParams.GenesisBlock)
		return tx.StoreBlock(block)
	})
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	err = db.View(func(tx database.Tx) error {
		blockHash := chaincfg.MainNetParams.GenesisHash
		for i := 0; i < b.N; i++ {
			_, err := tx.FetchBlockHeader(blockHash)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		b.Fatal(err)
	}

//不要以拆卸为基准。
	b.StopTimer()
}

//BenchmarkBlockHeader基准测试加载主网Genesis需要多长时间
//块。
func BenchmarkBlock(b *testing.B) {
//首先创建一个新的数据库，并用mainnet填充它
//创世纪大厦
	dbPath := filepath.Join(os.TempDir(), "ffldb-benchblk")
	_ = os.RemoveAll(dbPath)
	db, err := database.Create("ffldb", dbPath, blockDataNet)
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dbPath)
	defer db.Close()
	err = db.Update(func(tx database.Tx) error {
		block := btcutil.NewBlock(chaincfg.MainNetParams.GenesisBlock)
		return tx.StoreBlock(block)
	})
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	err = db.View(func(tx database.Tx) error {
		blockHash := chaincfg.MainNetParams.GenesisHash
		for i := 0; i < b.N; i++ {
			_, err := tx.FetchBlock(blockHash)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		b.Fatal(err)
	}

//不要以拆卸为基准。
	b.StopTimer()
}
