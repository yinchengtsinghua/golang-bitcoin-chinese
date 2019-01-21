
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
	"strconv"
	"testing"
	"time"
)

//
func TestMedianTime(t *testing.T) {
	tests := []struct {
		in         []int64
		wantOffset int64
		useDupID   bool
	}{
//样本不足必须导致偏移量为0。
		{in: []int64{1}, wantOffset: 0},
		{in: []int64{1, 2}, wantOffset: 0},
		{in: []int64{1, 2, 3}, wantOffset: 0},
		{in: []int64{1, 2, 3, 4}, wantOffset: 0},

//条目数不一。预期的偏移量仅为
//
		{in: []int64{-13, 57, -4, -23, -12}, wantOffset: -12},
		{in: []int64{55, -13, 61, -52, 39, 55}, wantOffset: 39},
		{in: []int64{-62, -58, -30, -62, 51, -30, 15}, wantOffset: -30},
		{in: []int64{29, -47, 39, 54, 42, 41, 8, -33}, wantOffset: 39},
		{in: []int64{37, 54, 9, -21, -56, -36, 5, -11, -39}, wantOffset: -11},
		{in: []int64{57, -28, 25, -39, 9, 63, -16, 19, -60, 25}, wantOffset: 9},
		{in: []int64{-5, -4, -3, -2, -1}, wantOffset: -3, useDupID: true},

//一旦最大条目数达到最大值，则停止更新偏移量。
//已联系。这实际上是比特币核心的漏洞，
//
//共识规则，必须反映出来。
		{in: []int64{-67, 67, -50, 24, 63, 17, 58, -14, 5, -32, -52}, wantOffset: 17},
		{in: []int64{-67, 67, -50, 24, 63, 17, 58, -14, 5, -32, -52, 45}, wantOffset: 17},
		{in: []int64{-67, 67, -50, 24, 63, 17, 58, -14, 5, -32, -52, 45, 4}, wantOffset: 17},

//离当地时间太远的偏移量应该
//被忽视。
		{in: []int64{-4201, 4202, -4203, 4204, -4205}, wantOffset: 0},

//在中间值偏移量较大的情况下进行练习
//大于允许的最大调整，但至少有一个
//与当前时间足够接近以避免
//触发关于无效本地时钟的警告。
		{in: []int64{4201, 4202, 4203, 4204, -299}, wantOffset: 0},
	}

//修改这些测试允许的最大中值时间项数。
	maxMedianTimeEntries = 10
	defer func() { maxMedianTimeEntries = 200 }()

	for i, test := range tests {
		filter := NewMedianTime()
		for j, offset := range test.in {
			id := strconv.Itoa(j)
			now := time.Unix(time.Now().Unix(), 0)
			tOffset := now.Add(time.Duration(offset) * time.Second)
			filter.AddTimeSample(id, tOffset)

//
			if test.useDupID {
//
//如果添加了副本，则会有所不同。
				tOffset = tOffset.Add(time.Duration(offset) *
					time.Second)
				filter.AddTimeSample(id, tOffset)
			}
		}

//因为时间是可能的。现在调用addTimesample
//
//第二，允许一个敷衍因素来补偿。
		gotOffset := filter.Offset()
		wantOffset := time.Duration(test.wantOffset) * time.Second
		wantOffset2 := time.Duration(test.wantOffset-1) * time.Second
		if gotOffset != wantOffset && gotOffset != wantOffset2 {
			t.Errorf("Offset #%d: unexpected offset -- got %v, "+
				"want %v or %v", i, gotOffset, wantOffset,
				wantOffset2)
			continue
		}

//因为时间是可能的。现在调用adjustedTime
//时间到了，现在打电话过来，测试一个就结束了
//第二，允许一个敷衍因素来补偿。
		adjustedTime := filter.AdjustedTime()
		now := time.Unix(time.Now().Unix(), 0)
		wantTime := now.Add(filter.Offset())
		wantTime2 := now.Add(filter.Offset() - time.Second)
		if !adjustedTime.Equal(wantTime) && !adjustedTime.Equal(wantTime2) {
			t.Errorf("AdjustedTime #%d: unexpected result -- got %v, "+
				"want %v or %v", i, adjustedTime, wantTime,
				wantTime2)
			continue
		}
	}
}
