
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

//注意：此文件用于存放受支持的rpc命令
//一个链式服务器。

package btcjson

import (
	"encoding/json"
	"fmt"

	"github.com/btcsuite/btcd/wire"
)

//addnodesubcmd定义在addnode json-rpc命令中用于
//子命令字段。
type AddNodeSubCmd string

const (
//
//同龄人。
	ANAdd AddNodeSubCmd = "add"

//anremove表示应删除指定的对等机。
	ANRemove AddNodeSubCmd = "remove"

//Anonetry表示指定主机应尝试连接一次，
//
	ANOneTry AddNodeSubCmd = "onetry"
)

//addnodeCmd定义addnode json-rpc命令。
type AddNodeCmd struct {
	Addr   string
	SubCmd AddNodeSubCmd `jsonrpcusage:"\"add|remove|onetry\""`
}

//newaddnodeCmd返回可用于发出addnode的新实例
//json-rpc命令。
func NewAddNodeCmd(addr string, subCmd AddNodeSubCmd) *AddNodeCmd {
	return &AddNodeCmd{
		Addr:   addr,
		SubCmd: subCmd,
	}
}

//TransactionInput表示事务的输入。具体地说
//事务哈希和输出编号对。
type TransactionInput struct {
	Txid string `json:"txid"`
	Vout uint32 `json:"vout"`
}

//createrawtransactionCmd定义createrawtransaction json-rpc命令。
type CreateRawTransactionCmd struct {
	Inputs   []TransactionInput
Amounts  map[string]float64 `jsonrpcusage:"{\"address\":amount,...}"` //在BTC中
	LockTime *int64
}

//newcreaterawtransactionCmd返回可用于发出的新实例
//createrawtransaction json-rpc命令。
//
//金额以BTC为单位。
func NewCreateRawTransactionCmd(inputs []TransactionInput, amounts map[string]float64,
	lockTime *int64) *CreateRawTransactionCmd {

	return &CreateRawTransactionCmd{
		Inputs:   inputs,
		Amounts:  amounts,
		LockTime: lockTime,
	}
}

//decoderawtransactionCmd定义decoderawtransaction json-rpc命令。
type DecodeRawTransactionCmd struct {
	HexTx string
}

//
//decoderawtransaction json-rpc命令。
func NewDecodeRawTransactionCmd(hexTx string) *DecodeRawTransactionCmd {
	return &DecodeRawTransactionCmd{
		HexTx: hexTx,
	}
}

//decodescriptcmd定义decodescript json-rpc命令。
type DecodeScriptCmd struct {
	HexScript string
}

//newdecodescriptcmd返回可用于发出
//解码脚本json-rpc命令。
func NewDecodeScriptCmd(hexScript string) *DecodeScriptCmd {
	return &DecodeScriptCmd{
		HexScript: hexScript,
	}
}

//getaddednodeinfocmd定义getaddednodeinfo json-rpc命令。
type GetAddedNodeInfoCmd struct {
	DNS  bool
	Node *string
}

//newgetaddednodeinfocmd返回一个可用于发出
//getaddednodeinfo json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewGetAddedNodeInfoCmd(dns bool, node *string) *GetAddedNodeInfoCmd {
	return &GetAddedNodeInfoCmd{
		DNS:  dns,
		Node: node,
	}
}

//getbestblockhashcmd定义getbestblockhash json-rpc命令。
type GetBestBlockHashCmd struct{}

//newgetbestblockhashcmd返回可用于发出
//getbestblockhash json-rpc命令。
func NewGetBestBlockHashCmd() *GetBestBlockHashCmd {
	return &GetBestBlockHashCmd{}
}

//getblockCmd定义getblock json-rpc命令。
type GetBlockCmd struct {
	Hash      string
	Verbose   *bool `jsonrpcdefault:"true"`
	VerboseTx *bool `jsonrpcdefault:"false"`
}

