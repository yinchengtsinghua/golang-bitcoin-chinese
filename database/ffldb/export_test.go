
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
此测试文件是ffldb包的一部分，而不是ffldb-u测试
包装，使其能够连接内部构件，以正确测试
不可能或不能通过公共接口可靠地进行测试。
仅在运行测试时导出函数。
**/


package ffldb

import "github.com/btcsuite/btcd/database"

//tstrunwithmaxblockfilesize以允许的最大值运行传递的函数
//数据库的文件大小设置为提供的值。将设置该值
//完成后返回原值。
func TstRunWithMaxBlockFileSize(idb database.DB, size uint32, fn func()) {
	ffldb := idb.(*db)
	origSize := ffldb.store.maxBlockFileSize

	ffldb.store.maxBlockFileSize = size
	fn()
	ffldb.store.maxBlockFileSize = origSize
}
