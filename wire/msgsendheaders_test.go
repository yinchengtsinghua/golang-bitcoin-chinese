
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wire

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

//testsendheaders根据最新协议测试msgsendheaders API
//版本。
func TestSendHeaders(t *testing.T) {
	pver := ProtocolVersion
	enc := BaseEncoding

//确保命令为预期值。
	wantCmd := "sendheaders"
	msg := NewMsgSendHeaders()
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgSendHeaders: wrong command - got %v want %v",
			cmd, wantCmd)
	}

//确保最大有效负载为预期值。
	wantPayload := uint32(0)
	maxPayload := msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}

//使用最新的协议版本进行测试编码。
	var buf bytes.Buffer
	err := msg.BtcEncode(&buf, pver, enc)
	if err != nil {
		t.Errorf("encode of MsgSendHeaders failed %v err <%v>", msg,
			err)
	}

//旧的协议版本应该无法编码，因为消息没有
//还存在。
	oldPver := SendHeadersVersion - 1
	err = msg.BtcEncode(&buf, oldPver, enc)
	if err == nil {
		s := "encode of MsgSendHeaders passed for old protocol " +
			"version %v err <%v>"
		t.Errorf(s, msg, err)
	}

//使用最新的协议版本测试解码。
	readmsg := NewMsgSendHeaders()
	err = readmsg.BtcDecode(&buf, pver, enc)
	if err != nil {
		t.Errorf("decode of MsgSendHeaders failed [%v] err <%v>", buf,
			err)
	}

//旧的协议版本应该无法解码，因为消息没有
//还存在。
	err = readmsg.BtcDecode(&buf, oldPver, enc)
	if err == nil {
		s := "decode of MsgSendHeaders passed for old protocol " +
			"version %v err <%v>"
		t.Errorf(s, msg, err)
	}
}

//testsendheadersBip0130根据协议测试msgsendheaders API
//在版本sendHeadersVersion之前。
func TestSendHeadersBIP0130(t *testing.T) {
//在发送头版本更改之前使用协议版本。
	pver := SendHeadersVersion - 1
	enc := BaseEncoding

	msg := NewMsgSendHeaders()

//使用旧协议版本进行测试编码。
	var buf bytes.Buffer
	err := msg.BtcEncode(&buf, pver, enc)
	if err == nil {
		t.Errorf("encode of MsgSendHeaders succeeded when it should " +
			"have failed")
	}

//使用旧协议版本测试解码。
	readmsg := NewMsgSendHeaders()
	err = readmsg.BtcDecode(&buf, pver, enc)
	if err == nil {
		t.Errorf("decode of MsgSendHeaders succeeded when it should " +
			"have failed")
	}
}

//testsendheadersCrossProtocol在用编码时测试msgsendheaders API
//最新的协议版本和使用sendHeadersVersion进行解码。
func TestSendHeadersCrossProtocol(t *testing.T) {
	enc := BaseEncoding
	msg := NewMsgSendHeaders()

//使用最新的协议版本进行编码。
	var buf bytes.Buffer
	err := msg.BtcEncode(&buf, ProtocolVersion, enc)
	if err != nil {
		t.Errorf("encode of MsgSendHeaders failed %v err <%v>", msg,
			err)
	}

//使用旧协议版本解码。
	readmsg := NewMsgSendHeaders()
	err = readmsg.BtcDecode(&buf, SendHeadersVersion, enc)
	if err != nil {
		t.Errorf("decode of MsgSendHeaders failed [%v] err <%v>", buf,
			err)
	}
}

//testsendheaderswire测试msgsendheaders线的编码和解码
//各种协议版本。
func TestSendHeadersWire(t *testing.T) {
	msgSendHeaders := NewMsgSendHeaders()
	msgSendHeadersEncoded := []byte{}

	tests := []struct {
in   *MsgSendHeaders //要编码的邮件
out  *MsgSendHeaders //预期的解码消息
buf  []byte          //有线编码
pver uint32          //有线编码协议版本
enc  MessageEncoding //消息编码格式
	}{
//最新协议版本。
		{
			msgSendHeaders,
			msgSendHeaders,
			msgSendHeadersEncoded,
			ProtocolVersion,
			BaseEncoding,
		},

//协议版本发送头版本+1
		{
			msgSendHeaders,
			msgSendHeaders,
			msgSendHeadersEncoded,
			SendHeadersVersion + 1,
			BaseEncoding,
		},

//协议版本发送头版本
		{
			msgSendHeaders,
			msgSendHeaders,
			msgSendHeadersEncoded,
			SendHeadersVersion,
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
		var msg MsgSendHeaders
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
