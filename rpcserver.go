
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2017 BTCSuite开发者
//版权所有（c）2015-2017法令开发商
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package main

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/blockchain/indexers"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/database"
	"github.com/btcsuite/btcd/mempool"
	"github.com/btcsuite/btcd/mining"
	"github.com/btcsuite/btcd/mining/cpuminer"
	"github.com/btcsuite/btcd/peer"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/websocket"
)

//API版本常量
const (
	jsonrpcSemverString = "1.3.0"
	jsonrpcSemverMajor  = 1
	jsonrpcSemverMinor  = 3
	jsonrpcSemverPatch  = 0
)

const (
//rpcauthTimeoutSeconds是连接到
//允许RPC服务器保持打开状态，而不在其之前进行身份验证
//关闭。
	rpcAuthTimeoutSeconds = 10

//uint256size是表示无符号所需的字节数
//256位整数。
	uint256Size = 32

//gbtnoncernage是两个32位的big-endian十六进制整数，其中
//表示GetBlockTemplate返回的非字符的有效范围
//RPC。
	gbtNonceRange = "00000000ffffffff"

//gbtregenerateseconds是之前必须经过的秒数
//当上一个块哈希没有生成新模板时，
//已更改，并且对可用事务进行了更改
//在内存池中。
	gbtRegenerateSeconds = 60

//MaxProtocolVersion是服务器支持的最大协议版本。
	maxProtocolVersion = 70002
)

var (
//gbtmutableFields是服务器允许进行的操作
//阻止由GetBlockTemplate RPC生成的模板。它是
//在这里声明以避免在
//调用常量数据。
	gbtMutableFields = []string{
		"time", "transactions/add", "prevblock", "coinbase/append",
	}

//GBTCoinbaseaux描述了矿工应包括的其他数据
//在CoinBase签名脚本中。这里声明是为了避免
//每次调用常量时创建新对象的开销
//数据。
	gbtCoinbaseAux = &btcjson.GetBlockTemplateResultAux{
		Flags: hex.EncodeToString(builderScript(txscript.
			NewScriptBuilder().
			AddData([]byte(mining.CoinbaseFlags)))),
	}

//GBTCapabilities描述返回的附加功能
//由GetBlockTemplate RPC生成的块模板。它是
//在这里声明以避免在
//调用常量数据。
	gbtCapabilities = []string{"proposal"}
)

//错误
var (
//errrpcunimplemented是在
//提供的命令已被识别，但尚未实现。
	ErrRPCUnimplemented = &btcjson.RPCError{
		Code:    btcjson.ErrRPCUnimplemented,
		Message: "Command unimplemented",
	}

//errrpconwallet是在提供
//命令被识别为钱包命令。
	ErrRPCNoWallet = &btcjson.RPCError{
		Code:    btcjson.ErrRPCNoWallet,
		Message: "This implementation does not implement wallet commands",
	}
)

type commandHandler func(*rpcServer, interface{}, <-chan struct{}) (interface{}, error)

//rpchandlers将rpc命令字符串映射到适当的处理程序函数。
//这是由init设置的，因为帮助引用了rpchandler，从而导致
//依赖循环。
var rpcHandlers map[string]commandHandler
var rpcHandlersBeforeInit = map[string]commandHandler{
	"addnode":               handleAddNode,
	"createrawtransaction":  handleCreateRawTransaction,
	"debuglevel":            handleDebugLevel,
	"decoderawtransaction":  handleDecodeRawTransaction,
	"decodescript":          handleDecodeScript,
	"estimatefee":           handleEstimateFee,
	"generate":              handleGenerate,
	"getaddednodeinfo":      handleGetAddedNodeInfo,
	"getbestblock":          handleGetBestBlock,
	"getbestblockhash":      handleGetBestBlockHash,
	"getblock":              handleGetBlock,
	"getblockchaininfo":     handleGetBlockChainInfo,
	"getblockcount":         handleGetBlockCount,
	"getblockhash":          handleGetBlockHash,
	"getblockheader":        handleGetBlockHeader,
	"getblocktemplate":      handleGetBlockTemplate,
	"getcfilter":            handleGetCFilter,
	"getcfilterheader":      handleGetCFilterHeader,
	"getconnectioncount":    handleGetConnectionCount,
	"getcurrentnet":         handleGetCurrentNet,
	"getdifficulty":         handleGetDifficulty,
	"getgenerate":           handleGetGenerate,
	"gethashespersec":       handleGetHashesPerSec,
	"getheaders":            handleGetHeaders,
	"getinfo":               handleGetInfo,
	"getmempoolinfo":        handleGetMempoolInfo,
	"getmininginfo":         handleGetMiningInfo,
	"getnettotals":          handleGetNetTotals,
	"getnetworkhashps":      handleGetNetworkHashPS,
	"getpeerinfo":           handleGetPeerInfo,
	"getrawmempool":         handleGetRawMempool,
	"getrawtransaction":     handleGetRawTransaction,
	"gettxout":              handleGetTxOut,
	"help":                  handleHelp,
	"node":                  handleNode,
	"ping":                  handlePing,
	"searchrawtransactions": handleSearchRawTransactions,
	"sendrawtransaction":    handleSendRawTransaction,
	"setgenerate":           handleSetGenerate,
	"stop":                  handleStop,
	"submitblock":           handleSubmitBlock,
	"uptime":                handleUptime,
	"validateaddress":       handleValidateAddress,
	"verifychain":           handleVerifyChain,
	"verifymessage":         handleVerifyMessage,
	"version":               handleVersion,
}

//我们识别的命令列表，但BTCD不支持这些命令，因为
//它缺乏对钱包功能的支持。对于这些命令，用户
//应询问btcwallet的已连接实例。
var rpcAskWallet = map[string]struct{}{
	"addmultisigaddress":     {},
	"backupwallet":           {},
	"createencryptedwallet":  {},
	"createmultisig":         {},
	"dumpprivkey":            {},
	"dumpwallet":             {},
	"encryptwallet":          {},
	"getaccount":             {},
	"getaccountaddress":      {},
	"getaddressesbyaccount":  {},
	"getbalance":             {},
	"getnewaddress":          {},
	"getrawchangeaddress":    {},
	"getreceivedbyaccount":   {},
	"getreceivedbyaddress":   {},
	"gettransaction":         {},
	"gettxoutsetinfo":        {},
	"getunconfirmedbalance":  {},
	"getwalletinfo":          {},
	"importprivkey":          {},
	"importwallet":           {},
	"keypoolrefill":          {},
	"listaccounts":           {},
	"listaddressgroupings":   {},
	"listlockunspent":        {},
	"listreceivedbyaccount":  {},
	"listreceivedbyaddress":  {},
	"listsinceblock":         {},
	"listtransactions":       {},
	"listunspent":            {},
	"lockunspent":            {},
	"move":                   {},
	"sendfrom":               {},
	"sendmany":               {},
	"sendtoaddress":          {},
	"setaccount":             {},
	"settxfee":               {},
	"signmessage":            {},
	"signrawtransaction":     {},
	"walletlock":             {},
	"walletpassphrase":       {},
	"walletpassphrasechange": {},
}

//当前未实现但最终应为的命令。
var rpcUnimplemented = map[string]struct{}{
	"estimatepriority": {},
	"getchaintips":     {},
	"getmempoolentry":  {},
	"getnetworkinfo":   {},
	"getwork":          {},
	"invalidateblock":  {},
	"preciousblock":    {},
	"reconsiderblock":  {},
}

//对有限用户可用的命令
var rpcLimited = map[string]struct{}{
//WebSockets命令
	"loadtxfilter":          {},
	"notifyblocks":          {},
	"notifynewtransactions": {},
	"notifyreceived":        {},
	"notifyspent":           {},
	"rescan":                {},
	"rescanblocks":          {},
	"session":               {},

//websockets和http/s命令
	"help": {},

//仅HTTP/S命令
	"createrawtransaction":  {},
	"decoderawtransaction":  {},
	"decodescript":          {},
	"estimatefee":           {},
	"getbestblock":          {},
	"getbestblockhash":      {},
	"getblock":              {},
	"getblockcount":         {},
	"getblockhash":          {},
	"getblockheader":        {},
	"getcfilter":            {},
	"getcfilterheader":      {},
	"getcurrentnet":         {},
	"getdifficulty":         {},
	"getheaders":            {},
	"getinfo":               {},
	"getnettotals":          {},
	"getnetworkhashps":      {},
	"getrawmempool":         {},
	"getrawtransaction":     {},
	"gettxout":              {},
	"searchrawtransactions": {},
	"sendrawtransaction":    {},
	"submitblock":           {},
	"uptime":                {},
	"validateaddress":       {},
	"verifymessage":         {},
	"version":               {},
}

//builderscript是一个方便的函数，用于硬编码脚本
//使用脚本生成器生成。任何错误都会变成恐慌，因为它
//仅和必须仅与硬编码一起使用，因此，已知良好，
//脚本。
func builderScript(builder *txscript.ScriptBuilder) []byte {
	script, err := builder.Script()
	if err != nil {
		panic(err)
	}
	return script
}

//InternalRpcError是一个将内部错误转换为
//具有适当代码集的RPC错误。它还将错误记录到
//RPC服务器子系统，因为实际上不应该发生内部错误。这个
//上下文参数仅在日志消息中使用，如果是
//不需要。
func internalRPCError(errStr, context string) *btcjson.RPCError {
	logStr := errStr
	if context != "" {
		logStr = context + ": " + errStr
	}
	rpcsLog.Error(logStr)
	return btcjson.NewRPCError(btcjson.ErrRPCInternal.Code, errStr)
}

//rpcdecodehexerror是返回格式良好的
//RPC错误，指示提供的十六进制字符串解码失败。
func rpcDecodeHexError(gotHex string) *btcjson.RPCError {
	return btcjson.NewRPCError(btcjson.ErrRPCDecodeHexString,
		fmt.Sprintf("Argument must be hexadecimal string (not %q)",
			gotHex))
}

//rpcnotxInfoError是返回格式良好的
//rpc错误，表示没有提供的可用信息
//事务哈希。
func rpcNoTxInfoError(txHash *chainhash.Hash) *btcjson.RPCError {
	return btcjson.NewRPCError(btcjson.ErrRPCNoTxInfo,
		fmt.Sprintf("No information available about transaction %v",
			txHash))
}

//gNetworkState包含在对的多个RPC调用之间使用的状态
//获取块模板。
type gbtWorkState struct {
	sync.Mutex
	lastTxUpdate  time.Time
	lastGenerated time.Time
	prevHash      *chainhash.Hash
	minTimestamp  time.Time
	template      *mining.BlockTemplate
	notifyMap     map[chainhash.Hash]map[int64]chan struct{}
	timeSource    blockchain.MedianTimeSource
}

//newgNetworkState返回具有所有内部
//字段已初始化并可以使用。
func newGbtWorkState(timeSource blockchain.MedianTimeSource) *gbtWorkState {
	return &gbtWorkState{
		notifyMap:  make(map[chainhash.Hash]map[int64]chan struct{}),
		timeSource: timeSource,
	}
}

//handleunimplemented是最终应该
//支持，但尚未实现。
func handleUnimplemented(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, ErrRPCUnimplemented
}

//handleaskwallet是识别为有效命令的处理程序，但是
//无法正确回答，因为它涉及钱包状态。
//这些命令将在btcwallet中实现。
func handleAskWallet(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, ErrRPCNoWallet
}

//handleaddnode处理addnode命令。
func handleAddNode(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.AddNodeCmd)

	addr := normalizeAddress(c.Addr, s.cfg.ChainParams.DefaultPort)
	var err error
	switch c.SubCmd {
	case "add":
		err = s.cfg.ConnMgr.Connect(addr, true)
	case "remove":
		err = s.cfg.ConnMgr.RemoveByAddr(addr)
	case "onetry":
		err = s.cfg.ConnMgr.Connect(addr, false)
	default:
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidParameter,
			Message: "invalid subcommand for addnode",
		}
	}

	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidParameter,
			Message: err.Error(),
		}
	}

//除非出现错误，否则不会返回任何数据。
	return nil, nil
}

//handlenode处理节点命令。
func handleNode(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.NodeCmd)

	var addr string
	var nodeID uint64
	var errN, err error
	params := s.cfg.ChainParams
	switch c.SubCmd {
	case "disconnect":
//如果我们有一个有效的uint disconnect by node id，否则，
//尝试按地址断开连接，如果
//未提供有效的IP地址。
		if nodeID, errN = strconv.ParseUint(c.Target, 10, 32); errN == nil {
			err = s.cfg.ConnMgr.DisconnectByID(int32(nodeID))
		} else {
			if _, _, errP := net.SplitHostPort(c.Target); errP == nil || net.ParseIP(c.Target) != nil {
				addr = normalizeAddress(c.Target, params.DefaultPort)
				err = s.cfg.ConnMgr.DisconnectByAddr(addr)
			} else {
				return nil, &btcjson.RPCError{
					Code:    btcjson.ErrRPCInvalidParameter,
					Message: "invalid address or node ID",
				}
			}
		}
		if err != nil && peerExists(s.cfg.ConnMgr, addr, int32(nodeID)) {

			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCMisc,
				Message: "can't disconnect a permanent peer, use remove",
			}
		}

	case "remove":
//如果我们有一个有效的uint disconnect by node id，否则，
//尝试按地址断开连接，如果
//未提供有效的IP地址。
		if nodeID, errN = strconv.ParseUint(c.Target, 10, 32); errN == nil {
			err = s.cfg.ConnMgr.RemoveByID(int32(nodeID))
		} else {
			if _, _, errP := net.SplitHostPort(c.Target); errP == nil || net.ParseIP(c.Target) != nil {
				addr = normalizeAddress(c.Target, params.DefaultPort)
				err = s.cfg.ConnMgr.RemoveByAddr(addr)
			} else {
				return nil, &btcjson.RPCError{
					Code:    btcjson.ErrRPCInvalidParameter,
					Message: "invalid address or node ID",
				}
			}
		}
		if err != nil && peerExists(s.cfg.ConnMgr, addr, int32(nodeID)) {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCMisc,
				Message: "can't remove a temporary peer, use disconnect",
			}
		}

	case "connect":
		addr = normalizeAddress(c.Target, params.DefaultPort)

//默认为临时连接。
		subCmd := "temp"
		if c.ConnectSubCmd != nil {
			subCmd = *c.ConnectSubCmd
		}

		switch subCmd {
		case "perm", "temp":
			err = s.cfg.ConnMgr.Connect(addr, subCmd == "perm")
		default:
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCInvalidParameter,
				Message: "invalid subcommand for node connect",
			}
		}
	default:
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidParameter,
			Message: "invalid subcommand for node",
		}
	}

	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidParameter,
			Message: err.Error(),
		}
	}

//除非出现错误，否则不会返回任何数据。
	return nil, nil
}

//Peerexists确定给定的某个对等机当前是否已连接
//有关当前连接的所有对等机的信息。对等存在是
//使用目标地址或节点ID确定。
func peerExists(connMgr rpcserverConnManager, addr string, nodeID int32) bool {
	for _, p := range connMgr.ConnectedPeers() {
		if p.ToPeer().ID() == nodeID || p.ToPeer().Addr() == addr {
			return true
		}
	}
	return false
}

//messagetohex使用
//并返回结果的十六进制编码字符串。
func messageToHex(msg wire.Message) (string, error) {
	var buf bytes.Buffer
	if err := msg.BtcEncode(&buf, maxProtocolVersion, wire.WitnessEncoding); err != nil {
		context := fmt.Sprintf("Failed to encode msg of type %T", msg)
		return "", internalRPCError(err.Error(), context)
	}

	return hex.EncodeToString(buf.Bytes()), nil
}

//handleCreaterawtransaction处理createrawtransaction命令。
func handleCreateRawTransaction(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.CreateRawTransactionCmd)

//验证锁定时间（如果给定）。
	if c.LockTime != nil &&
		(*c.LockTime < 0 || *c.LockTime > int64(wire.MaxTxInSequenceNum)) {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidParameter,
			Message: "Locktime out of range",
		}
	}

//执行后将所有事务输入添加到新事务
//一些有效性检查。
	mtx := wire.NewMsgTx(wire.TxVersion)
	for _, input := range c.Inputs {
		txHash, err := chainhash.NewHashFromStr(input.Txid)
		if err != nil {
			return nil, rpcDecodeHexError(input.Txid)
		}

		prevOut := wire.NewOutPoint(txHash, input.Vout)
		txIn := wire.NewTxIn(prevOut, []byte{}, nil)
		if c.LockTime != nil && *c.LockTime != 0 {
			txIn.Sequence = wire.MaxTxInSequenceNum - 1
		}
		mtx.AddTxIn(txIn)
	}

//执行后将所有事务输出添加到事务
//一些有效性检查。
	params := s.cfg.ChainParams
	for encodedAddr, amount := range c.Amounts {
//确保金额在货币金额的有效范围内。
		if amount <= 0 || amount > btcutil.MaxSatoshi {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCType,
				Message: "Invalid amount",
			}
		}

//解码提供的地址。
		addr, err := btcutil.DecodeAddress(encodedAddr, params)
		if err != nil {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCInvalidAddressOrKey,
				Message: "Invalid address or key: " + err.Error(),
			}
		}

