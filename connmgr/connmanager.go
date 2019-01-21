
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

package connmgr

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

//MaxFailedAttempts是连续失败连接的最大数目。
//假定网络故障后的尝试，新连接将
//被配置的重试持续时间延迟。
const maxFailedAttempts = 25

var (
//errdailnil用于指示在配置中拨号不能为nil。
	ErrDialNil = errors.New("Config: Dial cannot be nil")

//MaxRetryDuration是持久的
//允许连接增长到。这是必要的，因为重试
//逻辑使用退避机制来增加间隔基准时间
//已完成的重试次数。
	maxRetryDuration = time.Minute * 5

//DefaultRetryDuration是重试的默认持续时间
//持久连接。
	defaultRetryDuration = time.Second * 5

//DefaultTargetOutbound是默认的出站连接数
//维护。
	defaultTargetOutbound = uint32(8)
)

//Connstate表示请求的连接的状态。
type ConnState uint8

//connstate可以是挂起、建立、断开连接或失败。什么时候？
//请求了一个新连接，它被尝试并分类为
//已建立或失败取决于连接结果。已建立的
//断开的连接被归类为断开连接。
const (
	ConnPending ConnState = iota
	ConnFailing
	ConnCanceled
	ConnEstablished
	ConnDisconnected
)

//connreq是到网络地址的连接请求。如果是永久性的，
//断开连接时将重试连接。
type ConnReq struct {
//以下变量只能原子地使用。
	id uint64

	Addr      net.Addr
	Permanent bool

	conn       net.Conn
	state      ConnState
	stateMtx   sync.RWMutex
	retryCount uint32
}

//updateState更新连接请求的状态。
func (c *ConnReq) updateState(state ConnState) {
	c.stateMtx.Lock()
	c.state = state
	c.stateMtx.Unlock()
}

//ID返回连接请求的唯一标识符。
func (c *ConnReq) ID() uint64 {
	return atomic.LoadUint64(&c.id)
}

//State是所请求连接的连接状态。
func (c *ConnReq) State() ConnState {
	c.stateMtx.RLock()
	state := c.state
	c.stateMtx.RUnlock()
	return state
}

//String为连接请求返回一个人类可读字符串。
func (c *ConnReq) String() string {
	if c.Addr == nil || c.Addr.String() == "" {
		return fmt.Sprintf("reqid %d", atomic.LoadUint64(&c.id))
	}
	return fmt.Sprintf("%s (reqid %d)", c.Addr, atomic.LoadUint64(&c.id))
}

//config保存与连接管理器相关的配置选项。
type Config struct {
//侦听器定义连接所针对的侦听器切片
//经理将拥有并接受连接。当A
//连接被接受，OnAccept处理程序将用
//连接。因为连接管理器拥有这些
//侦听器，当连接管理器
//停止。
//
//如果onAccept字段不为
//也有规定。如果打电话的人不想听，可能是零
//对于传入连接。
	Listeners []net.Listener

//OnAccept是当入站连接
//认可的。呼叫方负责关闭连接。
//关闭连接失败将导致连接管理器
//相信这种联系仍然是活跃的，因此不受欢迎
//副作用，如仍在向最大连接限制计数。
//
//如果listeners字段不是
//还指定了，因为不可能接受任何
//这种情况下的连接。
	OnAccept func(net.Conn)

//目标输出是出站网络连接的数量。
//维护。默认值为8。
	TargetOutbound uint32

//RetryDuration是重试连接前等待的持续时间。
//请求。默认值为5s。
	RetryDuration time.Duration

//OnConnection是在新出站时激发的回调
//已建立连接。
	OnConnection func(*ConnReq, net.Conn)

//OnDisconnection是在出站时激发的回调
//连接已断开。
	OnDisconnection func(*ConnReq)

//GetNewAddress是一种获取地址以进行网络连接的方法
//去。如果为零，则不会自动建立新连接。
	GetNewAddress func() (net.Addr, error)

//拨号连接到指定网络上的地址。不能为零。
	Dial func(net.Addr) (net.Conn, error)
}

//RegisterPending用于注册挂起的连接尝试。通过
//注册挂起的连接尝试我们允许呼叫者取消挂起
//在成功之前尝试连接，或者在没有成功的情况下尝试连接
//更长的通缉令
type registerPending struct {
	c    *ConnReq
	done chan struct{}
}

//handleConnected用于排队成功的连接。
type handleConnected struct {
	c    *ConnReq
	conn net.Conn
}

//handledisconnected用于删除连接。
type handleDisconnected struct {
	id    uint64
	retry bool
}

//handlefailed用于删除挂起的连接。
type handleFailed struct {
	c   *ConnReq
	err error
}

//ConManager提供一个管理器来处理网络连接。
type ConnManager struct {
//以下变量只能原子地使用。
	connReqCount uint64
	start        int32
	stop         int32

	cfg            Config
	wg             sync.WaitGroup
	failedAttempts uint64
	requests       chan interface{}
	quit           chan struct{}
}

