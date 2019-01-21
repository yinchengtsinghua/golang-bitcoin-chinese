
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

package blockchain

//TimeSorter实现Sort.Interface以允许时间戳切片
//分类。
type timeSorter []int64

//len返回切片中的时间戳数。它是
//
func (s timeSorter) Len() int {
	return len(s)
}

//
//Sort.Interface实现。
func (s timeSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

//less返回索引为i的timstamp是否应在
//带有索引j的时间戳。它是sort.interface实现的一部分。
func (s timeSorter) Less(i, j int) bool {
	return s[i] < s[j]
}
