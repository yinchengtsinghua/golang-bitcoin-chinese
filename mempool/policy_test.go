
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
	"bytes"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

//TestCalcMinRequiredXRelayFee测试CalcMinRequiredXRelayFee API。
func TestCalcMinRequiredTxRelayFee(t *testing.T) {
	tests := []struct {
name     string         //测试说明。
size     int64          //事务大小（字节）。
relayFee btcutil.Amount //最低中继交易费。
want     int64          //预期费用。
	}{
		{
//确保规模和费用组合小于1000
//产生非零费用。
			"250 bytes with relay fee of 3",
			250,
			3,
			3,
		},
		{
			"100 bytes with default minimum relay fee",
			100,
			DefaultMinRelayTxFee,
			100,
		},
		{
			"max standard tx size with default minimum relay fee",
			maxStandardTxWeight / 4,
			DefaultMinRelayTxFee,
			100000,
		},
		{
			"max standard tx size with max satoshi relay fee",
			maxStandardTxWeight / 4,
			btcutil.MaxSatoshi,
			btcutil.MaxSatoshi,
		},
		{
			"1500 bytes with 5000 relay fee",
			1500,
			5000,
			7500,
		},
		{
			"1500 bytes with 3000 relay fee",
			1500,
			3000,
			4500,
		},
		{
			"782 bytes with 5000 relay fee",
			782,
			5000,
			3910,
		},
		{
			"782 bytes with 3000 relay fee",
			782,
			3000,
			2346,
		},
		{
			"782 bytes with 2550 relay fee",
			782,
			2550,
			1994,
		},
	}

	for _, test := range tests {
		got := calcMinRequiredTxRelayFee(test.size, test.relayFee)
		if got != test.want {
			t.Errorf("TestCalcMinRequiredTxRelayFee test '%s' "+
				"failed: got %v want %v", test.name, got,
				test.want)
			continue
		}
	}
}

//testcheckpkscriptStandard测试checkpkscriptStandard API。
func TestCheckPkScriptStandard(t *testing.T) {
	var pubKeys [][]byte
	for i := 0; i < 4; i++ {
		pk, err := btcec.NewPrivateKey(btcec.S256())
		if err != nil {
			t.Fatalf("TestCheckPkScriptStandard NewPrivateKey failed: %v",
				err)
			return
		}
		pubKeys = append(pubKeys, pk.PubKey().SerializeCompressed())
	}

	tests := []struct {
name       string //测试说明。
		script     *txscript.ScriptBuilder
		isStandard bool
	}{
		{
			"key1 and key2",
			txscript.NewScriptBuilder().AddOp(txscript.OP_2).
				AddData(pubKeys[0]).AddData(pubKeys[1]).
				AddOp(txscript.OP_2).AddOp(txscript.OP_CHECKMULTISIG),
			true,
		},
		{
			"key1 or key2",
			txscript.NewScriptBuilder().AddOp(txscript.OP_1).
				AddData(pubKeys[0]).AddData(pubKeys[1]).
				AddOp(txscript.OP_2).AddOp(txscript.OP_CHECKMULTISIG),
			true,
		},
		{
			"escrow",
			txscript.NewScriptBuilder().AddOp(txscript.OP_2).
				AddData(pubKeys[0]).AddData(pubKeys[1]).
				AddData(pubKeys[2]).
				AddOp(txscript.OP_3).AddOp(txscript.OP_CHECKMULTISIG),
			true,
		},
		{
			"one of four",
			txscript.NewScriptBuilder().AddOp(txscript.OP_1).
				AddData(pubKeys[0]).AddData(pubKeys[1]).
				AddData(pubKeys[2]).AddData(pubKeys[3]).
				AddOp(txscript.OP_4).AddOp(txscript.OP_CHECKMULTISIG),
			false,
		},
		{
			"malformed1",
			txscript.NewScriptBuilder().AddOp(txscript.OP_3).
				AddData(pubKeys[0]).AddData(pubKeys[1]).
				AddOp(txscript.OP_2).AddOp(txscript.OP_CHECKMULTISIG),
			false,
		},
		{
			"malformed2",
			txscript.NewScriptBuilder().AddOp(txscript.OP_2).
				AddData(pubKeys[0]).AddData(pubKeys[1]).
				AddOp(txscript.OP_3).AddOp(txscript.OP_CHECKMULTISIG),
			false,
		},
		{
			"malformed3",
			txscript.NewScriptBuilder().AddOp(txscript.OP_0).
				AddData(pubKeys[0]).AddData(pubKeys[1]).
				AddOp(txscript.OP_2).AddOp(txscript.OP_CHECKMULTISIG),
			false,
		},
		{
			"malformed4",
			txscript.NewScriptBuilder().AddOp(txscript.OP_1).
				AddData(pubKeys[0]).AddData(pubKeys[1]).
				AddOp(txscript.OP_0).AddOp(txscript.OP_CHECKMULTISIG),
			false,
		},
		{
			"malformed5",
			txscript.NewScriptBuilder().AddOp(txscript.OP_1).
				AddData(pubKeys[0]).AddData(pubKeys[1]).
				AddOp(txscript.OP_CHECKMULTISIG),
			false,
		},
		{
			"malformed6",
			txscript.NewScriptBuilder().AddOp(txscript.OP_1).
				AddData(pubKeys[0]).AddData(pubKeys[1]),
			false,
		},
	}

	for _, test := range tests {
		script, err := test.script.Script()
		if err != nil {
			t.Fatalf("TestCheckPkScriptStandard test '%s' "+
				"failed: %v", test.name, err)
			continue
		}
		scriptClass := txscript.GetScriptClass(script)
		got := checkPkScriptStandard(script, scriptClass)
		if (test.isStandard && got != nil) ||
			(!test.isStandard && got == nil) {

			t.Fatalf("TestCheckPkScriptStandard test '%s' failed",
				test.name)
			return
		}
	}
}

