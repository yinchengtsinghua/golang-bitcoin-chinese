
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

//DefaultInvlistAlloc是用于
//库存清单。数组将根据需要动态增长，但是
//图旨在为最大库存量提供足够的空间
//*典型*库存消息中不需要增加备份的向量
//数组多次。从技术上讲，这个列表可以增长到maxinvpermsg，但是
//而不是使用那个大数字，这个数字更准确地反映了
//典型案例。
const defaultInvListAlloc = 1000

//msginv实现消息接口并表示比特币inv消息。
//它用于公布对等方的已知数据，如块和事务。
//通过库存向量。可能会主动发送通知其他同行
//或响应getBlocks消息（msggetBlocks）。各
//消息仅限于最大数量的库存向量，即
//目前5万。
//
//使用addinvect函数在以下情况下建立库存向量列表：
//向另一个对等端发送INV消息。
type MsgInv struct {
	InvList []*InvVect
}

//addinvect向消息添加一个库存向量。
func (msg *MsgInv) AddInvVect(iv *InvVect) error {
	if len(msg.InvList)+1 > MaxInvPerMsg {
		str := fmt.Sprintf("too many invvect in message [max %v]",
			MaxInvPerMsg)
		return messageError("MsgInv.AddInvVect", str)
	}

	msg.InvList = append(msg.InvList, iv)
	return nil
}

//btcdecode使用比特币协议编码将r解码到接收器中。
//这是消息接口实现的一部分。
func (msg *MsgInv) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	count, err := ReadVarInt(r, pver)
	if err != nil {
		return err
	}

//限制为每条消息的最大库存向量。
	if count > MaxInvPerMsg {
		str := fmt.Sprintf("too many invvect in message [%v]", count)
		return messageError("MsgInv.BtcDecode", str)
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
func (msg *MsgInv) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
//限制为每条消息的最大库存向量。
	count := len(msg.InvList)
	if count > MaxInvPerMsg {
		str := fmt.Sprintf("too many invvect in message [%v]", count)
		return messageError("MsgInv.BtcEncode", str)
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
func (msg *MsgInv) Command() string {
	return CmdInv
}

//maxpayloadLength返回有效负载的最大长度
//接收器。这是消息接口实现的一部分。
func (msg *MsgInv) MaxPayloadLength(pver uint32) uint32 {
//num inventory vectors（varint）+允许的最大库存向量。
	return MaxVarIntPayload + (MaxInvPerMsg * maxInvVectPayload)
}

//newmsginv返回符合消息的新比特币inv消息
//接口。有关详细信息，请参阅msginv。
func NewMsgInv() *MsgInv {
	return &MsgInv{
		InvList: make([]*InvVect, 0, defaultInvListAlloc),
	}
}

//newmsginvsizehint返回符合
//消息接口。有关详细信息，请参阅msginv。此函数与
//newmsginv，因为它允许支持数组的默认分配大小
//其中包含库存向量列表。这允许知道
//提高库存清单的增长幅度以避免
//在追加大量数据时多次增大内部支持数组
//有addinvect的库存向量。注意，指定的提示只是
//这是用于默认分配大小的提示。添加更多
//（或更少）库存向量仍将正常工作。大小提示是
//仅限于Maxinvpermsg。
func NewMsgInvSizeHint(sizeHint uint) *MsgInv {
//将指定的提示限制为每条消息允许的最大值。
	if sizeHint > MaxInvPerMsg {
		sizeHint = MaxInvPerMsg
	}

	return &MsgInv{
		InvList: make([]*InvVect, 0, sizeHint),
	}
}
