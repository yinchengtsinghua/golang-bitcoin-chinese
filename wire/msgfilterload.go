
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2014-2015 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wire

import (
	"fmt"
	"io"
)

//BloomUpdateType指定在找到匹配项时如何更新筛选器
type BloomUpdateType uint8

const (
//BloomUpdateNone表示匹配项为
//找到了。
	BloomUpdateNone BloomUpdateType = 0

//BloomUpdateAll指示筛选器是否匹配
//公钥脚本，输出点被序列化并插入到
//过滤器。
	BloomUpdateAll BloomUpdateType = 1

//BloomUpdateP2PubKeyOnly指示筛选器是否匹配数据
//公共密钥脚本中的元素，脚本是标准的
//支付到pubkey或multisig，输出点被序列化并插入
//进入过滤器。
	BloomUpdateP2PubkeyOnly BloomUpdateType = 2
)

const (
//MaxFilterLoadHashFuncs是哈希函数的最大数目
//装入布卢姆过滤器。
	MaxFilterLoadHashFuncs = 50

//MaxFilterLoadFilterSize是筛选器的最大大小（以字节为单位）。
	MaxFilterLoadFilterSize = 36000
)

//msgfilterload实现消息接口并表示比特币
//filterload用于重置bloom筛选器的消息。
//
//在协议版本bip0037之前未添加此消息。
type MsgFilterLoad struct {
	Filter    []byte
	HashFuncs uint32
	Tweak     uint32
	Flags     BloomUpdateType
}

//btcdecode使用比特币协议编码将r解码到接收器中。
//这是消息接口实现的一部分。
func (msg *MsgFilterLoad) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	if pver < BIP0037Version {
		str := fmt.Sprintf("filterload message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgFilterLoad.BtcDecode", str)
	}

	var err error
	msg.Filter, err = ReadVarBytes(r, pver, MaxFilterLoadFilterSize,
		"filterload filter size")
	if err != nil {
		return err
	}

	err = readElements(r, &msg.HashFuncs, &msg.Tweak, &msg.Flags)
	if err != nil {
		return err
	}

	if msg.HashFuncs > MaxFilterLoadHashFuncs {
		str := fmt.Sprintf("too many filter hash functions for message "+
			"[count %v, max %v]", msg.HashFuncs, MaxFilterLoadHashFuncs)
		return messageError("MsgFilterLoad.BtcDecode", str)
	}

	return nil
}

//btcencode使用比特币协议编码将接收器编码为w。
//这是消息接口实现的一部分。
func (msg *MsgFilterLoad) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	if pver < BIP0037Version {
		str := fmt.Sprintf("filterload message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgFilterLoad.BtcEncode", str)
	}

	size := len(msg.Filter)
	if size > MaxFilterLoadFilterSize {
		str := fmt.Sprintf("filterload filter size too large for message "+
			"[size %v, max %v]", size, MaxFilterLoadFilterSize)
		return messageError("MsgFilterLoad.BtcEncode", str)
	}

	if msg.HashFuncs > MaxFilterLoadHashFuncs {
		str := fmt.Sprintf("too many filter hash functions for message "+
			"[count %v, max %v]", msg.HashFuncs, MaxFilterLoadHashFuncs)
		return messageError("MsgFilterLoad.BtcEncode", str)
	}

	err := WriteVarBytes(w, pver, msg.Filter)
	if err != nil {
		return err
	}

	return writeElements(w, msg.HashFuncs, msg.Tweak, msg.Flags)
}

//命令返回消息的协议命令字符串。这是一部分
//消息接口实现。
func (msg *MsgFilterLoad) Command() string {
	return CmdFilterLoad
}

//maxpayloadLength返回有效负载的最大长度
//接收器。这是消息接口实现的一部分。
func (msg *MsgFilterLoad) MaxPayloadLength(pver uint32) uint32 {
//num filter bytes（varint）+filter+4 bytes hash funcs+
//4字节调整+1字节标志。
	return uint32(VarIntSerializeSize(MaxFilterLoadFilterSize)) +
		MaxFilterLoadFilterSize + 9
}

//newmsgfilterload返回符合以下条件的新比特币filterload消息
//消息接口。有关详细信息，请参阅msgfilterload。
func NewMsgFilterLoad(filter []byte, hashFuncs uint32, tweak uint32, flags BloomUpdateType) *MsgFilterLoad {
	return &MsgFilterLoad{
		Filter:    filter,
		HashFuncs: hashFuncs,
		Tweak:     tweak,
		Flags:     flags,
	}
}
