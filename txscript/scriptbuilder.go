
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2015 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package txscript

import (
	"encoding/binary"
	"fmt"
)

const (
//DefaultScriptAlloc是用于支持数组的默认大小
//脚本生成器正在生成的脚本。数组将
//根据需要动态增长，但此图旨在提供
//足够的空间来容纳绝大多数脚本，而无需增加
//多次备份数组。
	defaultScriptAlloc = 500
)

//errscriptNotCanonical标识非规范脚本。呼叫方可以使用
//用于检测此错误类型的类型断言。
type ErrScriptNotCanonical string

//错误实现错误接口。
func (e ErrScriptNotCanonical) Error() string {
	return string(e)
}

//ScriptBuilder为构建自定义脚本提供了一个工具。它允许
//您可以在遵守规范编码的同时推送操作码、整数和数据。在
//一般情况下，它不能确保脚本正确执行，但是
//超过脚本引擎允许的最大限制的数据推送
//因此，保证不执行将不会被推送，并将导致
//脚本函数返回错误。
//
//例如，下面将构建一个三分之二的multisig脚本，用于
//付费脚本哈希（尽管在这种情况下multisigscript（）是
//生成脚本的更好选择）：
//生成器：=txscript.newscriptBuilder（）
//builder.addop（txscript.op_2）.adddata（pubkey1）.adddata（pubkey2）
//builder.adddata（pubkey3）.addop（txscript.op_3）
//builder.addop（txscript.op_checkmultisig）
//脚本，错误：=builder.script（）。
//如果犯错！= nIL{
////处理错误。
//返回
//}
//fmt.printf（“最终多信号脚本：%x\n”，脚本）
type ScriptBuilder struct {
	script []byte
	err    error
}

//addop将传递的操作码推送到脚本末尾。脚本不会
//如果按操作码会导致脚本超过
//允许的最大脚本引擎大小。
func (b *ScriptBuilder) AddOp(opcode byte) *ScriptBuilder {
	if b.err != nil {
		return b
	}

//推送将导致脚本超过允许的最大值
//脚本大小将导致非规范脚本。
	if len(b.script)+1 > MaxScriptSize {
		str := fmt.Sprintf("adding an opcode would exceed the maximum "+
			"allowed canonical script length of %d", MaxScriptSize)
		b.err = ErrScriptNotCanonical(str)
		return b
	}

	b.script = append(b.script, opcode)
	return b
}

//addops将传递的操作码推送到脚本末尾。脚本将
//如果按操作码会导致脚本超过
//允许的最大脚本引擎大小。
func (b *ScriptBuilder) AddOps(opcodes []byte) *ScriptBuilder {
	if b.err != nil {
		return b
	}

//推送将导致脚本超过允许的最大值
//脚本大小将导致非规范脚本。
	if len(b.script)+len(opcodes) > MaxScriptSize {
		str := fmt.Sprintf("adding opcodes would exceed the maximum "+
			"allowed canonical script length of %d", MaxScriptSize)
		b.err = ErrScriptNotCanonical(str)
		return b
	}

	b.script = append(b.script, opcodes...)
	return b
}

//CanonicalDataSize返回
//数据需要。
func canonicalDataSize(data []byte) int {
	dataLen := len(data)

//当数据由一个可以表示的数字组成时
//通过一个“小整数”操作码，该操作码将
//数据推送操作码，后面跟着数字。
	if dataLen == 0 {
		return 1
	} else if dataLen == 1 && data[0] <= 16 {
		return 1
	} else if dataLen == 1 && data[0] == 0x81 {
		return 1
	}

	if dataLen < OP_PUSHDATA1 {
		return 1 + dataLen
	} else if dataLen <= 0xff {
		return 2 + dataLen
	} else if dataLen <= 0xffff {
		return 3 + dataLen
	}

	return 5 + dataLen
}

