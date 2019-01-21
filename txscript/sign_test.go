
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

package txscript

import (
	"errors"
	"fmt"
	"testing"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

type addressToKey struct {
	key        *btcec.PrivateKey
	compressed bool
}

func mkGetKey(keys map[string]addressToKey) KeyDB {
	if keys == nil {
		return KeyClosure(func(addr btcutil.Address) (*btcec.PrivateKey,
			bool, error) {
			return nil, false, errors.New("nope")
		})
	}
	return KeyClosure(func(addr btcutil.Address) (*btcec.PrivateKey,
		bool, error) {
		a2k, ok := keys[addr.EncodeAddress()]
		if !ok {
			return nil, false, errors.New("nope")
		}
		return a2k.key, a2k.compressed, nil
	})
}

func mkGetScript(scripts map[string][]byte) ScriptDB {
	if scripts == nil {
		return ScriptClosure(func(addr btcutil.Address) ([]byte, error) {
			return nil, errors.New("nope")
		})
	}
	return ScriptClosure(func(addr btcutil.Address) ([]byte, error) {
		script, ok := scripts[addr.EncodeAddress()]
		if !ok {
			return nil, errors.New("nope")
		}
		return script, nil
	})
}

func checkScripts(msg string, tx *wire.MsgTx, idx int, inputAmt int64, sigScript, pkScript []byte) error {
	tx.TxIn[idx].SignatureScript = sigScript
	vm, err := NewEngine(pkScript, tx, idx,
		ScriptBip16|ScriptVerifyDERSignatures, nil, nil, inputAmt)
	if err != nil {
		return fmt.Errorf("failed to make script engine for %s: %v",
			msg, err)
	}

	err = vm.Execute()
	if err != nil {
		return fmt.Errorf("invalid script signature for %s: %v", msg,
			err)
	}

	return nil
}

func signAndCheck(msg string, tx *wire.MsgTx, idx int, inputAmt int64, pkScript []byte,
	hashType SigHashType, kdb KeyDB, sdb ScriptDB,
	previousScript []byte) error {

	sigScript, err := SignTxOutput(&chaincfg.TestNet3Params, tx, idx,
		pkScript, hashType, kdb, sdb, nil)
	if err != nil {
		return fmt.Errorf("failed to sign output %s: %v", msg, err)
	}

	return checkScripts(msg, tx, idx, inputAmt, sigScript, pkScript)
}

func TestSignTxOutput(t *testing.T) {
	t.Parallel()

//制作键
//根据键制作脚本。
//用魔法粉签名。
	hashTypes := []SigHashType{
SigHashOld, //不再使用，但应像所有
		SigHashAll,
		SigHashNone,
		SigHashSingle,
		SigHashAll | SigHashAnyOneCanPay,
		SigHashNone | SigHashAnyOneCanPay,
		SigHashSingle | SigHashAnyOneCanPay,
	}
	inputAmounts := []int64{5, 10, 15}
	tx := &wire.MsgTx{
		Version: 1,
		TxIn: []*wire.TxIn{
			{
				PreviousOutPoint: wire.OutPoint{
					Hash:  chainhash.Hash{},
					Index: 0,
				},
				Sequence: 4294967295,
			},
			{
				PreviousOutPoint: wire.OutPoint{
					Hash:  chainhash.Hash{},
					Index: 1,
				},
				Sequence: 4294967295,
			},
			{
				PreviousOutPoint: wire.OutPoint{
					Hash:  chainhash.Hash{},
					Index: 2,
				},
				Sequence: 4294967295,
			},
		},
		TxOut: []*wire.TxOut{
			{
				Value: 1,
			},
			{
				Value: 2,
			},
			{
				Value: 3,
			},
		},
		LockTime: 0,
	}

//支付到PubKey哈希（未压缩）
	for _, hashType := range hashTypes {
		for i := range tx.TxIn {
			msg := fmt.Sprintf("%d:%d", hashType, i)
			key, err := btcec.NewPrivateKey(btcec.S256())
			if err != nil {
				t.Errorf("failed to make privKey for %s: %v",
					msg, err)
				break
			}

			pk := (*btcec.PublicKey)(&key.PublicKey).
				SerializeUncompressed()
			address, err := btcutil.NewAddressPubKeyHash(
				btcutil.Hash160(pk), &chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make address for %s: %v",
					msg, err)
				break
			}

			pkScript, err := PayToAddrScript(address)
			if err != nil {
				t.Errorf("failed to make pkscript "+
					"for %s: %v", msg, err)
			}

			if err := signAndCheck(msg, tx, i, inputAmounts[i], pkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address.EncodeAddress(): {key, false},
				}), mkGetScript(nil), nil); err != nil {
				t.Error(err)
				break
			}
		}
	}

//支付到PubKey哈希（未压缩）（与正确的合并）
	for _, hashType := range hashTypes {
		for i := range tx.TxIn {
			msg := fmt.Sprintf("%d:%d", hashType, i)
			key, err := btcec.NewPrivateKey(btcec.S256())
			if err != nil {
				t.Errorf("failed to make privKey for %s: %v",
					msg, err)
				break
			}

			pk := (*btcec.PublicKey)(&key.PublicKey).
				SerializeUncompressed()
			address, err := btcutil.NewAddressPubKeyHash(
				btcutil.Hash160(pk), &chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make address for %s: %v",
					msg, err)
				break
			}

			pkScript, err := PayToAddrScript(address)
			if err != nil {
				t.Errorf("failed to make pkscript "+
					"for %s: %v", msg, err)
			}

			sigScript, err := SignTxOutput(&chaincfg.TestNet3Params,
				tx, i, pkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address.EncodeAddress(): {key, false},
				}), mkGetScript(nil), nil)
			if err != nil {
				t.Errorf("failed to sign output %s: %v", msg,
					err)
				break
			}

