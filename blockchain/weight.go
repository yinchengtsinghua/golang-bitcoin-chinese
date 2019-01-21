
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
	"fmt"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

const (
//maxblockweight定义最大块权重，其中“block
//重量”的定义见Bip0141。一个街区的重量是
//计算为现有事务中字节的和
//和头，加上事务中每个字节的权重。这个
//“基”字节的权重为4，而见证字节的权重为
//1。因此，要使块有效，块权重必须为
//小于或等于MaxBlockWeight。
	MaxBlockWeight = 4000000

//MaxBlockBaseSize是块中的最大字节数。
//可分配给非见证数据。
	MaxBlockBaseSize = 1000000

//maxblocksigopscost是签名操作的最大数目
//允许一个块。通过加权算法计算
//权重分离见证信号操作比常规信号操作低。
	MaxBlockSigOpsCost = 80000

//WitnessScaleFactor确定“折扣”见证数据的级别
//
//证人数据比普通的非证人数据便宜1/4。
	WitnessScaleFactor = 4

//MintXOutputWeight是事务的最小可能权重
//输出。
	MinTxOutputWeight = WitnessScaleFactor * wire.MinTxOutPayload

//MaxOutputsPerBlock是其中事务输出的最大数目。
//可以是最大重量大小的块。
	MaxOutputsPerBlock = MaxBlockWeight / MinTxOutputWeight
)

//GetBlockWeight计算给定块的权重度量值。
//目前，权重度量只是块序列化大小的总和。
//没有任何证人数据由证人比例因子按比例缩放，
//以及块的序列化大小，包括任何见证数据。
func GetBlockWeight(blk *btcutil.Block) int64 {
	msgBlock := blk.MsgBlock()

	baseSize := msgBlock.SerializeSizeStripped()
	totalSize := msgBlock.SerializeSize()

//（基本尺寸*3）+总尺寸
	return int64((baseSize * (WitnessScaleFactor - 1)) + totalSize)
}

//GetTransactionWeight计算给定
//交易。目前，重量度量只是
//事务的序列化大小，不缩放任何见证数据
//按比例通过见证scaleFactor，并将事务序列化
//
func GetTransactionWeight(tx *btcutil.Tx) int64 {
	msgTx := tx.MsgTx()

	baseSize := msgTx.SerializeSizeStripped()
	totalSize := msgTx.SerializeSize()

//（基本尺寸*3）+总尺寸
	return int64((baseSize * (WitnessScaleFactor - 1)) + totalSize)
}

//GetSigOpCost返回传递事务的统一Sig Op成本
//考虑到现行有效的软叉修改了SIG操作成本计算。
//一个事务的统一SIG OP成本计算为：
//传统SIG操作计数根据见证scaleFactor、SIG操作进行缩放
//统计由见证比例因子缩放的所有p2sh输入，最后是
//用于支出见证计划的任何输入的未标度SIG OP计数。
func GetSigOpCost(tx *btcutil.Tx, isCoinBaseTx bool, utxoView *UtxoViewpoint, bip16, segWit bool) (int, error) {
	numSigOps := CountSigOps(tx) * WitnessScaleFactor
	if bip16 {
		numP2SHSigOps, err := CountP2SHSigOps(tx, isCoinBaseTx, utxoView)
		if err != nil {
			return 0, nil
		}
		numSigOps += (numP2SHSigOps * WitnessScaleFactor)
	}

	if segWit && !isCoinBaseTx {
		msgTx := tx.MsgTx()
		for txInIndex, txIn := range msgTx.TxIn {
//确保参考输出可用且没有
//
			utxo := utxoView.LookupEntry(txIn.PreviousOutPoint)
			if utxo == nil || utxo.IsSpent() {
				str := fmt.Sprintf("output %v referenced from "+
					"transaction %s:%d either does not "+
					"exist or has already been spent",
					txIn.PreviousOutPoint, tx.Hash(),
					txInIndex)
				return 0, ruleError(ErrMissingTxOut, str)
			}

			witness := txIn.Witness
			sigScript := txIn.SignatureScript
			pkScript := utxo.PkScript()
			numSigOps += txscript.GetWitnessSigOpCount(sigScript, pkScript, witness)
		}

	}

	return numSigOps, nil
}
