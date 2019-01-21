
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package main

//从台北Torrent许可证获取的UPNP代码如下：
//版权所有（c）2010 Jack Palevich。版权所有。
//
//以源和二进制形式重新分配和使用，有或无
//允许修改，前提是以下条件
//遇见：
//
//*源代码的再分配必须保留上述版权。
//注意，此条件列表和以下免责声明。
//*二进制形式的再分配必须复制上述内容
//版权声明、此条件列表和以下免责声明
//在提供的文件和/或其他材料中，
//分布。
//*无论是谷歌公司的名称还是其
//贡献者可用于支持或推广源自
//本软件未经事先明确书面许可。
//
//本软件由版权所有者和贡献者提供。
//“原样”和任何明示或暗示的保证，包括但不包括
//仅限于对适销性和适用性的暗示保证
//不承认特定目的。在任何情况下，版权
//所有人或出资人对任何直接、间接、附带的，
//特殊、惩戒性或后果性损害（包括但不包括
//仅限于采购替代货物或服务；使用损失，
//数据或利润；或业务中断），无论如何引起的
//责任理论，无论是合同责任、严格责任还是侵权责任。
//（包括疏忽或其他）因使用不当而引起的
//即使已告知此类损坏的可能性。

//只需足够的UPNP就能转发端口
//

import (
	"bytes"
	"encoding/xml"
	"errors"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

//nat是表示nat遍历选项的接口，例如upnp或
//NAT-PMP。它提供了查询和操作此遍历的方法，以允许
//获得服务。
type NAT interface {
//从NAT外部获取外部地址。
	GetExternalAddress() (addr net.IP, err error)
//为协议（“udp”或“tcp”）添加从外部端口到的端口映射
//描述持续超时的内部端口。
	AddPortMapping(protocol string, externalPort, internalPort int, description string, timeout int) (mappedExternalPort int, err error)
//删除以前添加的从外部端口到的端口映射
//内部端口。
	DeletePortMapping(protocol string, externalPort, internalPort int) (err error)
}

type upnpNAT struct {
	serviceURL string
	ourIP      string
}

//Discover在本地网络中搜索返回NAT的UPNP路由器
//对于网络，如果是，则为零。
func Discover() (nat NAT, err error) {
	ssdp, err := net.ResolveUDPAddr("udp4", "239.255.255.250:1900")
	if err != nil {
		return
	}
	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		return
	}
	socket := conn.(*net.UDPConn)
	defer socket.Close()

	err = socket.SetDeadline(time.Now().Add(3 * time.Second))
	if err != nil {
		return
	}

	st := "ST: urn:schemas-upnp-org:device:InternetGatewayDevice:1\r\n"
	buf := bytes.NewBufferString(
		"M-SEARCH * HTTP/1.1\r\n" +
			"HOST: 239.255.255.250:1900\r\n" +
			st +
			"MAN: \"ssdp:discover\"\r\n" +
			"MX: 2\r\n\r\n")
	message := buf.Bytes()
	answerBytes := make([]byte, 1024)
	for i := 0; i < 3; i++ {
		_, err = socket.WriteToUDP(message, ssdp)
		if err != nil {
			return
		}
		var n int
		n, _, err = socket.ReadFromUDP(answerBytes)
		if err != nil {
			continue
//Socket（）
//返回
		}
		answer := string(answerBytes[0:n])
		if !strings.Contains(answer, "\r\n"+st) {
			continue
		}
//HTTP头字段名不区分大小写。
//http://www.w3.org/protocols/rfc2616/rfc2616-sec4.html sec4.2
		locString := "\r\nlocation: "
		locIndex := strings.Index(strings.ToLower(answer), locString)
		if locIndex < 0 {
			continue
		}
		loc := answer[locIndex+len(locString):]
		endIndex := strings.Index(loc, "\r\n")
		if endIndex < 0 {
			continue
		}
		locURL := loc[0:endIndex]
		var serviceURL string
		serviceURL, err = getServiceURL(locURL)
		if err != nil {
			return
		}
		var ourIP string
		ourIP, err = getOurIP()
		if err != nil {
			return
		}
		nat = &upnpNAT{serviceURL: serviceURL, ourIP: ourIP}
		return
	}
	err = errors.New("UPnP port discovery failed")
	return
}

//服务表示UPNP XML描述中的服务类型。
//只有我们关心的部分存在，因此XML可能有更多
//字段比结构中存在的字段多。
type service struct {
	ServiceType string `xml:"serviceType"`
	ControlURL  string `xml:"controlURL"`
}

