
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2014-2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package rpcclient

import (
	"bytes"
	"container/list"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/go-socks/socks"
	"github.com/btcsuite/websocket"
)

var (
//errInvalidauth是一个错误，用于描述客户端
//无法进行身份验证，或者指定的终结点是
//不正确。
	ErrInvalidAuth = errors.New("authentication failure")

//errInvalidEndpoint是一个错误，用于描述
//WebSocket握手失败，具有指定的终结点。
	ErrInvalidEndpoint = errors.New("the endpoint either does not support " +
		"websockets or does not exist")

//errClientNotConnected是一个错误，用于描述
//已创建WebSocket客户端，但从未创建连接
//建立。此条件与errclientdisconnect不同，后者
//表示丢失的已建立连接。
	ErrClientNotConnected = errors.New("the client was never connected")

//errclientdisconnect是一个错误，用于描述
//客户端已断开与RPC服务器的连接。当
//未设置DisableAutoReconnect选项，任何未结期货
//当客户端断开连接时，将返回此错误
//任何新请求。
	ErrClientDisconnect = errors.New("the client has been disconnected")

//errclientshutdown是一个错误，用于描述
//客户端已关闭，或者正在关闭
//下来。当客户关闭时，任何未完成的期货都将
//与任何新请求一样返回此错误。
	ErrClientShutdown = errors.New("the client has been shutdown")

//errnotwebsocketclient是一个错误，用于描述
//当
//客户端已配置为在HTTP POST模式下运行。
	ErrNotWebsocketClient = errors.New("client is not configured for " +
		"websockets")

//errclientalreadyconnected是一个错误，用于描述
//由于WebSocket，无法建立新的客户端连接
//客户端已连接到RPC服务器。
	ErrClientAlreadyConnected = errors.New("websocket client has already " +
		"connected")
)

const (
//SendBufferSize是WebSocket发送通道的元素数
//可以在阻塞前排队。
	sendBufferSize = 50

//sendpostbufferSize是HTTP Post发送的元素数
//通道可以在阻塞前排队。
	sendPostBufferSize = 100

//ConnectionRetryInterval是介于
//自动重新连接到RPC服务器时重试。
	connectionRetryInterval = time.Second * 5
)

//sendpostdetails包含发送到RPC服务器的HTTP POST请求
//作为最初的json-rpc命令和一个在服务器上应答的通道
//响应结果。
type sendPostDetails struct {
	httpRequest *http.Request
	jsonRequest *jsonRequest
}

//JSONREQUEST保存有关用于正确
//检测、解释并发送回复。
type jsonRequest struct {
	id             uint64
	method         string
	cmd            interface{}
	marshalledJSON []byte
	responseChan   chan *response
}

//客户端表示允许轻松访问
//比特币RPC服务器上可用的各种RPC方法。每个包装器
//函数处理将传递和返回类型转换为和的详细信息
//来自JSON-RPC所需的基础JSON类型
//调用
//
//客户端提供同步（阻塞）和异步的每个RPC
//（非阻塞）窗体。异步形式基于
//futures where they return an instance of a type that promises to deliver the
//在将来某个时间调用的结果。在上调用Receive方法
//如果没有结果，返回的未来将一直阻塞，直到结果可用为止
//已经。
type Client struct {
id uint64 //原子的，所以必须保持64位对齐

//CONFIG保存与此客户端相关的连接配置。
	config *ConnConfig

//wsconn是不在HTTP Post中时的基础WebSocket连接
//模式。
	wsConn *websocket.Conn

//http client是在HTTP中运行时要使用的基础HTTP客户端
//后模式。
	httpClient *http.Client

//MTX是一个互斥体，用于保护对连接相关字段的访问。
	mtx sync.Mutex

//disconnected指示服务器是否已断开连接。
	disconnected bool

//RetryCount保留客户端尝试的次数
//重新连接到RPC服务器。
	retryCount int64

//通过ID跟踪命令及其响应通道。
	requestLock sync.Mutex
	requestMap  map[uint64]*list.Element
	requestList *list.List

//通知。
	ntfnHandlers  *NotificationHandlers
	ntfnStateLock sync.Mutex
	ntfnState     *notificationState

//网络基础设施。
	sendChan        chan []byte
	sendPostChan    chan *sendPostDetails
	connEstablished chan struct{}
	disconnect      chan struct{}
	shutdown        chan struct{}
	wg              sync.WaitGroup
}

//next id返回发送JSON-RPC消息时要使用的下一个ID。这个
//ID允许响应与每个
//JSON-RPC规范。通常客户的消费者不需要
//但是，如果正在创建和使用自定义请求，则调用此函数
//此函数用于确保ID在所有请求中都是唯一的
//被制造出来。
func (c *Client) NextID() uint64 {
	return atomic.AddUint64(&c.id, 1)
}

