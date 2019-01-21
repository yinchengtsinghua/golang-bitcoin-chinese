
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package main

import (
	"sync/atomic"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/mempool"
	"github.com/btcsuite/btcd/netsync"
	"github.com/btcsuite/btcd/peer"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

//rpc peer提供与rpc服务器一起使用的对等机，并实现
//rpcserverPeer interface.
type rpcPeer serverPeer

//确保rpcpeer实现rpcserverpeer接口。
var _ rpcserverPeer = (*rpcPeer)(nil)

//topeer返回基础对等实例。
//
//此函数对于并发访问是安全的，并且是rpcserverpeer的一部分
//接口实现。
func (p *rpcPeer) ToPeer() *peer.Peer {
	if p == nil {
		return nil
	}
	return (*serverPeer)(p).Peer
}

//ISTXRelayDisabled返回对等方是否已禁用事务
//继电器。
//
//此函数对于并发访问是安全的，并且是rpcserverpeer的一部分
//接口实现。
func (p *rpcPeer) IsTxRelayDisabled() bool {
	return (*serverPeer)(p).disableRelayTx
}

//BanScore返回当前整数值，该整数值表示对等端的距离
//就是被禁止。
//
//此函数对于并发访问是安全的，并且是rpcserverpeer的一部分
//接口实现。
func (p *rpcPeer) BanScore() uint32 {
	return (*serverPeer)(p).banScore.Int()
}

//feefilter返回请求的当前最低费率，其中
//交易应当公布。
//
//此函数对于并发访问是安全的，并且是rpcserverpeer的一部分
//接口实现。
func (p *rpcPeer) FeeFilter() int64 {
	return atomic.LoadInt64(&(*serverPeer)(p).feeFilter)
}

//rpcconnmanager提供一个连接管理器，用于RPC服务器和
//实现rpcserverconmanager接口。
type rpcConnManager struct {
	server *server
}

//确保rpcconnmanager实现rpcserverconmanager接口。
var _ rpcserverConnManager = &rpcConnManager{}

//Connect将提供的地址添加为新的出站对等机。永久的旗帜
//指示是否使对等机持久化，如果
//连接丢失。尝试连接到已存在的对等计算机将
//返回一个错误。
//
//This function is safe for concurrent access and is part of the
//rpcserverconmanager接口实现。
func (cm *rpcConnManager) Connect(addr string, permanent bool) error {
	replyChan := make(chan error)
	cm.server.query <- connectNodeMsg{
		addr:      addr,
		permanent: permanent,
		reply:     replyChan,
	}
	return <-replyChan
}

//removeByID从以下列表中删除与提供的ID关联的对等方
//坚持不懈的同龄人。尝试删除不存在的ID将返回
//一个错误。
//
//此函数对于并发访问是安全的，并且是
//rpcserverconmanager接口实现。
func (cm *rpcConnManager) RemoveByID(id int32) error {
	replyChan := make(chan error)
	cm.server.query <- removeNodeMsg{
		cmp:   func(sp *serverPeer) bool { return sp.ID() == id },
		reply: replyChan,
	}
	return <-replyChan
}

//removebyaddr从
//持久对等体列表。试图删除不存在的地址
//exist将返回一个错误。
//
//此函数对于并发访问是安全的，并且是
//rpcserverconmanager接口实现。
func (cm *rpcConnManager) RemoveByAddr(addr string) error {
	replyChan := make(chan error)
	cm.server.query <- removeNodeMsg{
		cmp:   func(sp *serverPeer) bool { return sp.Addr() == addr },
		reply: replyChan,
	}
	return <-replyChan
}

//disconnectByID断开与提供的ID关联的对等机。此
//适用于入站和出站对等机。正在尝试删除
//不存在将返回错误。
//
//此函数对于并发访问是安全的，并且是
//rpcserverconmanager接口实现。
func (cm *rpcConnManager) DisconnectByID(id int32) error {
	replyChan := make(chan error)
	cm.server.query <- disconnectNodeMsg{
		cmp:   func(sp *serverPeer) bool { return sp.ID() == id },
		reply: replyChan,
	}
	return <-replyChan
}

//DisconnectByAddr disconnects the peer associated with the provided address.
//这适用于入站和出站对等机。正在尝试删除
//不存在的地址将返回错误。
//
//此函数对于并发访问是安全的，并且是
//rpcserverconmanager接口实现。
func (cm *rpcConnManager) DisconnectByAddr(addr string) error {
	replyChan := make(chan error)
	cm.server.query <- disconnectNodeMsg{
		cmp:   func(sp *serverPeer) bool { return sp.Addr() == addr },
		reply: replyChan,
	}
	return <-replyChan
}

//ConnectedCount返回当前连接的对等数。
//
//此函数对于并发访问是安全的，并且是
//rpcserverconmanager接口实现。
func (cm *rpcConnManager) ConnectedCount() int32 {
	return cm.server.ConnectedCount()
}

//nettotals返回通过网络接收和发送的所有字节的总和
//对于所有的同龄人。
//
//此函数对于并发访问是安全的，并且是
//rpcserverconmanager接口实现。
func (cm *rpcConnManager) NetTotals() (uint64, uint64) {
	return cm.server.NetTotals()
}

//ConnectedPeers返回一个由所有连接的对等方组成的数组。
//
//此函数对于并发访问是安全的，并且是
//rpcserverconmanager接口实现。
func (cm *rpcConnManager) ConnectedPeers() []rpcserverPeer {
	replyChan := make(chan []*serverPeer)
	cm.server.query <- getPeersMsg{reply: replyChan}
	serverPeers := <-replyChan

//Convert to RPC server peers.
	peers := make([]rpcserverPeer, 0, len(serverPeers))
	for _, sp := range serverPeers {
		peers = append(peers, (*rpcPeer)(sp))
	}
	return peers
}

//PersistentPeers返回一个由所有添加的Persistent组成的数组
//同龄人。
//
//此函数对于并发访问是安全的，并且是
//rpcserverconmanager接口实现。
func (cm *rpcConnManager) PersistentPeers() []rpcserverPeer {
	replyChan := make(chan []*serverPeer)
	cm.server.query <- getAddedNodesMsg{reply: replyChan}
	serverPeers := <-replyChan

//转换为通用对等端。
	peers := make([]rpcserverPeer, 0, len(serverPeers))
	for _, sp := range serverPeers {
		peers = append(peers, (*rpcPeer)(sp))
	}
	return peers
}

//BroadcastMessage sends the provided message to all currently connected peers.
//
//此函数对于并发访问是安全的，并且是
//rpcserverconmanager接口实现。
func (cm *rpcConnManager) BroadcastMessage(msg wire.Message) {
	cm.server.BroadcastMessage(msg)
}

//addrebroadcastinventory将提供的清单添加到
//在库存出现在
//块。
//
//此函数对于并发访问是安全的，并且是
//rpcserverconmanager接口实现。
func (cm *rpcConnManager) AddRebroadcastInventory(iv *wire.InvVect, data interface{}) {
	cm.server.AddRebroadcastInventory(iv, data)
}

//RelayTransactions为所有
//已将事务传递给所有连接的对等方。
func (cm *rpcConnManager) RelayTransactions(txns []*mempool.TxDesc) {
	cm.server.relayTransactions(txns)
}

//rpcSyncMgr provides a block manager for use with the RPC server and
//实现RpcServerSyncManager接口。
type rpcSyncMgr struct {
	server  *server
	syncMgr *netsync.SyncManager
}

//确保rpcsyncmgr实现rpcserversyncmanager接口。
var _ rpcserverSyncManager = (*rpcSyncMgr)(nil)

//iscurrent返回同步管理器是否相信链
//与网络其他部分相比的电流。
//
//此函数对于并发访问是安全的，并且是
//RpcServerSyncManager接口实现。
func (b *rpcSyncMgr) IsCurrent() bool {
	return b.syncMgr.IsCurrent()
}

//SubmitBlock处理后将提供的块提交给网络
//局部地。
//
//此函数对于并发访问是安全的，并且是
//RpcServerSyncManager接口实现。
func (b *rpcSyncMgr) SubmitBlock(block *btcutil.Block, flags blockchain.BehaviorFlags) (bool, error) {
	return b.syncMgr.ProcessBlock(block, flags)
}

//暂停暂停同步管理器，直到返回的通道关闭。
//
//此函数对于并发访问是安全的，并且是
//RpcServerSyncManager接口实现。
func (b *rpcSyncMgr) Pause() chan<- struct{} {
	return b.syncMgr.Pause()
}

//syncpeerid返回当前用于同步的对等机
//从…
//
//此函数对于并发访问是安全的，并且是
//RpcServerSyncManager接口实现。
func (b *rpcSyncMgr) SyncPeerID() int32 {
	return b.syncMgr.SyncPeerID()
}

//locateBlocks返回块在中第一个已知块之后的哈希值。
//提供的定位器，直到提供的停止哈希或当前提示
//已达到Wire.MaxBlockHeadersPermsg哈希的最大值。
//
//此函数对于并发访问是安全的，并且是
//RpcServerSyncManager接口实现。
func (b *rpcSyncMgr) LocateHeaders(locators []*chainhash.Hash, hashStop *chainhash.Hash) []wire.BlockHeader {
	return b.server.chain.LocateHeaders(locators, hashStop)
}
