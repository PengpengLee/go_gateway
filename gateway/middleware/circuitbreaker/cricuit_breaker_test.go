package circuitbreaker

import (
	"errors"
	"fmt"
	sr "gateway/middleware/router/http"
	"gateway/proxy"
	"github.com/afex/hystrix-go/hystrix"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"
)

// 测试 hystrix-go 基本使用：熔断、降级、限流集成类库
//
// 使用步骤：
// 	1.配置一个熔断器
// 	2.使用熔断器：业务逻辑与熔断器的整合
func TestHystrix(t *testing.T) {
	// 启动一个流服务器
	// 统计熔断、降级、限流的结果，实时发送到 :8070 服务器上，就可以通过dashboard查看
	hStreamHandler := hystrix.NewStreamHandler()
	hStreamHandler.Start()
	go http.ListenAndServe("192.168.110.13:8070", hStreamHandler)

	// 1.配置一个熔断器
	var hystrixName = "hystrixName"
	hystrix.ConfigureCommand(hystrixName, hystrix.CommandConfig{
		// 单次请求（等待命令完成）的超时时间：ms
		Timeout: 1000,
		// 最大并发量：同一类型的命令可以同时运行的数量（并发量）
		MaxConcurrentRequests: 1,
		// 熔断后超时重试的时间：熔断（打开）后多久（ms）去尝试恢复服务：打开 -> 半打开
		SleepWindow: 5000,
		// 验证熔断的请求数量：熔断发生之前所需的最小请求数，10秒内采样
		RequestVolumeThreshold: 1,
		// 验证熔断的错误百分比：1%。
		// 根据上一个字段，记录其10秒内熔断次数所占比例，是否达到 1%
		ErrorPercentThreshold: 1,
	})

	// 2.使用熔断器：业务逻辑与熔断器的整合
	for i := 0; i < 1000; i++ {
		// 异步调用:  hystrix.Go()
		// 同步调用:  hystrix.Do()
		// 同步方式运行函数，阻塞，直到函数成功或返回错误，包括 hystrix错误
		// fallback：降级方法
		err := hystrix.Do(hystrixName, func() error {
			// 错误测试
			if i == 0 {
				return errors.New("service error" + strconv.Itoa(i))
			}
			log.Println("do services", i)
			return nil
		}, /*nil,*/ func(e error) error {
			fmt.Println("here is Plan B")
			return errors.New("fallback err:" + e.Error())
		})
		if err != nil {
			log.Println("hystrix err:" + err.Error())
			time.Sleep(1 * time.Second)
			log.Println("sleep 1 second", i)
		}
	}
}

// 测试网关集成熔断方案
func TestCircuitBreaker(t *testing.T) {
	coreFunc := func(c *sr.SliceRouteContext) http.Handler {
		rs1 := "http://127.0.0.1:8001/"
		url1, err1 := url.Parse(rs1)
		if err1 != nil {
			log.Println(err1)
		}

		rs2 := "http://127.0.0.1:8002/base"
		url2, err2 := url.Parse(rs2)
		if err2 != nil {
			log.Println(err2)
		}

		urls := []*url.URL{url1, url2}
		return proxy.NewMultipleHostsReverseProxy(c.Ctx, urls)
	}

	var addr = "127.0.0.1:8006"
	log.Println("Starting httpserver at " + addr)

	// 配置熔断器
	var hystrixName = "common"
	ConfCircuitBreaker(hystrixName, 1000, 1, 5000, 1, 1)
	// 注册中间件
	sliceRouter := sr.NewSliceRouter()
	sliceRouter.Group("/").Use(CircuitBreaker(hystrixName, nil))
	routerHandler := sr.NewSliceRouterHandler(coreFunc, sliceRouter)
	log.Fatal(http.ListenAndServe(addr, routerHandler))
}
