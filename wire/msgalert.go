
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2015 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wire

import (
	"bytes"
	"fmt"
	"io"
)

//msgalert包含一个有效载荷和一个签名：
//
//================================
//字段数据类型大小
//================================
//有效载荷[]UChar？γ
//————————————————————————————————————————————————————————————————
//签名[]uchar？γ
//————————————————————————————————————————————————————————————————
//
//这里的有效负载是一个序列化为字节数组的警报，以确保
//使用不兼容警报格式的版本仍然可以中继
//互相提醒。
//
//警报是按以下方式反序列化的有效负载：
//
//================================
//字段数据类型大小
//================================
//版本Int32 4
//————————————————————————————————————————————————————————————————
//relayuntil_int64_8_
//————————————————————————————————————————————————————————————————
//到期Int64 8
//————————————————————————————————————————————————————————————————
//ID_Int32_4_
//————————————————————————————————————————————————————————————————
//取消Int32 4
//————————————————————————————————————————————————————————————————
//setcancel_set<int32>？γ
//————————————————————————————————————————————————————————————————
//Minver_Int32_4_
//————————————————————————————————————————————————————————————————
//Maxver_Int32_4_
//————————————————————————————————————————————————————————————————
//setsubver set<string>？γ
//————————————————————————————————————————————————————————————————
//优先级Int32 4
//————————————————————————————————————————————————————————————————
//注释字符串？γ
//————————————————————————————————————————————————————————————————
//状态栏字符串？γ
//————————————————————————————————————————————————————————————————
//保留字符串？γ
//————————————————————————————————————————————————————————————————
//合计（固定）45
//————————————————————————————————————————————————————————————————
//
//注：
//*string是varstring，即变量长度后跟字符串本身。
//*set<string>是一个变量，后面跟有尽可能多的字符串。
//*set<int32>是一个变量，后面跟着整数的数目。
//*固定公差尺寸=40+5*min（变量）=40+5*1=45
//
//现在我们可以定义警报大小、setcancel和setsubver的界限。

//警报有效负载的固定大小
const fixedAlertSize = 45

//MaxSignatureSize是ECDSA签名的最大大小。
//注：由于此尺寸是固定的且小于255，因此所需变量的尺寸为1。
const maxSignatureSize = 72

//MaxAlertSize是警报的最大大小。
//
//messagePayload=varint（alert）+alert+varint（signature）+signature
//maxmessagepayload=maxalertsize+max（varint）+maxsignatureize+1
const maxAlertSize = MaxMessagePayload - maxSignatureSize - MaxVarIntPayload - 1

//MaxCountSetCancel是可能的最大取消ID数。
//符合最大大小警报。
//
//maxalertsize=fixedalertsize+max（setcancel）+max（setsubver）+3*（string）
//要计算取消ID的最大数目，请将所有其他var大小设置为0。
//maxalertsize=fixedalertsize+（maxvarintpayload-1）+x*sizeof（int32）
//X=（MaxAlertSize-FixedAlertSize-MaxVarintPayLoad+1）/4
const maxCountSetCancel = (maxAlertSize - fixedAlertSize - MaxVarIntPayload + 1) / 4

//MaxCountSetSubver是可能的最大子版本数
//符合最大大小警报。
//
//maxalertsize=fixedalertsize+max（setcancel）+max（setsubver）+3*（string）
//要计算最大子版本数，请将所有其他var大小设置为0。
//maxalertsize=fixedalertsize+（maxvarintpayload-1）+x*sizeof（字符串）
//x=（maxalertsize-fixedalertsize-maxvarintpayload+1）/sizeof（字符串）
//Subversion通常类似于“/Satoshi:0.7.2/”（15字节）
//所以假设<255字节，sizeof（string）=sizeof（uint8）+255=256
const maxCountSetSubVer = (maxAlertSize - fixedAlertSize - MaxVarIntPayload + 1) / 256

