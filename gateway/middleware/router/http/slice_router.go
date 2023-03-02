package http

import (
	"context"
	"net/http"
	"strings"
)

// 最多 63 个中间件
const AbortIndex int8 = 63

// HandlerFunc 路由处理函数，以列表的形式存在
type HandlerFunc func(*SliceRouteContext)

// SliceRouter 方法数组路由器结构体
// 每个路由对应一个路径、多个处理器
type SliceRouter struct {
	groups []*sliceRoute
}

// sliceRoute 路由
// 维护一个路由数组，同一个请求可以有多个函数处理
type sliceRoute struct {
	// 反向指针，每个路由可以知道属于哪个路由器
	*SliceRouter

	// 请求路径
	path string
	// 请求处理器列表
	handlers []HandlerFunc
}

// SliceRouteContext 路由上下文
// 每个路由对应一个上下实例，同时维护请求和响应对象
type SliceRouteContext struct {
	*sliceRoute

	index int8

	Ctx context.Context
	Req *http.Request
	Rw  http.ResponseWriter
}

// NewSliceRouter 构造路由器实例
func NewSliceRouter() *SliceRouter {
	return &SliceRouter{}
}

// Group 根据指定路径构造路由
// 每个路由维护一个路由器指针
func (g *SliceRouter) Group(path string) *sliceRoute {
	// init sliceRoute
	return &sliceRoute{
		SliceRouter: g, // this
		path:        path,
	}
}

// Use 构造回调方法
// 将指定函数添加到路由的处理器列表中
func (route *sliceRoute) Use(middlewares ...HandlerFunc) *sliceRoute {
	// add func to handlers in sliceRoute
	route.handlers = append(route.handlers, middlewares...)
	// 当前路由在路由器中是否存在
	flag := false
	for _, r := range route.SliceRouter.groups {
		if route == r {
			flag = true
			break
		}
	}
	if !flag {
		// 不存在，则添加
		route.SliceRouter.groups = append(route.SliceRouter.groups, route)
	}
	return route
}

// 定义处理器类型函数
// 接收 *SliceRouteContext 类型作为参数
// 返回 http.Handler 结果
type handler func(*SliceRouteContext) http.Handler

// SliceRouterHandler 方法数组路由器的核心处理器
//	维护一个方法数组路由器的指针：*SliceRouter
// 	支持用户自定义处理器
type SliceRouterHandler struct {
	h handler
	// 维护一个方法数组路由器的指针
	router *SliceRouter
}

// NewSliceRouterHandler 创建 http 服务的处理器
// 将实现了 http.Handler 接口的实例返回
func NewSliceRouterHandler(h handler, router *SliceRouter) *SliceRouterHandler {
	// build http.handler instance with SliceRouter
	return &SliceRouterHandler{
		h:      h,
		router: router,
	}
}

// ServeHTTP 实现了 http.Handler 接口的方法
// 	作为当前路由器的 http 服务的处理器入口
// 实现步骤：
// 	1.初始化路由上下文实例
//	2.检查该路由是否绑定用户自定义处理函数，添加到路由处理列表中
// 	3.依次执行路由的处理函数（中间件）
func (rh *SliceRouterHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	c := NewSliceRouterContext(rw, req, rh.router)
	if rh.h != nil {
		c.handlers = append(c.handlers, func(c *SliceRouteContext) {
			rh.h(c).ServeHTTP(c.Rw, c.Req)
		})
	}
	// 	3.依次执行路由的处理函数（中间件）
	c.Reset()
	c.Next()
}

// NewSliceRouterContext 初始化路由上下文实例
func NewSliceRouterContext(rw http.ResponseWriter, req *http.Request, r *SliceRouter) *SliceRouteContext {
	// 初始化最长url匹配路由
	sr := &sliceRoute{}
	// 最长url前缀匹配
	matchUrlLen := 0
	for _, route := range r.groups {
		// uri匹配成功：前缀匹配
		if strings.HasPrefix(req.RequestURI, route.path) {
			// 记录最长匹配 uri
			pathLen := len(route.path)
			if pathLen > matchUrlLen {
				matchUrlLen = pathLen
				// 浅拷贝：拷贝数组指针
				*sr = *route
			}
		}
	}

	c := &SliceRouteContext{
		Rw:         rw,
		Req:        req,
		Ctx:        req.Context(),
		sliceRoute: sr}
	// 确保每一次请求，中间件（函数列表）都是从第一个开始执行
	c.Reset()
	return c
}

func (c *SliceRouteContext) Get(key interface{}) interface{} {
	return c.Ctx.Value(key)
}

func (c *SliceRouteContext) Set(key, val interface{}) {
	c.Ctx = context.WithValue(c.Ctx, key, val)
}

// Next 从最先加入中间件开始回调
func (c *SliceRouteContext) Next() {
	c.index++
	for c.index < int8(len(c.handlers)) {
		// 循环调用每一个handler
		c.handlers[c.index](c)
		c.index++
	}
}

// Abort 跳出中间件方法
func (c *SliceRouteContext) Abort() {
	c.index = AbortIndex
}

// IsAborted 是否跳过了回调
func (c *SliceRouteContext) IsAborted() bool {
	return c.index >= AbortIndex
}

// Reset 重置回调
func (c *SliceRouteContext) Reset() {
	c.index = -1
}
