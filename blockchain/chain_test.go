
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package blockchain

import (
	"reflect"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

//TestHaveBlock测试HaveBlock API以确保正确的功能。
func TestHaveBlock(t *testing.T) {
//把积木装好，这样就有了侧链。
//（Genesis区块）->1->2->3->4
//\> 3A
	testFiles := []string{
		"blk_0_to_4.dat.bz2",
		"blk_3A.dat.bz2",
	}

	var blocks []*btcutil.Block
	for _, file := range testFiles {
		blockTmp, err := loadBlocks(file)
		if err != nil {
			t.Errorf("Error loading file: %v\n", err)
			return
		}
		blocks = append(blocks, blockTmp...)
	}

//创建一个新的数据库和链实例来运行测试。
	chain, teardownFunc, err := chainSetup("haveblock",
		&chaincfg.MainNetParams)
	if err != nil {
		t.Errorf("Failed to setup chain instance: %v", err)
		return
	}
	defer teardownFunc()

//既然我们不处理真正的区块链，那么就设置coinbase
//成熟度为1。
	chain.TstSetCoinbaseMaturity(1)

	for i := 1; i < len(blocks); i++ {
		_, isOrphan, err := chain.ProcessBlock(blocks[i], BFNone)
		if err != nil {
			t.Errorf("ProcessBlock fail on block %v: %v\n", i, err)
			return
		}
		if isOrphan {
			t.Errorf("ProcessBlock incorrectly returned block %v "+
				"is an orphan\n", i)
			return
		}
	}

//插入孤立块。
	_, isOrphan, err := chain.ProcessBlock(btcutil.NewBlock(&Block100000),
		BFNone)
	if err != nil {
		t.Errorf("Unable to process block: %v", err)
		return
	}
	if !isOrphan {
		t.Errorf("ProcessBlock indicated block is an not orphan when " +
			"it should be\n")
		return
	}

	tests := []struct {
		hash string
		want bool
	}{
//应存在Genesis区块（在主链中）。
		{hash: chaincfg.MainNetParams.GenesisHash.String(), want: true},

//应该有3A块（在侧链上）。
		{hash: "00000000474284d20067a4d33f6a02284e6ef70764a3a26d6a5b9df52ef663dd", want: true},

//应该有100000个街区（作为孤儿）。
		{hash: "000000000003ba27aa200b1cecaad478d2b00432346c3f1f3986da1afd33e506", want: true},

//不应提供随机哈希。
		{hash: "123", want: false},
	}

	for i, test := range tests {
		hash, err := chainhash.NewHashFromStr(test.hash)
		if err != nil {
			t.Errorf("NewHashFromStr: %v", err)
			continue
		}

		result, err := chain.HaveBlock(hash)
		if err != nil {
			t.Errorf("HaveBlock #%d unexpected error: %v", i, err)
			return
		}
		if result != test.want {
			t.Errorf("HaveBlock #%d got %v want %v", i, result,
				test.want)
			continue
		}
	}
}

//TestCalcSequenceLock测试LockTimeToSequence函数，以及
//链实例的CalcSequenceLock方法。测试有几项
//CalcSequenceLock函数的输入组合，以确保
//返回的SequenceLocks对于每个测试实例都是正确的。
func TestCalcSequenceLock(t *testing.T) {
	netParams := &chaincfg.SimNetParams

//我们需要激活csv来测试处理逻辑，所以
//手动制作用于向软叉发送信号的块版本
//激活。
	csvBit := netParams.Deployments[chaincfg.DeploymentCSV].BitNumber
	blockVersion := int32(0x20000000 | (uint32(1) << csvBit))

//生成足够的合成块来激活csv。
	chain := newFakeChain(netParams)
	node := chain.bestChain.Tip()
	blockTime := node.Header().Timestamp
	numBlocksToActivate := (netParams.MinerConfirmationWindow * 3)
	for i := uint32(0); i < numBlocksToActivate; i++ {
		blockTime = blockTime.Add(time.Second)
		node = newFakeNode(node, blockVersion, 0, blockTime)
		chain.index.AddNode(node)
		chain.bestChain.SetTip(node)
	}

//为中使用的输入创建一个带有假utxo的utxo视图
//在下面创建的交易记录。该utxo的添加方式使其具有
//4个街区的年龄。
	targetTx := btcutil.NewTx(&wire.MsgTx{
		TxOut: []*wire.TxOut{{
			PkScript: nil,
			Value:    10,
		}},
	})
	utxoView := NewUtxoViewpoint()
	utxoView.AddTxOuts(targetTx, int32(numBlocksToActivate)-4)
	utxoView.SetBestHash(&node.hash)

//创建一个将上面创建的假utxo用于
//在测试中创建的事务。它有4个街区。注释
//序列锁高度总是从相同的
//他们最初是根据给定的
//UTXO。也就是说，它之前的高度。
	utxo := wire.OutPoint{
		Hash:  *targetTx.Hash(),
		Index: 0,
	}
	prevUtxoHeight := int32(numBlocksToActivate) - 4

//从上面创建的输入的POV中获取经过的中间时间。
//输入的MTP是块*prior*的POV的MTP。
//包括它的那个。
	medianTime := node.RelativeAncestor(5).CalcPastMedianTime().Unix()

//根据最佳块的POV计算的中间时间
//测试链。对于未确认的输入，将使用此值，因为
//MTP将根据尚未开采的POV计算。
//块。
	nextMedianTime := node.CalcPastMedianTime().Unix()
	nextBlockHeight := int32(numBlocksToActivate) + 1

//添加一个额外的交易，作为我们的未确认
//输出。
	unConfTx := &wire.MsgTx{
		TxOut: []*wire.TxOut{{
			PkScript: nil,
			Value:    5,
		}},
	}
	unConfUtxo := wire.OutPoint{
		Hash:  unConfTx.TxHash(),
		Index: 0,
	}

//添加高度为0x7fffffff的utxo表示输出
//当前未链接。
	utxoView.AddTxOuts(btcutil.NewTx(unConfTx), 0x7fffffff)

	tests := []struct {
		tx      *wire.MsgTx
		view    *UtxoViewpoint
		mempool bool
		want    *SequenceLock
	}{
//
//因为新的序列号语义只适用于
//事务版本2或更高。
		{
			tx: &wire.MsgTx{
				Version: 1,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(false, 3),
				}},
			},
			view: utxoView,
			want: &SequenceLock{
				Seconds:     -1,
				BlockHeight: -1,
			},
		},
//具有最大序列号的单个输入的事务。
//
//应该被禁用。
		{
			tx: &wire.MsgTx{
				Version: 2,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: utxo,
					Sequence:         wire.MaxTxInSequenceNum,
				}},
			},
			view: utxoView,
			want: &SequenceLock{
				Seconds:     -1,
				BlockHeight: -1,
			},
		},
