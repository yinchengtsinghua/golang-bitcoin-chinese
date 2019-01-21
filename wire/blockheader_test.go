
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
	"time"

	"github.com/davecgh/go-spew/spew"
)

//TestBlockHeader测试BlockHeader API。
func TestBlockHeader(t *testing.T) {
	nonce64, err := RandomUint64()
	if err != nil {
		t.Errorf("RandomUint64: Error generating nonce: %v", err)
	}
	nonce := uint32(nonce64)

	hash := mainNetGenesisHash
	merkleHash := mainNetGenesisMerkleRoot
	bits := uint32(0x1d00ffff)
	bh := NewBlockHeader(1, &hash, &merkleHash, bits, nonce)

//确保我们得到相同的数据。
	if !bh.PrevBlock.IsEqual(&hash) {
		t.Errorf("NewBlockHeader: wrong prev hash - got %v, want %v",
			spew.Sprint(bh.PrevBlock), spew.Sprint(hash))
	}
	if !bh.MerkleRoot.IsEqual(&merkleHash) {
		t.Errorf("NewBlockHeader: wrong merkle root - got %v, want %v",
			spew.Sprint(bh.MerkleRoot), spew.Sprint(merkleHash))
	}
	if bh.Bits != bits {
		t.Errorf("NewBlockHeader: wrong bits - got %v, want %v",
			bh.Bits, bits)
	}
	if bh.Nonce != nonce {
		t.Errorf("NewBlockHeader: wrong nonce - got %v, want %v",
			bh.Nonce, nonce)
	}
}

