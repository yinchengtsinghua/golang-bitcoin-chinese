
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

//这个接口的某些部分深受出色的BoltDB项目的启发。
//请访问https://github.com/boltdb/bolt，作者：Ben B.Johnson。

package database

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
)

//光标表示位于键/值对和嵌套存储桶上的光标
//桶。
//
//注意，在bucket更改和任何
//对存储桶的修改，但光标除外。删除，无效
//光标。无效后，必须重新定位光标或键
//并且返回的值可能是不可预测的。
type Cursor interface {
//bucket返回为其创建光标的bucket。
	Bucket() Bucket

//删除删除光标所在的当前键/值对
//使光标无效。
//
//接口合同至少保证以下错误
//be returned (other implementation-specific errors are possible):
//-如果在光标指向
//嵌套桶
//-errtxnotwritable（如果尝试对只读事务执行此操作）
//-如果事务已关闭，则返回errtxclosed
	Delete() error

//首先将光标定位在第一个键/值对上并返回
//这对是否存在。
	First() bool

//Last将光标定位在最后一个键/值对上并返回
//这对是否存在。
	Last() bool

//下一步将光标向前移动一个键/值对，并返回
//或者这对不存在。
	Next() bool

//prev将光标向后移动一个键/值对，并返回
//或者这对不存在。
	Prev() bool

//SEEK将光标定位在较大的第一个键/值对上。
//大于或等于已通过的查找键。是否返回
//对存在。
	Seek(seek []byte) bool

//键返回光标指向的当前键。
	Key() []byte

//Value returns the current value the cursor is pointing to.  本遗嘱
//嵌套存储桶为零。
	Value() []byte
}

