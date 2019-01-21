
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2015 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package peer

import (
	"bytes"
	"container/list"
	"fmt"
	"sync"

	"github.com/btcsuite/btcd/wire"
)

//mruinventorymap提供了一个并发安全映射，该映射的最大值为
//当限制为
//超过。
type mruInventoryMap struct {
	invMtx  sync.Mutex
invMap  map[wire.InvVect]*list.Element //近o（1）次查找
invList *list.List                     //o（1）插入、更新、删除
	limit   uint
}

//String returns the map as a human-readable string.
//
//此函数对于并发访问是安全的。
func (m *mruInventoryMap) String() string {
	m.invMtx.Lock()
	defer m.invMtx.Unlock()

	lastEntryNum := len(m.invMap) - 1
	curEntry := 0
	buf := bytes.NewBufferString("[")
	for iv := range m.invMap {
		buf.WriteString(fmt.Sprintf("%v", iv))
		if curEntry < lastEntryNum {
			buf.WriteString(", ")
		}
		curEntry++
	}
	buf.WriteString("]")

	return fmt.Sprintf("<%d>%s", m.limit, buf.String())
}

//exists返回传递的库存项是否在映射中。
//
//此函数对于并发访问是安全的。
func (m *mruInventoryMap) Exists(iv *wire.InvVect) bool {
	m.invMtx.Lock()
	_, exists := m.invMap[*iv]
	m.invMtx.Unlock()

	return exists
}

//添加将传递的库存添加到映射并处理最旧的
//如果添加新项，则将超过最大限制。添加现有
//item makes it the most recently used item.
//
//此函数对于并发访问是安全的。
func (m *mruInventoryMap) Add(iv *wire.InvVect) {
	m.invMtx.Lock()
	defer m.invMtx.Unlock()

//当极限为零时，地图中不能添加任何内容，因此
//返回。
	if m.limit == 0 {
		return
	}

//当条目已经存在时，将其移到列表的前面
//从而标记出最近使用的。
	if node, exists := m.invMap[*iv]; exists {
		m.invList.MoveToFront(node)
		return
	}

//Evict the least recently used entry (back of the list) if the the new
//输入将超过地图的大小限制。同时重用列表
//节点，因此不必分配新的节点。
	if uint(len(m.invMap))+1 > m.limit {
		node := m.invList.Back()
		lru := node.Value.(*wire.InvVect)

//Evict least recently used item.
		delete(m.invMap, *lru)

//重新使用刚从中逐出的项的列表节点
//新项目。
		node.Value = iv
		m.invList.MoveToFront(node)
		m.invMap[*iv] = node
		return
	}

//The limit hasn't been reached yet, so just add the new item.
	node := m.invList.PushFront(iv)
	m.invMap[*iv] = node
}

//删除从映射中删除已传递的库存项（如果存在）。
//
//此函数对于并发访问是安全的。
func (m *mruInventoryMap) Delete(iv *wire.InvVect) {
	m.invMtx.Lock()
	if node, exists := m.invMap[*iv]; exists {
		m.invList.Remove(node)
		delete(m.invMap, *iv)
	}
	m.invMtx.Unlock()
}

//newmruinventorymap返回一个限制为数字的新库存映射
//限制指定的条目。当条目数超过限制时，
//最旧的（最近使用的）条目将被删除，以腾出空间。
//新条目。
func newMruInventoryMap(limit uint) *mruInventoryMap {
	m := mruInventoryMap{
		invMap:  make(map[wire.InvVect]*list.Element),
		invList: list.New(),
		limit:   limit,
	}
	return &m
}
