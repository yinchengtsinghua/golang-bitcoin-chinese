
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
	"strings"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
)

//测试版本测试msgversion api。
func TestVersion(t *testing.T) {
	pver := ProtocolVersion

//创建版本消息数据。
	lastBlock := int32(234234)
	tcpAddrMe := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8333}
	me := NewNetAddress(tcpAddrMe, SFNodeNetwork)
	tcpAddrYou := &net.TCPAddr{IP: net.ParseIP("192.168.0.1"), Port: 8333}
	you := NewNetAddress(tcpAddrYou, SFNodeNetwork)
	nonce, err := RandomUint64()
	if err != nil {
		t.Errorf("RandomUint64: error generating nonce: %v", err)
	}

//确保我们得到正确的数据。
	msg := NewMsgVersion(me, you, nonce, lastBlock)
	if msg.ProtocolVersion != int32(pver) {
		t.Errorf("NewMsgVersion: wrong protocol version - got %v, want %v",
			msg.ProtocolVersion, pver)
	}
	if !reflect.DeepEqual(&msg.AddrMe, me) {
		t.Errorf("NewMsgVersion: wrong me address - got %v, want %v",
			spew.Sdump(&msg.AddrMe), spew.Sdump(me))
	}
	if !reflect.DeepEqual(&msg.AddrYou, you) {
		t.Errorf("NewMsgVersion: wrong you address - got %v, want %v",
			spew.Sdump(&msg.AddrYou), spew.Sdump(you))
	}
	if msg.Nonce != nonce {
		t.Errorf("NewMsgVersion: wrong nonce - got %v, want %v",
			msg.Nonce, nonce)
	}
	if msg.UserAgent != DefaultUserAgent {
		t.Errorf("NewMsgVersion: wrong user agent - got %v, want %v",
			msg.UserAgent, DefaultUserAgent)
	}
	if msg.LastBlock != lastBlock {
		t.Errorf("NewMsgVersion: wrong last block - got %v, want %v",
			msg.LastBlock, lastBlock)
	}
	if msg.DisableRelayTx {
		t.Errorf("NewMsgVersion: disable relay tx is not false by "+
			"default - got %v, want %v", msg.DisableRelayTx, false)
	}

	msg.AddUserAgent("myclient", "1.2.3", "optional", "comments")
	customUserAgent := DefaultUserAgent + "myclient:1.2.3(optional; comments)/"
	if msg.UserAgent != customUserAgent {
		t.Errorf("AddUserAgent: wrong user agent - got %s, want %s",
			msg.UserAgent, customUserAgent)
	}

	msg.AddUserAgent("mygui", "3.4.5")
	customUserAgent += "mygui:3.4.5/"
	if msg.UserAgent != customUserAgent {
		t.Errorf("AddUserAgent: wrong user agent - got %s, want %s",
			msg.UserAgent, customUserAgent)
	}

//会计处理“：”，/“
	err = msg.AddUserAgent(strings.Repeat("t",
		MaxUserAgentLen-len(customUserAgent)-2+1), "")
	if _, ok := err.(*MessageError); !ok {
		t.Errorf("AddUserAgent: expected error not received "+
			"- got %v, want %T", err, MessageError{})

	}

//默认情况下，版本消息不应设置任何服务。
	if msg.Services != 0 {
		t.Errorf("NewMsgVersion: wrong default services - got %v, want %v",
			msg.Services, 0)

	}
	if msg.HasService(SFNodeNetwork) {
		t.Errorf("HasService: SFNodeNetwork service is set")
	}

//确保命令为预期值。
	wantCmd := "version"
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgVersion: wrong command - got %v want %v",
			cmd, wantCmd)
	}

//确保最大有效负载为预期值。
//协议版本4字节+服务8字节+时间戳8字节+
//远程和本地网络地址+nonce 8字节+用户代理的长度
//（变量）+允许的最大用户代理长度+最后一个块4字节+
//中继事务标志1字节。
	wantPayload := uint32(358)
	maxPayload := msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}

