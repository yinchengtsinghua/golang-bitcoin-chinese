
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
	"errors"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

var (
//errWebSocketsRequired是描述
//调用方正在尝试使用仅WebSocket的功能，例如请求
//当客户端为
//配置为在HTTP POST模式下运行。
	ErrWebsocketsRequired = errors.New("a websocket connection is required " +
		"to use this feature")
)

//NotificationState用于跟踪的当前状态
//registered notification so the state can be automatically re-established on
//重新连接。
type notificationState struct {
	notifyBlocks       bool
	notifyNewTx        bool
	notifyNewTxVerbose bool
	notifyReceived     map[string]struct{}
	notifySpent        map[btcjson.OutPoint]struct{}
}

//copy返回接收器的深度副本。
func (s *notificationState) Copy() *notificationState {
	var stateCopy notificationState
	stateCopy.notifyBlocks = s.notifyBlocks
	stateCopy.notifyNewTx = s.notifyNewTx
	stateCopy.notifyNewTxVerbose = s.notifyNewTxVerbose
	stateCopy.notifyReceived = make(map[string]struct{})
	for addr := range s.notifyReceived {
		stateCopy.notifyReceived[addr] = struct{}{}
	}
	stateCopy.notifySpent = make(map[btcjson.OutPoint]struct{})
	for op := range s.notifySpent {
		stateCopy.notifySpent[op] = struct{}{}
	}

	return &stateCopy
}

//NewNotificationState返回准备填充的新通知状态。
func newNotificationState() *notificationState {
	return &notificationState{
		notifyReceived: make(map[string]struct{}),
		notifySpent:    make(map[btcjson.OutPoint]struct{}),
	}
}

//NewNilFutureResult返回一个新的未来结果通道，该通道已经具有
//结果等待回复设置为零的通道。这是有用的
//当调用方未指定任何
//通知处理程序。
func newNilFutureResult() chan *response {
	responseChan := make(chan *response, 1)
	responseChan <- &response{result: nil, err: nil}
	return responseChan
}

//notificationhandlers定义要调用的回调函数指针
//通知。由于默认情况下所有函数都为零，因此
//通知被有效忽略，直到它们的处理程序设置为
//具体回调。
//
//NOTE: Unless otherwise documented, these handlers must NOT directly call any
//由于输入读取器goroutine阻塞，因此阻止对客户端实例的调用
//直到回调完成。这样做会导致死锁
//情况。
type NotificationHandlers struct {
//当客户端连接或重新连接时调用OnClientConnected
//到RPC服务器。此回调与
//通知处理程序，并且对于阻止客户端请求是安全的。
	OnClientConnected func()

//当块连接到最长的
//（最好）链。只有在前面调用
//已设置notifyBlocks以注册通知和
//函数非零。
//
//注意：已弃用。请改用onfilteredblockconnected。
	OnBlockConnected func(hash *chainhash.Hash, height int32, t time.Time)

//当块连接到
//最长（最好）的链条。只有在前面调用
//已设置notifyBlocks以注册通知和
//函数非零。它的参数不同于onblockconnected:it
//接收块的高度、标题和相关事务。
	OnFilteredBlockConnected func(height int32, header *wire.BlockHeader,
		txs []*btcutil.Tx)

//当块与
//最长（最好）的链条。只有在前面调用
//已设置notifyBlocks以注册通知和
//函数非零。
//
//注意：已弃用。改用onfilteredblockdisconnected。
	OnBlockDisconnected func(hash *chainhash.Hash, height int32, t time.Time)

//断开块连接时调用OnFilteredBlockDisconnected
//从最长（最好）的链条。只有在
//已进行前面的通知块以注册通知
//对函数的调用是非零的。其参数不同于
//OnBlockDisconnected：接收块的高度和标题。
	OnFilteredBlockDisconnected func(height int32, header *wire.BlockHeader)

//当一个交易接收资金给
//注册地址被接收到内存池中，并且
//连接最长（最好）的链条。只有在
//对notifyReceived、rescan或rescanEndHeight的前一个调用
//用于注册通知，函数不为零。
//
//注意：已弃用。请改用onrelevantttxaccepted。
	OnRecvTx func(transaction *btcutil.Tx, details *btcjson.BlockDetails)

//当花费已注册的
//输出点被接收到内存池并连接到
//最长（最好）的链条。只有在前面调用
//已进行notifySpended、rescan或rescanEndHeight以注册
//通知和函数不为零。
//
//注意：接收到的通知将自动注册通知
//对于现在因接收而“拥有”的输出点
//资金到注册地址。这意味着
//这将作为notifyReceived调用的结果间接调用。
//
//注意：已弃用。请改用onrelevantttxaccepted。
	OnRedeemingTx func(transaction *btcutil.Tx, details *btcjson.BlockDetails)

//当未链接的事务通过时调用OnRelevantTxaccepted
//the client's transaction filter.
//
//注意：这是从中导入的BTCSuite扩展
//GITHUB/COMPUD/DCRRPCEclipse。
	OnRelevantTxAccepted func(transaction []byte)

//在RESCAN由于先前的一个完成之后调用OnReSCANFETCH。
//调用以重新扫描或重新扫描高度。完成的重新扫描应
//signaled on this notification, rather than relying on the return
//由于BTCD可能发送各种重新扫描，重新扫描请求的结果
//重新扫描请求后的通知已返回。
//
//注意：已弃用。不用于重新扫描块。
	OnRescanFinished func(hash *chainhash.Hash, height int32, blkTime time.Time)

//当重新扫描正在进行时，会定期调用OnRescanProgress。
//只有在前面调用重新扫描或
//重新扫描高度已设置，功能为非零。
//
//注意：已弃用。不用于重新扫描块。
	OnRescanProgress func(hash *chainhash.Hash, height int32, blkTime time.Time)

//当事务被接受到
//内存池。只有在调用之前调用
//已将verbose标志设置为false的notifyNewTransactions
//用于注册通知，函数不为零。
	OnTxAccepted func(hash *chainhash.Hash, amount btcutil.Amount)

//当事务被接受到
//内存池。只有在调用之前调用
//verbose标志设置为true的notifyNewTransactions已
//用于注册通知，函数不为零。
	OnTxAcceptedVerbose func(txDetails *btcjson.TxRawResult)

//当钱包连接或断开与
//BTCD。
//
//只有当客户机连接到钱包时，此功能才可用
//服务器，如btcwallet。
	OnBtcdConnected func(connected bool)

//通过帐户余额更新调用OnAccountBalance。
//
//只有在与钱包服务器通话时，此功能才可用
//such as btcwallet.
	OnAccountBalance func(account string, balance btcutil.Amount, confirmed bool)

//当钱包被锁定或解锁时，调用OnWalletLockState。
//
//只有当客户机连接到钱包时，此功能才可用
//服务器，如btcwallet。
	OnWalletLockState func(locked bool)

//当无法识别的通知时调用onUnknownNotification
//收到。这通常意味着通知处理代码
//for this package needs to be updated for a new notification type or
//调用方正在使用此包不知道的自定义通知
//关于。
	OnUnknownNotification func(method string, params []json.RawMessage)
}

