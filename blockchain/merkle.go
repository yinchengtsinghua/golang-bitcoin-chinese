
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

package blockchain

import (
	"bytes"
	"fmt"
	"math"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
)

const (
//coinbasewitnessdatalen是
//如果CoinBase事务包含
//见证承诺。
	CoinbaseWitnessDataLen = 32

//
//包含opu返回、见证magicBytes和见证
//
//包含证人承诺
	CoinbaseWitnessPkScriptLength = 38
)

var (
//witnessMagicBytes是公钥脚本中的前缀标记
//一个coinbase输出，以指示此输出包含见证
//一个街区的承诺。
	WitnessMagicBytes = []byte{
		txscript.OP_RETURN,
		txscript.OP_DATA_36,
		0xaa,
		0x21,
		0xa9,
		0xed,
	}
)

//nextPowerOfTwo返回给定数字的下一个最大二次幂，如果
//它还不是二的力量。这是在
//
func nextPowerOfTwo(n int) int {
//如果已经是2的幂，则返回该数字。
	if n&(n-1) == 0 {
		return n
	}

//算出并返回二的下一个幂。
	exponent := uint(math.Log2(float64(n))) + 1
return 1 << exponent //2 ^指数
}

//
//节点，并返回其串联的哈希值。这是个帮手
//
func HashMerkleBranches(left *chainhash.Hash, right *chainhash.Hash) *chainhash.Hash {
//连接左右节点。
	var hash [chainhash.HashSize * 2]byte
	copy(hash[:chainhash.HashSize], left[:])
	copy(hash[chainhash.HashSize:], right[:])

	newHash := chainhash.DoubleHashH(hash[:])
	return &newHash
}

//buildMerkleTreeStore从事务切片创建Merkle树，
//使用线性数组存储它，并返回支持数组的一个切片。一
//由于线性数组使用了
//大约是记忆的一半。下面描述了一棵梅克尔树以及它是如何
//存储在线性数组中。
//
//
//子节点。描述比特币交易如何运作的图表
//其中h（x）是双sha256，如下所示：
//
//根=h1234=h（h12+h34）
//
//h12=h（h1+h2）h34=h（h3+h4）
//
//
//
//
//
//[h1 h2 h3 h4 h12 h34根]
//
//如上所示，merkle根始终是数组中的最后一个元素。
//
//
//平衡的树结构如上所述。在这种情况下，没有
//子节点也是零，父节点只有一个左节点
//通过在散列前将左节点与其自身连接来计算。
//
//将是零。
//
//
//使用见证事务ID而不是常规事务ID。
//此外，还提供了一个附加案例，其中coinbase事务的wtxid
//是零哈希。
func BuildMerkleTreeStore(transactions []*btcutil.Tx, witness bool) []*chainhash.Hash {
//计算保存二进制merkle需要多少个条目
//
	nextPoT := nextPowerOfTwo(len(transactions))
	arraySize := nextPoT*2 - 1
	merkles := make([]*chainhash.Hash, arraySize)

//
	for i, tx := range transactions {
//如果我们正在计算证人merkle root，而不是
//常规txid，我们使用修改后的wtxid，其中包括
//摘要中的事务见证数据。此外，
//Coinbase的wtxid都是零。
		switch {
		case witness && i == 0:
			var zeroHash chainhash.Hash
			merkles[i] = &zeroHash
		case witness:
			wSha := tx.MsgTx().WitnessHash()
			merkles[i] = &wSha
		default:
			merkles[i] = tx.Hash()
		}

	}

//在最后一个事务之后启动数组偏移量并将其调整为
//
	offset := nextPoT
	for i := 0; i < arraySize-1; i += 2 {
		switch {
//当没有左子节点时，父节点也为零。
		case merkles[i] == nil:
			merkles[offset] = nil

//当没有正确的子级时，父级由
//
		case merkles[i+1] == nil:
			newHash := HashMerkleBranches(merkles[i], merkles[i])
			merkles[offset] = newHash

//
//左右儿童的终结。
		default:
			newHash := HashMerkleBranches(merkles[i], merkles[i+1])
			merkles[offset] = newHash
		}
		offset++
	}

	return merkles
}

