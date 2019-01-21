
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2014-2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package rpcclient

import (
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
)

//FutureGenerateResult是未来交付
//GenerateAsync RPC调用（或适用的错误）。
type FutureGenerateResult chan *response

//Receive等待将来承诺的响应，并返回
//由调用生成的块哈希。
func (r FutureGenerateResult) Receive() ([]*chainhash.Hash, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为字符串列表。
	var result []string
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}

//将每个块哈希转换为chainhash.hash并存储指向
//每一个。
	convertedResult := make([]*chainhash.Hash, len(result))
	for i, hashString := range result {
		convertedResult[i], err = chainhash.NewHashFromStr(hashString)
		if err != nil {
			return nil, err
		}
	}

	return convertedResult, nil
}

//generateasync返回可用于获取
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关块版本和更多详细信息，请参见生成。
func (c *Client) GenerateAsync(numBlocks uint32) FutureGenerateResult {
	cmd := btcjson.NewGenerateCmd(numBlocks)
	return c.sendCmd(cmd)
}

//生成生成numBlocks块并返回其散列值。
func (c *Client) Generate(numBlocks uint32) ([]*chainhash.Hash, error) {
	return c.GenerateAsync(numBlocks).Receive()
}

//FutureGetGenerateResult是未来交付
//GetGenerateAsync RPC调用（或适用的错误）。
type FutureGetGenerateResult chan *response

//receive等待将来承诺的响应，如果
//服务器设置为“我的”，否则为“假”。
func (r FutureGetGenerateResult) Receive() (bool, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return false, err
	}

//将结果取消标记为布尔值。
	var result bool
	err = json.Unmarshal(res, &result)
	if err != nil {
		return false, err
	}

	return result, nil
}

//GetGenerateAsync返回可用于获取
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅getgenerate。
func (c *Client) GetGenerateAsync() FutureGetGenerateResult {
	cmd := btcjson.NewGetGenerateCmd()
	return c.sendCmd(cmd)
}

//GetGenerate returns true if the server is set to mine, otherwise false.
func (c *Client) GetGenerate() (bool, error) {
	return c.GetGenerateAsync().Receive()
}

//未来生成结果是未来交付结果的承诺
//setGenerateAsync RPC调用（或适用的错误）。
type FutureSetGenerateResult chan *response

//接收等待未来承诺的响应并返回错误
//设置服务器是否生成硬币时发生。
func (r FutureSetGenerateResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

//setGenerateAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关块版本和更多详细信息，请参阅setgenerate。
func (c *Client) SetGenerateAsync(enable bool, numCPUs int) FutureSetGenerateResult {
	cmd := btcjson.NewSetGenerateCmd(enable, &numCPUs)
	return c.sendCmd(cmd)
}

//setgenerate设置服务器是否生成硬币（mine）。
func (c *Client) SetGenerate(enable bool, numCPUs int) error {
	return c.SetGenerateAsync(enable, numCPUs).Receive()
}

//Futuregethashespersecresult是未来交付结果的承诺
//gethashespersecasync RPC调用（或适用的错误）。
type FutureGetHashesPerSecResult chan *response

//receive等待将来承诺的响应并返回最近的
//在生成硬币（采矿）时，每秒进行哈希性能测量。
//如果服务器未挖掘，则返回零。
func (r FutureGetHashesPerSecResult) Receive() (int64, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return -1, err
	}

//将结果取消标记为Int64。
	var result int64
	err = json.Unmarshal(res, &result)
	if err != nil {
		return 0, err
	}

	return result, nil
}

//gethashespersecasync返回可用于获取
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅gethashespersec。
func (c *Client) GetHashesPerSecAsync() FutureGetHashesPerSecResult {
	cmd := btcjson.NewGetHashesPerSecCmd()
	return c.sendCmd(cmd)
}

//GetHashesPerSec returns a recent hashes per second performance measurement
//同时产生硬币（采矿）。如果服务器不是，则返回零
//采矿。
func (c *Client) GetHashesPerSec() (int64, error) {
	return c.GetHashesPerSecAsync().Receive()
}

//FutureGetMiningResult是未来交付
//GetMiningForAsync RPC调用（或适用的错误）。
type FutureGetMiningInfoResult chan *response

//receive等待将来承诺的响应并返回挖掘
//信息。
func (r FutureGetMiningInfoResult) Receive() (*btcjson.GetMiningInfoResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为GetMiningInfo结果对象。
	var infoResult btcjson.GetMiningInfoResult
	err = json.Unmarshal(res, &infoResult)
	if err != nil {
		return nil, err
	}

	return &infoResult, nil
}

//GetMiningInfoAsync返回一个可用于获取
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅getmininginfo。
func (c *Client) GetMiningInfoAsync() FutureGetMiningInfoResult {
	cmd := btcjson.NewGetMiningInfoCmd()
	return c.sendCmd(cmd)
}

//GetMiningInfo返回挖掘信息。
func (c *Client) GetMiningInfo() (*btcjson.GetMiningInfoResult, error) {
	return c.GetMiningInfoAsync().Receive()
}

//FutureGetnetworkHashps是未来交付
//GetNetworkHashpsAsync RPC调用（或适用的错误）。
type FutureGetNetworkHashPS chan *response

//receive等待将来承诺的响应并返回
//估计每秒网络哈希数，用于
//参数。
func (r FutureGetNetworkHashPS) Receive() (int64, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return -1, err
	}

//将结果取消标记为Int64。
	var result int64
	err = json.Unmarshal(res, &result)
	if err != nil {
		return 0, err
	}

	return result, nil
}

