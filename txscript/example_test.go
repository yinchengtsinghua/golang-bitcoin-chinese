
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2014-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package txscript_test

import (
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

//此示例演示如何创建支付比特币地址的脚本。
//它还打印创建的脚本hex，并使用disasmstring函数
//显示反汇编的脚本。
func ExamplePayToAddrScript() {
//分析地址以将硬币发送到btucil.address。
//这对确保地址的准确性和确定
//地址类型。它也需要为即将到来的呼叫
//付款地址脚本。
	addressStr := "12gpXQVcCL2qhTNQgyLVdCFG2Qs2px98nV"
	address, err := btcutil.DecodeAddress(addressStr, &chaincfg.MainNetParams)
	if err != nil {
		fmt.Println(err)
		return
	}

//创建一个支付地址的公钥脚本。
	script, err := txscript.PayToAddrScript(address)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Script Hex: %x\n", script)

	disasm, err := txscript.DisasmString(script)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Script Disassembly:", disasm)

//输出：
//手写体十六进制：76A9114128004ff2fca13b2b91eb654b1dc2b674f7ec6188ac
//脚本反汇编：op_dup op_hash160 128004f2f2fcaf13b2b91eb654b1dc2b674f7ec61 op_equalverify op_checksig
}

//此示例演示如何从标准公钥提取信息
//脚本。
func ExampleExtractPkScriptAddrs() {
//从标准的pay-to-pubkey哈希脚本开始。
	scriptHex := "76a914128004ff2fcaf13b2b91eb654b1dc2b674f7ec6188ac"
	script, err := hex.DecodeString(scriptHex)
	if err != nil {
		fmt.Println(err)
		return
	}

//从脚本中提取并打印详细信息。
	scriptClass, addresses, reqSigs, err := txscript.ExtractPkScriptAddrs(
		script, &chaincfg.MainNetParams)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Script Class:", scriptClass)
	fmt.Println("Addresses:", addresses)
	fmt.Println("Required Signatures:", reqSigs)

//输出：
//脚本类：pubkeyhash
//地址：【12gpxqvccl2qhtnqgylvdcfg2qs2px98nv】
//所需签名：1
}

//此示例演示如何手动创建和签署兑换交易。
func ExampleSignTxOutput() {
//通常，私钥来自任何存储机制
//正在使用，但对于本例，只需硬编码即可。
	privKeyBytes, err := hex.DecodeString("22a47fa09a223f2aa079edf85a7c2" +
		"d4f8720ee63e502ee2869afab7de234b80c")
	if err != nil {
		fmt.Println(err)
		return
	}
	privKey, pubKey := btcec.PrivKeyFromBytes(btcec.S256(), privKeyBytes)
	pubKeyHash := btcutil.Hash160(pubKey.SerializeCompressed())
	addr, err := btcutil.NewAddressPubKeyHash(pubKeyHash,
		&chaincfg.MainNetParams)
	if err != nil {
		fmt.Println(err)
		return
	}

//对于本例，创建一个表示
//通常是真正的交易。它
//包含一个向地址支付1 BTC金额的单个输出。
	originTx := wire.NewMsgTx(wire.TxVersion)
	prevOut := wire.NewOutPoint(&chainhash.Hash{}, ^uint32(0))
	txIn := wire.NewTxIn(prevOut, []byte{txscript.OP_0, txscript.OP_0}, nil)
	originTx.AddTxIn(txIn)
	pkScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		fmt.Println(err)
		return
	}
	txOut := wire.NewTxOut(100000000, pkScript)
	originTx.AddTxOut(txOut)
	originTxHash := originTx.TxHash()

//创建交易以赎回假交易。
	redeemTx := wire.NewMsgTx(wire.TxVersion)

//添加赎回交易将花费的输入。没有
//签名脚本现在还没有创建或签名
//然而，因此没有提供。
	prevOut = wire.NewOutPoint(&originTxHash, 0)
	txIn = wire.NewTxIn(prevOut, nil, nil)
	redeemTx.AddTxIn(txIn)

//通常，这将包含资金的实际目的地，
//但对于这个例子，不必费心。
	txOut = wire.NewTxOut(0, nil)
	redeemTx.AddTxOut(txOut)

//签署赎回交易。
	lookupKey := func(a btcutil.Address) (*btcec.PrivateKey, bool, error) {
//通常，这个功能包括查找
//提供的地址的键，但由于
//在本示例中签名时使用与
//上面的私钥，只需返回压缩后的
//标志集，因为地址使用关联的压缩
//公钥。
//
//注意：如果您想证明代码实际上正在签名
//事务处理正确，取消对以下行的注释
//故意返回要签名的无效密钥，其中
//turn将在脚本执行期间导致失败
//在验证签名时。
//
//privkey.d.setInt64（12345）
//
		return privKey, true, nil
	}
//注意，这里的脚本数据库参数为nil，因为它不是
//使用。当支付到脚本哈希事务
//被签署。
	sigScript, err := txscript.SignTxOutput(&chaincfg.MainNetParams,
		redeemTx, 0, originTx.TxOut[0].PkScript, txscript.SigHashAll,
		txscript.KeyClosure(lookupKey), nil, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	redeemTx.TxIn[0].SignatureScript = sigScript

//证明交易已通过执行
//脚本对。
	flags := txscript.ScriptBip16 | txscript.ScriptVerifyDERSignatures |
		txscript.ScriptStrictMultiSig |
		txscript.ScriptDiscourageUpgradableNops
	vm, err := txscript.NewEngine(originTx.TxOut[0].PkScript, redeemTx, 0,
		flags, nil, nil, -1)
	if err != nil {
		fmt.Println(err)
		return
	}
	if err := vm.Execute(); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Transaction successfully signed")

//输出：
//事务已成功签名
}
