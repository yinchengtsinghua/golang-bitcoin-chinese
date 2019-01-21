
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
	"fmt"
	"time"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

const (
//maxstandardp2shsigops是签名操作的最大数目
//这在付费脚本哈希脚本中被认为是标准的。
	maxStandardP2SHSigOps = 15

//MaxStandardTxCost是任何交易允许的最大重量
//根据当前默认策略。
	maxStandardTxWeight = 400000

//MaxStandardSigScriptSize是
//事务输入签名脚本应视为标准脚本。这个
//值允许15/15的支票多任务支付脚本哈希
//压缩键。
//
//The form of the overall script is: OP_0 <15 signatures> OP_PUSHDATA2
//<2 bytes len>[op_15<15 pubkeys>op_15 op_checkmultisig]
//
//对于p2sh脚本部分，15个压缩的pubkey中的每一个都是
//33字节（加上一个用于操作数据操作码），因此它总计
//到（15*34）+3=513字节。接下来，15个签名中的每一个都是最大值
//73字节（加上一个用于操作数据操作码）。还有一个
//初始额外操作的额外字节为0 push，3字节为
//op pushdata2需要为脚本push指定513字节。
//这使得总数达到1+（15*74）+3+513=1627。这个值也
//添加一些额外的字节以提供一点缓冲。
//（1+15*74+3）+（15*34+3）+23=1650
	maxStandardSigScriptSize = 1650

//DufftMunRelayTXXFIE是索托的最低费用
//对于被视为免费进行中继和挖掘的事务
//目的。它还用于帮助确定事务是否
//考虑灰尘，作为计算最低要求费用的基础
//对于较大的交易。该值以Satoshi/1000字节为单位。
	DefaultMinRelayTxFee = btcutil.Amount(1000)

//maxstandardmultisigkeys是允许的最大公钥数。
//在多签名事务输出脚本中，
//考虑的标准。
	maxStandardMultiSigKeys = 3
)

//CalcMinRequiredTxRelayFee返回
//传递的序列化大小的事务将被接受到内存中
//池和中继。
func calcMinRequiredTxRelayFee(serializedSize int64, minRelayTxFee btcutil.Amount) int64 {
//计算允许进入的交易的最低费用
//通过扩展基本费用（最低
//免费交易中继费）。mintxrelayfee在satoshi/kb中，所以
//乘以序列化大小（以字节为单位）并除以1000
//获得最小饱和值。
	minFee := (serializedSize * int64(minRelayTxFee)) / 1000

	if minFee == 0 && minRelayTxFee > 0 {
		minFee = int64(minRelayTxFee)
	}

//如果计算出
//费用不在货币金额的有效范围内。
	if minFee < 0 || minFee > btcutil.MaxSatoshi {
		minFee = btcutil.MaxSatoshi
	}

	return minFee
}

//checkinputsstandard对事务的输入执行一系列检查
//以确保它们是“标准”的。标准事务输入
//此函数的上下文是其引用的公钥脚本的
//标准表单和，对于付费脚本哈希，其值不超过
//maxstandardp2shsigops签名操作。但是，也应注意
//标准输入也是那些在执行后具有干净堆栈的输入
//并且只在它们的签名脚本中包含推送的数据。这个函数可以
//不执行这些检查，因为脚本引擎已经做了更多的检查
//通过txscript.scriptVerifyCleanStack和
//txscript.scriptVerifySigPushOnly标志。
func checkInputsStandard(tx *btcutil.Tx, utxoView *blockchain.UtxoViewpoint) error {
//注意：引用实现在这里也进行了一个coinbase检查，
//but coinbases have already been rejected prior to calling this
//功能，无需重新检查。

	for i, txIn := range tx.MsgTx().TxIn {
//这是安全的存在和索引检查这里以来
//打电话之前已经检查过了
//功能。
		entry := utxoView.LookupEntry(txIn.PreviousOutPoint)
		originPkScript := entry.PkScript()
		switch txscript.GetScriptClass(originPkScript) {
		case txscript.ScriptHashTy:
			numSigOps := txscript.GetPreciseSigOpCount(
				txIn.SignatureScript, originPkScript, true)
			if numSigOps > maxStandardP2SHSigOps {
				str := fmt.Sprintf("transaction input #%d has "+
					"%d signature operations which is more "+
					"than the allowed max amount of %d",
					i, numSigOps, maxStandardP2SHSigOps)
				return txRuleError(wire.RejectNonstandard, str)
			}

		case txscript.NonStandardTy:
			str := fmt.Sprintf("transaction input #%d has a "+
				"non-standard script form", i)
			return txRuleError(wire.RejectNonstandard, str)
		}
	}

	return nil
}

