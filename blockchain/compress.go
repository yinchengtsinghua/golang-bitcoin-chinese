
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package blockchain

import (
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
)

//————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//可变长度数量（VLQ）是使用任意数字的编码。
//表示任意大整数的二进制八进制数。方案
//使用最高有效字节（MSB）base-128编码，其中高位
//每个字节指示该字节是否为最后一个字节。此外，
//为了确保没有冗余编码，每隔
//一组7位移出的时间。因此，每个整数可以是
//用一种方式表示，每种表示都代表
//一个整数。
//
//
//表示通常用于指示大小的值。为了
//例如，值0-127用单字节128-16511表示。
//
//
//虽然编码允许任意大的整数，但它是人为的
//为了提高效率，此代码限制为无符号64位整数。
//
//编码示例：
//0＞[0x00 ]
//127->[0x7f]*最大1字节值
//128->[0x80 0x00]
//129->[0x80 0x01]
//255->[0x80 0x7f]
//256->[0x81 0x00]
//16511->[0xFF 0x7F]*最大2字节值
//16512->[0x80 0x80 0x00]
//32895->[0x80 0xFF 0x7F]
//2113663->[0xff 0xff 0x7f]*最大3字节值
//270549119->[0xff 0xff 0xff 0x7f]*最大4字节值
//2^64-1->[0x80 0xfe 0xfe 0xfe 0xfe 0xfe 0xfe 0xfe 0xfe 0x7f]
//
//参考文献：
//https://en.wikipedia.org/wiki/variable-length_数量
//http://www.codecodex.com/wiki/variable-length_integers
//————————————————————————————————————————————————————————————————————————————————————————————————————————————————

//serializevlq返回序列化
//根据所述格式将数字作为可变长度的数量传递
//上面。
func serializeSizeVLQ(n uint64) int {
	size := 1
	for ; n > 0x7f; n = (n >> 7) - 1 {
		size++
	}

	return size
}

//putvlq根据
//到上面描述的格式，并返回编码的字节数
//价值。结果直接放入传递的字节片中，该字节片必须
//至少大到足以处理
//序列化sizevlq函数，否则将死机。
func putVLQ(target []byte, n uint64) int {
	offset := 0
	for ; ; offset++ {
//当另一个字节跟随时，设置高位。
		highBitMask := byte(0x80)
		if offset == 0 {
			highBitMask = 0x00
		}

		target[offset] = byte(n&0x7f) | highBitMask
		if n <= 0x7f {
			break
		}
		n = (n >> 7) - 1
	}

//反转字节，使其为MSB编码。
	for i, j := 0, offset; i < j; i, j = i+1, j-1 {
		target[i], target[j] = target[j], target[i]
	}

	return offset + 1
}

//反序列化evlq根据
//按照上述格式。它还返回字节数
//反序列化。
func deserializeVLQ(serialized []byte) (uint64, int) {
	var n uint64
	var size int
	for _, val := range serialized {
		size++
		n = (n << 7) | uint64(val&0x7f)
		if val&0x80 != 0x80 {
			break
		}
		n++
	}

	return n, size
}

//————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//为了减少存储脚本的大小，特定于域的压缩
//使用算法识别标准脚本并使用
//小于原始脚本的字节数。这里使用的压缩算法是
//从比特币核心获得，所以算法的所有学分都归它。
//
//常规序列化格式为：
//
//<script size or type><script data>
//
//字段类型大小
//脚本大小或类型vlq变量
//脚本数据[]字节变量
//
//
//
//-支付到pubkey散列：（21字节）-<0><20字节pubkey散列>
//-付费脚本哈希：（21字节）-<1><20字节脚本哈希>
//
//
//4，5=未压缩pubkey，位0指定要使用的Y坐标
//**仅支持以0x02、0x03和0x04开头的有效公钥。
//
//任何未被视为上述标准之一的脚本
//使用常规序列化格式对脚本进行编码并对脚本进行编码
//大小是脚本实际大小和特殊字符数的总和
//病例。
//————————————————————————————————————————————————————————————————————————————————————————————————————————————————

