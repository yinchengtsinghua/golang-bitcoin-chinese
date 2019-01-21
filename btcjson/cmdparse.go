
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
	"strconv"
	"strings"
)

//makeparams为给定结构创建接口值切片。
func makeParams(rt reflect.Type, rv reflect.Value) []interface{} {
	numFields := rt.NumField()
	params := make([]interface{}, 0, numFields)
	for i := 0; i < numFields; i++ {
		rtf := rt.Field(i)
		rvf := rv.Field(i)
		if rtf.Type.Kind() == reflect.Ptr {
			if rvf.IsNil() {
				break
			}
			rvf.Elem()
		}
		params = append(params, rvf.Interface())
	}

	return params
}

//marshalCmd将传递的命令封送到json-rpc请求字节片，
//适用于传输到RPC服务器。提供的命令类型
//必须是注册类型。此包提供的所有命令都是
//默认注册。
func MarshalCmd(id interface{}, cmd interface{}) ([]byte, error) {
//查找命令类型，如果未注册，则出错。
	rt := reflect.TypeOf(cmd)
	registerLock.RLock()
	method, ok := concreteTypeToMethod[rt]
	registerLock.RUnlock()
	if !ok {
		str := fmt.Sprintf("%q is not registered", method)
		return nil, makeError(ErrUnregisteredMethod, str)
	}

//提供的命令不能为零。
	rv := reflect.ValueOf(cmd)
	if rv.IsNil() {
		str := "the specified command is nil"
		return nil, makeError(ErrInvalidType, str)
	}

//按结构字段的顺序创建接口值切片
//同时将指针字段作为可选参数，只添加
//如果不是零的话。
	params := makeParams(rt.Elem(), rv.Elem())

//生成并封送最终的JSON-RPC请求。
	rawCmd, err := NewRequest(id, method, params)
	if err != nil {
		return nil, err
	}
	return json.Marshal(rawCmd)
}

//checkNumParams确保提供的参数数量至少为最小值
//命令所需的数字，小于允许的最大值。
func checkNumParams(numParams int, info *methodInfo) error {
	if numParams < info.numReqParams || numParams > info.maxParams {
		if info.numReqParams == info.maxParams {
			str := fmt.Sprintf("wrong number of params (expected "+
				"%d, received %d)", info.numReqParams,
				numParams)
			return makeError(ErrNumParams, str)
		}

		str := fmt.Sprintf("wrong number of params (expected "+
			"between %d and %d, received %d)", info.numReqParams,
			info.maxParams, numParams)
		return makeError(ErrNumParams, str)
	}

	return nil
}

//populatedefaults将默认值填充到任何剩余的可选结构中
//没有显式提供参数的字段。打电话的人应该
//以前检查过正在传递的参数的数目为
//最少需要的参数数量，以避免在此过程中进行不必要的工作
//函数，但由于必需字段从来没有默认值，因此它将起作用
//不用支票也可以。
func populateDefaults(numParams int, info *methodInfo, rv reflect.Value) {
//当所提供的参数中没有剩余参数时，
//任何剩余的结构字段都必须是可选的。因此，填充它们
//根据需要使用其关联的默认值。
	for i := numParams; i < info.maxParams; i++ {
		rvf := rv.Field(i)
		if defaultVal, ok := info.defaults[i]; ok {
			rvf.Set(defaultVal)
		}
	}
}

//unmashalcmd将json-rpc请求解封为适当的具体命令
//只要包含在封送请求中的方法类型是
//注册的。
func UnmarshalCmd(r *Request) (interface{}, error) {
	registerLock.RLock()
	rtp, ok := methodToConcreteType[r.Method]
	info := methodToInfo[r.Method]
	registerLock.RUnlock()
	if !ok {
		str := fmt.Sprintf("%q is not registered", r.Method)
		return nil, makeError(ErrUnregisteredMethod, str)
	}
	rt := rtp.Elem()
	rvp := reflect.New(rt)
	rv := rvp.Elem()

//确保参数数量正确。
	numParams := len(r.Params)
	if err := checkNumParams(numParams, &info); err != nil {
		return nil, err
	}

//循环遍历每个结构字段并取消关联
//参数。
	for i := 0; i < numParams; i++ {
		rvf := rv.Field(i)
//将参数解组到结构字段中。
		concreteVal := rvf.Addr().Interface()
		if err := json.Unmarshal(r.Params[i], &concreteVal); err != nil {
//最常见的错误是类型错误，所以
//明确地检测错误并使其更好。
			fieldName := strings.ToLower(rt.Field(i).Name)
			if jerr, ok := err.(*json.UnmarshalTypeError); ok {
				str := fmt.Sprintf("parameter #%d '%s' must "+
					"be type %v (got %v)", i+1, fieldName,
					jerr.Type, jerr.Value)
				return nil, makeError(ErrInvalidType, str)
			}

//回退到显示基础错误。
			str := fmt.Sprintf("parameter #%d '%s' failed to "+
				"unmarshal: %v", i+1, fieldName, err)
			return nil, makeError(ErrInvalidType, str)
		}
	}

//当提供的参数少于
//参数，任何剩余结构字段都必须是可选的。因此，填充
//根据需要使用它们的关联默认值。
	if numParams < info.maxParams {
		populateDefaults(numParams, &info, rv)
	}

	return rvp.Interface(), nil
}

