
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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

//脚本测试名称返回给定引用脚本的描述性测试名称
//测试数据。
func scriptTestName(test []interface{}) (string, error) {
//说明任何可选的主要见证数据。
	var witnessOffset int
	if _, ok := test[0].([]interface{}); ok {
		witnessOffset++
	}

//除了可选的主要见证数据外，测试还必须
//至少包含签名脚本、公钥脚本、标志，
//以及预期的错误。最后，它可以选择包含注释。
	if len(test) < witnessOffset+4 || len(test) > witnessOffset+5 {
		return "", fmt.Errorf("invalid test length %d", len(test))
	}

//如果指定了测试名称，则使用测试名称的注释，否则，
//根据签名脚本、公钥脚本、
//和旗帜。
	var name string
	if len(test) == witnessOffset+5 {
		name = fmt.Sprintf("test (%s)", test[witnessOffset+4])
	} else {
		name = fmt.Sprintf("test ([%s, %s, %s])", test[witnessOffset],
			test[witnessOffset+1], test[witnessOffset+2])
	}
	return name, nil
}

//将十六进制字符串解析为一个[]字节。
func parseHex(tok string) ([]byte, error) {
	if !strings.HasPrefix(tok, "0x") {
		return nil, errors.New("not a hex number")
	}
	return hex.DecodeString(tok[2:])
}

//ParseWitnessStack将十六进制编码的见证项的JSON数组解析为
//见证元素切片。
func parseWitnessStack(elements []interface{}) ([][]byte, error) {
	witness := make([][]byte, len(elements))
	for i, e := range elements {
		witElement, err := hex.DecodeString(e.(string))
		if err != nil {
			return nil, err
		}

		witness[i] = witElement
	}

	return witness, nil
}

//shortformops保存操作码名称到值的映射，以便在短格式中使用
//解析。它在此处声明，因此只需要创建一次。
var shortFormOps map[string]byte

//parseshortform分析比特币核心参考测试中使用的字符串
//在剧本里。
//
//这些测试使用的格式非常简单，如果是特别的：
//-除推送操作码和未知操作码以外的其他操作码显示为
//要么是名字，要么就是名字
//-普通数字用于推送操作
//-以0x开头的数字按原样插入到[]字节中（所以
//0x14为op_data_20）
//-将单引号字符串作为数据推送
//-其他都是错误
func parseShortForm(script string) ([]byte, error) {
//只创建一次短格式操作码映射。
	if shortFormOps == nil {
		ops := make(map[string]byte)
		for opcodeName, opcodeValue := range OpcodeByName {
			if strings.Contains(opcodeName, "OP_UNKNOWN") {
				continue
			}
			ops[opcodeName] = opcodeValue

//名为op_的操作码不能有op_前缀
//脱光衣服否则会与平原冲突
//数字。另外，因为opu-false和opu-true是
//Op_0和Op_1的别名分别是
//具有相同的值，因此按名称和
//允许他们。
			if (opcodeName == "OP_FALSE" || opcodeName == "OP_TRUE") ||
				(opcodeValue != OP_0 && (opcodeValue < OP_1 ||
					opcodeValue > OP_16)) {

				ops[strings.TrimPrefix(opcodeName, "OP_")] = opcodeValue
			}
		}
		shortFormOps = ops
	}

//split只做一个分隔符，因此将\n和tab全部转换为空格。
	script = strings.Replace(script, "\n", " ", -1)
	script = strings.Replace(script, "\t", " ", -1)
	tokens := strings.Split(script, " ")
	builder := NewScriptBuilder()

	for _, tok := range tokens {
		if len(tok) == 0 {
			continue
		}
//如果解析为普通数字
		if num, err := strconv.ParseInt(tok, 10, 64); err == nil {
			builder.AddInt64(num)
			continue
		} else if bts, err := parseHex(tok); err == nil {
//从测试代码开始手动连接字节
//故意创建过大的脚本
//否则会导致生成器出错。
			if builder.err == nil {
				builder.script = append(builder.script, bts...)
			}
		} else if len(tok) >= 2 &&
			tok[0] == '\'' && tok[len(tok)-1] == '\'' {
			builder.AddFullData([]byte(tok[1 : len(tok)-1]))
		} else if opcode, ok := shortFormOps[tok]; ok {
			builder.AddOp(opcode)
		} else {
			return nil, fmt.Errorf("bad token %q", tok)
		}

	}
	return builder.Script()
}

