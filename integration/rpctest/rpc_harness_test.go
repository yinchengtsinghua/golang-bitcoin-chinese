
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

package rpctest

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

func testSendOutputs(r *Harness, t *testing.T) {
	genSpend := func(amt btcutil.Amount) *chainhash.Hash {
//从钱包里拿出一个新地址。
		addr, err := r.NewAddress()
		if err != nil {
			t.Fatalf("unable to get new address: %v", err)
		}

//下一步，发送amt btc到这个地址，从我们的一个成熟的消费
//CoinBase输出。
		addrScript, err := txscript.PayToAddrScript(addr)
		if err != nil {
			t.Fatalf("unable to generate pkscript to addr: %v", err)
		}
		output := wire.NewTxOut(int64(amt), addrScript)
		txid, err := r.SendOutputs([]*wire.TxOut{output}, 10)
		if err != nil {
			t.Fatalf("coinbase spend failed: %v", err)
		}
		return txid
	}

	assertTxMined := func(txid *chainhash.Hash, blockHash *chainhash.Hash) {
		block, err := r.Node.GetBlock(blockHash)
		if err != nil {
			t.Fatalf("unable to get block: %v", err)
		}

		numBlockTxns := len(block.Transactions)
		if numBlockTxns < 2 {
			t.Fatalf("crafted transaction wasn't mined, block should have "+
				"at least %v transactions instead has %v", 2, numBlockTxns)
		}

		minedTx := block.Transactions[1]
		txHash := minedTx.TxHash()
		if txHash != *txid {
			t.Fatalf("txid's don't match, %v vs %v", txHash, txid)
		}
	}

//首先，产生一个只需要一个
//输入。
	txid := genSpend(btcutil.Amount(5 * btcutil.SatoshiPerBitcoin))

//生成单个块，钱包创建的交易应
//在这个街区找到。
	blockHashes, err := r.Node.Generate(1)
	if err != nil {
		t.Fatalf("unable to generate single block: %v", err)
	}
	assertTxMined(txid, blockHashes[0])

//接下来，产生一个比区块奖励大得多的支出。这个
//事务也应该被正确挖掘。
	txid = genSpend(btcutil.Amount(500 * btcutil.SatoshiPerBitcoin))
	blockHashes, err = r.Node.Generate(1)
	if err != nil {
		t.Fatalf("unable to generate single block: %v", err)
	}
	assertTxMined(txid, blockHashes[0])
}

func assertConnectedTo(t *testing.T, nodeA *Harness, nodeB *Harness) {
	nodeAPeers, err := nodeA.Node.GetPeerInfo()
	if err != nil {
		t.Fatalf("unable to get nodeA's peer info")
	}

	nodeAddr := nodeB.node.config.listen
	addrFound := false
	for _, peerInfo := range nodeAPeers {
		if peerInfo.Addr == nodeAddr {
			addrFound = true
			break
		}
	}

	if !addrFound {
		t.Fatal("nodeA not connected to nodeB")
	}
}

