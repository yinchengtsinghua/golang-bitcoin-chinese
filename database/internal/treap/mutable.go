
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
	"math/rand"
)

//mutable表示用于保持有序的treap数据结构
//使用二进制搜索树和堆语义组合的键/值对。
//它是一种自组织随机数据结构，不需要
//维持平衡的复杂操作。搜索、插入和删除
//操作都是O（log n）。
type Mutable struct {
	root  *treapNode
	count int

//ToeStand是所有数据的总大小的最佳估计值。
//treap包括键、值和节点大小。
	totalSize uint64
}

//len返回存储在treap中的项目数。
func (t *Mutable) Len() int {
	return t.count
}

//SIZE返回treap的总字节数的最佳估计值。
//使用包括用于表示节点的所有字段以及
//键和值的大小。未检测到共享值，因此
//返回的大小假定每个值指向不同的内存。
func (t *Mutable) Size() uint64 {
	return t.totalSize
}

//get返回包含传递的键及其父级的treap节点。什么时候？
//找到的节点是树的根，父节点将为零。当钥匙
//不存在，节点和父级都将为零。
func (t *Mutable) get(key []byte) (*treapNode, *treapNode) {
	var parent *treapNode
	for node := t.root; node != nil; {
//根据
//比较。
		compareResult := bytes.Compare(key, node.key)
		if compareResult < 0 {
			parent = node
			node = node.left
			continue
		}
		if compareResult > 0 {
			parent = node
			node = node.right
			continue
		}

//密钥存在。
		return node, parent
	}

//达到了nil节点，这意味着该键不存在。
	return nil, nil
}

//has返回传递的键是否存在。
func (t *Mutable) Has(key []byte) bool {
	if node, _ := t.get(key); node != nil {
		return true
	}
	return false
}

//get返回传递的键的值。当
//密钥不存在。
func (t *Mutable) Get(key []byte) []byte {
	if node, _ := t.get(key); node != nil {
		return node.value
	}
	return nil
}

//relinkgrandplast在节点旋转后将其重新链接到treap中
//根据
//旧父节点所在的位置，指向已传递的节点。否则，在那里
//不是祖父母，这意味着节点现在是树的根，所以更新
//因此。
func (t *Mutable) relinkGrandparent(node, parent, grandparent *treapNode) {
//当没有祖父母时，节点现在是树的根。
	if grandparent == nil {
		t.root = node
		return
	}

//根据哪一侧重新链接祖父母的左指针或右指针
//the old parent was.
	if grandparent.left == parent {
		grandparent.left = node
	} else {
		grandparent.right = node
	}
}

//PUT插入传递的键/值对。
func (t *Mutable) Put(key, value []byte) {
//当没有提供值时，请为该值使用空字节片。这个
//最终允许从值以来确定密钥存在
//空字节片可以与零区分开。
	if value == nil {
		value = emptySlice
	}

//节点是树的根（如果还没有根的话）。
	if t.root == nil {
		node := newTreapNode(key, value, rand.Int())
		t.count = 1
		t.totalSize = nodeSize(node)
		t.root = node
		return
	}

//找到二叉树插入点并构造父级列表
//同时。当密钥与Treap中已有的条目匹配时，
//只需更新它的值并返回。
	var parents parentStack
	var compareResult int
	for node := t.root; node != nil; {
		parents.Push(node)
		compareResult = bytes.Compare(key, node.key)
		if compareResult < 0 {
			node = node.left
			continue
		}
		if compareResult > 0 {
			node = node.right
			continue
		}

//该键已存在，因此请更新其值。
		t.totalSize -= uint64(len(node.value))
		t.totalSize += uint64(len(value))
		node.value = value
		return
	}

//在正确的位置将新节点链接到二进制树。
	node := newTreapNode(key, value, rand.Int())
	t.count++
	t.totalSize += nodeSize(node)
	parent := parents.At(0)
	if compareResult < 0 {
		parent.left = node
	} else {
		parent.right = node
	}

//执行维护最小堆所需的任何旋转。
	for parents.Len() > 0 {
//当节点的优先级为
//大于或等于其父级的优先级。
		parent = parents.Pop()
		if node.priority >= parent.priority {
			break
		}

//如果节点位于左侧或
//如果节点位于右侧，则为左旋转。
		if parent.left == node {
			node.right, parent.left = parent, node.right
		} else {
			node.left, parent.right = parent, node.left
		}
		t.relinkGrandparent(node, parent, parents.At(0))
	}
}

//删除删除传递的密钥（如果存在）。
func (t *Mutable) Delete(key []byte) {
//查找键的节点及其父节点。没有什么可以
//如果键不存在，则执行此操作。
	node, parent := t.get(key)
	if node == nil {
		return
	}

//当树中唯一的节点是根节点并且是根节点时
//如果被删除，除了删除它没有其他事情可做。
	if parent == nil && node.left == nil && node.right == nil {
		t.root = nil
		t.count = 0
		t.totalSize = 0
		return
	}

//执行旋转以将要删除的节点移动到叶位置，同时
//维护最小堆。
	var isLeft bool
	var child *treapNode
	for node.left != nil || node.right != nil {
//选择优先级较高的孩子。
		if node.left == nil {
			child = node.right
			isLeft = false
		} else if node.right == nil {
			child = node.left
			isLeft = true
		} else if node.left.priority >= node.right.priority {
			child = node.left
			isLeft = true
		} else {
			child = node.right
			isLeft = false
		}

//根据子节点的一侧向左或向右旋转
//开始了。This has the effect of moving the node to delete
//朝向树的底部，同时保持
//最小堆。
		if isLeft {
			child.right, node.left = node, child.right
		} else {
			child.left, node.right = node, child.left
		}
		t.relinkGrandparent(child, node, parent)

//要删除的节点的父级现在是以前的
//它的孩子。
		parent = child
	}

//通过断开节点与的连接，删除该节点，该节点现在是叶节点。
//它的父母。
	if parent.right == node {
		parent.right = nil
	} else {
		parent.left = nil
	}
	t.count--
	t.totalSize -= nodeSize(node)
}

//foreach使用treap中的每个键/值对调用传递的函数
//按升序排列。
func (t *Mutable) ForEach(fn func(k, v []byte) bool) {
//将根节点及其左侧的所有子节点添加到
//要遍历和循环的节点，直到它们及其所有子节点，
//已被遍历。
	var parents parentStack
	for node := t.root; node != nil; node = node.left {
		parents.Push(node)
	}
	for parents.Len() > 0 {
		node := parents.Pop()
		if !fn(node.key, node.value) {
			return
		}

//将要由所有子级遍历的节点扩展到
//当前节点的右子级。
		for node := node.right; node != nil; node = node.left {
			parents.Push(node)
		}
	}
}

//重置有效地删除treap中的所有项目。
func (t *Mutable) Reset() {
	t.count = 0
	t.totalSize = 0
	t.root = nil
}

//new mutable返回新的空mutable treap，可以使用。见
//有关可变结构的详细信息，请参阅文档。
func NewMutable() *Mutable {
	return &Mutable{}
}
