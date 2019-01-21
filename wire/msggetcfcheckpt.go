
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2018 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wire

import (
	"io"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

//msggetcfcheckpt是以均匀间隔请求过滤器头
//在整个区块链历史中。它允许将filtertype字段设置为
//获取基本（0x00）或扩展（0x01）头链中的头。
type MsgGetCFCheckpt struct {
	FilterType FilterType
	StopHash   chainhash.Hash
}

//btcdecode使用比特币协议编码将r解码到接收器中。
//这是消息接口实现的一部分。
func (msg *MsgGetCFCheckpt) BtcDecode(r io.Reader, pver uint32, _ MessageEncoding) error {
	err := readElement(r, &msg.FilterType)
	if err != nil {
		return err
	}

	return readElement(r, &msg.StopHash)
}

//btcencode使用比特币协议编码将接收器编码为w。
//这是消息接口实现的一部分。
func (msg *MsgGetCFCheckpt) BtcEncode(w io.Writer, pver uint32, _ MessageEncoding) error {
	err := writeElement(w, msg.FilterType)
	if err != nil {
		return err
	}

	return writeElement(w, &msg.StopHash)
}

//命令返回消息的协议命令字符串。这是一部分
//消息接口实现。
func (msg *MsgGetCFCheckpt) Command() string {
	return CmdGetCFCheckpt
}

//maxpayloadLength返回有效负载的最大长度
//接收器。这是消息接口实现的一部分。
func (msg *MsgGetCFCheckpt) MaxPayloadLength(pver uint32) uint32 {
//筛选器类型+uint32+块哈希
	return 1 + chainhash.HashSize
}

//newmsggetcfcheckpt返回符合的新比特币getcfcheckpt消息
//使用传递的参数和
//剩余字段。
func NewMsgGetCFCheckpt(filterType FilterType, stopHash *chainhash.Hash) *MsgGetCFCheckpt {
	return &MsgGetCFCheckpt{
		FilterType: filterType,
		StopHash:   *stopHash,
	}
}
