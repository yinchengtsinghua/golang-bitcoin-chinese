
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

package btcjson

import "encoding/json"

//GetBlockHeaderVerboseResult models the data from the getblockheader command when
//设置了详细标志。未设置详细标志时，getBlockHeader
//返回十六进制编码的字符串。
type GetBlockHeaderVerboseResult struct {
	Hash          string  `json:"hash"`
	Confirmations int64   `json:"confirmations"`
	Height        int32   `json:"height"`
	Version       int32   `json:"version"`
	VersionHex    string  `json:"versionHex"`
	MerkleRoot    string  `json:"merkleroot"`
	Time          int64   `json:"time"`
	Nonce         uint64  `json:"nonce"`
	Bits          string  `json:"bits"`
	Difficulty    float64 `json:"difficulty"`
	PreviousHash  string  `json:"previousblockhash,omitempty"`
	NextHash      string  `json:"nextblockhash,omitempty"`
}

//当
//设置了详细标志。如果未设置verbose标志，getblock将返回
//十六进制编码字符串。
type GetBlockVerboseResult struct {
	Hash          string        `json:"hash"`
	Confirmations int64         `json:"confirmations"`
	StrippedSize  int32         `json:"strippedsize"`
	Size          int32         `json:"size"`
	Weight        int32         `json:"weight"`
	Height        int64         `json:"height"`
	Version       int32         `json:"version"`
	VersionHex    string        `json:"versionHex"`
	MerkleRoot    string        `json:"merkleroot"`
	Tx            []string      `json:"tx,omitempty"`
	RawTx         []TxRawResult `json:"rawtx,omitempty"`
	Time          int64         `json:"time"`
	Nonce         uint32        `json:"nonce"`
	Bits          string        `json:"bits"`
	Difficulty    float64       `json:"difficulty"`
	PreviousHash  string        `json:"previousblockhash"`
	NextHash      string        `json:"nextblockhash,omitempty"`
}

//createMultisigResult对从createMultisig返回的数据进行建模
//命令。
type CreateMultiSigResult struct {
	Address      string `json:"address"`
	RedeemScript string `json:"redeemScript"`
}

//decodescriptResult对decodescript命令返回的数据进行建模。
type DecodeScriptResult struct {
	Asm       string   `json:"asm"`
	ReqSigs   int32    `json:"reqSigs,omitempty"`
	Type      string   `json:"type"`
	Addresses []string `json:"addresses,omitempty"`
	P2sh      string   `json:"p2sh,omitempty"`
}

//getaddednodeinforesultaddr对
//getaddednodeinfo命令。
type GetAddedNodeInfoResultAddr struct {
	Address   string `json:"address"`
	Connected string `json:"connected"`
}

//getaddednodeinforesult对getaddednodeinfo命令中的数据进行建模。
type GetAddedNodeInfoResult struct {
	AddedNode string                        `json:"addednode"`
	Connected *bool                         `json:"connected,omitempty"`
	Addresses *[]GetAddedNodeInfoResultAddr `json:"addresses,omitempty"`
}

//SoftForkDescription描述了软分叉的当前状态，
//使用超多数分组信令部署。
type SoftForkDescription struct {
	ID      string `json:"id"`
	Version uint32 `json:"version"`
	Reject  struct {
		Status bool `json:"status"`
	} `json:"reject"`
}

//Bip9SoftForkDescription描述定义的Bip0009的当前状态
//版本位软叉。
type Bip9SoftForkDescription struct {
	Status    string `json:"status"`
	Bit       uint8  `json:"bit"`
	StartTime int64  `json:"startTime"`
	Timeout   int64  `json:"timeout"`
	Since     int32  `json:"since"`
}

//GetBlockChainInformation结果建模从GetBlockChainInformation返回的数据
//命令。
type GetBlockChainInfoResult struct {
	Chain                string                              `json:"chain"`
	Blocks               int32                               `json:"blocks"`
	Headers              int32                               `json:"headers"`
	BestBlockHash        string                              `json:"bestblockhash"`
	Difficulty           float64                             `json:"difficulty"`
	MedianTime           int64                               `json:"mediantime"`
	VerificationProgress float64                             `json:"verificationprogress,omitempty"`
	Pruned               bool                                `json:"pruned"`
	PruneHeight          int32                               `json:"pruneheight,omitempty"`
	ChainWork            string                              `json:"chainwork,omitempty"`
	SoftForks            []*SoftForkDescription              `json:"softforks"`
	Bip9SoftForks        map[string]*Bip9SoftForkDescription `json:"bip9_softforks"`
}

