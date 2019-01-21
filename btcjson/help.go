
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
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"text/tabwriter"
)

//basehelpDesc包含使用的各种帮助标签、类型和示例值
//生成帮助时。每个命令的概要、字段描述，
//条件和结果描述将由调用方提供。
var baseHelpDescs = map[string]string{
//其他帮助标签和输出。
	"help-arguments":      "Arguments",
	"help-arguments-none": "None",
	"help-result":         "Result",
	"help-result-nothing": "Nothing",
	"help-default":        "default",
	"help-optional":       "optional",
	"help-required":       "required",

//JSON类型。
	"json-type-numeric": "numeric",
	"json-type-string":  "string",
	"json-type-bool":    "boolean",
	"json-type-array":   "array of ",
	"json-type-object":  "object",
	"json-type-value":   "value",

//JSON实例。
	"json-example-string":   "value",
	"json-example-bool":     "true|false",
	"json-example-map-data": "data",
	"json-example-unknown":  "unknown",
}

//desclookupfunc是用于查找给定描述的函数
//一把钥匙。
type descLookupFunc func(string) string

//ReflectTypeToJSonType返回表示JSON类型的字符串
//与提供的Go类型关联。
func reflectTypeToJSONType(xT descLookupFunc, rt reflect.Type) string {
	kind := rt.Kind()
	if isNumeric(kind) {
		return xT("json-type-numeric")
	}

	switch kind {
	case reflect.String:
		return xT("json-type-string")

	case reflect.Bool:
		return xT("json-type-bool")

	case reflect.Array, reflect.Slice:
		return xT("json-type-array") + reflectTypeToJSONType(xT,
			rt.Elem())

	case reflect.Struct:
		return xT("json-type-object")

	case reflect.Map:
		return xT("json-type-object")
	}

	return xT("json-type-value")
}

//resultstructhelp返回包含结果帮助输出的字符串切片
//对于一个结构。每行使用制表符来分隔相关的部分，因此
//稍后可以使用制表符来排列所有内容。描述如下：
//从基于小写版本的活动帮助描述映射中提取
//提供的反射类型和JSON名称（或
//字段名（如果未指定JSON标记）。
func resultStructHelp(xT descLookupFunc, rt reflect.Type, indentLevel int) []string {
	indent := strings.Repeat(" ", indentLevel)
	typeName := strings.ToLower(rt.Name())

//为结果结构中的每个字段生成帮助。
	numField := rt.NumField()
	results := make([]string, 0, numField)
	for i := 0; i < numField; i++ {
		rtf := rt.Field(i)

//要显示的字段名是json名称
//可用，否则使用小写字段名。
		var fieldName string
		if tag := rtf.Tag.Get("json"); tag != "" {
			fieldName = strings.Split(tag, ",")[0]
		} else {
			fieldName = strings.ToLower(rtf.Name)
		}

//如果需要的话，可以使用差异指针。
		rtfType := rtf.Type
		if rtfType.Kind() == reflect.Ptr {
			rtfType = rtf.Type.Elem()
		}

//为此结构的结果类型生成JSON示例
//字段。当它是复杂类型时，请检查类型和
//相应地调整开口托架和支撑组合。
		fieldType := reflectTypeToJSONType(xT, rtfType)
		fieldDescKey := typeName + "-" + fieldName
		fieldExamples, isComplex := reflectTypeToJSONExample(xT,
			rtfType, indentLevel, fieldDescKey)
		if isComplex {
			var brace string
			kind := rtfType.Kind()
			if kind == reflect.Array || kind == reflect.Slice {
				brace = "[{"
			} else {
				brace = "{"
			}
			result := fmt.Sprintf("%s\"%s\": %s\t(%s)\t%s", indent,
				fieldName, brace, fieldType, xT(fieldDescKey))
			results = append(results, result)
			results = append(results, fieldExamples...)
		} else {
			result := fmt.Sprintf("%s\"%s\": %s,\t(%s)\t%s", indent,
				fieldName, fieldExamples[0], fieldType,
				xT(fieldDescKey))
			results = append(results, result)
		}
	}

	return results
}

