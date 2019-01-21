
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
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/database"
	_ "github.com/btcsuite/btcd/database/ffldb"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

var (
	btcdHomeDir     = btcutil.AppDataDir("btcd", false)
	knownDbTypes    = database.SupportedDrivers()
	activeNetParams = &chaincfg.MainNetParams

//默认全局配置。
	cfg = &config{
		DataDir: filepath.Join(btcdHomeDir, "data"),
		DbType:  "ffldb",
	}
)

//config定义全局配置选项。
type config struct {
	DataDir        string `short:"b" long:"datadir" description:"Location of the btcd data directory"`
	DbType         string `long:"dbtype" description:"Database backend to use for the Block Chain"`
	TestNet3       bool   `long:"testnet" description:"Use the test network"`
	RegressionTest bool   `long:"regtest" description:"Use the regression test network"`
	SimNet         bool   `long:"simnet" description:"Use the simulation test network"`
}

//file exists报告命名文件或目录是否存在。
func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

//validdbtype返回dbtype是否为受支持的数据库类型。
func validDbType(dbType string) bool {
	for _, knownType := range knownDbTypes {
		if dbType == knownType {
			return true
		}
	}

	return false
}

//netname返回引用比特币网络时使用的名称。在
//写入时间，btcd当前将testnet版本3的块放在
//数据和日志目录“testnet”，与
//CHAINCFG参数。此函数可用于重写此目录名
//当传递的活动网络与Wire.TestNet3匹配时，为“testnet”。
//
//要将此网络的数据和日志目录移动到
//“testnet3”是为将来而计划的，此时，此功能可以
//已删除，并改用网络参数的名称。
func netName(chainParams *chaincfg.Params) string {
	switch chainParams.Net {
	case wire.TestNet3:
		return "testnet"
	default:
		return chainParams.Name
	}
}

//设置全局配置检查全局配置选项是否存在任何条件
//它是无效的，并且在
//初始解析。
func setupGlobalConfig() error {
//无法同时选择多个网络。
//计数传递的网络标志数；分配活动的网络参数
//当我们在那里的时候
	numNets := 0
	if cfg.TestNet3 {
		numNets++
		activeNetParams = &chaincfg.TestNet3Params
	}
	if cfg.RegressionTest {
		numNets++
		activeNetParams = &chaincfg.RegressionNetParams
	}
	if cfg.SimNet {
		numNets++
		activeNetParams = &chaincfg.SimNetParams
	}
	if numNets > 1 {
		return errors.New("The testnet, regtest, and simnet params " +
			"can't be used together -- choose one of the three")
	}

//验证数据库类型。
	if !validDbType(cfg.DbType) {
		str := "The specified database type [%v] is invalid -- " +
			"supported types %v"
		return fmt.Errorf(str, cfg.DbType, knownDbTypes)
	}

//将网络类型附加到数据目录中，使其具有“名称空间”
//每个网络。除了块数据库，还有其他
//保存到磁盘上的数据块，如地址管理器状态。
//所有数据都是特定于一个网络的，因此名称间隔数据目录
//意味着每个单独的序列化数据不必
//担心更改每个网络的名称等等。
	cfg.DataDir = filepath.Join(cfg.DataDir, netName(activeNetParams))

	return nil
}
