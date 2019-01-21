
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2014-2015 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wire

import (
	"fmt"
	"io"
)

const (
//maxfilteradddatasize是数据的最大字节大小
//要添加到Bloom筛选器的元素。它等于
//脚本的最大元素大小。
	MaxFilterAddDataSize = 520
)

//msgfilteradd实现消息接口并表示比特币
//筛选器添加消息。它用于向现有Bloom添加数据元素
//过滤器。
//
//在协议版本bip0037之前未添加此消息。
type MsgFilterAdd struct {
	Data []byte
}

//btcdecode使用比特币协议编码将r解码到接收器中。
//这是消息接口实现的一部分。
func (msg *MsgFilterAdd) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	if pver < BIP0037Version {
		str := fmt.Sprintf("filteradd message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgFilterAdd.BtcDecode", str)
	}

	var err error
	msg.Data, err = ReadVarBytes(r, pver, MaxFilterAddDataSize,
		"filteradd data")
	return err
}

//btcencode使用比特币协议编码将接收器编码为w。
//这是消息接口实现的一部分。
func (msg *MsgFilterAdd) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	if pver < BIP0037Version {
		str := fmt.Sprintf("filteradd message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgFilterAdd.BtcEncode", str)
	}

	size := len(msg.Data)
	if size > MaxFilterAddDataSize {
		str := fmt.Sprintf("filteradd size too large for message "+
			"[size %v, max %v]", size, MaxFilterAddDataSize)
		return messageError("MsgFilterAdd.BtcEncode", str)
	}

	return WriteVarBytes(w, pver, msg.Data)
}

//命令返回消息的协议命令字符串。这是一部分
//消息接口实现。
func (msg *MsgFilterAdd) Command() string {
	return CmdFilterAdd
}

//maxpayloadLength返回有效负载的最大长度
//接收器。这是消息接口实现的一部分。
func (msg *MsgFilterAdd) MaxPayloadLength(pver uint32) uint32 {
	return uint32(VarIntSerializeSize(MaxFilterAddDataSize)) +
		MaxFilterAddDataSize
}

//newmsgfilteradd返回符合
//消息接口。有关详细信息，请参阅msgfilteradd。
func NewMsgFilterAdd(data []byte) *MsgFilterAdd {
	return &MsgFilterAdd{
		Data: data,
	}
}
