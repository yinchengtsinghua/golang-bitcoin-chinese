
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
	"encoding/json"
	"strconv"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

//***************************
//交易列表功能
//***************************

//FutureGetTransactionResult是未来交付结果的承诺
//GetTransactionAsync RPC调用（或适用的错误）。
type FutureGetTransactionResult chan *response

//receive等待将来承诺的响应并返回详细信息
//有关钱包交易的信息。
func (r FutureGetTransactionResult) Receive() (*btcjson.GetTransactionResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为GetTransaction结果对象
	var getTx btcjson.GetTransactionResult
	err = json.Unmarshal(res, &getTx)
	if err != nil {
		return nil, err
	}

	return &getTx, nil
}

//GetTransactionAsync返回可用于获取
//通过调用上的接收函数，在将来某个时间的RPC结果
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅GetTransaction。
func (c *Client) GetTransactionAsync(txHash *chainhash.Hash) FutureGetTransactionResult {
	hash := ""
	if txHash != nil {
		hash = txHash.String()
	}

	cmd := btcjson.NewGetTransactionCmd(hash, nil)
	return c.sendCmd(cmd)
}

//GetTransaction返回有关钱包交易的详细信息。
//
//请参阅getrawtransaction以返回原始事务。
func (c *Client) GetTransaction(txHash *chainhash.Hash) (*btcjson.GetTransactionResult, error) {
	return c.GetTransactionAsync(txHash).Receive()
}

//FutureListTransactionsResult是未来交付
//ListTransactionsAsync、ListTransactionsCountAsync或
//ListTransactionsCountFromAsync RPC调用（或适用的错误）。
type FutureListTransactionsResult chan *response

//Receive等待将来承诺的响应，并返回
//最近的交易。
func (r FutureListTransactionsResult) Receive() ([]btcjson.ListTransactionsResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为ListTransaction结果对象的数组。
	var transactions []btcjson.ListTransactionsResult
	err = json.Unmarshal(res, &transactions)
	if err != nil {
		return nil, err
	}

	return transactions, nil
}

//ListTransactionSasync返回可用于获取的类型的实例
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅ListTransactions。
func (c *Client) ListTransactionsAsync(account string) FutureListTransactionsResult {
	cmd := btcjson.NewListTransactionsCmd(&account, nil, nil, nil)
	return c.sendCmd(cmd)
}

//ListTransactions返回最新事务的列表。
//
//请参见ListTransactionsCount和ListTransactionsCountFrom以控制
//分别返回的事务数和起始点。
func (c *Client) ListTransactions(account string) ([]btcjson.ListTransactionsResult, error) {
	return c.ListTransactionsAsync(account).Receive()
}

//ListTransactionScountAsync返回一个可用于
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅ListTransactionsCount。
func (c *Client) ListTransactionsCountAsync(account string, count int) FutureListTransactionsResult {
	cmd := btcjson.NewListTransactionsCmd(&account, &count, nil, nil)
	return c.sendCmd(cmd)
}

//ListTransactionsCount返回最新事务的列表
//到通过的计数。
//
//请参见的ListTransactions和ListTransactionsCountFrom函数
//不同的选择。
func (c *Client) ListTransactionsCount(account string, count int) ([]btcjson.ListTransactionsResult, error) {
	return c.ListTransactionsCountAsync(account, count).Receive()
}

//ListTransactionsCountFromAsync返回可使用的类型的实例
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅ListTransactionsCountFrom。
func (c *Client) ListTransactionsCountFromAsync(account string, count, from int) FutureListTransactionsResult {
	cmd := btcjson.NewListTransactionsCmd(&account, &count, &from, nil)
	return c.sendCmd(cmd)
}

//listTransactionsCountFrom返回最新事务的列表
//跳过第一个“From”事务时传递的计数。
//
//请参见ListTransactions和ListTransactionsCount函数以使用默认值。
func (c *Client) ListTransactionsCountFrom(account string, count, from int) ([]btcjson.ListTransactionsResult, error) {
	return c.ListTransactionsCountFromAsync(account, count, from).Receive()
}

//FutureListUnstresult是未来交付
//listunnpentminasync、listunnpentminasync、listunnpentminmaxasync或
//listunnpentminmaxaddressesasync rpc调用（或适用的错误）。
type FutureListUnspentResult chan *response

//receive等待将来承诺的响应并返回所有
//由RPC调用返回的未用钱包事务输出。如果
//通过调用listunnaminasync、listunnaminmaxasync、
//或listunnpentminmaxaddressesasync，范围可能受
//RPC调用的参数。
func (r FutureListUnspentResult) Receive() ([]btcjson.ListUnspentResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为ListunPent结果的数组。
	var unspent []btcjson.ListUnspentResult
	err = json.Unmarshal(res, &unspent)
	if err != nil {
		return nil, err
	}

	return unspent, nil
}

//listunPentaSync返回可用于获取
//通过调用receive函数在将来某个时间的rpc结果
//在返回的实例上。
//
//有关阻止版本和更多详细信息，请参阅listunspent。
func (c *Client) ListUnspentAsync() FutureListUnspentResult {
	cmd := btcjson.NewListUnspentCmd(nil, nil, nil)
	return c.sendCmd(cmd)
}

//listunnaminasync返回可用于获取
//通过调用receive函数在将来某个时间的rpc结果
//在返回的实例上。
//
//有关阻止版本和更多详细信息，请参阅listunspentmin。
func (c *Client) ListUnspentMinAsync(minConf int) FutureListUnspentResult {
	cmd := btcjson.NewListUnspentCmd(&minConf, nil, nil)
	return c.sendCmd(cmd)
}

//listunspentminmaxasync返回可用于获取的类型的实例
//通过调用receive函数在将来某个时间的rpc结果
//在返回的实例上。
//
//有关阻止版本和更多详细信息，请参见listunspentminmax。
func (c *Client) ListUnspentMinMaxAsync(minConf, maxConf int) FutureListUnspentResult {
	cmd := btcjson.NewListUnspentCmd(&minConf, &maxConf, nil)
	return c.sendCmd(cmd)
}

//listunspentminmaxaddressesasync返回一个类型的实例，该类型可以是
//用于通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅listunspentminmaxaddresses。
func (c *Client) ListUnspentMinMaxAddressesAsync(minConf, maxConf int, addrs []btcutil.Address) FutureListUnspentResult {
	addrStrs := make([]string, 0, len(addrs))
	for _, a := range addrs {
		addrStrs = append(addrStrs, a.EncodeAddress())
	}

	cmd := btcjson.NewListUnspentCmd(&minConf, &maxConf, &addrStrs)
	return c.sendCmd(cmd)
}

//listunspent返回钱包已知的所有未用事务输出，使用
//默认的最小和最大确认数
//过滤器（分别为1和999999）。
func (c *Client) ListUnspent() ([]btcjson.ListUnspentResult, error) {
	return c.ListUnspentAsync().Receive()
}

//listunspentmin返回钱包已知的所有未用事务输出，
//使用指定的最小配置数和默认的
//作为过滤器的最大配置（999999）。
func (c *Client) ListUnspentMin(minConf int) ([]btcjson.ListUnspentResult, error) {
	return c.ListUnspentMinAsync(minConf).Receive()
}

//listunspentminmax返回钱包已知的所有未用事务输出，
//使用指定的最小和最大确认数作为
//过滤器。
func (c *Client) ListUnspentMinMax(minConf, maxConf int) ([]btcjson.ListUnspentResult, error) {
	return c.ListUnspentMinMaxAsync(minConf, maxConf).Receive()
}

//listunspentminmaxaddresses返回所有支付的未暂停事务输出
//钱包中使用指定号码的任何指定地址
//作为筛选器的最小和最大确认数。
func (c *Client) ListUnspentMinMaxAddresses(minConf, maxConf int, addrs []btcutil.Address) ([]btcjson.ListUnspentResult, error) {
	return c.ListUnspentMinMaxAddressesAsync(minConf, maxConf, addrs).Receive()
}

//FutureListsinceBlockResult是未来交付结果的承诺
//listsinceBlockAsync或listsinceBlockMinConfAsync RPC调用（或
//适用错误）。
type FutureListSinceBlockResult chan *response

//receive等待将来承诺的响应并返回所有
//自指定的块散列以来在块中添加的事务，或全部
//如果为零，则为交易。
func (r FutureListSinceBlockResult) Receive() (*btcjson.ListSinceBlockResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为ListsinceBlock结果对象。
	var listResult btcjson.ListSinceBlockResult
	err = json.Unmarshal(res, &listResult)
	if err != nil {
		return nil, err
	}

	return &listResult, nil
}

//listsinceblockasync返回可用于获取的类型的实例
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅listsinceblock。
func (c *Client) ListSinceBlockAsync(blockHash *chainhash.Hash) FutureListSinceBlockResult {
	var hash *string
	if blockHash != nil {
		hash = btcjson.String(blockHash.String())
	}

	cmd := btcjson.NewListSinceBlockCmd(hash, nil, nil)
	return c.sendCmd(cmd)
}

