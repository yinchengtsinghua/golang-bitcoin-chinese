
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2018 BTCSuite开发者
//版权所有（c）2015-2018法令开发商
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package blockchain

import (
	"container/list"
	"fmt"
	"sync"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

const (
//MaxOrphanBlocks是可以
//排队。
	maxOrphanBlocks = 100
)

//块定位器用于帮助定位特定的块。的算法
//构建块定位器是以相反的顺序添加哈希，直到
//到达Genesis区块。为了保留定位器哈希列表
//
//添加散列，然后将每个循环迭代的步骤加倍到
//以指数形式减少散列数作为距离的函数
//
//
//例如，假设有侧链的区块链如下所示：
//《创世纪》->1->2->->15->16->17->18
//> -16A-＞17A
//
//块17a的块定位器是块的散列：
//[17A 16A 15 14 13 12 11 10 9 8 7 6 4创世纪]
type BlockLocator []*chainhash.Hash

//孤立块表示我们还没有父块的块。它
//是一个普通块加上一个过期时间以防止缓存孤立块
//永远。
type orphanBlock struct {
	block      *btcutil.Block
	expiration time.Time
}

//BestState包含有关当前最佳块的信息和其他信息
//从
//当前最佳块。
//
//BestSnapshot方法可用于获取此信息的访问权限
//以并发安全的方式，数据不会从下更改。
//按照函数名的含义，当链状态发生更改时调用方。
//但是，返回的快照必须被视为不可变的，因为它是
//由所有呼叫者共享。
type BestState struct {
Hash        chainhash.Hash //块的哈希值。
Height      int32          //块的高度。
Bits        uint32         //块的难点部分。
BlockSize   uint64         //块的大小。
BlockWeight uint64         //块的重量。
NumTxns     uint64         //块中txn的数目。
TotalTxns   uint64         //链中TXN的总数。
MedianTime  time.Time      //根据calcpastmediantime确定的中间时间。
}

//NewbestState为给定参数返回新的最佳统计实例。
func newBestState(node *blockNode, blockSize, blockWeight, numTxns,
	totalTxns uint64, medianTime time.Time) *BestState {

	return &BestState{
		Hash:        node.hash,
		Height:      node.height,
		Bits:        node.bits,
		BlockSize:   blockSize,
		BlockWeight: blockWeight,
		NumTxns:     numTxns,
		TotalTxns:   totalTxns,
		MedianTime:  medianTime,
	}
}

//区块链提供使用比特币区块链的功能。
//它包括拒绝重复块、确保块等功能
//遵循所有规则、孤立处理、检查点处理和最佳链
//选择与重组。
type BlockChain struct {
//以下字段是在创建实例时设置的，不能
//之后再更改，因此无需使用
//单独互斥。
	checkpoints         []chaincfg.Checkpoint
	checkpointsByHeight map[int32]*chaincfg.Checkpoint
	db                  database.DB
	chainParams         *chaincfg.Params
	timeSource          MedianTimeSource
	sigCache            *txscript.SigCache
	indexManager        IndexManager
	hashCache           *txscript.HashCache

//以下字段是根据提供的链计算的
//参数。它们也在创建实例时设置，并且
//以后不能更改，因此无需使用
//一个单独的互斥体。
minRetargetTimespan int64 //目标时间跨度/调整系数
maxRetargetTimespan int64 //目标时间跨度*调整系数
blocksPerRetarget   int32 //每个块的目标时间跨度/目标时间

//chainlock保护对大多数
//此结构中低于此点的字段。
	chainLock sync.RWMutex

//
//他们自己的锁，但是他们也经常受到链条的保护
//锁定以帮助在处理块时防止逻辑争用。
//
//索引将整个块索引存储在内存中。块索引是
//树形结构。
//
//BestChain通过使用
//块索引中的高效链视图。
	index     *blockIndex
	bestChain *chainView

//这些字段与处理孤立块相关。他们是
//由链锁和孤立锁的组合保护。
	orphanLock   sync.RWMutex
	orphans      map[chainhash.Hash]*orphanBlock
	prevOrphans  map[chainhash.Hash][]*orphanBlock
	oldestOrphan *orphanBlock

//这些字段与检查点处理相关。它们受到保护
//通过链锁。
	nextCheckpoint *chaincfg.Checkpoint
	checkpointNode *blockNode

//状态被用作缓存信息的一种相当有效的方法。
//关于在以下情况下返回给调用方的当前最佳链状态：
//请求。它的工作原理是MVCC，因此任何时候
//新块成为最佳块，状态指针替换为
//新结构和旧状态保持不变。这样，
//多个调用方可以指向不同的最佳链状态。
//对于大多数呼叫者来说，这是可以接受的，因为状态
//在特定时间点查询。
//
//此外，一些字段存储在数据库中，因此
//链状态可以在加载时快速重建。
	stateLock     sync.RWMutex
	stateSnapshot *BestState

//以下缓存用于有效地跟踪
//每个规则的当前部署阈值状态将更改部署。
//
//此信息存储在数据库中，因此可以快速
//带载重建。
//
//警告缓存缓存块的当前部署阈值状态
//在每个**可能的**部署中。这是用来
//在投票表决新的未识别规则更改时检测和/或
//已被激活，如在旧版本的
//软件正在使用中
//
//DeploymentCaches缓存的当前部署阈值状态
//每个活动定义的部署中的块。
	warningCaches    []thresholdStateCache
	deploymentCaches []thresholdStateCache

//以下字段用于确定某些警告是否具有
//已经显示。
//
//未知规则是指由于未知规则
//激活。
//
//UnknownInversionsWarned是指由未知版本引起的警告。
//正在开采。
	unknownRulesWarned    bool
	unknownVersionsWarned bool

//“通知”字段存储要在其上执行的回调切片
//某些区块链事件。
	notificationsLock sync.RWMutex
	notifications     []NotificationCallback
}

//HaveBlock返回链实例是否具有表示的块
//通过传递的哈希。这包括检查一个块可以
//就像主链的一部分，在侧链上，或者在孤儿池中。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) HaveBlock(hash *chainhash.Hash) (bool, error) {
	exists, err := b.blockExists(hash)
	if err != nil {
		return false, err
	}
	return exists || b.IsKnownOrphan(hash), nil
}

//is known orphan返回传递的哈希当前是否为已知的孤立哈希。
//请记住，只有少数孤儿
//时间有限，因此不能将此函数用作绝对值
//测试块是否为孤立块的方法。一个完整的块（而不是
//必须将其哈希）传递给processBlock。但是，打电话
//带有已存在的孤立进程块会导致错误，因此
//函数提供了一种机制，使调用者能够智能地检测*最近的*
//复制孤立对象并做出相应的反应。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) IsKnownOrphan(hash *chainhash.Hash) bool {
//保护并发访问。使用只读锁
//读者可以在不阻塞彼此的情况下进行查询。
	b.orphanLock.RLock()
	_, exists := b.orphans[*hash]
	b.orphanLock.RUnlock()

	return exists
}

//GetOrphanRoot返回所提供哈希的链头
//孤立块的地图。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) GetOrphanRoot(hash *chainhash.Hash) *chainhash.Hash {
//保护并发访问。使用只读锁
//读者可以在不阻塞彼此的情况下进行查询。
	b.orphanLock.RLock()
	defer b.orphanLock.RUnlock()

//当每个孤立块的父级为
//是个孤儿。
	orphanRoot := hash
	prevHash := hash
	for {
		orphan, exists := b.orphans[*prevHash]
		if !exists {
			break
		}
		orphanRoot = prevHash
		prevHash = &orphan.block.MsgBlock().Header.PrevBlock
	}

	return orphanRoot
}

