
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
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/integration/rpctest"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

const (
	csvKey = "csv"
)

//makeTestOutput creates an on-chain output paying to a freshly generated
//p2pkh输出指定数量。
func makeTestOutput(r *rpctest.Harness, t *testing.T,
	amt btcutil.Amount) (*btcec.PrivateKey, *wire.OutPoint, []byte, error) {

//创建一个新的密钥，然后发送一些硬币到一个可消费的地址
//那把钥匙。
	key, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, nil, nil, err
	}

//使用上面创建的密钥，生成一个pkscript，它可以
//花费。
	a, err := btcutil.NewAddressPubKey(key.PubKey().SerializeCompressed(), r.ActiveNet)
	if err != nil {
		return nil, nil, nil, err
	}
	selfAddrScript, err := txscript.PayToAddrScript(a.AddressPubKeyHash())
	if err != nil {
		return nil, nil, nil, err
	}
	output := &wire.TxOut{PkScript: selfAddrScript, Value: 1e8}

//Next, create and broadcast a transaction paying to the output.
	fundTx, err := r.CreateTransaction([]*wire.TxOut{output}, 10, true)
	if err != nil {
		return nil, nil, nil, err
	}
	txHash, err := r.Node.SendRawTransaction(fundTx, true)
	if err != nil {
		return nil, nil, nil, err
	}

//上面创建的事务应包含在下一个
//生成的块。
	blockHash, err := r.Node.Generate(1)
	if err != nil {
		return nil, nil, nil, err
	}
	assertTxInBlock(r, t, blockHash[0], txHash)

//找到可使用的硬币的输出指数
//在上面生成，这是为了为
//这个输出。
	var outputIndex uint32
	if bytes.Equal(fundTx.TxOut[0].PkScript, selfAddrScript) {
		outputIndex = 0
	} else {
		outputIndex = 1
	}

	utxo := &wire.OutPoint{
		Hash:  fundTx.TxHash(),
		Index: outputIndex,
	}

	return key, utxo, selfAddrScript, nil
}

//testbip0113激活测试是否正确遵守bip 113规则
//要求所有事务终结性测试使用的MTP的约束
//最后11个块，而不是包含
//他们。
//
//概述：
//-预软叉：
//-来自MTP POV的非最终锁定时间的事务应为
//从内存池拒绝。
//-应接受非最终基于MTP的锁定时间内的交易。
//在有效块中。
//
//-后软叉：
//-来自MTP POV的非最终锁定时间的事务应为
//从mempool中拒绝并在其他有效块中找到。
//- Transactions with final lock-times from the PoV of MTP should be
//接受Mempool并在未来开采。
func TestBIP0113Activation(t *testing.T) {
	t.Parallel()

	btcdCfg := []string{"--rejectnonstd"}
	r, err := rpctest.New(&chaincfg.SimNetParams, nil, btcdCfg)
	if err != nil {
		t.Fatal("unable to create primary harness: ", err)
	}
	if err := r.SetUp(true, 1); err != nil {
		t.Fatalf("unable to setup test chain: %v", err)
	}
	defer r.TearDown()

//在下面的测试中创建一个新的输出供使用。
	const outputValue = btcutil.SatoshiPerBitcoin
	outputKey, testOutput, testPkScript, err := makeTestOutput(r, t,
		outputValue)
	if err != nil {
		t.Fatalf("unable to create test output: %v", err)
	}

//从安全带中获取新地址，我们将使用此地址
//把钱放回马具里。
	addr, err := r.NewAddress()
	if err != nil {
		t.Fatalf("unable to generate address: %v", err)
	}
	addrScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		t.Fatalf("unable to generate addr script: %v", err)
	}

//现在创建一个锁定时间为“最终”的事务
//to the latest block, but not according to the current median time
//过去的。
	tx := wire.NewMsgTx(1)
	tx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: *testOutput,
	})
	tx.AddTxOut(&wire.TxOut{
		PkScript: addrScript,
		Value:    outputValue - 1000,
	})

