
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
	"encoding/hex"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

//fakechain被池线束用来提供生成的测试utxos和
//当前池回调的假链高度。反过来，这允许
//交易看起来好像在支出完全有效的utxos。
type fakeChain struct {
	sync.RWMutex
	utxos          *blockchain.UtxoViewpoint
	currentHeight  int32
	medianTimePast time.Time
}

//fetchutxoview加载有关传递的引用的输入的utxo详细信息
//从假链的角度看交易。它还试图
//获取事务本身输出的utxos，以便返回
//可以检查视图是否有重复的事务。
//
//此函数对于并发访问是安全的，但是返回的视图不是。
func (s *fakeChain) FetchUtxoView(tx *btcutil.Tx) (*blockchain.UtxoViewpoint, error) {
	s.RLock()
	defer s.RUnlock()

//克隆所有条目以确保对返回视图进行修改
//不要影响假链的视图。

//将Tx本身的条目添加到新视图中。
	viewpoint := blockchain.NewUtxoViewpoint()
	prevOut := wire.OutPoint{Hash: *tx.Hash()}
	for txOutIdx := range tx.MsgTx().TxOut {
		prevOut.Index = uint32(txOutIdx)
		entry := s.utxos.LookupEntry(prevOut)
		viewpoint.Entries()[prevOut] = entry.Clone()
	}

//将所有输入的条目添加到新视图的tx。
	for _, txIn := range tx.MsgTx().TxIn {
		entry := s.utxos.LookupEntry(txIn.PreviousOutPoint)
		viewpoint.Entries()[txIn.PreviousOutPoint] = entry.Clone()
	}

	return viewpoint, nil
}

//BestHeight返回与假链关联的当前高度
//实例。
func (s *fakeChain) BestHeight() int32 {
	s.RLock()
	height := s.currentHeight
	s.RUnlock()
	return height
}

//setheight设置与假链实例关联的当前高度。
func (s *fakeChain) SetHeight(height int32) {
	s.Lock()
	s.currentHeight = height
	s.Unlock()
}

//MediantimePost返回与假事件相关联的当前中位时间
//链式实例。
func (s *fakeChain) MedianTimePast() time.Time {
	s.RLock()
	mtp := s.medianTimePast
	s.RUnlock()
	return mtp
}

//setMediantimePost设置与假关联的当前中值时间
//链式实例。
func (s *fakeChain) SetMedianTimePast(mtp time.Time) {
	s.Lock()
	s.medianTimePast = mtp
	s.Unlock()
}

//CalcSequenceLock returns the current sequence lock for the passed
//与假链实例关联的事务。
func (s *fakeChain) CalcSequenceLock(tx *btcutil.Tx,
	view *blockchain.UtxoViewpoint) (*blockchain.SequenceLock, error) {

	return &blockchain.SequenceLock{
		Seconds:     -1,
		BlockHeight: -1,
	}, nil
}

//SpendableOutput是一种方便的类型，它包含一个特定的utxo和
//与之关联的金额。
type spendableOutput struct {
	outPoint wire.OutPoint
	amount   btcutil.Amount
}

//TxOutToPendableOut返回给定事务和索引的可使用输出
//要使用的输出。这在创建测试时非常有用
//交易。
func txOutToSpendableOut(tx *btcutil.Tx, outputNum uint32) spendableOutput {
	return spendableOutput{
		outPoint: wire.OutPoint{Hash: *tx.Hash(), Index: outputNum},
		amount:   btcutil.Amount(tx.MsgTx().TxOut[outputNum].Value),
	}
}

//Poolharness提供了一个包含创建和
//签署事务以及提供utxos用于
//生成有效的事务。
type poolHarness struct {
//signkey是用于在整个
//测试。
//
//payaddr是签名密钥的p2sh地址，用于
//整个测试的付款地址。
	signKey     *btcec.PrivateKey
	payAddr     btcutil.Address
	payScript   []byte
	chainParams *chaincfg.Params

	chain  *fakeChain
	txPool *TxPool
}

