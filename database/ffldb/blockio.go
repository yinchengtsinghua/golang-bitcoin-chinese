
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

//此文件包含用于读取、写入和
//否则，使用包含实际块的平面文件。

package ffldb

import (
	"container/list"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcd/wire"
)

const (
//比特币协议将块高度编码为Int32，因此
//块数为2^31。每个协议的最大块大小为每个块32mib。
//所以写这篇评论的时候理论上最大值是64pib
//（二进制字节）对于每个文件@512mib，这将需要一个最大值
//共134217728个文件。因此，选择9位精度
//文件名。另一个好处是9位数提供10^9个文件@
//总共约476.84pib（约为电流的7.4倍），每个512mib
//理论上最大），因此最大块大小有空间增长
//未来。
	blockFilenameTemplate = "%09d.fdb"

//max open files是要在
//打开块缓存。注意，这不包括当前
//写入文件，因此通常会有一个以上的值打开。
	maxOpenFiles = 25

//maxBlockFileSize is the maximum size for each file used to store
//阻碍。
//
//注意：当前代码对所有偏移使用uint32，因此该值
//必须小于2^32（4 Gib）。这也是它被打印出来的原因
//常数。
maxBlockFileSize uint32 = 512 * 1024 * 1024 //512 MIB

//BlockLocSize是序列化块位置的字节数。
//存储在块索引中的数据。
//
//序列化块位置格式为：
//
//[0:4]块文件（4字节）
//[4:8]文件偏移量（4字节）
//[8:12]块长度（4字节）
	blockLocSize = 12
)

var (
//Castagnoli包含用于CRC-32校验和的Catagnoli多项式。
	castagnoli = crc32.MakeTable(crc32.Castagnoli)
)

//文件管理器是一个与*os.file非常相似的接口，通常
//由IT实施。它的存在使得测试代码可以为
//正确测试损坏和文件系统问题。
type filer interface {
	io.Closer
	io.WriterAt
	io.ReaderAt
	Truncate(size int64) error
	Sync() error
}

//LockableFile表示磁盘上已为
//读或读/写访问。它还包含一个读写互斥体来支持
//多个并发读卡器。
type lockableFile struct {
	sync.RWMutex
	file filer
}

//WriteCursor表示磁盘上块文件的当前文件和偏移量。
//用于执行所有写入操作。它还包含一个读写互斥体来支持
//可以重用文件句柄的多个并发读卡器。
type writeCursor struct {
	sync.RWMutex

//curfile是当前块文件，当
//正在写入新块。
	curFile *lockableFile

//curfilenum是当前块文件号，用于
//读卡器使用相同的打开文件句柄。
	curFileNum uint32

//curoffset是当前写块文件中的偏移量，其中
//将写入下一个新块。
	curOffset uint32
}

//Blockstore存储用于处理读写块的信息（以及
//part of blocks) into flat files with support for multiple concurrent readers.
type blockStore struct {
//网络是每个平面文件中使用的特定网络
//块。
	network wire.BitcoinNet

//base path是用于平面块文件和元数据的基本路径。
	basePath string

//MaxBlockFileSize是用于存储的每个文件的最大大小
//阻碍。它是在存储中定义的，因此白盒测试可以
//覆盖该值。
	maxBlockFileSize uint32

//以下字段与保存
//实际块。打开的文件数受maxopenfiles的限制。
//
//obfmutex保护对openblockfiles映射的并发访问。它是
//一个rwmutex，使多个读卡器可以同时访问打开的文件。
//
//OpenBlockFiles包含现有块文件的打开文件句柄
//它与一个单独的rwmutex一起以只读方式打开。
//此方案允许多个并发读卡器访问同一文件，同时
//防止文件被关闭。
//
//lrumutex保护对最近使用最少的列表的并发访问
//查找地图。
//
//openblockslru通过按
//最新使用的文件在列表的前面，因此
//列表末尾最近使用最少的文件。当文件需要时
//因超过允许打开的最大数量而关闭
//文件，列表末尾的文件关闭。
//
//filenumtolruelem是特定块文件号之间的映射
//以及最近使用最少的列表上的关联列表元素。
//
//因此，通过这些字段的组合，数据库支持
//跨多个和单个文件的并发非阻塞读取
//同时智能地限制打开文件句柄的数量
//根据需要关闭最近使用最少的文件。
//
//注意：整个过程中使用的锁定顺序定义明确，必须
//跟着。不这样做可能导致死锁。特别地，
//锁定顺序如下：
//1）ObfutMutX
//2）LRUMUTEX
//3）writecursor互斥
//4）特定文件互斥
//
//无需同时锁定任何互斥体，并且
//通常不会。但是，如果要同时锁定它们，它们
//必须按先前指定的顺序锁定。
//
//由于高性能和多读并发性要求，
//写锁只应保持所需的最短时间。
	obfMutex         sync.RWMutex
	lruMutex         sync.Mutex
openBlocksLRU    *list.List //包含uint32块文件编号。
	fileNumToLRUElem map[uint32]*list.Element
	openBlockFiles   map[uint32]*lockableFile

//WriteCursor存储当前文件的状态和
//新块将写入。
	writeCursor *writeCursor

//这些函数设置为openfile、openwritefile和deletefile
//默认，但在此公开以允许白盒测试替换
//在处理模拟文件时。
	openFileFunc      func(fileNum uint32) (*lockableFile, error)
	openWriteFileFunc func(fileNum uint32) (filer, error)
	deleteFileFunc    func(fileNum uint32) error
}

