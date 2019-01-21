
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
包ffldb为使用leveldb的数据库包实现驱动程序
用于备份元数据和块存储的平面文件。

此驱动程序是与BTCD一起使用的推荐驱动程序。它使用级别DB
对于元数据、块存储的平面文件和关键区域的校验和，请执行以下操作：
确保数据完整性。

用法

此包是数据库包的驱动程序，并提供数据库类型
“FFLDB”。open和create函数采用的参数是
作为字符串的数据库路径和块网络：

 db，err：=database.open（“ffldb”，“path/to/database”，wire.mainnet）
 如果犯错！= nIL{
  //句柄错误
 }

 db，err：=database.create（“ffldb”，“path/to/database”，wire.mainnet）
 如果犯错！= nIL{
  //句柄错误
 }
**/

package ffldb
