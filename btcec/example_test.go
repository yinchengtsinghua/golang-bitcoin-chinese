
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

package btcec_test

import (
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

//此示例演示如何使用secp256k1私钥对消息进行签名，该私钥
//
func Example_signMessage() {
//解码十六进制编码的私钥。
	pkBytes, err := hex.DecodeString("22a47fa09a223f2aa079edf85a7c2d4f87" +
		"20ee63e502ee2869afab7de234b80c")
	if err != nil {
		fmt.Println(err)
		return
	}
	privKey, pubKey := btcec.PrivKeyFromBytes(btcec.S256(), pkBytes)

//使用私钥签署消息。
	message := "test message"
	messageHash := chainhash.DoubleHashB([]byte(message))
	signature, err := privKey.Sign(messageHash)
	if err != nil {
		fmt.Println(err)
		return
	}

//序列化并显示签名。
	fmt.Printf("Serialized Signature: %x\n", signature.Serialize())

//使用公钥验证消息的签名。
	verified := signature.Verify(messageHash, pubKey)
	fmt.Printf("Signature Verified? %v\n", verified)

//输出：
//序列化签名：304402201008E236FA8CD0F25DF4482DDBB622E88B26EF0BA731719458DE3CCD93805B022032F8EBE514BA5F672466EBA334639261BB3C2F0AB9998037513D1F9E3D6D
//签名已验证？真
}

//此示例演示如何针对公共端验证secp256k1签名
//首先从原始字节解析的键。签名也从
//原始字节。
func Example_verifySignature() {
//解码十六进制编码的序列化公钥。
	pubKeyBytes, err := hex.DecodeString("02a673638cb9587cb68ea08dbef685c" +
		"6f2d2a751a8b3c6f2a7e9a4999e6e4bfaf5")
	if err != nil {
		fmt.Println(err)
		return
	}
	pubKey, err := btcec.ParsePubKey(pubKeyBytes, btcec.S256())
	if err != nil {
		fmt.Println(err)
		return
	}

//解码十六进制编码的序列化签名。
	sigBytes, err := hex.DecodeString("30450220090ebfb3690a0ff115bb1b38b" +
		"8b323a667b7653454f1bccb06d4bbdca42c2079022100ec95778b51e707" +
		"1cb1205f8bde9af6592fc978b0452dafe599481c46d6b2e479")

	if err != nil {
		fmt.Println(err)
		return
	}
	signature, err := btcec.ParseSignature(sigBytes, btcec.S256())
	if err != nil {
		fmt.Println(err)
		return
	}

//使用公钥验证消息的签名。
	message := "test message"
	messageHash := chainhash.DoubleHashB([]byte(message))
	verified := signature.Verify(messageHash, pubKey)
	fmt.Println("Signature Verified?", verified)

//输出：
//签名已验证？真
}

//此示例演示如何加密第一个公钥的消息
//从原始字节解析，然后使用相应的私钥对其进行解密。
func Example_encryptMessage() {
//解码收件人的十六进制编码的pubkey。
	pubKeyBytes, err := hex.DecodeString("04115c42e757b2efb7671c578530ec191a1" +
		"359381e6a71127a9d37c486fd30dae57e76dc58f693bd7e7010358ce6b165e483a29" +
"21010db67ac11b1b51b651953d2") //未压缩的pubkey
	if err != nil {
		fmt.Println(err)
		return
	}
	pubKey, err := btcec.ParsePubKey(pubKeyBytes, btcec.S256())
	if err != nil {
		fmt.Println(err)
		return
	}

//
	message := "test message"
	ciphertext, err := btcec.Encrypt(pubKey, []byte(message))
	if err != nil {
		fmt.Println(err)
		return
	}

//解码十六进制编码的私钥。
	pkBytes, err := hex.DecodeString("a11b0a4e1a132305652ee7a8eb7848f6ad" +
		"5ea381e3ce20a2c086a2e388230811")
	if err != nil {
		fmt.Println(err)
		return
	}
//注意，我们已经有了相应的pubkey
	privKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), pkBytes)

//尝试解密并验证它是否是同一条消息。
	plaintext, err := btcec.Decrypt(privKey, ciphertext)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(plaintext))

//输出：
//测试消息
}

//此示例演示如何使用私钥解密消息，该私钥是
//
func Example_decryptMessage() {
//解码十六进制编码的私钥。
	pkBytes, err := hex.DecodeString("a11b0a4e1a132305652ee7a8eb7848f6ad" +
		"5ea381e3ce20a2c086a2e388230811")
	if err != nil {
		fmt.Println(err)
		return
	}

	privKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), pkBytes)

	ciphertext, err := hex.DecodeString("35f644fbfb208bc71e57684c3c8b437402ca" +
		"002047a2f1b38aa1a8f1d5121778378414f708fe13ebf7b4a7bb74407288c1958969" +
		"00207cf4ac6057406e40f79961c973309a892732ae7a74ee96cd89823913b8b8d650" +
		"a44166dc61ea1c419d47077b748a9c06b8d57af72deb2819d98a9d503efc59fc8307" +
		"d14174f8b83354fac3ff56075162")

//
	plaintext, err := btcec.Decrypt(privKey, ciphertext)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(plaintext))

//输出：
//测试消息
}