//listsinceblock返回自指定的
//块哈希或所有事务（如果为零），使用默认的
//作为过滤器的最小确认。
//
//请参阅listsincblockminconf以覆盖最小确认数。
func (c *Client) ListSinceBlock(blockHash *chainhash.Hash) (*btcjson.ListSinceBlockResult, error) {
	return c.ListSinceBlockAsync(blockHash).Receive()
}

//listsinceblockminconfasync返回一个可用于
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅listsinceblockminconf。
func (c *Client) ListSinceBlockMinConfAsync(blockHash *chainhash.Hash, minConfirms int) FutureListSinceBlockResult {
	var hash *string
	if blockHash != nil {
		hash = btcjson.String(blockHash.String())
	}

	cmd := btcjson.NewListSinceBlockCmd(hash, &minConfirms, nil)
	return c.sendCmd(cmd)
}

//listsinceblockminconf返回自
//指定的块哈希，如果为零，则使用指定的
//作为筛选器的最小确认数。
//
//请参阅listsinceblock以使用默认的最小确认数。
func (c *Client) ListSinceBlockMinConf(blockHash *chainhash.Hash, minConfirms int) (*btcjson.ListSinceBlockResult, error) {
	return c.ListSinceBlockMinConfAsync(blockHash, minConfirms).Receive()
}

//**********************
//事务发送函数
//**********************

//FutureLockunspentResult是未来交付错误结果的承诺
//LockunPentaSync RPC调用。
type FutureLockUnspentResult chan *response