//通过上面的循环，这应该是有效的，现在签名
//再次合并。
			sigScript, err = SignTxOutput(&chaincfg.TestNet3Params,
				tx, i, pkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address.EncodeAddress(): {key, false},
				}), mkGetScript(nil), sigScript)
			if err != nil {
				t.Errorf("failed to sign output %s a "+
					"second time: %v", msg, err)
				break
			}

			err = checkScripts(msg, tx, i, inputAmounts[i], sigScript, pkScript)
			if err != nil {
				t.Errorf("twice signed script invalid for "+
					"%s: %v", msg, err)
				break
			}
		}
	}

//支付到PubKey哈希（压缩）
	for _, hashType := range hashTypes {
		for i := range tx.TxIn {
			msg := fmt.Sprintf("%d:%d", hashType, i)

			key, err := btcec.NewPrivateKey(btcec.S256())
			if err != nil {
				t.Errorf("failed to make privKey for %s: %v",
					msg, err)
				break
			}

			pk := (*btcec.PublicKey)(&key.PublicKey).
				SerializeCompressed()
			address, err := btcutil.NewAddressPubKeyHash(
				btcutil.Hash160(pk), &chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make address for %s: %v",
					msg, err)
				break
			}

			pkScript, err := PayToAddrScript(address)
			if err != nil {
				t.Errorf("failed to make pkscript "+
					"for %s: %v", msg, err)
			}

			if err := signAndCheck(msg, tx, i, inputAmounts[i],
				pkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address.EncodeAddress(): {key, true},
				}), mkGetScript(nil), nil); err != nil {
				t.Error(err)
				break
			}
		}
	}

//使用重复合并支付到PubKey哈希（压缩）
	for _, hashType := range hashTypes {
		for i := range tx.TxIn {
			msg := fmt.Sprintf("%d:%d", hashType, i)

			key, err := btcec.NewPrivateKey(btcec.S256())
			if err != nil {
				t.Errorf("failed to make privKey for %s: %v",
					msg, err)
				break
			}

			pk := (*btcec.PublicKey)(&key.PublicKey).
				SerializeCompressed()
			address, err := btcutil.NewAddressPubKeyHash(
				btcutil.Hash160(pk), &chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make address for %s: %v",
					msg, err)
				break
			}

			pkScript, err := PayToAddrScript(address)
			if err != nil {
				t.Errorf("failed to make pkscript "+
					"for %s: %v", msg, err)
			}

			sigScript, err := SignTxOutput(&chaincfg.TestNet3Params,
				tx, i, pkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address.EncodeAddress(): {key, true},
				}), mkGetScript(nil), nil)
			if err != nil {
				t.Errorf("failed to sign output %s: %v", msg,
					err)
				break
			}

//通过上面的循环，这应该是有效的，现在签名
//再次合并。
			sigScript, err = SignTxOutput(&chaincfg.TestNet3Params,
				tx, i, pkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address.EncodeAddress(): {key, true},
				}), mkGetScript(nil), sigScript)
			if err != nil {
				t.Errorf("failed to sign output %s a "+
					"second time: %v", msg, err)
				break
			}

			err = checkScripts(msg, tx, i, inputAmounts[i],
				sigScript, pkScript)
			if err != nil {
				t.Errorf("twice signed script invalid for "+
					"%s: %v", msg, err)
				break
			}
		}
	}

//支付到Pubkey（未压缩）
	for _, hashType := range hashTypes {
		for i := range tx.TxIn {
			msg := fmt.Sprintf("%d:%d", hashType, i)

			key, err := btcec.NewPrivateKey(btcec.S256())
			if err != nil {
				t.Errorf("failed to make privKey for %s: %v",
					msg, err)
				break
			}

			pk := (*btcec.PublicKey)(&key.PublicKey).
				SerializeUncompressed()
			address, err := btcutil.NewAddressPubKey(pk,
				&chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make address for %s: %v",
					msg, err)
				break
			}

			pkScript, err := PayToAddrScript(address)
			if err != nil {
				t.Errorf("failed to make pkscript "+
					"for %s: %v", msg, err)
			}

			if err := signAndCheck(msg, tx, i, inputAmounts[i],
				pkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address.EncodeAddress(): {key, false},
				}), mkGetScript(nil), nil); err != nil {
				t.Error(err)
				break
			}
		}
	}

//支付到Pubkey（未压缩）
	for _, hashType := range hashTypes {
		for i := range tx.TxIn {
			msg := fmt.Sprintf("%d:%d", hashType, i)

			key, err := btcec.NewPrivateKey(btcec.S256())
			if err != nil {
				t.Errorf("failed to make privKey for %s: %v",
					msg, err)
				break
			}

			pk := (*btcec.PublicKey)(&key.PublicKey).
				SerializeUncompressed()
			address, err := btcutil.NewAddressPubKey(pk,
				&chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make address for %s: %v",
					msg, err)
				break
			}

			pkScript, err := PayToAddrScript(address)
			if err != nil {
				t.Errorf("failed to make pkscript "+
					"for %s: %v", msg, err)
			}

			sigScript, err := SignTxOutput(&chaincfg.TestNet3Params,
				tx, i, pkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address.EncodeAddress(): {key, false},
				}), mkGetScript(nil), nil)
			if err != nil {
				t.Errorf("failed to sign output %s: %v", msg,
					err)
				break
			}

//通过上面的循环，这应该是有效的，现在签名
//再次合并。
			sigScript, err = SignTxOutput(&chaincfg.TestNet3Params,
				tx, i, pkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address.EncodeAddress(): {key, false},
				}), mkGetScript(nil), sigScript)
			if err != nil {
				t.Errorf("failed to sign output %s a "+
					"second time: %v", msg, err)
				break
			}

			err = checkScripts(msg, tx, i, inputAmounts[i], sigScript, pkScript)
			if err != nil {
				t.Errorf("twice signed script invalid for "+
					"%s: %v", msg, err)
				break
			}
		}
	}

