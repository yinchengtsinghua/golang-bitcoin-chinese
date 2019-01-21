
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
	"crypto/sha256"
	"testing"
)

//TestimmutableEmpty确保对空的不可变Treap调用函数
//按预期工作。
func TestImmutableEmpty(t *testing.T) {
	t.Parallel()

//确保treap长度为预期值。
	testTreap := NewImmutable()
	if gotLen := testTreap.Len(); gotLen != 0 {
		t.Fatalf("Len: unexpected length - got %d, want %d", gotLen, 0)
	}

//确保报告的大小为0。
	if gotSize := testTreap.Size(); gotSize != 0 {
		t.Fatalf("Size: unexpected byte size - got %d, want 0",
			gotSize)
	}

//Ensure there are no errors with requesting keys from an empty treap.
	key := serializeUint32(0)
	if gotVal := testTreap.Has(key); gotVal {
		t.Fatalf("Has: unexpected result - got %v, want false", gotVal)
	}
	if gotVal := testTreap.Get(key); gotVal != nil {
		t.Fatalf("Get: unexpected result - got %x, want nil", gotVal)
	}

//从空的treap中删除键时，确保没有恐慌。
	testTreap.Delete(key)

//确保foreach在空treap上迭代的键数为
//零。
	var numIterated int
	testTreap.ForEach(func(k, v []byte) bool {
		numIterated++
		return true
	})
	if numIterated != 0 {
		t.Fatalf("ForEach: unexpected iterate count - got %d, want 0",
			numIterated)
	}
}

//有助于确保将密钥放入不可变的叛国罪
//顺序顺序按预期工作。
func TestImmutableSequential(t *testing.T) {
	t.Parallel()

//在检查多个treap时插入一组顺序键
//功能按预期工作。
	expectedSize := uint64(0)
	numItems := 1000
	testTreap := NewImmutable()
	for i := 0; i < numItems; i++ {
		key := serializeUint32(uint32(i))
		testTreap = testTreap.Put(key, key)

//确保treap长度为预期值。
		if gotLen := testTreap.Len(); gotLen != i+1 {
			t.Fatalf("Len #%d: unexpected length - got %d, want %d",
				i, gotLen, i+1)
		}

//确保Treap有钥匙。
		if !testTreap.Has(key) {
			t.Fatalf("Has #%d: key %q is not in treap", i, key)
		}

//从treap中获取密钥并确保它是预期的
//价值。
		if gotVal := testTreap.Get(key); !bytes.Equal(gotVal, key) {
			t.Fatalf("Get #%d: unexpected value - got %x, want %x",
				i, gotVal, key)
		}

//确保报告了预期的大小。
		expectedSize += (nodeFieldsSize + 8)
		if gotSize := testTreap.Size(); gotSize != expectedSize {
			t.Fatalf("Size #%d: unexpected byte size - got %d, "+
				"want %d", i, gotSize, expectedSize)
		}
	}

//确保foreach按顺序迭代所有键。
	var numIterated int
	testTreap.ForEach(func(k, v []byte) bool {
		wantKey := serializeUint32(uint32(numIterated))

//确保密钥符合预期。
		if !bytes.Equal(k, wantKey) {
			t.Fatalf("ForEach #%d: unexpected key - got %x, want %x",
				numIterated, k, wantKey)
		}

//确保值符合预期。
		if !bytes.Equal(v, wantKey) {
			t.Fatalf("ForEach #%d: unexpected value - got %x, want %x",
				numIterated, v, wantKey)
		}

		numIterated++
		return true
	})

//Ensure all items were iterated.
	if numIterated != numItems {
		t.Fatalf("ForEach: unexpected iterate count - got %d, want %d",
			numIterated, numItems)
	}

//在检查多个treap时逐个删除密钥
//功能按预期工作。
	for i := 0; i < numItems; i++ {
		key := serializeUint32(uint32(i))
		testTreap = testTreap.Delete(key)

//确保treap长度为预期值。
		if gotLen := testTreap.Len(); gotLen != numItems-i-1 {
			t.Fatalf("Len #%d: unexpected length - got %d, want %d",
				i, gotLen, numItems-i-1)
		}

//确保Treap不再拥有密钥。
		if testTreap.Has(key) {
			t.Fatalf("Has #%d: key %q is in treap", i, key)
		}

//从treap中获取不再存在的密钥并确保
//它是零。
		if gotVal := testTreap.Get(key); gotVal != nil {
			t.Fatalf("Get #%d: unexpected value - got %x, want nil",
				i, gotVal)
		}

//确保报告了预期的大小。
		expectedSize -= (nodeFieldsSize + 8)
		if gotSize := testTreap.Size(); gotSize != expectedSize {
			t.Fatalf("Size #%d: unexpected byte size - got %d, "+
				"want %d", i, gotSize, expectedSize)
		}
	}
}

