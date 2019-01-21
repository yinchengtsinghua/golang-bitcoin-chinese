
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
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

const (
//MaxDatacarriersize是推送中允许的最大字节数。
//要视为空数据事务的数据
	MaxDataCarrierSize = 80

//StandardVerifyFlags是在
//执行事务脚本以强制执行其他检查
//需要将脚本视为标准脚本。这些支票
//帮助减少与事务可扩展性相关的问题以及
//允许付费脚本哈希事务。注意这些标志是
//不同于共识规则的要求
//更严格。
//
//TODO:此定义不属于此处。它属于一项政策
//包裹。
	StandardVerifyFlags = ScriptBip16 |
		ScriptVerifyDERSignatures |
		ScriptVerifyStrictEncoding |
		ScriptVerifyMinimalData |
		ScriptStrictMultiSig |
		ScriptDiscourageUpgradableNops |
		ScriptVerifyCleanStack |
		ScriptVerifyNullFail |
		ScriptVerifyCheckLockTimeVerify |
		ScriptVerifyCheckSequenceVerify |
		ScriptVerifyLowS |
		ScriptStrictMultiSig |
		ScriptVerifyWitness |
		ScriptVerifyDiscourageUpgradeableWitnessProgram |
		ScriptVerifyMinimalIf |
		ScriptVerifyWitnessPubKeyType
)

//scriptclass是标准脚本类型列表的枚举。
type ScriptClass byte

//区块链中已知的脚本支付类。
const (
NonStandardTy         ScriptClass = iota //没有任何公认的形式。
PubKeyTy                                 //付钱吧。
PubKeyHashTy                             //支付公钥哈希。
WitnessV0PubKeyHashTy                    //付款见证公钥哈希。
ScriptHashTy                             //付费脚本哈希。
WitnessV0ScriptHashTy                    //付费见证脚本哈希。
MultiSigTy                               //多重签名。
NullDataTy                               //仅空数据（可证明可删减）。
)

//scriptclasstoname包含了人类可读的字符串，这些字符串描述了
//脚本类。
var scriptClassToName = []string{
	NonStandardTy:         "nonstandard",
	PubKeyTy:              "pubkey",
	PubKeyHashTy:          "pubkeyhash",
	WitnessV0PubKeyHashTy: "witness_v0_keyhash",
	ScriptHashTy:          "scripthash",
	WitnessV0ScriptHashTy: "witness_v0_scripthash",
	MultiSigTy:            "multisig",
	NullDataTy:            "nulldata",
}

//字符串通过返回
//枚举脚本类。如果枚举无效，则“invalid”将是
//返回。
func (t ScriptClass) String() string {
	if int(t) > len(scriptClassToName) || int(t) < 0 {
		return "Invalid"
	}
	return scriptClassToName[t]
}

//如果传递的脚本是支付给pubkey事务，则is pubkey返回true，
//否则为假。
func isPubkey(pops []parsedOpcode) bool {
//有效的pubkey为33或65字节。
	return len(pops) == 2 &&
		(len(pops[0].data) == 33 || len(pops[0].data) == 65) &&
		pops[1].opcode.value == OP_CHECKSIG
}

//如果传递的脚本是支付给pubkey散列，则is pubkey hash返回true
//事务，否则为false。
func isPubkeyHash(pops []parsedOpcode) bool {
	return len(pops) == 5 &&
		pops[0].opcode.value == OP_DUP &&
		pops[1].opcode.value == OP_HASH160 &&
		pops[2].opcode.value == OP_DATA_20 &&
		pops[3].opcode.value == OP_EQUALVERIFY &&
		pops[4].opcode.value == OP_CHECKSIG

}

//如果传递的脚本是多任务事务，is multisig将返回true；如果传递的脚本是多任务事务，则返回false
//否则。
func isMultiSig(pops []parsedOpcode) bool {
//绝对最小值为1 pubkey：
//op_0/op_1-16<pubkey>op_1 op_checkmultisig
	l := len(pops)
	if l < 4 {
		return false
	}
	if !isSmallInt(pops[0].opcode) {
		return false
	}
	if !isSmallInt(pops[l-2].opcode) {
		return false
	}
	if pops[l-1].opcode.value != OP_CHECKMULTISIG {
		return false
	}

//验证指定的pubkeys数是否与实际数匹配
//提供了个公钥。
	if l-2-1 != asSmallInt(pops[l-2].opcode) {
		return false
	}

	for _, pop := range pops[1 : l-2] {
//有效的pubkey为33或65字节。
		if len(pop.data) != 33 && len(pop.data) != 65 {
			return false
		}
	}
	return true
}

