
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2017 BTCSuite开发者
//版权所有（c）2015-2018法令开发商
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package main

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/addrmgr"
	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/blockchain/indexers"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/connmgr"
	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcd/mempool"
	"github.com/btcsuite/btcd/mining"
	"github.com/btcsuite/btcd/mining/cpuminer"
	"github.com/btcsuite/btcd/netsync"
	"github.com/btcsuite/btcd/peer"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/bloom"
)

const (
//DefaultServices描述由支持的默认服务
//服务器。
	defaultServices = wire.SFNodeNetwork | wire.SFNodeBloom |
		wire.SFNodeWitness | wire.SFNodeCF

//DefaultRequiredServices描述的是
//需要出站对等机支持。
	defaultRequiredServices = wire.SFNodeNetwork

//DefaultTargetOutbound是目标的默认出站对等数。
	defaultTargetOutbound = 8

//ConnectionRetryInterval是介于
//连接到持久对等时重试。它是由
//重试次数，使重试后退。
	connectionRetryInterval = time.Second * 5
)

var (
//user agent name是用户代理名称，用于帮助识别
//我们和其他比特币同行。
	userAgentName = "btcd"

//user agent version是用户代理版本，用于帮助
//向其他比特币同行表明自己的身份。
	userAgentVersion = fmt.Sprintf("%d.%d.%d", appMajor, appMinor, appPatch)
)

//zero hash是零值哈希（全部为零）。它被定义为一种便利。
var zeroHash chainhash.Hash

//onionaddr实现net.addr接口并表示一个tor地址。
type onionAddr struct {
	addr string
}

//字符串返回洋葱地址。
//
//这是net.addr接口的一部分。
func (oa *onionAddr) String() string {
	return oa.addr
}

//网络返回“洋葱”。
//
//这是net.addr接口的一部分。
func (oa *onionAddr) Network() string {
	return "onion"
}

//确保onionaddr实现net.addr接口。
var _ net.Addr = (*onionAddr)(nil)

//simpleaddr使用两个结构字段实现net.addr接口
type simpleAddr struct {
	net, addr string
}

//字符串返回地址。
//
//这是net.addr接口的一部分。
func (a simpleAddr) String() string {
	return a.addr
}

//网络返回网络。
//
//这是net.addr接口的一部分。
func (a simpleAddr) Network() string {
	return a.net
}

//确保simpleaddr实现net.addr接口。
var _ net.Addr = simpleAddr{}

//Broadcastmsg提供存储待广播比特币消息的功能
//所有连接的对等机，指定排除的对等机除外。
type broadcastMsg struct {
	message      wire.Message
	excludePeers []*serverPeer
}

//BroadcastInventoryAdd是用于声明其包含的invvect的类型
//需要添加到重播地图
type broadcastInventoryAdd relayMsg

//BroadcastInventoryDel是用于声明其包含的invVect的类型
//需要从重播地图中删除
type broadcastInventoryDel *wire.InvVect

//relaymsg将一个库存向量与新发现的
//因此中继可以访问该信息。
type relayMsg struct {
	invVect *wire.InvVect
	data    interface{}
}

//updatepeerHeightsmsg是从BlockManager发送到服务器的消息
//接受新块后。消息的目的是更新
//在我们之前，大家都知道宣布封锁的同龄人的高度
//把它连接到主链上，或者把它识别为孤立的。用这些
//更新，对等高度将保持最新，允许在
//选择同步同行候选资格。
type updatePeerHeightsMsg struct {
	newHash    *chainhash.Hash
	newHeight  int32
	originPeer *peer.Peer
}

//PeerState还维护入站、持久、出站对等的状态
//作为被禁止的同龄人和出境团体。
type peerState struct {
	inboundPeers    map[int32]*serverPeer
	outboundPeers   map[int32]*serverPeer
	persistentPeers map[int32]*serverPeer
	banned          map[string]time.Time
	outboundGroups  map[string]int
}

//count返回所有已知对等方的计数。
func (ps *peerState) Count() int {
	return len(ps.inboundPeers) + len(ps.outboundPeers) +
		len(ps.persistentPeers)
}

//foralloutboundpeers是一个在所有出站上运行闭包的助手函数
//对等国已知的对等方。
func (ps *peerState) forAllOutboundPeers(closure func(sp *serverPeer)) {
	for _, e := range ps.outboundPeers {
		closure(e)
	}
	for _, e := range ps.persistentPeers {
		closure(e)
	}
}

//forallpeers是一个助手函数，它在已知的所有对等机上运行闭包。
//彼得的
func (ps *peerState) forAllPeers(closure func(sp *serverPeer)) {
	for _, e := range ps.inboundPeers {
		closure(e)
	}
	ps.forAllOutboundPeers(closure)
}

//cfheaderkv是一个过滤器头及其相关块散列的元组。这个
//结构用于缓存cfcheckpt响应。
type cfHeaderKV struct {
	blockHash    chainhash.Hash
	filterHeader chainhash.Hash
}

//服务器提供比特币服务器，用于处理与之之间的通信。
//比特币同行。
type server struct {
//以下变量只能原子地使用。
//首先放置uint64使它们与32位系统64位对齐。
bytesReceived uint64 //自启动后从所有对等方接收的总字节数。
bytesSent     uint64 //自启动后所有对等方发送的总字节数。
	started       int32
	shutdown      int32
	shutdownSched int32
	startupTime   int64

	chainParams          *chaincfg.Params
	addrManager          *addrmgr.AddrManager
	connManager          *connmgr.ConnManager
	sigCache             *txscript.SigCache
	hashCache            *txscript.HashCache
	rpcServer            *rpcServer
	syncManager          *netsync.SyncManager
	chain                *blockchain.BlockChain
	txMemPool            *mempool.TxPool
	cpuMiner             *cpuminer.CPUMiner
	modifyRebroadcastInv chan interface{}
	newPeers             chan *serverPeer
	donePeers            chan *serverPeer
	banPeers             chan *serverPeer
	query                chan interface{}
	relayInv             chan relayMsg
	broadcast            chan broadcastMsg
	peerHeightsUpdate    chan updatePeerHeightsMsg
	wg                   sync.WaitGroup
	quit                 chan struct{}
	nat                  NAT
	db                   database.DB
	timeSource           blockchain.MedianTimeSource
	services             wire.ServiceFlag

//以下字段用于可选索引。他们将是零
//如果未启用关联索引。这些字段是在
//服务器的初始创建，之后从未更改，因此它们
//不需要对并发访问进行保护。
	txIndex   *indexers.TxIndex
	addrIndex *indexers.AddrIndex
	cfIndex   *indexers.CfIndex

//费用估算器跟踪交易的剩余时间。
//在他们被开采成块之前。
	feeEstimator *mempool.FeeEstimator

//cfcheckptcaches为cfcheckpt存储一个过滤器头的缓存切片
//每个筛选器类型的消息。
	cfCheckptCaches    map[wire.FilterType][]cfHeaderKV
	cfCheckptCachesMtx sync.RWMutex
}

//server peer扩展对等机以维护服务器共享的状态，并
//块管理器。
type serverPeer struct {
//以下变量只能原子地使用
	feeFilter int64

	*peer.Peer

	connReq        *connmgr.ConnReq
	server         *server
	persistent     bool
	continueHash   *chainhash.Hash
	relayMtx       sync.Mutex
	disableRelayTx bool
	sentAddrs      bool
	isWhitelisted  bool
	filter         *bloom.Filter
	knownAddresses map[string]struct{}
	banScore       connmgr.DynamicBanScore
	quit           chan struct{}
//以下通道用于同步BlockManager和服务器。
	txProcessed    chan struct{}
	blockProcessed chan struct{}
}

//NewServerPeer返回新的ServerPeer实例。对等机需要由
//呼叫者。
func newServerPeer(s *server, isPersistent bool) *serverPeer {
	return &serverPeer{
		server:         s,
		persistent:     isPersistent,
		filter:         bloom.LoadFilter(nil),
		knownAddresses: make(map[string]struct{}),
		quit:           make(chan struct{}),
		txProcessed:    make(chan struct{}, 1),
		blockProcessed: make(chan struct{}, 1),
	}
}

//newestblock使用格式返回当前最佳块哈希和高度
//对等包的配置所需。
func (sp *serverPeer) newestBlock() (*chainhash.Hash, int32, error) {
	best := sp.server.chain.BestSnapshot()
	return &best.Hash, best.Height, nil
}

//addknownaddress将给定地址添加到已知地址集
//防止发送重复地址的对等机。
func (sp *serverPeer) addKnownAddresses(addresses []*wire.NetAddress) {
	for _, na := range addresses {
		sp.knownAddresses[addrmgr.NetAddressKey(na)] = struct{}{}
	}
}

//如果给定的地址已经为对等方所知，那么address known为true。
func (sp *serverPeer) addressKnown(na *wire.NetAddress) bool {
	_, exists := sp.knownAddresses[addrmgr.NetAddressKey(na)]
	return exists
}

//setDisableRelayTx为给定对等机切换事务的中继。
//它对于并发访问是安全的。
func (sp *serverPeer) setDisableRelayTx(disable bool) {
	sp.relayMtx.Lock()
	sp.disableRelayTx = disable
	sp.relayMtx.Unlock()
}

//relaysDisabled返回给定事务的中继
//对等机已禁用。
//它对于并发访问是安全的。
func (sp *serverPeer) relayTxDisabled() bool {
	sp.relayMtx.Lock()
	isDisabled := sp.disableRelayTx
	sp.relayMtx.Unlock()

	return isDisabled
}

//pushaddrmsg使用提供的
//地址。
func (sp *serverPeer) pushAddrMsg(addresses []*wire.NetAddress) {
//筛选器地址已为对等方所知。
	addrs := make([]*wire.NetAddress, 0, len(addresses))
	for _, addr := range addresses {
		if !sp.addressKnown(addr) {
			addrs = append(addrs, addr)
		}
	}
	known, err := sp.PushAddrMsg(addrs)
	if err != nil {
		peerLog.Errorf("Can't push address message to %s: %v", sp.Peer, err)
		sp.Disconnect()
		return
	}
	sp.addKnownAddresses(known)
}

//addbanscore增加了持久和衰退的ban score字段
//作为参数传递的值。如果结果分数超过禁令的一半
//阈值，记录一条警告，包括提供的原因。此外，如果
//分数高于禁令阈值，同行将被禁止，并且
//断开的。
func (sp *serverPeer) addBanScore(persistent, transient uint32, reason string) {
//如果禁用禁止，则不会记录警告，也不会计算分数。
	if cfg.DisableBanning {
		return
	}
	if sp.isWhitelisted {
		peerLog.Debugf("Misbehaving whitelisted peer %s: %s", sp, reason)
		return
	}

	warnThreshold := cfg.BanThreshold >> 1
	if transient == 0 && persistent == 0 {
//分数没有增加，但警告信息仍然存在
//如果分数高于警告阈值，则记录。
		score := sp.banScore.Int()
		if score > warnThreshold {
			peerLog.Warnf("Misbehaving peer %s: %s -- ban score is %d, "+
				"it was not increased this time", sp, reason, score)
		}
		return
	}
	score := sp.banScore.Increase(persistent, transient)
	if score > warnThreshold {
		peerLog.Warnf("Misbehaving peer %s: %s -- ban score increased to %d",
			sp, reason, score)
		if score > cfg.BanThreshold {
			peerLog.Warnf("Misbehaving peer %s -- banning and disconnecting",
				sp)
			sp.server.BanPeer(sp)
			sp.Disconnect()
		}
	}
}

//HasServices返回所提供的公布服务标志是否具有
//所有提供的所需服务标志集。
func hasServices(advertised, desired wire.ServiceFlag) bool {
	return advertised&desired == desired
}

