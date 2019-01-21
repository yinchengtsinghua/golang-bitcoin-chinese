
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2014-2017 BTCSuite开发者
//版权所有（c）2015-2017法令开发商
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package rpcclient

import (
	"bytes"
	"encoding/hex"
	"encoding/json"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

//FutureGetBestBlockHashResult是未来交付
//GetBestBlockAsync RPC调用（或适用的错误）。
type FutureGetBestBlockHashResult chan *response

//receive等待将来承诺的响应，并返回
//最长区块链中最好的区块。
func (r FutureGetBestBlockHashResult) Receive() (*chainhash.Hash, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//Unmarshal result as a string.
	var txHashStr string
	err = json.Unmarshal(res, &txHashStr)
	if err != nil {
		return nil, err
	}
	return chainhash.NewHashFromStr(txHashStr)
}

//GetBestBlockHasAsync返回可用于获取
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅GetBestBlockHash。
func (c *Client) GetBestBlockHashAsync() FutureGetBestBlockHashResult {
	cmd := btcjson.NewGetBestBlockHashCmd()
	return c.sendCmd(cmd)
}

//GetBestBlockHash返回最长块中最佳块的哈希
//链。
func (c *Client) GetBestBlockHash() (*chainhash.Hash, error) {
	return c.GetBestBlockHashAsync().Receive()
}

//FutureGetBlockResult是未来交付
//GetBlockAsync RPC调用（或适用的错误）。
type FutureGetBlockResult chan *response

//接收等待未来承诺的响应并返回原始
//从给定哈希的服务器请求的块。
func (r FutureGetBlockResult) Receive() (*wire.MsgBlock, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为字符串。
	var blockHex string
	err = json.Unmarshal(res, &blockHex)
	if err != nil {
		return nil, err
	}

//将序列化块十六进制解码为原始字节。
	serializedBlock, err := hex.DecodeString(blockHex)
	if err != nil {
		return nil, err
	}

//反序列化块并返回它。
	var msgBlock wire.MsgBlock
	err = msgBlock.Deserialize(bytes.NewReader(serializedBlock))
	if err != nil {
		return nil, err
	}
	return &msgBlock, nil
}

//GetBlockAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅getblock。
func (c *Client) GetBlockAsync(blockHash *chainhash.Hash) FutureGetBlockResult {
	hash := ""
	if blockHash != nil {
		hash = blockHash.String()
	}

	cmd := btcjson.NewGetBlockCmd(hash, btcjson.Bool(false), nil)
	return c.sendCmd(cmd)
}

//GetBlock返回给定哈希的服务器的原始块。
//
//请参阅getblockverbose以检索包含有关
//代替块。
func (c *Client) GetBlock(blockHash *chainhash.Hash) (*wire.MsgBlock, error) {
	return c.GetBlockAsync(blockHash).Receive()
}

//FutureGetBlockVerboseResult是未来交付
//GetBlockVerboseAsync RPC调用（或适用的错误）。
type FutureGetBlockVerboseResult chan *response

//receive等待将来承诺的响应并返回数据
//从服务器上构造关于请求块的信息。
func (r FutureGetBlockVerboseResult) Receive() (*btcjson.GetBlockVerboseResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将原始结果取消标记为blockresult。
	var blockResult btcjson.GetBlockVerboseResult
	err = json.Unmarshal(res, &blockResult)
	if err != nil {
		return nil, err
	}
	return &blockResult, nil
}

//GetBlockVerboseAsync返回可用于获取
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅getblockverbose。
func (c *Client) GetBlockVerboseAsync(blockHash *chainhash.Hash) FutureGetBlockVerboseResult {
	hash := ""
	if blockHash != nil {
		hash = blockHash.String()
	}

	cmd := btcjson.NewGetBlockCmd(hash, btcjson.Bool(true), nil)
	return c.sendCmd(cmd)
}

//GetBlockVerbose从服务器返回包含信息的数据结构
//关于给定哈希的块。
//
//请参阅getblockverbosetx以检索事务数据结构。
//请参阅getblock以检索原始块。
func (c *Client) GetBlockVerbose(blockHash *chainhash.Hash) (*btcjson.GetBlockVerboseResult, error) {
	return c.GetBlockVerboseAsync(blockHash).Receive()
}

//GetBlockVerbosetAsync返回可用于获取
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//请参阅getblockverbosetx或阻塞版本和更多详细信息。
func (c *Client) GetBlockVerboseTxAsync(blockHash *chainhash.Hash) FutureGetBlockVerboseResult {
	hash := ""
	if blockHash != nil {
		hash = blockHash.String()
	}

	cmd := btcjson.NewGetBlockCmd(hash, btcjson.Bool(true), btcjson.Bool(true))
	return c.sendCmd(cmd)
}

//GetBlockVerbosetx从服务器返回包含信息的数据结构
//关于一个块及其给定哈希的事务。
//
//如果只首选事务哈希，请参阅getblockverbose。
//请参阅getblock以检索原始块。
func (c *Client) GetBlockVerboseTx(blockHash *chainhash.Hash) (*btcjson.GetBlockVerboseResult, error) {
	return c.GetBlockVerboseTxAsync(blockHash).Receive()
}

//FutureGetBlockCountResult是未来交付
//GetBlockCountAsync RPC调用（或适用的错误）。
type FutureGetBlockCountResult chan *response

//receive等待将来承诺的响应并返回数字
//最长区块链中的区块。
func (r FutureGetBlockCountResult) Receive() (int64, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return 0, err
	}

//将结果取消标记为Int64。
	var count int64
	err = json.Unmarshal(res, &count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

//GetBlockCountAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅GetBlockCount。
func (c *Client) GetBlockCountAsync() FutureGetBlockCountResult {
	cmd := btcjson.NewGetBlockCountCmd()
	return c.sendCmd(cmd)
}

//GetBlockCount返回最长块链中的块数。
func (c *Client) GetBlockCount() (int64, error) {
	return c.GetBlockCountAsync().Receive()
}

//FutureGetTDfficultyResult是未来交付
//GetDifficultyAsync RPC调用（或适用的错误）。
type FutureGetDifficultyResult chan *response

//Receive waits for the response promised by the future and returns the
//证明工作难度是最低难度的倍数。
func (r FutureGetDifficultyResult) Receive() (float64, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return 0, err
	}

//将结果取消标记为float64。
	var difficulty float64
	err = json.Unmarshal(res, &difficulty)
	if err != nil {
		return 0, err
	}
	return difficulty, nil
}

//GetDifficultyAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅GetDifficulty。
func (c *Client) GetDifficultyAsync() FutureGetDifficultyResult {
	cmd := btcjson.NewGetDifficultyCmd()
	return c.sendCmd(cmd)
}

//GET难度将工作难度的证明作为倍数。
//最小难度。
func (c *Client) GetDifficulty() (float64, error) {
	return c.GetDifficultyAsync().Receive()
}

//FutureGetBlockChainInformationSult是一个承诺，它将提供
//GetBlockChainInfoAsync RPC调用（或适用的错误）。
type FutureGetBlockChainInfoResult chan *response

//接收等待未来承诺的响应并返回链信息
//服务器提供的结果。
func (r FutureGetBlockChainInfoResult) Receive() (*btcjson.GetBlockChainInfoResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

	var chainInfo btcjson.GetBlockChainInfoResult
	if err := json.Unmarshal(res, &chainInfo); err != nil {
		return nil, err
	}
	return &chainInfo, nil
}

//GetBlockChainInfoAsync返回可用于获取
//通过调用receive函数在将来某个时间的rpc结果
//在返回的实例上。
//
//See GetBlockChainInfo for the blocking version and more details.
func (c *Client) GetBlockChainInfoAsync() FutureGetBlockChainInfoResult {
	cmd := btcjson.NewGetBlockChainInfoCmd()
	return c.sendCmd(cmd)
}

//GetBlockChainInfo返回与处理状态有关的信息
//各种特定于链的细节，例如从尖端开始的当前困难
//主链。
func (c *Client) GetBlockChainInfo() (*btcjson.GetBlockChainInfoResult, error) {
	return c.GetBlockChainInfoAsync().Receive()
}

//FutureGetBlockHashResult是未来交付
//GetBlockHashAsync RPC调用（或适用的错误）。
type FutureGetBlockHashResult chan *response

//receive等待将来承诺的响应，并返回
//在给定高度的最佳区块链中的区块。
func (r FutureGetBlockHashResult) Receive() (*chainhash.Hash, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为字符串编码的sha。
	var txHashStr string
	err = json.Unmarshal(res, &txHashStr)
	if err != nil {
		return nil, err
	}
	return chainhash.NewHashFromStr(txHashStr)
}

//GetBlockHasAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅GetBlockHash。
func (c *Client) GetBlockHashAsync(blockHeight int64) FutureGetBlockHashResult {
	cmd := btcjson.NewGetBlockHashCmd(blockHeight)
	return c.sendCmd(cmd)
}

//GetBlockHash返回位于
//给定高度。
func (c *Client) GetBlockHash(blockHeight int64) (*chainhash.Hash, error) {
	return c.GetBlockHashAsync(blockHeight).Receive()
}

//FutureGetBlockHeaderResult是未来交付
//GetBlockHeaderAsync RPC调用（或适用的错误）。
type FutureGetBlockHeaderResult chan *response

//Receive waits for the response promised by the future and returns the
//从给定哈希的服务器请求BlockHeader。
func (r FutureGetBlockHeaderResult) Receive() (*wire.BlockHeader, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为字符串。
	var bhHex string
	err = json.Unmarshal(res, &bhHex)
	if err != nil {
		return nil, err
	}

	serializedBH, err := hex.DecodeString(bhHex)
	if err != nil {
		return nil, err
	}

//反序列化Bulk报头并返回它。
	var bh wire.BlockHeader
	err = bh.Deserialize(bytes.NewReader(serializedBH))
	if err != nil {
		return nil, err
	}

	return &bh, err
}

//GetBlockHeaderAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅GetBlockHeader。
func (c *Client) GetBlockHeaderAsync(blockHash *chainhash.Hash) FutureGetBlockHeaderResult {
	hash := ""
	if blockHash != nil {
		hash = blockHash.String()
	}

	cmd := btcjson.NewGetBlockHeaderCmd(hash, btcjson.Bool(false))
	return c.sendCmd(cmd)
}

//GetBlockHeader返回给定哈希的服务器的BlockHeader。
//
//请参阅getblockheaderbose以检索包含有关
//代替块。
func (c *Client) GetBlockHeader(blockHash *chainhash.Hash) (*wire.BlockHeader, error) {
	return c.GetBlockHeaderAsync(blockHash).Receive()
}

//FutureGetBlockHeaderVerboseResult is a future promise to deliver the result of a
//GetBlockAsync RPC调用（或适用的错误）。
type FutureGetBlockHeaderVerboseResult chan *response

//receive等待将来承诺的响应并返回
//data structure of the blockheader requested from the server given its hash.
func (r FutureGetBlockHeaderVerboseResult) Receive() (*btcjson.GetBlockHeaderVerboseResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为字符串。
	var bh btcjson.GetBlockHeaderVerboseResult
	err = json.Unmarshal(res, &bh)
	if err != nil {
		return nil, err
	}

	return &bh, nil
}

//GetBlockHeaderboseAsync返回一个可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅GetBlockHeader。
func (c *Client) GetBlockHeaderVerboseAsync(blockHash *chainhash.Hash) FutureGetBlockHeaderVerboseResult {
	hash := ""
	if blockHash != nil {
		hash = blockHash.String()
	}

	cmd := btcjson.NewGetBlockHeaderCmd(hash, btcjson.Bool(true))
	return c.sendCmd(cmd)
}

//GetBlockHeaderVerbose returns a data structure with information about the
//来自给定哈希的服务器的blockheader。
//
//See GetBlockHeader to retrieve a blockheader instead.
func (c *Client) GetBlockHeaderVerbose(blockHash *chainhash.Hash) (*btcjson.GetBlockHeaderVerboseResult, error) {
	return c.GetBlockHeaderVerboseAsync(blockHash).Receive()
}

//FuturegeTM工具结果是未来交付
//GetMemPoolentryAsync RPC调用（或适用的错误）。
type FutureGetMempoolEntryResult chan *response

//receive等待将来承诺的响应并返回数据
//结构，其中包含有关给定内存池中事务的信息
//它的散列。
func (r FutureGetMempoolEntryResult) Receive() (*btcjson.GetMempoolEntryResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为字符串数组。
	var mempoolEntryResult btcjson.GetMempoolEntryResult
	err = json.Unmarshal(res, &mempoolEntryResult)
	if err != nil {
		return nil, err
	}

	return &mempoolEntryResult, nil
}

//GetMemPoolentryAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅getmempoolentry。
func (c *Client) GetMempoolEntryAsync(txHash string) FutureGetMempoolEntryResult {
	cmd := btcjson.NewGetMempoolEntryCmd(txHash)
	return c.sendCmd(cmd)
}

//GetMempoolEntry returns a data structure with information about the
//给定哈希的内存池中的事务。
func (c *Client) GetMempoolEntry(txHash string) (*btcjson.GetMempoolEntryResult, error) {
	return c.GetMempoolEntryAsync(txHash).Receive()
}

//FutureGeWrMeMoPOLREST是一个未来的承诺交付的结果
//GetRawEmpoolAsync RPC调用（或适用的错误）。
type FutureGetRawMempoolResult chan *response

//Receive waits for the response promised by the future and returns the hashes
//内存池中的所有事务。
func (r FutureGetRawMempoolResult) Receive() ([]*chainhash.Hash, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为字符串数组。
	var txHashStrs []string
	err = json.Unmarshal(res, &txHashStrs)
	if err != nil {
		return nil, err
	}

//Create a slice of ShaHash arrays from the string slice.
	txHashes := make([]*chainhash.Hash, 0, len(txHashStrs))
	for _, hashStr := range txHashStrs {
		txHash, err := chainhash.NewHashFromStr(hashStr)
		if err != nil {
			return nil, err
		}
		txHashes = append(txHashes, txHash)
	}

	return txHashes, nil
}

//GetRawEmpoolAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅getrawmupool。
func (c *Client) GetRawMempoolAsync() FutureGetRawMempoolResult {
	cmd := btcjson.NewGetRawMempoolCmd(btcjson.Bool(false))
	return c.sendCmd(cmd)
}

//GetRawmEmpool返回内存池中所有事务的哈希值。
//
//请参阅getrawmumpoolverbose以检索包含以下信息的数据结构
//而不是交易。
func (c *Client) GetRawMempool() ([]*chainhash.Hash, error) {
	return c.GetRawMempoolAsync().Receive()
}

//FutureGetRawMempoolVerboseResult is a future promise to deliver the result of
//GetRawmPoolVerboseAsync RPC调用（或适用的错误）。
type FutureGetRawMempoolVerboseResult chan *response

//Receive waits for the response promised by the future and returns a map of
//事务散列到关联的数据结构，其中包含有关
//内存池中所有事务的事务。
func (r FutureGetRawMempoolVerboseResult) Receive() (map[string]btcjson.GetRawMempoolVerboseResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为字符串（tx-shas）到其详细信息的映射
//结果。
	var mempoolItems map[string]btcjson.GetRawMempoolVerboseResult
	err = json.Unmarshal(res, &mempoolItems)
	if err != nil {
		return nil, err
	}
	return mempoolItems, nil
}

//GetRawEmpoolVerboseAsync返回一个类型的实例，该类型可用于
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅getrawmumpoolverbose。
func (c *Client) GetRawMempoolVerboseAsync() FutureGetRawMempoolVerboseResult {
	cmd := btcjson.NewGetRawMempoolCmd(btcjson.Bool(true))
	return c.sendCmd(cmd)
}

//GetRawmEmpoolVerbose将事务哈希的映射返回到关联的
//包含有关中所有事务的事务信息的数据结构
//内存池。
//
//请参阅getrawmupool以仅检索事务哈希。
func (c *Client) GetRawMempoolVerbose() (map[string]btcjson.GetRawMempoolVerboseResult, error) {
	return c.GetRawMempoolVerboseAsync().Receive()
}

//未来预测结果是未来实现
//EstimateFeeAsync RPC调用（或适用的错误）。
type FutureEstimateFeeResult chan *response

//receive等待将来承诺的响应并返回信息
//由服务器提供。
func (r FutureEstimateFeeResult) Receive() (float64, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return -1, err
	}

//将结果取消标记为GetInfo结果对象。
	var fee float64
	err = json.Unmarshal(res, &fee)
	if err != nil {
		return -1, err
	}

	return fee, nil
}

