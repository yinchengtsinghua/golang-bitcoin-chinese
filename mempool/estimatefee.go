
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package mempool

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"sort"
	"strings"
	"sync"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/mining"
	"github.com/btcsuite/btcutil"
)

//托多将亚历克斯·莫科斯对加文最初模型的修改合并在一起。
//https://lists.linuxfoundation.org/pipermail/bitcoin-dev/2014-10月/006824.html

const (
//estimateFeeDepth is the maximum number of blocks before a transaction
//确认我们要跟踪。
	estimateFeeDepth = 25

//EstimateFeebinSize是存储在每个容器中的Tx的数目。
	estimateFeeBinSize = 100

//EstimateFeeMaxReplacements是
//可以通过在给定块中找到的TXS来生成。
	estimateFeeMaxReplacements = 10

//DefaultEstimateFeeMaxRollback是默认的回滚数
//由孤儿块的费用估计器所允许。
	DefaultEstimateFeeMaxRollback = 2

//DefaultEstimateFeeminRegisteredBlocks是默认的最小值
//在此之前，费用估算员必须遵守的区块数
//它将提供费用估算。
	DefaultEstimateFeeMinRegisteredBlocks = 3

	bytePerKb = 1000

	btcPerSatoshi = 1E-8
)

var (
//EstimateFeeDatabaseKey是我们用来
//在数据库中存储费用估算器。
	EstimateFeeDatabaseKey = []byte("estimatefee")
)

//SatoshiperByte是一个数字，每个字节的单位为Satoshis。
type SatoshiPerByte float64

//btcperkilobyte是以每千字节比特币为单位的数字。
type BtcPerKilobyte float64

//TobTcperkB返回表示给定值的浮点值
//satoshiperbyte转换为每kb的satoshis。
func (rate SatoshiPerByte) ToBtcPerKb() BtcPerKilobyte {
//如果我们的比率是错误值，则返回该值。
	if rate == SatoshiPerByte(-1.0) {
		return -1.0
	}

	return BtcPerKilobyte(float64(rate) * bytePerKb * btcPerSatoshi)
}

//fee返回给定规模的交易的费用
//给定的费率。
func (rate SatoshiPerByte) Fee(size uint32) btcutil.Amount {
//如果我们的比率是错误值，则返回该值。
	if rate == SatoshiPerByte(-1) {
		return btcutil.Amount(-1)
	}

	return btcutil.Amount(float64(rate) * float64(size))
}

//NewSatoshiPerByte creates a SatoshiPerByte from an Amount and a
//字节大小。
func NewSatoshiPerByte(fee btcutil.Amount, size uint32) SatoshiPerByte {
	return SatoshiPerByte(float64(fee) / float64(size))
}

//observedTransaction表示一个观察到的事务和一些
//additional data required for the fee estimation algorithm.
type observedTransaction struct {
//事务哈希。
	hash chainhash.Hash

//在Satoshis中，事务的每字节费用。
	feeRate SatoshiPerByte

//观测到的块高度。
	observed int32

//采矿块的高度。
//如果事务尚未挖掘，则为零。
	mined int32
}

func (o *observedTransaction) Serialize(w io.Writer) {
	binary.Write(w, binary.BigEndian, o.hash)
	binary.Write(w, binary.BigEndian, o.feeRate)
	binary.Write(w, binary.BigEndian, o.observed)
	binary.Write(w, binary.BigEndian, o.mined)
}

func deserializeObservedTransaction(r io.Reader) (*observedTransaction, error) {
	ot := observedTransaction{}

//前32个字节应该是哈希。
	binary.Read(r, binary.BigEndian, &ot.hash)

//接下来的8个是satoshiperbyte
	binary.Read(r, binary.BigEndian, &ot.feeRate)

//接下来是两个uint32。
	binary.Read(r, binary.BigEndian, &ot.observed)
	binary.Read(r, binary.BigEndian, &ot.mined)

	return &ot, nil
}

//registeredBlock has the hash of a block and the list of transactions
//它开采的矿藏曾被Feeestimator观察过。它
//如果调用rollback来逆转注册的效果，则使用
//一个街区
type registeredBlock struct {
	hash         chainhash.Hash
	transactions []*observedTransaction
}

func (rb *registeredBlock) serialize(w io.Writer, txs map[*observedTransaction]uint32) {
	binary.Write(w, binary.BigEndian, rb.hash)

	binary.Write(w, binary.BigEndian, uint32(len(rb.transactions)))
	for _, o := range rb.transactions {
		binary.Write(w, binary.BigEndian, txs[o])
	}
}

