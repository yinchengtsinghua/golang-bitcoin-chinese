
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015版权所有
//版权所有（c）2016-2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package chainhash

import "crypto/sha256"

//hash b计算hash（b）并返回结果字节。
func HashB(b []byte) []byte {
	hash := sha256.Sum256(b)
	return hash[:]
}

//hash计算散列（b）并以散列形式返回结果字节。
func HashH(b []byte) Hash {
	return Hash(sha256.Sum256(b))
}

//doublehashb计算散列（hash（b））并返回结果字节。
func DoubleHashB(b []byte) []byte {
	first := sha256.Sum256(b)
	second := sha256.Sum256(first[:])
	return second[:]
}

//doublehash计算哈希（hash（b）），并将结果字节返回为
//搞砸。
func DoubleHashH(b []byte) Hash {
	first := sha256.Sum256(b)
	return Hash(sha256.Sum256(first[:]))
}