//handlefailedconn处理由于断开或任何
//其他故障。如果是永久的，则在配置
//重试持续时间。否则，如果需要，它会发出新的连接请求。
//在maxfailedconnectionattempts之后，将在
//已配置重试持续时间。
func (cm *ConnManager) handleFailedConn(c *ConnReq) {
	if atomic.LoadInt32(&cm.stop) != 0 {
		return
	}
	if c.Permanent {
		c.retryCount++
		d := time.Duration(c.retryCount) * cm.cfg.RetryDuration
		if d > maxRetryDuration {
			d = maxRetryDuration
		}
		log.Debugf("Retrying connection to %v in %v", c, d)
		time.AfterFunc(d, func() {
			cm.Connect(c)
		})
	} else if cm.cfg.GetNewAddress != nil {
		cm.failedAttempts++
		if cm.failedAttempts >= maxFailedAttempts {
			log.Debugf("Max failed connection attempts reached: [%d] "+
				"-- retrying connection in: %v", maxFailedAttempts,
				cm.cfg.RetryDuration)
			time.AfterFunc(cm.cfg.RetryDuration, func() {
				cm.NewConnReq()
			})
		} else {
			go cm.NewConnReq()
		}
	}
}

//Connhandler处理所有与连接相关的请求。它必须作为
//高尔图
//
//连接处理程序确保维护活动出站池
//使我们保持与网络的连接。连接请求
//由分配的ID处理和映射。
func (cm *ConnManager) connHandler() {

	var (
//挂起保留所有尚未注册的conn请求
//成功。
		pending = make(map[uint64]*ConnReq)

//conns表示所有主动连接的对等端的集合。
		conns = make(map[uint64]*ConnReq, cm.cfg.TargetOutbound)
	)

out:
	for {
		select {
		case req := <-cm.requests:
			switch msg := req.(type) {

			case registerPending:
				connReq := msg.c
				connReq.updateState(ConnPending)
				pending[msg.c.id] = connReq
				close(msg.done)

			case handleConnected:
				connReq := msg.c

				if _, ok := pending[connReq.id]; !ok {
					if msg.conn != nil {
						msg.conn.Close()
					}
					log.Debugf("Ignoring connection for "+
						"canceled connreq=%v", connReq)
					continue
				}

				connReq.updateState(ConnEstablished)
				connReq.conn = msg.conn
				conns[connReq.id] = connReq
				log.Debugf("Connected to %v", connReq)
				connReq.retryCount = 0
				cm.failedAttempts = 0

				delete(pending, connReq.id)

				if cm.cfg.OnConnection != nil {
					go cm.cfg.OnConnection(connReq, msg.conn)
				}

			case handleDisconnected:
				connReq, ok := conns[msg.id]
				if !ok {
					connReq, ok = pending[msg.id]
					if !ok {
						log.Errorf("Unknown connid=%d",
							msg.id)
						continue
					}

//找到挂起的连接，删除
//如果我们应该的话，它来自待定的地图
//稍后忽略，成功
//连接。
					connReq.updateState(ConnCanceled)
					log.Debugf("Canceling: %v", connReq)
					delete(pending, msg.id)
					continue

				}

//已找到现有连接，标记为
//断开并执行断开
//回调。
				log.Debugf("Disconnected from %v", connReq)
				delete(conns, msg.id)

				if connReq.conn != nil {
					connReq.conn.Close()
				}

				if cm.cfg.OnDisconnection != nil {
					go cm.cfg.OnDisconnection(connReq)
				}

//所有内部状态都已清除，如果
//正在删除此连接，我们将
//不再尝试此请求。
				if !msg.retry {
					connReq.updateState(ConnDisconnected)
					continue
				}

//否则，我们将尝试重新连接，如果
//我们没有足够的同龄人，或者如果这是
//持久对等。连接请求是
//重新添加到挂起的映射，以便
//连接的后续处理和
//失败不会忽略请求。
				if uint32(len(conns)) < cm.cfg.TargetOutbound ||
					connReq.Permanent {

					connReq.updateState(ConnPending)
					log.Debugf("Reconnecting to %v",
						connReq)
					pending[msg.id] = connReq
					cm.handleFailedConn(connReq)
				}

			case handleFailed:
				connReq := msg.c

				if _, ok := pending[connReq.id]; !ok {
					log.Debugf("Ignoring connection for "+
						"canceled conn req: %v", connReq)
					continue
				}

				connReq.updateState(ConnFailing)
				log.Debugf("Failed to connect to %v: %v",
					connReq, msg.err)
				cm.handleFailedConn(connReq)
			}

		case <-cm.quit:
			break out
		}
	}

	cm.wg.Done()
	log.Trace("Connection handler done")
}

//newcnnreq创建新的连接请求并连接到
//对应地址。
func (cm *ConnManager) NewConnReq() {
	if atomic.LoadInt32(&cm.stop) != 0 {
		return
	}
	if cm.cfg.GetNewAddress == nil {
		return
	}

	c := &ConnReq{}
	atomic.StoreUint64(&c.id, atomic.AddUint64(&cm.connReqCount, 1))

//向连接提交挂起连接尝试的请求
//经理。在连接均匀之前注册ID
//已建立，稍后我们可以通过
//去除方法。
	done := make(chan struct{})
	select {
	case cm.requests <- registerPending{c, done}:
	case <-cm.quit:
		return
	}

//等待注册成功将挂起的conn req添加到
//连接管理器的内部状态。
	select {
	case <-done:
	case <-cm.quit:
		return
	}

	addr, err := cm.cfg.GetNewAddress()
	if err != nil {
		select {
		case cm.requests <- handleFailed{c, err}:
		case <-cm.quit:
		}
		return
	}

	c.Addr = addr

	cm.Connect(c)
}

