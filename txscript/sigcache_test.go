
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
	"crypto/rand"
	"testing"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

//genrandomsig返回随机消息，消息的签名位于
//公钥和公钥。此函数用于生成随机
//测试数据。
func genRandomSig() (*chainhash.Hash, *btcec.Signature, *btcec.PublicKey, error) {
	privKey, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, nil, nil, err
	}

	var msgHash chainhash.Hash
	if _, err := rand.Read(msgHash[:]); err != nil {
		return nil, nil, nil, err
	}

	sig, err := privKey.Sign(msgHash[:])
	if err != nil {
		return nil, nil, nil, err
	}

	return &msgHash, sig, privKey.PubKey(), nil
}

//testsigcacheaddexists测试添加的能力，稍后检查
//签名缓存中存在签名三元组。
func TestSigCacheAddExists(t *testing.T) {
	sigCache := NewSigCache(200)

//生成一个随机的sigcache条目三元组。
	msg1, sig1, key1, err := genRandomSig()
	if err != nil {
		t.Errorf("unable to generate random signature test data")
	}

//将三元组添加到签名缓存。
	sigCache.Add(*msg1, sig1, key1)

//先前添加的三元组现在应该在sigcache中找到。
	sig1Copy, _ := btcec.ParseSignature(sig1.Serialize(), btcec.S256())
	key1Copy, _ := btcec.ParsePubKey(key1.SerializeCompressed(), btcec.S256())
	if !sigCache.Exists(*msg1, sig1Copy, key1Copy) {
		t.Errorf("previously added item not found in signature cache")
	}
}

//testsigcacheaddevicentry测试新签名的逐出情况
//将三元组添加到完整的签名缓存中，该缓存将触发随机
//逐出，然后将新元素添加到缓存中。
func TestSigCacheAddEvictEntry(t *testing.T) {
//创建最多可容纳100个条目的sigcache。
	sigCacheSize := uint(100)
	sigCache := NewSigCache(sigCacheSize)

//用一些随机的sig三元组填充sigcache。
	for i := uint(0); i < sigCacheSize; i++ {
		msg, sig, key, err := genRandomSig()
		if err != nil {
			t.Fatalf("unable to generate random signature test data")
		}

		sigCache.Add(*msg, sig, key)

		sigCopy, _ := btcec.ParseSignature(sig.Serialize(), btcec.S256())
		keyCopy, _ := btcec.ParsePubKey(key.SerializeCompressed(), btcec.S256())
		if !sigCache.Exists(*msg, sigCopy, keyCopy) {
			t.Errorf("previously added item not found in signature" +
				"cache")
		}
	}

//sigcache现在应该包含sigcachesize条目。
	if uint(len(sigCache.validSigs)) != sigCacheSize {
		t.Fatalf("sigcache should now have %v entries, instead it has %v",
			sigCacheSize, len(sigCache.validSigs))
	}

//添加一个新条目，这将导致随机选择的
//以前的条目。
	msgNew, sigNew, keyNew, err := genRandomSig()
	if err != nil {
		t.Fatalf("unable to generate random signature test data")
	}
	sigCache.Add(*msgNew, sigNew, keyNew)

//sigcache应该仍然有sigcache条目。
	if uint(len(sigCache.validSigs)) != sigCacheSize {
		t.Fatalf("sigcache should now have %v entries, instead it has %v",
			sigCacheSize, len(sigCache.validSigs))
	}

//上面添加的条目应该在sigcache中找到。
	sigNewCopy, _ := btcec.ParseSignature(sigNew.Serialize(), btcec.S256())
	keyNewCopy, _ := btcec.ParsePubKey(keyNew.SerializeCompressed(), btcec.S256())
	if !sigCache.Exists(*msgNew, sigNewCopy, keyNewCopy) {
		t.Fatalf("previously added item not found in signature cache")
	}
}

//TestSigCacheAddMaxEntriesZeroOrNegative测试如果创建了SigCache
//如果最大大小小于等于0，则不会向sigcache添加任何条目。
func TestSigCacheAddMaxEntriesZeroOrNegative(t *testing.T) {
//创建最多可容纳0个条目的sigcache。
	sigCache := NewSigCache(0)

//生成一个随机的sigcache条目三元组。
	msg1, sig1, key1, err := genRandomSig()
	if err != nil {
		t.Errorf("unable to generate random signature test data")
	}

//将三元组添加到签名缓存。
	sigCache.Add(*msg1, sig1, key1)

//不应找到生成的三联体。
	sig1Copy, _ := btcec.ParseSignature(sig1.Serialize(), btcec.S256())
	key1Copy, _ := btcec.ParsePubKey(key1.SerializeCompressed(), btcec.S256())
	if sigCache.Exists(*msg1, sig1Copy, key1Copy) {
		t.Errorf("previously added signature found in sigcache, but" +
			"shouldn't have been")
	}

//sigcache中不应该有任何条目。
	if len(sigCache.validSigs) != 0 {
		t.Errorf("%v items found in sigcache, no items should have"+
			"been added", len(sigCache.validSigs))
	}
}
