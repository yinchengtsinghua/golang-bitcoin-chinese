
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

package blockchain

import (
	"reflect"
	"testing"
)

//
//旧版本0格式的条目按预期工作。
func TestDeserializeUtxoEntryV0(t *testing.T) {
	tests := []struct {
		name       string
		entries    map[uint32]*UtxoEntry
		serialized []byte
	}{
//来自主区块链中的Tx：
//0e3e2357e806b6cdb1f70b54c3a17b6714ee1f0e68beb44a74b1efd512098
		{
			name: "Only output 0, coinbase",
			entries: map[uint32]*UtxoEntry{
				0: {
					amount:      5000000000,
					pkScript:    hexToBytes("410496b538e853519c726a2c91e61ec11600ae1390813a627c66fb8be7947be63c52da7589379515d4e0a604f8141781e62294721166bf621e73a82cbf2342c858eeac"),
					blockHeight: 1,
					packedFlags: tfCoinBase,
				},
			},
			serialized: hexToBytes("010103320496b538e853519c726a2c91e61ec11600ae1390813a627c66fb8be7947be63c52"),
		},
//来自主区块链中的Tx：
//8131ffb0a2c945ecaf9b9063e59558784f9c3a74741ce6ae2a18d0571dac15bb
		{
			name: "Only output 1, not coinbase",
			entries: map[uint32]*UtxoEntry{
				1: {
					amount:      1000000,
					pkScript:    hexToBytes("76a914ee8bd501094a7d5ca318da2506de35e1cb025ddc88ac"),
					blockHeight: 100001,
					packedFlags: 0,
				},
			},
			serialized: hexToBytes("01858c21040700ee8bd501094a7d5ca318da2506de35e1cb025ddc"),
		},
//改编自主区块链中的Tx：
//df3f3f442d9699857f7f4f49de4f0b5d0f3448bec31cdc7b5bf6d25f2abd637d5
		{
			name: "Only output 2, coinbase",
			entries: map[uint32]*UtxoEntry{
				2: {
					amount:      100937281,
					pkScript:    hexToBytes("76a914da33f77cee27c2a975ed5124d7e4f7f97513510188ac"),
					blockHeight: 99004,
					packedFlags: tfCoinBase,
				},
			},
			serialized: hexToBytes("0185843c010182b095bf4100da33f77cee27c2a975ed5124d7e4f7f975135101"),
		},
//改编自主区块链中的Tx：
//4A16969AA4764DD7507fc1de7f0baa4850a246de90c45e59a207f9a26b5036f
		{
			name: "outputs 0 and 2 not coinbase",
			entries: map[uint32]*UtxoEntry{
				0: {
					amount:      20000000,
					pkScript:    hexToBytes("76a914e2ccd6ec7c6e2e581349c77e067385fa8236bf8a88ac"),
					blockHeight: 113931,
					packedFlags: 0,
				},
				2: {
					amount:      15000000,
					pkScript:    hexToBytes("76a914b8025be1b3efc63b0ad48e7f9f10e87544528d5888ac"),
					blockHeight: 113931,
					packedFlags: 0,
				},
			},
			serialized: hexToBytes("0185f90b0a011200e2ccd6ec7c6e2e581349c77e067385fa8236bf8a800900b8025be1b3efc63b0ad48e7f9f10e87544528d58"),
		},
//改编自主区块链中的Tx：
//1B02D1C8CFF60A189017B9A420C682CF4A0028175F2F563209E4F61C8C3620
		{
			name: "Only output 22, not coinbase",
			entries: map[uint32]*UtxoEntry{
				22: {
					amount:      366875659,
					pkScript:    hexToBytes("a9141dd46a006572d820e448e12d2bbb38640bc718e687"),
					blockHeight: 338156,
					packedFlags: 0,
				},
			},
			serialized: hexToBytes("0193d06c100000108ba5b9e763011dd46a006572d820e448e12d2bbb38640bc718e6"),
		},
	}

	for i, test := range tests {
//反序列化到输出索引键控的utxos映射。
		entries, err := deserializeUtxoEntryV0(test.serialized)
		if err != nil {
			t.Errorf("deserializeUtxoEntryV0 #%d (%s) unexpected "+
				"error: %v", i, test.name, err)
			continue
		}

//确保反序列化项与
//在测试条目中。
		if !reflect.DeepEqual(entries, test.entries) {
			t.Errorf("deserializeUtxoEntryV0 #%d (%s) unexpected "+
				"entries: got %v, want %v", i, test.name,
				entries, test.entries)
			continue
		}
	}
}
