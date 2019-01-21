
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
	"bytes"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

//MessageHeaderSize是比特币消息头中的字节数。
//比特币网络（magic）4字节+命令12字节+有效载荷长度4字节+
//校验和4字节。
const MessageHeaderSize = 24

//commandSize是通用比特币消息中所有命令的固定大小
//标题。较短的命令必须零填充。
const CommandSize = 12

//maxmessagepayload是一条消息可以不考虑其他内容的最大字节数。
//消息本身施加的个别限制。
const MaxMessagePayload = (1024 * 1024 * 32) //32兆字节

//比特币报文头中用于描述报文类型的命令。
const (
	CmdVersion      = "version"
	CmdVerAck       = "verack"
	CmdGetAddr      = "getaddr"
	CmdAddr         = "addr"
	CmdGetBlocks    = "getblocks"
	CmdInv          = "inv"
	CmdGetData      = "getdata"
	CmdNotFound     = "notfound"
	CmdBlock        = "block"
	CmdTx           = "tx"
	CmdGetHeaders   = "getheaders"
	CmdHeaders      = "headers"
	CmdPing         = "ping"
	CmdPong         = "pong"
	CmdAlert        = "alert"
	CmdMemPool      = "mempool"
	CmdFilterAdd    = "filteradd"
	CmdFilterClear  = "filterclear"
	CmdFilterLoad   = "filterload"
	CmdMerkleBlock  = "merkleblock"
	CmdReject       = "reject"
	CmdSendHeaders  = "sendheaders"
	CmdFeeFilter    = "feefilter"
	CmdGetCFilters  = "getcfilters"
	CmdGetCFHeaders = "getcfheaders"
	CmdGetCFCheckpt = "getcfcheckpt"
	CmdCFilter      = "cfilter"
	CmdCFHeaders    = "cfheaders"
	CmdCFCheckpt    = "cfcheckpt"
)

//message encoding表示要使用的有线消息编码格式。
type MessageEncoding uint32

const (
//baseencoding以指定的默认格式对所有消息进行编码
//比特币有线协议。
	BaseEncoding MessageEncoding = 1 << iota

//见证编码对除事务消息以外的所有消息进行编码
//使用默认比特币有线协议规范。为了交易
//消息，将使用bip0144中详细介绍的新编码格式。
	WitnessEncoding
)

//延迟编码是比特币线最近指定的编码方式。
//协议。
var LatestEncoding = WitnessEncoding

//消息是描述比特币消息的接口。一种类型
//实现消息对其数据的表示具有完全控制权
//因此，包含的字段可能比
//直接在协议编码的消息中使用。
type Message interface {
	BtcDecode(io.Reader, uint32, MessageEncoding) error
	BtcEncode(io.Writer, uint32, MessageEncoding) error
	Command() string
	MaxPayloadLength(uint32) uint32
}

//makeEmptyMessage创建基于适当具体类型的消息
//关于命令。
func makeEmptyMessage(command string) (Message, error) {
	var msg Message
	switch command {
	case CmdVersion:
		msg = &MsgVersion{}

	case CmdVerAck:
		msg = &MsgVerAck{}

	case CmdGetAddr:
		msg = &MsgGetAddr{}

	case CmdAddr:
		msg = &MsgAddr{}

	case CmdGetBlocks:
		msg = &MsgGetBlocks{}

	case CmdBlock:
		msg = &MsgBlock{}

	case CmdInv:
		msg = &MsgInv{}

	case CmdGetData:
		msg = &MsgGetData{}

	case CmdNotFound:
		msg = &MsgNotFound{}

	case CmdTx:
		msg = &MsgTx{}

	case CmdPing:
		msg = &MsgPing{}

	case CmdPong:
		msg = &MsgPong{}

	case CmdGetHeaders:
		msg = &MsgGetHeaders{}

	case CmdHeaders:
		msg = &MsgHeaders{}

	case CmdAlert:
		msg = &MsgAlert{}

	case CmdMemPool:
		msg = &MsgMemPool{}

	case CmdFilterAdd:
		msg = &MsgFilterAdd{}

	case CmdFilterClear:
		msg = &MsgFilterClear{}

	case CmdFilterLoad:
		msg = &MsgFilterLoad{}

	case CmdMerkleBlock:
		msg = &MsgMerkleBlock{}

	case CmdReject:
		msg = &MsgReject{}

	case CmdSendHeaders:
		msg = &MsgSendHeaders{}

	case CmdFeeFilter:
		msg = &MsgFeeFilter{}

	case CmdGetCFilters:
		msg = &MsgGetCFilters{}

	case CmdGetCFHeaders:
		msg = &MsgGetCFHeaders{}

	case CmdGetCFCheckpt:
		msg = &MsgGetCFCheckpt{}

	case CmdCFilter:
		msg = &MsgCFilter{}

	case CmdCFHeaders:
		msg = &MsgCFHeaders{}

	case CmdCFCheckpt:
		msg = &MsgCFCheckpt{}

	default:
		return nil, fmt.Errorf("unhandled command [%s]", command)
	}
	return msg, nil
}

