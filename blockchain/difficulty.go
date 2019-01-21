
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
	"math/big"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

var (
//bigone是1，表示为big.int。这里定义它是为了避免
//多次创建它的开销。
	bigOne = big.NewInt(1)

//一个lsh256是1左移位256位。这里的定义是为了避免
//多次创建它的开销。
	oneLsh256 = new(big.Int).Lsh(bigOne, 256)
)

//hash to big将chainhash.hash转换为可用于
//进行数学比较。
func HashToBig(hash *chainhash.Hash) *big.Int {
//哈希在小endian中，但大包希望字节在
//大尾数法，所以反转它们。
	buf := *hash
	blen := len(buf)
	for i := 0; i < blen/2; i++ {
		buf[i], buf[blen-1-i] = buf[blen-1-i], buf[i]
	}

	return new(big.Int).SetBytes(buf[:])
}

//CompactToBig将整数n的压缩表示形式转换为
//无符号32位数字。表示类似于IEEE754浮动
//点数。
//
//与IEEE754浮点一样，有三个基本组件：符号，
//指数和尾数。具体如下：
//
//*最重要的8位表示无符号的256指数基数。
//*位23（24位）表示符号位
//*最低有效23位表示尾数
//
//
//指数符号尾数
//————————————————————————————————————————————
//8位[31-24]1位[23]23位[22-00]
//————————————————————————————————————————————
//
//
//n=（-1^符号）*尾数*256^（指数-3）
//
//此压缩格式仅用于比特币编码无符号256位数字。
//
//
func CompactToBig(compact uint32) *big.Int {
//
	mantissa := compact & 0x007fffff
	isNegative := compact&0x00800000 != 0
	exponent := uint(compact >> 24)

//因为指数的基数是256，所以可以处理指数
//
//将指数视为字节数并移动尾数
//
//n=尾数*256^（指数-3）
	var bn *big.Int
	if exponent <= 3 {
		mantissa >>= 8 * (3 - exponent)
		bn = big.NewInt(int64(mantissa))
	} else {
		bn = big.NewInt(int64(mantissa))
		bn.Lsh(bn, 8*(exponent-3))
	}

//如果符号位设置为负数。
	if isNegative {
		bn = bn.Neg(bn)
	}

	return bn
}

//bigtocompact将整数n转换为紧凑的表示形式，使用
//
//精度，因此大于（2^23-1）的值编码最多
//数字的有效数字。有关详细信息，请参阅CompactToBig。
func BigToCompact(n *big.Int) uint32 {
//如果是零，就不需要做任何工作。
	if n.Sign() == 0 {
		return 0
	}

//因为指数的基数是256，所以可以处理指数
//作为字节数。所以，把数字右移或左移
//因此。这相当于：
//尾数=尾数/256^（指数-3）
	var mantissa uint32
	exponent := uint(len(n.Bytes()))
	if exponent <= 3 {
		mantissa = uint32(n.Bits()[0])
		mantissa <<= 8 * (3 - exponent)
	} else {
//使用副本可避免修改呼叫者的原始号码。
		tn := new(big.Int).Set(n)
		mantissa = uint32(tn.Rsh(tn, 8*(exponent-3)).Bits()[0])
	}

//当尾数已经设置了符号位时，数字也是
//大到可以容纳23位，所以将数字除以256
//并相应地增加指数。
	if mantissa&0x00800000 != 0 {
		mantissa >>= 8
		exponent++
	}

//将指数、符号位和尾数打包成无符号32位
//int并返回它。
	compact := uint32(exponent<<24) | mantissa
	if n.Sign() < 0 {
		compact |= 0x00800000
	}
	return compact
}

//CalcWork从难度位计算工时值。比特币增加
//通过减少数据块的
//生成的哈希必须小于。此难度目标存储在每个
//使用文档中描述的压缩表示形式阻止头段
//用于压实机。通过选择具有
//工作证明最多（难度最高）。因为目标难度较低
//价值等于较高的实际难度，即
//累积必须与难度成反比。同时，为了避免
//可能被零和非常小的浮点数除
//结果将分母加1，分子乘以2^256。
func CalcWork(bits uint32) *big.Int {
//如果传递的难度位表示
//一个负数。注意，这在实践中不应该发生在
//块，但无效的块可能会触发它。
	difficultyNum := CompactToBig(bits)
	if difficultyNum.Sign() <= 0 {
		return big.NewInt(0)
	}

//（1<<256）/（困难数+1）
	denominator := new(big.Int).Add(difficultyNum, bigOne)
	return new(big.Int).Div(oneLsh256, denominator)
}

//计算难度计算块
//可以给出起始难度位和持续时间。主要用于
//验证一个区块所要求的工作证明与
//
func (b *BlockChain) calcEasiestDifficulty(bits uint32, duration time.Duration) uint32 {
//转换下面计算中使用的类型。
	durationVal := int64(duration / time.Second)
	adjustmentFactor := big.NewInt(b.chainParams.RetargetAdjustmentFactor)

//测试网络规则允许在更多的
//生成块所需时间的两倍以上
//逝去。
	if b.chainParams.ReduceMinDifficulty {
		reductionTime := int64(b.chainParams.MinDiffReductionTime /
			time.Second)
		if durationVal > reductionTime {
			return b.chainParams.PowLimitBits
		}
	}

//因为简单的困难等同于更高的数字，最简单的
//给定时间内的难度是给定时间内可能的最大值。
//持续时间和开始难度的重定目标数
//乘以最大调整系数。
	newTarget := CompactToBig(bits)
	for durationVal > 0 && newTarget.Cmp(b.chainParams.PowLimit) < 0 {
		newTarget.Mul(newTarget, adjustmentFactor)
		durationVal -= b.maxRetargetTimespan
	}

//将新值限制为工作限制的证明。
	if newTarget.Cmp(b.chainParams.PowLimit) > 0 {
		newTarget.Set(b.chainParams.PowLimit)
	}

	return BigToCompact(newTarget)
}