//block location标识特定的块文件和位置。
type blockLocation struct {
	blockFileNum uint32
	fileOffset   uint32
	blockLen     uint32
}

//反序列化blockloc反序列化传递的序列化块位置
//信息。这是存储在每个块索引元数据中的数据
//块。传递到此函数的序列化数据必须至少为
//blocklocSize字节，否则会死机。这里避免了错误检查，因为
//此信息始终来自块索引，其中包括
//检查和以检测损坏。因此在这里不加检查是安全的。
func deserializeBlockLoc(serializedLoc []byte) blockLocation {
//序列化块位置格式为：
//
//[0:4]块文件（4字节）
//[4:8]文件偏移量（4字节）
//[8:12]块长度（4字节）
	return blockLocation{
		blockFileNum: byteOrder.Uint32(serializedLoc[0:4]),
		fileOffset:   byteOrder.Uint32(serializedLoc[4:8]),
		blockLen:     byteOrder.Uint32(serializedLoc[8:12]),
	}
}

//SerializeBlockLoc返回传递的块位置的序列化。
//这是要存储到每个块的块索引元数据中的数据。
func serializeBlockLoc(loc blockLocation) []byte {
//序列化块位置格式为：
//
//[0:4]块文件（4字节）
//[4:8]文件偏移量（4字节）
//[8:12]块长度（4字节）
	var serializedData [12]byte
	byteOrder.PutUint32(serializedData[0:4], loc.blockFileNum)
	byteOrder.PutUint32(serializedData[4:8], loc.fileOffset)
	byteOrder.PutUint32(serializedData[8:12], loc.blockLen)
	return serializedData[:]
}

//block file path返回提供的块文件号的文件路径。
func blockFilePath(dbPath string, fileNum uint32) string {
	fileName := fmt.Sprintf(blockFilenameTemplate, fileNum)
	return filepath.Join(dbPath, fileName)
}

//openwritefile返回传入的平面文件号的文件句柄
//读/写模式。如果需要，将创建该文件。通常使用
//对于将附加所有新数据的当前文件。与OpenFile不同，
//此函数不跟踪打开的文件，它不受
//maxopenfiles限制。
func (s *blockStore) openWriteFile(fileNum uint32) (filer, error) {
//当前块文件需要读写，因此可以
//追加它。而且，它不应该是最近使用最少的
//文件。
	filePath := blockFilePath(s.basePath, fileNum)
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		str := fmt.Sprintf("failed to open file %q: %v", filePath, err)
		return nil, makeDbErr(database.ErrDriverSpecific, str, err)
	}

	return file, nil
}

//openfile返回传递的平面文件号的只读文件句柄。
//该函数还跟踪打开的文件，执行时间最短
//使用跟踪，并通过关闭将打开的文件数限制为maxopenfiles
//最新使用的文件。
//
//调用此函数时必须锁定整个文件mutex（s.obfmutex）
//用于书写。
func (s *blockStore) openFile(fileNum uint32) (*lockableFile, error) {
//以只读方式打开相应的文件。
	filePath := blockFilePath(s.basePath, fileNum)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, makeDbErr(database.ErrDriverSpecific, err.Error(),
			err)
	}
	blockFile := &lockableFile{file: file}

