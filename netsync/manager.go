
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package netsync

import (
	"container/list"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcd/mempool"
	peerpkg "github.com/btcsuite/btcd/peer"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

const (
//mininflightBlocks是应为
//在头的请求队列中，请求前的第一个模式
//更多。
	minInFlightBlocks = 10

//MaxRejectedTxns是被拒绝的事务的最大数目
//哈希值存储在内存中。
	maxRejectedTxns = 1000

//MaxRequestedBlocks是请求的最大块数
//哈希值存储在内存中。
	maxRequestedBlocks = wire.MaxInvPerMsg

//MaxRequestedTxns是请求的最大事务数
//哈希值存储在内存中。
	maxRequestedTxns = wire.MaxInvPerMsg
)

//zero hash是零值哈希（全部为零）。它被定义为一种便利。
var zeroHash chainhash.Hash

//newpeermsg表示新连接到块处理程序的对等机。
type newPeerMsg struct {
	peer *peerpkg.Peer
}

//blockmsg将比特币阻塞消息和它来自一起的对等方打包在一起。
//所以块处理程序可以访问这些信息。
type blockMsg struct {
	block *btcutil.Block
	peer  *peerpkg.Peer
	reply chan struct{}
}

//invmsg将比特币inv消息和它来自一起的对等方打包
//所以块处理程序可以访问这些信息。
type invMsg struct {
	inv  *wire.MsgInv
	peer *peerpkg.Peer
}

//headersgmg打包比特币头消息及其来自的对等方
//这样块处理程序就可以访问这些信息。
type headersMsg struct {
	headers *wire.MsgHeaders
	peer    *peerpkg.Peer
}

//donepeermsg表示块处理程序的新断开的对等。
type donePeerMsg struct {
	peer *peerpkg.Peer
}

//txmsg将比特币的tx消息和它来自同一个地方的同伴打包在一起。
//所以块处理程序可以访问这些信息。
type txMsg struct {
	tx    *btcutil.Tx
	peer  *peerpkg.Peer
	reply chan struct{}
}

//getsyncpeermsg是要通过消息通道发送的消息类型
//正在检索当前同步对等机。
type getSyncPeerMsg struct {
	reply chan int32
}

//processBlockResponse是发送到
//进程块消息。
type processBlockResponse struct {
	isOrphan bool
	err      error
}

//processblockmsg是要通过消息通道发送的消息类型
//对于所请求的，将处理一个块。注意，此调用与blockmsg不同
//在上面的blockmsg中，用于来自对等方并且
//额外的处理，而此消息本质上只是一个并发安全的
//在内部块链实例上调用processBlock的方法。
type processBlockMsg struct {
	block *btcutil.Block
	flags blockchain.BehaviorFlags
	reply chan processBlockResponse
}

//isCurrentMsg is a message type to be sent across the message channel for
//请求同步管理器是否相信它与
//当前连接的对等机。
type isCurrentMsg struct {
	reply chan bool
}

//pausemsg是要通过消息通道发送的消息类型
//暂停同步管理器。这有效地为来电者提供了
//对管理器进行独占访问，直到对
//取消暂停频道。
type pauseMsg struct {
	unpause <-chan struct{}
}

//headernode用作链接在一起的标题列表中的节点。
//在检查点之间。
type headerNode struct {
	height int32
	hash   *chainhash.Hash
}

//PeerSyncState存储SyncManager跟踪的其他信息
//关于同伴。
type peerSyncState struct {
	syncCandidate   bool
	requestQueue    []*wire.InvVect
	requestedTxns   map[chainhash.Hash]struct{}
	requestedBlocks map[chainhash.Hash]struct{}
}

//SyncManager用于与对等端通信与块相关的消息。这个
//通过在goroutine中执行start（）启动SyncManager。一旦开始，
//它选择要同步的对等机并开始初始块下载。一旦
//链同步，同步管理器处理传入的块和头
//通知并将新块的通知转发给对等方。
type SyncManager struct {
	peerNotifier   PeerNotifier
	started        int32
	shutdown       int32
	chain          *blockchain.BlockChain
	txMemPool      *mempool.TxPool
	chainParams    *chaincfg.Params
	progressLogger *blockProgressLogger
	msgChan        chan interface{}
	wg             sync.WaitGroup
	quit           chan struct{}

//这些字段只能从blockhandler线程访问
	rejectedTxns    map[chainhash.Hash]struct{}
	requestedTxns   map[chainhash.Hash]struct{}
	requestedBlocks map[chainhash.Hash]struct{}
	syncPeer        *peerpkg.Peer
	peerStates      map[*peerpkg.Peer]*peerSyncState

//以下字段用于头一模式。
	headersFirstMode bool
	headerList       *list.List
	startHeader      *list.Element
	nextCheckpoint   *chaincfg.Checkpoint

//可选的费用估算器。
	feeEstimator *mempool.FeeEstimator
}

//resetheaderstate将头的第一模式状态设置为适合
//正在从新对等机同步。
func (sm *SyncManager) resetHeaderState(newestHash *chainhash.Hash, newestHeight int32) {
	sm.headersFirstMode = false
	sm.headerList.Init()
	sm.startHeader = nil

//当有下一个检查点时，添加一个最新已知的条目
//阻止进入头池。这允许下一个下载的头文件
//证明它与链条正确连接。
	if sm.nextCheckpoint != nil {
		node := headerNode{height: newestHeight, hash: newestHash}
		sm.headerList.PushBack(&node)
	}
}

//findNextHeaderCheckPoint返回通过高度后的下一个检查点。
//当没有高度时，它返回零，因为高度已经
//迟于最终检查点或其他原因，如禁用
//检查点。
func (sm *SyncManager) findNextHeaderCheckpoint(height int32) *chaincfg.Checkpoint {
	checkpoints := sm.chain.Checkpoints()
	if len(checkpoints) == 0 {
		return nil
	}

//如果高度在决赛之后，就没有下一个检查站了
//检查点。
	finalCheckpoint := &checkpoints[len(checkpoints)-1]
	if height >= finalCheckpoint.Height {
		return nil
	}

//找到下一个检查点。
	nextCheckpoint := finalCheckpoint
	for i := len(checkpoints) - 2; i >= 0; i-- {
		if height >= checkpoints[i].Height {
			break
		}
		nextCheckpoint = &checkpoints[i]
	}
	return nextCheckpoint
}

//StartSync将在可用的候选对等中选择最佳对等
//从下载/同步区块链。当同步已在运行时，
//只需返回即可。它也会检查那些不再是
//并根据需要将其删除。
func (sm *SyncManager) startSync() {
//如果已经同步，请立即返回。
	if sm.syncPeer != nil {
		return
	}

//一旦Segwit Soft Fork软件包激活，我们仅
//希望从启用了见证的对等机同步以确保
//我们完全验证所有区块链数据。
	segwitActive, err := sm.chain.IsDeploymentActive(chaincfg.DeploymentSegwit)
	if err != nil {
		log.Errorf("Unable to query for segwit soft-fork state: %v", err)
		return
	}

	best := sm.chain.BestSnapshot()
	var bestPeer *peerpkg.Peer
	for peer, state := range sm.peerStates {
		if !state.syncCandidate {
			continue
		}

		if segwitActive && !peer.IsWitnessEnabled() {
			log.Debugf("peer %v not witness enabled, skipping", peer)
			continue
		}

//删除不再是到期候选的同步候选对等点
//通过他们最新的已知街区。注：<
//有意而不是<=。从技术上讲，是同龄人
//当它相等时，没有后面的块，它很可能
//很快就有一个，所以这是一个合理的选择。它也允许
//两者都为0的情况，例如在回归测试期间。
		if peer.LastBlock() < best.Height {
			state.syncCandidate = false
			continue
		}

//TODO（Davec）：使用更好的算法来选择最佳对等。
//现在，只需选择第一个可用的候选人。
		bestPeer = peer
	}

//如果选择了最佳对等点，则从该对等点开始同步。
	if bestPeer != nil {
//如果同步对等更改，则清除请求的块，否则
//我们可以忽略最后一个同步对等失败所需的块
//发送。
		sm.requestedBlocks = make(map[chainhash.Hash]struct{})

		locator, err := sm.chain.LatestBlockLocator()
		if err != nil {
			log.Errorf("Failed to get block locator for the "+
				"latest block: %v", err)
			return
		}

		log.Infof("Syncing to block height %d from peer %v",
			bestPeer.LastBlock(), bestPeer.Addr())

//当当前高度小于已知的检查点时，我们
//可以使用块头了解哪些块包含
//到检查点的链，执行较少的验证
//对他们来说。这是可能的，因为每个头都包含
//前一个头和merkle根的哈希。因此如果
//我们验证所有接收到的头链接在一起
//正确地和检查点哈希匹配，我们可以确定
//中间块的哈希值是准确的。此外，一次
//下载完整的块，计算merkle根
//并与标题中的值进行比较，以证明
//完整块未被篡改。
//
//一旦我们通过了最后一个检查点，或者检查点是
//禁用，使用标准inv消息了解有关块的信息
//并完全验证它们。最后，回归测试模式可以
//不支持头文件第一个方法，所以正常块也不支持
//在回归测试模式下下载。
		if sm.nextCheckpoint != nil &&
			best.Height < sm.nextCheckpoint.Height &&
			sm.chainParams != &chaincfg.RegressionNetParams {

			bestPeer.PushGetHeadersMsg(locator, sm.nextCheckpoint.Hash)
			sm.headersFirstMode = true
			log.Infof("Downloading headers for blocks %d to "+
				"%d from peer %s", best.Height+1,
				sm.nextCheckpoint.Height, bestPeer.Addr())
		} else {
			bestPeer.PushGetBlocksMsg(locator, &zeroHash)
		}
		sm.syncPeer = bestPeer
	} else {
		log.Warnf("No sync peer candidates available")
	}
}

//IsSyncCandidate返回对等方是否为要考虑的候选
//同步。
func (sm *SyncManager) isSyncCandidate(peer *peerpkg.Peer) bool {
//通常，如果某个对等节点不是完整节点，则它不是同步的候选节点，
//然而回归测试的特殊之处在于回归工具
//不是一个完整的节点，仍然需要考虑同步候选。
	if sm.chainParams == &chaincfg.RegressionNetParams {
//如果对等计算机不是来自本地主机，则它不是候选计算机
//或者由于某种原因无法确定主机名。
		host, _, err := net.SplitHostPort(peer.Addr())
		if err != nil {
			return false
		}

		if host != "127.0.0.1" && host != "localhost" {
			return false
		}
	} else {
//如果不是完整的，则对等机不是同步的候选对象
//节点。Additionally, if the segwit soft-fork package has
//激活，则对等机也必须升级。
		segwitActive, err := sm.chain.IsDeploymentActive(chaincfg.DeploymentSegwit)
		if err != nil {
			log.Errorf("Unable to query for segwit "+
				"soft-fork state: %v", err)
		}
		nodeServices := peer.Services()
		if nodeServices&wire.SFNodeNetwork != wire.SFNodeNetwork ||
			(segwitActive && !peer.IsWitnessEnabled()) {
			return false
		}
	}

//如果所有支票都通过，则为候选人。
	return true
}

//handlenewpeermsg处理已发出信号的新对等机
//被视为同步对等（他们已经成功协商）。它
//如果需要，也开始同步。它是从同步处理程序Goroutine调用的。
func (sm *SyncManager) handleNewPeerMsg(peer *peerpkg.Peer) {
//关闭过程中忽略if。
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		return
	}

	log.Infof("New valid peer %s (%s)", peer, peer.UserAgent())

//初始化对等状态
	isSyncCandidate := sm.isSyncCandidate(peer)
	sm.peerStates[peer] = &peerSyncState{
		syncCandidate:   isSyncCandidate,
		requestedTxns:   make(map[chainhash.Hash]struct{}),
		requestedBlocks: make(map[chainhash.Hash]struct{}),
	}

//如果需要，通过选择最佳候选人开始同步。
	if isSyncCandidate && sm.syncPeer == nil {
		sm.startSync()
	}
}