//ParseScriptFlags根据中使用的格式分析提供的标志字符串
//将测试引用到适合在脚本引擎中使用的脚本标志中。
func parseScriptFlags(flagStr string) (ScriptFlags, error) {
	var flags ScriptFlags

	sFlags := strings.Split(flagStr, ",")
	for _, flag := range sFlags {
		switch flag {
		case "":
//没有什么。
		case "CHECKLOCKTIMEVERIFY":
			flags |= ScriptVerifyCheckLockTimeVerify
		case "CHECKSEQUENCEVERIFY":
			flags |= ScriptVerifyCheckSequenceVerify
		case "CLEANSTACK":
			flags |= ScriptVerifyCleanStack
		case "DERSIG":
			flags |= ScriptVerifyDERSignatures
		case "DISCOURAGE_UPGRADABLE_NOPS":
			flags |= ScriptDiscourageUpgradableNops
		case "LOW_S":
			flags |= ScriptVerifyLowS
		case "MINIMALDATA":
			flags |= ScriptVerifyMinimalData
		case "NONE":
//没有什么。
		case "NULLDUMMY":
			flags |= ScriptStrictMultiSig
		case "NULLFAIL":
			flags |= ScriptVerifyNullFail
		case "P2SH":
			flags |= ScriptBip16
		case "SIGPUSHONLY":
			flags |= ScriptVerifySigPushOnly
		case "STRICTENC":
			flags |= ScriptVerifyStrictEncoding
		case "WITNESS":
			flags |= ScriptVerifyWitness
		case "DISCOURAGE_UPGRADABLE_WITNESS_PROGRAM":
			flags |= ScriptVerifyDiscourageUpgradeableWitnessProgram
		case "MINIMALIF":
			flags |= ScriptVerifyMinimalIf
		case "WITNESS_PUBKEYTYPE":
			flags |= ScriptVerifyWitnessPubKeyType
		default:
			return flags, fmt.Errorf("invalid flag: %s", flag)
		}
	}
	return flags, nil
}

