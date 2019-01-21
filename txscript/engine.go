
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2018 BTCSuite开发者
//版权所有（c）2015-2018法令开发商
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package txscript

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/wire"
)

//scriptflags是一个位掩码，用于定义将要执行的其他操作或测试
//在执行脚本对时完成。
type ScriptFlags uint32

const (
//scriptBip16定义bip16阈值是否已通过，因此
//支付到脚本哈希事务将完全验证。
	ScriptBip16 ScriptFlags = 1 << iota

//scriptstrictmultisig定义是否验证堆栈项
//checkmultisig使用的长度为零。
	ScriptStrictMultiSig

//script不鼓励upgradablenops定义是否验证
//nop1到nop10是为将来的软分叉升级保留的。这个
//标志不能用于协商一致的关键代码，也不能应用于
//块，因为此标志仅用于更严格的标准事务
//检查。仅当上述操作码为
//执行。
	ScriptDiscourageUpgradableNops

//ScriptVerifyCheckLockTimeVerify定义是否验证
//基于锁定时间，事务输出是可以使用的。
//这是Bip0065。
	ScriptVerifyCheckLockTimeVerify

//ScriptVerifyCheckSequenceVerify定义是否允许执行
//根据输出的时间限制脚本的路径
//被浪费。这是Bip0112。
	ScriptVerifyCheckSequenceVerify

//scriptVerifyCleanStack定义堆栈必须只包含
//一个堆栈元素，在计算之后，该元素必须
//如果解释为布尔值，则为true。这是BIP0062的规则6。
//如果没有scriptBip16标志或
//脚本验证见证标志。
	ScriptVerifyCleanStack

//ScriptVerifyDerSignatures定义需要签名
//以DER格式编译。
	ScriptVerifyDERSignatures

//scriptVerifylows定义需要签名才能符合
//der格式，其s值<=order/2。这是规则5。
//BIP062的。
	ScriptVerifyLowS

//scriptVerifyMinimalData定义签名必须使用最小的
//推算符。这是bip0062的规则3和4。
	ScriptVerifyMinimalData

//scriptVerifyNullFail定义签名必须为空，如果
//checksig或checkmultisig操作失败。
	ScriptVerifyNullFail

//scriptVerifySigPushOnly定义签名脚本必须包含
//仅推送数据。这是bip0062的规则2。
	ScriptVerifySigPushOnly

//scriptVerifyStricteEncoding定义签名脚本和
//公钥必须遵循严格的编码要求。
	ScriptVerifyStrictEncoding

//脚本验证见证定义是否验证事务
//使用见证程序模板输出。
	ScriptVerifyWitness

//脚本验证阻止可升级的见证程序生成见证
//2-16版非标准程序。
	ScriptVerifyDiscourageUpgradeableWitnessProgram

//scriptVerifyMinimaif用op if/op notif制作脚本
//操作数不是空向量或[0x01]非标准的。
	ScriptVerifyMinimalIf

//scriptVerifyWitnessPubKeyType在检查信号中生成脚本
//其公钥未以压缩格式序列化的操作
//非标准的。
	ScriptVerifyWitnessPubKeyType
)

const (
//maxstacksize是堆栈和alt堆栈的最大组合高度
//执行期间。
	MaxStackSize = 1000

//maxscriptsize是原始脚本允许的最大长度。
	MaxScriptSize = 10000

//PayToWitnessPubKeyHashDataSize是见证程序的大小
//数据推送以获取付费见证发布密钥哈希输出。
	payToWitnessPubKeyHashDataSize = 20

//PayToWitnessScriptHashDataSize是见证程序的大小
//付款见证脚本哈希输出的数据推送。
	payToWitnessScriptHashDataSize = 32
)

//半阶用于调节ECDSA延展性（参见BIP0062）。
var halfOrder = new(big.Int).Rsh(btcec.S256().N, 1)

//引擎是执行脚本的虚拟机。
type Engine struct {
	scripts         [][]parsedOpcode
	scriptIdx       int
	scriptOff       int
	lastCodeSep     int
dstack          stack //数据堆栈
astack          stack //ALT栈
	tx              wire.MsgTx
	txIdx           int
	condStack       []int
	numOps          int
	flags           ScriptFlags
	sigCache        *SigCache
	hashCache       *TxSigHashes
bip16           bool     //将执行视为按脚本付费哈希
savedFirstStack [][]byte //bip16脚本的第一个脚本的堆栈
	witnessVersion  int
	witnessProgram  []byte
	inputAmount     int64
}