//具有单个输入的事务，其锁定时间为
//以秒表示。但是，指定的锁定时间是
//低于所需的时间锁定时间下限
//它们的时间粒度为512秒。因此，
//秒锁定时间应在
//目标块。
		{
			tx: &wire.MsgTx{
				Version: 2,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(true, 2),
				}},
			},
			view: utxoView,
			want: &SequenceLock{
				Seconds:     medianTime - 1,
				BlockHeight: -1,
			},
		},
//具有单个输入的事务，其锁定时间为
//以秒表示。秒数应为1023
//最后一个块的中位过去时间后的秒数
//链。
		{
			tx: &wire.MsgTx{
				Version: 2,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(true, 1024),
				}},
			},
			view: utxoView,
			want: &SequenceLock{
				Seconds:     medianTime + 1023,
				BlockHeight: -1,
			},
		},
//具有多个输入的事务。第一个输入具有
//锁定时间（秒）。第二个输入具有
//值为4的块中的序列锁定。最后输入
//具有值为5的序列号，但具有禁用
//位集。所以应该选择第一个锁，因为它是
//未禁用的最新锁。
		{
			tx: &wire.MsgTx{
				Version: 2,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(true, 2560),
				}, {
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(false, 4),
				}, {
					PreviousOutPoint: utxo,
					Sequence: LockTimeToSequence(false, 5) |
						wire.SequenceLockTimeDisabled,
				}},
			},
			view: utxoView,
			want: &SequenceLock{
				Seconds:     medianTime + (5 << wire.SequenceLockTimeGranularity) - 1,
				BlockHeight: prevUtxoHeight + 3,
			},
		},
