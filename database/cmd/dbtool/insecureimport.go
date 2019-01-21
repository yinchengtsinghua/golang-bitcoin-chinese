
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
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

//importCmd定义不安全导入命令的配置选项。
type importCmd struct {
	InFile   string `short:"i" long:"infile" description:"File containing the block(s)"`
	Progress int    `short:"p" long:"progress" description:"Show a progress message each time this number of seconds have passed -- Use 0 to disable progress announcements"`
}

var (
//importcfg定义命令的配置选项。
	importCfg = importCmd{
		InFile:   "bootstrap.dat",
		Progress: 10,
	}

//zerohash只是一个包含所有零的哈希。此处定义为
//避免多次创建。
	zeroHash = chainhash.Hash{}
)

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
//被跳过，孤立块被视为错误。返回是否
//块与任何潜在错误一起导入。
//
//注意：这不是安全导入，因为它不验证链规则。
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
	var exists bool
	err = bi.db.View(func(tx database.Tx) error {
		exists, err = tx.HasBlock(block.Hash())
		return err
	})
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}

//不要费心去处理孤儿。
	prevHash := &block.MsgBlock().Header.PrevBlock
	if !prevHash.IsEqual(&zeroHash) {
		var exists bool
		err := bi.db.View(func(tx database.Tx) error {
			exists, err = tx.HasBlock(prevHash)
			return err
		})
		if err != nil {
			return false, err
		}
		if !exists {
			return false, fmt.Errorf("import file contains block "+
				"%v which does not link to the available "+
				"block chain", prevHash)
		}
	}

//在不检查链规则的情况下将块放入数据库。
	err = bi.db.Update(func(tx database.Tx) error {
		return tx.StoreBlock(block)
	})
	if err != nil {
		return false, err
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
//防止垃圾邮件，它将每次importcfg.progress只记录一条消息
//秒，包括持续时间和总数。
func (bi *blockImporter) logProgress() {
	bi.receivedLogBlocks++

	now := time.Now()
	duration := now.Sub(bi.lastLogTime)
	if duration < time.Second*time.Duration(importCfg.Progress) {
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

//processhandler是处理块的主要处理程序。这允许阻止
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

//statusHandler等待导入操作的更新并通知
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

//导入正常完成。
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

//启动状态处理程序并返回它将
//导入完成后发送结果。
	resultChan := make(chan *importResults)
	go bi.statusHandler(resultChan)
	return resultChan
}

//newblockimporter为提供的文件读取器seeker返回新的导入程序
//和数据库。
func newBlockImporter(db database.DB, r io.ReadSeeker) *blockImporter {
	return &blockImporter{
		db:           db,
		r:            r,
		processQueue: make(chan []byte, 2),
		doneChan:     make(chan bool),
		errChan:      make(chan error),
		quit:         make(chan struct{}),
		lastLogTime:  time.Now(),
	}
}

//执行是命令的主要入口点。它由解析器调用。
func (cmd *importCmd) Execute(args []string) error {
//设置全局配置选项并确保它们有效。
	if err := setupGlobalConfig(); err != nil {
		return err
	}

//确保指定的块文件存在。
	if !fileExists(cmd.InFile) {
		str := "The specified block file [%v] does not exist"
		return fmt.Errorf(str, cmd.InFile)
	}

//加载块数据库。
	db, err := loadBlockDB()
	if err != nil {
		return err
	}
	defer db.Close()

//确保数据库在ctrl+c上同步并关闭。
	addInterruptHandler(func() {
		log.Infof("Gracefully shutting down the database...")
		db.Close()
	})

	fi, err := os.Open(importCfg.InFile)
	if err != nil {
		return err
	}
	defer fi.Close()

//为数据库和输入文件创建一个块导入程序并启动它。
//从开始返回的结果通道将包含一个错误，如果
//出了什么问题。
	importer := newBlockImporter(db, fi)

//异步执行导入，并在
//完成。这允许并行处理和读取块。这个
//从导入返回的结果通道包含有关
//导入包括出错时的错误。这样做了
//在一个单独的Goroutine而不是直接等待
//Goroutine可以通过完成、错误和
//或者来自主中断处理程序。这是必要的，因为
//Goroutine必须保持足够长的时间运行，以供中断处理程序使用。
//Goroutine完成。
	go func() {
		log.Info("Starting import")
		resultsChan := importer.Import()
		results := <-resultsChan
		if results.err != nil {
			dbErr, ok := results.err.(database.Error)
			if !ok || ok && dbErr.ErrorCode != database.ErrDbNotOpen {
				shutdownChannel <- results.err
				return
			}
		}

		log.Infof("Processed a total of %d blocks (%d imported, %d "+
			"already known)", results.blocksProcessed,
			results.blocksImported,
			results.blocksProcessed-results.blocksImported)
		shutdownChannel <- nil
	}()

//等待正常完成或
//中断处理程序。
	err = <-shutdownChannel
	return err
}
