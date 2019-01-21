
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
	"strconv"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

const (
//txversion是当前支持的最新事务版本。
	TxVersion = 1

//maxtxInSequenceNum是序列字段的最大序列号
//事务的输入可以是。
	MaxTxInSequenceNum uint32 = 0xffffffff

//MaxPrevOutIndex是前一个索引字段的最大索引
//输出点可以是。
	MaxPrevOutIndex uint32 = 0xffffffff

//SequenceLockTimeDisabled是一个标志，如果在事务上设置
//输入的序列号，序列号将不被解释
//作为相对锁定时间。
	SequenceLockTimeDisabled = 1 << 31

//SequenceLockTimeIsSeconds是一个标志，如果在事务上设置
//输入的序列号，相对锁定时间的单位为512
//秒。
	SequenceLockTimeIsSeconds = 1 << 22

//SequenceLockTimeMask是提取相对锁定时间的掩码
//当对事务输入序列号进行屏蔽时。
	SequenceLockTimeMask = 0x0000ffff

//SequenceLockTimeGranularity是定义的基于时间的粒度
//基于秒的相对时间锁定。从秒转换时
//到一个序列号，值被这个量右移，
//因此，512或2^9中的相对时间锁的粒度
//秒。强制的相对锁定时间是512秒的倍数。
	SequenceLockTimeGranularity = 9

//defaulttxinoutalloc是用于
//事务输入和输出。阵列将根据需要动态增长，
//但这个数字的目的是为
//典型事务中的输入和输出不需要增加
//多次备份数组。
	defaultTxInOutAlloc = 15

//MintXinPayload是事务输入的最小负载大小。
//previousOutpoint.hash+previousOutpoint.index 4 bytes+varint for
//签名脚本长度1字节+序列4字节。
	minTxInPayload = 9 + chainhash.HashSize

//MaxTxInPermessage是指
//一个符合消息的事务可能有。
	maxTxInPerMessage = (MaxMessagePayload / minTxInPayload) + 1

//MintXOutPayload是事务输出的最小负载大小。
//值8字节+pkscript长度1字节的变量。
	MinTxOutPayload = 9

//MaxTxOutperMessage是
//一个符合消息的事务可能有。
	maxTxOutPerMessage = (MaxMessagePayload / MinTxOutPayload) + 1

//MintXpayLoad是事务的最小有效负载大小。注释
//任何实际可用的交易必须至少有一个
//输入或输出，但这是在更高层强制执行的规则，所以
//这里特意不包括。
//版本4字节+变量事务输入数量1字节+变量
//事务输出数1字节+锁定时间4字节+最小输入
//有效载荷+最小输出有效载荷。
	minTxPayload = 10

//FreelistMaxScriptSize是空闲列表中每个缓冲区的大小
//用于在脚本
//连接成一个连续的缓冲区。已选择此值
//因为它的规模是绝大多数人的两倍多
//所有“标准”脚本。较大的脚本仍在反序列化
//正确地说，因为自由列表将被简单地绕过。
	freeListMaxScriptSize = 512

//FreelistMaxItems是要保留在可用列表中的缓冲区数
//用于脚本反序列化。此值最多允许100
//每个事务的脚本同时被125反序列化
//同龄人。因此，空闲列表的峰值使用率为12500*512=
//6400000字节。
	freeListMaxItems = 12500

//MaxWitnessItemsPerInput是要执行以下操作的见证项的最大数目
//读取单个txin的见证数据。这个数字是
//使用证人编码的可能下限派生
//项：长度为1字节+见证项本身为1字节，或两个
//字节。然后将该值除以当前允许的最大值
//交易的“成本”。
	maxWitnessItemsPerInput = 500000

//maxwitnessitemsize是在
//输入的见证数据。这个数字来源于
//对于脚本验证，每个推送到堆栈上的项必须小于
//超过10K字节。
	maxWitnessItemSize = 11000
)

//见证标记字节是一对特定于见证编码的字节。如果
//此序列被编码，则表示事务具有Iwtness
//数据。第一个字节是一个始终为0x00的标记字节，允许解码器
//区分带证人的序列化事务和常规（遗留）事务
//一个。第二个字节是标志字段，此时该字段始终为0x01，
//但将来可能会延长，以适应辅助性非承诺
//领域。
var witessMarkerBytes = []byte{0x00, 0x01}

