package loadbalance

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
)

type RandomBalance struct {
	// 服务器主机地址 host:port
	servAddrs []string
	// 当前轮询的节点索引
	curIndex int

	// 观察主体
	conf LoadBalanceConf
}

func (r *RandomBalance) Add(params ...string) error {
	if len(params) == 0 {
		return errors.New("params length at least 1")
	}
	for i := 0; i < len(params); i++ {
		r.servAddrs = append(r.servAddrs, params[i])
	}
	return nil
}

func (r *RandomBalance) Next() string {
	lens := len(r.servAddrs)
	if lens == 0 {
		return ""
	}

	r.curIndex = rand.Intn(len(r.servAddrs)) // 0, 1, 2
	return r.servAddrs[r.curIndex]
}

func (r *RandomBalance) Get(key string) (string, error) {
	return r.Next(), nil
}

func (r *RandomBalance) SetConf(conf LoadBalanceConf) {
	r.conf = conf
}

func (r *RandomBalance) Update() {
	if conf, ok := r.conf.(*LoadBalanceZkConf); ok {
		fmt.Println("Update get conf:", conf.GetConf())
		r.servAddrs = []string{}
		for _, ip := range conf.GetConf() {
			r.Add(strings.Split(ip, ",")...)
		}
	}
	//if conf, ok := r.conf.(*LoadBalanceCheckConf); ok {
	//	fmt.Println("Update get conf:", conf.GetConf())
	//	r.servAddrs = nil
	//	for _, ip := range conf.GetConf() {
	//		r.Add(strings.Split(ip, ",")...)
	//	}
	//}
}
