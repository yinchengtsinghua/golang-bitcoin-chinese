
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2014 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

/*
包btcec实现了对比特币所需椭圆曲线的支持。

比特币使用椭圆曲线加密，使用Koblitz曲线
（特别是secp256k1）用于加密函数。见
http://www.secg.org/collateral/sec2_final.pdf了解
标准。

此包提供实现
加密/椭圆曲线接口，以允许使用这些曲线
使用Go提供的标准Crypto/ECDSA包。帮手
提供了分析签名和公钥的功能
标准格式。它设计用于BTCD，但应该


**/

package btcec
