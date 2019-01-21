
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

//msgnotfound定义了一个比特币未找到消息，该消息作为响应发送到
//如果对等机上没有任何请求的数据，则返回getdata消息。
//每个消息都限制在最大数量的库存向量上，即
//目前5万。
//
//使用addinvect函数在以下情况下建立库存向量列表：
//向另一个对等端发送未找到的消息。
type MsgNotFound struct {
	InvList []*InvVect
}

//addinvect向消息添加一个库存向量。
func (msg *MsgNotFound) AddInvVect(iv *InvVect) error {
	if len(msg.InvList)+1 > MaxInvPerMsg {
		str := fmt.Sprintf("too many invvect in message [max %v]",
			MaxInvPerMsg)
		return messageError("MsgNotFound.AddInvVect", str)
	}

	msg.InvList = append(msg.InvList, iv)
	return nil
}

//btcdecode使用比特币协议编码将r解码到接收器中。
//这是消息接口实现的一部分。
func (msg *MsgNotFound) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	count, err := ReadVarInt(r, pver)
	if err != nil {
		return err
	}

//限制为每条消息的最大库存向量。
	if count > MaxInvPerMsg {
		str := fmt.Sprintf("too many invvect in message [%v]", count)
		return messageError("MsgNotFound.BtcDecode", str)
	}

//创建库存向量的连续切片，以便在中反序列化为
//以减少分配的数量。
	invList := make([]InvVect, count)
	msg.InvList = make([]*InvVect, 0, count)
	for i := uint64(0); i < count; i++ {
		iv := &invList[i]
		err := readInvVect(r, pver, iv)
		if err != nil {
			return err
		}
		msg.AddInvVect(iv)
	}

	return nil
}

//btcencode使用比特币协议编码将接收器编码为w。
//这是消息接口实现的一部分。
func (msg *MsgNotFound) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
//限制为每条消息的最大库存向量。
	count := len(msg.InvList)
	if count > MaxInvPerMsg {
		str := fmt.Sprintf("too many invvect in message [%v]", count)
		return messageError("MsgNotFound.BtcEncode", str)
	}

	err := WriteVarInt(w, pver, uint64(count))
	if err != nil {
		return err
	}

	for _, iv := range msg.InvList {
		err := writeInvVect(w, pver, iv)
		if err != nil {
			return err
		}
	}

	return nil
}

//命令返回消息的协议命令字符串。这是一部分
//消息接口实现。
func (msg *MsgNotFound) Command() string {
	return CmdNotFound
}

//maxpayloadLength返回有效负载的最大长度
//接收器。这是消息接口实现的一部分。
func (msg *MsgNotFound) MaxPayloadLength(pver uint32) uint32 {
//最大var int 9字节+最大invvect，每个36字节。
//num inventory vectors（varint）+允许的最大库存向量。
	return MaxVarIntPayload + (MaxInvPerMsg * maxInvVectPayload)
}

//newmsgnotfound返回符合
//消息接口。有关详细信息，请参阅msgnotfound。
func NewMsgNotFound() *MsgNotFound {
	return &MsgNotFound{
		InvList: make([]*InvVect, 0, defaultInvListAlloc),
	}
}