//receive等待将来承诺的响应并返回结果
//锁定或解锁未释放的输出。
func (r FutureLockUnspentResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

//lockunspentasync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参见Lockunspent。
func (c *Client) LockUnspentAsync(unlock bool, ops []*wire.OutPoint) FutureLockUnspentResult {
	outputs := make([]btcjson.TransactionInput, len(ops))
	for i, op := range ops {
		outputs[i] = btcjson.TransactionInput{
			Txid: op.Hash.String(),
			Vout: op.Index,
		}
	}
	cmd := btcjson.NewLockUnspentCmd(unlock, outputs)
	return c.sendCmd(cmd)
}

//Lockunspent将输出标记为锁定或解锁，具体取决于
//解锁布尔。锁定后，未释放的输出将不会被选择为
//新创建的非原始交易记录的输入，将不会在中返回
//在输出再次标记为“解锁”之前，将来的listunspent结果。
//
//如果unlock为false，则ops中的每个输出点都将标记为locked。如果解锁
//是真的，特定输出在ops（len！中指定。=0），正是那些
//输出将标记为解锁。如果解锁为真且没有输出点
//指定，所有以前锁定的输出都标记为解锁。
//
//输出的锁定或未锁定状态不会写入磁盘和之后
//重新启动钱包进程，此数据将被重置（每个输出都被解锁）。
//
//注意：如果unlock bool是
//反向（即，锁定未释放（真，…）锁定输出），它已经
//保留为unlock以保持与引用客户端API和
//对于已经熟悉锁止式RPC的用户，请避免混淆。
func (c *Client) LockUnspent(unlock bool, ops []*wire.OutPoint) error {
	return c.LockUnspentAsync(unlock, ops).Receive()
}

//未来锁定结果是未来交付结果的承诺
//listlockunspentasync RPC调用（或适用的错误）。
type FutureListLockUnspentResult chan *response

//receive等待将来承诺的响应并返回结果
//所有当前锁定的未暂停输出。
func (r FutureListLockUnspentResult) Receive() ([]*wire.OutPoint, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//取消标记为事务输入数组。
	var inputs []btcjson.TransactionInput
	err = json.Unmarshal(res, &inputs)
	if err != nil {
		return nil, err
	}

//从事务输入结构创建一个输出点切片。
	ops := make([]*wire.OutPoint, len(inputs))
	for i, input := range inputs {
		sha, err := chainhash.NewHashFromStr(input.Txid)
		if err != nil {
			return nil, err
		}
		ops[i] = wire.NewOutPoint(sha, input.Vout)
	}

	return ops, nil
}

//listlockunspentasync返回可用于获取
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅listlockunspent。
func (c *Client) ListLockUnspentAsync() FutureListLockUnspentResult {
	cmd := btcjson.NewListLockUnspentCmd()
	return c.sendCmd(cmd)
}

//listlockunspent返回标记的所有未暂停输出的一部分输出点
//就像被钱包锁住一样。未使用的输出可以标记为锁定，使用
//锁定输出。
func (c *Client) ListLockUnspent() ([]*wire.OutPoint, error) {
	return c.ListLockUnspentAsync().Receive()
}

//Futuresetxfeeresult是未来交付结果的承诺
//settxfeeasync rpc调用（或适用的错误）。
type FutureSetTxFeeResult chan *response

//receive等待将来承诺的响应并返回结果
//设置每千字节的可选事务费，以帮助确保事务
//快速处理。大多数交易是1千字节。
func (r FutureSetTxFeeResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

//settxfeeasync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅setxfee。
func (c *Client) SetTxFeeAsync(fee btcutil.Amount) FutureSetTxFeeResult {
	cmd := btcjson.NewSetTxFeeCmd(fee.ToBTC())
	return c.sendCmd(cmd)
}

//setxfee设置每kb的可选事务费，有助于确保
//事务处理很快。大多数交易是1千字节。
func (c *Client) SetTxFee(fee btcutil.Amount) error {
	return c.SetTxFeeAsync(fee).Receive()
}

//FuturesAndToAddressResult是未来交付
//sendToAddressSync RPC调用（或适用的错误）。
type FutureSendToAddressResult chan *response

//receive等待将来承诺的响应并返回哈希
//将传递的金额发送到给定地址的事务。
func (r FutureSendToAddressResult) Receive() (*chainhash.Hash, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为字符串。
	var txHash string
	err = json.Unmarshal(res, &txHash)
	if err != nil {
		return nil, err
	}

	return chainhash.NewHashFromStr(txHash)
}

//sendToAddressSync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅sendToAddress。
func (c *Client) SendToAddressAsync(address btcutil.Address, amount btcutil.Amount) FutureSendToAddressResult {
	addr := address.EncodeAddress()
	cmd := btcjson.NewSendToAddressCmd(addr, amount.ToBTC(), nil, nil)
	return c.sendCmd(cmd)
}

//sendToAddress将传递的金额发送到给定的地址。
//
//请参阅sendToAddressComment以将注释与中的事务关联
//钱包。注释不是事务的一部分，只是内部的
//钱包。
//
//注意：此功能要求钱包解锁。见
//有关详细信息，请参阅walletpassphrase函数。
func (c *Client) SendToAddress(address btcutil.Address, amount btcutil.Amount) (*chainhash.Hash, error) {
	return c.SendToAddressAsync(address, amount).Receive()
}

//sendToAddressCommentAsync返回可用于
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅sendToAddressComment。
func (c *Client) SendToAddressCommentAsync(address btcutil.Address,
	amount btcutil.Amount, comment,
	commentTo string) FutureSendToAddressResult {

	addr := address.EncodeAddress()
	cmd := btcjson.NewSendToAddressCmd(addr, amount.ToBTC(), &comment,
		&commentTo)
	return c.sendCmd(cmd)
}

//sendToAddressComment将传递的金额发送到给定的地址和存储区
//钱包中提供的评论和评论。comment参数是
//用于交易目的，而注释
//参数不确定用于将事务发送给谁。
//
//注释不是事务的一部分，只是内部的
//钱包。
//
//请参阅sendToAddress以避免使用注释。
//
//注意：此功能要求钱包解锁。见
//有关详细信息，请参阅walletpassphrase函数。
func (c *Client) SendToAddressComment(address btcutil.Address, amount btcutil.Amount, comment, commentTo string) (*chainhash.Hash, error) {
	return c.SendToAddressCommentAsync(address, amount, comment,
		commentTo).Receive()
}

//FuturesEndFromResult是未来交付
//sendFromAsync、sendFromMinConfAsync或sendFromCommentAsync RPC调用
//（或适用的错误）。
type FutureSendFromResult chan *response

//receive等待将来承诺的响应并返回哈希
//使用提供的
//账户作为资金来源。
func (r FutureSendFromResult) Receive() (*chainhash.Hash, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为字符串。
	var txHash string
	err = json.Unmarshal(res, &txHash)
	if err != nil {
		return nil, err
	}

	return chainhash.NewHashFromStr(txHash)
}

//sendFromAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅sendfrom。
func (c *Client) SendFromAsync(fromAccount string, toAddress btcutil.Address, amount btcutil.Amount) FutureSendFromResult {
	addr := toAddress.EncodeAddress()
	cmd := btcjson.NewSendFromCmd(fromAccount, addr, amount.ToBTC(), nil,
		nil, nil)
	return c.sendCmd(cmd)
}

//sendfrom使用提供的
//账户作为资金来源。仅限默认最小数目的基金
//将使用确认。
//
//有关不同的选项，请参阅sendfromminconf和sendfromcomment。
//
//注意：此功能要求钱包解锁。见
//有关详细信息，请参阅walletpassphrase函数。
func (c *Client) SendFrom(fromAccount string, toAddress btcutil.Address, amount btcutil.Amount) (*chainhash.Hash, error) {
	return c.SendFromAsync(fromAccount, toAddress, amount).Receive()
}

//sendFromMinConfAsync返回可用于获取
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅sendfromminconf。
func (c *Client) SendFromMinConfAsync(fromAccount string, toAddress btcutil.Address, amount btcutil.Amount, minConfirms int) FutureSendFromResult {
	addr := toAddress.EncodeAddress()
	cmd := btcjson.NewSendFromCmd(fromAccount, addr, amount.ToBTC(),
		&minConfirms, nil, nil)
	return c.sendCmd(cmd)
}

//sendfromminconf使用
//提供账户作为资金来源。只有通过了
//将使用最低确认。
//
//请参阅sendfrom以使用默认的最低确认数和
//有关其他选项的sendfromcomment。
//
//注意：此功能要求钱包解锁。见
//有关详细信息，请参阅walletpassphrase函数。
func (c *Client) SendFromMinConf(fromAccount string, toAddress btcutil.Address, amount btcutil.Amount, minConfirms int) (*chainhash.Hash, error) {
	return c.SendFromMinConfAsync(fromAccount, toAddress, amount,
		minConfirms).Receive()
}

//sendFromCommentAsync返回可用于获取
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅sendfromcomment。
func (c *Client) SendFromCommentAsync(fromAccount string,
	toAddress btcutil.Address, amount btcutil.Amount, minConfirms int,
	comment, commentTo string) FutureSendFromResult {

	addr := toAddress.EncodeAddress()
	cmd := btcjson.NewSendFromCmd(fromAccount, addr, amount.ToBTC(),
		&minConfirms, &comment, &commentTo)
	return c.sendCmd(cmd)
}

//sendfromcomment使用
//提供账户作为资金来源，并存储提供的评论和
//在钱包中留言。comment参数用于
//当commentto参数不确定时事务的目的
//用于将交易发送给谁。只有通过的资金
//将使用最小确认数。
//
//请参阅sendfrom和sendfromminconf以使用默认值。
//
//注意：此功能要求钱包解锁。见
//有关详细信息，请参阅walletpassphrase函数。
func (c *Client) SendFromComment(fromAccount string, toAddress btcutil.Address,
	amount btcutil.Amount, minConfirms int,
	comment, commentTo string) (*chainhash.Hash, error) {

	return c.SendFromCommentAsync(fromAccount, toAddress, amount,
		minConfirms, comment, commentTo).Receive()
}

//FuturesAndManyResult是未来交付
//sendmanyasync、sendmanyminofasync或sendmanycommentsync rpc调用
//（或适用的错误）。
type FutureSendManyResult chan *response

//receive等待将来承诺的响应并返回哈希
//将多个金额发送到多个地址的事务
//提供账户作为资金来源。
func (r FutureSendManyResult) Receive() (*chainhash.Hash, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消显示为字符串。
	var txHash string
	err = json.Unmarshal(res, &txHash)
	if err != nil {
		return nil, err
	}

	return chainhash.NewHashFromStr(txHash)
}

//sendmanyasync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅sendmany。
func (c *Client) SendManyAsync(fromAccount string, amounts map[btcutil.Address]btcutil.Amount) FutureSendManyResult {
	convertedAmounts := make(map[string]float64, len(amounts))
	for addr, amount := range amounts {
		convertedAmounts[addr.EncodeAddress()] = amount.ToBTC()
	}
	cmd := btcjson.NewSendManyCmd(fromAccount, convertedAmounts, nil, nil)
	return c.sendCmd(cmd)
}

//sendmany使用提供的
//在单一交易中作为资金来源的帐户。只有基金
//将使用默认的最小确认数。
//
//有关不同的选项，请参阅sendmanyminoff和sendmanycomment。
//
//注意：此功能要求钱包解锁。见
//有关详细信息，请参阅walletpassphrase函数。
func (c *Client) SendMany(fromAccount string, amounts map[btcutil.Address]btcutil.Amount) (*chainhash.Hash, error) {
	return c.SendManyAsync(fromAccount, amounts).Receive()
}

//sendmanyminofasync返回可用于获取
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅sendmanyminoff。
func (c *Client) SendManyMinConfAsync(fromAccount string,
	amounts map[btcutil.Address]btcutil.Amount,
	minConfirms int) FutureSendManyResult {

	convertedAmounts := make(map[string]float64, len(amounts))
	for addr, amount := range amounts {
		convertedAmounts[addr.EncodeAddress()] = amount.ToBTC()
	}
	cmd := btcjson.NewSendManyCmd(fromAccount, convertedAmounts,
		&minConfirms, nil)
	return c.sendCmd(cmd)
}

//sendmanyminoff使用
//在单一交易中作为资金来源的账户。只有资金
//通过后，将使用最小确认数。
//
//请参阅sendmany以使用默认的最小确认数和
//有关其他选项的sendmanycomment。
//
//注意：此功能要求钱包解锁。见
//有关详细信息，请参阅walletpassphrase函数。
func (c *Client) SendManyMinConf(fromAccount string,
	amounts map[btcutil.Address]btcutil.Amount,
	minConfirms int) (*chainhash.Hash, error) {

	return c.SendManyMinConfAsync(fromAccount, amounts, minConfirms).Receive()
}

//sendmanycommentsync返回可用于获取的类型的实例
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅sendmanycomment。
func (c *Client) SendManyCommentAsync(fromAccount string,
	amounts map[btcutil.Address]btcutil.Amount, minConfirms int,
	comment string) FutureSendManyResult {

	convertedAmounts := make(map[string]float64, len(amounts))
	for addr, amount := range amounts {
		convertedAmounts[addr.EncodeAddress()] = amount.ToBTC()
	}
	cmd := btcjson.NewSendManyCmd(fromAccount, convertedAmounts,
		&minConfirms, &comment)
	return c.sendCmd(cmd)
}

//sendmanycomment使用
//提供账户作为单一交易的资金来源，并存储
//在钱包里提供了评论。comment参数用于
//在交易中，只有通过
//将使用最低确认。
//
//请参见sendmany和sendmanyminoff以使用默认值。
//
//注意：此功能要求钱包解锁。见
//有关详细信息，请参阅walletpassphrase函数。
func (c *Client) SendManyComment(fromAccount string,
	amounts map[btcutil.Address]btcutil.Amount, minConfirms int,
	comment string) (*chainhash.Hash, error) {

	return c.SendManyCommentAsync(fromAccount, amounts, minConfirms,
		comment).Receive()
}

//**********************
//地址/帐户功能
//**********************

//FutureadMultisigAddressResult是未来交付
//AddMultiSigaddressSync RPC调用（或适用的错误）。
type FutureAddMultisigAddressResult chan *response

//receive等待将来承诺的响应并返回
//需要指定数量签名的多签名地址
//提供的地址。
func (r FutureAddMultisigAddressResult) Receive() (btcutil.Address, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为字符串。
	var addr string
	err = json.Unmarshal(res, &addr)
	if err != nil {
		return nil, err
	}

	return btcutil.DecodeAddress(addr, &chaincfg.MainNetParams)
}

//AddMultiSigaddressSync返回可用于获取的类型的实例
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅addmultisigaddress。
func (c *Client) AddMultisigAddressAsync(requiredSigs int, addresses []btcutil.Address, account string) FutureAddMultisigAddressResult {
	addrs := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		addrs = append(addrs, addr.String())
	}

	cmd := btcjson.NewAddMultisigAddressCmd(requiredSigs, addrs, &account)
	return c.sendCmd(cmd)
}

//addmultisigaddress添加需要指定的
//钱包所提供地址的签名数。
func (c *Client) AddMultisigAddress(requiredSigs int, addresses []btcutil.Address, account string) (btcutil.Address, error) {
	return c.AddMultisigAddressAsync(requiredSigs, addresses,
		account).Receive()
}

//FutureCreateMultisigResult是未来交付
//创建多协议同步RPC调用（或适用的错误）。
type FutureCreateMultisigResult chan *response

//receive等待将来承诺的响应并返回
//多签名地址和脚本需要兑换它。
func (r FutureCreateMultisigResult) Receive() (*btcjson.CreateMultiSigResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为CreateMulsig结果对象。
	var multisigRes btcjson.CreateMultiSigResult
	err = json.Unmarshal(res, &multisigRes)
	if err != nil {
		return nil, err
	}

	return &multisigRes, nil
}

//createMultisigasync返回可用于获取的类型的实例
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参见CreateMulsig。
func (c *Client) CreateMultisigAsync(requiredSigs int, addresses []btcutil.Address) FutureCreateMultisigResult {
	addrs := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		addrs = append(addrs, addr.String())
	}

	cmd := btcjson.NewCreateMultisigCmd(requiredSigs, addrs)
	return c.sendCmd(cmd)
}