func testConnectNode(r *Harness, t *testing.T) {
//创建新的测试线束。
	harness, err := New(&chaincfg.SimNetParams, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := harness.SetUp(false, 0); err != nil {
		t.Fatalf("unable to complete rpctest setup: %v", err)
	}
	defer harness.TearDown()

//建立从新的本地线束到主线束的P2P连接
//挽具。
	if err := ConnectNode(harness, r); err != nil {
		t.Fatalf("unable to connect local to main harness: %v", err)
	}

//主线束应显示在本地线束的对等列表中，
//反之亦然。
	assertConnectedTo(t, harness, r)
}

func testTearDownAll(t *testing.T) {
//获取当前活动线束的本地副本
//试图把它们全部拆掉。
	initialActiveHarnesses := ActiveHarnesses()

//拆下所有当前活动的线束。
	if err := TearDownAll(); err != nil {
		t.Fatalf("unable to teardown all harnesses: %v", err)
	}

//现在应该完全清除全局测试映射，不使用
//剩余激活的测试线束。
	if len(ActiveHarnesses()) != 0 {
		t.Fatalf("test harnesses still active after TearDownAll")
	}

	for _, harness := range initialActiveHarnesses {
//确保已删除所有测试目录。
		if _, err := os.Stat(harness.testNodeDir); err == nil {
			t.Errorf("created test datadir was not deleted.")
		}
	}
}

func testActiveHarnesses(r *Harness, t *testing.T) {
	numInitialHarnesses := len(ActiveHarnesses())

//创建单个测试线束。
	harness1, err := New(&chaincfg.SimNetParams, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer harness1.TearDown()

//利用上面创建的线束，应检测到单个线束
//作为活性。
	numActiveHarnesses := len(ActiveHarnesses())
	if !(numActiveHarnesses > numInitialHarnesses) {
		t.Fatalf("ActiveHarnesses not updated, should have an " +
			"additional test harness listed.")
	}
}

func testJoinMempools(r *Harness, t *testing.T) {
//断言主测试工具在其mempool中没有事务。
	pooledHashes, err := r.Node.GetRawMempool()
	if err != nil {
		t.Fatalf("unable to get mempool for main test harness: %v", err)
	}
	if len(pooledHashes) != 0 {
		t.Fatal("main test harness mempool not empty")
	}

//只使用Genesis块创建本地测试线束。节点
//将在下面同步，以便相同的事务可以发送到
//没有孤立节点的节点。
	harness, err := New(&chaincfg.SimNetParams, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := harness.SetUp(false, 0); err != nil {
		t.Fatalf("unable to complete rpctest setup: %v", err)
	}
	defer harness.TearDown()

	nodeSlice := []*Harness{r, harness}

//这两个内存池都是空的，因此应该认为是同步的。
//因此，这应该立即返回。
	if err := JoinNodes(nodeSlice, Mempools); err != nil {
		t.Fatalf("unable to join node on mempools: %v", err)
	}

//在主线束中生成一个新地址的CoinBase支出'
//内存池。
	addr, err := r.NewAddress()
	addrScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		t.Fatalf("unable to generate pkscript to addr: %v", err)
	}
	output := wire.NewTxOut(5e8, addrScript)
	testTx, err := r.CreateTransaction([]*wire.TxOut{output}, 10, true)
	if err != nil {
		t.Fatalf("coinbase spend failed: %v", err)
	}
	if _, err := r.Node.SendRawTransaction(testTx, true); err != nil {
		t.Fatalf("send transaction failed: %v", err)
	}

//等待事务出现，以确保两个内存池
//不一样。
	harnessSynced := make(chan struct{})
	go func() {
		for {
			poolHashes, err := r.Node.GetRawMempool()
			if err != nil {
				t.Fatalf("failed to retrieve harness mempool: %v", err)
			}
			if len(poolHashes) > 0 {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}
		harnessSynced <- struct{}{}
	}()
	select {
	case <-harnessSynced:
	case <-time.After(time.Minute):
		t.Fatalf("harness node never received transaction")
	}

//这个select case应该作为goroutine进入默认值。
//应该在joinnodes调用中被阻止。
	poolsSynced := make(chan struct{})
	go func() {
		if err := JoinNodes(nodeSlice, Mempools); err != nil {
			t.Fatalf("unable to join node on mempools: %v", err)
		}
		poolsSynced <- struct{}{}
	}()
	select {
	case <-poolsSynced:
		t.Fatalf("mempools detected as synced yet harness has a new tx")
	default:
	}

//建立从本地线束到主线束的出站连接
//系好安全带，等待链条同步。
	if err := ConnectNode(harness, r); err != nil {
		t.Fatalf("unable to connect harnesses: %v", err)
	}
	if err := JoinNodes(nodeSlice, Blocks); err != nil {
		t.Fatalf("unable to join node on blocks: %v", err)
	}

//将事务发送到本地线束，这将导致同步
//内存池。
	if _, err := harness.Node.SendRawTransaction(testTx, true); err != nil {
		t.Fatalf("send transaction failed: %v", err)
	}

//1分钟后用特殊的超时情况再次选择。这个
//上面的goroutine现在应该在发送到未缓存时被阻止
//通道。发送应立即成功。为了避免
//测试挂起无限期，1分钟超时。
	select {
	case <-poolsSynced:
//跌倒
	case <-time.After(time.Minute):
		t.Fatalf("mempools never detected as synced")
	}
}

func testJoinBlocks(r *Harness, t *testing.T) {
//创建第二个仅包含Genesis块的线束，使其位于后面
//主线束。
	harness, err := New(&chaincfg.SimNetParams, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := harness.SetUp(false, 0); err != nil {
		t.Fatalf("unable to complete rpctest setup: %v", err)
	}
	defer harness.TearDown()

	nodeSlice := []*Harness{r, harness}
	blocksSynced := make(chan struct{})
	go func() {
		if err := JoinNodes(nodeSlice, Blocks); err != nil {
			t.Fatalf("unable to join node on blocks: %v", err)
		}
		blocksSynced <- struct{}{}
	}()

//这个select case应该作为goroutine进入默认值。
//应该在joinnodes调用上被阻止。
	select {
	case <-blocksSynced:
		t.Fatalf("blocks detected as synced yet local harness is behind")
	default:
	}

//将本地线束连接到将同步
//链。
	if err := ConnectNode(harness, r); err != nil {
		t.Fatalf("unable to connect harnesses: %v", err)
	}

//1分钟后用特殊的超时情况再次选择。这个
//上面的goroutine现在应该在发送到未缓存时被阻止
//通道。发送应立即成功。为了避免
//测试挂起无限期，1分钟超时。
	select {
	case <-blocksSynced:
//跌倒
	case <-time.After(time.Minute):
		t.Fatalf("blocks never detected as synced")
	}
}

func testGenerateAndSubmitBlock(r *Harness, t *testing.T) {
//生成一些测试开销事务。
	addr, err := r.NewAddress()
	if err != nil {
		t.Fatalf("unable to generate new address: %v", err)
	}
	pkScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		t.Fatalf("unable to create script: %v", err)
	}
	output := wire.NewTxOut(btcutil.SatoshiPerBitcoin, pkScript)

	const numTxns = 5
	txns := make([]*btcutil.Tx, 0, numTxns)
	for i := 0; i < numTxns; i++ {
		tx, err := r.CreateTransaction([]*wire.TxOut{output}, 10, true)
		if err != nil {
			t.Fatalf("unable to create tx: %v", err)
		}

		txns = append(txns, btcutil.NewTx(tx))
	}

//现在用默认块版本生成一个块，零
//外出时间。
	block, err := r.GenerateAndSubmitBlock(txns, -1, time.Time{})
	if err != nil {
		t.Fatalf("unable to generate block: %v", err)
	}

//确保包含所有创建的事务，并且
//块版本已正确设置为默认值。
	numBlocksTxns := len(block.Transactions())
	if numBlocksTxns != numTxns+1 {
		t.Fatalf("block did not include all transactions: "+
			"expected %v, got %v", numTxns+1, numBlocksTxns)
	}
	blockVersion := block.MsgBlock().Header.Version
	if blockVersion != BlockVersion {
		t.Fatalf("block version is not default: expected %v, got %v",
			BlockVersion, blockVersion)
	}

//接下来生成一个带有“非标准”块版本的块以及
//上一个块的时间戳后一分钟的时间戳。
	timestamp := block.MsgBlock().Header.Timestamp.Add(time.Minute)
	targetBlockVersion := int32(1337)
	block, err = r.GenerateAndSubmitBlock(nil, targetBlockVersion, timestamp)
	if err != nil {
		t.Fatalf("unable to generate block: %v", err)
	}

//最后确保设置了所需的块版本和时间戳。
//适当地。
	header := block.MsgBlock().Header
	blockVersion = header.Version
	if blockVersion != targetBlockVersion {
		t.Fatalf("block version mismatch: expected %v, got %v",
			targetBlockVersion, blockVersion)
	}
	if !timestamp.Equal(header.Timestamp) {
		t.Fatalf("header time stamp mismatch: expected %v, got %v",
			timestamp, header.Timestamp)
	}
}

func testGenerateAndSubmitBlockWithCustomCoinbaseOutputs(r *Harness,
	t *testing.T) {
//生成一些测试开销事务。
	addr, err := r.NewAddress()
	if err != nil {
		t.Fatalf("unable to generate new address: %v", err)
	}
	pkScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		t.Fatalf("unable to create script: %v", err)
	}
	output := wire.NewTxOut(btcutil.SatoshiPerBitcoin, pkScript)

	const numTxns = 5
	txns := make([]*btcutil.Tx, 0, numTxns)
	for i := 0; i < numTxns; i++ {
		tx, err := r.CreateTransaction([]*wire.TxOut{output}, 10, true)
		if err != nil {
			t.Fatalf("unable to create tx: %v", err)
		}

		txns = append(txns, btcutil.NewTx(tx))
	}

//现在用默认的块版本生成一个块，一个零
//时间和燃烧输出。
	block, err := r.GenerateAndSubmitBlockWithCustomCoinbaseOutputs(txns,
		-1, time.Time{}, []wire.TxOut{{
			Value:    0,
			PkScript: []byte{},
		}})
	if err != nil {
		t.Fatalf("unable to generate block: %v", err)
	}

//确保包含所有创建的事务，并且
//块版本已正确设置为默认值。
	numBlocksTxns := len(block.Transactions())
	if numBlocksTxns != numTxns+1 {
		t.Fatalf("block did not include all transactions: "+
			"expected %v, got %v", numTxns+1, numBlocksTxns)
	}
	blockVersion := block.MsgBlock().Header.Version
	if blockVersion != BlockVersion {
		t.Fatalf("block version is not default: expected %v, got %v",
			BlockVersion, blockVersion)
	}

//接下来生成一个带有“非标准”块版本的块以及
//上一个块的时间戳后一分钟的时间戳。
	timestamp := block.MsgBlock().Header.Timestamp.Add(time.Minute)
	targetBlockVersion := int32(1337)
	block, err = r.GenerateAndSubmitBlockWithCustomCoinbaseOutputs(nil,
		targetBlockVersion, timestamp, []wire.TxOut{{
			Value:    0,
			PkScript: []byte{},
		}})
	if err != nil {
		t.Fatalf("unable to generate block: %v", err)
	}

//最后确保设置了所需的块版本和时间戳。
//适当地。
	header := block.MsgBlock().Header
	blockVersion = header.Version
	if blockVersion != targetBlockVersion {
		t.Fatalf("block version mismatch: expected %v, got %v",
			targetBlockVersion, blockVersion)
	}
	if !timestamp.Equal(header.Timestamp) {
		t.Fatalf("header time stamp mismatch: expected %v, got %v",
			timestamp, header.Timestamp)
	}
}

func testMemWalletReorg(r *Harness, t *testing.T) {
//创建一个新的线束，我们将使用主线束强制
//重新组织这个本地工具。
	harness, err := New(&chaincfg.SimNetParams, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := harness.SetUp(true, 5); err != nil {
		t.Fatalf("unable to complete rpctest setup: %v", err)
	}
	defer harness.TearDown()

//这个线束的内部钱包现在应该有250个BTC。
	expectedBalance := btcutil.Amount(250 * btcutil.SatoshiPerBitcoin)
	walletBalance := harness.ConfirmedBalance()
	if expectedBalance != walletBalance {
		t.Fatalf("wallet balance incorrect: expected %v, got %v",
			expectedBalance, walletBalance)
	}

//现在将本地线束连接到主线束，然后等待
//他们的链条要同步。
	if err := ConnectNode(harness, r); err != nil {
		t.Fatalf("unable to connect harnesses: %v", err)
	}
	nodeSlice := []*Harness{r, harness}
	if err := JoinNodes(nodeSlice, Blocks); err != nil {
		t.Fatalf("unable to join node on blocks: %v", err)
	}

//原来的钱包现在应该有一个0 BTC的余额作为它的整个
//链条应该被分解，以利于主线束'
//链。
	expectedBalance = btcutil.Amount(0)
	walletBalance = harness.ConfirmedBalance()
	if expectedBalance != walletBalance {
		t.Fatalf("wallet balance incorrect: expected %v, got %v",
			expectedBalance, walletBalance)
	}
}

func testMemWalletLockedOutputs(r *Harness, t *testing.T) {
//获取钱包的初始余额。
	startingBalance := r.ConfirmedBalance()

//首先，创建一个花费一些输出的已签名事务。
	addr, err := r.NewAddress()
	if err != nil {
		t.Fatalf("unable to generate new address: %v", err)
	}
	pkScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		t.Fatalf("unable to create script: %v", err)
	}
	outputAmt := btcutil.Amount(50 * btcutil.SatoshiPerBitcoin)
	output := wire.NewTxOut(int64(outputAmt), pkScript)
	tx, err := r.CreateTransaction([]*wire.TxOut{output}, 10, true)
	if err != nil {
		t.Fatalf("unable to create transaction: %v", err)
	}

//当前的钱包余额现在至少应减少50 BTC。
//（费用核算）比期间余额
	currentBalance := r.ConfirmedBalance()
	if !(currentBalance <= startingBalance-outputAmt) {
		t.Fatalf("spent outputs not locked: previous balance %v, "+
			"current balance %v", startingBalance, currentBalance)
	}

//现在解锁无负载签名的所有已用输入
//交易。现在的余额应该正好是
//starting balance.
	r.UnlockOutputs(tx.TxIn)
	currentBalance = r.ConfirmedBalance()
	if currentBalance != startingBalance {
		t.Fatalf("current and starting balance should now match: "+
			"expected %v, got %v", startingBalance, currentBalance)
	}
}

var harnessTestCases = []HarnessTestCase{
	testSendOutputs,
	testConnectNode,
	testActiveHarnesses,
	testJoinBlocks,
testJoinMempools, //取决于TestJoinBlocks的结果
	testGenerateAndSubmitBlock,
	testGenerateAndSubmitBlockWithCustomCoinbaseOutputs,
	testMemWalletReorg,
	testMemWalletLockedOutputs,
}

var mainHarness *Harness

const (
	numMatureOutputs = 25
)

func TestMain(m *testing.M) {
	var err error
	mainHarness, err = New(&chaincfg.SimNetParams, nil, nil)
	if err != nil {
		fmt.Println("unable to create main harness: ", err)
		os.Exit(1)
	}

//用长度为125的链初始化主挖掘节点，
//提供25个成熟的CoinBase，用于测试
//目的。
	if err = mainHarness.SetUp(true, numMatureOutputs); err != nil {
		fmt.Println("unable to setup test chain: ", err)

//即使线束没有完全安装，它仍然需要
//拆除以确保所有资源，如临时
//目录被清除。错误是故意的
//忽略，因为这已经是一个错误路径，没有其他内容
//不管怎样都可以解决。
		_ = mainHarness.TearDown()
		os.Exit(1)
	}

	exitCode := m.Run()

//清除当前仍在运行的所有激活线束。
	if len(ActiveHarnesses()) > 0 {
		if err := TearDownAll(); err != nil {
			fmt.Println("unable to tear down chain: ", err)
			os.Exit(1)
		}
	}

	os.Exit(exitCode)
}

func TestHarness(t *testing.T) {
//我们应该有（nummatureoutputs*50 btc）成熟的不可依赖的
//输出。
	expectedBalance := btcutil.Amount(numMatureOutputs * 50 * btcutil.SatoshiPerBitcoin)
	harnessBalance := mainHarness.ConfirmedBalance()
	if harnessBalance != expectedBalance {
		t.Fatalf("expected wallet balance of %v instead have %v",
			expectedBalance, harnessBalance)
	}

//当前提示的高度应为nummatureoutputs加上
//CoinBase到期所需的块数。
	nodeInfo, err := mainHarness.Node.GetInfo()
	if err != nil {
		t.Fatalf("unable to execute getinfo on node: %v", err)
	}
	expectedChainHeight := numMatureOutputs + uint32(mainHarness.ActiveNet.CoinbaseMaturity)
	if uint32(nodeInfo.Blocks) != expectedChainHeight {
		t.Errorf("Chain height is %v, should be %v",
			nodeInfo.Blocks, expectedChainHeight)
	}

	for _, testCase := range harnessTestCases {
		testCase(mainHarness, t)
	}

	testTearDownAll(t)
}
