
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015-2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package blockchain

import (
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcd/wire"
)

//BlockStatus是一个位字段，表示块的验证状态。
type blockStatus byte

const (
//statusDatastored指示块的有效负载存储在磁盘上。
	statusDataStored blockStatus = 1 << iota

//statusvalid表示块已完全验证。
	statusValid

//statusvalidatefailed指示块验证失败。
	statusValidateFailed

//statusinvalidancestor表示块的一个祖先
//验证失败，因此块也是无效的。
	statusInvalidAncestor

//statusNo表示块没有设置验证状态标志。
//
//
	statusNone blockStatus = 0
)

//HaveData返回完整块数据是否存储在数据库中。这个
//对于只下载头的块节点，或
//保持。
func (status blockStatus) HaveData() bool {
	return status&statusDataStored != 0
}

//known valid返回块是否已知有效。这个会回来的
//对于尚未完全验证的有效块，为false。
func (status blockStatus) KnownValid() bool {
	return status&statusValid != 0
}

//known invalid返回块是否已知无效。这可能是
//因为块本身未通过验证或其任何祖先是
//无效。对于未经验证的无效块，这将返回false。
//仍然无效。
func (status blockStatus) KnownInvalid() bool {
	return status&(statusValidateFailed|statusInvalidAncestor) != 0
}

//blocknode表示块链中的块，主要用于
//帮助选择最好的链条作为主链条。主链是
//存储到块数据库中。
type blockNode struct {
//注意：添加、删除或修改
//不应考虑更改此结构中的定义
//它如何影响64位平台上的对齐。当前订单是
//专门设计以使填充物最小化。将会有
//内存中有几十万个这样的内存，所以有几个额外的字节
//填充相加。

//Parent是此节点的父块。
	parent *blockNode

//hash是块的双sha 256。
	hash chainhash.Hash

//worksum是链中的总工作量，包括
//这个节点。
	workSum *big.Int

//高度是区块链中的位置。
	height int32

//块头中的一些字段有助于最佳链选择和
//正在从内存重建头。这些必须被视为
//不可变的，并被故意排序以避免在64位上填充
//平台。
	version    int32
	bits       uint32
	nonce      uint32
	timestamp  int64
	merkleRoot chainhash.Hash

//状态是表示块的验证状态的位字段。这个
//与其他字段不同，状态字段可以写入，因此应该写入
//只能使用上的并发安全nodestatus方法访问
//将节点添加到全局索引后，块索引。
	status blockStatus
}

//initblocknode从给定的头节点和父节点初始化块节点，
//从父级上的相应字段计算高度和工作空间。
//此函数对于并发访问不安全。只有在
//最初创建节点。
func initBlockNode(node *blockNode, blockHeader *wire.BlockHeader, parent *blockNode) {
	*node = blockNode{
		hash:       blockHeader.BlockHash(),
		workSum:    CalcWork(blockHeader.Bits),
		version:    blockHeader.Version,
		bits:       blockHeader.Bits,
		nonce:      blockHeader.Nonce,
		timestamp:  blockHeader.Timestamp.Unix(),
		merkleRoot: blockHeader.MerkleRoot,
	}
	if parent != nil {
		node.parent = parent
		node.height = parent.height + 1
		node.workSum = node.workSum.Add(parent.workSum, node.workSum)
	}
}

//new block node返回给定块头和父级的新块节点
//节点，从
//起源。此函数对于并发访问不安全。
func newBlockNode(blockHeader *wire.BlockHeader, parent *blockNode) *blockNode {
	var node blockNode
	initBlockNode(&node, blockHeader, parent)
	return &node
}

//header从节点构造一个块头并返回它。
//
//此函数对于并发访问是安全的。
func (node *blockNode) Header() wire.BlockHeader {
//不需要锁，因为所有访问的字段都是不可变的。
	prevHash := &zeroHash
	if node.parent != nil {
		prevHash = &node.parent.hash
	}
	return wire.BlockHeader{
		Version:    node.version,
		PrevBlock:  *prevHash,
		MerkleRoot: node.merkleRoot,
		Timestamp:  time.Unix(node.timestamp, 0),
		Bits:       node.bits,
		Nonce:      node.nonce,
	}
}

//ancestor通过以下方式返回位于提供高度的ancestor块节点
//从这个节点向后的链。当
//请求的高度在传递节点的高度之后或小于
//超过零。
//
//此函数对于并发访问是安全的。
func (node *blockNode) Ancestor(height int32) *blockNode {
	if height < 0 || height > node.height {
		return nil
	}

	n := node
	for ; n != nil && n.height != height; n = n.parent {
//有意留空
	}

	return n
}

//relative ancestor返回祖先块节点的相对“距离”块
//在此节点之前。这相当于使用节点的
//高度减去提供的距离。
//
//此函数对于并发访问是安全的。
func (node *blockNode) RelativeAncestor(distance int32) *blockNode {
	return node.Ancestor(node.height - distance)
}