//handledonepermsg处理已发出完成信号的对等机。它
//删除对等机作为同步的候选，如果是，则删除该对等机
//当前的同步对等，尝试从中选择新的最佳对等。它
//从同步处理程序goroutine调用。
func (sm *SyncManager) handleDonePeerMsg(peer *peerpkg.Peer) {
	state, exists := sm.peerStates[peer]
	if !exists {
		log.Warnf("Received done peer message for unknown peer %s", peer)
		return
	}

//从候选对等方列表中删除对等方。
	delete(sm.peerStates, peer)

	log.Infof("Lost peer %s", peer)

//从全局映射中删除请求的事务，以便
//下次我们收到发票时从别处取。
	for txHash := range state.requestedTxns {
		delete(sm.requestedTxns, txHash)
	}

//从全局映射中删除请求的块，以便
//下次我们收到发票时从别处取的。
//托多：我们可以在这里检查哪些对等机有这些块
//现在要求他们加快速度。
	for blockHash := range state.requestedBlocks {
		delete(sm.requestedBlocks, blockHash)
	}

//如果退出的对等机是
//同步对等体。另外，如果先在头中，则重置头的第一状态
//模式如此
	if sm.syncPeer == peer {
		sm.syncPeer = nil
		if sm.headersFirstMode {
			best := sm.chain.BestSnapshot()
			sm.resetHeaderState(&best.Hash, best.Height)
		}
		sm.startSync()
	}
}