//如果文件超过最大值，则关闭最近使用的文件
//允许打开文件。直到文件在
//如果文件无法打开，则无需关闭任何文件。
//
//这里的lru列表需要写锁来防止
//当已经打开的文件被读取和
//无序移动到列表的前面。
//
//另外，将刚刚打开的文件添加到最前面
//最近使用的列表，用于指示它是最近使用的文件，以及
//所以应该最后关闭。
	s.lruMutex.Lock()
	lruList := s.openBlocksLRU
	if lruList.Len() >= maxOpenFiles {
		lruFileNum := lruList.Remove(lruList.Back()).(uint32)
		oldBlockFile := s.openBlockFiles[lruFileNum]

//关闭文件写锁下的旧文件，以防
//任何读者当前都在阅读它，所以它不会关闭
//从他们下面出来。
		oldBlockFile.Lock()
		_ = oldBlockFile.file.Close()
		oldBlockFile.Unlock()

		delete(s.openBlockFiles, lruFileNum)
		delete(s.fileNumToLRUElem, lruFileNum)
	}
	s.fileNumToLRUElem[fileNum] = lruList.PushFront(fileNum)
	s.lruMutex.Unlock()

//在打开的块文件映射中存储对它的引用。
	s.openBlockFiles[fileNum] = blockFile

	return blockFile, nil
}

//删除文件删除传递的平面文件号的块文件。文件
//必须已关闭，呼叫方有责任执行
//其他必要的状态清理。
func (s *blockStore) deleteFile(fileNum uint32) error {
	filePath := blockFilePath(s.basePath, fileNum)
	if err := os.Remove(filePath); err != nil {
		return makeDbErr(database.ErrDriverSpecific, err.Error(), err)
	}

	return nil
}

//blockfile尝试返回传递的平面文件的现有文件句柄
//如果它已经打开，则编号，并将其标记为最近使用的编号。它
//还将在文件尚未按规则打开时打开该文件
//在OpenFile中描述。
//
//注意：返回的块文件将已经获得读取锁，并且
//调用方必须调用.runlock（）在完成所有读取后释放它。
//操作。这是必要的，否则
//分开goroutine以在文件从这里返回后关闭该文件，但是
//在调用方获得读取锁之前。
func (s *blockStore) blockFile(fileNum uint32) (*lockableFile, error) {
//当请求的块文件打开进行写入时，返回它。
	wc := s.writeCursor
	wc.RLock()
	if fileNum == wc.curFileNum && wc.curFile.file != nil {
		obf := wc.curFile
		obf.RLock()
		wc.RUnlock()
		return obf, nil
	}
	wc.RUnlock()

//尝试在“总体文件读取”锁定下返回打开的文件。
	s.obfMutex.RLock()
	if obf, ok := s.openBlockFiles[fileNum]; ok {
		s.lruMutex.Lock()
		s.openBlocksLRU.MoveToFront(s.fileNumToLRUElem[fileNum])
		s.lruMutex.Unlock()

		obf.RLock()
		s.obfMutex.RUnlock()
		return obf, nil
	}
	s.obfMutex.RUnlock()

//由于文件尚未打开，需要检查打开的块文件
//在写锁下再次映射，以防多个读卡器到达此处
//单独的一个已经在打开文件。
	s.obfMutex.Lock()
	if obf, ok := s.openBlockFiles[fileNum]; ok {
		obf.RLock()
		s.obfMutex.Unlock()
		return obf, nil
	}

//该文件未打开，因此在可能关闭最少的文件时打开它
//最近根据需要使用过。
	obf, err := s.openFileFunc(fileNum)
	if err != nil {
		s.obfMutex.Unlock()
		return nil, err
	}
	obf.RLock()
	s.obfMutex.Unlock()
	return obf, nil
}

//WriteData是写入所提供数据的WriteBlock的帮助函数。
//在当前写入偏移量，并相应地更新写入光标。这个
//字段名参数仅在出现错误以提供更好的
//错误信息。
//
//写入光标将前进实际写入的字节数
//故障事件。
//
//注意：必须使用写入光标当前文件锁调用此函数
//只有在写事务期间才能调用，因此它是有效的
//为写入而锁定。此外，写入光标当前文件不能为零。
func (s *blockStore) writeData(data []byte, fieldName string) error {
	wc := s.writeCursor
	n, err := wc.curFile.file.WriteAt(data, int64(wc.curOffset))
	wc.curOffset += uint32(n)
	if err != nil {
		str := fmt.Sprintf("failed to write %s to file %d at "+
			"offset %d: %v", fieldName, wc.curFileNum,
			wc.curOffset-uint32(n), err)
		return makeDbErr(database.ErrDriverSpecific, str, err)
	}

	return nil
}