//testmmutablereversequential确保将密钥放入不可变的
//按相反顺序排列的treap按预期工作。
func TestImmutableReverseSequential(t *testing.T) {
	t.Parallel()

//在检查多个treap时插入一组顺序键
//功能按预期工作。
	expectedSize := uint64(0)
	numItems := 1000
	testTreap := NewImmutable()
	for i := 0; i < numItems; i++ {
		key := serializeUint32(uint32(numItems - i - 1))
		testTreap = testTreap.Put(key, key)

//确保treap长度为预期值。
		if gotLen := testTreap.Len(); gotLen != i+1 {
			t.Fatalf("Len #%d: unexpected length - got %d, want %d",
				i, gotLen, i+1)
		}

//确保Treap有钥匙。
		if !testTreap.Has(key) {
			t.Fatalf("Has #%d: key %q is not in treap", i, key)
		}

//从treap中获取密钥并确保它是预期的
//价值。
		if gotVal := testTreap.Get(key); !bytes.Equal(gotVal, key) {
			t.Fatalf("Get #%d: unexpected value - got %x, want %x",
				i, gotVal, key)
		}

//确保报告了预期的大小。
		expectedSize += (nodeFieldsSize + 8)
		if gotSize := testTreap.Size(); gotSize != expectedSize {
			t.Fatalf("Size #%d: unexpected byte size - got %d, "+
				"want %d", i, gotSize, expectedSize)
		}
	}

//确保foreach按顺序迭代所有键。
	var numIterated int
	testTreap.ForEach(func(k, v []byte) bool {
		wantKey := serializeUint32(uint32(numIterated))

//确保密钥符合预期。
		if !bytes.Equal(k, wantKey) {
			t.Fatalf("ForEach #%d: unexpected key - got %x, want %x",
				numIterated, k, wantKey)
		}

//确保值符合预期。
		if !bytes.Equal(v, wantKey) {
			t.Fatalf("ForEach #%d: unexpected value - got %x, want %x",
				numIterated, v, wantKey)
		}

		numIterated++
		return true
	})

//确保所有项都已迭代。
	if numIterated != numItems {
		t.Fatalf("ForEach: unexpected iterate count - got %d, want %d",
			numIterated, numItems)
	}

//在检查多个treap时逐个删除密钥
//功能按预期工作。
	for i := 0; i < numItems; i++ {
//有意使用此处插入的相反顺序。
		key := serializeUint32(uint32(i))
		testTreap = testTreap.Delete(key)

//确保treap长度为预期值。
		if gotLen := testTreap.Len(); gotLen != numItems-i-1 {
			t.Fatalf("Len #%d: unexpected length - got %d, want %d",
				i, gotLen, numItems-i-1)
		}

//确保Treap不再拥有密钥。
		if testTreap.Has(key) {
			t.Fatalf("Has #%d: key %q is in treap", i, key)
		}

//从treap中获取不再存在的密钥并确保
//它是零。
		if gotVal := testTreap.Get(key); gotVal != nil {
			t.Fatalf("Get #%d: unexpected value - got %x, want nil",
				i, gotVal)
		}

//确保报告了预期的大小。
		expectedSize -= (nodeFieldsSize + 8)
		if gotSize := testTreap.Size(); gotSize != expectedSize {
			t.Fatalf("Size #%d: unexpected byte size - got %d, "+
				"want %d", i, gotSize, expectedSize)
		}
	}
}

