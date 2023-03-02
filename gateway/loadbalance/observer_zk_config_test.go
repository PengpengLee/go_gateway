package loadbalance

import (
	"fmt"
	"testing"
)

// 以观察者模式构建负载均衡配置
func TestNewLoadBalanceObserver(t *testing.T) {
	// 1.构建一个支持负载均衡的zk节点，启动监听
	concreteConf, err := NewLoadBalanceZkConf("%s",
		"/realserver",
		[]string{"192.168.154.132:2181"},
		map[string]string{"127.0.0.1:8007": "10"})
	if err != nil {
		fmt.Println("error :", err)
		return
	}
	// 2.构建一个观察者
	loadBalanceObserver := NewLoadBalanceObserver(concreteConf)
	// 3.绑定观察者，注册监听
	concreteConf.Attach(loadBalanceObserver)
	// 4.更新配置，通知观察者
	concreteConf.UpdateConf([]string{"192.168.10.31"})

	select {}
}
