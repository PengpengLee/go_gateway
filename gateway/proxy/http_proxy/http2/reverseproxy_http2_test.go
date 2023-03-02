package https

import (
	"golang.org/x/net/http2"
	"log"
	"net/http"
	"net/url"
	"testing"
	"time"

	"gateway/proxy/http_proxy/https/testdata"
)

func TestReverseProxyHttps(t *testing.T) {
	// 下游真实服务器地址
	rs1 := "https://127.0.0.1:8003/?a=1&b=2#container"
	url1, err1 := url.Parse(rs1)
	if err1 != nil {
		log.Println(err1)
	}
	urls := []*url.URL{url1}
	proxy := NewMultipleHostsReverseProxy(urls)

	var addr = "127.0.0.1:8083"
	log.Println("Starting https server at " + addr)

	mux := http.NewServeMux()
	mux.Handle("/", proxy)
	server := &http.Server{
		Addr:         addr,
		WriteTimeout: time.Second * 3, //设置3秒的写超时
		Handler:      mux,
	}

	// 支持HTTP2：ConfigureServer 配置服务器
	// s *http.Server, 自定义服务实例
	// conf *Server, 服务器配置，除 s 定义的内容外，其它属性走默认
	http2.ConfigureServer(server, &http2.Server{})
	// ListenAndServeTLS函数，支持HTTPS
	log.Fatal(server.ListenAndServeTLS(testdata.Path(TlsServerCrt), testdata.Path(TlsServerKey)))
	log.Fatal(server.ListenAndServe())
}
