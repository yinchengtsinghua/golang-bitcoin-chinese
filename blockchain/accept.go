
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package blockchain

import (
	"fmt"

	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcutil"
)

//maybeceptblock可能接受块链中的块，如果
//接受，返回它是否在主链上。它执行
//取决于其在区块链中的位置的几个验证检查
//在添加它之前。预计这个街区已经过了
//在用它调用这个函数之前。
//
//这些标志还传递给checkblockcontext和connectbestchain。见
//有关标志如何修改其行为的文档。
//
//必须在保持链状态锁的情况下调用此函数（用于写入）。
func (b *BlockChain) maybeAcceptBlock(block *btcutil.Block, flags BehaviorFlags) (bool, error) {
//此块的高度比引用的上一块高一倍
//块。
	prevHash := &block.MsgBlock().Header.PrevBlock
	prevNode := b.index.LookupNode(prevHash)
	if prevNode == nil {
		str := fmt.Sprintf("previous block %s is unknown", prevHash)
		return false, ruleError(ErrPreviousBlockUnknown, str)
	} else if b.index.NodeStatus(prevNode).KnownInvalid() {
		str := fmt.Sprintf("previous block %s is known to be invalid", prevHash)
		return false, ruleError(ErrInvalidAncestorBlock, str)
	}

	blockHeight := prevNode.height + 1
	block.SetHeight(blockHeight)

//块必须通过依赖于
//块在块链中的位置。
	err := b.checkBlockContext(block, prevNode, flags)
	if err != nil {
		return false, err
	}

//如果块不在数据库中，请将其插入数据库。偶数
//尽管有可能块最终无法连接，但是
//已经通过了所有的工作证明和有效性测试，这意味着
//对于攻击者来说，填充
//磁盘上有一堆无法连接的块。这是必要的
//因为它允许块下载与更多
//昂贵的连接逻辑。它还有其他一些很好的特性
//例如，制作从未成为主链一部分的块或
//无法连接的块可供进一步分析。
	err = b.db.Update(func(dbTx database.Tx) error {
		return dbStoreBlock(dbTx, block)
	})
	if err != nil {
		return false, err
	}

//为块创建新的块节点并将其添加到节点索引中。偶数
//如果块最终连接到主链，它将开始
//在侧链上。
	blockHeader := &block.MsgBlock().Header
	newNode := newBlockNode(blockHeader, prevNode)
	newNode.status = statusDataStored

	b.index.AddNode(newNode)
	err = b.index.flushToDB()
	if err != nil {
		return false, err
	}

//将传递的块连接到链条上，同时遵守适当的链条
//根据链条选择，工作证明最多。这个
//还处理事务脚本的验证。
	isMainChain, err := b.connectBestChain(newNode, block, flags)
	if err != nil {
		return false, err
	}

//通知调用方新块已被接受到该块中
//链。呼叫方通常希望通过中继
//其他同行的库存。
	b.chainLock.Unlock()
	b.sendNotification(NTBlockAccepted, block)
	b.chainLock.Lock()

	return isMainChain, nil
}
