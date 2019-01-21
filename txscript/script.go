
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

package txscript

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

//bip16activation是bip0016在
//块链。用于确定是否应调用bip0016。
//此时间戳对应于2012年4月1日00:00:00 UTC。
var Bip16Activation = time.Unix(1333238400, 0)

//sighashType表示签名末尾的哈希类型位。
type SigHashType uint32

//从签名结尾开始的哈希类型位。
const (
	SigHashOld          SigHashType = 0x0
	SigHashAll          SigHashType = 0x1
	SigHashNone         SigHashType = 0x2
	SigHashSingle       SigHashType = 0x3
	SigHashAnyOneCanPay SigHashType = 0x80

//sighashmask定义使用的哈希类型的位数
//识别哪些输出被签名。
	sigHashMask = 0x1f
)

//这些是为单个脚本中的最大值指定的常量。
const (
MaxOpsPerScript       = 201 //非强制操作的最大数目。
MaxPubKeysPerMultiSig = 20  //multisig不能有比这更多的信号。
MaxScriptElementSize  = 520 //可推送到堆栈的最大字节数。
)

//issmallint返回操作码是否被视为小整数，
//它是一个op_0，或op_1到op_16。
func isSmallInt(op *opcode) bool {
	if op.value == OP_0 || (op.value >= OP_1 && op.value <= OP_16) {
		return true
	}
	return false
}

//如果传递的脚本是付费脚本哈希，则IsScriptHash返回true
//事务，否则为false。
func isScriptHash(pops []parsedOpcode) bool {
	return len(pops) == 3 &&
		pops[0].opcode.value == OP_HASH160 &&
		pops[1].opcode.value == OP_DATA_20 &&
		pops[2].opcode.value == OP_EQUAL
}

//如果脚本在标准中，ispaytoScriptHash返回true
//pay-to-script散列（p2sh）格式，否则为false。
func IsPayToScriptHash(script []byte) bool {
	pops, err := parseScript(script)
	if err != nil {
		return false
	}
	return isScriptHash(pops)
}

//如果传递的脚本是
//付费见证脚本哈希事务，否则为false。
func isWitnessScriptHash(pops []parsedOpcode) bool {
	return len(pops) == 2 &&
		pops[0].opcode.value == OP_0 &&
		pops[1].opcode.value == OP_DATA_32
}

//IsPayToWitnessScriptHash如果在标准中，则返回true
//付费见证脚本哈希（p2wsh）格式，否则为false。
func IsPayToWitnessScriptHash(script []byte) bool {
	pops, err := parseScript(script)
	if err != nil {
		return false
	}
	return isWitnessScriptHash(pops)
}

//IsPayToWitnessPubKeyHash如果在标准中，则返回true
//付费见证pubkey散列（p2wkh）格式，否则为false。
func IsPayToWitnessPubKeyHash(script []byte) bool {
	pops, err := parseScript(script)
	if err != nil {
		return false
	}
	return isWitnessPubKeyHash(pops)
}

//如果传递的脚本是
//支付给证人pubkey散列，否则为false。
func isWitnessPubKeyHash(pops []parsedOpcode) bool {
	return len(pops) == 2 &&
		pops[0].opcode.value == OP_0 &&
		pops[1].opcode.value == OP_DATA_20
}

//如果传递的脚本是有效见证，则IsWitnessProgram返回true
//根据通过的见证程序版本进行编码的程序。一
//见证程序必须是一个小整数（从0到16），后跟2到40个字节
//推送数据。
func IsWitnessProgram(script []byte) bool {
//脚本的长度必须介于4到42个字节之间。这个
//最小的程序是见证版本，然后是
//2字节。允许的最大见证程序的数据推送为
//40字节。
	if len(script) < 4 || len(script) > 42 {
		return false
	}

	pops, err := parseScript(script)
	if err != nil {
		return false
	}

	return isWitnessProgram(pops)
}

//如果传递的脚本是见证程序，则IsWitnessProgram返回true，并且
//否则为假。见证程序必须遵守以下限制：
//必须有两个POP（程序版本和程序本身），即
//第一个操作码必须是一个小整数（0-16），推送数据必须是
//规范化，最后推送数据的大小必须介于2和40之间
//字节。
func isWitnessProgram(pops []parsedOpcode) bool {
	return len(pops) == 2 &&
		isSmallInt(pops[0].opcode) &&
		canonicalPush(pops[1]) &&
		(len(pops[1].data) >= 2 && len(pops[1].data) <= 40)
}

