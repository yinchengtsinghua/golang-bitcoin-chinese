
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package blockchain

import (
	"bytes"
	"errors"
	"math/big"
	"reflect"
	"testing"

	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcd/wire"
)

//测试器无主控确保与无主控工作相关的功能
//果不其然。
func TestErrNotInMainChain(t *testing.T) {
	errStr := "no block at height 1 exists"
	err := error(errNotInMainChain(errStr))

//确保错误的字符串化输出符合预期。
	if err.Error() != errStr {
		t.Fatalf("errNotInMainChain retuned unexpected error string - "+
			"got %q, want %q", err.Error(), errStr)
	}

//确保检测到错误类型正确。
	if !isNotInMainChainErr(err) {
		t.Fatalf("isNotInMainChainErr did not detect as expected type")
	}
	err = errors.New("something else")
	if isNotInMainChainErr(err) {
		t.Fatalf("isNotInMainChainErr detected incorrect type")
	}
}

//teststxoserialization确保对已用事务进行序列化和反序列化
//输出条目按预期工作。
func TestStxoSerialization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		stxo       SpentTxOut
		serialized []byte
	}{
//来自主区块链中的区块170。
		{
			name: "Spends last output of coinbase",
			stxo: SpentTxOut{
				Amount:     5000000000,
				PkScript:   hexToBytes("410411db93e1dcdb8a016b49840f8c53bc1eb68a382e97b1482ecad7b148a6909a5cb2e0eaddfb84ccf9744464f82e160bfa9b8b64f9d4c03f999b8643f656b412a3ac"),
				IsCoinBase: true,
				Height:     9,
			},
			serialized: hexToBytes("1300320511db93e1dcdb8a016b49840f8c53bc1eb68a382e97b1482ecad7b148a6909a5c"),
		},
//改编自主区块链中的区块100025。
		{
			name: "Spends last output of non coinbase",
			stxo: SpentTxOut{
				Amount:     13761000000,
				PkScript:   hexToBytes("76a914b2fb57eadf61e106a100a7445a8c3f67898841ec88ac"),
				IsCoinBase: false,
				Height:     100024,
			},
			serialized: hexToBytes("8b99700086c64700b2fb57eadf61e106a100a7445a8c3f67898841ec"),
		},
//改编自主区块链中的区块100025。
		{
			name: "Does not spend last output, legacy format",
			stxo: SpentTxOut{
				Amount:   34405000000,
				PkScript: hexToBytes("76a9146edbc6c4d31bae9f1ccc38538a114bf42de65e8688ac"),
			},
			serialized: hexToBytes("0091f20f006edbc6c4d31bae9f1ccc38538a114bf42de65e86"),
		},
	}

	for _, test := range tests {
//确保函数计算序列化大小
//实际上，对它进行序列化是正确计算的。
		gotSize := spentTxOutSerializeSize(&test.stxo)
		if gotSize != len(test.serialized) {
			t.Errorf("SpentTxOutSerializeSize (%s): did not get "+
				"expected size - got %d, want %d", test.name,
				gotSize, len(test.serialized))
			continue
		}

//
		gotSerialized := make([]byte, gotSize)
		gotBytesWritten := putSpentTxOut(gotSerialized, &test.stxo)
		if !bytes.Equal(gotSerialized, test.serialized) {
			t.Errorf("putSpentTxOut (%s): did not get expected "+
				"bytes - got %x, want %x", test.name,
				gotSerialized, test.serialized)
			continue
		}
		if gotBytesWritten != len(test.serialized) {
			t.Errorf("putSpentTxOut (%s): did not get expected "+
				"number of bytes written - got %d, want %d",
				test.name, gotBytesWritten,
				len(test.serialized))
			continue
		}

//确保将序列化字节解码回预期的
//STXO
		var gotStxo SpentTxOut
		gotBytesRead, err := decodeSpentTxOut(test.serialized, &gotStxo)
		if err != nil {
			t.Errorf("decodeSpentTxOut (%s): unexpected error: %v",
				test.name, err)
			continue
		}
		if !reflect.DeepEqual(gotStxo, test.stxo) {
			t.Errorf("decodeSpentTxOut (%s) mismatched entries - "+
				"got %v, want %v", test.name, gotStxo, test.stxo)
			continue
		}
		if gotBytesRead != len(test.serialized) {
			t.Errorf("decodeSpentTxOut (%s): did not get expected "+
				"number of bytes read - got %d, want %d",
				test.name, gotBytesRead, len(test.serialized))
			continue
		}
	}
}

