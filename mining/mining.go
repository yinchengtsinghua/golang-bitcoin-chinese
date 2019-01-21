
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
	"bytes"
	"container/heap"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

const (
//MinHighPriority是允许
//交易应被视为高优先级。
	MinHighPriority = btcutil.SatoshiPerBitcoin * 144.0 / 250

//BlockHeaderOverhead是序列化所需的最大字节数。
//块头和最大可能事务计数。
	blockHeaderOverhead = wire.MaxBlockHeaderPayload + wire.MaxVarIntPayload

//CoinBaseFlags添加到生成的块的CoinBase脚本中
//用于监控bip16支持以及
//通过BTCD生成。
	CoinbaseFlags = "/P2SH/btcd/"
)

//txtesc是关于事务源中事务的描述符，它与
//附加元数据。
type TxDesc struct {
//Tx是与条目关联的事务。
	Tx *btcutil.Tx

//“添加”是将条目添加到源池中的时间。
	Added time.Time

//Height是将项添加到源时的块高度。
//池。
	Height int32

//费用是与条目关联的交易支付的总费用。
	Fee int64

//feeperkb是以satoshi/1000字节为单位支付的费用。
	FeePerKB int64
}

//TxSource表示要考虑包含在
//新街区。
//
//接口合同要求所有这些方法对于
//对源的并发访问。
type TxSource interface {
//LastUpdated返回上次向或添加事务的时间
//已从源池中删除。
	LastUpdated() time.Time

//miningdescs返回所有
//源池中的事务。
	MiningDescs() []*TxDesc

//HaveTransaction返回是否传递了事务哈希
//存在于源池中。
	HaveTransaction(hash *chainhash.Hash) bool
}

//txPrioItem houses a transaction along with extra information that allows the
//要优先处理的事务并跟踪对其他事务的依赖性
//还没有开采成块。
type txPrioItem struct {
	tx       *btcutil.Tx
	fee      int64
	priority float64
	feePerKB int64

//Dependson持有此事务哈希所依赖的事务哈希的映射
//在。仅当事务引用其他
//源池中的事务，因此必须在
//一个街区
	dependsOn map[chainhash.Hash]struct{}
}

//txPriorityQueuelessFunc描述了一个可以用作比较的函数
//事务优先级队列（TxPriorityQueue）的函数。
type txPriorityQueueLessFunc func(*txPriorityQueue, int, int) bool

//TxPriorityQueue实现TxPrioritem元素的优先级队列
//支持由txPriorityQueuelessFunc定义的任意比较函数。
type txPriorityQueue struct {
	lessFunc txPriorityQueueLessFunc
	items    []*txPrioItem
}

//LeN返回优先级队列中的项数。它是
//堆。接口实现。
func (pq *txPriorityQueue) Len() int {
	return len(pq.items)
}

//较少的返回优先级索引队列中的项目是否应该排序
//在带有索引j的项之前，通过延迟分配的less函数。它
//是堆。接口实现的一部分。
func (pq *txPriorityQueue) Less(i, j int) bool {
	return pq.lessFunc(pq, i, j)
}

//交换在优先级队列中传递的索引处交换项目。它是
//堆的一部分。接口实现。
func (pq *txPriorityQueue) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
}

//push将传递的项推送到优先级队列中。它是
//堆。接口实现。
func (pq *txPriorityQueue) Push(x interface{}) {
	pq.items = append(pq.items, x.(*txPrioItem))
}

//pop从优先级中删除最高优先级的项（按“更少”）。
//排队并返回。它是heap.interface实现的一部分。
func (pq *txPriorityQueue) Pop() interface{} {
	n := len(pq.items)
	item := pq.items[n-1]
	pq.items[n-1] = nil
	pq.items = pq.items[0 : n-1]
	return item
}

//setlessfunc将优先级队列的比较函数设置为
//功能。它还使用新的
//函数，以便它可以立即与heap.push/pop一起使用。
func (pq *txPriorityQueue) SetLessFunc(lessFunc txPriorityQueueLessFunc) {
	pq.lessFunc = lessFunc
	heap.Init(pq)
}

//TxPqByPriority按事务优先级对TxPriorityQueue排序，然后按费用排序
//每千字节。
func txPQByPriority(pq *txPriorityQueue, i, j int) bool {
//在这里使用>以便pop提供最高优先级的项，而不是
//降到最低。先按优先级排序，然后按费用排序。
	if pq.items[i].priority == pq.items[j].priority {
		return pq.items[i].feePerKB > pq.items[j].feePerKB
	}
	return pq.items[i].priority > pq.items[j].priority

}

