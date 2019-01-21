
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2014-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package blockchain

import (
	"fmt"
)

//DeploymentError标识一个错误，该错误指示部署ID为
//
type DeploymentError uint32

//错误将断言错误作为可读字符串返回并满足
//错误接口。
func (e DeploymentError) Error() string {
	return fmt.Sprintf("deployment ID %d does not exist", uint32(e))
}

//断言错误标识指示内部代码一致性的错误
//问题，并应被视为一个关键和不可恢复的错误。
type AssertError string

//错误将断言错误作为可读字符串返回并满足
//错误接口。
func (e AssertError) Error() string {
	return "assertion failed: " + string(e)
}

//错误代码标识一种错误。
type ErrorCode int

//这些常量用于标识特定的RuleError。
const (
//errDuplicateBlock指示已经具有相同哈希的块
//存在。
	ErrDuplicateBlock ErrorCode = iota

//errblocktoobig指示序列化块大小超过
//最大允许大小。
	ErrBlockTooBig

//errblockweighttoohigh表示块的计算重量
//度量值超过了允许的最大值。
	ErrBlockWeightTooHigh

//errblockversiontooold指示块版本太旧，并且
//由于大部分网络已升级，不再接受
//更新的版本。
	ErrBlockVersionTooOld

//errInvalidTime表示传递的块中的时间具有精度
//超过一秒钟。链共识规则要求
//时间戳的最大精度为1秒。
	ErrInvalidTime

//errTimeTooold表示时间早于
//每个链的最后几个块共识规则或之前
//最近的检查点。
	ErrTimeTooOld

//errTimeToOnew表示与之相比，未来时间太远
//当前时间。
	ErrTimeTooNew

//errDifficultyToolow表示块的难度较低
//比最近一次检查站要求的难度大。
	ErrDifficultyTooLow

//errUnexpectedDifficulty表示指定的位与不对齐
//预期值，因为它与计算值不匹配
//根据难度重新获得的规则进行估价，或超出有效范围
//范围。
	ErrUnexpectedDifficulty

//
//低于要求的目标很难。
	ErrHighHash

//errbadmerkleroot表示计算的merkle根不匹配
//预期值。
	ErrBadMerkleRoot

//errbadcheckpoint指示预期位于
//检查点高度与预期高度不匹配。
	ErrBadCheckpoint

//errfooktoold表示块正试图分叉块链
//在最近的检查点之前。
	ErrForkTooOld

//errCheckpointTimeTooold指示块在
//最近的检查点。
	ErrCheckpointTimeTooOld

//errnotTransactions指示块没有一个
//交易。有效块必须至少具有coinbase
//交易。
	ErrNoTransactions

//errnotxinputs表示事务没有任何输入。一
//有效事务必须至少有一个输入。
	ErrNoTxInputs

//errnotxOutputs表示事务没有任何输出。一
//有效事务必须至少有一个输出。
	ErrNoTxOutputs

//errtxttoobig表示事务超出了允许的最大大小
//序列化时。
	ErrTxTooBig

//errbadtxoutvalue表示事务的输出值为
//在某些方面无效，例如超出范围。
	ErrBadTxOutValue

//errDuplicatetXinPuts表示事务引用相同
//多次输入。
	ErrDuplicateTxInputs

//errbadtxinput表示事务输入在某种程度上无效
//例如引用上一个超出
//范围或根本不引用一个。
	ErrBadTxInput

//errMissingXout表示由输入引用的事务输出
//要么不存在，要么已经用完了。
	ErrMissingTxOut

//errunfinalizedtx表示交易尚未完成。
//有效块只能包含已完成的事务。
	ErrUnfinalizedTx

//errDuplicateTx指示块包含相同的事务
//（或至少两个哈希值相同的事务）。一
//有效块只能包含唯一事务。
	ErrDuplicateTx

//erroverwritetx表示块包含的事务
//与上一个尚未完全完成的事务相同的哈希
//花了。
	ErrOverwriteTx

//ErrUndulizeSpend表示事务正试图花费
//尚未达到所需期限的CoinBase。
	ErrImmatureSpend

//errspendtoohigh表示事务正试图花费更多
//值大于其所有输入的总和。
	ErrSpendTooHigh

//ErrBadFees表示块的总费用因以下原因无效：
//超过最大可能值。
	ErrBadFees

//errtoomanysigops表示签名操作的总数
//对于事务或块，超出了允许的最大限制。
	ErrTooManySigOps

//errFirstTxNotCoinBase指示块中的第一个事务
//不是CoinBase事务。
	ErrFirstTxNotCoinbase

//errMultipleIntercases表示一个块包含多个
//CoinBase交易。
	ErrMultipleCoinbases

//errbadCoinBaseScriptlen指示签名脚本的长度
//因为CoinBase事务不在有效范围内。
	ErrBadCoinbaseScriptLen

//errbadCoinBaseValue指示CoinBase值的数量
//不符合补贴的预期价值加上所有费用的总和。
	ErrBadCoinbaseValue

//errMissingCoinBaseHeight指示
//块不是以序列化块高度作为开始
//版本2和更高版本块需要。
	ErrMissingCoinbaseHeight

//errbadCoinBaseHeight指示
//版本2和更高版本块的CoinBase事务不匹配
//预期值。
	ErrBadCoinbaseHeight

//errscriptMalformed表示中的事务脚本格式不正确
//某种方式。例如，它可能比允许的最大值长
//长度或分析失败。
	ErrScriptMalformed

//errscriptValidation指示执行事务的结果
//脚本失败。错误包括执行脚本时的任何失败
//这样的签名验证失败并在
//堆栈。
	ErrScriptValidation

//errUnexpectedWitness指示块包含事务
//有证人数据，但没有证人承诺
//CoinBase交易记录。
	ErrUnexpectedWitness

//ErrInvalidWitnessCommitment表示一个街区的证人
//
	ErrInvalidWitnessCommitment

//错误见证承诺不匹配表示见证承诺
//包含在块的CoinBase事务中与
//人工计算的见证承诺。
	ErrWitnessCommitmentMismatch

//errPreviousBlockUnknown表示上一个块未知。
	ErrPreviousBlockUnknown

//errInvalidancestorBlock指示此块的祖先具有
//验证已失败。
	ErrInvalidAncestorBlock

//errPrevBlockNotBest指示块的上一个块不是
//当前链尖。这不是块验证规则，但是必需的
//对于通过getblocktemplate rpc提交的块建议。
	ErrPrevBlockNotBest
)