//removeorphanblock从孤立池中删除传递的孤立块，并
//上一个孤立索引。
func (b *BlockChain) removeOrphanBlock(orphan *orphanBlock) {
//保护并发访问。
	b.orphanLock.Lock()
	defer b.orphanLock.Unlock()

//从孤立池中删除孤立块。
	orphanHash := orphan.block.Hash()
	delete(b.orphans, *orphanHash)

//也从以前的孤立索引中删除引用。索引
//这里的for循环专门用于一个范围，因为范围不是
//在每次迭代中重新评估切片，也不调整索引
//对于修改的切片。
	prevHash := &orphan.block.MsgBlock().Header.PrevBlock
	orphans := b.prevOrphans[*prevHash]
	for i := 0; i < len(orphans); i++ {
		hash := orphans[i].block.Hash()
		if hash.IsEqual(orphanHash) {
			copy(orphans[i:], orphans[i+1:])
			orphans[len(orphans)-1] = nil
			orphans = orphans[:len(orphans)-1]
			i--
		}
	}
	b.prevOrphans[*prevHash] = orphans

//如果不再有任何孤立项，请完全删除映射项
//这取决于父哈希。
	if len(b.prevOrphans[*prevHash]) == 0 {
		delete(b.prevOrphans, *prevHash)
	}
}

//AddOrphanBlock添加传递的块（已确定为
//调用此函数之前的孤立函数）。它懒洋洋地清洗
//清除所有过期的块，这样就不需要运行单独的清除轮询器。
//它还对未完成孤儿的数量施加了最大限制。
//阻止，如果限制为
//超过。
func (b *BlockChain) addOrphanBlock(block *btcutil.Block) {
//删除过期的孤立块。
	for _, oBlock := range b.orphans {
		if time.Now().After(oBlock.expiration) {
			b.removeOrphanBlock(oBlock)
			continue
		}

//更新最旧的孤立块指针，以便将其丢弃
//万一孤儿池满了。
		if b.oldestOrphan == nil || oBlock.expiration.Before(b.oldestOrphan.expiration) {
			b.oldestOrphan = oBlock
		}
	}

//限制孤立块以防止内存耗尽。
	if len(b.orphans)+1 > maxOrphanBlocks {
//把最老的孤儿带走，为新孤儿腾出地方。
		b.removeOrphanBlock(b.oldestOrphan)
		b.oldestOrphan = nil
	}

//保护并发访问。这是故意的
//接近顶部，因为removeOrphanBlock自己进行锁定，并且
//范围迭代器不会因删除映射项而失效。
	b.orphanLock.Lock()
	defer b.orphanLock.Unlock()

//将块插入到具有过期时间的孤立映射中
//1小时后。
	expiration := time.Now().Add(time.Hour)
	oBlock := &orphanBlock{
		block:      block,
		expiration: expiration,
	}
	b.orphans[*block.Hash()] = oBlock

//添加到上一个哈希查找索引以更快地查找依赖项。
	prevHash := &block.MsgBlock().Header.PrevBlock
	b.prevOrphans[*prevHash] = append(b.prevOrphans[*prevHash], oBlock)
}

//SequenceLock表示以秒为单位转换的相对锁定时间，以及
//事务输入的相对锁定时间的绝对块高度。
//根据SequenceLock，确认参考输入后
//在一个块中，输入的事务支出可以包括在
//在“秒”（根据过去的中值时间）后阻塞，或在
//已达到“blockheight”。
type SequenceLock struct {
	Seconds     int64
	BlockHeight int32
}

//CalcSequenceLock为传递的
//使用传递的utxoview获取过去中间时间的事务
//对于包含事务的引用输入的块
//内。生成的SequenceLock锁可以与
//块高度，并调整中间块时间以确定是否所有输入
//交易中引用的已达到足够的到期日，允许
//要包含在块中的候选事务。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) CalcSequenceLock(tx *btcutil.Tx, utxoView *UtxoViewpoint, mempool bool) (*SequenceLock, error) {
	b.chainLock.Lock()
	defer b.chainLock.Unlock()

	return b.calcSequenceLock(b.bestChain.Tip(), tx, utxoView, mempool)
}

//CalcSequenceLock计算传递的
//交易。有关详细信息，请参阅导出的版本calcSequenceLock。
//
//必须在保持链状态锁的情况下调用此函数（用于写入）。
func (b *BlockChain) calcSequenceLock(node *blockNode, tx *btcutil.Tx, utxoView *UtxoViewpoint, mempool bool) (*SequenceLock, error) {
//每个相对锁类型的值-1表示相对时间
//允许事务包含在块中的锁值
//在任何给定的高度或时间。此值作为相对值返回
//BIP 68被禁用或尚未被禁用时的锁定时间
//激活。
	sequenceLock := &SequenceLock{Seconds: -1, BlockHeight: -1}

//序列锁语义对于事务始终是活动的。
//在mempool中。
	csvSoftforkActive := mempool

//如果我们正在执行块验证，那么我们需要查询bip9
//状态。
	if !csvSoftforkActive {
//获取最新的bip9版本位状态
//csv软件包软分叉部署。序列的依从性
//锁定取决于当前的软分叉状态。
		csvState, err := b.deploymentState(node.parent, chaincfg.DeploymentCSV)
		if err != nil {
			return nil, err
		}
		csvSoftforkActive = csvState == ThresholdActive
	}

//如果事务的版本小于2，而BIP 68尚未
//被激活，然后序列锁被禁用。此外，
//序列锁不适用于coinbase事务，因此，我们
//返回-1的序列锁定值，指示此事务
//可以包括在一个块在任何给定的高度或时间。
	mTx := tx.MsgTx()
	sequenceLockActive := mTx.Version >= 2 && csvSoftforkActive
	if !sequenceLockActive || IsCoinBase(tx) {
		return sequenceLock, nil
	}

//从传递的blocknode的POV获取下一个高度以用于
//内存池中存在输入。
	nextHeight := node.height + 1

	for txInIndex, txIn := range mTx.TxIn {
		utxo := utxoView.LookupEntry(txIn.PreviousOutPoint)
		if utxo == nil {
			str := fmt.Sprintf("output %v referenced from "+
				"transaction %s:%d either does not exist or "+
				"has already been spent", txIn.PreviousOutPoint,
				tx.Hash(), txInIndex)
			return sequenceLock, ruleError(ErrMissingTxOut, str)
		}

//如果输入高度设置为mempool高度，则我们
//假设事务在以下情况下进入下一个块：
//正在评估其序列块。
		inputHeight := utxo.BlockHeight()
		if inputHeight == 0x7fffffff {
			inputHeight = nextHeight
		}

//给定序列号，我们应用相对时间锁
//蒙版以获得之前所需的时间锁定增量
//这个输入可以使用。
		sequenceNum := txIn.Sequence
		relativeLock := int64(sequenceNum & wire.SequenceLockTimeMask)

		switch {
//此输入的相对时间锁定被禁用，因此我们可以
//跳过任何进一步的计算。
		case sequenceNum&wire.SequenceLockTimeDisabled == wire.SequenceLockTimeDisabled:
			continue
		case sequenceNum&wire.SequenceLockTimeIsSeconds == wire.SequenceLockTimeIsSeconds:
//此输入需要表示的相对时间锁
//几秒钟后就可以用完了。因此，我们
//需要先查询块，然后再查询
//其中包括了这个输入，所以我们可以
//计算之前块的过去中间时间
//包含此引用输出的。
			prevInputHeight := inputHeight - 1
			if prevInputHeight < 0 {
				prevInputHeight = 0
			}
			blockNode := node.Ancestor(prevInputHeight)
			medianTime := blockNode.CalcPastMedianTime()

//根据BIP 68定义的基于时间的相对时间锁
//时间粒度为relativeLockSeconds，所以
//我们左移这个数，转换成
//适当的相对时间锁定。我们也从中减去一
//保持原始锁定时间的相对锁定
//语义学。
			timeLockSeconds := (relativeLock << wire.SequenceLockTimeGranularity) - 1
			timeLock := medianTime.Unix() + timeLockSeconds
			if timeLock > sequenceLock.Seconds {
				sequenceLock.Seconds = timeLock
			}
		default:
//表示此输入的相对锁定时间
//因此我们计算相对偏移量
//输入的高度作为其转换后的绝对值
//锁定时间。我们从相对锁定中减去一个
//以维护原始的锁时间语义。
			blockHeight := inputHeight + int32(relativeLock-1)
			if blockHeight > sequenceLock.BlockHeight {
				sequenceLock.BlockHeight = blockHeight
			}
		}
	}

	return sequenceLock, nil
}

