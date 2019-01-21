
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
	"encoding/hex"
	"fmt"
)

//asbool获取字节数组的布尔值。
func asBool(t []byte) bool {
	for i := range t {
		if t[i] != 0 {
//负0也被认为是错误的。
			if i == len(t)-1 && t[i] == 0x80 {
				return false
			}
			return true
		}
	}
	return false
}

//FromBool将布尔值转换为适当的字节数组。
func fromBool(v bool) []byte {
	if v {
		return []byte{1}
	}
	return nil
}

//Stack表示与比特币一起使用的不可变对象的堆栈
//脚本。对象可以被共享，因此在使用时，如果一个值
//已更改*必须*首先进行深度复制，以避免更改
//栈。
type stack struct {
	stk               [][]byte
	verifyMinimalData bool
}

//Depth返回堆栈上的项数。
func (s *stack) Depth() int32 {
	return int32(len(s.stk))
}

//pushbytearray将给定的返回数组添加到堆栈的顶部。
//
//堆栈转换：…X1 x2] ->…X1 X2数据
func (s *stack) PushByteArray(so []byte) {
	s.stk = append(s.stk, so)
}

//pushint将提供的scriptnum转换为合适的字节数组，然后push
//它在栈顶上。
//
//堆栈转换：…X1 x2] ->…x1 x2 int
func (s *stack) PushInt(val scriptNum) {
	s.PushByteArray(val.Bytes())
}

//pushbool将提供的布尔值转换为适当的字节数组，然后按
//它在栈顶上。
//
//堆栈转换：…X1 x2] ->…X1 x2 BOOL
func (s *stack) PushBool(val bool) {
	s.PushByteArray(fromBool(val))
}

//PopBytearray从堆栈顶部弹出值并返回它。
//
//堆栈转换：…x1 x2 x3]->[…X1 x2]
func (s *stack) PopByteArray() ([]byte, error) {
	return s.nipN(0)
}

//popint从堆栈顶部弹出值，将其转换为脚本
//然后返回。转换为script num的操作强制
//对被解释为数字的数据实施的共识规则。
//
//堆栈转换：…x1 x2 x3]->[…X1 x2]
func (s *stack) PopInt() (scriptNum, error) {
	so, err := s.PopByteArray()
	if err != nil {
		return 0, err
	}

	return makeScriptNum(so, s.verifyMinimalData, defaultScriptNumLen)
}

//popbool从堆栈顶部弹出值，将其转换为bool，然后
//返回它。
//
//堆栈转换：…x1 x2 x3]->[…X1 x2]
func (s *stack) PopBool() (bool, error) {
	so, err := s.PopByteArray()
	if err != nil {
		return false, err
	}

	return asBool(so), nil
}

//PeekBytearray返回堆栈上的第n个项，而不删除它。
func (s *stack) PeekByteArray(idx int32) ([]byte, error) {
	sz := int32(len(s.stk))
	if idx < 0 || idx >= sz {
		str := fmt.Sprintf("index %d is invalid for stack size %d", idx,
			sz)
		return nil, scriptError(ErrInvalidStackOperation, str)
	}

	return s.stk[sz-idx-1], nil
}

//peekint将堆栈上的第n个项作为脚本num返回，而不移除
//它。转换为script num的行为强制执行共识规则
//用于解释为数字的数据。
func (s *stack) PeekInt(idx int32) (scriptNum, error) {
	so, err := s.PeekByteArray(idx)
	if err != nil {
		return 0, err
	}

	return makeScriptNum(so, s.verifyMinimalData, defaultScriptNumLen)
}

//peekbool将堆栈上的第n个项作为bool返回，而不移除它。
func (s *stack) PeekBool(idx int32) (bool, error) {
	so, err := s.PeekByteArray(idx)
	if err != nil {
		return false, err
	}

	return asBool(so), nil
}

//NIPN是一个内部函数，它删除堆栈上的第n个项，并
//返回它。
//
//堆栈转换：
//NIPN（0）：…x1 x2 x3]->[…X1 x2]
//NIPN（1）：…x1 x2 x3]->[…X1 x3]
//NIPN（2）：…x1 x2 x3]->[…X2 x3]
func (s *stack) nipN(idx int32) ([]byte, error) {
	sz := int32(len(s.stk))
	if idx < 0 || idx > sz-1 {
		str := fmt.Sprintf("index %d is invalid for stack size %d", idx,
			sz)
		return nil, scriptError(ErrInvalidStackOperation, str)
	}

	so := s.stk[sz-idx-1]
	if idx == 0 {
		s.stk = s.stk[:sz-1]
	} else if idx == sz-1 {
		s1 := make([][]byte, sz-1)
		copy(s1, s.stk[1:])
		s.stk = s1
	} else {
		s1 := s.stk[sz-idx : sz]
		s.stk = s.stk[:sz-idx-1]
		s.stk = append(s.stk, s1...)
	}
	return so, nil
}

//NIPN删除堆栈上的第n个对象
//
//堆栈转换：
//NipN（0）：…x1 x2 x3]->[…X1 x2]
//NipN（1）：…x1 x2 x3]->[…X1 x3]
//NipN（2）：…x1 x2 x3]->[…X2 x3]
func (s *stack) NipN(idx int32) error {
	_, err := s.nipN(idx)
	return err
}

