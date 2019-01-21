
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

package treap

import (
	"bytes"
	"encoding/binary"
	"testing"
)

//TestMutableIterator确保可变Treap的一般行为
//迭代器和预期一样，包括first、last、ordered和reverse的测试
//有序迭代，限制范围，寻找和最初不定位。
func TestMutableIterator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		numKeys       int
		step          int
		startKey      []byte
		limitKey      []byte
		expectedFirst []byte
		expectedLast  []byte
		seekKey       []byte
		expectedSeek  []byte
	}{
//无范围限制。值是集合（0，1，2，…，49）。
//查找现有值。
		{
			numKeys:       50,
			step:          1,
			expectedFirst: serializeUint32(0),
			expectedLast:  serializeUint32(49),
			seekKey:       serializeUint32(12),
			expectedSeek:  serializeUint32(12),
		},

//限于范围[24，结束]。值是集合
//(0, 2, 4, ..., 48).  Seek value that doesn't exist and is
//大于最大现有密钥。
		{
			numKeys:       50,
			step:          2,
			startKey:      serializeUint32(24),
			expectedFirst: serializeUint32(24),
			expectedLast:  serializeUint32(48),
			seekKey:       serializeUint32(49),
			expectedSeek:  nil,
		},

//限制在范围内[开始，25）。值是集合
//（0，3，6，…，48）。寻找不存在但存在的价值
//before an existing value within the range.
		{
			numKeys:       50,
			step:          3,
			limitKey:      serializeUint32(25),
			expectedFirst: serializeUint32(0),
			expectedLast:  serializeUint32(24),
			seekKey:       serializeUint32(17),
			expectedSeek:  serializeUint32(18),
		},

//限于范围[10，21]。值是集合
//（0，4，…，48）。查找存在但位于
//最小允许范围。
		{
			numKeys:       50,
			step:          4,
			startKey:      serializeUint32(10),
			limitKey:      serializeUint32(21),
			expectedFirst: serializeUint32(12),
			expectedLast:  serializeUint32(20),
			seekKey:       serializeUint32(4),
			expectedSeek:  nil,
		},

//受前缀0,0,0，范围[0,0,0，0,0,1）限制。
//因为它是一个字节比较，0,0,0，…<0,0,1。
//在允许的范围内查找现有值。
		{
			numKeys:       300,
			step:          1,
			startKey:      []byte{0x00, 0x00, 0x00},
			limitKey:      []byte{0x00, 0x00, 0x01},
			expectedFirst: serializeUint32(0),
			expectedLast:  serializeUint32(255),
			seekKey:       serializeUint32(100),
			expectedSeek:  serializeUint32(100),
		},
	}

