
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2014-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package mining

import (
	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

const (
//Unlinedheight是用于
//事务存储中提供的上下文事务信息
//当它还没有被开采成一个街区时。
	UnminedHeight = 0x7fffffff
)

//策略包含用于控制的策略（配置参数）
//块模板的生成。参见文档
//newblocktemplate用于了解每个参数的更多详细信息。
type Policy struct {
//blockminweight是在
//生成块模板。
	BlockMinWeight uint32

//BlockMaxWeight是当
//生成块模板。
	BlockMaxWeight uint32

//BlockMinWeight是生成时要使用的最小块大小
//块模板。
	BlockMinSize uint32

//BlockMaxSize是生成
//块模板。
	BlockMaxSize uint32

//BlockPrioritySize是高优先级/低费用的字节大小
//生成块模板时要使用的事务。
	BlockPrioritySize uint32

//TXMnFielFor是Satoshi的最低费用/ 1000字节，即
//交易被视为自由采矿所必需的
//（块模板生成）。
	TxMinFreeFee btcutil.Amount
}

//minint是一个帮助函数，返回至少两个int。这避免了
//一个数学导入，需要强制转换为浮点数。
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

//cancaputValueAge是一个辅助函数，用于计算
//交易。txin的输入期限是确认的次数
//因为引用的txout乘以它的输出值。总投入
//年龄是每个txin的值之和。事务的任何输入
//目前在mempool中，因此尚未开采成区块，
//不为事务贡献额外的输入期限。
func calcInputValueAge(tx *wire.MsgTx, utxoView *blockchain.UtxoViewpoint, nextBlockHeight int32) float64 {
	var totalInputAge float64
	for _, txIn := range tx.TxIn {
//如果
//引用的事务输出不存在。
		entry := utxoView.LookupEntry(txIn.PreviousOutPoint)
		if entry != nil && !entry.IsSpent() {
//内存池中当前包含依赖项的输入
//将它们的块高度设置为一个特殊的常量。
//他们的输入年龄应计算为零，因为他们
//家长还没有把它变成一个街区。
			var inputAge int32
			originHeight := entry.BlockHeight()
			if originHeight == UnminedHeight {
				inputAge = 0
			} else {
				inputAge = nextBlockHeight - originHeight
			}

//输入值乘以年龄。
			inputValue := entry.Amount()
			totalInputAge += float64(inputValue * int64(inputAge))
		}
	}

	return totalInputAge
}

//CalcPriority返回给定事务和总和的事务优先级
//每个输入值乘以它们的年龄（确认）。
//因此，优先级的最终公式是：
//SUM（输入值*输入值）/AdjustedTxsize
func CalcPriority(tx *wire.MsgTx, utxoView *blockchain.UtxoViewpoint, nextBlockHeight int32) float64 {
//为了鼓励花费多个旧的未使用的交易
//因此输出减少了总设置，不计算常量
//每个输入的开销以及足够的签名字节
//用压缩的
//普基这通过提高优先级使额外的输入免费
//相应的交易。没有更多的激励来避免
//通过使用垃圾来鼓励游戏未来的交易
//输出。这与引用中使用的逻辑相同
//实施。
//
//txin的常量开销是41个字节，因为
//输出点为36字节+序列4字节+序列1字节
//签名脚本长度。
//
//一个压缩的pubkey pay-to-script哈希赎回，最大长度为
//签名形式如下：
//[op_data_73<73 byte sig>+op_data_35+op_data_33
//<33 byte compressed pubkey>+op_checksig]
//
//因此1+73+1+1+33+1=110
	overhead := 0
	for _, txIn := range tx.TxIn {
//最大输入+大小不能在此溢出。
		overhead += 41 + minInt(110, len(txIn.SignatureScript))
	}

	serializedTxSize := tx.SerializeSize()
	if overhead >= serializedTxSize {
		return 0.0
	}

	inputValueAge := calcInputValueAge(tx, utxoView, nextBlockHeight)
	return inputValueAge / float64(serializedTxSize-overhead)
}