//ScriptFreelist定义了一个字节片的自由列表（最多为最大数量
//由freelistmaxitems常量定义），根据
//FreelistMaxScriptSize常量。它用于为
//反序列化脚本以大大减少分配的数量
//必修的。
//
//调用方可以通过调用borrow从空闲列表中获取缓冲区。
//函数，使用完后应通过返回函数返回。
type scriptFreeList chan []byte

//borrow从空闲列表返回一个字节片，其长度根据
//提供尺寸。如果有可用的项目，将分配新的缓冲区。
//
//当大小大于自由列表中项目允许的最大大小时
//分配并返回适当大小的新缓冲区。它是安全的
//尝试通过返回函数返回所述缓冲区
//被忽略并允许进入垃圾收集器。
func (c scriptFreeList) Borrow(size uint64) []byte {
	if size > freeListMaxScriptSize {
		return make([]byte, size)
	}

	var buf []byte
	select {
	case buf = <-c:
	default:
		buf = make([]byte, freeListMaxScriptSize)
	}
	return buf[:size]
}

//返回将提供的字节片放回具有cap的空闲列表中
//预期长度的。预计通过以下方式获得缓冲区：
//借用功能。任何大小不合适的切片，例如
//当其大小大于允许的最大可用列表项大小时
//只是被忽略了，所以他们可以去垃圾收集器。
func (c scriptFreeList) Return(buf []byte) {
//忽略返回的任何缓冲区，这些缓冲区不是
//免费列表。
	if cap(buf) != freeListMaxScriptSize {
		return
	}

//当缓冲区未满时，将其返回到空闲列表。否则让
//它是垃圾收集。
	select {
	case c <- buf:
	default:
//把它交给垃圾收集器。
	}
}

//创建用于脚本反序列化的并发安全空闲列表。AS
//在前面的描述中，这个自由列表被维护为显著减少
//分配数。
var scriptPool scriptFreeList = make(chan []byte, freeListMaxItems)

//Outpoint定义用于跟踪上一个比特币数据类型
//事务输出。
type OutPoint struct {
	Hash  chainhash.Hash
	Index uint32
}

//new outpoint返回一个新的比特币交易outpoint point，
//提供哈希和索引。
func NewOutPoint(hash *chainhash.Hash, index uint32) *OutPoint {
	return &OutPoint{
		Hash:  *hash,
		Index: index,
	}
}

//字符串以可读形式返回输出点“hash:index”。
func (o OutPoint) String() string {
//为哈希字符串、冒号和10位数字分配足够的空间。虽然
//书写时，位数不能大于
//maxtxoutpermessage的十进制表示的长度，
//未来最大消息有效负载可能会增加，并且
//优化可能会被忽略，因此请为10位小数分配空间
//数字，适合任何uint32。
	buf := make([]byte, 2*chainhash.HashSize+1, 2*chainhash.HashSize+1+10)
	copy(buf, o.Hash.String())
	buf[2*chainhash.HashSize] = ':'
	buf = strconv.AppendUint(buf, uint64(o.Index), 10)
	return string(buf)
}

//txin定义比特币交易输入。
type TxIn struct {
	PreviousOutPoint OutPoint
	SignatureScript  []byte
	Witness          TxWitness
	Sequence         uint32
}

//serializesize返回序列化
//事务输入。
func (t *TxIn) SerializeSize() int {
//输出点哈希32字节+输出点索引4字节+序列4字节+
//SignatureScript+长度的序列化变量大小
//签名描述字节。
	return 40 + VarIntSerializeSize(uint64(len(t.SignatureScript))) +
		len(t.SignatureScript)
}

//newtxin返回一个新的比特币交易输入
//上一个输出点和签名脚本的默认序列为
//最大x频率。
func NewTxIn(prevOut *OutPoint, signatureScript []byte, witness [][]byte) *TxIn {
	return &TxIn{
		PreviousOutPoint: *prevOut,
		SignatureScript:  signatureScript,
		Witness:          witness,
		Sequence:         MaxTxInSequenceNum,
	}
}