//tuck复制堆栈顶部的项目，并在第2个之前将其插入
//到顶部项目。
//
//堆栈转换：…X1 x2] ->…X2 x1 x2]
func (s *stack) Tuck() error {
	so2, err := s.PopByteArray()
	if err != nil {
		return err
	}
	so1, err := s.PopByteArray()
	if err != nil {
		return err
	}
s.PushByteArray(so2) //堆栈[…X2]
s.PushByteArray(so1) //堆栈[…X2X1]
s.PushByteArray(so2) //堆栈[…X2 x1 x2]

	return nil
}

//dropn从堆栈中删除前n个项。
//
//堆栈转换：
//DROPN（1）：…X1 x2] ->…X1]
//DROPN（2）：…X1×2] > >…
func (s *stack) DropN(n int32) error {
	if n < 1 {
		str := fmt.Sprintf("attempt to drop %d items from stack", n)
		return scriptError(ErrInvalidStackOperation, str)
	}

	for ; n > 0; n-- {
		_, err := s.PopByteArray()
		if err != nil {
			return err
		}
	}
	return nil
}

//dupn复制堆栈中前n项。
//
//堆栈转换：
//DUPN（1）：…X1 x2] ->…X1 x2 x2]
//DUPN（2）：…X1 x2] ->…X1 x2 x1 x2]
func (s *stack) DupN(n int32) error {
	if n < 1 {
		str := fmt.Sprintf("attempt to dup %d stack items", n)
		return scriptError(ErrInvalidStackOperation, str)
	}

//迭代地将值n-1复制到堆栈中n次。
//这将在堆栈上留下前n个项的顺序副本。
	for i := n; i > 0; i-- {
		so, err := s.PeekByteArray(n - 1)
		if err != nil {
			return err
		}
		s.PushByteArray(so)
	}
	return nil
}

//rotn将堆栈上的前3n项向左旋转n次。
//
//堆栈转换：
//ROTN（1）：…x1 x2 x3]->[…X2 x3 x1]
//ROTN（2）：…x1 x2 x3 x4 x5 x6]->[…x3 x4 x5 x6 x1 x2]
func (s *stack) RotN(n int32) error {
	if n < 1 {
		str := fmt.Sprintf("attempt to rotate %d stack items", n)
		return scriptError(ErrInvalidStackOperation, str)
	}

//将3N-1th项从堆栈压入顶部N次以旋转
//它们一直到堆的顶端。
	entry := 3*n - 1
	for i := n; i > 0; i-- {
		so, err := s.nipN(entry)
		if err != nil {
			return err
		}

		s.PushByteArray(so)
	}
	return nil
}

//swapn将堆栈上的前n个项与下面的项交换。
//
//堆栈转换：
//SWAPN（1）：…X1 x2] ->…X2X1]
//SWAPN（2）：…x1 x2 x3 x4]->[…x3x4x1-x2]
func (s *stack) SwapN(n int32) error {
	if n < 1 {
		str := fmt.Sprintf("attempt to swap %d stack items", n)
		return scriptError(ErrInvalidStackOperation, str)
	}

	entry := 2*n - 1
	for i := n; i > 0; i-- {
//将2n-1th项切换到顶部。
		so, err := s.nipN(entry)
		if err != nil {
			return err
		}

		s.PushByteArray(so)
	}
	return nil
}

//overn将n个项目n个项目复制回堆栈顶部。
//
//堆栈转换：
//On n（1）：…x1 x2 x3]->[…X1 x2 x3 x2]
//On n（2）：…x1 x2 x3 x4]->[…x1 x2 x3 x4 x1 x2]
func (s *stack) OverN(n int32) error {
	if n < 1 {
		str := fmt.Sprintf("attempt to perform over on %d stack items",
			n)
		return scriptError(ErrInvalidStackOperation, str)
	}

//将2n-1th项复制到堆栈顶部。
	entry := 2*n - 1
	for ; n > 0; n-- {
		so, err := s.PeekByteArray(entry)
		if err != nil {
			return err
		}
		s.PushByteArray(so)
	}

	return nil
}

//pickn将n项复制回堆栈顶部。
//
//堆栈转换：
//选择（0）：[x1 x2 x3]->[x1 x2 x3]
//pickn（1）：[x1 x2 x3]->[x1 x2 x3 x2]
//pickn（2）：[x1 x2 x3]->[x1 x2 x3 x1]
func (s *stack) PickN(n int32) error {
	so, err := s.PeekByteArray(n)
	if err != nil {
		return err
	}
	s.PushByteArray(so)

	return nil
}

//Rolln将堆栈中的n个项移回顶部。
//
//堆栈转换：
//罗伦（0）：[x1 x2 x3]->[x1 x2 x3]
//罗伦（1）：[x1 x2 x3]->[x1 x3 x2]
//罗伦（2）：[x1 x2 x3]->[x2 x3 x1]
func (s *stack) RollN(n int32) error {
	so, err := s.nipN(n)
	if err != nil {
		return err
	}

	s.PushByteArray(so)

	return nil
}

//字符串以可读格式返回堆栈。
func (s *stack) String() string {
	var result string
	for _, stack := range s.stk {
		if len(stack) == 0 {
			result += "00000000  <empty>\n"
		}
		result += hex.Dump(stack)
	}

	return result
}
