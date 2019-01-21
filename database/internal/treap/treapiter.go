
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

import "bytes"

//迭代器表示向前和向后迭代的迭代器
//叛国罪的内容物（可变的或不变的）。
type Iterator struct {
t        *Mutable    //可变的treap迭代器与或nil关联
root     *treapNode  //treap迭代器的根节点与
node     *treapNode  //迭代器所在的节点
parents  parentStack //需要迭代的父级堆栈
isNew    bool        //是否已定位迭代器
seekKey  []byte      //Used to handle dynamic updates for mutable treap
startKey []byte      //用于将迭代器限制为一个范围
limitKey []byte      //用于将迭代器限制为一个范围
}

//limitIterator clears the current iterator node if it is outside of the range
//在创建迭代器时指定。它返回迭代器是否为
//有效。
func (iter *Iterator) limitIterator() bool {
	if iter.node == nil {
		return false
	}

	node := iter.node
	if iter.startKey != nil && bytes.Compare(node.key, iter.startKey) < 0 {
		iter.node = nil
		return false
	}

	if iter.limitKey != nil && bytes.Compare(node.key, iter.limitKey) >= 0 {
		iter.node = nil
		return false
	}

	return true
}

//Seek根据提供的键和标志移动迭代器。
//
//设置精确匹配标志时，迭代器将移动到第一个
//输入与提供的密钥完全匹配的treap，或
//前/后取决于较大的标志。
//
//如果未设置精确匹配标志，迭代器将移动到第一个
//根据较大的标志，在提供的键之前/之后输入treap。
//
//在所有情况下，创建迭代器时指定的限制是
//受人尊敬的。
func (iter *Iterator) seek(key []byte, exactMatch bool, greater bool) bool {
	iter.node = nil
	iter.parents = parentStack{}
	var selectedNodeDepth int
	for node := iter.root; node != nil; {
		iter.parents.Push(node)

//根据
//比较。另外，根据
//使迭代器在
//exact match isn't found.
		compareResult := bytes.Compare(key, node.key)
		if compareResult < 0 {
			if greater {
				iter.node = node
				selectedNodeDepth = iter.parents.Len() - 1
			}
			node = node.left
			continue
		}
		if compareResult > 0 {
			if !greater {
				iter.node = node
				selectedNodeDepth = iter.parents.Len() - 1
			}
			node = node.right
			continue
		}

//钥匙完全匹配。设置迭代器并立即返回
//设置完全匹配标志时。
		if exactMatch {
			iter.node = node
			iter.parents.Pop()
			return iter.limitIterator()
		}

//关键是精确匹配，但没有设置精确匹配，所以
//根据较大或较大的
//请求较小的密钥。
		if greater {
			node = node.right
		} else {
			node = node.left
		}
	}

//There was either no exact match or there was an exact match but the
//未设置完全匹配标志。在任何情况下，父堆栈可能
//需要进行调整，以仅包括选定的
//节点。Also, ensure the selected node's key does not exceed the
//迭代器的允许范围。
	for i := iter.parents.Len(); i > selectedNodeDepth; i-- {
		iter.parents.Pop()
	}
	return iter.limitIterator()
}

//首先将迭代器移动到第一个键/值对。当只有一个
//第一对和最后一对的单个键/值将指向同一对。
//如果没有键/值对，则返回false。
func (iter *Iterator) First() bool {
//如果迭代器是用起始键创建的，则查找起始键。本遗嘱
//导致精确匹配、第一个较大的键或
//如果不存在这样的键，则耗尽迭代器。
	iter.isNew = false
	if iter.startKey != nil {
		return iter.seek(iter.startKey, true, true)
	}

//最小的键位于最左边的节点中。
	iter.parents = parentStack{}
	for node := iter.root; node != nil; node = node.left {
		if node.left == nil {
			iter.node = node
			return true
		}
		iter.parents.Push(node)
	}
	return false
}

//Last将迭代器移动到最后一个键/值对。当只有一个
//第一对和最后一对的单个键/值将指向同一对。
//如果没有键/值对，则返回false。
func (iter *Iterator) Last() bool {
//如果迭代器是用limit键创建的，则查找limit键。本遗嘱
//导致第一个密钥小于限制密钥，或导致耗尽
//如果不存在这样的键，则使用迭代器。
	iter.isNew = false
	if iter.limitKey != nil {
		return iter.seek(iter.limitKey, false, false)
	}

//最高的键位于最右侧的节点中。
	iter.parents = parentStack{}
	for node := iter.root; node != nil; node = node.right {
		if node.right == nil {
			iter.node = node
			return true
		}
		iter.parents.Push(node)
	}
	return false
}

