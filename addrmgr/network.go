
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2014 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package addrmgr

import (
	"fmt"
	"net"

	"github.com/btcsuite/btcd/wire"
)

var (
//rfc1918nets指定由定义的IPv4专用地址块
//根据RFC1918（10.0.0.0/8、172.16.0.0/12和192.168.0.0/16）。
	rfc1918Nets = []net.IPNet{
		ipNet("10.0.0.0", 8, 32),
		ipNet("172.16.0.0", 12, 32),
		ipNet("192.168.0.0", 16, 32),
	}

//rfc2544net指定由rfc2544定义的IPv4块
//（1981 .18.0／15）
	rfc2544Net = ipNet("198.18.0.0", 15, 32)

//rfc3849net指定定义的IPv6文档地址块
//根据RFC3849（2001:DB8:：/32）。
	rfc3849Net = ipNet("2001:DB8::", 32, 128)

//rfc3927net将IPv4自动配置地址块指定为
//由RFC3927（169.254.0.0/16）定义。
	rfc3927Net = ipNet("169.254.0.0", 16, 32)

//rfc3964net将IPv6到IPv4封装地址块指定为
//由RFC3964（2002:：/16）定义。
	rfc3964Net = ipNet("2002::", 16, 128)

//rfc4193net指定定义的IPv6唯一本地地址块
//根据RFC4193（fc00:：/7）。
	rfc4193Net = ipNet("FC00::", 7, 128)

//rfc4380net指定UDP地址块上的IPv6 Teredo隧道
//如RFC4380（2001:：/32）所定义。
	rfc4380Net = ipNet("2001::", 32, 128)

//rfc4843net指定由定义的IPv6兰花地址块
//RFC4843（2001:10:：/28）。
	rfc4843Net = ipNet("2001:10::", 28, 128)

//rfc4862net指定IPv6无状态地址自动配置
//由rfc4862（fe80:：/64）定义的地址块。
	rfc4862Net = ipNet("FE80::", 64, 128)

//rfc5737net指定定义的IPv4文档地址块
//根据RFC5737（192.0.2.0/24、198.51.100.0/24、203.0.113.0/24）
	rfc5737Net = []net.IPNet{
		ipNet("192.0.2.0", 24, 32),
		ipNet("198.51.100.0", 24, 32),
		ipNet("203.0.113.0", 24, 32),
	}

//rfc6052net将IPv6众所周知的前缀地址块指定为
//由RFC6052（64:ff9b:：/96）定义。
	rfc6052Net = ipNet("64:FF9B::", 96, 128)

//rfc6145net将IPv6到IPv4的转换地址范围指定为
//由RFC6145（：：ffff:0:0:0/96）定义。
	rfc6145Net = ipNet("::FFFF:0:0:0", 96, 128)

//rfc6598net指定由rfc6598（100.64.0.0/10）定义的IPv4块
	rfc6598Net = ipNet("100.64.0.0", 10, 32)

//OnOncatnet定义用于支持ToR的IPv6地址块。
//比特币通过解码
//洋葱前的地址（即键散列）基32变为10
//字节数。然后它将地址的前6个字节存储为
//0xfd，0x87，0xd8，0x7e，0xeb，0x43。
//
//这是OnOncat使用的相同范围，它是
//RFC4193独特的本地IPv6范围。
//
//总之，格式为：
//魔术6字节，10字节base32密钥哈希解码
	onionCatNet = ipNet("fd87:d87e:eb43::", 48, 128)

//Zero4net定义以0开头的地址的IPv4地址块
//（0.0.0.0／8）。
	zero4Net = ipNet("0.0.0.0", 8, 32)

//Henet定义了Hurricane Electric IPv6地址块。
	heNet = ipNet("2001:470::", 32, 128)
)

//IPNET返回给定传递的IP地址字符串、数字的net.ip net结构
//在屏蔽开始时包含的一个位，以及位的总数
//为了面具。
func ipNet(ip string, ones, bits int) net.IPNet {
	return net.IPNet{IP: net.ParseIP(ip), Mask: net.CIDRMask(ones, bits)}
}

//ISIPv4返回给定地址是否为IPv4地址。
func IsIPv4(na *wire.NetAddress) bool {
	return na.IP.To4() != nil
}

//is local返回给定地址是否为本地地址。
func IsLocal(na *wire.NetAddress) bool {
	return na.IP.IsLoopback() || zero4Net.Contains(na.IP)
}

//isioncattor返回传递的地址是否在ipv6范围内
//比特币用于支持TOR（FD87:D87E:EB43:：/48）。注意这个范围
//是OnOncat使用的相同范围，它是RFC4193唯一本地
//IPv6的范围。
func IsOnionCatTor(na *wire.NetAddress) bool {
	return onionCatNet.Contains(na.IP)
}

//ISRFC1918返回传递的地址是否为IPv4的一部分
//由rfc1918定义的专用网络地址空间（10.0.0.0/8，
//172.16.0.0/12或192.168.0.0/16）。
func IsRFC1918(na *wire.NetAddress) bool {
	for _, rfc := range rfc1918Nets {
		if rfc.Contains(na.IP) {
			return true
		}
	}
	return false
}

//ISRFC2544返回传递的地址是否为IPv4的一部分
//RFC2544定义的地址空间（198.18.0.0/15）
func IsRFC2544(na *wire.NetAddress) bool {
	return rfc2544Net.Contains(na.IP)
}

//ISrfc3849返回传递的地址是否为IPv6的一部分
//文件范围见RFC3849（2001:DB8:：/32）。
func IsRFC3849(na *wire.NetAddress) bool {
	return rfc3849Net.Contains(na.IP)
}

