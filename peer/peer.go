
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2018 BTCSuite开发者
//版权所有（c）2016-2018法令开发商
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package peer

import (
	"bytes"
	"container/list"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/go-socks/socks"
	"github.com/davecgh/go-spew/spew"
)

const (
//Max PrimoLoad是对等体支持的MAX协议版本。
	MaxProtocolVersion = wire.FeeFilterVersion

//DefaultTrickInterval是尝试发送
//向对等体发送消息。
	DefaultTrickleInterval = 10 * time.Second

//MinacAcceptableProtocolVersion是
//连接的对等机可能支持。
	MinAcceptableProtocolVersion = wire.MultipleAddressVersion

//OutputBufferSize是输出通道使用的元素数。
	outputBufferSize = 50

//invTrickleSize is the maximum amount of inventory to send in a single
//向远程对等端滴送清单时的消息。
	maxInvTrickleSize = 1000

//maxknowninventory是在已知的
//库存缓存。
	maxKnownInventory = 1000

//PingInterval是发送Ping之间等待的时间间隔
//信息。
	pingInterval = 2 * time.Minute

//NegotiateTimeout是在超时之前不活动的持续时间
//尚未完成初始版本协商的对等机。
	negotiateTimeout = 30 * time.Second

//idleTimeout是在超时对等机之前的非活动持续时间。
	idleTimeout = 5 * time.Minute

//StallTickInterval是每次检查之间的时间间隔
//停滞不前的同龄人
	stallTickInterval = 15 * time.Second

//stallResponseTimeout is the base maximum amount of time messages that
//期望在断开对等机连接之前等待响应
//失速。最后期限根据回调运行时间和
//only checked on each stall tick interval.
	stallResponseTimeout = 30 * time.Second
)

var (
//NodeCount是自启动之后进行的对等连接总数。
//用于将ID分配给对等机。
	nodeCount int32

//zero hash是零值哈希（全部为零）。它被定义为
//方便。
	zeroHash chainhash.Hash

//Tun-Nokes拥有推送时产生的独特的不安。
//用于检测自我连接的版本消息。
	sentNonces = newMruNonceMap(50)

//allowselfconns仅用于允许测试绕过自身
//连接检测和断开逻辑，因为它们是故意的
//为了测试目的而这样做。
	allowSelfConns bool
)

//MessageListeners定义要用消息调用的回调函数指针
//同龄人的听众。未设置为具体回调的任何侦听器
//在对等机初始化期间被忽略。执行多条消息
//listeners occurs serially, so one callback blocks the execution of the next.
//
//注意：除非另有记录，否则这些侦听器不能直接调用
//自输入后阻止对等实例上的调用（如waitForShutdown）
//在回调完成之前，处理程序goroutine将一直阻塞。这样做会
//导致死锁。
type MessageListeners struct {
//OnGetAddr is invoked when a peer receives a getaddr bitcoin message.
	OnGetAddr func(p *Peer, msg *wire.MsgGetAddr)

//当对等端收到addr比特币消息时调用onaddr。
	OnAddr func(p *Peer, msg *wire.MsgAddr)

//当对等端收到ping比特币消息时，将调用onping。
	OnPing func(p *Peer, msg *wire.MsgPing)

//当对等体收到乒乓比特币消息时，OnPong被调用。
	OnPong func(p *Peer, msg *wire.MsgPong)

//当对等端收到比特币警报消息时，调用OnAlert。
	OnAlert func(p *Peer, msg *wire.MsgAlert)

//当对等端收到mempool比特币消息时，调用onmempool。
	OnMemPool func(p *Peer, msg *wire.MsgMemPool)

//当对等端收到Tx比特币消息时，会调用OnTx。
	OnTx func(p *Peer, msg *wire.MsgTx)

//当对等端接收到块比特币消息时，会调用onblock。
	OnBlock func(p *Peer, msg *wire.MsgBlock, buf []byte)

//当对等端接收到cfilter比特币消息时调用oncfilter。
	OnCFilter func(p *Peer, msg *wire.MsgCFilter)

//当对等体接收CFHeBitter比特币时，调用OnCFHead。
//消息。
	OnCFHeaders func(p *Peer, msg *wire.MsgCFHeaders)

//当对等方收到cfcheckpt比特币时调用oncfcheckpt。
//消息。
	OnCFCheckpt func(p *Peer, msg *wire.MsgCFCheckpt)

//当对等端收到一条INV比特币消息时，会调用ONIV。
	OnInv func(p *Peer, msg *wire.MsgInv)

//当对等端收到头比特币消息时，会调用OnHeaders。
	OnHeaders func(p *Peer, msg *wire.MsgHeaders)

//OnNotFound is invoked when a peer receives a notfound bitcoin
//消息。
	OnNotFound func(p *Peer, msg *wire.MsgNotFound)

//当对等端收到getdata比特币消息时，调用ongetdata。
	OnGetData func(p *Peer, msg *wire.MsgGetData)

//当对等端收到getBlocks比特币时调用ongetBlocks。
//消息。
	OnGetBlocks func(p *Peer, msg *wire.MsgGetBlocks)

//OnGetHeaders是在对等端接收到GetHeaders比特币时调用的。
//消息。
	OnGetHeaders func(p *Peer, msg *wire.MsgGetHeaders)

//OnGetCFilters is invoked when a peer receives a getcfilters bitcoin
//消息。
	OnGetCFilters func(p *Peer, msg *wire.MsgGetCFilters)

//当对等端收到getcfheaders时调用ongetcfheaders
//比特币信息。
	OnGetCFHeaders func(p *Peer, msg *wire.MsgGetCFHeaders)

//当对等端接收到getcfcheckpt时调用ongetcfcheckpt。
//比特币信息。
	OnGetCFCheckpt func(p *Peer, msg *wire.MsgGetCFCheckpt)

//当对等端收到feefilter比特币消息时，调用onfeefilter。
	OnFeeFilter func(p *Peer, msg *wire.MsgFeeFilter)

//OnFilterAdd is invoked when a peer receives a filteradd bitcoin message.
	OnFilterAdd func(p *Peer, msg *wire.MsgFilterAdd)

//当对等方收到filterclear比特币时，调用onfilterclear。
//消息。
	OnFilterClear func(p *Peer, msg *wire.MsgFilterClear)

//OnFilterLoad is invoked when a peer receives a filterload bitcoin
//消息。
	OnFilterLoad func(p *Peer, msg *wire.MsgFilterLoad)

//当对等方接收到MerkleBlock比特币时调用OnMerkleBlock。
//消息。
	OnMerkleBlock func(p *Peer, msg *wire.MsgMerkleBlock)

//当对等端收到版本比特币消息时，调用onversion。
//调用者可以返回拒绝消息，在这种情况下，消息将
//发送到对等端，对等端将断开连接。
	OnVersion func(p *Peer, msg *wire.MsgVersion) *wire.MsgReject

//当对等端收到verack比特币消息时，调用onverack。
	OnVerAck func(p *Peer, msg *wire.MsgVerAck)

//当对等端收到拒绝比特币消息时，调用onreject。
	OnReject func(p *Peer, msg *wire.MsgReject)

//当对等体接收StHealthBitBitcoin时，调用OnsEnthHead。
//消息。
	OnSendHeaders func(p *Peer, msg *wire.MsgSendHeaders)

//当对等端收到比特币消息时，调用OnRead。它
//包括读取的字节数、消息以及是否
//读取出错。通常，呼叫者会选择使用
//但是，特定消息类型的回调可以是
//适用于跟踪服务器范围字节等情况
//counts or working with custom message types for which the peer does
//不直接提供回调。
	OnRead func(p *Peer, bytesRead int, msg wire.Message, err error)

//当我们向对等方写入比特币消息时，会调用onwrite。它
//包括写入的字节数、消息以及
//没有发生写入错误。这对
//跟踪服务器范围的字节计数等情况。
	OnWrite func(p *Peer, bytesWritten int, msg wire.Message, err error)
}

//config是保存对对等机有用的配置选项的结构。
type Config struct {
//newest block指定一个回调，该回调提供最新的块
//根据需要向同行提供详细信息。这可能是零，在这种情况下，
//Peer将报告块高度为0，但这是
//同龄人指定这一点，以便他们当前最为人所知的是准确的
//报道。
	NewestBlock HashFunc

//hostToNetAddress返回给定主机的网络地址。这可以
//在这种情况下，主机将被解析为IP地址。
	HostToNetAddress HostToNetAddrFunc

//代理表示代理正在用于连接。唯一
//效果这是为了防止泄漏TOR代理地址，所以
//仅在使用TOR代理时需要指定。
	Proxy string

//user agent name指定要公布的用户代理名称。它是
//强烈建议指定此值。
	UserAgentName string

//user agent version指定要公布的用户代理版本。它
//强烈建议指定此值，并遵循
//格式“主要、次要、修订”，例如“2.6.41”。
	UserAgentVersion string

//user agent comments指定要公布的用户代理注释。这些
//值不能包含在BIP 14中指定的非法字符：
//“/”，“：”，“（”，“）”。
	UserAgentComments []string

//chainParams标识对等机关联的链参数
//用。强烈建议指定此字段，但它可以
//在这种情况下，将使用测试网络。
	ChainParams *chaincfg.Params

//服务指定由
//本地对等体。此字段可以省略，在这种情况下，它将为0
//因此不宣传支持的服务。
	Services wire.ServiceFlag

//ProtocolVersion指定要使用的最大协议版本和
//广告。在这种情况下，可以省略此字段
//将使用peer.maxprotocolversion。
	ProtocolVersion uint32

//DisableRelayTx指定是否应通知远程对等机
//不为交易发送INV消息。
	DisableRelayTx bool

//侦听器包含要在接收对等端时调用的回调函数
//信息。
	Listeners MessageListeners

//TrickleInterval是指向下滴入
//向同级盘点。
	TrickleInterval time.Duration
}

