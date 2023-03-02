package main

import (
	"fmt"
	"gateway/middleware/servicediscovery/zookeeper"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	server1 := &RealServer{Addr: "127.0.0.1:8001"}
	server1.Run()
	server2 := &RealServer{Addr: "127.0.0.1:8002"}
	server2.Run()

	// 监听系统的关闭信号， 否则主程终止后，将直接导致子程终止
	// 相当于监听： ctrl + C， kill命令
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}

// RealServer 下游真实服务器
type RealServer struct {
	Addr string // 服务器主机地址: {host:port}
}

// Run 新建协程启动服务器
// 下游机器启动时注册临时节点
func (r *RealServer) Run() {
	mux := http.NewServeMux()
	mux.HandleFunc("/realserver", r.HelloHandler)
	mux.HandleFunc("/realserver/error", r.ErrorHandler)
	server := &http.Server{
		Addr:         r.Addr,
		Handler:      mux,
		WriteTimeout: time.Second * 3,
	}
	// 以新的协程的方式启动服务
	go func() {
		// 下游服务器启动时，注册服务信息到zk
		zkManager := zookeeper.NewZkManager([]string{"192.168.154.132:2181"})
		err := zkManager.GetConnect()
		if err != nil {
			fmt.Println("connect zookeeper error:", err.Error())
		}
		defer zkManager.Close()
		// 注册的内容交付给zk服务器
		err = zkManager.RegisterServerPath("/realserver", r.Addr)
		if err != nil {
			fmt.Println("register node error:", err.Error())
		}
		log.Fatal(server.ListenAndServe())
	}()
}

// HelloHandler 路由处理器
func (r *RealServer) HelloHandler(w http.ResponseWriter, req *http.Request) {
	//newPath := fmt.Sprintf("Here is real grpc_server_client: http://%s%s", req.RemoteAddr, req.URL.Path)
	newPath := fmt.Sprintf("Here is real server: http://%s%s", r.Addr, req.URL.Path)
	w.Write([]byte(newPath))
	//go func() {
	//	// 此时，请求-响应已经完成
	//	for {
	//		// 服务器向客户端推送消息：失败 为什么？
	//		// 因为此时，请求-响应已经完成，尽管连接还保持着，但是服务器无法推送过去
	//		w.Write([]byte(newPath))
	//		time.Sleep(1 * time.Second)
	//	}
	//}()
}

// ErrorHandler 错误处理器
func (r *RealServer) ErrorHandler(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusInternalServerError) // 服务器内部错误
	w.Write([]byte("error: 服务器内部错误"))
}
