
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

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/davecgh/go-spew/spew"
)

//TestInvectStringer测试库存向量类型的字符串化输出。
func TestInvTypeStringer(t *testing.T) {
	tests := []struct {
		in   InvType
		want string
	}{
		{InvTypeError, "ERROR"},
		{InvTypeTx, "MSG_TX"},
		{InvTypeBlock, "MSG_BLOCK"},
		{0xffffffff, "Unknown InvType (4294967295)"},
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

//testinvvect测试invvect API。
func TestInvVect(t *testing.T) {
	ivType := InvTypeBlock
	hash := chainhash.Hash{}

//确保我们得到相同的有效载荷和签名。
	iv := NewInvVect(ivType, &hash)
	if iv.Type != ivType {
		t.Errorf("NewInvVect: wrong type - got %v, want %v",
			iv.Type, ivType)
	}
	if !iv.Hash.IsEqual(&hash) {
		t.Errorf("NewInvVect: wrong hash - got %v, want %v",
			spew.Sdump(iv.Hash), spew.Sdump(hash))
	}

}

//testinvvectire测试invvect线的各种编码和解码
//协议版本和支持的库存向量类型。
func TestInvVectWire(t *testing.T) {
//块203707哈希。
	hashStr := "3264bc2ac36a60840790ba1d475d01367e7c723da941069e9dc"
	baseHash, err := chainhash.NewHashFromStr(hashStr)
	if err != nil {
		t.Errorf("NewHashFromStr: %v", err)
	}

//errinvvect是一个有错误的库存向量。
	errInvVect := InvVect{
		Type: InvTypeError,
		Hash: chainhash.Hash{},
	}

//errinvvect encoded是errinvvect的有线编码字节。
	errInvVectEncoded := []byte{
0x00, 0x00, 0x00, 0x00, //输入错误
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, //无散列
	}

//txinvvect是表示交易的库存向量。
	txInvVect := InvVect{
		Type: InvTypeTx,
		Hash: *baseHash,
	}

//txinvvect encoded是txinvvect的有线编码字节。
	txInvVectEncoded := []byte{
0x01, 0x00, 0x00, 0x00, //输入字体
		0xdc, 0xe9, 0x69, 0x10, 0x94, 0xda, 0x23, 0xc7,
		0xe7, 0x67, 0x13, 0xd0, 0x75, 0xd4, 0xa1, 0x0b,
		0x79, 0x40, 0x08, 0xa6, 0x36, 0xac, 0xc2, 0x4b,
0x26, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, //块203707哈希
	}

//blockinvct是表示块的库存向量。
	blockInvVect := InvVect{
		Type: InvTypeBlock,
		Hash: *baseHash,
	}

//blockinvvect encoded是blockinvvect的线编码字节。
	blockInvVectEncoded := []byte{
0x02, 0x00, 0x00, 0x00, //输入块
		0xdc, 0xe9, 0x69, 0x10, 0x94, 0xda, 0x23, 0xc7,
		0xe7, 0x67, 0x13, 0xd0, 0x75, 0xd4, 0xa1, 0x0b,
		0x79, 0x40, 0x08, 0xa6, 0x36, 0xac, 0xc2, 0x4b,
0x26, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, //块203707哈希
	}

	tests := []struct {
in   InvVect //要编码的网络地址
out  InvVect //需要解码的网络地址
buf  []byte  //有线编码
pver uint32  //有线编码协议版本
	}{
//最新协议版本错误清单向量。
		{
			errInvVect,
			errInvVect,
			errInvVectEncoded,
			ProtocolVersion,
		},

//最新协议版本tx库存向量。
		{
			txInvVect,
			txInvVect,
			txInvVectEncoded,
			ProtocolVersion,
		},

//最新协议版本的块清单向量。
		{
			blockInvVect,
			blockInvVect,
			blockInvVectEncoded,
			ProtocolVersion,
		},

//协议版本bip0035版本错误库存向量。
		{
			errInvVect,
			errInvVect,
			errInvVectEncoded,
			BIP0035Version,
		},

//协议版本bip0035版本tx库存向量。
		{
			txInvVect,
			txInvVect,
			txInvVectEncoded,
			BIP0035Version,
		},

//协议版本bip0035版本块库存向量。
		{
			blockInvVect,
			blockInvVect,
			blockInvVectEncoded,
			BIP0035Version,
		},

//协议版本bip0031版本错误库存向量。
		{
			errInvVect,
			errInvVect,
			errInvVectEncoded,
			BIP0031Version,
		},

//协议版本bip0031版本tx库存向量。
		{
			txInvVect,
			txInvVect,
			txInvVectEncoded,
			BIP0031Version,
		},

//协议版本bip0031块库存向量。
		{
			blockInvVect,
			blockInvVect,
			blockInvVectEncoded,
			BIP0031Version,
		},

//协议版本NetAddressTimeVersion错误清单向量。
		{
			errInvVect,
			errInvVect,
			errInvVectEncoded,
			NetAddressTimeVersion,
		},

//协议版本netaddresstimeversion tx库存向量。
		{
			txInvVect,
			txInvVect,
			txInvVectEncoded,
			NetAddressTimeVersion,
		},

//协议版本NetAddressTimeVersion块清单向量。
		{
			blockInvVect,
			blockInvVect,
			blockInvVectEncoded,
			NetAddressTimeVersion,
		},

//协议版本multipleaddressversion错误清单向量。
		{
			errInvVect,
			errInvVect,
			errInvVectEncoded,
			MultipleAddressVersion,
		},

//协议版本多线程版本Tx库存向量。
		{
			txInvVect,
			txInvVect,
			txInvVectEncoded,
			MultipleAddressVersion,
		},

//协议版本multipleaddressversion块库存向量。
		{
			blockInvVect,
			blockInvVect,
			blockInvVectEncoded,
			MultipleAddressVersion,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//编码为有线格式。
		var buf bytes.Buffer
		err := writeInvVect(&buf, test.pver, &test.in)
		if err != nil {
			t.Errorf("writeInvVect #%d error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("writeInvVect #%d\n got: %s want: %s", i,
				spew.Sdump(buf.Bytes()), spew.Sdump(test.buf))
			continue
		}

//从有线格式解码消息。
		var iv InvVect
		rbuf := bytes.NewReader(test.buf)
		err = readInvVect(rbuf, test.pver, &iv)
		if err != nil {
			t.Errorf("readInvVect #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(iv, test.out) {
			t.Errorf("readInvVect #%d\n got: %s want: %s", i,
				spew.Sdump(iv), spew.Sdump(test.out))
			continue
		}
	}
}
