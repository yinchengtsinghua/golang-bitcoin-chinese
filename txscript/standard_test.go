
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
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

//mustParseShortForm分析传递的短格式脚本并返回
//结果字节。如果发生错误，它会恐慌。这只用于
//作为助手进行测试，因为它失败的唯一方法是
//测试源代码。
func mustParseShortForm(script string) []byte {
	s, err := parseShortForm(script)
	if err != nil {
		panic("invalid short form script in test source: err " +
			err.Error() + ", script: " + script)
	}

	return s
}

//NewAddressPubKey从提供的
//序列化公钥。如果发生错误，它会恐慌。这只用于
//测试作为一个助手，因为它失败的唯一方法是如果出现错误
//在测试源代码中。
func newAddressPubKey(serializedPubKey []byte) btcutil.Address {
	addr, err := btcutil.NewAddressPubKey(serializedPubKey,
		&chaincfg.MainNetParams)
	if err != nil {
		panic("invalid public key in test source")
	}

	return addr
}

//NewAddressPubKeyHash返回来自
//提供了哈希。如果发生错误，它会恐慌。这只在测试中使用
//作为帮助者，因为它失败的唯一方法是
//测试源代码。
func newAddressPubKeyHash(pkHash []byte) btcutil.Address {
	addr, err := btcutil.NewAddressPubKeyHash(pkHash, &chaincfg.MainNetParams)
	if err != nil {
		panic("invalid public key hash in test source")
	}

	return addr
}

//NewAddressScriptHash返回来自
//提供了哈希。如果发生错误，它会恐慌。这只在测试中使用
//作为帮助者，因为它失败的唯一方法是
//测试源代码。
func newAddressScriptHash(scriptHash []byte) btcutil.Address {
	addr, err := btcutil.NewAddressScriptHashFromHash(scriptHash,
		&chaincfg.MainNetParams)
	if err != nil {
		panic("invalid script hash in test source")
	}

	return addr
}

