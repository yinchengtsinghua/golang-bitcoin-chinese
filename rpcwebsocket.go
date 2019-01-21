
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2017 BTCSuite开发者
//版权所有（c）2015-2017法令开发商
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package main

import (
	"bytes"
	"container/list"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sync"
	"time"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/websocket"
	"golang.org/x/crypto/ripemd160"
)

const (
//WebSocketSendBufferSize是发送通道的元素数
//可以在阻塞前排队。请注意，这只适用于请求
//直接在WebSocket客户端输入处理程序或异步处理程序中处理
//处理程序，因为通知有自己的排队机制
//独立于发送通道缓冲区。
	websocketSendBufferSize = 50
)

type semaphore chan struct{}

func makeSemaphore(n int) semaphore {
	return make(chan struct{}, n)
}

func (s semaphore) acquire() { s <- struct{}{} }
func (s semaphore) release() { <-s }

//timezeroval只是一个时间的零值，用于避免
//创建多个实例。
var timeZeroVal time.Time

//wscommandhandler描述用于处理特定
//命令。
type wsCommandHandler func(*wsClient, interface{}) (interface{}, error)

//wshandlers将rpc命令字符串映射到适当的websocket处理程序
//功能。这是由init设置的，因为帮助引用了wshandler，因此
//导致依赖循环。
var wsHandlers map[string]wsCommandHandler
var wsHandlersBeforeInit = map[string]wsCommandHandler{
	"loadtxfilter":              handleLoadTxFilter,
	"help":                      handleWebsocketHelp,
	"notifyblocks":              handleNotifyBlocks,
	"notifynewtransactions":     handleNotifyNewTransactions,
	"notifyreceived":            handleNotifyReceived,
	"notifyspent":               handleNotifySpent,
	"session":                   handleSession,
	"stopnotifyblocks":          handleStopNotifyBlocks,
	"stopnotifynewtransactions": handleStopNotifyNewTransactions,
	"stopnotifyspent":           handleStopNotifySpent,
	"stopnotifyreceived":        handleStopNotifyReceived,
	"rescan":                    handleRescan,
	"rescanblocks":              handleRescanBlocks,
}

//WebSocketHandler通过创建新的wsclient来处理新的WebSocket客户端，
//启动它，然后阻塞直到连接关闭。因为它阻塞了，所以
//必须在单独的goroutine中运行。它应该从WebSocket调用
//在新的goroutine中运行每个新连接的服务器处理程序
//满足要求。
func (s *rpcServer) WebsocketHandler(conn *websocket.Conn, remoteAddr string,
	authenticated bool, isAdmin bool) {

//清除WebSocket被劫持之前设置的读取截止时间
//连接。
	conn.SetReadDeadline(timeZeroVal)

//限制WebSocket客户端的最大数量。
	rpcsLog.Infof("New websocket client %s", remoteAddr)
	if s.ntfnMgr.NumClients()+1 > cfg.RPCMaxWebsockets {
		rpcsLog.Infof("Max websocket clients exceeded [%d] - "+
			"disconnecting client %s", cfg.RPCMaxWebsockets,
			remoteAddr)
		conn.Close()
		return
	}

//创建新的WebSocket客户端以处理新的WebSocket连接
//等待它关闭。一旦停机（因此
//断开连接），删除它和它注册的任何通知。
	client, err := newWebsocketClient(s, conn, remoteAddr, authenticated, isAdmin)
	if err != nil {
		rpcsLog.Errorf("Failed to serve client %s: %v", remoteAddr, err)
		conn.Close()
		return
	}
	s.ntfnMgr.AddClient(client)
	client.Start()
	client.WaitForShutdown()
	s.ntfnMgr.RemoveClient(client)
	rpcsLog.Infof("Disconnected websocket client %s", remoteAddr)
}

//wsnotificationManager是用于
//WebSoCukes。它允许WebSocket客户端注册通知
//对…感兴趣。当代码中的其他地方发生事件时，例如
//正在添加到内存池或块连接/断开连接的事务，
//通知管理器提供了
//根据需要通知哪些WebSocket客户端
//已注册并通知他们。它也用来保存
//跟踪所有连接的WebSocket客户端。
type wsNotificationManager struct {
//服务器是与通知管理器关联的RPC服务器。
	server *rpcServer

//队列通知将处理通知排队。
	queueNotification chan interface{}

//通知MSGS向通知处理程序馈送通知
//以及来自队列的客户端（取消）注册请求
//来自客户端的注册和注销请求。
	notificationMsgs chan interface{}

//当前已连接客户端数的访问通道。
	numClients chan int

//停机处理
	wg   sync.WaitGroup
	quit chan struct{}
}

//QueueHandler管理一个空接口队列，从和中读取
//把最老的未发送的发送出去。当
//在或退出频道关闭，并在返回前关闭，没有
//正在等待发送仍留在队列中的任何变量。
func queueHandler(in <-chan interface{}, out chan<- interface{}, quit <-chan struct{}) {
	var q []interface{}
	var dequeue chan<- interface{}
	skipQueue := out
	var next interface{}
out:
	for {
		select {
		case n, ok := <-in:
			if !ok {
//发送器关闭输入通道。
				break out
			}

//如果skipqueue是
//非零（队列为空），读卡器已就绪，
//或者追加到队列，稍后发送。
			select {
			case skipQueue <- n:
			default:
				q = append(q, n)
				dequeue = out
				skipQueue = nil
				next = q[0]
			}

		case dequeue <- next:
			copy(q, q[1:])
q[len(q)-1] = nil //避免泄漏
			q = q[:len(q)-1]
			if len(q) == 0 {
				dequeue = nil
				skipQueue = out
			} else {
				next = q[0]
			}

		case <-quit:
			break out
		}
	}
	close(out)
}

//QueueHandler维护通知和通知处理程序的队列
//控制消息。
func (m *wsNotificationManager) queueHandler() {
	queueHandler(m.queueNotification, m.notificationMsgs, m.quit)
	m.wg.Done()
}

//notifyblockconnected传递新连接到最佳链的块
//到通知管理器进行块和事务通知
//处理。
func (m *wsNotificationManager) NotifyBlockConnected(block *btcutil.Block) {
//因为块管理器将调用notifyblockconnected
//并且RPC服务器可能不再运行，请使用选择
//语句取消阻止将通知排队一次RPC
//服务器已开始关闭。
	select {
	case m.queueNotification <- (*notificationBlockConnected)(block):
	case <-m.quit:
	}
}

//notifyblockdisconnected传递从最佳链断开的块
//到通知管理器进行块通知处理。
func (m *wsNotificationManager) NotifyBlockDisconnected(block *btcutil.Block) {
//因为块管理器将调用notifyblockdisconnected
//并且RPC服务器可能不再运行，请使用选择
//语句取消阻止将通知排队一次RPC
//服务器已开始关闭。
	select {
	case m.queueNotification <- (*notificationBlockDisconnected)(block):
	case <-m.quit:
	}
}

//notifymempooltx将mempool接受的事务传递给
//用于处理事务通知的通知管理器。如果
//is new是真的，tx是一个新事务，而不是一个
//在REORG期间添加到mempool。
func (m *wsNotificationManager) NotifyMempoolTx(tx *btcutil.Tx, isNew bool) {
	n := &notificationTxAcceptedByMempool{
		isNew: isNew,
		tx:    tx,
	}

//因为mempool和rpc服务器将调用notifymempooltx
//可能不再运行，请使用SELECT语句取消阻止
//在RPC服务器启动后将通知排队
//关闭。
	select {
	case m.queueNotification <- n:
	case <-m.quit:
	}
}

//wsclientfilter跟踪的每个WebSocket客户端的相关地址
//“rescanblocks”扩展名。它由“loadtxfilter”命令修改。
//
//注意：此扩展从github.com/decred/dcrd移植
type wsClientFilter struct {
	mu sync.Mutex

//实现了地址查找的快速路径。
	pubKeyHashes        map[[ripemd160.Size]byte]struct{}
	scriptHashes        map[[ripemd160.Size]byte]struct{}
	compressedPubKeys   map[[33]byte]struct{}
	uncompressedPubKeys map[[65]byte]struct{}

//如果不存在快速路径，则返回地址查找映射。
//只有完整性才存在。如果使用它出现在配置文件中，
//很有可能会添加一条快速路径。
	otherAddresses map[string]struct{}

//未消耗输出的输出点。
	unspent map[wire.OutPoint]struct{}
}

//new wsclientfilter创建要使用的新的空wsclientfilter结构
//对于WebSocket客户端。
//
//注意：此扩展从github.com/decred/dcrd移植
func newWSClientFilter(addresses []string, unspentOutPoints []wire.OutPoint, params *chaincfg.Params) *wsClientFilter {
	filter := &wsClientFilter{
		pubKeyHashes:        map[[ripemd160.Size]byte]struct{}{},
		scriptHashes:        map[[ripemd160.Size]byte]struct{}{},
		compressedPubKeys:   map[[33]byte]struct{}{},
		uncompressedPubKeys: map[[65]byte]struct{}{},
		otherAddresses:      map[string]struct{}{},
		unspent:             make(map[wire.OutPoint]struct{}, len(unspentOutPoints)),
	}

	for _, s := range addresses {
		filter.addAddressStr(s, params)
	}
	for i := range unspentOutPoints {
		filter.addUnspentOutPoint(&unspentOutPoints[i])
	}

	return filter
}

//addaddress将地址添加到wsclientfilter，并正确地处理它
//作为参数传递的地址类型。
//
//注意：此扩展从github.com/decred/dcrd移植
func (f *wsClientFilter) addAddress(a btcutil.Address) {
	switch a := a.(type) {
	case *btcutil.AddressPubKeyHash:
		f.pubKeyHashes[*a.Hash160()] = struct{}{}
		return
	case *btcutil.AddressScriptHash:
		f.scriptHashes[*a.Hash160()] = struct{}{}
		return
	case *btcutil.AddressPubKey:
		serializedPubKey := a.ScriptAddress()
		switch len(serializedPubKey) {
case 33: //压缩的
			var compressedPubKey [33]byte
			copy(compressedPubKey[:], serializedPubKey)
			f.compressedPubKeys[compressedPubKey] = struct{}{}
			return
case 65: //未压缩的
			var uncompressedPubKey [65]byte
			copy(uncompressedPubKey[:], serializedPubKey)
			f.uncompressedPubKeys[uncompressedPubKey] = struct{}{}
			return
		}
	}

	f.otherAddresses[a.EncodeAddress()] = struct{}{}
}

