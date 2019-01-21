
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2016-2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package rpctest

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

const (
//这些常量定义最小和最大p2p和rpc端口
//测试线束使用的编号。最小端口是包含的，而
//最大端口是独占的。
	minPeerPort = 10000
	maxPeerPort = 35000
	minRPCPort  = maxPeerPort
	maxRPCPort  = 60000

//BlockVersion是生成时使用的默认块版本
//阻碍。
	BlockVersion = 4
)

var (
//当前活动测试节点数。
	numTestInstances = 0

//processID是当前正在运行的进程的进程ID。它是
//用于在启动RPC时根据它计算端口
//线束。其目的是允许多个进程运行
//并行，无端口冲突。
//
//但是应该注意的是，仍然存在一些小的可能性
//由于其他进程，将发生端口冲突
//运行或仅仅是由于进程ID上的星对齐。
	processID = os.Getpid()

//TestInstances是用于跟踪
//所有激活的测试线束。此全局可用于执行
//各种“连接”，测试后关闭多个激活线束，
//等。
	testInstances = make(map[string]*Harness)

//用于表示对上述声明变量的并发访问。
	harnessStateMtx sync.RWMutex
)

//HarnessTestCase表示使用
//从安全带到运动功能。
type HarnessTestCase func(r *Harness, t *testing.T)

//Harness完全封装了活动的BTCD进程，以提供统一的
//用于创建涉及BTCD的RPC驱动集成测试的平台。这个
//活动的btcd节点通常在simnet模式下运行，以允许
//易于生成测试区块链。活动的BTCD进程已完全
//由处理必要初始化和拆卸的线束管理
//以及由此创建的任何临时目录。
//可以同时运行多个线束实例，以便
//测试涉及多个节点的复杂场景。安全带也
//包括一个内存钱包，以简化各种类型的测试。
type Harness struct {
//activenet是该工具所属区块链的参数。
//去。
	ActiveNet *chaincfg.Params

	Node     *rpcclient.Client
	node     *node
	handlers *rpcclient.NotificationHandlers

	wallet *memWallet

	testNodeDir    string
	maxConnRetries int
	nodeNum        int

	sync.Mutex
}

