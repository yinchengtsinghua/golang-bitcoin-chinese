
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package txscript

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"golang.org/x/crypto/ripemd160"
)

const (
//PubKeyHashsigScriptlen是尝试的签名脚本的长度
//使用p2pkh脚本。唯一可能的长度值是107
//字节，因为其中的签名。此长度由
//下列内容：
//0x47或0x48（71或72字节数据推送）<71或72字节信号
//0x21（33字节数据推送）<33字节压缩pubkey>
	pubKeyHashSigScriptLen = 106

//compressedSubkeylen是压缩公共对象的长度（以字节为单位）
//关键。
	compressedPubKeyLen = 33

//pubKeyHashlen是p2pkh脚本的长度。
	pubKeyHashLen = 25

//witnessv0pubkeyhashlen是p2wpkh脚本的长度。
	witnessV0PubKeyHashLen = 22

//scriptHashlen是p2sh脚本的长度。
	scriptHashLen = 23

//witnessv0scriptHashlen是p2wsh脚本的长度。
	witnessV0ScriptHashLen = 34

//maxlen是parsepkscript支持的最大脚本长度。
	maxLen = witnessV0ScriptHashLen
)

var (
//ErrUnsupportedScriptType是尝试执行以下操作时返回的错误
//将输出脚本解析/重新计算为pkscript结构。
	ErrUnsupportedScriptType = errors.New("unsupported script type")
)

//pkscript是一个围绕字节数组的包装结构，允许使用它
//作为地图索引。
type PkScript struct {
//类是在字节数组中编码的脚本类型。这个
//用于确定字节内脚本的正确长度
//数组。
	class ScriptClass

//脚本是包含在字节数组中的脚本。如果脚本是
//小于字节数组的长度，将用0填充
//最后。
	script [maxLen]byte
}

//parsepkscript将输出脚本解析为pkscript结构。
//尝试分析不受支持的
//脚本类型。
func ParsePkScript(pkScript []byte) (PkScript, error) {
	var outputScript PkScript
	scriptClass, _, _, err := ExtractPkScriptAddrs(
		pkScript, &chaincfg.MainNetParams,
	)
	if err != nil {
		return outputScript, fmt.Errorf("unable to parse script type: "+
			"%v", err)
	}

	if !isSupportedScriptType(scriptClass) {
		return outputScript, ErrUnsupportedScriptType
	}

	outputScript.class = scriptClass
	copy(outputScript.script[:], pkScript)

	return outputScript, nil
}

//IssupportedScriptType确定脚本类型是否受
//pkscript结构。
func isSupportedScriptType(class ScriptClass) bool {
	switch class {
	case PubKeyHashTy, WitnessV0PubKeyHashTy, ScriptHashTy,
		WitnessV0ScriptHashTy:
		return true
	default:
		return false
	}
}

//类返回脚本类型。
func (s PkScript) Class() ScriptClass {
	return s.class
}

//脚本以字节片的形式返回脚本，不带任何填充。
func (s PkScript) Script() []byte {
	var script []byte

	switch s.class {
	case PubKeyHashTy:
		script = make([]byte, pubKeyHashLen)
		copy(script, s.script[:pubKeyHashLen])

	case WitnessV0PubKeyHashTy:
		script = make([]byte, witnessV0PubKeyHashLen)
		copy(script, s.script[:witnessV0PubKeyHashLen])

	case ScriptHashTy:
		script = make([]byte, scriptHashLen)
		copy(script, s.script[:scriptHashLen])

	case WitnessV0ScriptHashTy:
		script = make([]byte, witnessV0ScriptHashLen)
		copy(script, s.script[:witnessV0ScriptHashLen])

	default:
//不支持的脚本类型。
		return nil
	}

	return script
}

//地址将脚本编码为给定链的地址。
func (s PkScript) Address(chainParams *chaincfg.Params) (btcutil.Address, error) {
	_, addrs, _, err := ExtractPkScriptAddrs(s.Script(), chainParams)
	if err != nil {
		return nil, fmt.Errorf("unable to parse address: %v", err)
	}

	return addrs[0], nil
}