//支付到Pubkey（压缩）
	for _, hashType := range hashTypes {
		for i := range tx.TxIn {
			msg := fmt.Sprintf("%d:%d", hashType, i)

			key, err := btcec.NewPrivateKey(btcec.S256())
			if err != nil {
				t.Errorf("failed to make privKey for %s: %v",
					msg, err)
				break
			}

			pk := (*btcec.PublicKey)(&key.PublicKey).
				SerializeCompressed()
			address, err := btcutil.NewAddressPubKey(pk,
				&chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make address for %s: %v",
					msg, err)
				break
			}

			pkScript, err := PayToAddrScript(address)
			if err != nil {
				t.Errorf("failed to make pkscript "+
					"for %s: %v", msg, err)
			}

			if err := signAndCheck(msg, tx, i, inputAmounts[i],
				pkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address.EncodeAddress(): {key, true},
				}), mkGetScript(nil), nil); err != nil {
				t.Error(err)
				break
			}
		}
	}

//使用重复合并支付到PubKey（压缩）
	for _, hashType := range hashTypes {
		for i := range tx.TxIn {
			msg := fmt.Sprintf("%d:%d", hashType, i)

			key, err := btcec.NewPrivateKey(btcec.S256())
			if err != nil {
				t.Errorf("failed to make privKey for %s: %v",
					msg, err)
				break
			}

			pk := (*btcec.PublicKey)(&key.PublicKey).
				SerializeCompressed()
			address, err := btcutil.NewAddressPubKey(pk,
				&chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make address for %s: %v",
					msg, err)
				break
			}

			pkScript, err := PayToAddrScript(address)
			if err != nil {
				t.Errorf("failed to make pkscript "+
					"for %s: %v", msg, err)
			}

			sigScript, err := SignTxOutput(&chaincfg.TestNet3Params,
				tx, i, pkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address.EncodeAddress(): {key, true},
				}), mkGetScript(nil), nil)
			if err != nil {
				t.Errorf("failed to sign output %s: %v", msg,
					err)
				break
			}

//通过上面的循环，这应该是有效的，现在签名
//再次合并。
			sigScript, err = SignTxOutput(&chaincfg.TestNet3Params,
				tx, i, pkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address.EncodeAddress(): {key, true},
				}), mkGetScript(nil), sigScript)
			if err != nil {
				t.Errorf("failed to sign output %s a "+
					"second time: %v", msg, err)
				break
			}

			err = checkScripts(msg, tx, i, inputAmounts[i],
				sigScript, pkScript)
			if err != nil {
				t.Errorf("twice signed script invalid for "+
					"%s: %v", msg, err)
				break
			}
		}
	}

//和以前一样，但现在有了p2sh。
//支付到PubKey哈希（未压缩）
	for _, hashType := range hashTypes {
		for i := range tx.TxIn {
			msg := fmt.Sprintf("%d:%d", hashType, i)
			key, err := btcec.NewPrivateKey(btcec.S256())
			if err != nil {
				t.Errorf("failed to make privKey for %s: %v",
					msg, err)
				break
			}

			pk := (*btcec.PublicKey)(&key.PublicKey).
				SerializeUncompressed()
			address, err := btcutil.NewAddressPubKeyHash(
				btcutil.Hash160(pk), &chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make address for %s: %v",
					msg, err)
				break
			}

			pkScript, err := PayToAddrScript(address)
			if err != nil {
				t.Errorf("failed to make pkscript "+
					"for %s: %v", msg, err)
				break
			}

			scriptAddr, err := btcutil.NewAddressScriptHash(
				pkScript, &chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make p2sh addr for %s: %v",
					msg, err)
				break
			}

			scriptPkScript, err := PayToAddrScript(
				scriptAddr)
			if err != nil {
				t.Errorf("failed to make script pkscript for "+
					"%s: %v", msg, err)
				break
			}

			if err := signAndCheck(msg, tx, i, inputAmounts[i],
				scriptPkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address.EncodeAddress(): {key, false},
				}), mkGetScript(map[string][]byte{
					scriptAddr.EncodeAddress(): pkScript,
				}), nil); err != nil {
				t.Error(err)
				break
			}
		}
	}

//使用重复合并支付到PubKey哈希（未压缩）
	for _, hashType := range hashTypes {
		for i := range tx.TxIn {
			msg := fmt.Sprintf("%d:%d", hashType, i)
			key, err := btcec.NewPrivateKey(btcec.S256())
			if err != nil {
				t.Errorf("failed to make privKey for %s: %v",
					msg, err)
				break
			}

			pk := (*btcec.PublicKey)(&key.PublicKey).
				SerializeUncompressed()
			address, err := btcutil.NewAddressPubKeyHash(
				btcutil.Hash160(pk), &chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make address for %s: %v",
					msg, err)
				break
			}

			pkScript, err := PayToAddrScript(address)
			if err != nil {
				t.Errorf("failed to make pkscript "+
					"for %s: %v", msg, err)
				break
			}

			scriptAddr, err := btcutil.NewAddressScriptHash(
				pkScript, &chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make p2sh addr for %s: %v",
					msg, err)
				break
			}

			scriptPkScript, err := PayToAddrScript(
				scriptAddr)
			if err != nil {
				t.Errorf("failed to make script pkscript for "+
					"%s: %v", msg, err)
				break
			}

			sigScript, err := SignTxOutput(&chaincfg.TestNet3Params,
				tx, i, scriptPkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address.EncodeAddress(): {key, false},
				}), mkGetScript(map[string][]byte{
					scriptAddr.EncodeAddress(): pkScript,
				}), nil)
			if err != nil {
				t.Errorf("failed to sign output %s: %v", msg,
					err)
				break
			}

//通过上面的循环，这应该是有效的，现在签名
//再次合并。
			sigScript, err = SignTxOutput(&chaincfg.TestNet3Params,
				tx, i, scriptPkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address.EncodeAddress(): {key, false},
				}), mkGetScript(map[string][]byte{
					scriptAddr.EncodeAddress(): pkScript,
				}), nil)
			if err != nil {
				t.Errorf("failed to sign output %s a "+
					"second time: %v", msg, err)
				break
			}

			err = checkScripts(msg, tx, i, inputAmounts[i],
				sigScript, scriptPkScript)
			if err != nil {
				t.Errorf("twice signed script invalid for "+
					"%s: %v", msg, err)
				break
			}
		}
	}