//handlenotification检查传递的通知类型，执行
//将原始通知类型转换为更高级别的类型和
//将通知传递给注册的相应on<x>处理程序
//客户。
func (c *Client) handleNotification(ntfn *rawNotification) {
//如果客户端对任何
//通知。
	if c.ntfnHandlers == nil {
		return
	}

	switch ntfn.Method {
//闭锁连接
	case btcjson.BlockConnectedNtfnMethod:
//Ignore the notification if the client is not interested in
//它。
		if c.ntfnHandlers.OnBlockConnected == nil {
			return
		}

		blockHash, blockHeight, blockTime, err := parseChainNtfnParams(ntfn.Params)
		if err != nil {
			log.Warnf("Received invalid block connected "+
				"notification: %v", err)
			return
		}

		c.ntfnHandlers.OnBlockConnected(blockHash, blockHeight, blockTime)

//已连接OnFilteredBlock
	case btcjson.FilteredBlockConnectedNtfnMethod:
//如果客户端不感兴趣，则忽略通知
//它。
		if c.ntfnHandlers.OnFilteredBlockConnected == nil {
			return
		}

		blockHeight, blockHeader, transactions, err :=
			parseFilteredBlockConnectedParams(ntfn.Params)
		if err != nil {
			log.Warnf("Received invalid filtered block "+
				"connected notification: %v", err)
			return
		}

		c.ntfnHandlers.OnFilteredBlockConnected(blockHeight,
			blockHeader, transactions)

//OnBlock已断开连接
	case btcjson.BlockDisconnectedNtfnMethod:
//如果客户端不感兴趣，则忽略通知
//它。
		if c.ntfnHandlers.OnBlockDisconnected == nil {
			return
		}

		blockHash, blockHeight, blockTime, err := parseChainNtfnParams(ntfn.Params)
		if err != nil {
			log.Warnf("Received invalid block connected "+
				"notification: %v", err)
			return
		}

		c.ntfnHandlers.OnBlockDisconnected(blockHash, blockHeight, blockTime)

//OnFilteredBlock已断开连接
	case btcjson.FilteredBlockDisconnectedNtfnMethod:
//如果客户端不感兴趣，则忽略通知
//它。
		if c.ntfnHandlers.OnFilteredBlockDisconnected == nil {
			return
		}

		blockHeight, blockHeader, err :=
			parseFilteredBlockDisconnectedParams(ntfn.Params)
		if err != nil {
			log.Warnf("Received invalid filtered block "+
				"disconnected notification: %v", err)
			return
		}

		c.ntfnHandlers.OnFilteredBlockDisconnected(blockHeight,
			blockHeader)

//OnReCVTX
	case btcjson.RecvTxNtfnMethod:
//如果客户端不感兴趣，则忽略通知
//它。
		if c.ntfnHandlers.OnRecvTx == nil {
			return
		}

		tx, block, err := parseChainTxNtfnParams(ntfn.Params)
		if err != nil {
			log.Warnf("Received invalid recvtx notification: %v",
				err)
			return
		}

		c.ntfnHandlers.OnRecvTx(tx, block)

//赎回票据
	case btcjson.RedeemingTxNtfnMethod:
//如果客户端不感兴趣，则忽略通知
//它。
		if c.ntfnHandlers.OnRedeemingTx == nil {
			return
		}

		tx, block, err := parseChainTxNtfnParams(ntfn.Params)
		if err != nil {
			log.Warnf("Received invalid redeemingtx "+
				"notification: %v", err)
			return
		}

		c.ntfnHandlers.OnRedeemingTx(tx, block)

//接受相关的ttxaccepted
	case btcjson.RelevantTxAcceptedNtfnMethod:
//如果客户端不感兴趣，则忽略通知
//它。
		if c.ntfnHandlers.OnRelevantTxAccepted == nil {
			return
		}

		transaction, err := parseRelevantTxAcceptedParams(ntfn.Params)
		if err != nil {
			log.Warnf("Received invalid relevanttxaccepted "+
				"notification: %v", err)
			return
		}

		c.ntfnHandlers.OnRelevantTxAccepted(transaction)

//OnRescanFinished（重新扫描完成）
	case btcjson.RescanFinishedNtfnMethod:
//如果客户端不感兴趣，则忽略通知
//它。
		if c.ntfnHandlers.OnRescanFinished == nil {
			return
		}

		hash, height, blkTime, err := parseRescanProgressParams(ntfn.Params)
		if err != nil {
			log.Warnf("Received invalid rescanfinished "+
				"notification: %v", err)
			return
		}

		c.ntfnHandlers.OnRescanFinished(hash, height, blkTime)

//重新扫描进度
	case btcjson.RescanProgressNtfnMethod:
//如果客户端不感兴趣，则忽略通知
//它。
		if c.ntfnHandlers.OnRescanProgress == nil {
			return
		}

		hash, height, blkTime, err := parseRescanProgressParams(ntfn.Params)
		if err != nil {
			log.Warnf("Received invalid rescanprogress "+
				"notification: %v", err)
			return
		}

		c.ntfnHandlers.OnRescanProgress(hash, height, blkTime)

//Onthx接受的
	case btcjson.TxAcceptedNtfnMethod:
//如果客户端不感兴趣，则忽略通知
//它。
		if c.ntfnHandlers.OnTxAccepted == nil {
			return
		}

		hash, amt, err := parseTxAcceptedNtfnParams(ntfn.Params)
		if err != nil {
			log.Warnf("Received invalid tx accepted "+
				"notification: %v", err)
			return
		}

		c.ntfnHandlers.OnTxAccepted(hash, amt)

//ontxAcceptedVerbose（接受详细信息）
	case btcjson.TxAcceptedVerboseNtfnMethod:
//如果客户端不感兴趣，则忽略通知
//它。
		if c.ntfnHandlers.OnTxAcceptedVerbose == nil {
			return
		}

		rawTx, err := parseTxAcceptedVerboseNtfnParams(ntfn.Params)
		if err != nil {
			log.Warnf("Received invalid tx accepted verbose "+
				"notification: %v", err)
			return
		}

		c.ntfnHandlers.OnTxAcceptedVerbose(rawTx)

//ONBTCD-连接的
	case btcjson.BtcdConnectedNtfnMethod:
//如果客户端不感兴趣，则忽略通知
//它。
		if c.ntfnHandlers.OnBtcdConnected == nil {
			return
		}

		connected, err := parseBtcdConnectedNtfnParams(ntfn.Params)
		if err != nil {
			log.Warnf("Received invalid btcd connected "+
				"notification: %v", err)
			return
		}

		c.ntfnHandlers.OnBtcdConnected(connected)

//应付帐款余额
	case btcjson.AccountBalanceNtfnMethod:
//如果客户端不感兴趣，则忽略通知
//它。
		if c.ntfnHandlers.OnAccountBalance == nil {
			return
		}

		account, bal, conf, err := parseAccountBalanceNtfnParams(ntfn.Params)
		if err != nil {
			log.Warnf("Received invalid account balance "+
				"notification: %v", err)
			return
		}

		c.ntfnHandlers.OnAccountBalance(account, bal, conf)

//在WalletLockState上
	case btcjson.WalletLockStateNtfnMethod:
//如果客户端不感兴趣，则忽略通知
//它。
		if c.ntfnHandlers.OnWalletLockState == nil {
			return
		}

//The account name is not notified, so the return value is
//丢弃的。
		_, locked, err := parseWalletLockStateNtfnParams(ntfn.Params)
		if err != nil {
			log.Warnf("Received invalid wallet lock state "+
				"notification: %v", err)
			return
		}

		c.ntfnHandlers.OnWalletLockState(locked)

//通知
	default:
		if c.ntfnHandlers.OnUnknownNotification == nil {
			return
		}

		c.ntfnHandlers.OnUnknownNotification(ntfn.Method, ntfn.Params)
	}
}