//单输入事务。输入的序列号
//以块（3个块）为单位对相对锁定时间进行编码。这个
//序列锁的值应为-1秒，但
//高度为2表示可以包括在高度3。
		{
			tx: &wire.MsgTx{
				Version: 2,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(false, 3),
				}},
			},
			view: utxoView,
			want: &SequenceLock{
				Seconds:     -1,
				BlockHeight: prevUtxoHeight + 2,
			},
		},
//具有两个输入的事务，锁定时间用
//秒。选定的序列锁定值（秒）应
//在未来更进一步。
		{
			tx: &wire.MsgTx{
				Version: 2,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(true, 5120),
				}, {
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(true, 2560),
				}},
			},
			view: utxoView,
			want: &SequenceLock{
				Seconds:     medianTime + (10 << wire.SequenceLockTimeGranularity) - 1,
				BlockHeight: -1,
			},
		},
//具有两个输入的事务，锁定时间用
//阻碍。为块选择的序列锁定值应
//未来的高度会更高，所以高度是10
//表示可包括在高度11处。
		{
			tx: &wire.MsgTx{
				Version: 2,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(false, 1),
				}, {
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(false, 11),
				}},
			},
			view: utxoView,
			want: &SequenceLock{
				Seconds:     -1,
				BlockHeight: prevUtxoHeight + 10,
			},
		},
//具有多个输入的事务。两个输入是时间
//另外两个是基于块的。锁着
//在未来，应选择两种输入。
		{
			tx: &wire.MsgTx{
				Version: 2,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(true, 2560),
				}, {
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(true, 6656),
				}, {
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(false, 3),
				}, {
					PreviousOutPoint: utxo,
					Sequence:         LockTimeToSequence(false, 9),
				}},
			},
			view: utxoView,
			want: &SequenceLock{
				Seconds:     medianTime + (13 << wire.SequenceLockTimeGranularity) - 1,
				BlockHeight: prevUtxoHeight + 8,
			},
		},
//具有单个未确认输入的事务。作为输入
//确认后，应解释输入的高度
//作为*下一个*块的高度。所以，一个2个街区的亲戚
//锁定表示序列锁定应在
//*下一个*块高度，表示可以包含2个块
//之后。
		{
			tx: &wire.MsgTx{
				Version: 2,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: unConfUtxo,
					Sequence:         LockTimeToSequence(false, 2),
				}},
			},
			view:    utxoView,
			mempool: true,
			want: &SequenceLock{
				Seconds:     -1,
				BlockHeight: nextBlockHeight + 1,
			},
		},
//具有单个未确认输入的事务。输入有
//基于时间的锁，因此锁定时间应基于
//*下一个*块的MTP。
		{
			tx: &wire.MsgTx{
				Version: 2,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: unConfUtxo,
					Sequence:         LockTimeToSequence(true, 1024),
				}},
			},
			view:    utxoView,
			mempool: true,
			want: &SequenceLock{
				Seconds:     nextMedianTime + 1023,
				BlockHeight: -1,
			},
		},
	}

	t.Logf("Running %v SequenceLock tests", len(tests))
	for i, test := range tests {
		utilTx := btcutil.NewTx(test.tx)
		seqLock, err := chain.CalcSequenceLock(utilTx, test.view, test.mempool)
		if err != nil {
			t.Fatalf("test #%d, unable to calc sequence lock: %v", i, err)
		}

		if seqLock.Seconds != test.want.Seconds {
			t.Fatalf("test #%d got %v seconds want %v seconds",
				i, seqLock.Seconds, test.want.Seconds)
		}
		if seqLock.BlockHeight != test.want.BlockHeight {
			t.Fatalf("test #%d got height of %v want height of %v ",
				i, seqLock.BlockHeight, test.want.BlockHeight)
		}
	}
}

