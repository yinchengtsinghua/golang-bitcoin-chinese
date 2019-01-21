
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package blockchain

import (
	"bytes"
	"encoding/hex"
	"testing"
)

//hextobytes将传递的十六进制字符串转换为字节，如果有，将死机
//是一个错误。这仅为硬编码常量提供，因此
//可以检测到源代码。它只能（而且必须）用
//硬编码值。
func hexToBytes(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic("invalid hex in source file: " + s)
	}
	return b
}

//testvlq确保可变长度的数量序列化、反序列化，
//尺寸计算工作如预期。
func TestVLQ(t *testing.T) {
	t.Parallel()

	tests := []struct {
		val        uint64
		serialized []byte
	}{
		{0, hexToBytes("00")},
		{1, hexToBytes("01")},
		{127, hexToBytes("7f")},
		{128, hexToBytes("8000")},
		{129, hexToBytes("8001")},
		{255, hexToBytes("807f")},
		{256, hexToBytes("8100")},
		{16383, hexToBytes("fe7f")},
		{16384, hexToBytes("ff00")},
{16511, hexToBytes("ff7f")}, //最大2字节值
		{16512, hexToBytes("808000")},
		{16513, hexToBytes("808001")},
		{16639, hexToBytes("80807f")},
		{32895, hexToBytes("80ff7f")},
{2113663, hexToBytes("ffff7f")}, //最大3字节值
		{2113664, hexToBytes("80808000")},
{270549119, hexToBytes("ffffff7f")}, //最大4字节值
		{270549120, hexToBytes("8080808000")},
		{2147483647, hexToBytes("86fefefe7f")},
		{2147483648, hexToBytes("86fefeff00")},
{4294967295, hexToBytes("8efefefe7f")}, //最大uint32，5字节
//最大uint64，10字节
		{18446744073709551615, hexToBytes("80fefefefefefefefe7f")},
	}

	for _, test := range tests {
//确保函数计算序列化大小
//实际上，对值进行序列化是正确计算的。
		gotSize := serializeSizeVLQ(test.val)
		if gotSize != len(test.serialized) {
			t.Errorf("serializeSizeVLQ: did not get expected size "+
				"for %d - got %d, want %d", test.val, gotSize,
				len(test.serialized))
			continue
		}

//确保值序列化为预期的字节。
		gotBytes := make([]byte, gotSize)
		gotBytesWritten := putVLQ(gotBytes, test.val)
		if !bytes.Equal(gotBytes, test.serialized) {
			t.Errorf("putVLQUnchecked: did not get expected bytes "+
				"for %d - got %x, want %x", test.val, gotBytes,
				test.serialized)
			continue
		}
		if gotBytesWritten != len(test.serialized) {
			t.Errorf("putVLQUnchecked: did not get expected number "+
				"of bytes written for %d - got %d, want %d",
				test.val, gotBytesWritten, len(test.serialized))
			continue
		}

//确保序列化的字节反序列化为预期的
//价值。
		gotVal, gotBytesRead := deserializeVLQ(test.serialized)
		if gotVal != test.val {
			t.Errorf("deserializeVLQ: did not get expected value "+
				"for %x - got %d, want %d", test.serialized,
				gotVal, test.val)
			continue
		}
		if gotBytesRead != len(test.serialized) {
			t.Errorf("deserializeVLQ: did not get expected number "+
				"of bytes read for %d - got %d, want %d",
				test.serialized, gotBytesRead,
				len(test.serialized))
			continue
		}
	}
}