//WriteBlock将指定的原始块字节追加到存储的写入光标上
//定位并相应增加。当块超过最大值时
//当前平面文件的文件大小，此函数将关闭当前
//文件，创建下一个文件，更新写入光标，并将块写入
//新文件。
//
//写入光标也将被推进实际写入的字节数。
//如果发生故障。
//
//格式：<network><block length><serialized block><checksum>
func (s *blockStore) writeBlock(rawBlock []byte) (blockLocation, error) {
//计算将写入的字节数。
//块网络4个字节，块长度+4个字节
//原始块的长度+校验和为4字节。
	blockLen := uint32(len(rawBlock))
	fullLen := blockLen + 12

//如果添加新块将超过
//当前块文件允许的最大大小。同时检测溢出
//偏执，即使目前不可能，数字
//将来可能会改变，使之成为可能。
//
//注意：writeCursor.offset字段不受互斥体保护
//因为它只在这个函数期间被读取/更改
//在写入事务期间调用，其中只能有一个
//一段时间。
	wc := s.writeCursor
	finalOffset := wc.curOffset + fullLen
	if finalOffset < wc.curOffset || finalOffset > s.maxBlockFileSize {
//这是在写光标锁定下完成的，因为curfilenum
//读卡器可以在其他地方访问字段。
//
//关闭当前写入文件以强制以只读方式重新打开
//带LRU跟踪。关闭是在写锁下完成的。
//为了防止文件被关闭
//当前正在阅读的任何读者。
		wc.Lock()
		wc.curFile.Lock()
		if wc.curFile.file != nil {
			_ = wc.curFile.file.Close()
			wc.curFile.file = nil
		}
		wc.curFile.Unlock()

//开始写入下一个文件。
		wc.curFileNum++
		wc.curOffset = 0
		wc.Unlock()
	}

//所有写操作都在文件的写锁下完成，以确保
//读卡器先完成并阻塞。
	wc.curFile.Lock()
	defer wc.curFile.Unlock()

//如果需要，打开当前文件。这通常只是
//移动到下一个要写入到初始数据库的文件时的大小写
//负载。但是，如果在
//文件写入在事务提交期间启动。
	if wc.curFile.file == nil {
		file, err := s.openWriteFileFunc(wc.curFileNum)
		if err != nil {
			return blockLocation{}, err
		}
		wc.curFile.file = file
	}

//比特币网络。
	origOffset := wc.curOffset
	hasher := crc32.New(castagnoli)
	var scratch [4]byte
	byteOrder.PutUint32(scratch[:], uint32(s.network))
	if err := s.writeData(scratch[:], "network"); err != nil {
		return blockLocation{}, err
	}
	_, _ = hasher.Write(scratch[:])

//块长度。
	byteOrder.PutUint32(scratch[:], blockLen)
	if err := s.writeData(scratch[:], "block length"); err != nil {
		return blockLocation{}, err
	}
	_, _ = hasher.Write(scratch[:])

//序列化块。
	if err := s.writeData(rawBlock[:], "block"); err != nil {
		return blockLocation{}, err
	}
	_, _ = hasher.Write(rawBlock)

//Castagnoli CRC-32作为所有之前的校验和。
	if err := s.writeData(hasher.Sum(nil), "checksum"); err != nil {
		return blockLocation{}, err
	}

	loc := blockLocation{
		blockFileNum: wc.curFileNum,
		fileOffset:   origOffset,
		blockLen:     fullLen,
	}
	return loc, nil
}

