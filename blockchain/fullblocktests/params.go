
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package fullblocktests

import (
	"encoding/hex"
	"math/big"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

//newhashfromstr将传递的big endian十六进制字符串转换为
//
//
//硬编码，因此已知良好，哈希。
func newHashFromStr(hexStr string) *chainhash.Hash {
	hash, err := chainhash.NewHashFromStr(hexStr)
	if err != nil {
		panic(err)
	}
	return hash
}

//
//
//
//
func fromHex(s string) []byte {
	r, err := hex.DecodeString(s)
	if err != nil {
		panic("invalid hex in source file: " + s)
	}
	return r
}

var (
//bigone是1，表示为big.int。这里定义它是为了避免
//多次创建它的开销。
	bigOne = big.NewInt(1)

//
//
	regressionPowLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 255), bigOne)

//regtestgenesisblock定义用于
//
	regTestGenesisBlock = wire.MsgBlock{
		Header: wire.BlockHeader{
			Version:    1,
			PrevBlock:  *newHashFromStr("0000000000000000000000000000000000000000000000000000000000000000"),
			MerkleRoot: *newHashFromStr("4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b"),
Timestamp:  time.Unix(1296688602, 0), //2011年2月2日23:16:42+0000 UTC
Bits:       0x207fffff,               //545259519[7fffff00000000000000000000000000000000000000000000000000000000000000000000]
			Nonce:      2,
		},
		Transactions: []*wire.MsgTx{{
			Version: 1,
			TxIn: []*wire.TxIn{{
				PreviousOutPoint: wire.OutPoint{
					Hash:  chainhash.Hash{},
					Index: 0xffffffff,
				},
				SignatureScript: fromHex("04ffff001d010445" +
					"5468652054696d65732030332f4a616e2f" +
					"32303039204368616e63656c6c6f72206f" +
					"6e206272696e6b206f66207365636f6e64" +
					"206261696c6f757420666f72206261686b73"),
				Sequence: 0xffffffff,
			}},
			TxOut: []*wire.TxOut{{
				Value: 0,
				PkScript: fromHex("4104678afdb0fe5548271967f1" +
					"a67130b7105cd6a828e03909a67962e0ea1f" +
					"61deb649f6bc3f4cef38c4f35504e51ec138" +
					"c4f35504e51ec112de5c384df7ba0b8d578a" +
					"4c702b6bf11d5fac"),
			}},
			LockTime: 0,
		}},
	}
)

//
//网络。
//
//
//在chaincfg包中，因为目的是能够生成已知的
//
//允许他们从可能使他们无效的测试中改变出来。
var regressionNetParams = &chaincfg.Params{
	Name:        "regtest",
	Net:         wire.TestNet,
	DefaultPort: "18444",

//
	GenesisBlock:             &regTestGenesisBlock,
	GenesisHash:              newHashFromStr("5bec7567af40504e0994db3b573c186fffcc4edefe096ff2e58d00523bd7e8a6"),
	PowLimit:                 regressionPowLimit,
	PowLimitBits:             0x207fffff,
	CoinbaseMaturity:         100,
BIP0034Height:            100000000, //
BIP0065Height:            1351,      //回归测试使用
BIP0066Height:            1251,      //回归测试使用
	SubsidyReductionInterval: 150,
TargetTimespan:           time.Hour * 24 * 14, //14天
TargetTimePerBlock:       time.Minute * 10,    //
RetargetAdjustmentFactor: 4,                   //少25%，多400%
	ReduceMinDifficulty:      true,
MinDiffReductionTime:     time.Minute * 20, //
	GenerateSupported:        true,

//
	Checkpoints: nil,

//
	RelayNonStdTxs: true,

//地址编码魔术师
PubKeyHashAddrID: 0x6f, //
ScriptHashAddrID: 0xc4, //
PrivateKeyID:     0xef, //

//
HDPrivateKeyID: [4]byte{0x04, 0x35, 0x83, 0x94}, //
HDPublicKeyID:  [4]byte{0x04, 0x35, 0x87, 0xcf}, //

//
//
	HDCoinType: 1,
}
