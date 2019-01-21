
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

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/btcsuite/btcutil"
	flags "github.com/jessevdk/go-flags"
)

type config struct {
	Directory    string   `short:"d" long:"directory" description:"Directory to write certificate pair"`
	Years        int      `short:"y" long:"years" description:"How many years a certificate is valid for"`
	Organization string   `short:"o" long:"org" description:"Organization in certificate"`
	ExtraHosts   []string `short:"H" long:"host" description:"Additional hosts/IPs to create certificate for"`
	Force        bool     `short:"f" long:"force" description:"Force overwriting of any old certs and keys"`
}

func main() {
	cfg := config{
		Years:        10,
		Organization: "gencerts",
	}
	parser := flags.NewParser(&cfg, flags.Default)
	_, err := parser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); !ok || e.Type != flags.ErrHelp {
			parser.WriteHelp(os.Stderr)
		}
		return
	}

	if cfg.Directory == "" {
		var err error
		cfg.Directory, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "no directory specified and cannot get working directory\n")
			os.Exit(1)
		}
	}
	cfg.Directory = cleanAndExpandPath(cfg.Directory)
	certFile := filepath.Join(cfg.Directory, "rpc.cert")
	keyFile := filepath.Join(cfg.Directory, "rpc.key")

	if !cfg.Force {
		if fileExists(certFile) || fileExists(keyFile) {
			fmt.Fprintf(os.Stderr, "%v: certificate and/or key files exist; use -f to force\n", cfg.Directory)
			os.Exit(1)
		}
	}

	validUntil := time.Now().Add(time.Duration(cfg.Years) * 365 * 24 * time.Hour)
	cert, key, err := btcutil.NewTLSCertPair(cfg.Organization, validUntil, cfg.ExtraHosts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot generate certificate pair: %v\n", err)
		os.Exit(1)
	}

//编写证书和密钥文件。
	if err = ioutil.WriteFile(certFile, cert, 0666); err != nil {
		fmt.Fprintf(os.Stderr, "cannot write cert: %v\n", err)
		os.Exit(1)
	}
	if err = ioutil.WriteFile(keyFile, key, 0600); err != nil {
		os.Remove(certFile)
		fmt.Fprintf(os.Stderr, "cannot write key: %v\n", err)
		os.Exit(1)
	}
}

//cleanandexpandpath扩展环境变量并在
//传递路径，清除结果并返回。
func cleanAndExpandPath(path string) string {
//将initial~扩展到操作系统特定的主目录。
	if strings.HasPrefix(path, "~") {
		appHomeDir := btcutil.AppDataDir("gencerts", false)
		homeDir := filepath.Dir(appHomeDir)
		path = strings.Replace(path, "~", homeDir, 1)
	}

//注意：os.expandenv不适用于Windows样式%variable%，
//但这些变量仍然可以通过posix样式的$variable进行扩展。
	return filepath.Clean(os.ExpandEnv(path))
}

//filesexists报告命名文件或目录是否存在。
func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
