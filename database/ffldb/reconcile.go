
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

package ffldb

import (
	"fmt"
	"hash/crc32"

	"github.com/btcsuite/btcd/database"
)

//序列化写入光标位置格式为：
//
//[0:4]块文件（4字节）
//[4:8]文件偏移量（4字节）
//[8:12]Castagnoli CRC-32校验和（4字节）

//serializewriterow序列化当前块文件并偏移新的
//将写入适合存储到元数据中的格式。
func serializeWriteRow(curBlockFileNum, curFileOffset uint32) []byte {
	var serializedRow [12]byte
	byteOrder.PutUint32(serializedRow[0:4], curBlockFileNum)
	byteOrder.PutUint32(serializedRow[4:8], curFileOffset)
	checksum := crc32.Checksum(serializedRow[:8], castagnoli)
	byteOrder.PutUint32(serializedRow[8:12], checksum)
	return serializedRow[:]
}

//反序列化WriteRow反序列化存储在
//元数据。如果项的校验和不匹配，则返回errCorruption。
func deserializeWriteRow(writeRow []byte) (uint32, uint32, error) {
//确保校验和匹配。校验和在末尾。
	gotChecksum := crc32.Checksum(writeRow[:8], castagnoli)
	wantChecksumBytes := writeRow[8:12]
	wantChecksum := byteOrder.Uint32(wantChecksumBytes)
	if gotChecksum != wantChecksum {
		str := fmt.Sprintf("metadata for write cursor does not match "+
			"the expected checksum - got %d, want %d", gotChecksum,
			wantChecksum)
		return 0, 0, makeDbErr(database.ErrCorruption, str, nil)
	}

	fileNum := byteOrder.Uint32(writeRow[0:4])
	fileOffset := byteOrder.Uint32(writeRow[4:8])
	return fileNum, fileOffset, nil
}

//协调数据库将元数据与磁盘上的平面块文件进行协调。它
//如果设置了创建标志，还将初始化基础数据库。
func reconcileDB(pdb *db, create bool) (database.DB, error) {
//在数据库期间执行初始内部存储桶和值创建
//创造。
	if create {
		if err := initDB(pdb.cache.ldb); err != nil {
			return nil, err
		}
	}

//从元数据加载当前写入光标位置。
	var curFileNum, curOffset uint32
	err := pdb.View(func(tx database.Tx) error {
		writeRow := tx.Metadata().Get(writeLocKeyName)
		if writeRow == nil {
			str := "write cursor does not exist"
			return makeDbErr(database.ErrCorruption, str, nil)
		}

		var err error
		curFileNum, curOffset, err = deserializeWriteRow(writeRow)
		return err
	})
	if err != nil {
		return nil, err
	}

//当通过扫描块文件找到写入光标位置时
//磁盘位于元数据认为为真的位置之后，截断
//磁盘上与元数据匹配的文件。这很常见
//当块文件位于不干净的关闭方案中时发生
//写作的中间部分。因为元数据在
//在写入块数据之后，这实际上只是一个回滚
//在不清洁停机前达到已知的良好状态。
	wc := pdb.store.writeCursor
	if wc.curFileNum > curFileNum || (wc.curFileNum == curFileNum &&
		wc.curOffset > curOffset) {

		log.Info("Detected unclean shutdown - Repairing...")
		log.Debugf("Metadata claims file %d, offset %d. Block data is "+
			"at file %d, offset %d", curFileNum, curOffset,
			wc.curFileNum, wc.curOffset)
		pdb.store.handleRollback(curFileNum, curOffset)
		log.Infof("Database sync complete")
	}

//当通过扫描块文件找到写入光标位置时
//磁盘在元数据认为为真的位置之前，返回
//损坏错误。因为同步是在每个块写入之后调用的
//在更新元数据之前，只应在
//丢失、删除或截断块文件的情况，通常
//不是一个容易恢复的方案。在未来，可能是
//可以从块文件重新扫描和重建元数据，
//然而，这需要更高层次的协调
//层，因为它可以使其他元数据无效。
	if wc.curFileNum < curFileNum || (wc.curFileNum == curFileNum &&
		wc.curOffset < curOffset) {

		str := fmt.Sprintf("metadata claims file %d, offset %d, but "+
			"block data is at file %d, offset %d", curFileNum,
			curOffset, wc.curFileNum, wc.curOffset)
		log.Warnf("***Database corruption detected***: %v", str)
		return nil, makeDbErr(database.ErrCorruption, str, nil)
	}

	return pdb, nil
}
