
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

package connmgr

import (
	"fmt"
	"math"
	"sync"
	"time"
)

const (
//半衰期定义瞬变部分的时间（以秒为单位）
//禁令的分数下降到原来的一半。
	Halflife = 60

//lambda是衰减常数。
	lambda = math.Ln2 / Halflife

//寿命定义了禁令暂时性部分的最大年龄。
//分数被视为非零分（秒）。
	Lifetime = 1800

//PrecomputedLen定义了
//应在初始化时预计算。
	precomputedLen = 64
)

//PrecomputedFactor存储第一个
//“PrecomputedLen”秒，从t==0开始。
var precomputedFactor [precomputedLen]float64

//init预计算衰减因子。
func init() {
	for i := range precomputedFactor {
		precomputedFactor[i] = math.Exp(-1.0 * float64(i) * lambda)
	}
}

//decay factor返回t秒时的衰减因子，使用预先计算的值
//如果可用，或根据需要计算系数。
func decayFactor(t int64) float64 {
	if t < precomputedLen {
		return precomputedFactor[t]
	}
	return math.Exp(-1.0 * float64(t) * lambda)
}

//dynamicbanscore提供由持久性和
//腐烂的成分。持久的分数可以用来创建简单的
//类似于其他比特币节点的附加禁止政策
//实施。
//
//衰减的分数可以创建处理
//行为不端的对等端（尤其是应用层DoS攻击）优雅地
//通过切断和禁止同龄人尝试各种洪水。
//dynamicbanscore允许这两种方法串联使用。
//
//零值：类型dynamicbanscore的值立即可以在
//宣言。
type DynamicBanScore struct {
	lastUnix   int64
	transient  float64
	persistent uint32
	mtx        sync.Mutex
}

//字符串将BAN分数返回为人类可读的字符串。
func (s *DynamicBanScore) String() string {
	s.mtx.Lock()
	r := fmt.Sprintf("persistent %v + transient %v at %v = %v as of now",
		s.persistent, s.transient, s.lastUnix, s.Int())
	s.mtx.Unlock()
	return r
}

//int返回当前禁止分数，持久和衰退的总和
//分数。
//
//此函数对于并发访问是安全的。
func (s *DynamicBanScore) Int() uint32 {
	s.mtx.Lock()
	r := s.int(time.Now())
	s.mtx.Unlock()
	return r
}

//增加值会增加持续和衰减分数
//作为参数传递。返回结果分数。
//
//此函数对于并发访问是安全的。
func (s *DynamicBanScore) Increase(persistent, transient uint32) uint32 {
	s.mtx.Lock()
	r := s.increase(persistent, transient, time.Now())
	s.mtx.Unlock()
	return r
}

//重置将持续和衰减分数都设置为零。
//
//此函数对于并发访问是安全的。
func (s *DynamicBanScore) Reset() {
	s.mtx.Lock()
	s.persistent = 0
	s.transient = 0
	s.lastUnix = 0
	s.mtx.Unlock()
}

//int返回BAN分数，即在
//给定点。
//
//此函数对于并发访问不安全。它的用途是
//内部和测试期间。
func (s *DynamicBanScore) int(t time.Time) uint32 {
	dt := t.Unix() - s.lastUnix
	if s.transient < 1 || dt < 0 || Lifetime < dt {
		return s.persistent
	}
	return s.persistent + uint32(s.transient*decayFactor(dt))
}

//增加值会增加持续、衰减或两个分数的值
//作为参数传递。结果分数的计算方式与
//在第三个参数表示的时间点执行。这个
//返回结果分数。
//
//此函数对于并发访问不安全。
func (s *DynamicBanScore) increase(persistent, transient uint32, t time.Time) uint32 {
	s.persistent += persistent
	tu := t.Unix()
	dt := tu - s.lastUnix

	if transient > 0 {
		if Lifetime < dt {
			s.transient = 0
		} else if s.transient > 1 && dt > 0 {
			s.transient *= decayFactor(dt)
		}
		s.transient += float64(transient)
		s.lastUnix = tu
	}
	return s.persistent + uint32(s.transient)
}
