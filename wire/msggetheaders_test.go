
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

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/davecgh/go-spew/spew"
)

//testgetheaders测试msggetader API。
func TestGetHeaders(t *testing.T) {
	pver := ProtocolVersion

//块99500哈希。
	hashStr := "000000000002e7ad7b9eef9479e4aabc65cb831269cc20d2632c13684406dee0"
	locatorHash, err := chainhash.NewHashFromStr(hashStr)
	if err != nil {
		t.Errorf("NewHashFromStr: %v", err)
	}

//确保命令为预期值。
	wantCmd := "getheaders"
	msg := NewMsgGetHeaders()
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgGetHeaders: wrong command - got %v want %v",
			cmd, wantCmd)
	}

//确保最大有效负载是最新协议版本的预期值。
//协议版本4 bytes+num hashes（varint）+max块定位器
//哈希+哈希停止。
	wantPayload := uint32(16045)
	maxPayload := msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}

//确保正确添加块定位器哈希。
	err = msg.AddBlockLocatorHash(locatorHash)
	if err != nil {
		t.Errorf("AddBlockLocatorHash: %v", err)
	}
	if msg.BlockLocatorHashes[0] != locatorHash {
		t.Errorf("AddBlockLocatorHash: wrong block locator added - "+
			"got %v, want %v",
			spew.Sprint(msg.BlockLocatorHashes[0]),
			spew.Sprint(locatorHash))
	}

//确保添加的块定位器哈希数超过允许的最大值。
//消息返回错误。
	for i := 0; i < MaxBlockLocatorsPerMsg; i++ {
		err = msg.AddBlockLocatorHash(locatorHash)
	}
	if err == nil {
		t.Errorf("AddBlockLocatorHash: expected error on too many " +
			"block locator hashes not received")
	}
}

