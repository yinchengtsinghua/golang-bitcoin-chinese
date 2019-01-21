
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

//注意：此文件用于存放以下RPC WebSocket通知：
//由链式服务器支持。

package btcjson

const (
//BlockConnectedNTFnMethod是用于
//来自链服务器的已连接块的通知。
//
//注意：已弃用。请改用filteredblockconnectedntfnmethod。
	BlockConnectedNtfnMethod = "blockconnected"

//blockdisconnectedntfnmethod是用于
//来自链服务器的通知
//断开的。
//
//注意：已弃用。请改用filteredblockdisconnectedntfnmethod。
	BlockDisconnectedNtfnMethod = "blockdisconnected"

//filteredblockconnectedntfnmethod是用于
//来自链服务器的已连接块的通知。
	FilteredBlockConnectedNtfnMethod = "filteredblockconnected"

//filteredblockdisconnectedntfnmethod是用于
//来自链服务器的通知
//断开的。
	FilteredBlockDisconnectedNtfnMethod = "filteredblockdisconnected"

//recvtxntfnmethod是用于
//来自链服务器的通知
//已处理注册地址。
//
//注意：已弃用。使用相关的ttxacceptedntfn方法和
//相反，filteredblockconnectedntfnmethod。
	RecvTxNtfnMethod = "recvtx"

//redeemingtxntfnmethod是用于
//来自链服务器的通知
//已处理注册的输出点。
//
//注意：已弃用。使用相关的ttxacceptedntfn方法和
//相反，filteredblockconnectedntfnmethod。
	RedeemingTxNtfnMethod = "redeemingtx"

//RescanFinishedNtfnMethod是用于
//来自链服务器的通知，旧的、不推荐使用的重新扫描
//操作已完成。
//
//注意：已弃用。不与rescanblocks命令一起使用。
	RescanFinishedNtfnMethod = "rescanfinished"

//rescanprogressntfnmethod是用于
//来自链服务器的通知，旧的、不推荐使用的重新扫描
//目前正在进行的行动已经取得进展。
//
//注意：已弃用。不与rescanblocks命令一起使用。
	RescanProgressNtfnMethod = "rescanprogress"

//txAcceptedNTFnMethod是用于来自
//
	TxAcceptedNtfnMethod = "txaccepted"

//txAcceptedVerbosentfnMethod是用于通知的方法
//事务已被接受到的链服务器
//内存池。这与TxAcceptedNTFnMethod不同，它提供
//通知中的详细信息。
	TxAcceptedVerboseNtfnMethod = "txacceptedverbose"

//relevantxacceptedntfnmethod是用于通知的新方法
//从通知客户机
//匹配已加载的筛选器已被mempool接受。
	RelevantTxAcceptedNtfnMethod = "relevanttxaccepted"
)

//blockconnectedntfn定义blockconnected json-rpc通知。
//
//注意：已弃用。请改用filteredblockconnectedntfn。
type BlockConnectedNtfn struct {
	Hash   string
	Height int32
	Time   int64
}

//NewBlockConnectedNTFN返回一个可用于发出
//blockconnected json-rpc通知。
//
//注意：已弃用。请改用newfilteredblockconnectedntfn。
func NewBlockConnectedNtfn(hash string, height int32, time int64) *BlockConnectedNtfn {
	return &BlockConnectedNtfn{
		Hash:   hash,
		Height: height,
		Time:   time,
	}
}

//blockdisconnectedntfn定义blockdisconnected json-rpc通知。
//
//注意：已弃用。请改用filteredblockdisconnectedNTFN。
type BlockDisconnectedNtfn struct {
	Hash   string
	Height int32
	Time   int64
}

//newblockdisconnectedntfn返回可用于发出
//blockdisconnected json-rpc通知。
//
//注意：已弃用。请改用newfilteredblockdisconnectedntfn。
func NewBlockDisconnectedNtfn(hash string, height int32, time int64) *BlockDisconnectedNtfn {
	return &BlockDisconnectedNtfn{
		Hash:   hash,
		Height: height,
		Time:   time,
	}
}

//filteredblockconnectedntfn定义filteredblockconnected json-rpc
//通知。
type FilteredBlockConnectedNtfn struct {
	Height        int32
	Header        string
	SubscribedTxs []string
}

//newfilteredblockconnectedntfn返回可用于
//发出filteredblockconnected json-rpc通知。
func NewFilteredBlockConnectedNtfn(height int32, header string, subscribedTxs []string) *FilteredBlockConnectedNtfn {
	return &FilteredBlockConnectedNtfn{
		Height:        height,
		Header:        header,
		SubscribedTxs: subscribedTxs,
	}
}

//filteredblockdisconnectedntfn定义filteredblockdisconnected json-rpc
//通知。
type FilteredBlockDisconnectedNtfn struct {
	Height int32
	Header string
}

//newfilteredblockdisconnectedntfn返回可用于
//发出filteredblockdisconnected json-rpc通知。
func NewFilteredBlockDisconnectedNtfn(height int32, header string) *FilteredBlockDisconnectedNtfn {
	return &FilteredBlockDisconnectedNtfn{
		Height: height,
		Header: header,
	}
}

//block details描述块中Tx的详细信息。
type BlockDetails struct {
	Height int32  `json:"height"`
	Hash   string `json:"hash"`
	Index  int    `json:"index"`
	Time   int64  `json:"time"`
}

