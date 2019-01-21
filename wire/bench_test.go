
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wire

import (
	"bytes"
	"compress/bzip2"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

//genesisCoinBaseTx是用于
//主网络、回归测试网络和测试网络（版本3）。
var genesisCoinbaseTx = MsgTx{
	Version: 1,
	TxIn: []*TxIn{
		{
			PreviousOutPoint: OutPoint{
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
	TxOut: []*TxOut{
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

//BenchmarkWriteVarint1对写入所需的时间执行基准测试
//单字节可变长度整数。
func基准标记写入变量1（b*testing.b）
 对于i：=0；i<b.n；i++
  writevarint（ioutil.discard，0，1）
 }
}

//BenchmarkWriteVarint3对写入所需的时间执行基准测试
//三字节可变长度整数。
func基准写入变量3（b*testing.b）
 对于i：=0；i<b.n；i++
  写入变量（ioutil.discard，0，65535）
 }
}

//BenchmarkWriteVarint5对写入所需的时间执行基准测试
//五字节可变长度整数。
func基准写入变量5（b*testing.b）
 对于i：=0；i<b.n；i++
  写入变量（ioutil.discard，0，4294967295）
 }
}

//BenchmarkWriteVarint9对写入所需的时间执行基准测试
//9字节可变长度整数。
func基准标记写入变量9（b*testing.b）
 对于i：=0；i<b.n；i++
  写入变量（ioutil.discard，0，18446744073709551615）
 }
}

//BenchmarkReadVarint1对读取所需的时间执行基准测试
//单字节可变长度整数。
func基准读数变量1（b*testing.b）
 buf：=[]字节0x01
 r：=bytes.newreader（buf）
 对于i：=0；i<b.n；i++
  R.BASE（0, 0）
  读变量（r，0）
 }
}

//BenchmarkReadVarint3对读取所需的时间执行基准测试
//三字节可变长度整数。
func基准读数变量3（b*testing.b）
 buf：=[]字节0x0fd，0xff，0xff
 r：=bytes.newreader（buf）
 对于i：=0；i<b.n；i++
  R.BASE（0, 0）
  读变量（r，0）
 }
}

//BenchmarkReadVarint5对读取所需的时间执行基准测试
//五字节可变长度整数。
func基准读数变量5（b*testing.b）
 buf：=[]字节0xfe，0xff，0xff，0xff，0xff
 r：=bytes.newreader（buf）
 对于i：=0；i<b.n；i++
  R.BASE（0, 0）
  读变量（r，0）
 }
}

//BenchmarkReadVarint9对读取所需的时间执行基准测试
//9字节可变长度整数。
func基准读数变量9（b*testing.b）
 buf：=[]字节0XFF、0XFF、0XFF、0XFF、0XFF、0XFF、0XFF、0XFF
 r：=bytes.newreader（buf）
 对于i：=0；i<b.n；i++
  R.BASE（0, 0）
  读变量（r，0）
 }
}

//Benchmarkreadvarstr4对读取
//四字节可变长度字符串。
func基准读数varstr4（b*testing.b）
 buf：=[]字节0x04，'t'、'e'、's'、't'
 r：=bytes.newreader（buf）
 对于i：=0；i<b.n；i++
  R.BASE（0, 0）
  readvarstring（r，0）
 }
}

//Benchmarkreadvarstr10对读取
//十字节可变长度字符串。
func基准读数varstr10（b*testing.b）
 buf：=[]字节0x0a，'t'、'e'、's'、't'、'0'、'1'、'2'、'3'、'4'、'5'
 r：=bytes.newreader（buf）
 对于i：=0；i<b.n；i++
  R.BASE（0, 0）
  readvarstring（r，0）
 }
}

//BenchmarkWritevarstr4对编写
//四字节可变长度字符串。
func基准markwritevarstr4（b*testing.b）
 对于i：=0；i<b.n；i++
  writevarstring（ioutil.discard，0，“测试”）
 }
}

//BenchmarkWritevarstr10对编写
//十字节可变长度字符串。
func基准markwritevarstr10（b*testing.b）
 对于i：=0；i<b.n；i++
  writevarstring（ioutil.discard，0，“test012345”）。
 }
}

//BenchmarkReadOutPoint对读取
//事务输出点。
func基准点读数（b*testing.b）
 BUF:= []字节{
  0x00，0x00，0x00，0x00，0x00，0x00，0x00，0x00，0x00，
  0x00，0x00，0x00，0x00，0x00，0x00，0x00，0x00，0x00，
  0x00，0x00，0x00，0x00，0x00，0x00，0x00，0x00，0x00，
  0x00、0x00、0x00、0x00、0x00、0x00、0x00、0x00、0x00，//上一个输出哈希
  0xff，0xff，0xff，0xff，//上一个输出索引
 }
 r：=bytes.newreader（buf）
 OP Out点
 对于i：=0；i<b.n；i++
  R.BASE（0, 0）
  读出点（R、0、0和OP）
 }
}