//extractwitnessprogramminfo尝试提取见证程序版本，
//以及通过脚本的见证程序本身。
func ExtractWitnessProgramInfo(script []byte) (int, []byte, error) {
	pops, err := parseScript(script)
	if err != nil {
		return 0, nil, err
	}

//如果在这一点上，脚本不像一个证人程序，
//然后我们会尽早退出，因为没有有效的版本或程序
//提取液。
	if !isWitnessProgram(pops) {
		return 0, nil, fmt.Errorf("script is not a witness program, " +
			"unable to extract version or witness program")
	}

	witnessVersion := asSmallInt(pops[0].opcode)
	witnessProgram := pops[1].data

	return witnessVersion, witnessProgram, nil
}

//如果脚本只推送数据，则ispushOnly返回true，否则返回false。
func isPushOnly(pops []parsedOpcode) bool {
//注意：此函数不直接验证操作码，因为它是
//内部的，并且仅使用已解析的操作码调用
//没有任何分析错误。因此，共识得到了妥善维护。

	for _, pop := range pops {
//所有操作码到opu 16都是数据推送指令。
//注意：这并不认为opu保留为数据推送
//指令，但执行opu reserved无论如何都会失败
//并且符合共识所要求的行为。
		if pop.opcode.value > OP_16 {
			return false
		}
	}
	return true
}

//ispushOnlyscript返回传递的脚本是否只推送数据。
//
//当脚本不分析时，将返回false。
func IsPushOnlyScript(script []byte) bool {
	pops, err := parseScript(script)
	if err != nil {
		return false
	}
	return isPushOnly(pops)
}

//ParseScriptTemplate与ParseScript相同，但允许传递
//用于测试的模板列表。当有分析错误时，它返回
//分析到故障点的操作码列表以及错误。
func parseScriptTemplate(script []byte, opcodes *[256]opcode) ([]parsedOpcode, error) {
	retScript := make([]parsedOpcode, 0, len(script))
	for i := 0; i < len(script); {
		instr := script[i]
		op := &opcodes[instr]
		pop := parsedOpcode{opcode: op}

//根据指令分析数据。
		switch {
//没有其他数据。注意一些操作码，特别是
//op1negate、op_0和op_[1-16]表示数据
//他们自己。
		case op.length == 1:
			i++

//特定长度的数据推送——op_data_u[1-75]。
		case op.length > 1:
			if len(script[i:]) < op.length {
				str := fmt.Sprintf("opcode %s requires %d "+
					"bytes, but script only has %d remaining",
					op.name, op.length, len(script[i:]))
				return retScript, scriptError(ErrMalformedPush,
					str)
			}

//切掉数据。
			pop.data = script[i+1 : i+op.length]
			i += op.length

//使用解析长度的数据推送——op pushdatap 1,2,4。
		case op.length < 0:
			var l uint
			off := i + 1

			if len(script[off:]) < -op.length {
				str := fmt.Sprintf("opcode %s requires %d "+
					"bytes, but script only has %d remaining",
					op.name, -op.length, len(script[off:]))
				return retScript, scriptError(ErrMalformedPush,
					str)
			}

//下一个长度字节是数据的小尾数长度。
			switch op.length {
			case -1:
				l = uint(script[off])
			case -2:
				l = ((uint(script[off+1]) << 8) |
					uint(script[off]))
			case -4:
				l = ((uint(script[off+3]) << 24) |
					(uint(script[off+2]) << 16) |
					(uint(script[off+1]) << 8) |
					uint(script[off]))
			default:
				str := fmt.Sprintf("invalid opcode length %d",
					op.length)
				return retScript, scriptError(ErrMalformedPush,
					str)
			}

//将偏移量移动到数据的开头。
			off += -op.length

//不允许不符合脚本或
//签名扩展。
			if int(l) > len(script[off:]) || int(l) < 0 {
				str := fmt.Sprintf("opcode %s pushes %d bytes, "+
					"but script only has %d remaining",
					op.name, int(l), len(script[off:]))
				return retScript, scriptError(ErrMalformedPush,
					str)
			}

			pop.data = script[off : off+int(l)]
			i += 1 - op.length + int(l)
		}

		retScript = append(retScript, pop)
	}

	return retScript, nil
}