//确保地址是受支持的类型之一，并且
//用地址编码的网络与
//服务器当前处于打开状态。
		switch addr.(type) {
		case *btcutil.AddressPubKeyHash:
		case *btcutil.AddressScriptHash:
		default:
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCInvalidAddressOrKey,
				Message: "Invalid address or key",
			}
		}
		if !addr.IsForNet(params) {
			return nil, &btcjson.RPCError{
				Code: btcjson.ErrRPCInvalidAddressOrKey,
				Message: "Invalid address: " + encodedAddr +
					" is for the wrong network",
			}
		}

//创建一个支付到所提供地址的新脚本。
		pkScript, err := txscript.PayToAddrScript(addr)
		if err != nil {
			context := "Failed to generate pay-to-address script"
			return nil, internalRPCError(err.Error(), context)
		}

//将金额转换为Satoshi。
		satoshi, err := btcutil.NewAmount(amount)
		if err != nil {
			context := "Failed to convert amount"
			return nil, internalRPCError(err.Error(), context)
		}

		txOut := wire.NewTxOut(int64(satoshi), pkScript)
		mtx.AddTxOut(txOut)
	}

//设置锁定时间（如果给定）。
	if c.LockTime != nil {
		mtx.LockTime = uint32(*c.LockTime)
	}

//返回序列化和十六进制编码的事务。注意这个
//故意不直接返回，因为第一次返回
//值是一个字符串，它将导致返回空字符串到
//在出现错误的情况下，客户机而不是零（nil）。
	mtxHex, err := messageToHex(mtx)
	if err != nil {
		return nil, err
	}
	return mtxHex, nil
}

//handledebuglevel处理debuglevel命令。
func handleDebugLevel(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.DebugLevelCmd)

//列出支持的子系统的特殊显示命令。
	if c.LevelSpec == "show" {
		return fmt.Sprintf("Supported subsystems %v",
			supportedSubsystems()), nil
	}

	err := parseAndSetDebugLevels(c.LevelSpec)
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidParams.Code,
			Message: err.Error(),
		}
	}

	return "Done.", nil
}

//WitnessToHex将传递的见证堆栈格式化为十六进制编码的切片
//要在JSON响应中使用的字符串。
func witnessToHex(witness wire.TxWitness) []string {
//确保在没有条目和空条目时返回nil
//切片，以便根据需要适当省略。
	if len(witness) == 0 {
		return nil
	}

	result := make([]string, 0, len(witness))
	for _, wit := range witness {
		result = append(result, hex.EncodeToString(wit))
	}

	return result
}

//CreateVinList为传递的
//交易。
func createVinList(mtx *wire.MsgTx) []btcjson.Vin {
//根据定义，coinbase事务只有一个txin。
	vinList := make([]btcjson.Vin, len(mtx.TxIn))
	if blockchain.IsCoinBaseTx(mtx) {
		txIn := mtx.TxIn[0]
		vinList[0].Coinbase = hex.EncodeToString(txIn.SignatureScript)
		vinList[0].Sequence = txIn.Sequence
		vinList[0].Witness = witnessToHex(txIn.Witness)
		return vinList
	}

	for i, txIn := range mtx.TxIn {
//反汇编的字符串将包含[错误]内联
//如果脚本没有完全解析，那么忽略
//这里出错。
		disbuf, _ := txscript.DisasmString(txIn.SignatureScript)

		vinEntry := &vinList[i]
		vinEntry.Txid = txIn.PreviousOutPoint.Hash.String()
		vinEntry.Vout = txIn.PreviousOutPoint.Index
		vinEntry.Sequence = txIn.Sequence
		vinEntry.ScriptSig = &btcjson.ScriptSig{
			Asm: disbuf,
			Hex: hex.EncodeToString(txIn.SignatureScript),
		}

		if mtx.HasWitness() {
			vinEntry.Witness = witnessToHex(txIn.Witness)
		}
	}

	return vinList
}

//createvoutlist为传递的输出返回一个JSON对象切片
//交易。
func createVoutList(mtx *wire.MsgTx, chainParams *chaincfg.Params, filterAddrMap map[string]struct{}) []btcjson.Vout {
	voutList := make([]btcjson.Vout, 0, len(mtx.TxOut))
	for i, v := range mtx.TxOut {
//如果
//脚本没有完全解析，因此忽略此处的错误。
		disbuf, _ := txscript.DisasmString(v.PkScript)

//忽略此处的错误，因为错误意味着脚本
//无法分析，没有关于
//无论如何。
		scriptClass, addrs, reqSigs, _ := txscript.ExtractPkScriptAddrs(
			v.PkScript, chainParams)

//在检查地址是否通过
//需要时过滤。
		passesFilter := len(filterAddrMap) == 0
		encodedAddrs := make([]string, len(addrs))
		for j, addr := range addrs {
			encodedAddr := addr.EncodeAddress()
			encodedAddrs[j] = encodedAddr

//如果过滤器已经存在，则无需再次检查地图。
//传球。
			if passesFilter {
				continue
			}
			if _, exists := filterAddrMap[encodedAddr]; exists {
				passesFilter = true
			}
		}

		if !passesFilter {
			continue
		}

		var vout btcjson.Vout
		vout.N = uint32(i)
		vout.Value = btcutil.Amount(v.Value).ToBTC()
		vout.ScriptPubKey.Addresses = encodedAddrs
		vout.ScriptPubKey.Asm = disbuf
		vout.ScriptPubKey.Hex = hex.EncodeToString(v.PkScript)
		vout.ScriptPubKey.Type = scriptClass.String()
		vout.ScriptPubKey.ReqSigs = int32(reqSigs)

		voutList = append(voutList, vout)
	}

	return voutList
}

//createtxrawresult转换传递的事务和相关参数
//到原始事务JSON对象。
func createTxRawResult(chainParams *chaincfg.Params, mtx *wire.MsgTx,
	txHash string, blkHeader *wire.BlockHeader, blkHash string,
	blkHeight int32, chainHeight int32) (*btcjson.TxRawResult, error) {

	mtxHex, err := messageToHex(mtx)
	if err != nil {
		return nil, err
	}

	txReply := &btcjson.TxRawResult{
		Hex:      mtxHex,
		Txid:     txHash,
		Hash:     mtx.WitnessHash().String(),
		Size:     int32(mtx.SerializeSize()),
		Vsize:    int32(mempool.GetTxVirtualSize(btcutil.NewTx(mtx))),
		Vin:      createVinList(mtx),
		Vout:     createVoutList(mtx, chainParams, nil),
		Version:  mtx.Version,
		LockTime: mtx.LockTime,
	}

	if blkHeader != nil {
//这不是打字错误，它们在比特币上也是一样的。
		txReply.Time = blkHeader.Timestamp.Unix()
		txReply.Blocktime = blkHeader.Timestamp.Unix()
		txReply.BlockHash = blkHash
		txReply.Confirmations = uint64(1 + chainHeight - blkHeight)
	}

	return txReply, nil
}

//handledecoderawTransaction处理decoderawTransaction命令。
func handleDecodeRawTransaction(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.DecodeRawTransactionCmd)

//反序列化事务。
	hexStr := c.HexTx
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}
	serializedTx, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, rpcDecodeHexError(hexStr)
	}
	var mtx wire.MsgTx
	err = mtx.Deserialize(bytes.NewReader(serializedTx))
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCDeserialization,
			Message: "TX decode failed: " + err.Error(),
		}
	}

//创建并返回结果。
	txReply := btcjson.TxRawDecodeResult{
		Txid:     mtx.TxHash().String(),
		Version:  mtx.Version,
		Locktime: mtx.LockTime,
		Vin:      createVinList(&mtx),
		Vout:     createVoutList(&mtx, s.cfg.ChainParams, nil),
	}
	return txReply, nil
}

//handledecodeDescription处理解码脚本命令。
func handleDecodeScript(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.DecodeScriptCmd)

//将十六进制脚本转换为字节。
	hexStr := c.HexScript
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}
	script, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, rpcDecodeHexError(hexStr)
	}

//如果脚本
//没有完全解析，因此忽略此处的错误。
	disbuf, _ := txscript.DisasmString(script)

//获取有关脚本的信息。
//忽略此处的错误，因为错误意味着脚本无法分析
//而且也没有关于它的额外信息。
	scriptClass, addrs, reqSigs, _ := txscript.ExtractPkScriptAddrs(script,
		s.cfg.ChainParams)
	addresses := make([]string, len(addrs))
	for i, addr := range addrs {
		addresses[i] = addr.EncodeAddress()
	}

//将脚本本身转换为付费脚本哈希地址。
	p2sh, err := btcutil.NewAddressScriptHash(script, s.cfg.ChainParams)
	if err != nil {
		context := "Failed to convert script to pay-to-script-hash"
		return nil, internalRPCError(err.Error(), context)
	}

//生成并返回答复。
	reply := btcjson.DecodeScriptResult{
		Asm:       disbuf,
		ReqSigs:   int32(reqSigs),
		Type:      scriptClass.String(),
		Addresses: addresses,
	}
	if scriptClass != txscript.ScriptHashTy {
		reply.P2sh = p2sh.EncodeAddress()
	}
	return reply, nil
}

//handleestimatefee处理estimatefee命令。
func handleEstimateFee(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.EstimateFeeCmd)

	if s.cfg.FeeEstimator == nil {
		return nil, errors.New("Fee estimation disabled")
	}

	if c.NumBlocks <= 0 {
		return -1.0, errors.New("Parameter NumBlocks must be positive")
	}

	feeRate, err := s.cfg.FeeEstimator.EstimateFee(uint32(c.NumBlocks))

	if err != nil {
		return -1.0, err
	}

//转换为每KB的Satoshis。
	return float64(feeRate), nil
}

//handlegenerate句柄生成命令。
func handleGenerate(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
//如果没有地址支付
//已将块创建到。
	if len(cfg.miningAddrs) == 0 {
		return nil, &btcjson.RPCError{
			Code: btcjson.ErrRPCInternal.Code,
			Message: "No payment addresses specified " +
				"via --miningaddr",
		}
	}

//如果挖掘块的机会几乎为零，则响应错误。
//用CPU。
	if !s.cfg.ChainParams.GenerateSupported {
		return nil, &btcjson.RPCError{
			Code: btcjson.ErrRPCDifficulty,
			Message: fmt.Sprintf("No support for `generate` on "+
				"the current network, %s, as it's unlikely to "+
				"be possible to mine a block with the CPU.",
				s.cfg.ChainParams.Net),
		}
	}

	c := cmd.(*btcjson.GenerateCmd)

//如果客户端请求生成0个块，则响应时出错。
	if c.NumBlocks == 0 {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInternal.Code,
			Message: "Please request a nonzero number of blocks to generate.",
		}
	}

//创建回复
	reply := make([]string, c.NumBlocks)

	blockHashes, err := s.cfg.CPUMiner.GenerateNBlocks(c.NumBlocks)
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInternal.Code,
			Message: err.Error(),
		}
	}

//挖掘正确的块数，指定
//把每一个散列到它在回复中的位置。
	for i, hash := range blockHashes {
		reply[i] = hash.String()
	}

	return reply, nil
}

//handleegatedNodeInfo处理getaddedNodeInfo命令。
func handleGetAddedNodeInfo(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetAddedNodeInfoCmd)

//从服务器检索持久（添加）对等点列表，并
//按指定地址（如果有）筛选对等点列表。
	peers := s.cfg.ConnMgr.PersistentPeers()
	if c.Node != nil {
		node := *c.Node
		found := false
		for i, peer := range peers {
			if peer.ToPeer().Addr() == node {
				peers = peers[i : i+1]
				found = true
			}
		}
		if !found {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCClientNodeNotAdded,
				Message: "Node has not been added",
			}
		}
	}

//如果没有DNS标志，结果只是地址的一部分
//串。
	if !c.DNS {
		results := make([]string, 0, len(peers))
		for _, peer := range peers {
			results = append(results, peer.ToPeer().Addr())
		}
		return results, nil
	}

//使用dns标志，结果是一个json对象数组，其中
//包括每个对等机的DNS查找结果。
	results := make([]*btcjson.GetAddedNodeInfoResult, 0, len(peers))
	for _, rpcPeer := range peers {
//设置可能是IP地址的对等机的“地址”
//或者域名。
		peer := rpcPeer.ToPeer()
		var result btcjson.GetAddedNodeInfoResult
		result.AddedNode = peer.Addr()
		result.Connected = btcjson.Bool(peer.Connected())

//将地址分成主机和端口部分，这样我们可以
//对主机的DNS查找。在中未指定端口时
//地址，只需使用该地址作为主机即可。
		host, _, err := net.SplitHostPort(peer.Addr())
		if err != nil {
			host = peer.Addr()
		}

		var ipList []string
		switch {
		case net.ParseIP(host) != nil, strings.HasSuffix(host, ".onion"):
			ipList = make([]string, 1)
			ipList[0] = host
		default:
//对地址执行DNS查找。如果查找失败，只需
//使用主机。
			ips, err := btcdLookup(host)
			if err != nil {
				ipList = make([]string, 1)
				ipList[0] = host
				break
			}
			ipList = make([]string, 0, len(ips))
			for _, ip := range ips {
				ipList = append(ipList, ip.String())
			}
		}

//将地址和连接信息添加到结果中。
		addrs := make([]btcjson.GetAddedNodeInfoResultAddr, 0, len(ipList))
		for _, ip := range ipList {
			var addr btcjson.GetAddedNodeInfoResultAddr
			addr.Address = ip
			addr.Connected = "false"
			if ip == host && peer.Connected() {
				addr.Connected = directionString(peer.Inbound())
			}
			addrs = append(addrs, addr)
		}
		result.Addresses = &addrs
		results = append(results, &result)
	}
	return results, nil
}

//handlegetbestblock实现getbestblock命令。
func handleGetBestBlock(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
//所有其他“get block”命令都给出了
//哈希，或者两者都需要块sha。这两者都是为了
//最好的街区。
	best := s.cfg.Chain.BestSnapshot()
	result := &btcjson.GetBestBlockResult{
		Hash:   best.Hash.String(),
		Height: best.Height,
	}
	return result, nil
}

//handlegetbestBlockHash实现getBestBlockHash命令。
func handleGetBestBlockHash(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	best := s.cfg.Chain.BestSnapshot()
	return best.Hash.String(), nil
}

//GetDifficultyratio将工作难度的证明作为
//使用块头传递的位字段的最小困难。
func getDifficultyRatio(bits uint32, params *chaincfg.Params) float64 {
//最小的困难是工作极限位的最大可能证明。
//已转换回数字。注意这与
//直接因为块难度被编码在块中而限制工作
//结构紧凑，精度低。
	max := blockchain.CompactToBig(params.PowLimitBits)
	target := blockchain.CompactToBig(bits)

	difficulty := new(big.Rat).SetFrac(max, target)
	outString := difficulty.FloatString(8)
	diff, err := strconv.ParseFloat(outString, 64)
	if err != nil {
		rpcsLog.Errorf("Cannot get difficulty: %v", err)
		return 0
	}
	return diff
}

//handlegetblock执行getblock命令。
func handleGetBlock(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetBlockCmd)

//从数据库加载原始块字节。
	hash, err := chainhash.NewHashFromStr(c.Hash)
	if err != nil {
		return nil, rpcDecodeHexError(c.Hash)
	}
	var blkBytes []byte
	err = s.cfg.DB.View(func(dbTx database.Tx) error {
		var err error
		blkBytes, err = dbTx.FetchBlock(hash)
		return err
	})
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCBlockNotFound,
			Message: "Block not found",
		}
	}

//如果没有设置verbose标志，只需返回序列化块
//作为十六进制编码的字符串。
	if c.Verbose != nil && !*c.Verbose {
		return hex.EncodeToString(blkBytes), nil
	}

//设置了verbose标志，因此生成JSON对象并返回它。

//反序列化块。
	blk, err := btcutil.NewBlockFromBytes(blkBytes)
	if err != nil {
		context := "Failed to deserialize block"
		return nil, internalRPCError(err.Error(), context)
	}

//从链条上获取块高度。
	blockHeight, err := s.cfg.Chain.BlockHeightByHash(hash)
	if err != nil {
		context := "Failed to obtain block height"
		return nil, internalRPCError(err.Error(), context)
	}
	blk.SetHeight(blockHeight)
	best := s.cfg.Chain.BestSnapshot()

//获取下一个块哈希，除非没有。
	var nextHashString string
	if blockHeight < best.Height {
		nextHash, err := s.cfg.Chain.BlockHashByHeight(blockHeight + 1)
		if err != nil {
			context := "No next block"
			return nil, internalRPCError(err.Error(), context)
		}
		nextHashString = nextHash.String()
	}

	params := s.cfg.ChainParams
	blockHeader := &blk.MsgBlock().Header
	blockReply := btcjson.GetBlockVerboseResult{
		Hash:          c.Hash,
		Version:       blockHeader.Version,
		VersionHex:    fmt.Sprintf("%08x", blockHeader.Version),
		MerkleRoot:    blockHeader.MerkleRoot.String(),
		PreviousHash:  blockHeader.PrevBlock.String(),
		Nonce:         blockHeader.Nonce,
		Time:          blockHeader.Timestamp.Unix(),
		Confirmations: int64(1 + best.Height - blockHeight),
		Height:        int64(blockHeight),
		Size:          int32(len(blkBytes)),
		StrippedSize:  int32(blk.MsgBlock().SerializeSizeStripped()),
		Weight:        int32(blockchain.GetBlockWeight(blk)),
		Bits:          strconv.FormatInt(int64(blockHeader.Bits), 16),
		Difficulty:    getDifficultyRatio(blockHeader.Bits, params),
		NextHash:      nextHashString,
	}

	if c.VerboseTx == nil || !*c.VerboseTx {
		transactions := blk.Transactions()
		txNames := make([]string, len(transactions))
		for i, tx := range transactions {
			txNames[i] = tx.Hash().String()
		}

		blockReply.Tx = txNames
	} else {
		txns := blk.Transactions()
		rawTxns := make([]btcjson.TxRawResult, len(txns))
		for i, tx := range txns {
			rawTxn, err := createTxRawResult(params, tx.MsgTx(),
				tx.Hash().String(), blockHeader, hash.String(),
				blockHeight, best.Height)
			if err != nil {
				return nil, err
			}
			rawTxns[i] = *rawTxn
		}
		blockReply.RawTx = rawTxns
	}

	return blockReply, nil
}