//wronNumParams是描述不可解析JSON-RPC的错误类型
//由于的参数数目不正确而导致的通知
//需要通知类型。该值是参数个数。
//无效通知的。
type wrongNumParams int

//错误满足内置错误接口。
func (e wrongNumParams) Error() string {
	return fmt.Sprintf("wrong number of parameters (%d)", e)
}

//parsechainntfnparams根据参数解析块散列和高度
//块连接和块断开连接的通知。
func parseChainNtfnParams(params []json.RawMessage) (*chainhash.Hash,
	int32, time.Time, error) {

	if len(params) != 3 {
		return nil, 0, time.Time{}, wrongNumParams(len(params))
	}

//将第一个参数取消标记为字符串。
	var blockHashStr string
	err := json.Unmarshal(params[0], &blockHashStr)
	if err != nil {
		return nil, 0, time.Time{}, err
	}

//将第二个参数解封为整数。
	var blockHeight int32
	err = json.Unmarshal(params[1], &blockHeight)
	if err != nil {
		return nil, 0, time.Time{}, err
	}

//将第三个参数取消标记为Unix时间。
	var blockTimeUnix int64
	err = json.Unmarshal(params[2], &blockTimeUnix)
	if err != nil {
		return nil, 0, time.Time{}, err
	}

//从块哈希字符串创建哈希。
	blockHash, err := chainhash.NewHashFromStr(blockHashStr)
	if err != nil {
		return nil, 0, time.Time{}, err
	}

//创建时间。从Unix时间开始。
	blockTime := time.Unix(blockTimeUnix, 0)

	return blockHash, blockHeight, blockTime, nil
}

