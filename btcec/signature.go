
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

package btcec

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"math/big"
)

//CanonicalPadding返回的错误。
var (
	errNegativeValue          = errors.New("value may be interpreted as negative")
	errExcessivelyPaddedValue = errors.New("value is excessively padded")
)

//签名是表示ECDSA签名的类型。
type Signature struct {
	R *big.Int
	S *big.Int
}

var (
//在测试nonce的正确性时在rfc6979实现中使用
	one = big.NewInt(1)

//OneInitializer用于用字节0x01填充字节片。它是提供的
//这里是为了避免多次创建它。
	oneInitializer = []byte{0x01}
)

//serialize以更严格的der格式返回ECDSA签名。注释
//返回的序列化字节不包括附加的哈希类型
//用于比特币签名脚本。
//
//编码/ASN1已损坏，因此我们手动滚动此输出：
//
//0x30<length>0x02<length r>r 0x02<length s>s
func (sig *Signature) Serialize() []byte {
//低韧性破胶剂
	sigS := sig.S
	if sigS.Cmp(S256().halfOrder) == 1 {
		sigS = new(big.Int).Sub(S256().N, sigS)
	}
//确保r和s值的编码字节是规范的，并且
//因此适用于DER编码。
	rb := canonicalizeInt(sig.R)
	sb := canonicalizeInt(sigS)

//返回的签名的总长度为每个magic和
//长度（共6个），加上R和S的长度
	length := 6 + len(rb) + len(sb)
	b := make([]byte, length)

	b[0] = 0x30
	b[1] = byte(length - 2)
	b[2] = 0x02
	b[3] = byte(len(rb))
	offset := copy(b[4:], rb) + 4
	b[offset] = 0x02
	b[offset+1] = byte(len(sb))
	copy(b[offset+2:], sb)
	return b
}

//verify调用ecdsa。verify使用public验证哈希的签名
//关键。如果签名有效，则返回true，否则返回false。
func (sig *Signature) Verify(hash []byte, pubKey *PublicKey) bool {
	return ecdsa.Verify(pubKey.ToECDSA(), hash, sig.R, sig.S)
}

//IsEqual将此签名实例与传递的实例进行比较，返回true
//如果两个签名相同。如果
//它们对于r和s都有相同的标量值。
func (sig *Signature) IsEqual(otherSig *Signature) bool {
	return sig.R.Cmp(otherSig.R) == 0 &&
		sig.S.Cmp(otherSig.S) == 0
}

//minsiglen是der编码签名的最小长度，并且是
//
//0x30+<1-字节>+0x02+0x01+<byte>+0x2+0x01+<byte>
const minSigLen = 8