//如果传递的脚本是空数据事务，则IsNullData返回true，
//否则为假。
func isNullData(pops []parsedOpcode) bool {
//nullData事务可以是单个opu返回，也可以是
//op_返回smalldata（其中smalldata是向上推送的数据）
//maxdatacarriersize字节）。
	l := len(pops)
	if l == 1 && pops[0].opcode.value == OP_RETURN {
		return true
	}

	return l == 2 &&
		pops[0].opcode.value == OP_RETURN &&
		(isSmallInt(pops[1].opcode) || pops[1].opcode.value <=
			OP_PUSHDATA4) &&
		len(pops[1].data) <= MaxDataCarrierSize
}

//scriptType返回从已知的
//标准类型。
func typeOfScript(pops []parsedOpcode) ScriptClass {
	if isPubkey(pops) {
		return PubKeyTy
	} else if isPubkeyHash(pops) {
		return PubKeyHashTy
	} else if isWitnessPubKeyHash(pops) {
		return WitnessV0PubKeyHashTy
	} else if isScriptHash(pops) {
		return ScriptHashTy
	} else if isWitnessScriptHash(pops) {
		return WitnessV0ScriptHashTy
	} else if isMultiSig(pops) {
		return MultiSigTy
	} else if isNullData(pops) {
		return NullDataTy
	}
	return NonStandardTy
}

//GetScriptClass返回传递的脚本的类。
//
//当脚本不分析时，将返回非标准的。
func GetScriptClass(script []byte) ScriptClass {
	pops, err := parseScript(script)
	if err != nil {
		return NonStandardTy
	}
	return typeOfScript(pops)
}

//ExpectedInputs返回脚本所需的参数数。
//如果脚本类型未知，无法确定数字
//然后返回-1。我们是一个内部函数，因此假定
//是真正的持久性有机污染物类（因此我们可以假设确定的事情
//同时找出类型）。
func expectedInputs(pops []parsedOpcode, class ScriptClass) int {
	switch class {
	case PubKeyTy:
		return 1

	case PubKeyHashTy:
		return 2

	case WitnessV0PubKeyHashTy:
		return 2

	case ScriptHashTy:
//不包括脚本。由呼叫方处理。
		return 1

	case WitnessV0ScriptHashTy:
//不包括脚本。由呼叫方处理。
		return 1

	case MultiSigTy:
//标准multisig有一个推小数字
//信号和键数。检查第一次推送指令
//查看需要多少参数。typeofscript已经
//检查过这个，所以我们知道它会是一个小整数。
//原来的比特币错误，在这里Op-Checkmultisig弹出一个
//堆栈中的附加项，添加一个额外的预期输入
//需要补偿的额外推力。
		return asSmallInt(pops[0].opcode) + 1

	case NullDataTy:
		fallthrough
	default:
		return -1
	}
}

//scriptinfo存储有关脚本对的信息，该脚本对由
//CalcScriptInfo。
type ScriptInfo struct {
//pkscriptClass是公钥脚本的类，并且是等效的
//调用GetScriptClass。
	PkScriptClass ScriptClass

//NumInputs是公钥脚本提供的输入数。
	NumInputs int

//ExpectedInputs是签名所需的输出数。
//脚本和任何付费脚本哈希脚本。如果
//未知的。
	ExpectedInputs int

//sigops是脚本对中的签名操作数。
	SigOps int
}