//GetBlockTemplateResultTx为
//getblocktemplate命令。
type GetBlockTemplateResultTx struct {
	Data    string  `json:"data"`
	Hash    string  `json:"hash"`
	Depends []int64 `json:"depends"`
	Fee     int64   `json:"fee"`
	SigOps  int64   `json:"sigops"`
	Weight  int64   `json:"weight"`
}

//getBlockTemplateResultAux为
//getblocktemplate命令。
type GetBlockTemplateResultAux struct {
	Flags string `json:"flags"`
}

//GetBlockTemplateResult为从GetBlockTemplate返回的数据建模
//命令。
type GetBlockTemplateResult struct {
//来自BIP 0022的基字段。CoinBaseAux是可选的。什么之中的一个
//必须指定coinbasetxn或coinbasevalue，但不能同时指定两者。
	Bits          string                     `json:"bits"`
	CurTime       int64                      `json:"curtime"`
	Height        int64                      `json:"height"`
	PreviousHash  string                     `json:"previousblockhash"`
	SigOpLimit    int64                      `json:"sigoplimit,omitempty"`
	SizeLimit     int64                      `json:"sizelimit,omitempty"`
	WeightLimit   int64                      `json:"weightlimit,omitempty"`
	Transactions  []GetBlockTemplateResultTx `json:"transactions"`
	Version       int32                      `json:"version"`
	CoinbaseAux   *GetBlockTemplateResultAux `json:"coinbaseaux,omitempty"`
	CoinbaseTxn   *GetBlockTemplateResultTx  `json:"coinbasetxn,omitempty"`
	CoinbaseValue *int64                     `json:"coinbasevalue,omitempty"`
	WorkID        string                     `json:"workid,omitempty"`

//BIP 0141中定义的证人承诺。
	DefaultWitnessCommitment string `json:"default_witness_commitment,omitempty"`

//BIP 0022的可选长轮询。
	LongPollID  string `json:"longpollid,omitempty"`
	LongPollURI string `json:"longpolluri,omitempty"`
	SubmitOld   *bool  `json:"submitold,omitempty"`

//BIP 0023的基本池扩展。
	Target  string `json:"target,omitempty"`
	Expires int64  `json:"expires,omitempty"`

//来自BIP 0023的突变。
	MaxTime    int64    `json:"maxtime,omitempty"`
	MinTime    int64    `json:"mintime,omitempty"`
	Mutable    []string `json:"mutable,omitempty"`
	NonceRange string   `json:"noncerange,omitempty"`

//来自BIP 0023的封锁提案。
	Capabilities  []string `json:"capabilities,omitempty"`
	RejectReasion string   `json:"reject-reason,omitempty"`
}

//GetMemPoolentryResult对从GetMemPoolentry返回的数据进行建模
//命令。
type GetMempoolEntryResult struct {
	Size             int32    `json:"size"`
	Fee              float64  `json:"fee"`
	ModifiedFee      float64  `json:"modifiedfee"`
	Time             int64    `json:"time"`
	Height           int64    `json:"height"`
	StartingPriority float64  `json:"startingpriority"`
	CurrentPriority  float64  `json:"currentpriority"`
	DescendantCount  int64    `json:"descendantcount"`
	DescendantSize   int64    `json:"descendantsize"`
	DescendantFees   float64  `json:"descendantfees"`
	AncestorCount    int64    `json:"ancestorcount"`
	AncestorSize     int64    `json:"ancestorsize"`
	AncestorFees     float64  `json:"ancestorfees"`
	Depends          []string `json:"depends"`
}

//GetMemPoolinForeResult对从GetMemPoolinfo返回的数据进行建模
//命令。
type GetMempoolInfoResult struct {
	Size  int64 `json:"size"`
	Bytes int64 `json:"bytes"`
}

//networksresult从getnetworkinfo命令对网络数据建模。
type NetworksResult struct {
	Name                      string `json:"name"`
	Limited                   bool   `json:"limited"`
	Reachable                 bool   `json:"reachable"`
	Proxy                     string `json:"proxy"`
	ProxyRandomizeCredentials bool   `json:"proxy_randomize_credentials"`
}