//handletxmsg处理来自所有对等方的事务消息。
func (sm *SyncManager) handleTxMsg(tmsg *txMsg) {
	peer := tmsg.peer
	state, exists := sm.peerStates[peer]
	if !exists {
		log.Warnf("Received tx message from unknown peer %s", peer)
		return
	}

//注意：比特币，可能还有其他钱包，不符合
//发送清单消息并允许远程对等端决定
//他们是否希望通过getdata请求事务
//消息。不幸的是，参考实施许可证
//未请求的数据，因此它允许钱包不遵循
//规格激增。虽然这不理想，但这里没有支票
//断开对等机的连接，以便发送未经请求的事务
//互操作性。
	txHash := tmsg.tx.Hash()

//忽略我们已拒绝的交易。不
//在此处发送拒绝消息，因为如果事务已经
//拒绝，交易是主动提出的。
	if _, exists = sm.rejectedTxns[*txHash]; exists {
		log.Debugf("Ignoring unsolicited previously rejected "+
			"transaction %v from %s", txHash, peer)
		return
	}

//处理事务以包括验证、在
//内存池、孤立处理等。
	acceptedTxs, err := sm.txMemPool.ProcessTransaction(tmsg.tx,
		true, true, mempool.Tag(peer.ID()))

//从请求映射中删除事务。Mempool/Chain
//已经知道了，所以我们不应该再有了
//尝试获取它的实例，或者我们未能插入，因此
//下次收到发票时我们会重试。
	delete(state.requestedTxns, *txHash)
	delete(sm.requestedTxns, *txHash)

	if err != nil {
//在新块之前不要再次请求此事务
//已处理。
		sm.rejectedTxns[*txHash] = struct{}{}
		sm.limitMap(sm.rejectedTxns, maxRejectedTxns)

//如果错误是规则错误，则表示事务
//只是被拒绝，而不是实际出了问题，
//所以记录下来。否则，确实出了点问题，
//所以把它记录为实际错误。
		if _, ok := err.(mempool.RuleError); ok {
			log.Debugf("Rejected transaction %v from %s: %v",
				txHash, peer, err)
		} else {
			log.Errorf("Failed to process transaction %v: %v",
				txHash, err)
		}

//将错误转换为适当的拒绝消息并
//把它寄出去。
		code, reason := mempool.ErrToRejectErr(err)
		peer.PushRejectMsg(wire.CmdTx, code, reason, txHash, false)
		return
	}

	sm.peerNotifier.AnnounceNewTransactions(acceptedTxs)
}

//如果我们相信我们与同龄人同步，那么当前的返回值为真；如果我们认为我们与同龄人同步，那么当前的返回值为假。
//还有块要检查
func (sm *SyncManager) current() bool {
	if !sm.chain.IsCurrent() {
		return false
	}

//如果区块链认为我们是最新的，而我们没有同步对等机
//可能是对的。
	if sm.syncPeer == nil {
		return true
	}

//不管链怎么想，如果我们在块下面，我们正在同步
//对我们来说不是最新的。
	if sm.chain.BestSnapshot().Height < sm.syncPeer.LastBlock() {
		return false
	}
	return true
}