//支付到PubKey哈希（压缩）
	for _, hashType := range hashTypes {
		for i := range tx.TxIn {
			msg := fmt.Sprintf("%d:%d", hashType, i)

			key, err := btcec.NewPrivateKey(btcec.S256())
			if err != nil {
				t.Errorf("failed to make privKey for %s: %v",
					msg, err)
				break
			}

			pk := (*btcec.PublicKey)(&key.PublicKey).
				SerializeCompressed()
			address, err := btcutil.NewAddressPubKeyHash(
				btcutil.Hash160(pk), &chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make address for %s: %v",
					msg, err)
				break
			}

			pkScript, err := PayToAddrScript(address)
			if err != nil {
				t.Errorf("failed to make pkscript "+
					"for %s: %v", msg, err)
			}

			scriptAddr, err := btcutil.NewAddressScriptHash(
				pkScript, &chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make p2sh addr for %s: %v",
					msg, err)
				break
			}

			scriptPkScript, err := PayToAddrScript(
				scriptAddr)
			if err != nil {
				t.Errorf("failed to make script pkscript for "+
					"%s: %v", msg, err)
				break
			}

			if err := signAndCheck(msg, tx, i, inputAmounts[i],
				scriptPkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address.EncodeAddress(): {key, true},
				}), mkGetScript(map[string][]byte{
					scriptAddr.EncodeAddress(): pkScript,
				}), nil); err != nil {
				t.Error(err)
				break
			}
		}
	}

//使用重复合并支付到PubKey哈希（压缩）
	for _, hashType := range hashTypes {
		for i := range tx.TxIn {
			msg := fmt.Sprintf("%d:%d", hashType, i)

			key, err := btcec.NewPrivateKey(btcec.S256())
			if err != nil {
				t.Errorf("failed to make privKey for %s: %v",
					msg, err)
				break
			}

			pk := (*btcec.PublicKey)(&key.PublicKey).
				SerializeCompressed()
			address, err := btcutil.NewAddressPubKeyHash(
				btcutil.Hash160(pk), &chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make address for %s: %v",
					msg, err)
				break
			}

			pkScript, err := PayToAddrScript(address)
			if err != nil {
				t.Errorf("failed to make pkscript "+
					"for %s: %v", msg, err)
			}

			scriptAddr, err := btcutil.NewAddressScriptHash(
				pkScript, &chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make p2sh addr for %s: %v",
					msg, err)
				break
			}

			scriptPkScript, err := PayToAddrScript(
				scriptAddr)
			if err != nil {
				t.Errorf("failed to make script pkscript for "+
					"%s: %v", msg, err)
				break
			}

			sigScript, err := SignTxOutput(&chaincfg.TestNet3Params,
				tx, i, scriptPkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address.EncodeAddress(): {key, true},
				}), mkGetScript(map[string][]byte{
					scriptAddr.EncodeAddress(): pkScript,
				}), nil)
			if err != nil {
				t.Errorf("failed to sign output %s: %v", msg,
					err)
				break
			}

//通过上面的循环，这应该是有效的，现在签名
//再次合并。
			sigScript, err = SignTxOutput(&chaincfg.TestNet3Params,
				tx, i, scriptPkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address.EncodeAddress(): {key, true},
				}), mkGetScript(map[string][]byte{
					scriptAddr.EncodeAddress(): pkScript,
				}), nil)
			if err != nil {
				t.Errorf("failed to sign output %s a "+
					"second time: %v", msg, err)
				break
			}

			err = checkScripts(msg, tx, i, inputAmounts[i],
				sigScript, scriptPkScript)
			if err != nil {
				t.Errorf("twice signed script invalid for "+
					"%s: %v", msg, err)
				break
			}
		}
	}

//支付到Pubkey（未压缩）
	for _, hashType := range hashTypes {
		for i := range tx.TxIn {
			msg := fmt.Sprintf("%d:%d", hashType, i)

			key, err := btcec.NewPrivateKey(btcec.S256())
			if err != nil {
				t.Errorf("failed to make privKey for %s: %v",
					msg, err)
				break
			}

			pk := (*btcec.PublicKey)(&key.PublicKey).
				SerializeUncompressed()
			address, err := btcutil.NewAddressPubKey(pk,
				&chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make address for %s: %v",
					msg, err)
				break
			}

			pkScript, err := PayToAddrScript(address)
			if err != nil {
				t.Errorf("failed to make pkscript "+
					"for %s: %v", msg, err)
			}

			scriptAddr, err := btcutil.NewAddressScriptHash(
				pkScript, &chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make p2sh addr for %s: %v",
					msg, err)
				break
			}

			scriptPkScript, err := PayToAddrScript(
				scriptAddr)
			if err != nil {
				t.Errorf("failed to make script pkscript for "+
					"%s: %v", msg, err)
				break
			}

			if err := signAndCheck(msg, tx, i, inputAmounts[i],
				scriptPkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address.EncodeAddress(): {key, false},
				}), mkGetScript(map[string][]byte{
					scriptAddr.EncodeAddress(): pkScript,
				}), nil); err != nil {
				t.Error(err)
				break
			}
		}
	}

//使用重复合并支付到PubKey（未压缩）
	for _, hashType := range hashTypes {
		for i := range tx.TxIn {
			msg := fmt.Sprintf("%d:%d", hashType, i)

			key, err := btcec.NewPrivateKey(btcec.S256())
			if err != nil {
				t.Errorf("failed to make privKey for %s: %v",
					msg, err)
				break
			}

			pk := (*btcec.PublicKey)(&key.PublicKey).
				SerializeUncompressed()
			address, err := btcutil.NewAddressPubKey(pk,
				&chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make address for %s: %v",
					msg, err)
				break
			}

			pkScript, err := PayToAddrScript(address)
			if err != nil {
				t.Errorf("failed to make pkscript "+
					"for %s: %v", msg, err)
			}

			scriptAddr, err := btcutil.NewAddressScriptHash(
				pkScript, &chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make p2sh addr for %s: %v",
					msg, err)
				break
			}

			scriptPkScript, err := PayToAddrScript(scriptAddr)
			if err != nil {
				t.Errorf("failed to make script pkscript for "+
					"%s: %v", msg, err)
				break
			}

			sigScript, err := SignTxOutput(&chaincfg.TestNet3Params,
				tx, i, scriptPkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address.EncodeAddress(): {key, false},
				}), mkGetScript(map[string][]byte{
					scriptAddr.EncodeAddress(): pkScript,
				}), nil)
			if err != nil {
				t.Errorf("failed to sign output %s: %v", msg,
					err)
				break
			}

