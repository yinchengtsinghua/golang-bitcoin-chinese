
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
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

const (
//MaxTimeOffsetSeconds是块时间的最大秒数
//允许超前于当前时间。这是目前的2
//小时。
	MaxTimeOffsetSeconds = 2 * 60 * 60

//MinCoinBaseScriptlen是CoinBase脚本的最小长度。
	MinCoinbaseScriptLen = 2

//MaxCoinBaseScriptlen是CoinBase脚本的最大长度。
	MaxCoinbaseScriptLen = 100

//MediantimeBlocks是应为
//用于计算用于验证块时间戳的中间时间。
	medianTimeBlocks = 11

//serializedheightversion是更改块的块版本
//以序列化块高度开始的CoinBase。
	serializedHeightVersion = 2

//基准补贴是开采区块的起始补贴金额。这个
//值在每个SublyAlvingInterval块中减半。
	baseSubsidy = 50 * btcutil.SatoshiPerBitcoin
)

var (
//zero hash是chainhash.hash的零值，定义为
//包级变量，以避免创建新实例
//每次需要检查时。
	zeroHash chainhash.Hash

//block91842hash是违反规则的两个节点之一
//在Bip0030中提出。它被定义为包级变量
//避免每次需要检查时都需要创建新实例。
	block91842Hash = newHashFromStr("00000000000a4d0a398161ffc163c503763b1f4360639393e0e4c8e300e0caec")

//block91880hash是违反规则的两个节点之一
//在Bip0030中提出。它被定义为包级变量
//避免每次需要检查时都需要创建新实例。
	block91880Hash = newHashFromStr("00000000000743f190a18c5577a3c2d2a1f610ae9601ac046a38084ccb7cd721")
)

//IsNullOutPoint确定是否为上一个事务输出点
//被设置。
func isNullOutpoint(outpoint *wire.OutPoint) bool {
	if outpoint.Index == math.MaxUint32 && outpoint.Hash == zeroHash {
		return true
	}
	return false
}

//ShouldHaveserializedBlockHeight确定块是否应具有
//嵌入在其脚本sig中的序列化块高度
//CoinBase交易。判断基于块中的块版本
//标题。版本为2及以上的块满足此条件。见BIP034
//更多信息。
func ShouldHaveSerializedBlockHeight(header *wire.BlockHeader) bool {
	return header.Version >= serializedHeightVersion
}

//iscoinbasetx确定事务是否为coinbase。硬币库
//是由没有输入的矿工创建的特殊事务。这是
//在块链中由一个具有
//以前的输出事务索引设置为最大值，同时
//零散列。
//
//此函数与iscoinBase的区别在于，它与原始线一起使用
//事务，而不是更高级别的UTIL事务。
func IsCoinBaseTx(msgTx *wire.MsgTx) bool {
//硬币库只能有一个交易输入。
	if len(msgTx.TxIn) != 1 {
		return false
	}

//硬币底座的先前输出必须具有最大值索引，并且
//零散列。
	prevOut := &msgTx.TxIn[0].PreviousOutPoint
	if prevOut.Index != math.MaxUint32 || prevOut.Hash != zeroHash {
		return false
	}

	return true
}

//iscoinBase确定事务是否为coinBase。硬币库
//是由没有输入的矿工创建的特殊事务。这是
//在块链中由一个具有
//以前的输出事务索引设置为最大值，同时
//零散列。
//
//此函数与iscoinbasetx的不同之处在于，它使用更高的
//级别Util事务，而不是原始线事务。
func IsCoinBase(tx *btcutil.Tx) bool {
	return IsCoinBaseTx(tx.MsgTx())
}

//SequenceLockActive确定事务的序列锁是否
//满足，表示给定事务的所有输入都已达到
//高度或时间足以满足它们的相对锁定时间成熟度。
func SequenceLockActive(sequenceLock *SequenceLock, blockHeight int32,
	medianTimePast time.Time) bool {

//如果秒或高度相对锁定时间尚未
//达到，则根据其
//序列锁。
	if sequenceLock.Seconds >= medianTimePast.Unix() ||
		sequenceLock.BlockHeight >= blockHeight {
		return false
	}

	return true
}

//IsFinalizedTransaction确定事务是否已完成。
func IsFinalizedTransaction(tx *btcutil.Tx, blockHeight int32, blockTime time.Time) bool {
	msgTx := tx.MsgTx()

//锁定时间为零表示事务已完成。
	lockTime := msgTx.LockTime
	if lockTime == 0 {
		return true
	}

//事务的锁定时间字段要么是块高度，
//哪个事务已完成或时间戳取决于
//值在txscript.lockTimeThreshold之前。当它在
//门槛是一个街区的高度。
	blockTimeOrHeight := int64(0)
	if lockTime < txscript.LockTimeThreshold {
		blockTimeOrHeight = int64(blockHeight)
	} else {
		blockTimeOrHeight = blockTime.Unix()
	}
	if int64(lockTime) < blockTimeOrHeight {
		return true
	}

//此时，事务的锁定时间还没有发生，但是
//如果序列号
//对于所有事务输入都是最大化的。
	for _, txIn := range msgTx.TxIn {
		if txIn.Sequence != math.MaxUint32 {
			return false
		}
	}
	return true
}

//isbip0030node返回传递的节点是否表示
//两个块违反了阻止交易
//覆盖旧的。
func isBIP0030Node(node *blockNode) bool {
	if node.height == 91842 && node.hash.IsEqual(block91842Hash) {
		return true
	}

	if node.height == 91880 && node.hash.IsEqual(block91880Hash) {
		return true
	}

	return false
}

