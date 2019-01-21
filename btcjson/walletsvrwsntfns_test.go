
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

package btcjson_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/btcsuite/btcd/btcjson"
)

//testwalletsvrwsntfs测试所有特定于链服务器WebSocket的
//通知封送和取消封送到有效结果中包括处理
//在marshalled命令中省略了可选字段，而可选字段
//具有默认值的字段具有未编排命令的默认值。
func TestWalletSvrWsNtfns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		newNtfn      func() (interface{}, error)
		staticNtfn   func() interface{}
		marshalled   string
		unmarshalled interface{}
	}{
		{
			name: "accountbalance",
			newNtfn: func() (interface{}, error) {
				return btcjson.NewCmd("accountbalance", "acct", 1.25, true)
			},
			staticNtfn: func() interface{} {
				return btcjson.NewAccountBalanceNtfn("acct", 1.25, true)
			},
			marshalled: `{"jsonrpc":"1.0","method":"accountbalance","params":["acct",1.25,true],"id":null}`,
			unmarshalled: &btcjson.AccountBalanceNtfn{
				Account:   "acct",
				Balance:   1.25,
				Confirmed: true,
			},
		},
		{
			name: "btcdconnected",
			newNtfn: func() (interface{}, error) {
				return btcjson.NewCmd("btcdconnected", true)
			},
			staticNtfn: func() interface{} {
				return btcjson.NewBtcdConnectedNtfn(true)
			},
			marshalled: `{"jsonrpc":"1.0","method":"btcdconnected","params":[true],"id":null}`,
			unmarshalled: &btcjson.BtcdConnectedNtfn{
				Connected: true,
			},
		},
		{
			name: "walletlockstate",
			newNtfn: func() (interface{}, error) {
				return btcjson.NewCmd("walletlockstate", true)
			},
			staticNtfn: func() interface{} {
				return btcjson.NewWalletLockStateNtfn(true)
			},
			marshalled: `{"jsonrpc":"1.0","method":"walletlockstate","params":[true],"id":null}`,
			unmarshalled: &btcjson.WalletLockStateNtfn{
				Locked: true,
			},
		},
		{
			name: "newtx",
			newNtfn: func() (interface{}, error) {
				return btcjson.NewCmd("newtx", "acct", `{"account":"acct","address":"1Address","category":"send","amount":1.5,"bip125-replaceable":"unknown","fee":0.0001,"confirmations":1,"trusted":true,"txid":"456","walletconflicts":[],"time":12345678,"timereceived":12345876,"vout":789,"otheraccount":"otheracct"}`)
			},
			staticNtfn: func() interface{} {
				result := btcjson.ListTransactionsResult{
					Abandoned:         false,
					Account:           "acct",
					Address:           "1Address",
					BIP125Replaceable: "unknown",
					Category:          "send",
					Amount:            1.5,
					Fee:               btcjson.Float64(0.0001),
					Confirmations:     1,
					TxID:              "456",
					WalletConflicts:   []string{},
					Time:              12345678,
					TimeReceived:      12345876,
					Trusted:           true,
					Vout:              789,
					OtherAccount:      "otheracct",
				}
				return btcjson.NewNewTxNtfn("acct", result)
			},
			marshalled: `{"jsonrpc":"1.0","method":"newtx","params":["acct",{"abandoned":false,"account":"acct","address":"1Address","amount":1.5,"bip125-replaceable":"unknown","category":"send","confirmations":1,"fee":0.0001,"time":12345678,"timereceived":12345876,"trusted":true,"txid":"456","vout":789,"walletconflicts":[],"otheraccount":"otheracct"}],"id":null}`,
			unmarshalled: &btcjson.NewTxNtfn{
				Account: "acct",
				Details: btcjson.ListTransactionsResult{
					Abandoned:         false,
					Account:           "acct",
					Address:           "1Address",
					BIP125Replaceable: "unknown",
					Category:          "send",
					Amount:            1.5,
					Fee:               btcjson.Float64(0.0001),
					Confirmations:     1,
					TxID:              "456",
					WalletConflicts:   []string{},
					Time:              12345678,
					TimeReceived:      12345876,
					Trusted:           true,
					Vout:              789,
					OtherAccount:      "otheracct",
				},
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
