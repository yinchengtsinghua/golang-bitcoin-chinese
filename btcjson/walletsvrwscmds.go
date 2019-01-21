
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

package btcjson

//注意：此文件用于存放受支持的rpc命令
//钱包服务器，但只能通过WebSockets提供。

//createCencryptedAlletCmd定义createCencryptedAllet json-rpc命令。
type CreateEncryptedWalletCmd struct {
	Passphrase string
}

//NewCreateEncryptedAlletCmd返回可用于发出的新实例
//createncryptedwallet json-rpc命令。
func NewCreateEncryptedWalletCmd(passphrase string) *CreateEncryptedWalletCmd {
	return &CreateEncryptedWalletCmd{
		Passphrase: passphrase,
	}
}

//exportwatchingwalletCmd定义exportwatchingwallet json-rpc命令。
type ExportWatchingWalletCmd struct {
	Account  *string
	Download *bool `jsonrpcdefault:"false"`
}

//newexportwatchingwalletcmd返回可用于发出的新实例
//exportwatchingwallet json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewExportWatchingWalletCmd(account *string, download *bool) *ExportWatchingWalletCmd {
	return &ExportWatchingWalletCmd{
		Account:  account,
		Download: download,
	}
}

//getunconfirmedbalanceCmd定义getunconfirmedbalance json-rpc命令。
type GetUnconfirmedBalanceCmd struct {
	Account *string
}

//newgetunconfirmedbalancecmd返回可用于发出的新实例
//getunconfirmedbalance json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewGetUnconfirmedBalanceCmd(account *string) *GetUnconfirmedBalanceCmd {
	return &GetUnconfirmedBalanceCmd{
		Account: account,
	}
}

//listaddresstransactionsCmd定义listaddresstransactions json-rpc
//命令。
type ListAddressTransactionsCmd struct {
	Addresses []string
	Account   *string
}

//newlistaddresstransactionsCmd返回可用于
//发出listaddresstransactions json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewListAddressTransactionsCmd(addresses []string, account *string) *ListAddressTransactionsCmd {
	return &ListAddressTransactionsCmd{
		Addresses: addresses,
		Account:   account,
	}
}

//listaltransactionsCmd定义listaltransactions json-rpc命令。
type ListAllTransactionsCmd struct {
	Account *string
}

//newlistaltransactionsCmd返回可用于发出
//listaltransactions json-rpc命令。
//
//指针参数表示它们是可选的。通过零
//对于可选参数，将使用默认值。
func NewListAllTransactionsCmd(account *string) *ListAllTransactionsCmd {
	return &ListAllTransactionsCmd{
		Account: account,
	}
}

//recoveraddressescmd定义recoveraddresses json-rpc命令。
type RecoverAddressesCmd struct {
	Account string
	N       int
}

//newrecoveraddressescmd返回可用于发出
//recoveraddresses json-rpc命令。
func NewRecoverAddressesCmd(account string, n int) *RecoverAddressesCmd {
	return &RecoverAddressesCmd{
		Account: account,
		N:       n,
	}
}

//walletislockedcmd定义walletislocked json-rpc命令。
type WalletIsLockedCmd struct{}

//newwalletislockedcmd返回一个可用于发出
//walletislocked json-rpc命令。
func NewWalletIsLockedCmd() *WalletIsLockedCmd {
	return &WalletIsLockedCmd{}
}

func init() {
//此文件中的命令仅可通过
//WebSoCukes。
	flags := UFWalletOnly | UFWebsocketOnly

	MustRegisterCmd("createencryptedwallet", (*CreateEncryptedWalletCmd)(nil), flags)
	MustRegisterCmd("exportwatchingwallet", (*ExportWatchingWalletCmd)(nil), flags)
	MustRegisterCmd("getunconfirmedbalance", (*GetUnconfirmedBalanceCmd)(nil), flags)
	MustRegisterCmd("listaddresstransactions", (*ListAddressTransactionsCmd)(nil), flags)
	MustRegisterCmd("listalltransactions", (*ListAllTransactionsCmd)(nil), flags)
	MustRegisterCmd("recoveraddresses", (*RecoverAddressesCmd)(nil), flags)
	MustRegisterCmd("walletislocked", (*WalletIsLockedCmd)(nil), flags)
}
