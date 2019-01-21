
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015-2017 BTCSuite开发者
//版权所有（c）2015-2017法令开发商
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package btcjson

//sessionresult对session命令中的数据进行建模。
type SessionResult struct {
	SessionID uint64 `json:"sessionid"`
}

//RescannedBlock包含单个的哈希和所有发现的事务
//重新扫描的块。
//
//注意：这是从中导入的BTCSuite扩展
//github.com/decred/dcrd/dcrjson。
type RescannedBlock struct {
	Hash         string   `json:"hash"`
	Transactions []string `json:"transactions"`
}