//minint32是一个帮助函数，返回至少两个uint32。
//这样就避免了数学导入和强制转换为浮点。
func minUint32(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

//NeNETAdvices试图从已通过的IP地址和端口中提取IP地址和端口
//net.addr接口并使用它创建比特币网络地址结构
//信息。
func newNetAddress(addr net.Addr, services wire.ServiceFlag) (*wire.NetAddress, error) {
//不使用代理时，addr将是net.tcpaddr。
	if tcpAddr, ok := addr.(*net.TCPAddr); ok {
		ip := tcpAddr.IP
		port := uint16(tcpAddr.Port)
		na := wire.NewNetAddressIPPort(ip, port, services)
		return na, nil
	}

//使用代理时，addr将是socks.proxiedAddr。
	if proxiedAddr, ok := addr.(*socks.ProxiedAddr); ok {
		ip := net.ParseIP(proxiedAddr.Host)
		if ip == nil {
			ip = net.ParseIP("0.0.0.0")
		}
		port := uint16(proxiedAddr.Port)
		na := wire.NewNetAddressIPPort(ip, port, services)
		return na, nil
	}

//在大多数情况下，addr应该是上述两种情况中的一种，但是
//为了安全起见，返回到尝试从
//地址字符串是最后的手段。
	host, portStr, err := net.SplitHostPort(addr.String())
	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(host)
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return nil, err
	}
	na := wire.NewNetAddressIPPort(ip, uint16(port), services)
	return na, nil
}

//outmsg用于存储要与通道到信号一起发送的消息。
//当邮件已发送（或由于以下原因无法发送）时
//停机）
type outMsg struct {
	msg      wire.Message
	doneChan chan<- struct{}
	encoding wire.MessageEncoding
}

//stallcontrolCmd表示stall控制消息的命令。
type stallControlCmd uint8

//Stall控制消息命令的常量。
const (
//SCCSendMessage表示正在向远程对等端发送消息。
	sccSendMessage stallControlCmd = iota

//SCCReceiveMessage表示已从
//远程对等体。
	sccReceiveMessage

//SCChandlerStart指示将要调用回调处理程序。
	sccHandlerStart

//SCChandlerStart表示回调处理程序已完成。
	sccHandlerDone
)

//stallcontrolmsg用于向stall处理程序发送有关特定事件的信号。
//因此，它可以正确地检测和处理停滞的远程对等点。
type stallControlMsg struct {
	command stallControlCmd
	message wire.Message
}

//StassSNAP是一个时间点上的对等统计数据的快照。
type StatsSnap struct {
	ID             int32
	Addr           string
	Services       wire.ServiceFlag
	LastSend       time.Time
	LastRecv       time.Time
	BytesSent      uint64
	BytesRecv      uint64
	ConnTime       time.Time
	TimeOffset     int64
	Version        uint32
	UserAgent      string
	Inbound        bool
	StartingHeight int32
	LastBlock      int32
	LastPingNonce  uint64
	LastPingTime   time.Time
	LastPingMicros int64
}

//hashfunc是返回块哈希、高度和错误的函数
//它被用作回调以获取最新的块细节。
type HashFunc func() (hash *chainhash.Hash, height int32, err error)

//addrfunc是一个接收地址并返回相关地址的func。
type AddrFunc func(remoteAddr *wire.NetAddress) *wire.NetAddress

//hosttonetaddrfunc是一个func，它接受主机、端口、服务并返回
//网络地址。
type HostToNetAddrFunc func(host string, port uint16,
	services wire.ServiceFlag) (*wire.NetAddress, error)

//注意：一个对等机的整体数据流分为3个goroutine。入站
//消息通过inhandler goroutine读取，通常发送到
//他们自己的管理者。对于与入站数据相关的消息，如块，
//交易和库存，数据由相应的
//消息处理程序。出站消息的数据流分为2个
//Goroutines、QueueHandler和OutHandler。使用第一个队列处理程序
//作为外部实体对消息进行排队的一种方式，通过queuemessage
//函数，无论对等机当前是否正在发送，都可以快速执行。
//它充当外部世界和实际世界之间的交通警察
//写入网络套接字的goroutine。

//对等端为处理比特币提供了一个基本的并发安全比特币对等端。
//通过对等协议进行通信。它提供全双工
//读写，初始握手过程的自动处理，
//查询使用统计信息和有关远程对等机的其他信息，例如
//作为其地址、用户代理和协议版本、输出消息队列，
//inventory trickling, and the ability to dynamically register and unregister
//处理比特币协议消息的回调。
//
//Outbound messages are typically queued via QueueMessage or QueueInventory.
//QueueMessage用于所有消息，包括对以下数据的响应：
//作为块和事务。另一方面，queueinventory只是
//用于中继库存，因为它使用滴流机制进行批处理
//把存货放在一起。但是，一些用于推送消息的助手函数
//通常需要常规特殊处理的特定类型有
//提供方便。
type Peer struct {
//以下变量只能原子地使用。
	bytesReceived uint64
	bytesSent     uint64
	lastRecv      int64
	lastSend      int64
	connected     int32
	disconnect    int32

	conn net.Conn

//这些字段在创建时设置，从不修改，因此
//safe to read from concurrently without a mutex.
	addr    string
	cfg     Config
	inbound bool

flagsMtx             sync.Mutex //保护下面的对等标志
	na                   *wire.NetAddress
	id                   int32
	userAgent            string
	services             wire.ServiceFlag
	versionKnown         bool
advertisedProtoVer   uint32 //远程通告的协议版本
protocolVersion      uint32 //negotiated protocol version
sendHeadersPreferred bool   //对等端发送了一个sendHeaders消息
	verAckReceived       bool
	witnessEnabled       bool

	wireEncoding wire.MessageEncoding

	knownInventory     *mruInventoryMap
	prevGetBlocksMtx   sync.Mutex
	prevGetBlocksBegin *chainhash.Hash
	prevGetBlocksStop  *chainhash.Hash
	prevGetHdrsMtx     sync.Mutex
	prevGetHdrsBegin   *chainhash.Hash
	prevGetHdrsStop    *chainhash.Hash

//这些字段跟踪对等机的统计信息并受到保护
//通过statsmtx互斥。
	statsMtx           sync.RWMutex
	timeOffset         int64
	timeConnected      time.Time
	startingHeight     int32
	lastBlock          int32
	lastAnnouncedBlock *chainhash.Hash
lastPingNonce      uint64    //如果有挂起的ping，则设置为nonce。
lastPingTime       time.Time //上次发送ping的时间。
lastPingMicros     int64     //上次Ping返回的时间。

	stallControl  chan stallControlMsg
	outputQueue   chan outMsg
	sendQueue     chan outMsg
	sendDoneQueue chan struct{}
	outputInvChan chan *wire.InvVect
	inQuit        chan struct{}
	queueQuit     chan struct{}
	outQuit       chan struct{}
	quit          chan struct{}
}

//String returns the peer's address and directionality as a human-readable
//字符串。
//
//此函数对于并发访问是安全的。
func (p *Peer) String() string {
	return fmt.Sprintf("%s (%s)", p.addr, directionString(p.inbound))
}

//updateLastBlockHeight更新对等机的最后一个已知块。
//
//此函数对于并发访问是安全的。
func (p *Peer) UpdateLastBlockHeight(newHeight int32) {
	p.statsMtx.Lock()
	log.Tracef("Updating last block height of peer %v from %v to %v",
		p.addr, p.lastBlock, newHeight)
	p.lastBlock = newHeight
	p.statsMtx.Unlock()
}

//updateLastAnnouncedBlock更新关于最后一个块的元数据散列此
//peer is known to have announced.
//
//此函数对于并发访问是安全的。
func (p *Peer) UpdateLastAnnouncedBlock(blkHash *chainhash.Hash) {
	log.Tracef("Updating last blk for peer %v, %v", p.addr, blkHash)

	p.statsMtx.Lock()
	p.lastAnnouncedBlock = blkHash
	p.statsMtx.Unlock()
}

