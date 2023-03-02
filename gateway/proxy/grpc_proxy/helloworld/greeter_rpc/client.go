package main

import (
	"fmt"
	"net/rpc"
)

// 1.用 rpc 链接服务器
// 2.调用远程函数
func main() {
	// 1.用 rpc 链接服务器 --Dial()
	conn, err := rpc.Dial("tcp", "192.168.154.128:8004")
	if err != nil {
		fmt.Println("Dial err:", err)
		return
	}
	defer conn.Close()
	// 2.调用远程函数
	var reply string // 接收返回值，传出参数
	err = conn.Call("hello.HelloWorld", "小李", &reply)
	if err != nil {
		fmt.Println("Call err:", err)
		return
	}
	fmt.Println(reply)
}
