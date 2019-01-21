
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

//clonetreapnode返回传递的节点的浅副本。
func cloneTreapNode(node *treapNode) *treapNode {
	return &treapNode{
		key:      node.key,
		value:    node.value,
		priority: node.priority,
		left:     node.left,
		right:    node.right,
	}
}

//Immutable represents a treap data structure which is used to hold ordered
//使用二进制搜索树和堆语义组合的键/值对。
//它是一种自组织随机数据结构，不需要
//维持平衡的复杂操作。搜索、插入和删除
//操作都是O（log n）。此外，它还为
//多版本并发控制（MVCC）。
//
//所有导致修改treap的操作都会返回新版本的
//只有修改后的节点才能更新的treap。所有未修改的节点都是
//与以前的版本共享。这在并发时非常有用
//applications since the caller only has to atomically replace the treap
//执行任何突变后，带有新返回版本的指针。所有
//读卡器可以简单地将其现有指针用作快照，因为
//它指向的是不可变的。这有效地提供了O（1）快照
//自旧节点以来具有高效内存使用特性的功能
//只有在不再有任何对它们的引用之前，才保持分配状态。
type Immutable struct {
	root  *treapNode
	count int

//ToeStand是所有数据的总大小的最佳估计值。
//treap包括键、值和节点大小。
	totalSize uint64
}

//newImmutable返回给定传递参数的新的不可变treap。
func newImmutable(root *treapNode, count int, totalSize uint64) *Immutable {
	return &Immutable{root: root, count: count, totalSize: totalSize}
}

//len返回存储在treap中的项目数。
func (t *Immutable) Len() int {
	return t.count
}

//SIZE返回treap的总字节数的最佳估计值。
//consuming including all of the fields used to represent the nodes as well as
//键和值的大小。未检测到共享值，因此
//返回的大小假定每个值指向不同的内存。
func (t *Immutable) Size() uint64 {
	return t.totalSize
}

//get返回包含传递的键的treap节点。它将返回零
//当密钥不存在时。
func (t *Immutable) get(key []byte) *treapNode {
	for node := t.root; node != nil; {
//根据
//比较。
		compareResult := bytes.Compare(key, node.key)
		if compareResult < 0 {
			node = node.left
			continue
		}
		if compareResult > 0 {
			node = node.right
			continue
		}

//密钥存在。
		return node
	}

//达到了nil节点，这意味着该键不存在。
	return nil
}

//has返回传递的键是否存在。
func (t *Immutable) Has(key []byte) bool {
	if node := t.get(key); node != nil {
		return true
	}
	return false
}

//get返回传递的键的值。当
//密钥不存在。
func (t *Immutable) Get(key []byte) []byte {
	if node := t.get(key); node != nil {
		return node.value
	}
	return nil
}

//PUT插入传递的键/值对。
func (t *Immutable) Put(key, value []byte) *Immutable {
//当没有提供值时，请为该值使用空字节片。这个
//最终允许从值以来确定密钥存在
//空字节片可以与零区分开。
	if value == nil {
		value = emptySlice
	}

//节点是树的根（如果还没有根的话）。
	if t.root == nil {
		root := newTreapNode(key, value, rand.Int())
		return newImmutable(root, 1, nodeSize(root))
	}

//找到二叉树插入点并构造
//父母同时这么做。这样做是因为这是不可变的
//数据结构，因此无论在treap中的何处
//配对结束后，所有祖先到并包括根都需要
//替换。
//
//When the key matches an entry already in the treap, replace the node
//用一个新的值设置并返回。
	var parents parentStack
	var compareResult int
	for node := t.root; node != nil; {
//克隆节点并在需要时将其父节点链接到该节点。
		nodeCopy := cloneTreapNode(node)
		if oldParent := parents.At(0); oldParent != nil {
			if oldParent.left == node {
				oldParent.left = nodeCopy
			} else {
				oldParent.right = nodeCopy
			}
		}
		parents.Push(nodeCopy)

//根据比较结果向左或向右移动
//钥匙。
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
		nodeCopy.value = value

//Return new immutable treap with the replaced node and
//包括树根在内的祖先。
		newRoot := parents.At(parents.Len() - 1)
		newTotalSize := t.totalSize - uint64(len(node.value)) +
			uint64(len(value))
		return newImmutable(newRoot, t.count, newTotalSize)
	}

//在正确的位置将新节点链接到二进制树。
	node := newTreapNode(key, value, rand.Int())
	parent := parents.At(0)
	if compareResult < 0 {
		parent.left = node
	} else {
		parent.right = node
	}

//执行维护最小堆和替换所需的任何旋转
//包括树根在内的祖先。
	newRoot := parents.At(parents.Len() - 1)
	for parents.Len() > 0 {
//当节点的优先级为
//greater than or equal to its parent's priority.
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

//如果没有
//祖父母或将祖父母重新链接到节点基于
//节点要替换的旧父级位于哪一侧。
		grandparent := parents.At(0)
		if grandparent == nil {
			newRoot = node
		} else if grandparent.left == parent {
			grandparent.left = node
		} else {
			grandparent.right = node
		}
	}

	return newImmutable(newRoot, t.count+1, t.totalSize+nodeSize(node))
}

