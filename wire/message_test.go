
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
	"encoding/binary"
	"io"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/davecgh/go-spew/spew"
)

//make header是一个方便的函数，可以将消息头的形式设置为
//字节切片。它用于在读取消息时强制出错。
func makeHeader(btcnet BitcoinNet, command string,
	payloadLen uint32, checksum uint32) []byte {

//比特币报文头的长度为24字节。
//比特币网络的4字节幻数+12字节命令+4字节
//有效负载长度+4字节校验和。
	buf := make([]byte, 24)
	binary.LittleEndian.PutUint32(buf, uint32(btcnet))
	copy(buf[4:], []byte(command))
	binary.LittleEndian.PutUint32(buf[16:], payloadLen)
	binary.LittleEndian.PutUint32(buf[20:], checksum)
	return buf
}

//测试消息测试读/写消息和读/写消息API。
func TestMessage(t *testing.T) {
	pver := ProtocolVersion

//创建要测试的各种类型的消息。

//MSGVIEW。
	addrYou := &net.TCPAddr{IP: net.ParseIP("192.168.0.1"), Port: 8333}
	you := NewNetAddress(addrYou, SFNodeNetwork)
you.Timestamp = time.Time{} //版本消息具有零值时间戳。
	addrMe := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8333}
	me := NewNetAddress(addrMe, SFNodeNetwork)
me.Timestamp = time.Time{} //版本消息具有零值时间戳。
	msgVersion := NewMsgVersion(me, you, 123123, 0)

	msgVerack := NewMsgVerAck()
	msgGetAddr := NewMsgGetAddr()
	msgAddr := NewMsgAddr()
	msgGetBlocks := NewMsgGetBlocks(&chainhash.Hash{})
	msgBlock := &blockOne
	msgInv := NewMsgInv()
	msgGetData := NewMsgGetData()
	msgNotFound := NewMsgNotFound()
	msgTx := NewMsgTx(1)
	msgPing := NewMsgPing(123123)
	msgPong := NewMsgPong(123123)
	msgGetHeaders := NewMsgGetHeaders()
	msgHeaders := NewMsgHeaders()
	msgAlert := NewMsgAlert([]byte("payload"), []byte("signature"))
	msgMemPool := NewMsgMemPool()
	msgFilterAdd := NewMsgFilterAdd([]byte{0x01})
	msgFilterClear := NewMsgFilterClear()
	msgFilterLoad := NewMsgFilterLoad([]byte{0x01}, 10, 0, BloomUpdateNone)
	bh := NewBlockHeader(1, &chainhash.Hash{}, &chainhash.Hash{}, 0, 0)
	msgMerkleBlock := NewMsgMerkleBlock(bh)
	msgReject := NewMsgReject("block", RejectDuplicate, "duplicate block")
	msgGetCFilters := NewMsgGetCFilters(GCSFilterRegular, 0, &chainhash.Hash{})
	msgGetCFHeaders := NewMsgGetCFHeaders(GCSFilterRegular, 0, &chainhash.Hash{})
	msgGetCFCheckpt := NewMsgGetCFCheckpt(GCSFilterRegular, &chainhash.Hash{})
	msgCFilter := NewMsgCFilter(GCSFilterRegular, &chainhash.Hash{},
		[]byte("payload"))
	msgCFHeaders := NewMsgCFHeaders()
	msgCFCheckpt := NewMsgCFCheckpt(GCSFilterRegular, &chainhash.Hash{}, 0)

	tests := []struct {
in     Message    //编码值
out    Message    //预期解码值
pver   uint32     //有线编码协议版本
btcnet BitcoinNet //用于有线编码的网络
bytes  int        //应为读/写的num字节
	}{
		{msgVersion, msgVersion, pver, MainNet, 125},
		{msgVerack, msgVerack, pver, MainNet, 24},
		{msgGetAddr, msgGetAddr, pver, MainNet, 24},
		{msgAddr, msgAddr, pver, MainNet, 25},
		{msgGetBlocks, msgGetBlocks, pver, MainNet, 61},
		{msgBlock, msgBlock, pver, MainNet, 239},
		{msgInv, msgInv, pver, MainNet, 25},
		{msgGetData, msgGetData, pver, MainNet, 25},
		{msgNotFound, msgNotFound, pver, MainNet, 25},
		{msgTx, msgTx, pver, MainNet, 34},
		{msgPing, msgPing, pver, MainNet, 32},
		{msgPong, msgPong, pver, MainNet, 32},
		{msgGetHeaders, msgGetHeaders, pver, MainNet, 61},
		{msgHeaders, msgHeaders, pver, MainNet, 25},
		{msgAlert, msgAlert, pver, MainNet, 42},
		{msgMemPool, msgMemPool, pver, MainNet, 24},
		{msgFilterAdd, msgFilterAdd, pver, MainNet, 26},
		{msgFilterClear, msgFilterClear, pver, MainNet, 24},
		{msgFilterLoad, msgFilterLoad, pver, MainNet, 35},
		{msgMerkleBlock, msgMerkleBlock, pver, MainNet, 110},
		{msgReject, msgReject, pver, MainNet, 79},
		{msgGetCFilters, msgGetCFilters, pver, MainNet, 61},
		{msgGetCFHeaders, msgGetCFHeaders, pver, MainNet, 61},
		{msgGetCFCheckpt, msgGetCFCheckpt, pver, MainNet, 57},
		{msgCFilter, msgCFilter, pver, MainNet, 65},
		{msgCFHeaders, msgCFHeaders, pver, MainNet, 90},
		{msgCFCheckpt, msgCFCheckpt, pver, MainNet, 58},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//编码为有线格式。
		var buf bytes.Buffer
		nw, err := WriteMessageN(&buf, test.in, test.pver, test.btcnet)
		if err != nil {
			t.Errorf("WriteMessage #%d error %v", i, err)
			continue
		}

//确保写入的字节数与预期值匹配。
		if nw != test.bytes {
			t.Errorf("WriteMessage #%d unexpected num bytes "+
				"written - got %d, want %d", i, nw, test.bytes)
		}

//从有线格式解码。
		rbuf := bytes.NewReader(buf.Bytes())
		nr, msg, _, err := ReadMessageN(rbuf, test.pver, test.btcnet)
		if err != nil {
			t.Errorf("ReadMessage #%d error %v, msg %v", i, err,
				spew.Sdump(msg))
			continue
		}
		if !reflect.DeepEqual(msg, test.out) {
			t.Errorf("ReadMessage #%d\n got: %v want: %v", i,
				spew.Sdump(msg), spew.Sdump(test.out))
			continue
		}

//确保读取的字节数与预期值匹配。
		if nr != test.bytes {
			t.Errorf("ReadMessage #%d unexpected num bytes read - "+
				"got %d, want %d", i, nr, test.bytes)
		}
	}

