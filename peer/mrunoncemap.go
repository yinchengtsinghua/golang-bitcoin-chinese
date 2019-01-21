
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
	"bytes"
	"container/list"
	"fmt"
	"sync"
)

//mrunoncemap提供了一个并发安全映射，最大限制为
//当限制为
//超过。
type mruNonceMap struct {
	mtx       sync.Mutex
nonceMap  map[uint64]*list.Element //近o（1）次查找
nonceList *list.List               //o（1）插入、更新、删除
	limit     uint
}

//字符串将映射返回为人类可读的字符串。
//
//此函数对于并发访问是安全的。
func (m *mruNonceMap) String() string {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	lastEntryNum := len(m.nonceMap) - 1
	curEntry := 0
	buf := bytes.NewBufferString("[")
	for nonce := range m.nonceMap {
		buf.WriteString(fmt.Sprintf("%d", nonce))
		if curEntry < lastEntryNum {
			buf.WriteString(", ")
		}
		curEntry++
	}
	buf.WriteString("]")

	return fmt.Sprintf("<%d>%s", m.limit, buf.String())
}

//exists返回传递的nonce是否在映射中。
//
//此函数对于并发访问是安全的。
func (m *mruNonceMap) Exists(nonce uint64) bool {
	m.mtx.Lock()
	_, exists := m.nonceMap[nonce]
	m.mtx.Unlock()

	return exists
}

//Add adds the passed nonce to the map and handles eviction of the oldest item
//如果添加新项将超过最大限制。添加现有项
//使其成为最近使用的项目。
//
//此函数对于并发访问是安全的。
func (m *mruNonceMap) Add(nonce uint64) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

//当极限为零时，地图中不能添加任何内容，因此
//返回。
	if m.limit == 0 {
		return
	}

//当条目已经存在时，将其移到列表的前面
//从而标记出最近使用的。
	if node, exists := m.nonceMap[nonce]; exists {
		m.nonceList.MoveToFront(node)
		return
	}

//如果新的
//输入将超过地图的大小限制。同时重用列表
//节点，因此不必分配新的节点。
	if uint(len(m.nonceMap))+1 > m.limit {
		node := m.nonceList.Back()
		lru := node.Value.(uint64)

//Evict least recently used item.
		delete(m.nonceMap, lru)

//重新使用刚从中逐出的项的列表节点
//新项目。
		node.Value = nonce
		m.nonceList.MoveToFront(node)
		m.nonceMap[nonce] = node
		return
	}

//尚未达到限制，请添加新项目。
	node := m.nonceList.PushFront(nonce)
	m.nonceMap[nonce] = node
}

//Delete deletes the passed nonce from the map (if it exists).
//
//此函数对于并发访问是安全的。
func (m *mruNonceMap) Delete(nonce uint64) {
	m.mtx.Lock()
	if node, exists := m.nonceMap[nonce]; exists {
		m.nonceList.Remove(node)
		delete(m.nonceMap, nonce)
	}
	m.mtx.Unlock()
}

//newmrunoncemap返回一个仅限于
//限制指定的条目。当条目数超过限制时，
//最旧的（最近使用的）条目将被删除，以腾出空间。
//新条目。
func newMruNonceMap(limit uint) *mruNonceMap {
	m := mruNonceMap{
		nonceMap:  make(map[uint64]*list.Element),
		nonceList: list.New(),
		limit:     limit,
	}
	return &m
}
