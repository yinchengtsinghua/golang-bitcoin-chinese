
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

//注意：此文件用于存放以下RPC WebSocket通知：
//由钱包服务器支持。

package btcjson

const (
//AccountBalanceFnMethod是用于帐户余额的方法
//通知。
	AccountBalanceNtfnMethod = "accountbalance"

//btcConnectedNTFnMethod是用于通知的方法，当
//钱包服务器连接到链服务器。
	BtcdConnectedNtfnMethod = "btcdconnected"

//walletlockstatentfnmethod是用于通知锁定状态的方法
//一个钱包的钱已经换了。
	WalletLockStateNtfnMethod = "walletlockstate"

//newtxtfnmethod是用于通知钱包服务器
//将新事务添加到事务存储。
	NewTxNtfnMethod = "newtx"
)

//accountbalancentfn定义accountbalance json-rpc通知。
type AccountBalanceNtfn struct {
	Account   string
Balance   float64 //在BTC中
Confirmed bool    //余额是否确认。
}

//newAccountBalancefn返回可用于发出
//AccountBalance JSON-RPC通知。
func NewAccountBalanceNtfn(account string, balance float64, confirmed bool) *AccountBalanceNtfn {
	return &AccountBalanceNtfn{
		Account:   account,
		Balance:   balance,
		Confirmed: confirmed,
	}
}

//btcconnectedntfn定义btcconnected json-rpc通知。
type BtcdConnectedNtfn struct {
	Connected bool
}

//NewBtcdConnectedNtfn返回可用于发出
//btcConnected JSON-RPC通知。
func NewBtcdConnectedNtfn(connected bool) *BtcdConnectedNtfn {
	return &BtcdConnectedNtfn{
		Connected: connected,
	}
}

//walletlockstatentfn定义walletlockstate json-rpc通知。
type WalletLockStateNtfn struct {
	Locked bool
}

//newwalletlockstatentfn返回可用于发出
//walletlockstate json-rpc通知。
func NewWalletLockStateNtfn(locked bool) *WalletLockStateNtfn {
	return &WalletLockStateNtfn{
		Locked: locked,
	}
}

//NeXTXNTFN定义了NeXTX JSON-RPC通知。
type NewTxNtfn struct {
	Account string
	Details ListTransactionsResult
}

//newnewtxtnfn返回可用于发出newtx的新实例
//JSON-RPC通知。
func NewNewTxNtfn(account string, details ListTransactionsResult) *NewTxNtfn {
	return &NewTxNtfn{
		Account: account,
		Details: details,
	}
}

func init() {
//此文件中的命令仅可通过
//WebSockets和是通知。
	flags := UFWalletOnly | UFWebsocketOnly | UFNotification

	MustRegisterCmd(AccountBalanceNtfnMethod, (*AccountBalanceNtfn)(nil), flags)
	MustRegisterCmd(BtcdConnectedNtfnMethod, (*BtcdConnectedNtfn)(nil), flags)
	MustRegisterCmd(WalletLockStateNtfnMethod, (*WalletLockStateNtfn)(nil), flags)
	MustRegisterCmd(NewTxNtfnMethod, (*NewTxNtfn)(nil), flags)
}
