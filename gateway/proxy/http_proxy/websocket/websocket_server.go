package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

func main() {
	var addr = "localhost:8002" // 下游真实服务器
	http.HandleFunc("/wsHandler", wsHandler)
	log.Println("Starting websocket grpc_server_client at " + addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	var upgrader = websocket.Upgrader{} // default options
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade error:", err)
		return
	}
	defer conn.Close()
	// Web Socket服务器主动向客户端推送消息
	//go func() {
	//	for {
	//		// TextMessage:1, BinaryMessage:2
	//		err := conn.WriteMessage(1, []byte("heart beat"))
	//		if err != nil {
	//			return
	//		}
	//		time.Sleep(3 * time.Second)
	//	}
	//}()

	for {
		// mt 消息类型，text/binary
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			log.Print("read error:", err)
			break
		}
		fmt.Printf("receive msg:%s", msg)
		newMsg := string(msg) + "haha"
		msg = []byte(newMsg)
		err = conn.WriteMessage(mt, msg)
		if err != nil {
			log.Print("write error:", err)
			break
		}
	}
}