//我们将事务的锁定时间设置为
//current MTP of the chain.
	chainInfo, err := r.Node.GetBlockChainInfo()
	if err != nil {
		t.Fatalf("unable to query for chain info: %v", err)
	}
	tx.LockTime = uint32(chainInfo.MedianTime) + 1

	sigScript, err := txscript.SignatureScript(tx, 0, testPkScript,
		txscript.SigHashAll, outputKey, true)
	if err != nil {
		t.Fatalf("unable to generate sig: %v", err)
	}
	tx.TxIn[0].SignatureScript = sigScript

//使用mtp时，应从mempool拒绝此事务
//对于事务，最终性现在是一个策略规则。另外，
//准确的错误应该是拒绝非最终交易。
	_, err = r.Node.SendRawTransaction(tx, true)
	if err == nil {
		t.Fatalf("transaction accepted, but should be non-final")
	} else if !strings.Contains(err.Error(), "not finalized") {
		t.Fatalf("transaction should be rejected due to being "+
			"non-final, instead: %v", err)
	}

//但是，由于块验证共识规则还没有
//激活后，应接受包含交易的块。
	txns := []*btcutil.Tx{btcutil.NewTx(tx)}
	block, err := r.GenerateAndSubmitBlock(txns, -1, time.Time{})
	if err != nil {
		t.Fatalf("unable to submit block: %v", err)
	}
	txid := tx.TxHash()
	assertTxInBlock(r, t, block.Hash(), &txid)

//此时，块高应为103：我们开采了101个块。
//创建一个成熟的输出，然后创建一个附加的块
//一个新的输出，然后在上面挖掘一个块来包含
//交易。
	assertChainHeight(r, t, 103)

//接下来，挖掘足够的石块，确保软叉
//激活。断言第二个到最后一个块的块版本
//in the final range is active.

//其次，我的确保块，以确保软叉
//主动的。我们的高度是103，我们需要挖掘200个街区
//创世纪的目标时期，所以我们开采了196个区块。这会让我们
//身高299。getBlockChainInfo调用检查
//在当前高度之后阻塞。
	numBlocks := (r.ActiveNet.MinerConfirmationWindow * 2) - 4
	if _, err := r.Node.Generate(numBlocks); err != nil {
		t.Fatalf("unable to generate blocks: %v", err)
	}

	assertChainHeight(r, t, 299)
	assertSoftForkStatus(r, t, csvKey, blockchain.ThresholdActive)

//TimeLockDeltas切片表示与
//用于测试边界条件w.r.t的当前MTP
//交易最终性。-1表示MTP前1秒，0
//indicates the current MTP, and 1 indicates 1 second after the
//当前MTP。
//
//这一次，所有根据MTP最终确定的交易
//*应该*同时被mempool和有效块接受。
//While transactions with lock-times *after* the current MTP should be
//拒绝。
	timeLockDeltas := []int64{-1, 0, 1}
	for _, timeLockDelta := range timeLockDeltas {
		chainInfo, err = r.Node.GetBlockChainInfo()
		if err != nil {
			t.Fatalf("unable to query for chain info: %v", err)
		}
		medianTimePast := chainInfo.MedianTime

//创建另一个测试输出，将在下面花费很长时间。
		outputKey, testOutput, testPkScript, err = makeTestOutput(r, t,
			outputValue)
		if err != nil {
			t.Fatalf("unable to create test output: %v", err)
		}

//创建一个锁定时间超过当前已知时间的新事务
//MTP。
		tx = wire.NewMsgTx(1)
		tx.AddTxIn(&wire.TxIn{
			PreviousOutPoint: *testOutput,
		})
		tx.AddTxOut(&wire.TxOut{
			PkScript: addrScript,
			Value:    outputValue - 1000,
		})
		tx.LockTime = uint32(medianTimePast + timeLockDelta)
		sigScript, err = txscript.SignatureScript(tx, 0, testPkScript,
			txscript.SigHashAll, outputKey, true)
		if err != nil {
			t.Fatalf("unable to generate sig: %v", err)
		}
		tx.TxIn[0].SignatureScript = sigScript

//如果时间锁定增量大于-1，则
//应拒绝来自mempool的事务，并在
//包含在一个块中。时间锁定增量应为-1
//接受，因为锁定时间为1
//第二步，在当前的MTP之前。

		_, err = r.Node.SendRawTransaction(tx, true)
		if err == nil && timeLockDelta >= 0 {
			t.Fatal("transaction was accepted into the mempool " +
				"but should be rejected!")
		} else if err != nil && !strings.Contains(err.Error(), "not finalized") {
			t.Fatalf("transaction should be rejected from mempool "+
				"due to being  non-final, instead: %v", err)
		}

		txns = []*btcutil.Tx{btcutil.NewTx(tx)}
		_, err := r.GenerateAndSubmitBlock(txns, -1, time.Time{})
		if err == nil && timeLockDelta >= 0 {
			t.Fatal("block should be rejected due to non-final " +
				"txn, but was accepted")
		} else if err != nil && !strings.Contains(err.Error(), "unfinalized") {
			t.Fatalf("block should be rejected due to non-final "+
				"tx, instead: %v", err)
		}
	}
}