//addrequest将传递的jsonrequest与其ID相关联。这允许
//来自远程服务器的响应将解编为适当的类型
//并在接收时发送到指定的通道。
//
//如果客户端已开始关闭，则返回errclientshutdown
//未添加请求。
//
//此函数对于并发访问是安全的。
func (c *Client) addRequest(jReq *jsonRequest) error {
	c.requestLock.Lock()
	defer c.requestLock.Unlock()

//具有请求锁的关闭通道的非阻塞读取
//保持可避免将请求添加到客户端的内部数据中
//结构，如果客户端正在关闭（和
//尚未获取请求锁），或已完成关闭
//已经（响应每个未完成的请求
//errclientshutdown）。
	select {
	case <-c.shutdown:
		return ErrClientShutdown
	default:
	}

	element := c.requestList.PushBack(jReq)
	c.requestMap[jReq.id] = element
	return nil
}

//removeRequest返回并删除包含响应的jsonRequest
//与传递的ID或nil（如果存在）关联的通道和原始方法
//没有联系。
//
//此函数对于并发访问是安全的。
func (c *Client) removeRequest(id uint64) *jsonRequest {
	c.requestLock.Lock()
	defer c.requestLock.Unlock()

	element := c.requestMap[id]
	if element != nil {
		delete(c.requestMap, id)
		request := c.requestList.Remove(element).(*jsonRequest)
		return request
	}

	return nil
}

//removeAllRequests删除包含响应的所有json请求
//未完成请求的通道。
//
//必须在保持请求锁的情况下调用此函数。
func (c *Client) removeAllRequests() {
	c.requestMap = make(map[uint64]*list.Element)
	c.requestList.Init()
}

//trackregisteredntfns检查传递的命令以查看它是否是
//通知命令并更新使用的通知状态
//重新连接时自动重新建立注册通知。
func (c *Client) trackRegisteredNtfns(cmd interface{}) {
//如果调用方对通知不感兴趣，则不执行任何操作。
	if c.ntfnHandlers == nil {
		return
	}

	c.ntfnStateLock.Lock()
	defer c.ntfnStateLock.Unlock()

	switch bcmd := cmd.(type) {
	case *btcjson.NotifyBlocksCmd:
		c.ntfnState.notifyBlocks = true

	case *btcjson.NotifyNewTransactionsCmd:
		if bcmd.Verbose != nil && *bcmd.Verbose {
			c.ntfnState.notifyNewTxVerbose = true
		} else {
			c.ntfnState.notifyNewTx = true

		}

	case *btcjson.NotifySpentCmd:
		for _, op := range bcmd.OutPoints {
			c.ntfnState.notifySpent[op] = struct{}{}
		}

	case *btcjson.NotifyReceivedCmd:
		for _, addr := range bcmd.Addresses {
			c.ntfnState.notifyReceived[addr] = struct{}{}
		}
	}
}

type (
//inmessage是第一种未标记传入消息的类型
//进入。它同时支持请求（通知支持）和
//响应。部分未标记的消息是一个通知，如果
//嵌入的ID（来自响应）为零。否则，它是一个
//反应。
	inMessage struct {
		ID *float64 `json:"id"`
		*rawNotification
		*rawResponse
	}

//rawnotification是一个部分未标记的json-rpc通知。
	rawNotification struct {
		Method string            `json:"method"`
		Params []json.RawMessage `json:"params"`
	}

//rawresponse是部分未解析的JSON-RPC响应。为此
//为了有效（根据JSON-RPC1.0规范），ID不能为零。
	rawResponse struct {
		Result json.RawMessage   `json:"result"`
		Error  *btcjson.RPCError `json:"error"`
	}
)

//响应是JSON-RPC结果的原始字节，或者如果响应
//错误对象非空。
type response struct {
	result []byte
	err    error
}

//结果检查未解析响应是否包含非零错误，
//如果是，则返回未标记的btcjson.rpcerror（或取消标记错误）。
//如果响应不是错误，则请求的原始字节为
//返回以进一步取消对特定结果类型的显示。
func (r rawResponse) result() (result []byte, err error) {
	if r.Error != nil {
		return nil, r.Error
	}
	return r.Result, nil
}