//testscriptcompression确保特定于域的脚本压缩和
//减压工作正常。
func TestScriptCompression(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		uncompressed []byte
		compressed   []byte
	}{
		{
			name:         "nil",
			uncompressed: nil,
			compressed:   hexToBytes("06"),
		},
		{
			name:         "pay-to-pubkey-hash 1",
			uncompressed: hexToBytes("76a9141018853670f9f3b0582c5b9ee8ce93764ac32b9388ac"),
			compressed:   hexToBytes("001018853670f9f3b0582c5b9ee8ce93764ac32b93"),
		},
		{
			name:         "pay-to-pubkey-hash 2",
			uncompressed: hexToBytes("76a914e34cce70c86373273efcc54ce7d2a491bb4a0e8488ac"),
			compressed:   hexToBytes("00e34cce70c86373273efcc54ce7d2a491bb4a0e84"),
		},
		{
			name:         "pay-to-script-hash 1",
			uncompressed: hexToBytes("a914da1745e9b549bd0bfa1a569971c77eba30cd5a4b87"),
			compressed:   hexToBytes("01da1745e9b549bd0bfa1a569971c77eba30cd5a4b"),
		},
		{
			name:         "pay-to-script-hash 2",
			uncompressed: hexToBytes("a914f815b036d9bbbce5e9f2a00abd1bf3dc91e9551087"),
			compressed:   hexToBytes("01f815b036d9bbbce5e9f2a00abd1bf3dc91e95510"),
		},
		{
			name:         "pay-to-pubkey compressed 0x02",
			uncompressed: hexToBytes("2102192d74d0cb94344c9569c2e77901573d8d7903c3ebec3a957724895dca52c6b4ac"),
			compressed:   hexToBytes("02192d74d0cb94344c9569c2e77901573d8d7903c3ebec3a957724895dca52c6b4"),
		},
		{
			name:         "pay-to-pubkey compressed 0x03",
			uncompressed: hexToBytes("2103b0bd634234abbb1ba1e986e884185c61cf43e001f9137f23c2c409273eb16e65ac"),
			compressed:   hexToBytes("03b0bd634234abbb1ba1e986e884185c61cf43e001f9137f23c2c409273eb16e65"),
		},
		{
			name:         "pay-to-pubkey uncompressed 0x04 even",
			uncompressed: hexToBytes("4104192d74d0cb94344c9569c2e77901573d8d7903c3ebec3a957724895dca52c6b40d45264838c0bd96852662ce6a847b197376830160c6d2eb5e6a4c44d33f453eac"),
			compressed:   hexToBytes("04192d74d0cb94344c9569c2e77901573d8d7903c3ebec3a957724895dca52c6b4"),
		},
		{
			name:         "pay-to-pubkey uncompressed 0x04 odd",
			uncompressed: hexToBytes("410411db93e1dcdb8a016b49840f8c53bc1eb68a382e97b1482ecad7b148a6909a5cb2e0eaddfb84ccf9744464f82e160bfa9b8b64f9d4c03f999b8643f656b412a3ac"),
			compressed:   hexToBytes("0511db93e1dcdb8a016b49840f8c53bc1eb68a382e97b1482ecad7b148a6909a5c"),
		},
		{
			name:         "pay-to-pubkey invalid pubkey",
			uncompressed: hexToBytes("3302aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaac"),
			compressed:   hexToBytes("293302aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaac"),
		},
		{
			name:         "null data",
			uncompressed: hexToBytes("6a200102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"),
			compressed:   hexToBytes("286a200102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"),
		},
		{
			name:         "requires 2 size bytes - data push 200 bytes",
			uncompressed: append(hexToBytes("4cc8"), bytes.Repeat([]byte{0x00}, 200)...),
//[0x80，0x50]=208，作为可变长度数量
//[0x4c，0xc8]=op_pushdata1 200
			compressed: append(hexToBytes("80504cc8"), bytes.Repeat([]byte{0x00}, 200)...),
		},
	}

	for _, test := range tests {
//确保函数计算序列化大小
//实际上，对值进行序列化是正确计算的。
		gotSize := compressedScriptSize(test.uncompressed)
		if gotSize != len(test.compressed) {
			t.Errorf("compressedScriptSize (%s): did not get "+
				"expected size - got %d, want %d", test.name,
				gotSize, len(test.compressed))
			continue
		}

//确保脚本压缩到预期的字节。
		gotCompressed := make([]byte, gotSize)
		gotBytesWritten := putCompressedScript(gotCompressed,
			test.uncompressed)
		if !bytes.Equal(gotCompressed, test.compressed) {
			t.Errorf("putCompressedScript (%s): did not get "+
				"expected bytes - got %x, want %x", test.name,
				gotCompressed, test.compressed)
			continue
		}
		if gotBytesWritten != len(test.compressed) {
			t.Errorf("putCompressedScript (%s): did not get "+
				"expected number of bytes written - got %d, "+
				"want %d", test.name, gotBytesWritten,
				len(test.compressed))
			continue
		}

//确保已正确解码压缩的脚本大小
//压缩后的脚本。
		gotDecodedSize := decodeCompressedScriptSize(test.compressed)
		if gotDecodedSize != len(test.compressed) {
			t.Errorf("decodeCompressedScriptSize (%s): did not get "+
				"expected size - got %d, want %d", test.name,
				gotDecodedSize, len(test.compressed))
			continue
		}

//确保脚本解压缩到预期的字节。
		gotDecompressed := decompressScript(test.compressed)
		if !bytes.Equal(gotDecompressed, test.uncompressed) {
			t.Errorf("decompressScript (%s): did not get expected "+
				"bytes - got %x, want %x", test.name,
				gotDecompressed, test.uncompressed)
			continue
		}
	}
}

