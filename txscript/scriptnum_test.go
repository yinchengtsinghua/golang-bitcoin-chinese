
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015-2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package txscript

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

//testscriptNumBytes确保从整数脚本号转换为
//字节表示按预期工作。
func TestScriptNumBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		num        scriptNum
		serialized []byte
	}{
		{0, nil},
		{1, hexToBytes("01")},
		{-1, hexToBytes("81")},
		{127, hexToBytes("7f")},
		{-127, hexToBytes("ff")},
		{128, hexToBytes("8000")},
		{-128, hexToBytes("8080")},
		{129, hexToBytes("8100")},
		{-129, hexToBytes("8180")},
		{256, hexToBytes("0001")},
		{-256, hexToBytes("0081")},
		{32767, hexToBytes("ff7f")},
		{-32767, hexToBytes("ffff")},
		{32768, hexToBytes("008000")},
		{-32768, hexToBytes("008080")},
		{65535, hexToBytes("ffff00")},
		{-65535, hexToBytes("ffff80")},
		{524288, hexToBytes("000008")},
		{-524288, hexToBytes("000088")},
		{7340032, hexToBytes("000070")},
		{-7340032, hexToBytes("0000f0")},
		{8388608, hexToBytes("00008000")},
		{-8388608, hexToBytes("00008080")},
		{2147483647, hexToBytes("ffffff7f")},
		{-2147483647, hexToBytes("ffffffff")},

//对于解释为
//数字，但允许作为数值运算的结果。
		{2147483648, hexToBytes("0000008000")},
		{-2147483648, hexToBytes("0000008080")},
		{2415919104, hexToBytes("0000009000")},
		{-2415919104, hexToBytes("0000009080")},
		{4294967295, hexToBytes("ffffffff00")},
		{-4294967295, hexToBytes("ffffffff80")},
		{4294967296, hexToBytes("0000000001")},
		{-4294967296, hexToBytes("0000000081")},
		{281474976710655, hexToBytes("ffffffffffff00")},
		{-281474976710655, hexToBytes("ffffffffffff80")},
		{72057594037927935, hexToBytes("ffffffffffffff00")},
		{-72057594037927935, hexToBytes("ffffffffffffff80")},
		{9223372036854775807, hexToBytes("ffffffffffffff7f")},
		{-9223372036854775807, hexToBytes("ffffffffffffffff")},
	}

	for _, test := range tests {
		gotBytes := test.num.Bytes()
		if !bytes.Equal(gotBytes, test.serialized) {
			t.Errorf("Bytes: did not get expected bytes for %d - "+
				"got %x, want %x", test.num, gotBytes,
				test.serialized)
			continue
		}
	}
}