//GetNetworkHashpsAsync返回可用于获取的类型的实例
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅GetNetworkHashps。
func (c *Client) GetNetworkHashPSAsync() FutureGetNetworkHashPS {
	cmd := btcjson.NewGetNetworkHashPSCmd(nil, nil)
	return c.sendCmd(cmd)
}

//GetNetworkHashps使用返回估计的每秒网络哈希数
//默认块数和最近的块高度。
//
//请参阅GetNetworkHashPS2以重写要使用的块的数目，以及
//getNetworkHashPS3覆盖计算估计的高度。
func (c *Client) GetNetworkHashPS() (int64, error) {
	return c.GetNetworkHashPSAsync().Receive()
}

//GetNetworkHashPS2Async返回可用于获取的类型的实例
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅GetNetworkHashPS2。
func (c *Client) GetNetworkHashPS2Async(blocks int) FutureGetNetworkHashPS {
	cmd := btcjson.NewGetNetworkHashPSCmd(&blocks, nil)
	return c.sendCmd(cmd)
}

//GetNetworkHashPS2返回
//指定从最近的块向后工作的前个块数
//块高度。blocks参数也可以是-1，在这种情况下，
//将使用自上次难度更改以来的块。
//
//请参见getnetworkhashps使用默认值，以及getnetworkhashps3覆盖
//计算估计值的高度。
func (c *Client) GetNetworkHashPS2(blocks int) (int64, error) {
	return c.GetNetworkHashPS2Async(blocks).Receive()
}

//GetNetworkHashPS3Async返回可用于获取
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅GetNetworkHashPS3。
func (c *Client) GetNetworkHashPS3Async(blocks, height int) FutureGetNetworkHashPS {
	cmd := btcjson.NewGetNetworkHashPSCmd(&blocks, &height)
	return c.sendCmd(cmd)
}

//GetNetworkHashPS3返回
//从指定的
//块高度。blocks参数也可以是-1，在这种情况下，
//将使用自上次难度更改以来的块。
//
//请参见GetNetworkHashPS和GetNetworkHashPS2以使用默认值。
func (c *Client) GetNetworkHashPS3(blocks, height int) (int64, error) {
	return c.GetNetworkHashPS3Async(blocks, height).Receive()
}

//未来网络是未来交付
//GetWorkAsync RPC调用（或适用的错误）。
type FutureGetWork chan *response

//receive等待将来承诺的响应并返回哈希
//数据工作。
func (r FutureGetWork) Receive() (*btcjson.GetWorkResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为GetWork结果对象。
	var result btcjson.GetWorkResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

//GetWorkAsync返回可用于获取结果的类型的实例
//在将来的某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅getwork。
func (c *Client) GetWorkAsync() FutureGetWork {
	cmd := btcjson.NewGetWorkCmd(nil)
	return c.sendCmd(cmd)
}

//GetWork返回要处理的哈希数据。
//
//请参阅GetWorksubmit提交找到的解决方案。
func (c *Client) GetWork() (*btcjson.GetWorkResult, error) {
	return c.GetWorkAsync().Receive()
}

//FutureGetNetworkSubmit是未来交付
//GetWorksubmitAsync RPC调用（或适用的错误）。
type FutureGetWorkSubmit chan *response

//receive等待将来承诺的响应并返回
//是否接受提交的块头。
func (r FutureGetWorkSubmit) Receive() (bool, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return false, err
	}

//将结果取消标记为布尔值。
	var accepted bool
	err = json.Unmarshal(res, &accepted)
	if err != nil {
		return false, err
	}

	return accepted, nil
}

//GetWorksubmitAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅GetWorksubmit。
func (c *Client) GetWorkSubmitAsync(data string) FutureGetWorkSubmit {
	cmd := btcjson.NewGetWorkCmd(&data)
	return c.sendCmd(cmd)
}

//GetWorksubmit提交一个块头，它是以前的解决方案
//请求的数据并返回是否接受该解决方案。
//
//请参见GetWork以请求要处理的数据。
func (c *Client) GetWorkSubmit(data string) (bool, error) {
	return c.GetWorkSubmitAsync(data).Receive()
}

//FutureSubmitBlockResult是未来交付
//SubmitBlockAsync RPC调用（或适用的错误）。
type FutureSubmitBlockResult chan *response

//接收等待未来承诺的响应并返回错误
//提交块时发生。
func (r FutureSubmitBlockResult) Receive() error {
	res, err := receiveFuture(r)
	if err != nil {
		return err
	}

	if string(res) != "null" {
		var result string
		err = json.Unmarshal(res, &result)
		if err != nil {
			return err
		}

		return errors.New(result)
	}

	return nil

}

//SubmitBlockAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅SubmitBlock。
func (c *Client) SubmitBlockAsync(block *btcutil.Block, options *btcjson.SubmitBlockOptions) FutureSubmitBlockResult {
	blockHex := ""
	if block != nil {
		blockBytes, err := block.Bytes()
		if err != nil {
			return newFutureError(err)
		}

		blockHex = hex.EncodeToString(blockBytes)
	}

	cmd := btcjson.NewSubmitBlockCmd(blockHex, options)
	return c.sendCmd(cmd)
}

//SubmitBlock试图向比特币网络提交一个新块。
func (c *Client) SubmitBlock(block *btcutil.Block, options *btcjson.SubmitBlockOptions) error {
	return c.SubmitBlockAsync(block, options).Receive()
}

//TODO（Davec）：实现GetBlockTemplate