//对读/写消息执行相同的操作，但忽略以下字节：
//他们不归还。
	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//编码为有线格式。
		var buf bytes.Buffer
		err := WriteMessage(&buf, test.in, test.pver, test.btcnet)
		if err != nil {
			t.Errorf("WriteMessage #%d error %v", i, err)
			continue
		}

//从有线格式解码。
		rbuf := bytes.NewReader(buf.Bytes())
		msg, _, err := ReadMessage(rbuf, test.pver, test.btcnet)
		if err != nil {
			t.Errorf("ReadMessage #%d error %v, msg %v", i, err,
				spew.Sdump(msg))
			continue
		}
		if !reflect.DeepEqual(msg, test.out) {
			t.Errorf("ReadMessage #%d\n got: %v want: %v", i,
				spew.Sdump(msg), spew.Sdump(test.out))
			continue
		}
	}
}

//testreadMessageWireErrors对
//确认错误路径正确工作的具体消息。
func TestReadMessageWireErrors(t *testing.T) {
	pver := ProtocolVersion
	btcnet := MainNet

//确保消息错误与预期的一样，且未指定任何函数。
	wantErr := "something bad happened"
	testErr := MessageError{Description: wantErr}
	if testErr.Error() != wantErr {
		t.Errorf("MessageError: wrong error - got %v, want %v",
			testErr.Error(), wantErr)
	}

//确保消息错误与指定函数的预期相同。
	wantFunc := "foo"
	testErr = MessageError{Func: wantFunc, Description: wantErr}
	if testErr.Error() != wantFunc+": "+wantErr {
		t.Errorf("MessageError: wrong error - got %v, want %v",
			testErr.Error(), wantErr)
	}

//主网络和testnet3网络魔术标识符的有线编码字节。
	testNet3Bytes := makeHeader(TestNet3, "", 0, 0)

//超过最大总消息数的消息的有线编码字节数
//长度。
	mpl := uint32(MaxMessagePayload)
	exceedMaxPayloadBytes := makeHeader(btcnet, "getaddr", mpl+1, 0)

//UTF-8无效的命令的有线编码字节。
	badCommandBytes := makeHeader(btcnet, "bogus", 0, 0)
	badCommandBytes[4] = 0x81

//有效但不受支持的命令的有线编码字节。
	unsupportedCommandBytes := makeHeader(btcnet, "bogus", 0, 0)

//超过最大有效负载的消息的有线编码字节
//特定的消息类型。
	exceedTypePayloadBytes := makeHeader(btcnet, "getaddr", 1, 0)

//未传递完整消息的有线编码字节
//根据收割台长度计算的有效负载。
	shortPayloadBytes := makeHeader(btcnet, "version", 115, 0)

//带有错误校验和的消息的有线编码字节。
	badChecksumBytes := makeHeader(btcnet, "version", 2, 0xbeef)
	badChecksumBytes = append(badChecksumBytes, []byte{0x0, 0x0}...)

//具有有效头但为
//格式错误。地址以变量开头
//包含在消息中。声称有两个，但不提供
//他们。同时，伪造头字段，使消息
//否则是准确的。
	badMessageBytes := makeHeader(btcnet, "addr", 1, 0xeaadc31c)
	badMessageBytes = append(badMessageBytes, 0x2)

//头声明具有15K的消息的有线编码字节
//要丢弃的数据字节。
	discardBytes := makeHeader(btcnet, "bogus", 15*1024, 0)

	tests := []struct {
buf     []byte     //有线编码
pver    uint32     //有线编码协议版本
btcnet  BitcoinNet //比特币有线编码网
max     int        //引发错误的固定缓冲区的最大大小
readErr error      //预期的读取错误
bytes   int        //预期读取的num字节数
	}{
//具有故意读取错误的最新协议版本。

//短标题。
		{
			[]byte{},
			pver,
			btcnet,
			0,
			io.EOF,
			0,
		},

//网络错误。想要主网，但是给testnet3。
		{
			testNet3Bytes,
			pver,
			btcnet,
			len(testNet3Bytes),
			&MessageError{},
			24,
		},

//超过最大总消息有效负载长度。
		{
			exceedMaxPayloadBytes,
			pver,
			btcnet,
			len(exceedMaxPayloadBytes),
			&MessageError{},
			24,
		},

//UTF-8命令无效。
		{
			badCommandBytes,
			pver,
			btcnet,
			len(badCommandBytes),
			&MessageError{},
			24,
		},

//有效但不受支持的命令。
		{
			unsupportedCommandBytes,
			pver,
			btcnet,
			len(unsupportedCommandBytes),
			&MessageError{},
			24,
		},

//超过特定类型的消息允许的最大有效负载。
		{
			exceedTypePayloadBytes,
			pver,
			btcnet,
			len(exceedTypePayloadBytes),
			&MessageError{},
			24,
		},

//有效负载小于收割台指示的消息。
		{
			shortPayloadBytes,
			pver,
			btcnet,
			len(shortPayloadBytes),
			io.EOF,
			24,
		},

//校验和错误的消息。
		{
			badChecksumBytes,
			pver,
			btcnet,
			len(badChecksumBytes),
			&MessageError{},
			26,
		},

//邮件头有效，但格式错误。
		{
			badMessageBytes,
			pver,
			btcnet,
			len(badMessageBytes),
			io.EOF,
			25,
		},

//要丢弃的15K字节数据。
		{
			discardBytes,
			pver,
			btcnet,
			len(discardBytes),
			&MessageError{},
			24,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//从有线格式解码。
		r := newFixedReader(test.max, test.buf)
		nr, _, _, err := ReadMessageN(r, test.pver, test.btcnet)
		if reflect.TypeOf(err) != reflect.TypeOf(test.readErr) {
			t.Errorf("ReadMessage #%d wrong error got: %v <%T>, "+
				"want: %T", i, err, err, test.readErr)
			continue
		}

//确保写入的字节数与预期值匹配。
		if nr != test.bytes {
			t.Errorf("ReadMessage #%d unexpected num bytes read - "+
				"got %d, want %d", i, nr, test.bytes)
		}

//对于不属于messageerror类型的错误，请检查它们
//平等。
		if _, ok := err.(*MessageError); !ok {
			if err != test.readErr {
				t.Errorf("ReadMessage #%d wrong error got: %v <%T>, "+
					"want: %v <%T>", i, err, err,
					test.readErr, test.readErr)
				continue
			}
		}
	}
}

//TestWriteMessageWireErrors对来自
//确认错误路径正确工作的具体消息。
func TestWriteMessageWireErrors(t *testing.T) {
	pver := ProtocolVersion
	btcnet := MainNet
	wireErr := &MessageError{}

//带有过长命令的假消息。
	badCommandMsg := &fakeMessage{command: "somethingtoolong"}

//在编码过程中出现问题的假消息
	encodeErrMsg := &fakeMessage{forceEncodeErr: true}

//负载超过最大总消息大小的假消息。
	exceedOverallPayload := make([]byte, MaxMessagePayload+1)
	exceedOverallPayloadErrMsg := &fakeMessage{payload: exceedOverallPayload}

//假消息的有效负载超过了每个消息允许的最大值。
	exceedPayload := make([]byte, 1)
	exceedPayloadErrMsg := &fakeMessage{payload: exceedPayload, forceLenErr: true}

//用于强制头段和有效负载中出现错误的假消息
//写。
	bogusPayload := []byte{0x01, 0x02, 0x03, 0x04}
	bogusMsg := &fakeMessage{command: "bogus", payload: bogusPayload}

	tests := []struct {
msg    Message    //要编码的邮件
pver   uint32     //有线编码协议版本
btcnet BitcoinNet //比特币有线编码网
max    int        //引发错误的固定缓冲区的最大大小
err    error      //期望误差
bytes  int        //预期写入的num字节数
	}{
//命令太长。
		{badCommandMsg, pver, btcnet, 0, wireErr, 0},
//有效负载编码中的强制错误。
		{encodeErrMsg, pver, btcnet, 0, wireErr, 0},
//由于超过最大消息有效负载大小而导致强制错误。
		{exceedOverallPayloadErrMsg, pver, btcnet, 0, wireErr, 0},
//由于超过消息类型的最大负载，强制错误。
		{exceedPayloadErrMsg, pver, btcnet, 0, wireErr, 0},
//头写入时强制出错。
		{bogusMsg, pver, btcnet, 0, io.ErrShortWrite, 0},
//有效负载写入中的强制错误。
		{bogusMsg, pver, btcnet, 24, io.ErrShortWrite, 24},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//编码电线格式。
		w := newFixedWriter(test.max)
		nw, err := WriteMessageN(w, test.msg, test.pver, test.btcnet)
		if reflect.TypeOf(err) != reflect.TypeOf(test.err) {
			t.Errorf("WriteMessage #%d wrong error got: %v <%T>, "+
				"want: %T", i, err, err, test.err)
			continue
		}

//确保写入的字节数与预期值匹配。
		if nw != test.bytes {
			t.Errorf("WriteMessage #%d unexpected num bytes "+
				"written - got %d, want %d", i, nw, test.bytes)
		}

//对于不属于messageerror类型的错误，请检查它们
//平等。
		if _, ok := err.(*MessageError); !ok {
			if err != test.err {
				t.Errorf("ReadMessage #%d wrong error got: %v <%T>, "+
					"want: %v <%T>", i, err, err,
					test.err, test.err)
				continue
			}
		}
	}
}
