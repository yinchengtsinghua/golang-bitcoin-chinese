
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2018 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wire

import (
	"errors"
	"fmt"
	"io"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

const (
//cfcheckptInterval是每个
//筛选头检查点。
	CFCheckptInterval = 1000

//maxcfheaderslen是我们将尝试的最大筛选器头数
//解码。
	maxCFHeadersLen = 100000
)

//错误错误错误读取要求我们解码
//cfilter头的数量不合理。
var ErrInsaneCFHeaderCount = errors.New(
	"refusing to decode unreasonable number of filter headers")

//msgcfcheckpt实现消息接口并表示比特币
//cfcheckpt消息。它用于传递提交的筛选器头信息
//响应getcfcheckpt消息（msggetcfcheckpt）。请参阅msggetcfcheckpt
//有关请求头的详细信息。
type MsgCFCheckpt struct {
	FilterType    FilterType
	StopHash      chainhash.Hash
	FilterHeaders []*chainhash.Hash
}

//addcfheader向消息添加新的已提交筛选器头。
func (msg *MsgCFCheckpt) AddCFHeader(header *chainhash.Hash) error {
	if len(msg.FilterHeaders) == cap(msg.FilterHeaders) {
		str := fmt.Sprintf("FilterHeaders has insufficient capacity for "+
			"additional header: len = %d", len(msg.FilterHeaders))
		return messageError("MsgCFCheckpt.AddCFHeader", str)
	}

	msg.FilterHeaders = append(msg.FilterHeaders, header)
	return nil
}

//btcdecode使用比特币协议编码将r解码到接收器中。
//这是消息接口实现的一部分。
func (msg *MsgCFCheckpt) BtcDecode(r io.Reader, pver uint32, _ MessageEncoding) error {
//读取筛选器类型
	err := readElement(r, &msg.FilterType)
	if err != nil {
		return err
	}

//读取停止哈希
	err = readElement(r, &msg.StopHash)
	if err != nil {
		return err
	}

//读取筛选器头的数目
	count, err := ReadVarInt(r, pver)
	if err != nil {
		return err
	}

//拒绝解码错误数量的cfheaders。
	if count > maxCFHeadersLen {
		return ErrInsaneCFHeaderCount
	}

//创建一个连续的哈希切片以反序列化为
//减少分配数量。
	msg.FilterHeaders = make([]*chainhash.Hash, count)
	for i := uint64(0); i < count; i++ {
		var cfh chainhash.Hash
		err := readElement(r, &cfh)
		if err != nil {
			return err
		}
		msg.FilterHeaders[i] = &cfh
	}

	return nil
}

//btcencode使用比特币协议编码将接收器编码为w。
//这是消息接口实现的一部分。
func (msg *MsgCFCheckpt) BtcEncode(w io.Writer, pver uint32, _ MessageEncoding) error {
//写入筛选器类型
	err := writeElement(w, msg.FilterType)
	if err != nil {
		return err
	}

//写入停止哈希
	err = writeElement(w, msg.StopHash)
	if err != nil {
		return err
	}

//filterheaders切片的写入长度
	count := len(msg.FilterHeaders)
	err = WriteVarInt(w, pver, uint64(count))
	if err != nil {
		return err
	}

	for _, cfh := range msg.FilterHeaders {
		err := writeElement(w, cfh)
		if err != nil {
			return err
		}
	}

	return nil
}

//反序列化使用格式将筛选器头从R解码到接收器
//它适用于数据库等长期存储。这个函数
//与BTCDecode不同的是，BTCDecode从比特币线解码
//通过网络发送的协议。有线编码可以
//技术上的差异取决于协议版本，甚至没有
//完全需要匹配存储的筛选器头的格式。随着时间的推移
//此注释已写入，编码的筛选器头在
//但是有一个明显的区别，将两者分开可以
//API要足够灵活以处理更改。
func (msg *MsgCFCheckpt) Deserialize(r io.Reader) error {
//目前，有线编码没有区别
//以及稳定的长期存储格式。因此，利用
//BtcDecode。
	return msg.BtcDecode(r, 0, BaseEncoding)
}

//命令返回消息的协议命令字符串。这是一部分
//消息接口实现。
func (msg *MsgCFCheckpt) Command() string {
	return CmdCFCheckpt
}

//maxpayloadLength返回有效负载的最大长度
//接收器。这是消息接口实现的一部分。
func (msg *MsgCFCheckpt) MaxPayloadLength(pver uint32) uint32 {
//消息大小取决于区块链高度，因此返回一般限制
//所有消息。
	return MaxMessagePayload
}

//newmsgcfcheckpt返回符合的新比特币cfheaders消息
//消息接口。有关详细信息，请参阅msgcfcheckpt。
func NewMsgCFCheckpt(filterType FilterType, stopHash *chainhash.Hash,
	headersCount int) *MsgCFCheckpt {
	return &MsgCFCheckpt{
		FilterType:    filterType,
		StopHash:      *stopHash,
		FilterHeaders: make([]*chainhash.Hash, 0, headersCount),
	}
}