//nodeHash是一个方便函数，它返回所有
//已传递所提供节点的索引。它用于构造预期的哈希
//在测试中进行切片。
func nodeHashes(nodes []*blockNode, indexes ...int) []chainhash.Hash {
	hashes := make([]chainhash.Hash, 0, len(indexes))
	for _, idx := range indexes {
		hashes = append(hashes, nodes[idx].hash)
	}
	return hashes
}

//nodeheaders是一个方便的函数，它返回
//所提供节点的已传递索引。它用于构造预期的
//已在测试中找到邮件头。
func nodeHeaders(nodes []*blockNode, indexes ...int) []wire.BlockHeader {
	headers := make([]wire.BlockHeader, 0, len(indexes))
	for _, idx := range indexes {
		headers = append(headers, nodes[idx].Header())
	}
	return headers
}

//testlocateinventory确保通过locateheaders和
//locateBlocks函数的行为与预期一致。
func TestLocateInventory(t *testing.T) {
//构造一个包含块索引的合成块链
//
//
//> -16A-＞17A
	tip := tstTip
	chain := newFakeChain(&chaincfg.MainNetParams)
	branch0Nodes := chainedNodes(chain.bestChain.Genesis(), 18)
	branch1Nodes := chainedNodes(branch0Nodes[14], 2)
	for _, node := range branch0Nodes {
		chain.index.AddNode(node)
	}
	for _, node := range branch1Nodes {
		chain.index.AddNode(node)
	}
	chain.bestChain.SetTip(tip(branch0Nodes))

//为整个链的不同分支创建链视图
//在链的不同部分模拟本地和远程节点。
	localView := newChainView(tip(branch0Nodes))
	remoteView := newChainView(tip(branch1Nodes))

//为完全不相关的块链创建链视图
//在完全不同的链上模拟远程节点。
	unrelatedBranchNodes := chainedNodes(nil, 5)
	unrelatedView := newChainView(tip(unrelatedBranchNodes))

	tests := []struct {
		name       string
locator    BlockLocator       //请求库存的定位器
hashStop   chainhash.Hash     //停止定位器的哈希
maxAllowed uint32             //要定位的最大值，0=导线常数
headers    []wire.BlockHeader //应为已定位的邮件头
hashes     []chainhash.Hash   //应为定位哈希
	}{
		{
//空块定位器和未知的停止哈希。不
//应找到库存。
			name:     "no locators, no stop",
			locator:  nil,
			hashStop: chainhash.Hash{},
			headers:  nil,
			hashes:   nil,
		},
		{
//清空块定位器并停止侧链中的哈希。
//预期结果是请求的块。
			name:     "no locators, stop in side",
			locator:  nil,
			hashStop: tip(branch1Nodes).hash,
			headers:  nodeHeaders(branch1Nodes, 1),
			hashes:   nodeHashes(branch1Nodes, 1),
		},
		{
//清空块定位器并停止主链中的哈希。
//预期结果是请求的块。
			name:     "no locators, stop in main",
			locator:  nil,
			hashStop: branch0Nodes[12].hash,
			headers:  nodeHeaders(branch0Nodes, 12),
			hashes:   nodeHashes(branch0Nodes, 12),
		},
		{
//基于远程侧链的定位器
//停止哈希本地节点不知道。这个
//预期结果是分叉点后的块
//主链和停止哈希没有效果。
			name:     "remote side chain, unknown stop",
			locator:  remoteView.BlockLocator(nil),
			hashStop: chainhash.Hash{0x01},
			headers:  nodeHeaders(branch0Nodes, 15, 16, 17),
			hashes:   nodeHashes(branch0Nodes, 15, 16, 17),
		},
		{
//基于远程侧链的定位器
//停止侧链中的哈希。预期结果是
//主链中叉点后的块和
//停止哈希无效。
			name:     "remote side chain, stop in side",
			locator:  remoteView.BlockLocator(nil),
			hashStop: tip(branch1Nodes).hash,
			headers:  nodeHeaders(branch0Nodes, 15, 16, 17),
			hashes:   nodeHashes(branch0Nodes, 15, 16, 17),
		},
		{
//基于远程侧链的定位器
//停止主链中的哈希，但在分叉点之前。这个
//预期结果是分叉点后的块
//主链和停止哈希没有效果。
			name:     "remote side chain, stop in main before",
			locator:  remoteView.BlockLocator(nil),
			hashStop: branch0Nodes[13].hash,
			headers:  nodeHeaders(branch0Nodes, 15, 16, 17),
			hashes:   nodeHashes(branch0Nodes, 15, 16, 17),
		},
		{
//基于远程侧链的定位器
//停止主链中的哈希，但正好在分叉处
//点。预期结果是
//主链中的分叉点和停止哈希没有
//效果。
			name:     "remote side chain, stop in main exact",
			locator:  remoteView.BlockLocator(nil),
			hashStop: branch0Nodes[14].hash,
			headers:  nodeHeaders(branch0Nodes, 15, 16, 17),
			hashes:   nodeHashes(branch0Nodes, 15, 16, 17),
		},
		{
//基于远程侧链的定位器
//在分叉点之后停止主链中的哈希。
//预期结果是fork后面的块
//主链上的点，直至并包括止动块
//搞砸。
			name:     "remote side chain, stop in main after",
			locator:  remoteView.BlockLocator(nil),
			hashStop: branch0Nodes[15].hash,
			headers:  nodeHeaders(branch0Nodes, 15),
			hashes:   nodeHashes(branch0Nodes, 15),
		},
		{
//基于远程侧链的定位器
//在分叉后一段时间停止主链中的哈希
//点。预期结果是
//主链上的叉点
//停止哈希。
			name:     "remote side chain, stop in main after more",
			locator:  remoteView.BlockLocator(nil),
			hashStop: branch0Nodes[16].hash,
			headers:  nodeHeaders(branch0Nodes, 15, 16),
			hashes:   nodeHashes(branch0Nodes, 15, 16),
		},
		{
//基于远程在主链上的定位器
//过去和停止哈希本地节点不知道。
//预期结果是已知的
//主链中的点，停止哈希没有
//效果。
			name:     "remote main chain past, unknown stop",
			locator:  localView.BlockLocator(branch0Nodes[12]),
			hashStop: chainhash.Hash{0x01},
			headers:  nodeHeaders(branch0Nodes, 13, 14, 15, 16, 17),
			hashes:   nodeHashes(branch0Nodes, 13, 14, 15, 16, 17),
		},
		{
//基于远程在主链上的定位器
//在侧链中经过一个停止哈希。预期的
//结果是块位于
//主链和停止哈希没有效果。
			name:     "remote main chain past, stop in side",
			locator:  localView.BlockLocator(branch0Nodes[12]),
			hashStop: tip(branch1Nodes).hash,
			headers:  nodeHeaders(branch0Nodes, 13, 14, 15, 16, 17),
			hashes:   nodeHashes(branch0Nodes, 13, 14, 15, 16, 17),
		},
		{
//基于远程在主链上的定位器
//前面的主链中有一个停止哈希
//点。预期结果是
//主链中的已知点和停止哈希具有
//没有效果。
			name:     "remote main chain past, stop in main before",
			locator:  localView.BlockLocator(branch0Nodes[12]),
			hashStop: branch0Nodes[11].hash,
			headers:  nodeHeaders(branch0Nodes, 13, 14, 15, 16, 17),
			hashes:   nodeHashes(branch0Nodes, 13, 14, 15, 16, 17),
		},
		{
//基于远程在主链上的定位器
//在主链上的一个停止哈希
//点。预期结果是
//主链中的已知点和停止哈希具有
//没有效果。
			name:     "remote main chain past, stop in main exact",
			locator:  localView.BlockLocator(branch0Nodes[12]),
			hashStop: branch0Nodes[12].hash,
			headers:  nodeHeaders(branch0Nodes, 13, 14, 15, 16, 17),
			hashes:   nodeHashes(branch0Nodes, 13, 14, 15, 16, 17),
		},
		{
//基于远程在主链上的定位器
//在主链中的过去和一个停止哈希之后
//那一点。预期结果是后面的块
//主链中的已知点和停止哈希
//没有效果。
			name:     "remote main chain past, stop in main after",
			locator:  localView.BlockLocator(branch0Nodes[12]),
			hashStop: branch0Nodes[13].hash,
			headers:  nodeHeaders(branch0Nodes, 13),
			hashes:   nodeHashes(branch0Nodes, 13),
		},
		{
//基于远程在主链上的定位器
//主链中的过去和停止哈希
//在那之后。预期结果是块
//在主链和挡块上的已知点之后
//哈希无效。
			name:     "remote main chain past, stop in main after more",
			locator:  localView.BlockLocator(branch0Nodes[12]),
			hashStop: branch0Nodes[15].hash,
			headers:  nodeHeaders(branch0Nodes, 13, 14, 15),
			hashes:   nodeHashes(branch0Nodes, 13, 14, 15),
		},
		{
//基于远程位置完全相同的定位器
//主链中的点和停止哈希本地节点
//不知道。预期结果为否
//已找到库存。
			name:     "remote main chain same, unknown stop",
			locator:  localView.BlockLocator(nil),
			hashStop: chainhash.Hash{0x01},
			headers:  nil,
			hashes:   nil,
		},
		{
//基于远程位置完全相同的定位器
//主链中的点和
//同样的观点。未找到预期结果
//库存。
			name:     "remote main chain same, stop same point",
			locator:  localView.BlockLocator(nil),
			hashStop: tip(branch0Nodes).hash,
			headers:  nil,
			hashes:   nil,
		},
		{
//远程定位器，不包括任何块
//本地节点知道。如果
//远程节点位于一个完全独立的链上，
//不是同一个起源块的根。这个
//预期结果是发生后的块体
//块。
			name:     "remote unrelated chain",
			locator:  unrelatedView.BlockLocator(nil),
			hashStop: chainhash.Hash{},
			headers: nodeHeaders(branch0Nodes, 0, 1, 2, 3, 4, 5, 6,
				7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17),
			hashes: nodeHashes(branch0Nodes, 0, 1, 2, 3, 4, 5, 6,
				7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17),
		},
		{
//主链第二个区块的远程定位器
//并且没有停止哈希，但具有重写的最大限制。
//预期结果是第二个块之后的块
//块受最大值限制。
			name:       "remote genesis",
			locator:    locatorHashes(branch0Nodes, 0),
			hashStop:   chainhash.Hash{},
			maxAllowed: 3,
			headers:    nodeHeaders(branch0Nodes, 1, 2, 3),
			hashes:     nodeHashes(branch0Nodes, 1, 2, 3),
		},
		{
//定位器格式不正确。
//
//远程定位器，仅包括一个
//在本地节点知道的侧链上阻塞。这个
//预期结果是发生后的块体
//阻止，因为即使该阻止已知，它仍处于打开状态
//侧链和没有更多的定位器可以找到
//叉点。
			name:     "weak locator, single known side block",
			locator:  locatorHashes(branch1Nodes, 1),
			hashStop: chainhash.Hash{},
			headers: nodeHeaders(branch0Nodes, 0, 1, 2, 3, 4, 5, 6,
				7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17),
			hashes: nodeHashes(branch0Nodes, 0, 1, 2, 3, 4, 5, 6,
				7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17),
		},
		{
//定位器格式不正确。
//
//远程定位器，仅包括多个
//但是，本地节点知道侧链上的块
//主链中没有。预期结果是
//在创世纪之后的街区
//块是已知的，它们都在侧链上，并且
//没有更多的定位器可以找到分叉点。
			name:     "weak locator, multiple known side blocks",
			locator:  locatorHashes(branch1Nodes, 1),
			hashStop: chainhash.Hash{},
			headers: nodeHeaders(branch0Nodes, 0, 1, 2, 3, 4, 5, 6,
				7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17),
			hashes: nodeHashes(branch0Nodes, 0, 1, 2, 3, 4, 5, 6,
				7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17),
		},
		{
//定位器格式不正确。
//
//远程定位器，仅包括多个
//但是，本地节点知道侧链上的块
//在主链中没有，但包含一个停止哈希
//主链。预期结果是块
//在Genesis阻塞之后，直到停止哈希
//尽管已知这些块，但它们都位于
//侧链和没有更多的定位器找到
//叉点。
			name:     "weak locator, multiple known side blocks, stop in main",
			locator:  locatorHashes(branch1Nodes, 1),
			hashStop: branch0Nodes[5].hash,
			headers:  nodeHeaders(branch0Nodes, 0, 1, 2, 3, 4, 5),
			hashes:   nodeHashes(branch0Nodes, 0, 1, 2, 3, 4, 5),
		},
	}
	for _, test := range tests {
//确保找到预期的收割台。
		var headers []wire.BlockHeader
		if test.maxAllowed != 0 {
//需要使用未排序函数重写
//头允许的最大值。
			chain.chainLock.RLock()
			headers = chain.locateHeaders(test.locator,
				&test.hashStop, test.maxAllowed)
			chain.chainLock.RUnlock()
		} else {
			headers = chain.LocateHeaders(test.locator,
				&test.hashStop)
		}
		if !reflect.DeepEqual(headers, test.headers) {
			t.Errorf("%s: unxpected headers -- got %v, want %v",
				test.name, headers, test.headers)
			continue
		}

//确保找到预期的块哈希。
		maxAllowed := uint32(wire.MaxBlocksPerMsg)
		if test.maxAllowed != 0 {
			maxAllowed = test.maxAllowed
		}
		hashes := chain.LocateBlocks(test.locator, &test.hashStop,
			maxAllowed)
		if !reflect.DeepEqual(hashes, test.hashes) {
			t.Errorf("%s: unxpected hashes -- got %v, want %v",
				test.name, hashes, test.hashes)
			continue
		}
	}
}