func parseSig(sigStr []byte, curve elliptic.Curve, der bool) (*Signature, error) {
//最初，此代码使用编码/ASN1来分析
//签名，但在这种方法中发现了许多问题。
//尽管签名被存储为der，但区别在于
//在Go关于Bignum（和他们有签名）的想法之间并不一致。
//
//走1.1路。最后，将代码重写为显式
//了解以下格式：
//0x30 <length of whole message> <0x02> <length of R> <R> 0x2
//<length of s><s>。

	signature := &Signature{}

	if len(sigStr) < minSigLen {
		return nil, errors.New("malformed signature: too short")
	}
//0x30
	index := 0
	if sigStr[index] != 0x30 {
		return nil, errors.New("malformed signature: no header magic")
	}
	index++
//剩余消息的长度
	siglen := sigStr[index]
	index++

//siglen应小于整个消息且大于
//最小邮件大小。
	if int(siglen+2) > len(sigStr) || int(siglen+2) < minSigLen {
		return nil, errors.New("malformed signature: bad length")
	}
//修剪我们正在处理的切片，这样我们只关注重要的内容。
	sigStr = sigStr[:siglen+2]

//0x02
	if sigStr[index] != 0x02 {
		return nil,
			errors.New("malformed signature: no 1st int marker")
	}
	index++

//签名长度r。
	rLen := int(sigStr[index])
//必须为正，必须能够适应另一个0x2，<len><s>
//因此-- 3。我们假设长度必须至少为一个字节。
	index++
	if rLen <= 0 || rLen > len(sigStr)-index-3 {
		return nil, errors.New("malformed signature: bogus R length")
	}

//然后R本身。
	rBytes := sigStr[index : index+rLen]
	if der {
		switch err := canonicalPadding(rBytes); err {
		case errNegativeValue:
			return nil, errors.New("signature R is negative")
		case errExcessivelyPaddedValue:
			return nil, errors.New("signature R is excessively padded")
		}
	}
	signature.R = new(big.Int).SetBytes(rBytes)
	index += rLen
//0x02。长度已在上一个if中签入。
	if sigStr[index] != 0x02 {
		return nil, errors.New("malformed signature: no 2nd int marker")
	}
	index++

//签名长度S。
	sLen := int(sigStr[index])
	index++
//s应该是字符串的其余部分。
	if sLen <= 0 || sLen > len(sigStr)-index {
		return nil, errors.New("malformed signature: bogus S length")
	}

//然后S本身。
	sBytes := sigStr[index : index+sLen]
	if der {
		switch err := canonicalPadding(sBytes); err {
		case errNegativeValue:
			return nil, errors.New("signature S is negative")
		case errExcessivelyPaddedValue:
			return nil, errors.New("signature S is excessively padded")
		}
	}
	signature.S = new(big.Int).SetBytes(sBytes)
	index += sLen

//健全性检查长度分析
	if index != len(sigStr) {
		return nil, fmt.Errorf("malformed signature: bad final length %v != %v",
			index, len(sigStr))
	}

//验证也会检查这个，但是我们可以更确定我们分析了
//如果我们也在这里验证的话是正确的。
//fwiw ECDSA规范规定R和S必须为1，N-1
//但是crypto/ecdsa只检查签名！= 0。照镜子。
	if signature.R.Sign() != 1 {
		return nil, errors.New("signature R isn't 1 or more")
	}
	if signature.S.Sign() != 1 {
		return nil, errors.New("signature S isn't 1 or more")
	}
	if signature.R.Cmp(curve.Params().N) >= 0 {
		return nil, errors.New("signature R is >= curve.N")
	}
	if signature.S.Cmp(curve.Params().N) >= 0 {
		return nil, errors.New("signature S is >= curve.N")
	}

	return signature, nil
}

//parseSignature为曲线类型“curve”解析BER格式的签名
//进入签名类型，执行一些基本的健全性检查。如果解析
//根据需要更严格的der格式，使用parsedersignature。
func ParseSignature(sigStr []byte, curve elliptic.Curve) (*Signature, error) {
	return parseSig(sigStr, curve, false)
}

//ParsederSignature以der格式分析曲线类型的签名
//将“曲线”转换为签名类型。如果按照不太严格的
//需要BER格式，请使用ParseSignature。
func ParseDERSignature(sigStr []byte, curve elliptic.Curve) (*Signature, error) {
	return parseSig(sigStr, curve, true)
}

//CANNORIZITEN返回已调整的大整数的字节
//必须确保big endian编码的整数不可能
//误译为负数。这可能发生在
//设置了有效位，因此在本例中它由前导零字节填充。
//此外，当传递
//数值为0。这对于DER编码是必需的。
func canonicalizeInt(val *big.Int) []byte {
	b := val.Bytes()
	if len(b) == 0 {
		b = []byte{0x00}
	}
	if b[0]&0x80 != 0 {
		paddedBytes := make([]byte, len(b)+1)
		copy(paddedBytes[1:], b)
		b = paddedBytes
	}
	return b
}

//canonicalpadding检查big endian编码的整数是否可以
//可能被错误地解释为负数（即使openssl
//将所有数字视为无符号），或者如果有任何不必要的
//前导零填充。
func canonicalPadding(b []byte) error {
	switch {
	case b[0]&0x80 == 0x80:
		return errNegativeValue
	case len(b) > 1 && b[0] == 0x00 && b[1]&0x80 != 0x80:
		return errExcessivelyPaddedValue
	default:
		return nil
	}
}

//hashToInt将哈希值转换为整数。有一些分歧
//关于如何做到这一点。[国家安全局]认为这是显而易见的
//方式，但[secg]将哈希截断为曲线顺序的位长度
//第一。我们遵循[secg]是因为OpenSSL就是这样做的。此外，
//如果散列太大，openssl right会从数字中移动多余的位。
//我们也照镜子。
//这是从crypto/ecdsa借来的。
func hashToInt(hash []byte, c elliptic.Curve) *big.Int {
	orderBits := c.Params().N.BitLen()
	orderBytes := (orderBits + 7) / 8
	if len(hash) > orderBytes {
		hash = hash[:orderBytes]
	}

	ret := new(big.Int).SetBytes(hash)
	excess := len(hash)*8 - orderBits
	if excess > 0 {
		ret.Rsh(ret, uint(excess))
	}
	return ret
}

