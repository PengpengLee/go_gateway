package proxy

import (
	"context"
	"fmt"
	lb "gateway/loadbalance"
	"log"
	"net/http"
	"testing"
)

// 以观察者模式构建反向代理
// http反向代理支持负载均衡、服务发现
//
// 数据流转过程：
// 	1.下游服务器启动时，注册服务信息到zk
// 	2.zk节点配置信息变更，通知所有观察者-负载均衡器
// 	3.负载均衡器根据更新后的可用服务列表，得出服务地址并提供反向代理服务
// 	4.客户端通过反向代理访问可用下游服务器主机，获得响应
//
// 实现步骤：
// 	1.定义观察主体：连接zk服务器，定义被监听节点，启动监听
// 	2.定义观察者：绑定观察主体
// 		初始化负载均衡器，实现了观察者接口 lb.Observer
// 	3.基于负载均衡器构建反向代理
// 	4.启动反向代理服务器
func TestHttpLoadBalanceWithConf(t *testing.T) {
	// 	1.定义观察主体：连接zk服务器，定义被监听节点，启动监听
	concreteConf, err := lb.NewLoadBalanceZkConf("http://%s/",
		"/realserver",
		[]string{"192.168.154.132:2181"},
		map[string]string{
			"127.0.0.1:8007": "10",
			"127.0.0.1:8001": "20",
			"127.0.0.1:8002": "40",
		})
	if err != nil {
		fmt.Println("error :", err)
		return
	}

	// 	2.定义观察者：绑定观察主体
	// 	初始化负载均衡器，实现了观察者接口 lb.Observer
	rb := lb.LoadBalanceFactoryWithConf(lb.LbWeightRoundRobin, concreteConf)
	// 	3.基于负载均衡器构建反向代理
	proxy := NewLoadBalanceReverseProxy(context.Background(), rb)
	// 	4.启动反向代理服务器
	var addr = "127.0.0.1:8007"
	log.Println("Starting http server at " + addr)
	log.Fatal(http.ListenAndServe(addr, proxy))
}