//teststxodecodeerrors对花费的解码执行负测试
//事务输出以确保错误路径按预期工作。
func TestStxoDecodeErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		stxo       SpentTxOut
		serialized []byte
bytesRead  int //预期的读取字节数。
		errType    error
	}{
		{
			name:       "nothing serialized",
			stxo:       SpentTxOut{},
			serialized: hexToBytes(""),
			errType:    errDeserialize(""),
			bytesRead:  0,
		},
		{
			name:       "no data after header code w/o reserved",
			stxo:       SpentTxOut{},
			serialized: hexToBytes("00"),
			errType:    errDeserialize(""),
			bytesRead:  1,
		},
		{
			name:       "no data after header code with reserved",
			stxo:       SpentTxOut{},
			serialized: hexToBytes("13"),
			errType:    errDeserialize(""),
			bytesRead:  1,
		},
		{
			name:       "no data after reserved",
			stxo:       SpentTxOut{},
			serialized: hexToBytes("1300"),
			errType:    errDeserialize(""),
			bytesRead:  2,
		},
		{
			name:       "incomplete compressed txout",
			stxo:       SpentTxOut{},
			serialized: hexToBytes("1332"),
			errType:    errDeserialize(""),
			bytesRead:  2,
		},
	}

	for _, test := range tests {
//确保返回预期的错误类型。
		gotBytesRead, err := decodeSpentTxOut(test.serialized,
			&test.stxo)
		if reflect.TypeOf(err) != reflect.TypeOf(test.errType) {
			t.Errorf("decodeSpentTxOut (%s): expected error type "+
				"does not match - got %T, want %T", test.name,
				err, test.errType)
			continue
		}

//确保返回预期的读取字节数。
		if gotBytesRead != test.bytesRead {
			t.Errorf("decodeSpentTxOut (%s): unexpected number of "+
				"bytes read - got %d, want %d", test.name,
				gotBytesRead, test.bytesRead)
			continue
		}
	}
}

//testspendjournalSerialization确保序列化和反序列化支出
//日记账分录按预期工作。
func TestSpendJournalSerialization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		entry      []SpentTxOut
		blockTxns  []*wire.MsgTx
		serialized []byte
	}{
//来自主区块链中的区块2。
		{
			name:       "No spends",
			entry:      nil,
			blockTxns:  nil,
			serialized: nil,
		},
//来自主区块链中的区块170。
		{
			name: "One tx with one input spends last output of coinbase",
			entry: []SpentTxOut{{
				Amount:     5000000000,
				PkScript:   hexToBytes("410411db93e1dcdb8a016b49840f8c53bc1eb68a382e97b1482ecad7b148a6909a5cb2e0eaddfb84ccf9744464f82e160bfa9b8b64f9d4c03f999b8643f656b412a3ac"),
				IsCoinBase: true,
				Height:     9,
			}},
blockTxns: []*wire.MsgTx{{ //省略了coinbase。
				Version: 1,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: wire.OutPoint{
						Hash:  *newHashFromStr("0437cd7f8525ceed2324359c2d0ba26006d92d856a9c20fa0241106ee5a597c9"),
						Index: 0,
					},
					SignatureScript: hexToBytes("47304402204e45e16932b8af514961a1d3a1a25fdf3f4f7732e9d624c6c61548ab5fb8cd410220181522ec8eca07de4860a4acdd12909d831cc56cbbac4622082221a8768d1d0901"),
					Sequence:        0xffffffff,
				}},
				TxOut: []*wire.TxOut{{
					Value:    1000000000,
					PkScript: hexToBytes("4104ae1a62fe09c5f51b13905f07f06b99a2f7159b2225f374cd378d71302fa28414e7aab37397f554a7df5f142c21c1b7303b8a0626f1baded5c72a704f7e6cd84cac"),
				}, {
					Value:    4000000000,
					PkScript: hexToBytes("410411db93e1dcdb8a016b49840f8c53bc1eb68a382e97b1482ecad7b148a6909a5cb2e0eaddfb84ccf9744464f82e160bfa9b8b64f9d4c03f999b8643f656b412a3ac"),
				}},
				LockTime: 0,
			}},
			serialized: hexToBytes("1300320511db93e1dcdb8a016b49840f8c53bc1eb68a382e97b1482ecad7b148a6909a5c"),
		},
