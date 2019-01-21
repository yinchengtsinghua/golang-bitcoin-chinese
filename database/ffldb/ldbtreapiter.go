
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

package ffldb

import (
	"github.com/btcsuite/btcd/database/internal/treap"
	"github.com/btcsuite/goleveldb/leveldb/iterator"
	"github.com/btcsuite/goleveldb/leveldb/util"
)

//ldbtreapiter包装一个treap迭代器以提供附加功能
//需要满足leveldb iterator.iterator接口。
type ldbTreapIter struct {
	*treap.Iterator
	tx       *transaction
	released bool
}

//强制ldbtreapiter实现leveldb iterator.iterator接口。
var _ iterator.Iterator = (*ldbTreapIter)(nil)

//提供的错误仅用于满足迭代器接口，因为没有
//仅此内存结构的错误。
//
//这是leveldb iterator.iterator接口实现的一部分。
func (iter *ldbTreapIter) Error() error {
	return nil
}

//提供setreleaser只是为了满足迭代器接口，因为没有
//需要覆盖它。
//
//这是leveldb iterator.iterator接口实现的一部分。
func (iter *ldbTreapIter) SetReleaser(releaser util.Releaser) {
}

//release通过从中移除基础的treap迭代器来释放迭代器
//针对挂起的键treap的活动迭代器列表。
//
//这是leveldb iterator.iterator接口实现的一部分。
func (iter *ldbTreapIter) Release() {
	if !iter.released {
		iter.tx.removeActiveIter(iter.Iterator)
		iter.released = true
	}
}

//newldbtreapiter针对
//已传递事务的挂起键，并将其包装在
//因此它可以用作LEVELDB迭代器。它还增加了新的
//迭代器到事务的活动迭代器列表。
func newLdbTreapIter(tx *transaction, slice *util.Range) *ldbTreapIter {
	iter := tx.pendingKeys.Iterator(slice.Start, slice.Limit)
	tx.addActiveIter(iter)
	return &ldbTreapIter{Iterator: iter, tx: tx}
}