testLoop:
	for i, test := range tests {
//插入一串钥匙。
		testTreap := NewMutable()
		for i := 0; i < test.numKeys; i += test.step {
			key := serializeUint32(uint32(i))
			testTreap.Put(key, key)
		}

//Create new iterator limited by the test params.
		iter := testTreap.Iterator(test.startKey, test.limitKey)

//确保第一项准确无误。
		hasFirst := iter.First()
		if !hasFirst && test.expectedFirst != nil {
			t.Errorf("First #%d: unexpected exhausted iterator", i)
			continue
		}
		gotKey := iter.Key()
		if !bytes.Equal(gotKey, test.expectedFirst) {
			t.Errorf("First.Key #%d: unexpected key - got %x, "+
				"want %x", i, gotKey, test.expectedFirst)
			continue
		}
		gotVal := iter.Value()
		if !bytes.Equal(gotVal, test.expectedFirst) {
			t.Errorf("First.Value #%d: unexpected value - got %x, "+
				"want %x", i, gotVal, test.expectedFirst)
			continue
		}

//确保迭代器按顺序提供预期的项。
		curNum := binary.BigEndian.Uint32(test.expectedFirst)
		for iter.Next() {
			curNum += uint32(test.step)

//确保密钥如预期的那样。
			gotKey := iter.Key()
			expectedKey := serializeUint32(curNum)
			if !bytes.Equal(gotKey, expectedKey) {
				t.Errorf("iter.Key #%d (%d): unexpected key - "+
					"got %x, want %x", i, curNum, gotKey,
					expectedKey)
				continue testLoop
			}

//确保值符合预期。
			gotVal := iter.Value()
			if !bytes.Equal(gotVal, expectedKey) {
				t.Errorf("iter.Value #%d (%d): unexpected "+
					"value - got %x, want %x", i, curNum,
					gotVal, expectedKey)
				continue testLoop
			}
		}

//确保迭代器已用完。
		if iter.Valid() {
			t.Errorf("Valid #%d: iterator should be exhausted", i)
			continue
		}

//确保最后一项准确无误。
		hasLast := iter.Last()
		if !hasLast && test.expectedLast != nil {
			t.Errorf("Last #%d: unexpected exhausted iterator", i)
			continue
		}
		gotKey = iter.Key()
		if !bytes.Equal(gotKey, test.expectedLast) {
			t.Errorf("Last.Key #%d: unexpected key - got %x, "+
				"want %x", i, gotKey, test.expectedLast)
			continue
		}
		gotVal = iter.Value()
		if !bytes.Equal(gotVal, test.expectedLast) {
			t.Errorf("Last.Value #%d: unexpected value - got %x, "+
				"want %x", i, gotVal, test.expectedLast)
			continue
		}

//确保迭代器返回预期的项
//秩序。
		curNum = binary.BigEndian.Uint32(test.expectedLast)
		for iter.Prev() {
			curNum -= uint32(test.step)

//确保密钥如预期的那样。
			gotKey := iter.Key()
			expectedKey := serializeUint32(curNum)
			if !bytes.Equal(gotKey, expectedKey) {
				t.Errorf("iter.Key #%d (%d): unexpected key - "+
					"got %x, want %x", i, curNum, gotKey,
					expectedKey)
				continue testLoop
			}

//确保值符合预期。
			gotVal := iter.Value()
			if !bytes.Equal(gotVal, expectedKey) {
				t.Errorf("iter.Value #%d (%d): unexpected "+
					"value - got %x, want %x", i, curNum,
					gotVal, expectedKey)
				continue testLoop
			}
		}

//确保迭代器已用完。
		if iter.Valid() {
			t.Errorf("Valid #%d: iterator should be exhausted", i)
			continue
		}

//查找提供的密钥。
		seekValid := iter.Seek(test.seekKey)
		if !seekValid && test.expectedSeek != nil {
			t.Errorf("Seek #%d: unexpected exhausted iterator", i)
			continue
		}
		gotKey = iter.Key()
		if !bytes.Equal(gotKey, test.expectedSeek) {
			t.Errorf("Seek.Key #%d: unexpected key - got %x, "+
				"want %x", i, gotKey, test.expectedSeek)
			continue
		}
		gotVal = iter.Value()
		if !bytes.Equal(gotVal, test.expectedSeek) {
			t.Errorf("Seek.Value #%d: unexpected value - got %x, "+
				"want %x", i, gotVal, test.expectedSeek)
			continue
		}

//Recreate the iterator and ensure calling Next on it before it
//已经定位的给出了第一个元素。
		iter = testTreap.Iterator(test.startKey, test.limitKey)
		hasNext := iter.Next()
		if !hasNext && test.expectedFirst != nil {
			t.Errorf("Next #%d: unexpected exhausted iterator", i)
			continue
		}
		gotKey = iter.Key()
		if !bytes.Equal(gotKey, test.expectedFirst) {
			t.Errorf("Next.Key #%d: unexpected key - got %x, "+
				"want %x", i, gotKey, test.expectedFirst)
			continue
		}
		gotVal = iter.Value()
		if !bytes.Equal(gotVal, test.expectedFirst) {
			t.Errorf("Next.Value #%d: unexpected value - got %x, "+
				"want %x", i, gotVal, test.expectedFirst)
			continue
		}

//重新创建迭代器并确保在它之前对它调用prev
//已经定位的给出了第一个元素。
		iter = testTreap.Iterator(test.startKey, test.limitKey)
		hasPrev := iter.Prev()
		if !hasPrev && test.expectedLast != nil {
			t.Errorf("Prev #%d: unexpected exhausted iterator", i)
			continue
		}
		gotKey = iter.Key()
		if !bytes.Equal(gotKey, test.expectedLast) {
			t.Errorf("Prev.Key #%d: unexpected key - got %x, "+
				"want %x", i, gotKey, test.expectedLast)
			continue
		}
		gotVal = iter.Value()
		if !bytes.Equal(gotVal, test.expectedLast) {
			t.Errorf("Next.Value #%d: unexpected value - got %x, "+
				"want %x", i, gotVal, test.expectedLast)
			continue
		}
	}
}

