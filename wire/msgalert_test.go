
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
	"io"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

//testmsgalert测试msgalert API。
func TestMsgAlert(t *testing.T) {
	pver := ProtocolVersion
	encoding := BaseEncoding
	serializedpayload := []byte("some message")
	signature := []byte("some sig")

//确保我们得到相同的有效载荷和签名。
	msg := NewMsgAlert(serializedpayload, signature)
	if !reflect.DeepEqual(msg.SerializedPayload, serializedpayload) {
		t.Errorf("NewMsgAlert: wrong serializedpayload - got %v, want %v",
			msg.SerializedPayload, serializedpayload)
	}
	if !reflect.DeepEqual(msg.Signature, signature) {
		t.Errorf("NewMsgAlert: wrong signature - got %v, want %v",
			msg.Signature, signature)
	}

//确保命令为预期值。
	wantCmd := "alert"
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgAlert: wrong command - got %v want %v",
			cmd, wantCmd)
	}

//确保最大有效负载为预期值。
	wantPayload := uint32(1024 * 1024 * 32)
	maxPayload := msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}

//有效载荷=零的测试btcencode
	var buf bytes.Buffer
	err := msg.BtcEncode(&buf, pver, encoding)
	if err != nil {
		t.Error(err.Error())
	}
//应为=0x0C+SerializedPayLoad+0x08+签名
	expectedBuf := append([]byte{0x0c}, serializedpayload...)
	expectedBuf = append(expectedBuf, []byte{0x08}...)
	expectedBuf = append(expectedBuf, signature...)
	if !bytes.Equal(buf.Bytes(), expectedBuf) {
		t.Errorf("BtcEncode got: %s want: %s",
			spew.Sdump(buf.Bytes()), spew.Sdump(expectedBuf))
	}

//用有效载荷测试btcencode！=零
//注意：有效负载是空警报，但不是零
	msg.Payload = new(Alert)
	buf = *new(bytes.Buffer)
	err = msg.BtcEncode(&buf, pver, encoding)
	if err != nil {
		t.Error(err.Error())
	}
//空警报为45个空字节，请参阅警报注释
//详情
//应为=0x2d+45*0x00+0x08+签名
	expectedBuf = append([]byte{0x2d}, bytes.Repeat([]byte{0x00}, 45)...)
	expectedBuf = append(expectedBuf, []byte{0x08}...)
	expectedBuf = append(expectedBuf, signature...)
	if !bytes.Equal(buf.Bytes(), expectedBuf) {
		t.Errorf("BtcEncode got: %s want: %s",
			spew.Sdump(buf.Bytes()), spew.Sdump(expectedBuf))
	}
}