//TestDust测试ISdust API。
func TestDust(t *testing.T) {
	pkScript := []byte{0x76, 0xa9, 0x21, 0x03, 0x2f, 0x7e, 0x43,
		0x0a, 0xa4, 0xc9, 0xd1, 0x59, 0x43, 0x7e, 0x84, 0xb9,
		0x75, 0xdc, 0x76, 0xd9, 0x00, 0x3b, 0xf0, 0x92, 0x2c,
		0xf3, 0xaa, 0x45, 0x28, 0x46, 0x4b, 0xab, 0x78, 0x0d,
		0xba, 0x5e, 0x88, 0xac}

	tests := []struct {
name     string //测试说明
		txOut    wire.TxOut
relayFee btcutil.Amount //最低中继交易费。
		isDust   bool
	}{
		{
//任何值都允许零中继费。
			"zero value with zero relay fee",
			wire.TxOut{Value: 0, PkScript: pkScript},
			0,
			false,
		},
		{
//零值为含任何中继费的灰尘”
			"zero value with very small tx fee",
			wire.TxOut{Value: 0, PkScript: pkScript},
			1,
			true,
		},
		{
			"38 byte public key script with value 584",
			wire.TxOut{Value: 584, PkScript: pkScript},
			1000,
			true,
		},
		{
			"38 byte public key script with value 585",
			wire.TxOut{Value: 585, PkScript: pkScript},
			1000,
			false,
		},
		{
//Maximum allowed value is never dust.
			"max satoshi amount is never dust",
			wire.TxOut{Value: btcutil.MaxSatoshi, PkScript: pkScript},
			btcutil.MaxSatoshi,
			false,
		},
		{
//最大Int64值导致溢出。
			"maximum int64 value",
			wire.TxOut{Value: 1<<63 - 1, PkScript: pkScript},
			1<<63 - 1,
			true,
		},
		{
//由于公钥无效而无法挂起的pkscript
//脚本。
			"unspendable pkScript",
			wire.TxOut{Value: 5000, PkScript: []byte{0x01}},
0, //无中继费
			true,
		},
	}
	for _, test := range tests {
		res := isDust(&test.txOut, test.relayFee)
		if res != test.isDust {
			t.Fatalf("Dust test '%s' failed: want %v got %v",
				test.name, test.isDust, res)
			continue
		}
	}
}

