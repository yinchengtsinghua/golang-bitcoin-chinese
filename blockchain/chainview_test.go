
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
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"github.com/btcsuite/btcd/wire"
)

//testnoncepring为生成的伪代码中的nonce提供了一个确定性prng
//节点。确保节点具有唯一的哈希。
var testNoncePrng = rand.New(rand.NewSource(0))

//chainedNodes返回指定数量的节点，这些节点构造为
//后续节点指向上一个节点以创建链。第一节点
//将指向传递的父级，如果需要，可以为零。
func chainedNodes(parent *blockNode, numNodes int) []*blockNode {
	nodes := make([]*blockNode, numNodes)
	tip := parent
	for i := 0; i < numNodes; i++ {
//这是无效的，但所需的一切足以使
//综合测试有效。
		header := wire.BlockHeader{Nonce: testNoncePrng.Uint32()}
		if tip != nil {
			header.PrevBlock = tip.hash
		}
		nodes[i] = newBlockNode(&header, tip)
		tip = nodes[i]
	}
	return nodes
}

//字符串将块节点返回为人类可读的名称。
func (node blockNode) String() string {
	return fmt.Sprintf("%s(%d)", node.hash, node.height)
}

//tsttip是一个方便的函数，用于获取块节点链的尖端。
//通过链节点创建。
func tstTip(nodes []*blockNode) *blockNode {
	return nodes[len(nodes)-1]
}

//locatorhashes是一个方便函数，它返回所有
//所提供节点的已传递索引。它用于构造预期的
//阻止测试中的定位器。
func locatorHashes(nodes []*blockNode, indexes ...int) BlockLocator {
	hashes := make(BlockLocator, 0, len(indexes))
	for _, idx := range indexes {
		hashes = append(hashes, &nodes[idx].hash)
	}
	return hashes
}

//ziplocators是一个返回单个块定位器的方便函数
//给定一个变量数，并在测试中使用。
func zipLocators(locators ...BlockLocator) BlockLocator {
	var hashes BlockLocator
	for _, locator := range locators {
		hashes = append(hashes, locator...)
	}
	return hashes
}