//handleblockmsg处理来自所有对等方的阻塞消息。
func (sm *SyncManager) handleBlockMsg(bmsg *blockMsg) {
	peer := bmsg.peer
	state, exists := sm.peerStates[peer]
	if !exists {
		log.Warnf("Received block message from unknown peer %s", peer)
		return
	}

//如果我们没有要求这一块，那么同龄人就是行为不端。
	blockHash := bmsg.block.Hash()
	if _, exists = state.requestedBlocks[*blockHash]; !exists {
//回归测试故意发送一些块两次
//测试重复块插入失败。不要断开
//当我们进行回归测试时，对等或忽略块
//在这种情况下，链代码实际上是
//重复块。
		if sm.chainParams != &chaincfg.RegressionNetParams {
			log.Warnf("Got unrequested block %v from %s -- "+
				"disconnecting", blockHash, peer.Addr())
			peer.Disconnect()
			return
		}
	}

//当处于headers-first模式时，如果块与
//正在获取的头列表中的第一个头，它是
//由于邮件头已经
//已验证连接在一起，并在下一个检查点之前有效。
//此外，删除除检查点之外的所有块的列表项
//因为需要验证下一轮的头链接
//适当地。
	isCheckpointBlock := false
	behaviorFlags := blockchain.BFNone
	if sm.headersFirstMode {
		firstNodeEl := sm.headerList.Front()
		if firstNodeEl != nil {
			firstNode := firstNodeEl.Value.(*headerNode)
			if blockHash.IsEqual(firstNode.hash) {
				behaviorFlags |= blockchain.BFFastAdd
				if firstNode.hash.IsEqual(sm.nextCheckpoint.Hash) {
					isCheckpointBlock = true
				} else {
					sm.headerList.Remove(firstNodeEl)
				}
			}
		}
	}

//从请求映射中删除块。任何一条链都会知道的
//所以我们不应该再有任何尝试获取它的实例，或者
//插入将失败，因此下次收到发票时我们将重试。
	delete(state.requestedBlocks, *blockHash)
	delete(sm.requestedBlocks, *blockHash)

//处理块以包括验证、最佳链选择、孤立
//处理等。
	_, isOrphan, err := sm.chain.ProcessBlock(bmsg.block, behaviorFlags)
	if err != nil {
//当错误是规则错误时，这意味着块
//与实际出错相反，被拒绝，所以记录
//就是这样。否则，确实出了问题，所以记录
//这是一个实际的错误。
		if _, ok := err.(blockchain.RuleError); ok {
			log.Infof("Rejected block %v from %s: %v", blockHash,
				peer, err)
		} else {
			log.Errorf("Failed to process block %v: %v",
				blockHash, err)
		}
		if dbErr, ok := err.(database.Error); ok && dbErr.ErrorCode ==
			database.ErrCorruption {
			panic(dbErr)
		}

//将错误转换为适当的拒绝消息并
//把它寄出去。
		code, reason := mempool.ErrToRejectErr(err)
		peer.PushRejectMsg(wire.CmdBlock, code, reason, blockHash, false)
		return
	}

//有关此对等方报告的新块的元数据。我们用这个
//下面更新此对等的最新块高度和
//其他同行基于他们最后公布的块哈希。这让我们
//动态更新对等块高度，避免过时
//寻找新同步对等时的高度。一块验收后
//或者识别一个孤儿，我们也使用这些信息来更新
//与其他同龄人相比，INV的街区高度可能已被忽略。
//如果我们在链尚未处于当前状态时主动同步，或者
//谁可能在宣布锁的比赛中输了。
	var heightUpdate int32
	var blkHashUpdate *chainhash.Hash

//从发送孤立块的对等端请求父块。
	if isOrphan {
//我们刚从一个同龄人那里收到一个孤立块。整齐
//为了更新对等点的高度，我们尝试提取
//来自coinbase事务的scriptsig的块高度。
//仅当块的版本为
//足够高（2+版）。
		header := &bmsg.block.MsgBlock().Header
		if blockchain.ShouldHaveSerializedBlockHeight(header) {
			coinbaseTx := bmsg.block.Transactions()[0]
			cbHeight, err := blockchain.ExtractCoinbaseHeight(coinbaseTx)
			if err != nil {
				log.Warnf("Unable to extract height from "+
					"coinbase tx: %v", err)
			} else {
				log.Debugf("Extracted height of %v from "+
					"orphan block", cbHeight)
				heightUpdate = cbHeight
				blkHashUpdate = blockHash
			}
		}

		orphanRoot := sm.chain.GetOrphanRoot(blockHash)
		locator, err := sm.chain.LatestBlockLocator()
		if err != nil {
			log.Warnf("Failed to get block locator for the "+
				"latest block: %v", err)
		} else {
			peer.PushGetBlocksMsg(locator, orphanRoot)
		}
	} else {
//当块不是孤立块时，记录有关该块的信息并
//更新链状态。
		sm.progressLogger.LogBlockHeight(bmsg.block)

//为将来更新此对等的最新块高度
//潜在的同步节点候选资格。
		best := sm.chain.BestSnapshot()
		heightUpdate = best.Height
		blkHashUpdate = &best.Hash

//清除已拒绝的交易记录。
		sm.rejectedTxns = make(map[chainhash.Hash]struct{})
	}

//更新此对等机的块高度。但只给
//更新对等高度的服务器（如果这是孤立的或我们的）
//链是“当前”的。这样可以避免发送大量消息
//如果我们从头开始同步链。
	if blkHashUpdate != nil && heightUpdate != 0 {
		peer.UpdateLastBlockHeight(heightUpdate)
		if isOrphan || sm.current() {
			go sm.peerNotifier.UpdatePeerHeights(blkHashUpdate, heightUpdate,
				peer)
		}
	}

//如果我们不处于头一模式，则无需执行其他操作。
	if !sm.headersFirstMode {
		return
	}

//这是头优先模式，因此如果块不是检查点
//当请求队列为
//变短了。
	if !isCheckpointBlock {
		if sm.startHeader != nil &&
			len(state.requestedBlocks) < minInFlightBlocks {
			sm.fetchHeaderBlocks()
		}
		return
	}

//这是头优先模式，块是检查点。什么时候？
//有下一个检查点，通过询问获得下一轮的标题
//对于从该块后到下一块的头
//检查点。
	prevHeight := sm.nextCheckpoint.Height
	prevHash := sm.nextCheckpoint.Hash
	sm.nextCheckpoint = sm.findNextHeaderCheckpoint(prevHeight)
	if sm.nextCheckpoint != nil {
		locator := blockchain.BlockLocator([]*chainhash.Hash{prevHash})
		err := peer.PushGetHeadersMsg(locator, sm.nextCheckpoint.Hash)
		if err != nil {
			log.Warnf("Failed to send getheaders message to "+
				"peer %s: %v", peer.Addr(), err)
			return
		}
		log.Infof("Downloading headers for blocks %d to %d from "+
			"peer %s", prevHeight+1, sm.nextCheckpoint.Height,
			sm.syncPeer.Addr())
		return
	}

//这是头优先模式，块是检查点，有
//没有更多的检查点，因此通过请求块切换到正常模式
//从这个块后到链的末尾（零哈希）。
	sm.headersFirstMode = false
	sm.headerList.Init()
	log.Infof("Reached the final checkpoint -- switching to normal mode")
	locator := blockchain.BlockLocator([]*chainhash.Hash{blockHash})
	err = peer.PushGetBlocksMsg(locator, &zeroHash)
	if err != nil {
		log.Warnf("Failed to send getblocks message to peer %s: %v",
			peer.Addr(), err)
		return
	}
}

