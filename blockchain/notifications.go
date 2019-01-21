
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package blockchain

import (
	"fmt"
)

//notificationType表示通知消息的类型。
type NotificationType int

//
//
type NotificationCallback func(*Notification)

//
const (
//
//
//
	NTBlockAccepted NotificationType = iota

//ntblockconnected表示关联的块已连接到
//主链。
	NTBlockConnected

//ntblockdisconnected表示关联的块已断开连接
//从主链。
	NTBlockDisconnected
)

//
//漂亮印刷体的名字。
var notificationTypeStrings = map[NotificationType]string{
	NTBlockAccepted:     "NTBlockAccepted",
	NTBlockConnected:    "NTBlockConnected",
	NTBlockDisconnected: "NTBlockDisconnected",
}

//字符串以可读形式返回notificationtype。
func (n NotificationType) String() string {
	if s, ok := notificationTypeStrings[n]; ok {
		return s
	}
	return fmt.Sprintf("Unknown Notification Type (%d)", int(n))
}

//
//在调用new期间提供的函数，由通知类型组成
//以及依赖于以下类型的关联数据：
//-接受ntblock:*bcutil.block
//-ntblockconnected:*bcutil.block
//-ntblockdisconnected:*bcutil.block
type Notification struct {
	Type NotificationType
	Data interface{}
}

//订阅区块链通知。注册要执行的回调
//当各种事件发生时。请参阅有关通知和
//notificationType获取有关通知类型和内容的详细信息。
func (b *BlockChain) Subscribe(callback NotificationCallback) {
	b.notificationsLock.Lock()
	b.notifications = append(b.notifications, callback)
	b.notificationsLock.Unlock()
}

//sendNotification发送带有传递类型和数据的通知，如果
//调用方通过在调用中提供回调函数来请求通知
//新的。
func (b *BlockChain) sendNotification(typ NotificationType, data interface{}) {
//生成并发送通知。
	n := Notification{Type: typ, Data: data}
	b.notificationsLock.RLock()
	for _, callback := range b.notifications {
		callback(&n)
	}
	b.notificationsLock.RUnlock()
}