//newgetblockCmd返回可用于发出getblock的新实例
//json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewGetBlockCmd(hash string, verbose, verboseTx *bool) *GetBlockCmd {
	return &GetBlockCmd{
		Hash:      hash,
		Verbose:   verbose,
		VerboseTx: verboseTx,
	}
}

//getblockchaininfocmd定义getblockchaininfo json-rpc命令。
type GetBlockChainInfoCmd struct{}

//newgetblockchaininfocmd返回可用于发出
//getblockchaininfo json-rpc命令。
func NewGetBlockChainInfoCmd() *GetBlockChainInfoCmd {
	return &GetBlockChainInfoCmd{}
}

//getblockcountcmd定义getblockcount json-rpc命令。
type GetBlockCountCmd struct{}

//newgetblockcountcmd返回一个可用于发出
//getblockcount json-rpc命令。
func NewGetBlockCountCmd() *GetBlockCountCmd {
	return &GetBlockCountCmd{}
}

//getblockhashcmd定义getblockhash json-rpc命令。
type GetBlockHashCmd struct {
	Index int64
}

//newgetblockhashcmd返回可用于发出
//getblockhash json-rpc命令。
func NewGetBlockHashCmd(index int64) *GetBlockHashCmd {
	return &GetBlockHashCmd{
		Index: index,
	}
}

//getblockheaderCmd定义getblockheader json-rpc命令。
type GetBlockHeaderCmd struct {
	Hash    string
	Verbose *bool `jsonrpcdefault:"true"`
}

//NewGetBlockHeaderCmd返回可用于发出
//getblockheader json-rpc命令。
func NewGetBlockHeaderCmd(hash string, verbose *bool) *GetBlockHeaderCmd {
	return &GetBlockHeaderCmd{
		Hash:    hash,
		Verbose: verbose,
	}
}

//templateRequest是一个在bip22中定义的请求对象
//（https://en.bitcoin.it/wiki/bip0022），可选作为
//指向getBlockTemplateCmd的指针参数。
type TemplateRequest struct {
	Mode         string   `json:"mode,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`

//可选长轮询。
	LongPollID string `json:"longpollid,omitempty"`

//可选模板调整。sigoplimit和sizelimit可以是int64
//或布尔。
	SigOpLimit interface{} `json:"sigoplimit,omitempty"`
	SizeLimit  interface{} `json:"sizelimit,omitempty"`
	MaxVersion uint32      `json:"maxversion,omitempty"`

//BIP 0023的基本池扩展。
	Target string `json:"target,omitempty"`

//来自BIP 0023的封锁提案。仅当模式为
//“提案”。
	Data   string `json:"data,omitempty"`
	WorkID string `json:"workid,omitempty"`
}

//ConvertTemplateRequestField可能将提供的值转换为
//需要。
func convertTemplateRequestField(fieldName string, iface interface{}) (interface{}, error) {
	switch val := iface.(type) {
	case nil:
		return nil, nil
	case bool:
		return val, nil
	case float64:
		if val == float64(int64(val)) {
			return int64(val), nil
		}
	}

	str := fmt.Sprintf("the %s field must be unspecified, a boolean, or "+
		"a 64-bit integer", fieldName)
	return nil, makeError(ErrInvalidType, str)
}

//Unmarshaljson为TemplateRequest提供了一个自定义的Unmarshal方法。这个
//是必需的，因为sigoplimit和sizelimit字段只能是特定的
//类型。
func (t *TemplateRequest) UnmarshalJSON(data []byte) error {
	type templateRequest TemplateRequest

	request := (*templateRequest)(t)
	if err := json.Unmarshal(data, &request); err != nil {
		return err
	}

//sigoplimit字段只能是nil、bool或int64。
	val, err := convertTemplateRequestField("sigoplimit", request.SigOpLimit)
	if err != nil {
		return err
	}
	request.SigOpLimit = val

//sizeLimit字段只能是nil、bool或int64。
	val, err = convertTemplateRequestField("sizelimit", request.SizeLimit)
	if err != nil {
		return err
	}
	request.SizeLimit = val

	return nil
}

