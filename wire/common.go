
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
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

const (
//maxvarintpayload是可变长度整数的最大负载大小。
	MaxVarIntPayload = 9

//BinaryFreelistMaxItems是要保留在空闲中的缓冲区数
//用于二进制序列化和反序列化的列表。
	binaryFreeListMaxItems = 1024
)

var (
//littleendian是一个方便变量，因为binary.littleendian是
//相当长。
	littleEndian = binary.LittleEndian

//bigendian是一个方便变量，因为binary.bigendian
//长。
	bigEndian = binary.BigEndian
)

//BinaryFreelist定义了字节片的并发安全空闲列表（最多
//由binaryFreelistMaxItems常量定义的最大数目），具有
//上限为8（因此它最多支持一个uint64）。用于提供临时
//缓冲区，用于对基元数进行序列化和反序列化
//二进制编码以大大减少分配的数量
//必修的。
//
//为了方便起见，为每个无符号原语提供了函数
//从空闲列表自动获取缓冲区的整数，执行
//必要的二进制转换，读或写给定的IO.reader或
//然后将缓冲区返回到空闲列表。
type binaryFreeList chan []byte

//borrow返回自由列表中长度为8的字节片。一个新的
//如果空闲列表中没有可用的缓冲区，则分配缓冲区。
func (l binaryFreeList) Borrow() []byte {
	var buf []byte
	select {
	case buf = <-l:
	default:
		buf = make([]byte, 8)
	}
	return buf[:8]
}

//返回将提供的字节片放回空闲列表。缓冲器必须
//已通过借用函数获得，因此上限为8。
func (l binaryFreeList) Return(buf []byte) {
	select {
	case l <- buf:
	default:
//把它交给垃圾收集器。
	}
}

//uint8使用缓冲区从提供的读卡器中读取单个字节
//释放列表并将其作为uint8返回。
func (l binaryFreeList) Uint8(r io.Reader) (uint8, error) {
	buf := l.Borrow()[:1]
	if _, err := io.ReadFull(r, buf); err != nil {
		l.Return(buf)
		return 0, err
	}
	rv := buf[0]
	l.Return(buf)
	return rv, nil
}

//uint16使用缓冲区从提供的读卡器读取两个字节
//自由列表，使用提供的字节顺序将其转换为数字，然后返回
//生成的uint16。
func (l binaryFreeList) Uint16(r io.Reader, byteOrder binary.ByteOrder) (uint16, error) {
	buf := l.Borrow()[:2]
	if _, err := io.ReadFull(r, buf); err != nil {
		l.Return(buf)
		return 0, err
	}
	rv := byteOrder.Uint16(buf)
	l.Return(buf)
	return rv, nil
}

//uint32使用缓冲区从提供的读卡器读取四个字节
//自由列表，使用提供的字节顺序将其转换为数字，然后返回
//生成的uint32。
func (l binaryFreeList) Uint32(r io.Reader, byteOrder binary.ByteOrder) (uint32, error) {
	buf := l.Borrow()[:4]
	if _, err := io.ReadFull(r, buf); err != nil {
		l.Return(buf)
		return 0, err
	}
	rv := byteOrder.Uint32(buf)
	l.Return(buf)
	return rv, nil
}

//uint64使用缓冲区从提供的读卡器读取8个字节
//自由列表，使用提供的字节顺序将其转换为数字，然后返回
//生成的uint64。
func (l binaryFreeList) Uint64(r io.Reader, byteOrder binary.ByteOrder) (uint64, error) {
	buf := l.Borrow()[:8]
	if _, err := io.ReadFull(r, buf); err != nil {
		l.Return(buf)
		return 0, err
	}
	rv := byteOrder.Uint64(buf)
	l.Return(buf)
	return rv, nil
}

//putunit8将提供的uint8从空闲列表复制到缓冲区中，并
//将结果字节写入给定的写入程序。
func (l binaryFreeList) PutUint8(w io.Writer, val uint8) error {
	buf := l.Borrow()[:1]
	buf[0] = val
	_, err := w.Write(buf)
	l.Return(buf)
	return err
}

//putunt16使用给定的字节顺序将提供的uint16序列化为
//缓冲区，并将结果两个字节写入给定的
//作家。
func (l binaryFreeList) PutUint16(w io.Writer, byteOrder binary.ByteOrder, val uint16) error {
	buf := l.Borrow()[:2]
	byteOrder.PutUint16(buf, val)
	_, err := w.Write(buf)
	l.Return(buf)
	return err
}

