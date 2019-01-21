
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

package btcjson

import (
	"fmt"
)

//错误代码标识一种错误。这些错误代码不用于
//JSON-RPC响应错误。
type ErrorCode int

//这些常量用于标识特定的RuleError。
const (
//errDuplicateMethod表示使用指定方法的命令
//已经存在。
	ErrDuplicateMethod ErrorCode = iota

//errInvalidUsageFlags表示一个或多个无法识别的标志位
//被指定。
	ErrInvalidUsageFlags

//errInvalidType表示传递的类型不是必需的
//类型。
	ErrInvalidType

//errEmbeddedType指示提供的命令结构包含
//不支持的嵌入类型。
	ErrEmbeddedType

//erUnexportedField指示提供的命令结构包含
//不支持的未排序字段。
	ErrUnexportedField

//errUnsupportedFieldType指示提供的字段的类型
//命令结构不是支持的类型之一。
	ErrUnsupportedFieldType

//errnonOptionalField表示指定了非可选字段
//在可选字段之后。
	ErrNonOptionalField

//errnonOptionalDefault表示“jsonRpcDefault”结构标记为
//为非可选字段指定。
	ErrNonOptionalDefault

//errMismatchedFault表示“jsonRpcDefault”结构标记包含
//与字段类型不匹配的值。
	ErrMismatchedDefault

//errUnregisteredMethod表示指定的方法没有
//已注册。
	ErrUnregisteredMethod

//errMissingDescription指示生成所需的描述
//帮助丢失。
	ErrMissingDescription

//errNumParams inidcates提供的参数数目不要
//匹配相关命令的要求。
	ErrNumParams

//numerorcodes是测试中使用的最大错误代码数。
	numErrorCodes
)

//将错误代码值映射回其常量名，以便进行漂亮的打印。
var errorCodeStrings = map[ErrorCode]string{
	ErrDuplicateMethod:      "ErrDuplicateMethod",
	ErrInvalidUsageFlags:    "ErrInvalidUsageFlags",
	ErrInvalidType:          "ErrInvalidType",
	ErrEmbeddedType:         "ErrEmbeddedType",
	ErrUnexportedField:      "ErrUnexportedField",
	ErrUnsupportedFieldType: "ErrUnsupportedFieldType",
	ErrNonOptionalField:     "ErrNonOptionalField",
	ErrNonOptionalDefault:   "ErrNonOptionalDefault",
	ErrMismatchedDefault:    "ErrMismatchedDefault",
	ErrUnregisteredMethod:   "ErrUnregisteredMethod",
	ErrMissingDescription:   "ErrMissingDescription",
	ErrNumParams:            "ErrNumParams",
}

//字符串将错误代码返回为人类可读的名称。
func (e ErrorCode) String() string {
	if s := errorCodeStrings[e]; s != "" {
		return s
	}
	return fmt.Sprintf("Unknown ErrorCode (%d)", int(e))
}

//错误标识常规错误。这与rpcerror的不同之处在于
//错误通常更多地由包的使用者使用，而不是
//RPCErrors，打算通过
//JSON-RPC响应。调用方可以使用类型断言来确定
//特定错误并访问错误代码字段。
type Error struct {
ErrorCode   ErrorCode //描述错误的类型
Description string    //问题的人类可读描述
}

//错误满足错误接口并打印人类可读的错误。
func (e Error) Error() string {
	return e.Description
}

//makeError在给定一组参数的情况下创建错误。
func makeError(c ErrorCode, desc string) Error {
	return Error{ErrorCode: c, Description: desc}
}