//devicelist表示upnp xml描述中的devicelist类型。
//只有我们关心的部分存在，因此XML可能有更多
//字段比结构中存在的字段多。
type deviceList struct {
	XMLName xml.Name `xml:"deviceList"`
	Device  []device `xml:"device"`
}

//ServiceList表示UPNP XML描述中的ServiceList类型。
//只有我们关心的部分存在，因此XML可能有更多
//字段比结构中存在的字段多。
type serviceList struct {
	XMLName xml.Name  `xml:"serviceList"`
	Service []service `xml:"service"`
}

//device在upnp xml描述中表示设备类型。
//只有我们关心的部分存在，因此XML可能有更多
//字段比结构中存在的字段多。
type device struct {
	XMLName     xml.Name    `xml:"device"`
	DeviceType  string      `xml:"deviceType"`
	DeviceList  deviceList  `xml:"deviceList"`
	ServiceList serviceList `xml:"serviceList"`
}

//specVersion表示UPNP XML描述中的specVersion。
//只有我们关心的部分存在，因此XML可能有更多
//字段比结构中存在的字段多。
type specVersion struct {
	XMLName xml.Name `xml:"specVersion"`
	Major   int      `xml:"major"`
	Minor   int      `xml:"minor"`
}

//根表示UPNP XML描述的根文档。
//只有我们关心的部分存在，因此XML可能有更多
//字段比结构中存在的字段多。
type root struct {
	XMLName     xml.Name `xml:"root"`
	SpecVersion specVersion
	Device      device
}

//getchilddevice用给定的
//类型。
func getChildDevice(d *device, deviceType string) *device {
	for i := range d.DeviceList.Device {
		if d.DeviceList.Device[i].DeviceType == deviceType {
			return &d.DeviceList.Device[i]
		}
	}
	return nil
}

//getchilddevice使用
//给定类型。
func getChildService(d *device, serviceType string) *service {
	for i := range d.ServiceList.Service {
		if d.ServiceList.Service[i].ServiceType == serviceType {
			return &d.ServiceList.Service[i]
		}
	}
	return nil
}

//getourip返回对本地IP的最佳猜测。
func getOurIP() (ip string, err error) {
	hostname, err := os.Hostname()
	if err != nil {
		return
	}
	return net.LookupCNAME(hostname)
}

//GetServiceURL解析给定根URL处的XML描述以查找
//用于端口转发的WanipConnection服务的URL。
func getServiceURL(rootURL string) (url string, err error) {
	r, err := http.Get(rootURL)
	if err != nil {
		return
	}
	defer r.Body.Close()
	if r.StatusCode >= 400 {
		err = errors.New(string(r.StatusCode))
		return
	}
	var root root
	err = xml.NewDecoder(r.Body).Decode(&root)
	if err != nil {
		return
	}
	a := &root.Device
	if a.DeviceType != "urn:schemas-upnp-org:device:InternetGatewayDevice:1" {
		err = errors.New("no InternetGatewayDevice")
		return
	}
	b := getChildDevice(a, "urn:schemas-upnp-org:device:WANDevice:1")
	if b == nil {
		err = errors.New("no WANDevice")
		return
	}
	c := getChildDevice(b, "urn:schemas-upnp-org:device:WANConnectionDevice:1")
	if c == nil {
		err = errors.New("no WANConnectionDevice")
		return
	}
	d := getChildService(c, "urn:schemas-upnp-org:service:WANIPConnection:1")
	if d == nil {
		err = errors.New("no WANIPConnection")
		return
	}
	url = combineURL(rootURL, d.ControlURL)
	return
}

//CombineURL将子URL附加到rootURL。
func combineURL(rootURL, subURL string) string {
protocolEnd := "://“
	protoEndIndex := strings.Index(rootURL, protocolEnd)
	a := rootURL[protoEndIndex+len(protocolEnd):]
	rootIndex := strings.Index(a, "/")
	return rootURL[0:protoEndIndex+len(protocolEnd)+rootIndex] + subURL
}

//soap body表示SOAP回复中的<s:body>元素。
//我们不关心的领域被忽略了。
type soapBody struct {
	XMLName xml.Name `xml:"Body"`
	Data    []byte   `xml:",innerxml"`
}

//soap envelope表示SOAP回复中的<s:envelope>元素。
//我们不关心的领域被忽略了。
type soapEnvelope struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    soapBody `xml:"Body"`
}