//txwitness定义txin的见证。证人应被解释为
//字节片的一片，或一个或多个元素的堆栈。
type TxWitness [][]byte

//serializesize返回序列化
//事务输入的见证。
func (t TxWitness) SerializeSize() int {
//表示证人所拥有的元素数量的变量。
	n := VarIntSerializeSize(uint64(len(t)))

//对于证人中的每个元素，我们需要一个变量来表示
//元素的大小，最后是元素的字节数
//它本身包括。
	for _, witItem := range t {
		n += VarIntSerializeSize(uint64(len(witItem)))
		n += len(witItem)
	}

	return n
}

//txout定义比特币交易输出。
type TxOut struct {
	Value    int64
	PkScript []byte
}

//serializesize返回序列化
//事务输出。
func (t *TxOut) SerializeSize() int {
//值8字节+pkscript+长度的序列化变量大小
//PKScript字节。
	return 8 + VarIntSerializeSize(uint64(len(t.PkScript))) + len(t.PkScript)
}

//newtxout返回带有
//事务值和公钥脚本。
func NewTxOut(value int64, pkScript []byte) *TxOut {
	return &TxOut{
		Value:    value,
		PkScript: pkScript,
	}
}

//MSGTX实现消息接口并表示比特币Tx消息。
//它用于响应getdata传递事务信息
//给定事务的消息（msggetdata）。
//
//使用addtxin和addtxout函数建立事务列表
//输入和输出。
type MsgTx struct {
	Version  int32
	TxIn     []*TxIn
	TxOut    []*TxOut
	LockTime uint32
}

//addtxin将事务输入添加到消息中。
func (msg *MsgTx) AddTxIn(ti *TxIn) {
	msg.TxIn = append(msg.TxIn, ti)
}

//addtxout向消息添加事务输出。
func (msg *MsgTx) AddTxOut(to *TxOut) {
	msg.TxOut = append(msg.TxOut, to)
}

//TxHash为事务生成哈希。
func (msg *MsgTx) TxHash() chainhash.Hash {
//对事务进行编码，并在结果上计算double sha256。
//忽略错误返回，因为编码唯一可能失败的方法是
//内存不足或指针为零，这两种情况都会
//导致运行时恐慌。
	buf := bytes.NewBuffer(make([]byte, 0, msg.SerializeSizeStripped()))
	_ = msg.SerializeNoWitness(buf)
	return chainhash.DoubleHashH(buf.Bytes())
}

//WitnessHash生成根据
//bip0141和bip0144中定义的新见证序列。决赛
//输出在所有证人的独立证人承诺范围内使用。
//在一个街区内。如果事务没有见证数据，则见证哈希，
//与它的txid相同。
func (msg *MsgTx) WitnessHash() chainhash.Hash {
	if msg.HasWitness() {
		buf := bytes.NewBuffer(make([]byte, 0, msg.SerializeSize()))
		_ = msg.Serialize(buf)
		return chainhash.DoubleHashH(buf.Bytes())
	}

	return msg.TxHash()
}