//createcsvoutput创建一个输出，支付给一个微不足道的可赎回csv
//具有指定时间锁的pkscript。
func createCSVOutput(r *rpctest.Harness, t *testing.T,
	numSatoshis btcutil.Amount, timeLock int32,
	isSeconds bool) ([]byte, *wire.OutPoint, *wire.MsgTx, error) {

//将时间锁转换为基于
//如果锁是基于秒或时间的。
	sequenceLock := blockchain.LockTimeToSequence(isSeconds,
		uint32(timeLock))

//我们的csv脚本只是：<sequencelock>op_csv op_drop
	b := txscript.NewScriptBuilder().
		AddInt64(int64(sequenceLock)).
		AddOp(txscript.OP_CHECKSEQUENCEVERIFY).
		AddOp(txscript.OP_DROP)
	csvScript, err := b.Script()
	if err != nil {
		return nil, nil, nil, err
	}

//使用上面生成的脚本，创建一个p2sh输出
//接受进入mempool。
	p2shAddr, err := btcutil.NewAddressScriptHash(csvScript, r.ActiveNet)
	if err != nil {
		return nil, nil, nil, err
	}
	p2shScript, err := txscript.PayToAddrScript(p2shAddr)
	if err != nil {
		return nil, nil, nil, err
	}
	output := &wire.TxOut{
		PkScript: p2shScript,
		Value:    int64(numSatoshis),
	}

//最后创建一个有效的事务，该事务创建精心设计的输出
//上面。
	tx, err := r.CreateTransaction([]*wire.TxOut{output}, 10, true)
	if err != nil {
		return nil, nil, nil, err
	}

	var outputIndex uint32
	if !bytes.Equal(tx.TxOut[0].PkScript, p2shScript) {
		outputIndex = 1
	}

	utxo := &wire.OutPoint{
		Hash:  tx.TxHash(),
		Index: outputIndex,
	}

	return csvScript, utxo, tx, nil
}

//SpendcSvOutput使用以前由CreateCsVoutPut创建的输出
//功能。SigScript是OptrueTrimes后面的一个微不足道的推送。
//ReDEMcript通过2SH评估。
func spendCSVOutput(redeemScript []byte, csvUTXO *wire.OutPoint,
	sequence uint32, targetOutput *wire.TxOut,
	txVersion int32) (*wire.MsgTx, error) {

	tx := wire.NewMsgTx(txVersion)
	tx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: *csvUTXO,
		Sequence:         sequence,
	})
	tx.AddTxOut(targetOutput)

	b := txscript.NewScriptBuilder().
		AddOp(txscript.OP_TRUE).
		AddData(redeemScript)

	sigScript, err := b.Script()
	if err != nil {
		return nil, err
	}
	tx.TxIn[0].SignatureScript = sigScript

	return tx, nil
}