//putunt32使用给定的字节顺序将提供的uint32序列化为
//从空闲列表中缓冲并将结果四个字节写入给定的
//作家。
func (l binaryFreeList) PutUint32(w io.Writer, byteOrder binary.ByteOrder, val uint32) error {
	buf := l.Borrow()[:4]
	byteOrder.PutUint32(buf, val)
	_, err := w.Write(buf)
	l.Return(buf)
	return err
}

//putunt64使用给定的字节顺序将提供的uint64序列化为
//从空闲列表中缓冲，并将结果8个字节写入给定的
//作家。
func (l binaryFreeList) PutUint64(w io.Writer, byteOrder binary.ByteOrder, val uint64) error {
	buf := l.Borrow()[:8]
	byteOrder.PutUint64(buf, val)
	_, err := w.Write(buf)
	l.Return(buf)
	return err
}

//Binaryserializer提供了一个用于序列化和
//正在对IO.readers和IO.writers之间的基元整数值进行反序列化。
var binarySerializer binaryFreeList = make(chan []byte, binaryFreeListMaxItems)

//erroncanonicalvarint是用于非规范的通用格式字符串
//编码的可变长度整数错误。
var errNonCanonicalVarInt = "non-canonical varint %x - discriminant %x must " +
	"encode a value greater than %x"

//uint32time表示用uint32编码的UNIX时间戳。它被用作
//一种向readelement函数发出如何将时间戳解码为go的信号的方法
//时间。时间，因为它是不明确的。
type uint32Time time.Time

//Int64Time表示用Int64编码的Unix时间戳。它被用作
//一种向readelement函数发出如何将时间戳解码为go的信号的方法
//时间。时间，因为它是不明确的。
type int64Time time.Time

