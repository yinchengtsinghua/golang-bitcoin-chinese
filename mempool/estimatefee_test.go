
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

package mempool

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/mining"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

//newtestfeestimator创建具有一些不同参数的feeestimator
//用于测试。
func newTestFeeEstimator(binSize, maxReplacements, maxRollback uint32) *FeeEstimator {
	return &FeeEstimator{
		maxRollback:         maxRollback,
		lastKnownHeight:     0,
		binSize:             int32(binSize),
		minRegisteredBlocks: 0,
		maxReplacements:     int32(maxReplacements),
		observed:            make(map[chainhash.Hash]*observedTransaction),
		dropped:             make([]*registeredBlock, 0, maxRollback),
	}
}

//LastBlock是块哈希的链接列表，
//由检测仪处理。
type lastBlock struct {
	hash *chainhash.Hash
	prev *lastBlock
}

//EstimateFeeter与FeeeStimator交互以跟踪
//它的预期状态。
type estimateFeeTester struct {
	ef      *FeeEstimator
	t       *testing.T
	version int32
	height  int32
	last    *lastBlock
}

func (eft *estimateFeeTester) testTx(fee btcutil.Amount) *TxDesc {
	eft.version++
	return &TxDesc{
		TxDesc: mining.TxDesc{
			Tx: btcutil.NewTx(&wire.MsgTx{
				Version: eft.version,
			}),
			Height: eft.height,
			Fee:    int64(fee),
		},
		StartingPriority: 0,
	}
}

func expectedFeePerKilobyte(t *TxDesc) BtcPerKilobyte {
	size := float64(t.TxDesc.Tx.MsgTx().SerializeSize())
	fee := float64(t.TxDesc.Fee)

	return SatoshiPerByte(fee / size).ToBtcPerKb()
}

func (eft *estimateFeeTester) newBlock(txs []*wire.MsgTx) {
	eft.height++

	block := btcutil.NewBlock(&wire.MsgBlock{
		Transactions: txs,
	})
	block.SetHeight(eft.height)

	eft.last = &lastBlock{block.Hash(), eft.last}

	eft.ef.RegisterBlock(block)
}

func (eft *estimateFeeTester) rollback() {
	if eft.last == nil {
		return
	}

	err := eft.ef.Rollback(eft.last.hash)

	if err != nil {
		eft.t.Errorf("Could not rollback: %v", err)
	}

	eft.height--
	eft.last = eft.last.prev
}

