package timerate

import (
	"fmt"
	sr "gateway/middleware/router/http"
	"golang.org/x/time/rate"
)

// RateLimiter 网关集成限流功能
func RateLimiter(params ...int) func(c *sr.SliceRouteContext) {
	var r rate.Limit = 1
	var b = 2
	if len(params) == 2 {
		r = rate.Limit(params[0])
		b = params[1]
	}
	l := rate.NewLimiter(r, b)
	return func(c *sr.SliceRouteContext) {
		// 1.如果无法获取到token，则跳出中间件，直接返回
		if !l.Allow() {
			c.Rw.Write([]byte(fmt.Sprintf("rate limit:%v, %v", l.Limit(), l.Burst())))
			c.Abort()
			return
		}
		// 2.可以获取到token，执行中间件
		c.Next()
	}
}
