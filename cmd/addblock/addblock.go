
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
	"path/filepath"
	"runtime"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/blockchain/indexers"
	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcd/limits"
	"github.com/btcsuite/btclog"
)

const (
//BlockDBNamePrefix是BTCD块数据库的前缀。
	blockDbNamePrefix = "blocks"
)

var (
	cfg *config
	log btclog.Logger
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
//加载配置并分析命令行。
	tcfg, _, err := loadConfig()
	if err != nil {
		return err
	}
	cfg = tcfg

//设置日志记录。
	backendLogger := btclog.NewBackend(os.Stdout)
	defer os.Stdout.Sync()
	log = backendLogger.Logger("MAIN")
	database.UseLogger(backendLogger.Logger("BCDB"))
	blockchain.UseLogger(backendLogger.Logger("CHAN"))
	indexers.UseLogger(backendLogger.Logger("INDX"))

//加载块数据库。
	db, err := loadBlockDB()
	if err != nil {
		log.Errorf("Failed to load database: %v", err)
		return err
	}
	defer db.Close()

	fi, err := os.Open(cfg.InFile)
	if err != nil {
		log.Errorf("Failed to open file %v: %v", cfg.InFile, err)
		return err
	}
	defer fi.Close()

//为数据库和输入文件创建一个块导入程序并启动它。
//从start返回的done通道将包含一个错误，如果
//出了什么问题。
	importer, err := newBlockImporter(db, fi)
	if err != nil {
		log.Errorf("Failed create block importer: %v", err)
		return err
	}

//异步执行导入。这允许块
//并行处理和读取。从返回的结果通道
//导入包含有关导入的统计信息，包括错误
//如果出了什么问题。
	log.Info("Starting import")
	resultsChan := importer.Import()
	results := <-resultsChan
	if results.err != nil {
		log.Errorf("%v", results.err)
		return results.err
	}

	log.Infof("Processed a total of %d blocks (%d imported, %d already "+
		"known)", results.blocksProcessed, results.blocksImported,
		results.blocksProcessed-results.blocksImported)
	return nil
}

func main() {
//使用所有处理器内核并达到某些限制。
	runtime.GOMAXPROCS(runtime.NumCPU())
	if err := limits.SetLimits(); err != nil {
		os.Exit(1)
	}

//解决OS.exit（）之后延迟不工作的问题
	if err := realMain(); err != nil {
		os.Exit(1)
	}
}
