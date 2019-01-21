
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2016版权所有
//版权所有（c）2016-2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package blockchain_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/blockchain/fullblocktests"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/database"
	_ "github.com/btcsuite/btcd/database/ffldb"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

const (
//testdbtype是用于测试的数据库后端类型。
	testDbType = "ffldb"

//testdbroot是用于创建所有测试数据库的根目录。
	testDbRoot = "testdbs"

//blockdatanet是测试块数据中的预期网络。
	blockDataNet = wire.MainNet
)

//filesexists返回命名文件或目录是否存在。
func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

//issupporteddbtype返回传递的数据库类型是否为
//当前支持。
func isSupportedDbType(dbType string) bool {
	supportedDrivers := database.SupportedDrivers()
	for _, driver := range supportedDrivers {
		if dbType == driver {
			return true
		}
	}

	return false
}

//chainsetup用于创建一个新的数据库和具有genesis的链实例。
//块已插入。除了新的链实例外，它还返回
//调用方在完成清理测试后应调用的拆卸函数。
func chainSetup(dbName string, params *chaincfg.Params) (*blockchain.BlockChain, func(), error) {
	if !isSupportedDbType(testDbType) {
		return nil, nil, fmt.Errorf("unsupported db type %v", testDbType)
	}

//专门处理内存数据库，因为它不需要磁盘
//具体处理。
	var db database.DB
	var teardown func()
	if testDbType == "memdb" {
		ndb, err := database.Create(testDbType)
		if err != nil {
			return nil, nil, fmt.Errorf("error creating db: %v", err)
		}
		db = ndb

//设置拆卸功能以进行清理。这个功能是
//返回给调用方，以便在测试完成后调用。
		teardown = func() {
			db.Close()
		}
	} else {
//为测试数据库创建根目录。
		if !fileExists(testDbRoot) {
			if err := os.MkdirAll(testDbRoot, 0700); err != nil {
				err := fmt.Errorf("unable to create test db "+
					"root: %v", err)
				return nil, nil, err
			}
		}

//创建一个新的数据库来存储接受的块。
		dbPath := filepath.Join(testDbRoot, dbName)
		_ = os.RemoveAll(dbPath)
		ndb, err := database.Create(testDbType, dbPath, blockDataNet)
		if err != nil {
			return nil, nil, fmt.Errorf("error creating db: %v", err)
		}
		db = ndb

//设置拆卸功能以进行清理。这个功能是
//返回给调用方，以便在测试完成后调用。
		teardown = func() {
			db.Close()
			os.RemoveAll(dbPath)
			os.RemoveAll(testDbRoot)
		}
	}

//复制链参数以确保测试对其所做的任何修改
//链参数不影响全局实例。
	paramsCopy := *params

//创建主链实例。
	chain, err := blockchain.New(&blockchain.Config{
		DB:          db,
		ChainParams: &paramsCopy,
		Checkpoints: nil,
		TimeSource:  blockchain.NewMedianTime(),
		SigCache:    txscript.NewSigCache(1000),
	})
	if err != nil {
		teardown()
		err := fmt.Errorf("failed to create chain instance: %v", err)
		return nil, nil, err
	}
	return chain, teardown, nil
}

