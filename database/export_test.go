
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

/*
此测试文件是数据库包的一部分，而不是
数据库测试包，以便它能够桥接对内部的访问，以便正确测试
不可能或不能通过公众可靠测试的案例
接口。函数、常量和变量仅在
正在运行测试。
**/


package database

//tstnumerrorcodes使内部numerorcodes参数可用于
//测试包。
const TstNumErrorCodes = numErrorCodes