//TxPqByFee按每千字节的费用对TxPriorityQueue进行排序，然后进行事务处理。
//优先。
func txPQByFee(pq *txPriorityQueue, i, j int) bool {
//使用>这里，这样弹出窗口会给出与之相反的最高收费项目
//降到最低。先按费用排序，然后按优先级排序。
	if pq.items[i].feePerKB == pq.items[j].feePerKB {
		return pq.items[i].priority > pq.items[j].priority
	}
	return pq.items[i].feePerKB > pq.items[j].feePerKB
}

//newtxPriorityQueue返回保留
//为元素传递的空间量。新的优先级队列使用
//txpqbypriority或txpqbyfee比较功能取决于
//SortByFee参数，已初始化以用于heap.push/pop。
//优先级队列可能会比保留空间大，但会增加额外的副本
//可以通过保留一个健全的值来避免底层数组的错误。
func newTxPriorityQueue(reserve int, sortByFee bool) *txPriorityQueue {
	pq := &txPriorityQueue{
		items: make([]*txPrioItem, 0, reserve),
	}
	if sortByFee {
		pq.SetLessFunc(txPQByFee)
	} else {
		pq.SetLessFunc(txPQByPriority)
	}
	return pq
}

//块模板包含一个尚未解决的块以及其他
//有关费用和每个签名操作的数量的详细信息
//块中的事务。
type BlockTemplate struct {
//区块是一个准备由矿工解决的区块。因此，它是
//除满足工作证明外，完全有效
//要求。
	Block *wire.MsgBlock

//费用包含生成的每个交易中的费用金额
//模板以基本单位支付。因为第一个事务是
//coinbase, the first entry (offset 0) will contain the negative of the
//所有其他交易费用的总和。
	Fees []int64

//SigOpCosts contains the number of signature operations each
//生成的模板中的事务将执行。
	SigOpCosts []int64

//高度是块模板连接到主模板的高度
//链。
	Height int32

//validpayaddress表示模板coinbase是否支付
//地址或任何人都可以赎回。请参阅上的文档
//newblocktemplate，用于生成有用的详细信息
//没有CoinBase付款地址的模板。
	ValidPayAddress bool

//见证承诺是对见证数据（如有）的承诺。
//在街区内。This field will only be populted once segregated
//见证已激活，并且块包含一个事务
//有证人资料。
	WitnessCommitment []byte
}

//
//VIEWA将包含所有原始条目和所有条目
//在VIEB中。它将替换VIEWB中也存在于VIEWA中的任何条目。
//如果VIEWA中的条目已用完。
func mergeUtxoView(viewA *blockchain.UtxoViewpoint, viewB *blockchain.UtxoViewpoint) {
	viewAEntries := viewA.Entries()
	for outpoint, entryB := range viewB.Entries() {
		if entryA, exists := viewAEntries[outpoint]; !exists ||
			entryA == nil || entryA.IsSpent() {

			viewAEntries[outpoint] = entryB
		}
	}
}

//StandardCoinBaseScript返回适合用作
//新块的CoinBase事务的签名脚本。特别地，
//它以版本2所需的块高度开始，并添加
//额外的nonce和额外的coinbase标志。
func standardCoinbaseScript(nextBlockHeight int32, extraNonce uint64) ([]byte, error) {
	return txscript.NewScriptBuilder().AddInt64(int64(nextBlockHeight)).
		AddInt64(int64(extraNonce)).AddData([]byte(CoinbaseFlags)).
		Script()
}

//CreateCoinBaseTx返回支付适当补贴的CoinBase交易
//基于传递到所提供地址的块高度。当地址
//如果为零，则任何人都可以赎回CoinBase交易。
//
//有关nil的原因的详细信息，请参见newblocktemplate的注释。
//地址处理很有用。
func createCoinbaseTx(params *chaincfg.Params, coinbaseScript []byte, nextBlockHeight int32, addr btcutil.Address) (*btcutil.Tx, error) {
//创建脚本以支付到提供的支付地址（如果有）
//明确规定。否则，创建一个脚本，允许coinbase
//任何人都可以赎回。
	var pkScript []byte
	if addr != nil {
		var err error
		pkScript, err = txscript.PayToAddrScript(addr)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		scriptBuilder := txscript.NewScriptBuilder()
		pkScript, err = scriptBuilder.AddOp(txscript.OP_TRUE).Script()
		if err != nil {
			return nil, err
		}
	}

	tx := wire.NewMsgTx(wire.TxVersion)
	tx.AddTxIn(&wire.TxIn{
//CoinBase事务没有输入，因此以前的输出点是
//零哈希和最大索引。
		PreviousOutPoint: *wire.NewOutPoint(&chainhash.Hash{},
			wire.MaxPrevOutIndex),
		SignatureScript: coinbaseScript,
		Sequence:        wire.MaxTxInSequenceNum,
	})
	tx.AddTxOut(&wire.TxOut{
		Value:    blockchain.CalcBlockSubsidy(nextBlockHeight, params),
		PkScript: pkScript,
	})
	return btcutil.NewTx(tx), nil
}