//HasFlag返回脚本引擎实例是否设置了传递的标志。
func (vm *Engine) hasFlag(flag ScriptFlags) bool {
	return vm.flags&flag == flag
}

//IsBranchExecuting返回当前条件分支是否为
//积极执行。例如，当数据栈上有一个opu-false时
//如果遇到OPU，则分支在其他OPU或
//遇到opendif。它正确地处理嵌套条件。
func (vm *Engine) isBranchExecuting() bool {
	if len(vm.condStack) == 0 {
		return true
	}
	return vm.condStack[len(vm.condStack)-1] == OpCondTrue
}

//executeopcode对传递的操作码执行peforms。它考虑到了
//它是否被条件隐藏，但某些规则仍然必须
//在这种情况下测试。
func (vm *Engine) executeOpcode(pop *parsedOpcode) error {
//禁用的操作码在程序计数器上失败。
	if pop.isDisabled() {
		str := fmt.Sprintf("attempt to execute disabled opcode %s",
			pop.opcode.name)
		return scriptError(ErrDisabledOpcode, str)
	}

//程序计数器上的非法操作码总是失败的。
	if pop.alwaysIllegal() {
		str := fmt.Sprintf("attempt to execute reserved opcode %s",
			pop.opcode.name)
		return scriptError(ErrReservedOpcode, str)
	}

//请注意，这包括作为推送操作计数的opu reserved。
	if pop.opcode.value > OP_16 {
		vm.numOps++
		if vm.numOps > MaxOpsPerScript {
			str := fmt.Sprintf("exceeded max operation limit of %d",
				MaxOpsPerScript)
			return scriptError(ErrTooManyOperations, str)
		}

	} else if len(pop.data) > MaxScriptElementSize {
		str := fmt.Sprintf("element size %d exceeds max allowed size %d",
			len(pop.data), MaxScriptElementSize)
		return scriptError(ErrElementTooBig, str)
	}

//如果这不是条件操作码，而且它是
//不在执行分支中。
	if !vm.isBranchExecuting() && !pop.isConditional() {
		return nil
	}

//确保所有执行的数据推送操作码在
//设置了最小数据验证标志。
	if vm.dstack.verifyMinimalData && vm.isBranchExecuting() &&
		pop.opcode.value >= 0 && pop.opcode.value <= OP_PUSHDATA4 {

		if err := pop.checkMinimalDataPush(); err != nil {
			return err
		}
	}

	return pop.opcode.opfunc(pop, vm)
}

//disasm是一个帮助函数，用于生成disasmpc和
//DisasmScript。它产生的操作码前缀是程序计数器
//在脚本中提供的位置。它没有错误检查并留下
//向调用方提供有效的偏移量。
func (vm *Engine) disasm(scriptIdx int, scriptOff int) string {
	return fmt.Sprintf("%02x:%04x: %s", scriptIdx, scriptOff,
		vm.scripts[scriptIdx][scriptOff].print(false))
}

//如果当前脚本位置对于
//执行，否则为零。
func (vm *Engine) validPC() error {
	if vm.scriptIdx >= len(vm.scripts) {
		str := fmt.Sprintf("past input scripts %v:%v %v:xxxx",
			vm.scriptIdx, vm.scriptOff, len(vm.scripts))
		return scriptError(ErrInvalidProgramCounter, str)
	}
	if vm.scriptOff >= len(vm.scripts[vm.scriptIdx]) {
		str := fmt.Sprintf("past input scripts %v:%v %v:%04d",
			vm.scriptIdx, vm.scriptOff, vm.scriptIdx,
			len(vm.scripts[vm.scriptIdx]))
		return scriptError(ErrInvalidProgramCounter, str)
	}
	return nil
}

//curpc返回当前脚本和偏移量，或者如果
//位置无效。
func (vm *Engine) curPC() (script int, off int, err error) {
	err = vm.validPC()
	if err != nil {
		return 0, 0, err
	}
	return vm.scriptIdx, vm.scriptOff, nil
}

//如果提取了见证程序，则IsWitnessVersionActive返回true
//在引擎初始化期间，程序的版本匹配
//指定的版本。
func (vm *Engine) isWitnessVersionActive(version uint) bool {
	return vm.witnessProgram != nil && uint(vm.witnessVersion) == version
}