//fetchheaderBlocks创建一个请求并将其发送到下一个同步对等机
//基于当前头列表下载的块列表。
func (sm *SyncManager) fetchHeaderBlocks() {
//如果没有起始标题，则不执行任何操作。
	if sm.startHeader == nil {
		log.Warnf("fetchHeaderBlocks called with no start header")
		return
	}

//为头文件的块列表生成getdata请求
//描述。大小提示将限于Wire.MaxInvPermsg
//功能，所以这里不需要再检查。
	gdmsg := wire.NewMsgGetDataSizeHint(uint(sm.headerList.Len()))
	numRequested := 0
	for e := sm.startHeader; e != nil; e = e.Next() {
		node, ok := e.Value.(*headerNode)
		if !ok {
			log.Warn("Header list node type is not a headerNode")
			continue
		}

		iv := wire.NewInvVect(wire.InvTypeBlock, node.hash)
		haveInv, err := sm.haveInventory(iv)
		if err != nil {
			log.Warnf("Unexpected failure when checking for "+
				"existing inventory during header block "+
				"fetch: %v", err)
		}
		if !haveInv {
			syncPeerState := sm.peerStates[sm.syncPeer]

			sm.requestedBlocks[*node.hash] = struct{}{}
			syncPeerState.requestedBlocks[*node.hash] = struct{}{}

//如果我们从一个支持证人的同伴那里
//后叉，然后确保我们收到所有
//块中的见证数据。
			if sm.syncPeer.IsWitnessEnabled() {
				iv.Type = wire.InvTypeWitnessBlock
			}

			gdmsg.AddInvVect(iv)
			numRequested++
		}
		sm.startHeader = e.Next()
		if numRequested >= wire.MaxInvPerMsg {
			break
		}
	}
	if len(gdmsg.InvList) > 0 {
		sm.syncPeer.QueueMessage(gdmsg, nil)
	}
}

//handleHeadersMsg处理来自所有对等方的块头消息。报头是
//首次同步邮件头时请求。
func (sm *SyncManager) handleHeadersMsg(hmsg *headersMsg) {
	peer := hmsg.peer
	_, exists := sm.peerStates[peer]
	if !exists {
		log.Warnf("Received headers message from unknown peer %s", peer)
		return
	}

//如果我们不请求报头，远程对等机就会出现问题。
	msg := hmsg.headers
	numHeaders := len(msg.Headers)
	if !sm.headersFirstMode {
		log.Warnf("Got %d unrequested headers from %s -- "+
			"disconnecting", numHeaders, peer.Addr())
		peer.Disconnect()
		return
	}

//对于空邮件头消息，不执行任何操作。
	if numHeaders == 0 {
		return
	}

//处理所有接收到的头，确保每个头连接到
//上一个和那个检查点匹配。
	receivedCheckpoint := false
	var finalHash *chainhash.Hash
	for _, blockHeader := range msg.Headers {
		blockHash := blockHeader.BlockHash()
		finalHash = &blockHash

//确保有上一个要比较的标题。
		prevNodeEl := sm.headerList.Back()
		if prevNodeEl == nil {
			log.Warnf("Header list does not contain a previous" +
				"element as expected -- disconnecting peer")
			peer.Disconnect()
			return
		}

//确保收割台与上一个收割台正确连接，并且
//将其添加到标题列表中。
		node := headerNode{hash: &blockHash}
		prevNode := prevNodeEl.Value.(*headerNode)
		if prevNode.hash.IsEqual(&blockHeader.PrevBlock) {
			node.height = prevNode.height + 1
			e := sm.headerList.PushBack(&node)
			if sm.startHeader == nil {
				sm.startHeader = e
			}
		} else {
			log.Warnf("Received block header that does not "+
				"properly connect to the chain from peer %s "+
				"-- disconnecting", peer.Addr())
			peer.Disconnect()
			return
		}

//验证下一个检查点高度处的头是否匹配。
		if node.height == sm.nextCheckpoint.Height {
			if node.hash.IsEqual(sm.nextCheckpoint.Hash) {
				receivedCheckpoint = true
				log.Infof("Verified downloaded block "+
					"header against checkpoint at height "+
					"%d/hash %s", node.height, node.hash)
			} else {
				log.Warnf("Block header at height %d/hash "+
					"%s from peer %s does NOT match "+
					"expected checkpoint hash of %s -- "+
					"disconnecting", node.height,
					node.hash, peer.Addr(),
					sm.nextCheckpoint.Hash)
				peer.Disconnect()
				return
			}
			break
		}
	}

//当此头是检查点时，切换到获取
//自上次检查点以来的所有头。
	if receivedCheckpoint {
//因为列表的第一个条目总是最后一个块
//数据库中已存在，仅用于确保
//下一个收割台链接正确，必须在
//正在获取块。
		sm.headerList.Remove(sm.headerList.Front())
		log.Infof("Received %v block headers: Fetching blocks",
			sm.headerList.Len())
		sm.progressLogger.SetLastLogTime(time.Now())
		sm.fetchHeaderBlocks()
		return
	}

//此头不是检查点，因此请求下一批
//头从最新的已知头开始，以
//下一个检查点。
	locator := blockchain.BlockLocator([]*chainhash.Hash{finalHash})
	err := peer.PushGetHeadersMsg(locator, sm.nextCheckpoint.Hash)
	if err != nil {
		log.Warnf("Failed to send getheaders message to "+
			"peer %s: %v", peer.Addr(), err)
		return
	}
}

