package main

import (
	"net/http"
)

// HTTP Server 源码解析
//
// 步骤：
// 	1.创建路由
//		设置路由规则
//		定义该路由规则的处理器：回调函数
// 	2.创建服务器
// 	3.监听端口并提供服务
func main() {
	// 1.注册路由和回调函数
	// pattern: 模式, 即路由
	// handler: 请求处理器, 接收请求并完成响应
	// 路由: []slice 按照长度,从长到短
	http.HandleFunc("/hello", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("hello, this is grpc_server_client!"))
	})
	//http.HandleFunc("/hello/world", func(writer http.ResponseWriter, request *http.Request) {
	//	writer.Write([]byte("hello world, this is grpc_server_client!"))
	//})
	//http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
	//	writer.Write([]byte("你好, this is grpc_server_client!"))
	//})
	// 2.启动监听并提供服务
	// addr: 服务器地址, 格式: host:port
	// handler: 处理HTTP请求的处理器, 默认使用http.DefaultServeMux=> ServeMux, 实现了 Handler 接口
	http.ListenAndServe("127.0.0.1:9527", nil)

	// 源码分析：
	// 1.创建路由器：设置路由规则
	// ServeMux.Handle(pattern, HandlerFunc(handler))
	//type ServeMux struct {		// HTTP请求多路复用器：路由器
	//	mu    sync.RWMutex			// 读写锁：可共享读，不可共享写；写时不读，读时不写（排它锁）
	//	m     map[string]muxEntry	// key: pattern; value: {Handler, pattern}
	//	es    []muxEntry 			// entries切片，按URL从长到短排序，方便匹配到最佳路由
	//	hosts bool       			// pattern中是否有主机名
	//}
	//func (mux *ServeMux) Handle(pattern string, handler Handler) {
	//	mux.mu.Lock()								// 加锁
	//	defer mux.mu.Unlock()
	//	......
	//	if mux.m == nil {							// map初始化
	//		mux.m = make(map[string]muxEntry)
	//	}
	//	e := muxEntry{h: handler, pattern: pattern}	// 完成路由与其处理函数的映射
	//	mux.m[pattern] = e
	//	if pattern[len(pattern)-1] == '/' {
	//		mux.es = appendSorted(mux.es, e)		// 将新的Entry放到entries切片正确的位置
	//	}
	//	if pattern[0] != '/' {
	//		mux.hosts = true						// 路径不以 / 开头，就是以主机名开头
	//	}
	//}
	// 1.创建路由器：回调函数。最终请求处理将交付给ServeHTTP的实现函数
	// ServeMux.Handle(pattern, HandlerFunc(handler))
	// 封装成 HandlerFunc 类型，该类型实现了 Handler 接口的 ServerHTTP 函数
	//type HandlerFunc func(ResponseWriter, *Request)
	//func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
	//	f(w, r)
	//}
	//type Handler interface {
	//	ServeHTTP(ResponseWriter, *Request)
	//}

	// 2.创建服务器
	//func ListenAndServe(addr string, handler Handler) error {
	//	grpc_server_client := &Server{Addr: addr, Handler: handler}	// 这里的handler即ServeMux
	//	return grpc_server_client.ListenAndServe()
	//}

	// 3.监听端口并提供服务
	//func (srv *Server) ListenAndServe() error {
	//	...
	//	ln, err := net.Listen("tcp_proxy", addr)		// 联网方式：tcp协议
	//	if err != nil {
	//		return err
	//	}
	//	return srv.Serve(ln)
	//}
	// Serve函数用于创建HTTP服务。
	// 为每一个请求创建一个新的协程（goroutine），该服务协程会读取请求信息，
	// 并调用对应的处理器（handler）完成响应。
	// 最后，监听会被关闭，连接也会断开。
	//func (srv *Server) Serve(l net.Listener) error {
	//	...
	//	origListener := l
	//	l = &onceCloseListener{Listener: l}		// 封装一把，保护监听器不受多次close调用的影响
	//	defer l.Close()
	//	...
	//
	//	var tempDelay time.Duration // how long to sleep on accept failure
	//
	//	ctx := context.WithValue(baseCtx, ServerContextKey, srv)
	//	for {									// 死循环：监听器一直尝试获取连接
	//		rw, err := l.Accept()				// 阻塞，直到监听器获取到连接。rw为TCPConn实例
	//		if err != nil {
	//			... // 连接失败，超时重试
	//		}
	//		...
	//		c := srv.newConn(rw)				// TCPConn是Conn接口的实现类，其中封装了一个内部类conn实例
	//		c.setState(c.rwc, StateNew, runHooks) // before Serve can return
	//		go c.serve(connCtx)					// 每成功获取一个连接，就启动一个协程处理请求和响应
	//	}
	//}
	// 继续往下：
	// func (c *conn) serve(ctx context.Context) {
	//	c.remoteAddr = c.rwc.RemoteAddr().String()
	//	ctx = context.WithValue(ctx, LocalAddrContextKey, c.rwc.LocalAddr())
	//	var inFlightResponse *response
	//	defer func() {		// 收尾工作：异常处理，日志，连接关闭，连接状态设置
	//		if err := recover(); err != nil && err != ErrAbortHandler {
	//			...
	//			c.grpc_server_client.logf("http_proxy: panic serving %v: %v\n%s", c.remoteAddr, err, buf)
	//			..
	//			c.close()
	//			...
	//			c.setState(c.rwc, StateClosed, runHooks)
	//		}
	//	}()
	//	// TLS握手，在TCP连接之后，HTTP报文之前
	//	if tlsConn, ok := c.rwc.(*tls.Conn); ok {
	//		tlsTO := c.grpc_server_client.tlsHandshakeTimeout()
	//		...
	//		c.tlsState = new(tls.ConnectionState)
	//		*c.tlsState = tlsConn.ConnectionState()
	//		...
	//	}
	//	// HTTP/1.x from here on.
	//	...
	//	c.r = &connReader{conn: c}									// 连接读取器
	//	c.bufr = newBufioReader(c.r)								// 读缓冲区
	//	c.bufw = newBufioWriterSize(checkConnErrorWriter{c}, 4<<10)	// 写缓冲区，4K
	//
	//	for {														// 又一个死循环，一直尝试获取请求数据
	//		w, err := c.readRequest(ctx)							// 读取请求，返回response实例 w
	//		if c.r.remain != c.grpc_server_client.initialReadLimitSize() {		// 读取过程处于活动状态
	//			// If we read any bytes off the wire, we're active.
	//			c.setState(c.rwc, StateActive, runHooks)
	//		}
	//		if err != nil {											// 错误处理
	//			const errorHeaders = "\r\nContent-Type: text/plain; charset=utf-8\r\nConnection: close\r\n\r\n"
	//			switch {
	//			case err == errTooLarge:							// 431，客户端发送文件太大，响应后可能还在发送
	//				// ...
	//			case isUnsupportedTEError(err):						// 501，无法识别客户端编码
	//				// ...
	//				fmt.Fprintf(c.rwc, "HTTP/1.1 %d %s%sUnsupported transfer encoding", code, StatusText(code), errorHeaders)
	//				return
	//			case isCommonNetReadError(err):
	//				return // don't reply
	//			default:											// 默认处理方式
	//				if v, ok := err.(statusError); ok {				// 20x，服务器接收请求并处理
	//					fmt.Fprintf(c.rwc, "HTTP/1.1 %d %s: %s%s%d %s: %s", v.code, StatusText(v.code), v.text, errorHeaders, v.code, StatusText(v.code), v.text)
	//					return
	//				}
	//				publicErr := "400 Bad Request"					// 400，错误请求，服务端无此资源
	//				fmt.Fprintf(c.rwc, "HTTP/1.1 "+publicErr+errorHeaders+publicErr)
	//				return
	//			}
	//		}
	//		...
	//
	//		// HTTP cannot have multiple simultaneous active requests.[*]
	//		// Until the grpc_server_client replies to this request, it can't read another,
	//		// so we might as well run the handler in this goroutine.
	//		// [*] Not strictly true: HTTP pipelining. We could let them all process
	//		// in parallel even if their responses need to be serialized.
	//		// But we're not going to implement HTTP pipelining because it
	//		// was never deployed in the wild and the answer is HTTP/2.
	//		inFlightResponse = w
	//		serverHandler{c.grpc_server_client}.ServeHTTP(w, w.req)			// 读取请求完成后，调用handler处理请求
	//		...
	//		w.finishRequest()									// 请求处理完成，response已写入，刷缓存
	//		if !w.shouldReuseConnection() {						// 尝试复用tcp连接
	//			if w.requestBodyLimitHit || w.closedRequestBodyEarly() {
	//				c.closeWriteAndWait()
	//			}
	//			return
	//		}
	//		c.setState(c.rwc, StateIdle, runHooks)				// 设置连接状态为 空闲
	//		c.curReq.Store((*response)(nil))					// 响应实例置空
	//
	//		if !w.conn.grpc_server_client.doKeepAlives() {					// HTTP1.1，持久连接，客户端可以继续发送
	//			// We're in shutdown mode. We might've replied
	//			// to the user without "Connection: close" and
	//			// they might think they can send another
	//			// request, but such is life with HTTP/1.1.
	//			return
	//		}
	//
	//		if d := c.grpc_server_client.idleTimeout(); d != 0 {			// 若服务器空闲时间未超时，
	//			c.rwc.SetReadDeadline(time.Now().Add(d))		// 则等待超时
	//			if _, err := c.bufr.Peek(4); err != nil {		// 再次尝试读取，看看是否有数据
	//				return
	//			}
	//		}
	//		c.rwc.SetReadDeadline(time.Time{})					// 设置截止时间：不截止。进入下次循环
	//	}
	// }
}

// HandlerFunc 定义Function类型，处理HTTP请求和响应
// go语言中，函数是类型，是一等公民：可以作为参数、返回值等
type HandlerFunc func(http.ResponseWriter, *http.Request)

// HTTP请求处理程序
//
// 类型 HandlerFunc 的默认实现
// 实现 Handler 接口的 ServeHTTP 函数
func (f HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f(w, r)
}