//通过上面的循环，这应该是有效的，现在签名
//再次合并。
			sigScript, err = SignTxOutput(&chaincfg.TestNet3Params,
				tx, i, scriptPkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address.EncodeAddress(): {key, false},
				}), mkGetScript(map[string][]byte{
					scriptAddr.EncodeAddress(): pkScript,
				}), nil)
			if err != nil {
				t.Errorf("failed to sign output %s a "+
					"second time: %v", msg, err)
				break
			}

			err = checkScripts(msg, tx, i, inputAmounts[i],
				sigScript, scriptPkScript)
			if err != nil {
				t.Errorf("twice signed script invalid for "+
					"%s: %v", msg, err)
				break
			}
		}
	}

//支付到Pubkey（压缩）
	for _, hashType := range hashTypes {
		for i := range tx.TxIn {
			msg := fmt.Sprintf("%d:%d", hashType, i)

			key, err := btcec.NewPrivateKey(btcec.S256())
			if err != nil {
				t.Errorf("failed to make privKey for %s: %v",
					msg, err)
				break
			}

			pk := (*btcec.PublicKey)(&key.PublicKey).
				SerializeCompressed()
			address, err := btcutil.NewAddressPubKey(pk,
				&chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make address for %s: %v",
					msg, err)
				break
			}

			pkScript, err := PayToAddrScript(address)
			if err != nil {
				t.Errorf("failed to make pkscript "+
					"for %s: %v", msg, err)
			}

			scriptAddr, err := btcutil.NewAddressScriptHash(
				pkScript, &chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make p2sh addr for %s: %v",
					msg, err)
				break
			}

			scriptPkScript, err := PayToAddrScript(scriptAddr)
			if err != nil {
				t.Errorf("failed to make script pkscript for "+
					"%s: %v", msg, err)
				break
			}

			if err := signAndCheck(msg, tx, i, inputAmounts[i],
				scriptPkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address.EncodeAddress(): {key, true},
				}), mkGetScript(map[string][]byte{
					scriptAddr.EncodeAddress(): pkScript,
				}), nil); err != nil {
				t.Error(err)
				break
			}
		}
	}

//支付到Pubkey（压缩）
	for _, hashType := range hashTypes {
		for i := range tx.TxIn {
			msg := fmt.Sprintf("%d:%d", hashType, i)

			key, err := btcec.NewPrivateKey(btcec.S256())
			if err != nil {
				t.Errorf("failed to make privKey for %s: %v",
					msg, err)
				break
			}

			pk := (*btcec.PublicKey)(&key.PublicKey).
				SerializeCompressed()
			address, err := btcutil.NewAddressPubKey(pk,
				&chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make address for %s: %v",
					msg, err)
				break
			}

			pkScript, err := PayToAddrScript(address)
			if err != nil {
				t.Errorf("failed to make pkscript "+
					"for %s: %v", msg, err)
			}

			scriptAddr, err := btcutil.NewAddressScriptHash(
				pkScript, &chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make p2sh addr for %s: %v",
					msg, err)
				break
			}

			scriptPkScript, err := PayToAddrScript(scriptAddr)
			if err != nil {
				t.Errorf("failed to make script pkscript for "+
					"%s: %v", msg, err)
				break
			}

			sigScript, err := SignTxOutput(&chaincfg.TestNet3Params,
				tx, i, scriptPkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address.EncodeAddress(): {key, true},
				}), mkGetScript(map[string][]byte{
					scriptAddr.EncodeAddress(): pkScript,
				}), nil)
			if err != nil {
				t.Errorf("failed to sign output %s: %v", msg,
					err)
				break
			}

//通过上面的循环，这应该是有效的，现在签名
//再次合并。
			sigScript, err = SignTxOutput(&chaincfg.TestNet3Params,
				tx, i, scriptPkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address.EncodeAddress(): {key, true},
				}), mkGetScript(map[string][]byte{
					scriptAddr.EncodeAddress(): pkScript,
				}), nil)
			if err != nil {
				t.Errorf("failed to sign output %s a "+
					"second time: %v", msg, err)
				break
			}

			err = checkScripts(msg, tx, i, inputAmounts[i],
				sigScript, scriptPkScript)
			if err != nil {
				t.Errorf("twice signed script invalid for "+
					"%s: %v", msg, err)
				break
			}
		}
	}

//基本多路转换器
	for _, hashType := range hashTypes {
		for i := range tx.TxIn {
			msg := fmt.Sprintf("%d:%d", hashType, i)

			key1, err := btcec.NewPrivateKey(btcec.S256())
			if err != nil {
				t.Errorf("failed to make privKey for %s: %v",
					msg, err)
				break
			}

			pk1 := (*btcec.PublicKey)(&key1.PublicKey).
				SerializeCompressed()
			address1, err := btcutil.NewAddressPubKey(pk1,
				&chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make address for %s: %v",
					msg, err)
				break
			}

			key2, err := btcec.NewPrivateKey(btcec.S256())
			if err != nil {
				t.Errorf("failed to make privKey 2 for %s: %v",
					msg, err)
				break
			}

			pk2 := (*btcec.PublicKey)(&key2.PublicKey).
				SerializeCompressed()
			address2, err := btcutil.NewAddressPubKey(pk2,
				&chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make address 2 for %s: %v",
					msg, err)
				break
			}

			pkScript, err := MultiSigScript(
				[]*btcutil.AddressPubKey{address1, address2},
				2)
			if err != nil {
				t.Errorf("failed to make pkscript "+
					"for %s: %v", msg, err)
			}

			scriptAddr, err := btcutil.NewAddressScriptHash(
				pkScript, &chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make p2sh addr for %s: %v",
					msg, err)
				break
			}

			scriptPkScript, err := PayToAddrScript(scriptAddr)
			if err != nil {
				t.Errorf("failed to make script pkscript for "+
					"%s: %v", msg, err)
				break
			}

			if err := signAndCheck(msg, tx, i, inputAmounts[i],
				scriptPkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address1.EncodeAddress(): {key1, true},
					address2.EncodeAddress(): {key2, true},
				}), mkGetScript(map[string][]byte{
					scriptAddr.EncodeAddress(): pkScript,
				}), nil); err != nil {
				t.Error(err)
				break
			}
		}
	}