//ParseExpectedResult将提供的预期结果字符串解析为允许的
//编写错误代码脚本。如果预期结果字符串为
//不支持。
func parseExpectedResult(expected string) ([]ErrorCode, error) {
	switch expected {
	case "OK":
		return nil, nil
	case "UNKNOWN_ERROR":
		return []ErrorCode{ErrNumberTooBig, ErrMinimalData}, nil
	case "PUBKEYTYPE":
		return []ErrorCode{ErrPubKeyType}, nil
	case "SIG_DER":
		return []ErrorCode{ErrSigTooShort, ErrSigTooLong,
			ErrSigInvalidSeqID, ErrSigInvalidDataLen, ErrSigMissingSTypeID,
			ErrSigMissingSLen, ErrSigInvalidSLen,
			ErrSigInvalidRIntID, ErrSigZeroRLen, ErrSigNegativeR,
			ErrSigTooMuchRPadding, ErrSigInvalidSIntID,
			ErrSigZeroSLen, ErrSigNegativeS, ErrSigTooMuchSPadding,
			ErrInvalidSigHashType}, nil
	case "EVAL_FALSE":
		return []ErrorCode{ErrEvalFalse, ErrEmptyStack}, nil
	case "EQUALVERIFY":
		return []ErrorCode{ErrEqualVerify}, nil
	case "NULLFAIL":
		return []ErrorCode{ErrNullFail}, nil
	case "SIG_HIGH_S":
		return []ErrorCode{ErrSigHighS}, nil
	case "SIG_HASHTYPE":
		return []ErrorCode{ErrInvalidSigHashType}, nil
	case "SIG_NULLDUMMY":
		return []ErrorCode{ErrSigNullDummy}, nil
	case "SIG_PUSHONLY":
		return []ErrorCode{ErrNotPushOnly}, nil
	case "CLEANSTACK":
		return []ErrorCode{ErrCleanStack}, nil
	case "BAD_OPCODE":
		return []ErrorCode{ErrReservedOpcode, ErrMalformedPush}, nil
	case "UNBALANCED_CONDITIONAL":
		return []ErrorCode{ErrUnbalancedConditional,
			ErrInvalidStackOperation}, nil
	case "OP_RETURN":
		return []ErrorCode{ErrEarlyReturn}, nil
	case "VERIFY":
		return []ErrorCode{ErrVerify}, nil
	case "INVALID_STACK_OPERATION", "INVALID_ALTSTACK_OPERATION":
		return []ErrorCode{ErrInvalidStackOperation}, nil
	case "DISABLED_OPCODE":
		return []ErrorCode{ErrDisabledOpcode}, nil
	case "DISCOURAGE_UPGRADABLE_NOPS":
		return []ErrorCode{ErrDiscourageUpgradableNOPs}, nil
	case "PUSH_SIZE":
		return []ErrorCode{ErrElementTooBig}, nil
	case "OP_COUNT":
		return []ErrorCode{ErrTooManyOperations}, nil
	case "STACK_SIZE":
		return []ErrorCode{ErrStackOverflow}, nil
	case "SCRIPT_SIZE":
		return []ErrorCode{ErrScriptTooBig}, nil
	case "PUBKEY_COUNT":
		return []ErrorCode{ErrInvalidPubKeyCount}, nil
	case "SIG_COUNT":
		return []ErrorCode{ErrInvalidSignatureCount}, nil
	case "MINIMALDATA":
		return []ErrorCode{ErrMinimalData}, nil
	case "NEGATIVE_LOCKTIME":
		return []ErrorCode{ErrNegativeLockTime}, nil
	case "UNSATISFIED_LOCKTIME":
		return []ErrorCode{ErrUnsatisfiedLockTime}, nil
	case "MINIMALIF":
		return []ErrorCode{ErrMinimalIf}, nil
	case "DISCOURAGE_UPGRADABLE_WITNESS_PROGRAM":
		return []ErrorCode{ErrDiscourageUpgradableWitnessProgram}, nil
	case "WITNESS_PROGRAM_WRONG_LENGTH":
		return []ErrorCode{ErrWitnessProgramWrongLength}, nil
	case "WITNESS_PROGRAM_WITNESS_EMPTY":
		return []ErrorCode{ErrWitnessProgramEmpty}, nil
	case "WITNESS_PROGRAM_MISMATCH":
		return []ErrorCode{ErrWitnessProgramMismatch}, nil
	case "WITNESS_MALLEATED":
		return []ErrorCode{ErrWitnessMalleated}, nil
	case "WITNESS_MALLEATED_P2SH":
		return []ErrorCode{ErrWitnessMalleatedP2SH}, nil
	case "WITNESS_UNEXPECTED":
		return []ErrorCode{ErrWitnessUnexpected}, nil
	case "WITNESS_PUBKEYTYPE":
		return []ErrorCode{ErrWitnessPubKeyType}, nil
	}

	return nil, fmt.Errorf("unrecognized expected result in test data: %v",
		expected)
}

//createSpendtx生成一个给定传递的基本支出事务
//签名、见证和公钥脚本。
func createSpendingTx(witness [][]byte, sigScript, pkScript []byte,
	outputValue int64) *wire.MsgTx {

	coinbaseTx := wire.NewMsgTx(wire.TxVersion)

	outPoint := wire.NewOutPoint(&chainhash.Hash{}, ^uint32(0))
	txIn := wire.NewTxIn(outPoint, []byte{OP_0, OP_0}, nil)
	txOut := wire.NewTxOut(outputValue, pkScript)
	coinbaseTx.AddTxIn(txIn)
	coinbaseTx.AddTxOut(txOut)

	spendingTx := wire.NewMsgTx(wire.TxVersion)
	coinbaseTxSha := coinbaseTx.TxHash()
	outPoint = wire.NewOutPoint(&coinbaseTxSha, 0)
	txIn = wire.NewTxIn(outPoint, sigScript, witness)
	txOut = wire.NewTxOut(outputValue, nil)

	spendingTx.AddTxIn(txIn)
	spendingTx.AddTxOut(txOut)

	return spendingTx
}

