
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

/*
封装线实现比特币线协议。

有关比特币协议的完整详细信息，请参见官方wiki条目。
网址：https://en.bitcoin.it/wiki/protocol_-specification。以下仅适用于
作为快速概述，提供有关如何使用包的信息。

在较高的层次上，此包为编组和解组提供支持
支持比特币信息进出线。这个包裹不行
消息处理的细节，例如当消息
收到。这为调用者提供了高度的灵活性。

比特币信息概述

比特币协议包括在对等方之间交换消息。各
消息前面有一个标题，用于标识有关消息的信息，例如
它是哪个比特币网络的一部分，它的类型，它有多大，以及校验和
验证有效性。消息头的所有编码和解码都由
这个包裹。

为了实现这一点，有一个名为
允许读取、写入或传递任何类型的消息的消息
通过渠道、功能等，另外，大多数
提供了当前支持的比特币信息。对于这些支持
消息，所有与
使用比特币编码的电线被处理，因此呼叫方不必担心
他们的具体情况。

消息交互

以下简要介绍了比特币信息的用途
相互交流。如上所述，这些相互作用不是
由这个包裹直接处理。有关
适当的互动，参见官方比特币协议wiki条目
https://en.bitcoin.it/wiki/protocol_规范。

最初的握手是由两个对等端互相发送版本消息组成的
（msgversion）然后用verack消息（msgverack）响应。两个
对等方使用版本消息（msgversion）中的信息进行协商
协议版本和相互支持的服务。一次
初始握手完成，下表显示消息
没有特定顺序的交互。

 对等A发送对等B响应
 ————————————————————————————————————————————————————————————————————————————————————————————————————————————————
 getaddr消息（msggetaddr）addr消息（msgaddr）
 getBlocks消息（msggetBlocks）inv消息（msginv）
 inv message（msginv）getdata message（msggetdata）
 getdata message（msggetdata）block message（msgblock）-或-
                                       Tx消息（MSGTX）-或-
                                       找不到消息（msgnotfound）
 getheaders消息（msggetaders）headers消息（msgheaders）
 ping消息（msgping）pong消息（msgheaders）*-或-
                                       （无——发送消息的能力就足够了）

 笔记：
 *在定义的更高协议版本之前，没有添加pong消息。
   在BIP031中。bip0031version常量可用于检测最近
   为此目的提供足够的协议版本（版本>bip0031版本）。

常用参数

在使用此包进行读取时，会出现几个常见的参数
写比特币信息。以下部分简要介绍了
这些参数使下一节可以建立在它们之上。

协议版本

协议版本应该在更高的位置与远程对等机协商。
但是，通过版本（msgversion）消息交换使其级别高于此包，
此包提供Wire.ProtocolVersion常量，该常量指示
此包支持的最新协议版本，通常是要使用的值
对于可能更低的协议版本之前的所有出站连接
谈判。

比特币网络

比特币网络是一个神奇的数字，用于识别
消息和消息适用的比特币网络。此包提供
以下常量：

 有线电视网
 Wire.testnet（回归测试网络）
 Wire.testNet3（测试网络版本3）
 Wire.Simnet（模拟测试网络）

确定消息类型

如比特币消息概述部分所述，此包
并使用名为message的通用接口编写比特币消息。在
要确定消息的实际具体类型，请使用类型
开关或类型断言。类型开关的示例如下：

 //假定msg已经是有效的具体消息，例如
 //通过newmsgversion或readmessage读取。
 开关消息：=msg.（类型）
 外壳*电线.msg版本：
  //该消息是指向msgversion结构的指针。
  fmt.printf（“协议版本：%v”，msg.protocol version）
 外壳*线.msgblock：
  //消息是指向msgblock结构的指针。
  fmt.printf（“块中Tx的数目：%v”，msg.header.txncount）
 }

正在读取消息

为了取消比特币信息的标记，请使用readmessage
功能。它接受任何IO.reader，但通常这是一个net.conn to
运行比特币对等机的远程节点。示例语法为：

 //读取并验证来自conn的下一比特币消息，使用
 //协议版本pver和比特币网络btcnet。回报
 //是Wire.Message，包含未编址的
 //原始负载和可能的错误。
 msg，rawpayload，err:=wire.readmessage（conn，pver，btcnet）
 如果犯错！= nIL{
  //记录并处理错误
 }

正在写入消息

要将比特币消息整理到有线，请使用writemessage
功能。它接受任何IO.Writer，但通常这是一个net.conn to
运行比特币对等机的远程节点。请求地址的语法示例
从远程对等机：

 //创建新的getaddr比特币消息。
 消息：=wire.newmsggetaddr（）

 //使用协议版本将比特币消息msg写入conn
 //pver和比特币网络btcnet。返回是可能的
 /错误。
 错误：=Wire.WriteMessage（conn、msg、pver、btcnet）
 如果犯错！= nIL{
  //记录并处理错误
 }

错误

此包返回的错误可能是底层提供的原始错误
从诸如io.eof、io.errUnexpectedeof和
IO.errShortWrite或Wire.MessageError类型。这允许呼叫者
通过类型区分常规IO错误和格式错误的消息
断言。

比特币改进建议

此包包括以下BIP概述的规范更改：

 bip0014（https://github.com/bitcoin/bips/blob/master/bip-0014.mediawiki）
 bip0031（https://github.com/bitcoin/bips/blob/master/bip-0031.mediawiki）
 bip0035（https://github.com/bitcoin/bips/blob/master/bip-0035.mediawiki）
 bip0037（https://github.com/bitcoin/bips/blob/master/bip-0037.mediawiki）
 bip0111（https://github.com/bitcoin/bips/blob/master/bip-0111.mediawiki）
 bip0130（https://github.com/bitcoin/bips/blob/master/bip-0130.mediawiki）
 bip0133（https://github.com/bitcoin/bips/blob/master/bip-0133.mediawiki）
**/

package wire