//testheighttohashrange确保通过start获取一系列块哈希
//高度和结束哈希按预期工作。
func TestHeightToHashRange(t *testing.T) {
//构造一个包含块索引的合成块链
//以下结构。
//《创世纪》->1->2->->15->16->17->18
//\->16A->17A->18A（未验证）
	tip := tstTip
	chain := newFakeChain(&chaincfg.MainNetParams)
	branch0Nodes := chainedNodes(chain.bestChain.Genesis(), 18)
	branch1Nodes := chainedNodes(branch0Nodes[14], 3)
	for _, node := range branch0Nodes {
		chain.index.SetStatusFlags(node, statusValid)
		chain.index.AddNode(node)
	}
	for _, node := range branch1Nodes {
		if node.height < 18 {
			chain.index.SetStatusFlags(node, statusValid)
		}
		chain.index.AddNode(node)
	}
	chain.bestChain.SetTip(tip(branch0Nodes))

	tests := []struct {
		name        string
startHeight int32            //请求库存的定位器
endHash     chainhash.Hash   //停止定位器的哈希
maxResults  int              //要定位的最大值，0=导线常数
hashes      []chainhash.Hash //应为定位哈希
		expectError bool
	}{
		{
			name:        "blocks below tip",
			startHeight: 11,
			endHash:     branch0Nodes[14].hash,
			maxResults:  10,
			hashes:      nodeHashes(branch0Nodes, 10, 11, 12, 13, 14),
		},
		{
			name:        "blocks on main chain",
			startHeight: 15,
			endHash:     branch0Nodes[17].hash,
			maxResults:  10,
			hashes:      nodeHashes(branch0Nodes, 14, 15, 16, 17),
		},
		{
			name:        "blocks on stale chain",
			startHeight: 15,
			endHash:     branch1Nodes[1].hash,
			maxResults:  10,
			hashes: append(nodeHashes(branch0Nodes, 14),
				nodeHashes(branch1Nodes, 0, 1)...),
		},
		{
			name:        "invalid start height",
			startHeight: 19,
			endHash:     branch0Nodes[17].hash,
			maxResults:  10,
			expectError: true,
		},
		{
			name:        "too many results",
			startHeight: 1,
			endHash:     branch0Nodes[17].hash,
			maxResults:  10,
			expectError: true,
		},
		{
			name:        "unvalidated block",
			startHeight: 15,
			endHash:     branch1Nodes[2].hash,
			maxResults:  10,
			expectError: true,
		},
	}
	for _, test := range tests {
		hashes, err := chain.HeightToHashRange(test.startHeight, &test.endHash,
			test.maxResults)
		if err != nil {
			if !test.expectError {
				t.Errorf("%s: unexpected error: %v", test.name, err)
			}
			continue
		}

		if !reflect.DeepEqual(hashes, test.hashes) {
			t.Errorf("%s: unxpected hashes -- got %v, want %v",
				test.name, hashes, test.hashes)
		}
	}
}

