
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
	"reflect"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/davecgh/go-spew/spew"
)

//测试块测试msgblock api。
func TestBlock(t *testing.T) {
	pver := ProtocolVersion

//块1标头。
	prevHash := &blockOne.Header.PrevBlock
	merkleHash := &blockOne.Header.MerkleRoot
	bits := blockOne.Header.Bits
	nonce := blockOne.Header.Nonce
	bh := NewBlockHeader(1, prevHash, merkleHash, bits, nonce)

//确保命令为预期值。
	wantCmd := "block"
	msg := NewMsgBlock(bh)
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

//确保返回相同的块头数据。
	if !reflect.DeepEqual(&msg.Header, bh) {
		t.Errorf("NewMsgBlock: wrong block header - got %v, want %v",
			spew.Sdump(&msg.Header), spew.Sdump(bh))
	}

//确保正确添加事务。
	tx := blockOne.Transactions[0].Copy()
	msg.AddTransaction(tx)
	if !reflect.DeepEqual(msg.Transactions, blockOne.Transactions) {
		t.Errorf("AddTransaction: wrong transactions - got %v, want %v",
			spew.Sdump(msg.Transactions),
			spew.Sdump(blockOne.Transactions))
	}

//确保交易正确结算。
	msg.ClearTransactions()
	if len(msg.Transactions) != 0 {
		t.Errorf("ClearTransactions: wrong transactions - got %v, want %v",
			len(msg.Transactions), 0)
	}
}

//testblocktxthashes测试生成所有事务切片的能力
//精确地从一个块散列。
func TestBlockTxHashes(t *testing.T) {
//块1，事务1哈希。
	hashStr := "0e3e2357e806b6cdb1f70b54c3a3a17b6714ee1f0e68bebb44a74b1efd512098"
	wantHash, err := chainhash.NewHashFromStr(hashStr)
	if err != nil {
		t.Errorf("NewHashFromStr: %v", err)
		return
	}

	wantHashes := []chainhash.Hash{*wantHash}
	hashes, err := blockOne.TxHashes()
	if err != nil {
		t.Errorf("TxHashes: %v", err)
	}
	if !reflect.DeepEqual(hashes, wantHashes) {
		t.Errorf("TxHashes: wrong transaction hashes - got %v, want %v",
			spew.Sdump(hashes), spew.Sdump(wantHashes))
	}
}

//testblockhash测试准确生成块哈希的能力。
func TestBlockHash(t *testing.T) {
//块1哈希。
	hashStr := "839a8e6886ab5951d76f411475428afc90947ee320161bbf18eb6048"
	wantHash, err := chainhash.NewHashFromStr(hashStr)
	if err != nil {
		t.Errorf("NewHashFromStr: %v", err)
	}

//确保所生成的哈希是预期的。
	blockHash := blockOne.BlockHash()
	if !blockHash.IsEqual(wantHash) {
		t.Errorf("BlockHash: wrong hash - got %v, want %v",
			spew.Sprint(blockHash), spew.Sprint(wantHash))
	}
}