//BenchmarkWriteOutpoint对编写
//事务输出点。
func基准标记写入输出点（b*testing.b）
 操作：=&outpoint
  hash:chainhash.hash，
  索引：0，
 }
 对于i：=0；i<b.n；i++
  写入输出点（ioutil.discard，0，0，op）
 }
}

//BenchmarkReadTxOut对读取
//事务输出。
func基准读数txout（b*testing.b）
 BUF:= []字节{
  0x00，0xF2，0x05，0x2A，0x01，0x00，0x00，0x00，//交易金额
  0x43，//pk脚本长度的变量
  0x41，//运算数据
  0x04、0x96、0xB5、0x38、0xE8、0x53、0x51、0x9C、
  0x72、0x6a、0x2c、0x91、0xe6、0x1e、0xc1、0x16、
  0x00，0xAE，0x13，0x90，0x81，0x3A，0x62，0x7C，
  0x66、0xFB、0x8B、0xE7、0x94、0x7B、0xE6、0x3C、
  0x52，0xda，0x75，0x89，0x37，0x95，0x15，0xd4，
  0xe0，0xa6，0x04，0xf8，0x14，0x17，0x81，0xe6，
  0x22、0x94、0x72、0x11、0x66、0xBF、0x62、0x1E、
  0x73，0xA8，0x2C，0xBF，0x23，0x42，0xC8，0x58，
  0xee，//65字节签名
  0xac，//操作检查信号
 }
 r：=bytes.newreader（buf）
 VX-TXOUT TXOUT
 对于i：=0；i<b.n；i++
  R.BASE（0, 0）
  读取txout（r、0、0和txout）
  scriptpool.return（txout.pkscript）
 }
}

//BenchmarkWriteTxOut对写入所需的时间执行基准测试
//事务输出。
func基准写入txout（b*testing.b）
 txout：=blockone.transactions[0].txout[0]
 对于i：=0；i<b.n；i++
  writetxout（ioutil.discard，0，0，txout）
 }
}

//BenchmarkReadTxIn对读取
//事务输入。
func基准读数txin（b*testing.b）
 BUF:= []字节{
  0x00，0x00，0x00，0x00，0x00，0x00，0x00，0x00，0x00，
  0x00，0x00，0x00，0x00，0x00，0x00，0x00，0x00，0x00，
  0x00，0x00，0x00，0x00，0x00，0x00，0x00，0x00，0x00，
  0x00、0x00、0x00、0x00、0x00、0x00、0x00、0x00、0x00，//上一个输出哈希
  0xff，0xff，0xff，0xff，//上一个输出索引
  0x07，//签名脚本长度的变量
  0x04，0xFF，0xFF，0x00，0x1D，0x01，0x04，//签名脚本
  0xff，0xff，0xff，0xff，//序列
 }
 r：=bytes.newreader（buf）
 新信
 对于i：=0；i<b.n；i++
  R.BASE（0, 0）
  读取txin（r、0、0和txin）
  scriptpool.return（txin.signaturescript）
 }
}

//BenchmarkWriteTxin对写入所需的时间执行基准测试
//事务输入。
func基准写入txin（b*testing.b）
 txin:=blockone.transactions[0].txin[0]
 对于i：=0；i<b.n；i++
  writetxin（ioutil.discard，0，0，txin）
 }
}