//CreateMultisig创建需要指定的
//所提供地址的签名数并返回
//多签名地址和脚本需要兑换它。
func (c *Client) CreateMultisig(requiredSigs int, addresses []btcutil.Address) (*btcjson.CreateMultiSigResult, error) {
	return c.CreateMultisigAsync(requiredSigs, addresses).Receive()
}

//FutureCreateneWaccountResult是未来交付
//CreateNewAccountAsync RPC调用（或适用的错误）。
type FutureCreateNewAccountResult chan *response

//receive等待将来承诺的响应并返回
//创建新帐户的结果。
func (r FutureCreateNewAccountResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

//CreateNewAccountAsync返回一个类型的实例，该类型可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅CreateNewAccount。
func (c *Client) CreateNewAccountAsync(account string) FutureCreateNewAccountResult {
	cmd := btcjson.NewCreateNewAccountCmd(account)
	return c.sendCmd(cmd)
}

//创建新帐户创建新的钱包帐户。
func (c *Client) CreateNewAccount(account string) error {
	return c.CreateNewAccountAsync(account).Receive()
}

//FutureGetNetWaddressResult是未来交付
//GetNewAddressAsync RPC调用（或适用的错误）。
type FutureGetNewAddressResult chan *response

//receive等待将来承诺的响应并返回新的
//地址。
func (r FutureGetNewAddressResult) Receive() (btcutil.Address, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为字符串。
	var addr string
	err = json.Unmarshal(res, &addr)
	if err != nil {
		return nil, err
	}

	return btcutil.DecodeAddress(addr, &chaincfg.MainNetParams)
}

//GetNewAddressAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅getnewaddress。
func (c *Client) GetNewAddressAsync(account string) FutureGetNewAddressResult {
	cmd := btcjson.NewGetNewAddressCmd(&account)
	return c.sendCmd(cmd)
}

//GetNewAddress返回新地址。
func (c *Client) GetNewAddress(account string) (btcutil.Address, error) {
	return c.GetNewAddressAsync(account).Receive()
}

//FutureGetrawChangeAddressResult是未来交付
//GetRawChangeAddressAsync RPC调用（或适用的错误）。
type FutureGetRawChangeAddressResult chan *response

//receive等待将来承诺的响应并返回新的
//接收将与提供的
//帐户。请注意，这只适用于原始事务，不适用于正常使用。
func (r FutureGetRawChangeAddressResult) Receive() (btcutil.Address, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为字符串。
	var addr string
	err = json.Unmarshal(res, &addr)
	if err != nil {
		return nil, err
	}

	return btcutil.DecodeAddress(addr, &chaincfg.MainNetParams)
}

//GetRawChangeAddressAsync返回可用于
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅getrawchangeaddress。
func (c *Client) GetRawChangeAddressAsync(account string) FutureGetRawChangeAddressResult {
	cmd := btcjson.NewGetRawChangeAddressCmd(&account)
	return c.sendCmd(cmd)
}

//GetRawChangeAddress返回用于接收将
//与提供的帐户关联。注意这只适用于生的
//交易，不能正常使用。
func (c *Client) GetRawChangeAddress(account string) (btcutil.Address, error) {
	return c.GetRawChangeAddressAsync(account).Receive()
}

//FutureADWITNESSAddressResult是未来交付
//AddWitnessAddressAsync RPC调用（或适用的错误）。
type FutureAddWitnessAddressResult chan *response

//receive等待未来承诺的响应并返回新的
//地址。
func (r FutureAddWitnessAddressResult) Receive() (btcutil.Address, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为字符串。
	var addr string
	err = json.Unmarshal(res, &addr)
	if err != nil {
		return nil, err
	}

	return btcutil.DecodeAddress(addr, &chaincfg.MainNetParams)
}

//AddWitnessAddressAsync返回可用于获取
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅AddWitnessAddress。
func (c *Client) AddWitnessAddressAsync(address string) FutureAddWitnessAddressResult {
	cmd := btcjson.NewAddWitnessAddressCmd(address)
	return c.sendCmd(cmd)
}

//addwitnessAddress为脚本添加见证地址并返回新的
//地址（证人脚本的p2sh）。
func (c *Client) AddWitnessAddress(address string) (btcutil.Address, error) {
	return c.AddWitnessAddressAsync(address).Receive()
}

//futuregetacountaddressresult是未来交付
//GetAccountAddressAsync RPC调用（或适用的错误）。
type FutureGetAccountAddressResult chan *response

//receive等待将来承诺的响应并返回当前
//用于接收指定帐户付款的比特币地址。
func (r FutureGetAccountAddressResult) Receive() (btcutil.Address, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为字符串。
	var addr string
	err = json.Unmarshal(res, &addr)
	if err != nil {
		return nil, err
	}

	return btcutil.DecodeAddress(addr, &chaincfg.MainNetParams)
}

//GetAccountAddressAsync返回可用于获取
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅getaccountaddress。
func (c *Client) GetAccountAddressAsync(account string) FutureGetAccountAddressResult {
	cmd := btcjson.NewGetAccountAddressCmd(account)
	return c.sendCmd(cmd)
}

//GetAccountAddress返回接收付款的当前比特币地址
//到指定的帐户。
func (c *Client) GetAccountAddress(account string) (btcutil.Address, error) {
	return c.GetAccountAddressAsync(account).Receive()
}

//futuregetacountresult是未来交付结果的承诺
//GetAccountAsync RPC调用（或适用的错误）。
type FutureGetAccountResult chan *response

//receive等待将来承诺的响应并返回帐户
//与传递的地址关联。
func (r FutureGetAccountResult) Receive() (string, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return "", err
	}

//将结果取消标记为字符串。
	var account string
	err = json.Unmarshal(res, &account)
	if err != nil {
		return "", err
	}

	return account, nil
}

//GetAccountAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅getaccount。
func (c *Client) GetAccountAsync(address btcutil.Address) FutureGetAccountResult {
	addr := address.EncodeAddress()
	cmd := btcjson.NewGetAccountCmd(addr)
	return c.sendCmd(cmd)
}

//GetAccount返回与传递的地址关联的帐户。
func (c *Client) GetAccount(address btcutil.Address) (string, error) {
	return c.GetAccountAsync(address).Receive()
}

//FutureStateCountResult是未来交付结果的承诺
//setAccountAsync RPC调用（或适用的错误）。
type FutureSetAccountResult chan *response

//receive等待将来承诺的响应并返回结果
//将帐户设置为与传递的地址关联。
func (r FutureSetAccountResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

//setAccountAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅setaccount。
func (c *Client) SetAccountAsync(address btcutil.Address, account string) FutureSetAccountResult {
	addr := address.EncodeAddress()
	cmd := btcjson.NewSetAccountCmd(addr, account)
	return c.sendCmd(cmd)
}

//setaccount设置与传递的地址关联的帐户。
func (c *Client) SetAccount(address btcutil.Address, account string) error {
	return c.SetAccountAsync(address, account).Receive()
}

//FutureGetAddResessByAccountResult是未来交付结果的承诺
//GetAddressByAccountAsync RPC调用（或适用的错误）。
type FutureGetAddressesByAccountResult chan *response

//receive等待将来承诺的响应并返回
//与传递的帐户关联的地址。
func (r FutureGetAddressesByAccountResult) Receive() ([]btcutil.Address, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消显示为字符串数组。
	var addrStrings []string
	err = json.Unmarshal(res, &addrStrings)
	if err != nil {
		return nil, err
	}

	addrs := make([]btcutil.Address, 0, len(addrStrings))
	for _, addrStr := range addrStrings {
		addr, err := btcutil.DecodeAddress(addrStr,
			&chaincfg.MainNetParams)
		if err != nil {
			return nil, err
		}
		addrs = append(addrs, addr)
	}

	return addrs, nil
}

//GetAddressByAccountAsync返回可用于
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅GetAddressByAccount。
func (c *Client) GetAddressesByAccountAsync(account string) FutureGetAddressesByAccountResult {
	cmd := btcjson.NewGetAddressesByAccountCmd(account)
	return c.sendCmd(cmd)
}

//GetAddressByAccount返回与
//通过帐户。
func (c *Client) GetAddressesByAccount(account string) ([]btcutil.Address, error) {
	return c.GetAddressesByAccountAsync(account).Receive()
}

//FutureModelResult是未来交付moveAsync结果的承诺，
//moveMinConfAsync或moveCommentAsync RPC调用（或适用的
//错误）。
type FutureMoveResult chan *response

//receive等待将来承诺的响应并返回结果
//移动操作。
func (r FutureMoveResult) Receive() (bool, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return false, err
	}

//将结果取消标记为布尔值。
	var moveResult bool
	err = json.Unmarshal(res, &moveResult)
	if err != nil {
		return false, err
	}

	return moveResult, nil
}

