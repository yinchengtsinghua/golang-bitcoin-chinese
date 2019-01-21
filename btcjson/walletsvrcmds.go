
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2014 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

//注意：此文件用于存放受支持的rpc命令
//钱包服务器。

package btcjson

//addmultisigaddressCmd定义addmutisigaddress json-rpc命令。
type AddMultisigAddressCmd struct {
	NRequired int
	Keys      []string
	Account   *string
}

//newaddmultisigaddressCmd返回可用于发出
//addmultisigaddress json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewAddMultisigAddressCmd(nRequired int, keys []string, account *string) *AddMultisigAddressCmd {
	return &AddMultisigAddressCmd{
		NRequired: nRequired,
		Keys:      keys,
		Account:   account,
	}
}

//addwitnessAddressCmd定义addwitnessAddress json-rpc命令。
type AddWitnessAddressCmd struct {
	Address string
}

//newaddwitnessAddressCmd返回可用于发出
//addwitnessaddress json-rpc命令。
func NewAddWitnessAddressCmd(address string) *AddWitnessAddressCmd {
	return &AddWitnessAddressCmd{
		Address: address,
	}
}

//createMultisigCmd定义createMultisig json-rpc命令。
type CreateMultisigCmd struct {
	NRequired int
	Keys      []string
}

//newcreateMultisigCmd返回可用于发出
//createMultisig json-rpc命令。
func NewCreateMultisigCmd(nRequired int, keys []string) *CreateMultisigCmd {
	return &CreateMultisigCmd{
		NRequired: nRequired,
		Keys:      keys,
	}
}

//dumpprivkeycmd定义dumpprivkey json-rpc命令。
type DumpPrivKeyCmd struct {
	Address string
}

//newdumpprivkeycmd返回可用于发出
//dumpprivkey json-rpc命令。
func NewDumpPrivKeyCmd(address string) *DumpPrivKeyCmd {
	return &DumpPrivKeyCmd{
		Address: address,
	}
}

//encryptwalletcmd定义encryptwallet json-rpc命令。
type EncryptWalletCmd struct {
	Passphrase string
}

//newencryptwalletcmd返回一个新实例，该实例可用于发出
//encryptwallet json-rpc命令。
func NewEncryptWalletCmd(passphrase string) *EncryptWalletCmd {
	return &EncryptWalletCmd{
		Passphrase: passphrase,
	}
}

//estimatefeecmd定义estimatefee json-rpc命令。
type EstimateFeeCmd struct {
	NumBlocks int64
}

//newEstimateFeCmd返回可用于发出
//estimatefee json-rpc命令。
func NewEstimateFeeCmd(numBlocks int64) *EstimateFeeCmd {
	return &EstimateFeeCmd{
		NumBlocks: numBlocks,
	}
}

//estimatepriorityCmd定义estimatepriority json-rpc命令。
type EstimatePriorityCmd struct {
	NumBlocks int64
}

//newEstimatePriorityCmd返回一个新实例，该实例可用于发出
//EstimatePriority JSON-RPC命令。
func NewEstimatePriorityCmd(numBlocks int64) *EstimatePriorityCmd {
	return &EstimatePriorityCmd{
		NumBlocks: numBlocks,
	}
}

//getaccountCmd定义getaccount json-rpc命令。
type GetAccountCmd struct {
	Address string
}

//newgetaccountCmd返回可用于发出
//getaccount json-rpc命令。
func NewGetAccountCmd(address string) *GetAccountCmd {
	return &GetAccountCmd{
		Address: address,
	}
}

//getaccountaddressCmd定义getaccountaddress json-rpc命令。
type GetAccountAddressCmd struct {
	Account string
}

//newgetaccountaddressCmd返回可用于发出
//getaccountaddress json-rpc命令。
func NewGetAccountAddressCmd(account string) *GetAccountAddressCmd {
	return &GetAccountAddressCmd{
		Account: account,
	}
}