//复制创建事务的深度副本，以便原始事务不会
//在操作副本时修改。
func (msg *MsgTx) Copy() *MsgTx {
//创建新的Tx，从复制基元值开始，并留出空间
//用于事务输入和输出。
	newTx := MsgTx{
		Version:  msg.Version,
		TxIn:     make([]*TxIn, 0, len(msg.TxIn)),
		TxOut:    make([]*TxOut, 0, len(msg.TxOut)),
		LockTime: msg.LockTime,
	}

//深度复制旧的txin数据。
	for _, oldTxIn := range msg.TxIn {
//深度复制以前的输出点。
		oldOutPoint := oldTxIn.PreviousOutPoint
		newOutPoint := OutPoint{}
		newOutPoint.Hash.SetBytes(oldOutPoint.Hash[:])
		newOutPoint.Index = oldOutPoint.Index

//深度复制旧的签名脚本。
		var newScript []byte
		oldScript := oldTxIn.SignatureScript
		oldScriptLen := len(oldScript)
		if oldScriptLen > 0 {
			newScript = make([]byte, oldScriptLen)
			copy(newScript, oldScript[:oldScriptLen])
		}

//使用深度复制的数据创建新的txin。
		newTxIn := TxIn{
			PreviousOutPoint: newOutPoint,
			SignatureScript:  newScript,
			Sequence:         oldTxIn.Sequence,
		}

//如果交易是见证，那么也复制
//目击者。
		if len(oldTxIn.Witness) != 0 {
//深度复制旧的证人数据。
			newTxIn.Witness = make([][]byte, len(oldTxIn.Witness))
			for i, oldItem := range oldTxIn.Witness {
				newItem := make([]byte, len(oldItem))
				copy(newItem, oldItem)
				newTxIn.Witness[i] = newItem
			}
		}

//最后，附加这个完全复制的txin。
		newTx.TxIn = append(newTx.TxIn, &newTxIn)
	}

//深度复制旧的txout数据。
	for _, oldTxOut := range msg.TxOut {
//深度复制旧pkscript
		var newScript []byte
		oldScript := oldTxOut.PkScript
		oldScriptLen := len(oldScript)
		if oldScriptLen > 0 {
			newScript = make([]byte, oldScriptLen)
			copy(newScript, oldScript[:oldScriptLen])
		}

//使用深度复制的数据创建新的txout并将其附加到
//新的TX。
		newTxOut := TxOut{
			Value:    oldTxOut.Value,
			PkScript: newScript,
		}
		newTx.TxOut = append(newTx.TxOut, &newTxOut)
	}

	return &newTx
}

//btcdecode使用比特币协议编码将r解码到接收器中。
//这是消息接口实现的一部分。
//请参阅反序列化以解码存储到磁盘的事务，例如
//数据库，而不是从网络解码事务。
func (msg *MsgTx) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	version, err := binarySerializer.Uint32(r, littleEndian)
	if err != nil {
		return err
	}
	msg.Version = int32(version)

	count, err := ReadVarInt(r, pver)
	if err != nil {
		return err
	}

//计数为零（意味着非初始化对象没有txin's）表示
//这是一个包含见证数据的事务。
	var flag [1]byte
	if count == 0 && enc == WitnessEncoding {
//接下来，我们需要读取标志，它是一个单字节。
		if _, err = io.ReadFull(r, flag[:]); err != nil {
			return err
		}

//目前，标志必须是0x01。在未来的其他
//可能支持标志类型。
		if flag[0] != 0x01 {
			str := fmt.Sprintf("witness tx but flag byte is %x", flag)
			return messageError("MsgTx.BtcDecode", str)
		}

//通过对隔离的证人特定字段进行解码，我们可以
//现在读取实际的txin计数。
		count, err = ReadVarInt(r, pver)
		if err != nil {
			return err
		}
	}

//阻止可能无法放入的更多输入事务
//消息。可能导致记忆衰竭和恐慌。
//在这个计数上没有一个健全的上限。
	if count > uint64(maxTxInPerMessage) {
		str := fmt.Sprintf("too many input transactions to fit into "+
			"max message size [count %d, max %d]", count,
			maxTxInPerMessage)
		return messageError("MsgTx.BtcDecode", str)
	}

//返回脚本缓冲区是返回
//当有任何反序列化时从池中借用
//错误。只有在最后一步之前调用
//用连续缓冲区中的位置替换脚本，并
//归还他们。
	returnScriptBuffers := func() {
		for _, txIn := range msg.TxIn {
			if txIn == nil {
				continue
			}

			if txIn.SignatureScript != nil {
				scriptPool.Return(txIn.SignatureScript)
			}

			for _, witnessElem := range txIn.Witness {
				if witnessElem != nil {
					scriptPool.Return(witnessElem)
				}
			}
		}
		for _, txOut := range msg.TxOut {
			if txOut == nil || txOut.PkScript == nil {
				continue
			}
			scriptPool.Return(txOut.PkScript)
		}
	}

//反序列化输入。
	var totalScriptSize uint64
	txIns := make([]TxIn, count)
	msg.TxIn = make([]*TxIn, count)
	for i := uint64(0); i < count; i++ {
//如果脚本缓冲区被借用，现在就设置指针。
//出错时需要返回池。
		ti := &txIns[i]
		msg.TxIn[i] = ti
		err = readTxIn(r, pver, msg.Version, ti)
		if err != nil {
			returnScriptBuffers()
			return err
		}
		totalScriptSize += uint64(len(ti.SignatureScript))
	}

	count, err = ReadVarInt(r, pver)
	if err != nil {
		returnScriptBuffers()
		return err
	}

