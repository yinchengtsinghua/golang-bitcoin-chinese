
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

//msggetdata实现消息接口并表示比特币
//获取数据消息。它用于请求数据，如块和事务
//来自另一个同伴。它应该用于响应inv（msginv）消息
//请求每个库存向量引用的实际数据
//同伴还没有。每条消息的最大数量限制为
//库存向量，目前为50000。因此，多条消息
//必须用于请求更大数量的数据。
//
//使用addinvect函数在以下情况下建立库存向量列表：
//向另一个对等端发送getdata消息。
type MsgGetData struct {
	InvList []*InvVect
}

//addinvect向消息添加一个库存向量。
func (msg *MsgGetData) AddInvVect(iv *InvVect) error {
	if len(msg.InvList)+1 > MaxInvPerMsg {
		str := fmt.Sprintf("too many invvect in message [max %v]",
			MaxInvPerMsg)
		return messageError("MsgGetData.AddInvVect", str)
	}

	msg.InvList = append(msg.InvList, iv)
	return nil
}

//btcdecode使用比特币协议编码将r解码到接收器中。
//这是消息接口实现的一部分。
func (msg *MsgGetData) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	count, err := ReadVarInt(r, pver)
	if err != nil {
		return err
	}

//限制为每条消息的最大库存向量。
	if count > MaxInvPerMsg {
		str := fmt.Sprintf("too many invvect in message [%v]", count)
		return messageError("MsgGetData.BtcDecode", str)
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
func (msg *MsgGetData) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
//限制为每条消息的最大库存向量。
	count := len(msg.InvList)
	if count > MaxInvPerMsg {
		str := fmt.Sprintf("too many invvect in message [%v]", count)
		return messageError("MsgGetData.BtcEncode", str)
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
func (msg *MsgGetData) Command() string {
	return CmdGetData
}

//maxpayloadLength返回有效负载的最大长度
//接收器。这是消息接口实现的一部分。
func (msg *MsgGetData) MaxPayloadLength(pver uint32) uint32 {
//num inventory vectors（varint）+允许的最大库存向量。
	return MaxVarIntPayload + (MaxInvPerMsg * maxInvVectPayload)
}

//newmsggetdata返回符合
//消息接口。有关详细信息，请参阅msggetdata。
func NewMsgGetData() *MsgGetData {
	return &MsgGetData{
		InvList: make([]*InvVect, 0, defaultInvListAlloc),
	}
}

//newmsggetdatasizehint返回符合以下条件的新比特币getdata消息
//消息接口。有关详细信息，请参阅msggetdata。此功能不同
//从newmsggetdata，因为它允许
//存储库存向量列表的支持数组。这允许呼叫者
//谁预先知道库存清单将增长到多大以避免
//追加时多次增加内部后备数组的开销
//大量带有addinvect的库存向量。请注意，指定的
//提示只是-用于默认分配大小的提示。
//添加更多（或更少）库存向量仍然可以正常工作。尺寸
//提示仅限于maxinvpermsg。
func NewMsgGetDataSizeHint(sizeHint uint) *MsgGetData {
//将指定的提示限制为每条消息允许的最大值。
	if sizeHint > MaxInvPerMsg {
		sizeHint = MaxInvPerMsg
	}

	return &MsgGetData{
		InvList: make([]*InvVect, 0, sizeHint),
	}
}
