
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

/*
btcd是一个用go编写的完整节点比特币实现。

The default options are sane for most users.  This means btcd will work 'out of
对于大多数用户来说。但是，也有许多不同的标志
可以用来控制它。

以下部分提供了一个枚举标志的用法概述。安
有趣的是，所有这些选项的长形式
（除了-c）可以在自动
在BTCD启动时分析。默认情况下，配置文件位于
posix风格操作系统上的~/.btcd/btcd.conf和%localappdata%\btcd\btcd.conf
在Windows上。如下所示，-c（--configfile）标志可用于重写
这个位置。

用途：
  BTCD [期权]

应用程序选项：
  -v，-version显示版本信息并退出
  -c，--configfile=配置文件的路径
  -b，--datadir=存储数据的目录
      --logdir=要记录输出的目录。
  -a，--add peer=在启动时添加要连接的对等机
      --connect=启动时仅连接到指定的对等端
      --nolisten禁用侦听传入连接--注意：
                            如果--connect
                            或者--使用代理选项时不同时指定
                            通过监听接口——监听
      --listen=添加接口/端口以侦听连接
                            （默认所有接口端口：8333，TESTNET:18333）
      --max peers=最大入站和出站对等数（125）
      ——禁止对行为不端的同龄人禁用禁令
      --banduration=禁止行为不端的同龄人多长时间。有效时间单位
                            是s，m，h。最小1秒（24h0m0s）
      --banthreshold=断开连接和
                            禁止行为不端的同龄人。
      --whitelist=添加一个不会被禁止的IP网络或IP。
                            （例如192.168.1.0/24或：：1）
  -u，-rpcuser=rpc连接的用户名
  -p，--rpcpass=rpc连接的密码
      --rpclimituser=有限RPC连接的用户名
      --rpclimitpass=有限RPC连接的密码
      --rpc listen=添加接口/端口以侦听RPC连接
                            （默认端口：8334，TESTNET:18334）
      --rpccert=包含证书文件的文件
      --RPCKEY =包含证书密钥的文件
      --rpc max clients=标准连接的最大RPC客户端数
                            （10）
      --rpcmaxwebsockets=   Max number of RPC websocket connections (25)
      --rpc quirks反映了比特币核心的一些json-rpc特性--注：
                            除非互操作性问题需要
                            be worked around
      --norpc禁用内置的rpc服务器--注意：rpc服务器
                            如果没有rpcuser/rpcpass或
                            指定rpclimituser/rpclimitpass
      --notls为rpc服务器禁用tls--注意：这只是
                            如果RPC服务器绑定到本地主机，则允许
      --nodnsseed           Disable DNS seeding for peers
      --externalIP=将IP添加到我们声明的本地地址列表中
                            听同龄人说
      --代理=通过socks5代理连接（例如127.0.0.1:9050）
      --proxyuser=代理服务器的用户名
      --proxypass=代理服务器的密码
      --洋葱=通过socks5代理连接到tor隐藏服务
                            （例如127.0.0.1:9050）
      --onionuser=洋葱代理服务器的用户名
      --onionpass=洋葱代理服务器的密码
      --noonion禁用连接到tor隐藏服务
      --Torisolation通过随机化用户启用Tor流隔离
                            每个连接的凭据。
      --测试网使用测试网
      利用回归测试网络
      ——Simnet使用模拟测试网络
      --AdvestalPosit=添加自定义检查点。格式：'Health>：<hash >
      --nocheckpoints禁用内置检查点。不要这样做除非
                            你知道你在做什么。
      --uacomment=要添加到用户代理的注释--
                            See BIP 14 for more information.
      --dbtype=用于块链（ffldb）的数据库后端
      --profile=在给定端口上启用HTTP分析--注意端口
                            必须介于1024和65536之间
      --cpu profile=将cpu配置文件写入指定文件
  -d, --debuglevel=         Logging level for all subsystems {trace, debug,
                            信息、警告、错误、关键--您还可以指定
                            <subsystem>=<level>，<subsystem2>=<level>，…设置
                            the log level for individual subsystems -- Use show
                            列出可用的子系统（信息）
      --upnp使用upnp在nat外映射监听端口
      --MnRelayTXXFI= BTC/KB中的最低交易费用
                            视为非零费用。
      --limitfreerrelay=无交易费交易的限制中继
                            以千字节为单位
                            分钟（15）
      --正常优先级不要求免费或低收费交易
                            中继优先级高
      --maxorphantx=要在内存中保留的最大孤立事务数
                            （100）
      --使用CPU生成（挖掘）比特币
      --miningaddr=将指定的付款地址添加到
                            addresses to use for generated blocks -- At least
                            如果生成选项是
                            设置
      --blockminSize=创建时要使用的最小块大小（以字节为单位）
                            一个街区
      --blockmaxsize=创建时要使用的最大块大小（以字节为单位）
                            一个街区（750000）
      --blockPrioritySize=高优先级/低费用事务的字节大小
                            创建块时（50000）
      --nopeerbloomfilters禁用bloom过滤支持。
      --nocfilters          Disable committed filtering (CF) support.
      --sigCacheMaxSize=签名中的最大条目数
                            验证缓存。
      --BlocksOnly不接受来自远程对等方的事务。
      --relaynonstd中继非标准事务，无论
                            活动网络的默认设置。
      --rejectnonstd        Reject non-standard transactions regardless of the
                            活动网络的默认设置。

帮助选项：
  -h, --help           Show this help message

**/

package main