//改编自主区块链中的区块100025。
		{
			name: "Two txns when one spends last output, one doesn't",
			entry: []SpentTxOut{{
				Amount:     34405000000,
				PkScript:   hexToBytes("76a9146edbc6c4d31bae9f1ccc38538a114bf42de65e8688ac"),
				IsCoinBase: false,
				Height:     100024,
			}, {
				Amount:     13761000000,
				PkScript:   hexToBytes("76a914b2fb57eadf61e106a100a7445a8c3f67898841ec88ac"),
				IsCoinBase: false,
				Height:     100024,
			}},
blockTxns: []*wire.MsgTx{{ //省略了coinbase。
				Version: 1,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: wire.OutPoint{
						Hash:  *newHashFromStr("c0ed017828e59ad5ed3cf70ee7c6fb0f426433047462477dc7a5d470f987a537"),
						Index: 1,
					},
					SignatureScript: hexToBytes("493046022100c167eead9840da4a033c9a56470d7794a9bb1605b377ebe5688499b39f94be59022100fb6345cab4324f9ea0b9ee9169337534834638d818129778370f7d378ee4a325014104d962cac5390f12ddb7539507065d0def320d68c040f2e73337c3a1aaaab7195cb5c4d02e0959624d534f3c10c3cf3d73ca5065ebd62ae986b04c6d090d32627c"),
					Sequence:        0xffffffff,
				}},
				TxOut: []*wire.TxOut{{
					Value:    5000000,
					PkScript: hexToBytes("76a914f419b8db4ba65f3b6fcc233acb762ca6f51c23d488ac"),
				}, {
					Value:    34400000000,
					PkScript: hexToBytes("76a914cadf4fc336ab3c6a4610b75f31ba0676b7f663d288ac"),
				}},
				LockTime: 0,
			}, {
				Version: 1,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: wire.OutPoint{
						Hash:  *newHashFromStr("92fbe1d4be82f765dfabc9559d4620864b05cc897c4db0e29adac92d294e52b7"),
						Index: 0,
					},
					SignatureScript: hexToBytes("483045022100e256743154c097465cf13e89955e1c9ff2e55c46051b627751dee0144183157e02201d8d4f02cde8496aae66768f94d35ce54465bd4ae8836004992d3216a93a13f00141049d23ce8686fe9b802a7a938e8952174d35dd2c2089d4112001ed8089023ab4f93a3c9fcd5bfeaa9727858bf640dc1b1c05ec3b434bb59837f8640e8810e87742"),
					Sequence:        0xffffffff,
				}},
				TxOut: []*wire.TxOut{{
					Value:    5000000,
					PkScript: hexToBytes("76a914a983ad7c92c38fc0e2025212e9f972204c6e687088ac"),
				}, {
					Value:    13756000000,
					PkScript: hexToBytes("76a914a6ebd69952ab486a7a300bfffdcb395dc7d47c2388ac"),
				}},
				LockTime: 0,
			}},
			serialized: hexToBytes("8b99700086c64700b2fb57eadf61e106a100a7445a8c3f67898841ec8b99700091f20f006edbc6c4d31bae9f1ccc38538a114bf42de65e86"),
		},
	}

	for i, test := range tests {
//确保日记条目序列化为预期值。
		gotBytes := serializeSpendJournalEntry(test.entry)
		if !bytes.Equal(gotBytes, test.serialized) {
			t.Errorf("serializeSpendJournalEntry #%d (%s): "+
				"mismatched bytes - got %x, want %x", i,
				test.name, gotBytes, test.serialized)
			continue
		}

//反序列化为支出日记条目。
		gotEntry, err := deserializeSpendJournalEntry(test.serialized,
			test.blockTxns)
		if err != nil {
			t.Errorf("deserializeSpendJournalEntry #%d (%s) "+
				"unexpected error: %v", i, test.name, err)
			continue
		}

//确保反序列化的支出日记条目具有
//正确的属性。
		if !reflect.DeepEqual(gotEntry, test.entry) {
			t.Errorf("deserializeSpendJournalEntry #%d (%s) "+
				"mismatched entries - got %v, want %v",
				i, test.name, gotEntry, test.entry)
			continue
		}
	}
}

