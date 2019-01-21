
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package rpctest

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

var (
//compilemtx保护对可执行路径的访问，以便项目
//只编译一次。
	compileMtx sync.Mutex

//ExecutablePath是已编译可执行文件的路径。这是空的
//字符串，直到编译BTCD为止。不应直接访问；
//而是使用函数btcdExecutablePath（）。
	executablePath string
)

//btcdExecutablePath返回要由使用的btcd可执行文件的路径
//RPC.确保对最新版本的
//btcd，此方法在第一次调用时编译btcd。之后，
//生成的二进制文件用于后续的测试线束。可执行文件
//不会清除，但由于它位于临时目录中的静态路径，
//没什么大不了的。
func btcdExecutablePath() (string, error) {
	compileMtx.Lock()
	defer compileMtx.Unlock()

//如果已经编译了BTCD，就使用它。
	if len(executablePath) != 0 {
		return executablePath, nil
	}

	testDir, err := baseDir()
	if err != nil {
		return "", err
	}

//生成btcd并在静态临时路径中输出可执行文件。
	outputPath := filepath.Join(testDir, "btcd")
	if runtime.GOOS == "windows" {
		outputPath += ".exe"
	}
	cmd := exec.Command(
		"go", "build", "-o", outputPath, "github.com/btcsuite/btcd",
	)
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("Failed to build btcd: %v", err)
	}

//保存可执行路径，以便以后的调用不会重新编译。
	executablePath = outputPath
	return executablePath, nil
}