//testExtracpkScriptAddrs确保提取类型、地址和
//pkscripts所需的签名数按预期工作。
func TestExtractPkScriptAddrs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		script  []byte
		addrs   []btcutil.Address
		reqSigs int
		class   ScriptClass
	}{
		{
			name: "standard p2pk with compressed pubkey (0x02)",
			script: hexToBytes("2102192d74d0cb94344c9569c2e779015" +
				"73d8d7903c3ebec3a957724895dca52c6b4ac"),
			addrs: []btcutil.Address{
				newAddressPubKey(hexToBytes("02192d74d0cb9434" +
					"4c9569c2e77901573d8d7903c3ebec3a9577" +
					"24895dca52c6b4")),
			},
			reqSigs: 1,
			class:   PubKeyTy,
		},
		{
			name: "standard p2pk with uncompressed pubkey (0x04)",
			script: hexToBytes("410411db93e1dcdb8a016b49840f8c53b" +
				"c1eb68a382e97b1482ecad7b148a6909a5cb2e0eaddf" +
				"b84ccf9744464f82e160bfa9b8b64f9d4c03f999b864" +
				"3f656b412a3ac"),
			addrs: []btcutil.Address{
				newAddressPubKey(hexToBytes("0411db93e1dcdb8a" +
					"016b49840f8c53bc1eb68a382e97b1482eca" +
					"d7b148a6909a5cb2e0eaddfb84ccf9744464" +
					"f82e160bfa9b8b64f9d4c03f999b8643f656" +
					"b412a3")),
			},
			reqSigs: 1,
			class:   PubKeyTy,
		},
		{
			name: "standard p2pk with hybrid pubkey (0x06)",
			script: hexToBytes("4106192d74d0cb94344c9569c2e779015" +
				"73d8d7903c3ebec3a957724895dca52c6b40d4526483" +
				"8c0bd96852662ce6a847b197376830160c6d2eb5e6a4" +
				"c44d33f453eac"),
			addrs: []btcutil.Address{
				newAddressPubKey(hexToBytes("06192d74d0cb9434" +
					"4c9569c2e77901573d8d7903c3ebec3a9577" +
					"24895dca52c6b40d45264838c0bd96852662" +
					"ce6a847b197376830160c6d2eb5e6a4c44d3" +
					"3f453e")),
			},
			reqSigs: 1,
			class:   PubKeyTy,
		},
		{
			name: "standard p2pk with compressed pubkey (0x03)",
			script: hexToBytes("2103b0bd634234abbb1ba1e986e884185" +
				"c61cf43e001f9137f23c2c409273eb16e65ac"),
			addrs: []btcutil.Address{
				newAddressPubKey(hexToBytes("03b0bd634234abbb" +
					"1ba1e986e884185c61cf43e001f9137f23c2" +
					"c409273eb16e65")),
			},
			reqSigs: 1,
			class:   PubKeyTy,
		},
		{
			name: "2nd standard p2pk with uncompressed pubkey (0x04)",
			script: hexToBytes("4104b0bd634234abbb1ba1e986e884185" +
				"c61cf43e001f9137f23c2c409273eb16e6537a576782" +
				"eba668a7ef8bd3b3cfb1edb7117ab65129b8a2e681f3" +
				"c1e0908ef7bac"),
			addrs: []btcutil.Address{
				newAddressPubKey(hexToBytes("04b0bd634234abbb" +
					"1ba1e986e884185c61cf43e001f9137f23c2" +
					"c409273eb16e6537a576782eba668a7ef8bd" +
					"3b3cfb1edb7117ab65129b8a2e681f3c1e09" +
					"08ef7b")),
			},
			reqSigs: 1,
			class:   PubKeyTy,
		},
		{
			name: "standard p2pk with hybrid pubkey (0x07)",
			script: hexToBytes("4107b0bd634234abbb1ba1e986e884185" +
				"c61cf43e001f9137f23c2c409273eb16e6537a576782" +
				"eba668a7ef8bd3b3cfb1edb7117ab65129b8a2e681f3" +
				"c1e0908ef7bac"),
			addrs: []btcutil.Address{
				newAddressPubKey(hexToBytes("07b0bd634234abbb" +
					"1ba1e986e884185c61cf43e001f9137f23c2" +
					"c409273eb16e6537a576782eba668a7ef8bd" +
					"3b3cfb1edb7117ab65129b8a2e681f3c1e09" +
					"08ef7b")),
			},
			reqSigs: 1,
			class:   PubKeyTy,
		},
		{
			name: "standard p2pkh",
			script: hexToBytes("76a914ad06dd6ddee55cbca9a9e3713bd" +
				"7587509a3056488ac"),
			addrs: []btcutil.Address{
				newAddressPubKeyHash(hexToBytes("ad06dd6ddee5" +
					"5cbca9a9e3713bd7587509a30564")),
			},
			reqSigs: 1,
			class:   PubKeyHashTy,
		},
		{
			name: "standard p2sh",
			script: hexToBytes("a91463bcc565f9e68ee0189dd5cc67f1b" +
				"0e5f02f45cb87"),
			addrs: []btcutil.Address{
				newAddressScriptHash(hexToBytes("63bcc565f9e6" +
					"8ee0189dd5cc67f1b0e5f02f45cb")),
			},
			reqSigs: 1,
			class:   ScriptHashTy,
		},
//来自Real TX 60A20BD93AA49AB4B28D514EC10B06E1829CE6818EC06CD3ABD013EBCD4BB1，凭证0
		{
			name: "standard 1 of 2 multisig",
			script: hexToBytes("514104cc71eb30d653c0c3163990c47b9" +
				"76f3fb3f37cccdcbedb169a1dfef58bbfbfaff7d8a47" +
				"3e7e2e6d317b87bafe8bde97e3cf8f065dec022b51d1" +
				"1fcdd0d348ac4410461cbdcc5409fb4b4d42b51d3338" +
				"1354d80e550078cb532a34bfa2fcfdeb7d76519aecc6" +
				"2770f5b0e4ef8551946d8a540911abe3e7854a26f39f" +
				"58b25c15342af52ae"),
			addrs: []btcutil.Address{
				newAddressPubKey(hexToBytes("04cc71eb30d653c0" +
					"c3163990c47b976f3fb3f37cccdcbedb169a" +
					"1dfef58bbfbfaff7d8a473e7e2e6d317b87b" +
					"afe8bde97e3cf8f065dec022b51d11fcdd0d" +
					"348ac4")),
				newAddressPubKey(hexToBytes("0461cbdcc5409fb4" +
					"b4d42b51d33381354d80e550078cb532a34b" +
					"fa2fcfdeb7d76519aecc62770f5b0e4ef855" +
					"1946d8a540911abe3e7854a26f39f58b25c1" +
					"5342af")),
			},
			reqSigs: 1,
			class:   MultiSigTy,
		},
//来自Real TX D646F82BD5FBDB94A36872CE460F97662B80C3050AD3209D1E398EA277AB，VIN 1
		{
			name: "standard 2 of 3 multisig",
			script: hexToBytes("524104cb9c3c222c5f7a7d3b9bd152f36" +
				"3a0b6d54c9eb312c4d4f9af1e8551b6c421a6a4ab0e2" +
				"9105f24de20ff463c1c91fcf3bf662cdde4783d4799f" +
				"787cb7c08869b4104ccc588420deeebea22a7e900cc8" +
				"b68620d2212c374604e3487ca08f1ff3ae12bdc63951" +
				"4d0ec8612a2d3c519f084d9a00cbbe3b53d071e9b09e" +
				"71e610b036aa24104ab47ad1939edcb3db65f7fedea6" +
				"2bbf781c5410d3f22a7a3a56ffefb2238af8627363bd" +
				"f2ed97c1f89784a1aecdb43384f11d2acc64443c7fc2" +
				"99cef0400421a53ae"),
			addrs: []btcutil.Address{
				newAddressPubKey(hexToBytes("04cb9c3c222c5f7a" +
					"7d3b9bd152f363a0b6d54c9eb312c4d4f9af" +
					"1e8551b6c421a6a4ab0e29105f24de20ff46" +
					"3c1c91fcf3bf662cdde4783d4799f787cb7c" +
					"08869b")),
				newAddressPubKey(hexToBytes("04ccc588420deeeb" +
					"ea22a7e900cc8b68620d2212c374604e3487" +
					"ca08f1ff3ae12bdc639514d0ec8612a2d3c5" +
					"19f084d9a00cbbe3b53d071e9b09e71e610b" +
					"036aa2")),
				newAddressPubKey(hexToBytes("04ab47ad1939edcb" +
					"3db65f7fedea62bbf781c5410d3f22a7a3a5" +
					"6ffefb2238af8627363bdf2ed97c1f89784a" +
					"1aecdb43384f11d2acc64443c7fc299cef04" +
					"00421a")),
			},
			reqSigs: 2,
			class:   MultiSigTy,
		},

//由于以下原因，以下是非标准脚本
//无效的PubKeys，分析失败，不属于
//标准格式。

		{
			name: "p2pk with uncompressed pk missing OP_CHECKSIG",
			script: hexToBytes("410411db93e1dcdb8a016b49840f8c53b" +
				"c1eb68a382e97b1482ecad7b148a6909a5cb2e0eaddf" +
				"b84ccf9744464f82e160bfa9b8b64f9d4c03f999b864" +
				"3f656b412a3"),
			addrs:   nil,
			reqSigs: 0,
			class:   NonStandardTy,
		},
		{
			name: "valid signature from a sigscript - no addresses",
			script: hexToBytes("47304402204e45e16932b8af514961a1d" +
				"3a1a25fdf3f4f7732e9d624c6c61548ab5fb8cd41022" +
				"0181522ec8eca07de4860a4acdd12909d831cc56cbba" +
				"c4622082221a8768d1d0901"),
			addrs:   nil,
			reqSigs: 0,
			class:   NonStandardTy,
		},
//注意pubkey在技术上是
//堆栈，但由于地址提取是有意的
//使用标准pkscripts时，不应返回任何
//地址。
		{
			name: "valid sigscript to reedeem p2pk - no addresses",
			script: hexToBytes("493046022100ddc69738bf2336318e4e0" +
				"41a5a77f305da87428ab1606f023260017854350ddc0" +
				"22100817af09d2eec36862d16009852b7e3a0f6dd765" +
				"98290b7834e1453660367e07a014104cd4240c198e12" +
				"523b6f9cb9f5bed06de1ba37e96a1bbd13745fcf9d11" +
				"c25b1dff9a519675d198804ba9962d3eca2d5937d58e" +
				"5a75a71042d40388a4d307f887d"),
			addrs:   nil,
			reqSigs: 0,
			class:   NonStandardTy,
		},
//来自Real Tx 691d277dc0e90a462a3d652a1171686de49cf19067cd33c7df0392833fb986a，凭证0
//无效的公钥
		{
			name: "1 of 3 multisig with invalid pubkeys",
			script: hexToBytes("51411c2200007353455857696b696c656" +
				"16b73204361626c6567617465204261636b75700a0a6" +
				"361626c65676174652d3230313031323034313831312" +
				"e377a0a0a446f41776e6c6f61642074686520666f6c6" +
				"c6f77696e67207472616e73616374696f6e732077697" +
				"468205361746f736869204e616b616d6f746f2773206" +
				"46f776e6c6f61416420746f6f6c2077686963680a636" +
				"16e20626520666f756e6420696e207472616e7361637" +
				"4696f6e2036633533636439383731313965663739376" +
				"435616463636453ae"),
			addrs:   []btcutil.Address{},
			reqSigs: 1,
			class:   MultiSigTy,
		},
//来自Real Tx:691d277dc0e90a462a3d652a1171686de49cf19067cd33c7df0392833fb986a，VOUT 44
//无效的公钥
		{
			name: "1 of 3 multisig with invalid pubkeys 2",
			script: hexToBytes("514134633365633235396337346461636" +
				"536666430383862343463656638630a6336366263313" +
				"93936633862393461333831316233363536313866653" +
				"16539623162354136636163636539393361333938386" +
				"134363966636336643664616266640a3236363363666" +
				"13963663463303363363039633539336333653931666" +
				"56465373032392131323364643432643235363339643" +
				"338613663663530616234636434340a00000053ae"),
			addrs:   []btcutil.Address{},
			reqSigs: 1,
			class:   MultiSigTy,
		},
		{
			name:    "empty script",
			script:  []byte{},
			addrs:   nil,
			reqSigs: 0,
			class:   NonStandardTy,
		},
		{
			name:    "script that does not parse",
			script:  []byte{OP_DATA_45},
			addrs:   nil,
			reqSigs: 0,
			class:   NonStandardTy,
		},
	}

	t.Logf("Running %d tests.", len(tests))
	for i, test := range tests {
		class, addrs, reqSigs, err := ExtractPkScriptAddrs(
			test.script, &chaincfg.MainNetParams)
		if err != nil {
		}

		if !reflect.DeepEqual(addrs, test.addrs) {
			t.Errorf("ExtractPkScriptAddrs #%d (%s) unexpected "+
				"addresses\ngot  %v\nwant %v", i, test.name,
				addrs, test.addrs)
			continue
		}

		if reqSigs != test.reqSigs {
			t.Errorf("ExtractPkScriptAddrs #%d (%s) unexpected "+
				"number of required signatures - got %d, "+
				"want %d", i, test.name, reqSigs, test.reqSigs)
			continue
		}

		if class != test.class {
			t.Errorf("ExtractPkScriptAddrs #%d (%s) unexpected "+
				"script type - got %s, want %s", i, test.name,
				class, test.class)
			continue
		}
	}
}

