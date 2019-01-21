
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
	"io"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

//MaxBlockHeaderPayLoad是块头可以达到的最大字节数。
//版本4字节+时间戳4字节+位4字节+nonce 4字节+
//prevblock和merkleroot哈希。
const MaxBlockHeaderPayload = 16 + (chainhash.HashSize * 2)

//blockheader定义关于块的信息，并在比特币中使用。
//阻止（msgblock）和头（msgheaders）消息。
type BlockHeader struct {
//块的版本。这与协议版本不同。
	Version int32

//块链中上一个块头的哈希。
	PrevBlock chainhash.Hash

//对块的所有事务哈希的Merkle树引用。
	MerkleRoot chainhash.Hash

//创建块的时间。不幸的是，这被编码为
//UINT32，因此仅限于2106。
	Timestamp time.Time

//块的难度目标。
	Bits uint32

//用于生成块的nonce。
	Nonce uint32
}

//blockheaderlen是一个常量，表示一个块的字节数。
//标题。
const blockHeaderLen = 80

//block hash为给定的块头计算块标识符哈希。
func (h *BlockHeader) BlockHash() chainhash.Hash {
//对头文件进行编码，并在
//交易。忽略错误返回，因为
//编码可能会失败，除非内存不足，否则会导致
//运行时的恐慌。
	buf := bytes.NewBuffer(make([]byte, 0, MaxBlockHeaderPayload))
	_ = writeBlockHeader(buf, 0, h)

	return chainhash.DoubleHashH(buf.Bytes())
}

//btcdecode使用比特币协议编码将r解码到接收器中。
//这是消息接口实现的一部分。
//请参阅反序列化以解码存储到磁盘的块头，例如
//数据库，而不是从线路解码块头。
func (h *BlockHeader) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	return readBlockHeader(r, pver, h)
}

//btcencode使用比特币协议编码将接收器编码为w。
//这是消息接口实现的一部分。
//有关要存储到磁盘的编码块头的信息，请参见序列化，例如
//数据库，而不是对连接的块头进行编码。
func (h *BlockHeader) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	return writeBlockHeader(w, pver, h)
}

//反序列化使用格式将块头从R解码到接收器
//适用于数据库等长期存储，同时尊重
//版本字段。
func (h *BlockHeader) Deserialize(r io.Reader) error {
//目前，有线编码没有区别
//在协议版本0和稳定的长期存储格式。AS
//结果，使用readBlockHeader。
	return readBlockHeader(r, 0, h)
}

//serialize使用格式将块头从r编码到接收器
//适用于数据库等长期存储，同时尊重
//版本字段。
func (h *BlockHeader) Serialize(w io.Writer) error {
//目前，有线编码没有区别
//在协议版本0和稳定的长期存储格式。AS
//结果，使用writeBlockHeader。
	return writeBlockHeader(w, 0, h)
}

//new blockheader使用提供的版本返回新的blockheader，上一个
//块哈希、merkle根哈希、困难位和nonce用于生成
//用默认值阻止其余字段。
func NewBlockHeader(version int32, prevHash, merkleRootHash *chainhash.Hash,
	bits uint32, nonce uint32) *BlockHeader {

//将时间戳限制为自协议以来的一秒精度
//不支持更好。
	return &BlockHeader{
		Version:    version,
		PrevBlock:  *prevHash,
		MerkleRoot: *merkleRootHash,
		Timestamp:  time.Unix(time.Now().Unix(), 0),
		Bits:       bits,
		Nonce:      nonce,
	}
}

//readblockheader从r中读取比特币块头。请参见反序列化
//解码存储到磁盘（如数据库中）的块头，而不是
//从电线解码。
func readBlockHeader(r io.Reader, pver uint32, bh *BlockHeader) error {
	return readElements(r, &bh.Version, &bh.PrevBlock, &bh.MerkleRoot,
		(*uint32Time)(&bh.Timestamp), &bh.Bits, &bh.Nonce)
}

//WriteBlockHeader将比特币块头写入w。请参见序列化
//对要存储到磁盘的块头（如数据库中）进行编码，例如
//与电线编码相反。
func writeBlockHeader(w io.Writer, pver uint32, bh *BlockHeader) error {
	sec := uint32(bh.Timestamp.Unix())
	return writeElements(w, bh.Version, &bh.PrevBlock, &bh.MerkleRoot,
		sec, bh.Bits, bh.Nonce)
}