//localaddressesresult对getnetworkinfo中的localaddresses数据进行建模
//命令。
type LocalAddressesResult struct {
	Address string `json:"address"`
	Port    uint16 `json:"port"`
	Score   int32  `json:"score"`
}

//getnetworkinforesult对从getnetworkinfo返回的数据进行建模
//命令。
type GetNetworkInfoResult struct {
	Version         int32                  `json:"version"`
	SubVersion      string                 `json:"subversion"`
	ProtocolVersion int32                  `json:"protocolversion"`
	LocalServices   string                 `json:"localservices"`
	LocalRelay      bool                   `json:"localrelay"`
	TimeOffset      int64                  `json:"timeoffset"`
	Connections     int32                  `json:"connections"`
	NetworkActive   bool                   `json:"networkactive"`
	Networks        []NetworksResult       `json:"networks"`
	RelayFee        float64                `json:"relayfee"`
	IncrementalFee  float64                `json:"incrementalfee"`
	LocalAddresses  []LocalAddressesResult `json:"localaddresses"`
	Warnings        string                 `json:"warnings"`
}

//getpeerinforesult对从getpeerinfo命令返回的数据进行建模。
type GetPeerInfoResult struct {
	ID             int32   `json:"id"`
	Addr           string  `json:"addr"`
	AddrLocal      string  `json:"addrlocal,omitempty"`
	Services       string  `json:"services"`
	RelayTxes      bool    `json:"relaytxes"`
	LastSend       int64   `json:"lastsend"`
	LastRecv       int64   `json:"lastrecv"`
	BytesSent      uint64  `json:"bytessent"`
	BytesRecv      uint64  `json:"bytesrecv"`
	ConnTime       int64   `json:"conntime"`
	TimeOffset     int64   `json:"timeoffset"`
	PingTime       float64 `json:"pingtime"`
	PingWait       float64 `json:"pingwait,omitempty"`
	Version        uint32  `json:"version"`
	SubVer         string  `json:"subver"`
	Inbound        bool    `json:"inbound"`
	StartingHeight int32   `json:"startingheight"`
	CurrentHeight  int32   `json:"currentheight,omitempty"`
	BanScore       int32   `json:"banscore"`
	FeeFilter      int64   `json:"feefilter"`
	SyncNode       bool    `json:"syncnode"`
}

//getrawmEmpoolVerboseResult对从getrawmEmpool返回的数据进行建模
//设置详细标志时的命令。如果未设置详细标志，
//GetRawmEmpool返回事务哈希数组。
type GetRawMempoolVerboseResult struct {
	Size             int32    `json:"size"`
	Vsize            int32    `json:"vsize"`
	Fee              float64  `json:"fee"`
	Time             int64    `json:"time"`
	Height           int64    `json:"height"`
	StartingPriority float64  `json:"startingpriority"`
	CurrentPriority  float64  `json:"currentpriority"`
	Depends          []string `json:"depends"`
}

//scriptPubKeyResult为Tx脚本的scriptPubKey数据建模。它是
//单独定义，因为它由多个命令使用。
type ScriptPubKeyResult struct {
	Asm       string   `json:"asm"`
	Hex       string   `json:"hex,omitempty"`
	ReqSigs   int32    `json:"reqSigs,omitempty"`
	Type      string   `json:"type"`
	Addresses []string `json:"addresses,omitempty"`
}

//gettxoutresult对gettxout命令中的数据进行建模。
type GetTxOutResult struct {
	BestBlock     string             `json:"bestblock"`
	Confirmations int64              `json:"confirmations"`
	Value         float64            `json:"value"`
	ScriptPubKey  ScriptPubKeyResult `json:"scriptPubKey"`
	Coinbase      bool               `json:"coinbase"`
}

//getnettotalsresult对从getnettotals命令返回的数据进行建模。
type GetNetTotalsResult struct {
	TotalBytesRecv uint64 `json:"totalbytesrecv"`
	TotalBytesSent uint64 `json:"totalbytessent"`
	TimeMillis     int64  `json:"timemillis"`
}

//scriptsig为签名脚本建模。它是单独定义的，因为它只是
//适用于非CoinBase。因此，VIN结构中的字段需要
//成为一个指针。
type ScriptSig struct {
	Asm string `json:"asm"`
	Hex string `json:"hex"`
}

