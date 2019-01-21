
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015-2017 BTCSuite开发者
//版权所有（c）2015-2017法令开发商
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package main

import (
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/btcsuite/btcd/btcjson"
)

//helpdescsenus定义用于帮助字符串的英文描述。
var helpDescsEnUS = map[string]string{
//debuglevelCmd帮助。
	"debuglevel--synopsis": "Dynamically changes the debug logging level.\n" +
		"The levelspec can either a debug level or of the form:\n" +
		"<subsystem>=<level>,<subsystem2>=<level2>,...\n" +
		"The valid debug levels are trace, debug, info, warn, error, and critical.\n" +
		"The valid subsystems are AMGR, ADXR, BCDB, BMGR, BTCD, CHAN, DISC, PEER, RPCS, SCRP, SRVR, and TXMP.\n" +
		"Finally the keyword 'show' will return a list of the available subsystems.",
	"debuglevel-levelspec":   "The debug level(s) to use or the keyword 'show'",
	"debuglevel--condition0": "levelspec!=show",
	"debuglevel--condition1": "levelspec=show",
	"debuglevel--result0":    "The string 'Done.'",
	"debuglevel--result1":    "The list of subsystems",

//添加nodeCmd帮助。
	"addnode--synopsis": "Attempts to add or remove a persistent peer.",
	"addnode-addr":      "IP address and port of the peer to operate on",
	"addnode-subcmd":    "'add' to add a persistent peer, 'remove' to remove a persistent peer, or 'onetry' to try a single connection to a peer",

//NODECMD帮助。
	"node--synopsis":     "Attempts to add or remove a peer.",
	"node-subcmd":        "'disconnect' to remove all matching non-persistent peers, 'remove' to remove a persistent peer, or 'connect' to connect to a peer",
	"node-target":        "Either the IP address and port of the peer to operate on, or a valid peer ID.",
	"node-connectsubcmd": "'perm' to make the connected peer a permanent one, 'temp' to try a single connect to a peer",

//事务输入帮助。
	"transactioninput-txid": "The hash of the input transaction",
	"transactioninput-vout": "The specific output of the input transaction to redeem",

//创建rawtransactionCmd帮助。
	"createrawtransaction--synopsis": "Returns a new transaction spending the provided inputs and sending to the provided addresses.\n" +
		"The transaction inputs are not signed in the created transaction.\n" +
		"The signrawtransaction RPC command provided by wallet must be used to sign the resulting transaction.",
	"createrawtransaction-inputs":         "The inputs to the transaction",
	"createrawtransaction-amounts":        "JSON object with the destination addresses as keys and amounts as values",
	"createrawtransaction-amounts--key":   "address",
	"createrawtransaction-amounts--value": "n.nnn",
	"createrawtransaction-amounts--desc":  "The destination address as the key and the amount in BTC as the value",
	"createrawtransaction-locktime":       "Locktime value; a non-zero value will also locktime-activate the inputs",
	"createrawtransaction--result0":       "Hex-encoded bytes of the serialized transaction",

//脚本帮助。
	"scriptsig-asm": "Disassembly of the script",
	"scriptsig-hex": "Hex-encoded bytes of the script",

//帮助排除。
	"prevout-addresses": "previous output addresses",
	"prevout-value":     "previous output value",

//vinprevout帮助。
	"vinprevout-coinbase":    "The hex-encoded bytes of the signature script (coinbase txns only)",
	"vinprevout-txid":        "The hash of the origin transaction (non-coinbase txns only)",
	"vinprevout-vout":        "The index of the output being redeemed from the origin transaction (non-coinbase txns only)",
	"vinprevout-scriptSig":   "The signature script used to redeem the origin transaction as a JSON object (non-coinbase txns only)",
	"vinprevout-txinwitness": "The witness stack of the passed input, encoded as a JSON string array",
	"vinprevout-prevOut":     "Data from the origin transaction output with index vout.",
	"vinprevout-sequence":    "The script sequence number",

//VIN帮助。
	"vin-coinbase":    "The hex-encoded bytes of the signature script (coinbase txns only)",
	"vin-txid":        "The hash of the origin transaction (non-coinbase txns only)",
	"vin-vout":        "The index of the output being redeemed from the origin transaction (non-coinbase txns only)",
	"vin-scriptSig":   "The signature script used to redeem the origin transaction as a JSON object (non-coinbase txns only)",
	"vin-txinwitness": "The witness used to redeem the input encoded as a string array of its items",
	"vin-sequence":    "The script sequence number",

//ScriptPubKeyResult帮助。
	"scriptpubkeyresult-asm":       "Disassembly of the script",
	"scriptpubkeyresult-hex":       "Hex-encoded bytes of the script",
	"scriptpubkeyresult-reqSigs":   "The number of required signatures",
	"scriptpubkeyresult-type":      "The type of the script (e.g. 'pubkeyhash')",
	"scriptpubkeyresult-addresses": "The bitcoin addresses associated with this script",

//VUT帮助。
	"vout-value":        "The amount in BTC",
	"vout-n":            "The index of this transaction output",
	"vout-scriptPubKey": "The public key script used to pay coins as a JSON object",

//txrawdecoderesult帮助。
	"txrawdecoderesult-txid":     "The hash of the transaction",
	"txrawdecoderesult-version":  "The transaction version",
	"txrawdecoderesult-locktime": "The transaction lock time",
	"txrawdecoderesult-vin":      "The transaction inputs as JSON objects",
	"txrawdecoderesult-vout":     "The transaction outputs as JSON objects",

//decoderawtransactionCmd帮助。
	"decoderawtransaction--synopsis": "Returns a JSON object representing the provided serialized, hex-encoded transaction.",
	"decoderawtransaction-hextx":     "Serialized, hex-encoded transaction",

//解码脚本结果帮助。
	"decodescriptresult-asm":       "Disassembly of the script",
	"decodescriptresult-reqSigs":   "The number of required signatures",
	"decodescriptresult-type":      "The type of the script (e.g. 'pubkeyhash')",
	"decodescriptresult-addresses": "The bitcoin addresses associated with this script",
	"decodescriptresult-p2sh":      "The script hash for use in pay-to-script-hash transactions (only present if the provided redeem script is not already a pay-to-script-hash script)",

//decodescriptcmd帮助。
	"decodescript--synopsis": "Returns a JSON object with information about the provided hex-encoded script.",
	"decodescript-hexscript": "Hex-encoded script",

//EstimateFeCmd帮助。
	"estimatefee--synopsis": "Estimate the fee per kilobyte in satoshis " +
		"required for a transaction to be mined before a certain number of " +
		"blocks have been generated.",
	"estimatefee-numblocks": "The maximum number of blocks which can be " +
		"generated before the transaction is mined.",
	"estimatefee--result0": "Estimated fee per kilobyte in satoshis for a block to " +
		"be mined in the next NumBlocks blocks.",

//GenerateCmd帮助
	"generate--synopsis": "Generates a set number of blocks (simnet or regtest only) and returns a JSON\n" +
		" array of their hashes.",
	"generate-numblocks": "Number of blocks to generate",
	"generate--result0":  "The hashes, in order, of blocks generated by the call",

//getaddednodeinforesultaddr帮助。
	"getaddednodeinforesultaddr-address":   "The ip address for this DNS entry",
	"getaddednodeinforesultaddr-connected": "The connection 'direction' (inbound/outbound/false)",

//GetAddedNodeInForest帮助。
	"getaddednodeinforesult-addednode": "The ip address or domain of the added peer",
	"getaddednodeinforesult-connected": "Whether or not the peer is currently connected",
	"getaddednodeinforesult-addresses": "DNS lookup and connection information about the peer",

//获取AddedNodeInfo帮助。
	"getaddednodeinfo--synopsis":   "Returns information about manually added (persistent) peers.",
	"getaddednodeinfo-dns":         "Specifies whether the returned data is a JSON object including DNS and connection information, or just a list of added peers",
	"getaddednodeinfo-node":        "Only return information about this specific peer instead of all added peers",
	"getaddednodeinfo--condition0": "dns=false",
	"getaddednodeinfo--condition1": "dns=true",
	"getaddednodeinfo--result0":    "List of added peers",

//GetBestBlockResult帮助。
	"getbestblockresult-hash":   "Hex-encoded bytes of the best block hash",
	"getbestblockresult-height": "Height of the best block",

//GetBestBlockCmd帮助。
	"getbestblock--synopsis": "Get block height and hash of best block in the main chain.",
	"getbestblock--result0":  "Get block height and hash of best block in the main chain.",

//GetBestBlockHashCmd帮助。
	"getbestblockhash--synopsis": "Returns the hash of the of the best (most recent) block in the longest block chain.",
	"getbestblockhash--result0":  "The hex-encoded block hash",

//GetBlockCmd帮助。
	"getblock--synopsis":   "Returns information about a block given its hash.",
	"getblock-hash":        "The hash of the block",
	"getblock-verbose":     "Specifies the block is returned as a JSON object instead of hex-encoded string",
	"getblock-verbosetx":   "Specifies that each transaction is returned as a JSON object and only applies if the verbose flag is true (btcd extension)",
	"getblock--condition0": "verbose=false",
	"getblock--condition1": "verbose=true",
	"getblock--result0":    "Hex-encoded bytes of the serialized block",

//GetBlockChainInfoCmd帮助。
	"getblockchaininfo--synopsis": "Returns information about the current blockchain state and the status of any active soft-fork deployments.",

//GetBlockChainInformation结果帮助。
	"getblockchaininforesult-chain":                 "The name of the chain the daemon is on (testnet, mainnet, etc)",
	"getblockchaininforesult-blocks":                "The number of blocks in the best known chain",
	"getblockchaininforesult-headers":               "The number of headers that we've gathered for in the best known chain",
	"getblockchaininforesult-bestblockhash":         "The block hash for the latest block in the main chain",
	"getblockchaininforesult-difficulty":            "The current chain difficulty",
	"getblockchaininforesult-mediantime":            "The median time from the PoV of the best block in the chain",
	"getblockchaininforesult-verificationprogress":  "An estimate for how much of the best chain we've verified",
	"getblockchaininforesult-pruned":                "A bool that indicates if the node is pruned or not",
	"getblockchaininforesult-pruneheight":           "The lowest block retained in the current pruned chain",
	"getblockchaininforesult-chainwork":             "The total cumulative work in the best chain",
	"getblockchaininforesult-softforks":             "The status of the super-majority soft-forks",
	"getblockchaininforesult-bip9_softforks":        "JSON object describing active BIP0009 deployments",
	"getblockchaininforesult-bip9_softforks--key":   "bip9_softforks",
	"getblockchaininforesult-bip9_softforks--value": "An object describing a particular BIP009 deployment",
	"getblockchaininforesult-bip9_softforks--desc":  "The status of any defined BIP0009 soft-fork deployments",

//SoftForkDescription帮助。
	"softforkdescription-reject":  "The current activation status of the softfork",
	"softforkdescription-version": "The block version that signals enforcement of this softfork",
	"softforkdescription-id":      "The string identifier for the soft fork",
	"-status":                     "A bool which indicates if the soft fork is active",

//txrawresult帮助。
	"txrawresult-hex":           "Hex-encoded transaction",
	"txrawresult-txid":          "The hash of the transaction",
	"txrawresult-version":       "The transaction version",
	"txrawresult-locktime":      "The transaction lock time",
	"txrawresult-vin":           "The transaction inputs as JSON objects",
	"txrawresult-vout":          "The transaction outputs as JSON objects",
	"txrawresult-blockhash":     "Hash of the block the transaction is part of",
	"txrawresult-confirmations": "Number of confirmations of the block",
	"txrawresult-time":          "Transaction time in seconds since 1 Jan 1970 GMT",
	"txrawresult-blocktime":     "Block time in seconds since the 1 Jan 1970 GMT",
	"txrawresult-size":          "The size of the transaction in bytes",
	"txrawresult-vsize":         "The virtual size of the transaction in bytes",
	"txrawresult-hash":          "The wtxid of the transaction",

//搜索rawtransactions结果帮助。
	"searchrawtransactionsresult-hex":           "Hex-encoded transaction",
	"searchrawtransactionsresult-txid":          "The hash of the transaction",
	"searchrawtransactionsresult-hash":          "The wxtid of the transaction",
	"searchrawtransactionsresult-version":       "The transaction version",
	"searchrawtransactionsresult-locktime":      "The transaction lock time",
	"searchrawtransactionsresult-vin":           "The transaction inputs as JSON objects",
	"searchrawtransactionsresult-vout":          "The transaction outputs as JSON objects",
	"searchrawtransactionsresult-blockhash":     "Hash of the block the transaction is part of",
	"searchrawtransactionsresult-confirmations": "Number of confirmations of the block",
	"searchrawtransactionsresult-time":          "Transaction time in seconds since 1 Jan 1970 GMT",
	"searchrawtransactionsresult-blocktime":     "Block time in seconds since the 1 Jan 1970 GMT",
	"searchrawtransactionsresult-size":          "The size of the transaction in bytes",
	"searchrawtransactionsresult-vsize":         "The virtual size of the transaction in bytes",

//GetBlockVerboseResult帮助。
	"getblockverboseresult-hash":              "The hash of the block (same as provided)",
	"getblockverboseresult-confirmations":     "The number of confirmations",
	"getblockverboseresult-size":              "The size of the block",
	"getblockverboseresult-height":            "The height of the block in the block chain",
	"getblockverboseresult-version":           "The block version",
	"getblockverboseresult-versionHex":        "The block version in hexadecimal",
	"getblockverboseresult-merkleroot":        "Root hash of the merkle tree",
	"getblockverboseresult-tx":                "The transaction hashes (only when verbosetx=false)",
	"getblockverboseresult-rawtx":             "The transactions as JSON objects (only when verbosetx=true)",
	"getblockverboseresult-time":              "The block time in seconds since 1 Jan 1970 GMT",
	"getblockverboseresult-nonce":             "The block nonce",
	"getblockverboseresult-bits":              "The bits which represent the block difficulty",
	"getblockverboseresult-difficulty":        "The proof-of-work difficulty as a multiple of the minimum difficulty",
	"getblockverboseresult-previousblockhash": "The hash of the previous block",
	"getblockverboseresult-nextblockhash":     "The hash of the next block (only if there is one)",
	"getblockverboseresult-strippedsize":      "The size of the block without witness data",
	"getblockverboseresult-weight":            "The weight of the block",

//GetBlockCountCmd帮助。
	"getblockcount--synopsis": "Returns the number of blocks in the longest block chain.",
	"getblockcount--result0":  "The current block count",

//GetBlockHashCmd帮助。
	"getblockhash--synopsis": "Returns hash of the block in best block chain at the given height.",
	"getblockhash-index":     "The block height",
	"getblockhash--result0":  "The block hash",

//GetBlockHeaderCmd帮助。
	"getblockheader--synopsis":   "Returns information about a block header given its hash.",
	"getblockheader-hash":        "The hash of the block",
	"getblockheader-verbose":     "Specifies the block header is returned as a JSON object instead of hex-encoded string",
	"getblockheader--condition0": "verbose=false",
	"getblockheader--condition1": "verbose=true",
	"getblockheader--result0":    "The block header hash",

//GetBlockHeaderboseResult帮助。
	"getblockheaderverboseresult-hash":              "The hash of the block (same as provided)",
	"getblockheaderverboseresult-confirmations":     "The number of confirmations",
	"getblockheaderverboseresult-height":            "The height of the block in the block chain",
	"getblockheaderverboseresult-version":           "The block version",
	"getblockheaderverboseresult-versionHex":        "The block version in hexadecimal",
	"getblockheaderverboseresult-merkleroot":        "Root hash of the merkle tree",
	"getblockheaderverboseresult-time":              "The block time in seconds since 1 Jan 1970 GMT",
	"getblockheaderverboseresult-nonce":             "The block nonce",
	"getblockheaderverboseresult-bits":              "The bits which represent the block difficulty",
	"getblockheaderverboseresult-difficulty":        "The proof-of-work difficulty as a multiple of the minimum difficulty",
	"getblockheaderverboseresult-previousblockhash": "The hash of the previous block",
	"getblockheaderverboseresult-nextblockhash":     "The hash of the next block (only if there is one)",

//模板请求帮助。
	"templaterequest-mode":         "This is 'template', 'proposal', or omitted",
	"templaterequest-capabilities": "List of capabilities",
	"templaterequest-longpollid":   "The long poll ID of a job to monitor for expiration; required and valid only for long poll requests ",
	"templaterequest-sigoplimit":   "Number of signature operations allowed in blocks (this parameter is ignored)",
	"templaterequest-sizelimit":    "Number of bytes allowed in blocks (this parameter is ignored)",
	"templaterequest-maxversion":   "Highest supported block version number (this parameter is ignored)",
	"templaterequest-target":       "The desired target for the block template (this parameter is ignored)",
	"templaterequest-data":         "Hex-encoded block data (only for mode=proposal)",
	"templaterequest-workid":       "The server provided workid if provided in block template (not applicable)",

//GetBlockTemplateResultTx帮助。
	"getblocktemplateresulttx-data":    "Hex-encoded transaction data (byte-for-byte)",
	"getblocktemplateresulttx-hash":    "Hex-encoded transaction hash (little endian if treated as a 256-bit number)",
	"getblocktemplateresulttx-depends": "Other transactions before this one (by 1-based index in the 'transactions'  list) that must be present in the final block if this one is",
	"getblocktemplateresulttx-fee":     "Difference in value between transaction inputs and outputs (in Satoshi)",
	"getblocktemplateresulttx-sigops":  "Total number of signature operations as counted for purposes of block limits",
	"getblocktemplateresulttx-weight":  "The weight of the transaction",

//GetBlockTemplateResultAux帮助。
	"getblocktemplateresultaux-flags": "Hex-encoded byte-for-byte data to include in the coinbase signature script",

//获取块模板结果帮助。
	"getblocktemplateresult-bits":                       "Hex-encoded compressed difficulty",
	"getblocktemplateresult-curtime":                    "Current time as seen by the server (recommended for block time); must fall within mintime/maxtime rules",
	"getblocktemplateresult-height":                     "Height of the block to be solved",
	"getblocktemplateresult-previousblockhash":          "Hex-encoded big-endian hash of the previous block",
	"getblocktemplateresult-sigoplimit":                 "Number of sigops allowed in blocks ",
	"getblocktemplateresult-sizelimit":                  "Number of bytes allowed in blocks",
	"getblocktemplateresult-transactions":               "Array of transactions as JSON objects",
	"getblocktemplateresult-version":                    "The block version",
	"getblocktemplateresult-coinbaseaux":                "Data that should be included in the coinbase signature script",
	"getblocktemplateresult-coinbasetxn":                "Information about the coinbase transaction",
	"getblocktemplateresult-coinbasevalue":              "Total amount available for the coinbase in Satoshi",
	"getblocktemplateresult-workid":                     "This value must be returned with result if provided (not provided)",
	"getblocktemplateresult-longpollid":                 "Identifier for long poll request which allows monitoring for expiration",
	"getblocktemplateresult-longpolluri":                "An alternate URI to use for long poll requests if provided (not provided)",
	"getblocktemplateresult-submitold":                  "Not applicable",
	"getblocktemplateresult-target":                     "Hex-encoded big-endian number which valid results must be less than",
	"getblocktemplateresult-expires":                    "Maximum number of seconds (starting from when the server sent the response) this work is valid for",
	"getblocktemplateresult-maxtime":                    "Maximum allowed time",
	"getblocktemplateresult-mintime":                    "Minimum allowed time",
	"getblocktemplateresult-mutable":                    "List of mutations the server explicitly allows",
	"getblocktemplateresult-noncerange":                 "Two concatenated hex-encoded big-endian 32-bit integers which represent the valid ranges of nonces the miner may scan",
	"getblocktemplateresult-capabilities":               "List of server capabilities including 'proposal' to indicate support for block proposals",
	"getblocktemplateresult-reject-reason":              "Reason the proposal was invalid as-is (only applies to proposal responses)",
	"getblocktemplateresult-default_witness_commitment": "The witness commitment itself. Will be populated if the block has witness data",
	"getblocktemplateresult-weightlimit":                "The current limit on the max allowed weight of a block",

//GetBlockTemplateCmd帮助。
	"getblocktemplate--synopsis": "Returns a JSON object with information necessary to construct a block to mine or accepts a proposal to validate.\n" +
		"See BIP0022 and BIP0023 for the full specification.",
	"getblocktemplate-request":     "Request object which controls the mode and several parameters",
	"getblocktemplate--condition0": "mode=template",
	"getblocktemplate--condition1": "mode=proposal, rejected",
	"getblocktemplate--condition2": "mode=proposal, accepted",
	"getblocktemplate--result1":    "An error string which represents why the proposal was rejected or nothing if accepted",

//getfilterCmd帮助。
	"getcfilter--synopsis":  "Returns a block's committed filter given its hash.",
	"getcfilter-filtertype": "The type of filter to return (0=regular)",
	"getcfilter-hash":       "The hash of the block",
	"getcfilter--result0":   "The block's committed filter",

//getfilterheaderCmd帮助。
	"getcfilterheader--synopsis":  "Returns a block's compact filter header given its hash.",
	"getcfilterheader-filtertype": "The type of filter header to return (0=regular)",
	"getcfilterheader-hash":       "The hash of the block",
	"getcfilterheader--result0":   "The block's gcs filter header",

//getConnectionCountCmd帮助。
	"getconnectioncount--synopsis": "Returns the number of active connections to other peers.",
	"getconnectioncount--result0":  "The number of connections",

//GetCurrentNetCmd帮助。
	"getcurrentnet--synopsis": "Get bitcoin network the server is running on.",
	"getcurrentnet--result0":  "The network identifer",

//GetDifficultyCmd帮助。
	"getdifficulty--synopsis": "Returns the proof-of-work difficulty as a multiple of the minimum difficulty.",
	"getdifficulty--result0":  "The difficulty",

//获取GenerateCmd帮助。
	"getgenerate--synopsis": "Returns if the server is set to generate coins (mine) or not.",
	"getgenerate--result0":  "True if mining, false if not",

//GetHashesperseCmd帮助。
	"gethashespersec--synopsis": "Returns a recent hashes per second performance measurement while generating coins (mining).",
	"gethashespersec--result0":  "The number of hashes per second",

//信息链结果帮助。
	"infochainresult-version":         "The version of the server",
	"infochainresult-protocolversion": "The latest supported protocol version",
	"infochainresult-blocks":          "The number of blocks processed",
	"infochainresult-timeoffset":      "The time offset",
	"infochainresult-connections":     "The number of connected peers",
	"infochainresult-proxy":           "The proxy used by the server",
	"infochainresult-difficulty":      "The current target difficulty",
	"infochainresult-testnet":         "Whether or not server is using testnet",
	"infochainresult-relayfee":        "The minimum relay fee for non-free transactions in BTC/KB",
	"infochainresult-errors":          "Any current errors",

//InfowalletResult帮助。
	"infowalletresult-version":         "The version of the server",
	"infowalletresult-protocolversion": "The latest supported protocol version",
	"infowalletresult-walletversion":   "The version of the wallet server",
	"infowalletresult-balance":         "The total bitcoin balance of the wallet",
	"infowalletresult-blocks":          "The number of blocks processed",
	"infowalletresult-timeoffset":      "The time offset",
	"infowalletresult-connections":     "The number of connected peers",
	"infowalletresult-proxy":           "The proxy used by the server",
	"infowalletresult-difficulty":      "The current target difficulty",
	"infowalletresult-testnet":         "Whether or not server is using testnet",
	"infowalletresult-keypoololdest":   "Seconds since 1 Jan 1970 GMT of the oldest pre-generated key in the key pool",
	"infowalletresult-keypoolsize":     "The number of new keys that are pre-generated",
	"infowalletresult-unlocked_until":  "The timestamp in seconds since 1 Jan 1970 GMT that the wallet is unlocked for transfers, or 0 if the wallet is locked",
	"infowalletresult-paytxfee":        "The transaction fee set in BTC/KB",
	"infowalletresult-relayfee":        "The minimum relay fee for non-free transactions in BTC/KB",
	"infowalletresult-errors":          "Any current errors",

//GetHeadersCmd帮助。
	"getheaders--synopsis":     "Returns block headers starting with the first known block hash from the request",
	"getheaders-blocklocators": "JSON array of hex-encoded hashes of blocks.  Headers are returned starting from the first known hash in this list",
	"getheaders-hashstop":      "Block hash to stop including block headers for; if not found, all headers to the latest known block are returned.",
	"getheaders--result0":      "Serialized block headers of all located blocks, limited to some arbitrary maximum number of hashes (currently 2000, which matches the wire protocol headers message, but this is not guaranteed)",

//GetInfoCmd帮助。
	"getinfo--synopsis": "Returns a JSON object containing various state info.",

//GetMempoolinFoCmd帮助。
	"getmempoolinfo--synopsis": "Returns memory pool information",

//GetMempoolinForeult帮助。
	"getmempoolinforesult-bytes": "Size in bytes of the mempool",
	"getmempoolinforesult-size":  "Number of transactions in the mempool",

//获取miningoresult帮助。
	"getmininginforesult-blocks":             "Height of the latest best block",
	"getmininginforesult-currentblocksize":   "Size of the latest best block",
	"getmininginforesult-currentblockweight": "Weight of the latest best block",
	"getmininginforesult-currentblocktx":     "Number of transactions in the latest best block",
	"getmininginforesult-difficulty":         "Current target difficulty",
	"getmininginforesult-errors":             "Any current errors",
	"getmininginforesult-generate":           "Whether or not server is set to generate coins",
	"getmininginforesult-genproclimit":       "Number of processors to use for coin generation (-1 when disabled)",
	"getmininginforesult-hashespersec":       "Recent hashes per second performance measurement while generating coins",
	"getmininginforesult-networkhashps":      "Estimated network hashes per second for the most recent blocks",
	"getmininginforesult-pooledtx":           "Number of transactions in the memory pool",
	"getmininginforesult-testnet":            "Whether or not server is using testnet",

//获取MiningForCmd帮助。
	"getmininginfo--synopsis": "Returns a JSON object containing mining-related information.",

//GetNetworkHashPSCmd帮助。
	"getnetworkhashps--synopsis": "Returns the estimated network hashes per second for the block heights provided by the parameters.",
	"getnetworkhashps-blocks":    "The number of blocks, or -1 for blocks since last difficulty change",
	"getnetworkhashps-height":    "Perform estimate ending with this height or -1 for current best chain block height",
	"getnetworkhashps--result0":  "Estimated hashes per second",

//getnettotalscmd帮助。
	"getnettotals--synopsis": "Returns a JSON object containing network traffic statistics.",

//GetNettoAlsResult帮助。
	"getnettotalsresult-totalbytesrecv": "Total bytes received",
	"getnettotalsresult-totalbytessent": "Total bytes sent",
	"getnettotalsresult-timemillis":     "Number of milliseconds since 1 Jan 1970 GMT",

//getpeerinforesult帮助。
	"getpeerinforesult-id":             "A unique node ID",
	"getpeerinforesult-addr":           "The ip address and port of the peer",
	"getpeerinforesult-addrlocal":      "Local address",
	"getpeerinforesult-services":       "Services bitmask which represents the services supported by the peer",
	"getpeerinforesult-relaytxes":      "Peer has requested transactions be relayed to it",
	"getpeerinforesult-lastsend":       "Time the last message was received in seconds since 1 Jan 1970 GMT",
	"getpeerinforesult-lastrecv":       "Time the last message was sent in seconds since 1 Jan 1970 GMT",
	"getpeerinforesult-bytessent":      "Total bytes sent",
	"getpeerinforesult-bytesrecv":      "Total bytes received",
	"getpeerinforesult-conntime":       "Time the connection was made in seconds since 1 Jan 1970 GMT",
	"getpeerinforesult-timeoffset":     "The time offset of the peer",
	"getpeerinforesult-pingtime":       "Number of microseconds the last ping took",
	"getpeerinforesult-pingwait":       "Number of microseconds a queued ping has been waiting for a response",
	"getpeerinforesult-version":        "The protocol version of the peer",
	"getpeerinforesult-subver":         "The user agent of the peer",
	"getpeerinforesult-inbound":        "Whether or not the peer is an inbound connection",
	"getpeerinforesult-startingheight": "The latest block height the peer knew about when the connection was established",
	"getpeerinforesult-currentheight":  "The current height of the peer",
	"getpeerinforesult-banscore":       "The ban score",
	"getpeerinforesult-feefilter":      "The requested minimum fee a transaction must have to be announced to the peer",
	"getpeerinforesult-syncnode":       "Whether or not the peer is the sync peer",

//getpeerinfocmd帮助。
	"getpeerinfo--synopsis": "Returns data about each connected network peer as an array of json objects.",

//获取rawmEmpoolVerboseResult帮助。
	"getrawmempoolverboseresult-size":             "Transaction size in bytes",
	"getrawmempoolverboseresult-fee":              "Transaction fee in bitcoins",
	"getrawmempoolverboseresult-time":             "Local time transaction entered pool in seconds since 1 Jan 1970 GMT",
	"getrawmempoolverboseresult-height":           "Block height when transaction entered the pool",
	"getrawmempoolverboseresult-startingpriority": "Priority when transaction entered the pool",
	"getrawmempoolverboseresult-currentpriority":  "Current priority",
	"getrawmempoolverboseresult-depends":          "Unconfirmed transactions used as inputs for this transaction",
	"getrawmempoolverboseresult-vsize":            "The virtual size of a transaction",

//GetRawmEmpoolCmd帮助。
	"getrawmempool--synopsis":   "Returns information about all of the transactions currently in the memory pool.",
	"getrawmempool-verbose":     "Returns JSON object when true or an array of transaction hashes when false",
	"getrawmempool--condition0": "verbose=false",
	"getrawmempool--condition1": "verbose=true",
	"getrawmempool--result0":    "Array of transaction hashes",

//获取rawtransactionCmd帮助。
	"getrawtransaction--synopsis":   "Returns information about a transaction given its hash.",
	"getrawtransaction-txid":        "The hash of the transaction",
	"getrawtransaction-verbose":     "Specifies the transaction is returned as a JSON object instead of a hex-encoded string",
	"getrawtransaction--condition0": "verbose=false",
	"getrawtransaction--condition1": "verbose=true",
	"getrawtransaction--result0":    "Hex-encoded bytes of the serialized transaction",

//GettXOutResult帮助。
	"gettxoutresult-bestblock":     "The block hash that contains the transaction output",
	"gettxoutresult-confirmations": "The number of confirmations",
	"gettxoutresult-value":         "The transaction amount in BTC",
	"gettxoutresult-scriptPubKey":  "The public key script used to pay coins as a JSON object",
	"gettxoutresult-version":       "The transaction version",
	"gettxoutresult-coinbase":      "Whether or not the transaction is a coinbase",

//GettXOutCmd帮助。
	"gettxout--synopsis":      "Returns information about an unspent transaction output..",
	"gettxout-txid":           "The hash of the transaction",
	"gettxout-vout":           "The index of the output",
	"gettxout-includemempool": "Include the mempool when true",

//帮助我。
	"help--synopsis":   "Returns a list of all commands or help for a specified command.",
	"help-command":     "The command to retrieve help for",
	"help--condition0": "no command provided",
	"help--condition1": "command specified",
	"help--result0":    "List of commands",
	"help--result1":    "Help for specified command",

//PangCMD帮助。
	"ping--synopsis": "Queues a ping to be sent to each connected peer.\n" +
		"Ping times are provided by getpeerinfo via the pingtime and pingwait fields.",

//搜索rawtransactionsCmd帮助。
	"searchrawtransactions--synopsis": "Returns raw data for transactions involving the passed address.\n" +
		"Returned transactions are pulled from both the database, and transactions currently in the mempool.\n" +
		"Transactions pulled from the mempool will have the 'confirmations' field set to 0.\n" +
		"Usage of this RPC requires the optional --addrindex flag to be activated, otherwise all responses will simply return with an error stating the address index has not yet been built.\n" +
		"Similarly, until the address index has caught up with the current best height, all requests will return an error response in order to avoid serving stale data.",
	"searchrawtransactions-address":     "The Bitcoin address to search for",
	"searchrawtransactions-verbose":     "Specifies the transaction is returned as a JSON object instead of hex-encoded string",
	"searchrawtransactions--condition0": "verbose=0",
	"searchrawtransactions--condition1": "verbose=1",
	"searchrawtransactions-skip":        "The number of leading transactions to leave out of the final response",
	"searchrawtransactions-count":       "The maximum number of transactions to return",
	"searchrawtransactions-vinextra":    "Specify that extra data from previous output will be returned in vin",
	"searchrawtransactions-reverse":     "Specifies that the transactions should be returned in reverse chronological order",
	"searchrawtransactions-filteraddrs": "Address list.  Only inputs or outputs with matching address will be returned",
	"searchrawtransactions--result0":    "Hex-encoded serialized transaction",

//sendrawtransactionCmd帮助。
	"sendrawtransaction--synopsis":     "Submits the serialized, hex-encoded transaction to the local peer and relays it to the network.",
	"sendrawtransaction-hextx":         "Serialized, hex-encoded signed transaction",
	"sendrawtransaction-allowhighfees": "Whether or not to allow insanely high fees (btcd does not yet implement this parameter, so it has no effect)",
	"sendrawtransaction--result0":      "The hash of the transaction",

//setGenerateCmd帮助。
	"setgenerate--synopsis":    "Set the server to generate coins (mine) or not.",
	"setgenerate-generate":     "Use true to enable generation, false to disable it",
	"setgenerate-genproclimit": "The number of processors (cores) to limit generation to or -1 for default",

//停止帮助。
	"stop--synopsis": "Shutdown btcd.",
	"stop--result0":  "The string 'btcd stopping.'",

//SubmitBlockOptions帮助。
	"submitblockoptions-workid": "This parameter is currently ignored",

//SubmitBlockCmd帮助。
	"submitblock--synopsis":   "Attempts to submit a new serialized, hex-encoded block to the network.",
	"submitblock-hexblock":    "Serialized, hex-encoded block",
	"submitblock-options":     "This parameter is currently ignored",
	"submitblock--condition0": "Block successfully submitted",
	"submitblock--condition1": "Block rejected",
	"submitblock--result1":    "The reason the block was rejected",

//验证读取结果帮助。
	"validateaddresschainresult-isvalid": "Whether or not the address is valid",
	"validateaddresschainresult-address": "The bitcoin address (only when isvalid is true)",

//ValidateADResCmd帮助。
	"validateaddress--synopsis": "Verify an address is valid.",
	"validateaddress-address":   "Bitcoin address to validate",

//VerifyChainCmd帮助。
	"verifychain--synopsis": "Verifies the block chain database.\n" +
		"The actual checks performed by the checklevel parameter are implementation specific.\n" +
		"For btcd this is:\n" +
		"checklevel=0 - Look up each block and ensure it can be loaded from the database.\n" +
		"checklevel=1 - Perform basic context-free sanity checks on each block.",
	"verifychain-checklevel": "How thorough the block verification is",
	"verifychain-checkdepth": "The number of blocks to check",
	"verifychain--result0":   "Whether or not the chain verified",

//VerifyMessageCmd帮助。
	"verifymessage--synopsis": "Verify a signed message.",
	"verifymessage-address":   "The bitcoin address to use for the signature",
	"verifymessage-signature": "The base-64 encoded signature provided by the signer",
	"verifymessage-message":   "The signed message",
	"verifymessage--result0":  "Whether or not the signature verified",

//--------WebSocket特定帮助-------

//会话帮助。
	"session--synopsis":       "Return details regarding a websocket client's current connection session.",
	"sessionresult-sessionid": "The unique session ID for a client's websocket connection.",

//NotifyBlocksCmd帮助。
	"notifyblocks--synopsis": "Request notifications for whenever a block is connected or disconnected from the main (best) chain.",

//StopNotifyBlocksCmd帮助。
	"stopnotifyblocks--synopsis": "Cancel registered notifications for whenever a block is connected or disconnected from the main (best) chain.",

//notifynewtransactionsCmd帮助。
	"notifynewtransactions--synopsis": "Send either a txaccepted or a txacceptedverbose notification when a new transaction is accepted into the mempool.",
	"notifynewtransactions-verbose":   "Specifies which type of notification to receive. If verbose is true, then the caller receives txacceptedverbose, otherwise the caller receives txaccepted",

//StopNotifyNewTransactionsCmd帮助。
	"stopnotifynewtransactions--synopsis": "Stop sending either a txaccepted or a txacceptedverbose notification when a new transaction is accepted into the mempool.",

//NotifyReceiveDCMD帮助。
	"notifyreceived--synopsis": "Send a recvtx notification when a transaction added to mempool or appears in a newly-attached block contains a txout pkScript sending to any of the passed addresses.\n" +
		"Matching outpoints are automatically registered for redeemingtx notifications.",
	"notifyreceived-addresses": "List of address to receive notifications about",

//StopNotifyReceiveDCMD帮助。
	"stopnotifyreceived--synopsis": "Cancel registered receive notifications for each passed address.",
	"stopnotifyreceived-addresses": "List of address to cancel receive notifications for",

//输出帮助。
	"outpoint-hash":  "The hex-encoded bytes of the outpoint hash",
	"outpoint-index": "The index of the outpoint",

//通知SpnetCmd帮助。
	"notifyspent--synopsis": "Send a redeemingtx notification when a transaction spending an outpoint appears in mempool (if relayed to this btcd instance) and when such a transaction first appears in a newly-attached block.",
	"notifyspent-outpoints": "List of transaction outpoints to monitor.",

//停止通知SpnetCmd帮助。
	"stopnotifyspent--synopsis": "Cancel registered spending notifications for each passed outpoint.",
	"stopnotifyspent-outpoints": "List of transaction outpoints to stop monitoring.",

//LoadTxFilterCmd帮助。
	"loadtxfilter--synopsis": "Load, add to, or reload a websocket client's transaction filter for mempool transactions, new blocks and rescanblocks.",
	"loadtxfilter-reload":    "Load a new filter instead of adding data to an existing one",
	"loadtxfilter-addresses": "Array of addresses to add to the transaction filter",
	"loadtxfilter-outpoints": "Array of outpoints to add to the transaction filter",

//再帮忙。
	"rescan--synopsis": "Rescan block chain for transactions to addresses.\n" +
		"When the endblock parameter is omitted, the rescan continues through the best block in the main chain.\n" +
		"Rescan results are sent as recvtx and redeemingtx notifications.\n" +
		"This call returns once the rescan completes.",
	"rescan-beginblock": "Hash of the first block to begin rescanning",
	"rescan-addresses":  "List of addresses to include in the rescan",
	"rescan-outpoints":  "List of transaction outpoints to include in the rescan",
	"rescan-endblock":   "Hash of final block to rescan",

//重新扫描块帮助。
	"rescanblocks--synopsis":   "Rescan blocks for transactions matching the loaded transaction filter.",
	"rescanblocks-blockhashes": "List of hashes to rescan.  Each next block must be a child of the previous.",
	"rescanblocks--result0":    "List of matching blocks.",

//重新扫描块帮助。
	"rescannedblock-hash":         "Hash of the matching block.",
	"rescannedblock-transactions": "List of matching transactions, serialized and hex-encoded.",

//正常运行时间的帮助。
	"uptime--synopsis": "Returns the total uptime of the server.",
	"uptime--result0":  "The number of seconds that the server has been running",

//版本帮助。
	"version--synopsis":       "Returns the JSON-RPC API version (semver)",
	"version--result0--desc":  "Version objects keyed by the program or API name",
	"version--result0--key":   "Program or API name",
	"version--result0--value": "Object containing the semantic version",

//版本结果帮助。
	"versionresult-versionstring": "The JSON-RPC API version (semver)",
	"versionresult-major":         "The major component of the JSON-RPC API version",
	"versionresult-minor":         "The minor component of the JSON-RPC API version",
	"versionresult-patch":         "The patch component of the JSON-RPC API version",
	"versionresult-prerelease":    "Prerelease info about the current build",
	"versionresult-buildmetadata": "Metadata about the current build",
}

