
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
	"fmt"
)

const (
	maxInt32 = 1<<31 - 1
	minInt32 = -1 << 31

//DefaultScriptNumlen是默认的字节数
//被解释为整数的数据可以是。
	defaultScriptNumLen = 4
)

//scriptnum表示脚本引擎中使用的数值，
//处理共识所需的微妙语义的特殊处理。
//
//所有的数字都存储在数据中，并且交替的堆栈编码得很少。
//以符号位结尾。所有数字操作码，如op_add、op_sub，
//和op mul，只允许对范围内的4字节整数进行操作
//[-2^31+1，2^31-1]，但是数值运算的结果可能溢出
//并保持有效，只要它们不被用作其他数字的输入
//操作或以其他方式解释为整数。
//
//例如，op_add的两个操作数可以有2^31-1
//结果2^32-2溢出，但仍作为
//添加的结果。然后，该值可用作操作验证的输入。
//这将成功，因为数据被解释为布尔值。
//但是，如果将相同的值用作另一个数字的输入
//操作码，如opu-sub，必须失败。
//
//此类型通过存储所有数字来处理上述要求
//操作结果为Int64以处理溢出并提供字节
//方法获取序列化表示形式（包括溢出的值）。
//
//然后，每当数据被解释为整数时，它就会被转换为
//通过使用makescriptnum函数键入，如果
//数字超出范围或不是最小编码取决于参数。
//因为所有的数字操作码都涉及从堆栈中提取数据和
//将其解释为整数，它提供所需的行为。
type scriptNum int64

//checkminimadataencoding返回传递的字节数组是否符合
//达到最低的编码要求。
func checkMinimalDataEncoding(v []byte) error {
	if len(v) == 0 {
		return nil
	}

//检查数字是否以最小可能值编码
//字节数。
//
//如果最高有效字节（不包括符号位）为零
//那我们就不是最小的了。注意此测试如何也拒绝
//负零编码，[0x80]。
	if v[len(v)-1]&0x7f == 0 {
//一个例外：如果有多个字节
//设置第二个最高有效字节的有效位
//它将与符号位冲突。这个案例的一个例子
//为+-255，分别编码为0xff00和0xff80。
//（大字节）。
		if len(v) == 1 || v[len(v)-2]&0x80 == 0 {
			str := fmt.Sprintf("numeric value encoded as %x is "+
				"not minimally encoded", v)
			return scriptError(ErrMinimalData, str)
		}
	}

	return nil
}

//字节返回序列化为带符号位的小尾数的数字。
//
//编码示例：
//127＞[0x7F]
//- 127＞0xFFF
//128->[0x80 0x00]
//-128->[0x80 0x80]
//129->[0x81 0x00]
//-129->[0x81 0x80]
//256->[0x00 0x01]
//-256->[0x00 0x81]
//32767->[0xFF 0x7F]
//-32767->[0xff 0xff]
//32768->[0x00 0x80 0x00]
//-32768->[0x00 0x80 0x80]
func (n scriptNum) Bytes() []byte {
//零编码为空字节片。
	if n == 0 {
		return nil
	}

//获取绝对值并跟踪它是否是最初的
//否定的。
	isNegative := n < 0
	if isNegative {
		n = -n
	}

//编码为小尾数。最大编码字节数为9
//（对于max int64，8字节加上符号扩展的潜在字节）。
	result := make([]byte, 0, 9)
	for n > 0 {
		result = append(result, byte(n&0xff))
		n >>= 8
	}

//当最高有效字节已经设置了高位时，
//需要额外的高字节来指示数字是否为
//阴性或阳性。转换时会删除附加字节
//回到整数，它的高位用来表示符号。
//
//否则，当最重要的字节没有
//高位设置，如果需要，使用它来指示值为负。
	if result[len(result)-1]&0x80 != 0 {
		extraByte := byte(0x00)
		if isNegative {
			extraByte = 0x80
		}
		result = append(result, extraByte)

	} else if isNegative {
		result[len(result)-1] |= 0x80
	}

	return result
}

//Int32返回夹紧到有效Int32的脚本号。这就是说
//当脚本号高于允许的最大Int32时，最大Int32
//对于最小值，返回值，反之亦然。注意这个
//行为与简单的Int32转换不同，因为它截断
//一致性规则规定了直接转换成整数的数字。
//提供此行为。
//
//在实践中，对于大多数操作码来说，数字永远不应该超出范围，因为
//它将使用makescriptnum创建，并使用defaultscriptlen
//值，拒绝它们。万一将来有什么事情
//此函数与某些算术的结果相反，允许
//在重新解释为整数之前超出范围，这将提供
//正确的行为。
func (n scriptNum) Int32() int32 {
	if n > maxInt32 {
		return maxInt32
	}

	if n < minInt32 {
		return minInt32
	}

	return int32(n)
}

//MakeScriptNum将传递的序列化字节解释为编码整数
//并以脚本号返回结果。
//
//因为共识规则规定串行字节被解释为int
//只允许在由最大字节数确定的范围内，
//在每个操作码的基础上，当提供的字节
//会导致一个超出该范围的数字。尤其是
//处理数值的绝大多数操作码仅限于4
//字节，因此会将该值传递给该函数，从而导致
//允许范围为[-2^31+1，2^31-1]。
//
//如果进行其他检查，RequireMinimal标志将导致返回错误
//在编码上，确定它不是用尽可能小的
//字节数或为负0编码，[0x80]。例如，考虑
//数字127。它可以编码为[0x7f]、[0x7f 0x00]，
//[0x7F 0x00 0x00…]等。除[0x7F]之外的所有窗体都将返回错误
//已启用RequireMinimal。
//
//scriptNumlen是编码值可以达到的最大字节数。
//返回errstackNumberTooBig之前。这有效地限制了
//允许值的范围。
//警告：如果传递的值大于
//defaultscriptnumlen，这可能导致加法和乘法
//溢出。
//
//参见字节函数文档，例如编码。
func makeScriptNum(v []byte, requireMinimal bool, scriptNumLen int) (scriptNum, error) {
//解释数据要求不大于
//传递的scriptNumlen值。
	if len(v) > scriptNumLen {
		str := fmt.Sprintf("numeric value encoded as %x is %d bytes "+
			"which exceeds the max allowed of %d", v, len(v),
			scriptNumLen)
		return 0, scriptError(ErrNumberTooBig, str)
	}

//如果请求，强制最小编码。
	if requireMinimal {
		if err := checkMinimalDataEncoding(v); err != nil {
			return 0, err
		}
	}

//零被编码为空字节片。
	if len(v) == 0 {
		return 0, nil
	}

//从小endian解码。
	var result int64
	for i, val := range v {
		result |= int64(val) << uint8(8*i)
	}

//当输入字节的最高有效字节具有符号位时
//设置，结果为负。所以，从结果中删除符号位
//把它变成负面的。
	if v[len(v)-1]&0x80 != 0 {
//V的最大长度已确定为4
//上面，所以uint8足以覆盖最大可能的移位
//价值24。
		result &= ^(int64(0x80) << uint8(8*(len(v)-1)))
		return scriptNum(-result), nil
	}

	return scriptNum(result), nil
}
