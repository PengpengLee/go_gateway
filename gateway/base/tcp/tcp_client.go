package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

// TCP 客户端
//
// 步骤：
// 	1.与服务器建立TCP连接
// 	2.接收服务端响应
// 	3.向服务端发消息
// 	4.关闭连接，释放资源
func main() {
	// 1.与服务器建立TCP连接
	conn, err := net.Dial("tcp", "localhost:9090")
	if err != nil {
		fmt.Printf("connect failed. err:%v\n", err)
		return
	}
	// 4.关闭连接，释放资源
	defer conn.Close()
	// 2.接收服务端响应
	go getServerData(conn)
	//clientBuf := make([]byte, 1024)
	//n, _ := conn.Read(clientBuf)
	//fmt.Println("Server said:", string(clientBuf[:n]))
	// 3.向服务端发消息
	//conn.Write([]byte("SayHello grpc_server_client"))
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

// getServerData 接收服务器响应
func getServerData(conn net.Conn) {
	// 客户端缓冲区
	buf := make([]byte, 1024)
	// 循环读取服务器信息,并打印到控制台
	for {
		n, _ := conn.Read(buf) // 读取到数据之前,阻塞
		data := strings.TrimSpace(string(buf[:n]))
		if data != "" {
			fmt.Println("from Server:", data)
		}
	}
}
