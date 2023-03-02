package tcp

import (
	"context"
	"gateway/middleware/router/http"
	tcp "gateway/proxy/tcp_proxy/server"
	"net"
)

type TcpHandlerFunc func(*TcpSliceRouteContext)

// TcpSliceRouter router 结构体
type TcpSliceRouter struct {
	groups []*TcpSliceRoute
}

// TcpSliceRoute tcp路由结构体
type TcpSliceRoute struct {
	*TcpSliceRouter
	path     string
	handlers []TcpHandlerFunc
}

// TcpSliceRouteContext router上下文
type TcpSliceRouteContext struct {
	*TcpSliceRoute

	index int8

	Ctx  context.Context
	Conn net.Conn
}

// NewTcpSliceRouter 构造 router
func NewTcpSliceRouter() *TcpSliceRouter {
	return &TcpSliceRouter{}
}

// Group 创建 Group
func (g *TcpSliceRouter) Group(path string) *TcpSliceRoute {
	return &TcpSliceRoute{
		TcpSliceRouter: g,
		path:           path,
	}
}

// Use 构造回调方法
func (tr *TcpSliceRoute) Use(middlewares ...TcpHandlerFunc) *TcpSliceRoute {
	tr.handlers = append(tr.handlers, middlewares...)
	existsFlag := false
	for _, tcpRoute := range tr.TcpSliceRouter.groups {
		if tcpRoute == tr {
			existsFlag = true
			break
		}
	}
	if !existsFlag {
		tr.TcpSliceRouter.groups = append(tr.TcpSliceRouter.groups, tr)
	}
	return tr
}

type tcpHandleFunc func(*TcpSliceRouteContext) tcp.TCPHandler

type TcpSliceRouterHandler struct {
	coreFunc tcpHandleFunc
	router   *TcpSliceRouter
}

func NewTcpSliceRouterHandler(coreFunc tcpHandleFunc, router *TcpSliceRouter) *TcpSliceRouterHandler {
	return &TcpSliceRouterHandler{
		coreFunc: coreFunc,
		router:   router,
	}
}

func (w *TcpSliceRouterHandler) ServeTCP(ctx context.Context, conn net.Conn) {
	c := NewTcpSliceRouterContext(conn, w.router, ctx)
	c.handlers = append(c.handlers, func(c *TcpSliceRouteContext) {
		w.coreFunc(c).ServeTCP(ctx, conn)
	})
	c.Reset()
	c.Next()
}

func NewTcpSliceRouterContext(conn net.Conn, r *TcpSliceRouter, ctx context.Context) *TcpSliceRouteContext {
	newTcpSliceGroup := &TcpSliceRoute{}
	*newTcpSliceGroup = *r.groups[0] //浅拷贝数组指针
	c := &TcpSliceRouteContext{Conn: conn, TcpSliceRoute: newTcpSliceGroup, Ctx: ctx}
	c.Reset()
	return c
}

func (c *TcpSliceRouteContext) Get(key interface{}) interface{} {
	return c.Ctx.Value(key)
}

func (c *TcpSliceRouteContext) Set(key, val interface{}) {
	c.Ctx = context.WithValue(c.Ctx, key, val)
}

// Next 从最先加入中间件开始回调
func (c *TcpSliceRouteContext) Next() {
	c.index++
	for c.index < int8(len(c.handlers)) {
		c.handlers[c.index](c)
		c.index++
	}
}

// Abort 跳出中间件方法
func (c *TcpSliceRouteContext) Abort() {
	c.index = http.AbortIndex
}

// IsAborted 是否跳过了回调
func (c *TcpSliceRouteContext) IsAborted() bool {
	return c.index >= http.AbortIndex
}

// Reset 重置回调
func (c *TcpSliceRouteContext) Reset() {
	c.index = -1
}