//getaddressbyaccountCmd定义getaddressbyaccount json-rpc命令。
type GetAddressesByAccountCmd struct {
	Account string
}

//newgetaddressbyaccountcmd返回可用于发出的新实例
//getaddressbyaccount json-rpc命令。
func NewGetAddressesByAccountCmd(account string) *GetAddressesByAccountCmd {
	return &GetAddressesByAccountCmd{
		Account: account,
	}
}

//getbalancecmd定义getbalance json-rpc命令。
type GetBalanceCmd struct {
	Account *string
	MinConf *int `jsonrpcdefault:"1"`
}

//newgetbalancecmd返回可用于发出
//getbalance json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewGetBalanceCmd(account *string, minConf *int) *GetBalanceCmd {
	return &GetBalanceCmd{
		Account: account,
		MinConf: minConf,
	}
}

//getnewaddressCmd定义getnewaddress json-rpc命令。
type GetNewAddressCmd struct {
	Account *string
}

//newgetnewaddressCmd返回可用于发出
//getnewaddress json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewGetNewAddressCmd(account *string) *GetNewAddressCmd {
	return &GetNewAddressCmd{
		Account: account,
	}
}

//getrawchangeaddressCmd定义getrawchangeaddress json-rpc命令。
type GetRawChangeAddressCmd struct {
	Account *string
}

//newgetrawchangeaddressCmd返回可用于发出
//getrawchangeaddress json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewGetRawChangeAddressCmd(account *string) *GetRawChangeAddressCmd {
	return &GetRawChangeAddressCmd{
		Account: account,
	}
}

//getreceivedbyaccountCmd定义getreceivedbyaccount json-rpc命令。
type GetReceivedByAccountCmd struct {
	Account string
	MinConf *int `jsonrpcdefault:"1"`
}

//newgetreceivedbyaccountCmd返回可用于发出的新实例
//getreceivedbyaccount json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewGetReceivedByAccountCmd(account string, minConf *int) *GetReceivedByAccountCmd {
	return &GetReceivedByAccountCmd{
		Account: account,
		MinConf: minConf,
	}
}

//getreceivedbyaddressCmd定义getreceivedbyaddress json-rpc命令。
type GetReceivedByAddressCmd struct {
	Address string
	MinConf *int `jsonrpcdefault:"1"`
}

//newgetreceivedbyaddressCmd返回可用于发出的新实例
//getreceivedbyaddress json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewGetReceivedByAddressCmd(address string, minConf *int) *GetReceivedByAddressCmd {
	return &GetReceivedByAddressCmd{
		Address: address,
		MinConf: minConf,
	}
}

//gettransactioncmd定义gettransaction json-rpc命令。
type GetTransactionCmd struct {
	Txid             string
	IncludeWatchOnly *bool `jsonrpcdefault:"false"`
}

//newgetTransactionCmd返回可用于发出
//getTransaction json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewGetTransactionCmd(txHash string, includeWatchOnly *bool) *GetTransactionCmd {
	return &GetTransactionCmd{
		Txid:             txHash,
		IncludeWatchOnly: includeWatchOnly,
	}
}

//getwalletinfo命令定义getwalletinfo json-rpc命令。
type GetWalletInfoCmd struct{}

//newgetwalletinfo返回可用于发出
//getwalletinfo json-rpc命令。
func NewGetWalletInfoCmd() *GetWalletInfoCmd {
	return &GetWalletInfoCmd{}
}

//importprivkeycmd定义importprivkey json-rpc命令。
type ImportPrivKeyCmd struct {
	PrivKey string
	Label   *string
	Rescan  *bool `jsonrpcdefault:"true"`
}

//newimportprivkeycmd返回可用于发出
//importprivkey json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewImportPrivKeyCmd(privKey string, label *string, rescan *bool) *ImportPrivKeyCmd {
	return &ImportPrivKeyCmd{
		PrivKey: privKey,
		Label:   label,
		Rescan:  rescan,
	}
}