//SoftForkStatus将阈值状态转换为人类可读的字符串
//与特定状态相对应。
func softForkStatus(state blockchain.ThresholdState) (string, error) {
	switch state {
	case blockchain.ThresholdDefined:
		return "defined", nil
	case blockchain.ThresholdStarted:
		return "started", nil
	case blockchain.ThresholdLockedIn:
		return "lockedin", nil
	case blockchain.ThresholdActive:
		return "active", nil
	case blockchain.ThresholdFailed:
		return "failed", nil
	default:
		return "", fmt.Errorf("unknown deployment state: %v", state)
	}
}

//handlegetblockchaininfo实现getblockchaininfo命令。
func handleGetBlockChainInfo(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
//获取当前最知名区块链状态的快照。我们将
//主要从该快照填充对此调用的响应。
	params := s.cfg.ChainParams
	chain := s.cfg.Chain
	chainSnapshot := chain.BestSnapshot()

	chainInfo := &btcjson.GetBlockChainInfoResult{
		Chain:         params.Name,
		Blocks:        chainSnapshot.Height,
		Headers:       chainSnapshot.Height,
		BestBlockHash: chainSnapshot.Hash.String(),
		Difficulty:    getDifficultyRatio(chainSnapshot.Bits, params),
		MedianTime:    chainSnapshot.MedianTime.Unix(),
		Pruned:        false,
		Bip9SoftForks: make(map[string]*btcjson.Bip9SoftForkDescription),
	}

//接下来，用描述当前
//通过超级多数块部署的软分叉的状态
//信号机制。
	height := chainSnapshot.Height
	chainInfo.SoftForks = []*btcjson.SoftForkDescription{
		{
			ID:      "bip34",
			Version: 2,
			Reject: struct {
				Status bool `json:"status"`
			}{
				Status: height >= params.BIP0034Height,
			},
		},
		{
			ID:      "bip66",
			Version: 3,
			Reject: struct {
				Status bool `json:"status"`
			}{
				Status: height >= params.BIP0066Height,
			},
		},
		{
			ID:      "bip65",
			Version: 4,
			Reject: struct {
				Status bool `json:"status"`
			}{
				Status: height >= params.BIP0065Height,
			},
		},
	}

//最后，查询当前所有的bip0009版本位状态
//定义了bip0009软分叉部署。
	for deployment, deploymentDetails := range params.Deployments {
//将整数部署ID映射到可读的
//叉名。
		var forkName string
		switch deployment {
		case chaincfg.DeploymentTestDummy:
			forkName = "dummy"

		case chaincfg.DeploymentCSV:
			forkName = "csv"

		case chaincfg.DeploymentSegwit:
			forkName = "segwit"

		default:
			return nil, &btcjson.RPCError{
				Code: btcjson.ErrRPCInternal.Code,
				Message: fmt.Sprintf("Unknown deployment %v "+
					"detected", deployment),
			}
		}

//查询链以了解部署的当前状态
//由其部署ID标识。
		deploymentStatus, err := chain.ThresholdState(uint32(deployment))
		if err != nil {
			context := "Failed to obtain deployment status"
			return nil, internalRPCError(err.Error(), context)
		}

//尝试将当前部署状态转换为
//可读字符串。如果状态无法识别，则
//返回非零错误。
		statusString, err := softForkStatus(deploymentStatus)
		if err != nil {
			return nil, &btcjson.RPCError{
				Code: btcjson.ErrRPCInternal.Code,
				Message: fmt.Sprintf("unknown deployment status: %v",
					deploymentStatus),
			}
		}

//最后，用所有
//以上收集的信息。
		chainInfo.Bip9SoftForks[forkName] = &btcjson.Bip9SoftForkDescription{
			Status:    strings.ToLower(statusString),
			Bit:       deploymentDetails.BitNumber,
			StartTime: int64(deploymentDetails.StartTime),
			Timeout:   int64(deploymentDetails.ExpireTime),
		}
	}

	return chainInfo, nil
}

//handlegetblockcount实现getblockcount命令。
func handleGetBlockCount(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	best := s.cfg.Chain.BestSnapshot()
	return int64(best.Height), nil
}

//handlegetblockhash实现getblockhash命令。
func handleGetBlockHash(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetBlockHashCmd)
	hash, err := s.cfg.Chain.BlockHashByHeight(int32(c.Index))
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCOutOfRange,
			Message: "Block number out of range",
		}
	}

	return hash.String(), nil
}

//handlegetblockheader实现getblockheader命令。
func handleGetBlockHeader(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetBlockHeaderCmd)

//从链中获取收割台。
	hash, err := chainhash.NewHashFromStr(c.Hash)
	if err != nil {
		return nil, rpcDecodeHexError(c.Hash)
	}
	blockHeader, err := s.cfg.Chain.HeaderByHash(hash)
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCBlockNotFound,
			Message: "Block not found",
		}
	}

//如果没有设置verbose标志，只需返回序列化块
//作为十六进制编码字符串的头。
	if c.Verbose != nil && !*c.Verbose {
		var headerBuf bytes.Buffer
		err := blockHeader.Serialize(&headerBuf)
		if err != nil {
			context := "Failed to serialize block header"
			return nil, internalRPCError(err.Error(), context)
		}
		return hex.EncodeToString(headerBuf.Bytes()), nil
	}

//设置了verbose标志，因此生成JSON对象并返回它。

//从链条上获取块高度。
	blockHeight, err := s.cfg.Chain.BlockHeightByHash(hash)
	if err != nil {
		context := "Failed to obtain block height"
		return nil, internalRPCError(err.Error(), context)
	}
	best := s.cfg.Chain.BestSnapshot()

//获取下一个块哈希，除非没有。
	var nextHashString string
	if blockHeight < best.Height {
		nextHash, err := s.cfg.Chain.BlockHashByHeight(blockHeight + 1)
		if err != nil {
			context := "No next block"
			return nil, internalRPCError(err.Error(), context)
		}
		nextHashString = nextHash.String()
	}

	params := s.cfg.ChainParams
	blockHeaderReply := btcjson.GetBlockHeaderVerboseResult{
		Hash:          c.Hash,
		Confirmations: int64(1 + best.Height - blockHeight),
		Height:        blockHeight,
		Version:       blockHeader.Version,
		VersionHex:    fmt.Sprintf("%08x", blockHeader.Version),
		MerkleRoot:    blockHeader.MerkleRoot.String(),
		NextHash:      nextHashString,
		PreviousHash:  blockHeader.PrevBlock.String(),
		Nonce:         uint64(blockHeader.Nonce),
		Time:          blockHeader.Timestamp.Unix(),
		Bits:          strconv.FormatInt(int64(blockHeader.Bits), 16),
		Difficulty:    getDifficultyRatio(blockHeader.Bits, params),
	}
	return blockHeaderReply, nil
}

//encodeTemplateID将传递的详细信息编码为可用于
//唯一标识块模板。
func encodeTemplateID(prevHash *chainhash.Hash, lastGenerated time.Time) string {
	return fmt.Sprintf("%s-%d", prevHash.String(), lastGenerated.Unix())
}

//decodeTemplateID解码用于唯一标识块的ID
//模板。这主要用作跟踪何时更新客户端的机制
//对块模板使用长轮询。ID由
//关联模板的上一个块哈希以及关联模板的时间
//模板已生成。
func decodeTemplateID(templateID string) (*chainhash.Hash, int64, error) {
	fields := strings.Split(templateID, "-")
	if len(fields) != 2 {
		return nil, 0, errors.New("invalid longpollid format")
	}

	prevHash, err := chainhash.NewHashFromStr(fields[0])
	if err != nil {
		return nil, 0, errors.New("invalid longpollid format")
	}
	lastGenerated, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return nil, 0, errors.New("invalid longpollid format")
	}

	return prevHash, lastGenerated, nil
}

//notifyLongPoller通知已注册为
//块模板过期时通知。
//
//必须在状态为“锁定”的情况下调用此函数。
func (state *gbtWorkState) notifyLongPollers(latestHash *chainhash.Hash, lastGenerated time.Time) {
//通知任何正在等待块模板更新的
//散列，这不是最佳链的尖端散列，因为它们
//工作现在无效。
	for hash, channels := range state.notifyMap {
		if !hash.IsEqual(latestHash) {
			for _, c := range channels {
				close(c)
			}
			delete(state.notifyMap, hash)
		}
	}

//如果提供的最后生成的时间戳尚未
//初始化。
	if lastGenerated.IsZero() {
		return
	}

//如果没有为当前更新注册的内容，请立即返回
//最佳块哈希。
	channels, ok := state.notifyMap[*latestHash]
	if !ok {
		return
	}

//通知任何正在等待块模板更新的
//在最近生成的块之前生成的块模板
//模板。
	lastGeneratedUnix := lastGenerated.Unix()
	for lastGen, c := range channels {
		if lastGen < lastGeneratedUnix {
			close(c)
			delete(channels, lastGen)
		}
	}

//如果不再注册，则完全删除条目
//渠道。
	if len(channels) == 0 {
		delete(state.notifyMap, *latestHash)
	}
}

//notify block connected使用新连接的块通知任何长轮询
//当现有块模板为
//由于新连接的块而过时。
func (state *gbtWorkState) NotifyBlockConnected(blockHash *chainhash.Hash) {
	go func() {
		state.Lock()
		defer state.Unlock()

		state.notifyLongPollers(blockHash, state.lastTxUpdate)
	}()
}

//notifymempooltx为事务内存使用上次更新的新时间
//用于通知具有新块模板的长轮询客户端的池
//现有的块模板由于经过足够的时间和内容而过时
//内存池的更改。
func (state *gbtWorkState) NotifyMempoolTx(lastUpdated time.Time) {
	go func() {
		state.Lock()
		defer state.Unlock()

//如果没有生成块模板，则无需通知任何内容
//然而。
		if state.prevHash == nil || state.lastGenerated.IsZero() {
			return
		}

		if time.Now().After(state.lastGenerated.Add(time.Second *
			gbtRegenerateSeconds)) {

			state.notifyLongPollers(state.prevHash, lastUpdated)
		}
	}()
}

//templateUpdateChan返回一个通道，该通道将在块
//与传递的上一个哈希和上一次生成时间关联的模板
//是陈旧的。此函数将返回现有通道进行复制
//允许多个客户端等待同一块模板的参数
//不需要为每个客户机使用不同的通道。
//
//必须在状态为“锁定”的情况下调用此函数。
func (state *gbtWorkState) templateUpdateChan(prevHash *chainhash.Hash, lastGenerated int64) chan struct{} {
//获取当前等待更新的频道列表
//更改前一个哈希的块模板或创建新的哈希模板。
	channels, ok := state.notifyMap[*prevHash]
	if !ok {
		m := make(map[int64]chan struct{})
		state.notifyMap[*prevHash] = m
		channels = m
	}

//获取与块模板时间关联的当前通道
//上次生成或创建新的。
	c, ok := channels[lastGenerated]
	if !ok {
		c = make(chan struct{})
		channels[lastGenerated] = c
	}

	return c
}

//updateBlockTemplate为工作状态创建或更新块模板。
//当当前最佳块具有
//已更改或内存池中的事务已更新，并且
//从上一个模板生成以来已经足够长了。否则，
//更新现有块模板的时间戳（可能
//根据Consesus规则，在测试网上有困难）。最后，如果
//useCoinBaseValue标志为false，而现有的块模板不为
//已包含有效的付款地址，将更新块模板
//从配置的列表中随机选择付款地址
//地址。
//
//必须在状态为“锁定”的情况下调用此函数。
func (state *gbtWorkState) updateBlockTemplate(s *rpcServer, useCoinbaseValue bool) error {
	generator := s.cfg.Generator
	lastTxUpdate := generator.TxSource().LastUpdated()
	if lastTxUpdate.IsZero() {
		lastTxUpdate = time.Now()
	}

//当当前最佳块具有
//已更改或内存池中的事务已更新，并且
//自从上一个模板
//生成。
	var msgBlock *wire.MsgBlock
	var targetDifficulty string
	latestHash := &s.cfg.Chain.BestSnapshot().Hash
	template := state.template
	if template == nil || state.prevHash == nil ||
		!state.prevHash.IsEqual(latestHash) ||
		(state.lastTxUpdate != lastTxUpdate &&
			time.Now().After(state.lastGenerated.Add(time.Second*
				gbtRegenerateSeconds))) {

//重置生成块模板的上一个最佳哈希
//因此下面的任何错误都会导致下一次调用尝试
//再一次。
		state.prevHash = nil

//如果呼叫者请求
//完全的CoinBase，而不是只需要相关的细节
//创造自己的硬币库。
		var payAddr btcutil.Address
		if !useCoinbaseValue {
			payAddr = cfg.miningAddrs[rand.Intn(len(cfg.miningAddrs))]
		}

//创建一个新的块模板，该模板具有任何
//可以赎回。这是可以接受的，因为返回的
//块模板不包含coinbase，因此调用方
//最终会创造出自己的硬币库，支付给
//适当的地址。
		blkTemplate, err := generator.NewBlockTemplate(payAddr)
		if err != nil {
			return internalRPCError("Failed to create new block "+
				"template: "+err.Error(), "")
		}
		template = blkTemplate
		msgBlock = template.Block
		targetDifficulty = fmt.Sprintf("%064x",
			blockchain.CompactToBig(msgBlock.Header.Bits))

//根据
//每个链最后几个块的中间时间戳
//共识规则。
		best := s.cfg.Chain.BestSnapshot()
		minTimestamp := mining.MinimumMedianTime(best)

//更新工作状态以确保另一个块模板
//在需要之前生成。
		state.template = template
		state.lastGenerated = time.Now()
		state.lastTxUpdate = lastTxUpdate
		state.prevHash = latestHash
		state.minTimestamp = minTimestamp

		rpcsLog.Debugf("Generated block template (timestamp %v, "+
			"target %s, merkle root %s)",
			msgBlock.Header.Timestamp, targetDifficulty,
			msgBlock.Header.MerkleRoot)

//通知长时间轮询新的
//模板。
		state.notifyLongPollers(latestHash, lastTxUpdate)
	} else {
//此时，存在一个已保存的块模板和另一个
//已请求模板，但
//交易没有变化，或者时间不够长
//触发要生成的新块模板。所以，更新
//现有块模板。

//当调用者需要一个完整的coinbase而不是
//创造自己的硬币库所需的相关细节，
//将付款地址添加到
//模板，如果它还没有。因为这需要
//要通过配置指定的挖掘地址，错误为
//如果未指定，则返回。
		if !useCoinbaseValue && !template.ValidPayAddress {
//随机选择付款地址。
			payToAddr := cfg.miningAddrs[rand.Intn(len(cfg.miningAddrs))]

//将模板的块coinbase输出更新为
//支付到随机选择的支付地址。
			pkScript, err := txscript.PayToAddrScript(payToAddr)
			if err != nil {
				context := "Failed to create pay-to-addr script"
				return internalRPCError(err.Error(), context)
			}
			template.Block.Transactions[0].TxOut[0].PkScript = pkScript
			template.ValidPayAddress = true

//更新merkle根目录。
			block := btcutil.NewBlock(template.Block)
			merkles := blockchain.BuildMerkleTreeStore(block.Transactions(), false)
			template.Block.Header.MerkleRoot = *merkles[len(merkles)-1]
		}

//为方便设置本地变量。
		msgBlock = template.Block
		targetDifficulty = fmt.Sprintf("%064x",
			blockchain.CompactToBig(msgBlock.Header.Bits))

//将块模板的时间更新为当前时间
//同时考虑了过去几年的中位数时间
//根据链共识规则阻止。
		generator.UpdateBlockTime(msgBlock)
		msgBlock.Header.Nonce = 0

		rpcsLog.Debugf("Updated block template (timestamp %v, "+
			"target %s)", msgBlock.Header.Timestamp,
			targetDifficulty)
	}

	return nil
}