//readblock读取指定的块记录并返回序列化的块。
//它通过检查
//网络匹配与块存储关联的当前网络，并且
//将计算的校验和与平面文件中存储的校验和进行比较。
//此功能还自动处理所有文件管理，如打开
//并根据需要关闭文件以保持在允许的最大打开文件范围内
//极限。
//
//如果由于任何原因无法读取数据，则返回errDriverSpecific，并且
//如果读取数据的校验和与校验和不匹配，则会发生错误。
//从文件中读取。
//
//格式：<network><block length><serialized block><checksum>
func (s *blockStore) readBlock(hash *chainhash.Hash, loc blockLocation) ([]byte, error) {
//获取引用的块文件句柄，根据需要打开该文件。这个
//函数还根据需要处理关闭文件，以避免
//允许的最大打开文件数。
	blockFile, err := s.blockFile(loc.blockFileNum)
	if err != nil {
		return nil, err
	}

	serializedData := make([]byte, loc.blockLen)
	n, err := blockFile.file.ReadAt(serializedData, int64(loc.fileOffset))
	blockFile.RUnlock()
	if err != nil {
		str := fmt.Sprintf("failed to read block %s from file %d, "+
			"offset %d: %v", hash, loc.blockFileNum, loc.fileOffset,
			err)
		return nil, makeDbErr(database.ErrDriverSpecific, str, err)
	}

//计算读取数据的校验和并确保其与
//序列化校验和。这将检测
//无需执行更昂贵的merkle根的平面文件
//加载块的计算。
	serializedChecksum := binary.BigEndian.Uint32(serializedData[n-4:])
	calculatedChecksum := crc32.Checksum(serializedData[:n-4], castagnoli)
	if serializedChecksum != calculatedChecksum {
		str := fmt.Sprintf("block data for block %s checksum "+
			"does not match - got %x, want %x", hash,
			calculatedChecksum, serializedChecksum)
		return nil, makeDbErr(database.ErrCorruption, str, nil)
	}

//与块关联的网络必须与当前活动的网络匹配
//网络，否则可能有人将块文件
//目录中的网络错误。
	serializedNet := byteOrder.Uint32(serializedData[:4])
	if serializedNet != uint32(s.network) {
		str := fmt.Sprintf("block data for block %s is for the "+
			"wrong network - got %d, want %d", hash, serializedNet,
			uint32(s.network))
		return nil, makeDbErr(database.ErrDriverSpecific, str, nil)
	}

//原始块不包括网络、块的长度和
//校验和。
	return serializedData[8 : n-4], nil
}

//readBlockRegion以提供的偏移量读取指定数量的数据
//给定的块位置。偏移相对于
//序列化块（而不是块记录的开头）。这个
//函数自动处理所有文件管理，如打开和
//根据需要关闭文件以保持在允许的最大打开文件范围内
//极限。
//
//如果由于任何原因无法读取数据，则返回errDriverSpecific。
func (s *blockStore) readBlockRegion(loc blockLocation, offset, numBytes uint32) ([]byte, error) {
//获取引用的块文件句柄，根据需要打开该文件。这个
//函数还根据需要处理关闭文件，以避免
//允许的最大打开文件数。
	blockFile, err := s.blockFile(loc.blockFileNum)
	if err != nil {
		return nil, err
	}

//区域是实际块的偏移量，但是
//块的数据包括网络的初始4字节+4字节
//对于块长度。因此，添加8个字节进行调整。
	readOffset := loc.fileOffset + 8 + offset
	serializedData := make([]byte, numBytes)
	_, err = blockFile.file.ReadAt(serializedData, int64(readOffset))
	blockFile.RUnlock()
	if err != nil {
		str := fmt.Sprintf("failed to read region from block file %d, "+
			"offset %d, len %d: %v", loc.blockFileNum, readOffset,
			numBytes, err)
		return nil, makeDbErr(database.ErrDriverSpecific, str, err)
	}

	return serializedData, nil
}

//同步块对与
//存储的当前写入光标。即使没有
//当前写入文件，在这种情况下，它将不起作用。
//
//这在将缓存元数据更新刷新到磁盘时使用，以确保
//块数据在更新元数据之前已完全写入。这确保了
//元数据和块数据可以在故障场景中正确地协调。
func (s *blockStore) syncBlocks() error {
	wc := s.writeCursor
	wc.RLock()
	defer wc.RUnlock()

//如果没有与写入相关联的当前文件，则不执行任何操作。
//光标。
	wc.curFile.RLock()
	defer wc.curFile.RUnlock()
	if wc.curFile.file == nil {
		return nil
	}

//将文件同步到磁盘。
	if err := wc.curFile.file.Sync(); err != nil {
		str := fmt.Sprintf("failed to sync file %d: %v", wc.curFileNum,
			err)
		return makeDbErr(database.ErrDriverSpecific, str, err)
	}

	return nil
}