//当对等端收到版本比特币消息时调用onversion
//用于协商协议版本详细信息以及启动
//通信。
func (sp *serverPeer) OnVersion(_ *peer.Peer, msg *wire.MsgVersion) *wire.MsgReject {
//使用发布的出站服务更新地址管理器
//连接以防更改。对于入站未执行此操作
//连接以帮助防止恶意行为，并在
//在模拟测试网络上运行，因为它仅用于
//连接到指定的对等点并积极避免广告和
//连接到发现的对等机。
//
//注意：这是在拒绝年龄太大而不能确保
//无论新的最低协议版本是什么，它都会更新。
//已强制，远程节点尚未升级。
	isInbound := sp.Inbound()
	remoteAddr := sp.NA()
	addrManager := sp.server.addrManager
	if !cfg.SimNet && !isInbound {
		addrManager.SetServices(remoteAddr, msg.Services)
	}

//忽略具有太旧的Protcol版本的对等机。同辈
//协商逻辑将在回调返回后断开它。
	if msg.ProtocolVersion < int32(peer.MinAcceptableProtocolVersion) {
		return nil
	}

//拒绝不是完整节点的出站对等端。
	wantServices := wire.SFNodeNetwork
	if !isInbound && !hasServices(msg.Services, wantServices) {
		missingServices := wantServices & ^msg.Services
		srvrLog.Debugf("Rejecting peer %s with services %v due to not "+
			"providing desired services %v", sp.Peer, msg.Services,
			missingServices)
		reason := fmt.Sprintf("required services %#x not offered",
			uint64(missingServices))
		return wire.NewMsgReject(msg.Command(), wire.RejectNonstandard, reason)
	}

//更新地址管理器并从
//出站连接的远程对等机。运行时跳过此项
//在模拟测试网络上，因为它只用于连接
//向指定的同行，并积极避免广告和连接到
//发现了对等点。
	if !cfg.SimNet && !isInbound {
//软分叉激活后，只进行出站
//与同龄人的联系，如果他们标记自己是赛格威特
//启用。
		chain := sp.server.chain
		segwitActive, err := chain.IsDeploymentActive(chaincfg.DeploymentSegwit)
		if err != nil {
			peerLog.Errorf("Unable to query for segwit soft-fork state: %v",
				err)
			return nil
		}

		if segwitActive && !sp.IsWitnessEnabled() {
			peerLog.Infof("Disconnecting non-segwit peer %v, isn't segwit "+
				"enabled and we need more segwit enabled peers", sp)
			sp.Disconnect()
			return nil
		}

//当服务器接受传入时公布本地地址
//它相信自己与最著名的提示很接近。
		if !cfg.DisableListen && sp.server.syncManager.IsCurrent() {
//获取最匹配的地址。
			lna := addrManager.GetBestLocalAddress(remoteAddr)
			if addrmgr.IsRoutable(lna) {
//对等方已经知道的筛选器地址。
				addresses := []*wire.NetAddress{lna}
				sp.pushAddrMsg(addresses)
			}
		}

//如果服务器地址管理器需要，请求已知地址
//更多，并且对等端的协议版本足够新
//包括带有地址的时间戳。
		hasTimestamp := sp.ProtocolVersion() >= wire.NetAddressTimeVersion
		if addrManager.NeedMoreAddresses() && hasTimestamp {
			sp.QueueMessage(wire.NewMsgGetAddr(), nil)
		}

//将地址标记为已知的好地址。
		addrManager.Good(remoteAddr)
	}

//添加远程对等时间作为创建偏移的示例
//保持网络时间同步的本地时钟。
	sp.server.timeSource.AddTimeSample(sp.Addr(), msg.Timestamp)

//向同步管理器发送信号此对等方是新的同步候选。
	sp.server.syncManager.NewPeer(sp.Peer)

//选择是否在筛选命令之前中继事务
//收到。
	sp.setDisableRelayTx(msg.DisableRelayTx)

//向服务器添加有效的对等机。
	sp.server.AddPeer(sp)
	return nil
}

//当对等端收到mempool比特币消息时，调用onmempool。
//它创建并发送带有内存内容的清单消息
//最多可容纳每条消息的最大库存量。当对方有
//布卢姆过滤器加载后，内容物会相应过滤。
func (sp *serverPeer) OnMemPool(_ *peer.Peer, msg *wire.MsgMemPool) {
//仅当服务器具有Bloom筛选时才允许mempool请求
//启用。
	if sp.server.services&wire.SFNodeBloom != wire.SFNodeBloom {
		peerLog.Debugf("peer %v sent mempool request with bloom "+
			"filtering disabled -- disconnecting", sp)
		sp.Disconnect()
		return
	}

//为了防止洪水泛滥，禁令分数不断提高。
//如果出现以下情况，禁令分数将累积并通过禁令阈值：
//mempool消息来自对等机。分数每分钟递减到
//价值的一半。
	sp.addBanScore(0, 33, "mempool")

//使用中的可用交易记录生成库存消息
//事务内存池。限制在允许的最大库存
//每个消息。newmsginvsizehint函数自动限制
//传递的提示达到了允许的最大值，因此可以安全地传递它
//这里不需要再检查。
	txMemPool := sp.server.txMemPool
	txDescs := txMemPool.TxDescs()
	invMsg := wire.NewMsgInvSizeHint(uint(len(txDescs)))

	for _, txDesc := range txDescs {
//或者在没有Bloom筛选器时添加所有事务，
//或者只有符合筛选条件的事务
//一个。
		if !sp.filter.IsLoaded() || sp.filter.MatchTxAndUpdate(txDesc.Tx) {
			iv := wire.NewInvVect(wire.InvTypeTx, txDesc.Tx.Hash())
			invMsg.AddInvVect(iv)
			if len(invMsg.InvList)+1 > wire.MaxInvPerMsg {
				break
			}
		}
	}

//如果有要发送的内容，则发送库存消息。
	if len(invMsg.InvList) > 0 {
		sp.QueueMessage(invMsg, nil)
	}
}

//当对等端收到Tx比特币消息时，会调用OnTx。信息块
//直到比特币交易被完全处理。解除锁定
//处理程序这不会通过单个线程序列化所有事务
//事务不依赖于前一个事务，而采用类似块的线性方式。
func (sp *serverPeer) OnTx(_ *peer.Peer, msg *wire.MsgTx) {
	if cfg.BlocksOnly {
		peerLog.Tracef("Ignoring tx %v from %v - blocksonly enabled",
			msg.TxHash(), sp)
		return
	}

//将事务添加到对等机的已知清单中。
//将原始msgtx转换为bcutil.tx，这提供了一些便利
//方法和事物，如哈希缓存。
	tx := btcutil.NewTx(msg)
	iv := wire.NewInvVect(wire.InvTypeTx, tx.Hash())
	sp.AddKnownInventory(iv)

//将要由同步管理器处理的事务排队，然后
//有意阻止进一步接收，直到交易完成
//处理过的和已知的好的或坏的。这有助于防止恶意对等
//在断开连接之前排队处理一堆坏事务（或
//断开连接）和浪费内存。
	sp.server.syncManager.QueueTx(tx, sp.Peer, sp.txProcessed)
	<-sp.txProcessed
}

//当对等端接收到块比特币消息时，会调用onblock。它
//块，直到比特币块被完全处理。
func (sp *serverPeer) OnBlock(_ *peer.Peer, msg *wire.MsgBlock, buf []byte) {
//将原始msgblock转换为bcutil.block，它提供
//便利的方法和诸如哈希缓存之类的东西。
	block := btcutil.NewBlockFromBlockAndBytes(msg, buf)

//将块添加到对等机的已知清单中。
	iv := wire.NewInvVect(wire.InvTypeBlock, block.Hash())
	sp.AddKnownInventory(iv)

//将要由块处理的块排队
//经理故意阻止进一步接收
//直到比特币块被完全处理和知道
//好与坏。这有助于防止恶意对等
//从排队之前的一堆坏街区
//断开（或断开）和浪费
//记忆。此外，这种行为还取决于
//至少通过试块验收测试工具作为
//参考实现过程块
//线程，因此阻止更多消息，直到
//比特币区块已完全处理。
	sp.server.syncManager.QueueBlock(block, sp.Peer, sp.blockProcessed)
	<-sp.blockProcessed
}

//当一个对等方接收到一条INV比特币消息时，会调用OnInV，并且
//用于检查远程对等机公布的清单并作出反应
//因此。我们将消息传递给BlockManager，它将调用
//带有任何适当响应的队列消息。
func (sp *serverPeer) OnInv(_ *peer.Peer, msg *wire.MsgInv) {
	if !cfg.BlocksOnly {
		if len(msg.InvList) > 0 {
			sp.server.syncManager.QueueInv(msg, sp.Peer)
		}
		return
	}

	newInv := wire.NewMsgInvSizeHint(uint(len(msg.InvList)))
	for _, invVect := range msg.InvList {
		if invVect.Type == wire.InvTypeTx {
			peerLog.Tracef("Ignoring tx %v in inv from %v -- "+
				"blocksonly enabled", invVect.Hash, sp)
			if sp.ProtocolVersion() >= wire.BIP0037Version {
				peerLog.Infof("Peer %v is announcing "+
					"transactions -- disconnecting", sp)
				sp.Disconnect()
				return
			}
			continue
		}
		err := newInv.AddInvVect(invVect)
		if err != nil {
			peerLog.Errorf("Failed to add inventory vector: %v", err)
			break
		}
	}

	if len(newInv.InvList) > 0 {
		sp.server.syncManager.QueueInv(newInv, sp.Peer)
	}
}

//当对等端收到头比特币时调用OnHeaders
//消息。消息将传递给同步管理器。
func (sp *serverPeer) OnHeaders(_ *peer.Peer, msg *wire.MsgHeaders) {
	sp.server.syncManager.QueueHeaders(msg, sp.Peer)
}

//当对等端接收到getdata比特币消息并且
//用于传递块和事务信息。
func (sp *serverPeer) OnGetData(_ *peer.Peer, msg *wire.MsgGetData) {
	numAdded := 0
	notFound := wire.NewMsgNotFound()

	length := len(msg.InvList)
//为防止资源枯竭，采用了一种逐渐降低的禁令分数。
//异常大的库存查询。
//在短时间内请求超过最大库存向量长度
//时间段产生的分数高于默认禁止阈值。持续的
//突发的小请求不会受到惩罚，因为这可能会禁止
//执行IBD的对等机。
//这个增量分数每分钟衰减到其值的一半。
	sp.addBanScore(0, uint32(length)*99/wire.MaxInvPerMsg, "getdata")

//我们定期等待这个等待通道以防止排队
//比我们在合理时间内发送的数据多得多，浪费了内存。
//在数据库提取后等待下一个
//提供一点管道。
	var waitChan chan struct{}
	doneChan := make(chan struct{}, 1)

	for i, iv := range msg.InvList {
		var c chan struct{}
//如果这是我们最后发送的信息。
		if i == length-1 && len(notFound.InvList) == 0 {
			c = doneChan
		} else if (i+1)%3 == 0 {
//缓冲以避免发送goroutine块。
			c = make(chan struct{}, 1)
		}
		var err error
		switch iv.Type {
		case wire.InvTypeWitnessTx:
			err = sp.server.pushTxMsg(sp, &iv.Hash, c, waitChan, wire.WitnessEncoding)
		case wire.InvTypeTx:
			err = sp.server.pushTxMsg(sp, &iv.Hash, c, waitChan, wire.BaseEncoding)
		case wire.InvTypeWitnessBlock:
			err = sp.server.pushBlockMsg(sp, &iv.Hash, c, waitChan, wire.WitnessEncoding)
		case wire.InvTypeBlock:
			err = sp.server.pushBlockMsg(sp, &iv.Hash, c, waitChan, wire.BaseEncoding)
		case wire.InvTypeFilteredWitnessBlock:
			err = sp.server.pushMerkleBlockMsg(sp, &iv.Hash, c, waitChan, wire.WitnessEncoding)
		case wire.InvTypeFilteredBlock:
			err = sp.server.pushMerkleBlockMsg(sp, &iv.Hash, c, waitChan, wire.BaseEncoding)
		default:
			peerLog.Warnf("Unknown type in inventory request %d",
				iv.Type)
			continue
		}
		if err != nil {
			notFound.AddInvVect(iv)

//当获取最终条目失败时
//完成的频道是因为那里才被发送进来的
//未发现未清存货，消耗
//因为现在没有找到存货
//这将暂时使用通道。
			if i == len(msg.InvList)-1 && c != nil {
				<-c
			}
		}
		numAdded++
		waitChan = c
	}
	if len(notFound.InvList) != 0 {
		sp.QueueMessage(notFound, doneChan)
	}

//等待消息发送。我们可以发送大量的数据
//这将使对等机在相当长的时间内保持忙碌。
//在这段时间内，我们不会通过他们处理任何其他事情，以便
//想一想我们什么时候应该收到他们的回信-否则就是闲人
//当我们只完成了一半发送块时，超时可能会触发。
	if numAdded > 0 {
		<-doneChan
	}
}

//当对等端收到getBlocks比特币时调用ongetBlocks。
//消息。
func (sp *serverPeer) OnGetBlocks(_ *peer.Peer, msg *wire.MsgGetBlocks) {
//根据块在最佳链中查找最新的已知块
//定位并获取所有块散列，直到
//Wire.MaxBlocksPerMsg已被提取或提供的停止哈希为
//遇到。
//
//如果Genesis块后面没有其他块，
//提供的定位器是已知的。这确实意味着客户端将启动
//如果提供未知区块定位器，则与Genesis区块一起结束。
//
//这反映了引用实现中的行为。
	chain := sp.server.chain
	hashList := chain.LocateBlocks(msg.BlockLocatorHashes, &msg.HashStop,
		wire.MaxBlocksPerMsg)

//生成库存消息。
	invMsg := wire.NewMsgInv()
	for i := range hashList {
		iv := wire.NewInvVect(wire.InvTypeBlock, &hashList[i])
		invMsg.AddInvVect(iv)
	}

//如果有要发送的内容，则发送库存消息。
	if len(invMsg.InvList) > 0 {
		invListLen := len(invMsg.InvList)
		if invListLen == wire.MaxBlocksPerMsg {
//故意使用最终哈希的副本，因此
//不是库存切片的引用，
//将阻止整个切片符合条件
//一旦发送给GC。
			continueHash := invMsg.InvList[invListLen-1].Hash
			sp.continueHash = &continueHash
		}
		sp.QueueMessage(invMsg, nil)
	}
}

