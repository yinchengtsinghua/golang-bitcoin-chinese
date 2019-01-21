
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package btcec

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"math/big"
)

//privatekey将ecdsa.privatekey包装起来，主要是为了方便签名
//使用私钥而不必直接导入ECDSA的内容
//包裹。
type PrivateKey ecdsa.PrivateKey

//privKeyFromBytes基于
//private key passed as an argument as a byte slice.
func PrivKeyFromBytes(curve elliptic.Curve, pk []byte) (*PrivateKey,
	*PublicKey) {
	x, y := curve.ScalarBaseMult(pk)

	priv := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
			X:     x,
			Y:     y,
		},
		D: new(big.Int).SetBytes(pk),
	}

	return (*PrivateKey)(priv), (*PublicKey)(&priv.PublicKey)
}

//newprivatekey是返回privatekey的ecdsa.generatekey的包装器。
//而不是普通的ecdsa.privatekey。
func NewPrivateKey(curve elliptic.Curve) (*PrivateKey, error) {
	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, err
	}
	return (*PrivateKey)(key), nil
}

//pubkey返回与此私钥对应的公钥。
func (p *PrivateKey) PubKey() *PublicKey {
	return (*PublicKey)(&p.PublicKey)
}

//toecdsa以*ecdsa.private key的形式返回私钥。
func (p *PrivateKey) ToECDSA() *ecdsa.PrivateKey {
	return (*ecdsa.PrivateKey)(p)
}

//sign为提供的哈希生成ECDSA签名（这应该是结果
//使用私钥散列较大的消息）。生成的签名
//具有确定性（相同的消息和相同的键产生相同的签名）和规范性
//根据RFC6979和BIP0062。
func (p *PrivateKey) Sign(hash []byte) (*Signature, error) {
	return signRFC6979(p, hash)
}

//privkeybyteslen以字节为单位定义序列化私钥的长度。
const PrivKeyBytesLen = 32

//serialize返回用big endian二进制编码的私钥号d
//数字，填充到32字节的长度。
func (p *PrivateKey) Serialize() []byte {
	b := make([]byte, 0, PrivKeyBytesLen)
	return paddedAppend(PrivKeyBytesLen, b, p.ToECDSA().D.Bytes())
}