//以下常量指定用于标识
//特定于域的压缩脚本编码中的特殊脚本类型。
//
//注：本节特别不使用IOTA，因为这些值是
//已序列化，并且必须对长期存储稳定。
const (
//cstpaytopubkeyhash标识压缩的pay-to-pubkey哈希脚本。
	cstPayToPubKeyHash = 0

//cstpayToScriptHash标识压缩的付薪脚本哈希脚本。
	cstPayToScriptHash = 1

//cstpaytopubkeycomp2标识压缩的pay-to-pubkey脚本
//压缩的pubkey。位0指定要使用的Y坐标
//重新构造完整的未压缩pubkey。
	cstPayToPubKeyComp2 = 2

//cstpaytopubkeycomp3标识压缩的pay-to-pubkey脚本
//压缩的pubkey。位0指定要使用的Y坐标
//重新构造完整的未压缩pubkey。
	cstPayToPubKeyComp3 = 3

//cstpaytopubkeyncomp4标识压缩的pay-to-pubkey脚本
//未压缩的pubkey。位0指定要使用的Y坐标
//重新构造完整的未压缩pubkey。
	cstPayToPubKeyUncomp4 = 4

//cstpaytopubkeyncomp5标识压缩的pay-to-pubkey脚本
//未压缩的pubkey。位0指定要使用的Y坐标
//重新构造完整的未压缩pubkey。
	cstPayToPubKeyUncomp5 = 5

//numspecialscripts是由
//特定于域的脚本压缩算法。
	numSpecialScripts = 6
)

//ispubkeyhash返回传递的公钥脚本是否为
//标准的pay-to-pubkey散列脚本及其支付的pubkey散列
//如果是。
func isPubKeyHash(script []byte) (bool, []byte) {
	if len(script) == 25 && script[0] == txscript.OP_DUP &&
		script[1] == txscript.OP_HASH160 &&
		script[2] == txscript.OP_DATA_20 &&
		script[23] == txscript.OP_EQUALVERIFY &&
		script[24] == txscript.OP_CHECKSIG {

		return true, script[3:23]
	}

	return false, nil
}

//
//标准付薪脚本哈希脚本及其支付的脚本哈希
//如果是。
func isScriptHash(script []byte) (bool, []byte) {
	if len(script) == 23 && script[0] == txscript.OP_HASH160 &&
		script[1] == txscript.OP_DATA_20 &&
		script[22] == txscript.OP_EQUAL {

		return true, script[2:22]
	}

	return false, nil
}

//IsSubkey返回传递的公钥脚本是否为标准脚本
//向有效的压缩或未压缩公共支付的PubKey脚本
//键以及它要支付的序列化pubkey（如果是）。
//
//注意：此函数确保公钥实际上是有效的，因为
//压缩算法需要有效的公钥。它不支持混合动力
//小狗。这意味着即使脚本具有
//pay to pubkey脚本，此函数仅在付款时返回true
//到有效的压缩或未压缩的pubkey。
func isPubKey(script []byte) (bool, []byte) {
//支付到压缩的pubkey脚本。
	if len(script) == 35 && script[0] == txscript.OP_DATA_33 &&
		script[34] == txscript.OP_CHECKSIG && (script[1] == 0x02 ||
		script[1] == 0x03) {

//确保公钥有效。
		serializedPubKey := script[1:34]
		_, err := btcec.ParsePubKey(serializedPubKey, btcec.S256())
		if err == nil {
			return true, serializedPubKey
		}
	}

//支付给未压缩的pubkey脚本。
	if len(script) == 67 && script[0] == txscript.OP_DATA_65 &&
		script[66] == txscript.OP_CHECKSIG && script[1] == 0x04 {

//确保公钥有效。
		serializedPubKey := script[1:66]
		_, err := btcec.ParsePubKey(serializedPubKey, btcec.S256())
		if err == nil {
			return true, serializedPubKey
		}
	}

	return false, nil
}

