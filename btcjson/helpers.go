
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2014 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package btcjson

//bool是一个助手例程，它分配一个新的bool值来存储v和
//返回指向它的指针。这在分配可选参数时很有用。
func Bool(v bool) *bool {
	p := new(bool)
	*p = v
	return p
}

//int是一个助手例程，它分配一个新的int值来存储v和
//返回指向它的指针。这在分配可选参数时很有用。
func Int(v int) *int {
	p := new(int)
	*p = v
	return p
}

//uint是一个助手例程，它为存储v和
//返回指向它的指针。这在分配可选参数时很有用。
func Uint(v uint) *uint {
	p := new(uint)
	*p = v
	return p
}

//Int32是一个助手例程，它分配一个新的Int32值来存储v和
//返回指向它的指针。这在分配可选参数时很有用。
func Int32(v int32) *int32 {
	p := new(int32)
	*p = v
	return p
}

//uint32是一个助手例程，它为存储v和
//返回指向它的指针。这在分配可选参数时很有用。
func Uint32(v uint32) *uint32 {
	p := new(uint32)
	*p = v
	return p
}

//Int64是一个帮助程序，它分配一个新的Int64值来存储v和
//返回指向它的指针。这在分配可选参数时很有用。
func Int64(v int64) *int64 {
	p := new(int64)
	*p = v
	return p
}

//uint64是一个助手例程，它为存储v和
//返回指向它的指针。这在分配可选参数时很有用。
func Uint64(v uint64) *uint64 {
	p := new(uint64)
	*p = v
	return p
}

//float64是一个助手例程，它为存储v和
//返回指向它的指针。这在分配可选参数时很有用。
func Float64(v float64) *float64 {
	p := new(float64)
	*p = v
	return p
}

//字符串是一个助手例程，它分配一个新的字符串值来存储v和
//返回指向它的指针。这在分配可选参数时很有用。
func String(v string) *string {
	p := new(string)
	*p = v
	return p
}
