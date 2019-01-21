
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
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

//testfilterclearlest根据最新的测试msgfilterclear api
//协议版本。
func TestFilterClearLatest(t *testing.T) {
	pver := ProtocolVersion

	msg := NewMsgFilterClear()

//确保命令为预期值。
	wantCmd := "filterclear"
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgFilterClear: wrong command - got %v want %v",
			cmd, wantCmd)
	}

//确保最大有效负载是最新协议版本的预期值。
	wantPayload := uint32(0)
	maxPayload := msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}
}

//testfilterclearCrossProtocol使用以下代码编码时测试msgfilterclear API
//最新的协议版本和使用bip0031版本的解码。
func TestFilterClearCrossProtocol(t *testing.T) {
	msg := NewMsgFilterClear()

//使用最新的协议版本进行编码。
	var buf bytes.Buffer
	err := msg.BtcEncode(&buf, ProtocolVersion, LatestEncoding)
	if err != nil {
		t.Errorf("encode of MsgFilterClear failed %v err <%v>", msg, err)
	}

//使用旧协议版本解码。
	var readmsg MsgFilterClear
	err = readmsg.BtcDecode(&buf, BIP0031Version, LatestEncoding)
	if err == nil {
		t.Errorf("decode of MsgFilterClear succeeded when it "+
			"shouldn't have %v", msg)
	}
}

//testfilterclearwire测试msgfilterclear线的编码和解码
//各种协议版本。
func TestFilterClearWire(t *testing.T) {
	msgFilterClear := NewMsgFilterClear()
	msgFilterClearEncoded := []byte{}

	tests := []struct {
in   *MsgFilterClear //要编码的邮件
out  *MsgFilterClear //预期的解码消息
buf  []byte          //有线编码
pver uint32          //有线编码协议版本
enc  MessageEncoding //消息编码格式
	}{
//最新协议版本。
		{
			msgFilterClear,
			msgFilterClear,
			msgFilterClearEncoded,
			ProtocolVersion,
			BaseEncoding,
		},

//协议版本Bip0037版本+1。
		{
			msgFilterClear,
			msgFilterClear,
			msgFilterClearEncoded,
			BIP0037Version + 1,
			BaseEncoding,
		},

//协议版本BIP0037版本。
		{
			msgFilterClear,
			msgFilterClear,
			msgFilterClearEncoded,
			BIP0037Version,
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
		var msg MsgFilterClear
		rbuf := bytes.NewReader(test.buf)
		err = msg.BtcDecode(rbuf, test.pver, test.enc)
		if err != nil {
			t.Errorf("BtcDecode #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(&msg, test.out) {
			t.Errorf("BtcDecode #%d\n got: %s want: %s", i,
				spew.Sdump(msg), spew.Sdump(test.out))
			continue
		}
	}
}

//TestFilterClearWireErrors对线编码和
//解码msgfilterclear以确认错误路径正常工作。
func TestFilterClearWireErrors(t *testing.T) {
	pverNoFilterClear := BIP0037Version - 1
	wireErr := &MessageError{}

	baseFilterClear := NewMsgFilterClear()
	baseFilterClearEncoded := []byte{}

	tests := []struct {
in       *MsgFilterClear //编码值
buf      []byte          //有线编码
pver     uint32          //有线编码协议版本
enc      MessageEncoding //消息编码格式
max      int             //引发错误的固定缓冲区的最大大小
writeErr error           //预期的写入错误
readErr  error           //预期的读取错误
	}{
//由于协议版本不受支持而强制出错。
		{
			baseFilterClear, baseFilterClearEncoded,
			pverNoFilterClear, BaseEncoding, 4, wireErr, wireErr,
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
		var msg MsgFilterClear
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
