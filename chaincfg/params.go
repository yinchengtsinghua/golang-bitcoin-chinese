
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

package chaincfg

import (
	"errors"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

//这些变量是每个默认值的工作限制参数的链式证明。
//网络。
var (
//bigone是1，表示为big.int。这里定义它是为了避免
//多次创建它的开销。
	bigOne = big.NewInt(1)

//mainpowlimit是比特币区块可以提供的最高工作价值证明。
//主网络有。它是2^224-1的值。
	mainPowLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 224), bigOne)

//回归功率限制是比特币区块工作价值的最高证明。
//对于回归测试网络可以有。它是2^255-1的值。
	regressionPowLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 255), bigOne)

//testNet3Powlimit是比特币区块工作价值的最高证明。
//可用于测试网络（版本3）。这就是价值
//2 ^ 224—1。
	testNet3PowLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 224), bigOne)

//simnetpowlimit是比特币区块工作价值的最高证明。
//can have for the simulation test network.  It is the value 2^255 - 1.
	simNetPowLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 255), bigOne)
)

//检查点标识块链中的已知良好点。使用
//检查点允许在初始下载期间对旧块进行一些优化
//也可以防止旧块的叉。
//
//每个检查点都是根据几个因素选择的。见
//区块链文档.ischeckpointcandidate有关
//选择标准。
type Checkpoint struct {
	Height int32
	Hash   *chainhash.Hash
}

//dnssed标识一个dns种子。
type DNSSeed struct {
//主机定义种子的主机名。
	Host string

//hasfiltering定义种子是否支持筛选
//按服务标志（Wire.ServiceFlag）。
	HasFiltering bool
}

//同意部署定义与特定共识规则相关的详细信息
//投票通过的改变。这是BIP0009的一部分。
type ConsensusDeployment struct {
//位号定义块版本中的特定位号
//这个特定的软分叉部署是指。
	BitNumber uint8

//StartTime是对
//部署开始。
	StartTime uint64

//ExpireTime是尝试
//部署过期。
	ExpireTime uint64
}

//在的Deployments字段中定义部署偏移量的常量
//每个部署的参数。这对了解细节很有用
//按名称指定的部署。
const (
//deploymenttesdummy定义用于测试的规则更改部署ID
//目的。
	DeploymentTestDummy = iota

//deployment csv定义csv的规则更改部署ID
//soft-fork package. The CSV package includes the deployment of BIPS
//68、112和113。
	DeploymentCSV

//deploymentsgewit定义的规则更改部署ID
//隔离见证（Segwit）软叉包。Segwit软件包
//包括部署BIPS 141、142、144、145、147和173。
	DeploymentSegwit

//注意：由于定义的部署用于
//确定当前有多少已定义的部署。

//DefinedDeployments是当前定义的部署数。
	DefinedDeployments
)