//scriptWithInputVal使用中的输出值包装目标pkscript
//它是被包含的。输入值是必要的，以便正确
//验证使用嵌套或本机见证程序的输入。
type scriptWithInputVal struct {
	inputVal int64
	pkScript []byte
}

//testscripts确保所有通过的脚本测试都使用预期的
//使用或不使用签名缓存的结果，如
//参数。
func testScripts(t *testing.T, tests [][]interface{}, useSigCache bool) {
//创建签名缓存，仅在请求时使用。
	var sigCache *SigCache
	if useSigCache {
		sigCache = NewSigCache(10)
	}

	for i, test := range tests {
//“格式为：【Wit…，Amount】？，脚本sig，脚本pubkey，
//标志，应为脚本错误…评论”

//跳过单行注释。
		if len(test) == 1 {
			continue
		}

//基于注释和测试构造测试的名称
//数据。
		name, err := scriptTestName(test)
		if err != nil {
			t.Errorf("TestScripts: invalid test #%d: %v", i, err)
			continue
		}

		var (
			witness  wire.TxWitness
			inputAmt btcutil.Amount
		)

//当测试数据的第一个字段是一个切片时，它包含
//因此，见证数据和其他所有数据都被1抵消。
		witnessOffset := 0
		if witnessData, ok := test[0].([]interface{}); ok {
			witnessOffset++

//如果这是一个见证测试，那么最后一个元素
//在切片内是输入量，因此我们忽略
//除最后一个元素以外的所有元素，以便分析
//见证堆栈。
			strWitnesses := witnessData[:len(witnessData)-1]
			witness, err = parseWitnessStack(strWitnesses)
			if err != nil {
				t.Errorf("%s: can't parse witness; %v", name, err)
				continue
			}

			inputAmt, err = btcutil.NewAmount(witnessData[len(witnessData)-1].(float64))
			if err != nil {
				t.Errorf("%s: can't parse input amt: %v",
					name, err)
				continue
			}

		}

//从测试字段中提取并分析签名脚本。
		scriptSigStr, ok := test[witnessOffset].(string)
		if !ok {
			t.Errorf("%s: signature script is not a string", name)
			continue
		}
		scriptSig, err := parseShortForm(scriptSigStr)
		if err != nil {
			t.Errorf("%s: can't parse signature script: %v", name,
				err)
			continue
		}

//从测试字段中提取和分析公钥脚本。
		scriptPubKeyStr, ok := test[witnessOffset+1].(string)
		if !ok {
			t.Errorf("%s: public key script is not a string", name)
			continue
		}
		scriptPubKey, err := parseShortForm(scriptPubKeyStr)
		if err != nil {
			t.Errorf("%s: can't parse public key script: %v", name,
				err)
			continue
		}

//从测试字段中提取和分析脚本标志。
		flagsStr, ok := test[witnessOffset+2].(string)
		if !ok {
			t.Errorf("%s: flags field is not a string", name)
			continue
		}
		flags, err := parseScriptFlags(flagsStr)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}

//从测试字段中提取并分析预期结果。
//
//将期望的结果字符串转换为允许的脚本
//错误代码。这是必要的，因为txscript
//与参考测试数据相比，它的错误是细粒度的，所以
//一些参考测试数据错误映射到多个
//可能性。
		resultStr, ok := test[witnessOffset+3].(string)
		if !ok {
			t.Errorf("%s: result field is not a string", name)
			continue
		}
		allowedErrorCodes, err := parseExpectedResult(resultStr)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}

//生成一个事务对，以便从
//其他和提供的签名和公钥脚本是
//使用，然后创建一个新的引擎来执行脚本。
		tx := createSpendingTx(witness, scriptSig, scriptPubKey,
			int64(inputAmt))
		vm, err := NewEngine(scriptPubKey, tx, 0, flags, sigCache, nil,
			int64(inputAmt))
		if err == nil {
			err = vm.Execute()
		}

//确保预期结果正常时没有错误。
		if resultStr == "OK" {
			if err != nil {
				t.Errorf("%s failed to execute: %v", name, err)
			}
			continue
		}

