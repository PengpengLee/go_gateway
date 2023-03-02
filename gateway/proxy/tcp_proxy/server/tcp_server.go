package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// TCPServer TCP服务核心结构体。监听指定主机，并提供服务
// Addr 	必选，主机地址
// Handler 	必选，回调函数，处理TCP请求。提供默认实现
type TCPServer struct {
	Addr    string          // 主机地址
	Handler TCPHandler      // 回调函数，处理TCP请求
	BaseCxt context.Context // 上下文，收集取消、终止、错误等信息
	err     error           // TCP Error

	ReadTimeout      time.Duration // 读超时
	WriteTimeout     time.Duration // 写超时
	KeepAliveTimeout time.Duration // 长连接超时

	mu         sync.Mutex         // 连接关闭等关键动作需要加锁
	doneChan   chan struct{}      // 服务已完成，监听系统信号
	inShutdown int32              // 服务终止：0-未关闭，1-已关闭
	l          *onceCloseListener // 服务器监听器，使用完成要进程关闭
}

type TCPHandler interface {
	// ServeTCP 提供TCP服务
	// ctx：连接上下文
	// conn：TCP连接实例，用于读写操作
	ServeTCP(ctx context.Context, conn net.Conn)
}

type tcpHandler struct {
}

func (t *tcpHandler) ServeTCP(ctx context.Context, conn net.Conn) {
	conn.Write([]byte("Pong! TCP handler here.\n"))
}

var (
	ErrServerClosed     = errors.New("tcp: Server closed")
	ErrAbortHandler     = errors.New("net/tcp: abort Handler")
	ServerContextKey    = &contextKey{"tcp-server"}
	LocalAddrContextKey = &contextKey{"local-addr"}
)

func (srv *TCPServer) ListenAndServe() error {
	if srv.shuttingDown() {
		return ErrServerClosed
	}
	addr := srv.Addr // 主机地址，非空
	if addr == "" {
		return errors.New("we need Address")
	}
	if srv.Handler == nil { // 回调函数，提供默认实现
		srv.Handler = &tcpHandler{}
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return srv.Serve(ln)
}

func ListenAndServe(addr string, handler TCPHandler) error {
	server := &TCPServer{Addr: addr, Handler: handler}
	return server.ListenAndServe()
}

func (srv *TCPServer) Serve(l net.Listener) error {
	srv.l = &onceCloseListener{Listener: l}
	defer l.Close() // 执行监听器的关闭

	if srv.BaseCxt == nil {
		srv.BaseCxt = context.Background()
	}
	baseCtx := srv.BaseCxt
	ctx := context.WithValue(baseCtx, ServerContextKey, srv)
	for {
		rw, err := l.Accept()
		if err != nil {
			select {
			case <-srv.getDoneChan():
				return ErrServerClosed
			default:
			}
			return err
		}
		c := srv.newConn(rw) // 对 TCPConn 二次封装
		go c.serve(ctx)      // handler回调函数的调用
	}
}

func (srv *TCPServer) newConn(rwc net.Conn) *conn {
	c := &conn{
		server:     srv,
		rwc:        rwc,
		remoteAddr: rwc.RemoteAddr().String(),
	}
	// 设置参数：从 TCPServer 中取字段，赋值给 TCPConn
	if t := srv.ReadTimeout; t != 0 {
		c.rwc.SetReadDeadline(time.Now().Add(t))
	}
	if t := srv.WriteTimeout; t != 0 {
		c.rwc.SetWriteDeadline(time.Now().Add(t))
	}
	if t := srv.KeepAliveTimeout; t != 0 {
		if tcpConn, ok := c.rwc.(*net.TCPConn); ok {
			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(t)
		}
	}
	return c
}
func (c *conn) serve(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil && err != ErrAbortHandler {
			const size = 64 << 10
			buf := make([]byte, size) // 65536
			buf = buf[:runtime.Stack(buf, false)]
			fmt.Printf("tcp: panic serving %v: %v\n%s", c.remoteAddr, err, buf)
		}
		c.rwc.Close()
	}()

	ctx = context.WithValue(ctx, LocalAddrContextKey, c.rwc.LocalAddr())
	if c.server.Handler == nil {
		panic("TCP handler empty!")
	}
	c.server.Handler.ServeTCP(ctx, c.rwc)
}

type conn struct {
	server     *TCPServer
	rwc        net.Conn
	remoteAddr string
}

type onceCloseListener struct {
	net.Listener
	once     sync.Once
	closeErr error
}

func (oc *onceCloseListener) Close() error {
	oc.once.Do(oc.close)
	return oc.closeErr
}

func (oc *onceCloseListener) close() { oc.closeErr = oc.Listener.Close() }

// Close TCPServer关闭功能
//
func (srv *TCPServer) Close() error {
	//srv.inShutdown = 1
	atomic.StoreInt32(&srv.inShutdown, 1) // 用原子操作修改服务器状态字段：1-关闭
	close(srv.doneChan)                   // 关闭channel
	srv.l.Close()                         // 关闭监听：listener
	return nil

}

// 检查当前服务器是否已关闭
func (srv *TCPServer) shuttingDown() bool {
	// 0-未关闭，1-已关闭
	return atomic.LoadInt32(&srv.inShutdown) != 0
}

type contextKey struct {
	name string
}

func (s *TCPServer) getDoneChan() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.doneChan == nil {
		s.doneChan = make(chan struct{})
	}
	return s.doneChan
}