//CalcBlockSubmity在规定高度返回一个区块的补贴金额
//
//新生成的区块奖励以及验证区块的CoinBase
//具有预期值。
//
//补贴是每一个补贴减免间隔块减半。数学上的
//这是：基本补贴/2^（身高/补贴教育间隔）
//
//在主网络的目标块生成速率下，这是
//大约每4年。
func CalcBlockSubsidy(height int32, chainParams *chaincfg.Params) int64 {
	if chainParams.SubsidyReductionInterval == 0 {
		return baseSubsidy
	}

//相当于：基本补贴/2^（高度/补贴间隔）
	return baseSubsidy >> uint(height/chainParams.SubsidyReductionInterval)
}

//CheckTransactionSanity对以下事务执行一些初步检查：
//确保它是健全的。这些检查是上下文无关的。
func CheckTransactionSanity(tx *btcutil.Tx) error {
//一个事务必须至少有一个输入。
	msgTx := tx.MsgTx()
	if len(msgTx.TxIn) == 0 {
		return ruleError(ErrNoTxInputs, "transaction has no inputs")
	}

//一个事务必须至少有一个输出。
	if len(msgTx.TxOut) == 0 {
		return ruleError(ErrNoTxOutputs, "transaction has no outputs")
	}

//在以下情况下，事务不能超过允许的最大块负载：
//序列化。
	serializedTxSize := tx.MsgTx().SerializeSizeStripped()
	if serializedTxSize > MaxBlockBaseSize {
		str := fmt.Sprintf("serialized transaction is too big - got "+
			"%d, max %d", serializedTxSize, MaxBlockBaseSize)
		return ruleError(ErrTxTooBig, str)
	}

//确保交易金额在范围内。每笔交易
//输出不能为负或大于每个
//交易。此外，所有输出的总和必须遵守相同的
//限制。交易中的所有金额都以已知的单位值为单位。
//作为一个寿司。一比特币是由
//饱和比特币常数。
	var totalSatoshi int64
	for _, txOut := range msgTx.TxOut {
		satoshi := txOut.Value
		if satoshi < 0 {
			str := fmt.Sprintf("transaction output has negative "+
				"value of %v", satoshi)
			return ruleError(ErrBadTxOutValue, str)
		}
		if satoshi > btcutil.MaxSatoshi {
			str := fmt.Sprintf("transaction output value of %v is "+
				"higher than max allowed value of %v", satoshi,
				btcutil.MaxSatoshi)
			return ruleError(ErrBadTxOutValue, str)
		}

//two的补码int64溢出保证任何溢出
//检测并报告。这对比特币来说是不可能的，但是
//如果ALT增加了总的货币供应量，也许是可能的。
		totalSatoshi += satoshi
		if totalSatoshi < 0 {
			str := fmt.Sprintf("total value of all transaction "+
				"outputs exceeds max allowed value of %v",
				btcutil.MaxSatoshi)
			return ruleError(ErrBadTxOutValue, str)
		}
		if totalSatoshi > btcutil.MaxSatoshi {
			str := fmt.Sprintf("total value of all transaction "+
				"outputs is %v which is higher than max "+
				"allowed value of %v", totalSatoshi,
				btcutil.MaxSatoshi)
			return ruleError(ErrBadTxOutValue, str)
		}
	}

//检查重复的事务输入。
	existingTxOut := make(map[wire.OutPoint]struct{})
	for _, txIn := range msgTx.TxIn {
		if _, exists := existingTxOut[txIn.PreviousOutPoint]; exists {
			return ruleError(ErrDuplicateTxInputs, "transaction "+
				"contains duplicate inputs")
		}
		existingTxOut[txIn.PreviousOutPoint] = struct{}{}
	}

//CoinBase脚本长度必须介于最小和最大长度之间。
	if IsCoinBase(tx) {
		slen := len(msgTx.TxIn[0].SignatureScript)
		if slen < MinCoinbaseScriptLen || slen > MaxCoinbaseScriptLen {
			str := fmt.Sprintf("coinbase transaction script length "+
				"of %d is out of range (min: %d, max: %d)",
				slen, MinCoinbaseScriptLen, MaxCoinbaseScriptLen)
			return ruleError(ErrBadCoinbaseScriptLen, str)
		}
	} else {
//以前的事务输出被此的输入引用
//事务不能为空。
		for _, txIn := range msgTx.TxIn {
			if isNullOutpoint(&txIn.PreviousOutPoint) {
				return ruleError(ErrBadTxInput, "transaction "+
					"input refers to previous output that "+
					"is null")
			}
		}
	}

	return nil
}

//校对工作确保指示目标的块头位
//困难在最小/最大范围内，并且块哈希小于
//目标难度如要求。
//
//这些标志按如下方式修改此函数的行为：
//-bfnopowcheck：检查以确保块哈希小于目标值
//没有执行困难。
func checkProofOfWork(header *wire.BlockHeader, powLimit *big.Int, flags BehaviorFlags) error {
//目标难度必须大于零。
	target := CompactToBig(header.Bits)
	if target.Sign() <= 0 {
		str := fmt.Sprintf("block target difficulty of %064x is too low",
			target)
		return ruleError(ErrUnexpectedDifficulty, str)
	}

//目标难度必须小于允许的最大值。
	if target.Cmp(powLimit) > 0 {
		str := fmt.Sprintf("block target difficulty of %064x is "+
			"higher than max of %064x", target, powLimit)
		return ruleError(ErrUnexpectedDifficulty, str)
	}

//块哈希必须小于声明的目标，除非标志
//为避免工作证明，设置了检查。
	if flags&BFNoPoWCheck != BFNoPoWCheck {
//块哈希必须小于声明的目标。
		hash := header.BlockHash()
		hashNum := HashToBig(&hash)
		if hashNum.Cmp(target) > 0 {
			str := fmt.Sprintf("block hash of %064x is higher than "+
				"expected max of %064x", hashNum, target)
			return ruleError(ErrHighHash, str)
		}
	}

	return nil
}