//BenchmarkDeserializeTx根据需要多长时间执行基准测试
//反序列化一个小事务。
func bencmarkdeserializetxsmall（b*testing.b）
 BUF:= []字节{
  0x01，0x00，0x00，0x00，//版本
  0x01，//输入事务数的变量
  0x00，0x00，0x00，0x00，0x00，0x00，0x00，0x00，0x00，
  0x00，0x00，0x00，0x00，0x00，0x00，0x00，0x00，0x00，
  0x00，0x00，0x00，0x00，0x00，0x00，0x00，0x00，0x00，
  0x00、0x00、0x00、0x00、0x00、0x00、0x00、0x00、0x00、///上一个输出哈希
  0xff，0xff，0xff，0xff，//上一输出索引
  0x07，//签名脚本长度的变量
  0x04，0xFF，0xFF，0x00，0x1D，0x01，0x04，//签名脚本
  0xff，0xff，0xff，0xff，//序列
  0x01，//输出事务数的变量
  0x00，0xF2，0x05，0x2A，0x01，0x00，0x00，0x00，//交易金额
  0x43，//pk脚本长度的变量
  0x41，//运算数据
  0x04、0x96、0xB5、0x38、0xE8、0x53、0x51、0x9C、
  0x72、0x6a、0x2c、0x91、0xe6、0x1e、0xc1、0x16、
  0x00，0xAE，0x13，0x90，0x81，0x3A，0x62，0x7C，
  0x66、0xFB、0x8B、0xE7、0x94、0x7B、0xE6、0x3C、
  0x52，0xda，0x75，0x89，0x37，0x95，0x15，0xd4，
  0xe0，0xa6，0x04，0xf8，0x14，0x17，0x81，0xe6，
  0x22、0x94、0x72、0x11、0x66、0xBF、0x62、0x1E、
  0x73，0xA8，0x2C，0xBF，0x23，0x42，0xC8，0x58，
  0xee，//65字节签名
  0xac，//操作检查信号
  0x00，0x00，0x00，0x00，//锁定时间
 }

 r：=bytes.newreader（buf）
 VX TX-MSGTX
 对于i：=0；i<b.n；i++
  R.BASE（0, 0）
  Tx.反序列化（r）
 }
}

//BenchmarkDeserializetXLarge根据需要多长时间执行基准测试
//反序列化一个非常大的事务。
func bencmarkdeserializetxlarge（b*testing.b）
 //tx bb41a757f405890fb0f5856228e23b715702d714d59bf2b1feb70d8b2b4e3e08
 //来自主区块链。
 fi，err：=os.open（“测试数据/megatx.bin.bz2”）
 如果犯错！= nIL{
  b.fatalf（“读取事务数据失败：%v”，错误）
 }
 推迟FI.CLOSE（）。
 buf，err：=ioutil.readall（bzip2.newreader（fi））。
 如果犯错！= nIL{
  b.fatalf（“读取事务数据失败：%v”，错误）
 }

 r：=bytes.newreader（buf）
 VX TX-MSGTX
 对于i：=0；i<b.n；i++
  R.BASE（0, 0）
  Tx.反序列化（r）
 }
}

//BenchmarkSerializeTx对序列化所需的时间执行基准测试
//事务。
func基准MarkSerializeTX（b*testing.b）
 tx：=blockone.transactions[0]
 对于i：=0；i<b.n；i++
  tx.serialize（ioutil.discard）

 }
}

//BenchmarkReadBlockHeader对需要多长时间执行基准测试
//反序列化块头。
func基准标记readblockheader（b*testing.b）
 BUF:= []字节{
  0x01，0x00，0x00，0x00，//版本1
  0x6F，0xE2，0x8C，0x0A，0xB6，0xF1，0xB3，0x72，
  0xc1、0xa6、0xa2、0x46、0xae、0x63、0xf7、0x4f、
  0X93、0X1E、0X83、0X65、0XE1、0X5A、0X8、0X9C、
  0x68，0xd6，0x19，0x00，0x00，0x00，0x00，0x00，0x00，//prevblock
  0x3b，0xa3，0xed，0xfd，0x7a，0x7b，0x12，0xb2，
  0x7a、0xc7、0x2c、0x3e、0x67、0x76、0x8f、0x61、
  0x7F，0xC8，0x1B，0xC3，0x88，0x8A，0x51，0x32，
  0x3a，0x9f，0xb8，0xaa，0x4b，0x1e，0x5e，0x4a，//merkleroot
  0x29，0xab，0x5f，0x49，//时间戳
  0xff，0xff，0x00，0x1d，//位
  0xf3，0xe0，0x01，0x00，//nonce
  0x00，//txncount变量
 }
 r：=bytes.newreader（buf）
 var头段blockheader
 对于i：=0；i<b.n；i++
  R.BASE（0, 0）
  readBlockHeader（R、0和Header）
 }
}

//BenchmarkWriteBlockHeader对需要多长时间执行基准测试
//序列化块头。
func基准MarkWriteBlockHeader（b*testing.b）
 标题：=blockone.header
 对于i：=0；i<b.n；i++
  WriteBlockHeader（ioutil.discard、0和Header）
 }
}