//next将迭代器移动到下一个键/值对，并在
//迭代器已用完。在新创建的迭代器上调用时，它将
//将迭代器定位在第一个项上。
func (iter *Iterator) Next() bool {
	if iter.isNew {
		return iter.First()
	}

	if iter.node == nil {
		return false
	}

//重新设置上一个键，但不允许在
//已请求强制查找。这将导致键大于
//前一个或已耗尽的迭代器（如果没有这样的键）。
	if seekKey := iter.seekKey; seekKey != nil {
		iter.seekKey = nil
		return iter.seek(seekKey, false, true)
	}

//当没有正确的节点时，向父节点移动，直到父节点正确为止。
//节点不等于上一个子节点。这将是下一个节点。
	if iter.node.right == nil {
		parent := iter.parents.Pop()
		for parent != nil && parent.right == iter.node {
			iter.node = parent
			parent = iter.parents.Pop()
		}
		iter.node = parent
		return iter.limitIterator()
	}

//有一个右节点，所以下一个节点是最左的向下节点
//右子树。
	iter.parents.Push(iter.node)
	iter.node = iter.node.right
	for node := iter.node.left; node != nil; node = node.left {
		iter.parents.Push(iter.node)
		iter.node = node
	}
	return iter.limitIterator()
}

//prev将迭代器移动到上一个键/值对，并在
//the iterator is exhausted.  When invoked on a newly created iterator it will
//将迭代器定位到最后一项。
func (iter *Iterator) Prev() bool {
	if iter.isNew {
		return iter.Last()
	}

	if iter.node == nil {
		return false
	}

//重新设置上一个键，但不允许在
//已请求强制查找。这会导致密钥小于
//前一个或已耗尽的迭代器（如果没有这样的键）。
	if seekKey := iter.seekKey; seekKey != nil {
		iter.seekKey = nil
		return iter.seek(seekKey, false, false)
	}

//当没有左节点时，将父节点移动到父节点的左侧
//节点不等于上一个子节点。这将是上一个
//节点。
	for iter.node.left == nil {
		parent := iter.parents.Pop()
		for parent != nil && parent.left == iter.node {
			iter.node = parent
			parent = iter.parents.Pop()
		}
		iter.node = parent
		return iter.limitIterator()
	}

//有一个左节点，所以前一个节点是最右的节点
//在左边的子树上。
	iter.parents.Push(iter.node)
	iter.node = iter.node.left
	for node := iter.node.right; node != nil; node = node.right {
		iter.parents.Push(iter.node)
		iter.node = node
	}
	return iter.limitIterator()
}

//用一个键将迭代器移动到第一个键/值对。
//大于或等于给定的键，如果成功，则返回true。
func (iter *Iterator) Seek(key []byte) bool {
	iter.isNew = false
	return iter.seek(key, true, true)
}

//键返回当前键/值对的键或当迭代器
//筋疲力尽。调用方不应修改返回的
//切片。
func (iter *Iterator) Key() []byte {
	if iter.node == nil {
		return nil
	}
	return iter.node.key
}

//值返回当前键/值对的值，或当
//迭代器已用完。调用方不应修改
//返回切片。
func (iter *Iterator) Value() []byte {
	if iter.node == nil {
		return nil
	}
	return iter.node.value
}

//valid指示迭代器是否定位在有效的键/值对上。
//当新创建或耗尽迭代器时，它将被视为无效。
func (iter *Iterator) Valid() bool {
	return iter.node != nil
}

//forcereseek通知迭代器基础可变的treap已经
//已更新，因此下一个对prev或next的调用需要重新发送以允许
//迭代器继续正常工作。
//
//注意：当迭代器与不可变的
//Treap没有你所期望的效果。
func (iter *Iterator) ForceReseek() {
//当迭代器与不可变的
//践踏。
	if iter.t == nil {
		return
	}

//将迭代器根更新为可变的treap根，以防它
//改变。
	iter.root = iter.t.root

//将SEEK键设置为当前节点。这将强制下一个/上一个
//函数重新设置，从而正确地重建迭代器
//他们的下一个电话。
	if iter.node == nil {
		iter.seekKey = nil
		return
	}
	iter.seekKey = iter.node.key
}

//迭代器返回可变treap的新迭代器。新回来的
//在调用某个方法之前，迭代器没有指向有效项
//以确定位置。
//
//start key和limit key参数导致迭代器被限制为
//一系列键。开始键是包含的，限制键是独占的。
//如果不需要该功能，则两者都可以为零。
//
//警告：如果
//the treap is mutated.  Failure to do so will cause the iterator to return
//意外的键和/或值。
//
//例如：
//iter：=t.迭代器（nil，nil）
//for iter.Next() {
//如果有什么情况
//删除（iter.key（））
//iter.forcereseek（）。
//}
//}
func (t *Mutable) Iterator(startKey, limitKey []byte) *Iterator {
	iter := &Iterator{
		t:        t,
		root:     t.root,
		isNew:    true,
		startKey: startKey,
		limitKey: limitKey,
	}
	return iter
}

//迭代器返回不可变treap的新迭代器。新回来的
//在调用某个方法之前，迭代器没有指向有效项
//以确定位置。
//
//start key和limit key参数导致迭代器被限制为
//一系列键。开始键是包含的，限制键是独占的。
//如果不需要该功能，则两者都可以为零。
func (t *Immutable) Iterator(startKey, limitKey []byte) *Iterator {
	iter := &Iterator{
		root:     t.root,
		isNew:    true,
		startKey: startKey,
		limitKey: limitKey,
	}
	return iter
}
