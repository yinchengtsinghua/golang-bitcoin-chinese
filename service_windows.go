
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
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/btcsuite/winsvc/eventlog"
	"github.com/btcsuite/winsvc/mgr"
	"github.com/btcsuite/winsvc/svc"
)

const (
//svcname是btcd服务的名称。
	svcName = "btcdsvc"

//svcdisplayname是将在Windows中显示的服务名称
//服务列表。不是svcname是使用的“real”名称
//控制服务。这仅用于显示目的。
	svcDisplayName = "Btcd Service"

//svcdesc是服务的描述。
	svcDesc = "Downloads and stays synchronized with the bitcoin block " +
		"chain and provides chain services to applications."
)

//ELOG用于向Windows事件日志发送消息。
var elog *eventlog.Log

//当主服务器
//已启动到Windows事件日志。
func logServiceStartOfDay(srvr *server) {
	var message string
	message += fmt.Sprintf("Version %s\n", version())
	message += fmt.Sprintf("Configuration directory: %s\n", defaultHomeDir)
	message += fmt.Sprintf("Configuration file: %s\n", cfg.ConfigFile)
	message += fmt.Sprintf("Data directory: %s\n", cfg.DataDir)

	elog.Info(1, message)
}

//btcdservice包含处理所有服务的主服务处理程序
//更新并启动BTCDMAIN。
type btcdService struct{}

//execute是winsvc包在接收时调用的主要入口点。
//来自Windows服务控制管理器的信息。它启动了
//长期运行的btcmain（btcd真正的肉）处理服务
//更改请求，并将更改通知服务控制管理器。
func (s *btcdService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
//服务启动挂起。
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

//在单独的goroutine中启动btcmain，以便服务可以启动
//迅速地。关闭（以及潜在错误）通过
//多尼琴ServerChan会收到一次主服务器实例的通知
//它是启动的，因此可以优雅地停止。
	doneChan := make(chan error)
	serverChan := make(chan *server)
	go func() {
		err := btcdMain(serverChan)
		doneChan <- err
	}()

//服务现在已启动。
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	var mainServer *server
loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus

			case svc.Stop, svc.Shutdown:
//服务停止挂起。不接受任何
//挂起时有更多命令。
				changes <- svc.Status{State: svc.StopPending}

//发出退出主功能的信号。
				shutdownRequestChannel <- struct{}{}

			default:
				elog.Error(1, fmt.Sprintf("Unexpected control "+
					"request #%d.", c))
			}

		case srvr := <-serverChan:
			mainServer = srvr
			logServiceStartOfDay(mainServer)

		case err := <-doneChan:
			if err != nil {
				elog.Error(1, err.Error())
			}
			break loop
		}
	}

//服务现在已停止。
	changes <- svc.Status{State: svc.Stopped}
	return false, 0
}

//InstallService尝试安装BTCD服务。通常这应该
//由MSI安装程序完成，但此处提供，因为它可能有用
//为了发展。
func installService() error {
//获取当前可执行文件的路径。这是需要的，因为
//args[0]可能会因应用程序的启动方式而有所不同。
//例如，在cmd.exe下，它将仅是应用程序的名称。
//没有路径或扩展，但在mingw下它将是完整的
//包含扩展名的路径。
	exePath, err := filepath.Abs(os.Args[0])
	if err != nil {
		return err
	}
	if filepath.Ext(exePath) == "" {
		exePath += ".exe"
	}

//连接到Windows服务管理器。
	serviceManager, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer serviceManager.Disconnect()

//确保服务不存在。
	service, err := serviceManager.OpenService(svcName)
	if err == nil {
		service.Close()
		return fmt.Errorf("service %s already exists", svcName)
	}

//安装服务。
	service, err = serviceManager.CreateService(svcName, exePath, mgr.Config{
		DisplayName: svcDisplayName,
		Description: svcDesc,
	})
	if err != nil {
		return err
	}
	defer service.Close()

//使用标准“标准”窗口支持事件日志中的事件
//eventcreate.exe消息文件。这允许轻松记录自定义
//而不需要创建自己的消息目录。
	eventlog.Remove(svcName)
	eventsSupported := uint32(eventlog.Error | eventlog.Warning | eventlog.Info)
	return eventlog.InstallAsEventCreate(svcName, eventsSupported)
}

