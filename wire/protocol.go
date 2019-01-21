
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wire

import (
	"fmt"
	"strconv"
	"strings"
)

//佩德罗：我们可能需要把这个撞上。
const (
//ProtocolVersion是此包支持的最新协议版本。
	ProtocolVersion uint32 = 70013

//multipleaddressversion是添加了多个
//每条消息的地址（pver>=multipleaddressversion）。
	MultipleAddressVersion uint32 = 209

//NetAddressTimeVersion是添加了
//时间戳字段（pver>=netAddressTimeVersion）。
	NetAddressTimeVersion uint32 = 31402

//bip0031版本是一个协议版本，在该协议版本之后，pong消息
//在Ping中添加了nonce字段（pver>bip0031版本）。
	BIP0031Version uint32 = 60000

//bip0035版本是添加mempool的协议版本
//消息（pver>=bip0035版本）。
	BIP0035Version uint32 = 60002

//bip0037版本是添加新连接的协议版本
//Bloom过滤相关消息并扩展版本消息
//带有中继标志（pver>=bip0037版本）。
	BIP0037Version uint32 = 70001

//RejectVersion是添加新拒绝的协议版本
//消息。
	RejectVersion uint32 = 70002

//bip0111版本是添加sfnodebloom的协议版本
//服务标志。
	BIP0111Version uint32 = 70011

//sendHeadersVersion是添加了新的
//sendHeaders消息。
	SendHeadersVersion uint32 = 70012

//feefilterversion是添加了新的
//F过滤器信息。
	FeeFilterVersion uint32 = 70013
)

//ServiceFlag标识比特币对等方支持的服务。
type ServiceFlag uint64

const (
//sfnodenetwork是一个标志，用于指示对等机是一个完整的节点。
	SFNodeNetwork ServiceFlag = 1 << iota

//sfnodegetuxo是一个标志，用于指示对等机支持
//getutxos和utxos命令（bip0064）。
	SFNodeGetUTXO

//sfnodebloom是用于指示对等机支持bloom的标志。
//过滤。
	SFNodeBloom

//sfnodewitness是用于指示对等支持块的标志。
//以及包括见证数据在内的交易（bip0144）。
	SFNodeWitness

//sfnodexthin是用于指示对等机支持xthin块的标志。
	SFNodeXthin

//sfnodebit5是一个标志，用于指示对等机支持服务
//由第5位定义。
	SFNodeBit5

//sfnodecf是一个标志，用于指示已提交对等支持
//过滤器（CFS）。
	SFNodeCF

//sfnode2x是用于指示对等机正在运行segwit2x的标志。
//软件。
	SFNode2X
)

//将服务标志映射回其常量名称，以便进行漂亮的打印。
var sfStrings = map[ServiceFlag]string{
	SFNodeNetwork: "SFNodeNetwork",
	SFNodeGetUTXO: "SFNodeGetUTXO",
	SFNodeBloom:   "SFNodeBloom",
	SFNodeWitness: "SFNodeWitness",
	SFNodeXthin:   "SFNodeXthin",
	SFNodeBit5:    "SFNodeBit5",
	SFNodeCF:      "SFNodeCF",
	SFNode2X:      "SFNode2X",
}

//orderedsfstrigs是从最高到
//最低的。
var orderedSFStrings = []ServiceFlag{
	SFNodeNetwork,
	SFNodeGetUTXO,
	SFNodeBloom,
	SFNodeWitness,
	SFNodeXthin,
	SFNodeBit5,
	SFNodeCF,
	SFNode2X,
}

//字符串以可读形式返回ServiceFlag。
func (f ServiceFlag) String() string {
//未设置标志。
	if f == 0 {
		return "0x0"
	}

//添加单个位标志。
	s := ""
	for _, flag := range orderedSFStrings {
		if f&flag == flag {
			s += sfStrings[flag] + "|"
			f -= flag
		}
	}

//添加任何不作为十六进制计算的剩余标志。
	s = strings.TrimRight(s, "|")
	if f != 0 {
		s += "|0x" + strconv.FormatUint(uint64(f), 16)
	}
	s = strings.TrimLeft(s, "|")
	return s
}

//比特币网络表示消息所属的比特币网络。
type BitcoinNet uint32

//用于指示消息比特币网络的常量。他们也可以
//用于在流的状态未知时查找下一条消息，但
//此包不提供该功能，因为它通常是
//更好的办法是简单地断开那些在TCP上行为不端的客户机。
const (
//mainnet代表主要比特币网络。
	MainNet BitcoinNet = 0xd9b4bef9

//testnet表示回归测试网络。
	TestNet BitcoinNet = 0xdab5bffa

//TestNet3表示测试网络（版本3）。
	TestNet3 BitcoinNet = 0x0709110b

//simnet表示模拟测试网络。
	SimNet BitcoinNet = 0x12141c16
)

//bnstrings是比特币网络的映射，返回到其常量名称
//印刷精美。
var bnStrings = map[BitcoinNet]string{
	MainNet:  "MainNet",
	TestNet:  "TestNet",
	TestNet3: "TestNet3",
	SimNet:   "SimNet",
}

//字符串以人类可读的形式返回比特币。
func (n BitcoinNet) String() string {
	if s, ok := bnStrings[n]; ok {
		return s
	}

	return fmt.Sprintf("Unknown BitcoinNet (%d)", uint32(n))
}