//addAddressStr解析字符串中的地址，然后将其添加到
//使用addaddress的wsclientfilter。
//
//注意：此扩展从github.com/decred/dcrd移植
func (f *wsClientFilter) addAddressStr(s string, params *chaincfg.Params) {
//如果地址不能解码，就没有必要保存它，因为它也应该
//无法从检查的事务输出创建地址
//脚本。
	a, err := btcutil.DecodeAddress(s, params)
	if err != nil {
		return
	}
	f.addAddress(a)
}

//如果已将传递的地址添加到
//WSCLC滤波器。
//
//注意：此扩展从github.com/decred/dcrd移植
func (f *wsClientFilter) existsAddress(a btcutil.Address) bool {
	switch a := a.(type) {
	case *btcutil.AddressPubKeyHash:
		_, ok := f.pubKeyHashes[*a.Hash160()]
		return ok
	case *btcutil.AddressScriptHash:
		_, ok := f.scriptHashes[*a.Hash160()]
		return ok
	case *btcutil.AddressPubKey:
		serializedPubKey := a.ScriptAddress()
		switch len(serializedPubKey) {
case 33: //压缩的
			var compressedPubKey [33]byte
			copy(compressedPubKey[:], serializedPubKey)
			_, ok := f.compressedPubKeys[compressedPubKey]
			if !ok {
				_, ok = f.pubKeyHashes[*a.AddressPubKeyHash().Hash160()]
			}
			return ok
case 65: //未压缩的
			var uncompressedPubKey [65]byte
			copy(uncompressedPubKey[:], serializedPubKey)
			_, ok := f.uncompressedPubKeys[uncompressedPubKey]
			if !ok {
				_, ok = f.pubKeyHashes[*a.AddressPubKeyHash().Hash160()]
			}
			return ok
		}
	}

	_, ok := f.otherAddresses[a.EncodeAddress()]
	return ok
}

//removeAddress将传递的地址（如果存在）从
//WSCLC滤波器。
//
//注意：此扩展从github.com/decred/dcrd移植
func (f *wsClientFilter) removeAddress(a btcutil.Address) {
	switch a := a.(type) {
	case *btcutil.AddressPubKeyHash:
		delete(f.pubKeyHashes, *a.Hash160())
		return
	case *btcutil.AddressScriptHash:
		delete(f.scriptHashes, *a.Hash160())
		return
	case *btcutil.AddressPubKey:
		serializedPubKey := a.ScriptAddress()
		switch len(serializedPubKey) {
case 33: //压缩的
			var compressedPubKey [33]byte
			copy(compressedPubKey[:], serializedPubKey)
			delete(f.compressedPubKeys, compressedPubKey)
			return
case 65: //未压缩的
			var uncompressedPubKey [65]byte
			copy(uncompressedPubKey[:], serializedPubKey)
			delete(f.uncompressedPubKeys, uncompressedPubKey)
			return
		}
	}

	delete(f.otherAddresses, a.EncodeAddress())
}

//removeAddressStr分析字符串中的地址，然后将其从
//使用removeAddress的wsclientfilter。
//
//注意：此扩展从github.com/decred/dcrd移植
func (f *wsClientFilter) removeAddressStr(s string, params *chaincfg.Params) {
	a, err := btcutil.DecodeAddress(s, params)
	if err == nil {
		f.removeAddress(a)
	} else {
		delete(f.otherAddresses, s)
	}
}

//addunspentoutpoint向wsclientfilter添加一个outpoint。
//
//注意：此扩展从github.com/decred/dcrd移植
func (f *wsClientFilter) addUnspentOutPoint(op *wire.OutPoint) {
	f.unspent[*op] = struct{}{}
}

//如果传递的输出点已添加到
//wsclientfilter。
//
//注意：此扩展从github.com/decred/dcrd移植
func (f *wsClientFilter) existsUnspentOutPoint(op *wire.OutPoint) bool {
	_, ok := f.unspent[*op]
	return ok
}

//removeUnpentOutpoint从
//WSCLC滤波器。
//
//注意：此扩展从github.com/decred/dcrd移植
func (f *wsClientFilter) removeUnspentOutPoint(op *wire.OutPoint) {
	delete(f.unspent, *op)
}

//通知类型
type notificationBlockConnected btcutil.Block
type notificationBlockDisconnected btcutil.Block
type notificationTxAcceptedByMempool struct {
	isNew bool
	tx    *btcutil.Tx
}

//通知控制请求
type notificationRegisterClient wsClient
type notificationUnregisterClient wsClient
type notificationRegisterBlocks wsClient
type notificationUnregisterBlocks wsClient
type notificationRegisterNewMempoolTxs wsClient
type notificationUnregisterNewMempoolTxs wsClient
type notificationRegisterSpent struct {
	wsc *wsClient
	ops []*wire.OutPoint
}
type notificationUnregisterSpent struct {
	wsc *wsClient
	op  *wire.OutPoint
}
type notificationRegisterAddr struct {
	wsc   *wsClient
	addrs []string
}
type notificationUnregisterAddr struct {
	wsc  *wsClient
	addr string
}

//通知处理程序从队列中读取通知和控制消息
//一次处理一个。
func (m *wsNotificationManager) notificationHandler() {
//客户端是所有当前连接的WebSocket客户端的映射。
	clients := make(map[chan struct{}]*wsClient)

//用于保存要通知的WebSocket客户端列表的映射
//某些事件。每个WebSocket客户端还保存事件的映射
//有多个触发器可以从这些列表中删除
//连接更紧密，价格更低。
//
//在可能的情况下，退出通道用作客户端的唯一ID
//因为它比使用整个结构更有效率。
	blockNotifications := make(map[chan struct{}]*wsClient)
	txNotifications := make(map[chan struct{}]*wsClient)
	watchedOutPoints := make(map[wire.OutPoint]map[chan struct{}]*wsClient)
	watchedAddrs := make(map[string]map[chan struct{}]*wsClient)

out:
	for {
		select {
		case n, ok := <-m.notificationMsgs:
			if !ok {
//队列处理程序退出。
				break out
			}
			switch n := n.(type) {
			case *notificationBlockConnected:
				block := (*btcutil.Block)(n)

//如果没有，跳过所有tx的迭代
//存在Tx通知请求。
				if len(watchedOutPoints) != 0 || len(watchedAddrs) != 0 {
					for _, tx := range block.Transactions() {
						m.notifyForTx(watchedOutPoints,
							watchedAddrs, tx, block)
					}
				}

				if len(blockNotifications) != 0 {
					m.notifyBlockConnected(blockNotifications,
						block)
					m.notifyFilteredBlockConnected(blockNotifications,
						block)
				}

			case *notificationBlockDisconnected:
				block := (*btcutil.Block)(n)

				if len(blockNotifications) != 0 {
					m.notifyBlockDisconnected(blockNotifications,
						block)
					m.notifyFilteredBlockDisconnected(blockNotifications,
						block)
				}

			case *notificationTxAcceptedByMempool:
				if n.isNew && len(txNotifications) != 0 {
					m.notifyForNewTx(txNotifications, n.tx)
				}
				m.notifyForTx(watchedOutPoints, watchedAddrs, n.tx, nil)
				m.notifyRelevantTxAccepted(n.tx, clients)

			case *notificationRegisterBlocks:
				wsc := (*wsClient)(n)
				blockNotifications[wsc.quit] = wsc

			case *notificationUnregisterBlocks:
				wsc := (*wsClient)(n)
				delete(blockNotifications, wsc.quit)

			case *notificationRegisterClient:
				wsc := (*wsClient)(n)
				clients[wsc.quit] = wsc

			case *notificationUnregisterClient:
				wsc := (*wsClient)(n)
//删除客户端发出的任何请求以及
//客户本身。
				delete(blockNotifications, wsc.quit)
				delete(txNotifications, wsc.quit)
				for k := range wsc.spentRequests {
					op := k
					m.removeSpentRequest(watchedOutPoints, wsc, &op)
				}
				for addr := range wsc.addrRequests {
					m.removeAddrRequest(watchedAddrs, wsc, addr)
				}
				delete(clients, wsc.quit)

			case *notificationRegisterSpent:
				m.addSpentRequests(watchedOutPoints, n.wsc, n.ops)

			case *notificationUnregisterSpent:
				m.removeSpentRequest(watchedOutPoints, n.wsc, n.op)

			case *notificationRegisterAddr:
				m.addAddrRequests(watchedAddrs, n.wsc, n.addrs)

			case *notificationUnregisterAddr:
				m.removeAddrRequest(watchedAddrs, n.wsc, n.addr)

			case *notificationRegisterNewMempoolTxs:
				wsc := (*wsClient)(n)
				txNotifications[wsc.quit] = wsc

			case *notificationUnregisterNewMempoolTxs:
				wsc := (*wsClient)(n)
				delete(txNotifications, wsc.quit)

			default:
				rpcsLog.Warn("Unhandled notification type")
			}

		case m.numClients <- len(clients):

		case <-m.quit:
//RPC服务器正在关闭。
			break out
		}
	}

	for _, c := range clients {
		c.Disconnect()
	}
	m.wg.Done()
}

//numclients返回正在提供服务的客户端数。
func (m *wsNotificationManager) NumClients() (n int) {
	select {
	case n = <-m.numClients:
case <-m.quit: //如果服务器已关闭，请使用默认的N（0）。
	}
	return
}

//RegisterBlockUpdates请求阻止对传递的更新通知
//WebSocket客户端。
func (m *wsNotificationManager) RegisterBlockUpdates(wsc *wsClient) {
	m.queueNotification <- (*notificationRegisterBlocks)(wsc)
}

//UnregisterBlockUpdates删除传递的块更新通知
//WebSocket客户端。
func (m *wsNotificationManager) UnregisterBlockUpdates(wsc *wsClient) {
	m.queueNotification <- (*notificationUnregisterBlocks)(wsc)
}

