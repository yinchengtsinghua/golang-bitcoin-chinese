
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
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func init() {
//在运行测试时覆盖最大重试持续时间。
	maxRetryDuration = 2 * time.Millisecond
}

//mockaddr模拟网络地址
type mockAddr struct {
	net, address string
}

func (m mockAddr) Network() string { return m.net }
func (m mockAddr) String() string  { return m.address }

//mockconn通过实现net.conn接口来模拟网络连接。
type mockConn struct {
	io.Reader
	io.Writer
	io.Closer

//本地网络，连接地址。
	lnet, laddr string

//远程网络，连接地址。
	rAddr net.Addr
}

//localaddr返回连接的本地地址。
func (c mockConn) LocalAddr() net.Addr {
	return &mockAddr{c.lnet, c.laddr}
}

//remoteaddr返回连接的远程地址。
func (c mockConn) RemoteAddr() net.Addr {
	return &mockAddr{c.rAddr.Network(), c.rAddr.String()}
}

//关闭手柄关闭连接。
func (c mockConn) Close() error {
	return nil
}

func (c mockConn) SetDeadline(t time.Time) error      { return nil }
func (c mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (c mockConn) SetWriteDeadline(t time.Time) error { return nil }

//mockdialer通过返回模拟连接来模拟net.dial接口
//给定的地址。
func mockDialer(addr net.Addr) (net.Conn, error) {
	r, w := io.Pipe()
	c := &mockConn{rAddr: addr}
	c.Reader = r
	c.Writer = w
	return c, nil
}

//TestNewConfig测试新的ConManager配置是否按预期进行验证。
func TestNewConfig(t *testing.T) {
	_, err := New(&Config{})
	if err == nil {
		t.Fatalf("New expected error: 'Dial can't be nil', got nil")
	}
	_, err = New(&Config{
		Dial: mockDialer,
	})
	if err != nil {
		t.Fatalf("New unexpected error: %v", err)
	}
}

//teststartstop测试连接管理器的启动和停止方式
//预期。
func TestStartStop(t *testing.T) {
	connected := make(chan *ConnReq)
	disconnected := make(chan *ConnReq)
	cmgr, err := New(&Config{
		TargetOutbound: 1,
		GetNewAddress: func() (net.Addr, error) {
			return &net.TCPAddr{
				IP:   net.ParseIP("127.0.0.1"),
				Port: 18555,
			}, nil
		},
		Dial: mockDialer,
		OnConnection: func(c *ConnReq, conn net.Conn) {
			connected <- c
		},
		OnDisconnection: func(c *ConnReq) {
			disconnected <- c
		},
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	cmgr.Start()
	gotConnReq := <-connected
	cmgr.Stop()
//已经停止
	cmgr.Stop()
//忽略
	cr := &ConnReq{
		Addr: &net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18555,
		},
		Permanent: true,
	}
	cmgr.Connect(cr)
	if cr.ID() != 0 {
		t.Fatalf("start/stop: got id: %v, want: 0", cr.ID())
	}
	cmgr.Disconnect(gotConnReq.ID())
	cmgr.Remove(gotConnReq.ID())
	select {
	case <-disconnected:
		t.Fatalf("start/stop: unexpected disconnection")
	case <-time.Tick(10 * time.Millisecond):
		break
	}
}

//测试连接模式测试连接管理器在连接模式下工作。
//
//在连接模式下，自动连接被禁用，因此我们测试
//使用connect的请求将被处理，并且不会建立其他连接。
func TestConnectMode(t *testing.T) {
	connected := make(chan *ConnReq)
	cmgr, err := New(&Config{
		TargetOutbound: 2,
		Dial:           mockDialer,
		OnConnection: func(c *ConnReq, conn net.Conn) {
			connected <- c
		},
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	cr := &ConnReq{
		Addr: &net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18555,
		},
		Permanent: true,
	}
	cmgr.Start()
	cmgr.Connect(cr)
	gotConnReq := <-connected
	wantID := cr.ID()
	gotID := gotConnReq.ID()
	if gotID != wantID {
		t.Fatalf("connect mode: %v - want ID %v, got ID %v", cr.Addr, wantID, gotID)
	}
	gotState := cr.State()
	wantState := ConnEstablished
	if gotState != wantState {
		t.Fatalf("connect mode: %v - want state %v, got state %v", cr.Addr, wantState, gotState)
	}
	select {
	case c := <-connected:
		t.Fatalf("connect mode: got unexpected connection - %v", c.Addr)
	case <-time.After(time.Millisecond):
		break
	}
	cmgr.Stop()
}

//TestTargetOutbound测试出站连接的目标数量。
//
//我们等待所有连接建立，然后测试它们
//仅建立连接。
func TestTargetOutbound(t *testing.T) {
	targetOutbound := uint32(10)
	connected := make(chan *ConnReq)
	cmgr, err := New(&Config{
		TargetOutbound: targetOutbound,
		Dial:           mockDialer,
		GetNewAddress: func() (net.Addr, error) {
			return &net.TCPAddr{
				IP:   net.ParseIP("127.0.0.1"),
				Port: 18555,
			}, nil
		},
		OnConnection: func(c *ConnReq, conn net.Conn) {
			connected <- c
		},
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	cmgr.Start()
	for i := uint32(0); i < targetOutbound; i++ {
		<-connected
	}

	select {
	case c := <-connected:
		t.Fatalf("target outbound: got unexpected connection - %v", c.Addr)
	case <-time.After(time.Millisecond):
		break
	}
	cmgr.Stop()
}

//TestRetryPermanent测试是否重试永久连接请求。
//
//我们使用connect发出永久连接请求，使用
//断开连接，我们等待它重新连接。
func TestRetryPermanent(t *testing.T) {
	connected := make(chan *ConnReq)
	disconnected := make(chan *ConnReq)
	cmgr, err := New(&Config{
		RetryDuration:  time.Millisecond,
		TargetOutbound: 1,
		Dial:           mockDialer,
		OnConnection: func(c *ConnReq, conn net.Conn) {
			connected <- c
		},
		OnDisconnection: func(c *ConnReq) {
			disconnected <- c
		},
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	cr := &ConnReq{
		Addr: &net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18555,
		},
		Permanent: true,
	}
	go cmgr.Connect(cr)
	cmgr.Start()
	gotConnReq := <-connected
	wantID := cr.ID()
	gotID := gotConnReq.ID()
	if gotID != wantID {
		t.Fatalf("retry: %v - want ID %v, got ID %v", cr.Addr, wantID, gotID)
	}
	gotState := cr.State()
	wantState := ConnEstablished
	if gotState != wantState {
		t.Fatalf("retry: %v - want state %v, got state %v", cr.Addr, wantState, gotState)
	}

	cmgr.Disconnect(cr.ID())
	gotConnReq = <-disconnected
	wantID = cr.ID()
	gotID = gotConnReq.ID()
	if gotID != wantID {
		t.Fatalf("retry: %v - want ID %v, got ID %v", cr.Addr, wantID, gotID)
	}
	gotState = cr.State()
	wantState = ConnPending
	if gotState != wantState {
		t.Fatalf("retry: %v - want state %v, got state %v", cr.Addr, wantState, gotState)
	}

	gotConnReq = <-connected
	wantID = cr.ID()
	gotID = gotConnReq.ID()
	if gotID != wantID {
		t.Fatalf("retry: %v - want ID %v, got ID %v", cr.Addr, wantID, gotID)
	}
	gotState = cr.State()
	wantState = ConnEstablished
	if gotState != wantState {
		t.Fatalf("retry: %v - want state %v, got state %v", cr.Addr, wantState, gotState)
	}

	cmgr.Remove(cr.ID())
	gotConnReq = <-disconnected
	wantID = cr.ID()
	gotID = gotConnReq.ID()
	if gotID != wantID {
		t.Fatalf("retry: %v - want ID %v, got ID %v", cr.Addr, wantID, gotID)
	}
	gotState = cr.State()
	wantState = ConnDisconnected
	if gotState != wantState {
		t.Fatalf("retry: %v - want state %v, got state %v", cr.Addr, wantState, gotState)
	}
	cmgr.Stop()
}

//testmaxretryduration测试最大重试持续时间。
//
//我们有一个定时拨号程序，它最初返回err，但在重试之后
//点击maxretryduration返回模拟连接。
func TestMaxRetryDuration(t *testing.T) {
	networkUp := make(chan struct{})
	time.AfterFunc(5*time.Millisecond, func() {
		close(networkUp)
	})
	timedDialer := func(addr net.Addr) (net.Conn, error) {
		select {
		case <-networkUp:
			return mockDialer(addr)
		default:
			return nil, errors.New("network down")
		}
	}

	connected := make(chan *ConnReq)
	cmgr, err := New(&Config{
		RetryDuration:  time.Millisecond,
		TargetOutbound: 1,
		Dial:           timedDialer,
		OnConnection: func(c *ConnReq, conn net.Conn) {
			connected <- c
		},
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	cr := &ConnReq{
		Addr: &net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18555,
		},
		Permanent: true,
	}
	go cmgr.Connect(cr)
	cmgr.Start()
//1ms重试
//2毫秒后重试-已达到最大重试持续时间
//2毫秒后重试-TimedDialer返回MockDial
	select {
	case <-connected:
	case <-time.Tick(100 * time.Millisecond):
		t.Fatalf("max retry duration: connection timeout")
	}
}

//测试网络连接管理器处理网络的失败测试
//优雅地失败。
func TestNetworkFailure(t *testing.T) {
	var dials uint32
	errDialer := func(net net.Addr) (net.Conn, error) {
		atomic.AddUint32(&dials, 1)
		return nil, errors.New("network down")
	}
	cmgr, err := New(&Config{
		TargetOutbound: 5,
		RetryDuration:  5 * time.Millisecond,
		Dial:           errDialer,
		GetNewAddress: func() (net.Addr, error) {
			return &net.TCPAddr{
				IP:   net.ParseIP("127.0.0.1"),
				Port: 18555,
			}, nil
		},
		OnConnection: func(c *ConnReq, conn net.Conn) {
			t.Fatalf("network failure: got unexpected connection - %v", c.Addr)
		},
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	cmgr.Start()
	time.AfterFunc(10*time.Millisecond, cmgr.Stop)
	cmgr.Wait()
	wantMaxDials := uint32(75)
	if atomic.LoadUint32(&dials) > wantMaxDials {
		t.Fatalf("network failure: unexpected number of dials - got %v, want < %v",
			atomic.LoadUint32(&dials), wantMaxDials)
	}
}

//teststopfailed在connmgr为
//停止。
//
//我们有一个Dailer，它在conn管理器上设置停止标志，并返回一个
//错误，以便处理程序假定conn管理器已停止并忽略
//失败了。
func TestStopFailed(t *testing.T) {
	done := make(chan struct{}, 1)
	waitDialer := func(addr net.Addr) (net.Conn, error) {
		done <- struct{}{}
		time.Sleep(time.Millisecond)
		return nil, errors.New("network down")
	}
	cmgr, err := New(&Config{
		Dial: waitDialer,
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	cmgr.Start()
	go func() {
		<-done
		atomic.StoreInt32(&cmgr.stop, 1)
		time.Sleep(2 * time.Millisecond)
		atomic.StoreInt32(&cmgr.stop, 0)
		cmgr.Stop()
	}()
	cr := &ConnReq{
		Addr: &net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18555,
		},
		Permanent: true,
	}
	go cmgr.Connect(cr)
	cmgr.Wait()
}

//testremovePendingConnection测试是否可以取消挂起的
//连接，从connmgr中删除其内部状态。
func TestRemovePendingConnection(t *testing.T) {
//创建一个带有拨号程序实例的connmgr实例
//成功。
	wait := make(chan struct{})
	indefiniteDialer := func(addr net.Addr) (net.Conn, error) {
		<-wait
		return nil, fmt.Errorf("error")
	}
	cmgr, err := New(&Config{
		Dial: indefiniteDialer,
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	cmgr.Start()

//建立到我们选择的随机IP的连接请求。
	cr := &ConnReq{
		Addr: &net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18555,
		},
		Permanent: true,
	}
	go cmgr.Connect(cr)

	time.Sleep(10 * time.Millisecond)

	if cr.State() != ConnPending {
		t.Fatalf("pending request hasn't been registered, status: %v",
			cr.State())
	}

//上述请求实际上永远无法建立
//连接。所以我们会在完成之前取消它。
	cmgr.Remove(cr.ID())

	time.Sleep(10 * time.Millisecond)

//现在检查连接请求的状态，它应该读取
//失败的状态。
	if cr.State() != ConnCanceled {
		t.Fatalf("request wasn't canceled, status is: %v", cr.State())
	}

	close(wait)
	cmgr.Stop()
}

//TestCancelIgnoreDlayedConnection测试取消的连接请求将
//即使有未完成的重试，也不执行连接上的回调
//成功了。
func TestCancelIgnoreDelayedConnection(t *testing.T) {
	retryTimeout := 10 * time.Millisecond

//设置一个拨号程序，该拨号程序将继续返回错误，直到
//连接通道已发出信号，将立即尝试拨号。
//成功返回连接。
	connect := make(chan struct{})
	failingDialer := func(addr net.Addr) (net.Conn, error) {
		select {
		case <-connect:
			return mockDialer(addr)
		default:
		}

		return nil, fmt.Errorf("error")
	}

	connected := make(chan *ConnReq)
	cmgr, err := New(&Config{
		Dial:          failingDialer,
		RetryDuration: retryTimeout,
		OnConnection: func(c *ConnReq, conn net.Conn) {
			connected <- c
		},
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	cmgr.Start()
	defer cmgr.Stop()

//建立到我们选择的随机IP的连接请求。
	cr := &ConnReq{
		Addr: &net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18555,
		},
	}
	cmgr.Connect(cr)

//允许第一次重试超时时间过去。
	time.Sleep(2 * retryTimeout)

//连接被标记为失败，即使在重新尝试
//连接。
	if cr.State() != ConnFailing {
		t.Fatalf("failing request should have status failed, status: %v",
			cr.State())
	}

//移除连接，然后立即允许下一个连接
//成功。
	cmgr.Remove(cr.ID())
	close(connect)

//允许连接管理器处理删除。
	time.Sleep(5 * time.Millisecond)

//现在检查连接请求的状态，它应该读取
//已取消的状态。
	if cr.State() != ConnCanceled {
		t.Fatalf("request wasn't canceled, status is: %v", cr.State())
	}

//最后，连接管理器不应向ON连接发送信号
//回调，因为我们显式地取消了这个请求。我们给出一个
//慷慨的窗口确保连接管理器的客户机回退
//允许适当地消逝。
	select {
	case <-connected:
		t.Fatalf("on-connect should not be called for canceled req")
	case <-time.After(5 * retryTimeout):
	}

}

//MockListener实现net.Listener接口，用于测试
//处理net.Listener的代码，而不必实际生成任何
//连接。
type mockListener struct {
	localAddr   string
	provideConn chan net.Conn
}

//接受通过连接接收信号时返回模拟连接
//功能。
//
//这是NET.Listener接口的一部分。
func (m *mockListener) Accept() (net.Conn, error) {
	for conn := range m.provideConn {
		return conn, nil
	}
	return nil, errors.New("network connection closed")
}

//关闭关闭模拟侦听器，这将导致任何被阻止的接受
//要取消阻止的操作并返回错误。
//
//这是NET.Listener接口的一部分。
func (m *mockListener) Close() error {
	close(m.provideConn)
	return nil
}

//addr返回模拟侦听器配置的地址。
//
//这是NET.Listener接口的一部分。
func (m *mockListener) Addr() net.Addr {
	return &mockAddr{"tcp", m.localAddr}
}

//Connect伪造从提供的远程服务器到模拟侦听器的连接
//地址。它将导致accept函数返回模拟连接
//使用提供的远程地址和本地地址配置
//模拟听众。
func (m *mockListener) Connect(ip string, port int) {
	m.provideConn <- &mockConn{
		laddr: m.localAddr,
		lnet:  "tcp",
		rAddr: &net.TCPAddr{
			IP:   net.ParseIP(ip),
			Port: port,
		},
	}
}

//NewMockListener为提供的本地地址返回新的模拟侦听器
//和端口。实际上没有打开任何端口。
func newMockListener(localAddr string) *mockListener {
	return &mockListener{
		localAddr:   localAddr,
		provideConn: make(chan net.Conn),
	}
}

//测试侦听器确保向连接管理器提供侦听器
//接受回调可以正常工作。
func TestListeners(t *testing.T) {
//用几个模拟侦听器设置连接管理器
//当接收到模拟连接时通知通道。
	receivedConns := make(chan net.Conn)
	listener1 := newMockListener("127.0.0.1:8333")
	listener2 := newMockListener("127.0.0.1:9333")
	listeners := []net.Listener{listener1, listener2}
	cmgr, err := New(&Config{
		Listeners: listeners,
		OnAccept: func(conn net.Conn) {
			receivedConns <- conn
		},
		Dial: mockDialer,
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	cmgr.Start()

//假装和每个听众都有几个假连接。
	go func() {
		for i, listener := range listeners {
			l := listener.(*mockListener)
			l.Connect("127.0.0.1", 10000+i*2)
			l.Connect("127.0.0.1", 10000+i*2+1)
		}
	}()

//统计接收连接以确保预期的数目
//收到。另外，在超时后测试失败，因此它不会挂起
//测试永远不起作用。
	expectedNumConns := len(listeners) * 2
	var numConns int
out:
	for {
		select {
		case <-receivedConns:
			numConns++
			if numConns == expectedNumConns {
				break out
			}

		case <-time.After(time.Millisecond * 50):
			t.Fatalf("Timeout waiting for %d expected connections",
				expectedNumConns)
		}
	}

	cmgr.Stop()
	cmgr.Wait()
}
