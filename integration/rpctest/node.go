
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

package rpctest

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	rpc "github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcutil"
)

//nodeconfig包含启动btcd进程所需的所有参数和数据。
//并将RPC客户机连接到它。
type nodeConfig struct {
	rpcUser    string
	rpcPass    string
	listen     string
	rpcListen  string
	rpcConnect string
	dataDir    string
	logDir     string
	profile    string
	debugLevel string
	extra      []string
	prefix     string

	exe          string
	endpoint     string
	certFile     string
	keyFile      string
	certificates []byte
}

//newconfig返回具有所有默认值的newconfig。
func newConfig(prefix, certFile, keyFile string, extra []string) (*nodeConfig, error) {
	btcdPath, err := btcdExecutablePath()
	if err != nil {
		btcdPath = "btcd"
	}

	a := &nodeConfig{
		listen:    "127.0.0.1:18555",
		rpcListen: "127.0.0.1:18556",
		rpcUser:   "user",
		rpcPass:   "pass",
		extra:     extra,
		prefix:    prefix,
		exe:       btcdPath,
		endpoint:  "ws",
		certFile:  certFile,
		keyFile:   keyFile,
	}
	if err := a.setDefaults(); err != nil {
		return nil, err
	}
	return a, nil
}

//setdefaults设置配置的默认值。它还创建了
//临时数据和日志目录，必须通过调用
//清除（）。
func (n *nodeConfig) setDefaults() error {
	datadir, err := ioutil.TempDir("", n.prefix+"-data")
	if err != nil {
		return err
	}
	n.dataDir = datadir
	logdir, err := ioutil.TempDir("", n.prefix+"-logs")
	if err != nil {
		return err
	}
	n.logDir = logdir
	cert, err := ioutil.ReadFile(n.certFile)
	if err != nil {
		return err
	}
	n.certificates = cert
	return nil
}

//arguments返回用于启动BTCD的参数数组
//过程。
func (n *nodeConfig) arguments() []string {
	args := []string{}
	if n.rpcUser != "" {
//--RPCUSER
		args = append(args, fmt.Sprintf("--rpcuser=%s", n.rpcUser))
	}
	if n.rpcPass != "" {
//——RPCPASS
		args = append(args, fmt.Sprintf("--rpcpass=%s", n.rpcPass))
	}
	if n.listen != "" {
//--倾听
		args = append(args, fmt.Sprintf("--listen=%s", n.listen))
	}
	if n.rpcListen != "" {
//--RPCclipse
		args = append(args, fmt.Sprintf("--rpclisten=%s", n.rpcListen))
	}
	if n.rpcConnect != "" {
//-- RPCONTION
		args = append(args, fmt.Sprintf("--rpcconnect=%s", n.rpcConnect))
	}
//——RPCRET
	args = append(args, fmt.Sprintf("--rpccert=%s", n.certFile))
//——RPCKEY
	args = append(args, fmt.Sprintf("--rpckey=%s", n.keyFile))
	if n.dataDir != "" {
//——达达迪
		args = append(args, fmt.Sprintf("--datadir=%s", n.dataDir))
	}
	if n.logDir != "" {
//——罗吉尔
		args = append(args, fmt.Sprintf("--logdir=%s", n.logDir))
	}
	if n.profile != "" {
//——轮廓
		args = append(args, fmt.Sprintf("--profile=%s", n.profile))
	}
	if n.debugLevel != "" {
//--调试级别
		args = append(args, fmt.Sprintf("--debuglevel=%s", n.debugLevel))
	}
	args = append(args, n.extra...)
	return args
}

//命令返回将用于启动BTCD进程的exec.cmd。
func (n *nodeConfig) command() *exec.Cmd {
	return exec.Command(n.exe, n.arguments()...)
}

