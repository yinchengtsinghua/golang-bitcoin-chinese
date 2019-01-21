
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

//在这个包中测试的绝大多数规则都是从
//基于Java的原始“官方”块验收测试
//https://github.com/thebluematt/test-scripts以及一些附加测试
//在同一核心python端口中可用。

package fullblocktests

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"runtime"
	"time"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

const (
//这里特意定义，而不是使用代码库中的常量
//确保检测到一致性变化。
	maxBlockSigOps       = 20000
	maxBlockSize         = 1000000
	minCoinbaseScriptLen = 2
	maxCoinbaseScriptLen = 100
	medianTimeBlocks     = 11
	maxScriptElementSize = 520

//numlargereorgblocks是要在大型块中使用的块数
//REORG测试（启用时）。这相当于一周的价值
//街区。
	numLargeReorgBlocks = 1088
)

var (
//optruesccript只是一个包含opu-true的公钥脚本
//操作码。这里定义它是为了减少垃圾创建。
	opTrueScript = []byte{txscript.OP_TRUE}

//低费用是一个单一的Satoshi，它的存在是为了使测试代码更多
//可读。
	lowFee = btcutil.Amount(1)
)

//TestInstance是描述返回的特定测试实例的接口
//通过在此包中生成的测试。它应该是断言为1的类型
//以进行相应的测试。
type TestInstance interface {
	FullBlockTestInstance()
}

//AcceptedBlock定义了一个测试实例，该实例希望将块接受为
//区块链可以通过扩展主链、侧链或作为
//孤儿。
type AcceptedBlock struct {
	Name        string
	Block       *wire.MsgBlock
	Height      int32
	IsMainChain bool
	IsOrphan    bool
}

//确保AcceptedBlock实现TestInstance接口。
var _ TestInstance = AcceptedBlock{}

//FullBlockTestInstance只存在于允许接受的块被视为
//测试实例。
//
//这实现了TestInstance接口。
func (b AcceptedBlock) FullBlockTestInstance() {}

//RejectedBlock定义希望块被拒绝的测试实例
//区块链共识规则。
type RejectedBlock struct {
	Name       string
	Block      *wire.MsgBlock
	Height     int32
	RejectCode blockchain.ErrorCode
}

//确保RejectedBlock实现TestInstance接口。
var _ TestInstance = RejectedBlock{}

//完全块测试状态只存在于允许将拒绝的块视为
//测试实例。
//
//这实现了TestInstance接口。
func (b RejectedBlock) FullBlockTestInstance() {}

//OrphanorRejectedBlock定义的测试实例要求块
//被接纳为孤儿或被拒绝。这是有用的，因为有些
//当
//他们的父母以前被拒绝，而其他人可能会接受它作为
//最终被刷新的孤立（因为父级永远不能被接受
//最终将其联系起来）。
type OrphanOrRejectedBlock struct {
	Name   string
	Block  *wire.MsgBlock
	Height int32
}

//确保ExpectedTip实现TestInstance接口。
var _ TestInstance = OrphanOrRejectedBlock{}

//fullBlockTestInstance只存在于允许OrphanorRejectedBlock
//作为证词对待。
//
//这实现了TestInstance接口。
func (b OrphanOrRejectedBlock) FullBlockTestInstance() {}

//ExpectedTip定义了一个测试实例，该实例希望块是当前的
//主链顶端。
type ExpectedTip struct {
	Name   string
	Block  *wire.MsgBlock
	Height int32
}

//确保ExpectedTip实现TestInstance接口。
var _ TestInstance = ExpectedTip{}

//FullBlockTestInstance只允许将ExpectedTip视为
//测试实例。
//
//这实现了TestInstance接口。
func (b ExpectedTip) FullBlockTestInstance() {}

//RejectedNonCanonicalBlock定义了一个需要序列化的测试实例
//不规范的块，因此应拒绝。
type RejectedNonCanonicalBlock struct {
	Name     string
	RawBlock []byte
	Height   int32
}

//fullblockTestInstance只允许将拒绝的非异常锁视为
//一个测试实例。
//
//这实现了TestInstance接口。
func (b RejectedNonCanonicalBlock) FullBlockTestInstance() {}

//Spendableout表示可与
//其他元数据，如块及其支付的金额。
type spendableOut struct {
	prevOut wire.OutPoint
	amount  btcutil.Amount
}

//MakeSpendableOutportX返回给定事务的可消费输出
//以及事务内的事务输出索引。
func makeSpendableOutForTx(tx *wire.MsgTx, txOutIndex uint32) spendableOut {
	return spendableOut{
		prevOut: wire.OutPoint{
			Hash:  tx.TxHash(),
			Index: txOutIndex,
		},
		amount: btcutil.Amount(tx.TxOut[txOutIndex].Value),
	}
}

//MakeSpendableOut返回给定块、事务的可消费输出
//块内的索引和事务内的事务输出索引。
func makeSpendableOut(block *wire.MsgBlock, txIndex, txOutIndex uint32) spendableOut {
	return makeSpendableOutForTx(block.Transactions[txIndex], txOutIndex)
}

//测试生成器包含易于生成测试块的状态
//相互之间建造，同时容纳其他有用的东西，如
//在整个测试中使用的可用可消费输出。
type testGenerator struct {
	params       *chaincfg.Params
	tip          *wire.MsgBlock
	tipName      string
	tipHeight    int32
	blocks       map[chainhash.Hash]*wire.MsgBlock
	blocksByName map[string]*wire.MsgBlock
	blockHeights map[string]int32

//用于跟踪可消费的coinbase输出。
	spendableOuts     []spendableOut
	prevCollectedHash chainhash.Hash

//需要签名事务的任何测试的通用密钥。
	privKey *btcec.PrivateKey
}

//MakeTestGenerator返回用初始化的测试生成器实例
//创世块作为尖端。
func makeTestGenerator(params *chaincfg.Params) (testGenerator, error) {
	privKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), []byte{0x01})
	genesis := params.GenesisBlock
	genesisHash := genesis.BlockHash()
	return testGenerator{
		params:       params,
		blocks:       map[chainhash.Hash]*wire.MsgBlock{genesisHash: genesis},
		blocksByName: map[string]*wire.MsgBlock{"genesis": genesis},
		blockHeights: map[string]int32{"genesis": 0},
		tip:          genesis,
		tipName:      "genesis",
		tipHeight:    0,
		privKey:      privKey,
	}, nil
}

//PayToScriptHashScript返回提供的
//赎回脚本。
func payToScriptHashScript(redeemScript []byte) []byte {
	redeemScriptHash := btcutil.Hash160(redeemScript)
	script, err := txscript.NewScriptBuilder().
		AddOp(txscript.OP_HASH160).AddData(redeemScriptHash).
		AddOp(txscript.OP_EQUAL).Script()
	if err != nil {
		panic(err)
	}
	return script
}

//pushdatascript返回单独推送所提供项的脚本
//到堆栈。
func pushDataScript(items ...[]byte) []byte {
	builder := txscript.NewScriptBuilder()
	for _, item := range items {
		builder.AddData(item)
	}
	script, err := builder.Script()
	if err != nil {
		panic(err)
	}
	return script
}

//StandardCoinBaseScript返回适合用作
//新块的CoinBase事务的签名脚本。特别地，
//它以版本2块所需的块高度开始。
func standardCoinbaseScript(blockHeight int32, extraNonce uint64) ([]byte, error) {
	return txscript.NewScriptBuilder().AddInt64(int64(blockHeight)).
		AddInt64(int64(extraNonce)).Script()
}

//op return script返回一个可证明的可删减的op_返回脚本，
//提供数据。
func opReturnScript(data []byte) []byte {
	builder := txscript.NewScriptBuilder()
	script, err := builder.AddOp(txscript.OP_RETURN).AddData(data).Script()
	if err != nil {
		panic(err)
	}
	return script
}

//uniqueopreturnscript返回一个标准的可证明可删减的op_返回脚本
//以随机的uint64编码为数据。
func uniqueOpReturnScript() []byte {
	rand, err := wire.RandomUint64()
	if err != nil {
		panic(err)
	}

	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data[0:8], rand)
	return opReturnScript(data)
}

//createCoinBaseTx返回一个支付适当
//根据通过的砌块高度给予补贴。CoinBase签名脚本
//符合版本2块的要求。
func (g *testGenerator) createCoinbaseTx(blockHeight int32) *wire.MsgTx {
	extraNonce := uint64(0)
	coinbaseScript, err := standardCoinbaseScript(blockHeight, extraNonce)
	if err != nil {
		panic(err)
	}

	tx := wire.NewMsgTx(1)
	tx.AddTxIn(&wire.TxIn{
//CoinBase事务没有输入，因此以前的输出点是
//零哈希和最大索引。
		PreviousOutPoint: *wire.NewOutPoint(&chainhash.Hash{},
			wire.MaxPrevOutIndex),
		Sequence:        wire.MaxTxInSequenceNum,
		SignatureScript: coinbaseScript,
	})
	tx.AddTxOut(&wire.TxOut{
		Value:    blockchain.CalcBlockSubsidy(blockHeight, g.params),
		PkScript: opTrueScript,
	})
	return tx
}

