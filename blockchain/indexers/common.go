
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

/*

**/

package indexers

import (
	"encoding/binary"
	"errors"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcutil"
)

var (
//字节顺序是用于序列化数字的首选字节顺序
//用于存储在数据库中的字段。
	byteOrder = binary.LittleEndian

//
//用户请求的中断。
	errInterruptRequested = errors.New("interrupt requested")
)

//
//
type NeedsInputser interface {
	NeedsInputs() bool
}

//
//索引管理器，如此包提供的管理器类型。
type Indexer interface {
//键以字节片形式返回索引的键。
	Key() []byte

//name返回索引的可读名称。
	Name() string

//当索引器管理器确定索引需要时调用create
//
	Create(dbTx database.Tx) error

//
//
//每次加载，包括刚刚创建索引的情况。
	Init() error

//当新块已连接到
//主链。在一个块中花费的输出集也被传入
//
//必修的。
	ConnectBlock(database.Tx, *btcutil.Block, []blockchain.SpentTxOut) error

//断开块从断开时调用disconnectblock
//主链。在
//还返回此块，以便索引器可以清除以前的索引
//
	DisconnectBlock(database.Tx, *btcutil.Block, []blockchain.SpentTxOut) error
}

//断言错误标识指示内部代码一致性的错误
//问题，并应被视为一个关键和不可恢复的错误。
type AssertError string

//
//错误接口。
func (e AssertError) Error() string {
	return "assertion failed: " + string(e)
}

//errDeserialize表示反序列化时遇到问题
//数据。
type errDeserialize string

//错误实现错误接口。
func (e errDeserialize) Error() string {
	return string(e)
}

//IsDeserializer返回传递的错误是否为errDeserialize
//错误。
func isDeserializeErr(err error) bool {
	_, ok := err.(errDeserialize)
	return ok
}

//InternalBucket是对数据库Bucket的抽象。它被用来制造
//代码更容易测试，因为它只允许测试中的模拟对象
//实现这些函数，而不是数据库所支持的一切。
type internalBucket interface {
	Get(key []byte) []byte
	Put(key []byte, value []byte) error
	Delete(key []byte) error
}

//当提供的通道关闭时，interruptrequested返回true。
//
//
func interruptRequested(interrupted <-chan struct{}) bool {
	select {
	case <-interrupted:
		return true
	default:
	}

	return false
}