//警报包含从msgalert负载反序列化的数据。
type Alert struct {
//警报格式版本
	Version int32

//超过此时间戳时，节点应停止中继此警报
	RelayUntil int64

//此警报不再有效的时间戳，并且
//应该被忽略
	Expiration int64

//此通知的唯一ID号
	ID int32

//ID小于或等于此数字的所有警报都应
//取消、删除、以后不接受
	Cancel int32

//应如上所述取消此集合中包含的所有警报ID
	SetCancel []int32

//此警报仅适用于大于或等于此版本的版本
//版本。其他版本仍然应该转发它。
	MinVer int32

//此警报仅适用于小于或等于此版本的版本。
//其他版本仍然应该转发它。
	MaxVer int32

//如果这个集合包含任何元素，那么只有节点
//此集合中包含的子容器受警报影响。其他版本
//还是应该转播。
	SetSubVer []string

//与其他警报相比的相对优先级
	Priority int32

//对未显示的警报的注释
	Comment string

//向用户显示的警报消息
	StatusBar string

//保留的
	Reserved string
}

//serialize使用警报协议编码格式将警报编码为w。
func (alert *Alert) Serialize(w io.Writer, pver uint32) error {
	err := writeElements(w, alert.Version, alert.RelayUntil,
		alert.Expiration, alert.ID, alert.Cancel)
	if err != nil {
		return err
	}

	count := len(alert.SetCancel)
	if count > maxCountSetCancel {
		str := fmt.Sprintf("too many cancel alert IDs for alert "+
			"[count %v, max %v]", count, maxCountSetCancel)
		return messageError("Alert.Serialize", str)
	}
	err = WriteVarInt(w, pver, uint64(count))
	if err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		err = writeElement(w, alert.SetCancel[i])
		if err != nil {
			return err
		}
	}

	err = writeElements(w, alert.MinVer, alert.MaxVer)
	if err != nil {
		return err
	}

	count = len(alert.SetSubVer)
	if count > maxCountSetSubVer {
		str := fmt.Sprintf("too many sub versions for alert "+
			"[count %v, max %v]", count, maxCountSetSubVer)
		return messageError("Alert.Serialize", str)
	}
	err = WriteVarInt(w, pver, uint64(count))
	if err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		err = WriteVarString(w, pver, alert.SetSubVer[i])
		if err != nil {
			return err
		}
	}

	err = writeElement(w, alert.Priority)
	if err != nil {
		return err
	}
	err = WriteVarString(w, pver, alert.Comment)
	if err != nil {
		return err
	}
	err = WriteVarString(w, pver, alert.StatusBar)
	if err != nil {
		return err
	}
	return WriteVarString(w, pver, alert.Reserved)
}

//使用警报协议将解码从R反序列化到接收器中
//编码格式。
func (alert *Alert) Deserialize(r io.Reader, pver uint32) error {
	err := readElements(r, &alert.Version, &alert.RelayUntil,
		&alert.Expiration, &alert.ID, &alert.Cancel)
	if err != nil {
		return err
	}

//setcancel：首先读取包含
//count-取消ID的数目，然后
//重复计数次并读取它们
	count, err := ReadVarInt(r, pver)
	if err != nil {
		return err
	}
	if count > maxCountSetCancel {
		str := fmt.Sprintf("too many cancel alert IDs for alert "+
			"[count %v, max %v]", count, maxCountSetCancel)
		return messageError("Alert.Deserialize", str)
	}
	alert.SetCancel = make([]int32, count)
	for i := 0; i < int(count); i++ {
		err := readElement(r, &alert.SetCancel[i])
		if err != nil {
			return err
		}
	}

	err = readElements(r, &alert.MinVer, &alert.MaxVer)
	if err != nil {
		return err
	}

//setsubver：类似于setcancel
//但读取计数子版本字符串的数目
	count, err = ReadVarInt(r, pver)
	if err != nil {
		return err
	}
	if count > maxCountSetSubVer {
		str := fmt.Sprintf("too many sub versions for alert "+
			"[count %v, max %v]", count, maxCountSetSubVer)
		return messageError("Alert.Deserialize", str)
	}
	alert.SetSubVer = make([]string, count)
	for i := 0; i < int(count); i++ {
		alert.SetSubVer[i], err = ReadVarString(r, pver)
		if err != nil {
			return err
		}
	}

	err = readElement(r, &alert.Priority)
	if err != nil {
		return err
	}
	alert.Comment, err = ReadVarString(r, pver)
	if err != nil {
		return err
	}
	alert.StatusBar, err = ReadVarString(r, pver)
	if err != nil {
		return err
	}
	alert.Reserved, err = ReadVarString(r, pver)
	return err
}

