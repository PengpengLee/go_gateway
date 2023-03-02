package whitelist

import (
	"fmt"
	"gateway/middleware/router/tcp"
	"strings"
)

var (
	WhiteList = []string{"127.0.0.1", "192.168.0.107"}
)

func IpWhiteListMiddleWare() func(c *tcp.TcpSliceRouteContext) {
	return func(c *tcp.TcpSliceRouteContext) {
		remoteAddr := c.Conn.RemoteAddr().String()
		if strings.ContainsAny(fmt.Sprint(WhiteList), remoteAddr) {
			c.Next()
		} else {
			c.Abort()
			c.Conn.Write([]byte("ip_whitelist auth invalid"))
			c.Conn.Close()
		}
	}
}
