
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

package txscript

import (
	"sync"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

//SigCacheEntry表示SigCache中的一个条目。中的条目
//sigcache是根据签名的叹息键控的。在
//缓存命中的场景（根据sighash），另一个比较
//将执行公钥以确保
//比赛。当两个叹息碰撞时，新的叹息会
//只需覆盖现有条目。
type sigCacheEntry struct {
	sig    *btcec.Signature
	pubKey *btcec.PublicKey
}

//sigcache使用随机的
//出入境驱逐政策。只有有效的签名才会添加到缓存中。这个
//SigCache的好处是双重的。首先，使用sigcache可以减轻DoS
//由于最坏情况，攻击导致受害者客户挂起的攻击
//处理攻击者编写的无效事务时触发的行为。一
//缓解的DoS攻击的详细描述如下：
//https://bitslog.wordpress.com/2013/01/23/fixed-bitcoin-viability-explation-why-the-signature-cache-is-a-dos-protection/。
//其次，使用sigcache引入了签名验证
//优化加快了块内事务的验证，
//如果已经在mempool中看到并验证了它们。
type SigCache struct {
	sync.RWMutex
	validSigs  map[chainhash.Hash]sigCacheEntry
	maxEntries uint
}

//new sigcache创建并初始化sigcache的新实例。它唯一
//参数“maxentries”表示允许的最大条目数
//在任何特定时刻都存在于sigcache中。随机条目被逐出
//为可能导致
//缓存超过最大值。
func NewSigCache(maxEntries uint) *SigCache {
	return &SigCache{
		validSigs:  make(map[chainhash.Hash]sigCacheEntry, maxEntries),
		maxEntries: maxEntries,
	}
}

//如果公共的“sigash”上存在“sig”的条目，则exists返回true
//在sigcache中找到键“pubkey”。否则，返回false。
//
//注意：此函数对于并发访问是安全的。不会阻止读者
//除非存在写入程序，否则将条目添加到sigcache。
func (s *SigCache) Exists(sigHash chainhash.Hash, sig *btcec.Signature, pubKey *btcec.PublicKey) bool {
	s.RLock()
	entry, ok := s.validSigs[sigHash]
	s.RUnlock()

	return ok && entry.pubKey.IsEqual(pubKey) && entry.sig.IsEqual(sig)
}

//添加在公钥“pubkey”下的“sighash”上添加签名条目
//到签名缓存。如果sigcache为“满”，则
//为了给现有条目留出空间，随机选择要收回的条目。
//新条目。
//
//注意：此函数对于并发访问是安全的。作者将阻止
//同时读卡器，直到函数执行结束。
func (s *SigCache) Add(sigHash chainhash.Hash, sig *btcec.Signature, pubKey *btcec.PublicKey) {
	s.Lock()
	defer s.Unlock()

	if s.maxEntries <= 0 {
		return
	}

//如果添加此新条目将使我们超过允许的最大数量
//条目，然后逐出一个条目。
	if uint(len(s.validSigs)+1) > s.maxEntries {
//从地图中删除一个随机条目。依靠随机
//Go地图迭代的起点。值得注意的是
//随机迭代起始点不能100%保证
//按照规范，但是大多数go编译器都支持它。
//最终，迭代顺序在这里并不重要，因为
//为了操纵被驱逐的物品，敌人
//需要能够对
//哈希函数，以便在特定的
//条目。
		for sigEntry := range s.validSigs {
			delete(s.validSigs, sigEntry)
			break
		}
	}
	s.validSigs[sigHash] = sigCacheEntry{sig, pubKey}
}
