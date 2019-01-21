
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
	"bytes"
	"encoding/hex"
	"encoding/json"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

//sighashType枚举
//signrawtransaction函数接受。
type SigHashType string

//用于指示signrawtransaction的签名哈希类型的常量。
const (
//SIGHASHALL表示所有输出都应该签名。
	SigHashAll SigHashType = "ALL"

//SIGHASHONE表示不应对任何输出进行签名。这个
//可以认为指定签名者不关心
//比特币走了。
	SigHashNone SigHashType = "NONE"

//sighashsingle表示应该对单个输出进行签名。这个
//可以认为指定签名者只关心其中一个
//the outputs goes, but not any of the others.
	SigHashSingle SigHashType = "SINGLE"

//Sighashallanyonecanpay表示签名者不关心
//事务的其他输入来自，因此它允许其他人
//添加输入。此外，它还使用了sighashall签名方法
//输出。
	SigHashAllAnyoneCanPay SigHashType = "ALL|ANYONECANPAY"

//Sighashnoneanyonecanpay表示签名者不关心
//事务的其他输入来自，因此它允许其他人
//添加输入。此外，它还使用了sighashnone签名方法
//输出。
	SigHashNoneAnyoneCanPay SigHashType = "NONE|ANYONECANPAY"

//SyasHouthLangyOnEncEn薪表示签名者不关心哪里
//事务的其他输入来自，因此它允许
//要添加输入的人员。此外，它还使用了叹息单签名
//输出方法。
	SigHashSingleAnyoneCanPay SigHashType = "SINGLE|ANYONECANPAY"
)

//字符串以可读形式返回sighhashType。
func (s SigHashType) String() string {
	return string(s)
}

//FutureGetrawtransactionResult是未来交付
//GetRawTransactionAsync RPC调用（或适用的错误）。
type FutureGetRawTransactionResult chan *response

//receive等待将来承诺的响应并返回
//给定哈希的事务。
func (r FutureGetRawTransactionResult) Receive() (*btcutil.Tx, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为字符串。
	var txHex string
	err = json.Unmarshal(res, &txHex)
	if err != nil {
		return nil, err
	}

//将序列化事务十六进制解码为原始字节。
	serializedTx, err := hex.DecodeString(txHex)
	if err != nil {
		return nil, err
	}

//反序列化事务并返回它。
	var msgTx wire.MsgTx
	if err := msgTx.Deserialize(bytes.NewReader(serializedTx)); err != nil {
		return nil, err
	}
	return btcutil.NewTx(&msgTx), nil
}

//getrawtransactionasync返回可用于获取
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅getrawtransaction。
func (c *Client) GetRawTransactionAsync(txHash *chainhash.Hash) FutureGetRawTransactionResult {
	hash := ""
	if txHash != nil {
		hash = txHash.String()
	}

	cmd := btcjson.NewGetRawTransactionCmd(hash, btcjson.Int(0))
	return c.sendCmd(cmd)
}

//GetRawTransaction返回给定哈希的事务。
//
//请参阅getrawtransactionverbose以获取有关
//交易。
func (c *Client) GetRawTransaction(txHash *chainhash.Hash) (*btcutil.Tx, error) {
	return c.GetRawTransactionAsync(txHash).Receive()
}

//FutureGetTrawTransactionVerboseResult是未来交付
//GetRawTransactionVerboseAsync RPC调用的结果（或适用的
//错误）。
type FutureGetRawTransactionVerboseResult chan *response

//接收等待未来承诺的响应并返回信息
//关于给定哈希的事务。
func (r FutureGetRawTransactionVerboseResult) Receive() (*btcjson.TxRawResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果整理为GETRAWSWATECT结果对象。
	var rawTxResult btcjson.TxRawResult
	err = json.Unmarshal(res, &rawTxResult)
	if err != nil {
		return nil, err
	}

	return &rawTxResult, nil
}

//GetRawTransactionVerboseAsync返回可使用的类型的实例
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅getrawtransactionverbose。
func (c *Client) GetRawTransactionVerboseAsync(txHash *chainhash.Hash) FutureGetRawTransactionVerboseResult {
	hash := ""
	if txHash != nil {
		hash = txHash.String()
	}

	cmd := btcjson.NewGetRawTransactionCmd(hash, btcjson.Int(1))
	return c.sendCmd(cmd)
}

