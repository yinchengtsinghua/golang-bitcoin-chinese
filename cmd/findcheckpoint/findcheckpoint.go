
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
	"os"
	"path/filepath"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/database"
)

const blockDbNamePrefix = "blocks"

var (
	cfg *config
)

//loadblockdb打开块数据库并返回其句柄。
func loadBlockDB() (database.DB, error) {
//数据库名称基于数据库类型。
	dbName := blockDbNamePrefix + "_" + cfg.DbType
	dbPath := filepath.Join(cfg.DataDir, dbName)
	fmt.Printf("Loading block database from '%s'\n", dbPath)
	db, err := database.Open(cfg.DbType, dbPath, activeNetParams.Net)
	if err != nil {
		return nil, err
	}
	return db, nil
}

//findcandidates向后搜索链中的候选检查点和
//返回找到的候选项的切片（如果有）。它也停止搜索
//已硬编码为btchain的最后一个检查点的候选
//因为在已经存在的情况下找候选人毫无意义
//检查点。
func findCandidates(chain *blockchain.BlockChain, latestHash *chainhash.Hash) ([]*chaincfg.Checkpoint, error) {
//从最新的主链块开始。
	block, err := chain.BlockByHash(latestHash)
	if err != nil {
		return nil, err
	}

//获取最新的已知检查点。
	latestCheckpoint := chain.LatestCheckpoint()
	if latestCheckpoint == nil {
//如果没有，设置最新的检查点到Genesis块
//已经有一个了。
		latestCheckpoint = &chaincfg.Checkpoint{
			Hash:   activeNetParams.GenesisHash,
			Height: 0,
		}
	}

//最新的已知块必须至少是最后一个已知检查点
//加上必要的检查点确认。
	checkpointConfirmations := int32(blockchain.CheckpointConfirmations)
	requiredHeight := latestCheckpoint.Height + checkpointConfirmations
	if block.Height() < requiredHeight {
		return nil, fmt.Errorf("the block database is only at height "+
			"%d which is less than the latest checkpoint height "+
			"of %d plus required confirmations of %d",
			block.Height(), latestCheckpoint.Height,
			checkpointConfirmations)
	}

//对于第一个检查点，所需高度为
//Genesis区块，只要链至少有所需数量
//确认书（上面强制执行）。
	if len(activeNetParams.Checkpoints) == 0 {
		requiredHeight = 1
	}

//不确定的进度设置。
	numBlocksToTest := block.Height() - requiredHeight
progressInterval := (numBlocksToTest / 100) + 1 //闽1
	fmt.Print("Searching for candidates")
	defer fmt.Println()

//在链中向后循环以查找候选检查点。
	candidates := make([]*chaincfg.Checkpoint, 0, cfg.NumCandidates)
	numTested := int32(0)
	for len(candidates) < cfg.NumCandidates && block.Height() > requiredHeight {
//显示进度。
		if numTested%progressInterval == 0 {
			fmt.Print(".")
		}

//确定此块是否为候选检查点。
		isCandidate, err := chain.IsCheckpointCandidate(block)
		if err != nil {
			return nil, err
		}

//所有检查都已通过，因此此节点似乎是合理的
//检查点候选。
		if isCandidate {
			checkpoint := chaincfg.Checkpoint{
				Height: block.Height(),
				Hash:   block.Hash(),
			}
			candidates = append(candidates, &checkpoint)
		}

		prevHash := &block.MsgBlock().Header.PrevBlock
		block, err = chain.BlockByHash(prevHash)
		if err != nil {
			return nil, err
		}
		numTested++
	}
	return candidates, nil
}

//showcandidate使用和输出格式显示候选检查点
//determined by the configuration parameters.  The Go syntax output
//使用btchain代码希望用于添加到列表中的检查点的格式。
func showCandidate(candidateNum int, checkpoint *chaincfg.Checkpoint) {
	if cfg.UseGoOutput {
		fmt.Printf("Candidate %d -- {%d, newShaHashFromStr(\"%v\")},\n",
			candidateNum, checkpoint.Height, checkpoint.Hash)
		return
	}

	fmt.Printf("Candidate %d -- Height: %d, Hash: %v\n", candidateNum,
		checkpoint.Height, checkpoint.Hash)

}

func main() {
//加载配置并分析命令行。
	tcfg, _, err := loadConfig()
	if err != nil {
		return
	}
	cfg = tcfg

//加载块数据库。
	db, err := loadBlockDB()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load database:", err)
		return
	}
	defer db.Close()

//设置链。忽略通知，因为不需要通知
//UTIL
	chain, err := blockchain.New(&blockchain.Config{
		DB:          db,
		ChainParams: activeNetParams,
		TimeSource:  blockchain.NewMedianTime(),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize chain: %v\n", err)
		return
	}

//从数据库中获取最新的块哈希和高度并报告
//状态。
	best := chain.BestSnapshot()
	fmt.Printf("Block database loaded with block height %d\n", best.Height)

//查找候选检查站。
	candidates, err := findCandidates(chain, &best.Hash)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to identify candidates:", err)
		return
	}

//没有候选人。
	if len(candidates) == 0 {
		fmt.Println("No candidates found.")
		return
	}

//展示候选人。
	for i, checkpoint := range candidates {
		showCandidate(i+1, checkpoint)
	}
}