//TestMutableEmptyIterator确保各种函数的行为
//可变叛国罪为空时应为。
func TestMutableEmptyIterator(t *testing.T) {
	t.Parallel()

//针对空的treap创建迭代器。
	testTreap := NewMutable()
	iter := testTreap.Iterator(nil, nil)

//确保空迭代器的有效性报告已用完。
	if iter.Valid() {
		t.Fatal("Valid: iterator should be exhausted")
	}

//确保第一个和最后一个空的迭代器报告它已耗尽。
	if iter.First() {
		t.Fatal("First: iterator should be exhausted")
	}
	if iter.Last() {
		t.Fatal("Last: iterator should be exhausted")
	}

//Ensure Next and Prev on empty iterator report it as exhausted.
	if iter.Next() {
		t.Fatal("Next: iterator should be exhausted")
	}
	if iter.Prev() {
		t.Fatal("Prev: iterator should be exhausted")
	}

//确保空迭代器上的键和值为零。
	if gotKey := iter.Key(); gotKey != nil {
		t.Fatalf("Key: should be nil - got %q", gotKey)
	}
	if gotVal := iter.Value(); gotVal != nil {
		t.Fatalf("Value: should be nil - got %q", gotVal)
	}

//确保在强制重新搜索
//空迭代器。
	iter.ForceReseek()
	if iter.Next() {
		t.Fatal("Next: iterator should be exhausted")
	}
	iter.ForceReseek()
	if iter.Prev() {
		t.Fatal("Prev: iterator should be exhausted")
	}
}

