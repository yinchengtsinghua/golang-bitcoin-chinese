
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2015 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wire

import (
	"encoding/binary"
	"io"
	"net"
	"time"
)

//max netaddress payload返回比特币netaddress的最大有效负载大小
//基于协议版本。
func maxNetAddressPayload(pver uint32) uint32 {
//服务8字节+IP 16字节+端口2字节。
	plen := uint32(26)

//NetAddressTimeVersion添加了时间戳字段。
	if pver >= NetAddressTimeVersion {
//时间戳4字节。
		plen += 4
	}

	return plen
}

//netaddress定义网络上对等机的信息，包括时间
//最后一次看到它，它支持的服务，它的IP地址和端口。
type NetAddress struct {
//上次看到地址时。不幸的是，这被编码为
//
//比特币版本信息（msgversion）中不存在，也不存在
//添加到协议版本>=netAddressTimeVersion。
	Timestamp time.Time

//标识地址支持的服务的位字段。
	Services ServiceFlag

//对等机的IP地址。
	IP net.IP

//对等机使用的端口。这是在线路上用big endian编码的
//这和其他的东西都不一样。
	Port uint16
}

//HASSERVICE返回地址是否支持指定的服务。
func (na *NetAddress) HasService(service ServiceFlag) bool {
	return na.Services&service == service
}

//addService通过生成
//消息。
func (na *NetAddress) AddService(service ServiceFlag) {
	na.Services |= service
}

//new netaddress ip port使用提供的IP、端口和返回新的网络地址
//支持的服务，其余字段为默认值。
func NewNetAddressIPPort(ip net.IP, port uint16, services ServiceFlag) *NetAddress {
	return NewNetAddressTimestamp(time.Now(), services, ip, port)
}

//newNetAddressTimestamp使用提供的
//时间戳、IP、端口和支持的服务。时间戳四舍五入为
//单秒精度。
func NewNetAddressTimestamp(
	timestamp time.Time, services ServiceFlag, ip net.IP, port uint16) *NetAddress {
//将时间戳限制为自协议以来的一秒精度
//不支持更好。
	na := NetAddress{
		Timestamp: time.Unix(timestamp.Unix(), 0),
		Services:  services,
		IP:        ip,
		Port:      port,
	}
	return &na
}

//new netaddress返回使用提供的TCP地址和
//支持的服务，其余字段为默认值。
func NewNetAddress(addr *net.TCPAddr, services ServiceFlag) *NetAddress {
	return NewNetAddressIPPort(addr.IP, uint16(addr.Port), services)
}

//readnetaddress根据协议从r中读取编码的netaddress
//版本以及时间戳是否包含在每个TS中。一些消息
//LIKE版本不包括时间戳。
func readNetAddress(r io.Reader, pver uint32, na *NetAddress, ts bool) error {
	var ip [16]byte

//注意：比特币协议使用uint32作为时间戳，因此它将
//在2106年左右停止工作。另外，时间戳直到
//协议版本>=netAddressTimeVersion
	if ts && pver >= NetAddressTimeVersion {
		err := readElement(r, (*uint32Time)(&na.Timestamp))
		if err != nil {
			return err
		}
	}

	err := readElements(r, &na.Services, &ip)
	if err != nil {
		return err
	}
//叹息。比特币协议混合了小尾数和大尾数。
	port, err := binarySerializer.Uint16(r, bigEndian)
	if err != nil {
		return err
	}

	*na = NetAddress{
		Timestamp: na.Timestamp,
		Services:  na.Services,
		IP:        net.IP(ip[:]),
		Port:      port,
	}
	return nil
}

//根据协议，writenetaddress将netaddress序列化为w
//版本以及时间戳是否包含在每个TS中。一些消息
//LIKE版本不包括时间戳。
func writeNetAddress(w io.Writer, pver uint32, na *NetAddress, ts bool) error {
//注意：比特币协议使用uint32作为时间戳，因此它将
//在2106年左右停止工作。另外，时间戳直到
//直到协议版本>=netAddressTimeVersion。
	if ts && pver >= NetAddressTimeVersion {
		err := writeElement(w, uint32(na.Timestamp.Unix()))
		if err != nil {
			return err
		}
	}

//确保始终写入16个字节，即使IP为零。
	var ip [16]byte
	if na.IP != nil {
		copy(ip[:], na.IP.To16())
	}
	err := writeElements(w, na.Services, ip)
	if err != nil {
		return err
	}

//叹息。比特币协议混合了小尾数和大尾数。
	return binary.Write(w, bigEndian, na.Port)
}