//ISrfc3927返回传递的地址是否为IPv4的一部分
//根据RFC3927（169.254.0.0/16）定义的自动配置范围。
func IsRFC3927(na *wire.NetAddress) bool {
	return rfc3927Net.Contains(na.IP)
}

//ISrfc3964返回传递的地址是否为IPv6的一部分
//rfc3964（2002:：/16）定义的IPv4封装范围。
func IsRFC3964(na *wire.NetAddress) bool {
	return rfc3964Net.Contains(na.IP)
}

//ISrfc4193返回传递的地址是否为IPv6的一部分
//由rfc4193（fc00:：/7）定义的唯一本地范围。
func IsRFC4193(na *wire.NetAddress) bool {
	return rfc4193Net.Contains(na.IP)
}

//ISrfc4380返回传递的地址是否为IPv6的一部分
//根据rfc4380（2001:：/32）定义的UDP范围内的Teredo隧道。
func IsRFC4380(na *wire.NetAddress) bool {
	return rfc4380Net.Contains(na.IP)
}

//ISrfc4843返回传递的地址是否为IPv6的一部分
//兰花的范围由RFC4843定义（2001:10:：/28）。
func IsRFC4843(na *wire.NetAddress) bool {
	return rfc4843Net.Contains(na.IP)
}

//ISrfc4862返回传递的地址是否为IPv6的一部分
//根据RFC4862（FE80:：/64）定义的无状态地址自动配置范围。
func IsRFC4862(na *wire.NetAddress) bool {
	return rfc4862Net.Contains(na.IP)
}

//ISRFC5737返回传递的地址是否为IPv4的一部分
//文件地址空间如RFC5737（192.0.2.0/24，
//198.51.100.0/24、203.0.113.0/24）
func IsRFC5737(na *wire.NetAddress) bool {
	for _, rfc := range rfc5737Net {
		if rfc.Contains(na.IP) {
			return true
		}
	}

	return false
}

//ISRFC6052返回传递的地址是否为IPv6的一部分
//由rfc6052（64:ff9b:：/96）定义的已知前缀范围。
func IsRFC6052(na *wire.NetAddress) bool {
	return rfc6052Net.Contains(na.IP)
}

//ISrfc6145返回传递的地址是否为IPv6的一部分
//由rfc6145（：：ffff:0:0:0/96）定义的IPv4转换地址范围。
func IsRFC6145(na *wire.NetAddress) bool {
	return rfc6145Net.Contains(na.IP)
}

//ISRFC6598返回传递的地址是否为IPv4的一部分
//由RFC6598（100.64.0.0/10）指定的共享地址空间
func IsRFC6598(na *wire.NetAddress) bool {
	return rfc6598Net.Contains(na.IP)
}

//is valid返回传递的地址是否有效。地址是
//在下列情况下被视为无效：
//IPv4：它是一个零位或全位设置地址。
//ipv6:它是零或rfc3849文档地址。
func IsValid(na *wire.NetAddress) bool {
//如果地址为0，isUnspeciated将返回，因此仅设置所有位，并且
//需要明确检查RFC3849。
	return na.IP != nil && !(na.IP.IsUnspecified() ||
		na.IP.Equal(net.IPv4bcast))
}

//is routable返回传递的地址是否可路由
//公共互联网。只要地址有效而不是
//在任何保留范围内。
func IsRoutable(na *wire.NetAddress) bool {
	return IsValid(na) && !(IsRFC1918(na) || IsRFC2544(na) ||
		IsRFC3927(na) || IsRFC4862(na) || IsRFC3849(na) ||
		IsRFC4843(na) || IsRFC5737(na) || IsRFC6598(na) ||
		IsLocal(na) || (IsRFC4193(na) && !IsOnionCatTor(na)))
}

//groupkey返回一个字符串，该字符串表示地址是网络组的一部分
//的。这是IPv4的/16，IPv6的/32（/36），字符串
//“local”对于本地地址，字符串“tor:key”，其中key是
//Tor地址的洋葱地址，不可输出的字符串“unroutable”
//地址。
func GroupKey(na *wire.NetAddress) string {
	if IsLocal(na) {
		return "local"
	}
	if !IsRoutable(na) {
		return "unroutable"
	}
	if IsIPv4(na) {
		return na.IP.Mask(net.CIDRMask(16, 32)).String()
	}
	if IsRFC6145(na) || IsRFC6052(na) {
//最后四个字节是IP地址
		ip := na.IP[12:16]
		return ip.Mask(net.CIDRMask(16, 32)).String()
	}

	if IsRFC3964(na) {
		ip := na.IP[2:6]
		return ip.Mask(net.CIDRMask(16, 32)).String()

	}
	if IsRFC4380(na) {
//Teredo隧道的最后4个字节是v4地址xor
//0xFF。
		ip := net.IP(make([]byte, 4))
		for i, byte := range na.IP[12:16] {
			ip[i] = byte ^ 0xff
		}
		return ip.Mask(net.CIDRMask(16, 32)).String()
	}
	if IsOnionCatTor(na) {
//组是从实际洋葱键的前4位键入的。
		return fmt.Sprintf("tor:%d", na.IP[6]&((1<<4)-1))
	}

//好吧，现在我们知道自己是一个ipv6地址。
//比特币使用/32，除了飓风电力
//（he.net）IP范围，它使用/36。
	bits := 32
	if heNet.Contains(na.IP) {
		bits = 36
	}

	return na.IP.Mask(net.CIDRMask(bits, 128)).String()
}