//rpc result types指定每个rpc命令可以返回的结果类型。
//此信息用于生成帮助。每个结果类型必须是
//指向类型的指针（或nil表示没有返回值）。
var rpcResultTypes = map[string][]interface{}{
	"addnode":               nil,
	"createrawtransaction":  {(*string)(nil)},
	"debuglevel":            {(*string)(nil), (*string)(nil)},
	"decoderawtransaction":  {(*btcjson.TxRawDecodeResult)(nil)},
	"decodescript":          {(*btcjson.DecodeScriptResult)(nil)},
	"estimatefee":           {(*float64)(nil)},
	"generate":              {(*[]string)(nil)},
	"getaddednodeinfo":      {(*[]string)(nil), (*[]btcjson.GetAddedNodeInfoResult)(nil)},
	"getbestblock":          {(*btcjson.GetBestBlockResult)(nil)},
	"getbestblockhash":      {(*string)(nil)},
	"getblock":              {(*string)(nil), (*btcjson.GetBlockVerboseResult)(nil)},
	"getblockcount":         {(*int64)(nil)},
	"getblockhash":          {(*string)(nil)},
	"getblockheader":        {(*string)(nil), (*btcjson.GetBlockHeaderVerboseResult)(nil)},
	"getblocktemplate":      {(*btcjson.GetBlockTemplateResult)(nil), (*string)(nil), nil},
	"getblockchaininfo":     {(*btcjson.GetBlockChainInfoResult)(nil)},
	"getcfilter":            {(*string)(nil)},
	"getcfilterheader":      {(*string)(nil)},
	"getconnectioncount":    {(*int32)(nil)},
	"getcurrentnet":         {(*uint32)(nil)},
	"getdifficulty":         {(*float64)(nil)},
	"getgenerate":           {(*bool)(nil)},
	"gethashespersec":       {(*float64)(nil)},
	"getheaders":            {(*[]string)(nil)},
	"getinfo":               {(*btcjson.InfoChainResult)(nil)},
	"getmempoolinfo":        {(*btcjson.GetMempoolInfoResult)(nil)},
	"getmininginfo":         {(*btcjson.GetMiningInfoResult)(nil)},
	"getnettotals":          {(*btcjson.GetNetTotalsResult)(nil)},
	"getnetworkhashps":      {(*int64)(nil)},
	"getpeerinfo":           {(*[]btcjson.GetPeerInfoResult)(nil)},
	"getrawmempool":         {(*[]string)(nil), (*btcjson.GetRawMempoolVerboseResult)(nil)},
	"getrawtransaction":     {(*string)(nil), (*btcjson.TxRawResult)(nil)},
	"gettxout":              {(*btcjson.GetTxOutResult)(nil)},
	"node":                  nil,
	"help":                  {(*string)(nil), (*string)(nil)},
	"ping":                  nil,
	"searchrawtransactions": {(*string)(nil), (*[]btcjson.SearchRawTransactionsResult)(nil)},
	"sendrawtransaction":    {(*string)(nil)},
	"setgenerate":           nil,
	"stop":                  {(*string)(nil)},
	"submitblock":           {nil, (*string)(nil)},
	"uptime":                {(*int64)(nil)},
	"validateaddress":       {(*btcjson.ValidateAddressChainResult)(nil)},
	"verifychain":           {(*bool)(nil)},
	"verifymessage":         {(*bool)(nil)},
	"version":               {(*map[string]btcjson.VersionResult)(nil)},

//WebSocket命令。
	"loadtxfilter":              nil,
	"session":                   {(*btcjson.SessionResult)(nil)},
	"notifyblocks":              nil,
	"stopnotifyblocks":          nil,
	"notifynewtransactions":     nil,
	"stopnotifynewtransactions": nil,
	"notifyreceived":            nil,
	"stopnotifyreceived":        nil,
	"notifyspent":               nil,
	"stopnotifyspent":           nil,
	"rescan":                    nil,
	"rescanblocks":              {(*[]btcjson.RescannedBlock)(nil)},
}