//CalcScriptinfo返回一个提供有关所提供脚本的数据的结构
//一对。如果一对在某种程度上无效，以致它们不能
//分析，例如，如果它们不解析或者pkscript不是一个push-only
//脚本
func CalcScriptInfo(sigScript, pkScript []byte, witness wire.TxWitness,
	bip16, segwit bool) (*ScriptInfo, error) {

	sigPops, err := parseScript(sigScript)
	if err != nil {
		return nil, err
	}

	pkPops, err := parseScript(pkScript)
	if err != nil {
		return nil, err
	}

//只推sigscript没有任何意义。
	si := new(ScriptInfo)
	si.PkScriptClass = typeOfScript(pkPops)

//不能有不只是推送数据的签名脚本。
	if !isPushOnly(sigPops) {
		return nil, scriptError(ErrNotPushOnly,
			"signature script is not push only")
	}

	si.ExpectedInputs = expectedInputs(pkPops, si.PkScriptClass)

	switch {
//计算sigops，并考虑到pay-to-script散列。
	case si.PkScriptClass == ScriptHashTy && bip16 && !segwit:
//pay-to-hash脚本是
//签名脚本。
		script := sigPops[len(sigPops)-1].data
		shPops, err := parseScript(script)
		if err != nil {
			return nil, err
		}

		shInputs := expectedInputs(shPops, typeOfScript(shPops))
		if shInputs == -1 {
			si.ExpectedInputs = -1
		} else {
			si.ExpectedInputs += shInputs
		}
		si.SigOps = getSigOpCount(shPops, true)

//所有被推到堆栈的条目（或是opu reserved和exec
//将失败。
		si.NumInputs = len(sigPops)

//如果segwit是活动的，并且这是一个常规的p2wkh输出，那么我们将
//本质上，将脚本视为p2pkh输出。
	case si.PkScriptClass == WitnessV0PubKeyHashTy && segwit:

		si.SigOps = GetWitnessSigOpCount(sigScript, pkScript, witness)
		si.NumInputs = len(witness)

//我们将尝试检测嵌套的p2sh情况，以便准确地
//统计所涉及的签名操作。
	case si.PkScriptClass == ScriptHashTy &&
		IsWitnessProgram(sigScript[1:]) && bip16 && segwit:

//从sigscript中提取推送的见证程序，以便
//可以确定预期输入的数量。
		pkPops, _ := parseScript(sigScript[1:])
		shInputs := expectedInputs(pkPops, typeOfScript(pkPops))
		if shInputs == -1 {
			si.ExpectedInputs = -1
		} else {
			si.ExpectedInputs += shInputs
		}

		si.SigOps = GetWitnessSigOpCount(sigScript, pkScript, witness)

		si.NumInputs = len(witness)
		si.NumInputs += len(sigPops)

//如果segwit是活动的，并且这是p2wsh输出，那么我们需要
//检查见证脚本以生成准确的脚本信息。
	case si.PkScriptClass == WitnessV0ScriptHashTy && segwit:
//证人脚本是证人的最终元素
//栈。
		witnessScript := witness[len(witness)-1]
		pops, _ := parseScript(witnessScript)

		shInputs := expectedInputs(pops, typeOfScript(pops))
		if shInputs == -1 {
			si.ExpectedInputs = -1
		} else {
			si.ExpectedInputs += shInputs
		}

		si.SigOps = GetWitnessSigOpCount(sigScript, pkScript, witness)
		si.NumInputs = len(witness)

	default:
		si.SigOps = getSigOpCount(pkPops, true)

//所有被推到堆栈的条目（或是opu reserved和exec
//将失败。
		si.NumInputs = len(sigPops)
	}

	return si, nil
}

//calcmulsigstats返回公钥和签名的数目
//多签名事务脚本。传递的脚本必须已经
//已知是多签名脚本。
func CalcMultiSigStats(script []byte) (int, int, error) {
	pops, err := parseScript(script)
	if err != nil {
		return 0, 0, err
	}

//多签名脚本的模式如下：
//数字标记Pubkey Pubkey…num_pubkeys op_checkmultisig
//因此，签名数是堆栈上最早的项
//PubKeys的数目是倒数第二个。另外，绝对
//多签名脚本的最小值为1 pubkey，因此至少为4
//项必须位于堆栈上，根据：
//Op_1 Pubkey Op_1 Op_Checkmultisig
	if len(pops) < 4 {
		str := fmt.Sprintf("script %x is not a multisig script", script)
		return 0, 0, scriptError(ErrNotMultisigScript, str)
	}

	numSigs := asSmallInt(pops[0].opcode)
	numPubKeys := asSmallInt(pops[len(pops)-2].opcode)
	return numPubKeys, numSigs, nil
}

