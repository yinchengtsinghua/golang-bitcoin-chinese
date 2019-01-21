
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2016 BTCSuite开发者
//版权所有（c）2013-2016 Dave Collins
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package btcec

//参考文献：
//【HAC】：应用密码学手册，门尼泽，范奥斯乔，万通。
//http://cacr.uwaterloo.ca/hac/

//
//以256位素数为特征的。鉴于此精度大于
//最大的可用本地类型，显然需要某种形式的bignum数学。
//
//比依靠一个任意精度的算术包，如math/big
//用于处理字段数学，因为大小是已知的。结果，更确切地说
//
//优化不适用于任意精度算术和泛型
//模块化算术算法。
//
//在内部表示每个有限域单元有多种方法。
//例如，最明显的表示方法是使用4的数组
//uint64s（64位*4=256位）。然而，这种代表性受到
//几个问题。首先，没有足够大的本地Go类型来处理
//两个64位数字相加或相乘时的中间结果，以及
//
//每一个数组元素之间的算术运算会导致昂贵的进位
//传播。
//
//鉴于上述情况，此实现将字段元素表示为
//10个uint32，每个字（数组项）被视为基数2^26。这是
//选择原因如下：
//1）目前大多数系统都是64位（或至少有64位）
//专用寄存器，如mmx），因此
//中间结果通常可以使用本机寄存器（和
//使用uint64s避免需要额外的半字算术）
//2）为了不必添加内部单词
//
//
//3）由于我们处理的是32位值，64位溢出是
//合理选择2
//4）鉴于需要256位精度和1中所述的属性，
//2和3，最适合这一点的表示是10个uint32
//以2^26为基数（26位*10=260位，因此最后一个字只需要22位
//位），其留下所需的64位（32*10=320，320-256=64）
//溢流
//
//因为它是如此重要，以至于场的运算速度非常快
//高性能加密，此包不执行任何验证，其中
//通常会。例如，一些函数只给出正确的
//结果是字段被规范化，并且没有检查以确保它是规范化的。
//
//包，此代码实际上只在内部使用，并且每次额外检查
//计数。

import (
	"encoding/hex"
)

//用于提高代码可读性的常量。
const (
	twoBitsMask   = 0x3
	fourBitsMask  = 0xf
	sixBitsMask   = 0x3f
	eightBitsMask = 0xff
)

//与字段表示形式相关的常量。
const (
//FieldWords是用于在内部表示
//256位值。
	fieldWords = 10

//
//2^（fieldbase*i），其中i是单词position。
	fieldBase = 26

//FieldOverflowBits是每个字段的最小“溢出”位数。
//字段值中的字。
	fieldOverflowBits = 32 - fieldBase

//FieldBaseMask是每个字中需要的位的掩码，用于
//
//单词）。
	fieldBaseMask = (1 << fieldBase) - 1

//fieldmsbbits是所用最重要字中的位数。
//表示值。
	fieldMSBBits = 256 - (fieldBase * (fieldWords - 1))

//fieldmsbmask是最重要字中位的掩码。
//需要表示值。
	fieldMSBMask = (1 << fieldMSBBits) - 1

//FieldPrimeWordZero是中secp256k1 prime的字零
//内部字段表示。在否定时使用。
	fieldPrimeWordZero = 0x3fffc2f

//FieldPrimeWordOne是
//内部字段表示。在否定时使用。
	fieldPrimeWordOne = 0x3ffffbf
)

//FieldVal在
//
//0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffc2f.it
//以2^26为基数，将每个256位值表示为10个32位整数。这个
//每个字提供6位溢出（最重要的10位
//字）总共64位溢出（9*6+10=64）。它只实现
//椭圆曲线运算所需的算法。
//
//以下描述了内部表示：
//
//_n[9]n[8]…_n[0]
//32位可用32位可用…32位可用
//22位表示值26位表示值…26位表示值_
//10位溢出6位溢出…6位溢出
//Mult:2^（26*9）Mult:2^（26*8）…Mult:2^（26*0）
//————————————————————————————————————————————————————————————————
//
//
//n〔0〕＝1
//n〔1〕＝2 ^ 23
//n [ 2…9 ]＝0
//
//然后，通过从9..0循环I计算完整的256位值，并
//这样做总和（n[i]*2^（26i））：
//n[9]*2^（26*9）=0*2^234=0
//n[8]*2^（26*8）=0*2^208=0
//…
//n[1]*2^（26*1）=2^23*2^26=2^49
//n[0]*2^（26*0）=1*2^0=1
//总和：0+0+…+2^49+1=2^49+1
type fieldVal struct {
	n [10]uint32
}

//
func (f fieldVal) String() string {
	t := new(fieldVal).Set(&f).Normalize()
	return hex.EncodeToString(t.Bytes()[:])
}

//零将字段值设置为零。新创建的字段值已经
//设置为零。此函数可用于清除现有字段值
//重复使用。
func (f *fieldVal) Zero() {
	f.n[0] = 0
	f.n[1] = 0
	f.n[2] = 0
	f.n[3] = 0
	f.n[4] = 0
	f.n[5] = 0
	f.n[6] = 0
	f.n[7] = 0
	f.n[8] = 0
	f.n[9] = 0
}

//set将字段值设置为等于传递的值。
//
//返回字段值以支持链接。这将启用如下语法：
//f：=新建（fieldval）。设置（f2）。添加（1），使f=f2+1，其中f2不是
//被改进的。
func (f *fieldVal) Set(val *fieldVal) *fieldVal {
	*f = *val
	return f
}

//setint将字段值设置为传递的整数。这是个方便
//函数，因为使用小的
//本机整数。
//
//返回字段值以支持链接。这将启用以下语法：
//因为f：=new（fieldval）.setint（2）.mul（f2），所以f=2*f2。
func (f *fieldVal) SetInt(ui uint) *fieldVal {
	f.Zero()
	f.n[0] = uint32(ui)
	return f
}