//LockTimeToSequence将传递的相对锁定时间转换为序列
//根据BIP-68编号。
//参见：https://github.com/bitcoin/bips/blob/master/bip-0068.mediawiki
//*（兼容性）
func LockTimeToSequence(isSeconds bool, locktime uint32) uint32 {
//如果我们用块表示相对锁定时间，那么
//相应的序列号只是所需的输入年龄。
	if !isSeconds {
		return locktime
	}

//设置第22位，表示锁定时间以秒为单位，然后
//由于时间粒度在
//512秒间隔（2^9）。这将导致最大锁定时间为
//33553920秒，或1.1年。
	return wire.SequenceLockTimeIsSeconds |
		locktime>>wire.SequenceLockTimeGranularity
}

//getReorganizeNodes查找主链和传递的
//并返回需要从中分离的块节点列表
//需要附加到的主链和块节点列表
//叉点（分离后将作为主链的末端）
//返回的块节点列表），以便重新组织链，以便
//传递的节点是主链的新端。如果
//传递的节点不在侧链上。
//
//此函数可以在不刷新的情况下修改块索引中的节点状态。
//
//必须在保持链状态锁的情况下调用此函数（用于读取）。
func (b *BlockChain) getReorganizeNodes(node *blockNode) (*list.List, *list.List) {
	attachNodes := list.New()
	detachNodes := list.New()

//不要重新组织为已知的无效链。祖先比
//直接父级在下面进行了检查，但这是一个快速检查
//更多不必要的工作。
	if b.index.NodeStatus(node.parent).KnownInvalid() {
		b.index.SetStatusFlags(node, statusInvalidAncestor)
		return detachNodes, attachNodes
	}

//找到分叉点（如果有），将每个块添加到节点列表中
//连接到主树上。按相反的顺序将它们推到列表中
//因此，在迭代列表时，它们是以适当的顺序附加的
//后来。
	forkNode := b.bestChain.FindFork(node)
	invalidChain := false
	for n := node; n != nil && n != forkNode; n = n.parent {
		if b.index.NodeStatus(n).KnownInvalid() {
			invalidChain = true
			break
		}
		attachNodes.PushFront(n)
	}

//如果节点的任何祖先无效，请展开AttachNodes，标记
//每一个都是无效的，以备将来参考。
	if invalidChain {
		var next *list.Element
		for e := attachNodes.Front(); e != nil; e = next {
			next = e.Next()
			n := attachNodes.Remove(e).(*blockNode)
			b.index.SetStatusFlags(n, statusInvalidAncestor)
		}
		return detachNodes, attachNodes
	}

//从主链的末端开始向后工作，直到
//将每个块添加到要分离的节点列表中的公共祖先
//主链。
	for n := b.bestChain.Tip(); n != nil && n != forkNode; n = n.parent {
		detachNodes.PushBack(n)
	}

	return detachNodes, attachNodes
}

