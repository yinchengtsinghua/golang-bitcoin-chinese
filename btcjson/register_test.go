
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2014 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package btcjson_test

import (
	"reflect"
	"sort"
	"testing"

	"github.com/btcsuite/btcd/btcjson"
)

//testuageFlagstringer测试usageFlag类型的字符串化输出。
func TestUsageFlagStringer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   btcjson.UsageFlag
		want string
	}{
		{0, "0x0"},
		{btcjson.UFWalletOnly, "UFWalletOnly"},
		{btcjson.UFWebsocketOnly, "UFWebsocketOnly"},
		{btcjson.UFNotification, "UFNotification"},
		{btcjson.UFWalletOnly | btcjson.UFWebsocketOnly,
			"UFWalletOnly|UFWebsocketOnly"},
		{btcjson.UFWalletOnly | btcjson.UFWebsocketOnly | (1 << 31),
			"UFWalletOnly|UFWebsocketOnly|0x80000000"},
	}

//检测没有添加字符串的其他用法标志。
	numUsageFlags := 0
	highestUsageFlagBit := btcjson.TstHighestUsageFlagBit
	for highestUsageFlagBit > 1 {
		numUsageFlags++
		highestUsageFlagBit >>= 1
	}
	if len(tests)-3 != numUsageFlags {
		t.Errorf("It appears a usage flag was added without adding " +
			"an associated stringer test")
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

//testregisterCmdErrors确保registerCmd函数返回预期的
//提供的类型无效时出错。
func TestRegisterCmdErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		method  string
		cmdFunc func() interface{}
		flags   btcjson.UsageFlag
		err     btcjson.Error
	}{
		{
			name:   "duplicate method",
			method: "getblock",
			cmdFunc: func() interface{} {
				return struct{}{}
			},
			err: btcjson.Error{ErrorCode: btcjson.ErrDuplicateMethod},
		},
		{
			name:   "invalid usage flags",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				return 0
			},
			flags: btcjson.TstHighestUsageFlagBit,
			err:   btcjson.Error{ErrorCode: btcjson.ErrInvalidUsageFlags},
		},
		{
			name:   "invalid type",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				return 0
			},
			err: btcjson.Error{ErrorCode: btcjson.ErrInvalidType},
		},
		{
			name:   "invalid type 2",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				return &[]string{}
			},
			err: btcjson.Error{ErrorCode: btcjson.ErrInvalidType},
		},
		{
			name:   "embedded field",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				type test struct{ int }
				return (*test)(nil)
			},
			err: btcjson.Error{ErrorCode: btcjson.ErrEmbeddedType},
		},
		{
			name:   "unexported field",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				type test struct{ a int }
				return (*test)(nil)
			},
			err: btcjson.Error{ErrorCode: btcjson.ErrUnexportedField},
		},
		{
			name:   "unsupported field type 1",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				type test struct{ A **int }
				return (*test)(nil)
			},
			err: btcjson.Error{ErrorCode: btcjson.ErrUnsupportedFieldType},
		},
		{
			name:   "unsupported field type 2",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				type test struct{ A chan int }
				return (*test)(nil)
			},
			err: btcjson.Error{ErrorCode: btcjson.ErrUnsupportedFieldType},
		},
		{
			name:   "unsupported field type 3",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				type test struct{ A complex64 }
				return (*test)(nil)
			},
			err: btcjson.Error{ErrorCode: btcjson.ErrUnsupportedFieldType},
		},
		{
			name:   "unsupported field type 4",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				type test struct{ A complex128 }
				return (*test)(nil)
			},
			err: btcjson.Error{ErrorCode: btcjson.ErrUnsupportedFieldType},
		},
		{
			name:   "unsupported field type 5",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				type test struct{ A func() }
				return (*test)(nil)
			},
			err: btcjson.Error{ErrorCode: btcjson.ErrUnsupportedFieldType},
		},
		{
			name:   "unsupported field type 6",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				type test struct{ A interface{} }
				return (*test)(nil)
			},
			err: btcjson.Error{ErrorCode: btcjson.ErrUnsupportedFieldType},
		},
		{
			name:   "required after optional",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				type test struct {
					A *int
					B int
				}
				return (*test)(nil)
			},
			err: btcjson.Error{ErrorCode: btcjson.ErrNonOptionalField},
		},
		{
			name:   "non-optional with default",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				type test struct {
					A int `jsonrpcdefault:"1"`
				}
				return (*test)(nil)
			},
			err: btcjson.Error{ErrorCode: btcjson.ErrNonOptionalDefault},
		},
		{
			name:   "mismatched default",
			method: "registertestcmd",
			cmdFunc: func() interface{} {
				type test struct {
					A *int `jsonrpcdefault:"1.7"`
				}
				return (*test)(nil)
			},
			err: btcjson.Error{ErrorCode: btcjson.ErrMismatchedDefault},
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		err := btcjson.RegisterCmd(test.method, test.cmdFunc(),
			test.flags)
		if reflect.TypeOf(err) != reflect.TypeOf(test.err) {
			t.Errorf("Test #%d (%s) wrong error - got %T, "+
				"want %T", i, test.name, err, test.err)
			continue
		}
		gotErrorCode := err.(btcjson.Error).ErrorCode
		if gotErrorCode != test.err.ErrorCode {
			t.Errorf("Test #%d (%s) mismatched error code - got "+
				"%v, want %v", i, test.name, gotErrorCode,
				test.err.ErrorCode)
			continue
		}
	}
}

//testmustregisterCmdPanic确保mustregisterCmd函数在
//用于注册无效类型。
func TestMustRegisterCmdPanic(t *testing.T) {
	t.Parallel()

//设置延迟以捕捉预期的恐慌，以确保
//泛冰的
	defer func() {
		if err := recover(); err == nil {
			t.Error("MustRegisterCmd did not panic as expected")
		}
	}()

//故意尝试注册一个无效类型以强制恐慌。
	btcjson.MustRegisterCmd("panicme", 0, 0)
}

//TestRegisteredCmdMethods测试RegisteredCmdMethods函数确保
//按预期工作。
func TestRegisteredCmdMethods(t *testing.T) {
	t.Parallel()

//确保返回已注册的方法。
	methods := btcjson.RegisteredCmdMethods()
	if len(methods) == 0 {
		t.Fatal("RegisteredCmdMethods: no methods")
	}

//确保对返回的方法进行排序。
	sortedMethods := make([]string, len(methods))
	copy(sortedMethods, methods)
	sort.Sort(sort.StringSlice(sortedMethods))
	if !reflect.DeepEqual(sortedMethods, methods) {
		t.Fatal("RegisteredCmdMethods: methods are not sorted")
	}
}