//setbytes将传递的32字节big endian值打包到内部字段中
//
//
//返回字段值以支持链接。这将启用如下语法：
//f：=new（fieldval）.setbytes（bytearray）.mul（f2），使f=ba*f2。
func (f *fieldVal) SetBytes(b *[32]byte) *fieldVal {
//将10个uint32字的256个总位打包，最大值为
//每个字26位。这可以通过几个for循环来完成，
//但这个展开的版本要快得多。基准显示
//这比使用循环的变体快34倍。
	f.n[0] = uint32(b[31]) | uint32(b[30])<<8 | uint32(b[29])<<16 |
		(uint32(b[28])&twoBitsMask)<<24
	f.n[1] = uint32(b[28])>>2 | uint32(b[27])<<6 | uint32(b[26])<<14 |
		(uint32(b[25])&fourBitsMask)<<22
	f.n[2] = uint32(b[25])>>4 | uint32(b[24])<<4 | uint32(b[23])<<12 |
		(uint32(b[22])&sixBitsMask)<<20
	f.n[3] = uint32(b[22])>>6 | uint32(b[21])<<2 | uint32(b[20])<<10 |
		uint32(b[19])<<18
	f.n[4] = uint32(b[18]) | uint32(b[17])<<8 | uint32(b[16])<<16 |
		(uint32(b[15])&twoBitsMask)<<24
	f.n[5] = uint32(b[15])>>2 | uint32(b[14])<<6 | uint32(b[13])<<14 |
		(uint32(b[12])&fourBitsMask)<<22
	f.n[6] = uint32(b[12])>>4 | uint32(b[11])<<4 | uint32(b[10])<<12 |
		(uint32(b[9])&sixBitsMask)<<20
	f.n[7] = uint32(b[9])>>6 | uint32(b[8])<<2 | uint32(b[7])<<10 |
		uint32(b[6])<<18
	f.n[8] = uint32(b[5]) | uint32(b[4])<<8 | uint32(b[3])<<16 |
		(uint32(b[2])&twoBitsMask)<<24
	f.n[9] = uint32(b[2])>>2 | uint32(b[1])<<6 | uint32(b[0])<<14
	return f
}

//setbyteslice将传递的big endian值打包到内部字段值中
//代表。只使用前32个字节。因此，这取决于
//调用方确保使用适当大小的数字或值
//将被截断。
//
//返回字段值以支持链接。这将启用如下语法：
//F：=新建（fieldval）.setbyteslice（byteslice）
func (f *fieldVal) SetByteSlice(b []byte) *fieldVal {
	var b32 [32]byte
	for i := 0; i < len(b); i++ {
		if i < 32 {
			b32[i+(32-len(b))] = b[i]
		}
	}
	return f.SetBytes(&b32)
}

//sethex将传递的big endian十六进制字符串解码为内部字段值
//代表。只使用前32个字节。
//
//返回字段值以支持链接。这将启用如下语法：
//F：=new（fieldval）.sethex（“0abc”）.add（1）使F=0x0abc+1
func (f *fieldVal) SetHex(hexString string) *fieldVal {
	if len(hexString)%2 != 0 {
		hexString = "0" + hexString
	}
	bytes, _ := hex.DecodeString(hexString)
	return f.SetByteSlice(bytes)
}

//normalize将内部字段词规范化为所需的范围，并
//通过使用
//素数的特殊形式。
func (f *fieldVal) Normalize() *fieldVal {
//字段表示法在每个字中留下6位溢出，因此
//中间计算无需
//在计算期间将进位传播到每个更高的字。在
//为了规范化，我们需要将完整的256位值“压缩”到
//传播任何一个的权利都是高阶的。
//
//
//因为这个域是做secp256k1素数的算术模，所以我们
//还需要在基本阶段执行模块化缩减。
//
//根据[HAC]第14.3.4节：特殊形式模量的折减方法，
//当模量为特殊形式m=b^t-c时，效率很高
//可以减少。
//
//secp256k1 prime相当于2^256-4294968273，因此它适合
//这个标准。
//
//4294968273字段表示（以2^26为基数）是：
//n〔0〕＝977
//n〔1〕＝64
//
//
//参考章节中给出的算法通常重复
//直到商为零。但是，由于我们的现场代表
//
//重复，因为它是高位字的最高位。因此我们
//可以简单地将震级乘以
//启动并执行一次迭代。在这一步之后，可能会有一个
//附加进位到位256（高位字的位22）。
	t9 := f.n[9]
	m := t9 >> fieldMSBBits
	t9 = t9 & fieldMSBMask
	t0 := f.n[0] + m*977
	t1 := (t0 >> fieldBase) + f.n[1] + (m << 6)
	t0 = t0 & fieldBaseMask
	t2 := (t1 >> fieldBase) + f.n[2]
	t1 = t1 & fieldBaseMask
	t3 := (t2 >> fieldBase) + f.n[3]
	t2 = t2 & fieldBaseMask
	t4 := (t3 >> fieldBase) + f.n[4]
	t3 = t3 & fieldBaseMask
	t5 := (t4 >> fieldBase) + f.n[5]
	t4 = t4 & fieldBaseMask
	t6 := (t5 >> fieldBase) + f.n[6]
	t5 = t5 & fieldBaseMask
	t7 := (t6 >> fieldBase) + f.n[7]
	t6 = t6 & fieldBaseMask
	t8 := (t7 >> fieldBase) + f.n[8]
	t7 = t7 & fieldBaseMask
	t9 = (t8 >> fieldBase) + t9
	t8 = t8 & fieldBaseMask

//
//如果存在
//执行到位256（高位字的位22）或
//值大于或等于字段特征。这个
//以下内容确定这些条件是否为真
//在恒定时间内的最终减少。
//
//注意这里的if/else语句有意按位执行
//操作员即使不改变值也能保证固定时间
//在树枝之间。还要注意，当两者都不存在时，“m”将为零。
//上述条件中的条件是真实的，且该值不会
//“m”为零时更改。
	m = 1
	if t9 == fieldMSBMask {
		m &= 1
	} else {
		m &= 0
	}
	if t2&t3&t4&t5&t6&t7&t8 == fieldBaseMask {
		m &= 1
	} else {
		m &= 0
	}
	if ((t0+977)>>fieldBase + t1 + 64) > fieldBaseMask {
		m &= 1
	} else {
		m &= 0
	}
	if t9>>fieldMSBBits != 0 {
		m |= 1
	} else {
		m |= 0
	}
	t0 = t0 + m*977
	t1 = (t0 >> fieldBase) + t1 + (m << 6)
	t0 = t0 & fieldBaseMask
	t2 = (t1 >> fieldBase) + t2
	t1 = t1 & fieldBaseMask
	t3 = (t2 >> fieldBase) + t3
	t2 = t2 & fieldBaseMask
	t4 = (t3 >> fieldBase) + t4
	t3 = t3 & fieldBaseMask
	t5 = (t4 >> fieldBase) + t5
	t4 = t4 & fieldBaseMask
	t6 = (t5 >> fieldBase) + t6
	t5 = t5 & fieldBaseMask
	t7 = (t6 >> fieldBase) + t7
	t6 = t6 & fieldBaseMask
	t8 = (t7 >> fieldBase) + t8
	t7 = t7 & fieldBaseMask
	t9 = (t8 >> fieldBase) + t9
	t8 = t8 & fieldBaseMask
t9 = t9 & fieldMSBMask //删除2^256的潜在倍数。

//最后，设置归一化和约简字。
	f.n[0] = t0
	f.n[1] = t1
	f.n[2] = t2
	f.n[3] = t3
	f.n[4] = t4
	f.n[5] = t5
	f.n[6] = t6
	f.n[7] = t7
	f.n[8] = t8
	f.n[9] = t9
	return f
}

