
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package txscript

import (
	"math/rand"
	"testing"
	"time"

	"github.com/btcsuite/btcd/wire"
	"github.com/davecgh/go-spew/spew"
)

//gentestx为在测试用例中使用创建一个随机事务。
func genTestTx() (*wire.MsgTx, error) {
	tx := wire.NewMsgTx(2)
	tx.Version = rand.Int31()

	numTxins := rand.Intn(11)
	for i := 0; i < numTxins; i++ {
		randTxIn := wire.TxIn{
			PreviousOutPoint: wire.OutPoint{
				Index: uint32(rand.Int31()),
			},
			Sequence: uint32(rand.Int31()),
		}
		_, err := rand.Read(randTxIn.PreviousOutPoint.Hash[:])
		if err != nil {
			return nil, err
		}

		tx.TxIn = append(tx.TxIn, &randTxIn)
	}

	numTxouts := rand.Intn(11)
	for i := 0; i < numTxouts; i++ {
		randTxOut := wire.TxOut{
			Value:    rand.Int63(),
			PkScript: make([]byte, rand.Intn(30)),
		}
		if _, err := rand.Read(randTxOut.PkScript); err != nil {
			return nil, err
		}
		tx.TxOut = append(tx.TxOut, &randTxOut)
	}

	return tx, nil
}

//testhashcacheaddcontainsHash测试将项添加到
//哈希缓存，containsHashes方法为所有项返回true
//插入的。相反，containsHashes对于任何项都应返回false
//不在哈希缓存中。
func TestHashCacheAddContainsHashes(t *testing.T) {
	t.Parallel()

	rand.Seed(time.Now().Unix())

	cache := NewHashCache(10)

	var err error

//首先，我们将生成10个随机事务用于
//测验。
	const numTxns = 10
	txns := make([]*wire.MsgTx, numTxns)
	for i := 0; i < numTxns; i++ {
		txns[i], err = genTestTx()
		if err != nil {
			t.Fatalf("unable to generate test tx: %v", err)
		}
	}

//生成事务后，我们将把它们中的每一个添加到哈希中
//隐藏物。
	for _, tx := range txns {
		cache.AddSigHashes(tx)
	}

//接下来，我们将确保在
//通过containsHashes方法正确定位缓存。
	for _, tx := range txns {
		txid := tx.TxHash()
		if ok := cache.ContainsHashes(&txid); !ok {
			t.Fatalf("txid %v not found in cache but should be: ",
				txid)
		}
	}

	randTx, err := genTestTx()
	if err != nil {
		t.Fatalf("unable to generate tx: %v", err)
	}

//最后，我们将断言没有添加到
//包含哈希将不会报告缓存存在。
//方法。
	randTxid := randTx.TxHash()
	if ok := cache.ContainsHashes(&randTxid); ok {
		t.Fatalf("txid %v wasn't inserted into cache but was found",
			randTxid)
	}
}

//testHashcacheAddget测试特定事务的叹息
//由GetSightAsh函数正确检索。
func TestHashCacheAddGet(t *testing.T) {
	t.Parallel()

	rand.Seed(time.Now().Unix())

	cache := NewHashCache(10)

//首先，我们将生成一个随机事务并计算
//为交易叹息。
	randTx, err := genTestTx()
	if err != nil {
		t.Fatalf("unable to generate tx: %v", err)
	}
	sigHashes := NewTxSigHashes(randTx)

//接下来，将事务添加到哈希缓存中。
	cache.AddSigHashes(randTx)

//应该找到插入到上面缓存中的事务。
	txid := randTx.TxHash()
	cacheHashes, ok := cache.GetSigHashes(&txid)
	if !ok {
		t.Fatalf("tx %v wasn't found in cache", txid)
	}

//最后，检索到的叹息应该与叹息完全匹配。
//最初插入到缓存中。
	if *sigHashes != *cacheHashes {
		t.Fatalf("sighashes don't match: expected %v, got %v",
			spew.Sdump(sigHashes), spew.Sdump(cacheHashes))
	}
}

//testhashcachepurge测试可以从
//哈希缓存。
func TestHashCachePurge(t *testing.T) {
	t.Parallel()

	rand.Seed(time.Now().Unix())

	cache := NewHashCache(10)

	var err error

//首先，我们将从将numtxns事务插入哈希缓存开始。
	const numTxns = 10
	txns := make([]*wire.MsgTx, numTxns)
	for i := 0; i < numTxns; i++ {
		txns[i], err = genTestTx()
		if err != nil {
			t.Fatalf("unable to generate test tx: %v", err)
		}
	}
	for _, tx := range txns {
		cache.AddSigHashes(tx)
	}

//插入所有事务后，我们将从
//哈希缓存。
	for _, tx := range txns {
		txid := tx.TxHash()
		cache.PurgeSigHashes(&txid)
	}

//此时，没有任何事务插入到哈希缓存中
//应该在缓存中找到。
	for _, tx := range txns {
		txid := tx.TxHash()
		if ok := cache.ContainsHashes(&txid); ok {
			t.Fatalf("tx %v found in cache but should have "+
				"been purged: ", txid)
		}
	}
}