//ConnectBlock句柄将传递的节点/块连接到主节点的末尾
//（最好）链。
//
//此传递的utxo视图必须具有块花费标记的所有引用txo
//随着时间的推移和所有新的txo块创建添加到它。此外，
//传递的stxos切片必须用
//使用过的TXOS。使用此方法是因为连接验证
//必须在调用此函数之前发生，需要相同的详细信息，因此
//重复这一过程是没有效率的。
//
//必须在保持链状态锁的情况下调用此函数（用于写入）。
func (b *BlockChain) connectBlock(node *blockNode, block *btcutil.Block,
	view *UtxoViewpoint, stxos []SpentTxOut) error {

//确保它延伸到最好的链条末端。
	prevHash := &block.MsgBlock().Header.PrevBlock
	if !prevHash.IsEqual(&b.bestChain.Tip().hash) {
		return AssertError("connectBlock must be called with a block " +
			"that extends the main chain")
	}

//请检查是否提供了正确数量的STXO。
	if len(stxos) != countSpentOutputs(block) {
		return AssertError("connectBlock called with inconsistent " +
			"spent transaction out information")
	}

//在链
//电流。
	if b.isCurrent() {
//如果任何未知的新规则即将激活或
//已经激活。
		if err := b.warnUnknownRuleActivations(node); err != nil {
			return err
		}

//如果最后一个块的百分比足够高，则发出警告
//意外的版本。
		if err := b.warnUnknownVersions(node); err != nil {
			return err
		}
	}

//在更新最佳状态之前，将任何块状态更改写入数据库。
	err := b.index.flushToDB()
	if err != nil {
		return err
	}

//生成新的最佳状态快照，该快照将用于更新
//数据库和以后的内存（如果所有数据库更新都成功）。
	b.stateLock.RLock()
	curTotalTxns := b.stateSnapshot.TotalTxns
	b.stateLock.RUnlock()
	numTxns := uint64(len(block.MsgBlock().Transactions))
	blockSize := uint64(block.MsgBlock().SerializeSize())
	blockWeight := uint64(GetBlockWeight(block))
	state := newBestState(node, blockSize, blockWeight, numTxns,
		curTotalTxns+numTxns, node.CalcPastMedianTime())

//自动将信息插入数据库。
	err = b.db.Update(func(dbTx database.Tx) error {
//更新最佳块状态。
		err := dbPutBestState(dbTx, state, node.workSum)
		if err != nil {
			return err
		}

//将块哈希和高度添加到跟踪的块索引中
//主链。
		err = dbPutBlockIndex(dbTx, block.Hash(), node.height)
		if err != nil {
			return err
		}

//使用utxo视图的状态更新utxo集。这个
//需要删除所有已花费的utxo并添加新的
//由块创建的。
		err = dbPutUtxoView(dbTx, view)
		if err != nil {
			return err
		}

//通过添加以下项的记录来更新交易记录支出日记帐：
//包含它花费的所有TxO的块。
		err = dbPutSpendJournalEntry(dbTx, block.Hash(), stxos)
		if err != nil {
			return err
		}

//允许索引管理器调用当前活动的
//与块连接的可选索引，以便
//相应地更新自己。
		if b.indexManager != nil {
			err := b.indexManager.ConnectBlock(dbTx, block, stxos)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

//修剪完全用完的条目，并将视图中的所有条目标记为未修改
//现在修改已经提交到数据库。
	view.commit()

//这个节点现在是最佳链的末端。
	b.bestChain.SetTip(node)

//更新最佳块的状态。注意这将如何替换
//整个结构，而不是更新现有结构。这是有效的
//允许旧版本充当调用方可以使用的快照
//不需要在这段时间内保持锁的自由。见
//有关状态变量的注释以了解更多详细信息。
	b.stateLock.Lock()
	b.stateSnapshot = state
	b.stateLock.Unlock()

//通知调用方块已连接到主链。
//调用者通常希望对诸如
//更新钱包。
	b.chainLock.Unlock()
	b.sendNotification(NTBlockConnected, block)
	b.chainLock.Lock()

	return nil
}

//disconnectBlock句柄从
//主链（最好的）。
//
//必须在保持链状态锁的情况下调用此函数（用于写入）。
func (b *BlockChain) disconnectBlock(node *blockNode, block *btcutil.Block, view *UtxoViewpoint) error {
//确保要断开的节点是最佳链的末端。
	if !node.hash.IsEqual(&b.bestChain.Tip().hash) {
		return AssertError("disconnectBlock must be called with the " +
			"block at the end of the main chain")
	}

//加载前一个块，因为下面需要它的一些详细信息。
	prevNode := node.parent
	var prevBlock *btcutil.Block
	err := b.db.View(func(dbTx database.Tx) error {
		var err error
		prevBlock, err = dbFetchBlockByNode(dbTx, prevNode)
		return err
	})
	if err != nil {
		return err
	}

//在更新最佳状态之前，将任何块状态更改写入数据库。
	err = b.index.flushToDB()
	if err != nil {
		return err
	}

//生成新的最佳状态快照，该快照将用于更新
//数据库和以后的内存（如果所有数据库更新都成功）。
	b.stateLock.RLock()
	curTotalTxns := b.stateSnapshot.TotalTxns
	b.stateLock.RUnlock()
	numTxns := uint64(len(prevBlock.MsgBlock().Transactions))
	blockSize := uint64(prevBlock.MsgBlock().SerializeSize())
	blockWeight := uint64(GetBlockWeight(prevBlock))
	newTotalTxns := curTotalTxns - uint64(len(block.MsgBlock().Transactions))
	state := newBestState(prevNode, blockSize, blockWeight, numTxns,
		newTotalTxns, prevNode.CalcPastMedianTime())

	err = b.db.Update(func(dbTx database.Tx) error {
//更新最佳块状态。
		err := dbPutBestState(dbTx, state, node.workSum)
		if err != nil {
			return err
		}

//从块索引中删除块哈希和高度
//跟踪主链。
		err = dbRemoveBlockIndex(dbTx, block.Hash(), node.height)
		if err != nil {
			return err
		}

//使用utxo视图的状态更新utxo集。这个
//需要恢复所有使用过的utxo并移除新的
//由块创建的。
		err = dbPutUtxoView(dbTx, view)
		if err != nil {
			return err
		}

//在删除此备份的支出日记条目之前，
//我们会照原样取来，以便索引器在需要时可以使用。
		stxos, err := dbFetchSpendJournalEntry(dbTx, block)
		if err != nil {
			return err
		}

//通过删除记录更新交易记录支出日记帐
//包含块花费的所有TxO。
		err = dbRemoveSpendJournalEntry(dbTx, block.Hash())
		if err != nil {
			return err
		}

//允许索引管理器调用当前活动的
//块断开连接后的可选索引，因此
//可以相应地更新自己。
		if b.indexManager != nil {
			err := b.indexManager.DisconnectBlock(dbTx, block, stxos)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

//修剪完全用完的条目，并将视图中的所有条目标记为未修改
//现在修改已经提交到数据库。
	view.commit()

//此节点的父节点现在是最佳链的结尾。
	b.bestChain.SetTip(node.parent)

//更新最佳块的状态。注意这将如何替换
//整个结构，而不是更新现有结构。这是有效的
//允许旧版本充当调用方可以使用的快照
//不需要在这段时间内保持锁的自由。见
//有关状态变量的注释以了解更多详细信息。
	b.stateLock.Lock()
	b.stateSnapshot = state
	b.stateLock.Unlock()

//通知调用方块已断开与主块的连接
//链。调用者通常希望对诸如
//更新钱包。
	b.chainLock.Unlock()
	b.sendNotification(NTBlockDisconnected, block)
	b.chainLock.Lock()

	return nil
}

//COUNTSPENTOUTPUTS返回传递的块开销的utxo数。
func countSpentOutputs(block *btcutil.Block) int {
//排除coinbase事务，因为它不能花费任何东西。
	var numSpent int
	for _, tx := range block.Transactions()[1:] {
		numSpent += len(tx.MsgTx().TxIn)
	}
	return numSpent
}

//重新组织链通过断开块链中的节点重新组织块链
//分离节点列表并连接附加列表中的节点。它期待
//列表的顺序已经正确，并且与
//当前最佳链的结尾。具体来说，正在
//断开连接的顺序必须相反（考虑将它们从
//所连接的链）和节点必须按向前顺序排列。
//（想想把它们推到链条的末端）。
//
//此函数可以在不刷新的情况下修改块索引中的节点状态。
//
//必须在保持链状态锁的情况下调用此函数（用于写入）。
func (b *BlockChain) reorganizeChain(detachNodes, attachNodes *list.List) error {
//如果没有提供重新组织节点，则不执行任何操作。
	if detachNodes.Len() == 0 && attachNodes.Len() == 0 {
		return nil
	}

//确保提供的节点与当前最佳链匹配。
	tip := b.bestChain.Tip()
	if detachNodes.Len() != 0 {
		firstDetachNode := detachNodes.Front().Value.(*blockNode)
		if firstDetachNode.hash != tip.hash {
			return AssertError(fmt.Sprintf("reorganize nodes to detach are "+
				"not for the current best chain -- first detach node %v, "+
				"current chain %v", &firstDetachNode.hash, &tip.hash))
		}
	}

//确保提供的节点用于同一分叉点。
	if attachNodes.Len() != 0 && detachNodes.Len() != 0 {
		firstAttachNode := attachNodes.Front().Value.(*blockNode)
		lastDetachNode := detachNodes.Back().Value.(*blockNode)
		if firstAttachNode.parent.hash != lastDetachNode.parent.hash {
			return AssertError(fmt.Sprintf("reorganize nodes do not have the "+
				"same fork point -- first attach parent %v, last detach "+
				"parent %v", &firstAttachNode.parent.hash,
				&lastDetachNode.parent.hash))
		}
	}

//跟踪新旧最好的链头。
	oldBest := tip
	newBest := tip

//要分离的所有块以及所需的相关支出日记条目
//要在断开连接的块中取消挂起事务输出，必须
//在下面的REORG检查阶段从数据库加载
//然后，在进行实际的数据库更新时，需要再次使用它们。
//不执行两次加载，而是将加载的数据缓存到这些切片中。
	detachBlocks := make([]*btcutil.Block, 0, detachNodes.Len())
	detachSpentTxOuts := make([][]SpentTxOut, 0, detachNodes.Len())
	attachBlocks := make([]*btcutil.Block, 0, attachNodes.Len())

//断开所有挡块，使其回到叉点。这个
//需要从
//数据库，并使用该信息取消所有已用txo的挂起
//并删除由块创建的utxos。
	view := NewUtxoViewpoint()
	view.SetBestHash(&oldBest.hash)
	for e := detachNodes.Front(); e != nil; e = e.Next() {
		n := e.Value.(*blockNode)
		var block *btcutil.Block
		err := b.db.View(func(dbTx database.Tx) error {
			var err error
			block, err = dbFetchBlockByNode(dbTx, n)
			return err
		})
		if err != nil {
			return err
		}
		if n.hash != *block.Hash() {
			return AssertError(fmt.Sprintf("detach block node hash %v (height "+
				"%v) does not match previous parent block hash %v", &n.hash,
				n.height, block.Hash()))
		}

//加载该块引用的所有非utxo
//已经在视图中。
		err = view.fetchInputUtxos(b.db, block)
		if err != nil {
			return err
		}

//从支出中加载块的所有已用txo
//期刊。
		var stxos []SpentTxOut
		err = b.db.View(func(dbTx database.Tx) error {
			stxos, err = dbFetchSpendJournalEntry(dbTx, block)
			return err
		})
		if err != nil {
			return err
		}

//存储已加载的块并花费日记帐条目供以后使用。
		detachBlocks = append(detachBlocks, block)
		detachSpentTxOuts = append(detachSpentTxOuts, stxos)

		err = view.disconnectTransactions(b.db, block, stxos)
		if err != nil {
			return err
		}

		newBest = n.parent
	}

//仅当存在要附加的节点时才设置分叉点，否则
//块只被断开，因此没有分叉点。
	var forkNode *blockNode
	if attachNodes.Len() > 0 {
		forkNode = newBest
	}

//执行多个检查以验证需要连接的每个块
//可以在不违反任何规则的情况下连接到主链
//没有实际连接块。
//
//注意：这些检查可以在连接块时直接进行。
//但是，这种方法的缺点是，如果这些检查
//断开某些块或连接其他块后失败，所有
//必须回滚操作才能使链返回到
//在违反规则（或其他失败）之前声明。有
//至少有两种方法可以完成回滚，但这两种方法都涉及
//调整链和/或数据库。这个方法捕捉到这些
//修改链之前的问题。
	for e := attachNodes.Front(); e != nil; e = e.Next() {
		n := e.Value.(*blockNode)

		var block *btcutil.Block
		err := b.db.View(func(dbTx database.Tx) error {
			var err error
			block, err = dbFetchBlockByNode(dbTx, n)
			return err
		})
		if err != nil {
			return err
		}

//存储加载的块供以后使用。
		attachBlocks = append(attachBlocks, block)

//跳过检查节点是否已完全验证。虽然
//checkConnectBlock被跳过，我们仍然需要更新utxo
//查看。
		if b.index.NodeStatus(n).KnownValid() {
			err = view.fetchInputUtxos(b.db, block)
			if err != nil {
				return err
			}
			err = view.connectTransactions(block, nil)
			if err != nil {
				return err
			}

			newBest = n
			continue
		}

//请注意，此处不要求提供已用txout的详细信息，并且
//因此不会生成。这是因为州政府
//没有立即写入数据库，因此
//不需要。
//
//如果由于
//违反规则，将其标记为无效，并将其全部标记为
//后代具有无效的祖先。
		err = b.checkConnectBlock(n, block, view, nil)
		if err != nil {
			if _, ok := err.(RuleError); ok {
				b.index.SetStatusFlags(n, statusValidateFailed)
				for de := e.Next(); de != nil; de = de.Next() {
					dn := de.Value.(*blockNode)
					b.index.SetStatusFlags(dn, statusInvalidAncestor)
				}
			}
			return err
		}
		b.index.SetStatusFlags(n, statusValid)

		newBest = n
	}

//重置下面实际连接代码的视图。这是
//必需，因为在检查是否
//REORG将成功，并且连接代码需要
//从连接的每个块的角度来看，视图是有效的，或者
//断开的。
	view = NewUtxoViewpoint()
	view.SetBestHash(&b.bestChain.Tip().hash)

//从主链上断开挡块。
	for i, e := 0, detachNodes.Front(); e != nil; i, e = i+1, e.Next() {
		n := e.Value.(*blockNode)
		block := detachBlocks[i]

//加载该块引用的所有非utxo
//已经在视图中。
		err := view.fetchInputUtxos(b.db, block)
		if err != nil {
			return err
		}

//更新视图以取消所有已用txos的挂起并删除
//块创建的utxos。
		err = view.disconnectTransactions(b.db, block,
			detachSpentTxOuts[i])
		if err != nil {
			return err
		}

//更新数据库和链状态。
		err = b.disconnectBlock(n, block, view)
		if err != nil {
			return err
		}
	}

//连接新的最佳链节。
	for i, e := 0, attachNodes.Front(); e != nil; i, e = i+1, e.Next() {
		n := e.Value.(*blockNode)
		block := attachBlocks[i]

//加载该块引用的所有非utxo
//已经在视图中。
		err := view.fetchInputUtxos(b.db, block)
		if err != nil {
			return err
		}

//更新视图以标记块引用的所有utxo
//作为已用并添加此块创建的所有事务
//对它。另外，提供一个stxo切片，以便使用txout
//生成详细信息。
		stxos := make([]SpentTxOut, 0, countSpentOutputs(block))
		err = view.connectTransactions(block, &stxos)
		if err != nil {
			return err
		}

//更新数据库和链状态。
		err = b.connectBlock(n, block, view, stxos)
		if err != nil {
			return err
		}
	}

//记录链条分叉点和新旧最佳链条
//头。
	if forkNode != nil {
		log.Infof("REORGANIZE: Chain forks at %v (height %v)", forkNode.hash,
			forkNode.height)
	}
	log.Infof("REORGANIZE: Old best chain head was %v (height %v)",
		&oldBest.hash, oldBest.height)
	log.Infof("REORGANIZE: New best chain head is %v (height %v)",
		newBest.hash, newBest.height)

	return nil
}

//connectBestChain句柄将传递的块连接到链，同时
//尊重链的正确选择
//工作证明。在典型的情况下，新的块只扩展主块
//链。然而，它也可能延伸（或创造）侧链（叉）
//这可能最终成为或不可能成为主链取决于哪个叉子
//累积的工作证据最多。它返回是否块
//
//重组成为主链）。
//
//这些标志按如下方式修改此函数的行为：
//-bfastadd：避免了一些昂贵的事务验证操作。
//这在使用检查点时很有用。
//
//必须在保持链状态锁的情况下调用此函数（用于写入）。
func (b *BlockChain) connectBestChain(node *blockNode, block *btcutil.Block, flags BehaviorFlags) (bool, error) {
	fastAdd := flags&BFFastAdd == BFFastAdd

	flushIndexState := func() {
//故意忽略将更新的节点状态写入数据库时出现的错误。如果
//它写不出来，这不是世界末日。如果块是
//有效，我们在connectBlock中刷新，如果该块无效，则
//最糟糕的情况是重新启动后重新验证块。
		if writeErr := b.index.flushToDB(); writeErr != nil {
			log.Warnf("Error flushing block index changes to disk: %v",
				writeErr)
		}
	}

//我们正在用一个新的块扩展主（最佳）链。这就是
//最常见的情况。
	parentHash := &block.MsgBlock().Header.PrevBlock
	if parentHash.IsEqual(&b.bestChain.Tip().hash) {
//跳过检查节点是否已完全验证。
		fastAdd = fastAdd || b.index.NodeStatus(node).KnownValid()

//执行多个检查以验证块是否可以连接
//在不违反任何规则的情况下
//实际连接块。
		view := NewUtxoViewpoint()
		view.SetBestHash(parentHash)
		stxos := make([]SpentTxOut, 0, countSpentOutputs(block))
		if !fastAdd {
			err := b.checkConnectBlock(node, block, view, &stxos)
			if err == nil {
				b.index.SetStatusFlags(node, statusValid)
			} else if _, ok := err.(RuleError); ok {
				b.index.SetStatusFlags(node, statusValidateFailed)
			} else {
				return false, err
			}

			flushIndexState()

			if err != nil {
				return false, err
			}
		}

//在快速添加的情况下，检查块连接的代码
//已跳过，因此utxo视图需要加载引用的
//utxos，使用它们，并添加由创建的新utxos
//这个街区。
		if fastAdd {
			err := view.fetchInputUtxos(b.db, block)
			if err != nil {
				return false, err
			}
			err = view.connectTransactions(block, &stxos)
			if err != nil {
				return false, err
			}
		}

//将挡块连接到主链。
		err := b.connectBlock(node, block, view, stxos)
		if err != nil {
//如果我们被规则错误击中，那么我们会标记
//块的状态为“无效”并刷新
//返回错误前将状态索引到磁盘。
			if _, ok := err.(RuleError); ok {
				b.index.SetStatusFlags(
					node, statusValidateFailed,
				)
			}

			flushIndexState()

			return false, err
		}

//如果这是快速添加，或者此块节点尚未标记为
//有效，然后我们将更新其状态并将状态刷新为
//再次显示磁盘。
		if fastAdd || !b.index.NodeStatus(node).KnownValid() {
			b.index.SetStatusFlags(node, statusValid)
			flushIndexState()
		}

		return true, nil
	}
	if fastAdd {
		log.Warnf("fastAdd set in the side chain case? %v\n",
			block.Hash())
	}

//我们正在扩展（或创建）侧链，但是
//为这条新的侧链工作还不足以使它成为新的链条。
	if node.workSum.Cmp(b.bestChain.Tip().workSum) <= 0 {
//有关块如何分叉链的日志信息。
		fork := b.bestChain.FindFork(node)
		if fork.hash.IsEqual(parentHash) {
			log.Infof("FORK: Block %v forks the chain at height %d"+
				"/block %v, but does not cause a reorganize",
				node.hash, fork.height, fork.hash)
		} else {
			log.Infof("EXTEND FORK: Block %v extends a side chain "+
				"which forks the chain at height %d/block %v",
				node.hash, fork.height, fork.hash)
		}

		return false, nil
	}

//我们正在扩展（或创建）侧链和累积工作
//因为新的边链比旧的最好的边链多，所以这边
//链条需要成为主链。为了实现这一目标，
//找到叉子两边的共同祖先，断开
//从主链形成（现在的）旧叉块，并连接
//形成新链条到主链条的块，从
//普通取消（链条分叉点）。
	detachNodes, attachNodes := b.getReorganizeNodes(node)

//重新组织链条。
	log.Infof("REORGANIZE: Block %v is causing a reorganize.", node.hash)
	err := b.reorganizeChain(detachNodes, attachNodes)

//GetReorganizeNodes或ReorganizeChain可能未保存
//对块索引的更改，因此无论是否存在
//错误。只有当块连接失败时，索引才会变脏，所以
//我们可以忽略任何书写错误。
	if writeErr := b.index.flushToDB(); writeErr != nil {
		log.Warnf("Error flushing block index changes to disk: %v", writeErr)
	}

	return err == nil, err
}

//is current返回链是否相信它是当前的。几个
//因素是用来猜测的，但关键的因素是允许链
//相信现在的情况是：
//-最新块高度在最新检查点之后（如果启用）
//-最新块的时间戳比24小时前更新
//
//必须在保持链状态锁的情况下调用此函数（用于读取）。
func (b *BlockChain) isCurrent() bool {
//如果最新的主（最佳）链高度在
//最新的已知良好检查点（启用检查点时）。
	checkpoint := b.LatestCheckpoint()
	if checkpoint != nil && b.bestChain.Tip().height < checkpoint.Height {
		return false
	}

//如果最新的最佳块在24小时前有时间戳，则不是最新的
//以前。
//
//如果没有报告任何检查，则链似乎是最新的。
//否则。
	minus24Hours := b.timeSource.AdjustedTime().Add(-24 * time.Hour).Unix()
	return b.bestChain.Tip().timestamp >= minus24Hours
}

//is current返回链是否相信它是当前的。几个
//因素是用来猜测的，但关键的因素是允许链
//相信现在的情况是：
//-最新块高度在最新检查点之后（如果启用）
//-最新块的时间戳比24小时前更新
//
//此函数对于并发访问是安全的。
func (b *BlockChain) IsCurrent() bool {
	b.chainLock.RLock()
	defer b.chainLock.RUnlock()

	return b.isCurrent()
}

//BestSnapshot返回有关当前最佳链块和
//当前时间点的相关状态。返回的实例必须是
//被视为不可变的，因为它由所有调用方共享。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) BestSnapshot() *BestState {
	b.stateLock.RLock()
	snapshot := b.stateSnapshot
	b.stateLock.RUnlock()
	return snapshot
}

//HeaderByHash返回由给定哈希或
//如果不存在则出错。请注意，这将返回
//主链条和侧链。
func (b *BlockChain) HeaderByHash(hash *chainhash.Hash) (wire.BlockHeader, error) {
	node := b.index.LookupNode(hash)
	if node == nil {
		err := fmt.Errorf("block %s is not known", hash)
		return wire.BlockHeader{}, err
	}

	return node.Header(), nil
}

//MainchainHasBlock返回具有给定哈希的块是否在
//主链。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) MainChainHasBlock(hash *chainhash.Hash) bool {
	node := b.index.LookupNode(hash)
	return node != nil && b.bestChain.Contains(node)
}

//blocklocatorFromHash返回传递的块哈希的块定位器。
//有关用于创建块定位器的算法的详细信息，请参见块定位器。
//
//除了上面提到的一般算法外，此函数还将
//返回主（最佳）链最新已知尖端的块定位器，如果
//传递的哈希当前未知。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) BlockLocatorFromHash(hash *chainhash.Hash) BlockLocator {
	b.chainLock.RLock()
	node := b.index.LookupNode(hash)
	locator := b.bestChain.blockLocator(node)
	b.chainLock.RUnlock()
	return locator
}

//latest block locator返回最新已知提示的块定位器
//主（最佳）链。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) LatestBlockLocator() (BlockLocator, error) {
	b.chainLock.RLock()
	locator := b.bestChain.BlockLocator(nil)
	b.chainLock.RUnlock()
	return locator, nil
}

//blockheightbyhash返回具有给定哈希的块的高度
//主链。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) BlockHeightByHash(hash *chainhash.Hash) (int32, error) {
	node := b.index.LookupNode(hash)
	if node == nil || !b.bestChain.Contains(node) {
		str := fmt.Sprintf("block %s is not in the main chain", hash)
		return 0, errNotInMainChain(str)
	}

	return node.height, nil
}

//BlockHashByHeight返回块在给定高度的哈希
//主链。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) BlockHashByHeight(blockHeight int32) (*chainhash.Hash, error) {
	node := b.bestChain.NodeByHeight(blockHeight)
	if node == nil {
		str := fmt.Sprintf("no block at height %d exists", blockHeight)
		return nil, errNotInMainChain(str)

	}

	return &node.hash, nil
}

