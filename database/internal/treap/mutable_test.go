
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

//TestMutableEmpty确保空的可变Treap上的调用函数的工作方式为
//预期。
func TestMutableEmpty(t *testing.T) {
	t.Parallel()

//确保treap长度为预期值。
	testTreap := NewMutable()
	if gotLen := testTreap.Len(); gotLen != 0 {
		t.Fatalf("Len: unexpected length - got %d, want %d", gotLen, 0)
	}

//确保报告的大小为0。
	if gotSize := testTreap.Size(); gotSize != 0 {
		t.Fatalf("Size: unexpected byte size - got %d, want 0",
			gotSize)
	}

//确保从空TRAP请求密钥没有错误。
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

//TestMutabReSeET确保重置现有的可变TRAP工作
//预期。
func TestMutableReset(t *testing.T) {
	t.Parallel()

//插入几把钥匙。
	numItems := 10
	testTreap := NewMutable()
	for i := 0; i < numItems; i++ {
		key := serializeUint32(uint32(i))
		testTreap.Put(key, key)
	}

//重置它。
	testTreap.Reset()

//确保treap长度现在为0。
	if gotLen := testTreap.Len(); gotLen != 0 {
		t.Fatalf("Len: unexpected length - got %d, want %d", gotLen, 0)
	}

//确保报告的大小现在为0。
	if gotSize := testTreap.Size(); gotSize != 0 {
		t.Fatalf("Size: unexpected byte size - got %d, want 0",
			gotSize)
	}

//确保Treap不再有任何钥匙。
	for i := 0; i < numItems; i++ {
		key := serializeUint32(uint32(i))

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
	}

//确保foreach迭代的键数为零。
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

//TestMutableSequential确保将密钥放入可变的treap中
//顺序顺序按预期工作。
func TestMutableSequential(t *testing.T) {
	t.Parallel()

//在检查多个treap时插入一组顺序键
//功能按预期工作。
	expectedSize := uint64(0)
	numItems := 1000
	testTreap := NewMutable()
	for i := 0; i < numItems; i++ {
		key := serializeUint32(uint32(i))
		testTreap.Put(key, key)

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
		key := serializeUint32(uint32(i))
		testTreap.Delete(key)

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

//testmutablereversequential确保将密钥放入可变的treap中
//按相反的顺序，按预期工作。
func TestMutableReverseSequential(t *testing.T) {
	t.Parallel()

//在检查多个treap时插入一组顺序键
//功能按预期工作。
	expectedSize := uint64(0)
	numItems := 1000
	testTreap := NewMutable()
	for i := 0; i < numItems; i++ {
		key := serializeUint32(uint32(numItems - i - 1))
		testTreap.Put(key, key)

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
		testTreap.Delete(key)

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

//testmutableunordered确保在no中将密钥放入可变的treap
//部分顺序按预期工作。
func TestMutableUnordered(t *testing.T) {
	t.Parallel()

//在检查多个
//treap函数按预期工作。
	expectedSize := uint64(0)
	numItems := 1000
	testTreap := NewMutable()
	for i := 0; i < numItems; i++ {
//散列序列化的int以生成无序键。
		hash := sha256.Sum256(serializeUint32(uint32(i)))
		key := hash[:]
		testTreap.Put(key, key)

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
		testTreap.Delete(key)

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

//TestMutableDuplicatePut确保将重复的密钥放入可变的
//treap updates the existing value.
func TestMutableDuplicatePut(t *testing.T) {
	t.Parallel()

	key := serializeUint32(0)
	val := []byte("testval")

//将键放置两次，第二次放置为预期的最终值。
	testTreap := NewMutable()
	testTreap.Put(key, key)
	testTreap.Put(key, val)

//确保键仍然存在并且是新值。
	if gotVal := testTreap.Has(key); !gotVal {
		t.Fatalf("Has: unexpected result - got %v, want true", gotVal)
	}
	if gotVal := testTreap.Get(key); !bytes.Equal(gotVal, val) {
		t.Fatalf("Get: unexpected result - got %x, want %x", gotVal, val)
	}

//确保报告了预期的大小。
	expectedSize := uint64(nodeFieldsSize + len(key) + len(val))
	if gotSize := testTreap.Size(); gotSize != expectedSize {
		t.Fatalf("Size: unexpected byte size - got %d, want %d",
			gotSize, expectedSize)
	}
}

//testmutablenilvalue确保将nil值放入可变的treap中
//结果添加了一个带有空字节片的键。
func TestMutableNilValue(t *testing.T) {
	t.Parallel()

	key := serializeUint32(0)

//将该键的值设为零。
	testTreap := NewMutable()
	testTreap.Put(key, nil)

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

//TestMutableForEachStopIterator确保从ForEach返回false
//可变treap的回调会提前停止迭代。
func TestMutableForEachStopIterator(t *testing.T) {
	t.Parallel()

//插入几把钥匙。
	numItems := 10
	testTreap := NewMutable()
	for i := 0; i < numItems; i++ {
		key := serializeUint32(uint32(i))
		testTreap.Put(key, key)
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
