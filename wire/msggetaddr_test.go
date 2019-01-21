
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
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

//testgetaddr测试msggetaddr API。
func TestGetAddr(t *testing.T) {
	pver := ProtocolVersion

//确保命令为预期值。
	wantCmd := "getaddr"
	msg := NewMsgGetAddr()
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgGetAddr: wrong command - got %v want %v",
			cmd, wantCmd)
	}

//确保最大有效负载是最新协议版本的预期值。
//num addresses（varint）+允许的最大地址。
	wantPayload := uint32(0)
	maxPayload := msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}
}

//testgetaddrWire测试msggetaddr线的编码和解码
//协议版本。
func TestGetAddrWire(t *testing.T) {
	msgGetAddr := NewMsgGetAddr()
	msgGetAddrEncoded := []byte{}

	tests := []struct {
in   *MsgGetAddr     //要编码的邮件
out  *MsgGetAddr     //预期的解码消息
buf  []byte          //有线编码
pver uint32          //有线编码协议版本
enc  MessageEncoding //消息编码变量。
	}{
//最新协议版本。
		{
			msgGetAddr,
			msgGetAddr,
			msgGetAddrEncoded,
			ProtocolVersion,
			BaseEncoding,
		},

//协议版本BIP0035版本。
		{
			msgGetAddr,
			msgGetAddr,
			msgGetAddrEncoded,
			BIP0035Version,
			BaseEncoding,
		},

//协议版本Bip0031版本。
		{
			msgGetAddr,
			msgGetAddr,
			msgGetAddrEncoded,
			BIP0031Version,
			BaseEncoding,
		},

//协议版本NetAddressTimeVersion。
		{
			msgGetAddr,
			msgGetAddr,
			msgGetAddrEncoded,
			NetAddressTimeVersion,
			BaseEncoding,
		},

//协议版本multipleaddressversion。
		{
			msgGetAddr,
			msgGetAddr,
			msgGetAddrEncoded,
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
		var msg MsgGetAddr
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
