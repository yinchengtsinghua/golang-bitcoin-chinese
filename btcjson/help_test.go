
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

//testHelpReflectInternals确保处理
//反射类型按预期工作于各种go类型。
func TestHelpReflectInternals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		reflectType reflect.Type
		indentLevel int
		key         string
		examples    []string
		isComplex   bool
		help        string
		isInvalid   bool
	}{
		{
			name:        "int",
			reflectType: reflect.TypeOf(int(0)),
			key:         "json-type-numeric",
			examples:    []string{"n"},
			help:        "n (json-type-numeric) fdk",
		},
		{
			name:        "*int",
			reflectType: reflect.TypeOf((*int)(nil)),
			key:         "json-type-value",
			examples:    []string{"n"},
			help:        "n (json-type-value) fdk",
			isInvalid:   true,
		},
		{
			name:        "int8",
			reflectType: reflect.TypeOf(int8(0)),
			key:         "json-type-numeric",
			examples:    []string{"n"},
			help:        "n (json-type-numeric) fdk",
		},
		{
			name:        "int16",
			reflectType: reflect.TypeOf(int16(0)),
			key:         "json-type-numeric",
			examples:    []string{"n"},
			help:        "n (json-type-numeric) fdk",
		},
		{
			name:        "int32",
			reflectType: reflect.TypeOf(int32(0)),
			key:         "json-type-numeric",
			examples:    []string{"n"},
			help:        "n (json-type-numeric) fdk",
		},
		{
			name:        "int64",
			reflectType: reflect.TypeOf(int64(0)),
			key:         "json-type-numeric",
			examples:    []string{"n"},
			help:        "n (json-type-numeric) fdk",
		},
		{
			name:        "uint",
			reflectType: reflect.TypeOf(uint(0)),
			key:         "json-type-numeric",
			examples:    []string{"n"},
			help:        "n (json-type-numeric) fdk",
		},
		{
			name:        "uint8",
			reflectType: reflect.TypeOf(uint8(0)),
			key:         "json-type-numeric",
			examples:    []string{"n"},
			help:        "n (json-type-numeric) fdk",
		},
		{
			name:        "uint16",
			reflectType: reflect.TypeOf(uint16(0)),
			key:         "json-type-numeric",
			examples:    []string{"n"},
			help:        "n (json-type-numeric) fdk",
		},
		{
			name:        "uint32",
			reflectType: reflect.TypeOf(uint32(0)),
			key:         "json-type-numeric",
			examples:    []string{"n"},
			help:        "n (json-type-numeric) fdk",
		},
		{
			name:        "uint64",
			reflectType: reflect.TypeOf(uint64(0)),
			key:         "json-type-numeric",
			examples:    []string{"n"},
			help:        "n (json-type-numeric) fdk",
		},
		{
			name:        "float32",
			reflectType: reflect.TypeOf(float32(0)),
			key:         "json-type-numeric",
			examples:    []string{"n.nnn"},
			help:        "n.nnn (json-type-numeric) fdk",
		},
		{
			name:        "float64",
			reflectType: reflect.TypeOf(float64(0)),
			key:         "json-type-numeric",
			examples:    []string{"n.nnn"},
			help:        "n.nnn (json-type-numeric) fdk",
		},
		{
			name:        "string",
			reflectType: reflect.TypeOf(""),
			key:         "json-type-string",
			examples:    []string{`"json-example-string"`},
			help:        "\"json-example-string\" (json-type-string) fdk",
		},
		{
			name:        "bool",
			reflectType: reflect.TypeOf(true),
			key:         "json-type-bool",
			examples:    []string{"json-example-bool"},
			help:        "json-example-bool (json-type-bool) fdk",
		},
		{
			name:        "array of int",
			reflectType: reflect.TypeOf([1]int{0}),
			key:         "json-type-arrayjson-type-numeric",
			examples:    []string{"[n,...]"},
			help:        "[n,...] (json-type-arrayjson-type-numeric) fdk",
		},
		{
			name:        "slice of int",
			reflectType: reflect.TypeOf([]int{0}),
			key:         "json-type-arrayjson-type-numeric",
			examples:    []string{"[n,...]"},
			help:        "[n,...] (json-type-arrayjson-type-numeric) fdk",
		},
		{
			name:        "struct",
			reflectType: reflect.TypeOf(struct{}{}),
			key:         "json-type-object",
			examples:    []string{"{", "}\t\t"},
			isComplex:   true,
			help:        "{\n} ",
		},
		{
			name:        "struct indent level 1",
			reflectType: reflect.TypeOf(struct{ field int }{}),
			indentLevel: 1,
			key:         "json-type-object",
			examples: []string{
				"  \"field\": n,\t(json-type-numeric)\t-field",
				" },\t\t",
			},
			help: "{\n" +
				" \"field\": n, (json-type-numeric) -field\n" +
				"}            ",
			isComplex: true,
		},
		{
			name: "array of struct indent level 0",
			reflectType: func() reflect.Type {
				type s struct {
					field int
				}
				return reflect.TypeOf([]s{})
			}(),
			key: "json-type-arrayjson-type-object",
			examples: []string{
				"[{",
				" \"field\": n,\t(json-type-numeric)\ts-field",
				"},...]",
			},
			help: "[{\n" +
				" \"field\": n, (json-type-numeric) s-field\n" +
				"},...]",
			isComplex: true,
		},
		{
			name: "array of struct indent level 1",
			reflectType: func() reflect.Type {
				type s struct {
					field int
				}
				return reflect.TypeOf([]s{})
			}(),
			indentLevel: 1,
			key:         "json-type-arrayjson-type-object",
			examples: []string{
				"  \"field\": n,\t(json-type-numeric)\ts-field",
				" },...],\t\t",
			},
			help: "[{\n" +
				" \"field\": n, (json-type-numeric) s-field\n" +
				"},...]",
			isComplex: true,
		},
		{
			name:        "map",
			reflectType: reflect.TypeOf(map[string]string{}),
			key:         "json-type-object",
			examples: []string{"{",
				" \"fdk--key\": fdk--value, (json-type-object) fdk--desc",
				" ...", "}",
			},
			help: "{\n" +
				" \"fdk--key\": fdk--value, (json-type-object) fdk--desc\n" +
				" ...\n" +
				"}",
			isComplex: true,
		},
		{
			name:        "complex",
			reflectType: reflect.TypeOf(complex64(0)),
			key:         "json-type-value",
			examples:    []string{"json-example-unknown"},
			help:        "json-example-unknown (json-type-value) fdk",
			isInvalid:   true,
		},
	}

	xT := func(key string) string {
		return key
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//确保描述键为预期值。
		key := btcjson.TstReflectTypeToJSONType(xT, test.reflectType)
		if key != test.key {
			t.Errorf("Test #%d (%s) unexpected key - got: %v, "+
				"want: %v", i, test.name, key, test.key)
			continue
		}

//确保生成的示例符合预期。
		examples, isComplex := btcjson.TstReflectTypeToJSONExample(xT,
			test.reflectType, test.indentLevel, "fdk")
		if isComplex != test.isComplex {
			t.Errorf("Test #%d (%s) unexpected isComplex - got: %v, "+
				"want: %v", i, test.name, isComplex,
				test.isComplex)
			continue
		}
		if len(examples) != len(test.examples) {
			t.Errorf("Test #%d (%s) unexpected result length - "+
				"got: %v, want: %v", i, test.name, len(examples),
				len(test.examples))
			continue
		}
		for j, example := range examples {
			if example != test.examples[j] {
				t.Errorf("Test #%d (%s) example #%d unexpected "+
					"example - got: %v, want: %v", i,
					test.name, j, example, test.examples[j])
				continue
			}
		}

//确保生成的结果类型帮助与预期的一样。
		helpText := btcjson.TstResultTypeHelp(xT, test.reflectType, "fdk")
		if helpText != test.help {
			t.Errorf("Test #%d (%s) unexpected result help - "+
				"got: %v, want: %v", i, test.name, helpText,
				test.help)
			continue
		}

		isValid := btcjson.TstIsValidResultType(test.reflectType.Kind())
		if isValid != !test.isInvalid {
			t.Errorf("Test #%d (%s) unexpected result type validity "+
				"- got: %v", i, test.name, isValid)
			continue
		}
	}
}