//testerateOrUpdates确保在迭代器上发出对forceReseek的调用
//有潜在的可变叛国者更新工作如预期。
func TestIteratorUpdates(t *testing.T) {
	t.Parallel()

//Create a new treap with various values inserted in no particular
//秩序。生成的键是集合（2、4、7、11、18、25）。
	testTreap := NewMutable()
	testTreap.Put(serializeUint32(7), nil)
	testTreap.Put(serializeUint32(2), nil)
	testTreap.Put(serializeUint32(18), nil)
	testTreap.Put(serializeUint32(11), nil)
	testTreap.Put(serializeUint32(25), nil)
	testTreap.Put(serializeUint32(4), nil)

//Create an iterator against the treap with a range that excludes the
//最低和最高条目。有限集为（4、7、11、18）
	iter := testTreap.Iterator(serializeUint32(3), serializeUint32(25))

//从范围中间删除一个键并通知迭代器
//强迫重新开始
	testTreap.Delete(serializeUint32(11))
	iter.ForceReseek()

//确保在强制reseek之后对迭代器调用next
//提供所需的密钥。此时有限的密钥集是
//（4、7、18）迭代器尚未定位。
	if !iter.Next() {
		t.Fatal("ForceReseek.Next: unexpected exhausted iterator")
	}
	wantKey := serializeUint32(4)
	gotKey := iter.Key()
	if !bytes.Equal(gotKey, wantKey) {
		t.Fatalf("ForceReseek.Key: unexpected key - got %x, want %x",
			gotKey, wantKey)
	}

//删除迭代器当前所在的键并通知
//迭代器强制重置。
	testTreap.Delete(serializeUint32(4))
	iter.ForceReseek()

//确保在强制reseek之后对迭代器调用next
//提供所需的密钥。此时有限的密钥集是
//（7，18）迭代器定位在7之前的已删除条目处。
	if !iter.Next() {
		t.Fatal("ForceReseek.Next: unexpected exhausted iterator")
	}
	wantKey = serializeUint32(7)
	gotKey = iter.Key()
	if !bytes.Equal(gotKey, wantKey) {
		t.Fatalf("ForceReseek.Key: unexpected key - got %x, want %x",
			gotKey, wantKey)
	}

//在迭代器位于和的当前键之前添加一个键
//通知迭代器强制重置。
	testTreap.Put(serializeUint32(4), nil)
	iter.ForceReseek()

//确保在强制reseek之后对迭代器调用prev
//提供所需的密钥。此时有限的密钥集是
//（4，7，18）迭代器位于7。
	if !iter.Prev() {
		t.Fatal("ForceReseek.Prev: unexpected exhausted iterator")
	}
	wantKey = serializeUint32(4)
	gotKey = iter.Key()
	if !bytes.Equal(gotKey, wantKey) {
		t.Fatalf("ForceReseek.Key: unexpected key - got %x, want %x",
			gotKey, wantKey)
	}

//删除迭代器通常移动到的下一个键，然后通知
//用于强制重置的迭代器。
	testTreap.Delete(serializeUint32(7))
	iter.ForceReseek()

//确保在强制reseek之后对迭代器调用next
//提供所需的密钥。此时有限的密钥集是
//（4，18）迭代器定位在4。
	if !iter.Next() {
		t.Fatal("ForceReseek.Next: unexpected exhausted iterator")
	}
	wantKey = serializeUint32(18)
	gotKey = iter.Key()
	if !bytes.Equal(gotKey, wantKey) {
		t.Fatalf("ForceReseek.Key: unexpected key - got %x, want %x",
			gotKey, wantKey)
	}
}

//TestimmutableIterator确保不可变的Treap的一般行为
//迭代器和预期一样，包括first、last、ordered和reverse的测试
//有序迭代，限制范围，寻找和最初不定位。
func TestImmutableIterator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		numKeys       int
		step          int
		startKey      []byte
		limitKey      []byte
		expectedFirst []byte
		expectedLast  []byte
		seekKey       []byte
		expectedSeek  []byte
	}{
//无范围限制。值是集合（0，1，2，…，49）。
//查找现有值。
		{
			numKeys:       50,
			step:          1,
			expectedFirst: serializeUint32(0),
			expectedLast:  serializeUint32(49),
			seekKey:       serializeUint32(12),
			expectedSeek:  serializeUint32(12),
		},

//限于范围[24，结束]。值是集合
//（0，2，4，…，48）。寻找不存在的价值
//大于最大现有密钥。
		{
			numKeys:       50,
			step:          2,
			startKey:      serializeUint32(24),
			expectedFirst: serializeUint32(24),
			expectedLast:  serializeUint32(48),
			seekKey:       serializeUint32(49),
			expectedSeek:  nil,
		},

//限制在范围内[开始，25）。值是集合
//（0，3，6，…，48）。寻找不存在但存在的价值
//在该范围内的现有值之前。
		{
			numKeys:       50,
			step:          3,
			limitKey:      serializeUint32(25),
			expectedFirst: serializeUint32(0),
			expectedLast:  serializeUint32(24),
			seekKey:       serializeUint32(17),
			expectedSeek:  serializeUint32(18),
		},

//限于范围[10，21]。值是集合
//（0，4，…，48）。查找存在但位于
//最小允许范围。
		{
			numKeys:       50,
			step:          4,
			startKey:      serializeUint32(10),
			limitKey:      serializeUint32(21),
			expectedFirst: serializeUint32(12),
			expectedLast:  serializeUint32(20),
			seekKey:       serializeUint32(4),
			expectedSeek:  nil,
		},

//受前缀0,0,0，范围[0,0,0，0,0,1）限制。
//因为它是一个字节比较，0,0,0，…<0,0,1。
//在允许的范围内查找现有值。
		{
			numKeys:       300,
			step:          1,
			startKey:      []byte{0x00, 0x00, 0x00},
			limitKey:      []byte{0x00, 0x00, 0x01},
			expectedFirst: serializeUint32(0),
			expectedLast:  serializeUint32(255),
			seekKey:       serializeUint32(100),
			expectedSeek:  serializeUint32(100),
		},
	}

