
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

package blockchain

import (
	"math"
	"sort"
	"sync"
	"time"
)

const (
//maxallowedoffsetseconds是其中一个的最大秒数
//
//如果网络超出此范围，则不会应用任何偏移。
maxAllowedOffsetSecs = 70 * 60 //

//SimilarTimeSecs是从
//
//
similarTimeSecs = 5 * 60 //5分钟
)

var (
//
//
//测试代码可以修改它。
	maxMedianTimeEntries = 200
)

//
//用于确定中间时间，然后将其用作对局部时间的偏移量
//时钟。
type MedianTimeSource interface {
//AdjustedTime返回由中间时间调整的当前时间
//由addTimesample添加的时间样本计算的偏移量。
	AdjustedTime() time.Time

//
//添加样品的中间时间。
	AddTimeSample(id string, timeVal time.Time)

//
//根据addTimeData添加的时间样本的中位数。
	Offset() time.Duration
}

//Int64Sorter实现Sort.Interface以允许64位整数的切片
//分类。
type int64Sorter []int64

//len返回切片中64位整数的数目。它是
//Sort.Interface实现。
func (s int64Sorter) Len() int {
	return len(s)
}

//交换在传递的索引处交换64位整数。它是
//Sort.Interface实现。
func (s int64Sorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

//less返回索引为的64位整数是否应在
//
//实施。
func (s int64Sorter) Less(i, j int) bool {
	return s[i] < s[j]
}

//Mediantime提供了MediantimeSource接口的实现。
//
//比特币核心的时间偏移机制。这是必要的，因为它是
//
type medianTime struct {
	mtx                sync.Mutex
	knownIDs           map[string]struct{}
	offsets            []int64
	offsetSecs         int64
	invalidTimeChecked bool
}

//确保mediantime类型实现mediantimesource接口。
var _ MedianTimeSource = (*medianTime)(nil)

//AdjustedTime返回由中间时间偏移调整的当前时间，如下所示
//由addTimesample添加的时间样本计算得出。
//
//
//MediantimeSource接口实现。
func (m *medianTime) AdjustedTime() time.Time {
	m.mtx.Lock()
	defer m.mtx.Unlock()

//将调整时间限制为1秒精度。
	now := time.Unix(time.Now().Unix(), 0)
	return now.Add(time.Duration(m.offsetSecs) * time.Second)
}

//addTimesample添加一个时间样本，用于确定中间值
//添加样品的时间。
//
//
//MediantimeSource接口实现。
func (m *medianTime) AddTimeSample(sourceID string, timeVal time.Time) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

//不要从同一来源添加时间数据。
	if _, exists := m.knownIDs[sourceID]; exists {
		return
	}
	m.knownIDs[sourceID] = struct{}{}

//将提供的偏移量截断为秒并将其附加到切片
//
//将最旧的条目替换为新条目一次最大数目
//
	now := time.Unix(time.Now().Unix(), 0)
	offsetSecs := int64(timeVal.Sub(now).Seconds())
	numOffsets := len(m.offsets)
	if numOffsets == maxMedianTimeEntries && maxMedianTimeEntries > 0 {
		m.offsets = m.offsets[1:]
		numOffsets--
	}
	m.offsets = append(m.offsets, offsetSecs)
	numOffsets++

//对偏移进行排序，以便在以后需要时获得中间值。
	sortedOffsets := make([]int64, numOffsets)
	copy(sortedOffsets, m.offsets)
	sort.Sort(int64Sorter(sortedOffsets))

	offsetDuration := time.Duration(offsetSecs) * time.Second
	log.Debugf("Added time sample of %v (total: %v)", offsetDuration,
		numOffsets)

//
//比特币核心的马车行为，因为中间时间被用于
//共识规则。
//
//尤其是，只有当条目数
//
//一旦最大条目数为
//达到。

//只有当有足够的偏移量和
//
//因此，当这些条件不满足时，没有什么可做的。
	if numOffsets < 5 || numOffsets&0x01 != 1 {
		return
	}

//此时列表中的偏移数是奇数，因此
//排序偏移的中间值是中间值。
	median := sortedOffsets[numOffsets/2]

//
//偏移范围。
	if math.Abs(float64(median)) < maxAllowedOffsetSecs {
		m.offsetSecs = median
	} else {
//所有添加的时间数据的中间偏移量大于
//允许的最大偏移量，因此不要使用偏移量。这个
//有效地限制了本地时钟的倾斜程度。
		m.offsetSecs = 0

		if !m.invalidTimeChecked {
			m.invalidTimeChecked = true

//查找时间样本是否有接近的时间
//到当地时间。
			var remoteHasCloseTime bool
			for _, offset := range sortedOffsets {
				if math.Abs(float64(offset)) < similarTimeSecs {
					remoteHasCloseTime = true
					break
				}
			}

//
			if !remoteHasCloseTime {
				log.Warnf("Please check your date and time " +
					"are correct!  btcd will not work " +
					"properly with an invalid time")
			}
		}
	}

	medianDuration := time.Duration(m.offsetSecs) * time.Second
	log.Debugf("New time offset: %v", medianDuration)
}

//offset返回根据
//
//
//
//MediantimeSource接口实现。
func (m *medianTime) Offset() time.Duration {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	return time.Duration(m.offsetSecs) * time.Second
}

//NewMediantime返回的并发安全实现的新实例
//MediantimeSource接口。返回的实现包含
//链共识规则中适当时间处理所需的规则
//期望从版本的时间戳字段添加时间示例
//从成功连接和协商的远程对等端接收的消息。
func NewMedianTime() MedianTimeSource {
	return &medianTime{
		knownIDs: make(map[string]struct{}),
		offsets:  make([]int64, 0, maxMedianTimeEntries),
	}
}