//paytopubkeyhashscript创建一个新脚本来支付事务
//输出到一个20字节的pubkey散列。输入应该是有效的
//搞砸。
func payToPubKeyHashScript(pubKeyHash []byte) ([]byte, error) {
	return NewScriptBuilder().AddOp(OP_DUP).AddOp(OP_HASH160).
		AddData(pubKeyHash).AddOp(OP_EQUALVERIFY).AddOp(OP_CHECKSIG).
		Script()
}

//PayToWitnessPubKeyHashScript创建新脚本以支付到版本0
//Pubkey哈希见证程序。传递的哈希应该是有效的。
func payToWitnessPubKeyHashScript(pubKeyHash []byte) ([]byte, error) {
	return NewScriptBuilder().AddOp(OP_0).AddData(pubKeyHash).Script()
}

//paytoscripthashscript创建一个新脚本，用于将事务输出支付给
//脚本哈希。输入应该是有效的哈希。
func payToScriptHashScript(scriptHash []byte) ([]byte, error) {
	return NewScriptBuilder().AddOp(OP_HASH160).AddData(scriptHash).
		AddOp(OP_EQUAL).Script()
}

//PayToWitnessPubKeyHashScript创建新脚本以支付到版本0
//编写哈希见证程序脚本。传递的哈希应该是有效的。
func payToWitnessScriptHashScript(scriptHash []byte) ([]byte, error) {
	return NewScriptBuilder().AddOp(OP_0).AddData(scriptHash).Script()
}

//paytopubkeyscript创建一个新脚本，将事务输出支付给
//公钥。输入应该是有效的pubkey。
func payToPubKeyScript(serializedPubKey []byte) ([]byte, error) {
	return NewScriptBuilder().AddData(serializedPubKey).
		AddOp(OP_CHECKSIG).Script()
}

//paytoaddrscript创建一个新脚本，将事务输出支付给
//指定地址。
func PayToAddrScript(addr btcutil.Address) ([]byte, error) {
	const nilAddrErrStr = "unable to generate payment script for nil address"

	switch addr := addr.(type) {
	case *btcutil.AddressPubKeyHash:
		if addr == nil {
			return nil, scriptError(ErrUnsupportedAddress,
				nilAddrErrStr)
		}
		return payToPubKeyHashScript(addr.ScriptAddress())

	case *btcutil.AddressScriptHash:
		if addr == nil {
			return nil, scriptError(ErrUnsupportedAddress,
				nilAddrErrStr)
		}
		return payToScriptHashScript(addr.ScriptAddress())

	case *btcutil.AddressPubKey:
		if addr == nil {
			return nil, scriptError(ErrUnsupportedAddress,
				nilAddrErrStr)
		}
		return payToPubKeyScript(addr.ScriptAddress())

	case *btcutil.AddressWitnessPubKeyHash:
		if addr == nil {
			return nil, scriptError(ErrUnsupportedAddress,
				nilAddrErrStr)
		}
		return payToWitnessPubKeyHashScript(addr.ScriptAddress())
	case *btcutil.AddressWitnessScriptHash:
		if addr == nil {
			return nil, scriptError(ErrUnsupportedAddress,
				nilAddrErrStr)
		}
		return payToWitnessScriptHashScript(addr.ScriptAddress())
	}

	str := fmt.Sprintf("unable to generate payment script for unsupported "+
		"address type %T", addr)
	return nil, scriptError(ErrUnsupportedAddress, str)
}

//nulldatascript创建一个可证明的可删减脚本，其中包含op_返回
//然后是传递的数据。错误代码为errtoomuchnulldata的错误
//如果传递的数据长度超过MaxDataCarrierize，则返回。
func NullDataScript(data []byte) ([]byte, error) {
	if len(data) > MaxDataCarrierSize {
		str := fmt.Sprintf("data size %d is larger than max "+
			"allowed size %d", len(data), MaxDataCarrierSize)
		return nil, scriptError(ErrTooMuchNullData, str)
	}

	return NewScriptBuilder().AddOp(OP_RETURN).AddData(data).Script()
}