//CreateCoinBaseTx返回一个CoinBase事务，请求的事务数为
//根据通过的块高度支付适当补贴的输出
//与线束关联的地址。它自动使用一个标准
//以所需的块高度开始的签名脚本
//版本2块。
func (p *poolHarness) CreateCoinbaseTx(blockHeight int32, numOutputs uint32) (*btcutil.Tx, error) {
//创建标准的coinbase脚本。
	extraNonce := int64(0)
	coinbaseScript, err := txscript.NewScriptBuilder().
		AddInt64(int64(blockHeight)).AddInt64(extraNonce).Script()
	if err != nil {
		return nil, err
	}

	tx := wire.NewMsgTx(wire.TxVersion)
	tx.AddTxIn(&wire.TxIn{
//CoinBase事务没有输入，因此以前的输出点是
//零哈希和最大索引。
		PreviousOutPoint: *wire.NewOutPoint(&chainhash.Hash{},
			wire.MaxPrevOutIndex),
		SignatureScript: coinbaseScript,
		Sequence:        wire.MaxTxInSequenceNum,
	})
	totalInput := blockchain.CalcBlockSubsidy(blockHeight, p.chainParams)
	amountPerOutput := totalInput / int64(numOutputs)
	remainder := totalInput - amountPerOutput*int64(numOutputs)
	for i := uint32(0); i < numOutputs; i++ {
//确保所有可能
//不要拆分输入金额。
		amount := amountPerOutput
		if i == numOutputs-1 {
			amount = amountPerOutput + remainder
		}
		tx.AddTxOut(&wire.TxOut{
			PkScript: p.payScript,
			Value:    amount,
		})
	}

	return btcutil.NewTx(tx), nil
}

//CreateSignedTx创建一个新的已签名事务，该事务使用提供的
//通过平均分割
//总输入金额。所有输出都将指向关联的付款脚本
//对于线束和所有输入，都假定执行相同的操作。
func (p *poolHarness) CreateSignedTx(inputs []spendableOutput, numOutputs uint32) (*btcutil.Tx, error) {
//计算总输入金额，并将其拆分到请求的
//输出数量。
	var totalInput btcutil.Amount
	for _, input := range inputs {
		totalInput += input.amount
	}
	amountPerOutput := int64(totalInput) / int64(numOutputs)
	remainder := int64(totalInput) - amountPerOutput*int64(numOutputs)

	tx := wire.NewMsgTx(wire.TxVersion)
	for _, input := range inputs {
		tx.AddTxIn(&wire.TxIn{
			PreviousOutPoint: input.outPoint,
			SignatureScript:  nil,
			Sequence:         wire.MaxTxInSequenceNum,
		})
	}
	for i := uint32(0); i < numOutputs; i++ {
//确保所有可能
//不要拆分输入金额。
		amount := amountPerOutput
		if i == numOutputs-1 {
			amount = amountPerOutput + remainder
		}
		tx.AddTxOut(&wire.TxOut{
			PkScript: p.payScript,
			Value:    amount,
		})
	}

//签署新交易。
	for i := range tx.TxIn {
		sigScript, err := txscript.SignatureScript(tx, i, p.payScript,
			txscript.SigHashAll, p.signKey, true)
		if err != nil {
			return nil, err
		}
		tx.TxIn[i].SignatureScript = sigScript
	}

	return btcutil.NewTx(tx), nil
}

