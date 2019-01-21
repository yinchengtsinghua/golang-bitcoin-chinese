
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2014-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wire

import (
	"fmt"
	"io"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

//MaxFlagsPermerkleBlock是可以
//可能适合梅克尔街区。因为每个事务都由
//单个位，这是每个块的最大事务数除以
//每字节8位。然后再加一个部分。
const maxFlagsPerMerkleBlock = maxTxPerBlock / 8

//msgmerkleblock实现消息接口并表示比特币
//用于重置Bloom筛选器的MerkleBlock消息。
//
//在协议版本bip0037之前未添加此消息。
type MsgMerkleBlock struct {
	Header       BlockHeader
	Transactions uint32
	Hashes       []*chainhash.Hash
	Flags        []byte
}

//addtxthash向消息添加新的事务哈希。
func (msg *MsgMerkleBlock) AddTxHash(hash *chainhash.Hash) error {
	if len(msg.Hashes)+1 > maxTxPerBlock {
		str := fmt.Sprintf("too many tx hashes for message [max %v]",
			maxTxPerBlock)
		return messageError("MsgMerkleBlock.AddTxHash", str)
	}

	msg.Hashes = append(msg.Hashes, hash)
	return nil
}

//btcdecode使用比特币协议编码将r解码到接收器中。
//这是消息接口实现的一部分。
func (msg *MsgMerkleBlock) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	if pver < BIP0037Version {
		str := fmt.Sprintf("merkleblock message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgMerkleBlock.BtcDecode", str)
	}

	err := readBlockHeader(r, pver, &msg.Header)
	if err != nil {
		return err
	}

	err = readElement(r, &msg.Transactions)
	if err != nil {
		return err
	}

//读取num块定位器散列并限制为max。
	count, err := ReadVarInt(r, pver)
	if err != nil {
		return err
	}
	if count > maxTxPerBlock {
		str := fmt.Sprintf("too many transaction hashes for message "+
			"[count %v, max %v]", count, maxTxPerBlock)
		return messageError("MsgMerkleBlock.BtcDecode", str)
	}

//创建一个连续的哈希切片以反序列化为
//减少分配数量。
	hashes := make([]chainhash.Hash, count)
	msg.Hashes = make([]*chainhash.Hash, 0, count)
	for i := uint64(0); i < count; i++ {
		hash := &hashes[i]
		err := readElement(r, hash)
		if err != nil {
			return err
		}
		msg.AddTxHash(hash)
	}

	msg.Flags, err = ReadVarBytes(r, pver, maxFlagsPerMerkleBlock,
		"merkle block flags size")
	return err
}

//btcencode使用比特币协议编码将接收器编码为w。
//这是消息接口实现的一部分。
func (msg *MsgMerkleBlock) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	if pver < BIP0037Version {
		str := fmt.Sprintf("merkleblock message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgMerkleBlock.BtcEncode", str)
	}

//读取num事务散列并限制为max。
	numHashes := len(msg.Hashes)
	if numHashes > maxTxPerBlock {
		str := fmt.Sprintf("too many transaction hashes for message "+
			"[count %v, max %v]", numHashes, maxTxPerBlock)
		return messageError("MsgMerkleBlock.BtcDecode", str)
	}
	numFlagBytes := len(msg.Flags)
	if numFlagBytes > maxFlagsPerMerkleBlock {
		str := fmt.Sprintf("too many flag bytes for message [count %v, "+
			"max %v]", numFlagBytes, maxFlagsPerMerkleBlock)
		return messageError("MsgMerkleBlock.BtcDecode", str)
	}

	err := writeBlockHeader(w, pver, &msg.Header)
	if err != nil {
		return err
	}

	err = writeElement(w, msg.Transactions)
	if err != nil {
		return err
	}

	err = WriteVarInt(w, pver, uint64(numHashes))
	if err != nil {
		return err
	}
	for _, hash := range msg.Hashes {
		err = writeElement(w, hash)
		if err != nil {
			return err
		}
	}

	return WriteVarBytes(w, pver, msg.Flags)
}

//命令返回消息的协议命令字符串。这是一部分
//消息接口实现。
func (msg *MsgMerkleBlock) Command() string {
	return CmdMerkleBlock
}

//maxpayloadLength返回有效负载的最大长度
//接收器。这是消息接口实现的一部分。
func (msg *MsgMerkleBlock) MaxPayloadLength(pver uint32) uint32 {
	return MaxBlockPayload
}

//newmsgmerkleblock返回符合以下条件的新比特币merkleblock消息
//消息接口。有关详细信息，请参阅msgmerkleblock。
func NewMsgMerkleBlock(bh *BlockHeader) *MsgMerkleBlock {
	return &MsgMerkleBlock{
		Header:       *bh,
		Transactions: 0,
		Hashes:       make([]*chainhash.Hash, 0),
		Flags:        make([]byte, 0),
	}
}