//keypoolrefillCmd定义keypoolrefill json-rpc命令。
type KeyPoolRefillCmd struct {
	NewSize *uint `jsonrpcdefault:"100"`
}

//newkeypoolrefillCmd返回可用于发出
//keypoolrefill json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewKeyPoolRefillCmd(newSize *uint) *KeyPoolRefillCmd {
	return &KeyPoolRefillCmd{
		NewSize: newSize,
	}
}

//listaccountscmd定义listaccounts json-rpc命令。
type ListAccountsCmd struct {
	MinConf *int `jsonrpcdefault:"1"`
}

//newlistaccountscmd返回可用于发出
//listaccounts json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewListAccountsCmd(minConf *int) *ListAccountsCmd {
	return &ListAccountsCmd{
		MinConf: minConf,
	}
}

//listaddressgroupingsCmd定义listaddressgroupings json-rpc命令。
type ListAddressGroupingsCmd struct{}

//NewListAddressGroupingsCmd返回可用于发出的新实例
//listaddressgroupoings json-rpc命令。
func NewListAddressGroupingsCmd() *ListAddressGroupingsCmd {
	return &ListAddressGroupingsCmd{}
}

//listlockunspentcmd定义listlockunspent json-rpc命令。
type ListLockUnspentCmd struct{}

//newlistlockunspentcmd返回可用于发出
//listlockunspent json-rpc命令。
func NewListLockUnspentCmd() *ListLockUnspentCmd {
	return &ListLockUnspentCmd{}
}

//lisreceivedbyaccountCmd定义lisreceivedbyaccount json-rpc命令。
type ListReceivedByAccountCmd struct {
	MinConf          *int  `jsonrpcdefault:"1"`
	IncludeEmpty     *bool `jsonrpcdefault:"false"`
	IncludeWatchOnly *bool `jsonrpcdefault:"false"`
}

//newlisreceivedbyaccountCmd返回可用于发出的新实例
//listreceivedbyaccount json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewListReceivedByAccountCmd(minConf *int, includeEmpty, includeWatchOnly *bool) *ListReceivedByAccountCmd {
	return &ListReceivedByAccountCmd{
		MinConf:          minConf,
		IncludeEmpty:     includeEmpty,
		IncludeWatchOnly: includeWatchOnly,
	}
}

//lisreceivedbyaddressCmd定义lisreceivedbyaddress json-rpc命令。
type ListReceivedByAddressCmd struct {
	MinConf          *int  `jsonrpcdefault:"1"`
	IncludeEmpty     *bool `jsonrpcdefault:"false"`
	IncludeWatchOnly *bool `jsonrpcdefault:"false"`
}

//newlisreceivedbyaddressCmd返回可用于发出的新实例
//listreceivedbyaddress json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewListReceivedByAddressCmd(minConf *int, includeEmpty, includeWatchOnly *bool) *ListReceivedByAddressCmd {
	return &ListReceivedByAddressCmd{
		MinConf:          minConf,
		IncludeEmpty:     includeEmpty,
		IncludeWatchOnly: includeWatchOnly,
	}
}

//listsinceblockCmd定义listsinceblock json-rpc命令。
type ListSinceBlockCmd struct {
	BlockHash           *string
	TargetConfirmations *int  `jsonrpcdefault:"1"`
	IncludeWatchOnly    *bool `jsonrpcdefault:"false"`
}

//newlistsinceblockCmd返回可用于发出
//listsincblock json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewListSinceBlockCmd(blockHash *string, targetConfirms *int, includeWatchOnly *bool) *ListSinceBlockCmd {
	return &ListSinceBlockCmd{
		BlockHash:           blockHash,
		TargetConfirmations: targetConfirms,
		IncludeWatchOnly:    includeWatchOnly,
	}
}

