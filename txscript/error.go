
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package txscript

import (
	"fmt"
)

//错误代码标识一种脚本错误。
type ErrorCode int

//这些常量用于标识特定的错误。
const (
//如果内部一致性检查失败，则返回errInternal。在
//实践这一错误不应被视为意味着
//引擎逻辑错误。
	ErrInternal ErrorCode = iota

//——————————————————————————————————
//与API使用不当相关的失败。
//——————————————————————————————————

//传递给newengine的标志时返回errInvalidFlags
//包含无效的组合。
	ErrInvalidFlags

//当向传递越界索引时返回errInvalidIndex
//函数。
	ErrInvalidIndex

//当具体类型
//实现Bcutil。地址不是受支持的类型。
	ErrUnsupportedAddress

//当
//提供的脚本不是multisig脚本。
	ErrNotMultisigScript

//当
//指定的必需签名数大于
//提供了公钥。
	ErrTooManyRequiredSigs

//当长度为
//提供的数据超过了MaxDatacarriersize。
	ErrTooMuchNullData

//——————————————————————————————————
//与最终执行状态相关的失败。
//——————————————————————————————————

//在脚本中执行opu-return时返回errrearlyreturn。
	ErrEarlyReturn

//当脚本计算无误时返回errEmptyStack，
//但以空的顶级堆栈元素结尾。
	ErrEmptyStack

//当脚本无错误地计算时返回errevalse，但
//以错误的顶级堆栈元素终止。
	ErrEvalFalse

//调用checkErrorCondition时返回errScriptUnfinished
//尚未完成执行的脚本。
	ErrScriptUnfinished

//当尝试执行操作码时返回errscriptdone
//一旦全部执行完毕。这可能发生
//由于诸如第二次调用来执行或调用之后的步骤
//所有操作码都已执行。
	ErrInvalidProgramCounter

//————————————————————————————————————————————————
//与超过最大允许限值相关的故障。
//————————————————————————————————————————————————

//如果脚本大于maxscriptsize，则返回errscripttoobig。
	ErrScriptTooBig

//如果要推送的元素的大小，则返回errElementTooBig
//堆栈超过了MaxScriptElementSize。
	ErrElementTooBig

//如果脚本的值大于
//不推送数据的maxopsperscript操作码。
	ErrTooManyOperations

//当stack和altstack组合深度时返回errstackoverflow
//超出限制。
	ErrStackOverflow

//当公钥数为
//为multsig指定的值为负或大于
//最大键数
	ErrInvalidPubKeyCount

//当签名数为
//为multisig指定的值为负或大于
//公钥的数目。
	ErrInvalidSignatureCount

//当操作码的参数
//预期的数字输入大于预期的最大值
//字节。在大多数情况下，处理堆栈操作的操作码
//通过偏移、算术、数字比较和布尔逻辑
//这些都适用。但是，任何需要数字的操作码
//此代码可能导致输入失败。
	ErrNumberTooBig

//———————————————————————————————————————
//与验证操作相关的失败。
//———————————————————————————————————————

//当在脚本中遇到op_verify并且
//数据堆栈上的顶级项的计算结果不为true。
	ErrVerify

//当在
//脚本和数据堆栈上的顶级项的计算结果不为true。
	ErrEqualVerify

//遇到op_numEqualVerify时返回errNumEqualVerify
//在脚本中，数据堆栈上的顶级项的计算结果不为
//真的。
	ErrNumEqualVerify

//遇到op_checksigverify时返回errchecksigverify
//在脚本中，数据堆栈上的顶级项的计算结果不为
//真的。
	ErrCheckSigVerify

//当op_checkmultisigverify为
//在脚本中遇到，而数据堆栈上的顶级项没有
//计算为真。
	ErrCheckMultiSigVerify

//———————————————————————————————————————
//与操作码使用不当有关的故障。
//———————————————————————————————————————

//遇到禁用的操作码时返回errDisabledOpcode
//在脚本中。
	ErrDisabledOpcode

//当操作码标记为保留时返回errReservedOpcode
//在脚本中遇到。
	ErrReservedOpcode

//当数据推送操作码尝试推送时返回errMalformedPush
//比脚本中剩余的字节多。
	ErrMalformedPush

//当堆栈操作为
//尝试使用对当前堆栈大小无效的数字。
	ErrInvalidStackOperation

//当op-else或op-endif为
//在脚本中遇到，但未首先具有op-if或op-notif或
//到达脚本结尾时没有遇到op-endif
//以前遇到过op-if或op-notif。
	ErrUnbalancedConditional

//----------------------
//与延展性有关的失效。
//----------------------

//当scriptVerifyMinimalData标志时返回errMinimalData
//已设置，并且脚本包含不使用的推送操作
//所需的最小操作码。
	ErrMinimalData

//当签名哈希类型不是
//支持的类型之一。
	ErrInvalidSigHashType

//当签名应为
//规范编码的der签名太短。
	ErrSigTooShort

//当签名应为
//规范编码的der签名太长。
	ErrSigTooLong

//当签名应为
//规范编码的DER签名没有预期的ASN.1
//序列ID。
	ErrSigInvalidSeqID

//返回的签名应为
//规范编码的der签名未指定正确的数字
//R和S部分的剩余字节数。
	ErrSigInvalidDataLen

//返回的签名应为
//规范编码的der签名不提供ASN.1类型ID
//为S
	ErrSigMissingSTypeID

//当签名应为
//规范编码的der签名不提供s的长度。
	ErrSigMissingSLen

//返回的签名应为
//规范编码的der签名未指定正确的数字
//s部分的字节数。
	ErrSigInvalidSLen

//当签名应为
//规范编码的DER签名没有预期的ASN.1
//R的整数ID。
	ErrSigInvalidRIntID

//当签名应为
//规范编码的der签名的r长度为零。
	ErrSigZeroRLen

//当签名应为
//规范编码的der签名的r值为负值。
	ErrSigNegativeR

//当签名应为
//规范编码的der签名对r填充太多。
	ErrSigTooMuchRPadding

//当签名应为
//规范编码的DER签名没有预期的ASN.1
//S的整数ID。
	ErrSigInvalidSIntID

//当签名应为
//规范编码的der签名的s长度为零。
	ErrSigZeroSLen

//当签名应为
//规范编码的der签名的s值为负值。
	ErrSigNegativeS

//当签名应为
//规范编码的der签名对s填充太多。
	ErrSigTooMuchSPadding

//当设置了scriptVerifyLows标志并且
//脚本包含S值高于
//半序。
	ErrSigHighS

//当只需要
//将数据推送到堆栈执行其他操作。几个案子
//在这种情况下，当
//bip16处于活动状态，并且当设置了scriptVerifySigPushOnly标志时。
	ErrNotPushOnly

//设置scriptstrictmultisig标志时返回errsignuldummy
//一个multisig脚本除了0之外还有其他的虚拟对象
//争论。
	ErrSigNullDummy

//当ScriptVerifyStricteEncoding
//已设置标志，并且脚本包含无效的公钥。
	ErrPubKeyType

//当scriptVerifyCleanStack标志时返回errCleanStack
//已设置，并且在计算后，堆栈不仅包含
//单一元素。
	ErrCleanStack

//当scriptVerifyNullFail标志为
//在失败的checksig或checkmultisig上设置和签名不为空
//操作。
	ErrNullFail

//如果设置了scriptVerifyWitness并且
//遇到具有非空sigscript的本机p2wsh程序。
	ErrWitnessMalleated

//如果设置了scriptverifywitness，则返回errwitnessmalleatedp2sh
//嵌套p2sh的验证逻辑遇到一个sigscript
//这不是证人程序的数据推送。
	ErrWitnessMalleatedP2SH

//---------------------------
//与软叉相关的故障。
//---------------------------

//当
//已设置scriptUnrecordablenops标志，nop操作码为
//在脚本中遇到。
	ErrDiscourageUpgradableNOPs

//当脚本包含操作码
//解释负锁定时间。
	ErrNegativeLockTime

//当脚本包含操作码时，返回errUnsuccessiedLockTime
//这涉及到锁定时间，但所需的锁定时间尚未
//达到。
	ErrUnsatisfiedLockTime

//如果设置了scriptVerifyWitness并且
//op-if/op-nof-if的操作数不是空向量或
//[0x01]。
	ErrMinimalIf

//如果
//设置脚本验证见证，并设置执行见证的Versino
//程序在当前定义的见证程序集之外
//水泡。
	ErrDiscourageUpgradableWitnessProgram

//——————————————————————————————————
//与隔离证人有关的故障。
//——————————————————————————————————

//如果设置了scriptVerifyWitness并且
//见证堆栈本身为空。
	ErrWitnessProgramEmpty

//如果设置了scriptVerifyWitness，则返回errWitnessProgrammisMatch
//p2wkh证人计划的证人本身并不是
//项目或p2wsh的证人不是证人的sha255
//脚本。
	ErrWitnessProgramMismatch

//如果scriptVerifyWitness为
//设置并且见证程序的长度与长度冲突为
//由当前见证版本决定。
	ErrWitnessProgramWrongLength

//如果设置了scriptVerifyWitness并且
//事务包括见证数据，但不花费
//见证程序（嵌套或本机）。
	ErrWitnessUnexpected

//如果设置了scriptVerifyWitness并且
//在check sig或check multi sig中使用的公钥不是
//以压缩格式序列化。
	ErrWitnessPubKeyType

//numerorcodes是测试中使用的最大错误代码数。这个
//条目必须是枚举中的最后一个条目。
	numErrorCodes
)

