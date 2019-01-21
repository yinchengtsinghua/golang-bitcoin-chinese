
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
	"bytes"
	"fmt"
	"io"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

//DefaultTransactionalLoc是用于支持数组的默认大小
//对于交易。事务数组将根据需要动态增长，但
//此数字旨在为
//绝大多数块中的事务不需要增长
//多次备份数组。
const defaultTransactionAlloc = 2048

//MaxBlocksPerMsg是每条消息允许的最大块数。
const MaxBlocksPerMsg = 500

//MaxBlockPayload是块消息的最大字节数。
//隔离见证后，最大块有效负载已提升到4MB。
const MaxBlockPayload = 4000000

//MaxTxPerBlock是可以
//可能适合一个区块。
const maxTxPerBlock = (MaxBlockPayload / minTxPayload) + 1

//txloc保存事务所在位置的偏移量和长度的定位器数据。
//位于msgblock数据缓冲区中。
type TxLoc struct {
	TxStart int
	TxLen   int
}

//msgblock实现消息接口并表示比特币
//阻止消息。它用于在
//对给定块哈希的getdata消息（msggetdata）的响应。
type MsgBlock struct {
	Header       BlockHeader
	Transactions []*MsgTx
}

//AddTransaction将事务添加到消息中。
func (msg *MsgBlock) AddTransaction(tx *MsgTx) error {
	msg.Transactions = append(msg.Transactions, tx)
	return nil

}

//ClearTransactions从消息中删除所有事务。
func (msg *MsgBlock) ClearTransactions() {
	msg.Transactions = make([]*MsgTx, 0, defaultTransactionAlloc)
}

//btcdecode使用比特币协议编码将r解码到接收器中。
//这是消息接口实现的一部分。
//请参阅反序列化以获取存储到磁盘的解码块，例如数据库中的解码块，例如
//而不是从电线上解码块。
func (msg *MsgBlock) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	err := readBlockHeader(r, pver, &msg.Header)
	if err != nil {
		return err
	}

	txCount, err := ReadVarInt(r, pver)
	if err != nil {
		return err
	}

//阻止可能无法放入块的事务。
//它可能导致记忆衰竭和恐慌
//在这一点上有一个健全的上限。
	if txCount > maxTxPerBlock {
		str := fmt.Sprintf("too many transactions to fit into a block "+
			"[count %d, max %d]", txCount, maxTxPerBlock)
		return messageError("MsgBlock.BtcDecode", str)
	}

	msg.Transactions = make([]*MsgTx, 0, txCount)
	for i := uint64(0); i < txCount; i++ {
		tx := MsgTx{}
		err := tx.BtcDecode(r, pver, enc)
		if err != nil {
			return err
		}
		msg.Transactions = append(msg.Transactions, &tx)
	}

	return nil
}

//反序列化使用以下格式将块从R解码到接收器：
//适合长期存储，如数据库，同时尊重
//块中的版本字段。此函数与btcdecode的不同之处在于
//BTCDecode从比特币有线协议解码，因为它是通过
//网络。根据协议，有线编码在技术上可能有所不同。
//版本，甚至不需要匹配存储块的格式
//所有。在写入此注释时，编码块是相同的
//在这两种情况下，有一个明显的区别，将两者分开
//允许API足够灵活地处理更改。
func (msg *MsgBlock) Deserialize(r io.Reader) error {
//目前，有线编码没有区别
//在协议版本0和稳定的长期存储格式。AS
//因此，使用btcdecode。
//
//将见证编码的编码类型传递给
//messageencoding参数指示
//块应根据新的
//bip0141中定义的序列化结构。
	return msg.BtcDecode(r, 0, WitnessEncoding)
}

//反序列化enowesting将块从R解码到接收器，类似于
//反序列化，但反序列化维修性会将所有（如果有）见证数据剥离
//在对块内的事务进行编码之前。
func (msg *MsgBlock) DeserializeNoWitness(r io.Reader) error {
	return msg.BtcDecode(r, 0, BaseEncoding)
}

//反序列化etxloc以反序列化的相同方式对r进行解码，但它需要
//字节缓冲区，而不是一般的读卡器，并返回一个包含
//原始数据中每个事务的开始和长度
//反序列化。
func (msg *MsgBlock) DeserializeTxLoc(r *bytes.Buffer) ([]TxLoc, error) {
	fullLen := r.Len()

//目前，有线编码没有区别
//在协议版本0和稳定的长期存储格式。AS
//结果，利用现有的有线协议功能。
	err := readBlockHeader(r, 0, &msg.Header)
	if err != nil {
		return nil, err
	}

	txCount, err := ReadVarInt(r, 0)
	if err != nil {
		return nil, err
	}

//阻止可能无法放入块的事务。
//它可能导致记忆衰竭和恐慌
//在这一点上有一个健全的上限。
	if txCount > maxTxPerBlock {
		str := fmt.Sprintf("too many transactions to fit into a block "+
			"[count %d, max %d]", txCount, maxTxPerBlock)
		return nil, messageError("MsgBlock.DeserializeTxLoc", str)
	}

//反序列化每个事务，同时跟踪其位置
//在字节流中。
	msg.Transactions = make([]*MsgTx, 0, txCount)
	txLocs := make([]TxLoc, txCount)
	for i := uint64(0); i < txCount; i++ {
		txLocs[i].TxStart = fullLen - r.Len()
		tx := MsgTx{}
		err := tx.Deserialize(r)
		if err != nil {
			return nil, err
		}
		msg.Transactions = append(msg.Transactions, &tx)
		txLocs[i].TxLen = (fullLen - r.Len()) - txLocs[i].TxStart
	}

	return txLocs, nil
}

