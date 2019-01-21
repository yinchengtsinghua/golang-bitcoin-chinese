
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
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/davecgh/go-spew/spew"
)

//mainnetgenesHash是块链中第一个块的哈希
//主网络（Genesis区块）。
var mainNetGenesisHash = chainhash.Hash([chainhash.HashSize]byte{ //让退伍军人高兴。
	0x6f, 0xe2, 0x8c, 0x0a, 0xb6, 0xf1, 0xb3, 0x72,
	0xc1, 0xa6, 0xa2, 0x46, 0xae, 0x63, 0xf7, 0x4f,
	0x93, 0x1e, 0x83, 0x65, 0xe1, 0x5a, 0x08, 0x9c,
	0x68, 0xd6, 0x19, 0x00, 0x00, 0x00, 0x00, 0x00,
})

//mainnetgenesismerkleroot是Genesis中第一个事务的哈希
//主网络的块。
var mainNetGenesisMerkleRoot = chainhash.Hash([chainhash.HashSize]byte{ //让退伍军人高兴。
	0x3b, 0xa3, 0xed, 0xfd, 0x7a, 0x7b, 0x12, 0xb2,
	0x7a, 0xc7, 0x2c, 0x3e, 0x67, 0x76, 0x8f, 0x61,
	0x7f, 0xc8, 0x1b, 0xc3, 0x88, 0x8a, 0x51, 0x32,
	0x3a, 0x9f, 0xb8, 0xaa, 0x4b, 0x1e, 0x5e, 0x4a,
})

//FakerandReader实现IO.Reader接口，用于强制
//randomunt64函数出错。
type fakeRandReader struct {
	n   int
	err error
}

//read返回假读卡器错误和假读卡器值中的较小值
//以及p的长度。
func (r *fakeRandReader) Read(p []byte) (int, error) {
	n := r.n
	if n > len(p) {
		n = len(p)
	}
	return n, r.err
}

