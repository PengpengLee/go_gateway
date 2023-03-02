package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// HTTP反向代理简单版：用ReverseProxy实现
func main() {
	// 下游真实服务器地址
	realServer := "http://127.0.0.1:8001?a=1&b=2#container"
	// parse解析url
	// 从"http://127.0.0.1:8001?a=1&b=2#container"
	// 到"http://127.0.0.1:8001"
	serverURL, err := url.Parse(realServer)
	if err != nil {
		log.Println(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(serverURL)
	// 代理服务器地址：8081
	var addr = "127.0.0.1:8081"
	log.Println("Starting proxy http grpc_server_client at:" + addr)
	http.ListenAndServe(addr, proxy)
}
