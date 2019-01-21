
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

//InterruptChannel用于接收SIGINT（ctrl+c）信号。
var interruptChannel chan os.Signal

//addhandlerChannel用于将中断处理程序添加到处理程序列表中
//在SIGINT（ctrl+c）信号上调用。
var addHandlerChannel = make(chan func())

//MainInterruptHandler在
//InterruptChannel并相应地调用注册的InterruptCallbacks。
//它还监听回调注册。它必须像野人一样运作。
func mainInterruptHandler() {
//InterruptCallbacks是当
//接收到sigint（ctrl+c）。
	var interruptCallbacks []func()

//IsShutdown是一个标志，用于指示是否
//关闭信号已经收到，因此任何未来
//尝试添加新的中断处理程序应调用它们
//立即。
	var isShutdown bool

	for {
		select {
		case <-interruptChannel:
//忽略多个停机信号。
			if isShutdown {
				log.Infof("Received SIGINT (Ctrl+C).  " +
					"Already shutting down...")
				continue
			}

			isShutdown = true
			log.Infof("Received SIGINT (Ctrl+C).  Shutting down...")

//按后进先出顺序运行处理程序。
			for i := range interruptCallbacks {
				idx := len(interruptCallbacks) - 1 - i
				callback := interruptCallbacks[idx]
				callback()
			}

//向主Goroutine发出关闭信号。
			go func() {
				shutdownChannel <- nil
			}()

		case handler := <-addHandlerChannel:
//停机信号已经收到，所以
//只需立即调用和新的处理程序。
			if isShutdown {
				handler()
			}

			interruptCallbacks = append(interruptCallbacks, handler)
		}
	}
}

//当sigint（ctrl+c）为
//收到。
func addInterruptHandler(handler func()) {
//创建通道并启动调用
//所有其他回调和退出（如果尚未完成）。
	if interruptChannel == nil {
		interruptChannel = make(chan os.Signal, 1)
		signal.Notify(interruptChannel, os.Interrupt)
		go mainInterruptHandler()
	}

	addHandlerChannel <- handler
}