//CreateTxChain创建零费用交易链（每个后续交易
//交易支出前一个）的全部金额
//一个花费提供的输出点。每笔交易都要花费整个
//前一笔的金额，因此不包括任何费用。
func (p *poolHarness) CreateTxChain(firstOutput spendableOutput, numTxns uint32) ([]*btcutil.Tx, error) {
	txChain := make([]*btcutil.Tx, 0, numTxns)
	prevOutPoint := firstOutput.outPoint
	spendableAmount := firstOutput.amount
	for i := uint32(0); i < numTxns; i++ {
//使用上一个事务输出创建事务
//并将全额支付至相关的支付地址。
//带上安全带。
		tx := wire.NewMsgTx(wire.TxVersion)
		tx.AddTxIn(&wire.TxIn{
			PreviousOutPoint: prevOutPoint,
			SignatureScript:  nil,
			Sequence:         wire.MaxTxInSequenceNum,
		})
		tx.AddTxOut(&wire.TxOut{
			PkScript: p.payScript,
			Value:    int64(spendableAmount),
		})

//签署新交易。
		sigScript, err := txscript.SignatureScript(tx, 0, p.payScript,
			txscript.SigHashAll, p.signKey, true)
		if err != nil {
			return nil, err
		}
		tx.TxIn[0].SignatureScript = sigScript

		txChain = append(txChain, btcutil.NewTx(tx))

//下一个事务使用这个事务的输出。
		prevOutPoint = wire.OutPoint{Hash: tx.TxHash(), Index: 0}
	}

	return txChain, nil
}

//new pool harness返回用初始化的池线束的新实例
//假链和绑定到它的TxPool，它配置了合适的策略
//用于测试。此外，假链中还填充了返回的可消费的
//输出以便调用者可以轻松创建新的有效事务
//离开它。
func newPoolHarness(chainParams *chaincfg.Params) (*poolHarness, []spendableOutput, error) {
//使用硬编码密钥对获得确定性结果。
	keyBytes, err := hex.DecodeString("700868df1838811ffbdf918fb482c1f7e" +
		"ad62db4b97bd7012c23e726485e577d")
	if err != nil {
		return nil, nil, err
	}
	signKey, signPub := btcec.PrivKeyFromBytes(btcec.S256(), keyBytes)

//生成关联的付款到脚本哈希地址和结果付款
//脚本。
	pubKeyBytes := signPub.SerializeCompressed()
	payPubKeyAddr, err := btcutil.NewAddressPubKey(pubKeyBytes, chainParams)
	if err != nil {
		return nil, nil, err
	}
	payAddr := payPubKeyAddr.AddressPubKeyHash()
	pkScript, err := txscript.PayToAddrScript(payAddr)
	if err != nil {
		return nil, nil, err
	}

//创建一个新的假链和绑定到它的安全带。
	chain := &fakeChain{utxos: blockchain.NewUtxoViewpoint()}
	harness := poolHarness{
		signKey:     signKey,
		payAddr:     payAddr,
		payScript:   pkScript,
		chainParams: chainParams,

		chain: chain,
		txPool: New(&Config{
			Policy: Policy{
				DisableRelayPriority: true,
				FreeTxRelayLimit:     15.0,
				MaxOrphanTxs:         5,
				MaxOrphanTxSize:      1000,
				MaxSigOpCostPerTx:    blockchain.MaxBlockSigOpsCost / 4,
MinRelayTxFee:        1000, //每字节1个Satoshi
				MaxTxVersion:         1,
			},
			ChainParams:      chainParams,
			FetchUtxoView:    chain.FetchUtxoView,
			BestHeight:       chain.BestHeight,
			MedianTimePast:   chain.MedianTimePast,
			CalcSequenceLock: chain.CalcSequenceLock,
			SigCache:         nil,
			AddrIndex:        nil,
		}),
	}

//创建单个CoinBase事务并将其添加到线束中
//链的utxo设置并设置线束链的高度，以便
//Coinbase将在下一个街区成熟。这样可以确保txpool
//接受那些花费不成熟的硬币的交易
//在下一个街区成熟。
	numOutputs := uint32(1)
	outputs := make([]spendableOutput, 0, numOutputs)
	curHeight := harness.chain.BestHeight()
	coinbase, err := harness.CreateCoinbaseTx(curHeight+1, numOutputs)
	if err != nil {
		return nil, nil, err
	}
	harness.chain.utxos.AddTxOuts(coinbase, curHeight+1)
	for i := uint32(0); i < numOutputs; i++ {
		outputs = append(outputs, txOutToSpendableOut(coinbase, i))
	}
	harness.chain.SetHeight(int32(chainParams.CoinbaseMaturity) + curHeight)
	harness.chain.SetMedianTimePast(time.Now())

	return &harness, outputs, nil
}