//
//没有应用特殊的testnet最低难度规则。
//
//必须在保持链状态锁的情况下调用此函数（用于写入）。
func (b *BlockChain) findPrevTestNetDifficulty(startNode *blockNode) uint32 {
//在链中向后搜索最后一个块
//适用的特殊规则。
	iterNode := startNode
	for iterNode != nil && iterNode.height%b.blocksPerRetarget != 0 &&
		iterNode.bits == b.chainParams.PowLimitBits {

		iterNode = iterNode.parent
	}

//返回发现的难度或最小难度（如果没有）
//找到适当的块。
	lastBits := b.chainParams.PowLimitBits
	if iterNode != nil {
		lastBits = iterNode.bits
	}
	return lastBits
}

//CalcNextRequiredDifficulty计算块所需的难度
//在通过前一个块节点后，根据难度重定目标规则。
//此函数与导出的CalcNextRequiredDifficulty的不同之处在于
//导出的版本使用当前最佳链作为前一个块节点
//当此函数接受任何块节点时。
func (b *BlockChain) calcNextRequiredDifficulty(lastNode *blockNode, newBlockTime time.Time) (uint32, error) {
//创世纪大厦
	if lastNode == nil {
		return b.chainParams.PowLimitBits, nil
	}

//返回上一个块的难度要求，如果此块
//不是很难重定目标间隔。
	if (lastNode.height+1)%b.blocksPerRetarget != 0 {
//对于支持它的网络，允许
//一旦过了太多的时间，
//挖掘一个街区。
		if b.chainParams.ReduceMinDifficulty {
//当超过期望值时返回最小难度
//未挖掘块所用的时间。
			reductionTime := int64(b.chainParams.MinDiffReductionTime /
				time.Second)
			allowMinTime := lastNode.timestamp + reductionTime
			if newBlockTime.Unix() > allowMinTime {
				return b.chainParams.PowLimitBits, nil
			}

//该区块是在预期时间内开采的，因此
//返回最后一个块的难度。
//没有应用特殊的最低难度规则。
			return b.findPrevTestNetDifficulty(lastNode), nil
		}

//对于主网络（或任何无法识别的网络），只需
//返回上一个块的难度要求。
		return lastNode.bits, nil
	}

//在上一个重定目标处获取块节点（TargetTimeSpan天
//块的价值）。
	firstNode := lastNode.RelativeAncestor(b.blocksPerRetarget - 1)
	if firstNode == nil {
		return 0, AssertError("unable to obtain previous retarget block")
	}

//限制以前可能发生的调整量
//困难。
	actualTimespan := lastNode.timestamp - firstNode.timestamp
	adjustedTimespan := actualTimespan
	if actualTimespan < b.minRetargetTimespan {
		adjustedTimespan = b.minRetargetTimespan
	} else if actualTimespan > b.maxRetargetTimespan {
		adjustedTimespan = b.maxRetargetTimespan
	}

//计算新目标难度为：
//当前难度*（调整的时间跨度/目标时间跨度）
//结果使用整数除法，这意味着它将稍微
//四舍五入比特币也使用整数除法来计算
//结果。
	oldTarget := CompactToBig(lastNode.bits)
	newTarget := new(big.Int).Mul(oldTarget, big.NewInt(adjustedTimespan))
	targetTimeSpan := int64(b.chainParams.TargetTimespan / time.Second)
	newTarget.Div(newTarget, big.NewInt(targetTimeSpan))

//将新值限制为工作限制的证明。
	if newTarget.Cmp(b.chainParams.PowLimit) > 0 {
		newTarget.Set(b.chainParams.PowLimit)
	}

//记录新目标难度并返回。新的目标日志记录是
//故意将位转换回数字而不是使用
//自转换为紧凑表示形式后，newTarget将丢失
//精度。
	newTargetBits := BigToCompact(newTarget)
	log.Debugf("Difficulty retarget at block height %d", lastNode.height+1)
	log.Debugf("Old target %08x (%064x)", lastNode.bits, oldTarget)
	log.Debugf("New target %08x (%064x)", newTargetBits, CompactToBig(newTargetBits))
	log.Debugf("Actual timespan %v, adjusted timespan %v, target timespan %v",
		time.Duration(actualTimespan)*time.Second,
		time.Duration(adjustedTimespan)*time.Second,
		b.chainParams.TargetTimespan)

	return newTargetBits, nil
}

//CalcNextRequiredDifficulty计算块所需的难度
//基于难度重定目标的当前最佳链结束后
//规则。
//
//此函数对于并发访问是安全的。
func (b *BlockChain) CalcNextRequiredDifficulty(timestamp time.Time) (uint32, error) {
	b.chainLock.Lock()
	difficulty, err := b.calcNextRequiredDifficulty(b.bestChain.Tip(), timestamp)
	b.chainLock.Unlock()
	return difficulty, err
}