//handleMessage是传入通知和响应的主要处理程序。
func (c *Client) handleMessage(msg []byte) {
//尝试将邮件取消标记为通知或
//反应。
	var in inMessage
	in.rawResponse = new(rawResponse)
	in.rawNotification = new(rawNotification)
	err := json.Unmarshal(msg, &in)
	if err != nil {
		log.Warnf("Remote server sent invalid message: %v", err)
		return
	}

//JSON-RPC1.0通知是ID为空的请求。
	if in.ID == nil {
		ntfn := in.rawNotification
		if ntfn == nil {
			log.Warn("Malformed notification: missing " +
				"method and parameters")
			return
		}
		if ntfn.Method == "" {
			log.Warn("Malformed notification: missing method")
			return
		}
//参数不是可选的：nil无效（但len==0是）
		if ntfn.Params == nil {
			log.Warn("Malformed notification: missing params")
			return
		}
//发出通知。
		log.Tracef("Received notification [%s]", in.Method)
		c.handleNotification(in.rawNotification)
		return
	}

//确保in.id可以转换为整数而不丢失精度
	if *in.ID < 0 || *in.ID != math.Trunc(*in.ID) {
		log.Warn("Malformed response: invalid identifier")
		return
	}

	if in.rawResponse == nil {
		log.Warn("Malformed response: missing result and error")
		return
	}

	id := uint64(*in.ID)
	log.Tracef("Received response for id %d (result %s)", id, in.Result)
	request := c.removeRequest(id)

//如果没有与此答复关联的请求，则无需执行其他操作。
	if request == nil || request.responseChan == nil {
		log.Warnf("Received unexpected reply: %s (id %d)", in.Result,
			id)
		return
	}

//由于命令成功，请检查它是否是
//通知，如果是，则将其添加到通知状态，以便
//可以在重新连接时自动重新建立。
	c.trackRegisteredNtfns(request.cmd)

//做出回应。
	result, err := in.rawResponse.result()
	request.responseChan <- &response{result: result, err: err}
}

//shouldLogreadError返回传递的错误是否为预期错误
//从wsinhandler的websocket连接中读取，
//应记录。
func (c *Client) shouldLogReadError(err error) bool {
//强制断开连接时不记录日志。
	select {
	case <-c.shutdown:
		return false
	default:
	}

//断开连接后不记录日志。
	if err == io.EOF {
		return false
	}
	if opErr, ok := err.(*net.OpError); ok && !opErr.Temporary() {
		return false
	}

	return true
}

//wsinhandler处理WebSocket连接的所有传入消息
//与客户端关联。它必须像野人一样运作。
func (c *Client) wsInHandler() {
out:
	for {
//一旦关闭通道
//关闭。在这里使用非阻塞选择，这样我们就可以通过
//否则。
		select {
		case <-c.shutdown:
			break out
		default:
		}

		_, msg, err := c.wsConn.ReadMessage()
		if err != nil {
//如果不是由于断开连接导致的，请记录错误。
			if c.shouldLogReadError(err) {
				log.Errorf("Websocket receive error from "+
					"%s: %v", c.config.Host, err)
			}
			break out
		}
		c.handleMessage(msg)
	}

//确保连接已关闭。
	c.Disconnect()
	c.wg.Done()
	log.Tracef("RPC client input handler done for %s", c.config.Host)
}

//disconnectchan返回当前断开通道的副本。渠道
//受客户端互斥体的读保护，并且在通道
//正在重新连接期间重新分配。
func (c *Client) disconnectChan() <-chan struct{} {
	c.mtx.Lock()
	ch := c.disconnect
	c.mtx.Unlock()
	return ch
}

//WSouth-Adle处理WebSoT连接的所有传出消息。它
//使用缓冲通道序列化输出消息，同时允许
//发送程序继续异步运行。它必须像野人一样运作。
func (c *Client) wsOutHandler() {
out:
	for {
//发送任何准备发送的消息，直到客户端
//断开关闭。
		select {
		case msg := <-c.sendChan:
			err := c.wsConn.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				c.Disconnect()
				break out
			}

		case <-c.disconnectChan():
			break out
		}
	}

//在退出前排空所有通道，这样就不会有任何东西等待。
//发送。
cleanup:
	for {
		select {
		case <-c.sendChan:
		default:
			break cleanup
		}
	}
	c.wg.Done()
	log.Tracef("RPC client output handler done for %s", c.config.Host)
}

//sendmessage使用
//WebSocket连接。它由一个缓冲通道支持，因此它不会
//阻止，直到发送通道满为止。
func (c *Client) sendMessage(marshalledJSON []byte) {
//如果断开连接，不要发送消息。
	select {
	case c.sendChan <- marshalledJSON:
	case <-c.disconnectChan():
		return
	}
}