//testscriptcompressionErrors确保调用与
//使用不正确数据的脚本压缩将返回预期的结果。
func TestScriptCompressionErrors(t *testing.T) {
	t.Parallel()

//nil脚本必须导致解码后的大小为0。
	if gotSize := decodeCompressedScriptSize(nil); gotSize != 0 {
		t.Fatalf("decodeCompressedScriptSize with nil script did not "+
			"return 0 - got %d", gotSize)
	}

//nil脚本必须导致nil解压缩脚本。
	if gotScript := decompressScript(nil); gotScript != nil {
		t.Fatalf("decompressScript with nil script did not return nil "+
			"decompressed script - got %x", gotScript)
	}

//用于“支付到发布”键（未压缩）的压缩脚本，结果为
//在无效的pubkey中，必须生成nil解压脚本。
	compressedScript := hexToBytes("04012d74d0cb94344c9569c2e77901573d8d" +
		"7903c3ebec3a957724895dca52c6b4")
	if gotScript := decompressScript(compressedScript); gotScript != nil {
		t.Fatalf("decompressScript with compressed pay-to-"+
			"uncompressed-pubkey that is invalid did not return "+
			"nil decompressed script - got %x", gotScript)
	}
}

//testamountcompression确保特定于域的事务输出量
//压缩和解压缩按预期工作。
func TestAmountCompression(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		uncompressed uint64
		compressed   uint64
	}{
		{
			name:         "0 BTC (sometimes used in nulldata)",
			uncompressed: 0,
			compressed:   0,
		},
		{
			name:         "546 Satoshi (current network dust value)",
			uncompressed: 546,
			compressed:   4911,
		},
		{
			name:         "0.00001 BTC (typical transaction fee)",
			uncompressed: 1000,
			compressed:   4,
		},
		{
			name:         "0.0001 BTC (typical transaction fee)",
			uncompressed: 10000,
			compressed:   5,
		},
		{
			name:         "0.12345678 BTC",
			uncompressed: 12345678,
			compressed:   111111101,
		},
		{
			name:         "0.5 BTC",
			uncompressed: 50000000,
			compressed:   48,
		},
		{
			name:         "1 BTC",
			uncompressed: 100000000,
			compressed:   9,
		},
		{
			name:         "5 BTC",
			uncompressed: 500000000,
			compressed:   49,
		},
		{
			name:         "21000000 BTC (max minted coins)",
			uncompressed: 2100000000000000,
			compressed:   21000000,
		},
	}

	for _, test := range tests {
//确保金额压缩到预期值。
		gotCompressed := compressTxOutAmount(test.uncompressed)
		if gotCompressed != test.compressed {
			t.Errorf("compressTxOutAmount (%s): did not get "+
				"expected value - got %d, want %d", test.name,
				gotCompressed, test.compressed)
			continue
		}

//确保值解压缩到预期值。
		gotDecompressed := decompressTxOutAmount(test.compressed)
		if gotDecompressed != test.uncompressed {
			t.Errorf("decompressTxOutAmount (%s): did not get "+
				"expected value - got %d, want %d", test.name,
				gotDecompressed, test.uncompressed)
			continue
		}
	}
}