//subscribedclients返回所有WebSocket客户端退出通道的集合
//注册接收有关发送的通知，无论是由于发送
//花费被监视的输出或输出到被监视的地址。匹配
//根据此事务的输出和输出更新客户端的筛选器
//可能与客户有关的地址。
func (m *wsNotificationManager) subscribedClients(tx *btcutil.Tx,
	clients map[chan struct{}]*wsClient) map[chan struct{}]struct{} {

//使用客户机退出通道的映射作为密钥，以防止在
//多个输入和/或输出与客户相关。
	subscribed := make(map[chan struct{}]struct{})

	msgTx := tx.MsgTx()
	for _, input := range msgTx.TxIn {
		for quitChan, wsc := range clients {
			wsc.Lock()
			filter := wsc.filterData
			wsc.Unlock()
			if filter == nil {
				continue
			}
			filter.mu.Lock()
			if filter.existsUnspentOutPoint(&input.PreviousOutPoint) {
				subscribed[quitChan] = struct{}{}
			}
			filter.mu.Unlock()
		}
	}

	for i, output := range msgTx.TxOut {
		_, addrs, _, err := txscript.ExtractPkScriptAddrs(
			output.PkScript, m.server.cfg.ChainParams)
		if err != nil {
//客户端无法订阅
//非标准或非地址输出。
			continue
		}
		for quitChan, wsc := range clients {
			wsc.Lock()
			filter := wsc.filterData
			wsc.Unlock()
			if filter == nil {
				continue
			}
			filter.mu.Lock()
			for _, a := range addrs {
				if filter.existsAddress(a) {
					subscribed[quitChan] = struct{}{}
					op := wire.OutPoint{
						Hash:  *tx.Hash(),
						Index: uint32(i),
					}
					filter.addUnspentOutPoint(&op)
				}
			}
			filter.mu.Unlock()
		}
	}

	return subscribed
}

//NotifyBlockConnected通知已注册的WebSocket客户端
//当块连接到主链时，块会更新。
func (*wsNotificationManager) notifyBlockConnected(clients map[chan struct{}]*wsClient,
	block *btcutil.Block) {

//通知感兴趣的WebSocket客户端有关连接的块的信息。
	ntfn := btcjson.NewBlockConnectedNtfn(block.Hash().String(), block.Height(),
		block.MsgBlock().Header.Timestamp.Unix())
	marshalledJSON, err := btcjson.MarshalCmd(nil, ntfn)
	if err != nil {
		rpcsLog.Errorf("Failed to marshal block connected notification: "+
			"%v", err)
		return
	}
	for _, wsc := range clients {
		wsc.QueueNotification(marshalledJSON)
	}
}

//notifyblockdisconnected通知已注册的WebSocket客户端
//当块与主链断开连接时（由于
//重新组织）。
func (*wsNotificationManager) notifyBlockDisconnected(clients map[chan struct{}]*wsClient, block *btcutil.Block) {
//如果没有客户端请求块，则跳过通知创建
//已连接/已断开连接的通知。
	if len(clients) == 0 {
		return
	}

//将断开连接的块通知相关的WebSocket客户端。
	ntfn := btcjson.NewBlockDisconnectedNtfn(block.Hash().String(),
		block.Height(), block.MsgBlock().Header.Timestamp.Unix())
	marshalledJSON, err := btcjson.MarshalCmd(nil, ntfn)
	if err != nil {
		rpcsLog.Errorf("Failed to marshal block disconnected "+
			"notification: %v", err)
		return
	}
	for _, wsc := range clients {
		wsc.QueueNotification(marshalledJSON)
	}
}

//notifyfilteredblockconnected通知已注册的WebSocket客户端
//当块连接到主链时，块会更新。
func (m *wsNotificationManager) notifyFilteredBlockConnected(clients map[chan struct{}]*wsClient,
	block *btcutil.Block) {

//为通知创建相同的公共部分
//每个客户。
	var w bytes.Buffer
	err := block.MsgBlock().Header.Serialize(&w)
	if err != nil {
		rpcsLog.Errorf("Failed to serialize header for filtered block "+
			"connected notification: %v", err)
		return
	}
	ntfn := btcjson.NewFilteredBlockConnectedNtfn(block.Height(),
		hex.EncodeToString(w.Bytes()), nil)

//搜索每个客户的相关交易并保存它们
//以十六进制编码对通知进行序列化。
	subscribedTxs := make(map[chan struct{}][]string)
	for _, tx := range block.Transactions() {
		var txHex string
		for quitChan := range m.subscribedClients(tx, clients) {
			if txHex == "" {
				txHex = txHexString(tx.MsgTx())
			}
			subscribedTxs[quitChan] = append(subscribedTxs[quitChan], txHex)
		}
	}
	for quitChan, wsc := range clients {
//添加此客户端的所有发现的事务。为客户服务
//如果没有新样式的过滤器，则添加空字符串切片。
		ntfn.SubscribedTxs = subscribedTxs[quitChan]

//封送和队列通知。
		marshalledJSON, err := btcjson.MarshalCmd(nil, ntfn)
		if err != nil {
			rpcsLog.Errorf("Failed to marshal filtered block "+
				"connected notification: %v", err)
			return
		}
		wsc.QueueNotification(marshalledJSON)
	}
}

//notifyfilteredblockdisconnected通知已注册的WebSocket客户端
//当块与主链断开连接时（由于
//重新组织）。
func (*wsNotificationManager) notifyFilteredBlockDisconnected(clients map[chan struct{}]*wsClient,
	block *btcutil.Block) {
//如果没有客户端请求块，则跳过通知创建
//已连接/已断开连接的通知。
	if len(clients) == 0 {
		return
	}

//将断开连接的块通知相关的WebSocket客户端。
	var w bytes.Buffer
	err := block.MsgBlock().Header.Serialize(&w)
	if err != nil {
		rpcsLog.Errorf("Failed to serialize header for filtered block "+
			"disconnected notification: %v", err)
		return
	}
	ntfn := btcjson.NewFilteredBlockDisconnectedNtfn(block.Height(),
		hex.EncodeToString(w.Bytes()))
	marshalledJSON, err := btcjson.MarshalCmd(nil, ntfn)
	if err != nil {
		rpcsLog.Errorf("Failed to marshal filtered block disconnected "+
			"notification: %v", err)
		return
	}
	for _, wsc := range clients {
		wsc.QueueNotification(marshalledJSON)
	}
}

//RegisterNewMemPoolTxsUpdates请求向传递的WebSocket发送通知
//将新事务添加到内存池时的客户端。
func (m *wsNotificationManager) RegisterNewMempoolTxsUpdates(wsc *wsClient) {
	m.queueNotification <- (*notificationRegisterNewMempoolTxs)(wsc)
}

//UnregisterEmpoolTxSupdates删除对传递的WebSocket的通知
//将新事务添加到内存池时的客户端。
func (m *wsNotificationManager) UnregisterNewMempoolTxsUpdates(wsc *wsClient) {
	m.queueNotification <- (*notificationUnregisterNewMempoolTxs)(wsc)
}

//notifyfornewtx通知已注册更新的WebSocket客户端
//当新事务添加到内存池时。
func (m *wsNotificationManager) notifyForNewTx(clients map[chan struct{}]*wsClient, tx *btcutil.Tx) {
	txHashStr := tx.Hash().String()
	mtx := tx.MsgTx()

	var amount int64
	for _, txOut := range mtx.TxOut {
		amount += txOut.Value
	}

	ntfn := btcjson.NewTxAcceptedNtfn(txHashStr, btcutil.Amount(amount).ToBTC())
	marshalledJSON, err := btcjson.MarshalCmd(nil, ntfn)
	if err != nil {
		rpcsLog.Errorf("Failed to marshal tx notification: %s", err.Error())
		return
	}

	var verboseNtfn *btcjson.TxAcceptedVerboseNtfn
	var marshalledJSONVerbose []byte
	for _, wsc := range clients {
		if wsc.verboseTxUpdates {
			if marshalledJSONVerbose != nil {
				wsc.QueueNotification(marshalledJSONVerbose)
				continue
			}

			net := m.server.cfg.ChainParams
			rawTx, err := createTxRawResult(net, mtx, txHashStr, nil,
				"", 0, 0)
			if err != nil {
				return
			}

			verboseNtfn = btcjson.NewTxAcceptedVerboseNtfn(*rawTx)
			marshalledJSONVerbose, err = btcjson.MarshalCmd(nil,
				verboseNtfn)
			if err != nil {
				rpcsLog.Errorf("Failed to marshal verbose tx "+
					"notification: %s", err.Error())
				return
			}
			wsc.QueueNotification(marshalledJSONVerbose)
		} else {
			wsc.QueueNotification(marshalledJSON)
		}
	}
}

//当每个通过的
//确认输出点已花费（包含在连接到主机的块中）
//链）用于传递的WebSocket客户端。请求是自动的
//发送通知后删除。
func (m *wsNotificationManager) RegisterSpentRequests(wsc *wsClient, ops []*wire.OutPoint) {
	m.queueNotification <- &notificationRegisterSpent{
		wsc: wsc,
		ops: ops,
	}
}

//addSpentRequests将被监视输出点的映射修改为一组WebSocket
//要添加新请求的客户端监视ops中的所有输出点并创建
//并在花费到WebSocket客户端wsc时发送通知。
func (m *wsNotificationManager) addSpentRequests(opMap map[wire.OutPoint]map[chan struct{}]*wsClient,
	wsc *wsClient, ops []*wire.OutPoint) {

	for _, op := range ops {
//同时跟踪客户机中的请求，以便快速
//断开时拆下。
		wsc.spentRequests[*op] = struct{}{}

//将客户机添加到列表中，以便在看到输出点时发出通知。
//根据需要创建列表。
		cmap, ok := opMap[*op]
		if !ok {
			cmap = make(map[chan struct{}]*wsClient)
			opMap[*op] = cmap
		}
		cmap[wsc.quit] = wsc
	}

//检查是否有任何花费这些输出的事务已经存在于
//内存池，如果是，立即发送通知。
	spends := make(map[chainhash.Hash]*btcutil.Tx)
	for _, op := range ops {
		spend := m.server.cfg.TxMemPool.CheckSpend(*op)
		if spend != nil {
			rpcsLog.Debugf("Found existing mempool spend for "+
				"outpoint<%v>: %v", op, spend.Hash())
			spends[*spend.Hash()] = spend
		}
	}

	for _, spend := range spends {
		m.notifyForTx(opMap, nil, spend, nil)
	}
}

//UnregisterPendRequest从传递的WebSocket客户端删除请求
//当确认已通过的输出点已用完（包含在
//与主链相连的滑轮组）。
func (m *wsNotificationManager) UnregisterSpentRequest(wsc *wsClient, op *wire.OutPoint) {
	m.queueNotification <- &notificationUnregisterSpent{
		wsc: wsc,
		op:  op,
	}
}

//removePentRequest修改被监视输出点的映射以删除
//WebSocket客户机WSC，来自当
//被监视的前哨点被消耗掉。如果wsc是最后一个客户端，则输出点
//从地图中删除键。
func (*wsNotificationManager) removeSpentRequest(ops map[wire.OutPoint]map[chan struct{}]*wsClient,
	wsc *wsClient, op *wire.OutPoint) {

//从客户端删除请求跟踪。
	delete(wsc.spentRequests, *op)

//从要通知的列表中删除客户端。
	notifyMap, ok := ops[*op]
	if !ok {
		rpcsLog.Warnf("Attempt to remove nonexistent spent request "+
			"for websocket client %s", wsc.addr)
		return
	}
	delete(notifyMap, wsc.quit)

//如果有地图条目，请将其全部删除
//没有更多的客户对此感兴趣。
	if len(notifyMap) == 0 {
		delete(ops, *op)
	}
}