//feeestimator管理创建
//费用估算。它对于并发访问是安全的。
type FeeEstimator struct {
	maxRollback uint32
	binSize     int32

//可在单个中进行的最大替换数
//每箱一箱。默认值为EstimateFeeMaxReplacements
	maxReplacements int32

//可在费用中注册的块的最小数目
//在此之前，估计者将提供答案。
	minRegisteredBlocks uint32

//最后一个已知高度。
	lastKnownHeight int32

//已注册的块数。
	numBlocksRegistered uint32

	mtx      sync.RWMutex
	observed map[chainhash.Hash]*observedTransaction
	bin      [estimateFeeDepth][]*observedTransaction

//缓存的估计。
	cached []SatoshiPerByte

//已从容器中移除的事务。这让我们
//在孤立块的情况下还原。
	dropped []*registeredBlock
}

//newfeestimator创建一个feeestimator，最多为其maxrollback块
//可以取消注册并返回错误，除非minRegisteredBlocks
//have been registered with it.
func NewFeeEstimator(maxRollback, minRegisteredBlocks uint32) *FeeEstimator {
	return &FeeEstimator{
		maxRollback:         maxRollback,
		minRegisteredBlocks: minRegisteredBlocks,
		lastKnownHeight:     mining.UnminedHeight,
		binSize:             estimateFeeBinSize,
		maxReplacements:     estimateFeeMaxReplacements,
		observed:            make(map[chainhash.Hash]*observedTransaction),
		dropped:             make([]*registeredBlock, 0, maxRollback),
	}
}

//当在mempool中观察到新事务时，将调用ObserveTransaction。
func (ef *FeeEstimator) ObserveTransaction(t *TxDesc) {
	ef.mtx.Lock()
	defer ef.mtx.Unlock()

//如果我们还没有看到一个街区，我们不知道这个街区是什么时候到的，
//所以我们忽略了它。
	if ef.lastKnownHeight == mining.UnminedHeight {
		return
	}

	hash := *t.Tx.Hash()
	if _, ok := ef.observed[hash]; !ok {
		size := uint32(GetTxVirtualSize(t.Tx))

		ef.observed[hash] = &observedTransaction{
			hash:     hash,
			feeRate:  NewSatoshiPerByte(btcutil.Amount(t.Fee), size),
			observed: t.Height,
			mined:    mining.UnminedHeight,
		}
	}
}

//RegisterBlock通知费用估算员要考虑的新块。
func (ef *FeeEstimator) RegisterBlock(block *btcutil.Block) error {
	ef.mtx.Lock()
	defer ef.mtx.Unlock()

//上一个排序列表无效，请将其删除。
	ef.cached = nil

	height := block.Height()
	if height != ef.lastKnownHeight+1 && ef.lastKnownHeight != mining.UnminedHeight {
		return fmt.Errorf("intermediate block not recorded; current height is %d; new height is %d",
			ef.lastKnownHeight, height)
	}

//更新上一个已知高度。
	ef.lastKnownHeight = height
	ef.numBlocksRegistered++

//在块中随机排序txs。
	transactions := make(map[*btcutil.Tx]struct{})
	for _, t := range block.Transactions() {
		transactions[t] = struct{}{}
	}

//计算每个箱子的替换件数量，这样我们就不会
//更换太多。
	var replacementCounts [estimateFeeDepth]int

//Keep track of which txs were dropped in case of an orphan block.
	dropped := &registeredBlock{
		hash:         *block.Hash(),
		transactions: make([]*observedTransaction, 0, 100),
	}

//穿过街区的TXS。
	for t := range transactions {
		hash := *t.Hash()

//我们在mempool中观察到这个tx了吗？
		o, ok := ef.observed[hash]
		if !ok {
			continue
		}

//将观察到的Tx放入合适的容器中。
		blocksToConfirm := height - o.observed - 1

//如果费用估算器工作正常，就不会发生这种情况，
//但如果有，则返回一个错误。
		if o.mined != mining.UnminedHeight {
			log.Error("Estimate fee: transaction ", hash.String(), " has already been mined")
			return errors.New("Transaction has already been mined")
		}

//This shouldn't happen but check just in case to avoid
//稍后的越界数组索引。
		if blocksToConfirm >= estimateFeeDepth {
			continue
		}

//确保我们每分钟不替换太多的事务。
		if replacementCounts[blocksToConfirm] == int(ef.maxReplacements) {
			continue
		}

		o.mined = height

		replacementCounts[blocksToConfirm]++

		bin := ef.bin[blocksToConfirm]

//删除一个随机元素并用这个新的tx替换它。
		if len(bin) == int(ef.binSize) {
//不要删除我们刚从同一块添加的事务。
			l := int(ef.binSize) - replacementCounts[blocksToConfirm]
			drop := rand.Intn(l)
			dropped.transactions = append(dropped.transactions, bin[drop])

			bin[drop] = bin[l-1]
			bin[l-1] = o
		} else {
			bin = append(bin, o)
		}
		ef.bin[blocksToConfirm] = bin
	}

//通过TMs的内存池已经过长。
	for hash, o := range ef.observed {
		if o.mined == mining.UnminedHeight && height-o.observed >= estimateFeeDepth {
			delete(ef.observed, hash)
		}
	}

//将删除的列表添加到历史记录。
	if ef.maxRollback == 0 {
		return nil
	}

	if uint32(len(ef.dropped)) == ef.maxRollback {
		ef.dropped = append(ef.dropped[1:], dropped)
	} else {
		ef.dropped = append(ef.dropped, dropped)
	}

	return nil
}

