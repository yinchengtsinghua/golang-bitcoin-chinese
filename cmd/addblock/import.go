
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
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/blockchain/indexers"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

var zeroHash = chainhash.Hash{}

//importResults将状态和结果作为导入操作存储。
type importResults struct {
	blocksProcessed int64
	blocksImported  int64
	err             error
}

//块导入程序包含有关正在从块数据导入的信息
//文件到块数据库。
type blockImporter struct {
	db                database.DB
	chain             *blockchain.BlockChain
	r                 io.ReadSeeker
	processQueue      chan []byte
	doneChan          chan bool
	errChan           chan error
	quit              chan struct{}
	wg                sync.WaitGroup
	blocksProcessed   int64
	blocksImported    int64
	receivedLogBlocks int64
	receivedLogTx     int64
	lastHeight        int64
	lastBlockTime     time.Time
	lastLogTime       time.Time
}

//readblock从输入文件读取下一个块。
func (bi *blockImporter) readBlock() ([]byte, error) {
//块文件格式为：
//<network><block length><serialized block>
	var net uint32
	err := binary.Read(bi.r, binary.LittleEndian, &net)
	if err != nil {
		if err != io.EOF {
			return nil, err
		}

//没有块和错误意味着没有更多的块可以读取。
		return nil, nil
	}
	if net != uint32(activeNetParams.Net) {
		return nil, fmt.Errorf("network mismatch -- got %x, want %x",
			net, uint32(activeNetParams.Net))
	}

//读取块长度并确保其正常。
	var blockLen uint32
	if err := binary.Read(bi.r, binary.LittleEndian, &blockLen); err != nil {
		return nil, err
	}
	if blockLen > wire.MaxBlockPayload {
		return nil, fmt.Errorf("block payload of %d bytes is larger "+
			"than the max allowed %d bytes", blockLen,
			wire.MaxBlockPayload)
	}

	serializedBlock := make([]byte, blockLen)
	if _, err := io.ReadFull(bi.r, serializedBlock); err != nil {
		return nil, err
	}

	return serializedBlock, nil
}

//processBlock可能会将块导入数据库。它首先
//在检查错误时反序列化原始块。已知块
//被跳过，孤立块被视为错误。最后，它运行
//阻塞链规则以确保它遵循所有规则和匹配
//到已知的检查点。返回块是否随导入
//有任何潜在的错误。
func (bi *blockImporter) processBlock(serializedBlock []byte) (bool, error) {
//反序列化块，其中包括对格式错误的块的检查。
	block, err := btcutil.NewBlockFromBytes(serializedBlock)
	if err != nil {
		return false, err
	}

//更新进度统计信息
	bi.lastBlockTime = block.MsgBlock().Header.Timestamp
	bi.receivedLogTx += int64(len(block.MsgBlock().Transactions))

//跳过已经存在的块。
	blockHash := block.Hash()
	exists, err := bi.chain.HaveBlock(blockHash)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}

//不要费心去处理孤儿。
	prevHash := &block.MsgBlock().Header.PrevBlock
	if !prevHash.IsEqual(&zeroHash) {
		exists, err := bi.chain.HaveBlock(prevHash)
		if err != nil {
			return false, err
		}
		if !exists {
			return false, fmt.Errorf("import file contains block "+
				"%v which does not link to the available "+
				"block chain", prevHash)
		}
	}

//确保块遵循所有链规则并与
//已知的检查点。
	isMainChain, isOrphan, err := bi.chain.ProcessBlock(block,
		blockchain.BFFastAdd)
	if err != nil {
		return false, err
	}
	if !isMainChain {
		return false, fmt.Errorf("import file contains an block that "+
			"does not extend the main chain: %v", blockHash)
	}
	if isOrphan {
		return false, fmt.Errorf("import file contains an orphan "+
			"block: %v", blockHash)
	}

	return true, nil
}

//readhandler是从导入文件读取块的主要处理程序。
//这允许块处理与块读取并行进行。
//它必须像野人一样运作。
func (bi *blockImporter) readHandler() {
out:
	for {
//从文件中读取下一个块，如果有任何问题
//通知状态处理程序错误并保释。
		serializedBlock, err := bi.readBlock()
		if err != nil {
			bi.errChan <- fmt.Errorf("Error reading from input "+
				"file: %v", err.Error())
			break out
		}

//一个没有错误的零块意味着我们结束了。
		if serializedBlock == nil {
			break out
		}

//如果有信号指示我们退出，请发送阻止或退出
//由于其他地方出错而导致的状态处理程序。
		select {
		case bi.processQueue <- serializedBlock:
		case <-bi.quit:
			break out
		}
	}

//关闭处理通道，以表示不再有数据块出现。
	close(bi.processQueue)
	bi.wg.Done()
}