//EstimateFeeAsync返回可用于获取结果的类型的实例
//在将来的某个时间通过调用
//返回实例。
//
//有关阻塞版本和更多详细信息，请参阅EstimateFee。
func (c *Client) EstimateFeeAsync(numBlocks int64) FutureEstimateFeeResult {
	cmd := btcjson.NewEstimateFeeCmd(numBlocks)
	return c.sendCmd(cmd)
}

//EstimateFee提供每千字节比特币的估计费用。
func (c *Client) EstimateFee(numBlocks int64) (float64, error) {
	return c.EstimateFeeAsync(numBlocks).Receive()
}

//未来每一天的结果是一个未来的承诺，交付的结果是
//VerifyChainAsync、VerifyChainLevelAsyncRpc或VerifyChainBlocksAsync
//调用（或适用的错误）。
type FutureVerifyChainResult chan *response

//receive等待将来承诺的响应并返回
//是否根据检查级别和块数验证链
//验证在原始调用中指定的。
func (r FutureVerifyChainResult) Receive() (bool, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return false, err
	}

//将结果取消标记为布尔值。
	var verified bool
	err = json.Unmarshal(res, &verified)
	if err != nil {
		return false, err
	}
	return verified, nil
}

//VerifyChainAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅VerifyChain。
func (c *Client) VerifyChainAsync() FutureVerifyChainResult {
	cmd := btcjson.NewVerifyChainCmd(nil, nil)
	return c.sendCmd(cmd)
}