//测试链视图确保链视图的所有导出功能正常工作。
//除某些特殊情况外
//其他测试。
func TestChainView(t *testing.T) {
//构造一个由以下内容组成的综合块索引
//结构。
//0->1->2->3->4
//\->2a->3a->4a->5a->6a->7a->-> 26A
//\->3a'->4a'->5a'
	branch0Nodes := chainedNodes(nil, 5)
	branch1Nodes := chainedNodes(branch0Nodes[1], 25)
	branch2Nodes := chainedNodes(branch1Nodes[0], 3)

	tip := tstTip
	tests := []struct {
		name       string
view       *chainView   //活动视图
genesis    *blockNode   //活动视图的预期创世块
tip        *blockNode   //活动视图的预期提示
side       *chainView   //侧链视图
sideTip    *blockNode   //侧链视图的预期尖端
fork       *blockNode   //应为分叉节点
contains   []*blockNode //活动视图中的预期节点
noContains []*blockNode //预期节点不在活动视图中
equal      *chainView   //视图应等于活动视图
unequal    *chainView   //视图应不等于活动
locator    BlockLocator //活动视图提示的预期定位器
	}{
		{
//将分支0的视图创建为活动链，并
//分支1作为侧链的另一个视图。
			name:       "chain0-chain1",
			view:       newChainView(tip(branch0Nodes)),
			genesis:    branch0Nodes[0],
			tip:        tip(branch0Nodes),
			side:       newChainView(tip(branch1Nodes)),
			sideTip:    tip(branch1Nodes),
			fork:       branch0Nodes[1],
			contains:   branch0Nodes,
			noContains: branch1Nodes,
			equal:      newChainView(tip(branch0Nodes)),
			unequal:    newChainView(tip(branch1Nodes)),
			locator:    locatorHashes(branch0Nodes, 4, 3, 2, 1, 0),
		},
		{
//将分支1的视图创建为活动链，并
//分支2作为侧链的另一个视图。
			name:       "chain1-chain2",
			view:       newChainView(tip(branch1Nodes)),
			genesis:    branch0Nodes[0],
			tip:        tip(branch1Nodes),
			side:       newChainView(tip(branch2Nodes)),
			sideTip:    tip(branch2Nodes),
			fork:       branch1Nodes[0],
			contains:   branch1Nodes,
			noContains: branch2Nodes,
			equal:      newChainView(tip(branch1Nodes)),
			unequal:    newChainView(tip(branch2Nodes)),
			locator: zipLocators(
				locatorHashes(branch1Nodes, 24, 23, 22, 21, 20,
					19, 18, 17, 16, 15, 14, 13, 11, 7),
				locatorHashes(branch0Nodes, 1, 0)),
		},
		{
//将分支2的视图创建为活动链，并
//分支0作为侧链的另一个视图。
			name:       "chain2-chain0",
			view:       newChainView(tip(branch2Nodes)),
			genesis:    branch0Nodes[0],
			tip:        tip(branch2Nodes),
			side:       newChainView(tip(branch0Nodes)),
			sideTip:    tip(branch0Nodes),
			fork:       branch0Nodes[1],
			contains:   branch2Nodes,
			noContains: branch0Nodes[2:],
			equal:      newChainView(tip(branch2Nodes)),
			unequal:    newChainView(tip(branch0Nodes)),
			locator: zipLocators(
				locatorHashes(branch2Nodes, 2, 1, 0),
				locatorHashes(branch1Nodes, 0),
				locatorHashes(branch0Nodes, 1, 0)),
		},
	}
testLoop:
	for _, test := range tests {
//确保主动链和侧链高度符合预期。
//价值观。
		if test.view.Height() != test.tip.height {
			t.Errorf("%s: unexpected active view height -- got "+
				"%d, want %d", test.name, test.view.Height(),
				test.tip.height)
			continue
		}
		if test.side.Height() != test.sideTip.height {
			t.Errorf("%s: unexpected side view height -- got %d, "+
				"want %d", test.name, test.side.Height(),
				test.sideTip.height)
			continue
		}

//确保主动和侧链Genesis区块
//期望值。
		if test.view.Genesis() != test.genesis {
			t.Errorf("%s: unexpected active view genesis -- got "+
				"%v, want %v", test.name, test.view.Genesis(),
				test.genesis)
			continue
		}
		if test.side.Genesis() != test.genesis {
			t.Errorf("%s: unexpected side view genesis -- got %v, "+
				"want %v", test.name, test.view.Genesis(),
				test.genesis)
			continue
		}

//确保活动和侧链尖端是预期的节点。
		if test.view.Tip() != test.tip {
			t.Errorf("%s: unexpected active view tip -- got %v, "+
				"want %v", test.name, test.view.Tip(), test.tip)
			continue
		}
		if test.side.Tip() != test.sideTip {
			t.Errorf("%s: unexpected active view tip -- got %v, "+
				"want %v", test.name, test.side.Tip(),
				test.sideTip)
			continue
		}

//无论这两条链条的顺序如何，
//相比之下，它们都返回了预期的分叉点。
		forkNode := test.view.FindFork(test.side.Tip())
		if forkNode != test.fork {
			t.Errorf("%s: unexpected fork node (view, side) -- "+
				"got %v, want %v", test.name, forkNode,
				test.fork)
			continue
		}
		forkNode = test.side.FindFork(test.view.Tip())
		if forkNode != test.fork {
			t.Errorf("%s: unexpected fork node (side, view) -- "+
				"got %v, want %v", test.name, forkNode,
				test.fork)
			continue
		}

//确保已经是一部分的节点的分叉点
//链视图的对象是节点本身。
		forkNode = test.view.FindFork(test.view.Tip())
		if forkNode != test.view.Tip() {
			t.Errorf("%s: unexpected fork node (view, tip) -- "+
				"got %v, want %v", test.name, forkNode,
				test.view.Tip())
			continue
		}

//确保所有预期节点都包含在活动视图中。
		for _, node := range test.contains {
			if !test.view.Contains(node) {
				t.Errorf("%s: expected %v in active view",
					test.name, node)
				continue testLoop
			}
		}

//确保侧链视图中的所有节点不包含在
//活动视图。
		for _, node := range test.noContains {
			if test.view.Contains(node) {
				t.Errorf("%s: unexpected %v in active view",
					test.name, node)
				continue testLoop
			}
		}

//在同一个连锁工程中确保不同观点的平等性
//如预期的那样。
		if !test.view.Equals(test.equal) {
			t.Errorf("%s: unexpected unequal views", test.name)
			continue
		}
		if test.view.Equals(test.unequal) {
			t.Errorf("%s: unexpected equal views", test.name)
			continue
		}

//确保视图中包含的所有节点返回预期的
//下一个节点。
		for i, node := range test.contains {
//最后一个节点要求下一个节点为零。
			var expected *blockNode
			if i < len(test.contains)-1 {
				expected = test.contains[i+1]
			}
			if next := test.view.Next(node); next != expected {
				t.Errorf("%s: unexpected next node -- got %v, "+
					"want %v", test.name, next, expected)
				continue testLoop
			}
		}

//确保视图中未包含的节点不
//生成后续节点。
		for _, node := range test.noContains {
			if next := test.view.Next(node); next != nil {
				t.Errorf("%s: unexpected next node -- got %v, "+
					"want nil", test.name, next)
				continue testLoop
			}
		}

//确保视图中包含的所有节点都可以通过
//高度。
		for _, wantNode := range test.contains {
			node := test.view.NodeByHeight(wantNode.height)
			if node != wantNode {
				t.Errorf("%s: unexpected node for height %d -- "+
					"got %v, want %v", test.name,
					wantNode.height, node, wantNode)
				continue testLoop
			}
		}

//确保活动视图尖端的块定位器
//由预期的哈希组成。
		locator := test.view.BlockLocator(test.view.tip())
		if !reflect.DeepEqual(locator, test.locator) {
			t.Errorf("%s: unexpected locator -- got %v, want %v",
				test.name, locator, test.locator)
			continue
		}
	}
}