//有库存退货，无论是否通过
//库存向量已知。这包括检查所有不同的地方
//当库存处于不同状态时，例如作为部件的块时，库存可以是
//主链、侧链、孤立池以及
//在内存池中（主池或孤立池）。
func (sm *SyncManager) haveInventory(invVect *wire.InvVect) (bool, error) {
	switch invVect.Type {
	case wire.InvTypeWitnessBlock:
		fallthrough
	case wire.InvTypeBlock:
//询问链是否以任何形式知道块（主
//链条、侧链或孤立）。
		return sm.chain.HaveBlock(&invVect.Hash)

	case wire.InvTypeWitnessTx:
		fallthrough
	case wire.InvTypeTx:
//询问事务内存池是否知道事务
//以任何形式（主池或孤立池）发送给它。
		if sm.txMemPool.HaveTransaction(&invVect.Hash) {
			return true, nil
		}

//从
//主链的末端。注意这只是一个最大的努力
//因为检查每个输出和
//此检查的唯一目的是避免下载
//已知事务。只有前两个输出是
//检查，因为绝大多数交易都包含
//两个输出，其中一个是某种形式的“支付给他人”
//另一个是变化输出。
		prevOut := wire.OutPoint{Hash: invVect.Hash}
		for i := uint32(0); i < 2; i++ {
			prevOut.Index = i
			entry, err := sm.chain.FetchUtxoEntry(prevOut)
			if err != nil {
				return false, err
			}
			if entry != nil && !entry.IsSpent() {
				return true, nil
			}
		}

		return false, nil
	}

//请求的库存是不支持的类型，因此只需索赔
//众所周知，这是为了避免请求。
	return true, nil
}

//handleinvmsg处理来自所有对等方的inv消息。
//我们检查远程对等端公布的清单并采取相应的行动。
func (sm *SyncManager) handleInvMsg(imsg *invMsg) {
	peer := imsg.peer
	state, exists := sm.peerStates[peer]
	if !exists {
		log.Warnf("Received inv message from unknown peer %s", peer)
		return
	}

//尝试在库存列表中查找最终块。有可能
//不是一个。
	lastBlock := -1
	invVects := imsg.inv.InvList
	for i := len(invVects) - 1; i >= 0; i-- {
		if invVects[i].Type == wire.InvTypeBlock {
			lastBlock = i
			break
		}
	}

//如果此发票包含阻止通知，而此通知不是来自
//我们当前的同步对等或我们是当前的，然后更新最后一个
//已宣布此对等机的阻止。我们稍后将使用此信息
//根据我们接受的数据块更新同行的高度
//先前宣布。
	if lastBlock != -1 && (peer != sm.syncPeer || sm.current()) {
		peer.UpdateLastAnnouncedBlock(&invVects[lastBlock].Hash)
	}

//如果我们不是最新的，请忽略来自不同步的对等方的INV。
//有助于防止抓到大量孤儿。
	if peer != sm.syncPeer && !sm.current() {
		return
	}

//如果我们的链是当前的，并且有一个对等方宣布一个块，那么我们已经
//知道，然后更新当前块高度。
	if lastBlock != -1 && sm.current() {
		blkHeight, err := sm.chain.BlockHeightByHash(&invVects[lastBlock].Hash)
		if err == nil {
			peer.UpdateLastBlockHeight(blkHeight)
		}
	}

//如果我们还没有广告库存，请索取。也，
//request parent blocks of orphans if we receive one we already have.
//最后，尝试检测由于长侧链导致的潜在失速。
//我们已经有并请求更多的块来阻止它们。
	for i, iv := range invVects {
//忽略不支持的清单类型。
		switch iv.Type {
		case wire.InvTypeBlock:
		case wire.InvTypeTx:
		case wire.InvTypeWitnessBlock:
		case wire.InvTypeWitnessTx:
		default:
			continue
		}

//将清单添加到已知清单的缓存中
//为同行。
		peer.AddKnownInventory(iv)

//当我们处于标题优先模式时忽略清单。
		if sm.headersFirstMode {
			continue
		}

//如果我们还没有库存，请申请库存。
		haveInv, err := sm.haveInventory(iv)
		if err != nil {
			log.Warnf("Unexpected failure when checking for "+
				"existing inventory during inv message "+
				"processing: %v", err)
			continue
		}
		if !haveInv {
			if iv.Type == wire.InvTypeTx {
//如果事务已经
//拒绝。
				if _, exists := sm.rejectedTxns[iv.Hash]; exists {
					continue
				}
			}

//忽略未启用见证的invs块invs
//同龄人，在Segwit激活后，我们只想
//从能为我们提供完整见证的同行下载
//数据块。
			if !peer.IsWitnessEnabled() && iv.Type == wire.InvTypeBlock {
				continue
			}

//将其添加到请求队列。
			state.requestQueue = append(state.requestQueue, iv)
			continue
		}

		if iv.Type == wire.InvTypeBlock {
//该块是我们已经拥有的孤立块。
//当处理现有孤立对象时，它请求
//缺少的父块。当这个场景
//碰巧，这意味着有更多的街区丢失了
//在单个库存消息中允许的。AS
//结果是，一旦该对等方请求
//公告块，远程对等机注意到，现在
//将孤立块重新发送为可用块
//为了表示有更多丢失的块需要
//被要求。
			if sm.chain.IsKnownOrphan(&iv.Hash) {
//最新已知的请求块
//一直到刚来的孤儿的根
//在。
				orphanRoot := sm.chain.GetOrphanRoot(&iv.Hash)
				locator, err := sm.chain.LatestBlockLocator()
				if err != nil {
					log.Errorf("PEER: Failed to get block "+
						"locator for the latest block: "+
						"%v", err)
					continue
				}
				peer.PushGetBlocksMsg(locator, orphanRoot)
				continue
			}

//我们已经有最后一块广告了
//库存消息，因此强制请求更多。这个
//只有当我们站在一个很长的一边
//链。
			if i == lastBlock {
//请求块在此之后一直到
//远程对等机知道的最后一个（零
//停止哈希）。
				locator := sm.chain.BlockLocatorFromHash(&iv.Hash)
				peer.PushGetBlocksMsg(locator, &zeroHash)
			}
		}
	}

//一次尽可能多的请求。任何不适合的东西
//请求将在下一个INV消息中被请求。
	numRequested := 0
	gdmsg := wire.NewMsgGetData()
	requestQueue := state.requestQueue
	for len(requestQueue) != 0 {
		iv := requestQueue[0]
		requestQueue[0] = nil
		requestQueue = requestQueue[1:]

		switch iv.Type {
		case wire.InvTypeWitnessBlock:
			fallthrough
		case wire.InvTypeBlock:
//如果尚未有挂起的块，则请求该块
//请求。
			if _, exists := sm.requestedBlocks[iv.Hash]; !exists {
				sm.requestedBlocks[iv.Hash] = struct{}{}
				sm.limitMap(sm.requestedBlocks, maxRequestedBlocks)
				state.requestedBlocks[iv.Hash] = struct{}{}

				if peer.IsWitnessEnabled() {
					iv.Type = wire.InvTypeWitnessBlock
				}

				gdmsg.AddInvVect(iv)
				numRequested++
			}

		case wire.InvTypeWitnessTx:
			fallthrough
		case wire.InvTypeTx:
//如果还没有
//挂起的请求。
			if _, exists := sm.requestedTxns[iv.Hash]; !exists {
				sm.requestedTxns[iv.Hash] = struct{}{}
				sm.limitMap(sm.requestedTxns, maxRequestedTxns)
				state.requestedTxns[iv.Hash] = struct{}{}

//如果对等机有能力，请求txn
//包括所有见证数据。
				if peer.IsWitnessEnabled() {
					iv.Type = wire.InvTypeWitnessTx
				}

				gdmsg.AddInvVect(iv)
				numRequested++
			}
		}

		if numRequested >= wire.MaxInvPerMsg {
			break
		}
	}
	state.requestQueue = requestQueue
	if len(gdmsg.InvList) > 0 {
		peer.QueueMessage(gdmsg, nil)
	}
}

