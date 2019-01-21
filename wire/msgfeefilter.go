
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wire

import (
	"fmt"
	"io"
)

//msgfeefilter实现消息接口并表示比特币
//F过滤器信息。它用于请求接收端没有
//宣布低于指定最低费率的任何交易。
//
//在以开始的协议版本之前未添加此消息
//过滤器版本。
type MsgFeeFilter struct {
	MinFee int64
}

//btcdecode使用比特币协议编码将r解码到接收器中。
//这是消息接口实现的一部分。
func (msg *MsgFeeFilter) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	if pver < FeeFilterVersion {
		str := fmt.Sprintf("feefilter message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgFeeFilter.BtcDecode", str)
	}

	return readElement(r, &msg.MinFee)
}

//btcencode使用比特币协议编码将接收器编码为w。
//这是消息接口实现的一部分。
func (msg *MsgFeeFilter) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	if pver < FeeFilterVersion {
		str := fmt.Sprintf("feefilter message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgFeeFilter.BtcEncode", str)
	}

	return writeElement(w, msg.MinFee)
}

//命令返回消息的协议命令字符串。这是一部分
//消息接口实现。
func (msg *MsgFeeFilter) Command() string {
	return CmdFeeFilter
}

//maxpayloadLength返回有效负载的最大长度
//接收器。这是消息接口实现的一部分。
func (msg *MsgFeeFilter) MaxPayloadLength(pver uint32) uint32 {
	return 8
}

//newmsgfeefilter返回符合的新比特币feefilter消息
//消息接口。有关详细信息，请参阅msgfeefilter。
func NewMsgFeeFilter(minfee int64) *MsgFeeFilter {
	return &MsgFeeFilter{
		MinFee: minfee,
	}
}