//calcmerkleroot从事务切片创建一个merkle树，并
//返回树的根。
func calcMerkleRoot(txns []*wire.MsgTx) chainhash.Hash {
	if len(txns) == 0 {
		return chainhash.Hash{}
	}

	utilTxns := make([]*btcutil.Tx, 0, len(txns))
	for _, tx := range txns {
		utilTxns = append(utilTxns, btcutil.NewTx(tx))
	}
	merkles := blockchain.BuildMerkleTreeStore(utilTxns, false)
	return *merkles[len(merkles)-1]
}

//SolveBlock尝试查找一个使传递的块头散列的nonce
//小于目标难度的值。当成功的解决方案是
//返回found true并更新传递的头的nonce字段
//解决方案。如果不存在解决方案，则返回false。
//
//注意：此函数永远不会求解nonce为0的块。这样做了
//因此“nextblock”函数可以正确检测nonce的修改时间
//孟格函数。
func solveBlock(header *wire.BlockHeader) bool {
//解算器goroutine使用sbresult发送结果。
	type sbResult struct {
		found bool
		nonce uint32
	}

//解算器接受要测试的块头和非ce范围。它是
//打算作为一个野人来运作。
	targetDifficulty := blockchain.CompactToBig(header.Bits)
	quit := make(chan bool)
	results := make(chan sbResult)
	solver := func(hdr wire.BlockHeader, startNonce, stopNonce uint32) {
//我们需要修改标题的nonce字段，因此请确保
//我们使用原始标题的副本。
		for i := startNonce; i >= startNonce && i <= stopNonce; i++ {
			select {
			case <-quit:
				return
			default:
				hdr.Nonce = i
				hash := hdr.BlockHash()
				if blockchain.HashToBig(&hash).Cmp(
					targetDifficulty) <= 0 {

					results <- sbResult{true, i}
					return
				}
			}
		}
		results <- sbResult{false, 0}
	}

	startNonce := uint32(1)
	stopNonce := uint32(math.MaxUint32)
	numCores := uint32(runtime.NumCPU())
	noncesPerCore := (stopNonce - startNonce) / numCores
	for i := uint32(0); i < numCores; i++ {
		rangeStart := startNonce + (noncesPerCore * i)
		rangeStop := startNonce + (noncesPerCore * (i + 1)) - 1
		if i == numCores-1 {
			rangeStop = stopNonce
		}
		go solver(*header, rangeStart, rangeStop)
	}
	for i := uint32(0); i < numCores; i++ {
		result := <-results
		if result.found {
			close(quit)
			header.Nonce = result.nonce
			return true
		}
	}

	return false
}

//AdditionalCoinBase返回一个函数，该函数本身接受一个块并
//通过将提供的金额添加到coinbase补贴中对其进行修改。
func additionalCoinbase(amount btcutil.Amount) func(*wire.MsgBlock) {
	return func(b *wire.MsgBlock) {
//增加第一个工作证明coinbase补贴
//提供的金额。
		b.Transactions[0].TxOut[0].Value += int64(amount)
	}
}

//AdditionalSpendFee返回一个函数，该函数本身接受一个块并修改
//通过将提供的费用添加到支出交易中。
//
//注意：CoinBase值不会更新以反映附加费用。使用
//为此目的，“additionalCoinBase”。
func additionalSpendFee(fee btcutil.Amount) func(*wire.MsgBlock) {
	return func(b *wire.MsgBlock) {
//通过减少
//支付金额。
		if int64(fee) > b.Transactions[1].TxOut[0].Value {
			panic(fmt.Sprintf("additionalSpendFee: fee of %d "+
				"exceeds available spend transaction value",
				fee))
		}
		b.Transactions[1].TxOut[0].Value -= int64(fee)
	}
}

//replaceSpendscript返回一个函数，该函数本身接受一个块并修改
//它通过替换支出事务的公钥脚本来实现。
func replaceSpendScript(pkScript []byte) func(*wire.MsgBlock) {
	return func(b *wire.MsgBlock) {
		b.Transactions[1].TxOut[0].PkScript = pkScript
	}
}

//replaceCointBaseSigscript返回一个函数，该函数本身接受一个块并
//通过替换coinbase的签名密钥脚本来修改它。
func replaceCoinbaseSigScript(script []byte) func(*wire.MsgBlock) {
	return func(b *wire.MsgBlock) {
		b.Transactions[0].TxIn[0].SignatureScript = script
	}
}

//additionalTx返回一个函数，该函数本身接受一个块并通过
//添加提供的事务。
func additionalTx(tx *wire.MsgTx) func(*wire.MsgBlock) {
	return func(b *wire.MsgBlock) {
		b.AddTransaction(tx)
	}
}

//createSpendtx创建一个从提供的可消费
//输出并包括一个额外的独特的操作返回输出，以确保
//事务以一个唯一的哈希结束。脚本是一个简单的操作
//避免跟踪地址和签名脚本的脚本
//测验。
func createSpendTx(spend *spendableOut, fee btcutil.Amount) *wire.MsgTx {
	spendTx := wire.NewMsgTx(1)
	spendTx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: spend.prevOut,
		Sequence:         wire.MaxTxInSequenceNum,
		SignatureScript:  nil,
	})
	spendTx.AddTxOut(wire.NewTxOut(int64(spend.amount-fee),
		opTrueScript))
	spendTx.AddTxOut(wire.NewTxOut(0, uniqueOpReturnScript()))

	return spendTx
}

//createSpendtxFortx创建从
//提供的事务，并包括一个额外的唯一opu返回输出
//以确保事务以唯一的哈希结束。公钥脚本
//是一个简单的opu-true脚本，它避免了跟踪地址和
//测试中的签名脚本。签名脚本为零。
func createSpendTxForTx(tx *wire.MsgTx, fee btcutil.Amount) *wire.MsgTx {
	spend := makeSpendableOutForTx(tx, 0)
	return createSpendTx(&spend, fee)
}

//NextBlock构建一个新块，扩展与
//生成并将生成器提示更新到新生成的块。
//
//该区块将包括以下内容：
//-向一个剧本支付所需补贴的CoinBase
//-当提供可消耗输出时：
//-从提供的输出中支出以下输出的事务：
//-向opu-true脚本支付输入量减去1个原子的金额。
//-包含带有随机uint64的op_返回输出，以便
//确保事务具有唯一的哈希
//
//此外，如果指定了一个或多个munge函数，则它们将
//在解算块之前用该块调用。这为呼叫者提供了
//修改对测试特别有用的块的机会。
//
//为了简化munge函数中的逻辑，以下规则是
//在调用所有munge函数后应用：
//-除非手动更改，否则将重新计算merkle根
//-除非更改nonce，否则块将被解决。
func (g *testGenerator) nextBlock(blockName string, spend *spendableOut, mungers ...func(*wire.MsgBlock)) *wire.MsgBlock {
//使用任何附加的
//补贴（如有规定）。
	nextHeight := g.tipHeight + 1
	coinbaseTx := g.createCoinbaseTx(nextHeight)
	txns := []*wire.MsgTx{coinbaseTx}
	if spend != nil {
//创建收费为1 Atom的交易
//并相应增加Coinbase补贴。
		fee := btcutil.Amount(1)
		coinbaseTx.TxOut[0].Value += int64(fee)

//创建一个从提供的可消费的
//输出并包括一个额外的唯一操作返回输出到
//确保事务以唯一的哈希结束，然后添加
//将其添加到要包含在块中的事务列表中。
//为了避免
//需要跟踪测试中的地址和签名脚本。
		txns = append(txns, createSpendTx(spend, fee))
	}

//使用上一个块后一秒的时间戳，除非
//这是使用当前时间的第一个块。
	var ts time.Time
	if nextHeight == 1 {
		ts = time.Unix(time.Now().Unix(), 0)
	} else {
		ts = g.tip.Header.Timestamp.Add(time.Second)
	}

	block := wire.MsgBlock{
		Header: wire.BlockHeader{
			Version:    1,
			PrevBlock:  g.tip.BlockHash(),
			MerkleRoot: calcMerkleRoot(txns),
			Bits:       g.params.PowLimitBits,
			Timestamp:  ts,
Nonce:      0, //有待解决。
		},
		Transactions: txns,
	}

//在解决问题之前，执行任何一个块咀嚼。只重新计算
//如果不是由munge函数手动更改，则返回merkle root。
	curMerkleRoot := block.Header.MerkleRoot
	curNonce := block.Header.Nonce
	for _, f := range mungers {
		f(&block)
	}
	if block.Header.MerkleRoot == curMerkleRoot {
		block.Header.MerkleRoot = calcMerkleRoot(block.Transactions)
	}

//只有当nonce没有被munge手动更改时，才能解决该块。
//功能。
	if block.Header.Nonce == curNonce && !solveBlock(&block.Header) {
		panic(fmt.Sprintf("Unable to solve block at height %d",
			nextHeight))
	}

//更新生成器状态并返回块。
	blockHash := block.BlockHash()
	g.blocks[blockHash] = &block
	g.blocksByName[blockName] = &block
	g.blockHeights[blockName] = nextHeight
	g.tip = &block
	g.tipName = blockName
	g.tipHeight = nextHeight
	return &block
}