//VerifyChain请求服务器使用
//要验证的默认检查级别和块数。
//
//请参见verifychainlevel和verifychainblocks以覆盖默认值。
func (c *Client) VerifyChain() (bool, error) {
	return c.VerifyChainAsync().Receive()
}

//VerifyChainLevelAsync返回可用于获取
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关块版本和更多详细信息，请参阅verifychainlevel。
func (c *Client) VerifyChainLevelAsync(checkLevel int32) FutureVerifyChainResult {
	cmd := btcjson.NewVerifyChainCmd(&checkLevel, nil)
	return c.sendCmd(cmd)
}

//verifychainlevel请求服务器使用
//通过的检查级别和要验证的默认块数。
//
//检查级别用更高的数字控制验证的彻底性。
//增加支票数量，因此
//验证需要。
//
//请参见verifychain使用默认检查级别，并将verifychainblocks用于
//覆盖要验证的块数。
func (c *Client) VerifyChainLevel(checkLevel int32) (bool, error) {
	return c.VerifyChainLevelAsync(checkLevel).Receive()
}

//verifychainblocksasync返回可用于获取
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关块版本和更多详细信息，请参阅VerifyChainBlocks。
func (c *Client) VerifyChainBlocksAsync(checkLevel, numBlocks int32) FutureVerifyChainResult {
	cmd := btcjson.NewVerifyChainCmd(&checkLevel, &numBlocks)
	return c.sendCmd(cmd)
}