//limitmap是一个辅助函数，用于需要最大限制的映射
//如果添加新值，则逐出随机事务将导致
//溢出允许的最大值。
func (sm *SyncManager) limitMap(m map[chainhash.Hash]struct{}, limit int) {
	if len(m)+1 > limit {
//从地图中删除一个随机条目。对于大多数编译器，go's
//range语句从随机项开始迭代，尽管
//这不是规范100%保证的。迭代顺序
//在这里并不重要，因为对手必须
//能够在
//以任何方式将特定条目逐出为目标。
		for txHash := range m {
			delete(m, txHash)
			return
		}
	}
}

//BlockHandler是同步管理器的主要处理程序。它必须作为
//高尔图它在单独的goroutine中处理块和inv消息
//来自对等处理程序，因此块（msgblock）消息由
//单线程，无需锁定内存数据结构。这是
//重要的是，同步管理器控制需要哪些块以及如何
//应继续提取。
func (sm *SyncManager) blockHandler() {
out:
	for {
		select {
		case m := <-sm.msgChan:
			switch msg := m.(type) {
			case *newPeerMsg:
				sm.handleNewPeerMsg(msg.peer)

			case *txMsg:
				sm.handleTxMsg(msg)
				msg.reply <- struct{}{}

			case *blockMsg:
				sm.handleBlockMsg(msg)
				msg.reply <- struct{}{}

			case *invMsg:
				sm.handleInvMsg(msg)

			case *headersMsg:
				sm.handleHeadersMsg(msg)

			case *donePeerMsg:
				sm.handleDonePeerMsg(msg.peer)

			case getSyncPeerMsg:
				var peerID int32
				if sm.syncPeer != nil {
					peerID = sm.syncPeer.ID()
				}
				msg.reply <- peerID

			case processBlockMsg:
				_, isOrphan, err := sm.chain.ProcessBlock(
					msg.block, msg.flags)
				if err != nil {
					msg.reply <- processBlockResponse{
						isOrphan: false,
						err:      err,
					}
				}

				msg.reply <- processBlockResponse{
					isOrphan: isOrphan,
					err:      nil,
				}

			case isCurrentMsg:
				msg.reply <- sm.current()

			case pauseMsg:
//等待发送方解除对管理器的暂停。
				<-msg.unpause

			default:
				log.Warnf("Invalid message type in block "+
					"handler: %T", msg)
			}

		case <-sm.quit:
			break out
		}
	}

	sm.wg.Done()
	log.Trace("Block handler done")
}

//handleBlockChainNotification处理来自区块链的通知。它确实
//请求孤立块父级和将接受的块中继到
//互联对等。
func (sm *SyncManager) handleBlockchainNotification(notification *blockchain.Notification) {
	switch notification.Type {
//区块链中已接受一个区块。把它转给其他人
//同龄人。
	case blockchain.NTBlockAccepted:
//如果我们没有电流，就不要继电器。其他同行
//电流应该已经知道了。
		if !sm.current() {
			return
		}

		block, ok := notification.Data.(*btcutil.Block)
		if !ok {
			log.Warnf("Chain accepted notification is not a block.")
			break
		}

//生成库存向量并传递它。
		iv := wire.NewInvVect(wire.InvTypeBlock, block.Hash())
		sm.peerNotifier.RelayInventory(iv, block.MsgBlock().Header)

//已将一个块连接到主块链。
	case blockchain.NTBlockConnected:
		block, ok := notification.Data.(*btcutil.Block)
		if !ok {
			log.Warnf("Chain connected notification is not a block.")
			break
		}

//删除中的所有事务（coinbase除外）
//从事务池连接的块。其次，删除任何
//由于这些原因，现在花费加倍的交易
//新交易。最后，删除
//不再是孤儿了。依赖确认的交易
//事务不会递归删除，因为它们仍然是
//有效。
		for _, tx := range block.Transactions()[1:] {
			sm.txMemPool.RemoveTransaction(tx, false)
			sm.txMemPool.RemoveDoubleSpends(tx)
			sm.txMemPool.RemoveOrphan(tx)
			sm.peerNotifier.TransactionConfirmed(tx)
			acceptedTxs := sm.txMemPool.ProcessOrphans(tx)
			sm.peerNotifier.AnnounceNewTransactions(acceptedTxs)
		}

//如果存在费用估算器，则向其登记区块。
		if sm.feeEstimator != nil {
			err := sm.feeEstimator.RegisterBlock(block)

//如果某种程度上产生了错误，那么费用估计量
//已进入无效状态。因为它不知道如何
//要恢复，请创建一个新的。
			if err != nil {
				sm.feeEstimator = mempool.NewFeeEstimator(
					mempool.DefaultEstimateFeeMaxRollback,
					mempool.DefaultEstimateFeeMinRegisteredBlocks)
			}
		}

//一个滑轮已从主滑轮链上断开。
	case blockchain.NTBlockDisconnected:
		block, ok := notification.Data.(*btcutil.Block)
		if !ok {
			log.Warnf("Chain disconnected notification is not a block.")
			break
		}

//将所有交易（coinbase除外）重新插入
//事务池。
		for _, tx := range block.Transactions()[1:] {
			_, _, err := sm.txMemPool.MaybeAcceptTransaction(tx,
				false, false)
			if err != nil {
//删除事务和所有事务
//这取决于它是否被接受
//事务池。
				sm.txMemPool.RemoveTransaction(tx, true)
			}
		}

//回滚费用估算器记录的上一个块。
		if sm.feeEstimator != nil {
			sm.feeEstimator.Rollback(block.Hash())
		}
	}
}

