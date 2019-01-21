
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package txscript

import (
	"testing"
)

//TesterRorCodeStringer测试错误代码类型的字符串化输出。
func TestErrorCodeStringer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   ErrorCode
		want string
	}{
		{ErrInternal, "ErrInternal"},
		{ErrInvalidFlags, "ErrInvalidFlags"},
		{ErrInvalidIndex, "ErrInvalidIndex"},
		{ErrUnsupportedAddress, "ErrUnsupportedAddress"},
		{ErrTooManyRequiredSigs, "ErrTooManyRequiredSigs"},
		{ErrTooMuchNullData, "ErrTooMuchNullData"},
		{ErrNotMultisigScript, "ErrNotMultisigScript"},
		{ErrEarlyReturn, "ErrEarlyReturn"},
		{ErrEmptyStack, "ErrEmptyStack"},
		{ErrEvalFalse, "ErrEvalFalse"},
		{ErrScriptUnfinished, "ErrScriptUnfinished"},
		{ErrInvalidProgramCounter, "ErrInvalidProgramCounter"},
		{ErrScriptTooBig, "ErrScriptTooBig"},
		{ErrElementTooBig, "ErrElementTooBig"},
		{ErrTooManyOperations, "ErrTooManyOperations"},
		{ErrStackOverflow, "ErrStackOverflow"},
		{ErrInvalidPubKeyCount, "ErrInvalidPubKeyCount"},
		{ErrInvalidSignatureCount, "ErrInvalidSignatureCount"},
		{ErrNumberTooBig, "ErrNumberTooBig"},
		{ErrVerify, "ErrVerify"},
		{ErrEqualVerify, "ErrEqualVerify"},
		{ErrNumEqualVerify, "ErrNumEqualVerify"},
		{ErrCheckSigVerify, "ErrCheckSigVerify"},
		{ErrCheckMultiSigVerify, "ErrCheckMultiSigVerify"},
		{ErrDisabledOpcode, "ErrDisabledOpcode"},
		{ErrReservedOpcode, "ErrReservedOpcode"},
		{ErrMalformedPush, "ErrMalformedPush"},
		{ErrInvalidStackOperation, "ErrInvalidStackOperation"},
		{ErrUnbalancedConditional, "ErrUnbalancedConditional"},
		{ErrMinimalData, "ErrMinimalData"},
		{ErrInvalidSigHashType, "ErrInvalidSigHashType"},
		{ErrSigTooShort, "ErrSigTooShort"},
		{ErrSigTooLong, "ErrSigTooLong"},
		{ErrSigInvalidSeqID, "ErrSigInvalidSeqID"},
		{ErrSigInvalidDataLen, "ErrSigInvalidDataLen"},
		{ErrSigMissingSTypeID, "ErrSigMissingSTypeID"},
		{ErrSigMissingSLen, "ErrSigMissingSLen"},
		{ErrSigInvalidSLen, "ErrSigInvalidSLen"},
		{ErrSigInvalidRIntID, "ErrSigInvalidRIntID"},
		{ErrSigZeroRLen, "ErrSigZeroRLen"},
		{ErrSigNegativeR, "ErrSigNegativeR"},
		{ErrSigTooMuchRPadding, "ErrSigTooMuchRPadding"},
		{ErrSigInvalidSIntID, "ErrSigInvalidSIntID"},
		{ErrSigZeroSLen, "ErrSigZeroSLen"},
		{ErrSigNegativeS, "ErrSigNegativeS"},
		{ErrSigTooMuchSPadding, "ErrSigTooMuchSPadding"},
		{ErrSigHighS, "ErrSigHighS"},
		{ErrNotPushOnly, "ErrNotPushOnly"},
		{ErrSigNullDummy, "ErrSigNullDummy"},
		{ErrPubKeyType, "ErrPubKeyType"},
		{ErrCleanStack, "ErrCleanStack"},
		{ErrNullFail, "ErrNullFail"},
		{ErrDiscourageUpgradableNOPs, "ErrDiscourageUpgradableNOPs"},
		{ErrNegativeLockTime, "ErrNegativeLockTime"},
		{ErrUnsatisfiedLockTime, "ErrUnsatisfiedLockTime"},
		{ErrWitnessProgramEmpty, "ErrWitnessProgramEmpty"},
		{ErrWitnessProgramMismatch, "ErrWitnessProgramMismatch"},
		{ErrWitnessProgramWrongLength, "ErrWitnessProgramWrongLength"},
		{ErrWitnessMalleated, "ErrWitnessMalleated"},
		{ErrWitnessMalleatedP2SH, "ErrWitnessMalleatedP2SH"},
		{ErrWitnessUnexpected, "ErrWitnessUnexpected"},
		{ErrMinimalIf, "ErrMinimalIf"},
		{ErrWitnessPubKeyType, "ErrWitnessPubKeyType"},
		{ErrDiscourageUpgradableWitnessProgram, "ErrDiscourageUpgradableWitnessProgram"},
		{0xffff, "Unknown ErrorCode (65535)"},
	}

//检测没有添加字符串的其他错误代码。
	if len(tests)-1 != int(numErrorCodes) {
		t.Errorf("It appears an error code was added without adding an " +
			"associated stringer test")
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

//TesterRor测试错误类型的错误输出。
func TestError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   Error
		want string
	}{
		{
			Error{Description: "some error"},
			"some error",
		},
		{
			Error{Description: "human-readable error"},
			"human-readable error",
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		result := test.in.Error()
		if result != test.want {
			t.Errorf("Error #%d\n got: %s want: %s", i, result,
				test.want)
			continue
		}
	}
}