//Params defines a Bitcoin network by its parameters.  These parameters may be
//比特币应用程序用于区分网络和地址
//以及用于一个网络的密钥。
type Params struct {
//名称为网络定义一个人类可读的标识符。
	Name string

//NET定义用于标识网络的幻数字节。
	Net wire.BitcoinNet

//默认端口定义网络的默认对等端口。
	DefaultPort string

//dnsseed定义所用网络的dns种子列表
//作为发现同龄人的一种方法。
	DNSSeeds []DNSSeed

//genesBlock定义链的第一个块。
	GenesisBlock *wire.MsgBlock

//genesishash是起始块哈希。
	GenesisHash *chainhash.Hash

//PowLimit defines the highest allowed proof of work value for a block
//作为UTIN 256。
	PowLimit *big.Int

//PowLimitBits定义了
//以紧凑形式阻塞。
	PowLimitBits uint32

//这些字段定义指定SoftFork的块高度
//BIP变得活跃起来。
	BIP0034Height int32
	BIP0065Height int32
	BIP0066Height int32

//CoinBaseMaturity是新开采前所需的块数。
//coins (coinbase transactions) can be spent.
	CoinbaseMaturity uint16

//subsidyReducationInterval是补贴前的区块间隔
//减少。
	SubsidyReductionInterval int32

//TargetTimespan is the desired amount of time that should elapse
//在检查块难度要求以确定如何
//为了保持所需的程序块，应更改该程序块。
//发电量。
	TargetTimespan time.Duration

//TargetTimePerBlock是生成每个
//块。
	TargetTimePerBlock time.Duration

//RetargetAdjustmentFactor is the adjustment factor used to limit
//介于
//困难重定目标。
	RetargetAdjustmentFactor int64

//ReleMeNo难度定义网络是否应该减少
//经过足够长的时间后所需的最小难度
//没有找到一个街区就通过了。这只对测试有用
//不能在主网络上设置。
	ReduceMinDifficulty bool

//MindiffReductionTime是在最短时间之后
//当没有发现障碍物时，应降低要求的难度。
//
//注意：这仅适用于ReduceMindfficulty为真的情况。
	MinDiffReductionTime time.Duration

//generatesupported指定是否允许CPU挖掘。
	GenerateSupported bool

//从最旧到最新排序的检查点。
	Checkpoints []Checkpoint

//这些字段与对共识规则更改的投票有关，如
//由Bip0009定义。
//
//RuleChangeActivationThreshold是阈值中的块数。
//对规则更改进行正面投票的状态重定目标窗口
//必须强制转换才能锁定规则更改。它应该是典型的
//主网络为95%，测试网络为75%。
//
//MinerConfirmationWindow是每个阈值中的块数
//状态重定目标窗口。
//
//部署定义要投票的特定共识规则更改
//在。
	RuleChangeActivationThreshold uint32
	MinerConfirmationWindow       uint32
	Deployments                   [DefinedDeployments]ConsensusDeployment

//内存池参数
	RelayNonStdTxs bool

//BECH32编码Segwit地址的人可读部分，如定义
//在BIP 173。
	Bech32HRPSegwit string

//地址编码魔术师
PubKeyHashAddrID        byte //p2pkh地址的第一个字节
ScriptHashAddrID        byte //p2sh地址的第一个字节
PrivateKeyID            byte //WIF私钥的第一个字节
WitnessPubKeyHashAddrID byte //p2wpkh地址的第一个字节
WitnessScriptHashAddrID byte //First byte of a P2WSH address

//BIP32层次确定性扩展密钥魔术师
	HDPrivateKeyID [4]byte
	HDPublicKeyID  [4]byte

//在等级确定路径中使用的bip44硬币类型
//address generation.
	HDCoinType uint32
}