//ParseFilteredBlockConnectedParams解析包含在
//filteredblockconnected notification.
//
//注意：这是从github.com/decred/dcrrpcclient移植的BTCD扩展
//需要WebSocket连接。
func parseFilteredBlockConnectedParams(params []json.RawMessage) (int32,
	*wire.BlockHeader, []*btcutil.Tx, error) {

	if len(params) < 3 {
		return 0, nil, nil, wrongNumParams(len(params))
	}

//将第一个参数取消标记为整数。
	var blockHeight int32
	err := json.Unmarshal(params[0], &blockHeight)
	if err != nil {
		return 0, nil, nil, err
	}

//将第二个参数取消标记为字节片。
	blockHeaderBytes, err := parseHexParam(params[1])
	if err != nil {
		return 0, nil, nil, err
	}

//从字节切片反序列化块头。
	var blockHeader wire.BlockHeader
	err = blockHeader.Deserialize(bytes.NewReader(blockHeaderBytes))
	if err != nil {
		return 0, nil, nil, err
	}

//解压缩第三参数作为十六进制编码字符串的一个切片。
	var hexTransactions []string
	err = json.Unmarshal(params[2], &hexTransactions)
	if err != nil {
		return 0, nil, nil, err
	}

//通过十六进制解码从字符串切片创建事务切片。
	transactions := make([]*btcutil.Tx, len(hexTransactions))
	for i, hexTx := range hexTransactions {
		transaction, err := hex.DecodeString(hexTx)
		if err != nil {
			return 0, nil, nil, err
		}

		transactions[i], err = btcutil.NewTxFromBytes(transaction)
		if err != nil {
			return 0, nil, nil, err
		}
	}

	return blockHeight, &blockHeader, transactions, nil
}

//ParseFilteredBlockDisconnectedParams解析包含在
//filteredblockdisconnected通知。
//
//注意：这是从github.com/decred/dcrrpcclient移植的BTCD扩展
//需要WebSocket连接。
func parseFilteredBlockDisconnectedParams(params []json.RawMessage) (int32,
	*wire.BlockHeader, error) {
	if len(params) < 2 {
		return 0, nil, wrongNumParams(len(params))
	}

//将第一个参数取消标记为整数。
	var blockHeight int32
	err := json.Unmarshal(params[0], &blockHeight)
	if err != nil {
		return 0, nil, err
	}

//Unmarshal second parmeter as a slice of bytes.
	blockHeaderBytes, err := parseHexParam(params[1])
	if err != nil {
		return 0, nil, err
	}

//从字节切片反序列化块头。
	var blockHeader wire.BlockHeader
	err = blockHeader.Deserialize(bytes.NewReader(blockHeaderBytes))
	if err != nil {
		return 0, nil, err
	}

	return blockHeight, &blockHeader, nil
}

func parseHexParam(param json.RawMessage) ([]byte, error) {
	var s string
	err := json.Unmarshal(param, &s)
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(s)
}

//parseRelevantTxacceptedParams解析包含在
//已接受相关通知。
func parseRelevantTxAcceptedParams(params []json.RawMessage) (transaction []byte, err error) {
	if len(params) < 1 {
		return nil, wrongNumParams(len(params))
	}

	return parseHexParam(params[0])
}

//parsechaintxtfnparams解析事务和有关的可选详细信息
//从recvtx和redeemingtx的参数中挖掘的块
//通知。
func parseChainTxNtfnParams(params []json.RawMessage) (*btcutil.Tx,
	*btcjson.BlockDetails, error) {

	if len(params) == 0 || len(params) > 2 {
		return nil, nil, wrongNumParams(len(params))
	}

//将第一个参数取消标记为字符串。
	var txHex string
	err := json.Unmarshal(params[0], &txHex)
	if err != nil {
		return nil, nil, err
	}

//如果存在，将第二个可选参数取消标记为块详细信息
//JSON对象。
	var block *btcjson.BlockDetails
	if len(params) > 1 {
		err = json.Unmarshal(params[1], &block)
		if err != nil {
			return nil, nil, err
		}
	}

//十六进制解码和反序列化事务。
	serializedTx, err := hex.DecodeString(txHex)
	if err != nil {
		return nil, nil, err
	}
	var msgTx wire.MsgTx
	err = msgTx.Deserialize(bytes.NewReader(serializedTx))
	if err != nil {
		return nil, nil, err
	}

//TODO:更改recvtx和redeemingtx回调签名以使用
//关于块的详细信息的更好类型（块哈希）
//chainhash.hash、块时间作为时间、时间等）。
	return btcutil.NewTx(&msgTx), block, nil
}