//ParseScript将脚本以字节为单位准备到ParseDoCodes列表中，同时
//应用一些健全的检查。
func parseScript(script []byte) ([]parsedOpcode, error) {
	return parseScriptTemplate(script, &opcodeArray)
}

//UnparseScript撤消了ParseScript的操作，并返回
//将dopcodes解析为字节列表
func unparseScript(pops []parsedOpcode) ([]byte, error) {
	script := make([]byte, 0, len(pops))
	for _, pop := range pops {
		b, err := pop.bytes()
		if err != nil {
			return nil, err
		}
		script = append(script, b...)
	}
	return script, nil
}

//disasmstring为一行打印格式化已反汇编的脚本。当
//脚本无法分析，返回的字符串将包含反汇编的
//脚本到发生故障的点，以及字符串“[错误]”
//附加的。此外，返回脚本未能分析的原因
//如果调用者想要更多关于故障的信息。
func DisasmString(buf []byte) (string, error) {
	var disbuf bytes.Buffer
	opcodes, err := parseScript(buf)
	for _, pop := range opcodes {
		disbuf.WriteString(pop.print(true))
		disbuf.WriteByte(' ')
	}
	if disbuf.Len() > 0 {
		disbuf.Truncate(disbuf.Len() - 1)
	}
	if err != nil {
		disbuf.WriteString("[error]")
	}
	return disbuf.String(), err
}

//remove opcode将从操作码中删除任何与“opcode”匹配的操作码。
//pkscript中的流
func removeOpcode(pkscript []parsedOpcode, opcode byte) []parsedOpcode {
	retScript := make([]parsedOpcode, 0, len(pkscript))
	for _, pop := range pkscript {
		if pop.opcode.value != opcode {
			retScript = append(retScript, pop)
		}
	}
	return retScript
}

//如果对象不是push指令，Canonicalpush返回true
//或包含的push指令，其中与规范形式匹配
//或者使用最小的指令来完成这项工作。否则为假。
func canonicalPush(pop parsedOpcode) bool {
	opcode := pop.opcode.value
	data := pop.data
	dataLen := len(pop.data)
	if opcode > OP_16 {
		return true
	}

	if opcode < OP_PUSHDATA1 && opcode > OP_0 && (dataLen == 1 && data[0] <= 16) {
		return false
	}
	if opcode == OP_PUSHDATA1 && dataLen < OP_PUSHDATA1 {
		return false
	}
	if opcode == OP_PUSHDATA2 && dataLen <= 0xff {
		return false
	}
	if opcode == OP_PUSHDATA4 && dataLen <= 0xffff {
		return false
	}
	return true
}

//removeopcodebydata将返回脚本减去任何将推送的操作码
//传递给堆栈的数据。
func removeOpcodeByData(pkscript []parsedOpcode, data []byte) []parsedOpcode {
	retScript := make([]parsedOpcode, 0, len(pkscript))
	for _, pop := range pkscript {
		if !canonicalPush(pop) || !bytes.Contains(pop.data, data) {
			retScript = append(retScript, pop)
		}
	}
	return retScript

}

//calchashprevouts计算以前所有输出的单个哈希
//（txid:index）在传递的事务中引用。此计算哈希
//可在验证所有支出segwit输出的输入时重新使用，
//sighashall的签名哈希类型。这允许验证重复使用以前的
//散列计算，减少验证sighashall输入的复杂性
//从O（n^2）到O（n）。
func calcHashPrevOuts(tx *wire.MsgTx) chainhash.Hash {
	var b bytes.Buffer
	for _, in := range tx.TxIn {
//首先写出32字节事务ID，其中一个
//此输入正在引用输出。
		b.Write(in.PreviousOutPoint.Hash[:])

//接下来，我们将引用输出的索引编码为
//小端整数。
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], in.PreviousOutPoint.Index)
		b.Write(buf[:])
	}

	return chainhash.DoubleHashH(b.Bytes())
}

//CalcHashSequence计算每个序列号的聚合哈希
//在已传递事务的输入中。这个散列可以重复使用
//当验证所有花费Segwit输出（包括签名）的输入时
//使用sighashall sighash类型。这允许验证重复使用以前的
//散列计算，减少验证sighashall输入的复杂性
//从O（n^2）到O（n）。
func calcHashSequence(tx *wire.MsgTx) chainhash.Hash {
	var b bytes.Buffer
	for _, in := range tx.TxIn {
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], in.Sequence)
		b.Write(buf[:])
	}

	return chainhash.DoubleHashH(b.Bytes())
}

