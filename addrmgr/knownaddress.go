
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2014 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package addrmgr

import (
	"time"

	"github.com/btcsuite/btcd/wire"
)

//knownAddress跟踪有关使用的已知网络地址的信息
//以确定地址的可行性。
type KnownAddress struct {
	na          *wire.NetAddress
	srcAddr     *wire.NetAddress
	attempts    int
	lastattempt time.Time
	lastsuccess time.Time
	tried       bool
refs        int //新存储桶的引用计数
}

//netaddress返回与
//已知地址。
func (ka *KnownAddress) NetAddress() *wire.NetAddress {
	return ka.na
}

//LastTeast返回上次尝试已知地址的时间。
func (ka *KnownAddress) LastAttempt() time.Time {
	return ka.lastattempt
}

//chance返回已知地址的选择概率。优先权
//取决于地址最近被看到的时间和最近被看到的时间
//尝试连接的次数和尝试连接失败的次数。
func (ka *KnownAddress) chance() float64 {
	now := time.Now()
	lastAttempt := now.Sub(ka.lastattempt)

	if lastAttempt < 0 {
		lastAttempt = 0
	}

	c := 1.0

//最近的尝试不太可能被重试。
	if lastAttempt < 10*time.Minute {
		c *= 0.01
	}

//失败的尝试会降低优先级。
	for i := ka.attempts; i > 0; i-- {
		c /= 1.5
	}

	return c
}

//如果上一次未尝试有关地址，则IsBad返回true
//会议记录并满足以下条件之一：
//1）声称来自未来
//2）一个多月没见了
//3）至少失败三次，从未成功
//4）上个星期失败了十次
//所有符合这些标准的地址都被认为是无用的，而不是
//值得一看。
func (ka *KnownAddress) isBad() bool {
	if ka.lastattempt.After(time.Now().Add(-1 * time.Minute)) {
		return false
	}

//来自未来？
	if ka.na.Timestamp.After(time.Now().Add(10 * time.Minute)) {
		return true
	}

//一个多月大？
	if ka.na.Timestamp.Before(time.Now().Add(-1 * numMissingDays * time.Hour * 24)) {
		return true
	}

//从未成功过？
	if ka.lastsuccess.IsZero() && ka.attempts >= numRetries {
		return true
	}

//太久没有成功？
	if !ka.lastsuccess.After(time.Now().Add(-1*minBadDays*time.Hour*24)) &&
		ka.attempts >= maxFailures {
		return true
	}

	return false
}
