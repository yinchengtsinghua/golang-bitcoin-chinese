
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/go-socks/socks"
)

//new http client返回根据
//关联连接配置中的代理和TLS设置。
func newHTTPClient(cfg *config) (*http.Client, error) {
//根据需要配置代理。
	var dial func(network, addr string) (net.Conn, error)
	if cfg.Proxy != "" {
		proxy := &socks.Proxy{
			Addr:     cfg.Proxy,
			Username: cfg.ProxyUser,
			Password: cfg.ProxyPass,
		}
		dial = func(network, addr string) (net.Conn, error) {
			c, err := proxy.Dial(network, addr)
			if err != nil {
				return nil, err
			}
			return c, nil
		}
	}

//根据需要配置TLS。
	var tlsConfig *tls.Config
	if !cfg.NoTLS && cfg.RPCCert != "" {
		pem, err := ioutil.ReadFile(cfg.RPCCert)
		if err != nil {
			return nil, err
		}

		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(pem)
		tlsConfig = &tls.Config{
			RootCAs:            pool,
			InsecureSkipVerify: cfg.TLSSkipVerify,
		}
	}

//创建并返回可能配置了
//代理和TLS。
	client := http.Client{
		Transport: &http.Transport{
			Dial:            dial,
			TLSClientConfig: tlsConfig,
		},
	}
	return &client, nil
}

//sendpostrequest使用http-post模式发送已封送的json-rpc命令
//到传递的配置结构中描述的服务器。它还试图
//unmarshal the response as a JSON-RPC response and returns either the result
//字段或错误字段，取决于是否存在错误。
func sendPostRequest(marshalledJSON []byte, cfg *config) ([]byte, error) {
//向配置的RPC服务器生成请求。
	protocol := "http"
	if !cfg.NoTLS {
		protocol = "https"
	}
url := protocol + "://“+cfg.rpcs服务器
	bodyReader := bytes.NewReader(marshalledJSON)
	httpRequest, err := http.NewRequest("POST", url, bodyReader)
	if err != nil {
		return nil, err
	}
	httpRequest.Close = true
	httpRequest.Header.Set("Content-Type", "application/json")

//配置基本访问授权。
	httpRequest.SetBasicAuth(cfg.RPCUser, cfg.RPCPassword)

//创建根据用户配置的新HTTP客户端-
//指定选项并提交请求。
	httpClient, err := newHTTPClient(cfg)
	if err != nil {
		return nil, err
	}
	httpResponse, err := httpClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}

//读取原始字节并关闭响应。
	respBytes, err := ioutil.ReadAll(httpResponse.Body)
	httpResponse.Body.Close()
	if err != nil {
		err = fmt.Errorf("error reading json reply: %v", err)
		return nil, err
	}

//处理不成功的HTTP响应
	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
//如果服务器主体是
//空的。这种情况不应该经常发生，但最好是
//如果目标服务器性能不佳，则不会显示任何内容。
//实施。
		if len(respBytes) == 0 {
			return nil, fmt.Errorf("%d %s", httpResponse.StatusCode,
				http.StatusText(httpResponse.StatusCode))
		}
		return nil, fmt.Errorf("%s", respBytes)
	}

//取消标记响应。
	var resp btcjson.Response
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, resp.Error
	}
	return resp.Result, nil
}
