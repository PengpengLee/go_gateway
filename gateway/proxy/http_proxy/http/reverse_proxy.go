package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"net/url"
)

// HTTP反向代理基本功能
//
// 请求流程:
//	客户端 -> 代理服务器(:8080) -> 下游真实服务器(:8001)
// 响应流程:
//	下游真实服务器 -> 代理服务器 -> 客户端
//
// 实现步骤:
//	0. 启动一个真实服务器: gateway/proxy/http_proxy/downstream_real_server.go
// 	1.代理接收客户端请求，更改请求结构体信息
// 	2.通过一定的负载均衡算法获取下游服务地址
// 	3.把请求发送到下游服务器，并获取返回内容
// 	4.对返回内容做一些处理，然后返回给客户端
func main() {
	var port = "8080" // 当前代理服务器端口
	http.HandleFunc("/", handler)
	fmt.Println("反向代理服务器启动: " + port)
	http.ListenAndServe(":"+port, nil)
}

var (
	proxyAddr = "http://127.0.0.1:8001?a&=1#af"
)

func handler(w http.ResponseWriter, r *http.Request) {
	// 1.解析下游服务器地址，更改请求地址
	// 被代理的下游真实服务器地址，应该通过一定的负载均衡算法获取
	realServer, _ := url.Parse(proxyAddr) // http://127.0.0.1:8001
	r.URL.Scheme = realServer.Scheme      // http
	r.URL.Host = realServer.Host          // 127.0.0.1:8001
	// 2.请求下游(真实服务器)，并获取返回内容
	transport := http.DefaultTransport
	resp, err := transport.RoundTrip(r) // 得到下游服务器的响应
	defer resp.Body.Close()
	if err != nil {
		log.Println(err)
		return
	}
	// 3.把下游请求内容做一些处理，然后返回给上游(客户端)
	for k, v := range resp.Header { // 修改上游响应头
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	bufio.NewReader(resp.Body).WriteTo(w) // 将下游响应体写回上游客户端
}