//
//
//
//
//在传递的事务中。见证承诺存储为数据推送
//对于带特殊魔力字节的返回操作，以帮助定位。
func ExtractWitnessCommitment(tx *btcutil.Tx) ([]byte, bool) {
//见证承诺*必须*位于一个CoinBase内
//事务的输出。
	if !IsCoinBase(tx) {
		return nil, false
	}

	msgTx := tx.MsgTx()
	for i := len(msgTx.TxOut) - 1; i >= 0; i-- {
//包含证人承诺的公钥脚本
//必须与WitnessMagicBytes共享前缀，并位于
//
		pkScript := msgTx.TxOut[i].PkScript
		if len(pkScript) >= CoinbaseWitnessPkScriptLength &&
			bytes.HasPrefix(pkScript, WitnessMagicBytes) {

//见证承诺本身是一个32字节的哈希
//
//超过第38字节的字节当前没有一致意见
//意义。
			start := len(WitnessMagicBytes)
			end := CoinbaseWitnessPkScriptLength
			return msgTx.TxOut[i].PkScript[start:end], true
		}
	}

	return nil, false
}

//
//在传递的块的CoinBase事务中。
func ValidateWitnessCommitment(blk *btcutil.Block) error {
//如果区块根本没有任何交易，那么我们就不会
//能够从不存在的CoinBase中提取承诺
//交易。所以我们早点离开这里。
	if len(blk.Transactions()) == 0 {
		str := "cannot validate witness commitment of block without " +
			"transactions"
		return ruleError(ErrNoTransactions, str)
	}

	coinbaseTx := blk.Transactions()[0]
	if len(coinbaseTx.MsgTx().TxIn) == 0 {
		return ruleError(ErrNoTxInputs, "transaction has no inputs")
	}

	witnessCommitment, witnessFound := ExtractWitnessCommitment(coinbaseTx)

//如果我们找不到任何Coinbase的证人承诺
//输出，则块不能包含
//见证数据。
	if !witnessFound {
		for _, tx := range blk.Transactions() {
			msgTx := tx.MsgTx()
			if msgTx.HasWitness() {
				str := fmt.Sprintf("block contains transaction with witness" +
					" data, yet no witness commitment present")
				return ruleError(ErrUnexpectedWitness, str)
			}
		}
		return nil
	}

//在这一点上，区块包含证人承诺，因此
//
//它的见证数据和元素必须
//coinbasewitnessdatalen字节。
	coinbaseWitness := coinbaseTx.MsgTx().TxIn[0].Witness
	if len(coinbaseWitness) != 1 {
		str := fmt.Sprintf("the coinbase transaction has %d items in "+
			"its witness stack when only one is allowed",
			len(coinbaseWitness))
		return ruleError(ErrInvalidWitnessCommitment, str)
	}
	witnessNonce := coinbaseWitness[0]
	if len(witnessNonce) != CoinbaseWitnessDataLen {
		str := fmt.Sprintf("the coinbase transaction witness nonce "+
			"has %d bytes when it must be %d bytes",
			len(witnessNonce), CoinbaseWitnessDataLen)
		return ruleError(ErrInvalidWitnessCommitment, str)
	}

//最后，在初步检查结束后，我们可以检查
//提取的见证承诺等于：
//sha256（见证人：merkleroot见证人：nonce）。目击证人在哪里
//CoinBase事务的唯一见证项。
	witnessMerkleTree := BuildMerkleTreeStore(blk.Transactions(), true)
	witnessMerkleRoot := witnessMerkleTree[len(witnessMerkleTree)-1]

	var witnessPreimage [chainhash.HashSize * 2]byte
	copy(witnessPreimage[:], witnessMerkleRoot[:])
	copy(witnessPreimage[chainhash.HashSize:], witnessNonce)

	computedCommitment := chainhash.DoubleHashB(witnessPreimage[:])
	if !bytes.Equal(computedCommitment, witnessCommitment) {
		str := fmt.Sprintf("witness commitment does not match: "+
			"computed %v, coinbase includes %v", computedCommitment,
			witnessCommitment)
		return ruleError(ErrWitnessCommitmentMismatch, str)
	}

	return nil
}