//addknowninventory将传递的清单添加到已知清单的缓存中
//为同行。
//
//此函数对于并发访问是安全的。
func (p *Peer) AddKnownInventory(invVect *wire.InvVect) {
	p.knownInventory.Add(invVect)
}

//StatsSnapshot返回当前对等标记和统计信息的快照。
//
//此函数对于并发访问是安全的。
func (p *Peer) StatsSnapshot() *StatsSnap {
	p.statsMtx.RLock()

	p.flagsMtx.Lock()
	id := p.id
	addr := p.addr
	userAgent := p.userAgent
	services := p.services
	protocolVersion := p.advertisedProtoVer
	p.flagsMtx.Unlock()

//获取所有相关标志和统计信息的副本。
	statsSnap := &StatsSnap{
		ID:             id,
		Addr:           addr,
		UserAgent:      userAgent,
		Services:       services,
		LastSend:       p.LastSend(),
		LastRecv:       p.LastRecv(),
		BytesSent:      p.BytesSent(),
		BytesRecv:      p.BytesReceived(),
		ConnTime:       p.timeConnected,
		TimeOffset:     p.timeOffset,
		Version:        protocolVersion,
		Inbound:        p.inbound,
		StartingHeight: p.startingHeight,
		LastBlock:      p.lastBlock,
		LastPingNonce:  p.lastPingNonce,
		LastPingMicros: p.lastPingMicros,
		LastPingTime:   p.lastPingTime,
	}

	p.statsMtx.RUnlock()
	return statsSnap
}

//ID返回对等ID。
//
//此函数对于并发访问是安全的。
func (p *Peer) ID() int32 {
	p.flagsMtx.Lock()
	id := p.id
	p.flagsMtx.Unlock()

	return id
}

//NA返回对等网络地址。
//
//此函数对于并发访问是安全的。
func (p *Peer) NA() *wire.NetAddress {
	p.flagsMtx.Lock()
	na := p.na
	p.flagsMtx.Unlock()

	return na
}

//addr返回对等地址。
//
//此函数对于并发访问是安全的。
func (p *Peer) Addr() string {
//初始化后地址不变，因此
//由互斥保护。
	return p.addr
}

//Inbound返回对等机是否入站。
//
//此函数对于并发访问是安全的。
func (p *Peer) Inbound() bool {
	return p.inbound
}

//服务返回远程对等机的服务标志。
//
//此函数对于并发访问是安全的。
func (p *Peer) Services() wire.ServiceFlag {
	p.flagsMtx.Lock()
	services := p.services
	p.flagsMtx.Unlock()

	return services
}

//user agent返回远程对等机的用户代理。
//
//此函数对于并发访问是安全的。
func (p *Peer) UserAgent() string {
	p.flagsMtx.Lock()
	userAgent := p.userAgent
	p.flagsMtx.Unlock()

	return userAgent
}

//last announced block返回远程对等机的最后一个已公告块。
//
//此函数对于并发访问是安全的。
func (p *Peer) LastAnnouncedBlock() *chainhash.Hash {
	p.statsMtx.RLock()
	lastAnnouncedBlock := p.lastAnnouncedBlock
	p.statsMtx.RUnlock()

	return lastAnnouncedBlock
}

//lastpingnoce返回远程对等机的最后一个ping nonce。
//
//此函数对于并发访问是安全的。
func (p *Peer) LastPingNonce() uint64 {
	p.statsMtx.RLock()
	lastPingNonce := p.lastPingNonce
	p.statsMtx.RUnlock()

	return lastPingNonce
}

//last ping time返回远程对等机的最后一次Ping时间。
//
//此函数对于并发访问是安全的。
func (p *Peer) LastPingTime() time.Time {
	p.statsMtx.RLock()
	lastPingTime := p.lastPingTime
	p.statsMtx.RUnlock()

	return lastPingTime
}

//last ping micros返回远程对等机的最后一个ping micros。
//
//此函数对于并发访问是安全的。
func (p *Peer) LastPingMicros() int64 {
	p.statsMtx.RLock()
	lastPingMicros := p.lastPingMicros
	p.statsMtx.RUnlock()

	return lastPingMicros
}

//version known返回对等机的版本是否已知
//局部地。
//
//此函数对于并发访问是安全的。
func (p *Peer) VersionKnown() bool {
	p.flagsMtx.Lock()
	versionKnown := p.versionKnown
	p.flagsMtx.Unlock()

	return versionKnown
}

//verack received返回verack消息是否由
//同龄人。
//
//此函数对于并发访问是安全的。
func (p *Peer) VerAckReceived() bool {
	p.flagsMtx.Lock()
	verAckReceived := p.verAckReceived
	p.flagsMtx.Unlock()

	return verAckReceived
}

//ProtocolVersion返回协商的对等协议版本。
//
//此函数对于并发访问是安全的。
func (p *Peer) ProtocolVersion() uint32 {
	p.flagsMtx.Lock()
	protocolVersion := p.protocolVersion
	p.flagsMtx.Unlock()

	return protocolVersion
}

//last block返回对等机的最后一个块。
//
//此函数对于并发访问是安全的。
func (p *Peer) LastBlock() int32 {
	p.statsMtx.RLock()
	lastBlock := p.lastBlock
	p.statsMtx.RUnlock()

	return lastBlock
}

//last send返回对等端的最后一次发送时间。
//
//此函数对于并发访问是安全的。
func (p *Peer) LastSend() time.Time {
	return time.Unix(atomic.LoadInt64(&p.lastSend), 0)
}

//last recv返回对等机的最后一次接收时间。
//
//此函数对于并发访问是安全的。
func (p *Peer) LastRecv() time.Time {
	return time.Unix(atomic.LoadInt64(&p.lastRecv), 0)
}

//localaddr返回连接的本地地址。
//
//此函数对于并发访问是安全的。
func (p *Peer) LocalAddr() net.Addr {
	var localAddr net.Addr
	if atomic.LoadInt32(&p.connected) != 0 {
		localAddr = p.conn.LocalAddr()
	}
	return localAddr
}

//bytes sent返回对等机发送的字节总数。
//
//此函数对于并发访问是安全的。
func (p *Peer) BytesSent() uint64 {
	return atomic.LoadUint64(&p.bytesSent)
}

//BytesReceived返回对等机接收的字节总数。
//
//此函数对于并发访问是安全的。
func (p *Peer) BytesReceived() uint64 {
	return atomic.LoadUint64(&p.bytesReceived)
}

//TimeConnected返回对等机连接的时间。
//
//此函数对于并发访问是安全的。
func (p *Peer) TimeConnected() time.Time {
	p.statsMtx.RLock()
	timeConnected := p.timeConnected
	p.statsMtx.RUnlock()

	return timeConnected
}

//TimeOffset返回本地时间从
//对等方在初始协商阶段报告的时间。负值
//表示远程对等机的时间早于本地时间。
//
//此函数对于并发访问是安全的。
func (p *Peer) TimeOffset() int64 {
	p.statsMtx.RLock()
	timeOffset := p.timeOffset
	p.statsMtx.RUnlock()

	return timeOffset
}

//SaldSHIFT高度返回对等体在报告期间的最后已知高度。
//初步谈判阶段。
//
//此函数对于并发访问是安全的。
func (p *Peer) StartingHeight() int32 {
	p.statsMtx.RLock()
	startingHeight := p.startingHeight
	p.statsMtx.RUnlock()

	return startingHeight
}

//如果对等端需要头消息而不是
//块的库存向量。
//
//此函数对于并发访问是安全的。
func (p *Peer) WantsHeaders() bool {
	p.flagsMtx.Lock()
	sendHeadersPreferred := p.sendHeadersPreferred
	p.flagsMtx.Unlock()

	return sendHeadersPreferred
}

//如果对等机发出支持的信号，则IsWitnessEnabled返回true
//隔离证人。
//
//此函数对于并发访问是安全的。
func (p *Peer) IsWitnessEnabled() bool {
	p.flagsMtx.Lock()
	witnessEnabled := p.witnessEnabled
	p.flagsMtx.Unlock()

	return witnessEnabled
}

//pushaddrmsg使用提供的
//地址。此功能对于通过
//队列消息，因为它自动将地址限制为最大值
//消息允许的数字，并随机化所选地址
//太多了。它返回实际发送的地址，不返回
//如果提供的地址切片中没有条目，则将发送消息。
//
//此函数对于并发访问是安全的。
func (p *Peer) PushAddrMsg(addresses []*wire.NetAddress) ([]*wire.NetAddress, error) {
	addressCount := len(addresses)

//没有要发送的内容。
	if addressCount == 0 {
		return nil, nil
	}

	msg := wire.NewMsgAddr()
	msg.AddrList = make([]*wire.NetAddress, addressCount)
	copy(msg.AddrList, addresses)

//Randomize the addresses sent if there are more than the maximum allowed.
	if addressCount > wire.MaxAddrPerMsg {
//无序播放地址列表。
		for i := 0; i < wire.MaxAddrPerMsg; i++ {
			j := i + rand.Intn(addressCount-i)
			msg.AddrList[i], msg.AddrList[j] = msg.AddrList[j], msg.AddrList[i]
		}

//将其截断到最大大小。
		msg.AddrList = msg.AddrList[:wire.MaxAddrPerMsg]
	}

	p.QueueMessage(msg, nil)
	return msg.AddrList, nil
}