//SpendTransaction通过将输入标记为
//已用事务。它还添加了传递事务中的所有输出
//它们不能作为可用的未暂停事务输出被证明是不可暂停的。
func spendTransaction(utxoView *blockchain.UtxoViewpoint, tx *btcutil.Tx, height int32) error {
	for _, txIn := range tx.MsgTx().TxIn {
		entry := utxoView.LookupEntry(txIn.PreviousOutPoint)
		if entry != nil {
			entry.Spend()
		}
	}

	utxoView.AddTxOuts(tx, height)
	return nil
}

//logskippeddedps记录由于
//在跟踪级别生成块模板时跳过事务。
func logSkippedDeps(tx *btcutil.Tx, deps map[chainhash.Hash]*txPrioItem) {
	if deps == nil {
		return
	}

	for _, item := range deps {
		log.Tracef("Skipping tx %s since it depends on %s\n",
			item.tx.Hash(), tx.Hash())
	}
}

//minimammediantime返回块构建允许的最小时间戳
//在提供的最佳链的末端。尤其是一秒钟之后
//每个链共识最后几个块的中间时间戳
//规则。
func MinimumMedianTime(chainState *blockchain.BestState) time.Time {
	return chainState.MedianTime.Add(time.Second)
}

//medianadjustedtime返回调整后的当前时间，以确保至少
//在每个
//连锁共识规则。
func medianAdjustedTime(chainState *blockchain.BestState, timeSource blockchain.MedianTimeSource) time.Time {
//块的时间戳不能早于中间时间戳
//最后几个街区。因此，在
//当前时间和上一个中间时间后一秒。电流
//在比较之前，时间戳被截断为第二个边界，因为
//块时间戳不支持大于1的精度
//第二。
	newTimestamp := timeSource.AdjustedTime()
	minTimestamp := MinimumMedianTime(chainState)
	if newTimestamp.Before(minTimestamp) {
		newTimestamp = minTimestamp
	}

	return newTimestamp
}

//blktmplGenerator提供了一种类型，可用于生成块模板
//基于给定的挖掘策略和要从中选择的事务源。
//它还包含确保模板所需的其他状态
//是建立在当前最好的链条之上，并遵守共识规则。
type BlkTmplGenerator struct {
	policy      *Policy
	chainParams *chaincfg.Params
	txSource    TxSource
	chain       *blockchain.BlockChain
	timeSource  blockchain.MedianTimeSource
	sigCache    *txscript.SigCache
	hashCache   *txscript.HashCache
}

//newblktmplgenerator返回给定的块模板生成器
//使用来自所提供事务源的事务的策略。
//
//为了确保
//模板构建在当前最佳链的顶部，并遵循
//共识规则。
func NewBlkTmplGenerator(policy *Policy, params *chaincfg.Params,
	txSource TxSource, chain *blockchain.BlockChain,
	timeSource blockchain.MedianTimeSource,
	sigCache *txscript.SigCache,
	hashCache *txscript.HashCache) *BlkTmplGenerator {

	return &BlkTmplGenerator{
		policy:      policy,
		chainParams: params,
		txSource:    txSource,
		chain:       chain,
		timeSource:  timeSource,
		sigCache:    sigCache,
		hashCache:   hashCache,
	}
}