//verifywitnessprogram使用传递的
//见证作为输入。
func (vm *Engine) verifyWitnessProgram(witness [][]byte) error {
	if vm.isWitnessVersionActive(0) {
		switch len(vm.witnessProgram) {
case payToWitnessPubKeyHashDataSize: //2WKH
//见证堆栈应该正好由两个组成
//项目：签名和公钥。
			if len(witness) != 2 {
				err := fmt.Sprintf("should have exactly two "+
					"items in witness, instead have %v", len(witness))
				return scriptError(ErrWitnessProgramMismatch, err)
			}

//现在我们将恢复执行，就好像它是一个常规的
//p2pkh事务。
			pkScript, err := payToPubKeyHashScript(vm.witnessProgram)
			if err != nil {
				return err
			}
			pops, err := parseScript(pkScript)
			if err != nil {
				return err
			}

//将堆栈设置为提供的见证堆栈，然后
//将上面生成的pkscript附加为下一个
//要执行的脚本。
			vm.scripts = append(vm.scripts, pops)
			vm.SetStack(witness)

case payToWitnessScriptHashDataSize: //2WSH
//此外，见证堆栈在
//这一点。
			if len(witness) == 0 {
				return scriptError(ErrWitnessProgramEmpty, "witness "+
					"program empty passed empty witness")
			}

//获取最后一个证人脚本
//传递的堆栈中的元素。脚本的大小
//不得超过最大脚本大小。
			witnessScript := witness[len(witness)-1]
			if len(witnessScript) > MaxScriptSize {
				str := fmt.Sprintf("witnessScript size %d "+
					"is larger than max allowed size %d",
					len(witnessScript), MaxScriptSize)
				return scriptError(ErrScriptTooBig, str)
			}

//确保在
//见证堆栈与见证程序匹配。
			witnessHash := sha256.Sum256(witnessScript)
			if !bytes.Equal(witnessHash[:], vm.witnessProgram) {
				return scriptError(ErrWitnessProgramMismatch,
					"witness program hash mismatch")
			}

//通过所有有效性检查后，分析
//将脚本编写成单独的操作代码，以便W可以执行它
//作为下一个脚本。
			pops, err := parseScript(witnessScript)
			if err != nil {
				return err
			}

//哈希匹配成功，因此使用见证作为
//堆栈，并将见证脚本设置为下一个
//脚本已执行。
			vm.scripts = append(vm.scripts, pops)
			vm.SetStack(witness[:len(witness)-1])

		default:
			errStr := fmt.Sprintf("length of witness program "+
				"must either be %v or %v bytes, instead is %v bytes",
				payToWitnessPubKeyHashDataSize,
				payToWitnessScriptHashDataSize,
				len(vm.witnessProgram))
			return scriptError(ErrWitnessProgramWrongLength, errStr)
		}
	} else if vm.hasFlag(ScriptVerifyDiscourageUpgradeableWitnessProgram) {
		errStr := fmt.Sprintf("new witness program versions "+
			"invalid: %v", vm.witnessProgram)
		return scriptError(ErrDiscourageUpgradableWitnessProgram, errStr)
	} else {
//如果我们遇到一个未知的证人程序版本，
//不会阻碍未来未知证人的软分叉，
//然后我们在虚拟机中取消激活segwit行为
//执行的剩余部分。
		vm.witnessProgram = nil
	}

	if vm.isWitnessVersionActive(0) {
//见证堆栈中的所有元素不得大于
//超过允许推送到的最大字节数
//堆栈。
		for _, witElement := range vm.GetStack() {
			if len(witElement) > MaxScriptElementSize {
				str := fmt.Sprintf("element size %d exceeds "+
					"max allowed size %d", len(witElement),
					MaxScriptElementSize)
				return scriptError(ErrElementTooBig, str)
			}
		}
	}

	return nil
}

//disasmpc返回用于反汇编操作码的字符串，
//在调用step（）时执行。
func (vm *Engine) DisasmPC() (string, error) {
	scriptIdx, scriptOff, err := vm.curPC()
	if err != nil {
		return "", err
	}
	return vm.disasm(scriptIdx, scriptOff), nil
}

//disasmscript返回请求的脚本的反汇编字符串
//偏移指数索引0是签名脚本，1是公钥
//脚本。
func (vm *Engine) DisasmScript(idx int) (string, error) {
	if idx >= len(vm.scripts) {
		str := fmt.Sprintf("script index %d >= total scripts %d", idx,
			len(vm.scripts))
		return "", scriptError(ErrInvalidIndex, str)
	}

	var disstr string
	for i := range vm.scripts[idx] {
		disstr = disstr + vm.disasm(idx, i) + "\n"
	}
	return disstr, nil
}

