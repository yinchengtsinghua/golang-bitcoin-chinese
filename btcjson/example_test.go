
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
	"encoding/json"
	"fmt"

	"github.com/btcsuite/btcd/btcjson"
)

//此示例演示如何创建命令并将其封送到JSON-RPC中。
//请求。
func ExampleMarshalCmd() {
//创建新的getblock命令。注意nil参数表示
//使用该字段的默认参数。这是常见的
//此包中所有新的<foo>cmd函数中使用的模式
//可选字段。另外，请注意对btcjson.bool的调用，它是
//从基元中创建指针的便利函数
//可选参数。
	blockHash := "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f"
	gbCmd := btcjson.NewGetBlockCmd(blockHash, btcjson.Bool(false), nil)

//将命令封送为适合发送到RPC的格式
//服务器。通常，客户机会在此处增加ID，即
//请求以便识别响应。
	id := 1
	marshalledBytes, err := btcjson.MarshalCmd(id, gbCmd)
	if err != nil {
		fmt.Println(err)
		return
	}

//显示已封送的命令。通常情况下，这会被送过去
//到RPC服务器的连线，但在本例中，只显示它。
	fmt.Printf("%s\n", marshalledBytes)

//输出：
//“jsonrpc”：“1.0”，“method”：“getblock”，“params”：[“000000000019689c085ae165831e934f763ae46a2a6c172b3f1b60a8ce26f”，false），“id”：1_
}

//此示例演示如何取消JSON-RPC请求的标记，然后
//将具体请求解组为具体命令。
func ExampleUnmarshalCmd() {
//通常情况下，这将从电线上读取，但在本例中，
//为了清晰起见，这里硬编码。
	data := []byte(`{"jsonrpc":"1.0","method":"getblock","params":["000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f",false],"id":1}`)

//将线路中的原始字节解组为JSON-RPC请求。
	var request btcjson.Request
	if err := json.Unmarshal(data, &request); err != nil {
		fmt.Println(err)
		return
	}

//通常不需要直接检查请求字段
//就像这样，因为调用者已经知道基于期望的响应
//根据它发出的命令。然而，这是为了证明
//解组过程有两个步骤。
	if request.ID == nil {
		fmt.Println("Unexpected notification")
		return
	}
	if request.Method != "getblock" {
		fmt.Println("Unexpected method")
		return
	}

//将请求解封为一个具体的命令。
	cmd, err := btcjson.UnmarshalCmd(&request)
	if err != nil {
		fmt.Println(err)
		return
	}

//类型将命令断言为适当的类型。
	gbCmd, ok := cmd.(*btcjson.GetBlockCmd)
	if !ok {
		fmt.Printf("Incorrect command type: %T\n", cmd)
		return
	}

//显示具体命令中的字段。
	fmt.Println("Hash:", gbCmd.Hash)
	fmt.Println("Verbose:", *gbCmd.Verbose)
	fmt.Println("VerboseTx:", *gbCmd.VerboseTx)

//输出：
//哈希：000000000019689C085AE165831E934F763AE46A2A6C172B3F1B60A8CE26F
//冗长的：虚假的
//verbosetx:错误
}

//此示例演示如何封送JSON-RPC响应。
func ExampleMarshalResponse() {
//封送新的JSON-RPC响应。例如，这是一个响应
//到GetBlockHeight请求。
	marshalledBytes, err := btcjson.MarshalResponse(1, 350001, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

//显示已编组的响应。通常情况下，这会被发送
//通过连接到RPC客户机，但在本例中，只显示
//它。
	fmt.Printf("%s\n", marshalledBytes)

//输出：
//“result”：350001，“error”：空，“id”：1_
}

//此示例演示如何取消标记JSON-RPC响应，然后
//在对具体类型的响应中取消对结果字段的标记。
func Example_unmarshalResponse() {
//通常情况下，这将从电线上读取，但在本例中，
//为了清晰起见，这里硬编码。这是对
//GetBlockHeight请求。
	data := []byte(`{"result":350001,"error":null,"id":1}`)

//将线路中的原始字节解组为JSON-RPC响应。
	var response btcjson.Response
	if err := json.Unmarshal(data, &response); err != nil {
		fmt.Println("Malformed JSON-RPC response:", err)
		return
	}

//检查服务器的响应是否有错误。例如，
//如果无效/未知的块哈希为
//请求。
	if response.Error != nil {
		fmt.Println(response.Error)
		return
	}

//将结果取消标记为响应的预期类型。
	var blockHeight int32
	if err := json.Unmarshal(response.Result, &blockHeight); err != nil {
		fmt.Printf("Unexpected result type: %T\n", response.Result)
		return
	}
	fmt.Println("Block height:", blockHeight)

//输出：
//块高：350001
}
