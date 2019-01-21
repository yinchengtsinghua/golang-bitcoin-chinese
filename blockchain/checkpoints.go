
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package blockchain

import (
	"fmt"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
)

//CheckpointConfirmations是当前块结束之前的块数
//一个好的检查点候选必须是的最佳块链。
const CheckpointConfirmations = 2016

//newhashfromstr将传递的big endian十六进制字符串转换为
//chainhash.hash。它只与chainhash中可用的不同之处在于
//它忽略错误，因为它只能（并且必须）用
//硬编码，因此已知良好，哈希。
func newHashFromStr(hexStr string) *chainhash.Hash {
	hash, _ := chainhash.NewHashFromStr(hexStr)
	return hash
}

//检查点返回一部分检查点（不管它们是否
//已经知道了）。当链没有检查点时，它将返回
//零。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) Checkpoints() []chaincfg.Checkpoint {
	return b.checkpoints
}

//hasCheckpoints返回此区块链是否定义了检查点。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) HasCheckpoints() bool {
	return len(b.checkpoints) > 0
}

//LatestCheckpoint返回最新的检查点（无论是否
//已经知道）。当没有为活动链定义检查点时
//例如，它将返回nil。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) LatestCheckpoint() *chaincfg.Checkpoint {
	if !b.HasCheckpoints() {
		return nil
	}
	return &b.checkpoints[len(b.checkpoints)-1]
}

//verifycheckpoint返回传递的块高度和哈希组合
//匹配检查点数据。如果没有检查点，它也返回true
//传递的块高度的数据。
func (b *BlockChain) verifyCheckpoint(height int32, hash *chainhash.Hash) bool {
	if !b.HasCheckpoints() {
		return true
	}

//不检查块高度是否没有检查点数据。
	checkpoint, exists := b.checkpointsByHeight[height]
	if !exists {
		return true
	}

	if !checkpoint.Hash.IsEqual(hash) {
		return false
	}

	log.Infof("Verified checkpoint at height %d/block %s", checkpoint.Height,
		checkpoint.Hash)
	return true
}

//findPreviousCheckpoint查找已存在的最新检查点
//在块链的下载部分中可用，并返回
//关联的块节点。如果找不到检查点，则返回nil（此
//应该只对第一个检查点之前的块发生）。
//
//必须在保持链锁的情况下调用此函数（用于读取）。
func (b *BlockChain) findPreviousCheckpoint() (*blockNode, error) {
	if !b.HasCheckpoints() {
		return nil, nil
	}

//执行初始搜索以查找和缓存最新的已知
//如果最好的链还不知道或者我们还没有
//以前搜索过。
	checkpoints := b.checkpoints
	numCheckpoints := len(checkpoints)
	if b.checkpointNode == nil && b.nextCheckpoint == nil {
//向后循环通过可用的检查点以查找一个
//已经有了。
		for i := numCheckpoints - 1; i >= 0; i-- {
			node := b.index.LookupNode(checkpoints[i].Hash)
			if node == nil || !b.bestChain.Contains(node) {
				continue
			}

//找到检查点。缓存它以备将来查找和
//相应地设置下一个预期的检查点。
			b.checkpointNode = node
			if i < numCheckpoints-1 {
				b.nextCheckpoint = &checkpoints[i+1]
			}
			return b.checkpointNode, nil
		}

//没有已知的最新检查点。这只会发生在街区上
//在第一个已知的检查点之前。所以，设置下一个预期值
//检查点到第一个检查点并返回到那里
//不是最新的已知检查点块。
		b.nextCheckpoint = &checkpoints[0]
		return nil, nil
	}

//现在我们已经搜索了最新的已知检查点，
//所以当没有下一个检查点时，当前的检查点锁定
//将始终是最新的已知检查点。
	if b.nextCheckpoint == nil {
		return b.checkpointNode, nil
	}

//当有下一个检查点和当前最佳高度时
//链没有超过它，当前检查点锁定仍然是
//最新的已知检查点。
	if b.bestChain.Tip().height < b.nextCheckpoint.Height {
		return b.checkpointNode, nil
	}

//我们已达到或超过下一个检查点高度。注意
//一旦达到检查点锁定，就可以阻止fork
//检查站前的任何街区，所以我们不必担心
//检查站由于连锁重组而从我们下面消失。

//缓存最新的已知检查点以备将来查找。注意如果
//这个查找失败了，因为链已经
//通过了插入前已验证为准确的检查点
//它。
	checkpointNode := b.index.LookupNode(b.nextCheckpoint.Hash)
	if checkpointNode == nil {
		return nil, AssertError(fmt.Sprintf("findPreviousCheckpoint "+
			"failed lookup of known good block node %s",
			b.nextCheckpoint.Hash))
	}
	b.checkpointNode = checkpointNode

//设置下一个预期检查点。
	checkpointIndex := -1
	for i := numCheckpoints - 1; i >= 0; i-- {
		if checkpoints[i].Hash.IsEqual(b.nextCheckpoint.Hash) {
			checkpointIndex = i
			break
		}
	}
	b.nextCheckpoint = nil
	if checkpointIndex != -1 && checkpointIndex < numCheckpoints-1 {
		b.nextCheckpoint = &checkpoints[checkpointIndex+1]
	}

	return b.checkpointNode, nil
}