//bucket表示键/值对的集合。
type Bucket interface {
//bucket用给定的键检索嵌套的bucket。返回零IF
//桶不存在。
	Bucket(key []byte) Bucket

//createBucket创建并返回具有给定
//关键。
//
//接口合同至少保证以下错误
//返回（可能存在其他特定于实现的错误）：
//-errbacketexists如果存储桶已经存在
//-如果密钥为空，则需要errbacketname
//-errcompatibleValue，如果密钥对于
//具体实施
//-errtxnotwritable（如果尝试对只读事务执行此操作）
//-如果事务已关闭，则返回errtxclosed
	CreateBucket(key []byte) (Bucket, error)

//CreateBacketifnotexists创建并返回一个新的嵌套bucket，其中
//给定的键（如果它不存在）。
//
//接口合同至少保证以下错误
//返回（可能存在其他特定于实现的错误）：
//-如果密钥为空，则需要errbacketname
//-errcompatibleValue，如果密钥对于
//具体实施
//-errtxnotwritable（如果尝试对只读事务执行此操作）
//-如果事务已关闭，则返回errtxclosed
	CreateBucketIfNotExists(key []byte) (Bucket, error)

//删除bucket删除具有给定键的嵌套bucket。这也
//包括移除所有嵌套存储桶和存储桶下的键
//删除。
//
//接口合同至少保证以下错误
//返回（可能存在其他特定于实现的错误）：
//-errbacketnotfound如果指定的存储桶不存在
//-errtxnotwritable（如果尝试对只读事务执行此操作）
//-如果事务已关闭，则返回errtxclosed
	DeleteBucket(key []byte) error

//foreach使用中的每个键/值对调用传递的函数
//桶。这不包括嵌套存储桶或键/值对
//在那些嵌套的桶中。
//
//警告：使用此函数进行迭代时更改数据是不安全的
//方法。这样做可能导致基础光标无效
//并返回意外的键和/或值。
//
//接口合同至少保证以下错误
//返回（可能存在其他特定于实现的错误）：
//-如果事务已关闭，则返回errtxclosed
//
//注意：此函数返回的切片仅在
//交易。在事务结束后尝试访问它们
//导致未定义的行为。此外，切片不能
//由调用者修改。这些约束会阻止其他数据
//复制并允许支持内存映射数据库实现。
	ForEach(func(k, v []byte) error) error

//前脚桶用每个键调用传递函数
//当前存储桶中的嵌套存储桶。这不包括
//在这些嵌套桶中嵌套桶。
//
//警告：使用此函数进行迭代时更改数据是不安全的
//方法。这样做可能导致基础光标无效
//并返回意外的键和/或值。
//
//接口合同至少保证以下错误
//返回（可能存在其他特定于实现的错误）：
//-如果事务已关闭，则返回errtxclosed
//
//注意：此函数返回的密钥仅在
//交易。在事务结束后尝试访问它们
//导致未定义的行为。此约束可防止
//数据复制并支持内存映射数据库
//实施。
	ForEachBucket(func(k []byte) error) error

//cursor返回一个新的光标，允许在bucket的
//键/值对和嵌套存储桶的前向或后向顺序。
//
//必须使用First、Last或Seek函数查找位置
//在调用next、prev、key或value函数之前。失败到
//这样做将产生与耗尽的光标相同的返回值，
//上一个和下一个函数为假，键和
//值函数。
	Cursor() Cursor

//可写返回bucket是否可写。
	Writable() bool

//PUT将指定的键/值对保存到存储桶中。做的钥匙
//已添加不存在的键，已存在的键是
//改写。
//
//接口合同至少保证以下错误
//返回（可能存在其他特定于实现的错误）：
//-如果密钥为空，则需要errkey
//-errcompatibleValue（如果密钥与现有存储桶相同）
//-errtxnotwritable（如果尝试对只读事务执行此操作）
//-如果事务已关闭，则返回errtxclosed
//
//注意：传递给这个函数的切片不能被修改。
//来电者。此约束阻止了对附加数据的要求
//复制并允许支持内存映射数据库实现。
	Put(key, value []byte) error

//get返回给定键的值。如果键为，则返回nil
//此存储桶中不存在。返回一个空切片，用于
//存在但未分配值。
//
//注意：此函数返回的值仅在
//交易。在事务结束后尝试访问它
//导致未定义的行为。此外，该值不能
//由调用者修改。这些约束会阻止其他数据
//复制并允许支持内存映射数据库实现。
	Get(key []byte) []byte

//删除从存储桶中删除指定的键。删除密钥
//不存在不会返回错误。
//
//接口合同至少保证以下错误
//返回（可能存在其他特定于实现的错误）：
//-如果密钥为空，则需要errkey
//-errcompatibleValue（如果密钥与现有存储桶相同）
//-errtxnotwritable（如果尝试对只读事务执行此操作）
//-如果事务已关闭，则返回errtxclosed
	Delete(key []byte) error
}

//blockRegion指定由
//指定哈希，给定偏移量和长度。
type BlockRegion struct {
	Hash   *chainhash.Hash
	Offset uint32
	Len    uint32
}

