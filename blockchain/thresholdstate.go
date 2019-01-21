
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
	"fmt"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

//阈值状态定义投票时使用的各种阈值状态
//共识改变。
type ThresholdState byte

//这些常量用于识别特定的阈值状态。
const (
//ThresholdDefined是每个部署的第一个状态，它是
//根据定义，Genesis区块的状态适用于所有部署。
	ThresholdDefined ThresholdState = iota

//
//已联系。
	ThresholdStarted

//ThresholdLockedin是重定目标期间部署的状态
//阈值开始状态期间之后的期间和
//为部署投票的块数等于或超过
//部署所需的投票数。
	ThresholdLockedIn

//ThresholdActive是在
//重新确定部署处于阈值锁定中的目标期间
//状态。
	ThresholdActive

//ThresholdFailed是部署过期后的状态
//时间已到，但未达到锁定的阈值
//状态。
	ThresholdFailed

//NumThresholdsStates是中使用的最大阈值状态数。
//测验。
	numThresholdsStates
)

//阈值状态字符串是阈值状态值返回到其
//
var thresholdStateStrings = map[ThresholdState]string{
	ThresholdDefined:  "ThresholdDefined",
	ThresholdStarted:  "ThresholdStarted",
	ThresholdLockedIn: "ThresholdLockedIn",
	ThresholdActive:   "ThresholdActive",
	ThresholdFailed:   "ThresholdFailed",
}

//字符串将阈值状态返回为人类可读的名称。
func (t ThresholdState) String() string {
	if s := thresholdStateStrings[t]; s != "" {
		return s
	}
	return fmt.Sprintf("Unknown ThresholdState (%d)", int(t))
}

//ThresholdConditionChecker提供一个通用接口，调用它
//确定何时应更改共识规则更改阈值。
type thresholdConditionChecker interface {
//BeginTime返回Unix时间戳的中间块时间
//对规则更改的投票将开始（在下一个窗口）。
	BeginTime() uint64

//endtime返回Unix时间戳的中间块时间
//
//锁定或激活。
	EndTime() uint64

//RuleChangeActivationThreshold是为其
//条件必须为true才能锁定规则更改。
	RuleChangeActivationThreshold() uint32

//MinerConfirmationWindow是每个阈值中的块数
//状态重定目标窗口。
	MinerConfirmationWindow() uint32

//条件返回规则是否更改激活条件
//已经满足。这通常涉及检查
//与条件关联的位已设置，但可能更复杂，因为
//需要。
	Condition(*blockNode) (bool, error)
}

//ThresholdStateCache提供了一种类型来缓存每个
//
type thresholdStateCache struct {
	entries map[chainhash.Hash]ThresholdState
}

//查找返回与给定哈希关联的阈值状态以及
//一个布尔值，指示它是否有效。
func (c *thresholdStateCache) Lookup(hash *chainhash.Hash) (ThresholdState, bool) {
	state, ok := c.entries[*hash]
	return state, ok
}

//
//映射。
func (c *thresholdStateCache) Update(hash *chainhash.Hash, state ThresholdState) {
	c.entries[*hash] = state
}

//NewThresholdCaches返回计算时要使用的新缓存数组
//阈值状态。
func newThresholdCaches(numCaches uint32) []thresholdStateCache {
	caches := make([]thresholdStateCache, numCaches)
	for i := 0; i < len(caches); i++ {
		caches[i] = thresholdStateCache{
			entries: make(map[chainhash.Hash]ThresholdState),
		}
	}
	return caches
}