//校对工作确保指示目标的块头位
//困难在最小/最大范围内，并且块哈希小于
//目标难度如要求。
func CheckProofOfWork(block *btcutil.Block, powLimit *big.Int) error {
	return checkProofOfWork(&block.MsgBlock().Header, powLimit, BFNone)
}

//CountSigops返回所有事务的签名操作数
//在提供的事务中输入和输出脚本。这使用了
//更快但不精确的签名操作计数机制
//TXScript。
func CountSigOps(tx *btcutil.Tx) int {
	msgTx := tx.MsgTx()

//
//输入。
	totalSigOps := 0
	for _, txIn := range msgTx.TxIn {
		numSigOps := txscript.GetSigOpCount(txIn.SignatureScript)
		totalSigOps += numSigOps
	}

//累积所有事务中的签名操作数
//输出。
	for _, txOut := range msgTx.TxOut {
		numSigOps := txscript.GetSigOpCount(txOut.PkScript)
		totalSigOps += numSigOps
	}

	return totalSigOps
}

//
//属于“按脚本付费”哈希类型的事务。这使用了
//来自脚本引擎的精确签名操作计数机制
//需要访问输入事务脚本。
func CountP2SHSigOps(tx *btcutil.Tx, isCoinBaseTx bool, utxoView *UtxoViewpoint) (int, error) {
//CoinBase事务没有有趣的输入。
	if isCoinBaseTx {
		return 0, nil
	}

//累积所有事务中的签名操作数
//输入。
	msgTx := tx.MsgTx()
	totalSigOps := 0
	for txInIndex, txIn := range msgTx.TxIn {
//确保引用的输入事务可用。
		utxo := utxoView.LookupEntry(txIn.PreviousOutPoint)
		if utxo == nil || utxo.IsSpent() {
			str := fmt.Sprintf("output %v referenced from "+
				"transaction %s:%d either does not exist or "+
				"has already been spent", txIn.PreviousOutPoint,
				tx.Hash(), txInIndex)
			return 0, ruleError(ErrMissingTxOut, str)
		}

//我们只对付费脚本散列类型感兴趣，所以跳过
//如果不是一个输入。
		pkScript := utxo.PkScript()
		if !txscript.IsPayToScriptHash(pkScript) {
			continue
		}

//统计
//引用的公钥脚本。
		sigScript := txIn.SignatureScript
		numSigOps := txscript.GetPreciseSigOpCount(sigScript, pkScript,
			true)

//
//溢出。
		lastSigOps := totalSigOps
		totalSigOps += numSigOps
		if totalSigOps < lastSigOps {
			str := fmt.Sprintf("the public key script from output "+
				"%v contains too many signature operations - "+
				"overflow", txIn.PreviousOutPoint)
			return 0, ruleError(ErrTooManySigOps, str)
		}
	}

	return totalSigOps, nil
}

//checkblockheadersanitiy对块头执行一些初步检查
//在继续处理之前，请确保它是正常的。这些支票是
//上下文无关。
//
//标志不会直接修改此函数的行为，但是它们
//需要通过检查工作证明。
func checkBlockHeaderSanity(header *wire.BlockHeader, powLimit *big.Int, timeSource MedianTimeSource, flags BehaviorFlags) error {
//确保块头中的工作位证明在最小/最大范围内。
//块散列值小于
//位。
	err := checkProofOfWork(header, powLimit, flags)
	if err != nil {
		return err
	}

//块时间戳的精度不得超过1秒。
//此检查是必需的，因为Go Time.Time值支持
//纳秒精度，而共识规则只适用于
//秒和它是更好的标准去处理时间值。
//而不是在任何地方转换为秒。
	if !header.Timestamp.Equal(time.Unix(header.Timestamp.Unix(), 0)) {
		str := fmt.Sprintf("block timestamp of %v has a higher "+
			"precision than one second", header.Timestamp)
		return ruleError(ErrInvalidTime, str)
	}

//确保未来的阻塞时间不太远。
	maxTimestamp := timeSource.AdjustedTime().Add(time.Second *
		MaxTimeOffsetSeconds)
	if header.Timestamp.After(maxTimestamp) {
		str := fmt.Sprintf("block timestamp of %v is too far in the "+
			"future", header.Timestamp)
		return ruleError(ErrTimeTooNew, str)
	}

	return nil
}

