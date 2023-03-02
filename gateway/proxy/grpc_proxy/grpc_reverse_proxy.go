package grpc_proxy

import (
	"context"
	"gateway/proxy/grpc_proxy/public"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"io"
	"strings"
)

type handler struct {
	// 流式 请求协调者
	director StreamDirector
}

// handler from here
// 0.过滤非RPC请求
// 1.构建一个下游连接器：ClientStream
//	创建下游连接：往下游真实服务器创建连接
// 	封装下游客户端流实例
// 2.上游与下游数据拷贝
// 3.关闭双向流
func (h *handler) handler(srv interface{}, pxyServerStream grpc.ServerStream) error {
	// 0.过滤非RPC请求
	// "/service/method"
	methodName, ok := grpc.MethodFromServerStream(pxyServerStream)
	if !ok { // 非RPC请求
		return status.Errorf(codes.Internal, "There is no RPC-Request in this context")
	}
	// 不处理内部请求
	if strings.HasPrefix(methodName, "/com.example.internal") {
		return status.Errorf(codes.Unimplemented, "Unimplemented method")
	}

	// 1.构建一个下游连接器：ClientStream
	ctx := pxyServerStream.Context()

	// 1.1.	创建下游连接：往下游真实服务器创建连接
	//pxyClientConn, err := grpc.DialContext(ctx, "localhost:8005",
	//	// 自定义编码
	//	grpc.WithDefaultCallOptions(grpc.CallContentSubtype(public.Codec().Name())),
	//	// 禁用安全传输
	//	grpc.WithTransportCredentials(insecure.NewCredentials()))
	// 负载均衡算法获取下游服务器地址
	ctx, pxyClientConn, err := h.director(ctx, methodName)
	if err != nil {
		return err
	}
	defer pxyClientConn.Close()

	// 从上游请求上下文中获取元数据
	md, _ := metadata.FromIncomingContext(ctx)
	// 获取取消函数
	outCtx, clientCancel := context.WithCancel(ctx)
	// 封装下游请求的上下文
	outCtx = metadata.NewOutgoingContext(outCtx, md)

	// 1.2.	封装下游客户端流实例
	pxyStreamDesc := &grpc.StreamDesc{
		ServerStreams: true,
		ClientStreams: true,
	}
	pxyClientStream, err := grpc.NewClientStream(outCtx, pxyStreamDesc, pxyClientConn, methodName)
	if err != nil {
		return err
	}

	// 2.上游与下游数据拷贝
	// 把上游请求消息，发送给下游真实服务器
	s2cErrChan := h.serverToClient(pxyClientStream, pxyServerStream)
	// 把下游响应消息，发回给上游客户端
	c2sErrChan := h.clientToServer(pxyServerStream, pxyClientStream)

	// 3.关闭双向流

	// C/S双方谁会先关闭chanel，是不确定的，因此用select语句进行随机选择
	for i := 0; i < 2; i++ {
		select { // select 会阻塞，直到有任何一个（或多个）case可以被执行
		case s2cErr := <-s2cErrChan: // 向下游发消息
			if s2cErr == io.EOF {
				// 接收到了发送结束的信号，并且不再发送.
				// 关闭代理客户端发送流
				pxyClientStream.CloseSend()
			} else {
				// 接收上游的消息过程出现了问题/往下游发送过程出现了问题
				// 取消发送，并返回错误
				if clientCancel != nil {
					clientCancel()
				}
				return status.Errorf(codes.Internal, "failed proxying server to client:%v", s2cErr)
			}
		case c2sErr := <-c2sErrChan: // 往上游回写消息
			// 返回的error：io.EOF; gPRC error
			// Trailer：metadata，当流被关闭（ClientStream），读取消息得到error（gRPC，io.EOF）生成元数据
			pxyServerStream.SetTrailer(pxyClientStream.Trailer())

			if c2sErr != io.EOF {
				return c2sErr
			}
			return nil
		}
	}

	return nil
}

func (h *handler) clientToServer(dst grpc.ServerStream, src grpc.ClientStream) chan error {
	res := make(chan error, 1)
	go func() {
		//msg := &proto.EchoResponse{}
		msg := &public.Frame{}
		for i := 0; ; i++ {
			if i == 0 {
				// response header进行处理
				// 客户端读取响应时，会先读取响应头，然后作出相应的处理
				// 所以有必要设置响应头
				md, err := src.Header()
				if err != nil {
					res <- err
					break
				}
				if err = dst.SendHeader(md); err != nil {
					res <- err
					break
				}
			}

			if err := src.RecvMsg(msg); err != nil {
				res <- err // may be io.EOF / error
				break
			}
			if err := dst.SendMsg(msg); err != nil {
				res <- err // stream done, breaks
				break
			}
		}
	}()
	return res
}

func (h *handler) serverToClient(dst grpc.ClientStream, src grpc.ServerStream) chan error {
	res := make(chan error, 1)
	go func() {
		//msg := &proto.EchoRequest{}
		msg := &public.Frame{}
		for {
			// 客户端请求头，拷贝到下游
			// X-Forward-For
			// clientHeaderToServer(dst grpc.ClientStream, src grpc.ServerStream)
			// 服务器只有读取到第一条客户端消息的之后，才可以读取请求头

			if err := src.RecvMsg(msg); err != nil {
				res <- err // may be io.EOF / error
				break
			}
			if err := dst.SendMsg(msg); err != nil {
				res <- err
				break
			}
		}

	}()
	return res
}

// StreamDirector returns a gRPC ClientConn to be used to forward the call to.
//
// The presence of the `Context` allows for rich filtering, e.g. based on Metadata (headers).
// If no handling is meant to be done, a `codes.NotImplemented` gRPC error should be returned.
//
// The context returned from this function should be the context for the *outgoing* (to backend) call. In case you want
// to forward any Metadata between the inbound request and outbound requests, you should do it manually. However, you
// *must* propagate the cancel function (`context.WithCancel`) of the inbound context to the one returned.
//
// It is worth noting that the StreamDirector will be fired *after* all server-side stream interceptors
// are invoked. So decisions around authorization, monitoring etc. are better to be handled there.
//
// See the rather rich example.
type StreamDirector func(ctx context.Context, fullMethodName string) (context.Context, *grpc.ClientConn, error)

// TransparentHandler returns a handler that attempts to proxy all requests that are not registered in the server.
// The indented use here is as a transparent proxy, where the server doesn't know about the services implemented by the
// backends. It should be used as a `grpc.UnknownServiceHandler`.
func TransparentHandler(director StreamDirector) grpc.StreamHandler {
	streamer := &handler{director}
	return streamer.handler
}