//SoapRequests使用给定的参数执行SOAP请求并返回
//XML的响应去掉了SOAP头。如果请求是
//不成功返回错误。
func soapRequest(url, function, message string) (replyXML []byte, err error) {
	fullMessage := "<?xml version=\"1.0\" ?>" +
"<s:Envelope xmlns:s=\"http://schemas.xmlsoap.org/soap/envelope/\”s:encodingstyle=\”http://schemas.xmlsoap.org/soap/encoding/\”>\r\n“+
		"<s:Body>" + message + "</s:Body></s:Envelope>"

	req, err := http.NewRequest("POST", url, strings.NewReader(fullMessage))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/xml ; charset=\"utf-8\"")
	req.Header.Set("User-Agent", "Darwin/10.0.0, UPnP/1.0, MiniUPnPc/1.3")
//req.header.set（“传输编码”，“分块”）。
	req.Header.Set("SOAPAction", "\"urn:schemas-upnp-org:service:WANIPConnection:1#"+function+"\"")
	req.Header.Set("Connection", "Close")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")

	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if r.Body != nil {
		defer r.Body.Close()
	}

	if r.StatusCode >= 400 {
//log.stderr（函数，r.statuscode）
		err = errors.New("Error " + strconv.Itoa(r.StatusCode) + " for " + function)
		r = nil
		return
	}
	var reply soapEnvelope
	err = xml.NewDecoder(r.Body).Decode(&reply)
	if err != nil {
		return nil, err
	}
	return reply.Body.Data, nil
}

//GetExternalIPAddressResponse表示对
//GetExternalIPAddress SOAP请求。
type getExternalIPAddressResponse struct {
	XMLName           xml.Name `xml:"GetExternalIPAddressResponse"`
	ExternalIPAddress string   `xml:"NewExternalIPAddress"`
}

//GetExternalAddress通过获取外部IP来实现NAT接口
//来自UPNP路由器。
func (n *upnpNAT) GetExternalAddress() (addr net.IP, err error) {
	message := "<u:GetExternalIPAddress xmlns:u=\"urn:schemas-upnp-org:service:WANIPConnection:1\"/>\r\n"
	response, err := soapRequest(n.serviceURL, "GetExternalIPAddress", message)
	if err != nil {
		return nil, err
	}

	var reply getExternalIPAddressResponse
	err = xml.Unmarshal(response, &reply)
	if err != nil {
		return nil, err
	}

	addr = net.ParseIP(reply.ExternalIPAddress)
	if addr == nil {
		return nil, errors.New("unable to parse ip address")
	}
	return addr, nil
}

//addportmapping通过设置端口转发来实现NAT接口
//从UPNP路由器到具有给定端口和协议的本地计算机。
func (n *upnpNAT) AddPortMapping(protocol string, externalPort, internalPort int, description string, timeout int) (mappedExternalPort int, err error) {
//单个串联将中断ARM编译。
	message := "<u:AddPortMapping xmlns:u=\"urn:schemas-upnp-org:service:WANIPConnection:1\">\r\n" +
		"<NewRemoteHost></NewRemoteHost><NewExternalPort>" + strconv.Itoa(externalPort)
	message += "</NewExternalPort><NewProtocol>" + strings.ToUpper(protocol) + "</NewProtocol>"
	message += "<NewInternalPort>" + strconv.Itoa(internalPort) + "</NewInternalPort>" +
		"<NewInternalClient>" + n.ourIP + "</NewInternalClient>" +
		"<NewEnabled>1</NewEnabled><NewPortMappingDescription>"
	message += description +
		"</NewPortMappingDescription><NewLeaseDuration>" + strconv.Itoa(timeout) +
		"</NewLeaseDuration></u:AddPortMapping>"

	response, err := soapRequest(n.serviceURL, "AddPortMapping", message)
	if err != nil {
		return
	}

//TODO:检查响应以查看端口是否已转发
//如果端口不是通配符，我们将无法得到端口在中的答复。
//它。还不确定通配符。MiniupNPC只是检查错误
//代码在这里。
	mappedExternalPort = externalPort
	_ = response
	return
}

//DeletePortMapping通过删除端口转发来实现NAT接口
//从UPNP路由器到具有给定端口和的本地计算机。
func (n *upnpNAT) DeletePortMapping(protocol string, externalPort, internalPort int) (err error) {

	message := "<u:DeletePortMapping xmlns:u=\"urn:schemas-upnp-org:service:WANIPConnection:1\">\r\n" +
		"<NewRemoteHost></NewRemoteHost><NewExternalPort>" + strconv.Itoa(externalPort) +
		"</NewExternalPort><NewProtocol>" + strings.ToUpper(protocol) + "</NewProtocol>" +
		"</u:DeletePortMapping>"

	response, err := soapRequest(n.serviceURL, "DeletePortMapping", message)
	if err != nil {
		return
	}

//TODO:检查响应以查看端口是否已被删除
//log.println（消息，响应）
	_ = response
	return
}