//testmmutableunordered确保在
//没有一个部分顺序按预期工作。
func TestImmutableUnordered(t *testing.T) {
	t.Parallel()

//在检查多个
//treap函数按预期工作。
	expectedSize := uint64(0)
	numItems := 1000
	testTreap := NewImmutable()
	for i := 0; i < numItems; i++ {
//散列序列化的int以生成无序键。
		hash := sha256.Sum256(serializeUint32(uint32(i)))
		key := hash[:]
		testTreap = testTreap.Put(key, key)

//确保treap长度为预期值。
		if gotLen := testTreap.Len(); gotLen != i+1 {
			t.Fatalf("Len #%d: unexpected length - got %d, want %d",
				i, gotLen, i+1)
		}

//确保Treap有钥匙。
		if !testTreap.Has(key) {
			t.Fatalf("Has #%d: key %q is not in treap", i, key)
		}

//从treap中获取密钥并确保它是预期的
//价值。
		if gotVal := testTreap.Get(key); !bytes.Equal(gotVal, key) {
			t.Fatalf("Get #%d: unexpected value - got %x, want %x",
				i, gotVal, key)
		}

//确保报告了预期的大小。
		expectedSize += nodeFieldsSize + uint64(len(key)+len(key))
		if gotSize := testTreap.Size(); gotSize != expectedSize {
			t.Fatalf("Size #%d: unexpected byte size - got %d, "+
				"want %d", i, gotSize, expectedSize)
		}
	}

//在检查多个treap时逐个删除密钥
//功能按预期工作。
	for i := 0; i < numItems; i++ {
//散列序列化的int以生成无序键。
		hash := sha256.Sum256(serializeUint32(uint32(i)))
		key := hash[:]
		testTreap = testTreap.Delete(key)

//确保treap长度为预期值。
		if gotLen := testTreap.Len(); gotLen != numItems-i-1 {
			t.Fatalf("Len #%d: unexpected length - got %d, want %d",
				i, gotLen, numItems-i-1)
		}

//确保Treap不再拥有密钥。
		if testTreap.Has(key) {
			t.Fatalf("Has #%d: key %q is in treap", i, key)
		}

//从treap中获取不再存在的密钥并确保
//它是零。
		if gotVal := testTreap.Get(key); gotVal != nil {
			t.Fatalf("Get #%d: unexpected value - got %x, want nil",
				i, gotVal)
		}

//确保报告了预期的大小。
		expectedSize -= (nodeFieldsSize + 64)
		if gotSize := testTreap.Size(); gotSize != expectedSize {
			t.Fatalf("Size #%d: unexpected byte size - got %d, "+
				"want %d", i, gotSize, expectedSize)
		}
	}
}

//TestImmutableDuplicatePut ensures that putting a duplicate key into an
//不变的叛国罪按预期运作。
func TestImmutableDuplicatePut(t *testing.T) {
	t.Parallel()

	expectedVal := []byte("testval")
	expectedSize := uint64(0)
	numItems := 1000
	testTreap := NewImmutable()
	for i := 0; i < numItems; i++ {
		key := serializeUint32(uint32(i))
		testTreap = testTreap.Put(key, key)
		expectedSize += nodeFieldsSize + uint64(len(key)+len(key))

//放置一个具有预期最终值的重复键。
		testTreap = testTreap.Put(key, expectedVal)

//确保键仍然存在并且是新值。
		if gotVal := testTreap.Has(key); !gotVal {
			t.Fatalf("Has: unexpected result - got %v, want true",
				gotVal)
		}
		if gotVal := testTreap.Get(key); !bytes.Equal(gotVal, expectedVal) {
			t.Fatalf("Get: unexpected result - got %x, want %x",
				gotVal, expectedVal)
		}

//确保报告了预期的大小。
		expectedSize -= uint64(len(key))
		expectedSize += uint64(len(expectedVal))
		if gotSize := testTreap.Size(); gotSize != expectedSize {
			t.Fatalf("Size: unexpected byte size - got %d, want %d",
				gotSize, expectedSize)
		}
	}
}