//IsNumeric返回传递的反射类型是有符号的还是无符号的
//任何数量级的整数或任何数量级的浮点。
func isNumeric(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Uint64, reflect.Float32, reflect.Float64:

		return true
	}

	return false
}

//typesmaybecompatible返回源类型是否可以
//分配给目标类型。这是一个相对快速的
//选中可清除明显无效的转换。
func typesMaybeCompatible(dest reflect.Type, src reflect.Type) bool {
//同样的类型显然是兼容的。
	if dest == src {
		return true
	}

//当这两种类型都是数字时，它们可能是兼容的。
	srcKind := src.Kind()
	destKind := dest.Kind()
	if isNumeric(destKind) && isNumeric(srcKind) {
		return true
	}

	if srcKind == reflect.String {
//字符串可能会转换为数字类型。
		if isNumeric(destKind) {
			return true
		}

		switch destKind {
//字符串可能通过以下方式转换为bools
//strconv.parsebool.
		case reflect.Bool:
			return true

//字符串可以转换为具有
//字符串的基础类型。
		case reflect.String:
			return true

//字符串可能会被转换为数组、切片，
//通过json.unmashal构造和映射。
		case reflect.Array, reflect.Slice, reflect.Struct, reflect.Map:
			return true
		}
	}

	return false
}

//baseType返回在对所有参数进行间接寻址之后的参数类型
//指针以及需要多少间接指令。
func baseType(arg reflect.Type) (reflect.Type, int) {
	var numIndirects int
	for arg.Kind() == reflect.Ptr {
		arg = arg.Elem()
		numIndirects++
	}
	return arg, numIndirects
}