//reregisterntfs创建并发送重新建立当前
//与客户端关联的通知状态。它应该只被调用
//通过ResendRequests函数重新连接时。
func (c *Client) reregisterNtfns() error {
//如果调用方对通知不感兴趣，则不执行任何操作。
	if c.ntfnHandlers == nil {
		return nil
	}

//为了避免在通知状态下为
//在下面发布的可能长时间运行的RPC的整个时间内，
//它的副本和工作从那个。
//
//另外，其他命令将同时运行，可以修改
//通知状态（当然不是处于锁定状态），其中
//同时在远程RPC服务器中注册它，这样可以防止重复
//注册。
	c.ntfnStateLock.Lock()
	stateCopy := c.ntfnState.Copy()
	c.ntfnStateLock.Unlock()

//如果需要，请重新注册notifyblocks。
	if stateCopy.notifyBlocks {
		log.Debugf("Reregistering [notifyblocks]")
		if err := c.NotifyBlocks(); err != nil {
			return err
		}
	}

//如果需要，请重新注册notifynewtransactions。
	if stateCopy.notifyNewTx || stateCopy.notifyNewTxVerbose {
		log.Debugf("Reregistering [notifynewtransactions] (verbose=%v)",
			stateCopy.notifyNewTxVerbose)
		err := c.NotifyNewTransactions(stateCopy.notifyNewTxVerbose)
		if err != nil {
			return err
		}
	}

//重新注册所有以前注册的notifyspeed的组合
//如果需要，在一个命令中输出点。
	nslen := len(stateCopy.notifySpent)
	if nslen > 0 {
		outpoints := make([]btcjson.OutPoint, 0, nslen)
		for op := range stateCopy.notifySpent {
			outpoints = append(outpoints, op)
		}
		log.Debugf("Reregistering [notifyspent] outpoints: %v", outpoints)
		if err := c.notifySpentInternal(outpoints).Receive(); err != nil {
			return err
		}
	}

//重新注册所有以前注册的
//如果需要，在一个命令中通知接收到的地址。
	nrlen := len(stateCopy.notifyReceived)
	if nrlen > 0 {
		addresses := make([]string, 0, nrlen)
		for addr := range stateCopy.notifyReceived {
			addresses = append(addresses, addr)
		}
		log.Debugf("Reregistering [notifyreceived] addresses: %v", addresses)
		if err := c.notifyReceivedInternal(addresses).Receive(); err != nil {
			return err
		}
	}

	return nil
}

//对于“长时间运行”的请求，ignoreresends是一组所有方法
//在重新连接时客户端不会重新发布。
var ignoreResends = map[string]struct{}{
	"rescan": {},
}

//ResendRequests重新发送客户端未完成的任何请求
//断开的。一旦客户端重新连接为
//a separate goroutine.
func (c *Client) resendRequests() {
//将通知状态设置为备份。如果出了什么问题，
//断开客户端连接。
	if err := c.reregisterNtfns(); err != nil {
		log.Warnf("Unable to re-establish notification state: %v", err)
		c.Disconnect()
		return
	}

//因为可以阻止发送，更多请求可能
//由调用者在重新发送时添加，复制所有
//需要立即重新发送并从副本开始工作的请求。这个
//还允许快速释放锁。
	c.requestLock.Lock()
	resendReqs := make([]*jsonRequest, 0, c.requestList.Len())
	var nextElem *list.Element
	for e := c.requestList.Front(); e != nil; e = nextElem {
		nextElem = e.Next()

		jReq := e.Value.(*jsonRequest)
		if _, ok := ignoreResends[jReq.method]; ok {
//如果重新连接时未发送请求，请将其删除。
//来自请求结构，因为没有答复
//预期。
			delete(c.requestMap, jReq.id)
			c.requestList.Remove(e)
		} else {
			resendReqs = append(resendReqs, jReq)
		}
	}
	c.requestLock.Unlock()

	for _, jReq := range resendReqs {
//如果客户端再次断开连接，则停止重新发送命令
//因为下一次重新连接将处理它们。
		if c.Disconnected() {
			return
		}

		log.Tracef("Sending command [%s] with id %d", jReq.method,
			jReq.id)
		c.sendMessage(jReq.marshalledJSON)
	}
}

//WSReconnectHandler侦听客户端断开连接并自动尝试
//要重新连接，请使用根据重试次数缩放的重试间隔。
//它还重新发送在客户端
//断开连接，因此断开/重新连接过程对
//呼叫者。当disableAutoReconnect配置
//选项被设置。
//
//此函数必须作为goroutine运行。
func (c *Client) wsReconnectHandler() {
out:
	for {
		select {
		case <-c.disconnect:
//断开时，通过故障恢复
//连接。

		case <-c.shutdown:
			break out
		}

	reconnect:
		for {
			select {
			case <-c.shutdown:
				break out
			default:
			}

			wsConn, err := dial(c.config)
			if err != nil {
				c.retryCount++
				log.Infof("Failed to connect to %s: %v",
					c.config.Host, err)

//按以下数字缩放重试间隔：
//重试，使后退到最大值
//1分钟。
				scaledInterval := connectionRetryInterval.Nanoseconds() * c.retryCount
				scaledDuration := time.Duration(scaledInterval)
				if scaledDuration > time.Minute {
					scaledDuration = time.Minute
				}
				log.Infof("Retrying connection to %s in "+
					"%s", c.config.Host, scaledDuration)
				time.Sleep(scaledDuration)
				continue reconnect
			}

			log.Infof("Reestablished connection to RPC server %s",
				c.config.Host)

//重置连接状态并向重新连接发送信号
//已经发生了。
			c.wsConn = wsConn
			c.retryCount = 0

			c.mtx.Lock()
			c.disconnect = make(chan struct{})
			c.disconnected = false
			c.mtx.Unlock()

//Start processing input and output for the
//新连接。
			c.start()

//在另一个goroutine中重新发出挂起的请求，因为
//发送可以阻止。
			go c.resendRequests()

//中断重新连接循环以等待
//再次断开。
			break reconnect
		}
	}
	c.wg.Done()
	log.Tracef("RPC client reconnect handler done for %s", c.config.Host)
}

