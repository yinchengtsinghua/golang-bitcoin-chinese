
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wire

import (
	"fmt"
	"io"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

//filter type用于表示筛选器类型。
type FilterType uint8

const (
//gcsfilterRegular是常规筛选器类型。
	GCSFilterRegular FilterType = iota
)

const (
//MaxCfilterDataSize是已提交筛选器的最大字节大小。
//当前最大大小定义为256KB。
	MaxCFilterDataSize = 256 * 1024
)

//msgcfilter实现消息接口并表示比特币cfilter
//消息。它用于响应
//getcpilters（msggetcpilters）消息。
type MsgCFilter struct {
	FilterType FilterType
	BlockHash  chainhash.Hash
	Data       []byte
}

//btcdecode使用比特币协议编码将r解码到接收器中。
//这是消息接口实现的一部分。
func (msg *MsgCFilter) BtcDecode(r io.Reader, pver uint32, _ MessageEncoding) error {
//读取筛选器类型
	err := readElement(r, &msg.FilterType)
	if err != nil {
		return err
	}

//读取筛选器块的哈希值
	err = readElement(r, &msg.BlockHash)
	if err != nil {
		return err
	}

//读取筛选数据
	msg.Data, err = ReadVarBytes(r, pver, MaxCFilterDataSize,
		"cfilter data")
	return err
}

//btcencode使用比特币协议编码将接收器编码为w。
//这是消息接口实现的一部分。
func (msg *MsgCFilter) BtcEncode(w io.Writer, pver uint32, _ MessageEncoding) error {
	size := len(msg.Data)
	if size > MaxCFilterDataSize {
		str := fmt.Sprintf("cfilter size too large for message "+
			"[size %v, max %v]", size, MaxCFilterDataSize)
		return messageError("MsgCFilter.BtcEncode", str)
	}

	err := writeElement(w, msg.FilterType)
	if err != nil {
		return err
	}

	err = writeElement(w, msg.BlockHash)
	if err != nil {
		return err
	}

	return WriteVarBytes(w, pver, msg.Data)
}

//反序列化使用以下格式将过滤器从R解码到接收器：
//适用于数据库等长期存储。此功能不同
//从btcdecode中，btcdecode将比特币有线协议解码为
//它是通过网络发送的。线编码在技术上可能有所不同
//取决于协议版本，甚至不需要匹配
//存储过滤器的格式。在写这篇评论的时候，
//编码的过滤器在两个实例中是相同的，但是
//区别和分离允许API足够灵活
//应对变化。
func (msg *MsgCFilter) Deserialize(r io.Reader) error {
//目前，有线编码没有区别
//以及稳定的长期存储格式。因此，利用
//BtcDecode。
	return msg.BtcDecode(r, 0, BaseEncoding)
}

//命令返回消息的协议命令字符串。这是一部分
//消息接口实现。
func (msg *MsgCFilter) Command() string {
	return CmdCFilter
}

//maxpayloadLength返回有效负载的最大长度
//接收器。这是消息接口实现的一部分。
func (msg *MsgCFilter) MaxPayloadLength(pver uint32) uint32 {
	return uint32(VarIntSerializeSize(MaxCFilterDataSize)) +
		MaxCFilterDataSize + chainhash.HashSize + 1
}

//newmsgcfilter返回符合
//消息接口。有关详细信息，请参阅MSGCfilter。
func NewMsgCFilter(filterType FilterType, blockHash *chainhash.Hash,
	data []byte) *MsgCFilter {
	return &MsgCFilter{
		FilterType: filterType,
		BlockHash:  *blockHash,
		Data:       data,
	}
}
