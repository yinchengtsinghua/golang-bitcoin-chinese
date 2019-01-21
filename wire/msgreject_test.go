
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
	"io"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

//TestRejectCodeStringer测试拒绝代码类型的字符串化输出。
func TestRejectCodeStringer(t *testing.T) {
	tests := []struct {
		in   RejectCode
		want string
	}{
		{RejectMalformed, "REJECT_MALFORMED"},
		{RejectInvalid, "REJECT_INVALID"},
		{RejectObsolete, "REJECT_OBSOLETE"},
		{RejectDuplicate, "REJECT_DUPLICATE"},
		{RejectNonstandard, "REJECT_NONSTANDARD"},
		{RejectDust, "REJECT_DUST"},
		{RejectInsufficientFee, "REJECT_INSUFFICIENTFEE"},
		{RejectCheckpoint, "REJECT_CHECKPOINT"},
		{0xff, "Unknown RejectCode (255)"},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		result := test.in.String()
		if result != test.want {
			t.Errorf("String #%d\n got: %s want: %s", i, result,
				test.want)
			continue
		}
	}

}

//testReject根据最新的协议版本测试msgpong API。
func TestRejectLatest(t *testing.T) {
	pver := ProtocolVersion
	enc := BaseEncoding

//创建拒绝消息数据。
	rejCommand := (&MsgBlock{}).Command()
	rejCode := RejectDuplicate
	rejReason := "duplicate block"
	rejHash := mainNetGenesisHash

//确保我们得到正确的数据。
	msg := NewMsgReject(rejCommand, rejCode, rejReason)
	msg.Hash = rejHash
	if msg.Cmd != rejCommand {
		t.Errorf("NewMsgReject: wrong rejected command - got %v, "+
			"want %v", msg.Cmd, rejCommand)
	}
	if msg.Code != rejCode {
		t.Errorf("NewMsgReject: wrong rejected code - got %v, "+
			"want %v", msg.Code, rejCode)
	}
	if msg.Reason != rejReason {
		t.Errorf("NewMsgReject: wrong rejected reason - got %v, "+
			"want %v", msg.Reason, rejReason)
	}

//确保命令为预期值。
	wantCmd := "reject"
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgReject: wrong command - got %v want %v",
			cmd, wantCmd)
	}

//确保最大有效负载是最新协议版本的预期值。
	wantPayload := uint32(MaxMessagePayload)
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
		t.Errorf("encode of MsgReject failed %v err <%v>", msg, err)
	}

//使用最新的协议版本测试解码。
	readMsg := MsgReject{}
	err = readMsg.BtcDecode(&buf, pver, enc)
	if err != nil {
		t.Errorf("decode of MsgReject failed %v err <%v>", buf.Bytes(),
			err)
	}

//确保解码数据相同。
	if msg.Cmd != readMsg.Cmd {
		t.Errorf("Should get same reject command - got %v, want %v",
			readMsg.Cmd, msg.Cmd)
	}
	if msg.Code != readMsg.Code {
		t.Errorf("Should get same reject code - got %v, want %v",
			readMsg.Code, msg.Code)
	}
	if msg.Reason != readMsg.Reason {
		t.Errorf("Should get same reject reason - got %v, want %v",
			readMsg.Reason, msg.Reason)
	}
	if msg.Hash != readMsg.Hash {
		t.Errorf("Should get same reject hash - got %v, want %v",
			readMsg.Hash, msg.Hash)
	}
}

