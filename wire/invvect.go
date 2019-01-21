
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

package wire

import (
	"fmt"
	"io"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

const (
//maxinvpermsg是可以在
//单一比特币发票信息。
	MaxInvPerMsg = 50000

//库存向量的最大有效负载大小。
	maxInvVectPayload = 4 + chainhash.HashSize

//invwitnessFlag表示库存向量类型正在请求，
//或发送包含证人数据的版本。
	InvWitnessFlag = 1 << 30
)

//invtype表示允许的库存向量类型。参见VIVECT。
type InvType uint32

//这些常量定义了各种支持的库存向量类型。
const (
	InvTypeError                InvType = 0
	InvTypeTx                   InvType = 1
	InvTypeBlock                InvType = 2
	InvTypeFilteredBlock        InvType = 3
	InvTypeWitnessBlock         InvType = InvTypeBlock | InvWitnessFlag
	InvTypeWitnessTx            InvType = InvTypeTx | InvWitnessFlag
	InvTypeFilteredWitnessBlock InvType = InvTypeFilteredBlock | InvWitnessFlag
)

//将服务标志映射回其常量名称，以便进行漂亮的打印。
var ivStrings = map[InvType]string{
	InvTypeError:                "ERROR",
	InvTypeTx:                   "MSG_TX",
	InvTypeBlock:                "MSG_BLOCK",
	InvTypeFilteredBlock:        "MSG_FILTERED_BLOCK",
	InvTypeWitnessBlock:         "MSG_WITNESS_BLOCK",
	InvTypeWitnessTx:            "MSG_WITNESS_TX",
	InvTypeFilteredWitnessBlock: "MSG_FILTERED_WITNESS_BLOCK",
}

//字符串返回invtype的可读形式。
func (invtype InvType) String() string {
	if s, ok := ivStrings[invtype]; ok {
		return s
	}

	return fmt.Sprintf("Unknown InvType (%d)", uint32(invtype))
}

//invvect定义了比特币库存向量，用于描述数据，
//按照类型字段的指定，对等方需要、拥有或不需要
//另一个同伴。
type InvVect struct {
Type InvType        //数据类型
Hash chainhash.Hash //数据的哈希
}

//new invvect使用提供的类型和哈希返回新的invvect。
func NewInvVect(typ InvType, hash *chainhash.Hash) *InvVect {
	return &InvVect{
		Type: typ,
		Hash: *hash,
	}
}

//readinvvect根据协议从r中读取编码的invvect
//版本。
func readInvVect(r io.Reader, pver uint32, iv *InvVect) error {
	return readElements(r, &iv.Type, &iv.Hash)
}

//WriteInvvect根据协议版本将Invvect序列化为w。
func writeInvVect(w io.Writer, pver uint32, iv *InvVect) error {
	return writeElements(w, iv.Type, &iv.Hash)
}