//getblocktemplateCmd定义getblocktemplate json-rpc命令。
type GetBlockTemplateCmd struct {
	Request *TemplateRequest
}

//NewGetBlockTemplateCmd返回一个可用于发出
//getblocktemplate json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewGetBlockTemplateCmd(request *TemplateRequest) *GetBlockTemplateCmd {
	return &GetBlockTemplateCmd{
		Request: request,
	}
}

//
type GetCFilterCmd struct {
	Hash       string
	FilterType wire.FilterType
}

//newgetfilterCmd返回可用于发出
//getcpilter json-rpc命令。
func NewGetCFilterCmd(hash string, filterType wire.FilterType) *GetCFilterCmd {
	return &GetCFilterCmd{
		Hash:       hash,
		FilterType: filterType,
	}
}

//getfilterheaderCmd定义getfilterheader json-rpc命令。
type GetCFilterHeaderCmd struct {
	Hash       string
	FilterType wire.FilterType
}

//newgetfilterheaderCmd返回可用于发出
//GETCFIRTHORKER JSON-RPC命令。
func NewGetCFilterHeaderCmd(hash string,
	filterType wire.FilterType) *GetCFilterHeaderCmd {
	return &GetCFilterHeaderCmd{
		Hash:       hash,
		FilterType: filterType,
	}
}

//getchaintipscmd定义getchaintips json-rpc命令。
type GetChainTipsCmd struct{}

//newgetchaintipscmd返回可用于发出
//getchaintips json-rpc命令。
func NewGetChainTipsCmd() *GetChainTipsCmd {
	return &GetChainTipsCmd{}
}

//getConnectionCountCmd定义getConnectionCount json-rpc命令。
type GetConnectionCountCmd struct{}

//newgetconnectioncountcmd返回可用于发出
//getconnectioncount json-rpc命令。
func NewGetConnectionCountCmd() *GetConnectionCountCmd {
	return &GetConnectionCountCmd{}
}

//getdifficulticmd定义getdifficulticjson-rpc命令。
type GetDifficultyCmd struct{}

//NewGetDifficultyCmd返回可用于发出
//getdifficulty json-rpc命令。
func NewGetDifficultyCmd() *GetDifficultyCmd {
	return &GetDifficultyCmd{}
}

//getgeneratecmd定义getgenerate json-rpc命令。
type GetGenerateCmd struct{}

//NewGetGenerateCmd返回一个可用于发出
//getgenerate json-rpc命令。
func NewGetGenerateCmd() *GetGenerateCmd {
	return &GetGenerateCmd{}
}

//gethashespersecmd定义gethashespersec json-rpc命令。
type GetHashesPerSecCmd struct{}

//newgethashespersecmd返回可用于发出
//gethashespersec json-rpc命令。
func NewGetHashesPerSecCmd() *GetHashesPerSecCmd {
	return &GetHashesPerSecCmd{}
}

//getinfocmd定义getinfo json-rpc命令。
type GetInfoCmd struct{}

//newgetinfocmd返回可用于发出
//getinfo json-rpc命令。
func NewGetInfoCmd() *GetInfoCmd {
	return &GetInfoCmd{}
}

//getmempoolentrycmd定义getmempoolentry json-rpc命令。
type GetMempoolEntryCmd struct {
	TxID string
}

//newgetmempoolentrycmd返回可用于发出
//getmempoolentry json-rpc命令。
func NewGetMempoolEntryCmd(txHash string) *GetMempoolEntryCmd {
	return &GetMempoolEntryCmd{
		TxID: txHash,
	}
}

//getmempoolinfocmd定义getmempoolinfo json-rpc命令。
type GetMempoolInfoCmd struct{}

//newgetmempoolinfocmd返回可用于发出
//getmempool json-rpc命令。
func NewGetMempoolInfoCmd() *GetMempoolInfoCmd {
	return &GetMempoolInfoCmd{}
}

//getmininginfocmd定义getmininginfo json-rpc命令。
type GetMiningInfoCmd struct{}