//compressedscriptsize返回传递的脚本将占用的字节数
//当使用上述特定于域的压缩算法进行编码时。
func compressedScriptSize(pkScript []byte) int {
//支付到PubKey哈希脚本。
	if valid, _ := isPubKeyHash(pkScript); valid {
		return 21
	}

//付费脚本哈希脚本。
	if valid, _ := isScriptHash(pkScript); valid {
		return 21
	}

//支付到Pubkey（压缩或未压缩）脚本。
	if valid, _ := isPubKey(pkScript); valid {
		return 33
	}

//当以上所有特殊情况都不适用时，按原样对脚本进行编码
//前面是它的大小和特殊情况的数量之和
//编码为可变长度的数量。
	return serializeSizeVLQ(uint64(len(pkScript)+numSpecialScripts)) +
		len(pkScript)
}

//decodecompressedscriptsize将传递的序列化字节视为压缩的
//脚本，可能后跟其他数据，并返回它的字节数
//考虑到脚本大小的特殊编码，
//上面描述的特定于域的压缩算法。
func decodeCompressedScriptSize(serialized []byte) int {
	scriptSize, bytesRead := deserializeVLQ(serialized)
	if bytesRead == 0 {
		return 0
	}

	switch scriptSize {
	case cstPayToPubKeyHash:
		return 21

	case cstPayToScriptHash:
		return 21

	case cstPayToPubKeyComp2, cstPayToPubKeyComp3, cstPayToPubKeyUncomp4,
		cstPayToPubKeyUncomp5:
		return 33
	}

	scriptSize -= numSpecialScripts
	scriptSize += uint64(bytesRead)
	return int(scriptSize)
}

//putcompressedscript根据域压缩传递的脚本
//上面描述的特定压缩算法直接进入
//目标字节片。目标字节片必须至少大到足以
//处理compressedscriptsize函数返回的字节数或
//它会恐慌。
func putCompressedScript(target, pkScript []byte) int {
//支付到PubKey哈希脚本。
	if valid, hash := isPubKeyHash(pkScript); valid {
		target[0] = cstPayToPubKeyHash
		copy(target[1:21], hash)
		return 21
	}

//付费脚本哈希脚本。
	if valid, hash := isScriptHash(pkScript); valid {
		target[0] = cstPayToScriptHash
		copy(target[1:21], hash)
		return 21
	}

//支付到Pubkey（压缩或未压缩）脚本。
	if valid, serializedPubKey := isPubKey(pkScript); valid {
		pubKeyFormat := serializedPubKey[0]
		switch pubKeyFormat {
		case 0x02, 0x03:
			target[0] = pubKeyFormat
			copy(target[1:33], serializedPubKey[1:33])
			return 33
		case 0x04:
//将序列化pubkey的奇异性编码到
//压缩的脚本类型。
			target[0] = pubKeyFormat | (serializedPubKey[64] & 0x01)
			copy(target[1:33], serializedPubKey[1:33])
			return 33
		}
	}

//当上述特殊情况均不适用时，对未修改的
//前接大小和特殊字符数之和的脚本
//编码为可变长度数量的事例。
	encodedSize := uint64(len(pkScript) + numSpecialScripts)
	vlqSizeLen := putVLQ(target, encodedSize)
	copy(target[vlqSizeLen:], pkScript)
	return vlqSizeLen + len(pkScript)
}