//testcompressedtxout确保事务输出序列化和
//反序列化按预期工作。
func TestCompressedTxOut(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		amount     uint64
		pkScript   []byte
		compressed []byte
	}{
		{
			name:       "nulldata with 0 BTC",
			amount:     0,
			pkScript:   hexToBytes("6a200102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"),
			compressed: hexToBytes("00286a200102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"),
		},
		{
			name:       "pay-to-pubkey-hash dust",
			amount:     546,
			pkScript:   hexToBytes("76a9141018853670f9f3b0582c5b9ee8ce93764ac32b9388ac"),
			compressed: hexToBytes("a52f001018853670f9f3b0582c5b9ee8ce93764ac32b93"),
		},
		{
			name:       "pay-to-pubkey uncompressed 1 BTC",
			amount:     100000000,
			pkScript:   hexToBytes("4104192d74d0cb94344c9569c2e77901573d8d7903c3ebec3a957724895dca52c6b40d45264838c0bd96852662ce6a847b197376830160c6d2eb5e6a4c44d33f453eac"),
			compressed: hexToBytes("0904192d74d0cb94344c9569c2e77901573d8d7903c3ebec3a957724895dca52c6b4"),
		},
	}

	for _, test := range tests {
//确保函数计算序列化大小
//实际上，序列化txout是正确计算的。
		gotSize := compressedTxOutSize(test.amount, test.pkScript)
		if gotSize != len(test.compressed) {
			t.Errorf("compressedTxOutSize (%s): did not get "+
				"expected size - got %d, want %d", test.name,
				gotSize, len(test.compressed))
			continue
		}

//确保txout压缩到预期值。
		gotCompressed := make([]byte, gotSize)
		gotBytesWritten := putCompressedTxOut(gotCompressed,
			test.amount, test.pkScript)
		if !bytes.Equal(gotCompressed, test.compressed) {
			t.Errorf("compressTxOut (%s): did not get expected "+
				"bytes - got %x, want %x", test.name,
				gotCompressed, test.compressed)
			continue
		}
		if gotBytesWritten != len(test.compressed) {
			t.Errorf("compressTxOut (%s): did not get expected "+
				"number of bytes written - got %d, want %d",
				test.name, gotBytesWritten,
				len(test.compressed))
			continue
		}

//确保将序列化字节解码回预期的
//未压缩的值。
		gotAmount, gotScript, gotBytesRead, err := decodeCompressedTxOut(
			test.compressed)
		if err != nil {
			t.Errorf("decodeCompressedTxOut (%s): unexpected "+
				"error: %v", test.name, err)
			continue
		}
		if gotAmount != test.amount {
			t.Errorf("decodeCompressedTxOut (%s): did not get "+
				"expected amount - got %d, want %d",
				test.name, gotAmount, test.amount)
			continue
		}
		if !bytes.Equal(gotScript, test.pkScript) {
			t.Errorf("decodeCompressedTxOut (%s): did not get "+
				"expected script - got %x, want %x",
				test.name, gotScript, test.pkScript)
			continue
		}
		if gotBytesRead != len(test.compressed) {
			t.Errorf("decodeCompressedTxOut (%s): did not get "+
				"expected number of bytes read - got %d, want %d",
				test.name, gotBytesRead, len(test.compressed))
			continue
		}
	}
}

//testxoutcompressionErrors确保调用与
//使用不正确数据的txout压缩返回预期结果。
func TestTxOutCompressionErrors(t *testing.T) {
	t.Parallel()

//缺少压缩脚本的压缩txout必须出错。
	compressedTxOut := hexToBytes("00")
	_, _, _, err := decodeCompressedTxOut(compressedTxOut)
	if !isDeserializeErr(err) {
		t.Fatalf("decodeCompressedTxOut with missing compressed script "+
			"did not return expected error type - got %T, want "+
			"errDeserialize", err)
	}

//
	compressedTxOut = hexToBytes("0010")
	_, _, _, err = decodeCompressedTxOut(compressedTxOut)
	if !isDeserializeErr(err) {
		t.Fatalf("decodeCompressedTxOut with short compressed script "+
			"did not return expected error type - got %T, want "+
			"errDeserialize", err)
	}
}
