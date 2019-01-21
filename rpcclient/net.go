
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
	"encoding/json"

	"github.com/btcsuite/btcd/btcjson"
)

//addnodecommand枚举addnode函数的可用命令
//接受。
type AddNodeCommand string

//用于指示addnode函数命令的常量。
const (
//anadd指示应将指定主机作为持久主机添加
//同龄人。
	ANAdd AddNodeCommand = "add"

//anremove表示应删除指定的对等机。
	ANRemove AddNodeCommand = "remove"

//Anonetry表示指定主机应尝试连接一次，
//但这不应该是持久的。
	ANOneTry AddNodeCommand = "onetry"
)

//字符串以可读形式返回addnodecommand。
func (cmd AddNodeCommand) String() string {
	return string(cmd)
}

//FutureADNoderesult是未来交付
//addnodeasync RPC调用（或适用的错误）。
type FutureAddNodeResult chan *response

//接收等待未来承诺的响应并返回错误
//执行指定命令时发生。
func (r FutureAddNodeResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

//addnodeasync返回可用于获取结果的类型的实例
//在将来的某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅addnode。
func (c *Client) AddNodeAsync(host string, command AddNodeCommand) FutureAddNodeResult {
	cmd := btcjson.NewAddNodeCmd(host, btcjson.AddNodeSubCmd(command))
	return c.sendCmd(cmd)
}

//addnode尝试对传递的持久对等端执行传递的命令。
//例如，它可以用于添加或删除持久对等，或者
//与对等机的一次性连接。
//
//它不能用于删除非持久性对等。
func (c *Client) AddNode(host string, command AddNodeCommand) error {
	return c.AddNodeAsync(host, command).Receive()
}

//futurenoderesult是未来交付nodeasync结果的承诺。
//RPC调用（或适用的错误）。
type FutureNodeResult chan *response

//接收等待未来承诺的响应并返回错误
//执行指定命令时发生。
func (r FutureNodeResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

//nodeAsync返回可用于获取结果的类型的实例
//在将来的某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参见节点。
func (c *Client) NodeAsync(command btcjson.NodeSubCmd, host string,
	connectSubCmd *string) FutureNodeResult {
	cmd := btcjson.NewNodeCmd(command, host, connectSubCmd)
	return c.sendCmd(cmd)
}

//节点尝试在主机上执行传递的节点命令。
//例如，它可以用于添加或删除持久对等，或者
//连接或断开非持久性连接。
//
//connectSubCmd应设置为“perm”或“temp”，具体取决于
//无论我们的目标是持久的还是非持久的对等。通过零
//将使用当前为“temp”的默认值。
func (c *Client) Node(command btcjson.NodeSubCmd, host string,
	connectSubCmd *string) error {
	return c.NodeAsync(command, host, connectSubCmd).Receive()
}

//FutureGetAddedNodeForesult是未来交付
//GetAddedNodeInfoAsync RPC调用（或适用的错误）。
type FutureGetAddedNodeInfoResult chan *response

//接收等待未来承诺的响应并返回信息
//about manually added (persistent) peers.
func (r FutureGetAddedNodeInfoResult) Receive() ([]btcjson.GetAddedNodeInfoResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//取消标记为getAddedNodeInfo结果对象的数组。
	var nodeInfo []btcjson.GetAddedNodeInfoResult
	err = json.Unmarshal(res, &nodeInfo)
	if err != nil {
		return nil, err
	}

	return nodeInfo, nil
}

//GetAddedNodeInfoAsync returns an instance of a type that can be used to get
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅getaddednodeinfo。
func (c *Client) GetAddedNodeInfoAsync(peer string) FutureGetAddedNodeInfoResult {
	cmd := btcjson.NewGetAddedNodeInfoCmd(true, &peer)
	return c.sendCmd(cmd)
}

//GetAddedNodeInfo返回有关手动添加（持久）对等机的信息。
//
//请参阅getaddednodeinfonodns以仅检索添加的列表（持久）
//同龄人。
func (c *Client) GetAddedNodeInfo(peer string) ([]btcjson.GetAddedNodeInfoResult, error) {
	return c.GetAddedNodeInfoAsync(peer).Receive()
}

//FutureGetAddedNodeinFonodNsresult是未来交付结果的承诺
//GetAddedNodeInFonodNSAsync RPC调用（或适用的错误）。
type FutureGetAddedNodeInfoNoDNSResult chan *response

//Receive等待将来承诺的响应，并返回
//手动添加（持久）对等机。
func (r FutureGetAddedNodeInfoNoDNSResult) Receive() ([]string, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为字符串数组。
	var nodes []string
	err = json.Unmarshal(res, &nodes)
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

//GetAddedNodeInFonodNSAsync返回可用于
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅getaddednodeinfonodns。
func (c *Client) GetAddedNodeInfoNoDNSAsync(peer string) FutureGetAddedNodeInfoNoDNSResult {
	cmd := btcjson.NewGetAddedNodeInfoCmd(false, &peer)
	return c.sendCmd(cmd)
}

//getaddednodeinfonodns返回手动添加（持久）对等机的列表。
//这是通过在基础RPC中将DNS标志设置为false来实现的。
//
//See GetAddedNodeInfo to obtain more information about each added (persistent)
//同龄人。
func (c *Client) GetAddedNodeInfoNoDNS(peer string) ([]string, error) {
	return c.GetAddedNodeInfoNoDNSAsync(peer).Receive()
}

//FutureGetConnectionCountResult是未来交付结果的承诺
//of a GetConnectionCountAsync RPC invocation (or an applicable error).
type FutureGetConnectionCountResult chan *response

//receive等待将来承诺的响应并返回数字
//与其他对等机的活动连接。
func (r FutureGetConnectionCountResult) Receive() (int64, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return 0, err
	}

//将结果取消标记为Int64。
	var count int64
	err = json.Unmarshal(res, &count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

//getConnectionCountAsync返回可用于获取的类型的实例
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅getConnectionCount。
func (c *Client) GetConnectionCountAsync() FutureGetConnectionCountResult {
	cmd := btcjson.NewGetConnectionCountCmd()
	return c.sendCmd(cmd)
}

//getConnectionCount返回到其他对等方的活动连接数。
func (c *Client) GetConnectionCount() (int64, error) {
	return c.GetConnectionCountAsync().Receive()
}

//FuturePingResult是未来交付PingAsyncRPC结果的承诺。
//调用（或适用的错误）。
type FuturePingResult chan *response

//receive等待将来承诺的响应并返回结果
//对发送到每个连接的对等机的ping进行排队。
func (r FuturePingResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

//PingAsync返回一个类型的实例，该实例可用于获取
//通过调用返回的
//实例。
//
//有关阻止版本和更多详细信息，请参阅ping。
func (c *Client) PingAsync() FuturePingResult {
	cmd := btcjson.NewPingCmd()
	return c.sendCmd(cmd)
}

//Ping queues a ping to be sent to each connected peer.
//
//使用getpeerinfo函数并检查pingtime和pingwait字段以
//访问Ping时间。
func (c *Client) Ping() error {
	return c.PingAsync().Receive()
}

//FutureGetPeerInformationResult是未来交付
//GetPeerInfoAsync RPC调用（或适用的错误）。
type FutureGetPeerInfoResult chan *response

//receive等待将来承诺的响应并返回有关
//每个连接的网络对等机。
func (r FutureGetPeerInfoResult) Receive() ([]btcjson.GetPeerInfoResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为getpeerinfo结果对象的数组。
	var peerInfo []btcjson.GetPeerInfoResult
	err = json.Unmarshal(res, &peerInfo)
	if err != nil {
		return nil, err
	}

	return peerInfo, nil
}

//GetPeerInfoAsync returns an instance of a type that can be used to get the
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅getpeerinfo。
func (c *Client) GetPeerInfoAsync() FutureGetPeerInfoResult {
	cmd := btcjson.NewGetPeerInfoCmd()
	return c.sendCmd(cmd)
}

//getpeerinfo返回每个连接的网络对等端的数据。
func (c *Client) GetPeerInfo() ([]btcjson.GetPeerInfoResult, error) {
	return c.GetPeerInfoAsync().Receive()
}

//FutureGetNetTotalsResult是未来交付
//GetNettotalAsync RPC调用（或适用的错误）。
type FutureGetNetTotalsResult chan *response

//接收等待未来承诺的响应并返回网络
//交通统计。
func (r FutureGetNetTotalsResult) Receive() (*btcjson.GetNetTotalsResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为GetNettotals结果对象。
	var totals btcjson.GetNetTotalsResult
	err = json.Unmarshal(res, &totals)
	if err != nil {
		return nil, err
	}

	return &totals, nil
}

//GetNetTotalAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅getnettotals。
func (c *Client) GetNetTotalsAsync() FutureGetNetTotalsResult {
	cmd := btcjson.NewGetNetTotalsCmd()
	return c.sendCmd(cmd)
}

//GetNetTotals returns network traffic statistics.
func (c *Client) GetNetTotals() (*btcjson.GetNetTotalsResult, error) {
	return c.GetNetTotalsAsync().Receive()
}