//VIN为部分Tx数据建模。它是单独定义的，因为
//getrawtransaction、decoderawtransaction和searchrawtransaction使用
//相同的结构。
type Vin struct {
	Coinbase  string     `json:"coinbase"`
	Txid      string     `json:"txid"`
	Vout      uint32     `json:"vout"`
	ScriptSig *ScriptSig `json:"scriptSig"`
	Sequence  uint32     `json:"sequence"`
	Witness   []string   `json:"txinwitness"`
}

//is coinbase返回一个bool来显示VIN是否为coinbase。
func (v *Vin) IsCoinBase() bool {
	return len(v.Coinbase) > 0
}

//has witness返回一个bool以显示VIN是否有任何关联的见证数据
//不管它与否。
func (v *Vin) HasWitness() bool {
	return len(v.Witness) > 0
}

//marshaljson为VIN提供了一个自定义的marshal方法。
func (v *Vin) MarshalJSON() ([]byte, error) {
	if v.IsCoinBase() {
		coinbaseStruct := struct {
			Coinbase string   `json:"coinbase"`
			Sequence uint32   `json:"sequence"`
			Witness  []string `json:"witness,omitempty"`
		}{
			Coinbase: v.Coinbase,
			Sequence: v.Sequence,
			Witness:  v.Witness,
		}
		return json.Marshal(coinbaseStruct)
	}

	if v.HasWitness() {
		txStruct := struct {
			Txid      string     `json:"txid"`
			Vout      uint32     `json:"vout"`
			ScriptSig *ScriptSig `json:"scriptSig"`
			Witness   []string   `json:"txinwitness"`
			Sequence  uint32     `json:"sequence"`
		}{
			Txid:      v.Txid,
			Vout:      v.Vout,
			ScriptSig: v.ScriptSig,
			Witness:   v.Witness,
			Sequence:  v.Sequence,
		}
		return json.Marshal(txStruct)
	}

	txStruct := struct {
		Txid      string     `json:"txid"`
		Vout      uint32     `json:"vout"`
		ScriptSig *ScriptSig `json:"scriptSig"`
		Sequence  uint32     `json:"sequence"`
	}{
		Txid:      v.Txid,
		Vout:      v.Vout,
		ScriptSig: v.ScriptSig,
		Sequence:  v.Sequence,
	}
	return json.Marshal(txStruct)
}

//prevout表示输入VIN的前一个输出。
type PrevOut struct {
	Addresses []string `json:"addresses,omitempty"`
	Value     float64  `json:"value"`
}

//vin prevout与vin类似，只是它包括prevout。它由searchrawtransaction使用
type VinPrevOut struct {
	Coinbase  string     `json:"coinbase"`
	Txid      string     `json:"txid"`
	Vout      uint32     `json:"vout"`
	ScriptSig *ScriptSig `json:"scriptSig"`
	Witness   []string   `json:"txinwitness"`
	PrevOut   *PrevOut   `json:"prevOut"`
	Sequence  uint32     `json:"sequence"`
}

//is coinbase返回一个bool来显示VIN是否为coinbase。
func (v *VinPrevOut) IsCoinBase() bool {
	return len(v.Coinbase) > 0
}

//has witness返回一个bool以显示VIN是否有任何关联的见证数据
//不管它与否。
func (v *VinPrevOut) HasWitness() bool {
	return len(v.Witness) > 0
}

//marshaljson为vinprevout提供了一个自定义的marshal方法。
func (v *VinPrevOut) MarshalJSON() ([]byte, error) {
	if v.IsCoinBase() {
		coinbaseStruct := struct {
			Coinbase string `json:"coinbase"`
			Sequence uint32 `json:"sequence"`
		}{
			Coinbase: v.Coinbase,
			Sequence: v.Sequence,
		}
		return json.Marshal(coinbaseStruct)
	}

	if v.HasWitness() {
		txStruct := struct {
			Txid      string     `json:"txid"`
			Vout      uint32     `json:"vout"`
			ScriptSig *ScriptSig `json:"scriptSig"`
			Witness   []string   `json:"txinwitness"`
			PrevOut   *PrevOut   `json:"prevOut,omitempty"`
			Sequence  uint32     `json:"sequence"`
		}{
			Txid:      v.Txid,
			Vout:      v.Vout,
			ScriptSig: v.ScriptSig,
			Witness:   v.Witness,
			PrevOut:   v.PrevOut,
			Sequence:  v.Sequence,
		}
		return json.Marshal(txStruct)
	}

	txStruct := struct {
		Txid      string     `json:"txid"`
		Vout      uint32     `json:"vout"`
		ScriptSig *ScriptSig `json:"scriptSig"`
		PrevOut   *PrevOut   `json:"prevOut,omitempty"`
		Sequence  uint32     `json:"sequence"`
	}{
		Txid:      v.Txid,
		Vout:      v.Vout,
		ScriptSig: v.ScriptSig,
		PrevOut:   v.PrevOut,
		Sequence:  v.Sequence,
	}
	return json.Marshal(txStruct)
}