//解压缩脚本返回通过解压缩
//根据特定于域的压缩传递压缩脚本
//上述算法。
//
//注意：脚本参数必须已经被证明足够长
//包含decodecompressedscriptsize或它返回的字节数
//会恐慌。这是可以接受的，因为它只是一个内部函数。
func decompressScript(compressedPkScript []byte) []byte {
//实际上，不会使用零长度或
//nil脚本，因为nil脚本编码包括长度
//下面的代码假定长度存在，因此如果
//函数最终在
//未来。
	if len(compressedPkScript) == 0 {
		return nil
	}

//解码脚本大小并检查是否存在特殊情况。
	encodedScriptSize, bytesRead := deserializeVLQ(compressedPkScript)
	switch encodedScriptSize {
//支付到PubKey哈希脚本。生成的脚本是：
//<op_dup><op_hash160><20 byte hash><op_equalverify><op_checksig>
	case cstPayToPubKeyHash:
		pkScript := make([]byte, 25)
		pkScript[0] = txscript.OP_DUP
		pkScript[1] = txscript.OP_HASH160
		pkScript[2] = txscript.OP_DATA_20
		copy(pkScript[3:], compressedPkScript[bytesRead:bytesRead+20])
		pkScript[23] = txscript.OP_EQUALVERIFY
		pkScript[24] = txscript.OP_CHECKSIG
		return pkScript

//付费脚本哈希脚本。生成的脚本是：
//<op_hash160><20 byte script hash><op_equal>
	case cstPayToScriptHash:
		pkScript := make([]byte, 23)
		pkScript[0] = txscript.OP_HASH160
		pkScript[1] = txscript.OP_DATA_20
		copy(pkScript[2:], compressedPkScript[bytesRead:bytesRead+20])
		pkScript[22] = txscript.OP_EQUAL
		return pkScript

//支付到压缩的pubkey脚本。生成的脚本是：
//<op_data_33><33 byte compressed pubkey><op_checksig>
	case cstPayToPubKeyComp2, cstPayToPubKeyComp3:
		pkScript := make([]byte, 35)
		pkScript[0] = txscript.OP_DATA_33
		pkScript[1] = byte(encodedScriptSize)
		copy(pkScript[2:], compressedPkScript[bytesRead:bytesRead+32])
		pkScript[34] = txscript.OP_CHECKSIG
		return pkScript

//支付给未压缩的pubkey脚本。生成的脚本是：
//<op_data_65><65 byte uncompressed pubkey><op_checksig>
	case cstPayToPubKeyUncomp4, cstPayToPubKeyUncomp5:
//将前导字节更改为适当的压缩pubkey
//标识符（0x02或0x03），以便将其解码为
//压缩的pubkey。这真的不应该失败，因为
//编码确保它在压缩到该类型之前是有效的。
		compressedKey := make([]byte, 33)
		compressedKey[0] = byte(encodedScriptSize - 2)
		copy(compressedKey[1:], compressedPkScript[1:])
		key, err := btcec.ParsePubKey(compressedKey, btcec.S256())
		if err != nil {
			return nil
		}

		pkScript := make([]byte, 67)
		pkScript[0] = txscript.OP_DATA_65
		copy(pkScript[1:], key.SerializeUncompressed())
		pkScript[66] = txscript.OP_CHECKSIG
		return pkScript
	}

//当所有特殊情况都不适用时，脚本使用
//常规格式，因此将脚本大小减少
//特殊情况下，返回未修改的脚本。
	scriptSize := int(encodedScriptSize - numSpecialScripts)
	pkScript := make([]byte, scriptSize)
	copy(pkScript, compressedPkScript[bytesRead:bytesRead+scriptSize])
	return pkScript
}

//————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//为了减少存储量的大小，特定于域的压缩
//使用的算法依赖于通常在
//金额的结尾。这里使用的压缩算法是从
//比特币核心，所以算法的所有学分都归它。
//
//虽然这只是将一个uint64换成另一个uint64，但结果值
//对于典型的数量，其数量级要小得多，这会导致更少的字节。
//当编码为可变长度数量时。例如，考虑金额
//0.1 Btc，即10000000 Satoshi。将10000000编码为VLQ需要
//将8的压缩值编码为VLQ时，4个字节只需要1个字节。
//
//实际上，压缩是通过将值拆分为
//指数在[0-9]范围内，数字在[1-9]范围内，如果可能，
//并以可解码的方式对其进行编码。更具体地说，
//编码如下：
//0是0
//-求指数e，作为10的最大幂，平均地除以
//最大值为9
//-当e<9时，最后一位不能为0，所以将其存储为d，并将其删除
//将值除以10（调用结果n）。因此，编码值为：
//1+10*（9*N+D-1）+E
//-当e==9时，唯一知道的是数量不是0。编码值
//因此：
//1+10*（n-1）+e==10+10*（n-1）
//
//编码示例：
//（括号中的数字是序列化为VLQ时的字节数）
//0（1）->0（1）*0.00000000 BTC
//1000（2）->4（1）*0.00001000 BTC
//10000（2）->5（1）*0.00010000 BTC
//12345678（4）->1111111 01（4）*0.12345678 BTC
//50000000（4）->47（1）*0.50000000 BTC
//100000000（4）->9（1）*1.00000000 BTC
//500000000（5）->49（1）*5.00000000 BTC
//100000000（5）->10（1）*10.00000000 BTC
//————————————————————————————————————————————————————————————————————————————————————————————————————————————————