//testcalcscriptinfo确保calcscriptinfo提供预期的结果
//用于各种有效和无效的脚本对。
func TestCalcScriptInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sigScript string
		pkScript  string
		witness   []string

		bip16  bool
		segwit bool

		scriptInfo    ScriptInfo
		scriptInfoErr error
	}{
		{
//发明了脚本，哈希值不匹配
//以下测试的截断版本：
			name: "pkscript doesn't parse",
			sigScript: "1 81 DATA_8 2DUP EQUAL NOT VERIFY ABS " +
				"SWAP ABS EQUAL",
			pkScript: "HASH160 DATA_20 0xfe441065b6532231de2fac56" +
				"3152205ec4f59c",
			bip16:         true,
			scriptInfoErr: scriptError(ErrMalformedPush, ""),
		},
		{
			name: "sigScript doesn't parse",
//下面是p2sh脚本的截断版本。
			sigScript: "1 81 DATA_8 2DUP EQUAL NOT VERIFY ABS " +
				"SWAP ABS",
			pkScript: "HASH160 DATA_20 0xfe441065b6532231de2fac56" +
				"3152205ec4f59c74 EQUAL",
			bip16:         true,
			scriptInfoErr: scriptError(ErrMalformedPush, ""),
		},
		{
//发明了脚本，哈希值不匹配
			name: "p2sh standard script",
			sigScript: "1 81 DATA_25 DUP HASH160 DATA_20 0x010203" +
				"0405060708090a0b0c0d0e0f1011121314 EQUALVERIFY " +
				"CHECKSIG",
			pkScript: "HASH160 DATA_20 0xfe441065b6532231de2fac56" +
				"3152205ec4f59c74 EQUAL",
			bip16: true,
			scriptInfo: ScriptInfo{
				PkScriptClass:  ScriptHashTy,
				NumInputs:      3,
ExpectedInputs: 3, //非标准P2SH。
				SigOps:         1,
			},
		},
		{
//自567A53D1CE19CE3D00771188516848443996550536D0294C5D46D46C10E53B起
//来自区块链。
			name: "p2sh nonstandard script",
			sigScript: "1 81 DATA_8 2DUP EQUAL NOT VERIFY ABS " +
				"SWAP ABS EQUAL",
			pkScript: "HASH160 DATA_20 0xfe441065b6532231de2fac56" +
				"3152205ec4f59c74 EQUAL",
			bip16: true,
			scriptInfo: ScriptInfo{
				PkScriptClass:  ScriptHashTy,
				NumInputs:      3,
ExpectedInputs: -1, //非标准P2SH。
				SigOps:         0,
			},
		},
		{
//剧本是发明出来的，数字都是假的。
			name: "multisig script",
//额外的0 arg在最后为op-checkmultisig错误。
			sigScript: "1 1 1 0",
			pkScript: "3 " +
				"DATA_33 0x0102030405060708090a0b0c0d0e0f1011" +
				"12131415161718191a1b1c1d1e1f2021 DATA_33 " +
				"0x0102030405060708090a0b0c0d0e0f101112131415" +
				"161718191a1b1c1d1e1f2021 DATA_33 0x010203040" +
				"5060708090a0b0c0d0e0f101112131415161718191a1" +
				"b1c1d1e1f2021 3 CHECKMULTISIG",
			bip16: true,
			scriptInfo: ScriptInfo{
				PkScriptClass:  MultiSigTy,
				NumInputs:      4,
				ExpectedInputs: 4,
				SigOps:         3,
			},
		},
		{
//一个v0 p2wkh花费。
			name:     "p2wkh script",
			pkScript: "OP_0 DATA_20 0x365ab47888e150ff46f8d51bce36dcd680f1283f",
			witness: []string{
				"3045022100ee9fe8f9487afa977" +
					"6647ebcf0883ce0cd37454d7ce19889d34ba2c9" +
					"9ce5a9f402200341cb469d0efd3955acb9e46" +
					"f568d7e2cc10f9084aaff94ced6dc50a59134ad01",
				"03f0000d0639a22bfaf217e4c9428" +
					"9c2b0cc7fa1036f7fd5d9f61a9d6ec153100e",
			},
			segwit: true,
			scriptInfo: ScriptInfo{
				PkScriptClass:  WitnessV0PubKeyHashTy,
				NumInputs:      2,
				ExpectedInputs: 2,
				SigOps:         1,
			},
		},
		{
//嵌套的2SH V0
			name: "p2wkh nested inside p2sh",
			pkScript: "HASH160 DATA_20 " +
				"0xb3a84b564602a9d68b4c9f19c2ea61458ff7826c EQUAL",
			sigScript: "DATA_22 0x0014ad0ffa2e387f07e7ead14dc56d5a97dbd6ff5a23",
			witness: []string{
				"3045022100cb1c2ac1ff1d57d" +
					"db98f7bdead905f8bf5bcc8641b029ce8eef25" +
					"c75a9e22a4702203be621b5c86b771288706be5" +
					"a7eee1db4fceabf9afb7583c1cc6ee3f8297b21201",
				"03f0000d0639a22bfaf217e4c9" +
					"4289c2b0cc7fa1036f7fd5d9f61a9d6ec153100e",
			},
			segwit: true,
			bip16:  true,
			scriptInfo: ScriptInfo{
				PkScriptClass:  ScriptHashTy,
				NumInputs:      3,
				ExpectedInputs: 3,
				SigOps:         1,
			},
		},
		{
//一个v0 p2wsh开销。
			name: "p2wsh spend of a p2wkh witness script",
			pkScript: "0 DATA_32 0xe112b88a0cd87ba387f44" +
				"9d443ee2596eb353beb1f0351ab2cba8909d875db23",
			witness: []string{
				"3045022100cb1c2ac1ff1d57d" +
					"db98f7bdead905f8bf5bcc8641b029ce8eef25" +
					"c75a9e22a4702203be621b5c86b771288706be5" +
					"a7eee1db4fceabf9afb7583c1cc6ee3f8297b21201",
				"03f0000d0639a22bfaf217e4c9" +
					"4289c2b0cc7fa1036f7fd5d9f61a9d6ec153100e",
				"76a914064977cb7b4a2e0c9680df0ef696e9e0e296b39988ac",
			},
			segwit: true,
			scriptInfo: ScriptInfo{
				PkScriptClass:  WitnessV0ScriptHashTy,
				NumInputs:      3,
				ExpectedInputs: 3,
				SigOps:         1,
			},
		},
	}

	for _, test := range tests {
		sigScript := mustParseShortForm(test.sigScript)
		pkScript := mustParseShortForm(test.pkScript)

		var witness wire.TxWitness

		for _, witElement := range test.witness {
			wit, err := hex.DecodeString(witElement)
			if err != nil {
				t.Fatalf("unable to decode witness "+
					"element: %v", err)
			}

			witness = append(witness, wit)
		}

		si, err := CalcScriptInfo(sigScript, pkScript, witness,
			test.bip16, test.segwit)
		if e := tstCheckScriptError(err, test.scriptInfoErr); e != nil {
			t.Errorf("scriptinfo test %q: %v", test.name, e)
			continue
		}
		if err != nil {
			continue
		}

		if *si != test.scriptInfo {
			t.Errorf("%s: scriptinfo doesn't match expected. "+
				"got: %q expected %q", test.name, *si,
				test.scriptInfo)
			continue
		}
	}
}