//newpeer通知同步管理器新活动的对等机。
func (sm *SyncManager) NewPeer(peer *peerpkg.Peer) {
//如果要关闭，请忽略。
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		return
	}
	sm.msgChan <- &newPeerMsg{peer: peer}
}

//queuetx添加传递的事务消息并与块处理对等
//排队。在发送消息后响应done通道参数
//处理。
func (sm *SyncManager) QueueTx(tx *btcutil.Tx, peer *peerpkg.Peer, done chan struct{}) {
//如果我们要关闭，不要接受更多的事务。
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		done <- struct{}{}
		return
	}

	sm.msgChan <- &txMsg{tx: tx, peer: peer, reply: done}
}

//QueueBlock添加传递的块消息并与块处理对等
//排队。在块消息为
//处理。
func (sm *SyncManager) QueueBlock(block *btcutil.Block, peer *peerpkg.Peer, done chan struct{}) {
//如果我们要关闭，不要接受更多的块。
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		done <- struct{}{}
		return
	}

	sm.msgChan <- &blockMsg{block: block, peer: peer, reply: done}
}

//queue inv将传递的inv消息添加到块处理队列。
func (sm *SyncManager) QueueInv(inv *wire.MsgInv, peer *peerpkg.Peer) {
//这里没有通道处理，因为对等端不需要阻塞inv
//信息。
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		return
	}

	sm.msgChan <- &invMsg{inv: inv, peer: peer}
}

//QueueHeaders添加传递的Headers消息并与块处理对等
//排队。
func (sm *SyncManager) QueueHeaders(headers *wire.MsgHeaders, peer *peerpkg.Peer) {
//这里没有通道处理，因为对等端不需要阻塞
//邮件头。
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		return
	}

	sm.msgChan <- &headersMsg{headers: headers, peer: peer}
}

//Donepeer通知BlockManager对等机已断开连接。
func (sm *SyncManager) DonePeer(peer *peerpkg.Peer) {
//如果要关闭，请忽略。
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		return
	}

	sm.msgChan <- &donePeerMsg{peer: peer}
}

//Start开始处理块和inv消息的核心块处理程序。
func (sm *SyncManager) Start() {
//已经开始？
	if atomic.AddInt32(&sm.started, 1) != 1 {
		return
	}

	log.Trace("Starting sync manager")
	sm.wg.Add(1)
	go sm.blockHandler()
}

//停止通过停止所有异步方式优雅地关闭同步管理器
//处理程序并等待它们完成。
func (sm *SyncManager) Stop() error {
	if atomic.AddInt32(&sm.shutdown, 1) != 1 {
		log.Warnf("Sync manager is already in the process of " +
			"shutting down")
		return nil
	}

	log.Infof("Sync manager shutting down")
	close(sm.quit)
	sm.wg.Wait()
	return nil
}

//sync peer id返回当前同步对等的ID，如果没有，则返回0。
func (sm *SyncManager) SyncPeerID() int32 {
	reply := make(chan int32)
	sm.msgChan <- getSyncPeerMsg{reply: reply}
	return <-reply
}

//processBlock在块的内部实例上使用processBlock
//链。
func (sm *SyncManager) ProcessBlock(block *btcutil.Block, flags blockchain.BehaviorFlags) (bool, error) {
	reply := make(chan processBlockResponse, 1)
	sm.msgChan <- processBlockMsg{block: block, flags: flags, reply: reply}
	response := <-reply
	return response.isOrphan, response.err
}

//iscurrent返回同步管理器是否相信它与同步
//连接的对等机。
func (sm *SyncManager) IsCurrent() bool {
	reply := make(chan bool)
	sm.msgChan <- isCurrentMsg{reply: reply}
	return <-reply
}

//暂停暂停同步管理器，直到返回的通道关闭。
//
//注意，暂停时，所有对等和块处理都会停止。这个
//消息发送者应避免长时间暂停同步管理器。
func (sm *SyncManager) Pause() chan<- struct{} {
	c := make(chan struct{})
	sm.msgChan <- pauseMsg{c}
	return c
}

//New构造新的SyncManager。使用Start开始异步处理
//block、tx和inv更新。
func New(config *Config) (*SyncManager, error) {
	sm := SyncManager{
		peerNotifier:    config.PeerNotifier,
		chain:           config.Chain,
		txMemPool:       config.TxMemPool,
		chainParams:     config.ChainParams,
		rejectedTxns:    make(map[chainhash.Hash]struct{}),
		requestedTxns:   make(map[chainhash.Hash]struct{}),
		requestedBlocks: make(map[chainhash.Hash]struct{}),
		peerStates:      make(map[*peerpkg.Peer]*peerSyncState),
		progressLogger:  newBlockProgressLogger("Processed", log),
		msgChan:         make(chan interface{}, config.MaxPeers*3),
		headerList:      list.New(),
		quit:            make(chan struct{}),
		feeEstimator:    config.FeeEstimator,
	}

	best := sm.chain.BestSnapshot()
	if !config.DisableCheckpoints {
//根据当前高度初始化下一个检查点。
		sm.nextCheckpoint = sm.findNextHeaderCheckpoint(best.Height)
		if sm.nextCheckpoint != nil {
			sm.resetHeaderState(&best.Hash, best.Height)
		}
	} else {
		log.Info("Checkpoints are disabled")
	}

	sm.chain.Subscribe(sm.handleBlockchainNotification)

	return &sm, nil
}
