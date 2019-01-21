
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
	"testing"

	"github.com/btcsuite/btcd/btcjson"
)

//testhelpers测试创建指向
//基本类型。
func TestHelpers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		f        func() interface{}
		expected interface{}
	}{
		{
			name: "bool",
			f: func() interface{} {
				return btcjson.Bool(true)
			},
			expected: func() interface{} {
				val := true
				return &val
			}(),
		},
		{
			name: "int",
			f: func() interface{} {
				return btcjson.Int(5)
			},
			expected: func() interface{} {
				val := int(5)
				return &val
			}(),
		},
		{
			name: "uint",
			f: func() interface{} {
				return btcjson.Uint(5)
			},
			expected: func() interface{} {
				val := uint(5)
				return &val
			}(),
		},
		{
			name: "int32",
			f: func() interface{} {
				return btcjson.Int32(5)
			},
			expected: func() interface{} {
				val := int32(5)
				return &val
			}(),
		},
		{
			name: "uint32",
			f: func() interface{} {
				return btcjson.Uint32(5)
			},
			expected: func() interface{} {
				val := uint32(5)
				return &val
			}(),
		},
		{
			name: "int64",
			f: func() interface{} {
				return btcjson.Int64(5)
			},
			expected: func() interface{} {
				val := int64(5)
				return &val
			}(),
		},
		{
			name: "uint64",
			f: func() interface{} {
				return btcjson.Uint64(5)
			},
			expected: func() interface{} {
				val := uint64(5)
				return &val
			}(),
		},
		{
			name: "string",
			f: func() interface{} {
				return btcjson.String("abc")
			},
			expected: func() interface{} {
				val := "abc"
				return &val
			}(),
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		result := test.f()
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("Test #%d (%s) unexpected value - got %v, "+
				"want %v", i, test.name, result, test.expected)
			continue
		}
	}
}
