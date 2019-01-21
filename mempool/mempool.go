
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

package mempool

import (
	"container/list"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/blockchain/indexers"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/mining"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

const (
//DeFultBasePrimoResithSead是高字节的默认大小。
//优先级/低费用交易。它用于帮助确定
//允许进入内存池，从而影响它们的中继和
//生成块模板时包含。
	DefaultBlockPrioritySize = 50000

//OrphantTTL是允许Orphan执行的最大时间量。
//在孤儿池到期前留在孤儿池中，并在
//下一次扫描。
	orphanTTL = time.Minute * 15

//OrphanExpiresCanInterval是介于
//扫描孤立池以逐出过期的事务。
	orphanExpireScanInterval = time.Minute * 5
)

//标记表示用于标记孤立事务的标识符。这个
//调用者可以选择它想要的任何方案，但是使用对等ID很常见
//这样，孤儿就可以通过同伴第一次中继他们来识别。
type Tag uint64

//配置是包含内存池配置的描述符。
type Config struct {
//策略定义了与
//政策。
	Policy Policy

//chainParams标识TxPool的链参数
//与关联。
	ChainParams *chaincfg.Params

//fetchutxoview定义用于提取未使用的函数
//事务输出信息。
	FetchUtxoView func(*btcutil.Tx) (*blockchain.UtxoViewpoint, error)

//BestHeight定义用于访问块高度的函数
//目前最好的链条。
	BestHeight func() int32

//MedianTimePast defines the function to use in order to access the
//从电流的角度计算过去的中值时间
//链尖在最好的链内。
	MedianTimePast func() time.Time

//CalcSequenceLock defines the function to use in order to generate
//使用传递的
//UTXO视图。
	CalcSequenceLock func(*btcutil.Tx, *blockchain.UtxoViewpoint) (*blockchain.SequenceLock, error)

//如果目标DeploymentID为，则IsDeploymentActive返回true。
//活动，否则为假。mempool使用此函数测量
//如果事务使用new to be软分叉规则，则应允许
//是否进入记忆池。
	IsDeploymentActive func(deploymentID uint32) (bool, error)

//sigcache定义要使用的签名缓存。
	SigCache *txscript.SigCache

//hash cache定义要使用的事务哈希中间状态缓存。
	HashCache *txscript.HashCache

//AddrIndex defines the optional address index instance to use for
//索引内存池中未确认的事务。
//如果未启用地址索引，则可以为零。
	AddrIndex *indexers.AddrIndex

//Fuff估计器提供了一个估计器。如果不是零，则内存池
//将其观察到的所有新交易记录到feeestimator中。
	FeeEstimator *FeeEstimator
}

//Policy houses the policy (configuration parameters) which is used to
//控制内存池。
type Policy struct {
//maxtxversion是mempool应该使用的事务版本
//接受。此版本以上的所有事务都被拒绝为
//非标准的。
	MaxTxVersion int32

//DisableRelayPriority defines whether to relay free or low-fee
//没有足够优先级来中继的事务。
	DisableRelayPriority bool

//AcceptNonStd defines whether to accept non-standard transactions. 如果
//是的，非标准事务将被接受到mempool中。
//否则，所有非标准交易将被拒绝。
	AcceptNonStd bool

//FreeTxRelayLimit defines the given amount in thousands of bytes
//每分钟不收费交易的费率限制为。
	FreeTxRelayLimit float64

//maxorphantxs是孤立事务的最大数目
//可以排队。
	MaxOrphanTxs int

//maxorphantxsize是孤立事务允许的最大大小。
//这有助于防止内存耗尽攻击发送大量
//大孤儿。
	MaxOrphanTxSize int

//MaxSigOpCostPerTx is the cumulative maximum cost of all the signature
//在一次交易中，我们将进行中继或挖掘。这是一个
//块的最大签名操作的分数。
	MaxSigOpCostPerTx int

//MinRelayTxFee defines the minimum transaction fee in BTC/kB to be
//视为非零费用。
	MinRelayTxFee btcutil.Amount
}

//txtesc是一个描述符，包含mempool中的事务以及
//附加元数据。
type TxDesc struct {
	mining.TxDesc

//StartingPriority是添加事务时事务的优先级
//去游泳池。
	StartingPriority float64
}

//orphantx是引用祖先事务的普通事务
//目前还没有。它还包含其他相关信息
//比如说一个有效期来帮助防止永远缓存孤儿。
type orphanTx struct {
	tx         *btcutil.Tx
	tag        Tag
	expiration time.Time
}

//TxPool用作需要挖掘成块的事务的源
//并转发给其他同行。从多个服务器并发访问是安全的
//同龄人。
type TxPool struct {
//以下变量只能原子地使用。
lastUpdated int64 //上次更新池的时间

	mtx           sync.RWMutex
	cfg           Config
	pool          map[chainhash.Hash]*TxDesc
	orphans       map[chainhash.Hash]*orphanTx
	orphansByPrev map[wire.OutPoint]map[chainhash.Hash]*btcutil.Tx
	outpoints     map[wire.OutPoint]*btcutil.Tx
pennyTotal    float64 //Penny支出的指数衰减。
lastPennyUnix int64   //上次“便士消费”的Unix时间

//NextExpirescan是孤儿池将在
//扫描以驱逐孤儿。这不是一个艰难的最后期限，因为
//只有将孤立项添加到池中时，扫描才会运行，相反
//无条件定时器。
	nextExpireScan time.Time
}

//确保TxPool类型实现Mining.TxSource接口。
var _ mining.TxSource = (*TxPool)(nil)

//removeorphan是实现公共
//RemoveOrphan。有关详细信息，请参阅removeorphan的注释。
//
//必须在保持mempool锁的情况下调用此函数（用于写入）。
func (mp *TxPool) removeOrphan(tx *btcutil.Tx, removeRedeemers bool) {
//如果传递的tx不是孤立的，则不执行任何操作。
	txHash := tx.Hash()
	otx, exists := mp.orphans[*txHash]
	if !exists {
		return
	}

//从上一个孤立索引中删除引用。
	for _, txIn := range otx.tx.MsgTx().TxIn {
		orphans, exists := mp.orphansByPrev[txIn.PreviousOutPoint]
		if exists {
			delete(orphans, *txHash)

//如果没有
//任何依赖它的孤儿。
			if len(orphans) == 0 {
				delete(mp.orphansByPrev, txIn.PreviousOutPoint)
			}
		}
	}

//如果需要，请删除从该输出中提取输出的所有孤立项。
	if removeRedeemers {
		prevOut := wire.OutPoint{Hash: *txHash}
		for txOutIdx := range tx.MsgTx().TxOut {
			prevOut.Index = uint32(txOutIdx)
			for _, orphan := range mp.orphansByPrev[prevOut] {
				mp.removeOrphan(orphan, true)
			}
		}
	}

//从孤立池中删除事务。
	delete(mp.orphans, *txHash)
}

//RemoveOrphan从孤儿池中移除经过的孤儿事务。
//上一个孤立索引。
//
//此函数对于并发访问是安全的。
func (mp *TxPool) RemoveOrphan(tx *btcutil.Tx) {
	mp.mtx.Lock()
	mp.removeOrphan(tx, false)
	mp.mtx.Unlock()
}

//RemoveOrphansByTag removes all orphan transactions tagged with the provided
//标识符。
//
//此函数对于并发访问是安全的。
func (mp *TxPool) RemoveOrphansByTag(tag Tag) uint64 {
	var numEvicted uint64
	mp.mtx.Lock()
	for _, otx := range mp.orphans {
		if otx.tag == tag {
			mp.removeOrphan(otx.tx, true)
			numEvicted++
		}
	}
	mp.mtx.Unlock()
	return numEvicted
}

//limitNumOrphans通过逐出一个随机的
//如果添加新的将导致溢出允许的最大值，则为孤立。
//
//必须在保持mempool锁的情况下调用此函数（用于写入）。
func (mp *TxPool) limitNumOrphans() error {
//扫描孤立池并删除所有过期的孤立池
//时间。这样做是为了提高效率，所以只能进行扫描
//定期而不是对添加到池中的每个孤立对象。
	if now := time.Now(); now.After(mp.nextExpireScan) {
		origNumOrphans := len(mp.orphans)
		for _, otx := range mp.orphans {
			if now.After(otx.expiration) {
//因为丢失了
//父母不太可能实现
//since the orphan has already been around more
//足够长的时间来交付。
				mp.removeOrphan(otx.tx, true)
			}
		}

//将下一次过期扫描设置为在扫描间隔之后进行。
		mp.nextExpireScan = now.Add(orphanExpireScanInterval)

		numOrphans := len(mp.orphans)
		if numExpired := origNumOrphans - numOrphans; numExpired > 0 {
			log.Debugf("Expired %d %s (remaining: %d)", numExpired,
				pickNoun(numExpired, "orphan", "orphans"),
				numOrphans)
		}
	}

//Nothing to do if adding another orphan will not cause the pool to
//超出限制。
	if len(mp.orphans)+1 <= mp.cfg.Policy.MaxOrphanTxs {
		return nil
	}

//从地图中删除一个随机条目。对于大多数编译器，go's
//range statement iterates starting at a random item although
//这不是规范100%保证的。迭代顺序
//在这里并不重要，因为对手必须
//能够在
//以任何方式将特定条目逐出为目标。
	for _, otx := range mp.orphans {
//在随机驱逐的情况下，不要移除赎回人，因为
//很可能很快又需要它。
		mp.removeOrphan(otx.tx, false)
		break
	}

	return nil
}

//addorphan将孤立事务添加到孤立池中。
//
//必须在保持mempool锁的情况下调用此函数（用于写入）。
func (mp *TxPool) addOrphan(tx *btcutil.Tx, tag Tag) {
//如果不允许孤儿，就不做任何事。
	if mp.cfg.Policy.MaxOrphanTxs <= 0 {
		return
	}

//限制孤立事务数以防止内存耗尽。
//这将定期删除所有过期的孤儿并随机逐出
//如果仍然需要空间，则为孤立。
	mp.limitNumOrphans()

	mp.orphans[*tx.Hash()] = &orphanTx{
		tx:         tx,
		tag:        tag,
		expiration: time.Now().Add(orphanTTL),
	}
	for _, txIn := range tx.MsgTx().TxIn {
		if _, exists := mp.orphansByPrev[txIn.PreviousOutPoint]; !exists {
			mp.orphansByPrev[txIn.PreviousOutPoint] =
				make(map[chainhash.Hash]*btcutil.Tx)
		}
		mp.orphansByPrev[txIn.PreviousOutPoint][*tx.Hash()] = tx
	}

	log.Debugf("Stored orphan transaction %v (total: %d)", tx.Hash(),
		len(mp.orphans))
}

//maybeaddorphan可能会将孤儿添加到孤儿池中。
//
//必须在保持mempool锁的情况下调用此函数（用于写入）。
func (mp *TxPool) maybeAddOrphan(tx *btcutil.Tx, tag Tag) error {
//忽略太大的孤立事务。这有助于避免
//一种基于发送大量非常大数据的内存耗尽攻击
//孤儿。如果存在大于此值的有效事务，
//它最终将在母公司交易后重新广播。
//已开采或以其他方式接收。
//
//Note that the number of orphan transactions in the orphan pool is
//也有限，所以这相当于使用的最大内存
//mp.cfg.policy.maxorphantxsize*mp.cfg.policy.maxorphantxs（约5 MB
//使用编写此注释时的默认值）。
	serializedLen := tx.MsgTx().SerializeSize()
	if serializedLen > mp.cfg.Policy.MaxOrphanTxSize {
		str := fmt.Sprintf("orphan transaction size of %d bytes is "+
			"larger than max allowed size of %d bytes",
			serializedLen, mp.cfg.Policy.MaxOrphanTxSize)
		return txRuleError(wire.RejectNonstandard, str)
	}

//如果以上都不合格，则添加孤儿。
	mp.addOrphan(tx, tag)

	return nil
}

//removeOrphanDoublepends删除所有使用
//从孤儿池中传递事务。去除那些孤儿然后引线
//递归地删除所有依赖它们的孤儿。这是必要的
//当事务被添加到主池中时，因为它可能花费输出
//那些孤儿也会花钱。
//
//必须在保持mempool锁的情况下调用此函数（用于写入）。
func (mp *TxPool) removeOrphanDoubleSpends(tx *btcutil.Tx) {
	msgTx := tx.MsgTx()
	for _, txIn := range msgTx.TxIn {
		for _, orphan := range mp.orphansByPrev[txIn.PreviousOutPoint] {
			mp.removeOrphan(orphan, true)
		}
	}
}

//ItRANSANACONIONPOUNT返回是否已通过事务
//存在于主池中。
//
//必须在保持mempool锁的情况下调用此函数（用于读取）。
func (mp *TxPool) isTransactionInPool(hash *chainhash.Hash) bool {
	if _, exists := mp.pool[*hash]; exists {
		return true
	}

	return false
}

//ISTransactionInPool返回是否已传递事务
//存在于主池中。
//
//此函数对于并发访问是安全的。
func (mp *TxPool) IsTransactionInPool(hash *chainhash.Hash) bool {
//保护并发访问。
	mp.mtx.RLock()
	inPool := mp.isTransactionInPool(hash)
	mp.mtx.RUnlock()

	return inPool
}

//isorphaninpool返回传递的事务是否已存在
//在孤儿池里。
//
//必须在保持mempool锁的情况下调用此函数（用于读取）。
func (mp *TxPool) isOrphanInPool(hash *chainhash.Hash) bool {
	if _, exists := mp.orphans[*hash]; exists {
		return true
	}

	return false
}

//isorphaninpool返回传递的事务是否已存在
//在孤儿池里。
//
//此函数对于并发访问是安全的。
func (mp *TxPool) IsOrphanInPool(hash *chainhash.Hash) bool {
//保护并发访问。
	mp.mtx.RLock()
	inPool := mp.isOrphanInPool(hash)
	mp.mtx.RUnlock()

	return inPool
}

//HaveTransaction返回传递的事务是否已存在
//在主游泳池或孤儿游泳池。
//
//必须在保持mempool锁的情况下调用此函数（用于读取）。
func (mp *TxPool) haveTransaction(hash *chainhash.Hash) bool {
	return mp.isTransactionInPool(hash) || mp.isOrphanInPool(hash)
}

//HaveTransaction returns whether or not the passed transaction already exists
//在主游泳池或孤儿游泳池。
//
//此函数对于并发访问是安全的。
func (mp *TxPool) HaveTransaction(hash *chainhash.Hash) bool {
//保护并发访问。
	mp.mtx.RLock()
	haveTx := mp.haveTransaction(hash)
	mp.mtx.RUnlock()

	return haveTx
}

//removeTransaction is the internal function which implements the public
//拆下变速器。有关详细信息，请参阅removeTransaction的注释。
//
//必须在保持mempool锁的情况下调用此函数（用于写入）。
func (mp *TxPool) removeTransaction(tx *btcutil.Tx, removeRedeemers bool) {
	txHash := tx.Hash()
	if removeRedeemers {
//Remove any transactions which rely on this one.
		for i := uint32(0); i < uint32(len(tx.MsgTx().TxOut)); i++ {
			prevOut := wire.OutPoint{Hash: *txHash, Index: i}
			if txRedeemer, exists := mp.outpoints[prevOut]; exists {
				mp.removeTransaction(txRedeemer, true)
			}
		}
	}

//如果需要，删除事务。
	if txDesc, exists := mp.pool[*txHash]; exists {
//删除与
//事务（如果启用）。
		if mp.cfg.AddrIndex != nil {
			mp.cfg.AddrIndex.RemoveUnconfirmedTx(txHash)
		}

//将引用的输出点标记为池未占用。
		for _, txIn := range txDesc.Tx.MsgTx().TxIn {
			delete(mp.outpoints, txIn.PreviousOutPoint)
		}
		delete(mp.pool, *txHash)
		atomic.StoreInt64(&mp.lastUpdated, time.Now().Unix())
	}
}

//removeTransaction从mempool中删除传递的事务。当
//设置了removeedemers标志，任何从
//删除的事务也将从mempool中递归删除，如
//否则他们就会变成孤儿。
//
//此函数对于并发访问是安全的。
func (mp *TxPool) RemoveTransaction(tx *btcutil.Tx, removeRedeemers bool) {
//保护并发访问。
	mp.mtx.Lock()
	mp.removeTransaction(tx, removeRedeemers)
	mp.mtx.Unlock()
}

//RemovedoublePends删除所有花费输出的事务
//已从内存池传递事务。然后删除这些事务
//导致递归地删除所有依赖它们的事务。这是
//当一个滑轮连接到主链时是必要的，因为滑轮可能
//包含以前内存池未知的事务。
//
//此函数对于并发访问是安全的。
func (mp *TxPool) RemoveDoubleSpends(tx *btcutil.Tx) {
//保护并发访问。
	mp.mtx.Lock()
	for _, txIn := range tx.MsgTx().TxIn {
		if txRedeemer, ok := mp.outpoints[txIn.PreviousOutPoint]; ok {
			if !txRedeemer.Hash().IsEqual(tx.Hash()) {
				mp.removeTransaction(txRedeemer, true)
			}
		}
	}
	mp.mtx.Unlock()
}

//addTransaction将传递的事务添加到内存池中。它应该
//不直接调用，因为它不执行任何验证。这是一个
//maybeceptTransaction的帮助程序。
//
//必须在保持mempool锁的情况下调用此函数（用于写入）。
func (mp *TxPool) addTransaction(utxoView *blockchain.UtxoViewpoint, tx *btcutil.Tx, height int32, fee int64) *TxDesc {
//将事务添加到池并标记引用的输出点
//在泳池边度过的时光。
	txD := &TxDesc{
		TxDesc: mining.TxDesc{
			Tx:       tx,
			Added:    time.Now(),
			Height:   height,
			Fee:      fee,
			FeePerKB: fee * 1000 / GetTxVirtualSize(tx),
		},
		StartingPriority: mining.CalcPriority(tx.MsgTx(), utxoView, height),
	}

	mp.pool[*tx.Hash()] = txD
	for _, txIn := range tx.MsgTx().TxIn {
		mp.outpoints[txIn.PreviousOutPoint] = tx
	}
	atomic.StoreInt64(&mp.lastUpdated, time.Now().Unix())

//添加与事务关联的未确认地址索引项
//如果启用。
	if mp.cfg.AddrIndex != nil {
		mp.cfg.AddrIndex.AddUnconfirmedTx(tx, utxoView)
	}

//Record this tx for fee estimation if enabled.
	if mp.cfg.FeeEstimator != nil {
		mp.cfg.FeeEstimator.ObserveTransaction(txD)
	}

	return txD
}

//checkpoolDoublesPend检查传递的事务是否
//试图在池中花费其他交易已经花费的硬币。
//注意，它不检查已在
//主链。
//
//必须在保持mempool锁的情况下调用此函数（用于读取）。
func (mp *TxPool) checkPoolDoubleSpend(tx *btcutil.Tx) error {
	for _, txIn := range tx.MsgTx().TxIn {
		if txR, exists := mp.outpoints[txIn.PreviousOutPoint]; exists {
			str := fmt.Sprintf("output %v already spent by "+
				"transaction %v in the memory pool",
				txIn.PreviousOutPoint, txR.Hash())
			return txRuleError(wire.RejectDuplicate, str)
		}
	}

	return nil
}

//checkSpend检查通过的输出点是否已由
//内存池中的事务。如果是这样的话，支出交易将
//如果不是零，将被退回。
func (mp *TxPool) CheckSpend(op wire.OutPoint) *btcutil.Tx {
	mp.mtx.RLock()
	txR := mp.outpoints[op]
	mp.mtx.RUnlock()

	return txR
}

//fetchinputxos加载有关由引用的输入事务的utxo详细信息
//已传递的事务。首先，它从
//主链，然后根据
//事务池。
//
//必须在保持mempool锁的情况下调用此函数（用于读取）。
func (mp *TxPool) fetchInputUtxos(tx *btcutil.Tx) (*blockchain.UtxoViewpoint, error) {
	utxoView, err := mp.cfg.FetchUtxoView(tx)
	if err != nil {
		return nil, err
	}

//尝试填充事务池中缺少的任何输入。
	for _, txIn := range tx.MsgTx().TxIn {
		prevOut := &txIn.PreviousOutPoint
		entry := utxoView.LookupEntry(*prevOut)
		if entry != nil && !entry.IsSpent() {
			continue
		}

		if poolTxDesc, exists := mp.pool[prevOut.Hash]; exists {
//addtxout忽略超出范围的索引值，因此
//在这里可以不受限制地调用。
			utxoView.AddTxOut(poolTxDesc.Tx, prevOut.Index,
				mining.UnminedHeight)
		}
	}

	return utxoView, nil
}

//fetchTransaction从事务池返回请求的事务。
//这只从主事务池中提取，不包括
//孤儿。
//
//此函数对于并发访问是安全的。
func (mp *TxPool) FetchTransaction(txHash *chainhash.Hash) (*btcutil.Tx, error) {
//保护并发访问。
	mp.mtx.RLock()
	txDesc, exists := mp.pool[*txHash]
	mp.mtx.RUnlock()

	if exists {
		return txDesc.Tx, nil
	}

	return nil, fmt.Errorf("transaction is not in the pool")
}

//maybeceptTransaction是实现公共
//可能接受事务。请参阅maybecepttransaction的注释
//更多细节。
//
//必须在保持mempool锁的情况下调用此函数（用于写入）。
func (mp *TxPool) maybeAcceptTransaction(tx *btcutil.Tx, isNew, rateLimit, rejectDupOrphans bool) ([]*chainhash.Hash, *TxDesc, error) {
	txHash := tx.Hash()

//如果事务具有Iwtness数据，且segwit尚未激活，则
//Segwit还没有激活，那么我们不会接受它作为
//它还不能开采。
	if tx.MsgTx().HasWitness() {
		segwitActive, err := mp.cfg.IsDeploymentActive(chaincfg.DeploymentSegwit)
		if err != nil {
			return nil, nil, err
		}

		if !segwitActive {
			str := fmt.Sprintf("transaction %v has witness data, "+
				"but segwit isn't active yet", txHash)
			return nil, nil, txRuleError(wire.RejectNonstandard, str)
		}
	}

//如果事务已存在于池中，则不接受该事务。这个
//当拒绝重复项
//orphans flag is set.  This check is intended to be a quick check to
//剔除重复项。
	if mp.isTransactionInPool(txHash) || (rejectDupOrphans &&
		mp.isOrphanInPool(txHash)) {

		str := fmt.Sprintf("already have transaction %v", txHash)
		return nil, nil, txRuleError(wire.RejectDuplicate, str)
	}

//对交易进行初步的健全性检查。这使得
//使用包含不变规则的区块链
//允许将事务分成块。
	err := blockchain.CheckTransactionSanity(tx)
	if err != nil {
		if cerr, ok := err.(blockchain.RuleError); ok {
			return nil, nil, chainRuleError(cerr)
		}
		return nil, nil, err
	}

//独立事务不能是CoinBase事务。
	if blockchain.IsCoinBase(tx) {
		str := fmt.Sprintf("transaction %v is an individual coinbase",
			txHash)
		return nil, nil, txRuleError(wire.RejectInvalid, str)
	}

//获取主链的当前高度。独立事务
//最多将开采到下一个区块，因此其高度至少为
//比当前高度多一个。
	bestHeight := mp.cfg.BestHeight()
	nextBlockHeight := bestHeight + 1

	medianTimePast := mp.cfg.MedianTimePast()

//如果网络参数为
//禁止他们接受。
	if !mp.cfg.Policy.AcceptNonStd {
		err = checkTransactionStandard(tx, nextBlockHeight,
			medianTimePast, mp.cfg.Policy.MinRelayTxFee,
			mp.cfg.Policy.MaxTxVersion)
		if err != nil {
//尝试从错误中提取拒绝代码，因此
//可以保留。如果不可能，返回
//非标准错误。
			rejectCode, found := extractRejectCode(err)
			if !found {
				rejectCode = wire.RejectNonstandard
			}
			str := fmt.Sprintf("transaction %v is not standard: %v",
				txHash, err)
			return nil, nil, txRuleError(rejectCode, str)
		}
	}

//该事务不能使用与其他事务相同的任何输出
//已经存在于池中的事务，因为这最终会导致
//双花钱。此检查旨在快速进行，因此仅
//检测事务池本身的双倍开销。这个
//交易仍可能是来自主链的双倍消费硬币
//在这一点上。稍后会进行更深入的检查
//从主链获取引用的事务输入后
//它检查实际支出数据并防止重复支出。
	err = mp.checkPoolDoubleSpend(tx)
	if err != nil {
		return nil, nil, err
	}

//获取输入引用的所有未暂停事务输出
//到这个交易。此函数还尝试获取
//用于检测重复事务的事务本身
//不需要单独查找。
	utxoView, err := mp.fetchInputUtxos(tx)
	if err != nil {
		if cerr, ok := err.(blockchain.RuleError); ok {
			return nil, nil, chainRuleError(cerr)
		}
		return nil, nil, err
	}

//如果事务存在于主链中且不存在，则不允许该事务
//还没有完全花光。
	prevOut := wire.OutPoint{Hash: *txHash}
	for txOutIdx := range tx.MsgTx().TxOut {
		prevOut.Index = uint32(txOutIdx)
		entry := utxoView.LookupEntry(prevOut)
		if entry != nil && !entry.IsSpent() {
			return nil, nil, txRuleError(wire.RejectDuplicate,
				"transaction already exists")
		}
		utxoView.RemoveEntry(prevOut)
	}

//如果任何引用的事务输出，则事务是孤立的
//不存在或者已经花掉了。将孤立项添加到孤立项池
//不是由该函数处理的，调用方应使用
//如果需要此行为，则可能是孤立的。
	var missingParents []*chainhash.Hash
	for outpoint, entry := range utxoView.Entries() {
		if entry == nil || entry.IsSpent() {
//必须在此处复制哈希，因为迭代器
//被替换，直接获取其地址
//导致所有条目指向同一个
//内存位置，因此都是最终的散列值。
			hashCopy := outpoint.Hash
			missingParents = append(missingParents, &hashCopy)
		}
	}
	if len(missingParents) > 0 {
		return missingParents, nil, nil
	}

//不允许事务进入mempool，除非它的序列
//锁处于活动状态，这意味着它将被允许进入下一个块
//关于其定义的相对锁定时间。
	sequenceLock, err := mp.cfg.CalcSequenceLock(tx, utxoView)
	if err != nil {
		if cerr, ok := err.(blockchain.RuleError); ok {
			return nil, nil, chainRuleError(cerr)
		}
		return nil, nil, err
	}
	if !blockchain.SequenceLockActive(sequenceLock, nextBlockHeight,
		medianTimePast) {
		return nil, nil, txRuleError(wire.RejectNonstandard,
			"transaction's sequence locks on inputs not met")
	}

//使用不变量对事务输入执行多个检查
//区块链中允许哪些交易成为区块的规则。
//同时返回与交易相关的费用
//后来使用。
	txFee, err := blockchain.CheckTransactionInputs(tx, nextBlockHeight,
		utxoView, mp.cfg.ChainParams)
	if err != nil {
		if cerr, ok := err.(blockchain.RuleError); ok {
			return nil, nil, chainRuleError(cerr)
		}
		return nil, nil, err
	}

//如果网络
//参数禁止接受。
	if !mp.cfg.Policy.AcceptNonStd {
		err := checkInputsStandard(tx, utxoView)
		if err != nil {
//尝试从错误中提取拒绝代码，因此
//可以保留。如果不可能，返回
//非标准错误。
			rejectCode, found := extractRejectCode(err)
			if !found {
				rejectCode = wire.RejectNonstandard
			}
			str := fmt.Sprintf("transaction %v has a non-standard "+
				"input: %v", txHash, err)
			return nil, nil, txRuleError(rejectCode, str)
		}
	}

//注意：如果修改此代码以接受非标准事务，
//您应该在此处添加代码以检查事务是否执行
//ECDSA签名验证的合理数量。

//不允许签名过多的事务
//导致无法开采的作业。自从
//coinbase地址本身可以包含签名操作，
//每个事务允许的最大签名操作数小于
//每个块允许的最大签名操作数。
//TODO（roasbef）：最后一个bool应以segwit激活为条件。
	sigOpCost, err := blockchain.GetSigOpCost(tx, false, utxoView, true, true)
	if err != nil {
		if cerr, ok := err.(blockchain.RuleError); ok {
			return nil, nil, chainRuleError(cerr)
		}
		return nil, nil, err
	}
	if sigOpCost > mp.cfg.Policy.MaxSigOpCostPerTx {
		str := fmt.Sprintf("transaction %v sigop cost is too high: %d > %d",
			txHash, sigOpCost, mp.cfg.Policy.MaxSigOpCostPerTx)
		return nil, nil, txRuleError(wire.RejectNonstandard, str)
	}

//不允许费用太低的交易进入开采区。
//
//大多数矿工允许他们开采的区块内有一个自由交易区。
//以及用于高优先级事务的区域
//收费交易。高达1000字节的事务大小是
//被认为可以安全进入这一部分。此外，最低费用
//下面单独计算会鼓励
//避免费用的交易，而不是单笔较大的交易
//更可取的是。因此，只要
//交易记录的保留空间不超过1000
//高优先级交易，不需要支付费用。
	serializedSize := GetTxVirtualSize(tx)
	minFee := calcMinRequiredTxRelayFee(serializedSize,
		mp.cfg.Policy.MinRelayTxFee)
	if serializedSize >= (DefaultBlockPrioritySize-1000) && txFee < minFee {
		str := fmt.Sprintf("transaction %v has %d fees which is under "+
			"the required amount of %d", txHash, txFee,
			minFee)
		return nil, nil, txRuleError(wire.RejectInsufficientFee, str)
	}

//要求自由交易有足够的优先权进行挖掘
//在下一个街区。正在添加回
//内存池来自在REORG期间断开连接的块
//被豁免。
	if isNew && !mp.cfg.Policy.DisableRelayPriority && txFee < minFee {
		currentPriority := mining.CalcPriority(tx.MsgTx(), utxoView,
			nextBlockHeight)
		if currentPriority <= mining.MinHighPriority {
			str := fmt.Sprintf("transaction %v has insufficient "+
				"priority (%g <= %g)", txHash,
				currentPriority, mining.MinHighPriority)
			return nil, nil, txRuleError(wire.RejectInsufficientFee, str)
		}
	}

//免费中继交易的费率在这里受到限制，以防止
//小额交易作为攻击的一种形式泛滥。
	if rateLimit && txFee < minFee {
		nowUnix := time.Now().Unix()
//衰减传递的数据指数衰减约10分钟
//窗口-匹配比特币处理。
		mp.pennyTotal *= math.Pow(1.0-1.0/600.0,
			float64(nowUnix-mp.lastPennyUnix))
		mp.lastPennyUnix = nowUnix

//我们还超过限额吗？
		if mp.pennyTotal >= mp.cfg.Policy.FreeTxRelayLimit*10*1000 {
			str := fmt.Sprintf("transaction %v has been rejected "+
				"by the rate limiter due to low fees", txHash)
			return nil, nil, txRuleError(wire.RejectInsufficientFee, str)
		}
		oldTotal := mp.pennyTotal

		mp.pennyTotal += float64(serializedSize)
		log.Tracef("rate limit: curTotal %v, nextTotal: %v, "+
			"limit %v", oldTotal, mp.pennyTotal,
			mp.cfg.Policy.FreeTxRelayLimit*10*1000)
	}

//验证每个输入的加密签名，如果
//任何不核实。
	err = blockchain.ValidateTransactionScripts(tx, utxoView,
		txscript.StandardVerifyFlags, mp.cfg.SigCache,
		mp.cfg.HashCache)
	if err != nil {
		if cerr, ok := err.(blockchain.RuleError); ok {
			return nil, nil, chainRuleError(cerr)
		}
		return nil, nil, err
	}

//添加到事务池。
	txD := mp.addTransaction(utxoView, tx, bestHeight, txFee)

	log.Debugf("Accepted transaction %v (pool size: %v)", txHash,
		len(mp.pool))

	return nil, txD, nil
}

//maybeceptTransaction是处理插入新的
//将独立的事务放入内存池。它包括功能
//例如拒绝重复的交易，确保交易遵循所有
//规则、检测孤立事务和插入内存池。
//
//如果事务是孤立的（缺少父事务），则
//未将事务添加到孤立池，但引用了每个未知的
//parent is returned.  Use ProcessTransaction instead if new orphans should
//添加到孤立池。
//
//此函数对于并发访问是安全的。
func (mp *TxPool) MaybeAcceptTransaction(tx *btcutil.Tx, isNew, rateLimit bool) ([]*chainhash.Hash, *TxDesc, error) {
//保护并发访问。
	mp.mtx.Lock()
	hashes, txD, err := mp.maybeAcceptTransaction(tx, isNew, rateLimit, true)
	mp.mtx.Unlock()

	return hashes, txD, err
}

//processOrphans是实现公共
//进程孤立。有关详细信息，请参阅ProcessOrphans的注释。
//
//必须在保持mempool锁的情况下调用此函数（用于写入）。
func (mp *TxPool) processOrphans(acceptedTx *btcutil.Tx) []*TxDesc {
	var acceptedTxns []*TxDesc

//至少从处理传递的事务开始。
	processList := list.New()
	processList.PushBack(acceptedTx)
	for processList.Len() > 0 {
//从列表前面弹出要处理的事务。
		firstElement := processList.Remove(processList.Front())
		processItem := firstElement.(*btcutil.Tx)

		prevOut := wire.OutPoint{Hash: *processItem.Hash()}
		for txOutIdx := range processItem.MsgTx().TxOut {
//Look up all orphans that redeem the output that is
//现已推出。这通常只有一个，但是
//如果孤立池包含
//双花钱。While it may seem odd that the orphan
//游泳池会允许这样做，因为只有可能
//最终成为一个救世主，重要的是
//以这种方式跟踪，以防止恶意参与者
//能够有目的地建立孤儿
//否则将使输出不可依赖。
//
//如果没有，跳到下一个可用输出。
			prevOut.Index = uint32(txOutIdx)
			orphans, exists := mp.orphansByPrev[prevOut]
			if !exists {
				continue
			}

//可能接受一个孤儿进入TX池。
			for _, tx := range orphans {
				missing, txD, err := mp.maybeAcceptTransaction(
					tx, true, true, false)
				if err != nil {
//The orphan is now invalid, so there
//任何其他孤儿都不可能
//兑现它的任何输出可以是
//认可的。除去它们。
					mp.removeOrphan(tx, true)
					break
				}

//事务仍然是孤立的。尝试下一步
//重设此输出的孤立项。
				if len(missing) > 0 {
					continue
				}

//事务已被接受到主池中。
//
//将其添加到接受的交易列表中
//不再是孤儿，请将其从
//孤立池，并将其添加到
//要处理的事务，以便
//依靠它也被处理。
				acceptedTxns = append(acceptedTxns, txD)
				mp.removeOrphan(tx, false)
				processList.PushBack(tx)

//此输出点只能有一个事务
//接受了，所以剩下的是双倍的花费
//稍后移除。
				break
			}
		}
	}

//以递归方式删除同时兑现任何已兑现输出的所有孤立项
//被接受的交易，因为这些现在是决定性的双倍
//花费。
	mp.removeOrphanDoubleSpends(acceptedTx)
	for _, txD := range acceptedTxns {
		mp.removeOrphanDoubleSpends(txD.Tx)
	}

	return acceptedTxns
}

//processOrphans确定是否有依赖于传递的
//事务哈希（可能它们不再是孤立的）和
//可能接受它们到内存池。它重复了
//新接受的交易（检测可能不再是
//孤儿）直到没有了。
//
//它返回添加到mempool的事务切片。零片意味着
//没有将事务从孤立池移动到mempool。
//
//此函数对于并发访问是安全的。
func (mp *TxPool) ProcessOrphans(acceptedTx *btcutil.Tx) []*TxDesc {
	mp.mtx.Lock()
	acceptedTxns := mp.processOrphans(acceptedTx)
	mp.mtx.Unlock()

	return acceptedTxns
}

//processTransaction是处理插入新的
//内存池中的独立事务。它包括功能
//例如拒绝重复的交易，确保交易遵循所有
//规则、孤立事务处理和插入内存池。
//
//它返回添加到mempool的事务切片。当
//错误为零，列表将包含传递的事务本身
//由于以下原因而添加的任何其他孤立传输
//通过的那个被接受了。
//
//此函数对于并发访问是安全的。
func (mp *TxPool) ProcessTransaction(tx *btcutil.Tx, allowOrphan, rateLimit bool, tag Tag) ([]*TxDesc, error) {
	log.Tracef("Processing transaction %v", tx.Hash())

//保护并发访问。
	mp.mtx.Lock()
	defer mp.mtx.Unlock()

//可能接受到内存池的事务。
	missingParents, txD, err := mp.maybeAcceptTransaction(tx, true, rateLimit,
		true)
	if err != nil {
		return nil, err
	}

	if len(missingParents) == 0 {
//接受任何依赖于此的孤立事务
//事务（如果所有输入都是孤立的，则它们可能不再是孤立的
//现在可以使用）并对接受的重复
//直到没有更多的交易。
		newTxs := mp.processOrphans(tx)
		acceptedTxs := make([]*TxDesc, len(newTxs)+1)

//首先添加父事务，以便远程节点
//不要添加孤立项。
		acceptedTxs[0] = txD
		copy(acceptedTxs[1:], newTxs)

		return acceptedTxs, nil
	}

//事务是孤立的（缺少输入）。拒绝
//如果未设置允许孤立项的标志，则返回。
	if !allowOrphan {
//仅使用中第一个缺少的父事务
//错误消息。
//
//注意：rejectDuplicate确实不准确
//此处拒绝代码，但它与引用匹配
//实施，没有更好的选择
//拒绝代码的数量有限。遗失
//假设输入意味着它们已经花掉了。
//事实并非总是如此。
		str := fmt.Sprintf("orphan transaction %v references "+
			"outputs of unknown or fully-spent "+
			"transaction %v", tx.Hash(), missingParents[0])
		return nil, txRuleError(wire.RejectDuplicate, str)
	}

//可能会将孤立事务添加到孤立池中。
	err = mp.maybeAddOrphan(tx, tag)
	return nil, err
}

//COUNT返回主池中的事务数。它不
//包括孤儿池。
//
//此函数对于并发访问是安全的。
func (mp *TxPool) Count() int {
	mp.mtx.RLock()
	count := len(mp.pool)
	mp.mtx.RUnlock()

	return count
}

//txshashes返回内存中所有事务的哈希切片
//池。
//
//此函数对于并发访问是安全的。
func (mp *TxPool) TxHashes() []*chainhash.Hash {
	mp.mtx.RLock()
	hashes := make([]*chainhash.Hash, len(mp.pool))
	i := 0
	for hash := range mp.pool {
		hashCopy := hash
		hashes[i] = &hashCopy
		i++
	}
	mp.mtx.RUnlock()

	return hashes
}

//TxDescs返回池中所有事务的一部分描述符。
//描述符将被视为只读的。
//
//此函数对于并发访问是安全的。
func (mp *TxPool) TxDescs() []*TxDesc {
	mp.mtx.RLock()
	descs := make([]*TxDesc, len(mp.pool))
	i := 0
	for _, desc := range mp.pool {
		descs[i] = desc
		i++
	}
	mp.mtx.RUnlock()

	return descs
}

//miningdescs返回所有事务的挖掘描述符切片
//在游泳池里。
//
//This is part of the mining.TxSource interface implementation and is safe for
//接口合同要求的并发访问。
func (mp *TxPool) MiningDescs() []*mining.TxDesc {
	mp.mtx.RLock()
	descs := make([]*mining.TxDesc, len(mp.pool))
	i := 0
	for _, desc := range mp.pool {
		descs[i] = &desc.TxDesc
		i++
	}
	mp.mtx.RUnlock()

	return descs
}

//rawmEmpoolVerbose将mempool中的所有条目作为
//已填充btcjson结果。
//
//此函数对于并发访问是安全的。
func (mp *TxPool) RawMempoolVerbose() map[string]*btcjson.GetRawMempoolVerboseResult {
	mp.mtx.RLock()
	defer mp.mtx.RUnlock()

	result := make(map[string]*btcjson.GetRawMempoolVerboseResult,
		len(mp.pool))
	bestHeight := mp.cfg.BestHeight()

	for _, desc := range mp.pool {
//根据输入计算当前优先级
//交易。如果一个或多个
//由于某些原因，找不到输入交易记录。
		tx := desc.Tx
		var currentPriority float64
		utxos, err := mp.fetchInputUtxos(tx)
		if err == nil {
			currentPriority = mining.CalcPriority(tx.MsgTx(), utxos,
				bestHeight+1)
		}

		mpd := &btcjson.GetRawMempoolVerboseResult{
			Size:             int32(tx.MsgTx().SerializeSize()),
			Vsize:            int32(GetTxVirtualSize(tx)),
			Fee:              btcutil.Amount(desc.Fee).ToBTC(),
			Time:             desc.Added.Unix(),
			Height:           int64(desc.Height),
			StartingPriority: desc.StartingPriority,
			CurrentPriority:  currentPriority,
			Depends:          make([]string, 0),
		}
		for _, txIn := range tx.MsgTx().TxIn {
			hash := &txIn.PreviousOutPoint.Hash
			if mp.haveTransaction(hash) {
				mpd.Depends = append(mpd.Depends,
					hash.String())
			}
		}

		result[tx.Hash().String()] = mpd
	}

	return result
}

//LastUpdated返回上次向事务添加或从中删除事务的时间
//主水池。它不包括孤立池。
//
//此函数对于并发访问是安全的。
func (mp *TxPool) LastUpdated() time.Time {
	return time.Unix(atomic.LoadInt64(&mp.lastUpdated), 0)
}

//new返回一个新的内存池，用于独立验证和存储
//直到它们被挖掘成一个块为止。
func New(cfg *Config) *TxPool {
	return &TxPool{
		cfg:            *cfg,
		pool:           make(map[chainhash.Hash]*TxDesc),
		orphans:        make(map[chainhash.Hash]*orphanTx),
		orphansByPrev:  make(map[wire.OutPoint]map[chainhash.Hash]*btcutil.Tx),
		nextExpireScan: time.Now().Add(orphanExpireScanInterval),
		outpoints:      make(map[wire.OutPoint]*btcutil.Tx),
	}
}