//handlesEndPostMessage处理执行传递的HTTP请求，读取
//结果，将其解组，并将未解组的结果传递给
//提供响应通道。
func (c *Client) handleSendPostMessage(details *sendPostDetails) {
	jReq := details.jsonRequest
	log.Tracef("Sending command [%s] with id %d", jReq.method, jReq.id)
	httpResponse, err := c.httpClient.Do(details.httpRequest)
	if err != nil {
		jReq.responseChan <- &response{err: err}
		return
	}

//读取原始字节并关闭响应。
	respBytes, err := ioutil.ReadAll(httpResponse.Body)
	httpResponse.Body.Close()
	if err != nil {
		err = fmt.Errorf("error reading json reply: %v", err)
		jReq.responseChan <- &response{err: err}
		return
	}

//尝试将响应取消标记为常规JSON-RPC响应。
	var resp rawResponse
	err = json.Unmarshal(respBytes, &resp)
	if err != nil {
//当响应本身不是有效的JSON-RPC响应时
//返回一个错误，其中包括http状态代码和raw
//响应字节。
		err = fmt.Errorf("status code: %d, response: %q",
			httpResponse.StatusCode, string(respBytes))
		jReq.responseChan <- &response{err: err}
		return
	}

	res, err := resp.result()
	jReq.responseChan <- &response{result: res, err: err}
}

//sendposthandler在客户端运行时处理所有传出消息
//在HTTP POST模式下。它使用缓冲通道来序列化输出消息
//同时允许发送方继续异步运行。必须运行
//作为一个傀儡。
func (c *Client) sendPostHandler() {
out:
	for {
//发送任何准备发送的消息，直到关闭通道
//关闭。
		select {
		case details := <-c.sendPostChan:
			c.handleSendPostMessage(details)

		case <-c.shutdown:
			break out
		}
	}

//退出前排出所有等待通道，这样就不会有任何等待。
//左右发送。
cleanup:
	for {
		select {
		case details := <-c.sendPostChan:
			details.jsonRequest.responseChan <- &response{
				result: nil,
				err:    ErrClientShutdown,
			}

		default:
			break cleanup
		}
	}
	c.wg.Done()
	log.Tracef("RPC client send handler done for %s", c.config.Host)

}

//sendpostrequest使用
//与客户端关联的HTTP客户端。它由一个缓冲通道支持，
//所以在发送通道满之前它不会阻塞。
func (c *Client) sendPostRequest(httpReq *http.Request, jReq *jsonRequest) {
//关闭时不发送消息。
	select {
	case <-c.shutdown:
		jReq.responseChan <- &response{result: nil, err: ErrClientShutdown}
	default:
	}

	c.sendPostChan <- &sendPostDetails{
		jsonRequest: jReq,
		httpRequest: httpReq,
	}
}

//newFutureError返回一个新的未来结果通道，该通道已经具有
//在回复设置为nil的频道上传递了错误watin。这是有用的
//从各种异步函数中轻松返回错误。
func newFutureError(err error) chan *response {
	responseChan := make(chan *response, 1)
	responseChan <- &response{err: err}
	return responseChan
}

//ReceiveFuture从传递的FutureResult通道接收以提取
//回复或任何错误。检查的错误包括
//FutureResult和来自服务器的答复中的错误。这将阻止
//直到结果在通过的通道上可用。
func receiveFuture(f chan *response) ([]byte, error) {
//在返回的通道上等待响应。
	r := <-f
	return r.result, r.err
}

//sendpost通过发出http post将传递的请求发送到服务器
//使用提供的响应通道请求答复。通常是新的
//使用此方法时，将为每个命令打开和关闭连接，
//但是，底层HTTP客户机可能合并多个命令
//取决于包括远程服务器配置在内的几个因素。
func (c *Client) sendPost(jReq *jsonRequest) {
//向配置的RPC服务器生成请求。
	protocol := "http"
	if !c.config.DisableTLS {
		protocol = "https"
	}
url := protocol + "://“+c.config.host（主机）
	bodyReader := bytes.NewReader(jReq.marshalledJSON)
	httpReq, err := http.NewRequest("POST", url, bodyReader)
	if err != nil {
		jReq.responseChan <- &response{result: nil, err: err}
		return
	}
	httpReq.Close = true
	httpReq.Header.Set("Content-Type", "application/json")

//配置基本访问授权。
	httpReq.SetBasicAuth(c.config.User, c.config.Pass)

	log.Tracef("Sending command [%s] with id %d", jReq.method, jReq.id)
	c.sendPostRequest(httpReq, jReq)
}