//moveasync返回可用于获取以下结果的类型的实例
//通过调用返回的
//实例。
//
//有关阻止版本和更多详细信息，请参见移动。
func (c *Client) MoveAsync(fromAccount, toAccount string, amount btcutil.Amount) FutureMoveResult {
	cmd := btcjson.NewMoveCmd(fromAccount, toAccount, amount.ToBTC(), nil,
		nil)
	return c.sendCmd(cmd)
}

//移动将指定金额从钱包中的一个帐户移动到另一个帐户。只有
//将使用默认最低确认数的资金。
//
//有关不同的选项，请参见moveminconf和movecomment。
func (c *Client) Move(fromAccount, toAccount string, amount btcutil.Amount) (bool, error) {
	return c.MoveAsync(fromAccount, toAccount, amount).Receive()
}

//moveMinConfAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅moveminconf。
func (c *Client) MoveMinConfAsync(fromAccount, toAccount string,
	amount btcutil.Amount, minConfirms int) FutureMoveResult {

	cmd := btcjson.NewMoveCmd(fromAccount, toAccount, amount.ToBTC(),
		&minConfirms, nil)
	return c.sendCmd(cmd)
}

//moveminconf将指定金额从钱包中的一个帐户移动到
//另一个。只有通过最低确认数的资金将
//使用。
//
//请参阅移动以使用默认的最小确认数和movecomment
//其他选项。
func (c *Client) MoveMinConf(fromAccount, toAccount string, amount btcutil.Amount, minConf int) (bool, error) {
	return c.MoveMinConfAsync(fromAccount, toAccount, amount, minConf).Receive()
}

//moveCommentAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关块版本和更多详细信息，请参见movecomment。
func (c *Client) MoveCommentAsync(fromAccount, toAccount string,
	amount btcutil.Amount, minConfirms int, comment string) FutureMoveResult {

	cmd := btcjson.NewMoveCmd(fromAccount, toAccount, amount.ToBTC(),
		&minConfirms, &comment)
	return c.sendCmd(cmd)
}

//movecomment将指定金额从钱包中的一个帐户移动到
//另一个则将提供的评论存储在钱包中。评论
//参数仅在钱包中可用。只有通过号码的基金
//将使用最少的确认。
//
//请参见move和moveminconf以使用默认值。
func (c *Client) MoveComment(fromAccount, toAccount string, amount btcutil.Amount,
	minConf int, comment string) (bool, error) {

	return c.MoveCommentAsync(fromAccount, toAccount, amount, minConf,
		comment).Receive()
}

//FutureRenameAccountResult是未来交付
//RenameAccountAsync RPC调用（或适用的错误）。
type FutureRenameAccountResult chan *response

//receive等待将来承诺的响应并返回
//创建新帐户的结果。
func (r FutureRenameAccountResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

//RenameAccountAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅RenameAccount。
func (c *Client) RenameAccountAsync(oldAccount, newAccount string) FutureRenameAccountResult {
	cmd := btcjson.NewRenameAccountCmd(oldAccount, newAccount)
	return c.sendCmd(cmd)
}

//重命名帐户创建新的钱包帐户。
func (c *Client) RenameAccount(oldAccount, newAccount string) error {
	return c.RenameAccountAsync(oldAccount, newAccount).Receive()
}

//FutureValidateAddressResult是未来交付
//validateadressasync RPC调用（或适用的错误）。
type FutureValidateAddressResult chan *response

//接收等待未来承诺的响应并返回信息
//关于比特币地址。
func (r FutureValidateAddressResult) Receive() (*btcjson.ValidateAddressWalletResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为validateAddress结果对象。
	var addrResult btcjson.ValidateAddressWalletResult
	err = json.Unmarshal(res, &addrResult)
	if err != nil {
		return nil, err
	}

	return &addrResult, nil
}

//validatedResAsync返回可用于获取
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅validateAddress。
func (c *Client) ValidateAddressAsync(address btcutil.Address) FutureValidateAddressResult {
	addr := address.EncodeAddress()
	cmd := btcjson.NewValidateAddressCmd(addr)
	return c.sendCmd(cmd)
}

//validateAddress返回有关给定比特币地址的信息。
func (c *Client) ValidateAddress(address btcutil.Address) (*btcjson.ValidateAddressWalletResult, error) {
	return c.ValidateAddressAsync(address).Receive()
}

//FutureKeyPoolRefillResult是未来交付
//keypoolrefillasync RPC调用（或适用的错误）。
type FutureKeyPoolRefillResult chan *response

//receive等待将来承诺的响应并返回结果
//补充钥匙池。
func (r FutureKeyPoolRefillResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

//keypoolrefillasync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅keypoolrefill。
func (c *Client) KeyPoolRefillAsync() FutureKeyPoolRefillResult {
	cmd := btcjson.NewKeyPoolRefillCmd(nil)
	return c.sendCmd(cmd)
}

//keypoolrefill根据需要填充密钥池以达到默认大小。
//
//请参见keypoolrefillsize以覆盖密钥池的大小。
func (c *Client) KeyPoolRefill() error {
	return c.KeyPoolRefillAsync().Receive()
}

//keypoolRefillSizeAsync返回可用于获取
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅keypoolrefillsize。
func (c *Client) KeyPoolRefillSizeAsync(newSize uint) FutureKeyPoolRefillResult {
	cmd := btcjson.NewKeyPoolRefillCmd(&newSize)
	return c.sendCmd(cmd)
}

//keypoolRefillSize根据需要填充密钥池以达到指定的
//尺寸。
func (c *Client) KeyPoolRefillSize(newSize uint) error {
	return c.KeyPoolRefillSizeAsync(newSize).Receive()
}

//*****************
//金额/余额功能
//*****************

//未来会计结果是未来交付结果的承诺
//listaccountsAsync或listaccountsMinFasync RPC调用（或
//适用错误）。
type FutureListAccountsResult chan *response

//receive等待将来承诺的响应，并返回
//帐户名及其关联余额的映射。
func (r FutureListAccountsResult) Receive() (map[string]btcutil.Amount, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为JSON对象。
	var accounts map[string]float64
	err = json.Unmarshal(res, &accounts)
	if err != nil {
		return nil, err
	}

	accountsMap := make(map[string]btcutil.Amount)
	for k, v := range accounts {
		amount, err := btcutil.NewAmount(v)
		if err != nil {
			return nil, err
		}

		accountsMap[k] = amount
	}

	return accountsMap, nil
}

//listaccountsasync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参见ListAccounts。
func (c *Client) ListAccountsAsync() FutureListAccountsResult {
	cmd := btcjson.NewListAccountsCmd(nil)
	return c.sendCmd(cmd)
}

//listaccounts返回帐户名及其关联余额的映射
//使用默认的最小确认数。
//
//请参阅listaccountsMinConf以覆盖最小确认数。
func (c *Client) ListAccounts() (map[string]btcutil.Amount, error) {
	return c.ListAccountsAsync().Receive()
}

//listaccountsmincofasync返回可用于
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅listaccountsminconf。
func (c *Client) ListAccountsMinConfAsync(minConfirms int) FutureListAccountsResult {
	cmd := btcjson.NewListAccountsCmd(&minConfirms)
	return c.sendCmd(cmd)
}

//listaccountsminconf返回帐户名及其关联的映射
//使用指定的最低确认数的余额。
//
//请参阅ListAccounts以使用默认的最小确认数。
func (c *Client) ListAccountsMinConf(minConfirms int) (map[string]btcutil.Amount, error) {
	return c.ListAccountsMinConfAsync(minConfirms).Receive()
}

//FutureGetBalanceResult是未来交付
//GetBalanceAsync或GetBalanceMinConfAsync RPC调用（或适用的
//错误）。
type FutureGetBalanceResult chan *response

//receive等待将来承诺的响应并返回
//服务器上指定帐户的可用余额。
func (r FutureGetBalanceResult) Receive() (btcutil.Amount, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return 0, err
	}

//将结果取消标记为浮点数。
	var balance float64
	err = json.Unmarshal(res, &balance)
	if err != nil {
		return 0, err
	}

	amount, err := btcutil.NewAmount(balance)
	if err != nil {
		return 0, err
	}

	return amount, nil
}

//FutureGetBalanceParseResult与FutureGetBalanceResult相同，只是
//结果应该是一个字符串，然后将其解析为
//FLAUT64值
//这是与区块链.info等服务器兼容所必需的。
type FutureGetBalanceParseResult chan *response

//receive等待将来承诺的响应并返回
//服务器上指定帐户的可用余额。
func (r FutureGetBalanceParseResult) Receive() (btcutil.Amount, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return 0, err
	}

//将结果取消标记为字符串
	var balanceString string
	err = json.Unmarshal(res, &balanceString)
	if err != nil {
		return 0, err
	}

	balance, err := strconv.ParseFloat(balanceString, 64)
	if err != nil {
		return 0, err
	}
	amount, err := btcutil.NewAmount(balance)
	if err != nil {
		return 0, err
	}

	return amount, nil
}

//GetBalanceAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻塞版本和更多详细信息，请参阅getbalance。
func (c *Client) GetBalanceAsync(account string) FutureGetBalanceResult {
	cmd := btcjson.NewGetBalanceCmd(&account, nil)
	return c.sendCmd(cmd)
}

//GetBalance返回指定服务器的可用余额
//使用默认最低确认数的帐户。帐户可能
//所有账户均为“*”。
//
//请参阅getBalanceMinConf以覆盖最小确认数。
func (c *Client) GetBalance(account string) (btcutil.Amount, error) {
	return c.GetBalanceAsync(account).Receive()
}

//GetBalanceMinConfAsync返回可用于获取
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅getbalanceminconf。
func (c *Client) GetBalanceMinConfAsync(account string, minConfirms int) FutureGetBalanceResult {
	cmd := btcjson.NewGetBalanceCmd(&account, &minConfirms)
	return c.sendCmd(cmd)
}

//GetBalanceMinConf从服务器返回
//使用指定数量的最小确认的指定帐户。这个
//所有帐户的帐户都可以是“*”。
//
//请参阅getbalance以使用默认的最小确认数。
func (c *Client) GetBalanceMinConf(account string, minConfirms int) (btcutil.Amount, error) {
	if c.config.EnableBCInfoHacks {
		response := c.GetBalanceMinConfAsync(account, minConfirms)
		return FutureGetBalanceParseResult(response).Receive()
	}
	return c.GetBalanceMinConfAsync(account, minConfirms).Receive()
}

//FutureGetTreedByAccountResult是未来交付
//GetReceiveDBYacCountAsync或GetReceiveDBYacCountMinConfAsync RPC
//调用（或适用的错误）。
type FutureGetReceivedByAccountResult chan *response

//receive等待将来承诺的响应并返回总数
//使用指定帐户收到的金额。
func (r FutureGetReceivedByAccountResult) Receive() (btcutil.Amount, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return 0, err
	}

//将结果取消标记为浮点数。
	var balance float64
	err = json.Unmarshal(res, &balance)
	if err != nil {
		return 0, err
	}

	amount, err := btcutil.NewAmount(balance)
	if err != nil {
		return 0, err
	}

	return amount, nil
}

//GetReceiveDByaCountAsync返回一个可用于
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅getreceivedbyaccount。
func (c *Client) GetReceivedByAccountAsync(account string) FutureGetReceivedByAccountResult {
	cmd := btcjson.NewGetReceivedByAccountCmd(account, nil)
	return c.sendCmd(cmd)
}

//GetReceiveDByaccount返回使用指定的
//至少具有默认最低确认数的帐户。
//
//请参阅getreceivedbyaccountminconf以重写
//确认。
func (c *Client) GetReceivedByAccount(account string) (btcutil.Amount, error) {
	return c.GetReceivedByAccountAsync(account).Receive()
}

//GetReceiveDByaccountMinConfAsync返回一个类型可以是
//用于通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅getreceivedbyaccountminconf。
func (c *Client) GetReceivedByAccountMinConfAsync(account string, minConfirms int) FutureGetReceivedByAccountResult {
	cmd := btcjson.NewGetReceivedByAccountCmd(account, &minConfirms)
	return c.sendCmd(cmd)
}

//getreceivedbyaccountminconf返回
//指定帐户至少具有指定的最小数目
//确认。
//
//请参阅getreceivedbyaccount以使用默认的最小确认数。
func (c *Client) GetReceivedByAccountMinConf(account string, minConfirms int) (btcutil.Amount, error) {
	return c.GetReceivedByAccountMinConfAsync(account, minConfirms).Receive()
}

//FutureGetUnconfirmedBalanceResult是未来交付结果的承诺
//GetUnconfirmedBalanceAsync RPC调用（或适用的错误）。
type FutureGetUnconfirmedBalanceResult chan *response

//receive等待将来承诺的响应，并返回
//来自服务器的指定帐户的未确认余额。
func (r FutureGetUnconfirmedBalanceResult) Receive() (btcutil.Amount, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return 0, err
	}

//将结果取消标记为浮点数。
	var balance float64
	err = json.Unmarshal(res, &balance)
	if err != nil {
		return 0, err
	}

	amount, err := btcutil.NewAmount(balance)
	if err != nil {
		return 0, err
	}

	return amount, nil
}

//GetUnconfirmedBalanceAsync返回可用于
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅getunconfirmedbalance。
func (c *Client) GetUnconfirmedBalanceAsync(account string) FutureGetUnconfirmedBalanceResult {
	cmd := btcjson.NewGetUnconfirmedBalanceCmd(&account)
	return c.sendCmd(cmd)
}

//GetUnconfirmedBalance从服务器返回的未确认余额
//指定的帐户。
func (c *Client) GetUnconfirmedBalance(account string) (btcutil.Amount, error) {
	return c.GetUnconfirmedBalanceAsync(account).Receive()
}

//FutureGetTreeCeivedByAddressResult是未来交付
//GetReceiveDBYAddressSync或GetReceiveDBYAddressMinConfAsync RPC
//调用（或适用的错误）。
type FutureGetReceivedByAddressResult chan *response

//receive等待将来承诺的响应并返回总数
//按指定地址接收的金额。
func (r FutureGetReceivedByAddressResult) Receive() (btcutil.Amount, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return 0, err
	}

//将结果取消标记为浮点数。
	var balance float64
	err = json.Unmarshal(res, &balance)
	if err != nil {
		return 0, err
	}

	amount, err := btcutil.NewAmount(balance)
	if err != nil {
		return 0, err
	}

	return amount, nil
}

//GetReceiveDByaddressSync返回可用于
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅getreceivedbyaddress。
func (c *Client) GetReceivedByAddressAsync(address btcutil.Address) FutureGetReceivedByAddressResult {
	addr := address.EncodeAddress()
	cmd := btcjson.NewGetReceivedByAddressCmd(addr, nil)
	return c.sendCmd(cmd)

}

//GetReceiveDBYAddress返回由指定的
//地址至少包含默认的最小确认数。
//
//请参阅getreceivedbyaddressminconf以覆盖
//确认。
func (c *Client) GetReceivedByAddress(address btcutil.Address) (btcutil.Amount, error) {
	return c.GetReceivedByAddressAsync(address).Receive()
}

//GetReceiveDBYAddressMinConfAsync返回一个类型的实例，该类型可以是
//用于通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅getreceivedbyaddressminconf。
func (c *Client) GetReceivedByAddressMinConfAsync(address btcutil.Address, minConfirms int) FutureGetReceivedByAddressResult {
	addr := address.EncodeAddress()
	cmd := btcjson.NewGetReceivedByAddressCmd(addr, &minConfirms)
	return c.sendCmd(cmd)
}

//getreceivedbyaddressminconf返回指定的
//至少具有指定数量的最小确认的地址。
//
//请参阅getreceivedbyaddress以使用默认的最小确认数。
func (c *Client) GetReceivedByAddressMinConf(address btcutil.Address, minConfirms int) (btcutil.Amount, error) {
	return c.GetReceivedByAddressMinConfAsync(address, minConfirms).Receive()
}

//未来交付由accountresult接收是未来交付结果的承诺
//ListReceiveDBYacCountAsync、ListReceiveDBYacCountMinConfAsync或
//listReceiveDByaCountincludeEmptyAsync RPC调用（或适用的
//错误）。
type FutureListReceivedByAccountResult chan *response

//Receive等待将来承诺的响应，并返回
//账户余额。
func (r FutureListReceivedByAccountResult) Receive() ([]btcjson.ListReceivedByAccountResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//取消标记为ListReceiveDByaccount结果对象的数组。
	var received []btcjson.ListReceivedByAccountResult
	err = json.Unmarshal(res, &received)
	if err != nil {
		return nil, err
	}

	return received, nil
}

//ListReceiveDByaCountAsync返回一个可用于
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅ListReceiveDByaccount。
func (c *Client) ListReceivedByAccountAsync() FutureListReceivedByAccountResult {
	cmd := btcjson.NewListReceivedByAccountCmd(nil, nil, nil)
	return c.sendCmd(cmd)
}

//lisreceivedbyaccount使用默认数字按帐户列出余额
//最低确认金额，包括未收到任何
//付款。
//
//请参见ListReceiveDByaccountMinConf以重写
//Confirmations和ListReceiveDByaCountincludeEmpty用于筛选
//没有收到任何来自结果的付款。
func (c *Client) ListReceivedByAccount() ([]btcjson.ListReceivedByAccountResult, error) {
	return c.ListReceivedByAccountAsync().Receive()
}