//mainnetparams定义主比特币网络的网络参数。
var MainNetParams = Params{
	Name:        "mainnet",
	Net:         wire.MainNet,
	DefaultPort: "8333",
	DNSSeeds: []DNSSeed{
		{"seed.bitcoin.sipa.be", true},
		{"dnsseed.bluematt.me", true},
		{"dnsseed.bitcoin.dashjr.org", false},
		{"seed.bitcoinstats.com", true},
		{"seed.bitnodes.io", false},
		{"seed.bitcoin.jonasschnelli.ch", true},
	},

//链参数
	GenesisBlock:             &genesisBlock,
	GenesisHash:              &genesisHash,
	PowLimit:                 mainPowLimit,
	PowLimitBits:             0x1d00ffff,
BIP0034Height:            227931, //000000000000 24B89B42A942FE0D9FEA3BB44AB7BD1B19115DD6A759C0808B8
BIP0065Height:            388381, //0000000000000004c2b624ed5d7756c508d90fd0da2c7c679febfa6c4735f0
BIP0066Height:            363725, //00000000000000000379eaa19dce8c9b722d46ae6a57c2f1a988119488b50931
	CoinbaseMaturity:         100,
	SubsidyReductionInterval: 210000,
TargetTimespan:           time.Hour * 24 * 14, //14天
TargetTimePerBlock:       time.Minute * 10,    //10分钟
RetargetAdjustmentFactor: 4,                   //少25%，多400%
	ReduceMinDifficulty:      false,
	MinDiffReductionTime:     0,
	GenerateSupported:        false,

//从最旧到最新排序的检查点。
	Checkpoints: []Checkpoint{
		{11111, newHashFromStr("0000000069e244f73d78e8fd29ba2fd2ed618bd6fa2ee92559f542fdb26e7c1d")},
		{33333, newHashFromStr("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")},
		{74000, newHashFromStr("0000000000573993a3c9e41ce34471c079dcf5f52a0e824a81e7f953b8661a20")},
		{105000, newHashFromStr("00000000000291ce28027faea320c8d2b054b2e0fe44a773f3eefb151d6bdc97")},
		{134444, newHashFromStr("00000000000005b12ffd4cd315cd34ffd4a594f430ac814c91184a0d42d2b0fe")},
		{168000, newHashFromStr("000000000000099e61ea72015e79632f216fe6cb33d7899acb35b75c8303b763")},
		{193000, newHashFromStr("000000000000059f452a5f7340de6682a977387c17010ff6e6c3bd83ca8b1317")},
		{210000, newHashFromStr("000000000000048b95347e83192f69cf0366076336c639f9b7228e9ba171342e")},
		{216116, newHashFromStr("00000000000001b4f4b433e81ee46494af945cf96014816a4e2370f11b23df4e")},
		{225430, newHashFromStr("00000000000001c108384350f74090433e7fcf79a606b8e797f065b130575932")},
		{250000, newHashFromStr("000000000000003887df1f29024b06fc2200b55f8af8f35453d7be294df2d214")},
		{267300, newHashFromStr("000000000000000a83fbd660e918f218bf37edd92b748ad940483c7c116179ac")},
		{279000, newHashFromStr("0000000000000001ae8c72a0b0c301f67e3afca10e819efa9041e458e9bd7e40")},
		{300255, newHashFromStr("0000000000000000162804527c6e9b9f0563a280525f9d08c12041def0a0f3b2")},
		{319400, newHashFromStr("000000000000000021c6052e9becade189495d1c539aa37c58917305fd15f13b")},
		{343185, newHashFromStr("0000000000000000072b8bf361d01a6ba7d445dd024203fafc78768ed4368554")},
		{352940, newHashFromStr("000000000000000010755df42dba556bb72be6a32f3ce0b6941ce4430152c9ff")},
		{382320, newHashFromStr("00000000000000000a8dc6ed5b133d0eb2fd6af56203e4159789b092defd8ab2")},
	},

//共识规则变更部署。
//
//矿工确认窗口定义为：
//工作时间目标的证明/工作间距的目标证明
RuleChangeActivationThreshold: 1916, //95%的矿工确认窗口
MinerConfirmationWindow:       2016, //
	Deployments: [DefinedDeployments]ConsensusDeployment{
		DeploymentTestDummy: {
			BitNumber:  28,
StartTime:  1199145601, //2008年1月1日UTC
ExpireTime: 1230767999, //2008年12月31日UTC
		},
		DeploymentCSV: {
			BitNumber:  0,
StartTime:  1462060800, //2016年5月1日
ExpireTime: 1493596800, //2017年5月1日
		},
		DeploymentSegwit: {
			BitNumber:  1,
StartTime:  1479168000, //2016年11月15日UTC
ExpireTime: 1510704000, //2017年11月15日UTC。
		},
	},

//内存池参数
	RelayNonStdTxs: false,

//BECH32编码Segwit地址的人可读部分，如中所定义
//BIP 173。
Bech32HRPSegwit: "bc", //主网始终为BC

//地址编码魔术师
PubKeyHashAddrID:        0x00, //从1开始
ScriptHashAddrID:        0x05, //从3开始
PrivateKeyID:            0x80, //从5（未压缩）或k（压缩）开始
WitnessPubKeyHashAddrID: 0x06, //从P2开始
WitnessScriptHashAddrID: 0x0A, //从7XH开始

//BIP32层次确定性扩展密钥魔术师
HDPrivateKeyID: [4]byte{0x04, 0x88, 0xad, 0xe4}, //从xprv开始
HDPublicKeyID:  [4]byte{0x04, 0x88, 0xb2, 0x1e}, //从xpub开始

//在等级确定路径中使用的bip44硬币类型
//地址生成。
	HDCoinType: 0,
}

