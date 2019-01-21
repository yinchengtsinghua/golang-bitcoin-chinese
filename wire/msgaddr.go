
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

//maxaddrpermsg是单个地址中的最大地址数。
//比特币地址信息（msgaddr）。
const MaxAddrPerMsg = 1000

//msgaddr实现消息接口并表示比特币
//ADDR消息。它用于提供
//网络。活动对等机被认为是传输了消息的对等机。
//在过去3小时内。当时未传输的节点
//框架应该被遗忘。每条消息的最大数量限制为
//地址，当前为1000。因此，多条消息必须
//用于转发完整列表。
//
//当
//向另一个对等端发送addr消息。
type MsgAddr struct {
	AddrList []*NetAddress
}

//addaddress向消息添加已知的活动对等点。
func (msg *MsgAddr) AddAddress(na *NetAddress) error {
	if len(msg.AddrList)+1 > MaxAddrPerMsg {
		str := fmt.Sprintf("too many addresses in message [max %v]",
			MaxAddrPerMsg)
		return messageError("MsgAddr.AddAddress", str)
	}

	msg.AddrList = append(msg.AddrList, na)
	return nil
}

//addaddresses向消息添加多个已知的活动对等点。
func (msg *MsgAddr) AddAddresses(netAddrs ...*NetAddress) error {
	for _, na := range netAddrs {
		err := msg.AddAddress(na)
		if err != nil {
			return err
		}
	}
	return nil
}

//ClearAddresses从消息中删除所有地址。
func (msg *MsgAddr) ClearAddresses() {
	msg.AddrList = []*NetAddress{}
}

//btcdecode使用比特币协议编码将r解码到接收器中。
//这是消息接口实现的一部分。
func (msg *MsgAddr) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	count, err := ReadVarInt(r, pver)
	if err != nil {
		return err
	}

//限制为每条消息的最大地址。
	if count > MaxAddrPerMsg {
		str := fmt.Sprintf("too many addresses for message "+
			"[count %v, max %v]", count, MaxAddrPerMsg)
		return messageError("MsgAddr.BtcDecode", str)
	}

	addrList := make([]NetAddress, count)
	msg.AddrList = make([]*NetAddress, 0, count)
	for i := uint64(0); i < count; i++ {
		na := &addrList[i]
		err := readNetAddress(r, pver, na, true)
		if err != nil {
			return err
		}
		msg.AddAddress(na)
	}
	return nil
}

//btcencode使用比特币协议编码将接收器编码为w。
//这是消息接口实现的一部分。
func (msg *MsgAddr) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
//多线程版本之前的协议版本仅允许1个地址
//每个消息。
	count := len(msg.AddrList)
	if pver < MultipleAddressVersion && count > 1 {
		str := fmt.Sprintf("too many addresses for message of "+
			"protocol version %v [count %v, max 1]", pver, count)
		return messageError("MsgAddr.BtcEncode", str)

	}
	if count > MaxAddrPerMsg {
		str := fmt.Sprintf("too many addresses for message "+
			"[count %v, max %v]", count, MaxAddrPerMsg)
		return messageError("MsgAddr.BtcEncode", str)
	}

	err := WriteVarInt(w, pver, uint64(count))
	if err != nil {
		return err
	}

	for _, na := range msg.AddrList {
		err = writeNetAddress(w, pver, na, true)
		if err != nil {
			return err
		}
	}

	return nil
}

//命令返回消息的协议命令字符串。这是一部分
//消息接口实现。
func (msg *MsgAddr) Command() string {
	return CmdAddr
}

//maxpayloadLength返回有效负载的最大长度
//接收器。这是消息接口实现的一部分。
func (msg *MsgAddr) MaxPayloadLength(pver uint32) uint32 {
	if pver < MultipleAddressVersion {
//num addresses（varint）+单个网络地址。
		return MaxVarIntPayload + maxNetAddressPayload(pver)
	}

//num addresses（varint）+允许的最大地址。
	return MaxVarIntPayload + (MaxAddrPerMsg * maxNetAddressPayload(pver))
}

//newmsgaddr返回符合
//消息接口。有关详细信息，请参阅msgaddr。
func NewMsgAddr() *MsgAddr {
	return &MsgAddr{
		AddrList: make([]*NetAddress, 0, MaxAddrPerMsg),
	}
}
