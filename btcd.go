
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
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"runtime/pprof"

	"github.com/btcsuite/btcd/blockchain/indexers"
	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcd/limits"
)

const (
//BlockDBNamePrefix是块数据库名称的前缀。这个
//数据库类型附加到此值后形成完整的块
//数据库名称。
	blockDbNamePrefix = "blocks"
)

var (
	cfg *config
)

//仅在Windows上调用WinServiceMain。它检测BTCD何时运行
//作为一种服务，并做出相应的反应。
var winServiceMain func() (bool, error)

//BTCDMAIN是BTCD真正的主要功能。有必要四处工作
//当调用os.exit（）时，延迟函数不运行。这个
//可选的serverchan参数主要用于
//在服务器设置后通知它，以便它可以在
//从服务控制管理器请求。
func btcdMain(serverChan chan<- *server) error {
//加载配置并分析命令行。此功能也
//初始化日志并进行相应的配置。
	tcfg, _, err := loadConfig()
	if err != nil {
		return err
	}
	cfg = tcfg
	defer func() {
		if logRotator != nil {
			logRotator.Close()
		}
	}()

//获取关闭信号时将关闭的通道
//从操作系统信号（如sigint（ctrl+c））或从
//
	interrupt := interruptListener()
	defer btcdLog.Info("Shutdown complete")

//启动时显示版本。
	btcdLog.Infof("Version %s", version())

//如果请求，启用HTTP分析服务器。
	if cfg.Profile != "" {
		go func() {
			listenAddr := net.JoinHostPort("", cfg.Profile)
			btcdLog.Infof("Profile server listening on %s", listenAddr)
			profileRedirect := http.RedirectHandler("/debug/pprof",
				http.StatusSeeOther)
			http.Handle("/", profileRedirect)
			btcdLog.Errorf("%v", http.ListenAndServe(listenAddr, nil))
		}()
	}

//如果需要，写入CPU配置文件。
	if cfg.CPUProfile != "" {
		f, err := os.Create(cfg.CPUProfile)
		if err != nil {
			btcdLog.Errorf("Unable to create cpu profile: %v", err)
			return err
		}
		pprof.StartCPUProfile(f)
		defer f.Close()
		defer pprof.StopCPUProfile()
	}

//根据新版本的需要升级到BTCD。
	if err := doUpgrades(); err != nil {
		btcdLog.Errorf("%v", err)
		return err
	}

//如果触发了中断信号，立即返回。
	if interruptRequested(interrupt) {
		return nil
	}

//加载块数据库。
	db, err := loadBlockDB()
	if err != nil {
		btcdLog.Errorf("%v", err)
		return err
	}
	defer func() {
//确保数据库已同步并在关闭时关闭。
		btcdLog.Infof("Gracefully shutting down the database...")
		db.Close()
	}()

//如果触发了中断信号，立即返回。
	if interruptRequested(interrupt) {
		return nil
	}

//删除索引并在请求时退出。
//
//注意：这里的顺序很重要，因为删除tx索引也很重要
//删除地址索引，因为它依赖于它。
	if cfg.DropAddrIndex {
		if err := indexers.DropAddrIndex(db, interrupt); err != nil {
			btcdLog.Errorf("%v", err)
			return err
		}

		return nil
	}
	if cfg.DropTxIndex {
		if err := indexers.DropTxIndex(db, interrupt); err != nil {
			btcdLog.Errorf("%v", err)
			return err
		}

		return nil
	}
	if cfg.DropCfIndex {
		if err := indexers.DropCfIndex(db, interrupt); err != nil {
			btcdLog.Errorf("%v", err)
			return err
		}

		return nil
	}

//
	server, err := newServer(cfg.Listeners, db, activeNetParams.Params,
		interrupt)
	if err != nil {
//托多：这种伐木可以美化环境。
		btcdLog.Errorf("Unable to start server on %v: %v",
			cfg.Listeners, err)
		return err
	}
	defer func() {
		btcdLog.Infof("Gracefully shutting down the server...")
		server.Stop()
		server.WaitForShutdown()
		srvrLog.Infof("Server shutdown complete")
	}()
	server.Start()
	if serverChan != nil {
		serverChan <- server
	}

//等待直到从操作系统信号接收到中断信号或
//通过一个子系统（如rpc）请求关闭
//服务器。
	<-interrupt
	return nil
}