//将错误代码值映射回其常量名，以便进行漂亮的打印。
var errorCodeStrings = map[ErrorCode]string{
	ErrDuplicateBlock:            "ErrDuplicateBlock",
	ErrBlockTooBig:               "ErrBlockTooBig",
	ErrBlockVersionTooOld:        "ErrBlockVersionTooOld",
	ErrBlockWeightTooHigh:        "ErrBlockWeightTooHigh",
	ErrInvalidTime:               "ErrInvalidTime",
	ErrTimeTooOld:                "ErrTimeTooOld",
	ErrTimeTooNew:                "ErrTimeTooNew",
	ErrDifficultyTooLow:          "ErrDifficultyTooLow",
	ErrUnexpectedDifficulty:      "ErrUnexpectedDifficulty",
	ErrHighHash:                  "ErrHighHash",
	ErrBadMerkleRoot:             "ErrBadMerkleRoot",
	ErrBadCheckpoint:             "ErrBadCheckpoint",
	ErrForkTooOld:                "ErrForkTooOld",
	ErrCheckpointTimeTooOld:      "ErrCheckpointTimeTooOld",
	ErrNoTransactions:            "ErrNoTransactions",
	ErrNoTxInputs:                "ErrNoTxInputs",
	ErrNoTxOutputs:               "ErrNoTxOutputs",
	ErrTxTooBig:                  "ErrTxTooBig",
	ErrBadTxOutValue:             "ErrBadTxOutValue",
	ErrDuplicateTxInputs:         "ErrDuplicateTxInputs",
	ErrBadTxInput:                "ErrBadTxInput",
	ErrMissingTxOut:              "ErrMissingTxOut",
	ErrUnfinalizedTx:             "ErrUnfinalizedTx",
	ErrDuplicateTx:               "ErrDuplicateTx",
	ErrOverwriteTx:               "ErrOverwriteTx",
	ErrImmatureSpend:             "ErrImmatureSpend",
	ErrSpendTooHigh:              "ErrSpendTooHigh",
	ErrBadFees:                   "ErrBadFees",
	ErrTooManySigOps:             "ErrTooManySigOps",
	ErrFirstTxNotCoinbase:        "ErrFirstTxNotCoinbase",
	ErrMultipleCoinbases:         "ErrMultipleCoinbases",
	ErrBadCoinbaseScriptLen:      "ErrBadCoinbaseScriptLen",
	ErrBadCoinbaseValue:          "ErrBadCoinbaseValue",
	ErrMissingCoinbaseHeight:     "ErrMissingCoinbaseHeight",
	ErrBadCoinbaseHeight:         "ErrBadCoinbaseHeight",
	ErrScriptMalformed:           "ErrScriptMalformed",
	ErrScriptValidation:          "ErrScriptValidation",
	ErrUnexpectedWitness:         "ErrUnexpectedWitness",
	ErrInvalidWitnessCommitment:  "ErrInvalidWitnessCommitment",
	ErrWitnessCommitmentMismatch: "ErrWitnessCommitmentMismatch",
	ErrPreviousBlockUnknown:      "ErrPreviousBlockUnknown",
	ErrInvalidAncestorBlock:      "ErrInvalidAncestorBlock",
	ErrPrevBlockNotBest:          "ErrPrevBlockNotBest",
}

//字符串将错误代码返回为人类可读的名称。
func (e ErrorCode) String() string {
	if s := errorCodeStrings[e]; s != "" {
		return s
	}
	return fmt.Sprintf("Unknown ErrorCode (%d)", int(e))
}

//RuleError标识规则冲突。用来表示
//由于许多验证之一，块或事务处理失败
//规则。调用方可以使用类型断言来确定失败是否是
//特别是由于违反规则，访问错误代码字段
//确定违反规则的具体原因。
type RuleError struct {
ErrorCode   ErrorCode //描述错误的类型
Description string    //问题的人类可读描述
}

//错误满足错误接口并打印人类可读的错误。
func (e RuleError) Error() string {
	return e.Description
}

//RuleError在给定一组参数的情况下创建RuleError。
func ruleError(c ErrorCode, desc string) RuleError {
	return RuleError{ErrorCode: c, Description: desc}
}