//Tx表示数据库事务。它可以是只读的，也可以是
//读写。事务提供了一个元数据存储桶，所有
//读写发生。
//
//与事务预期的一样，不会将任何更改保存到
//数据库，直到提交为止。交易将只提供
//创建数据库时的数据库视图。交易不应
//长时间运行的操作。
type Tx interface {
//元数据返回所有元数据存储的最高存储桶。
	Metadata() Bucket

//storeblock将提供的块存储到数据库中。没有
//检查以确保块连接到上一个块，包含
//双倍花费，或任何附加功能，如事务
//索引。它只是将块存储在数据库中。
//
//接口合同至少保证以下错误
//返回（可能存在其他特定于实现的错误）：
//-当块哈希已存在时，errblockexists
//-errtxnotwritable（如果尝试对只读事务执行此操作）
//-如果事务已关闭，则返回errtxclosed
//
//其他错误可能取决于实现。
	StoreBlock(block *btcutil.Block) error

//HasBlock返回具有给定哈希的块是否存在
//在数据库中。
//
//接口合同至少保证以下错误
//返回（可能存在其他特定于实现的错误）：
//-如果事务已关闭，则返回errtxclosed
//
//其他错误可能取决于实现。
	HasBlock(hash *chainhash.Hash) (bool, error)

//HasBlocks返回具有提供的哈希值的块
//存在于数据库中。
//
//接口合同至少保证以下错误
//返回（可能存在其他特定于实现的错误）：
//-如果事务已关闭，则返回errtxclosed
//
//其他错误可能取决于实现。
	HasBlocks(hashes []chainhash.Hash) ([]bool, error)

//FetchBlockHeader返回块的原始序列化字节
//header identified by the given hash.  The raw bytes are in the format
//通过Wire.BlockHeader上的序列化返回。
//
//强烈建议使用此函数（或fetchblockheaders）
//在fetchblockregion函数上获取块头，因为
//它为后端驱动程序提供了执行非常具体的
//当
//使用标题。
//
//接口合同至少保证以下错误
//返回（可能存在其他特定于实现的错误）：
//-errblocknotfound如果请求的块哈希不存在
//-如果事务已关闭，则返回errtxclosed
//-errCorrupt如果数据库已损坏
//
//注意：此函数返回的数据仅在
//数据库事务。试图在事务后访问它
//已结束将导致未定义的行为。此约束可防止
//其他数据副本，并允许支持内存映射数据库
//实施。
	FetchBlockHeader(hash *chainhash.Hash) ([]byte, error)

//FetchBlockHeaders返回块的原始序列化字节
//由给定哈希标识的头。原始字节位于
//Wire.BlockHeader上的序列化返回的格式。
//
//强烈建议使用此函数（或fetchblockheader）
//在fetchblockregion函数上获取块头，因为
//它为后端驱动程序提供了执行非常具体的
//当
//使用标题。
//
//此外，根据具体的实现，此函数
//对于批量加载多个块头，效率可能比
//使用fetchblockheader逐个加载它们。
//
//接口合同至少保证以下错误
//返回（可能存在其他特定于实现的错误）：
//-errblocknotfound如果任何请求块哈希不存在
//-如果事务已关闭，则返回errtxclosed
//-errCorrupt如果数据库已损坏
//
//注意：此函数返回的数据仅在
//数据库事务。试图在事务后访问它
//已结束将导致未定义的行为。此约束可防止
//其他数据副本，并允许支持内存映射数据库
//实施。
	FetchBlockHeaders(hashes []chainhash.Hash) ([][]byte, error)

//fetchblock返回所标识块的原始序列化字节
//按给定的哈希。原始字节的格式为
//在Wire.msgBlock上序列化。
//
//接口合同至少保证以下错误
//返回（可能存在其他特定于实现的错误）：
//-errblocknotfound如果请求的块哈希不存在
//-如果事务已关闭，则返回errtxclosed
//-errCorrupt如果数据库已损坏
//
//注意：此函数返回的数据仅在
//数据库事务。试图在事务后访问它
//已结束将导致未定义的行为。此约束可防止
//其他数据副本，并允许支持内存映射数据库
//实施。
	FetchBlock(hash *chainhash.Hash) ([]byte, error)

//FetchBlocks返回块的原始序列化字节
//由给定哈希标识。原始字节的格式为
//在WiR.MSGBULL中序列化返回。
//
//接口合同至少保证以下错误
//返回（可能存在其他特定于实现的错误）：
//-errblocknotfound如果任何请求的块散列没有找到
//存在
//-如果事务已关闭，则返回errtxclosed
//-errCorrupt如果数据库已损坏
//
//注意：此函数返回的数据仅在
//数据库事务。试图在事务后访问它
//已结束将导致未定义的行为。此约束可防止
//其他数据副本，并允许支持内存映射数据库
//实施。
	FetchBlocks(hashes []chainhash.Hash) ([][]byte, error)

//FetchBlockRegion返回给定的原始序列化字节
//块区。
//
//例如，可以直接提取比特币交易
//和/或具有此函数的块中的脚本。取决于
//后端实现，这可以通过
//避免加载整个块。
//
//原始字节的格式是在
//wire.MsgBlock and the Offset field in the provided BlockRegion is
//从零开始，相对于块的开头（字节0）。
//
//接口合同至少保证以下错误
//返回（可能存在其他特定于实现的错误）：
//-errblocknotfound如果请求的块哈希不存在
//-errblockregioninvalid如果区域超过
//关联块
//-如果事务已关闭，则返回errtxclosed
//-errCorrupt如果数据库已损坏
//
//注意：此函数返回的数据仅在
//数据库事务。试图在事务后访问它
//已结束将导致未定义的行为。此约束可防止
//其他数据副本，并允许支持内存映射数据库
//实施。
	FetchBlockRegion(region *BlockRegion) ([]byte, error)

//FetchBlockRegions返回给定的原始序列化字节
//块区域。
//
//例如，可以直接提取比特币交易
//和/或来自具有此功能的各个块的脚本。取决于
//后端实现，这可以大大节省
//避免加载整个块。
//
//原始字节的格式是在
//Wire.msgBlock和所提供块区域中的偏移字段是
//从零开始，相对于块的开头（字节0）。
//
//接口合同至少保证以下错误
//返回（可能存在其他特定于实现的错误）：
//-errblocknotfound如果请求的块散列中没有
//存在
//- ErrBlockRegionInvalid if one or more region exceed the bounds of
//关联的块
//-如果事务已关闭，则返回errtxclosed
//-errCorrupt如果数据库已损坏
//
//注意：此函数返回的数据仅在
//数据库事务。试图在事务后访问它
//已结束将导致未定义的行为。此约束可防止
//其他数据副本，并允许支持内存映射数据库
//实施。
	FetchBlockRegions(regions []BlockRegion) ([][]byte, error)

//********************************************************
//与原子元数据存储和块存储相关的方法。
//********************************************************

//提交提交对元数据的所有更改或
//块存储。根据后端实现的不同，这可能是
//到定期同步到持久存储的缓存，或
//直接存储到持久存储。在任何情况下，所有交易
//在提交完成后启动将包括所做的所有更改
//通过这个交易。在托管事务上调用此函数
//会导致恐慌。
	Commit() error

//回滚撤消对元数据所做的所有更改，或
//块存储。Calling this function on a managed transaction will
//导致恐慌。
	Rollback() error
}