//btcencode使用比特币协议编码将接收器编码为w。
//这是消息接口实现的一部分。
//有关要存储到磁盘的编码块，请参见序列化，例如
//数据库，而不是电线的编码块。
func (msg *MsgBlock) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	err := writeBlockHeader(w, pver, &msg.Header)
	if err != nil {
		return err
	}

	err = WriteVarInt(w, pver, uint64(len(msg.Transactions)))
	if err != nil {
		return err
	}

	for _, tx := range msg.Transactions {
		err = tx.BtcEncode(w, pver, enc)
		if err != nil {
			return err
		}
	}

	return nil
}

//serialize使用适合长期使用的格式将块编码为w
//存储，如数据库，同时考虑块中的版本字段。
//此函数与btcencode不同，btcencode将块编码为
//比特币有线协议，以便通过网络发送。电线
//根据协议版本的不同，编码可能在技术上有所不同，而不是
//甚至需要匹配存储块的格式。至于
//写入此注释时，编码块在
//但是有一个明显的区别，将两者分开可以
//API要足够灵活以处理更改。
func (msg *MsgBlock) Serialize(w io.Writer) error {
//目前，有线编码没有区别
//在协议版本0和稳定的长期存储格式。AS
//结果，使用btcencode。
//
//将见证编码作为此处的编码类型传递表示
//每个事务都应该使用见证进行序列化
//bip0141中定义的序列化结构。
	return msg.BtcEncode(w, 0, WitnessEncoding)
}

//serialinowitness使用相同的格式将块编码为w
//序列化，从所有事务中除去所有（如果有）见证数据。
//除了常规序列化之外，还提供了此方法，以便
//允许有选择地将事务见证数据编码为未升级
//不知道新编码的对等机。
func (msg *MsgBlock) SerializeNoWitness(w io.Writer) error {
	return msg.BtcEncode(w, 0, BaseEncoding)
}

//serializesize返回序列化
//块，处理事务中的任何见证数据。
func (msg *MsgBlock) SerializeSize() int {
//块头字节+序列变量大小
//交易。
	n := blockHeaderLen + VarIntSerializeSize(uint64(len(msg.Transactions)))

	for _, tx := range msg.Transactions {
		n += tx.SerializeSize()
	}

	return n
}

//SerializeSizeStripped返回序列化所需的字节数
//块，不包括任何见证数据（如果有）。
func (msg *MsgBlock) SerializeSizeStripped() int {
//块头字节+序列变量大小
//交易。
	n := blockHeaderLen + VarIntSerializeSize(uint64(len(msg.Transactions)))

	for _, tx := range msg.Transactions {
		n += tx.SerializeSizeStripped()
	}

	return n
}

//命令返回消息的协议命令字符串。这是一部分
//消息接口实现。
func (msg *MsgBlock) Command() string {
	return CmdBlock
}

//maxpayloadLength返回有效负载的最大长度
//接收器。这是消息接口实现的一部分。
func (msg *MsgBlock) MaxPayloadLength(pver uint32) uint32 {
//块头为80字节+事务计数+最大事务数
//最多可以改变MaxBlockPayload（包括块头
//和事务计数）。
	return MaxBlockPayload
}

//block hash计算此块的块标识符哈希。
func (msg *MsgBlock) BlockHash() chainhash.Hash {
	return msg.Header.BlockHash()
}

//txshashes返回此块中所有事务的哈希切片。
func (msg *MsgBlock) TxHashes() ([]chainhash.Hash, error) {
	hashList := make([]chainhash.Hash, 0, len(msg.Transactions))
	for _, tx := range msg.Transactions {
		hashList = append(hashList, tx.TxHash())
	}
	return hashList, nil
}

//newmsgblock返回符合
//消息接口。有关详细信息，请参阅msgblock。
func NewMsgBlock(blockHeader *BlockHeader) *MsgBlock {
	return &MsgBlock{
		Header:       *blockHeader,
		Transactions: make([]*MsgTx, 0, defaultTransactionAlloc),
	}
}
