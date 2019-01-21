
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2015 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wire

import (
	"fmt"
	"io"
)

//msgmempool实现消息接口并表示比特币内存池
//消息。它用于请求仍处于活动状态的事务列表
//中继的内存池。
//
//此消息没有有效负载，在协议版本之前未添加。
//从Bip0035版本开始。
type MsgMemPool struct{}

//btcdecode使用比特币协议编码将r解码到接收器中。
//这是消息接口实现的一部分。
func (msg *MsgMemPool) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	if pver < BIP0035Version {
		str := fmt.Sprintf("mempool message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgMemPool.BtcDecode", str)
	}

	return nil
}

//btcencode使用比特币协议编码将接收器编码为w。
//这是消息接口实现的一部分。
func (msg *MsgMemPool) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	if pver < BIP0035Version {
		str := fmt.Sprintf("mempool message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgMemPool.BtcEncode", str)
	}

	return nil
}

//命令返回消息的协议命令字符串。这是一部分
//消息接口实现。
func (msg *MsgMemPool) Command() string {
	return CmdMemPool
}

//maxpayloadLength返回有效负载的最大长度
//接收器。这是消息接口实现的一部分。
func (msg *MsgMemPool) MaxPayloadLength(pver uint32) uint32 {
	return 0
}

//newmsgmempool返回符合消息的新比特币pong消息
//接口。有关详细信息，请参阅msgpong。
func NewMsgMemPool() *MsgMemPool {
	return &MsgMemPool{}
}