//NewAlert返回一个提供了值的新警报。
func NewAlert(version int32, relayUntil int64, expiration int64,
	id int32, cancel int32, setCancel []int32, minVer int32,
	maxVer int32, setSubVer []string, priority int32, comment string,
	statusBar string) *Alert {
	return &Alert{
		Version:    version,
		RelayUntil: relayUntil,
		Expiration: expiration,
		ID:         id,
		Cancel:     cancel,
		SetCancel:  setCancel,
		MinVer:     minVer,
		MaxVer:     maxVer,
		SetSubVer:  setSubVer,
		Priority:   priority,
		Comment:    comment,
		StatusBar:  statusBar,
		Reserved:   "",
	}
}

//NewAlertFromPayload返回一个警报，其中值从
//序列化负载。
func NewAlertFromPayload(serializedPayload []byte, pver uint32) (*Alert, error) {
	var alert Alert
	r := bytes.NewReader(serializedPayload)
	err := alert.Deserialize(r, pver)
	if err != nil {
		return nil, err
	}
	return &alert, nil
}

//msgalert实现消息接口并定义比特币警报
//消息。
//
//这是一条已签名的消息，它提供客户端应
//显示签名是否与密钥匹配。比特币/比特币Qt专用支票
//根据核心开发人员的签名。
type MsgAlert struct {
//SerializedPayLoad是作为字符串序列化的警报负载，以便
//版本可以更改，但旧版本仍可以传递警报
//客户。
	SerializedPayload []byte

//签名是消息的ECDSA签名。
	Signature []byte

//反序列化负载
	Payload *Alert
}

//btcdecode使用比特币协议编码将r解码到接收器中。
//这是消息接口实现的一部分。
func (msg *MsgAlert) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	var err error

	msg.SerializedPayload, err = ReadVarBytes(r, pver, MaxMessagePayload,
		"alert serialized payload")
	if err != nil {
		return err
	}

	msg.Payload, err = NewAlertFromPayload(msg.SerializedPayload, pver)
	if err != nil {
		msg.Payload = nil
	}

	msg.Signature, err = ReadVarBytes(r, pver, MaxMessagePayload,
		"alert signature")
	return err
}

//btcencode使用比特币协议编码将接收器编码为w。
//这是消息接口实现的一部分。
func (msg *MsgAlert) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	var err error
	var serializedpayload []byte
	if msg.Payload != nil {
//如果可能，尝试序列化负载
		r := new(bytes.Buffer)
		err = msg.Payload.Serialize(r, pver)
		if err != nil {
//序列化失败-忽略并回退
//要序列化payload
			serializedpayload = msg.SerializedPayload
		} else {
			serializedpayload = r.Bytes()
		}
	} else {
		serializedpayload = msg.SerializedPayload
	}
	slen := uint64(len(serializedpayload))
	if slen == 0 {
		return messageError("MsgAlert.BtcEncode", "empty serialized payload")
	}
	err = WriteVarBytes(w, pver, serializedpayload)
	if err != nil {
		return err
	}
	return WriteVarBytes(w, pver, msg.Signature)
}

//命令返回消息的协议命令字符串。这是一部分
//消息接口实现。
func (msg *MsgAlert) Command() string {
	return CmdAlert
}

//maxpayloadLength返回有效负载的最大长度
//接收器。这是消息接口实现的一部分。
func (msg *MsgAlert) MaxPayloadLength(pver uint32) uint32 {
//因为这可能因信息而异，所以将其设为最大值
//允许尺寸。
	return MaxMessagePayload
}

//newmsgalert返回符合消息的新比特币警报消息
//接口。有关详细信息，请参阅msgalert。
func NewMsgAlert(serializedPayload []byte, signature []byte) *MsgAlert {
	return &MsgAlert{
		SerializedPayload: serializedPayload,
		Signature:         signature,
		Payload:           nil,
	}
}
