package loadbalance

type LbType int

const (
	LbRandom LbType = iota
	LbRoundRobin
	LbWeightRoundRobin
	LbConsistentHash
)

type LoadBalance interface {
	Add(...string) error
	Get(string) (string, error)

	SetConf(LoadBalanceConf)
	// Update 用于服务发现
	Update()
}

func LoadBalanceFactory(lbType LbType) LoadBalance {
	switch lbType {
	case LbRandom:
		return &RandomBalance{}
	case LbConsistentHash:
		return NewConsistentHashBalance(10, nil)
	case LbRoundRobin:
		return &RoundRobinBalance{}
	case LbWeightRoundRobin:
		return &WeightRoundRobinBalance{}
	default:
		return &RandomBalance{}
	}
}

func LoadBalanceFactoryWithConf(lbType LbType, mConf LoadBalanceConf) LoadBalance {
	var lb LoadBalance = nil
	switch lbType {
	case LbRandom:
		lb = &RandomBalance{}
		initLoadBalance(lb, mConf)
	case LbConsistentHash:
		lb = NewConsistentHashBalance(10, nil)
		initLoadBalance(lb, mConf)
	case LbRoundRobin:
		lb = &RoundRobinBalance{}
		initLoadBalance(lb, mConf)
	case LbWeightRoundRobin:
		lb = &WeightRoundRobinBalance{}
		initLoadBalance(lb, mConf)
	default:
		lb = &RandomBalance{}
		initLoadBalance(lb, mConf)
	}
	return lb
}

func initLoadBalance(lb LoadBalance, mConf LoadBalanceConf) {
	// 初始化配置信息（可用服务器列表）
	lb.SetConf(mConf)
	// 绑定观察者与被观察者
	mConf.Attach(lb)
	lb.Update()
}
