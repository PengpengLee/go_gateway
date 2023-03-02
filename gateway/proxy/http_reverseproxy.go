package proxy

import (
	"bytes"
	"compress/gzip"
	"context"
	"gateway/loadbalance"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// HTTP反向代理完整版：用ReverseProxy实现
//
// 支持功能：
//	URL重写、更改内容、错误信息回调、连接池
// 	随机负载均衡、兼容websocket、兼容gzip压缩

// HTTP 连接池
var transport = &http.Transport{
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second, // 连接超时，拨号超时时间
		KeepAlive: 30 * time.Second, // 长连接超时时间
	}).DialContext,
	MaxIdleConns:          100,              // 最大空闲连接数
	IdleConnTimeout:       90 * time.Second, // 空闲连接超时时间
	TLSHandshakeTimeout:   10 * time.Second, // tls握手超时时间
	ExpectContinueTimeout: 1 * time.Second,  // 100-continue 超时时间
}

func NewLoadBalanceReverseProxy(ctx context.Context, lb loadbalance.LoadBalance) *httputil.ReverseProxy {
	// 请求协调者
	director := func(req *http.Request) {
		// 使用指定的负载均衡策略，获取服务地址
		nextAddr, err := lb.Get(req.URL.String())
		if err != nil {
			log.Fatal("get next addr fail")
		}
		target, err := url.Parse(nextAddr)
		if err != nil {
			log.Fatal(err)
		}

		targetQuery := target.RawQuery
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = joinURLPath(target.Path, req.URL.Path)
		req.Host = target.Host
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		if _, ok := req.Header["User-Agent"]; !ok {
			req.Header.Set("User-Agent", "user-agent")
		}
	}

	// 更改内容
	modifyFunc := func(resp *http.Response) error {
		// 兼容websocket
		if strings.Contains(resp.Header.Get("Connection"), "Upgrade") {
			return nil
		}
		var payload []byte
		var readErr error

		// 兼容gzip压缩
		if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
			gr, err := gzip.NewReader(resp.Body)
			if err != nil {
				return err
			}
			payload, readErr = ioutil.ReadAll(gr)
			resp.Header.Del("Content-Encoding")
		} else {
			payload, readErr = ioutil.ReadAll(resp.Body)
		}
		if readErr != nil {
			return readErr
		}

		// 异常请求时设置StatusCode
		if resp.StatusCode != 200 {
			payload = []byte("StatusCode error:" + string(payload))
		}
		// 预读了数据，需要内容重新回写
		context.WithValue(ctx, "payload", payload)
		context.WithValue(ctx, "status_code", resp.StatusCode)

		resp.Body = ioutil.NopCloser(bytes.NewBuffer(payload))
		resp.ContentLength = int64(len(payload))
		resp.Header.Set("Content-Length", strconv.FormatInt(int64(len(payload)), 10))
		return nil
	}

	// 错误回调 ：关闭real_server时测试，错误回调
	// 范围：transport.RoundTrip发生的错误、以及ModifyResponse发生的错误
	errFunc := func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "ErrorHandler error:"+err.Error(), http.StatusInternalServerError)
		log.Fatal(err)
	}

	return &httputil.ReverseProxy{Director: director, Transport: transport, ModifyResponse: modifyFunc, ErrorHandler: errFunc}
}

func NewMultipleHostsReverseProxy(ctx context.Context, targets []*url.URL) *httputil.ReverseProxy {
	// 请求协调者
	director := func(req *http.Request) {
		// 随机负载均衡：提供相同服务的URLs
		targetIndex := rand.Intn(len(targets))
		targetIndex = 0
		target := targets[targetIndex]
		targetQuery := target.RawQuery

		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = joinURLPath(target.Path, req.URL.Path)
		// 当对域名(非内网)反向代理时需要设置此项, 当作后端反向代理时不需要
		req.Host = target.Host
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		if _, ok := req.Header["User-Agent"]; !ok {
			req.Header.Set("User-Agent", "user-agent")
		}
	}

	// 修改返回内容
	modifyFunc := func(resp *http.Response) error {
		// 兼容websocket
		if strings.Contains(resp.Header.Get("Connection"), "Upgrade") {
			// websocket协议，不需要修改返回内容
			return nil
		}

		var payload []byte
		var readErr error
		// 兼容gzip压缩
		if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
			gr, err := gzip.NewReader(resp.Body)
			if err != nil {
				return err
			}
			payload, readErr = ioutil.ReadAll(gr)
			resp.Header.Del("Content-Encoding")
		} else {
			payload, readErr = ioutil.ReadAll(resp.Body)
		}
		if readErr != nil {
			return readErr
		}

		// 异常请求时设置状态码 StatusCode
		if resp.StatusCode != 200 {
			payload = []byte("StatusCode error:" + string(payload))
		}
		// 预读了数据，需要内容重新回写
		context.WithValue(ctx, "payload", payload)
		context.WithValue(ctx, "status_code", resp.StatusCode)

		resp.Body = ioutil.NopCloser(bytes.NewBuffer(payload))
		resp.ContentLength = int64(len(payload))
		resp.Header.Set("Content-Length", strconv.FormatInt(int64(len(payload)), 10))
		return nil
	}

	// 错误回调：当后台出现错误响应，会自动调用此函数
	// ModifyResponse 返回error，也会调用此函数
	// 为空时，出现错误返回502（错误网关）
	errFunc := func(w http.ResponseWriter, r *http.Request, err error) {
		// TODO error log
		http.Error(w, "ErrorHandler error:"+err.Error(), http.StatusInternalServerError)
	}

	return &httputil.ReverseProxy{
		Director:       director,
		Transport:      transport,
		ModifyResponse: modifyFunc,
		ErrorHandler:   errFunc}
}

// joinURLPath 合并 a 和 b 两个字符串，a在前，且不能有多余的斜杠
// a: "" or "/"
// b: /realserver ""
func joinURLPath(a, b string) string {
	aSlash := strings.HasSuffix(a, "/")
	bSlash := strings.HasPrefix(b, "/") // /realserver ->
	switch {
	case aSlash && bSlash:
		return a + b[1:] // b: /realserver -> realserver
	case aSlash || bSlash:
		return a + b
	}
	return a + "/" + b
}
