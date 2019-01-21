
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
	"io"
	"os"
	"path/filepath"
)

//dirempty返回指定的目录路径是否为空。
func dirEmpty(dirPath string) (bool, error) {
	f, err := os.Open(dirPath)
	if err != nil {
		return false, err
	}
	defer f.Close()

//从目录中最多读取一个条目的名称。当
//目录为空，将返回IO.EOF错误，因此允许它。
	names, err := f.Readdirnames(1)
	if err != nil && err != io.EOF {
		return false, err
	}

	return len(names) == 0, nil
}

//oldbtcdhomedir返回在
//0.3.3版。这已经被btcutil.appdatadir替换，但是
//此功能仍为自动升级路径提供。
func oldBtcdHomeDir() string {
//首先搜索Windows AppData。这在POSIX操作系统上不存在。
	appData := os.Getenv("APPDATA")
	if appData != "" {
		return filepath.Join(appData, "btcd")
	}

//回到适用于大多数POSIX操作系统的标准主目录。
	home := os.Getenv("HOME")
	if home != "" {
		return filepath.Join(home, ".btcd")
	}

//在最坏的情况下，使用当前目录。
	return "."
}

//upgradedbpathnet将特定网络的数据库从其
//在BTCD 0.2.0之前的位置，并使用启发式方法确定旧的
//要重命名为新格式的数据库类型。
func upgradeDBPathNet(oldDbPath, netName string) error {
//在0.2.0版本之前，数据库的名称与
//sqlite和leveldb。使用启发式计算出类型
//并将其移动到
//版本0.2.0。
	fi, err := os.Stat(oldDbPath)
	if err == nil {
		oldDbType := "sqlite"
		if fi.IsDir() {
			oldDbType = "leveldb"
		}

//新数据库名称基于数据库类型和
//位于以网络类型命名的目录中。
		newDbRoot := filepath.Join(filepath.Dir(cfg.DataDir), netName)
		newDbName := blockDbNamePrefix + "_" + oldDbType
		if oldDbType == "sqlite" {
			newDbName = newDbName + ".db"
		}
		newDbPath := filepath.Join(newDbRoot, newDbName)

//如果需要，创建新路径。
		err = os.MkdirAll(newDbRoot, 0700)
		if err != nil {
			return err
		}

//移动并重命名旧数据库。
		err := os.Rename(oldDbPath, newDbPath)
		if err != nil {
			return err
		}
	}

	return nil
}

//upgradedbpaths将数据库从BTCD之前的位置移动
//版本0.2.0到新位置。
func upgradeDBPaths() error {
//在0.2.0版本之前，数据库位于“db”目录中，并且
//他们的名字后加了“testnet”和“regtest”作为他们的后缀。
//各自的网络。检查旧数据库并将其更新到
//相应地，0.2.0版引入了新的路径。
	oldDbRoot := filepath.Join(oldBtcdHomeDir(), "db")
	upgradeDBPathNet(filepath.Join(oldDbRoot, "btcd.db"), "mainnet")
	upgradeDBPathNet(filepath.Join(oldDbRoot, "btcd_testnet.db"), "testnet")
	upgradeDBPathNet(filepath.Join(oldDbRoot, "btcd_regtest.db"), "regtest")

//删除旧的db目录。
	return os.RemoveAll(oldDbRoot)
}

//upgradeDataPaths将应用程序数据从BTCD之前的位置移动
//版本0.3.3到其新位置。
func upgradeDataPaths() error {
//如果旧的和新的主路径相同，则无需迁移。
	oldHomePath := oldBtcdHomeDir()
	newHomePath := defaultHomeDir
	if oldHomePath == newHomePath {
		return nil
	}

//仅当旧路径存在而新路径不存在时迁移。
	if fileExists(oldHomePath) && !fileExists(newHomePath) {
//创建新路径。
		btcdLog.Infof("Migrating application home path from '%s' to '%s'",
			oldHomePath, newHomePath)
		err := os.MkdirAll(newHomePath, 0700)
		if err != nil {
			return err
		}

//如果需要，将旧btcd.conf移到新位置。
		oldConfPath := filepath.Join(oldHomePath, defaultConfigFilename)
		newConfPath := filepath.Join(newHomePath, defaultConfigFilename)
		if fileExists(oldConfPath) && !fileExists(newConfPath) {
			err := os.Rename(oldConfPath, newConfPath)
			if err != nil {
				return err
			}
		}

//如果需要，将旧数据目录移到新位置。
		oldDataPath := filepath.Join(oldHomePath, defaultDataDirname)
		newDataPath := filepath.Join(newHomePath, defaultDataDirname)
		if fileExists(oldDataPath) && !fileExists(newDataPath) {
			err := os.Rename(oldDataPath, newDataPath)
			if err != nil {
				return err
			}
		}

//如果旧房子是空的，就把它搬走；如果不是空的，就发出警告。
		ohpEmpty, err := dirEmpty(oldHomePath)
		if err != nil {
			return err
		}
		if ohpEmpty {
			err := os.Remove(oldHomePath)
			if err != nil {
				return err
			}
		} else {
			btcdLog.Warnf("Not removing '%s' since it contains files "+
				"not created by this application.  You may "+
				"want to manually move them or delete them.",
				oldHomePath)
		}
	}

	return nil
}

//doupgrades根据新版本的需要对btcd进行升级。
func doUpgrades() error {
	err := upgradeDBPaths()
	if err != nil {
		return err
	}
	return upgradeDataPaths()
}