//OnGetHeaders是在对等端接收到GetHeaders比特币时调用的。
//消息。
func (sp *serverPeer) OnGetHeaders(_ *peer.Peer, msg *wire.MsgGetHeaders) {
//如果不同步，则忽略GetHeaders请求。
	if !sp.server.syncManager.IsCurrent() {
		return
	}

//根据块在最佳链中查找最新的已知块
//定位并在其之后获取所有头文件，直到
//已获取Wire.MaxBlockHeadersPermsg或提供的停止
//遇到哈希。
//
//如果Genesis块后面没有其他块，
//提供的定位器是已知的。这确实意味着客户端将启动
//如果提供未知区块定位器，则与Genesis区块一起结束。
//
//这反映了引用实现中的行为。
	chain := sp.server.chain
	headers := chain.LocateHeaders(msg.BlockLocatorHashes, &msg.HashStop)

//将找到的头发送到请求的对等端。
	blockHeaders := make([]*wire.BlockHeader, len(headers))
	for i := range headers {
		blockHeaders[i] = &headers[i]
	}
	sp.QueueMessage(&wire.MsgHeaders{Headers: blockHeaders}, nil)
}

//当对等端收到getfilters比特币消息时，调用ongetcfilters。
func (sp *serverPeer) OnGetCFilters(_ *peer.Peer, msg *wire.MsgGetCFilters) {
//如果不同步，忽略getcpilters请求。
	if !sp.server.syncManager.IsCurrent() {
		return
	}

//我们还将确保远程方正在请求
//我们目前实际维护的过滤器。
	switch msg.FilterType {
	case wire.GCSFilterRegular:
		break

	default:
		peerLog.Debug("Filter request for unknown filter: %v",
			msg.FilterType)
		return
	}

	hashes, err := sp.server.chain.HeightToHashRange(
		int32(msg.StartHeight), &msg.StopHash, wire.MaxGetCFiltersReqRange,
	)
	if err != nil {
		peerLog.Debugf("Invalid getcfilters request: %v", err)
		return
	}

//从[]chainhash.hash创建[]*chainhash.hash传递给
//筛选yblockhashes。
	hashPtrs := make([]*chainhash.Hash, len(hashes))
	for i := range hashes {
		hashPtrs[i] = &hashes[i]
	}

	filters, err := sp.server.cfIndex.FiltersByBlockHashes(
		hashPtrs, msg.FilterType,
	)
	if err != nil {
		peerLog.Errorf("Error retrieving cfilters: %v", err)
		return
	}

	for i, filterBytes := range filters {
		if len(filterBytes) == 0 {
			peerLog.Warnf("Could not obtain cfilter for %v",
				hashes[i])
			return
		}

		filterMsg := wire.NewMsgCFilter(
			msg.FilterType, &hashes[i], filterBytes,
		)
		sp.QueueMessage(filterMsg, nil)
	}
}

//当对等端收到getcfheader比特币消息时，调用ongetcfheaders。
func (sp *serverPeer) OnGetCFHeaders(_ *peer.Peer, msg *wire.MsgGetCFHeaders) {
//如果不同步，忽略getcFilterHeader请求。
	if !sp.server.syncManager.IsCurrent() {
		return
	}

//我们还将确保远程方正在请求
//当前实际维护的过滤器的标题。
	switch msg.FilterType {
	case wire.GCSFilterRegular:
		break

	default:
		peerLog.Debug("Filter request for unknown headers for "+
			"filter: %v", msg.FilterType)
		return
	}

	startHeight := int32(msg.StartHeight)
	maxResults := wire.MaxCFHeadersPerMsg

//如果startheight为正，则获取前置块哈希，以便
//无法填充PrevFilterHeader字段。
	if msg.StartHeight > 0 {
		startHeight--
		maxResults++
	}

//从块索引中获取哈希。
	hashList, err := sp.server.chain.HeightToHashRange(
		startHeight, &msg.StopHash, maxResults,
	)
	if err != nil {
		peerLog.Debugf("Invalid getcfheaders request: %v", err)
	}

//如果startheight大于
//stophash，我们提取一个有效的哈希范围，包括前面的
//过滤头。
	if len(hashList) == 0 || (msg.StartHeight > 0 && len(hashList) == 1) {
		peerLog.Debug("No results for getcfheaders request")
		return
	}

//从[]chainhash.hash创建[]*chainhash.hash传递给
//FilterHeadersByBlockHashes。
	hashPtrs := make([]*chainhash.Hash, len(hashList))
	for i := range hashList {
		hashPtrs[i] = &hashList[i]
	}

//从数据库中获取所有块的原始筛选器哈希字节。
	filterHashes, err := sp.server.cfIndex.FilterHashesByBlockHashes(
		hashPtrs, msg.FilterType,
	)
	if err != nil {
		peerLog.Errorf("Error retrieving cfilter hashes: %v", err)
		return
	}

//生成并发送cfheaders消息。
	headersMsg := wire.NewMsgCFHeaders()

//填充PrevFilterHeader字段。
	if msg.StartHeight > 0 {
		prevBlockHash := &hashList[0]

//从中获取原始提交的筛选器头字节
//数据库。
		headerBytes, err := sp.server.cfIndex.FilterHeaderByBlockHash(
			prevBlockHash, msg.FilterType)
		if err != nil {
			peerLog.Errorf("Error retrieving CF header: %v", err)
			return
		}
		if len(headerBytes) == 0 {
			peerLog.Warnf("Could not obtain CF header for %v", prevBlockHash)
			return
		}

//将哈希反序列化为PrevFilterHeader。
		err = headersMsg.PrevFilterHeader.SetBytes(headerBytes)
		if err != nil {
			peerLog.Warnf("Committed filter header deserialize "+
				"failed: %v", err)
			return
		}

		hashList = hashList[1:]
		filterHashes = filterHashes[1:]
	}

//填充头饰。
	for i, hashBytes := range filterHashes {
		if len(hashBytes) == 0 {
			peerLog.Warnf("Could not obtain CF hash for %v", hashList[i])
			return
		}

//反序列化哈希。
		filterHash, err := chainhash.NewHash(hashBytes)
		if err != nil {
			peerLog.Warnf("Committed filter hash deserialize "+
				"failed: %v", err)
			return
		}

		headersMsg.AddCFHash(filterHash)
	}

	headersMsg.FilterType = msg.FilterType
	headersMsg.StopHash = msg.StopHash

	sp.QueueMessage(headersMsg, nil)
}

//当对等端收到getcfcheckpt比特币消息时，调用ongetcfcheckpt。
func (sp *serverPeer) OnGetCFCheckpt(_ *peer.Peer, msg *wire.MsgGetCFCheckpt) {
//如果不同步，忽略getcfcheckpt请求。
	if !sp.server.syncManager.IsCurrent() {
		return
	}

//我们还将确保远程方正在请求
//我们目前实际维护的过滤器的检查点。
	switch msg.FilterType {
	case wire.GCSFilterRegular:
		break

	default:
		peerLog.Debug("Filter request for unknown checkpoints for "+
			"filter: %v", msg.FilterType)
		return
	}

//现在我们知道客户机正在获取我们知道的过滤器，
//我们将获取每个检查点间隔的块哈希，以便
//与缓存进行比较，必要时创建新的检查点。
	blockHashes, err := sp.server.chain.IntervalBlockHashes(
		&msg.StopHash, wire.CFCheckptInterval,
	)
	if err != nil {
		peerLog.Debugf("Invalid getcfilters request: %v", err)
		return
	}

	checkptMsg := wire.NewMsgCFCheckpt(
		msg.FilterType, &msg.StopHash, len(blockHashes),
	)

//获取当前的现有缓存，以便我们可以决定是否需要
//扩展它，或者如果它足够的话。
	sp.server.cfCheckptCachesMtx.RLock()
	checkptCache := sp.server.cfCheckptCaches[msg.FilterType]

//如果块哈希集超出了缓存的当前大小，
//然后我们将扩展缓存的大小并保留写操作
//锁。
	var updateCache bool
	if len(blockHashes) > len(checkptCache) {
//既然我们知道需要修改缓存的大小，
//我们将释放读锁并获取写锁
//可能扩展缓存大小。
		sp.server.cfCheckptCachesMtx.RUnlock()

		sp.server.cfCheckptCachesMtx.Lock()
		defer sp.server.cfCheckptCachesMtx.Unlock()

//既然我们有了写锁，我们会再检查一下
//可能缓存已展开。
		checkptCache = sp.server.cfCheckptCaches[msg.FilterType]

//如果我们仍然需要扩展缓存，那么我们将标记它
//我们需要更新下面的缓存并扩展
//缓存的大小。
		if len(blockHashes) > len(checkptCache) {
			updateCache = true

			additionalLength := len(blockHashes) - len(checkptCache)
			newEntries := make([]cfHeaderKV, additionalLength)

			peerLog.Infof("Growing size of checkpoint cache from %v to %v "+
				"block hashes", len(checkptCache), len(blockHashes))

			checkptCache = append(
				sp.server.cfCheckptCaches[msg.FilterType],
				newEntries...,
			)
		}
	} else {
//否则，我们将保留剩余的读取锁
//这种方法。
		defer sp.server.cfCheckptCachesMtx.RUnlock()

		peerLog.Tracef("Serving stale cache of size %v",
			len(checkptCache))
	}

//既然我们知道缓存的大小合适，那么我们将迭代
//向后直到找到块散列。我们尽可能做到这一点
//重新组织已发生，因此数据库中的项目现在位于主要中国
//缓存已部分失效。
	var forkIdx int
	for forkIdx = len(blockHashes); forkIdx > 0; forkIdx-- {
		if checkptCache[forkIdx-1].blockHash == blockHashes[forkIdx-1] {
			break
		}
	}

//现在我们知道了多少缓存与此相关
//查询时，我们将按原样用缓存填充检查点消息。
//下面，我们将填充缓存的新元素。
	for i := 0; i < forkIdx; i++ {
		checkptMsg.AddCFHeader(&checkptCache[i].filterHeader)
	}

//我们现在将收集超出缓存范围的哈希集，因此
//可以查找筛选器头以填充最终缓存。
	blockHashPtrs := make([]*chainhash.Hash, 0, len(blockHashes)-forkIdx)
	for i := forkIdx; i < len(blockHashes); i++ {
		blockHashPtrs = append(blockHashPtrs, &blockHashes[i])
	}
	filterHeaders, err := sp.server.cfIndex.FilterHeadersByBlockHashes(
		blockHashPtrs, msg.FilterType,
	)
	if err != nil {
		peerLog.Errorf("Error retrieving cfilter headers: %v", err)
		return
	}

//既然我们有了完整的过滤器头集，我们将把它们添加到
//检查点消息，还可以更新缓存。
	for i, filterHeaderBytes := range filterHeaders {
		if len(filterHeaderBytes) == 0 {
			peerLog.Warnf("Could not obtain CF header for %v",
				blockHashPtrs[i])
			return
		}

		filterHeader, err := chainhash.NewHash(filterHeaderBytes)
		if err != nil {
			peerLog.Warnf("Committed filter header deserialize "+
				"failed: %v", err)
			return
		}

		checkptMsg.AddCFHeader(filterHeader)

//如果新的主链比缓存中的长，
//然后我们将超越分叉点。
		if updateCache {
			checkptCache[forkIdx+i] = cfHeaderKV{
				blockHash:    blockHashes[forkIdx+i],
				filterHeader: *filterHeader,
			}
		}
	}

//最后，如果需要，我们将更新缓存，并发送最终
//向请求的对等端返回消息。
	if updateCache {
		sp.server.cfCheckptCaches[msg.FilterType] = checkptCache
	}

	sp.QueueMessage(checkptMsg, nil)
}

//如果服务器未配置为
//允许使用布卢姆过滤器。此外，如果对等方已协商到协议
//足以观察Bloom过滤器服务支持位的版本，
//它将被禁止，因为它是故意违反协议。
func (sp *serverPeer) enforceNodeBloomFlag(cmd string) bool {
	if sp.server.services&wire.SFNodeBloom != wire.SFNodeBloom {
//如果协议版本足够高，
//对等方故意违反协议，禁止
//启用。
//
//注意：即使addBanScore函数已经检查过了
//不管是否启用了禁止，这里也会检查它。
//以确保记录违规行为，并且对等机
//断开连接。
		if sp.ProtocolVersion() >= wire.BIP0111Version &&
			!cfg.DisableBanning {

//断开对等机的连接，无论它是否
//被禁止的
			sp.addBanScore(100, 0, cmd)
			sp.Disconnect()
			return false
		}

//断开对等机的连接，无论协议版本或禁止。
//状态。
		peerLog.Debugf("%s sent an unsupported %s request -- "+
			"disconnecting", sp, cmd)
		sp.Disconnect()
		return false
	}

	return true
}