//new创建并初始化rpc测试线束的新实例。
//或者，可以传递WebSocket处理程序和指定的配置。
//如果传递了nil配置，则默认配置为
//使用。
//
//注意：此函数对于并发访问是安全的。
func New(activeNet *chaincfg.Params, handlers *rpcclient.NotificationHandlers,
	extraArgs []string) (*Harness, error) {

	harnessStateMtx.Lock()
	defer harnessStateMtx.Unlock()

//根据提供的
//链参数。
	switch activeNet.Net {
	case wire.MainNet:
//没有额外的标志，因为mainnet是默认的
	case wire.TestNet3:
		extraArgs = append(extraArgs, "--testnet")
	case wire.TestNet:
		extraArgs = append(extraArgs, "--regtest")
	case wire.SimNet:
		extraArgs = append(extraArgs, "--simnet")
	default:
		return nil, fmt.Errorf("rpctest.New must be called with one " +
			"of the supported chain networks")
	}

	testDir, err := baseDir()
	if err != nil {
		return nil, err
	}

	harnessID := strconv.Itoa(numTestInstances)
	nodeTestData, err := ioutil.TempDir(testDir, "harness-"+harnessID)
	if err != nil {
		return nil, err
	}

	certFile := filepath.Join(nodeTestData, "rpc.cert")
	keyFile := filepath.Join(nodeTestData, "rpc.key")
	if err := genCertPair(certFile, keyFile); err != nil {
		return nil, err
	}

	wallet, err := newMemWallet(activeNet, uint32(numTestInstances))
	if err != nil {
		return nil, err
	}

	miningAddr := fmt.Sprintf("--miningaddr=%s", wallet.coinbaseAddr)
	extraArgs = append(extraArgs, miningAddr)

	config, err := newConfig("rpctest", certFile, keyFile, extraArgs)
	if err != nil {
		return nil, err
	}

//生成p2p+rpc侦听地址。
	config.listen, config.rpcListen = generateListeningAddresses()

//创建绑定到simnet的测试节点。
	node, err := newNode(config, nodeTestData)
	if err != nil {
		return nil, err
	}

	nodeNum := numTestInstances
	numTestInstances++

	if handlers == nil {
		handlers = &rpcclient.NotificationHandlers{}
	}

//如果onfilteredblock已连接、已断开连接回调的处理程序
//已设置回调，然后创建包装回调
//执行当前注册的回调和mem wallet的
//回调。
	if handlers.OnFilteredBlockConnected != nil {
		obc := handlers.OnFilteredBlockConnected
		handlers.OnFilteredBlockConnected = func(height int32, header *wire.BlockHeader, filteredTxns []*btcutil.Tx) {
			wallet.IngestBlock(height, header, filteredTxns)
			obc(height, header, filteredTxns)
		}
	} else {
//否则，我们可以自己声明回调。
		handlers.OnFilteredBlockConnected = wallet.IngestBlock
	}
	if handlers.OnFilteredBlockDisconnected != nil {
		obd := handlers.OnFilteredBlockDisconnected
		handlers.OnFilteredBlockDisconnected = func(height int32, header *wire.BlockHeader) {
			wallet.UnwindBlock(height, header)
			obd(height, header)
		}
	} else {
		handlers.OnFilteredBlockDisconnected = wallet.UnwindBlock
	}

	h := &Harness{
		handlers:       handlers,
		node:           node,
		maxConnRetries: 20,
		testNodeDir:    nodeTestData,
		ActiveNet:      activeNet,
		nodeNum:        nodeNum,
		wallet:         wallet,
	}

//在包级别内跟踪这个新创建的测试实例
//所有活动测试实例的全局映射。
	testInstances[h.testNodeDir] = h

	return h, nil
}

//安装程序初始化RPC测试状态。初始化包括：启动
//Simnet节点，创建WebSockets客户端并连接到已启动的
//节点，最后：可以选择生成和提交带有
//成熟的coinbase输出的可配置数量coinbase输出。
//
//注意：此方法和Teardown应该始终从同一个
//因为它们不是同时安全的。
func (h *Harness) SetUp(createTestChain bool, numMatureOutputs uint32) error {
//启动BTCD节点本身。这将产生一个新的过程
//管理
	if err := h.node.start(); err != nil {
		return err
	}
	if err := h.connectRPCClient(); err != nil {
		return err
	}

	h.wallet.Start()

//筛选支付给与
//钱包。
	filterAddrs := []btcutil.Address{h.wallet.coinbaseAddr}
	if err := h.Node.LoadTxFilter(true, filterAddrs, nil); err != nil {
		return err
	}

//确保BTCD为每一个新客户正确发送我们注册的回电。
//块。否则，Memwallet将无法正常工作。
	if err := h.Node.NotifyBlocks(); err != nil {
		return err
	}

//使用所需数量的成熟coinbase创建测试链
//输出。
	if createTestChain && numMatureOutputs != 0 {
		numToGenerate := (uint32(h.ActiveNet.CoinbaseMaturity) +
			numMatureOutputs)
		_, err := h.Node.Generate(numToGenerate)
		if err != nil {
			return err
		}
	}

//阻止，直到钱包完全同步到主屏幕的顶端
//链。
	_, height, err := h.Node.GetBestBlock()
	if err != nil {
		return err
	}
	ticker := time.NewTicker(time.Millisecond * 100)
	for range ticker.C {
		walletHeight := h.wallet.SyncedHeight()
		if walletHeight == height {
			break
		}
	}
	ticker.Stop()

	return nil
}

//TearDown停止正在运行的RPC测试实例。所有创建的流程都是
//删除了临时目录。
//
//必须在保持线束状态互斥（用于写入）的情况下调用此函数。
func (h *Harness) tearDown() error {
	if h.Node != nil {
		h.Node.Shutdown()
	}

	if err := h.node.shutdown(); err != nil {
		return err
	}

	if err := os.RemoveAll(h.testNodeDir); err != nil {
		return err
	}

	delete(testInstances, h.testNodeDir)

	return nil
}

