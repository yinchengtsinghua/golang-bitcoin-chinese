
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
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcutil"
)

//BehaviorFlags是一个位掩码，用于定义在
//执行链处理和共识规则检查。
type BehaviorFlags uint32

const (
//bfastadd可以设置为表示可以避免多次检查
//因为已经知道它适合链条
//已经证明它与已知的
//检查点。这主要用于头一模式。
	BFFastAdd BehaviorFlags = 1 << iota

//bfNoBowCheck可设置为指示工作证明检查
//确保块哈希值小于所需目标值
//
	BFNoPoWCheck

//bfnone是一个方便的值，专门表示没有标志。
	BFNone BehaviorFlags = 0
)

//block exists确定具有给定哈希的块是否存在于
//主链或任何侧链。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) blockExists(hash *chainhash.Hash) (bool, error) {
//首先检查块索引（可以是主链或侧链块）。
	if b.index.HaveBlock(hash) {
		return true, nil
	}

//签入数据库。
	var exists bool
	err := b.db.View(func(dbTx database.Tx) error {
		var err error
		exists, err = dbTx.HasBlock(hash)
		if err != nil || !exists {
			return err
		}

//忽略数据库中的侧链块。这是必要的
//因为当前没有任何相关联的
//块索引数据，如其块高度，因此尚未
//可以有效地加载块并执行任何有用的操作
//有了它。
//
//
//而不仅仅是当前的主链，因此可以参考
//直接。
		_, err = dbFetchHeightByHash(dbTx, hash)
		if isNotInMainChainErr(err) {
			exists = false
			return nil
		}
		return err
	})
	return exists, err
}

//processOrphans确定是否有依赖于传递的
//
//它重复新接受块的过程（以进一步检测
//
//
//
//需要传递到maybeceptblock。
//
//必须在保持链状态锁的情况下调用此函数（用于写入）。
func (b *BlockChain) processOrphans(hash *chainhash.Hash, flags BehaviorFlags) error {
//至少从处理传递的哈希开始。留下一点空间
//对于需要处理的其他孤立块
//
	processHashes := make([]*chainhash.Hash, 0, 10)
	processHashes = append(processHashes, hash)
	for len(processHashes) > 0 {
//从切片中弹出要处理的第一个哈希。
		processHash := processHashes[0]
processHashes[0] = nil //防止GC泄漏。
		processHashes = processHashes[1:]

//看看所有的孤儿，他们都是我们的父母
//认可的。这通常只有一个，但它可以
//如果挖掘和广播多个块，则为多个
//大约在同一时间。最有工作证明的人
//最终会胜出。循环的索引是
//故意在一个范围内使用，因为范围不
//在每次迭代中重新评估切片，也不调整
//
		for i := 0; i < len(b.prevOrphans[*processHash]); i++ {
			orphan := b.prevOrphans[*processHash][i]
			if orphan == nil {
				log.Warnf("Found a nil entry at index %d in the "+
					"orphan dependency list for block %v", i,
					processHash)
				continue
			}

//从孤儿池中移除孤儿。
			orphanHash := orphan.block.Hash()
			b.removeOrphanBlock(orphan)
			i--

//可能会将块接受到块链中。
			_, err := b.maybeAcceptBlock(orphan.block, flags)
			if err != nil {
				return err
			}

//将此块添加到要处理的块列表中，以便
//依赖于此块的任何孤立块都是
//也处理过。
			processHashes = append(processHashes, orphanHash)
		}
	}
	return nil
}

//processBlock是处理将新块插入到
//区块链。它包括拒绝复制等功能
//块，确保块遵循所有规则、孤立处理和插入
//区块链以及最佳的链选择和重组。
//
//当处理过程中没有发生错误时，第一个返回值表示
//块是否在主链上，第二个指示
//块是否为孤立块。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) ProcessBlock(block *btcutil.Block, flags BehaviorFlags) (bool, bool, error) {
	b.chainLock.Lock()
	defer b.chainLock.Unlock()

	fastAdd := flags&BFFastAdd == BFFastAdd

	blockHash := block.Hash()
	log.Tracef("Processing block %v", blockHash)

//主链或侧链中不得存在块。
	exists, err := b.blockExists(blockHash)
	if err != nil {
		return false, false, err
	}
	if exists {
		str := fmt.Sprintf("already have block %v", blockHash)
		return false, false, ruleError(ErrDuplicateBlock, str)
	}

//该块不能作为孤立块存在。
	if _, exists := b.orphans[*blockHash]; exists {
		str := fmt.Sprintf("already have block (orphan) %v", blockHash)
		return false, false, ruleError(ErrDuplicateBlock, str)
	}

//对块及其事务执行初步的健全性检查。
	err = checkBlockSanity(block, b.chainParams.PowLimit, b.timeSource, flags)
	if err != nil {
		return false, false, err
	}

//查找上一个检查点并根据
//在检查站。这提供了一些很好的属性，例如
//
//拒绝容易被我发现，但其他方面是伪造的，可能是
//用来吃记忆，并确保预期（与声称）的证据
//满足上一个检查点以来的工作要求。
	blockHeader := &block.MsgBlock().Header
	checkpointNode, err := b.findPreviousCheckpoint()
	if err != nil {
		return false, false, err
	}
	if checkpointNode != nil {
//确保块时间戳在检查点时间戳之后。
		checkpointTime := time.Unix(checkpointNode.timestamp, 0)
		if blockHeader.Timestamp.Before(checkpointTime) {
			str := fmt.Sprintf("block %v has timestamp %v before "+
				"last checkpoint timestamp %v", blockHash,
				blockHeader.Timestamp, checkpointTime)
			return false, false, ruleError(ErrCheckpointTimeTooOld, str)
		}
		if !fastAdd {
//即使之前的检查已经确保
//工程证明超过索赔金额，索赔金额
//是块头中可以锻造的字段。这个
//检查确保工作证明至少是最低限度的
//根据上次检查点和
//重定目标规则允许的最大调整。
			duration := blockHeader.Timestamp.Sub(checkpointTime)
			requiredTarget := CompactToBig(b.calcEasiestDifficulty(
				checkpointNode.bits, duration))
			currentTarget := CompactToBig(blockHeader.Bits)
			if currentTarget.Cmp(requiredTarget) > 0 {
				str := fmt.Sprintf("block target difficulty of %064x "+
					"is too low when compared to the previous "+
					"checkpoint", currentTarget)
				return false, false, ruleError(ErrDifficultyTooLow, str)
			}
		}
	}

//处理孤立块。
	prevHash := &blockHeader.PrevBlock
	prevHashExists, err := b.blockExists(prevHash)
	if err != nil {
		return false, false, err
	}
	if !prevHashExists {
		log.Infof("Adding orphan block %v with parent %v", blockHash, prevHash)
		b.addOrphanBlock(block)

		return false, true, nil
	}

//该块已通过所有上下文无关的检查，并且看起来正常
//足以接受它进入区块链。
	isMainChain, err := b.maybeAcceptBlock(block, flags)
	if err != nil {
		return false, false, err
	}

//接受依赖于此块的任何孤立块（它们是
//
//再也没有了。
	err = b.processOrphans(blockHash, flags)
	if err != nil {
		return false, false, err
	}

	log.Debugf("Accepted block %v", blockHash)

	return isMainChain, false, nil
}
