
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

package database_test

import (
	"errors"
	"testing"

	"github.com/btcsuite/btcd/database"
)

//TesterRorCodeStringer测试错误代码类型的字符串化输出。
func TestErrorCodeStringer(t *testing.T) {
	tests := []struct {
		in   database.ErrorCode
		want string
	}{
		{database.ErrDbTypeRegistered, "ErrDbTypeRegistered"},
		{database.ErrDbUnknownType, "ErrDbUnknownType"},
		{database.ErrDbDoesNotExist, "ErrDbDoesNotExist"},
		{database.ErrDbExists, "ErrDbExists"},
		{database.ErrDbNotOpen, "ErrDbNotOpen"},
		{database.ErrDbAlreadyOpen, "ErrDbAlreadyOpen"},
		{database.ErrInvalid, "ErrInvalid"},
		{database.ErrCorruption, "ErrCorruption"},
		{database.ErrTxClosed, "ErrTxClosed"},
		{database.ErrTxNotWritable, "ErrTxNotWritable"},
		{database.ErrBucketNotFound, "ErrBucketNotFound"},
		{database.ErrBucketExists, "ErrBucketExists"},
		{database.ErrBucketNameRequired, "ErrBucketNameRequired"},
		{database.ErrKeyRequired, "ErrKeyRequired"},
		{database.ErrKeyTooLarge, "ErrKeyTooLarge"},
		{database.ErrValueTooLarge, "ErrValueTooLarge"},
		{database.ErrIncompatibleValue, "ErrIncompatibleValue"},
		{database.ErrBlockNotFound, "ErrBlockNotFound"},
		{database.ErrBlockExists, "ErrBlockExists"},
		{database.ErrBlockRegionInvalid, "ErrBlockRegionInvalid"},
		{database.ErrDriverSpecific, "ErrDriverSpecific"},

		{0xffff, "Unknown ErrorCode (65535)"},
	}

//检测没有添加字符串的其他错误代码。
	if len(tests)-1 != int(database.TstNumErrorCodes) {
		t.Errorf("It appears an error code was added without adding " +
			"an associated stringer test")
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		result := test.in.String()
		if result != test.want {
			t.Errorf("String #%d\ngot: %s\nwant: %s", i, result,
				test.want)
			continue
		}
	}
}

//TesterRor测试错误类型的错误输出。
func TestError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   database.Error
		want string
	}{
		{
			database.Error{Description: "some error"},
			"some error",
		},
		{
			database.Error{Description: "human-readable error"},
			"human-readable error",
		},
		{
			database.Error{
				ErrorCode:   database.ErrDriverSpecific,
				Description: "some error",
				Err:         errors.New("driver-specific error"),
			},
			"some error: driver-specific error",
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
