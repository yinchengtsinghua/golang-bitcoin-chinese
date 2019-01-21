
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
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

//FutureDebugLevelResult是未来交付
//调试异步RPC调用（或适用的错误）。
type FutureDebugLevelResult chan *response

//receive等待将来承诺的响应并返回结果
//将调试日志记录级别设置为传递的级别规范或
//特殊关键字“show”的可用子系统列表。
func (r FutureDebugLevelResult) Receive() (string, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return "", err
	}

//将结果取消显示为字符串。
	var result string
	err = json.Unmarshal(res, &result)
	if err != nil {
		return "", err
	}
	return result, nil
}

//debuglevelasync返回可用于获取
//通过调用上的接收函数，在将来某个时间的RPC结果
//返回的实例。
//
//有关块版本和更多详细信息，请参见DebugLevel。
//
//注意：这是BTCD扩展。
func (c *Client) DebugLevelAsync(levelSpec string) FutureDebugLevelResult {
	cmd := btcjson.NewDebugLevelCmd(levelSpec)
	return c.sendCmd(cmd)
}

//DebugLevel动态地将调试日志级别设置为传递的级别
//规范。
//
//级别spec可以是调试级别，也可以是以下形式：
//<subsystem>=<level>，<subsystem2>=<level2>，…
//
//此外，特殊关键字“show”可用于获取
//可用子系统。
//
//注意：这是BTCD扩展。
func (c *Client) DebugLevel(levelSpec string) (string, error) {
	return c.DebugLevelAsync(levelSpec).Receive()
}

//FutureCreateEncryptedAlletResult是未来交付错误的承诺
//CreateCencryptedAlletAsync RPC调用的结果。
type FutureCreateEncryptedWalletResult chan *response

//receive等待并返回将来承诺的错误响应。
func (r FutureCreateEncryptedWalletResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

//CreateCencryptedAlletAsync返回可用于
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅CreateEncryptedAllet。
//
//注意：这是btcwallet扩展。
func (c *Client) CreateEncryptedWalletAsync(passphrase string) FutureCreateEncryptedWalletResult {
	cmd := btcjson.NewCreateEncryptedWalletCmd(passphrase)
	return c.sendCmd(cmd)
}

//CreateCencryptedAllet请求创建加密钱包。钱包
//由btcwallet管理的只使用加密的私钥写入磁盘，
//而且不可能在飞行中生成钱包，因为它需要用户输入
//加密密码短语。此RPC指定密码短语并指示
//钱包的创造。如果钱包已经打开，或者
//新钱包无法写入磁盘。
//
//注意：这是btcwallet扩展。
func (c *Client) CreateEncryptedWallet(passphrase string) error {
	return c.CreateEncryptedWalletAsync(passphrase).Receive()
}

//未来发展趋势结果是未来交付成果的承诺。
//ListAddressTransactionsAsync RPC调用（或适用的错误）。
type FutureListAddressTransactionsResult chan *response

//接收等待未来承诺的响应并返回信息
//关于与提供的地址关联的所有事务。
func (r FutureListAddressTransactionsResult) Receive() ([]btcjson.ListTransactionsResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为ListTransactions对象数组。
	var transactions []btcjson.ListTransactionsResult
	err = json.Unmarshal(res, &transactions)
	if err != nil {
		return nil, err
	}
	return transactions, nil
}

//ListAddressTransactionsAsync返回可使用的类型的实例
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅ListAddressTransactions。
//
//注意：这是BTCD扩展。
func (c *Client) ListAddressTransactionsAsync(addresses []btcutil.Address, account string) FutureListAddressTransactionsResult {
//将地址转换为字符串。
	addrs := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		addrs = append(addrs, addr.EncodeAddress())
	}
	cmd := btcjson.NewListAddressTransactionsCmd(addrs, &account)
	return c.sendCmd(cmd)
}

//ListAddressTransactions返回有关所有关联事务的信息
//提供地址。
//
//注意：这是btcwallet扩展。
func (c *Client) ListAddressTransactions(addresses []btcutil.Address, account string) ([]btcjson.ListTransactionsResult, error) {
	return c.ListAddressTransactionsAsync(addresses, account).Receive()
}

//FutureGetBestBlockResult是未来交付
//GetBestBlockAsync RPC调用（或适用的错误）。
type FutureGetBestBlockResult chan *response

//receive等待将来承诺的响应并返回哈希
//以及最长（最好）链条中的滑轮高度。
func (r FutureGetBestBlockResult) Receive() (*chainhash.Hash, int32, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, 0, err
	}