//messageheader定义所有比特币协议消息的头结构。
type messageHeader struct {
magic    BitcoinNet //4字节
command  string     //12字节
length   uint32     //4字节
checksum [4]byte    //4字节
}

//readmessageheader从R读取比特币消息头。
func readMessageHeader(r io.Reader) (int, *messageHeader, error) {
//由于readElements不返回读取的字节数，请尝试
//如果存在
//短读，以便知道正确的读取字节数。这作品
//因为标题的大小是固定的。
	var headerBytes [MessageHeaderSize]byte
	n, err := io.ReadFull(r, headerBytes[:])
	if err != nil {
		return n, nil, err
	}
	hr := bytes.NewReader(headerBytes[:])

//从原始头字节创建和填充messageheader结构。
	hdr := messageHeader{}
	var command [CommandSize]byte
	readElements(hr, &hdr.magic, &command, &hdr.length, &hdr.checksum)

//从命令字符串中删除尾随零。
	hdr.command = string(bytes.TrimRight(command[:], string(0)))

	return n, &hdr, nil
}

//Discardinput从读卡器R中分块读取n个字节，并丢弃读取的数据。
//字节。这用于在发生各种错误时跳过有效负载，并有助于
//防止流氓节点通过伪造造成大量内存分配
//标题长度。
func discardInput(r io.Reader, n uint32) {
maxSize := uint32(10 * 1024) //每次10K
	numReads := n / maxSize
	bytesRemaining := n % maxSize
	if n > 0 {
		buf := make([]byte, maxSize)
		for i := uint32(0); i < numReads; i++ {
			io.ReadFull(r, buf)
		}
	}
	if bytesRemaining > 0 {
		buf := make([]byte, bytesRemaining)
		io.ReadFull(r, buf)
	}
}

//writemessagen将比特币消息写入w，包括必要的头
//并返回写入的字节数。此函数是
//与writemessage相同，但它还返回写入的字节数。
func WriteMessageN(w io.Writer, msg Message, pver uint32, btcnet BitcoinNet) (int, error) {
	return WriteMessageWithEncodingN(w, msg, pver, btcnet, BaseEncoding)
}

//WriteMessage将比特币消息写入w，包括必要的头
//信息。此函数与writemessagen相同，但它不相同
//不返回写入的字节数。此功能主要提供
//为了与原始API向后兼容，但它也有助于
//不关心字节计数的调用程序。
func WriteMessage(w io.Writer, msg Message, pver uint32, btcnet BitcoinNet) error {
	_, err := WriteMessageN(w, msg, pver, btcnet)
	return err
}

//writemessagewithencodingn向w写入比特币消息，包括
//必要的头信息并返回写入的字节数。
//此函数与writemessagen相同，只是它还允许调用方
//指定序列化导线时要使用的消息编码格式
//信息。
func WriteMessageWithEncodingN(w io.Writer, msg Message, pver uint32,
	btcnet BitcoinNet, encoding MessageEncoding) (int, error) {

	totalBytes := 0

//强制最大命令大小。
	var command [CommandSize]byte
	cmd := msg.Command()
	if len(cmd) > CommandSize {
		str := fmt.Sprintf("command [%s] is too long [max %v]",
			cmd, CommandSize)
		return totalBytes, messageError("WriteMessage", str)
	}
	copy(command[:], []byte(cmd))

//对消息有效负载进行编码。
	var bw bytes.Buffer
	err := msg.BtcEncode(&bw, pver, encoding)
	if err != nil {
		return totalBytes, err
	}
	payload := bw.Bytes()
	lenp := len(payload)

//强制最大总体消息负载。
	if lenp > MaxMessagePayload {
		str := fmt.Sprintf("message payload is too large - encoded "+
			"%d bytes, but maximum message payload is %d bytes",
			lenp, MaxMessagePayload)
		return totalBytes, messageError("WriteMessage", str)
	}

//基于消息类型强制最大消息负载。
	mpl := msg.MaxPayloadLength(pver)
	if uint32(lenp) > mpl {
		str := fmt.Sprintf("message payload is too large - encoded "+
			"%d bytes, but maximum message payload size for "+
			"messages of type [%s] is %d.", lenp, cmd, mpl)
		return totalBytes, messageError("WriteMessage", str)
	}

//创建邮件头。
	hdr := messageHeader{}
	hdr.magic = btcnet
	hdr.command = cmd
	hdr.length = uint32(lenp)
	copy(hdr.checksum[:], chainhash.DoubleHashB(payload)[0:4])

//对邮件头进行编码。对缓冲区执行此操作
//而不是直接指向编写器，因为WriteElements没有
//返回写入的字节数。
	hw := bytes.NewBuffer(make([]byte, 0, MessageHeaderSize))
	writeElements(hw, hdr.magic, command, hdr.length, hdr.checksum)

//写入头。
	n, err := w.Write(hw.Bytes())
	totalBytes += n
	if err != nil {
		return totalBytes, err
	}

//写入有效载荷。
	n, err = w.Write(payload)
	totalBytes += n
	return totalBytes, err
}