//testmsgalertwire测试各种协议的msgalert线编码和解码
//版本。
func TestMsgAlertWire(t *testing.T) {
	baseMsgAlert := NewMsgAlert([]byte("some payload"), []byte("somesig"))
	baseMsgAlertEncoded := []byte{
0x0c, //有效载荷长度变量
		0x73, 0x6f, 0x6d, 0x65, 0x20, 0x70, 0x61, 0x79,
0x6c, 0x6f, 0x61, 0x64, //“一些有效载荷”
0x07,                                     //签名长度变量
0x73, 0x6f, 0x6d, 0x65, 0x73, 0x69, 0x67, //“SOMESIG”
	}

	tests := []struct {
in   *MsgAlert       //要编码的邮件
out  *MsgAlert       //预期的解码消息
buf  []byte          //有线编码
pver uint32          //有线编码协议版本
enc  MessageEncoding //消息编码格式
	}{
//最新协议版本。
		{
			baseMsgAlert,
			baseMsgAlert,
			baseMsgAlertEncoded,
			ProtocolVersion,
			BaseEncoding,
		},

//协议版本BIP0035版本。
		{
			baseMsgAlert,
			baseMsgAlert,
			baseMsgAlertEncoded,
			BIP0035Version,
			BaseEncoding,
		},

//协议版本Bip0031版本。
		{
			baseMsgAlert,
			baseMsgAlert,
			baseMsgAlertEncoded,
			BIP0031Version,
			BaseEncoding,
		},

//协议版本NetAddressTimeVersion。
		{
			baseMsgAlert,
			baseMsgAlert,
			baseMsgAlertEncoded,
			NetAddressTimeVersion,
			BaseEncoding,
		},

//协议版本multipleaddressversion。
		{
			baseMsgAlert,
			baseMsgAlert,
			baseMsgAlertEncoded,
			MultipleAddressVersion,
			BaseEncoding,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//将邮件编码为有线格式。
		var buf bytes.Buffer
		err := test.in.BtcEncode(&buf, test.pver, test.enc)
		if err != nil {
			t.Errorf("BtcEncode #%d error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("BtcEncode #%d\n got: %s want: %s", i,
				spew.Sdump(buf.Bytes()), spew.Sdump(test.buf))
			continue
		}

//从有线格式解码消息。
		var msg MsgAlert
		rbuf := bytes.NewReader(test.buf)
		err = msg.BtcDecode(rbuf, test.pver, test.enc)
		if err != nil {
			t.Errorf("BtcDecode #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(&msg, test.out) {
			t.Errorf("BtcDecode #%d\n got: %s want: %s", i,
				spew.Sdump(msg), spew.Sdump(test.out))
			continue
		}
	}
}

//testmsGallertwireErrors对线编码和解码执行负测试
//以确认错误路径正常工作。
func TestMsgAlertWireErrors(t *testing.T) {
	pver := ProtocolVersion
	encoding := BaseEncoding

	baseMsgAlert := NewMsgAlert([]byte("some payload"), []byte("somesig"))
	baseMsgAlertEncoded := []byte{
0x0c, //有效载荷长度变量
		0x73, 0x6f, 0x6d, 0x65, 0x20, 0x70, 0x61, 0x79,
0x6c, 0x6f, 0x61, 0x64, //“一些有效载荷”
0x07,                                     //签名长度变量
0x73, 0x6f, 0x6d, 0x65, 0x73, 0x69, 0x67, //“SOMESIG”
	}

	tests := []struct {
in       *MsgAlert       //编码值
buf      []byte          //有线编码
pver     uint32          //有线编码协议版本
enc      MessageEncoding //消息编码格式
max      int             //引发错误的固定缓冲区的最大大小
writeErr error           //预期的写入错误
readErr  error           //预期的读取错误
	}{
//有效负载长度中的强制错误。
		{baseMsgAlert, baseMsgAlertEncoded, pver, BaseEncoding, 0, io.ErrShortWrite, io.EOF},
//有效负载中的强制错误。
		{baseMsgAlert, baseMsgAlertEncoded, pver, BaseEncoding, 1, io.ErrShortWrite, io.EOF},
//强制签名长度出错。
		{baseMsgAlert, baseMsgAlertEncoded, pver, BaseEncoding, 13, io.ErrShortWrite, io.EOF},
//强制签名出错。
		{baseMsgAlert, baseMsgAlertEncoded, pver, BaseEncoding, 14, io.ErrShortWrite, io.EOF},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//编码为有线格式。
		w := newFixedWriter(test.max)
		err := test.in.BtcEncode(w, test.pver, test.enc)
		if reflect.TypeOf(err) != reflect.TypeOf(test.writeErr) {
			t.Errorf("BtcEncode #%d wrong error got: %v, want: %v",
				i, err, test.writeErr)
			continue
		}

//对于不属于messageerror类型的错误，请检查它们
//平等。
		if _, ok := err.(*MessageError); !ok {
			if err != test.writeErr {
				t.Errorf("BtcEncode #%d wrong error got: %v, "+
					"want: %v", i, err, test.writeErr)
				continue
			}
		}

//从有线格式解码。
		var msg MsgAlert
		r := newFixedReader(test.max, test.buf)
		err = msg.BtcDecode(r, test.pver, test.enc)
		if reflect.TypeOf(err) != reflect.TypeOf(test.readErr) {
			t.Errorf("BtcDecode #%d wrong error got: %v, want: %v",
				i, err, test.readErr)
			continue
		}

//对于不属于messageerror类型的错误，请检查它们
//平等。
		if _, ok := err.(*MessageError); !ok {
			if err != test.readErr {
				t.Errorf("BtcDecode #%d wrong error got: %v, "+
					"want: %v", i, err, test.readErr)
				continue
			}
		}
	}

//空负载上的测试错误
	baseMsgAlert.SerializedPayload = []byte{}
	w := new(bytes.Buffer)
	err := baseMsgAlert.BtcEncode(w, pver, encoding)
	if _, ok := err.(*MessageError); !ok {
		t.Errorf("MsgAlert.BtcEncode wrong error got: %T, want: %T",
			err, MessageError{})
	}

//测试负载序列化错误
//溢出setcancel中的最大元素数
	baseMsgAlert.Payload = new(Alert)
	baseMsgAlert.Payload.SetCancel = make([]int32, maxCountSetCancel+1)
	buf := *new(bytes.Buffer)
	err = baseMsgAlert.BtcEncode(&buf, pver, encoding)
	if _, ok := err.(*MessageError); !ok {
		t.Errorf("MsgAlert.BtcEncode wrong error got: %T, want: %T",
			err, MessageError{})
	}

//溢出setsubver中的最大元素数
	baseMsgAlert.Payload = new(Alert)
	baseMsgAlert.Payload.SetSubVer = make([]string, maxCountSetSubVer+1)
	buf = *new(bytes.Buffer)
	err = baseMsgAlert.BtcEncode(&buf, pver, encoding)
	if _, ok := err.(*MessageError); !ok {
		t.Errorf("MsgAlert.BtcEncode wrong error got: %T, want: %T",
			err, MessageError{})
	}
}

//测试程序测试序列化和反序列化
//要警报的有效载荷
func TestAlert(t *testing.T) {
	pver := ProtocolVersion
	alert := NewAlert(
		1, 1337093712, 1368628812, 1015,
		1013, []int32{1014}, 0, 40599, []string{"/Satoshi:0.7.2/"}, 5000, "",
"URGENT: upgrade required, see http://bitcoin.org/dos了解详细信息“，
	)
	w := new(bytes.Buffer)
	err := alert.Serialize(w, pver)
	if err != nil {
		t.Error(err.Error())
	}
	serializedpayload := w.Bytes()
	newAlert, err := NewAlertFromPayload(serializedpayload, pver)
	if err != nil {
		t.Error(err.Error())
	}

	if alert.Version != newAlert.Version {
		t.Errorf("NewAlertFromPayload: wrong Version - got %v, want %v ",
			alert.Version, newAlert.Version)
	}
	if alert.RelayUntil != newAlert.RelayUntil {
		t.Errorf("NewAlertFromPayload: wrong RelayUntil - got %v, want %v ",
			alert.RelayUntil, newAlert.RelayUntil)
	}
	if alert.Expiration != newAlert.Expiration {
		t.Errorf("NewAlertFromPayload: wrong Expiration - got %v, want %v ",
			alert.Expiration, newAlert.Expiration)
	}
	if alert.ID != newAlert.ID {
		t.Errorf("NewAlertFromPayload: wrong ID - got %v, want %v ",
			alert.ID, newAlert.ID)
	}
	if alert.Cancel != newAlert.Cancel {
		t.Errorf("NewAlertFromPayload: wrong Cancel - got %v, want %v ",
			alert.Cancel, newAlert.Cancel)
	}
	if len(alert.SetCancel) != len(newAlert.SetCancel) {
		t.Errorf("NewAlertFromPayload: wrong number of SetCancel - got %v, want %v ",
			len(alert.SetCancel), len(newAlert.SetCancel))
	}
	for i := 0; i < len(alert.SetCancel); i++ {
		if alert.SetCancel[i] != newAlert.SetCancel[i] {
			t.Errorf("NewAlertFromPayload: wrong SetCancel[%v] - got %v, want %v ",
				len(alert.SetCancel), alert.SetCancel[i], newAlert.SetCancel[i])
		}
	}
	if alert.MinVer != newAlert.MinVer {
		t.Errorf("NewAlertFromPayload: wrong MinVer - got %v, want %v ",
			alert.MinVer, newAlert.MinVer)
	}
	if alert.MaxVer != newAlert.MaxVer {
		t.Errorf("NewAlertFromPayload: wrong MaxVer - got %v, want %v ",
			alert.MaxVer, newAlert.MaxVer)
	}
	if len(alert.SetSubVer) != len(newAlert.SetSubVer) {
		t.Errorf("NewAlertFromPayload: wrong number of SetSubVer - got %v, want %v ",
			len(alert.SetSubVer), len(newAlert.SetSubVer))
	}
	for i := 0; i < len(alert.SetSubVer); i++ {
		if alert.SetSubVer[i] != newAlert.SetSubVer[i] {
			t.Errorf("NewAlertFromPayload: wrong SetSubVer[%v] - got %v, want %v ",
				len(alert.SetSubVer), alert.SetSubVer[i], newAlert.SetSubVer[i])
		}
	}
	if alert.Priority != newAlert.Priority {
		t.Errorf("NewAlertFromPayload: wrong Priority - got %v, want %v ",
			alert.Priority, newAlert.Priority)
	}
	if alert.Comment != newAlert.Comment {
		t.Errorf("NewAlertFromPayload: wrong Comment - got %v, want %v ",
			alert.Comment, newAlert.Comment)
	}
	if alert.StatusBar != newAlert.StatusBar {
		t.Errorf("NewAlertFromPayload: wrong StatusBar - got %v, want %v ",
			alert.StatusBar, newAlert.StatusBar)
	}
	if alert.Reserved != newAlert.Reserved {
		t.Errorf("NewAlertFromPayload: wrong Reserved - got %v, want %v ",
			alert.Reserved, newAlert.Reserved)
	}
}

//TestalerTerrs对有效负载序列化执行负测试，
//反序列化警报以确认错误路径正常工作。
func TestAlertErrors(t *testing.T) {
	pver := ProtocolVersion

	baseAlert := NewAlert(
		1, 1337093712, 1368628812, 1015,
		1013, []int32{1014}, 0, 40599, []string{"/Satoshi:0.7.2/"}, 5000, "",
		"URGENT",
	)
	baseAlertEncoded := []byte{
0x01, 0x00, 0x00, 0x00, 0x50, 0x6e, 0xb2, 0x4f, 0x00, 0x00, 0x00, 0x00, 0x4c, 0x9e, 0x93, 0x51, //……零件号……L.Q
0x00, 0x00, 0x00, 0x00, 0xf7, 0x03, 0x00, 0x00, 0xf5, 0x03, 0x00, 0x00, 0x01, 0xf6, 0x03, 0x00, //…………………
0x00, 0x00, 0x00, 0x00, 0x00, 0x97, 0x9e, 0x00, 0x00, 0x01, 0x0f, 0x2f, 0x53, 0x61, 0x74, 0x6f, //………/Sato_
0x73, 0x68, 0x69, 0x3a, 0x30, 0x2e, 0x37, 0x2e, 0x32, 0x2f, 0x88, 0x13, 0x00, 0x00, 0x00, 0x06, //时：0.7.2/………
0x55, 0x52, 0x47, 0x45, 0x4e, 0x54, 0x00, //紧急。
	}
	tests := []struct {
in       *Alert //编码值
buf      []byte //有线编码
pver     uint32 //有线编码协议版本
max      int    //引发错误的固定缓冲区的最大大小
writeErr error  //预期的写入错误
readErr  error  //预期的读取错误
	}{
//强制版本错误
		{baseAlert, baseAlertEncoded, pver, 0, io.ErrShortWrite, io.EOF},
//setcancel变量中的强制错误。
		{baseAlert, baseAlertEncoded, pver, 28, io.ErrShortWrite, io.EOF},
//setcancel ints中的强制错误。
		{baseAlert, baseAlertEncoded, pver, 29, io.ErrShortWrite, io.EOF},
//Minver中的强制错误
		{baseAlert, baseAlertEncoded, pver, 40, io.ErrShortWrite, io.EOF},
//setsubver字符串变量中的强制错误。
		{baseAlert, baseAlertEncoded, pver, 41, io.ErrShortWrite, io.EOF},
//SetSubver字符串中的强制错误。
		{baseAlert, baseAlertEncoded, pver, 48, io.ErrShortWrite, io.EOF},
//强制优先错误
		{baseAlert, baseAlertEncoded, pver, 60, io.ErrShortWrite, io.EOF},
//强制注释字符串出错。
		{baseAlert, baseAlertEncoded, pver, 62, io.ErrShortWrite, io.EOF},
//statusbar字符串中的强制错误。
		{baseAlert, baseAlertEncoded, pver, 64, io.ErrShortWrite, io.EOF},
//保留字符串中的强制错误。
		{baseAlert, baseAlertEncoded, pver, 70, io.ErrShortWrite, io.EOF},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		w := newFixedWriter(test.max)
		err := test.in.Serialize(w, test.pver)
		if reflect.TypeOf(err) != reflect.TypeOf(test.writeErr) {
			t.Errorf("Alert.Serialize #%d wrong error got: %v, want: %v",
				i, err, test.writeErr)
			continue
		}

		var alert Alert
		r := newFixedReader(test.max, test.buf)
		err = alert.Deserialize(r, test.pver)
		if reflect.TypeOf(err) != reflect.TypeOf(test.readErr) {
			t.Errorf("Alert.Deserialize #%d wrong error got: %v, want: %v",
				i, err, test.readErr)
			continue
		}
	}

//溢出setcancel中的最大元素数
//maxCountSetCancel+1==8388575==\xdf\xff\x7f\x00
//替换字节29-33
	badAlertEncoded := []byte{
0x01, 0x00, 0x00, 0x00, 0x50, 0x6e, 0xb2, 0x4f, 0x00, 0x00, 0x00, 0x00, 0x4c, 0x9e, 0x93, 0x51, //……零件号……L.Q
0x00, 0x00, 0x00, 0x00, 0xf7, 0x03, 0x00, 0x00, 0xf5, 0x03, 0x00, 0x00, 0xfe, 0xdf, 0xff, 0x7f, //…………………
0x00, 0x00, 0x00, 0x00, 0x00, 0x97, 0x9e, 0x00, 0x00, 0x01, 0x0f, 0x2f, 0x53, 0x61, 0x74, 0x6f, //………/Sato_
0x73, 0x68, 0x69, 0x3a, 0x30, 0x2e, 0x37, 0x2e, 0x32, 0x2f, 0x88, 0x13, 0x00, 0x00, 0x00, 0x06, //时：0.7.2/………
0x55, 0x52, 0x47, 0x45, 0x4e, 0x54, 0x00, //紧急。
	}
	var alert Alert
	r := bytes.NewReader(badAlertEncoded)
	err := alert.Deserialize(r, pver)
	if _, ok := err.(*MessageError); !ok {
		t.Errorf("Alert.Deserialize wrong error got: %T, want: %T",
			err, MessageError{})
	}

//溢出setsubver中的最大元素数
//maxCountSetSubver+1==131071+1==\x00\x00\x02\x00
//替换字节42-46
	badAlertEncoded = []byte{
0x01, 0x00, 0x00, 0x00, 0x50, 0x6e, 0xb2, 0x4f, 0x00, 0x00, 0x00, 0x00, 0x4c, 0x9e, 0x93, 0x51, //……零件号……L.Q
0x00, 0x00, 0x00, 0x00, 0xf7, 0x03, 0x00, 0x00, 0xf5, 0x03, 0x00, 0x00, 0x01, 0xf6, 0x03, 0x00, //…………………
0x00, 0x00, 0x00, 0x00, 0x00, 0x97, 0x9e, 0x00, 0x00, 0xfe, 0x00, 0x00, 0x02, 0x00, 0x74, 0x6f, //………/Sato_
0x73, 0x68, 0x69, 0x3a, 0x30, 0x2e, 0x37, 0x2e, 0x32, 0x2f, 0x88, 0x13, 0x00, 0x00, 0x00, 0x06, //时：0.7.2/………
0x55, 0x52, 0x47, 0x45, 0x4e, 0x54, 0x00, //紧急。
	}
	r = bytes.NewReader(badAlertEncoded)
	err = alert.Deserialize(r, pver)
	if _, ok := err.(*MessageError); !ok {
		t.Errorf("Alert.Deserialize wrong error got: %T, want: %T",
			err, MessageError{})
	}
}