//readelement使用little endian从r中读取下一个字节序列
//取决于所指构件的具体类型。
func readElement(r io.Reader, element interface{}) error {
//尝试通过fast读取基于具体类型的元素
//首先键入断言。
	switch e := element.(type) {
	case *int32:
		rv, err := binarySerializer.Uint32(r, littleEndian)
		if err != nil {
			return err
		}
		*e = int32(rv)
		return nil

	case *uint32:
		rv, err := binarySerializer.Uint32(r, littleEndian)
		if err != nil {
			return err
		}
		*e = rv
		return nil

	case *int64:
		rv, err := binarySerializer.Uint64(r, littleEndian)
		if err != nil {
			return err
		}
		*e = int64(rv)
		return nil

	case *uint64:
		rv, err := binarySerializer.Uint64(r, littleEndian)
		if err != nil {
			return err
		}
		*e = rv
		return nil

	case *bool:
		rv, err := binarySerializer.Uint8(r)
		if err != nil {
			return err
		}
		if rv == 0x00 {
			*e = false
		} else {
			*e = true
		}
		return nil

//unix时间戳编码为uint32。
	case *uint32Time:
		rv, err := binarySerializer.Uint32(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = uint32Time(time.Unix(int64(rv), 0))
		return nil

//Unix时间戳编码为Int64。
	case *int64Time:
		rv, err := binarySerializer.Uint64(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = int64Time(time.Unix(int64(rv), 0))
		return nil

//消息头校验和。
	case *[4]byte:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil

//消息头命令。
	case *[CommandSize]uint8:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil

//IP地址。
	case *[16]byte:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil

	case *chainhash.Hash:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil

	case *ServiceFlag:
		rv, err := binarySerializer.Uint64(r, littleEndian)
		if err != nil {
			return err
		}
		*e = ServiceFlag(rv)
		return nil

	case *InvType:
		rv, err := binarySerializer.Uint32(r, littleEndian)
		if err != nil {
			return err
		}
		*e = InvType(rv)
		return nil

	case *BitcoinNet:
		rv, err := binarySerializer.Uint32(r, littleEndian)
		if err != nil {
			return err
		}
		*e = BitcoinNet(rv)
		return nil

	case *BloomUpdateType:
		rv, err := binarySerializer.Uint8(r)
		if err != nil {
			return err
		}
		*e = BloomUpdateType(rv)
		return nil

	case *RejectCode:
		rv, err := binarySerializer.Uint8(r)
		if err != nil {
			return err
		}
		*e = RejectCode(rv)
		return nil
	}

//返回较慢的二进制文件。如果快速路径不可用，则读取
//上面。
	return binary.Read(r, littleEndian, element)
}

//readElements从r中读取多个项。它相当于多个
//调用readelement。
func readElements(r io.Reader, elements ...interface{}) error {
	for _, element := range elements {
		err := readElement(r, element)
		if err != nil {
			return err
		}
	}
	return nil
}

//writeElement将元素的小endian表示形式写入w。
func writeElement(w io.Writer, element interface{}) error {
//尝试通过fast根据具体类型编写元素
//首先键入断言。
	switch e := element.(type) {
	case int32:
		err := binarySerializer.PutUint32(w, littleEndian, uint32(e))
		if err != nil {
			return err
		}
		return nil

	case uint32:
		err := binarySerializer.PutUint32(w, littleEndian, e)
		if err != nil {
			return err
		}
		return nil

	case int64:
		err := binarySerializer.PutUint64(w, littleEndian, uint64(e))
		if err != nil {
			return err
		}
		return nil

	case uint64:
		err := binarySerializer.PutUint64(w, littleEndian, e)
		if err != nil {
			return err
		}
		return nil

	case bool:
		var err error
		if e {
			err = binarySerializer.PutUint8(w, 0x01)
		} else {
			err = binarySerializer.PutUint8(w, 0x00)
		}
		if err != nil {
			return err
		}
		return nil

//消息头校验和。
	case [4]byte:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil

//消息头命令。
	case [CommandSize]uint8:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil

//IP地址。
	case [16]byte:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil

	case *chainhash.Hash:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil

	case ServiceFlag:
		err := binarySerializer.PutUint64(w, littleEndian, uint64(e))
		if err != nil {
			return err
		}
		return nil

	case InvType:
		err := binarySerializer.PutUint32(w, littleEndian, uint32(e))
		if err != nil {
			return err
		}
		return nil

	case BitcoinNet:
		err := binarySerializer.PutUint32(w, littleEndian, uint32(e))
		if err != nil {
			return err
		}
		return nil

	case BloomUpdateType:
		err := binarySerializer.PutUint8(w, uint8(e))
		if err != nil {
			return err
		}
		return nil

	case RejectCode:
		err := binarySerializer.PutUint8(w, uint8(e))
		if err != nil {
			return err
		}
		return nil
	}

//返回较慢的二进制文件。如果快速路径不可用，则写入
//上面。
	return binary.Write(w, littleEndian, element)
}

//WriteElements将多个项写入w。它相当于多个
//调用WriteElement。
func writeElements(w io.Writer, elements ...interface{}) error {
	for _, element := range elements {
		err := writeElement(w, element)
		if err != nil {
			return err
		}
	}
	return nil
}

//readvarint从r中读取一个长度可变的整数，并将其作为uint64返回。
func ReadVarInt(r io.Reader, pver uint32) (uint64, error) {
	discriminant, err := binarySerializer.Uint8(r)
	if err != nil {
		return 0, err
	}

	var rv uint64
	switch discriminant {
	case 0xff:
		sv, err := binarySerializer.Uint64(r, littleEndian)
		if err != nil {
			return 0, err
		}
		rv = sv

//如果该值可能是
//使用更少的字节进行编码。
		min := uint64(0x100000000)
		if rv < min {
			return 0, messageError("ReadVarInt", fmt.Sprintf(
				errNonCanonicalVarInt, rv, discriminant, min))
		}

	case 0xfe:
		sv, err := binarySerializer.Uint32(r, littleEndian)
		if err != nil {
			return 0, err
		}
		rv = uint64(sv)

//如果该值可能是
//使用更少的字节进行编码。
		min := uint64(0x10000)
		if rv < min {
			return 0, messageError("ReadVarInt", fmt.Sprintf(
				errNonCanonicalVarInt, rv, discriminant, min))
		}

	case 0xfd:
		sv, err := binarySerializer.Uint16(r, littleEndian)
		if err != nil {
			return 0, err
		}
		rv = uint64(sv)

//如果该值可能是
//使用更少的字节进行编码。
		min := uint64(0xfd)
		if rv < min {
			return 0, messageError("ReadVarInt", fmt.Sprintf(
				errNonCanonicalVarInt, rv, discriminant, min))
		}

	default:
		rv = uint64(discriminant)
	}

	return rv, nil
}

//WRITEVARINT使用可变字节数将VAL序列化为W，具体取决于
//论其价值。
func WriteVarInt(w io.Writer, pver uint32, val uint64) error {
	if val < 0xfd {
		return binarySerializer.PutUint8(w, uint8(val))
	}

	if val <= math.MaxUint16 {
		err := binarySerializer.PutUint8(w, 0xfd)
		if err != nil {
			return err
		}
		return binarySerializer.PutUint16(w, littleEndian, uint16(val))
	}

	if val <= math.MaxUint32 {
		err := binarySerializer.PutUint8(w, 0xfe)
		if err != nil {
			return err
		}
		return binarySerializer.PutUint32(w, littleEndian, uint32(val))
	}

	err := binarySerializer.PutUint8(w, 0xff)
	if err != nil {
		return err
	}
	return binarySerializer.PutUint64(w, littleEndian, val)
}

//varintserializesize返回序列化所需的字节数
//作为可变长度整数的val。
func VarIntSerializeSize(val uint64) int {
//这个值足够小，可以用它自己来表示，所以
//只有1字节。
	if val < 0xfd {
		return 1
	}

//区分uint16的1字节加2字节。
	if val <= math.MaxUint16 {
		return 3
	}

//对于uint32，区分1个字节加4个字节。
	if val <= math.MaxUint32 {
		return 5
	}

//对于uint64，区分1字节加8字节。
	return 9
}

//readvarstring从r中读取一个可变长度的字符串，并将其作为go返回
//字符串。可变长度字符串编码为可变长度整数。
//包含字符串的长度，后跟表示
//字符串本身。如果长度大于
//最大块有效负载大小，因为它有助于防止内存耗尽
//攻击和通过畸形的消息强制恐慌。
func ReadVarString(r io.Reader, pver uint32) (string, error) {
	count, err := ReadVarInt(r, pver)
	if err != nil {
		return "", err
	}

//阻止大于最大值的可变长度字符串
//消息大小。可能导致记忆衰竭和
//在这个计数上没有健全上限的恐慌。
	if count > MaxMessagePayload {
		str := fmt.Sprintf("variable length string is too long "+
			"[count %d, max %d]", count, MaxMessagePayload)
		return "", messageError("ReadVarString", str)
	}

	buf := make([]byte, count)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

//writevarstring将str序列化为w，作为包含
//字符串的长度，后跟表示字符串的字节
//本身。
func WriteVarString(w io.Writer, pver uint32, str string) error {
	err := WriteVarInt(w, pver, uint64(len(str)))
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(str))
	return err
}

//readvarbytes读取变长字节数组。对字节数组进行编码
//作为包含数组长度和字节的变量
//他们自己。如果长度大于
//传递的maxallowed参数有助于防止内存耗尽
//攻击和通过畸形的消息强制恐慌。字段名
//参数仅用于错误消息，因此它在
//错误。
func ReadVarBytes(r io.Reader, pver uint32, maxAllowed uint32,
	fieldName string) ([]byte, error) {

	count, err := ReadVarInt(r, pver)
	if err != nil {
		return nil, err
	}

//防止字节数组大于最大消息大小。它会
//在没有理智的情况下可能导致记忆衰竭和恐慌。
//此计数的上限。
	if count > uint64(maxAllowed) {
		str := fmt.Sprintf("%s is larger than the max allowed size "+
			"[count %d, max %d]", fieldName, count, maxAllowed)
		return nil, messageError("ReadVarBytes", str)
	}

	b := make([]byte, count)
	_, err = io.ReadFull(r, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

//writevarbytes将变长字节数组序列化为w作为变量
//包含字节数，后跟字节本身。
func WriteVarBytes(w io.Writer, pver uint32, bytes []byte) error {
	slen := uint64(len(bytes))
	err := WriteVarInt(w, pver, slen)
	if err != nil {
		return err
	}

	_, err = w.Write(bytes)
	return err
}

//randomunt64返回密码随机的uint64值。这个
//未分析的版本主要采用读卡器来确保错误路径
//可以通过测试中的假阅读器来正确测试。
func randomUint64(r io.Reader) (uint64, error) {
	rv, err := binarySerializer.Uint64(r, bigEndian)
	if err != nil {
		return 0, err
	}
	return rv, nil
}

//randomunt64返回密码随机的uint64值。
func RandomUint64() (uint64, error) {
	return randomUint64(rand.Reader)
}