//bogusaddress实现btructil.address接口，这样测试可以确保
//正确处理了不支持的地址类型。
type bogusAddress struct{}

//encodeaddress只返回一个空字符串。它的存在是为了满足
//地址接口。
func (b *bogusAddress) EncodeAddress() string {
	return ""
}

//scriptAddress只返回一个空字节片。它的存在是为了满足
//地址接口。
func (b *bogusAddress) ScriptAddress() []byte {
	return nil
}

//isfornet公开地躺在那里以满足btcutil.address接口。
func (b *bogusAddress) IsForNet(chainParams *chaincfg.Params) bool {
return true //为什么不？
}

//字符串只返回空字符串。它的存在是为了满足
//地址接口。
func (b *bogusAddress) String() string {
	return ""
}

//testpaytoaddrscript确保paytoaddrscript函数生成
//为各种类型的地址更正脚本。
func TestPayToAddrScript(t *testing.T) {
	t.Parallel()

//1MIRQ9BWYQCGVJPWKUGAPU5OUK2E2EY4GX型
	p2pkhMain, err := btcutil.NewAddressPubKeyHash(hexToBytes("e34cce70c86"+
		"373273efcc54ce7d2a491bb4a0e84"), &chaincfg.MainNetParams)
	if err != nil {
		t.Fatalf("Unable to create public key hash address: %v", err)
	}

//取自交易：
//b0539a45de13b3e0403909b8bd1a55b8cBe45fd4e3f3fda76f3a5f52835c29d
	p2shMain, _ := btcutil.NewAddressScriptHashFromHash(hexToBytes("e8c300"+
		"c87986efa84c37c0519929019ef86eb5b4"), &chaincfg.MainNetParams)
	if err != nil {
		t.Fatalf("Unable to create script hash address: %v", err)
	}

//主网p2pk 13cg6sj3yhuxo4cr2thljrnfug3gug
	p2pkCompressedMain, err := btcutil.NewAddressPubKey(hexToBytes("02192d"+
		"74d0cb94344c9569c2e77901573d8d7903c3ebec3a957724895dca52c6b4"),
		&chaincfg.MainNetParams)
	if err != nil {
		t.Fatalf("Unable to create pubkey address (compressed): %v",
			err)
	}
	p2pkCompressed2Main, err := btcutil.NewAddressPubKey(hexToBytes("03b0b"+
		"d634234abbb1ba1e986e884185c61cf43e001f9137f23c2c409273eb16e65"),
		&chaincfg.MainNetParams)
	if err != nil {
		t.Fatalf("Unable to create pubkey address (compressed 2): %v",
			err)
	}

	p2pkUncompressedMain, err := btcutil.NewAddressPubKey(hexToBytes("0411"+
		"db93e1dcdb8a016b49840f8c53bc1eb68a382e97b1482ecad7b148a6909a5"+
		"cb2e0eaddfb84ccf9744464f82e160bfa9b8b64f9d4c03f999b8643f656b4"+
		"12a3"), &chaincfg.MainNetParams)
	if err != nil {
		t.Fatalf("Unable to create pubkey address (uncompressed): %v",
			err)
	}

//为方便和
//保持水平测试尺寸较短。
	errUnsupportedAddress := scriptError(ErrUnsupportedAddress, "")

	tests := []struct {
		in       btcutil.Address
		expected string
		err      error
	}{
//支付到mainnet上的pubkey散列地址
		{
			p2pkhMain,
			"DUP HASH160 DATA_20 0xe34cce70c86373273efcc54ce7d2a4" +
				"91bb4a0e8488 CHECKSIG",
			nil,
		},
//支付到脚本主机上的哈希地址
		{
			p2shMain,
			"HASH160 DATA_20 0xe8c300c87986efa84c37c0519929019ef8" +
				"6eb5b4 EQUAL",
			nil,
		},
//支付到mainnet上的pubkey地址。压缩密钥。
		{
			p2pkCompressedMain,
			"DATA_33 0x02192d74d0cb94344c9569c2e77901573d8d7903c3" +
				"ebec3a957724895dca52c6b4 CHECKSIG",
			nil,
		},
//支付到mainnet上的pubkey地址。压缩键（另一种方式）。
		{
			p2pkCompressed2Main,
			"DATA_33 0x03b0bd634234abbb1ba1e986e884185c61cf43e001" +
				"f9137f23c2c409273eb16e65 CHECKSIG",
			nil,
		},
//支付到mainnet上的pubkey地址。未压缩的密钥。
		{
			p2pkUncompressedMain,
			"DATA_65 0x0411db93e1dcdb8a016b49840f8c53bc1eb68a382e" +
				"97b1482ecad7b148a6909a5cb2e0eaddfb84ccf97444" +
				"64f82e160bfa9b8b64f9d4c03f999b8643f656b412a3 " +
				"CHECKSIG",
			nil,
		},

//支持带零指针的地址类型。
		{(*btcutil.AddressPubKeyHash)(nil), "", errUnsupportedAddress},
		{(*btcutil.AddressScriptHash)(nil), "", errUnsupportedAddress},
		{(*btcutil.AddressPubKey)(nil), "", errUnsupportedAddress},

//不支持的地址类型。
		{&bogusAddress{}, "", errUnsupportedAddress},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		pkScript, err := PayToAddrScript(test.in)
		if e := tstCheckScriptError(err, test.err); e != nil {
			t.Errorf("PayToAddrScript #%d unexpected error - "+
				"got %v, want %v", i, err, test.err)
			continue
		}

		expected := mustParseShortForm(test.expected)
		if !bytes.Equal(pkScript, expected) {
			t.Errorf("PayToAddrScript #%d got: %x\nwant: %x",
				i, pkScript, expected)
			continue
		}
	}
}