//recvtxntfn定义recvtx json-rpc通知。
//
//注意：已弃用。使用相关的ttxacceptedntfn和filteredblockconnectedntfn
//相反。
type RecvTxNtfn struct {
	HexTx string
	Block *BlockDetails
}

//newrecvtxntfn返回可用于发出recvtx的新实例
//JSON-RPC通知。
//
//注意：已弃用。使用新的相关ttxacceptedntfn和
//而新的filteredblockconnectedntfn。
func NewRecvTxNtfn(hexTx string, block *BlockDetails) *RecvTxNtfn {
	return &RecvTxNtfn{
		HexTx: hexTx,
		Block: block,
	}
}

//redeemingtxntfn定义redeemingtx json-rpc通知。
//
//注意：已弃用。使用相关的ttxacceptedntfn和filteredblockconnectedntfn
//相反。
type RedeemingTxNtfn struct {
	HexTx string
	Block *BlockDetails
}

//newredeemingtxntfn返回可用于发出
//Redeemingtx JSON-RPC通知。
//
//注意：已弃用。使用新的相关ttxacceptedntfn和
//而新的filteredblockconnectedntfn。
func NewRedeemingTxNtfn(hexTx string, block *BlockDetails) *RedeemingTxNtfn {
	return &RedeemingTxNtfn{
		HexTx: hexTx,
		Block: block,
	}
}

//rescanfinishedntfn定义rescanfinished json-rpc通知。
//
//注意：已弃用。不与rescanblocks命令一起使用。
type RescanFinishedNtfn struct {
	Hash   string
	Height int32
	Time   int64
}

//newRescanFinishedNTFN返回一个新实例，该实例可用于发出
//重新扫描完成的JSON-RPC通知。
//
//注意：已弃用。不与rescanblocks命令一起使用。
func NewRescanFinishedNtfn(hash string, height int32, time int64) *RescanFinishedNtfn {
	return &RescanFinishedNtfn{
		Hash:   hash,
		Height: height,
		Time:   time,
	}
}

//rescanprogressntfn定义rescanprogress json-rpc通知。
//
//注意：已弃用。不与rescanblocks命令一起使用。
type RescanProgressNtfn struct {
	Hash   string
	Height int32
	Time   int64
}

//newrescanprogressntfn返回可用于发出
//rescanprogress json-rpc通知。
//
//注意：已弃用。不与rescanblocks命令一起使用。
func NewRescanProgressNtfn(hash string, height int32, time int64) *RescanProgressNtfn {
	return &RescanProgressNtfn{
		Hash:   hash,
		Height: height,
		Time:   time,
	}
}

//TxAcceptedNtfn defines the txaccepted JSON-RPC notification.
type TxAcceptedNtfn struct {
	TxID   string
	Amount float64
}

//newtxacceptedntfn返回可用于发出
//TxAccepted JSON-RPC通知。
func NewTxAcceptedNtfn(txHash string, amount float64) *TxAcceptedNtfn {
	return &TxAcceptedNtfn{
		TxID:   txHash,
		Amount: amount,
	}
}

//txAcceptedVerbosentfn定义txAcceptedVerbose JSON-RPC通知。
type TxAcceptedVerboseNtfn struct {
	RawTx TxRawResult
}

//newtxAcceptedVerbosentfn返回可用于发出
//TxAcceptedVerbose JSON-RPC通知。
func NewTxAcceptedVerboseNtfn(rawTx TxRawResult) *TxAcceptedVerboseNtfn {
	return &TxAcceptedVerboseNtfn{
		RawTx: rawTx,
	}
}

//relevantxacceptedntfn定义了relevantxaccepted的参数
//JSON-RPC通知。
type RelevantTxAcceptedNtfn struct {
	Transaction string `json:"transaction"`
}

//newrelevantxacceptedntfn返回可用于发出
//relevantx接受JSON-RPC通知。
func NewRelevantTxAcceptedNtfn(txHex string) *RelevantTxAcceptedNtfn {
	return &RelevantTxAcceptedNtfn{Transaction: txHex}
}

func init() {
//此文件中的命令只能由WebSockets使用，并且
//通知。
	flags := UFWebsocketOnly | UFNotification

	MustRegisterCmd(BlockConnectedNtfnMethod, (*BlockConnectedNtfn)(nil), flags)
	MustRegisterCmd(BlockDisconnectedNtfnMethod, (*BlockDisconnectedNtfn)(nil), flags)
	MustRegisterCmd(FilteredBlockConnectedNtfnMethod, (*FilteredBlockConnectedNtfn)(nil), flags)
	MustRegisterCmd(FilteredBlockDisconnectedNtfnMethod, (*FilteredBlockDisconnectedNtfn)(nil), flags)
	MustRegisterCmd(RecvTxNtfnMethod, (*RecvTxNtfn)(nil), flags)
	MustRegisterCmd(RedeemingTxNtfnMethod, (*RedeemingTxNtfn)(nil), flags)
	MustRegisterCmd(RescanFinishedNtfnMethod, (*RescanFinishedNtfn)(nil), flags)
	MustRegisterCmd(RescanProgressNtfnMethod, (*RescanProgressNtfn)(nil), flags)
	MustRegisterCmd(TxAcceptedNtfnMethod, (*TxAcceptedNtfn)(nil), flags)
	MustRegisterCmd(TxAcceptedVerboseNtfnMethod, (*TxAcceptedVerboseNtfn)(nil), flags)
	MustRegisterCmd(RelevantTxAcceptedNtfnMethod, (*RelevantTxAcceptedNtfn)(nil), flags)
}
