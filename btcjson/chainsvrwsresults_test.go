
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2017 BTCSuite开发者
//版权所有（c）2017版权所有
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package btcjson_test

import (
	"encoding/json"
	"testing"

	"github.com/btcsuite/btcd/btcjson"
)

//testchainsvrwsresults确保具有自定义编组的任何结果
//按计划工作。
func TestChainSvrWsResults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		result   interface{}
		expected string
	}{
		{
			name: "RescannedBlock",
			result: &btcjson.RescannedBlock{
				Hash:         "blockhash",
				Transactions: []string{"serializedtx"},
			},
			expected: `{"hash":"blockhash","transactions":["serializedtx"]}`,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		marshalled, err := json.Marshal(test.result)
		if err != nil {
			t.Errorf("Test #%d (%s) unexpected error: %v", i,
				test.name, err)
			continue
		}
		if string(marshalled) != test.expected {
			t.Errorf("Test #%d (%s) unexpected marhsalled data - "+
				"got %s, want %s", i, test.name, marshalled,
				test.expected)
			continue
		}
	}
}