//如果正在运行的脚本已结束且
//成功，在堆栈上留下一个真正的布尔值。否则是一个错误，
//包括脚本未完成的情况。
func (vm *Engine) CheckErrorCondition(finalScript bool) error {
//检查执行实际上已经完成。当PC超过脚本结尾时
//数组没有更多要运行的脚本。
	if vm.scriptIdx < len(vm.scripts) {
		return scriptError(ErrScriptUnfinished,
			"error check when script unfinished")
	}

//如果我们处于版本零见证执行模式，这是
//最后一个脚本，那么堆栈必须是干净的才能维护
//与Bip16兼容。
	if finalScript && vm.isWitnessVersionActive(0) && vm.dstack.Depth() != 1 {
		return scriptError(ErrEvalFalse, "witness program must "+
			"have clean stack")
	}

	if finalScript && vm.hasFlag(ScriptVerifyCleanStack) &&
		vm.dstack.Depth() != 1 {

		str := fmt.Sprintf("stack contains %d unexpected items",
			vm.dstack.Depth()-1)
		return scriptError(ErrCleanStack, str)
	} else if vm.dstack.Depth() < 1 {
		return scriptError(ErrEmptyStack,
			"stack empty at end of script execution")
	}

	v, err := vm.dstack.PopBool()
	if err != nil {
		return err
	}
	if !v {
//记录有趣的数据。
		log.Tracef("%v", newLogClosure(func() string {
			dis0, _ := vm.DisasmScript(0)
			dis1, _ := vm.DisasmScript(1)
			return fmt.Sprintf("scripts failed: script0: %s\n"+
				"script1: %s", dis0, dis1)
		}))
		return scriptError(ErrEvalFalse,
			"false stack entry at end of script execution")
	}
	return nil
}

//步骤将执行下一条指令并将程序计数器移到
//脚本中的下一个操作码，如果当前脚本已结束，则为下一个脚本。步骤
//如果最后一个操作码成功执行，则返回true。
//
//如果错误为
//返回。
func (vm *Engine) Step() (done bool, err error) {
//验证它是否指向有效的脚本地址。
	err = vm.validPC()
	if err != nil {
		return true, err
	}
	opcode := &vm.scripts[vm.scriptIdx][vm.scriptOff]
	vm.scriptOff++

//在执行操作码的同时考虑以下几点：
//禁用的操作码，非法操作码，每个操作码允许的最大操作数
//脚本、最大脚本元素大小和条件。
	err = vm.executeOpcode(opcode)
	if err != nil {
		return true, err
	}

//数据和alt堆栈组合中的元素数
//不能超过允许的最大堆栈元素数。
	combinedStackSize := vm.dstack.Depth() + vm.astack.Depth()
	if combinedStackSize > MaxStackSize {
		str := fmt.Sprintf("combined stack size %d > max allowed %d",
			combinedStackSize, MaxStackSize)
		return false, scriptError(ErrStackOverflow, str)
	}

//准备下一个指令。
	if vm.scriptOff >= len(vm.scripts[vm.scriptIdx]) {
//跨越两个脚本的“if”是非法的。
		if err == nil && len(vm.condStack) != 0 {
			return false, scriptError(ErrUnbalancedConditional,
				"end of script reached in conditional execution")
		}

//alt堆栈不存在。
		_ = vm.astack.DropN(vm.astack.Depth())

vm.numOps = 0 //每个脚本的操作数。
		vm.scriptOff = 0
		if vm.scriptIdx == 0 && vm.bip16 {
			vm.scriptIdx++
			vm.savedFirstStack = vm.GetStack()
		} else if vm.scriptIdx == 1 && vm.bip16 {
//将我们置于checkErrorCondition（）的末尾
			vm.scriptIdx++
//检查脚本是否成功运行并提取脚本
//从第一个堆栈中执行。
			err := vm.CheckErrorCondition(false)
			if err != nil {
				return false, err
			}

			script := vm.savedFirstStack[len(vm.savedFirstStack)-1]
			pops, err := parseScript(script)
			if err != nil {
				return false, err
			}
			vm.scripts = append(vm.scripts, pops)

//将stack设置为第一个脚本减去
//脚本本身
			vm.SetStack(vm.savedFirstStack[:len(vm.savedFirstStack)-1])
		} else if (vm.scriptIdx == 1 && vm.witnessProgram != nil) ||
(vm.scriptIdx == 2 && vm.witnessProgram != nil && vm.bip16) { //嵌套的2SH。

			vm.scriptIdx++

			witness := vm.tx.TxIn[vm.txIdx].Witness
			if err := vm.verifyWitnessProgram(witness); err != nil {
				return false, err
			}
		} else {
			vm.scriptIdx++
		}
//在野外有零长度的脚本
		if vm.scriptIdx < len(vm.scripts) && vm.scriptOff >= len(vm.scripts[vm.scriptIdx]) {
			vm.scriptIdx++
		}
		vm.lastCodeSep = 0
		if vm.scriptIdx >= len(vm.scripts) {
			return true, nil
		}
	}
	return false, nil
}