//testmakesccriptnum确保从字节表示转换为
//完整的脚本编号按预期工作。
func TestMakeScriptNum(t *testing.T) {
	t.Parallel()

//为方便和
//保持水平测试尺寸较短。
	errNumTooBig := scriptError(ErrNumberTooBig, "")
	errMinimalData := scriptError(ErrMinimalData, "")

	tests := []struct {
		serialized      []byte
		num             scriptNum
		numLen          int
		minimalEncoding bool
		err             error
	}{
//最小编码必须拒绝负0。
		{hexToBytes("80"), 0, defaultScriptNumLen, true, errMinimalData},

//最小编码有效值和最小编码标志。
//不应出错并返回预期的整数。
		{nil, 0, defaultScriptNumLen, true, nil},
		{hexToBytes("01"), 1, defaultScriptNumLen, true, nil},
		{hexToBytes("81"), -1, defaultScriptNumLen, true, nil},
		{hexToBytes("7f"), 127, defaultScriptNumLen, true, nil},
		{hexToBytes("ff"), -127, defaultScriptNumLen, true, nil},
		{hexToBytes("8000"), 128, defaultScriptNumLen, true, nil},
		{hexToBytes("8080"), -128, defaultScriptNumLen, true, nil},
		{hexToBytes("8100"), 129, defaultScriptNumLen, true, nil},
		{hexToBytes("8180"), -129, defaultScriptNumLen, true, nil},
		{hexToBytes("0001"), 256, defaultScriptNumLen, true, nil},
		{hexToBytes("0081"), -256, defaultScriptNumLen, true, nil},
		{hexToBytes("ff7f"), 32767, defaultScriptNumLen, true, nil},
		{hexToBytes("ffff"), -32767, defaultScriptNumLen, true, nil},
		{hexToBytes("008000"), 32768, defaultScriptNumLen, true, nil},
		{hexToBytes("008080"), -32768, defaultScriptNumLen, true, nil},
		{hexToBytes("ffff00"), 65535, defaultScriptNumLen, true, nil},
		{hexToBytes("ffff80"), -65535, defaultScriptNumLen, true, nil},
		{hexToBytes("000008"), 524288, defaultScriptNumLen, true, nil},
		{hexToBytes("000088"), -524288, defaultScriptNumLen, true, nil},
		{hexToBytes("000070"), 7340032, defaultScriptNumLen, true, nil},
		{hexToBytes("0000f0"), -7340032, defaultScriptNumLen, true, nil},
		{hexToBytes("00008000"), 8388608, defaultScriptNumLen, true, nil},
		{hexToBytes("00008080"), -8388608, defaultScriptNumLen, true, nil},
		{hexToBytes("ffffff7f"), 2147483647, defaultScriptNumLen, true, nil},
		{hexToBytes("ffffffff"), -2147483647, defaultScriptNumLen, true, nil},
		{hexToBytes("ffffffff7f"), 549755813887, 5, true, nil},
		{hexToBytes("ffffffffff"), -549755813887, 5, true, nil},
		{hexToBytes("ffffffffffffff7f"), 9223372036854775807, 8, true, nil},
		{hexToBytes("ffffffffffffffff"), -9223372036854775807, 8, true, nil},
		{hexToBytes("ffffffffffffffff7f"), -1, 9, true, nil},
		{hexToBytes("ffffffffffffffffff"), 1, 9, true, nil},
		{hexToBytes("ffffffffffffffffff7f"), -1, 10, true, nil},
		{hexToBytes("ffffffffffffffffffff"), 1, 10, true, nil},

//超出数据范围的最小编码值
//被解释为具有最小编码的脚本编号
//标志集。应出错并返回0。
		{hexToBytes("0000008000"), 0, defaultScriptNumLen, true, errNumTooBig},
		{hexToBytes("0000008080"), 0, defaultScriptNumLen, true, errNumTooBig},
		{hexToBytes("0000009000"), 0, defaultScriptNumLen, true, errNumTooBig},
		{hexToBytes("0000009080"), 0, defaultScriptNumLen, true, errNumTooBig},
		{hexToBytes("ffffffff00"), 0, defaultScriptNumLen, true, errNumTooBig},
		{hexToBytes("ffffffff80"), 0, defaultScriptNumLen, true, errNumTooBig},
		{hexToBytes("0000000001"), 0, defaultScriptNumLen, true, errNumTooBig},
		{hexToBytes("0000000081"), 0, defaultScriptNumLen, true, errNumTooBig},
		{hexToBytes("ffffffffffff00"), 0, defaultScriptNumLen, true, errNumTooBig},
		{hexToBytes("ffffffffffff80"), 0, defaultScriptNumLen, true, errNumTooBig},
		{hexToBytes("ffffffffffffff00"), 0, defaultScriptNumLen, true, errNumTooBig},
		{hexToBytes("ffffffffffffff80"), 0, defaultScriptNumLen, true, errNumTooBig},
		{hexToBytes("ffffffffffffff7f"), 0, defaultScriptNumLen, true, errNumTooBig},
		{hexToBytes("ffffffffffffffff"), 0, defaultScriptNumLen, true, errNumTooBig},

//非最小编码，但在其他情况下有效值为
//最小编码标志。应出错并返回0。
{hexToBytes("00"), 0, defaultScriptNumLen, true, errMinimalData},       //零
{hexToBytes("0100"), 0, defaultScriptNumLen, true, errMinimalData},     //一
{hexToBytes("7f00"), 0, defaultScriptNumLen, true, errMinimalData},     //一百二十七
{hexToBytes("800000"), 0, defaultScriptNumLen, true, errMinimalData},   //一百二十八
{hexToBytes("810000"), 0, defaultScriptNumLen, true, errMinimalData},   //一百二十九
{hexToBytes("000100"), 0, defaultScriptNumLen, true, errMinimalData},   //二百五十六
{hexToBytes("ff7f00"), 0, defaultScriptNumLen, true, errMinimalData},   //三万二千七百六十七
{hexToBytes("00800000"), 0, defaultScriptNumLen, true, errMinimalData}, //三万二千七百六十八
{hexToBytes("ffff0000"), 0, defaultScriptNumLen, true, errMinimalData}, //六万五千五百三十五
{hexToBytes("00000800"), 0, defaultScriptNumLen, true, errMinimalData}, //五十二万四千二百八十八
{hexToBytes("00007000"), 0, defaultScriptNumLen, true, errMinimalData}, //七百三十四万零三十二
{hexToBytes("0009000100"), 0, 5, true, errMinimalData},                 //一千六百七十七万九千五百二十

//非最小编码，但在其他情况下有效值没有
//最小编码标志。不应出错，应返回
//整数。
		{hexToBytes("00"), 0, defaultScriptNumLen, false, nil},
		{hexToBytes("0100"), 1, defaultScriptNumLen, false, nil},
		{hexToBytes("7f00"), 127, defaultScriptNumLen, false, nil},
		{hexToBytes("800000"), 128, defaultScriptNumLen, false, nil},
		{hexToBytes("810000"), 129, defaultScriptNumLen, false, nil},
		{hexToBytes("000100"), 256, defaultScriptNumLen, false, nil},
		{hexToBytes("ff7f00"), 32767, defaultScriptNumLen, false, nil},
		{hexToBytes("00800000"), 32768, defaultScriptNumLen, false, nil},
		{hexToBytes("ffff0000"), 65535, defaultScriptNumLen, false, nil},
		{hexToBytes("00000800"), 524288, defaultScriptNumLen, false, nil},
		{hexToBytes("00007000"), 7340032, defaultScriptNumLen, false, nil},
		{hexToBytes("0009000100"), 16779520, 5, false, nil},
	}

	for _, test := range tests {
//确保错误代码是预期的类型，并且错误
//代码与测试实例中指定的值匹配。
		gotNum, err := makeScriptNum(test.serialized, test.minimalEncoding,
			test.numLen)
		if e := tstCheckScriptError(err, test.err); e != nil {
			t.Errorf("makeScriptNum(%#x): %v", test.serialized, e)
			continue
		}

		if gotNum != test.num {
			t.Errorf("makeScriptNum(%#x): did not get expected "+
				"number - got %d, want %d", test.serialized,
				gotNum, test.num)
			continue
		}
	}
}

