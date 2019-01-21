
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2016-2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package rpctest

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/hdkeychain"
)

var (
//hdseed是memwallet用来初始化其
//HD根密钥。为了确保
//跨测试运行的确定性行为。
	hdSeed = [chainhash.HashSize]byte{
		0x79, 0xa6, 0x1a, 0xdb, 0xc6, 0xe5, 0xa2, 0xe1,
		0x39, 0xd2, 0x71, 0x3a, 0x54, 0x6e, 0xc7, 0xc8,
		0x75, 0x63, 0x2e, 0x75, 0xf1, 0xdf, 0x9c, 0x3f,
		0xa6, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
)

//utxo代表Memwallet可消费的未消耗输出。成熟度
//记录事务的高度，以便正确地观察
//直接coinbase输出的到期期。
type utxo struct {
	pkScript       []byte
	value          btcutil.Amount
	keyIndex       uint32
	maturityHeight int32
	isLocked       bool
}

//如果目标utxo在
//通过的块高度。否则，返回false。
func (u *utxo) isMature(height int32) bool {
	return height >= u.maturityHeight
}

//chain update封装对当前主链的更新。这个结构是
//用于在每次新数据块连接到主数据块时同步MemWallet
//链。
type chainUpdate struct {
	blockHeight  int32
	filteredTxns []*btcutil.Tx
isConnect    bool //如果连接为真，如果断开为假
}

//UndoEntry在功能上与ChainUpdate相反。撤消是
//为接收到的每个新块创建，然后存储在日志中，以便
//正确处理块重新组织。
type undoEntry struct {
	utxosDestroyed map[wire.OutPoint]*utxo
	utxosCreated   []wire.OutPoint
}

//Memwallet是一个简单的内存钱包，其目的是提供基本的
//安全带的钱包功能。钱包使用硬编码的高清钥匙
//促进线束测试运行之间再现性的层次结构。
type memWallet struct {
	coinbaseKey  *btcec.PrivateKey
	coinbaseAddr btcutil.Address

//hdroot是钱包的根主私钥。
	hdRoot *hdkeychain.ExtendedKey

//hindex是下一个可用的键索引偏移量。
	hdIndex uint32

//当前高度是已知要同步钱包的最新高度
//去。
	currentHeight int32

//addrs跟踪属于钱包的所有地址。地址
//通过其来自hdroot的键路径进行索引。
	addrs map[uint32]btcutil.Address

//utxos是钱包里可以消费的一套utxos。
	utxos map[wire.OutPoint]*utxo

//reorgjournal是一个映射，它为每个新块存储一个撤消条目。
//收到。一旦断开一个块，则
//对特定高度进行评估，从而重新缠绕
//断开了钱包上的一组可消费的utxos。
	reorgJournal map[int32]*undoEntry

	chainUpdates      []*chainUpdate
	chainUpdateSignal chan struct{}
	chainMtx          sync.Mutex

	net *chaincfg.Params

	rpc *rpcclient.Client

	sync.RWMutex
}

//newmemwallet创建并返回
//Memwallet给出了特定区块链的参数。
func newMemWallet(net *chaincfg.Params, harnessID uint32) (*memWallet, error) {
//钱包的最终高清种子是：HDSEED Harnesid。这种方法
//确保每个线束实例使用确定性根种子
//基于其线束ID。
	var harnessHDSeed [chainhash.HashSize + 4]byte
	copy(harnessHDSeed[:], hdSeed[:])
	binary.BigEndian.PutUint32(harnessHDSeed[:chainhash.HashSize], harnessID)

	hdRoot, err := hdkeychain.NewMaster(harnessHDSeed[:], net)
	if err != nil {
		return nil, nil
	}

//hd根中的第一个子键保留为coinbase
//生成地址。
	coinbaseChild, err := hdRoot.Child(0)
	if err != nil {
		return nil, err
	}
	coinbaseKey, err := coinbaseChild.ECPrivKey()
	if err != nil {
		return nil, err
	}
	coinbaseAddr, err := keyToAddr(coinbaseKey, net)
	if err != nil {
		return nil, err
	}

//跟踪CoinBase生成地址以确保我们正确跟踪
//新生成的比特币我们可以消费。
	addrs := make(map[uint32]btcutil.Address)
	addrs[0] = coinbaseAddr

	return &memWallet{
		net:               net,
		coinbaseKey:       coinbaseKey,
		coinbaseAddr:      coinbaseAddr,
		hdIndex:           1,
		hdRoot:            hdRoot,
		addrs:             addrs,
		utxos:             make(map[wire.OutPoint]*utxo),
		chainUpdateSignal: make(chan struct{}),
		reorgJournal:      make(map[int32]*undoEntry),
	}, nil
}

//启动启动钱包正常工作所需的所有Goroutines。
func (m *memWallet) Start() {
	go m.chainSyncer()
}

//SyncedHeight返回钱包已知要同步到的高度。
//
//此函数对于并发访问是安全的。
func (m *memWallet) SyncedHeight() int32 {
	m.RLock()
	defer m.RUnlock()
	return m.currentHeight
}

//setrpcclient将传递到BTCD的RPC连接保存为钱包的
//个人RPC连接。
func (m *memWallet) SetRPCClient(rpcClient *rpcclient.Client) {
	m.rpc = rpcClient
}

//InfectBlock是一个回调，每次新的块
//与主链相连。它为链同步器的更新排队，
//按顺序调用私有版本。
func (m *memWallet) IngestBlock(height int32, header *wire.BlockHeader, filteredTxns []*btcutil.Tx) {
//将此新的链更新附加到新链队列的末尾
//更新。
	m.chainMtx.Lock()
	m.chainUpdates = append(m.chainUpdates, &chainUpdate{height,
		filteredTxns, true})
	m.chainMtx.Unlock()

//启动Goroutine以向链式同步器发出新更新的信号
//可用。我们在新的Goroutine中这样做是为了避免阻塞
//RPC客户端的主循环。
	go func() {
		m.chainUpdateSignal <- struct{}{}
	}()
}

//InfectBlock根据输出更新钱包的内部utxo状态
//在每个块中创建和销毁。
func (m *memWallet) ingestBlock(update *chainUpdate) {
//更新最新的同步高度，然后处理每个筛选的高度
//块中的事务，在其中创建和销毁utxos
//结果就是钱包。
	m.currentHeight = update.blockHeight
	undo := &undoEntry{
		utxosDestroyed: make(map[wire.OutPoint]*utxo),
	}
	for _, tx := range update.filteredTxns {
		mtx := tx.MsgTx()
		isCoinbase := blockchain.IsCoinBaseTx(mtx)
		txHash := mtx.TxHash()
		m.evalOutputs(mtx.TxOut, &txHash, isCoinbase, undo)
		m.evalInputs(mtx.TxIn, undo)
	}

//最后，记录此块的撤消项以便
//正确更新内部状态以响应块
//从主链中重新组织。
	m.reorgJournal[update.blockHeight] = undo
}

//ChainSyncer是一个Goroutine，专门用于处理新块，以便
//保持钱包的utxo状态最新。
//
//注意：这必须作为goroutine运行。
func (m *memWallet) chainSyncer() {
	var update *chainUpdate

	for range m.chainUpdateSignal {
//有新的更新可用，因此从中弹出新的链更新
//更新队列的前面。
		m.chainMtx.Lock()
		update = m.chainUpdates[0]
m.chainUpdates[0] = nil //设置为零以防止GC泄漏。
		m.chainUpdates = m.chainUpdates[1:]
		m.chainMtx.Unlock()

		m.Lock()
		if update.isConnect {
			m.ingestBlock(update)
		} else {
			m.unwindBlock(update)
		}
		m.Unlock()
	}
}

//evaluotputs评估每个传递的输出，创建一个新的匹配
//如果我们能花掉输出，那么utxo就在钱包里。
func (m *memWallet) evalOutputs(outputs []*wire.TxOut, txHash *chainhash.Hash,
	isCoinbase bool, undo *undoEntry) {

	for i, output := range outputs {
		pkScript := output.PkScript

//扫描我们当前控制的所有地址，查看
//产出正在为我们付出代价。
		for keyIndex, addr := range m.addrs {
			pkHash := addr.ScriptAddress()
			if !bytes.Contains(pkScript, pkHash) {
				continue
			}

//如果这是一个coinbase输出，那么我们将标记
//在适当的块高度成熟的高度
//未来。
			var maturityHeight int32
			if isCoinbase {
				maturityHeight = m.currentHeight + int32(m.net.CoinbaseMaturity)
			}

			op := wire.OutPoint{Hash: *txHash, Index: uint32(i)}
			m.utxos[op] = &utxo{
				value:          btcutil.Amount(output.Value),
				keyIndex:       keyIndex,
				maturityHeight: maturityHeight,
				pkScript:       pkScript,
			}
			undo.utxosCreated = append(undo.utxosCreated, op)
		}
	}
}

//evalinputs扫描所有传递的输入，销毁
//通过输入消费的钱包。
func (m *memWallet) evalInputs(inputs []*wire.TxIn, undo *undoEntry) {
	for _, txIn := range inputs {
		op := txIn.PreviousOutPoint
		oldUtxo, ok := m.utxos[op]
		if !ok {
			continue
		}

		undo.utxosDestroyed[op] = oldUtxo
		delete(m.utxos, op)
	}
}

//UnwindBlock是一个回调，每次一个块
//从主链上断开。它为链同步器的更新排队，
//按顺序调用私有版本。
func (m *memWallet) UnwindBlock(height int32, header *wire.BlockHeader) {
//将此新的链更新附加到新链队列的末尾
//更新。
	m.chainMtx.Lock()
	m.chainUpdates = append(m.chainUpdates, &chainUpdate{height,
		nil, false})
	m.chainMtx.Unlock()

//启动Goroutine以向链式同步器发出新更新的信号
//可用。我们在新的Goroutine中这样做是为了避免阻塞
//RPC客户端的主循环。
	go func() {
		m.chainUpdateSignal <- struct{}{}
	}()
}

//UnwindBlock撤消特定块对钱包的影响
//内部utxo状态。
func (m *memWallet) unwindBlock(update *chainUpdate) {
	undo := m.reorgJournal[update.blockHeight]

	for _, utxo := range undo.utxosCreated {
		delete(m.utxos, utxo)
	}

	for outPoint, utxo := range undo.utxosDestroyed {
		m.utxos[outPoint] = utxo
	}

	delete(m.reorgJournal, update.blockHeight)
}

//new address从钱包的HD密钥链返回新地址。它也
//将地址加载到RPC客户端的事务筛选器中，以确保
//涉及它的事务通过通知传递。
func (m *memWallet) newAddress() (btcutil.Address, error) {
	index := m.hdIndex

	childKey, err := m.hdRoot.Child(index)
	if err != nil {
		return nil, err
	}
	privKey, err := childKey.ECPrivKey()
	if err != nil {
		return nil, err
	}

	addr, err := keyToAddr(privKey, m.net)
	if err != nil {
		return nil, err
	}

	err = m.rpc.LoadTxFilter(false, []btcutil.Address{addr}, nil)
	if err != nil {
		return nil, err
	}

	m.addrs[index] = addr

	m.hdIndex++

	return addr, nil
}

//newaddress返回一个新地址，可通过钱包消费。
//
//此函数对于并发访问是安全的。
func (m *memWallet) NewAddress() (btcutil.Address, error) {
	m.Lock()
	defer m.Unlock()

	return m.newAddress()
}

//Fundtx试图为发送amt比特币的交易提供资金。硬币是
//选择这样最终花费的金额支付足够的费用由
//通过的费率。通过的费率应以
//每字节饱和。正在资助的交易可以选择包括
//更改由更改布尔值指示的输出。
//
//注意：调用此函数时，必须保持memwallet的互斥。
func (m *memWallet) fundTx(tx *wire.MsgTx, amt btcutil.Amount,
	feeRate btcutil.Amount, change bool) error {

	const (
//SpendSize是sigscript的最大字节数
//其中p2pkh输出：op_data_73<sig>op_data_33<pubkey>
		spendSize = 1 + 73 + 1 + 33
	)

	var (
		amtSelected btcutil.Amount
		txSize      int
	)

	for outPoint, utxo := range m.utxos {
//跳过当前尚未成熟或
//当前已锁定。
		if !utxo.isMature(m.currentHeight) || utxo.isLocked {
			continue
		}

		amtSelected += utxo.value

//将所选输出添加到事务，更新
//当前Tx大小，同时考虑未来的大小
//SigScript。
		tx.AddTxIn(wire.NewTxIn(&outPoint, nil, nil))
		txSize = tx.SerializeSize() + spendSize*len(tx.TxIn)

//计算此时TXN所需的费用
//遵守规定的费率。如果我们没有足够的
//从他选择的当前数额的硬币支付费用，然后
//继续抓取更多的硬币。
		reqFee := btcutil.Amount(txSize * int(feeRate))
		if amtSelected-reqFee < amt {
			continue
		}

//如果我们还有任何变化，我们应该创造一个变化
//输出，然后向事务添加附加输出
//为它保留。
		changeVal := amtSelected - amt - reqFee
		if changeVal > 0 && change {
			addr, err := m.newAddress()
			if err != nil {
				return err
			}
			pkScript, err := txscript.PayToAddrScript(addr)
			if err != nil {
				return err
			}
			changeOutput := &wire.TxOut{
				Value:    int64(changeVal),
				PkScript: pkScript,
			}
			tx.AddTxOut(changeOutput)
		}

		return nil
	}

//如果我们达到了这一点，那么硬币选择就失败了，因为
//硬币数量不足。
	return fmt.Errorf("not enough funds for coin selection")
}

//sendOutputs创建一个事务，然后将其发送到指定的输出
//同时观察通过的费率。应说明通过的费率
//以每字节的Satoshis为单位。
func (m *memWallet) SendOutputs(outputs []*wire.TxOut,
	feeRate btcutil.Amount) (*chainhash.Hash, error) {

	tx, err := m.CreateTransaction(outputs, feeRate, true)
	if err != nil {
		return nil, err
	}

	return m.rpc.SendRawTransaction(tx, true)
}

//sendOutputswithoutchange创建并发送一个向
//观察通过的费率并忽略更改时的指定输出
//输出。通过的费率应以SAT/B表示。
func (m *memWallet) SendOutputsWithoutChange(outputs []*wire.TxOut,
	feeRate btcutil.Amount) (*chainhash.Hash, error) {

	tx, err := m.CreateTransaction(outputs, feeRate, false)
	if err != nil {
		return nil, err
	}

	return m.rpc.SendRawTransaction(tx, true)
}

//CreateTransaction返回向指定的
//在观察所需费率的同时输出。通过的费率应该是
//以每字节的Satoshis表示。正在创建的事务可以选择
//包括由更改布尔值指示的更改输出。
//
//此函数对于并发访问是安全的。
func (m *memWallet) CreateTransaction(outputs []*wire.TxOut,
	feeRate btcutil.Amount, change bool) (*wire.MsgTx, error) {

	m.Lock()
	defer m.Unlock()

	tx := wire.NewMsgTx(wire.TxVersion)

//计算要发送的总金额以进行投币
//选择就在下面。
	var outputAmt btcutil.Amount
	for _, output := range outputs {
		outputAmt += btcutil.Amount(output.Value)
		tx.AddTxOut(output)
	}

//尝试使用可消费的utxos为交易提供资金。
	if err := m.fundTx(tx, outputAmt, feeRate, change); err != nil {
		return nil, err
	}

//用有效的sigscript填充所有选定的输入以进行开销。
//一路上记录所有的输出，以避免
//潜在的双重消费。
	spentOutputs := make([]*utxo, 0, len(tx.TxIn))
	for i, txIn := range tx.TxIn {
		outPoint := txIn.PreviousOutPoint
		utxo := m.utxos[outPoint]

		extendedKey, err := m.hdRoot.Child(utxo.keyIndex)
		if err != nil {
			return nil, err
		}

		privKey, err := extendedKey.ECPrivKey()
		if err != nil {
			return nil, err
		}

		sigScript, err := txscript.SignatureScript(tx, i, utxo.pkScript,
			txscript.SigHashAll, privKey, true)
		if err != nil {
			return nil, err
		}

		txIn.SignatureScript = sigScript

		spentOutputs = append(spentOutputs, utxo)
	}

//因为这些输出现在正被这个新创建的
//事务处理，将输出标记为“锁定”。此操作确保
//这些输出不会被任何后续事务花费两倍。
//这些锁定的输出可以通过调用UnlockOutputs释放。
	for _, utxo := range spentOutputs {
		utxo.isLocked = true
	}

	return tx, nil
}

//unlockoutputs解锁先前由于
//被选中通过CreateTransaction方法为交易提供资金。
//
//此函数对于并发访问是安全的。
func (m *memWallet) UnlockOutputs(inputs []*wire.TxIn) {
	m.Lock()
	defer m.Unlock()

	for _, input := range inputs {
		utxo, ok := m.utxos[input.PreviousOutPoint]
		if !ok {
			continue
		}

		utxo.isLocked = false
	}
}

//确认余额返回钱包的确认余额。
//
//此函数对于并发访问是安全的。
func (m *memWallet) ConfirmedBalance() btcutil.Amount {
	m.RLock()
	defer m.RUnlock()

	var balance btcutil.Amount
	for _, utxo := range m.utxos {
//防止任何不成熟或锁定的输出对
//钱包的总确认余额。
		if !utxo.isMature(m.currentHeight) || utxo.isLocked {
			continue
		}

		balance += utxo.value
	}

	return balance
}

//keytoaddr将传递的private映射到相应的p2pkh地址。
func keyToAddr(key *btcec.PrivateKey, net *chaincfg.Params) (btcutil.Address, error) {
	serializedKey := key.PubKey().SerializeCompressed()
	pubKeyAddr, err := btcutil.NewAddressPubKey(serializedKey, net)
	if err != nil {
		return nil, err
	}
	return pubKeyAddr.AddressPubKeyHash(), nil
}