//两份多册，先用一把钥匙签字，然后用另一把钥匙签字。
	for _, hashType := range hashTypes {
		for i := range tx.TxIn {
			msg := fmt.Sprintf("%d:%d", hashType, i)

			key1, err := btcec.NewPrivateKey(btcec.S256())
			if err != nil {
				t.Errorf("failed to make privKey for %s: %v",
					msg, err)
				break
			}

			pk1 := (*btcec.PublicKey)(&key1.PublicKey).
				SerializeCompressed()
			address1, err := btcutil.NewAddressPubKey(pk1,
				&chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make address for %s: %v",
					msg, err)
				break
			}

			key2, err := btcec.NewPrivateKey(btcec.S256())
			if err != nil {
				t.Errorf("failed to make privKey 2 for %s: %v",
					msg, err)
				break
			}

			pk2 := (*btcec.PublicKey)(&key2.PublicKey).
				SerializeCompressed()
			address2, err := btcutil.NewAddressPubKey(pk2,
				&chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make address 2 for %s: %v",
					msg, err)
				break
			}

			pkScript, err := MultiSigScript(
				[]*btcutil.AddressPubKey{address1, address2},
				2)
			if err != nil {
				t.Errorf("failed to make pkscript "+
					"for %s: %v", msg, err)
			}

			scriptAddr, err := btcutil.NewAddressScriptHash(
				pkScript, &chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make p2sh addr for %s: %v",
					msg, err)
				break
			}

			scriptPkScript, err := PayToAddrScript(scriptAddr)
			if err != nil {
				t.Errorf("failed to make script pkscript for "+
					"%s: %v", msg, err)
				break
			}

			sigScript, err := SignTxOutput(&chaincfg.TestNet3Params,
				tx, i, scriptPkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address1.EncodeAddress(): {key1, true},
				}), mkGetScript(map[string][]byte{
					scriptAddr.EncodeAddress(): pkScript,
				}), nil)
			if err != nil {
				t.Errorf("failed to sign output %s: %v", msg,
					err)
				break
			}

//只有2个签名中的1个，这个*应该*失败。
			if checkScripts(msg, tx, i, inputAmounts[i], sigScript,
				scriptPkScript) == nil {
				t.Errorf("part signed script valid for %s", msg)
				break
			}

//用另一个密钥签名并合并
			sigScript, err = SignTxOutput(&chaincfg.TestNet3Params,
				tx, i, scriptPkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address2.EncodeAddress(): {key2, true},
				}), mkGetScript(map[string][]byte{
					scriptAddr.EncodeAddress(): pkScript,
				}), sigScript)
			if err != nil {
				t.Errorf("failed to sign output %s: %v", msg, err)
				break
			}

			err = checkScripts(msg, tx, i, inputAmounts[i], sigScript,
				scriptPkScript)
			if err != nil {
				t.Errorf("fully signed script invalid for "+
					"%s: %v", msg, err)
				break
			}
		}
	}

//由两部分组成，先用一把钥匙签字，然后用两个钥匙签字，检查钥匙是否已清除。
//正确地。
	for _, hashType := range hashTypes {
		for i := range tx.TxIn {
			msg := fmt.Sprintf("%d:%d", hashType, i)

			key1, err := btcec.NewPrivateKey(btcec.S256())
			if err != nil {
				t.Errorf("failed to make privKey for %s: %v",
					msg, err)
				break
			}

			pk1 := (*btcec.PublicKey)(&key1.PublicKey).
				SerializeCompressed()
			address1, err := btcutil.NewAddressPubKey(pk1,
				&chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make address for %s: %v",
					msg, err)
				break
			}

			key2, err := btcec.NewPrivateKey(btcec.S256())
			if err != nil {
				t.Errorf("failed to make privKey 2 for %s: %v",
					msg, err)
				break
			}

			pk2 := (*btcec.PublicKey)(&key2.PublicKey).
				SerializeCompressed()
			address2, err := btcutil.NewAddressPubKey(pk2,
				&chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make address 2 for %s: %v",
					msg, err)
				break
			}

			pkScript, err := MultiSigScript(
				[]*btcutil.AddressPubKey{address1, address2},
				2)
			if err != nil {
				t.Errorf("failed to make pkscript "+
					"for %s: %v", msg, err)
			}

			scriptAddr, err := btcutil.NewAddressScriptHash(
				pkScript, &chaincfg.TestNet3Params)
			if err != nil {
				t.Errorf("failed to make p2sh addr for %s: %v",
					msg, err)
				break
			}

			scriptPkScript, err := PayToAddrScript(scriptAddr)
			if err != nil {
				t.Errorf("failed to make script pkscript for "+
					"%s: %v", msg, err)
				break
			}

			sigScript, err := SignTxOutput(&chaincfg.TestNet3Params,
				tx, i, scriptPkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address1.EncodeAddress(): {key1, true},
				}), mkGetScript(map[string][]byte{
					scriptAddr.EncodeAddress(): pkScript,
				}), nil)
			if err != nil {
				t.Errorf("failed to sign output %s: %v", msg,
					err)
				break
			}

//只有2个签名中的1个，这个*应该*失败。
			if checkScripts(msg, tx, i, inputAmounts[i], sigScript,
				scriptPkScript) == nil {
				t.Errorf("part signed script valid for %s", msg)
				break
			}

//用另一个密钥签名并合并
			sigScript, err = SignTxOutput(&chaincfg.TestNet3Params,
				tx, i, scriptPkScript, hashType,
				mkGetKey(map[string]addressToKey{
					address1.EncodeAddress(): {key1, true},
					address2.EncodeAddress(): {key2, true},
				}), mkGetScript(map[string][]byte{
					scriptAddr.EncodeAddress(): pkScript,
				}), sigScript)
			if err != nil {
				t.Errorf("failed to sign output %s: %v", msg, err)
				break
			}