//TxHexString返回以十六进制编码的序列化事务。
func txHexString(tx *wire.MsgTx) string {
	buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
//忽略序列化错误，因为写入bytes.buffer不会失败。
	tx.Serialize(buf)
	return hex.EncodeToString(buf.Bytes())
}

//blockdetails创建要包含在btcws通知中的blockdetails结构
//来自块和事务的块索引。
func blockDetails(block *btcutil.Block, txIndex int) *btcjson.BlockDetails {
	if block == nil {
		return nil
	}
	return &btcjson.BlockDetails{
		Height: block.Height(),
		Hash:   block.Hash().String(),
		Index:  txIndex,
		Time:   block.MsgBlock().Header.Timestamp.Unix(),
	}
}

//new redeemingtx notification返回新的已封送redeemingtx通知
//传递的参数。
func newRedeemingTxNotification(txHex string, index int, block *btcutil.Block) ([]byte, error) {
//创建并封送通知。
	ntfn := btcjson.NewRedeemingTxNtfn(txHex, blockDetails(block, index))
	return btcjson.MarshalCmd(nil, ntfn)
}

//notifyfortxouts检查每个事务输出，通知感兴趣的
//如果输出花费在被监视的
//地址。已用通知请求将自动注册为
//每个匹配输出的客户端。
func (m *wsNotificationManager) notifyForTxOuts(ops map[wire.OutPoint]map[chan struct{}]*wsClient,
	addrs map[string]map[chan struct{}]*wsClient, tx *btcutil.Tx, block *btcutil.Block) {

//如果没有人在监听地址通知，则无需执行任何操作。
	if len(addrs) == 0 {
		return
	}

	txHex := ""
	wscNotified := make(map[chan struct{}]struct{})
	for i, txOut := range tx.MsgTx().TxOut {
		_, txAddrs, _, err := txscript.ExtractPkScriptAddrs(
			txOut.PkScript, m.server.cfg.ChainParams)
		if err != nil {
			continue
		}

		for _, txAddr := range txAddrs {
			cmap, ok := addrs[txAddr.EncodeAddress()]
			if !ok {
				continue
			}

			if txHex == "" {
				txHex = txHexString(tx.MsgTx())
			}
			ntfn := btcjson.NewRecvTxNtfn(txHex, blockDetails(block,
				tx.Index()))

			marshalledJSON, err := btcjson.MarshalCmd(nil, ntfn)
			if err != nil {
				rpcsLog.Errorf("Failed to marshal processedtx notification: %v", err)
				continue
			}

			op := []*wire.OutPoint{wire.NewOutPoint(tx.Hash(), uint32(i))}
			for wscQuit, wsc := range cmap {
				m.addSpentRequests(ops, wsc, op)

				if _, ok := wscNotified[wscQuit]; !ok {
					wscNotified[wscQuit] = struct{}{}
					wsc.QueueNotification(marshalledJSON)
				}
			}
		}
	}
}

//notifyrelevantxaccepted检查传递的
//事务，通知WebSocket客户端监视对象的输出开销
//地址和输入花费一个被监视的输出点。任何支付给
//被监视的地址会导致将来也会监视输出
//通知。
func (m *wsNotificationManager) notifyRelevantTxAccepted(tx *btcutil.Tx,
	clients map[chan struct{}]*wsClient) {

	clientsToNotify := m.subscribedClients(tx, clients)

	if len(clientsToNotify) != 0 {
		n := btcjson.NewRelevantTxAcceptedNtfn(txHexString(tx.MsgTx()))
		marshalled, err := btcjson.MarshalCmd(nil, n)
		if err != nil {
			rpcsLog.Errorf("Failed to marshal notification: %v", err)
			return
		}
		for quitChan := range clientsToNotify {
			clients[quitChan].QueueNotification(marshalled)
		}
	}
}

//notifyfortx检查已传递事务的输入和输出，
//通知WebSocket客户端将输出花费到被监视地址
//投入投入，投入，投入，投入，投入。
func (m *wsNotificationManager) notifyForTx(ops map[wire.OutPoint]map[chan struct{}]*wsClient,
	addrs map[string]map[chan struct{}]*wsClient, tx *btcutil.Tx, block *btcutil.Block) {

	if len(ops) != 0 {
		m.notifyForTxIns(ops, tx, block)
	}
	if len(addrs) != 0 {
		m.notifyForTxOuts(ops, addrs, tx, block)
	}
}

//notifyfortxins检查已传递事务的输入并发送
//感兴趣的websocket客户机如果有任何输入，将发出redeemingtx通知
//花费被监视的输出。如果块为非零，则任何匹配都将花费
//请求被删除。
func (m *wsNotificationManager) notifyForTxIns(ops map[wire.OutPoint]map[chan struct{}]*wsClient,
	tx *btcutil.Tx, block *btcutil.Block) {

//如果没有人在监视输出点，则无需采取任何措施。
	if len(ops) == 0 {
		return
	}

	txHex := ""
	wscNotified := make(map[chan struct{}]struct{})
	for _, txIn := range tx.MsgTx().TxIn {
		prevOut := &txIn.PreviousOutPoint
		if cmap, ok := ops[*prevOut]; ok {
			if txHex == "" {
				txHex = txHexString(tx.MsgTx())
			}
			marshalledJSON, err := newRedeemingTxNotification(txHex, tx.Index(), block)
			if err != nil {
				rpcsLog.Warnf("Failed to marshal redeemingtx notification: %v", err)
				continue
			}
			for wscQuit, wsc := range cmap {
				if block != nil {
					m.removeSpentRequest(ops, wsc, prevOut)
				}

				if _, ok := wscNotified[wscQuit]; !ok {
					wscNotified[wscQuit] = struct{}{}
					wsc.QueueNotification(marshalledJSON)
				}
			}
		}
	}
}

//RegisterTxOutAddressRequests向传递的WebSocket请求通知
//当事务输出花费到传递的地址时的客户端。
func (m *wsNotificationManager) RegisterTxOutAddressRequests(wsc *wsClient, addrs []string) {
	m.queueNotification <- &notificationRegisterAddr{
		wsc:   wsc,
		addrs: addrs,
	}
}

//AddAddrRequests将WebSocket客户端wsc添加到客户端集的地址
//addrmap，以便将任何mempool或block事务输出通知wsc
//花费到addrs中的任何地址。
func (*wsNotificationManager) addAddrRequests(addrMap map[string]map[chan struct{}]*wsClient,
	wsc *wsClient, addrs []string) {

	for _, addr := range addrs {
//同时跟踪客户机中的请求，以便
//断开时拆下。
		wsc.addrRequests[addr] = struct{}{}

//将客户端添加到要在
//可以看到前哨点。根据需要创建地图。
		cmap, ok := addrMap[addr]
		if !ok {
			cmap = make(map[chan struct{}]*wsClient)
			addrMap[addr] = cmap
		}
		cmap[wsc.quit] = wsc
	}
}

//UnregisterXoutAddressRequest从传递的WebSocket中删除请求
//当事务花费到传递的地址时要通知的客户端。
func (m *wsNotificationManager) UnregisterTxOutAddressRequest(wsc *wsClient, addr string) {
	m.queueNotification <- &notificationUnregisterAddr{
		wsc:  wsc,
		addr: addr,
	}
}

//removeAddrRequest将WebSocket客户端wsc从地址中删除到
//客户端设置加法器，使其不再接收的通知更新
//任何事务输出发送到地址。
func (*wsNotificationManager) removeAddrRequest(addrs map[string]map[chan struct{}]*wsClient,
	wsc *wsClient, addr string) {

//从客户端删除请求跟踪。
	delete(wsc.addrRequests, addr)

//从要通知的列表中删除客户端。
	cmap, ok := addrs[addr]
	if !ok {
		rpcsLog.Warnf("Attempt to remove nonexistent addr request "+
			"<%s> for websocket client %s", addr, wsc.addr)
		return
	}
	delete(cmap, wsc.quit)

//如果没有更多的客户机，请完全删除映射项
//对它感兴趣。
	if len(cmap) == 0 {
		delete(addrs, addr)
	}
}

//AddClient将传递的WebSocket客户端添加到通知管理器。
func (m *wsNotificationManager) AddClient(wsc *wsClient) {
	m.queueNotification <- (*notificationRegisterClient)(wsc)
}

//removeclient删除传递的WebSocket客户端和所有通知
//已注册。
func (m *wsNotificationManager) RemoveClient(wsc *wsClient) {
	select {
	case m.queueNotification <- (*notificationUnregisterClient)(wsc):
	case <-m.quit:
	}
}

//Start启动管理器排队和处理所需的goroutines
//WebSocket客户端通知。
func (m *wsNotificationManager) Start() {
	m.wg.Add(2)
	go m.queueHandler()
	go m.notificationHandler()
}

//等待关闭块，直到所有通知管理器goroutine
//完成了。
func (m *wsNotificationManager) WaitForShutdown() {
	m.wg.Wait()
}

//shutdown关闭管理器，停止通知队列和
//通知处理程序goroutines。
func (m *wsNotificationManager) Shutdown() {
	close(m.quit)
}

//newwsnotificationmanager返回一个新的通知管理器，可以使用。
//有关详细信息，请参阅wsnotificationmanager。
func newWsNotificationManager(server *rpcServer) *wsNotificationManager {
	return &wsNotificationManager{
		server:            server,
		queueNotification: make(chan interface{}),
		notificationMsgs:  make(chan interface{}),
		numClients:        make(chan int),
		quit:              make(chan struct{}),
	}
}

//wsresponse包含一条消息，以作为发送到连接的WebSocket客户端
//以及在发送消息时进行答复的通道。
type wsResponse struct {
	msg      []byte
	doneChan chan bool
}

