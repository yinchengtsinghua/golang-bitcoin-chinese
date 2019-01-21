
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2015 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package txscript

import (
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

//rawtxinwitnessSignature返回输入的序列化ECDA签名
//给定事务的IDX，附加了哈希类型。这个
//函数与rawtxinsignature相同，但生成的签名
//标志着BIP0143中定义的新的叹息消化。
func RawTxInWitnessSignature(tx *wire.MsgTx, sigHashes *TxSigHashes, idx int,
	amt int64, subScript []byte, hashType SigHashType,
	key *btcec.PrivateKey) ([]byte, error) {

	parsedScript, err := parseScript(subScript)
	if err != nil {
		return nil, fmt.Errorf("cannot parse output script: %v", err)
	}

	hash, err := calcWitnessSignatureHash(parsedScript, sigHashes, hashType, tx,
		idx, amt)
	if err != nil {
		return nil, err
	}

	signature, err := key.Sign(hash)
	if err != nil {
		return nil, fmt.Errorf("cannot sign tx input: %s", err)
	}

	return append(signature.Serialize(), byte(hashType)), nil
}

//见证签名为Tx创建一个输入见证堆栈，以便花费发送的BTC
//从以前的输出到使用p2wkh脚本的privkey的所有者
//模板。传递的事务必须包含所有输入和输出
//由传递的哈希类型指定。生成的签名观察新的
//bip0143中定义的事务摘要算法。
func WitnessSignature(tx *wire.MsgTx, sigHashes *TxSigHashes, idx int, amt int64,
	subscript []byte, hashType SigHashType, privKey *btcec.PrivateKey,
	compress bool) (wire.TxWitness, error) {

	sig, err := RawTxInWitnessSignature(tx, sigHashes, idx, amt, subscript,
		hashType, privKey)
	if err != nil {
		return nil, err
	}

	pk := (*btcec.PublicKey)(&privKey.PublicKey)
	var pkData []byte
	if compress {
		pkData = pk.SerializeCompressed()
	} else {
		pkData = pk.SerializeUncompressed()
	}

//见证脚本实际上是一个堆栈，因此我们返回一个字节数组
//在这里切片，而不是单字节切片。
	return wire.TxWitness{sig, pkData}, nil
}

//rawtxinsignature返回的输入idx的序列化ecdsa签名
//附加了hashtype的给定事务。
func RawTxInSignature(tx *wire.MsgTx, idx int, subScript []byte,
	hashType SigHashType, key *btcec.PrivateKey) ([]byte, error) {

	hash, err := CalcSignatureHash(subScript, hashType, tx, idx)
	if err != nil {
		return nil, err
	}
	signature, err := key.Sign(hash)
	if err != nil {
		return nil, fmt.Errorf("cannot sign tx input: %s", err)
	}

	return append(signature.Serialize(), byte(hashType)), nil
}

//signature script为tx创建一个输入签名脚本，用于花费发送的btc
//从以前的输出到privkey的所有者。Tx必须包括所有
//事务输入和输出，但是允许填写txin脚本
//或者是空的。计算返回的脚本以用作idx'th txin
//sigscript for tx.subscript是正在使用的上一个输出的pkscript
//作为IDX的输入。privkey在压缩或
//基于压缩的未压缩格式。此格式必须与同一格式匹配
//用于生成付款地址，否则脚本验证将失败。
func SignatureScript(tx *wire.MsgTx, idx int, subscript []byte, hashType SigHashType, privKey *btcec.PrivateKey, compress bool) ([]byte, error) {
	sig, err := RawTxInSignature(tx, idx, subscript, hashType, privKey)
	if err != nil {
		return nil, err
	}

	pk := (*btcec.PublicKey)(&privKey.PublicKey)
	var pkData []byte
	if compress {
		pkData = pk.SerializeCompressed()
	} else {
		pkData = pk.SerializeUncompressed()
	}

	return NewScriptBuilder().AddData(sig).AddData(pkData).Script()
}

func p2pkSignatureScript(tx *wire.MsgTx, idx int, subScript []byte, hashType SigHashType, privKey *btcec.PrivateKey) ([]byte, error) {
	sig, err := RawTxInSignature(tx, idx, subScript, hashType, privKey)
	if err != nil {
		return nil, err
	}

	return NewScriptBuilder().AddData(sig).Script()
}

//将提供的multisig脚本中的多个输出签名为
//可能的。它返回生成的脚本和一个布尔值，如果脚本满足
//合同（即提供要求的签名）。因为有争议
//法律上不能签署任何输出，没有返回错误。
func signMultiSig(tx *wire.MsgTx, idx int, subScript []byte, hashType SigHashType,
	addresses []btcutil.Address, nRequired int, kdb KeyDB) ([]byte, bool) {
//我们从一个操作错误开始，围绕（现在的标准）
//但是在引用实现中，会导致
//op_checkmultisig结束。
	builder := NewScriptBuilder().AddOp(OP_FALSE)
	signed := 0
	for _, addr := range addresses {
		key, _, err := kdb.GetKey(addr)
		if err != nil {
			continue
		}
		sig, err := RawTxInSignature(tx, idx, subScript, hashType, key)
		if err != nil {
			continue
		}

		builder.AddData(sig)
		signed++
		if signed == nRequired {
			break
		}

	}

	script, _ := builder.Script()
	return script, signed == nRequired
}

func sign(chainParams *chaincfg.Params, tx *wire.MsgTx, idx int,
	subScript []byte, hashType SigHashType, kdb KeyDB, sdb ScriptDB) ([]byte,
	ScriptClass, []btcutil.Address, int, error) {

	class, addresses, nrequired, err := ExtractPkScriptAddrs(subScript,
		chainParams)
	if err != nil {
		return nil, NonStandardTy, nil, 0, err
	}

	switch class {
	case PubKeyTy:
//地址查找键
		key, _, err := kdb.GetKey(addresses[0])
		if err != nil {
			return nil, class, nil, 0, err
		}

		script, err := p2pkSignatureScript(tx, idx, subScript, hashType,
			key)
		if err != nil {
			return nil, class, nil, 0, err
		}

		return script, class, addresses, nrequired, nil
	case PubKeyHashTy:
//地址查找键
		key, compressed, err := kdb.GetKey(addresses[0])
		if err != nil {
			return nil, class, nil, 0, err
		}

		script, err := SignatureScript(tx, idx, subScript, hashType,
			key, compressed)
		if err != nil {
			return nil, class, nil, 0, err
		}

		return script, class, addresses, nrequired, nil
	case ScriptHashTy:
		script, err := sdb.GetScript(addresses[0])
		if err != nil {
			return nil, class, nil, 0, err
		}

		return script, class, addresses, nrequired, nil
	case MultiSigTy:
		script, _ := signMultiSig(tx, idx, subScript, hashType,
			addresses, nrequired, kdb)
		return script, class, addresses, nrequired, nil
	case NullDataTy:
		return nil, class, nil, 0,
			errors.New("can't sign NULLDATA transactions")
	default:
		return nil, class, nil, 0,
			errors.New("can't sign unknown transactions")
	}
}

//合并脚本合并sigscript和prevscript，假定它们都是
//pkscript支出tx输出idx的部分解决方案。类，地址
//而nRequired是从pkscript中提取地址的结果。
//返回值是两个脚本的最佳合并。调用此
//地址、类和非必需与pkscript不匹配的函数为
//一种错误，会导致未定义的行为。
func mergeScripts(chainParams *chaincfg.Params, tx *wire.MsgTx, idx int,
	pkScript []byte, class ScriptClass, addresses []btcutil.Address,
	nRequired int, sigScript, prevScript []byte) []byte {

//TODO:这里的scripthash和multisig路径太多了
//效率低下，因为它们将重新计算已知数据。
//一些内部重构可能会使这避免不必要的
//额外计算。
	switch class {
	case ScriptHashTy:
//删除脚本中的最后一个推送，然后重复。
//这可能会大大降低效率。
		sigPops, err := parseScript(sigScript)
		if err != nil || len(sigPops) == 0 {
			return prevScript
		}
		prevPops, err := parseScript(prevScript)
		if err != nil || len(prevPops) == 0 {
			return sigScript
		}

//假设sigpops中的脚本是正确的，我们只是
//做到了。
		script := sigPops[len(sigPops)-1].data

//我们已经知道这个信息在堆栈的某个位置。
		class, addresses, nrequired, _ :=
			ExtractPkScriptAddrs(script, chainParams)

//重新生成脚本。
		sigScript, _ := unparseScript(sigPops)
		prevScript, _ := unparseScript(prevPops)

//合并
		mergedScript := mergeScripts(chainParams, tx, idx, script,
			class, addresses, nrequired, sigScript, prevScript)

//重新应用脚本并返回结果。
		builder := NewScriptBuilder()
		builder.AddOps(mergedScript)
		builder.AddData(script)
		finalScript, _ := builder.Script()
		return finalScript
	case MultiSigTy:
		return mergeMultiSig(tx, idx, addresses, nRequired, pkScript,
			sigScript, prevScript)

//合并除multig之外的任何内容实际上都没有意义
//和scripthash（因为它可以包含multisig）。其他一切
//具有零签名、不能使用或只有一个签名
//是否存在。另外两个案件处理完毕
//上面。在这里的冲突案例中，我们假设最长的是
//正确（这与引用实现的行为匹配）。
	default:
		if len(sigScript) > len(prevScript) {
			return sigScript
		}
		return prevScript
	}
}

//mergemultisig结合了两个签名脚本sigscript和prevscript
//它们都为Tx地址的输出IDX中的pkscript提供签名。
//需要的结果应该是从
//PKScript。因为这个函数是内部的，所以我们假定参数
//来自内部的其他功能，因此都与
//如果本合同被违反，双方的行为是不明确的。
func mergeMultiSig(tx *wire.MsgTx, idx int, addresses []btcutil.Address,
	nRequired int, pkScript, sigScript, prevScript []byte) []byte {

//这是一个仅限内部的函数，我们已经分析了此脚本
//对于multisig（这是我们的方法），如果失败了，那么
//所有的假设都被打破了，谁知道是哪条路？
	pkPops, _ := parseScript(pkScript)

	sigPops, err := parseScript(sigScript)
	if err != nil || len(sigPops) == 0 {
		return prevScript
	}

	prevPops, err := parseScript(prevScript)
	if err != nil || len(prevPops) == 0 {
		return sigScript
	}

//方便功能，避免重复。
	extractSigs := func(pops []parsedOpcode, sigs [][]byte) [][]byte {
		for _, pop := range pops {
			if len(pop.data) != 0 {
				sigs = append(sigs, pop.data)
			}
		}
		return sigs
	}

	possibleSigs := make([][]byte, 0, len(sigPops)+len(prevPops))
	possibleSigs = extractSigs(sigPops, possibleSigs)
	possibleSigs = extractSigs(prevPops, possibleSigs)

//现在我们需要将签名与pubkeys匹配，这是
//这是为了验证它们并将其与pubkey匹配吗？
//这证实了这一点。然后我们可以按顺序浏览地址
//来构建我们的脚本。任何不解析或不验证我们
//扔掉。
	addrToSig := make(map[string][]byte)
sigLoop:
	for _, sig := range possibleSigs {

//不能有至少没有的有效签名
//hashtype，在实践中它甚至比这个长。但是
//下次再查。
		if len(sig) < 1 {
			continue
		}
		tSig := sig[:len(sig)-1]
		hashType := SigHashType(sig[len(sig)-1])

		pSig, err := btcec.ParseDERSignature(tSig, btcec.S256())
		if err != nil {
			continue
		}

//因为散列类型可能会有所不同，所以我们每轮都要这样做。
//在签名和之间，哈希值会有所不同。我们可以，
//但是，假设脚本中没有符号等，因为
//会使事务不标准，因此
//多尺度，所以我们只需要散列完整的东西。
		hash := calcSignatureHash(pkPops, hashType, tx, idx)

		for _, addr := range addresses {
//所有multisig地址都应为pubkey地址
//调用此内部函数时出错
//输入错误。
			pkaddr := addr.(*btcutil.AddressPubKey)

			pubKey := pkaddr.PubKey()

//如果匹配，我们就把它放在地图上。我们只
//每个公钥可以接受一个签名，因此如果我们
//已经有了，我们可以把这个扔掉。
			if pSig.Verify(hash, pubKey) {
				aStr := addr.EncodeAddress()
				if _, ok := addrToSig[aStr]; !ok {
					addrToSig[aStr] = sig
				}
				continue sigLoop
			}
		}
	}

//额外的操作码来处理所消耗的额外参数（由于以前的错误）
//在引用实现中）。
	builder := NewScriptBuilder().AddOp(OP_FALSE)
	doneSigs := 0
//这假定地址的顺序与脚本中的顺序相同。
	for _, addr := range addresses {
		sig, ok := addrToSig[addr.EncodeAddress()]
		if !ok {
			continue
		}
		builder.AddData(sig)
		doneSigs++
		if doneSigs == nRequired {
			break
		}
	}

//为丢失的填充。
	for i := doneSigs; i < nRequired; i++ {
		builder.AddOp(OP_0)
	}

	script, _ := builder.Script()
	return script
}

//keydb是为signtxoutput提供的接口类型，它封装
//获取地址的私钥所需的任何用户状态。
type KeyDB interface {
	GetKey(btcutil.Address) (*btcec.PrivateKey, bool, error)
}

//keyclosure通过一个闭包实现keydb。
type KeyClosure func(btcutil.Address) (*btcec.PrivateKey, bool, error)

//getkey通过返回调用闭包的结果来实现keydb。
func (kc KeyClosure) GetKey(address btcutil.Address) (*btcec.PrivateKey,
	bool, error) {
	return kc(address)
}

//scriptdb是为signtxoutput提供的接口类型，它封装了
//获取付薪脚本哈希地址的脚本所需的用户状态。
type ScriptDB interface {
	GetScript(btcutil.Address) ([]byte, error)
}

//scriptClosing使用一个闭包实现scriptDB。
type ScriptClosure func(btcutil.Address) ([]byte, error)

//getscript通过返回调用闭包的结果来实现scriptdb。
func (sc ScriptClosure) GetScript(address btcutil.Address) ([]byte, error) {
	return sc(address)
}

//signtxoutput为给定tx的输出idx签名以解析中给定的脚本
//签名类型为hashtype的pkscript。所需的任何钥匙
//通过使用给定地址的字符串调用getkey（）进行查找。
//通过调用
//GETScript。如果提供了previousscript，则返回previousscript
//将以依赖于类型的方式与新生成的合并。
//签名脚本。
func SignTxOutput(chainParams *chaincfg.Params, tx *wire.MsgTx, idx int,
	pkScript []byte, hashType SigHashType, kdb KeyDB, sdb ScriptDB,
	previousScript []byte) ([]byte, error) {

	sigScript, class, addresses, nrequired, err := sign(chainParams, tx,
		idx, pkScript, hashType, kdb, sdb)
	if err != nil {
		return nil, err
	}

	if class == ScriptHashTy {
//要保留子地址并传递给合并。
		realSigScript, _, _, _, err := sign(chainParams, tx, idx,
			sigScript, hashType, kdb, sdb)
		if err != nil {
			return nil, err
		}

//附加p2sh脚本作为脚本中的最后一个推送。
		builder := NewScriptBuilder()
		builder.AddOps(realSigScript)
		builder.AddData(sigScript)

		sigScript, _ = builder.Script()
//要保留脚本的副本以进行合并。
	}

//合并脚本。如果有任何以前的数据。
	mergedScript := mergeScripts(chainParams, tx, idx, pkScript, class,
		addresses, nrequired, sigScript, previousScript)
	return mergedScript, nil
}