//此时会出现错误，因此确保
//执行与之匹配。
		success := false
		for _, code := range allowedErrorCodes {
			if IsErrorCode(err, code) {
				success = true
				break
			}
		}
		if !success {
			if serr, ok := err.(Error); ok {
				t.Errorf("%s: want error codes %v, got %v", name,
					allowedErrorCodes, serr.ErrorCode)
				continue
			}
			t.Errorf("%s: want error codes %v, got err: %v (%T)",
				name, allowedErrorCodes, err, err)
			continue
		}
	}
}

//testscripts确保script_tests.json中的所有测试使用
//测试数据中定义的预期结果。
func TestScripts(t *testing.T) {
	file, err := ioutil.ReadFile("data/script_tests.json")
	if err != nil {
		t.Fatalf("TestScripts: %v\n", err)
	}

	var tests [][]interface{}
	err = json.Unmarshal(file, &tests)
	if err != nil {
		t.Fatalf("TestScripts couldn't Unmarshal: %v", err)
	}

//使用和不使用签名缓存运行所有脚本测试。
	testScripts(t, tests, true)
	testScripts(t, tests, false)
}

//testvecf64touint32正确处理从JSON读取的float64s的转换
//将数据测试为无符号32位整数。这是必要的，因为
//测试数据使用-1作为表示最大uint32和直接转换
//负浮点到无符号int依赖于实现，因此
//不会在所有平台上产生预期值。这个功能很管用
//首先转换为32位有符号整数，然后
//然后转换为32位无符号整数，该整数将导致
//所有平台。
func testVecF64ToUint32(f float64) uint32 {
	return uint32(int32(f))
}

//testxtinvalidtests确保tx_invalid.json中的所有测试失败为
//预期。
func TestTxInvalidTests(t *testing.T) {
	file, err := ioutil.ReadFile("data/tx_invalid.json")
	if err != nil {
		t.Fatalf("TestTxInvalidTests: %v\n", err)
	}

	var tests [][]interface{}
	err = json.Unmarshal(file, &tests)
	if err != nil {
		t.Fatalf("TestTxInvalidTests couldn't Unmarshal: %v\n", err)
	}

//形式是：
//[这是评论]
//或：
//[[[previous hash，previous index，previous scriptpubkey]…]
//序列化传输，verifyflags]
testloop:
	for i, test := range tests {
		inputs, ok := test[0].([]interface{})
		if !ok {
			continue
		}

		if len(test) != 3 {
			t.Errorf("bad test (bad length) %d: %v", i, test)
			continue

		}
		serializedhex, ok := test[1].(string)
		if !ok {
			t.Errorf("bad test (arg 2 not string) %d: %v", i, test)
			continue
		}
		serializedTx, err := hex.DecodeString(serializedhex)
		if err != nil {
			t.Errorf("bad test (arg 2 not hex %v) %d: %v", err, i,
				test)
			continue
		}

		tx, err := btcutil.NewTxFromBytes(serializedTx)
		if err != nil {
			t.Errorf("bad test (arg 2 not msgtx %v) %d: %v", err,
				i, test)
			continue
		}

		verifyFlags, ok := test[2].(string)
		if !ok {
			t.Errorf("bad test (arg 3 not string) %d: %v", i, test)
			continue
		}

		flags, err := parseScriptFlags(verifyFlags)
		if err != nil {
			t.Errorf("bad test %d: %v", i, err)
			continue
		}

		prevOuts := make(map[wire.OutPoint]scriptWithInputVal)
		for j, iinput := range inputs {
			input, ok := iinput.([]interface{})
			if !ok {
				t.Errorf("bad test (%dth input not array)"+
					"%d: %v", j, i, test)
				continue testloop
			}

			if len(input) < 3 || len(input) > 4 {
				t.Errorf("bad test (%dth input wrong length)"+
					"%d: %v", j, i, test)
				continue testloop
			}

			previoustx, ok := input[0].(string)
			if !ok {
				t.Errorf("bad test (%dth input hash not string)"+
					"%d: %v", j, i, test)
				continue testloop
			}

			prevhash, err := chainhash.NewHashFromStr(previoustx)
			if err != nil {
				t.Errorf("bad test (%dth input hash not hash %v)"+
					"%d: %v", j, err, i, test)
				continue testloop
			}

			idxf, ok := input[1].(float64)
			if !ok {
				t.Errorf("bad test (%dth input idx not number)"+
					"%d: %v", j, i, test)
				continue testloop
			}
			idx := testVecF64ToUint32(idxf)

			oscript, ok := input[2].(string)
			if !ok {
				t.Errorf("bad test (%dth input script not "+
					"string) %d: %v", j, i, test)
				continue testloop
			}

			script, err := parseShortForm(oscript)
			if err != nil {
				t.Errorf("bad test (%dth input script doesn't "+
					"parse %v) %d: %v", j, err, i, test)
				continue testloop
			}

			var inputValue float64
			if len(input) == 4 {
				inputValue, ok = input[3].(float64)
				if !ok {
					t.Errorf("bad test (%dth input value not int) "+
						"%d: %v", j, i, test)
					continue
				}
			}

			v := scriptWithInputVal{
				inputVal: int64(inputValue),
				pkScript: script,
			}
			prevOuts[*wire.NewOutPoint(prevhash, idx)] = v
		}

		for k, txin := range tx.MsgTx().TxIn {
			prevOut, ok := prevOuts[txin.PreviousOutPoint]
			if !ok {
				t.Errorf("bad test (missing %dth input) %d:%v",
					k, i, test)
				continue testloop
			}
//这些都是注定要失败的，只要第一个
//输入失败事务失败。（一些）
//测试txns也有良好的输入。
			vm, err := NewEngine(prevOut.pkScript, tx.MsgTx(), k,
				flags, nil, nil, prevOut.inputVal)
			if err != nil {
				continue testloop
			}

			err = vm.Execute()
			if err != nil {
				continue testloop
			}

		}
		t.Errorf("test (%d:%v) succeeded when should fail",
			i, test)
	}
}

