
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

//testnetaddress测试netaddress api。
func TestNetAddress(t *testing.T) {
	ip := net.ParseIP("127.0.0.1")
	port := 8333

//测试newnetaddress。
	na := NewNetAddress(&net.TCPAddr{IP: ip, Port: port}, 0)

//确保我们得到相同的IP、端口和服务。
	if !na.IP.Equal(ip) {
		t.Errorf("NetNetAddress: wrong ip - got %v, want %v", na.IP, ip)
	}
	if na.Port != uint16(port) {
		t.Errorf("NetNetAddress: wrong port - got %v, want %v", na.Port,
			port)
	}
	if na.Services != 0 {
		t.Errorf("NetNetAddress: wrong services - got %v, want %v",
			na.Services, 0)
	}
	if na.HasService(SFNodeNetwork) {
		t.Errorf("HasService: SFNodeNetwork service is set")
	}

//确保添加完整服务节点标志有效。
	na.AddService(SFNodeNetwork)
	if na.Services != SFNodeNetwork {
		t.Errorf("AddService: wrong services - got %v, want %v",
			na.Services, SFNodeNetwork)
	}
	if !na.HasService(SFNodeNetwork) {
		t.Errorf("HasService: SFNodeNetwork service not set")
	}

//确保最大有效负载是最新协议版本的预期值。
	pver := ProtocolVersion
	wantPayload := uint32(30)
	maxPayload := maxNetAddressPayload(ProtocolVersion)
	if maxPayload != wantPayload {
		t.Errorf("maxNetAddressPayload: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}

//协议版本早于NetAddressTimeVersion，时间戳为
//补充。确保最大有效负载是它的预期值。
	pver = NetAddressTimeVersion - 1
	wantPayload = 26
	maxPayload = maxNetAddressPayload(pver)
	if maxPayload != wantPayload {
		t.Errorf("maxNetAddressPayload: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}
}

//testNetAddressWire测试网络地址线编码和解码
//协议版本和时间戳标志组合。
func TestNetAddressWire(t *testing.T) {
//basenetaddr在各种测试中用作基线netaddress。
	baseNetAddr := NetAddress{
Timestamp: time.Unix(0x495fab29, 0), //2009年1月3日12:15:05-0600 cst
		Services:  SFNodeNetwork,
		IP:        net.ParseIP("127.0.0.1"),
		Port:      8333,
	}

//basenetaddrnots是basenetaddr，时间戳的值为零。
	baseNetAddrNoTS := baseNetAddr
	baseNetAddrNoTS.Timestamp = time.Time{}

//basenetaddr encoded是basenetaddr的线编码字节。
	baseNetAddrEncoded := []byte{
0x29, 0xab, 0x5f, 0x49, //时间戳
0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, //小字体
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
0x00, 0x00, 0xff, 0xff, 0x7f, 0x00, 0x00, 0x01, //IP127.0.0.1
0x20, 0x8d, //大端8333端口
	}

//basenetaddrnots encoded是basenetaddrnots的线编码字节。
	baseNetAddrNoTSEncoded := []byte{
//无时间戳
0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, //小字体
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
0x00, 0x00, 0xff, 0xff, 0x7f, 0x00, 0x00, 0x01, //IP127.0.0.1
0x20, 0x8d, //大端8333端口
	}

	tests := []struct {
in   NetAddress //要编码的网络地址
out  NetAddress //需要解码的网络地址
ts   bool       //包括时间戳？
buf  []byte     //有线编码
pver uint32     //有线编码协议版本
	}{
//没有TS标志的最新协议版本。
		{
			baseNetAddr,
			baseNetAddrNoTS,
			false,
			baseNetAddrNoTSEncoded,
			ProtocolVersion,
		},

//带有TS标志的最新协议版本。
		{
			baseNetAddr,
			baseNetAddr,
			true,
			baseNetAddrEncoded,
			ProtocolVersion,
		},

//协议版本NetAddressTimeVersion，不带TS标志。
		{
			baseNetAddr,
			baseNetAddrNoTS,
			false,
			baseNetAddrNoTSEncoded,
			NetAddressTimeVersion,
		},

//协议版本NetAddressTimeVersion，带有TS标志。
		{
			baseNetAddr,
			baseNetAddr,
			true,
			baseNetAddrEncoded,
			NetAddressTimeVersion,
		},

//协议版本NetAddressTimeVersion-1，不带TS标志。
		{
			baseNetAddr,
			baseNetAddrNoTS,
			false,
			baseNetAddrNoTSEncoded,
			NetAddressTimeVersion - 1,
		},

//协议版本NetAddressTimeVersion-1，带有时间戳。
//即使设置了时间戳标志，它也不应该具有
//时间戳，因为它是以前的协议版本
//补充。
		{
			baseNetAddr,
			baseNetAddrNoTS,
			true,
			baseNetAddrNoTSEncoded,
			NetAddressTimeVersion - 1,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//编码为有线格式。
		var buf bytes.Buffer
		err := writeNetAddress(&buf, test.pver, &test.in, test.ts)
		if err != nil {
			t.Errorf("writeNetAddress #%d error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("writeNetAddress #%d\n got: %s want: %s", i,
				spew.Sdump(buf.Bytes()), spew.Sdump(test.buf))
			continue
		}

//从有线格式解码消息。
		var na NetAddress
		rbuf := bytes.NewReader(test.buf)
		err = readNetAddress(rbuf, test.pver, &na, test.ts)
		if err != nil {
			t.Errorf("readNetAddress #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(na, test.out) {
			t.Errorf("readNetAddress #%d\n got: %s want: %s", i,
				spew.Sdump(na), spew.Sdump(test.out))
			continue
		}
	}
}

//testNetAddressWireErrors对线编码和
//解码网络地址以确认错误路径正常工作。
func TestNetAddressWireErrors(t *testing.T) {
	pver := ProtocolVersion
	pverNAT := NetAddressTimeVersion - 1

//basenetaddr在各种测试中用作基线netaddress。
	baseNetAddr := NetAddress{
Timestamp: time.Unix(0x495fab29, 0), //2009年1月3日12:15:05-0600 cst
		Services:  SFNodeNetwork,
		IP:        net.ParseIP("127.0.0.1"),
		Port:      8333,
	}

	tests := []struct {
in       *NetAddress //编码值
buf      []byte      //有线编码
pver     uint32      //有线编码协议版本
ts       bool        //包含时间戳标志
max      int         //引发错误的固定缓冲区的最大大小
writeErr error       //预期的写入错误
readErr  error       //预期的读取错误
	}{
//最新的协议版本，带有时间戳和意图
//读/写错误。
//强制时间戳出错。
		{&baseNetAddr, []byte{}, pver, true, 0, io.ErrShortWrite, io.EOF},
//强制服务出错。
		{&baseNetAddr, []byte{}, pver, true, 4, io.ErrShortWrite, io.EOF},
//强制IP出错。
		{&baseNetAddr, []byte{}, pver, true, 12, io.ErrShortWrite, io.EOF},
//强制端口出错。
		{&baseNetAddr, []byte{}, pver, true, 28, io.ErrShortWrite, io.EOF},

//最新的协议版本，没有时间戳和有意的
//读/写错误。
//强制服务出错。
		{&baseNetAddr, []byte{}, pver, false, 0, io.ErrShortWrite, io.EOF},
//强制IP出错。
		{&baseNetAddr, []byte{}, pver, false, 8, io.ErrShortWrite, io.EOF},
//强制端口出错。
		{&baseNetAddr, []byte{}, pver, false, 24, io.ErrShortWrite, io.EOF},

//协议版本早于NetAddressTimeVersion，时间戳为
//标志集（由于旧协议，不应具有时间戳
//版本）和有意的读/写错误。
//强制服务出错。
		{&baseNetAddr, []byte{}, pverNAT, true, 0, io.ErrShortWrite, io.EOF},
//强制IP出错。
		{&baseNetAddr, []byte{}, pverNAT, true, 8, io.ErrShortWrite, io.EOF},
//强制端口出错。
		{&baseNetAddr, []byte{}, pverNAT, true, 24, io.ErrShortWrite, io.EOF},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//编码为有线格式。
		w := newFixedWriter(test.max)
		err := writeNetAddress(w, test.pver, test.in, test.ts)
		if err != test.writeErr {
			t.Errorf("writeNetAddress #%d wrong error got: %v, want: %v",
				i, err, test.writeErr)
			continue
		}

//从有线格式解码。
		var na NetAddress
		r := newFixedReader(test.max, test.buf)
		err = readNetAddress(r, test.pver, &na, test.ts)
		if err != test.readErr {
			t.Errorf("readNetAddress #%d wrong error got: %v, want: %v",
				i, err, test.readErr)
			continue
		}
	}
}