//将结果取消标记为GetBestBlock结果对象。
	var bestBlock btcjson.GetBestBlockResult
	err = json.Unmarshal(res, &bestBlock)
	if err != nil {
		return nil, 0, err
	}

//Convert to hash from string.
	hash, err := chainhash.NewHashFromStr(bestBlock.Hash)
	if err != nil {
		return nil, 0, err
	}

	return hash, bestBlock.Height, nil
}

//GetBestBlockAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅GetBestBlock。
//
//注意：这是BTCD扩展。
func (c *Client) GetBestBlockAsync() FutureGetBestBlockResult {
	cmd := btcjson.NewGetBestBlockCmd()
	return c.sendCmd(cmd)
}

//GetBestBlock返回块的最长哈希值和高度（最佳）
//链。
//
//注意：这是BTCD扩展。
func (c *Client) GetBestBlock() (*chainhash.Hash, int32, error) {
	return c.GetBestBlockAsync().Receive()
}

//FutureGetCurrentNetResult是未来交付
//getcurrentNetAsync RPC调用（或适用错误）。
type FutureGetCurrentNetResult chan *response

//Receive waits for the response promised by the future and returns the network
//服务器正在运行。
func (r FutureGetCurrentNetResult) Receive() (wire.BitcoinNet, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return 0, err
	}

//将结果取消标记为Int64。
	var net int64
	err = json.Unmarshal(res, &net)
	if err != nil {
		return 0, err
	}

	return wire.BitcoinNet(net), nil
}

//getcurrentNetAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅getcurrentnet。
//
//注意：这是BTCD扩展。
func (c *Client) GetCurrentNetAsync() FutureGetCurrentNetResult {
	cmd := btcjson.NewGetCurrentNetCmd()
	return c.sendCmd(cmd)
}

//getcurrentnet返回服务器运行的网络。
//
//注意：这是BTCD扩展。
func (c *Client) GetCurrentNet() (wire.BitcoinNet, error) {
	return c.GetCurrentNetAsync().Receive()
}

//FuturegeTheadersresult是未来交付结果的承诺
//GetHeaders RPC调用（或适用的错误）。
//
//注意：这是从中导入的BTCSuite扩展
//github.com/decred/dcrrpcclient.
type FutureGetHeadersResult chan *response

//receive等待将来承诺的响应并返回
//GetHeaders结果。
//
//注意：这是从中导入的BTCSuite扩展
//github.com/decred/dcrrpcclient.
func (r FutureGetHeadersResult) Receive() ([]wire.BlockHeader, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为一段字符串。
	var result []string
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}

//将[]字符串反序列化为[]Wire.BlockHeader。
	headers := make([]wire.BlockHeader, len(result))
	for i, headerHex := range result {
		serialized, err := hex.DecodeString(headerHex)
		if err != nil {
			return nil, err
		}
		err = headers[i].Deserialize(bytes.NewReader(serialized))
		if err != nil {
			return nil, err
		}
	}
	return headers, nil
}

//GetHeaderAsync返回可用于获取结果的类型的实例
//通过在返回的实例上调用接收函数，在将来某个时间调用RPC。
//
//有关阻止版本和更多详细信息，请参阅getheaders。
//
//注意：这是从中导入的BTCSuite扩展
//github.com/decred/dcrrpcclient.
func (c *Client) GetHeadersAsync(blockLocators []chainhash.Hash, hashStop *chainhash.Hash) FutureGetHeadersResult {
	locators := make([]string, len(blockLocators))
	for i := range blockLocators {
		locators[i] = blockLocators[i].String()
	}
	hash := ""
	if hashStop != nil {
		hash = hashStop.String()
	}
	cmd := btcjson.NewGetHeadersCmd(locators, hash)
	return c.sendCmd(cmd)
}

//getheaders模拟有线协议getheaders和headers消息
//在中的第一个已知块之后返回主链上的所有头段
//定位器，直到块散列与hashstop匹配。
//
//注意：这是从中导入的BTCSuite扩展
//github.com/decred/dcrrpcclient.
func (c *Client) GetHeaders(blockLocators []chainhash.Hash, hashStop *chainhash.Hash) ([]wire.BlockHeader, error) {
	return c.GetHeadersAsync(blockLocators, hashStop).Receive()
}

//未来出口观察墙结果是未来交付结果的承诺
//exportwatchingwalletasync RPC调用（或适用的错误）。
type FutureExportWatchingWalletResult chan *response