//testEstimateFee测试feeestimator中的基本功能。
func TestEstimateFee(t *testing.T) {
	ef := newTestFeeEstimator(5, 3, 1)
	eft := estimateFeeTester{ef: ef, t: t}

//尝试不使用TXS，所有查询均为零。
	expected := BtcPerKilobyte(0.0)
	for i := uint32(1); i <= estimateFeeDepth; i++ {
		estimated, _ := ef.EstimateFee(i)

		if estimated != expected {
			t.Errorf("Estimate fee error: expected %f when estimator is empty; got %f", expected, estimated)
		}
	}

//现在插入一个Tx。
	tx := eft.testTx(1000000)
	ef.ObserveTransaction(tx)

//预期值仍应为零，因为它仍在mempool中。
	expected = BtcPerKilobyte(0.0)
	for i := uint32(1); i <= estimateFeeDepth; i++ {
		estimated, _ := ef.EstimateFee(i)

		if estimated != expected {
			t.Errorf("Estimate fee error: expected %f when estimator has one tx in mempool; got %f", expected, estimated)
		}
	}

//Change minRegisteredBlocks to make sure that works. 错误返回
//期望值。
	ef.minRegisteredBlocks = 1
	expected = BtcPerKilobyte(-1.0)
	for i := uint32(1); i <= estimateFeeDepth; i++ {
		estimated, _ := ef.EstimateFee(i)

		if estimated != expected {
			t.Errorf("Estimate fee error: expected %f before any blocks have been registered; got %f", expected, estimated)
		}
	}

//用新的Tx记录一个数据块。
	eft.newBlock([]*wire.MsgTx{tx.Tx.MsgTx()})
	expected = expectedFeePerKilobyte(tx)
	for i := uint32(1); i <= estimateFeeDepth; i++ {
		estimated, _ := ef.EstimateFee(i)

		if estimated != expected {
			t.Errorf("Estimate fee error: expected %f when one tx is binned; got %f", expected, estimated)
		}
	}

//回滚最后一个块；这是一个孤立块。
	ef.minRegisteredBlocks = 0
	eft.rollback()
	expected = BtcPerKilobyte(0.0)
	for i := uint32(1); i <= estimateFeeDepth; i++ {
		estimated, _ := ef.EstimateFee(i)

		if estimated != expected {
			t.Errorf("Estimate fee error: expected %f after rolling back block; got %f", expected, estimated)
		}
	}

//记录一个空块，然后用新的Tx记录一个块。
//这个测试是因为一个只有在
//第一个箱子中没有交易记录。
	eft.newBlock([]*wire.MsgTx{})
	eft.newBlock([]*wire.MsgTx{tx.Tx.MsgTx()})
	expected = expectedFeePerKilobyte(tx)
	for i := uint32(1); i <= estimateFeeDepth; i++ {
		estimated, _ := ef.EstimateFee(i)

		if estimated != expected {
			t.Errorf("Estimate fee error: expected %f when one tx is binned; got %f", expected, estimated)
		}
	}

//创建更多事务。
	txA := eft.testTx(500000)
	txB := eft.testTx(2000000)
	txC := eft.testTx(4000000)
	ef.ObserveTransaction(txA)
	ef.ObserveTransaction(txB)
	ef.ObserveTransaction(txC)

//记录7个空块。
	for i := 0; i < 7; i++ {
		eft.newBlock([]*wire.MsgTx{})
	}

//我的第一个德克萨斯州。
	eft.newBlock([]*wire.MsgTx{txA.Tx.MsgTx()})

//现在估计的金额应该取决于价值
//估计费用的论据。
	for i := uint32(1); i <= estimateFeeDepth; i++ {
		estimated, _ := ef.EstimateFee(i)
		if i > 2 {
			expected = expectedFeePerKilobyte(txA)
		} else {
			expected = expectedFeePerKilobyte(tx)
		}
		if estimated != expected {
			t.Errorf("Estimate fee error: expected %f on round %d; got %f", expected, i, estimated)
		}
	}

//再记录5个空块。
	for i := 0; i < 5; i++ {
		eft.newBlock([]*wire.MsgTx{})
	}

//我的下一个德克萨斯州。
	eft.newBlock([]*wire.MsgTx{txB.Tx.MsgTx()})

//现在估计的金额应该取决于价值
//估计费用的论据。
	for i := uint32(1); i <= estimateFeeDepth; i++ {
		estimated, _ := ef.EstimateFee(i)
		if i <= 2 {
			expected = expectedFeePerKilobyte(txB)
		} else if i <= 8 {
			expected = expectedFeePerKilobyte(tx)
		} else {
			expected = expectedFeePerKilobyte(txA)
		}

		if estimated != expected {
			t.Errorf("Estimate fee error: expected %f on round %d; got %f", expected, i, estimated)
		}
	}

//再记录9个空块。
	for i := 0; i < 10; i++ {
		eft.newBlock([]*wire.MsgTx{})
	}

//矿山TXC
	eft.newBlock([]*wire.MsgTx{txC.Tx.MsgTx()})

//这对结果应该没有影响，因为
//为记录TXC，已开采了许多区块。
	for i := uint32(1); i <= estimateFeeDepth; i++ {
		estimated, _ := ef.EstimateFee(i)
		if i <= 2 {
			expected = expectedFeePerKilobyte(txC)
		} else if i <= 8 {
			expected = expectedFeePerKilobyte(txB)
		} else if i <= 8+6 {
			expected = expectedFeePerKilobyte(tx)
		} else {
			expected = expectedFeePerKilobyte(txA)
		}

		if estimated != expected {
			t.Errorf("Estimate fee error: expected %f on round %d; got %f", expected, i, estimated)
		}
	}
}

func (eft *estimateFeeTester) estimates() [estimateFeeDepth]BtcPerKilobyte {

//生成估计
	var estimates [estimateFeeDepth]BtcPerKilobyte
	for i := 0; i < estimateFeeDepth; i++ {
		estimates[i], _ = eft.ef.EstimateFee(uint32(i + 1))
	}

//检查所有估计费用结果是否按降序排列。
	for i := 1; i < estimateFeeDepth; i++ {
		if estimates[i] > estimates[i-1] {
			eft.t.Error("Estimates not in descending order; got ",
				estimates[i], " for estimate ", i, " and ", estimates[i-1], " for ", (i - 1))
			panic("invalid state.")
		}
	}

	return estimates
}