//HeightRange返回给定开始和结束的块哈希范围
//高度。包括起点高度，不包括终点
//高度。末端高度将限制在当前主链高度。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) HeightRange(startHeight, endHeight int32) ([]chainhash.Hash, error) {
//确保要求的高度正常。
	if startHeight < 0 {
		return nil, fmt.Errorf("start height of fetch range must not "+
			"be less than zero - got %d", startHeight)
	}
	if endHeight < startHeight {
		return nil, fmt.Errorf("end height of fetch range must not "+
			"be less than the start height - got start %d, end %d",
			startHeight, endHeight)
	}

//如果起点和终点高度相同，则无需执行任何操作，
//所以现在返回以避免链视图锁定。
	if startHeight == endHeight {
		return nil, nil
	}

//抓住链视图上的锁，以防止它由于
//在构建散列时重新排序。
	b.bestChain.mtx.Lock()
	defer b.bestChain.mtx.Unlock()

//当请求的起始高度在最近的最佳链之后时
//身高，没什么可做的。
	latestHeight := b.bestChain.tip().height
	if startHeight > latestHeight {
		return nil, nil
	}

//将末端高度限制为链条的最新高度。
	if endHeight > latestHeight+1 {
		endHeight = latestHeight + 1
	}

//在指定范围内尽可能多地提取可用数据。
	hashes := make([]chainhash.Hash, 0, endHeight-startHeight)
	for i := startHeight; i < endHeight; i++ {
		hashes = append(hashes, b.bestChain.nodeByHeight(i).hash)
	}
	return hashes, nil
}