//testxtvalidtests确保tx-valid.json中的所有测试按预期通过。
func TestTxValidTests(t *testing.T) {
	file, err := ioutil.ReadFile("data/tx_valid.json")
	if err != nil {
		t.Fatalf("TestTxValidTests: %v\n", err)
	}

	var tests [][]interface{}
	err = json.Unmarshal(file, &tests)
	if err != nil {
		t.Fatalf("TestTxValidTests couldn't Unmarshal: %v\n", err)
	}

//形式是：
//[这是评论]
//或：
//[[[previous hash，previous index，previous scriptpubkey，input value]…]
//序列化传输，verifyflags]
testloop:
	for i, test := range tests {
		inputs, ok := test[0].([]interface{})
		if !ok {
			continue
		}

		if len(test) != 3 {
			t.Errorf("bad test (bad length) %d: %v", i, test)
			continue
		}
		serializedhex, ok := test[1].(string)
		if !ok {
			t.Errorf("bad test (arg 2 not string) %d: %v", i, test)
			continue
		}
		serializedTx, err := hex.DecodeString(serializedhex)
		if err != nil {
			t.Errorf("bad test (arg 2 not hex %v) %d: %v", err, i,
				test)
			continue
		}

		tx, err := btcutil.NewTxFromBytes(serializedTx)
		if err != nil {
			t.Errorf("bad test (arg 2 not msgtx %v) %d: %v", err,
				i, test)
			continue
		}

		verifyFlags, ok := test[2].(string)
		if !ok {
			t.Errorf("bad test (arg 3 not string) %d: %v", i, test)
			continue
		}

		flags, err := parseScriptFlags(verifyFlags)
		if err != nil {
			t.Errorf("bad test %d: %v", i, err)
			continue
		}

		prevOuts := make(map[wire.OutPoint]scriptWithInputVal)
		for j, iinput := range inputs {
			input, ok := iinput.([]interface{})
			if !ok {
				t.Errorf("bad test (%dth input not array)"+
					"%d: %v", j, i, test)
				continue
			}

			if len(input) < 3 || len(input) > 4 {
				t.Errorf("bad test (%dth input wrong length)"+
					"%d: %v", j, i, test)
				continue
			}

			previoustx, ok := input[0].(string)
			if !ok {
				t.Errorf("bad test (%dth input hash not string)"+
					"%d: %v", j, i, test)
				continue
			}

			prevhash, err := chainhash.NewHashFromStr(previoustx)
			if err != nil {
				t.Errorf("bad test (%dth input hash not hash %v)"+
					"%d: %v", j, err, i, test)
				continue
			}

			idxf, ok := input[1].(float64)
			if !ok {
				t.Errorf("bad test (%dth input idx not number)"+
					"%d: %v", j, i, test)
				continue
			}
			idx := testVecF64ToUint32(idxf)

			oscript, ok := input[2].(string)
			if !ok {
				t.Errorf("bad test (%dth input script not "+
					"string) %d: %v", j, i, test)
				continue
			}

			script, err := parseShortForm(oscript)
			if err != nil {
				t.Errorf("bad test (%dth input script doesn't "+
					"parse %v) %d: %v", j, err, i, test)
				continue
			}

			var inputValue float64
			if len(input) == 4 {
				inputValue, ok = input[3].(float64)
				if !ok {
					t.Errorf("bad test (%dth input value not int) "+
						"%d: %v", j, i, test)
					continue
				}
			}

			v := scriptWithInputVal{
				inputVal: int64(inputValue),
				pkScript: script,
			}
			prevOuts[*wire.NewOutPoint(prevhash, idx)] = v
		}

		for k, txin := range tx.MsgTx().TxIn {
			prevOut, ok := prevOuts[txin.PreviousOutPoint]
			if !ok {
				t.Errorf("bad test (missing %dth input) %d:%v",
					k, i, test)
				continue testloop
			}
			vm, err := NewEngine(prevOut.pkScript, tx.MsgTx(), k,
				flags, nil, nil, prevOut.inputVal)
			if err != nil {
				t.Errorf("test (%d:%v:%d) failed to create "+
					"script: %v", i, test, k, err)
				continue
			}

			err = vm.Execute()
			if err != nil {
				t.Errorf("test (%d:%v:%d) failed to execute: "+
					"%v", i, test, k, err)
				continue
			}
		}
	}
}

