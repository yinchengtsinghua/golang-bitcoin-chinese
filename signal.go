
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

package main

import (
	"os"
	"os/signal"
)

//ShutdownRequestChannel用于从
//使用与接收中断信号时相同的代码路径的子系统。
var shutdownRequestChannel = make(chan struct{})

//InterruptSignals定义要捕获的默认信号，以便
//停机。根据平台的不同，可以在初始化期间修改。
var interruptSignals = []os.Signal{os.Interrupt}

//InterruptListener侦听操作系统信号，如sigint（ctrl+c）和shutdown
//来自ShutdownRequestChannel的请求。它返回一个关闭的通道
//当接收到任一信号时。
func interruptListener() <-chan struct{} {
	c := make(chan struct{})
	go func() {
		interruptChannel := make(chan os.Signal, 1)
		signal.Notify(interruptChannel, interruptSignals...)

//监听初始停机信号并关闭返回的
//通知呼叫者的频道。
		select {
		case sig := <-interruptChannel:
			btcdLog.Infof("Received signal (%s).  Shutting down...",
				sig)

		case <-shutdownRequestChannel:
			btcdLog.Info("Shutdown requested.  Shutting down...")
		}
		close(c)

//倾听重复的信号并显示一条消息，以便用户
//知道关闭正在进行，但进程没有
//挂。
		for {
			select {
			case sig := <-interruptChannel:
				btcdLog.Infof("Received signal (%s).  Already "+
					"shutting down...", sig)

			case <-shutdownRequestChannel:
				btcdLog.Info("Shutdown requested.  Already " +
					"shutting down...")
			}
		}
	}()

	return c
}

//InterruptRequested在由返回的通道
//InterruptListener已关闭。这稍微简化了早期停机，因为
//调用方只能使用if语句而不是select。
func interruptRequested(interrupted <-chan struct{}) bool {
	select {
	case <-interrupted:
		return true
	default:
	}

	return false
}