//adddata是一个内部函数，它实际上将传递的数据推送到
//脚本结束。它自动选择标准操作码取决于
//数据的长度。零长度缓冲区将导致空的推送
//数据到堆栈（op_0）。此函数不强制任何数据限制。
func (b *ScriptBuilder) addData(data []byte) *ScriptBuilder {
	dataLen := len(data)

//当数据由一个可以表示的数字组成时
//通过一个“小整数”操作码，使用该操作码而不是
//数据推送操作码，后跟数字。
	if dataLen == 0 || dataLen == 1 && data[0] == 0 {
		b.script = append(b.script, OP_0)
		return b
	} else if dataLen == 1 && data[0] <= 16 {
		b.script = append(b.script, (OP_1-1)+data[0])
		return b
	} else if dataLen == 1 && data[0] == 0x81 {
		b.script = append(b.script, byte(OP_1NEGATE))
		return b
	}

//如果数据长度较小，请使用其中一个操作码
//足够了，所以数据推送指令只是一个字节。
//否则，选择尽可能小的op pushdata操作码
//可以表示数据的长度。
	if dataLen < OP_PUSHDATA1 {
		b.script = append(b.script, byte((OP_DATA_1-1)+dataLen))
	} else if dataLen <= 0xff {
		b.script = append(b.script, OP_PUSHDATA1, byte(dataLen))
	} else if dataLen <= 0xffff {
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, uint16(dataLen))
		b.script = append(b.script, OP_PUSHDATA2)
		b.script = append(b.script, buf...)
	} else {
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, uint32(dataLen))
		b.script = append(b.script, OP_PUSHDATA4)
		b.script = append(b.script, buf...)
	}

//附加实际数据。
	b.script = append(b.script, data...)

	return b
}

//addfulldata通常不应该由普通用户使用，因为它不应该
//包括防止数据推送大于允许的最大值的检查
//导致无法执行的脚本的大小。这是为
//测试目的，如有意制作尺寸的回归测试
//大于允许值。
//
//改用adddata。
func (b *ScriptBuilder) AddFullData(data []byte) *ScriptBuilder {
	if b.err != nil {
		return b
	}

	return b.addData(data)
}

//adddata将传递的数据推送到脚本的末尾。它是自动的
//根据数据长度选择规范操作码。零长
//缓冲区将导致将空数据推送到堆栈（op_0）和任何推送
//大于MaxScriptElementSize的数据将不会修改脚本，因为
//脚本引擎不允许这样做。另外，脚本将不会
//如果推送数据会导致脚本超过最大值，则进行修改
//允许的脚本引擎大小。
func (b *ScriptBuilder) AddData(data []byte) *ScriptBuilder {
	if b.err != nil {
		return b
	}

//推送将导致脚本超过允许的最大值
//脚本大小将导致非规范脚本。
	dataSize := canonicalDataSize(data)
	if len(b.script)+dataSize > MaxScriptSize {
		str := fmt.Sprintf("adding %d bytes of data would exceed the "+
			"maximum allowed canonical script length of %d",
			dataSize, MaxScriptSize)
		b.err = ErrScriptNotCanonical(str)
		return b
	}

//推送大于最大脚本元素大小将导致
//脚本不规范。
	dataLen := len(data)
	if dataLen > MaxScriptElementSize {
		str := fmt.Sprintf("adding a data element of %d bytes would "+
			"exceed the maximum allowed script element size of %d",
			dataLen, MaxScriptElementSize)
		b.err = ErrScriptNotCanonical(str)
		return b
	}

	return b.addData(data)
}

//addint64将传递的整数推送到脚本末尾。脚本将
//如果推送数据会导致脚本超过
//允许的最大脚本引擎大小。
func (b *ScriptBuilder) AddInt64(val int64) *ScriptBuilder {
	if b.err != nil {
		return b
	}

//推送将导致脚本超过允许的最大值
//脚本大小将导致非规范脚本。
	if len(b.script)+1 > MaxScriptSize {
		str := fmt.Sprintf("adding an integer would exceed the "+
			"maximum allow canonical script length of %d",
			MaxScriptSize)
		b.err = ErrScriptNotCanonical(str)
		return b
	}

//小整数和运算负的快速路径。
	if val == 0 {
		b.script = append(b.script, OP_0)
		return b
	}
	if val == -1 || (val >= 1 && val <= 16) {
		b.script = append(b.script, byte((OP_1-1)+val))
		return b
	}

	return b.AddData(scriptNum(val).Bytes())
}

//重置将重置脚本，使其不包含任何内容。
func (b *ScriptBuilder) Reset() *ScriptBuilder {
	b.script = b.script[0:0]
	b.err = nil
	return b
}

//脚本返回当前生成的脚本。当发生任何错误时
//构建脚本时，脚本将返回到第一个
//错误和错误。
func (b *ScriptBuilder) Script() ([]byte, error) {
	return b.script, b.err
}

//NewScriptBuilder返回脚本生成器的新实例。见
//有关详细信息，请参阅ScriptBuilder。
func NewScriptBuilder() *ScriptBuilder {
	return &ScriptBuilder{
		script: make([]byte, 0, defaultScriptAlloc),
	}
}
