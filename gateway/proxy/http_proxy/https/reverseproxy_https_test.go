package https

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"testing"

	"gateway/proxy/http_proxy/https/testdata"
)

func TestRealServer(t *testing.T) {
	rs1 := &RealServer{Addr: "example1.com:8003"}
	rs1.Run()
	//rs2 := &RealServer{Addr: "127.0.0.1:8004"}
	//rs2.Run()

	// 监听关闭信号
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}

func TestReverseProxyHttps(t *testing.T) {
	// 下游真实服务器地址
	rs1 := "https://example1.com:8003/?a=1&b=2#container"
	url1, err1 := url.Parse(rs1)
	if err1 != nil {
		log.Println(err1)
	}
	urls := []*url.URL{url1}
	proxy := NewMultipleHostsReverseProxy(urls)

	var addr = "example1.com:8083"
	log.Println("Starting https server at " + addr)
	log.Fatal(http.ListenAndServeTLS(
		addr, testdata.Path(TlsServerCrt), testdata.Path(TlsServerKey), proxy))
}