func (eft *estimateFeeTester) round(txHistory [][]*TxDesc,
	estimateHistory [][estimateFeeDepth]BtcPerKilobyte,
	txPerRound, txPerBlock uint32) ([][]*TxDesc, [][estimateFeeDepth]BtcPerKilobyte) {

//generate new txs.
	var newTxs []*TxDesc
	for i := uint32(0); i < txPerRound; i++ {
		newTx := eft.testTx(btcutil.Amount(rand.Intn(1000000)))
		eft.ef.ObserveTransaction(newTx)
		newTxs = append(newTxs, newTx)
	}

//生成内存池。
	mempool := make(map[*observedTransaction]*TxDesc)
	for _, h := range txHistory {
		for _, t := range h {
			if o, exists := eft.ef.observed[*t.Tx.Hash()]; exists && o.mined == mining.UnminedHeight {
				mempool[o] = t
			}
		}
	}

//生成新块，没有重复项。
	i := uint32(0)
	newBlockList := make([]*wire.MsgTx, 0, txPerBlock)
	for _, t := range mempool {
		newBlockList = append(newBlockList, t.TxDesc.Tx.MsgTx())
		i++

		if i == txPerBlock {
			break
		}
	}

//注册新块。
	eft.newBlock(newBlockList)

//返回结果。
	estimates := eft.estimates()

//返回结果
	return append(txHistory, newTxs), append(estimateHistory, estimates)
}

//testEstimateFerrollback测试回滚函数，该函数撤消
//添加新块的效果。
func TestEstimateFeeRollback(t *testing.T) {
	txPerRound := uint32(7)
	txPerBlock := uint32(5)
	binSize := uint32(6)
	maxReplacements := uint32(4)
	stepsBack := 2
	rounds := 30

	eft := estimateFeeTester{ef: newTestFeeEstimator(binSize, maxReplacements, uint32(stepsBack)), t: t}
	var txHistory [][]*TxDesc
	estimateHistory := [][estimateFeeDepth]BtcPerKilobyte{eft.estimates()}

	for round := 0; round < rounds; round++ {
//向前走几圈。
		for step := 0; step <= stepsBack; step++ {
			txHistory, estimateHistory =
				eft.round(txHistory, estimateHistory, txPerRound, txPerBlock)
		}

//现在回去。
		for step := 0; step < stepsBack; step++ {
			eft.rollback()

//在回滚之后，我们应该有相同的估计
//以前一样收费。
			expected := estimateHistory[len(estimateHistory)-step-2]
			estimates := eft.estimates()

//确保两者相同。
			for i := 0; i < estimateFeeDepth; i++ {
				if expected[i] != estimates[i] {
					t.Errorf("Rollback value mismatch. Expected %f, got %f. ",
						expected[i], estimates[i])
					return
				}
			}
		}

//抹去历史。
		txHistory = txHistory[0 : len(txHistory)-stepsBack]
		estimateHistory = estimateHistory[0 : len(estimateHistory)-stepsBack]
	}
}

func (eft *estimateFeeTester) checkSaveAndRestore(
	previousEstimates [estimateFeeDepth]BtcPerKilobyte) {

//获取保存状态。
	save := eft.ef.Save()

//保存并还原数据库。
	var err error
	eft.ef, err = RestoreFeeEstimator(save)
	if err != nil {
		eft.t.Fatalf("Could not restore database: %s", err)
	}

//Save again and check that it matches the previous one.
	redo := eft.ef.Save()
	if !bytes.Equal(save, redo) {
		eft.t.Fatalf("Restored states do not match: %v %v", save, redo)
	}

//检查结果是否匹配。
	newEstimates := eft.estimates()

	for i, prev := range previousEstimates {
		if prev != newEstimates[i] {
			eft.t.Error("Mismatch in estimate ", i, " after restore; got ", newEstimates[i], " but expected ", prev)
		}
	}
}

//testsave测试保存并还原为[]字节。
func TestDatabase(t *testing.T) {

	txPerRound := uint32(7)
	txPerBlock := uint32(5)
	binSize := uint32(6)
	maxReplacements := uint32(4)
	rounds := 8

	eft := estimateFeeTester{ef: newTestFeeEstimator(binSize, maxReplacements, uint32(rounds)+1), t: t}
	var txHistory [][]*TxDesc
	estimateHistory := [][estimateFeeDepth]BtcPerKilobyte{eft.estimates()}

	for round := 0; round < rounds; round++ {
		eft.checkSaveAndRestore(estimateHistory[len(estimateHistory)-1])

//前进一步。
		txHistory, estimateHistory =
			eft.round(txHistory, estimateHistory, txPerRound, txPerBlock)
	}

//请反转过程，然后重试。
	for round := 1; round <= rounds; round++ {
		eft.rollback()
		eft.checkSaveAndRestore(estimateHistory[len(estimateHistory)-round-1])
	}
}