//TearDown停止正在运行的RPC测试实例。所有创建的流程都是
//删除了临时目录。
//
//注意：此方法和设置应始终从同一goroutine调用
//因为它们不是同时安全的。
func (h *Harness) TearDown() error {
	harnessStateMtx.Lock()
	defer harnessStateMtx.Unlock()

	return h.tearDown()
}

//ConnectRpcClient尝试与创建的BTCD建立RPC连接
//属于此线束实例的进程。如果初始连接
//尝试失败，此函数将重试h.maxconnretries次，后退
//随后尝试之间的时间。如果在h.maxconnretries尝试之后，
//我们无法建立连接，此函数返回
//错误。
func (h *Harness) connectRPCClient() error {
	var client *rpcclient.Client
	var err error

	rpcConf := h.node.config.rpcConnConfig()
	for i := 0; i < h.maxConnRetries; i++ {
		if client, err = rpcclient.New(&rpcConf, h.handlers); err != nil {
			time.Sleep(time.Duration(i) * 50 * time.Millisecond)
			continue
		}
		break
	}

	if client == nil {
		return fmt.Errorf("connection timeout")
	}

	h.Node = client
	h.wallet.SetRPCClient(client)
	return nil
}

//newaddress返回一个新地址，该地址可由线束的内部
//钱包。
//
//此函数对于并发访问是安全的。
func (h *Harness) NewAddress() (btcutil.Address, error) {
	return h.wallet.NewAddress()
}

//确认余额返回线束内部确认余额
//钱包。
//
//此函数对于并发访问是安全的。
func (h *Harness) ConfirmedBalance() btcutil.Amount {
	return h.wallet.ConfirmedBalance()
}

//sendOutputs创建、签名并最终广播事务开销
//可利用的成熟的coinbase输出创建新的输出
//根据目标输出。
//
//此函数对于并发访问是安全的。
func (h *Harness) SendOutputs(targetOutputs []*wire.TxOut,
	feeRate btcutil.Amount) (*chainhash.Hash, error) {

	return h.wallet.SendOutputs(targetOutputs, feeRate)
}

//sendOutputswithoutchange创建并发送一个向
//观察通过的费率并忽略更改时的指定输出
//输出。通过的费率应以SAT/B表示。
//
//此函数对于并发访问是安全的。
func (h *Harness) SendOutputsWithoutChange(targetOutputs []*wire.TxOut,
	feeRate btcutil.Amount) (*chainhash.Hash, error) {

	return h.wallet.SendOutputsWithoutChange(targetOutputs, feeRate)
}

//CreateTransaction返回向指定的
//在观察所需费率的同时输出。通过的费率应该是
//以每字节的Satoshis表示。正在创建的事务可以选择
//包括由更改布尔值指示的更改输出。任何未暂停的输出
//选择作为精心编制的事务的输入，将在中标记为不可挂起
//为了避免将来调用此方法可能导致的双倍开销。如果
//创建的交易因任何原因被取消，则所选输入必须
//通过调用解锁输出释放。否则，锁定的输入将不会
//返回到可消费输出池。
//
//此函数对于并发访问是安全的。
func (h *Harness) CreateTransaction(targetOutputs []*wire.TxOut,
	feeRate btcutil.Amount, change bool) (*wire.MsgTx, error) {

	return h.wallet.CreateTransaction(targetOutputs, feeRate, change)
}

//unlockoutputs解锁以前标记为的任何输出
//由于被选中通过
//CreateTransaction方法。
//
//此函数对于并发访问是安全的。
func (h *Harness) UnlockOutputs(inputs []*wire.TxIn) {
	h.wallet.UnlockOutputs(inputs)
}

//rpcconfig返回线束当前的RPC配置。这允许其他
//potential RPC clients created within tests to connect to a given test
//线束实例。
func (h *Harness) RPCConfig() rpcclient.ConnConfig {
	return h.node.config.rpcConnConfig()
}