//newgetmininginfocmd返回可用于发出
//
func NewGetMiningInfoCmd() *GetMiningInfoCmd {
	return &GetMiningInfoCmd{}
}

//getnetworkinfocmd定义getnetworkinfo json-rpc命令。
type GetNetworkInfoCmd struct{}

//newgetnetworkinfocmd返回可用于发出
//
func NewGetNetworkInfoCmd() *GetNetworkInfoCmd {
	return &GetNetworkInfoCmd{}
}

//getnettotalscmd定义getnettotals json-rpc命令。
type GetNetTotalsCmd struct{}

//newgetnettotalscmd返回可用于发出
//getnettotals json-rpc命令。
func NewGetNetTotalsCmd() *GetNetTotalsCmd {
	return &GetNetTotalsCmd{}
}

//GETNETWORKHASPSCMD定义了GETNETWORKHASPS JSON-RPC命令。
type GetNetworkHashPSCmd struct {
	Blocks *int `jsonrpcdefault:"120"`
	Height *int `jsonrpcdefault:"-1"`
}

//newgetnetworkhashpscmd返回可用于发出
//getnetworkhashps JSON-RPC command.
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewGetNetworkHashPSCmd(numBlocks, height *int) *GetNetworkHashPSCmd {
	return &GetNetworkHashPSCmd{
		Blocks: numBlocks,
		Height: height,
	}
}

//getpeerinfocmd定义getpeerinfo json-rpc命令。
type GetPeerInfoCmd struct{}

//newgetpeerinfocmd返回可用于发出getpeer的新实例
//json-rpc命令。
func NewGetPeerInfoCmd() *GetPeerInfoCmd {
	return &GetPeerInfoCmd{}
}

//getrawmumpoolcmd定义getmempool json-rpc命令。
type GetRawMempoolCmd struct {
	Verbose *bool `jsonrpcdefault:"false"`
}

//newgetrawmumpoolcmd返回可用于发出
//getrawmempool json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewGetRawMempoolCmd(verbose *bool) *GetRawMempoolCmd {
	return &GetRawMempoolCmd{
		Verbose: verbose,
	}
}

//getrawtransactionCmd定义getrawtransaction json-rpc命令。
//
//注意：此字段是int与bool，以保持与比特币兼容。
//核心，即使它真的应该是一个bool。
type GetRawTransactionCmd struct {
	Txid    string
	Verbose *int `jsonrpcdefault:"0"`
}

//newgetrawtransactionCmd返回可用于发出
//getrawtransaction json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewGetRawTransactionCmd(txHash string, verbose *int) *GetRawTransactionCmd {
	return &GetRawTransactionCmd{
		Txid:    txHash,
		Verbose: verbose,
	}
}

//gettXoutCmd定义gettXout json-rpc命令。
type GetTxOutCmd struct {
	Txid           string
	Vout           uint32
	IncludeMempool *bool `jsonrpcdefault:"true"`
}

//newgettXoutCmd返回一个可用于发出gettXout的新实例
//
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewGetTxOutCmd(txHash string, vout uint32, includeMempool *bool) *GetTxOutCmd {
	return &GetTxOutCmd{
		Txid:           txHash,
		Vout:           vout,
		IncludeMempool: includeMempool,
	}
}

//gettxoutproofCmd定义gettxoutproof json-rpc命令。
type GetTxOutProofCmd struct {
	TxIDs     []string
	BlockHash *string
}

//newgettXoutProofCmd返回一个可用于发出
//gettxoutproof json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewGetTxOutProofCmd(txIDs []string, blockHash *string) *GetTxOutProofCmd {
	return &GetTxOutProofCmd{
		TxIDs:     txIDs,
		BlockHash: blockHash,
	}
}

//gettxoutsetinfocmd定义gettxoutsetinfo json-rpc命令。
type GetTxOutSetInfoCmd struct{}