//lastknownheight返回注册的最后一个块的高度。
func (ef *FeeEstimator) LastKnownHeight() int32 {
	ef.mtx.Lock()
	defer ef.mtx.Unlock()

	return ef.lastKnownHeight
}

//回滚从feeestimator中注销最近注册的块。
//这可以用来逆转孤立块对费用的影响。
//估计器。允许的最大回滚数由
//最大回滚。
//
//注意：并非所有事务都可以回滚，因为某些事务
//deleted if they have been observed too long ago. That means the result
//如果最后一个块没有
//发生了，但应该足够近。
func (ef *FeeEstimator) Rollback(hash *chainhash.Hash) error {
	ef.mtx.Lock()
	defer ef.mtx.Unlock()

//Find this block in the stack of recent registered blocks.
	var n int
	for n = 1; n <= len(ef.dropped); n++ {
		if ef.dropped[len(ef.dropped)-n].hash.IsEqual(hash) {
			break
		}
	}

	if n > len(ef.dropped) {
		return errors.New("no such block was recently registered")
	}

	for i := 0; i < n; i++ {
		ef.rollback()
	}

	return nil
}

//回滚回滚回滚堆栈中最后一个块的效果
//注册的块。
func (ef *FeeEstimator) rollback() {
//上一个排序列表无效，请将其删除。
	ef.cached = nil

//从堆栈中弹出最后一个已删除tx的列表。
	last := len(ef.dropped) - 1
	if last == -1 {
//无法真正发生，因为导出的调用函数
//only rolls back a block already known to be in the list
//删除的事务数。
		return
	}

	dropped := ef.dropped[last]

//更换TXS时，我们在每个箱子中的位置？
	var replacementCounters [estimateFeeDepth]int

//通过掉块中的TXS。
	for _, o := range dropped.transactions {
//德克萨斯州的哪个垃圾箱？
		blocksToConfirm := o.mined - o.observed - 1

		bin := ef.bin[blocksToConfirm]

		var counter = replacementCounters[blocksToConfirm]

//继续穿过我们离开的那个垃圾箱。
		for {
			if counter >= len(bin) {
//Panic, as we have entered an unrecoverable invalid state.
				panic(errors.New("illegal state: cannot rollback dropped transaction"))
			}

			prev := bin[counter]

			if prev.mined == ef.lastKnownHeight {
				prev.mined = mining.UnminedHeight

				bin[counter] = o

				counter++
				break
			}

			counter++
		}

		replacementCounters[blocksToConfirm] = counter
	}

//继续通过垃圾箱找到其他要移除的TXS
//当他们进入时，没有替换任何其他的。
	for i, j := range replacementCounters {
		for {
			l := len(ef.bin[i])
			if j >= l {
				break
			}

			prev := ef.bin[i][j]

			if prev.mined == ef.lastKnownHeight {
				prev.mined = mining.UnminedHeight

				newBin := append(ef.bin[i][0:j], ef.bin[i][j+1:l]...)
//TODO This line should prevent an unintentional memory
//泄漏，但当它未经处理时会引起恐慌。
//ef.bin[i][j]=无
				ef.bin[i] = newBin

				continue
			}

			j++
		}
	}

	ef.dropped = ef.dropped[0:last]

//The number of blocks the fee estimator has seen is decrimented.
	ef.numBlocksRegistered--
	ef.lastKnownHeight--
}

//EstimateFeeSet是一组可以排序的tx
//按每千字节收费率计算。
type estimateFeeSet struct {
	feeRate []SatoshiPerByte
	bin     [estimateFeeDepth]uint32
}