//assignfield是处理newcmd函数的主要工作程序
//将提供的源值分配给目标字段。它支持
//直接类型分配、间接、数字类型转换，以及
//通过以下方式将字符串解组为数组、切片、结构和映射
//解说员。
func assignField(paramNum int, fieldName string, dest reflect.Value, src reflect.Value) error {
//当类型不可能兼容时，现在就出错。
	destBaseType, destIndirects := baseType(dest.Type())
	srcBaseType, srcIndirects := baseType(src.Type())
	if !typesMaybeCompatible(destBaseType, srcBaseType) {
		str := fmt.Sprintf("parameter #%d '%s' must be type %v (got "+
			"%v)", paramNum, fieldName, destBaseType, srcBaseType)
		return makeError(ErrInvalidType, str)
	}

//检查是否可以简单地将dest设置为提供的源。
//当基类型相同或两者都是时，就会出现这种情况。
//可以直接指向相同的指针，而无需
//为目标字段创建指针。
	if destBaseType == srcBaseType && srcIndirects >= destIndirects {
		for i := 0; i < srcIndirects-destIndirects; i++ {
			src = src.Elem()
		}
		dest.Set(src)
		return nil
	}

//当目的地的间接数比源多时，额外的
//必须创建指针。只创建足够的指针
//与源的间接级别相同，因此dest可以简单地
//当类型相同时设置为提供的源。
	destIndirectsRemaining := destIndirects
	if destIndirects > srcIndirects {
		indirectDiff := destIndirects - srcIndirects
		for i := 0; i < indirectDiff; i++ {
			dest.Set(reflect.New(dest.Type().Elem()))
			dest = dest.Elem()
			destIndirectsRemaining--
		}
	}

	if destBaseType == srcBaseType {
		dest.Set(src)
		return nil
	}

//生成获取基dest类型所需的任何剩余指针，因为
//无法执行上述直接分配，已完成转换
//与基类型相反。
	for i := 0; i < destIndirectsRemaining; i++ {
		dest.Set(reflect.New(dest.Type().Elem()))
		dest = dest.Elem()
	}

//间接到基源值。
	for src.Kind() == reflect.Ptr {
		src = src.Elem()
	}

//执行支持的类型转换。
	switch src.Kind() {
//源值是各种大小的有符号整数。
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Int64:

		switch dest.Kind() {
//目的地是各种大小的有符号整数。
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
			reflect.Int64:

			srcInt := src.Int()
			if dest.OverflowInt(srcInt) {
				str := fmt.Sprintf("parameter #%d '%s' "+
					"overflows destination type %v",
					paramNum, fieldName, destBaseType)
				return makeError(ErrInvalidType, str)
			}

			dest.SetInt(srcInt)

//目的地是一个不同大小的无符号整数。
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
			reflect.Uint64:

			srcInt := src.Int()
			if srcInt < 0 || dest.OverflowUint(uint64(srcInt)) {
				str := fmt.Sprintf("parameter #%d '%s' "+
					"overflows destination type %v",
					paramNum, fieldName, destBaseType)
				return makeError(ErrInvalidType, str)
			}
			dest.SetUint(uint64(srcInt))

		default:
			str := fmt.Sprintf("parameter #%d '%s' must be type "+
				"%v (got %v)", paramNum, fieldName, destBaseType,
				srcBaseType)
			return makeError(ErrInvalidType, str)
		}

//源值是各种大小的无符号整数。
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Uint64:

		switch dest.Kind() {
//目的地是各种大小的有符号整数。
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
			reflect.Int64:

			srcUint := src.Uint()
			if srcUint > uint64(1<<63)-1 {
				str := fmt.Sprintf("parameter #%d '%s' "+
					"overflows destination type %v",
					paramNum, fieldName, destBaseType)
				return makeError(ErrInvalidType, str)
			}
			if dest.OverflowInt(int64(srcUint)) {
				str := fmt.Sprintf("parameter #%d '%s' "+
					"overflows destination type %v",
					paramNum, fieldName, destBaseType)
				return makeError(ErrInvalidType, str)
			}
			dest.SetInt(int64(srcUint))

//目的地是一个不同大小的无符号整数。
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
			reflect.Uint64:

			srcUint := src.Uint()
			if dest.OverflowUint(srcUint) {
				str := fmt.Sprintf("parameter #%d '%s' "+
					"overflows destination type %v",
					paramNum, fieldName, destBaseType)
				return makeError(ErrInvalidType, str)
			}
			dest.SetUint(srcUint)

		default:
			str := fmt.Sprintf("parameter #%d '%s' must be type "+
				"%v (got %v)", paramNum, fieldName, destBaseType,
				srcBaseType)
			return makeError(ErrInvalidType, str)
		}

//源值是一个浮点。
	case reflect.Float32, reflect.Float64:
		destKind := dest.Kind()
		if destKind != reflect.Float32 && destKind != reflect.Float64 {
			str := fmt.Sprintf("parameter #%d '%s' must be type "+
				"%v (got %v)", paramNum, fieldName, destBaseType,
				srcBaseType)
			return makeError(ErrInvalidType, str)
		}

		srcFloat := src.Float()
		if dest.OverflowFloat(srcFloat) {
			str := fmt.Sprintf("parameter #%d '%s' overflows "+
				"destination type %v", paramNum, fieldName,
				destBaseType)
			return makeError(ErrInvalidType, str)
		}
		dest.SetFloat(srcFloat)

//源值是字符串。
	case reflect.String:
		switch dest.Kind() {
//字符串>布尔
		case reflect.Bool:
			b, err := strconv.ParseBool(src.String())
			if err != nil {
				str := fmt.Sprintf("parameter #%d '%s' must "+
					"parse to a %v", paramNum, fieldName,
					destBaseType)
				return makeError(ErrInvalidType, str)
			}
			dest.SetBool(b)

//字符串->大小不同的有符号整数。
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
			reflect.Int64:

			srcInt, err := strconv.ParseInt(src.String(), 0, 0)
			if err != nil {
				str := fmt.Sprintf("parameter #%d '%s' must "+
					"parse to a %v", paramNum, fieldName,
					destBaseType)
				return makeError(ErrInvalidType, str)
			}
			if dest.OverflowInt(srcInt) {
				str := fmt.Sprintf("parameter #%d '%s' "+
					"overflows destination type %v",
					paramNum, fieldName, destBaseType)
				return makeError(ErrInvalidType, str)
			}
			dest.SetInt(srcInt)

//字符串->大小不同的无符号整数。
		case reflect.Uint, reflect.Uint8, reflect.Uint16,
			reflect.Uint32, reflect.Uint64:

			srcUint, err := strconv.ParseUint(src.String(), 0, 0)
			if err != nil {
				str := fmt.Sprintf("parameter #%d '%s' must "+
					"parse to a %v", paramNum, fieldName,
					destBaseType)
				return makeError(ErrInvalidType, str)
			}
			if dest.OverflowUint(srcUint) {
				str := fmt.Sprintf("parameter #%d '%s' "+
					"overflows destination type %v",
					paramNum, fieldName, destBaseType)
				return makeError(ErrInvalidType, str)
			}
			dest.SetUint(srcUint)

//字符串->不同大小的浮点。
		case reflect.Float32, reflect.Float64:
			srcFloat, err := strconv.ParseFloat(src.String(), 0)
			if err != nil {
				str := fmt.Sprintf("parameter #%d '%s' must "+
					"parse to a %v", paramNum, fieldName,
					destBaseType)
				return makeError(ErrInvalidType, str)
			}
			if dest.OverflowFloat(srcFloat) {
				str := fmt.Sprintf("parameter #%d '%s' "+
					"overflows destination type %v",
					paramNum, fieldName, destBaseType)
				return makeError(ErrInvalidType, str)
			}
			dest.SetFloat(srcFloat)

//string->string（类型转换）。
		case reflect.String:
			dest.SetString(src.String())

//字符串->数组、切片、结构和映射方式
//解说员。
		case reflect.Array, reflect.Slice, reflect.Struct, reflect.Map:
			concreteVal := dest.Addr().Interface()
			err := json.Unmarshal([]byte(src.String()), &concreteVal)
			if err != nil {
				str := fmt.Sprintf("parameter #%d '%s' must "+
					"be valid JSON which unsmarshals to a %v",
					paramNum, fieldName, destBaseType)
				return makeError(ErrInvalidType, str)
			}
			dest.Set(reflect.ValueOf(concreteVal).Elem())
		}
	}

	return nil
}

