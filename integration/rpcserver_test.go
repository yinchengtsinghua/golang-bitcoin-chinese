
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

//由于以下生成标记，在常规测试期间忽略此文件。
//+建立RPCTEST

package integration

import (
	"bytes"
	"fmt"
	"os"
	"runtime/debug"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/integration/rpctest"
)

func testGetBestBlock(r *rpctest.Harness, t *testing.T) {
	_, prevbestHeight, err := r.Node.GetBestBlock()
	if err != nil {
		t.Fatalf("Call to `getbestblock` failed: %v", err)
	}

//创建一个连接到当前提示的新块。
	generatedBlockHashes, err := r.Node.Generate(1)
	if err != nil {
		t.Fatalf("Unable to generate block: %v", err)
	}

	bestHash, bestHeight, err := r.Node.GetBestBlock()
	if err != nil {
		t.Fatalf("Call to `getbestblock` failed: %v", err)
	}

//哈希应与新提交的块相同。
	if !bytes.Equal(bestHash[:], generatedBlockHashes[0][:]) {
		t.Fatalf("Block hashes do not match. Returned hash %v, wanted "+
			"hash %v", bestHash, generatedBlockHashes[0][:])
	}

//块高度现在应反映最新高度。
	if bestHeight != prevbestHeight+1 {
		t.Fatalf("Block heights do not match. Got %v, wanted %v",
			bestHeight, prevbestHeight+1)
	}
}

func testGetBlockCount(r *rpctest.Harness, t *testing.T) {
//保存当前计数。
	currentCount, err := r.Node.GetBlockCount()
	if err != nil {
		t.Fatalf("Unable to get block count: %v", err)
	}

	if _, err := r.Node.Generate(1); err != nil {
		t.Fatalf("Unable to generate block: %v", err)
	}

//计数应该增加一。
	newCount, err := r.Node.GetBlockCount()
	if err != nil {
		t.Fatalf("Unable to get block count: %v", err)
	}
	if newCount != currentCount+1 {
		t.Fatalf("Block count incorrect. Got %v should be %v",
			newCount, currentCount+1)
	}
}

func testGetBlockHash(r *rpctest.Harness, t *testing.T) {
//创建一个连接到当前提示的新块。
	generatedBlockHashes, err := r.Node.Generate(1)
	if err != nil {
		t.Fatalf("Unable to generate block: %v", err)
	}

	info, err := r.Node.GetInfo()
	if err != nil {
		t.Fatalf("call to getinfo cailed: %v", err)
	}

	blockHash, err := r.Node.GetBlockHash(int64(info.Blocks))
	if err != nil {
		t.Fatalf("Call to `getblockhash` failed: %v", err)
	}

//块哈希应与新创建的块匹配。
	if !bytes.Equal(generatedBlockHashes[0][:], blockHash[:]) {
		t.Fatalf("Block hashes do not match. Returned hash %v, wanted "+
			"hash %v", blockHash, generatedBlockHashes[0][:])
	}
}

var rpcTestCases = []rpctest.HarnessTestCase{
	testGetBestBlock,
	testGetBlockCount,
	testGetBlockHash,
}

var primaryHarness *rpctest.Harness

func TestMain(m *testing.M) {
	var err error

//为了像在Mainnet上一样正确地测试场景，
//确保不接受非标准交易
//内存池或中继。
	btcdCfg := []string{"--rejectnonstd"}
	primaryHarness, err = rpctest.New(&chaincfg.SimNetParams, nil, btcdCfg)
	if err != nil {
		fmt.Println("unable to create primary harness: ", err)
		os.Exit(1)
	}

//用长度为125的链初始化主挖掘节点，
//提供25个成熟的CoinBase，用于测试
//目的。
	if err := primaryHarness.SetUp(true, 25); err != nil {
		fmt.Println("unable to setup test chain: ", err)

//即使线束没有完全安装，它仍然需要
//拆除以确保所有资源，如临时
//目录被清除。错误是故意的
//忽略，因为这已经是一个错误路径，没有其他内容
//不管怎样都可以解决。
		_ = primaryHarness.TearDown()
		os.Exit(1)
	}

	exitCode := m.Run()

//清除当前仍在运行的所有活动线束。此
//包括删除所有临时目录和关闭
//已创建进程。
	if err := rpctest.TearDownAll(); err != nil {
		fmt.Println("unable to tear down all harnesses: ", err)
		os.Exit(1)
	}

	os.Exit(exitCode)
}

func TestRpcServer(t *testing.T) {
	var currentTestNum int
	defer func() {
//如果其中一个集成测试在主系统中引起了恐慌
//Goroutine，然后拆下所有的安全带以避免
//任何泄漏的BTCD过程。
		if r := recover(); r != nil {
			fmt.Println("recovering from test panic: ", r)
			if err := rpctest.TearDownAll(); err != nil {
				fmt.Println("unable to tear down all harnesses: ", err)
			}
			t.Fatalf("test #%v panicked: %s", currentTestNum, debug.Stack())
		}
	}()

	for _, testCase := range rpcTestCases {
		testCase(primaryHarness, t)

		currentTestNum++
	}
}