//BlockTemplateResult返回与
//状态为btcjson.getBlockTemplateResult，已准备好编码为json
//然后返回给呼叫者。
//
//必须在状态为“锁定”的情况下调用此函数。
func (state *gbtWorkState) blockTemplateResult(useCoinbaseValue bool, submitOld *bool) (*btcjson.GetBlockTemplateResult, error) {
//确保时间戳仍在模板的有效范围内。
//只有当本地时钟改变时才会发生这种情况
//在生成模板之后，但必须避免提供
//块模板无效。
	template := state.template
	msgBlock := template.Block
	header := &msgBlock.Header
	adjustedTime := state.timeSource.AdjustedTime()
	maxTime := adjustedTime.Add(time.Second * blockchain.MaxTimeOffsetSeconds)
	if header.Timestamp.After(maxTime) {
		return nil, &btcjson.RPCError{
			Code: btcjson.ErrRPCOutOfRange,
			Message: fmt.Sprintf("The template time is after the "+
				"maximum allowed time for a block - template "+
				"time %v, maximum time %v", adjustedTime,
				maxTime),
		}
	}

//将块模板中的每个事务转换为模板结果
//交易。结果不包括coinbase，因此请注意
//对不同长度和指数的调整。
	numTx := len(msgBlock.Transactions)
	transactions := make([]btcjson.GetBlockTemplateResultTx, 0, numTx-1)
	txIndex := make(map[chainhash.Hash]int64, numTx)
	for i, tx := range msgBlock.Transactions {
		txHash := tx.TxHash()
		txIndex[txHash] = int64(i)

//跳过coinbase事务。
		if i == 0 {
			continue
		}

//为以下事务创建一个基于1的索引数组
//在交易列表中的这个之前
//取决于。这是必需的，因为创建的块必须
//确保依赖项的顺序正确。使用地图
//在创建最终数组以防止重复项之前
//当多个输入引用同一事务时。
		dependsMap := make(map[int64]struct{})
		for _, txIn := range tx.TxIn {
			if idx, ok := txIndex[txIn.PreviousOutPoint.Hash]; ok {
				dependsMap[idx] = struct{}{}
			}
		}
		depends := make([]int64, 0, len(dependsMap))
		for idx := range dependsMap {
			depends = append(depends, idx)
		}

//将事务序列化，以便以后转换为十六进制。
		txBuf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
		if err := tx.Serialize(txBuf); err != nil {
			context := "Failed to serialize transaction"
			return nil, internalRPCError(err.Error(), context)
		}

		bTx := btcutil.NewTx(tx)
		resultTx := btcjson.GetBlockTemplateResultTx{
			Data:    hex.EncodeToString(txBuf.Bytes()),
			Hash:    txHash.String(),
			Depends: depends,
			Fee:     template.Fees[i],
			SigOps:  template.SigOpCosts[i],
			Weight:  blockchain.GetTransactionWeight(bTx),
		}
		transactions = append(transactions, resultTx)
	}

//生成块模板回复。注意以下突变是
//字段的包含或省略所暗示的：
//包括mintime->时间/减量
//省略coinbasetxn->coinbase，生成
	targetDifficulty := fmt.Sprintf("%064x", blockchain.CompactToBig(header.Bits))
	templateID := encodeTemplateID(state.prevHash, state.lastGenerated)
	reply := btcjson.GetBlockTemplateResult{
		Bits:         strconv.FormatInt(int64(header.Bits), 16),
		CurTime:      header.Timestamp.Unix(),
		Height:       int64(template.Height),
		PreviousHash: header.PrevBlock.String(),
		WeightLimit:  blockchain.MaxBlockWeight,
		SigOpLimit:   blockchain.MaxBlockSigOpsCost,
		SizeLimit:    wire.MaxBlockPayload,
		Transactions: transactions,
		Version:      header.Version,
		LongPollID:   templateID,
		SubmitOld:    submitOld,
		Target:       targetDifficulty,
		MinTime:      state.minTimestamp.Unix(),
		MaxTime:      maxTime.Unix(),
		Mutable:      gbtMutableFields,
		NonceRange:   gbtNonceRange,
		Capabilities: gbtCapabilities,
	}
//如果生成的块模板包含带见证的事务
//数据，然后将见证承诺包括在GBT结果中。
	if template.WitnessCommitment != nil {
		reply.DefaultWitnessCommitment = hex.EncodeToString(template.WitnessCommitment)
	}

	if useCoinbaseValue {
		reply.CoinbaseAux = gbtCoinbaseAux
		reply.CoinbaseValue = &msgBlock.Transactions[0].TxOut[0].Value
	} else {
//确保模板具有关联的有效付款地址
//当一个完整的硬币库被要求时。
		if !template.ValidPayAddress {
			return nil, &btcjson.RPCError{
				Code: btcjson.ErrRPCInternal.Code,
				Message: "A coinbase transaction has been " +
					"requested, but the server has not " +
					"been configured with any payment " +
					"addresses via --miningaddr",
			}
		}

//序列化事务以转换为十六进制。
		tx := msgBlock.Transactions[0]
		txBuf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
		if err := tx.Serialize(txBuf); err != nil {
			context := "Failed to serialize transaction"
			return nil, internalRPCError(err.Error(), context)
		}

		resultTx := btcjson.GetBlockTemplateResultTx{
			Data:    hex.EncodeToString(txBuf.Bytes()),
			Hash:    tx.TxHash().String(),
			Depends: []int64{},
			Fee:     template.Fees[0],
			SigOps:  template.SigOpCosts[0],
		}

		reply.CoinbaseTxn = &resultTx
	}

	return &reply, nil
}

//handlegetblockTemplateLongPoll是handlegetblockTemplateRequest的助手
//它处理块模板的长轮询。当呼叫者
//发送一个具有以前返回的长投票ID的请求，一个响应
//直到调用方停止处理上一个块时才发送
//模板支持新模板。尤其是当
//由于已找到解决方案，旧块模板不再有效
//并添加到区块链中，或者新的事务已经出现，并且有一段时间
//没有找到解决方案。
//
//更多详情请参见https://en.bitcoin.it/wiki/bip0022。
func handleGetBlockTemplateLongPoll(s *rpcServer, longPollID string, useCoinbaseValue bool, closeChan <-chan struct{}) (interface{}, error) {
	state := s.gbtWorkState
	state.Lock()
//国家解锁在这里是故意不拖延的，因为它需要
//在等待有关块的通知之前手动解锁
//模板更改。

	if err := state.updateBlockTemplate(s, useCoinbaseValue); err != nil {
		state.Unlock()
		return nil, err
	}

//如果提供的长轮询ID
//呼叫方无效。
	prevHash, lastGenerated, err := decodeTemplateID(longPollID)
	if err != nil {
		result, err := state.blockTemplateResult(useCoinbaseValue, nil)
		if err != nil {
			state.Unlock()
			return nil, err
		}

		state.Unlock()
		return result, nil
	}

//如果特定的块模板
//由长轮询ID标识的不再与当前块匹配
//模板，因为这意味着提供的模板已过时。
	prevTemplateHash := &state.template.Block.Header.PrevBlock
	if !prevHash.IsEqual(prevTemplateHash) ||
		lastGenerated != state.lastGenerated.Unix() {

//包括提交工作是否有效
//旧的块模板取决于解决方案是否具有
//已找到并添加到区块链。
		submitOld := prevHash.IsEqual(prevTemplateHash)
		result, err := state.blockTemplateResult(useCoinbaseValue,
			&submitOld)
		if err != nil {
			state.Unlock()
			return nil, err
		}

		state.Unlock()
		return result, nil
	}

//为通知注册上一个哈希和上一次生成的时间
//获取将在与关联的模板
//提供的ID已过时，应将新的块模板返回到
//呼叫者。
	longPollChan := state.templateUpdateChan(prevHash, lastGenerated)
	state.Unlock()

	select {
//当客户机在发送回复之前关闭时，只需返回
//现在，Goroutine就不在这里了。
	case <-closeChan:
		return nil, ErrClientQuit

//等待收到信号后发送回复。
	case <-longPollChan:
//坠落
	}

//获取最新的块模板
	state.Lock()
	defer state.Unlock()

	if err := state.updateBlockTemplate(s, useCoinbaseValue); err != nil {
		return nil, err
	}

//包括提交与旧版本相比的工作是否有效
//块模板取决于解决方案是否已经
//找到并添加到区块链。
	submitOld := prevHash.IsEqual(&state.template.Block.Header.PrevBlock)
	result, err := state.blockTemplateResult(useCoinbaseValue, &submitOld)
	if err != nil {
		return nil, err
	}

	return result, nil
}

//handlegetblocktemplateRequest是handlegetblocktemplate的助手，它
//处理生成块模板并将其返回给调用方。它
//处理bip 0022指定的长轮询请求和常规请求
//请求。此外，它还检测调用方报告的功能
//关于它是否支持创建自己的货币库
//并修改返回的块
//相应的模板。
func handleGetBlockTemplateRequest(s *rpcServer, request *btcjson.TemplateRequest, closeChan <-chan struct{}) (interface{}, error) {
//提取相关传递的功能并将结果限制为
//CoinBase值或CoinBase事务对象取决于
//请求。默认为仅提供coinbase值。
	useCoinbaseValue := true
	if request != nil {
		var hasCoinbaseValue, hasCoinbaseTxn bool
		for _, capability := range request.Capabilities {
			switch capability {
			case "coinbasetxn":
				hasCoinbaseTxn = true
			case "coinbasevalue":
				hasCoinbaseValue = true
			}
		}

		if hasCoinbaseTxn && !hasCoinbaseValue {
			useCoinbaseValue = false
		}
	}

//当请求了CoinBase事务时，响应时出错
//如果没有要向其支付创建的块模板的地址。
	if !useCoinbaseValue && len(cfg.miningAddrs) == 0 {
		return nil, &btcjson.RPCError{
			Code: btcjson.ErrRPCInternal.Code,
			Message: "A coinbase transaction has been requested, " +
				"but the server has not been configured with " +
				"any payment addresses via --miningaddr",
		}
	}

//如果没有连接的对等端，则返回错误，因为没有
//中继找到的块或接收要处理的事务的方法。
//但是，在回归测试中运行或
//模拟测试模式。
	if !(cfg.RegressionTest || cfg.SimNet) &&
		s.cfg.ConnMgr.ConnectedCount() == 0 {

		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCClientNotConnected,
			Message: "Bitcoin is not connected",
		}
	}

//同步链之前，没有生成或接受工作的意义。
	currentHeight := s.cfg.Chain.BestSnapshot().Height
	if currentHeight != 0 && !s.cfg.SyncMgr.IsCurrent() {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCClientInInitialDownload,
			Message: "Bitcoin is downloading blocks...",
		}
	}

//当提供长投票ID时，这是由
//当ID引用的块模板应该
//换成新的。
	if request != nil && request.LongPollID != "" {
		return handleGetBlockTemplateLongPoll(s, request.LongPollID,
			useCoinbaseValue, closeChan)
	}

//更新块模板时保护并发访问。
	state := s.gbtWorkState
	state.Lock()
	defer state.Unlock()

//获取并返回块模板。新的块模板将
//当当前最佳块或事务更改时生成
//内存池中的已更新，至少有5个
//自上次生成模板以来的秒数。否则，
//更新现有块模板的时间戳（可能
//根据Consesus规则测试网络的难度）。
	if err := state.updateBlockTemplate(s, useCoinbaseValue); err != nil {
		return nil, err
	}
	return state.blockTemplateResult(useCoinbaseValue, nil)
}

//ChainerTogBterrString将从BtcChain返回的错误转换为字符串
//与Bip0022中描述的拒绝原因和格式相匹配
//原因。
func chainErrToGBTErrString(err error) string {
//当传递的错误不是RuleError时，只返回一个泛型
//已拒绝包含错误文本的字符串。
	ruleErr, ok := err.(blockchain.RuleError)
	if !ok {
		return "rejected: " + err.Error()
	}

	switch ruleErr.ErrorCode {
	case blockchain.ErrDuplicateBlock:
		return "duplicate"
	case blockchain.ErrBlockTooBig:
		return "bad-blk-length"
	case blockchain.ErrBlockWeightTooHigh:
		return "bad-blk-weight"
	case blockchain.ErrBlockVersionTooOld:
		return "bad-version"
	case blockchain.ErrInvalidTime:
		return "bad-time"
	case blockchain.ErrTimeTooOld:
		return "time-too-old"
	case blockchain.ErrTimeTooNew:
		return "time-too-new"
	case blockchain.ErrDifficultyTooLow:
		return "bad-diffbits"
	case blockchain.ErrUnexpectedDifficulty:
		return "bad-diffbits"
	case blockchain.ErrHighHash:
		return "high-hash"
	case blockchain.ErrBadMerkleRoot:
		return "bad-txnmrklroot"
	case blockchain.ErrBadCheckpoint:
		return "bad-checkpoint"
	case blockchain.ErrForkTooOld:
		return "fork-too-old"
	case blockchain.ErrCheckpointTimeTooOld:
		return "checkpoint-time-too-old"
	case blockchain.ErrNoTransactions:
		return "bad-txns-none"
	case blockchain.ErrNoTxInputs:
		return "bad-txns-noinputs"
	case blockchain.ErrNoTxOutputs:
		return "bad-txns-nooutputs"
	case blockchain.ErrTxTooBig:
		return "bad-txns-size"
	case blockchain.ErrBadTxOutValue:
		return "bad-txns-outputvalue"
	case blockchain.ErrDuplicateTxInputs:
		return "bad-txns-dupinputs"
	case blockchain.ErrBadTxInput:
		return "bad-txns-badinput"
	case blockchain.ErrMissingTxOut:
		return "bad-txns-missinginput"
	case blockchain.ErrUnfinalizedTx:
		return "bad-txns-unfinalizedtx"
	case blockchain.ErrDuplicateTx:
		return "bad-txns-duplicate"
	case blockchain.ErrOverwriteTx:
		return "bad-txns-overwrite"
	case blockchain.ErrImmatureSpend:
		return "bad-txns-maturity"
	case blockchain.ErrSpendTooHigh:
		return "bad-txns-highspend"
	case blockchain.ErrBadFees:
		return "bad-txns-fees"
	case blockchain.ErrTooManySigOps:
		return "high-sigops"
	case blockchain.ErrFirstTxNotCoinbase:
		return "bad-txns-nocoinbase"
	case blockchain.ErrMultipleCoinbases:
		return "bad-txns-multicoinbase"
	case blockchain.ErrBadCoinbaseScriptLen:
		return "bad-cb-length"
	case blockchain.ErrBadCoinbaseValue:
		return "bad-cb-value"
	case blockchain.ErrMissingCoinbaseHeight:
		return "bad-cb-height"
	case blockchain.ErrBadCoinbaseHeight:
		return "bad-cb-height"
	case blockchain.ErrScriptMalformed:
		return "bad-script-malformed"
	case blockchain.ErrScriptValidation:
		return "bad-script-validate"
	case blockchain.ErrUnexpectedWitness:
		return "unexpected-witness"
	case blockchain.ErrInvalidWitnessCommitment:
		return "bad-witness-nonce-size"
	case blockchain.ErrWitnessCommitmentMismatch:
		return "bad-witness-merkle-match"
	case blockchain.ErrPreviousBlockUnknown:
		return "prev-blk-not-found"
	case blockchain.ErrInvalidAncestorBlock:
		return "bad-prevblk"
	case blockchain.ErrPrevBlockNotBest:
		return "inconclusive-not-best-prvblk"
	}

	return "rejected: " + err.Error()
}

//handlegetblocktemplateproposal是handlegetblocktemplate的助手，它
//处理块建议。
//
//更多详情请参见https://en.bitcoin.it/wiki/bip0023。
func handleGetBlockTemplateProposal(s *rpcServer, request *btcjson.TemplateRequest) (interface{}, error) {
	hexData := request.Data
	if hexData == "" {
		return false, &btcjson.RPCError{
			Code: btcjson.ErrRPCType,
			Message: fmt.Sprintf("Data must contain the " +
				"hex-encoded serialized block that is being " +
				"proposed"),
		}
	}

//确保提供的数据是健全的，并反序列化建议的块。
	if len(hexData)%2 != 0 {
		hexData = "0" + hexData
	}
	dataBytes, err := hex.DecodeString(hexData)
	if err != nil {
		return false, &btcjson.RPCError{
			Code: btcjson.ErrRPCDeserialization,
			Message: fmt.Sprintf("Data must be "+
				"hexadecimal string (not %q)", hexData),
		}
	}
	var msgBlock wire.MsgBlock
	if err := msgBlock.Deserialize(bytes.NewReader(dataBytes)); err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCDeserialization,
			Message: "Block decode failed: " + err.Error(),
		}
	}
	block := btcutil.NewBlock(&msgBlock)

//确保块是从预期的前一个块构建的。
	expectedPrevHash := s.cfg.Chain.BestSnapshot().Hash
	prevHash := &block.MsgBlock().Header.PrevBlock
	if !expectedPrevHash.IsEqual(prevHash) {
		return "bad-prevblk", nil
	}

	if err := s.cfg.Chain.CheckConnectBlockTemplate(block); err != nil {
		if _, ok := err.(blockchain.RuleError); !ok {
			errStr := fmt.Sprintf("Failed to process block proposal: %v", err)
			rpcsLog.Error(errStr)
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCVerify,
				Message: errStr,
			}
		}

		rpcsLog.Infof("Rejected block proposal: %v", err)
		return chainErrToGBTErrString(err), nil
	}

	return nil, nil
}

//handlegetblocktemplate实现getblocktemplate命令。
//
//参见https://en.bitcoin.it/wiki/bip0022和
//更多详情请访问https://en.bitcoin.it/wiki/bip0023。
func handleGetBlockTemplate(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetBlockTemplateCmd)
	request := c.Request

//设置默认模式并在提供时覆盖它。
	mode := "template"
	if request != nil && request.Mode != "" {
		mode = request.Mode
	}

	switch mode {
	case "template":
		return handleGetBlockTemplateRequest(s, request, closeChan)
	case "proposal":
		return handleGetBlockTemplateProposal(s, request)
	}

	return nil, &btcjson.RPCError{
		Code:    btcjson.ErrRPCInvalidParameter,
		Message: "Invalid mode",
	}
}

