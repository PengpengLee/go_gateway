package proxy

import (
	"context"
	"gateway/loadbalance"
	"gateway/proxy/grpc_proxy"
	"gateway/proxy/grpc_proxy/public"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
)

func NewGrpcLoadBalanceHandler(lb loadbalance.LoadBalance) grpc.StreamHandler {
	return func() grpc.StreamHandler {
		// 定义入口函数：实用负载均衡算法获取下游主机地址
		director := func(ctx context.Context, fullMethodName string) (context.Context, *grpc.ClientConn, error) {
			nextAddr, err := lb.Get(fullMethodName)
			if err != nil {
				log.Fatal("get next address fail")
			}
			c, err := grpc.DialContext(ctx, nextAddr,
				// 自定义编码
				grpc.WithDefaultCallOptions(grpc.CallContentSubtype(public.Codec().Name())),
				// 禁用安全传输
				grpc.WithTransportCredentials(insecure.NewCredentials()))
			return ctx, c, err
		}

		return grpc_proxy.TransparentHandler(director)
	}()
}
