package main

import (
	"context"
	"fmt"
	"gateway/proxy/grpc_proxy/proto"
	"gateway/proxy/grpc_proxy/public"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"io"
	"log"
	"time"
)

var msg = "this is client data "

func main() {
	conn, err := grpc.Dial("127.0.0.1:8005", grpc.WithTransportCredentials(insecure.NewCredentials()))
	//conn, err := grpc.Dial("192.168.154.136:8020", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Did not connect:%v", err)
		return
	}
	defer conn.Close()

	c := proto.NewEchoClient(conn)

	// 1.调用一元RPC方法
	unaryEchoWithMetadata(c, msg)
	time.Sleep(1 * time.Second)

	// 2.调用服务端流式处理RPC方法
	serverStreamingWithMetadata(c, msg)
	time.Sleep(1 * time.Second)

	// 3.调用客户端流式处理RPC方法
	clientStreamingWithMetadata(c, msg)
	time.Sleep(1 * time.Second)

	// 4.调用双向流式处理RPC方法
	bidirectionalStreamingWithMetadata(c, msg)
	time.Sleep(1 * time.Second)

}

func unaryEchoWithMetadata(c proto.EchoClient, msg string) {
	fmt.Println("---- UnaryEcho Client -----")

	md := metadata.Pairs("timestamp", time.Now().Format(time.StampNano))
	//md.Append("authorization", "token.....")
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	resp, err := c.UnaryEcho(ctx, &proto.EchoRequest{Message: msg},
		grpc.CallContentSubtype(public.Codec().Name()))
	if err != nil {
		log.Fatalf("failed to call UnaryEcho method error:%v", err)
	} else {
		fmt.Printf("response:%v\n", resp.Message)
	}
}

func serverStreamingWithMetadata(c proto.EchoClient, msg string) {
	fmt.Println("---- ServerStreaming Client -----")

	md := metadata.Pairs("timestamp", time.Now().Format(time.StampNano))
	//md.Append("authorization", "token.....")
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	stream, err := c.ServerStreamingEcho(ctx, &proto.EchoRequest{Message: msg},
		grpc.CallContentSubtype(public.Codec().Name()))
	if err != nil {
		log.Fatalf("failed to call ServerStreamingEcho method error:%v", err)
	}

	var rpcError error
	for {
		// err 读取到流末尾，err = io.EOF
		resp, err := stream.Recv()
		if err != nil {
			rpcError = err
			break
		}
		fmt.Printf("response is :%s\n", resp.Message)
	}
	if rpcError != io.EOF {
		log.Fatalf("failed to finish ServerStreaming:%v", rpcError)
	}
}

func clientStreamingWithMetadata(c proto.EchoClient, msg string) {
	fmt.Println("---- ClientStreaming Client -----")

	md := metadata.Pairs("timestamp", time.Now().Format(time.StampNano))
	//md.Append("authorization", "token.....")
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	stream, err := c.ClientStreamingEcho(ctx, grpc.CallContentSubtype(public.Codec().Name()))
	if err != nil {
		log.Fatalf("failed to call ClientStreamingEcho method error:%v", err)
	}

	for i := 0; i < 5; i++ {
		err := stream.Send(&proto.EchoRequest{Message: msg})
		if err != nil {
			log.Fatalf("Failed to send:", err)
		}
	}

	// 获取响应
	resp, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("failed to finish ClientStreaming:%v", err)
	}
	// 处理服务端响应
	fmt.Printf("response:%v\n", resp.Message)
}

func bidirectionalStreamingWithMetadata(c proto.EchoClient, msg string) {
	fmt.Println("---- bidirectionalStreamingWithMetadata Client -----")

	md := metadata.Pairs("timestamp", time.Now().Format(time.StampNano))
	//md.Append("authorization", "token.....")
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	stream, err := c.BidirectionalStreamingEcho(ctx, grpc.CallContentSubtype(public.Codec().Name()))
	if err != nil {
		log.Fatalf("failed to call ClientStreamingEcho method error:%v", err)
	}

	// 新建协程发送消息
	go func() {
		for i := 0; i < 5; i++ {
			err := stream.Send(&proto.EchoRequest{Message: msg})
			if err != nil {
				log.Fatalf("Failed to send:", err)
			}
		}
		//stream.CloseSend()
	}()

	// 获取响应
	var rpcError error
	for {
		// err 读取到流末尾，err = io.EOF
		resp, err := stream.Recv()
		if err != nil {
			rpcError = err
			break
		}
		fmt.Printf("response is :%s\n", resp.Message)
	}
	if rpcError != io.EOF {
		log.Fatalf("failed to finish ServerStreaming:%v", rpcError)
	}
}