//IsOnStandardTransaction确定事务是否包含
//不是标准类型的脚本。
func isNonstandardTransaction(tx *btcutil.Tx) bool {
//检查所有输出公钥脚本的非标准脚本。
	for _, txOut := range tx.MsgTx().TxOut {
		scriptClass := txscript.GetScriptClass(txOut.PkScript)
		if scriptClass == txscript.NonStandardTy {
			return true
		}
	}
	return false
}

//ischeckpointcandidate返回传递的块是否为好的
//检查点候选。
//
//用于确定良好检查点的因素有：
//-滑轮必须在主链中。
//-块必须至少是“checkpointconfirmations”块，然后才能
//主链的当前末端
//-检查点前后块的时间戳必须
//分别在检查点之前和之后的时间戳
//（由于时间允许的中位数，情况并非总是如此）
//-块不能包含任何奇怪的事务，例如
//非标准脚本
//
//其目的是让开发人员对候选人进行审查，以确定最终结果。
//然后手动添加到网络的检查点列表中。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) IsCheckpointCandidate(block *btcutil.Block) (bool, error) {
	b.chainLock.RLock()
	defer b.chainLock.RUnlock()

//检查点必须在主链中。
	node := b.index.LookupNode(block.Hash())
	if node == nil || !b.bestChain.Contains(node) {
		return false, nil
	}

//确保通过的挡块的高度和挡块的入口
//主链匹配。除非
//调用方提供了无效的块。
	if node.height != block.Height() {
		return false, fmt.Errorf("passed block height of %d does not "+
			"match the main chain height of %d", block.Height(),
			node.height)
	}

//检查点必须至少是检查点确认块
//在主链末端之前。
	mainChainHeight := b.bestChain.Tip().height
	if node.height > (mainChainHeight - CheckpointConfirmations) {
		return false, nil
	}

//检查点后面必须至少有一个块。
//
//这应该总是成功的，因为上面的检查已经确认了
//检查点确认返回，但在常量
//变化。
	nextNode := b.bestChain.Next(node)
	if nextNode == nil {
		return false, nil
	}

//检查点之前必须至少有一个块。
	if node.parent == nil {
		return false, nil
	}

//检查点必须具有块和块的时间戳
//它的任何一边都是有序的（由于中间时间允许，这是
//情况并非总是如此）。
	prevTime := time.Unix(node.parent.timestamp, 0)
	curTime := block.MsgBlock().Header.Timestamp
	nextTime := time.Unix(nextNode.timestamp, 0)
	if prevTime.After(curTime) || nextTime.Before(curTime) {
		return false, nil
	}

//检查点必须具有仅包含标准的事务
//脚本。
	for _, tx := range block.Transactions() {
		if isNonstandardTransaction(tx) {
			return false, nil
		}
	}

//所有的检查都通过了，所以该块是一个候选块。
	return true, nil
}
