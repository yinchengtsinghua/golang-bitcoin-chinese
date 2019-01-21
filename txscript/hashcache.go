
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

package txscript

import (
	"sync"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

//txsighash包含了在bip0143中引入的部分sighash集。
//此部分叹息集可以在每个输入中重新使用
//验证所有输入时的事务。因此，验证的复杂性
//因为叹息可以用多项式因子来减少。
type TxSigHashes struct {
	HashPrevOuts chainhash.Hash
	HashSequence chainhash.Hash
	HashOutputs  chainhash.Hash
}

//newtxsigash计算并返回给定的
//交易。
func NewTxSigHashes(tx *wire.MsgTx) *TxSigHashes {
	return &TxSigHashes{
		HashPrevOuts: calcHashPrevOuts(tx),
		HashSequence: calcHashSequence(tx),
		HashOutputs:  calcHashOutputs(tx),
	}
}

//hashcache包含一组由txid键控的部分叹息。部分集
//叹息是新的更有效的在BIP1043中引入的。
//sighash摘要计算算法。使用此线程安全共享缓存，
//多个goroutine可以安全地重复使用预先计算的部分瞄准具
//加快在一个块内找到的所有输入之间的验证时间。
type HashCache struct {
	sigHashes map[chainhash.Hash]*TxSigHashes

	sync.RWMutex
}

//new hashcache返回给定最大数目的hashcache的新实例
//随时可能存在于其中的条目。
func NewHashCache(maxSize uint) *HashCache {
	return &HashCache{
		sigHashes: make(map[chainhash.Hash]*TxSigHashes, maxSize),
	}
}

//addsighash计算，然后为传递的添加部分sighash
//交易。
func (h *HashCache) AddSigHashes(tx *wire.MsgTx) {
	h.Lock()
	h.sigHashes[tx.TxHash()] = NewTxSigHashes(tx)
	h.Unlock()
}

//如果传递的部分叹息，则containsHashes返回true
//事务当前存在于hashcache中，否则为false。
func (h *HashCache) ContainsHashes(txid *chainhash.Hash) bool {
	h.RLock()
	_, found := h.sigHashes[*txid]
	h.RUnlock()

	return found
}

//GetSightAsh可能返回以前缓存的部分SightAsh
//已传递的事务。此函数还返回一个附加的布尔值
//值，指示是否找到已传递事务的叹息
//出现在哈希缓存中。
func (h *HashCache) GetSigHashes(txid *chainhash.Hash) (*TxSigHashes, bool) {
	h.RLock()
	item, found := h.sigHashes[*txid]
	h.RUnlock()

	return item, found
}

//purgesighashes从属于
//已传递的事务。
func (h *HashCache) PurgeSigHashes(txid *chainhash.Hash) {
	h.Lock()
	delete(h.sigHashes, *txid)
	h.Unlock()
}