//将错误代码值映射回其常量名，以便进行漂亮的打印。
var errorCodeStrings = map[ErrorCode]string{
	ErrInternal:                           "ErrInternal",
	ErrInvalidFlags:                       "ErrInvalidFlags",
	ErrInvalidIndex:                       "ErrInvalidIndex",
	ErrUnsupportedAddress:                 "ErrUnsupportedAddress",
	ErrNotMultisigScript:                  "ErrNotMultisigScript",
	ErrTooManyRequiredSigs:                "ErrTooManyRequiredSigs",
	ErrTooMuchNullData:                    "ErrTooMuchNullData",
	ErrEarlyReturn:                        "ErrEarlyReturn",
	ErrEmptyStack:                         "ErrEmptyStack",
	ErrEvalFalse:                          "ErrEvalFalse",
	ErrScriptUnfinished:                   "ErrScriptUnfinished",
	ErrInvalidProgramCounter:              "ErrInvalidProgramCounter",
	ErrScriptTooBig:                       "ErrScriptTooBig",
	ErrElementTooBig:                      "ErrElementTooBig",
	ErrTooManyOperations:                  "ErrTooManyOperations",
	ErrStackOverflow:                      "ErrStackOverflow",
	ErrInvalidPubKeyCount:                 "ErrInvalidPubKeyCount",
	ErrInvalidSignatureCount:              "ErrInvalidSignatureCount",
	ErrNumberTooBig:                       "ErrNumberTooBig",
	ErrVerify:                             "ErrVerify",
	ErrEqualVerify:                        "ErrEqualVerify",
	ErrNumEqualVerify:                     "ErrNumEqualVerify",
	ErrCheckSigVerify:                     "ErrCheckSigVerify",
	ErrCheckMultiSigVerify:                "ErrCheckMultiSigVerify",
	ErrDisabledOpcode:                     "ErrDisabledOpcode",
	ErrReservedOpcode:                     "ErrReservedOpcode",
	ErrMalformedPush:                      "ErrMalformedPush",
	ErrInvalidStackOperation:              "ErrInvalidStackOperation",
	ErrUnbalancedConditional:              "ErrUnbalancedConditional",
	ErrMinimalData:                        "ErrMinimalData",
	ErrInvalidSigHashType:                 "ErrInvalidSigHashType",
	ErrSigTooShort:                        "ErrSigTooShort",
	ErrSigTooLong:                         "ErrSigTooLong",
	ErrSigInvalidSeqID:                    "ErrSigInvalidSeqID",
	ErrSigInvalidDataLen:                  "ErrSigInvalidDataLen",
	ErrSigMissingSTypeID:                  "ErrSigMissingSTypeID",
	ErrSigMissingSLen:                     "ErrSigMissingSLen",
	ErrSigInvalidSLen:                     "ErrSigInvalidSLen",
	ErrSigInvalidRIntID:                   "ErrSigInvalidRIntID",
	ErrSigZeroRLen:                        "ErrSigZeroRLen",
	ErrSigNegativeR:                       "ErrSigNegativeR",
	ErrSigTooMuchRPadding:                 "ErrSigTooMuchRPadding",
	ErrSigInvalidSIntID:                   "ErrSigInvalidSIntID",
	ErrSigZeroSLen:                        "ErrSigZeroSLen",
	ErrSigNegativeS:                       "ErrSigNegativeS",
	ErrSigTooMuchSPadding:                 "ErrSigTooMuchSPadding",
	ErrSigHighS:                           "ErrSigHighS",
	ErrNotPushOnly:                        "ErrNotPushOnly",
	ErrSigNullDummy:                       "ErrSigNullDummy",
	ErrPubKeyType:                         "ErrPubKeyType",
	ErrCleanStack:                         "ErrCleanStack",
	ErrNullFail:                           "ErrNullFail",
	ErrDiscourageUpgradableNOPs:           "ErrDiscourageUpgradableNOPs",
	ErrNegativeLockTime:                   "ErrNegativeLockTime",
	ErrUnsatisfiedLockTime:                "ErrUnsatisfiedLockTime",
	ErrWitnessProgramEmpty:                "ErrWitnessProgramEmpty",
	ErrWitnessProgramMismatch:             "ErrWitnessProgramMismatch",
	ErrWitnessProgramWrongLength:          "ErrWitnessProgramWrongLength",
	ErrWitnessMalleated:                   "ErrWitnessMalleated",
	ErrWitnessMalleatedP2SH:               "ErrWitnessMalleatedP2SH",
	ErrWitnessUnexpected:                  "ErrWitnessUnexpected",
	ErrMinimalIf:                          "ErrMinimalIf",
	ErrWitnessPubKeyType:                  "ErrWitnessPubKeyType",
	ErrDiscourageUpgradableWitnessProgram: "ErrDiscourageUpgradableWitnessProgram",
}

