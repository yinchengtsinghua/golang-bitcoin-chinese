
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package treap

import (
	"encoding/binary"
	"encoding/hex"
	"math/rand"
	"reflect"
	"testing"
)

//fromhex将传递的十六进制字符串转换为字节片，如果
//有一个错误。这只为硬编码常量提供，因此
//可以检测到源代码中的错误。只有（而且必须）是
//called for initialization purposes.
func fromHex(s string) []byte {
	r, err := hex.DecodeString(s)
	if err != nil {
		panic("invalid hex in source file: " + s)
	}
	return r
}

//serialiuint32返回传递的uint32的big-endian编码。
func serializeUint32(ui uint32) []byte {
	var ret [4]byte
	binary.BigEndian.PutUint32(ret[:], ui)
	return ret[:]
}

//TestParentStack确保TreaparentStack功能按预期工作。
func TestParentStack(t *testing.T) {
	t.Parallel()

	tests := []struct {
		numNodes int
	}{
		{numNodes: 1},
		{numNodes: staticDepth},
{numNodes: staticDepth + 1}, //测试动态代码路径
	}

testLoop:
	for i, test := range tests {
		nodes := make([]*treapNode, 0, test.numNodes)
		for j := 0; j < test.numNodes; j++ {
			var key [4]byte
			binary.BigEndian.PutUint32(key[:], uint32(j))
			node := newTreapNode(key[:], key[:], 0)
			nodes = append(nodes, node)
		}

//测试时将所有节点推送到父堆栈上
//各种堆栈属性。
		stack := &parentStack{}
		for j, node := range nodes {
			stack.Push(node)

//确保堆栈长度为预期值。
			if stack.Len() != j+1 {
				t.Errorf("Len #%d (%d): unexpected stack "+
					"length - got %d, want %d", i, j,
					stack.Len(), j+1)
				continue testLoop
			}

//确保每个索引处的节点都是预期的节点。
			for k := 0; k <= j; k++ {
				atNode := stack.At(j - k)
				if !reflect.DeepEqual(atNode, nodes[k]) {
					t.Errorf("At #%d (%d): mismatched node "+
						"- got %v, want %v", i, j-k,
						atNode, nodes[k])
					continue testLoop
				}
			}
		}

//确保每个弹出的节点都是预期的节点。
		for j := 0; j < len(nodes); j++ {
			node := stack.Pop()
			expected := nodes[len(nodes)-j-1]
			if !reflect.DeepEqual(node, expected) {
				t.Errorf("At #%d (%d): mismatched node - "+
					"got %v, want %v", i, j, node, expected)
				continue testLoop
			}
		}

//确保堆栈现在是空的。
		if stack.Len() != 0 {
			t.Errorf("Len #%d: stack is not empty - got %d", i,
				stack.Len())
			continue testLoop
		}

//确保尝试在超出
//堆栈的长度返回零。
		if node := stack.At(2); node != nil {
			t.Errorf("At #%d: did not give back nil - got %v", i,
				node)
			continue testLoop
		}

//Ensure attempting to pop a node from an empty stack returns
//零。
		if node := stack.Pop(); node != nil {
			t.Errorf("Pop #%d: did not give back nil - got %v", i,
				node)
			continue testLoop
		}
	}
}

func init() {
//对每个测试运行强制使用相同的伪随机数。
	rand.Seed(0)
}