//CalcHashOutputs计算由
//使用有线格式编码的事务。这个散列可以重复使用
//在验证所有投入时，支出见证计划，包括
//签名使用sighashall sighash类型。这允许计算
//缓存，将总哈希复杂性从O（n^2）降低到O（n）。
func calcHashOutputs(tx *wire.MsgTx) chainhash.Hash {
	var b bytes.Buffer
	for _, out := range tx.TxOut {
		wire.WriteTxOut(&b, 0, 0, out)
	}

	return chainhash.DoubleHashH(b.Bytes())
}

//CalcWitnessSignatureHash计算交易的Sightash摘要
//使用新的、优化的摘要计算算法定义的segwit输入
//在bip0143中：https://github.com/bitcoin/bips/blob/master/bip-0143.mediawiki。
//此函数使用存储在
//传递的hashcache用于在以下情况下消除重复的哈希计算：
//计算最终摘要，将复杂性从O（n^2）降低到O（n）。
//此外，签名现在覆盖引用的未使用的输入值
//输出。这允许离线或硬件钱包计算确切数量
//除最终交易费外，还将被花费。在这种情况下
//钱包如果输入量无效，实际叹息将不同，导致
//生成的签名无效。
func calcWitnessSignatureHash(subScript []parsedOpcode, sigHashes *TxSigHashes,
	hashType SigHashType, tx *wire.MsgTx, idx int, amt int64) ([]byte, error) {

//作为健全性检查，请确保已传递事务的输入索引
//是有效的。
	if idx > len(tx.TxIn)-1 {
		return nil, fmt.Errorf("idx %d but %d txins", idx, len(tx.TxIn))
	}

//我们将在整个过程中使用这个缓冲区来增量计算
//此事务的签名哈希。
	var sigHash bytes.Buffer

//首先写出，然后对事务的版本号进行编码。
	var bVersion [4]byte
	binary.LittleEndian.PutUint32(bVersion[:], uint32(tx.Version))
	sigHash.Write(bVersion[:])

//接下来，写出序列可能预先计算的哈希值。
//所有输入的数目，以及所有输入的先前输出的哈希值
//输出。
	var zeroHash chainhash.Hash

//如果有人可以付费，那么我们可以使用缓存
//hashprevout，否则我们只为prev out写零。
	if hashType&SigHashAnyOneCanPay == 0 {
		sigHash.Write(sigHashes.HashPrevOuts[:])
	} else {
		sigHash.Write(zeroHash[:])
	}

//如果叹息声不是任何人都能支付的，单人的，或没有，使用
//缓存的哈希序列，否则写入
//哈希序列。
	if hashType&SigHashAnyOneCanPay == 0 &&
		hashType&sigHashMask != SigHashSingle &&
		hashType&sigHashMask != SigHashNone {
		sigHash.Write(sigHashes.HashSequence[:])
	} else {
		sigHash.Write(zeroHash[:])
	}

	txIn := tx.TxIn[idx]

//接下来，写下正在花费的输出点。
	sigHash.Write(txIn.PreviousOutPoint.Hash[:])
	var bIndex [4]byte
	binary.LittleEndian.PutUint32(bIndex[:], txIn.PreviousOutPoint.Index)
	sigHash.Write(bIndex[:])

	if isWitnessPubKeyHash(subScript) {
//p2wkh的脚本代码是的长度前缀变量
//接下来的25个字节，然后重新创建原始
//p2pkh pk脚本。
		sigHash.Write([]byte{0x19})
		sigHash.Write([]byte{OP_DUP})
		sigHash.Write([]byte{OP_HASH160})
		sigHash.Write([]byte{OP_DATA_20})
		sigHash.Write(subScript[1].data)
		sigHash.Write([]byte{OP_EQUALVERIFY})
		sigHash.Write([]byte{OP_CHECKSIG})
	} else {
//对于p2wsh输出和将来的输出，脚本代码是
//删除了所有代码分隔符的原始脚本，
//用var int长度前缀序列化。
		rawScript, _ := unparseScript(subScript)
		wire.WriteVarBytes(&sigHash, 0, rawScript)
	}

//接下来，添加输入量和被输入的序列号。
//签署。
	var bAmount [8]byte
	binary.LittleEndian.PutUint64(bAmount[:], uint64(amt))
	sigHash.Write(bAmount[:])
	var bSequence [4]byte
	binary.LittleEndian.PutUint32(bSequence[:], txIn.Sequence)
	sigHash.Write(bSequence[:])

//如果当前的签名模式不是单签名或无签名，那么我们可以
//重新使用预先生成的hashoutputs sighash片段。否则，
//我们将序列化并只将目标输出索引添加到签名中
//预图像。
	if hashType&SigHashSingle != SigHashSingle &&
		hashType&SigHashNone != SigHashNone {
		sigHash.Write(sigHashes.HashOutputs[:])
	} else if hashType&sigHashMask == SigHashSingle && idx < len(tx.TxOut) {
		var b bytes.Buffer
		wire.WriteTxOut(&b, 0, 0, tx.TxOut[idx])
		sigHash.Write(chainhash.DoubleHashB(b.Bytes()))
	} else {
		sigHash.Write(zeroHash[:])
	}

//最后，写出事务的锁时间和sig散列
//类型。
	var bLockTime [4]byte
	binary.LittleEndian.PutUint32(bLockTime[:], tx.LockTime)
	sigHash.Write(bLockTime[:])
	var bHashType [4]byte
	binary.LittleEndian.PutUint32(bHashType[:], uint32(hashType))
	sigHash.Write(bHashType[:])

	return chainhash.DoubleHashB(sigHash.Bytes()), nil
}

