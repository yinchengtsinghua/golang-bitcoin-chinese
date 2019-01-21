
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

package indexers

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/btcsuite/btcd/wire"
)

//
//
type addrIndexBucket struct {
	levels map[[levelKeySize]byte][]byte
}

//
func (b *addrIndexBucket) Clone() *addrIndexBucket {
	levels := make(map[[levelKeySize]byte][]byte)
	for k, v := range b.levels {
		vCopy := make([]byte, len(v))
		copy(vCopy, v)
		levels[k] = vCopy
	}
	return &addrIndexBucket{levels: levels}
}

//
//桶。
//
//
func (b *addrIndexBucket) Get(key []byte) []byte {
	var levelKey [levelKeySize]byte
	copy(levelKey[:], key)
	return b.levels[levelKey]
}

//
//
//
func (b *addrIndexBucket) Put(key []byte, value []byte) error {
	var levelKey [levelKeySize]byte
	copy(levelKey[:], key)
	b.levels[levelKey] = value
	return nil
}

//删除从模拟地址索引桶中删除提供的密钥。
//
//
func (b *addrIndexBucket) Delete(key []byte) error {
	var levelKey [levelKeySize]byte
	copy(levelKey[:], key)
	delete(b.levels, levelKey)
	return nil
}

//
//考虑到每个级别的最大大小的地址键。它是有用的
//在创建和调试测试用例时。
func (b *addrIndexBucket) printLevels(addrKey [addrKeySize]byte) string {
	highestLevel := uint8(0)
	for k := range b.levels {
		if !bytes.Equal(k[:levelOffset], addrKey[:]) {
			continue
		}
		level := uint8(k[levelOffset])
		if level > highestLevel {
			highestLevel = level
		}
	}

	var levelBuf bytes.Buffer
	_, _ = levelBuf.WriteString("\n")
	maxEntries := level0MaxEntries
	for level := uint8(0); level <= highestLevel; level++ {
		data := b.levels[keyForLevel(addrKey, level)]
		numEntries := len(data) / txEntrySize
		for i := 0; i < numEntries; i++ {
			start := i * txEntrySize
			num := byteOrder.Uint32(data[start:])
			_, _ = levelBuf.WriteString(fmt.Sprintf("%02d ", num))
		}
		for i := numEntries; i < maxEntries; i++ {
			_, _ = levelBuf.WriteString("_  ")
		}
		_, _ = levelBuf.WriteString("\n")
		maxEntries *= 2
	}

	return levelBuf.String()
}

//
//
//文档。
func (b *addrIndexBucket) sanityCheck(addrKey [addrKeySize]byte, expectedTotal int) error {
//
	highestLevel := uint8(0)
	for k := range b.levels {
		if !bytes.Equal(k[:levelOffset], addrKey[:]) {
			continue
		}
		level := uint8(k[levelOffset])
		if level > highestLevel {
			highestLevel = level
		}
	}

//
//
//文档。
	var totalEntries int
	maxEntries := level0MaxEntries
	for level := uint8(0); level <= highestLevel; level++ {
//
//
//级别必须为半满或满。
		data := b.levels[keyForLevel(addrKey, level)]
		numEntries := len(data) / txEntrySize
		totalEntries += numEntries
		if level == 0 {
			if (highestLevel != 0 && numEntries == 0) ||
				numEntries > maxEntries {

				return fmt.Errorf("level %d has %d entries",
					level, numEntries)
			}
		} else if numEntries != maxEntries && numEntries != maxEntries/2 {
			return fmt.Errorf("level %d has %d entries", level,
				numEntries)
		}
		maxEntries *= 2
	}
	if totalEntries != expectedTotal {
		return fmt.Errorf("expected %d entries - got %d", expectedTotal,
			totalEntries)
	}

//
//
	expectedNum := uint32(0)
	for level := highestLevel + 1; level > 0; level-- {
		data := b.levels[keyForLevel(addrKey, level)]
		numEntries := len(data) / txEntrySize
		for i := 0; i < numEntries; i++ {
			start := i * txEntrySize
			num := byteOrder.Uint32(data[start:])
			if num != expectedNum {
				return fmt.Errorf("level %d offset %d does "+
					"not contain the expected number of "+
					"%d - got %d", level, i, num,
					expectedNum)
			}
			expectedNum++
		}
	}

	return nil
}

//
//
//文档。
func TestAddrIndexLevels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		key         [addrKeySize]byte
		numInsert   int
printLevels bool //
	}{
		{
			name:      "level 0 not full",
			numInsert: level0MaxEntries - 1,
		},
		{
			name:      "level 1 half",
			numInsert: level0MaxEntries + 1,
		},
		{
			name:      "level 1 full",
			numInsert: level0MaxEntries*2 + 1,
		},
		{
			name:      "level 2 half, level 1 half",
			numInsert: level0MaxEntries*3 + 1,
		},
		{
			name:      "level 2 half, level 1 full",
			numInsert: level0MaxEntries*4 + 1,
		},
		{
			name:      "level 2 full, level 1 half",
			numInsert: level0MaxEntries*5 + 1,
		},
		{
			name:      "level 2 full, level 1 full",
			numInsert: level0MaxEntries*6 + 1,
		},
		{
			name:      "level 3 half, level 2 half, level 1 half",
			numInsert: level0MaxEntries*7 + 1,
		},
		{
			name:      "level 3 full, level 2 half, level 1 full",
			numInsert: level0MaxEntries*12 + 1,
		},
	}

nextTest:
	for testNum, test := range tests {
//按顺序插入条目。
		populatedBucket := &addrIndexBucket{
			levels: make(map[[levelKeySize]byte][]byte),
		}
		for i := 0; i < test.numInsert; i++ {
			txLoc := wire.TxLoc{TxStart: i * 2}
			err := dbPutAddrIndexEntry(populatedBucket, test.key,
				uint32(i), txLoc)
			if err != nil {
				t.Errorf("dbPutAddrIndexEntry #%d (%s) - "+
					"unexpected error: %v", testNum,
					test.name, err)
				continue nextTest
			}
		}
		if test.printLevels {
			t.Log(populatedBucket.printLevels(test.key))
		}

//
//
//
//
//
//工作正常。
		for numDelete := 0; numDelete <= test.numInsert+1; numDelete++ {
//克隆填充的存储桶以运行每个删除操作。
			bucket := populatedBucket.Clone()

//删除此迭代的条目数。
			err := dbRemoveAddrIndexEntries(bucket, test.key,
				numDelete)
			if err != nil {
				if numDelete <= test.numInsert {
					t.Errorf("dbRemoveAddrIndexEntries (%s) "+
						" delete %d - unexpected error: "+
						"%v", test.name, numDelete, err)
					continue nextTest
				}
			}
			if test.printLevels {
				t.Log(bucket.printLevels(test.key))
			}

//健全检查水平，确保坚持所有
//规则。
			numExpected := test.numInsert
			if numDelete <= test.numInsert {
				numExpected -= numDelete
			}
			err = bucket.sanityCheck(test.key, numExpected)
			if err != nil {
				t.Errorf("sanity check fail (%s) delete %d: %v",
					test.name, numDelete, err)
				continue nextTest
			}
		}
	}
}