//pushgetblocksmsg为提供的块定位器发送getblocks消息
//停止散列。它将忽略背靠背的重复请求。
//
//此函数对于并发访问是安全的。
func (p *Peer) PushGetBlocksMsg(locator blockchain.BlockLocator, stopHash *chainhash.Hash) error {
//从块定位器提取begin散列，如果指定了一个，
//用于筛选重复的GetBlocks请求。
	var beginHash *chainhash.Hash
	if len(locator) > 0 {
		beginHash = locator[0]
	}

//Filter duplicate getblocks requests.
	p.prevGetBlocksMtx.Lock()
	isDuplicate := p.prevGetBlocksStop != nil && p.prevGetBlocksBegin != nil &&
		beginHash != nil && stopHash.IsEqual(p.prevGetBlocksStop) &&
		beginHash.IsEqual(p.prevGetBlocksBegin)
	p.prevGetBlocksMtx.Unlock()

	if isDuplicate {
		log.Tracef("Filtering duplicate [getblocks] with begin "+
			"hash %v, stop hash %v", beginHash, stopHash)
		return nil
	}

//构造getBlocks请求并将其排队发送。
	msg := wire.NewMsgGetBlocks(stopHash)
	for _, hash := range locator {
		err := msg.AddBlockLocatorHash(hash)
		if err != nil {
			return err
		}
	}
	p.QueueMessage(msg, nil)

//更新以前的GetBlocks请求信息以进行筛选
//复制品。
	p.prevGetBlocksMtx.Lock()
	p.prevGetBlocksBegin = beginHash
	p.prevGetBlocksStop = stopHash
	p.prevGetBlocksMtx.Unlock()
	return nil
}

//pushgetheadermsg为提供的块定位器发送getblocks消息
//停止散列。它将忽略背靠背的重复请求。
//
//此函数对于并发访问是安全的。
func (p *Peer) PushGetHeadersMsg(locator blockchain.BlockLocator, stopHash *chainhash.Hash) error {
//从块定位器提取begin散列，如果指定了一个，
//用于筛选重复的GetHeaders请求。
	var beginHash *chainhash.Hash
	if len(locator) > 0 {
		beginHash = locator[0]
	}

//Filter duplicate getheaders requests.
	p.prevGetHdrsMtx.Lock()
	isDuplicate := p.prevGetHdrsStop != nil && p.prevGetHdrsBegin != nil &&
		beginHash != nil && stopHash.IsEqual(p.prevGetHdrsStop) &&
		beginHash.IsEqual(p.prevGetHdrsBegin)
	p.prevGetHdrsMtx.Unlock()

	if isDuplicate {
		log.Tracef("Filtering duplicate [getheaders] with begin hash %v",
			beginHash)
		return nil
	}

//构造getheaders请求并将其排队发送。
	msg := wire.NewMsgGetHeaders()
	msg.HashStop = *stopHash
	for _, hash := range locator {
		err := msg.AddBlockLocatorHash(hash)
		if err != nil {
			return err
		}
	}
	p.QueueMessage(msg, nil)

//更新以前的GetHeaders请求信息以进行筛选
//复制品。
	p.prevGetHdrsMtx.Lock()
	p.prevGetHdrsBegin = beginHash
	p.prevGetHdrsStop = stopHash
	p.prevGetHdrsMtx.Unlock()
	return nil
}

//pushrejectmsg为提供的命令发送拒绝消息，拒绝代码，
//拒绝原因和散列。The hash will only be used when the command is a tx
//或阻塞，在其他情况下应为零。wait参数将导致
//函数来阻止，直到实际发送拒绝消息为止。
//
//此函数对于并发访问是安全的。
func (p *Peer) PushRejectMsg(command string, code wire.RejectCode, reason string, hash *chainhash.Hash, wait bool) {
//如果协议版本为
//太低了。
	if p.VersionKnown() && p.ProtocolVersion() < wire.RejectVersion {
		return
	}

	msg := wire.NewMsgReject(command, code, reason)
	if command == wire.CmdTx || command == wire.CmdBlock {
		if hash == nil {
			log.Warnf("Sending a reject message for command "+
				"type %v which should have specified a hash "+
				"but does not", command)
			hash = &zeroHash
		}
		msg.Hash = *hash
	}

//如果呼叫者没有请求，则无需等待即可发送消息。
	if !wait {
		p.QueueMessage(msg, nil)
		return
	}

//发送消息并阻止，直到它在返回之前被发送。
	doneChan := make(chan struct{}, 1)
	p.QueueMessage(msg, doneChan)
	<-doneChan
}

//当对等端收到ping比特币消息时，将调用handlepingmsg。为了
//最近的客户端（协议版本>bip0031版本），它用pong回复
//消息。对于老客户，它除了失败什么都不做。
//被认为是成功的ping。
func (p *Peer) handlePingMsg(msg *wire.MsgPing) {
//只有当消息来自一个足够新的客户机时，才用pong回复。
	if p.ProtocolVersion() > wire.BIP0031Version {
//包含来自乒乓球的nonce，以便识别乒乓球。
		p.QueueMessage(wire.NewMsgPong(msg.Nonce), nil)
	}
}

//当对等端收到pong比特币消息时，将调用handlepongmsg。它
//根据需要更新最近客户端的ping统计信息（协议
//版本>Bip0031版本）。对于老客户或
//以前未发送ping。
func (p *Peer) handlePongMsg(msg *wire.MsgPong) {
//我们可以在这里使用缓冲通道发送数据。
//当我们发送一个ping或一个跟踪
//每平的时间。For now we just make a best effort and
//仅记录上次发送的ping的统计信息。任何前面
//and overlapping pings will be ignored. It is unlikely to occur
//没有大量使用ping-rpc调用，因为我们很少ping
//如果它们重叠的话，我们就已经超时了。
	if p.ProtocolVersion() > wire.BIP0031Version {
		p.statsMtx.Lock()
		if p.lastPingNonce != 0 && msg.Nonce == p.lastPingNonce {
			p.lastPingMicros = time.Since(p.lastPingTime).Nanoseconds()
p.lastPingMicros /= 1000 //转换为usec。
			p.lastPingNonce = 0
		}
		p.statsMtx.Unlock()
	}
}

//readmessage通过日志记录从对等端读取下一个比特币消息。
func (p *Peer) readMessage(encoding wire.MessageEncoding) (wire.Message, []byte, error) {
	n, msg, buf, err := wire.ReadMessageWithEncodingN(p.conn,
		p.ProtocolVersion(), p.cfg.ChainParams.Net, encoding)
	atomic.AddUint64(&p.bytesReceived, uint64(n))
	if p.cfg.Listeners.OnRead != nil {
		p.cfg.Listeners.OnRead(p, n, msg, err)
	}
	if err != nil {
		return nil, nil, err
	}

//使用闭包记录昂贵的操作，因此它们仅在
//the logging level requires it.
	log.Debugf("%v", newLogClosure(func() string {
//调试消息摘要。
		summary := messageSummary(msg)
		if len(summary) > 0 {
			summary = " (" + summary + ")"
		}
		return fmt.Sprintf("Received %v%s from %s",
			msg.Command(), summary, p)
	}))
	log.Tracef("%v", newLogClosure(func() string {
		return spew.Sdump(msg)
	}))
	log.Tracef("%v", newLogClosure(func() string {
		return spew.Sdump(buf)
	}))

	return msg, buf, nil
}

//WRITEMESSAGE通过日志向对等端发送比特币消息。
func (p *Peer) writeMessage(msg wire.Message, enc wire.MessageEncoding) error {
//如果我们断开连接，不要做任何事情。
	if atomic.LoadInt32(&p.disconnect) != 0 {
		return nil
	}

//使用闭包记录昂贵的操作，因此它们仅在
//日志级别需要它。
	log.Debugf("%v", newLogClosure(func() string {
//调试消息摘要。
		summary := messageSummary(msg)
		if len(summary) > 0 {
			summary = " (" + summary + ")"
		}
		return fmt.Sprintf("Sending %v%s to %s", msg.Command(),
			summary, p)
	}))
	log.Tracef("%v", newLogClosure(func() string {
		return spew.Sdump(msg)
	}))
	log.Tracef("%v", newLogClosure(func() string {
		var buf bytes.Buffer
		_, err := wire.WriteMessageWithEncodingN(&buf, msg, p.ProtocolVersion(),
			p.cfg.ChainParams.Net, enc)
		if err != nil {
			return err.Error()
		}
		return spew.Sdump(buf.Bytes())
	}))

//将消息写入对等端。
	n, err := wire.WriteMessageWithEncodingN(p.conn, msg,
		p.ProtocolVersion(), p.cfg.ChainParams.Net, enc)
	atomic.AddUint64(&p.bytesSent, uint64(n))
	if p.cfg.Listeners.OnWrite != nil {
		p.cfg.Listeners.OnWrite(p, n, msg, err)
	}
	return err
}