//reflecttypetojsone示例以
//帮助输出。它递归地处理数组、切片和结构。输出
//作为一段行返回，这样最终帮助可以通过
//制表符还返回一个bool，该bool指定
//类型导致复杂的JSON对象，因为需要处理它们
//不同的。
func reflectTypeToJSONExample(xT descLookupFunc, rt reflect.Type, indentLevel int, fieldDescKey string) ([]string, bool) {
//间接指针（如果需要）。
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	kind := rt.Kind()
	if isNumeric(kind) {
		if kind == reflect.Float32 || kind == reflect.Float64 {
			return []string{"n.nnn"}, false
		}

		return []string{"n"}, false
	}

	switch kind {
	case reflect.String:
		return []string{`"` + xT("json-example-string") + `"`}, false

	case reflect.Bool:
		return []string{xT("json-example-bool")}, false

	case reflect.Struct:
		indent := strings.Repeat(" ", indentLevel)
		results := resultStructHelp(xT, rt, indentLevel+1)

//第一个缩进级别需要一个左大括号。为了
//所有其他，它将作为前一部分包括在内
//字段。
		if indentLevel == 0 {
			newResults := make([]string, len(results)+1)
			newResults[0] = "{"
			copy(newResults[1:], results)
			results = newResults
		}

//右大括号后除第一个外还有一个逗号
//缩进级别。最后的制表符是必需的，因此制表符编写器
//把事情排好。
		closingBrace := indent + "}"
		if indentLevel > 0 {
			closingBrace += ","
		}
		results = append(results, closingBrace+"\t\t")
		return results, true

	case reflect.Array, reflect.Slice:
		results, isComplex := reflectTypeToJSONExample(xT, rt.Elem(),
			indentLevel, fieldDescKey)

//当结果很复杂时，这是因为这是一个
//物体。
		if isComplex {
//在缩进级别为零时，没有
//上一个字段用于容纳开始数组括号，因此
//用数组替换开放对象大括号
//语法。另外，更换最后的闭合对象支架
//使用可变数组结束语法。
			indent := strings.Repeat(" ", indentLevel)
			if indentLevel == 0 {
				results[0] = indent + "[{"
				results[len(results)-1] = indent + "},...]"
				return results, true
			}

//此时，缩进级别大于0，因此
//开场数组括号和对象括号是
//已经是前一字段的一部分。然而，
//关闭条目是一个简单的对象大括号，因此请替换它
//使用可变数组结束语法。决赛
//制表符是必需的，因此制表符编写器会将内容排成一行
//适当地。
			results[len(results)-1] = indent + "},...],\t\t"
			return results, true
		}

//它是一个基元数组，因此返回格式化的文本
//因此。
		return []string{fmt.Sprintf("[%s,...]", results[0])}, false

	case reflect.Map:
		indent := strings.Repeat(" ", indentLevel)
		results := make([]string, 0, 3)

//第一个缩进级别需要一个左大括号。为了
//所有其他，它将作为前一部分包括在内
//字段。
		if indentLevel == 0 {
			results = append(results, indent+"{")
		}

//地图有点特别，因为它们需要钥匙，
//值，并具体描述对象条目
//大声叫喊。
		innerIndent := strings.Repeat(" ", indentLevel+1)
		result := fmt.Sprintf("%s%q: %s, (%s) %s", innerIndent,
			xT(fieldDescKey+"--key"), xT(fieldDescKey+"--value"),
			reflectTypeToJSONType(xT, rt), xT(fieldDescKey+"--desc"))
		results = append(results, result)
		results = append(results, innerIndent+"...")

		results = append(results, indent+"}")
		return results, true
	}

	return []string{xT("json-example-unknown")}, false
}

//结果类型帮助生成并返回所提供结果的格式化帮助
//类型。
func resultTypeHelp(xT descLookupFunc, rt reflect.Type, fieldDescKey string) string {
//为结果类型生成JSON示例。
	results, isComplex := reflectTypeToJSONExample(xT, rt, 0, fieldDescKey)

//当这是基元类型时，添加关联的JSON类型和
//将结果描述转换为最终字符串，并相应地格式化，
//然后把它还给我。
	if !isComplex {
		return fmt.Sprintf("%s (%s) %s", results[0],
			reflectTypeToJSONType(xT, rt), xT(fieldDescKey))
	}

//此时，这是一个已经具有JSON类型的复杂类型
//以及结果中的描述。因此，使用制表符编写器
//对齐帮助文本。
	var formatted bytes.Buffer
	w := new(tabwriter.Writer)
	w.Init(&formatted, 0, 4, 1, ' ', 0)
	for i, text := range results {
		if i == len(results)-1 {
			fmt.Fprintf(w, text)
		} else {
			fmt.Fprintln(w, text)
		}
	}
	w.Flush()
	return formatted.String()
}

//argtypeHelp将提供的命令参数的类型作为字符串返回到
//帮助输出使用的格式。特别是，它包括JSON类型
//（布尔值、数字、字符串、数组、对象）以及可选和默认值
//值（如果适用）。
func argTypeHelp(xT descLookupFunc, structField reflect.StructField, defaultVal *reflect.Value) string {
//如果需要，间接使用指针，如果是可选字段，则跟踪它。
	fieldType := structField.Type
	var isOptional bool
	if fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
		isOptional = true
	}