//PutBytes使用
//传递了字节数组。有一个类似的函数bytes，它解包
//将字段值输入新数组并返回该值。提供此版本
//因为通过允许
//
//
//必须规范化字段值，此函数才能返回正确的
//结果。
func (f *fieldVal) PutBytes(b *[32]byte) {
//从10个uint32字中解包256个总位，最大值为
//每个字26位。这可以通过几个for循环来完成，
//但这个展开的版本要快一点。基准测试表明
//比使用循环的变体快10倍。
	b[31] = byte(f.n[0] & eightBitsMask)
	b[30] = byte((f.n[0] >> 8) & eightBitsMask)
	b[29] = byte((f.n[0] >> 16) & eightBitsMask)
	b[28] = byte((f.n[0]>>24)&twoBitsMask | (f.n[1]&sixBitsMask)<<2)
	b[27] = byte((f.n[1] >> 6) & eightBitsMask)
	b[26] = byte((f.n[1] >> 14) & eightBitsMask)
	b[25] = byte((f.n[1]>>22)&fourBitsMask | (f.n[2]&fourBitsMask)<<4)
	b[24] = byte((f.n[2] >> 4) & eightBitsMask)
	b[23] = byte((f.n[2] >> 12) & eightBitsMask)
	b[22] = byte((f.n[2]>>20)&sixBitsMask | (f.n[3]&twoBitsMask)<<6)
	b[21] = byte((f.n[3] >> 2) & eightBitsMask)
	b[20] = byte((f.n[3] >> 10) & eightBitsMask)
	b[19] = byte((f.n[3] >> 18) & eightBitsMask)
	b[18] = byte(f.n[4] & eightBitsMask)
	b[17] = byte((f.n[4] >> 8) & eightBitsMask)
	b[16] = byte((f.n[4] >> 16) & eightBitsMask)
	b[15] = byte((f.n[4]>>24)&twoBitsMask | (f.n[5]&sixBitsMask)<<2)
	b[14] = byte((f.n[5] >> 6) & eightBitsMask)
	b[13] = byte((f.n[5] >> 14) & eightBitsMask)
	b[12] = byte((f.n[5]>>22)&fourBitsMask | (f.n[6]&fourBitsMask)<<4)
	b[11] = byte((f.n[6] >> 4) & eightBitsMask)
	b[10] = byte((f.n[6] >> 12) & eightBitsMask)
	b[9] = byte((f.n[6]>>20)&sixBitsMask | (f.n[7]&twoBitsMask)<<6)
	b[8] = byte((f.n[7] >> 2) & eightBitsMask)
	b[7] = byte((f.n[7] >> 10) & eightBitsMask)
	b[6] = byte((f.n[7] >> 18) & eightBitsMask)
	b[5] = byte(f.n[8] & eightBitsMask)
	b[4] = byte((f.n[8] >> 8) & eightBitsMask)
	b[3] = byte((f.n[8] >> 16) & eightBitsMask)
	b[2] = byte((f.n[8]>>24)&twoBitsMask | (f.n[9]&sixBitsMask)<<2)
	b[1] = byte((f.n[9] >> 6) & eightBitsMask)
	b[0] = byte((f.n[9] >> 14) & eightBitsMask)
}

//字节将字段值解包为32字节的big endian值。参见字节
//
//通过允许调用者重用
//缓冲器。
//
//必须规范化字段值，此函数才能返回正确的
//结果。
func (f *fieldVal) Bytes() *[32]byte {
	b := new([32]byte)
	f.PutBytes(b)
	return b
}

//is zero返回字段值是否等于零。
func (f *fieldVal) IsZero() bool {
//只有在任何字中未设置任何位时，该值才可以为零。
//这是一个固定时间的实现。
	bits := f.n[0] | f.n[1] | f.n[2] | f.n[3] | f.n[4] |
		f.n[5] | f.n[6] | f.n[7] | f.n[8] | f.n[9]

	return bits == 0
}

//
//
//必须规范化字段值，此函数才能返回正确的
//结果。
func (f *fieldVal) IsOdd() bool {
//只有奇数才有底位集。
	return f.n[0]&1 == 1
}

//等于返回两个字段值是否相同。两个
//要使此函数返回正在比较的字段值，必须对其进行规范化
//正确的结果。
func (f *fieldVal) Equals(val *fieldVal) bool {
//XOR只在不同的情况下设置位，因此这两个字段值
//只有在对每个字执行异或运算后未设置任何位时，才能相同。
//这是一个固定时间的实现。
	bits := (f.n[0] ^ val.n[0]) | (f.n[1] ^ val.n[1]) | (f.n[2] ^ val.n[2]) |
		(f.n[3] ^ val.n[3]) | (f.n[4] ^ val.n[4]) | (f.n[5] ^ val.n[5]) |
		(f.n[6] ^ val.n[6]) | (f.n[7] ^ val.n[7]) | (f.n[8] ^ val.n[8]) |
		(f.n[9] ^ val.n[9])

	return bits == 0
}

//negateval否定传递的值并将结果存储在f中。调用方
//必须提供传递值的大小才能得到正确的结果。
//
//返回字段值以支持链接。这将启用如下语法：
//F.Negateval（F2）。添加剂（1），使F=-F2+1。
func (f *fieldVal) NegateVal(val *fieldVal, magnitude uint32) *fieldVal {
//字段中的负数只是质数减去值。然而，
//为了允许对字段值求反而不必
//先将其规格化/缩小，再乘以大小（即
//"far" away it is from the normalized value) to adjust.  此外，由于
//对一个值求反会使它远离
//归一化范围，加1进行补偿。
//
//对于这里的一些直觉，假设您正在执行mod 12算术
//(picture a clock) and you are negating the number 7.  所以你开始
//12（当然是12型下的0）和倒计数（左上
//时钟）7次到5点。注意这只是12-7=5。
//现在，假设你从19开始，这是一个数字
//already larger than the modulus and congruent to 7 (mod 12).  当A
//值已在所需范围内，其大小为1。19以来
//是一个额外的“步骤”，它的幅度（mod 12）是2。因为任何
//模量的倍数等于零（mod m），答案可以
//通过简单地将幅度乘以模量和
//减去。与示例保持一致，这将是（2*12）-19=5。
	f.n[0] = (magnitude+1)*fieldPrimeWordZero - val.n[0]
	f.n[1] = (magnitude+1)*fieldPrimeWordOne - val.n[1]
	f.n[2] = (magnitude+1)*fieldBaseMask - val.n[2]
	f.n[3] = (magnitude+1)*fieldBaseMask - val.n[3]
	f.n[4] = (magnitude+1)*fieldBaseMask - val.n[4]
	f.n[5] = (magnitude+1)*fieldBaseMask - val.n[5]
	f.n[6] = (magnitude+1)*fieldBaseMask - val.n[6]
	f.n[7] = (magnitude+1)*fieldBaseMask - val.n[7]
	f.n[8] = (magnitude+1)*fieldBaseMask - val.n[8]
	f.n[9] = (magnitude+1)*fieldMSBMask - val.n[9]

	return f
}

