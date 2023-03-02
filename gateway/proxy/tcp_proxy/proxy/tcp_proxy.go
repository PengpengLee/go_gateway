package proxy

import (
	"context"
	"gateway/loadbalance"
	"io"
	"log"
	"net"
	"time"
)

// TCPReverseProxy TCP反向代理核心结构体
type TCPReverseProxy struct {
	// 下游真实服务器地址：host：port
	Addr string
	Ctx  context.Context // 上下文，单次请求单独设置

	DialTimeout     time.Duration // 拨号超时时间，持续时间
	Deadline        time.Duration // 拨号截止时间，截止日期
	KeepAlivePeriod time.Duration // 长连接超时时间

	// 拨号器，支持自定义：拨号成功，返回连接；拨号失败，返回error
	DialContext func(ctx context.Context, network, address string) (net.Conn, error)

	// TCP整合负载均衡器 入口函数
	// 执行指定的负载均衡算法，返回 TCP 服务器地址
	Director func(remoteAddr string) (string, error)

	// 修改响应，可选
	// 如果返回错误，则由 ErrorHandler 处理
	ModifyResponse func(net.Conn) error
	// 错误处理，可选
	ErrorHandler func(net.Conn, error)
}

// NewTcpLoadBalanceReverseProxy 创建支持负载均衡的TCP主机反向代理实例
func NewTcpLoadBalanceReverseProxy(c context.Context, lb loadbalance.LoadBalance) *TCPReverseProxy {
	pxy := &TCPReverseProxy{
		Ctx:             c,
		KeepAlivePeriod: time.Hour,
		DialTimeout:     10 * time.Second,
		Deadline:        time.Minute,
	}
	// 定义入口函数：通过负载均衡算法得出TCP服务器地址
	director := func(remoteAddr string) (nextAddr string, err error) {
		nextAddr, err = lb.Get(remoteAddr)
		if err != nil {
			log.Fatal("get next addr fail")
		}
		// 给代理实例属性赋值
		pxy.Addr = nextAddr
		return
	}
	pxy.Director = director
	return pxy
}

// NewSingleHostReverseProxy 创建单TCP主机反向代理实例
func NewSingleHostReverseProxy(addr string) *TCPReverseProxy {
	if addr == "" {
		panic("TCP ADDRESS must not be empty!")
	}
	return &TCPReverseProxy{
		Addr:            addr,             // 下游服务器地址
		DialTimeout:     10 * time.Second, // 拨号超时：10s
		Deadline:        time.Minute,      // 拨号截至时间，1min
		KeepAlivePeriod: time.Hour,        // 保活时间：1h
	}
}

// ServeTCP TCP服务函数，实现TCPHandler接口。
//
// 完成上下游连接，及数据的交换：
// 	接收上游连接
// 	向下游发送请求
// 	接收下游响应
// 	拷贝/修改，响应到上游连接
func (pxy *TCPReverseProxy) ServeTCP(ctx context.Context, src net.Conn) {
	var cancel context.CancelFunc // 检查是否有取消操作
	if pxy.DialTimeout >= 0 {     // 连接超时时间
		ctx, cancel = context.WithTimeout(ctx, pxy.DialTimeout)
	}
	if pxy.Deadline >= 0 { // 连接截至时间
		ctx, cancel = context.WithDeadline(ctx, time.Now().Add(pxy.DialTimeout))
	}
	if cancel != nil {
		defer cancel()
	}
	// 拨号器：使用系统默认拨号器，还是自定义拨号器
	if pxy.DialContext == nil {
		pxy.DialContext = (&net.Dialer{
			Timeout:   pxy.DialTimeout,              // 连接超时
			Deadline:  time.Now().Add(pxy.Deadline), // 连接截至时间
			KeepAlive: pxy.KeepAlivePeriod,          // 长连接超时
		}).DialContext
	}

	// 执行入口函数：获取下游TCP服务器地址
	pxy.Director(src.RemoteAddr().String())

	// 向下游发送请求
	dst, err := pxy.DialContext(ctx, "tcp", pxy.Addr)
	if err != nil {
		// 错误处理
		pxy.getErrorHandler()(src, err)
		src.Close()
		return
	}
	// 关闭下游连接
	defer func() { go dst.Close() }()
	// 修改下游服务器响应
	if !pxy.modifyResponse(dst) {
		return
	}

	// 设置dst的 keepAlive 参数，在数据请求之前
	if ka := pxy.keepAlivePeriod(); ka > 0 {
		if c, ok := dst.(*net.TCPConn); ok {
			c.SetKeepAlive(true)
			c.SetKeepAlivePeriod(ka)
		}
	}

	//// 从下游拷贝到上游
	//_, err = io.Copy(src, dst)
	//if err != nil {
	//	// 错误处理
	//	pxy.getErrorHandler()(dst, err)
	//	dst.Close()
	//}

	// 数据拷贝：TCP连接是双向通道，支持全双工通信
	// 启动两个协程完成拷贝动作，二者互不干扰
	errc := make(chan error, 1)
	go bytesCopy(errc, src, dst) //	下游 -> 上游
	go bytesCopy(errc, dst, src) //	上游 -> 下游
	if errc != nil {
		// 错误处理
		pxy.getErrorHandler()(dst, <-errc)
	}
}

// 通过此函数修改响应，如果没有问题，则返回true，否则返回false
func (pxy *TCPReverseProxy) modifyResponse(res net.Conn) bool {
	if pxy.ModifyResponse == nil {
		return true
	}
	if err := pxy.ModifyResponse(res); err != nil {
		res.Close() // 关闭连接
		// 错误处理
		pxy.getErrorHandler()(res, err)
		return false
	}
	return true
}

func (pxy *TCPReverseProxy) getErrorHandler() func(net.Conn, error) {
	if pxy.ErrorHandler == nil {
		return pxy.defaultErrorHandler
	}
	return pxy.ErrorHandler
}

func (pxy *TCPReverseProxy) defaultErrorHandler(conn net.Conn, err error) {
	log.Printf("TCP proxy: for conn %v, error: %v", conn.RemoteAddr().String(), err)
}

func (pxy *TCPReverseProxy) keepAlivePeriod() time.Duration {
	if pxy.KeepAlivePeriod != 0 {
		return pxy.KeepAlivePeriod
	}
	return time.Minute
}

// bytesCopy 拷贝两个连接中的数据
// 第一个参数：错误通道
// 第二个参数：目标位置
// 第三个参数：源位置
func bytesCopy(errc chan<- error, dst, src net.Conn) {
	_, err := io.Copy(dst, src)
	errc <- err
}