//rpcconconfig返回可用于连接的rpc连接配置
//到通过start（）启动的BTCD进程。
func (n *nodeConfig) rpcConnConfig() rpc.ConnConfig {
	return rpc.ConnConfig{
		Host:                 n.rpcListen,
		Endpoint:             n.endpoint,
		User:                 n.rpcUser,
		Pass:                 n.rpcPass,
		Certificates:         n.certificates,
		DisableAutoReconnect: true,
	}
}

//字符串返回此节点配置的字符串表示形式。
func (n *nodeConfig) String() string {
	return n.prefix
}

//清除删除tmp数据和日志目录。
func (n *nodeConfig) cleanup() error {
	dirs := []string{
		n.logDir,
		n.dataDir,
	}
	var err error
	for _, dir := range dirs {
		if err = os.RemoveAll(dir); err != nil {
			log.Printf("Cannot remove dir %s: %v", dir, err)
		}
	}
	return err
}

//节点包含配置、启动和管理
//BTCD工艺。
type node struct {
	config *nodeConfig

	cmd     *exec.Cmd
	pidFile string

	dataDir string
}

//new node根据传递的配置创建新的节点实例。数据中心
//将用于保存记录已启动进程的PID的文件，以及
//作为btcd日志和数据目录的基础。
func newNode(config *nodeConfig, dataDir string) (*node, error) {
	return &node{
		config:  config,
		dataDir: dataDir,
		cmd:     config.command(),
	}, nil
}

//start创建一个新的btcd进程，并将其pid写入为
//记录已启动进程的PID。此文件可用于
//如果出现挂起或恐慌，则终止进程。在失败的情况下
//测试用例或死机，必须通过stop（）停止进程，
//否则，除非明确杀死，否则它将持续存在。
func (n *node) start() error {
	if err := n.cmd.Start(); err != nil {
		return err
	}

	pid, err := os.Create(filepath.Join(n.dataDir,
		fmt.Sprintf("%s.pid", n.config)))
	if err != nil {
		return err
	}

	n.pidFile = pid.Name()
	if _, err = fmt.Fprintf(pid, "%d\n", n.cmd.Process.Pid); err != nil {
		return err
	}

	if err := pid.Close(); err != nil {
		return err
	}

	return nil
}

//stop中断正在运行的btcd进程，并等待它退出。
//适当地。在Windows上，不支持中断，因此使用了终止信号
//相反
func (n *node) stop() error {
	if n.cmd == nil || n.cmd.Process == nil {
//如果未正确初始化则返回
//或启动进程时出错
		return nil
	}
	defer n.cmd.Wait()
	if runtime.GOOS == "windows" {
		return n.cmd.Process.Signal(os.Kill)
	}
	return n.cmd.Process.Signal(os.Interrupt)
}

//清理清理进程和args文件。包含PID的文件
//创建的进程以及
//过程。
func (n *node) cleanup() error {
	if n.pidFile != "" {
		if err := os.Remove(n.pidFile); err != nil {
			log.Printf("unable to remove file %s: %v", n.pidFile,
				err)
		}
	}

	return n.config.cleanup()
}

//shutdown终止正在运行的btcd进程，并清除所有
//由节点创建的文件/目录。
func (n *node) shutdown() error {
	if err := n.stop(); err != nil {
		return err
	}
	if err := n.cleanup(); err != nil {
		return err
	}
	return nil
}

//gencertpair生成指向所提供路径的密钥/证书对。
func genCertPair(certFile, keyFile string) error {
	org := "rpctest autogenerated cert"
	validUntil := time.Now().Add(10 * 365 * 24 * time.Hour)
	cert, key, err := btcutil.NewTLSCertPair(org, validUntil, nil)
	if err != nil {
		return err
	}

//编写证书和密钥文件。
	if err = ioutil.WriteFile(certFile, cert, 0666); err != nil {
		return err
	}
	if err = ioutil.WriteFile(keyFile, key, 0600); err != nil {
		os.Remove(certFile)
		return err
	}

	return nil
}
