
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

package connmgr

import (
	"fmt"
	mrand "math/rand"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
)

const (
//DNS种子代码使用这些常量来随机选取最后一个
//看时间。
	secondsIn3Days int32 = 24 * 60 * 60 * 3
	secondsIn4Days int32 = 24 * 60 * 60 * 4
)

//onseed是回调函数的签名，当dns
//播种成功。
type OnSeed func(addrs []*wire.NetAddress)

//lookupfunc是DNS查找函数的签名。
type LookupFunc func(string) ([]net.IP, error)

//seedFromDNS使用DNS种子设定来用对等方填充地址管理器。
func SeedFromDNS(chainParams *chaincfg.Params, reqServices wire.ServiceFlag,
	lookupFn LookupFunc, seedFn OnSeed) {

	for _, dnsseed := range chainParams.DNSSeeds {
		var host string
		if !dnsseed.HasFiltering || reqServices == wire.SFNodeNetwork {
			host = dnsseed.Host
		} else {
			host = fmt.Sprintf("x%x.%s", uint64(reqServices), dnsseed.Host)
		}

		go func(host string) {
			randSource := mrand.New(mrand.NewSource(time.Now().UnixNano()))

			seedpeers, err := lookupFn(host)
			if err != nil {
				log.Infof("DNS discovery failed on seed %s: %v", host, err)
				return
			}
			numPeers := len(seedpeers)

			log.Infof("%d addresses found from DNS seed %s", numPeers, host)

			if numPeers == 0 {
				return
			}
			addresses := make([]*wire.NetAddress, len(seedpeers))
//如果这个错误，那么我们就有*真正的*问题。
			intPort, _ := strconv.Atoi(chainParams.DefaultPort)
			for i, peer := range seedpeers {
				addresses[i] = wire.NewNetAddressTimestamp(
//比特币种子地址来自
//在3之间随机选择的时间
//7天前。
					time.Now().Add(-1*time.Second*time.Duration(secondsIn3Days+
						randSource.Int31n(secondsIn4Days))),
					0, peer, uint16(intPort))
			}

			seedFn(addresses)
		}(host)
	}
}
