package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
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
//
func main() {
	// 下游真实服务器地址
	realServer := "http://127.0.0.1:8001/?a=1&b=2#container"
	// parse解析url
	// 从"http://127.0.0.1:8001?a=1&b=2#container"
	// 到"http://127.0.0.1:8001"
	serverURL, err := url.Parse(realServer)
	if err != nil {
		log.Println(err)
	}
	proxy := NewSingleHostReverseProxy(serverURL)
	// 代理服务器地址：8081
	var addr = "127.0.0.1:8081"
	log.Println("Starting proxy http grpc_server_client at:" + addr)
	http.ListenAndServe(addr, proxy)
}

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

func NewSingleHostReverseProxy(target *url.URL) *httputil.ReverseProxy {
	// 入口函数，协调者，管理者，URL重写
	targetQuery := target.RawQuery        // a=1&b=2
	director := func(req *http.Request) { // req: http://127.0.0.1:8081/realserver
		// 一个完整的URL包含：Scheme, Host, Path, RawQuery
		// Scheme: http
		// Host: 127.0.0.1:8000
		// Path: /realserver
		// RawQuery: a=1&b=2
		// Fragment: container
		// target: "http://127.0.0.1:8001/?a=1&b=2#container"
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host // 127.0.0.1:8001
		// result:req.URL: http://127.0.0.1:8001/realserver?a=1&b=2#container
		// target.Path: "" or "/"
		// req.URL.Path: /realserver
		// 合并两个Path，谁在前，谁在后？
		// 答案：target.Path 在前
		req.URL.Path = joinURLPath(target.Path, req.URL.Path)
		// 注意，应该将上游客户端请求参数与下游请求参数进行合并
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
	}

	// 修改返回内容
	modifyResponse := func(res *http.Response) error {
		fmt.Println("here is ModifyResponse Function")
		// 升级协议，不需要进行修改
		if res.StatusCode == 101 { // 101 Switching Protocols
			if strings.Contains(res.Header.Get("Connection"), "Upgrade") {
				return nil
			}
		}
		if res.StatusCode == 200 {
			srcBody, _ := ioutil.ReadAll(res.Body)
			newBody := []byte(string(srcBody) + "  mashibing")
			res.Body = ioutil.NopCloser(bytes.NewBuffer(newBody))
			length := int64(len(newBody))
			res.ContentLength = length
			res.Header.Set("Content-Length", strconv.FormatInt(length, 10))
		}
		return nil
		//return errors.New("出错了")
	}

	// 错误回调：当后台出现错误响应，会自动调用此函数
	// ModifyResponse 返回error，也会调用此函数
	// 为空时，出现错误返回502（错误网关）
	errFunc := func(w http.ResponseWriter, r *http.Request, err error) {
		//fmt.Println("here is error function....")
		http.Error(w, "ErrorHandler error:"+err.Error(), http.StatusInternalServerError)
	}

	return &httputil.ReverseProxy{
		Director:       director,
		ModifyResponse: modifyResponse,
		ErrorHandler:   errFunc,
		Transport:      transport,
	}
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
