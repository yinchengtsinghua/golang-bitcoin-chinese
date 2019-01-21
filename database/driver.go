
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

import (
	"fmt"

	"github.com/btcsuite/btclog"
)

//驱动程序定义后端驱动程序在注册时使用的结构
//它们本身是实现数据库接口的后端。
type Driver struct {
//dbtype是用于唯一标识特定
//数据库驱动程序。只能有一个同名的驱动程序。
	DbType string

//create是将在所有用户指定的情况下调用的函数
//用于创建数据库的参数。此函数必须返回
//如果数据库已存在，则errdbexists。
	Create func(args ...interface{}) (DB, error)

//open是将在所有用户指定的情况下调用的函数
//用于打开数据库的参数。此函数必须返回
//如果尚未创建数据库，则返回errdbdoesnotex。
	Open func(args ...interface{}) (DB, error)

//uselogger使用指定的记录器输出包日志信息。
	UseLogger func(logger btclog.Logger)
}

//DriverList保存所有注册的数据库后端。
var drivers = make(map[string]*Driver)

//RegisterDriver向可用接口添加后端数据库驱动程序。
//如果驱动程序的数据库类型
//已经注册。
func RegisterDriver(driver Driver) error {
	if _, exists := drivers[driver.DbType]; exists {
		str := fmt.Sprintf("driver %q is already registered",
			driver.DbType)
		return makeError(ErrDbTypeRegistered, str, nil)
	}

	drivers[driver.DbType] = &driver
	return nil
}

//SupportedDrivers返回表示数据库的字符串切片
//已注册并因此受支持的驱动程序。
func SupportedDrivers() []string {
	supportedDBs := make([]string, 0, len(drivers))
	for _, drv := range drivers {
		supportedDBs = append(supportedDBs, drv.DbType)
	}
	return supportedDBs
}

//create初始化并打开指定类型的数据库。这个
//参数特定于数据库类型驱动程序。参见文档
//有关数据库驱动程序的详细信息。
//
//如果数据库类型未注册，则返回errdUnknownType。
func Create(dbType string, args ...interface{}) (DB, error) {
	drv, exists := drivers[dbType]
	if !exists {
		str := fmt.Sprintf("driver %q is not registered", dbType)
		return nil, makeError(ErrDbUnknownType, str, nil)
	}

	return drv.Create(args...)
}

//打开打开指定类型的现有数据库。这些论点是
//特定于数据库类型驱动程序。请参阅数据库的文档
//有关详细信息，请参阅驱动程序。
//
//如果数据库类型未注册，则返回errdUnknownType。
func Open(dbType string, args ...interface{}) (DB, error) {
	drv, exists := drivers[dbType]
	if !exists {
		str := fmt.Sprintf("driver %q is not registered", dbType)
		return nil, makeError(ErrDbUnknownType, str, nil)
	}

	return drv.Open(args...)
}