//regressionnetparams定义回归测试的网络参数
//比特币网络。不要与测试比特币网络混淆（版本
//3）这个网络有时简单地称为“testnet”。
var RegressionNetParams = Params{
	Name:        "regtest",
	Net:         wire.TestNet,
	DefaultPort: "18444",
	DNSSeeds:    []DNSSeed{},

//链参数
	GenesisBlock:             &regTestGenesisBlock,
	GenesisHash:              &regTestGenesisHash,
	PowLimit:                 regressionPowLimit,
	PowLimitBits:             0x207fffff,
	CoinbaseMaturity:         100,
BIP0034Height:            100000000, //Not active - Permit ver 1 blocks
BIP0065Height:            1351,      //回归测试使用
BIP0066Height:            1251,      //回归测试使用
	SubsidyReductionInterval: 150,
TargetTimespan:           time.Hour * 24 * 14, //14天
TargetTimePerBlock:       time.Minute * 10,    //10分钟
RetargetAdjustmentFactor: 4,                   //少25%，多400%
	ReduceMinDifficulty:      true,
MinDiffReductionTime:     time.Minute * 20, //TargetTimePerBlock * 2
	GenerateSupported:        true,

//从最旧到最新排序的检查点。
	Checkpoints: nil,

//共识规则变更部署。
//
//矿工确认窗口定义为：
//工作时间目标的证明/工作间距的目标证明
RuleChangeActivationThreshold: 108, //75%的矿工确认窗口
	MinerConfirmationWindow:       144,
	Deployments: [DefinedDeployments]ConsensusDeployment{
		DeploymentTestDummy: {
			BitNumber:  28,
StartTime:  0,             //始终可供投票
ExpireTime: math.MaxInt64, //永不过期
		},
		DeploymentCSV: {
			BitNumber:  0,
StartTime:  0,             //始终可供投票
ExpireTime: math.MaxInt64, //永不过期
		},
		DeploymentSegwit: {
			BitNumber:  1,
StartTime:  0,             //始终可供投票
ExpireTime: math.MaxInt64, //永不过期。
		},
	},

//内存池参数
	RelayNonStdTxs: true,

//BECH32编码Segwit地址的人可读部分，如中所定义
//BIP 173。
Bech32HRPSegwit: "bcrt", //总是针对Reg测试网络进行BCRT

//地址编码魔术师
PubKeyHashAddrID: 0x6f, //以m或n开头
ScriptHashAddrID: 0xc4, //从2开始
PrivateKeyID:     0xef, //从9（未压缩）或C（压缩）开始

//BIP32层次确定性扩展密钥魔术师
HDPrivateKeyID: [4]byte{0x04, 0x35, 0x83, 0x94}, //从TPRV开始
HDPublicKeyID:  [4]byte{0x04, 0x35, 0x87, 0xcf}, //从tpub开始

//在等级确定路径中使用的bip44硬币类型
//地址生成。
	HDCoinType: 1,
}