//new block template返回一个准备好解决的新块模板
//使用传递的事务源池和CoinBase中的事务
//如果不为零，则支付到传递的地址，或者
//如果传递的地址为零，任何人都可以赎回。零地址
//功能非常有用，因为存在诸如getBlockTemplate之类的情况
//rpc，其中外部挖掘软件负责创建自己的
//将替换为块模板生成的CoinBase。因此
//可以避免需要配置地址。
//
//所选和包含的事务根据以下几项进行优先级排序
//因素。首先，每个事务都有一个基于其
//值、输入时间和大小。包含较大交易的交易
//数量、旧输入和小尺寸具有最高优先级。第二，A
//每千字节的费用是为每个交易计算的。与
//每千字节的费用越高越好。最后，块生成相关
//策略设置都会考虑在内。
//
//仅花费已存在于
//块链立即添加到优先级队列
//根据优先级（然后是每千字节的费用）或
//千字节（然后是优先级）取决于blockPrioritySize
//策略设置为高优先级事务分配空间。交易
//将源池中其他事务的支出输出添加到
//依赖关系映射，以便在
//它们所依赖的事务已包括在内。
//
//一旦高优先级区域（如果配置）被填满
//交易，或者优先级低于被视为高优先级的事务，
//优先级队列将更新为按每千字节费用划分优先级（然后
//优先权）。
//
//当每千字节的费用低于txminfreefee策略设置时，
//除非BlockMinSize策略设置为
//非零，在这种情况下，块将填充低费用/免费
//直到块大小达到最小大小为止的事务。
//
//导致块超过blockMaxSize的任何事务
//策略设置，超过了每个块允许的最大签名操作数，或者
//否则将跳过块无效。
//
//鉴于上述情况，此函数生成的块的形式如下：
//
//————————————————————————————————
//CoinBase交易
//-----------------------------__
//----policy.blockPrioritySize
//高优先级事务
//_
//----------------------------------_
//|                                   |   |
//|                                   |   |
//---策略.blockMaxSize
//按费用排序的交易
//直到<=policy.txminfreefee
//|                                   |   |
//|                                   |   |
//|                                   |   |
//-----------------------------------
//低收费/非高优先级（免费）
//事务（而块大小
//<=policy.blockminize）
//—————————————————————————
func (g *BlkTmplGenerator) NewBlockTemplate(payToAddress btcutil.Address) (*BlockTemplate, error) {
//扩展最近已知的最佳块。
	best := g.chain.BestSnapshot()
	nextBlockHeight := best.Height + 1

//创建向提供的
//地址。注意：CoinBase值将更新为包含
//所选交易的费用
//已选定。在这里创建它是为了尽早检测任何错误
//在下面做很多工作之前。额外的一段时间有助于
//确保交易不是重复的交易（支付
//相同的值到相同的公钥地址，否则将是
//块版本1的相同事务）。
	extraNonce := uint64(0)
	coinbaseScript, err := standardCoinbaseScript(nextBlockHeight, extraNonce)
	if err != nil {
		return nil, err
	}
	coinbaseTx, err := createCoinbaseTx(g.chainParams, coinbaseScript,
		nextBlockHeight, payToAddress)
	if err != nil {
		return nil, err
	}
	coinbaseSigOpCost := int64(blockchain.CountSigOps(coinbaseTx)) * blockchain.WitnessScaleFactor

//获取当前源事务并创建优先级队列
//保留准备包含到块中的事务
//以及一些与优先级相关的和费用元数据。保留相同的
//可用于优先级队列的项目数。也，
//根据是否选择优先级队列的初始排序顺序
//或者不存在为高优先级事务分配的区域。
	sourceTxns := g.txSource.MiningDescs()
	sortedByFee := g.policy.BlockPrioritySize == 0
	priorityQueue := newTxPriorityQueue(len(sourceTxns), sortedByFee)

//创建一个切片以保存要包含在
//已生成具有保留空间的块。同时创建一个utxo视图
//包含所有输入事务，以便可以进行多个查找
//避免。
	blockTxns := make([]*btcutil.Tx, 0, len(sourceTxns))
	blockTxns = append(blockTxns, coinbaseTx)
	blockUtxos := blockchain.NewUtxoViewpoint()

//依赖项用于跟踪依赖于另一个
//源池中的事务。这与
//与每个相关事务一起保存的Dependson映射有助于快速
//确定哪些从属交易现在可以包含
//一旦每个事务都包含在块中。
	dependers := make(map[chainhash.Hash]map[chainhash.Hash]*txPrioItem)

//创建切片以保存签名操作的费用和数量
//对于每个选定的事务，并为
//钴基。这允许下面的代码简单地附加有关
//选定要包含在最终块中的事务。
//但是，由于还不知道总费用，请使用虚拟值
//稍后将更新的CoinBase费用。
	txFees := make([]int64, 0, len(sourceTxns))
	txSigOpCosts := make([]int64, 0, len(sourceTxns))
txFees = append(txFees, -1) //已知时更新
	txSigOpCosts = append(txSigOpCosts, coinbaseSigOpCost)

	log.Debugf("Considering %d transactions for inclusion to new block",
		len(sourceTxns))

mempoolLoop:
	for _, txDesc := range sourceTxns {
//一个块不能有多个coinbase或包含
//未定案交易。
		tx := txDesc.Tx
		if blockchain.IsCoinBase(tx) {
			log.Tracef("Skipping coinbase tx %s", tx.Hash())
			continue
		}
		if !blockchain.IsFinalizedTransaction(tx, nextBlockHeight,
			g.timeSource.AdjustedTime()) {

			log.Tracef("Skipping non-finalized tx %s", tx.Hash())
			continue
		}

//获取此事务引用的所有utxos。
//注意：这不会从
//自依赖于其他
//mempool中的事务必须在这些事务之后
//最终生成的块中的依赖项。
		utxos, err := g.chain.FetchUtxoView(tx)
		if err != nil {
			log.Warnf("Unable to fetch utxo view for tx %s: %v",
				tx.Hash(), err)
			continue
		}

//为引用的任何事务设置依赖项
//内存池中的其他事务，以便
//以下命令。
		prioItem := &txPrioItem{tx: tx}
		for _, txIn := range tx.MsgTx().TxIn {
			originHash := &txIn.PreviousOutPoint.Hash
			entry := utxos.LookupEntry(txIn.PreviousOutPoint)
			if entry == nil || entry.IsSpent() {
				if !g.txSource.HaveTransaction(originHash) {
					log.Tracef("Skipping tx %s because it "+
						"references unspent output %s "+
						"which is not available",
						tx.Hash(), txIn.PreviousOutPoint)
					continue mempoolLoop
				}

//该事务正在引用另一个事务
//源池中的事务，因此设置
//排序依赖项。
				deps, exists := dependers[*originHash]
				if !exists {
					deps = make(map[chainhash.Hash]*txPrioItem)
					dependers[*originHash] = deps
				}
				deps[*prioItem.tx.Hash()] = prioItem
				if prioItem.dependsOn == nil {
					prioItem.dependsOn = make(
						map[chainhash.Hash]struct{})
				}
				prioItem.dependsOn[*originHash] = struct{}{}

//跳过下面的检查。我们已经知道
//引用的事务可用。
				continue
			}
		}

//使用输入计算最终事务优先级
//价值年限总和以及调整后的交易规模。这个
//公式为：SUM（输入值*输入值）/ADjustedTxsize
		prioItem.priority = CalcPriority(tx.MsgTx(), utxos,
			nextBlockHeight)

//计算Satoshi /KB的费用。
		prioItem.feePerKB = txDesc.FeePerKB
		prioItem.fee = txDesc.Fee

//将事务添加到优先级队列以将其标记为就绪
//用于包含在块中，除非它具有依赖项。
		if prioItem.dependsOn == nil {
			heap.Push(priorityQueue, prioItem)
		}

//将输入事务中引用的输出合并到
//此事务进入块utxo视图。这允许
//下面的代码避免再次查找。
		mergeUtxoView(blockUtxos, utxos)
	}

	log.Tracef("Priority queue len %d, dependers len %d",
		priorityQueue.Len(), len(dependers))

//起始块大小是块头的大小加上最大值
//可能的事务计数大小，加上coinbase的大小
//交易。
	blockWeight := uint32((blockHeaderOverhead * blockchain.WitnessScaleFactor) +
		blockchain.GetTransactionWeight(coinbaseTx))
	blockSigOpCost := coinbaseSigOpCost
	totalFees := int64(0)

//查询版本位状态，查看segwit是否已激活，如果
//所以这意味着我们将包括与证人的任何交易
//在mempool中添加数据，并将证人承诺作为
//op_返回coinbase事务中的输出。
	segwitState, err := g.chain.ThresholdState(chaincfg.DeploymentSegwit)
	if err != nil {
		return nil, err
	}
	segwitActive := segwitState == blockchain.ThresholdActive

	witnessIncluded := false

//选择将其放入块中的事务。
	for priorityQueue.Len() > 0 {
//获取最高优先级（或每千字节的最高费用）
//取决于排序顺序）事务。
		prioItem := heap.Pop(priorityQueue).(*txPrioItem)
		tx := prioItem.tx

		switch {
//如果隔离证人还没有被激活，那么我们
//不应包括块中的任何见证事务。
		case !segwitActive && tx.HasWitness():
			continue

//否则，跟踪是否包括交易
//是否有证人资料。如果是，那么我们需要包括
//作为CoinBase中最后一个输出的见证承诺
//交易。
		case segwitActive && !witnessIncluded && tx.HasWitness():
//如果我们要包括交易承担
//证人数据，那么我们还需要包括
//见证CoinBase交易中的承诺。
//因此，我们考虑了额外的重量
//在带有CoinBase TX模型的块内，
//见证承诺。
			coinbaseCopy := btcutil.NewTx(coinbaseTx.MsgTx().Copy())
			coinbaseCopy.MsgTx().TxIn[0].Witness = [][]byte{
				bytes.Repeat([]byte("a"),
					blockchain.CoinbaseWitnessDataLen),
			}
			coinbaseCopy.MsgTx().AddTxOut(&wire.TxOut{
				PkScript: bytes.Repeat([]byte("a"),
					blockchain.CoinbaseWitnessPkScriptLength),
			})

//为了准确计算重量
//由于这个CoinBase交易，我们将添加
//交易前后的差额
//增加了对块重的承诺。
			weightDiff := blockchain.GetTransactionWeight(coinbaseCopy) -
				blockchain.GetTransactionWeight(coinbaseTx)

			blockWeight += uint32(weightDiff)

			witnessIncluded = true
		}

//获取任何依赖于此事务的事务。
		deps := dependers[*tx.Hash()]

//强制最大块大小。同时检查是否溢出。
		txWeight := uint32(blockchain.GetTransactionWeight(tx))
		blockPlusTxWeight := blockWeight + txWeight
		if blockPlusTxWeight < blockWeight ||
			blockPlusTxWeight >= g.policy.BlockMaxWeight {

			log.Tracef("Skipping tx %s because it would exceed "+
				"the max block weight", tx.Hash())
			logSkippedDeps(tx, deps)
			continue
		}

//强制每个块的最大签名操作成本。阿尔索
//检查是否溢出。
		sigOpCost, err := blockchain.GetSigOpCost(tx, false,
			blockUtxos, true, segwitActive)
		if err != nil {
			log.Tracef("Skipping tx %s due to error in "+
				"GetSigOpCost: %v", tx.Hash(), err)
			logSkippedDeps(tx, deps)
			continue
		}
		if blockSigOpCost+int64(sigOpCost) < blockSigOpCost ||
			blockSigOpCost+int64(sigOpCost) > blockchain.MaxBlockSigOpsCost {
			log.Tracef("Skipping tx %s because it would "+
				"exceed the maximum sigops per block", tx.Hash())
			logSkippedDeps(tx, deps)
			continue
		}

//一旦块大于
//最小块大小。
		if sortedByFee &&
			prioItem.feePerKB < int64(g.policy.TxMinFreeFee) &&
			blockPlusTxWeight >= g.policy.BlockMinWeight {

			log.Tracef("Skipping tx %s with feePerKB %d "+
				"< TxMinFreeFee %d and block weight %d >= "+
				"minBlockWeight %d", tx.Hash(), prioItem.feePerKB,
				g.policy.TxMinFreeFee, blockPlusTxWeight,
				g.policy.BlockMinWeight)
			logSkippedDeps(tx, deps)
			continue
		}

//一旦块大于
//优先级大小或没有更高的优先级
//交易。
		if !sortedByFee && (blockPlusTxWeight >= g.policy.BlockPrioritySize ||
			prioItem.priority <= MinHighPriority) {

			log.Tracef("Switching to sort by fees per "+
				"kilobyte blockSize %d >= BlockPrioritySize "+
				"%d || priority %.2f <= minHighPriority %.2f",
				blockPlusTxWeight, g.policy.BlockPrioritySize,
				prioItem.priority, MinHighPriority)

			sortedByFee = true
			priorityQueue.SetLessFunc(txPQByFee)

//将事务放回优先级队列，然后
//跳过它，如果不这样做的话，它将被费用重新优先考虑。
//适合高优先级部分或优先级
//太低了。否则，此事务将是
//在高优先级部分的最后一个，所以就下降吧
//但是下面的代码，所以现在添加了它。
			if blockPlusTxWeight > g.policy.BlockPrioritySize ||
				prioItem.priority < MinHighPriority {

				heap.Push(priorityQueue, prioItem)
				continue
			}
		}

//确保事务输入通过所有必要的
//允许将其添加到块之前的先决条件。
		_, err = blockchain.CheckTransactionInputs(tx, nextBlockHeight,
			blockUtxos, g.chainParams)
		if err != nil {
			log.Tracef("Skipping tx %s due to error in "+
				"CheckTransactionInputs: %v", tx.Hash(), err)
			logSkippedDeps(tx, deps)
			continue
		}
		err = blockchain.ValidateTransactionScripts(tx, blockUtxos,
			txscript.StandardVerifyFlags, g.sigCache,
			g.hashCache)
		if err != nil {
			log.Tracef("Skipping tx %s due to error in "+
				"ValidateTransactionScripts: %v", tx.Hash(), err)
			logSkippedDeps(tx, deps)
			continue
		}

//使用块utxo视图中的事务输入并添加
//用于确保引用的任何交易的条目
//这一个将它作为输入提供，并可以确保它们
//不是双倍消费。
		spendTransaction(blockUtxos, tx, nextBlockHeight)

//将事务添加到块、递增计数器和
//将费用和签名操作计数保存到块中
//模板。
		blockTxns = append(blockTxns, tx)
		blockWeight += txWeight
		blockSigOpCost += int64(sigOpCost)
		totalFees += prioItem.fee
		txFees = append(txFees, prioItem.fee)
		txSigOpCosts = append(txSigOpCosts, int64(sigOpCost))

		log.Tracef("Adding tx %s (priority %.2f, feePerKB %.2f)",
			prioItem.tx.Hash(), prioItem.priority, prioItem.feePerKB)

//添加依赖于此事务的事务（也不添加
//有任何其他未经授权的依赖项）
//排队。
		for _, item := range deps {
//将事务添加到优先级队列（如果存在）
//在此之后不再依赖。
			delete(item.dependsOn, *tx.Hash())
			if len(item.dependsOn) == 0 {
				heap.Push(priorityQueue, item)
			}
		}
	}

//现在已选择实际交易记录，请更新
//实际交易计数和coinbase值的块权重
//相应的总费用。
	blockWeight -= wire.MaxVarIntPayload -
		(uint32(wire.VarIntSerializeSize(uint64(len(blockTxns)))) *
			blockchain.WitnessScaleFactor)
	coinbaseTx.MsgTx().TxOut[0].Value += totalFees
	txFees[0] = -totalFees

//如果Segwit是活跃的，并且我们包括有见证数据的交易，
//然后我们需要在
//op_返回coinbase事务中的输出。
	var witnessCommitment []byte
	if witnessIncluded {
//CoinBase事务的见证必须正好为32个字节
//全零的
		var witnessNonce [blockchain.CoinbaseWitnessDataLen]byte
		coinbaseTx.MsgTx().TxIn[0].Witness = wire.TxWitness{witnessNonce[:]}

//接下来，获取由
//块中所有事务的wtxid。硬币库
//事务将具有所有零的特殊wtxid。
		witnessMerkleTree := blockchain.BuildMerkleTreeStore(blockTxns,
			true)
		witnessMerkleRoot := witnessMerkleTree[len(witnessMerkleTree)-1]

//证人承诺的预兆是：
//目击证人
		var witnessPreimage [64]byte
		copy(witnessPreimage[:32], witnessMerkleRoot[:])
		copy(witnessPreimage[32:], witnessNonce[:])

//证人承诺本身就是
//见证上图。带着承诺
//生成，输出的见证脚本为：op_-return
//操作数据0XA21A9ED见证承诺。领导
//前缀被称为“见证魔法字节”。
		witnessCommitment = chainhash.DoubleHashB(witnessPreimage[:])
		witnessScript := append(blockchain.WitnessMagicBytes, witnessCommitment...)

//最后，创建带证人承诺的Op_返回
//输出作为coinbase中的附加输出。
		commitmentOutput := &wire.TxOut{
			Value:    0,
			PkScript: witnessScript,
		}
		coinbaseTx.MsgTx().TxOut = append(coinbaseTx.MsgTx().TxOut,
			commitmentOutput)
	}

//计算块所需的难度。时间戳
//可能会进行调整，以确保它在
//最后几个区块按照链共识规则。
	ts := medianAdjustedTime(best, g.timeSource)
	reqDifficulty, err := g.chain.CalcNextRequiredDifficulty(ts)
	if err != nil {
		return nil, err
	}

//根据
//规则更改部署。
	nextBlockVersion, err := g.chain.CalcNextBlockVersion()
	if err != nil {
		return nil, err
	}

//创建一个准备解决的新块。
	merkles := blockchain.BuildMerkleTreeStore(blockTxns, false)
	var msgBlock wire.MsgBlock
	msgBlock.Header = wire.BlockHeader{
		Version:    nextBlockVersion,
		PrevBlock:  best.Hash,
		MerkleRoot: *merkles[len(merkles)-1],
		Timestamp:  ts,
		Bits:       reqDifficulty,
	}
	for _, tx := range blockTxns {
		if err := msgBlock.AddTransaction(tx.MsgTx()); err != nil {
			return nil, err
		}
	}

//最后，根据链对创建的块执行完全检查
//一致同意的规则，以确保它正确地连接到当前的最佳
//链没有问题。
	block := btcutil.NewBlock(&msgBlock)
	block.SetHeight(nextBlockHeight)
	if err := g.chain.CheckConnectBlockTemplate(block); err != nil {
		return nil, err
	}

	log.Debugf("Created new block template (%d transactions, %d in "+
		"fees, %d signature operations cost, %d weight, target difficulty "+
		"%064x)", len(msgBlock.Transactions), totalFees, blockSigOpCost,
		blockWeight, blockchain.CompactToBig(msgBlock.Header.Bits))

	return &BlockTemplate{
		Block:             &msgBlock,
		Fees:              txFees,
		SigOpCosts:        txSigOpCosts,
		Height:            nextBlockHeight,
		ValidPayAddress:   payToAddress != nil,
		WitnessCommitment: witnessCommitment,
	}, nil
}