//execute将执行脚本引擎中的所有脚本并返回nil
//验证成功或出现错误。
func (vm *Engine) Execute() (err error) {
	done := false
	for !done {
		log.Tracef("%v", newLogClosure(func() string {
			dis, err := vm.DisasmPC()
			if err != nil {
				return fmt.Sprintf("stepping (%v)", err)
			}
			return fmt.Sprintf("stepping %v", dis)
		}))

		done, err = vm.Step()
		if err != nil {
			return err
		}
		log.Tracef("%v", newLogClosure(func() string {
			var dstr, astr string

//如果我们在追踪，就把这些堆扔掉。
			if vm.dstack.Depth() != 0 {
				dstr = "Stack:\n" + vm.dstack.String()
			}
			if vm.astack.Depth() != 0 {
				astr = "AltStack:\n" + vm.astack.String()
			}

			return dstr + astr
		}))
	}

	return vm.CheckErrorCondition(true)
}

//subscript返回自上一个操作代码分隔符以来的脚本。
func (vm *Engine) subScript() []parsedOpcode {
	return vm.scripts[vm.scriptIdx][vm.lastCodeSep:]
}

//checkhashtypeencoding返回传递的hashtype是否符合
//如果启用严格的编码要求。
func (vm *Engine) checkHashTypeEncoding(hashType SigHashType) error {
	if !vm.hasFlag(ScriptVerifyStrictEncoding) {
		return nil
	}

	sigHashType := hashType & ^SigHashAnyOneCanPay
	if sigHashType < SigHashAll || sigHashType > SigHashSingle {
		str := fmt.Sprintf("invalid hash type 0x%x", hashType)
		return scriptError(ErrInvalidSigHashType, str)
	}
	return nil
}

//checkPubKeyEncoding返回传递的公钥是否符合
//如果启用严格的编码要求。
func (vm *Engine) checkPubKeyEncoding(pubKey []byte) error {
	if vm.hasFlag(ScriptVerifyWitnessPubKeyType) &&
		vm.isWitnessVersionActive(0) && !btcec.IsCompressedPubKey(pubKey) {

		str := "only uncompressed keys are accepted post-segwit"
		return scriptError(ErrWitnessPubKeyType, str)
	}

	if !vm.hasFlag(ScriptVerifyStrictEncoding) {
		return nil
	}

	if len(pubKey) == 33 && (pubKey[0] == 0x02 || pubKey[0] == 0x03) {
//压缩的
		return nil
	}
	if len(pubKey) == 65 && pubKey[0] == 0x04 {
//未压缩的
		return nil
	}

	return scriptError(ErrPubKeyType, "unsupported public key type")
}

