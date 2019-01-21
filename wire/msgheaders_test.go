
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

	"github.com/davecgh/go-spew/spew"
)

//测试头测试MSgheaders API。
func TestHeaders(t *testing.T) {
	pver := uint32(60002)

//确保命令为预期值。
	wantCmd := "headers"
	msg := NewMsgHeaders()
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgHeaders: wrong command - got %v want %v",
			cmd, wantCmd)
	}

//确保最大有效负载是最新协议版本的预期值。
//num headers（varint）+max allowed headers（header length+1 byte
//对于始终为0的事务数）。
	wantPayload := uint32(162009)
	maxPayload := msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}

//确保正确添加标题。
	bh := &blockOne.Header
	msg.AddBlockHeader(bh)
	if !reflect.DeepEqual(msg.Headers[0], bh) {
		t.Errorf("AddHeader: wrong header - got %v, want %v",
			spew.Sdump(msg.Headers),
			spew.Sdump(bh))
	}

//确保每封邮件添加的邮件头数超过了允许的最大值。
//错误。
	var err error
	for i := 0; i < MaxBlockHeadersPerMsg+1; i++ {
		err = msg.AddBlockHeader(bh)
	}
	if reflect.TypeOf(err) != reflect.TypeOf(&MessageError{}) {
		t.Errorf("AddBlockHeader: expected error on too many headers " +
			"not received")
	}
}

//测试头Wire测试MSgheaders线的各种编码和解码
//头和协议版本的数量。
func TestHeadersWire(t *testing.T) {
	hash := mainNetGenesisHash
	merkleHash := blockOne.Header.MerkleRoot
	bits := uint32(0x1d00ffff)
	nonce := uint32(0x9962e301)
	bh := NewBlockHeader(1, &hash, &merkleHash, bits, nonce)
	bh.Version = blockOne.Header.Version
	bh.Timestamp = blockOne.Header.Timestamp

//空邮件头消息。
	noHeaders := NewMsgHeaders()
	noHeadersEncoded := []byte{
0x00, //标题数量的变量
	}

//带有一个邮件头的邮件头。
	oneHeader := NewMsgHeaders()
	oneHeader.AddBlockHeader(bh)
	oneHeaderEncoded := []byte{
0x01,                   //标题数量的变量。
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
0x00, //txncount（0表示邮件头）
	}

	tests := []struct {
in   *MsgHeaders     //要编码的邮件
out  *MsgHeaders     //预期的解码消息
buf  []byte          //有线编码
pver uint32          //有线编码协议版本
enc  MessageEncoding //消息编码格式
	}{
//没有标题的最新协议版本。
		{
			noHeaders,
			noHeaders,
			noHeadersEncoded,
			ProtocolVersion,
			BaseEncoding,
		},

//具有一个头的最新协议版本。
		{
			oneHeader,
			oneHeader,
			oneHeaderEncoded,
			ProtocolVersion,
			BaseEncoding,
		},

//协议版本bip0035版本，无标题。
		{
			noHeaders,
			noHeaders,
			noHeadersEncoded,
			BIP0035Version,
			BaseEncoding,
		},

//协议版本bip0035版本，带有一个标题。
		{
			oneHeader,
			oneHeader,
			oneHeaderEncoded,
			BIP0035Version,
			BaseEncoding,
		},

//协议版本bip0031，无标题。
		{
			noHeaders,
			noHeaders,
			noHeadersEncoded,
			BIP0031Version,
			BaseEncoding,
		},

//协议版本bip0031版本，带有一个标题。
		{
			oneHeader,
			oneHeader,
			oneHeaderEncoded,
			BIP0031Version,
			BaseEncoding,
		},
//协议版本NetAddressTimeVersion，没有标题。
		{
			noHeaders,
			noHeaders,
			noHeadersEncoded,
			NetAddressTimeVersion,
			BaseEncoding,
		},

//协议版本NetAddressTimeVersion，带有一个头。
		{
			oneHeader,
			oneHeader,
			oneHeaderEncoded,
			NetAddressTimeVersion,
			BaseEncoding,
		},

//协议版本无标题的multipleaddressversion。
		{
			noHeaders,
			noHeaders,
			noHeadersEncoded,
			MultipleAddressVersion,
			BaseEncoding,
		},

//协议版本具有一个标题的multipleaddressversion。
		{
			oneHeader,
			oneHeader,
			oneHeaderEncoded,
			MultipleAddressVersion,
			BaseEncoding,
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
		var msg MsgHeaders
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

//测试头WireErrors对线编码和解码执行负测试
//以确认错误路径正常工作。
func TestHeadersWireErrors(t *testing.T) {
	pver := ProtocolVersion
	wireErr := &MessageError{}

	hash := mainNetGenesisHash
	merkleHash := blockOne.Header.MerkleRoot
	bits := uint32(0x1d00ffff)
	nonce := uint32(0x9962e301)
	bh := NewBlockHeader(1, &hash, &merkleHash, bits, nonce)
	bh.Version = blockOne.Header.Version
	bh.Timestamp = blockOne.Header.Timestamp

//带有一个邮件头的邮件头。
	oneHeader := NewMsgHeaders()
	oneHeader.AddBlockHeader(bh)
	oneHeaderEncoded := []byte{
0x01,                   //标题数量的变量。
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
0x00, //txncount（0表示邮件头）
	}

//通过超过允许的最大值而强制出错的消息
//标题。
	maxHeaders := NewMsgHeaders()
	for i := 0; i < MaxBlockHeadersPerMsg; i++ {
		maxHeaders.AddBlockHeader(bh)
	}
	maxHeaders.Headers = append(maxHeaders.Headers, bh)
	maxHeadersEncoded := []byte{
0xfd, 0xd1, 0x07, //地址数变量（2001）7d1
	}

//故意使用事务计数的块头无效
//强制错误。
	bhTrans := NewBlockHeader(1, &hash, &merkleHash, bits, nonce)
	bhTrans.Version = blockOne.Header.Version
	bhTrans.Timestamp = blockOne.Header.Timestamp

	transHeader := NewMsgHeaders()
	transHeader.AddBlockHeader(bhTrans)
	transHeaderEncoded := []byte{
0x01,                   //标题数量的变量。
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
0x01, //txncount（对于头消息，应为0，但对于强制错误，应为1）
	}

	tests := []struct {
in       *MsgHeaders     //编码值
buf      []byte          //有线编码
pver     uint32          //有线编码协议版本
enc      MessageEncoding //消息编码格式
max      int             //引发错误的固定缓冲区的最大大小
writeErr error           //预期的写入错误
readErr  error           //预期的读取错误
	}{
//具有故意读/写错误的最新协议版本。
//头计数中的强制错误。
		{oneHeader, oneHeaderEncoded, pver, BaseEncoding, 0, io.ErrShortWrite, io.EOF},
//块头中的强制错误。
		{oneHeader, oneHeaderEncoded, pver, BaseEncoding, 5, io.ErrShortWrite, io.EOF},
//强制出错，头数大于最大值。
		{maxHeaders, maxHeadersEncoded, pver, BaseEncoding, 3, wireErr, wireErr},
//强制事务数出错。
		{transHeader, transHeaderEncoded, pver, BaseEncoding, 81, io.ErrShortWrite, io.EOF},
//强制包含的事务出错。
		{transHeader, transHeaderEncoded, pver, BaseEncoding, len(transHeaderEncoded), nil, wireErr},
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
		var msg MsgHeaders
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