//testRestructHelp确保返回预期的帮助文本格式
//各种go结构类型。
func TestResultStructHelp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		reflectType reflect.Type
		expected    []string
	}{
		{
			name: "empty struct",
			reflectType: func() reflect.Type {
				type s struct{}
				return reflect.TypeOf(s{})
			}(),
			expected: nil,
		},
		{
			name: "struct with primitive field",
			reflectType: func() reflect.Type {
				type s struct {
					field int
				}
				return reflect.TypeOf(s{})
			}(),
			expected: []string{
				"\"field\": n,\t(json-type-numeric)\ts-field",
			},
		},
		{
			name: "struct with primitive field and json tag",
			reflectType: func() reflect.Type {
				type s struct {
					Field int `json:"f"`
				}
				return reflect.TypeOf(s{})
			}(),
			expected: []string{
				"\"f\": n,\t(json-type-numeric)\ts-f",
			},
		},
		{
			name: "struct with array of primitive field",
			reflectType: func() reflect.Type {
				type s struct {
					field []int
				}
				return reflect.TypeOf(s{})
			}(),
			expected: []string{
				"\"field\": [n,...],\t(json-type-arrayjson-type-numeric)\ts-field",
			},
		},
		{
			name: "struct with sub-struct field",
			reflectType: func() reflect.Type {
				type s2 struct {
					subField int
				}
				type s struct {
					field s2
				}
				return reflect.TypeOf(s{})
			}(),
			expected: []string{
				"\"field\": {\t(json-type-object)\ts-field",
				"{",
				" \"subfield\": n,\t(json-type-numeric)\ts2-subfield",
				"}\t\t",
			},
		},
		{
			name: "struct with sub-struct field pointer",
			reflectType: func() reflect.Type {
				type s2 struct {
					subField int
				}
				type s struct {
					field *s2
				}
				return reflect.TypeOf(s{})
			}(),
			expected: []string{
				"\"field\": {\t(json-type-object)\ts-field",
				"{",
				" \"subfield\": n,\t(json-type-numeric)\ts2-subfield",
				"}\t\t",
			},
		},
		{
			name: "struct with array of structs field",
			reflectType: func() reflect.Type {
				type s2 struct {
					subField int
				}
				type s struct {
					field []s2
				}
				return reflect.TypeOf(s{})
			}(),
			expected: []string{
				"\"field\": [{\t(json-type-arrayjson-type-object)\ts-field",
				"[{",
				" \"subfield\": n,\t(json-type-numeric)\ts2-subfield",
				"},...]",
			},
		},
	}

	xT := func(key string) string {
		return key
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		results := btcjson.TstResultStructHelp(xT, test.reflectType, 0)
		if len(results) != len(test.expected) {
			t.Errorf("Test #%d (%s) unexpected result length - "+
				"got: %v, want: %v", i, test.name, len(results),
				len(test.expected))
			continue
		}
		for j, result := range results {
			if result != test.expected[j] {
				t.Errorf("Test #%d (%s) result #%d unexpected "+
					"result - got: %v, want: %v", i,
					test.name, j, result, test.expected[j])
				continue
			}
		}
	}
}

