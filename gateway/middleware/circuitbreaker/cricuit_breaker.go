package circuitbreaker

import (
	"errors"
	http2 "gateway/middleware/router/http"
	"github.com/afex/hystrix-go/hystrix"
	"log"
	"net"
	"net/http"
)

// ConfCircuitBreaker 配置 hystrix common.
//
// parameter list：
//	name,
// 	Timeout(ms), MaxConcurrentRequests, SleepWindow(ms),
//	RequestVolumeThreshold, ErrorPercentThreshold
func ConfCircuitBreaker(name string, p ...int) {
	hystrix.ConfigureCommand(name, hystrix.CommandConfig{
		Timeout:                p[0], // 单次请求 超时时间
		MaxConcurrentRequests:  p[1], // 最大并发量
		SleepWindow:            p[2], // 熔断后多久去尝试服务是否可用
		RequestVolumeThreshold: p[3], // 验证熔断请求数量, 10秒内采样
		ErrorPercentThreshold:  p[4], // 验证熔断错误百分比
	})
}

func ConfCircuitBreakerWithOpenStream(openStream bool) {
	hystrix.ConfigureCommand("common", hystrix.CommandConfig{
		Timeout:                1000, // 单次请求 超时时间
		MaxConcurrentRequests:  1,    // 最大并发量
		SleepWindow:            5000, // 熔断后多久去尝试服务是否可用
		RequestVolumeThreshold: 1,    // 验证熔断的 请求数量, 10秒内采样
		ErrorPercentThreshold:  1,    // 验证熔断的 错误百分比
	})

	if openStream {
		hystrixStreamHandler := hystrix.NewStreamHandler()
		hystrixStreamHandler.Start()
		go func() {
			err := http.ListenAndServe(net.JoinHostPort("", "2001"), hystrixStreamHandler)
			log.Fatal(err)
		}()
	}
}

func CircuitBreaker(name string, fallback func(e error) error) func(c *http2.SliceRouteContext) {
	return func(c *http2.SliceRouteContext) {
		err := hystrix.Do(name, func() error {
			c.Next()
			statusCode, ok := c.Get("status_code").(int)
			if !ok || statusCode != 200 {
				return errors.New("downstream error")
			}
			return nil
		}, fallback)
		if err != nil {
			// 加入自动降级处理，如获取缓存数据等
			switch err {
			case hystrix.ErrCircuitOpen:
				c.Rw.Write([]byte("circuit error:" + err.Error()))
			case hystrix.ErrMaxConcurrency:
				c.Rw.Write([]byte("circuit error:" + err.Error()))
			default:
				c.Rw.Write([]byte("circuit error:" + err.Error()))
			}
			c.Abort()
		}
	}
}

// OpenStream 在指定的主机上打开流
func OpenStream(host string) {
	hystrixStreamHandler := hystrix.NewStreamHandler()
	hystrixStreamHandler.Start()
	go func() {
		log.Fatal(http.ListenAndServe(host, hystrixStreamHandler))
	}()
}