//CalcWitnessSightash为指定的输入计算Sightash摘要
//观察所需SIG哈希类型的目标事务。
func CalcWitnessSigHash(script []byte, sigHashes *TxSigHashes, hType SigHashType,
	tx *wire.MsgTx, idx int, amt int64) ([]byte, error) {

	parsedScript, err := parseScript(script)
	if err != nil {
		return nil, fmt.Errorf("cannot parse output script: %v", err)
	}

	return calcWitnessSignatureHash(parsedScript, sigHashes, hType, tx, idx,
		amt)
}

//shallowcopytx创建事务的浅副本，以便在
//正在计算签名哈希。它用于
//因为事务本身是一个深层次的拷贝，因此可以做更多的工作，并且
//分配的空间比需要的要大得多。
func shallowCopyTx(tx *wire.MsgTx) wire.MsgTx {
//作为额外的内存优化，使用连续的备份数组
//对于复制的输入和输出，并指向
//指向连续数组的指针。这样可以避免很多小问题
//分配。
	txCopy := wire.MsgTx{
		Version:  tx.Version,
		TxIn:     make([]*wire.TxIn, len(tx.TxIn)),
		TxOut:    make([]*wire.TxOut, len(tx.TxOut)),
		LockTime: tx.LockTime,
	}
	txIns := make([]wire.TxIn, len(tx.TxIn))
	for i, oldTxIn := range tx.TxIn {
		txIns[i] = *oldTxIn
		txCopy.TxIn[i] = &txIns[i]
	}
	txOuts := make([]wire.TxOut, len(tx.TxOut))
	for i, oldTxOut := range tx.TxOut {
		txOuts[i] = *oldTxOut
		txCopy.TxOut[i] = &txOuts[i]
	}
	return txCopy
}

//CalcSignatureHash将为当前脚本提供脚本和哈希类型
//引擎实例，计算用于签名和
//验证。
func CalcSignatureHash(script []byte, hashType SigHashType, tx *wire.MsgTx, idx int) ([]byte, error) {
	parsedScript, err := parseScript(script)
	if err != nil {
		return nil, fmt.Errorf("cannot parse output script: %v", err)
	}
	return calcSignatureHash(parsedScript, hashType, tx, idx), nil
}