//handlegetcfilter执行getcpilter命令。
func handleGetCFilter(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	if s.cfg.CfIndex == nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCNoCFIndex,
			Message: "The CF index must be enabled for this command",
		}
	}

	c := cmd.(*btcjson.GetCFilterCmd)
	hash, err := chainhash.NewHashFromStr(c.Hash)
	if err != nil {
		return nil, rpcDecodeHexError(c.Hash)
	}

	filterBytes, err := s.cfg.CfIndex.FilterByBlockHash(hash, c.FilterType)
	if err != nil {
		rpcsLog.Debugf("Could not find committed filter for %v: %v",
			hash, err)
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCBlockNotFound,
			Message: "Block not found",
		}
	}

	rpcsLog.Debugf("Found committed filter for %v", hash)
	return hex.EncodeToString(filterBytes), nil
}

//handleGetFilterHeader执行getFilterHeader命令。
func handleGetCFilterHeader(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	if s.cfg.CfIndex == nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCNoCFIndex,
			Message: "The CF index must be enabled for this command",
		}
	}

	c := cmd.(*btcjson.GetCFilterHeaderCmd)
	hash, err := chainhash.NewHashFromStr(c.Hash)
	if err != nil {
		return nil, rpcDecodeHexError(c.Hash)
	}

	headerBytes, err := s.cfg.CfIndex.FilterHeaderByBlockHash(hash, c.FilterType)
	if len(headerBytes) > 0 {
		rpcsLog.Debugf("Found header of committed filter for %v", hash)
	} else {
		rpcsLog.Debugf("Could not find header of committed filter for %v: %v",
			hash, err)
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCBlockNotFound,
			Message: "Block not found",
		}
	}

	hash.SetBytes(headerBytes)
	return hash.String(), nil
}

//handlegetConnectionCount实现getConnectionCount命令。
func handleGetConnectionCount(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return s.cfg.ConnMgr.ConnectedCount(), nil
}

//handlegetcurrentnet执行getcurrentnet命令。
func handleGetCurrentNet(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return s.cfg.ChainParams.Net, nil
}

//handlegedtfficity实现getdifficulty命令。
func handleGetDifficulty(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	best := s.cfg.Chain.BestSnapshot()
	return getDifficultyRatio(best.Bits, s.cfg.ChainParams), nil
}

//handleGetGenerate实现getGenerate命令。
func handleGetGenerate(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return s.cfg.CPUMiner.IsMining(), nil
}

//handlegethashespersec实现gethashespersec命令。
func handleGetHashesPerSec(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return int64(s.cfg.CPUMiner.HashesPerSecond()), nil
}

//handlegeaders实现getheaders命令。
//
//注意：这是一个btcSuite扩展，最初从
//github.com/decred/dcrd.
func handleGetHeaders(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetHeadersCmd)

//从链中获取请求的头，同时考虑提供的
//阻止定位器并停止哈希。
	blockLocators := make([]*chainhash.Hash, len(c.BlockLocators))
	for i := range c.BlockLocators {
		blockLocator, err := chainhash.NewHashFromStr(c.BlockLocators[i])
		if err != nil {
			return nil, rpcDecodeHexError(c.BlockLocators[i])
		}
		blockLocators[i] = blockLocator
	}
	var hashStop chainhash.Hash
	if c.HashStop != "" {
		err := chainhash.Decode(&hashStop, c.HashStop)
		if err != nil {
			return nil, rpcDecodeHexError(c.HashStop)
		}
	}
	headers := s.cfg.SyncMgr.LocateHeaders(blockLocators, &hashStop)

//以十六进制编码字符串的形式返回序列化的块头。
	hexBlockHeaders := make([]string, len(headers))
	var buf bytes.Buffer
	for i, h := range headers {
		err := h.Serialize(&buf)
		if err != nil {
			return nil, internalRPCError(err.Error(),
				"Failed to serialize block header")
		}
		hexBlockHeaders[i] = hex.EncodeToString(buf.Bytes())
		buf.Reset()
	}
	return hexBlockHeaders, nil
}

//handlegetinfo实现getinfo命令。我们只返回田地
//与钱包功能无关。
func handleGetInfo(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	best := s.cfg.Chain.BestSnapshot()
	ret := &btcjson.InfoChainResult{
		Version:         int32(1000000*appMajor + 10000*appMinor + 100*appPatch),
		ProtocolVersion: int32(maxProtocolVersion),
		Blocks:          best.Height,
		TimeOffset:      int64(s.cfg.TimeSource.Offset().Seconds()),
		Connections:     s.cfg.ConnMgr.ConnectedCount(),
		Proxy:           cfg.Proxy,
		Difficulty:      getDifficultyRatio(best.Bits, s.cfg.ChainParams),
		TestNet:         cfg.TestNet3,
		RelayFee:        cfg.minRelayTxFee.ToBTC(),
	}

	return ret, nil
}

//handlegetmempoolinfo实现getmempoolinfo命令。
func handleGetMempoolInfo(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	mempoolTxns := s.cfg.TxMemPool.TxDescs()

	var numBytes int64
	for _, txD := range mempoolTxns {
		numBytes += int64(txD.Tx.MsgTx().SerializeSize())
	}

	ret := &btcjson.GetMempoolInfoResult{
		Size:  int64(len(mempoolTxns)),
		Bytes: numBytes,
	}

	return ret, nil
}

//handlegetmininginfo实现getmininginfo命令。我们只返回
//与钱包功能无关的字段。
func handleGetMiningInfo(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
//创建默认的getnetworkhashps命令以使用默认值并使
//使用现有的GetNetworkHashPS处理程序。
	gnhpsCmd := btcjson.NewGetNetworkHashPSCmd(nil, nil)
	networkHashesPerSecIface, err := handleGetNetworkHashPS(s, gnhpsCmd,
		closeChan)
	if err != nil {
		return nil, err
	}
	networkHashesPerSec, ok := networkHashesPerSecIface.(int64)
	if !ok {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInternal.Code,
			Message: "networkHashesPerSec is not an int64",
		}
	}

	best := s.cfg.Chain.BestSnapshot()
	result := btcjson.GetMiningInfoResult{
		Blocks:             int64(best.Height),
		CurrentBlockSize:   best.BlockSize,
		CurrentBlockWeight: best.BlockWeight,
		CurrentBlockTx:     best.NumTxns,
		Difficulty:         getDifficultyRatio(best.Bits, s.cfg.ChainParams),
		Generate:           s.cfg.CPUMiner.IsMining(),
		GenProcLimit:       s.cfg.CPUMiner.NumWorkers(),
		HashesPerSec:       int64(s.cfg.CPUMiner.HashesPerSecond()),
		NetworkHashPS:      networkHashesPerSec,
		PooledTx:           uint64(s.cfg.TxMemPool.Count()),
		TestNet:            cfg.TestNet3,
	}
	return &result, nil
}

//handlegetnettotals实现getnettotals命令。
func handleGetNetTotals(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	totalBytesRecv, totalBytesSent := s.cfg.ConnMgr.NetTotals()
	reply := &btcjson.GetNetTotalsResult{
		TotalBytesRecv: totalBytesRecv,
		TotalBytesSent: totalBytesSent,
		TimeMillis:     time.Now().UTC().UnixNano() / int64(time.Millisecond),
	}
	return reply, nil
}

//handlegenetworkhashps实现getnetworkhashps命令。
func handleGetNetworkHashPS(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
//注意：所有有效的错误返回路径都应返回Int64。
//文本零被推断为int，不会强制为int64
//因为返回值是一个接口。

	c := cmd.(*btcjson.GetNetworkHashPSCmd)

//当通过的高度太高或为零时，立即返回0
//因为我们不能合理地计算网络散列的数量
//每秒来自无效值的。当为负时，使用电流
//最佳块高度。
	best := s.cfg.Chain.BestSnapshot()
	endHeight := int32(-1)
	if c.Height != nil {
		endHeight = int32(*c.Height)
	}
	if endHeight > best.Height || endHeight == 0 {
		return int64(0), nil
	}
	if endHeight < 0 {
		endHeight = best.Height
	}

//根据以下公式计算每个重定目标间隔的块数：
//链参数。
	blocksPerRetarget := int32(s.cfg.ChainParams.TargetTimespan /
		s.cfg.ChainParams.TargetTimePerBlock)

//根据通过的
//阻碍。当传递的值为负时，使用最后一个块
//难度随着起始高度的变化而变化。同时确保
//开始高度不在链条开始之前。
	numBlocks := int32(120)
	if c.Blocks != nil {
		numBlocks = int32(*c.Blocks)
	}
	var startHeight int32
	if numBlocks <= 0 {
		startHeight = endHeight - ((endHeight % blocksPerRetarget) + 1)
	} else {
		startHeight = endHeight - numBlocks
	}
	if startHeight < 0 {
		startHeight = 0
	}
	rpcsLog.Debugf("Calculating network hashes per second from %d to %d",
		startHeight, endHeight)

//查找最小和最大块时间戳并计算总数
//开始块和结束块之间发生的工作量。
	var minTimestamp, maxTimestamp time.Time
	totalWork := big.NewInt(0)
	for curHeight := startHeight; curHeight <= endHeight; curHeight++ {
		hash, err := s.cfg.Chain.BlockHashByHeight(curHeight)
		if err != nil {
			context := "Failed to fetch block hash"
			return nil, internalRPCError(err.Error(), context)
		}

//从链中获取收割台。
		header, err := s.cfg.Chain.HeaderByHash(hash)
		if err != nil {
			context := "Failed to fetch block header"
			return nil, internalRPCError(err.Error(), context)
		}

		if curHeight == startHeight {
			minTimestamp = header.Timestamp
			maxTimestamp = minTimestamp
		} else {
			totalWork.Add(totalWork, blockchain.CalcWork(header.Bits))

			if minTimestamp.After(header.Timestamp) {
				minTimestamp = header.Timestamp
			}
			if maxTimestamp.Before(header.Timestamp) {
				maxTimestamp = header.Timestamp
			}
		}
	}

//计算最小和最大块之间的秒数差
//如果没有时间戳，则避免被零除
//时差。
	timeDiff := int64(maxTimestamp.Sub(minTimestamp) / time.Second)
	if timeDiff == 0 {
		return int64(0), nil
	}

	hashesPerSec := new(big.Int).Div(totalWork, big.NewInt(timeDiff))
	return hashesPerSec.Int64(), nil
}

//handlegetpeerinfo实现getpeerinfo命令。
func handleGetPeerInfo(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	peers := s.cfg.ConnMgr.ConnectedPeers()
	syncPeerID := s.cfg.SyncMgr.SyncPeerID()
	infos := make([]*btcjson.GetPeerInfoResult, 0, len(peers))
	for _, p := range peers {
		statsSnap := p.ToPeer().StatsSnapshot()
		info := &btcjson.GetPeerInfoResult{
			ID:             statsSnap.ID,
			Addr:           statsSnap.Addr,
			AddrLocal:      p.ToPeer().LocalAddr().String(),
			Services:       fmt.Sprintf("%08d", uint64(statsSnap.Services)),
			RelayTxes:      !p.IsTxRelayDisabled(),
			LastSend:       statsSnap.LastSend.Unix(),
			LastRecv:       statsSnap.LastRecv.Unix(),
			BytesSent:      statsSnap.BytesSent,
			BytesRecv:      statsSnap.BytesRecv,
			ConnTime:       statsSnap.ConnTime.Unix(),
			PingTime:       float64(statsSnap.LastPingMicros),
			TimeOffset:     statsSnap.TimeOffset,
			Version:        statsSnap.Version,
			SubVer:         statsSnap.UserAgent,
			Inbound:        statsSnap.Inbound,
			StartingHeight: statsSnap.StartingHeight,
			CurrentHeight:  statsSnap.LastBlock,
			BanScore:       int32(p.BanScore()),
			FeeFilter:      p.FeeFilter(),
			SyncNode:       statsSnap.ID == syncPeerID,
		}
		if p.ToPeer().LastPingNonce() != 0 {
			wait := float64(time.Since(statsSnap.LastPingTime).Nanoseconds())
//我们实际上需要微秒。
			info.PingWait = wait / 1000
		}
		infos = append(infos, info)
	}
	return infos, nil
}

//handleGetrawmEmpool实现getrawmEmpool命令。
func handleGetRawMempool(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetRawMempoolCmd)
	mp := s.cfg.TxMemPool

	if c.Verbose != nil && *c.Verbose {
		return mp.RawMempoolVerbose(), nil
	}

//如果
//未设置详细标志。
	descs := mp.TxDescs()
	hashStrings := make([]string, len(descs))
	for i := range hashStrings {
		hashStrings[i] = descs[i].Tx.Hash().String()
	}

	return hashStrings, nil
}

//handlegrawtransaction实现getrawtransaction命令。
func handleGetRawTransaction(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetRawTransactionCmd)

//将提供的事务哈希十六进制转换为哈希。
	txHash, err := chainhash.NewHashFromStr(c.Txid)
	if err != nil {
		return nil, rpcDecodeHexError(c.Txid)
	}

	verbose := false
	if c.Verbose != nil {
		verbose = *c.Verbose != 0
	}

//尝试从内存池中提取事务，如果失败，
//尝试块数据库。
	var mtx *wire.MsgTx
	var blkHash *chainhash.Hash
	var blkHeight int32
	tx, err := s.cfg.TxMemPool.FetchTransaction(txHash)
	if err != nil {
		if s.cfg.TxIndex == nil {
			return nil, &btcjson.RPCError{
				Code: btcjson.ErrRPCNoTxInfo,
				Message: "The transaction index must be " +
					"enabled to query the blockchain " +
					"(specify --txindex)",
			}
		}

//查找事务的位置。
		blockRegion, err := s.cfg.TxIndex.TxBlockRegion(txHash)
		if err != nil {
			context := "Failed to retrieve transaction location"
			return nil, internalRPCError(err.Error(), context)
		}
		if blockRegion == nil {
			return nil, rpcNoTxInfoError(txHash)
		}

//从数据库加载原始事务字节。
		var txBytes []byte
		err = s.cfg.DB.View(func(dbTx database.Tx) error {
			var err error
			txBytes, err = dbTx.FetchBlockRegion(blockRegion)
			return err
		})
		if err != nil {
			return nil, rpcNoTxInfoError(txHash)
		}

//如果没有设置verbose标志，只需返回序列化的
//事务作为十六进制编码的字符串。这是为了
//避免反序列化它，只在以后重新序列化它。
		if !verbose {
			return hex.EncodeToString(txBytes), nil
		}

//抓住木块高度。
		blkHash = blockRegion.Hash
		blkHeight, err = s.cfg.Chain.BlockHeightByHash(blkHash)
		if err != nil {
			context := "Failed to retrieve block height"
			return nil, internalRPCError(err.Error(), context)
		}

//反序列化事务
		var msgTx wire.MsgTx
		err = msgTx.Deserialize(bytes.NewReader(txBytes))
		if err != nil {
			context := "Failed to deserialize transaction"
			return nil, internalRPCError(err.Error(), context)
		}
		mtx = &msgTx
	} else {
//如果没有设置verbose标志，只需返回
//网络将事务序列化为十六进制编码字符串。
		if !verbose {
//注意，这是有意而非直接
//返回，因为第一个返回值是
//字符串，它将导致返回空的
//字符串到客户端，而不是
//出现错误的情况。
			mtxHex, err := messageToHex(tx.MsgTx())
			if err != nil {
				return nil, err
			}
			return mtxHex, nil
		}

		mtx = tx.MsgTx()
	}

//设置了verbose标志，因此生成JSON对象并返回它。
	var blkHeader *wire.BlockHeader
	var blkHashStr string
	var chainHeight int32
	if blkHash != nil {
//从链中获取收割台。
		header, err := s.cfg.Chain.HeaderByHash(blkHash)
		if err != nil {
			context := "Failed to fetch block header"
			return nil, internalRPCError(err.Error(), context)
		}

		blkHeader = &header
		blkHashStr = blkHash.String()
		chainHeight = s.cfg.Chain.BestSnapshot().Height
	}

	rawTxn, err := createTxRawResult(s.cfg.ChainParams, mtx, txHash.String(),
		blkHeader, blkHashStr, blkHeight, chainHeight)
	if err != nil {
		return nil, err
	}
	return *rawTxn, nil
}

//handlegettxout处理gettxout命令。
func handleGetTxOut(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetTxOutCmd)

//将提供的事务哈希十六进制转换为哈希。
	txHash, err := chainhash.NewHashFromStr(c.Txid)
	if err != nil {
		return nil, rpcDecodeHexError(c.Txid)
	}

//如果请求，并且Tx在mempool中可用，请尝试获取它
//否则，尝试从块数据库中提取。
	var bestBlockHash string
	var confirmations int32
	var value int64
	var pkScript []byte
	var isCoinbase bool
	includeMempool := true
	if c.IncludeMempool != nil {
		includeMempool = *c.IncludeMempool
	}