//testRejectBeforeAdded根据协议版本测试msgreject api
//在引入它的版本之前（拒绝版本）。
func TestRejectBeforeAdded(t *testing.T) {
//在拒绝版本之前使用协议版本。
	pver := RejectVersion - 1
	enc := BaseEncoding

//创建拒绝消息数据。
	rejCommand := (&MsgBlock{}).Command()
	rejCode := RejectDuplicate
	rejReason := "duplicate block"
	rejHash := mainNetGenesisHash

	msg := NewMsgReject(rejCommand, rejCode, rejReason)
	msg.Hash = rejHash

//确保旧协议版本的最大负载为预期值。
	size := msg.MaxPayloadLength(pver)
	if size != 0 {
		t.Errorf("Max length should be 0 for reject protocol version %d.",
			pver)
	}

//使用旧协议版本进行测试编码。
	var buf bytes.Buffer
	err := msg.BtcEncode(&buf, pver, enc)
	if err == nil {
		t.Errorf("encode of MsgReject succeeded when it shouldn't "+
			"have %v", msg)
	}

////使用旧协议版本测试解码。
	readMsg := MsgReject{}
	err = readMsg.BtcDecode(&buf, pver, enc)
	if err == nil {
		t.Errorf("decode of MsgReject succeeded when it shouldn't "+
			"have %v", spew.Sdump(buf.Bytes()))
	}

//由于此协议版本不支持拒绝，请确保
//字段没有被编码和解码。
	if msg.Cmd == readMsg.Cmd {
		t.Errorf("Should not get same reject command for protocol "+
			"version %d", pver)
	}
	if msg.Code == readMsg.Code {
		t.Errorf("Should not get same reject code for protocol "+
			"version %d", pver)
	}
	if msg.Reason == readMsg.Reason {
		t.Errorf("Should not get same reject reason for protocol "+
			"version %d", pver)
	}
	if msg.Hash == readMsg.Hash {
		t.Errorf("Should not get same reject hash for protocol "+
			"version %d", pver)
	}
}

//TestRejectCrossProtocol在使用最新的
//协议版本，并使用该版本之前的版本进行解码
//介绍了它（拒绝版本）。
func TestRejectCrossProtocol(t *testing.T) {
//创建拒绝消息数据。
	rejCommand := (&MsgBlock{}).Command()
	rejCode := RejectDuplicate
	rejReason := "duplicate block"
	rejHash := mainNetGenesisHash

	msg := NewMsgReject(rejCommand, rejCode, rejReason)
	msg.Hash = rejHash

//使用最新的协议版本进行编码。
	var buf bytes.Buffer
	err := msg.BtcEncode(&buf, ProtocolVersion, BaseEncoding)
	if err != nil {
		t.Errorf("encode of MsgReject failed %v err <%v>", msg, err)
	}

//使用旧协议版本解码。
	readMsg := MsgReject{}
	err = readMsg.BtcDecode(&buf, RejectVersion-1, BaseEncoding)
	if err == nil {
		t.Errorf("encode of MsgReject succeeded when it shouldn't "+
			"have %v", msg)
	}

//因为其中一个协议版本不支持拒绝
//消息，确保各个字段没有被编码和解码
//退出。
	if msg.Cmd == readMsg.Cmd {
		t.Errorf("Should not get same reject command for cross protocol")
	}
	if msg.Code == readMsg.Code {
		t.Errorf("Should not get same reject code for cross protocol")
	}
	if msg.Reason == readMsg.Reason {
		t.Errorf("Should not get same reject reason for cross protocol")
	}
	if msg.Hash == readMsg.Hash {
		t.Errorf("Should not get same reject hash for cross protocol")
	}
}