//testspendjournalErrors对反序列化支出执行负测试
//日记条目以确保错误路径按预期工作。
func TestSpendJournalErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		blockTxns  []*wire.MsgTx
		serialized []byte
		errType    error
	}{
//改编自主区块链中的区块170。
		{
			name: "Force assertion due to missing stxos",
blockTxns: []*wire.MsgTx{{ //省略了coinbase。
				Version: 1,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: wire.OutPoint{
						Hash:  *newHashFromStr("0437cd7f8525ceed2324359c2d0ba26006d92d856a9c20fa0241106ee5a597c9"),
						Index: 0,
					},
					SignatureScript: hexToBytes("47304402204e45e16932b8af514961a1d3a1a25fdf3f4f7732e9d624c6c61548ab5fb8cd410220181522ec8eca07de4860a4acdd12909d831cc56cbbac4622082221a8768d1d0901"),
					Sequence:        0xffffffff,
				}},
				LockTime: 0,
			}},
			serialized: hexToBytes(""),
			errType:    AssertError(""),
		},
		{
			name: "Force deserialization error in stxos",
blockTxns: []*wire.MsgTx{{ //省略了coinbase。
				Version: 1,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: wire.OutPoint{
						Hash:  *newHashFromStr("0437cd7f8525ceed2324359c2d0ba26006d92d856a9c20fa0241106ee5a597c9"),
						Index: 0,
					},
					SignatureScript: hexToBytes("47304402204e45e16932b8af514961a1d3a1a25fdf3f4f7732e9d624c6c61548ab5fb8cd410220181522ec8eca07de4860a4acdd12909d831cc56cbbac4622082221a8768d1d0901"),
					Sequence:        0xffffffff,
				}},
				LockTime: 0,
			}},
			serialized: hexToBytes("1301320511db93e1dcdb8a016b49840f8c53bc1eb68a382e97b1482ecad7b148a6909a"),
			errType:    errDeserialize(""),
		},
	}

	for _, test := range tests {
//确保返回预期的错误类型，并返回
//切片是零。
		stxos, err := deserializeSpendJournalEntry(test.serialized,
			test.blockTxns)
		if reflect.TypeOf(err) != reflect.TypeOf(test.errType) {
			t.Errorf("deserializeSpendJournalEntry (%s): expected "+
				"error type does not match - got %T, want %T",
				test.name, err, test.errType)
			continue
		}
		if stxos != nil {
			t.Errorf("deserializeSpendJournalEntry (%s): returned "+
				"slice of spent transaction outputs is not nil",
				test.name)
			continue
		}
	}
}

