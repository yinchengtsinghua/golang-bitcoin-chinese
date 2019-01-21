
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2014-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wire

import (
	"fmt"
	"io"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

//rejectcode表示一个数值，远程对等机通过该数值表示
//拒绝邮件的原因。
type RejectCode uint8

//这些常量定义了各种支持的拒绝代码。
const (
	RejectMalformed       RejectCode = 0x01
	RejectInvalid         RejectCode = 0x10
	RejectObsolete        RejectCode = 0x11
	RejectDuplicate       RejectCode = 0x12
	RejectNonstandard     RejectCode = 0x40
	RejectDust            RejectCode = 0x41
	RejectInsufficientFee RejectCode = 0x42
	RejectCheckpoint      RejectCode = 0x43
)

//拒绝代码的映射返回字符串以进行漂亮的打印。
var rejectCodeStrings = map[RejectCode]string{
	RejectMalformed:       "REJECT_MALFORMED",
	RejectInvalid:         "REJECT_INVALID",
	RejectObsolete:        "REJECT_OBSOLETE",
	RejectDuplicate:       "REJECT_DUPLICATE",
	RejectNonstandard:     "REJECT_NONSTANDARD",
	RejectDust:            "REJECT_DUST",
	RejectInsufficientFee: "REJECT_INSUFFICIENTFEE",
	RejectCheckpoint:      "REJECT_CHECKPOINT",
}

//字符串以可读形式返回拒绝代码。
func (code RejectCode) String() string {
	if s, ok := rejectCodeStrings[code]; ok {
		return s
	}

	return fmt.Sprintf("Unknown RejectCode (%d)", uint8(code))
}

//msgreject实现消息接口并表示比特币拒绝
//消息。
//
//在协议版本被拒绝之前，未添加此消息。
type MsgReject struct {
//cmd是被拒绝的消息的命令，例如
//作为命令块或命令x。这可以从命令函数中获得
//消息的
	Cmd string

//REJECTCODE是一个指示命令被拒绝原因的代码。它
//在线路上编码为uint8。
	Code RejectCode

//原因是一个人类可读的字符串，具有特定的详细信息（超过和
//上面的拒绝代码）关于命令被拒绝的原因。
	Reason string

//哈希标识被拒绝的特定块或事务
//因此只应用msgblock和msgtx消息。
	Hash chainhash.Hash
}

//btcdecode使用比特币协议编码将r解码到接收器中。
//这是消息接口实现的一部分。
func (msg *MsgReject) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	if pver < RejectVersion {
		str := fmt.Sprintf("reject message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgReject.BtcDecode", str)
	}

//被拒绝的命令。
	cmd, err := ReadVarString(r, pver)
	if err != nil {
		return err
	}
	msg.Cmd = cmd

//指示命令被拒绝原因的代码。
	err = readElement(r, &msg.Code)
	if err != nil {
		return err
	}

//具有特定细节的可读字符串（在
//拒绝上面的代码）关于命令被拒绝的原因。
	reason, err := ReadVarString(r, pver)
	if err != nil {
		return err
	}
	msg.Reason = reason

//CmdBlock和CmdTx消息有一个额外的哈希字段，
//标识特定的块或事务。
	if msg.Cmd == CmdBlock || msg.Cmd == CmdTx {
		err := readElement(r, &msg.Hash)
		if err != nil {
			return err
		}
	}

	return nil
}

//btcencode使用比特币协议编码将接收器编码为w。
//这是消息接口实现的一部分。
func (msg *MsgReject) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	if pver < RejectVersion {
		str := fmt.Sprintf("reject message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgReject.BtcEncode", str)
	}

//被拒绝的命令。
	err := WriteVarString(w, pver, msg.Cmd)
	if err != nil {
		return err
	}

//指示命令被拒绝原因的代码。
	err = writeElement(w, msg.Code)
	if err != nil {
		return err
	}

//具有特定细节的可读字符串（在
//拒绝上面的代码）关于命令被拒绝的原因。
	err = WriteVarString(w, pver, msg.Reason)
	if err != nil {
		return err
	}

//CmdBlock和CmdTx消息有一个额外的哈希字段，
//标识特定的块或事务。
	if msg.Cmd == CmdBlock || msg.Cmd == CmdTx {
		err := writeElement(w, &msg.Hash)
		if err != nil {
			return err
		}
	}

	return nil
}

//命令返回消息的协议命令字符串。这是一部分
//消息接口实现。
func (msg *MsgReject) Command() string {
	return CmdReject
}

//maxpayloadLength返回有效负载的最大长度
//接收器。这是消息接口实现的一部分。
func (msg *MsgReject) MaxPayloadLength(pver uint32) uint32 {
	plen := uint32(0)
//协议版本之前不存在拒绝消息
//拒绝版本。
	if pver >= RejectVersion {
//不幸的是，比特币协议没有强制执行一个健全的
//限制原因的长度，因此最大有效负载是
//总最大消息有效负载。
		plen = MaxMessagePayload
	}

	return plen
}

//newmsgreject返回符合
//消息接口。有关详细信息，请参阅msgreject。
func NewMsgReject(command string, code RejectCode, reason string) *MsgReject {
	return &MsgReject{
		Cmd:    command,
		Code:   code,
		Reason: reason,
	}
}