//helpcacher提供了一个并发的安全类型，它为
//rpc服务器命令并缓存将来调用的结果。
type helpCacher struct {
	sync.Mutex
	usage      string
	methodHelp map[string]string
}

//rpc method help返回所提供方法的rpc帮助字符串。
//
//此函数对于并发访问是安全的。
func (c *helpCacher) rpcMethodHelp(method string) (string, error) {
	c.Lock()
	defer c.Unlock()

//返回缓存方法帮助（如果存在）。
	if help, exists := c.methodHelp[method]; exists {
		return help, nil
	}

//查找方法的结果类型。
	resultTypes, ok := rpcResultTypes[method]
	if !ok {
		return "", errors.New("no result types specified for method " +
			method)
	}

//生成、缓存并返回帮助。
	help, err := btcjson.GenerateHelp(method, helpDescsEnUS, resultTypes...)
	if err != nil {
		return "", err
	}
	c.methodHelp[method] = help
	return help, nil
}

//rpc usage返回所有支持rpc命令的一行用法。
//
//此函数对于并发访问是安全的。
func (c *helpCacher) rpcUsage(includeWebsockets bool) (string, error) {
	c.Lock()
	defer c.Unlock()

//返回缓存的用法（如果可用）。
	if c.usage != "" {
		return c.usage, nil
	}

//为每个命令生成一行用法列表。
	usageTexts := make([]string, 0, len(rpcHandlers))
	for k := range rpcHandlers {
		usage, err := btcjson.MethodUsageText(k)
		if err != nil {
			return "", err
		}
		usageTexts = append(usageTexts, usage)
	}

//如果需要，包括websockets命令。
	if includeWebsockets {
		for k := range wsHandlers {
			usage, err := btcjson.MethodUsageText(k)
			if err != nil {
				return "", err
			}
			usageTexts = append(usageTexts, usage)
		}
	}

	sort.Sort(sort.StringSlice(usageTexts))
	c.usage = strings.Join(usageTexts, "\n")
	return c.usage, nil
}

//new help cacher返回帮助缓存的新实例，该实例提供帮助和
//使用rpc服务器命令并缓存将来调用的结果。
func newHelpCacher() *helpCacher {
	return &helpCacher{
		methodHelp: make(map[string]string),
	}
}