//checkblocksanity对一个块执行一些初步检查，以确保
//在继续块处理之前保持清醒。这些检查是上下文无关的。
//
//标志不会直接修改此函数的行为，但是它们
//需要传递给checkblockheadersanitiy。
func checkBlockSanity(block *btcutil.Block, powLimit *big.Int, timeSource MedianTimeSource, flags BehaviorFlags) error {
	msgBlock := block.MsgBlock()
	header := &msgBlock.Header
	err := checkBlockHeaderSanity(header, powLimit, timeSource, flags)
	if err != nil {
		return err
	}

//一个块必须至少有一个事务。
	numTx := len(msgBlock.Transactions)
	if numTx == 0 {
		return ruleError(ErrNoTransactions, "block does not contain "+
			"any transactions")
	}

//块的事务数不得超过最大块有效负载或
//否则肯定超过了重量限制。
	if numTx > MaxBlockBaseSize {
		str := fmt.Sprintf("block contains too many transactions - "+
			"got %d, max %d", numTx, MaxBlockBaseSize)
		return ruleError(ErrBlockTooBig, str)
	}

//块不能超过允许的最大块负载，当
//序列化。
	serializedSize := msgBlock.SerializeSizeStripped()
	if serializedSize > MaxBlockBaseSize {
		str := fmt.Sprintf("serialized block is too big - got %d, "+
			"max %d", serializedSize, MaxBlockBaseSize)
		return ruleError(ErrBlockTooBig, str)
	}

//
	transactions := block.Transactions()
	if !IsCoinBase(transactions[0]) {
		return ruleError(ErrFirstTxNotCoinbase, "first transaction in "+
			"block is not a coinbase")
	}

//一个块不能有多个coinbase。
	for i, tx := range transactions[1:] {
		if IsCoinBase(tx) {
			str := fmt.Sprintf("block contains second coinbase at "+
				"index %d", i+1)
			return ruleError(ErrMultipleCoinbases, str)
		}
	}

//对每笔交易进行初步检查，以确保
//
	for _, tx := range transactions {
		err := CheckTransactionSanity(tx)
		if err != nil {
			return err
		}
	}

//构建merkle树并确保计算的merkle根与
//块头中的条目。这还具有缓存所有
//块中的事务哈希数，以加速将来的哈希数
//
//经过以下检查，但没有理由不检查
//梅克尔根匹配这里。
	merkles := BuildMerkleTreeStore(block.Transactions(), false)
	calculatedMerkleRoot := merkles[len(merkles)-1]
	if !header.MerkleRoot.IsEqual(calculatedMerkleRoot) {
		str := fmt.Sprintf("block merkle root is invalid - block "+
			"header indicates %v, but calculated value is %v",
			header.MerkleRoot, calculatedMerkleRoot)
		return ruleError(ErrBadMerkleRoot, str)
	}

//检查重复的交易记录。这张支票比较快
//由于生成
//上面是梅克尔树。
	existingTxHashes := make(map[chainhash.Hash]struct{})
	for _, tx := range transactions {
		hash := tx.Hash()
		if _, exists := existingTxHashes[*hash]; exists {
			str := fmt.Sprintf("block contains duplicate "+
				"transaction %v", hash)
			return ruleError(ErrDuplicateTx, str)
		}
		existingTxHashes[*hash] = struct{}{}
	}

//签名操作数必须小于最大值
//允许每个块。
	totalSigOps := 0
	for _, tx := range transactions {
//我们可能会溢出蓄能器，因此检查
//溢出。
		lastSigOps := totalSigOps
		totalSigOps += (CountSigOps(tx) * WitnessScaleFactor)
		if totalSigOps < lastSigOps || totalSigOps > MaxBlockSigOpsCost {
			str := fmt.Sprintf("block contains too many signature "+
				"operations - got %v, max %v", totalSigOps,
				MaxBlockSigOpsCost)
			return ruleError(ErrTooManySigOps, str)
		}
	}

	return nil
}

//checkblocksanity对一个块执行一些初步检查，以确保
//在继续块处理之前保持清醒。这些检查是上下文无关的。
func CheckBlockSanity(block *btcutil.Block, powLimit *big.Int, timeSource MedianTimeSource) error {
	return checkBlockSanity(block, powLimit, timeSource, BFNone)
}

//
//CoinBase事务的脚本签名。CoinBase高度仅存在于
//版本2或更高版本的块。这是作为BIP0034的一部分添加的。
func ExtractCoinbaseHeight(coinbaseTx *btcutil.Tx) (int32, error) {
	sigScript := coinbaseTx.MsgTx().TxIn[0].SignatureScript
	if len(sigScript) < 1 {
		str := "the coinbase signature script for blocks of " +
			"version %d or greater must start with the " +
			"length of the serialized block height"
		str = fmt.Sprintf(str, serializedHeightVersion)
		return 0, ruleError(ErrMissingCoinbaseHeight, str)
	}

//当块高度是用
//作为一个字节。
	opcode := int(sigScript[0])
	if opcode == txscript.OP_0 {
		return 0, nil
	}
	if opcode >= txscript.OP_1 && opcode <= txscript.OP_16 {
		return int32(opcode - (txscript.OP_1 - 1)), nil
	}

//否则，操作码是以下字节的长度，
//按块高度编码。
	serializedLen := int(sigScript[0])
	if len(sigScript[1:]) < serializedLen {
		str := "the coinbase signature script for blocks of " +
			"version %d or greater must start with the " +
			"serialized block height"
		str = fmt.Sprintf(str, serializedLen)
		return 0, ruleError(ErrMissingCoinbaseHeight, str)
	}

	serializedHeightBytes := make([]byte, 8)
	copy(serializedHeightBytes, sigScript[1:serializedLen+1])
	serializedHeight := binary.LittleEndian.Uint64(serializedHeightBytes)

	return int32(serializedHeight), nil
}

//checkserializedheight检查传递的
//事务以Wantheight的序列化块高度开始。
func checkSerializedHeight(coinbaseTx *btcutil.Tx, wantHeight int32) error {
	serializedHeight, err := ExtractCoinbaseHeight(coinbaseTx)
	if err != nil {
		return err
	}

	if serializedHeight != wantHeight {
		str := fmt.Sprintf("the coinbase signature script serialized "+
			"block height is %d when %d was expected",
			serializedHeight, wantHeight)
		return ruleError(ErrBadCoinbaseHeight, str)
	}
	return nil
}

