
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
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"
)

//maxuseragentlen是用户代理字段在
//版本消息（msgversion）。
const MaxUserAgentLen = 256

//堆栈中连线的默认用户代理
const DefaultUserAgent = "/btcwire:0.5.0/"

//msgversion实现消息接口并表示比特币版本
//消息。它用于对等端在出站时立即进行自我宣传
//已建立连接。然后远程对等机将此信息与
//它自己谈判。然后，远程对等机必须用版本响应
//包含协商值和verack的消息
//消息（msgverack）。这种交换必须在任何进一步之前进行
//允许继续通信。
type MsgVersion struct {
//节点使用的协议版本。
	ProtocolVersion int32

//标识已启用服务的位字段。
	Services ServiceFlag

//生成消息的时间。这在线路上编码为Int64。
	Timestamp time.Time

//远程对等机的地址。
	AddrYou NetAddress

//本地对等机的地址。
	AddrMe NetAddress

//与用于检测自身的消息关联的唯一值
//连接。
	Nonce uint64

//生成message的用户代理。这是一个编码为varstring的
//在电线上。它的最大长度为maxuseragentlen。
	UserAgent string

//版本消息生成器看到的最后一个块。
	LastBlock int32

//不要向对等端宣布事务。
	DisableRelayTx bool
}

//HASSERVICE返回对等端是否支持指定的服务
//生成了消息。
func (msg *MsgVersion) HasService(service ServiceFlag) bool {
	return msg.Services&service == service
}

//addService通过生成
//消息。
func (msg *MsgVersion) AddService(service ServiceFlag) {
	msg.Services |= service
}

//btcdecode使用比特币协议编码将r解码到接收器中。
//版本消息是特殊的，因为协议版本没有
//谈判了。因此，将忽略pver字段和
//在新版本中添加是可选的。这也意味着r必须是
//*bytes.buffer，以便确定剩余字节数。
//
//这是消息接口实现的一部分。
func (msg *MsgVersion) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	buf, ok := r.(*bytes.Buffer)
	if !ok {
		return fmt.Errorf("MsgVersion.BtcDecode reader is not a " +
			"*bytes.Buffer")
	}

	err := readElements(buf, &msg.ProtocolVersion, &msg.Services,
		(*int64Time)(&msg.Timestamp))
	if err != nil {
		return err
	}

	err = readNetAddress(buf, pver, &msg.AddrYou, false)
	if err != nil {
		return err
	}

//协议版本>=106添加了发件人地址、nonce和用户代理
//字段，只有在存在字节时才认为它们存在
//保留在邮件中。
	if buf.Len() > 0 {
		err = readNetAddress(buf, pver, &msg.AddrMe, false)
		if err != nil {
			return err
		}
	}
	if buf.Len() > 0 {
		err = readElement(buf, &msg.Nonce)
		if err != nil {
			return err
		}
	}
	if buf.Len() > 0 {
		userAgent, err := ReadVarString(buf, pver)
		if err != nil {
			return err
		}
		err = validateUserAgent(userAgent)
		if err != nil {
			return err
		}
		msg.UserAgent = userAgent
	}

//协议版本>=209添加了最后一个已知的块字段。它只是
//如果消息中还有字节，则认为存在。
	if buf.Len() > 0 {
		err = readElement(buf, &msg.LastBlock)
		if err != nil {
			return err
		}
	}

//在bip0037版本之前没有中继事务字段，但是
//添加字段之前的默认行为是
//中继事务。
	if buf.Len() > 0 {
//忽略这里的错误是安全的，因为缓冲区
//至少一个字节，该字节将产生一个布尔值
//不管它的价值如何。另外，用于
//当应中继事务时，字段为true，因此反转
//它用于DisableRelayTx字段。
		var relayTx bool
		readElement(r, &relayTx)
		msg.DisableRelayTx = !relayTx
	}

	return nil
}

//btcencode使用比特币协议编码将接收器编码为w。
//这是消息接口实现的一部分。
func (msg *MsgVersion) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	err := validateUserAgent(msg.UserAgent)
	if err != nil {
		return err
	}

	err = writeElements(w, msg.ProtocolVersion, msg.Services,
		msg.Timestamp.Unix())
	if err != nil {
		return err
	}

	err = writeNetAddress(w, pver, &msg.AddrYou, false)
	if err != nil {
		return err
	}

	err = writeNetAddress(w, pver, &msg.AddrMe, false)
	if err != nil {
		return err
	}

	err = writeElement(w, msg.Nonce)
	if err != nil {
		return err
	}

	err = WriteVarString(w, pver, msg.UserAgent)
	if err != nil {
		return err
	}

	err = writeElement(w, msg.LastBlock)
	if err != nil {
		return err
	}

//在bip0037版本之前没有中继事务字段。也，
//当事务应为
//已中继，因此将其从DisableRelayTx字段中反转。
	if pver >= BIP0037Version {
		err = writeElement(w, !msg.DisableRelayTx)
		if err != nil {
			return err
		}
	}
	return nil
}

//命令返回消息的协议命令字符串。这是一部分
//消息接口实现。
func (msg *MsgVersion) Command() string {
	return CmdVersion
}

//maxpayloadLength返回有效负载的最大长度
//接收器。这是消息接口实现的一部分。
func (msg *MsgVersion) MaxPayloadLength(pver uint32) uint32 {
//XXX：<=106不同

//协议版本4字节+服务8字节+时间戳8字节+
//远程和本地网络地址+nonce 8字节+用户长度
//代理（变量）+允许的最大用户代理长度+最后一个块4字节+
//中继事务标志1字节。
	return 33 + (maxNetAddressPayload(pver) * 2) + MaxVarIntPayload +
		MaxUserAgentLen
}

//newmsgversion返回符合
//消息接口，使用传递的参数和其余的默认值
//领域。
func NewMsgVersion(me *NetAddress, you *NetAddress, nonce uint64,
	lastBlock int32) *MsgVersion {

//将时间戳限制为自协议以来的一秒精度
//不支持更好。
	return &MsgVersion{
		ProtocolVersion: int32(ProtocolVersion),
		Services:        0,
		Timestamp:       time.Unix(time.Now().Unix(), 0),
		AddrYou:         *you,
		AddrMe:          *me,
		Nonce:           nonce,
		UserAgent:       DefaultUserAgent,
		LastBlock:       lastBlock,
		DisableRelayTx:  false,
	}
}

//validateUserAgent根据maxUserAgentLen检查UserAgent长度
func validateUserAgent(userAgent string) error {
	if len(userAgent) > MaxUserAgentLen {
		str := fmt.Sprintf("user agent too long [len %v, max %v]",
			len(userAgent), MaxUserAgentLen)
		return messageError("MsgVersion", str)
	}
	return nil
}

//adduseragent将用户代理添加到版本的用户代理字符串中
//消息。版本字符串没有定义为任何严格的格式，尽管
//建议使用“主要、次要、修订”格式，例如“2.6.41”。
func (msg *MsgVersion) AddUserAgent(name string, version string,
	comments ...string) error {

	newUserAgent := fmt.Sprintf("%s:%s", name, version)
	if len(comments) != 0 {
		newUserAgent = fmt.Sprintf("%s(%s)", newUserAgent,
			strings.Join(comments, "; "))
	}
	newUserAgent = fmt.Sprintf("%s%s/", msg.UserAgent, newUserAgent)
	err := validateUserAgent(newUserAgent)
	if err != nil {
		return err
	}
	msg.UserAgent = newUserAgent
	return nil
}