//negate使字段值为负数。已修改现有字段值。这个
//
//
//返回字段值以支持链接。这将启用如下语法：
//f.Negate().AddInt(1) so that f = -f + 1.
func (f *fieldVal) Negate(magnitude uint32) *fieldVal {
	return f.NegateVal(f, magnitude)
}

//AddInt adds the passed integer to the existing field value and stores the
//result in f.  This is a convenience function since it is fairly common to
//用小的本机整数执行一些算术运算。
//
//返回字段值以支持链接。这将启用如下语法：
//f.AddInt(1).Add(f2) so that f = f + 1 + f2.
func (f *fieldVal) AddInt(ui uint) *fieldVal {
//由于字段表示有意提供溢出位，
//it's ok to use carryless addition as the carry bit is safely part of
//the word and will be normalized out.
	f.n[0] += uint32(ui)

	return f
}

//添加将传递的值添加到现有字段值并存储结果
//在F
//
//返回字段值以支持链接。这将启用如下语法：
//F.添加（F2）。添加（1），使F=F+F2+1。
func (f *fieldVal) Add(val *fieldVal) *fieldVal {
//由于字段表示有意提供溢出位，
//it's ok to use carryless addition as the carry bit is safely part of
//每个单词和将被规范化。这显然可以做到
//在循环中，但展开的版本更快。
	f.n[0] += val.n[0]
	f.n[1] += val.n[1]
	f.n[2] += val.n[2]
	f.n[3] += val.n[3]
	f.n[4] += val.n[4]
	f.n[5] += val.n[5]
	f.n[6] += val.n[6]
	f.n[7] += val.n[7]
	f.n[8] += val.n[8]
	f.n[9] += val.n[9]

	return f
}

//add2将传递的两个字段值相加，并将结果存储在f中。
//
//返回字段值以支持链接。这将启用如下语法：
//f3.加2（f，f2）。加1，使f3=f+f2+1。
func (f *fieldVal) Add2(val *fieldVal, val2 *fieldVal) *fieldVal {
//由于字段表示有意提供溢出位，
//由于进位是安全的一部分，所以可以使用无进位加法。
//每个单词和将被规范化。这显然可以做到
//在循环中，但展开的版本更快。
	f.n[0] = val.n[0] + val2.n[0]
	f.n[1] = val.n[1] + val2.n[1]
	f.n[2] = val.n[2] + val2.n[2]
	f.n[3] = val.n[3] + val2.n[3]
	f.n[4] = val.n[4] + val2.n[4]
	f.n[5] = val.n[5] + val2.n[5]
	f.n[6] = val.n[6] + val2.n[6]
	f.n[7] = val.n[7] + val2.n[7]
	f.n[8] = val.n[8] + val2.n[8]
	f.n[9] = val.n[9] + val2.n[9]

	return f
}

//mulint将字段值乘以传递的int，并将结果存储在
//f.请注意，如果将该值乘以
//单个单词超过了最大值uint32。因此，重要的是
//调用方确保在使用此函数之前不会发生溢出。
//
//返回字段值以支持链接。这将启用如下语法：
//F.Mulint（2）。添加（F2），使F=2*F+F2。
func (f *fieldVal) MulInt(val uint) *fieldVal {
//因为字段表示的每个字都可以
//FieldOverflowBits额外的位将被规范化，这是安全的。
//不使用较大的类型或进位，使每个字成倍增加
//propagation so long as the values won't overflow a uint32.  这个
//显然可以在循环中完成，但展开的版本是
//更快。
	ui := uint32(val)
	f.n[0] *= ui
	f.n[1] *= ui
	f.n[2] *= ui
	f.n[3] *= ui
	f.n[4] *= ui
	f.n[5] *= ui
	f.n[6] *= ui
	f.n[7] *= ui
	f.n[8] *= ui
	f.n[9] *= ui

	return f
}

//mul将传递的值乘以现有字段值，并存储
//结果为f。请注意，如果将任何
//单个单词的最大值超过了uint32。实际上，这意味着
//乘法中涉及的任一值的最大值必须是
//8。
//
//返回字段值以支持链接。这将启用如下语法：
//F.MUL（F2）。添加剂（1），使F=（F*F2）+1。
func (f *fieldVal) Mul(val *fieldVal) *fieldVal {
	return f.Mul2(f, val)
}