//testcontext包含一个与测试相关的状态，该状态可用于传递给helper
//作为单个参数。
type testContext struct {
	t       *testing.T
	harness *poolHarness
}

//testpoolmembership测试与提供的
//测试上下文以确定传递的事务是否与提供的
//孤立池和事务池状态。它还进一步确定
//应该由HaveTransaction函数根据
//这两个标志并测试该条件。
func testPoolMembership(tc *testContext, tx *btcutil.Tx, inOrphanPool, inTxPool bool) {
	txHash := tx.Hash()
	gotOrphanPool := tc.harness.txPool.IsOrphanInPool(txHash)
	if inOrphanPool != gotOrphanPool {
		_, file, line, _ := runtime.Caller(1)
		tc.t.Fatalf("%s:%d -- IsOrphanInPool: want %v, got %v", file,
			line, inOrphanPool, gotOrphanPool)
	}

	gotTxPool := tc.harness.txPool.IsTransactionInPool(txHash)
	if inTxPool != gotTxPool {
		_, file, line, _ := runtime.Caller(1)
		tc.t.Fatalf("%s:%d -- IsTransactionInPool: want %v, got %v",
			file, line, inTxPool, gotTxPool)
	}

	gotHaveTx := tc.harness.txPool.HaveTransaction(txHash)
	wantHaveTx := inOrphanPool || inTxPool
	if wantHaveTx != gotHaveTx {
		_, file, line, _ := runtime.Caller(1)
		tc.t.Fatalf("%s:%d -- HaveTransaction: want %v, got %v", file,
			line, wantHaveTx, gotHaveTx)
	}
}

//testsimpleorphanchain确保处理简单的孤立链
//适当地。特别是，它生成一个单输入单输出的链
//并在跳过第一个链接事务时插入它们，因此
//他们都是孤儿。最后，它添加了链接事务并确保
//整个孤立链将移动到事务池。
func TestSimpleOrphanChain(t *testing.T) {
	t.Parallel()

	harness, spendableOuts, err := newPoolHarness(&chaincfg.MainNetParams)
	if err != nil {
		t.Fatalf("unable to create test pool: %v", err)
	}
	tc := &testContext{t, harness}

//创建基于第一个可消费输出的事务链
//由线束提供。
	maxOrphans := uint32(harness.txPool.cfg.Policy.MaxOrphanTxs)
	chainedTxns, err := harness.CreateTxChain(spendableOuts[0], maxOrphans+1)
	if err != nil {
		t.Fatalf("unable to create transaction chain: %v", err)
	}

//确保孤儿被接受（仅在允许的最大限度内）
//没有被逐出）。
	for _, tx := range chainedTxns[1 : maxOrphans+1] {
		acceptedTxns, err := harness.txPool.ProcessTransaction(tx, true,
			false, 0)
		if err != nil {
			t.Fatalf("ProcessTransaction: failed to accept valid "+
				"orphan %v", err)
		}

//确保没有交易报告为已接受。
		if len(acceptedTxns) != 0 {
			t.Fatalf("ProcessTransaction: reported %d accepted "+
				"transactions from what should be an orphan",
				len(acceptedTxns))
		}

//确保事务在孤立池中，不在
//事务池，并报告为可用。
		testPoolMembership(tc, tx, true, false)
	}

//添加完成孤立链的事务并确保它们
//全部接受。注意，这里的接受孤立标志也是假的
//以确保它与是否已经存在无关
//池中的孤儿被链接。
	acceptedTxns, err := harness.txPool.ProcessTransaction(chainedTxns[0],
		false, false, 0)
	if err != nil {
		t.Fatalf("ProcessTransaction: failed to accept valid "+
			"orphan %v", err)
	}
	if len(acceptedTxns) != len(chainedTxns) {
		t.Fatalf("ProcessTransaction: reported accepted transactions "+
			"length does not match expected -- got %d, want %d",
			len(acceptedTxns), len(chainedTxns))
	}
	for _, txD := range acceptedTxns {
//确保事务不再位于孤立池中，是
//现在在事务池中，并报告为可用。
		testPoolMembership(tc, txD.Tx, false, true)
	}
}

