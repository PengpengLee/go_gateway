package http

import (
	"context"
	"fmt"
	"gateway/proxy"
	"log"
	"net/http"
	"net/url"
	"testing"
)

// TestSliceRouter 测试基于方法数组构建的中间件
// 	中间件绑定到路由上。
//	实现步骤：
// 	1.创建路由器：
// 		一个路由器，包含多个路由（请求）
//		每个路由都可以有多个处理器（回调函数）
//	2.构建 URI路由中间件：使用路由器对每个请求 URI构建路由中间件
//	3.构建方法数组（一系列的回调函数），并整合到 URI路由中间件
// 	4.将路由器作为 http 服务的处理器
func TestSliceRouter(t *testing.T) {
	// 服务器地址
	var addr = "127.0.0.1:8006"
	log.Println("Starting httpserver at " + addr)

	// 	1.创建路由器：
	// 一个路由器，包含多个路由（请求）
	// 每个路由都可以有多个处理器（回调函数）
	sliceRouter := NewSliceRouter()

	//	2.构建 URI路由中间件：注册请求URI
	routeRoot := sliceRouter.Group("/")
	//	3.构建方法数组（一系列的回调函数），并整合到 URI路由中间件
	// 绑定处理函数
	routeRoot.Use(handle, func(c *SliceRouteContext) {
		fmt.Println("reverse proxy")
		// 中间件请求到反向代理
		reverseProxy(c.Ctx).ServeHTTP(c.Rw, c.Req)
	})

	//	2.构建 URI路由中间件：注册请求URI
	routeBase := sliceRouter.Group("/base")
	//	3.构建方法数组（一系列的回调函数），并整合到 URI路由中间件
	// 绑定处理函数
	routeBase.Use(handle, func(c *SliceRouteContext) {
		// 中间件作为业务逻辑处理代码
		c.Rw.Write([]byte("test function"))
	})

	// 	4.将路由器作为 http 服务的处理器
	// 封装 sliceRouter 作为http服务的处理器
	var routerHandler http.Handler = NewSliceRouterHandler(nil, sliceRouter)
	http.ListenAndServe(addr, routerHandler)
}

func handle(c *SliceRouteContext) {
	log.Println("trace_in")
	c.Next()
	log.Println("trace_out")
}

func reverseProxy(c context.Context) http.Handler {
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
	return proxy.NewMultipleHostsReverseProxy(c, urls)
}