//compressTxOutAmount根据域压缩传递的金额
//上面描述的特定压缩算法。
func compressTxOutAmount(amount uint64) uint64 {
//如果是零，就不需要做任何工作。
	if amount == 0 {
		return 0
	}

//找到最大的10次方（最大9次方），将
//价值。
	exponent := uint64(0)
	for amount%10 == 0 && exponent < 9 {
		amount /= 10
		exponent++
	}

//指数小于9的压缩结果为：
//1+10*（9*N+D-1）+E
	if exponent < 9 {
		lastDigit := amount % 10
		amount /= 10
		return 1 + 10*(9*amount+lastDigit-1) + exponent
	}

//指数9的压缩结果是：
//1+10*（n-1）+e==10+10*（n-1）
	return 10 + 10*(amount-1)
}

//compresstxoutamount返回传递的压缩文件的原始数量
//根据特定于域的压缩算法表示的数量
//如上所述。
func decompressTxOutAmount(amount uint64) uint64 {
//如果是零，就不需要做任何工作。
	if amount == 0 {
		return 0
	}

//减压量为以下两个方程之一：
//X=1+10*（9*N+D-1）+E
//x=1+10*（n-1）+9
	amount--

//减压量现在是以下两个方程之一：
//X=10*（9*N+D-1）+E
//x=10*（n-1）+9
	exponent := amount % 10
	amount /= 10

//减压量现在是以下两个方程之一：
//x=9*n+d-1式中e<9
//x=n-1式中e=9
	n := uint64(0)
	if exponent < 9 {
		lastDigit := amount%9 + 1
		amount /= 9
		n = amount*10 + lastDigit
	} else {
		n = amount + 1
	}

//应用指数。
	for ; exponent > 0; exponent-- {
		n *= 10
	}

	return n
}

//————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//压缩的事务输出由一个数量和一个公钥脚本组成
//这两种压缩都使用了之前特定于域的压缩算法
//描述。
//
//
//
//<compressed amount><compressed script>
//
//字段类型大小
//压缩量VLQ变量
//压缩脚本[]字节变量
//————————————————————————————————————————————————————————————————————————————————————————————————————————————————

//compressedtxoutsize返回传递的事务输出的字节数
//当使用上面描述的格式编码时，字段将采用。
func compressedTxOutSize(amount uint64, pkScript []byte) int {
	return serializeSizeVLQ(compressTxOutAmount(amount)) +
		compressedScriptSize(pkScript)
}

//putcompressedtxout根据其
//特定于域的压缩算法，并将其直接编码到
//以上述格式传递了目标字节片。目标字节
//切片必须至少大到足以处理
//compressedtxoutsize函数，否则将死机。
func putCompressedTxOut(target []byte, amount uint64, pkScript []byte) int {
	offset := putVLQ(target, compressTxOutAmount(amount))
	offset += putCompressedScript(target[offset:], pkScript)
	return offset
}

//decodecompressedtxout解码传递的压缩txout，可能后面跟着
//通过其他数据，转换为未压缩的数量和脚本，并返回它们
//它们在解压之前所占的字节数。
func decodeCompressedTxOut(serialized []byte) (uint64, []byte, int, error) {
//反序列化压缩量并确保有字节
//保留用于压缩脚本。
	compressedAmount, bytesRead := deserializeVLQ(serialized)
	if bytesRead >= len(serialized) {
		return 0, nil, bytesRead, errDeserialize("unexpected end of " +
			"data after compressed amount")
	}

//解码压缩的脚本大小并确保有足够的字节
//留在切片中。
	scriptSize := decodeCompressedScriptSize(serialized[bytesRead:])
	if len(serialized[bytesRead:]) < scriptSize {
		return 0, nil, bytesRead, errDeserialize("unexpected end of " +
			"data after script size")
	}

//解压缩并返回金额和脚本。
	amount := decompressTxOutAmount(compressedAmount)
	script := decompressScript(serialized[bytesRead : bytesRead+scriptSize])
	return amount, script, bytesRead + scriptSize, nil
}