//sendRequest使用
//为答复提供了响应通道。它处理WebSocket和HTTP
//开机自检模式取决于客户端的配置。
func (c *Client) sendRequest(jReq *jsonRequest) {
//根据是否使用
//是否以HTTP POST模式运行的客户端。在HTTP中运行时
//POST模式下，命令通过HTTP客户机发出。否则，
//该命令通过异步WebSocket通道发出。
	if c.config.HTTPPostMode {
		c.sendPost(jReq)
		return
	}

//检查WebSocket连接是否从未建立，
//在这种情况下，处理程序goroutines没有运行。
	select {
	case <-c.connEstablished:
	default:
		jReq.responseChan <- &response{err: ErrClientNotConnected}
		return
	}

//将请求添加到内部跟踪映射，以便
//可以正确检测远程服务器并将其路由到响应
//通道。然后通过WebSocket发送已封送的请求
//连接。
	if err := c.addRequest(jReq); err != nil {
		jReq.responseChan <- &response{err: err}
		return
	}
	log.Tracef("Sending command [%s] with id %d", jReq.method, jReq.id)
	c.sendMessage(jReq.marshalledJSON)
}

//sendcmd将传递的命令发送到关联的服务器并返回
//响应通道，在该通道中的某个时间点将发送答复
//未来。它处理WebSocket和HTTP Post模式，具体取决于
//客户端的配置。
func (c *Client) sendCmd(cmd interface{}) chan *response {
//获取与命令关联的方法。
	method, err := btcjson.CmdMethod(cmd)
	if err != nil {
		return newFutureError(err)
	}

//整理命令。
	id := c.NextID()
	marshalledJSON, err := btcjson.MarshalCmd(id, cmd)
	if err != nil {
		return newFutureError(err)
	}

//生成请求并将其与要响应的通道一起发送。
	responseChan := make(chan *response, 1)
	jReq := &jsonRequest{
		id:             id,
		method:         method,
		cmd:            cmd,
		marshalledJSON: marshalledJSON,
		responseChan:   responseChan,
	}
	c.sendRequest(jReq)

	return responseChan
}

//sendcmdandwait将传递的命令发送到关联的服务器，waities
//并返回其结果。它将返回错误
//如果有，则返回字段。
func (c *Client) sendCmdAndWait(cmd interface{}) (interface{}, error) {
//将命令封送到JSON-RPC，并将其发送到连接的服务器，以及
//在返回的通道上等待响应。
	return receiveFuture(c.sendCmd(cmd))
}

//disconnected返回服务器是否已断开连接。如果A
//WebSocket客户端已创建，但从未连接，这也会返回false。
func (c *Client) Disconnected() bool {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	select {
	case <-c.connEstablished:
		return c.disconnected
	default:
		return false
	}
}