//UpdateBlockState手动更新生成器状态以删除所有内部
//通过旧哈希映射对块的引用，并为新哈希插入新哈希
//块哈希。如果测试代码必须手动更改块，则此选项非常有用
//“NextBlock”返回后。
func (g *testGenerator) updateBlockState(oldBlockName string, oldBlockHash chainhash.Hash, newBlockName string, newBlock *wire.MsgBlock) {
//从现有条目中查找高度。
	blockHeight := g.blockHeights[oldBlockName]

//删除现有条目。
	delete(g.blocks, oldBlockHash)
	delete(g.blocksByName, oldBlockName)
	delete(g.blockHeights, oldBlockName)

//添加新条目。
	newBlockHash := newBlock.BlockHash()
	g.blocks[newBlockHash] = newBlock
	g.blocksByName[newBlockName] = newBlock
	g.blockHeights[newBlockName] = blockHeight
}

//setip使用提供的名称将实例的尖端更改为块。
//这很有用，因为提示用于生成后续
//阻碍。
func (g *testGenerator) setTip(blockName string) {
	g.tip = g.blocksByName[blockName]
	g.tipName = blockName
	g.tipHeight = g.blockHeights[blockName]
}

//OldestCoinBaseOuts删除以前
//保存到生成器并将集作为切片返回。
func (g *testGenerator) oldestCoinbaseOut() spendableOut {
	op := g.spendableOuts[0]
	g.spendableOuts = g.spendableOuts[1:]
	return op
}

//saveTipCoinBaseOut将当前提示块中的CoinBase Tx输出添加到
//可消费输出的列表。
func (g *testGenerator) saveTipCoinbaseOut() {
	g.spendableOuts = append(g.spendableOuts, makeSpendableOut(g.tip, 0, 0))
	g.prevCollectedHash = g.tip.BlockHash()
}

//saveSpendablePointOuts添加最后一个块的所有CoinBase输出，
//将其coinbase tx输出收集到当前提示。这对
//一旦测试达到稳定点，批量收集coinbase输出
//因此，他们不必为正确的测试手动添加它们，这将
//最终成为最好的链条。
func (g *testGenerator) saveSpendableCoinbaseOuts() {
//完成后，确保将提示重置为当前提示。
	curTipName := g.tipName
	defer g.setTip(curTipName)

//循环浏览当前提示的祖先，直到
//到达已具有coinbase输出的块
//收集。
	var collectBlocks []*wire.MsgBlock
	for b := g.tip; b != nil; b = g.blocks[b.Header.PrevBlock] {
		if b.BlockHash() == g.prevCollectedHash {
			break
		}
		collectBlocks = append(collectBlocks, b)
	}
	for i := range collectBlocks {
		g.tip = collectBlocks[len(collectBlocks)-1-i]
		g.saveTipCoinbaseOut()
	}
}

//noncanonicalvarint返回已编码的变长编码整数
//使用9个字节，即使它可以用最小的规范
//编码。
func nonCanonicalVarInt(val uint32) []byte {
	var rv [9]byte
	rv[0] = 0xff
	binary.LittleEndian.PutUint64(rv[1:], uint64(val))
	return rv[:]
}

//encodenoncanonicalblock以非规范方式序列化块
//使用可变长度编码整数对事务数进行编码
//使用9个字节，即使它应该用最小的规范
//编码。
func encodeNonCanonicalBlock(b *wire.MsgBlock) []byte {
	var buf bytes.Buffer
	b.Header.BtcEncode(&buf, 0, wire.BaseEncoding)
	buf.Write(nonCanonicalVarInt(uint32(len(b.Transactions))))
	for _, tx := range b.Transactions {
		tx.BtcEncode(&buf, 0, wire.BaseEncoding)
	}
	return buf.Bytes()
}

//CloneBlock返回所提供块的深度副本。
func cloneBlock(b *wire.MsgBlock) wire.MsgBlock {
	var blockCopy wire.MsgBlock
	blockCopy.Header = b.Header
	for _, tx := range b.Transactions {
		blockCopy.AddTransaction(tx.Copy())
	}
	return blockCopy
}

//repeatopcode返回一个字节片，其中提供的操作码重复了
//指定的次数。
func repeatOpcode(opcode uint8, numRepeats int) []byte {
	return bytes.Repeat([]byte{opcode}, numRepeats)
}

//如果提供的脚本没有
//指定的签名操作数。
func assertScriptSigOpsCount(script []byte, expected int) {
	numSigOps := txscript.GetSigOpCount(script)
	if numSigOps != expected {
		_, file, line, _ := runtime.Caller(1)
		panic(fmt.Sprintf("assertion failed at %s:%d: generated number "+
			"of sigops for script is %d instead of expected %d",
			file, line, numSigOps, expected))
	}
}

//countblocksigops返回
//传递的块中的脚本。
func countBlockSigOps(block *wire.MsgBlock) int {
	totalSigOps := 0
	for _, tx := range block.Transactions {
		for _, txIn := range tx.TxIn {
			numSigOps := txscript.GetSigOpCount(txIn.SignatureScript)
			totalSigOps += numSigOps
		}
		for _, txOut := range tx.TxOut {
			numSigOps := txscript.GetSigOpCount(txOut.PkScript)
			totalSigOps += numSigOps
		}
	}

	return totalSigOps
}

//如果与关联的当前提示块
//生成器没有指定数量的签名操作。
func (g *testGenerator) assertTipBlockSigOpsCount(expected int) {
	numSigOps := countBlockSigOps(g.tip)
	if numSigOps != expected {
		panic(fmt.Sprintf("generated number of sigops for block %q "+
			"(height %d) is %d instead of expected %d", g.tipName,
			g.tipHeight, numSigOps, expected))
	}
}

//如果当前提示块与
//序列化时生成器没有指定的大小。
func (g *testGenerator) assertTipBlockSize(expected int) {
	serializeSize := g.tip.SerializeSize()
	if serializeSize != expected {
		panic(fmt.Sprintf("block size of block %q (height %d) is %d "+
			"instead of expected %d", g.tipName, g.tipHeight,
			serializeSize, expected))
	}
}

//如果当前提示块
//与生成器关联的没有指定的非规范大小
//序列化时。
func (g *testGenerator) assertTipNonCanonicalBlockSize(expected int) {
	serializeSize := len(encodeNonCanonicalBlock(g.tip))
	if serializeSize != expected {
		panic(fmt.Sprintf("block size of block %q (height %d) is %d "+
			"instead of expected %d", g.tipName, g.tipHeight,
			serializeSize, expected))
	}
}

//如果当前提示中的事务数为
//与生成器关联的块与指定的值不匹配。
func (g *testGenerator) assertTipBlockNumTxns(expected int) {
	numTxns := len(g.tip.Transactions)
	if numTxns != expected {
		panic(fmt.Sprintf("number of txns in block %q (height %d) is "+
			"%d instead of expected %d", g.tipName, g.tipHeight,
			numTxns, expected))
	}
}

//如果与
//生成器与指定的哈希不匹配。
func (g *testGenerator) assertTipBlockHash(expected chainhash.Hash) {
	hash := g.tip.BlockHash()
	if hash != expected {
		panic(fmt.Sprintf("block hash of block %q (height %d) is %v "+
			"instead of expected %v", g.tipName, g.tipHeight, hash,
			expected))
	}
}

//如果当前头段中的merkle根，则断言tipblockmerkleroot将终止
//与生成器关联的提示块与指定的哈希不匹配。
func (g *testGenerator) assertTipBlockMerkleRoot(expected chainhash.Hash) {
	hash := g.tip.Header.MerkleRoot
	if hash != expected {
		panic(fmt.Sprintf("merkle root of block %q (height %d) is %v "+
			"instead of expected %v", g.tipName, g.tipHeight, hash,
			expected))
	}
}

//如果与关联的当前提示块
//生成器没有用于处的事务输出的op_返回脚本
//提供的Tx索引和输出索引。
func (g *testGenerator) assertTipBlockTxOutOpReturn(txIndex, txOutIndex uint32) {
	if txIndex >= uint32(len(g.tip.Transactions)) {
		panic(fmt.Sprintf("Transaction index %d in block %q "+
			"(height %d) does not exist", txIndex, g.tipName,
			g.tipHeight))
	}

	tx := g.tip.Transactions[txIndex]
	if txOutIndex >= uint32(len(tx.TxOut)) {
		panic(fmt.Sprintf("transaction index %d output %d in block %q "+
			"(height %d) does not exist", txIndex, txOutIndex,
			g.tipName, g.tipHeight))
	}

	txOut := tx.TxOut[txOutIndex]
	if txOut.PkScript[0] != txscript.OP_RETURN {
		panic(fmt.Sprintf("transaction index %d output %d in block %q "+
			"(height %d) is not an OP_RETURN", txIndex, txOutIndex,
			g.tipName, g.tipHeight))
	}
}

