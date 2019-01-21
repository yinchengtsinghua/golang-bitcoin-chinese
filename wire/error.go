
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2015 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wire

import (
	"fmt"
)

//messageerror描述了带有消息的问题。
//一些潜在问题的例子是来自错误比特币的信息
//网络、无效命令、不匹配的校验和以及超过最大有效负载。
//
//这为调用者提供了一种将错误断言到
//区分一般IO错误（如IO.EOF）和
//邮件格式不正确。
type MessageError struct {
Func        string //函数名
Description string //问题的人类可读描述
}

//错误满足错误接口并打印人类可读的错误。
func (e *MessageError) Error() string {
	if e.Func != "" {
		return fmt.Sprintf("%v: %v", e.Func, e.Description)
	}
	return e.Description
}

//messageerror为给定的函数和说明创建错误。
func messageError(f string, desc string) *MessageError {
	return &MessageError{Func: f, Description: desc}
}
