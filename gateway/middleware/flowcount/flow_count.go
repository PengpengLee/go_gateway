package flowcount

import (
	"fmt"
	"gateway/middleware/router/tcp"
	"sync/atomic"
	"time"
)

type FlowCountService struct {
	AppID       string        // 应用ID
	Interval    time.Duration // 采集频率
	TotalCount  int64         // 当前总共请求数
	QPS         int64         // QPS
	Unix        int64         // 上次unix时间戳
	TickerCount int64         // 当前流量
}

func NewFlowCountService(appID string, interval time.Duration) (*FlowCountService, error) {
	reqCounter := &FlowCountService{
		AppID:       appID,
		Interval:    interval,
		QPS:         0,
		Unix:        0,
		TickerCount: 0,
	}
	go func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println(err)
			}
		}()
		ticker := time.NewTicker(interval)
		for {
			<-ticker.C
			// 原子操作
			tickerCount := atomic.LoadInt64(&reqCounter.TickerCount)
			atomic.StoreInt64(&reqCounter.TickerCount, 0)
			nowUnix := time.Now().Unix()
			if reqCounter.Unix == 0 {
				reqCounter.Unix = time.Now().Unix()
				continue
			}
			if nowUnix > reqCounter.Unix {
				reqCounter.QPS = tickerCount / (nowUnix - reqCounter.Unix)
				reqCounter.TotalCount = reqCounter.TotalCount + tickerCount
				reqCounter.Unix = time.Now().Unix()
			}
		}
	}()
	return reqCounter, nil
}

// Increase 原子增加
func (o *FlowCountService) Increase() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println(err)
			}
		}()
		atomic.AddInt64(&o.TickerCount, 1)
	}()
}

func FlowCountMiddleWare(counter *FlowCountService) func(c *tcp.TcpSliceRouteContext) {
	return func(c *tcp.TcpSliceRouteContext) {
		counter.Increase()
		fmt.Println("QPS:", counter.QPS)
		fmt.Println("TotalCount:", counter.TotalCount)
		c.Next()
	}
}