//testmulsigscript确保musigscript函数返回预期的
//脚本和错误。
func TestMultiSigScript(t *testing.T) {
	t.Parallel()

//主网p2pk 13cg6sj3yhuxo4cr2thljrnfug3gug
	p2pkCompressedMain, err := btcutil.NewAddressPubKey(hexToBytes("02192d"+
		"74d0cb94344c9569c2e77901573d8d7903c3ebec3a957724895dca52c6b4"),
		&chaincfg.MainNetParams)
	if err != nil {
		t.Fatalf("Unable to create pubkey address (compressed): %v",
			err)
	}
	p2pkCompressed2Main, err := btcutil.NewAddressPubKey(hexToBytes("03b0b"+
		"d634234abbb1ba1e986e884185c61cf43e001f9137f23c2c409273eb16e65"),
		&chaincfg.MainNetParams)
	if err != nil {
		t.Fatalf("Unable to create pubkey address (compressed 2): %v",
			err)
	}

	p2pkUncompressedMain, err := btcutil.NewAddressPubKey(hexToBytes("0411"+
		"db93e1dcdb8a016b49840f8c53bc1eb68a382e97b1482ecad7b148a6909a5"+
		"cb2e0eaddfb84ccf9744464f82e160bfa9b8b64f9d4c03f999b8643f656b4"+
		"12a3"), &chaincfg.MainNetParams)
	if err != nil {
		t.Fatalf("Unable to create pubkey address (uncompressed): %v",
			err)
	}

	tests := []struct {
		keys      []*btcutil.AddressPubKey
		nrequired int
		expected  string
		err       error
	}{
		{
			[]*btcutil.AddressPubKey{
				p2pkCompressedMain,
				p2pkCompressed2Main,
			},
			1,
			"1 DATA_33 0x02192d74d0cb94344c9569c2e77901573d8d7903c" +
				"3ebec3a957724895dca52c6b4 DATA_33 0x03b0bd634" +
				"234abbb1ba1e986e884185c61cf43e001f9137f23c2c4" +
				"09273eb16e65 2 CHECKMULTISIG",
			nil,
		},
		{
			[]*btcutil.AddressPubKey{
				p2pkCompressedMain,
				p2pkCompressed2Main,
			},
			2,
			"2 DATA_33 0x02192d74d0cb94344c9569c2e77901573d8d7903c" +
				"3ebec3a957724895dca52c6b4 DATA_33 0x03b0bd634" +
				"234abbb1ba1e986e884185c61cf43e001f9137f23c2c4" +
				"09273eb16e65 2 CHECKMULTISIG",
			nil,
		},
		{
			[]*btcutil.AddressPubKey{
				p2pkCompressedMain,
				p2pkCompressed2Main,
			},
			3,
			"",
			scriptError(ErrTooManyRequiredSigs, ""),
		},
		{
			[]*btcutil.AddressPubKey{
				p2pkUncompressedMain,
			},
			1,
			"1 DATA_65 0x0411db93e1dcdb8a016b49840f8c53bc1eb68a382" +
				"e97b1482ecad7b148a6909a5cb2e0eaddfb84ccf97444" +
				"64f82e160bfa9b8b64f9d4c03f999b8643f656b412a3 " +
				"1 CHECKMULTISIG",
			nil,
		},
		{
			[]*btcutil.AddressPubKey{
				p2pkUncompressedMain,
			},
			2,
			"",
			scriptError(ErrTooManyRequiredSigs, ""),
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		script, err := MultiSigScript(test.keys, test.nrequired)
		if e := tstCheckScriptError(err, test.err); e != nil {
			t.Errorf("MultiSigScript #%d: %v", i, e)
			continue
		}

		expected := mustParseShortForm(test.expected)
		if !bytes.Equal(script, expected) {
			t.Errorf("MultiSigScript #%d got: %x\nwant: %x",
				i, script, expected)
			continue
		}
	}
}

//testcalcmulsigstats确保calmcmutilisigstats函数返回
//预期错误。
func TestCalcMultiSigStats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		script string
		err    error
	}{
		{
			name: "short script",
			script: "0x046708afdb0fe5548271967f1a67130b7105cd6a828" +
				"e03909a67962e0ea1f61d",
			err: scriptError(ErrMalformedPush, ""),
		},
		{
			name: "stack underflow",
			script: "RETURN DATA_41 0x046708afdb0fe5548271967f1a" +
				"67130b7105cd6a828e03909a67962e0ea1f61deb649f6" +
				"bc3f4cef308",
			err: scriptError(ErrNotMultisigScript, ""),
		},
		{
			name: "multisig script",
			script: "0 DATA_72 0x30450220106a3e4ef0b51b764a2887226" +
				"2ffef55846514dacbdcbbdd652c849d395b4384022100" +
				"e03ae554c3cbb40600d31dd46fc33f25e47bf8525b1fe" +
				"07282e3b6ecb5f3bb2801 CODESEPARATOR 1 DATA_33 " +
				"0x0232abdc893e7f0631364d7fd01cb33d24da45329a0" +
				"0357b3a7886211ab414d55a 1 CHECKMULTISIG",
			err: nil,
		},
	}

	for i, test := range tests {
		script := mustParseShortForm(test.script)
		_, _, err := CalcMultiSigStats(script)
		if e := tstCheckScriptError(err, test.err); e != nil {
			t.Errorf("CalcMultiSigStats #%d (%s): %v", i, test.name,
				e)
			continue
		}
	}
}

