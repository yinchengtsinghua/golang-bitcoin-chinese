
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2016-2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package blockchain

import (
	"math"

	"github.com/btcsuite/btcd/chaincfg"
)

const (
//vblegacyblockversion是在
//版本位方案已激活。
	vbLegacyBlockVersion = 4

//
//正在使用版本位方案。
	vbTopBits = 0x20000000

//vbTopMask是用于确定
//正在使用版本位方案。
	vbTopMask = 0xe0000000

//vbNumBits是可用于
//版本位方案。
	vbNumBits = 29

//UnknownInverNumtocheck是要考虑的前一个块的数目
//检查未知块版本的阈值时，
//警告用户的目的。
	unknownVerNumToCheck = 100

//UnknowNverwarnnum是具有
//用于警告用户的未知版本。
	unknownVerWarnNum = unknownVerNumToCheck / 2
)

//
//测试当一个特定的位不应该被设置时是否被设置。
//根据已知部署和
//链的当前状态。这对于检测和警告
//未知的规则激活。
type bitConditionChecker struct {
	bit   uint32
	chain *BlockChain
}

//确保BitconditionChecker类型实现阈值条件检查器
//接口。
var _ thresholdConditionChecker = bitConditionChecker{}

//BeginTime返回Unix时间戳的中间块时间，在此之后
//
//
//由于此实现检查未知规则，它返回0，因此规则
//总是被视为活跃的。
//
//
func (c bitConditionChecker) BeginTime() uint64 {
	return 0
}

//endtime返回unix时间戳的中间块时间，之后
//尝试的规则更改如果尚未锁定或
//激活。
//
//由于此实现检查未知规则，因此它返回最大值
//可能的时间戳，因此规则始终被视为活动的。
//
//这是ThresholdConditionChecker接口实现的一部分。
func (c bitConditionChecker) EndTime() uint64 {
	return math.MaxUint64
}

//RuleChangeActivationThreshold是条件所针对的块数。
//必须为true才能锁定规则更改。
//
//此实现返回由检查程序的链参数定义的值
//与关联。
//
//这是ThresholdConditionChecker接口实现的一部分。
func (c bitConditionChecker) RuleChangeActivationThreshold() uint32 {
	return c.chain.chainParams.RuleChangeActivationThreshold
}

//MinerConfirmationWindow是每个阈值状态下的块数。
//重新定位窗口。
//
//此实现返回由检查程序的链参数定义的值
//与关联。
//
//这是ThresholdConditionChecker接口实现的一部分。
func (c bitConditionChecker) MinerConfirmationWindow() uint32 {
	return c.chain.chainParams.MinerConfirmationWindow
}

//当与检查器关联的特定位为
//设置，它不应该符合基于
//已知的部署和链的当前状态。
//
//必须在保持链状态锁的情况下调用此函数（用于写入）。
//
//这是ThresholdConditionChecker接口实现的一部分。
func (c bitConditionChecker) Condition(node *blockNode) (bool, error) {
	conditionMask := uint32(1) << c.bit
	version := uint32(node.version)
	if version&vbTopMask != vbTopBits {
		return false, nil
	}
	if version&conditionMask == 0 {
		return false, nil
	}

	expectedVersion, err := c.chain.calcNextBlockVersion(node.parent)
	if err != nil {
		return false, err
	}
	return uint32(expectedVersion)&conditionMask == 0, nil
}

//Deploymenttcher提供了一个阈值条件检查器，可用于
//测试特定部署规则。这是正确检测所必需的
//并激活共识规则变化。
type deploymentChecker struct {
	deployment *chaincfg.ConsensusDeployment
	chain      *BlockChain
}

//确保Deploymenttcher类型实现ThresholdConditionChecker
//接口。
var _ thresholdConditionChecker = deploymentChecker{}

//BeginTime返回Unix时间戳的中间块时间，在此之后
//对规则更改的投票开始（在下一个窗口）。
//
//此实现返回由特定部署定义的值
//检查器与关联。
//
//这是ThresholdConditionChecker接口实现的一部分。
func (c deploymentChecker) BeginTime() uint64 {
	return c.deployment.StartTime
}

//endtime返回unix时间戳的中间块时间，之后
//尝试的规则更改如果尚未锁定或
//激活。
//
//此实现返回由特定部署定义的值
//检查器与关联。
//
//这是ThresholdConditionChecker接口实现的一部分。
func (c deploymentChecker) EndTime() uint64 {
	return c.deployment.ExpireTime
}

