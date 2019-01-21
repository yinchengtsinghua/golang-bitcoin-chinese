
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

//由于以下生成标记，在常规测试期间忽略此文件。
//+建立RPCTEST

package integration

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/integration/rpctest"
)

const (
//vblegacyblockversion是在
//版本位方案已激活。
	vbLegacyBlockVersion = 4

//vbTopBits定义要在版本中设置的位，以指示
//正在使用版本位方案。
	vbTopBits = 0x20000000
)

//断言版本位从给定的测试工具获取传递的块哈希，并
//确保其版本具有所提供的位集或未设置每个集
//旗帜。
func assertVersionBit(r *rpctest.Harness, t *testing.T, hash *chainhash.Hash, bit uint8, set bool) {
	block, err := r.Node.GetBlock(hash)
	if err != nil {
		t.Fatalf("failed to retrieve block %v: %v", hash, err)
	}
	switch {
	case set && block.Header.Version&(1<<bit) == 0:
		_, _, line, _ := runtime.Caller(1)
		t.Fatalf("assertion failed at line %d: block %s, version 0x%x "+
			"does not have bit %d set", line, hash,
			block.Header.Version, bit)
	case !set && block.Header.Version&(1<<bit) != 0:
		_, _, line, _ := runtime.Caller(1)
		t.Fatalf("assertion failed at line %d: block %s, version 0x%x "+
			"has bit %d set", line, hash, block.Header.Version, bit)
	}
}

//断言链高度从给定测试中检索当前链高度
//系好安全带，确保其与所提供的预期高度相匹配。
func assertChainHeight(r *rpctest.Harness, t *testing.T, expectedHeight uint32) {
	height, err := r.Node.GetBlockCount()
	if err != nil {
		t.Fatalf("failed to retrieve block height: %v", err)
	}
	if uint32(height) != expectedHeight {
		_, _, line, _ := runtime.Caller(1)
		t.Fatalf("assertion failed at line %d: block height of %d "+
			"is not the expected %d", line, height, expectedHeight)
	}
}

//ThresholdStateToStatus将传递的阈值状态转换为等效的
//GetBlockChainInfo RPC中返回的状态字符串。
func thresholdStateToStatus(state blockchain.ThresholdState) (string, error) {
	switch state {
	case blockchain.ThresholdDefined:
		return "defined", nil
	case blockchain.ThresholdStarted:
		return "started", nil
	case blockchain.ThresholdLockedIn:
		return "lockedin", nil
	case blockchain.ThresholdActive:
		return "active", nil
	case blockchain.ThresholdFailed:
		return "failed", nil
	}

	return "", fmt.Errorf("unrecognized threshold state: %v", state)
}

//断言SoftForkStatus从给定的
//测试线束并确保提供的软叉钥匙可用，以及
//状态与已传递状态等效。
func assertSoftForkStatus(r *rpctest.Harness, t *testing.T, forkKey string, state blockchain.ThresholdState) {
//将期望的阈值状态转换为等效的
//GetBlockChainInfo RPC状态字符串。
	status, err := thresholdStateToStatus(state)
	if err != nil {
		_, _, line, _ := runtime.Caller(1)
		t.Fatalf("assertion failed at line %d: unable to convert "+
			"threshold state %v to string", line, state)
	}

	info, err := r.Node.GetBlockChainInfo()
	if err != nil {
		t.Fatalf("failed to retrieve chain info: %v", err)
	}

//确保钥匙可用。
	desc, ok := info.Bip9SoftForks[forkKey]
	if !ok {
		_, _, line, _ := runtime.Caller(1)
		t.Fatalf("assertion failed at line %d: softfork status for %q "+
			"is not in getblockchaininfo results", line, forkKey)
	}

//确保状态达到预期值。
	if desc.Status != status {
		_, _, line, _ := runtime.Caller(1)
		t.Fatalf("assertion failed at line %d: softfork status for %q "+
			"is %v instead of expected %v", line, forkKey,
			desc.Status, status)
	}
}

