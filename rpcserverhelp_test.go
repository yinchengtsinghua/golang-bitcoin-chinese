
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

package main

import "testing"

//测试帮助通过检查
//指定的命令还定义了结果类型和单行用法，以及
//可以为它们生成帮助文本。
func TestHelp(t *testing.T) {
//确保为每个处理程序指定了结果类型。
	for k := range rpcHandlers {
		if _, ok := rpcResultTypes[k]; !ok {
			t.Errorf("RPC handler defined for method '%v' without "+
				"also specifying result types", k)
			continue
		}

	}
	for k := range wsHandlers {
		if _, ok := rpcResultTypes[k]; !ok {
			t.Errorf("RPC handler defined for method '%v' without "+
				"also specifying result types", k)
			continue
		}

	}

//确保可以生成每个命令的用法而不会出错。
	helpCacher := newHelpCacher()
	if _, err := helpCacher.rpcUsage(true); err != nil {
		t.Fatalf("Failed to generate one-line usage: %v", err)
	}
	if _, err := helpCacher.rpcUsage(true); err != nil {
		t.Fatalf("Failed to generate one-line usage (cached): %v", err)
	}

//确保可以生成每个命令的帮助而不会出错。
	for k := range rpcHandlers {
		if _, err := helpCacher.rpcMethodHelp(k); err != nil {
			t.Errorf("Failed to generate help for method '%v': %v",
				k, err)
			continue
		}
		if _, err := helpCacher.rpcMethodHelp(k); err != nil {
			t.Errorf("Failed to generate help for method '%v'"+
				"(cached): %v", k, err)
			continue
		}
	}
	for k := range wsHandlers {
		if _, err := helpCacher.rpcMethodHelp(k); err != nil {
			t.Errorf("Failed to generate help for method '%v': %v",
				k, err)
			continue
		}
		if _, err := helpCacher.rpcMethodHelp(k); err != nil {
			t.Errorf("Failed to generate help for method '%v'"+
				"(cached): %v", k, err)
			continue
		}
	}
}