//testRejectWire测试msgreject Wire编码和解码
//协议版本。
func TestRejectWire(t *testing.T) {
	tests := []struct {
msg  MsgReject       //要编码的邮件
buf  []byte          //有线编码
pver uint32          //有线编码协议版本
enc  MessageEncoding //消息编码格式
	}{
//最新的协议版本拒绝了命令版本（无哈希）。
		{
			MsgReject{
				Cmd:    "version",
				Code:   RejectDuplicate,
				Reason: "duplicate version",
			},
			[]byte{
0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, //“版本”
0x12, //拒绝复制
				0x11, 0x64, 0x75, 0x70, 0x6c, 0x69, 0x63, 0x61,
				0x74, 0x65, 0x20, 0x76, 0x65, 0x72, 0x73, 0x69,
0x6f, 0x6e, //“重复版本”
			},
			ProtocolVersion,
			BaseEncoding,
		},
//最新的协议版本拒绝了命令块（具有哈希）。
		{
			MsgReject{
				Cmd:    "block",
				Code:   RejectDuplicate,
				Reason: "duplicate block",
				Hash:   mainNetGenesisHash,
			},
			[]byte{
0x05, 0x62, 0x6c, 0x6f, 0x63, 0x6b, //“拦网”
0x12, //拒绝复制
				0x0f, 0x64, 0x75, 0x70, 0x6c, 0x69, 0x63, 0x61,
0x74, 0x65, 0x20, 0x62, 0x6c, 0x6f, 0x63, 0x6b, //“重复块”
				0x6f, 0xe2, 0x8c, 0x0a, 0xb6, 0xf1, 0xb3, 0x72,
				0xc1, 0xa6, 0xa2, 0x46, 0xae, 0x63, 0xf7, 0x4f,
				0x93, 0x1e, 0x83, 0x65, 0xe1, 0x5a, 0x08, 0x9c,
0x68, 0xd6, 0x19, 0x00, 0x00, 0x00, 0x00, 0x00, //MainnegenesHash
			},
			ProtocolVersion,
			BaseEncoding,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//将邮件编码为有线格式。
		var buf bytes.Buffer
		err := test.msg.BtcEncode(&buf, test.pver, test.enc)
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
		var msg MsgReject
		rbuf := bytes.NewReader(test.buf)
		err = msg.BtcDecode(rbuf, test.pver, test.enc)
		if err != nil {
			t.Errorf("BtcDecode #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(msg, test.msg) {
			t.Errorf("BtcDecode #%d\n got: %s want: %s", i,
				spew.Sdump(msg), spew.Sdump(test.msg))
			continue
		}
	}
}

//TestRejectWireErrors对线编码和解码执行负测试
//以确认错误路径正常工作。
func TestRejectWireErrors(t *testing.T) {
	pver := ProtocolVersion
	pverNoReject := RejectVersion - 1
	wireErr := &MessageError{}

	baseReject := NewMsgReject("block", RejectDuplicate, "duplicate block")
	baseReject.Hash = mainNetGenesisHash
	baseRejectEncoded := []byte{
0x05, 0x62, 0x6c, 0x6f, 0x63, 0x6b, //“拦网”
0x12, //拒绝复制
		0x0f, 0x64, 0x75, 0x70, 0x6c, 0x69, 0x63, 0x61,
0x74, 0x65, 0x20, 0x62, 0x6c, 0x6f, 0x63, 0x6b, //“重复块”
		0x6f, 0xe2, 0x8c, 0x0a, 0xb6, 0xf1, 0xb3, 0x72,
		0xc1, 0xa6, 0xa2, 0x46, 0xae, 0x63, 0xf7, 0x4f,
		0x93, 0x1e, 0x83, 0x65, 0xe1, 0x5a, 0x08, 0x9c,
0x68, 0xd6, 0x19, 0x00, 0x00, 0x00, 0x00, 0x00, //MainnegenesHash
	}

	tests := []struct {
in       *MsgReject      //编码值
buf      []byte          //有线编码
pver     uint32          //有线编码协议版本
enc      MessageEncoding //消息编码格式
max      int             //引发错误的固定缓冲区的最大大小
writeErr error           //预期的写入错误
readErr  error           //预期的读取错误
	}{
//具有故意读/写错误的最新协议版本。
//拒绝命令中的强制错误。
		{baseReject, baseRejectEncoded, pver, BaseEncoding, 0, io.ErrShortWrite, io.EOF},
//拒绝代码中的强制错误。
		{baseReject, baseRejectEncoded, pver, BaseEncoding, 6, io.ErrShortWrite, io.EOF},
//拒绝原因中的强制错误。
		{baseReject, baseRejectEncoded, pver, BaseEncoding, 7, io.ErrShortWrite, io.EOF},
//拒绝哈希中的强制错误。
		{baseReject, baseRejectEncoded, pver, BaseEncoding, 23, io.ErrShortWrite, io.EOF},
//由于协议版本不受支持而强制出错。
		{baseReject, baseRejectEncoded, pverNoReject, BaseEncoding, 6, wireErr, wireErr},
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
		var msg MsgReject
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
