
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
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
)

//usageflag定义用于指定有关
//可以使用命令的情况。
type UsageFlag uint32

const (
//UFwalletOnly指示该命令只能与RPC一起使用
//支持钱包命令的服务器。
	UFWalletOnly UsageFlag = 1 << iota

//ufwebsocketOnly指示只有在
//通过WebSockets与RPC服务器通信。这通常
//适用于通知和通知注册功能
//因为Neiher在使用单次发送的HTTP-Post请求时发出了。
	UFWebsocketOnly

//ufnotification表示该命令实际上是一个通知。
//这意味着当它被编组时，ID必须为零。
	UFNotification

//highestusageflagbit是最大使用标志位，用于
//桁条和测试，以确保上述所有常数
//测试。
	highestUsageFlagBit
)

//将usageflag值映射回其常量名，以便进行漂亮的打印。
var usageFlagStrings = map[UsageFlag]string{
	UFWalletOnly:    "UFWalletOnly",
	UFWebsocketOnly: "UFWebsocketOnly",
	UFNotification:  "UFNotification",
}

//字符串以可读形式返回usageflag。
func (fl UsageFlag) String() string {
//未设置标志。
	if fl == 0 {
		return "0x0"
	}

//添加单个位标志。
	s := ""
	for flag := UFWalletOnly; flag < highestUsageFlagBit; flag <<= 1 {
		if fl&flag == flag {
			s += usageFlagStrings[flag] + "|"
			fl -= flag
		}
	}

//添加剩余值作为原始十六进制。
	s = strings.TrimRight(s, "|")
	if fl != 0 {
		s += "|0x" + strconv.FormatUint(uint64(fl), 16)
	}
	s = strings.TrimLeft(s, "|")
	return s
}

//MethodInfo跟踪每个注册方法的信息，例如
//参数信息。
type methodInfo struct {
	maxParams    int
	numReqParams int
	numOptParams int
	defaults     map[int]reflect.Value
	flags        UsageFlag
	usage        string
}

var (
//这些字段用于将已注册的类型映射到方法名。
	registerLock         sync.RWMutex
	methodToConcreteType = make(map[string]reflect.Type)
	methodToInfo         = make(map[string]methodInfo)
	concreteTypeToMethod = make(map[reflect.Type]string)
)

//basekindstring返回给定reflect.type之后的基类型
//间接通过所有指针。
func baseKindString(rt reflect.Type) string {
	numIndirects := 0
	for rt.Kind() == reflect.Ptr {
		numIndirects++
		rt = rt.Elem()
	}

	return fmt.Sprintf("%s%s", strings.Repeat("*", numIndirects), rt.Kind())
}

//IsAcceptableKind返回传递的字段类型是否受支持
//类型。它是在第一个指针间接寻址之后调用的，因此进一步的指针
//不支持。
func isAcceptableKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Chan:
		fallthrough
	case reflect.Complex64:
		fallthrough
	case reflect.Complex128:
		fallthrough
	case reflect.Func:
		fallthrough
	case reflect.Ptr:
		fallthrough
	case reflect.Interface:
		return false
	}

	return true
}

