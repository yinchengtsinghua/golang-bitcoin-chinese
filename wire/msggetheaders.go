
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

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

//msggetaders实现消息接口并表示比特币
//GetHeaders消息。它用于请求块头列表
//块定位器切片中最后一个已知哈希之后开始的块
//散列。列表通过头消息（msgheaders）返回，并且
//受要停止的特定哈希或块头的最大数目限制
//每条消息，目前是2000条。
//
//将hash stop字段设置为要停止和使用的哈希
//添加blocklocatorhash以建立块定位器哈希列表。
//
//构建块定位器散列的算法应该是添加
//按相反的顺序散列，直到到达Genesis区块。为了保持
//定位器列表散列到一个可共振的条目数，首先添加
//最近10个块散列，然后将每个循环迭代的步骤加倍到
//以指数形式减少离头部越远的散列数，并且
//靠近你得到的创世块。
type MsgGetHeaders struct {
	ProtocolVersion    uint32
	BlockLocatorHashes []*chainhash.Hash
	HashStop           chainhash.Hash
}

//addBlockLocatorHash向消息添加新的块定位器哈希。
func (msg *MsgGetHeaders) AddBlockLocatorHash(hash *chainhash.Hash) error {
	if len(msg.BlockLocatorHashes)+1 > MaxBlockLocatorsPerMsg {
		str := fmt.Sprintf("too many block locator hashes for message [max %v]",
			MaxBlockLocatorsPerMsg)
		return messageError("MsgGetHeaders.AddBlockLocatorHash", str)
	}

	msg.BlockLocatorHashes = append(msg.BlockLocatorHashes, hash)
	return nil
}

//btcdecode使用比特币协议编码将r解码到接收器中。
//这是消息接口实现的一部分。
func (msg *MsgGetHeaders) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	err := readElement(r, &msg.ProtocolVersion)
	if err != nil {
		return err
	}

//读取num块定位器散列并限制为max。
	count, err := ReadVarInt(r, pver)
	if err != nil {
		return err
	}
	if count > MaxBlockLocatorsPerMsg {
		str := fmt.Sprintf("too many block locator hashes for message "+
			"[count %v, max %v]", count, MaxBlockLocatorsPerMsg)
		return messageError("MsgGetHeaders.BtcDecode", str)
	}

//创建一个连续的哈希切片以反序列化为
//减少分配数量。
	locatorHashes := make([]chainhash.Hash, count)
	msg.BlockLocatorHashes = make([]*chainhash.Hash, 0, count)
	for i := uint64(0); i < count; i++ {
		hash := &locatorHashes[i]
		err := readElement(r, hash)
		if err != nil {
			return err
		}
		msg.AddBlockLocatorHash(hash)
	}

	return readElement(r, &msg.HashStop)
}

//btcencode使用比特币协议编码将接收器编码为w。
//这是消息接口实现的一部分。
func (msg *MsgGetHeaders) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
//每个消息的最大块定位器哈希数限制。
	count := len(msg.BlockLocatorHashes)
	if count > MaxBlockLocatorsPerMsg {
		str := fmt.Sprintf("too many block locator hashes for message "+
			"[count %v, max %v]", count, MaxBlockLocatorsPerMsg)
		return messageError("MsgGetHeaders.BtcEncode", str)
	}

	err := writeElement(w, msg.ProtocolVersion)
	if err != nil {
		return err
	}

	err = WriteVarInt(w, pver, uint64(count))
	if err != nil {
		return err
	}

	for _, hash := range msg.BlockLocatorHashes {
		err := writeElement(w, hash)
		if err != nil {
			return err
		}
	}

	return writeElement(w, &msg.HashStop)
}

//命令返回消息的协议命令字符串。这是一部分
//消息接口实现。
func (msg *MsgGetHeaders) Command() string {
	return CmdGetHeaders
}

//maxpayloadLength返回有效负载的最大长度
//接收器。这是消息接口实现的一部分。
func (msg *MsgGetHeaders) MaxPayloadLength(pver uint32) uint32 {
//版本4 bytes+num block locator hashes（varint）+max allowed block
//定位器+散列停止。
	return 4 + MaxVarIntPayload + (MaxBlockLocatorsPerMsg *
		chainhash.HashSize) + chainhash.HashSize
}

//NewMsggetHeaders返回符合以下条件的新比特币getHeaders消息
//消息接口。有关详细信息，请参阅msggetalers。
func NewMsgGetHeaders() *MsgGetHeaders {
	return &MsgGetHeaders{
		BlockLocatorHashes: make([]*chainhash.Hash, 0,
			MaxBlockLocatorsPerMsg),
	}
}
