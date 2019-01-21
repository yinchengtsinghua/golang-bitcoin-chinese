
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
	"fmt"
	"io"
)

//MaxBlockHeadersPermsg是可以位于
//一个比特币头信息。
const MaxBlockHeadersPerMsg = 2000

//MSgheaders实现消息接口并表示比特币头
//消息。它用于响应传递块头信息
//到getheaders消息（msggetaders）。块头的最大数目
//每条消息当前为2000。有关请求的详细信息，请参阅msggetages。
//标题。
type MsgHeaders struct {
	Headers []*BlockHeader
}

//AddBlockHeader向消息添加新的块头。
func (msg *MsgHeaders) AddBlockHeader(bh *BlockHeader) error {
	if len(msg.Headers)+1 > MaxBlockHeadersPerMsg {
		str := fmt.Sprintf("too many block headers in message [max %v]",
			MaxBlockHeadersPerMsg)
		return messageError("MsgHeaders.AddBlockHeader", str)
	}

	msg.Headers = append(msg.Headers, bh)
	return nil
}

//btcdecode使用比特币协议编码将r解码到接收器中。
//这是消息接口实现的一部分。
func (msg *MsgHeaders) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	count, err := ReadVarInt(r, pver)
	if err != nil {
		return err
	}

//限制为每条消息的最大块头。
	if count > MaxBlockHeadersPerMsg {
		str := fmt.Sprintf("too many block headers for message "+
			"[count %v, max %v]", count, MaxBlockHeadersPerMsg)
		return messageError("MsgHeaders.BtcDecode", str)
	}

//创建一个连续的头切片以反序列化为
//减少分配数量。
	headers := make([]BlockHeader, count)
	msg.Headers = make([]*BlockHeader, 0, count)
	for i := uint64(0); i < count; i++ {
		bh := &headers[i]
		err := readBlockHeader(r, pver, bh)
		if err != nil {
			return err
		}

		txCount, err := ReadVarInt(r, pver)
		if err != nil {
			return err
		}

//确保头的事务计数为零。
		if txCount > 0 {
			str := fmt.Sprintf("block headers may not contain "+
				"transactions [count %v]", txCount)
			return messageError("MsgHeaders.BtcDecode", str)
		}
		msg.AddBlockHeader(bh)
	}

	return nil
}

//btcencode使用比特币协议编码将接收器编码为w。
//这是消息接口实现的一部分。
func (msg *MsgHeaders) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
//限制为每条消息的最大块头。
	count := len(msg.Headers)
	if count > MaxBlockHeadersPerMsg {
		str := fmt.Sprintf("too many block headers for message "+
			"[count %v, max %v]", count, MaxBlockHeadersPerMsg)
		return messageError("MsgHeaders.BtcEncode", str)
	}

	err := WriteVarInt(w, pver, uint64(count))
	if err != nil {
		return err
	}

	for _, bh := range msg.Headers {
		err := writeBlockHeader(w, pver, bh)
		if err != nil {
			return err
		}

//有线协议编码始终包含数字的0
//头消息上的事务数。这真的只是一个
//原始实现序列化方式的工件
//阻止头，但它是必需的。
		err = WriteVarInt(w, pver, 0)
		if err != nil {
			return err
		}
	}

	return nil
}

//命令返回消息的协议命令字符串。这是一部分
//消息接口实现。
func (msg *MsgHeaders) Command() string {
	return CmdHeaders
}

//maxpayloadLength返回有效负载的最大长度
//接收器。这是消息接口实现的一部分。
func (msg *MsgHeaders) MaxPayloadLength(pver uint32) uint32 {
//num headers（varint）+max allowed headers（header length+1 byte
//对于始终为0的事务数）。
	return MaxVarIntPayload + ((MaxBlockHeaderPayload + 1) *
		MaxBlockHeadersPerMsg)
}

//newmsgheaders返回符合
//消息接口。有关详细信息，请参阅msgheaders。
func NewMsgHeaders() *MsgHeaders {
	return &MsgHeaders{
		Headers: make([]*BlockHeader, 0, MaxBlockHeadersPerMsg),
	}
}