//ParseRescanProgressParams解析出上次重新扫描的块的高度
//来自RescanFinished和RescanProgress通知的参数。
func parseRescanProgressParams(params []json.RawMessage) (*chainhash.Hash, int32, time.Time, error) {
	if len(params) != 3 {
		return nil, 0, time.Time{}, wrongNumParams(len(params))
	}

//将第一个参数取消标记为字符串。
	var hashStr string
	err := json.Unmarshal(params[0], &hashStr)
	if err != nil {
		return nil, 0, time.Time{}, err
	}

//将第二个参数解封为整数。
	var height int32
	err = json.Unmarshal(params[1], &height)
	if err != nil {
		return nil, 0, time.Time{}, err
	}

//将第三个参数取消标记为整数。
	var blkTime int64
	err = json.Unmarshal(params[2], &blkTime)
	if err != nil {
		return nil, 0, time.Time{}, err
	}

//解码块哈希的字符串编码。
	hash, err := chainhash.NewHashFromStr(hashStr)
	if err != nil {
		return nil, 0, time.Time{}, err
	}

	return hash, height, time.Unix(blkTime, 0), nil
}

//parseTxAcceptedNtfnParams解析事务哈希和总金额
//来自txaccepted通知的参数。
func parseTxAcceptedNtfnParams(params []json.RawMessage) (*chainhash.Hash,
	btcutil.Amount, error) {

	if len(params) != 2 {
		return nil, 0, wrongNumParams(len(params))
	}

//将第一个参数取消标记为字符串。
	var txHashStr string
	err := json.Unmarshal(params[0], &txHashStr)
	if err != nil {
		return nil, 0, err
	}

//将第二个参数取消标记为浮点数。
	var famt float64
	err = json.Unmarshal(params[1], &famt)
	if err != nil {
		return nil, 0, err
	}

//Bounds check amount.
	amt, err := btcutil.NewAmount(famt)
	if err != nil {
		return nil, 0, err
	}

//解码事务sha的字符串编码。
	txHash, err := chainhash.NewHashFromStr(txHashStr)
	if err != nil {
		return nil, 0, err
	}

	return txHash, amt, nil
}

//parseTxAcceptedVerbosentFnParams解析有关原始事务的详细信息
//来自txAcceptedVerbose通知的参数。
func parseTxAcceptedVerboseNtfnParams(params []json.RawMessage) (*btcjson.TxRawResult,
	error) {

	if len(params) != 1 {
		return nil, wrongNumParams(len(params))
	}

//将第一个参数取消标记为原始事务结果对象。
	var rawTx btcjson.TxRawResult
	err := json.Unmarshal(params[0], &rawTx)
	if err != nil {
		return nil, err
	}

//TODO:将txAcceptedVerbose通知回调更改为使用nicer
//有关事务的所有详细信息的类型（即解码哈希
//从它们的字符串编码）。
	return &rawTx, nil
}

//parsebtcconnectedntfnparams解析出btcd的连接状态
//和btcwallet来自btcconnected通知的参数。
func parseBtcdConnectedNtfnParams(params []json.RawMessage) (bool, error) {
	if len(params) != 1 {
		return false, wrongNumParams(len(params))
	}

//将第一个参数取消标记为布尔值。
	var connected bool
	err := json.Unmarshal(params[0], &connected)
	if err != nil {
		return false, err
	}

	return connected, nil
}

//parseAccountBalanceFnParams解析出帐户名、总余额，
//and whether or not the balance is confirmed or unconfirmed from the
//帐户余额通知的参数。
func parseAccountBalanceNtfnParams(params []json.RawMessage) (account string,
	balance btcutil.Amount, confirmed bool, err error) {

	if len(params) != 3 {
		return "", 0, false, wrongNumParams(len(params))
	}

//将第一个参数取消标记为字符串。
	err = json.Unmarshal(params[0], &account)
	if err != nil {
		return "", 0, false, err
	}

//将第二个参数取消标记为浮点数。
	var fbal float64
	err = json.Unmarshal(params[1], &fbal)
	if err != nil {
		return "", 0, false, err
	}

//将第三个参数取消标记为布尔值。
	err = json.Unmarshal(params[2], &confirmed)
	if err != nil {
		return "", 0, false, err
	}

//边界检查金额。
	bal, err := btcutil.NewAmount(fbal)
	if err != nil {
		return "", 0, false, err
	}

	return account, bal, confirmed, nil
}

//parsewalletlockstatentfnparams解析出帐户名并锁定
//来自walletlockstate通知参数的帐户状态。
func parseWalletLockStateNtfnParams(params []json.RawMessage) (account string,
	locked bool, err error) {

	if len(params) != 2 {
		return "", false, wrongNumParams(len(params))
	}

//将第一个参数取消标记为字符串。
	err = json.Unmarshal(params[0], &account)
	if err != nil {
		return "", false, err
	}

//将第二个参数取消标记为布尔值。
	err = json.Unmarshal(params[1], &locked)
	if err != nil {
		return "", false, err
	}

	return account, locked, nil
}

//FutureNotifyBlocksResult是未来交付
//NotifyBlocksAsync RPC调用（或适用的错误）。
type FutureNotifyBlocksResult chan *response

