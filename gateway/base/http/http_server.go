package main

import "net/http"

// HTTP 服务器
//
// 步骤：
// 	1.注册路由
//		设置路由规则，即访问路径
//		定义该路由规则的处理器：回调函数
// 	2.启动监听并提供服务
func main() {
	// 1.注册路由和回调函数
	http.HandleFunc("/hello", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("hello, this is http server!"))
	})
	// 2.启动监听并提供服务
	http.ListenAndServe("127.0.0.1:9527", nil)
}
