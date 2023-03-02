package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

// HTTP 客户端
//
// 步骤：
// 	1.创建客户端
// 	2.发起请求
// 	3.处理服务器响应
// 	4.关闭连接
func main() {
	// 1.创建客户端
	client := &http.Client{}
	// 2.发起请求
	// 使用GET方式发起请求
	// URL: 协议 + 主机 + 服务器端口 + 路由
	resp, err := client.Get("http://127.0.0.1:9527/hello")
	// 4.关闭连接
	defer resp.Body.Close()
	if err != nil {
		panic(err)
	}
	// 3.处理服务器响应：读取内容并打印
	bds, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(bds))
}