//listractionscmd定义listractions json-rpc命令。
type ListTransactionsCmd struct {
	Account          *string
	Count            *int  `jsonrpcdefault:"10"`
	From             *int  `jsonrpcdefault:"0"`
	IncludeWatchOnly *bool `jsonrpcdefault:"false"`
}

//newlistransactionsCmd返回可用于发出
//listtransactions json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewListTransactionsCmd(account *string, count, from *int, includeWatchOnly *bool) *ListTransactionsCmd {
	return &ListTransactionsCmd{
		Account:          account,
		Count:            count,
		From:             from,
		IncludeWatchOnly: includeWatchOnly,
	}
}

//listunspentcmd定义listunspent json-rpc命令。
type ListUnspentCmd struct {
	MinConf   *int `jsonrpcdefault:"1"`
	MaxConf   *int `jsonrpcdefault:"9999999"`
	Addresses *[]string
}

//newlistunspentcmd返回可用于发出
//listunspent json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewListUnspentCmd(minConf, maxConf *int, addresses *[]string) *ListUnspentCmd {
	return &ListUnspentCmd{
		MinConf:   minConf,
		MaxConf:   maxConf,
		Addresses: addresses,
	}
}

//lockunspentCmd定义lockunspent json-rpc命令。
type LockUnspentCmd struct {
	Unlock       bool
	Transactions []TransactionInput
}

//newlockunspentCmd返回可用于发出
//lockunspent json-rpc命令。
func NewLockUnspentCmd(unlock bool, transactions []TransactionInput) *LockUnspentCmd {
	return &LockUnspentCmd{
		Unlock:       unlock,
		Transactions: transactions,
	}
}

//moveCmd定义move json-rpc命令。
type MoveCmd struct {
	FromAccount string
	ToAccount   string
Amount      float64 //在BTC中
	MinConf     *int    `jsonrpcdefault:"1"`
	Comment     *string
}

//newmoveCmd返回一个可用于发出move json-rpc的新实例
//命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewMoveCmd(fromAccount, toAccount string, amount float64, minConf *int, comment *string) *MoveCmd {
	return &MoveCmd{
		FromAccount: fromAccount,
		ToAccount:   toAccount,
		Amount:      amount,
		MinConf:     minConf,
		Comment:     comment,
	}
}

//sendfromCmd定义sendfrom json-rpc命令。
type SendFromCmd struct {
	FromAccount string
	ToAddress   string
Amount      float64 //在BTC中
	MinConf     *int    `jsonrpcdefault:"1"`
	Comment     *string
	CommentTo   *string
}

//newsendfromCmd返回可用于发出sendfrom的新实例
//json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewSendFromCmd(fromAccount, toAddress string, amount float64, minConf *int, comment, commentTo *string) *SendFromCmd {
	return &SendFromCmd{
		FromAccount: fromAccount,
		ToAddress:   toAddress,
		Amount:      amount,
		MinConf:     minConf,
		Comment:     comment,
		CommentTo:   commentTo,
	}
}

//sendmanycmd定义sendmany json-rpc命令。
type SendManyCmd struct {
	FromAccount string
Amounts     map[string]float64 `jsonrpcusage:"{\"address\":amount,...}"` //在BTC中
	MinConf     *int               `jsonrpcdefault:"1"`
	Comment     *string
}

//newsendmanyCmd返回一个可用于发出sendmany的新实例
//json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewSendManyCmd(fromAccount string, amounts map[string]float64, minConf *int, comment *string) *SendManyCmd {
	return &SendManyCmd{
		FromAccount: fromAccount,
		Amounts:     amounts,
		MinConf:     minConf,
		Comment:     comment,
	}
}

//sendtoAddressCmd定义sendtoAddress json-rpc命令。
type SendToAddressCmd struct {
	Address   string
	Amount    float64
	Comment   *string
	CommentTo *string
}

