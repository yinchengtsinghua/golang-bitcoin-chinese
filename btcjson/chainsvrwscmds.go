
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2014-2017 BTCSuite开发者
//版权所有（c）2015-2017法令开发商
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

//注意：此文件用于存放受支持的rpc命令
//链服务器，但只能通过WebSockets提供。

package btcjson

//authenticateCmd定义authenticate json-rpc命令。
type AuthenticateCmd struct {
	Username   string
	Passphrase string
}

//NewAuthenticateCmd返回一个可用于发出
//验证json-rpc命令。
func NewAuthenticateCmd(username, passphrase string) *AuthenticateCmd {
	return &AuthenticateCmd{
		Username:   username,
		Passphrase: passphrase,
	}
}

//notifyblockscmd定义notifyblocks json-rpc命令。
type NotifyBlocksCmd struct{}

//newnotifyblockscmd返回可用于发出
//notifyblocks json-rpc命令。
func NewNotifyBlocksCmd() *NotifyBlocksCmd {
	return &NotifyBlocksCmd{}
}

//stopnotifyblockscmd定义stopnotifyblocks json-rpc命令。
type StopNotifyBlocksCmd struct{}

//newstopNotifyBlocksCmd返回可用于发出
//stopnotifyblocks json-rpc命令。
func NewStopNotifyBlocksCmd() *StopNotifyBlocksCmd {
	return &StopNotifyBlocksCmd{}
}

//notifynewtransactionsCmd定义notifynewtransactions json-rpc命令。
type NotifyNewTransactionsCmd struct {
	Verbose *bool `jsonrpcdefault:"false"`
}

//newnotifynewtransactionsCmd返回可用于发出的新实例
//notifynewtransactions json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewNotifyNewTransactionsCmd(verbose *bool) *NotifyNewTransactionsCmd {
	return &NotifyNewTransactionsCmd{
		Verbose: verbose,
	}
}

//sessionCmd定义session json-rpc命令。
type SessionCmd struct{}

//newsessionCmd返回可用于发出会话的新实例
//json-rpc命令。
func NewSessionCmd() *SessionCmd {
	return &SessionCmd{}
}

//stopnotifynewtransactionsCmd定义stopnotifynewtransactions json-rpc命令。
type StopNotifyNewTransactionsCmd struct{}

//newstopnotifynewtransactionsCmd返回可用于发出的新实例
//stopnotifynewtransactions json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewStopNotifyNewTransactionsCmd() *StopNotifyNewTransactionsCmd {
	return &StopNotifyNewTransactionsCmd{}
}

//notifyreceivedcmd定义notifyreceived json-rpc命令。
//
//注意：已弃用。改用loadtxfilterCmd。
type NotifyReceivedCmd struct {
	Addresses []string
}

//newnotifyreceivedcmd返回可用于发出
//notifyreceived json-rpc命令。
//
//注意：已弃用。请改用newloadtxfiltercmd。
func NewNotifyReceivedCmd(addresses []string) *NotifyReceivedCmd {
	return &NotifyReceivedCmd{
		Addresses: addresses,
	}
}

//Outpoint描述将被编组到和的事务Outpoint。
//来自JSON。
type OutPoint struct {
	Hash  string `json:"hash"`
	Index uint32 `json:"index"`
}

//loadtxfilterCmd定义要加载或
//重新加载事务筛选器。
//
//注意：这是从github.com/decred/dcrd/dcrjson导入的BTCD扩展
//需要WebSocket连接。
type LoadTxFilterCmd struct {
	Reload    bool
	Addresses []string
	OutPoints []OutPoint
}

//newloadtxfilterCmd返回可用于发出
//loadtxfilter json-rpc命令。
//
//注意：这是从github.com/decred/dcrd/dcrjson导入的BTCD扩展
//需要WebSocket连接。
func NewLoadTxFilterCmd(reload bool, addresses []string, outPoints []OutPoint) *LoadTxFilterCmd {
	return &LoadTxFilterCmd{
		Reload:    reload,
		Addresses: addresses,
		OutPoints: outPoints,
	}
}

//notifyspendetcmd定义notifyspended json-rpc命令。
//
//注意：已弃用。改用loadtxfilterCmd。
type NotifySpentCmd struct {
	OutPoints []OutPoint
}

