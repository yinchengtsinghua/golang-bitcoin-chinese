
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btclog"
	flags "github.com/jessevdk/go-flags"
)

const (
//BlockDBNamePrefix是BTCD块数据库的前缀。
	blockDbNamePrefix = "blocks"
)

var (
	log             btclog.Logger
	shutdownChannel = make(chan error)
)

//loadblockdb打开块数据库并返回其句柄。
func loadBlockDB() (database.DB, error) {
//数据库名称基于数据库类型。
	dbName := blockDbNamePrefix + "_" + cfg.DbType
	dbPath := filepath.Join(cfg.DataDir, dbName)

	log.Infof("Loading block database from '%s'", dbPath)
	db, err := database.Open(cfg.DbType, dbPath, activeNetParams.Net)
	if err != nil {
//如果不是因为数据库没有
//存在。
		if dbErr, ok := err.(database.Error); !ok || dbErr.ErrorCode !=
			database.ErrDbDoesNotExist {

			return nil, err
		}

//如果数据库不存在，则创建它。
		err = os.MkdirAll(cfg.DataDir, 0700)
		if err != nil {
			return nil, err
		}
		db, err = database.Create(cfg.DbType, dbPath, activeNetParams.Net)
		if err != nil {
			return nil, err
		}
	}

	log.Info("Block database loaded")
	return db, nil
}

//real main是实用程序的真正主要功能。有必要工作
//在调用os.exit（）时，延迟函数不运行。
func realMain() error {
//设置日志记录。
	backendLogger := btclog.NewBackend(os.Stdout)
	defer os.Stdout.Sync()
	log = backendLogger.Logger("MAIN")
	dbLog := backendLogger.Logger("BCDB")
	dbLog.SetLevel(btclog.LevelDebug)
	database.UseLogger(dbLog)

//设置解析器选项和命令。
	appName := filepath.Base(os.Args[0])
	appName = strings.TrimSuffix(appName, filepath.Ext(appName))
	parserFlags := flags.Options(flags.HelpFlag | flags.PassDoubleDash)
	parser := flags.NewNamedParser(appName, parserFlags)
	parser.AddGroup("Global Options", "", cfg)
	parser.AddCommand("insecureimport",
		"Insecurely import bulk block data from bootstrap.dat",
		"Insecurely import bulk block data from bootstrap.dat.  "+
			"WARNING: This is NOT secure because it does NOT "+
			"verify chain rules.  It is only provided for testing "+
			"purposes.", &importCfg)
	parser.AddCommand("loadheaders",
		"Time how long to load headers for all blocks in the database",
		"", &headersCfg)
	parser.AddCommand("fetchblock",
		"Fetch the specific block hash from the database", "",
		&fetchBlockCfg)
	parser.AddCommand("fetchblockregion",
		"Fetch the specified block region from the database", "",
		&blockRegionCfg)

//分析命令行并为指定的
//命令。
	if _, err := parser.Parse(); err != nil {
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			parser.WriteHelp(os.Stderr)
		} else {
			log.Error(err)
		}

		return err
	}

	return nil
}

func main() {
//使用所有处理器核心。
	runtime.GOMAXPROCS(runtime.NumCPU())

//解决OS.exit（）之后延迟不工作的问题
	if err := realMain(); err != nil {
		os.Exit(1)
	}
}
