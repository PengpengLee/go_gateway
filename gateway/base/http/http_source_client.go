package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

func main() {
	// 创建连接池
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second, //连接超时
			KeepAlive: 30 * time.Second, //探活时间
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,              //最大空闲连接
		IdleConnTimeout:       90 * time.Second, //空闲超时时间
		TLSHandshakeTimeout:   10 * time.Second, //tls握手超时时间
		ExpectContinueTimeout: 1 * time.Second,  //100-continue状态码超时时间
	}
	// 创建客户端
	client := &http.Client{
		Timeout:   time.Second * 30,
		Transport: transport,
	}
	// 请求数据
	resp, err := client.Get("http://127.0.0.1:9527/hello")
	defer resp.Body.Close()
	if err != nil {
		panic(err)
	}
	// 读取内容
	bds, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(bds))
}

// HTTP Client源码解析（Go1.18）
// 定义了执行单个HTTP事务的能力，返回给定请求对应的响应
// 请求并接收响应的主函数
//type RoundTripper interface {
//	RoundTrip(*Request) (*Response, error)
//}
// 传输实体————连接池，用于发起每一个HTTP请求
//var DefaultTransport RoundTripper = &Transport{
//	Proxy: ProxyFromEnvironment,
//	DialContext: (&net.Dialer{
//		Timeout:   30 * time.Second, //连接超时
//		KeepAlive: 30 * time.Second, //探活时间
//	}),
//	ForceAttemptHTTP2:     true,
//	MaxIdleConns:          100,              //最大空闲连接
//	IdleConnTimeout:       90 * time.Second, //空闲超时时间
//	TLSHandshakeTimeout:   10 * time.Second, //tls握手超时时间
//	ExpectContinueTimeout: 1 * time.Second,  //100-continue状态码超时时间
//}
// 完整的Transport结构体参考：
// type Transport struct {
//	idleMu       sync.Mutex
//	closeIdle    bool                                // 用户请求关闭所有的闲置连接
//	idleConn     map[connectMethodKey][]*persistConn // 每个host对应的闲置连接列表
//	idleConnWait map[connectMethodKey]wantConnQueue  // 每个host对应的等待闲置连接列表，在其它request将连接放回连接池前先看一下这个队列是否为空，不为空则直接将连接交由其中一个等待对象
//	idleLRU      connLRU                             // 用来清理过期的连接
//	reqMu       sync.Mutex
//	reqCanceler map[*Request]func(error)
//
//	connsPerHostMu   sync.Mutex
//	connsPerHost     map[connectMethodKey]int           // 每个host对应的等待连接个数
// 在当前主机/Client/Transport/连接池实体 中，等待获取连接的队列集合
// map：[key:请求方法, value:使用此方法发送请求的等待队列]
//	connsPerHostWait map[connectMethodKey]wantConnQueue // 每个host对应的等待连接列表
//	// 用于指定创建未加密的TCP连接的dial功能，如果该函数为空，则使用net包下的dial函数
//	DialContext func(ctx context.Context, network, addr string) (net.Conn, error)
//	Dial        func(network, addr string) (net.Conn, error)
//	// 以下两个函数处理https的请求
//	DialTLSContext func(ctx context.Context, network, addr string) (net.Conn, error)
//	DialTLS        func(network, addr string) (net.Conn, error)
//
//	DisableKeepAlives bool				// 是否复用连接
//	DisableCompression bool				// 是否压缩
//
//	MaxIdleConns int					// 总的最大闲置连接的个数
//	MaxIdleConnsPerHost int				// 每个host最大闲置连接的个数
//	MaxConnsPerHost int					// 每个host的最大连接个数，如果已经达到该数字，dial连接会被block住
//	IdleConnTimeout time.Duration		// 闲置连接的最大等待时间，一旦超过该时间，连接会被关闭
//
//	ResponseHeaderTimeout time.Duration	// 读超时，从写完请求到接受到返回头的总时间
//	ExpectContinueTimeout time.Duration	// Expect:100-continue两个请求间的超时时间
//	MaxResponseHeaderBytes int64		// 返回中header的限制
//	WriteBufferSize int					// write buffer的使用量
//	ReadBufferSize int 					// read buffer的使用量
//}
//
// HTTP请求结构体默认属性（部分）
//	req := &Request{
//		ctx:        ctx,
//		Method:     method,
//		URL:        u,
//		Proto:      "HTTP/1.1",
//		ProtoMajor: 1,
//		ProtoMinor: 1,
//		Header:     make(Header),
//		Body:       rc,
//		Host:       u.Host,
//	}
//
//	type Header map[string][]string
//	Header = map[string][]string{
//		"Accept-Encoding": {"gzip, deflate"},
//		"Accept-Language": {"en-us"},
//		"Foo": {"Bar", "two"},
//	}
// 发送请求
// grpc_client.GET(){
// 	c.do(...) { // GET、POST、HEAD、PostForm等方法都会执行这个do
// 		resp, didTimeout, err = c.send(req, deadline) { // line 725
//			resp, didTimeout, err = send(req, c.transport(), deadline) // line 176
//			send函数如下：
//			func send(ireq *Request, rt RoundTripper, deadline time.Time) (resp *Response, didTimeout func() bool, err error) {
//				......
//				发送请求的主函数，返回响应
//				resp, err = rt.RoundTrip(req) // line 252 rt=>RoundTripper
//			}
//		}
//	}
//}
// 	roundtrip.go line 16
//	func (t *Transport) RoundTrip(req *Request) (*Response, error) {
//		return t.roundTrip(req)
//	}
//	transport.go line 504
//	 func (t *Transport) roundTrip(req *Request) (*Response, error) {
//		trace := httptrace.ContextClientTrace(ctx)
// 		......
// 		for {
//			select {
//			case <-ctx.Done(): // 监听取消事件
//				req.closeBody()
//				return nil, ctx.Err()
//			default:
//			}
//			treq := &transportRequest{Request: req, trace: trace, cancelKey: cancelKey}
//			cm, err := t.connectMethodForRequest(treq) // cm connectMethod
//			....
//			pconn, err := t.getConn(treq, cm) // 获取连接
//			// 获取连接成功，则返回reponse
// 			// 获取连接失败，则进行清理工作，并尝试重试
//		}
//	 }
// type persistConn struct {
//		br        *bufio.Reader       // from conn
//		bw        *bufio.Writer       // to conn
//		reqch     chan requestAndChan  //read by readLoop
//		writech   chan writeRequest		//read by writeLoop
//	}
// func getConn(){
//		// 尝试获取空闲连接，成功则返回 persisConn持久连接实例
//		if delivered := t.queueForIdleConn(w); delivered { // line 1354
//			pc := w.pc
//			// Trace only for HTTP/1.
//			// HTTP/2 calls trace.GotConn itself.
//			if pc.alt == nil && trace != nil && trace.GotConn != nil {
//				trace.GotConn(pc.gotIdleConnTrace(pc.idleAt))
//			}
//			// set request canceler to some non-nil function so we
//			// can detect whether it was cleared between now and when
//			// we enter roundTrip
//			t.setReqCanceler(treq.cancelKey, func(error) {})
//			return pc, nil // 返回连接
//		}
//		// 获取空闲连接失败，排队等待拨号
// 		t.queueForDial(w)  // line 1372
//		// Wait for completion or cancellation.
//		select {
//		case <-w.ready: // 连接就绪(ready管道中收到一个信号)，获取新连接成功
//			// Trace success but only for HTTP/1.
//			// HTTP/2 calls trace.GotConn itself.
//			if w.pc != nil && w.pc.alt == nil && trace != nil && trace.GotConn != nil {
//				trace.GotConn(httptrace.GotConnInfo{Conn: w.pc.conn, Reused: w.pc.isReused()})
//			}
// 			......
//			return w.pc, w.err // 返回新建连接
//		}
//}
//	// queueForDial queues w to wait for permission to begin dialing.
//	// Once w receives permission to dial, it will do so in a separate goroutine.
//	func (t *Transport) queueForDial(w *wantConn) { // 1415
//		w.beforeDial()
//		if t.MaxConnsPerHost <= 0 { // 主机连接数没有上限
//			go t.dialConnFor(w)  	// 启动协程进行拨号
//			return
//		}
//		// 主机支持的连接数有上限，则创建连接过程要加锁，避免创建过多连接
//		t.connsPerHostMu.Lock()
//		defer t.connsPerHostMu.Unlock()
//		// 连接数未达上限
//		if n := t.connsPerHost[w.key]; n < t.MaxConnsPerHost {
//			if t.connsPerHost == nil {
//				t.connsPerHost = make(map[connectMethodKey]int)
//			}
//			t.connsPerHost[w.key] = n + 1
//			go t.dialConnFor(w)		// 启动协程进行拨号，异步创建连接
//			return
//		}
//		// 连接数已达上限。入队等待创建连接
//		if t.connsPerHostWait == nil { // 当前客户端第一次创建连接
//			t.connsPerHostWait = make(map[connectMethodKey]wantConnQueue)
//		}
//		q := t.connsPerHostWait[w.key]
//		q.cleanFront() 	// 尝试清理队头不再等待的连接->wantConn
//		q.pushBack(w) 	// 当前连接 入队尾，排队等待别的连接唤醒
//		t.connsPerHostWait[w.key] = q
//	}
// map的key的数据结构：
//	type connectMethodKey struct {
//		proxy, scheme, addr string
//		onlyH1              bool
//	}
// 正在创建的连接的数据结构
//	type wantConn struct {
//		cm    connectMethod
//		key   connectMethodKey // cm.key()
//		ctx   context.Context  // context for dial
//		ready chan struct{}    // closed when pc, err pair is delivered
//
//		// hooks for testing to know when dials are done
//		// beforeDial is called in the getConn goroutine when the dial is queued.
//		// afterDial is called when the dial is completed or canceled.
//		beforeDial func()
//		afterDial  func()
//
//		mu  sync.Mutex // protects pc, err, close(ready)
//		pc  *persistConn
//		err error
//	}
//
// 为连接进行拨号
//	func (t *Transport) dialConnFor(w *wantConn) {	// line 1446
//		......
//		pc, err := t.dialConn(w.ctx, w.cm)   // 为wantConn进行拨号
//		delivered := w.tryDeliver(pc, err) { // 尝试交付连接，即向wantConn.ready管道发送一个关闭信号
// 			......
//			close(w.ready) // 向已创建完成的连接的ready管道发送关闭信号
//			return true
//		}
//		t.putOrCloseIdleConn(pc) // 放到空闲连接池中
//		.......
//	}
//
// func (t *Transport) dialConn(ctx context.Context, cm connectMethod) (pconn *persistConn, err error) {
//		// 初始化persistConn实例
//		pconn = &persistConn{
//			t:             t,
//			cacheKey:      cm.key(),
//			reqch:         make(chan requestAndChan, 1),
//			writech:       make(chan writeRequest, 1),
//			closech:       make(chan struct{}),
//			writeErrCh:    make(chan error, 1),
//			writeLoopDone: make(chan struct{}),
//		}
//		if cm.scheme() == "https" && t.hasCustomTLSDialer() {
//			// 使用自定义的 ssl握手方式，进行拨号并创建连接（非代理）
//			pconn.conn, err = t.customDialTLS(ctx, "tcp", cm.addr())
//		} else {
//			// 基于 tcp 进行拨号并创建连接
//			conn, err := t.dial(ctx, "tcp", cm.addr())
//			if err != nil {
//				return nil, wrapErr(err)
//			}
//			pconn.conn = conn
//			if cm.scheme() == "https" {
//				var firstTLSHost string
//				if firstTLSHost, _, err = net.SplitHostPort(cm.addr()); err != nil {
//					return nil, wrapErr(err)
//				}
//				// tcp连接创建之后，进行ssl握手
//				if err = pconn.addTLS(ctx, firstTLSHost, trace); err != nil {
//					return nil, wrapErr(err)
//				}
//			}
//		}
//		// Proxy 服务器处理.
//		switch {
//		case cm.proxyURL == nil: // 无代理
//			// Do nothing. Not using a proxy.
//		case cm.proxyURL.Scheme == "socks5": // socks5代理，会话层，基于tcp，直接转发应用层数据包，不作修改，兼容各应用层协议
//		case cm.targetScheme == "http": // http to proxy, then CONNECT to 目标服务器
//		case cm.targetScheme == "https":
//		}
//		......
//		pconn.br = bufio.NewReaderSize(pconn, t.readBufferSize())
//		pconn.bw = bufio.NewWriterSize(persistConnWriter{pconn}, t.writeBufferSize())
//
//		go pconn.readLoop()		// 启用新的协程,监听pconn.reqch,循环地从管道中,读取数据
//		go pconn.writeLoop()	// 启用新的协程,监听pconn.writech,循环地往管道中,写入数据
//		return pconn, nil
//