//TestChainViewForkCorners确保在两个链之间查找叉
//在某些拐角情况下工作，例如当两条链条完全
//无关的历史。
func TestChainViewForkCorners(t *testing.T) {
//构造两个无关的单分支合成块索引。
	branchNodes := chainedNodes(nil, 5)
	unrelatedBranchNodes := chainedNodes(nil, 7)

//为两个不相关的历史创建链视图。
	view1 := newChainView(tstTip(branchNodes))
	view2 := newChainView(tstTip(unrelatedBranchNodes))

//确保尝试查找不存在节点的分叉点
//不生成节点。
	if fork := view1.FindFork(nil); fork != nil {
		t.Fatalf("FindFork: unexpected fork -- got %v, want nil", fork)
	}

//确保尝试在两个链视图中查找分叉点
//完全无关的历史不会产生节点。
	for _, node := range branchNodes {
		if fork := view2.FindFork(node); fork != nil {
			t.Fatalf("FindFork: unexpected fork -- got %v, want nil",
				fork)
		}
	}
	for _, node := range unrelatedBranchNodes {
		if fork := view1.FindFork(node); fork != nil {
			t.Fatalf("FindFork: unexpected fork -- got %v, want nil",
				fork)
		}
	}
}

//testchainviewsettip确保按预期更改提示，包括
//容量变化。
func TestChainViewSetTip(t *testing.T) {
//构造一个由以下内容组成的综合块索引
//结构。
//0->1->2->3->4
//\->2a->3a->4a->5a->6a->7a->-> 26A
	branch0Nodes := chainedNodes(nil, 5)
	branch1Nodes := chainedNodes(branch0Nodes[1], 25)

	tip := tstTip
	tests := []struct {
		name     string
view     *chainView     //活动视图
tips     []*blockNode   //设置提示
contains [][]*blockNode //每个提示视图中的预期节点
	}{
		{
//创建一个空视图并将提示设置为
//更长的链条。
			name:     "increasing",
			view:     newChainView(nil),
			tips:     []*blockNode{tip(branch0Nodes), tip(branch1Nodes)},
			contains: [][]*blockNode{branch0Nodes, branch1Nodes},
		},
		{
//使用较长的链创建视图，并将尖端设置为
//链条越来越短。
			name:     "decreasing",
			view:     newChainView(tip(branch1Nodes)),
			tips:     []*blockNode{tip(branch0Nodes), nil},
			contains: [][]*blockNode{branch0Nodes, nil},
		},
		{
//使用较短的链创建视图，并将尖端设置为
//一条较长的链条，然后将其放回
//短链。
			name:     "small-large-small",
			view:     newChainView(tip(branch0Nodes)),
			tips:     []*blockNode{tip(branch1Nodes), tip(branch0Nodes)},
			contains: [][]*blockNode{branch1Nodes, branch0Nodes},
		},
		{
//使用较长的链创建视图，并将尖端设置为
//一个较小的链，然后将其设置回
//更长的链。
			name:     "large-small-large",
			view:     newChainView(tip(branch1Nodes)),
			tips:     []*blockNode{tip(branch0Nodes), tip(branch1Nodes)},
			contains: [][]*blockNode{branch0Nodes, branch1Nodes},
		},
	}

testLoop:
	for _, test := range tests {
		for i, tip := range test.tips {
//确保视图提示是预期的节点。
			test.view.SetTip(tip)
			if test.view.Tip() != tip {
				t.Errorf("%s: unexpected view tip -- got %v, "+
					"want %v", test.name, test.view.Tip(),
					tip)
				continue testLoop
			}

//确保视图中包含所有预期节点。
			for _, node := range test.contains[i] {
				if !test.view.Contains(node) {
					t.Errorf("%s: expected %v in active view",
						test.name, node)
					continue testLoop
				}
			}

		}
	}
}