//newsendtoAddressCmd返回可用于发出
//sendtoAddress json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewSendToAddressCmd(address string, amount float64, comment, commentTo *string) *SendToAddressCmd {
	return &SendToAddressCmd{
		Address:   address,
		Amount:    amount,
		Comment:   comment,
		CommentTo: commentTo,
	}
}

//setaccountCmd定义setaccount json-rpc命令。
type SetAccountCmd struct {
	Address string
	Account string
}

//newsetaccountCmd返回可用于发出
//setaccount json-rpc命令。
func NewSetAccountCmd(address, account string) *SetAccountCmd {
	return &SetAccountCmd{
		Address: address,
		Account: account,
	}
}

//setxfeecmd定义setxfee json-rpc命令。
type SetTxFeeCmd struct {
Amount float64 //在BTC中
}

//newsetxfeecmd返回一个可用于发出setxfee的新实例
//json-rpc命令。
func NewSetTxFeeCmd(amount float64) *SetTxFeeCmd {
	return &SetTxFeeCmd{
		Amount: amount,
	}
}

//signmessagecmd定义signmessage json-rpc命令。
type SignMessageCmd struct {
	Address string
	Message string
}

//newsignmessageCmd返回可用于发出
//signmessage json-rpc命令。
func NewSignMessageCmd(address, message string) *SignMessageCmd {
	return &SignMessageCmd{
		Address: address,
		Message: message,
	}
}

//rawtxinput为用于
//signrawtransactionCmd结构。
type RawTxInput struct {
	Txid         string `json:"txid"`
	Vout         uint32 `json:"vout"`
	ScriptPubKey string `json:"scriptPubKey"`
	RedeemScript string `json:"redeemScript"`
}

//signrawtransactionCmd定义signrawtransaction json-rpc命令。
type SignRawTransactionCmd struct {
	RawTx    string
	Inputs   *[]RawTxInput
	PrivKeys *[]string
	Flags    *string `jsonrpcdefault:"\"ALL\""`
}

//newsignrawtransactionCmd返回可用于发出
//signrawtransaction json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewSignRawTransactionCmd(hexEncodedTx string, inputs *[]RawTxInput, privKeys *[]string, flags *string) *SignRawTransactionCmd {
	return &SignRawTransactionCmd{
		RawTx:    hexEncodedTx,
		Inputs:   inputs,
		PrivKeys: privKeys,
		Flags:    flags,
	}
}

//walletlockcmd定义walletlock json-rpc命令。
type WalletLockCmd struct{}

//newwalletlockCmd返回一个可用于发出
//walletlock json-rpc命令。
func NewWalletLockCmd() *WalletLockCmd {
	return &WalletLockCmd{}
}

//walletpassphrasecmd定义walletpassphrase json-rpc命令。
type WalletPassphraseCmd struct {
	Passphrase string
	Timeout    int64
}

//newwalletpassphrasecmd返回可用于发出
//walletpassphrase json-rpc命令。
func NewWalletPassphraseCmd(passphrase string, timeout int64) *WalletPassphraseCmd {
	return &WalletPassphraseCmd{
		Passphrase: passphrase,
		Timeout:    timeout,
	}
}

//walletpassphrasechangeCmd定义walletpassphrase json-rpc命令。
type WalletPassphraseChangeCmd struct {
	OldPassphrase string
	NewPassphrase string
}

//newwalletpassphrasechangeCmd返回可用于
//发出walletpassphrasechange json-rpc命令。
func NewWalletPassphraseChangeCmd(oldPassphrase, newPassphrase string) *WalletPassphraseChangeCmd {
	return &WalletPassphraseChangeCmd{
		OldPassphrase: oldPassphrase,
		NewPassphrase: newPassphrase,
	}
}