//确保添加完整服务节点标志有效。
	msg.AddService(SFNodeNetwork)
	if msg.Services != SFNodeNetwork {
		t.Errorf("AddService: wrong services - got %v, want %v",
			msg.Services, SFNodeNetwork)
	}
	if !msg.HasService(SFNodeNetwork) {
		t.Errorf("HasService: SFNodeNetwork service not set")
	}
}

//TestVersionWire测试MSGVersion Wire编码和解码
//协议版本。
func TestVersionWire(t *testing.T) {
//verrelaytxfalse和verrelaytxfalseencoded是截至的版本消息
//BIP0037版本，禁用事务中继。
	baseVersionBIP0037Copy := *baseVersionBIP0037
	verRelayTxFalse := &baseVersionBIP0037Copy
	verRelayTxFalse.DisableRelayTx = true
	verRelayTxFalseEncoded := make([]byte, len(baseVersionBIP0037Encoded))
	copy(verRelayTxFalseEncoded, baseVersionBIP0037Encoded)
	verRelayTxFalseEncoded[len(verRelayTxFalseEncoded)-1] = 0

	tests := []struct {
in   *MsgVersion     //要编码的邮件
out  *MsgVersion     //预期的解码消息
buf  []byte          //有线编码
pver uint32          //有线编码协议版本
enc  MessageEncoding //消息编码格式
	}{
//最新协议版本。
		{
			baseVersionBIP0037,
			baseVersionBIP0037,
			baseVersionBIP0037Encoded,
			ProtocolVersion,
			BaseEncoding,
		},

//协议版本BIP0037VERSION带中继事务字段
//真的。
		{
			baseVersionBIP0037,
			baseVersionBIP0037,
			baseVersionBIP0037Encoded,
			BIP0037Version,
			BaseEncoding,
		},

//协议版本BIP0037VERSION带中继事务字段
//错误的。
		{
			verRelayTxFalse,
			verRelayTxFalse,
			verRelayTxFalseEncoded,
			BIP0037Version,
			BaseEncoding,
		},

//协议版本BIP0035版本。
		{
			baseVersion,
			baseVersion,
			baseVersionEncoded,
			BIP0035Version,
			BaseEncoding,
		},

//协议版本Bip0031版本。
		{
			baseVersion,
			baseVersion,
			baseVersionEncoded,
			BIP0031Version,
			BaseEncoding,
		},

//协议版本NetAddressTimeVersion。
		{
			baseVersion,
			baseVersion,
			baseVersionEncoded,
			NetAddressTimeVersion,
			BaseEncoding,
		},

//协议版本multipleaddressversion。
		{
			baseVersion,
			baseVersion,
			baseVersionEncoded,
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
		var msg MsgVersion
		rbuf := bytes.NewBuffer(test.buf)
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

//TestVersionWireErrors对线编码和
//解码消息生成器以确认错误路径正常工作。
func TestVersionWireErrors(t *testing.T) {
//在这里特别使用协议版本60002，而不是最新版本
//因为测试数据使用的是用该协议编码的字节
//版本。
	pver := uint32(60002)
	enc := BaseEncoding
	wireErr := &MessageError{}

//确保使用非*字节调用msgversion.btcdecode。buffer返回
//错误。
	fr := newFixedReader(0, []byte{})
	if err := baseVersion.BtcDecode(fr, pver, enc); err == nil {
		t.Errorf("Did not received error when calling " +
			"MsgVersion.BtcDecode with non *bytes.Buffer")
	}

//复制基本版本并将用户代理更改为超过最大限制。
	bvc := *baseVersion
	exceedUAVer := &bvc
	newUA := "/" + strings.Repeat("t", MaxUserAgentLen-8+1) + ":0.0.1/"
	exceedUAVer.UserAgent = newUA

//将新的UA长度编码为变量。
	var newUAVarIntBuf bytes.Buffer
	err := WriteVarInt(&newUAVarIntBuf, pver, uint64(len(newUA)))
	if err != nil {
		t.Errorf("WriteVarInt: error %v", err)
	}

//创建一个足够大的新缓冲区来容纳基本版本和新的
//用于较大变量保存用户代理新大小的字节数
//以及新的用户代理字符串。然后把它们粘在一起。
	newLen := len(baseVersionEncoded) - len(baseVersion.UserAgent)
	newLen = newLen + len(newUAVarIntBuf.Bytes()) - 1 + len(newUA)
	exceedUAVerEncoded := make([]byte, newLen)
	copy(exceedUAVerEncoded, baseVersionEncoded[0:80])
	copy(exceedUAVerEncoded[80:], newUAVarIntBuf.Bytes())
	copy(exceedUAVerEncoded[83:], []byte(newUA))
	copy(exceedUAVerEncoded[83+len(newUA):], baseVersionEncoded[97:100])

	tests := []struct {
in       *MsgVersion     //编码值
buf      []byte          //有线编码
pver     uint32          //有线编码协议版本
enc      MessageEncoding //消息编码格式
max      int             //引发错误的固定缓冲区的最大大小
writeErr error           //预期的写入错误
readErr  error           //预期的读取错误
	}{
//协议版本中的强制错误。
		{baseVersion, baseVersionEncoded, pver, BaseEncoding, 0, io.ErrShortWrite, io.EOF},
//服务中的强制错误。
		{baseVersion, baseVersionEncoded, pver, BaseEncoding, 4, io.ErrShortWrite, io.EOF},
//强制时间戳出错。
		{baseVersion, baseVersionEncoded, pver, BaseEncoding, 12, io.ErrShortWrite, io.EOF},
//强制远程地址出错。
		{baseVersion, baseVersionEncoded, pver, BaseEncoding, 20, io.ErrShortWrite, io.EOF},
//强制本地地址出错。
		{baseVersion, baseVersionEncoded, pver, BaseEncoding, 47, io.ErrShortWrite, io.ErrUnexpectedEOF},
//当前强制错误。
		{baseVersion, baseVersionEncoded, pver, BaseEncoding, 73, io.ErrShortWrite, io.ErrUnexpectedEOF},
//强制用户代理长度出错。
		{baseVersion, baseVersionEncoded, pver, BaseEncoding, 81, io.ErrShortWrite, io.EOF},
//在用户代理中强制出错。
		{baseVersion, baseVersionEncoded, pver, BaseEncoding, 82, io.ErrShortWrite, io.ErrUnexpectedEOF},
//最后一个块中的强制错误。
		{baseVersion, baseVersionEncoded, pver, BaseEncoding, 98, io.ErrShortWrite, io.ErrUnexpectedEOF},
//继电器Tx中的强制错误-此后不应发生读取错误
//这是可选的。
		{
			baseVersionBIP0037, baseVersionBIP0037Encoded,
			BIP0037Version, BaseEncoding, 101, io.ErrShortWrite, nil,
		},
//由于用户代理太大而强制出错
		{exceedUAVer, exceedUAVerEncoded, pver, BaseEncoding, newLen, wireErr, wireErr},
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
		var msg MsgVersion
		buf := bytes.NewBuffer(test.buf[0:test.max])
		err = msg.BtcDecode(buf, test.pver, test.enc)
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

//TestVersionOptionalFields执行测试以确保编码的版本
//忽略可选字段的消息处理正确。
func TestVersionOptionalFields(t *testing.T) {
//OnlyRequiredVersion是只包含
//所需版本和所有其他值设置为其默认值。
	onlyRequiredVersion := MsgVersion{
		ProtocolVersion: 60002,
		Services:        SFNodeNetwork,
Timestamp:       time.Unix(0x495fab29, 0), //2009年1月3日12:15:05-0600 cst）
		AddrYou: NetAddress{
Timestamp: time.Time{}, //零值--版本中没有时间戳
			Services:  SFNodeNetwork,
			IP:        net.ParseIP("192.168.0.1"),
			Port:      8333,
		},
	}
	onlyRequiredVersionEncoded := make([]byte, len(baseVersionEncoded)-55)
	copy(onlyRequiredVersionEncoded, baseVersionEncoded)

//addrmeversion是一个版本消息，包含通过
//addrme字段。
	addrMeVersion := onlyRequiredVersion
	addrMeVersion.AddrMe = NetAddress{
Timestamp: time.Time{}, //零值--版本中没有时间戳
		Services:  SFNodeNetwork,
		IP:        net.ParseIP("127.0.0.1"),
		Port:      8333,
	}
	addrMeVersionEncoded := make([]byte, len(baseVersionEncoded)-29)
	copy(addrMeVersionEncoded, baseVersionEncoded)

//非转换是一个版本消息，包含
//nonce字段。
	nonceVersion := addrMeVersion
nonceVersion.Nonce = 123123 //0x1E0F3
	nonceVersionEncoded := make([]byte, len(baseVersionEncoded)-21)
	copy(nonceVersionEncoded, baseVersionEncoded)

//uaversion是一个版本消息，包含通过
//用户代理字段。
	uaVersion := nonceVersion
	uaVersion.UserAgent = "/btcdtest:0.0.1/"
	uaVersionEncoded := make([]byte, len(baseVersionEncoded)-4)
	copy(uaVersionEncoded, baseVersionEncoded)

//LastBlockVersion是包含所有字段的版本消息
//通过lastblock字段。
	lastBlockVersion := uaVersion
lastBlockVersion.LastBlock = 234234 //0x39 2FA
	lastBlockVersionEncoded := make([]byte, len(baseVersionEncoded))
	copy(lastBlockVersionEncoded, baseVersionEncoded)

	tests := []struct {
msg  *MsgVersion     //预期消息
buf  []byte          //有线编码
pver uint32          //有线编码协议版本
enc  MessageEncoding //消息编码格式
	}{
		{
			&onlyRequiredVersion,
			onlyRequiredVersionEncoded,
			ProtocolVersion,
			BaseEncoding,
		},
		{
			&addrMeVersion,
			addrMeVersionEncoded,
			ProtocolVersion,
			BaseEncoding,
		},
		{
			&nonceVersion,
			nonceVersionEncoded,
			ProtocolVersion,
			BaseEncoding,
		},
		{
			&uaVersion,
			uaVersionEncoded,
			ProtocolVersion,
			BaseEncoding,
		},
		{
			&lastBlockVersion,
			lastBlockVersionEncoded,
			ProtocolVersion,
			BaseEncoding,
		},
	}

	for i, test := range tests {
//从有线格式解码消息。
		var msg MsgVersion
		rbuf := bytes.NewBuffer(test.buf)
		err := msg.BtcDecode(rbuf, test.pver, test.enc)
		if err != nil {
			t.Errorf("BtcDecode #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(&msg, test.msg) {
			t.Errorf("BtcDecode #%d\n got: %s want: %s", i,
				spew.Sdump(msg), spew.Sdump(test.msg))
			continue
		}
	}
}

//baseversion在各种测试中用作基线msgversion。
var baseVersion = &MsgVersion{
	ProtocolVersion: 60002,
	Services:        SFNodeNetwork,
Timestamp:       time.Unix(0x495fab29, 0), //2009年1月3日12:15:05-0600 cst）
	AddrYou: NetAddress{
Timestamp: time.Time{}, //零值--版本中没有时间戳
		Services:  SFNodeNetwork,
		IP:        net.ParseIP("192.168.0.1"),
		Port:      8333,
	},
	AddrMe: NetAddress{
Timestamp: time.Time{}, //零值--版本中没有时间戳
		Services:  SFNodeNetwork,
		IP:        net.ParseIP("127.0.0.1"),
		Port:      8333,
	},
Nonce:     123123, //0x1E0F3
	UserAgent: "/btcdtest:0.0.1/",
LastBlock: 234234, //0x39 2FA
}

//BaseVersionEncoded是使用协议的BaseVersion的有线编码字节
//版本60002，用于各种测试。
var baseVersionEncoded = []byte{
0x62, 0xea, 0x00, 0x00, //协议版本60002
0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, //小字体
0x29, 0xab, 0x5f, 0x49, 0x00, 0x00, 0x00, 0x00, //64位时间戳
//addryou——版本消息中没有netaddress的时间戳
0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, //小字体
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
0x00, 0x00, 0xff, 0xff, 0xc0, 0xa8, 0x00, 0x01, //IP 192.160.0.1
0x20, 0x8d, //大端8333端口
//addrme—版本消息中没有netaddress的时间戳
0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, //小字体
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
0x00, 0x00, 0xff, 0xff, 0x7f, 0x00, 0x00, 0x01, //IP127.0.0.1
0x20, 0x8d, //大端8333端口
0xf3, 0xe0, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, //临时工
0x10, //用户代理长度变量
	0x2f, 0x62, 0x74, 0x63, 0x64, 0x74, 0x65, 0x73,
0x74, 0x3a, 0x30, 0x2e, 0x30, 0x2e, 0x31, 0x2f, //用户代理
0xfa, 0x92, 0x03, 0x00, //最后一个街区
}

//BaseVersionBip0037在各种测试中用作
//BIP037
var baseVersionBIP0037 = &MsgVersion{
	ProtocolVersion: 70001,
	Services:        SFNodeNetwork,
Timestamp:       time.Unix(0x495fab29, 0), //2009年1月3日12:15:05-0600 cst）
	AddrYou: NetAddress{
Timestamp: time.Time{}, //零值--版本中没有时间戳
		Services:  SFNodeNetwork,
		IP:        net.ParseIP("192.168.0.1"),
		Port:      8333,
	},
	AddrMe: NetAddress{
Timestamp: time.Time{}, //零值--版本中没有时间戳
		Services:  SFNodeNetwork,
		IP:        net.ParseIP("127.0.0.1"),
		Port:      8333,
	},
Nonce:     123123, //0x1E0F3
	UserAgent: "/btcdtest:0.0.1/",
LastBlock: 234234, //0x39 2FA
}

//BaseVersionBip0037 Encoded是BaseVersionBip0037的有线编码字节
//使用协议版本bip0037，用于各种测试。
var baseVersionBIP0037Encoded = []byte{
0x71, 0x11, 0x01, 0x00, //协议版本70001
0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, //小字体
0x29, 0xab, 0x5f, 0x49, 0x00, 0x00, 0x00, 0x00, //64位时间戳
//addryou——版本消息中没有netaddress的时间戳
0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, //小字体
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
0x00, 0x00, 0xff, 0xff, 0xc0, 0xa8, 0x00, 0x01, //IP 192.160.0.1
0x20, 0x8d, //大端8333端口
//addrme—版本消息中没有netaddress的时间戳
0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, //
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
0x00, 0x00, 0xff, 0xff, 0x7f, 0x00, 0x00, 0x01, //IP127.0.0.1
0x20, 0x8d, //大端8333端口
0xf3, 0xe0, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, //临时工
0x10, //用户代理长度变量
	0x2f, 0x62, 0x74, 0x63, 0x64, 0x74, 0x65, 0x73,
0x74, 0x3a, 0x30, 0x2e, 0x30, 0x2e, 0x31, 0x2f, //用户代理
0xfa, 0x92, 0x03, 0x00, //最后一个街区
0x01, //继电器继电器
}
