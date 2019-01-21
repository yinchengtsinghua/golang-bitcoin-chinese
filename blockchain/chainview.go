
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package blockchain

import (
	"sync"
)

//approxnodespewerweek是新块数量的近似值。
//平均一周。
const approxNodesPerWeek = 6 * 24 * 7

//log2floormsks定义快速计算时要使用的掩码
//常数log2（32）中的floor（log2（x））=5步，其中x是uint32，使用
//轮班。它们来自（2^（2^x）-1）*（2^（2^x）），对于4..0中的x。
var log2FloorMasks = []uint32{0xffff0000, 0xff00, 0xf0, 0xc, 0x2}

//fastlog2floor以5个常量步骤计算并返回floor（log2（x））。
func fastLog2Floor(n uint32) uint8 {
	rv := uint8(0)
	exponent := uint8(16)
	for i := 0; i < 5; i++ {
		if n&log2FloorMasks[i] != 0 {
			rv += exponent
			n >>= exponent
		}
		exponent >>= 1
	}
	return rv
}

//chain view提供块链特定分支的平面视图
//它的尖端回到Genesis区块，提供各种便利功能
//用于比较链。
//
//例如，假设有侧链的区块链如下所示：
//《创世纪》->1->2->3->4->5->6->7->8
//\->4a->5a->6a
//
//以6a结尾的分支的链视图包括：
//Genesis->1->2->3->4a->5a->6a
type chainView struct {
	mtx   sync.Mutex
	nodes []*blockNode
}

//new chain view返回给定tip block节点的新链视图。经过
//nil，因为tip将导致未初始化的链视图。小费
//可通过SETTIP功能随时更新。
func newChainView(tip *blockNode) *chainView {
//互斥体不是有意保留的，因为它是一个构造函数。
	var c chainView
	c.setTip(tip)
	return &c
}

//Genesis返回链视图的Genesis块。这和
//导出的版本，因为它取决于调用方以确保锁定
//举行。
//
//调用此函数时必须锁定视图mutex（用于读取）。
func (c *chainView) genesis() *blockNode {
	if len(c.nodes) == 0 {
		return nil
	}

	return c.nodes[0]
}

//Genesis返回链视图的Genesis块。
//
//此函数对于并发访问是安全的。
func (c *chainView) Genesis() *blockNode {
	c.mtx.Lock()
	genesis := c.genesis()
	c.mtx.Unlock()
	return genesis
}

//TIP返回链视图的当前提示块节点。它会回来
//如果没有小费就没有。这与导出版本的区别在于
//由调用者来确保锁被保持。
//
//调用此函数时必须锁定视图mutex（用于读取）。
func (c *chainView) tip() *blockNode {
	if len(c.nodes) == 0 {
		return nil
	}

	return c.nodes[len(c.nodes)-1]
}

//TIP返回链视图的当前提示块节点。它会回来
//如果没有小费就没有。
//
//此函数对于并发访问是安全的。
func (c *chainView) Tip() *blockNode {
	c.mtx.Lock()
	tip := c.tip()
	c.mtx.Unlock()
	return tip
}

//settip将链视图设置为使用提供的块节点作为当前提示
//并通过将获取的节点填充到视图中来确保视图的一致性。
//在必要的时候向后走到创世纪街区。进一步
//调用将只执行所需的最小工作，因此在链之间切换
//小费是有效率的。这与导出版本的区别在于
//直到调用方确认锁被保持。
//
//必须在视图mutex锁定的情况下调用此函数（用于写入）。
func (c *chainView) setTip(node *blockNode) {
	if node == nil {
//保留备用阵列以备将来使用。
		c.nodes = c.nodes[:0]
		return
	}

//创建或调整将块节点保持在
//提供尖端高度。创建切片时，使用
//与append一样，底层数组的一些额外容量
//以便在以后扩展链时减少开销。一样长
//由于底层数组已经有足够的容量，只需扩展或
//相应地收缩切片。选择附加容量
//这样数组就只需要扩展大约一次
//星期。
	needed := node.height + 1
	if int32(cap(c.nodes)) < needed {
		nodes := make([]*blockNode, needed, needed+approxNodesPerWeek)
		copy(nodes, c.nodes)
		c.nodes = nodes
	} else {
		prevLen := int32(len(c.nodes))
		c.nodes = c.nodes[0:needed]
		for i := prevLen; i < needed; i++ {
			c.nodes[i] = nil
		}
	}

	for node != nil && c.nodes[node.height] != node {
		c.nodes[node.height] = node
		node = node.parent
	}
}