//托多：这很刺激。它应该尝试直接获取并检查
//错误。
	if includeMempool && s.cfg.TxMemPool.HaveTransaction(txHash) {
		tx, err := s.cfg.TxMemPool.FetchTransaction(txHash)
		if err != nil {
			return nil, rpcNoTxInfoError(txHash)
		}

		mtx := tx.MsgTx()
		if c.Vout > uint32(len(mtx.TxOut)-1) {
			return nil, &btcjson.RPCError{
				Code: btcjson.ErrRPCInvalidTxVout,
				Message: "Output index number (vout) does not " +
					"exist for transaction.",
			}
		}

		txOut := mtx.TxOut[c.Vout]
		if txOut == nil {
			errStr := fmt.Sprintf("Output index: %d for txid: %s "+
				"does not exist", c.Vout, txHash)
			return nil, internalRPCError(errStr, "")
		}

		best := s.cfg.Chain.BestSnapshot()
		bestBlockHash = best.Hash.String()
		confirmations = 0
		value = txOut.Value
		pkScript = txOut.PkScript
		isCoinbase = blockchain.IsCoinBaseTx(mtx)
	} else {
		out := wire.OutPoint{Hash: *txHash, Index: c.Vout}
		entry, err := s.cfg.Chain.FetchUtxoEntry(out)
		if err != nil {
			return nil, rpcNoTxInfoError(txHash)
		}

//若要匹配引用客户端的行为，请返回nil
//（json-null）如果事务输出由另一个使用
//事务已在主链中。挖掘的事务
//由mempool事务花费的不受
//这个。
		if entry == nil || entry.IsSpent() {
			return nil, nil
		}

		best := s.cfg.Chain.BestSnapshot()
		bestBlockHash = best.Hash.String()
		confirmations = 1 + best.Height - entry.BlockHeight()
		value = entry.Amount()
		pkScript = entry.PkScript()
		isCoinbase = entry.IsCoinBase()
	}

//将脚本分解为单行可打印格式。
//如果脚本
//没有完全解析，因此忽略此处的错误。
	disbuf, _ := txscript.DisasmString(pkScript)

//获取关于脚本的更多信息。
//忽略此处的错误，因为错误意味着脚本无法分析
//而且也没有关于它的额外信息。
	scriptClass, addrs, reqSigs, _ := txscript.ExtractPkScriptAddrs(pkScript,
		s.cfg.ChainParams)
	addresses := make([]string, len(addrs))
	for i, addr := range addrs {
		addresses[i] = addr.EncodeAddress()
	}

	txOutReply := &btcjson.GetTxOutResult{
		BestBlock:     bestBlockHash,
		Confirmations: int64(confirmations),
		Value:         btcutil.Amount(value).ToBTC(),
		ScriptPubKey: btcjson.ScriptPubKeyResult{
			Asm:       disbuf,
			Hex:       hex.EncodeToString(pkScript),
			ReqSigs:   int32(reqSigs),
			Type:      scriptClass.String(),
			Addresses: addresses,
		},
		Coinbase: isCoinbase,
	}
	return txOutReply, nil
}

//handlehelp执行帮助命令。
func handleHelp(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.HelpCmd)

//当没有特定命令时，提供所有命令的使用概述
//指定。
	var command string
	if c.Command != nil {
		command = *c.Command
	}
	if command == "" {
		usage, err := s.helpCacher.rpcUsage(false)
		if err != nil {
			context := "Failed to generate RPC usage"
			return nil, internalRPCError(err.Error(), context)
		}
		return usage, nil
	}

//检查请求的命令是否受支持和实现。只有
//搜索处理程序的主列表，因为不应提供帮助
//对于未执行或与钱包相关的命令
//功能。
	if _, ok := rpcHandlers[command]; !ok {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidParameter,
			Message: "Unknown command: " + command,
		}
	}

//获取命令的帮助。
	help, err := s.helpCacher.rpcMethodHelp(command)
	if err != nil {
		context := "Failed to generate help"
		return nil, internalRPCError(err.Error(), context)
	}
	return help, nil
}

//handleping执行ping命令。
func handlePing(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
//要求服务器ping \o\u
	nonce, err := wire.RandomUint64()
	if err != nil {
		return nil, internalRPCError("Not sending ping - failed to "+
			"generate nonce: "+err.Error(), "")
	}
	s.cfg.ConnMgr.BroadcastMessage(wire.NewMsgPing(nonce))

	return nil, nil
}

//RetrieveDTX表示从
//事务内存池或来自数据库。当加载事务时
//从数据库中，它使用原始序列化字节加载，而
//mempool具有完全反序列化的结构。因此，该结构将
//根据从何处检索，设置两个字段中的一个。
//这主要是为了提高效率，以避免在
//可能的。
type retrievedTx struct {
	txBytes []byte
blkHash *chainhash.Hash //仅当事务在块中时设置。
	tx      *btcutil.Tx
}

//fetchinputtxos从由引用的所有事务中获取输出点。
//通过首先检查事务内存池，输入传递的事务
//然后是那些已经被挖掘成块的事务索引。
func fetchInputTxos(s *rpcServer, tx *wire.MsgTx) (map[wire.OutPoint]wire.TxOut, error) {
	mp := s.cfg.TxMemPool
	originOutputs := make(map[wire.OutPoint]wire.TxOut)
	for txInIndex, txIn := range tx.TxIn {
//尝试从中获取和使用引用的事务
//内存池。
		origin := &txIn.PreviousOutPoint
		originTx, err := mp.FetchTransaction(&origin.Hash)
		if err == nil {
			txOuts := originTx.MsgTx().TxOut
			if origin.Index >= uint32(len(txOuts)) {
				errStr := fmt.Sprintf("unable to find output "+
					"%v referenced from transaction %s:%d",
					origin, tx.TxHash(), txInIndex)
				return nil, internalRPCError(errStr, "")
			}

			originOutputs[*origin] = *txOuts[origin.Index]
			continue
		}

//查找事务的位置。
		blockRegion, err := s.cfg.TxIndex.TxBlockRegion(&origin.Hash)
		if err != nil {
			context := "Failed to retrieve transaction location"
			return nil, internalRPCError(err.Error(), context)
		}
		if blockRegion == nil {
			return nil, rpcNoTxInfoError(&origin.Hash)
		}

//从数据库加载原始事务字节。
		var txBytes []byte
		err = s.cfg.DB.View(func(dbTx database.Tx) error {
			var err error
			txBytes, err = dbTx.FetchBlockRegion(blockRegion)
			return err
		})
		if err != nil {
			return nil, rpcNoTxInfoError(&origin.Hash)
		}

//反序列化事务
		var msgTx wire.MsgTx
		err = msgTx.Deserialize(bytes.NewReader(txBytes))
		if err != nil {
			context := "Failed to deserialize transaction"
			return nil, internalRPCError(err.Error(), context)
		}

//将引用的输出添加到映射。
		if origin.Index >= uint32(len(msgTx.TxOut)) {
			errStr := fmt.Sprintf("unable to find output %v "+
				"referenced from transaction %s:%d", origin,
				tx.TxHash(), txInIndex)
			return nil, internalRPCError(errStr, "")
		}
		originOutputs[*origin] = *msgTx.TxOut[origin.Index]
	}

	return originOutputs, nil
}

//createvinlisprevout返回一个JSON对象切片，用于
//已传递事务。
func createVinListPrevOut(s *rpcServer, mtx *wire.MsgTx, chainParams *chaincfg.Params, vinExtra bool, filterAddrMap map[string]struct{}) ([]btcjson.VinPrevOut, error) {
//根据定义，coinbase事务只有一个txin。
	if blockchain.IsCoinBaseTx(mtx) {
//仅当筛选器映射为空时包括事务
//因为coinbase输入没有地址，所以永远不会
//匹配非空筛选器。
		if len(filterAddrMap) != 0 {
			return nil, nil
		}

		txIn := mtx.TxIn[0]
		vinList := make([]btcjson.VinPrevOut, 1)
		vinList[0].Coinbase = hex.EncodeToString(txIn.SignatureScript)
		vinList[0].Sequence = txIn.Sequence
		return vinList, nil
	}

//使用动态大小的列表来容纳地址筛选器。
	vinList := make([]btcjson.VinPrevOut, 0, len(mtx.TxIn))

//查找填充所需的所有引用事务输出
//以前的输出信息（如果需要）。
	var originOutputs map[wire.OutPoint]wire.TxOut
	if vinExtra || len(filterAddrMap) > 0 {
		var err error
		originOutputs, err = fetchInputTxos(s, mtx)
		if err != nil {
			return nil, err
		}
	}

	for _, txIn := range mtx.TxIn {
//反汇编的字符串将包含[错误]内联
//如果脚本没有完全解析，那么忽略
//这里出错。
		disbuf, _ := txscript.DisasmString(txIn.SignatureScript)

//创建基本输入项，不带附加的可选项
//以前的输出详细信息，如果
//请求和可用。
		prevOut := &txIn.PreviousOutPoint
		vinEntry := btcjson.VinPrevOut{
			Txid:     prevOut.Hash.String(),
			Vout:     prevOut.Index,
			Sequence: txIn.Sequence,
			ScriptSig: &btcjson.ScriptSig{
				Asm: disbuf,
				Hex: hex.EncodeToString(txIn.SignatureScript),
			},
		}

		if len(txIn.Witness) != 0 {
			vinEntry.Witness = witnessToHex(txIn.Witness)
		}

//如果已通过筛选，则立即将条目添加到列表中
//因为以前的输出可能不可用。
		passesFilter := len(filterAddrMap) == 0
		if passesFilter {
			vinList = append(vinList, vinEntry)
		}

//仅在请求和时填充以前的输出信息
//可用。
		if len(originOutputs) == 0 {
			continue
		}
		originTxOut, ok := originOutputs[*prevOut]
		if !ok {
			continue
		}

//忽略此处的错误，因为错误意味着脚本
//无法分析，没有关于
//无论如何。
		_, addrs, _, _ := txscript.ExtractPkScriptAddrs(
			originTxOut.PkScript, chainParams)

//在检查地址是否通过
//需要时过滤。
		encodedAddrs := make([]string, len(addrs))
		for j, addr := range addrs {
			encodedAddr := addr.EncodeAddress()
			encodedAddrs[j] = encodedAddr

//如果过滤器已经存在，则无需再次检查地图。
//传球。
			if passesFilter {
				continue
			}
			if _, exists := filterAddrMap[encodedAddr]; exists {
				passesFilter = true
			}
		}

//如果条目未通过筛选，则忽略它。
		if !passesFilter {
			continue
		}

//如果还没有在上面完成，请将条目添加到列表中。
		if len(filterAddrMap) != 0 {
			vinList = append(vinList, vinEntry)
		}

//如果
//请求。
		if vinExtra {
			vinListEntry := &vinList[len(vinList)-1]
			vinListEntry.PrevOut = &btcjson.PrevOut{
				Addresses: encodedAddrs,
				Value:     btcutil.Amount(originTxOut.Value).ToBTC(),
			}
		}
	}

	return vinList, nil
}

//fetchmumpooltxnsforaddress查询所有未确认的地址索引
//涉及所提供地址的交易。结果有限
//按要跳过的号码和请求的号码。
func fetchMempoolTxnsForAddress(s *rpcServer, addr btcutil.Address, numToSkip, numRequested uint32) ([]*btcutil.Tx, uint32) {
//当可用性低于
//跳过的数字。
	mpTxns := s.cfg.AddrIndex.UnconfirmedTxnsForAddress(addr)
	numAvailable := uint32(len(mpTxns))
	if numToSkip > numAvailable {
		return nil, numAvailable
	}

//根据要跳过的数字和数字筛选可用条目
//请求。
	rangeEnd := numToSkip + numRequested
	if rangeEnd > numAvailable {
		rangeEnd = numAvailable
	}
	return mpTxns[numToSkip:rangeEnd], numToSkip
}

//handlesearchrawtransactions实现searchrawtransactions命令。
func handleSearchRawTransactions(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
//如果未启用地址索引，则响应错误。
	addrIndex := s.cfg.AddrIndex
	if addrIndex == nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCMisc,
			Message: "Address index must be enabled (--addrindex)",
		}
	}

//覆盖标志，以便在
//如果需要，每个输入。
	c := cmd.(*btcjson.SearchRawTransactionsCmd)
	vinExtra := false
	if c.VinExtra != nil {
		vinExtra = *c.VinExtra != 0
	}

//包括额外的先前输出信息需要
//事务索引。目前，地址索引依赖于
//事务索引，因此此检查是多余的，但最好是
//以防地址索引被更改为不依赖于它。
	if vinExtra && s.cfg.TxIndex == nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCMisc,
			Message: "Transaction index must be enabled (--txindex)",
		}
	}

//尝试解码提供的地址。
	params := s.cfg.ChainParams
	addr, err := btcutil.DecodeAddress(c.Address, params)
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidAddressOrKey,
			Message: "Invalid address or key: " + err.Error(),
		}
	}

//如果需要，覆盖请求条目的默认数量。也，
//如果请求的条目数为零，请立即返回以避免
//额外的工作。
	numRequested := 100
	if c.Count != nil {
		numRequested = *c.Count
		if numRequested < 0 {
			numRequested = 1
		}
	}
	if numRequested == 0 {
		return nil, nil
	}

//如果需要，重写要跳过的默认条目数。
	var numToSkip int
	if c.Skip != nil {
		numToSkip = *c.Skip
		if numToSkip < 0 {
			numToSkip = 0
		}
	}

//如果需要，覆盖反转标志。
	var reverse bool
	if c.Reverse != nil {
		reverse = *c.Reverse
	}

//如果客户要求反向，请先从mempool添加事务
//秩序。否则，它们将最后添加（根据需要，取决于
//请求的计数）。
//
//注意：此代码不按依赖项排序。这可能是什么
//为了客户的方便在将来做，或者把它留给
//客户端。
	numSkipped := uint32(0)
	addressTxns := make([]retrievedTx, 0, numRequested)
	if reverse {
//mempool中的事务还不在块头中，
//因此，在retieved transaction结构中的block header字段
//剩下零。
		mpTxns, mpSkipped := fetchMempoolTxnsForAddress(s, addr,
			uint32(numToSkip), uint32(numRequested))
		numSkipped += mpSkipped
		for _, tx := range mpTxns {
			addressTxns = append(addressTxns, retrievedTx{tx: tx})
		}
	}

//如果有更多事务，则按所需顺序从数据库中提取事务
//需要。
	if len(addressTxns) < numRequested {
		err = s.cfg.DB.View(func(dbTx database.Tx) error {
			regions, dbSkipped, err := addrIndex.TxRegionsForAddress(
				dbTx, addr, uint32(numToSkip)-numSkipped,
				uint32(numRequested-len(addressTxns)), reverse)
			if err != nil {
				return err
			}

//从数据库加载原始事务字节。
			serializedTxns, err := dbTx.FetchBlockRegions(regions)
			if err != nil {
				return err
			}

//添加事务及其块的哈希
//包含在列表中。注意交易
//在此处保留序列化，因为调用方可能
//请求非详细输出，因此
//反序列化它只是为了重新序列化没有意义
//后来。
			for i, serializedTx := range serializedTxns {
				addressTxns = append(addressTxns, retrievedTx{
					txBytes: serializedTx,
					blkHash: regions[i].Hash,
				})
			}
			numSkipped += dbSkipped

			return nil
		})
		if err != nil {
			context := "Failed to load address index entries"
			return nil, internalRPCError(err.Error(), context)
		}

	}

//如果客户端没有请求反向，则最后添加来自mempool的事务
//订单和结果数量仍低于请求的数量。
	if !reverse && len(addressTxns) < numRequested {
//mempool中的事务还不在块头中，
//因此，在retieved transaction结构中的block header字段
//剩下零。
		mpTxns, mpSkipped := fetchMempoolTxnsForAddress(s, addr,
			uint32(numToSkip)-numSkipped, uint32(numRequested-
				len(addressTxns)))
		numSkipped += mpSkipped
		for _, tx := range mpTxns {
			addressTxns = append(addressTxns, retrievedTx{tx: tx})
		}
	}

//如果两个源均未产生任何结果，则从未使用过地址。
	if len(addressTxns) == 0 {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCNoTxInfo,
			Message: "No information available about address",
		}
	}

//将所有事务序列化为十六进制。
	hexTxns := make([]string, len(addressTxns))
	for i := range addressTxns {
//当检索到
//事务已序列化。
		rtx := &addressTxns[i]
		if rtx.txBytes != nil {
			hexTxns[i] = hex.EncodeToString(rtx.txBytes)
			continue
		}

//首先序列化事务，并在
//检索到的事务是反序列化的结构。
		hexTxns[i], err = messageToHex(rtx.tx.MsgTx())
		if err != nil {
			return nil, err
		}
	}

//不在详细模式下时，只需返回序列化txn的列表。
	if c.Verbose != nil && *c.Verbose == 0 {
		return hexTxns, nil
	}

//规范化提供的筛选器地址（如果有），以确保
//没有重复。
	filterAddrMap := make(map[string]struct{})
	if c.FilterAddrs != nil && len(*c.FilterAddrs) > 0 {
		for _, addr := range *c.FilterAddrs {
			filterAddrMap[addr] = struct{}{}
		}
	}

