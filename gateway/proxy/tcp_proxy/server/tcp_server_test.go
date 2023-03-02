package server

import (
	"log"
	"testing"
)

// TCP 服务器，实现服务与代理分离
//
// 创建一个TCP服务器：
// 	1.监听端口
// 	2.获取连接
// 	3.封装新连接对象，设置服务参数
//		上下文、超时、连接关闭
// 	4.回调handler（需要定义一个接口）
func TestTcpServer(t *testing.T) {
	// 127.0.0.1是本机回环地址，只能在本机测试
	// 如果在别的机器上测试，需要写真实IP
	// 测试命令：telnet 192.168.110.11 8003
	// var addr = "192.168.110.11:8003"
	var addr = "192.168.0.107:8003"
	// 1.创建TCPServer实例
	tcpServer := &TCPServer{
		Addr:    addr,
		Handler: &tcpHandler{},
	}
	// 2.启动监听提供服务
	log.Println("Starting TCP server at " + addr)
	tcpServer.ListenAndServe()
}