//HeightToHashRange返回给定起始高度的块哈希范围
//和结束哈希，两端都包含。哈希值适用于以下所有块：
//
//结束哈希必须属于已知有效的块。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) HeightToHashRange(startHeight int32,
	endHash *chainhash.Hash, maxResults int) ([]chainhash.Hash, error) {

	endNode := b.index.LookupNode(endHash)
	if endNode == nil {
		return nil, fmt.Errorf("no known block header with hash %v", endHash)
	}
	if !b.index.NodeStatus(endNode).KnownValid() {
		return nil, fmt.Errorf("block %v is not yet validated", endHash)
	}
	endHeight := endNode.height

	if startHeight < 0 {
		return nil, fmt.Errorf("start height (%d) is below 0", startHeight)
	}
	if startHeight > endHeight {
		return nil, fmt.Errorf("start height (%d) is past end height (%d)",
			startHeight, endHeight)
	}

	resultsLength := int(endHeight - startHeight + 1)
	if resultsLength > maxResults {
		return nil, fmt.Errorf("number of results (%d) would exceed max (%d)",
			resultsLength, maxResults)
	}

//从endheight向后走到startheight，收集块散列。
	node := endNode
	hashes := make([]chainhash.Hash, resultsLength)
	for i := resultsLength - 1; i >= 0; i-- {
		hashes[i] = node.hash
		node = node.parent
	}
	return hashes, nil
}