//checkSignatureEncoding返回传递的签名是否符合
//如果启用严格的编码要求。
func (vm *Engine) checkSignatureEncoding(sig []byte) error {
	if !vm.hasFlag(ScriptVerifyDERSignatures) &&
		!vm.hasFlag(ScriptVerifyLowS) &&
		!vm.hasFlag(ScriptVerifyStrictEncoding) {

		return nil
	}

//DER编码签名的格式如下：
//
//0x30<total length>0x02<length of r><r>0x02<length of s><s>
//-0x30是序列的ASN.1标识符
//-总长度为1字节，并指定所有剩余数据的长度
//-0x02是ASN.1标识符，指定后面跟着一个整数
//-r的长度为1字节，并指定r占用的字节数
//-r是任意长度的big-endian编码数字，其中
//表示签名的r值。DER编码指示
//必须使用最小可能数字对值进行编码
//字节的。这意味着只有当
//设置下一个字节的最高位，以防止
//被解释为负数。
//-0x02再次是ASN.1整数标识符
//-s的长度为1字节，并指定s占用的字节数
//-s是任意长度的big endian编码数字，其中
//表示签名的s值。编码规则是
//与R相同。
	const (
		asn1SequenceID = 0x30
		asn1IntegerID  = 0x02

//minsiglen是der编码签名的最小长度，并且是
//当R和S各为1字节时。
//
//0x30+<1-字节>+0x02+0x01+<byte>+0x2+0x01+<byte>
		minSigLen = 8

//maxsiglen是der编码签名的最大长度，并且是
//当r和s都是33字节时。它是33字节，因为
//256位整数需要32个字节和一个额外的前导空字节
//如果在值中设置了高位，则可能需要。
//
//0x30+<1字节>+0x02+0x21+<33字节>+0x2+0x21+<33字节>
		maxSigLen = 72

//SequenceOffset是签名中的字节偏移量
//应为ASN.1序列标识符。
		sequenceOffset = 0

//datalenoffset是预期的
//签名中所有剩余数据的总长度。
		dataLenOffset = 1

//RtypeOffset是ASN.1签名中的字节偏移量。
//r的标识符，应指示asn.1整数。
		rTypeOffset = 2

//rlenoffset是签名中长度为
//R.
		rLenOffset = 3

//r offset是r签名中的字节偏移量。
		rOffset = 4
	)

//签名必须符合允许的最小和最大长度。
	sigLen := len(sig)
	if sigLen < minSigLen {
		str := fmt.Sprintf("malformed signature: too short: %d < %d", sigLen,
			minSigLen)
		return scriptError(ErrSigTooShort, str)
	}
	if sigLen > maxSigLen {
		str := fmt.Sprintf("malformed signature: too long: %d > %d", sigLen,
			maxSigLen)
		return scriptError(ErrSigTooLong, str)
	}

//签名必须以ASN.1序列标识符开始。
	if sig[sequenceOffset] != asn1SequenceID {
		str := fmt.Sprintf("malformed signature: format has wrong type: %#x",
			sig[sequenceOffset])
		return scriptError(ErrSigInvalidSeqID, str)
	}

//签名必须指示所有元素的正确数据量。
//与R和S有关。
	if int(sig[dataLenOffset]) != sigLen-2 {
		str := fmt.Sprintf("malformed signature: bad length: %d != %d",
			sig[dataLenOffset], sigLen-2)
		return scriptError(ErrSigInvalidDataLen, str)
	}

//计算与s相关的元素的偏移量并确保s在内部
//签名。
//
//rlen指定big-endian编码的数字的长度，
//表示签名的r值。
//
//stypeoffset是s的asn.1标识符的偏移量，与r类似
//对应项，应指示asn.1整数。
//
//slenoffset和soffset是签名中的字节偏移量
//s和s本身的长度。
	rLen := int(sig[rLenOffset])
	sTypeOffset := rOffset + rLen
	sLenOffset := sTypeOffset + 1
	if sTypeOffset >= sigLen {
		str := "malformed signature: S type indicator missing"
		return scriptError(ErrSigMissingSTypeID, str)
	}
	if sLenOffset >= sigLen {
		str := "malformed signature: S length missing"
		return scriptError(ErrSigMissingSLen, str)
	}

//R和S的长度必须与签名的总长度匹配。
//
//slen指定big endian编码的数字的长度
//表示签名的s值。
	sOffset := sLenOffset + 1
	sLen := int(sig[sLenOffset])
	if sOffset+sLen != sigLen {
		str := "malformed signature: invalid S length"
		return scriptError(ErrSigInvalidSLen, str)
	}

//r元素必须是asn.1整数。
	if sig[rTypeOffset] != asn1IntegerID {
		str := fmt.Sprintf("malformed signature: R integer marker: %#x != %#x",
			sig[rTypeOffset], asn1IntegerID)
		return scriptError(ErrSigInvalidRIntID, str)
	}

//r不允许使用零长度整数。
	if rLen == 0 {
		str := "malformed signature: R length is zero"
		return scriptError(ErrSigZeroRLen, str)
	}

//R不能为负。
	if sig[rOffset]&0x80 != 0 {
		str := "malformed signature: R is negative"
		return scriptError(ErrSigNegativeR, str)
	}

//不允许在r开头有空字节，除非r
//解释为负数。
	if rLen > 1 && sig[rOffset] == 0x00 && sig[rOffset+1]&0x80 == 0 {
		str := "malformed signature: R value has too much padding"
		return scriptError(ErrSigTooMuchRPadding, str)
	}

//s元素必须是asn.1整数。
	if sig[sTypeOffset] != asn1IntegerID {
		str := fmt.Sprintf("malformed signature: S integer marker: %#x != %#x",
			sig[sTypeOffset], asn1IntegerID)
		return scriptError(ErrSigInvalidSIntID, str)
	}

//s不允许使用零长度整数。
	if sLen == 0 {
		str := "malformed signature: S length is zero"
		return scriptError(ErrSigZeroSLen, str)
	}

//S不能为负。
	if sig[sOffset]&0x80 != 0 {
		str := "malformed signature: S is negative"
		return scriptError(ErrSigNegativeS, str)
	}

//不允许在s开头使用空字节，除非s
//解释为负数。
	if sLen > 1 && sig[sOffset] == 0x00 && sig[sOffset+1]&0x80 == 0 {
		str := "malformed signature: S value has too much padding"
		return scriptError(ErrSigTooMuchSPadding, str)
	}

//验证S值<=曲线阶数的一半。这张支票办完了
//因为当它更高时，可以使用顺序的补码模
//相反，这是一个较短的1字节编码。更进一步，没有
//执行此操作时，可以在有效的
//具有补码的事务，但仍然是有效的签名
//验证。这将导致更改事务哈希，因此
//延展性的来源。
	if vm.hasFlag(ScriptVerifyLowS) {
		sValue := new(big.Int).SetBytes(sig[sOffset : sOffset+sLen])
		if sValue.Cmp(halfOrder) > 0 {
			return scriptError(ErrSigHighS, "signature is not canonical due "+
				"to unnecessarily high S value")
		}
	}

	return nil
}

