
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

const (
//MaxCfHeaderPayLoad是提交的
//过滤头。
	MaxCFHeaderPayload = chainhash.HashSize

//MaxCfHeadersPermsg是已提交的最大筛选器头数
//它可以出现在一个比特币的cfheaders消息中。
	MaxCFHeadersPerMsg = 2000
)

//msgcfheaders实现消息接口并表示比特币
//cfheaders消息。它用于传递提交的筛选器头信息
//响应getcfheaders消息（msggetcfheaders）。最大值
//每封邮件的已提交筛选器头数当前为2000。见
//msggetchheaders获取有关请求头的详细信息。
type MsgCFHeaders struct {
	FilterType       FilterType
	StopHash         chainhash.Hash
	PrevFilterHeader chainhash.Hash
	FilterHashes     []*chainhash.Hash
}

//addcfhash向消息添加新的筛选器哈希。
func (msg *MsgCFHeaders) AddCFHash(hash *chainhash.Hash) error {
	if len(msg.FilterHashes)+1 > MaxCFHeadersPerMsg {
		str := fmt.Sprintf("too many block headers in message [max %v]",
			MaxBlockHeadersPerMsg)
		return messageError("MsgCFHeaders.AddCFHash", str)
	}

	msg.FilterHashes = append(msg.FilterHashes, hash)
	return nil
}

//btcdecode使用比特币协议编码将r解码到接收器中。
//这是消息接口实现的一部分。
func (msg *MsgCFHeaders) BtcDecode(r io.Reader, pver uint32, _ MessageEncoding) error {
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

//读取上一个筛选器标题
	err = readElement(r, &msg.PrevFilterHeader)
	if err != nil {
		return err
	}

//读取筛选器头的数目
	count, err := ReadVarInt(r, pver)
	if err != nil {
		return err
	}

//限制为每封邮件的最大提交筛选器头数。
	if count > MaxCFHeadersPerMsg {
		str := fmt.Sprintf("too many committed filter headers for "+
			"message [count %v, max %v]", count,
			MaxBlockHeadersPerMsg)
		return messageError("MsgCFHeaders.BtcDecode", str)
	}

//创建一个连续的哈希切片以反序列化为
//减少分配数量。
	msg.FilterHashes = make([]*chainhash.Hash, 0, count)
	for i := uint64(0); i < count; i++ {
		var cfh chainhash.Hash
		err := readElement(r, &cfh)
		if err != nil {
			return err
		}
		msg.AddCFHash(&cfh)
	}

	return nil
}

//btcencode使用比特币协议编码将接收器编码为w。
//这是消息接口实现的一部分。
func (msg *MsgCFHeaders) BtcEncode(w io.Writer, pver uint32, _ MessageEncoding) error {
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

//写入上一个筛选器头
	err = writeElement(w, msg.PrevFilterHeader)
	if err != nil {
		return err
	}

//限制为每封邮件的最大提交邮件头数。
	count := len(msg.FilterHashes)
	if count > MaxCFHeadersPerMsg {
		str := fmt.Sprintf("too many committed filter headers for "+
			"message [count %v, max %v]", count,
			MaxBlockHeadersPerMsg)
		return messageError("MsgCFHeaders.BtcEncode", str)
	}

	err = WriteVarInt(w, pver, uint64(count))
	if err != nil {
		return err
	}

	for _, cfh := range msg.FilterHashes {
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
func (msg *MsgCFHeaders) Deserialize(r io.Reader) error {
//目前，有线编码没有区别
//以及稳定的长期存储格式。因此，利用
//BtcDecode。
	return msg.BtcDecode(r, 0, BaseEncoding)
}

//命令返回消息的协议命令字符串。这是一部分
//消息接口实现。
func (msg *MsgCFHeaders) Command() string {
	return CmdCFHeaders
}

//maxpayloadLength返回有效负载的最大长度
//接收器。这是消息接口实现的一部分。
func (msg *MsgCFHeaders) MaxPayloadLength(pver uint32) uint32 {
//哈希大小+筛选器类型+num headers（varint）+
//（收割台尺寸*最大收割台）。
	return 1 + chainhash.HashSize + chainhash.HashSize + MaxVarIntPayload +
		(MaxCFHeaderPayload * MaxCFHeadersPerMsg)
}

//newmsgcfheaders返回符合的新比特币cfheaders消息
//消息接口。有关详细信息，请参阅msgcfheaders。
func NewMsgCFHeaders() *MsgCFHeaders {
	return &MsgCFHeaders{
		FilterHashes: make([]*chainhash.Hash, 0, MaxCFHeadersPerMsg),
	}
}
