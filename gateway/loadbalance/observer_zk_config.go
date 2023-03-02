package loadbalance

import (
	"fmt"
	"gateway/middleware/servicediscovery/zookeeper"
)

// LoadBalanceConf 负载均衡配置（抽象主体）
type LoadBalanceConf interface {
	Attach(o Observer)
	GetConf() []string
	WatchConf()
	UpdateConf(conf []string)
}

// Observer 观察者接口（抽象观察者）
type Observer interface {
	Update()
}

// LoadBalanceZkConf 负载均衡配置（具体主体）
type LoadBalanceZkConf struct {
	observers    []Observer        // 观察者列表
	path         string            // zk的path地址
	zkHosts      []string          // zk的集群列表
	confIpWeight map[string]string // IP与权重的映射表：IP -> weight
	activeList   []string          // 可用主机列表

	format string // 格式化
}

// NewLoadBalanceZkConf 创建负载均衡zk配置实例
func NewLoadBalanceZkConf(format, path string, zkHosts []string, conf map[string]string) (*LoadBalanceZkConf, error) {
	zkManager := zookeeper.NewZkManager(zkHosts)
	zkManager.GetConnect()
	defer zkManager.Close()
	zList, err := zkManager.GetServerListByPath(path)
	if err != nil {
		return nil, err
	}
	// 创建具体主体
	mConf := &LoadBalanceZkConf{format: format, activeList: zList, confIpWeight: conf, zkHosts: zkHosts, path: path}
	// 启动监听
	mConf.WatchConf()
	return mConf, nil
}

// Attach 绑定到观察者列表
func (s *LoadBalanceZkConf) Attach(o Observer) {
	s.observers = append(s.observers, o)
}

// NotifyAllObservers 通知所有观察者
func (s *LoadBalanceZkConf) NotifyAllObservers() {
	for _, obs := range s.observers {
		obs.Update()
	}
}

// GetConf 获取服务器配置
func (s *LoadBalanceZkConf) GetConf() []string {
	confList := []string{}
	for _, ip := range s.activeList {
		weight, ok := s.confIpWeight[ip]
		if !ok {
			weight = "50" //默认weight
		}
		confList = append(confList, fmt.Sprintf(s.format, ip)+","+weight)
	}
	return confList
}

// WatchConf 监听当前节点的所有下级节点的变化
// 更新配置时，通知监听者也更新
func (s *LoadBalanceZkConf) WatchConf() {
	zkManager := zookeeper.NewZkManager(s.zkHosts)
	zkManager.GetConnect()
	chanList, chanErr := zkManager.WatchServerListByPath(s.path)
	go func() {
		defer zkManager.Close()
		for {
			select {
			case changeErr := <-chanErr:
				fmt.Println("changeErr", changeErr)
			case changedList := <-chanList:
				fmt.Println("watch node changed")
				s.UpdateConf(changedList)
			}
		}
	}()
}

// UpdateConf 更新配置时，通知监听者也更新
func (s *LoadBalanceZkConf) UpdateConf(conf []string) {
	s.activeList = conf
	for _, obs := range s.observers {
		obs.Update()
	}
}

// LoadBalanceObserver 观察者实现（具体观察者）
type LoadBalanceObserver struct {
	ZkConf *LoadBalanceZkConf
}

func (l *LoadBalanceObserver) Update() {
	fmt.Println("Update get conf:", l.ZkConf.GetConf())
}

func NewLoadBalanceObserver(conf *LoadBalanceZkConf) *LoadBalanceObserver {
	return &LoadBalanceObserver{
		ZkConf: conf,
	}
}