//calcpastmediantime计算前几个块的中间时间
//块节点之前（包括该节点）。
//
//此函数对于并发访问是安全的。
func (node *blockNode) CalcPastMedianTime() time.Time {
//创建用于计算的前几个块时间戳的切片
//由常量mediantimeblocks定义的每个数字的中位数。
	timestamps := make([]int64, medianTimeBlocks)
	numNodes := 0
	iterNode := node
	for i := 0; i < medianTimeBlocks && iterNode != nil; i++ {
		timestamps[i] = iterNode.timestamp
		numNodes++

		iterNode = iterNode.parent
	}

//将切片修剪为可用时间戳的实际数量，该时间戳
//在区块链的开始处将少于所需数量
//把它们分类。
	timestamps = timestamps[:numNodes]
	sort.Sort(timeSorter(timestamps))

//注：共识规则错误地计算了偶数的中位数
//块数。一个真正的中位数平均中间两个元素
//对于包含偶数个元素的集合。因为常数
//因为要使用的前几个块是奇数，所以这只是一个
//在链条开始附近的几个区块发出。我怀疑
//这是一个优化，尽管结果与
//前几个街区之后的几个街区
//将始终是集合中每个常量的奇数个块。
//
//但为了确保使用相同的规则，该代码也适用于
//注意，如果将mediantimeblocks常量更改为
//偶数，此代码将是错误的。
	medianTimestamp := timestamps[numNodes/2]
	return time.Unix(medianTimestamp, 0)
}

//blockindex提供跟踪
//砌块链。尽管名称块链建议
//块，它实际上是一个树形结构，任何节点都可以
//多个子项。但是，只能有一个活动分支
//事实上，从顶端到创世块形成一条链子。
type blockIndex struct {
//以下字段是在创建实例时设置的，不能
//之后再更改，因此无需使用
//单独互斥。
	db          database.DB
	chainParams *chaincfg.Params

	sync.RWMutex
	index map[chainhash.Hash]*blockNode
	dirty map[*blockNode]struct{}
}

//NewBlockIndex返回块索引的新空实例。索引将
//当从数据库加载块节点时动态填充
//手动添加。
func newBlockIndex(db database.DB, chainParams *chaincfg.Params) *blockIndex {
	return &blockIndex{
		db:          db,
		chainParams: chainParams,
		index:       make(map[chainhash.Hash]*blockNode),
		dirty:       make(map[*blockNode]struct{}),
	}
}

//HaveBlock返回块索引是否包含提供的哈希。
//
//此函数对于并发访问是安全的。
func (bi *blockIndex) HaveBlock(hash *chainhash.Hash) bool {
	bi.RLock()
	_, hasBlock := bi.index[*hash]
	bi.RUnlock()
	return hasBlock
}

//lookupnode返回由提供的哈希标识的块节点。它将
//如果哈希没有条目，则返回nil。
//
//此函数对于并发访问是安全的。
func (bi *blockIndex) LookupNode(hash *chainhash.Hash) *blockNode {
	bi.RLock()
	node := bi.index[*hash]
	bi.RUnlock()
	return node
}

//addnode将提供的节点添加到块索引并将其标记为脏节点。
//重复的条目不会被选中，所以由调用者来避免添加它们。
//
//此函数对于并发访问是安全的。
func (bi *blockIndex) AddNode(node *blockNode) {
	bi.Lock()
	bi.addNode(node)
	bi.dirty[node] = struct{}{}
	bi.Unlock()
}

//addnode将提供的节点添加到块索引，但不将其标记为
//脏了。这可以在初始化块索引时使用。
//
//此函数对于并发访问不安全。
func (bi *blockIndex) addNode(node *blockNode) {
	bi.index[node.hash] = node
}

//node status提供对节点状态字段的并发安全访问。
//
//此函数对于并发访问是安全的。
func (bi *blockIndex) NodeStatus(node *blockNode) blockStatus {
	bi.RLock()
	status := node.status
	bi.RUnlock()
	return status
}

//setstatusflags将块节点上提供的状态标志翻转为on，
//不管他们以前是开着还是关着。这不会使任何
//当前打开的标志。
//
//此函数对于并发访问是安全的。
func (bi *blockIndex) SetStatusFlags(node *blockNode, flags blockStatus) {
	bi.Lock()
	node.status |= flags
	bi.dirty[node] = struct{}{}
	bi.Unlock()
}

//unsetStatusFlags将块节点上提供的状态标志翻转为关闭，
//不管他们以前是开着还是关着。
//
//此函数对于并发访问是安全的。
func (bi *blockIndex) UnsetStatusFlags(node *blockNode, flags blockStatus) {
	bi.Lock()
	node.status &^= flags
	bi.dirty[node] = struct{}{}
	bi.Unlock()
}

//FlushToDB将所有脏块节点写入数据库。如果所有写入
//成功，这将清除脏集。
func (bi *blockIndex) flushToDB() error {
	bi.Lock()
	if len(bi.dirty) == 0 {
		bi.Unlock()
		return nil
	}

	err := bi.db.Update(func(dbTx database.Tx) error {
		for node := range bi.dirty {
			err := dbStoreBlockNode(dbTx, node)
			if err != nil {
				return err
			}
		}
		return nil
	})

//如果写入成功，请清除脏集。
	if err == nil {
		bi.dirty = make(map[*blockNode]struct{})
	}

	bi.Unlock()
	return err
}
