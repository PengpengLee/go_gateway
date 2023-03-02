package main

import (
	"context"
	"fmt"
	"gateway/proxy/grpc_proxy/helloworld/greeter_grpc/pb"
	"google.golang.org/grpc"
	"net"
)

type HelloService struct {
}

func (hs *HelloService) Hello(ctx context.Context, person *pb.Person) (*pb.Person, error) {
	reply := &pb.Person{
		Name: "张三" + person.Name,
		Age:  20,
	}
	return reply, nil

	/*if err != nil {
		return pb.Hello(ctx, person)
	}*/
}

func main() {
	// 1.注册gRPC服务
	grpcServer := grpc.NewServer()
	pb.RegisterHelloServiceServer(grpcServer, &HelloService{})

	listener, err := net.Listen("tcp", "127.0.0.1:8004")
	if err != nil {
		fmt.Println("Listen err:", err)
		return
	}
	defer listener.Close()

	// 2.启动服务
	grpcServer.Serve(listener)
}