//TestFullBlocks确保由FullBlockTests包生成的所有测试
//通过ProcessBlock处理时具有预期结果。
func TestFullBlocks(t *testing.T) {
	tests, err := fullblocktests.Generate(false)
	if err != nil {
		t.Fatalf("failed to generate tests: %v", err)
	}

//创建一个新的数据库和链实例来运行测试。
	chain, teardownFunc, err := chainSetup("fullblocktest",
		&chaincfg.RegressionNetParams)
	if err != nil {
		t.Errorf("Failed to setup chain instance: %v", err)
		return
	}
	defer teardownFunc()

//TestAcceptedBlock尝试在提供的测试中处理块
//实例并确保它是根据标志接受的
//在测试中指定。
	testAcceptedBlock := func(item fullblocktests.AcceptedBlock) {
		blockHeight := item.Height
		block := btcutil.NewBlock(item.Block)
		block.SetHeight(blockHeight)
		t.Logf("Testing block %s (hash %s, height %d)",
			item.Name, block.Hash(), blockHeight)

		isMainChain, isOrphan, err := chain.ProcessBlock(block,
			blockchain.BFNone)
		if err != nil {
			t.Fatalf("block %q (hash %s, height %d) should "+
				"have been accepted: %v", item.Name,
				block.Hash(), blockHeight, err)
		}

//确保主链和孤立标志与值匹配
//在测试中指定。
		if isMainChain != item.IsMainChain {
			t.Fatalf("block %q (hash %s, height %d) unexpected main "+
				"chain flag -- got %v, want %v", item.Name,
				block.Hash(), blockHeight, isMainChain,
				item.IsMainChain)
		}
		if isOrphan != item.IsOrphan {
			t.Fatalf("block %q (hash %s, height %d) unexpected "+
				"orphan flag -- got %v, want %v", item.Name,
				block.Hash(), blockHeight, isOrphan,
				item.IsOrphan)
		}
	}

//testRejectedBlock尝试在提供的测试中处理块
//实例并确保使用拒绝代码拒绝它
//在测试中指定。
	testRejectedBlock := func(item fullblocktests.RejectedBlock) {
		blockHeight := item.Height
		block := btcutil.NewBlock(item.Block)
		block.SetHeight(blockHeight)
		t.Logf("Testing block %s (hash %s, height %d)",
			item.Name, block.Hash(), blockHeight)

		_, _, err := chain.ProcessBlock(block, blockchain.BFNone)
		if err == nil {
			t.Fatalf("block %q (hash %s, height %d) should not "+
				"have been accepted", item.Name, block.Hash(),
				blockHeight)
		}

//确保错误代码为预期类型，并拒绝
//
		rerr, ok := err.(blockchain.RuleError)
		if !ok {
			t.Fatalf("block %q (hash %s, height %d) returned "+
				"unexpected error type -- got %T, want "+
				"blockchain.RuleError", item.Name, block.Hash(),
				blockHeight, err)
		}
		if rerr.ErrorCode != item.RejectCode {
			t.Fatalf("block %q (hash %s, height %d) does not have "+
				"expected reject code -- got %v, want %v",
				item.Name, block.Hash(), blockHeight,
				rerr.ErrorCode, item.RejectCode)
		}
	}

//
//
//消息错误。
	testRejectedNonCanonicalBlock := func(item fullblocktests.RejectedNonCanonicalBlock) {
		headerLen := len(item.RawBlock)
		if headerLen > 80 {
			headerLen = 80
		}
		blockHash := chainhash.DoubleHashH(item.RawBlock[0:headerLen])
		blockHeight := item.Height
		t.Logf("Testing block %s (hash %s, height %d)", item.Name,
			blockHash, blockHeight)

//确保反序列化块时出错。
		var msgBlock wire.MsgBlock
		err := msgBlock.BtcDecode(bytes.NewReader(item.RawBlock), 0, wire.BaseEncoding)
		if _, ok := err.(*wire.MessageError); !ok {
			t.Fatalf("block %q (hash %s, height %d) should have "+
				"failed to decode", item.Name, blockHash,
				blockHeight)
		}
	}

//testOrphanorRejectedBlock尝试在
//提供测试实例并确保它被接受为
//孤立或因违反规则而被拒绝。
	testOrphanOrRejectedBlock := func(item fullblocktests.OrphanOrRejectedBlock) {
		blockHeight := item.Height
		block := btcutil.NewBlock(item.Block)
		block.SetHeight(blockHeight)
		t.Logf("Testing block %s (hash %s, height %d)",
			item.Name, block.Hash(), blockHeight)

		_, isOrphan, err := chain.ProcessBlock(block, blockchain.BFNone)
		if err != nil {
//确保错误代码是预期类型。
			if _, ok := err.(blockchain.RuleError); !ok {
				t.Fatalf("block %q (hash %s, height %d) "+
					"returned unexpected error type -- "+
					"got %T, want blockchain.RuleError",
					item.Name, block.Hash(), blockHeight,
					err)
			}
		}

		if !isOrphan {
			t.Fatalf("block %q (hash %s, height %d) was accepted, "+
				"but is not considered an orphan", item.Name,
				block.Hash(), blockHeight)
		}
	}

//testExpectedTip确保区块链的当前提示是
//在提供的测试实例中指定的块。
	testExpectedTip := func(item fullblocktests.ExpectedTip) {
		blockHeight := item.Height
		block := btcutil.NewBlock(item.Block)
		block.SetHeight(blockHeight)
		t.Logf("Testing tip for block %s (hash %s, height %d)",
			item.Name, block.Hash(), blockHeight)

//确保哈希和高度匹配。
		best := chain.BestSnapshot()
		if best.Hash != item.Block.BlockHash() ||
			best.Height != blockHeight {

			t.Fatalf("block %q (hash %s, height %d) should be "+
				"the current tip -- got (hash %s, height %d)",
				item.Name, block.Hash(), blockHeight, best.Hash,
				best.Height)
		}
	}

	for testNum, test := range tests {
		for itemNum, item := range test {
			switch item := item.(type) {
			case fullblocktests.AcceptedBlock:
				testAcceptedBlock(item)
			case fullblocktests.RejectedBlock:
				testRejectedBlock(item)
			case fullblocktests.RejectedNonCanonicalBlock:
				testRejectedNonCanonicalBlock(item)
			case fullblocktests.OrphanOrRejectedBlock:
				testOrphanOrRejectedBlock(item)
			case fullblocktests.ExpectedTip:
				testExpectedTip(item)
			default:
				t.Fatalf("test #%d, item #%d is not one of "+
					"the supported test instance types -- "+
					"got type: %T", testNum, itemNum, item)
			}
		}
	}
}
