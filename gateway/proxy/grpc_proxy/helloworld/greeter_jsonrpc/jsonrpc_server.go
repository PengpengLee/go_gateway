package main

import (
	"errors"
	"fmt"
	"gateway/proxy/grpc_proxy/helloworld/greeter_jsonrpc/inters"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
)

type HelloWorld struct {
}

func (hw *HelloWorld) HelloWorld(req string, resp *string) error {
	*resp = req + " 你好！"
	return nil
}

func RegisterService(handler inters.MyInterface) error {
	err := rpc.RegisterName(inters.HelloServiceName, handler)
	if err != nil {
		return errors.New("注册 rpc 服务失败！" + err.Error())
	}
	return nil
}

// rpc：像调用本地函数一样调用远程函数
//
// 1.注册 rpc 服务。给对象绑定方法
// 	指定服务名称 和 服务接收者（处理器）
// 2.创建监听器
//	listen("tcp", "addr")
// 3.建立连接
// 4.将连接绑定 rpc 服务
// 	ServeConn(conn)
func main() {
	// 1. 注册RPC服务, 绑定对象方法
	// 服务名称：hello，处理器：HelloWorld.HelloWorld
	err := RegisterService(&HelloWorld{})
	if err != nil {
		fmt.Println("注册 rpc 服务失败！", err)
		return
	}
	// 2. 设置监听
	listener, err := net.Listen("tcp", "192.168.110.11:8004")
	if err != nil {
		fmt.Println("net.Listen err:", err)
		return
	}
	fmt.Println("listening port:127.0.0.1:8004")
	// 3. 建立连接
	conn, err := listener.Accept()
	if err != nil {
		fmt.Println("Accept err:", err)
		return
	}
	defer conn.Close()
	fmt.Println("connection accepted....")
	// 4. 绑定服务
	// gob 序列化
	//rpc.ServeConn(conn)
	jsonrpc.ServeConn(conn)
}