//testbip0009确保bip0009软叉机构遵循状态
//BIP为提供的软分叉键规定的转换规则。它
//uses the regression test network to signal support and advance through the
//各种阈值状态，包括无法实现锁定状态。
//
//有关测试内容的概述，请参见TestBip0009。
//
//注意：这只与导出的版本不同，因为它接受
//要测试的特定软叉部署。
func testBIP0009(t *testing.T, forkKey string, deploymentID uint32) {
//Initialize the primary mining node with only the genesis block.
	r, err := rpctest.New(&chaincfg.RegressionNetParams, nil, nil)
	if err != nil {
		t.Fatalf("unable to create primary harness: %v", err)
	}
	if err := r.SetUp(false, 0); err != nil {
		t.Fatalf("unable to setup test chain: %v", err)
	}
	defer r.TearDown()

//***阈值已定义***
//
//断言链条高度是预期值，软叉
//状态从定义开始。
	assertChainHeight(r, t, 0)
	assertSoftForkStatus(r, t, forkKey, blockchain.ThresholdDefined)

//***阈值定义的第2部分-阈值开始前的1个块***
//
//生成足够的块以达到第一个块之前的高度
//没有信号支持的状态转换，因为状态应该
//一旦到达开始时间，无论
//支持信令。
//
//注意：这是确认窗口前两个街区，因为
//GetBlockChainInfo RPC在
//当前之一。因此，下面的所有高度都被1到
//补偿。
//
//断言链条高度为预期值，软叉状态为
//仍定义，未移动到“开始”。
	confirmationWindow := r.ActiveNet.MinerConfirmationWindow
	for i := uint32(0); i < confirmationWindow-2; i++ {
		_, err := r.GenerateAndSubmitBlock(nil, vbLegacyBlockVersion,
			time.Time{})
		if err != nil {
			t.Fatalf("failed to generated block %d: %v", i, err)
		}
	}
	assertChainHeight(r, t, confirmationWindow-2)
	assertSoftForkStatus(r, t, forkKey, blockchain.ThresholdDefined)

//***阈值已启动***
//
//生成另一个块以到达下一个窗口。
//
//断言链条高度是预期值，软叉
//状态已启动。
	_, err = r.GenerateAndSubmitBlock(nil, vbLegacyBlockVersion, time.Time{})
	if err != nil {
		t.Fatalf("failed to generated block: %v", err)
	}
	assertChainHeight(r, t, confirmationWindow-1)
	assertSoftForkStatus(r, t, forkKey, blockchain.ThresholdStarted)

//***阈值启动第2部分-未能实现阈值锁定***
//
//Generate enough blocks to reach the next window in such a way that
//版本位设置为信号支持的数字块为1
//小于达到锁定状态所需的值。
//
//断言链条高度是预期值，软叉
//状态仍处于启动状态，未移动到锁定状态。
	if deploymentID > uint32(len(r.ActiveNet.Deployments)) {
		t.Fatalf("deployment ID %d does not exist", deploymentID)
	}
	deployment := &r.ActiveNet.Deployments[deploymentID]
	activationThreshold := r.ActiveNet.RuleChangeActivationThreshold
	signalForkVersion := int32(1<<deployment.BitNumber) | vbTopBits
	for i := uint32(0); i < activationThreshold-1; i++ {
		_, err := r.GenerateAndSubmitBlock(nil, signalForkVersion,
			time.Time{})
		if err != nil {
			t.Fatalf("failed to generated block %d: %v", i, err)
		}
	}
	for i := uint32(0); i < confirmationWindow-(activationThreshold-1); i++ {
		_, err := r.GenerateAndSubmitBlock(nil, vbLegacyBlockVersion,
			time.Time{})
		if err != nil {
			t.Fatalf("failed to generated block %d: %v", i, err)
		}
	}
	assertChainHeight(r, t, (confirmationWindow*2)-1)
	assertSoftForkStatus(r, t, forkKey, blockchain.ThresholdStarted)

//***锁定阈值***
//
//生成足够的块以达到下一个窗口的方式
//版本位设置为信号支持的数字块是
//正是达到锁定状态所需的数字。
//
//断言链条高度是预期值，软叉
//状态已移至锁定状态。
	for i := uint32(0); i < activationThreshold; i++ {
		_, err := r.GenerateAndSubmitBlock(nil, signalForkVersion,
			time.Time{})
		if err != nil {
			t.Fatalf("failed to generated block %d: %v", i, err)
		}
	}
	for i := uint32(0); i < confirmationWindow-activationThreshold; i++ {
		_, err := r.GenerateAndSubmitBlock(nil, vbLegacyBlockVersion,
			time.Time{})
		if err != nil {
			t.Fatalf("failed to generated block %d: %v", i, err)
		}
	}
	assertChainHeight(r, t, (confirmationWindow*3)-1)
	assertSoftForkStatus(r, t, forkKey, blockchain.ThresholdLockedIn)

//***第2部分中的阈值锁定——阈值激活前的1个块***
//
//生成足够的块以在下一个块之前达到高度
//窗口，因为它已经
//上了锁。
//
//断言链条高度是预期值，软叉
//状态仍处于锁定状态，未移动到活动状态。
	for i := uint32(0); i < confirmationWindow-1; i++ {
		_, err := r.GenerateAndSubmitBlock(nil, vbLegacyBlockVersion,
			time.Time{})
		if err != nil {
			t.Fatalf("failed to generated block %d: %v", i, err)
		}
	}
	assertChainHeight(r, t, (confirmationWindow*4)-2)
	assertSoftForkStatus(r, t, forkKey, blockchain.ThresholdLockedIn)

//***阈值激活***
//
//生成另一个块以到达下一个窗口而不继续
//信号支持，因为它已经锁定。
//
//断言链条高度是预期值，软叉
//状态已移动到活动状态。
	_, err = r.GenerateAndSubmitBlock(nil, vbLegacyBlockVersion, time.Time{})
	if err != nil {
		t.Fatalf("failed to generated block: %v", err)
	}
	assertChainHeight(r, t, (confirmationWindow*4)-1)
	assertSoftForkStatus(r, t, forkKey, blockchain.ThresholdActive)
}