//wsclient提供用于处理WebSocket客户机的抽象。这个
//总体数据流分为3个主要Goroutine，可能是第4个Goroutine
//对于长时间运行的操作（仅在发出请求时启动），以及
//用于允许广播等内容的WebSocket管理器
//已向所有连接的WebSocket客户端请求通知。入站
//消息通过inhandler goroutine读取，通常发送到
//他们自己的管理者。但是，某些可能长期运行的操作，如
//作为重新扫描，发送到Asynchander Goroutine，并且在
//时间。有两种出站消息类型-一种用于响应客户端
//请求和另一个异步通知。对客户端请求的响应
//使用使用缓冲通道的sendmessage，从而限制数字
//可以提出的未完成请求。通知通过发送
//通过notificationQueueHandler实现队列的QueueNotification
//确保无法阻止从其他子系统发送通知。最终，
//所有消息都通过outhandler发送。
type wsClient struct {
	sync.Mutex

//服务器是为客户端提供服务的RPC服务器。
	server *rpcServer

//conn是基础WebSocket连接。
	conn *websocket.Conn

//已断开连接指示WebSocket客户端是否
//断开的。
	disconnected bool

//addr是客户端的远程地址。
	addr string

//authenticated指定客户端是否已通过身份验证
//因此允许通过WebSocket进行通信。
	authenticated bool

//isadmin指定客户端是否可以更改服务器的状态；
//false意味着它只能访问有限的一组RPC调用。
	isAdmin bool

//sessionid是在连接时为每个客户机生成的随机ID。
//客户机可以使用会话RPC查询这些ID。一个变化
//到会话ID表示客户端已重新连接。
	sessionID uint64

//verbosetxupdates指定客户端是否已请求verbose
//有关所有新交易的信息。
	verboseTxUpdates bool

//AddrRequests是调用方请求的一组地址。
//已通知。在这里维护，以便删除所有请求
//当钱包断开时。属于通知管理器。
	addrRequests map[string]struct{}

//SpentRequests是钱包请求的一组未使用的输出点。
//已处理事务占用它们的时间通知。
//属于通知管理器。
	spentRequests map[wire.OutPoint]struct{}

//filterdata是新一代事务筛选器，从
//github.com/decred/dcrd，用于新的后台端口“loadtxfilter”和
//`RescanBlocks`方法。
	filterData *wsClientFilter

//网络基础设施。
	serviceRequestSem semaphore
	ntfnChan          chan []byte
	sendChan          chan wsResponse
	quit              chan struct{}
	wg                sync.WaitGroup
}

//inhandler处理WebSocket连接的所有传入消息。它
//必须作为goroutine运行。
func (c *wsClient) inHandler() {
out:
	for {
//退出通道关闭后，退出循环。
//在这里使用非阻塞选择，否则我们会失败。
		select {
		case <-c.quit:
			break out
		default:
		}

		_, msg, err := c.conn.ReadMessage()
		if err != nil {
//如果不是由于断开连接导致的，请记录错误。
			if err != io.EOF {
				rpcsLog.Errorf("Websocket receive error from "+
					"%s: %v", c.addr, err)
			}
			break out
		}

		var request btcjson.Request
		err = json.Unmarshal(msg, &request)
		if err != nil {
			if !c.authenticated {
				break out
			}

			jsonErr := &btcjson.RPCError{
				Code:    btcjson.ErrRPCParse.Code,
				Message: "Failed to parse request: " + err.Error(),
			}
			reply, err := createMarshalledReply(nil, nil, jsonErr)
			if err != nil {
				rpcsLog.Errorf("Failed to marshal parse failure "+
					"reply: %v", err)
				continue
			}
			c.SendMessage(reply, nil)
			continue
		}

//json-rpc 1.0规范定义通知必须具有其“id”
//设置为空并声明通知没有响应。
//
//json-rpc 2.0通知是带有“json-rpc”：“2.0”的请求，并且
//没有“id”成员。规范声明通知
//不能响应。json-rpc 2.0允许空值作为
//有效的请求ID，因此此类请求不是通知。
//
//比特币核心以“id”服务请求：空甚至缺少“id”，
//并以“id”响应这些请求：响应中为空。
//
//btcd不响应没有和“id”或“id”的任何请求：空，
//无论指定的JSON-RPC协议版本如何，除非RPC要求
//启用。启用rpc quirk后，将响应此类请求
//如果request不指示json-rpc版本，则返回。
//
//用户可以启用rpc-quirk以避免兼容性问题
//软件依赖核心的行为。
		if request.ID == nil && !(cfg.RPCQuirks && request.Jsonrpc == "") {
			if !c.authenticated {
				break out
			}
			continue
		}

		cmd := parseCmd(&request)
		if cmd.err != nil {
			if !c.authenticated {
				break out
			}

			reply, err := createMarshalledReply(cmd.id, nil, cmd.err)
			if err != nil {
				rpcsLog.Errorf("Failed to marshal parse failure "+
					"reply: %v", err)
				continue
			}
			c.SendMessage(reply, nil)
			continue
		}
		rpcsLog.Debugf("Received command <%s> from %s", cmd.method, c.addr)

//检查AUTH。如果
//未经授权的WebSocket客户端的第一个请求不是
//认证请求，收到认证请求
//当客户端已经过身份验证或不正确时
//请求中提供了身份验证凭据。
		switch authCmd, ok := cmd.cmd.(*btcjson.AuthenticateCmd); {
		case c.authenticated && ok:
			rpcsLog.Warnf("Websocket client %s is already authenticated",
				c.addr)
			break out
		case !c.authenticated && !ok:
			rpcsLog.Warnf("Unauthenticated websocket message " +
				"received")
			break out
		case !c.authenticated:
//检查凭证。
			login := authCmd.Username + ":" + authCmd.Passphrase
			auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
			authSha := sha256.Sum256([]byte(auth))
			cmp := subtle.ConstantTimeCompare(authSha[:], c.server.authsha[:])
			limitcmp := subtle.ConstantTimeCompare(authSha[:], c.server.limitauthsha[:])
			if cmp != 1 && limitcmp != 1 {
				rpcsLog.Warnf("Auth failure.")
				break out
			}
			c.authenticated = true
			c.isAdmin = cmp == 1

//整理并发送响应。
			reply, err := createMarshalledReply(cmd.id, nil, nil)
			if err != nil {
				rpcsLog.Errorf("Failed to marshal authenticate reply: "+
					"%v", err.Error())
				continue
			}
			c.SendMessage(reply, nil)
			continue
		}

//检查客户端是否使用有限的RPC凭据和
//未授权调用此RPC时出错。
		if !c.isAdmin {
			if _, ok := rpcLimited[request.Method]; !ok {
				jsonErr := &btcjson.RPCError{
					Code:    btcjson.ErrRPCInvalidParams.Code,
					Message: "limited user not authorized for this method",
				}
//整理并发送响应。
				reply, err := createMarshalledReply(request.ID, nil, jsonErr)
				if err != nil {
					rpcsLog.Errorf("Failed to marshal parse failure "+
						"reply: %v", err)
					continue
				}
				c.SendMessage(reply, nil)
				continue
			}
		}

//异步处理请求。信号量用于
//限制当前正在进行的并发请求数
//服务的。如果无法获取信号量，只需等待
//直到一个请求在读取下一个RPC请求之前完成
//来自WebSocket客户端。
//
//通过超时和出错，这可能有点夸张。
//当服务请求时间太长时，但如果是
//完成后，下一个请求的读取不应被阻止
//此信号量，否则将读取下一个请求并
//可能会在计时前再坐几秒钟
//也出来了。这将导致
//以后的请求要比这里的支票长得多
//暗示。
//
//如果添加了超时，则信号量获取应该是
//使用select语句移动到新goroutine的内部
//也可以读取一个时间。在频道之后。这将取消阻止
//从WebSocket客户端读取下一个请求并允许
//许多请求需要同时等待。
		c.serviceRequestSem.acquire()
		go func() {
			c.serviceRequest(cmd)
			c.serviceRequestSem.release()
		}()
	}

//确保连接已关闭。
	c.Disconnect()
	c.wg.Done()
	rpcsLog.Tracef("Websocket client input handler done for %s", c.addr)
}

//ServiceRequest通过查找和执行
//适当的RPC处理程序。响应被编组并发送到
//WebSocket客户端。
func (c *wsClient) serviceRequest(r *parsedRPCCmd) {
	var (
		result interface{}
		err    error
	)

//查找命令的WebSocket扩展，如果不查找
//exist回退以将命令作为标准命令处理。
	wsHandler, ok := wsHandlers[r.method]
	if ok {
		result, err = wsHandler(c, r.cmd)
	} else {
		result, err = c.server.standardCmdResult(r, nil)
	}
	reply, err := createMarshalledReply(r.id, result, err)
	if err != nil {
		rpcsLog.Errorf("Failed to marshal reply for <%s> "+
			"command: %v", r.method, err)
		return
	}
	c.SendMessage(reply, nil)
}

//通知队列处理程序处理的传出通知队列
//WebSocket客户端。它作为muxer运行，用于各种输入源
//确保要发送的排队通知不会阻塞。否则，
//缓慢的客户机可能会使其他系统（如mempool或block）陷入困境
//管理器）正在排队处理数据。将数据传递给OutHandler
//实际上是书面的。它必须像野人一样运作。
func (c *wsClient) notificationQueueHandler() {
ntfnSentChan := make(chan bool, 1) //非阻塞同步

//PendingNTFns用作准备就绪的通知的队列
//在当前没有未完成通知时发送
//发送。等待标志用于检查
//待处理列表，以确保清除知道已发送和未发送的内容
//到Outhandler。但是，目前不需要特别清理。
//如果在
//未来，不知道什么已经和没有被发送到outhandler
//（因此，谁应该对“完成”频道作出回应）将是
//没有使用这种方法就有问题。
	pendingNtfns := list.New()
	waiting := false
out:
	for {
		select {
//当消息排队时，此通道会得到通知。
//通过网络套接字发送。它将发送
//如果发送尚未进行，则立即发送消息，或者
//将要发送的消息排队，直到其他挂起的消息
//发送。
		case msg := <-c.ntfnChan:
			if !waiting {
				c.SendMessage(msg, ntfnSentChan)
			} else {
				pendingNtfns.PushBack(msg)
			}
			waiting = true

//发送通知后会通知此频道
//通过网络插座。
		case <-ntfnSentChan:
//如果中没有更多邮件，则不再等待
//挂起的消息队列。
			next := pendingNtfns.Front()
			if next == nil {
				waiting = false
				continue
			}

//通知OutHandler下一项
//异步发送。
			msg := pendingNtfns.Remove(next).([]byte)
			c.SendMessage(msg, ntfnSentChan)

		case <-c.quit:
			break out
		}
	}

//退出前排出所有等待通道，这样就不会有任何等待。
//左右发送。
cleanup:
	for {
		select {
		case <-c.ntfnChan:
		case <-ntfnSentChan:
		default:
			break cleanup
		}
	}
	c.wg.Done()
	rpcsLog.Tracef("Websocket client notification queue handler done "+
		"for %s", c.addr)
}

