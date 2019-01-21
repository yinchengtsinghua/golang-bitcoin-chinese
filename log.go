
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2017 BTCSuite开发者
//版权所有（c）2017版权所有
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/btcsuite/btcd/addrmgr"
	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/blockchain/indexers"
	"github.com/btcsuite/btcd/connmgr"
	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcd/mempool"
	"github.com/btcsuite/btcd/mining"
	"github.com/btcsuite/btcd/mining/cpuminer"
	"github.com/btcsuite/btcd/netsync"
	"github.com/btcsuite/btcd/peer"
	"github.com/btcsuite/btcd/txscript"

	"github.com/btcsuite/btclog"
	"github.com/jrick/logrotate/rotator"
)

//LogWriter实现了一个IO.Writer，它同时输出到标准输出和
//初始化的日志旋转器的写入结束管道。
type logWriter struct{}

func (logWriter) Write(p []byte) (n int, err error) {
	os.Stdout.Write(p)
	logRotator.Write(p)
	return len(p), nil
}

//每个子系统的记录器。创建一个后端记录器，并创建所有子系统
//从中创建的记录器将写入后端。添加新
//子系统，将subsystem logger变量添加到
//子系统记录器映射。
//
//在用初始化日志旋转器之前，不能使用记录器
//日志文件。必须在应用程序启动的早期通过调用
//内旋转器
var (
//backendlog是用于创建所有子系统记录器的日志后端。
//在日志旋转器初始化之前，不能使用后端。
//否则将发生数据争用和/或零指针取消引用。
	backendLog = btclog.NewBackend(logWriter{})

//logrotator是日志输出之一。它应该关闭
//应用程序关闭。
	logRotator *rotator.Rotator

	adxrLog = backendLog.Logger("ADXR")
	amgrLog = backendLog.Logger("AMGR")
	cmgrLog = backendLog.Logger("CMGR")
	bcdbLog = backendLog.Logger("BCDB")
	btcdLog = backendLog.Logger("BTCD")
	chanLog = backendLog.Logger("CHAN")
	discLog = backendLog.Logger("DISC")
	indxLog = backendLog.Logger("INDX")
	minrLog = backendLog.Logger("MINR")
	peerLog = backendLog.Logger("PEER")
	rpcsLog = backendLog.Logger("RPCS")
	scrpLog = backendLog.Logger("SCRP")
	srvrLog = backendLog.Logger("SRVR")
	syncLog = backendLog.Logger("SYNC")
	txmpLog = backendLog.Logger("TXMP")
)

//初始化包全局记录器变量。
func init() {
	addrmgr.UseLogger(amgrLog)
	connmgr.UseLogger(cmgrLog)
	database.UseLogger(bcdbLog)
	blockchain.UseLogger(chanLog)
	indexers.UseLogger(indxLog)
	mining.UseLogger(minrLog)
	cpuminer.UseLogger(minrLog)
	peer.UseLogger(peerLog)
	txscript.UseLogger(scrpLog)
	netsync.UseLogger(syncLog)
	mempool.UseLogger(txmpLog)
}

//子系统记录器将每个子系统标识符映射到其关联的记录器。
var subsystemLoggers = map[string]btclog.Logger{
	"ADXR": adxrLog,
	"AMGR": amgrLog,
	"CMGR": cmgrLog,
	"BCDB": bcdbLog,
	"BTCD": btcdLog,
	"CHAN": chanLog,
	"DISC": discLog,
	"INDX": indxLog,
	"MINR": minrLog,
	"PEER": peerLog,
	"RPCS": rpcsLog,
	"SCRP": scrpLog,
	"SRVR": srvrLog,
	"SYNC": syncLog,
	"TXMP": txmpLog,
}

//initlogrotator初始化日志记录旋转器，将日志写入日志文件并
//在同一目录中创建滚动文件。必须在
//使用包全局日志旋转器变量。
func initLogRotator(logFile string) {
	logDir, _ := filepath.Split(logFile)
	err := os.MkdirAll(logDir, 0700)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create log directory: %v\n", err)
		os.Exit(1)
	}
	r, err := rotator.New(logFile, 10*1024, false, 3)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create file rotator: %v\n", err)
		os.Exit(1)
	}

	logRotator = r
}

//setloglevel为提供的子系统设置日志级别。无效
//忽略子系统。未初始化的子系统动态创建为
//需要。
func setLogLevel(subsystemID string, logLevel string) {
//忽略无效的子系统。
	logger, ok := subsystemLoggers[subsystemID]
	if !ok {
		return
	}

//如果日志级别无效，则默认为INFO。
	level, _ := btclog.LevelFromString(logLevel)
	logger.SetLevel(level)
}

//setloglevels将所有子系统记录器的日志级别设置为
//水平。它还根据需要动态创建子系统记录器，因此
//可用于初始化日志记录系统。
func setLogLevels(logLevel string) {
//使用新的日志级别配置所有子系统。动态地
//根据需要创建记录器。
	for subsystemID := range subsystemLoggers {
		setLogLevel(subsystemID, logLevel)
	}
}

//DirectionString是一个助手函数，它返回一个表示
//连接的方向（入站或出站）。
func directionString(inbound bool) string {
	if inbound {
		return "inbound"
	}
	return "outbound"
}

//picknoun返回名词的单数或复数形式，具体取决于
//在计数上
func pickNoun(n uint64, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