//阻止可能无法放入的更多输出事务
//消息。可能导致记忆衰竭和恐慌。
//在这个计数上没有一个健全的上限。
	if count > uint64(maxTxOutPerMessage) {
		returnScriptBuffers()
		str := fmt.Sprintf("too many output transactions to fit into "+
			"max message size [count %d, max %d]", count,
			maxTxOutPerMessage)
		return messageError("MsgTx.BtcDecode", str)
	}

//反序列化输出。
	txOuts := make([]TxOut, count)
	msg.TxOut = make([]*TxOut, count)
	for i := uint64(0); i < count; i++ {
//如果脚本缓冲区被借用，现在就设置指针。
//出错时需要返回池。
		to := &txOuts[i]
		msg.TxOut[i] = to
		err = readTxOut(r, pver, msg.Version, to)
		if err != nil {
			returnScriptBuffers()
			return err
		}
		totalScriptSize += uint64(len(to.PkScript))
	}

//如果此时事务的标志字节不是0x00，则为一个或
//它的更多输入有伴随的见证数据。
	if flag[0] != 0 && enc == WitnessEncoding {
		for _, txin := range msg.TxIn {
//对于每个输入，见证被编码为堆栈
//一个或多个项目。因此，我们首先读到
//对堆栈项数进行编码的变量。
			witCount, err := ReadVarInt(r, pver)
			if err != nil {
				returnScriptBuffers()
				return err
			}

//通过以下方式防止可能的内存耗尽攻击：
//将witcount值限制为健全的上限。
			if witCount > maxWitnessItemsPerInput {
				returnScriptBuffers()
				str := fmt.Sprintf("too many witness items to fit "+
					"into max message size [count %d, max %d]",
					witCount, maxWitnessItemsPerInput)
				return messageError("MsgTx.BtcDecode", str)
			}

//然后对于WITCOUNT堆栈项的数目，每个项
//具有可变长度前缀，后跟见证人
//项目本身。
			txin.Witness = make([][]byte, witCount)
			for j := uint64(0); j < witCount; j++ {
				txin.Witness[j], err = readScript(r, pver,
					maxWitnessItemSize, "script witness item")
				if err != nil {
					returnScriptBuffers()
					return err
				}
				totalScriptSize += uint64(len(txin.Witness[j]))
			}
		}
	}

	msg.LockTime, err = binarySerializer.Uint32(r, littleEndian)
	if err != nil {
		returnScriptBuffers()
		return err
	}

//创建一个分配来存放所有脚本并设置每个脚本
//输入签名脚本并将公钥脚本输出到
//整个连续缓冲区的适当子片。然后，返回
//每个单独的脚本缓冲区都返回到池中，以便可以重用它们
//用于将来的反序列化。这样做是因为它非常重要
//减少垃圾收集器需要分配的数量
//从而提高性能并大幅降低
//否则需要保留的运行时开销量
//追踪数以百万计的小额拨款。
//
//注意：调用ReturnScriptBuffers闭包不再有效
//在这些代码块运行之后，因为它已经完成，并且
//事务输入和输出中的脚本不再指向
//缓冲器。
	var offset uint64
	scripts := make([]byte, totalScriptSize)
	for i := 0; i < len(msg.TxIn); i++ {
//将签名脚本复制到
//适当的偏移。
		signatureScript := msg.TxIn[i].SignatureScript
		copy(scripts[offset:], signatureScript)

//将事务输入的签名脚本重置为
//脚本所在的连续缓冲区的切片。
		scriptSize := uint64(len(signatureScript))
		end := offset + scriptSize
		msg.TxIn[i].SignatureScript = scripts[offset:end:end]
		offset += scriptSize

//将临时脚本缓冲区返回池。
		scriptPool.Return(signatureScript)

		for j := 0; j < len(msg.TxIn[i].Witness); j++ {
//为此复制见证堆栈中的每个项
//在适当的
//抵消。
			witnessElem := msg.TxIn[i].Witness[j]
			copy(scripts[offset:], witnessElem)

//将堆栈中的见证项重置为切片
//证人居住的相邻缓冲区。
			witnessElemSize := uint64(len(witnessElem))
			end := offset + witnessElemSize
			msg.TxIn[i].Witness[j] = scripts[offset:end:end]
			offset += witnessElemSize

//返回用于见证堆栈的临时缓冲区
//项目到池。
			scriptPool.Return(witnessElem)
		}
	}
	for i := 0; i < len(msg.TxOut); i++ {
//将公钥脚本复制到
//适当的偏移。
		pkScript := msg.TxOut[i].PkScript
		copy(scripts[offset:], pkScript)

//将事务输出的公钥脚本重置为
//脚本所在的连续缓冲区的切片。
		scriptSize := uint64(len(pkScript))
		end := offset + scriptSize
		msg.TxOut[i].PkScript = scripts[offset:end:end]
		offset += scriptSize

//将临时脚本缓冲区返回池。
		scriptPool.Return(pkScript)
	}

	return nil
}