//断言TxInBlock断言找到具有指定TxID的事务
//在具有传递的块哈希的块中。
func assertTxInBlock(r *rpctest.Harness, t *testing.T, blockHash *chainhash.Hash,
	txid *chainhash.Hash) {

	block, err := r.Node.GetBlock(blockHash)
	if err != nil {
		t.Fatalf("unable to get block: %v", err)
	}
	if len(block.Transactions) < 2 {
		t.Fatal("target transaction was not mined")
	}

	for _, txn := range block.Transactions {
		txHash := txn.TxHash()
		if txn.TxHash() == txHash {
			return
		}
	}

	_, _, line, _ := runtime.Caller(1)
	t.Fatalf("assertion failed at line %v: txid %v was not found in "+
		"block %v", line, txid, blockHash)
}

//TestBIP0068AndBIP0112Activation tests for the proper adherence to the BIP
//112和BIP 68规则在激活csv软件包软分叉后设置。
//
//概述：
//-预软叉：
//-应拒绝有效使用csv输出的交易
//但在有效的生成块中接受，包括
//交易。
//-后软叉：
//-见表驱动测试中的案例。
//这个测试。
func TestBIP0068AndBIP0112Activation(t *testing.T) {
	t.Parallel()

//We'd like the test proper evaluation and validation of the BIP 68
//（序列锁）和BIP 112规则集，用于添加基于输入时间的规则集
//相对锁定时间。

	btcdCfg := []string{"--rejectnonstd"}
	r, err := rpctest.New(&chaincfg.SimNetParams, nil, btcdCfg)
	if err != nil {
		t.Fatal("unable to create primary harness: ", err)
	}
	if err := r.SetUp(true, 1); err != nil {
		t.Fatalf("unable to setup test chain: %v", err)
	}
	defer r.TearDown()

	assertSoftForkStatus(r, t, csvKey, blockchain.ThresholdStarted)

	harnessAddr, err := r.NewAddress()
	if err != nil {
		t.Fatalf("unable to obtain harness address: %v", err)
	}
	harnessScript, err := txscript.PayToAddrScript(harnessAddr)
	if err != nil {
		t.Fatalf("unable to generate pkScript: %v", err)
	}

	const (
		outputAmt         = btcutil.SatoshiPerBitcoin
		relativeBlockLock = 10
	)

	sweepOutput := &wire.TxOut{
		Value:    outputAmt - 5000,
		PkScript: harnessScript,
	}

//由于软叉尚未激活
//应接受使用csv操作码的。因为在这一点上，
//csv实际上并不存在，它只是一个nop。
	for txVersion := int32(0); txVersion < 3; txVersion++ {
//创建一个具有csv锁定时间为
//10个相对块。
		redeemScript, testUTXO, tx, err := createCSVOutput(r, t, outputAmt,
			relativeBlockLock, false)
		if err != nil {
			t.Fatalf("unable to create CSV encumbered output: %v", err)
		}

//由于交易是P2SH，因此应将其接受为
//在下一个生成的块中找到。
		if _, err := r.Node.SendRawTransaction(tx, true); err != nil {
			t.Fatalf("unable to broadcast tx: %v", err)
		}
		blocks, err := r.Node.Generate(1)
		if err != nil {
			t.Fatalf("unable to generate blocks: %v", err)
		}
		txid := tx.TxHash()
		assertTxInBlock(r, t, blocks[0], &txid)

//生成使用csv输出的自定义事务。
		sequenceNum := blockchain.LockTimeToSequence(false, 10)
		spendingTx, err := spendCSVOutput(redeemScript, testUTXO,
			sequenceNum, sweepOutput, txVersion)
		if err != nil {
			t.Fatalf("unable to spend csv output: %v", err)
		}

//此事务应从mempool中拒绝，因为
//csv验证已经是mempool策略预分叉。
		_, err = r.Node.SendRawTransaction(spendingTx, true)
		if err == nil {
			t.Fatalf("transaction should have been rejected, but was " +
				"instead accepted")
		}

//但是，此事务应在自定义中接受
//生成块作为块内脚本的csv验证
//还不应该活跃。
		txns := []*btcutil.Tx{btcutil.NewTx(spendingTx)}
		block, err := r.GenerateAndSubmitBlock(txns, -1, time.Time{})
		if err != nil {
			t.Fatalf("unable to submit block: %v", err)
		}
		txid = spendingTx.TxHash()
		assertTxInBlock(r, t, block.Hash(), &txid)
	}

//此时，木块高度应为107：我们从高处开始。
//101，然后在上面的每个循环迭代中生成2个块。
	assertChainHeight(r, t, 107)

//With the height at 107 we need 200 blocks to be mined after the
//创世纪目标期，所以我们开采了192个区块。这会让我们
//身高299。getBlockChainInfo调用检查
//在当前高度之后阻塞。
	numBlocks := (r.ActiveNet.MinerConfirmationWindow * 2) - 8
	if _, err := r.Node.Generate(numBlocks); err != nil {
		t.Fatalf("unable to generate blocks: %v", err)
	}

	assertChainHeight(r, t, 299)
	assertSoftForkStatus(r, t, csvKey, blockchain.ThresholdActive)

//知道下面测试所需的输出数量，创建一个
//在下面的每个测试用例中使用的新输出。
	const relativeTimeLock = 512
	const numTests = 8
	type csvOutput struct {
		RedeemScript []byte
		Utxo         *wire.OutPoint
		Timelock     int32
	}
	var spendableInputs [numTests]csvOutput

//创建具有基于块的序列锁的三个输出，以及
//使用上述基于时间的序列锁的三个输出。
	for i := 0; i < numTests; i++ {
		timeLock := relativeTimeLock
		isSeconds := true
		if i < 7 {
			timeLock = relativeBlockLock
			isSeconds = false
		}

		redeemScript, utxo, tx, err := createCSVOutput(r, t, outputAmt,
			int32(timeLock), isSeconds)
		if err != nil {
			t.Fatalf("unable to create CSV output: %v", err)
		}

		if _, err := r.Node.SendRawTransaction(tx, true); err != nil {
			t.Fatalf("unable to broadcast transaction: %v", err)
		}

		spendableInputs[i] = csvOutput{
			RedeemScript: redeemScript,
			Utxo:         utxo,
			Timelock:     int32(timeLock),
		}
	}

//挖掘单个块，包括上面生成的所有事务。
	if _, err := r.Node.Generate(1); err != nil {
		t.Fatalf("unable to generate block: %v", err)
	}

//现在挖掘10个额外的块，给出上面生成的输入
//年龄11岁。在上一个街区10分钟后，给每个街区留出空间。
	prevBlockHash, err := r.Node.GetBestBlockHash()
	if err != nil {
		t.Fatalf("unable to get prior block hash: %v", err)
	}
	prevBlock, err := r.Node.GetBlock(prevBlockHash)
	if err != nil {
		t.Fatalf("unable to get block: %v", err)
	}
	for i := 0; i < relativeBlockLock; i++ {
		timeStamp := prevBlock.Header.Timestamp.Add(time.Minute * 10)
		b, err := r.GenerateAndSubmitBlock(nil, -1, timeStamp)
		if err != nil {
			t.Fatalf("unable to generate block: %v", err)
		}

		prevBlock = b.MsgBlock()
	}

//帮助程序函数，用于在
//下面的数组初始化。
	var inputIndex uint32
	makeTxCase := func(sequenceNum uint32, txVersion int32) *wire.MsgTx {
		csvInput := spendableInputs[inputIndex]

		tx, err := spendCSVOutput(csvInput.RedeemScript, csvInput.Utxo,
			sequenceNum, sweepOutput, txVersion)
		if err != nil {
			t.Fatalf("unable to spend CSV output: %v", err)
		}

		inputIndex++
		return tx
	}

	tests := [numTests]struct {
		tx     *wire.MsgTx
		accept bool
	}{
//单输入序列号的有效事务
//创建100个块的相对时间锁定。本次交易
//应该被拒绝，因为它的版本号是1，并且只有tx
//版本>2将触发csv行为。
		{
			tx:     makeTxCase(blockchain.LockTimeToSequence(false, 100), 1),
			accept: false,
		},
//第2版的一种事务，只使用一个输入。这个
//输入的相对时间锁定为1个块，但禁用
//比特设置。因此，应拒绝该事务。
		{
			tx: makeTxCase(
				blockchain.LockTimeToSequence(false, 1)|wire.SequenceLockTimeDisabled,
				2,
			),
			accept: false,
		},
//一种具有9个块的单个输入的v2事务。
//相对时间锁定。参考输入是11块旧的，
//但是csv输出需要10个块的相对锁定时间。
//因此，交易应该被拒绝。
		{
			tx:     makeTxCase(blockchain.LockTimeToSequence(false, 9), 2),
			accept: false,
		},
//一种具有10个块的单个输入的v2事务。
//相对时间锁定。参考输入为11个块，所以
//交易应被接受。
		{
			tx:     makeTxCase(blockchain.LockTimeToSequence(false, 10), 2),
			accept: true,
		},
//A v2 transaction with a single input having a 11 block
//相对时间锁定。引用的输入的输入期限为
//11 and the CSV op-code requires 10 blocks to have passed, so
//这笔交易应该被接受。
		{
			tx:     makeTxCase(blockchain.LockTimeToSequence(false, 11), 2),
			accept: true,
		},
//其输入具有1000 blck相对时间的v2事务
//锁。这应该被拒绝，因为输入的年龄只有11岁
//阻碍。
		{
			tx:     makeTxCase(blockchain.LockTimeToSequence(false, 1000), 2),
			accept: false,
		},
//具有一个512000秒的单输入的v2事务
//相对时间锁定。此交易应作为6拒绝
//数天的街区尚未开采。被引用的
//输入的时间不足。
		{
			tx:     makeTxCase(blockchain.LockTimeToSequence(true, 512000), 2),
			accept: false,
		},
//一个V2事务，其单个输入有512秒
//相对时间锁定。此交易应被接受为
//定稿。
		{
			tx:     makeTxCase(blockchain.LockTimeToSequence(true, 512), 2),
			accept: true,
		},
	}

	for i, test := range tests {
		txid, err := r.Node.SendRawTransaction(test.tx, true)
		switch {
//测试用例通过，无需进一步报告。
		case test.accept && err == nil:

//交易本应被接受，但我们有一个非零
//错误。
		case test.accept && err != nil:
			t.Fatalf("test #%d, transaction should be accepted, "+
				"but was rejected: %v", i, err)

//交易本应被拒绝，但已被接受。
		case !test.accept && err == nil:
			t.Fatalf("test #%d, transaction should be rejected, "+
				"but was accepted", i)

//交易因需要而被拒绝，没什么可做的。
		case !test.accept && err != nil:
		}

//如果该事务应被拒绝，请手动挖掘块
//与非最终交易。它应该被拒绝。
		if !test.accept {
			txns := []*btcutil.Tx{btcutil.NewTx(test.tx)}
			_, err := r.GenerateAndSubmitBlock(txns, -1, time.Time{})
			if err == nil {
				t.Fatalf("test #%d, invalid block accepted", i)
			}

			continue
		}

//生成一个块，该事务应包含在
//新开采的区块
		blockHashes, err := r.Node.Generate(1)
		if err != nil {
			t.Fatalf("unable to mine block: %v", err)
		}
		assertTxInBlock(r, t, blockHashes[0], txid)
	}
}