//multisigscript返回多签名兑换的有效脚本，其中
//需要pubkeys中的密钥才能签署事务
//为了成功。错误代码为errtoomanyRequiredisigs的错误将是
//如果nRequired大于提供的密钥数，则返回。
func MultiSigScript(pubkeys []*btcutil.AddressPubKey, nrequired int) ([]byte, error) {
	if len(pubkeys) < nrequired {
		str := fmt.Sprintf("unable to generate multisig script with "+
			"%d required signatures when there are only %d public "+
			"keys available", nrequired, len(pubkeys))
		return nil, scriptError(ErrTooManyRequiredSigs, str)
	}

	builder := NewScriptBuilder().AddInt64(int64(nrequired))
	for _, key := range pubkeys {
		builder.AddData(key.ScriptAddress())
	}
	builder.AddInt64(int64(len(pubkeys)))
	builder.AddOp(OP_CHECKMULTISIG)

	return builder.Script()
}

//pushed data返回包含找到的任何推送数据的字节片数组
//在传递的脚本中。这包括op_0，但不包括op_1-op_16。
func PushedData(script []byte) ([][]byte, error) {
	pops, err := parseScript(script)
	if err != nil {
		return nil, err
	}

	var data [][]byte
	for _, pop := range pops {
		if pop.data != nil {
			data = append(data, pop.data)
		} else if pop.opcode.value == OP_0 {
			data = append(data, nil)
		}
	}
	return data, nil
}

//ExtracpkScriptAddrs返回脚本类型、地址和必需的
//与传递的pkscript关联的签名。注意它只适用于
//“标准”事务脚本类型。任何数据，如公钥
//结果中省略了无效。
func ExtractPkScriptAddrs(pkScript []byte, chainParams *chaincfg.Params) (ScriptClass, []btcutil.Address, int, error) {
	var addrs []btcutil.Address
	var requiredSigs int

//如果脚本没有，则没有有效的地址或必需的签名
//解析。
	pops, err := parseScript(pkScript)
	if err != nil {
		return NonStandardTy, nil, 0, err
	}

	scriptClass := typeOfScript(pops)
	switch scriptClass {
	case PubKeyHashTy:
//pay-to-pubkey哈希脚本的格式为：
//op_dup op_hash160<hash>op_equalverify op_checksig
//因此，pubkey散列是堆栈上的第三个项。
//如果pubkey散列由于某种原因无效，则跳过它。
		requiredSigs = 1
		addr, err := btcutil.NewAddressPubKeyHash(pops[2].data,
			chainParams)
		if err == nil {
			addrs = append(addrs, addr)
		}

	case WitnessV0PubKeyHashTy:
//支付见证pubkey哈希脚本的格式为thw：
//op_0<20字节哈希>
//因此，pubkey散列是堆栈上的第二个项。
//如果pubkey散列由于某种原因无效，则跳过它。
		requiredSigs = 1
		addr, err := btcutil.NewAddressWitnessPubKeyHash(pops[1].data,
			chainParams)
		if err == nil {
			addrs = append(addrs, addr)
		}

	case PubKeyTy:
//支付到发布密钥脚本的格式为：
//<pubkey>op_checksig
//因此pubkey是堆栈上的第一个项。
//如果pubkey由于某种原因无效，则跳过它。
		requiredSigs = 1
		addr, err := btcutil.NewAddressPubKey(pops[0].data, chainParams)
		if err == nil {
			addrs = append(addrs, addr)
		}

	case ScriptHashTy:
//付费脚本哈希脚本的格式为：
//op_hash160<scripthash>op_equal
//因此，脚本散列是堆栈上的第二项。
//如果脚本哈希因某种原因无效，则跳过它。
		requiredSigs = 1
		addr, err := btcutil.NewAddressScriptHashFromHash(pops[1].data,
			chainParams)
		if err == nil {
			addrs = append(addrs, addr)
		}

	case WitnessV0ScriptHashTy:
//付费见证脚本哈希脚本的格式为：
//op_0<32字节哈希>
//因此，脚本散列是堆栈上的第二项。
//如果脚本哈希因某种原因无效，则跳过它。
		requiredSigs = 1
		addr, err := btcutil.NewAddressWitnessScriptHash(pops[1].data,
			chainParams)
		if err == nil {
			addrs = append(addrs, addr)
		}

	case MultiSigTy:
//多签名脚本的形式如下：
//<numsigs><pubkey><pubkey><pubkey>。<numpubkeys>op_checkmultisig
//因此，所需签名的数量是第一项
//在堆栈上，公钥的数目是倒数第二个。
//堆栈上的项。
		requiredSigs = asSmallInt(pops[0].opcode)
		numPubKeys := asSmallInt(pops[len(pops)-2].opcode)

//在跳过任何无效的公钥时提取公钥。
		addrs = make([]btcutil.Address, 0, numPubKeys)
		for i := 0; i < numPubKeys; i++ {
			addr, err := btcutil.NewAddressPubKey(pops[i+1].data,
				chainParams)
			if err == nil {
				addrs = append(addrs, addr)
			}
		}

	case NullDataTy:
//空数据事务没有地址或是必需的
//签名。

	case NonStandardTy:
//不要尝试提取的地址或所需签名
//非标准交易。
	}

	return scriptClass, addrs, requiredSigs, nil
}

