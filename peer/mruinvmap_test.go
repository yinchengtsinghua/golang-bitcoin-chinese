
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package peer

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

//testmruinventorymap确保mruinventorymap的行为符合预期，包括
//限制、驱逐最近使用过的条目、删除特定条目，
//以及存在测试。
func TestMruInventoryMap(t *testing.T) {
//创建一组假库存向量用于测试MRU
//库存代码。
	numInvVects := 10
	invVects := make([]*wire.InvVect, 0, numInvVects)
	for i := 0; i < numInvVects; i++ {
		hash := &chainhash.Hash{byte(i)}
		iv := wire.NewInvVect(wire.InvTypeBlock, hash)
		invVects = append(invVects, iv)
	}

	tests := []struct {
		name  string
		limit int
	}{
		{name: "limit 0", limit: 0},
		{name: "limit 1", limit: 1},
		{name: "limit 5", limit: 5},
		{name: "limit 7", limit: 7},
		{name: "limit one less than available", limit: numInvVects - 1},
		{name: "limit all available", limit: numInvVects},
	}

testLoop:
	for i, test := range tests {
//创建受指定测试限制的新MRU库存映射
//限制并添加所有测试库存向量。本遗嘱
//因为有更多的测试库存向量
//超过极限。
		mruInvMap := newMruInventoryMap(uint(test.limit))
		for j := 0; j < numInvVects; j++ {
			mruInvMap.Add(invVects[j])
		}

//确保在
//存在库存矢量列表。
		for j := numInvVects - test.limit; j < numInvVects; j++ {
			if !mruInvMap.Exists(invVects[j]) {
				t.Errorf("Exists #%d (%s) entry %s does not "+
					"exist", i, test.name, *invVects[j])
				continue testLoop
			}
		}

//确保在最新的有限数量之前输入
//entries in the inventory vector list do not exist.
		for j := 0; j < numInvVects-test.limit; j++ {
			if mruInvMap.Exists(invVects[j]) {
				t.Errorf("Exists #%d (%s) entry %s exists", i,
					test.name, *invVects[j])
				continue testLoop
			}
		}

//Readd the entry that should currently be the least-recently
//所以它成为最近使用的条目，然后
//通过添加不存在的条目和
//确保收回的条目是最近使用最少的新条目
//条目。
//
//此检查至少需要2个条目。
		if test.limit > 1 {
			origLruIndex := numInvVects - test.limit
			mruInvMap.Add(invVects[origLruIndex])

			iv := wire.NewInvVect(wire.InvTypeBlock,
				&chainhash.Hash{0x00, 0x01})
			mruInvMap.Add(iv)

//确保原始LRU条目仍然存在，因为它
//已更新，应该已成为MRU条目。
			if !mruInvMap.Exists(invVects[origLruIndex]) {
				t.Errorf("MRU #%d (%s) entry %s does not exist",
					i, test.name, *invVects[origLruIndex])
				continue testLoop
			}

//确保本应成为新LRU的条目
//条目被逐出。
			newLruIndex := origLruIndex + 1
			if mruInvMap.Exists(invVects[newLruIndex]) {
				t.Errorf("MRU #%d (%s) entry %s exists", i,
					test.name, *invVects[newLruIndex])
				continue testLoop
			}
		}

//删除库存向量列表中的所有条目，
//包括那些地图上不存在的，并确保它们
//不再存在。
		for j := 0; j < numInvVects; j++ {
			mruInvMap.Delete(invVects[j])
			if mruInvMap.Exists(invVects[j]) {
				t.Errorf("Delete #%d (%s) entry %s exists", i,
					test.name, *invVects[j])
				continue testLoop
			}
		}
	}
}

//testmruinventoryMapStringer测试
//mruinventorymap类型。
func TestMruInventoryMapStringer(t *testing.T) {
//创建几个假库存向量用于测试MRU
//库存字符串代码。
	hash1 := &chainhash.Hash{0x01}
	hash2 := &chainhash.Hash{0x02}
	iv1 := wire.NewInvVect(wire.InvTypeBlock, hash1)
	iv2 := wire.NewInvVect(wire.InvTypeBlock, hash2)

//创建新的MRU库存地图并添加库存向量。
	mruInvMap := newMruInventoryMap(uint(2))
	mruInvMap.Add(iv1)
	mruInvMap.Add(iv2)

//确保桁条给出预期结果。自映射迭代以来
//未排序，任何一个条目都可以是第一个条目，因此请同时考虑这两个条目
//病例。
	wantStr1 := fmt.Sprintf("<%d>[%s, %s]", 2, *iv1, *iv2)
	wantStr2 := fmt.Sprintf("<%d>[%s, %s]", 2, *iv2, *iv1)
	gotStr := mruInvMap.String()
	if gotStr != wantStr1 && gotStr != wantStr2 {
		t.Fatalf("unexpected string representation - got %q, want %q "+
			"or %q", gotStr, wantStr1, wantStr2)
	}
}

//Benchmarkmruinventorylist对最近的
//已使用的库存处理。
func BenchmarkMruInventoryList(b *testing.B) {
//创建一组用于基准测试的假库存向量
//the mru inventory code.
	b.StopTimer()
	numInvVects := 100000
	invVects := make([]*wire.InvVect, 0, numInvVects)
	for i := 0; i < numInvVects; i++ {
		hashBytes := make([]byte, chainhash.HashSize)
		rand.Read(hashBytes)
		hash, _ := chainhash.NewHash(hashBytes)
		iv := wire.NewInvVect(wire.InvTypeBlock, hash)
		invVects = append(invVects, iv)
	}
	b.StartTimer()

//对附加验证代码进行基准测试。
	limit := 20000
	mruInvMap := newMruInventoryMap(uint(limit))
	for i := 0; i < b.N; i++ {
		mruInvMap.Add(invVects[i%numInvVects])
	}
}
