
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

package btcec

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"io"
)

var (
//当消息身份验证检查（MAC）失败时，出现errInvalidMac
//在解密过程中。这是因为私钥无效或
//密文已损坏。
	ErrInvalidMAC = errors.New("invalid mac hash")

//errInputToShort发生在解密的输入密文
//函数的长度小于134字节。
	errInputTooShort = errors.New("ciphertext too short")

//当加密的前两个字节
//文本不是0x02ca（=712=secp256k1，来自openssl）。
	errUnsupportedCurve = errors.New("unsupported curve")

	errInvalidXLength = errors.New("invalid X length, must be 32")
	errInvalidYLength = errors.New("invalid Y length, must be 32")
	errInvalidPadding = errors.New("invalid PKCS#7 padding")

//0x02CA＝714
	ciphCurveBytes = [2]byte{0x02, 0xCA}
//0x20＝32
	ciphCoordLength = [2]byte{0x00, 0x20}
)

//GenerateSharedSecret基于私钥和
//使用Diffie-Hellman密钥交换（ECDH）的公钥（RFC 4753）。
//RFC5903第9节规定，我们只应返回X。
func GenerateSharedSecret(privkey *PrivateKey, pubkey *PublicKey) []byte {
	x, _ := pubkey.Curve.ScalarMult(pubkey.X, pubkey.Y, privkey.D.Bytes())
	return x.Bytes()
}

//
//生成一个私钥（其pubkey也在输出中）。唯一
//支持曲线为secp256k1。它将所有内容编码为的“结构”
//是：
//
//结构{
////用于aes-256-cbc的初始化向量
//IV [ 16 ]字节
////公钥：曲线（2）+len_pubkeyx（2）+pubkeyx+
////PubKeyy的Len_（2）+PubKeyy（曲线=714）
//公钥[70]字节
////密文
//数据[]字节
////hmac-sha-256消息身份验证码
//HMAC〔32〕字节
//}
//
//主要目的是确保字节与PyElliptic的兼容性。此外，参考
//参见ANSI X9.63第5.8.1节，了解此格式的基本原理。
func Encrypt(pubkey *PublicKey, in []byte) ([]byte, error) {
	ephemeral, err := NewPrivateKey(S256())
	if err != nil {
		return nil, err
	}
	ecdhKey := GenerateSharedSecret(ephemeral, pubkey)
	derivedKey := sha512.Sum512(ecdhKey)
	keyE := derivedKey[:32]
	keyM := derivedKey[32:]

	paddedIn := addPKCSPadding(in)
//IV+曲线参数/X/Y+填充纯文本/密文+HMAC-256
	out := make([]byte, aes.BlockSize+70+len(paddedIn)+sha256.Size)
	iv := out[:aes.BlockSize]
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
//开始写入公钥
	pb := ephemeral.PubKey().SerializeUncompressed()
	offset := aes.BlockSize

//曲线和X长度
	copy(out[offset:offset+4], append(ciphCurveBytes[:], ciphCoordLength[:]...))
	offset += 4
//X
	copy(out[offset:offset+32], pb[1:33])
	offset += 32
//Y长
	copy(out[offset:offset+2], ciphCoordLength[:])
	offset += 2
//Y
	copy(out[offset:offset+32], pb[33:])
	offset += 32

//开始加密
	block, err := aes.NewCipher(keyE)
	if err != nil {
		return nil, err
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(out[offset:len(out)-sha256.Size], paddedIn)

//启动HMAC-SHA-256
	hm := hmac.New(sha256.New, keyM)
hm.Write(out[:len(out)-sha256.Size])          //一切都是散列的
copy(out[len(out)-sha256.Size:], hm.Sum(nil)) //写入校验和

	return out, nil
}

//decrypt解密使用encrypt函数加密的数据。
func Decrypt(priv *PrivateKey, in []byte) ([]byte, error) {
//IV+曲线参数/X/Y+1块+HMAC-256
	if len(in) < aes.BlockSize+70+aes.BlockSize+sha256.Size {
		return nil, errInputTooShort
	}

//阅读四
	iv := in[:aes.BlockSize]
	offset := aes.BlockSize

//开始读取pubkey
	if !bytes.Equal(in[offset:offset+2], ciphCurveBytes[:]) {
		return nil, errUnsupportedCurve
	}
	offset += 2

	if !bytes.Equal(in[offset:offset+2], ciphCoordLength[:]) {
		return nil, errInvalidXLength
	}
	offset += 2

	xBytes := in[offset : offset+32]
	offset += 32

	if !bytes.Equal(in[offset:offset+2], ciphCoordLength[:]) {
		return nil, errInvalidYLength
	}
	offset += 2

	yBytes := in[offset : offset+32]
	offset += 32

	pb := make([]byte, 65)
pb[0] = byte(0x04) //未压缩的
	copy(pb[1:33], xBytes)
	copy(pb[33:], yBytes)
//检查（x，y）是否位于曲线上，如果位于曲线上，则创建pubkey
	pubkey, err := ParsePubKey(pb, S256())
	if err != nil {
		return nil, err
	}

//检查密码文本长度
	if (len(in)-aes.BlockSize-offset-sha256.Size)%aes.BlockSize != 0 {
return nil, errInvalidPadding //未填充到16字节
	}

//读HMAC
	messageMAC := in[len(in)-sha256.Size:]

//生成共享机密
	ecdhKey := GenerateSharedSecret(priv, pubkey)
	derivedKey := sha512.Sum512(ecdhKey)
	keyE := derivedKey[:32]
	keyM := derivedKey[32:]

//验证MAC
	hm := hmac.New(sha256.New, keyM)
hm.Write(in[:len(in)-sha256.Size]) //一切都是散列的
	expectedMAC := hm.Sum(nil)
	if !hmac.Equal(messageMAC, expectedMAC) {
		return nil, ErrInvalidMAC
	}

//开始解密
	block, err := aes.NewCipher(keyE)
	if err != nil {
		return nil, err
	}
	mode := cipher.NewCBCDecrypter(block, iv)
//
	plaintext := make([]byte, len(in)-offset-sha256.Size)
	mode.CryptBlocks(plaintext, in[offset:len(in)-sha256.Size])

	return removePKCSPadding(plaintext)
}

//实现块大小为16（aes块大小）的pkcs 7填充。

//addpkcspadding向数据块添加填充
func addPKCSPadding(src []byte) []byte {
	padding := aes.BlockSize - len(src)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

//
func removePKCSPadding(src []byte) ([]byte, error) {
	length := len(src)
	padLength := int(src[length-1])
	if padLength > aes.BlockSize || length < aes.BlockSize {
		return nil, errInvalidPadding
	}

	return src[:length-padLength], nil
}