//getrawtransactionverbose返回有关给定事务的信息
//它的散列。
//
//请参阅getrawtransaction以仅获取已反序列化的事务。
func (c *Client) GetRawTransactionVerbose(txHash *chainhash.Hash) (*btcjson.TxRawResult, error) {
	return c.GetRawTransactionVerboseAsync(txHash).Receive()
}

//FutureCoderawtransactionResult是未来交付结果的承诺
//decoderawtransactionasync RPC调用（或适用的错误）。
type FutureDecodeRawTransactionResult chan *response

//接收等待未来承诺的响应并返回信息
//关于给定其序列化字节的事务。
func (r FutureDecodeRawTransactionResult) Receive() (*btcjson.TxRawResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为DecodeRawTransaction结果对象。
	var rawTxResult btcjson.TxRawResult
	err = json.Unmarshal(res, &rawTxResult)
	if err != nil {
		return nil, err
	}

	return &rawTxResult, nil
}

//decoderawtransactionasync返回可用于
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅decoderawtransaction。
func (c *Client) DecodeRawTransactionAsync(serializedTx []byte) FutureDecodeRawTransactionResult {
	txHex := hex.EncodeToString(serializedTx)
	cmd := btcjson.NewDecodeRawTransactionCmd(txHex)
	return c.sendCmd(cmd)
}

//decoderawtransaction返回给定其
//序列化字节。
func (c *Client) DecodeRawTransaction(serializedTx []byte) (*btcjson.TxRawResult, error) {
	return c.DecodeRawTransactionAsync(serializedTx).Receive()
}

//FutureCreaterawtransactionResult是未来交付结果的承诺
//创建rawtransactionasync RPC调用（或适用的错误）。
type FutureCreateRawTransactionResult chan *response

//receive等待将来承诺的响应并返回新的
//交易支出提供的输入并发送给
//地址。
func (r FutureCreateRawTransactionResult) Receive() (*wire.MsgTx, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为字符串。
	var txHex string
	err = json.Unmarshal(res, &txHex)
	if err != nil {
		return nil, err
	}

//将序列化事务十六进制解码为原始字节。
	serializedTx, err := hex.DecodeString(txHex)
	if err != nil {
		return nil, err
	}

//反序列化事务并返回它。
	var msgTx wire.MsgTx
	if err := msgTx.Deserialize(bytes.NewReader(serializedTx)); err != nil {
		return nil, err
	}
	return &msgTx, nil
}

//createrawtransactionasync返回可用于
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅createrawtransaction。
func (c *Client) CreateRawTransactionAsync(inputs []btcjson.TransactionInput,
	amounts map[btcutil.Address]btcutil.Amount, lockTime *int64) FutureCreateRawTransactionResult {

	convertedAmts := make(map[string]float64, len(amounts))
	for addr, amount := range amounts {
		convertedAmts[addr.String()] = amount.ToBTC()
	}
	cmd := btcjson.NewCreateRawTransactionCmd(inputs, convertedAmts, lockTime)
	return c.sendCmd(cmd)
}

//CreateRawTransaction返回使用所提供输入的新事务
//并发送至所提供的地址。
func (c *Client) CreateRawTransaction(inputs []btcjson.TransactionInput,
	amounts map[btcutil.Address]btcutil.Amount, lockTime *int64) (*wire.MsgTx, error) {

	return c.CreateRawTransactionAsync(inputs, amounts, lockTime).Receive()
}

//FutureSendrawTransactionResult是未来交付结果的承诺
//sendrawtransactionasync RPC调用（或适用的错误）。
type FutureSendRawTransactionResult chan *response

//receive等待将来承诺的响应并返回结果
//将编码的事务提交到服务器，然后将其转发给
//网络。
func (r FutureSendRawTransactionResult) Receive() (*chainhash.Hash, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为字符串。
	var txHashStr string
	err = json.Unmarshal(res, &txHashStr)
	if err != nil {
		return nil, err
	}

	return chainhash.NewHashFromStr(txHashStr)
}

