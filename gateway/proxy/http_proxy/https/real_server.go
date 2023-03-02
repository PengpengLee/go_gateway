package https

import (
	"fmt"
	"gateway/proxy/http_proxy/https/testdata"
	"golang.org/x/net/http2"
	"io"
	"log"
	"net/http"
	"time"
)

type RealServer struct {
	Addr string
}

func (r *RealServer) Run() {
	log.Println("Starting http server at " + r.Addr)
	mux := http.NewServeMux()
	mux.HandleFunc("/", r.HelloHandler)
	mux.HandleFunc("/base/error", r.ErrorHandler)
	server := &http.Server{
		Addr:         r.Addr,
		WriteTimeout: time.Second * 3,
		Handler:      mux,
	}
	go func() {
		// 支持HTTP2
		http2.ConfigureServer(server, &http2.Server{})
		// 支持HTTPS
		log.Fatal(server.ListenAndServeTLS(testdata.Path(TlsServerCrt), testdata.Path(TlsServerKey)))
	}()
}

func (r *RealServer) HelloHandler(w http.ResponseWriter, req *http.Request) {
	upath := fmt.Sprintf("http://%s%s\n", r.Addr, req.URL.Path)
	io.WriteString(w, upath)
}

func (r *RealServer) ErrorHandler(w http.ResponseWriter, req *http.Request) {
	upath := "error handler"
	w.WriteHeader(500)
	io.WriteString(w, upath)
}