//testBlockWire测试MSGBlock线对各种数字的编码和解码
//事务输入和输出以及协议版本。
func TestBlockWire(t *testing.T) {
	tests := []struct {
in     *MsgBlock       //要编码的邮件
out    *MsgBlock       //预期的解码消息
buf    []byte          //有线编码
txLocs []TxLoc         //预期交易地点
pver   uint32          //有线编码协议版本
enc    MessageEncoding //消息编码格式
	}{
//最新协议版本。
		{
			&blockOne,
			&blockOne,
			blockOneBytes,
			blockOneTxLocs,
			ProtocolVersion,
			BaseEncoding,
		},

//协议版本BIP0035版本。
		{
			&blockOne,
			&blockOne,
			blockOneBytes,
			blockOneTxLocs,
			BIP0035Version,
			BaseEncoding,
		},

//协议版本Bip0031版本。
		{
			&blockOne,
			&blockOne,
			blockOneBytes,
			blockOneTxLocs,
			BIP0031Version,
			BaseEncoding,
		},

//协议版本NetAddressTimeVersion。
		{
			&blockOne,
			&blockOne,
			blockOneBytes,
			blockOneTxLocs,
			NetAddressTimeVersion,
			BaseEncoding,
		},

//协议版本multipleaddressversion。
		{
			&blockOne,
			&blockOne,
			blockOneBytes,
			blockOneTxLocs,
			MultipleAddressVersion,
			BaseEncoding,
		},
//TODO（roasbef）：添加见证块的大小写
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
		var msg MsgBlock
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

//TestBlockWireErrors对线编码和解码执行负测试
//以确认错误路径正常工作。
func TestBlockWireErrors(t *testing.T) {
//在这里特别使用协议版本60002，而不是最新版本
//因为测试数据使用的是用该协议编码的字节
//版本。
	pver := uint32(60002)

	tests := []struct {
in       *MsgBlock       //编码值
buf      []byte          //有线编码
pver     uint32          //有线编码协议版本
enc      MessageEncoding //消息编码格式
max      int             //引发错误的固定缓冲区的最大大小
writeErr error           //预期的写入错误
readErr  error           //预期的读取错误
	}{
//强制版本错误。
		{&blockOne, blockOneBytes, pver, BaseEncoding, 0, io.ErrShortWrite, io.EOF},
//前一个块哈希中的强制错误。
		{&blockOne, blockOneBytes, pver, BaseEncoding, 4, io.ErrShortWrite, io.EOF},
//在merkle根中强制错误。
		{&blockOne, blockOneBytes, pver, BaseEncoding, 36, io.ErrShortWrite, io.EOF},
//强制时间戳出错。
		{&blockOne, blockOneBytes, pver, BaseEncoding, 68, io.ErrShortWrite, io.EOF},
//强制难度位出错。
		{&blockOne, blockOneBytes, pver, BaseEncoding, 72, io.ErrShortWrite, io.EOF},
//当前标题中的强制错误。
		{&blockOne, blockOneBytes, pver, BaseEncoding, 76, io.ErrShortWrite, io.EOF},
//事务计数中的强制错误。
		{&blockOne, blockOneBytes, pver, BaseEncoding, 80, io.ErrShortWrite, io.EOF},
//强制事务出错。
		{&blockOne, blockOneBytes, pver, BaseEncoding, 81, io.ErrShortWrite, io.EOF},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//编码为有线格式。
		w := newFixedWriter(test.max)
		err := test.in.BtcEncode(w, test.pver, test.enc)
		if err != test.writeErr {
			t.Errorf("BtcEncode #%d wrong error got: %v, want: %v",
				i, err, test.writeErr)
			continue
		}

//从有线格式解码。
		var msg MsgBlock
		r := newFixedReader(test.max, test.buf)
		err = msg.BtcDecode(r, test.pver, test.enc)
		if err != test.readErr {
			t.Errorf("BtcDecode #%d wrong error got: %v, want: %v",
				i, err, test.readErr)
			continue
		}
	}
}

//testblockserialize测试msgblock序列化和反序列化。
func TestBlockSerialize(t *testing.T) {
	tests := []struct {
in     *MsgBlock //要编码的邮件
out    *MsgBlock //预期的解码消息
buf    []byte    //序列化数据
txLocs []TxLoc   //预期交易地点
	}{
		{
			&blockOne,
			&blockOne,
			blockOneBytes,
			blockOneTxLocs,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//序列化块。
		var buf bytes.Buffer
		err := test.in.Serialize(&buf)
		if err != nil {
			t.Errorf("Serialize #%d error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("Serialize #%d\n got: %s want: %s", i,
				spew.Sdump(buf.Bytes()), spew.Sdump(test.buf))
			continue
		}

//反序列化块。
		var block MsgBlock
		rbuf := bytes.NewReader(test.buf)
		err = block.Deserialize(rbuf)
		if err != nil {
			t.Errorf("Deserialize #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(&block, test.out) {
			t.Errorf("Deserialize #%d\n got: %s want: %s", i,
				spew.Sdump(&block), spew.Sdump(test.out))
			continue
		}

//在收集事务位置时反序列化块
//信息。
		var txLocBlock MsgBlock
		br := bytes.NewBuffer(test.buf)
		txLocs, err := txLocBlock.DeserializeTxLoc(br)
		if err != nil {
			t.Errorf("DeserializeTxLoc #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(&txLocBlock, test.out) {
			t.Errorf("DeserializeTxLoc #%d\n got: %s want: %s", i,
				spew.Sdump(&txLocBlock), spew.Sdump(test.out))
			continue
		}
		if !reflect.DeepEqual(txLocs, test.txLocs) {
			t.Errorf("DeserializeTxLoc #%d\n got: %s want: %s", i,
				spew.Sdump(txLocs), spew.Sdump(test.txLocs))
			continue
		}
	}
}

//TestBlockSerializeErrors对线编码和
//解码msgblock以确认错误路径正常工作。
func TestBlockSerializeErrors(t *testing.T) {
	tests := []struct {
in       *MsgBlock //编码值
buf      []byte    //序列化数据
max      int       //引发错误的固定缓冲区的最大大小
writeErr error     //预期的写入错误
readErr  error     //预期的读取错误
	}{
//强制版本错误。
		{&blockOne, blockOneBytes, 0, io.ErrShortWrite, io.EOF},
//前一个块哈希中的强制错误。
		{&blockOne, blockOneBytes, 4, io.ErrShortWrite, io.EOF},
//在merkle根中强制错误。
		{&blockOne, blockOneBytes, 36, io.ErrShortWrite, io.EOF},
//强制时间戳出错。
		{&blockOne, blockOneBytes, 68, io.ErrShortWrite, io.EOF},
//强制难度位出错。
		{&blockOne, blockOneBytes, 72, io.ErrShortWrite, io.EOF},
//当前标题中的强制错误。
		{&blockOne, blockOneBytes, 76, io.ErrShortWrite, io.EOF},
//事务计数中的强制错误。
		{&blockOne, blockOneBytes, 80, io.ErrShortWrite, io.EOF},
//强制事务出错。
		{&blockOne, blockOneBytes, 81, io.ErrShortWrite, io.EOF},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//序列化块。
		w := newFixedWriter(test.max)
		err := test.in.Serialize(w)
		if err != test.writeErr {
			t.Errorf("Serialize #%d wrong error got: %v, want: %v",
				i, err, test.writeErr)
			continue
		}

//反序列化块。
		var block MsgBlock
		r := newFixedReader(test.max, test.buf)
		err = block.Deserialize(r)
		if err != test.readErr {
			t.Errorf("Deserialize #%d wrong error got: %v, want: %v",
				i, err, test.readErr)
			continue
		}

		var txLocBlock MsgBlock
		br := bytes.NewBuffer(test.buf[0:test.max])
		_, err = txLocBlock.DeserializeTxLoc(br)
		if err != test.readErr {
			t.Errorf("DeserializeTxLoc #%d wrong error got: %v, want: %v",
				i, err, test.readErr)
			continue
		}
	}
}

//TestBlockOverflowErrors执行测试以确保反序列化块
//有意为交易数量使用大值
//处理得当。否则，这可能会被用作攻击
//矢量。
func TestBlockOverflowErrors(t *testing.T) {
//在此处特别使用协议版本70001，而不是最新版本
//协议版本，因为测试数据使用的是
//那个版本。
	pver := uint32(70001)

	tests := []struct {
buf  []byte          //有线编码
pver uint32          //有线编码协议版本
enc  MessageEncoding //消息编码格式
err  error           //期望误差
	}{
//声称有~uint64（0）个事务的块。
		{
			[]byte{
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
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
0xff, //TXNCART
			}, pver, BaseEncoding, &MessageError{},
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//从有线格式解码。
		var msg MsgBlock
		r := bytes.NewReader(test.buf)
		err := msg.BtcDecode(r, test.pver, test.enc)
		if reflect.TypeOf(err) != reflect.TypeOf(test.err) {
			t.Errorf("BtcDecode #%d wrong error got: %v, want: %v",
				i, err, reflect.TypeOf(test.err))
			continue
		}

//从有线格式反序列化。
		r = bytes.NewReader(test.buf)
		err = msg.Deserialize(r)
		if reflect.TypeOf(err) != reflect.TypeOf(test.err) {
			t.Errorf("Deserialize #%d wrong error got: %v, want: %v",
				i, err, reflect.TypeOf(test.err))
			continue
		}

//使用Wire格式的事务位置信息反序列化。
		br := bytes.NewBuffer(test.buf)
		_, err = msg.DeserializeTxLoc(br)
		if reflect.TypeOf(err) != reflect.TypeOf(test.err) {
			t.Errorf("DeserializeTxLoc #%d wrong error got: %v, "+
				"want: %v", i, err, reflect.TypeOf(test.err))
			continue
		}
	}
}

//TestBlockSerializeSize执行测试以确保
//各块准确。
func TestBlockSerializeSize(t *testing.T) {
//不带事务的块。
	noTxBlock := NewMsgBlock(&blockOne.Header)

	tests := []struct {
in   *MsgBlock //块编码
size int       //应为序列化大小
	}{
//不带事务的块。
		{noTxBlock, 81},

//主网区块链中的第一个区块。
		{&blockOne, len(blockOneBytes)},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		serializedSize := test.in.SerializeSize()
		if serializedSize != test.size {
			t.Errorf("MsgBlock.SerializeSize: #%d got: %d, want: "+
				"%d", i, serializedSize, test.size)
			continue
		}
	}
}

//blockone是主网块链中的第一个块。
var blockOne = MsgBlock{
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
	Transactions: []*MsgTx{
		{
			Version: 1,
			TxIn: []*TxIn{
				{
					PreviousOutPoint: OutPoint{
						Hash:  chainhash.Hash{},
						Index: 0xffffffff,
					},
					SignatureScript: []byte{
						0x04, 0xff, 0xff, 0x00, 0x1d, 0x01, 0x04,
					},
					Sequence: 0xffffffff,
				},
			},
			TxOut: []*TxOut{
				{
					Value: 0x12a05f200,
					PkScript: []byte{
0x41, //OPDA DATA65
						0x04, 0x96, 0xb5, 0x38, 0xe8, 0x53, 0x51, 0x9c,
						0x72, 0x6a, 0x2c, 0x91, 0xe6, 0x1e, 0xc1, 0x16,
						0x00, 0xae, 0x13, 0x90, 0x81, 0x3a, 0x62, 0x7c,
						0x66, 0xfb, 0x8b, 0xe7, 0x94, 0x7b, 0xe6, 0x3c,
						0x52, 0xda, 0x75, 0x89, 0x37, 0x95, 0x15, 0xd4,
						0xe0, 0xa6, 0x04, 0xf8, 0x14, 0x17, 0x81, 0xe6,
						0x22, 0x94, 0x72, 0x11, 0x66, 0xbf, 0x62, 0x1e,
						0x73, 0xa8, 0x2c, 0xbf, 0x23, 0x42, 0xc8, 0x58,
0xee, //65字节签名
0xac, //奥普克西格
					},
				},
			},
			LockTime: 0,
		},
	},
}

//阻止一个序列化字节。
var blockOneBytes = []byte{
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
0x01,                   //TXNCART
0x01, 0x00, 0x00, 0x00, //版本
0x01, //事务输入数量的变量
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, //上一个输出哈希
0xff, 0xff, 0xff, 0xff, //前期产出指数
0x07,                                     //签名脚本长度的变量
0x04, 0xff, 0xff, 0x00, 0x1d, 0x01, 0x04, //签名脚本（coinbase）
0xff, 0xff, 0xff, 0xff, //序列
0x01,                                           //事务输出数量的变量
0x00, 0xf2, 0x05, 0x2a, 0x01, 0x00, 0x00, 0x00, //交易金额
0x43, //pk脚本长度的变量
0x41, //OPDA DATA65
	0x04, 0x96, 0xb5, 0x38, 0xe8, 0x53, 0x51, 0x9c,
	0x72, 0x6a, 0x2c, 0x91, 0xe6, 0x1e, 0xc1, 0x16,
	0x00, 0xae, 0x13, 0x90, 0x81, 0x3a, 0x62, 0x7c,
	0x66, 0xfb, 0x8b, 0xe7, 0x94, 0x7b, 0xe6, 0x3c,
	0x52, 0xda, 0x75, 0x89, 0x37, 0x95, 0x15, 0xd4,
	0xe0, 0xa6, 0x04, 0xf8, 0x14, 0x17, 0x81, 0xe6,
	0x22, 0x94, 0x72, 0x11, 0x66, 0xbf, 0x62, 0x1e,
	0x73, 0xa8, 0x2c, 0xbf, 0x23, 0x42, 0xc8, 0x58,
0xee,                   //65字节的未压缩公钥
0xac,                   //奥普克西格
0x00, 0x00, 0x00, 0x00, //锁定时间
}

//块1事务的事务位置信息。
var blockOneTxLocs = []TxLoc{
	{TxStart: 81, TxLen: 134},
}
