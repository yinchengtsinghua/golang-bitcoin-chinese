
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

package btcec

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
)

//这些常量定义序列化公钥的长度。
const (
	PubKeyBytesLenCompressed   = 33
	PubKeyBytesLenUncompressed = 65
	PubKeyBytesLenHybrid       = 65
)

func isOdd(a *big.Int) bool {
	return a.Bit(0) == 1
}

//减压点对给定曲线上给定X点的点进行减压，并
//要使用的解决方案。
func decompressPoint(curve *KoblitzCurve, x *big.Int, ybit bool) (*big.Int, error) {
//TODO:这可能只对secp256k1有效，因为
//优化。

//y=+-sqrt（x^3+b）
	x3 := new(big.Int).Mul(x, x)
	x3.Mul(x3, x)
	x3.Add(x3, curve.Params().B)
	x3.Mod(x3, curve.Params().P)

//现在计算x^3+b的sqrt mod p
//此代码用于基于tonelli/shanks执行完整的sqrt，
//但这已被中引用的算法所取代。
//https://bitcointalk.org/index.php？主题=162805.msg1712294 msg1712294
	y := new(big.Int).Exp(x3, curve.QPlus1Div4(), curve.Params().P)

	if ybit != isOdd(y) {
		y.Sub(curve.Params().P, y)
	}

//检查y是x^3+b的平方根。
	y2 := new(big.Int).Mul(y, y)
	y2.Mod(y2, curve.Params().P)
	if y2.Cmp(x3) != 0 {
		return nil, fmt.Errorf("invalid square root")
	}

//验证y-coord是否具有预期的奇偶性。
	if ybit != isOdd(y) {
		return nil, fmt.Errorf("ybit doesn't match oddness")
	}

	return y, nil
}

const (
pubkeyCompressed   byte = 0x2 //YYBIT+X COORD
pubkeyUncompressed byte = 0x4 //X坐标+Y坐标
pubkeyHybrid       byte = 0x6 //Y轴位+X轴+Y轴
)

//iscompressedpubkey返回true传递的序列化公钥具有
//以压缩格式编码，否则为false。
func IsCompressedPubKey(pubKey []byte) bool {
//只有当公钥的长度正确并且
//
	return len(pubKey) == PubKeyBytesLenCompressed &&
		(pubKey[0]&^byte(0x1) == pubkeyCompressed)
}

//ParsePubKey将Koblitz曲线的公钥从字节串解析为
//ecdsa.publickey，验证它是否有效。它支持压缩，
//未压缩和混合签名格式。
func ParsePubKey(pubKeyStr []byte, curve *KoblitzCurve) (key *PublicKey, err error) {
	pubkey := PublicKey{}
	pubkey.Curve = curve

	if len(pubKeyStr) == 0 {
		return nil, errors.New("pubkey string is empty")
	}

	format := pubKeyStr[0]
	ybit := (format & 0x1) == 0x1
	format &= ^byte(0x1)

	switch len(pubKeyStr) {
	case PubKeyBytesLenUncompressed:
		if format != pubkeyUncompressed && format != pubkeyHybrid {
			return nil, fmt.Errorf("invalid magic in pubkey str: "+
				"%d", pubKeyStr[0])
		}

		pubkey.X = new(big.Int).SetBytes(pubKeyStr[1:33])
		pubkey.Y = new(big.Int).SetBytes(pubKeyStr[33:])
//混合钥匙有额外的信息，利用它。
		if format == pubkeyHybrid && ybit != isOdd(pubkey.Y) {
			return nil, fmt.Errorf("ybit doesn't match oddness")
		}
	case PubKeyBytesLenCompressed:
//格式为0x2_solution，<x coordinate>
//解决定了曲线的哪个解。
//
		if format != pubkeyCompressed {
			return nil, fmt.Errorf("invalid magic in compressed "+
				"pubkey string: %d", pubKeyStr[0])
		}
		pubkey.X = new(big.Int).SetBytes(pubKeyStr[1:33])
		pubkey.Y, err = decompressPoint(curve, pubkey.X, ybit)
		if err != nil {
			return nil, err
		}
default: //
		return nil, fmt.Errorf("invalid pub key length %d",
			len(pubKeyStr))
	}

	if pubkey.X.Cmp(pubkey.Curve.Params().P) >= 0 {
		return nil, fmt.Errorf("pubkey X parameter is >= to P")
	}
	if pubkey.Y.Cmp(pubkey.Curve.Params().P) >= 0 {
		return nil, fmt.Errorf("pubkey Y parameter is >= to P")
	}
	if !pubkey.Curve.IsOnCurve(pubkey.X, pubkey.Y) {
		return nil, fmt.Errorf("pubkey isn't on secp256k1 curve")
	}
	return &pubkey, nil
}

//publickey是一个ecdsa.publickey，具有以下附加功能：
//以未压缩、压缩和混合格式序列化。
type PublicKey ecdsa.PublicKey

//toecdsa以*ecdsa.public key的形式返回公钥。
func (p *PublicKey) ToECDSA() *ecdsa.PublicKey {
	return (*ecdsa.PublicKey)(p)
}

//序列化未压缩序列化未压缩的65字节中的公钥
//格式。
func (p *PublicKey) SerializeUncompressed() []byte {
	b := make([]byte, 0, PubKeyBytesLenUncompressed)
	b = append(b, pubkeyUncompressed)
	b = paddedAppend(32, b, p.X.Bytes())
	return paddedAppend(32, b, p.Y.Bytes())
}

//SerializeCompressed以33字节的压缩格式序列化公钥。
func (p *PublicKey) SerializeCompressed() []byte {
	b := make([]byte, 0, PubKeyBytesLenCompressed)
	format := pubkeyCompressed
	if isOdd(p.Y) {
		format |= 0x1
	}
	b = append(b, format)
	return paddedAppend(32, b, p.X.Bytes())
}

//SerializeHybrid以65字节的混合格式序列化公钥。
func (p *PublicKey) SerializeHybrid() []byte {
	b := make([]byte, 0, PubKeyBytesLenHybrid)
	format := pubkeyHybrid
	if isOdd(p.Y) {
		format |= 0x1
	}
	b = append(b, format)
	b = paddedAppend(32, b, p.X.Bytes())
	return paddedAppend(32, b, p.Y.Bytes())
}

//IsEqual将此公钥实例与传递的实例进行比较，如果
//两个公钥都是等效的。如果一个公钥
//两者具有相同的X和Y坐标。
func (p *PublicKey) IsEqual(otherPubKey *PublicKey) bool {
	return p.X.Cmp(otherPubKey.X) == 0 &&
		p.Y.Cmp(otherPubKey.Y) == 0
}

//paddedappend将src字节片附加到dst，返回新片。
//如果源的长度小于传递的大小，则从零开始
//字节在附加SRC之前附加到DST切片。
func paddedAppend(size uint, dst, src []byte) []byte {
	for i := 0; i < int(size)-len(src); i++ {
		dst = append(dst, 0)
	}
	return append(dst, src...)
}
