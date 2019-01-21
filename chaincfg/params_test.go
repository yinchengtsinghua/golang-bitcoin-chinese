
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package chaincfg

import "testing"

//testinvalidhashstr确保newshahashfromstr函数在用于
//使用无效的哈希字符串。
func TestInvalidHashStr(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic for invalid hash, got nil")
		}
	}()
	newHashFromStr("banana")
}

//testmustregisterpanic确保mustregister函数在用于
//注册无效网络。
func TestMustRegisterPanic(t *testing.T) {
	t.Parallel()

//设置延迟以捕捉预期的恐慌，以确保
//泛冰的
	defer func() {
		if err := recover(); err == nil {
			t.Error("mustRegister did not panic as expected")
		}
	}()

//故意尝试注册重复的参数以强制恐慌。
	mustRegister(&MainNetParams)
}