testLoop:
	for i, test := range tests {
//插入一串钥匙。
		testTreap := NewImmutable()
		for i := 0; i < test.numKeys; i += test.step {
			key := serializeUint32(uint32(i))
			testTreap = testTreap.Put(key, key)
		}

//创建受测试参数限制的新迭代器。
		iter := testTreap.Iterator(test.startKey, test.limitKey)

//确保第一项准确无误。
		hasFirst := iter.First()
		if !hasFirst && test.expectedFirst != nil {
			t.Errorf("First #%d: unexpected exhausted iterator", i)
			continue
		}
		gotKey := iter.Key()
		if !bytes.Equal(gotKey, test.expectedFirst) {
			t.Errorf("First.Key #%d: unexpected key - got %x, "+
				"want %x", i, gotKey, test.expectedFirst)
			continue
		}
		gotVal := iter.Value()
		if !bytes.Equal(gotVal, test.expectedFirst) {
			t.Errorf("First.Value #%d: unexpected value - got %x, "+
				"want %x", i, gotVal, test.expectedFirst)
			continue
		}

//确保迭代器按顺序提供预期的项。
		curNum := binary.BigEndian.Uint32(test.expectedFirst)
		for iter.Next() {
			curNum += uint32(test.step)

//确保密钥如预期的那样。
			gotKey := iter.Key()
			expectedKey := serializeUint32(curNum)
			if !bytes.Equal(gotKey, expectedKey) {
				t.Errorf("iter.Key #%d (%d): unexpected key - "+
					"got %x, want %x", i, curNum, gotKey,
					expectedKey)
				continue testLoop
			}

//确保值符合预期。
			gotVal := iter.Value()
			if !bytes.Equal(gotVal, expectedKey) {
				t.Errorf("iter.Value #%d (%d): unexpected "+
					"value - got %x, want %x", i, curNum,
					gotVal, expectedKey)
				continue testLoop
			}
		}

//确保迭代器已用完。
		if iter.Valid() {
			t.Errorf("Valid #%d: iterator should be exhausted", i)
			continue
		}

//确保最后一项准确无误。
		hasLast := iter.Last()
		if !hasLast && test.expectedLast != nil {
			t.Errorf("Last #%d: unexpected exhausted iterator", i)
			continue
		}
		gotKey = iter.Key()
		if !bytes.Equal(gotKey, test.expectedLast) {
			t.Errorf("Last.Key #%d: unexpected key - got %x, "+
				"want %x", i, gotKey, test.expectedLast)
			continue
		}
		gotVal = iter.Value()
		if !bytes.Equal(gotVal, test.expectedLast) {
			t.Errorf("Last.Value #%d: unexpected value - got %x, "+
				"want %x", i, gotVal, test.expectedLast)
			continue
		}

//确保迭代器返回预期的项
//秩序。
		curNum = binary.BigEndian.Uint32(test.expectedLast)
		for iter.Prev() {
			curNum -= uint32(test.step)

//确保密钥如预期的那样。
			gotKey := iter.Key()
			expectedKey := serializeUint32(curNum)
			if !bytes.Equal(gotKey, expectedKey) {
				t.Errorf("iter.Key #%d (%d): unexpected key - "+
					"got %x, want %x", i, curNum, gotKey,
					expectedKey)
				continue testLoop
			}

//确保值符合预期。
			gotVal := iter.Value()
			if !bytes.Equal(gotVal, expectedKey) {
				t.Errorf("iter.Value #%d (%d): unexpected "+
					"value - got %x, want %x", i, curNum,
					gotVal, expectedKey)
				continue testLoop
			}
		}

//确保迭代器已用完。
		if iter.Valid() {
			t.Errorf("Valid #%d: iterator should be exhausted", i)
			continue
		}

//查找提供的密钥。
		seekValid := iter.Seek(test.seekKey)
		if !seekValid && test.expectedSeek != nil {
			t.Errorf("Seek #%d: unexpected exhausted iterator", i)
			continue
		}
		gotKey = iter.Key()
		if !bytes.Equal(gotKey, test.expectedSeek) {
			t.Errorf("Seek.Key #%d: unexpected key - got %x, "+
				"want %x", i, gotKey, test.expectedSeek)
			continue
		}
		gotVal = iter.Value()
		if !bytes.Equal(gotVal, test.expectedSeek) {
			t.Errorf("Seek.Value #%d: unexpected value - got %x, "+
				"want %x", i, gotVal, test.expectedSeek)
			continue
		}

//重新创建迭代器并确保在它之前对它调用next
//已经定位的给出了第一个元素。
		iter = testTreap.Iterator(test.startKey, test.limitKey)
		hasNext := iter.Next()
		if !hasNext && test.expectedFirst != nil {
			t.Errorf("Next #%d: unexpected exhausted iterator", i)
			continue
		}
		gotKey = iter.Key()
		if !bytes.Equal(gotKey, test.expectedFirst) {
			t.Errorf("Next.Key #%d: unexpected key - got %x, "+
				"want %x", i, gotKey, test.expectedFirst)
			continue
		}
		gotVal = iter.Value()
		if !bytes.Equal(gotVal, test.expectedFirst) {
			t.Errorf("Next.Value #%d: unexpected value - got %x, "+
				"want %x", i, gotVal, test.expectedFirst)
			continue
		}

//重新创建迭代器并确保在它之前对它调用prev
//已经定位的给出了第一个元素。
		iter = testTreap.Iterator(test.startKey, test.limitKey)
		hasPrev := iter.Prev()
		if !hasPrev && test.expectedLast != nil {
			t.Errorf("Prev #%d: unexpected exhausted iterator", i)
			continue
		}
		gotKey = iter.Key()
		if !bytes.Equal(gotKey, test.expectedLast) {
			t.Errorf("Prev.Key #%d: unexpected key - got %x, "+
				"want %x", i, gotKey, test.expectedLast)
			continue
		}
		gotVal = iter.Value()
		if !bytes.Equal(gotVal, test.expectedLast) {
			t.Errorf("Next.Value #%d: unexpected value - got %x, "+
				"want %x", i, gotVal, test.expectedLast)
			continue
		}
	}
}