//BenchmarkDecodeGetHeaders对需要多长时间执行基准测试
//使用最大块定位器哈希数解码GetHeaders消息。
func基准标记解码头（b*testing.b）
 //创建具有最大块定位器数的消息。
 pVer：=协议版本
 变量管理器
 对于i：=0；i<maxblocklocatorspermsg；i++
  hash，错误：=chainhash.newhashfromstr（fmt.sprintf（“%x”，i））。
  如果犯错！= nIL{
   b.fatalf（“newhashfromstr:意外错误：%v”，err）
  }
  m.addBlockLocatorHash（哈希）
 }

 //将其序列化，以便字节可用于测试下面的解码。
 变量bb字节缓冲区
 如果错误：=m.btcencode（&bb，pver，latestencoding）；错误！= nIL{
  b.fatalf（“msggetheaders.btcencode:意外错误：%v”，err）
 }
 buf：=bb.bytes（）。

 r：=bytes.newreader（buf）
 var msg msggetheaders变量消息
 ReTimeTime（）
 对于i：=0；i<b.n；i++
  R.BASE（0, 0）
  消息btcdecode（r，pver，latestencoding）
 }
}

//BenchmarkDecodeHeaders对需要多长时间执行基准测试
//使用最大头数解码头消息。
func基准解码头（b*testing.b）
 //创建具有最大头数的消息。
 pVer：=协议版本
 变量m m m m m s起重机
 对于i：=0；i<maxblockheaderspemsg；i++
  hash，错误：=chainhash.newhashfromstr（fmt.sprintf（“%x”，i））。
  
   b.fatalf（“newhashfromstr:意外错误：%v”，err）
  }
  m.addBlockHeader（newBlockHeader（1，哈希，哈希，0，uint32（i）））
 }

 //将其序列化，以便字节可用于测试下面的解码。
 变量bb字节缓冲区
 如果错误：=m.btcencode（&bb，pver，latestencoding）；错误！= nIL{
  b.fatalf（“msgheaders.btcencode:意外错误：%v”，err）
 }
 buf：=bb.bytes（）。

 r：=bytes.newreader（buf）
 变量消息MSgheaders
 ReTimeTime（）
 对于i：=0；i<b.n；i++
  R.BASE（0, 0）
  消息btcdecode（r，pver，latestencoding）
 }
}

//BenchmarkDecodeGetBlocks对需要多长时间执行基准测试
//使用块定位器哈希的最大数目解码GetBlocks消息。
func基准标记解码块（b*testing.b）
 //创建具有最大块定位器数的消息。
 pVer：=协议版本
 变量m msggetblock
 对于i：=0；i<maxblocklocatorspermsg；i++
  hash，错误：=chainhash.newhashfromstr（fmt.sprintf（“%x”，i））。
  如果犯错！= nIL{
   b.fatalf（“newhashfromstr:意外错误：%v”，err）
  }
  m.addBlockLocatorHash（哈希）
 }

 //将其序列化，以便字节可用于测试下面的解码。
 变量bb字节缓冲区
 如果错误：=m.btcencode（&bb，pver，latestencoding）；错误！= nIL{
  b.fatalf（“msggetblocks.btcencode:意外错误：%v”，err）
 }
 buf：=bb.bytes（）。

 r：=bytes.newreader（buf）
 变量消息msggetblocks
 ReTimeTime（）
 对于i：=0；i<b.n；i++
  R.BASE（0, 0）
  消息btcdecode（r，pver，latestencoding）
 }
}

//BenchmarkDecodeAddr对解码
//地址最大的addr消息。
func基准解码地址（b*testing.b）
 //创建具有最大地址数的消息。
 pVer：=协议版本
 ip：=net.parseip（“127.0.0.1”）。
 ma：=newmsgaddr（）。
 对于端口：=uint16（0）；port<maxaddrpermsg；port++
  ma.addaddress（newnetaddressipport（ip，port，sfnodenetwork））。
 }

 //将其序列化，以便字节可用于测试下面的解码。
 变量bb字节缓冲区
 如果错误：=ma.btcencode（&bb，pver，latestencoding）；err！= nIL{
  b.fatalf（“msgaddr.btcencode:意外错误：%v”，err）
 }
 buf：=bb.bytes（）。

 r：=bytes.newreader（buf）
 VaR MSG MSGADR
 ReTimeTime（）
 对于i：=0；i<b.n；i++
  R.BASE（0, 0）
  消息btcdecode（r，pver，latestencoding）
 }
}