//当存在默认值时，它还必须是指针，因为
//由registerCmd执行的规则。
	if defaultVal != nil {
		indirect := defaultVal.Elem()
		defaultVal = &indirect
	}

//将字段类型转换为JSON类型。
	details := make([]string, 0, 3)
	details = append(details, reflectTypeToJSONType(xT, fieldType))

//如果需要，可在详细信息中添加可选和默认值。
	if isOptional {
		details = append(details, xT("help-optional"))

//添加默认值（如果有）。这只是检查一下
//当字段是可选的，因为非可选字段不能
//有一个默认值。
		if defaultVal != nil {
			val := defaultVal.Interface()
			if defaultVal.Kind() == reflect.String {
				val = fmt.Sprintf(`"%s"`, val)
			}
			str := fmt.Sprintf("%s=%v", xT("help-default"), val)
			details = append(details, str)
		}
	} else {
		details = append(details, xT("help-required"))
	}

	return strings.Join(details, ", ")
}

//arghelp生成并返回所提供命令的格式化帮助。
func argHelp(xT descLookupFunc, rtp reflect.Type, defaults map[int]reflect.Value, method string) string {
//如果命令没有参数，现在返回。
	rt := rtp.Elem()
	numFields := rt.NumField()
	if numFields == 0 {
		return ""
	}

//为命令中的每个参数生成帮助。几个
//简化假设是因为registerCmd
//函数已经严格执行了布局。
	args := make([]string, 0, numFields)
	for i := 0; i < numFields; i++ {
		rtf := rt.Field(i)
		var defaultVal *reflect.Value
		if defVal, ok := defaults[i]; ok {
			defaultVal = &defVal
		}

		fieldName := strings.ToLower(rtf.Name)
		helpText := fmt.Sprintf("%d.\t%s\t(%s)\t%s", i+1, fieldName,
			argTypeHelp(xT, rtf, defaultVal),
			xT(method+"-"+fieldName))
		args = append(args, helpText)

//对于需要JSON对象或JSON数组的类型
//对象，生成参数的完整语法。
		fieldType := rtf.Type
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}
		kind := fieldType.Kind()
		switch kind {
		case reflect.Struct:
			fieldDescKey := fmt.Sprintf("%s-%s", method, fieldName)
			resultText := resultTypeHelp(xT, fieldType, fieldDescKey)
			args = append(args, resultText)

		case reflect.Map:
			fieldDescKey := fmt.Sprintf("%s-%s", method, fieldName)
			resultText := resultTypeHelp(xT, fieldType, fieldDescKey)
			args = append(args, resultText)

		case reflect.Array, reflect.Slice:
			fieldDescKey := fmt.Sprintf("%s-%s", method, fieldName)
			if rtf.Type.Elem().Kind() == reflect.Struct {
				resultText := resultTypeHelp(xT, fieldType,
					fieldDescKey)
				args = append(args, resultText)
			}
		}
	}

//添加参数名称、类型和描述（如果有）。使用A
//制表符编写器可以很好地对齐帮助文本。
	var formatted bytes.Buffer
	w := new(tabwriter.Writer)
	w.Init(&formatted, 0, 4, 1, ' ', 0)
	for _, text := range args {
		fmt.Fprintln(w, text)
	}
	w.Flush()
	return formatted.String()
}

//methodhelp生成并返回所提供命令的帮助输出
//和方法信息。这是导出方法帮助的主要工作单元
//功能。
func methodHelp(xT descLookupFunc, rtp reflect.Type, defaults map[int]reflect.Value, method string, resultTypes []interface{}) string {
//从方法用法和帮助概要开始。
	help := fmt.Sprintf("%s\n\n%s\n", methodUsageText(rtp, defaults, method),
		xT(method+"--synopsis"))

//为命令中的每个参数生成帮助。
	if argText := argHelp(xT, rtp, defaults, method); argText != "" {
		help += fmt.Sprintf("\n%s:\n%s", xT("help-arguments"),
			argText)
	} else {
		help += fmt.Sprintf("\n%s:\n%s\n", xT("help-arguments"),
			xT("help-arguments-none"))
	}

//为每个结果类型生成帮助文本。
	resultTexts := make([]string, 0, len(resultTypes))
	for i := range resultTypes {
		rtp := reflect.TypeOf(resultTypes[i])
		fieldDescKey := fmt.Sprintf("%s--result%d", method, i)
		if resultTypes[i] == nil {
			resultText := xT("help-result-nothing")
			resultTexts = append(resultTexts, resultText)
			continue
		}

		resultText := resultTypeHelp(xT, rtp.Elem(), fieldDescKey)
		resultTexts = append(resultTexts, resultText)
	}

//添加结果类型和说明。当有多个
//结果类型，还添加触发它的条件。
	if len(resultTexts) > 1 {
		for i, resultText := range resultTexts {
			condKey := fmt.Sprintf("%s--condition%d", method, i)
			help += fmt.Sprintf("\n%s (%s):\n%s\n",
				xT("help-result"), xT(condKey), resultText)
		}
	} else if len(resultTexts) > 0 {
		help += fmt.Sprintf("\n%s:\n%s\n", xT("help-result"),
			resultTexts[0])
	} else {
		help += fmt.Sprintf("\n%s:\n%s\n", xT("help-result"),
			xT("help-result-nothing"))
	}
	return help
}

