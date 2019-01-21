
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

package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcutil"
	flags "github.com/jessevdk/go-flags"
)

const (
//unusableflags是命令使用标志，该实用程序不是
//能够使用。尤其是它不支持WebSockets和
//因此，通知。
	unusableFlags = btcjson.UFWebsocketOnly | btcjson.UFNotification
)

var (
	btcdHomeDir           = btcutil.AppDataDir("btcd", false)
	btcctlHomeDir         = btcutil.AppDataDir("btcctl", false)
	btcwalletHomeDir      = btcutil.AppDataDir("btcwallet", false)
	defaultConfigFile     = filepath.Join(btcctlHomeDir, "btcctl.conf")
	defaultRPCServer      = "localhost"
	defaultRPCCertFile    = filepath.Join(btcdHomeDir, "rpc.cert")
	defaultWalletCertFile = filepath.Join(btcwalletHomeDir, "rpc.cert")
)

//listcommands分类并列出所有可用的命令以及
//它们的单行用法。
func listCommands() {
	const (
		categoryChain uint8 = iota
		categoryWallet
		numCategories
	)

//获取已注册命令的列表，并对其进行分类和筛选。
	cmdMethods := btcjson.RegisteredCmdMethods()
	categorized := make([][]string, numCategories)
	for _, method := range cmdMethods {
		flags, err := btcjson.MethodUsageFlags(method)
		if err != nil {
//这不应该发生，因为方法只是
//从包中返回，但要安全。
			continue
		}

//跳过此实用程序中不可用的命令。
		if flags&unusableFlags != 0 {
			continue
		}

		usage, err := btcjson.MethodUsageText(method)
		if err != nil {
//这不应该发生，因为方法只是
//从包中返回，但要安全。
			continue
		}

//根据使用标志对命令进行分类。
		category := categoryChain
		if flags&btcjson.UFWalletOnly != 0 {
			category = categoryWallet
		}
		categorized[category] = append(categorized[category], usage)
	}

//根据命令的类别显示命令。
	categoryTitles := make([]string, numCategories)
	categoryTitles[categoryChain] = "Chain Server Commands:"
	categoryTitles[categoryWallet] = "Wallet Server Commands (--wallet):"
	for category := uint8(0); category < numCategories; category++ {
		fmt.Println(categoryTitles[category])
		for _, usage := range categorized[category] {
			fmt.Println(usage)
		}
		fmt.Println()
	}
}

//config定义btctl的配置选项。
//
//有关配置加载过程的详细信息，请参阅loadconfig。
type config struct {
	ShowVersion   bool   `short:"V" long:"version" description:"Display version information and exit"`
	ListCommands  bool   `short:"l" long:"listcommands" description:"List all of the supported commands and exit"`
	ConfigFile    string `short:"C" long:"configfile" description:"Path to configuration file"`
	RPCUser       string `short:"u" long:"rpcuser" description:"RPC username"`
	RPCPassword   string `short:"P" long:"rpcpass" default-mask:"-" description:"RPC password"`
	RPCServer     string `short:"s" long:"rpcserver" description:"RPC server to connect to"`
	RPCCert       string `short:"c" long:"rpccert" description:"RPC server certificate chain for validation"`
	NoTLS         bool   `long:"notls" description:"Disable TLS"`
	Proxy         string `long:"proxy" description:"Connect via SOCKS5 proxy (eg. 127.0.0.1:9050)"`
	ProxyUser     string `long:"proxyuser" description:"Username for proxy server"`
	ProxyPass     string `long:"proxypass" default-mask:"-" description:"Password for proxy server"`
	TestNet3      bool   `long:"testnet" description:"Connect to testnet"`
	SimNet        bool   `long:"simnet" description:"Connect to the simulation test network"`
	TLSSkipVerify bool   `long:"skipverify" description:"Do not verify tls certificates (not recommended!)"`
	Wallet        bool   `long:"wallet" description:"Connect to wallet"`
}

//normalizedAddress返回addr，并附加传递的默认端口if
//尚未指定端口。
func normalizeAddress(addr string, useTestNet3, useSimNet, useWallet bool) string {
	_, _, err := net.SplitHostPort(addr)
	if err != nil {
		var defaultPort string
		switch {
		case useTestNet3:
			if useWallet {
				defaultPort = "18332"
			} else {
				defaultPort = "18334"
			}
		case useSimNet:
			if useWallet {
				defaultPort = "18554"
			} else {
				defaultPort = "18556"
			}
		default:
			if useWallet {
				defaultPort = "8332"
			} else {
				defaultPort = "8334"
			}
		}

		return net.JoinHostPort(addr, defaultPort)
	}
	return addr
}

//cleanandexpandpath扩展环境变量并在
//传递路径，清除结果并返回。
func cleanAndExpandPath(path string) string {
//将initial~扩展到操作系统特定的主目录。
	if strings.HasPrefix(path, "~") {
		homeDir := filepath.Dir(btcctlHomeDir)
		path = strings.Replace(path, "~", homeDir, 1)
	}

//注意：os.expandenv不适用于Windows样式%variable%，
//但这些变量仍然可以通过posix样式的$variable进行扩展。
	return filepath.Clean(os.ExpandEnv(path))
}