//newcmd提供了一种通用机制来创建一个新命令，该命令可以封送
//到JSON-RPC请求，同时遵守提供的
//方法。方法必须已与包一起注册
//它的类型定义。与导出的命令关联的所有方法
//默认情况下，此包已注册。
//
//当参数与
//和方法关联的命令结构中的基础字段，
//但是，此函数还将执行各种转换以使其
//更灵活。例如，这允许命令行参数是字符串
//不会改变。特别是，以下转换是
//支持：
//
//-任何大小的有符号或无符号整数之间的转换，只要
//值不会溢出目标类型
//-在float32和float64之间转换，只要该值不
//溢出目标类型
//-将strconv.parsebool从字符串转换为布尔值
//认识到
//-从字符串转换为任意大小的整数
//strconv.parseint和strconv.parseuint识别
//-从字符串转换为任何大小的浮点
//strconv.parsefloat识别
//-通过处理从字符串到数组、切片、结构和映射的转换
//作为封送JSON并将json.unmashal调用到
//目标字段
func NewCmd(method string, args ...interface{}) (interface{}, error) {
//查找有关所提供方法的详细信息。任何方法
//注册是一个错误。
	registerLock.RLock()
	rtp, ok := methodToConcreteType[method]
	info := methodToInfo[method]
	registerLock.RUnlock()
	if !ok {
		str := fmt.Sprintf("%q is not registered", method)
		return nil, makeError(ErrUnregisteredMethod, str)
	}

//确保参数数量正确。
	numParams := len(args)
	if err := checkNumParams(numParams, &info); err != nil {
		return nil, err
	}

//为该方法创建适当的命令类型。既然所有类型
//在注册时强制为指向结构的指针，
//现在可以安全地间接指向结构。
	rvp := reflect.New(rtp.Elem())
	rv := rvp.Elem()
	rt := rtp.Elem()

//循环遍历每个结构字段并分配关联的
//在检查其类型有效性之后将参数输入到它们中。
	for i := 0; i < numParams; i++ {
//尝试将每个参数分配给
//结构域。
		rvf := rv.Field(i)
		fieldName := strings.ToLower(rt.Field(i).Name)
		err := assignField(i+1, fieldName, rvf, reflect.ValueOf(args[i]))
		if err != nil {
			return nil, err
		}
	}

	return rvp.Interface(), nil
}