//settip将链视图设置为使用提供的块节点作为当前提示
//并通过将获取的节点填充到视图中来确保视图的一致性。
//在必要的时候向后走到创世纪街区。进一步
//调用将只执行所需的最小工作，因此在链之间切换
//小费是有效率的。
//
//此函数对于并发访问是安全的。
func (c *chainView) SetTip(node *blockNode) {
	c.mtx.Lock()
	c.setTip(node)
	c.mtx.Unlock()
}

//Height返回链视图顶端的高度。它会返回-1如果
//没有提示（只有当链视图没有
//初始化）。这与导出版本的唯一不同之处在于
//以确保锁定被保持。
//
//调用此函数时必须锁定视图mutex（用于读取）。
func (c *chainView) height() int32 {
	return int32(len(c.nodes) - 1)
}

//Height返回链视图顶端的高度。它会返回-1如果
//没有提示（只有当链视图没有
//初始化）。
//
//此函数对于并发访问是安全的。
func (c *chainView) Height() int32 {
	c.mtx.Lock()
	height := c.height()
	c.mtx.Unlock()
	return height
}

//nodebyheight返回指定高度的块节点。将是零
//如果高度不存在，则返回。这只与导出的不同
//版本，它取决于调用方，以确保锁被持有。
//
//调用此函数时必须锁定视图mutex（用于读取）。
func (c *chainView) nodeByHeight(height int32) *blockNode {
	if height < 0 || height >= int32(len(c.nodes)) {
		return nil
	}

	return c.nodes[height]
}

//nodebyheight返回指定高度的块节点。将是零
//如果高度不存在，则返回。
//
//此函数对于并发访问是安全的。
func (c *chainView) NodeByHeight(height int32) *blockNode {
	c.mtx.Lock()
	node := c.nodeByHeight(height)
	c.mtx.Unlock()
	return node
}

//等于返回两个链视图是否相同。未初始化的
//视图（提示设置为零）被视为相等。
//
//此函数对于并发访问是安全的。
func (c *chainView) Equals(other *chainView) bool {
	c.mtx.Lock()
	other.mtx.Lock()
	equals := len(c.nodes) == len(other.nodes) && c.tip() == other.tip()
	other.mtx.Unlock()
	c.mtx.Unlock()
	return equals
}

//包含返回链视图是否包含传递的块
//节点。这与导出版本的不同之处在于
//调用方以确保锁定被保持。
//
//调用此函数时必须锁定视图mutex（用于读取）。
func (c *chainView) contains(node *blockNode) bool {
	return c.nodeByHeight(node.height) == node
}

//包含返回链视图是否包含传递的块
//节点。
//
//此函数对于并发访问是安全的。
func (c *chainView) Contains(node *blockNode) bool {
	c.mtx.Lock()
	contains := c.contains(node)
	c.mtx.Unlock()
	return contains
}

//下一步返回为链视图提供的节点的后续节点。它将
//如果没有后继节点或提供的节点不是
//查看。这与导出版本的不同之处在于
//调用方以确保锁定被保持。
//
//有关详细信息，请参见导出函数的注释。
//
//
func (c *chainView) next(node *blockNode) *blockNode {
	if node == nil || !c.contains(node) {
		return nil
	}

	return c.nodeByHeight(node.height + 1)
}

//下一步返回为链视图提供的节点的后续节点。它将
//如果没有successfor或提供的节点不是
//查看。
//
//例如，假设有侧链的区块链如下所示：
//《创世纪》->1->2->3->4->5->6->7->8
//\->4a->5a->6a
//
//此外，假设视图是针对上面描述的较长链的。那就是
//假设它包括：
//《创世纪》->1->2->3->4->5->6->7->8
//
//使用块节点5调用此函数将返回块节点6，同时
//用块节点5a调用它将返回nil，因为该节点不是部分节点
//的观点。
//
//此函数对于并发访问是安全的。
func (c *chainView) Next(node *blockNode) *blockNode {
	c.mtx.Lock()
	next := c.next(node)
	c.mtx.Unlock()
	return next
}

