package interceptor

import (
	"context"
	"fmt"
	"gateway/middleware/flowcount"
	"google.golang.org/grpc"
	"log"
	"time"
)

// GrpcFlowCountUnaryInterceptor 流量统计
// 一元RPC拦截器
func GrpcFlowCountUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	counter, _ := flowcount.NewFlowCountService("local_app", time.Second)
	counter.Increase()
	fmt.Println("QPS:", counter.QPS)
	fmt.Println("TotalCount:", counter.TotalCount)
	m, err := handler(ctx, req)
	if err != nil {
		log.Printf("RPC failed with error %v\n", err)
	}
	return m, err
}

//流量统计
func GrpcFlowCountStreamInterceptor(counter *flowcount.FlowCountService) func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		counter.Increase()
		fmt.Println("Grpc Stream QPS:", counter.QPS)
		fmt.Println("Grpc Stream TotalCount:", counter.TotalCount)
		err := handler(srv, newWrappedStream(ss))
		if err != nil {
			log.Printf("RPC failed with error %v\n", err)
		}
		return err
	}
}