//newgettxoutsetinfocmd返回可用于发出
//gettxoutsetinfo json-rpc命令。
func NewGetTxOutSetInfoCmd() *GetTxOutSetInfoCmd {
	return &GetTxOutSetInfoCmd{}
}

//getworkCmd定义getwork json-rpc命令。
type GetWorkCmd struct {
	Data *string
}

//newgetworkCmd返回可用于发出getwork的新实例
//json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewGetWorkCmd(data *string) *GetWorkCmd {
	return &GetWorkCmd{
		Data: data,
	}
}

//helpcmd定义帮助json-rpc命令。
type HelpCmd struct {
	Command *string
}

//newhelpcmd返回一个可用于发出帮助json-rpc的新实例
//命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewHelpCmd(command *string) *HelpCmd {
	return &HelpCmd{
		Command: command,
	}
}

//invalidBlockCmd定义invalidBlock json-rpc命令。
type InvalidateBlockCmd struct {
	BlockHash string
}

//newinvalidBlockCmd返回可用于发出
//invalidblock json-rpc命令。
func NewInvalidateBlockCmd(blockHash string) *InvalidateBlockCmd {
	return &InvalidateBlockCmd{
		BlockHash: blockHash,
	}
}

//pingcmd定义ping json-rpc命令。
type PingCmd struct{}

//newpingCmd返回一个新实例，可用于发出ping json-rpc
//命令。
func NewPingCmd() *PingCmd {
	return &PingCmd{}
}

//preciousblockCmd定义preciousblock json-rpc命令。
type PreciousBlockCmd struct {
	BlockHash string
}

//
//preciousblock json-rpc命令。
func NewPreciousBlockCmd(blockHash string) *PreciousBlockCmd {
	return &PreciousBlockCmd{
		BlockHash: blockHash,
	}
}

//reconsiderblockCmd定义reconsiderblock json-rpc命令。
type ReconsiderBlockCmd struct {
	BlockHash string
}

//newreconsiderBlockCmd返回一个可用于发出
//重新考虑block json-rpc命令。
func NewReconsiderBlockCmd(blockHash string) *ReconsiderBlockCmd {
	return &ReconsiderBlockCmd{
		BlockHash: blockHash,
	}
}

//searchrawtransactionscmd定义searchrawtransactions json-rpc命令。
type SearchRawTransactionsCmd struct {
	Address     string
	Verbose     *int  `jsonrpcdefault:"1"`
	Skip        *int  `jsonrpcdefault:"0"`
	Count       *int  `jsonrpcdefault:"100"`
	VinExtra    *int  `jsonrpcdefault:"0"`
	Reverse     *bool `jsonrpcdefault:"false"`
	FilterAddrs *[]string
}

//newsearchrawtransactionsCmd返回可用于发出
//sendrawtransaction json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewSearchRawTransactionsCmd(address string, verbose, skip, count *int, vinExtra *int, reverse *bool, filterAddrs *[]string) *SearchRawTransactionsCmd {
	return &SearchRawTransactionsCmd{
		Address:     address,
		Verbose:     verbose,
		Skip:        skip,
		Count:       count,
		VinExtra:    vinExtra,
		Reverse:     reverse,
		FilterAddrs: filterAddrs,
	}
}

//sendrawtransactionCmd定义sendrawtransaction json-rpc命令。
type SendRawTransactionCmd struct {
	HexTx         string
	AllowHighFees *bool `jsonrpcdefault:"false"`
}

//newsendrawtransactionCmd返回可用于发出
//sendrawtransaction json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewSendRawTransactionCmd(hexTx string, allowHighFees *bool) *SendRawTransactionCmd {
	return &SendRawTransactionCmd{
		HexTx:         hexTx,
		AllowHighFees: allowHighFees,
	}
}

//setgeneratecmd定义setgenerate json-rpc命令。
type SetGenerateCmd struct {
	Generate     bool
	GenProcLimit *int `jsonrpcdefault:"-1"`
}

