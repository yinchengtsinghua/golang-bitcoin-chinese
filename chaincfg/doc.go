
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//package chaincfg定义链配置参数。
//
//除主要比特币网络外，该网络用于传输
//在货币价值方面，目前还存在两个活跃的标准网络：
//回归测试和测试网（版本3）。这些网络不兼容
//彼此（每个共享一个不同的Genesis块）和软件应该
//在应用程序上使用一个网络的输入时处理错误
//在其他网络上运行的实例。
//
//对于库包，chaincfg提供了查找链的功能
//参数和编码在传递*参数时变魔术。旧的API未更新
//对于传递*参数的新约定，可以查找
//Wire.bitcoinnet使用paramsfornet，但请注意此用法是
//已弃用，将来将从chainecfg中删除。
//
//对于主包，可以为（通常是全局的）var分配
//作为应用程序的“活动”网络使用的标准参数变量之一。
//当需要一个网络参数时，可以通过这个
//变量（直接或隐藏在库调用中）。
//
//包装主体
//
//进口（
//“旗帜”
//“FMT”
//“日志”
//
//"github.com/btcsuite/btcutil"
//“github.com/btcsuite/btcd/chaincfg（Github.com/btcsuite/btcd/chaincfg）”。
//）
//
//var testnet=flag.bool（“testnet”，false，“在testnet比特币网络上操作”）。
//
////默认情况下（不带-testnet），使用mainnet。
//var chainparams=&chaincfg.mainnetparams
//
//FUNC主体（）
//标记（PARSE）
//
////如果在testnet上操作，则修改活动的网络参数。
//如果*TestNET{
//chainParams=&chainCfg.testNet3Params
//}
//
//…
//
////创建并打印特定于活动网络的新付款地址。
//pubkeyhash：=make（[]字节，20）
//地址，错误：=btcutil.newAddressPubKeyHash（PubKeyHash，chainParams）
//如果犯错！= nIL{
//日志致命（错误）
//}
//fmt.println（地址）
//}
//
//如果应用程序不使用三个标准比特币网络中的一个，
//可以创建新的params结构，该结构定义
//非标准网络。作为一般经验法则，所有网络参数
//应该是网络独有的，但参数冲突仍然可能发生
//（不幸的是，使用regtest和testnet3共享magics就是这样）。
package chaincfg