//现在我们应该过去了。
			err = checkScripts(msg, tx, i, inputAmounts[i],
				sigScript, scriptPkScript)
			if err != nil {
				t.Errorf("fully signed script invalid for "+
					"%s: %v", msg, err)
				break
			}
		}
	}
}

type tstInput struct {
	txout              *wire.TxOut
	sigscriptGenerates bool
	inputValidates     bool
	indexOutOfRange    bool
}

type tstSigScript struct {
	name               string
	inputs             []tstInput
	hashType           SigHashType
	compress           bool
	scriptAtWrongIndex bool
}

var coinbaseOutPoint = &wire.OutPoint{
	Index: (1 << 32) - 1,
}

//预生成的私钥，带有关联的公钥和pkscripts
//对于未压缩和压缩的哈希160。
var (
	privKeyD = []byte{0x6b, 0x0f, 0xd8, 0xda, 0x54, 0x22, 0xd0, 0xb7,
		0xb4, 0xfc, 0x4e, 0x55, 0xd4, 0x88, 0x42, 0xb3, 0xa1, 0x65,
		0xac, 0x70, 0x7f, 0x3d, 0xa4, 0x39, 0x5e, 0xcb, 0x3b, 0xb0,
		0xd6, 0x0e, 0x06, 0x92}
	pubkeyX = []byte{0xb2, 0x52, 0xf0, 0x49, 0x85, 0x78, 0x03, 0x03, 0xc8,
		0x7d, 0xce, 0x51, 0x7f, 0xa8, 0x69, 0x0b, 0x91, 0x95, 0xf4,
		0xf3, 0x5c, 0x26, 0x73, 0x05, 0x05, 0xa2, 0xee, 0xbc, 0x09,
		0x38, 0x34, 0x3a}
	pubkeyY = []byte{0xb7, 0xc6, 0x7d, 0xb2, 0xe1, 0xff, 0xc8, 0x43, 0x1f,
		0x63, 0x32, 0x62, 0xaa, 0x60, 0xc6, 0x83, 0x30, 0xbd, 0x24,
		0x7e, 0xef, 0xdb, 0x6f, 0x2e, 0x8d, 0x56, 0xf0, 0x3c, 0x9f,
		0x6d, 0xb6, 0xf8}
	uncompressedPkScript = []byte{0x76, 0xa9, 0x14, 0xd1, 0x7c, 0xb5,
		0xeb, 0xa4, 0x02, 0xcb, 0x68, 0xe0, 0x69, 0x56, 0xbf, 0x32,
		0x53, 0x90, 0x0e, 0x0a, 0x86, 0xc9, 0xfa, 0x88, 0xac}
	compressedPkScript = []byte{0x76, 0xa9, 0x14, 0x27, 0x4d, 0x9f, 0x7f,
		0x61, 0x7e, 0x7c, 0x7a, 0x1c, 0x1f, 0xb2, 0x75, 0x79, 0x10,
		0x43, 0x65, 0x68, 0x27, 0x9d, 0x86, 0x88, 0xac}
	shortPkScript = []byte{0x76, 0xa9, 0x14, 0xd1, 0x7c, 0xb5,
		0xeb, 0xa4, 0x02, 0xcb, 0x68, 0xe0, 0x69, 0x56, 0xbf, 0x32,
		0x53, 0x90, 0x0e, 0x0a, 0x88, 0xac}
	uncompressedAddrStr = "1L6fd93zGmtzkK6CsZFVVoCwzZV3MUtJ4F"
	compressedAddrStr   = "14apLppt9zTq6cNw8SDfiJhk9PhkZrQtYZ"
)

//假设输出量。
const coinbaseVal = 2500000000
const fee = 5000000