//testintervalblockhashes确保在指定的
//按结束哈希排序的间隔按预期工作。
func TestIntervalBlockHashes(t *testing.T) {
//构造一个包含块索引的合成块链
//以下结构。
//《创世纪》->1->2->->15->16->17->18
//\->16A->17A->18A（未验证）
	tip := tstTip
	chain := newFakeChain(&chaincfg.MainNetParams)
	branch0Nodes := chainedNodes(chain.bestChain.Genesis(), 18)
	branch1Nodes := chainedNodes(branch0Nodes[14], 3)
	for _, node := range branch0Nodes {
		chain.index.SetStatusFlags(node, statusValid)
		chain.index.AddNode(node)
	}
	for _, node := range branch1Nodes {
		if node.height < 18 {
			chain.index.SetStatusFlags(node, statusValid)
		}
		chain.index.AddNode(node)
	}
	chain.bestChain.SetTip(tip(branch0Nodes))

	tests := []struct {
		name        string
		endHash     chainhash.Hash
		interval    int
		hashes      []chainhash.Hash
		expectError bool
	}{
		{
			name:     "blocks on main chain",
			endHash:  branch0Nodes[17].hash,
			interval: 8,
			hashes:   nodeHashes(branch0Nodes, 7, 15),
		},
		{
			name:     "blocks on stale chain",
			endHash:  branch1Nodes[1].hash,
			interval: 8,
			hashes: append(nodeHashes(branch0Nodes, 7),
				nodeHashes(branch1Nodes, 0)...),
		},
		{
			name:     "no results",
			endHash:  branch0Nodes[17].hash,
			interval: 20,
			hashes:   []chainhash.Hash{},
		},
		{
			name:        "unvalidated block",
			endHash:     branch1Nodes[2].hash,
			interval:    8,
			expectError: true,
		},
	}
	for _, test := range tests {
		hashes, err := chain.IntervalBlockHashes(&test.endHash, test.interval)
		if err != nil {
			if !test.expectError {
				t.Errorf("%s: unexpected error: %v", test.name, err)
			}
			continue
		}

		if !reflect.DeepEqual(hashes, test.hashes) {
			t.Errorf("%s: unxpected hashes -- got %v, want %v",
				test.name, hashes, test.hashes)
		}
	}
}
