
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
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
)

//testaddr测试msgaddr api。
func TestAddr(t *testing.T) {
	pver := ProtocolVersion

//确保命令为预期值。
	wantCmd := "addr"
	msg := NewMsgAddr()
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgAddr: wrong command - got %v want %v",
			cmd, wantCmd)
	}

//确保最大有效负载是最新协议版本的预期值。
//num addresses（varint）+允许的最大地址。
	wantPayload := uint32(30009)
	maxPayload := msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}

//确保正确添加网络地址。
	tcpAddr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8333}
	na := NewNetAddress(tcpAddr, SFNodeNetwork)
	err := msg.AddAddress(na)
	if err != nil {
		t.Errorf("AddAddress: %v", err)
	}
	if msg.AddrList[0] != na {
		t.Errorf("AddAddress: wrong address added - got %v, want %v",
			spew.Sprint(msg.AddrList[0]), spew.Sprint(na))
	}

//确保正确清除地址列表。
	msg.ClearAddresses()
	if len(msg.AddrList) != 0 {
		t.Errorf("ClearAddresses: address list is not empty - "+
			"got %v [%v], want %v", len(msg.AddrList),
			spew.Sprint(msg.AddrList[0]), 0)
	}

//确保添加的地址超过每个消息返回的最大允许地址
//错误。
	for i := 0; i < MaxAddrPerMsg+1; i++ {
		err = msg.AddAddress(na)
	}
	if err == nil {
		t.Errorf("AddAddress: expected error on too many addresses " +
			"not received")
	}
	err = msg.AddAddresses(na)
	if err == nil {
		t.Errorf("AddAddresses: expected error on too many addresses " +
			"not received")
	}

//确保之前协议版本的最大有效负载为预期值
//时间戳已添加到netaddress。
//num addresses（varint）+允许的最大地址。
	pver = NetAddressTimeVersion - 1
	wantPayload = uint32(26009)
	maxPayload = msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}

//确保之前协议版本的最大有效负载为预期值
//允许多个地址。
//num addresses（varint）+单个网络地址。
	pver = MultipleAddressVersion - 1
	wantPayload = uint32(35)
	maxPayload = msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}
}

//testaddrWire测试msgaddr线对各种数字的编码和解码
//地址和协议版本。
func TestAddrWire(t *testing.T) {
//用于测试的几个网络地址。
	na := &NetAddress{
Timestamp: time.Unix(0x495fab29, 0), //
		Services:  SFNodeNetwork,
		IP:        net.ParseIP("127.0.0.1"),
		Port:      8333,
	}
	na2 := &NetAddress{
Timestamp: time.Unix(0x495fab29, 0), //2009年1月3日12:15:05-0600 cst
		Services:  SFNodeNetwork,
		IP:        net.ParseIP("192.168.0.1"),
		Port:      8334,
	}

//空地址消息。
	noAddr := NewMsgAddr()
	noAddrEncoded := []byte{
0x00, //地址数变量
	}

//多个地址的地址消息。
	multiAddr := NewMsgAddr()
	multiAddr.AddAddresses(na, na2)
	multiAddrEncoded := []byte{
0x02,                   //地址数变量
0x29, 0xab, 0x5f, 0x49, //时间戳
0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, //小字体
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
0x00, 0x00, 0xff, 0xff, 0x7f, 0x00, 0x00, 0x01, //IP127.0.0.1
0x20, 0x8d, //大端8333端口
0x29, 0xab, 0x5f, 0x49, //时间戳
0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, //小字体
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
0x00, 0x00, 0xff, 0xff, 0xc0, 0xa8, 0x00, 0x01, //IP 192.160.0.1
0x20, 0x8e, //大端8334端口

	}

	tests := []struct {
in   *MsgAddr        //要编码的邮件
out  *MsgAddr        //预期的解码消息
buf  []byte          //有线编码
pver uint32          //有线编码协议版本
enc  MessageEncoding //消息编码格式
	}{
//没有地址的最新协议版本。
		{
			noAddr,
			noAddr,
			noAddrEncoded,
			ProtocolVersion,
			BaseEncoding,
		},

//具有多个地址的最新协议版本。
		{
			multiAddr,
			multiAddr,
			multiAddrEncoded,
			ProtocolVersion,
			BaseEncoding,
		},

//协议版本multipleaddressversion-1，没有地址。
		{
			noAddr,
			noAddr,
			noAddrEncoded,
			MultipleAddressVersion - 1,
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
		var msg MsgAddr
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

//TestAddWireErrors对线编码和解码执行负测试
//以确认错误路径正常工作。
func TestAddrWireErrors(t *testing.T) {
	pver := ProtocolVersion
	pverMA := MultipleAddressVersion
	wireErr := &MessageError{}

//用于测试的几个网络地址。
	na := &NetAddress{
Timestamp: time.Unix(0x495fab29, 0), //2009年1月3日12:15:05-0600 cst
		Services:  SFNodeNetwork,
		IP:        net.ParseIP("127.0.0.1"),
		Port:      8333,
	}
	na2 := &NetAddress{
Timestamp: time.Unix(0x495fab29, 0), //2009年1月3日12:15:05-0600 cst
		Services:  SFNodeNetwork,
		IP:        net.ParseIP("192.168.0.1"),
		Port:      8334,
	}

//多个地址的地址消息。
	baseAddr := NewMsgAddr()
	baseAddr.AddAddresses(na, na2)
	baseAddrEncoded := []byte{
0x02,                   //地址数变量
0x29, 0xab, 0x5f, 0x49, //时间戳
0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, //小字体
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
0x00, 0x00, 0xff, 0xff, 0x7f, 0x00, 0x00, 0x01, //IP127.0.0.1
0x20, 0x8d, //大端8333端口
0x29, 0xab, 0x5f, 0x49, //时间戳
0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, //小字体
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
0x00, 0x00, 0xff, 0xff, 0xc0, 0xa8, 0x00, 0x01, //IP 192.160.0.1
0x20, 0x8e, //大端8334端口

	}

//通过超过允许的最大值而强制出错的消息
//地址。
	maxAddr := NewMsgAddr()
	for i := 0; i < MaxAddrPerMsg; i++ {
		maxAddr.AddAddress(na)
	}
	maxAddr.AddrList = append(maxAddr.AddrList, na)
	maxAddrEncoded := []byte{
0xfd, 0x03, 0xe9, //地址数变量（1001）
	}

	tests := []struct {
in       *MsgAddr        //编码值
buf      []byte          //有线编码
pver     uint32          //有线编码协议版本
enc      MessageEncoding //消息编码格式
max      int             //引发错误的固定缓冲区的最大大小
writeErr error           //预期的写入错误
readErr  error           //预期的读取错误
	}{
//具有故意读/写错误的最新协议版本。
//地址计数中的强制错误
		{baseAddr, baseAddrEncoded, pver, BaseEncoding, 0, io.ErrShortWrite, io.EOF},
//强制地址列表出错。
		{baseAddr, baseAddrEncoded, pver, BaseEncoding, 1, io.ErrShortWrite, io.EOF},
//强制错误大于最大库存向量。
		{maxAddr, maxAddrEncoded, pver, BaseEncoding, 3, wireErr, wireErr},
//强制错误大于最大库存向量
//允许多个地址之前的协议版本。
		{maxAddr, maxAddrEncoded, pverMA - 1, BaseEncoding, 3, wireErr, wireErr},
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
		var msg MsgAddr
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
