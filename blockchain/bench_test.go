
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package blockchain

import (
	"testing"

	"github.com/btcsuite/btcutil"
)

//BenchmarkiscoinBase对iscoinBase执行简单的基准测试
//功能。
func BenchmarkIsCoinBase(b *testing.B) {
	tx, _ := btcutil.NewBlock(&Block100000).Tx(1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsCoinBase(tx)
	}
}

//BenchmarkiscoinBaseTx对iscoinBaseTx执行简单的基准测试
//功能。
func BenchmarkIsCoinBaseTx(b *testing.B) {
	tx := Block100000.Transactions[1]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsCoinBaseTx(tx)
	}
}
