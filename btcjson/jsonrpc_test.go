
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
	"encoding/json"
	"reflect"
	"testing"

	"github.com/btcsuite/btcd/btcjson"
)

//testisvalididType确保isvalididType函数的行为符合预期。
func TestIsValidIDType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		id      interface{}
		isValid bool
	}{
		{"int", int(1), true},
		{"int8", int8(1), true},
		{"int16", int16(1), true},
		{"int32", int32(1), true},
		{"int64", int64(1), true},
		{"uint", uint(1), true},
		{"uint8", uint8(1), true},
		{"uint16", uint16(1), true},
		{"uint32", uint32(1), true},
		{"uint64", uint64(1), true},
		{"string", "1", true},
		{"nil", nil, true},
		{"float32", float32(1), true},
		{"float64", float64(1), true},
		{"bool", true, false},
		{"chan int", make(chan int), false},
		{"complex64", complex64(1), false},
		{"complex128", complex128(1), false},
		{"func", func() {}, false},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		if btcjson.IsValidIDType(test.id) != test.isValid {
			t.Errorf("Test #%d (%s) valid mismatch - got %v, "+
				"want %v", i, test.name, !test.isValid,
				test.isValid)
			continue
		}
	}
}

//TestMarshalResponse确保MarshalResponse函数按预期工作。
func TestMarshalResponse(t *testing.T) {
	t.Parallel()

	testID := 1
	tests := []struct {
		name     string
		result   interface{}
		jsonErr  *btcjson.RPCError
		expected []byte
	}{
		{
			name:     "ordinary bool result with no error",
			result:   true,
			jsonErr:  nil,
			expected: []byte(`{"result":true,"error":null,"id":1}`),
		},
		{
			name:   "result with error",
			result: nil,
			jsonErr: func() *btcjson.RPCError {
				return btcjson.NewRPCError(btcjson.ErrRPCBlockNotFound, "123 not found")
			}(),
			expected: []byte(`{"result":null,"error":{"code":-5,"message":"123 not found"},"id":1}`),
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		_, _ = i, test
		marshalled, err := btcjson.MarshalResponse(testID, test.result, test.jsonErr)
		if err != nil {
			t.Errorf("Test #%d (%s) unexpected error: %v", i,
				test.name, err)
			continue
		}

		if !reflect.DeepEqual(marshalled, test.expected) {
			t.Errorf("Test #%d (%s) mismatched result - got %s, "+
				"want %s", i, test.name, marshalled,
				test.expected)
		}
	}
}

//TestMiscErrors测试其他地方没有涉及的一些错误条件。
func TestMiscErrors(t *testing.T) {
	t.Parallel()

//通过为newRequest指定参数类型来强制执行错误
//不支持。
	_, err := btcjson.NewRequest(nil, "test", []interface{}{make(chan int)})
	if err == nil {
		t.Error("NewRequest: did not receive error")
		return
	}

//通过给MarshalResponse中的ID类型
//支持。
	wantErr := btcjson.Error{ErrorCode: btcjson.ErrInvalidType}
	_, err = btcjson.MarshalResponse(make(chan int), nil, nil)
	if jerr, ok := err.(btcjson.Error); !ok || jerr.ErrorCode != wantErr.ErrorCode {
		t.Errorf("MarshalResult: did not receive expected error - got "+
			"%v (%[1]T), want %v (%[2]T)", err, wantErr)
		return
	}

//通过给MarshalResponse中的结果类型
//无法整理。
	_, err = btcjson.MarshalResponse(1, make(chan int), nil)
	if _, ok := err.(*json.UnsupportedTypeError); !ok {
		wantErr := &json.UnsupportedTypeError{}
		t.Errorf("MarshalResult: did not receive expected error - got "+
			"%v (%[1]T), want %T", err, wantErr)
		return
	}
}

//TestRpcError测试RpcError类型的错误输出。
func TestRPCError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   *btcjson.RPCError
		want string
	}{
		{
			btcjson.ErrRPCInvalidRequest,
			"-32600: Invalid request",
		},
		{
			btcjson.ErrRPCMethodNotFound,
			"-32601: Method not found",
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