func (b *estimateFeeSet) Len() int { return len(b.feeRate) }

func (b *estimateFeeSet) Less(i, j int) bool {
	return b.feeRate[i] > b.feeRate[j]
}

func (b *estimateFeeSet) Swap(i, j int) {
	b.feeRate[i], b.feeRate[j] = b.feeRate[j], b.feeRate[i]
}

//EstimateFee返回交易的估计费用
//从现在起在确认栏中确认，给出
//我们收集的数据集。
func (b *estimateFeeSet) estimateFee(confirmations int) SatoshiPerByte {
	if confirmations <= 0 {
		return SatoshiPerByte(math.Inf(1))
	}

	if confirmations > estimateFeeDepth {
		return 0
	}

//我们没有任何交易！
	if len(b.feeRate) == 0 {
		return 0
	}

	var min, max int = 0, 0
	for i := 0; i < confirmations-1; i++ {
		min += int(b.bin[i])
	}

	max = min + int(b.bin[confirmations-1]) - 1
	if max < min {
		max = min
	}
	feeIndex := (min + max) / 2
	if feeIndex >= len(b.feeRate) {
		feeIndex = len(b.feeRate) - 1
	}

	return b.feeRate[feeIndex]
}

//newEstimateFeeSet创建一个临时数据结构，
//可用于查找所有费用估计。
func (ef *FeeEstimator) newEstimateFeeSet() *estimateFeeSet {
	set := &estimateFeeSet{}

	capacity := 0
	for i, b := range ef.bin {
		l := len(b)
		set.bin[i] = uint32(l)
		capacity += l
	}

	set.feeRate = make([]SatoshiPerByte, capacity)

	i := 0
	for _, b := range ef.bin {
		for _, o := range b {
			set.feeRate[i] = o.feeRate
			i++
		}
	}

	sort.Sort(set)

	return set
}

//估计返回所有费用估计的集合从1到估计深度。
//从现在起确认。
func (ef *FeeEstimator) estimates() []SatoshiPerByte {
	set := ef.newEstimateFeeSet()

	estimates := make([]SatoshiPerByte, estimateFeeDepth)
	for i := 0; i < estimateFeeDepth; i++ {
		estimates[i] = set.estimateFee(i + 1)
	}

	return estimates
}

//估计费估计每个字节的费用有一个TX证实了一个给定的
//从现在起的块数。
func (ef *FeeEstimator) EstimateFee(numBlocks uint32) (BtcPerKilobyte, error) {
	ef.mtx.Lock()
	defer ef.mtx.Unlock()

//如果注册的块数低于最小值，则返回
//一个错误。
	if ef.numBlocksRegistered < ef.minRegisteredBlocks {
		return -1, errors.New("not enough blocks have been observed")
	}

	if numBlocks == 0 {
		return -1, errors.New("cannot confirm transaction in zero blocks")
	}

	if numBlocks > estimateFeeDepth {
		return -1, fmt.Errorf(
			"can only estimate fees for up to %d blocks from now",
			estimateFeeBinSize)
	}

//如果没有缓存的结果，则生成它们。
	if ef.cached == nil {
		ef.cached = ef.estimates()
	}

	return ef.cached[int(numBlocks)-1].ToBtcPerKb(), nil
}

//如果序列版本的feeestimator格式发生变化，
//我们使用版本号。如果版本号更改，则不会
//尝试将以前的版本升级到新版本。相反，只是
//开始费用估算。
const estimateFeeSaveVersion = 1

func deserializeRegisteredBlock(r io.Reader, txs map[uint32]*observedTransaction) (*registeredBlock, error) {
	var lenTransactions uint32

	rb := &registeredBlock{}
	binary.Read(r, binary.BigEndian, &rb.hash)
	binary.Read(r, binary.BigEndian, &lenTransactions)

	rb.transactions = make([]*observedTransaction, lenTransactions)

	for i := uint32(0); i < lenTransactions; i++ {
		var index uint32
		binary.Read(r, binary.BigEndian, &index)
		rb.transactions[i] = txs[index]
	}

	return rb, nil
}

//feeestimatorstate表示保存的feeestimator，可以
//用程序早期会话中的数据还原。
type FeeEstimatorState []byte

//observedTxset是一组可以排序的Tx
//通过哈希。它是为了序列化而存在的，因此
//序列化状态总是出现相同的结果。
type observedTxSet []*observedTransaction

func (q observedTxSet) Len() int { return len(q) }

func (q observedTxSet) Less(i, j int) bool {
	return strings.Compare(q[i].hash.String(), q[j].hash.String()) < 0
}

