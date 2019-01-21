
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

//注意：此文件用于存放受支持的rpc命令
//带有btcwallet扩展名的钱包服务器。

package btcjson

//createwaccountCmd定义createwaccount json-rpc命令。
type CreateNewAccountCmd struct {
	Account string
}

//newcreatewaccountCmd返回可用于发出
//createwaccount json-rpc命令。
func NewCreateNewAccountCmd(account string) *CreateNewAccountCmd {
	return &CreateNewAccountCmd{
		Account: account,
	}
}

//dumpwalletcmd定义dumpwallet json-rpc命令。
type DumpWalletCmd struct {
	Filename string
}

//newdumpwalletCmd返回一个可用于发出
//dumpwallet json-rpc命令。
func NewDumpWalletCmd(filename string) *DumpWalletCmd {
	return &DumpWalletCmd{
		Filename: filename,
	}
}

//importAddressCmd定义importAddress json-rpc命令。
type ImportAddressCmd struct {
	Address string
	Account string
	Rescan  *bool `jsonrpcdefault:"true"`
}

//
//importaddress json-rpc命令。
func NewImportAddressCmd(address string, account string, rescan *bool) *ImportAddressCmd {
	return &ImportAddressCmd{
		Address: address,
		Account: account,
		Rescan:  rescan,
	}
}

//importpubkeycmd定义importpubkey json-rpc命令。
type ImportPubKeyCmd struct {
	PubKey string
	Rescan *bool `jsonrpcdefault:"true"`
}

//
//importpubkey json-rpc命令。
func NewImportPubKeyCmd(pubKey string, rescan *bool) *ImportPubKeyCmd {
	return &ImportPubKeyCmd{
		PubKey: pubKey,
		Rescan: rescan,
	}
}

//importwalletcmd定义importwallet json-rpc命令。
type ImportWalletCmd struct {
	Filename string
}

//
//importwallet json-rpc命令。
func NewImportWalletCmd(filename string) *ImportWalletCmd {
	return &ImportWalletCmd{
		Filename: filename,
	}
}

//renameaccountCmd定义renameaccount json-rpc命令。
type RenameAccountCmd struct {
	OldAccount string
	NewAccount string
}

//newrenameaccountCmd返回可用于发出
//renameaccount json-rpc命令。
func NewRenameAccountCmd(oldAccount, newAccount string) *RenameAccountCmd {
	return &RenameAccountCmd{
		OldAccount: oldAccount,
		NewAccount: newAccount,
	}
}

func init() {
//此文件中的命令仅可用于钱包服务器。
	flags := UFWalletOnly

	MustRegisterCmd("createnewaccount", (*CreateNewAccountCmd)(nil), flags)
	MustRegisterCmd("dumpwallet", (*DumpWalletCmd)(nil), flags)
	MustRegisterCmd("importaddress", (*ImportAddressCmd)(nil), flags)
	MustRegisterCmd("importpubkey", (*ImportPubKeyCmd)(nil), flags)
	MustRegisterCmd("importwallet", (*ImportWalletCmd)(nil), flags)
	MustRegisterCmd("renameaccount", (*RenameAccountCmd)(nil), flags)
}