//当对等端接收到feefilter比特币消息并且
//被远程对等方用于请求没有具有费率的事务
//低于规定值的则向其列出清单。同行将
//如果提供的费用筛选值无效，则断开连接。
func (sp *serverPeer) OnFeeFilter(_ *peer.Peer, msg *wire.MsgFeeFilter) {
//检查通过的最低费用是否为有效金额。
	if msg.MinFee < 0 || msg.MinFee > btcutil.MaxSatoshi {
		peerLog.Debugf("Peer %v sent an invalid feefilter '%v' -- "+
			"disconnecting", sp, btcutil.Amount(msg.MinFee))
		sp.Disconnect()
		return
	}

	atomic.StoreInt64(&sp.feeFilter, msg.MinFee)
}

//当对等方收到filteradd比特币时调用onfilteradd
//远程对等方使用消息和向已加载的Bloom添加数据
//过滤器。如果在此情况下未加载筛选器，则将断开对等机的连接。
//收到消息或服务器未配置为允许Bloom筛选器。
func (sp *serverPeer) OnFilterAdd(_ *peer.Peer, msg *wire.MsgFilterAdd) {
//根据节点Bloom服务标志和
//协商的协议版本。
	if !sp.enforceNodeBloomFlag(msg.Command()) {
		return
	}

	if !sp.filter.IsLoaded() {
		peerLog.Debugf("%s sent a filteradd request with no filter "+
			"loaded -- disconnecting", sp)
		sp.Disconnect()
		return
	}

	sp.filter.Add(msg.Data)
}

//当对等方收到filterclear比特币时，调用onfilterclear。
//远程对等方使用消息和清除已加载的Bloom筛选器。
//如果此消息未加载筛选器，则将断开对等机的连接。
//接收到或服务器未配置为允许Bloom筛选器。
func (sp *serverPeer) OnFilterClear(_ *peer.Peer, msg *wire.MsgFilterClear) {
//根据节点Bloom服务标志和
//协商的协议版本。
	if !sp.enforceNodeBloomFlag(msg.Command()) {
		return
	}

	if !sp.filter.IsLoaded() {
		peerLog.Debugf("%s sent a filterclear request with no "+
			"filter loaded -- disconnecting", sp)
		sp.Disconnect()
		return
	}

	sp.filter.Unload()
}

//当对等端收到filterload比特币时调用onfilterload。
//消息和它用于加载应用于
//传递与过滤器匹配的Merkle块和关联事务。
//如果服务器未配置为允许Bloom，则对等机将断开连接。
//过滤器。
func (sp *serverPeer) OnFilterLoad(_ *peer.Peer, msg *wire.MsgFilterLoad) {
//根据节点Bloom服务标志和
//协商的协议版本。
	if !sp.enforceNodeBloomFlag(msg.Command()) {
		return
	}

	sp.setDisableRelayTx(false)

	sp.filter.Reload(msg)
}

//当对等端收到getaddr比特币消息时调用ongetaddr。
//用于向对等方提供地址中的已知地址
//经理。
func (sp *serverPeer) OnGetAddr(_ *peer.Peer, msg *wire.MsgGetAddr) {
//在模拟测试上运行时不返回任何地址
//网络。这有助于防止网络成为另一个
//公共测试网络，因为它将无法了解其他
//没有特别提供的对等机。
	if cfg.SimNet {
		return
	}

//不接受来自出站对等方的getaddr请求。这减少了
//指纹攻击。
	if !sp.Inbound() {
		peerLog.Debugf("Ignoring getaddr request from outbound peer ",
			"%v", sp)
		return
	}

//每个连接只允许一个getaddr请求以阻止
//发票公告地址盖章。
	if sp.sentAddrs {
		peerLog.Debugf("Ignoring repeated getaddr request from peer ",
			"%v", sp)
		return
	}
	sp.sentAddrs = true

//从地址管理器中获取当前已知地址。
	addrCache := sp.server.addrManager.AddressCache()

//按地址。
	sp.pushAddrMsg(addrCache)
}

//当对等端接收到addr比特币消息并且
//用于通知服务器有关公布的地址。
func (sp *serverPeer) OnAddr(_ *peer.Peer, msg *wire.MsgAddr) {
//在模拟测试网络上运行时忽略地址。这个
//有助于防止网络成为另一个公共测试网络
//因为它将无法了解其他没有
//具体规定。
	if cfg.SimNet {
		return
	}

//忽略不包含时间戳的旧式地址。
	if sp.ProtocolVersion() < wire.NetAddressTimeVersion {
		return
	}

//没有地址的邮件无效。
	if len(msg.AddrList) == 0 {
		peerLog.Errorf("Command [%s] from %s does not contain any addresses",
			msg.Command(), sp.Peer)
		sp.Disconnect()
		return
	}

	for _, na := range msg.AddrList {
//如果断开连接，请不要添加更多地址。
		if !sp.Connected() {
			return
		}

//如果时间戳超过24小时，则将其设置为5天前
//所以这个地址是第一个
//需要空间时移除。
		now := time.Now()
		if na.Timestamp.After(now.Add(time.Minute * 10)) {
			na.Timestamp = now.Add(-1 * time.Hour * 24 * 5)
		}

//将地址添加到此对等机的已知地址。
		sp.addKnownAddresses([]*wire.NetAddress{na})
	}

//将地址添加到服务器地址管理器。地址管理器处理
//防止重复地址等细节，max
//地址和上次看到的更新。
//XXX比特币在这里被罚2小时，我们想这样做吗？
//一样吗？
	sp.server.addrManager.AddAddresses(msg.AddrList, sp.NA())
}

//当对等端接收到消息并用于更新时调用OnRead
//服务器接收的字节。
func (sp *serverPeer) OnRead(_ *peer.Peer, bytesRead int, msg wire.Message, err error) {
	sp.server.AddBytesReceived(uint64(bytesRead))
}

//当对等端发送消息并用于更新时调用OnWrite
//服务器发送的字节。
func (sp *serverPeer) OnWrite(_ *peer.Peer, bytesWritten int, msg wire.Message, err error) {
	sp.server.AddBytesSent(uint64(bytesWritten))
}

//randomunt16number返回指定输入范围内的随机uint16。注释
//范围是按零排序的；如果通过1800，您将得到
//值从0到1800。
func randomUint16Number(max uint16) uint16 {
//为了避免模偏差，确保
//[0，max）的概率相等，必须对随机数进行抽样。
//从范围限制为
//模量。
	var randomNumber uint16
	var limitRange = (math.MaxUint16 / max) * max
	for {
		binary.Read(rand.Reader, binary.LittleEndian, &randomNumber)
		if randomNumber < limitRange {
			return (randomNumber % max)
		}
	}
}

//addrebroadcastinventory将“iv”添加到要
//以随机间隔重铸，直到它们出现在一个街区。
func (s *server) AddRebroadcastInventory(iv *wire.InvVect, data interface{}) {
//关闭时忽略。
	if atomic.LoadInt32(&s.shutdown) != 0 {
		return
	}

	s.modifyRebroadcastInv <- broadcastInventoryAdd{invVect: iv, data: data}
}

//RemoveRoboadcastinventory从要删除的项目列表中删除“iv”
//重新铸造（如有）。
func (s *server) RemoveRebroadcastInventory(iv *wire.InvVect) {
//关闭时忽略。
	if atomic.LoadInt32(&s.shutdown) != 0 {
		return
	}

	s.modifyRebroadcastInv <- broadcastInventoryDel(iv)
}

//RelayTransactions为所有
//已将事务传递给所有连接的对等方。
func (s *server) relayTransactions(txns []*mempool.TxDesc) {
	for _, txD := range txns {
		iv := wire.NewInvVect(wire.InvTypeTx, txD.Tx.Hash())
		s.RelayInventory(iv, txD)
	}
}

//AnnounceNewTransactions生成和传递库存向量并通知
//WebSocket和GetBlockTemplate的长轮询客户端
//交易。每当有新的事务时都应调用此函数
//添加到mempool。
func (s *server) AnnounceNewTransactions(txns []*mempool.TxDesc) {
//生成并中继所有新接受的库存向量
//交易。
	s.relayTransactions(txns)

//通知WebSocket和GetBlockTemplate长轮询客户端
//新接受的交易。
	if s.rpcServer != nil {
		s.rpcServer.NotifyNewTransactions(txns)
	}
}

//事务在主链上有一个确认。现在我们可以把它标为不
//需要更长时间的重播。
func (s *server) TransactionConfirmed(tx *btcutil.Tx) {
//只有在RPC服务器处于活动状态时才需要重新广播。
	if s.rpcServer == nil {
		return
	}

	iv := wire.NewInvVect(wire.InvTypeTx, tx.Hash())
	s.RemoveRebroadcastInventory(iv)
}

//pushtxmsg将所提供事务哈希的tx消息发送到
//已连接的对等机。如果事务哈希未知，则返回错误。
func (s *server) pushTxMsg(sp *serverPeer, hash *chainhash.Hash, doneChan chan<- struct{},
	waitChan <-chan struct{}, encoding wire.MessageEncoding) error {

//尝试从池中提取请求的事务。一
//可以先打电话检查是否存在，但只需尝试
//以相同的行为获取丢失的事务结果。
	tx, err := s.txMemPool.FetchTransaction(hash)
	if err != nil {
		peerLog.Tracef("Unable to fetch tx %v from transaction "+
			"pool: %v", hash, err)

		if doneChan != nil {
			doneChan <- struct{}{}
		}
		return err
	}

//一旦我们获取了数据，就等待之前的任何操作完成。
	if waitChan != nil {
		<-waitChan
	}

	sp.QueueMessageWithEncoding(tx.MsgTx(), doneChan, encoding)

	return nil
}

//pushblockmsg将所提供的块哈希的块消息发送到
//已连接的对等机。如果块哈希未知，则返回错误。
func (s *server) pushBlockMsg(sp *serverPeer, hash *chainhash.Hash, doneChan chan<- struct{},
	waitChan <-chan struct{}, encoding wire.MessageEncoding) error {

//从数据库中提取原始块字节。
	var blockBytes []byte
	err := sp.server.db.View(func(dbTx database.Tx) error {
		var err error
		blockBytes, err = dbTx.FetchBlock(hash)
		return err
	})
	if err != nil {
		peerLog.Tracef("Unable to fetch requested block hash %v: %v",
			hash, err)

		if doneChan != nil {
			doneChan <- struct{}{}
		}
		return err
	}

//反序列化块。
	var msgBlock wire.MsgBlock
	err = msgBlock.Deserialize(bytes.NewReader(blockBytes))
	if err != nil {
		peerLog.Tracef("Unable to deserialize requested block hash "+
			"%v: %v", hash, err)

		if doneChan != nil {
			doneChan <- struct{}{}
		}
		return err
	}

//一旦我们获取了数据，就等待之前的任何操作完成。
	if waitChan != nil {
		<-waitChan
	}

//如果不发送，我们只发送此消息的频道
//一张发票。
	var dc chan<- struct{}
	continueHash := sp.continueHash
	sendInv := continueHash != nil && continueHash.IsEqual(hash)
	if !sendInv {
		dc = doneChan
	}
	sp.QueueMessageWithEncoding(&msgBlock, dc, encoding)

//当对等端请求在中公布的最后一个块时
//对GetBlocks消息的响应，该消息请求的块数超过
//将适合一条消息，发送一条新的库存消息
//触发它为下一个发出另一个getBlocks消息
//一批存货。
	if sendInv {
		best := sp.server.chain.BestSnapshot()
		invMsg := wire.NewMsgInvSizeHint(1)
		iv := wire.NewInvVect(wire.InvTypeBlock, &best.Hash)
		invMsg.AddInvVect(iv)
		sp.QueueMessage(invMsg, doneChan)
		sp.continueHash = nil
	}
	return nil
}

//pushmerkleblockmsg将提供的块哈希的merkleblock消息发送到
//连接的对等机。因为merkle块要求对等机具有筛选器
//加载后，如果没有加载筛选器，则只会忽略此调用。安
//如果块哈希未知，则返回错误。
func (s *server) pushMerkleBlockMsg(sp *serverPeer, hash *chainhash.Hash,
	doneChan chan<- struct{}, waitChan <-chan struct{}, encoding wire.MessageEncoding) error {

//如果对等端没有加载筛选器，则不要发送响应。
	if !sp.filter.IsLoaded() {
		if doneChan != nil {
			doneChan <- struct{}{}
		}
		return nil
	}

//从数据库中提取原始块字节。
	blk, err := sp.server.chain.BlockByHash(hash)
	if err != nil {
		peerLog.Tracef("Unable to fetch requested block hash %v: %v",
			hash, err)

		if doneChan != nil {
			doneChan <- struct{}{}
		}
		return err
	}

//通过筛选请求的块，根据
//到对等机的筛选器。
	merkle, matchedTxIndices := bloom.NewMerkleBlock(blk, sp.filter)

//一旦我们获取了数据，就等待之前的任何操作完成。
	if waitChan != nil {
		<-waitChan
	}

//发送merkleblock。只发送包含此消息的“完成”频道
//如果以后不发送任何交易。
	var dc chan<- struct{}
	if len(matchedTxIndices) == 0 {
		dc = doneChan
	}
	sp.QueueMessage(merkle, dc)

//最后，发送任何匹配的事务。
	blkTransactions := blk.MsgBlock().Transactions
	for i, txIndex := range matchedTxIndices {
//仅在最终事务上发送完成的通道。
		var dc chan<- struct{}
		if i == len(matchedTxIndices)-1 {
			dc = doneChan
		}
		if txIndex < uint32(len(blkTransactions)) {
			sp.QueueMessageWithEncoding(blkTransactions[txIndex], dc,
				encoding)
		}
	}

	return nil
}