//testGetHeaderWire测试msggetHeaderWire对各种
//块定位器散列数和协议版本。
func TestGetHeadersWire(t *testing.T) {
//在getheaders消息中设置协议。使用协议版本60002
//尤其是这里，而不是最新的，因为测试数据是
//使用该协议版本编码的字节。
	pver := uint32(60002)

//块99499哈希。
	hashStr := "2710f40c87ec93d010a6fd95f42c59a2cbacc60b18cf6b7957535"
	hashLocator, err := chainhash.NewHashFromStr(hashStr)
	if err != nil {
		t.Errorf("NewHashFromStr: %v", err)
	}

//块99500哈希。
	hashStr = "2e7ad7b9eef9479e4aabc65cb831269cc20d2632c13684406dee0"
	hashLocator2, err := chainhash.NewHashFromStr(hashStr)
	if err != nil {
		t.Errorf("NewHashFromStr: %v", err)
	}

//阻止100000哈希。
	hashStr = "3ba27aa200b1cecaad478d2b00432346c3f1f3986da1afd33e506"
	hashStop, err := chainhash.NewHashFromStr(hashStr)
	if err != nil {
		t.Errorf("NewHashFromStr: %v", err)
	}

//msggetalers消息没有块定位器或停止哈希。
	noLocators := NewMsgGetHeaders()
	noLocators.ProtocolVersion = pver
	noLocatorsEncoded := []byte{
0x62, 0xea, 0x00, 0x00, //协议版本60002
0x00, //块定位器哈希数的变量
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, //哈希停止
	}

//msggetalers消息带有多个块定位器和一个停止哈希。
	multiLocators := NewMsgGetHeaders()
	multiLocators.ProtocolVersion = pver
	multiLocators.HashStop = *hashStop
	multiLocators.AddBlockLocatorHash(hashLocator2)
	multiLocators.AddBlockLocatorHash(hashLocator)
	multiLocatorsEncoded := []byte{
0x62, 0xea, 0x00, 0x00, //协议版本60002
0x02, //块定位器哈希数的变量
		0xe0, 0xde, 0x06, 0x44, 0x68, 0x13, 0x2c, 0x63,
		0xd2, 0x20, 0xcc, 0x69, 0x12, 0x83, 0xcb, 0x65,
		0xbc, 0xaa, 0xe4, 0x79, 0x94, 0xef, 0x9e, 0x7b,
0xad, 0xe7, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, //块99500哈希
		0x35, 0x75, 0x95, 0xb7, 0xf6, 0x8c, 0xb1, 0x60,
		0xcc, 0xba, 0x2c, 0x9a, 0xc5, 0x42, 0x5f, 0xd9,
		0x6f, 0x0a, 0x01, 0x3d, 0xc9, 0x7e, 0xc8, 0x40,
0x0f, 0x71, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, //块99499哈希
		0x06, 0xe5, 0x33, 0xfd, 0x1a, 0xda, 0x86, 0x39,
		0x1f, 0x3f, 0x6c, 0x34, 0x32, 0x04, 0xb0, 0xd2,
		0x78, 0xd4, 0xaa, 0xec, 0x1c, 0x0b, 0x20, 0xaa,
0x27, 0xba, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, //哈希停止
	}

	tests := []struct {
in   *MsgGetHeaders  //要编码的邮件
out  *MsgGetHeaders  //预期的解码消息
buf  []byte          //有线编码
pver uint32          //有线编码协议版本
enc  MessageEncoding //消息编码格式
	}{
//没有块定位器的最新协议版本。
		{
			noLocators,
			noLocators,
			noLocatorsEncoded,
			ProtocolVersion,
			BaseEncoding,
		},

//具有多个块定位器的最新协议版本。
		{
			multiLocators,
			multiLocators,
			multiLocatorsEncoded,
			ProtocolVersion,
			BaseEncoding,
		},

//协议版本Bip0035，无块定位器。
		{
			noLocators,
			noLocators,
			noLocatorsEncoded,
			BIP0035Version,
			BaseEncoding,
		},

//协议版本Bip0035，带有多个块定位器。
		{
			multiLocators,
			multiLocators,
			multiLocatorsEncoded,
			BIP0035Version,
			BaseEncoding,
		},

//协议版本Bip0031，无块定位器。
		{
			noLocators,
			noLocators,
			noLocatorsEncoded,
			BIP0031Version,
			BaseEncoding,
		},

//协议版本bip0031带有多个块定位器的版本。
		{
			multiLocators,
			multiLocators,
			multiLocatorsEncoded,
			BIP0031Version,
			BaseEncoding,
		},

//协议版本NetAddressTimeVersion，没有块定位器。
		{
			noLocators,
			noLocators,
			noLocatorsEncoded,
			NetAddressTimeVersion,
			BaseEncoding,
		},

//协议版本NetAddressTimeVersion多块定位器。
		{
			multiLocators,
			multiLocators,
			multiLocatorsEncoded,
			NetAddressTimeVersion,
			BaseEncoding,
		},

//没有块定位器的协议版本multipleaddressversion。
		{
			noLocators,
			noLocators,
			noLocatorsEncoded,
			MultipleAddressVersion,
			BaseEncoding,
		},

//协议版本multipleaddressversion多块定位器。
		{
			multiLocators,
			multiLocators,
			multiLocatorsEncoded,
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
		var msg MsgGetHeaders
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

//TestGetHeadersWireErrors对线编码和
//解码消息生成器以确认错误路径正常工作。
func TestGetHeadersWireErrors(t *testing.T) {
//在getheaders消息中设置协议。使用协议版本60002
//尤其是这里，而不是最新的，因为测试数据是
//使用该协议版本编码的字节。
	pver := uint32(60002)
	wireErr := &MessageError{}

//块99499哈希。
	hashStr := "2710f40c87ec93d010a6fd95f42c59a2cbacc60b18cf6b7957535"
	hashLocator, err := chainhash.NewHashFromStr(hashStr)
	if err != nil {
		t.Errorf("NewHashFromStr: %v", err)
	}

//块99500哈希。
	hashStr = "2e7ad7b9eef9479e4aabc65cb831269cc20d2632c13684406dee0"
	hashLocator2, err := chainhash.NewHashFromStr(hashStr)
	if err != nil {
		t.Errorf("NewHashFromStr: %v", err)
	}

//阻止100000哈希。
	hashStr = "3ba27aa200b1cecaad478d2b00432346c3f1f3986da1afd33e506"
	hashStop, err := chainhash.NewHashFromStr(hashStr)
	if err != nil {
		t.Errorf("NewHashFromStr: %v", err)
	}

//msggetalers消息带有多个块定位器和一个停止哈希。
	baseGetHeaders := NewMsgGetHeaders()
	baseGetHeaders.ProtocolVersion = pver
	baseGetHeaders.HashStop = *hashStop
	baseGetHeaders.AddBlockLocatorHash(hashLocator2)
	baseGetHeaders.AddBlockLocatorHash(hashLocator)
	baseGetHeadersEncoded := []byte{
0x62, 0xea, 0x00, 0x00, //协议版本60002
0x02, //块定位器哈希数的变量
		0xe0, 0xde, 0x06, 0x44, 0x68, 0x13, 0x2c, 0x63,
		0xd2, 0x20, 0xcc, 0x69, 0x12, 0x83, 0xcb, 0x65,
		0xbc, 0xaa, 0xe4, 0x79, 0x94, 0xef, 0x9e, 0x7b,
0xad, 0xe7, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, //块99500哈希
		0x35, 0x75, 0x95, 0xb7, 0xf6, 0x8c, 0xb1, 0x60,
		0xcc, 0xba, 0x2c, 0x9a, 0xc5, 0x42, 0x5f, 0xd9,
		0x6f, 0x0a, 0x01, 0x3d, 0xc9, 0x7e, 0xc8, 0x40,
0x0f, 0x71, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, //块99499哈希
		0x06, 0xe5, 0x33, 0xfd, 0x1a, 0xda, 0x86, 0x39,
		0x1f, 0x3f, 0x6c, 0x34, 0x32, 0x04, 0xb0, 0xd2,
		0x78, 0xd4, 0xaa, 0xec, 0x1c, 0x0b, 0x20, 0xaa,
0x27, 0xba, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, //哈希停止
	}

//通过超过允许的最大值而强制出错的消息
//块定位器哈希。
	maxGetHeaders := NewMsgGetHeaders()
	for i := 0; i < MaxBlockLocatorsPerMsg; i++ {
		maxGetHeaders.AddBlockLocatorHash(&mainNetGenesisHash)
	}
	maxGetHeaders.BlockLocatorHashes = append(maxGetHeaders.BlockLocatorHashes,
		&mainNetGenesisHash)
	maxGetHeadersEncoded := []byte{
0x62, 0xea, 0x00, 0x00, //协议版本60002
0xfd, 0xf5, 0x01, //块loc散列数变量（501）
	}

	tests := []struct {
in       *MsgGetHeaders  //编码值
buf      []byte          //有线编码
pver     uint32          //有线编码协议版本
enc      MessageEncoding //消息编码格式
max      int             //引发错误的固定缓冲区的最大大小
writeErr error           //预期的写入错误
readErr  error           //预期的读取错误
	}{
//协议版本中的强制错误。
		{baseGetHeaders, baseGetHeadersEncoded, pver, BaseEncoding, 0, io.ErrShortWrite, io.EOF},
//强制块定位器哈希计数出错。
		{baseGetHeaders, baseGetHeadersEncoded, pver, BaseEncoding, 4, io.ErrShortWrite, io.EOF},
//块定位器哈希中的强制错误。
		{baseGetHeaders, baseGetHeadersEncoded, pver, BaseEncoding, 5, io.ErrShortWrite, io.EOF},
//强制停止哈希出错。
		{baseGetHeaders, baseGetHeadersEncoded, pver, BaseEncoding, 69, io.ErrShortWrite, io.EOF},
//强制使用大于最大块定位器哈希的错误。
		{maxGetHeaders, maxGetHeadersEncoded, pver, BaseEncoding, 7, wireErr, wireErr},
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
		var msg MsgGetHeaders
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
