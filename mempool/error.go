
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

package mempool

import (
	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/wire"
)

//RuleError标识规则冲突。用来表示
//由于许多验证之一，事务处理失败
//规则。调用方可以使用类型断言来确定失败是否是
//特别是由于违反规则，使用err字段访问
//基础错误，它将是txruleerror或
//区块链。规则错误。
type RuleError struct {
	Err error
}

//错误满足错误接口并打印人类可读的错误。
func (e RuleError) Error() string {
	if e.Err == nil {
		return "<nil>"
	}
	return e.Err.Error()
}

//txRuleError标识规则违规。它用来表示
//由于许多验证之一，事务处理失败
//规则。调用方可以使用类型断言来确定失败是否是
//特别是由于违反规则，访问错误代码字段
//确定违反规则的具体原因。
type TxRuleError struct {
RejectCode  wire.RejectCode //与拒绝消息一起发送的代码
Description string          //问题的人类可读描述
}

//错误满足错误接口并打印人类可读的错误。
func (e TxRuleError) Error() string {
	return e.Description
}

//TxRuleError使用给定的一组
//参数并返回一个封装它的RuleError。
func txRuleError(c wire.RejectCode, desc string) RuleError {
	return RuleError{
		Err: TxRuleError{RejectCode: c, Description: desc},
	}
}

//ChanReuleError返回一个封装给定的规则错误
//区块链。规则错误。
func chainRuleError(chainErr blockchain.RuleError) RuleError {
	return RuleError{
		Err: chainErr,
	}
}

//ExtractRejectCode尝试返回给定错误的相关拒绝代码
//通过检查已知类型的错误。如果一个代码
//已成功提取。
func extractRejectCode(err error) (wire.RejectCode, bool) {
//从RuleError中提取基础错误。
	if rerr, ok := err.(RuleError); ok {
		err = rerr.Err
	}

	switch err := err.(type) {
	case blockchain.RuleError:
//将链错误转换为拒绝代码。
		var code wire.RejectCode
		switch err.ErrorCode {
//因重复而被拒绝。
		case blockchain.ErrDuplicateBlock:
			code = wire.RejectDuplicate

//因版本过时而被拒绝。
		case blockchain.ErrBlockVersionTooOld:
			code = wire.RejectObsolete

//由于检查点拒绝。
		case blockchain.ErrCheckpointTimeTooOld:
			fallthrough
		case blockchain.ErrDifficultyTooLow:
			fallthrough
		case blockchain.ErrBadCheckpoint:
			fallthrough
		case blockchain.ErrForkTooOld:
			code = wire.RejectCheckpoint

//其他一切都是由于块或事务无效。
		default:
			code = wire.RejectInvalid
		}

		return code, true

	case TxRuleError:
		return err.RejectCode, true

	case nil:
		return wire.RejectInvalid, false
	}

	return wire.RejectInvalid, false
}

//errtorejecterr检查错误的基础类型并返回拒绝
//适合在Wire.msgreject消息中发送的代码和字符串。
func ErrToRejectErr(err error) (wire.RejectCode, string) {
//如果可能的话，将拒绝代码与错误文本一起返回。
//从错误中提取。
	rejectCode, found := extractRejectCode(err)
	if found {
		return rejectCode, err.Error()
	}

//如果没有错误，则返回一般拒绝字符串。这真的
//除非其他地方的代码没有设置错误，否则不应发生
//应该是这样，但最好是安全的，并且只返回一个泛型
//字符串，而不允许以下代码取消引用
//害怕恐慌。
	if err == nil {
		return wire.RejectInvalid, "rejected"
	}

//如果基础错误不是上述情况之一，则返回
//WIRE.REJECTINVALID，包含一个被拒绝的通用字符串和错误
//文本。
	return wire.RejectInvalid, "rejected: " + err.Error()
}
