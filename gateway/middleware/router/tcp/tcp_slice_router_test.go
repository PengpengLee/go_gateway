package tcp

import (
	"fmt"
	lb "gateway/loadbalance"
	"gateway/middleware/flowcount"
	"gateway/middleware/whitelist"
	"gateway/proxy"
	tcp_proxy "gateway/proxy/tcp_proxy/server"
	"testing"
	"time"
)

// TestTcpSliceRouter 测试基于方法数组构建的中间件
func TestTcpSliceRouter(t *testing.T) {
	rb := lb.LoadBalanceFactory(lb.LbWeightRoundRobin)
	rb.Add("192.168.0.107:8003", "40")
	//rb.Add("192.168.0.107:8004", "30")

	// 构建路由及设置中间件
	counter, _ := flowcount.NewFlowCountService("local_app", time.Second)
	router := NewTcpSliceRouter()
	router.Group("/").Use(whitelist.IpWhiteListMiddleWare(), flowcount.FlowCountMiddleWare(counter))

	// 构建回调handler
	routerHandler := NewTcpSliceRouterHandler(func(c *TcpSliceRouteContext) tcp_proxy.TCPHandler {
		return proxy.NewTcpLoadBalanceReverseProxy(c.Ctx, rb)
	}, router)

	// 启动服务
	var addr = "192.168.0.107:8006"
	tcpServ := tcp_proxy.TCPServer{Addr: addr, Handler: routerHandler}
	fmt.Println("Starting tcp_proxy at " + addr)
	tcpServ.ListenAndServe()
}
