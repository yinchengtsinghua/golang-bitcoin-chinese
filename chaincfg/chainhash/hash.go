
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2016 BTCSuite开发者
//版权所有（c）2015版权所有
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package chainhash

import (
	"encoding/hex"
	"fmt"
)

//用于存储哈希的数组的哈希大小。参见哈希。
const HashSize = 32

//MaxHashstringSize是哈希字符串的最大长度。
const MaxHashStringSize = HashSize * 2

//errHashStrSize描述一个错误，该错误指示调用方指定了哈希
//字符太多的字符串。
var ErrHashStrSize = fmt.Errorf("max hash string length is %v bytes", MaxHashStringSize)

//哈希用于比特币消息和常见结构中。它
//通常表示数据的双sha256。
type Hash [HashSize]byte

//string返回哈希值，将字节的十六进制字符串反转
//搞砸。
func (hash Hash) String() string {
	for i := 0; i < HashSize/2; i++ {
		hash[i], hash[HashSize-1-i] = hash[HashSize-1-i], hash[i]
	}
	return hex.EncodeToString(hash[:])
}

//CloneBytes返回字节的副本，该副本将哈希表示为一个字节。
//切片。
//
//注意：直接切碎散列通常比较便宜，这样可以重复使用
//相同的字节，而不是调用此方法。
func (hash *Hash) CloneBytes() []byte {
	newHash := make([]byte, HashSize)
	copy(newHash, hash[:])

	return newHash
}

//setbytes设置表示哈希的字节。如果
//传入的字节数不是hashsize。
func (hash *Hash) SetBytes(newHash []byte) error {
	nhlen := len(newHash)
	if nhlen != HashSize {
		return fmt.Errorf("invalid hash length of %v, want %v", nhlen,
			HashSize)
	}
	copy(hash[:], newHash)

	return nil
}

//如果目标与哈希相同，则IsEqual返回true。
func (hash *Hash) IsEqual(target *Hash) bool {
	if hash == nil && target == nil {
		return true
	}
	if hash == nil || target == nil {
		return false
	}
	return *hash == *target
}

//new hash从字节片返回新的哈希。如果
//传入的字节数不是hashsize。
func NewHash(newHash []byte) (*Hash, error) {
	var sh Hash
	err := sh.SetBytes(newHash)
	if err != nil {
		return nil, err
	}
	return &sh, err
}

//newhashfromstr从哈希字符串创建哈希。字符串应该是
//字节反向散列的十六进制字符串，但缺少任何字符
//在散列结尾处导致零填充。
func NewHashFromStr(hash string) (*Hash, error) {
	ret := new(Hash)
	err := Decode(ret, hash)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

//decode将哈希的字节反向十六进制字符串编码解码为
//目的地。
func Decode(dst *Hash, src string) error {
//如果哈希字符串太长，则返回错误。
	if len(src) > MaxHashStringSize {
		return ErrHashStrSize
	}

//十六进制解码器要求哈希为二的倍数。当不是，垫
//以零开头。
	var srcBytes []byte
	if len(src)%2 == 0 {
		srcBytes = []byte(src)
	} else {
		srcBytes = make([]byte, 1+len(src))
		srcBytes[0] = '0'
		copy(srcBytes[1:], src)
	}

//十六进制将源字节解码为临时目标。
	var reversedHash Hash
	_, err := hex.Decode(reversedHash[HashSize-hex.DecodedLen(len(srcBytes)):], srcBytes)
	if err != nil {
		return err
	}

//从临时哈希反向复制到目标。因为
//临时调零，将正确填充写入的结果。
	for i, b := range reversedHash[:HashSize/2] {
		dst[i], dst[HashSize-1-i] = reversedHash[HashSize-1-i], b
	}

	return nil
}
