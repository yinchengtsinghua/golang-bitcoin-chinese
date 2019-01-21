
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

package mining

import (
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

//newhashfromstr将传递的big endian十六进制字符串转换为
//chainhash.hash。它只与chainhash中可用的不同之处在于
//它会因错误而惊慌失措，因为它只能（而且必须）用
//硬编码，因此已知良好，哈希。
func newHashFromStr(hexStr string) *chainhash.Hash {
	hash, err := chainhash.NewHashFromStr(hexStr)
	if err != nil {
		panic("invalid hash in source file: " + hexStr)
	}
	return hash
}

//hextobytes将传递的十六进制字符串转换为字节，如果有，将死机
//是一个错误。这仅为硬编码常量提供，因此
//可以检测到源代码。它只能（而且必须）用
//硬编码值。
func hexToBytes(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic("invalid hex in source file: " + s)
	}
	return b
}

//new utxo view返回一个新的utxo视图，其中填充了
//提供源交易，如同
//高度切片中指定的块高度。源Txns的长度
//和源Tx高度必须匹配，否则它会恐慌。
func newUtxoViewpoint(sourceTxns []*wire.MsgTx, sourceTxHeights []int32) *blockchain.UtxoViewpoint {
	if len(sourceTxns) != len(sourceTxHeights) {
		panic("each transaction must have its block height specified")
	}

	view := blockchain.NewUtxoViewpoint()
	for i, tx := range sourceTxns {
		view.AddTxOuts(btcutil.NewTx(tx), sourceTxHeights[i])
	}
	return view
}

//TestCalcPriority确保优先级计算按预期工作。
func TestCalcPriority(t *testing.T) {
//commonourcetx1是在下面的测试中用作
//对计算优先级的事务的输入。
//
//来自主区块链中的区块7。
//德克萨斯州0437CD7F8525CEED2324359C2D0BA26006D92D856A9C20FA0241106EE5A597C9
	commonSourceTx1 := &wire.MsgTx{
		Version: 1,
		TxIn: []*wire.TxIn{{
			PreviousOutPoint: wire.OutPoint{
				Hash:  chainhash.Hash{},
				Index: wire.MaxPrevOutIndex,
			},
			SignatureScript: hexToBytes("04ffff001d0134"),
			Sequence:        0xffffffff,
		}},
		TxOut: []*wire.TxOut{{
			Value: 5000000000,
			PkScript: hexToBytes("410411db93e1dcdb8a016b49840f8c5" +
				"3bc1eb68a382e97b1482ecad7b148a6909a5cb2e0ead" +
				"dfb84ccf9744464f82e160bfa9b8b64f9d4c03f999b8" +
				"643f656b412a3ac"),
		}},
		LockTime: 0,
	}

//CommonRedeemTx1是以下测试中使用的有效事务
//要计算优先级的事务。
//
//它最初来自主区块链中的区块170。
	commonRedeemTx1 := &wire.MsgTx{
		Version: 1,
		TxIn: []*wire.TxIn{{
			PreviousOutPoint: wire.OutPoint{
				Hash: *newHashFromStr("0437cd7f8525ceed232435" +
					"9c2d0ba26006d92d856a9c20fa0241106ee5" +
					"a597c9"),
				Index: 0,
			},
			SignatureScript: hexToBytes("47304402204e45e16932b8af" +
				"514961a1d3a1a25fdf3f4f7732e9d624c6c61548ab5f" +
				"b8cd410220181522ec8eca07de4860a4acdd12909d83" +
				"1cc56cbbac4622082221a8768d1d0901"),
			Sequence: 0xffffffff,
		}},
		TxOut: []*wire.TxOut{{
			Value: 1000000000,
			PkScript: hexToBytes("4104ae1a62fe09c5f51b13905f07f06" +
				"b99a2f7159b2225f374cd378d71302fa28414e7aab37" +
				"397f554a7df5f142c21c1b7303b8a0626f1baded5c72" +
				"a704f7e6cd84cac"),
		}, {
			Value: 4000000000,
			PkScript: hexToBytes("410411db93e1dcdb8a016b49840f8c5" +
				"3bc1eb68a382e97b1482ecad7b148a6909a5cb2e0ead" +
				"dfb84ccf9744464f82e160bfa9b8b64f9d4c03f999b8" +
				"643f656b412a3ac"),
		}},
		LockTime: 0,
	}

	tests := []struct {
name       string                    //测试说明
tx         *wire.MsgTx               //Tx到计算优先级
utxoView   *blockchain.UtxoViewpoint //对TX的输入
nextHeight int32                     //优先级计算的高度
want       float64                   //预期优先级
	}{
		{
			name: "one height 7 input, prio tx height 169",
			tx:   commonRedeemTx1,
			utxoView: newUtxoViewpoint([]*wire.MsgTx{commonSourceTx1},
				[]int32{7}),
			nextHeight: 169,
			want:       5e9,
		},
		{
			name: "one height 100 input, prio tx height 169",
			tx:   commonRedeemTx1,
			utxoView: newUtxoViewpoint([]*wire.MsgTx{commonSourceTx1},
				[]int32{100}),
			nextHeight: 169,
			want:       2129629629.6296296,
		},
		{
			name: "one height 7 input, prio tx height 100000",
			tx:   commonRedeemTx1,
			utxoView: newUtxoViewpoint([]*wire.MsgTx{commonSourceTx1},
				[]int32{7}),
			nextHeight: 100000,
			want:       3086203703703.7036,
		},
		{
			name: "one height 100 input, prio tx height 100000",
			tx:   commonRedeemTx1,
			utxoView: newUtxoViewpoint([]*wire.MsgTx{commonSourceTx1},
				[]int32{100}),
			nextHeight: 100000,
			want:       3083333333333.3335,
		},
	}

	for i, test := range tests {
		got := CalcPriority(test.tx, test.utxoView, test.nextHeight)
		if got != test.want {
			t.Errorf("CalcPriority #%d (%q): unexpected priority "+
				"got %v want %v", i, test.name, got, test.want)
			continue
		}
	}
}