//测试BlockHeaderWire测试BlockHeader线对各种
//协议版本。
func TestBlockHeaderWire(t *testing.T) {
nonce := uint32(123123) //0x1E0F3
	pver := uint32(70001)

//baseblockhdr在各种测试中用作基线blockheader。
	bits := uint32(0x1d00ffff)
	baseBlockHdr := &BlockHeader{
		Version:    1,
		PrevBlock:  mainNetGenesisHash,
		MerkleRoot: mainNetGenesisMerkleRoot,
Timestamp:  time.Unix(0x495fab29, 0), //2009年1月3日12:15:05-0600 cst
		Bits:       bits,
		Nonce:      nonce,
	}

//baseblockhdr encoded是baseblockhdr的有线编码字节。
	baseBlockHdrEncoded := []byte{
0x01, 0x00, 0x00, 0x00, //版本1
		0x6f, 0xe2, 0x8c, 0x0a, 0xb6, 0xf1, 0xb3, 0x72,
		0xc1, 0xa6, 0xa2, 0x46, 0xae, 0x63, 0xf7, 0x4f,
		0x93, 0x1e, 0x83, 0x65, 0xe1, 0x5a, 0x08, 0x9c,
0x68, 0xd6, 0x19, 0x00, 0x00, 0x00, 0x00, 0x00, //预防阻滞
		0x3b, 0xa3, 0xed, 0xfd, 0x7a, 0x7b, 0x12, 0xb2,
		0x7a, 0xc7, 0x2c, 0x3e, 0x67, 0x76, 0x8f, 0x61,
		0x7f, 0xc8, 0x1b, 0xc3, 0x88, 0x8a, 0x51, 0x32,
0x3a, 0x9f, 0xb8, 0xaa, 0x4b, 0x1e, 0x5e, 0x4a, //木兰科植物
0x29, 0xab, 0x5f, 0x49, //时间戳
0xff, 0xff, 0x00, 0x1d, //位
0xf3, 0xe0, 0x01, 0x00, //临时工
	}

	tests := []struct {
in   *BlockHeader    //编码数据
out  *BlockHeader    //预期解码数据
buf  []byte          //有线编码
pver uint32          //有线编码协议版本
enc  MessageEncoding //要使用的消息编码变量
	}{
//最新协议版本。
		{
			baseBlockHdr,
			baseBlockHdr,
			baseBlockHdrEncoded,
			ProtocolVersion,
			BaseEncoding,
		},

//协议版本BIP0035版本。
		{
			baseBlockHdr,
			baseBlockHdr,
			baseBlockHdrEncoded,
			BIP0035Version,
			BaseEncoding,
		},

//协议版本Bip0031版本。
		{
			baseBlockHdr,
			baseBlockHdr,
			baseBlockHdrEncoded,
			BIP0031Version,
			BaseEncoding,
		},

//协议版本NetAddressTimeVersion。
		{
			baseBlockHdr,
			baseBlockHdr,
			baseBlockHdrEncoded,
			NetAddressTimeVersion,
			BaseEncoding,
		},

//协议版本multipleaddressversion。
		{
			baseBlockHdr,
			baseBlockHdr,
			baseBlockHdrEncoded,
			MultipleAddressVersion,
			BaseEncoding,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//编码为有线格式。
		var buf bytes.Buffer
		err := writeBlockHeader(&buf, test.pver, test.in)
		if err != nil {
			t.Errorf("writeBlockHeader #%d error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("writeBlockHeader #%d\n got: %s want: %s", i,
				spew.Sdump(buf.Bytes()), spew.Sdump(test.buf))
			continue
		}

		buf.Reset()
		err = test.in.BtcEncode(&buf, pver, 0)
		if err != nil {
			t.Errorf("BtcEncode #%d error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("BtcEncode #%d\n got: %s want: %s", i,
				spew.Sdump(buf.Bytes()), spew.Sdump(test.buf))
			continue
		}

//从Wire格式解码块头。
		var bh BlockHeader
		rbuf := bytes.NewReader(test.buf)
		err = readBlockHeader(rbuf, test.pver, &bh)
		if err != nil {
			t.Errorf("readBlockHeader #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(&bh, test.out) {
			t.Errorf("readBlockHeader #%d\n got: %s want: %s", i,
				spew.Sdump(&bh), spew.Sdump(test.out))
			continue
		}

		rbuf = bytes.NewReader(test.buf)
		err = bh.BtcDecode(rbuf, pver, test.enc)
		if err != nil {
			t.Errorf("BtcDecode #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(&bh, test.out) {
			t.Errorf("BtcDecode #%d\n got: %s want: %s", i,
				spew.Sdump(&bh), spew.Sdump(test.out))
			continue
		}
	}
}

//TestBlockHeader序列化测试BlockHeader序列化和反序列化。
func TestBlockHeaderSerialize(t *testing.T) {
nonce := uint32(123123) //0x1E0F3

//baseblockhdr在各种测试中用作基线blockheader。
	bits := uint32(0x1d00ffff)
	baseBlockHdr := &BlockHeader{
		Version:    1,
		PrevBlock:  mainNetGenesisHash,
		MerkleRoot: mainNetGenesisMerkleRoot,
Timestamp:  time.Unix(0x495fab29, 0), //2009年1月3日12:15:05-0600 cst
		Bits:       bits,
		Nonce:      nonce,
	}

//baseblockhdr encoded是baseblockhdr的有线编码字节。
	baseBlockHdrEncoded := []byte{
0x01, 0x00, 0x00, 0x00, //版本1
		0x6f, 0xe2, 0x8c, 0x0a, 0xb6, 0xf1, 0xb3, 0x72,
		0xc1, 0xa6, 0xa2, 0x46, 0xae, 0x63, 0xf7, 0x4f,
		0x93, 0x1e, 0x83, 0x65, 0xe1, 0x5a, 0x08, 0x9c,
0x68, 0xd6, 0x19, 0x00, 0x00, 0x00, 0x00, 0x00, //预防阻滞
		0x3b, 0xa3, 0xed, 0xfd, 0x7a, 0x7b, 0x12, 0xb2,
		0x7a, 0xc7, 0x2c, 0x3e, 0x67, 0x76, 0x8f, 0x61,
		0x7f, 0xc8, 0x1b, 0xc3, 0x88, 0x8a, 0x51, 0x32,
0x3a, 0x9f, 0xb8, 0xaa, 0x4b, 0x1e, 0x5e, 0x4a, //木兰科植物
0x29, 0xab, 0x5f, 0x49, //时间戳
0xff, 0xff, 0x00, 0x1d, //位
0xf3, 0xe0, 0x01, 0x00, //临时工
	}

	tests := []struct {
in  *BlockHeader //编码数据
out *BlockHeader //预期解码数据
buf []byte       //序列化数据
	}{
		{
			baseBlockHdr,
			baseBlockHdr,
			baseBlockHdrEncoded,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//序列化块头。
		var buf bytes.Buffer
		err := test.in.Serialize(&buf)
		if err != nil {
			t.Errorf("Serialize #%d error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("Serialize #%d\n got: %s want: %s", i,
				spew.Sdump(buf.Bytes()), spew.Sdump(test.buf))
			continue
		}

//反序列化块头。
		var bh BlockHeader
		rbuf := bytes.NewReader(test.buf)
		err = bh.Deserialize(rbuf)
		if err != nil {
			t.Errorf("Deserialize #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(&bh, test.out) {
			t.Errorf("Deserialize #%d\n got: %s want: %s", i,
				spew.Sdump(&bh), spew.Sdump(test.out))
			continue
		}
	}
}
