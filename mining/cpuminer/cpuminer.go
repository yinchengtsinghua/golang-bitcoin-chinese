
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

package cpuminer

import (
	"errors"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/mining"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

const (
//Max是一个块头中的最大值。
maxNonce = ^uint32(0) //2 ^ 32—1

//maxextrance是coinbase中使用的最大值。
//交易可以。
maxExtraNonce = ^uint64(0) //2 ^ 64—1

//hpsupdatesecs是每种情况之间等待的秒数。
//每秒更新哈希数监视器。
	hpsUpdateSecs = 10

//HashUpdateSec是每个工作进程在这段时间内等待的秒数。
//用已完成的哈希数通知速度监视器
//他们在积极寻找解决方案。这样做是为了
//减少必须执行的工作人员之间的同步量
//记录每秒的哈希值。
	hashUpdateSecs = 15
)

var (
//Debug TnWorkWord是用于采矿的默认工人数量。
//并基于处理器核心的数量。这有助于确保
//系统在重载下保持合理的响应。
	defaultNumWorkers = uint32(runtime.NumCPU())
)

//config是包含cpu miner配置的描述符。
type Config struct {
//chainParams标识CPU矿工的链参数
//与关联。
	ChainParams *chaincfg.Params

//BlockTemplateGenerator标识要用于
//生成矿工将尝试解决的块模板。
	BlockTemplateGenerator *mining.BlkTmplGenerator

//miningaddrs是用于生成的
//阻碍。每个生成的块将随机选择其中一个。
	MiningAddrs []btcutil.Address

//processBlock定义用任何已解块调用的函数。
//它通常必须通过同一组
//与来自网络的任何其他块一样的规则和处理。
	ProcessBlock func(*btcutil.Block, blockchain.BehaviorFlags) (bool, error)

//ConnectedCount定义用于获取其他多少个函数的函数
//与服务器连接的对等方。这是自动的
//用于确定是否应尝试
//采矿。这是有用的，因为在没有挖掘点的情况下没有挖掘点
//因为没有人可以发送任何
//找到块到。
	ConnectedCount func() int32

//iscurrent定义用于获取
//区块链是当前的。这由自动持久性使用
//用于确定是否应尝试挖掘的挖掘例程。
//这很有用，因为如果链是
//不是最新的，因为任何求解块都将位于侧链上，并且
//无论如何都是孤儿。
	IsCurrent func() bool
}

//cpuminer提供了使用CPU解决块（挖掘）的工具
//a concurrency-safe manner.  It consists of two main goroutines -- a speed
//监控和控制生成和解决的工人goroutine
//阻碍。可以通过setmaxgoroutine设置goroutine的数目
//函数，但默认值基于
//系统通常是足够的。
type CPUMiner struct {
	sync.Mutex
	g                 *mining.BlkTmplGenerator
	cfg               Config
	numWorkers        uint32
	started           bool
	discreteMining    bool
	submitBlockLock   sync.Mutex
	wg                sync.WaitGroup
	workerWg          sync.WaitGroup
	updateNumWorkers  chan struct{}
	queryHashesPerSec chan float64
	updateHashes      chan uint64
	speedMonitorQuit  chan struct{}
	quit              chan struct{}
}

//speedmonitor处理跟踪每秒挖掘的哈希数
//进程正在执行。它必须像野人一样运作。
func (m *CPUMiner) speedMonitor() {
	log.Tracef("CPU miner speed monitor started")

	var hashesPerSec float64
	var totalHashes uint64
	ticker := time.NewTicker(time.Second * hpsUpdateSecs)
	defer ticker.Stop()

out:
	for {
		select {
//定期更新工人的哈希数
//表演过。
		case numHashes := <-m.updateHashes:
			totalHashes += numHashes

//每秒更新哈希的时间。
		case <-ticker.C:
			curHashesPerSec := float64(totalHashes) / hpsUpdateSecs
			if hashesPerSec == 0 {
				hashesPerSec = curHashesPerSec
			}
			hashesPerSec = (hashesPerSec + curHashesPerSec) / 2
			totalHashes = 0
			if hashesPerSec != 0 {
				log.Debugf("Hash speed: %6.0f kilohashes/s",
					hashesPerSec/1000)
			}

//Request for the number of hashes per second.
		case m.queryHashesPerSec <- hashesPerSec:
//无事可做。

		case <-m.speedMonitorQuit:
			break out
		}
	}

	m.wg.Done()
	log.Tracef("CPU miner speed monitor done")
}

//SubmitBlock在确保所有块都通过后将传递的块提交给网络
//共识验证规则。
func (m *CPUMiner) submitBlock(block *btcutil.Block) bool {
	m.submitBlockLock.Lock()
	defer m.submitBlockLock.Unlock()

//确保块没有过时，因为可能会出现新块
//找到解决方案时。通常情况是
//检测到并停止旧块上的所有工作以开始工作
//一个新的块，但检查只定期进行，因此
//可能在两者之间找到并提交了块。
	msgBlock := block.MsgBlock()
	if !msgBlock.Header.PrevBlock.IsEqual(&m.g.BestSnapshot().Hash) {
		log.Debugf("Block submitted via CPU miner with previous "+
			"block %s is stale", msgBlock.Header.PrevBlock)
		return false
	}

//使用与来自其他块相同的规则处理此块
//节点。这将反过来像正常一样将其中继到网络。
	isOrphan, err := m.cfg.ProcessBlock(block, blockchain.BFNone)
	if err != nil {
//除违反规则外，任何其他都是意外错误，
//因此，将该错误记录为内部错误。
		if _, ok := err.(blockchain.RuleError); !ok {
			log.Errorf("Unexpected error while processing "+
				"block submitted via CPU miner: %v", err)
			return false
		}

		log.Debugf("Block submitted via CPU miner rejected: %v", err)
		return false
	}
	if isOrphan {
		log.Debugf("Block submitted via CPU miner is an orphan")
		return false
	}

//这个街区被接受了。
	coinbaseTx := block.MsgBlock().Transactions[0].TxOut[0]
	log.Infof("Block submitted via CPU miner accepted (hash %s, "+
		"amount %v)", block.Hash(), btcutil.Amount(coinbaseTx.Value))
	return true
}

//SolveBlock尝试查找nonce、extra nonce和
//当前时间戳，使传递的块哈希值小于
//目标难度。时间戳会定期更新并通过
//在这个过程中，用所有调整来修改块。这意味着
//当函数返回true时，块就可以提交了。
//
//当触发
//过时的块，如新块出现或定期出现
//新的事务和足够的时间已经过去，但没有找到解决方案。
func (m *CPUMiner) solveBlock(msgBlock *wire.MsgBlock, blockHeight int32,
	ticker *time.Ticker, quit chan struct{}) bool {

//为此块模板选择一个随机的额外nonce偏移量，然后
//工人。
	enOffset, err := wire.RandomUint64()
	if err != nil {
		log.Errorf("Unexpected error while generating random "+
			"extra nonce offset: %v", err)
		enOffset = 0
	}

//创建一些方便变量。
	header := &msgBlock.Header
	targetDifficulty := blockchain.CompactToBig(header.Bits)

//初始状态。
	lastGenerated := time.Now()
	lastTxUpdate := m.g.TxSource().LastUpdated()
	hashesCompleted := uint64(0)

//注意，整个额外的nonce范围是迭代的，偏移量是
//根据溢出将环绕0的事实添加为
//由GO规范提供。
	for extraNonce := uint64(0); extraNonce < maxExtraNonce; extraNonce++ {
//用
//通过重新生成coinbase脚本和
//将merkle根设置为新值。
		m.g.UpdateExtraNonce(msgBlock, blockHeight, extraNonce+enOffset)

//Search through the entire nonce range for a solution while
//定期检查早期退出和过时块
//条件以及速度监视器的更新。
		for i := uint32(0); i <= maxNonce; i++ {
			select {
			case <-quit:
				return false

			case <-ticker.C:
				m.updateHashes <- hashesCompleted
				hashesCompleted = 0

//如果最佳块为
//改变了。
				best := m.g.BestSnapshot()
				if !header.PrevBlock.IsEqual(&best.Hash) {
					return false
				}

//如果内存池
//自块模板
//已生成，并且至少有一个
//分钟。
				if lastTxUpdate != m.g.TxSource().LastUpdated() &&
					time.Now().After(lastGenerated.Add(time.Minute)) {

					return false
				}

				m.g.UpdateBlockTime(msgBlock)

			default:
//非阻塞选择通过
			}

//更新NANCE和散列块头。各
//hash实际上是一个双sha256（两个hash），所以
//增加每个哈希的完成数量
//相应地尝试。
			header.Nonce = i
			hash := header.BlockHash()
			hashesCompleted += 2

//当新的块散列值小于
//比目标难度大。哎呀！
			if blockchain.HashToBig(&hash).Cmp(targetDifficulty) <= 0 {
				m.updateHashes <- hashesCompleted
				return true
			}
		}
	}

	return false
}

//GenerateBlocks是由MiningWorkerController控制的工人。
//它自包含在创建块模板并尝试解决
//当检测到它在执行过时的工作和响应时，
//相应地，生成一个新的块模板。当一个块被解决时，
//提交。
//
//它必须像野人一样运作。
func (m *CPUMiner) generateBlocks(quit chan struct{}) {
	log.Tracef("Starting generate blocks worker")

//启动一个断续器，用于对陈旧的工作进行信号检查，以及
//速度监视器的更新。
	ticker := time.NewTicker(time.Second * hashUpdateSecs)
	defer ticker.Stop()
out:
	for {
//当矿工停止时退出。
		select {
		case <-quit:
			break out
		default:
//非阻塞选择通过
		}

//等待，直到与至少一个其他对等机建立连接
//因为无法中继找到的块或接收
//没有连接的对等端时要处理的事务。
		if m.cfg.ConnectedCount() == 0 {
			time.Sleep(time.Second)
			continue
		}

//在链出现之前没有必要寻找解决方案
//同步。另外，抓取与块相同的锁
//提交，因为当前块将更改并且
//否则，最终将在
//正在变旧的块。
		m.submitBlockLock.Lock()
		curHeight := m.g.BestSnapshot().Height
		if curHeight != 0 && !m.cfg.IsCurrent() {
			m.submitBlockLock.Unlock()
			time.Sleep(time.Second)
			continue
		}

//随机选择付款地址。
		rand.Seed(time.Now().UnixNano())
		payToAddr := m.cfg.MiningAddrs[rand.Intn(len(m.cfg.MiningAddrs))]

//使用可用事务创建新的块模板
//在内存池中作为事务源
//包括在块中。
		template, err := m.g.NewBlockTemplate(payToAddr)
		m.submitBlockLock.Unlock()
		if err != nil {
			errStr := fmt.Sprintf("Failed to create new block "+
				"template: %v", err)
			log.Errorf(errStr)
			continue
		}

//尝试解决该块。功能将提前退出
//当触发过时块的条件为false时，
//可以生成新的块模板。当回报是
//如果找到了解决方案，请提交已解决的块。
		if m.solveBlock(template.Block, curHeight+1, ticker, quit) {
			block := btcutil.NewBlock(template.Block)
			m.submitBlock(block)
		}
	}

	m.workerWg.Done()
	log.Tracef("Generate blocks worker done")
}

//MiningWorkerController启动用于
//生成块模板并解决它们。它还提供了
//动态调整正在运行的worker goroutine的数量。
//
//它必须像野人一样运作。
func (m *CPUMiner) miningWorkerController() {
//LaunchWorkers将通用代码分组以启动指定数量的
//用于生成块的工人。
	var runningWorkers []chan struct{}
	launchWorkers := func(numWorkers uint32) {
		for i := uint32(0); i < numWorkers; i++ {
			quit := make(chan struct{})
			runningWorkers = append(runningWorkers, quit)

			m.workerWg.Add(1)
			go m.generateBlocks(quit)
		}
	}

//默认启动当前工人数量。
	runningWorkers = make([]chan struct{}, 0, m.numWorkers)
	launchWorkers(m.numWorkers)

out:
	for {
		select {
//更新正在运行的工人的数量。
		case <-m.updateNumWorkers:
//没有变化。
			numRunning := uint32(len(runningWorkers))
			if m.numWorkers == numRunning {
				continue
			}

//添加新员工。
			if m.numWorkers > numRunning {
				launchWorkers(m.numWorkers - numRunning)
				continue
			}

//向最近创建的goroutine发出退出信号。
			for i := numRunning - 1; i >= m.numWorkers; i-- {
				close(runningWorkers[i])
				runningWorkers[i] = nil
				runningWorkers = runningWorkers[:i]
			}

		case <-m.quit:
			for _, quit := range runningWorkers {
				close(quit)
			}
			break out
		}
	}

//等待所有工人关闭，停止速度监视器，因为
//他们依赖于能够向其发送更新。
	m.workerWg.Wait()
	close(m.speedMonitorQuit)
	m.wg.Done()
}

//Start开始CPU挖掘进程以及用于
//跟踪哈希度量。当CPU矿工
//已启动将不起作用。
//
//此函数对于并发访问是安全的。
func (m *CPUMiner) Start() {
	m.Lock()
	defer m.Unlock()

//如果矿工已经在运行或正在运行，则无需执行任何操作。
//离散模式（使用GenerateBlocks）。
	if m.started || m.discreteMining {
		return
	}

	m.quit = make(chan struct{})
	m.speedMonitorQuit = make(chan struct{})
	m.wg.Add(2)
	go m.speedMonitor()
	go m.miningWorkerController()

	m.started = true
	log.Infof("CPU miner started")
}

//通过向所有工人发出信号，优雅地停止采矿过程，并且
//要退出的速度监视器。当CPU矿工没有
//已启动将不起作用。
//
//此函数对于并发访问是安全的。
func (m *CPUMiner) Stop() {
	m.Lock()
	defer m.Unlock()

//如果矿工当前未运行或正在运行，则无需执行任何操作。
//离散模式（使用GenerateBlocks）。
	if !m.started || m.discreteMining {
		return
	}

	close(m.quit)
	m.wg.Wait()
	m.started = false
	log.Infof("CPU miner stopped")
}

//ismining返回CPU矿工是否已启动并且
//因此进行采矿。
//
//此函数对于并发访问是安全的。
func (m *CPUMiner) IsMining() bool {
	m.Lock()
	defer m.Unlock()

	return m.started
}

//hassespersecond返回挖掘进程每秒的哈希数
//正在执行。如果矿工当前未运行，则返回0。
//
//此函数对于并发访问是安全的。
func (m *CPUMiner) HashesPerSecond() float64 {
	m.Lock()
	defer m.Unlock()

//如果矿工当前未运行，则不执行任何操作。
	if !m.started {
		return 0
	}

	return <-m.queryHashesPerSec
}

//setNumWorkers设置要创建的求解块的工人数。任何
//负值将导致使用默认数量的工人，即
//基于系统中处理器核心的数量。0的值将
//导致所有CPU挖掘停止。
//
//此函数对于并发访问是安全的。
func (m *CPUMiner) SetNumWorkers(numWorkers int32) {
	if numWorkers == 0 {
		m.Stop()
	}

//在第一次检查后才锁定，因为Stop本身
//锁定。
	m.Lock()
	defer m.Unlock()

//如果提供的值为负数，则使用默认值。
	if numWorkers < 0 {
		m.numWorkers = defaultNumWorkers
	} else {
		m.numWorkers = uint32(numWorkers)
	}

//当矿工已经在运行时，通知控制器
//变化。
	if m.started {
		m.updateNumWorkers <- struct{}{}
	}
}

//numWorkers返回正在运行以解算块的工人数。
//
//此函数对于并发访问是安全的。
func (m *CPUMiner) NumWorkers() int32 {
	m.Lock()
	defer m.Unlock()

	return int32(m.numWorkers)
}

//GenerateBlocks生成请求的块数。它是自我
//它创建块模板并尝试在
//在执行过时工作时进行检测，并通过
//生成新的块模板。当一个块被解决时，它被提交。
//函数返回生成块的哈希列表。
func (m *CPUMiner) GenerateNBlocks(n uint32) ([]*chainhash.Hash, error) {
	m.Lock()

//如果服务器已在挖掘，则响应并返回错误。
	if m.started || m.discreteMining {
		m.Unlock()
		return nil, errors.New("Server is already CPU mining. Please call " +
			"`setgenerate 0` before calling discrete `generate` commands.")
	}

	m.started = true
	m.discreteMining = true

	m.speedMonitorQuit = make(chan struct{})
	m.wg.Add(1)
	go m.speedMonitor()

	m.Unlock()

	log.Tracef("Generating %d blocks", n)

	i := uint32(0)
	blockHashes := make([]*chainhash.Hash, n)

//启动一个断续器，用于对陈旧的工作进行信号检查，以及
//速度监视器的更新。
	ticker := time.NewTicker(time.Second * hashUpdateSecs)
	defer ticker.Stop()

	for {
//读取updatenumworkers，以防有人在尝试“setgenerate”时
//我们正在生成。我们可以忽略它作为“generate”rpc调用
//使用1名工人。
		select {
		case <-m.updateNumWorkers:
		default:
		}

//获取用于块提交的锁，因为当前块将
//正在更改，否则最终将生成一个新块
//正在过时的块上的模板。
		m.submitBlockLock.Lock()
		curHeight := m.g.BestSnapshot().Height

//随机选择付款地址。
		rand.Seed(time.Now().UnixNano())
		payToAddr := m.cfg.MiningAddrs[rand.Intn(len(m.cfg.MiningAddrs))]

//使用可用事务创建新的块模板
//在内存池中作为事务源
//包括在块中。
		template, err := m.g.NewBlockTemplate(payToAddr)
		m.submitBlockLock.Unlock()
		if err != nil {
			errStr := fmt.Sprintf("Failed to create new block "+
				"template: %v", err)
			log.Errorf(errStr)
			continue
		}

//尝试解决该块。功能将提前退出
//当触发过时块的条件为false时，
//可以生成新的块模板。当回报是
//如果找到了解决方案，请提交已解决的块。
		if m.solveBlock(template.Block, curHeight+1, ticker, nil) {
			block := btcutil.NewBlock(template.Block)
			m.submitBlock(block)
			blockHashes[i] = block.Hash()
			i++
			if i == n {
				log.Tracef("Generated %d blocks", i)
				m.Lock()
				close(m.speedMonitorQuit)
				m.wg.Wait()
				m.started = false
				m.discreteMining = false
				m.Unlock()
				return blockHashes, nil
			}
		}
	}
}

//new返回所提供配置的新CPU矿工实例。
//使用Start开始挖掘过程。请参阅cpuminer的文档
//键入以获取详细信息。
func New(cfg *Config) *CPUMiner {
	return &CPUMiner{
		g:                 cfg.BlockTemplateGenerator,
		cfg:               *cfg,
		numWorkers:        defaultNumWorkers,
		updateNumWorkers:  make(chan struct{}),
		queryHashesPerSec: make(chan float64),
		updateHashes:      make(chan uint64),
	}
}