//CheckBlockHeaderContext对块头执行多个验证检查
//这取决于它在区块链中的位置。
//
//这些标志按如下方式修改此函数的行为：
//-bfastadd:除涉及将标题与
//不执行检查点。
//
//必须在保持链状态锁的情况下调用此函数（用于写入）。
func (b *BlockChain) checkBlockHeaderContext(header *wire.BlockHeader, prevNode *blockNode, flags BehaviorFlags) error {
	fastAdd := flags&BFFastAdd == BFFastAdd
	if !fastAdd {
//确保块头中指定的难度匹配
//
//难以重新定位规则。
		expectedDifficulty, err := b.calcNextRequiredDifficulty(prevNode,
			header.Timestamp)
		if err != nil {
			return err
		}
		blockDifficulty := header.Bits
		if blockDifficulty != expectedDifficulty {
			str := "block difficulty of %d is not the expected value of %d"
			str = fmt.Sprintf(str, blockDifficulty, expectedDifficulty)
			return ruleError(ErrUnexpectedDifficulty, str)
		}

//确保块头的时间戳在
//最后几个块的中间时间（median time blocks）。
		medianTime := prevNode.CalcPastMedianTime()
		if !header.Timestamp.After(medianTime) {
			str := "block timestamp of %v is not after expected %v"
			str = fmt.Sprintf(str, header.Timestamp, medianTime)
			return ruleError(ErrTimeTooOld, str)
		}
	}

//此块的高度比引用的上一块高一倍
//块。
	blockHeight := prevNode.height + 1

//确保链匹配到预定的检查点。
	blockHash := header.BlockHash()
	if !b.verifyCheckpoint(blockHeight, &blockHash) {
		str := fmt.Sprintf("block at height %d does not match "+
			"checkpoint hash", blockHeight)
		return ruleError(ErrBadCheckpoint, str)
	}

//找到上一个检查点，并防止块分叉主
//前面有链条。这样可以防止存储新的，否则是有效的，
//由可能更容易
//很难，因此可以用来浪费缓存和磁盘空间。
	checkpointNode, err := b.findPreviousCheckpoint()
	if err != nil {
		return err
	}
	if checkpointNode != nil && blockHeight < checkpointNode.height {
		str := fmt.Sprintf("block at height %d forks the main chain "+
			"before the previous checkpoint at height %d",
			blockHeight, checkpointNode.height)
		return ruleError(ErrForkTooOld, str)
	}

//在大多数网络中拒绝过时的块版本
//已经升级。最初是由BIP0034投票决定的，
//Bip0065和Bip0066。
	params := b.chainParams
	if header.Version < 2 && blockHeight >= params.BIP0034Height ||
		header.Version < 3 && blockHeight >= params.BIP0066Height ||
		header.Version < 4 && blockHeight >= params.BIP0065Height {

		str := "new blocks with version %d are no longer valid"
		str = fmt.Sprintf(str, header.Version)
		return ruleError(ErrBlockVersionTooOld, str)
	}

	return nil
}

//checkblockContext对依赖于
//在区块链内的位置。
//
//这些标志按如下方式修改此函数的行为：
//-bfastadd:不检查交易是否已完成
//并且不执行稍微昂贵的bip0034验证。
//
//这些标志还传递给checkblockheadercontext。查看其文档
//以了解标志如何修改其行为。
//
//必须在保持链状态锁的情况下调用此函数（用于写入）。
func (b *BlockChain) checkBlockContext(block *btcutil.Block, prevNode *blockNode, flags BehaviorFlags) error {
//执行所有与块头相关的验证检查。
	header := &block.MsgBlock().Header
	err := b.checkBlockHeaderContext(header, prevNode, flags)
	if err != nil {
		return err
	}

	fastAdd := flags&BFFastAdd == BFFastAdd
	if !fastAdd {
//获取已部署的csv软分叉的最新状态
//以正确保护新的验证行为
//当前的BIP 9版本位状态。
		csvState, err := b.deploymentState(prevNode, chaincfg.DeploymentCSV)
		if err != nil {
			return err
		}

//
//使用上一个块的当前中值时间
//所有基于锁定时间的检查的时间戳。
		blockTime := header.Timestamp
		if csvState == ThresholdActive {
			blockTime = prevNode.CalcPastMedianTime()
		}

//此块的高度比引用的高一倍
//前一个块。
		blockHeight := prevNode.height + 1

//确保块中的所有事务都已完成。
		for _, tx := range block.Transactions() {
			if !IsFinalizedTransaction(tx, blockHeight,
				blockTime) {

				str := fmt.Sprintf("block contains unfinalized "+
					"transaction %v", tx.Hash())
				return ruleError(ErrUnfinalizedTx, str)
			}
		}

//确保coinbase以序列化块高度开始
//阻止其版本为SerializedEightVersion或更高版本
//一旦大部分网络升级。这是一部分
//BIP034
		if ShouldHaveSerializedBlockHeight(header) &&
			blockHeight >= b.chainParams.BIP0034Height {

			coinbaseTx := block.Transactions()[0]
			err := checkSerializedHeight(coinbaseTx, blockHeight)
			if err != nil {
				return err
			}
		}

//查询Segwit软分叉的版本位状态
//
//执行所有新规则。
		segwitState, err := b.deploymentState(prevNode,
			chaincfg.DeploymentSegwit)
		if err != nil {
			return err
		}

//如果Segwit是活跃的，那么我们需要完全验证
//新的证人承诺遵守规则。
		if segwitState == ThresholdActive {
//
//块。这涉及到断言
//
//merkle根匹配所有
//块内事务的wtxid。在
//此外，还针对
//Coinbase的见证堆栈。
			if err := ValidateWitnessCommitment(block); err != nil {
				return err
			}

//一旦证人承诺，立即证人和签名
//运营成本已经确认，我们最终可以断言
//块的重量不超过电流
//共识参数。
			blockWeight := GetBlockWeight(block)
			if blockWeight > MaxBlockWeight {
				str := fmt.Sprintf("block's weight metric is "+
					"too high - got %v, max %v",
					blockWeight, MaxBlockWeight)
				return ruleError(ErrBlockWeightTooHigh, str)
			}
		}
	}

	return nil
}

