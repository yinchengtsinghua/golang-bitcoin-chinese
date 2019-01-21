
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有2015 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package btcec

import (
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"io/ioutil"
	"strings"
)

//go：生成go run-标记gensecp256k1 genprecomps.go

//loads256bytepoints对预先计算的字节点进行解压缩和反序列化
//用于加速secp256k1曲线的标量基乘法。这个
//approach is used since it allows the compile to use significantly less ram
//在内存中对最终版本进行硬编码时，执行速度要快得多。
//数据结构。同时，在内存中生成
//使用此方法初始化时的数据结构，而不是计算表。
func loadS256BytePoints() error {
//在生成字节点时将没有要加载的字节点。
	bp := secp256k1BytePoints
	if len(bp) == 0 {
		return nil
	}

//
//乘法。
	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(bp))
	r, err := zlib.NewReader(decoder)
	if err != nil {
		return err
	}
	serialized, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

//反序列化预计算的字节点并将曲线设置为它们。
	offset := 0
	var bytePoints [32][256][3]fieldVal
	for byteNum := 0; byteNum < 32; byteNum++ {
//此窗口中的所有点。
		for i := 0; i < 256; i++ {
			px := &bytePoints[byteNum][i][0]
			py := &bytePoints[byteNum][i][1]
			pz := &bytePoints[byteNum][i][2]
			for i := 0; i < 10; i++ {
				px.n[i] = binary.LittleEndian.Uint32(serialized[offset:])
				offset += 4
			}
			for i := 0; i < 10; i++ {
				py.n[i] = binary.LittleEndian.Uint32(serialized[offset:])
				offset += 4
			}
			for i := 0; i < 10; i++ {
				pz.n[i] = binary.LittleEndian.Uint32(serialized[offset:])
				offset += 4
			}
		}
	}
	secp256k1.bytePoints = &bytePoints
	return nil
}