var sigScriptTests = []tstSigScript{
	{
		name: "one input uncompressed",
		inputs: []tstInput{
			{
				txout:              wire.NewTxOut(coinbaseVal, uncompressedPkScript),
				sigscriptGenerates: true,
				inputValidates:     true,
				indexOutOfRange:    false,
			},
		},
		hashType:           SigHashAll,
		compress:           false,
		scriptAtWrongIndex: false,
	},
	{
		name: "two inputs uncompressed",
		inputs: []tstInput{
			{
				txout:              wire.NewTxOut(coinbaseVal, uncompressedPkScript),
				sigscriptGenerates: true,
				inputValidates:     true,
				indexOutOfRange:    false,
			},
			{
				txout:              wire.NewTxOut(coinbaseVal+fee, uncompressedPkScript),
				sigscriptGenerates: true,
				inputValidates:     true,
				indexOutOfRange:    false,
			},
		},
		hashType:           SigHashAll,
		compress:           false,
		scriptAtWrongIndex: false,
	},
	{
		name: "one input compressed",
		inputs: []tstInput{
			{
				txout:              wire.NewTxOut(coinbaseVal, compressedPkScript),
				sigscriptGenerates: true,
				inputValidates:     true,
				indexOutOfRange:    false,
			},
		},
		hashType:           SigHashAll,
		compress:           true,
		scriptAtWrongIndex: false,
	},
	{
		name: "two inputs compressed",
		inputs: []tstInput{
			{
				txout:              wire.NewTxOut(coinbaseVal, compressedPkScript),
				sigscriptGenerates: true,
				inputValidates:     true,
				indexOutOfRange:    false,
			},
			{
				txout:              wire.NewTxOut(coinbaseVal+fee, compressedPkScript),
				sigscriptGenerates: true,
				inputValidates:     true,
				indexOutOfRange:    false,
			},
		},
		hashType:           SigHashAll,
		compress:           true,
		scriptAtWrongIndex: false,
	},
	{
		name: "hashType SigHashNone",
		inputs: []tstInput{
			{
				txout:              wire.NewTxOut(coinbaseVal, uncompressedPkScript),
				sigscriptGenerates: true,
				inputValidates:     true,
				indexOutOfRange:    false,
			},
		},
		hashType:           SigHashNone,
		compress:           false,
		scriptAtWrongIndex: false,
	},
	{
		name: "hashType SigHashSingle",
		inputs: []tstInput{
			{
				txout:              wire.NewTxOut(coinbaseVal, uncompressedPkScript),
				sigscriptGenerates: true,
				inputValidates:     true,
				indexOutOfRange:    false,
			},
		},
		hashType:           SigHashSingle,
		compress:           false,
		scriptAtWrongIndex: false,
	},
	{
		name: "hashType SigHashAnyoneCanPay",
		inputs: []tstInput{
			{
				txout:              wire.NewTxOut(coinbaseVal, uncompressedPkScript),
				sigscriptGenerates: true,
				inputValidates:     true,
				indexOutOfRange:    false,
			},
		},
		hashType:           SigHashAnyOneCanPay,
		compress:           false,
		scriptAtWrongIndex: false,
	},
	{
		name: "hashType non-standard",
		inputs: []tstInput{
			{
				txout:              wire.NewTxOut(coinbaseVal, uncompressedPkScript),
				sigscriptGenerates: true,
				inputValidates:     true,
				indexOutOfRange:    false,
			},
		},
		hashType:           0x04,
		compress:           false,
		scriptAtWrongIndex: false,
	},
	{
		name: "invalid compression",
		inputs: []tstInput{
			{
				txout:              wire.NewTxOut(coinbaseVal, uncompressedPkScript),
				sigscriptGenerates: true,
				inputValidates:     false,
				indexOutOfRange:    false,
			},
		},
		hashType:           SigHashAll,
		compress:           true,
		scriptAtWrongIndex: false,
	},
	{
		name: "short PkScript",
		inputs: []tstInput{
			{
				txout:              wire.NewTxOut(coinbaseVal, shortPkScript),
				sigscriptGenerates: false,
				indexOutOfRange:    false,
			},
		},
		hashType:           SigHashAll,
		compress:           false,
		scriptAtWrongIndex: false,
	},
	{
		name: "valid script at wrong index",
		inputs: []tstInput{
			{
				txout:              wire.NewTxOut(coinbaseVal, uncompressedPkScript),
				sigscriptGenerates: true,
				inputValidates:     true,
				indexOutOfRange:    false,
			},
			{
				txout:              wire.NewTxOut(coinbaseVal+fee, uncompressedPkScript),
				sigscriptGenerates: true,
				inputValidates:     true,
				indexOutOfRange:    false,
			},
		},
		hashType:           SigHashAll,
		compress:           false,
		scriptAtWrongIndex: true,
	},
	{
		name: "index out of range",
		inputs: []tstInput{
			{
				txout:              wire.NewTxOut(coinbaseVal, uncompressedPkScript),
				sigscriptGenerates: true,
				inputValidates:     true,
				indexOutOfRange:    false,
			},
			{
				txout:              wire.NewTxOut(coinbaseVal+fee, uncompressedPkScript),
				sigscriptGenerates: true,
				inputValidates:     true,
				indexOutOfRange:    false,
			},
		},
		hashType:           SigHashAll,
		compress:           false,
		scriptAtWrongIndex: true,
	},
}

//测试sigscript生成的有效和无效输入，全部
//hashtypes，有或没有压缩。此测试创建
//使用假coinbase输入的sigscripts，因为sigscripts不能
//在txtests中为msgtx创建，因为它们来自区块链
//我们没有私人钥匙。
func TestSignatureScript(t *testing.T) {
	t.Parallel()

	privKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), privKeyD)

nexttest:
	for i := range sigScriptTests {
		tx := wire.NewMsgTx(wire.TxVersion)

		output := wire.NewTxOut(500, []byte{OP_RETURN})
		tx.AddTxOut(output)

		for range sigScriptTests[i].inputs {
			txin := wire.NewTxIn(coinbaseOutPoint, nil, nil)
			tx.AddTxIn(txin)
		}

		var script []byte
		var err error
		for j := range tx.TxIn {
			var idx int
			if sigScriptTests[i].inputs[j].indexOutOfRange {
				t.Errorf("at test %v", sigScriptTests[i].name)
				idx = len(sigScriptTests[i].inputs)
			} else {
				idx = j
			}
			script, err = SignatureScript(tx, idx,
				sigScriptTests[i].inputs[j].txout.PkScript,
				sigScriptTests[i].hashType, privKey,
				sigScriptTests[i].compress)

			if (err == nil) != sigScriptTests[i].inputs[j].sigscriptGenerates {
				if err == nil {
					t.Errorf("passed test '%v' incorrectly",
						sigScriptTests[i].name)
				} else {
					t.Errorf("failed test '%v': %v",
						sigScriptTests[i].name, err)
				}
				continue nexttest
			}
			if !sigScriptTests[i].inputs[j].sigscriptGenerates {
//完成了这个测试
				continue nexttest
			}

			tx.TxIn[j].SignatureScript = script
		}

//如果使用正确的sigscript进行测试，但
//索引，对第一个输入使用最后一个输入脚本。要求＞0
//测试输入。
		if sigScriptTests[i].scriptAtWrongIndex {
			tx.TxIn[0].SignatureScript = script
			sigScriptTests[i].inputs[0].inputValidates = false
		}

//验证Tx输入脚本
		scriptFlags := ScriptBip16 | ScriptVerifyDERSignatures
		for j := range tx.TxIn {
			vm, err := NewEngine(sigScriptTests[i].
				inputs[j].txout.PkScript, tx, j, scriptFlags, nil, nil, 0)
			if err != nil {
				t.Errorf("cannot create script vm for test %v: %v",
					sigScriptTests[i].name, err)
				continue nexttest
			}
			err = vm.Execute()
			if (err == nil) != sigScriptTests[i].inputs[j].inputValidates {
				if err == nil {
					t.Errorf("passed test '%v' validation incorrectly: %v",
						sigScriptTests[i].name, err)
				} else {
					t.Errorf("failed test '%v' validation: %v",
						sigScriptTests[i].name, err)
				}
				continue nexttest
			}
		}
	}
}