//receive等待将来承诺的响应并返回
//出口钱包。
func (r FutureExportWatchingWalletResult) Receive() ([]byte, []byte, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, nil, err
	}

//将结果取消标记为JSON对象。
	var obj map[string]interface{}
	err = json.Unmarshal(res, &obj)
	if err != nil {
		return nil, nil, err
	}

//检查对象中的wallet和tx字符串字段。
	base64Wallet, ok := obj["wallet"].(string)
	if !ok {
		return nil, nil, fmt.Errorf("unexpected response type for "+
			"exportwatchingwallet 'wallet' field: %T\n",
			obj["wallet"])
	}
	base64TxStore, ok := obj["tx"].(string)
	if !ok {
		return nil, nil, fmt.Errorf("unexpected response type for "+
			"exportwatchingwallet 'tx' field: %T\n",
			obj["tx"])
	}

	walletBytes, err := base64.StdEncoding.DecodeString(base64Wallet)
	if err != nil {
		return nil, nil, err
	}

	txStoreBytes, err := base64.StdEncoding.DecodeString(base64TxStore)
	if err != nil {
		return nil, nil, err
	}

	return walletBytes, txStoreBytes, nil

}

//exportwatchingwalletasync返回可用于
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅exportwatchingwallet。
//
//注意：这是btcwallet扩展。
func (c *Client) ExportWatchingWalletAsync(account string) FutureExportWatchingWalletResult {
	cmd := btcjson.NewExportWatchingWalletCmd(&account, btcjson.Bool(true))
	return c.sendCmd(cmd)
}

//exportwatchingwallet返回仅监视版本的原始字节
//wallet.bin和tx.bin分别用于指定的帐户，可以
//btcwallet用于启用没有私钥的钱包
//需要花费资金。
//
//注意：这是btcwallet扩展。
func (c *Client) ExportWatchingWallet(account string) ([]byte, []byte, error) {
	return c.ExportWatchingWalletAsync(account).Receive()
}

//未来会话结果是未来交付结果的承诺
//SessionAsync RPC invocation (or an applicable error).
type FutureSessionResult chan *response

//receive等待将来承诺的响应并返回
//会话结果。
func (r FutureSessionResult) Receive() (*btcjson.SessionResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为会话结果对象。
	var session btcjson.SessionResult
	err = json.Unmarshal(res, &session)
	if err != nil {
		return nil, err
	}

	return &session, nil
}

//sessionasync返回可用于获取结果的类型的实例
//在将来的某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅会话。
//
//注意：这是BTCSuite扩展。
func (c *Client) SessionAsync() FutureSessionResult {
//HTTP POST模式不支持。
	if c.config.HTTPPostMode {
		return newFutureError(ErrWebsocketsRequired)
	}

	cmd := btcjson.NewSessionCmd()
	return c.sendCmd(cmd)
}

//会话返回有关WebSocket客户端当前连接的详细信息。
//
//此RPC要求客户端以WebSocket模式运行。
//
//注意：这是BTCSuite扩展。
func (c *Client) Session() (*btcjson.SessionResult, error) {
	return c.SessionAsync().Receive()
}

//FutureEversionResult是未来交付版本结果的承诺
//RPC调用（或适用的错误）。
//
//注意：这是从中导入的BTCSuite扩展
//github.com/decred/dcrrpcclient.
type FutureVersionResult chan *response

//receive等待将来承诺的响应并返回版本
//结果。
//
//注意：这是从中导入的BTCSuite扩展
//github.com/decred/dcrrpcclient.
func (r FutureVersionResult) Receive() (map[string]btcjson.VersionResult,
	error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为版本结果对象。
	var vr map[string]btcjson.VersionResult
	err = json.Unmarshal(res, &vr)
	if err != nil {
		return nil, err
	}

	return vr, nil
}

//VersionAsync返回可用于获取结果的类型的实例
//在将来的某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅版本。
//
//注意：这是从中导入的BTCSuite扩展
//GITHUB/COMPUD/DCRRPCEclipse。
func (c *Client) VersionAsync() FutureVersionResult {
	cmd := btcjson.NewVersionCmd()
	return c.sendCmd(cmd)
}

//版本返回有关服务器的JSON-RPCAPI版本的信息。
//
//注意：这是从中导入的BTCSuite扩展
//GITHUB/COMPUD/DCRRPCEclipse。
func (c *Client) Version() (map[string]btcjson.VersionResult, error) {
	return c.VersionAsync().Receive()
}
