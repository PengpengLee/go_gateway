package flowcount

import (
	"gateway/middleware/circuitbreaker"
	router "gateway/middleware/router/http"
	proxy "gateway/proxy/http_proxy/http"
	"log"
	"net/http"
	"net/url"
	"testing"
	"time"
)

var addr = "127.0.0.1:2002"

func TestRedisFlowCount(t *testing.T) {
	coreFunc := func(c *router.SliceRouteContext) http.Handler {
		rs1 := "http://127.0.0.1:2003/base"
		url1, err1 := url.Parse(rs1)
		if err1 != nil {
			log.Println(err1)
		}

		rs2 := "http://127.0.0.1:2004/base"
		url2, err2 := url.Parse(rs2)
		if err2 != nil {
			log.Println(err2)
		}

		urls := []*url.URL{url1, url2}
		return proxy.NewMultipleHostsReverseProxy(c.Ctx, urls)
	}

	log.Println("Starting httpserver at " + addr)
	circuitbreaker.ConfCircuitBreakerWithOpenStream(true)
	sliceRouter := router.NewSliceRouter()
	redisCounter, _ := NewRedisFlowCountService("redis_app", time.Second)
	sliceRouter.Group("/").Use(RedisFlowCountMiddleWare(redisCounter))
	routerHandler := router.NewSliceRouterHandler(coreFunc, sliceRouter)
	log.Fatal(http.ListenAndServe(addr, routerHandler))
}