//testnet3参数定义测试比特币网络的网络参数
//（版本3）。不要与回归测试网络混淆，这是
//网络有时简单地称为“测试网”。
var TestNet3Params = Params{
	Name:        "testnet3",
	Net:         wire.TestNet3,
	DefaultPort: "18333",
	DNSSeeds: []DNSSeed{
		{"testnet-seed.bitcoin.jonasschnelli.ch", true},
		{"testnet-seed.bitcoin.schildbach.de", false},
		{"seed.tbtc.petertodd.org", true},
		{"testnet-seed.bluematt.me", false},
	},

//链参数
	GenesisBlock:             &testNet3GenesisBlock,
	GenesisHash:              &testNet3GenesisHash,
	PowLimit:                 testNet3PowLimit,
	PowLimitBits:             0x1d00ffff,
BIP0034Height:            21111,  //00000000 23B3A96D3484E5ABB3755C413E7D41500F8E2A5C3F0DD01299CD8EF8
BIP0065Height:            581885, //0000000000 7F6655F22F98E72ED80D8B06DC761D5DA09DF0FA1DC4BE4F861EB6
BIP0066Height:            330776, //00000000 2104C8C45E99A8853285A3B592602A3CCDE2B832481DA85E9E4BA182
	CoinbaseMaturity:         100,
	SubsidyReductionInterval: 210000,
TargetTimespan:           time.Hour * 24 * 14, //14天
TargetTimePerBlock:       time.Minute * 10,    //10分钟
RetargetAdjustmentFactor: 4,                   //少25%，多400%
	ReduceMinDifficulty:      true,
MinDiffReductionTime:     time.Minute * 20, //目标时间块*2
	GenerateSupported:        false,

//从最旧到最新排序的检查点。
	Checkpoints: []Checkpoint{
		{546, newHashFromStr("000000002a936ca763904c3c35fce2f3556c559c0214345d31b1bcebf76acb70")},
		{100000, newHashFromStr("00000000009e2958c15ff9290d571bf9459e93b19765c6801ddeccadbb160a1e")},
		{200000, newHashFromStr("0000000000287bffd321963ef05feab753ebe274e1d78b2fd4e2bfe9ad3aa6f2")},
		{300001, newHashFromStr("0000000000004829474748f3d1bc8fcf893c88be255e6d7f571c548aff57abf4")},
		{400002, newHashFromStr("0000000005e2c73b8ecb82ae2dbc2e8274614ebad7172b53528aba7501f5a089")},
		{500011, newHashFromStr("00000000000929f63977fbac92ff570a9bd9e7715401ee96f2848f7b07750b02")},
		{600002, newHashFromStr("000000000001f471389afd6ee94dcace5ccc44adc18e8bff402443f034b07240")},
		{700000, newHashFromStr("000000000000406178b12a4dea3b27e13b3c4fe4510994fd667d7c1e6a3f4dc1")},
		{800010, newHashFromStr("000000000017ed35296433190b6829db01e657d80631d43f5983fa403bfdb4c1")},
		{900000, newHashFromStr("0000000000356f8d8924556e765b7a94aaebc6b5c8685dcfa2b1ee8b41acd89b")},
		{1000007, newHashFromStr("00000000001ccb893d8a1f25b70ad173ce955e5f50124261bbbc50379a612ddf")},
	},

//共识规则变更部署。
//
//矿工确认窗口定义为：
//工作时间目标的证明/工作间距的目标证明
RuleChangeActivationThreshold: 1512, //75%矿化窗
	MinerConfirmationWindow:       2016,
	Deployments: [DefinedDeployments]ConsensusDeployment{
		DeploymentTestDummy: {
			BitNumber:  28,
StartTime:  1199145601, //2008年1月1日UTC
ExpireTime: 1230767999, //2008年12月31日UTC
		},
		DeploymentCSV: {
			BitNumber:  0,
StartTime:  1456790400, //2016年3月1日
ExpireTime: 1493596800, //2017年5月1日
		},
		DeploymentSegwit: {
			BitNumber:  1,
StartTime:  1462060800, //2016年5月1日UTC
ExpireTime: 1493596800, //2017年5月1日UTC。
		},
	},

//内存池参数
	RelayNonStdTxs: true,

//BECH32编码Segwit地址的人可读部分，如中所定义
//BIP 173。
Bech32HRPSegwit: "tb", //测试网络始终为TB

//地址编码魔术师
PubKeyHashAddrID:        0x6f, //以m或n开头
ScriptHashAddrID:        0xc4, //从2开始
WitnessPubKeyHashAddrID: 0x03, //从QW开始
WitnessScriptHashAddrID: 0x28, //从T7N开始
PrivateKeyID:            0xef, //从9（未压缩）或C（压缩）开始

//BIP32层次确定性扩展密钥魔术师
HDPrivateKeyID: [4]byte{0x04, 0x35, 0x83, 0x94}, //从TPRV开始
HDPublicKeyID:  [4]byte{0x04, 0x35, 0x87, 0xcf}, //从tpub开始

//在等级确定路径中使用的bip44硬币类型
//地址生成。
	HDCoinType: 1,
}