//TestCheckTransactions标准测试CheckTransactions标准API。
func TestCheckTransactionStandard(t *testing.T) {
//为事务创建一些虚拟的，但不是标准的数据。
	prevOutHash, err := chainhash.NewHashFromStr("01")
	if err != nil {
		t.Fatalf("NewShaHashFromStr: unexpected error: %v", err)
	}
	dummyPrevOut := wire.OutPoint{Hash: *prevOutHash, Index: 1}
	dummySigScript := bytes.Repeat([]byte{0x00}, 65)
	dummyTxIn := wire.TxIn{
		PreviousOutPoint: dummyPrevOut,
		SignatureScript:  dummySigScript,
		Sequence:         wire.MaxTxInSequenceNum,
	}
	addrHash := [20]byte{0x01}
	addr, err := btcutil.NewAddressPubKeyHash(addrHash[:],
		&chaincfg.TestNet3Params)
	if err != nil {
		t.Fatalf("NewAddressPubKeyHash: unexpected error: %v", err)
	}
	dummyPkScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		t.Fatalf("PayToAddrScript: unexpected error: %v", err)
	}
	dummyTxOut := wire.TxOut{
Value:    100000000, //1 BTC
		PkScript: dummyPkScript,
	}

	tests := []struct {
		name       string
		tx         wire.MsgTx
		height     int32
		isStandard bool
		code       wire.RejectCode
	}{
		{
			name: "Typical pay-to-pubkey-hash transaction",
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{&dummyTxIn},
				TxOut:    []*wire.TxOut{&dummyTxOut},
				LockTime: 0,
			},
			height:     300000,
			isStandard: true,
		},
		{
			name: "Transaction version too high",
			tx: wire.MsgTx{
				Version:  wire.TxVersion + 1,
				TxIn:     []*wire.TxIn{&dummyTxIn},
				TxOut:    []*wire.TxOut{&dummyTxOut},
				LockTime: 0,
			},
			height:     300000,
			isStandard: false,
			code:       wire.RejectNonstandard,
		},
		{
			name: "Transaction is not finalized",
			tx: wire.MsgTx{
				Version: 1,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: dummyPrevOut,
					SignatureScript:  dummySigScript,
					Sequence:         0,
				}},
				TxOut:    []*wire.TxOut{&dummyTxOut},
				LockTime: 300001,
			},
			height:     300000,
			isStandard: false,
			code:       wire.RejectNonstandard,
		},
		{
			name: "Transaction size is too large",
			tx: wire.MsgTx{
				Version: 1,
				TxIn:    []*wire.TxIn{&dummyTxIn},
				TxOut: []*wire.TxOut{{
					Value: 0,
					PkScript: bytes.Repeat([]byte{0x00},
						(maxStandardTxWeight/4)+1),
				}},
				LockTime: 0,
			},
			height:     300000,
			isStandard: false,
			code:       wire.RejectNonstandard,
		},
		{
			name: "Signature script size is too large",
			tx: wire.MsgTx{
				Version: 1,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: dummyPrevOut,
					SignatureScript: bytes.Repeat([]byte{0x00},
						maxStandardSigScriptSize+1),
					Sequence: wire.MaxTxInSequenceNum,
				}},
				TxOut:    []*wire.TxOut{&dummyTxOut},
				LockTime: 0,
			},
			height:     300000,
			isStandard: false,
			code:       wire.RejectNonstandard,
		},
		{
			name: "Signature script that does more than push data",
			tx: wire.MsgTx{
				Version: 1,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: dummyPrevOut,
					SignatureScript: []byte{
						txscript.OP_CHECKSIGVERIFY},
					Sequence: wire.MaxTxInSequenceNum,
				}},
				TxOut:    []*wire.TxOut{&dummyTxOut},
				LockTime: 0,
			},
			height:     300000,
			isStandard: false,
			code:       wire.RejectNonstandard,
		},
		{
			name: "Valid but non standard public key script",
			tx: wire.MsgTx{
				Version: 1,
				TxIn:    []*wire.TxIn{&dummyTxIn},
				TxOut: []*wire.TxOut{{
					Value:    100000000,
					PkScript: []byte{txscript.OP_TRUE},
				}},
				LockTime: 0,
			},
			height:     300000,
			isStandard: false,
			code:       wire.RejectNonstandard,
		},
		{
			name: "More than one nulldata output",
			tx: wire.MsgTx{
				Version: 1,
				TxIn:    []*wire.TxIn{&dummyTxIn},
				TxOut: []*wire.TxOut{{
					Value:    0,
					PkScript: []byte{txscript.OP_RETURN},
				}, {
					Value:    0,
					PkScript: []byte{txscript.OP_RETURN},
				}},
				LockTime: 0,
			},
			height:     300000,
			isStandard: false,
			code:       wire.RejectNonstandard,
		},
		{
			name: "Dust output",
			tx: wire.MsgTx{
				Version: 1,
				TxIn:    []*wire.TxIn{&dummyTxIn},
				TxOut: []*wire.TxOut{{
					Value:    0,
					PkScript: dummyPkScript,
				}},
				LockTime: 0,
			},
			height:     300000,
			isStandard: false,
			code:       wire.RejectDust,
		},
		{
			name: "One nulldata output with 0 amount (standard)",
			tx: wire.MsgTx{
				Version: 1,
				TxIn:    []*wire.TxIn{&dummyTxIn},
				TxOut: []*wire.TxOut{{
					Value:    0,
					PkScript: []byte{txscript.OP_RETURN},
				}},
				LockTime: 0,
			},
			height:     300000,
			isStandard: true,
		},
	}

	pastMedianTime := time.Now()
	for _, test := range tests {
//确保标准符合预期。
		err := checkTransactionStandard(btcutil.NewTx(&test.tx),
			test.height, pastMedianTime, DefaultMinRelayTxFee, 1)
		if err == nil && test.isStandard {
//测试通过，因为函数返回了
//旨在成为标准的交易。
			continue
		}
		if err == nil && !test.isStandard {
			t.Errorf("checkTransactionStandard (%s): standard when "+
				"it should not be", test.name)
			continue
		}
		if err != nil && test.isStandard {
			t.Errorf("checkTransactionStandard (%s): nonstandard "+
				"when it should not be: %v", test.name, err)
			continue
		}

//确保错误类型是RuleError内部的TxRuleError。
		rerr, ok := err.(RuleError)
		if !ok {
			t.Errorf("checkTransactionStandard (%s): unexpected "+
				"error type - got %T", test.name, err)
			continue
		}
		txrerr, ok := rerr.Err.(TxRuleError)
		if !ok {
			t.Errorf("checkTransactionStandard (%s): unexpected "+
				"error type - got %T", test.name, rerr.Err)
			continue
		}

//Ensure the reject code is the expected one.
		if txrerr.RejectCode != test.code {
			t.Errorf("checkTransactionStandard (%s): unexpected "+
				"error code - got %v, want %v", test.name,
				txrerr.RejectCode, test.code)
			continue
		}
	}
}
