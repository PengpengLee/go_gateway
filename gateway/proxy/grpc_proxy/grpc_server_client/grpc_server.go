package main

import (
	"context"
	"flag"
	"fmt"
	"gateway/proxy/grpc_proxy/proto"
	"google.golang.org/grpc"
	"io"
	"log"
	"net"
)

var port = flag.Int("port", 8005, "the port to serve on")

func main() {
	flag.Parse()

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		log.Fatalf("failed lisenting: %v", err)
	}
	s := grpc.NewServer()
	proto.RegisterEchoServer(s, &server{})
	s.Serve(listener)
}

type server struct {
}

// UnaryEcho 一元RPC服务方式实现
// metadata：元数据，map[string]string
// 	token, timestamp, 授权
func (s *server) UnaryEcho(ctx context.Context, req *proto.EchoRequest) (*proto.EchoResponse, error) {
	//fmt.Println("----- UnaryEcho ------")

	//md, ok := metadata.FromIncomingContext(ctx)
	//if !ok {
	//	log.Println("miss metadata from context")
	//}
	//fmt.Println("md :", md)

	return &proto.EchoResponse{Message: req.Message}, nil
}

// ServerStreamingEcho 服务端流式处理RPC方法实现
func (s *server) ServerStreamingEcho(req *proto.EchoRequest, stream proto.Echo_ServerStreamingEchoServer) error {
	fmt.Println("----- ServerStreamingEcho ------")

	for i := 0; i < 5; i++ {
		err := stream.Send(&proto.EchoResponse{Message: req.Message})
		if err != nil {
			return err
		}
	}
	return nil
}

// ClientStreamingEcho 客户端流式处理RPC方法实现
func (s *server) ClientStreamingEcho(stream proto.Echo_ClientStreamingEchoServer) error {
	fmt.Println("----- ClientStreamingEcho ------")

	var message = "received over !"
	for {
		req, err := stream.Recv()
		if err == io.EOF { // 读取到流的末尾
			fmt.Println("echo last received message")
			return stream.SendAndClose(&proto.EchoResponse{Message: message})
		}
		if err != nil {
			// 读取客户端请求消息出错
			return err
		}
		// 根据实际业务需要处理请求消息
		fmt.Println("request received:", req.Message)
	}
}

// BidirectionalStreamingEcho 双向流处理RPC方法实现
func (s *server) BidirectionalStreamingEcho(stream proto.Echo_BidirectionalStreamingEchoServer) error {
	fmt.Println("----- BidirectionalStreamingEcho ------")

	var message = "received over !"
	for {
		req, err := stream.Recv()
		if err == io.EOF { // 读取到流的末尾
			fmt.Println("echo last received message")
			// Acknowledge
			return stream.Send(&proto.EchoResponse{Message: message})
			//return nil
		}
		if err != nil {
			// 读取客户端请求消息出错
			return err
		}
		// 根据实际业务需要处理请求消息
		fmt.Println("request received:", req.Message)
		err = stream.Send(&proto.EchoResponse{Message: "request received:" + req.Message})
		if err != nil {
			return err
		}
	}
}
