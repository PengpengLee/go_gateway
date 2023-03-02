package main

import (
	"fmt"
	"io"
	"net/http"
)

// HTTP 正向代理：客户端代理
//
// 实现步骤：
//	0. 启动一个真实服务器: gateway/base/http/http_server.go
//	1. 代理服务器接收客户端请求，复制，封装成新请求
//	2. 发送新请求到真实服务器，接收响应
//	3. 处理响应并返回客户端
//
// 配置本地计算机Internet代理：
// 	Windows：设置-网络和Internet-代理-手动设置代理（打开开关）-输入IP和端口号
// 	Mac/Linux：设置-网络-网络代理-手动-web代理（HTTP代理）-输入IP和端口号
//		命令行：vim /etc/profile 或者 vim ~/.bashrc
//		添加：# add proxy for network
//		export http_proxy="ip:port" # 你的代理服务器ip和端口
//		source /etc/profile 或者 source ~/.bashrc
func main() {
	fmt.Println("正向代理服务器启动 :8080")
	http.Handle("/", &Pxy{})
	http.ListenAndServe("127.0.0.1:8080", nil)
}

// Pxy 定义一个类型，实现 Handler interface
type Pxy struct{}

// ServeHTTP 具体实现方法
func (p *Pxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	fmt.Printf("Received request %s %s %s:\n", req.Method, req.Host, req.RemoteAddr)
	// 1. 代理服务器接收客户端请求，复制，封装成新请求
	outReq := &http.Request{}
	*outReq = *req
	// 2. 发送新请求到下游真实服务器，接收响应
	transport := http.DefaultTransport
	res, err := transport.RoundTrip(outReq)
	if err != nil {
		rw.WriteHeader(http.StatusBadGateway)
		return
	}
	// 3. 处理响应并返回上游客户端
	for key, value := range res.Header {
		for _, v := range value {
			rw.Header().Add(key, v)
		}
	}
	rw.WriteHeader(res.StatusCode)
	io.Copy(rw, res.Body)
	res.Body.Close()
}
