package proxy

import (
	"context"
	"fmt"
	lb "gateway/loadbalance"
	"gateway/proxy/tcp_proxy/proxy"
	tcp "gateway/proxy/tcp_proxy/server"
	"testing"
)

// TestTcpLoadBalanceReverseProxy 测试TCP负载均衡代理服务器
// 定义 TCPReverseProxy 入口函数 director，执行指定的负载均衡算法，返回 TCP 服务器地址
//
// 实现步骤：
// 1.选择合适的负载均衡器
// 2.创建一个支持负载均衡算法的 handler
// 3.启动 TCP 代理服务
//
// 测试命令：
// 	telnet 192.168.0.107 8088
func TestTcpLoadBalanceReverseProxy(t *testing.T) {
	// 1.选择合适的负载均衡器
	rb := lb.LoadBalanceFactory(lb.LbWeightRoundRobin)
	rb.Add("192.168.0.107:8003", "5")
	rb.Add("192.168.0.107:8004", "3")

	// 2.创建一个支持负载均衡算法的 handler
	// ctx：可用 tcp.TcpSliceRouteContext 替换
	proxy := NewTcpLoadBalanceReverseProxy(context.Background(), rb)

	// 3.启动 TCP 代理服务
	var addr = "192.168.0.107:8088"
	tcpServ := tcp.TCPServer{Addr: addr, Handler: proxy}
	fmt.Println("tcp_proxy start at :", addr)
	tcpServ.ListenAndServe()
}

func TestTcpLoadBalanceWithRedis(t *testing.T) {
	// redis服务器测试
	rb := lb.LoadBalanceFactory(lb.LbWeightRoundRobin)
	rb.Add("127.0.0.1:6379", "40")

	var addr = "192.168.0.107:8002"
	proxy := proxy.NewTcpLoadBalanceReverseProxy(context.Background(), rb)
	tcpServ := tcp.TCPServer{Addr: addr, Handler: proxy}
	fmt.Println("Starting tcp_proxy at " + addr)
	tcpServ.ListenAndServe()
}
