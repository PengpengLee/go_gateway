package main

import (
	"context"
	"fmt"
	"gateway/proxy/tcp_proxy/proxy"
	"gateway/proxy/tcp_proxy/server"
	"net"
	"os"
	"os/signal"
	"syscall"
)

// TCP 服务器，实现服务与代理分离
//
// 创建一个TCP服务器：
// 	1.监听端口
// 	2.获取连接
// 	3.封装新连接对象，设置服务参数
//		上下文、超时、连接关闭
// 	4.回调handler（需要定义一个接口）
func main() {
	// 启动TCP服务器
	go func() {
		var addr = "192.168.110.11:8003"
		// 1.创建TCPServer实例
		tcpServer := &server.TCPServer{
			Addr:    addr,
			Handler: &handler{},
		}
		// 2.启动监听提供服务
		fmt.Println("Starting TCP grpc_server_client at " + addr)
		tcpServer.ListenAndServe()
	}()

	// 启动TCP代理服务器
	go func() {
		var tcpServerAddr = "192.168.110.11:8003"
		// 1.创建TCP代理实例
		tcpProxy := proxy.NewSingleHostReverseProxy(tcpServerAddr)
		// 2.启动监听提供服务
		var tcpProxyAddr = "192.168.110.11:8083"
		fmt.Println("Starting TCP grpc_server_client at " + tcpProxyAddr)
		server.ListenAndServe(tcpProxyAddr, tcpProxy)
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}

type handler struct {
}

func (t *handler) ServeTCP(ctx context.Context, conn net.Conn) {
	conn.Write([]byte("haha\n"))
}
