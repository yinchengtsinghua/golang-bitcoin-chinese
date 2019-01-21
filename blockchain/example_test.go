
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2014-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package blockchain_test

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/database"
	_ "github.com/btcsuite/btcd/database/ffldb"
	"github.com/btcsuite/btcutil"
)

//这个例子演示了如何创建一个新的链实例并使用
//processBlock尝试向链中添加块。作为包装
//概述文件描述，这包括所有比特币共识
//规则。这个例子故意尝试插入一个复制的Genesis
//块以说明如何处理无效块。
func ExampleBlockChain_ProcessBlock() {
//创建一个新的数据库来存储接受的块。典型地
//这将打开现有数据库，不会删除
//创建一个这样的新数据库，但在这里完成了，所以这是
//完整的工作示例，不留临时文件
//周围。
	dbPath := filepath.Join(os.TempDir(), "exampleprocessblock")
	_ = os.RemoveAll(dbPath)
	db, err := database.Create("ffldb", dbPath, chaincfg.MainNetParams.Net)
	if err != nil {
		fmt.Printf("Failed to create database: %v\n", err)
		return
	}
	defer os.RemoveAll(dbPath)
	defer db.Close()

//使用基础数据库创建新的区块链实例
//主要比特币网络。此示例不演示
//其他可用配置选项，如指定
//通知回调和签名缓存。另外，打电话的人会
//通常保留对中间时间源的引用并添加时间
//从网络上的其他对等点获取的值，因此本地时间为
//调整为与其他同行一致。
	chain, err := blockchain.New(&blockchain.Config{
		DB:          db,
		ChainParams: &chaincfg.MainNetParams,
		TimeSource:  blockchain.NewMedianTime(),
	})
	if err != nil {
		fmt.Printf("Failed to create chain instance: %v\n", err)
		return
	}

//处理一个块。对于这个例子，我们打算
//通过尝试处理已经
//存在。
	genesisBlock := btcutil.NewBlock(chaincfg.MainNetParams.GenesisBlock)
	isMainChain, isOrphan, err := chain.ProcessBlock(genesisBlock,
		blockchain.BFNone)
	if err != nil {
		fmt.Printf("Failed to process block: %v\n", err)
		return
	}
	fmt.Printf("Block accepted. Is it on the main chain?: %v", isMainChain)
	fmt.Printf("Block accepted. Is it an orphan?: %v", isOrphan)

//输出：
//未能处理块：已经有块000000000019D6689C085AE165831E934F763AE46A2A6C172B3F1B60A8CE26F
}

//此示例演示如何转换块头中的压缩“位”
//它将目标难度表示为一个大整数，并使用
//典型的十六进制符号。
func ExampleCompactToBig() {
//转换主块链中块300000的位。
	bits := uint32(419465580)
	targetDifficulty := blockchain.CompactToBig(bits)

//以十六进制显示。
	fmt.Printf("%064x\n", targetDifficulty.Bytes())

//输出：
//000000000000000896c000000000000000000000000000000000000000000000000万
}

//这个例子演示了如何将目标难度转换为紧凑型
//块头中表示目标难度的“位”。
func ExampleBigToCompact() {
//从主块300000块转换目标难度
//链条要紧凑。
	t := "0000000000000000896c00000000000000000000000000000000000000000000"
	targetDifficulty, success := new(big.Int).SetString(t, 16)
	if !success {
		fmt.Println("invalid target difficulty")
		return
	}
	bits := blockchain.BigToCompact(targetDifficulty)

	fmt.Println(bits)

//输出：
//四亿一千九百四十六万五千五百八十
}