//checkpkscriptStandard对事务输出执行一系列检查
//脚本（公钥脚本），以确保它是“标准”公钥脚本。
//标准公钥脚本是可识别的形式，并且
//multi-signature scripts, only contains from 1 to maxStandardMultiSigKeys
//公钥。
func checkPkScriptStandard(pkScript []byte, scriptClass txscript.ScriptClass) error {
	switch scriptClass {
	case txscript.MultiSigTy:
		numPubKeys, numSigs, err := txscript.CalcMultiSigStats(pkScript)
		if err != nil {
			str := fmt.Sprintf("multi-signature script parse "+
				"failure: %v", err)
			return txRuleError(wire.RejectNonstandard, str)
		}

//标准多签名公钥脚本必须包含
//从1到maxstandardmultisigkeys公钥。
		if numPubKeys < 1 {
			str := "multi-signature script with no pubkeys"
			return txRuleError(wire.RejectNonstandard, str)
		}
		if numPubKeys > maxStandardMultiSigKeys {
			str := fmt.Sprintf("multi-signature script with %d "+
				"public keys which is more than the allowed "+
				"max of %d", numPubKeys, maxStandardMultiSigKeys)
			return txRuleError(wire.RejectNonstandard, str)
		}

//标准多签名公钥脚本必须具有
//至少1个签名，最多不超过可用的签名
//公钥。
		if numSigs < 1 {
			return txRuleError(wire.RejectNonstandard,
				"multi-signature script with no signatures")
		}
		if numSigs > numPubKeys {
			str := fmt.Sprintf("multi-signature script with %d "+
				"signatures which is more than the available "+
				"%d public keys", numSigs, numPubKeys)
			return txRuleError(wire.RejectNonstandard, str)
		}

	case txscript.NonStandardTy:
		return txRuleError(wire.RejectNonstandard,
			"non-standard script form")
	}

	return nil
}

//ISDUST返回传递的事务输出量是否为
//是否考虑灰尘基于通过的最低交易中继费。
//灰尘是根据最低交易中继费定义的。在
//特别是，如果网络花费硬币的成本超过
//最低交易中继费，视为灰尘。
func isDust(txOut *wire.TxOut, minRelayTxFee btcutil.Amount) bool {
//不可靠的输出被认为是灰尘。
	if txscript.IsUnspendable(txOut.PkScript) {
		return true
	}

//总序列化大小由输出和关联的
//输入脚本以兑现它。因为没有输入脚本
//要兑现它，请使用典型输入脚本的最小大小。
//
//支付到PubKey哈希字节细分：
//
//输出到哈希（34字节）：
//8个值，1个脚本长度，25个脚本[1个opu-dup，1个opu-hash-160，
//1 OpthDATAY20，20哈希，1 opyQualalTalk，1 OpCalsixGig]
//
//使用压缩的pubkey（148字节）输入：
//36个前输出点，1个脚本长度，107个脚本[1个操作数据，72个信号，
//1 op_data_33，33 compressed pubkey]，4序列
//
//Input with uncompressed pubkey (180 bytes):
//36个前输出点，1个脚本长度，139个脚本[1个操作数据，72个信号，
//1 op_data_65，65 compressed pubkey]，4序列
//
//支付到Pubkey字节细分：
//
//输出到压缩的pubkey（44字节）：
//8个值，1个脚本长度，35个脚本[1个操作数据
//33压缩Pubkey，1个opu checksig]
//
//Output to uncompressed pubkey (76 bytes):
//8个值，1个脚本长度，67个脚本[1个操作数据，65个pubkey，
//1 OpsiCalgsig
//
//输入（114字节）：
//36个前输出点，1个脚本长度，73个脚本[1个操作数据点72，
//72个信号，4个序列
//
//Pay-to-witness-pubkey-hash bytes breakdown:
//
//输出到见证密钥哈希（31字节）；
//8个值，1个脚本长度，22个脚本[1个op_0，1个op_数据_20，
//20字节哈希160]
//
//输入（67字节，因为107见证堆栈已折扣）：
//36前输出点，1个脚本长度，0个脚本（非sigscript），107
//见证堆栈字节[1个元素长度，33个压缩pubkey，
//元件长度72 sig]，4序列
//
//
//理论上，这可以检查输出脚本的脚本类型
//并使用不同的大小作为
//按上述细分向pubkey支付与向pubkey支付哈希输入，
//但唯一的组合小于所选择的价值
//使用压缩的pubkey的pay-to-pubkey脚本
//常见的。
//
//最常见的脚本是pay-to-pubkey散列，根据上面的内容
//细分，p2pkh输入脚本的最小大小为148字节。所以
//that figure is used. If the output being spent is a witness program,
//然后我们将证人折扣应用于签名的大小。
//
//The segwit analogue to p2pkh is a p2wkh output. This is the smallest
//使用新的Segwit功能可以输出。的107字节
//证人数据的贴现系数为4，得出
//67字节见证数据的值。
//
//两种情况都共享一个引用输入所需的41字节前导码。
//被花费和输入的序列号。
	totalSize := txOut.SerializeSize() + 41
	if txscript.IsWitnessProgram(txOut.PkScript) {
		totalSize += (107 / blockchain.WitnessScaleFactor)
	} else {
		totalSize += 107
	}

//如果网络花费的成本
//硬币超过最低免费交易中继费的1/3。
//minfreetxrelayfee以satoshi/kb为单位，因此乘以1000到
//转换为字节。
//
//使用付款到公钥哈希事务的典型值
//the breakdown above and the default minimum free transaction relay
//收费1000元，相当于小于546元的Satoshi
//考虑灰尘。
//
//以下相当于（值/总大小）*（1/3）*1000
//不需要做浮点运算。
	return txOut.Value*1000/(3*int64(totalSize)) < int64(minRelayTxFee)
}