//TestElementWire测试各种元素类型的线编码和解码。这个
//主要测试readelement和writeelement中使用的“fast”路径
//尽可能键入断言以避免反射。
func TestElementWire(t *testing.T) {
	type writeElementReflect int32

	tests := []struct {
in  interface{} //编码值
buf []byte      //有线编码
	}{
		{int32(1), []byte{0x01, 0x00, 0x00, 0x00}},
		{uint32(256), []byte{0x00, 0x01, 0x00, 0x00}},
		{
			int64(65536),
			[]byte{0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			uint64(4294967296),
			[]byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00},
		},
		{
			true,
			[]byte{0x01},
		},
		{
			false,
			[]byte{0x00},
		},
		{
			[4]byte{0x01, 0x02, 0x03, 0x04},
			[]byte{0x01, 0x02, 0x03, 0x04},
		},
		{
			[CommandSize]byte{
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
				0x09, 0x0a, 0x0b, 0x0c,
			},
			[]byte{
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
				0x09, 0x0a, 0x0b, 0x0c,
			},
		},
		{
			[16]byte{
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
				0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
			},
			[]byte{
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
				0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
			},
		},
		{
(*chainhash.Hash)(&[chainhash.HashSize]byte{ //让退伍军人高兴。
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
				0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
				0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
				0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
			}),
			[]byte{
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
				0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
				0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
				0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
			},
		},
		{
			ServiceFlag(SFNodeNetwork),
			[]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			InvType(InvTypeTx),
			[]byte{0x01, 0x00, 0x00, 0x00},
		},
		{
			BitcoinNet(MainNet),
			[]byte{0xf9, 0xbe, 0xb4, 0xd9},
		},
//类型不受“fast”路径支持，需要反射。
		{
			writeElementReflect(1),
			[]byte{0x01, 0x00, 0x00, 0x00},
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//写入有线格式。
		var buf bytes.Buffer
		err := writeElement(&buf, test.in)
		if err != nil {
			t.Errorf("writeElement #%d error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("writeElement #%d\n got: %s want: %s", i,
				spew.Sdump(buf.Bytes()), spew.Sdump(test.buf))
			continue
		}

//读取有线格式。
		rbuf := bytes.NewReader(test.buf)
		val := test.in
		if reflect.ValueOf(test.in).Kind() != reflect.Ptr {
			val = reflect.New(reflect.TypeOf(test.in)).Interface()
		}
		err = readElement(rbuf, val)
		if err != nil {
			t.Errorf("readElement #%d error %v", i, err)
			continue
		}
		ival := val
		if reflect.ValueOf(test.in).Kind() != reflect.Ptr {
			ival = reflect.Indirect(reflect.ValueOf(val)).Interface()
		}
		if !reflect.DeepEqual(ival, test.in) {
			t.Errorf("readElement #%d\n got: %s want: %s", i,
				spew.Sdump(ival), spew.Sdump(test.in))
			continue
		}
	}
}

//TestElementWireErrors对线编码和解码执行负测试
//用于确认错误路径是否正常工作的各种元素类型。
func TestElementWireErrors(t *testing.T) {
	tests := []struct {
in       interface{} //编码值
max      int         //引发错误的固定缓冲区的最大大小
writeErr error       //预期的写入错误
readErr  error       //预期的读取错误
	}{
		{int32(1), 0, io.ErrShortWrite, io.EOF},
		{uint32(256), 0, io.ErrShortWrite, io.EOF},
		{int64(65536), 0, io.ErrShortWrite, io.EOF},
		{true, 0, io.ErrShortWrite, io.EOF},
		{[4]byte{0x01, 0x02, 0x03, 0x04}, 0, io.ErrShortWrite, io.EOF},
		{
			[CommandSize]byte{
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
				0x09, 0x0a, 0x0b, 0x0c,
			},
			0, io.ErrShortWrite, io.EOF,
		},
		{
			[16]byte{
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
				0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
			},
			0, io.ErrShortWrite, io.EOF,
		},
		{
(*chainhash.Hash)(&[chainhash.HashSize]byte{ //让退伍军人高兴。
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
				0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
				0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
				0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
			}),
			0, io.ErrShortWrite, io.EOF,
		},
		{ServiceFlag(SFNodeNetwork), 0, io.ErrShortWrite, io.EOF},
		{InvType(InvTypeTx), 0, io.ErrShortWrite, io.EOF},
		{BitcoinNet(MainNet), 0, io.ErrShortWrite, io.EOF},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//编码为有线格式。
		w := newFixedWriter(test.max)
		err := writeElement(w, test.in)
		if err != test.writeErr {
			t.Errorf("writeElement #%d wrong error got: %v, want: %v",
				i, err, test.writeErr)
			continue
		}

//从有线格式解码。
		r := newFixedReader(test.max, nil)
		val := test.in
		if reflect.ValueOf(test.in).Kind() != reflect.Ptr {
			val = reflect.New(reflect.TypeOf(test.in)).Interface()
		}
		err = readElement(r, val)
		if err != test.readErr {
			t.Errorf("readElement #%d wrong error got: %v, want: %v",
				i, err, test.readErr)
			continue
		}
	}
}

//TestVarintWire测试线对可变长度整数进行编码和解码。
func TestVarIntWire(t *testing.T) {
	pver := ProtocolVersion

	tests := []struct {
in   uint64 //编码值
out  uint64 //预期解码值
buf  []byte //有线编码
pver uint32 //有线编码协议版本
	}{
//最新协议版本。
//单字节
		{0, 0, []byte{0x00}, pver},
//马克斯单字节
		{0xfc, 0xfc, []byte{0xfc}, pver},
//min 2字节
		{0xfd, 0xfd, []byte{0xfd, 0x0fd, 0x00}, pver},
//最大2字节
		{0xffff, 0xffff, []byte{0xfd, 0xff, 0xff}, pver},
//min 4字节
		{0x10000, 0x10000, []byte{0xfe, 0x00, 0x00, 0x01, 0x00}, pver},
//最大4字节
		{0xffffffff, 0xffffffff, []byte{0xfe, 0xff, 0xff, 0xff, 0xff}, pver},
//最小8字节
		{
			0x100000000, 0x100000000,
			[]byte{0xff, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00},
			pver,
		},
//最大8字节
		{
			0xffffffffffffffff, 0xffffffffffffffff,
			[]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			pver,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//编码为有线格式。
		var buf bytes.Buffer
		err := WriteVarInt(&buf, test.pver, test.in)
		if err != nil {
			t.Errorf("WriteVarInt #%d error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("WriteVarInt #%d\n got: %s want: %s", i,
				spew.Sdump(buf.Bytes()), spew.Sdump(test.buf))
			continue
		}

//从有线格式解码。
		rbuf := bytes.NewReader(test.buf)
		val, err := ReadVarInt(rbuf, test.pver)
		if err != nil {
			t.Errorf("ReadVarInt #%d error %v", i, err)
			continue
		}
		if val != test.out {
			t.Errorf("ReadVarInt #%d\n got: %d want: %d", i,
				val, test.out)
			continue
		}
	}
}

//TestVarintWireErrors对线编码和解码执行负测试
//以确认错误路径正确工作。
func TestVarIntWireErrors(t *testing.T) {
	pver := ProtocolVersion

	tests := []struct {
in       uint64 //编码值
buf      []byte //有线编码
pver     uint32 //有线编码协议版本
max      int    //引发错误的固定缓冲区的最大大小
writeErr error  //预期的写入错误
readErr  error  //预期的读取错误
	}{
//强制判别错误。
		{0, []byte{0x00}, pver, 0, io.ErrShortWrite, io.EOF},
//强制2字节读/写出错。
		{0xfd, []byte{0xfd}, pver, 2, io.ErrShortWrite, io.ErrUnexpectedEOF},
//强制4字节读/写出错。
		{0x10000, []byte{0xfe}, pver, 2, io.ErrShortWrite, io.ErrUnexpectedEOF},
//强制8字节读/写出错。
		{0x100000000, []byte{0xff}, pver, 2, io.ErrShortWrite, io.ErrUnexpectedEOF},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//编码为有线格式。
		w := newFixedWriter(test.max)
		err := WriteVarInt(w, test.pver, test.in)
		if err != test.writeErr {
			t.Errorf("WriteVarInt #%d wrong error got: %v, want: %v",
				i, err, test.writeErr)
			continue
		}

//从有线格式解码。
		r := newFixedReader(test.max, test.buf)
		_, err = ReadVarInt(r, test.pver)
		if err != test.readErr {
			t.Errorf("ReadVarInt #%d wrong error got: %v, want: %v",
				i, err, test.readErr)
			continue
		}
	}
}

//testvarintnoncanonical确保不编码的可变长度整数
//通常返回预期的错误。
func TestVarIntNonCanonical(t *testing.T) {
	pver := ProtocolVersion

	tests := []struct {
name string //便于识别的测试名称
in   []byte //解码值
pver uint32 //有线编码协议版本
	}{
		{
			"0 encoded with 3 bytes", []byte{0xfd, 0x00, 0x00},
			pver,
		},
		{
			"max single-byte value encoded with 3 bytes",
			[]byte{0xfd, 0xfc, 0x00}, pver,
		},
		{
			"0 encoded with 5 bytes",
			[]byte{0xfe, 0x00, 0x00, 0x00, 0x00}, pver,
		},
		{
			"max three-byte value encoded with 5 bytes",
			[]byte{0xfe, 0xff, 0xff, 0x00, 0x00}, pver,
		},
		{
			"0 encoded with 9 bytes",
			[]byte{0xff, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			pver,
		},
		{
			"max five-byte value encoded with 9 bytes",
			[]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x00},
			pver,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//从有线格式解码。
		rbuf := bytes.NewReader(test.in)
		val, err := ReadVarInt(rbuf, test.pver)
		if _, ok := err.(*MessageError); !ok {
			t.Errorf("ReadVarInt #%d (%s) unexpected error %v", i,
				test.name, err)
			continue
		}
		if val != 0 {
			t.Errorf("ReadVarInt #%d (%s)\n got: %d want: 0", i,
				test.name, val)
			continue
		}
	}
}

//TestVarintWire测试可变长度整数的序列化大小。
func TestVarIntSerializeSize(t *testing.T) {
	tests := []struct {
val  uint64 //获取序列化大小的值
size int    //应为序列化大小
	}{
//单字节
		{0, 1},
//马克斯单字节
		{0xfc, 1},
//min 2字节
		{0xfd, 3},
//最大2字节
		{0xffff, 3},
//min 4字节
		{0x10000, 5},
//最大4字节
		{0xffffffff, 5},
//最小8字节
		{0x100000000, 9},
//最大8字节
		{0xffffffffffffffff, 9},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		serializedSize := VarIntSerializeSize(test.val)
		if serializedSize != test.size {
			t.Errorf("VarIntSerializeSize #%d got: %d, want: %d", i,
				serializedSize, test.size)
			continue
		}
	}
}

//testvarStringWire测试可变长度字符串的线编码和解码。
func TestVarStringWire(t *testing.T) {
	pver := ProtocolVersion

//str256是一个需要2字节变量进行编码的字符串。
	str256 := strings.Repeat("test", 64)

	tests := []struct {
in   string //要编码的字符串
out  string //解码值字符串
buf  []byte //有线编码
pver uint32 //有线编码协议版本
	}{
//最新协议版本。
//空字符串
		{"", "", []byte{0x00}, pver},
//单字节变量+字符串
		{"Test", "Test", append([]byte{0x04}, []byte("Test")...), pver},
//2字节变量+字符串
		{str256, str256, append([]byte{0xfd, 0x00, 0x01}, []byte(str256)...), pver},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//编码为有线格式。
		var buf bytes.Buffer
		err := WriteVarString(&buf, test.pver, test.in)
		if err != nil {
			t.Errorf("WriteVarString #%d error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("WriteVarString #%d\n got: %s want: %s", i,
				spew.Sdump(buf.Bytes()), spew.Sdump(test.buf))
			continue
		}

//从有线格式解码。
		rbuf := bytes.NewReader(test.buf)
		val, err := ReadVarString(rbuf, test.pver)
		if err != nil {
			t.Errorf("ReadVarString #%d error %v", i, err)
			continue
		}
		if val != test.out {
			t.Errorf("ReadVarString #%d\n got: %s want: %s", i,
				val, test.out)
			continue
		}
	}
}

//testvarStringWireErrors对线编码和
//解码可变长度字符串以确认错误路径正常工作。
func TestVarStringWireErrors(t *testing.T) {
	pver := ProtocolVersion

//str256是一个需要2字节变量进行编码的字符串。
	str256 := strings.Repeat("test", 64)

	tests := []struct {
in       string //编码值
buf      []byte //有线编码
pver     uint32 //有线编码协议版本
max      int    //引发错误的固定缓冲区的最大大小
writeErr error  //预期的写入错误
readErr  error  //预期的读取错误
	}{
//具有故意读/写错误的最新协议版本。
//强制空字符串出错。
		{"", []byte{0x00}, pver, 0, io.ErrShortWrite, io.EOF},
//强制单字节变量+字符串出错。
		{"Test", []byte{0x04}, pver, 2, io.ErrShortWrite, io.ErrUnexpectedEOF},
//强制2字节变量+字符串出错。
		{str256, []byte{0xfd}, pver, 2, io.ErrShortWrite, io.ErrUnexpectedEOF},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//编码为有线格式。
		w := newFixedWriter(test.max)
		err := WriteVarString(w, test.pver, test.in)
		if err != test.writeErr {
			t.Errorf("WriteVarString #%d wrong error got: %v, want: %v",
				i, err, test.writeErr)
			continue
		}

//从有线格式解码。
		r := newFixedReader(test.max, test.buf)
		_, err = ReadVarString(r, test.pver)
		if err != test.readErr {
			t.Errorf("ReadVarString #%d wrong error got: %v, want: %v",
				i, err, test.readErr)
			continue
		}
	}
}

//testvarStringOverflowErrors执行测试以确保反序列化变量
//故意为字符串使用大值而设计的长度字符串
//长度处理得当。否则，它可能被用作
//攻击向量。
func TestVarStringOverflowErrors(t *testing.T) {
	pver := ProtocolVersion

	tests := []struct {
buf  []byte //有线编码
pver uint32 //有线编码协议版本
err  error  //期望误差
	}{
		{[]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			pver, &MessageError{}},
		{[]byte{0xff, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			pver, &MessageError{}},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//从有线格式解码。
		rbuf := bytes.NewReader(test.buf)
		_, err := ReadVarString(rbuf, test.pver)
		if reflect.TypeOf(err) != reflect.TypeOf(test.err) {
			t.Errorf("ReadVarString #%d wrong error got: %v, "+
				"want: %v", i, err, reflect.TypeOf(test.err))
			continue
		}
	}

}

//testvarbyteswire测试电线编码和解码可变长度字节数组。
func TestVarBytesWire(t *testing.T) {
	pver := ProtocolVersion

//字节256是一个需要2字节变量进行编码的字节数组。
	bytes256 := bytes.Repeat([]byte{0x01}, 256)

	tests := []struct {
in   []byte //要写入的字节数组
buf  []byte //有线编码
pver uint32 //有线编码协议版本
	}{
//最新协议版本。
//空字节数组
		{[]byte{}, []byte{0x00}, pver},
//单字节变量+字节数组
		{[]byte{0x01}, []byte{0x01, 0x01}, pver},
//2字节变量+字节数组
		{bytes256, append([]byte{0xfd, 0x00, 0x01}, bytes256...), pver},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//编码为有线格式。
		var buf bytes.Buffer
		err := WriteVarBytes(&buf, test.pver, test.in)
		if err != nil {
			t.Errorf("WriteVarBytes #%d error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("WriteVarBytes #%d\n got: %s want: %s", i,
				spew.Sdump(buf.Bytes()), spew.Sdump(test.buf))
			continue
		}

//从有线格式解码。
		rbuf := bytes.NewReader(test.buf)
		val, err := ReadVarBytes(rbuf, test.pver, MaxMessagePayload,
			"test payload")
		if err != nil {
			t.Errorf("ReadVarBytes #%d error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("ReadVarBytes #%d\n got: %s want: %s", i,
				val, test.buf)
			continue
		}
	}
}

//TestVarBytesWireErrors对线编码和
//解码可变长度字节数组以确认错误路径正常工作。
func TestVarBytesWireErrors(t *testing.T) {
	pver := ProtocolVersion

//字节256是一个需要2字节变量进行编码的字节数组。
	bytes256 := bytes.Repeat([]byte{0x01}, 256)

	tests := []struct {
in       []byte //要写入的字节数组
buf      []byte //有线编码
pver     uint32 //有线编码协议版本
max      int    //引发错误的固定缓冲区的最大大小
writeErr error  //预期的写入错误
readErr  error  //预期的读取错误
	}{
//具有故意读/写错误的最新协议版本。
//强制空字节数组出错。
		{[]byte{}, []byte{0x00}, pver, 0, io.ErrShortWrite, io.EOF},
//强制单字节变量+字节数组出错。
		{[]byte{0x01, 0x02, 0x03}, []byte{0x04}, pver, 2, io.ErrShortWrite, io.ErrUnexpectedEOF},
//强制2字节变量+字节数组出错。
		{bytes256, []byte{0xfd}, pver, 2, io.ErrShortWrite, io.ErrUnexpectedEOF},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//编码为有线格式。
		w := newFixedWriter(test.max)
		err := WriteVarBytes(w, test.pver, test.in)
		if err != test.writeErr {
			t.Errorf("WriteVarBytes #%d wrong error got: %v, want: %v",
				i, err, test.writeErr)
			continue
		}

//从有线格式解码。
		r := newFixedReader(test.max, test.buf)
		_, err = ReadVarBytes(r, test.pver, MaxMessagePayload,
			"test payload")
		if err != test.readErr {
			t.Errorf("ReadVarBytes #%d wrong error got: %v, want: %v",
				i, err, test.readErr)
			continue
		}
	}
}

//testvarbytesoverflowerrrors执行测试以确保对变量进行反序列化
//故意为数组使用大值而设计的长度字节数组
//长度处理得当。否则，它可能被用作
//攻击向量。
func TestVarBytesOverflowErrors(t *testing.T) {
	pver := ProtocolVersion

	tests := []struct {
buf  []byte //有线编码
pver uint32 //有线编码协议版本
err  error  //期望误差
	}{
		{[]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			pver, &MessageError{}},
		{[]byte{0xff, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			pver, &MessageError{}},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//从有线格式解码。
		rbuf := bytes.NewReader(test.buf)
		_, err := ReadVarBytes(rbuf, test.pver, MaxMessagePayload,
			"test payload")
		if reflect.TypeOf(err) != reflect.TypeOf(test.err) {
			t.Errorf("ReadVarBytes #%d wrong error got: %v, "+
				"want: %v", i, err, reflect.TypeOf(test.err))
			continue
		}
	}

}

//testradomunit64在上运行随机数生成器的随机性
//系统通过确保生成的数字的概率。如果RNG
//是均匀分布的，作为一个适当的加密RNG应该是，真的
//对于64位数字，在2^8次尝试中只能是1个小于2^56的数字。然而，
//使用更高的5个数字以确保测试不会失败，除非
//RNG真是可怕。
func TestRandomUint64(t *testing.T) {
tries := 1 << 8              //2 ^ 8
watermark := uint64(1 << 56) //2 ^ 56
	maxHits := 5
	badRNG := "The random number generator on this system is clearly " +
		"terrible since we got %d values less than %d in %d runs " +
		"when only %d was expected"

	numHits := 0
	for i := 0; i < tries; i++ {
		nonce, err := RandomUint64()
		if err != nil {
			t.Errorf("RandomUint64 iteration %d failed - err %v",
				i, err)
			return
		}
		if nonce < watermark {
			numHits++
		}
		if numHits > maxHits {
			str := fmt.Sprintf(badRNG, numHits, watermark, tries, maxHits)
			t.Errorf("Random Uint64 iteration %d failed - %v %v", i,
				str, numHits)
			return
		}
	}
}

//testradomunt64错误使用假读卡器强制执行错误路径
//并相应地检查结果。
func TestRandomUint64Errors(t *testing.T) {
//测试短读。
	fr := &fakeRandReader{n: 2, err: io.EOF}
	nonce, err := randomUint64(fr)
	if err != io.ErrUnexpectedEOF {
		t.Errorf("Error not expected value of %v [%v]",
			io.ErrUnexpectedEOF, err)
	}
	if nonce != 0 {
		t.Errorf("Nonce is not 0 [%v]", nonce)
	}
}