//NewNotifySpentcmd返回一个可用于发出
//notifyspeed json-rpc命令。
//
//注意：已弃用。请改用newloadtxfiltercmd。
func NewNotifySpentCmd(outPoints []OutPoint) *NotifySpentCmd {
	return &NotifySpentCmd{
		OutPoints: outPoints,
	}
}

//stopnotifyreceivedcmd定义stopnotifyreceived json-rpc命令。
//
//注意：已弃用。改用loadtxfilterCmd。
type StopNotifyReceivedCmd struct {
	Addresses []string
}

//newstopnotifyreceivedcmd返回可用于发出
//StopNotifyReceived JSON-RPC命令。
//
//注意：已弃用。请改用newloadtxfiltercmd。
func NewStopNotifyReceivedCmd(addresses []string) *StopNotifyReceivedCmd {
	return &StopNotifyReceivedCmd{
		Addresses: addresses,
	}
}

//stopnotifyspended定义stopnotifyspended json-rpc命令。
//
//注意：已弃用。改用loadtxfilterCmd。
type StopNotifySpentCmd struct {
	OutPoints []OutPoint
}

//newstopNotifySpentcmd返回可用于发出
//stopnotifyspended json-rpc命令。
//
//注意：已弃用。请改用newloadtxfiltercmd。
func NewStopNotifySpentCmd(outPoints []OutPoint) *StopNotifySpentCmd {
	return &StopNotifySpentCmd{
		OutPoints: outPoints,
	}
}

//rescancmd定义rescan json-rpc命令。
//
//注意：已弃用。请改用rescanblockscmd。
type RescanCmd struct {
	BeginBlock string
	Addresses  []string
	OutPoints  []OutPoint
	EndBlock   *string
}

//NewRescanCmd返回可用于发出重新扫描的新实例
//json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
//
//注意：已弃用。请改用newrescanblockscmd。
func NewRescanCmd(beginBlock string, addresses []string, outPoints []OutPoint, endBlock *string) *RescanCmd {
	return &RescanCmd{
		BeginBlock: beginBlock,
		Addresses:  addresses,
		OutPoints:  outPoints,
		EndBlock:   endBlock,
	}
}

//rescanblockscmd定义rescan json-rpc命令。
//
//注意：这是从github.com/decred/dcrd/dcrjson导入的BTCD扩展
//需要WebSocket连接。
type RescanBlocksCmd struct {
//块散列作为字符串数组。
	BlockHashes []string
}

//NewRescanBlocksCmd返回可用于发出重新扫描的新实例
//json-rpc命令。
//
//注意：这是从github.com/decred/dcrd/dcrjson导入的BTCD扩展
//需要WebSocket连接。
func NewRescanBlocksCmd(blockHashes []string) *RescanBlocksCmd {
	return &RescanBlocksCmd{BlockHashes: blockHashes}
}

func init() {
//此文件中的命令只能由WebSockets使用。
	flags := UFWebsocketOnly

	MustRegisterCmd("authenticate", (*AuthenticateCmd)(nil), flags)
	MustRegisterCmd("loadtxfilter", (*LoadTxFilterCmd)(nil), flags)
	MustRegisterCmd("notifyblocks", (*NotifyBlocksCmd)(nil), flags)
	MustRegisterCmd("notifynewtransactions", (*NotifyNewTransactionsCmd)(nil), flags)
	MustRegisterCmd("notifyreceived", (*NotifyReceivedCmd)(nil), flags)
	MustRegisterCmd("notifyspent", (*NotifySpentCmd)(nil), flags)
	MustRegisterCmd("session", (*SessionCmd)(nil), flags)
	MustRegisterCmd("stopnotifyblocks", (*StopNotifyBlocksCmd)(nil), flags)
	MustRegisterCmd("stopnotifynewtransactions", (*StopNotifyNewTransactionsCmd)(nil), flags)
	MustRegisterCmd("stopnotifyspent", (*StopNotifySpentCmd)(nil), flags)
	MustRegisterCmd("stopnotifyreceived", (*StopNotifyReceivedCmd)(nil), flags)
	MustRegisterCmd("rescan", (*RescanCmd)(nil), flags)
	MustRegisterCmd("rescanblocks", (*RescanBlocksCmd)(nil), flags)
}
