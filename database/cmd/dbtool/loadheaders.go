
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
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/database"
)

//headerscmd定义loadheaders命令的配置选项。
type headersCmd struct {
	Bulk bool `long:"bulk" description:"Use bulk loading of headers instead of one at a time"`
}

var (
//headerscfg定义命令的配置选项。
	headersCfg = headersCmd{
		Bulk: false,
	}
)

//执行是命令的主要入口点。它由解析器调用。
func (cmd *headersCmd) Execute(args []string) error {
//设置全局配置选项并确保它们有效。
	if err := setupGlobalConfig(); err != nil {
		return err
	}

//加载块数据库。
	db, err := loadBlockDB()
	if err != nil {
		return err
	}
	defer db.Close()

//注意：此代码仅适用于ffldb。理想情况下，包使用
//数据库将保留自己的元数据索引。
	blockIdxName := []byte("ffldb-blockidx")
	if !headersCfg.Bulk {
		err = db.View(func(tx database.Tx) error {
			totalHdrs := 0
			blockIdxBucket := tx.Metadata().Bucket(blockIdxName)
			blockIdxBucket.ForEach(func(k, v []byte) error {
				totalHdrs++
				return nil
			})
			log.Infof("Loading headers for %d blocks...", totalHdrs)
			numLoaded := 0
			startTime := time.Now()
			blockIdxBucket.ForEach(func(k, v []byte) error {
				var hash chainhash.Hash
				copy(hash[:], k)
				_, err := tx.FetchBlockHeader(&hash)
				if err != nil {
					return err
				}
				numLoaded++
				return nil
			})
			log.Infof("Loaded %d headers in %v", numLoaded,
				time.Since(startTime))
			return nil
		})
		return err
	}

//大容量加载标题。
	err = db.View(func(tx database.Tx) error {
		blockIdxBucket := tx.Metadata().Bucket(blockIdxName)
		hashes := make([]chainhash.Hash, 0, 500000)
		blockIdxBucket.ForEach(func(k, v []byte) error {
			var hash chainhash.Hash
			copy(hash[:], k)
			hashes = append(hashes, hash)
			return nil
		})

		log.Infof("Loading headers for %d blocks...", len(hashes))
		startTime := time.Now()
		hdrs, err := tx.FetchBlockHeaders(hashes)
		if err != nil {
			return err
		}
		log.Infof("Loaded %d headers in %v", len(hdrs),
			time.Since(startTime))
		return nil
	})
	return err
}
