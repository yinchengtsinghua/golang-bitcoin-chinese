
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2014-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package chaincfg

import (
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

//genesisCoinBaseTx是用于
//主网络、回归测试网络和测试网络（版本3）。
var genesisCoinbaseTx = wire.MsgTx{
	Version: 1,
	TxIn: []*wire.TxIn{
		{
			PreviousOutPoint: wire.OutPoint{
				Hash:  chainhash.Hash{},
				Index: 0xffffffff,
			},
			SignatureScript: []byte{
    /*4，0xFF，0xFF，0x00，0x1D，0x01，0x04，0x45，/*……E*/
    0x54，0x68，0x65，0x20，0x54，0x69，0x6d，0x65，/*时间*/

    /*3，0x20，0x30，0x33，0x2F，0x4A，0x61，0x6E，/*S 03/Jan*/
    0x2F、0x32、0x30、0x30、0x39、0x20、0x43、0x68、/*/2009频道*/

    /*1，0X6E，0X63，0X65，0X6C，0X6C，0X6F，0X72，/*ANCELLOR*/
    0×20、0×6f、0×6e、0×20、0×62、0×72、0×69、0×6e，/*布林*/

    /*b，0x20，0x6f，0x66，0x20，0x73，0x65，0x63，/*秒k/
    0X6F，0X6E，0X64，0X20，0X62，0X61，0X69，0X6C，/*开环*/

    /*F，0x75，0x74，0x20，0x66，0x6F，0x72，0x20，/*输出用于*/
    0X62、0X61、0X6E、0X6B、0X73、/*银行*/

			},
			Sequence: 0xffffffff,
		},
	},
	TxOut: []*wire.TxOut{
		{
			Value: 0x12a05f200,
			PkScript: []byte{
    /*1，0x04，0x67，0x8A，0xfd，0xb0，0xfe，0x55，/*A.G…U*/
    0x48，0x27，0x19，0x67，0xf1，0xa6，0x71，0x30，/*H'.G..Q0*/

    /*7，0×10，0×5C，0×D6，0×A8，0×28，0×E0，0×39，/*…..（.9*/
    0x09，0xA6，0x79，0x62，0xE0，0xEA，0x1F，0x61，/*.YB…A_*/

    /*E，0XB6，0X49，0XF6，0XBC，0X3F，0X4C，0XEF，/*…I.？L**
    0x38，0xC4，0xF3，0x55，0x04，0xE5，0x1E，0xC1，/*8..U…*/

    /*2，0XDE，0X5C，0X38，0X4D，0XF7，0XBA，0X0B，/*..\8M…*/
    0x8d，0x57，0x8a，0x4c，0x70，0x2b，0x6b，0xf1，/*.w.lp+k.*/

    /*d，0x5f，0xac，/*.*/
   }
  }
 }
 锁定时间：0，
}

//genesishash是主块链中第一个块的哈希
//网络（Genesis块）。
var genesishash=chainhash.hash（[chainhash.hashsize]byte//使go vet高兴。
 0x6F，0xE2，0x8C，0x0A，0xB6，0xF1，0xB3，0x72，
 0xc1、0xa6、0xa2、0x46、0xae、0x63、0xf7、0x4f、
 0X93、0X1E、0X83、0X65、0XE1、0X5A、0X8、0X9C、
 0x68，0xD6，0x19，0x00，0x00，0x00，0x00，0x00，0x00，
}）

//genesismerkleroot是genesis块中第一个事务的哈希
//对于主网络。
var genesismerkleroot=chainhash.hash（[chainhash.hashsize]byte//make go vet happy.
 0x3b，0xa3，0xed，0xfd，0x7a，0x7b，0x12，0xb2，
 0x7a、0xc7、0x2c、0x3e、0x67、0x76、0x8f、0x61、
 0x7F，0xC8，0x1B，0xC3，0x88，0x8A，0x51，0x32，
 0x3a、0x9f、0xb8、0xaa、0x4b、0x1e、0x5e、0x4a、
}）

//genesis block定义作为
//主网络的公共事务分类帐。
var genesisblock=线.msgblock
 标题：Wire.BlockHeader
  版本：1，
  prevblock:chainhash.hash，//0000000000000000000000000000000000000000000000000000000000000000000000000000
  merkleroot:genesismerkleroot，//4A5E1E4BAAB89F3A32518A88C31BC87F618F6673E2CC77AB2127B7AFDEDA33B
  时间戳：time.unix（0x495fab29，0），//2009-01-03 18:15:05+0000 UTC
  位：0x1d00ffff，//486604799[00000000 ff0000000000000000000000000000000000000000000000000000000000000]
  nonce:0x7c2bac1d，//2083236893
 }
 交易：[]*Wire.msgtx&GenerisCoinBaseTx，
}