//NewsetGenerateCmd返回一个可用于发出
//setgenerate json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewSetGenerateCmd(generate bool, genProcLimit *int) *SetGenerateCmd {
	return &SetGenerateCmd{
		Generate:     generate,
		GenProcLimit: genProcLimit,
	}
}

//stopcmd定义stop json-rpc命令。
type StopCmd struct{}

//newstopcmd返回一个可用于发出stop json-rpc的新实例
//命令。
func NewStopCmd() *StopCmd {
	return &StopCmd{}
}

//SubmitBlockOptions表示随
//SubmitBlockCmd命令。
type SubmitBlockOptions struct {
//如果服务器提供了带模板的工作标识，则必须提供。
	WorkID string `json:"workid,omitempty"`
}

//submitblockCmd定义submitblock json-rpc命令。
type SubmitBlockCmd struct {
	HexBlock string
	Options  *SubmitBlockOptions
}

//NewSubmitBlockCmd返回一个可用于发出
//submitblock json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewSubmitBlockCmd(hexBlock string, options *SubmitBlockOptions) *SubmitBlockCmd {
	return &SubmitBlockCmd{
		HexBlock: hexBlock,
		Options:  options,
	}
}

//uptimeCmd定义uptime json-rpc命令。
type UptimeCmd struct{}

//newuptimeCmd返回一个可用于发出uptime json-rpc命令的新实例。
func NewUptimeCmd() *UptimeCmd {
	return &UptimeCmd{}
}

//validateadressCmd定义validateddress json-rpc命令。
type ValidateAddressCmd struct {
	Address string
}

//newvalidateAddResCmd返回可用于发出
//validateAddress json-rpc命令。
func NewValidateAddressCmd(address string) *ValidateAddressCmd {
	return &ValidateAddressCmd{
		Address: address,
	}
}

//verifychainCmd定义verifychain json-rpc命令。
type VerifyChainCmd struct {
	CheckLevel *int32 `jsonrpcdefault:"3"`
CheckDepth *int32 `jsonrpcdefault:"288"` //0 =全部
}

//newverifychainCmd返回可用于发出
//verifychain JSON-RPC command.
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewVerifyChainCmd(checkLevel, checkDepth *int32) *VerifyChainCmd {
	return &VerifyChainCmd{
		CheckLevel: checkLevel,
		CheckDepth: checkDepth,
	}
}

//
type VerifyMessageCmd struct {
	Address   string
	Signature string
	Message   string
}

//NewVerifyMessageCmd returns a new instance which can be used to issue a
//verifymessage JSON-RPC command.
func NewVerifyMessageCmd(address, signature, message string) *VerifyMessageCmd {
	return &VerifyMessageCmd{
		Address:   address,
		Signature: signature,
		Message:   message,
	}
}

//VrIFYTXOUTUBECUM CMD定义了JSON-RPC命令的VIEFIFTXOUT命令。
type VerifyTxOutProofCmd struct {
	Proof string
}

//NewVerifyTxOutProofCmd returns a new instance which can be used to issue a
//verifytxoutproof JSON-RPC command.
func NewVerifyTxOutProofCmd(proof string) *VerifyTxOutProofCmd {
	return &VerifyTxOutProofCmd{
		Proof: proof,
	}
}