//反序列化使用格式将事务从R解码到接收器
//适用于数据库等长期存储，同时尊重
//事务中的版本字段。此函数与btcdecode不同
//其中，BTCDecode在发送比特币有线协议时对其进行解码。
//通过网络。电线编码在技术上可能有所不同，具体取决于
//协议版本，甚至不需要与
//存储的事务。在撰写此评论时，
//编码事务在两个实例中都是相同的，但有一个不同的
//区别和分离允许API足够灵活
//应对变化。
func (msg *MsgTx) Deserialize(r io.Reader) error {
//目前，有线编码没有区别
//在协议版本0和稳定的长期存储格式。AS
//因此，使用btcdecode。
	return msg.BtcDecode(r, 0, WitnessEncoding)
}

//反序列化enowesting将事务从r解码到接收器，其中
//R中的事务编码格式不能使用新的
//为编码包含见证数据的事务而创建的序列化格式
//输入之内。
func (msg *MsgTx) DeserializeNoWitness(r io.Reader) error {
	return msg.BtcDecode(r, 0, BaseEncoding)
}

//btcencode使用比特币协议编码将接收器编码为w。
//这是消息接口实现的一部分。
//有关要存储到磁盘的编码事务（如
//数据库，而不是对线的事务进行编码。
func (msg *MsgTx) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	err := binarySerializer.PutUint32(w, littleEndian, uint32(msg.Version))
	if err != nil {
		return err
	}

//如果编码版本设置为见证编码，则
//msgtx的字段不是0x00，则表示事务
//将使用新的证人包容性结构进行编码
//在BIP014中定义。
	doWitness := enc == WitnessEncoding && msg.HasWitness()
	if doWitness {
//在txn的version字段之后，我们还包括两个
//特定于见证编码的字节。第一个字节是
//始终为0x00标记字节，允许解码器
//区分带证人的序列化事务与
//常规（传统）的。第二个字节是标志字段，
//目前它始终是0x01，但可以扩展到
//未来可容纳辅助非提交字段。
		if _, err := w.Write(witessMarkerBytes); err != nil {
			return err
		}
	}

	count := uint64(len(msg.TxIn))
	err = WriteVarInt(w, pver, count)
	if err != nil {
		return err
	}

	for _, ti := range msg.TxIn {
		err = writeTxIn(w, pver, msg.Version, ti)
		if err != nil {
			return err
		}
	}

	count = uint64(len(msg.TxOut))
	err = WriteVarInt(w, pver, count)
	if err != nil {
		return err
	}

	for _, to := range msg.TxOut {
		err = WriteTxOut(w, pver, msg.Version, to)
		if err != nil {
			return err
		}
	}

//如果此交易是见证交易，并且见证人
//需要编码，然后对每个输入的见证进行编码
//在事务中。
	if doWitness {
		for _, ti := range msg.TxIn {
			err = writeTxWitness(w, pver, msg.Version, ti.Witness)
			if err != nil {
				return err
			}
		}
	}

	return binarySerializer.PutUint32(w, littleEndian, msg.LockTime)
}