//testimmutablenilvalue确保将nil值放入不可变的
//treap results in a key being added with an empty byte slice.
func TestImmutableNilValue(t *testing.T) {
	t.Parallel()

	key := serializeUint32(0)

//将该键的值设为零。
	testTreap := NewImmutable()
	testTreap = testTreap.Put(key, nil)

//确保键存在并且是空字节片。
	if gotVal := testTreap.Has(key); !gotVal {
		t.Fatalf("Has: unexpected result - got %v, want true", gotVal)
	}
	if gotVal := testTreap.Get(key); gotVal == nil {
		t.Fatalf("Get: unexpected result - got nil, want empty slice")
	}
	if gotVal := testTreap.Get(key); len(gotVal) != 0 {
		t.Fatalf("Get: unexpected result - got %x, want empty slice",
			gotVal)
	}
}

//TestimmutableForEachStopIterator确保从ForEach返回false
//对不可变的treap的回调会提前停止迭代。
func TestImmutableForEachStopIterator(t *testing.T) {
	t.Parallel()

//插入几把钥匙。
	numItems := 10
	testTreap := NewImmutable()
	for i := 0; i < numItems; i++ {
		key := serializeUint32(uint32(i))
		testTreap = testTreap.Put(key, key)
	}

//确保foreach在调用方错误返回时提前退出。
	var numIterated int
	testTreap.ForEach(func(k, v []byte) bool {
		numIterated++
		return numIterated != numItems/2
	})
	if numIterated != numItems/2 {
		t.Fatalf("ForEach: unexpected iterate count - got %d, want %d",
			numIterated, numItems/2)
	}
}

//可证明的快照确保不可变的叛国罪实际上是不可变的
//参考先前的TREAP，执行突变，然后
//确保引用的treap没有应用突变。
func TestImmutableSnapshot(t *testing.T) {
	t.Parallel()

//在检查多个treap时插入一组顺序键
//功能按预期工作。
	expectedSize := uint64(0)
	numItems := 1000
	testTreap := NewImmutable()
	for i := 0; i < numItems; i++ {
		treapSnap := testTreap

		key := serializeUint32(uint32(i))
		testTreap = testTreap.Put(key, key)

//确保treap快照的长度是预期的
//价值。
		if gotLen := treapSnap.Len(); gotLen != i {
			t.Fatalf("Len #%d: unexpected length - got %d, want %d",
				i, gotLen, i)
		}

//确保treap快照没有密钥。
		if treapSnap.Has(key) {
			t.Fatalf("Has #%d: key %q is in treap", i, key)
		}

//获取Treap快照中不存在的密钥，然后
//确保它是零。
		if gotVal := treapSnap.Get(key); gotVal != nil {
			t.Fatalf("Get #%d: unexpected value - got %x, want nil",
				i, gotVal)
		}

//确保报告了预期的大小。
		if gotSize := treapSnap.Size(); gotSize != expectedSize {
			t.Fatalf("Size #%d: unexpected byte size - got %d, "+
				"want %d", i, gotSize, expectedSize)
		}
		expectedSize += (nodeFieldsSize + 8)
	}

//在检查多个treap时逐个删除密钥
//功能按预期工作。
	for i := 0; i < numItems; i++ {
		treapSnap := testTreap

		key := serializeUint32(uint32(i))
		testTreap = testTreap.Delete(key)

//确保treap快照的长度是预期的
//价值。
		if gotLen := treapSnap.Len(); gotLen != numItems-i {
			t.Fatalf("Len #%d: unexpected length - got %d, want %d",
				i, gotLen, numItems-i)
		}

//确保treap快照仍有密钥。
		if !treapSnap.Has(key) {
			t.Fatalf("Has #%d: key %q is not in treap", i, key)
		}

//从treap快照中获取密钥并确保它仍然
//预期值。
		if gotVal := treapSnap.Get(key); !bytes.Equal(gotVal, key) {
			t.Fatalf("Get #%d: unexpected value - got %x, want %x",
				i, gotVal, key)
		}

//确保报告了预期的大小。
		if gotSize := treapSnap.Size(); gotSize != expectedSize {
			t.Fatalf("Size #%d: unexpected byte size - got %d, "+
				"want %d", i, gotSize, expectedSize)
		}
		expectedSize -= (nodeFieldsSize + 8)
	}
}