//checkbip0030确保块不包含重复的事务
//“覆盖”未完全使用的旧事务。这可以防止
//攻击一个coinbase及其所有相关事务可能
//复制以有效地将覆盖的事务还原为单个
//从而使他们容易受到双重消费的影响。
//
//
//https://github.com/bitcoin/bips/blob/master/bip-0030.mediawiki和
//http://r6.ca/blog/20120206t005236z.html.
//
//必须在保持链状态锁的情况下调用此函数（用于读取）。
func (b *BlockChain) checkBIP0030(node *blockNode, block *btcutil.Block, view *UtxoViewpoint) error {
//获取此块中所有事务输出的utxos。
//通常，任何输出都不会有任何utxo。
	fetchSet := make(map[wire.OutPoint]struct{})
	for _, tx := range block.Transactions() {
		prevOut := wire.OutPoint{Hash: *tx.Hash()}
		for txOutIdx := range tx.MsgTx().TxOut {
			prevOut.Index = uint32(txOutIdx)
			fetchSet[prevOut] = struct{}{}
		}
	}
	err := view.fetchUtxos(b.db, fetchSet)
	if err != nil {
		return err
	}

//只有在上一个事务
//完全用完了。
	for outpoint := range fetchSet {
		utxo := view.LookupEntry(outpoint)
		if utxo != nil && !utxo.IsSpent() {
			str := fmt.Sprintf("tried to overwrite transaction %v "+
				"at block height %d that is not fully spent",
				outpoint.Hash, utxo.BlockHeight())
			return ruleError(ErrOverwriteTx, str)
		}
	}

	return nil
}

//checkTransactionOuts对
//确保它们有效的事务。一些检查的示例
//包括验证所有输入是否存在，确保CoinBase调味料
//满足要求，检测双倍支出，验证所有价值和费用
//在合法范围内，总产量不超过投入
//金额，并核实签名以证明挥霍者是
//比特币，因此允许使用它们。当它检查输入时，
//它还计算交易的总费用并返回该值。
//
//注意：该事务必须已通过
//在调用此函数之前检查TransactionSanity函数。
func CheckTransactionInputs(tx *btcutil.Tx, txHeight int32, utxoView *UtxoViewpoint, chainParams *chaincfg.Params) (int64, error) {
//CoinBase事务没有输入。
	if IsCoinBase(tx) {
		return 0, nil
	}

	txHash := tx.Hash()
	var totalSatoshiIn int64
	for txInIndex, txIn := range tx.MsgTx().TxIn {
//确保引用的输入事务可用。
		utxo := utxoView.LookupEntry(txIn.PreviousOutPoint)
		if utxo == nil || utxo.IsSpent() {
			str := fmt.Sprintf("output %v referenced from "+
				"transaction %s:%d either does not exist or "+
				"has already been spent", txIn.PreviousOutPoint,
				tx.Hash(), txInIndex)
			return 0, ruleError(ErrMissingTxOut, str)
		}

//确保交易没有花费没有
//但达到了所需的货币基础成熟度。
		if utxo.IsCoinBase() {
			originHeight := utxo.BlockHeight()
			blocksSincePrev := txHeight - originHeight
			coinbaseMaturity := int32(chainParams.CoinbaseMaturity)
			if blocksSincePrev < coinbaseMaturity {
				str := fmt.Sprintf("tried to spend coinbase "+
					"transaction output %v from height %v "+
					"at height %v before required maturity "+
					"of %v blocks", txIn.PreviousOutPoint,
					originHeight, txHeight,
					coinbaseMaturity)
				return 0, ruleError(ErrImmatureSpend, str)
			}
		}

//确保交易金额在范围内。每一个
//输入事务的输出值不能为负数
//或超过每个事务允许的最大值。所有金额
//
//比特币是由
//饱和比特币常数。
		originTxSatoshi := utxo.Amount()
		if originTxSatoshi < 0 {
			str := fmt.Sprintf("transaction output has negative "+
				"value of %v", btcutil.Amount(originTxSatoshi))
			return 0, ruleError(ErrBadTxOutValue, str)
		}
		if originTxSatoshi > btcutil.MaxSatoshi {
			str := fmt.Sprintf("transaction output value of %v is "+
				"higher than max allowed value of %v",
				btcutil.Amount(originTxSatoshi),
				btcutil.MaxSatoshi)
			return 0, ruleError(ErrBadTxOutValue, str)
		}

//所有输出的总和不得超过最大值
//允许每个事务。此外，我们还可能溢出
//这样可以检查蓄能器是否溢出。
		lastSatoshiIn := totalSatoshiIn
		totalSatoshiIn += originTxSatoshi
		if totalSatoshiIn < lastSatoshiIn ||
			totalSatoshiIn > btcutil.MaxSatoshi {
			str := fmt.Sprintf("total value of all transaction "+
				"inputs is %v which is higher than max "+
				"allowed value of %v", totalSatoshiIn,
				btcutil.MaxSatoshi)
			return 0, ruleError(ErrBadTxOutValue, str)
		}
	}

//计算此交易记录的总输出金额。它是安全的
//忽略溢出和超出范围的错误，因为这些错误
//条件可能已经被checkTransactionSanity捕获。
	var totalSatoshiOut int64
	for _, txOut := range tx.MsgTx().TxOut {
		totalSatoshiOut += txOut.Value
	}

//确保事务支出不超过其输入。
	if totalSatoshiIn < totalSatoshiOut {
		str := fmt.Sprintf("total value of all transaction inputs for "+
			"transaction %v is %v which is less than the amount "+
			"spent of %v", txHash, totalSatoshiIn, totalSatoshiOut)
		return 0, ruleError(ErrSpendTooHigh, str)
	}

//注：比特币检查这里的交易费用是否小于0，但是
//是不可能的情况，因为上面的检查确保
//输入大于等于输出。
	txFeeInSatoshi := totalSatoshiIn - totalSatoshiOut
	return txFeeInSatoshi, nil
}