//DB提供了一个通用接口，用于存储比特币块和
//相关元数据。此接口旨在与实际
//用于后端数据存储的机制。RegisterDriver功能可以是
//用于添加新的后端数据存储方法。
//
//这个接口分为两类不同的功能。
//
//第一类是支持bucket的原子元数据存储。这是
//通过使用数据库事务来完成。
//
//第二类是通用块存储。此功能是
//intentionally separate because the mechanism used for block storage may or
//可能与用于元数据存储的机制不同。例如，它是
//通常更高效地将块数据存储为平面文件，而元数据
//保存在数据库中。然而，这个接口的目标是足够通用
//如果特定后端需要，也支持数据库中的块。
type DB interface {
//类型返回数据库驱动程序类型当前数据库实例
//是用创建的。
	Type() string

//BEGIN启动只读或读写的事务
//取决于指定的标志。多个只读事务
//只能同时启动一个读写
//事务可以一次启动。呼叫将在何时阻塞
//当读写事务已打开时启动。
//
//注意：事务必须通过调用回滚或提交来关闭。
//当它不再需要的时候。不这样做会导致
//无人认领的内存和/或由于锁定而无法关闭数据库
//取决于具体的数据库实现。
	Begin(writable bool) (Tx, error)

//视图在托管的上下文中调用传递的函数
//只读事务。从提供的用户返回的任何错误
//函数从这个函数返回。
//
//对传递给的事务调用rollback或commit
//用户提供的函数将导致死机。
	View(fn func(tx Tx) error) error

//Update invokes the passed function in the context of a managed
//读写事务。从提供的用户返回的任何错误
//函数将导致事务回滚，并且
//从此函数返回。否则，事务被提交
//当用户提供的函数返回nil错误时。
//
//对传递给的事务调用rollback或commit
//用户提供的函数将导致死机。
	Update(fn func(tx Tx) error) error

//CLOSE干净地关闭数据库并同步所有数据。它将
//阻止，直到完成所有数据库事务（滚动
//支持或承诺）。
	Close() error
}