//字符串返回脚本的十六进制编码字符串表示形式。
func (s PkScript) String() string {
	str, _ := DisasmString(s.Script())
	return str
}

//computepkscript通过查看
//事务输入的签名脚本或见证。
//
//注意：只支持p2pkh、p2sh、p2wsh和p2wpkh兑换脚本。
func ComputePkScript(sigScript []byte, witness wire.TxWitness) (PkScript, error) {
	var pkScript PkScript

//确保输入的签名脚本或证人
//提供。
	if len(sigScript) == 0 && len(witness) == 0 {
		return pkScript, ErrUnsupportedScriptType
	}

//我们将首先检查输入的签名脚本（如果提供）。
	switch {
//如果签名脚本的长度足以
//表示p2pkh脚本，然后我们将尝试解析压缩的
//它的公钥。
	case len(sigScript) == pubKeyHashSigScriptLen ||
		len(sigScript) == pubKeyHashSigScriptLen+1:

//公钥应作为
//签名脚本。我们将尝试分析它以确保
//p2pkh兑换脚本。
		pubKey := sigScript[len(sigScript)-compressedPubKeyLen:]
		if btcec.IsCompressedPubKey(pubKey) {
			pubKeyHash := hash160(pubKey)
			script, err := payToPubKeyHashScript(pubKeyHash)
			if err != nil {
				return pkScript, err
			}

			pkScript.class = PubKeyHashTy
			copy(pkScript.script[:], script)
			return pkScript, nil
		}

//如果不是，我们假设它是一个p2sh签名脚本。
		fallthrough

//如果未能从脚本中分析压缩的公钥，
//如果脚本长度不是p2pkh脚本的长度，或者如果脚本长度不是p2pkh脚本的长度，并且
//我们的兑换脚本仅由推送的数据组成，我们可以假设它是
//p2sh签名脚本。
	case len(sigScript) > 0 && IsPushOnlyScript(sigScript):
//兑换脚本将始终是
//签名脚本，因此我们将把脚本解析为操作码
//得到它。
		parsedOpcodes, err := parseScript(sigScript)
		if err != nil {
			return pkScript, err
		}
		redeemScript := parsedOpcodes[len(parsedOpcodes)-1].data

		scriptHash := hash160(redeemScript)
		script, err := payToScriptHashScript(scriptHash)
		if err != nil {
			return pkScript, err
		}

		pkScript.class = ScriptHashTy
		copy(pkScript.script[:], script)
		return pkScript, nil

	case len(sigScript) > 0:
		return pkScript, ErrUnsupportedScriptType
	}

//如果提供了证人，我们将使用
//见证堆栈以确定正确的见证类型。
	lastWitnessItem := witness[len(witness)-1]

	switch {
//如果见证堆栈的大小为2，并且其最后一项是
//压缩的公钥，那么这是一个p2wpkh见证人。
	case len(witness) == 2 && len(lastWitnessItem) == compressedPubKeyLen:
		pubKeyHash := hash160(lastWitnessItem)
		script, err := payToWitnessPubKeyHashScript(pubKeyHash)
		if err != nil {
			return pkScript, err
		}

		pkScript.class = WitnessV0PubKeyHashTy
		copy(pkScript.script[:], script)
		return pkScript, nil

//对于任何其他证人，我们假设他是P2WSH证人。
	default:
		scriptHash := sha256.Sum256(lastWitnessItem)
		script, err := payToWitnessScriptHashScript(scriptHash[:])
		if err != nil {
			return pkScript, err
		}

		pkScript.class = WitnessV0ScriptHashTy
		copy(pkScript.script[:], script)
		return pkScript, nil
	}
}

//hash160返回给定数据的sha-256哈希的ripemd160哈希。
func hash160(data []byte) []byte {
	h := sha256.Sum256(data)
	return ripemd160h(h[:])
}

//ripemd160h返回给定数据的ripemd160哈希。
func ripemd160h(data []byte) []byte {
	h := ripemd160.New()
	h.Write(data)
	return h.Sum(nil)
}