//handleupatepeerheight更新所有已知同龄人的身高
//宣布我们最近接受的阻止。
func (s *server) handleUpdatePeerHeights(state *peerState, umsg updatePeerHeightsMsg) {
	state.forAllPeers(func(sp *serverPeer) {
//原始对等机应该已经具有更新的高度。
		if sp.Peer == umsg.originPeer {
			return
		}

//这是指向底层内存的指针，而底层内存没有
//改变。
		latestBlkHash := sp.LastAnnouncedBlock()

//如果最近没有发布任何新块，则跳过此对等项。
		if latestBlkHash == nil {
			return
		}

//如果对等端最近宣布了一个块，则此块
//匹配我们新接受的块，然后更新它们的块
//高度。
		if *latestBlkHash == *umsg.newHash {
			sp.UpdateLastBlockHeight(umsg.newHeight)
			sp.UpdateLastAnnouncedBlock(nil)
		}
	})
}

//handleaddpeermsg处理添加新对等。它是从
//对等处理程序Goroutine。
func (s *server) handleAddPeerMsg(state *peerState, sp *serverPeer) bool {
	if sp == nil {
		return false
	}

//如果要关闭，请忽略新的对等机。
	if atomic.LoadInt32(&s.shutdown) != 0 {
		srvrLog.Infof("New peer %s ignored - server is shutting down", sp)
		sp.Disconnect()
		return false
	}

//断开禁止的对等机。
	host, _, err := net.SplitHostPort(sp.Addr())
	if err != nil {
		srvrLog.Debugf("can't split hostport %v", err)
		sp.Disconnect()
		return false
	}
	if banEnd, ok := state.banned[host]; ok {
		if time.Now().Before(banEnd) {
			srvrLog.Debugf("Peer %s is banned for another %v - disconnecting",
				host, time.Until(banEnd))
			sp.Disconnect()
			return false
		}

		srvrLog.Infof("Peer %s is no longer banned", host)
		delete(state.banned, host)
	}

//TODO:检查单个IP的最大对等点。

//限制总对等机的最大数目。
	if state.Count() >= cfg.MaxPeers {
		srvrLog.Infof("Max peers reached [%d] - disconnecting peer %s",
			cfg.MaxPeers, sp)
		sp.Disconnect()
//托多：如何处理这里的永久性同龄人？
//他们应该重新安排时间。
		return false
	}

//添加新的对等点并启动它。
	srvrLog.Debugf("New peer %s", sp)
	if sp.Inbound() {
		state.inboundPeers[sp.ID()] = sp
	} else {
		state.outboundGroups[addrmgr.GroupKey(sp.NA())]++
		if sp.persistent {
			state.persistentPeers[sp.ID()] = sp
		} else {
			state.outboundPeers[sp.ID()] = sp
		}
	}

	return true
}

//handledonepermsg处理已发出完成信号的对等机。它是
//从PeerHandler Goroutine调用。
func (s *server) handleDonePeerMsg(state *peerState, sp *serverPeer) {
	var list map[int32]*serverPeer
	if sp.persistent {
		list = state.persistentPeers
	} else if sp.Inbound() {
		list = state.inboundPeers
	} else {
		list = state.outboundPeers
	}
	if _, ok := list[sp.ID()]; ok {
		if !sp.Inbound() && sp.VersionKnown() {
			state.outboundGroups[addrmgr.GroupKey(sp.NA())]--
		}
		if !sp.Inbound() && sp.connReq != nil {
			s.connManager.Disconnect(sp.connReq.ID())
		}
		delete(list, sp.ID())
		srvrLog.Debugf("Removed peer %s", sp)
		return
	}

	if sp.connReq != nil {
		s.connManager.Disconnect(sp.connReq.ID())
	}

//如果对等方已确认，则更新地址“上次看到的时间”
//我们的版本和已经发送给我们的版本。
	if sp.VerAckReceived() && sp.VersionKnown() && sp.NA() != nil {
		s.addrManager.Connected(sp.NA())
	}

//如果我们到了这里，就意味着我们不知道对方的情况
//或者我们故意删除了它。
}

//handlebanpermsg处理禁止同行。它是从
//对等处理程序Goroutine。
func (s *server) handleBanPeerMsg(state *peerState, sp *serverPeer) {
	host, _, err := net.SplitHostPort(sp.Addr())
	if err != nil {
		srvrLog.Debugf("can't split ban peer %s %v", sp.Addr(), err)
		return
	}
	direction := directionString(sp.Inbound())
	srvrLog.Infof("Banned peer %s (%s) for %v", host, direction,
		cfg.BanDuration)
	state.banned[host] = time.Now().Add(cfg.BanDuration)
}

//handlerelayinvmsg处理将库存中继到尚未完成的对等方
//已知拥有它。它是从peerhandler goroutine调用的。
func (s *server) handleRelayInvMsg(state *peerState, msg relayMsg) {
	state.forAllPeers(func(sp *serverPeer) {
		if !sp.Connected() {
			return
		}

//如果资源清册是一个块，并且对等方更喜欢标题，
//生成并发送标题消息，而不是清单
//消息。
		if msg.invVect.Type == wire.InvTypeBlock && sp.WantsHeaders() {
			blockHeader, ok := msg.data.(wire.BlockHeader)
			if !ok {
				peerLog.Warnf("Underlying data for headers" +
					" is not a block header")
				return
			}
			msgHeaders := wire.NewMsgHeaders()
			if err := msgHeaders.AddBlockHeader(&blockHeader); err != nil {
				peerLog.Errorf("Failed to add block"+
					" header: %v", err)
				return
			}
			sp.QueueMessage(msgHeaders, nil)
			return
		}

		if msg.invVect.Type == wire.InvTypeTx {
//当事务具有以下条件时，不要将其中继给对等机
//已禁用事务中继。
			if sp.relayTxDisabled() {
				return
			}

			txD, ok := msg.data.(*mempool.TxDesc)
			if !ok {
				peerLog.Warnf("Underlying data for tx inv "+
					"relay is not a *mempool.TxDesc: %T",
					msg.data)
				return
			}

//如果事务费为每KB，则不中继事务
//小于同级的筛选器。
			feeFilter := atomic.LoadInt64(&sp.feeFilter)
			if feeFilter > 0 && txD.FeePerKB < feeFilter {
				return
			}

//如果有花束，不要中继交易
//已加载筛选器，但事务与之不匹配。
			if sp.filter.IsLoaded() {
				if !sp.filter.MatchTxAndUpdate(txD.Tx) {
					return
				}
			}
		}

//对要与下一批中继的库存进行排队。
//如果该对等方已知
//有存货。
		sp.QueueInventory(msg.invVect)
	})
}

//handlebroadcastmsg处理向对等方广播消息。它被调用
//来自PeerHandler Goroutine。
func (s *server) handleBroadcastMsg(state *peerState, bmsg *broadcastMsg) {
	state.forAllPeers(func(sp *serverPeer) {
		if !sp.Connected() {
			return
		}

		for _, ep := range bmsg.excludePeers {
			if sp == ep {
				return
			}
		}

		sp.QueueMessage(bmsg.message, nil)
	})
}

type getConnCountMsg struct {
	reply chan int32
}

type getPeersMsg struct {
	reply chan []*serverPeer
}

type getOutboundGroup struct {
	key   string
	reply chan int
}

type getAddedNodesMsg struct {
	reply chan []*serverPeer
}

type disconnectNodeMsg struct {
	cmp   func(*serverPeer) bool
	reply chan error
}

type connectNodeMsg struct {
	addr      string
	permanent bool
	reply     chan error
}

type removeNodeMsg struct {
	cmp   func(*serverPeer) bool
	reply chan error
}

//handlequery是其他查询和命令的中心处理程序
//与对等状态相关的goroutine。
func (s *server) handleQuery(state *peerState, querymsg interface{}) {
	switch msg := querymsg.(type) {
	case getConnCountMsg:
		nconnected := int32(0)
		state.forAllPeers(func(sp *serverPeer) {
			if sp.Connected() {
				nconnected++
			}
		})
		msg.reply <- nconnected

	case getPeersMsg:
		peers := make([]*serverPeer, 0, state.Count())
		state.forAllPeers(func(sp *serverPeer) {
			if !sp.Connected() {
				return
			}
			peers = append(peers, sp)
		})
		msg.reply <- peers

	case connectNodeMsg:
//托多：重复一次？
//限制总对等机的最大数目。
		if state.Count() >= cfg.MaxPeers {
			msg.reply <- errors.New("max peers reached")
			return
		}
		for _, peer := range state.persistentPeers {
			if peer.Addr() == msg.addr {
				if msg.permanent {
					msg.reply <- errors.New("peer already connected")
				} else {
					msg.reply <- errors.New("peer exists as a permanent peer")
				}
				return
			}
		}

		netAddr, err := addrStringToNetAddr(msg.addr)
		if err != nil {
			msg.reply <- err
			return
		}

//托多：如果太多，就用核武器攻击一个不烫发的同伴。
		go s.connManager.Connect(&connmgr.ConnReq{
			Addr:      netAddr,
			Permanent: msg.permanent,
		})
		msg.reply <- nil
	case removeNodeMsg:
		found := disconnectPeer(state.persistentPeers, msg.cmp, func(sp *serverPeer) {
//从删除后保持组计数正常
//名单现在。
			state.outboundGroups[addrmgr.GroupKey(sp.NA())]--
		})

		if found {
			msg.reply <- nil
		} else {
			msg.reply <- errors.New("peer not found")
		}
	case getOutboundGroup:
		count, ok := state.outboundGroups[msg.key]
		if ok {
			msg.reply <- count
		} else {
			msg.reply <- 0
		}
//请求持久（添加）对等点的列表。
	case getAddedNodesMsg:
//用相关同行的一部分做出回应。
		peers := make([]*serverPeer, 0, len(state.persistentPeers))
		for _, sp := range state.persistentPeers {
			peers = append(peers, sp)
		}
		msg.reply <- peers
	case disconnectNodeMsg:
//检查入站对等机。我们通过了零回拨，因为我们没有
//需要对入站对等机的断开连接执行任何其他操作。
		found := disconnectPeer(state.inboundPeers, msg.cmp, nil)
		if found {
			msg.reply <- nil
			return
		}

//检查出站对等机。
		found = disconnectPeer(state.outboundPeers, msg.cmp, func(sp *serverPeer) {
//从删除后保持组计数正常
//名单现在。
			state.outboundGroups[addrmgr.GroupKey(sp.NA())]--
		})
		if found {
//如果有多个出站连接到同一个
//ip:port，继续断开它们直到
//找到对等。
			for found {
				found = disconnectPeer(state.outboundPeers, msg.cmp, func(sp *serverPeer) {
					state.outboundGroups[addrmgr.GroupKey(sp.NA())]--
				})
			}
			msg.reply <- nil
			return
		}

		msg.reply <- errors.New("peer not found")
	}
}

//断开连接对等机尝试断开目标对等机的连接
//已通过对等列表。通过使用传递的
//'comparefunc`，如果传递的对等方是目标，则返回'true'
//同龄人。此函数在成功时返回true，在对等端无法返回时返回false。
//定位。如果找到对等方，并且传递的回调为：“whenfound”
//不是nil，我们在移除它之前用对等方作为参数来调用它。
//从对等列表中，并与服务器断开连接。
func disconnectPeer(peerList map[int32]*serverPeer, compareFunc func(*serverPeer) bool, whenFound func(*serverPeer)) bool {
	for addr, peer := range peerList {
		if compareFunc(peer) {
			if whenFound != nil {
				whenFound(peer)
			}

//没关系，因为我们不会继续
//这样迭代不会破坏循环。
			delete(peerList, addr)
			peer.Disconnect()
			return true
		}
	}
	return false
}