//SimnetParams定义模拟测试比特币的网络参数
//网络。此网络与正常测试网络相似，只是
//在一组进行模拟的个人中供私人使用
//测试。功能的不同之处在于
//专门指定用于创建网络而不是
//遵循正常的发现规则。这很重要，否则它会
//变成另一个公共测试网。
var SimNetParams = Params{
	Name:        "simnet",
	Net:         wire.SimNet,
	DefaultPort: "18555",
DNSSeeds:    []DNSSeed{}, //NOTE: There must NOT be any seeds.

//链参数
	GenesisBlock:             &simNetGenesisBlock,
	GenesisHash:              &simNetGenesisHash,
	PowLimit:                 simNetPowLimit,
	PowLimitBits:             0x207fffff,
BIP0034Height:            0, //在Simnet上始终处于活动状态
BIP0065Height:            0, //在Simnet上始终处于活动状态
BIP0066Height:            0, //在Simnet上始终处于活动状态
	CoinbaseMaturity:         100,
	SubsidyReductionInterval: 210000,
TargetTimespan:           time.Hour * 24 * 14, //14天
TargetTimePerBlock:       time.Minute * 10,    //10分钟
RetargetAdjustmentFactor: 4,                   //少25%，多400%
	ReduceMinDifficulty:      true,
MinDiffReductionTime:     time.Minute * 20, //目标时间块*2
	GenerateSupported:        true,

//从最旧到最新排序的检查点。
	Checkpoints: nil,

//共识规则变更部署。
//
//矿工确认窗口定义为：
//工作时间目标的证明/工作间距的目标证明
RuleChangeActivationThreshold: 75, //75%矿化窗
	MinerConfirmationWindow:       100,
	Deployments: [DefinedDeployments]ConsensusDeployment{
		DeploymentTestDummy: {
			BitNumber:  28,
StartTime:  0,             //始终可供投票
ExpireTime: math.MaxInt64, //永不过期
		},
		DeploymentCSV: {
			BitNumber:  0,
StartTime:  0,             //始终可供投票
ExpireTime: math.MaxInt64, //永不过期
		},
		DeploymentSegwit: {
			BitNumber:  1,
StartTime:  0,             //始终可供投票
ExpireTime: math.MaxInt64, //永不过期。
		},
	},

//内存池参数
	RelayNonStdTxs: true,

//BECH32编码Segwit地址的人可读部分，如中所定义
//BIP 173。
Bech32HRPSegwit: "sb", //总是某人为SIM网络

//地址编码魔术师
PubKeyHashAddrID:        0x3f, //从S开始
ScriptHashAddrID:        0x7b, //从S开始
PrivateKeyID:            0x64, //从4（未压缩）或f（压缩）开始
WitnessPubKeyHashAddrID: 0x19, //从Gg开始
WitnessScriptHashAddrID: 0x28, //从什么开始？

//BIP32层次确定性扩展密钥魔术师
HDPrivateKeyID: [4]byte{0x04, 0x20, 0xb9, 0x00}, //从SPRV开始
HDPublicKeyID:  [4]byte{0x04, 0x20, 0xbd, 0x3a}, //从土豆开始

//在等级确定路径中使用的bip44硬币类型
//地址生成。
HDCoinType: 115, //ASCII为S
}

var (
//errDuplicateNet描述一个错误，其中比特币的参数
//由于网络已经是标准网络，无法设置网络
//网络或以前注册到此包中的。
	ErrDuplicateNet = errors.New("duplicate Bitcoin network")

//errUnknownHdKeyID描述了一个错误，其中提供的ID
//旨在确定网络的层次确定性
//private extended key is not registered.
	ErrUnknownHDKeyID = errors.New("unknown hd private extended key bytes")
)

var (
	registeredNets       = make(map[wire.BitcoinNet]struct{})
	pubKeyHashAddrIDs    = make(map[byte]struct{})
	scriptHashAddrIDs    = make(map[byte]struct{})
	bech32SegwitPrefixes = make(map[string]struct{})
	hdPrivToPubKeyIDs    = make(map[[4]byte][]byte)
)

//String以人类可读的形式返回DNS种子的主机名。
func (d DNSSeed) String() string {
	return d.Host
}