//VerifyChainBlocks请求服务器验证区块链数据库
//使用通过的检查级别和要验证的块数。
//
//检查级别用更高的数字控制验证的彻底性。
//增加支票数量，因此
//验证需要。
//
//块数是指从
//当前最长的链。
//
//请参见VerifyChain和VerifyChainLevel以使用默认值。
func (c *Client) VerifyChainBlocks(checkLevel, numBlocks int32) (bool, error) {
	return c.VerifyChainBlocksAsync(checkLevel, numBlocks).Receive()
}

//FutureGetXOutResult是未来交付
//GettXoutAsync RPC调用（或适用的错误）。
type FutureGetTxOutResult chan *response

//receive等待将来承诺的响应并返回
//给定哈希的事务。
func (r FutureGetTxOutResult) Receive() (*btcjson.GetTxOutResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//注意输出已经消耗掉的特殊情况。
//它应该返回字符串“null”
	if string(res) == "null" {
		return nil, nil
	}

//将结果取消标记为gettxout结果对象。
	var txOutInfo *btcjson.GetTxOutResult
	err = json.Unmarshal(res, &txOutInfo)
	if err != nil {
		return nil, err
	}

	return txOutInfo, nil
}

//GettXoutAsync返回可用于获取
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关块版本和更多详细信息，请参见gettxout。
func (c *Client) GetTxOutAsync(txHash *chainhash.Hash, index uint32, mempool bool) FutureGetTxOutResult {
	hash := ""
	if txHash != nil {
		hash = txHash.String()
	}

	cmd := btcjson.NewGetTxOutCmd(hash, index, &mempool)
	return c.sendCmd(cmd)
}