//delete从treap中删除传递的键并返回结果treap
//如果它存在。如果密钥不存在，则返回原来的不可变的TRAP。
//存在。
func (t *Immutable) Delete(key []byte) *Immutable {
//在构造父项列表时查找键的节点，同时
//这样做。
	var parents parentStack
	var delNode *treapNode
	for node := t.root; node != nil; {
		parents.Push(node)

//根据
//比较。
		compareResult := bytes.Compare(key, node.key)
		if compareResult < 0 {
			node = node.left
			continue
		}
		if compareResult > 0 {
			node = node.right
			continue
		}

//密钥存在。
		delNode = node
		break
	}

//There is nothing to do if the key does not exist.
	if delNode == nil {
		return t
	}

//当树中唯一的节点是根节点并且是根节点时
//如果被删除，除了删除它没有其他事情可做。
	parent := parents.At(1)
	if parent == nil && delNode.left == nil && delNode.right == nil {
		return newImmutable(nil, 0, 0)
	}

//构造替换的父节点列表和要删除其自身的节点。
//这样做是因为这是一个不可变的数据结构，并且
//therefore all ancestors of the node that will be deleted, up to and
//包括根部，需要更换。
	var newParents parentStack
	for i := parents.Len(); i > 0; i-- {
		node := parents.At(i - 1)
		nodeCopy := cloneTreapNode(node)
		if oldParent := newParents.At(0); oldParent != nil {
			if oldParent.left == node {
				oldParent.left = nodeCopy
			} else {
				oldParent.right = nodeCopy
			}
		}
		newParents.Push(nodeCopy)
	}
	delNode = newParents.Pop()
	parent = newParents.At(0)

//执行旋转以将要删除的节点移动到叶位置，同时
//在替换修改的子级时维护最小堆。
	var child *treapNode
	newRoot := newParents.At(newParents.Len() - 1)
	for delNode.left != nil || delNode.right != nil {
//选择优先级较高的孩子。
		var isLeft bool
		if delNode.left == nil {
			child = delNode.right
		} else if delNode.right == nil {
			child = delNode.left
			isLeft = true
		} else if delNode.left.priority >= delNode.right.priority {
			child = delNode.left
			isLeft = true
		} else {
			child = delNode.right
		}

//根据子节点的一侧向左或向右旋转
//开始了。This has the effect of moving the node to delete
//朝向树的底部，同时保持
//最小堆。
		child = cloneTreapNode(child)
		if isLeft {
			child.right, delNode.left = delNode, child.right
		} else {
			child.left, delNode.right = delNode, child.left
		}

//如果没有
//祖父母或将祖父母重新链接到节点基于
//节点要替换的旧父级位于哪一侧。
//
//由于要删除的节点刚向下移动了一个级别，因此
//新祖父母现在是现任父母和新父母
//是当前子级。
		if parent == nil {
			newRoot = child
		} else if parent.left == delNode {
			parent.left = child
		} else {
			parent.right = child
		}

//要删除的节点的父级现在是以前的
//它的孩子。
		parent = child
	}

//通过断开节点与的连接，删除该节点，该节点现在是叶节点。
//它的父母。
	if parent.right == delNode {
		parent.right = nil
	} else {
		parent.left = nil
	}

	return newImmutable(newRoot, t.count-1, t.totalSize-nodeSize(delNode))
}

//foreach使用treap中的每个键/值对调用传递的函数
//按升序排列。
func (t *Immutable) ForEach(fn func(k, v []byte) bool) {
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

//newImmutable返回一个新的空的、不可变的treap，可以使用。见
//有关不可变结构的详细信息，请参阅文档。
func NewImmutable() *Immutable {
	return &Immutable{}
}