//readmessagewithencodingn读取、验证和分析下一比特币消息
//对于提供的协议版本和比特币网络。它返回
//除了解析的消息和原始字节外，读取的字节数
//组成信息。此函数与readmessagen相同，只是
//允许调用方指定在以下情况下要查询的消息编码：
//解码有线消息。
func ReadMessageWithEncodingN(r io.Reader, pver uint32, btcnet BitcoinNet,
	enc MessageEncoding) (int, Message, []byte, error) {

	totalBytes := 0
	n, hdr, err := readMessageHeader(r)
	totalBytes += n
	if err != nil {
		return totalBytes, nil, nil, err
	}

//强制最大消息负载。
	if hdr.length > MaxMessagePayload {
		str := fmt.Sprintf("message payload is too large - header "+
			"indicates %d bytes, but max message payload is %d "+
			"bytes.", hdr.length, MaxMessagePayload)
		return totalBytes, nil, nil, messageError("ReadMessage", str)

	}

//检查来自错误比特币网络的信息。
	if hdr.magic != btcnet {
		discardInput(r, hdr.length)
		str := fmt.Sprintf("message from other network [%v]", hdr.magic)
		return totalBytes, nil, nil, messageError("ReadMessage", str)
	}

//检查命令格式是否错误。
	command := hdr.command
	if !utf8.ValidString(command) {
		discardInput(r, hdr.length)
		str := fmt.Sprintf("invalid command %v", []byte(command))
		return totalBytes, nil, nil, messageError("ReadMessage", str)
	}

//基于命令创建适当消息类型的结构。
	msg, err := makeEmptyMessage(command)
	if err != nil {
		discardInput(r, hdr.length)
		return totalBytes, nil, nil, messageError("ReadMessage",
			err.Error())
	}

//根据恶意客户端的消息类型检查最大长度
//否则将创建格式良好的头并将长度设置为max
//数字以耗尽机器的记忆。
	mpl := msg.MaxPayloadLength(pver)
	if hdr.length > mpl {
		discardInput(r, hdr.length)
		str := fmt.Sprintf("payload exceeds max length - header "+
			"indicates %v bytes, but max payload size for "+
			"messages of type [%v] is %v.", hdr.length, command, mpl)
		return totalBytes, nil, nil, messageError("ReadMessage", str)
	}

//读取有效载荷。
	payload := make([]byte, hdr.length)
	n, err = io.ReadFull(r, payload)
	totalBytes += n
	if err != nil {
		return totalBytes, nil, nil, err
	}

//测试校验和。
	checksum := chainhash.DoubleHashB(payload)[0:4]
	if !bytes.Equal(checksum[:], hdr.checksum[:]) {
		str := fmt.Sprintf("payload checksum failed - header "+
			"indicates %v, but actual checksum is %v.",
			hdr.checksum, checksum)
		return totalBytes, nil, nil, messageError("ReadMessage", str)
	}

//取消标记邮件。注意：这必须是*bytes.buffer，因为
//msgversion btcdecode函数需要它。
	pr := bytes.NewBuffer(payload)
	err = msg.BtcDecode(pr, pver, enc)
	if err != nil {
		return totalBytes, nil, nil, err
	}

	return totalBytes, msg, payload, nil
}

//readmessagen读取、验证和分析R的下一个比特币消息
//提供的协议版本和比特币网络。它返回
//除了解析的消息和包含
//消息。此函数与readmessage相同，只是它还返回
//读取的字节数。
func ReadMessageN(r io.Reader, pver uint32, btcnet BitcoinNet) (int, Message, []byte, error) {
	return ReadMessageWithEncodingN(r, pver, btcnet, BaseEncoding)
}

//readmessage读取、验证和分析R的下一个比特币消息
//提供的协议版本和比特币网络。它返回已解析的
//包含消息的消息和原始字节。此功能仅不同
//因为它不返回读取的字节数。这个
//功能主要提供与原始版本的向后兼容性
//API，但对于不关心字节计数的调用程序也很有用。
func ReadMessage(r io.Reader, pver uint32, btcnet BitcoinNet) (Message, []byte, error) {
	_, msg, buf, err := ReadMessageN(r, pver, btcnet)
	return msg, buf, err
}