//testbip0009确保bip0009软叉机构遵循状态
//BIP为所有软分叉制定的转换规则。它使用
//回归测试网络对信号的支持和推进
//阈值状态包括无法达到锁定状态。
//
//概述：
//-断言链高度为0，状态为thresholdDefined
//-生成的块比达到第一状态转换所需的块少1个
//-要求断言链高度，状态仍为thresholdDefined
//-再生成一个块以达到第一状态转换
//- Assert chain height is expected and state moved to ThresholdStarted
//-生成足够的块以到达下一个状态转换窗口，但仅限于
//信号支持比所需数量少1个以实现
//ThresholdLockedIn
//-要求断言链高度，状态仍为thresholdStarted
//-生成足够的块以到达下一个状态转换窗口
//实现锁定状态信号所需的确切块数。
//支持。
//-预计断言链高度，状态移动到thresholdlockedin
//- Generate 1 fewer blocks than needed to reach the next state transition
//-要求断言链高度，状态仍为thresholdLocked。
//-再生成一个块以到达下一个状态转换
//- Assert chain height is expected and state moved to ThresholdActive
func TestBIP0009(t *testing.T) {
	t.Parallel()

	testBIP0009(t, "dummy", chaincfg.DeploymentTestDummy)
	testBIP0009(t, "segwit", chaincfg.DeploymentSegwit)
}

//testbip0009挖掘确保通过btcd的cpu miner生成的块遵循规则
//通过使用试验假人展开，由BIP0009提出。
//
//概述：
//-生成块1
//-未设置断言位（已定义阈值）
//- Generate enough blocks to reach first state transition
//-在状态转换之前没有为块设置断言位
//-在状态转换时为块设置断言位（thresholdstarted）
//- Generate enough blocks to reach second state transition
//-在状态转换时为块设置断言位（thresholdlockedin）
//-生成足够的块以达到第三状态转换
//- Assert bit is set for block prior to state transition (ThresholdLockedIn)
//-在状态转换时没有为块设置断言位（thresholdActive）
func TestBIP0009Mining(t *testing.T) {
	t.Parallel()

//仅使用Genesis块初始化主挖掘节点。
	r, err := rpctest.New(&chaincfg.SimNetParams, nil, nil)
	if err != nil {
		t.Fatalf("unable to create primary harness: %v", err)
	}
	if err := r.SetUp(true, 0); err != nil {
		t.Fatalf("unable to setup test chain: %v", err)
	}
	defer r.TearDown()

//断言链只包含gensis块。
	assertChainHeight(r, t, 0)

//***阈值已定义***
//
//生成扩展Genesis块的块。它不应该
//自第一个窗口以来在版本中设置的测试虚拟位是
//处于定义的阈值状态。
	deployment := &r.ActiveNet.Deployments[chaincfg.DeploymentTestDummy]
	testDummyBitNum := deployment.BitNumber
	hashes, err := r.Node.Generate(1)
	if err != nil {
		t.Fatalf("unable to generate blocks: %v", err)
	}
	assertChainHeight(r, t, 1)
	assertVersionBit(r, t, hashes[0], testDummyBitNum, false)

//***阈值已启动***
//
//生成足够的块以达到第一状态转换。
//
//第二个到最后一个生成的块不应设置测试位
//在版本中。
//
//最后生成的块现在应该在
//自BTCD挖掘代码识别测试以来的版本
//启动虚拟部署。
	confirmationWindow := r.ActiveNet.MinerConfirmationWindow
	numNeeded := confirmationWindow - 1
	hashes, err = r.Node.Generate(numNeeded)
	if err != nil {
		t.Fatalf("failed to generated %d blocks: %v", numNeeded, err)
	}
	assertChainHeight(r, t, confirmationWindow)
	assertVersionBit(r, t, hashes[len(hashes)-2], testDummyBitNum, false)
	assertVersionBit(r, t, hashes[len(hashes)-1], testDummyBitNum, true)

//***锁定阈值***
//
//生成足够的块以到达下一个状态转换。
//
//最后生成的块中仍应设置测试位
//自BTCD挖掘代码识别测试以来的版本
//虚拟部署已锁定。
	hashes, err = r.Node.Generate(confirmationWindow)
	if err != nil {
		t.Fatalf("failed to generated %d blocks: %v", confirmationWindow,
			err)
	}
	assertChainHeight(r, t, confirmationWindow*2)
	assertVersionBit(r, t, hashes[len(hashes)-1], testDummyBitNum, true)

//*** ThresholdActivated ***
//
//生成足够的块以到达下一个状态转换。
//
//第二个到最后一个生成的块仍应设置测试位
//在版本中，因为它仍然被锁定。
//
//最后生成的块不应在
//自BTCD挖掘代码识别测试以来的版本
//虚拟部署已激活，因此不再需要
//设置位。
	hashes, err = r.Node.Generate(confirmationWindow)
	if err != nil {
		t.Fatalf("failed to generated %d blocks: %v", confirmationWindow,
			err)
	}
	assertChainHeight(r, t, confirmationWindow*3)
	assertVersionBit(r, t, hashes[len(hashes)-2], testDummyBitNum, true)
	assertVersionBit(r, t, hashes[len(hashes)-1], testDummyBitNum, false)
}