//设置了verbose标志，因此生成JSON对象并返回它。
	best := s.cfg.Chain.BestSnapshot()
	srtList := make([]btcjson.SearchRawTransactionsResult, len(addressTxns))
	for i := range addressTxns {
//需要反序列化事务，因此反序列化
//如果事务是序列化的，则检索到该事务（这将
//如果是从数据库中查找的话）。
//否则，使用现有的反序列化事务。
		rtx := &addressTxns[i]
		var mtx *wire.MsgTx
		if rtx.tx == nil {
//反序列化事务。
			mtx = new(wire.MsgTx)
			err := mtx.Deserialize(bytes.NewReader(rtx.txBytes))
			if err != nil {
				context := "Failed to deserialize transaction"
				return nil, internalRPCError(err.Error(),
					context)
			}
		} else {
			mtx = rtx.tx.MsgTx()
		}

		result := &srtList[i]
		result.Hex = hexTxns[i]
		result.Txid = mtx.TxHash().String()
		result.Vin, err = createVinListPrevOut(s, mtx, params, vinExtra,
			filterAddrMap)
		if err != nil {
			return nil, err
		}
		result.Vout = createVoutList(mtx, params, filterAddrMap)
		result.Version = mtx.Version
		result.LockTime = mtx.LockTime

//从mempool获取的事务还没有在一个块中，
//所以有条件地在这里获取块细节。这将是
//反映在最终的JSON输出中（mempool不会
//确认或封锁信息）。
		var blkHeader *wire.BlockHeader
		var blkHashStr string
		var blkHeight int32
		if blkHash := rtx.blkHash; blkHash != nil {
//从链中获取收割台。
			header, err := s.cfg.Chain.HeaderByHash(blkHash)
			if err != nil {
				return nil, &btcjson.RPCError{
					Code:    btcjson.ErrRPCBlockNotFound,
					Message: "Block not found",
				}
			}

//从链条上获取块高度。
			height, err := s.cfg.Chain.BlockHeightByHash(blkHash)
			if err != nil {
				context := "Failed to obtain block height"
				return nil, internalRPCError(err.Error(), context)
			}

			blkHeader = &header
			blkHashStr = blkHash.String()
			blkHeight = height
		}

//将块信息添加到结果中（如果有）。
		if blkHeader != nil {
//这不是一个打字错误，它们在比特币上是一样的。
//核心也是如此。
			result.Time = blkHeader.Timestamp.Unix()
			result.Blocktime = blkHeader.Timestamp.Unix()
			result.BlockHash = blkHashStr
			result.Confirmations = uint64(1 + best.Height - blkHeight)
		}
	}

	return srtList, nil
}

//handlesendrawtransaction执行sendrawtransaction命令。
func handleSendRawTransaction(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.SendRawTransactionCmd)
//反序列化并发送至TX继电器
	hexStr := c.HexTx
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}
	serializedTx, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, rpcDecodeHexError(hexStr)
	}
	var msgTx wire.MsgTx
	err = msgTx.Deserialize(bytes.NewReader(serializedTx))
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCDeserialization,
			Message: "TX decode failed: " + err.Error(),
		}
	}

//使用0表示标记的本地节点。
	tx := btcutil.NewTx(&msgTx)
	acceptedTxs, err := s.cfg.TxMemPool.ProcessTransaction(tx, false, false, 0)
	if err != nil {
//如果错误是规则错误，则表示事务
//只是被拒绝，而不是实际出了问题，
//所以记录下来。否则，确实出了点问题，
//所以把它记录为实际错误。在这两种情况下，JSON-RPC
//反序列化时返回给客户端的错误
//错误代码（匹配比特币行为）。
		if _, ok := err.(mempool.RuleError); ok {
			rpcsLog.Debugf("Rejected transaction %v: %v", tx.Hash(),
				err)
		} else {
			rpcsLog.Errorf("Failed to process transaction %v: %v",
				tx.Hash(), err)
		}
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCDeserialization,
			Message: "TX rejected: " + err.Error(),
		}
	}

//当交易被接受时，它应该是
//返回已接受事务的数组。唯一的办法就是
//如果processTransaction的API更改并且此代码为
//未正确更新，但确保条件作为保障。
//
//此外，由于向调用者返回错误，请确保
//事务将从内存池中删除。
	if len(acceptedTxs) == 0 || !acceptedTxs[0].Tx.Hash().IsEqual(tx.Hash()) {
		s.cfg.TxMemPool.RemoveTransaction(tx, true)

		errStr := fmt.Sprintf("transaction %v is not in accepted list",
			tx.Hash())
		return nil, internalRPCError(errStr, "")
	}

//生成并中继所有新接受的库存向量
//由于原始的
//认可的。
	s.cfg.ConnMgr.RelayTransactions(acceptedTxs)

//通知WebSocket和GetBlockTemplate长轮询客户端
//新接受的交易。
	s.NotifyNewTransactions(acceptedTxs)

//跟踪所有sendrawtransaction请求txn，以便
//如果他们不进入一个街区，就可以重播。
	txD := acceptedTxs[0]
	iv := wire.NewInvVect(wire.InvTypeTx, txD.Tx.Hash())
	s.cfg.ConnMgr.AddRebroadcastInventory(iv, txD)

	return tx.Hash().String(), nil
}

//handlesetgenerate实现setgenerate命令。
func handleSetGenerate(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.SetGenerateCmd)

//如果
//线程的最大数量（Goroutines用于我们的目的）是0。
//否则，根据提供的标志启用或禁用它。
	generate := c.Generate
	genProcLimit := -1
	if c.GenProcLimit != nil {
		genProcLimit = *c.GenProcLimit
	}
	if genProcLimit == 0 {
		generate = false
	}

	if !generate {
		s.cfg.CPUMiner.Stop()
	} else {
//如果没有地址支付
//已将块创建到。
		if len(cfg.miningAddrs) == 0 {
			return nil, &btcjson.RPCError{
				Code: btcjson.ErrRPCInternal.Code,
				Message: "No payment addresses specified " +
					"via --miningaddr",
			}
		}

//即使已经启动，也可以安全地调用Start。
		s.cfg.CPUMiner.SetNumWorkers(int32(genProcLimit))
		s.cfg.CPUMiner.Start()
	}
	return nil, nil
}

//handlestop执行停止命令。
func handleStop(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	select {
	case s.requestProcessShutdown <- struct{}{}:
	default:
	}
	return "btcd stopping.", nil
}

//handleSubmitBlock实现SubmitBlock命令。
func handleSubmitBlock(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.SubmitBlockCmd)

//反序列化提交的块。
	hexStr := c.HexBlock
	if len(hexStr)%2 != 0 {
		hexStr = "0" + c.HexBlock
	}
	serializedBlock, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, rpcDecodeHexError(hexStr)
	}

	block, err := btcutil.NewBlockFromBytes(serializedBlock)
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCDeserialization,
			Message: "Block decode failed: " + err.Error(),
		}
	}

//使用与来自其他块相同的规则处理此块
//节点。这将反过来像正常一样将其中继到网络。
	_, err = s.cfg.SyncMgr.SubmitBlock(block, blockchain.BFNone)
	if err != nil {
		return fmt.Sprintf("rejected: %s", err.Error()), nil
	}

	rpcsLog.Infof("Accepted block %s via submitblock", block.Hash())
	return nil, nil
}

//handleuptime执行uptime命令。
func handleUptime(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return time.Now().Unix() - s.cfg.StartupTime, nil
}

//handlevalidateAddress实现validateAddress命令。
func handleValidateAddress(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.ValidateAddressCmd)

	result := btcjson.ValidateAddressChainResult{}
	addr, err := btcutil.DecodeAddress(c.Address, s.cfg.ChainParams)
	if err != nil {
//返回isvalid的默认值（false）。
		return result, nil
	}

	result.Address = addr.EncodeAddress()
	result.IsValid = true

	return result, nil
}

func verifyChain(s *rpcServer, level, depth int32) error {
	best := s.cfg.Chain.BestSnapshot()
	finishHeight := best.Height - depth
	if finishHeight < 0 {
		finishHeight = 0
	}
	rpcsLog.Infof("Verifying chain for %d blocks at level %d",
		best.Height-finishHeight, level)

	for height := best.Height; height > finishHeight; height-- {
//0级只是查找块。
		block, err := s.cfg.Chain.BlockByHeight(height)
		if err != nil {
			rpcsLog.Errorf("Verify is unable to fetch block at "+
				"height %d: %v", height, err)
			return err
		}

//1级执行基本的链健全性检查。
		if level > 0 {
			err := blockchain.CheckBlockSanity(block,
				s.cfg.ChainParams.PowLimit, s.cfg.TimeSource)
			if err != nil {
				rpcsLog.Errorf("Verify is unable to validate "+
					"block at hash %v height %d: %v",
					block.Hash(), height, err)
				return err
			}
		}
	}
	rpcsLog.Infof("Chain verify completed successfully")

	return nil
}

//handleverifychain实现verifychain命令。
func handleVerifyChain(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.VerifyChainCmd)

	var checkLevel, checkDepth int32
	if c.CheckLevel != nil {
		checkLevel = *c.CheckLevel
	}
	if c.CheckDepth != nil {
		checkDepth = *c.CheckDepth
	}

	err := verifyChain(s, checkLevel, checkDepth)
	return err == nil, nil
}

//handleverifymessage实现verifymessage命令。
func handleVerifyMessage(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.VerifyMessageCmd)

//解码提供的地址。
	params := s.cfg.ChainParams
	addr, err := btcutil.DecodeAddress(c.Address, params)
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidAddressOrKey,
			Message: "Invalid address or key: " + err.Error(),
		}
	}

//只有p2pkh地址对签名有效。
	if _, ok := addr.(*btcutil.AddressPubKeyHash); !ok {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCType,
			Message: "Address is not a pay-to-pubkey-hash address",
		}
	}

//解码base64签名。
	sig, err := base64.StdEncoding.DecodeString(c.Signature)
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCParse.Code,
			Message: "Malformed base64 encoding: " + err.Error(),
		}
	}

//验证签名-这表明它是有效的。
//我们将它与下一个键进行比较。
	var buf bytes.Buffer
	wire.WriteVarString(&buf, 0, "Bitcoin Signed Message:\n")
	wire.WriteVarString(&buf, 0, c.Message)
	expectedMessageHash := chainhash.DoubleHashB(buf.Bytes())
	pk, wasCompressed, err := btcec.RecoverCompact(btcec.S256(), sig,
		expectedMessageHash)
	if err != nil {
//反映比特币核心行为，处理错误
//recovercompact作为无效签名。
		return false, nil
	}

//重新构造pubkey哈希。
	var serializedPK []byte
	if wasCompressed {
		serializedPK = pk.SerializeCompressed()
	} else {
		serializedPK = pk.SerializeUncompressed()
	}
	address, err := btcutil.NewAddressPubKey(serializedPK, params)
	if err != nil {
//再次镜像比特币核心行为，它处理公钥中的错误
//重建为无效签名。
		return false, nil
	}

//如果地址匹配，则返回布尔值。
	return address.EncodeAddress() == c.Address, nil
}

//handleversion执行版本命令。
//
//注意：这是从github.com/decred/dcrd导入的btcsuite扩展。
func handleVersion(s *rpcServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	result := map[string]btcjson.VersionResult{
		"btcdjsonrpcapi": {
			VersionString: jsonrpcSemverString,
			Major:         jsonrpcSemverMajor,
			Minor:         jsonrpcSemverMinor,
			Patch:         jsonrpcSemverPatch,
		},
	}
	return result, nil
}

//rpc server为链服务器提供并发安全的rpc服务器。
type rpcServer struct {
	started                int32
	shutdown               int32
	cfg                    rpcserverConfig
	authsha                [sha256.Size]byte
	limitauthsha           [sha256.Size]byte
	ntfnMgr                *wsNotificationManager
	numClients             int32
	statusLines            map[int]string
	statusLock             sync.RWMutex
	wg                     sync.WaitGroup
	gbtWorkState           *gbtWorkState
	helpCacher             *helpCacher
	requestProcessShutdown chan struct{}
	quit                   chan int
}

//httpstatusline返回响应状态行（RFC 2616第6.1节）
//对于给定的请求和响应状态代码。此功能被提升，并且
//由于未导出，因此改编自标准库HTTP服务器代码。
func (s *rpcServer) httpStatusLine(req *http.Request, code int) string {
//快速路径：
	key := code
	proto11 := req.ProtoAtLeast(1, 1)
	if !proto11 {
		key = -key
	}
	s.statusLock.RLock()
	line, ok := s.statusLines[key]
	s.statusLock.RUnlock()
	if ok {
		return line
	}

//慢路径：
	proto := "HTTP/1.0"
	if proto11 {
		proto = "HTTP/1.1"
	}
	codeStr := strconv.Itoa(code)
	text := http.StatusText(code)
	if text != "" {
		line = proto + " " + codeStr + " " + text + "\r\n"
		s.statusLock.Lock()
		s.statusLines[key] = line
		s.statusLock.Unlock()
	} else {
		text = "status code " + codeStr
		line = proto + " " + codeStr + " " + text + "\r\n"
	}

	return line
}

//writehttpResponseHeaders在
//在给定用于协议协商的请求时写入HTTP主体，头
//写，状态码和写程序。
func (s *rpcServer) writeHTTPResponseHeaders(req *http.Request, headers http.Header, code int, w io.Writer) error {
	_, err := io.WriteString(w, s.httpStatusLine(req, code))
	if err != nil {
		return err
	}

	err = headers.Write(w)
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, "\r\n")
	return err
}

//stop由server.go用于停止RPC侦听器。
func (s *rpcServer) Stop() error {
	if atomic.AddInt32(&s.shutdown, 1) != 1 {
		rpcsLog.Infof("RPC server is already in the process of shutting down")
		return nil
	}
	rpcsLog.Warnf("RPC server shutting down")
	for _, listener := range s.cfg.Listeners {
		err := listener.Close()
		if err != nil {
			rpcsLog.Errorf("Problem shutting down rpc: %v", err)
			return err
		}
	}
	s.ntfnMgr.Shutdown()
	s.ntfnMgr.WaitForShutdown()
	close(s.quit)
	s.wg.Wait()
	rpcsLog.Infof("RPC server shutdown complete")
	return nil
}

//RequestedProcessShutdown返回一个通道，当
//RPC客户端请求关闭进程。如果无法读取请求
//马上就掉下来了。
func (s *rpcServer) RequestedProcessShutdown() <-chan struct{} {
	return s.requestProcessShutdown
}

//notifyNewTransactions同时通知WebSocket和GetBlockTemplate long
//对通过的交易的客户端进行投票。应调用此函数
//每当新事务添加到mempool时。
func (s *rpcServer) NotifyNewTransactions(txns []*mempool.TxDesc) {
	for _, txD := range txns {
//通知WebSocket客户端有关mempool事务的信息。
		s.ntfnMgr.NotifyMempoolTx(txD.Tx, true)

//可能通知任何GetBlockTemplate长轮询客户端
//关于由于新事务而过时的块模板。
		s.gbtWorkState.NotifyMempoolTx(s.cfg.TxMemPool.LastUpdated())
	}
}

//limitconnections响应503服务不可用，如果
//添加另一个客户端将超过允许的最大RPC客户端数。
//
//此函数对于并发访问是安全的。
func (s *rpcServer) limitConnections(w http.ResponseWriter, remoteAddr string) bool {
	if int(atomic.LoadInt32(&s.numClients)+1) > cfg.RPCMaxClients {
		rpcsLog.Infof("Max RPC clients exceeded [%d] - "+
			"disconnecting client %s", cfg.RPCMaxClients,
			remoteAddr)
		http.Error(w, "503 Too busy.  Try again later.",
			http.StatusServiceUnavailable)
		return true
	}
	return false
}

//incrementclients在已连接的RPC客户端数中添加一个。注释
//这只适用于标准客户机。WebSocket客户端有自己的
//限制和单独跟踪。
//
//此函数对于并发访问是安全的。
func (s *rpcServer) incrementClients() {
	atomic.AddInt32(&s.numClients, 1)
}

//递减客户端从连接的RPC客户端数中减去一个。
//注意：这只适用于标准客户机。WebSocket客户端有自己的
//限制和单独跟踪。
//
//此函数对于并发访问是安全的。
func (s *rpcServer) decrementClients() {
	atomic.AddInt32(&s.numClients, -1)
}

//checkauth检查钱包提供的HTTP基本身份验证
//或HTTP请求R中的RPC客户端。如果提供的身份验证
//与预期的用户名和密码不匹配，非零错误为
//返回。
//
//这种检查是时间常数。
//
//第一个bool返回值表示auth成功（如果成功，则为true）和
//第二个bool返回值指定用户是否可以更改状态
//服务器（true）或用户是否受限制（false）。二是
//如果第一个总是错误的。
func (s *rpcServer) checkAuth(r *http.Request, require bool) (bool, bool, error) {
	authhdr := r.Header["Authorization"]
	if len(authhdr) <= 0 {
		if require {
			rpcsLog.Warnf("RPC authentication failure from %s",
				r.RemoteAddr)
			return false, false, errors.New("auth failure")
		}

		return false, false, nil
	}

	authsha := sha256.Sum256([]byte(authhdr[0]))

//首先检查有限身份验证，就像在用户有限的环境中一样
//可能会有更大的通话量
	limitcmp := subtle.ConstantTimeCompare(authsha[:], s.limitauthsha[:])
	if limitcmp == 1 {
		return true, false, nil
	}

//检查管理级身份验证
	cmp := subtle.ConstantTimeCompare(authsha[:], s.authsha[:])
	if cmp == 1 {
		return true, true, nil
	}

//请求的身份验证与任何用户都不匹配
	rpcsLog.Warnf("RPC authentication failure from %s", r.RemoteAddr)
	return false, false, errors.New("auth failure")
}

//parsedrpcmd表示已被解析为
//一个已知的具体命令以及在此期间可能发生的任何错误
//解析它。
type parsedRPCCmd struct {
	id     interface{}
	method string
	cmd    interface{}
	err    *btcjson.RPCError
}

