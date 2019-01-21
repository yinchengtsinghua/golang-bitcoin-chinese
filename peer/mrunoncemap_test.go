
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package peer

import (
	"fmt"
	"testing"
)

//testmrunoncemap确保mrunoncemap的行为符合预期，包括
//限制、驱逐最近使用过的条目、删除特定条目，
//以及存在测试。
func TestMruNonceMap(t *testing.T) {
//创建一组用于测试MRU nonce代码的伪nonce。
	numNonces := 10
	nonces := make([]uint64, 0, numNonces)
	for i := 0; i < numNonces; i++ {
		nonces = append(nonces, uint64(i))
	}

	tests := []struct {
		name  string
		limit int
	}{
		{name: "limit 0", limit: 0},
		{name: "limit 1", limit: 1},
		{name: "limit 5", limit: 5},
		{name: "limit 7", limit: 7},
		{name: "limit one less than available", limit: numNonces - 1},
		{name: "limit all available", limit: numNonces},
	}

testLoop:
	for i, test := range tests {
//创建受指定测试限制的新mru nonce映射
//限制并添加所有测试时态。这将导致
//证据，因为有比限制更多的测试时刻。
		mruNonceMap := newMruNonceMap(uint(test.limit))
		for j := 0; j < numNonces; j++ {
			mruNonceMap.Add(nonces[j])
		}

//确保列表中最近条目的数量有限
//存在。
		for j := numNonces - test.limit; j < numNonces; j++ {
			if !mruNonceMap.Exists(nonces[j]) {
				t.Errorf("Exists #%d (%s) entry %d does not "+
					"exist", i, test.name, nonces[j])
				continue testLoop
			}
		}

//确保在最新的有限数量之前输入
//列表中的条目不存在。
		for j := 0; j < numNonces-test.limit; j++ {
			if mruNonceMap.Exists(nonces[j]) {
				t.Errorf("Exists #%d (%s) entry %d exists", i,
					test.name, nonces[j])
				continue testLoop
			}
		}

//读取当前应为最近最少的条目
//所以它成为最近使用的条目，然后
//通过添加不存在的条目和
//确保收回的条目是最近使用最少的新条目
//条目。
//
//此检查至少需要2个条目。
		if test.limit > 1 {
			origLruIndex := numNonces - test.limit
			mruNonceMap.Add(nonces[origLruIndex])

			mruNonceMap.Add(uint64(numNonces) + 1)

//确保原始LRU条目仍然存在，因为它
//已更新，应该已成为MRU条目。
			if !mruNonceMap.Exists(nonces[origLruIndex]) {
				t.Errorf("MRU #%d (%s) entry %d does not exist",
					i, test.name, nonces[origLruIndex])
				continue testLoop
			}

//确保本应成为新LRU的条目
//条目被逐出。
			newLruIndex := origLruIndex + 1
			if mruNonceMap.Exists(nonces[newLruIndex]) {
				t.Errorf("MRU #%d (%s) entry %d exists", i,
					test.name, nonces[newLruIndex])
				continue testLoop
			}
		}

//Delete all of the entries in the list, including those that
//不存在于地图中，并确保它们不再存在。
		for j := 0; j < numNonces; j++ {
			mruNonceMap.Delete(nonces[j])
			if mruNonceMap.Exists(nonces[j]) {
				t.Errorf("Delete #%d (%s) entry %d exists", i,
					test.name, nonces[j])
				continue testLoop
			}
		}
	}
}

//testmrunoncemapStringer测试mrunoncemap类型的字符串化输出。
func TestMruNonceMapStringer(t *testing.T) {
//创建两个用于测试MRU nonce的假nonce
//纵梁代码。
	nonce1 := uint64(10)
	nonce2 := uint64(20)

//创建新的mru nonce映射并添加nonce。
	mruNonceMap := newMruNonceMap(uint(2))
	mruNonceMap.Add(nonce1)
	mruNonceMap.Add(nonce2)

//确保桁条给出预期结果。自映射迭代以来
//未排序，任何一个条目都可以是第一个条目，因此请同时考虑这两个条目
//病例。
	wantStr1 := fmt.Sprintf("<%d>[%d, %d]", 2, nonce1, nonce2)
	wantStr2 := fmt.Sprintf("<%d>[%d, %d]", 2, nonce2, nonce1)
	gotStr := mruNonceMap.String()
	if gotStr != wantStr1 && gotStr != wantStr2 {
		t.Fatalf("unexpected string representation - got %q, want %q "+
			"or %q", gotStr, wantStr1, wantStr2)
	}
}

//BenchmarkmrunonceList对最近使用的
//临时处理。
func BenchmarkMruNonceList(b *testing.B) {
//创建一组用于对MRU nonce进行基准测试的假nonce
//代码。
	b.StopTimer()
	numNonces := 100000
	nonces := make([]uint64, 0, numNonces)
	for i := 0; i < numNonces; i++ {
		nonces = append(nonces, uint64(i))
	}
	b.StartTimer()

//对附加验证代码进行基准测试。
	limit := 20000
	mruNonceMap := newMruNonceMap(uint(limit))
	for i := 0; i < b.N; i++ {
		mruNonceMap.Add(nonces[i%numNonces])
	}
}
