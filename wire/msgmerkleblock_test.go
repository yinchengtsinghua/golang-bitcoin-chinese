
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
	"bytes"
	"crypto/rand"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/davecgh/go-spew/spew"
)

//testmerkleblock测试msgmerkleblock API。
func TestMerkleBlock(t *testing.T) {
	pver := ProtocolVersion
	enc := BaseEncoding

//块1标头。
	prevHash := &blockOne.Header.PrevBlock
	merkleHash := &blockOne.Header.MerkleRoot
	bits := blockOne.Header.Bits
	nonce := blockOne.Header.Nonce
	bh := NewBlockHeader(1, prevHash, merkleHash, bits, nonce)

//确保命令为预期值。
	wantCmd := "merkleblock"
	msg := NewMsgMerkleBlock(bh)
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgBlock: wrong command - got %v want %v",
			cmd, wantCmd)
	}

//确保最大有效负载是最新协议版本的预期值。
//num addresses（varint）+允许的最大地址。
	wantPayload := uint32(4000000)
	maxPayload := msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}

//加载MaxTxPerBlock哈希
	data := make([]byte, 32)
	for i := 0; i < maxTxPerBlock; i++ {
		rand.Read(data)
		hash, err := chainhash.NewHash(data)
		if err != nil {
			t.Errorf("NewHash failed: %v\n", err)
			return
		}

		if err = msg.AddTxHash(hash); err != nil {
			t.Errorf("AddTxHash failed: %v\n", err)
			return
		}
	}

//再添加一个Tx以测试失败。
	rand.Read(data)
	hash, err := chainhash.NewHash(data)
	if err != nil {
		t.Errorf("NewHash failed: %v\n", err)
		return
	}

	if err = msg.AddTxHash(hash); err == nil {
		t.Errorf("AddTxHash succeeded when it should have failed")
		return
	}

//使用最新的协议版本进行测试编码。
	var buf bytes.Buffer
	err = msg.BtcEncode(&buf, pver, enc)
	if err != nil {
		t.Errorf("encode of MsgMerkleBlock failed %v err <%v>", msg, err)
	}

//使用最新的协议版本测试解码。
	readmsg := MsgMerkleBlock{}
	err = readmsg.BtcDecode(&buf, pver, enc)
	if err != nil {
		t.Errorf("decode of MsgMerkleBlock failed [%v] err <%v>", buf, err)
	}

//强制额外哈希测试MaxTxPerBlock。
	msg.Hashes = append(msg.Hashes, hash)
	err = msg.BtcEncode(&buf, pver, enc)
	if err == nil {
		t.Errorf("encode of MsgMerkleBlock succeeded with too many " +
			"tx hashes when it should have failed")
		return
	}

//强制过多标志字节以测试MaxFlagsPermerKleBlock。
//将哈希数重置回有效值。
	msg.Hashes = msg.Hashes[len(msg.Hashes)-1:]
	msg.Flags = make([]byte, maxFlagsPerMerkleBlock+1)
	err = msg.BtcEncode(&buf, pver, enc)
	if err == nil {
		t.Errorf("encode of MsgMerkleBlock succeeded with too many " +
			"flag bytes when it should have failed")
		return
	}
}

//testmerkleBlockCrossProtocol在使用
//最新的协议版本和使用bip0031版本的解码。
func TestMerkleBlockCrossProtocol(t *testing.T) {
//块1标头。
	prevHash := &blockOne.Header.PrevBlock
	merkleHash := &blockOne.Header.MerkleRoot
	bits := blockOne.Header.Bits
	nonce := blockOne.Header.Nonce
	bh := NewBlockHeader(1, prevHash, merkleHash, bits, nonce)

	msg := NewMsgMerkleBlock(bh)

//使用最新的协议版本进行编码。
	var buf bytes.Buffer
	err := msg.BtcEncode(&buf, ProtocolVersion, BaseEncoding)
	if err != nil {
		t.Errorf("encode of NewMsgFilterLoad failed %v err <%v>", msg,
			err)
	}

//使用旧协议版本解码。
	var readmsg MsgFilterLoad
	err = readmsg.BtcDecode(&buf, BIP0031Version, BaseEncoding)
	if err == nil {
		t.Errorf("decode of MsgFilterLoad succeeded when it shouldn't have %v",
			msg)
	}
}