func init() {
//此文件中的命令仅可用于钱包服务器。
	flags := UFWalletOnly

	MustRegisterCmd("addmultisigaddress", (*AddMultisigAddressCmd)(nil), flags)
	MustRegisterCmd("addwitnessaddress", (*AddWitnessAddressCmd)(nil), flags)
	MustRegisterCmd("createmultisig", (*CreateMultisigCmd)(nil), flags)
	MustRegisterCmd("dumpprivkey", (*DumpPrivKeyCmd)(nil), flags)
	MustRegisterCmd("encryptwallet", (*EncryptWalletCmd)(nil), flags)
	MustRegisterCmd("estimatefee", (*EstimateFeeCmd)(nil), flags)
	MustRegisterCmd("estimatepriority", (*EstimatePriorityCmd)(nil), flags)
	MustRegisterCmd("getaccount", (*GetAccountCmd)(nil), flags)
	MustRegisterCmd("getaccountaddress", (*GetAccountAddressCmd)(nil), flags)
	MustRegisterCmd("getaddressesbyaccount", (*GetAddressesByAccountCmd)(nil), flags)
	MustRegisterCmd("getbalance", (*GetBalanceCmd)(nil), flags)
	MustRegisterCmd("getnewaddress", (*GetNewAddressCmd)(nil), flags)
	MustRegisterCmd("getrawchangeaddress", (*GetRawChangeAddressCmd)(nil), flags)
	MustRegisterCmd("getreceivedbyaccount", (*GetReceivedByAccountCmd)(nil), flags)
	MustRegisterCmd("getreceivedbyaddress", (*GetReceivedByAddressCmd)(nil), flags)
	MustRegisterCmd("gettransaction", (*GetTransactionCmd)(nil), flags)
	MustRegisterCmd("getwalletinfo", (*GetWalletInfoCmd)(nil), flags)
	MustRegisterCmd("importprivkey", (*ImportPrivKeyCmd)(nil), flags)
	MustRegisterCmd("keypoolrefill", (*KeyPoolRefillCmd)(nil), flags)
	MustRegisterCmd("listaccounts", (*ListAccountsCmd)(nil), flags)
	MustRegisterCmd("listaddressgroupings", (*ListAddressGroupingsCmd)(nil), flags)
	MustRegisterCmd("listlockunspent", (*ListLockUnspentCmd)(nil), flags)
	MustRegisterCmd("listreceivedbyaccount", (*ListReceivedByAccountCmd)(nil), flags)
	MustRegisterCmd("listreceivedbyaddress", (*ListReceivedByAddressCmd)(nil), flags)
	MustRegisterCmd("listsinceblock", (*ListSinceBlockCmd)(nil), flags)
	MustRegisterCmd("listtransactions", (*ListTransactionsCmd)(nil), flags)
	MustRegisterCmd("listunspent", (*ListUnspentCmd)(nil), flags)
	MustRegisterCmd("lockunspent", (*LockUnspentCmd)(nil), flags)
	MustRegisterCmd("move", (*MoveCmd)(nil), flags)
	MustRegisterCmd("sendfrom", (*SendFromCmd)(nil), flags)
	MustRegisterCmd("sendmany", (*SendManyCmd)(nil), flags)
	MustRegisterCmd("sendtoaddress", (*SendToAddressCmd)(nil), flags)
	MustRegisterCmd("setaccount", (*SetAccountCmd)(nil), flags)
	MustRegisterCmd("settxfee", (*SetTxFeeCmd)(nil), flags)
	MustRegisterCmd("signmessage", (*SignMessageCmd)(nil), flags)
	MustRegisterCmd("signrawtransaction", (*SignRawTransactionCmd)(nil), flags)
	MustRegisterCmd("walletlock", (*WalletLockCmd)(nil), flags)
	MustRegisterCmd("walletpassphrase", (*WalletPassphraseCmd)(nil), flags)
	MustRegisterCmd("walletpassphrasechange", (*WalletPassphraseChangeCmd)(nil), flags)
}