//寄存器为比特币网络注册网络参数。这可能
//error with ErrDuplicateNet if the network is already registered (either
//由于以前的寄存器调用，或者网络是默认的
//网络）。
//
//网络参数应通过主包注册到此包中
//越早越好。然后，库包可以查找网络或网络
//基于输入和工作的参数，无论网络是标准的
//或者没有。
func Register(params *Params) error {
	if _, ok := registeredNets[params.Net]; ok {
		return ErrDuplicateNet
	}
	registeredNets[params.Net] = struct{}{}
	pubKeyHashAddrIDs[params.PubKeyHashAddrID] = struct{}{}
	scriptHashAddrIDs[params.ScriptHashAddrID] = struct{}{}
	hdPrivToPubKeyIDs[params.HDPrivateKeyID] = params.HDPublicKeyID[:]

//有效的BECH32编码segwit地址的前缀始终为
//给定网络的可读部分，后跟“1”。
	bech32SegwitPrefixes[params.Bech32HRPSegwit+"1"] = struct{}{}
	return nil
}

//mustregister与register执行相同的函数，但如果存在，则会出现恐慌
//是一个错误。这只能从package init函数调用。
func mustRegister(params *Params) {
	if err := Register(params); err != nil {
		panic("failed to register network: " + err.Error())
	}
}

//ispubKeyHashAddRid返回ID是否是已知的前缀a的标识符
//在任何默认或注册的网络上支付到PubKey哈希地址。这是
//将地址字符串解码为特定地址类型时使用。它上升了
//调用方检查this和IsscriptHashAddRid并决定
//地址是pubkey散列地址，脚本散列地址，两者都不是，或者
//无法确定（如果两者都返回真）。
func IsPubKeyHashAddrID(id byte) bool {
	_, ok := pubKeyHashAddrIDs[id]
	return ok
}

//IsScriptHashAddRid返回ID是否是已知的前缀为a的标识符
//在任何默认或注册的网络上，按脚本付费散列地址。这是
//将地址字符串解码为特定地址类型时使用。它上升了
//调用方检查this和ispubKeyHashAddRid，并决定
//地址是pubkey散列地址，脚本散列地址，两者都不是，或者
//无法确定（如果两者都返回真）。
func IsScriptHashAddrID(id byte) bool {
	_, ok := scriptHashAddrIDs[id]
	return ok
}

//isbech32segwitfix返回前缀是否为segwit的已知前缀
//任何默认或注册网络上的地址。解码时使用
//特定地址类型的地址字符串。
func IsBech32SegwitPrefix(prefix string) bool {
	prefix = strings.ToLower(prefix)
	_, ok := bech32SegwitPrefixes[prefix]
	return ok
}

//hdprivatekeytopublickeyid接受私有层次确定性
//extended key id and returns the associated public key id.  When the provided
//ID未注册，将返回ErrUnknownHdKeyID错误。
func HDPrivateKeyToPublicKeyID(id []byte) ([]byte, error) {
	if len(id) != 4 {
		return nil, ErrUnknownHDKeyID
	}

	var key [4]byte
	copy(key[:], id)
	pubBytes, ok := hdPrivToPubKeyIDs[key]
	if !ok {
		return nil, ErrUnknownHDKeyID
	}

	return pubBytes, nil
}

//newhashfromstr将传递的big endian十六进制字符串转换为
//chainhash.hash。它只与chainhash中可用的不同之处在于
//it panics on an error since it will only (and must only) be called with
//硬编码，因此已知良好，哈希。
func newHashFromStr(hexStr string) *chainhash.Hash {
	hash, err := chainhash.NewHashFromStr(hexStr)
	if err != nil {
//通常我不喜欢图书馆代码中的恐慌，因为它
//可以在没有机会的情况下删除应用程序
//恢复这是非常烦人的，但是一个例外是
//在这种情况下，因为唯一的方法就是恐慌
//是否存在硬编码哈希中的错误。因此它
//只会在init上出现恐慌，因此
//100%可预测。
		panic(err)
	}
	return hash
}

func init() {
//初始化包时注册所有默认网络。
	mustRegister(&MainNetParams)
	mustRegister(&TestNet3Params)
	mustRegister(&RegressionNetParams)
	mustRegister(&SimNetParams)
}