//testHelpArgInternals确保处理
//对于各种参数类型，参数的工作方式与预期一致。
func TestHelpArgInternals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		method      string
		reflectType reflect.Type
		defaults    map[int]reflect.Value
		help        string
	}{
		{
			name:   "command with no args",
			method: "test",
			reflectType: func() reflect.Type {
				type s struct{}
				return reflect.TypeOf((*s)(nil))
			}(),
			defaults: nil,
			help:     "",
		},
		{
			name:   "command with one required arg",
			method: "test",
			reflectType: func() reflect.Type {
				type s struct {
					Field int
				}
				return reflect.TypeOf((*s)(nil))
			}(),
			defaults: nil,
			help:     "1. field (json-type-numeric, help-required) test-field\n",
		},
		{
			name:   "command with one optional arg, no default",
			method: "test",
			reflectType: func() reflect.Type {
				type s struct {
					Optional *int
				}
				return reflect.TypeOf((*s)(nil))
			}(),
			defaults: nil,
			help:     "1. optional (json-type-numeric, help-optional) test-optional\n",
		},
		{
			name:   "command with one optional arg with default",
			method: "test",
			reflectType: func() reflect.Type {
				type s struct {
					Optional *string
				}
				return reflect.TypeOf((*s)(nil))
			}(),
			defaults: func() map[int]reflect.Value {
				defVal := "test"
				return map[int]reflect.Value{
					0: reflect.ValueOf(&defVal),
				}
			}(),
			help: "1. optional (json-type-string, help-optional, help-default=\"test\") test-optional\n",
		},
		{
			name:   "command with struct field",
			method: "test",
			reflectType: func() reflect.Type {
				type s2 struct {
					F int8
				}
				type s struct {
					Field s2
				}
				return reflect.TypeOf((*s)(nil))
			}(),
			defaults: nil,
			help: "1. field (json-type-object, help-required) test-field\n" +
				"{\n" +
				" \"f\": n, (json-type-numeric) s2-f\n" +
				"}        \n",
		},
		{
			name:   "command with map field",
			method: "test",
			reflectType: func() reflect.Type {
				type s struct {
					Field map[string]float64
				}
				return reflect.TypeOf((*s)(nil))
			}(),
			defaults: nil,
			help: "1. field (json-type-object, help-required) test-field\n" +
				"{\n" +
				" \"test-field--key\": test-field--value, (json-type-object) test-field--desc\n" +
				" ...\n" +
				"}\n",
		},
		{
			name:   "command with slice of primitives field",
			method: "test",
			reflectType: func() reflect.Type {
				type s struct {
					Field []int64
				}
				return reflect.TypeOf((*s)(nil))
			}(),
			defaults: nil,
			help:     "1. field (json-type-arrayjson-type-numeric, help-required) test-field\n",
		},
		{
			name:   "command with slice of structs field",
			method: "test",
			reflectType: func() reflect.Type {
				type s2 struct {
					F int64
				}
				type s struct {
					Field []s2
				}
				return reflect.TypeOf((*s)(nil))
			}(),
			defaults: nil,
			help: "1. field (json-type-arrayjson-type-object, help-required) test-field\n" +
				"[{\n" +
				" \"f\": n, (json-type-numeric) s2-f\n" +
				"},...]\n",
		},
	}

	xT := func(key string) string {
		return key
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		help := btcjson.TstArgHelp(xT, test.reflectType, test.defaults,
			test.method)
		if help != test.help {
			t.Errorf("Test #%d (%s) unexpected help - got:\n%v\n"+
				"want:\n%v", i, test.name, help, test.help)
			continue
		}
	}
}