//CalcSignatureHash将为当前脚本提供脚本和哈希类型
//引擎实例，计算用于签名和
//验证。
func calcSignatureHash(script []parsedOpcode, hashType SigHashType, tx *wire.MsgTx, idx int) []byte {
//sighashSingle签名类型只对相应的输入签名
//输出（与输入索引号相同的输出）。
//
//因为事务可以有比输出更多的输入，这意味着它
//不适合在没有
//相应的输出。
//
//原始Satoshi客户机实现中的bug意味着指定
//超出范围的索引导致签名哈希为1（作为
//uint256小endian）。最初的意图似乎是
//表示故障，但不幸的是，从未检查过，因此
//视为实际签名哈希。这辆马车的行为现在
//部分共识和一个硬叉将需要解决它。
//
//因此，创建事务的软件必须小心。
//利用叹息，因为它可以导致
//无效输入最终将签署一个
//散列值为1。这反过来为攻击者提供了一个机会
//巧妙地构建可以窃取提供的硬币的交易
//它们可以重用签名。
	if hashType&sigHashMask == SigHashSingle && idx >= len(tx.TxOut) {
		var hash chainhash.Hash
		hash[0] = 0x01
		return hash[:]
	}

//从脚本中删除opu codeseparator的所有实例。
	script = removeOpcode(script, OP_CODESEPARATOR)

//对事务进行简单复制，将脚本归零
//当前未处理的所有输入。
	txCopy := shallowCopyTx(tx)
	for i := range txCopy.TxIn {
		if i == idx {
//Unparsescript不能在此处失败，因为removeOpcode
//上面只返回有效的脚本。
			sigScript, _ := unparseScript(script)
			txCopy.TxIn[idx].SignatureScript = sigScript
		} else {
			txCopy.TxIn[i].SignatureScript = nil
		}
	}

	switch hashType & sigHashMask {
	case SigHashNone:
txCopy.TxOut = txCopy.TxOut[0:0] //空切片。
		for i := range txCopy.TxIn {
			if i != idx {
				txCopy.TxIn[i].Sequence = 0
			}
		}

	case SigHashSingle:
//将输出数组的大小调整为最大并包括请求的索引。
		txCopy.TxOut = txCopy.TxOut[:idx+1]

//除了电流输出以外，所有的都归零了。
		for i := 0; i < idx; i++ {
			txCopy.TxOut[i].Value = -1
			txCopy.TxOut[i].PkScript = nil
		}

//所有其他输入的序列也是0。
		for i := range txCopy.TxIn {
			if i != idx {
				txCopy.TxIn[i].Sequence = 0
			}
		}

	default:
//共识将未定义的哈希类型视为普通的sighashall
//用于哈希生成。
		fallthrough
	case SigHashOld:
		fallthrough
	case SigHashAll:
//这里没什么特别的。
	}
	if hashType&SigHashAnyOneCanPay != 0 {
		txCopy.TxIn = txCopy.TxIn[idx : idx+1]
	}

//最后一个哈希是两个序列化修改的
//事务和哈希类型（编码为4字节的小尾数
//附加值）。
	wbuf := bytes.NewBuffer(make([]byte, 0, txCopy.SerializeSizeStripped()+4))
	txCopy.SerializeNoWitness(wbuf)
	binary.Write(wbuf, binary.LittleEndian, hashType)
	return chainhash.DoubleHashB(wbuf.Bytes())
}

//assmallint返回传递的操作码，根据
//IssMallint（），作为整数。
func asSmallInt(op *opcode) int {
	if op.value == OP_0 {
		return 0
	}

	return int(op.value - (OP_1 - 1))
}

//getsigocount是用于计算
//pops提供的脚本中的签名操作。如果精确模式为
//请求，然后我们尝试计算多任务集的操作数
//否则我们使用最大值。
func getSigOpCount(pops []parsedOpcode, precise bool) int {
	nSigs := 0
	for i, pop := range pops {
		switch pop.opcode.value {
		case OP_CHECKSIG:
			fallthrough
		case OP_CHECKSIGVERIFY:
			nSigs++
		case OP_CHECKMULTISIG:
			fallthrough
		case OP_CHECKMULTISIGVERIFY:
//如果我们是准确的，那么寻找熟悉的
//多图像模式，目前我们所认识的是
//op_1-op_16表示公钥的数目。
//否则，我们最多使用20个。
			if precise && i > 0 &&
				pops[i-1].opcode.value >= OP_1 &&
				pops[i-1].opcode.value <= OP_16 {
				nSigs += asSmallInt(pops[i-1].opcode)
			} else {
				nSigs += MaxPubKeysPerMultiSig
			}
		default:
//不是SIGOP。
		}
	}

	return nSigs
}

