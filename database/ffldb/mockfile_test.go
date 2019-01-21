
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

//此文件是FFLDB包的一部分，而不是作为
//它是白盒测试的一部分。

package ffldb

import (
	"errors"
	"io"
	"sync"
)

//用于模拟文件的错误。
var (
//errmockfileclosed用于指示模拟文件已关闭。
	errMockFileClosed = errors.New("file closed")

//errInvalidOffset用于指示超出范围的偏移量
//为文件提供了。
	errInvalidOffset = errors.New("invalid offset")

//errsyncfail用于指示模拟的同步失败。
	errSyncFail = errors.New("simulated sync failure")
)

//mockfile实现文件管理器接口，用于强制失败
//与从平面块文件读取和写入相关的数据库代码。
//最大大小为-1是不受限制的。
type mockFile struct {
	sync.RWMutex
	maxSize      int64
	data         []byte
	forceSyncErr bool
	closed       bool
}

//关闭关闭模拟文件，而不释放与之关联的任何数据。
//这样可以在不丢失数据的情况下“重新打开”。
//
//这是文件管理器实现的一部分。
func (f *mockFile) Close() error {
	f.Lock()
	defer f.Unlock()

	if f.closed {
		return errMockFileClosed
	}
	f.closed = true
	return nil
}

//ReadAt reads len(b) bytes from the mock file starting at byte offset off. 它
//返回读取的字节数和错误（如果有）。总是读取
//当n<len（b）时返回非零错误。在文件末尾，该错误是
//IOF。
//
//这是文件管理器实现的一部分。
func (f *mockFile) ReadAt(b []byte, off int64) (int, error) {
	f.RLock()
	defer f.RUnlock()

	if f.closed {
		return 0, errMockFileClosed
	}
	maxSize := int64(len(f.data))
	if f.maxSize > -1 && maxSize > f.maxSize {
		maxSize = f.maxSize
	}
	if off < 0 || off > maxSize {
		return 0, errInvalidOffset
	}

//限制为“最大大小”字段（如果设置）。
	numToRead := int64(len(b))
	endOffset := off + numToRead
	if endOffset > maxSize {
		numToRead = maxSize - off
	}

	copy(b, f.data[off:off+numToRead])
	if numToRead < int64(len(b)) {
		return int(numToRead), io.EOF
	}
	return int(numToRead), nil
}

//truncate更改模拟文件的大小。
//
//这是文件管理器实现的一部分。
func (f *mockFile) Truncate(size int64) error {
	f.Lock()
	defer f.Unlock()

	if f.closed {
		return errMockFileClosed
	}
	maxSize := int64(len(f.data))
	if f.maxSize > -1 && maxSize > f.maxSize {
		maxSize = f.maxSize
	}
	if size > maxSize {
		return errInvalidOffset
	}

	f.data = f.data[:size]
	return nil
}

//写入将len（b）字节写入模拟文件。它返回字节数
//写的和一个错误，如果有的话。WRITE随时返回非零错误
//n！= LeN（b）。
//
//这是文件管理器实现的一部分。
func (f *mockFile) WriteAt(b []byte, off int64) (int, error) {
	f.Lock()
	defer f.Unlock()

	if f.closed {
		return 0, errMockFileClosed
	}
	maxSize := f.maxSize
	if maxSize < 0 {
maxSize = 100 * 1024 //100KiB
	}
	if off < 0 || off > maxSize {
		return 0, errInvalidOffset
	}

//限制到最大大小字段（如果设置），并根据需要增大切片。
	numToWrite := int64(len(b))
	if off+numToWrite > maxSize {
		numToWrite = maxSize - off
	}
	if off+numToWrite > int64(len(f.data)) {
		newData := make([]byte, off+numToWrite)
		copy(newData, f.data)
		f.data = newData
	}

	copy(f.data[off:], b[:numToWrite])
	if numToWrite < int64(len(b)) {
		return int(numToWrite), io.EOF
	}
	return int(numToWrite), nil
}

//同步对模拟文件没有任何作用。但是，如果
//模拟文件的forcesyncerr标志已设置。
//
//这是文件管理器实现的一部分。
func (f *mockFile) Sync() error {
	if f.forceSyncErr {
		return errSyncFail
	}

	return nil
}

//确保mockfile类型实现文件管理器接口。
var _ filer = (*mockFile)(nil)
