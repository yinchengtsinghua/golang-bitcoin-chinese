
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package blockchain

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
)

//TestNotifications确保对事件触发通知回调。
func TestNotifications(t *testing.T) {
	blocks, err := loadBlocks("blk_0_to_4.dat.bz2")
	if err != nil {
		t.Fatalf("Error loading file: %v\n", err)
	}

//创建一个新的数据库和链实例来运行测试。
	chain, teardownFunc, err := chainSetup("notifications",
		&chaincfg.MainNetParams)
	if err != nil {
		t.Fatalf("Failed to setup chain instance: %v", err)
	}
	defer teardownFunc()

	notificationCount := 0
	callback := func(notification *Notification) {
		if notification.Type == NTBlockAccepted {
			notificationCount++
		}
	}

//
//时代。
	const numSubscribers = 3
	for i := 0; i < numSubscribers; i++ {
		chain.Subscribe(callback)
	}

	_, _, err = chain.ProcessBlock(blocks[1], BFNone)
	if err != nil {
		t.Fatalf("ProcessBlock fail on block 1: %v\n", err)
	}

	if notificationCount != numSubscribers {
		t.Fatalf("Expected notification callback to be executed %d "+
			"times, found %d", numSubscribers, notificationCount)
	}
}