//IsallowedReadError返回是否允许传递的错误
//断开对等机的连接。尤其是，需要允许回归测试
//发送格式错误的消息而不断开对等端的连接。
func (p *Peer) isAllowedReadError(err error) bool {
//仅允许在回归测试模式下出现读取错误。
	if p.cfg.ChainParams.Net != wire.TestNet {
		return false
	}

//如果不是特定的错误消息错误，请不要允许错误。
	if _, ok := err.(*wire.MessageError); !ok {
		return false
	}

//如果错误不是来自本地主机或
//由于某种原因无法确定主机名。
	host, _, err := net.SplitHostPort(p.addr)
	if err != nil {
		return false
	}

	if host != "127.0.0.1" && host != "localhost" {
		return false
	}

//如果所有支票都通过，则允许。
	return true
}

//shouldHandleReadError返回传递的错误，即
//预期来自于从inhandler中的远程对等机读取，
//应记录并以拒绝消息响应。
func (p *Peer) shouldHandleReadError(err error) bool {
//在强制对等端时不记录或拒绝消息
//断开的。
	if atomic.LoadInt32(&p.disconnect) != 0 {
		return false
	}

//当远程对等机
//断开的。
	if err == io.EOF {
		return false
	}
	if opErr, ok := err.(*net.OpError); ok && !opErr.Temporary() {
		return false
	}

	return true
}

//MaybeaddDeadline可能会为适当的预期
//对传递到挂起响应映射的有线协议命令的响应。
func (p *Peer) maybeAddDeadline(pendingResponses map[string]time.Time, msgCmd string) {
//为需要响应的每封正在发送的邮件设置截止时间。
//
//注意：这里故意忽略ping，因为它们通常是
//异步发送，由于消息的长时间后锁，
//例如，在初始块下载的情况下，
//将无法及时收到响应。
	deadline := time.Now().Add(stallResponseTimeout)
	switch msgCmd {
	case wire.CmdVersion:
//需要verack消息。
		pendingResponses[wire.CmdVerAck] = deadline

	case wire.CmdMemPool:
//Expects an inv message.
		pendingResponses[wire.CmdInv] = deadline

	case wire.CmdGetBlocks:
//需要INV消息。
		pendingResponses[wire.CmdInv] = deadline

	case wire.CmdGetData:
//需要block、merkleblock、tx或notfound消息。
		pendingResponses[wire.CmdBlock] = deadline
		pendingResponses[wire.CmdMerkleBlock] = deadline
		pendingResponses[wire.CmdTx] = deadline
		pendingResponses[wire.CmdNotFound] = deadline

	case wire.CmdGetHeaders:
//Expects a headers message.  Use a longer deadline since it
//远程对等机可能需要一段时间来加载所有
//标题。
		deadline = time.Now().Add(stallResponseTimeout * 3)
		pendingResponses[wire.CmdHeaders] = deadline
	}
}

//StallHandler处理对等机的失速检测。这需要保持
//跟踪预期的响应并在考虑
//回拨所花的时间。它必须像野人一样运作。
func (p *Peer) stallHandler() {
//这些变量用于调整
//执行回调所需的时间。这是因为新的
//在前一封邮件完成处理之前，不会读取邮件
//（包括回拨），所以收到回复的最后期限
//对于给定的消息，也必须考虑处理时间。
	var handlerActive bool
	var handlersStartTime time.Time
	var deadlineOffset time.Duration

//PendingResponses跟踪预期的响应截止时间。
	pendingResponses := make(map[string]time.Time)

//stallticker用于定期检查
//exceeded the expected deadline and disconnect the peer due to
//失速。
	stallTicker := time.NewTicker(stallTickInterval)
	defer stallTicker.Stop()

//iostopped用于检测输入和输出处理程序
//Goroutines完成了。
	var ioStopped bool
out:
	for {
		select {
		case msg := <-p.stallControl:
			switch msg.command {
			case sccSendMessage:
//为预期响应添加截止时间
//如果需要，请发送消息。
				p.maybeAddDeadline(pendingResponses,
					msg.message.Command())

			case sccReceiveMessage:
//从预期的
//响应图。因为某些命令期望
//一组响应之一，删除
//相应地，预期组中的所有内容。
				switch msgCmd := msg.message.Command(); msgCmd {
				case wire.CmdBlock:
					fallthrough
				case wire.CmdMerkleBlock:
					fallthrough
				case wire.CmdTx:
					fallthrough
				case wire.CmdNotFound:
					delete(pendingResponses, wire.CmdBlock)
					delete(pendingResponses, wire.CmdMerkleBlock)
					delete(pendingResponses, wire.CmdTx)
					delete(pendingResponses, wire.CmdNotFound)

				default:
					delete(pendingResponses, msgCmd)
				}

			case sccHandlerStart:
//对不平衡回拨信号发出警告。
				if handlerActive {
					log.Warn("Received handler start " +
						"control command while a " +
						"handler is already active")
					continue
				}

				handlerActive = true
				handlersStartTime = time.Now()

			case sccHandlerDone:
//对不平衡回拨信号发出警告。
				if !handlerActive {
					log.Warn("Received handler done " +
						"control command when a " +
						"handler is not already active")
					continue
				}

//Extend active deadlines by the time it took
//执行回调。
				duration := time.Since(handlersStartTime)
				deadlineOffset += duration
				handlerActive = false

			default:
				log.Warnf("Unsupported message command %v",
					msg.command)
			}

		case <-stallTicker.C:
//计算应用于截止日期的偏移量
//关于处理程序执行以来的时间
//最后一滴答声。
			now := time.Now()
			offset := deadlineOffset
			if handlerActive {
				offset += now.Sub(handlersStartTime)
			}

//如果有任何挂起的响应，请断开对等机的连接
//不要在调整后的最后期限前到达。
			for command, deadline := range pendingResponses {
				if now.Before(deadline.Add(offset)) {
					continue
				}

				log.Debugf("Peer %s appears to be stalled or "+
					"misbehaving, %s timeout -- "+
					"disconnecting", p, command)
				p.Disconnect()
				break
			}

//重置下一个勾选的截止日期偏移量。
			deadlineOffset = 0

		case <-p.inQuit:
//一旦输入和
//输出处理程序goroutines已完成。
			if ioStopped {
				break out
			}
			ioStopped = true

		case <-p.outQuit:
//一旦输入和
//输出处理程序goroutines已完成。
			if ioStopped {
				break out
			}
			ioStopped = true
		}
	}

//在离开之前排干所有的等待通道，这样就什么都没有了。
//等着这鬼东西。
cleanup:
	for {
		select {
		case <-p.stallControl:
		default:
			break cleanup
		}
	}
	log.Tracef("Peer stall handler done for %s", p)
}