//updateBlockTime将传递的块头中的时间戳更新为
//当前时间，同时考虑最后一个时间的中间值
//several blocks to ensure the new time is after that time per the chain
//共识规则。最后，如果需要，它将更新目标难度
//基于测试网络的新时间，因为它们的目标难度可以
//根据时间变化。
func (g *BlkTmplGenerator) UpdateBlockTime(msgBlock *wire.MsgBlock) error {
//新的时间戳可能会被调整，以确保它在
//每个链共识最后几个块的中间时间
//规则。
	newTime := medianAdjustedTime(g.chain.BestSnapshot(), g.timeSource)
	msgBlock.Header.Timestamp = newTime

//如果在需要的网络上运行，则重新计算难度。
	if g.chainParams.ReduceMinDifficulty {
		difficulty, err := g.chain.CalcNextRequiredDifficulty(newTime)
		if err != nil {
			return err
		}
		msgBlock.Header.Bits = difficulty
	}

	return nil
}

//updateExtrance更新传递的coinBase脚本中的额外nonce
//通过使用传递的值和块重新生成coinbase脚本来阻止
//高度。它还重新计算并更新产生的新merkle根目录
//更改coinbase脚本。
func (g *BlkTmplGenerator) UpdateExtraNonce(msgBlock *wire.MsgBlock, blockHeight int32, extraNonce uint64) error {
	coinbaseScript, err := standardCoinbaseScript(blockHeight, extraNonce)
	if err != nil {
		return err
	}
	if len(coinbaseScript) > blockchain.MaxCoinbaseScriptLen {
		return fmt.Errorf("coinbase transaction script length "+
			"of %d is out of range (min: %d, max: %d)",
			len(coinbaseScript), blockchain.MinCoinbaseScriptLen,
			blockchain.MaxCoinbaseScriptLen)
	}
	msgBlock.Transactions[0].TxIn[0].SignatureScript = coinbaseScript

//TODO（Davec）：bcutil.block应使用保存在状态以避免
//重新计算所有其他事务哈希。
//块.事务[0].无效缓存（）

//使用更新的额外nonce重新计算merkle根。
	block := btcutil.NewBlock(msgBlock)
	merkles := blockchain.BuildMerkleTreeStore(block.Transactions(), false)
	msgBlock.Header.MerkleRoot = *merkles[len(merkles)-1]
	return nil
}

//BestSnapshot返回有关当前最佳链块和
//使用链实例的当前时间点的相关状态
//与块模板生成器关联。返回的状态必须为
//被视为不可变的，因为它由所有调用方共享。
//
//此函数对于并发访问是安全的。
func (g *BlkTmplGenerator) BestSnapshot() *blockchain.BestState {
	return g.chain.BestSnapshot()
}

//TxSource返回关联的事务源。
//
//此函数对于并发访问是安全的。
func (g *BlkTmplGenerator) TxSource() TxSource {
	return g.txSource
}
