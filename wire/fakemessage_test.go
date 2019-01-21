
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

import "io"

//fakemessage实现消息接口并用于强制编码
//邮件中有错误。
type fakeMessage struct {
	command        string
	payload        []byte
	forceEncodeErr bool
	forceLenErr    bool
}

//btcdecode什么都不做。它只是满足了电报。
//接口。
func (msg *fakeMessage) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	return nil
}

//btcencode写入假消息的有效负载字段或强制出错
//如果设置了假消息的forceEncodeerr标志。它还满足
//Wire.Message接口。
func (msg *fakeMessage) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	if msg.forceEncodeErr {
		err := &MessageError{
			Func:        "fakeMessage.BtcEncode",
			Description: "intentional error",
		}
		return err
	}

	_, err := w.Write(msg.payload)
	return err
}

//命令返回假消息的命令字段并满足
//消息接口。
func (msg *fakeMessage) Command() string {
	return msg.command
}

//maxpayloadLength返回假消息的有效负载字段的长度
//如果设置了假消息的forcelenerr标志，则为较小的值。它
//满足消息接口。
func (msg *fakeMessage) MaxPayloadLength(pver uint32) uint32 {
	lenp := uint32(len(msg.payload))
	if msg.forceLenErr {
		return lenp - 1
	}

	return lenp
}
