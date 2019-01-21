
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package rpctest

import (
	"errors"
	"math"
	"math/big"
	"runtime"
	"time"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

//SolveBlock尝试查找一个使传递的块头散列的nonce
//小于目标难度的值。当成功的解决方案是
//返回found true并更新传递的头的nonce字段
//解决方案。如果不存在解决方案，则返回false。
func solveBlock(header *wire.BlockHeader, targetDifficulty *big.Int) bool {
//解算器goroutine使用sbresult发送结果。
	type sbResult struct {
		found bool
		nonce uint32
	}

//解算器接受要测试的块头和非ce范围。它是
//打算作为一个野人来运作。
	quit := make(chan bool)
	results := make(chan sbResult)
	solver := func(hdr wire.BlockHeader, startNonce, stopNonce uint32) {
//我们需要修改标题的nonce字段，因此请确保
//我们使用原始标题的副本。
		for i := startNonce; i >= startNonce && i <= stopNonce; i++ {
			select {
			case <-quit:
				return
			default:
				hdr.Nonce = i
				hash := hdr.BlockHash()
				if blockchain.HashToBig(&hash).Cmp(targetDifficulty) <= 0 {
					select {
					case results <- sbResult{true, i}:
						return
					case <-quit:
						return
					}
				}
			}
		}
		select {
		case results <- sbResult{false, 0}:
		case <-quit:
			return
		}
	}

	startNonce := uint32(0)
	stopNonce := uint32(math.MaxUint32)
	numCores := uint32(runtime.NumCPU())
	noncesPerCore := (stopNonce - startNonce) / numCores
	for i := uint32(0); i < numCores; i++ {
		rangeStart := startNonce + (noncesPerCore * i)
		rangeStop := startNonce + (noncesPerCore * (i + 1)) - 1
		if i == numCores-1 {
			rangeStop = stopNonce
		}
		go solver(*header, rangeStart, rangeStop)
	}
	for i := uint32(0); i < numCores; i++ {
		result := <-results
		if result.found {
			close(quit)
			header.Nonce = result.nonce
			return true
		}
	}

	return false
}

//StandardCoinBaseScript返回适合用作
//新块的CoinBase事务的签名脚本。特别地，
//它以版本2块所需的块高度开始。
func standardCoinbaseScript(nextBlockHeight int32, extraNonce uint64) ([]byte, error) {
	return txscript.NewScriptBuilder().AddInt64(int64(nextBlockHeight)).
		AddInt64(int64(extraNonce)).Script()
}

//createCoinBaseTx返回一个支付适当
//根据所提供地址通过的街区高度给予补贴。
func createCoinbaseTx(coinbaseScript []byte, nextBlockHeight int32,
	addr btcutil.Address, mineTo []wire.TxOut,
	net *chaincfg.Params) (*btcutil.Tx, error) {

//创建脚本以支付到提供的支付地址。
	pkScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		return nil, err
	}

	tx := wire.NewMsgTx(wire.TxVersion)
	tx.AddTxIn(&wire.TxIn{
//CoinBase事务没有输入，因此以前的输出点是
//零哈希和最大索引。
		PreviousOutPoint: *wire.NewOutPoint(&chainhash.Hash{},
			wire.MaxPrevOutIndex),
		SignatureScript: coinbaseScript,
		Sequence:        wire.MaxTxInSequenceNum,
	})
	if len(mineTo) == 0 {
		tx.AddTxOut(&wire.TxOut{
			Value:    blockchain.CalcBlockSubsidy(nextBlockHeight, net),
			PkScript: pkScript,
		})
	} else {
		for i := range mineTo {
			tx.AddTxOut(&mineTo[i])
		}
	}
	return btcutil.NewTx(tx), nil
}

//CreateBlock使用
//指定的块版本和时间戳。如果传递的时间戳为零（不是
//初始化），则使用上一个块的时间戳加上1
//使用第二个。为上一个块传递nil将导致
//为指定链构建Genesis块。
func CreateBlock(prevBlock *btcutil.Block, inclusionTxs []*btcutil.Tx,
	blockVersion int32, blockTime time.Time, miningAddr btcutil.Address,
	mineTo []wire.TxOut, net *chaincfg.Params) (*btcutil.Block, error) {

	var (
		prevHash      *chainhash.Hash
		blockHeight   int32
		prevBlockTime time.Time
	)

//如果没有指定前一个块，那么我们将构造一个块
//这是在Genesis区块的基础上建立起来的。
	if prevBlock == nil {
		prevHash = net.GenesisHash
		blockHeight = 1
		prevBlockTime = net.GenesisBlock.Header.Timestamp.Add(time.Minute)
	} else {
		prevHash = prevBlock.Hash()
		blockHeight = prevBlock.Height() + 1
		prevBlockTime = prevBlock.MsgBlock().Header.Timestamp
	}

//如果指定了目标块时间，则将其用作标题
//时间戳。否则，请向上一个块添加一秒钟，除非
//在这种情况下，它是Genesis块，使用当前时间。
	var ts time.Time
	switch {
	case !blockTime.IsZero():
		ts = blockTime
	default:
		ts = prevBlockTime.Add(time.Second)
	}

	extraNonce := uint64(0)
	coinbaseScript, err := standardCoinbaseScript(blockHeight, extraNonce)
	if err != nil {
		return nil, err
	}
	coinbaseTx, err := createCoinbaseTx(coinbaseScript, blockHeight,
		miningAddr, mineTo, net)
	if err != nil {
		return nil, err
	}

//创建一个准备解决的新块。
	blockTxns := []*btcutil.Tx{coinbaseTx}
	if inclusionTxs != nil {
		blockTxns = append(blockTxns, inclusionTxs...)
	}
	merkles := blockchain.BuildMerkleTreeStore(blockTxns, false)
	var block wire.MsgBlock
	block.Header = wire.BlockHeader{
		Version:    blockVersion,
		PrevBlock:  *prevHash,
		MerkleRoot: *merkles[len(merkles)-1],
		Timestamp:  ts,
		Bits:       net.PowLimitBits,
	}
	for _, tx := range blockTxns {
		if err := block.AddTransaction(tx.MsgTx()); err != nil {
			return nil, err
		}
	}

	found := solveBlock(&block.Header, net.PowLimit)
	if !found {
		return nil, errors.New("Unable to solve block")
	}

	utilBlock := btcutil.NewBlock(&block)
	utilBlock.SetHeight(blockHeight)
	return utilBlock, nil
}