//GetStack以字节数组的形式自下而上返回堆栈的内容
func getStack(stack *stack) [][]byte {
	array := make([][]byte, stack.Depth())
	for i := range array {
//PeekBytearry无法因溢出而失败，已检查
		array[len(array)-i-1], _ = stack.PeekByteArray(int32(i))
	}
	return array
}

//setstack将堆栈设置为数组的内容，其中最后一个项位于
//数组是堆栈中的第一项。
func setStack(stack *stack, data [][]byte) {
//这不能出错。只有错误适用于无效参数。
	_ = stack.DropN(stack.Depth())

	for i := range data {
		stack.PushByteArray(data[i])
	}
}

//GetStack以数组形式返回主堆栈的内容。何处
//数组中的最后一项是堆栈的顶部。
func (vm *Engine) GetStack() [][]byte {
	return getStack(&vm.dstack)
}

//setstack将主堆栈的内容设置为
//提供的数组，其中数组中的最后一项将是堆栈的顶部。
func (vm *Engine) SetStack(data [][]byte) {
	setStack(&vm.dstack, data)
}

//GetAltStack以数组的形式返回备用堆栈的内容，其中
//数组中的最后一项是堆栈的顶部。
func (vm *Engine) GetAltStack() [][]byte {
	return getStack(&vm.astack)
}

//setAltStack将备用堆栈的内容设置为
//提供的数组，其中数组中的最后一项将是堆栈的顶部。
func (vm *Engine) SetAltStack(data [][]byte) {
	setStack(&vm.astack, data)
}