//atomicswapadapush包含原子交换合同中的数据推送。
type AtomicSwapDataPushes struct {
	RecipientHash160 [20]byte
	RefundHash160    [20]byte
	SecretHash       [32]byte
	SecretSize       int64
	LockTime         int64
}

//extractatomicswapdatapushes返回来自原子交换的数据推送
//合同。如果脚本不是原子交换合同，
//抽提物微粒子弹丸返回（零，零）。返回非零错误
//用于不可分析的脚本。
//
//注意：DCRD不认为原子交换是标准脚本类型
//mempool策略，应与p2sh一起使用。原子交换格式也是
//希望将来更改为使用更安全的哈希函数。
//
//由于API限制，此函数仅在txscript包中定义。
//它防止调用方使用txscript解析非标准脚本。
func ExtractAtomicSwapDataPushes(version uint16, pkScript []byte) (*AtomicSwapDataPushes, error) {
	pops, err := parseScript(pkScript)
	if err != nil {
		return nil, err
	}

	if len(pops) != 20 {
		return nil, nil
	}
	isAtomicSwap := pops[0].opcode.value == OP_IF &&
		pops[1].opcode.value == OP_SIZE &&
		canonicalPush(pops[2]) &&
		pops[3].opcode.value == OP_EQUALVERIFY &&
		pops[4].opcode.value == OP_SHA256 &&
		pops[5].opcode.value == OP_DATA_32 &&
		pops[6].opcode.value == OP_EQUALVERIFY &&
		pops[7].opcode.value == OP_DUP &&
		pops[8].opcode.value == OP_HASH160 &&
		pops[9].opcode.value == OP_DATA_20 &&
		pops[10].opcode.value == OP_ELSE &&
		canonicalPush(pops[11]) &&
		pops[12].opcode.value == OP_CHECKLOCKTIMEVERIFY &&
		pops[13].opcode.value == OP_DROP &&
		pops[14].opcode.value == OP_DUP &&
		pops[15].opcode.value == OP_HASH160 &&
		pops[16].opcode.value == OP_DATA_20 &&
		pops[17].opcode.value == OP_ENDIF &&
		pops[18].opcode.value == OP_EQUALVERIFY &&
		pops[19].opcode.value == OP_CHECKSIG
	if !isAtomicSwap {
		return nil, nil
	}

	pushes := new(AtomicSwapDataPushes)
	copy(pushes.SecretHash[:], pops[5].data)
	copy(pushes.RecipientHash160[:], pops[9].data)
	copy(pushes.RefundHash160[:], pops[16].data)
	if pops[2].data != nil {
		locktime, err := makeScriptNum(pops[2].data, true, 5)
		if err != nil {
			return nil, nil
		}
		pushes.SecretSize = int64(locktime)
	} else if op := pops[2].opcode; isSmallInt(op) {
		pushes.SecretSize = int64(asSmallInt(op))
	} else {
		return nil, nil
	}
	if pops[11].data != nil {
		locktime, err := makeScriptNum(pops[11].data, true, 5)
		if err != nil {
			return nil, nil
		}
		pushes.LockTime = int64(locktime)
	} else if op := pops[11].opcode; isSmallInt(op) {
		pushes.LockTime = int64(asSmallInt(op))
	} else {
		return nil, nil
	}
	return pushes, nil
}
