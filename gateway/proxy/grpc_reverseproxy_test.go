package proxy

import (
	"fmt"
	lb "gateway/loadbalance"
	"gateway/middleware/flowcount"
	"gateway/proxy/grpc_proxy/interceptor"
	"google.golang.org/grpc"
	"log"
	"net"
	"testing"
	"time"
)

// grpc 反向代理整合中间件
func TestGrpcRPWithAdvance(t *testing.T) {
	const port = ":50051"
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	rb := lb.LoadBalanceFactory(lb.LbWeightRoundRobin)
	rb.Add("127.0.0.1:50055", "40")

	counter, _ := flowcount.NewFlowCountService("local_app", time.Second)
	grpcHandler := NewGrpcLoadBalanceHandler(rb)
	s := grpc.NewServer(
		// 流 拦截器链
		grpc.ChainStreamInterceptor(
			interceptor.GrpcAuthStreamInterceptor,
			interceptor.GrpcFlowCountStreamInterceptor(counter)),
		grpc.UnknownServiceHandler(grpcHandler))

	fmt.Printf("server listening at %v\n", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// grpc反向代理整合负载均衡器
func TestGrpcRPWithLoadBalance(t *testing.T) {
	rb := lb.LoadBalanceFactory(lb.LbWeightRoundRobin)
	rb.Add("127.0.0.1:8005", "40")

	grpcHandler := NewGrpcLoadBalanceHandler(rb)
	s := grpc.NewServer(grpc.UnknownServiceHandler(grpcHandler))

	var port = ":8085"
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen:%v", err)
	}
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to server :%v", err)
	}
}