//inhandler处理对等端的所有传入消息。它必须作为
//高尔图
func (p *Peer) inHandler() {
//当接收到一条新消息时，计时器停止，然后重置。
//进行处理。
	idleTimer := time.AfterFunc(idleTimeout, func() {
		log.Warnf("Peer %s no answer for %s -- disconnecting", p, idleTimeout)
		p.Disconnect()
	})

out:
	for atomic.LoadInt32(&p.disconnect) == 0 {
//读取一条消息，并在读取后立即停止空闲计时器
//完成了。如果
//需要。
		rmsg, buf, err := p.readMessage(p.wireEncoding)
		idleTimer.Stop()
		if err != nil {
//为了允许对格式错误的消息进行回归测试，请不要
//当我们处于回归测试模式时，断开对等端的连接，
//错误是允许的错误之一。
			if p.isAllowedReadError(err) {
				log.Errorf("Allowed test error from %s: %v", p, err)
				idleTimer.Reset(idleTimeout)
				continue
			}

//仅记录错误并发送拒绝消息，如果
//本地对等机没有强制断开连接，并且
//远程对等机未断开连接。
			if p.shouldHandleReadError(err) {
				errMsg := fmt.Sprintf("Can't read message from %s: %v", p, err)
				if err != io.ErrUnexpectedEOF {
					log.Errorf(errMsg)
				}

//为格式错误的邮件推送拒绝邮件并等待
//断开连接前要发送的消息。
//
//注意：理想情况下，如果
//至少有那么多信息是有效的，但那不是
//目前被电线暴露，所以刚用畸形的
//命令。
				p.PushRejectMsg("malformed", wire.RejectMalformed, errMsg, nil,
					true)
			}
			break out
		}
		atomic.StoreInt64(&p.lastRecv, time.Now().Unix())
		p.stallControl <- stallControlMsg{sccReceiveMessage, rmsg}

//处理每个支持的消息类型。
		p.stallControl <- stallControlMsg{sccHandlerStart, rmsg}
		switch msg := rmsg.(type) {
		case *wire.MsgVersion:
//每个对等端只能有一个版本消息。
			p.PushRejectMsg(msg.Command(), wire.RejectDuplicate,
				"duplicate version message", nil, true)
			break out

		case *wire.MsgVerAck:

//不需要读锁，因为未写入verackreceived
//去任何其他的戈罗廷。
			if p.verAckReceived {
				log.Infof("Already received 'verack' from peer %v -- "+
					"disconnecting", p)
				break out
			}
			p.flagsMtx.Lock()
			p.verAckReceived = true
			p.flagsMtx.Unlock()
			if p.cfg.Listeners.OnVerAck != nil {
				p.cfg.Listeners.OnVerAck(p, msg)
			}

		case *wire.MsgGetAddr:
			if p.cfg.Listeners.OnGetAddr != nil {
				p.cfg.Listeners.OnGetAddr(p, msg)
			}

		case *wire.MsgAddr:
			if p.cfg.Listeners.OnAddr != nil {
				p.cfg.Listeners.OnAddr(p, msg)
			}

		case *wire.MsgPing:
			p.handlePingMsg(msg)
			if p.cfg.Listeners.OnPing != nil {
				p.cfg.Listeners.OnPing(p, msg)
			}

		case *wire.MsgPong:
			p.handlePongMsg(msg)
			if p.cfg.Listeners.OnPong != nil {
				p.cfg.Listeners.OnPong(p, msg)
			}

		case *wire.MsgAlert:
			if p.cfg.Listeners.OnAlert != nil {
				p.cfg.Listeners.OnAlert(p, msg)
			}

		case *wire.MsgMemPool:
			if p.cfg.Listeners.OnMemPool != nil {
				p.cfg.Listeners.OnMemPool(p, msg)
			}

		case *wire.MsgTx:
			if p.cfg.Listeners.OnTx != nil {
				p.cfg.Listeners.OnTx(p, msg)
			}

		case *wire.MsgBlock:
			if p.cfg.Listeners.OnBlock != nil {
				p.cfg.Listeners.OnBlock(p, msg, buf)
			}

		case *wire.MsgInv:
			if p.cfg.Listeners.OnInv != nil {
				p.cfg.Listeners.OnInv(p, msg)
			}

		case *wire.MsgHeaders:
			if p.cfg.Listeners.OnHeaders != nil {
				p.cfg.Listeners.OnHeaders(p, msg)
			}

		case *wire.MsgNotFound:
			if p.cfg.Listeners.OnNotFound != nil {
				p.cfg.Listeners.OnNotFound(p, msg)
			}

		case *wire.MsgGetData:
			if p.cfg.Listeners.OnGetData != nil {
				p.cfg.Listeners.OnGetData(p, msg)
			}

		case *wire.MsgGetBlocks:
			if p.cfg.Listeners.OnGetBlocks != nil {
				p.cfg.Listeners.OnGetBlocks(p, msg)
			}

		case *wire.MsgGetHeaders:
			if p.cfg.Listeners.OnGetHeaders != nil {
				p.cfg.Listeners.OnGetHeaders(p, msg)
			}

		case *wire.MsgGetCFilters:
			if p.cfg.Listeners.OnGetCFilters != nil {
				p.cfg.Listeners.OnGetCFilters(p, msg)
			}

		case *wire.MsgGetCFHeaders:
			if p.cfg.Listeners.OnGetCFHeaders != nil {
				p.cfg.Listeners.OnGetCFHeaders(p, msg)
			}

		case *wire.MsgGetCFCheckpt:
			if p.cfg.Listeners.OnGetCFCheckpt != nil {
				p.cfg.Listeners.OnGetCFCheckpt(p, msg)
			}

		case *wire.MsgCFilter:
			if p.cfg.Listeners.OnCFilter != nil {
				p.cfg.Listeners.OnCFilter(p, msg)
			}

		case *wire.MsgCFHeaders:
			if p.cfg.Listeners.OnCFHeaders != nil {
				p.cfg.Listeners.OnCFHeaders(p, msg)
			}

		case *wire.MsgFeeFilter:
			if p.cfg.Listeners.OnFeeFilter != nil {
				p.cfg.Listeners.OnFeeFilter(p, msg)
			}

		case *wire.MsgFilterAdd:
			if p.cfg.Listeners.OnFilterAdd != nil {
				p.cfg.Listeners.OnFilterAdd(p, msg)
			}

		case *wire.MsgFilterClear:
			if p.cfg.Listeners.OnFilterClear != nil {
				p.cfg.Listeners.OnFilterClear(p, msg)
			}

		case *wire.MsgFilterLoad:
			if p.cfg.Listeners.OnFilterLoad != nil {
				p.cfg.Listeners.OnFilterLoad(p, msg)
			}

		case *wire.MsgMerkleBlock:
			if p.cfg.Listeners.OnMerkleBlock != nil {
				p.cfg.Listeners.OnMerkleBlock(p, msg)
			}

		case *wire.MsgReject:
			if p.cfg.Listeners.OnReject != nil {
				p.cfg.Listeners.OnReject(p, msg)
			}

		case *wire.MsgSendHeaders:
			p.flagsMtx.Lock()
			p.sendHeadersPreferred = true
			p.flagsMtx.Unlock()

			if p.cfg.Listeners.OnSendHeaders != nil {
				p.cfg.Listeners.OnSendHeaders(p, msg)
			}

		default:
			log.Debugf("Received unhandled message of type %v "+
				"from %v", rmsg.Command(), p)
		}
		p.stallControl <- stallControlMsg{sccHandlerDone, rmsg}

//收到一条消息，重置空闲计时器。
		idleTimer.Reset(idleTimeout)
	}

//确保空闲计时器已停止，以避免资源泄漏。
	idleTimer.Stop()

//确保连接已关闭。
	p.Disconnect()

	close(p.inQuit)
	log.Tracef("Peer input handler done for %s", p)
}

//QueueHandler处理对等端的传出数据队列。这就像
//一个muxer，用于各种输入源，这样我们就可以确保服务器和对等机
//处理程序不会阻止我们发送消息。然后数据被传递
//要实际写入的outhandler。
func (p *Peer) queueHandler() {
	pendingMsgs := list.New()
	invSendQueue := list.New()
	trickleTicker := time.NewTicker(p.cfg.TrickleInterval)
	defer trickleTicker.Stop()

//我们保留等待标志以便知道是否有消息排队
//还是到外面去。我们可以用一个脑袋的存在
//但我们对是否
//它是在清理时得到的，因此谁发送
//消息已完成频道。为了避免这种混乱，我们保持不同的
//Flag和PendingMSG仅包含我们尚未包含的消息
//传递给OutHandler。
	waiting := false

//为了避免下面的重复。
	queuePacket := func(msg outMsg, list *list.List, waiting bool) bool {
		if !waiting {
			p.sendQueue <- msg
		} else {
			list.PushBack(msg)
		}
//我们现在总是在等。
		return true
	}
out:
	for {
		select {
		case msg := <-p.outputQueue:
			waiting = queuePacket(msg, pendingMsgs, waiting)

//当消息通过
//网络插座。
		case <-p.sendDoneQueue:
//如果没有更多消息，则不再等待
//在挂起的消息队列中。
			next := pendingMsgs.Front()
			if next == nil {
				waiting = false
				continue
			}

//通知OutHandler下一项
//异步发送。
			val := pendingMsgs.Remove(next)
			p.sendQueue <- val.(outMsg)

		case iv := <-p.outputInvChan:
//没有握手？他们很快就会知道的。
			if p.VersionKnown() {
//如果这是一个新的街区，我们就炸掉它
//立刻出去，啜饮投资部的涓涓细流
//排队。
				if iv.Type == wire.InvTypeBlock ||
					iv.Type == wire.InvTypeWitnessBlock {

					invMsg := wire.NewMsgInvSizeHint(1)
					invMsg.AddInvVect(iv)
					waiting = queuePacket(outMsg{msg: invMsg},
						pendingMsgs, waiting)
				} else {
					invSendQueue.PushBack(iv)
				}
			}

		case <-trickleTicker.C:
//如果我们断开或在那里，不要发送任何东西
//没有排队的库存。
//如果发送队列有任何条目，则版本是已知的。
			if atomic.LoadInt32(&p.disconnect) != 0 ||
				invSendQueue.Len() == 0 {
				continue
			}

//根据需要创建和发送尽可能多的信息
//清空库存发送队列。
			invMsg := wire.NewMsgInvSizeHint(uint(invSendQueue.Len()))
			for e := invSendQueue.Front(); e != nil; e = invSendQueue.Front() {
				iv := invSendQueue.Remove(e).(*wire.InvVect)

//不要发送在
//初始检查。
				if p.knownInventory.Exists(iv) {
					continue
				}

				invMsg.AddInvVect(iv)
				if len(invMsg.InvList) >= maxInvTrickleSize {
					waiting = queuePacket(
						outMsg{msg: invMsg},
						pendingMsgs, waiting)
					invMsg = wire.NewMsgInvSizeHint(uint(invSendQueue.Len()))
				}

//添加正在中继到的清单
//对等机的已知清单。
				p.AddKnownInventory(iv)
			}
			if len(invMsg.InvList) > 0 {
				waiting = queuePacket(outMsg{msg: invMsg},
					pendingMsgs, waiting)
			}

		case <-p.quit:
			break out
		}
	}

//在我们离开之前先排干所有的等待通道，这样我们就不会留下什么
//等待我们。
	for e := pendingMsgs.Front(); e != nil; e = pendingMsgs.Front() {
		val := pendingMsgs.Remove(e)
		msg := val.(outMsg)
		if msg.doneChan != nil {
			msg.doneChan <- struct{}{}
		}
	}
cleanup:
	for {
		select {
		case msg := <-p.outputQueue:
			if msg.doneChan != nil {
				msg.doneChan <- struct{}{}
			}
		case <-p.outputInvChan:
//只是排水渠
//sendDonequeue已缓冲，因此不需要排出。
		default:
			break cleanup
		}
	}
	close(p.queueQuit)
	log.Tracef("Peer queue handler done for %s", p)
}

