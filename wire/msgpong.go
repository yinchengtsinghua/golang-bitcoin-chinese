
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

//msgpong实现消息接口并表示比特币pong
//主要用于确认连接仍然有效的消息
//响应比特币ping消息（msgping）。
//
//此消息在协议版本高于bip0031版本之前未添加。
type MsgPong struct {
//与用于标识的消息关联的唯一值
//特定的ping消息。
	Nonce uint64
}

//btcdecode使用比特币协议编码将r解码到接收器中。
//这是消息接口实现的一部分。
func (msg *MsgPong) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
//注意：<=这里不是一个错误。bip0031的定义如下：
//这个版本与大多数其他版本不同。
	if pver <= BIP0031Version {
		str := fmt.Sprintf("pong message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgPong.BtcDecode", str)
	}

	return readElement(r, &msg.Nonce)
}

//btcencode使用比特币协议编码将接收器编码为w。
//这是消息接口实现的一部分。
func (msg *MsgPong) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
//注意：<=这里不是一个错误。bip0031的定义如下：
//这个版本与大多数其他版本不同。
	if pver <= BIP0031Version {
		str := fmt.Sprintf("pong message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgPong.BtcEncode", str)
	}

	return writeElement(w, msg.Nonce)
}

//命令返回消息的协议命令字符串。这是一部分
//消息接口实现。
func (msg *MsgPong) Command() string {
	return CmdPong
}

//maxpayloadLength返回有效负载的最大长度
//接收器。这是消息接口实现的一部分。
func (msg *MsgPong) MaxPayloadLength(pver uint32) uint32 {
	plen := uint32(0)
//对于bip0031版本和更早版本，pong消息不存在。
//注意：>这里不是一个错误。bip0031的定义如下：
//这个版本与大多数其他版本不同。
	if pver > BIP0031Version {
//8字节。
		plen += 8
	}

	return plen
}

//newmsgpong返回符合消息的新比特币pong消息
//接口。有关详细信息，请参阅msgpong。
func NewMsgPong(nonce uint64) *MsgPong {
	return &MsgPong{
		Nonce: nonce,
	}
}
