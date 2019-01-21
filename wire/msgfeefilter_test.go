
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
	"math/rand"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

//testfefilterlatest根据最新的协议版本测试msgfeefilter API。
func TestFeeFilterLatest(t *testing.T) {
	pver := ProtocolVersion

	minfee := rand.Int63()
	msg := NewMsgFeeFilter(minfee)
	if msg.MinFee != minfee {
		t.Errorf("NewMsgFeeFilter: wrong minfee - got %v, want %v",
			msg.MinFee, minfee)
	}

//确保命令为预期值。
	wantCmd := "feefilter"
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgFeeFilter: wrong command - got %v want %v",
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

//使用最新的协议版本进行测试编码。
	var buf bytes.Buffer
	err := msg.BtcEncode(&buf, pver, BaseEncoding)
	if err != nil {
		t.Errorf("encode of MsgFeeFilter failed %v err <%v>", msg, err)
	}

//使用最新的协议版本测试解码。
	readmsg := NewMsgFeeFilter(0)
	err = readmsg.BtcDecode(&buf, pver, BaseEncoding)
	if err != nil {
		t.Errorf("decode of MsgFeeFilter failed [%v] err <%v>", buf, err)
	}

//确保minfee相同。
	if msg.MinFee != readmsg.MinFee {
		t.Errorf("Should get same minfee for protocol version %d", pver)
	}
}

//testfeefilterwire测试MSGFeefilter线对各种协议的编码和解码
//版本。
func TestFeeFilterWire(t *testing.T) {
	tests := []struct {
in   MsgFeeFilter //要编码的邮件
out  MsgFeeFilter //预期的解码消息
buf  []byte       //有线编码
pver uint32       //有线编码协议版本
	}{
//最新协议版本。
		{
MsgFeeFilter{MinFee: 123123}, //0x1E0F3
MsgFeeFilter{MinFee: 123123}, //0x1E0F3
			[]byte{0xf3, 0xe0, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00},
			ProtocolVersion,
		},

//协议版本过滤器版本
		{
MsgFeeFilter{MinFee: 456456}, //0x6F708
MsgFeeFilter{MinFee: 456456}, //0x6F708
			[]byte{0x08, 0xf7, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00},
			FeeFilterVersion,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//将邮件编码为有线格式。
		var buf bytes.Buffer
		err := test.in.BtcEncode(&buf, test.pver, BaseEncoding)
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
		var msg MsgFeeFilter
		rbuf := bytes.NewReader(test.buf)
		err = msg.BtcDecode(rbuf, test.pver, BaseEncoding)
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

//TestFeeFilterWireErrors对线编码和解码执行负测试
//以确认错误路径正常工作。
func TestFeeFilterWireErrors(t *testing.T) {
	pver := ProtocolVersion
	pverNoFeeFilter := FeeFilterVersion - 1
	wireErr := &MessageError{}

baseFeeFilter := NewMsgFeeFilter(123123) //0x1E0F3
	baseFeeFilterEncoded := []byte{
		0xf3, 0xe0, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	tests := []struct {
in       *MsgFeeFilter //编码值
buf      []byte        //有线编码
pver     uint32        //有线编码协议版本
max      int           //引发错误的固定缓冲区的最大大小
writeErr error         //预期的写入错误
readErr  error         //预期的读取错误
	}{
//具有故意读/写错误的最新协议版本。
//minfee中的强制错误。
		{baseFeeFilter, baseFeeFilterEncoded, pver, 0, io.ErrShortWrite, io.EOF},
//由于协议版本不受支持而强制出错。
		{baseFeeFilter, baseFeeFilterEncoded, pverNoFeeFilter, 4, wireErr, wireErr},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//编码为有线格式。
		w := newFixedWriter(test.max)
		err := test.in.BtcEncode(w, test.pver, BaseEncoding)
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
		var msg MsgFeeFilter
		r := newFixedReader(test.max, test.buf)
		err = msg.BtcDecode(r, test.pver, BaseEncoding)
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