//ListReceiveDByaccountMinConfAsync返回一个类型的实例，该类型可以是
//用于通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅ListReceiveDByaccountMinConf。
func (c *Client) ListReceivedByAccountMinConfAsync(minConfirms int) FutureListReceivedByAccountResult {
	cmd := btcjson.NewListReceivedByAccountCmd(&minConfirms, nil, nil)
	return c.sendCmd(cmd)
}

//listreceivedbyaccountminconf使用指定的
//最低确认数，不包括未收到的账户
//任何付款。
//
//请参阅ListReceiveDByaccount以使用默认的最小确认数
//和lisreceivedbyaccountincludeEmpty还包括没有
//在结果中收到任何付款。
func (c *Client) ListReceivedByAccountMinConf(minConfirms int) ([]btcjson.ListReceivedByAccountResult, error) {
	return c.ListReceivedByAccountMinConfAsync(minConfirms).Receive()
}

//ListReceiveDByaCountincludeEmptyAsync返回一个类型的实例，该类型可以
//通过调用
//返回的实例上的接收函数。
//
//有关阻止版本和更多详细信息，请参见ListReceiveDByaCountincludeEmpty。
func (c *Client) ListReceivedByAccountIncludeEmptyAsync(minConfirms int, includeEmpty bool) FutureListReceivedByAccountResult {
	cmd := btcjson.NewListReceivedByAccountCmd(&minConfirms, &includeEmpty,
		nil)
	return c.sendCmd(cmd)
}

//listReceiveDByaCountincludeEmpty按帐户列出余额，使用
//指定的最低确认数，包括
//根据指定标志，尚未收到任何付款。
//
//请参见ListReceiveDByaccount和ListReceiveDByaccountMinConf以使用默认值。
func (c *Client) ListReceivedByAccountIncludeEmpty(minConfirms int, includeEmpty bool) ([]btcjson.ListReceivedByAccountResult, error) {
	return c.ListReceivedByAccountIncludeEmptyAsync(minConfirms,
		includeEmpty).Receive()
}

//FutureElistreceedByAddressResult是未来交付结果的承诺
//ListReceivedByAddressSync、ListReceivedByAddressMinConfAsync或
//listReceiveDBYAddressincludeEmptyAsync RPC调用（或适用的
//错误）。
type FutureListReceivedByAddressResult chan *response

//Receive等待将来承诺的响应，并返回
//按地址列出的余额。
func (r FutureListReceivedByAddressResult) Receive() ([]btcjson.ListReceivedByAddressResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//取消标记为ListReceivedByAddress结果对象的数组。
	var received []btcjson.ListReceivedByAddressResult
	err = json.Unmarshal(res, &received)
	if err != nil {
		return nil, err
	}

	return received, nil
}

//ListReceiveDByaddressSync返回一个可用于
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参见ListReceiveDByaAddress。
func (c *Client) ListReceivedByAddressAsync() FutureListReceivedByAddressResult {
	cmd := btcjson.NewListReceivedByAddressCmd(nil, nil, nil)
	return c.sendCmd(cmd)
}

//lisreceivedbyaddress使用默认数字按地址列出余额
//最低确认数，不包括未收到的地址
//付款或只看地址。
//
//请参见ListReceiveDByaddressMinConf以覆盖
//确认和列表接收到的地址也包括地址
//结果中没有收到任何付款。
func (c *Client) ListReceivedByAddress() ([]btcjson.ListReceivedByAddressResult, error) {
	return c.ListReceivedByAddressAsync().Receive()
}

//ListReceiveDBYAddressMinConfAsync返回一个类型的实例，该类型可以是
//用于通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅ListReceiveDByaddressMinConf。
func (c *Client) ListReceivedByAddressMinConfAsync(minConfirms int) FutureListReceivedByAddressResult {
	cmd := btcjson.NewListReceivedByAddressCmd(&minConfirms, nil, nil)
	return c.sendCmd(cmd)
}

//lisreceivedbyaddressminconf使用指定的
//最低确认数，不包括尚未收到的地址
//任何付款。
//
//请参阅listReceiveDBYAddress以使用默认的最小确认数
//和ListReceivedByAddressIncludeEmpty还包括地址没有
//在结果中收到任何付款。
func (c *Client) ListReceivedByAddressMinConf(minConfirms int) ([]btcjson.ListReceivedByAddressResult, error) {
	return c.ListReceivedByAddressMinConfAsync(minConfirms).Receive()
}

//ListReceivedByAddressIncludeEmptyAsync返回一个类型的实例，该类型可以
//通过调用
//返回的实例上的接收函数。
//
//有关阻止版本和更多详细信息，请参见ListReceiveDByaCountincludeEmpty。
func (c *Client) ListReceivedByAddressIncludeEmptyAsync(minConfirms int, includeEmpty bool) FutureListReceivedByAddressResult {
	cmd := btcjson.NewListReceivedByAddressCmd(&minConfirms, &includeEmpty,
		nil)
	return c.sendCmd(cmd)
}

//listreceivedbyaddressincludeEmpty按地址列出余额，使用
//指定的最低确认数，包括
//根据指定标志，尚未收到任何付款。
//
//请参见lisreceivedbyaddress和lisreceivedbyaddressminconf以使用默认值。
func (c *Client) ListReceivedByAddressIncludeEmpty(minConfirms int, includeEmpty bool) ([]btcjson.ListReceivedByAddressResult, error) {
	return c.ListReceivedByAddressIncludeEmptyAsync(minConfirms,
		includeEmpty).Receive()
}

//*****************
//钱包锁定功能
//*****************

//FutureWalletLockResult是未来交付
//WalletLockAsync RPC调用（或适用的错误）。
type FutureWalletLockResult chan *response