//outhandler处理WebSocket连接的所有传出消息。它
//必须作为goroutine运行。它使用缓冲通道来序列化输出
//同时允许发件人继续异步运行的消息。它
//必须作为goroutine运行。
func (c *wsClient) outHandler() {
out:
	for {
//发送任何准备发送的消息，直到退出通道
//关闭。
		select {
		case r := <-c.sendChan:
			err := c.conn.WriteMessage(websocket.TextMessage, r.msg)
			if err != nil {
				c.Disconnect()
				break out
			}
			if r.doneChan != nil {
				r.doneChan <- true
			}

		case <-c.quit:
			break out
		}
	}

//退出前排出所有等待通道，这样就不会有任何等待。
//左右发送。
cleanup:
	for {
		select {
		case r := <-c.sendChan:
			if r.doneChan != nil {
				r.doneChan <- false
			}
		default:
			break cleanup
		}
	}
	c.wg.Done()
	rpcsLog.Tracef("Websocket client output handler done for %s", c.addr)
}

//sendmessage将传递的JSON发送到WebSocket客户端。它是后盾
//通过缓冲通道，因此在发送通道满之前它不会阻塞。
//但是请注意，必须使用queuenotification发送异步
//通知而不是此函数。这种方法允许
//客户端可以在不阻止或
//阻止异步通知。
func (c *wsClient) SendMessage(marshalledJSON []byte, doneChan chan bool) {
//如果断开连接，不要发送消息。
	if c.Disconnected() {
		if doneChan != nil {
			doneChan <- false
		}
		return
	}

	c.sendChan <- wsResponse{msg: marshalledJSON, doneChan: doneChan}
}

//errclientquit描述了由于以下原因未处理客户端发送的错误：
//已断开或删除到客户端。
var ErrClientQuit = errors.New("client quit")

//queuenotification将要发送到WebSocket的传递通知排队
//客户端。顾名思义，此函数仅用于
//通知，因为它有额外的逻辑来阻止其他子系统，例如
//作为内存池和块管理器，即使在发送
//频道已满。
//
//如果客户端正在关闭，则此函数返回
//errclientquit。这将通过长时间运行的通知进行检查。
//如果不需要再做任何工作，则停止处理的处理程序。
func (c *wsClient) QueueNotification(marshalledJSON []byte) error {
//如果断开连接，不要将消息排队。
	if c.Disconnected() {
		return ErrClientQuit
	}

	c.ntfnChan <- marshalledJSON
	return nil
}

//disconnected返回WebSocket客户端是否已断开连接。
func (c *wsClient) Disconnected() bool {
	c.Lock()
	isDisconnected := c.disconnected
	c.Unlock()

	return isDisconnected
}

//断开连接断开WebSocket客户端。
func (c *wsClient) Disconnect() {
	c.Lock()
	defer c.Unlock()

//如果已断开连接，则不执行任何操作。
	if c.disconnected {
		return
	}

	rpcsLog.Tracef("Disconnecting websocket client %s", c.addr)
	close(c.quit)
	c.conn.Close()
	c.disconnected = true
}

//开始处理输入和输出消息。
func (c *wsClient) Start() {
	rpcsLog.Tracef("Starting websocket client %s", c.addr)

//开始处理输入和输出。
	c.wg.Add(3)
	go c.inHandler()
	go c.notificationQueueHandler()
	go c.outHandler()
}

//WaitForShutdown块，直到WebSocket客户端goroutine停止
//连接已关闭。
func (c *wsClient) WaitForShutdown() {
	c.wg.Wait()
}

//newWebSocketClient返回给定通知的新WebSocket客户端
//管理器、WebSocket连接、远程地址以及客户端
//已经过身份验证（通过HTTP基本访问身份验证）。这个
//返回的客户端已准备好启动。一旦启动，客户端将处理
//单独goroutine中的传入和传出消息，以及队列
//以及对长时间运行的操作的异步处理。
func newWebsocketClient(server *rpcServer, conn *websocket.Conn,
	remoteAddr string, authenticated bool, isAdmin bool) (*wsClient, error) {

	sessionID, err := wire.RandomUint64()
	if err != nil {
		return nil, err
	}

	client := &wsClient{
		conn:              conn,
		addr:              remoteAddr,
		authenticated:     authenticated,
		isAdmin:           isAdmin,
		sessionID:         sessionID,
		server:            server,
		addrRequests:      make(map[string]struct{}),
		spentRequests:     make(map[wire.OutPoint]struct{}),
		serviceRequestSem: makeSemaphore(cfg.RPCMaxConcurrentReqs),
ntfnChan:          make(chan []byte, 1), //非阻塞同步
		sendChan:          make(chan wsResponse, websocketSendBufferSize),
		quit:              make(chan struct{}),
	}
	return client, nil
}

//handleWebSocketHelp实现WebSocket连接的帮助命令。
func handleWebsocketHelp(wsc *wsClient, icmd interface{}) (interface{}, error) {
	cmd, ok := icmd.(*btcjson.HelpCmd)
	if !ok {
		return nil, btcjson.ErrRPCInternal
	}

//当没有特定命令时，提供所有命令的使用概述
//指定。
	var command string
	if cmd.Command != nil {
		command = *cmd.Command
	}
	if command == "" {
		usage, err := wsc.server.helpCacher.rpcUsage(true)
		if err != nil {
			context := "Failed to generate RPC usage"
			return nil, internalRPCError(err.Error(), context)
		}
		return usage, nil
	}

//检查请求的命令是否受支持和实现。
//搜索WebSocket处理程序列表以及
//处理程序，因为只应为这些情况提供帮助。
	valid := true
	if _, ok := rpcHandlers[command]; !ok {
		if _, ok := wsHandlers[command]; !ok {
			valid = false
		}
	}
	if !valid {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidParameter,
			Message: "Unknown command: " + command,
		}
	}

//获取命令的帮助。
	help, err := wsc.server.helpCacher.rpcMethodHelp(command)
	if err != nil {
		context := "Failed to generate help"
		return nil, internalRPCError(err.Error(), context)
	}
	return help, nil
}

//handleloadtxfilter为实现loadtxfilter命令扩展
//WebSocket连接。
//
//注意：此扩展从github.com/decred/dcrd移植
func handleLoadTxFilter(wsc *wsClient, icmd interface{}) (interface{}, error) {
	cmd := icmd.(*btcjson.LoadTxFilterCmd)

	outPoints := make([]wire.OutPoint, len(cmd.OutPoints))
	for i := range cmd.OutPoints {
		hash, err := chainhash.NewHashFromStr(cmd.OutPoints[i].Hash)
		if err != nil {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCInvalidParameter,
				Message: err.Error(),
			}
		}
		outPoints[i] = wire.OutPoint{
			Hash:  *hash,
			Index: cmd.OutPoints[i].Index,
		}
	}

	params := wsc.server.cfg.ChainParams

	wsc.Lock()
	if cmd.Reload || wsc.filterData == nil {
		wsc.filterData = newWSClientFilter(cmd.Addresses, outPoints,
			params)
		wsc.Unlock()
	} else {
		wsc.Unlock()

		wsc.filterData.mu.Lock()
		for _, a := range cmd.Addresses {
			wsc.filterData.addAddressStr(a, params)
		}
		for i := range outPoints {
			wsc.filterData.addUnspentOutPoint(&outPoints[i])
		}
		wsc.filterData.mu.Unlock()
	}

	return nil, nil
}

//handlenotifyblocks为实现notifyblocks命令扩展
//WebSocket连接。
func handleNotifyBlocks(wsc *wsClient, icmd interface{}) (interface{}, error) {
	wsc.server.ntfnMgr.RegisterBlockUpdates(wsc)
	return nil, nil
}

//handlesession实现WebSocket的会话命令扩展
//连接。
func handleSession(wsc *wsClient, icmd interface{}) (interface{}, error) {
	return &btcjson.SessionResult{SessionID: wsc.sessionID}, nil
}

//handlestopnotifyblocks为实现stopnotifyblocks命令扩展
//WebSocket连接。
func handleStopNotifyBlocks(wsc *wsClient, icmd interface{}) (interface{}, error) {
	wsc.server.ntfnMgr.UnregisterBlockUpdates(wsc)
	return nil, nil
}

//handlenotifyspeed实现的notifyspeed命令扩展
//WebSocket连接。
func handleNotifySpent(wsc *wsClient, icmd interface{}) (interface{}, error) {
	cmd, ok := icmd.(*btcjson.NotifySpentCmd)
	if !ok {
		return nil, btcjson.ErrRPCInternal
	}

	outpoints, err := deserializeOutpoints(cmd.OutPoints)
	if err != nil {
		return nil, err
	}

	wsc.server.ntfnMgr.RegisterSpentRequests(wsc, outpoints)
	return nil, nil
}

//handlenotifynewtransactions实现notifynewtransactions命令
//WebSocket连接的扩展。
func handleNotifyNewTransactions(wsc *wsClient, icmd interface{}) (interface{}, error) {
	cmd, ok := icmd.(*btcjson.NotifyNewTransactionsCmd)
	if !ok {
		return nil, btcjson.ErrRPCInternal
	}

	wsc.verboseTxUpdates = cmd.Verbose != nil && *cmd.Verbose
	wsc.server.ntfnMgr.RegisterNewMempoolTxsUpdates(wsc)
	return nil, nil
}

//handlestopNotifyNewTransactions实现stopNotifyNewTransactions
//WebSocket连接的命令扩展。
func handleStopNotifyNewTransactions(wsc *wsClient, icmd interface{}) (interface{}, error) {
	wsc.server.ntfnMgr.UnregisterNewMempoolTxsUpdates(wsc)
	return nil, nil
}

//handlenotifyreceived实现的notifyreceived命令扩展
//WebSocket连接。
func handleNotifyReceived(wsc *wsClient, icmd interface{}) (interface{}, error) {
	cmd, ok := icmd.(*btcjson.NotifyReceivedCmd)
	if !ok {
		return nil, btcjson.ErrRPCInternal
	}

//解码地址以验证输入，但使用字符串切片
//如果这些都可以的话，直接。
	err := checkAddressValidity(cmd.Addresses, wsc.server.cfg.ChainParams)
	if err != nil {
		return nil, err
	}

	wsc.server.ntfnMgr.RegisterTxOutAddressRequests(wsc, cmd.Addresses)
	return nil, nil
}

//handlestopnotifyspeed为实现stopnotifyspeed命令扩展
//WebSocket连接。
func handleStopNotifySpent(wsc *wsClient, icmd interface{}) (interface{}, error) {
	cmd, ok := icmd.(*btcjson.StopNotifySpentCmd)
	if !ok {
		return nil, btcjson.ErrRPCInternal
	}

	outpoints, err := deserializeOutpoints(cmd.OutPoints)
	if err != nil {
		return nil, err
	}

	for _, outpoint := range outpoints {
		wsc.server.ntfnMgr.UnregisterSpentRequest(wsc, outpoint)
	}

	return nil, nil
}

