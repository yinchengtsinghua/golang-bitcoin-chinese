
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

package main

import (
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
)

//activenetparams是指向特定于
//当前活跃的比特币网络。
var activeNetParams = &mainNetParams

//参数用于对各种网络的参数进行分组，例如
//网络和测试网络。
type params struct {
	*chaincfg.Params
	rpcPort string
}

//mainnetparams包含特定于主网络的参数
//（Wire.Mainnet）。注意：rpc端口有意与
//参考实现，因为BTCD不处理钱包请求。这个
//单独的钱包进程监听已知端口并转发请求
//它不能处理BTCD。这种方法允许钱包过程
//以模拟完整引用实现rpc api。
var mainNetParams = params{
	Params:  &chaincfg.MainNetParams,
	rpcPort: "8334",
}

//regressionnetparams包含特定于回归测试的参数
//网络（Wire.TestNet）。注意：rpc端口故意不同
//而不是引用实现-有关
//细节。
var regressionNetParams = params{
	Params:  &chaincfg.RegressionNetParams,
	rpcPort: "18334",
}

//testnet3参数包含特定于测试网络的参数（版本3）
//（Wire.TestNet3）。注意：rpc端口有意与
//参考实现-有关详细信息，请参阅mainnetparams注释。
var testNet3Params = params{
	Params:  &chaincfg.TestNet3Params,
	rpcPort: "18334",
}

//simnetparams包含特定于模拟测试网络的参数
//（电线，SimNet）
var simNetParams = params{
	Params:  &chaincfg.SimNetParams,
	rpcPort: "18556",
}

//netname返回引用比特币网络时使用的名称。在
//写入时间，btcd当前将testnet版本3的块放在
//数据和日志目录“testnet”，与
//CHAINCFG参数。此函数可用于重写此目录
//当传递的活动网络与Wire.TestNet3匹配时，将其命名为“testnet”。
//
//要将此网络的数据和日志目录移动到
//“testnet3”是为将来而计划的，此时，此功能可以
//已删除，并改用网络参数的名称。
func netName(chainParams *params) string {
	switch chainParams.Net {
	case wire.TestNet3:
		return "testnet"
	default:
		return chainParams.Name
	}
}