//shouldLogWriteError返回传递的错误是否为
//预期来自于对outhandler中的远程对等机的写入，
//应记录。
func (p *Peer) shouldLogWriteError(err error) bool {
//当对等端被强制断开连接时不记录日志。
	if atomic.LoadInt32(&p.disconnect) != 0 {
		return false
	}

//远程对等机断开连接后不记录日志。
	if err == io.EOF {
		return false
	}
	if opErr, ok := err.(*net.OpError); ok && !opErr.Temporary() {
		return false
	}

	return true
}

//outhandler处理对等端的所有传出消息。它必须作为
//高尔图它使用缓冲通道序列化输出消息，而
//allowing the sender to continue running asynchronously.
func (p *Peer) outHandler() {
out:
	for {
		select {
		case msg := <-p.sendQueue:
			switch m := msg.msg.(type) {
			case *wire.MsgPing:
//在以后的协议中只需要一个pong消息
//版本。还设置统计。
				if p.ProtocolVersion() > wire.BIP0031Version {
					p.statsMtx.Lock()
					p.lastPingNonce = m.Nonce
					p.lastPingTime = time.Now()
					p.statsMtx.Unlock()
				}
			}

			p.stallControl <- stallControlMsg{sccSendMessage, msg.msg}

			err := p.writeMessage(msg.msg, msg.encoding)
			if err != nil {
				p.Disconnect()
				if p.shouldLogWriteError(err) {
					log.Errorf("Failed to send message to "+
						"%s: %v", p, err)
				}
				if msg.doneChan != nil {
					msg.doneChan <- struct{}{}
				}
				continue
			}

//此时，消息已成功发送，因此
//update the last send time, signal the sender of the
//已发送的消息（如果要求），以及
//将发送队列发送到下一个队列
//消息。
			atomic.StoreInt64(&p.lastSend, time.Now().Unix())
			if msg.doneChan != nil {
				msg.doneChan <- struct{}{}
			}
			p.sendDoneQueue <- struct{}{}

		case <-p.quit:
			break out
		}
	}

	<-p.queueQuit

//在我们离开之前先排干所有的等待通道，这样我们就不会留下什么
//在等我们。我们排队等候，因此我们可以确定
//我们不会错过发送队列上发送的任何内容。
cleanup:
	for {
		select {
		case msg := <-p.sendQueue:
			if msg.doneChan != nil {
				msg.doneChan <- struct{}{}
			}
//由于队列处理程序，无需在senddonequeue上发送
//has been waited on and already exited.
		default:
			break cleanup
		}
	}
	close(p.outQuit)
	log.Tracef("Peer output handler done for %s", p)
}

//PingHandler定期对对等机执行Ping。它必须像野人一样运作。
func (p *Peer) pingHandler() {
	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

out:
	for {
		select {
		case <-pingTicker.C:
			nonce, err := wire.RandomUint64()
			if err != nil {
				log.Errorf("Not sending ping to %s: %v", p, err)
				continue
			}
			p.QueueMessage(wire.NewMsgPing(nonce), nil)

		case <-p.quit:
			break out
		}
	}
}

//QueueMessage将传递的比特币消息添加到对等发送队列中。
//
//此函数对于并发访问是安全的。
func (p *Peer) QueueMessage(msg wire.Message, doneChan chan<- struct{}) {
	p.QueueMessageWithEncoding(msg, doneChan, wire.BaseEncoding)
}

//QueueMessageWithEncoding将传递的比特币消息添加到对等发送中
//排队。此函数与QueueMessage相同，但它允许
//调用方指定当
//编码/解码块和事务。
//
//此函数对于并发访问是安全的。
func (p *Peer) QueueMessageWithEncoding(msg wire.Message, doneChan chan<- struct{},
	encoding wire.MessageEncoding) {

//如果Goroutine已经退出，则避免死锁风险。龙骨
//我们将派他们四处游荡直到他们知道
//it is marked as disconnected and *then* it drains the channels.
	if !p.Connected() {
		if doneChan != nil {
			go func() {
				doneChan <- struct{}{}
			}()
		}
		return
	}
	p.outputQueue <- outMsg{msg: msg, encoding: encoding, doneChan: doneChan}
}

//queue inventory将传递的库存添加到库存发送队列，该队列
//可能不会立即发送，而是分批发送给对等机。
//已忽略对等方已知拥有的清单。
//
//此函数对于并发访问是安全的。
func (p *Peer) QueueInventory(invVect *wire.InvVect) {
//如果对等点已经存在，不要将库存添加到发送队列中。
//已知拥有它。
	if p.knownInventory.Exists(invVect) {
		return
	}

//如果Goroutine已经退出，则避免死锁风险。龙骨
//我们将派他们四处游荡直到他们知道
//它被标记为断开，然后*排出通道。
	if !p.Connected() {
		return
	}

	p.outputInvChan <- invVect
}

//connected返回对等机当前是否已连接。
//
//此函数对于并发访问是安全的。
func (p *Peer) Connected() bool {
	return atomic.LoadInt32(&p.connected) != 0 &&
		atomic.LoadInt32(&p.disconnect) == 0
}

//断开连接通过关闭连接来断开对等机的连接。调用此
//当对等端已断开连接或正在
//断开不会有任何效果。
func (p *Peer) Disconnect() {
	if atomic.AddInt32(&p.disconnect, 1) != 1 {
		return
	}

	log.Tracef("Disconnecting %s", p)
	if atomic.LoadInt32(&p.connected) != 0 {
		p.conn.Close()
	}
	close(p.quit)
}

//Read ReaveViston MSG等待从远程到达的下一条消息
//同龄人。如果下一条消息不是版本消息或版本不是
//可接受，然后返回错误。
func (p *Peer) readRemoteVersionMsg() error {
//阅读他们的版本信息。
	remoteMsg, _, err := p.readMessage(wire.LatestEncoding)
	if err != nil {
		return err
	}

//如果第一条消息不是版本，则通知并断开客户端连接
//消息。
	msg, ok := remoteMsg.(*wire.MsgVersion)
	if !ok {
		reason := "a version message must precede all others"
		rejectMsg := wire.NewMsgReject(msg.Command(), wire.RejectMalformed,
			reason)
		_ = p.writeMessage(rejectMsg, wire.LatestEncoding)
		return errors.New(reason)
	}

//检测自我连接。
	if !allowSelfConns && sentNonces.Exists(msg.Nonce) {
		return errors.New("disconnecting peer connected to self")
	}

//协商协议版本并将服务设置为远程
//peer advertised.
	p.flagsMtx.Lock()
	p.advertisedProtoVer = uint32(msg.ProtocolVersion)
	p.protocolVersion = minUint32(p.protocolVersion, p.advertisedProtoVer)
	p.versionKnown = true
	p.services = msg.Services
	p.flagsMtx.Unlock()
	log.Debugf("Negotiated protocol version %d for peer %s",
		p.protocolVersion, p)

//更新一系列统计信息，包括基于块的统计信息，以及
//对等机的时间偏移。
	p.statsMtx.Lock()
	p.lastBlock = msg.LastBlock
	p.startingHeight = msg.LastBlock
	p.timeOffset = msg.Timestamp.Unix() - time.Now().Unix()
	p.statsMtx.Unlock()

//设置对等机的ID、用户代理，以及
//specifies the witness support is enabled.
	p.flagsMtx.Lock()
	p.id = atomic.AddInt32(&nodeCount, 1)
	p.userAgent = msg.UserAgent

//确定对等端是否希望接收见证数据
//是否交易。
	if p.services&wire.SFNodeWitness == wire.SFNodeWitness {
		p.witnessEnabled = true
	}
	p.flagsMtx.Unlock()

//一旦交换了版本消息，我们就可以确定
//如果此对等方知道如何在线路上编码见证数据
//协议。如果是这样，那么我们将切换到解码模式，即
//为作为
//BIP0144
	if p.services&wire.SFNodeWitness == wire.SFNodeWitness {
		p.wireEncoding = wire.WitnessEncoding
	}

//如果指定，则调用回调。
	if p.cfg.Listeners.OnVersion != nil {
		rejectMsg := p.cfg.Listeners.OnVersion(p, msg)
		if rejectMsg != nil {
			_ = p.writeMessage(rejectMsg, wire.LatestEncoding)
			return errors.New(rejectMsg.Reason)
		}
	}

//通知并断开具有协议版本的客户端
//太老了。
//
//注：如果minacacceptableprotocolversion被提升到高于
//Wire.RejectVersion，这应该在
//断开连接。
	if uint32(msg.ProtocolVersion) < MinAcceptableProtocolVersion {
//发送拒绝消息，指示协议版本为
//过时，等待消息在
//断开连接。
		reason := fmt.Sprintf("protocol version must be %d or greater",
			MinAcceptableProtocolVersion)
		rejectMsg := wire.NewMsgReject(msg.Command(), wire.RejectObsolete,
			reason)
		_ = p.writeMessage(rejectMsg, wire.LatestEncoding)
		return errors.New(reason)
	}

	return nil
}