//TestimmutableEmptyIterator确保各种函数的行为
//当不变的Treap为空时应为。
func TestImmutableEmptyIterator(t *testing.T) {
	t.Parallel()

//针对空的treap创建迭代器。
	testTreap := NewImmutable()
	iter := testTreap.Iterator(nil, nil)

//确保空迭代器的有效性报告已用完。
	if iter.Valid() {
		t.Fatal("Valid: iterator should be exhausted")
	}

//确保第一个和最后一个空的迭代器报告它已耗尽。
	if iter.First() {
		t.Fatal("First: iterator should be exhausted")
	}
	if iter.Last() {
		t.Fatal("Last: iterator should be exhausted")
	}

//确保空迭代器上的next和prev报告它已耗尽。
	if iter.Next() {
		t.Fatal("Next: iterator should be exhausted")
	}
	if iter.Prev() {
		t.Fatal("Prev: iterator should be exhausted")
	}

//确保空迭代器上的键和值为零。
	if gotKey := iter.Key(); gotKey != nil {
		t.Fatalf("Key: should be nil - got %q", gotKey)
	}
	if gotVal := iter.Value(); gotVal != nil {
		t.Fatalf("Value: should be nil - got %q", gotVal)
	}

//确保在不可变的treap迭代器上调用forcereseek不会
//导致任何问题，因为它只适用于可变的treap迭代器。
	iter.ForceReseek()
	if iter.Next() {
		t.Fatal("Next: iterator should be exhausted")
	}
	iter.ForceReseek()
	if iter.Prev() {
		t.Fatal("Prev: iterator should be exhausted")
	}
}
