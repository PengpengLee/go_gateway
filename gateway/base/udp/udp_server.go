package main

import (
	"fmt"
	"net"
)

// UDP 服务器
//
// 步骤：
// 	1.监听服务器指定端口
// 	2.读取客户端数据
// 	3.处理请求并响应
func main() {
	// 1.监听服务器指定端口
	// network: 联网方式, 必须的UDP协议,注意小写
	// laddr: 服务器地址,默认本机地址
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP: net.IPv4(127, 0, 0, 1),
		// 端口号, 如果不指定,则由操作系统随机分配
		Port: 8080,
	})
	if err != nil {
		fmt.Printf("conn failed, err:%v\n", err)
	}
	fmt.Println("UDP Server is started!")
	// 2.读取客户端数据
	var data [1024]byte
	n, clientAddr, err := conn.ReadFromUDP(data[:]) // 阻塞
	if err != nil {
		fmt.Printf("read error, clientAddr: %v, err: %v\n", clientAddr, err)
	}
	fmt.Printf("clientAddr: %v data:%v count: %v\n", clientAddr, string(data[:n]), n)
	// 3.处理请求并响应
	// TODO sth else
	_, err = conn.WriteToUDP([]byte("received success!"), clientAddr)
	if err != nil {
		fmt.Printf("write failed, err:%v \n", err)
	}
}