//testscriptNumInt32确保脚本号上的Int32函数的行为
//果不其然。
func TestScriptNumInt32(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   scriptNum
		want int32
	}{
//有效Int32范围内的值只是值
//他们自己铸造成一个Int32。
		{0, 0},
		{1, 1},
		{-1, -1},
		{127, 127},
		{-127, -127},
		{128, 128},
		{-128, -128},
		{129, 129},
		{-129, -129},
		{256, 256},
		{-256, -256},
		{32767, 32767},
		{-32767, -32767},
		{32768, 32768},
		{-32768, -32768},
		{65535, 65535},
		{-65535, -65535},
		{524288, 524288},
		{-524288, -524288},
		{7340032, 7340032},
		{-7340032, -7340032},
		{8388608, 8388608},
		{-8388608, -8388608},
		{2147483647, 2147483647},
		{-2147483647, -2147483647},
		{-2147483648, -2147483648},

//有效Int32范围之外的值限制为Int32。
		{2147483648, 2147483647},
		{-2147483649, -2147483648},
		{1152921504606846975, 2147483647},
		{-1152921504606846975, -2147483648},
		{2305843009213693951, 2147483647},
		{-2305843009213693951, -2147483648},
		{4611686018427387903, 2147483647},
		{-4611686018427387903, -2147483648},
		{9223372036854775807, 2147483647},
		{-9223372036854775808, -2147483648},
	}

	for _, test := range tests {
		got := test.in.Int32()
		if got != test.want {
			t.Errorf("Int32: did not get expected value for %d - "+
				"got %d, want %d", test.in, got, test.want)
			continue
		}
	}
}