//testOrphanReject确保当允许
//未在ProcessTransaction上设置孤立标志。
func TestOrphanReject(t *testing.T) {
	t.Parallel()

	harness, outputs, err := newPoolHarness(&chaincfg.MainNetParams)
	if err != nil {
		t.Fatalf("unable to create test pool: %v", err)
	}
	tc := &testContext{t, harness}

//创建基于第一个可消费输出的事务链
//由线束提供。
	maxOrphans := uint32(harness.txPool.cfg.Policy.MaxOrphanTxs)
	chainedTxns, err := harness.CreateTxChain(outputs[0], maxOrphans+1)
	if err != nil {
		t.Fatalf("unable to create transaction chain: %v", err)
	}

//如果未设置“允许孤立对象”标志，请确保拒绝孤立对象。
	for _, tx := range chainedTxns[1:] {
		acceptedTxns, err := harness.txPool.ProcessTransaction(tx, false,
			false, 0)
		if err == nil {
			t.Fatalf("ProcessTransaction: did not fail on orphan "+
				"%v when allow orphans flag is false", tx.Hash())
		}
		expectedErr := RuleError{}
		if reflect.TypeOf(err) != reflect.TypeOf(expectedErr) {
			t.Fatalf("ProcessTransaction: wrong error got: <%T> %v, "+
				"want: <%T>", err, err, expectedErr)
		}
		code, extracted := extractRejectCode(err)
		if !extracted {
			t.Fatalf("ProcessTransaction: failed to extract reject "+
				"code from error %q", err)
		}
		if code != wire.RejectDuplicate {
			t.Fatalf("ProcessTransaction: unexpected reject code "+
				"-- got %v, want %v", code, wire.RejectDuplicate)
		}

//确保没有交易报告为已接受。
		if len(acceptedTxns) != 0 {
			t.Fatal("ProcessTransaction: reported %d accepted "+
				"transactions from failed orphan attempt",
				len(acceptedTxns))
		}

//确保事务不在孤立池中，不在
//事务池，但未报告为可用
		testPoolMembership(tc, tx, false, false)
	}
}

//testOrphanection确保超过最大孤立数
//为新的房间腾出空间。
func TestOrphanEviction(t *testing.T) {
	t.Parallel()

	harness, outputs, err := newPoolHarness(&chaincfg.MainNetParams)
	if err != nil {
		t.Fatalf("unable to create test pool: %v", err)
	}
	tc := &testContext{t, harness}

//创建基于第一个可消费输出的事务链
//由足够长的安全带提供
//一些孤儿被驱逐。
	maxOrphans := uint32(harness.txPool.cfg.Policy.MaxOrphanTxs)
	chainedTxns, err := harness.CreateTxChain(outputs[0], maxOrphans+5)
	if err != nil {
		t.Fatalf("unable to create transaction chain: %v", err)
	}

//添加足够的孤儿以超过允许的最大值，同时确保他们
//都接受了。这将导致驱逐。
	for _, tx := range chainedTxns[1:] {
		acceptedTxns, err := harness.txPool.ProcessTransaction(tx, true,
			false, 0)
		if err != nil {
			t.Fatalf("ProcessTransaction: failed to accept valid "+
				"orphan %v", err)
		}

//确保没有交易报告为已接受。
		if len(acceptedTxns) != 0 {
			t.Fatalf("ProcessTransaction: reported %d accepted "+
				"transactions from what should be an orphan",
				len(acceptedTxns))
		}

//确保事务在孤立池中，不在
//事务池，并报告为可用。
		testPoolMembership(tc, tx, true, false)
	}

//找出哪些事务被收回，并确保
//收回的与预期的数字匹配。
	var evictedTxns []*btcutil.Tx
	for _, tx := range chainedTxns[1:] {
		if !harness.txPool.IsOrphanInPool(tx.Hash()) {
			evictedTxns = append(evictedTxns, tx)
		}
	}
	expectedEvictions := len(chainedTxns) - 1 - int(maxOrphans)
	if len(evictedTxns) != expectedEvictions {
		t.Fatalf("unexpected number of evictions -- got %d, want %d",
			len(evictedTxns), expectedEvictions)
	}

//Ensure none of the evicted transactions ended up in the transaction
//池。
	for _, tx := range evictedTxns {
		testPoolMembership(tc, tx, false, false)
	}
}

