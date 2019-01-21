
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

//testping根据最新的协议版本测试msgping API。
func TestPing(t *testing.T) {
	pver := ProtocolVersion

//确保我们把同样的东西拿出来。
	nonce, err := RandomUint64()
	if err != nil {
		t.Errorf("RandomUint64: Error generating nonce: %v", err)
	}
	msg := NewMsgPing(nonce)
	if msg.Nonce != nonce {
		t.Errorf("NewMsgPing: wrong nonce - got %v, want %v",
			msg.Nonce, nonce)
	}

//确保命令为预期值。
	wantCmd := "ping"
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgPing: wrong command - got %v want %v",
			cmd, wantCmd)
	}

//确保最大有效负载是最新协议版本的预期值。
	wantPayload := uint32(8)
	maxPayload := msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}
}

//testpingbip0031根据协议版本测试msgping api
//BIP031版本。
func TestPingBIP0031(t *testing.T) {
//在更改bip0031版本之前使用协议版本。
	pver := BIP0031Version
	enc := BaseEncoding

	nonce, err := RandomUint64()
	if err != nil {
		t.Errorf("RandomUint64: Error generating nonce: %v", err)
	}
	msg := NewMsgPing(nonce)
	if msg.Nonce != nonce {
		t.Errorf("NewMsgPing: wrong nonce - got %v, want %v",
			msg.Nonce, nonce)
	}

//确保旧协议版本的最大负载为预期值。
	wantPayload := uint32(0)
	maxPayload := msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}

//使用旧协议版本进行测试编码。
	var buf bytes.Buffer
	err = msg.BtcEncode(&buf, pver, enc)
	if err != nil {
		t.Errorf("encode of MsgPing failed %v err <%v>", msg, err)
	}

//使用旧协议版本测试解码。
	readmsg := NewMsgPing(0)
	err = readmsg.BtcDecode(&buf, pver, enc)
	if err != nil {
		t.Errorf("decode of MsgPing failed [%v] err <%v>", buf, err)
	}

//由于此协议版本不支持nonce，请确保
//它没有被编码和解码出来。
	if msg.Nonce == readmsg.Nonce {
		t.Errorf("Should not get same nonce for protocol version %d", pver)
	}
}

//TestPingCrossProtocol在使用最新的
//协议版本和使用bip0031版本解码。
func TestPingCrossProtocol(t *testing.T) {
	nonce, err := RandomUint64()
	if err != nil {
		t.Errorf("RandomUint64: Error generating nonce: %v", err)
	}
	msg := NewMsgPing(nonce)
	if msg.Nonce != nonce {
		t.Errorf("NewMsgPing: wrong nonce - got %v, want %v",
			msg.Nonce, nonce)
	}

//使用最新的协议版本进行编码。
	var buf bytes.Buffer
	err = msg.BtcEncode(&buf, ProtocolVersion, BaseEncoding)
	if err != nil {
		t.Errorf("encode of MsgPing failed %v err <%v>", msg, err)
	}

//使用旧协议版本解码。
	readmsg := NewMsgPing(0)
	err = readmsg.BtcDecode(&buf, BIP0031Version, BaseEncoding)
	if err != nil {
		t.Errorf("decode of MsgPing failed [%v] err <%v>", buf, err)
	}

//因为其中一个协议版本不支持nonce，所以
//当然，它没有被编码和解码出来。
	if msg.Nonce == readmsg.Nonce {
		t.Error("Should not get same nonce for cross protocol")
	}
}

//TestPingWire测试MSGPing线对各种协议的编码和解码
//版本。
func TestPingWire(t *testing.T) {
	tests := []struct {
in   MsgPing         //要编码的邮件
out  MsgPing         //预期的解码消息
buf  []byte          //有线编码
pver uint32          //有线编码协议版本
enc  MessageEncoding //消息编码格式
	}{
//最新协议版本。
		{
MsgPing{Nonce: 123123}, //0x1E0F3
MsgPing{Nonce: 123123}, //0x1E0F3
			[]byte{0xf3, 0xe0, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00},
			ProtocolVersion,
			BaseEncoding,
		},

//协议版本Bip0031版本+1
		{
MsgPing{Nonce: 456456}, //0x6F708
MsgPing{Nonce: 456456}, //0x6F708
			[]byte{0x08, 0xf7, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00},
			BIP0031Version + 1,
			BaseEncoding,
		},

//协议版本Bip0031版本
		{
MsgPing{Nonce: 789789}, //0xC0D1D
MsgPing{Nonce: 0},      //对PVER来说不是很重要
[]byte{},               //对PVER来说不是很重要
			BIP0031Version,
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
		var msg MsgPing
		rbuf := bytes.NewReader(test.buf)
		err = msg.BtcDecode(rbuf, test.pver, test.enc)
		if err != nil {
			t.Errorf("BtcDecode #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(msg, test.out) {
			t.Errorf("BtcDecode #%d\n got: %s want: %s", i,
				spew.Sdump(msg), spew.Sdump(test.out))
			continue
		}
	}
}

//TestPingWireErrors对线编码和解码执行负测试
//以确认错误路径正常工作。
func TestPingWireErrors(t *testing.T) {
	pver := ProtocolVersion

	tests := []struct {
in       *MsgPing        //编码值
buf      []byte          //有线编码
pver     uint32          //有线编码协议版本
enc      MessageEncoding //消息编码格式
max      int             //引发错误的固定缓冲区的最大大小
writeErr error           //预期的写入错误
readErr  error           //预期的读取错误
	}{
//具有故意读/写错误的最新协议版本。
		{
&MsgPing{Nonce: 123123}, //0x1E0F3
			[]byte{0xf3, 0xe0, 0x01, 0x00},
			pver,
			BaseEncoding,
			2,
			io.ErrShortWrite,
			io.ErrUnexpectedEOF,
		},
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
		var msg MsgPing
		r := newFixedReader(test.max, test.buf)
		err = msg.BtcDecode(r, test.pver, test.enc)
		if err != test.readErr {
			t.Errorf("BtcDecode #%d wrong error got: %v, want: %v",
				i, err, test.readErr)
			continue
		}
	}
}