//benchmark decode inv对一个inv的解码时间进行基准测试
//最大条目数的消息。
func基准解码inv（b*testing.b）
 //创建最大条目数的消息。
 pVer：=协议版本
 VAR M MSGVIN
 对于i：=0；i<maxinvpermsg；i++
  hash，错误：=chainhash.newhashfromstr（fmt.sprintf（“%x”，i））。
  如果犯错！= nIL{
   b.fatalf（“newhashfromstr:意外错误：%v”，err）
  }
  m.addinvvect（newinvvect（invtyblock，hash））。
 }

 //将其序列化，以便字节可用于测试下面的解码。
 变量bb字节缓冲区
 如果错误：=m.btcencode（&bb，pver，latestencoding）；错误！= nIL{
  b.fatalf（“msginv.btcencode:意外错误：%v”，err）
 }
 buf：=bb.bytes（）。

 r：=bytes.newreader（buf）
 VaR MSG MSGVIN
 ReTimeTime（）
 对于i：=0；i<b.n；i++
  R.BASE（0, 0）
  消息btcdecode（r，pver，latestencoding）
 }
}

//BenchmarkDecodeNotFound对解码所需的时间执行基准测试
//notfound消息的最大条目数。
func基准标记解码未找到（b*testing.b）
 //创建最大条目数的消息。
 pVer：=协议版本
 变量m msgnotfound
 对于i：=0；i<maxinvpermsg；i++
  hash，错误：=chainhash.newhashfromstr（fmt.sprintf（“%x”，i））。
  如果犯错！= nIL{
   b.fatalf（“newhashfromstr:意外错误：%v”，err）
  }
  m.addinvvect（newinvvect（invtyblock，hash））。
 }

 //将其序列化，以便字节可用于测试下面的解码。
 变量bb字节缓冲区
 如果错误：=m.btcencode（&bb，pver，latestencoding）；错误！= nIL{
  b.fatalf（“msgnotfound.btcencode:意外错误：%v”，err）
 }
 buf：=bb.bytes（）。

 r：=bytes.newreader（buf）
 变量消息msgnotfound
 ReTimeTime（）
 对于i：=0；i<b.n；i++
  R.BASE（0, 0）
  消息btcdecode（r，pver，latestencoding）
 }
}

//BenchmarkDecodeMerkleBlock对需要多长时间执行基准测试
//解码合理大小的merkleblock消息。
func基准码解码块（b*testing.b）
 //创建包含随机数据的消息。
 pVer：=协议版本
 变量m msgmerkleblock
 hash，错误：=chainhash.newhashfromstr（fmt.sprintf（“%x”，10000））。
 如果犯错！= nIL{
  b.fatalf（“newhashfromstr:意外错误：%v”，err）
 }
 m.header=*newblockheader（1，hash，hash，0，uint32（10000））。
 对于i：=0；i<105；i++
  hash，错误：=chainhash.newhashfromstr（fmt.sprintf（“%x”，i））。
  如果犯错！= nIL{
   b.fatalf（“newhashfromstr:意外错误：%v”，err）
  }
  m.addtxshash（哈希）
  如果i % 8＝＝0 {
   m.flags=附加（m.flags，uint8（i））
  }
 }

 //将其序列化，以便字节可用于测试下面的解码。
 变量bb字节缓冲区
 如果错误：=m.btcencode（&bb，pver，latestencoding）；错误！= nIL{
  b.fatalf（“msgmerkleblock.btcencode:意外错误：%v”，err）
 }
 buf：=bb.bytes（）。

 r：=bytes.newreader（buf）
 变量消息msgmerkleblock
 ReTimeTime（）
 对于i：=0；i<b.n；i++
  R.BASE（0, 0）
  消息btcdecode（r，pver，latestencoding）
 }
}

//BenchmarkTxHash对哈希
/ /事务。
func基准标记txshash（b*testing.b）
 对于i：=0；i<b.n；i++
  genesisCoinBaseTx.txshash（）。
 }
}

//BenchmarkDoubleHashB对执行
//返回字节片的双哈希。
func基准双峰（b*testing.b）
 var buf bytes.buffer
 如果错误：=genesisCoinBaseTx.serialize（&buf）；错误！= nIL{
  b.errorf（“序列化：意外错误：%v”，err）
  返回
 }
 txbytes：=buf.bytes（）。

 ReTimeTime（）
 对于i：=0；i<b.n；i++
  _ux=chainhash.doublehashb（txbytes）
 }
}

//BenchmarkDoubleHash根据执行所需的时间执行基准
//返回chainhash.hash的双哈希。
func基准双h（b*testing.b）
 var buf bytes.buffer
 如果错误：=genesisCoinBaseTx.serialize（&buf）；错误！= nIL{
  b.errorf（“序列化：意外错误：%v”，err）
  返回
 }
 txbytes：=buf.bytes（）。

 ReTimeTime（）
 对于i：=0；i<b.n；i++
  _ux=chainhash.doublehash（txbytes）
 }
}
