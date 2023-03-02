package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

var (
	proxy_addr = "127.0.0.1:8082"
	serverURL  = "http://127.0.0.1:8002"
)

func main() {
	url, err := url.Parse(serverURL)
	if err != nil {
		log.Println(err)
	}
	_proxy := httputil.NewSingleHostReverseProxy(url)
	log.Println("Starting websocket proxy at " + proxy_addr)
	log.Fatal(http.ListenAndServe(proxy_addr, _proxy))
}