//removeService尝试卸载BTCD服务。通常这应该
//由MSI卸载程序完成，但此处提供，因为它可以
//有助于发展。不是故意不删除事件日志条目
//因为它会使任何现有的事件日志消息失效。
func removeService() error {
//连接到Windows服务管理器。
	serviceManager, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer serviceManager.Disconnect()

//确保服务存在。
	service, err := serviceManager.OpenService(svcName)
	if err != nil {
		return fmt.Errorf("service %s is not installed", svcName)
	}
	defer service.Close()

//移除服务。
	return service.Delete()
}

//StartService尝试启动BTCD服务。
func startService() error {
//连接到Windows服务管理器。
	serviceManager, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer serviceManager.Disconnect()

	service, err := serviceManager.OpenService(svcName)
	if err != nil {
		return fmt.Errorf("could not access service: %v", err)
	}
	defer service.Close()

	err = service.Start(os.Args)
	if err != nil {
		return fmt.Errorf("could not start service: %v", err)
	}

	return nil
}

//ControlService允许更改服务状态的命令。它
//同时等待最多10秒，让服务更改为通过的
//状态。
func controlService(c svc.Cmd, to svc.State) error {
//连接到Windows服务管理器。
	serviceManager, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer serviceManager.Disconnect()

	service, err := serviceManager.OpenService(svcName)
	if err != nil {
		return fmt.Errorf("could not access service: %v", err)
	}
	defer service.Close()

	status, err := service.Control(c)
	if err != nil {
		return fmt.Errorf("could not send control=%d: %v", c, err)
	}

//发送控制消息。
	timeout := time.Now().Add(10 * time.Second)
	for status.State != to {
		if timeout.Before(time.Now()) {
			return fmt.Errorf("timeout waiting for service to go "+
				"to state=%d", to)
		}
		time.Sleep(300 * time.Millisecond)
		status, err = service.Query()
		if err != nil {
			return fmt.Errorf("could not retrieve service "+
				"status: %v", err)
		}
	}

	return nil
}

//performservicecommand尝试运行受支持的服务命令之一
//通过服务命令标志在命令行上提供。适当的
//如果指定的命令无效，则返回错误。
func performServiceCommand(command string) error {
	var err error
	switch command {
	case "install":
		err = installService()

	case "remove":
		err = removeService()

	case "start":
		err = startService()

	case "stop":
		err = controlService(svc.Stop, svc.Stopped)

	default:
		err = fmt.Errorf("invalid service command [%s]", command)
	}

	return err
}

//ServiceMain检查是否将我们作为服务调用，如果是，则使用
//用于启动长时间运行的服务器的服务控制管理器。旗是
//返回给调用方，以便应用程序可以确定是否退出（当
//作为服务运行）或以正常交互模式启动。
func serviceMain() (bool, error) {
//如果我们以交互方式运行（或不能以交互方式运行），则不作为服务运行
//因错误而确定）。
	isInteractive, err := svc.IsAnInteractiveSession()
	if err != nil {
		return false, err
	}
	if isInteractive {
		return false, nil
	}

	elog, err = eventlog.Open(svcName)
	if err != nil {
		return false, err
	}
	defer elog.Close()

	err = svc.Run(svcName, &btcdService{})
	if err != nil {
		elog.Error(1, fmt.Sprintf("Service start failed: %v", err))
		return true, err
	}

	return true, nil
}

//将特定于Windows的函数设置为实际函数。
func init() {
	runServiceCommand = performServiceCommand
	winServiceMain = serviceMain
}