//newpeerconfig返回给定serverpeer的配置。
func newPeerConfig(sp *serverPeer) *peer.Config {
	return &peer.Config{
		Listeners: peer.MessageListeners{
			OnVersion:      sp.OnVersion,
			OnMemPool:      sp.OnMemPool,
			OnTx:           sp.OnTx,
			OnBlock:        sp.OnBlock,
			OnInv:          sp.OnInv,
			OnHeaders:      sp.OnHeaders,
			OnGetData:      sp.OnGetData,
			OnGetBlocks:    sp.OnGetBlocks,
			OnGetHeaders:   sp.OnGetHeaders,
			OnGetCFilters:  sp.OnGetCFilters,
			OnGetCFHeaders: sp.OnGetCFHeaders,
			OnGetCFCheckpt: sp.OnGetCFCheckpt,
			OnFeeFilter:    sp.OnFeeFilter,
			OnFilterAdd:    sp.OnFilterAdd,
			OnFilterClear:  sp.OnFilterClear,
			OnFilterLoad:   sp.OnFilterLoad,
			OnGetAddr:      sp.OnGetAddr,
			OnAddr:         sp.OnAddr,
			OnRead:         sp.OnRead,
			OnWrite:        sp.OnWrite,

//注意：引用客户端当前禁止发送警报的对等机
//没有用它的密钥签名。我们可以用他们的钥匙来验证，但是
//因为样板客户目前不愿意支持
//其他实现的警报消息，我们不会中继它们的。
			OnAlert: nil,
		},
		NewestBlock:       sp.newestBlock,
		HostToNetAddress:  sp.server.addrManager.HostToNetAddress,
		Proxy:             cfg.Proxy,
		UserAgentName:     userAgentName,
		UserAgentVersion:  userAgentVersion,
		UserAgentComments: cfg.UserAgentComments,
		ChainParams:       sp.server.chainParams,
		Services:          sp.server.services,
		DisableRelayTx:    cfg.BlocksOnly,
		ProtocolVersion:   peer.MaxProtocolVersion,
		TrickleInterval:   cfg.TrickleInterval,
	}
}

//当新的入站时，连接管理器将调用inboundpeerconnected。
//已建立连接。它初始化新的入站服务器对等机
//实例，将其与连接关联，并启动Goroutine以等待
//断开连接。
func (s *server) inboundPeerConnected(conn net.Conn) {
	sp := newServerPeer(s, false)
	sp.isWhitelisted = isWhitelisted(conn.RemoteAddr())
	sp.Peer = peer.NewInboundPeer(newPeerConfig(sp))
	sp.AssociateConnection(conn)
	go s.peerDoneHandler(sp)
}

//连接管理器调用OutboundPeerConnected时，
//已建立出站连接。它初始化新的出站服务器
//对等实例，将其与连接等相关状态关联
//请求实例和连接本身，最后通知地址
//尝试的经理。
func (s *server) outboundPeerConnected(c *connmgr.ConnReq, conn net.Conn) {
	sp := newServerPeer(s, c.Permanent)
	p, err := peer.NewOutboundPeer(newPeerConfig(sp), c.Addr.String())
	if err != nil {
		srvrLog.Debugf("Cannot create outbound peer %s: %v", c.Addr, err)
		s.connManager.Disconnect(c.ID())
	}
	sp.Peer = p
	sp.connReq = c
	sp.isWhitelisted = isWhitelisted(conn.RemoteAddr())
	sp.AssociateConnection(conn)
	go s.peerDoneHandler(sp)
	s.addrManager.Attempt(sp.NA())
}

//peerdonehandler通过通知服务器它是
//与其他人一起进行其他必要的清理。
func (s *server) peerDoneHandler(sp *serverPeer) {
	sp.WaitForDisconnect()
	s.donePeers <- sp

//只有告诉同步管理器我们已经离开，如果我们曾经告诉它我们存在。
	if sp.VersionKnown() {
		s.syncManager.DonePeer(sp.Peer)

//逐出对等发送的所有剩余孤儿。
		numEvicted := s.txMemPool.RemoveOrphansByTag(mempool.Tag(sp.ID()))
		if numEvicted > 0 {
			txmpLog.Debugf("Evicted %d %s from peer %v (id %d)",
				numEvicted, pickNoun(numEvicted, "orphan",
					"orphans"), sp, sp.ID())
		}
	}
	close(sp.quit)
}

//PeerHandler用于处理对等操作，如添加和删除
//与服务器进行对等，禁止对等，并将消息广播到
//同龄人。它必须在Goroutine中运行。
func (s *server) peerHandler() {
//启动地址管理器和同步管理器，这两者都是必需的
//同龄人。这是因为它们的生命周期是紧密相连的
//对这个处理程序，而不是添加更多的通道来同步化
//事情，简单地启动和停止它们是容易和稍微快一点的。
//在这个处理程序中。
	s.addrManager.Start()
	s.syncManager.Start()

	srvrLog.Tracef("Starting peer handler")

	state := &peerState{
		inboundPeers:    make(map[int32]*serverPeer),
		persistentPeers: make(map[int32]*serverPeer),
		outboundPeers:   make(map[int32]*serverPeer),
		banned:          make(map[string]time.Time),
		outboundGroups:  make(map[string]int),
	}

	if !cfg.DisableDNSSeed {
//将通过DNS发现的对等端添加到地址管理器。
		connmgr.SeedFromDNS(activeNetParams.Params, defaultRequiredServices,
			btcdLookup, func(addrs []*wire.NetAddress) {
//比特币在这里使用DNS种子器的查找。这个
//很奇怪，因为
//DNS种子查找将变化很大。
//为了复制这种行为，我们将所有地址
//来自第一个。
				s.addrManager.AddAddresses(addrs, addrs[0])
			})
	}
	go s.connManager.Start()

out:
	for {
		select {
//连接到服务器的新对等机。
		case p := <-s.newPeers:
			s.handleAddPeerMsg(state, p)

//断开连接的对等机。
		case p := <-s.donePeers:
			s.handleDonePeerMsg(state, p)

//在主链或孤立中接受块，更新对等高度。
		case umsg := <-s.peerHeightsUpdate:
			s.handleUpdatePeerHeights(state, umsg)

//禁止同行。
		case p := <-s.banPeers:
			s.handleBanPeerMsg(state, p)

//新库存可能会转发给其他同行。
		case invMsg := <-s.relayInv:
			s.handleRelayInvMsg(state, invMsg)

//向所有连接的对等端广播的消息，但那些对等端除外
//被消息排除在外。
		case bmsg := <-s.broadcast:
			s.handleBroadcastMsg(state, &bmsg)

		case qmsg := <-s.query:
			s.handleQuery(state, qmsg)

		case <-s.quit:
//关闭服务器时断开所有对等机的连接。
			state.forAllPeers(func(sp *serverPeer) {
				srvrLog.Tracef("Shutdown peer %s", sp)
				sp.Disconnect()
			})
			break out
		}
	}

	s.connManager.Stop()
	s.syncManager.Stop()
	s.addrManager.Stop()

//在退出前排出通道，这样就不会有任何东西等待。
//发送。
cleanup:
	for {
		select {
		case <-s.newPeers:
		case <-s.donePeers:
		case <-s.peerHeightsUpdate:
		case <-s.relayInv:
		case <-s.broadcast:
		case <-s.query:
		default:
			break cleanup
		}
	}
	s.wg.Done()
	srvrLog.Tracef("Peer handler done")
}

//addpeer添加已连接到服务器的新对等。
func (s *server) AddPeer(sp *serverPeer) {
	s.newPeers <- sp
}

//Banpeer禁止已通过IP连接到服务器的对等机。
func (s *server) BanPeer(sp *serverPeer) {
	s.banPeers <- sp
}

//relayinventory将传递的库存向量传递给所有连接的对等方
//还不知道有没有。
func (s *server) RelayInventory(invVect *wire.InvVect, data interface{}) {
	s.relayInv <- relayMsg{invVect: invVect, data: data}
}

//Broadcastmessage向当前连接到服务器的所有对等端发送消息
//除了那些在通过的同行中排除。
func (s *server) BroadcastMessage(msg wire.Message, exclPeers ...*serverPeer) {
//需要确定这是否是一个已经
//广播，不要再广播。
	bmsg := broadcastMsg{message: msg, excludePeers: exclPeers}
	s.broadcast <- bmsg
}

//ConnectedCount返回当前连接的对等数。
func (s *server) ConnectedCount() int32 {
	replyChan := make(chan int32)

	s.query <- getConnCountMsg{reply: replyChan}

	return <-replyChan
}

//OutboundGroupCount返回连接到给定
//出站组密钥。
func (s *server) OutboundGroupCount(key string) int {
	replyChan := make(chan int)
	s.query <- getOutboundGroup{key: key, reply: replyChan}
	return <-replyChan
}

//addbytessent将传递的字节数添加到发送的总字节数计数器中
//对于服务器。它对于并发访问是安全的。
func (s *server) AddBytesSent(bytesSent uint64) {
	atomic.AddUint64(&s.bytesSent, bytesSent)
}

//AddBytesReceived将传递的字节数与接收的总字节数相加。
//服务器计数器。它对于并发访问是安全的。
func (s *server) AddBytesReceived(bytesReceived uint64) {
	atomic.AddUint64(&s.bytesReceived, bytesReceived)
}

//nettotals返回通过网络接收和发送的所有字节的总和
//所有同龄人。它对于并发访问是安全的。
func (s *server) NetTotals() (uint64, uint64) {
	return atomic.LoadUint64(&s.bytesReceived),
		atomic.LoadUint64(&s.bytesSent)
}

//更新peerheights更新已公布的所有同级的高度
//最新连接的主链块，或已识别的孤立块。这些高度
//更新允许我们动态刷新对等高度，确保同步对等
//选择可以访问每个对等点的最新块高度。
func (s *server) UpdatePeerHeights(latestBlkHash *chainhash.Hash, latestHeight int32, updateSource *peer.Peer) {
	s.peerHeightsUpdate <- updatePeerHeightsMsg{
		newHash:    latestBlkHash,
		newHeight:  latestHeight,
		originPeer: updateSource,
	}
}

//rebroadcasthandler跟踪用户提交的库存
//发出了，但还没有进入一个街区。我们定期重播
//以防我们的同龄人重新启动或以其他方式失去他们的踪迹。
func (s *server) rebroadcastHandler() {
//在第一次Tx重播前等待5分钟。
	timer := time.NewTimer(5 * time.Minute)
	pendingInvs := make(map[wire.InvVect]interface{})

out:
	for {
		select {
		case riv := <-s.modifyRebroadcastInv:
			switch msg := riv.(type) {
//传入的invvect将添加到我们的rpc txs映射中。
			case broadcastInventoryAdd:
				pendingInvs[*msg.invVect] = msg.data

//将invvect添加到块后，我们可以
//如果有，现在把它取下来。
			case broadcastInventoryDel:
				if _, ok := pendingInvs[*msg]; ok {
					delete(pendingInvs, *msg)
				}
			}

		case <-timer.C:
//我们还没有把它做成一个区块的任何存货
//然而。我们定期重新提交它们，直到它们完成。
			for iv, data := range pendingInvs {
				ivCopy := iv
				s.RelayInventory(&ivCopy, data)
			}

//随机处理，最长30分钟（秒）
//未来。
			timer.Reset(time.Second *
				time.Duration(randomUint16Number(1800)))

		case <-s.quit:
			break out
		}
	}

	timer.Stop()

//在退出前排出通道，这样就不会有任何东西等待。
//发送。
cleanup:
	for {
		select {
		case <-s.modifyRebroadcastInv:
		default:
			break cleanup
		}
	}
	s.wg.Done()
}

//Start开始接受来自对等端的连接。
func (s *server) Start() {
//已经开始？
	if atomic.AddInt32(&s.started, 1) != 1 {
		return
	}

	srvrLog.Trace("Starting server")

//服务器启动时间。用于运行时间计算的运行时间命令。
	s.startupTime = time.Now().Unix()

//启动对等处理程序，该处理程序依次启动地址和块
//管理者。
	s.wg.Add(1)
	go s.peerHandler()

	if s.nat != nil {
		s.wg.Add(1)
		go s.upnpUpdateThread()
	}

	if !cfg.DisableRPC {
		s.wg.Add(1)

//启动重新广播处理程序，确保用户Tx接收到
//在包含在块中之前，将重新广播RPC服务器。
		go s.rebroadcastHandler()

		s.rpcServer.Start()
	}

//如果启用生成，则启动CPU矿工。
	if cfg.Generate {
		s.cpuMiner.Start()
	}
}

//停止通过停止并断开所有连接而优雅地关闭服务器
//同龄人和主要听众。
func (s *server) Stop() error {
//确保这只发生一次。
	if atomic.AddInt32(&s.shutdown, 1) != 1 {
		srvrLog.Infof("Server is already in the process of shutting down")
		return nil
	}

	srvrLog.Warnf("Server shutting down")

//必要时停止CPU矿工
	s.cpuMiner.Stop()

//如果未禁用，请关闭RPC服务器。
	if !cfg.DisableRPC {
		s.rpcServer.Stop()
	}

//在数据库中保存费用估算器状态。
	s.db.Update(func(tx database.Tx) error {
		metadata := tx.Metadata()
		metadata.Put(mempool.EstimateFeeDatabaseKey, s.feeEstimator.Save())

		return nil
	})

//向其余Goroutines发出退出信号。
	close(s.quit)
	return nil
}

