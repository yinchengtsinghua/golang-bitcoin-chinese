
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
	"io"
)

//MSGPing实现消息接口并表示比特币ping
//消息。
//
//对于bip0031及更早版本，主要用于确认
//连接仍然有效。传输错误通常是
//解释为一个关闭的连接，并且应该删除对等端。
//对于bip0031版本之后的版本，它包含一个标识符，可以
//在PONG消息中返回以确定网络时间。
//
//此消息的有效负载仅由用于标识
//后来。
type MsgPing struct {
//与用于标识的消息关联的唯一值
//特定的ping消息。
	Nonce uint64
}

//btcdecode使用比特币协议编码将r解码到接收器中。
//这是消息接口实现的一部分。
func (msg *MsgPing) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
//对于bip0031版本和更早版本，没有当前的版本。
//注意：>这里不是一个错误。bip0031的定义如下：
//这个版本与大多数其他版本不同。
	if pver > BIP0031Version {
		err := readElement(r, &msg.Nonce)
		if err != nil {
			return err
		}
	}

	return nil
}

//btcencode使用比特币协议编码将接收器编码为w。
//这是消息接口实现的一部分。
func (msg *MsgPing) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
//对于bip0031版本和更早版本，没有当前的版本。
//注意：>这里不是一个错误。bip0031的定义如下：
//这个版本与大多数其他版本不同。
	if pver > BIP0031Version {
		err := writeElement(w, msg.Nonce)
		if err != nil {
			return err
		}
	}

	return nil
}

//命令返回消息的协议命令字符串。这是一部分
//消息接口实现。
func (msg *MsgPing) Command() string {
	return CmdPing
}

//maxpayloadLength返回有效负载的最大长度
//接收器。这是消息接口实现的一部分。
func (msg *MsgPing) MaxPayloadLength(pver uint32) uint32 {
	plen := uint32(0)
//对于bip0031版本和更早版本，没有当前的版本。
//注意：>这里不是一个错误。bip0031的定义如下：
//这个版本与大多数其他版本不同。
	if pver > BIP0031Version {
//8字节。
		plen += 8
	}

	return plen
}

//newmsgping返回符合消息的新比特币ping消息
//接口。有关详细信息，请参阅MSGPing。
func NewMsgPing(nonce uint64) *MsgPing {
	return &MsgPing{
		Nonce: nonce,
	}
}