//
//结果为f。请注意，如果将任何
//单个单词超过了最大值uint32。实际上，这意味着
//乘法中涉及的任一值的最大值必须是
//8。
//
//返回字段值以支持链接。这将启用如下语法：
//f3.mul2（f，f2）。添加量（1），使f3=（f*f2）+1。
func (f *fieldVal) Mul2(val *fieldVal, val2 *fieldVal) *fieldVal {
//这可以通过几个for循环和一个要存储的数组来完成。
//中间术语，但这个展开的版本
//更快。

//2^（FieldBase*0）的术语。
	m := uint64(val.n[0]) * uint64(val2.n[0])
	t0 := m & fieldBaseMask

//2^（FieldBase*1）的术语。
	m = (m >> fieldBase) +
		uint64(val.n[0])*uint64(val2.n[1]) +
		uint64(val.n[1])*uint64(val2.n[0])
	t1 := m & fieldBaseMask

//2^（FieldBase*2）的术语。
	m = (m >> fieldBase) +
		uint64(val.n[0])*uint64(val2.n[2]) +
		uint64(val.n[1])*uint64(val2.n[1]) +
		uint64(val.n[2])*uint64(val2.n[0])
	t2 := m & fieldBaseMask

//2^（FieldBase*3）的术语。
	m = (m >> fieldBase) +
		uint64(val.n[0])*uint64(val2.n[3]) +
		uint64(val.n[1])*uint64(val2.n[2]) +
		uint64(val.n[2])*uint64(val2.n[1]) +
		uint64(val.n[3])*uint64(val2.n[0])
	t3 := m & fieldBaseMask

//2^（FieldBase*4）的术语。
	m = (m >> fieldBase) +
		uint64(val.n[0])*uint64(val2.n[4]) +
		uint64(val.n[1])*uint64(val2.n[3]) +
		uint64(val.n[2])*uint64(val2.n[2]) +
		uint64(val.n[3])*uint64(val2.n[1]) +
		uint64(val.n[4])*uint64(val2.n[0])
	t4 := m & fieldBaseMask

//2^（fieldbase*5）的术语。
	m = (m >> fieldBase) +
		uint64(val.n[0])*uint64(val2.n[5]) +
		uint64(val.n[1])*uint64(val2.n[4]) +
		uint64(val.n[2])*uint64(val2.n[3]) +
		uint64(val.n[3])*uint64(val2.n[2]) +
		uint64(val.n[4])*uint64(val2.n[1]) +
		uint64(val.n[5])*uint64(val2.n[0])
	t5 := m & fieldBaseMask

//2^（FieldBase*6）的术语。
	m = (m >> fieldBase) +
		uint64(val.n[0])*uint64(val2.n[6]) +
		uint64(val.n[1])*uint64(val2.n[5]) +
		uint64(val.n[2])*uint64(val2.n[4]) +
		uint64(val.n[3])*uint64(val2.n[3]) +
		uint64(val.n[4])*uint64(val2.n[2]) +
		uint64(val.n[5])*uint64(val2.n[1]) +
		uint64(val.n[6])*uint64(val2.n[0])
	t6 := m & fieldBaseMask

//
	m = (m >> fieldBase) +
		uint64(val.n[0])*uint64(val2.n[7]) +
		uint64(val.n[1])*uint64(val2.n[6]) +
		uint64(val.n[2])*uint64(val2.n[5]) +
		uint64(val.n[3])*uint64(val2.n[4]) +
		uint64(val.n[4])*uint64(val2.n[3]) +
		uint64(val.n[5])*uint64(val2.n[2]) +
		uint64(val.n[6])*uint64(val2.n[1]) +
		uint64(val.n[7])*uint64(val2.n[0])
	t7 := m & fieldBaseMask

//2^（FieldBase*8）的术语。
	m = (m >> fieldBase) +
		uint64(val.n[0])*uint64(val2.n[8]) +
		uint64(val.n[1])*uint64(val2.n[7]) +
		uint64(val.n[2])*uint64(val2.n[6]) +
		uint64(val.n[3])*uint64(val2.n[5]) +
		uint64(val.n[4])*uint64(val2.n[4]) +
		uint64(val.n[5])*uint64(val2.n[3]) +
		uint64(val.n[6])*uint64(val2.n[2]) +
		uint64(val.n[7])*uint64(val2.n[1]) +
		uint64(val.n[8])*uint64(val2.n[0])
	t8 := m & fieldBaseMask

//2^（fieldbase*9）的术语。
	m = (m >> fieldBase) +
		uint64(val.n[0])*uint64(val2.n[9]) +
		uint64(val.n[1])*uint64(val2.n[8]) +
		uint64(val.n[2])*uint64(val2.n[7]) +
		uint64(val.n[3])*uint64(val2.n[6]) +
		uint64(val.n[4])*uint64(val2.n[5]) +
		uint64(val.n[5])*uint64(val2.n[4]) +
		uint64(val.n[6])*uint64(val2.n[3]) +
		uint64(val.n[7])*uint64(val2.n[2]) +
		uint64(val.n[8])*uint64(val2.n[1]) +
		uint64(val.n[9])*uint64(val2.n[0])
	t9 := m & fieldBaseMask

//2^（fieldbase*10）的术语。
	m = (m >> fieldBase) +
		uint64(val.n[1])*uint64(val2.n[9]) +
		uint64(val.n[2])*uint64(val2.n[8]) +
		uint64(val.n[3])*uint64(val2.n[7]) +
		uint64(val.n[4])*uint64(val2.n[6]) +
		uint64(val.n[5])*uint64(val2.n[5]) +
		uint64(val.n[6])*uint64(val2.n[4]) +
		uint64(val.n[7])*uint64(val2.n[3]) +
		uint64(val.n[8])*uint64(val2.n[2]) +
		uint64(val.n[9])*uint64(val2.n[1])
	t10 := m & fieldBaseMask

//Terms for 2^(fieldBase*11).
	m = (m >> fieldBase) +
		uint64(val.n[2])*uint64(val2.n[9]) +
		uint64(val.n[3])*uint64(val2.n[8]) +
		uint64(val.n[4])*uint64(val2.n[7]) +
		uint64(val.n[5])*uint64(val2.n[6]) +
		uint64(val.n[6])*uint64(val2.n[5]) +
		uint64(val.n[7])*uint64(val2.n[4]) +
		uint64(val.n[8])*uint64(val2.n[3]) +
		uint64(val.n[9])*uint64(val2.n[2])
	t11 := m & fieldBaseMask

//2^（fieldbase*12）的术语。
	m = (m >> fieldBase) +
		uint64(val.n[3])*uint64(val2.n[9]) +
		uint64(val.n[4])*uint64(val2.n[8]) +
		uint64(val.n[5])*uint64(val2.n[7]) +
		uint64(val.n[6])*uint64(val2.n[6]) +
		uint64(val.n[7])*uint64(val2.n[5]) +
		uint64(val.n[8])*uint64(val2.n[4]) +
		uint64(val.n[9])*uint64(val2.n[3])
	t12 := m & fieldBaseMask

//2^（FieldBase*13）的术语。
	m = (m >> fieldBase) +
		uint64(val.n[4])*uint64(val2.n[9]) +
		uint64(val.n[5])*uint64(val2.n[8]) +
		uint64(val.n[6])*uint64(val2.n[7]) +
		uint64(val.n[7])*uint64(val2.n[6]) +
		uint64(val.n[8])*uint64(val2.n[5]) +
		uint64(val.n[9])*uint64(val2.n[4])
	t13 := m & fieldBaseMask

//
	m = (m >> fieldBase) +
		uint64(val.n[5])*uint64(val2.n[9]) +
		uint64(val.n[6])*uint64(val2.n[8]) +
		uint64(val.n[7])*uint64(val2.n[7]) +
		uint64(val.n[8])*uint64(val2.n[6]) +
		uint64(val.n[9])*uint64(val2.n[5])
	t14 := m & fieldBaseMask

//2^（fieldbase*15）的术语。
	m = (m >> fieldBase) +
		uint64(val.n[6])*uint64(val2.n[9]) +
		uint64(val.n[7])*uint64(val2.n[8]) +
		uint64(val.n[8])*uint64(val2.n[7]) +
		uint64(val.n[9])*uint64(val2.n[6])
	t15 := m & fieldBaseMask

//2^（fieldbase*16）的术语。
	m = (m >> fieldBase) +
		uint64(val.n[7])*uint64(val2.n[9]) +
		uint64(val.n[8])*uint64(val2.n[8]) +
		uint64(val.n[9])*uint64(val2.n[7])
	t16 := m & fieldBaseMask

//
	m = (m >> fieldBase) +
		uint64(val.n[8])*uint64(val2.n[9]) +
		uint64(val.n[9])*uint64(val2.n[8])
	t17 := m & fieldBaseMask

//2^（FieldBase*18）的术语。
	m = (m >> fieldBase) + uint64(val.n[9])*uint64(val2.n[9])
	t18 := m & fieldBaseMask

//剩下的是2^（FieldBase*19）。
	t19 := m >> fieldBase

//
//底座。
//
//根据[HAC]第14.3.4节：特殊形式模量的折减方法，
//当模量为特殊形式m=b^t-c时，效率很高
//根据提供的算法可以实现减少。
//
//secp256k1 prime相当于2^256-4294968273，因此它适合
//这个标准。
//
//4294968273字段表示（以2^26为基数）是：
//n〔0〕＝977
//n〔1〕＝64
//也就是说（2^26*64）+977=4294968273
//
//因为每个单词都以26为基数，所以上面的术语（t10和up）开始
//at 260 bits (versus the final desired range of 256 bits), so the
//
//将它乘以2^4=16得到额外的4位。4294968273*16=
//68719492368。因此，“c”的调整字段表示为：
//
//N[1]=64*16=1024
//也就是说（2^26*1024）+15632=68719492368
//
//为了减少最后一项t19，需要整个“c”值
//只有n[0]的，因为没有更多的术语可以处理n[1]。
//这意味着在上面的位中可能还有一个量级，
//在下面处理。
	m = t0 + t10*15632
	t0 = m & fieldBaseMask
	m = (m >> fieldBase) + t1 + t10*1024 + t11*15632
	t1 = m & fieldBaseMask
	m = (m >> fieldBase) + t2 + t11*1024 + t12*15632
	t2 = m & fieldBaseMask
	m = (m >> fieldBase) + t3 + t12*1024 + t13*15632
	t3 = m & fieldBaseMask
	m = (m >> fieldBase) + t4 + t13*1024 + t14*15632
	t4 = m & fieldBaseMask
	m = (m >> fieldBase) + t5 + t14*1024 + t15*15632
	t5 = m & fieldBaseMask
	m = (m >> fieldBase) + t6 + t15*1024 + t16*15632
	t6 = m & fieldBaseMask
	m = (m >> fieldBase) + t7 + t16*1024 + t17*15632
	t7 = m & fieldBaseMask
	m = (m >> fieldBase) + t8 + t17*1024 + t18*15632
	t8 = m & fieldBaseMask
	m = (m >> fieldBase) + t9 + t18*1024 + t19*68719492368
	t9 = m & fieldMSBMask
	m = m >> fieldMSBBits

//此时，如果震级大于0，则整体值
//大于最大可能256位值。尤其是
//
//
//[HAC]第14.3.4节中给出的算法重复到
//商是零。然而，由于上述原因，我们已经知道
//至少要重复几次，因为这是价值所在
//因此，我们可以简单地将震级乘以
//质数的场表示，并进行一次迭代。通知
//当震级为零时，什么都不会改变，所以我们可以
//在这种情况下跳过此项，但始终运行，无论是否允许
//在固定时间内运行。最终结果将在范围内
//0<=结果<=prime+（2^64-c），因此保证
//大小为1，但它是非规范化的。
	d := t0 + m*977
	f.n[0] = uint32(d & fieldBaseMask)
	d = (d >> fieldBase) + t1 + m*64
	f.n[1] = uint32(d & fieldBaseMask)
	f.n[2] = uint32((d >> fieldBase) + t2)
	f.n[3] = uint32(t3)
	f.n[4] = uint32(t4)
	f.n[5] = uint32(t5)
	f.n[6] = uint32(t6)
	f.n[7] = uint32(t7)
	f.n[8] = uint32(t8)
	f.n[9] = uint32(t9)

	return f
}