//sendrawtransactionasync返回可用于获取
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅sendrawtransaction。
func (c *Client) SendRawTransactionAsync(tx *wire.MsgTx, allowHighFees bool) FutureSendRawTransactionResult {
	txHex := ""
	if tx != nil {
//序列化事务并转换为十六进制字符串。
		buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
		if err := tx.Serialize(buf); err != nil {
			return newFutureError(err)
		}
		txHex = hex.EncodeToString(buf.Bytes())
	}

	cmd := btcjson.NewSendRawTransactionCmd(txHex, &allowHighFees)
	return c.sendCmd(cmd)
}

//sendrawtransaction将编码的事务提交到服务器，服务器将
//然后将其中继到网络。
func (c *Client) SendRawTransaction(tx *wire.MsgTx, allowHighFees bool) (*chainhash.Hash, error) {
	return c.SendRawTransactionAsync(tx, allowHighFees).Receive()
}

//未来设计和交易结果是未来交付结果的承诺
//SignRawTransactionAsync族的RPC调用之一（或
//适用错误）。
type FutureSignRawTransactionResult chan *response

//receive等待将来承诺的响应并返回
//已签名的事务以及是否所有输入现在都已签名。
func (r FutureSignRawTransactionResult) Receive() (*wire.MsgTx, bool, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, false, err
	}

//取消标记为signrawtransaction结果。
	var signRawTxResult btcjson.SignRawTransactionResult
	err = json.Unmarshal(res, &signRawTxResult)
	if err != nil {
		return nil, false, err
	}

//将序列化事务十六进制解码为原始字节。
	serializedTx, err := hex.DecodeString(signRawTxResult.Hex)
	if err != nil {
		return nil, false, err
	}

//反序列化事务并返回它。
	var msgTx wire.MsgTx
	if err := msgTx.Deserialize(bytes.NewReader(serializedTx)); err != nil {
		return nil, false, err
	}

	return &msgTx, signRawTxResult.Complete, nil
}

//signrawtransactionasync返回可用于获取
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅signrawtransaction。
func (c *Client) SignRawTransactionAsync(tx *wire.MsgTx) FutureSignRawTransactionResult {
	txHex := ""
	if tx != nil {
//序列化事务并转换为十六进制字符串。
		buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
		if err := tx.Serialize(buf); err != nil {
			return newFutureError(err)
		}
		txHex = hex.EncodeToString(buf.Bytes())
	}

	cmd := btcjson.NewSignRawTransactionCmd(txHex, nil, nil, nil)
	return c.sendCmd(cmd)
}

//signrawtransaction对传递的事务的输入进行签名，并返回
//已签名的事务以及是否所有输入现在都已签名。
//
//此函数假定RPC服务器已经知道输入事务，并且
//需要签名并使用
//默认签名哈希类型。使用其中一个标志transaction变体
//如果需要，请指定该信息。
func (c *Client) SignRawTransaction(tx *wire.MsgTx) (*wire.MsgTx, bool, error) {
	return c.SignRawTransactionAsync(tx).Receive()
}

//signrawtransaction2async返回可用于
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅signrawtransaction2。
func (c *Client) SignRawTransaction2Async(tx *wire.MsgTx, inputs []btcjson.RawTxInput) FutureSignRawTransactionResult {
	txHex := ""
	if tx != nil {
//序列化事务并转换为十六进制字符串。
		buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
		if err := tx.Serialize(buf); err != nil {
			return newFutureError(err)
		}
		txHex = hex.EncodeToString(buf.Bytes())
	}

	cmd := btcjson.NewSignRawTransactionCmd(txHex, &inputs, nil, nil)
	return c.sendCmd(cmd)
}

//signrawtransaction2为给定列表的已传递事务的输入签名
//有关执行签名所需的输入事务的信息
//过程。
//
//只有需要指定的输入事务才是
//RPC服务器不知道。已知的输入事务将
//与指定的事务合并。
//
//如果RPC服务器已经知道输入，请参阅signrawtransaction
//交易。
func (c *Client) SignRawTransaction2(tx *wire.MsgTx, inputs []btcjson.RawTxInput) (*wire.MsgTx, bool, error) {
	return c.SignRawTransaction2Async(tx, inputs).Receive()
}