//registerCmd注册一个新命令，该命令将自动封送到和
//来自具有完整类型检查和位置参数支持的JSON-RPC。它
//还接受用于标识
//可以使用命令。
//
//默认情况下，此包自动注册所有导出的命令
//但是，使用此函数也会导出它，以便调用者可以轻松地
//注册自定义类型。
//
//类型格式非常严格，因为它需要能够自动
//从JSON-RPC 1.0封送。以下列举了要求：
//
//-提供的命令必须是指向结构的单个指针
//-必须导出所有字段
//-编组JSON中的位置参数的顺序为
//与结构定义中声明的顺序相同
//-不支持结构嵌入
//-结构字段不能是通道、函数、复杂或接口
//-提供的结构中带有指针的字段被视为可选字段
//-不支持多个间接寻址（即**int）
//-一旦遇到第一个可选字段（指针），剩余的
//字段还必须是位置所需的可选字段（指针）。
//帕拉姆
//-具有“jsonrpcdefault”结构标记的字段必须是可选字段
//（指针）
//
//注意：此函数只需要能够检查
//传递了结构，因此它不需要是实际实例。因此，它
//建议只将nil指针转换传递给适当的类型。
//例如，（*FOOCMD）（无）。
func RegisterCmd(method string, cmd interface{}, flags UsageFlag) error {
	registerLock.Lock()
	defer registerLock.Unlock()

	if _, ok := methodToConcreteType[method]; ok {
		str := fmt.Sprintf("method %q is already registered", method)
		return makeError(ErrDuplicateMethod, str)
	}

//确保没有指定无法识别的标志位。
	if ^(highestUsageFlagBit-1)&flags != 0 {
		str := fmt.Sprintf("invalid usage flags specified for method "+
			"%s: %v", method, flags)
		return makeError(ErrInvalidUsageFlags, str)
	}

	rtp := reflect.TypeOf(cmd)
	if rtp.Kind() != reflect.Ptr {
		str := fmt.Sprintf("type must be *struct not '%s (%s)'", rtp,
			rtp.Kind())
		return makeError(ErrInvalidType, str)
	}
	rt := rtp.Elem()
	if rt.Kind() != reflect.Struct {
		str := fmt.Sprintf("type must be *struct not '%s (*%s)'",
			rtp, rt.Kind())
		return makeError(ErrInvalidType, str)
	}

//枚举结构字段以验证它们并收集参数
//信息。
	numFields := rt.NumField()
	numOptFields := 0
	defaults := make(map[int]reflect.Value)
	for i := 0; i < numFields; i++ {
		rtf := rt.Field(i)
		if rtf.Anonymous {
			str := fmt.Sprintf("embedded fields are not supported "+
				"(field name: %q)", rtf.Name)
			return makeError(ErrEmbeddedType, str)
		}
		if rtf.PkgPath != "" {
			str := fmt.Sprintf("unexported fields are not supported "+
				"(field name: %q)", rtf.Name)
			return makeError(ErrUnexportedField, str)
		}

//不允许不能使用JSON编码的类型。同时，确定
//如果字段是可选的，则基于它是指针。
		var isOptional bool
		switch kind := rtf.Type.Kind(); kind {
		case reflect.Ptr:
			isOptional = true
			kind = rtf.Type.Elem().Kind()
			fallthrough
		default:
			if !isAcceptableKind(kind) {
				str := fmt.Sprintf("unsupported field type "+
					"'%s (%s)' (field name %q)", rtf.Type,
					baseKindString(rtf.Type), rtf.Name)
				return makeError(ErrUnsupportedFieldType, str)
			}
		}

//对可选字段进行计数，并确保在
//第一个可选字段也是可选的。
		if isOptional {
			numOptFields++
		} else {
			if numOptFields > 0 {
				str := fmt.Sprintf("all fields after the first "+
					"optional field must also be optional "+
					"(field name %q)", rtf.Name)
				return makeError(ErrNonOptionalField, str)
			}
		}

//确保默认值可以取消编组到类型中
//并且该默认值仅为可选字段指定。
		if tag := rtf.Tag.Get("jsonrpcdefault"); tag != "" {
			if !isOptional {
				str := fmt.Sprintf("required fields must not "+
					"have a default specified (field name "+
					"%q)", rtf.Name)
				return makeError(ErrNonOptionalDefault, str)
			}

			rvf := reflect.New(rtf.Type.Elem())
			err := json.Unmarshal([]byte(tag), rvf.Interface())
			if err != nil {
				str := fmt.Sprintf("default value of %q is "+
					"the wrong type (field name %q)", tag,
					rtf.Name)
				return makeError(ErrMismatchedDefault, str)
			}
			defaults[i] = rvf
		}
	}

//更新注册地图。
	methodToConcreteType[method] = rtp
	methodToInfo[method] = methodInfo{
		maxParams:    numFields,
		numReqParams: numFields - numOptFields,
		numOptParams: numOptFields,
		defaults:     defaults,
		flags:        flags,
	}
	concreteTypeToMethod[rtp] = method
	return nil
}

//mustregisterCmd执行与registerCmd相同的功能，但它会死机
//如果有错误。只能从package init调用此函数
//功能。
func MustRegisterCmd(method string, cmd interface{}, flags UsageFlag) {
	if err := RegisterCmd(method, cmd, flags); err != nil {
		panic(fmt.Sprintf("failed to register type %q: %v\n", method,
			err))
	}
}

//RegisteredCmdMethods返回所有已注册的方法的排序列表
//命令。
func RegisteredCmdMethods() []string {
	registerLock.Lock()
	defer registerLock.Unlock()

	methods := make([]string, 0, len(methodToInfo))
	for k := range methodToInfo {
		methods = append(methods, k)
	}

	sort.Sort(sort.StringSlice(methods))
	return methods
}