//生成返回可用于执行一致性的测试切片
//验证规则。试验应足够灵活，以便
//直接针对区块链代码的单元风格测试以及集成
//对等网络上的样式测试。为了实现这一目标，每项测试
//包含有关预期结果的其他信息，但是
//在两个测试之间进行比较测试时，可以忽略信息
//对等网络上的独立版本。
func Generate(includeLargeReorg bool) (tests [][]TestInstance, err error) {
//为了简化生成代码
//失败，除非测试代码本身被破坏，否则将使用恐慌
//内部的。这个延迟的func确保任何恐慌都不会逃脱
//通过将命名的错误返回替换为基础
//恐慌错误。
	defer func() {
		if r := recover(); r != nil {
			tests = nil

			switch rt := r.(type) {
			case string:
				err = errors.New(rt)
			case error:
				err = rt
			default:
				err = errors.New("Unknown panic")
			}
		}
	}()

//创建用Genesis块初始化的测试生成器实例
//作为小费。
	g, err := makeTestGenerator(regressionNetParams)
	if err != nil {
		return nil, err
	}

//定义一些方便助手函数以返回单个测试
//具有所述特征的实例。
//
//AcceptBlock创建需要提供的块的测试实例
//被共识规则接受。
//
//rejectBlock创建期望提供的块的测试实例
//被共识规则拒绝。
//
//rejectnoncanonicalblock创建一个测试实例，该实例对
//提供了使用非规范编码的块，如
//encodenoncanonicalblock函数，应将其拒绝。
//
//OrphanorRejectBlock创建的测试实例应为
//被接受为孤儿或被
//共识规则。
//
//ExpectTipBlock创建的测试实例需要
//块是块链的当前尖端。
	acceptBlock := func(blockName string, block *wire.MsgBlock, isMainChain, isOrphan bool) TestInstance {
		blockHeight := g.blockHeights[blockName]
		return AcceptedBlock{blockName, block, blockHeight, isMainChain,
			isOrphan}
	}
	rejectBlock := func(blockName string, block *wire.MsgBlock, code blockchain.ErrorCode) TestInstance {
		blockHeight := g.blockHeights[blockName]
		return RejectedBlock{blockName, block, blockHeight, code}
	}
	rejectNonCanonicalBlock := func(blockName string, block *wire.MsgBlock) TestInstance {
		blockHeight := g.blockHeights[blockName]
		encoded := encodeNonCanonicalBlock(block)
		return RejectedNonCanonicalBlock{blockName, encoded, blockHeight}
	}
	orphanOrRejectBlock := func(blockName string, block *wire.MsgBlock) TestInstance {
		blockHeight := g.blockHeights[blockName]
		return OrphanOrRejectedBlock{blockName, block, blockHeight}
	}
	expectTipBlock := func(blockName string, block *wire.MsgBlock) TestInstance {
		blockHeight := g.blockHeights[blockName]
		return ExpectedTip{blockName, block, blockHeight}
	}

//定义一些方便助手函数来填充测试切片
//具有所述特征的测试实例。
//
//accepted为创建并附加单个acceptBlock测试实例
//当前提示，希望块被主服务器接受。
//链。
//
//AcceptedToSideChainwithExpectedTip创建一个附加两个实例
//测试。第一个实例是
//当前提示，期望块被接受为侧链。
//第二个实例是提供的ExpectBlockTip测试实例
//价值观。
//
//被拒绝为创建并附加单个rejectBlock测试实例
//当前提示。
//
//拒绝非规范创建并附加一个
//拒绝当前提示的非异常锁测试实例。
//
//OrphanedOrRejected创建并附加单个OrphaneOrRejectBlock
//当前提示的测试实例。
	accepted := func() {
		tests = append(tests, []TestInstance{
			acceptBlock(g.tipName, g.tip, true, false),
		})
	}
	acceptedToSideChainWithExpectedTip := func(tipName string) {
		tests = append(tests, []TestInstance{
			acceptBlock(g.tipName, g.tip, false, false),
			expectTipBlock(tipName, g.blocksByName[tipName]),
		})
	}
	rejected := func(code blockchain.ErrorCode) {
		tests = append(tests, []TestInstance{
			rejectBlock(g.tipName, g.tip, code),
		})
	}
	rejectedNonCanonical := func() {
		tests = append(tests, []TestInstance{
			rejectNonCanonicalBlock(g.tipName, g.tip),
		})
	}
	orphanedOrRejected := func() {
		tests = append(tests, []TestInstance{
			orphanOrRejectBlock(g.tipName, g.tip),
		})
	}

//————————————————————————————————————————————————————————————————
//生成足够的块，以便有成熟的coinbase输出可供使用。
//
//Genesis->BM0->BM1->-BM99
//————————————————————————————————————————————————————————————————

	coinbaseMaturity := g.params.CoinbaseMaturity
	var testInstances []TestInstance
	for i := uint16(0); i < coinbaseMaturity; i++ {
		blockName := fmt.Sprintf("bm%d", i)
		g.nextBlock(blockName, nil)
		g.saveTipCoinbaseOut()
		testInstances = append(testInstances, acceptBlock(g.tipName,
			g.tip, true, false))
	}
	tests = append(tests, testInstances)

//收集可消费的输出。这简化了下面的代码。
	var outs []*spendableOut
	for i := uint16(0); i < coinbaseMaturity; i++ {
		op := g.oldestCoinbaseOut()
		outs = append(outs, &op)
	}

//————————————————————————————————————————————————————————————————
//基本的分叉和重组测试。
//————————————————————————————————————————————————————————————————

//————————————————————————————————————————————————————————————————
//下面的注释确定了正在构建的链的结构。
//
//括号中的值表示正在使用的输出。
//
//例如，b1（0）表示第一个收集到的可消费输出
//因为上面的代码创建了正确的块数，
//在当前块高度可以使用的第一个输出
//符合Coinbase到期要求。
//————————————————————————————————————————————————————————————————

//首先在当前提示处构建两个块（以parens为单位的值
//是消耗哪个输出）：
//
//…->b1（0）->b2（1）
	g.nextBlock("b1", outs[0])
	accepted()

	g.nextBlock("b2", outs[1])
	accepted()

//从b1创建一个分叉。不应该有REORG，因为看到了B2
//第一。
//
//…->b1（0）->b2（1）
//-B3（1）
	g.setTip("b1")
	g.nextBlock("b3", outs[1])
	b3Tx1Out := makeSpendableOut(g.tip, 1, 0)
	acceptedToSideChainWithExpectedTip("b2")

//延伸B3叉，使替代链更长，并强制重新排列。
//
//…->b1（0）->b2（1）
//\->B3（1）->B4（2）
	g.nextBlock("b4", outs[2])
	accepted()

//将b2拨叉伸出两次，使第一个链条变长并强制重新定位。
//
//…->b1（0）->b2（1）->b5（2）->b6（3）
//\->B3（1）->B4（2）
	g.setTip("b2")
	g.nextBlock("b5", outs[2])
	acceptedToSideChainWithExpectedTip("b4")

	g.nextBlock("b6", outs[3])
	accepted()

//————————————————————————————————————————————————————————————————
//双倍花费测试。
//————————————————————————————————————————————————————————————————

//创造一个双倍消费的叉子。
//
//…->b1（0）->b2（1）->b5（2）->b6（3）
//\->B7（2）->B8（4）
//\->B3（1）->B4（2）
	g.setTip("b5")
	g.nextBlock("b7", outs[2])
	acceptedToSideChainWithExpectedTip("b6")

	g.nextBlock("b8", outs[4])
	rejected(blockchain.ErrMissingTxOut)

//————————————————————————————————————————————————————————————————
//工作证明太多，coinbase测试。
//————————————————————————————————————————————————————————————————

//创建一个生成过多coinbase的块。
//
//…->b1（0）->b2（1）->b5（2）->b6（3）
//-B9（4）
//\->B3（1）->B4（2）
	g.setTip("b6")
	g.nextBlock("b9", outs[4], additionalCoinbase(1))
	rejected(blockchain.ErrBadCoinbaseValue)

//创建一个以生成过多coinbase的块结尾的分叉。
//
//…->b1（0）->b2（1）->b5（2）->b6（3）
//\->B10（3）->B11（4）
//\->B3（1）->B4（2）
	g.setTip("b5")
	g.nextBlock("b10", outs[3])
	acceptedToSideChainWithExpectedTip("b6")

	g.nextBlock("b11", outs[4], additionalCoinbase(1))
	rejected(blockchain.ErrBadCoinbaseValue)

//创建一个以生成过多coinbase的块结尾的分叉
//和以前一样，但先用一个有效的叉子。
//
//…->b1（0）->b2（1）->b5（2）->b6（3）
//\->B12（3）->B13（4）->B14（5）
//（最后添加B12）
//\->B3（1）->B4（2）
	g.setTip("b5")
	b12 := g.nextBlock("b12", outs[3])
	b13 := g.nextBlock("b13", outs[4])
	b14 := g.nextBlock("b14", outs[5], additionalCoinbase(1))
	tests = append(tests, []TestInstance{
		acceptBlock("b13", b13, false, true),
		acceptBlock("b14", b14, false, true),
		rejectBlock("b12", b12, blockchain.ErrBadCoinbaseValue),
		expectTipBlock("b13", b13),
	})

//————————————————————————————————————————————————————————————————
//checksig签名操作计数测试。
//————————————————————————————————————————————————————————————————

//使用op_checksig添加具有最大允许签名操作的块。
//
//…->B5（2）->B12（3）->B13（4）->B15（5）
//\->B3（1）->B4（2）
	g.setTip("b13")
	manySigOps := repeatOpcode(txscript.OP_CHECKSIG, maxBlockSigOps)
	g.nextBlock("b15", outs[5], replaceSpendScript(manySigOps))
	g.assertTipBlockSigOpsCount(maxBlockSigOps)
	accepted()

//尝试添加具有超过允许的最大签名操作数的块
//使用op_checksig。
//
//…->B5（2）->B12（3）->B13（4）->B15（5）
//
//\->B3（1）->B4（2）
	tooManySigOps := repeatOpcode(txscript.OP_CHECKSIG, maxBlockSigOps+1)
	g.nextBlock("b16", outs[6], replaceSpendScript(tooManySigOps))
	g.assertTipBlockSigOpsCount(maxBlockSigOps + 1)
	rejected(blockchain.ErrTooManySigOps)

//————————————————————————————————————————————————————————————————
//交叉叉支出测试。
//————————————————————————————————————————————————————————————————

//创建在另一个分叉上花费Tx的块。
//
//…->B5（2）->B12（3）->B13（4）->B15（5）
//\->B17（b3.tx[1]）
//\->B3（1）->B4（2）
	g.setTip("b15")
	g.nextBlock("b17", &b3Tx1Out)
	rejected(blockchain.ErrMissingTxOut)

//创建分叉块并在第三个分叉上花费创建的Tx。
//
//…->B5（2）->B12（3）->B13（4）->B15（5）
//\->B18（B3.TX[1]）->B19（6）
//\->B3（1）->B4（2）
	g.setTip("b13")
	g.nextBlock("b18", &b3Tx1Out)
	acceptedToSideChainWithExpectedTip("b15")

	g.nextBlock("b19", outs[6])
	rejected(blockchain.ErrMissingTxOut)

//————————————————————————————————————————————————————————————————
//不成熟的硬币库测试。
//————————————————————————————————————————————————————————————————

//创建使用不成熟的coinbase的块。
//
//…->B13（4）->B15（5）
//
	g.setTip("b15")
	g.nextBlock("b20", outs[7])
	rejected(blockchain.ErrImmatureSpend)

//创建将不成熟的硬币放在叉子上的块。
//
//…->B13（4）->B15（5）
//\->B21（5）->B22（7）
	g.setTip("b13")
	g.nextBlock("b21", outs[5])
	acceptedToSideChainWithExpectedTip("b15")

	g.nextBlock("b22", outs[7])
	rejected(blockchain.ErrImmatureSpend)

//————————————————————————————————————————————————————————————————
//最大块大小测试。
//————————————————————————————————————————————————————————————————

//创建最大允许大小的块。
//
//…->B15（5）->B23（6）
	g.setTip("b15")
	g.nextBlock("b23", outs[6], func(b *wire.MsgBlock) {
		bytesToMaxSize := maxBlockSize - b.SerializeSize() - 3
		sizePadScript := repeatOpcode(0x00, bytesToMaxSize)
		replaceSpendScript(sizePadScript)(b)
	})
	g.assertTipBlockSize(maxBlockSize)
	accepted()

//创建大于最大允许大小一个字节的块。这个
//是在叉子上完成的，不管怎样都应该被拒绝。
//
//…->B15（5）->B23（6）
//\->B24（6）->B25（7）
	g.setTip("b15")
	g.nextBlock("b24", outs[6], func(b *wire.MsgBlock) {
		bytesToMaxSize := maxBlockSize - b.SerializeSize() - 3
		sizePadScript := repeatOpcode(0x00, bytesToMaxSize+1)
		replaceSpendScript(sizePadScript)(b)
	})
	g.assertTipBlockSize(maxBlockSize + 1)
	rejected(blockchain.ErrBlockTooBig)

//父级被拒绝，因此此块必须是孤立的或
//由于父级无效而直接拒绝。
	g.nextBlock("b25", outs[7])
	orphanedOrRejected()

//————————————————————————————————————————————————————————————————
//CoinBase脚本长度限制测试。
//————————————————————————————————————————————————————————————————

//创建具有小于
//所需长度。这是在叉子上完成的，应该被拒绝
//无论如何。另外，创建一个构建在被拒绝块上的块。
//
//…->B15（5）->B23（6）
//\->B26（6）->B27（7）
	g.setTip("b15")
	tooSmallCbScript := repeatOpcode(0x00, minCoinbaseScriptLen-1)
	g.nextBlock("b26", outs[6], replaceCoinbaseSigScript(tooSmallCbScript))
	rejected(blockchain.ErrBadCoinbaseScriptLen)

//父级被拒绝，因此此块必须是孤立的或
//由于父级无效而直接拒绝。
	g.nextBlock("b27", outs[7])
	orphanedOrRejected()

//创建具有大于
//允许长度。这是在叉子上完成的，应该被拒绝
//无论如何。另外，创建一个构建在被拒绝块上的块。
//
//…->B15（5）->B23（6）
//\->B28（6）->B29（7）
	g.setTip("b15")
	tooLargeCbScript := repeatOpcode(0x00, maxCoinbaseScriptLen+1)
	g.nextBlock("b28", outs[6], replaceCoinbaseSigScript(tooLargeCbScript))
	rejected(blockchain.ErrBadCoinbaseScriptLen)

//父级被拒绝，因此此块必须是孤立的或
//由于父级无效而直接拒绝。
	g.nextBlock("b29", outs[7])
	orphanedOrRejected()

//创建具有最大长度coinbase脚本的块。
//
//…->B23（6）->B30（7）
	g.setTip("b23")
	maxSizeCbScript := repeatOpcode(0x00, maxCoinbaseScriptLen)
	g.nextBlock("b30", outs[7], replaceCoinbaseSigScript(maxSizeCbScript))
	accepted()

//————————————————————————————————————————————————————————————————
//multisig[verify]/checksigverify签名操作计数测试。
//————————————————————————————————————————————————————————————————

//创建带有最大签名操作的块作为op_checkmultisig。
//
//…->B30（7）->B31（8）
//
//操作检查20个信号的多信号计数。
	manySigOps = repeatOpcode(txscript.OP_CHECKMULTISIG, maxBlockSigOps/20)
	g.nextBlock("b31", outs[8], replaceSpendScript(manySigOps))
	g.assertTipBlockSigOpsCount(maxBlockSigOps)
	accepted()

//创建具有超过允许的最大签名操作的块，使用
//操作检查多图像。
//
//…-B31（8）
//-B32（9）
//
//操作检查20个信号的多信号计数。
	tooManySigOps = repeatOpcode(txscript.OP_CHECKMULTISIG, maxBlockSigOps/20)
	tooManySigOps = append(manySigOps, txscript.OP_CHECKSIG)
	g.nextBlock("b32", outs[9], replaceSpendScript(tooManySigOps))
	g.assertTipBlockSigOpsCount(maxBlockSigOps + 1)
	rejected(blockchain.ErrTooManySigOps)

//创建带有最大签名操作的块作为op_checkmultisigverify。
//
//…->B31（8）->B33（9）
	g.setTip("b31")
	manySigOps = repeatOpcode(txscript.OP_CHECKMULTISIGVERIFY, maxBlockSigOps/20)
	g.nextBlock("b33", outs[9], replaceSpendScript(manySigOps))
	g.assertTipBlockSigOpsCount(maxBlockSigOps)
	accepted()

//创建具有超过允许的最大签名操作的块，使用
//Op_Checkmultisigverify（检查多图像验证）。
//
//…-B33（9）
//-B34（10）
//
	tooManySigOps = repeatOpcode(txscript.OP_CHECKMULTISIGVERIFY, maxBlockSigOps/20)
	tooManySigOps = append(manySigOps, txscript.OP_CHECKSIG)
	g.nextBlock("b34", outs[10], replaceSpendScript(tooManySigOps))
	g.assertTipBlockSigOpsCount(maxBlockSigOps + 1)
	rejected(blockchain.ErrTooManySigOps)

//使用最大签名操作创建块作为op_checksigverify。
//
//…->B33（9）->B35（10）
//
	g.setTip("b33")
	manySigOps = repeatOpcode(txscript.OP_CHECKSIGVERIFY, maxBlockSigOps)
	g.nextBlock("b35", outs[10], replaceSpendScript(manySigOps))
	g.assertTipBlockSigOpsCount(maxBlockSigOps)
	accepted()

//创建具有超过允许的最大签名操作的块，使用
//操作检查信号验证。
//
//…-B35（10）
//-B36（11）
//
	tooManySigOps = repeatOpcode(txscript.OP_CHECKSIGVERIFY, maxBlockSigOps+1)
	g.nextBlock("b36", outs[11], replaceSpendScript(tooManySigOps))
	g.assertTipBlockSigOpsCount(maxBlockSigOps + 1)
	rejected(blockchain.ErrTooManySigOps)

//————————————————————————————————————————————————————————————————
//在未能连接测试的块中花费Tx输出。
//————————————————————————————————————————————————————————————————

//创建从失败的块花费事务的块
//连接（由于包含了双倍开销）。
//
//…-B35（10）
//-B37（11）
//\->B38（B37.TX[1]）
//
	g.setTip("b35")
	doubleSpendTx := createSpendTx(outs[11], lowFee)
	g.nextBlock("b37", outs[11], additionalTx(doubleSpendTx))
	b37Tx1Out := makeSpendableOut(g.tip, 1, 0)
	rejected(blockchain.ErrMissingTxOut)

	g.setTip("b35")
	g.nextBlock("b38", &b37Tx1Out)
	rejected(blockchain.ErrMissingTxOut)

//————————————————————————————————————————————————————————————————
//付费脚本哈希签名操作计数测试。
//————————————————————————————————————————————————————————————————

//创建一个由9个
//将在接下来的三个块中使用的签名操作。
	const redeemScriptSigOps = 9
	redeemScript := pushDataScript(g.privKey.PubKey().SerializeCompressed())
	redeemScript = append(redeemScript, bytes.Repeat([]byte{txscript.OP_2DUP,
		txscript.OP_CHECKSIGVERIFY}, redeemScriptSigOps-1)...)
	redeemScript = append(redeemScript, txscript.OP_CHECKSIG)
	assertScriptSigOpsCount(redeemScript, redeemScriptSigOps)

//创建一个具有足够的pay-to脚本哈希输出的块，以便
//可以创建另一个块，该块将全部使用并超过
//每个块允许的最大签名操作数。
//
//…->B35（10）->B39（11）
	g.setTip("b35")
	b39 := g.nextBlock("b39", outs[11], func(b *wire.MsgBlock) {
//创建一个交易链，每个支出来自
//前一个，这样每个包含一个输出，支付给
//兑换脚本和签名总数
//这些兑换脚本中的操作将超过
//每个块允许的最大值。
		p2shScript := payToScriptHashScript(redeemScript)
		txnsNeeded := (maxBlockSigOps / redeemScriptSigOps) + 1
		prevTx := b.Transactions[1]
		for i := 0; i < txnsNeeded; i++ {
			prevTx = createSpendTxForTx(prevTx, lowFee)
			prevTx.TxOut[0].Value -= 2
			prevTx.AddTxOut(wire.NewTxOut(2, p2shScript))
			b.AddTransaction(prevTx)
		}
	})
	g.assertTipBlockNumTxns((maxBlockSigOps / redeemScriptSigOps) + 3)
	accepted()

//创建一个具有超过允许的最大签名操作的块，其中
//它们中的大多数是付费脚本散列脚本。
//
//…->B35（10）->B39（11）
//-B40（12）
	g.setTip("b39")
	g.nextBlock("b40", outs[12], func(b *wire.MsgBlock) {
		txnsNeeded := (maxBlockSigOps / redeemScriptSigOps)
		for i := 0; i < txnsNeeded; i++ {
//创建从
//B39中的相关P2SH输出。
			spend := makeSpendableOutForTx(b39.Transactions[i+2], 2)
			tx := createSpendTx(&spend, lowFee)
			sig, err := txscript.RawTxInSignature(tx, 0,
				redeemScript, txscript.SigHashAll, g.privKey)
			if err != nil {
				panic(err)
			}
			tx.TxIn[0].SignatureScript = pushDataScript(sig,
				redeemScript)
			b.AddTransaction(tx)
		}

//创建包含非付费脚本哈希的最终Tx
//输出所需的签名操作数
//超过最大允许值的第一块。
		fill := maxBlockSigOps - (txnsNeeded * redeemScriptSigOps) + 1
		finalTx := b.Transactions[len(b.Transactions)-1]
		tx := createSpendTxForTx(finalTx, lowFee)
		tx.TxOut[0].PkScript = repeatOpcode(txscript.OP_CHECKSIG, fill)
		b.AddTransaction(tx)
	})
	rejected(blockchain.ErrTooManySigOps)

//使用允许的最大签名操作创建一个块，其中
//它们中的大多数都是付费脚本散列脚本。
//
//…->B35（10）->B39（11）->B41（12）
	g.setTip("b39")
	g.nextBlock("b41", outs[12], func(b *wire.MsgBlock) {
		txnsNeeded := (maxBlockSigOps / redeemScriptSigOps)
		for i := 0; i < txnsNeeded; i++ {
			spend := makeSpendableOutForTx(b39.Transactions[i+2], 2)
			tx := createSpendTx(&spend, lowFee)
			sig, err := txscript.RawTxInSignature(tx, 0,
				redeemScript, txscript.SigHashAll, g.privKey)
			if err != nil {
				panic(err)
			}
			tx.TxIn[0].SignatureScript = pushDataScript(sig,
				redeemScript)
			b.AddTransaction(tx)
		}

//创建包含非付费脚本哈希的最终Tx
//输出所需的签名操作数
//块精确到允许的最大值。
		fill := maxBlockSigOps - (txnsNeeded * redeemScriptSigOps)
		if fill == 0 {
			return
		}
		finalTx := b.Transactions[len(b.Transactions)-1]
		tx := createSpendTxForTx(finalTx, lowFee)
		tx.TxOut[0].PkScript = repeatOpcode(txscript.OP_CHECKSIG, fill)
		b.AddTransaction(tx)
	})
	accepted()

//————————————————————————————————————————————————————————————————
//将链条重置为稳定的底座。
//
//…->B35（10）->B39（11）->B42（12）->B43（13）
//-B41（12）
//————————————————————————————————————————————————————————————————

	g.setTip("b39")
	g.nextBlock("b42", outs[12])
	acceptedToSideChainWithExpectedTip("b41")

	g.nextBlock("b43", outs[13])
	accepted()

//————————————————————————————————————————————————————————————————
//各种格式错误的块测试。
//————————————————————————————————————————————————————————————————

//创建块时使用其他有效事务代替Where
//CoinBase必须是。
//
//…-B43（13）
//-> B44（14）
	g.nextBlock("b44", nil, func(b *wire.MsgBlock) {
		nonCoinbaseTx := createSpendTx(outs[14], lowFee)
		b.Transactions[0] = nonCoinbaseTx
	})
	rejected(blockchain.ErrFirstTxNotCoinbase)

//创建不带事务的块。
//
//…-B43（13）
//b>（45）
	g.setTip("b43")
	g.nextBlock("b45", nil, func(b *wire.MsgBlock) {
		b.Transactions = nil
	})
	rejected(blockchain.ErrNoTransactions)

//使用无效的工作证明创建块。
//
//…-B43（13）
//-B46（14）
	g.setTip("b43")
	b46 := g.nextBlock("b46", outs[14])
//不能在传递给nextblock的munge函数内执行此操作
//因为块是在函数返回并进行此测试后解决的
//需要未解决的块。
	{
		origHash := b46.BlockHash()
		for {
//继续递增nonce，直到哈希被视为
//uint256高于限制。
			b46.Header.Nonce++
			blockHash := b46.BlockHash()
			hashNum := blockchain.HashToBig(&blockHash)
			if hashNum.Cmp(g.params.PowLimit) >= 0 {
				break
			}
		}
		g.updateBlockState("b46", origHash, "b46", b46)
	}
	rejected(blockchain.ErrHighHash)

//创建时间戳太远的块。
//
//…-B43（13）
//-B47（14）
	g.setTip("b43")
	g.nextBlock("b47", outs[14], func(b *wire.MsgBlock) {
//3小时后夹紧精度达到1秒。
		nowPlus3Hours := time.Now().Add(time.Hour * 3)
		b.Header.Timestamp = time.Unix(nowPlus3Hours.Unix(), 0)
	})
	rejected(blockchain.ErrTimeTooNew)

//使用无效的merkle根创建块。
//
//…-B43（13）
//> -B48（14）
	g.setTip("b43")
	g.nextBlock("b48", outs[14], func(b *wire.MsgBlock) {
		b.Header.MerkleRoot = chainhash.Hash{}
	})
	rejected(blockchain.ErrBadMerkleRoot)

//使用无效的工作限制证明创建块。
//
//…-B43（13）
//-B49（14）
	g.setTip("b43")
	g.nextBlock("b49", outs[14], func(b *wire.MsgBlock) {
		b.Header.Bits--
	})
	rejected(blockchain.ErrUnexpectedDifficulty)

//使用无效的负工作限制证明创建块。
//
//…-B43（13）
//-B4A（14）
	g.setTip("b43")
	b49a := g.nextBlock("b49a", outs[14])
//不能在传递给nextblock的munge函数内执行此操作
//因为块是在函数返回并进行此测试后解决的
//涉及无法解决的块。
	{
		origHash := b49a.BlockHash()
b49a.Header.Bits = 0x01810000 //-1个紧凑型。
		g.updateBlockState("b49a", origHash, "b49a", b49a)
	}
	rejected(blockchain.ErrUnexpectedDifficulty)

//使用两个CoinBase事务创建块。
//
//…-B43（13）
//-> B50（14）
	g.setTip("b43")
	coinbaseTx := g.createCoinbaseTx(g.tipHeight + 1)
	g.nextBlock("b50", outs[14], additionalTx(coinbaseTx))
	rejected(blockchain.ErrMultipleCoinbases)

//创建具有重复事务的块。
//
//这个测试依赖于Merkle树的形状来测试
//预期的条件，因此在下面断言。
//
//…-B43（13）
//-B51（14）
	g.setTip("b43")
	g.nextBlock("b51", outs[14], func(b *wire.MsgBlock) {
		b.AddTransaction(b.Transactions[1])
	})
	g.assertTipBlockNumTxns(3)
	rejected(blockchain.ErrDuplicateTx)

//创建花费不存在的事务的块。
//
//…-B43（13）
//-> B52（14）
	g.setTip("b43")
	g.nextBlock("b52", outs[14], func(b *wire.MsgBlock) {
		hash := newHashFromStr("00000000000000000000000000000000" +
			"00000000000000000123456789abcdef")
		b.Transactions[1].TxIn[0].PreviousOutPoint.Hash = *hash
		b.Transactions[1].TxIn[0].PreviousOutPoint.Index = 0
	})
	rejected(blockchain.ErrMissingTxOut)

//————————————————————————————————————————————————————————————————
//阻塞头段中间时间测试。
//————————————————————————————————————————————————————————————————

//将链条重置为稳定的底座。
//
//…->B33（9）->B35（10）->B39（11）->B42（12）->B43（13）->B53（14）
	g.setTip("b43")
	g.nextBlock("b53", outs[14])
	accepted()

//创建一个时间戳正好是中间时间的块。这个
//必须拒绝块。
//
//…->B33（9）->B35（10）->B39（11）->B42（12）->B43（13）->B53（14）
//-> B54（15）
	g.nextBlock("b54", outs[15], func(b *wire.MsgBlock) {
		medianBlock := g.blocks[b.Header.PrevBlock]
		for i := 0; i < medianTimeBlocks/2; i++ {
			medianBlock = g.blocks[medianBlock.Header.PrevBlock]
		}
		b.Header.Timestamp = medianBlock.Header.Timestamp
	})
	rejected(blockchain.ErrTimeTooOld)

//创建一个时间戳在中间值后一秒钟的块
//时间。必须接受该块。
//
//…->B33（9）->B35（10）->B39（11）->B42（12）->B43（13）->B53（14）->B55（15）
	g.setTip("b53")
	g.nextBlock("b55", outs[15], func(b *wire.MsgBlock) {
		medianBlock := g.blocks[b.Header.PrevBlock]
		for i := 0; i < medianTimeBlocks/2; i++ {
			medianBlock = g.blocks[medianBlock.Header.PrevBlock]
		}
		medianBlockTime := medianBlock.Header.Timestamp
		b.Header.Timestamp = medianBlockTime.Add(time.Second)
	})
	accepted()

//————————————————————————————————————————————————————————————————
//CVE-2012-2459（Merkle Tree Algo导致的块散列冲突）测试。
//————————————————————————————————————————————————————————————————

//通过merkle树技巧创建两个具有相同哈希的块
//确保接受有效的块，即使它具有相同的块
//哈希作为首先被拒绝的无效块。
//
//这是通过如下构建块来实现的：
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
	g.setTip("b55")
	b57 := g.nextBlock("b57", outs[16], func(b *wire.MsgBlock) {
		tx2 := b.Transactions[1]
		tx3 := createSpendTxForTx(tx2, lowFee)
		b.AddTransaction(tx3)
	})
	g.assertTipBlockNumTxns(3)

	g.setTip("b55")
	b56 := g.nextBlock("b56", nil, func(b *wire.MsgBlock) {
		*b = cloneBlock(b57)
		b.AddTransaction(b.Transactions[2])
	})
	g.assertTipBlockNumTxns(4)
	g.assertTipBlockHash(b57.BlockHash())
	g.assertTipBlockMerkleRoot(b57.Header.MerkleRoot)
	rejected(blockchain.ErrDuplicateTx)

//因为这两个块现在具有相同的哈希和生成器状态
//将b56与哈希关联，手动删除b56，替换它
//使用B57，然后将提示重置为它。
	g.updateBlockState("b56", b56.BlockHash(), "b57", b57)
	g.setTip("b57")
	accepted()

//创建一个包含两个不在
//梅克尔树中的连续位置。
//
//这是通过如下方式构建块来实现的：
//
//交易：CoinBase、TX2、TX3、TX4、TX5、TX6、TX3、TX4
//梅克尔树2级：h12=h（h（cb）h（tx2）
//H34=H（H（TX3）H（TX4））
//
//
//
//h5678=h（h56 h78）
//
//
//
//
//-> B56P2（16）
	g.setTip("b55")
	g.nextBlock("b56p2", outs[16], func(b *wire.MsgBlock) {
//创建4个交易记录，每个交易记录从上一个Tx中支出
//在街区。
		spendTx := b.Transactions[1]
		for i := 0; i < 4; i++ {
			spendTx = createSpendTxForTx(spendTx, lowFee)
			b.AddTransaction(spendTx)
		}

//添加重复的事务（第3和第4）。
		b.AddTransaction(b.Transactions[2])
		b.AddTransaction(b.Transactions[3])
	})
	g.assertTipBlockNumTxns(8)
	rejected(blockchain.ErrDuplicateTx)

//————————————————————————————————————————————————————————————————
//
//————————————————————————————————————————————————————————————————

//
//超出了其他有效和现有Tx的范围。
//
//…-B57（16）
//
	g.setTip("b57")
	g.nextBlock("b58", outs[17], func(b *wire.MsgBlock) {
		b.Transactions[1].TxIn[0].PreviousOutPoint.Index = 42
	})
	rejected(blockchain.ErrMissingTxOut)

//
//
//…-B57（16）
//-> B59（17）
	g.setTip("b57")
	g.nextBlock("b59", outs[17], func(b *wire.MsgBlock) {
		b.Transactions[1].TxOut[0].Value = int64(outs[17].amount) + 1
	})
	rejected(blockchain.ErrSpendTooHigh)

//————————————————————————————————————————————————————————————————
//BIP030试验。
//————————————————————————————————————————————————————————————————

//
//
//
	g.setTip("b57")
	g.nextBlock("b60", outs[17])
	accepted()

//
//
//
//
//
	g.nextBlock("b61", outs[18], func(b *wire.MsgBlock) {
//
//
		parent := g.blocks[b.Header.PrevBlock]
		b.Transactions[0] = parent.Transactions[0]
	})
	rejected(blockchain.ErrOverwriteTx)

//————————————————————————————————————————————————————————————————
//
//————————————————————————————————————————————————————————————————

//
//
//
//
	g.setTip("b60")
	g.nextBlock("b62", outs[18], func(b *wire.MsgBlock) {
//
//
//
		b.Transactions[1].LockTime = 0xffffffff
		b.Transactions[1].TxIn[0].Sequence = 0
	})
	rejected(blockchain.ErrUnfinalizedTx)

//
//
//
//-B63（18）
	g.setTip("b60")
	g.nextBlock("b63", outs[18], func(b *wire.MsgBlock) {
//
//
//
		b.Transactions[0].LockTime = 0xffffffff
		b.Transactions[0].TxIn[0].Sequence = 0
	})
	rejected(blockchain.ErrUnfinalizedTx)

//————————————————————————————————————————————————————————————————
//
//————————————————————————————————————————————————————————————————

//
//
//
//
//
//
//
//
//
//
	g.setTip("b60")
	b64a := g.nextBlock("b64a", outs[18], func(b *wire.MsgBlock) {
		bytesToMaxSize := maxBlockSize - b.SerializeSize() - 3
		sizePadScript := repeatOpcode(0x00, bytesToMaxSize)
		replaceSpendScript(sizePadScript)(b)
	})
	g.assertTipNonCanonicalBlockSize(maxBlockSize + 8)
	rejectedNonCanonical()

	g.setTip("b60")
	b64 := g.nextBlock("b64", outs[18], func(b *wire.MsgBlock) {
		*b = cloneBlock(b64a)
	})
//因为这两个块现在具有相同的哈希和生成器状态
//
//使用B64，然后将提示重置为它。
	g.updateBlockState("b64a", b64a.BlockHash(), "b64", b64)
	g.setTip("b64")
	g.assertTipBlockHash(b64a.BlockHash())
	g.assertTipBlockSize(maxBlockSize)
	accepted()

//————————————————————————————————————————————————————————————————
//
//————————————————————————————————————————————————————————————————

//
//
//
	g.setTip("b64")
	g.nextBlock("b65", outs[19], func(b *wire.MsgBlock) {
		tx3 := createSpendTxForTx(b.Transactions[1], lowFee)
		b.AddTransaction(tx3)
	})
	accepted()

//
//
//
//
	g.nextBlock("b66", nil, func(b *wire.MsgBlock) {
		tx2 := createSpendTx(outs[20], lowFee)
		tx3 := createSpendTxForTx(tx2, lowFee)
		b.AddTransaction(tx3)
		b.AddTransaction(tx2)
	})
	rejected(blockchain.ErrMissingTxOut)

//
//块。
//
//
//
	g.setTip("b65")
	g.nextBlock("b67", outs[20], func(b *wire.MsgBlock) {
		tx2 := b.Transactions[1]
		tx3 := createSpendTxForTx(tx2, lowFee)
		tx4 := createSpendTxForTx(tx2, lowFee)
		b.AddTransaction(tx3)
		b.AddTransaction(tx4)
	})
	rejected(blockchain.ErrMissingTxOut)

//————————————————————————————————————————————————————————————————
//
//————————————————————————————————————————————————————————————————

//
//
//
//
//
	g.setTip("b65")
	g.nextBlock("b68", outs[20], additionalCoinbase(10), additionalSpendFee(9))
	rejected(blockchain.ErrBadCoinbaseValue)

//
//
//
//
	g.setTip("b65")
	g.nextBlock("b69", outs[20], additionalCoinbase(10), additionalSpendFee(10))
	accepted()

//————————————————————————————————————————————————————————————————
//
//
//
//
//
//
//
//————————————————————————————————————————————————————————————————

//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
	scriptSize := maxBlockSigOps + 5 + (maxScriptElementSize + 1) + 1
	tooManySigOps = repeatOpcode(txscript.OP_CHECKSIG, scriptSize)
	tooManySigOps[maxBlockSigOps] = txscript.OP_PUSHDATA4
	binary.LittleEndian.PutUint32(tooManySigOps[maxBlockSigOps+1:],
		maxScriptElementSize+1)
	g.nextBlock("b70", outs[21], replaceSpendScript(tooManySigOps))
	g.assertTipBlockSigOpsCount(maxBlockSigOps + 1)
	rejected(blockchain.ErrTooManySigOps)

//
//
//一个无效的推送数据，它声明大量数据，即使
//没有提供那么多的数据。
//
//…-B59（20）
//-B761（21）
	g.setTip("b69")
	scriptSize = maxBlockSigOps + 5 + maxScriptElementSize + 1
	tooManySigOps = repeatOpcode(txscript.OP_CHECKSIG, scriptSize)
	tooManySigOps[maxBlockSigOps+1] = txscript.OP_PUSHDATA4
	binary.LittleEndian.PutUint32(tooManySigOps[maxBlockSigOps+2:], 0xffffffff)
	g.nextBlock("b71", outs[21], replaceSpendScript(tooManySigOps))
	g.assertTipBlockSigOpsCount(maxBlockSigOps + 1)
	rejected(blockchain.ErrTooManySigOps)

//使用允许的最大签名操作创建块，以便
//计数的签名操作位于无效的推送数据之前，
//声称有大量的数据，即使很多数据不是
//提供。推送数据本身包含opu checksig，因此
//如果计算了其中任何一个块，则该块将被拒绝。
//
//…->B69（20）->B72（21）
	g.setTip("b69")
	scriptSize = maxBlockSigOps + 5 + maxScriptElementSize
	manySigOps = repeatOpcode(txscript.OP_CHECKSIG, scriptSize)
	manySigOps[maxBlockSigOps] = txscript.OP_PUSHDATA4
	binary.LittleEndian.PutUint32(manySigOps[maxBlockSigOps+1:], 0xffffffff)
	g.nextBlock("b72", outs[21], replaceSpendScript(manySigOps))
	g.assertTipBlockSigOpsCount(maxBlockSigOps)
	accepted()

//使用允许的最大签名操作创建块，以便
//计数的签名操作位于无效的推送数据之前，
//
//
//
//
	scriptSize = maxBlockSigOps + 5 + (maxScriptElementSize + 1)
	manySigOps = repeatOpcode(txscript.OP_CHECKSIG, scriptSize)
	manySigOps[maxBlockSigOps] = txscript.OP_PUSHDATA4
	g.nextBlock("b73", outs[22], replaceSpendScript(manySigOps))
	g.assertTipBlockSigOpsCount(maxBlockSigOps)
	accepted()

//————————————————————————————————————————————————————————————————
//
//————————————————————————————————————————————————————————————————

//
//
//
	script := []byte{txscript.OP_IF, txscript.OP_INVALIDOPCODE,
		txscript.OP_ELSE, txscript.OP_TRUE, txscript.OP_ENDIF}
	g.nextBlock("b74", outs[23], replaceSpendScript(script), func(b *wire.MsgBlock) {
		tx2 := b.Transactions[1]
		tx3 := createSpendTxForTx(tx2, lowFee)
		tx3.TxIn[0].SignatureScript = []byte{txscript.OP_FALSE}
		b.AddTransaction(tx3)
	})
	accepted()

//————————————————————————————————————————————————————————————————
//
//————————————————————————————————————————————————————————————————

//
//
//
//
	g.nextBlock("b75", outs[24], func(b *wire.MsgBlock) {
//
//
		const numAdditionalOutputs = 4
		const zeroCoin = int64(0)
		spendTx := b.Transactions[1]
		for i := 0; i < numAdditionalOutputs; i++ {
			spendTx.AddTxOut(wire.NewTxOut(zeroCoin, opTrueScript))
		}

//
//
//
//
		zeroFee := btcutil.Amount(0)
		for i := uint32(0); i < numAdditionalOutputs; i++ {
			spend := makeSpendableOut(b, 1, i+2)
			tx := createSpendTx(&spend, zeroFee)
			b.AddTransaction(tx)
		}
	})
	g.assertTipBlockNumTxns(6)
	g.assertTipBlockTxOutOpReturn(5, 1)
	b75OpReturnOut := makeSpendableOut(g.tip, 5, 1)
	accepted()

//
//
//…-> b74(23) -> b75(24)
//\-> b76(24) -> b77(25)
	g.setTip("b74")
	g.nextBlock("b76", outs[24])
	acceptedToSideChainWithExpectedTip("b75")

	g.nextBlock("b77", outs[25])
	accepted()

//重新组织到包含op_返回的原始链。
//
//…->B74（23）->B75（24）->B78（25）->B79（26）
//\->B76（24）->B77（25）
	g.setTip("b75")
	g.nextBlock("b78", outs[25])
	acceptedToSideChainWithExpectedTip("b77")

	g.nextBlock("b79", outs[26])
	accepted()

//创建用于返回opu的块。
//
//…->B74（23）->B75（24）->B78（25）->B79（26）
//
//
//
//
//有效地否定这种行为。
	b75OpReturnOut.amount++
	g.nextBlock("b80", &b75OpReturnOut)
	rejected(blockchain.ErrMissingTxOut)

//创建一个具有多个opu返回的事务的块。偶数
//
//按照协商一致的规则。
//
//
//
	g.setTip("b79")
	g.nextBlock("b81", outs[27], func(b *wire.MsgBlock) {
		const numAdditionalOutputs = 4
		const zeroCoin = int64(0)
		spendTx := b.Transactions[1]
		for i := 0; i < numAdditionalOutputs; i++ {
			opRetScript := uniqueOpReturnScript()
			spendTx.AddTxOut(wire.NewTxOut(zeroCoin, opRetScript))
		}
	})
	for i := uint32(2); i < 6; i++ {
		g.assertTipBlockTxOutOpReturn(1, i)
	}
	accepted()

//————————————————————————————————————————————————————————————————
//
//————————————————————————————————————————————————————————————————

	if !includeLargeReorg {
		return tests, nil
	}

//
//
//…->B81（27）->
	g.setTip("b81")

//收集以前的所有可消费的CoinBase输出
//收集点到当前提示。
	g.saveSpendableCoinbaseOuts()
	spendableOutOffset := g.tipHeight - int32(coinbaseMaturity)

//将主链延伸大量最大尺寸的块。
//
//…->BR0->BR1->-布鲁
	testInstances = nil
	reorgSpend := *outs[spendableOutOffset]
	reorgStartBlockName := g.tipName
	chain1TipName := g.tipName
	for i := int32(0); i < numLargeReorgBlocks; i++ {
		chain1TipName = fmt.Sprintf("br%d", i)
		g.nextBlock(chain1TipName, &reorgSpend, func(b *wire.MsgBlock) {
			bytesToMaxSize := maxBlockSize - b.SerializeSize() - 3
			sizePadScript := repeatOpcode(0x00, bytesToMaxSize)
			replaceSpendScript(sizePadScript)(b)
		})
		g.assertTipBlockSize(maxBlockSize)
		g.saveTipCoinbaseOut()
		testInstances = append(testInstances, acceptBlock(g.tipName,
			g.tip, true, false))

//使用下一个可用的可消费输出。先用完任何
//剩余的可消费的输出已经进入
//输出切片，然后从堆栈中弹出。
		if spendableOutOffset+1+i < int32(len(outs)) {
			reorgSpend = *outs[spendableOutOffset+1+i]
		} else {
			reorgSpend = g.oldestCoinbaseOut()
		}
	}
	tests = append(tests, testInstances)

//创建具有相同长度的侧链。
//
//…->BR0->-布鲁
//\->bralt0->>勃拉特
	g.setTip(reorgStartBlockName)
	testInstances = nil
	chain2TipName := g.tipName
	for i := uint16(0); i < numLargeReorgBlocks; i++ {
		chain2TipName = fmt.Sprintf("bralt%d", i)
		g.nextBlock(chain2TipName, nil)
		testInstances = append(testInstances, acceptBlock(g.tipName,
			g.tip, false, false))
	}
	testInstances = append(testInstances, expectTipBlock(chain1TipName,
		g.blocksByName[chain1TipName]))
	tests = append(tests, testInstances)

//将侧链延伸一条，以强制执行大型REORG。
//
//…->bralt0->->bralt->bralt+1
//
	g.nextBlock(fmt.Sprintf("bralt%d", g.tipHeight+1), nil)
	chain2TipName = g.tipName
	accepted()

//
//
//
//
	g.setTip(chain1TipName)
	g.nextBlock(fmt.Sprintf("br%d", g.tipHeight+1), nil)
	chain1TipName = g.tipName
	acceptedToSideChainWithExpectedTip(chain2TipName)

	g.nextBlock(fmt.Sprintf("br%d", g.tipHeight+2), nil)
	chain1TipName = g.tipName
	accepted()

	return tests, nil
}