//IntervalBlockHashes返回属于的所有块的哈希
//endhash，其中块高度是间隔的正倍数。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) IntervalBlockHashes(endHash *chainhash.Hash, interval int,
) ([]chainhash.Hash, error) {

	endNode := b.index.LookupNode(endHash)
	if endNode == nil {
		return nil, fmt.Errorf("no known block header with hash %v", endHash)
	}
	if !b.index.NodeStatus(endNode).KnownValid() {
		return nil, fmt.Errorf("block %v is not yet validated", endHash)
	}
	endHeight := endNode.height

	resultsLength := int(endHeight) / interval
	hashes := make([]chainhash.Hash, resultsLength)

	b.bestChain.mtx.Lock()
	defer b.bestChain.mtx.Unlock()

	blockNode := endNode
	for index := int(endHeight) / interval; index > 0; index-- {
//使用bestchain chainview在查找交叉时快速查找
//最好的链子。
		blockHeight := int32(index * interval)
		if b.bestChain.contains(blockNode) {
			blockNode = b.bestChain.nodeByHeight(blockHeight)
		} else {
			blockNode = blockNode.Ancestor(blockHeight)
		}

		hashes[index-1] = blockNode.hash
	}

	return hashes, nil
}

//locateinventory返回块中第一个已知块之后的节点
//定位器以及需要到达的后续节点的数量
//提供的停止哈希或提供的最大条目数。
//
//此外，还有两种特殊情况：
//
//-如果没有提供定位器，则停止哈希将被视为
//该块，因此它将返回与停止哈希关联的节点
//如果已知，则为零；如果未知，则为零。
//-如果提供了定位器，但没有已知定位器，则节点将启动
//在Genesis块返回后
//
//这主要是locateblocks和locateheaders的助手函数
//功能。
//
//必须在保持链状态锁的情况下调用此函数（用于读取）。
func (b *BlockChain) locateInventory(locator BlockLocator, hashStop *chainhash.Hash, maxEntries uint32) (*blockNode, uint32) {
//没有块定位器，因此正在请求特定的块
//由停止哈希标识。
	stopNode := b.index.LookupNode(hashStop)
	if len(locator) == 0 {
		if stopNode == nil {
//找不到具有停止哈希的块，因此
//无事可做。
			return nil, 0
		}
		return stopNode, 1
	}

//在主链中查找最新的定位块哈希。在
//如果定位器中没有散列在主链中，则下降
//回到创世纪街区。
	startNode := b.bestChain.Genesis()
	for _, hash := range locator {
		node := b.index.LookupNode(hash)
		if node != nil && b.bestChain.Contains(node) {
			startNode = node
			break
		}
	}

//从最近已知的块之后的块开始。当那里
//不是下一个块，这意味着最近已知的块是
//最好的链条，所以没什么可做的。
	startNode = b.bestChain.Next(startNode)
	if startNode == nil {
		return nil, 0
	}

//计算需要多少条目。
	total := uint32((b.bestChain.Tip().height - startNode.height) + 1)
	if stopNode != nil && b.bestChain.Contains(stopNode) &&
		stopNode.height >= startNode.height {

		total = uint32((stopNode.height - startNode.height) + 1)
	}
	if total > maxEntries {
		total = maxEntries
	}

	return startNode, total
}

//locateBlocks返回块在中第一个已知块之后的哈希值。
//定位符，直到到达提供的停止哈希，或达到提供的
//最大块哈希数。
//
//有关特殊情况的详细信息，请参见导出函数的注释。
//
//必须在保持链状态锁的情况下调用此函数（用于读取）。
func (b *BlockChain) locateBlocks(locator BlockLocator, hashStop *chainhash.Hash, maxHashes uint32) []chainhash.Hash {
//在定位器中的第一个已知块之后查找节点，然后
//在考虑停止哈希的同时，在需要之后的节点总数
//和最大输入。
	node, total := b.locateInventory(locator, hashStop, maxHashes)
	if total == 0 {
		return nil
	}

//填充并返回找到的哈希。
	hashes := make([]chainhash.Hash, 0, total)
	for i := uint32(0); i < total; i++ {
		hashes = append(hashes, node.hash)
		node = b.bestChain.Next(node)
	}
	return hashes
}

//locateBlocks返回块在中第一个已知块之后的哈希值。
//定位符，直到到达提供的停止哈希，或达到提供的
//最大块哈希数。
//
//此外，还有两种特殊情况：
//
//-如果没有提供定位器，则停止哈希将被视为
//该块，因此如果知道stop散列，它将返回stop散列本身，
//如果未知，则为零。
//-如果提供了定位器，但没有已知定位器，则哈希开始
//在Genesis块返回后
//
//此函数对于并发访问是安全的。
func (b *BlockChain) LocateBlocks(locator BlockLocator, hashStop *chainhash.Hash, maxHashes uint32) []chainhash.Hash {
	b.chainLock.RLock()
	hashes := b.locateBlocks(locator, hashStop, maxHashes)
	b.chainLock.RUnlock()
	return hashes
}