//testmethodHelp确保方法帮助函数按预期工作
//命令结构。
func TestMethodHelp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		method      string
		reflectType reflect.Type
		defaults    map[int]reflect.Value
		resultTypes []interface{}
		help        string
	}{
		{
			name:   "command with no args or results",
			method: "test",
			reflectType: func() reflect.Type {
				type s struct{}
				return reflect.TypeOf((*s)(nil))
			}(),
			help: "test\n\ntest--synopsis\n\n" +
				"help-arguments:\nhelp-arguments-none\n\n" +
				"help-result:\nhelp-result-nothing\n",
		},
		{
			name:   "command with no args and one primitive result",
			method: "test",
			reflectType: func() reflect.Type {
				type s struct{}
				return reflect.TypeOf((*s)(nil))
			}(),
			resultTypes: []interface{}{(*int64)(nil)},
			help: "test\n\ntest--synopsis\n\n" +
				"help-arguments:\nhelp-arguments-none\n\n" +
				"help-result:\nn (json-type-numeric) test--result0\n",
		},
		{
			name:   "command with no args and two results",
			method: "test",
			reflectType: func() reflect.Type {
				type s struct{}
				return reflect.TypeOf((*s)(nil))
			}(),
			resultTypes: []interface{}{(*int64)(nil), nil},
			help: "test\n\ntest--synopsis\n\n" +
				"help-arguments:\nhelp-arguments-none\n\n" +
				"help-result (test--condition0):\nn (json-type-numeric) test--result0\n\n" +
				"help-result (test--condition1):\nhelp-result-nothing\n",
		},
		{
			name:   "command with primitive arg and no results",
			method: "test",
			reflectType: func() reflect.Type {
				type s struct {
					Field bool
				}
				return reflect.TypeOf((*s)(nil))
			}(),
			help: "test field\n\ntest--synopsis\n\n" +
				"help-arguments:\n1. field (json-type-bool, help-required) test-field\n\n" +
				"help-result:\nhelp-result-nothing\n",
		},
		{
			name:   "command with primitive optional and no results",
			method: "test",
			reflectType: func() reflect.Type {
				type s struct {
					Field *bool
				}
				return reflect.TypeOf((*s)(nil))
			}(),
			help: "test (field)\n\ntest--synopsis\n\n" +
				"help-arguments:\n1. field (json-type-bool, help-optional) test-field\n\n" +
				"help-result:\nhelp-result-nothing\n",
		},
	}

	xT := func(key string) string {
		return key
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		help := btcjson.TestMethodHelp(xT, test.reflectType,
			test.defaults, test.method, test.resultTypes)
		if help != test.help {
			t.Errorf("Test #%d (%s) unexpected help - got:\n%v\n"+
				"want:\n%v", i, test.name, help, test.help)
			continue
		}
	}
}

