
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

//msgverack定义了比特币verack消息，用于
//使用信息后确认版本消息（msgversion）
//协商参数。它实现了消息接口。
//
//此消息没有有效负载。
type MsgVerAck struct{}

//btcdecode使用比特币协议编码将r解码到接收器中。
//这是消息接口实现的一部分。
func (msg *MsgVerAck) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	return nil
}

//btcencode使用比特币协议编码将接收器编码为w。
//这是消息接口实现的一部分。
func (msg *MsgVerAck) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	return nil
}

//命令返回消息的协议命令字符串。这是一部分
//消息接口实现。
func (msg *MsgVerAck) Command() string {
	return CmdVerAck
}

//maxpayloadLength返回有效负载的最大长度
//接收器。这是消息接口实现的一部分。
func (msg *MsgVerAck) MaxPayloadLength(pver uint32) uint32 {
	return 0
}

//newmsgverack返回符合
//消息接口。
func NewMsgVerAck() *MsgVerAck {
	return &MsgVerAck{}
}
