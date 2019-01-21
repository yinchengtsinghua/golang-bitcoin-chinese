
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

import (
	"encoding/json"
	"fmt"
)

//rpcerror code表示要用作rpcerror一部分的错误代码
//它依次用于JSON-RPC响应对象。
//
//使用特定类型有助于确保不使用错误的错误。
type RPCErrorCode int

//rpc error表示用作JSON-RPC响应的一部分的错误
//对象。
type RPCError struct {
	Code    RPCErrorCode `json:"code,omitempty"`
	Message string       `json:"message,omitempty"`
}

//确保rpcerror满足内置错误接口。
var _, _ error = RPCError{}, (*RPCError)(nil)

//错误返回描述RPC错误的字符串。这使
//内置错误接口。
func (e RPCError) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

//new rpc error构造并返回一个适合的新json-rpc错误
//用于JSON-RPC响应对象。
func NewRPCError(code RPCErrorCode, message string) *RPCError {
	return &RPCError{
		Code:    code,
		Message: message,
	}
}

//IsvalidIdType检查ID字段（它可以进入任何JSON-RPC
//请求、响应或通知）有效。JSON-RPC 1.0允许
//有效的JSON类型。JSON-RPC 2.0（某些部分按比特币计价）
//允许字符串、数字或空，因此此函数限制允许的类型
//到那个名单。此函数仅在调用方是手动的情况下提供
//出于某种原因进行编组。接受此中的ID的函数
//包已调用此函数以确保提供的ID有效。
func IsValidIDType(id interface{}) bool {
	switch id.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64,
		string,
		nil:
		return true
	default:
		return false
	}
}

//请求是原始JSON-RPC1.0请求的类型。方法字段标识
//导致不同参数的特定命令类型。
//由于此包提供了
//处理创建这些命令的静态类型的命令基础结构
//请求，但是如果调用方希望导出此结构
//出于某种原因构造原始请求。
type Request struct {
	Jsonrpc string            `json:"jsonrpc"`
	Method  string            `json:"method"`
	Params  []json.RawMessage `json:"params"`
	ID      interface{}       `json:"id"`
}

//new request返回给定id的新json-rpc 1.0请求对象，
//方法和参数。参数被编组为json.rawmessage
//返回的请求对象的参数字段。此函数仅
//以防调用方出于某种原因想要构造原始请求。
//
//通常，调用方希望创建一个已注册的具体命令
//使用newCmd或new<foo>Cmd函数键入并调用marshalCmd
//函数与该命令一起生成已封送的JSON-RPC请求。
func NewRequest(id interface{}, method string, params []interface{}) (*Request, error) {
	if !IsValidIDType(id) {
		str := fmt.Sprintf("the id of type '%T' is invalid", id)
		return nil, makeError(ErrInvalidType, str)
	}

	rawParams := make([]json.RawMessage, 0, len(params))
	for _, param := range params {
		marshalledParam, err := json.Marshal(param)
		if err != nil {
			return nil, err
		}
		rawMessage := json.RawMessage(marshalledParam)
		rawParams = append(rawParams, rawMessage)
	}

	return &Request{
		Jsonrpc: "1.0",
		ID:      id,
		Method:  method,
		Params:  rawParams,
	}, nil
}

//响应是JSON-RPC响应的一般形式。结果的类型
//字段随命令的不同而不同，因此它被实现为
//接口。ID字段必须是一个指针，当
//空的。
type Response struct {
	Result json.RawMessage `json:"result"`
	Error  *RPCError       `json:"error"`
	ID     *interface{}    `json:"id"`
}

//newResponse返回给定ID的新JSON-RPC响应对象，
//封送结果和RPC错误。此功能仅在
//调用方出于某种原因想要构造原始响应。
//
//通常，调用者将希望创建完全封送的JSON-RPC
//使用MarshalResponse函数通过线路发送的响应。
func NewResponse(id interface{}, marshalledResult []byte, rpcErr *RPCError) (*Response, error) {
	if !IsValidIDType(id) {
		str := fmt.Sprintf("the id of type '%T' is invalid", id)
		return nil, makeError(ErrInvalidType, str)
	}

	pid := &id
	return &Response{
		Result: marshalledResult,
		Error:  rpcErr,
		ID:     pid,
	}, nil
}

//MarshalResponse将传递的ID、结果和RpcError封送到JSON-RPC
//适合传输到JSON-RPC客户机的响应字节片。
func MarshalResponse(id interface{}, result interface{}, rpcErr *RPCError) ([]byte, error) {
	marshalledResult, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	response, err := NewResponse(id, marshalledResult, rpcErr)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&response)
}