//testmerkleblockwire测试msgmerkleblock线的编码和解码
//不同数量的事务散列和协议版本。
func TestMerkleBlockWire(t *testing.T) {
	tests := []struct {
in   *MsgMerkleBlock //要编码的邮件
out  *MsgMerkleBlock //预期的解码消息
buf  []byte          //有线编码
pver uint32          //有线编码协议版本
enc  MessageEncoding //消息编码格式
	}{
//最新协议版本。
		{
			&merkleBlockOne, &merkleBlockOne, merkleBlockOneBytes,
			ProtocolVersion, BaseEncoding,
		},

//协议版本BIP0037版本。
		{
			&merkleBlockOne, &merkleBlockOne, merkleBlockOneBytes,
			BIP0037Version, BaseEncoding,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//将邮件编码为有线格式。
		var buf bytes.Buffer
		err := test.in.BtcEncode(&buf, test.pver, test.enc)
		if err != nil {
			t.Errorf("BtcEncode #%d error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("BtcEncode #%d\n got: %s want: %s", i,
				spew.Sdump(buf.Bytes()), spew.Sdump(test.buf))
			continue
		}

//从有线格式解码消息。
		var msg MsgMerkleBlock
		rbuf := bytes.NewReader(test.buf)
		err = msg.BtcDecode(rbuf, test.pver, test.enc)
		if err != nil {
			t.Errorf("BtcDecode #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(&msg, test.out) {
			t.Errorf("BtcDecode #%d\n got: %s want: %s", i,
				spew.Sdump(&msg), spew.Sdump(test.out))
			continue
		}
	}
}

//TestMerkleBlockWireErrors对线编码和
//解码msgblock以确认错误路径正常工作。
func TestMerkleBlockWireErrors(t *testing.T) {
//在此处特别使用协议版本70001，而不是最新版本
//因为测试数据使用的是用该协议编码的字节
//版本。
	pver := uint32(70001)
	pverNoMerkleBlock := BIP0037Version - 1
	wireErr := &MessageError{}

	tests := []struct {
in       *MsgMerkleBlock //编码值
buf      []byte          //有线编码
pver     uint32          //有线编码协议版本
enc      MessageEncoding //消息编码格式
max      int             //引发错误的固定缓冲区的最大大小
writeErr error           //预期的写入错误
readErr  error           //预期的读取错误
	}{
//强制版本错误。
		{
			&merkleBlockOne, merkleBlockOneBytes, pver, BaseEncoding, 0,
			io.ErrShortWrite, io.EOF,
		},
//前一个块哈希中的强制错误。
		{
			&merkleBlockOne, merkleBlockOneBytes, pver, BaseEncoding, 4,
			io.ErrShortWrite, io.EOF,
		},
//在merkle根中强制错误。
		{
			&merkleBlockOne, merkleBlockOneBytes, pver, BaseEncoding, 36,
			io.ErrShortWrite, io.EOF,
		},
//强制时间戳出错。
		{
			&merkleBlockOne, merkleBlockOneBytes, pver, BaseEncoding, 68,
			io.ErrShortWrite, io.EOF,
		},
//强制难度位出错。
		{
			&merkleBlockOne, merkleBlockOneBytes, pver, BaseEncoding, 72,
			io.ErrShortWrite, io.EOF,
		},
//当前标题中的强制错误。
		{
			&merkleBlockOne, merkleBlockOneBytes, pver, BaseEncoding, 76,
			io.ErrShortWrite, io.EOF,
		},
//事务计数中的强制错误。
		{
			&merkleBlockOne, merkleBlockOneBytes, pver, BaseEncoding, 80,
			io.ErrShortWrite, io.EOF,
		},
//num散列中的强制错误。
		{
			&merkleBlockOne, merkleBlockOneBytes, pver, BaseEncoding, 84,
			io.ErrShortWrite, io.EOF,
		},
//哈希中的强制错误。
		{
			&merkleBlockOne, merkleBlockOneBytes, pver, BaseEncoding, 85,
			io.ErrShortWrite, io.EOF,
		},
//强制num标志字节出错。
		{
			&merkleBlockOne, merkleBlockOneBytes, pver, BaseEncoding, 117,
			io.ErrShortWrite, io.EOF,
		},
//强制标记字节出错。
		{
			&merkleBlockOne, merkleBlockOneBytes, pver, BaseEncoding, 118,
			io.ErrShortWrite, io.EOF,
		},
//由于协议版本不受支持而强制出错。
		{
			&merkleBlockOne, merkleBlockOneBytes, pverNoMerkleBlock,
			BaseEncoding, 119, wireErr, wireErr,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//编码为有线格式。
		w := newFixedWriter(test.max)
		err := test.in.BtcEncode(w, test.pver, test.enc)
		if reflect.TypeOf(err) != reflect.TypeOf(test.writeErr) {
			t.Errorf("BtcEncode #%d wrong error got: %v, want: %v",
				i, err, test.writeErr)
			continue
		}

//对于不属于messageerror类型的错误，请检查它们
//平等。
		if _, ok := err.(*MessageError); !ok {
			if err != test.writeErr {
				t.Errorf("BtcEncode #%d wrong error got: %v, "+
					"want: %v", i, err, test.writeErr)
				continue
			}
		}

//从有线格式解码。
		var msg MsgMerkleBlock
		r := newFixedReader(test.max, test.buf)
		err = msg.BtcDecode(r, test.pver, test.enc)
		if reflect.TypeOf(err) != reflect.TypeOf(test.readErr) {
			t.Errorf("BtcDecode #%d wrong error got: %v, want: %v",
				i, err, test.readErr)
			continue
		}

//对于不属于messageerror类型的错误，请检查它们
//平等。
		if _, ok := err.(*MessageError); !ok {
			if err != test.readErr {
				t.Errorf("BtcDecode #%d wrong error got: %v, "+
					"want: %v", i, err, test.readErr)
				continue
			}
		}
	}
}

//testmerkleBlockOverflowErrors执行测试以确保编码和解码
//merkle块是专门为使用大值
//正确处理哈希和标志的数目。否则的话
//可能用作攻击媒介。
func TestMerkleBlockOverflowErrors(t *testing.T) {
//在此处特别使用协议版本70001，而不是最新版本
//协议版本，因为测试数据使用的是
//那个版本。
	pver := uint32(70001)

//为声称超过最大值的merkle块创建字节
//允许的Tx哈希。
	var buf bytes.Buffer
	WriteVarInt(&buf, pver, maxTxPerBlock+1)
	numHashesOffset := 84
	exceedMaxHashes := make([]byte, numHashesOffset)
	copy(exceedMaxHashes, merkleBlockOneBytes[:numHashesOffset])
	exceedMaxHashes = append(exceedMaxHashes, buf.Bytes()...)

//为声称超过最大值的merkle块创建字节
//允许的标志字节。
	buf.Reset()
	WriteVarInt(&buf, pver, maxFlagsPerMerkleBlock+1)
	numFlagBytesOffset := 117
	exceedMaxFlagBytes := make([]byte, numFlagBytesOffset)
	copy(exceedMaxFlagBytes, merkleBlockOneBytes[:numFlagBytesOffset])
	exceedMaxFlagBytes = append(exceedMaxFlagBytes, buf.Bytes()...)

	tests := []struct {
buf  []byte          //有线编码
pver uint32          //有线编码协议版本
enc  MessageEncoding //消息编码格式
err  error           //期望误差
	}{
//声明具有超过最大允许哈希数的块。
		{exceedMaxHashes, pver, BaseEncoding, &MessageError{}},
//声明具有超过最大允许标志字节的块。
		{exceedMaxFlagBytes, pver, BaseEncoding, &MessageError{}},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//从有线格式解码。
		var msg MsgMerkleBlock
		r := bytes.NewReader(test.buf)
		err := msg.BtcDecode(r, test.pver, test.enc)
		if reflect.TypeOf(err) != reflect.TypeOf(test.err) {
			t.Errorf("BtcDecode #%d wrong error got: %v, want: %v",
				i, err, reflect.TypeOf(test.err))
			continue
		}
	}
}

//MerkleBlockOne是一个由区块链中的一个区块创建的Merkle区块。
//第一个事务匹配的位置。
var merkleBlockOne = MsgMerkleBlock{
	Header: BlockHeader{
		Version: 1,
PrevBlock: chainhash.Hash([chainhash.HashSize]byte{ //让退伍军人高兴。
			0x6f, 0xe2, 0x8c, 0x0a, 0xb6, 0xf1, 0xb3, 0x72,
			0xc1, 0xa6, 0xa2, 0x46, 0xae, 0x63, 0xf7, 0x4f,
			0x93, 0x1e, 0x83, 0x65, 0xe1, 0x5a, 0x08, 0x9c,
			0x68, 0xd6, 0x19, 0x00, 0x00, 0x00, 0x00, 0x00,
		}),
MerkleRoot: chainhash.Hash([chainhash.HashSize]byte{ //让退伍军人高兴。
			0x98, 0x20, 0x51, 0xfd, 0x1e, 0x4b, 0xa7, 0x44,
			0xbb, 0xbe, 0x68, 0x0e, 0x1f, 0xee, 0x14, 0x67,
			0x7b, 0xa1, 0xa3, 0xc3, 0x54, 0x0b, 0xf7, 0xb1,
			0xcd, 0xb6, 0x06, 0xe8, 0x57, 0x23, 0x3e, 0x0e,
		}),
Timestamp: time.Unix(0x4966bc61, 0), //2009年1月8日20:54:25-0600 cSt
Bits:      0x1d00ffff,               //四亿八千六百六十万四千七百九十九
Nonce:     0x9962e301,               //二十五亿七千三百三十九万四千六百八十九
	},
	Transactions: 1,
	Hashes: []*chainhash.Hash{
(*chainhash.Hash)(&[chainhash.HashSize]byte{ //让退伍军人高兴。
			0x98, 0x20, 0x51, 0xfd, 0x1e, 0x4b, 0xa7, 0x44,
			0xbb, 0xbe, 0x68, 0x0e, 0x1f, 0xee, 0x14, 0x67,
			0x7b, 0xa1, 0xa3, 0xc3, 0x54, 0x0b, 0xf7, 0xb1,
			0xcd, 0xb6, 0x06, 0xe8, 0x57, 0x23, 0x3e, 0x0e,
		}),
	},
	Flags: []byte{0x80},
}

//MerkleBlockoneBytes是从创建的Merkle块的序列化字节
//阻塞第一个事务匹配的块链之一。
var merkleBlockOneBytes = []byte{
0x01, 0x00, 0x00, 0x00, //版本1
	0x6f, 0xe2, 0x8c, 0x0a, 0xb6, 0xf1, 0xb3, 0x72,
	0xc1, 0xa6, 0xa2, 0x46, 0xae, 0x63, 0xf7, 0x4f,
	0x93, 0x1e, 0x83, 0x65, 0xe1, 0x5a, 0x08, 0x9c,
0x68, 0xd6, 0x19, 0x00, 0x00, 0x00, 0x00, 0x00, //预防阻滞
	0x98, 0x20, 0x51, 0xfd, 0x1e, 0x4b, 0xa7, 0x44,
	0xbb, 0xbe, 0x68, 0x0e, 0x1f, 0xee, 0x14, 0x67,
	0x7b, 0xa1, 0xa3, 0xc3, 0x54, 0x0b, 0xf7, 0xb1,
0xcd, 0xb6, 0x06, 0xe8, 0x57, 0x23, 0x3e, 0x0e, //木兰科植物
0x61, 0xbc, 0x66, 0x49, //时间戳
0xff, 0xff, 0x00, 0x1d, //位
0x01, 0xe3, 0x62, 0x99, //临时工
0x01, 0x00, 0x00, 0x00, //TXNCART
0x01, //数字散列
	0x98, 0x20, 0x51, 0xfd, 0x1e, 0x4b, 0xa7, 0x44,
	0xbb, 0xbe, 0x68, 0x0e, 0x1f, 0xee, 0x14, 0x67,
	0x7b, 0xa1, 0xa3, 0xc3, 0x54, 0x0b, 0xf7, 0xb1,
0xcd, 0xb6, 0x06, 0xe8, 0x57, 0x23, 0x3e, 0x0e, //搞砸
0x01, //num标志字节
0x80, //旗帜
}
