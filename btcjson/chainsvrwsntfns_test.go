
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

package btcjson_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/btcsuite/btcd/btcjson"
)

//testchainsvrwsntfs测试所有特定于chain服务器websocket的
//通知封送和取消封送到有效结果中包括处理
//在marshalled命令中省略了可选字段，而可选字段
//具有默认值的字段具有未编排命令的默认值。
func TestChainSvrWsNtfns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		newNtfn      func() (interface{}, error)
		staticNtfn   func() interface{}
		marshalled   string
		unmarshalled interface{}
	}{
		{
			name: "blockconnected",
			newNtfn: func() (interface{}, error) {
				return btcjson.NewCmd("blockconnected", "123", 100000, 123456789)
			},
			staticNtfn: func() interface{} {
				return btcjson.NewBlockConnectedNtfn("123", 100000, 123456789)
			},
			marshalled: `{"jsonrpc":"1.0","method":"blockconnected","params":["123",100000,123456789],"id":null}`,
			unmarshalled: &btcjson.BlockConnectedNtfn{
				Hash:   "123",
				Height: 100000,
				Time:   123456789,
			},
		},
		{
			name: "blockdisconnected",
			newNtfn: func() (interface{}, error) {
				return btcjson.NewCmd("blockdisconnected", "123", 100000, 123456789)
			},
			staticNtfn: func() interface{} {
				return btcjson.NewBlockDisconnectedNtfn("123", 100000, 123456789)
			},
			marshalled: `{"jsonrpc":"1.0","method":"blockdisconnected","params":["123",100000,123456789],"id":null}`,
			unmarshalled: &btcjson.BlockDisconnectedNtfn{
				Hash:   "123",
				Height: 100000,
				Time:   123456789,
			},
		},
		{
			name: "filteredblockconnected",
			newNtfn: func() (interface{}, error) {
				return btcjson.NewCmd("filteredblockconnected", 100000, "header", []string{"tx0", "tx1"})
			},
			staticNtfn: func() interface{} {
				return btcjson.NewFilteredBlockConnectedNtfn(100000, "header", []string{"tx0", "tx1"})
			},
			marshalled: `{"jsonrpc":"1.0","method":"filteredblockconnected","params":[100000,"header",["tx0","tx1"]],"id":null}`,
			unmarshalled: &btcjson.FilteredBlockConnectedNtfn{
				Height:        100000,
				Header:        "header",
				SubscribedTxs: []string{"tx0", "tx1"},
			},
		},
		{
			name: "filteredblockdisconnected",
			newNtfn: func() (interface{}, error) {
				return btcjson.NewCmd("filteredblockdisconnected", 100000, "header")
			},
			staticNtfn: func() interface{} {
				return btcjson.NewFilteredBlockDisconnectedNtfn(100000, "header")
			},
			marshalled: `{"jsonrpc":"1.0","method":"filteredblockdisconnected","params":[100000,"header"],"id":null}`,
			unmarshalled: &btcjson.FilteredBlockDisconnectedNtfn{
				Height: 100000,
				Header: "header",
			},
		},
		{
			name: "recvtx",
			newNtfn: func() (interface{}, error) {
				return btcjson.NewCmd("recvtx", "001122", `{"height":100000,"hash":"123","index":0,"time":12345678}`)
			},
			staticNtfn: func() interface{} {
				blockDetails := btcjson.BlockDetails{
					Height: 100000,
					Hash:   "123",
					Index:  0,
					Time:   12345678,
				}
				return btcjson.NewRecvTxNtfn("001122", &blockDetails)
			},
			marshalled: `{"jsonrpc":"1.0","method":"recvtx","params":["001122",{"height":100000,"hash":"123","index":0,"time":12345678}],"id":null}`,
			unmarshalled: &btcjson.RecvTxNtfn{
				HexTx: "001122",
				Block: &btcjson.BlockDetails{
					Height: 100000,
					Hash:   "123",
					Index:  0,
					Time:   12345678,
				},
			},
		},
		{
			name: "redeemingtx",
			newNtfn: func() (interface{}, error) {
				return btcjson.NewCmd("redeemingtx", "001122", `{"height":100000,"hash":"123","index":0,"time":12345678}`)
			},
			staticNtfn: func() interface{} {
				blockDetails := btcjson.BlockDetails{
					Height: 100000,
					Hash:   "123",
					Index:  0,
					Time:   12345678,
				}
				return btcjson.NewRedeemingTxNtfn("001122", &blockDetails)
			},
			marshalled: `{"jsonrpc":"1.0","method":"redeemingtx","params":["001122",{"height":100000,"hash":"123","index":0,"time":12345678}],"id":null}`,
			unmarshalled: &btcjson.RedeemingTxNtfn{
				HexTx: "001122",
				Block: &btcjson.BlockDetails{
					Height: 100000,
					Hash:   "123",
					Index:  0,
					Time:   12345678,
				},
			},
		},
		{
			name: "rescanfinished",
			newNtfn: func() (interface{}, error) {
				return btcjson.NewCmd("rescanfinished", "123", 100000, 12345678)
			},
			staticNtfn: func() interface{} {
				return btcjson.NewRescanFinishedNtfn("123", 100000, 12345678)
			},
			marshalled: `{"jsonrpc":"1.0","method":"rescanfinished","params":["123",100000,12345678],"id":null}`,
			unmarshalled: &btcjson.RescanFinishedNtfn{
				Hash:   "123",
				Height: 100000,
				Time:   12345678,
			},
		},
		{
			name: "rescanprogress",
			newNtfn: func() (interface{}, error) {
				return btcjson.NewCmd("rescanprogress", "123", 100000, 12345678)
			},
			staticNtfn: func() interface{} {
				return btcjson.NewRescanProgressNtfn("123", 100000, 12345678)
			},
			marshalled: `{"jsonrpc":"1.0","method":"rescanprogress","params":["123",100000,12345678],"id":null}`,
			unmarshalled: &btcjson.RescanProgressNtfn{
				Hash:   "123",
				Height: 100000,
				Time:   12345678,
			},
		},
		{
			name: "txaccepted",
			newNtfn: func() (interface{}, error) {
				return btcjson.NewCmd("txaccepted", "123", 1.5)
			},
			staticNtfn: func() interface{} {
				return btcjson.NewTxAcceptedNtfn("123", 1.5)
			},
			marshalled: `{"jsonrpc":"1.0","method":"txaccepted","params":["123",1.5],"id":null}`,
			unmarshalled: &btcjson.TxAcceptedNtfn{
				TxID:   "123",
				Amount: 1.5,
			},
		},
		{
			name: "txacceptedverbose",
			newNtfn: func() (interface{}, error) {
				return btcjson.NewCmd("txacceptedverbose", `{"hex":"001122","txid":"123","version":1,"locktime":4294967295,"vin":null,"vout":null,"confirmations":0}`)
			},
			staticNtfn: func() interface{} {
				txResult := btcjson.TxRawResult{
					Hex:           "001122",
					Txid:          "123",
					Version:       1,
					LockTime:      4294967295,
					Vin:           nil,
					Vout:          nil,
					Confirmations: 0,
				}
				return btcjson.NewTxAcceptedVerboseNtfn(txResult)
			},
			marshalled: `{"jsonrpc":"1.0","method":"txacceptedverbose","params":[{"hex":"001122","txid":"123","version":1,"locktime":4294967295,"vin":null,"vout":null}],"id":null}`,
			unmarshalled: &btcjson.TxAcceptedVerboseNtfn{
				RawTx: btcjson.TxRawResult{
					Hex:           "001122",
					Txid:          "123",
					Version:       1,
					LockTime:      4294967295,
					Vin:           nil,
					Vout:          nil,
					Confirmations: 0,
				},
			},
		},
		{
			name: "relevanttxaccepted",
			newNtfn: func() (interface{}, error) {
				return btcjson.NewCmd("relevanttxaccepted", "001122")
			},
			staticNtfn: func() interface{} {
				return btcjson.NewRelevantTxAcceptedNtfn("001122")
			},
			marshalled: `{"jsonrpc":"1.0","method":"relevanttxaccepted","params":["001122"],"id":null}`,
			unmarshalled: &btcjson.RelevantTxAcceptedNtfn{
				Transaction: "001122",
			},
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
//封送由新静态创建的通知
//创建函数。通知的ID为零。
		marshalled, err := btcjson.MarshalCmd(nil, test.staticNtfn())
		if err != nil {
			t.Errorf("MarshalCmd #%d (%s) unexpected error: %v", i,
				test.name, err)
			continue
		}

		if !bytes.Equal(marshalled, []byte(test.marshalled)) {
			t.Errorf("Test #%d (%s) unexpected marshalled data - "+
				"got %s, want %s", i, test.name, marshalled,
				test.marshalled)
			continue
		}

//确保通过
//常规的新通知创建函数。
		cmd, err := test.newNtfn()
		if err != nil {
			t.Errorf("Test #%d (%s) unexpected NewCmd error: %v ",
				i, test.name, err)
		}

//封送由常规new创建的通知
//通知创建功能。身份证是零
//通知。
		marshalled, err = btcjson.MarshalCmd(nil, cmd)
		if err != nil {
			t.Errorf("MarshalCmd #%d (%s) unexpected error: %v", i,
				test.name, err)
			continue
		}

		if !bytes.Equal(marshalled, []byte(test.marshalled)) {
			t.Errorf("Test #%d (%s) unexpected marshalled data - "+
				"got %s, want %s", i, test.name, marshalled,
				test.marshalled)
			continue
		}

		var request btcjson.Request
		if err := json.Unmarshal(marshalled, &request); err != nil {
			t.Errorf("Test #%d (%s) unexpected error while "+
				"unmarshalling JSON-RPC request: %v", i,
				test.name, err)
			continue
		}

		cmd, err = btcjson.UnmarshalCmd(&request)
		if err != nil {
			t.Errorf("UnmarshalCmd #%d (%s) unexpected error: %v", i,
				test.name, err)
			continue
		}

		if !reflect.DeepEqual(cmd, test.unmarshalled) {
			t.Errorf("Test #%d (%s) unexpected unmarshalled command "+
				"- got %s, want %s", i, test.name,
				fmt.Sprintf("(%T) %+[1]v", cmd),
				fmt.Sprintf("(%T) %+[1]v\n", test.unmarshalled))
			continue
		}
	}
}