//threshold state返回块的当前规则更改阈值状态
//在给定的节点和部署ID之后。缓存用于确保
//以前窗口的阈值状态只计算一次。
//
//必须在保持链状态锁的情况下调用此函数（用于写入）。
func (b *BlockChain) thresholdState(prevNode *blockNode, checker thresholdConditionChecker, cache *thresholdStateCache) (ThresholdState, error) {
//包含Genesis块的窗口的阈值状态为
//由定义定义。
	confirmationWindow := int32(checker.MinerConfirmationWindow())
	if prevNode == nil || (prevNode.height+1) < confirmationWindow {
		return ThresholdDefined, nil
	}

//获取上一个确认的最后一个块的祖先
//窗口以获取其阈值状态。这样做是因为
//给定窗口中所有块的状态都相同。
	prevNode = prevNode.Ancestor(prevNode.height -
		(prevNode.height+1)%confirmationWindow)

//在以前的每个确认窗口中向后迭代
//查找最近缓存的阈值状态。
	var neededStates []*blockNode
	for prevNode != nil {
//如果块的状态为
//缓存。
		if _, ok := cache.Lookup(&prevNode.hash); ok {
			break
		}

//开始和到期时间基于中间块
//时间，现在计算一下。
		medianTime := prevNode.CalcPastMedianTime()

//
//已联系。
		if uint64(medianTime.Unix()) < checker.BeginTime() {
			cache.Update(&prevNode.hash, ThresholdDefined)
			break
		}

//将此节点添加到需要状态的节点列表中
//计算并缓存。
		neededStates = append(neededStates, prevNode)

//获取上一个块的最后一个块的祖先
//确认窗口。
		prevNode = prevNode.RelativeAncestor(confirmationWindow)
	}

//从最新确认的阈值状态开始
//具有缓存状态的窗口。
	state := ThresholdDefined
	if prevNode != nil {
		var ok bool
		state, ok = cache.Lookup(&prevNode.hash)
		if !ok {
			return ThresholdFailed, AssertError(fmt.Sprintf(
				"thresholdState: cache lookup failed for %v",
				prevNode.hash))
		}
	}

//因为每个阈值状态都依赖于前一个阈值状态
//窗口，从最旧的未知窗口开始迭代。
	for neededNum := len(neededStates) - 1; neededNum >= 0; neededNum-- {
		prevNode := neededStates[neededNum]

		switch state {
		case ThresholdDefined:
//
//在被接受和锁定之前。
			medianTime := prevNode.CalcPastMedianTime()
			medianTimeUnix := uint64(medianTime.Unix())
			if medianTimeUnix >= checker.EndTime() {
				state = ThresholdFailed
				break
			}

//规则的状态将移动到启动状态
//一旦到达开始时间（但还没有
//已过期）。
			if medianTimeUnix >= checker.BeginTime() {
				state = ThresholdStarted
			}

		case ThresholdStarted:
//如果规则更改过期，则部署失败
//在被接受和锁定之前。
			medianTime := prevNode.CalcPastMedianTime()
			if uint64(medianTime.Unix()) >= checker.EndTime() {
				state = ThresholdFailed
				break
			}

//在这一点上，规则的改变仍在投票中
//
//用于计算其中所有投票数的确认窗口。
			var count uint32
			countNode := prevNode
			for i := int32(0); i < confirmationWindow; i++ {
				condition, err := checker.Condition(countNode)
				if err != nil {
					return ThresholdFailed, err
				}
				if condition {
					count++
				}

//获取上一个块节点。
				countNode = countNode.parent
			}

//如果
//为规则更改投票的期间满足
//激活阈值。
			if count >= checker.RuleChangeActivationThreshold() {
				state = ThresholdLockedIn
			}

		case ThresholdLockedIn:
//当新规则的前一个状态变为活动状态时
//被锁在里面。
			state = ThresholdActive

//如果以前的状态为活动或失败，则从以下时间起不执行任何操作
//它们都是终态。
		case ThresholdActive:
		case ThresholdFailed:
		}

//更新缓存以避免重新计算
//未来。
		cache.Update(&prevNode.hash, state)
	}

	return state, nil
}

//threshold state返回给定的当前规则更改阈值状态
//当前最佳链结束后块的部署ID。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) ThresholdState(deploymentID uint32) (ThresholdState, error) {
	b.chainLock.Lock()
	state, err := b.deploymentState(b.bestChain.Tip(), deploymentID)
	b.chainLock.Unlock()

	return state, err
}

//如果目标DeploymentID处于活动状态，则IsDeploymentActive返回true，并且
//否则为假。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) IsDeploymentActive(deploymentID uint32) (bool, error) {
	b.chainLock.Lock()
	state, err := b.deploymentState(b.bestChain.Tip(), deploymentID)
	b.chainLock.Unlock()
	if err != nil {
		return false, err
	}

	return state == ThresholdActive, nil
}

//DeploymentState返回给定的当前规则更改阈值
//部署程序。从块的角度评估阈值
//作为此方法的第一个参数传入的节点。
//
//重要的是要注意，正如变量名所示，这个函数
//需要块节点位于部署状态为的块之前
//渴望的。换句话说，返回的部署状态是针对块的
//在经过的节点之后。
//
//必须在保持链状态锁的情况下调用此函数（用于写入）。
func (b *BlockChain) deploymentState(prevNode *blockNode, deploymentID uint32) (ThresholdState, error) {
	if deploymentID > uint32(len(b.chainParams.Deployments)) {
		return ThresholdFailed, DeploymentError(deploymentID)
	}

	deployment := &b.chainParams.Deployments[deploymentID]
	checker := deploymentChecker{deployment: deployment, chain: b}
	cache := &b.deploymentCaches[deploymentID]

	return b.thresholdState(prevNode, checker, cache)
}

//initThresholdCaches初始化每个警告的阈值状态缓存
//位和定义的部署，如果链是当前的，则提供警告。
//WarnunKnownversions和WarnunKnownRuleActivations功能。
func (b *BlockChain) initThresholdCaches() error {
//通过计算
//它们各自的阈值状态。这将确保缓存
//由于以下原因需要重新计算的填充状态和任何状态
//定义更改现在完成。
	prevNode := b.bestChain.Tip().parent
	for bit := uint32(0); bit < vbNumBits; bit++ {
		checker := bitConditionChecker{bit: bit, chain: b}
		cache := &b.warningCaches[bit]
		_, err := b.thresholdState(prevNode, checker, cache)
		if err != nil {
			return err
		}
	}
	for id := 0; id < len(b.chainParams.Deployments); id++ {
		deployment := &b.chainParams.Deployments[id]
		cache := &b.deploymentCaches[id]
		checker := deploymentChecker{deployment: deployment, chain: b}
		_, err := b.thresholdState(prevNode, checker, cache)
		if err != nil {
			return err
		}
	}

//在链
//电流。
	if b.isCurrent() {
//如果最后一个块的百分比足够高，则发出警告
//意外的版本。
		bestNode := b.bestChain.Tip()
		if err := b.warnUnknownVersions(bestNode); err != nil {
			return err
		}

//如果任何未知的新规则即将激活或
//已经激活。
		if err := b.warnUnknownRuleActivations(bestNode); err != nil {
			return err
		}
	}

	return nil
}