//testbasicophanremoval确保在
//当有另一个孤儿
//在没有的时候赎回它。
func TestBasicOrphanRemoval(t *testing.T) {
	t.Parallel()

	const maxOrphans = 4
	harness, spendableOuts, err := newPoolHarness(&chaincfg.MainNetParams)
	if err != nil {
		t.Fatalf("unable to create test pool: %v", err)
	}
	harness.txPool.cfg.Policy.MaxOrphanTxs = maxOrphans
	tc := &testContext{t, harness}

//创建基于第一个可消费输出的事务链
//由线束提供。
	chainedTxns, err := harness.CreateTxChain(spendableOuts[0], maxOrphans+1)
	if err != nil {
		t.Fatalf("unable to create transaction chain: %v", err)
	}

//确保孤儿被接受（仅在允许的最大限度内）
//没有被逐出）。
	for _, tx := range chainedTxns[1 : maxOrphans+1] {
		acceptedTxns, err := harness.txPool.ProcessTransaction(tx, true,
			false, 0)
		if err != nil {
			t.Fatalf("ProcessTransaction: failed to accept valid "+
				"orphan %v", err)
		}

//确保没有交易报告为已接受。
		if len(acceptedTxns) != 0 {
			t.Fatalf("ProcessTransaction: reported %d accepted "+
				"transactions from what should be an orphan",
				len(acceptedTxns))
		}

//确保事务在孤立池中，而不是在
//事务池，并报告为可用。
		testPoolMembership(tc, tx, true, false)
	}

//如果一个孤儿没有救世主，也不在场，就要把他除掉，
//确保所有其他孤儿的状态不受影响。
	nonChainedOrphanTx, err := harness.CreateSignedTx([]spendableOutput{{
		amount:   btcutil.Amount(5000000000),
		outPoint: wire.OutPoint{Hash: chainhash.Hash{}, Index: 0},
	}}, 1)
	if err != nil {
		t.Fatalf("unable to create signed tx: %v", err)
	}

	harness.txPool.RemoveOrphan(nonChainedOrphanTx)
	testPoolMembership(tc, nonChainedOrphanTx, false, false)
	for _, tx := range chainedTxns[1 : maxOrphans+1] {
		testPoolMembership(tc, tx, true, false)
	}

//尝试移除一个已有的救赎者但自身
//不在场并确保所有其他孤儿（包括
//赎回它的人）不会受到影响。
	harness.txPool.RemoveOrphan(chainedTxns[0])
	testPoolMembership(tc, chainedTxns[0], false, false)
	for _, tx := range chainedTxns[1 : maxOrphans+1] {
		testPoolMembership(tc, tx, true, false)
	}

//逐个移除每个孤立对象，并确保将其移除为
//预期。
	for _, tx := range chainedTxns[1 : maxOrphans+1] {
		harness.txPool.RemoveOrphan(tx)
		testPoolMembership(tc, tx, false, false)
	}
}