//loadconfig使用配置文件和命令初始化并分析配置
//行选项。
//
//配置过程如下：
//1）从具有健全设置的默认配置开始
//2）预分析命令行以检查备用配置文件
//3）使用任何指定选项加载配置文件覆盖默认值
//4）解析cli选项并覆盖/添加任何指定选项
//
//以上结果导致在没有任何配置设置的情况下正常工作
//同时仍允许用户使用配置文件和
//命令行选项。命令行选项始终优先。
func loadConfig() (*config, []string, error) {
//默认配置。
	cfg := config{
		ConfigFile: defaultConfigFile,
		RPCServer:  defaultRPCServer,
		RPCCert:    defaultRPCCertFile,
	}

//预分析命令行选项，以查看是否有其他配置
//指定了文件、版本标志或列表命令标志。任何
//除了帮助消息之外的错误在这里可以忽略，因为
//它们将被下面的最终分析捕获。
	preCfg := cfg
	preParser := flags.NewParser(&preCfg, flags.HelpFlag)
	_, err := preParser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			fmt.Fprintln(os.Stderr, err)
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "The special parameter `-` "+
				"indicates that a parameter should be read "+
				"from the\nnext unread line from standard "+
				"input.")
			return nil, nil, err
		}
	}

//显示版本，如果指定了版本标志，则退出。
	appName := filepath.Base(os.Args[0])
	appName = strings.TrimSuffix(appName, filepath.Ext(appName))
	usageMessage := fmt.Sprintf("Use %s -h to show options", appName)
	if preCfg.ShowVersion {
		fmt.Println(appName, "version", version())
		os.Exit(0)
	}

//显示可用的命令，如果关联的标志是
//明确规定。
	if preCfg.ListCommands {
		listCommands()
		os.Exit(0)
	}

	if _, err := os.Stat(preCfg.ConfigFile); os.IsNotExist(err) {
//使用rpc服务器的配置文件创建默认btctl config
		var serverConfigPath string
		if preCfg.Wallet {
			serverConfigPath = filepath.Join(btcwalletHomeDir, "btcwallet.conf")
		} else {
			serverConfigPath = filepath.Join(btcdHomeDir, "btcd.conf")
		}

		err := createDefaultConfigFile(preCfg.ConfigFile, serverConfigPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating a default config file: %v\n", err)
		}
	}

//从文件加载附加配置。
	parser := flags.NewParser(&cfg, flags.Default)
	err = flags.NewIniParser(parser).ParseFile(preCfg.ConfigFile)
	if err != nil {
		if _, ok := err.(*os.PathError); !ok {
			fmt.Fprintf(os.Stderr, "Error parsing config file: %v\n",
				err)
			fmt.Fprintln(os.Stderr, usageMessage)
			return nil, nil, err
		}
	}

//再次分析命令行选项以确保它们优先。
	remainingArgs, err := parser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); !ok || e.Type != flags.ErrHelp {
			fmt.Fprintln(os.Stderr, usageMessage)
		}
		return nil, nil, err
	}

//无法同时选择多个网络。
	numNets := 0
	if cfg.TestNet3 {
		numNets++
	}
	if cfg.SimNet {
		numNets++
	}
	if numNets > 1 {
		str := "%s: The testnet and simnet params can't be used " +
			"together -- choose one of the two"
		err := fmt.Errorf(str, "loadConfig")
		fmt.Fprintln(os.Stderr, err)
		return nil, nil, err
	}

//如果指定了--wallet标志并且
//用户未指定。
	if cfg.Wallet && cfg.RPCCert == defaultRPCCertFile {
		cfg.RPCCert = defaultWalletCertFile
	}

//处理RPC证书路径中的环境变量扩展。
	cfg.RPCCert = cleanAndExpandPath(cfg.RPCCert)

//基于--testnet和--wallet标志向rpc服务器添加默认端口
//如果需要的话。
	cfg.RPCServer = normalizeAddress(cfg.RPCServer, cfg.TestNet3,
		cfg.SimNet, cfg.Wallet)

	return &cfg, remainingArgs, nil
}

//createDefaultConfig在给定的目标路径创建一个基本配置文件。
//为此，它尝试读取RPC服务器的配置文件（btcd或
//btcwallet），并从中提取RPC用户和密码。
func createDefaultConfigFile(destinationPath, serverConfigPath string) error {
//读取RPC服务器配置
	serverConfigFile, err := os.Open(serverConfigPath)
	if err != nil {
		return err
	}
	defer serverConfigFile.Close()
	content, err := ioutil.ReadAll(serverConfigFile)
	if err != nil {
		return err
	}

//取出RPCUSER
	rpcUserRegexp, err := regexp.Compile(`(?m)^\s*rpcuser=([^\s]+)`)
	if err != nil {
		return err
	}
	userSubmatches := rpcUserRegexp.FindSubmatch(content)
	if userSubmatches == nil {
//找不到用户，无需执行任何操作
		return nil
	}

//提取RPCPass
	rpcPassRegexp, err := regexp.Compile(`(?m)^\s*rpcpass=([^\s]+)`)
	if err != nil {
		return err
	}
	passSubmatches := rpcPassRegexp.FindSubmatch(content)
	if passSubmatches == nil {
//找不到密码，无需执行任何操作
		return nil
	}

//提取notls
	noTLSRegexp, err := regexp.Compile(`(?m)^\s*notls=(0|1)(?:\s|$)`)
	if err != nil {
		return err
	}
	noTLSSubmatches := noTLSRegexp.FindSubmatch(content)

//如果目标目录不存在，则创建该目录
	err = os.MkdirAll(filepath.Dir(destinationPath), 0700)
	if err != nil {
		return err
	}

//创建目标文件并将rpcuser和rpcpass写入其中
	dest, err := os.OpenFile(destinationPath,
		os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer dest.Close()

	destString := fmt.Sprintf("rpcuser=%s\nrpcpass=%s\n",
		string(userSubmatches[1]), string(passSubmatches[1]))
	if noTLSSubmatches != nil {
		destString += fmt.Sprintf("notls=%s\n", noTLSSubmatches[1])
	}

	dest.WriteString(destString)

	return nil
}