//TestGenerateHelpErrors确保GenerateHelp函数返回预期的
//错误。
func TestGenerateHelpErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		method      string
		resultTypes []interface{}
		err         btcjson.Error
	}{
		{
			name:   "unregistered command",
			method: "boguscommand",
			err:    btcjson.Error{ErrorCode: btcjson.ErrUnregisteredMethod},
		},
		{
			name:        "non-pointer result type",
			method:      "help",
			resultTypes: []interface{}{0},
			err:         btcjson.Error{ErrorCode: btcjson.ErrInvalidType},
		},
		{
			name:        "invalid result type",
			method:      "help",
			resultTypes: []interface{}{(*complex64)(nil)},
			err:         btcjson.Error{ErrorCode: btcjson.ErrInvalidType},
		},
		{
			name:        "missing description",
			method:      "help",
			resultTypes: []interface{}{(*string)(nil), nil},
			err:         btcjson.Error{ErrorCode: btcjson.ErrMissingDescription},
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		_, err := btcjson.GenerateHelp(test.method, nil,
			test.resultTypes...)
		if reflect.TypeOf(err) != reflect.TypeOf(test.err) {
			t.Errorf("Test #%d (%s) wrong error - got %T (%v), "+
				"want %T", i, test.name, err, err, test.err)
			continue
		}
		gotErrorCode := err.(btcjson.Error).ErrorCode
		if gotErrorCode != test.err.ErrorCode {
			t.Errorf("Test #%d (%s) mismatched error code - got "+
				"%v (%v), want %v", i, test.name, gotErrorCode,
				err, test.err.ErrorCode)
			continue
		}
	}
}

//testGenerateHelp执行一个非常基本的测试，以确保GenerateHelp正常工作。
//果不其然。内部测试在其他测试中更彻底，所以
//这里不需要添加更多的测试。
func TestGenerateHelp(t *testing.T) {
	t.Parallel()

	descs := map[string]string{
		"help--synopsis": "test",
		"help-command":   "test",
	}
	help, err := btcjson.GenerateHelp("help", descs)
	if err != nil {
		t.Fatalf("GenerateHelp: unexpected error: %v", err)
	}
	wantHelp := "help (\"command\")\n\n" +
		"test\n\nArguments:\n1. command (string, optional) test\n\n" +
		"Result:\nNothing\n"
	if help != wantHelp {
		t.Fatalf("GenerateHelp: unexpected help - got\n%v\nwant\n%v",
			help, wantHelp)
	}
}