//Connect分配一个ID，并将连接拨到
//连接请求。
func (cm *ConnManager) Connect(c *ConnReq) {
	if atomic.LoadInt32(&cm.stop) != 0 {
		return
	}
	if atomic.LoadUint64(&c.id) == 0 {
		atomic.StoreUint64(&c.id, atomic.AddUint64(&cm.connReqCount, 1))

//将挂起连接尝试的请求提交到
//连接管理器。通过在
//我们甚至建立了联系，以后我们可以
//通过删除方法取消连接。
		done := make(chan struct{})
		select {
		case cm.requests <- registerPending{c, done}:
		case <-cm.quit:
			return
		}

//等待注册成功添加挂起的
//连接请求到连接管理器的内部状态。
		select {
		case <-done:
		case <-cm.quit:
			return
		}
	}

	log.Debugf("Attempting to connect to %v", c)

	conn, err := cm.cfg.Dial(c.Addr)
	if err != nil {
		select {
		case cm.requests <- handleFailed{c, err}:
		case <-cm.quit:
		}
		return
	}

	select {
	case cm.requests <- handleConnected{c, conn}:
	case <-cm.quit:
	}
}

//断开连接断开与给定连接对应的连接
//ID.如果是永久连接，则将通过增加回退重试连接。
//持续时间。
func (cm *ConnManager) Disconnect(id uint64) {
	if atomic.LoadInt32(&cm.stop) != 0 {
		return
	}

	select {
	case cm.requests <- handleDisconnected{id, true}:
	case <-cm.quit:
	}
}

//移除从中移除与给定连接ID对应的连接
//已知连接。
//
//注意：此方法也可用于取消延迟连接尝试。
//这还没有成功。
func (cm *ConnManager) Remove(id uint64) {
	if atomic.LoadInt32(&cm.stop) != 0 {
		return
	}

	select {
	case cm.requests <- handleDisconnected{id, false}:
	case <-cm.quit:
	}
}

//listenhandler接受给定侦听器上的传入连接。一定是
//像野人一样奔跑。
func (cm *ConnManager) listenHandler(listener net.Listener) {
	log.Infof("Server listening on %s", listener.Addr())
	for atomic.LoadInt32(&cm.stop) == 0 {
		conn, err := listener.Accept()
		if err != nil {
//如果不强制关闭，则只记录错误。
			if atomic.LoadInt32(&cm.stop) == 0 {
				log.Errorf("Can't accept connection: %v", err)
			}
			continue
		}
		go cm.cfg.OnAccept(conn)
	}

	cm.wg.Done()
	log.Tracef("Listener handler done for %s", listener.Addr())
}

//Start启动连接管理器并开始连接到网络。
func (cm *ConnManager) Start() {
//已经开始？
	if atomic.AddInt32(&cm.start, 1) != 1 {
		return
	}

	log.Trace("Connection manager started")
	cm.wg.Add(1)
	go cm.connHandler()

//只要呼叫者请求，就启动所有的听众，并且
//提供了在接受连接时调用的回调。
	if cm.cfg.OnAccept != nil {
		for _, listner := range cm.cfg.Listeners {
			cm.wg.Add(1)
			go cm.listenHandler(listner)
		}
	}

	for i := atomic.LoadUint64(&cm.connReqCount); i < uint64(cm.cfg.TargetOutbound); i++ {
		go cm.NewConnReq()
	}
}

//等待块，直到连接管理器正常停止。
func (cm *ConnManager) Wait() {
	cm.wg.Wait()
}

//停止优雅地关闭连接管理器。
func (cm *ConnManager) Stop() {
	if atomic.AddInt32(&cm.stop, 1) != 1 {
		log.Warnf("Connection manager already stopped")
		return
	}

//停止所有听众。如果
//听力被禁用。
	for _, listener := range cm.cfg.Listeners {
//忽略错误，因为这是关闭的，并且没有办法
//无论如何都要恢复。
		_ = listener.Close()
	}

	close(cm.quit)
	log.Trace("Connection manager stopped")
}

//New返回新的连接管理器。
//使用“开始”开始连接到网络。
func New(cfg *Config) (*ConnManager, error) {
	if cfg.Dial == nil {
		return nil, ErrDialNil
	}
//默认为正常值
	if cfg.RetryDuration <= 0 {
		cfg.RetryDuration = defaultRetryDuration
	}
	if cfg.TargetOutbound == 0 {
		cfg.TargetOutbound = defaultTargetOutbound
	}
	cm := ConnManager{
cfg:      *cfg, //复制以使调用者不能变异
		requests: make(chan interface{}),
		quit:     make(chan struct{}),
	}
	return &cm, nil
}
