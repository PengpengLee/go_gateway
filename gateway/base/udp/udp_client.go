package main

import (
	"fmt"
	"net"
)

// UDP 客户端
//
// 步骤：
// 	1.连接服务器
// 	2.发送数据
// 	3.接收数据
func main() {
	// 1.连接服务器
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{
		// 服务器IP地址
		IP: net.IPv4(127, 0, 0, 1),
		// 服务器监听端口号
		Port: 8080,
	})
	if err != nil {
		fmt.Printf("connection failed. err:%v\n", err)
	}
	// 2.发送数据包
	data := "SayHello UDP Server!"
	_, err = conn.Write([]byte(data))
	if err != nil {
		fmt.Println("err:", err)
	}
	// 3.接收数据包
	result := make([]byte, 1024)
	len, remoteAddr, err := conn.ReadFromUDP(result) // 阻塞
	if err != nil {
		fmt.Println("receive failed. err:", err)
	}
	fmt.Printf("response from grpc_server_client, addr:%v data:%v\n", remoteAddr, string(result[:len]))
}
