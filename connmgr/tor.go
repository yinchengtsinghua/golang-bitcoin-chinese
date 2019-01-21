
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

package connmgr

import (
	"encoding/binary"
	"errors"
	"net"
)

const (
	torSucceeded         = 0x00
	torGeneralError      = 0x01
	torNotAllowed        = 0x02
	torNetUnreachable    = 0x03
	torHostUnreachable   = 0x04
	torConnectionRefused = 0x05
	torTTLExpired        = 0x06
	torCmdNotSupported   = 0x07
	torAddrNotSupported  = 0x08
)

var (
//errtorinvalidaddresResponse表示无效地址为
//由Tor DNS解析程序返回。
	ErrTorInvalidAddressResponse = errors.New("invalid address response")

//errtorinvalidProxyResponse指示Tor代理返回
//以意外格式响应。
	ErrTorInvalidProxyResponse = errors.New("invalid proxy response")

//errtorUnrecognizedAuthmethod表示身份验证方法
//无法识别提供的。
	ErrTorUnrecognizedAuthMethod = errors.New("invalid proxy authentication method")

	torStatusErrors = map[byte]error{
		torSucceeded:         errors.New("tor succeeded"),
		torGeneralError:      errors.New("tor general error"),
		torNotAllowed:        errors.New("tor not allowed"),
		torNetUnreachable:    errors.New("tor network is unreachable"),
		torHostUnreachable:   errors.New("tor host is unreachable"),
		torConnectionRefused: errors.New("tor connection refused"),
		torTTLExpired:        errors.New("tor TTL expired"),
		torCmdNotSupported:   errors.New("tor command not supported"),
		torAddrNotSupported:  errors.New("tor address type not supported"),
	}
)

//TorLookupIP使用Tor通过它们提供的SOCKS扩展解析DNS
//Tor网络上的分辨率。Tor本身不支持IPv6，所以
//也不是。
func TorLookupIP(host, proxy string) ([]net.IP, error) {
	conn, err := net.Dial("tcp", proxy)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	buf := []byte{'\x05', '\x01', '\x00'}
	_, err = conn.Write(buf)
	if err != nil {
		return nil, err
	}

	buf = make([]byte, 2)
	_, err = conn.Read(buf)
	if err != nil {
		return nil, err
	}
	if buf[0] != '\x05' {
		return nil, ErrTorInvalidProxyResponse
	}
	if buf[1] != '\x00' {
		return nil, ErrTorUnrecognizedAuthMethod
	}

	buf = make([]byte, 7+len(host))
buf[0] = 5      //协议版本
buf[1] = '\xF0' //Tor解析
buf[2] = 0      //保留的
buf[3] = 3      //Tor解析
	buf[4] = byte(len(host))
	copy(buf[5:], host)
buf[5+len(host)] = 0 //端口0

	_, err = conn.Write(buf)
	if err != nil {
		return nil, err
	}

	buf = make([]byte, 4)
	_, err = conn.Read(buf)
	if err != nil {
		return nil, err
	}
	if buf[0] != 5 {
		return nil, ErrTorInvalidProxyResponse
	}
	if buf[1] != 0 {
		if int(buf[1]) >= len(torStatusErrors) {
			return nil, ErrTorInvalidProxyResponse
		} else if err := torStatusErrors[buf[1]]; err != nil {
			return nil, err
		}
		return nil, ErrTorInvalidProxyResponse
	}
	if buf[3] != 1 {
		err := torStatusErrors[torGeneralError]
		return nil, err
	}

	buf = make([]byte, 4)
	bytes, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	if bytes != 4 {
		return nil, ErrTorInvalidAddressResponse
	}

	r := binary.BigEndian.Uint32(buf)

	addr := make([]net.IP, 1)
	addr[0] = net.IPv4(byte(r>>24), byte(r>>16), byte(r>>8), byte(r))

	return addr, nil
}
