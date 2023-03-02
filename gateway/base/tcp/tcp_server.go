package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

// TCP 服务器
//
// 步骤：
// 	1.监听服务器指定端口
// 	2.创建TCP连接
// 	3.处理请求
//	4.对客户端进行响应
// 	5.释放连接
func main() {
	// 1.监听服务器指定端口
	// network: 联网方式,必须是tcp,注意小写
	// address: 服务器地址,默认为本机地址
	listener, err := net.Listen("tcp", "127.0.0.1:9090")
	if err != nil {
		fmt.Println("listen failed. err:", err)
		return
	}
	fmt.Println("监听已启动....")
	// 2.创建TCP连接
	conn, err := listener.Accept() // 连接创建完成之前,阻塞
	// 5.释放连接
	//defer conn.Close()
	if err != nil {
		fmt.Println("accept failed, err:", err)
	}
	// 连接成功, 给客户端一个回复
	conn.Write([]byte("received success!"))

	// 3.处理客户端请求：打印数据到控制台
	go getClientData(conn)

	// 4.对客户端进行响应
	//conn.Write([]byte("thank you !"))
	inputReader := bufio.NewReader(os.Stdin)
	for {
		input, _ := inputReader.ReadString('\n')
		input = strings.TrimSpace(input)
		_, err := conn.Write([]byte(input))
		if err != nil {
			fmt.Println("write failed, err:", err)
			break
		}
	}
}

// 3.处理请求
func getClientData(conn net.Conn) {
	// 服务器缓冲区
	buf := make([]byte, 1024)
	// 循环读取客户端信息,并打印到控制台
	for {
		n, _ := conn.Read(buf) // 读取到数据之前,阻塞
		data := strings.TrimSpace(string(buf[:n]))
		if data != "" {
			fmt.Println("from Client:", data)
		}
	}
}
