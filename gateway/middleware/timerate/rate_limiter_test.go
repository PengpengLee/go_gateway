package timerate

import (
	"context"
	r "gateway/middleware/router/http"
	"gateway/proxy"
	"golang.org/x/time/rate"
	"log"
	"net/http"
	"net/url"
	"testing"
	"time"
)

// TestRateLimiter 测试 time/rate 限速器使用
//	rate.NewLimiter(limit, burst)
//		limit表示每秒产生token数
//		burst最多存token数
// 实现步骤：
//	1.构建一个限速器
//	2.获取token
//		Wait 阻塞，直到获取token
// 		Reserve 预约，等待指定时间，再获取token
//		Allow 返回bool值，判断当天是否可以获取token
// 		三选其一即可
func TestRateLimiter(t *testing.T) {
	//	1.构建一个限速器
	// r: l.Limit() 每秒产生 token的数量
	// b: l.Burst() 最大的 token 数量
	l := rate.NewLimiter(1, 5)
	//	2.获取token
	for i := 0; i < 100; i++ {
		log.Println("before Wait", i)
		// 阻塞等待直到获取到一个token
		// 最大超时时间： 2秒
		c, _ := context.WithTimeout(context.Background(), time.Second*2)
		if err := l.Wait(c); err != nil {
			log.Println("limiter wait err:" + err.Error())
		}
		log.Println("after Wait")

		// 返回预计需要等待多久才有新的 token，可以通过等待指定时间再执行任务
		r := l.Reserve()
		// 检查限速器是否在最长等待时间内提供token
		if !r.OK() {
			return
		}
		log.Println("reserve Delay:", r.Delay())
		time.Sleep(r.Delay())

		// 判断当前是否可以取到token
		// 若返回true，则已经获取到token
		log.Println("Allow:", l.Allow())

		time.Sleep(200 * time.Millisecond)
	}
}

func TestRateLimiter2(t *testing.T) {
	customHandler := func(c *r.SliceRouteContext) http.Handler {
		rs1 := "http://127.0.0.1:8001/"
		url1, err1 := url.Parse(rs1)
		if err1 != nil {
			log.Println(err1)
		}

		rs2 := "http://127.0.0.1:8002/haha"
		url2, err2 := url.Parse(rs2)
		if err2 != nil {
			log.Println(err2)
		}

		urls := []*url.URL{url1, url2}
		return proxy.NewMultipleHostsReverseProxy(c.Ctx, urls)
	}

	var addr = "127.0.0.1:8006"
	log.Println("Starting http server at:" + addr)

	router := r.NewSliceRouter()
	router.Group("/").Use(RateLimiter())
	routerHandler := r.NewSliceRouterHandler(customHandler, router)
	log.Fatal(http.ListenAndServe(addr, routerHandler))
}