//locateheaders返回第一个已知块之后的块的头
//在定位器中，直到达到所提供的停止哈希，或达到所提供的
//块头的最大数目。
//
//有关特殊情况的详细信息，请参见导出函数的注释。
//
//必须在保持链状态锁的情况下调用此函数（用于读取）。
func (b *BlockChain) locateHeaders(locator BlockLocator, hashStop *chainhash.Hash, maxHeaders uint32) []wire.BlockHeader {
//在定位器中的第一个已知块之后查找节点，然后
//在考虑停止哈希的同时，在需要之后的节点总数
//和最大输入。
	node, total := b.locateInventory(locator, hashStop, maxHeaders)
	if total == 0 {
		return nil
	}

//填充并返回找到的头。
	headers := make([]wire.BlockHeader, 0, total)
	for i := uint32(0); i < total; i++ {
		headers = append(headers, node.Header())
		node = b.bestChain.Next(node)
	}
	return headers
}

//locateheaders返回第一个已知块之后的块的头
//在定位器中，直到达到提供的停止哈希，或达到最大值
//Wire.MaxBlockHeadersPermsg头。
//
//此外，还有两种特殊情况：
//
//-如果没有提供定位器，则停止哈希将被视为
//该头，因此它将返回stop散列本身的头
//如果已知，则为零；如果未知，则为零。
//-如果提供了定位器，但没有已知定位器，则头开始
//在Genesis块返回后
//
//此函数对于并发访问是安全的。
func (b *BlockChain) LocateHeaders(locator BlockLocator, hashStop *chainhash.Hash) []wire.BlockHeader {
	b.chainLock.RLock()
	headers := b.locateHeaders(locator, hashStop, wire.MaxBlockHeadersPerMsg)
	b.chainLock.RUnlock()
	return headers
}

//indexManager提供一个通用接口，当块
//连接并断开主链顶端
//支持可选索引的目的。
type IndexManager interface {
//在链初始化期间调用init以允许索引
//管理器初始化自身及其管理的任何索引。这个
//通道参数指定调用者可以接近信号的通道
//进程应该被中断。如果那样的话可以是零
//不需要行为。
	Init(*BlockChain, <-chan struct{}) error

//当新块已连接到
//主链。在一个块中花费的输出集也被传入
//因此索引器可以访问以前的输出脚本输入，如果
//必修的。
	ConnectBlock(database.Tx, *btcutil.Block, []SpentTxOut) error

//断开块从断开时调用disconnectblock
//主链。在
//还返回此块，以便索引器可以清除以前的索引
//此块的状态。
	DisconnectBlock(database.Tx, *btcutil.Block, []SpentTxOut) error
}

//config是指定区块链实例配置的描述符。
type Config struct {
//数据库定义存储块的数据库，并将用于
//存储此包创建的所有元数据，如utxo集。
//
//此字段必填。
	DB database.DB

//中断指定调用方可以关闭的一个通道，以向其发出
//
//应中断数据库迁移。
//
//如果调用方不希望此行为，则此字段可以为零。
	Interrupt <-chan struct{}

//chainParams标识链关联的链参数
//用。
//
//此字段必填。
	ChainParams *chaincfg.Params

//检查点保留应添加到的调用方定义的检查点
//chainParams中的默认检查点。必须对检查点进行排序
//按身高。
//
//如果调用方不希望指定任何
//检查点。
	Checkpoints []chaincfg.Checkpoint

//时间源定义用于诸如
//块处理和确定链是否为当前链。
//
//调用方还应保留对时间源的引用
//并添加来自网络上其他对等方的时间样本，以便
//调整时间使之与其他同行一致。
	TimeSource MedianTimeSource

//sigcache定义验证时要使用的签名缓存
//签名。当个人
//在将交易包括在
//一种块，如通常通过事务存储器池进行的操作。
//
//如果调用方不想使用
//签名缓存。
	SigCache *txscript.SigCache

//索引管理器定义在初始化
//链条和连接和断开块。
//
//如果调用方不希望使用
//索引管理器。
	IndexManager IndexManager

//hash cache定义事务哈希中间状态缓存，以便在
//正在验证事务。这个缓存有很大的潜力
//加速事务验证，因为重新使用预先计算的
//中间状态消除了O（n^2）验证的复杂性，因为
//叹息信号旗。
//
//如果调用方不想使用
//签名缓存。
	HashCache *txscript.HashCache
}

//new使用提供的配置详细信息返回区块链实例。
func New(config *Config) (*BlockChain, error) {
//强制必需的配置字段。
	if config.DB == nil {
		return nil, AssertError("blockchain.New database is nil")
	}
	if config.ChainParams == nil {
		return nil, AssertError("blockchain.New chain parameters nil")
	}
	if config.TimeSource == nil {
		return nil, AssertError("blockchain.New timesource is nil")
	}

//从提供的检查点按高度生成检查点地图
//并断言所提供的检查点根据需要按高度排序。
	var checkpointsByHeight map[int32]*chaincfg.Checkpoint
	var prevCheckpointHeight int32
	if len(config.Checkpoints) > 0 {
		checkpointsByHeight = make(map[int32]*chaincfg.Checkpoint)
		for i := range config.Checkpoints {
			checkpoint := &config.Checkpoints[i]
			if checkpoint.Height <= prevCheckpointHeight {
				return nil, AssertError("blockchain.New " +
					"checkpoints are not sorted by height")
			}

			checkpointsByHeight[checkpoint.Height] = checkpoint
			prevCheckpointHeight = checkpoint.Height
		}
	}

	params := config.ChainParams
	targetTimespan := int64(params.TargetTimespan / time.Second)
	targetTimePerBlock := int64(params.TargetTimePerBlock / time.Second)
	adjustmentFactor := params.RetargetAdjustmentFactor
	b := BlockChain{
		checkpoints:         config.Checkpoints,
		checkpointsByHeight: checkpointsByHeight,
		db:                  config.DB,
		chainParams:         params,
		timeSource:          config.TimeSource,
		sigCache:            config.SigCache,
		indexManager:        config.IndexManager,
		minRetargetTimespan: targetTimespan / adjustmentFactor,
		maxRetargetTimespan: targetTimespan * adjustmentFactor,
		blocksPerRetarget:   int32(targetTimespan / targetTimePerBlock),
		index:               newBlockIndex(config.DB, params),
		hashCache:           config.HashCache,
		bestChain:           newChainView(nil),
		orphans:             make(map[chainhash.Hash]*orphanBlock),
		prevOrphans:         make(map[chainhash.Hash][]*orphanBlock),
		warningCaches:       newThresholdCaches(vbNumBits),
		deploymentCaches:    newThresholdCaches(chaincfg.DefinedDeployments),
	}

//从传递的数据库初始化链状态。当数据库
//还不包含任何链状态，包括它和链状态
//将初始化为仅包含Genesis块。
	if err := b.initChainState(); err != nil {
		return nil, err
	}

//根据需要对各种链特定的存储桶执行任何升级。
	if err := b.maybeUpgradeDbBuckets(config.Interrupt); err != nil {
		return nil, err
	}

//初始化并捕获所有当前活动的可选索引
//根据需要。
	if config.IndexManager != nil {
		err := config.IndexManager.Init(&b, config.Interrupt)
		if err != nil {
			return nil, err
		}
	}

//初始化规则更改阈值状态缓存。
	if err := b.initThresholdCaches(); err != nil {
		return nil, err
	}

	bestNode := b.bestChain.Tip()
	log.Infof("Chain state (height %d, hash %v, totaltx %d, work %v)",
		bestNode.height, bestNode.hash, b.stateSnapshot.TotalTxns,
		bestNode.workSum)

	return &b, nil
}