//recoverkefromsignature从上的签名“sig”中恢复公钥
//
//第1节2.0版，第47-48页（PDF中的53和54）。这将执行详细信息
//在步骤1的内环中。提供的计数器实际上是j参数
//循环*2-在j的第一次迭代中，我们做r例，否则-r
//步骤1.6中的情况。此计数器用于比特币压缩签名
//格式，因此我们匹配比特币的行为。
func recoverKeyFromSignature(curve *KoblitzCurve, sig *Signature, msg []byte,
	iter int, doChecks bool) (*PublicKey, error) {
//1.1 x=（n*i）+r
	Rx := new(big.Int).Mul(curve.Params().N,
		new(big.Int).SetInt64(int64(iter/2)))
	Rx.Add(Rx, sig.R)
	if Rx.Cmp(curve.Params().P) != -1 {
		return nil, errors.New("calculated Rx is larger than curve P")
	}

//将02<Rx>转换为R点（步骤1.2和1.3）。如果我们在奇数区
//然后用-r迭代1.6，所以我们计算另一个
//
	Ry, err := decompressPoint(curve, Rx, iter%2 == 1)
	if err != nil {
		return nil, err
	}

//1.4检查n*r为无穷远点
	if doChecks {
		nRx, nRy := curve.ScalarMult(Rx, Ry, curve.Params().N.Bytes())
		if nRx.Sign() != 0 || nRy.Sign() != 0 {
			return nil, errors.New("n*R does not equal the point at infinity")
		}
	}

//1.5使用与ECDSA相同的算法从消息中计算e
//签名计算。
	e := hashToInt(msg, curve)

//步骤1.6-1：
//
//R的倒数（从签名）。然后我们把它们加起来计算
//q= r^－1（SR EG）
	invr := new(big.Int).ModInverse(sig.R, curve.Params().N)

//第一学期。
	invrS := new(big.Int).Mul(invr, sig.S)
	invrS.Mod(invrS, curve.Params().N)
	sRx, sRy := curve.ScalarMult(Rx, Ry, invrS.Bytes())

//第二学期。
	e.Neg(e)
	e.Mod(e, curve.Params().N)
	e.Mul(e, invr)
	e.Mod(e, curve.Params().N)
	minuseGx, minuseGy := curve.ScalarBaseMult(e.Bytes())

//托多：如果我们做了一个mult和add-in-one，这会更快。
//防止雅可比变换的步骤。
	Qx, Qy := curve.Add(sRx, sRy, minuseGx, minuseGy)

	return &PublicKey{
		Curve: curve,
		X:     Qx,
		Y:     Qy,
	}, nil
}

//signcompact使用给定的
//给定Koblitz曲线上的私钥。iscompressed参数应该
//用于详细说明给定签名是否应引用压缩的
//是否公钥。如果成功，压缩签名的字节将是
//返回格式：
//<（27+公钥解决方案的字节数）+4 if compressed><padded byte s for signature r><padded byte s for signature s>
//其中r和s参数被填充到曲线的位长度。
func SignCompact(curve *KoblitzCurve, key *PrivateKey,
	hash []byte, isCompressedKey bool) ([]byte, error) {
	sig, err := key.Sign(hash)
	if err != nil {
		return nil, err
	}

//比特币在这里检查r和s的位长。ECDSA签名
//算法返回r和s mod n，因此它们将是
//曲线，因此大小正确。
	for i := 0; i < (curve.H+1)*2; i++ {
		pk, err := recoverKeyFromSignature(curve, sig, hash, i, true)
		if err == nil && pk.X.Cmp(key.X) == 0 && pk.Y.Cmp(key.Y) == 0 {
			result := make([]byte, 1, 2*curve.byteSize+1)
			result[0] = 27 + byte(i)
			if isCompressedKey {
				result[0] += 4
			}
//不确定这需要四舍五入，但这样做更安全。
			curvelen := (curve.BitSize + 7) / 8

//如果需要，将R和S垫到曲线。
			bytelen := (sig.R.BitLen() + 7) / 8
			if bytelen < curvelen {
				result = append(result,
					make([]byte, curvelen-bytelen)...)
			}
			result = append(result, sig.R.Bytes()...)

			bytelen = (sig.S.BitLen() + 7) / 8
			if bytelen < curvelen {
				result = append(result,
					make([]byte, curvelen-bytelen)...)
			}
			result = append(result, sig.S.Bytes()...)

			return result, nil
		}
	}

	return nil, errors.New("no valid solution for pubkey found")
}