//RuleChangeActivationThreshold是条件所针对的块数。
//必须为true才能锁定规则更改。
//
//此实现返回由检查程序的链参数定义的值
//与关联。
//
//这是ThresholdConditionChecker接口实现的一部分。
func (c deploymentChecker) RuleChangeActivationThreshold() uint32 {
	return c.chain.chainParams.RuleChangeActivationThreshold
}

//MinerConfirmationWindow是每个阈值状态下的块数。
//重新定位窗口。
//
//此实现返回由检查程序的链参数定义的值
//与关联。
//
//这是ThresholdConditionChecker接口实现的一部分。
func (c deploymentChecker) MinerConfirmationWindow() uint32 {
	return c.chain.chainParams.MinerConfirmationWindow
}

//当由部署定义的特定位时，条件返回true
//已设置与检查器关联的。
//
//这是ThresholdConditionChecker接口实现的一部分。
func (c deploymentChecker) Condition(node *blockNode) (bool, error) {
	conditionMask := uint32(1) << c.deployment.BitNumber
	version := uint32(node.version)
	return (version&vbTopMask == vbTopBits) && (version&conditionMask != 0),
		nil
}

//CalcNextBlockVersion计算块在
//基于已启动和锁定的状态传递上一个块节点
//规则更改部署。
//
//此函数与导出的CalcNextBlockVersion不同，因为
//导出的版本使用当前最佳链作为前一个块节点
//当此函数接受任何块节点时。
//
//必须在保持链状态锁的情况下调用此函数（用于写入）。
func (b *BlockChain) calcNextBlockVersion(prevNode *blockNode) (int32, error) {
//为每个活动定义的规则部署设置适当的位
//要么是在投票过程中，要么是被锁定在
//在下一个阈值窗口更改时激活。
	expectedVersion := uint32(vbTopBits)
	for id := 0; id < len(b.chainParams.Deployments); id++ {
		deployment := &b.chainParams.Deployments[id]
		cache := &b.deploymentCaches[id]
		checker := deploymentChecker{deployment: deployment, chain: b}
		state, err := b.thresholdState(prevNode, checker, cache)
		if err != nil {
			return 0, err
		}
		if state == ThresholdStarted || state == ThresholdLockedIn {
			expectedVersion |= uint32(1) << deployment.BitNumber
		}
	}
	return int32(expectedVersion), nil
}

//CalcNextBlockVersion计算块在
//
//规则更改部署。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) CalcNextBlockVersion() (int32, error) {
	b.chainLock.Lock()
	version, err := b.calcNextBlockVersion(b.bestChain.Tip())
	b.chainLock.Unlock()
	return version, err
}

//当任何未知的新规则
//即将激活或已激活。这只会发生一次
//当新的规则被激活时，对于那些即将被激活的
//激活。
//
//必须在保持链状态锁的情况下调用此函数（用于写入）
func (b *BlockChain) warnUnknownRuleActivations(node *blockNode) error {
//
//
	for bit := uint32(0); bit < vbNumBits; bit++ {
		checker := bitConditionChecker{bit: bit, chain: b}
		cache := &b.warningCaches[bit]
		state, err := b.thresholdState(node.parent, checker, cache)
		if err != nil {
			return err
		}

		switch state {
		case ThresholdActive:
			if !b.unknownRulesWarned {
				log.Warnf("Unknown new rules activated (bit %d)",
					bit)
				b.unknownRulesWarned = true
			}

		case ThresholdLockedIn:
			window := int32(checker.MinerConfirmationWindow())
			activationHeight := window - (node.height % window)
			log.Warnf("Unknown new rules are about to activate in "+
				"%d blocks (bit %d)", activationHeight, bit)
		}
	}

	return nil
}

//如果最后一个版本的百分比足够高，则warnunknownversions会记录一条警告。
//块具有意外的版本。
//
//必须在保持链状态锁的情况下调用此函数（用于写入）
func (b *BlockChain) warnUnknownVersions(node *blockNode) error {
//如果已经发出警告，则无需采取任何措施。
	if b.unknownVersionsWarned {
		return nil
	}

//如果以前的块有足够多的意外版本，则发出警告。
	numUpgraded := uint32(0)
	for i := uint32(0); i < unknownVerNumToCheck && node != nil; i++ {
		expectedVersion, err := b.calcNextBlockVersion(node.parent)
		if err != nil {
			return err
		}
		if expectedVersion > vbLegacyBlockVersion &&
			(node.version & ^expectedVersion) != 0 {

			numUpgraded++
		}

		node = node.parent
	}
	if numUpgraded > unknownVerWarnNum {
		log.Warn("Unknown block versions are being mined, so new " +
			"rules might be in effect.  Are you running the " +
			"latest version of the software?")
		b.unknownVersionsWarned = true
	}

	return nil
}