func (q observedTxSet) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
}

//将feestimator的当前状态保存到一个[]字节，
//可以稍后恢复。
func (ef *FeeEstimator) Save() FeeEstimatorState {
	ef.mtx.Lock()
	defer ef.mtx.Unlock()

//要想知道容量应该是多少。
	w := bytes.NewBuffer(make([]byte, 0))

	binary.Write(w, binary.BigEndian, uint32(estimateFeeSaveVersion))

//插入基本参数。
	binary.Write(w, binary.BigEndian, &ef.maxRollback)
	binary.Write(w, binary.BigEndian, &ef.binSize)
	binary.Write(w, binary.BigEndian, &ef.maxReplacements)
	binary.Write(w, binary.BigEndian, &ef.minRegisteredBlocks)
	binary.Write(w, binary.BigEndian, &ef.lastKnownHeight)
	binary.Write(w, binary.BigEndian, &ef.numBlocksRegistered)

//将所有观察到的事务放在一个排序列表中。
	var txCount uint32
	ots := make([]*observedTransaction, len(ef.observed))
	for hash := range ef.observed {
		ots[txCount] = ef.observed[hash]
		txCount++
	}

	sort.Sort(observedTxSet(ots))

	txCount = 0
	observed := make(map[*observedTransaction]uint32)
	binary.Write(w, binary.BigEndian, uint32(len(ef.observed)))
	for _, ot := range ots {
		ot.Serialize(w)
		observed[ot] = txCount
		txCount++
	}

//保存所有正确的箱子。
	for _, list := range ef.bin {

		binary.Write(w, binary.BigEndian, uint32(len(list)))

		for _, o := range list {
			binary.Write(w, binary.BigEndian, observed[o])
		}
	}

//丢弃的事务。
	binary.Write(w, binary.BigEndian, uint32(len(ef.dropped)))
	for _, registered := range ef.dropped {
		registered.serialize(w, observed)
	}

//提交发送并返回。
	return FeeEstimatorState(w.Bytes())
}

//RestorefeeEstimator采用以前的feeeEstimatorState
//通过保存返回并恢复到feeestimator
func RestoreFeeEstimator(data FeeEstimatorState) (*FeeEstimator, error) {
	r := bytes.NewReader([]byte(data))

//检查版本
	var version uint32
	err := binary.Read(r, binary.BigEndian, &version)
	if err != nil {
		return nil, err
	}
	if version != estimateFeeSaveVersion {
		return nil, fmt.Errorf("Incorrect version: expected %d found %d", estimateFeeSaveVersion, version)
	}

	ef := &FeeEstimator{
		observed: make(map[chainhash.Hash]*observedTransaction),
	}

//读取基本参数。
	binary.Read(r, binary.BigEndian, &ef.maxRollback)
	binary.Read(r, binary.BigEndian, &ef.binSize)
	binary.Read(r, binary.BigEndian, &ef.maxReplacements)
	binary.Read(r, binary.BigEndian, &ef.minRegisteredBlocks)
	binary.Read(r, binary.BigEndian, &ef.lastKnownHeight)
	binary.Read(r, binary.BigEndian, &ef.numBlocksRegistered)

//读取事务。
	var numObserved uint32
	observed := make(map[uint32]*observedTransaction)
	binary.Read(r, binary.BigEndian, &numObserved)
	for i := uint32(0); i < numObserved; i++ {
		ot, err := deserializeObservedTransaction(r)
		if err != nil {
			return nil, err
		}
		observed[i] = ot
		ef.observed[ot.hash] = ot
	}

//读取容器。
	for i := 0; i < estimateFeeDepth; i++ {
		var numTransactions uint32
		binary.Read(r, binary.BigEndian, &numTransactions)
		bin := make([]*observedTransaction, numTransactions)
		for j := uint32(0); j < numTransactions; j++ {
			var index uint32
			binary.Read(r, binary.BigEndian, &index)

			var exists bool
			bin[j], exists = observed[index]
			if !exists {
				return nil, fmt.Errorf("Invalid transaction reference %d", index)
			}
		}
		ef.bin[i] = bin
	}

//读取丢弃的事务。
	var numDropped uint32
	binary.Read(r, binary.BigEndian, &numDropped)
	ef.dropped = make([]*registeredBlock, numDropped)
	for i := uint32(0); i < numDropped; i++ {
		var err error
		ef.dropped[int(i)], err = deserializeRegisteredBlock(r, observed)
		if err != nil {
			return nil, err
		}
	}

	return ef, nil
}