//GettXOut返回事务输出信息（如果事务输出信息未释放并且
//反之亦然。
func (c *Client) GetTxOut(txHash *chainhash.Hash, index uint32, mempool bool) (*btcjson.GetTxOutResult, error) {
	return c.GetTxOutAsync(txHash, index, mempool).Receive()
}

//FutureRescanBlocksResult是未来交付
//重新扫描块同步RPC调用（或适用的错误）。
//
//注意：这是从中导入的BTCSuite扩展
//github.com/decred/dcrrpcclient.
type FutureRescanBlocksResult chan *response

//receive等待将来承诺的响应并返回
//发现重新扫描块数据。
//
//注意：这是从中导入的BTCSuite扩展
//github.com/decred/dcrrpcclient.
func (r FutureRescanBlocksResult) Receive() ([]btcjson.RescannedBlock, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

	var rescanBlocksResult []btcjson.RescannedBlock
	err = json.Unmarshal(res, &rescanBlocksResult)
	if err != nil {
		return nil, err
	}

	return rescanBlocksResult, nil
}

//rescanblocksasync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关块版本和更多详细信息，请参阅重新扫描块。
//
//注意：这是从中导入的BTCSuite扩展
//github.com/decred/dcrrpcclient.
func (c *Client) RescanBlocksAsync(blockHashes []chainhash.Hash) FutureRescanBlocksResult {
	strBlockHashes := make([]string, len(blockHashes))
	for i := range blockHashes {
		strBlockHashes[i] = blockHashes[i].String()
	}

	cmd := btcjson.NewRescanBlocksCmd(strBlockHashes)
	return c.sendCmd(cmd)
}