//logprogress以信息消息的形式记录阻止进度。为了
//防止垃圾邮件，它将日志记录限制为每cfg.progress秒记录一条消息。
//包括持续时间和总数。
func (bi *blockImporter) logProgress() {
	bi.receivedLogBlocks++

	now := time.Now()
	duration := now.Sub(bi.lastLogTime)
	if duration < time.Second*time.Duration(cfg.Progress) {
		return
	}

//将持续时间截断为10毫秒。
	durationMillis := int64(duration / time.Millisecond)
	tDuration := 10 * time.Millisecond * time.Duration(durationMillis/10)

//有关新块高度的日志信息。
	blockStr := "blocks"
	if bi.receivedLogBlocks == 1 {
		blockStr = "block"
	}
	txStr := "transactions"
	if bi.receivedLogTx == 1 {
		txStr = "transaction"
	}
	log.Infof("Processed %d %s in the last %s (%d %s, height %d, %s)",
		bi.receivedLogBlocks, blockStr, tDuration, bi.receivedLogTx,
		txStr, bi.lastHeight, bi.lastBlockTime)

	bi.receivedLogBlocks = 0
	bi.receivedLogTx = 0
	bi.lastLogTime = now
}

//processHandler is the main handler for processing blocks.  This allows block
//与导入文件中的块读取并行进行的处理。
//它必须像野人一样运作。
func (bi *blockImporter) processHandler() {
out:
	for {
		select {
		case serializedBlock, ok := <-bi.processQueue:
//频道关闭时我们就结束了。
			if !ok {
				break out
			}

			bi.blocksProcessed++
			bi.lastHeight++
			imported, err := bi.processBlock(serializedBlock)
			if err != nil {
				bi.errChan <- err
				break out
			}

			if imported {
				bi.blocksImported++
			}

			bi.logProgress()

		case <-bi.quit:
			break out
		}
	}
	bi.wg.Done()
}

//statusHandler waits for updates from the import operation and notifies
//传递的Donechan和导入结果。它也导致所有
//Goroutines在其中任何一个报告错误时退出。
func (bi *blockImporter) statusHandler(resultsChan chan *importResults) {
	select {
//任何一个Goroutines的错误都意味着我们已经完成了这样的信号
//有错误的呼叫者，并向所有Goroutine发出退出信号。
	case err := <-bi.errChan:
		resultsChan <- &importResults{
			blocksProcessed: bi.blocksProcessed,
			blocksImported:  bi.blocksImported,
			err:             err,
		}
		close(bi.quit)

//The import finished normally.
	case <-bi.doneChan:
		resultsChan <- &importResults{
			blocksProcessed: bi.blocksProcessed,
			blocksImported:  bi.blocksImported,
			err:             nil,
		}
	}
}

//导入是处理从文件导入块的核心功能
//与数据库的块导入程序关联。它返回一个频道
//将在操作完成后返回结果。
func (bi *blockImporter) Import() chan *importResults {
//启动读取和处理goroutine。此设置允许
//处理时并行从磁盘读取的块。
	bi.wg.Add(2)
	go bi.readHandler()
	go bi.processHandler()

//等待导入在单独的goroutine和信号中完成
//完成后的状态处理程序。
	go func() {
		bi.wg.Wait()
		bi.doneChan <- true
	}()

//Start the status handler and return the result channel that it will
//导入完成后发送结果。
	resultChan := make(chan *importResults)
	go bi.statusHandler(resultChan)
	return resultChan
}

//newblockimporter为提供的文件读取器seeker返回新的导入程序
//和数据库。
func newBlockImporter(db database.DB, r io.ReadSeeker) (*blockImporter, error) {
//根据需要创建事务和地址索引。
//
//注意：在索引数组中，txindex必须是第一个，因为
//在捕获过程中，addrindex使用来自txindex的数据。如果
//首先运行addrindex，它可能没有来自
//当前块已索引。
	var indexes []indexers.Indexer
	if cfg.TxIndex || cfg.AddrIndex {
//如果启用了地址索引，则启用事务索引，因为它
//需要它。
		if !cfg.TxIndex {
			log.Infof("Transaction index enabled because it is " +
				"required by the address index")
			cfg.TxIndex = true
		} else {
			log.Info("Transaction index is enabled")
		}
		indexes = append(indexes, indexers.NewTxIndex(db))
	}
	if cfg.AddrIndex {
		log.Info("Address index is enabled")
		indexes = append(indexes, indexers.NewAddrIndex(db, activeNetParams))
	}

//如果启用了任何可选索引，则创建索引管理器。
	var indexManager blockchain.IndexManager
	if len(indexes) > 0 {
		indexManager = indexers.NewManager(db, indexes)
	}

	chain, err := blockchain.New(&blockchain.Config{
		DB:           db,
		ChainParams:  activeNetParams,
		TimeSource:   blockchain.NewMedianTime(),
		IndexManager: indexManager,
	})
	if err != nil {
		return nil, err
	}

	return &blockImporter{
		db:           db,
		r:            r,
		processQueue: make(chan []byte, 2),
		doneChan:     make(chan bool),
		errChan:      make(chan error),
		quit:         make(chan struct{}),
		chain:        chain,
		lastLogTime:  time.Now(),
	}, nil
}
