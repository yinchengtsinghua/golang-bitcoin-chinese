
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

/*
package mempool提供了一个策略强制的非限制比特币交易池。

A key responsbility of the bitcoin network is mining user-generated transactions
成块。为了促进这一点，采矿过程依赖于
包含在当前块中的现成的事务源
解决了的。

在较高的层次上，此包通过提供
完全验证的事务的内存池，也可以选择
根据可配置策略进一步筛选。

策略配置选项之一控制是否为“标准”
接受交易。实质上，“标准”交易是指
满足一组相当严格的要求，这些要求主要是为了帮助
为所有用户提供系统的合理使用。重要的是要注意
被认为是一个“标准”的交易随着时间的变化。为了了解更多信息，请访问
写这篇文章的时间，一个需要的一些标准的例子
对于一个被认为是标准的交易来说，它是最近的
支持的版本已定稿，不超过特定大小，仅由
特定的脚本形式。

因为这个套餐不涉及其他比特币的细节，比如网络
通信和事务中继，它返回
接受了这一点，使呼叫者在他们想要的方式上具有高度的灵活性。
继续进行。通常情况下，这将涉及到诸如中继事务之类的事情
发送给网络上的其他对等方，并通知挖掘进程
交易记录可用。

功能概述

以下是对主要功能的快速概述。不打算
做一份详尽的清单。

 - Maintain a pool of fully validated transactions
   -拒绝未完全使用的重复事务
   -拒绝CoinBase交易
   -拒绝双重支出（来自链和池中的其他事务）
   -根据网络共识规则拒绝无效交易
   使用签名缓存支持的完整脚本执行和验证
   -个人交易查询支持
 -孤立事务支持（从未知输出支出的事务）
   - Configurable limits (see transaction acceptance policy)
   -自动添加不再是新孤立事务的孤立事务
     将事务添加到池中
   -单个孤立事务查询支持
 -可配置的事务接受策略
   -接受或拒绝标准交易的选项
   -根据优先级计算接受或拒绝交易的选项
   -低收费和免费交易的利率限制
   -非零费用阈值
   - Max signature operations per transaction
   -最大孤立事务大小
   -允许的最大孤立事务数
 -每个事务的附加元数据跟踪
   -将事务添加到池的时间戳
   -将事务添加到池时的最新块高度
   -交易支付的费用
   -事务的启动优先级
 -手动控制事务删除
   -递归删除所有相关事务

错误

此包返回的错误可能是底层提供的原始错误
调用或类型为mempool.ruleError。因为有两类规则
（mempool接受规则和区块链（共识）接受规则），即
mempool.ruleError类型包含一个单独的错误字段，该字段反过来将
be a mempool.TxRuleError or a blockchain.RuleError.  The first indicates a
违反了MEMPOOL验收规则，而MEMPOOL验收规则表示违反了
共识接受规则。这使呼叫者能够轻松区分
在数据库错误等意外错误与规则错误之间
通过类型断言违反。此外，调用者可以通过编程
通过将err字段断言为
the aforementioned types and examining their underlying ErrorCode field.
**/

package mempool
