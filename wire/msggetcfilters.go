
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wire

import (
	"io"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

//maxgetfiltersreqrange中可能请求的最大筛选器数
//一个getcfheaders消息。
const MaxGetCFiltersReqRange = 1000

//msggetcfilters实现消息接口并表示比特币
//获取过滤器消息。它用于请求范围为
//阻碍。
type MsgGetCFilters struct {
	FilterType  FilterType
	StartHeight uint32
	StopHash    chainhash.Hash
}

//btcdecode使用比特币协议编码将r解码到接收器中。
//这是消息接口实现的一部分。
func (msg *MsgGetCFilters) BtcDecode(r io.Reader, pver uint32, _ MessageEncoding) error {
	err := readElement(r, &msg.FilterType)
	if err != nil {
		return err
	}

	err = readElement(r, &msg.StartHeight)
	if err != nil {
		return err
	}

	return readElement(r, &msg.StopHash)
}

//btcencode使用比特币协议编码将接收器编码为w。
//这是消息接口实现的一部分。
func (msg *MsgGetCFilters) BtcEncode(w io.Writer, pver uint32, _ MessageEncoding) error {
	err := writeElement(w, msg.FilterType)
	if err != nil {
		return err
	}

	err = writeElement(w, &msg.StartHeight)
	if err != nil {
		return err
	}

	return writeElement(w, &msg.StopHash)
}

//命令返回消息的协议命令字符串。这是一部分
//消息接口实现。
func (msg *MsgGetCFilters) Command() string {
	return CmdGetCFilters
}

//maxpayloadLength返回有效负载的最大长度
//接收器。这是消息接口实现的一部分。
func (msg *MsgGetCFilters) MaxPayloadLength(pver uint32) uint32 {
//筛选器类型+uint32+块哈希
	return 1 + 4 + chainhash.HashSize
}

//newmsggetcpilters返回符合以下条件的新比特币getcpilters消息
//使用传递的参数和默认值的消息接口
//剩余字段。
func NewMsgGetCFilters(filterType FilterType, startHeight uint32,
	stopHash *chainhash.Hash) *MsgGetCFilters {
	return &MsgGetCFilters{
		FilterType:  filterType,
		StartHeight: startHeight,
		StopHash:    *stopHash,
	}
}
