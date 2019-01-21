
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
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"testing"
)

//TSTCheckScriptError确保传递的两个错误的类型是
//同一类型（零或两种类型的错误）及其错误代码
//不为零时匹配。
func tstCheckScriptError(gotErr, wantErr error) error {
//确保错误代码是预期的类型，并且错误
//代码与测试实例中指定的值匹配。
	if reflect.TypeOf(gotErr) != reflect.TypeOf(wantErr) {
		return fmt.Errorf("wrong error - got %T (%[1]v), want %T",
			gotErr, wantErr)
	}
	if gotErr == nil {
		return nil
	}

//确保所需的错误类型是脚本错误。
	werr, ok := wantErr.(Error)
	if !ok {
		return fmt.Errorf("unexpected test error type %T", wantErr)
	}

//确保错误代码匹配。使用原始类型断言是安全的
//因为上面的代码已经证明了它们是同一类型并且
//所需错误是脚本错误。
	gotErrorCode := gotErr.(Error).ErrorCode
	if gotErrorCode != werr.ErrorCode {
		return fmt.Errorf("mismatched error code - got %v (%v), want %v",
			gotErrorCode, gotErr, werr.ErrorCode)
	}

	return nil
}

//测试堆栈测试所有堆栈操作是否按预期工作。
func TestStack(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		before    [][]byte
		operation func(*stack) error
		err       error
		after     [][]byte
	}{
		{
			"noop",
			[][]byte{{1}, {2}, {3}, {4}, {5}},
			func(s *stack) error {
				return nil
			},
			nil,
			[][]byte{{1}, {2}, {3}, {4}, {5}},
		},
		{
			"peek underflow (byte)",
			[][]byte{{1}, {2}, {3}, {4}, {5}},
			func(s *stack) error {
				_, err := s.PeekByteArray(5)
				return err
			},
			scriptError(ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"peek underflow (int)",
			[][]byte{{1}, {2}, {3}, {4}, {5}},
			func(s *stack) error {
				_, err := s.PeekInt(5)
				return err
			},
			scriptError(ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"peek underflow (bool)",
			[][]byte{{1}, {2}, {3}, {4}, {5}},
			func(s *stack) error {
				_, err := s.PeekBool(5)
				return err
			},
			scriptError(ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"pop",
			[][]byte{{1}, {2}, {3}, {4}, {5}},
			func(s *stack) error {
				val, err := s.PopByteArray()
				if err != nil {
					return err
				}
				if !bytes.Equal(val, []byte{5}) {
					return errors.New("not equal")
				}
				return err
			},
			nil,
			[][]byte{{1}, {2}, {3}, {4}},
		},
		{
			"pop everything",
			[][]byte{{1}, {2}, {3}, {4}, {5}},
			func(s *stack) error {
				for i := 0; i < 5; i++ {
					_, err := s.PopByteArray()
					if err != nil {
						return err
					}
				}
				return nil
			},
			nil,
			nil,
		},
		{
			"pop underflow",
			[][]byte{{1}, {2}, {3}, {4}, {5}},
			func(s *stack) error {
				for i := 0; i < 6; i++ {
					_, err := s.PopByteArray()
					if err != nil {
						return err
					}
				}
				return nil
			},
			scriptError(ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"pop bool",
			[][]byte{nil},
			func(s *stack) error {
				val, err := s.PopBool()
				if err != nil {
					return err
				}

				if val {
					return errors.New("unexpected value")
				}
				return nil
			},
			nil,
			nil,
		},
		{
			"pop bool",
			[][]byte{{1}},
			func(s *stack) error {
				val, err := s.PopBool()
				if err != nil {
					return err
				}

				if !val {
					return errors.New("unexpected value")
				}
				return nil
			},
			nil,
			nil,
		},
		{
			"pop bool",
			nil,
			func(s *stack) error {
				_, err := s.PopBool()
				return err
			},
			scriptError(ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"popInt 0",
			[][]byte{{0x0}},
			func(s *stack) error {
				v, err := s.PopInt()
				if err != nil {
					return err
				}
				if v != 0 {
					return errors.New("0 != 0 on popInt")
				}
				return nil
			},
			nil,
			nil,
		},
		{
			"popInt -0",
			[][]byte{{0x80}},
			func(s *stack) error {
				v, err := s.PopInt()
				if err != nil {
					return err
				}
				if v != 0 {
					return errors.New("-0 != 0 on popInt")
				}
				return nil
			},
			nil,
			nil,
		},
		{
			"popInt 1",
			[][]byte{{0x01}},
			func(s *stack) error {
				v, err := s.PopInt()
				if err != nil {
					return err
				}
				if v != 1 {
					return errors.New("1 != 1 on popInt")
				}
				return nil
			},
			nil,
			nil,
		},
		{
			"popInt 1 leading 0",
			[][]byte{{0x01, 0x00, 0x00, 0x00}},
			func(s *stack) error {
				v, err := s.PopInt()
				if err != nil {
					return err
				}
				if v != 1 {
					fmt.Printf("%v != %v\n", v, 1)
					return errors.New("1 != 1 on popInt")
				}
				return nil
			},
			nil,
			nil,
		},
		{
			"popInt -1",
			[][]byte{{0x81}},
			func(s *stack) error {
				v, err := s.PopInt()
				if err != nil {
					return err
				}
				if v != -1 {
					return errors.New("-1 != -1 on popInt")
				}
				return nil
			},
			nil,
			nil,
		},
		{
			"popInt -1 leading 0",
			[][]byte{{0x01, 0x00, 0x00, 0x80}},
			func(s *stack) error {
				v, err := s.PopInt()
				if err != nil {
					return err
				}
				if v != -1 {
					fmt.Printf("%v != %v\n", v, -1)
					return errors.New("-1 != -1 on popInt")
				}
				return nil
			},
			nil,
			nil,
		},
//在asint中触发多字节大小写
		{
			"popInt -513",
			[][]byte{{0x1, 0x82}},
			func(s *stack) error {
				v, err := s.PopInt()
				if err != nil {
					return err
				}
				if v != -513 {
					fmt.Printf("%v != %v\n", v, -513)
					return errors.New("1 != 1 on popInt")
				}
				return nil
			},
			nil,
			nil,
		},
//确认asint代码不会修改基础数据。
		{
			"peekint nomodify -1",
			[][]byte{{0x01, 0x00, 0x00, 0x80}},
			func(s *stack) error {
				v, err := s.PeekInt(0)
				if err != nil {
					return err
				}
				if v != -1 {
					fmt.Printf("%v != %v\n", v, -1)
					return errors.New("-1 != -1 on popInt")
				}
				return nil
			},
			nil,
			[][]byte{{0x01, 0x00, 0x00, 0x80}},
		},
		{
			"PushInt 0",
			nil,
			func(s *stack) error {
				s.PushInt(scriptNum(0))
				return nil
			},
			nil,
			[][]byte{{}},
		},
		{
			"PushInt 1",
			nil,
			func(s *stack) error {
				s.PushInt(scriptNum(1))
				return nil
			},
			nil,
			[][]byte{{0x1}},
		},
		{
			"PushInt -1",
			nil,
			func(s *stack) error {
				s.PushInt(scriptNum(-1))
				return nil
			},
			nil,
			[][]byte{{0x81}},
		},
		{
			"PushInt two bytes",
			nil,
			func(s *stack) error {
				s.PushInt(scriptNum(256))
				return nil
			},
			nil,
//小endian……叹息*
			[][]byte{{0x00, 0x01}},
		},
		{
			"PushInt leading zeros",
			nil,
			func(s *stack) error {
//这个会有高点的
				s.PushInt(scriptNum(128))
				return nil
			},
			nil,
			[][]byte{{0x80, 0x00}},
		},
		{
			"dup",
			[][]byte{{1}},
			func(s *stack) error {
				return s.DupN(1)
			},
			nil,
			[][]byte{{1}, {1}},
		},
		{
			"dup2",
			[][]byte{{1}, {2}},
			func(s *stack) error {
				return s.DupN(2)
			},
			nil,
			[][]byte{{1}, {2}, {1}, {2}},
		},
		{
			"dup3",
			[][]byte{{1}, {2}, {3}},
			func(s *stack) error {
				return s.DupN(3)
			},
			nil,
			[][]byte{{1}, {2}, {3}, {1}, {2}, {3}},
		},
		{
			"dup0",
			[][]byte{{1}},
			func(s *stack) error {
				return s.DupN(0)
			},
			scriptError(ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"dup-1",
			[][]byte{{1}},
			func(s *stack) error {
				return s.DupN(-1)
			},
			scriptError(ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"dup too much",
			[][]byte{{1}},
			func(s *stack) error {
				return s.DupN(2)
			},
			scriptError(ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"PushBool true",
			nil,
			func(s *stack) error {
				s.PushBool(true)

				return nil
			},
			nil,
			[][]byte{{1}},
		},
		{
			"PushBool false",
			nil,
			func(s *stack) error {
				s.PushBool(false)

				return nil
			},
			nil,
			[][]byte{nil},
		},
		{
			"PushBool PopBool",
			nil,
			func(s *stack) error {
				s.PushBool(true)
				val, err := s.PopBool()
				if err != nil {
					return err
				}
				if !val {
					return errors.New("unexpected value")
				}

				return nil
			},
			nil,
			nil,
		},
		{
			"PushBool PopBool 2",
			nil,
			func(s *stack) error {
				s.PushBool(false)
				val, err := s.PopBool()
				if err != nil {
					return err
				}
				if val {
					return errors.New("unexpected value")
				}

				return nil
			},
			nil,
			nil,
		},
		{
			"PushInt PopBool",
			nil,
			func(s *stack) error {
				s.PushInt(scriptNum(1))
				val, err := s.PopBool()
				if err != nil {
					return err
				}
				if !val {
					return errors.New("unexpected value")
				}

				return nil
			},
			nil,
			nil,
		},
		{
			"PushInt PopBool 2",
			nil,
			func(s *stack) error {
				s.PushInt(scriptNum(0))
				val, err := s.PopBool()
				if err != nil {
					return err
				}
				if val {
					return errors.New("unexpected value")
				}

				return nil
			},
			nil,
			nil,
		},
		{
			"Nip top",
			[][]byte{{1}, {2}, {3}},
			func(s *stack) error {
				return s.NipN(0)
			},
			nil,
			[][]byte{{1}, {2}},
		},
		{
			"Nip middle",
			[][]byte{{1}, {2}, {3}},
			func(s *stack) error {
				return s.NipN(1)
			},
			nil,
			[][]byte{{1}, {3}},
		},
		{
			"Nip low",
			[][]byte{{1}, {2}, {3}},
			func(s *stack) error {
				return s.NipN(2)
			},
			nil,
			[][]byte{{2}, {3}},
		},
		{
			"Nip too much",
			[][]byte{{1}, {2}, {3}},
			func(s *stack) error {
//咬掉的比我们能咀嚼的还多
				return s.NipN(3)
			},
			scriptError(ErrInvalidStackOperation, ""),
			[][]byte{{2}, {3}},
		},
		{
			"keep on tucking",
			[][]byte{{1}, {2}, {3}},
			func(s *stack) error {
				return s.Tuck()
			},
			nil,
			[][]byte{{1}, {3}, {2}, {3}},
		},
		{
			"a little tucked up",
[][]byte{{1}}, //塔克的论点太少了
			func(s *stack) error {
				return s.Tuck()
			},
			scriptError(ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"all tucked up",
nil, //塔克的论点太少了
			func(s *stack) error {
				return s.Tuck()
			},
			scriptError(ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"drop 1",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) error {
				return s.DropN(1)
			},
			nil,
			[][]byte{{1}, {2}, {3}},
		},
		{
			"drop 2",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) error {
				return s.DropN(2)
			},
			nil,
			[][]byte{{1}, {2}},
		},
		{
			"drop 3",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) error {
				return s.DropN(3)
			},
			nil,
			[][]byte{{1}},
		},
		{
			"drop 4",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) error {
				return s.DropN(4)
			},
			nil,
			nil,
		},
		{
			"drop 4/5",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) error {
				return s.DropN(5)
			},
			scriptError(ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"drop invalid",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) error {
				return s.DropN(0)
			},
			scriptError(ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"Rot1",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) error {
				return s.RotN(1)
			},
			nil,
			[][]byte{{1}, {3}, {4}, {2}},
		},
		{
			"Rot2",
			[][]byte{{1}, {2}, {3}, {4}, {5}, {6}},
			func(s *stack) error {
				return s.RotN(2)
			},
			nil,
			[][]byte{{3}, {4}, {5}, {6}, {1}, {2}},
		},
		{
			"Rot too little",
			[][]byte{{1}, {2}},
			func(s *stack) error {
				return s.RotN(1)
			},
			scriptError(ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"Rot0",
			[][]byte{{1}, {2}, {3}},
			func(s *stack) error {
				return s.RotN(0)
			},
			scriptError(ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"Swap1",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) error {
				return s.SwapN(1)
			},
			nil,
			[][]byte{{1}, {2}, {4}, {3}},
		},
		{
			"Swap2",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) error {
				return s.SwapN(2)
			},
			nil,
			[][]byte{{3}, {4}, {1}, {2}},
		},
		{
			"Swap too little",
			[][]byte{{1}},
			func(s *stack) error {
				return s.SwapN(1)
			},
			scriptError(ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"Swap0",
			[][]byte{{1}, {2}, {3}},
			func(s *stack) error {
				return s.SwapN(0)
			},
			scriptError(ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"Over1",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) error {
				return s.OverN(1)
			},
			nil,
			[][]byte{{1}, {2}, {3}, {4}, {3}},
		},
		{
			"Over2",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) error {
				return s.OverN(2)
			},
			nil,
			[][]byte{{1}, {2}, {3}, {4}, {1}, {2}},
		},
		{
			"Over too little",
			[][]byte{{1}},
			func(s *stack) error {
				return s.OverN(1)
			},
			scriptError(ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"Over0",
			[][]byte{{1}, {2}, {3}},
			func(s *stack) error {
				return s.OverN(0)
			},
			scriptError(ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"Pick1",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) error {
				return s.PickN(1)
			},
			nil,
			[][]byte{{1}, {2}, {3}, {4}, {3}},
		},
		{
			"Pick2",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) error {
				return s.PickN(2)
			},
			nil,
			[][]byte{{1}, {2}, {3}, {4}, {2}},
		},
		{
			"Pick too little",
			[][]byte{{1}},
			func(s *stack) error {
				return s.PickN(1)
			},
			scriptError(ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"Roll1",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) error {
				return s.RollN(1)
			},
			nil,
			[][]byte{{1}, {2}, {4}, {3}},
		},
		{
			"Roll2",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) error {
				return s.RollN(2)
			},
			nil,
			[][]byte{{1}, {3}, {4}, {2}},
		},
		{
			"Roll too little",
			[][]byte{{1}},
			func(s *stack) error {
				return s.RollN(1)
			},
			scriptError(ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"Peek bool",
			[][]byte{{1}},
			func(s *stack) error {
//Peek Bool在其他方面都经过了很好的测试，
//只要检查一下就行了。
				val, err := s.PeekBool(0)
				if err != nil {
					return err
				}
				if !val {
					return errors.New("invalid result")
				}
				return nil
			},
			nil,
			[][]byte{{1}},
		},
		{
			"Peek bool 2",
			[][]byte{nil},
			func(s *stack) error {
//Peek Bool在其他方面都经过了很好的测试，
//只要检查一下就行了。
				val, err := s.PeekBool(0)
				if err != nil {
					return err
				}
				if val {
					return errors.New("invalid result")
				}
				return nil
			},
			nil,
			[][]byte{nil},
		},
		{
			"Peek int",
			[][]byte{{1}},
			func(s *stack) error {
//Peek Int在其他方面测试得很好，
//只要检查一下就行了。
				val, err := s.PeekInt(0)
				if err != nil {
					return err
				}
				if val != 1 {
					return errors.New("invalid result")
				}
				return nil
			},
			nil,
			[][]byte{{1}},
		},
		{
			"Peek int 2",
			[][]byte{{0}},
			func(s *stack) error {
//Peek Int在其他方面测试得很好，
//只要检查一下就行了。
				val, err := s.PeekInt(0)
				if err != nil {
					return err
				}
				if val != 0 {
					return errors.New("invalid result")
				}
				return nil
			},
			nil,
			[][]byte{{0}},
		},
		{
			"pop int",
			nil,
			func(s *stack) error {
				s.PushInt(scriptNum(1))
//Peek Int在其他方面测试得很好，
//只要检查一下就行了。
				val, err := s.PopInt()
				if err != nil {
					return err
				}
				if val != 1 {
					return errors.New("invalid result")
				}
				return nil
			},
			nil,
			nil,
		},
		{
			"pop empty",
			nil,
			func(s *stack) error {
//Peek Int在其他方面测试得很好，
//只要检查一下就行了。
				_, err := s.PopInt()
				return err
			},
			scriptError(ErrInvalidStackOperation, ""),
			nil,
		},
	}

	for _, test := range tests {
//设置初始堆栈状态并执行测试操作。
		s := stack{}
		for i := range test.before {
			s.PushByteArray(test.before[i])
		}
		err := test.operation(&s)

//确保错误代码是预期的类型，并且错误
//代码与测试实例中指定的值匹配。
		if e := tstCheckScriptError(err, test.err); e != nil {
			t.Errorf("%s: %v", test.name, e)
			continue
		}
		if err != nil {
			continue
		}

//确保生成的堆栈为预期长度。
		if int32(len(test.after)) != s.Depth() {
			t.Errorf("%s: stack depth doesn't match expected: %v "+
				"vs %v", test.name, len(test.after),
				s.Depth())
			continue
		}

//确保结果堆栈的所有项都是预期的
//价值观。
		for i := range test.after {
			val, err := s.PeekByteArray(s.Depth() - int32(i) - 1)
			if err != nil {
				t.Errorf("%s: can't peek %dth stack entry: %v",
					test.name, i, err)
				break
			}

			if !bytes.Equal(val, test.after[i]) {
				t.Errorf("%s: %dth stack entry doesn't match "+
					"expected: %v vs %v", test.name, i, val,
					test.after[i])
				break
			}
		}
	}
}