//signrawtransaction3async返回可用于
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅signrawtransaction3。
func (c *Client) SignRawTransaction3Async(tx *wire.MsgTx,
	inputs []btcjson.RawTxInput,
	privKeysWIF []string) FutureSignRawTransactionResult {

	txHex := ""
	if tx != nil {
//序列化事务并转换为十六进制字符串。
		buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
		if err := tx.Serialize(buf); err != nil {
			return newFutureError(err)
		}
		txHex = hex.EncodeToString(buf.Bytes())
	}

	cmd := btcjson.NewSignRawTransactionCmd(txHex, &inputs, &privKeysWIF,
		nil)
	return c.sendCmd(cmd)
}

//signrawtransaction3为给定列表的已传递事务的输入签名
//关于额外输入事务和私钥列表的信息
//执行签名过程所需的。私钥必须在钱包中。
//导入格式（WIF）。
//
//只有需要指定的输入事务才是
//RPC服务器不知道。已知的输入事务将
//与指定的事务合并。这意味着交易清单
//如果RPC服务器已经知道所有输入，则输入可以为零。
//
//注意：与输入事务的合并功能不同，只有
//将使用指定的私钥，因此即使服务器已经知道
//在私钥中，不会使用它们。
//
//如果RPC服务器已经知道输入，请参阅signrawtransaction
//事务和私钥或signrawtransaction2，如果它已经知道
//私钥。
func (c *Client) SignRawTransaction3(tx *wire.MsgTx,
	inputs []btcjson.RawTxInput,
	privKeysWIF []string) (*wire.MsgTx, bool, error) {

	return c.SignRawTransaction3Async(tx, inputs, privKeysWIF).Receive()
}

//signrawtransaction4async返回可用于
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅signrawtransaction4。
func (c *Client) SignRawTransaction4Async(tx *wire.MsgTx,
	inputs []btcjson.RawTxInput, privKeysWIF []string,
	hashType SigHashType) FutureSignRawTransactionResult {

	txHex := ""
	if tx != nil {
//序列化事务并转换为十六进制字符串。
		buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
		if err := tx.Serialize(buf); err != nil {
			return newFutureError(err)
		}
		txHex = hex.EncodeToString(buf.Bytes())
	}

	cmd := btcjson.NewSignRawTransactionCmd(txHex, &inputs, &privKeysWIF,
		btcjson.String(string(hashType)))
	return c.sendCmd(cmd)
}

//signrawtransaction4使用
//指定的签名哈希类型给出了有关
//输入事务和执行所需的潜在私钥列表
//签署过程。如有规定，私人钥匙必须放在钱包里。
//导入格式（WIF）。
//
//需要指定的唯一输入事务是RPC服务器
//还不知道。这意味着事务输入列表可以为零。
//如果RPC服务器已经知道了它们。
//
//注意：与输入事务的合并功能不同，只有
//将使用指定的私钥，因此即使服务器已经知道
//在私钥中，不会使用它们。私钥列表可以是
//在这种情况下，将使用RPC服务器知道的任何私钥。
//
//仅当非默认签名哈希类型为
//渴望的。否则，如果RPC服务器已经知道，请参阅signrawtransaction
//输入事务和私钥，signrawtransaction2
//知道私钥，如果不知道二者，则标识RAWTransaction3。
func (c *Client) SignRawTransaction4(tx *wire.MsgTx,
	inputs []btcjson.RawTxInput, privKeysWIF []string,
	hashType SigHashType) (*wire.MsgTx, bool, error) {

	return c.SignRawTransaction4Async(tx, inputs, privKeysWIF,
		hashType).Receive()
}

//未来搜索结果是未来交付结果的承诺
//searchrawtransactionsasync RPC调用（或适用的错误）。
type FutureSearchRawTransactionsResult chan *response

//receive等待将来承诺的响应并返回
//找到原始交易记录。
func (r FutureSearchRawTransactionsResult) Receive() ([]*wire.MsgTx, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//取消标记为字符串数组。
	var searchRawTxnsResult []string
	err = json.Unmarshal(res, &searchRawTxnsResult)
	if err != nil {
		return nil, err
	}

//对每个事务进行解码和反序列化。
	msgTxns := make([]*wire.MsgTx, 0, len(searchRawTxnsResult))
	for _, hexTx := range searchRawTxnsResult {
//将序列化事务十六进制解码为原始字节。
		serializedTx, err := hex.DecodeString(hexTx)
		if err != nil {
			return nil, err
		}

//反序列化事务并将其添加到结果切片。
		var msgTx wire.MsgTx
		err = msgTx.Deserialize(bytes.NewReader(serializedTx))
		if err != nil {
			return nil, err
		}
		msgTxns = append(msgTxns, &msgTx)
	}

	return msgTxns, nil
}