//ScriptClassTests包含几个用于确保各种类的测试脚本
//决心如预期般有效。它被定义为测试全局与
//在函数范围内，因为这跨越了标准测试和
//共识测试（付费脚本哈希是共识的一部分）。
var scriptClassTests = []struct {
	name   string
	script string
	class  ScriptClass
}{
	{
		name: "Pay Pubkey",
		script: "DATA_65 0x0411db93e1dcdb8a016b49840f8c53bc1eb68a382e" +
			"97b1482ecad7b148a6909a5cb2e0eaddfb84ccf9744464f82e16" +
			"0bfa9b8b64f9d4c03f999b8643f656b412a3 CHECKSIG",
		class: PubKeyTy,
	},
//德克萨斯州599E47A8114FE098103663029548811D2651991B62397E057F0C863C2BC9F9EA
	{
		name: "Pay PubkeyHash",
		script: "DUP HASH160 DATA_20 0x660d4ef3a743e3e696ad990364e555" +
			"c271ad504b EQUALVERIFY CHECKSIG",
		class: PubKeyHashTy,
	},
//Tx 6d36bc17e947ce00bb6f12f8e7a56a1585c5a3688ffa2b05e10b4743273a74b的一部分
//代码分隔符部分已删除。（比特币核心支票
//multisig类型也没有codesep）。
	{
		name: "multisig",
		script: "1 DATA_33 0x0232abdc893e7f0631364d7fd01cb33d24da4" +
			"5329a00357b3a7886211ab414d55a 1 CHECKMULTISIG",
		class: MultiSigTy,
	},
//德克萨斯州E5779B9E78F9650DEBC2893FD9636D827B26B4DDFA6A8172FE8708C924F5C39D
	{
		name: "P2SH",
		script: "HASH160 DATA_20 0x433ec2ac1ffa1b7b7d027f564529c57197f" +
			"9ae88 EQUAL",
		class: ScriptHashTy,
	},

	{
//无数据的空数据。
		name:   "nulldata no data",
		script: "RETURN",
		class:  NullDataTy,
	},
	{
//单次零推送的空数据。
		name:   "nulldata zero",
		script: "RETURN 0",
		class:  NullDataTy,
	},
	{
//使用小整数push的nulldata。
		name:   "nulldata small int",
		script: "RETURN 1",
		class:  NullDataTy,
	},
	{
//使用最大小整数推送的nulldata。
		name:   "nulldata max small int",
		script: "RETURN 16",
		class:  NullDataTy,
	},
	{
//使用小数据推送为空数据。
		name:   "nulldata small data",
		script: "RETURN DATA_8 0x046708afdb0fe554",
		class:  NullDataTy,
	},
	{
//带有60字节数据推送的规范nulldata。
		name: "canonical nulldata 60-byte push",
		script: "RETURN 0x3c 0x046708afdb0fe5548271967f1a67130b7105cd" +
			"6a828e03909a67962e0ea1f61deb649f6bc3f4cef3046708afdb" +
			"0fe5548271967f1a67130b7105cd6a",
		class: NullDataTy,
	},
	{
//带有60字节数据推送的非规范nulldata。
		name: "non-canonical nulldata 60-byte push",
		script: "RETURN PUSHDATA1 0x3c 0x046708afdb0fe5548271967f1a67" +
			"130b7105cd6a828e03909a67962e0ea1f61deb649f6bc3f4cef3" +
			"046708afdb0fe5548271967f1a67130b7105cd6a",
		class: NullDataTy,
	},
	{
//nulldata的最大允许数据被视为标准数据。
		name: "nulldata max standard push",
		script: "RETURN PUSHDATA1 0x50 0x046708afdb0fe5548271967f1a67" +
			"130b7105cd6a828e03909a67962e0ea1f61deb649f6bc3f4cef3" +
			"046708afdb0fe5548271967f1a67130b7105cd6a828e03909a67" +
			"962e0ea1f61deb649f6bc3f4cef3",
		class: NullDataTy,
	},
	{
//考虑的数据超过允许的最大值的空数据
//标准（因此不标准）
		name: "nulldata exceed max standard push",
		script: "RETURN PUSHDATA1 0x51 0x046708afdb0fe5548271967f1a67" +
			"130b7105cd6a828e03909a67962e0ea1f61deb649f6bc3f4cef3" +
			"046708afdb0fe5548271967f1a67130b7105cd6a828e03909a67" +
			"962e0ea1f61deb649f6bc3f4cef308",
		class: NonStandardTy,
	},
	{
//几乎为空数据，但在数据后添加一个额外的操作码
//使之不标准。
		name:   "almost nulldata",
		script: "RETURN 4 TRUE",
		class:  NonStandardTy,
	},

//接下来的几个几乎是multisig（它是更复杂的脚本类型）
//但随着各种变化，使之失败。
	{
//多搜索，但NSIG无效。
		name: "strange 1",
		script: "DUP DATA_33 0x0232abdc893e7f0631364d7fd01cb33d24da45" +
			"329a00357b3a7886211ab414d55a 1 CHECKMULTISIG",
		class: NonStandardTy,
	},
	{
//multisig，但pubkey无效。
		name:   "strange 2",
		script: "1 1 1 CHECKMULTISIG",
		class:  NonStandardTy,
	},
	{
//多搜索，但没有匹配的npubkeys操作码。
		name: "strange 3",
		script: "1 DATA_33 0x0232abdc893e7f0631364d7fd01cb33d24da4532" +
			"9a00357b3a7886211ab414d55a DATA_33 0x0232abdc893e7f0" +
			"631364d7fd01cb33d24da45329a00357b3a7886211ab414d55a " +
			"CHECKMULTISIG",
		class: NonStandardTy,
	},
	{
//multisig，但带有multisigverify。
		name: "strange 4",
		script: "1 DATA_33 0x0232abdc893e7f0631364d7fd01cb33d24da4532" +
			"9a00357b3a7886211ab414d55a 1 CHECKMULTISIGVERIFY",
		class: NonStandardTy,
	},
	{
//多段但长度错误。
		name:   "strange 5",
		script: "1 CHECKMULTISIG",
		class:  NonStandardTy,
	},
	{
		name:   "doesn't parse",
		script: "DATA_5 0x01020304",
		class:  NonStandardTy,
	},
	{
		name: "multisig script with wrong number of pubkeys",
		script: "2 " +
			"DATA_33 " +
			"0x027adf5df7c965a2d46203c781bd4dd8" +
			"21f11844136f6673af7cc5a4a05cd29380 " +
			"DATA_33 " +
			"0x02c08f3de8ee2de9be7bd770f4c10eb0" +
			"d6ff1dd81ee96eedd3a9d4aeaf86695e80 " +
			"3 CHECKMULTISIG",
		class: NonStandardTy,
	},

//新的标准Segwit脚本模板。
	{
//付费见证发布密钥哈希pk脚本。
		name:   "Pay To Witness PubkeyHash",
		script: "0 DATA_20 0x1d0f172a0ecb48aee1be1f2687d2963ae33f71a1",
		class:  WitnessV0PubKeyHashTy,
	},
	{
//付费见证脚本hash pk脚本。
		name:   "Pay To Witness Scripthash",
		script: "0 DATA_32 0x9f96ade4b41d5433f4eda31e1738ec2b36f6e7d1420d94a6af99801a88f7f7ff",
		class:  WitnessV0ScriptHashTy,
	},
}

