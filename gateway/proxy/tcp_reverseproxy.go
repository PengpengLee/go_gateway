package proxy

import (
	"context"
	"gateway/loadbalance"
	tcp_proxy "gateway/proxy/tcp_proxy/proxy"
	"log"
	"time"
)

// NewTcpLoadBalanceReverseProxy 创建支持负载均衡的 TCP 代理实例
// 实现步骤：
// 1.创建一个 TCPReverseProxy 实例
// 2.封装入口函数 Director：根据负载均衡算法获取 TCP 服务器地址
// 3.返回 TCPReverseProxy 实例
func NewTcpLoadBalanceReverseProxy(c context.Context, lb loadbalance.LoadBalance) *tcp_proxy.TCPReverseProxy {
	pxy := &tcp_proxy.TCPReverseProxy{
		Ctx:             c,
		Deadline:        time.Minute,
		DialTimeout:     10 * time.Second,
		KeepAlivePeriod: time.Hour,
	}
	// 定义入口函数，根据负载均衡算法获取 TCP 服务器地址
	pxy.Director = func(remoteAddr string) (nextAddr string, err error) {
		nextAddr, err = lb.Get(remoteAddr)
		if err != nil {
			log.Fatal("get next address fail")
		}
		pxy.Addr = nextAddr
		return
	}
	return pxy
}