//new engine为提供的公钥脚本返回新的脚本引擎，
//事务和输入索引。这些标志修改脚本的行为
//发动机根据每个标志提供的说明。
func NewEngine(scriptPubKey []byte, tx *wire.MsgTx, txIdx int, flags ScriptFlags,
	sigCache *SigCache, hashCache *TxSigHashes, inputAmount int64) (*Engine, error) {

//提供的事务输入索引必须引用有效的输入。
	if txIdx < 0 || txIdx >= len(tx.TxIn) {
		str := fmt.Sprintf("transaction input index %d is negative or "+
			">= %d", txIdx, len(tx.TxIn))
		return nil, scriptError(ErrInvalidIndex, str)
	}
	scriptSig := tx.TxIn[txIdx].SignatureScript

//当签名脚本和公钥脚本都为空时，
//结果必然是一个错误，因为堆栈最终会
//空，相当于假top元素。所以，就回来吧
//相关的错误现在作为优化。
	if len(scriptSig) == 0 && len(scriptPubKey) == 0 {
		return nil, scriptError(ErrEvalFalse,
			"false stack entry at end of script execution")
	}

//不允许使用clean stack标志（scriptVerifyCleanStack），除非
//付费脚本哈希（p2sh）评估（scriptBip16）
//标志或隔离见证（scriptverifywitness）标志。
//
//回想一下，在没有标志集的情况下评估p2sh脚本会导致
//将p2sh输入留在堆栈上的非p2sh评估。
//因此，允许没有p2sh标志的clean stack标志将使
//可能会出现p2sh不是软叉的情况
//应该是什么时候。Segwit也会有同样的情况
//用于从见证堆栈执行的其他脚本。
	vm := Engine{flags: flags, sigCache: sigCache, hashCache: hashCache,
		inputAmount: inputAmount}
	if vm.hasFlag(ScriptVerifyCleanStack) && (!vm.hasFlag(ScriptBip16) &&
		!vm.hasFlag(ScriptVerifyWitness)) {
		return nil, scriptError(ErrInvalidFlags,
			"invalid flags combination")
	}

//签名脚本只能包含在
//已设置关联标志。
	if vm.hasFlag(ScriptVerifySigPushOnly) && !IsPushOnlyScript(scriptSig) {
		return nil, scriptError(ErrNotPushOnly,
			"signature script is not push only")
	}

//引擎使用一个切片以解析的形式存储脚本。这个
//允许按顺序执行多个脚本。例如，
//通过付费脚本哈希事务，最终将
//要执行的第三个脚本。
	scripts := [][]byte{scriptSig, scriptPubKey}
	vm.scripts = make([][]parsedOpcode, len(scripts))
	for i, scr := range scripts {
		if len(scr) > MaxScriptSize {
			str := fmt.Sprintf("script size %d is larger than max "+
				"allowed size %d", len(scr), MaxScriptSize)
			return nil, scriptError(ErrScriptTooBig, str)
		}
		var err error
		vm.scripts[i], err = parseScript(scr)
		if err != nil {
			return nil, err
		}
	}

//如果签名
//脚本为空，因为其中没有要执行的内容
//案例。
	if len(scripts[0]) == 0 {
		vm.scriptIdx++
	}

	if vm.hasFlag(ScriptBip16) && isScriptHash(vm.scripts[1]) {
//只接受为p2sh推送数据的输入脚本。
		if !isPushOnly(vm.scripts[0]) {
			return nil, scriptError(ErrNotPushOnly,
				"pay to script hash is not push only")
		}
		vm.bip16 = true
	}
	if vm.hasFlag(ScriptVerifyMinimalData) {
		vm.dstack.verifyMinimalData = true
		vm.astack.verifyMinimalData = true
	}

//检查是否应在见证验证模式下执行
//根据设置的标志。我们检查pkscript和sigscript
//这里，因为在嵌套p2sh的情况下，scriptsig将是有效的
//见证程序。对于嵌套p2sh，第一个数据之后的所有字节
//push应该*完全*匹配见证程序模板。
	if vm.hasFlag(ScriptVerifyWitness) {
//如果启用见证评估，则p2sh也必须
//主动的。
		if !vm.hasFlag(ScriptBip16) {
			errStr := "P2SH must be enabled to do witness verification"
			return nil, scriptError(ErrInvalidFlags, errStr)
		}

		var witProgram []byte

		switch {
		case isWitnessProgram(vm.scripts[1]):
//对于所有本机见证，scriptsig必须为*空*。
//程序，否则我们引入延展性。
			if len(scriptSig) != 0 {
				errStr := "native witness program cannot " +
					"also have a signature script"
				return nil, scriptError(ErrWitnessMalleated, errStr)
			}

			witProgram = scriptPubKey
		case len(tx.TxIn[txIdx].Witness) != 0 && vm.bip16:
//sigscript必须*精确*单个规范
//见证程序的数据推送，否则我们
//重新引入延展性。
			sigPops := vm.scripts[0]
			if len(sigPops) == 1 && canonicalPush(sigPops[0]) &&
				IsWitnessProgram(sigPops[0].data) {

				witProgram = sigPops[0].data
			} else {
				errStr := "signature script for witness " +
					"nested p2sh is not canonical"
				return nil, scriptError(ErrWitnessMalleatedP2SH, errStr)
			}
		}

		if witProgram != nil {
			var err error
			vm.witnessVersion, vm.witnessProgram, err = ExtractWitnessProgramInfo(witProgram)
			if err != nil {
				return nil, err
			}
		} else {
//如果我们在
//pkscript或作为sigscript中的数据推送，然后
//不得有任何与
//正在验证的输入。
			if vm.witnessProgram == nil && len(tx.TxIn[txIdx].Witness) != 0 {
				errStr := "non-witness inputs cannot have a witness"
				return nil, scriptError(ErrWitnessUnexpected, errStr)
			}
		}

	}

	vm.tx = *tx
	vm.txIdx = txIdx

	return &vm, nil
}