//WaitForShutdown块，直到主侦听器和对等处理程序停止。
func (s *server) WaitForShutdown() {
	s.wg.Wait()
}

//scheduleShutdown计划在指定的持续时间之后关闭服务器。
//它还动态地调整警告服务器停机的频率
//剩余持续时间。
func (s *server) ScheduleShutdown(duration time.Duration) {
//不要安排多次关机。
	if atomic.AddInt32(&s.shutdownSched, 1) != 1 {
		return
	}
	srvrLog.Warnf("Server shutdown in %v", duration)
	go func() {
		remaining := duration
		tickDuration := dynamicTickDuration(remaining)
		done := time.After(remaining)
		ticker := time.NewTicker(tickDuration)
	out:
		for {
			select {
			case <-done:
				ticker.Stop()
				s.Stop()
				break out
			case <-ticker.C:
				remaining = remaining - tickDuration
				if remaining < time.Second {
					continue
				}

//根据剩余时间动态更改勾选持续时间。
				newDuration := dynamicTickDuration(remaining)
				if tickDuration != newDuration {
					tickDuration = newDuration
					ticker.Stop()
					ticker = time.NewTicker(tickDuration)
				}
				srvrLog.Warnf("Server shutdown in %v", remaining)
			}
		}
	}()
}

//分析侦听器确定每个侦听地址是否为IPv4和IPv6，以及
//返回要用TCP侦听的适当net.addr的切片。它也
//正确检测应用于“所有接口”的地址并添加
//地址为IPv4和IPv6。
func parseListeners(addrs []string) ([]net.Addr, error) {
	netAddrs := make([]net.Addr, 0, len(addrs)*2)
	for _, addr := range addrs {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
//不应该发生，因为已经被规范化了。
			return nil, err
		}

//在PLAN9上的空主机或主机是IPv4和IPv6。
		if host == "" || (host == "*" && runtime.GOOS == "plan9") {
			netAddrs = append(netAddrs, simpleAddr{net: "tcp4", addr: addr})
			netAddrs = append(netAddrs, simpleAddr{net: "tcp6", addr: addr})
			continue
		}

//如果存在，则删除ipv6区域ID，因为net.parseip不存在
//处理它。
		zoneIndex := strings.LastIndex(host, "%")
		if zoneIndex > 0 {
			host = host[:zoneIndex]
		}

//解析IP。
		ip := net.ParseIP(host)
		if ip == nil {
			return nil, fmt.Errorf("'%s' is not a valid IP address", host)
		}

//当IP不是IPv4地址时，to4返回nil，因此使用
//这决定了地址类型。
		if ip.To4() == nil {
			netAddrs = append(netAddrs, simpleAddr{net: "tcp6", addr: addr})
		} else {
			netAddrs = append(netAddrs, simpleAddr{net: "tcp4", addr: addr})
		}
	}
	return netAddrs, nil
}

func (s *server) upnpUpdateThread() {
//立即停止以防止代码重复，然后我们更新
//每15分钟租赁一次。
	timer := time.NewTimer(0 * time.Second)
	lport, _ := strconv.ParseInt(activeNetParams.DefaultPort, 10, 16)
	first := true
out:
	for {
		select {
		case <-timer.C:
//TODO:更巧妙地选择外部端口
//TODO:知道我们在外部网络上监听哪些端口。
//TODO:如果特定的侦听端口不工作，则请求通配符
//侦听端口？
//这假设超时以秒为单位。
			listenPort, err := s.nat.AddPortMapping("tcp", int(lport), int(lport),
				"btcd listen port", 20*60)
			if err != nil {
				srvrLog.Warnf("can't add UPnP port mapping: %v", err)
			}
			if first && err == nil {
//TODO:定期查找此内容以查看UPNP域是否已更改
//IP也是如此。
				externalip, err := s.nat.GetExternalAddress()
				if err != nil {
					srvrLog.Warnf("UPnP can't get external address: %v", err)
					continue out
				}
				na := wire.NewNetAddressIPPort(externalip, uint16(listenPort),
					s.services)
				err = s.addrManager.AddLocalAddress(na, addrmgr.UpnpPrio)
				if err != nil {
//XXX删除端口映射？
				}
				srvrLog.Warnf("Successfully bound via UPnP to %s", addrmgr.NetAddressKey(na))
				first = false
			}
			timer.Reset(time.Minute * 15)
		case <-s.quit:
			break out
		}
	}

	timer.Stop()

	if err := s.nat.DeletePortMapping("tcp", int(lport), int(lport)); err != nil {
		srvrLog.Warnf("unable to remove UPnP port mapping: %v", err)
	} else {
		srvrLog.Debugf("successfully disestablished UPnP port mapping")
	}

	s.wg.Done()
}

//SETUPRPCListeners返回已配置为使用的侦听器切片
//使用RPC服务器取决于侦听的配置设置
//地址和TLS。
func setupRPCListeners() ([]net.Listener, error) {
//如果未禁用，则安装TLS。
	listenFunc := net.Listen
	if !cfg.DisableTLS {
//如果两者都不存在，则生成TLS证书和密钥文件
//存在。
		if !fileExists(cfg.RPCKey) && !fileExists(cfg.RPCCert) {
			err := genCertPair(cfg.RPCCert, cfg.RPCKey)
			if err != nil {
				return nil, err
			}
		}
		keypair, err := tls.LoadX509KeyPair(cfg.RPCCert, cfg.RPCKey)
		if err != nil {
			return nil, err
		}

		tlsConfig := tls.Config{
			Certificates: []tls.Certificate{keypair},
			MinVersion:   tls.VersionTLS12,
		}

//将标准net.listen函数更改为tls函数。
		listenFunc = func(net string, laddr string) (net.Listener, error) {
			return tls.Listen(net, laddr, &tlsConfig)
		}
	}

	netAddrs, err := parseListeners(cfg.RPCListeners)
	if err != nil {
		return nil, err
	}

	listeners := make([]net.Listener, 0, len(netAddrs))
	for _, addr := range netAddrs {
		listener, err := listenFunc(addr.Network(), addr.String())
		if err != nil {
			rpcsLog.Warnf("Can't listen on %s: %v", addr, err)
			continue
		}
		listeners = append(listeners, listener)
	}

	return listeners, nil
}

//new server返回一个新的btcd服务器，该服务器配置为在addr上侦听
//由chainParams指定的比特币网络类型。使用“开始”开始接受
//来自对等方的连接。
func newServer(listenAddrs []string, db database.DB, chainParams *chaincfg.Params, interrupt <-chan struct{}) (*server, error) {
	services := defaultServices
	if cfg.NoPeerBloomFilters {
		services &^= wire.SFNodeBloom
	}
	if cfg.NoCFilters {
		services &^= wire.SFNodeCF
	}

	amgr := addrmgr.New(cfg.DataDir, btcdLookup)

	var listeners []net.Listener
	var nat NAT
	if !cfg.DisableListen {
		var err error
		listeners, nat, err = initListeners(amgr, listenAddrs, services)
		if err != nil {
			return nil, err
		}
		if len(listeners) == 0 {
			return nil, errors.New("no valid listen address")
		}
	}

	s := server{
		chainParams:          chainParams,
		addrManager:          amgr,
		newPeers:             make(chan *serverPeer, cfg.MaxPeers),
		donePeers:            make(chan *serverPeer, cfg.MaxPeers),
		banPeers:             make(chan *serverPeer, cfg.MaxPeers),
		query:                make(chan interface{}),
		relayInv:             make(chan relayMsg, cfg.MaxPeers),
		broadcast:            make(chan broadcastMsg, cfg.MaxPeers),
		quit:                 make(chan struct{}),
		modifyRebroadcastInv: make(chan interface{}),
		peerHeightsUpdate:    make(chan updatePeerHeightsMsg),
		nat:                  nat,
		db:                   db,
		timeSource:           blockchain.NewMedianTime(),
		services:             services,
		sigCache:             txscript.NewSigCache(cfg.SigCacheMaxSize),
		hashCache:            txscript.NewHashCache(cfg.SigCacheMaxSize),
		cfCheckptCaches:      make(map[wire.FilterType][]cfHeaderKV),
	}

//根据需要创建事务和地址索引。
//
//注意：在索引数组中，txindex必须是第一个，因为
//在捕获过程中，addrindex使用来自txindex的数据。如果
//首先运行addrindex，它可能没有来自
//当前块已索引。
	var indexes []indexers.Indexer
	if cfg.TxIndex || cfg.AddrIndex {
//如果启用了地址索引，则启用事务索引，因为它
//需要它。
		if !cfg.TxIndex {
			indxLog.Infof("Transaction index enabled because it " +
				"is required by the address index")
			cfg.TxIndex = true
		} else {
			indxLog.Info("Transaction index is enabled")
		}

		s.txIndex = indexers.NewTxIndex(db)
		indexes = append(indexes, s.txIndex)
	}
	if cfg.AddrIndex {
		indxLog.Info("Address index is enabled")
		s.addrIndex = indexers.NewAddrIndex(db, chainParams)
		indexes = append(indexes, s.addrIndex)
	}
	if !cfg.NoCFilters {
		indxLog.Info("Committed filter index is enabled")
		s.cfIndex = indexers.NewCfIndex(db, chainParams)
		indexes = append(indexes, s.cfIndex)
	}

//如果启用了任何可选索引，则创建索引管理器。
	var indexManager blockchain.IndexManager
	if len(indexes) > 0 {
		indexManager = indexers.NewManager(db, indexes)
	}

//除非禁用，否则将给定的检查点与默认检查点合并。
	var checkpoints []chaincfg.Checkpoint
	if !cfg.DisableCheckpoints {
		checkpoints = mergeCheckpoints(s.chainParams.Checkpoints, cfg.addCheckpoints)
	}

//使用适当的配置创建新的区块链实例。
	var err error
	s.chain, err = blockchain.New(&blockchain.Config{
		DB:           s.db,
		Interrupt:    interrupt,
		ChainParams:  s.chainParams,
		Checkpoints:  checkpoints,
		TimeSource:   s.timeSource,
		SigCache:     s.sigCache,
		IndexManager: indexManager,
		HashCache:    s.hashCache,
	})
	if err != nil {
		return nil, err
	}

//在数据库中搜索feeestimator状态。如果找不到
//或者如果无法加载，则创建一个新的。
	db.Update(func(tx database.Tx) error {
		metadata := tx.Metadata()
		feeEstimationData := metadata.Get(mempool.EstimateFeeDatabaseKey)
		if feeEstimationData != nil {
//从数据库中删除它，这样我们就不会尝试还原
//不知何故又是一样的。
			metadata.Delete(mempool.EstimateFeeDatabaseKey)

//如果有错误，记录下来，做一个新的费用估算器。
			var err error
			s.feeEstimator, err = mempool.RestoreFeeEstimator(feeEstimationData)

			if err != nil {
				peerLog.Errorf("Failed to restore fee estimator %v", err)
			}
		}

		return nil
	})

//如果没有发现feeestimator，或者如果找到了feeestimator
//不知何故落后了，创建一个新的，然后重新开始。
	if s.feeEstimator == nil || s.feeEstimator.LastKnownHeight() != s.chain.BestSnapshot().Height {
		s.feeEstimator = mempool.NewFeeEstimator(
			mempool.DefaultEstimateFeeMaxRollback,
			mempool.DefaultEstimateFeeMinRegisteredBlocks)
	}

	txC := mempool.Config{
		Policy: mempool.Policy{
			DisableRelayPriority: cfg.NoRelayPriority,
			AcceptNonStd:         cfg.RelayNonStd,
			FreeTxRelayLimit:     cfg.FreeTxRelayLimit,
			MaxOrphanTxs:         cfg.MaxOrphanTxs,
			MaxOrphanTxSize:      defaultMaxOrphanTxSize,
			MaxSigOpCostPerTx:    blockchain.MaxBlockSigOpsCost / 4,
			MinRelayTxFee:        cfg.minRelayTxFee,
			MaxTxVersion:         2,
		},
		ChainParams:    chainParams,
		FetchUtxoView:  s.chain.FetchUtxoView,
		BestHeight:     func() int32 { return s.chain.BestSnapshot().Height },
		MedianTimePast: func() time.Time { return s.chain.BestSnapshot().MedianTime },
		CalcSequenceLock: func(tx *btcutil.Tx, view *blockchain.UtxoViewpoint) (*blockchain.SequenceLock, error) {
			return s.chain.CalcSequenceLock(tx, view, true)
		},
		IsDeploymentActive: s.chain.IsDeploymentActive,
		SigCache:           s.sigCache,
		HashCache:          s.hashCache,
		AddrIndex:          s.addrIndex,
		FeeEstimator:       s.feeEstimator,
	}
	s.txMemPool = mempool.New(&txC)

	s.syncManager, err = netsync.New(&netsync.Config{
		PeerNotifier:       &s,
		Chain:              s.chain,
		TxMemPool:          s.txMemPool,
		ChainParams:        s.chainParams,
		DisableCheckpoints: cfg.DisableCheckpoints,
		MaxPeers:           cfg.MaxPeers,
		FeeEstimator:       s.feeEstimator,
	})
	if err != nil {
		return nil, err
	}

//基于创建挖掘策略和块模板生成器
//配置选项。
//
//注意：CPU矿工依赖于mempool，因此mempool必须
//在调用函数以创建CPU矿工之前创建。
	policy := mining.Policy{
		BlockMinWeight:    cfg.BlockMinWeight,
		BlockMaxWeight:    cfg.BlockMaxWeight,
		BlockMinSize:      cfg.BlockMinSize,
		BlockMaxSize:      cfg.BlockMaxSize,
		BlockPrioritySize: cfg.BlockPrioritySize,
		TxMinFreeFee:      cfg.minRelayTxFee,
	}
	blockTemplateGenerator := mining.NewBlkTmplGenerator(&policy,
		s.chainParams, s.txMemPool, s.chain, s.timeSource,
		s.sigCache, s.hashCache)
	s.cpuMiner = cpuminer.New(&cpuminer.Config{
		ChainParams:            chainParams,
		BlockTemplateGenerator: blockTemplateGenerator,
		MiningAddrs:            cfg.miningAddrs,
		ProcessBlock:           s.syncManager.ProcessBlock,
		ConnectedCount:         s.ConnectedCount,
		IsCurrent:              s.syncManager.IsCurrent,
	})

//仅设置一个函数以返回要连接到的新地址
//未在仅连接模式下运行。模拟网络总是
//处于仅连接模式，因为它仅用于连接到
//指定的对等点，并积极避免广告和连接到
//发现同龄人以防止它成为公共测试
//网络。
	var newAddressFunc func() (net.Addr, error)
	if !cfg.SimNet && len(cfg.ConnectPeers) == 0 {
		newAddressFunc = func() (net.Addr, error) {
			for tries := 0; tries < 100; tries++ {
				addr := s.addrManager.GetAddress()
				if addr == nil {
					break
				}

//地址将不会无效、本地或无法路由
//因为addrmanager拒绝添加。
//检查一下我们还没有地址
//在同一组中，以便我们不连接
//以牺牲
//其他。
				key := addrmgr.GroupKey(addr.NetAddress())
				if s.OutboundGroupCount(key) != 0 {
					continue
				}

//失败30后仅允许最近的节点（10分钟）
//时代
				if tries < 30 && time.Since(addr.LastAttempt()) < 10*time.Minute {
					continue
				}

//尝试50次失败后允许非默认端口。
				if tries < 50 && fmt.Sprintf("%d", addr.NetAddress().Port) !=
					activeNetParams.DefaultPort {
					continue
				}

				addrString := addrmgr.NetAddressKey(addr.NetAddress())
				return addrStringToNetAddr(addrString)
			}

			return nil, errors.New("no valid connect address")
		}
	}

//创建连接管理器。
	targetOutbound := defaultTargetOutbound
	if cfg.MaxPeers < targetOutbound {
		targetOutbound = cfg.MaxPeers
	}
	cmgr, err := connmgr.New(&connmgr.Config{
		Listeners:      listeners,
		OnAccept:       s.inboundPeerConnected,
		RetryDuration:  connectionRetryInterval,
		TargetOutbound: uint32(targetOutbound),
		Dial:           btcdDial,
		OnConnection:   s.outboundPeerConnected,
		GetNewAddress:  newAddressFunc,
	})
	if err != nil {
		return nil, err
	}
	s.connManager = cmgr

//启动持久的对等机。
	permanentPeers := cfg.ConnectPeers
	if len(permanentPeers) == 0 {
		permanentPeers = cfg.AddPeers
	}
	for _, addr := range permanentPeers {
		netAddr, err := addrStringToNetAddr(addr)
		if err != nil {
			return nil, err
		}

		go s.connManager.Connect(&connmgr.ConnReq{
			Addr:      netAddr,
			Permanent: true,
		})
	}

	if !cfg.DisableRPC {
//为配置的RPC侦听地址和
//TLS设置。
		rpcListeners, err := setupRPCListeners()
		if err != nil {
			return nil, err
		}
		if len(rpcListeners) == 0 {
			return nil, errors.New("RPCS: No valid listen address")
		}

		s.rpcServer, err = newRPCServer(&rpcserverConfig{
			Listeners:    rpcListeners,
			StartupTime:  s.startupTime,
			ConnMgr:      &rpcConnManager{&s},
			SyncMgr:      &rpcSyncMgr{&s, s.syncManager},
			TimeSource:   s.timeSource,
			Chain:        s.chain,
			ChainParams:  chainParams,
			DB:           db,
			TxMemPool:    s.txMemPool,
			Generator:    blockTemplateGenerator,
			CPUMiner:     s.cpuMiner,
			TxIndex:      s.txIndex,
			AddrIndex:    s.addrIndex,
			CfIndex:      s.cfIndex,
			FeeEstimator: s.feeEstimator,
		})
		if err != nil {
			return nil, err
		}

//当RPC服务器请求关闭进程时发出信号。
		go func() {
			<-s.rpcServer.RequestedProcessShutdown()
			shutdownRequestChannel <- struct{}{}
		}()
	}

	return &s, nil
}