//p2p address返回线束的p2p侦听地址。这允许潜力
//在测试中创建的用于连接给定测试的对等方（如SPV对等方）
//线束实例。
func (h *Harness) P2PAddress() string {
	return h.node.config.listen
}

//GenerateAndSubmitBlock创建其内容包括传递的
//并将其提交到正在运行的simnet节点。用于生成
//只使用coinbase tx的块，调用方可以简单地传递nil而不是
//要挖掘的事务。此外，自定义块版本可以通过
//呼叫者。块版本为-1表示当前默认块
//应使用版本。未初始化的时间。时间应用于
//如果不希望设置自定义时间，则使用blocktime参数。
//
//此函数对于并发访问是安全的。
func (h *Harness) GenerateAndSubmitBlock(txns []*btcutil.Tx, blockVersion int32,
	blockTime time.Time) (*btcutil.Block, error) {
	return h.GenerateAndSubmitBlockWithCustomCoinbaseOutputs(txns,
		blockVersion, blockTime, []wire.TxOut{})
}

//GenerateAndSubmitBlockWithCustomCoinBaseOutputs创建的块
//内容包括通过的CoinBase输出、事务和提交
//发送到正在运行的simnet节点。对于只使用coinbase tx生成块，
//调用方可以简单地传递nil而不是要挖掘的事务。
//此外，调用程序可以设置自定义块版本。封锁
//of-1表示应使用当前默认块版本。安
//未初始化的时间。如果有时间，则应将时间用于blocktime参数
//不希望设置自定义时间。将添加Mineto输出列表
//to the coinbase; this is not checked for correctness until the block is
//因此，呼叫者有责任确保输出
//是正确的。如果列表是空的，那么coinbase奖励将转到钱包中。
//由安全带管理。
//
//此函数对于并发访问是安全的。
func (h *Harness) GenerateAndSubmitBlockWithCustomCoinbaseOutputs(
	txns []*btcutil.Tx, blockVersion int32, blockTime time.Time,
	mineTo []wire.TxOut) (*btcutil.Block, error) {

	h.Lock()
	defer h.Unlock()

	if blockVersion == -1 {
		blockVersion = BlockVersion
	}

	prevBlockHash, prevBlockHeight, err := h.Node.GetBestBlock()
	if err != nil {
		return nil, err
	}
	mBlock, err := h.Node.GetBlock(prevBlockHash)
	if err != nil {
		return nil, err
	}
	prevBlock := btcutil.NewBlock(mBlock)
	prevBlock.SetHeight(prevBlockHeight)

//创建包含指定事务的新块
	newBlock, err := CreateBlock(prevBlock, txns, blockVersion,
		blockTime, h.wallet.coinbaseAddr, mineTo, h.ActiveNet)
	if err != nil {
		return nil, err
	}

//将块提交到simnet节点。
	if err := h.Node.SubmitBlock(newBlock, nil); err != nil {
		return nil, err
	}

	return newBlock, nil
}

//GenerateListeningAddresses返回两个表示侦听的字符串
//为当前RPC测试指定的地址。如果没有
//已创建测试实例，使用默认端口。否则，为了
//支持多个测试节点同时运行，p2p和rpc端口为
//每次初始化后递增。
func generateListeningAddresses() (string, string) {
	localhost := "127.0.0.1"

	portString := func(minPort, maxPort int) string {
		port := minPort + numTestInstances + ((20 * processID) %
			(maxPort - minPort))
		return strconv.Itoa(port)
	}

	p2p := net.JoinHostPort(localhost, portString(minPeerPort, maxPeerPort))
	rpc := net.JoinHostPort(localhost, portString(minRPCPort, maxRPCPort))
	return p2p, rpc
}

//basedir是所有rpctest文件的temp目录的目录路径。
func baseDir() (string, error) {
	dirPath := filepath.Join(os.TempDir(), "btcd", "rpctest")
	err := os.MkdirAll(dirPath, 0755)
	return dirPath, err
}