func init() {
//此文件中的命令没有特殊标志。
	flags := UsageFlag(0)

	MustRegisterCmd("addnode", (*AddNodeCmd)(nil), flags)
	MustRegisterCmd("createrawtransaction", (*CreateRawTransactionCmd)(nil), flags)
	MustRegisterCmd("decoderawtransaction", (*DecodeRawTransactionCmd)(nil), flags)
	MustRegisterCmd("decodescript", (*DecodeScriptCmd)(nil), flags)
	MustRegisterCmd("getaddednodeinfo", (*GetAddedNodeInfoCmd)(nil), flags)
	MustRegisterCmd("getbestblockhash", (*GetBestBlockHashCmd)(nil), flags)
	MustRegisterCmd("getblock", (*GetBlockCmd)(nil), flags)
	MustRegisterCmd("getblockchaininfo", (*GetBlockChainInfoCmd)(nil), flags)
	MustRegisterCmd("getblockcount", (*GetBlockCountCmd)(nil), flags)
	MustRegisterCmd("getblockhash", (*GetBlockHashCmd)(nil), flags)
	MustRegisterCmd("getblockheader", (*GetBlockHeaderCmd)(nil), flags)
	MustRegisterCmd("getblocktemplate", (*GetBlockTemplateCmd)(nil), flags)
	MustRegisterCmd("getcfilter", (*GetCFilterCmd)(nil), flags)
	MustRegisterCmd("getcfilterheader", (*GetCFilterHeaderCmd)(nil), flags)
	MustRegisterCmd("getchaintips", (*GetChainTipsCmd)(nil), flags)
	MustRegisterCmd("getconnectioncount", (*GetConnectionCountCmd)(nil), flags)
	MustRegisterCmd("getdifficulty", (*GetDifficultyCmd)(nil), flags)
	MustRegisterCmd("getgenerate", (*GetGenerateCmd)(nil), flags)
	MustRegisterCmd("gethashespersec", (*GetHashesPerSecCmd)(nil), flags)
	MustRegisterCmd("getinfo", (*GetInfoCmd)(nil), flags)
	MustRegisterCmd("getmempoolentry", (*GetMempoolEntryCmd)(nil), flags)
	MustRegisterCmd("getmempoolinfo", (*GetMempoolInfoCmd)(nil), flags)
	MustRegisterCmd("getmininginfo", (*GetMiningInfoCmd)(nil), flags)
	MustRegisterCmd("getnetworkinfo", (*GetNetworkInfoCmd)(nil), flags)
	MustRegisterCmd("getnettotals", (*GetNetTotalsCmd)(nil), flags)
	MustRegisterCmd("getnetworkhashps", (*GetNetworkHashPSCmd)(nil), flags)
	MustRegisterCmd("getpeerinfo", (*GetPeerInfoCmd)(nil), flags)
	MustRegisterCmd("getrawmempool", (*GetRawMempoolCmd)(nil), flags)
	MustRegisterCmd("getrawtransaction", (*GetRawTransactionCmd)(nil), flags)
	MustRegisterCmd("gettxout", (*GetTxOutCmd)(nil), flags)
	MustRegisterCmd("gettxoutproof", (*GetTxOutProofCmd)(nil), flags)
	MustRegisterCmd("gettxoutsetinfo", (*GetTxOutSetInfoCmd)(nil), flags)
	MustRegisterCmd("getwork", (*GetWorkCmd)(nil), flags)
	MustRegisterCmd("help", (*HelpCmd)(nil), flags)
	MustRegisterCmd("invalidateblock", (*InvalidateBlockCmd)(nil), flags)
	MustRegisterCmd("ping", (*PingCmd)(nil), flags)
	MustRegisterCmd("preciousblock", (*PreciousBlockCmd)(nil), flags)
	MustRegisterCmd("reconsiderblock", (*ReconsiderBlockCmd)(nil), flags)
	MustRegisterCmd("searchrawtransactions", (*SearchRawTransactionsCmd)(nil), flags)
	MustRegisterCmd("sendrawtransaction", (*SendRawTransactionCmd)(nil), flags)
	MustRegisterCmd("setgenerate", (*SetGenerateCmd)(nil), flags)
	MustRegisterCmd("stop", (*StopCmd)(nil), flags)
	MustRegisterCmd("submitblock", (*SubmitBlockCmd)(nil), flags)
	MustRegisterCmd("uptime", (*UptimeCmd)(nil), flags)
	MustRegisterCmd("validateaddress", (*ValidateAddressCmd)(nil), flags)
	MustRegisterCmd("verifychain", (*VerifyChainCmd)(nil), flags)
	MustRegisterCmd("verifymessage", (*VerifyMessageCmd)(nil), flags)
	MustRegisterCmd("verifytxoutproof", (*VerifyTxOutProofCmd)(nil), flags)
}
