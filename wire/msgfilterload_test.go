
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

//testfilterclearnest根据最新协议测试msgfilterload api
//版本。
func TestFilterLoadLatest(t *testing.T) {
	pver := ProtocolVersion
	enc := BaseEncoding

	data := []byte{0x01, 0x02}
	msg := NewMsgFilterLoad(data, 10, 0, 0)

//确保命令为预期值。
	wantCmd := "filterload"
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgFilterLoad: wrong command - got %v want %v",
			cmd, wantCmd)
	}

//确保最大有效负载是最新协议版本的预期值。
	wantPayload := uint32(36012)
	maxPayload := msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayLoadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}

//使用最新的协议版本进行测试编码。
	var buf bytes.Buffer
	err := msg.BtcEncode(&buf, pver, enc)
	if err != nil {
		t.Errorf("encode of MsgFilterLoad failed %v err <%v>", msg, err)
	}

//使用最新的协议版本测试解码。
	readmsg := MsgFilterLoad{}
	err = readmsg.BtcDecode(&buf, pver, enc)
	if err != nil {
		t.Errorf("decode of MsgFilterLoad failed [%v] err <%v>", buf, err)
	}
}

//testfilterloadcrossProtocol在使用
//最新的协议版本和使用bip0031版本的解码。
func TestFilterLoadCrossProtocol(t *testing.T) {
	data := []byte{0x01, 0x02}
	msg := NewMsgFilterLoad(data, 10, 0, 0)

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

//testfilterloadmaxfiltersize测试msgfilterload api最大筛选器大小。
func TestFilterLoadMaxFilterSize(t *testing.T) {
	data := bytes.Repeat([]byte{0xff}, 36001)
	msg := NewMsgFilterLoad(data, 10, 0, 0)

//使用最新的协议版本进行编码。
	var buf bytes.Buffer
	err := msg.BtcEncode(&buf, ProtocolVersion, BaseEncoding)
	if err == nil {
		t.Errorf("encode of MsgFilterLoad succeeded when it shouldn't "+
			"have %v", msg)
	}

//使用最新的协议版本进行解码。
	readbuf := bytes.NewReader(data)
	err = msg.BtcDecode(readbuf, ProtocolVersion, BaseEncoding)
	if err == nil {
		t.Errorf("decode of MsgFilterLoad succeeded when it shouldn't "+
			"have %v", msg)
	}
}

//testfilterloadmaxhashfuncssize测试msgfilterload api最大哈希函数。
func TestFilterLoadMaxHashFuncsSize(t *testing.T) {
	data := bytes.Repeat([]byte{0xff}, 10)
	msg := NewMsgFilterLoad(data, 61, 0, 0)

//使用最新的协议版本进行编码。
	var buf bytes.Buffer
	err := msg.BtcEncode(&buf, ProtocolVersion, BaseEncoding)
	if err == nil {
		t.Errorf("encode of MsgFilterLoad succeeded when it shouldn't have %v",
			msg)
	}

	newBuf := []byte{
0x0a,                                                       //滤波器尺寸
0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, //滤波器
0x3d, 0x00, 0x00, 0x00, //最大哈希函数
0x00, 0x00, 0x00, 0x00, //扭
0x00, //更新类型
	}
//使用最新的协议版本进行解码。
	readbuf := bytes.NewReader(newBuf)
	err = msg.BtcDecode(readbuf, ProtocolVersion, BaseEncoding)
	if err == nil {
		t.Errorf("decode of MsgFilterLoad succeeded when it shouldn't have %v",
			msg)
	}
}

//testfilterloadwireerrors对线编码和解码执行负测试
//以确认错误路径是否正常工作。
func TestFilterLoadWireErrors(t *testing.T) {
	pver := ProtocolVersion
	pverNoFilterLoad := BIP0037Version - 1
	wireErr := &MessageError{}

	baseFilter := []byte{0x01, 0x02, 0x03, 0x04}
	baseFilterLoad := NewMsgFilterLoad(baseFilter, 10, 0, BloomUpdateNone)
	baseFilterLoadEncoded := append([]byte{0x04}, baseFilter...)
	baseFilterLoadEncoded = append(baseFilterLoadEncoded,
0x00, 0x00, 0x00, 0x0a, //哈什曼斯
0x00, 0x00, 0x00, 0x00, //扭
0x00) //旗帜

	tests := []struct {
in       *MsgFilterLoad  //编码值
buf      []byte          //有线编码
pver     uint32          //有线编码协议版本
enc      MessageEncoding //消息编码格式
max      int             //引发错误的固定缓冲区的最大大小
writeErr error           //预期的写入错误
readErr  error           //预期的读取错误
	}{
//具有故意读/写错误的最新协议版本。
//筛选器大小中的强制错误。
		{
			baseFilterLoad, baseFilterLoadEncoded, pver, BaseEncoding, 0,
			io.ErrShortWrite, io.EOF,
		},
//过滤器中的强制错误。
		{
			baseFilterLoad, baseFilterLoadEncoded, pver, BaseEncoding, 1,
			io.ErrShortWrite, io.EOF,
		},
//哈希函数中的强制错误。
		{
			baseFilterLoad, baseFilterLoadEncoded, pver, BaseEncoding, 5,
			io.ErrShortWrite, io.EOF,
		},
//强制调整错误。
		{
			baseFilterLoad, baseFilterLoadEncoded, pver, BaseEncoding, 9,
			io.ErrShortWrite, io.EOF,
		},
//强制标记出错。
		{
			baseFilterLoad, baseFilterLoadEncoded, pver, BaseEncoding, 13,
			io.ErrShortWrite, io.EOF,
		},
//由于协议版本不受支持而强制出错。
		{
			baseFilterLoad, baseFilterLoadEncoded, pverNoFilterLoad, BaseEncoding,
			10, wireErr, wireErr,
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
		var msg MsgFilterLoad
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
