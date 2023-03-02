package main

import (
	"fmt"
	"gateway/proxy/grpc_proxy/helloworld/greeter_jsonrpc/inters"
	"net/rpc"
	"net/rpc/jsonrpc"
)

// MyClient 客户端
type MyClient struct {
	c *rpc.Client
}

// HelloWorld 实现函数，参照 inters.MyInterface 来实现
func (mc *MyClient) HelloWorld(arg string, reply *string) error {
	return mc.c.Call(inters.HelloServiceMethod, arg, reply)
}

func NewClient(addr string) MyClient {
	// gob
	//conn, err := rpc.Dial("tcp", addr)
	conn, err := jsonrpc.Dial("tcp", addr)
	if err != nil {
		fmt.Println("Dial err:", err)
	}
	return MyClient{c: conn}
}

// 1.用 rpc 链接服务器
// 2.调用远程函数
func main() {
	// 1.用 rpc 链接服务器 --Dial()
	// {"method":"Hello.HelloWorld","params":["小白"],"id":0}
	myClient := NewClient("192.168.154.128:8004")

	// 2.调用远程函数
	var reply string // 接收返回值，传出参数
	err := myClient.HelloWorld("小白", &reply)
	if err != nil {
		fmt.Println("Call err:", err)
		return
	}
	fmt.Println(reply)
}