//checkTransactionStandard对要执行的事务执行一系列检查
//确保这是“标准”交易。标准交易是指
//在被认为是
//“正常”事务，例如在支持的范围内有一个版本，即
//最终确定，符合更严格的大小限制，具有脚本
//具有公认的形式，不包含“灰尘”输出（那些
//so small it costs more to process them than they are worth).
func checkTransactionStandard(tx *btcutil.Tx, height int32,
	medianTimePast time.Time, minRelayTxFee btcutil.Amount,
	maxTxVersion int32) error {

//事务必须是当前支持的版本。
	msgTx := tx.MsgTx()
	if msgTx.Version > maxTxVersion || msgTx.Version < 1 {
		str := fmt.Sprintf("transaction version %d is not in the "+
			"valid range of %d-%d", msgTx.Version, 1,
			maxTxVersion)
		return txRuleError(wire.RejectNonstandard, str)
	}

//交易必须最终确定为标准交易，因此
//考虑包含在一个块中。
	if !blockchain.IsFinalizedTransaction(tx, height, medianTimePast) {
		return txRuleError(wire.RejectNonstandard,
			"transaction is not finalized")
	}

//因为具有大量输入的非常大的事务可能需要成本
//几乎和发送方费用一样多，限制最大
//事务的大小。这也有助于减轻CPU消耗
//攻击。
	txWeight := blockchain.GetTransactionWeight(tx)
	if txWeight > maxStandardTxWeight {
		str := fmt.Sprintf("weight of transaction %v is larger than max "+
			"allowed weight of %v", txWeight, maxStandardTxWeight)
		return txRuleError(wire.RejectNonstandard, str)
	}

	for i, txIn := range msgTx.TxIn {
//每个事务输入签名脚本不得超过
//标准事务允许的最大大小。见
//有关MaxStandardSigScriptSize的注释以了解更多详细信息。
		sigScriptLen := len(txIn.SignatureScript)
		if sigScriptLen > maxStandardSigScriptSize {
			str := fmt.Sprintf("transaction input %d: signature "+
				"script size of %d bytes is large than max "+
				"allowed size of %d bytes", i, sigScriptLen,
				maxStandardSigScriptSize)
			return txRuleError(wire.RejectNonstandard, str)
		}

//每个事务输入签名脚本只能包含
//将数据推送到堆栈上的操作码。
		if !txscript.IsPushOnlyScript(txIn.SignatureScript) {
			str := fmt.Sprintf("transaction input %d: signature "+
				"script is not push only", i)
			return txRuleError(wire.RejectNonstandard, str)
		}
	}

//所有输出的公钥脚本都不能是非标准脚本，或者
//为“dust”（除非脚本为空数据脚本）。
	numNullDataOutputs := 0
	for i, txOut := range msgTx.TxOut {
		scriptClass := txscript.GetScriptClass(txOut.PkScript)
		err := checkPkScriptStandard(txOut.PkScript, scriptClass)
		if err != nil {
//尝试从错误中提取拒绝代码，因此
//可以保留。如果不可能，返回
//非标准错误。
			rejectCode := wire.RejectNonstandard
			if rejCode, found := extractRejectCode(err); found {
				rejectCode = rejCode
			}
			str := fmt.Sprintf("transaction output %d: %v", i, err)
			return txRuleError(rejectCode, str)
		}

//累积只携带数据的输出数量。为了
//所有其他脚本类型，确保输出值不是
//“灰尘”。
		if scriptClass == txscript.NullDataTy {
			numNullDataOutputs++
		} else if isDust(txOut, minRelayTxFee) {
			str := fmt.Sprintf("transaction output %d: payment "+
				"of %d is dust", i, txOut.Value)
			return txRuleError(wire.RejectDust, str)
		}
	}

//标准事务不能有多个输出脚本，
//只携带数据。
	if numNullDataOutputs > 1 {
		str := "more than one transaction output in a nulldata script"
		return txRuleError(wire.RejectNonstandard, str)
	}

	return nil
}

//gettxVirtualSize计算给定事务的虚拟大小。一
//交易的虚拟大小基于其权重，为
//它包含的任何见证数据，与当前数据成比例
//区块链。见证尺度因子值。
func GetTxVirtualSize(tx *btcutil.Tx) int64 {
//V尺寸：=（重量（Tx）+3）/4
//：=（（基本尺寸*3）+总尺寸+3）/4
//我们在这里加3，作为计算先前算法上限的一种方法。
//到4。除以4可为WIT见证数据创建折扣。
	return (blockchain.GetTransactionWeight(tx) + (blockchain.WitnessScaleFactor - 1)) /
		blockchain.WitnessScaleFactor
}