//recovercompact验证“hash”的压缩签名“signature”
//“曲线”中的Koblitz曲线。如果签名匹配，则恢复的公共
//如果原始密钥被压缩，则返回密钥和布尔值。
//否则将返回错误。
func RecoverCompact(curve *KoblitzCurve, signature,
	hash []byte) (*PublicKey, bool, error) {
	bitlen := (curve.BitSize + 7) / 8
	if len(signature) != 1+bitlen*2 {
		return nil, false, errors.New("invalid compact signature size")
	}

	iteration := int((signature[0] - 27) & ^byte(4))

//格式为<header byte><bitlen r><bitlen s>
	sig := &Signature{
		R: new(big.Int).SetBytes(signature[1 : bitlen+1]),
		S: new(big.Int).SetBytes(signature[bitlen+1:]),
	}
//这里使用的迭代是编码的。
	key, err := recoverKeyFromSignature(curve, sig, hash, iteration, false)
	if err != nil {
		return nil, false, err
	}

	return key, ((signature[0] - 27) & 4) == 4, nil
}

//signrfc6979根据rfc 6979和bip 62生成确定性ECDSA签名。
func signRFC6979(privateKey *PrivateKey, hash []byte) (*Signature, error) {

	privkey := privateKey.ToECDSA()
	N := S256().N
	halfOrder := S256().halfOrder
	k := nonceRFC6979(privkey.D, hash)
	inv := new(big.Int).ModInverse(k, N)
	r, _ := privkey.Curve.ScalarBaseMult(k.Bytes())
	r.Mod(r, N)

	if r.Sign() == 0 {
		return nil, errors.New("calculated R is zero")
	}

	e := hashToInt(hash, privkey.Curve)
	s := new(big.Int).Mul(privkey.D, r)
	s.Add(s, e)
	s.Mul(s, inv)
	s.Mod(s, N)

	if s.Cmp(halfOrder) == 1 {
		s.Sub(N, s)
	}
	if s.Sign() == 0 {
		return nil, errors.New("calculated S is zero")
	}
	return &Signature{R: r, S: s}, nil
}

//nonce rfc 6979根据RFC6979确定地生成一个ecdsa nonce（`k`）。
//它以一个32字节的哈希作为输入，并返回32字节的nonce以用于ECDSA算法。
func nonceRFC6979(privkey *big.Int, hash []byte) *big.Int {

	curve := S256()
	q := curve.Params().N
	x := privkey
	alg := sha256.New

	qlen := q.BitLen()
	holen := alg().Size()
	rolen := (qlen + 7) >> 3
	bx := append(int2octets(x, rolen), bits2octets(hash, curve, rolen)...)

//步骤B
	v := bytes.Repeat(oneInitializer, holen)

//步骤c（归零所有分配的内存）
	k := make([]byte, holen)

//步骤D
	k = mac(alg, k, append(append(v, 0x00), bx...))

//步骤e
	v = mac(alg, k, v)

//步骤f
	k = mac(alg, k, append(append(v, 0x01), bx...))

//步骤G
	v = mac(alg, k, v)

//步骤H
	for {
//步骤H1
		var t []byte

//步骤H2
		for len(t)*8 < qlen {
			v = mac(alg, k, v)
			t = append(t, v...)
		}

//步骤H3
		secret := hashToInt(t, curve)
		if secret.Cmp(one) >= 0 && secret.Cmp(q) < 0 {
			return secret
		}
		k = mac(alg, k, append(v, 0x00))
		v = mac(alg, k, v)
	}
}

//MAC返回给定密钥和消息的HMAC。
func mac(alg func() hash.Hash, k, m []byte) []byte {
	h := hmac.New(alg, k)
	h.Write(m)
	return h.Sum(nil)
}

//https://tools.ietf.org/html/rfc6979第2.3.3节
func int2octets(v *big.Int, rolen int) []byte {
	out := v.Bytes()

//如果太短，用0填充
	if len(out) < rolen {
		out2 := make([]byte, rolen)
		copy(out2[rolen-len(out):], out)
		return out2
	}

//
	if len(out) > rolen {
		out2 := make([]byte, rolen)
		copy(out2, out[len(out)-rolen:])
		return out2
	}

	return out
}

//
func bits2octets(in []byte, curve elliptic.Curve, rolen int) []byte {
	z1 := hashToInt(in, curve)
	z2 := new(big.Int).Sub(z1, curve.Params().N)
	if z2.Sign() < 0 {
		return int2octets(z1, rolen)
	}
	return int2octets(z2, rolen)
}