//initListeners初始化配置的网络侦听器并添加任何绑定
//地址管理器的地址。返回侦听器和一个NAT接口，
//如果使用UPNP，则不为零。
func initListeners(amgr *addrmgr.AddrManager, listenAddrs []string, services wire.ServiceFlag) ([]net.Listener, NAT, error) {
//在配置的地址上侦听TCP连接
	netAddrs, err := parseListeners(listenAddrs)
	if err != nil {
		return nil, nil, err
	}

	listeners := make([]net.Listener, 0, len(netAddrs))
	for _, addr := range netAddrs {
		listener, err := net.Listen(addr.Network(), addr.String())
		if err != nil {
			srvrLog.Warnf("Can't listen on %s: %v", addr, err)
			continue
		}
		listeners = append(listeners, listener)
	}

	var nat NAT
	if len(cfg.ExternalIPs) != 0 {
		defaultPort, err := strconv.ParseUint(activeNetParams.DefaultPort, 10, 16)
		if err != nil {
			srvrLog.Errorf("Can not parse default port %s for active chain: %v",
				activeNetParams.DefaultPort, err)
			return nil, nil, err
		}

		for _, sip := range cfg.ExternalIPs {
			eport := uint16(defaultPort)
			host, portstr, err := net.SplitHostPort(sip)
			if err != nil {
//没有端口，使用默认值。
				host = sip
			} else {
				port, err := strconv.ParseUint(portstr, 10, 16)
				if err != nil {
					srvrLog.Warnf("Can not parse port from %s for "+
						"externalip: %v", sip, err)
					continue
				}
				eport = uint16(port)
			}
			na, err := amgr.HostToNetAddress(host, eport, services)
			if err != nil {
				srvrLog.Warnf("Not adding %s as externalip: %v", sip, err)
				continue
			}

			err = amgr.AddLocalAddress(na, addrmgr.ManualPrio)
			if err != nil {
				amgrLog.Warnf("Skipping specified external IP: %v", err)
			}
		}
	} else {
		if cfg.Upnp {
			var err error
			nat, err = Discover()
			if err != nil {
				srvrLog.Warnf("Can't discover upnp: %v", err)
			}
//这里的nil nat很好，只是意味着网络上没有upnp。
		}

//向地址管理器添加绑定地址，以便向对等方公布。
		for _, listener := range listeners {
			addr := listener.Addr().String()
			err := addLocalAddress(amgr, addr, services)
			if err != nil {
				amgrLog.Warnf("Skipping bound address %s: %v", addr, err)
			}
		}
	}

	return listeners, nat, nil
}

//addrstringtonetaddr以“host:port”的形式获取地址并返回
//一个net.addr，它映射到原始地址，并解析任何主机名。
//到IP地址。它还通过返回
//封装地址的net.addr。
func addrStringToNetAddr(addr string) (net.Addr, error) {
	host, strPort, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	port, err := strconv.Atoi(strPort)
	if err != nil {
		return nil, err
	}

//如果主机已经是IP地址，则跳过。
	if ip := net.ParseIP(host); ip != nil {
		return &net.TCPAddr{
			IP:   ip,
			Port: port,
		}, nil
	}

//Tor地址无法解析为IP，所以只需返回一个洋葱
//改为地址。
	if strings.HasSuffix(host, ".onion") {
		if cfg.NoOnion {
			return nil, errors.New("tor has been disabled")
		}

		return &onionAddr{addr: addr}, nil
	}

//尝试查找与已分析主机关联的IP地址。
	ips, err := btcdLookup(host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no addresses found for %s", host)
	}

	return &net.TCPAddr{
		IP:   ips[0],
		Port: port,
	}, nil
}

//addlocaladdress添加此节点正在侦听的地址
//地址管理器，以便它可以中继到对等机。
func addLocalAddress(addrMgr *addrmgr.AddrManager, addr string, services wire.ServiceFlag) error {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return err
	}

	if ip := net.ParseIP(host); ip != nil && ip.IsUnspecified() {
//如果绑定到未指定的地址，则通告所有本地接口
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			return err
		}

		for _, addr := range addrs {
			ifaceIP, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				continue
			}

//如果绑定到0.0.0.0，则不要添加IPv6接口，如果绑定到
//：：，不要添加IPv4接口。
			if (ip.To4() == nil) != (ifaceIP.To4() == nil) {
				continue
			}

			netAddr := wire.NewNetAddressIPPort(ifaceIP, uint16(port), services)
			addrMgr.AddLocalAddress(netAddr, addrmgr.BoundPrio)
		}
	} else {
		netAddr, err := addrMgr.HostToNetAddress(host, uint16(port), services)
		if err != nil {
			return err
		}

		addrMgr.AddLocalAddress(netAddr, addrmgr.BoundPrio)
	}

	return nil
}

//dynamickDuration是一个方便的函数，用于动态选择
//根据剩余时间勾选持续时间。主要用于
//关闭服务器以使关闭警告随着关闭时间的增加而更频繁
//方法。
func dynamicTickDuration(remaining time.Duration) time.Duration {
	switch {
	case remaining <= time.Second*5:
		return time.Second
	case remaining <= time.Second*15:
		return time.Second * 5
	case remaining <= time.Minute:
		return time.Second * 15
	case remaining <= time.Minute*5:
		return time.Minute
	case remaining <= time.Minute*15:
		return time.Minute * 5
	case remaining <= time.Hour:
		return time.Minute * 15
	}
	return time.Hour
}

//IsWhiteList返回IP地址是否包含在白名单中
//网络和IP。
func isWhitelisted(addr net.Addr) bool {
	if len(cfg.whitelists) == 0 {
		return false
	}

	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		srvrLog.Warnf("Unable to SplitHostPort on '%s': %v", addr, err)
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil {
		srvrLog.Warnf("Unable to parse IP '%s'", addr)
		return false
	}

	for _, ipnet := range cfg.whitelists {
		if ipnet.Contains(ip) {
			return true
		}
	}
	return false
}

//checkpointsorter实现sort.interface以允许检查点切片
//分类。
type checkpointSorter []chaincfg.Checkpoint

//len返回切片中的检查点数量。它是
//Sort.Interface实现。
func (s checkpointSorter) Len() int {
	return len(s)
}

//交换在通过的索引处交换检查点。它是
//Sort.Interface实现。
func (s checkpointSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

//less返回索引为i的检查点是否应在
//索引为j的检查点。它是sort.interface实现的一部分。
func (s checkpointSorter) Less(i, j int) bool {
	return s[i].Height < s[j].Height
}

//mergecheckpoints返回合并到一个切片中的两个切片的检查点
//使检查点按高度排序。在这种情况下，附加的
//检查点包含与中的检查点高度相同的检查点
//默认检查点，附加检查点优先，并且
//覆盖默认值。
func mergeCheckpoints(defaultCheckpoints, additional []chaincfg.Checkpoint) []chaincfg.Checkpoint {
//创建附加检查点的映射以在
//离开最近指定的检查点。
	extra := make(map[int32]chaincfg.Checkpoint)
	for _, checkpoint := range additional {
		extra[checkpoint.Height] = checkpoint
	}

//添加在中没有重写的所有默认检查点
//其他检查点。
	numDefault := len(defaultCheckpoints)
	checkpoints := make([]chaincfg.Checkpoint, 0, numDefault+len(extra))
	for _, checkpoint := range defaultCheckpoints {
		if _, exists := extra[checkpoint.Height]; !exists {
			checkpoints = append(checkpoints, checkpoint)
		}
	}

//附加附加检查点并返回排序结果。
	for _, checkpoint := range extra {
		checkpoints = append(checkpoints, checkpoint)
	}
	sort.Sort(checkpointSorter(checkpoints))
	return checkpoints
}