//receive等待将来承诺的响应并返回结果
//把钱包锁上。
func (r FutureWalletLockResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

//WalletLockAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅walletlock。
func (c *Client) WalletLockAsync() FutureWalletLockResult {
	cmd := btcjson.NewWalletLockCmd()
	return c.sendCmd(cmd)
}

//WalletLock通过从内存中取出加密密钥来锁定钱包。
//
//调用此函数后，必须使用walletpassphrase函数
//在调用任何其他需要
//钱包要解锁。
func (c *Client) WalletLock() error {
	return c.WalletLockAsync().Receive()
}

//wallet passphrase通过使用passphrase派生
//解密密钥，然后在指定的超时时间存储在内存中
//（以秒为单位）
func (c *Client) WalletPassphrase(passphrase string, timeoutSecs int64) error {
	cmd := btcjson.NewWalletPassphraseCmd(passphrase, timeoutSecs)
	_, err := c.sendCmdAndWait(cmd)
	return err
}

//未来AlletPassPhraseChangeResult是未来交付结果的承诺
//一个walletpassphrasechangeasync RPC调用（或一个适用的错误）。
type FutureWalletPassphraseChangeResult chan *response

//receive等待将来承诺的响应并返回结果
//更改钱包密码。
func (r FutureWalletPassphraseChangeResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

//walletpassphrasechangeasync返回可用于
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅walletpassphrasechange。
func (c *Client) WalletPassphraseChangeAsync(old, new string) FutureWalletPassphraseChangeResult {
	cmd := btcjson.NewWalletPassphraseChangeCmd(old, new)
	return c.sendCmd(cmd)
}

//walletpassphrasechange将钱包密码从指定的旧密码更改为
//新密码。
func (c *Client) WalletPassphraseChange(old, new string) error {
	return c.WalletPassphraseChangeAsync(old, new).Receive()
}

//**********************
//消息签名功能
//**********************

//FutureDesignMessageResult是未来交付
//SignMessageAsync RPC调用（或适用的错误）。
type FutureSignMessageResult chan *response

//receive等待将来承诺的响应并返回消息
//用指定地址的私钥签名。
func (r FutureSignMessageResult) Receive() (string, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return "", err
	}

//将结果取消标记为字符串。
	var b64 string
	err = json.Unmarshal(res, &b64)
	if err != nil {
		return "", err
	}

	return b64, nil
}

//SignMessageAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅signmessage。
func (c *Client) SignMessageAsync(address btcutil.Address, message string) FutureSignMessageResult {
	addr := address.EncodeAddress()
	cmd := btcjson.NewSignMessageCmd(addr, message)
	return c.sendCmd(cmd)
}

//signmessage使用指定地址的私钥对消息进行签名。
//
//注意：此功能要求钱包解锁。见
//有关详细信息，请参阅walletpassphrase函数。
func (c *Client) SignMessage(address btcutil.Address, message string) (string, error) {
	return c.SignMessageAsync(address, message).Receive()
}

//未来每一个信息结果都是未来实现
//VerifyMessageAsync RPC调用（或适用的错误）。
type FutureVerifyMessageResult chan *response

//
//未成功验证邮件。
func (r FutureVerifyMessageResult) Receive() (bool, error) {
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

//VerifyMessageAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅verifymessage。
func (c *Client) VerifyMessageAsync(address btcutil.Address, signature, message string) FutureVerifyMessageResult {
	addr := address.EncodeAddress()
	cmd := btcjson.NewVerifyMessageCmd(addr, signature, message)
	return c.sendCmd(cmd)
}

//verifymessage验证已签名的消息。
//
//注意：此功能要求钱包解锁。见
//有关详细信息，请参阅walletpassphrase函数。
func (c *Client) VerifyMessage(address btcutil.Address, signature, message string) (bool, error) {
	return c.VerifyMessageAsync(address, signature, message).Receive()
}

//*****************
//转储/导入功能
//*****************

//FutureDumpprivKeyResult是未来交付
//dumpprivkeyasync RPC调用（或适用的错误）。
type FutureDumpPrivKeyResult chan *response

//receive等待将来承诺的响应并返回private
//与以钱包导入格式编码的传递地址对应的密钥
//（WIF）
func (r FutureDumpPrivKeyResult) Receive() (*btcutil.WIF, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为字符串。
	var privKeyWIF string
	err = json.Unmarshal(res, &privKeyWIF)
	if err != nil {
		return nil, err
	}

	return btcutil.DecodeWIF(privKeyWIF)
}

//dumpprivkeyanc返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅dumpprivkey。
func (c *Client) DumpPrivKeyAsync(address btcutil.Address) FutureDumpPrivKeyResult {
	addr := address.EncodeAddress()
	cmd := btcjson.NewDumpPrivKeyCmd(addr)
	return c.sendCmd(cmd)
}

//dumpprivkey获取与编码的传递地址对应的私钥
//钱包导入格式（WIF）。
//
//注意：此功能要求钱包解锁。见
//有关详细信息，请参阅walletpassphrase函数。
func (c *Client) DumpPrivKey(address btcutil.Address) (*btcutil.WIF, error) {
	return c.DumpPrivKeyAsync(address).Receive()
}

//FutureImportAddressResult是未来交付
//importAddressAsync RPC调用（或适用的错误）。
type FutureImportAddressResult chan *response

//receive等待将来承诺的响应并返回结果
//导入已传递的公共地址。
func (r FutureImportAddressResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

//importAddressAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅importaddress。
func (c *Client) ImportAddressAsync(address string) FutureImportAddressResult {
	cmd := btcjson.NewImportAddressCmd(address, "", nil)
	return c.sendCmd(cmd)
}

//importAddress导入传递的公共地址。
func (c *Client) ImportAddress(address string) error {
	return c.ImportAddressAsync(address).Receive()
}

//importAddressRescanAsync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅importaddress。
func (c *Client) ImportAddressRescanAsync(address string, account string, rescan bool) FutureImportAddressResult {
	cmd := btcjson.NewImportAddressCmd(address, account, &rescan)
	return c.sendCmd(cmd)
}

//importAddressRescan导入传递的公共地址。当Rescan为真时，
//扫描块历史记录以查找发往所提供地址的事务。
func (c *Client) ImportAddressRescan(address string, account string, rescan bool) error {
	return c.ImportAddressRescanAsync(address, account, rescan).Receive()
}

//FutureImportPrivKeyResult是未来交付
//importprivkeyasync RPC调用（或适用的错误）。
type FutureImportPrivKeyResult chan *response

//receive等待将来承诺的响应并返回结果
//导入已传递的私钥，该私钥必须是钱包导入格式
//（WIF）。
func (r FutureImportPrivKeyResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

//importprivkeyasync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅importprivkey。
func (c *Client) ImportPrivKeyAsync(privKeyWIF *btcutil.WIF) FutureImportPrivKeyResult {
	wif := ""
	if privKeyWIF != nil {
		wif = privKeyWIF.String()
	}

	cmd := btcjson.NewImportPrivKeyCmd(wif, nil, nil)
	return c.sendCmd(cmd)
}

//importprivkey导入传递的私钥，必须是钱包导入
//格式（WIF）。
func (c *Client) ImportPrivKey(privKeyWIF *btcutil.WIF) error {
	return c.ImportPrivKeyAsync(privKeyWIF).Receive()
}

//importprivkeylabelasync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅importprivkey。
func (c *Client) ImportPrivKeyLabelAsync(privKeyWIF *btcutil.WIF, label string) FutureImportPrivKeyResult {
	wif := ""
	if privKeyWIF != nil {
		wif = privKeyWIF.String()
	}

	cmd := btcjson.NewImportPrivKeyCmd(wif, &label, nil)
	return c.sendCmd(cmd)
}

//importprivkeylabel导入传递的私钥，该私钥必须是钱包导入
//格式（WIF）。它将帐户标签设置为提供的标签。
func (c *Client) ImportPrivKeyLabel(privKeyWIF *btcutil.WIF, label string) error {
	return c.ImportPrivKeyLabelAsync(privKeyWIF, label).Receive()
}

//importprivkeyrescanasync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅importprivkey。
func (c *Client) ImportPrivKeyRescanAsync(privKeyWIF *btcutil.WIF, label string, rescan bool) FutureImportPrivKeyResult {
	wif := ""
	if privKeyWIF != nil {
		wif = privKeyWIF.String()
	}

	cmd := btcjson.NewImportPrivKeyCmd(wif, &label, &rescan)
	return c.sendCmd(cmd)
}

//importprivkeyrescan导入传递的私钥，该私钥必须是钱包导入
//格式（WIF）。它将帐户标签设置为提供的标签。当Rescan为真时，
//扫描块历史记录以查找发往提供的私钥的事务。
func (c *Client) ImportPrivKeyRescan(privKeyWIF *btcutil.WIF, label string, rescan bool) error {
	return c.ImportPrivKeyRescanAsync(privKeyWIF, label, rescan).Receive()
}

//FutureImportPubKeyResult是未来交付
//importPubKeyAsync RPC调用（或适用的错误）。
type FutureImportPubKeyResult chan *response

//receive等待将来承诺的响应并返回结果
//导入传递的公钥。
func (r FutureImportPubKeyResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

//importpubkeyasync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅importpubkey。
func (c *Client) ImportPubKeyAsync(pubKey string) FutureImportPubKeyResult {
	cmd := btcjson.NewImportPubKeyCmd(pubKey, nil)
	return c.sendCmd(cmd)
}

//importpubkey导入传递的公钥。
func (c *Client) ImportPubKey(pubKey string) error {
	return c.ImportPubKeyAsync(pubKey).Receive()
}

//importpubkeyrescanasync返回可用于获取
//在将来某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅importpubkey。
func (c *Client) ImportPubKeyRescanAsync(pubKey string, rescan bool) FutureImportPubKeyResult {
	cmd := btcjson.NewImportPubKeyCmd(pubKey, &rescan)
	return c.sendCmd(cmd)
}

//importpubkeyrescan导入传递的公钥。如果Rescan为真，则
//将扫描块历史记录以查找发往提供的pubkey的事务。
func (c *Client) ImportPubKeyRescan(pubKey string, rescan bool) error {
	return c.ImportPubKeyRescanAsync(pubKey, rescan).Receive()
}

//*****************
//其他函数
//*****************

//注意：当getinfo在这里实现时（在wallet.go中），一个btcd链服务器
//也将响应getinfo请求，不包括任何钱包信息。

//未来预测是未来交付
//GetInfoAsync RPC调用（或适用的错误）。
type FutureGetInfoResult chan *response

//receive等待将来承诺的响应并返回信息
//由服务器提供。
func (r FutureGetInfoResult) Receive() (*btcjson.InfoWalletResult, error) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

//将结果取消标记为GetInfo结果对象。
	var infoRes btcjson.InfoWalletResult
	err = json.Unmarshal(res, &infoRes)
	if err != nil {
		return nil, err
	}

	return &infoRes, nil
}

//GetInfoAsync返回可用于获取结果的类型的实例
//在将来的某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅getinfo。
func (c *Client) GetInfoAsync() FutureGetInfoResult {
	cmd := btcjson.NewGetInfoCmd()
	return c.sendCmd(cmd)
}

//GetInfo返回有关RPC服务器的其他信息。归还的人
//如果远程服务器执行此操作，则信息对象可能无效。
//不包括钱包功能。
func (c *Client) GetInfo() (*btcjson.InfoWalletResult, error) {
	return c.GetInfoAsync().Receive()
}

//TODO（Davec）：实现
//备份钱包（nyi in btcwallet）
//EncryptWallet（因为它总是加密的，所以btcwallet不支持）
//getwalletinfo（nyi在btcwallet或btcjson中）
//列表地址分组（btcwallet中的nyi）
//ListReceiveDByaccount（btcwallet中的nyi）

//转储
//importwallet（nyi在btcwallet中）
//dumpwallet（NYI在btcwallet）