//testutxoserialization确保序列化和反序列化未暂停
//传输输出条目按预期工作。
func TestUtxoSerialization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		entry      *UtxoEntry
		serialized []byte
	}{
//来自主区块链中的Tx：
//0e3e2357e806b6db1f70b54c3a17b6714ee1f0e68beb44a74b1efd512098:0
		{
			name: "height 1, coinbase",
			entry: &UtxoEntry{
				amount:      5000000000,
				pkScript:    hexToBytes("410496b538e853519c726a2c91e61ec11600ae1390813a627c66fb8be7947be63c52da7589379515d4e0a604f8141781e62294721166bf621e73a82cbf2342c858eeac"),
				blockHeight: 1,
				packedFlags: tfCoinBase,
			},
			serialized: hexToBytes("03320496b538e853519c726a2c91e61ec11600ae1390813a627c66fb8be7947be63c52"),
		},
//来自主区块链中的Tx：
//0e3e2357e806b6db1f70b54c3a17b6714ee1f0e68beb44a74b1efd512098:0
		{
			name: "height 1, coinbase, spent",
			entry: &UtxoEntry{
				amount:      5000000000,
				pkScript:    hexToBytes("410496b538e853519c726a2c91e61ec11600ae1390813a627c66fb8be7947be63c52da7589379515d4e0a604f8141781e62294721166bf621e73a82cbf2342c858eeac"),
				blockHeight: 1,
				packedFlags: tfCoinBase | tfSpent,
			},
			serialized: nil,
		},
//来自主区块链中的Tx：
//8131ffb0a2c945ecaf9b9063e59558784f9c3a74741ce6ae2a18d0571dac15bb:1
		{
			name: "height 100001, not coinbase",
			entry: &UtxoEntry{
				amount:      1000000,
				pkScript:    hexToBytes("76a914ee8bd501094a7d5ca318da2506de35e1cb025ddc88ac"),
				blockHeight: 100001,
				packedFlags: 0,
			},
			serialized: hexToBytes("8b99420700ee8bd501094a7d5ca318da2506de35e1cb025ddc"),
		},
//来自主区块链中的Tx：
//8131ffb0a2c945ecaf9b9063e59558784f9c3a74741ce6ae2a18d0571dac15bb:1
		{
			name: "height 100001, not coinbase, spent",
			entry: &UtxoEntry{
				amount:      1000000,
				pkScript:    hexToBytes("76a914ee8bd501094a7d5ca318da2506de35e1cb025ddc88ac"),
				blockHeight: 100001,
				packedFlags: tfSpent,
			},
			serialized: nil,
		},
	}

	for i, test := range tests {
//确保utxo项序列化为预期值。
		gotBytes, err := serializeUtxoEntry(test.entry)
		if err != nil {
			t.Errorf("serializeUtxoEntry #%d (%s) unexpected "+
				"error: %v", i, test.name, err)
			continue
		}
		if !bytes.Equal(gotBytes, test.serialized) {
			t.Errorf("serializeUtxoEntry #%d (%s): mismatched "+
				"bytes - got %x, want %x", i, test.name,
				gotBytes, test.serialized)
			continue
		}

//不要尝试反序列化测试项是否已使用，因为它
//将具有nil序列化。
		if test.entry.IsSpent() {
			continue
		}

//反序列化为utxo项。
		utxoEntry, err := deserializeUtxoEntry(test.serialized)
		if err != nil {
			t.Errorf("deserializeUtxoEntry #%d (%s) unexpected "+
				"error: %v", i, test.name, err)
			continue
		}

//反序列化项不能标记为已用，因为它已被取消搁置
//条目未序列化。
		if utxoEntry.IsSpent() {
			t.Errorf("deserializeUtxoEntry #%d (%s) output should "+
				"not be marked spent", i, test.name)
			continue
		}

//确保反序列化项与
//在测试条目中。
		if utxoEntry.Amount() != test.entry.Amount() {
			t.Errorf("deserializeUtxoEntry #%d (%s) mismatched "+
				"amounts: got %d, want %d", i, test.name,
				utxoEntry.Amount(), test.entry.Amount())
			continue
		}

		if !bytes.Equal(utxoEntry.PkScript(), test.entry.PkScript()) {
			t.Errorf("deserializeUtxoEntry #%d (%s) mismatched "+
				"scripts: got %x, want %x", i, test.name,
				utxoEntry.PkScript(), test.entry.PkScript())
			continue
		}
		if utxoEntry.BlockHeight() != test.entry.BlockHeight() {
			t.Errorf("deserializeUtxoEntry #%d (%s) mismatched "+
				"block height: got %d, want %d", i, test.name,
				utxoEntry.BlockHeight(), test.entry.BlockHeight())
			continue
		}
		if utxoEntry.IsCoinBase() != test.entry.IsCoinBase() {
			t.Errorf("deserializeUtxoEntry #%d (%s) mismatched "+
				"coinbase flag: got %v, want %v", i, test.name,
				utxoEntry.IsCoinBase(), test.entry.IsCoinBase())
			continue
		}
	}
}

//testutxEntryHeaderCodeErrors对未使用的执行负测试
//事务输出头代码，以确保错误路径按预期工作。
func TestUtxoEntryHeaderCodeErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		entry   *UtxoEntry
		code    uint64
		errType error
	}{
		{
			name:    "Force assertion due to spent output",
			entry:   &UtxoEntry{packedFlags: tfSpent},
			errType: AssertError(""),
		},
	}

	for _, test := range tests {
//确保返回预期的错误类型，代码为0。
		code, err := utxoEntryHeaderCode(test.entry)
		if reflect.TypeOf(err) != reflect.TypeOf(test.errType) {
			t.Errorf("utxoEntryHeaderCode (%s): expected error "+
				"type does not match - got %T, want %T",
				test.name, err, test.errType)
			continue
		}
		if code != 0 {
			t.Errorf("utxoEntryHeaderCode (%s): unexpected code "+
				"on error - got %d, want 0", test.name, code)
			continue
		}
	}
}

