package https

import (
	"crypto/tls"
	"crypto/x509"
	"gateway/proxy/http_proxy/https/testdata"
	"golang.org/x/net/http2"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

/*
证书签名生成方式：

// 生成2048位的CA私钥
openssl genrsa -out d:/dev/ca.key 2048
// 生成CA公钥：信息可以随便填
openssl req -new -key d:/dev/ca.key -out d:/dev/ca.csr
// 生成CA证书
openssl x509 -req -in d:/dev/ca.csr -extensions v3_ca -signkey d:/dev/ca.key -out d:/dev/ca.crt

// 生成2048位的服务器私钥
openssl genrsa -out d:/dev/server.key 2048
// 生成服务器公钥，域名：example1.com（需要配置本地hosts）
openssl req -new -key d:/dev/server.key -subj "/CN=example1.com" -config d:/dev/openssl.cnf -out d:/dev/pub.csr
// 使用CA证书对服务器公钥进行签名，生成服务器证书
// 配置开启扩展SAN(Subject Alternative Name)的证书，追加配置：
// go_gateway/proxy/http_proxy/https/testdata/openssl.cnf: line 166-173
// 增加参数：-extensions req_ext -extfile d:/dev/openssl.cnf
openssl x509 -days 3650 -req -in d:/dev/pub.csr -extensions v3_req -CAkey d:/dev/ca.key -CA d:/dev/ca.crt -CAcreateserial -extensions req_ext -extfile d:/dev/openssl.cnf -out d:/dev/server.crt

// 客户端安装 CA证书，则浏览器不出现安全警告：ca.crt
// 服务端配置私钥和证书：server.key  server.crt
*/

const (
	TlsCa        = "ca.crt"
	TlsServerCrt = "server.crt"
	TlsServerKey = "server.key"
)

var transport = &http.Transport{
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second, //连接超时
		KeepAlive: 30 * time.Second, //长连接超时时间
	}).DialContext,
	// 下面这行代码是跳过证书验证
	//TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	// 不跳过验证，使用证书访问
	TLSClientConfig: func() *tls.Config {
		pool := x509.NewCertPool()
		caCertPath := testdata.Path(TlsCa)
		caCrt, _ := ioutil.ReadFile(caCertPath)
		pool.AppendCertsFromPEM(caCrt)
		return &tls.Config{RootCAs: pool}
	}(),
	MaxIdleConns:          100,              //最大空闲连接
	IdleConnTimeout:       90 * time.Second, //空闲超时时间
	TLSHandshakeTimeout:   10 * time.Second, //tls握手超时时间
	ExpectContinueTimeout: 1 * time.Second,  //100-continue 超时时间
}

func NewMultipleHostsReverseProxy(targets []*url.URL) *httputil.ReverseProxy {
	// 请求协调者
	director := func(req *http.Request) {
		targetIndex := rand.Intn(len(targets))
		target := targets[targetIndex]
		targetQuery := target.RawQuery
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		if _, ok := req.Header["User-Agent"]; !ok {
			req.Header.Set("User-Agent", "user-agent")
		}
	}
	// 支持 HTTP2
	http2.ConfigureTransport(transport)
	return &httputil.ReverseProxy{Director: director, Transport: transport}
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