//regtestgenesHash是块链中第一个块的哈希，
//回归测试网络（Genesis块）。
var regtestgeneshash=chainhash.hash（[chainhash.hashsize]byte//使go vet高兴。
 0x06，0x22，0x6e，0x46，0x11，0x1A，0x0B，0x59，
 0XCA、0XAF、0X12、0X60、0X43、0XEB、0X5B、0XBF、
 0x28，0xc3，0x4f，0x3a，0x5e，0x33，0x2a，0x1f，
 0xC7、0xB2、0xB7、0x3C、0xF1、0x88、0x91、0x0F、
}）

//regtestgenesismerkleroot是Genesis中第一个事务的哈希
//回归测试网络的块。它和merkle的根是一样的
//主网络。
var regtestgenesismerkleroot=genesismerkleroot

//regtestgeneissblock定义块链的Genesis块，该块链用于
//作为回归测试网络的公共事务分类帐。
var regtestgenesblock=线.msgblock
 标题：Wire.BlockHeader
  版本：1，
  prevblock:chainhash.hash，//0000000000000000000000000000000000000000000000000000000000000000000000000000
  merkleroot:regtestgenesismerkleroot，//4A5E1E4BAAB89F3A32518A88C31BC87F618F7673E2CC77AB2127B7AFDEDA33B
  时间戳：time.unix（129668602，0），//2011-02-02 23:16:42+0000 UTC
  位：0x207ffffff，//545259519[7fffff0000000000000000000000000000000000000000000000000000000000000000000000000]
  随机数：2，
 }
 交易：[]*Wire.msgtx&GenerisCoinBaseTx，
}

//testnet3genesishash是块链中第一个块的哈希，
//测试网络（版本3）。
var testnet3genesishash=chainhash.hash（[chainhash.hashsize]byte//使go vet高兴。
 0x43、0x49、0x7f、0xd7、0xf8、0x26、0x95、0x71、
 0x08，0xf4，0xa3，0x0f，0xd9，0xce，0xc3，0xae，
 0XBA，0X79，0X97，0X20，0X84，0XE9，0X0E，0XAD，0位
 0x01，0xEA，0x33，0x09，0x00，0x00，0x00，0x00，0x00，
}）

//testnet3genesismerkleroot是Genesis中第一个事务的哈希
//测试网络块（版本3）。它和梅克尔根一样
//对于主网络。
var testnet3genesismerkleroot=genesismerkleroot

//testnet3genesisblock定义块链的genesis块，该块链
//作为测试网络（版本3）的公共事务分类账。
var testnet3geneissblock=wire.msgblock_
 标题：Wire.BlockHeader
  版本：1，
  prevblock:chainhash.hash，//0000000000000000000000000000000000000000000000000000000000000000000000000000
  merkleroot:testnet3genesismerkleroot，//4a5e1e4baab89f3a2518a88c31bc87f618f6673e2cc77ab2127b7afdeda33b
  时间戳：time.unix（129668602，0），//2011-02-02 23:16:42+0000 UTC
  位：0x1d00ffff，//486604799[00000000 ff0000000000000000000000000000000000000000000000000000000000000]
  nonce:0x18AEA41A，//414098458
 }
 交易：[]*Wire.msgtx&GenerisCoinBaseTx，
}

//simnetgenesHash是块链中第一个块的哈希，
//模拟测试网络。
var simnetgeneshash=chainhash.hash（[chainhash.hashsize]byte//使go vet高兴。
 0XF6、0X7A、0XD7、0X69、0X5D、0X9B、0X66、0X2A、
 0x72、0xff、0x3d、0x8e、0xdb、0xbb、0x2d、0xe0、
 0XBF、0XA6、0X7B、0X13、0X97、0X4B、0XB9、0X91、
 0x0D，0x11，0x6D，0x5C，0xBD，0x86，0x3E，0x68，
}）

//simnetgenesimerkleroot是Genesis中第一个事务的哈希
//模拟测试网络的块。它和merkle的根是一样的
//主网络。
var simnetgenesismerkleroot=genesismerkleroot

//simnetgenesblock定义服务于
//作为模拟测试网络的公共事务分类账。
var simnetgenesblock=线.msgblock
 标题：Wire.BlockHeader
  版本：1，
  prevblock:chainhash.hash，//0000000000000000000000000000000000000000000000000000000000000000000000000000
  merkleroot:simnetgenesismerkleroot，//4A5E1E4BAAB89F3A32518A88C31BC87F618F7673E2CC77AB2127B7AFDEDA33B
  时间戳：time.unix（1401292357，0），//2014-05-28 15:52:37+0000 UTC
  位：0x207ffffff，//545259519[7fffff0000000000000000000000000000000000000000000000000000000000000000000000000]
  随机数：2，
 }
 交易：[]*Wire.msgtx&GenerisCoinBaseTx，
}