//handlerollback将磁盘上的块文件回滚到提供的文件号
//和偏移量。这可能涉及删除和截断
//部分是书面的。
//
//这里有两种情况需要考虑：
//1）可从中恢复的瞬时写入失败
//2）更持久的故障，如硬盘死机和/或删除
//
//无论哪种情况，写入光标都将重新定位到旧的块文件中。
//无论尝试撤消时发生任何其他错误，都将进行偏移。
//写。
//
//对于第一个场景，这将导致无法撤消的任何数据
//被覆盖，从而在系统继续运行时按需运行。
//
//对于第二个场景，存储当前写入光标的元数据
//块文件中的位置尚未更新，因此如果
//系统最终会恢复（可能硬盘重新连接），它
//也会导致无法撤消的任何数据被覆盖和
//这样就可以随心所欲了。
//
//因此，任何错误都只是记录在警告级别，而不是
//返回，因为无论如何都无法做更多的事情。
func (s *blockStore) handleRollback(oldBlockFileNum, oldBlockOffset uint32) {
//抓取写光标mutex，因为它在整个过程中被修改
//功能。
	wc := s.writeCursor
	wc.Lock()
	defer wc.Unlock()

//如果回滚点与当前写入相同，则不执行任何操作。
//光标。
	if wc.curFileNum == oldBlockFileNum && wc.curOffset == oldBlockOffset {
		return
	}

//不管下面发生什么故障，重新定位写操作
//光标到旧的块文件和偏移。
	defer func() {
		wc.curFileNum = oldBlockFileNum
		wc.curOffset = oldBlockOffset
	}()

	log.Debugf("ROLLBACK: Rolling back to file %d, offset %d",
		oldBlockFileNum, oldBlockOffset)

//如果需要删除当前写入文件，请将其关闭。然后删除
//所有比提供的回滚文件更新的文件
//同时相应地向后移动写光标文件。
	if wc.curFileNum > oldBlockFileNum {
		wc.curFile.Lock()
		if wc.curFile.file != nil {
			_ = wc.curFile.file.Close()
			wc.curFile.file = nil
		}
		wc.curFile.Unlock()
	}
	for ; wc.curFileNum > oldBlockFileNum; wc.curFileNum-- {
		if err := s.deleteFileFunc(wc.curFileNum); err != nil {
			log.Warnf("ROLLBACK: Failed to delete block file "+
				"number %d: %v", wc.curFileNum, err)
			return
		}
	}

//如果需要，打开当前写入光标的文件。
	wc.curFile.Lock()
	if wc.curFile.file == nil {
		obf, err := s.openWriteFileFunc(wc.curFileNum)
		if err != nil {
			wc.curFile.Unlock()
			log.Warnf("ROLLBACK: %v", err)
			return
		}
		wc.curFile.file = obf
	}

//将截断为提供的回滚偏移量。
	if err := wc.curFile.file.Truncate(int64(oldBlockOffset)); err != nil {
		wc.curFile.Unlock()
		log.Warnf("ROLLBACK: Failed to truncate file %d: %v",
			wc.curFileNum, err)
		return
	}

//将文件同步到磁盘。
	err := wc.curFile.file.Sync()
	wc.curFile.Unlock()
	if err != nil {
		log.Warnf("ROLLBACK: Failed to sync file %d: %v",
			wc.curFileNum, err)
		return
	}
}

//scanblockfiles在数据库目录中搜索所有平面块文件
//查找最新文件的结尾。这个位置被认为是
//当前写入光标，也存储在元数据中。因此，它被使用
//在写入过程中检测意外关闭，以便阻止文件
//可以和解。
func scanBlockFiles(dbPath string) (int, uint32) {
	lastFile := -1
	fileLen := uint32(0)
	for i := 0; ; i++ {
		filePath := blockFilePath(dbPath, uint32(i))
		st, err := os.Stat(filePath)
		if err != nil {
			break
		}
		lastFile = i

		fileLen = uint32(st.Size())
	}

	log.Tracef("Scan found latest block file #%d with length %d", lastFile,
		fileLen)
	return lastFile, fileLen
}

//new block store返回具有当前块文件号的新块存储
//以及偏移集和所有字段初始化。
func newBlockStore(basePath string, network wire.BitcoinNet) *blockStore {
//查找要归档的最新块的结尾，以确定
//写入光标位置从块文件的视点
//磁盘。
	fileNum, fileOff := scanBlockFiles(basePath)
	if fileNum == -1 {
		fileNum = 0
		fileOff = 0
	}

	store := &blockStore{
		network:          network,
		basePath:         basePath,
		maxBlockFileSize: maxBlockFileSize,
		openBlockFiles:   make(map[uint32]*lockableFile),
		openBlocksLRU:    list.New(),
		fileNumToLRUElem: make(map[uint32]*list.Element),

		writeCursor: &writeCursor{
			curFile:    &lockableFile{},
			curFileNum: uint32(fileNum),
			curOffset:  fileOff,
		},
	}
	store.openFileFunc = store.openFile
	store.openWriteFileFunc = store.openWriteFile
	store.deleteFileFunc = store.deleteFile
	return store
}