//字符串将错误代码返回为人类可读的名称。
func (e ErrorCode) String() string {
	if s := errorCodeStrings[e]; s != "" {
		return s
	}
	return fmt.Sprintf("Unknown ErrorCode (%d)", int(e))
}

//错误标识与脚本相关的错误。它用来表示三个
//错误类别：
//1）由于违反许多要求之一而导致脚本执行失败
//由脚本引擎强制执行或评估为假
//2）调用方使用API不当
//3）内部一致性检查失败
//
//调用方可以对返回的错误使用类型断言来访问
//用于确定错误的特定原因的错误代码字段。作为一个
//为方便起见，调用方可以使用ISerrocode函数
//检查特定的错误代码。
type Error struct {
	ErrorCode   ErrorCode
	Description string
}

//错误满足错误接口并打印人类可读的错误。
func (e Error) Error() string {
	return e.Description
}

//ScriptError在给定一组参数的情况下创建错误。
func scriptError(c ErrorCode, desc string) Error {
	return Error{ErrorCode: c, Description: desc}
}

//ISerrorCode返回所提供的错误是否为脚本错误，
//提供的错误代码。
func IsErrorCode(err error, c ErrorCode) bool {
	serr, ok := err.(Error)
	return ok && serr.ErrorCode == c
}