//TestUtxEntrySerializeErrors对反序列化执行负测试
//未暂停的事务输出，以确保错误路径按预期工作。
func TestUtxoEntryDeserializeErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		serialized []byte
		errType    error
	}{
		{
			name:       "no data after header code",
			serialized: hexToBytes("02"),
			errType:    errDeserialize(""),
		},
		{
			name:       "incomplete compressed txout",
			serialized: hexToBytes("0232"),
			errType:    errDeserialize(""),
		},
	}

	for _, test := range tests {
//确保返回预期的错误类型，并返回
//条目是零。
		entry, err := deserializeUtxoEntry(test.serialized)
		if reflect.TypeOf(err) != reflect.TypeOf(test.errType) {
			t.Errorf("deserializeUtxoEntry (%s): expected error "+
				"type does not match - got %T, want %T",
				test.name, err, test.errType)
			continue
		}
		if entry != nil {
			t.Errorf("deserializeUtxoEntry (%s): returned entry "+
				"is not nil", test.name)
			continue
		}
	}
}

//TestBestChainStateSerialization确保序列化和反序列化
//最佳链状态按预期工作。
func TestBestChainStateSerialization(t *testing.T) {
	t.Parallel()

	workSum := new(big.Int)
	tests := []struct {
		name       string
		state      bestChainState
		serialized []byte
	}{
		{
			name: "genesis",
			state: bestChainState{
				hash:      *newHashFromStr("000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f"),
				height:    0,
				totalTxns: 1,
				workSum: func() *big.Int {
					workSum.Add(workSum, CalcWork(486604799))
					return new(big.Int).Set(workSum)
}(), //0x01000 10001
			},
			serialized: hexToBytes("6fe28c0ab6f1b372c1a6a246ae63f74f931e8365e15a089c68d6190000000000000000000100000000000000050000000100010001"),
		},
		{
			name: "block 1",
			state: bestChainState{
				hash:      *newHashFromStr("00000000839a8e6886ab5951d76f411475428afc90947ee320161bbf18eb6048"),
				height:    1,
				totalTxns: 2,
				workSum: func() *big.Int {
					workSum.Add(workSum, CalcWork(486604799))
					return new(big.Int).Set(workSum)
}(), //0x02000
			},
			serialized: hexToBytes("4860eb18bf1b1620e37e9490fc8a427514416fd75159ab86688e9a8300000000010000000200000000000000050000000200020002"),
		},
	}

	for i, test := range tests {
//确保状态序列化为预期值。
		gotBytes := serializeBestChainState(test.state)
		if !bytes.Equal(gotBytes, test.serialized) {
			t.Errorf("serializeBestChainState #%d (%s): mismatched "+
				"bytes - got %x, want %x", i, test.name,
				gotBytes, test.serialized)
			continue
		}

//确保将序列化字节解码回预期的
//状态。
		state, err := deserializeBestChainState(test.serialized)
		if err != nil {
			t.Errorf("deserializeBestChainState #%d (%s) "+
				"unexpected error: %v", i, test.name, err)
			continue
		}
		if !reflect.DeepEqual(state, test.state) {
			t.Errorf("deserializeBestChainState #%d (%s) "+
				"mismatched state - got %v, want %v", i,
				test.name, state, test.state)
			continue

		}
	}
}

//TestBestChainStateDeserializerRors对
//反序列化链状态以确保错误路径按预期工作。
func TestBestChainStateDeserializeErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		serialized []byte
		errType    error
	}{
		{
			name:       "nothing serialized",
			serialized: hexToBytes(""),
			errType:    database.Error{ErrorCode: database.ErrCorruption},
		},
		{
			name:       "short data in hash",
			serialized: hexToBytes("0000"),
			errType:    database.Error{ErrorCode: database.ErrCorruption},
		},
		{
			name:       "short data in work sum",
			serialized: hexToBytes("6fe28c0ab6f1b372c1a6a246ae63f74f931e8365e15a089c68d61900000000000000000001000000000000000500000001000100"),
			errType:    database.Error{ErrorCode: database.ErrCorruption},
		},
	}

	for _, test := range tests {
//确保返回预期的错误类型和代码。
		_, err := deserializeBestChainState(test.serialized)
		if reflect.TypeOf(err) != reflect.TypeOf(test.errType) {
			t.Errorf("deserializeBestChainState (%s): expected "+
				"error type does not match - got %T, want %T",
				test.name, err, test.errType)
			continue
		}
		if derr, ok := err.(database.Error); ok {
			tderr := test.errType.(database.Error)
			if derr.ErrorCode != tderr.ErrorCode {
				t.Errorf("deserializeBestChainState (%s): "+
					"wrong  error code got: %v, want: %v",
					test.name, derr.ErrorCode,
					tderr.ErrorCode)
				continue
			}
		}
	}
}