//GetSigoCount提供签名操作数的快速计数
//在脚本中。checksig操作计数为1，check_multisig计数为20。
//如果脚本解析失败，则到失败点的计数为
//返回。
func GetSigOpCount(script []byte) int {
//不要检查错误，因为ParseScript返回已分析到错误的
//持久性有机污染物列表。
	pops, _ := parseScript(script)
	return getSigOpCount(pops, false)
}

//GetPrecisesigopCount返回中的签名操作数
//脚本PUBKEY。如果bip16为真，则可以在scriptsig中搜索
//付费脚本哈希脚本，以便找到精确的签名数
//事务中的操作。如果脚本无法解析，则计数
//返回到故障点。
func GetPreciseSigOpCount(scriptSig, scriptPubKey []byte, bip16 bool) int {
//不要检查错误，因为ParseScript返回已分析到错误的
//持久性有机污染物列表。
	pops, _ := parseScript(scriptPubKey)

//将非P2SH事务视为正常事务。
	if !(bip16 && isScriptHash(pops)) {
		return getSigOpCount(pops, true)
	}

//公钥脚本是付费脚本哈希，因此请分析签名
//获取最终项的脚本。未能完全分析计数的脚本
//作为0签名操作。
	sigPops, err := parseScript(scriptSig)
	if err != nil {
		return 0
	}

//签名脚本必须只将数据推送到堆栈，以便p2sh
//一对有效的，因此签名操作计数为0，而不是0
//案件。
	if !isPushOnly(sigPops) || len(sigPops) == 0 {
		return 0
	}

//p2sh脚本是签名脚本推送到的最后一项
//栈。当脚本为空时，没有签名操作。
	shScript := sigPops[len(sigPops)-1].data
	if len(shScript) == 0 {
		return 0
	}

//分析p2sh脚本，不要检查自parse script以来的错误
//返回已分析的POP错误列表和共识规则
//指令签名操作被计算到第一个解析
//失败。
	shPops, _ := parseScript(shScript)
	return getSigOpCount(shPops, true)
}

//GetWitnessSigoCount返回由
//将传递的pkscript与指定的见证或sigscript一起使用。
//与getPrecisesigopCount不同，此函数能够准确计算
//支出见证计划生成的签名操作数，以及
//嵌套的p2sh见证程序。如果脚本无法解析，则计数
//返回到故障点。
func GetWitnessSigOpCount(sigScript, pkScript []byte, witness wire.TxWitness) int {
//如果这是一个常规的证人程序，那么我们可以直接进行
//对其签名操作进行计数而不进行任何进一步的处理。
	if IsWitnessProgram(pkScript) {
		return getWitnessSigOps(pkScript, witness)
	}

//接下来，我们将检查sigscript，看看这是否是嵌套的p2sh
//见证程序。在这种情况下，sigscript实际上是
//p2wsh见证程序的数据推送。
	sigPops, err := parseScript(sigScript)
	if err != nil {
		return 0
	}
	if IsPayToScriptHash(pkScript) && isPushOnly(sigPops) &&
		IsWitnessProgram(sigScript[1:]) {
		return getWitnessSigOps(sigScript[1:], witness)
	}

	return 0
}

//GetWitnessSigops返回由
//把通过的证人计划花在通过的证人身上。确切的
//签名计数启发式由传递的版本修改
//见证程序。如果证人程序的版本不能
//提取，然后返回0作为sig op计数。
func getWitnessSigOps(pkScript []byte, witness wire.TxWitness) int {
//尝试提取见证程序版本。
	witnessVersion, witnessProgram, err := ExtractWitnessProgramInfo(
		pkScript,
	)
	if err != nil {
		return 0
	}

	switch witnessVersion {
	case 0:
		switch {
		case len(witnessProgram) == payToWitnessPubKeyHashDataSize:
			return 1
		case len(witnessProgram) == payToWitnessScriptHashDataSize &&
			len(witness) > 0:

			witnessScript := witness[len(witness)-1]
			pops, _ := parseScript(witnessScript)
			return getSigOpCount(pops, true)
		}
	}

	return 0
}

//isUnsendable返回传递的公钥脚本是否不可挂起，或者
//保证在执行时失败。这允许立即修剪输入
//当输入utxo集时。
func IsUnspendable(pkScript []byte) bool {
	pops, err := parseScript(pkScript)
	if err != nil {
		return true
	}

	return len(pops) > 0 && pops[0].opcode.value == OP_RETURN
}