//handlestopnotifyreceived实现stopnotifyreceived命令扩展
//用于WebSocket连接。
func handleStopNotifyReceived(wsc *wsClient, icmd interface{}) (interface{}, error) {
	cmd, ok := icmd.(*btcjson.StopNotifyReceivedCmd)
	if !ok {
		return nil, btcjson.ErrRPCInternal
	}

//解码地址以验证输入，但使用字符串切片
//如果这些都可以的话，直接。
	err := checkAddressValidity(cmd.Addresses, wsc.server.cfg.ChainParams)
	if err != nil {
		return nil, err
	}

	for _, addr := range cmd.Addresses {
		wsc.server.ntfnMgr.UnregisterTxOutAddressRequest(wsc, addr)
	}

	return nil, nil
}

//checkaddressvalidity检查传递的每个地址的有效性
//字符串切片。它通过尝试使用
//当前活动网络参数。如果任何一个地址无法解码
//正常情况下，函数返回一个错误。否则，将返回零。
func checkAddressValidity(addrs []string, params *chaincfg.Params) error {
	for _, addr := range addrs {
		_, err := btcutil.DecodeAddress(addr, params)
		if err != nil {
			return &btcjson.RPCError{
				Code: btcjson.ErrRPCInvalidAddressOrKey,
				Message: fmt.Sprintf("Invalid address or key: %v",
					addr),
			}
		}
	}
	return nil
}

//反序列化输出点反序列化每个序列化输出点。
func deserializeOutpoints(serializedOuts []btcjson.OutPoint) ([]*wire.OutPoint, error) {
	outpoints := make([]*wire.OutPoint, 0, len(serializedOuts))
	for i := range serializedOuts {
		blockHash, err := chainhash.NewHashFromStr(serializedOuts[i].Hash)
		if err != nil {
			return nil, rpcDecodeHexError(serializedOuts[i].Hash)
		}
		index := serializedOuts[i].Index
		outpoints = append(outpoints, wire.NewOutPoint(blockHash, index))
	}

	return outpoints, nil
}

type rescanKeys struct {
	addrs   map[string]struct{}
	unspent map[wire.OutPoint]struct{}
}

//unspent slice返回当前未暂停的输出点切片，用于重新扫描
//查找键。主要用于注册输出点
//用于重新扫描完成后的连续通知。
func (r *rescanKeys) unspentSlice() []*wire.OutPoint {
	ops := make([]*wire.OutPoint, 0, len(r.unspent))
	for op := range r.unspent {
		opCopy := op
		ops = append(ops, &opCopy)
	}
	return ops
}

//errrescanreorg定义当不可恢复时返回的错误
//重新组织在重新扫描期间检测到。
var ErrRescanReorg = btcjson.RPCError{
	Code:    btcjson.ErrRPCDatabase,
	Message: "Reorganize",
}

//RescanBlock重新扫描单个块中的所有事务。这是个帮手
//用于handlerscan的函数。
func rescanBlock(wsc *wsClient, lookups *rescanKeys, blk *btcutil.Block) {
	for _, tx := range blk.Transactions() {
//此Tx的十六进制表示形式。仅当
//如果已经发出通知，则需要重新使用。
		var txHex string

//所有的输入和输出必须经过迭代才能正确
//但是，只需一个通知就可以修改未使用的映射
//对于任何匹配的事务，输入或输出应该
//已创建并发送。
		spentNotified := false
		recvNotified := false

//notifySpend是一个闭包，我们将在第一次检测到它时使用
//事务在过滤器列表中花费一个输出点/脚本。
		notifySpend := func() error {
			if txHex == "" {
				txHex = txHexString(tx.MsgTx())
			}
			marshalledJSON, err := newRedeemingTxNotification(
				txHex, tx.Index(), blk,
			)
			if err != nil {
				return fmt.Errorf("unable to marshal "+
					"btcjson.RedeeminTxNtfn: %v", err)
			}

			return wsc.QueueNotification(marshalledJSON)
		}

//我们将从迭代事务的输入开始
//确定它是否在筛选列表中花费了一个输出点/脚本。
		for _, txin := range tx.MsgTx().TxIn {
//如果它花费了一个前哨点，我们将分配一个花费
//交易通知。
			if _, ok := lookups.unspent[txin.PreviousOutPoint]; ok {
				delete(lookups.unspent, txin.PreviousOutPoint)

				if spentNotified {
					continue
				}

				err := notifySpend()

//如果WebSocket客户端
//断开的。
				if err == ErrClientQuit {
					return
				}
				if err != nil {
					rpcsLog.Errorf("Unable to notify "+
						"redeeming transaction %v: %v",
						tx.Hash(), err)
					continue
				}

				spentNotified = true
			}

//我们还将重新计算输入的pkscript
//尝试花费以确定它是否
//与我们有关。
			pkScript, err := txscript.ComputePkScript(
				txin.SignatureScript, txin.Witness,
			)
			if err != nil {
				continue
			}
			addr, err := pkScript.Address(wsc.server.cfg.ChainParams)
			if err != nil {
				continue
			}

//如果是，我们还将发送一个支出通知
//如果我们还没有的话。
			if _, ok := lookups.addrs[addr.String()]; ok {
				if spentNotified {
					continue
				}

				err := notifySpend()

//如果WebSocket客户端
//断开的。
				if err == ErrClientQuit {
					return
				}
				if err != nil {
					rpcsLog.Errorf("Unable to notify "+
						"redeeming transaction %v: %v",
						tx.Hash(), err)
					continue
				}

				spentNotified = true
			}
		}

		for txOutIdx, txout := range tx.MsgTx().TxOut {
			_, addrs, _, _ := txscript.ExtractPkScriptAddrs(
				txout.PkScript, wsc.server.cfg.ChainParams)

			for _, addr := range addrs {
				if _, ok := lookups.addrs[addr.String()]; !ok {
					continue
				}

				outpoint := wire.OutPoint{
					Hash:  *tx.Hash(),
					Index: uint32(txOutIdx),
				}
				lookups.unspent[outpoint] = struct{}{}

				if recvNotified {
					continue
				}

				if txHex == "" {
					txHex = txHexString(tx.MsgTx())
				}
				ntfn := btcjson.NewRecvTxNtfn(txHex,
					blockDetails(blk, tx.Index()))

				marshalledJSON, err := btcjson.MarshalCmd(nil, ntfn)
				if err != nil {
					rpcsLog.Errorf("Failed to marshal recvtx notification: %v", err)
					return
				}

				err = wsc.QueueNotification(marshalledJSON)
//如果WebSocket客户端
//断开的。
				if err == ErrClientQuit {
					return
				}
				recvNotified = true
			}
		}
	}
}

//rescanblockfilter重新扫描块以查找
//已传递查找键。任何发现的事务都返回十六进制编码为
//一个字符串切片。
//
//注意：此扩展从github.com/decred/dcrd移植
func rescanBlockFilter(filter *wsClientFilter, block *btcutil.Block, params *chaincfg.Params) []string {
	var transactions []string

	filter.mu.Lock()
	for _, tx := range block.Transactions() {
		msgTx := tx.MsgTx()

//跟踪交易记录是否已添加
//结果。不应添加两次。
		added := false

//如果不是CoinBase事务，则扫描输入。
		if !blockchain.IsCoinBaseTx(msgTx) {
			for _, input := range msgTx.TxIn {
				if !filter.existsUnspentOutPoint(&input.PreviousOutPoint) {
					continue
				}
				if !added {
					transactions = append(
						transactions,
						txHexString(msgTx))
					added = true
				}
			}
		}

//扫描输出。
		for i, output := range msgTx.TxOut {
			_, addrs, _, err := txscript.ExtractPkScriptAddrs(
				output.PkScript, params)
			if err != nil {
				continue
			}
			for _, a := range addrs {
				if !filter.existsAddress(a) {
					continue
				}

				op := wire.OutPoint{
					Hash:  *tx.Hash(),
					Index: uint32(i),
				}
				filter.addUnspentOutPoint(&op)

				if !added {
					transactions = append(
						transactions,
						txHexString(msgTx))
					added = true
				}
			}
		}
	}
	filter.mu.Unlock()

	return transactions
}

//handleRescanBlocks为实现RescanBlocks命令扩展
//WebSocket连接。
//
//注意：此扩展从github.com/decred/dcrd移植
func handleRescanBlocks(wsc *wsClient, icmd interface{}) (interface{}, error) {
	cmd, ok := icmd.(*btcjson.RescanBlocksCmd)
	if !ok {
		return nil, btcjson.ErrRPCInternal
	}

//加载客户端的事务筛选器。必须存在才能继续。
	wsc.Lock()
	filter := wsc.filterData
	wsc.Unlock()
	if filter == nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCMisc,
			Message: "Transaction filter must be loaded before rescanning",
		}
	}

	blockHashes := make([]*chainhash.Hash, len(cmd.BlockHashes))

	for i := range cmd.BlockHashes {
		hash, err := chainhash.NewHashFromStr(cmd.BlockHashes[i])
		if err != nil {
			return nil, err
		}
		blockHashes[i] = hash
	}

	discoveredData := make([]btcjson.RescannedBlock, 0, len(blockHashes))

//迭代请求中的每个块并重新扫描。当一个街区
//包含相关事务，将其添加到响应中。
	bc := wsc.server.cfg.Chain
	params := wsc.server.cfg.ChainParams
	var lastBlockHash *chainhash.Hash
	for i := range blockHashes {
		block, err := bc.BlockByHash(blockHashes[i])
		if err != nil {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCBlockNotFound,
				Message: "Failed to fetch block: " + err.Error(),
			}
		}
		if lastBlockHash != nil && block.MsgBlock().Header.PrevBlock != *lastBlockHash {
			return nil, &btcjson.RPCError{
				Code: btcjson.ErrRPCInvalidParameter,
				Message: fmt.Sprintf("Block %v is not a child of %v",
					blockHashes[i], lastBlockHash),
			}
		}
		lastBlockHash = blockHashes[i]

		transactions := rescanBlockFilter(filter, block, params)
		if len(transactions) != 0 {
			discoveredData = append(discoveredData, btcjson.RescannedBlock{
				Hash:         cmd.BlockHashes[i],
				Transactions: transactions,
			})
		}
	}

	return &discoveredData, nil
}