//searchrawtransactionsasync返回可用于
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅SearchRawTransactions。
func (c *Client) SearchRawTransactionsAsync(address btcutil.Address, skip, count int, reverse bool, filterAddrs []string) FutureSearchRawTransactionsResult {
	addr := address.EncodeAddress()
	verbose := btcjson.Int(0)
	cmd := btcjson.NewSearchRawTransactionsCmd(addr, verbose, &skip, &count,
		nil, &reverse, &filterAddrs)
	return c.sendCmd(cmd)
}

//searchrawtransactions返回涉及传递地址的事务。
//
//注意：链服务器通常不提供此功能，除非
//具体启用。
//
//请参阅searchrawtransactionsverbose以检索具有
//有关事务而不是事务本身的信息。
func (c *Client) SearchRawTransactions(address btcutil.Address, skip, count int, reverse bool, filterAddrs []string) ([]*wire.MsgTx, error) {
	return c.SearchRawTransactionsAsync(address, skip, count, reverse, filterAddrs).Receive()
}

//FutureSearchRawtransactionsVerboseResult是未来交付
//searchrawtransactionsverboseasync rpc调用的结果（或
//适用错误）。
type FutureSearchRawTransactionsVerboseResult chan *response

//receive等待将来承诺的响应并返回
//找到原始交易记录。
func (r FutureSearchRawTransactionsVerboseResult) Receive() ([]*btcjson.SearchRawTransactionsResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//取消标记为原始事务结果数组。
	var result []*btcjson.SearchRawTransactionsResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

//searchrawtransactionsverboseasync返回的实例类型可以是
//用于通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅searchrawtransactionsverbose。
func (c *Client) SearchRawTransactionsVerboseAsync(address btcutil.Address, skip,
	count int, includePrevOut, reverse bool, filterAddrs *[]string) FutureSearchRawTransactionsVerboseResult {

	addr := address.EncodeAddress()
	verbose := btcjson.Int(1)
	var prevOut *int
	if includePrevOut {
		prevOut = btcjson.Int(1)
	}
	cmd := btcjson.NewSearchRawTransactionsCmd(addr, verbose, &skip, &count,
		prevOut, &reverse, filterAddrs)
	return c.sendCmd(cmd)
}

//searchrawtransactionsverbose返回描述
//涉及传递地址的事务。
//
//注意：链服务器通常不提供此功能，除非
//具体启用。
//
//请参阅SearchRawTransactions以检索原始事务列表。
func (c *Client) SearchRawTransactionsVerbose(address btcutil.Address, skip,
	count int, includePrevOut, reverse bool, filterAddrs []string) ([]*btcjson.SearchRawTransactionsResult, error) {

	return c.SearchRawTransactionsVerboseAsync(address, skip, count,
		includePrevOut, reverse, &filterAddrs).Receive()
}

//FutureCodeScriptResult是未来交付结果的承诺
//解码脚本异步RPC调用（或适用的错误）。
type FutureDecodeScriptResult chan *response

//接收等待未来承诺的响应并返回信息
//关于给定其序列化字节的脚本。
func (r FutureDecodeScriptResult) Receive() (*btcjson.DecodeScriptResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为解码脚本结果对象。
	var decodeScriptResult btcjson.DecodeScriptResult
	err = json.Unmarshal(res, &decodeScriptResult)
	if err != nil {
		return nil, err
	}

	return &decodeScriptResult, nil
}

//DecodeScriptAsync返回可用于
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅decodescript。
func (c *Client) DecodeScriptAsync(serializedScript []byte) FutureDecodeScriptResult {
	scriptHex := hex.EncodeToString(serializedScript)
	cmd := btcjson.NewDecodeScriptCmd(scriptHex)
	return c.sendCmd(cmd)
}

//decodescript返回给定序列化字节的脚本信息。
func (c *Client) DecodeScript(serializedScript []byte) (*btcjson.DecodeScriptResult, error) {
	return c.DecodeScriptAsync(serializedScript).Receive()
}