//testchainviewnil确保创建和访问nil链视图的行为
//果不其然。
func TestChainViewNil(t *testing.T) {
//确保将两个未初始化的视图视为相等。
	view := newChainView(nil)
	if !view.Equals(newChainView(nil)) {
		t.Fatal("uninitialized nil views unequal")
	}

//确保未初始化视图的起源不会生成节点。
	if genesis := view.Genesis(); genesis != nil {
		t.Fatalf("Genesis: unexpected genesis -- got %v, want nil",
			genesis)
	}

//确保未初始化视图的提示不生成节点。
	if tip := view.Tip(); tip != nil {
		t.Fatalf("Tip: unexpected tip -- got %v, want nil", tip)
	}

//确保未初始化视图的高度为预期值。
	if height := view.Height(); height != -1 {
		t.Fatalf("Height: unexpected height -- got %d, want -1", height)
	}

//确保尝试获取不存在高度的节点不存在
//不生成节点。
	if node := view.NodeByHeight(10); node != nil {
		t.Fatalf("NodeByHeight: unexpected node -- got %v, want nil", node)
	}

//确保未初始化的视图不报告它包含节点。
	fakeNode := chainedNodes(nil, 1)[0]
	if view.Contains(fakeNode) {
		t.Fatalf("Contains: view claims it contains node %v", fakeNode)
	}

//确保不存在的节点的下一个节点不会生成
//一个节点。
	if next := view.Next(nil); next != nil {
		t.Fatalf("Next: unexpected next node -- got %v, want nil", next)
	}

//确保存在的节点的下一个节点不会生成节点。
	if next := view.Next(fakeNode); next != nil {
		t.Fatalf("Next: unexpected next node -- got %v, want nil", next)
	}

//确保尝试查找不存在节点的分叉点
//不生成节点。
	if fork := view.FindFork(nil); fork != nil {
		t.Fatalf("FindFork: unexpected fork -- got %v, want nil", fork)
	}

//确保尝试获取尖端的块定位器不会产生
//因为小费是零。
	if locator := view.BlockLocator(nil); locator != nil {
		t.Fatalf("BlockLocator: unexpected locator -- got %v, want nil",
			locator)
	}

//确保尝试获取仍存在的节点的块定位器
//按预期工作。
	branchNodes := chainedNodes(nil, 50)
	wantLocator := locatorHashes(branchNodes, 49, 48, 47, 46, 45, 44, 43,
		42, 41, 40, 39, 38, 36, 32, 24, 8, 0)
	locator := view.BlockLocator(tstTip(branchNodes))
	if !reflect.DeepEqual(locator, wantLocator) {
		t.Fatalf("BlockLocator: unexpected locator -- got %v, want %v",
			locator, wantLocator)
	}
}