//findfork返回所提供节点和
//链视图。如果没有公共块，则返回零。这只有
//与导出版本不同的是，它取决于调用方以确保
//锁被锁住了。
//
//有关详细信息，请参阅导出的findfork注释。
//
//调用此函数时必须锁定视图mutex（用于读取）。
func (c *chainView) findFork(node *blockNode) *blockNode {
//不存在节点的分叉点。
	if node == nil {
		return nil
	}

//当通过的节点的高度高于
//当前链视图的尖端，向后遍历
//另一条链，直到高度匹配（或没有节点
//在这种情况下，两者之间没有共同的节点）。
//
//注意：这不是严格必要的，因为以下部分将
//同时找到节点，但是，避免
//包含检查，因为已知公共节点不能
//可能已超过当前链视图的结尾。它也允许
//此代码利用未来任何可能的优化
//祖先函数，例如使用O（log n）跳过列表。
	chainHeight := c.height()
	if node.height > chainHeight {
		node = node.Ancestor(chainHeight)
	}

//只要当前链条不向后移动，就向后移动另一条链条
//包含节点或没有更多节点，在这种情况下，没有
//两者之间的公共节点。
	for node != nil && !c.contains(node) {
		node = node.parent
	}

	return node
}

//findfork返回所提供节点和
//链视图。如果没有公共块，则返回零。
//
//例如，假设有侧链的区块链如下所示：
//《创世纪》->1->2->->5->6->7->8
//-> 6A-＞7A
//
//此外，假设视图是针对上面描述的较长链的。那就是
//假设它包括：
//《创世纪》->1->2->->5->6->7->8。
//
//使用块节点7a调用此函数将返回块节点5，同时
//用块节点7调用它将返回自身，因为它已经是
//由视图形成的分支。
//
//此函数对于并发访问是安全的。
func (c *chainView) FindFork(node *blockNode) *blockNode {
	c.mtx.Lock()
	fork := c.findFork(node)
	c.mtx.Unlock()
	return fork
}

//BlockLocator返回传递的块节点的块定位器。通过
//节点可以为零，在这种情况下，当前提示的块定位器
//将返回与该视图关联的。这与
//导出的版本是由调用者决定的，以确保锁定被保持。
//
//有关详细信息，请参见导出的blocklocator函数注释。
//
//调用此函数时必须锁定视图mutex（用于读取）。
func (c *chainView) blockLocator(node *blockNode) BlockLocator {
//如果需要，请使用当前提示。
	if node == nil {
		node = c.tip()
	}
	if node == nil {
		return nil
	}

//计算最终将在
//块定位器。请参阅算法的描述以了解这些
//数字是派生出来的。
	var maxEntries uint8
	if node.height <= 12 {
		maxEntries = uint8(node.height) + 1
	} else {
//请求哈希本身+前10个条目+Genesis块。
//然后是跳过部分的floor（log2（height-10））条目。
		adjustedHeight := uint32(node.height) - 10
		maxEntries = 12 + fastLog2Floor(adjustedHeight)
	}
	locator := make(BlockLocator, 0, maxEntries)

	step := int32(1)
	for node != nil {
		locator = append(locator, &node.hash)

//在Genesis块被添加后，就不再添加任何内容。
		if node.height == 0 {
			break
		}

//计算上一个节点的高度，包括确保
//最后一个节点是Genesis块。
		height := node.height - step
		if height < 0 {
			height = 0
		}

//当节点位于当前链视图中时，其所有
//祖先也必须如此，因此在中使用更快的o（1）查找
//那个案子。否则，返回到向后穿过
//其他链的节点指向正确的祖先。
		if c.contains(node) {
			node = c.nodes[height]
		} else {
			node = node.Ancestor(height)
		}

//一旦包含11个条目，开始将
//包含哈希之间的距离。
		if len(locator) > 10 {
			step *= 2
		}
	}

	return locator
}

//BlockLocator返回传递的块节点的块定位器。通过
//节点可以为零，在这种情况下，当前提示的块定位器
//将返回与该视图关联的。
//
//有关用于创建块的算法的详细信息，请参见块定位器类型
//定位器。
//
//此函数对于并发访问是安全的。
func (c *chainView) BlockLocator(node *blockNode) BlockLocator {
	c.mtx.Lock()
	locator := c.blockLocator(node)
	c.mtx.Unlock()
	return locator
}
