
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package blockchain

import (
	"fmt"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

//txoflags是一个位掩码，用于定义
//utxo视图中的事务输出。
type txoFlags uint8

const (
//tfcoinbase表示一个txout包含在一个coinbase tx中。
	tfCoinBase txoFlags = 1 << iota

//TfsPent表示TxOut已用完。
	tfSpent

//TfModified表示TxOut自
//加载。
	tfModified
)

//utxoEntry包含有关utxo中单个事务输出的详细信息
//视图，例如它是否包含在CoinBase Tx中，高度
//包含Tx的块，不管它是否被使用，它的公钥
//脚本，以及它的报酬。
type UtxoEntry struct {
//注意：添加、删除或修改
//不应考虑更改此结构中的定义
//它如何影响64位平台上的对齐。当前订单是
//专门设计以使填充物最小化。将会有一个
//其中很多都在内存中，所以加上一些额外的填充字节。

	amount      int64
pkScript    []byte //输出的公钥脚本。
blockHeight int32  //包含Tx的块的高度。

//packedFlags包含有关输出的其他信息，例如
//
//因为它是装上去的。使用这种方法是为了减少内存
//因为在内存中会有很多这样的用法。
	packedFlags txoFlags
}

//ismodified返回输出自
//加载。
func (entry *UtxoEntry) isModified() bool {
	return entry.packedFlags&tfModified == tfModified
}

//iscoinbase返回输出是否包含在coinbase中
//交易。
func (entry *UtxoEntry) IsCoinBase() bool {
	return entry.packedFlags&tfCoinBase == tfCoinBase
}

//block height返回包含输出的块的高度。
func (entry *UtxoEntry) BlockHeight() int32 {
	return entry.blockHeight
}

//IsSpent返回输出是否基于
//从中获取未占用事务输出视图的当前状态。
func (entry *UtxoEntry) IsSpent() bool {
	return entry.packedFlags&tfSpent == tfSpent
}

//Spend将输出标记为已用。花费已经花费的产出
//没有效果。
func (entry *UtxoEntry) Spend() {
//如果输出已用完，则不执行任何操作。
	if entry.IsSpent() {
		return
	}

//将输出标记为已用和已修改。
	entry.packedFlags |= tfSpent | tfModified
}

//amount返回输出量。
func (entry *UtxoEntry) Amount() int64 {
	return entry.amount
}

//pkscript返回输出的公钥脚本。
func (entry *UtxoEntry) PkScript() []byte {
	return entry.pkScript
}

//clone返回utxo项的浅副本。
func (entry *UtxoEntry) Clone() *UtxoEntry {
	if entry == nil {
		return nil
	}

	return &UtxoEntry{
		amount:      entry.amount,
		pkScript:    entry.pkScript,
		blockHeight: entry.blockHeight,
		packedFlags: entry.packedFlags,
	}
}

//utxoview表示未暂停事务输出集合中的视图
//
//主链的末端，主链历史上的某个点，或
//沿着侧链。
//
//其他事务需要未消耗的输出，例如
//脚本验证和双倍开销预防。
type UtxoViewpoint struct {
	entries  map[wire.OutPoint]*UtxoEntry
	bestHash chainhash.Hash
}

//BestHash返回视图当前链中最佳块的哈希
//礼物。
func (view *UtxoViewpoint) BestHash() *chainhash.Hash {
	return &view.bestHash
}

//setBestHash设置当前视图链中最佳块的哈希
//礼物。
func (view *UtxoViewpoint) SetBestHash(hash *chainhash.Hash) {
	view.bestHash = *hash
}

//LookupEntry根据
//视图的当前状态。如果传递的输出为零，则返回零
//
//
func (view *UtxoViewpoint) LookupEntry(outpoint wire.OutPoint) *UtxoEntry {
	return view.entries[outpoint]
}

//如果无法证实，则addtxout将指定的输出添加到视图中
//不可费解的当视图已经有一个输出条目时，它将
//标记为未使用。所有字段都将针对现有条目进行更新，因为
//可能在REORG期间发生了变化。
func (view *UtxoViewpoint) addTxOut(outpoint wire.OutPoint, txOut *wire.TxOut, isCoinBase bool, blockHeight int32) {
//不要添加可证明不可靠的输出。
	if txscript.IsUnspendable(txOut.PkScript) {
		return
	}

//更新现有条目。所有字段都会更新，因为
//
//被具有相同哈希的不同事务替换。这个
//只要上一个事务已完全用完，就允许。
	entry := view.LookupEntry(outpoint)
	if entry == nil {
		entry = new(UtxoEntry)
		view.entries[outpoint] = entry
	}

	entry.amount = txOut.Value
	entry.pkScript = txOut.PkScript
	entry.blockHeight = blockHeight
	entry.packedFlags = tfModified
	if isCoinBase {
		entry.packedFlags |= tfCoinBase
	}
}

//addtxout将传递的事务的指定输出添加到视图if
//它是存在的，不能证明是不可依赖的。当视图已经具有
//输入输出，它将被标记为未释放。所有字段都将更新
//对于现有条目，因为它可能在REORG期间发生了更改。
func (view *UtxoViewpoint) AddTxOut(tx *btcutil.Tx, txOutIdx uint32, blockHeight int32) {
//无法为越界索引添加输出。
	if txOutIdx >= uint32(len(tx.MsgTx().TxOut)) {
		return
	}

//更新现有条目。所有字段都会更新，因为
//现有入口可能（尽管极不可能）
//被具有相同哈希的不同事务替换。这个
//只要上一个事务已完全用完，就允许。
	prevOut := wire.OutPoint{Hash: *tx.Hash(), Index: txOutIdx}
	txOut := tx.MsgTx().TxOut[txOutIdx]
	view.addTxOut(prevOut, txOut, IsCoinBase(tx), blockHeight)
}

//addtxouts在传递的事务中添加所有不能证明的输出
//不依赖于视图。当视图已有任何
//输出，它们只是标记为未使用。将为更新所有字段
//现有条目，因为它可能在REORG期间发生了更改。
func (view *UtxoViewpoint) AddTxOuts(tx *btcutil.Tx, blockHeight int32) {
//循环所有事务输出，并添加那些不是
//可以证明是不可靠的。
	isCoinBase := IsCoinBase(tx)
	prevOut := wire.OutPoint{Hash: *tx.Hash()}
	for txOutIdx, txOut := range tx.MsgTx().TxOut {
//更新现有条目。所有字段都会更新，因为
//可能（尽管极不可能）现有的
//
//相同的散列。这是允许的，只要上一个
//事务已完全用完。
		prevOut.Index = uint32(txOutIdx)
		view.addTxOut(prevOut, txOut, isCoinBase, blockHeight)
	}
}

//ConnectTransaction通过添加由
//已传递事务并将事务所花费的所有utxo标记为
//花了。此外，当“stxos”参数不是nil时，它将被更新。
//为每个花费的txout附加一个条目。如果
//
func (view *UtxoViewpoint) connectTransaction(tx *btcutil.Tx, blockHeight int32, stxos *[]SpentTxOut) error {
//CoinBase交易没有任何要花费的输入。
	if IsCoinBase(tx) {
//将事务的输出添加为可用的utxos。
		view.AddTxOuts(tx, blockHeight)
		return nil
	}

//通过在视图中标记引用的utxo来使用它们，
//如果为用过的txout详细信息提供了一个切片，请附加一个条目。
//对它。
	for _, txIn := range tx.MsgTx().TxIn {
//确保视图中存在引用的utxo。这应该
//除非代码中引入了错误，否则永远不会发生。
		entry := view.entries[txIn.PreviousOutPoint]
		if entry == nil {
			return AssertError(fmt.Sprintf("view missing input %v",
				txIn.PreviousOutPoint))
		}

//仅在需要时创建STXO详细信息。
		if stxos != nil {
//使用utxo条目填充stxo详细信息。
			var stxo = SpentTxOut{
				Amount:     entry.Amount(),
				PkScript:   entry.PkScript(),
				Height:     entry.BlockHeight(),
				IsCoinBase: entry.IsCoinBase(),
			}
			*stxos = append(*stxos, stxo)
		}

//将条目标记为已用。这要等到
//相关的细节已经被访问，因为它可能
//将来清除内存中的字段。
		entry.Spend()
	}

//将事务的输出添加为可用的utxos。
	view.AddTxOuts(tx, blockHeight)
	return nil
}

//ConnectTransactions通过添加所有由
//在传递的块中的事务中，将所有utxo标记为事务
//花费为已用，并将视图的最佳哈希设置为已传递的块。
//此外，当“stxos”参数不是nil时，它将更新为
//为每个花费的txout附加一个条目。
func (view *UtxoViewpoint) connectTransactions(block *btcutil.Block, stxos *[]SpentTxOut) error {
	for _, tx := range block.Transactions() {
		err := view.connectTransaction(tx, block.Height(), stxos)
		if err != nil {
			return err
		}
	}

//更新视图的最佳哈希以包含此块，因为
//
	view.SetBestHash(block.Hash())
	return nil
}

//fetchentrybash尝试通过以下方式查找给定哈希的任何可用的utxo
//搜索给定哈希的整个可能输出集。它检查
//如果需要，视图首先返回数据库。
func (view *UtxoViewpoint) fetchEntryByHash(db database.DB, hash *chainhash.Hash) (*UtxoEntry, error) {
//首先尝试在视图中查找具有所提供哈希的utxo。
	prevOut := wire.OutPoint{Hash: *hash}
	for idx := uint32(0); idx < MaxOutputsPerBlock; idx++ {
		prevOut.Index = idx
		entry := view.LookupEntry(prevOut)
		if entry != nil {
			return entry, nil
		}
	}

//检查数据库，因为它在视图中不存在。本遗嘱
//通常情况下，因为只加载特定引用的utxo
//进入视野。
	var entry *UtxoEntry
	err := db.View(func(dbTx database.Tx) error {
		var err error
		entry, err = dbFetchUtxoEntryByHash(dbTx, hash)
		return err
	})
	return entry, err
}

//DisconnectTransactions通过删除所有事务来更新视图
//由传递的块创建，恢复由
//使用提供的已用txo信息，并为
//查看传递的块之前的块。
func (view *UtxoViewpoint) disconnectTransactions(db database.DB, block *btcutil.Block, stxos []SpentTxOut) error {
//请检查是否提供了正确数量的STXO。
	if len(stxos) != countSpentOutputs(block) {
		return AssertError("disconnectTransactions called with bad " +
			"spent transaction out information")
	}

//在所有事务中向后循环，使所有事务都不被占用
//颠倒顺序。这是必需的，因为事务在块的后面
//可以从以前的消费。
	stxoIdx := len(stxos) - 1
	transactions := block.Transactions()
	for txIdx := len(transactions) - 1; txIdx > -1; txIdx-- {
		tx := transactions[txIdx]

//所有条目都可能需要标记为CoinBase。
		var packedFlags txoFlags
		isCoinBase := txIdx == 0
		if isCoinBase {
			packedFlags |= tfCoinBase
		}

//标记最初由
//已用事务。值得注意的是，
//输出实际上不是花在这里，而是没有
//由于使用了修剪后的utxo集，因此不再存在
//不存在的utxo与
//一个已经花掉的。
//
//当视图中不存在utxo时，添加一个
//输入它，然后标记它已用完。这样做是因为
//代码依赖于它在视图中的存在，以便
//信号发生了变化。
		txHash := tx.Hash()
		prevOut := wire.OutPoint{Hash: *txHash}
		for txOutIdx, txOut := range tx.MsgTx().TxOut {
			if txscript.IsUnspendable(txOut.PkScript) {
				continue
			}

			prevOut.Index = uint32(txOutIdx)
			entry := view.entries[prevOut]
			if entry == nil {
				entry = &UtxoEntry{
					amount:      txOut.Value,
					pkScript:    txOut.PkScript,
					blockHeight: block.Height(),
					packedFlags: packedFlags,
				}

				view.entries[prevOut] = entry
			}

			entry.Spend()
		}

//在所有事务输入中向后循环（除了
//对于没有输入的coinbase）和unspend
//引用的txos。这是必要的，以符合
//已用txout项。
		if isCoinBase {
			continue
		}
		for txInIdx := len(tx.MsgTx().TxIn) - 1; txInIdx > -1; txInIdx-- {
//确保已用txout索引已递减以保持
//与事务输入同步。
			stxo := &stxos[stxoIdx]
			stxoIdx--

//当没有引用的条目时
//在视图中输出，这意味着它以前被使用过，
//所以创建一个新的utxo条目来恢复它。
			originOut := &tx.MsgTx().TxIn[txInIdx].PreviousOutPoint
			entry := view.entries[*originOut]
			if entry == nil {
				entry = new(UtxoEntry)
				view.entries[*originOut] = entry
			}

//Legacy v1 Spend Journal格式只存储了
//当输出是最后一个时的coinbase标志和高度
//未占用的事务输出。因此，当
//信息丢失，请通过扫描进行搜索
//事务的所有可能输出，因为它必须
//加入其中一个。
//
//应该注意的是，这是非常低效的，
//但事实上，它几乎永远不会运行
//新条目包括所有输出的信息
//因此，唯一的方法就是
//足够多的REORG发生这样一个块
//支出数据正在断开连接。概率
//在实践中，这是非常低的开始和
//变得越来越小
//有联系的。如果一个新的数据库
//仅使用新的v2格式运行，此代码路径
//永远不会跑。
			if stxo.Height == 0 {
				utxo, err := view.fetchEntryByHash(db, txHash)
				if err != nil {
					return err
				}
				if utxo == nil {
					return AssertError(fmt.Sprintf("unable "+
						"to resurrect legacy stxo %v",
						*originOut))
				}

				stxo.Height = utxo.BlockHeight()
				stxo.IsCoinBase = utxo.IsCoinBase()
			}

//使用支出中的stxo数据恢复utxo
//日记并标记为已修改。
			entry.amount = stxo.Amount
			entry.pkScript = stxo.PkScript
			entry.blockHeight = stxo.Height
			entry.packedFlags = tfModified
			if stxo.IsCoinBase {
				entry.packedFlags |= tfCoinBase
			}
		}
	}

//将视图的最佳哈希更新到上一个块，因为
//当前块的事务已断开连接。
	view.SetBestHash(&block.MsgBlock().Header.PrevBlock)
	return nil
}

//removeentry从的当前状态中移除给定的事务输出
//风景。如果传递的输出不存在于
//查看。
func (view *UtxoViewpoint) RemoveEntry(outpoint wire.OutPoint) {
	delete(view.entries, outpoint)
}

//entries返回存储所有utxo项的基础映射。
func (view *UtxoViewpoint) Entries() map[wire.OutPoint]*UtxoEntry {
	return view.entries
}

//
//所有条目未修改。
func (view *UtxoViewpoint) commit() {
	for outpoint, entry := range view.entries {
		if entry == nil || (entry.isModified() && entry.IsSpent()) {
			delete(view.entries, outpoint)
			continue
		}

		entry.packedFlags ^= tfModified
	}
}

//
//从主链末端的角度看
//通话时间。
//
//完成此功能后，视图将包含每个
//请求的输出点。用过的输出，或者那些不存在的输出，
//将在视图中生成一个零条目。
func (view *UtxoViewpoint) fetchUtxosMain(db database.DB, outpoints map[wire.OutPoint]struct{}) error {
//如果没有请求的输出，则不执行任何操作。
	if len(outpoints) == 0 {
		return nil
	}

//从点加载请求的未暂停事务输出集
//主链末端的视图。
//
//
//将导致视图中没有条目。这是故意的
//因此其他代码可以使用商店中存在的条目作为一种方法
//不必要地避免尝试从数据库重新加载它。
	return db.View(func(dbTx database.Tx) error {
		for outpoint := range outpoints {
			entry, err := dbFetchUtxoEntry(dbTx, outpoint)
			if err != nil {
				return err
			}

			view.entries[outpoint] = entry
		}

		return nil
	})
}

//fetchutxos为提供的一组
//根据需要从数据库输出到视图中，除非它们已经存在
//在这种情况下，它们将被忽略。
func (view *UtxoViewpoint) fetchUtxos(db database.DB, outpoints map[wire.OutPoint]struct{}) error {
//如果没有请求的输出，则不执行任何操作。
	if len(outpoints) == 0 {
		return nil
	}

//筛选视图中已有的条目。
	neededSet := make(map[wire.OutPoint]struct{})
	for outpoint := range outpoints {
//已加载到当前视图中。
		if _, ok := view.entries[outpoint]; ok {
			continue
		}

		neededSet[outpoint] = struct{}{}
	}

//从数据库请求输入utxos。
	return view.fetchUtxosMain(db, neededSet)
}

//FetchInputXOS为输入加载未暂停的事务输出
//由给定块中的事务引用到视图中
//根据需要提供数据库。特别是，在
//将块添加到视图中，视图中已有的条目为
//未修改。
func (view *UtxoViewpoint) fetchInputUtxos(db database.DB, block *btcutil.Block) error {
//
//此块可能引用了在此之前的其他事务
//链中还没有的块。
	txInFlight := map[chainhash.Hash]int{}
	transactions := block.Transactions()
	for i, tx := range transactions {
		txInFlight[*tx.Hash()] = i
	}

//
//没有输入）将它们收集到需要的集合中
//已经知道的（飞行中）。
	neededSet := make(map[wire.OutPoint]struct{})
	for i, tx := range transactions[1:] {
		for _, txIn := range tx.MsgTx().TxIn {
//交易输入可供参考。
//仅此块中另一个事务的输出
//如果引用的事务在
//当前块中的一个。添加的输出
//引用事务作为可用的utxos
//情况就是这样。否则，utxo的细节仍然是
//需要。
//
//注意：这里的>=是正确的，因为我少了一个
//比交易的实际位置
//由于跳过coinbase而导致的块。
			originHash := &txIn.PreviousOutPoint.Hash
			if inFlightIndex, ok := txInFlight[*originHash]; ok &&
				i >= inFlightIndex {

				originTx := transactions[inFlightIndex]
				view.AddTxOuts(originTx, block.Height())
				continue
			}

//不要请求已经在视图中的条目
//从数据库中。
			if _, ok := view.entries[txIn.PreviousOutPoint]; ok {
				continue
			}

			neededSet[txIn.PreviousOutPoint] = struct{}{}
		}
	}

//从数据库请求输入utxos。
	return view.fetchUtxosMain(db, neededSet)
}

//newutxovidence返回一个新的空的未暂停事务输出视图。
func NewUtxoViewpoint() *UtxoViewpoint {
	return &UtxoViewpoint{
		entries: make(map[wire.OutPoint]*UtxoEntry),
	}
}

//fetchutxoview为引用的输入加载未暂停的事务输出
//从主链末端的角度来看，传递的事务。
//它还尝试为事务本身的输出获取utxos。
//因此，可以检查返回的视图是否有重复的事务。
//
//此函数对于并发访问是安全的，但是返回的视图不是。
func (b *BlockChain) FetchUtxoView(tx *btcutil.Tx) (*UtxoViewpoint, error) {
//根据所引用的输出创建一组所需的输出
//传递的事务的输入和事务的输出
//本身。
	neededSet := make(map[wire.OutPoint]struct{})
	prevOut := wire.OutPoint{Hash: *tx.Hash()}
	for txOutIdx := range tx.MsgTx().TxOut {
		prevOut.Index = uint32(txOutIdx)
		neededSet[prevOut] = struct{}{}
	}
	if !IsCoinBase(tx) {
		for _, txIn := range tx.MsgTx().TxIn {
			neededSet[txIn.PreviousOutPoint] = struct{}{}
		}
	}

//从主文件结尾的角度请求utxos
//链。
	view := NewUtxoViewpoint()
	b.chainLock.RLock()
	err := view.fetchUtxosMain(b.db, neededSet)
	b.chainLock.RUnlock()
	return view, err
}

//fetchutxoEntry加载并返回请求的未暂停事务输出
//从主链末端看。
//
//注意：请求没有数据的输出不会返回
//错误。相反，条目和错误都将为零。这样做是为了
//允许修剪已用事务输出。实际上，这意味着
//调用方必须在调用其方法之前检查返回的条目是否为零。
//
//此函数对于并发访问是安全的，但是返回的条目（如果
//不是）。
func (b *BlockChain) FetchUtxoEntry(outpoint wire.OutPoint) (*UtxoEntry, error) {
	b.chainLock.RLock()
	defer b.chainLock.RUnlock()

	var entry *UtxoEntry
	err := b.db.View(func(dbTx database.Tx) error {
		var err error
		entry, err = dbFetchUtxoEntry(dbTx, outpoint)
		return err
	})
	if err != nil {
		return nil, err
	}

	return entry, nil
}
