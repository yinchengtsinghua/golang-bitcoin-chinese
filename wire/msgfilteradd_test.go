
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
)

//testfilteraddlatest根据最新协议测试msgfilteradd api
//版本。
func TestFilterAddLatest(t *testing.T) {
	enc := BaseEncoding
	pver := ProtocolVersion

	data := []byte{0x01, 0x02}
	msg := NewMsgFilterAdd(data)

//确保命令为预期值。
	wantCmd := "filteradd"
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgFilterAdd: wrong command - got %v want %v",
			cmd, wantCmd)
	}

//确保最大有效负载是最新协议版本的预期值。
	wantPayload := uint32(523)
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
		t.Errorf("encode of MsgFilterAdd failed %v err <%v>", msg, err)
	}

//使用最新的协议版本测试解码。
	var readmsg MsgFilterAdd
	err = readmsg.BtcDecode(&buf, pver, enc)
	if err != nil {
		t.Errorf("decode of MsgFilterAdd failed [%v] err <%v>", buf, err)
	}
}

//testfilterAddCrossProtocol在使用
//最新的协议版本和使用bip0031版本的解码。
func TestFilterAddCrossProtocol(t *testing.T) {
	data := []byte{0x01, 0x02}
	msg := NewMsgFilterAdd(data)
	if !bytes.Equal(msg.Data, data) {
		t.Errorf("should get same data back out")
	}

//使用最新的协议版本进行编码。
	var buf bytes.Buffer
	err := msg.BtcEncode(&buf, ProtocolVersion, LatestEncoding)
	if err != nil {
		t.Errorf("encode of MsgFilterAdd failed %v err <%v>", msg, err)
	}

//使用旧协议版本解码。
	var readmsg MsgFilterAdd
	err = readmsg.BtcDecode(&buf, BIP0031Version, LatestEncoding)
	if err == nil {
		t.Errorf("decode of MsgFilterAdd succeeded when it shouldn't "+
			"have %v", msg)
	}

//因为其中一个协议版本不支持filteradd
//消息，确保数据没有被编码和解码。
	if bytes.Equal(msg.Data, readmsg.Data) {
		t.Error("should not get same data for cross protocol")
	}

}

//testfilteraddmaxdatasize测试msgfilteradd api最大数据大小。
func TestFilterAddMaxDataSize(t *testing.T) {
	data := bytes.Repeat([]byte{0xff}, 521)
	msg := NewMsgFilterAdd(data)

//使用最新的协议版本进行编码。
	var buf bytes.Buffer
	err := msg.BtcEncode(&buf, ProtocolVersion, LatestEncoding)
	if err == nil {
		t.Errorf("encode of MsgFilterAdd succeeded when it shouldn't "+
			"have %v", msg)
	}

//使用最新的协议版本进行解码。
	readbuf := bytes.NewReader(data)
	err = msg.BtcDecode(readbuf, ProtocolVersion, LatestEncoding)
	if err == nil {
		t.Errorf("decode of MsgFilterAdd succeeded when it shouldn't "+
			"have %v", msg)
	}
}

//testfilterAddWireErrors对线编码和解码执行负测试
//添加以确认错误路径正常工作。
func TestFilterAddWireErrors(t *testing.T) {
	pver := ProtocolVersion
	pverNoFilterAdd := BIP0037Version - 1
	wireErr := &MessageError{}

	baseData := []byte{0x01, 0x02, 0x03, 0x04}
	baseFilterAdd := NewMsgFilterAdd(baseData)
	baseFilterAddEncoded := append([]byte{0x04}, baseData...)

	tests := []struct {
in       *MsgFilterAdd   //编码值
buf      []byte          //有线编码
pver     uint32          //有线编码协议版本
enc      MessageEncoding //消息编码格式
max      int             //引发错误的固定缓冲区的最大大小
writeErr error           //预期的写入错误
readErr  error           //预期的读取错误
	}{
//具有故意读/写错误的最新协议版本。
//强制数据大小出错。
		{
			baseFilterAdd, baseFilterAddEncoded, pver, BaseEncoding, 0,
			io.ErrShortWrite, io.EOF,
		},
//强制数据出错。
		{
			baseFilterAdd, baseFilterAddEncoded, pver, BaseEncoding, 1,
			io.ErrShortWrite, io.EOF,
		},
//由于协议版本不受支持而强制出错。
		{
			baseFilterAdd, baseFilterAddEncoded, pverNoFilterAdd, BaseEncoding, 5,
			wireErr, wireErr,
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
		var msg MsgFilterAdd
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
