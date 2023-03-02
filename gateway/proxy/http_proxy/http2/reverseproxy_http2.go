package https

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
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
		fmt.Println("target.Scheme")
		fmt.Println(target.Scheme)
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
