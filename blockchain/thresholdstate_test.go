
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

package blockchain

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

//TestThresholdStateStringer测试
//阈值状态类型。
func TestThresholdStateStringer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   ThresholdState
		want string
	}{
		{ThresholdDefined, "ThresholdDefined"},
		{ThresholdStarted, "ThresholdStarted"},
		{ThresholdLockedIn, "ThresholdLockedIn"},
		{ThresholdActive, "ThresholdActive"},
		{ThresholdFailed, "ThresholdFailed"},
		{0xff, "Unknown ThresholdState (255)"},
	}

//
	if len(tests)-1 != int(numThresholdsStates) {
		t.Errorf("It appears a threshold statewas added without " +
			"adding an associated stringer test")
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		result := test.in.String()
		if result != test.want {
			t.Errorf("String #%d\n got: %s want: %s", i, result,
				test.want)
			continue
		}
	}
}

//testThresholdStateCache确保阈值状态缓存按预期工作
//包括添加条目、更新现有条目和刷新。
func TestThresholdStateCache(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		numEntries int
		state      ThresholdState
	}{
		{name: "2 entries defined", numEntries: 2, state: ThresholdDefined},
		{name: "7 entries started", numEntries: 7, state: ThresholdStarted},
		{name: "10 entries active", numEntries: 10, state: ThresholdActive},
		{name: "5 entries locked in", numEntries: 5, state: ThresholdLockedIn},
		{name: "3 entries failed", numEntries: 3, state: ThresholdFailed},
	}

nextTest:
	for _, test := range tests {
		cache := &newThresholdCaches(1)[0]
		for i := 0; i < test.numEntries; i++ {
			var hash chainhash.Hash
			hash[0] = uint8(i + 1)

//确保哈希在缓存中不可用。
			_, ok := cache.Lookup(&hash)
			if ok {
				t.Errorf("Lookup (%s): has entry for hash %v",
					test.name, hash)
				continue nextTest
			}

//
//可用，状态为预期值。
			cache.Update(&hash, test.state)
			state, ok := cache.Lookup(&hash)
			if !ok {
				t.Errorf("Lookup (%s): missing entry for hash "+
					"%v", test.name, hash)
				continue nextTest
			}
			if state != test.state {
				t.Errorf("Lookup (%s): state mismatch - got "+
					"%v, want %v", test.name, state,
					test.state)
				continue nextTest
			}

//确保添加具有相同状态的现有哈希
//不会破坏现有条目。
			cache.Update(&hash, test.state)
			state, ok = cache.Lookup(&hash)
			if !ok {
				t.Errorf("Lookup (%s): missing entry after "+
					"second add for hash %v", test.name,
					hash)
				continue nextTest
			}
			if state != test.state {
				t.Errorf("Lookup (%s): state mismatch after "+
					"second add - got %v, want %v",
					test.name, state, test.state)
				continue nextTest
			}

//确保添加具有不同状态的现有哈希
//更新现有条目。
			newState := ThresholdFailed
			if newState == test.state {
				newState = ThresholdStarted
			}
			cache.Update(&hash, newState)
			state, ok = cache.Lookup(&hash)
			if !ok {
				t.Errorf("Lookup (%s): missing entry after "+
					"state change for hash %v", test.name,
					hash)
				continue nextTest
			}
			if state != newState {
				t.Errorf("Lookup (%s): state mismatch after "+
					"state change - got %v, want %v",
					test.name, state, newState)
				continue nextTest
			}
		}
	}
}