//如果事务中没有任何输入，则hasWitness返回false
//包含见证数据，否则为真或假。
func (msg *MsgTx) HasWitness() bool {
	for _, txIn := range msg.TxIn {
		if len(txIn.Witness) != 0 {
			return true
		}
	}

	return false
}

//序列化使用适合的格式将事务编码为w
//长期存储，如数据库，同时考虑中的版本字段
//交易。此函数与btcencode不同，因为btcencode
//将事务编码为比特币有线协议以便发送
//通过网络。电线编码在技术上可能有所不同，具体取决于
//协议版本，甚至不需要与
//存储的事务。在撰写此评论时，
//编码事务在两个实例中都是相同的，但有一个不同的
//区别和分离允许API足够灵活
//应对变化。
func (msg *MsgTx) Serialize(w io.Writer) error {
//目前，有线编码没有区别
//在协议版本0和稳定的长期存储格式。AS
//结果，使用btcencode。
//
//将见证编码的编码类型传递给msgtx的btcencode
//表示交易的见证人（如果有的话）应该是
//根据中定义的新序列化结构进行序列化
//BIP0144
	return msg.BtcEncode(w, 0, WitnessEncoding)
}

//serialinowitness以相同的方式将事务编码为w
//但是，序列化，即使源事务具有带见证的输入
//数据，仍将使用旧的序列化格式。
func (msg *MsgTx) SerializeNoWitness(w io.Writer) error {
	return msg.BtcEncode(w, 0, BaseEncoding)
}

//baseSize返回事务的序列化大小，不计算
//任何见证数据。
func (msg *MsgTx) baseSize() int {
//版本4字节+锁定时间4字节+的序列化变量大小
//事务输入和输出的数目。
	n := 8 + VarIntSerializeSize(uint64(len(msg.TxIn))) +
		VarIntSerializeSize(uint64(len(msg.TxOut)))

	for _, txIn := range msg.TxIn {
		n += txIn.SerializeSize()
	}

	for _, txOut := range msg.TxOut {
		n += txOut.SerializeSize()
	}

	return n
}

//serializesize返回序列化
//交易。
func (msg *MsgTx) SerializeSize() int {
	n := msg.baseSize()

	if msg.HasWitness() {
//标记字段和标志字段占用另外两个字节。
		n += 2

//此外，每个
//每个TXIN的见证人。
		for _, txin := range msg.TxIn {
			n += txin.Witness.SerializeSize()
		}
	}

	return n
}

//SerializeSizeStripped返回序列化所需的字节数
//交易，不包括任何包含的见证数据。
func (msg *MsgTx) SerializeSizeStripped() int {
	return msg.baseSize()
}

//命令返回消息的协议命令字符串。这是一部分
//消息接口实现。
func (msg *MsgTx) Command() string {
	return CmdTx
}

//maxpayloadLength返回有效负载的最大长度
//接收器。这是消息接口实现的一部分。
func (msg *MsgTx) MaxPayloadLength(pver uint32) uint32 {
	return MaxBlockPayload
}

//pkscriptlocs返回一个包含每个公钥脚本开头的切片
//在原始序列化事务中。呼叫者可以轻松获得
//在脚本上使用len的每个脚本的长度
//适当的事务输出条目。
func (msg *MsgTx) PkScriptLocs() []int {
	numTxOut := len(msg.TxOut)
	if numTxOut == 0 {
		return nil
	}

//第一个的序列化事务中的起始偏移量
//事务输出为：
//
//版本4字节+序列化变量大小
//事务输入和输出+每个事务的序列化大小
//输入。
	n := 4 + VarIntSerializeSize(uint64(len(msg.TxIn))) +
		VarIntSerializeSize(uint64(numTxOut))

//如果此事务具有见证输入，则
//对于标记，需要考虑标志字节。
	if len(msg.TxIn) > 0 && msg.TxIn[0].Witness != nil {
		n += 2
	}

	for _, txIn := range msg.TxIn {
		n += txIn.SerializeSize()
	}

//为每个公钥脚本计算并设置适当的偏移量。
	pkScriptLocs := make([]int, numTxOut)
	for i, txOut := range msg.TxOut {
//事务输出中脚本的偏移量为：
//
//值8字节+长度为的序列化变量大小
//PkScript。
		n += 8 + VarIntSerializeSize(uint64(len(txOut.PkScript)))
		pkScriptLocs[i] = n
		n += len(txOut.PkScript)
	}

	return pkScriptLocs
}

