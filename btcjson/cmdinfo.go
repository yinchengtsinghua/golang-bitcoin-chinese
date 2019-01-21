
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package btcjson

import (
	"fmt"
	"reflect"
	"strings"
)

//CmdMethod返回传递的命令的方法。提供的命令
//类型必须是已注册的类型。此包提供的所有命令都是
//默认注册。
func CmdMethod(cmd interface{}) (string, error) {
//查找命令类型，如果未注册，则出错。
	rt := reflect.TypeOf(cmd)
	registerLock.RLock()
	method, ok := concreteTypeToMethod[rt]
	registerLock.RUnlock()
	if !ok {
		str := fmt.Sprintf("%q is not registered", method)
		return "", makeError(ErrUnregisteredMethod, str)
	}

	return method, nil
}

//method usage flags返回传递的命令方法的使用标志。这个
//提供的方法必须与已注册的类型关联。所有命令
//默认情况下，此包提供的是注册的。
func MethodUsageFlags(method string) (UsageFlag, error) {
//查找有关所提供方法的详细信息，如果没有，则返回错误信息
//注册的。
	registerLock.RLock()
	info, ok := methodToInfo[method]
	registerLock.RUnlock()
	if !ok {
		str := fmt.Sprintf("%q is not registered", method)
		return 0, makeError(ErrUnregisteredMethod, str)
	}

	return info.flags, nil
}

//substructusage返回一个字符串，用于给定
//子结构。请注意，这是专门针对由
//结构（或结构的数组/切片），而不是顶级命令
//结构。
//
//包含jsonrpcusage结构标记的任何字段都将使用该标记，而不是
//正在自动生成。
func subStructUsage(structType reflect.Type) string {
	numFields := structType.NumField()
	fieldUsages := make([]string, 0, numFields)
	for i := 0; i < structType.NumField(); i++ {
		rtf := structType.Field(i)

//当字段具有指定的jsonrpcusage结构标记时，请使用
//而不是自动生成。
		if tag := rtf.Tag.Get("jsonrpcusage"); tag != "" {
			fieldUsages = append(fieldUsages, tag)
			continue
		}

//在考虑时为字段创建名称/值条目
//字段的类型。并非所有可能的类型都包括在内
//在这里，当其中一种没有特别涉及的类型是
//遇到此问题时，字段名只需重新用于该值。
		fieldName := strings.ToLower(rtf.Name)
		fieldValue := fieldName
		fieldKind := rtf.Type.Kind()
		switch {
		case isNumeric(fieldKind):
			if fieldKind == reflect.Float32 || fieldKind == reflect.Float64 {
				fieldValue = "n.nnn"
			} else {
				fieldValue = "n"
			}
		case fieldKind == reflect.String:
			fieldValue = `"value"`

		case fieldKind == reflect.Struct:
			fieldValue = subStructUsage(rtf.Type)

		case fieldKind == reflect.Array || fieldKind == reflect.Slice:
			fieldValue = subArrayUsage(rtf.Type, fieldName)
		}

		usage := fmt.Sprintf("%q:%s", fieldName, fieldValue)
		fieldUsages = append(fieldUsages, usage)
	}

	return fmt.Sprintf("{%s}", strings.Join(fieldUsages, ","))
}

//SubarrayUsage返回一个字符串，用于给定
//数组或切片。它还包含将复数字段名转换为
//单数，因此生成的使用字符串读起来更好。
func subArrayUsage(arrayType reflect.Type, fieldName string) string {
//将复数字段名转换为单数。只适用于英语。
	singularFieldName := fieldName
	if strings.HasSuffix(fieldName, "ies") {
		singularFieldName = strings.TrimSuffix(fieldName, "ies")
		singularFieldName = singularFieldName + "y"
	} else if strings.HasSuffix(fieldName, "es") {
		singularFieldName = strings.TrimSuffix(fieldName, "es")
	} else if strings.HasSuffix(fieldName, "s") {
		singularFieldName = strings.TrimSuffix(fieldName, "s")
	}

	elemType := arrayType.Elem()
	switch elemType.Kind() {
	case reflect.String:
		return fmt.Sprintf("[%q,...]", singularFieldName)

	case reflect.Struct:
		return fmt.Sprintf("[%s,...]", subStructUsage(elemType))
	}

//返回到只以数组语法显示字段名。
	return fmt.Sprintf(`[%s,...]`, singularFieldName)
}