//rescanblocks按顺序使用blockhash重新扫描由blockhash标识的块。
//客户端加载的事务筛选器。块不需要在
//主链，但它们需要彼此相邻。
//
//注意：这是从中导入的BTCSuite扩展
//github.com/decred/dcrrpcclient.
func (c *Client) RescanBlocks(blockHashes []chainhash.Hash) ([]btcjson.RescannedBlock, error) {
	return c.RescanBlocksAsync(blockHashes).Receive()
}

//FutureInvalidateBlockResult是未来交付
//使BlockAsync RPC调用无效（或发生适用错误）。
type FutureInvalidateBlockResult chan *response

//接收等待未来承诺的响应并返回原始
//从给定哈希的服务器请求的块。
func (r FutureInvalidateBlockResult) Receive() error {
	_, err := receiveFuture(r)

	return err
}

//invalidBlockAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅invalidBlock。
func (c *Client) InvalidateBlockAsync(blockHash *chainhash.Hash) FutureInvalidateBlockResult {
	hash := ""
	if blockHash != nil {
		hash = blockHash.String()
	}

	cmd := btcjson.NewInvalidateBlockCmd(hash)
	return c.sendCmd(cmd)
}

//无效块使特定块无效。
func (c *Client) InvalidateBlock(blockHash *chainhash.Hash) error {
	return c.InvalidateBlockAsync(blockHash).Receive()
}