//recoverfromreorg尝试从检测到的重新组织中恢复
//再扫描。它从数据库中获取一个新的块shas范围，并
//验证新的块范围是否与前一个块位于同一个分叉上
//块的范围。如果此条件不成立，则JSON-RPC错误
//对于不可恢复的重新组织，返回。
func recoverFromReorg(chain *blockchain.BlockChain, minBlock, maxBlock int32,
	lastBlock *chainhash.Hash) ([]chainhash.Hash, error) {

	hashList, err := chain.HeightRange(minBlock, maxBlock)
	if err != nil {
		rpcsLog.Errorf("Error looking up block range: %v", err)
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCDatabase,
			Message: "Database error: " + err.Error(),
		}
	}
	if lastBlock == nil || len(hashList) == 0 {
		return hashList, nil
	}

	blk, err := chain.BlockByHash(&hashList[0])
	if err != nil {
		rpcsLog.Errorf("Error looking up possibly reorged block: %v",
			err)
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCDatabase,
			Message: "Database error: " + err.Error(),
		}
	}
	jsonErr := descendantBlock(lastBlock, blk)
	if jsonErr != nil {
		return nil, jsonErr
	}
	return hashList, nil
}

//如果当前块
//在重新组织期间获取的不是父块哈希的直接子级。
func descendantBlock(prevHash *chainhash.Hash, curBlock *btcutil.Block) error {
	curHash := &curBlock.MsgBlock().Header.PrevBlock
	if !prevHash.IsEqual(curHash) {
		rpcsLog.Errorf("Stopping rescan for reorged block %v "+
			"(replaced by block %v)", prevHash, curHash)
		return &ErrRescanReorg
	}
	return nil
}

//handleRescan为WebSocket实现Rescan命令扩展
//连接。
//
//注意：这不能智能地处理REORG，修复需要数据库
//更改（用于安全、同时访问完整块范围和支持
//对于最好的链条以外的其他链条）。但是，它将检测
//REORG删除了以前处理过的块，并导致
//处理程序出错。客户端必须通过查找仍在
//链（可能来自RescanProgress通知）以恢复其
//再扫描。
func handleRescan(wsc *wsClient, icmd interface{}) (interface{}, error) {
	cmd, ok := icmd.(*btcjson.RescanCmd)
	if !ok {
		return nil, btcjson.ErrRPCInternal
	}

	outpoints := make([]*wire.OutPoint, 0, len(cmd.OutPoints))
	for i := range cmd.OutPoints {
		cmdOutpoint := &cmd.OutPoints[i]
		blockHash, err := chainhash.NewHashFromStr(cmdOutpoint.Hash)
		if err != nil {
			return nil, rpcDecodeHexError(cmdOutpoint.Hash)
		}
		outpoint := wire.NewOutPoint(blockHash, cmdOutpoint.Index)
		outpoints = append(outpoints, outpoint)
	}

	numAddrs := len(cmd.Addresses)
	if numAddrs == 1 {
		rpcsLog.Info("Beginning rescan for 1 address")
	} else {
		rpcsLog.Infof("Beginning rescan for %d addresses", numAddrs)
	}

//生成查找映射。
	lookups := rescanKeys{
		addrs:   map[string]struct{}{},
		unspent: map[wire.OutPoint]struct{}{},
	}
	for _, addrStr := range cmd.Addresses {
		lookups.addrs[addrStr] = struct{}{}
	}
	for _, outpoint := range outpoints {
		lookups.unspent[*outpoint] = struct{}{}
	}

	chain := wsc.server.cfg.Chain

	minBlockHash, err := chainhash.NewHashFromStr(cmd.BeginBlock)
	if err != nil {
		return nil, rpcDecodeHexError(cmd.BeginBlock)
	}
	minBlock, err := chain.BlockHeightByHash(minBlockHash)
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCBlockNotFound,
			Message: "Error getting block: " + err.Error(),
		}
	}

	maxBlock := int32(math.MaxInt32)
	if cmd.EndBlock != nil {
		maxBlockHash, err := chainhash.NewHashFromStr(*cmd.EndBlock)
		if err != nil {
			return nil, rpcDecodeHexError(*cmd.EndBlock)
		}
		maxBlock, err = chain.BlockHeightByHash(maxBlockHash)
		if err != nil {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCBlockNotFound,
				Message: "Error getting block: " + err.Error(),
			}
		}
	}

//lastblock和lastblockhash跟踪以前重新扫描的块。
//当没有重新扫描以前的块时，它们等于零。
	var lastBlock *btcutil.Block
	var lastBlockHash *chainhash.Hash

//创建一个断续器，以至少等待10秒，然后通知
//由重新扫描完成的当前进度的WebSocket客户端。
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

//不要同时获取所有块sha，而是以较小的块获取
//为了确保大量的重新扫描占用有限的内存。
fetchRange:
	for minBlock < maxBlock {
//限制一次提取到的最大哈希数
//单个库存中允许的最大项目数。
//这个值可能会更高，因为它不会创建库存
//但这反映了
//对等协议。
		maxLoopBlock := maxBlock
		if maxLoopBlock-minBlock > wire.MaxInvPerMsg {
			maxLoopBlock = minBlock + wire.MaxInvPerMsg
		}
		hashList, err := chain.HeightRange(minBlock, maxLoopBlock)
		if err != nil {
			rpcsLog.Errorf("Error looking up block range: %v", err)
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCDatabase,
				Message: "Database error: " + err.Error(),
			}
		}
		if len(hashList) == 0 {
//如果没有块哈希，则重新扫描完成。
//已成功获取范围和停止块
//提供。
			if maxBlock != math.MaxInt32 {
				break
			}

//如果重新扫描是通过当前块，请设置
//继续接收通知的客户端
//关于所有重新扫描的地址和当前集
//未消耗的输出。
//
//这是通过临时获取
//访问块管理器。如果没有更多的街区
//在这个暂停和上面的获取之间附加，
//然后可以安全地为注册WebSocket客户端
//必要时持续通知。否则，
//再次继续提取循环以重新扫描新的
//块（或由于无法恢复的重新组织而导致的错误）。
			pauseGuard := wsc.server.cfg.SyncMgr.Pause()
			best := wsc.server.cfg.Chain.BestSnapshot()
			curHash := &best.Hash
			again := true
			if lastBlockHash == nil || *lastBlockHash == *curHash {
				again = false
				n := wsc.server.ntfnMgr
				n.RegisterSpentRequests(wsc, lookups.unspentSlice())
				n.RegisterTxOutAddressRequests(wsc, cmd.Addresses)
			}
			close(pauseGuard)
			if err != nil {
				rpcsLog.Errorf("Error fetching best block "+
					"hash: %v", err)
				return nil, &btcjson.RPCError{
					Code: btcjson.ErrRPCDatabase,
					Message: "Database error: " +
						err.Error(),
				}
			}
			if again {
				continue
			}
			break
		}

	loopHashList:
		for i := range hashList {
			blk, err := chain.BlockByHash(&hashList[i])
			if err != nil {
//仅当块不能
//为哈希找到。
				if dbErr, ok := err.(database.Error); !ok ||
					dbErr.ErrorCode != database.ErrBlockNotFound {

					rpcsLog.Errorf("Error looking up "+
						"block: %v", err)
					return nil, &btcjson.RPCError{
						Code: btcjson.ErrRPCDatabase,
						Message: "Database error: " +
							err.Error(),
					}
				}

//如果指定了绝对最大块，则不要
//尝试处理REORG。
				if maxBlock != math.MaxInt32 {
					rpcsLog.Errorf("Stopping rescan for "+
						"reorged block %v",
						cmd.EndBlock)
					return nil, &ErrRescanReorg
				}

//如果查找以前有效的块
//哈希失败，可能有一个REORG。
//获取新的块哈希范围并验证
//以前处理过的块（如果有
//数据库中仍然存在。如果它
//不，我们出错了。
//
//goto用于将执行分支回
//在评估范围之前，必须
//重新评估新哈希表。
				minBlock += int32(i)
				hashList, err = recoverFromReorg(chain,
					minBlock, maxBlock, lastBlockHash)
				if err != nil {
					return nil, err
				}
				if len(hashList) == 0 {
					break fetchRange
				}
				goto loopHashList
			}
			if i == 0 && lastBlockHash != nil {
//确保新哈希列表位于同一个分叉上
//作为旧哈希表的最后一个块。
				jsonErr := descendantBlock(lastBlockHash, blk)
				if jsonErr != nil {
					return nil, jsonErr
				}
			}

//如果
//请求重新扫描的客户端已断开连接。
			select {
			case <-wsc.quit:
				rpcsLog.Debugf("Stopped rescan at height %v "+
					"for disconnected client", blk.Height())
				return nil, nil
			default:
				rescanBlock(wsc, &lookups, blk)
				lastBlock = blk
				lastBlockHash = blk.Hash()
			}

//定期通知客户进度
//完整的。如果没有进展，继续下一个块
//还需要通知。
			select {
case <-ticker.C: //坠落
			default:
				continue
			}

			n := btcjson.NewRescanProgressNtfn(hashList[i].String(),
				blk.Height(), blk.MsgBlock().Header.Timestamp.Unix())
			mn, err := btcjson.MarshalCmd(nil, n)
			if err != nil {
				rpcsLog.Errorf("Failed to marshal rescan "+
					"progress notification: %v", err)
				continue
			}

			if err = wsc.QueueNotification(mn); err == ErrClientQuit {
//如果客户端断开连接，则完成。
				rpcsLog.Debugf("Stopped rescan at height %v "+
					"for disconnected client", blk.Height())
				return nil, nil
			}
		}

		minBlock += int32(len(hashList))
	}

//将完成的重新扫描通知WebSocket客户端。由于BTCD
//异步排队通知以不阻止调用代码，
//不保证在
//Rescan（例如RescanProgress、Recvtx和Redeemingtx）将
//在重新扫描RPC返回之前收到。因此，另一种方法
//需要安全地通知客户端所有重新扫描通知
//被送来。
	n := btcjson.NewRescanFinishedNtfn(lastBlockHash.String(),
		lastBlock.Height(),
		lastBlock.MsgBlock().Header.Timestamp.Unix())
	if mn, err := btcjson.MarshalCmd(nil, n); err != nil {
		rpcsLog.Errorf("Failed to marshal rescan finished "+
			"notification: %v", err)
	} else {
//重新扫描完成了，所以我们不关心客户是否
//此时已断开连接，因此放弃错误。
		_ = wsc.QueueNotification(mn)
	}

	rpcsLog.Info("Finished rescan")
	return nil, nil
}

func init() {
	wsHandlers = wsHandlersBeforeInit
}