//testscriptClass确保scriptclasstests中的所有脚本
//班级。
func TestScriptClass(t *testing.T) {
	t.Parallel()

	for _, test := range scriptClassTests {
		script := mustParseShortForm(test.script)
		class := GetScriptClass(script)
		if class != test.class {
			t.Errorf("%s: expected %s got %s (script %x)", test.name,
				test.class, class, script)
			continue
		}
	}
}

//teststringifyclass确保脚本类字符串返回预期的
//每个脚本类的字符串。
func TestStringifyClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		class    ScriptClass
		stringed string
	}{
		{
			name:     "nonstandardty",
			class:    NonStandardTy,
			stringed: "nonstandard",
		},
		{
			name:     "pubkey",
			class:    PubKeyTy,
			stringed: "pubkey",
		},
		{
			name:     "pubkeyhash",
			class:    PubKeyHashTy,
			stringed: "pubkeyhash",
		},
		{
			name:     "witnesspubkeyhash",
			class:    WitnessV0PubKeyHashTy,
			stringed: "witness_v0_keyhash",
		},
		{
			name:     "scripthash",
			class:    ScriptHashTy,
			stringed: "scripthash",
		},
		{
			name:     "witnessscripthash",
			class:    WitnessV0ScriptHashTy,
			stringed: "witness_v0_scripthash",
		},
		{
			name:     "multisigty",
			class:    MultiSigTy,
			stringed: "multisig",
		},
		{
			name:     "nulldataty",
			class:    NullDataTy,
			stringed: "nulldata",
		},
		{
			name:     "broken",
			class:    ScriptClass(255),
			stringed: "Invalid",
		},
	}

	for _, test := range tests {
		typeString := test.class.String()
		if typeString != test.stringed {
			t.Errorf("%s: got %#q, want %#q", test.name,
				typeString, test.stringed)
		}
	}
}