//FutureGetFilterResult是未来交付
//getfilterasync RPC调用（或适用的错误）。
type FutureGetCFilterResult chan *response

//接收等待未来承诺的响应并返回原始
//从给定其块哈希的服务器请求的筛选器。
func (r FutureGetCFilterResult) Receive() (*wire.MsgCFilter, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为字符串。
	var filterHex string
	err = json.Unmarshal(res, &filterHex)
	if err != nil {
		return nil, err
	}

//将序列化的CF十六进制解码为原始字节。
	serializedFilter, err := hex.DecodeString(filterHex)
	if err != nil {
		return nil, err
	}

//将筛选字节分配给有线消息的正确字段。
//我们不会设置块哈希或扩展标志，因为我们
//不要在rpc响应中实际得到它。
	var msgCFilter wire.MsgCFilter
	msgCFilter.Data = serializedFilter
	return &msgCFilter, nil
}

//getfilterasync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻塞版本和更多详细信息，请参阅getfilter。
func (c *Client) GetCFilterAsync(blockHash *chainhash.Hash,
	filterType wire.FilterType) FutureGetCFilterResult {
	hash := ""
	if blockHash != nil {
		hash = blockHash.String()
	}

	cmd := btcjson.NewGetCFilterCmd(hash, filterType)
	return c.sendCmd(cmd)
}

//getfilter返回给定块哈希的服务器的原始筛选器。
func (c *Client) GetCFilter(blockHash *chainhash.Hash,
	filterType wire.FilterType) (*wire.MsgCFilter, error) {
	return c.GetCFilterAsync(blockHash, filterType).Receive()
}

//FutureGetFilterHeaderResult是未来交付
//getfilterheaderasync RPC调用（或适用的错误）。
type FutureGetCFilterHeaderResult chan *response

//接收等待未来承诺的响应并返回原始
//从给定其块哈希的服务器请求的筛选器头。
func (r FutureGetCFilterHeaderResult) Receive() (*wire.MsgCFHeaders, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为字符串。
	var headerHex string
	err = json.Unmarshal(res, &headerHex)
	if err != nil {
		return nil, err
	}

//将解码的头分配到哈希中
	headerHash, err := chainhash.NewHashFromStr(headerHex)
	if err != nil {
		return nil, err
	}

//将哈希分配给头消息并返回它。
	msgCFHeaders := wire.MsgCFHeaders{PrevFilterHeader: *headerHash}
	return &msgCFHeaders, nil

}

//getfilterheaderasync返回可用于获取
//通过调用receive函数在将来某个时间的rpc结果
//在返回的实例上。
//
//有关阻止版本和更多详细信息，请参阅getcfilterheader。
func (c *Client) GetCFilterHeaderAsync(blockHash *chainhash.Hash,
	filterType wire.FilterType) FutureGetCFilterHeaderResult {
	hash := ""
	if blockHash != nil {
		hash = blockHash.String()
	}

	cmd := btcjson.NewGetCFilterHeaderCmd(hash, filterType)
	return c.sendCmd(cmd)
}

//getfilterheader返回给定块的服务器的原始筛选器头
//搞砸。
func (c *Client) GetCFilterHeader(blockHash *chainhash.Hash,
	filterType wire.FilterType) (*wire.MsgCFHeaders, error) {
	return c.GetCFilterHeaderAsync(blockHash, filterType).Receive()
}