//ISvalidResultType返回传递的反射类型是否为
//结果的可接受类型。
func isValidResultType(kind reflect.Kind) bool {
	if isNumeric(kind) {
		return true
	}

	switch kind {
	case reflect.String, reflect.Struct, reflect.Array, reflect.Slice,
		reflect.Bool, reflect.Map:

		return true
	}

	return false
}

//generateHelp生成并返回所提供方法的帮助输出，以及
//为该方法提供适当键的映射的结果类型
//概要、字段描述、条件和结果描述。这个
//方法必须与已注册的类型关联。由提供的所有命令
//默认情况下，此包已注册。
//
//结果类型必须是指向表示特定类型的类型的指针
//命令返回的值。例如，如果命令只返回
//布尔值，应该只有一个（*bool）（nil）条目。注释
//每个类型必须是指向该类型的单个指针。因此，它是
//建议简单地将nil指针转换传递给适当的类型
//之前显示。
//
//提供的描述映射必须包含所有键，否则将出现错误
//返回，其中包括丢失的密钥，或在
//缺少多个密钥。在这种情况下产生的帮助
//错误将使用键代替描述。
//
//以下概述了所需的键：
//命令的“概要”
//“每个命令参数的说明”
//“<typename>-<lowerfieldname>”每个对象字段的说明
//“<method>——条件<>”每个结果条件的说明
//“<method>——result<>”每个基本结果编号的说明
//
//请注意，“特殊”键概要、条件<>和结果<>是
//以双破折号开头，以确保它们不会与字段名冲突。
//
//仅当结果类型上有多个时才需要条件键，
//并且只有当给定结果类型不是
//对象。
//
//例如，考虑“help”命令本身。有两种可能
//根据提供的参数返回。所以，帮助是
//通过如下方式调用函数生成：
//generatehelp（“帮助”，descs，（（*string）（nil），（*string）（nil））。
//
//然后在提供的描述图中需要以下键：
//
//“help--synopsis”：“返回所有命令的列表或..的帮助。”
//“help command”：“检索帮助的命令”，
//“help--condition0”：“未提供命令”
//“help--condition1”：“指定的命令”
//“help--result0”：“命令列表”
//“help--result1”：“指定命令的帮助”
func GenerateHelp(method string, descs map[string]string, resultTypes ...interface{}) (string, error) {
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

//验证每个结果类型都是指向受支持类型（或nil）的指针。
	for i, resultType := range resultTypes {
		if resultType == nil {
			continue
		}

		rtp := reflect.TypeOf(resultType)
		if rtp.Kind() != reflect.Ptr {
			str := fmt.Sprintf("result #%d (%v) is not a pointer",
				i, rtp.Kind())
			return "", makeError(ErrInvalidType, str)
		}

		elemKind := rtp.Elem().Kind()
		if !isValidResultType(elemKind) {
			str := fmt.Sprintf("result #%d (%v) is not an allowed "+
				"type", i, elemKind)
			return "", makeError(ErrInvalidType, str)
		}
	}

//为返回的描述查找函数创建一个闭包
//到基本帮助描述映射无法识别的键和轨迹
//还有丢失的钥匙。
	var missingKey string
	xT := func(key string) string {
		if desc, ok := descs[key]; ok {
			return desc
		}
		if desc, ok := baseHelpDescs[key]; ok {
			return desc
		}

		missingKey = key
		return key
	}

//生成并返回方法的帮助。
	help := methodHelp(xT, rtp, info.defaults, method, resultTypes)
	if missingKey != "" {
		return help, makeError(ErrMissingDescription, missingKey)
	}
	return help, nil
}