//receive等待将来承诺的响应并返回错误
//如果注册不成功。
func (r FutureNotifyBlocksResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

//notifyblocksasync返回可用于获取
//通过调用上的接收函数，在将来某个时间的RPC结果
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅通知块。
//
//注意：这是BTCD扩展，需要WebSocket连接。
func (c *Client) NotifyBlocksAsync() FutureNotifyBlocksResult {
//HTTP POST模式不支持。
	if c.config.HTTPPostMode {
		return newFutureError(ErrWebsocketsRequired)
	}

//如果客户端不感兴趣，则忽略通知
//通知。
	if c.ntfnHandlers == nil {
		return newNilFutureResult()
	}

	cmd := btcjson.NewNotifyBlocksCmd()
	return c.sendCmd(cmd)
}

//NotifyBlocks registers the client to receive notifications when blocks are
//连接并断开主链。通知是
//传递到与客户端关联的通知处理程序。打电话
//如果没有通知处理程序，则此函数无效，并且将
//如果客户机配置为在HTTP POST模式下运行，则会导致错误。
//
//由于此呼叫而传递的通知将通过
//OnBlockConnected或OnBlockDisconnected。
//
//注意：这是BTCD扩展，需要WebSocket连接。
func (c *Client) NotifyBlocks() error {
	return c.NotifyBlocksAsync().Receive()
}

//FutureNotifyPendResult是未来交付结果的承诺
//NotifySpentasync RPC调用（或适用的错误）。
//
//注意：已弃用。请改用FutureLoadTxFilterResult。
type FutureNotifySpentResult chan *response

//receive等待将来承诺的响应并返回错误
//如果注册不成功。
func (r FutureNotifySpentResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

//notifySpendInternal与notifySpendAsync相同，只是它接受
//转换后的输出点作为参数，以便客户机能够更高效地
//在重新连接上重新创建以前的通知状态。
func (c *Client) notifySpentInternal(outpoints []btcjson.OutPoint) FutureNotifySpentResult {
//HTTP POST模式不支持。
	if c.config.HTTPPostMode {
		return newFutureError(ErrWebsocketsRequired)
	}

//如果客户端不感兴趣，则忽略通知
//通知。
	if c.ntfnHandlers == nil {
		return newNilFutureResult()
	}

	cmd := btcjson.NewNotifySpentCmd(outpoints)
	return c.sendCmd(cmd)
}

//newoutpointfromwire构造事务的btcjson表示
//线类型的输出点。
func newOutPointFromWire(op *wire.OutPoint) btcjson.OutPoint {
	return btcjson.OutPoint{
		Hash:  op.Hash.String(),
		Index: op.Index,
	}
}

//notifySpnetAsync返回一个类型的实例，该实例可用于获取
//通过调用上的接收函数，在将来某个时间的RPC结果
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅notifyspeed。
//
//注意：这是BTCD扩展，需要WebSocket连接。
//
//注意：已弃用。改用loadtxfilterasync。
func (c *Client) NotifySpentAsync(outpoints []*wire.OutPoint) FutureNotifySpentResult {
//HTTP POST模式不支持。
	if c.config.HTTPPostMode {
		return newFutureError(ErrWebsocketsRequired)
	}

//如果客户端不感兴趣，则忽略通知
//通知。
	if c.ntfnHandlers == nil {
		return newNilFutureResult()
	}

	ops := make([]btcjson.OutPoint, 0, len(outpoints))
	for _, outpoint := range outpoints {
		ops = append(ops, newOutPointFromWire(outpoint))
	}
	cmd := btcjson.NewNotifySpentCmd(ops)
	return c.sendCmd(cmd)
}

//当通过时，notifyspeed注册客户端以接收通知
//事务输出被占用。通知将传递到
//与客户端关联的通知处理程序。调用此函数时
//如果没有通知处理程序并将导致错误，则无效果
//如果客户机配置为在HTTP POST模式下运行。
//
//由于此呼叫而传递的通知将通过
//OnRedeemingTx。
//
//注意：这是BTCD扩展，需要WebSocket连接。
//
//注意：已弃用。改用loadtxfilter。
func (c *Client) NotifySpent(outpoints []*wire.OutPoint) error {
	return c.NotifySpentAsync(outpoints).Receive()
}

//未来的创新是未来交付成果的承诺。
//notifyNewTransactionsAsync RPC调用（或适用的错误）。
type FutureNotifyNewTransactionsResult chan *response

//receive等待将来承诺的响应并返回错误
//如果注册不成功。
func (r FutureNotifyNewTransactionsResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

//notifyNewTransactionsAsync返回可用于
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻止版本和更多详细信息，请参阅notifynewtransactionsasync。
//
//注意：这是BTCD扩展，需要WebSocket连接。
func (c *Client) NotifyNewTransactionsAsync(verbose bool) FutureNotifyNewTransactionsResult {
//HTTP POST模式不支持。
	if c.config.HTTPPostMode {
		return newFutureError(ErrWebsocketsRequired)
	}

//如果客户端不感兴趣，则忽略通知
//通知。
	if c.ntfnHandlers == nil {
		return newNilFutureResult()
	}

	cmd := btcjson.NewNotifyNewTransactionsCmd(&verbose)
	return c.sendCmd(cmd)
}

//notifyNewTransactions注册客户端以接收通知
//接受新事务到内存池的时间。通知是
//传递到与客户端关联的通知处理程序。打电话
//如果没有通知处理程序，则此函数无效，并且将
//如果客户机配置为在HTTP POST模式下运行，则会导致错误。
//
//由于此呼叫而传递的通知将通过
//ontxaccepted（verbose为false时）或ontxaccepted verbose（verbose为
//真的）。
//
//注意：这是BTCD扩展，需要WebSocket连接。
func (c *Client) NotifyNewTransactions(verbose bool) error {
	return c.NotifyNewTransactionsAsync(verbose).Receive()
}

//未来接收结果是未来交付结果的承诺
//notifyReceivedAsync RPC调用（或适用的错误）。
//
//注意：已弃用。请改用FutureLoadTxFilterResult。
type FutureNotifyReceivedResult chan *response

//receive等待将来承诺的响应并返回错误
//如果注册不成功。
func (r FutureNotifyReceivedResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

//notifyReceivedInternal与notifyReceivedAsync相同，但它接受
//将转换后的地址作为参数，以便客户机能够更高效地
//在重新连接上重新创建以前的通知状态。
func (c *Client) notifyReceivedInternal(addresses []string) FutureNotifyReceivedResult {
//HTTP POST模式不支持。
	if c.config.HTTPPostMode {
		return newFutureError(ErrWebsocketsRequired)
	}

//如果客户端不感兴趣，则忽略通知
//通知。
	if c.ntfnHandlers == nil {
		return newNilFutureResult()
	}

//将地址转换为字符串。
	cmd := btcjson.NewNotifyReceivedCmd(addresses)
	return c.sendCmd(cmd)
}

//notifyReceivedAsync返回可用于获取
//通过调用上的接收函数，在将来某个时间的RPC结果
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅notifyreceived。
//
//注意：这是BTCD扩展，需要WebSocket连接。
//
//注意：已弃用。改用loadtxfilterasync。
func (c *Client) NotifyReceivedAsync(addresses []btcutil.Address) FutureNotifyReceivedResult {
//HTTP POST模式不支持。
	if c.config.HTTPPostMode {
		return newFutureError(ErrWebsocketsRequired)
	}

//如果客户端不感兴趣，则忽略通知
//通知。
	if c.ntfnHandlers == nil {
		return newNilFutureResult()
	}

//将地址转换为字符串。
	addrs := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		addrs = append(addrs, addr.String())
	}
	cmd := btcjson.NewNotifyReceivedCmd(addrs)
	return c.sendCmd(cmd)
}

//notifyReceived每次注册客户端以接收通知
//支付到某个已传递地址的新交易被接受到
//内存池或连接到块链的块中。此外，当
//检测到其中一个事务，客户端也会自动
//在新事务输出地址时注册以接收通知
//现在可用的已花费（请参阅notifyspended）。通知是
//传递到与客户端关联的通知处理程序。打电话
//如果没有通知处理程序，则此函数无效，并且将
//如果客户机配置为在HTTP POST模式下运行，则会导致错误。
//
//由于此呼叫而传递的通知将通过
//*OnRecvtx（对于接收到某个已通过的
//地址）或OnRedeemingtx（用于从一个
//在收到资金后自动注册的输出点
//地址）。
//
//注意：这是BTCD扩展，需要WebSocket连接。
//
//注意：已弃用。改用loadtxfilter。
func (c *Client) NotifyReceived(addresses []btcutil.Address) error {
	return c.NotifyReceivedAsync(addresses).Receive()
}

//FutureRescanResult是未来提供重新同步结果的承诺
//或重新扫描高度异步RPC调用（或适用的错误）。
//
//注意：已弃用。请改用FutureRescanBlocksResult。
type FutureRescanResult chan *response

//receive等待将来承诺的响应并返回错误
//如果重新扫描失败。
func (r FutureRescanResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

//rescanasync返回可用于获取结果的类型的实例
//在将来的某个时间通过调用
//返回实例。
//
//有关阻止版本和更多详细信息，请参阅重新扫描。
//
//注意：重新连接客户端时不发出重新扫描请求，必须
//手动执行（理想情况下，新的开始高度基于最后一个
//重新扫描进度通知）。请参阅onclientConnected通知
//回调一个好的调用站点，以便在连接和重新发出重新扫描请求
//重新连接。
//
//注意：这是BTCD扩展，需要WebSocket连接。
//
//注意：已弃用。请改用rescanblocksasync。
func (c *Client) RescanAsync(startBlock *chainhash.Hash,
	addresses []btcutil.Address,
	outpoints []*wire.OutPoint) FutureRescanResult {

//HTTP POST模式不支持。
	if c.config.HTTPPostMode {
		return newFutureError(ErrWebsocketsRequired)
	}

//如果客户端不感兴趣，则忽略通知
//通知。
	if c.ntfnHandlers == nil {
		return newNilFutureResult()
	}

//将块哈希转换为字符串。
	var startBlockHashStr string
	if startBlock != nil {
		startBlockHashStr = startBlock.String()
	}

//将地址转换为字符串。
	addrs := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		addrs = append(addrs, addr.String())
	}

//转换输出点。
	ops := make([]btcjson.OutPoint, 0, len(outpoints))
	for _, op := range outpoints {
		ops = append(ops, newOutPointFromWire(op))
	}

	cmd := btcjson.NewRescanCmd(startBlockHashStr, addrs, ops, nil)
	return c.sendCmd(cmd)
}

//重新扫描将块链从提供的起始块重新扫描到
//支付给通过的交易的最长链的结尾
//使用传递的输出点的地址和事务。
//
//找到的事务的通知将传递到通知
//与客户端和此调用关联的处理程序在
//重新扫描已完成。如果没有
//通知处理程序，如果配置了客户端，将导致错误
//以HTTP POST模式运行。
//
//由于此呼叫而传递的通知将通过
//OnRedeemingtx（用于从
//通过的输出点），onrecvtx（用于接收资金的交易
//到其中一个已传递的地址）和onRescanProgress（用于重新扫描进度）
//更新。
//
//请参见RescanEndBlock以指定结束块以完成重新扫描。
//without continuing through the best block on the main chain.
//
//注意：重新连接客户端时不发出重新扫描请求，必须
//手动执行（理想情况下，新的开始高度基于最后一个
//重新扫描进度通知）。请参阅onclientConnected通知
//回调一个好的调用站点，以便在连接和重新发出重新扫描请求
//重新连接。
//
//注意：这是BTCD扩展，需要WebSocket连接。
//
//注意：已弃用。改用RescanBlocks。
func (c *Client) Rescan(startBlock *chainhash.Hash,
	addresses []btcutil.Address,
	outpoints []*wire.OutPoint) error {

	return c.RescanAsync(startBlock, addresses, outpoints).Receive()
}

//rescanendblockasync返回可用于获取的类型的实例
//通过调用上的接收函数，在将来某个时候得到RPC的结果。
//返回的实例。
//
//有关阻止版本和更多详细信息，请参阅rescanendblock。
//
//注意：这是BTCD扩展，需要WebSocket连接。
//
//注意：已弃用。请改用rescanblocksasync。
func (c *Client) RescanEndBlockAsync(startBlock *chainhash.Hash,
	addresses []btcutil.Address, outpoints []*wire.OutPoint,
	endBlock *chainhash.Hash) FutureRescanResult {

//HTTP POST模式不支持。
	if c.config.HTTPPostMode {
		return newFutureError(ErrWebsocketsRequired)
	}

//如果客户端不感兴趣，则忽略通知
//通知。
	if c.ntfnHandlers == nil {
		return newNilFutureResult()
	}

//将块哈希转换为字符串。
	var startBlockHashStr, endBlockHashStr string
	if startBlock != nil {
		startBlockHashStr = startBlock.String()
	}
	if endBlock != nil {
		endBlockHashStr = endBlock.String()
	}

//将地址转换为字符串。
	addrs := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		addrs = append(addrs, addr.String())
	}

//转换输出点。
	ops := make([]btcjson.OutPoint, 0, len(outpoints))
	for _, op := range outpoints {
		ops = append(ops, newOutPointFromWire(op))
	}

	cmd := btcjson.NewRescanCmd(startBlockHashStr, addrs, ops,
		&endBlockHashStr)
	return c.sendCmd(cmd)
}

//重新扫描高度从提供的开始重新扫描区块链
//对于支付给
//传递的地址和使用传递的输出点的事务。
//
//找到的事务的通知将传递到通知
//与客户端和此调用关联的处理程序在
//重新扫描已完成。如果没有
//通知处理程序，如果配置了客户端，将导致错误
//以HTTP POST模式运行。
//
//由于此呼叫而传递的通知将通过
//OnRedeemingtx（用于从
//通过的输出点），onrecvtx（用于接收资金的交易
//到其中一个已传递的地址）和onRescanProgress（用于重新扫描进度）
//更新。
//
//请参见“重新扫描”以通过最长链的当前端执行重新扫描。
//
//注意：这是BTCD扩展，需要WebSocket连接。
//
//注意：已弃用。改用RescanBlocks。
func (c *Client) RescanEndHeight(startBlock *chainhash.Hash,
	addresses []btcutil.Address, outpoints []*wire.OutPoint,
	endBlock *chainhash.Hash) error {

	return c.RescanEndBlockAsync(startBlock, addresses, outpoints,
		endBlock).Receive()
}

//FutureLoadTxFilterResult是未来交付结果的承诺
//loadtxfilterasync RPC调用（或适用的错误）。
//
//注意：这是从github.com/decred/dcrrpcclient移植的BTCD扩展
//需要WebSocket连接。
type FutureLoadTxFilterResult chan *response

//receive等待将来承诺的响应并返回错误
//如果注册不成功。
//
//注意：这是从github.com/decred/dcrrpcclient移植的BTCD扩展
//需要WebSocket连接。
func (r FutureLoadTxFilterResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

//loadtxfilterasync返回可用于
//通过调用接收在将来某个时间获取RPC的结果
//函数。
//
//有关阻塞版本和更多详细信息，请参阅loadtxfilter。
//
//注意：这是从github.com/decred/dcrrpcclient移植的BTCD扩展
//需要WebSocket连接。
func (c *Client) LoadTxFilterAsync(reload bool, addresses []btcutil.Address,
	outPoints []wire.OutPoint) FutureLoadTxFilterResult {

	addrStrs := make([]string, len(addresses))
	for i, a := range addresses {
		addrStrs[i] = a.EncodeAddress()
	}
	outPointObjects := make([]btcjson.OutPoint, len(outPoints))
	for i := range outPoints {
		outPointObjects[i] = btcjson.OutPoint{
			Hash:  outPoints[i].Hash.String(),
			Index: outPoints[i].Index,
		}
	}

	cmd := btcjson.NewLoadTxFilterCmd(reload, addrStrs, outPointObjects)
	return c.sendCmd(cmd)
}

//loadtxfilter加载、重新加载或向WebSocket客户端的事务添加数据
//过滤器。根据已检查的事务，始终更新筛选器
//在mempool验收期间，块验收，以及所有重新扫描的块。
//
//注意：这是从github.com/decred/dcrrpcclient移植的BTCD扩展
//需要WebSocket连接。
func (c *Client) LoadTxFilter(reload bool, addresses []btcutil.Address, outPoints []wire.OutPoint) error {
	return c.LoadTxFilterAsync(reload, addresses, outPoints).Receive()
}
