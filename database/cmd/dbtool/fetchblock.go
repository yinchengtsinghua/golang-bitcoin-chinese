
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
	"encoding/hex"
	"errors"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/database"
)

//fetchblockCmd定义fetchblock命令的配置选项。
type fetchBlockCmd struct{}

var (
//fetchblockcfg定义命令的配置选项。
	fetchBlockCfg = fetchBlockCmd{}
)

//执行是命令的主要入口点。它由解析器调用。
func (cmd *fetchBlockCmd) Execute(args []string) error {
//设置全局配置选项并确保它们有效。
	if err := setupGlobalConfig(); err != nil {
		return err
	}

	if len(args) < 1 {
		return errors.New("required block hash parameter not specified")
	}
	blockHash, err := chainhash.NewHashFromStr(args[0])
	if err != nil {
		return err
	}

//加载块数据库。
	db, err := loadBlockDB()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.View(func(tx database.Tx) error {
		log.Infof("Fetching block %s", blockHash)
		startTime := time.Now()
		blockBytes, err := tx.FetchBlock(blockHash)
		if err != nil {
			return err
		}
		log.Infof("Loaded block in %v", time.Since(startTime))
		log.Infof("Block Hex: %s", hex.EncodeToString(blockBytes))
		return nil
	})
}

//用法覆盖命令的用法显示。
func (cmd *fetchBlockCmd) Usage() string {
	return "<block-hash>"
}