//testOrphanChainRemove确保孤立链（花费输出的孤立链）
//从其他孤儿中）按预期移除。
func TestOrphanChainRemoval(t *testing.T) {
	t.Parallel()

	const maxOrphans = 10
	harness, spendableOuts, err := newPoolHarness(&chaincfg.MainNetParams)
	if err != nil {
		t.Fatalf("unable to create test pool: %v", err)
	}
	harness.txPool.cfg.Policy.MaxOrphanTxs = maxOrphans
	tc := &testContext{t, harness}

//创建基于第一个可消费输出的事务链
//由线束提供。
	chainedTxns, err := harness.CreateTxChain(spendableOuts[0], maxOrphans+1)
	if err != nil {
		t.Fatalf("unable to create transaction chain: %v", err)
	}

//确保孤儿被接受（仅在允许的最大限度内）
//没有被逐出）。
	for _, tx := range chainedTxns[1 : maxOrphans+1] {
		acceptedTxns, err := harness.txPool.ProcessTransaction(tx, true,
			false, 0)
		if err != nil {
			t.Fatalf("ProcessTransaction: failed to accept valid "+
				"orphan %v", err)
		}

//确保没有交易报告为已接受。
		if len(acceptedTxns) != 0 {
			t.Fatalf("ProcessTransaction: reported %d accepted "+
				"transactions from what should be an orphan",
				len(acceptedTxns))
		}

//确保事务在孤立池中，而不是在
//事务池，并报告为可用。
		testPoolMembership(tc, tx, true, false)
	}

//移除启动孤立链的第一个孤立项，而不使用
//删除redeemer标志集，并确保只有第一个孤立的
//远离的。
	harness.txPool.mtx.Lock()
	harness.txPool.removeOrphan(chainedTxns[1], false)
	harness.txPool.mtx.Unlock()
	testPoolMembership(tc, chainedTxns[1], false, false)
	for _, tx := range chainedTxns[2 : maxOrphans+1] {
		testPoolMembership(tc, tx, true, false)
	}

//移除启动孤立链的第一个剩余孤立项
//设置了移除重新激活器标志，并确保它们都已移除。
	harness.txPool.mtx.Lock()
	harness.txPool.removeOrphan(chainedTxns[2], true)
	harness.txPool.mtx.Unlock()
	for _, tx := range chainedTxns[2 : maxOrphans+1] {
		testPoolMembership(tc, tx, false, false)
	}
}

//testmulinputorphandoublespend确保从
//将删除另一个进入池的事务所花费的输出。
func TestMultiInputOrphanDoubleSpend(t *testing.T) {
	t.Parallel()

	const maxOrphans = 4
	harness, outputs, err := newPoolHarness(&chaincfg.MainNetParams)
	if err != nil {
		t.Fatalf("unable to create test pool: %v", err)
	}
	harness.txPool.cfg.Policy.MaxOrphanTxs = maxOrphans
	tc := &testContext{t, harness}

//创建基于第一个可消费输出的事务链
//由线束提供。
	chainedTxns, err := harness.CreateTxChain(outputs[0], maxOrphans+1)
	if err != nil {
		t.Fatalf("unable to create transaction chain: %v", err)
	}

//首先从生成的链中添加孤立事务
//除了最后一个。
	for _, tx := range chainedTxns[1:maxOrphans] {
		acceptedTxns, err := harness.txPool.ProcessTransaction(tx, true,
			false, 0)
		if err != nil {
			t.Fatalf("ProcessTransaction: failed to accept valid "+
				"orphan %v", err)
		}
		if len(acceptedTxns) != 0 {
			t.Fatalf("ProcessTransaction: reported %d accepted transactions "+
				"from what should be an orphan", len(acceptedTxns))
		}
		testPoolMembership(tc, tx, true, false)
	}

//确保包含相同输出的双倍开销的事务
//作为第二个刚刚加入的孤儿以及一个有效的花费
//从上面生成的链中的最后一个孤立项（不在
//孤立池）被接受为孤立池。必须允许这样做
//因为如果不是这样，恶意行为人可能会破坏
//TX连锁店。
	doubleSpendTx, err := harness.CreateSignedTx([]spendableOutput{
		txOutToSpendableOut(chainedTxns[1], 0),
		txOutToSpendableOut(chainedTxns[maxOrphans], 0),
	}, 1)
	if err != nil {
		t.Fatalf("unable to create signed tx: %v", err)
	}
	acceptedTxns, err := harness.txPool.ProcessTransaction(doubleSpendTx,
		true, false, 0)
	if err != nil {
		t.Fatalf("ProcessTransaction: failed to accept valid orphan %v",
			err)
	}
	if len(acceptedTxns) != 0 {
		t.Fatalf("ProcessTransaction: reported %d accepted transactions "+
			"from what should be an orphan", len(acceptedTxns))
	}
	testPoolMembership(tc, doubleSpendTx, true, false)

//添加完成孤立链的事务并确保
//链被接受。请注意，接受孤立标志也是假的
//以确保它与是否已经存在无关
//池中的孤儿被链接。
//
//这将导致共享的输出成为一个具体的支出，
//反过来，遗嘱也必须使花钱加倍的孤儿被除名。
	acceptedTxns, err = harness.txPool.ProcessTransaction(chainedTxns[0],
		false, false, 0)
	if err != nil {
		t.Fatalf("ProcessTransaction: failed to accept valid tx %v", err)
	}
	if len(acceptedTxns) != maxOrphans {
		t.Fatalf("ProcessTransaction: reported accepted transactions "+
			"length does not match expected -- got %d, want %d",
			len(acceptedTxns), maxOrphans)
	}
	for _, txD := range acceptedTxns {
//确保事务不再位于孤立池中，是
//在事务池中，并报告为可用。
		testPoolMembership(tc, txD.Tx, false, true)
	}

//确保花了双倍的钱的孤儿不再在孤儿池里，并且
//未移动到事务池。
	testPoolMembership(tc, doubleSpendTx, false, false)
}