//如果正在运行，则removeexcessiondb将删除现有的回归测试数据库
//在回归测试模式中，它已经存在。
func removeRegressionDB(dbPath string) error {
//如果不处于回归测试模式，则不要执行任何操作。
	if !cfg.RegressionTest {
		return nil
	}

//删除旧的回归测试数据库（如果它已经存在）。
	fi, err := os.Stat(dbPath)
	if err == nil {
		btcdLog.Infof("Removing regression test database from '%s'", dbPath)
		if fi.IsDir() {
			err := os.RemoveAll(dbPath)
			if err != nil {
				return err
			}
		} else {
			err := os.Remove(dbPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

//dbpath返回给定数据库类型的块数据库的路径。
func blockDbPath(dbType string) string {
//数据库名称基于数据库类型。
	dbName := blockDbNamePrefix + "_" + dbType
	if dbType == "sqlite" {
		dbName = dbName + ".db"
	}
	dbPath := filepath.Join(cfg.DataDir, dbName)
	return dbPath
}

//如果检测到多个块数据库类型，warnmultipledbs将显示警告。
//
//支持多个并行数据库。
func warnMultipleDBs() {
//这是故意不使用已知的数据库类型
//关于编译成二进制文件的数据库类型，因为我们希望
//同时检测旧数据库类型。
	dbTypes := []string{"ffldb", "leveldb", "sqlite"}
	duplicateDbPaths := make([]string, 0, len(dbTypes)-1)
	for _, dbType := range dbTypes {
		if dbType == cfg.DbType {
			continue
		}

//如果数据库路径存在，则将其存储为重复的数据库。
		dbPath := blockDbPath(dbType)
		if fileExists(dbPath) {
			duplicateDbPaths = append(duplicateDbPaths, dbPath)
		}
	}

//如果有额外的数据库，则发出警告。
	if len(duplicateDbPaths) > 0 {
		selectedDbPath := blockDbPath(cfg.DbType)
		btcdLog.Warnf("WARNING: There are multiple block chain databases "+
			"using different database types.\nYou probably don't "+
			"want to waste disk space by having more than one.\n"+
			"Your current database is located at [%v].\nThe "+
			"additional database is located at %v", selectedDbPath,
			duplicateDbPaths)
	}
}

//loadblockdb加载（或在需要时创建）块数据库
//帐户选定的数据库后端并返回其句柄。它也
//包含其他逻辑，如警告用户存在多个
//消耗文件系统空间并确保回归的数据库
//在回归测试模式下，测试数据库是干净的。
func loadBlockDB() (database.DB, error) {
//memdb后端没有与其关联的文件路径，因此
//独特处理。我们也不想担心多重性
//使用内存数据库运行时出现数据库类型警告。
	if cfg.DbType == "memdb" {
		btcdLog.Infof("Creating block database in memory.")
		db, err := database.Create(cfg.DbType)
		if err != nil {
			return nil, err
		}
		return db, nil
	}

	warnMultipleDBs()

//数据库名称基于数据库类型。
	dbPath := blockDbPath(cfg.DbType)

//回归测试的特殊之处在于它需要一个干净的数据库
//每次运行，如果它已经存在，现在就删除它。
	removeRegressionDB(dbPath)

	btcdLog.Infof("Loading block database from '%s'", dbPath)
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

	btcdLog.Info("Block database loaded")
	return db, nil
}

func main() {
//使用所有处理器核心。
	runtime.GOMAXPROCS(runtime.NumCPU())

//块和事务处理可能会导致激烈的分配。这个
//限制垃圾收集器在
//爆发。此值是在分析Live的帮助下获得的
//用法。
	debug.SetGCPercent(10)

//有一些限制。
	if err := limits.SetLimits(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set limits: %v\n", err)
		os.Exit(1)
	}

//调用Windows上的ServiceMain以处理作为服务运行的情况。什么时候？
//返回IsService标志为true，现在退出，因为我们作为
//服务。否则，就只能正常运行。
	if runtime.GOOS == "windows" {
		isService, err := winServiceMain()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if isService {
			os.Exit(0)
		}
	}

//解决OS.exit（）之后延迟不工作的问题
	if err := btcdMain(nil); err != nil {
		os.Exit(1)
	}
}