//FieldUsage返回一个字符串，用于结构的单行用法
//命令字段。
//
//包含jsonrpcusage结构标记的任何字段都将使用该标记，而不是
//正在自动生成。
func fieldUsage(structField reflect.StructField, defaultVal *reflect.Value) string {
//当字段指定了jsonrpcusage结构标记时，请使用
//而不是自动生成。
	if tag := structField.Tag.Get("jsonrpcusage"); tag != "" {
		return tag
	}

//如果需要，请间接使用指针。
	fieldType := structField.Type
	if fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
	}

//当存在默认值时，它还必须是指针，因为
//由registerCmd执行的规则。
	if defaultVal != nil {
		indirect := defaultVal.Elem()
		defaultVal = &indirect
	}

//处理特定类型以提供更好的用法。
	fieldName := strings.ToLower(structField.Name)
	switch fieldType.Kind() {
	case reflect.String:
		if defaultVal != nil {
			return fmt.Sprintf("%s=%q", fieldName,
				defaultVal.Interface())
		}

		return fmt.Sprintf("%q", fieldName)

	case reflect.Array, reflect.Slice:
		return subArrayUsage(fieldType, fieldName)

	case reflect.Struct:
		return subStructUsage(fieldType)
	}

//如果没有上述特殊情况，只需返回字段名
//申请。
	if defaultVal != nil {
		return fmt.Sprintf("%s=%v", fieldName, defaultVal.Interface())
	}
	return fieldName
}

//methodusagettext为提供的命令返回一行用法字符串，并且
//方法信息。This is the main work horse for the exported MethodUsageText
//功能。
func methodUsageText(rtp reflect.Type, defaults map[int]reflect.Value, method string) string {
//为命令中的每个字段生成单独的用法。几个
//简化假设是因为registerCmd
//函数已经严格执行了布局。
	rt := rtp.Elem()
	numFields := rt.NumField()
	reqFieldUsages := make([]string, 0, numFields)
	optFieldUsages := make([]string, 0, numFields)
	for i := 0; i < numFields; i++ {
		rtf := rt.Field(i)
		var isOptional bool
		if kind := rtf.Type.Kind(); kind == reflect.Ptr {
			isOptional = true
		}

		var defaultVal *reflect.Value
		if defVal, ok := defaults[i]; ok {
			defaultVal = &defVal
		}

//将人类可读的用法添加到适当的切片，即
//稍后用于生成单行用法。
		usage := fieldUsage(rtf, defaultVal)
		if isOptional {
			optFieldUsages = append(optFieldUsages, usage)
		} else {
			reqFieldUsages = append(reqFieldUsages, usage)
		}
	}

//生成并返回单行用法字符串。
	usageStr := method
	if len(reqFieldUsages) > 0 {
		usageStr += " " + strings.Join(reqFieldUsages, " ")
	}
	if len(optFieldUsages) > 0 {
		usageStr += fmt.Sprintf(" (%s)", strings.Join(optFieldUsages, " "))
	}
	return usageStr
}

//methodusageText为提供的方法返回一行用法字符串。这个
//提供的方法必须与已注册的类型关联。所有命令
//默认情况下，此包提供的是注册的。
func MethodUsageText(method string) (string, error) {
//查找有关所提供方法的详细信息，如果没有，则返回错误信息
//注册的。
	registerLock.RLock()
	rtp, ok := methodToConcreteType[method]
	info := methodToInfo[method]
	registerLock.RUnlock()
	if !ok {
		str := fmt.Sprintf("%q is not registered", method)
		return "", makeError(ErrUnregisteredMethod, str)
	}

//当已经生成此方法的用法时，只需
//把它还给我。
	if info.usage != "" {
		return info.usage, nil
	}

//为将来的调用生成和存储使用字符串并返回它。
	usage := methodUsageText(rtp, info.defaults, method)
	registerLock.Lock()
	info.usage = usage
	methodToInfo[method] = info
	registerLock.Unlock()
	return usage, nil
}
