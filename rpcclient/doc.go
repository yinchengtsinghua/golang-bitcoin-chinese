
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2014-2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

/*
package rpc client实现一个支持websocket的比特币json-rpc客户端。

概述

此客户机提供了一个健壮且易于使用的客户机，用于与
使用BTCD/比特币核心兼容比特币JSON-RPC的比特币RPC服务器
应用程序编程接口。此客户端已使用btcd（https://github.com/btcsuite/btcd）进行测试，
btcwallet（https://github.com/btcsuite/btcwallet），和
比特币核心（https://github.com/bitcoin）。

除了兼容的标准HTTP Post JSON-RPC API、BTCD和
btcwallet提供的WebSocket接口比标准接口更高效
访问RPC的HTTP Post方法。以下章节讨论了这些差异
在HTTP Post和WebSockets之间。

默认情况下，此客户端假定RPC服务器支持WebSockets，并且
启用TLS。实际上，这意味着它假定您正在与
默认为btcd或btcwallet。但是，配置选项提供给
返回HTTP Post并禁用TLS以支持与劣质比特币对话
核心风格的RPC服务器。

WebSockets与HTTP Post

在基于HTTP Post的JSON-RPC中，每个请求都创建一个新的HTTP连接，
发出呼叫，等待响应，然后关闭连接。这增加了
每次调用都会有很大的开销，并且缺乏灵活性，例如
通知。

相反，btcd和
btcwallet仅使用保持打开并允许
异步双向通信。

WebSocket接口支持与HTTP Post相同的所有命令，但是它们
可以调用，而不必为每个
打电话。此外，WebSocket接口还提供了其他不错的功能，如
注册不同事件的异步通知的能力。

同步与异步API

客户端提供同步（阻塞）和异步API。

对于大多数用例，同步（阻塞）API通常是足够的。它
通过发出RPC和阻塞工作，直到收到响应。这个
允许简单的代码，只要函数
返回。

异步API基于未来的概念。当您调用异步时
命令的版本，它将快速返回承诺的类型的实例
以在将来某个时间提供RPC的结果。在背景中，
发出RPC调用，结果存储在返回的实例中。调用
返回的实例上的Receive方法将返回结果
如果它已经到达，立即停止，直到它到达。这是有用的
因为它为调用者提供了对并发性的更大控制。

通知

通知的第一个重要部分是认识到它们只会
通过WebSockets连接时工作。这在直觉上应该是合理的
因为HTTP POST模式不保持连接打开！

BTCD提供的所有通知都需要注册才能选择加入。例如，
如果您希望在收到一组地址的资金时得到通知，您可以
通过notifyReceived（或notifyReceivedAsync）函数注册地址。

通知处理程序

客户端通过使用回调处理程序来公开通知
它是通过由
创建客户端时调用方。

这些通知处理程序必须快速完成，因为它们
在主读取循环中故意阻止进一步的读取，直到
他们完成了。这使调用者能够灵活地决定
当通知的传入速度比处理的速度快时执行此操作。

特别是这意味着从回调处理程序发出阻塞的RPC调用
将导致死锁，因为在回调之前将无法读取更多服务器响应
返回，但回调将等待响应。因此，任何
额外的RPC必须以完全分离的方式发布。

自动重新连接

默认情况下，在WebSockets模式下运行时，此客户端将自动
如果连接断开，请继续尝试重新连接到RPC服务器。那里
是每次连接尝试之间的后退，直到达到每一次尝试
分钟。一旦重新建立连接，所有以前注册的
通知自动重新注册，任何飞行中的命令
重新发行。这意味着从调用者的角度来看，请求只需要
完成时间更长。

调用方可以在客户端上调用Shutdown方法来强制客户端
停止重新连接尝试并返回所有未完成的errclientshutdown
命令。

通过设置DisableAutoReconnect，可以禁用自动重新连接。
在创建客户端时，在连接配置中标记为true。

轻微的RPC服务器差异和链/钱包分离

有些命令是特定于特定RPC服务器的扩展。为了
例如，debuglevel调用是仅由btcd提供的扩展（和
btcwallet直通）。因此，如果您调用这些命令之一
如果RPC服务器不提供它们，您将得到一个未实现的错误。
从服务器。已经努力找出哪些命令是
文档中的扩展。

此外，重要的是要认识到BTCD有意将钱包分开
一个名为btcwallet的独立进程中的功能。这意味着如果你是
直接连接到BTCD RPC服务器，仅与
提供连锁服务。根据您的应用程序，您可能只能
需要链相关的RPC。相比之下，btcwallet提供直通治疗
对于链相关的RPC，它除了支持钱包相关的RPC外，还支持它们。

错误

在整个程序包中，将返回三类错误：

  -与客户端连接相关的错误，如身份验证、终结点、
    断开和关闭
  -在与远程RPC服务器通信之前发生的错误，例如
    命令创建和封送错误或与远程通信的问题
    服务器
  -从远程RPC服务器返回的错误，如未执行的命令，
    不存在请求的块和事务、格式错误的数据和不正确的
    网络

第一类错误通常是errInvalidauth，
errInvalidEndpoint、errClientDisconnect或errClientShutdown。

注意：除非
由于客户端自动处理，因此设置了DisableAutoReconnect标志
如前所述，默认情况下重新连接。

第二类错误通常表示程序员错误，因此
类型可能有所不同，但通常最好通过简单地显示/记录来处理。
它。

第三类错误，即服务器返回的错误，可以是
由断言*btcjson.rpcerror中的错误的类型检测。例如，到
检测远程RPC服务器是否未执行命令：

  金额，错误：=client.getbalance（“”）
  如果犯错！= nIL{
   如果是jerr，则确定：=err.（*btcjson.rpcerror）；确定
    开关Jerr.代码
    案例btcjson.errrpcunimplemented:
     //句柄未实现错误

    //处理您关心的其他特定错误
  }
   }

   //记录或以其他方式处理错误，知道它不是返回的错误
   //来自远程RPC服务器。
  }

示例用法

示例目录中有以下完整的客户端示例：

 -比特币核心HTTP
   在禁用TLS的情况下，使用HTTP POST模式连接到比特币核心RPC服务器
   并获取当前块计数
 -BTCDWebSockets
   使用TLS安全WebSockets连接到BTCD RPC服务器，注册
   阻止已连接和已断开连接的通知，并获取当前
   块计数
 -btcwalletwebsockets
   使用TLS安全WebSockets、寄存器连接到btcwallet RPC服务器
   有关帐户余额更改的通知，并获取
   钱包可以签名的未用交易输出（utxos）
**/

package rpcclient
