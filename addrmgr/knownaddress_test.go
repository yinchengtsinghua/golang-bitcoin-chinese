
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2015 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package addrmgr_test

import (
	"math"
	"testing"
	"time"

	"github.com/btcsuite/btcd/addrmgr"
	"github.com/btcsuite/btcd/wire"
)

func TestChance(t *testing.T) {
	now := time.Unix(time.Now().Unix(), 0)
	var tests = []struct {
		addr     *addrmgr.KnownAddress
		expected float64
	}{
		{
//测试正常情况
			addrmgr.TstNewKnownAddress(&wire.NetAddress{Timestamp: now.Add(-35 * time.Second)},
				0, time.Now().Add(-30*time.Minute), time.Now(), false, 0),
			1.0,
		}, {
//LastSeen<0的测试用例
			addrmgr.TstNewKnownAddress(&wire.NetAddress{Timestamp: now.Add(20 * time.Second)},
				0, time.Now().Add(-30*time.Minute), time.Now(), false, 0),
			1.0,
		}, {
//上次尝试小于0的测试用例
			addrmgr.TstNewKnownAddress(&wire.NetAddress{Timestamp: now.Add(-35 * time.Second)},
				0, time.Now().Add(30*time.Minute), time.Now(), false, 0),
			1.0 * .01,
		}, {
//上次尝试时间小于10分钟的测试用例
			addrmgr.TstNewKnownAddress(&wire.NetAddress{Timestamp: now.Add(-35 * time.Second)},
				0, time.Now().Add(-5*time.Minute), time.Now(), false, 0),
			1.0 * .01,
		}, {
//多次失败的测试用例。
			addrmgr.TstNewKnownAddress(&wire.NetAddress{Timestamp: now.Add(-35 * time.Second)},
				2, time.Now().Add(-30*time.Minute), time.Now(), false, 0),
			1 / 1.5 / 1.5,
		},
	}

	err := .0001
	for i, test := range tests {
		chance := addrmgr.TstKnownAddressChance(test.addr)
		if math.Abs(test.expected-chance) >= err {
			t.Errorf("case %d: got %f, expected %f", i, chance, test.expected)
		}
	}
}

func TestIsBad(t *testing.T) {
	now := time.Unix(time.Now().Unix(), 0)
	future := now.Add(35 * time.Minute)
	monthOld := now.Add(-43 * time.Hour * 24)
	secondsOld := now.Add(-2 * time.Second)
	minutesOld := now.Add(-27 * time.Minute)
	hoursOld := now.Add(-5 * time.Hour)
	zeroTime := time.Time{}

	futureNa := &wire.NetAddress{Timestamp: future}
	minutesOldNa := &wire.NetAddress{Timestamp: minutesOld}
	monthOldNa := &wire.NetAddress{Timestamp: monthOld}
	currentNa := &wire.NetAddress{Timestamp: secondsOld}

//在最后一分钟尝试的测试地址。
	if addrmgr.TstKnownAddressIsBad(addrmgr.TstNewKnownAddress(futureNa, 3, secondsOld, zeroTime, false, 0)) {
		t.Errorf("test case 1: addresses that have been tried in the last minute are not bad.")
	}
	if addrmgr.TstKnownAddressIsBad(addrmgr.TstNewKnownAddress(monthOldNa, 3, secondsOld, zeroTime, false, 0)) {
		t.Errorf("test case 2: addresses that have been tried in the last minute are not bad.")
	}
	if addrmgr.TstKnownAddressIsBad(addrmgr.TstNewKnownAddress(currentNa, 3, secondsOld, zeroTime, false, 0)) {
		t.Errorf("test case 3: addresses that have been tried in the last minute are not bad.")
	}
	if addrmgr.TstKnownAddressIsBad(addrmgr.TstNewKnownAddress(currentNa, 3, secondsOld, monthOld, true, 0)) {
		t.Errorf("test case 4: addresses that have been tried in the last minute are not bad.")
	}
	if addrmgr.TstKnownAddressIsBad(addrmgr.TstNewKnownAddress(currentNa, 2, secondsOld, secondsOld, true, 0)) {
		t.Errorf("test case 5: addresses that have been tried in the last minute are not bad.")
	}

//声称来自未来的测试地址。
	if !addrmgr.TstKnownAddressIsBad(addrmgr.TstNewKnownAddress(futureNa, 0, minutesOld, hoursOld, true, 0)) {
		t.Errorf("test case 6: addresses that claim to be from the future are bad.")
	}

//一个多月没有看到的测试地址。
	if !addrmgr.TstKnownAddressIsBad(addrmgr.TstNewKnownAddress(monthOldNa, 0, minutesOld, hoursOld, true, 0)) {
		t.Errorf("test case 7: addresses more than a month old are bad.")
	}

//它至少失败了三次，从未成功过。
	if !addrmgr.TstKnownAddressIsBad(addrmgr.TstNewKnownAddress(minutesOldNa, 3, minutesOld, zeroTime, true, 0)) {
		t.Errorf("test case 8: addresses that have never succeeded are bad.")
	}

//上星期它失败了十次
	if !addrmgr.TstKnownAddressIsBad(addrmgr.TstNewKnownAddress(minutesOldNa, 10, minutesOld, monthOld, true, 0)) {
		t.Errorf("test case 9: addresses that have not succeeded in too long are bad.")
	}

//测试一个应该工作的地址。
	if addrmgr.TstKnownAddressIsBad(addrmgr.TstNewKnownAddress(minutesOldNa, 2, minutesOld, hoursOld, true, 0)) {
		t.Errorf("test case 10: This should be a valid address.")
	}
}
