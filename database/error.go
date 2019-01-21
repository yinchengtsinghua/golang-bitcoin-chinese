
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

package database

import "fmt"

//错误代码标识一种错误。
type ErrorCode int

//这些常量用于标识特定的数据库错误。
const (
//********************************
//与驱动程序注册相关的错误。
//********************************

//errdbtyperegistered表示两个不同的数据库驱动程序
//尝试用名称数据库类型注册。
	ErrDbTypeRegistered ErrorCode = iota

//********************************
//与数据库函数相关的错误。
//********************************

//errdUnknownType表示没有为注册的驱动程序
//指定的数据库类型。
	ErrDbUnknownType

//errddoesNoteList指示对以下数据库调用open
//不存在。
	ErrDbDoesNotExist

//errdbexists指示为数据库调用create
//已经存在。
	ErrDbExists

//errdnotopen表示以前访问过数据库实例
//打开或关闭后。
	ErrDbNotOpen

//errdbalreadyopen表示在数据库上调用了open
//已经打开。
	ErrDbAlreadyOpen

//errInvalid表示指定的数据库无效。
	ErrInvalid

//errCorruption表示发生校验和故障，该故障总是
//表示数据库已损坏。
	ErrCorruption

//*************************************
//与数据库事务相关的错误。
//*************************************

//errtxclosed表示试图提交或回滚
//已执行其中一个操作的事务。
	ErrTxClosed

//errtxnotwritable表示需要对其进行写访问的操作
//数据库试图用于只读事务。
	ErrTxNotWritable

//********************************
//与元数据操作相关的错误。
//********************************

//ErrBucketNotFound表示尝试访问具有
//尚未创建。
	ErrBucketNotFound

//errbacketexists表示试图创建已经
//存在。
	ErrBucketExists

//errBucketnameRequired表示尝试使用
//空白名称。
	ErrBucketNameRequired

//errkeyRequired指示尝试插入零长度密钥。
	ErrKeyRequired

//errkeytoolarge指示attmempt插入较大的键
//大于允许的最大密钥大小。最大键大小取决于
//正在使用的特定后端驱动程序。一般来说，密钥大小
//应该是相对的，所以这应该很少成为一个问题。
	ErrKeyTooLarge

//errValuetoolArge表示要插入较大值的attmpt
//大于最大允许值大小。最大键大小取决于
//正在使用的特定后端驱动程序。
	ErrValueTooLarge

//errUncompatibleValue指示相关值对于无效
//请求的特定操作。例如，尝试创建或
//删除具有现有非bucket键的bucket，尝试创建
//或者用现有的bucket键删除非bucket键，或者尝试
//当一个值指向一个嵌套的bucket时，通过光标删除它。
	ErrIncompatibleValue

//*************************************
//与块I/O操作相关的错误。
//*************************************

//errblocknotfound表示具有提供的哈希的块没有
//存在于数据库中。
	ErrBlockNotFound

//errblockexists表示已提供哈希的块
//数据库中存在。
	ErrBlockExists

//errBlockRegionInvalid表示区域超出了
//已请求指定的块。当哈希由
//区域与现有块不对应，错误将为
//却没有找到errblock。
	ErrBlockRegionInvalid

//***************************
//支持特定于驱动程序的错误。
//***************************

//errDriverSpecific表示err字段是驱动程序特定的错误。
//这为驱动程序提供了一种机制来插入他们自己的自定义
//任何未包含在错误中的情况的错误
//此包提供的代码。
	ErrDriverSpecific

//numerorcodes是测试中使用的最大错误代码数。
	numErrorCodes
)

//将错误代码值映射回其常量名，以便进行漂亮的打印。
var errorCodeStrings = map[ErrorCode]string{
	ErrDbTypeRegistered:   "ErrDbTypeRegistered",
	ErrDbUnknownType:      "ErrDbUnknownType",
	ErrDbDoesNotExist:     "ErrDbDoesNotExist",
	ErrDbExists:           "ErrDbExists",
	ErrDbNotOpen:          "ErrDbNotOpen",
	ErrDbAlreadyOpen:      "ErrDbAlreadyOpen",
	ErrInvalid:            "ErrInvalid",
	ErrCorruption:         "ErrCorruption",
	ErrTxClosed:           "ErrTxClosed",
	ErrTxNotWritable:      "ErrTxNotWritable",
	ErrBucketNotFound:     "ErrBucketNotFound",
	ErrBucketExists:       "ErrBucketExists",
	ErrBucketNameRequired: "ErrBucketNameRequired",
	ErrKeyRequired:        "ErrKeyRequired",
	ErrKeyTooLarge:        "ErrKeyTooLarge",
	ErrValueTooLarge:      "ErrValueTooLarge",
	ErrIncompatibleValue:  "ErrIncompatibleValue",
	ErrBlockNotFound:      "ErrBlockNotFound",
	ErrBlockExists:        "ErrBlockExists",
	ErrBlockRegionInvalid: "ErrBlockRegionInvalid",
	ErrDriverSpecific:     "ErrDriverSpecific",
}

//字符串将错误代码返回为人类可读的名称。
func (e ErrorCode) String() string {
	if s := errorCodeStrings[e]; s != "" {
		return s
	}
	return fmt.Sprintf("Unknown ErrorCode (%d)", int(e))
}

//错误为数据库期间可能发生的错误提供单一类型
//操作。它用于指示几种类型的故障，包括错误
//使用调用方请求，例如指定无效的块区域或尝试
//根据已关闭的数据库事务、驱动程序错误、错误访问数据
//检索数据，以及与数据库服务器通信时出错。
//
//调用方可以使用类型断言来确定错误是否是错误，以及
//access the ErrorCode field to ascertain the specific reason for the failure.
//
//errDriverSpecific错误代码还将使用
//潜在错误。根据后端驱动程序的不同，err字段可能是
//对于其他错误代码也设置为基础错误。
type Error struct {
ErrorCode   ErrorCode //描述错误的类型
Description string    //问题的人类可读描述
Err         error     //潜在错误
}

//错误满足错误接口并打印人类可读的错误。
func (e Error) Error() string {
	if e.Err != nil {
		return e.Description + ": " + e.Err.Error()
	}
	return e.Description
}

//makeError在给定一组参数的情况下创建错误。错误代码必须
//是此包提供的错误代码之一。
func makeError(c ErrorCode, desc string, err error) Error {
	return Error{ErrorCode: c, Description: desc, Err: err}
}
