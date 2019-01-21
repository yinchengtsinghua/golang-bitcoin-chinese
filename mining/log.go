
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package mining

import (
	"github.com/btcsuite/btclog"
)

//日志是一个没有输出过滤器初始化的日志程序。这个
//意味着在调用方之前，包默认不会执行任何日志记录
//请求它。
var log btclog.Logger

//默认的日志记录量为“无”。
func init() {
	DisableLog()
}

//DisableLog禁用所有库日志输出。日志记录输出被禁用
//默认情况下，直到调用uselogger。
func DisableLog() {
	log = btclog.Disabled
}

//uselogger使用指定的记录器输出包日志信息。
func UseLogger(logger btclog.Logger) {
	log = logger
}