//DoDisconnect断开与客户端关联的WebSocket，如果
//尚未断开连接。如果断开连接为
//不需要，或者客户端正在HTTP POST模式下运行。
//
//此函数对于并发访问是安全的。
func (c *Client) doDisconnect() bool {
	if c.config.HTTPPostMode {
		return false
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()

//如果已断开连接，则不执行任何操作。
	if c.disconnected {
		return false
	}

	log.Tracef("Disconnecting RPC client %s", c.config.Host)
	close(c.disconnect)
	if c.wsConn != nil {
		c.wsConn.Close()
	}
	c.disconnected = true
	return true
}

//DoShutdown关闭关闭通道并记录关闭，除非关闭
//is already in progress.  It will return false if the shutdown is not needed.
//
//此函数对于并发访问是安全的。
func (c *Client) doShutdown() bool {
//如果客户端已经在进程中，则忽略关闭请求
//关闭或已经关闭。
	select {
	case <-c.shutdown:
		return false
	default:
	}

	log.Tracef("Shutting down RPC client %s", c.config.Host)
	close(c.shutdown)
	return true
}

//断开连接断开与客户端关联的当前WebSocket。这个
//除非客户端
//使用DisableAutoReconnect标志创建。
//
//当客户端在HTTP POST模式下运行时，此函数不起作用。
func (c *Client) Disconnect() {
//如果已经断开连接或在HTTP POST模式下运行，则不执行任何操作。
	if !c.doDisconnect() {
		return
	}

	c.requestLock.Lock()
	defer c.requestLock.Unlock()

//在不自动重新连接的情况下操作时，将错误发送到任何挂起的
//请求并关闭客户端。
	if c.config.DisableAutoReconnect {
		for e := c.requestList.Front(); e != nil; e = e.Next() {
			req := e.Value.(*jsonRequest)
			req.responseChan <- &response{
				result: nil,
				err:    ErrClientDisconnect,
			}
		}
		c.removeAllRequests()
		c.doShutdown()
	}
}

//shutdown通过断开任何关联的连接关闭客户端
//当启用自动重新连接时，阻止将来
//尝试重新连接。它还可以阻止所有的Goroutine。
func (c *Client) Shutdown() {
//在请求锁下执行关闭操作，以防止客户端
//在启动客户端关闭进程时添加新请求。
	c.requestLock.Lock()
	defer c.requestLock.Unlock()

//如果客户端已经在进程中，则忽略关闭请求
//关闭或已经关闭。
	if !c.doShutdown() {
		return
	}

//将errclientshutdown错误发送到任何挂起的请求。
	for e := c.requestList.Front(); e != nil; e = e.Next() {
		req := e.Value.(*jsonRequest)
		req.responseChan <- &response{
			result: nil,
			err:    ErrClientShutdown,
		}
	}
	c.removeAllRequests()

//必要时断开客户机。
	c.doDisconnect()
}

//开始处理输入和输出消息。
func (c *Client) start() {
	log.Tracef("Starting RPC client %s", c.config.Host)

//根据客户机是否
//在HTTP POST模式或默认WebSocket模式下。
	if c.config.HTTPPostMode {
		c.wg.Add(1)
		go c.sendPostHandler()
	} else {
		c.wg.Add(3)
		go func() {
			if c.ntfnHandlers != nil {
				if c.ntfnHandlers.OnClientConnected != nil {
					c.ntfnHandlers.OnClientConnected()
				}
			}
			c.wg.Done()
		}()
		go c.wsInHandler()
		go c.wsOutHandler()
	}
}

//WaitForShutdown块，直到客户机Goroutines停止，并且
//连接已关闭。
func (c *Client) WaitForShutdown() {
	c.wg.Wait()
}

//ConnConfig描述客户端的连接配置参数。
//这个
type ConnConfig struct {
//主机是要连接的RPC服务器的IP地址和端口
//去。
	Host string

//Endpoint是RPC服务器上的WebSocket终结点。这是
//通常是“WS”。
	Endpoint string

//用户是用于对RPC服务器进行身份验证的用户名。
	User string

//pass是用于对RPC服务器进行身份验证的密码。
	Pass string

//disabletls指定传输层安全性是否应为
//残疾人。如果RPC服务器
//支持它，否则您的用户名和密码会被发送到
//明文中的电线。
	DisableTLS bool

//证书是用于PEM编码证书链的字节
//用于TLS连接。如果disabletls参数
//是真的。
	Certificates []byte

//代理指定通过SOCKS 5代理服务器进行连接。它可能
//如果不需要代理，则为空字符串。
	Proxy string

//proxyuser是代理服务器的可选用户名，如果
//需要身份验证。如果代理参数
//未设置。
	ProxyUser string

//proxypass是用于代理服务器的可选密码，如果
//需要身份验证。如果代理参数
//未设置。
	ProxyPass string

//DisableAutoReconnect指定客户端不应自动
//断开连接后，尝试重新连接到服务器。
	DisableAutoReconnect bool

//DisableConnectionNew指定WebSocket客户端连接
//使用new创建客户端时不应尝试。相反，
//客户端已创建并返回为未连接，并且connect必须
//手动调用。
	DisableConnectOnNew bool

//httpPostcode指示客户端使用多个独立的
//发出HTTP POST请求而不是使用默认值的连接
//网络插座。WebSockets通常作为
//客户端的这些通知的功能只适用于WebSockets，
//但是，并非所有服务器都支持WebSocket扩展，因此
//可以将标志设置为true以改用基本的HTTP POST请求。
	HTTPPostMode bool

//EnableBcInfohacks是一个选项，用于启用兼容性hacks
//连接到blockback.info RPC服务器时
	EnableBCInfoHacks bool
}

//new http client返回根据
//关联连接配置中的代理和TLS设置。
func newHTTPClient(config *ConnConfig) (*http.Client, error) {
//Set proxy function if there is a proxy configured.
	var proxyFunc func(*http.Request) (*url.URL, error)
	if config.Proxy != "" {
		proxyURL, err := url.Parse(config.Proxy)
		if err != nil {
			return nil, err
		}
		proxyFunc = http.ProxyURL(proxyURL)
	}

//根据需要配置TLS。
	var tlsConfig *tls.Config
	if !config.DisableTLS {
		if len(config.Certificates) > 0 {
			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM(config.Certificates)
			tlsConfig = &tls.Config{
				RootCAs: pool,
			}
		}
	}

	client := http.Client{
		Transport: &http.Transport{
			Proxy:           proxyFunc,
			TLSClientConfig: tlsConfig,
		},
	}

	return &client, nil
}

//拨号使用传递的连接配置打开WebSocket连接
//细节。
func dial(config *ConnConfig) (*websocket.Conn, error) {
//如果未禁用，则安装TLS。
	var tlsConfig *tls.Config
	var scheme = "ws"
	if !config.DisableTLS {
		tlsConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
		if len(config.Certificates) > 0 {
			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM(config.Certificates)
			tlsConfig.RootCAs = pool
		}
		scheme = "wss"
	}

//创建用于建立连接的WebSocket拨号程序。
//它可以根据需要通过下面的代理设置进行修改。
	dialer := websocket.Dialer{TLSClientConfig: tlsConfig}

//如果配置了代理，则设置代理。
	if config.Proxy != "" {
		proxy := &socks.Proxy{
			Addr:     config.Proxy,
			Username: config.ProxyUser,
			Password: config.ProxyPass,
		}
		dialer.NetDial = proxy.Dial
	}

//RPC服务器需要基本授权，因此创建自定义
//设置了授权头的请求头。
	login := config.User + ":" + config.Pass
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
	requestHeader := make(http.Header)
	requestHeader.Add("Authorization", auth)

//拨号连接。
url := fmt.Sprintf("%s://%s/%s“，方案，配置主机，配置终结点）
	wsConn, resp, err := dialer.Dial(url, requestHeader)
	if err != nil {
		if err != websocket.ErrBadHandshake || resp == nil {
			return nil, err
		}

//检测HTTP身份验证错误状态代码。
		if resp.StatusCode == http.StatusUnauthorized ||
			resp.StatusCode == http.StatusForbidden {
			return nil, ErrInvalidAuth
		}

//连接已通过身份验证，状态响应为
//好的，但是WebSocket握手仍然失败，因此端点
//在某些方面无效。
		if resp.StatusCode == http.StatusOK {
			return nil, ErrInvalidEndpoint
		}

//如果没有特殊的
//cases above apply.
		return nil, errors.New(resp.Status)
	}
	return wsConn, nil
}

//新建基于提供的连接配置创建新的RPC客户端
//细节。如果不是，通知处理程序参数可能为零
//有兴趣接收通知，如果
//配置设置为在HTTP POST模式下运行。
func New(config *ConnConfig, ntfnHandlers *NotificationHandlers) (*Client, error) {
//打开WebSocket连接或创建HTTP客户端，具体取决于
//在HTTP POST模式下。另外，将通知处理程序设置为nil
//在HTTP POST模式下运行时。
	var wsConn *websocket.Conn
	var httpClient *http.Client
	connEstablished := make(chan struct{})
	var start bool
	if config.HTTPPostMode {
		ntfnHandlers = nil
		start = true

		var err error
		httpClient, err = newHTTPClient(config)
		if err != nil {
			return nil, err
		}
	} else {
		if !config.DisableConnectOnNew {
			var err error
			wsConn, err = dial(config)
			if err != nil {
				return nil, err
			}
			start = true
		}
	}

	client := &Client{
		config:          config,
		wsConn:          wsConn,
		httpClient:      httpClient,
		requestMap:      make(map[uint64]*list.Element),
		requestList:     list.New(),
		ntfnHandlers:    ntfnHandlers,
		ntfnState:       newNotificationState(),
		sendChan:        make(chan []byte, sendBufferSize),
		sendPostChan:    make(chan *sendPostDetails, sendPostBufferSize),
		connEstablished: connEstablished,
		disconnect:      make(chan struct{}),
		shutdown:        make(chan struct{}),
	}

	if start {
		log.Infof("Established connection to RPC server %s",
			config.Host)
		close(connEstablished)
		client.start()
		if !client.config.HTTPPostMode && !client.config.DisableAutoReconnect {
			client.wg.Add(1)
			go client.wsReconnectHandler()
		}
	}

	return client, nil
}

//Connect建立初始WebSocket连接。这是必要的，当
//a client was created after setting the DisableConnectOnNew field of the
//配置结构。
//
//最多可尝试个连接数（每个连接在越来越多的回退之后）
//be tried if the connection can not be established.  The special value of 0
//表示连接尝试次数不受限制。
//
//如果没有为WebSockets配置客户端，则此方法将出错，如果
//连接已建立，或者如果没有连接
//尝试成功。
func (c *Client) Connect(tries int) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.config.HTTPPostMode {
		return ErrNotWebsocketClient
	}
	if c.wsConn != nil {
		return ErrClientAlreadyConnected
	}

//Begin connection attempts.  Increase the backoff after each failed
//尝试，最多一分钟。
	var err error
	var backoff time.Duration
	for i := 0; tries == 0 || i < tries; i++ {
		var wsConn *websocket.Conn
		wsConn, err = dial(c.config)
		if err != nil {
			backoff = connectionRetryInterval * time.Duration(i+1)
			if backoff > time.Minute {
				backoff = time.Minute
			}
			time.Sleep(backoff)
			continue
		}

//已建立连接。设置WebSocket连接
//客户的成员并启动必要的Goroutines
//运行客户端。
		log.Infof("Established connection to RPC server %s",
			c.config.Host)
		c.wsConn = wsConn
		close(c.connEstablished)
		c.start()
		if !c.config.DisableAutoReconnect {
			c.wg.Add(1)
			go c.wsReconnectHandler()
		}
		return nil
	}

//所有连接尝试都失败，因此返回上一个错误。
	return err
}
