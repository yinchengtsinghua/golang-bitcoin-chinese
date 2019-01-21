
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/btcsuite/btcd/btcjson"
)

const (
	showHelpMessage = "Specify -h to show available options"
	listCmdMessage  = "Specify -l to list available commands"
)

//command usage显示特定命令的用法。
func commandUsage(method string) {
	usage, err := btcjson.MethodUsageText(method)
	if err != nil {
//这不应该发生，因为该方法已被检查
//在调用此函数之前，请确保安全。
		fmt.Fprintln(os.Stderr, "Failed to obtain command usage:", err)
		return
	}

	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintf(os.Stderr, "  %s\n", usage)
}

//用法显示未显示帮助标志时的常规用法，以及
//指定的命令无效。使用commandusage函数
//而是在指定有效命令时。
func usage(errorMessage string) {
	appName := filepath.Base(os.Args[0])
	appName = strings.TrimSuffix(appName, filepath.Ext(appName))
	fmt.Fprintln(os.Stderr, errorMessage)
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintf(os.Stderr, "  %s [OPTIONS] <command> <args...>\n\n",
		appName)
	fmt.Fprintln(os.Stderr, showHelpMessage)
	fmt.Fprintln(os.Stderr, listCmdMessage)
}

func main() {
	cfg, args, err := loadConfig()
	if err != nil {
		os.Exit(1)
	}
	if len(args) < 1 {
		usage("No command specified")
		os.Exit(1)
	}

//确保指定的方法标识有效的已注册命令并
//是可用类型之一。
	method := args[0]
	usageFlags, err := btcjson.MethodUsageFlags(method)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unrecognized command '%s'\n", method)
		fmt.Fprintln(os.Stderr, listCmdMessage)
		os.Exit(1)
	}
	if usageFlags&unusableFlags != 0 {
		fmt.Fprintf(os.Stderr, "The '%s' command can only be used via "+
			"websockets\n", method)
		fmt.Fprintln(os.Stderr, listCmdMessage)
		os.Exit(1)
	}

//Convert remaining command line args to a slice of interface values
//作为参数传递给新的命令创建函数。
//
//由于某些命令（如submitblock）可能涉及到
//操作系统太大，无法作为常规命令行使用
//参数，支持使用“-”作为参数来允许参数
//从标准输入管道读取。
	bio := bufio.NewReader(os.Stdin)
	params := make([]interface{}, 0, len(args[1:]))
	for _, arg := range args[1:] {
		if arg == "-" {
			param, err := bio.ReadString('\n')
			if err != nil && err != io.EOF {
				fmt.Fprintf(os.Stderr, "Failed to read data "+
					"from stdin: %v\n", err)
				os.Exit(1)
			}
			if err == io.EOF && len(param) == 0 {
				fmt.Fprintln(os.Stderr, "Not enough lines "+
					"provided on stdin")
				os.Exit(1)
			}
			param = strings.TrimRight(param, "\r\n")
			params = append(params, param)
			continue
		}

		params = append(params, arg)
	}

//尝试使用参数创建适当的命令
//由用户提供。
	cmd, err := btcjson.NewCmd(method, params...)
	if err != nil {
//当错误为
//btcjson.error，因为它实际上是
//newCmd函数只应返回该函数的错误
//类型。
		if jerr, ok := err.(btcjson.Error); ok {
			fmt.Fprintf(os.Stderr, "%s command: %v (code: %s)\n",
				method, err, jerr.ErrorCode)
			commandUsage(method)
			os.Exit(1)
		}

//该错误不是btcjson.error，而实际上不应该是
//发生。然而，回退到只显示错误
//如果它是由于包中的错误而发生的。
		fmt.Fprintf(os.Stderr, "%s command: %v\n", method, err)
		commandUsage(method)
		os.Exit(1)
	}

//将命令封送到JSON-RPC字节片中，以准备
//正在将其发送到RPC服务器。
	marshalledJSON, err := btcjson.MarshalCmd(1, cmd)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

//使用指定的用户将JSON-RPC请求发送到服务器
//连接配置。
	result, err := sendPostRequest(marshalledJSON, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

//选择如何根据其类型显示结果。
	strResult := string(result)
	if strings.HasPrefix(strResult, "{") || strings.HasPrefix(strResult, "[") {
		var dst bytes.Buffer
		if err := json.Indent(&dst, result, "", "  "); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to format result: %v",
				err)
			os.Exit(1)
		}
		fmt.Println(dst.String())

	} else if strings.HasPrefix(strResult, `"`) {
		var str string
		if err := json.Unmarshal(result, &str); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to unmarshal result: %v",
				err)
			os.Exit(1)
		}
		fmt.Println(str)

	} else if strResult != "null" {
		fmt.Println(strResult)
	}
}