//
//
type Vout struct {
	Value        float64            `json:"value"`
	N            uint32             `json:"n"`
	ScriptPubKey ScriptPubKeyResult `json:"scriptPubKey"`
}

//
type GetMiningInfoResult struct {
	Blocks             int64   `json:"blocks"`
	CurrentBlockSize   uint64  `json:"currentblocksize"`
	CurrentBlockWeight uint64  `json:"currentblockweight"`
	CurrentBlockTx     uint64  `json:"currentblocktx"`
	Difficulty         float64 `json:"difficulty"`
	Errors             string  `json:"errors"`
	Generate           bool    `json:"generate"`
	GenProcLimit       int32   `json:"genproclimit"`
	HashesPerSec       int64   `json:"hashespersec"`
	NetworkHashPS      int64   `json:"networkhashps"`
	PooledTx           uint64  `json:"pooledtx"`
	TestNet            bool    `json:"testnet"`
}

//
type GetWorkResult struct {
	Data     string `json:"data"`
	Hash1    string `json:"hash1"`
	Midstate string `json:"midstate"`
	Target   string `json:"target"`
}

//
type InfoChainResult struct {
	Version         int32   `json:"version"`
	ProtocolVersion int32   `json:"protocolversion"`
	Blocks          int32   `json:"blocks"`
	TimeOffset      int64   `json:"timeoffset"`
	Connections     int32   `json:"connections"`
	Proxy           string  `json:"proxy"`
	Difficulty      float64 `json:"difficulty"`
	TestNet         bool    `json:"testnet"`
	RelayFee        float64 `json:"relayfee"`
	Errors          string  `json:"errors"`
}

//txrawresult对getrawtransaction命令中的数据进行建模。
type TxRawResult struct {
	Hex           string `json:"hex"`
	Txid          string `json:"txid"`
	Hash          string `json:"hash,omitempty"`
	Size          int32  `json:"size,omitempty"`
	Vsize         int32  `json:"vsize,omitempty"`
	Version       int32  `json:"version"`
	LockTime      uint32 `json:"locktime"`
	Vin           []Vin  `json:"vin"`
	Vout          []Vout `json:"vout"`
	BlockHash     string `json:"blockhash,omitempty"`
	Confirmations uint64 `json:"confirmations,omitempty"`
	Time          int64  `json:"time,omitempty"`
	Blocktime     int64  `json:"blocktime,omitempty"`
}

//SearchRawTransactions结果为SearchRawTransaction中的数据建模
//命令。
type SearchRawTransactionsResult struct {
	Hex           string       `json:"hex,omitempty"`
	Txid          string       `json:"txid"`
	Hash          string       `json:"hash"`
	Size          string       `json:"size"`
	Vsize         string       `json:"vsize"`
	Version       int32        `json:"version"`
	LockTime      uint32       `json:"locktime"`
	Vin           []VinPrevOut `json:"vin"`
	Vout          []Vout       `json:"vout"`
	BlockHash     string       `json:"blockhash,omitempty"`
	Confirmations uint64       `json:"confirmations,omitempty"`
	Time          int64        `json:"time,omitempty"`
	Blocktime     int64        `json:"blocktime,omitempty"`
}

//txrawdecoderesult对decoderawtransaction命令中的数据进行建模。
type TxRawDecodeResult struct {
	Txid     string `json:"txid"`
	Version  int32  `json:"version"`
	Locktime uint32 `json:"locktime"`
	Vin      []Vin  `json:"vin"`
	Vout     []Vout `json:"vout"`
}

//validatedReschainResult为链服务器返回的数据建模
//validateAddress命令。
type ValidateAddressChainResult struct {
	IsValid bool   `json:"isvalid"`
	Address string `json:"address,omitempty"`
}
