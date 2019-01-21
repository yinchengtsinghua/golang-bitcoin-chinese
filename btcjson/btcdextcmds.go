
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2014-2016 BTCSuite开发者
//版权所有（c）2015-2016法令开发商
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

//注意：此文件用于存放受支持的rpc命令
//具有BTCD扩展名的链服务器。

package btcjson

//nodesubcmd定义在addnode json-rpc命令中用于
//子命令字段。
type NodeSubCmd string

const (
//nconnect指示应连接到的指定主机。
	NConnect NodeSubCmd = "connect"

//remove表示应作为
//持久对等。
	NRemove NodeSubCmd = "remove"

//NDISConnect指示应断开指定的对等机的连接。
	NDisconnect NodeSubCmd = "disconnect"
)

//nodeCmd定义dropnode json-rpc命令。
type NodeCmd struct {
	SubCmd        NodeSubCmd `jsonrpcusage:"\"connect|remove|disconnect\""`
	Target        string
	ConnectSubCmd *string `jsonrpcusage:"\"perm|temp\""`
}

//
//json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewNodeCmd(subCmd NodeSubCmd, target string, connectSubCmd *string) *NodeCmd {
	return &NodeCmd{
		SubCmd:        subCmd,
		Target:        target,
		ConnectSubCmd: connectSubCmd,
	}
}

//debuglevelCmd定义debuglevel json-rpc命令。此命令不是
//标准比特币命令。它是BTCD的扩展。
type DebugLevelCmd struct {
	LevelSpec string
}

//newdebuglevelCmd返回一个新的debuglevelCmd，可用于发出
//调试json-rpc命令。此命令不是标准比特币命令。
//它是BTCD的扩展。
func NewDebugLevelCmd(levelSpec string) *DebugLevelCmd {
	return &DebugLevelCmd{
		LevelSpec: levelSpec,
	}
}

//generatecmd定义generate json-rpc命令。
type GenerateCmd struct {
	NumBlocks uint32
}

//NeNeGeATECMD返回一个新的实例，用于生成一个生成
//json-rpc命令。
func NewGenerateCmd(numBlocks uint32) *GenerateCmd {
	return &GenerateCmd{
		NumBlocks: numBlocks,
	}
}

//getbestblockCmd定义getbestblock json-rpc命令。
type GetBestBlockCmd struct{}

//newgetbestblockCmd返回一个可用于发出
//getbestblock json-rpc命令。
func NewGetBestBlockCmd() *GetBestBlockCmd {
	return &GetBestBlockCmd{}
}

//getcurrentnetcmd定义getcurrentnet json-rpc命令。
type GetCurrentNetCmd struct{}

//newgetcurrentnetcmd返回可用于发出
//getcurrentnet json-rpc命令。
func NewGetCurrentNetCmd() *GetCurrentNetCmd {
	return &GetCurrentNetCmd{}
}

//getheadersCmd定义getheaders json-rpc命令。
//
//注意：这是从中导入的BTCSuite扩展
//github.com/decred/dcrd/dcrjson。
type GetHeadersCmd struct {
	BlockLocators []string `json:"blocklocators"`
	HashStop      string   `json:"hashstop"`
}

//
//getheaders json-rpc命令。
//
//注意：这是从中导入的BTCSuite扩展
//github.com/decred/dcrd/dcrjson。
func NewGetHeadersCmd(blockLocators []string, hashStop string) *GetHeadersCmd {
	return &GetHeadersCmd{
		BlockLocators: blockLocators,
		HashStop:      hashStop,
	}
}

//versionCmd定义版本json-rpc命令。
//
//注意：这是从中导入的BTCSuite扩展
//github.com/decred/dcrd/dcrjson。
type VersionCmd struct{}

//newversionCmd返回可用于发出JSON-RPC的新实例
//version command.
//
//注意：这是从中导入的BTCSuite扩展
//github.com/decred/dcrd/dcrjson。
func NewVersionCmd() *VersionCmd { return new(VersionCmd) }

func init() {
//此文件中的命令没有特殊标志。
	flags := UsageFlag(0)

	MustRegisterCmd("debuglevel", (*DebugLevelCmd)(nil), flags)
	MustRegisterCmd("node", (*NodeCmd)(nil), flags)
	MustRegisterCmd("generate", (*GenerateCmd)(nil), flags)
	MustRegisterCmd("getbestblock", (*GetBestBlockCmd)(nil), flags)
	MustRegisterCmd("getcurrentnet", (*GetCurrentNetCmd)(nil), flags)
	MustRegisterCmd("getheaders", (*GetHeadersCmd)(nil), flags)
	MustRegisterCmd("version", (*VersionCmd)(nil), flags)
}