//localversionmsg创建可用于发送到
//远程对等体。
func (p *Peer) localVersionMsg() (*wire.MsgVersion, error) {
	var blockNum int32
	if p.cfg.NewestBlock != nil {
		var err error
		_, blockNum, err = p.cfg.NewestBlock()
		if err != nil {
			return nil, err
		}
	}

	theirNA := p.na

//如果我们在代理后面，并且连接来自代理，那么
//we return an unroutable address as their address. This is to prevent
//正在泄漏Tor代理地址。
	if p.cfg.Proxy != "" {
		proxyaddress, _, err := net.SplitHostPort(p.cfg.Proxy)
//无效的代理意味着配置不好，处于安全方面。
		if err != nil || p.na.IP.String() == proxyaddress {
			theirNA = wire.NewNetAddressIPPort(net.IP([]byte{0, 0, 0, 0}), 0,
				theirNA.Services)
		}
	}

//Create a wire.NetAddress with only the services set to use as the
//版本消息中的“addrme”。
//
//以前将IP和端口信息添加到
//地址管理器被证明是不可靠的入站地址管理器
//来自对等端的连接并不一定意味着对等端本身
//接受的入站连接。
//
//另外，时间戳在版本消息中是未使用的。
	ourNA := &wire.NetAddress{
		Services: p.cfg.Services,
	}

//为此对等端生成唯一的nonce，以便可以
//检测。这是通过将其添加到
//最近看到的nonces。
	nonce := uint64(rand.Int63())
	sentNonces.Add(nonce)

//版本消息。
	msg := wire.NewMsgVersion(ourNA, theirNA, nonce, blockNum)
	msg.AddUserAgent(p.cfg.UserAgentName, p.cfg.UserAgentVersion,
		p.cfg.UserAgentComments...)

//Advertise local services.
	msg.Services = p.cfg.Services

//公布我们支持的最大协议版本。
	msg.ProtocolVersion = int32(p.cfg.ProtocolVersion)

//如果需要交易的INV消息，则进行广告。
	msg.DisableRelayTx = p.cfg.DisableRelayTx

	return msg, nil
}

//writeLocalVersionMsg writes our version message to the remote peer.
func (p *Peer) writeLocalVersionMsg() error {
	localVerMsg, err := p.localVersionMsg()
	if err != nil {
		return err
	}

	return p.writeMessage(localVerMsg, wire.LatestEncoding)
}

//NegotiateBoundProtocol等待从对等端接收版本消息
//然后发送我们的版本消息。如果事件不是按这个顺序发生的，那么
//it returns an error.
func (p *Peer) negotiateInboundProtocol() error {
	if err := p.readRemoteVersionMsg(); err != nil {
		return err
	}

	return p.writeLocalVersionMsg()
}

//NegotiateOutboundProtocol发送我们的版本消息，然后等待接收
//来自对等方的版本消息。如果事件不是按这个顺序发生的，那么
//它返回一个错误。
func (p *Peer) negotiateOutboundProtocol() error {
	if err := p.writeLocalVersionMsg(); err != nil {
		return err
	}

	return p.readRemoteVersionMsg()
}

//开始处理输入和输出消息。
func (p *Peer) start() error {
	log.Tracef("Starting peer %s", p)

	negotiateErr := make(chan error, 1)
	go func() {
		if p.inbound {
			negotiateErr <- p.negotiateInboundProtocol()
		} else {
			negotiateErr <- p.negotiateOutboundProtocol()
		}
	}()

//在指定的协商超时内协商协议。
	select {
	case err := <-negotiateErr:
		if err != nil {
			p.Disconnect()
			return err
		}
	case <-time.After(negotiateTimeout):
		p.Disconnect()
		return errors.New("protocol negotiation timeout")
	}
	log.Debugf("Connected to %s", p.Addr())

//协议已成功协商，因此开始处理输入
//并输出消息。
	go p.stallHandler()
	go p.inHandler()
	go p.queueHandler()
	go p.outHandler()
	go p.pingHandler()

//现在IO处理机器已经启动，请发送我们的verack消息。
	p.QueueMessage(wire.NewMsgVerAck(), nil)
	return nil
}

//associateConnection将给定的conn关联到对等端。调用此
//对等端已连接时的函数将不起作用。
func (p *Peer) AssociateConnection(conn net.Conn) {
//已经连接？
	if !atomic.CompareAndSwapInt32(&p.connected, 0, 1) {
		return
	}

	p.conn = conn
	p.timeConnected = time.Now()

	if p.inbound {
		p.addr = p.conn.RemoteAddr().String()

//为要与addrmanager一起使用的对等机设置网络地址。我们
//只做这个入站，因为出站在连接时设置了这个
//没有必要重新计算。
		na, err := newNetAddress(p.conn.RemoteAddr(), p.services)
		if err != nil {
			log.Errorf("Cannot create remote net address: %v", err)
			p.Disconnect()
			return
		}
		p.na = na
	}

	go func() {
		if err := p.start(); err != nil {
			log.Debugf("Cannot start peer %v: %v", p, err)
			p.Disconnect()
		}
	}()
}

//WaitForDisconnect等待对等端完全断开连接
//清理资源。如果本地或远程
//端已断开连接或通过以下方式强制断开对等机
//断开连接。
func (p *Peer) WaitForDisconnect() {
	<-p.quit
}

//NewPeerBase基于入站标志返回新的基本比特币对等。这个
//由NeWangBooLee和NeXOutBoin对等函数使用以执行基
//两种类型的对等机都需要设置。
func newPeerBase(origCfg *Config, inbound bool) *Peer {
//如果未由指定，则默认为支持的最大协议版本
//来电者。
cfg := *origCfg //复制以避免调用者发生变化。
	if cfg.ProtocolVersion == 0 {
		cfg.ProtocolVersion = MaxProtocolVersion
	}

//Set the chain parameters to testnet if the caller did not specify any.
	if cfg.ChainParams == nil {
		cfg.ChainParams = &chaincfg.TestNet3Params
	}

//如果指定了非正值，则设置涓流间隔。
	if cfg.TrickleInterval <= 0 {
		cfg.TrickleInterval = DefaultTrickleInterval
	}

	p := Peer{
		inbound:         inbound,
		wireEncoding:    wire.BaseEncoding,
		knownInventory:  newMruInventoryMap(maxKnownInventory),
stallControl:    make(chan stallControlMsg, 1), //非阻塞同步
		outputQueue:     make(chan outMsg, outputBufferSize),
sendQueue:       make(chan outMsg, 1),   //非阻塞同步
sendDoneQueue:   make(chan struct{}, 1), //非阻塞同步
		outputInvChan:   make(chan *wire.InvVect, outputBufferSize),
		inQuit:          make(chan struct{}),
		queueQuit:       make(chan struct{}),
		outQuit:         make(chan struct{}),
		quit:            make(chan struct{}),
cfg:             cfg, //复制以便调用者不能变异。
		services:        cfg.Services,
		protocolVersion: cfg.ProtocolVersion,
	}
	return &p
}

//new inbound peer返回新的入站比特币对等。使用“开始”开始
//处理传入和传出消息。
func NewInboundPeer(cfg *Config) *Peer {
	return newPeerBase(cfg, true)
}

//NeXOutBanger-Peer-Read推出新的比特币同行。
func NewOutboundPeer(cfg *Config, addr string) (*Peer, error) {
	p := newPeerBase(cfg, false)
	p.addr = addr

	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return nil, err
	}

	if cfg.HostToNetAddress != nil {
		na, err := cfg.HostToNetAddress(host, uint16(port), 0)
		if err != nil {
			return nil, err
		}
		p.na = na
	} else {
		p.na = wire.NewNetAddressIPPort(net.ParseIP(host), uint16(port), 0)
	}

	return p, nil
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