//testcheckspend用于返回在
//记忆库。
func TestCheckSpend(t *testing.T) {
	t.Parallel()

	harness, outputs, err := newPoolHarness(&chaincfg.MainNetParams)
	if err != nil {
		t.Fatalf("unable to create test pool: %v", err)
	}

//mempool为空，因此任何可花费的输出都不应具有
//在那里度过。
	for _, op := range outputs {
		spend := harness.txPool.CheckSpend(op.outPoint)
		if spend != nil {
			t.Fatalf("Unexpeced spend found in pool: %v", spend)
		}
	}

//创建以第一个可消费的
//线束提供的输出。
	const txChainLength = 5
	chainedTxns, err := harness.CreateTxChain(outputs[0], txChainLength)
	if err != nil {
		t.Fatalf("unable to create transaction chain: %v", err)
	}
	for _, tx := range chainedTxns {
		_, err := harness.txPool.ProcessTransaction(tx, true,
			false, 0)
		if err != nil {
			t.Fatalf("ProcessTransaction: failed to accept "+
				"tx: %v", err)
		}
	}

//链中的第一个Tx应该是可消费的
//输出。
	op := outputs[0].outPoint
	spend := harness.txPool.CheckSpend(op)
	if spend != chainedTxns[0] {
		t.Fatalf("expected %v to be spent by %v, instead "+
			"got %v", op, chainedTxns[0], spend)
	}

//现在，除了最后一个Tx以外，所有的Tx都应该在下一个Tx中使用。
	for i := 0; i < len(chainedTxns)-1; i++ {
		op = wire.OutPoint{
			Hash:  *chainedTxns[i].Hash(),
			Index: 0,
		}
		expSpend := chainedTxns[i+1]
		spend = harness.txPool.CheckSpend(op)
		if spend != expSpend {
			t.Fatalf("expected %v to be spent by %v, instead "+
				"got %v", op, expSpend, spend)
		}
	}

//最后一个Tx应该没有花费。
	op = wire.OutPoint{
		Hash:  *chainedTxns[txChainLength-1].Hash(),
		Index: 0,
	}
	spend = harness.txPool.CheckSpend(op)
	if spend != nil {
		t.Fatalf("Unexpeced spend found in pool: %v", spend)
	}
}