//newmsgtx返回符合消息的新比特币Tx消息
//接口。返回实例有一个默认的txversion版本，
//没有事务输入或输出。此外，锁定时间设置为零
//指示事务立即有效，而不是在
//未来。
func NewMsgTx(version int32) *MsgTx {
	return &MsgTx{
		Version: version,
		TxIn:    make([]*TxIn, 0, defaultTxInOutAlloc),
		TxOut:   make([]*TxOut, 0, defaultTxInOutAlloc),
	}
}

//readoutpoint作为输出点从r读取下一个字节序列。
func readOutPoint(r io.Reader, pver uint32, version int32, op *OutPoint) error {
	_, err := io.ReadFull(r, op.Hash[:])
	if err != nil {
		return err
	}

	op.Index, err = binarySerializer.Uint32(r, littleEndian)
	return err
}

//WriteOutpoint将op编码为输出点的比特币协议编码
//到W
func writeOutPoint(w io.Writer, pver uint32, version int32, op *OutPoint) error {
	_, err := w.Write(op.Hash[:])
	if err != nil {
		return err
	}

	return binarySerializer.PutUint32(w, littleEndian, op.Index)
}

//readscript读取表示事务的变长字节数组
//脚本。它被编码为包含数组长度的变量
//后面是字节本身。如果长度为
//大于传递的maxallowed参数，该参数有助于防止
//记忆耗尽攻击和通过畸形信息强制恐慌。这个
//FieldName参数仅用于错误消息，因此它提供了更多
//错误中的上下文。
func readScript(r io.Reader, pver uint32, maxAllowed uint32, fieldName string) ([]byte, error) {
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
		return nil, messageError("readScript", str)
	}

	b := scriptPool.Borrow(count)
	_, err = io.ReadFull(r, b)
	if err != nil {
		scriptPool.Return(b)
		return nil, err
	}
	return b, nil
}

//readtxin从r中读取下一个字节序列作为事务输入
//（TXIN）。
func readTxIn(r io.Reader, pver uint32, version int32, ti *TxIn) error {
	err := readOutPoint(r, pver, version, &ti.PreviousOutPoint)
	if err != nil {
		return err
	}

	ti.SignatureScript, err = readScript(r, pver, MaxMessagePayload,
		"transaction input signature script")
	if err != nil {
		return err
	}

	return readElement(r, &ti.Sequence)
}

//writetxin将ti编码为交易的比特币协议编码
//输入（txin）到w。
func writeTxIn(w io.Writer, pver uint32, version int32, ti *TxIn) error {
	err := writeOutPoint(w, pver, version, &ti.PreviousOutPoint)
	if err != nil {
		return err
	}

	err = WriteVarBytes(w, pver, ti.SignatureScript)
	if err != nil {
		return err
	}

	return binarySerializer.PutUint32(w, littleEndian, ti.Sequence)
}

//readtxout从r中读取下一个字节序列作为事务输出
//（TXOUT）。
func readTxOut(r io.Reader, pver uint32, version int32, to *TxOut) error {
	err := readElement(r, &to.Value)
	if err != nil {
		return err
	}

	to.PkScript, err = readScript(r, pver, MaxMessagePayload,
		"transaction output public key script")
	return err
}

//writetxout编码到交易的比特币协议编码中
//输出（txout）到w。
//
//注意：为了允许txscript计算
//见证交易新叹息（BIP0143）。
func WriteTxOut(w io.Writer, pver uint32, version int32, to *TxOut) error {
	err := binarySerializer.PutUint64(w, littleEndian, uint64(to.Value))
	if err != nil {
		return err
	}

	return WriteVarBytes(w, pver, to.PkScript)
}

//WRITETXWITNESS为事务编码比特币协议编码
//输入W的见证。
func writeTxWitness(w io.Writer, pver uint32, version int32, wit [][]byte) error {
	err := WriteVarInt(w, pver, uint64(len(wit)))
	if err != nil {
		return err
	}
	for _, item := range wit {
		err = WriteVarBytes(w, pver, item)
		if err != nil {
			return err
		}
	}
	return nil
}