//CheckConnectBlock执行多个检查以确认连接
//块到由传递的视图表示的链不会违反任何规则。
//此外，将更新传递的视图以使用所有引用的
//输出并添加由块创建的所有新utxo。因此，视图将
//表示链的状态，就好像块是实际连接的，
//因此，视图的最佳哈希也会更新为传递的块。
//
//执行的一些检查的示例是确保连接块
//不会导致旧事务的任何重复事务哈希
//尚未完全消费，双倍消费，超过允许的最大值
//每个块的签名操作，与预期的
//冻结补贴，或交易脚本验证失败。
//
//checkConnectBlockTemplate函数使用此函数执行
//它的大部分工作。唯一的区别是这个函数接受一个节点
//可能需要或可能不需要重组才能将其连接到主链
//而checkConnectBlockTemplate创建一个新节点，该节点
//
//用那个节点。
//
//必须在保持链状态锁的情况下调用此函数（用于写入）。
func (b *BlockChain) checkConnectBlock(node *blockNode, block *btcutil.Block, view *UtxoViewpoint, stxos *[]SpentTxOut) error {
//如果侧链块最终出现在数据库中，则调用
//如果是以前的版本，则应在此处执行checkblocksanity
//允许一个不再有效的块。然而，自从
//实现目前只使用侧链块的内存，
//目前不需要。

//创世纪块的硬币库是不可消费的，所以只要返回
//现在出错了。
	if node.hash.IsEqual(b.chainParams.GenesisHash) {
		str := "the coinbase for the genesis block is not spendable"
		return ruleError(ErrMissingTxOut, str)
	}

//确保视图适用于正在检查的节点。
	parentHash := &block.MsgBlock().Header.PrevBlock
	if !view.BestHash().IsEqual(parentHash) {
		return AssertError(fmt.Sprintf("inconsistent view when "+
			"checking block connection: best hash is %v instead "+
			"of expected %v", view.BestHash(), parentHash))
	}

//bip0030添加了一个规则以防止包含重复的块
//
//
//
//链中有两个块违反了此规则，因此
//
//用于确定此块是否是必须
//跳过。
//
//
//由于其要求将块高度包括在
//这样就不可能再创建交易了
//“覆盖”旧的。因此，只有在
//BIP0034尚未激活。这是一个有用的优化，因为
//bip0030检查很昂贵，因为它涉及大量的缓存未命中
//乌托索
	if !isBIP0030Node(node) && (node.height < b.chainParams.BIP0034Height) {
		err := b.checkBIP0030(node, block, view)
		if err != nil {
			return err
		}
	}

//加载所有事务的输入所引用的所有utxo
//在数据库的utxo视图中不存在。
//
//这些utxo条目是验证诸如
//事务输入、计算付薪到脚本散列和脚本。
	err := view.fetchInputUtxos(b.db, block)
	if err != nil {
		return err
	}

//bip0016描述了被认为是
//“标准”类型。此BIP的规则仅适用于事务
//在txscript.bip16activation定义的时间戳之后。见
//
	enforceBIP0016 := node.timestamp >= txscript.Bip16Activation.Unix()

//查询Segwit软分叉的版本位状态
//部署。如果Segwit激活，我们将切换到强制
//新规则。
	segwitState, err := b.deploymentState(node.parent, chaincfg.DeploymentSegwit)
	if err != nil {
		return err
	}
	enforceSegWit := segwitState == ThresholdActive

//签名操作数必须小于最大值
//允许每个块。注意，初步的健全性检查
//块中也包含类似于此支票的支票，但此支票
//扩展计数以包括精确的付薪脚本哈希计数
//每个输入事务公钥中的签名操作
//脚本。
	transactions := block.Transactions()
	totalSigOpCost := 0
	for i, tx := range transactions {
//自从第一个（并且只有第一个）交易
//已经被证实是CoinBase交易，
//使用i==0优化标志
//countp2shsigops用于确定事务是否为
//CoinBase事务，而不是必须执行
//再次检查全部CoinBase。
		sigOpCost, err := GetSigOpCost(tx, i == 0, view, enforceBIP0016,
			enforceSegWit)
		if err != nil {
			return err
		}

//检查是否溢出或超出限制。我们必须这样做
//这在每个循环迭代中都可以避免溢出。
		lastSigOpCost := totalSigOpCost
		totalSigOpCost += sigOpCost
		if totalSigOpCost < lastSigOpCost || totalSigOpCost > MaxBlockSigOpsCost {
			str := fmt.Sprintf("block contains too many "+
				"signature operations - got %v, max %v",
				totalSigOpCost, MaxBlockSigOpsCost)
			return ruleError(ErrTooManySigOps, str)
		}
	}

//对每个事务的输入执行多个检查。阿尔索
//累积总费用。这在技术上可以与
//上面的循环而不是在事务上运行另一个循环，
//但是，通过分离，我们可以避免运行更昂贵的（尽管
//与运行脚本相比仍然相对便宜）检查
//当签名操作不在时针对所有输入
//界限。
	var totalFees int64
	for _, tx := range transactions {
		txFee, err := CheckTransactionInputs(tx, node.height, view,
			b.chainParams)
		if err != nil {
			return err
		}

//合计总费用，确保我们不会超出
//累加器。
		lastTotalFees := totalFees
		totalFees += txFee
		if totalFees < lastTotalFees {
			return ruleError(ErrBadFees, "total fees for block "+
				"overflows accumulator")
		}

//添加此事务的所有输出，这些输出不是
//可证明是不可依赖的，如可用的utxo。同时，通过
//
//按每个事务花费它们的顺序花费txout。
		err = view.connectTransaction(tx, node.height, stxos)
		if err != nil {
			return err
		}
	}

//CoinBase事务的总输出值不得超过
//预期补贴价值加上从
//挖掘区块。忽略溢出和超出范围是安全的
//此处出错，因为这些错误条件可能已经
//被CheckTransactionSanity捕获。
	var totalSatoshiOut int64
	for _, txOut := range transactions[0].MsgTx().TxOut {
		totalSatoshiOut += txOut.Value
	}
	expectedSatoshiOut := CalcBlockSubsidy(node.height, b.chainParams) +
		totalFees
	if totalSatoshiOut > expectedSatoshiOut {
		str := fmt.Sprintf("coinbase transaction for block pays %v "+
			"which is more than expected value of %v",
			totalSatoshiOut, expectedSatoshiOut)
		return ruleError(ErrBadCoinbaseValue, str)
	}

//如果此节点早于最新的已知良好状态，则不运行脚本
//检查点，因为通过检查点验证有效性（所有
//事务包含在merkle根散列和任何更改中
//
//优化，因为运行脚本最耗时
//块处理的一部分。
	checkpoint := b.LatestCheckpoint()
	runScripts := true
	if checkpoint != nil && node.height <= checkpoint.Height {
		runScripts = false
	}

//在bip0016激活时间之后创建的块需要
//已启用按脚本付费哈希检查。
	var scriptFlags txscript.ScriptFlags
	if enforceBIP0016 {
		scriptFlags |= txscript.ScriptBip16
	}

//
//已达到激活阈值。这是Bip0066的一部分。
	blockHeader := &block.MsgBlock().Header
	if blockHeader.Version >= 3 && node.height >= b.chainParams.BIP0066Height {
		scriptFlags |= txscript.ScriptVerifyDERSignatures
	}

//对块版本4+执行checkLockTimeVerify
//已达到激活阈值。这是Bip0065的一部分。
	if blockHeader.Version >= 4 && node.height >= b.chainParams.BIP0065Height {
		scriptFlags |= txscript.ScriptVerifyCheckLockTimeVerify
	}

//在所有块验证检查期间执行一次CheckSequenceVerify
//软分叉部署已完全激活。
	csvState, err := b.deploymentState(node.parent, chaincfg.DeploymentCSV)
	if err != nil {
		return err
	}
	if csvState == ThresholdActive {
//如果csv软分叉现在处于活动状态，则修改
//确保csv操作代码正确的脚本标志
//在脚本检查过程中验证。
		scriptFlags |= txscript.ScriptVerifyCheckSequenceVerify

//我们获取*上一个*块的MTP，以便
//确定当前块中的事务是否为最终事务。
		medianTime := node.parent.CalcPastMedianTime()

//此外，如果csv软分叉软件包现在处于活动状态，
//然后，我们还执行基于相对序列号的
//此中所有事务的输入中的锁定时间
//候选块。
		for _, tx := range block.Transactions() {
//事务只能包含在块中
//一旦*所有*输入的序列锁
//主动的。
			sequenceLock, err := b.calcSequenceLock(node, tx, view,
				false)
			if err != nil {
				return err
			}
			if !SequenceLockActive(sequenceLock, node.height,
				medianTime) {
				str := fmt.Sprintf("block contains " +
					"transaction whose input sequence " +
					"locks are not met")
				return ruleError(ErrUnfinalizedTx, str)
			}
		}
	}

//在软叉移动后强制使用Segwit软叉包
//进入“激活”版本位状态。
	if enforceSegWit {
		scriptFlags |= txscript.ScriptVerifyWitness
		scriptFlags |= txscript.ScriptStrictMultiSig
	}

//既然便宜的检查已经完成并通过了，请验证
//
//昂贵的ECDSA签名检查脚本。做这最后的帮助
//防止CPU耗尽攻击。
	if runScripts {
		err := checkBlockScripts(block, view, scriptFlags, b.sigCache,
			b.hashCache)
		if err != nil {
			return err
		}
	}

//更新视图的最佳哈希以包含此块，因为
//交易已连接。
	view.SetBestHash(&node.hash)

	return nil
}

//checkConnectBlockTemplate完全验证将传递的块连接到
//除了证明
//工作要求。块必须连接到主链的当前尖端。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) CheckConnectBlockTemplate(block *btcutil.Block) error {
	b.chainLock.Lock()
	defer b.chainLock.Unlock()

//跳过工作证明检查，因为这只是一个块模板。
	flags := BFNoPoWCheck

//这只检查块是否可以连接到
//电流链。
	tip := b.bestChain.Tip()
	header := block.MsgBlock().Header
	if tip.hash != header.PrevBlock {
		str := fmt.Sprintf("previous block must be the current chain tip %v, "+
			"instead got %v", tip.hash, header.PrevBlock)
		return ruleError(ErrPrevBlockNotBest, str)
	}

	err := checkBlockSanity(block, b.chainParams.PowLimit, b.timeSource, flags)
	if err != nil {
		return err
	}

	err = b.checkBlockContext(block, tip, flags)
	if err != nil {
		return err
	}

//由于信息
//不需要，因此可以避免额外的工作。
	view := NewUtxoViewpoint()
	view.SetBestHash(&tip.hash)
	newNode := newBlockNode(&header, tip)
	return b.checkConnectBlock(newNode, block, view, nil)
}
