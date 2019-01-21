
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015-2018 BTCSuite开发者
//版权所有（c）2016-2018法令开发商
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package peer_test

import (
	"fmt"
	"net"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/peer"
	"github.com/btcsuite/btcd/wire"
)

//mockremotepeer创建在simnet端口上侦听的基本入站对等机
//使用示例“PeerConnection”。它不会回来，直到李斯特
//主动的。
func mockRemotePeer() error {
//将对等机配置为不提供服务的Simnet节点。
	peerCfg := &peer.Config{
UserAgentName:    "peer",  //User agent name to advertise.
UserAgentVersion: "1.0.0", //要公布的用户代理版本。
		ChainParams:      &chaincfg.SimNetParams,
		TrickleInterval:  time.Second * 10,
	}

//接受SIMNET端口上的连接。
	listener, err := net.Listen("tcp", "127.0.0.1:18555")
	if err != nil {
		return err
	}
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Accept: error %v\n", err)
			return
		}

//创建并启动入站对等机。
		p := peer.NewInboundPeer(peerCfg)
		p.AssociateConnection(conn)
	}()

	return nil
}

//This example demonstrates the basic process for initializing and creating an
//出站对等机。对等方通过交换版本和verack消息进行协商。
//为了演示，版本消息的简单处理程序附加到
//同龄人。
func Example_newOutboundPeer() {
//通常不需要这样做，因为出站对等机将
//但是，由于执行了此示例，因此连接到远程对等机
//并且测试时，需要模拟远程对等体来监听出站。
//同龄人。
	if err := mockRemotePeer(); err != nil {
		fmt.Printf("mockRemotePeer: unexpected error %v\n", err)
		return
	}

//创建配置为充当simnet节点的出站对等机
//它不提供任何服务，并且有版本和verack的侦听器
//信息。这里使用verack侦听器向下面的代码发送信号
//当通过发送一个信道信号完成握手时。
	verack := make(chan struct{})
	peerCfg := &peer.Config{
UserAgentName:    "peer",  //要公布的用户代理名称。
UserAgentVersion: "1.0.0", //要公布的用户代理版本。
		ChainParams:      &chaincfg.SimNetParams,
		Services:         0,
		TrickleInterval:  time.Second * 10,
		Listeners: peer.MessageListeners{
			OnVersion: func(p *peer.Peer, msg *wire.MsgVersion) *wire.MsgReject {
				fmt.Println("outbound: received version")
				return nil
			},
			OnVerAck: func(p *peer.Peer, msg *wire.MsgVerAck) {
				verack <- struct{}{}
			},
		},
	}
	p, err := peer.NewOutboundPeer(peerCfg, "127.0.0.1:18555")
	if err != nil {
		fmt.Printf("NewOutboundPeer: error %v\n", err)
		return
	}

//建立到对等地址的连接并将其标记为已连接。
	conn, err := net.Dial("tcp", p.Addr())
	if err != nil {
		fmt.Printf("net.Dial: error %v\n", err)
		return
	}
	p.AssociateConnection(conn)

//在失败的情况下等待VARACK消息或超时。
	select {
	case <-verack:
	case <-time.After(time.Second * 1):
		fmt.Printf("Example_peerConnection: verack timeout")
	}

//断开对等机。
	p.Disconnect()
	p.WaitForDisconnect()

//输出：
//outbound: received version
}
