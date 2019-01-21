
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
	"math/rand"
	"time"
)

const (
//StaticDepth是用于跟踪的静态数组的大小
//在treap迭代期间的父堆栈。因为叛国者
//很可能树的高度是对数的，它是
//父堆栈极不可能超过此大小
//即使是对于大量的项目。
	staticDepth = 128

//nodefieldsize是每个节点字段的大小，不包括
//键和值的内容。它假定64位指针，所以
//从技术上讲，它在32位平台上较小，但高估了
//在这种情况下，尺寸是可以接受的，因为它避免了导入
//不安全的。它由每个键24个字节和值+8个字节组成
//每个优先级、左、右字段（24*2+8*3）。
	nodeFieldsSize = 72
)

var (
//EmptySlice用于没有关联值的键
//因此调用方可以区分不存在的键和一个键
//它没有关联的值。
	emptySlice = make([]byte, 0)
)

//treap node表示treap中的一个节点。
type treapNode struct {
	key      []byte
	value    []byte
	priority int
	left     *treapNode
	right    *treapNode
}

//nodesize返回指定节点占用的字节数，包括
//结构字段以及键和值的内容。
func nodeSize(node *treapNode) uint64 {
	return nodeFieldsSize + uint64(len(node.key)+len(node.value))
}

//newtreapnode从给定的键、值和优先级返回一个新节点。这个
//节点最初未链接到任何其他节点。
func newTreapNode(key, value []byte, priority int) *treapNode {
	return &treapNode{key: key, value: value, priority: priority}
}

//ParentStack表示在
//迭代。它由一个静态数组组成，用于存放父级和
//动态溢出切片。极不可能发生溢流
//然而，在正常操作中，由于TRAP的高度是
//概率上，溢出的情况需要妥善处理。这种方法
//因为大多数情况下它比
//每次迭代treap时动态分配堆空间。
type parentStack struct {
	index    int
	items    [staticDepth]*treapNode
	overflow []*treapNode
}

//len返回堆栈中的当前项数。
func (s *parentStack) Len() int {
	return s.index
}

//at返回堆栈顶部的项n个数，其中0是
//最上面的项目，不移除它。如果n超过
//堆栈上的项数。
func (s *parentStack) At(n int) *treapNode {
	index := s.index - n - 1
	if index < 0 {
		return nil
	}

	if index < staticDepth {
		return s.items[index]
	}

	return s.overflow[index-staticDepth]
}

//pop从堆栈中删除最上面的项。如果堆栈是
//空的。
func (s *parentStack) Pop() *treapNode {
	if s.index == 0 {
		return nil
	}

	s.index--
	if s.index < staticDepth {
		node := s.items[s.index]
		s.items[s.index] = nil
		return node
	}

	node := s.overflow[s.index-staticDepth]
	s.overflow[s.index-staticDepth] = nil
	return node
}

//push将传递的项推送到堆栈顶部。
func (s *parentStack) Push(node *treapNode) {
	if s.index < staticDepth {
		s.items[s.index] = node
		s.index++
		return
	}

//此方法用于追加，因为将切片重新折叠到pop
//the item causes the compiler to make unneeded allocations.  也，
//因为最大项目数与树的深度有关，
//需要明确增加更多项目，只增加上限
//one item at a time.  This is more intelligent than the generic append
//通常使cap加倍的展开算法。
	index := s.index - staticDepth
	if index+1 > cap(s.overflow) {
		overflow := make([]*treapNode, index+1)
		copy(overflow, s.overflow)
		s.overflow = overflow
	}
	s.overflow[index] = node
	s.index++
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