//平方等于字段值。已修改现有字段值。注释
//如果将单个单词相乘，则此函数可能溢出
//超过最大值uint32。实际上，这意味着磁场的大小
//
//
//返回字段值以支持链接。这将启用如下语法：
//f.square（）.mul（f2），使f=f^2*f2。
func (f *fieldVal) Square() *fieldVal {
	return f.SquareVal(f)
}

//Squareval将传递的值平方，并将结果存储在f中。请注意
//如果将单个单词相乘，则此函数可能溢出
//超过最大值uint32。实际上，这意味着磁场的大小
//为防止溢出，最大值必须为8。
//
//返回字段值以支持链接。这将启用如下语法：
//f3.squareval（f）.mul（f），使f3=f^2*f=f^3。
func (f *fieldVal) SquareVal(val *fieldVal) *fieldVal {
//这可以通过几个for循环和一个要存储的数组来完成。
//中间术语，但这个展开的版本
//更快。

//2^（FieldBase*0）的术语。
	m := uint64(val.n[0]) * uint64(val.n[0])
	t0 := m & fieldBaseMask

//2^（FieldBase*1）的术语。
	m = (m >> fieldBase) + 2*uint64(val.n[0])*uint64(val.n[1])
	t1 := m & fieldBaseMask

//2^（FieldBase*2）的术语。
	m = (m >> fieldBase) +
		2*uint64(val.n[0])*uint64(val.n[2]) +
		uint64(val.n[1])*uint64(val.n[1])
	t2 := m & fieldBaseMask

//2^（FieldBase*3）的术语。
	m = (m >> fieldBase) +
		2*uint64(val.n[0])*uint64(val.n[3]) +
		2*uint64(val.n[1])*uint64(val.n[2])
	t3 := m & fieldBaseMask

//2^（FieldBase*4）的术语。
	m = (m >> fieldBase) +
		2*uint64(val.n[0])*uint64(val.n[4]) +
		2*uint64(val.n[1])*uint64(val.n[3]) +
		uint64(val.n[2])*uint64(val.n[2])
	t4 := m & fieldBaseMask

//2^（fieldbase*5）的术语。
	m = (m >> fieldBase) +
		2*uint64(val.n[0])*uint64(val.n[5]) +
		2*uint64(val.n[1])*uint64(val.n[4]) +
		2*uint64(val.n[2])*uint64(val.n[3])
	t5 := m & fieldBaseMask

//2^（FieldBase*6）的术语。
	m = (m >> fieldBase) +
		2*uint64(val.n[0])*uint64(val.n[6]) +
		2*uint64(val.n[1])*uint64(val.n[5]) +
		2*uint64(val.n[2])*uint64(val.n[4]) +
		uint64(val.n[3])*uint64(val.n[3])
	t6 := m & fieldBaseMask

//2^（fieldbase*7）的术语。
	m = (m >> fieldBase) +
		2*uint64(val.n[0])*uint64(val.n[7]) +
		2*uint64(val.n[1])*uint64(val.n[6]) +
		2*uint64(val.n[2])*uint64(val.n[5]) +
		2*uint64(val.n[3])*uint64(val.n[4])
	t7 := m & fieldBaseMask

//2^（FieldBase*8）的术语。
	m = (m >> fieldBase) +
		2*uint64(val.n[0])*uint64(val.n[8]) +
		2*uint64(val.n[1])*uint64(val.n[7]) +
		2*uint64(val.n[2])*uint64(val.n[6]) +
		2*uint64(val.n[3])*uint64(val.n[5]) +
		uint64(val.n[4])*uint64(val.n[4])
	t8 := m & fieldBaseMask

//2^（fieldbase*9）的术语。
	m = (m >> fieldBase) +
		2*uint64(val.n[0])*uint64(val.n[9]) +
		2*uint64(val.n[1])*uint64(val.n[8]) +
		2*uint64(val.n[2])*uint64(val.n[7]) +
		2*uint64(val.n[3])*uint64(val.n[6]) +
		2*uint64(val.n[4])*uint64(val.n[5])
	t9 := m & fieldBaseMask

//2^（fieldbase*10）的术语。
	m = (m >> fieldBase) +
		2*uint64(val.n[1])*uint64(val.n[9]) +
		2*uint64(val.n[2])*uint64(val.n[8]) +
		2*uint64(val.n[3])*uint64(val.n[7]) +
		2*uint64(val.n[4])*uint64(val.n[6]) +
		uint64(val.n[5])*uint64(val.n[5])
	t10 := m & fieldBaseMask

//Terms for 2^(fieldBase*11).
	m = (m >> fieldBase) +
		2*uint64(val.n[2])*uint64(val.n[9]) +
		2*uint64(val.n[3])*uint64(val.n[8]) +
		2*uint64(val.n[4])*uint64(val.n[7]) +
		2*uint64(val.n[5])*uint64(val.n[6])
	t11 := m & fieldBaseMask

//2^（fieldbase*12）的术语。
	m = (m >> fieldBase) +
		2*uint64(val.n[3])*uint64(val.n[9]) +
		2*uint64(val.n[4])*uint64(val.n[8]) +
		2*uint64(val.n[5])*uint64(val.n[7]) +
		uint64(val.n[6])*uint64(val.n[6])
	t12 := m & fieldBaseMask

//2^（FieldBase*13）的术语。
	m = (m >> fieldBase) +
		2*uint64(val.n[4])*uint64(val.n[9]) +
		2*uint64(val.n[5])*uint64(val.n[8]) +
		2*uint64(val.n[6])*uint64(val.n[7])
	t13 := m & fieldBaseMask

//条款为2 ^（现场基地* 14）。
	m = (m >> fieldBase) +
		2*uint64(val.n[5])*uint64(val.n[9]) +
		2*uint64(val.n[6])*uint64(val.n[8]) +
		uint64(val.n[7])*uint64(val.n[7])
	t14 := m & fieldBaseMask

//2^（fieldbase*15）的术语。
	m = (m >> fieldBase) +
		2*uint64(val.n[6])*uint64(val.n[9]) +
		2*uint64(val.n[7])*uint64(val.n[8])
	t15 := m & fieldBaseMask

//2^（fieldbase*16）的术语。
	m = (m >> fieldBase) +
		2*uint64(val.n[7])*uint64(val.n[9]) +
		uint64(val.n[8])*uint64(val.n[8])
	t16 := m & fieldBaseMask

//Terms for 2^(fieldBase*17).
	m = (m >> fieldBase) + 2*uint64(val.n[8])*uint64(val.n[9])
	t17 := m & fieldBaseMask

//2^（FieldBase*18）的术语。
	m = (m >> fieldBase) + uint64(val.n[9])*uint64(val.n[9])
	t18 := m & fieldBaseMask

//剩下的是2^（FieldBase*19）。
	t19 := m >> fieldBase

//At this point, all of the terms are grouped into their respective
//底座。
//
//根据[HAC]第14.3.4节：特殊形式模量的折减方法，
//当模量为特殊形式m=b^t-c时，效率很高
//根据提供的算法可以实现减少。
//
//secp256k1 prime相当于2^256-4294968273，因此它适合
//这个标准。
//
//4294968273字段表示（以2^26为基数）是：
//n〔0〕＝977
//n〔1〕＝64
//也就是说（2^26*64）+977=4294968273
//
//因为每个单词都以26为基数，所以上面的术语（t10和up）开始
//在260位（相对于256位的最终期望范围），因此
//上述“c”的字段表示需要针对
//将它乘以2^4=16得到额外的4位。4294968273*16=
//68719492368。因此，“c”的调整字段表示为：
//N[0]=977*16=15632
//N[1]=64*16=1024
//也就是说（2^26*1024）+15632=68719492368
//
//为了减少最后一项t19，需要整个“c”值
//只有n[0]的，因为没有更多的术语可以处理n[1]。
//这意味着在上面的位中可能还有一个量级，
//在下面处理。
	m = t0 + t10*15632
	t0 = m & fieldBaseMask
	m = (m >> fieldBase) + t1 + t10*1024 + t11*15632
	t1 = m & fieldBaseMask
	m = (m >> fieldBase) + t2 + t11*1024 + t12*15632
	t2 = m & fieldBaseMask
	m = (m >> fieldBase) + t3 + t12*1024 + t13*15632
	t3 = m & fieldBaseMask
	m = (m >> fieldBase) + t4 + t13*1024 + t14*15632
	t4 = m & fieldBaseMask
	m = (m >> fieldBase) + t5 + t14*1024 + t15*15632
	t5 = m & fieldBaseMask
	m = (m >> fieldBase) + t6 + t15*1024 + t16*15632
	t6 = m & fieldBaseMask
	m = (m >> fieldBase) + t7 + t16*1024 + t17*15632
	t7 = m & fieldBaseMask
	m = (m >> fieldBase) + t8 + t17*1024 + t18*15632
	t8 = m & fieldBaseMask
	m = (m >> fieldBase) + t9 + t18*1024 + t19*68719492368
	t9 = m & fieldMSBMask
	m = m >> fieldMSBBits

//此时，如果震级大于0，则整体值
//大于最大可能256位值。尤其是
//“比最大值大多少倍”。
//
//[HAC]第14.3.4节中给出的算法重复到
//商是零。然而，由于上述原因，我们已经知道
//至少要重复几次，因为这是价值所在
//因此，我们可以简单地将震级乘以
//质数的场表示，并进行一次迭代。通知
//当震级为零时，什么都不会改变，所以我们可以
//在这种情况下跳过此项，但始终运行，无论是否允许
//在固定时间内运行。最终结果将在范围内
//0<=结果<=prime+（2^64-c），因此保证
//大小为1，但它是非规范化的。
	n := t0 + m*977
	f.n[0] = uint32(n & fieldBaseMask)
	n = (n >> fieldBase) + t1 + m*64
	f.n[1] = uint32(n & fieldBaseMask)
	f.n[2] = uint32((n >> fieldBase) + t2)
	f.n[3] = uint32(t3)
	f.n[4] = uint32(t4)
	f.n[5] = uint32(t5)
	f.n[6] = uint32(t6)
	f.n[7] = uint32(t7)
	f.n[8] = uint32(t8)
	f.n[9] = uint32(t9)

	return f
}