//testnulldatascript测试nulldatascript是否返回有效的脚本。
func TestNullDataScript(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected []byte
		err      error
		class    ScriptClass
	}{
		{
			name:     "small int",
			data:     hexToBytes("01"),
			expected: mustParseShortForm("RETURN 1"),
			err:      nil,
			class:    NullDataTy,
		},
		{
			name:     "max small int",
			data:     hexToBytes("10"),
			expected: mustParseShortForm("RETURN 16"),
			err:      nil,
			class:    NullDataTy,
		},
		{
			name: "data of size before OP_PUSHDATA1 is needed",
			data: hexToBytes("0102030405060708090a0b0c0d0e0f10111" +
				"2131415161718"),
			expected: mustParseShortForm("RETURN 0x18 0x01020304" +
				"05060708090a0b0c0d0e0f101112131415161718"),
			err:   nil,
			class: NullDataTy,
		},
		{
			name: "just right",
			data: hexToBytes("000102030405060708090a0b0c0d0e0f101" +
				"112131415161718191a1b1c1d1e1f202122232425262" +
				"728292a2b2c2d2e2f303132333435363738393a3b3c3" +
				"d3e3f404142434445464748494a4b4c4d4e4f"),
			expected: mustParseShortForm("RETURN PUSHDATA1 0x50 " +
				"0x000102030405060708090a0b0c0d0e0f101112131" +
				"415161718191a1b1c1d1e1f20212223242526272829" +
				"2a2b2c2d2e2f303132333435363738393a3b3c3d3e3" +
				"f404142434445464748494a4b4c4d4e4f"),
			err:   nil,
			class: NullDataTy,
		},
		{
			name: "too big",
			data: hexToBytes("000102030405060708090a0b0c0d0e0f101" +
				"112131415161718191a1b1c1d1e1f202122232425262" +
				"728292a2b2c2d2e2f303132333435363738393a3b3c3" +
				"d3e3f404142434445464748494a4b4c4d4e4f50"),
			expected: nil,
			err:      scriptError(ErrTooMuchNullData, ""),
			class:    NonStandardTy,
		},
	}

	for i, test := range tests {
		script, err := NullDataScript(test.data)
		if e := tstCheckScriptError(err, test.err); e != nil {
			t.Errorf("NullDataScript: #%d (%s): %v", i, test.name,
				e)
			continue

		}

//检查是否返回了预期结果。
		if !bytes.Equal(script, test.expected) {
			t.Errorf("NullDataScript: #%d (%s) wrong result\n"+
				"got: %x\nwant: %x", i, test.name, script,
				test.expected)
			continue
		}

//检查脚本的类型是否正确。
		scriptType := GetScriptClass(script)
		if scriptType != test.class {
			t.Errorf("GetScriptClass: #%d (%s) wrong result -- "+
				"got: %v, want: %v", i, test.name, scriptType,
				test.class)
			continue
		}
	}
}