//StandardCmdResult检查解析的命令是否为标准比特币JSON-RPC
//命令并运行相应的处理程序来回复命令。任何
//未识别或未实现的命令将返回错误
//适用于答复。
func (s *rpcServer) standardCmdResult(cmd *parsedRPCCmd, closeChan <-chan struct{}) (interface{}, error) {
	handler, ok := rpcHandlers[cmd.method]
	if ok {
		goto handled
	}
	_, ok = rpcAskWallet[cmd.method]
	if ok {
		handler = handleAskWallet
		goto handled
	}
	_, ok = rpcUnimplemented[cmd.method]
	if ok {
		handler = handleUnimplemented
		goto handled
	}
	return nil, btcjson.ErrRPCMethodNotFound
handled:

	return handler(s, cmd.cmd, closeChan)
}

//parseCmd将JSON-RPC请求对象解析为已知的具体命令。这个
//返回的parsedrpccmd结构的err字段将包含一个rpc错误，该错误
//如果命令在某些方面无效，例如
//未注册的命令或无效参数。
func parseCmd(request *btcjson.Request) *parsedRPCCmd {
	var parsedCmd parsedRPCCmd
	parsedCmd.id = request.ID
	parsedCmd.method = request.Method

	cmd, err := btcjson.UnmarshalCmd(request)
	if err != nil {
//如果错误是因为方法未注册，
//产生找不到方法的RPC错误。
		if jerr, ok := err.(btcjson.Error); ok &&
			jerr.ErrorCode == btcjson.ErrUnregisteredMethod {

			parsedCmd.err = btcjson.ErrRPCMethodNotFound
			return &parsedCmd
		}

//否则，某些类型的无效参数是
//原因，因此产生等效的RPC错误。
		parsedCmd.err = btcjson.NewRPCError(
			btcjson.ErrRPCInvalidParams.Code, err.Error())
		return &parsedCmd
	}

	parsedCmd.cmd = cmd
	return &parsedCmd
}

//createMarshalledReply返回一个新的已封送JSON-RPC响应
//传递的参数。它将自动转换不属于的错误
//根据需要将*btcjson.rpcerror类型转换为适当的类型。
func createMarshalledReply(id, result interface{}, replyErr error) ([]byte, error) {
	var jsonErr *btcjson.RPCError
	if replyErr != nil {
		if jErr, ok := replyErr.(*btcjson.RPCError); ok {
			jsonErr = jErr
		} else {
			jsonErr = internalRPCError(replyErr.Error(), "")
		}
	}

	return btcjson.MarshalResponse(id, result, jsonErr)
}

//jsonrpcread处理对rpc消息的读取和响应。
func (s *rpcServer) jsonRPCRead(w http.ResponseWriter, r *http.Request, isAdmin bool) {
	if atomic.LoadInt32(&s.shutdown) != 0 {
		return
	}

//从调用者读取并关闭JSON-RPC请求主体。
	body, err := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		errCode := http.StatusBadRequest
		http.Error(w, fmt.Sprintf("%d error reading JSON message: %v",
			errCode, err), errCode)
		return
	}

//不幸的是，HTTP服务器不提供
//更改新连接的读取截止时间并中断一次
//长轮询。但是，最初没有阅读截止日期
//连接意味着客户机可以永远连接和空闲。因此，
//从HTTP服务器劫持Connecton，清除读取截止时间，
//手动编写响应。
	hj, ok := w.(http.Hijacker)
	if !ok {
		errMsg := "webserver doesn't support hijacking"
		rpcsLog.Warnf(errMsg)
		errCode := http.StatusInternalServerError
		http.Error(w, strconv.Itoa(errCode)+" "+errMsg, errCode)
		return
	}
	conn, buf, err := hj.Hijack()
	if err != nil {
		rpcsLog.Warnf("Failed to hijack HTTP connection: %v", err)
		errCode := http.StatusInternalServerError
		http.Error(w, strconv.Itoa(errCode)+" "+err.Error(), errCode)
		return
	}
	defer conn.Close()
	defer buf.Flush()
	conn.SetReadDeadline(timeZeroVal)

//尝试将原始主体解析为JSON-RPC请求。
	var responseID interface{}
	var jsonErr error
	var result interface{}
	var request btcjson.Request
	if err := json.Unmarshal(body, &request); err != nil {
		jsonErr = &btcjson.RPCError{
			Code:    btcjson.ErrRPCParse.Code,
			Message: "Failed to parse request: " + err.Error(),
		}
	}
	if jsonErr == nil {
//json-rpc 1.0规范定义通知必须具有其“id”
//设置为空并声明通知没有响应。
//
//json-rpc 2.0通知是带有“json-rpc”：“2.0”的请求，并且
//没有“id”成员。规范声明通知
//不能响应。json-rpc 2.0允许空值作为
//有效的请求ID，因此此类请求不是通知。
//
//比特币核心以“id”服务请求：空甚至缺少“id”，
//并以“id”响应这些请求：响应中为空。
//
//btcd不响应没有和“id”或“id”的任何请求：空，
//无论指定的JSON-RPC协议版本如何，除非RPC要求
//启用。启用rpc quirk后，将响应此类请求
//如果request不指示json-rpc版本，则返回。
//
//用户可以启用rpc-quirk以避免兼容性问题
//软件依赖核心的行为。
		if request.ID == nil && !(cfg.RPCQuirks && request.Jsonrpc == "") {
			return
		}

//分析至少成功到有一个ID，所以
//设置为响应。
		responseID = request.ID

//设置关闭通知程序。既然连接被劫持，
//响应写入程序上的CloseNotifer不可用。
		closeChan := make(chan struct{}, 1)
		go func() {
			_, err := conn.Read(make([]byte, 1))
			if err != nil {
				close(closeChan)
			}
		}()

//检查用户是否受到限制，如果方法未经授权，则设置错误
		if !isAdmin {
			if _, ok := rpcLimited[request.Method]; !ok {
				jsonErr = &btcjson.RPCError{
					Code:    btcjson.ErrRPCInvalidParams.Code,
					Message: "limited user not authorized for this method",
				}
			}
		}

		if jsonErr == nil {
//尝试将JSON-RPC请求解析为已知的具体请求
//命令。
			parsedCmd := parseCmd(&request)
			if parsedCmd.err != nil {
				jsonErr = parsedCmd.err
			} else {
				result, jsonErr = s.standardCmdResult(parsedCmd, closeChan)
			}
		}
	}

//整理响应。
	msg, err := createMarshalledReply(responseID, result, jsonErr)
	if err != nil {
		rpcsLog.Errorf("Failed to marshal reply: %v", err)
		return
	}

//写下回答。
	err = s.writeHTTPResponseHeaders(r, w.Header(), http.StatusOK, buf)
	if err != nil {
		rpcsLog.Error(err)
		return
	}
	if _, err := buf.Write(msg); err != nil {
		rpcsLog.Errorf("Failed to write marshalled reply: %v", err)
	}

//以换行方式终止，以保持与比特币核心的兼容性。
	if err := buf.WriteByte('\n'); err != nil {
		rpcsLog.Errorf("Failed to append terminating newline to reply: %v", err)
	}
}

//如果HTTP认证被拒绝，jsonAuthfail将向客户机发送一条消息。
func jsonAuthFail(w http.ResponseWriter) {
	w.Header().Add("WWW-Authenticate", `Basic realm="btcd RPC"`)
	http.Error(w, "401 Unauthorized.", http.StatusUnauthorized)
}

//start由server.go使用以启动RPC侦听器。
func (s *rpcServer) Start() {
	if atomic.AddInt32(&s.started, 1) != 1 {
		return
	}

	rpcsLog.Trace("Starting RPC server")
	rpcServeMux := http.NewServeMux()
	httpServer := &http.Server{
		Handler: rpcServeMux,

//无法完成初始连接的超时连接
//在允许的时间范围内握手。
		ReadTimeout: time.Second * rpcAuthTimeoutSeconds,
	}
	rpcServeMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "close")
		w.Header().Set("Content-Type", "application/json")
		r.Close = true

//将连接数限制为允许的最大值。
		if s.limitConnections(w, r.RemoteAddr) {
			return
		}

//跟踪已连接客户端的数量。
		s.incrementClients()
		defer s.decrementClients()
		_, isAdmin, err := s.checkAuth(r, true)
		if err != nil {
			jsonAuthFail(w)
			return
		}

//阅读并响应请求。
		s.jsonRPCRead(w, r, isAdmin)
	})

//WebSocket终结点。
	rpcServeMux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		authenticated, isAdmin, err := s.checkAuth(r, false)
		if err != nil {
			jsonAuthFail(w)
			return
		}

//尝试将连接升级到WebSocket连接
//使用读/写缓冲区的默认大小。
		ws, err := websocket.Upgrade(w, r, nil, 0, 0)
		if err != nil {
			if _, ok := err.(websocket.HandshakeError); !ok {
				rpcsLog.Errorf("Unexpected websocket error: %v",
					err)
			}
			http.Error(w, "400 Bad Request.", http.StatusBadRequest)
			return
		}
		s.WebsocketHandler(ws, r.RemoteAddr, authenticated, isAdmin)
	})

	for _, listener := range s.cfg.Listeners {
		s.wg.Add(1)
		go func(listener net.Listener) {
			rpcsLog.Infof("RPC server listening on %s", listener.Addr())
			httpServer.Serve(listener)
			rpcsLog.Tracef("RPC listener done for %s", listener.Addr())
			s.wg.Done()
		}(listener)
	}

	s.ntfnMgr.Start()
}

//gencertpair生成指向所提供路径的密钥/证书对。
func genCertPair(certFile, keyFile string) error {
	rpcsLog.Infof("Generating TLS certificates...")

	org := "btcd autogenerated cert"
	validUntil := time.Now().Add(10 * 365 * 24 * time.Hour)
	cert, key, err := btcutil.NewTLSCertPair(org, validUntil, nil)
	if err != nil {
		return err
	}

//编写证书和密钥文件。
	if err = ioutil.WriteFile(certFile, cert, 0666); err != nil {
		return err
	}
	if err = ioutil.WriteFile(keyFile, key, 0600); err != nil {
		os.Remove(certFile)
		return err
	}

	rpcsLog.Infof("Done generating TLS certificates")
	return nil
}

//rpc server peer表示与rpc服务器一起使用的对等机。
//
//接口合同要求所有这些方法对于
//并发访问。
type rpcserverPeer interface {
//topeer返回基础对等实例。
	ToPeer() *peer.Peer

//ISTXRelayDisabled返回对等机是否已禁用
//事务中继。
	IsTxRelayDisabled() bool

//BanScore返回当前整数值，该整数值表示
//同龄人将被禁止。
	BanScore() uint32

//feefilter返回请求的当前最低费率，其中
//交易应当公布。
	FeeFilter() int64
}

//rpcserverconmanager表示与rpc一起使用的连接管理器
//服务器。
//
//接口合同要求所有这些方法对于
//并发访问。
type rpcserverConnManager interface {
//Connect将提供的地址添加为新的出站对等机。这个
//永久标志指示是否使对等机持久化
//如果连接断开，重新连接。正在尝试连接到
//已存在的对等将返回一个错误。
	Connect(addr string, permanent bool) error

//removeByID从
//持久对等的列表。正在尝试删除一个没有
//exist将返回一个错误。
	RemoveByID(id int32) error

//removebyaddr删除与提供的地址关联的对等机
//从持久对等的列表中。正在尝试删除地址
//不存在将返回错误。
	RemoveByAddr(addr string) error

//disconnectByID断开与提供的ID关联的对等机。
//这适用于入站和出站对等机。试图
//删除不存在的ID将返回错误。
	DisconnectByID(id int32) error

//disconnectbyaddr断开与提供的
//地址。这适用于入站和出站对等机。
//尝试删除不存在的地址将返回
//错误。
	DisconnectByAddr(addr string) error

//ConnectedCount返回当前连接的对等数。
	ConnectedCount() int32

//Nettotals返回通过
//所有对等网络。
	NetTotals() (uint64, uint64)

//ConnectedPeers返回一个由所有连接的对等方组成的数组。
	ConnectedPeers() []rpcserverPeer

//PersistentPeers返回一个由所有Persistent
//同龄人。
	PersistentPeers() []rpcserverPeer

//Broadcastmessage将提供的消息发送到当前所有
//互联对等。
	BroadcastMessage(msg wire.Message)

//addrebroadcastinventory将提供的清单添加到
//在库存出现之前，应随机重新分配库存。
//在一个街区。
	AddRebroadcastInventory(iv *wire.InvVect, data interface{})

//RelayTransactions为所有
//传递给所有连接的对等方的事务。
	RelayTransactions(txns []*mempool.TxDesc)
}

//rpcserverSyncManager表示用于RPC服务器的同步管理器。
//
//接口合同要求所有这些方法对于
//并发访问。
type rpcserverSyncManager interface {
//iscurrent返回同步管理器是否相信链
//与网络的其他部分相比是最新的。
	IsCurrent() bool

//SubmitBlock在以下时间之后将提供的块提交到网络
//本地处理。
	SubmitBlock(block *btcutil.Block, flags blockchain.BehaviorFlags) (bool, error)

//暂停暂停同步管理器，直到返回的通道关闭。
	Pause() chan<- struct{}

//syncpeerid返回当前对等机的ID
//用于从或0同步（如果没有）。
	SyncPeerID() int32

//locateheaders返回在第一个已知的
//在提供的定位器中阻塞，直到提供的停止哈希或
//达到当前提示，最多可达Wire.MaxBlockHeadersPermsg
//散列。
	LocateHeaders(locators []*chainhash.Hash, hashStop *chainhash.Hash) []wire.BlockHeader
}

//rpcserverconfig是包含rpc服务器配置的描述符。
type rpcserverConfig struct {
//侦听器定义一个侦听器切片，rpc服务器将为其定义
//拥有并接受连接。因为RPC服务器
//这些侦听器的所有权，当RPC服务器
//停止。
	Listeners []net.Listener

//startuptime是托管服务器的Unix时间戳。
//RPC服务器已启动。
	StartupTime int64

//connmgr定义要使用的RPC服务器的连接管理器。它
//为RPC服务器提供执行诸如添加、
//删除、连接、断开连接和查询对等端以及其他对等端
//连接相关数据和任务。
	ConnMgr rpcserverConnManager

//syncmgr定义要使用的RPC服务器的同步管理器。
	SyncMgr rpcserverSyncManager

//这些字段允许RPC服务器与本地块交互。
//链数据和状态。
	TimeSource  blockchain.MedianTimeSource
	Chain       *blockchain.BlockChain
	ChainParams *chaincfg.Params
	DB          database.DB

//TXMEMPOOL定义要与之交互的事务内存池。
	TxMemPool *mempool.TxPool

//这些字段允许RPC服务器与挖掘交互。
//
//生成器生成块模板，cpuminer使用
//CPU。CPU挖掘通常仅在测试时有用
//进行回归或模拟测试。
	Generator *mining.BlkTmplGenerator
	CPUMiner  *cpuminer.CPUMiner

//这些字段定义RPC服务器可以使用的任何可选索引
//，以便在查询时提供其他数据。
	TxIndex   *indexers.TxIndex
	AddrIndex *indexers.AddrIndex
	CfIndex   *indexers.CfIndex

//费用估算器跟踪交易的剩余时间。
//在他们被开采成块之前。
	FeeEstimator *mempool.FeeEstimator
}

//new rpcserver返回rpcserver结构的新实例。
func newRPCServer(config *rpcserverConfig) (*rpcServer, error) {
	rpc := rpcServer{
		cfg:                    *config,
		statusLines:            make(map[int]string),
		gbtWorkState:           newGbtWorkState(config.TimeSource),
		helpCacher:             newHelpCacher(),
		requestProcessShutdown: make(chan struct{}),
		quit: make(chan int),
	}
	if cfg.RPCUser != "" && cfg.RPCPass != "" {
		login := cfg.RPCUser + ":" + cfg.RPCPass
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
		rpc.authsha = sha256.Sum256([]byte(auth))
	}
	if cfg.RPCLimitUser != "" && cfg.RPCLimitPass != "" {
		login := cfg.RPCLimitUser + ":" + cfg.RPCLimitPass
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
		rpc.limitauthsha = sha256.Sum256([]byte(auth))
	}
	rpc.ntfnMgr = newWsNotificationManager(&rpc)
	rpc.cfg.Chain.Subscribe(rpc.handleBlockchainNotification)

	return &rpc, nil
}

//从区块链回调通知。它通知客户
//长时间轮询更改或订阅WebSockets通知。
func (s *rpcServer) handleBlockchainNotification(notification *blockchain.Notification) {
	switch notification.Type {
	case blockchain.NTBlockAccepted:
		block, ok := notification.Data.(*btcutil.Block)
		if !ok {
			rpcsLog.Warnf("Chain accepted notification is not a block.")
			break
		}

//允许任何客户端通过
//当新块导致
//它们的旧块模板将过时。
		s.gbtWorkState.NotifyBlockConnected(block.Hash())

	case blockchain.NTBlockConnected:
		block, ok := notification.Data.(*btcutil.Block)
		if !ok {
			rpcsLog.Warnf("Chain connected notification is not a block.")
			break
		}

//通知已注册的WebSocket客户端传入块。
		s.ntfnMgr.NotifyBlockConnected(block)

	case blockchain.NTBlockDisconnected:
		block, ok := notification.Data.(*btcutil.Block)
		if !ok {
			rpcsLog.Warnf("Chain disconnected notification is not a block.")
			break
		}

//通知已注册的WebSocket客户端。
		s.ntfnMgr.NotifyBlockDisconnected(block)
	}
}

func init() {
	rpcHandlers = rpcHandlersBeforeInit
	rand.Seed(time.Now().UnixNano())
}