//反转查找字段值的模乘性反转。这个
//现有字段值已修改。
//
//返回字段值以支持链接。这将启用如下语法：
//f.inverse（）.mul（f2），使f=f^-1*f2。
func (f *fieldVal) Inverse() *fieldVal {
//费马小定理指出，对于非零数a和素数
//prime p，a^（p-1）=1（mod p）。因为乘法逆是
//a*b=1（mod p），则b=a*a^（p-2）=a^（p-1）=1（mod p）。
//因此，^（p-2）是乘法逆。
//
//为了有效地计算a^（p-2），需要将p-2拆分为
//一系列的正方形和多面体，使
//需要乘法（因为它们比平方运算更昂贵）。
//中间结果也会被保存和重用。
//
//secp256k1 prime-2是2^256-4294968275。
//
//这需要258个场平方和33个场乘法。
	var a2, a3, a4, a10, a11, a21, a42, a45, a63, a1019, a1023 fieldVal
	a2.SquareVal(f)
	a3.Mul2(&a2, f)
	a4.SquareVal(&a2)
	a10.SquareVal(&a4).Mul(&a2)
	a11.Mul2(&a10, f)
	a21.Mul2(&a10, &a11)
	a42.SquareVal(&a21)
	a45.Mul2(&a42, &a3)
	a63.Mul2(&a42, &a21)
	a1019.SquareVal(&a63).Square().Square().Square().Mul(&a11)
	a1023.Mul2(&a1019, &a4)
f.Set(&a63)                                    //f＝a^（2 ^ 6—1）
f.Square().Square().Square().Square().Square() //f = a^(2^11 - 32)
f.Square().Square().Square().Square().Square() //F=A^（2^16-1024）
f.Mul(&a1023)                                  //F=A^（2^16-1）
f.Square().Square().Square().Square().Square() //F=A^（2^21-32）
f.Square().Square().Square().Square().Square() //F=A^（2^26-1024）
f.Mul(&a1023)                                  //F=A^（2^26-1）
f.Square().Square().Square().Square().Square() //F=A^（2^31-32）
f.Square().Square().Square().Square().Square() //F=A^（2^36-1024）
f.Mul(&a1023)                                  //F=A^（2^36-1）
f.Square().Square().Square().Square().Square() //F=A^（2^41-32）
f.Square().Square().Square().Square().Square() //F=A^（2^46-1024）
f.Mul(&a1023)                                  //F=A^（2^46-1）
f.Square().Square().Square().Square().Square() //F=A^（2^51-32）
f.Square().Square().Square().Square().Square() //F=A^（2^56-1024）
f.Mul(&a1023)                                  //F=A^（2^56-1）
f.Square().Square().Square().Square().Square() //F=A^（2^61-32）
f.Square().Square().Square().Square().Square() //F=A^（2^66-1024）
f.Mul(&a1023)                                  //F=A^（2^66-1）
f.Square().Square().Square().Square().Square() //F=A^（2^71-32）
f.Square().Square().Square().Square().Square() //F=A^（2^76-1024）
f.Mul(&a1023)                                  //F=A^（2^76-1）
f.Square().Square().Square().Square().Square() //F=A^（2^81-32）
f.Square().Square().Square().Square().Square() //F=A^（2^86-1024）
f.Mul(&a1023)                                  //F=A^（2^86-1）
f.Square().Square().Square().Square().Square() //F=A^（2^91-32）
f.Square().Square().Square().Square().Square() //F=A^（2^96-1024）
f.Mul(&a1023)                                  //F=A^（2^96-1）
f.Square().Square().Square().Square().Square() //F=A^（2^101-32）
f.Square().Square().Square().Square().Square() //F=A^（2^106-1024）
f.Mul(&a1023)                                  //F=A^（2^106-1）
f.Square().Square().Square().Square().Square() //F=A^（2^111-32）
f.Square().Square().Square().Square().Square() //F=A^（2^116-1024）
f.Mul(&a1023)                                  //F=A^（2^116-1）
f.Square().Square().Square().Square().Square() //F=A^（2^121-32）
f.Square().Square().Square().Square().Square() //F=A^（2^126-1024）
f.Mul(&a1023)                                  //F=A^（2^126-1）
f.Square().Square().Square().Square().Square() //F=A^（2^131-32）
f.Square().Square().Square().Square().Square() //F=A^（2^136-1024）
f.Mul(&a1023)                                  //F=A^（2^136-1）
f.Square().Square().Square().Square().Square() //F=A^（2^141-32）
f.Square().Square().Square().Square().Square() //F=A^（2^146-1024）
f.Mul(&a1023)                                  //F=A^（2^146-1）
f.Square().Square().Square().Square().Square() //F=A^（2^151-32）
f.Square().Square().Square().Square().Square() //F=A^（2^156-1024）
f.Mul(&a1023)                                  //F=A^（2^156-1）
f.Square().Square().Square().Square().Square() //F=A^（2^161-32）
f.Square().Square().Square().Square().Square() //F=A^（2^166-1024）
f.Mul(&a1023)                                  //F=A^（2^166-1）
f.Square().Square().Square().Square().Square() //
f.Square().Square().Square().Square().Square() //F=A^（2^176-1024）
f.Mul(&a1023)                                  //F=A^（2^176-1）
f.Square().Square().Square().Square().Square() //F=A^（2^181-32）
f.Square().Square().Square().Square().Square() //F=A^（2^186-1024）
f.Mul(&a1023)                                  //F=A^（2^186-1）
f.Square().Square().Square().Square().Square() //F=A^（2^191-32）
f.Square().Square().Square().Square().Square() //F=A^（2^196-1024）
f.Mul(&a1023)                                  //F=A^（2^196-1）
f.Square().Square().Square().Square().Square() //F=A^（2^201-32）
f.Square().Square().Square().Square().Square() //F=A^（2^206-1024）
f.Mul(&a1023)                                  //F=A^（2^206-1）
f.Square().Square().Square().Square().Square() //F=A^（2^211-32）
f.Square().Square().Square().Square().Square() //F=A^（2^216-1024）
f.Mul(&a1023)                                  //F=A^（2^216-1）
f.Square().Square().Square().Square().Square() //
f.Square().Square().Square().Square().Square() //F=A^（2^226-1024）
f.Mul(&a1019)                                  //F=A^（2^226-5）
f.Square().Square().Square().Square().Square() //F=A^（2^231-160）
f.Square().Square().Square().Square().Square() //F=A^（2^236-5120）
f.Mul(&a1023)                                  //
f.Square().Square().Square().Square().Square() //F=A^（2^241-131104）
f.Square().Square().Square().Square().Square() //F=A^（2^246-4195328）
f.Mul(&a1023)                                  //
f.Square().Square().Square().Square().Square() //F=A^（2^251-134217760）
f.Square().Square().Square().Square().Square() //F=A^（2^256-4294968320）
return f.Mul(&a45)                             //F=A^（2^256-4294968275）=A^（P-2）
}