//testcalSignatureHash运行比特币核心签名哈希计算测试
//在sighash.json中。
//https://github.com/bitcoin/bitcoin/blob/master/src/test/data/sighash.json
func TestCalcSignatureHash(t *testing.T) {
	file, err := ioutil.ReadFile("data/sighash.json")
	if err != nil {
		t.Fatalf("TestCalcSignatureHash: %v\n", err)
	}

	var tests [][]interface{}
	err = json.Unmarshal(file, &tests)
	if err != nil {
		t.Fatalf("TestCalcSignatureHash couldn't Unmarshal: %v\n",
			err)
	}

	for i, test := range tests {
		if i == 0 {
//跳过第一行——只包含注释。
			continue
		}
		if len(test) != 5 {
			t.Fatalf("TestCalcSignatureHash: Test #%d has "+
				"wrong length.", i)
		}
		var tx wire.MsgTx
		rawTx, _ := hex.DecodeString(test[0].(string))
		err := tx.Deserialize(bytes.NewReader(rawTx))
		if err != nil {
			t.Errorf("TestCalcSignatureHash failed test #%d: "+
				"Failed to parse transaction: %v", i, err)
			continue
		}

		subScript, _ := hex.DecodeString(test[1].(string))
		parsedScript, err := parseScript(subScript)
		if err != nil {
			t.Errorf("TestCalcSignatureHash failed test #%d: "+
				"Failed to parse sub-script: %v", i, err)
			continue
		}

		hashType := SigHashType(testVecF64ToUint32(test[3].(float64)))
		hash := calcSignatureHash(parsedScript, hashType, &tx,
			int(test[2].(float64)))

		expectedHash, _ := chainhash.NewHashFromStr(test[4].(string))
		if !bytes.Equal(hash, expectedHash[:]) {
			t.Errorf("TestCalcSignatureHash failed test #%d: "+
				"Signature hash mismatch.", i)
		}
	}
}
