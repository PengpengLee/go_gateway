package loadbalance

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// 最大失败次数
	maxFails = 3
	// 失败次数最多之后，不能访问的超时时间
	failTimeout = time.Second * 3
)

type WeightRoundRobinBalance struct {
	// 服务器主机地址 host:port
	servAddrs []*node
	// 当前轮询的节点索引
	curIndex int

	// 观察主体
	conf LoadBalanceConf
}

// node 每个服务器节点有不同的权重，并且在每一轮访问后可能会发生变化
//
// weight：初始化权重，仅作记录，不修改
// currentWeight：本次轮询的权重，依次判断最大值并返回
// effectiveWeight：有效权重，服务器宕机、故障、延时，没发生一次 -1
//		此属性用于计算 currentWeight，可能导致该服务器被移除
type node struct {
	// 主机地址：host:port
	addr string
	// 初始化权重
	weight int
	// 节点当前/临时权重，每一轮都可能变化
	// currentWeight = currentWeight + effectiveWeight
	// 每一轮都选择当前权重最大的节点
	currentWeight int
	// 有效权重，默认与weight相同
	// 用于移除故障节点，每次故障 -1
	effectiveWeight int

	// 在 failTimeout 时间内，最大的失败次数
	// 达到最大失败次数时，failTimeout 时间内（后）不能再次选择该服务器
	maxFails int
	// 指定的超时时间（用于衡量最大失败次数，也用于超时计算）
	// 单位是秒，默认是10秒
	failTimeout time.Duration
	// 失败时间点，按时间正序排列，只保留 failTimeout 时间内的记录
	failTimes []time.Time
}

// Add 添加带权重的服务器主机
//
// 格式："host:port", "weight", "host:port", "weight", "host:port", "weight"...
func (r *WeightRoundRobinBalance) Add(params ...string) error {
	length := len(params)
	if length == 0 || length%2 != 0 {
		return errors.New("params' length must be 2 or multiple of 2")
	}
	for i := 0; i < length; i += 2 {
		addr := params[i]
		weight, err := strconv.ParseInt(params[i+1], 10, 32)
		if err != nil {
			return err
		}
		// 默认权重为1
		if weight <= 0 {
			weight = 1
		}
		n := &node{
			addr:            addr,
			weight:          int(weight),
			effectiveWeight: int(weight),
			maxFails:        maxFails,
			failTimeout:     failTimeout,
		}
		r.servAddrs = append(r.servAddrs, n)
	}
	return nil
}

// Next 获取下一个服务器地址：找到权重最大的服务器
// 为了避免每次都访问同一个服务器（权重最大），每一轮选中之后，需要对其进行降权操作
// 通过每轮降权，权值较大的服务器，被选中的次数较多，实现了按权重访问的逻辑
//
// 带权服务器轮询方式核心逻辑：
//	循环计算每个服务器的权值（currentWeight），选择最大的返回
// 	对选中的服务器进行降权：
//		currentWeight - 本轮所有有效权重之和
//
// 实现步骤：
// 	1.定义变量maxNode，记录本轮权值最大的服务器
// 	2.循环计算每个服务器的权重：临时权重 + 有效权重，选择最大的临时权重节点
// 	3.记录所有有效权重之和：effectiveTotal
// 	4.对选中节点进行降权
func (r *WeightRoundRobinBalance) Next() (string, error) {
	var index = 0
	// 所有节点的有效权重之和（作为降权参数）
	var effectiveTotal = 0
	// 	1.定义变量 maxNode，记录本轮权值最大的服务器
	var maxNode *node
	// 	2.循环计算每个服务器的权重：临时权重 + 有效权重，选择最大的临时权重节点
	for i := 0; i < len(r.servAddrs); i++ {
		w := r.servAddrs[i]

		// 检查小黑屋
		if w.maxFails <= 0 {
			// 刷新错误记录
			refreshErrRecords(w)
			w.maxFails = maxFails - len(w.failTimes)
			if w.maxFails <= 0 {
				//fmt.Println("小黑屋：" + w.addr)
				//time.Sleep(time.Second * 3)
				continue
			}
		}

		w.currentWeight += w.effectiveWeight
		if maxNode == nil || w.currentWeight > maxNode.currentWeight {
			maxNode = w
			index = i
		}
		// 	3.记录所有有效权重之和：effectiveTotal
		effectiveTotal += w.effectiveWeight
	}
	if maxNode == nil {
		// 服务器列表为空，返回error
		return "", errors.New("there is no server address. please call 'Add(...string)' first.")
	}
	// 	4.对选中节点进行降权
	maxNode.currentWeight -= effectiveTotal
	r.curIndex = index
	return maxNode.addr, nil
}

func (r *WeightRoundRobinBalance) Get(key string) (string, error) {
	return r.Next()
}

func (r *WeightRoundRobinBalance) SetConf(conf LoadBalanceConf) {
	r.conf = conf
}

func (r *WeightRoundRobinBalance) Update() {
	if conf, ok := r.conf.(*LoadBalanceZkConf); ok {
		fmt.Println("WeightRoundRobinBalance get conf:", conf.GetConf())
		r.servAddrs = nil
		for _, ip := range conf.GetConf() {
			r.Add(strings.Split(ip, ",")...)
		}
	}
	//if conf, ok := r.conf.(*LoadBalanceCheckConf); ok {
	//	fmt.Println("WeightRoundRobinBalance get conf:", conf.GetConf())
	//	r.servAddrs = nil
	//	for _, ip := range conf.GetConf() {
	//		r.Add(strings.Split(ip, ",")...)
	//	}
	//}
}

func (r *WeightRoundRobinBalance) Callback(addr string, flag bool) {
	for i := 0; i < len(r.servAddrs); i++ {
		w := r.servAddrs[i]
		if w.addr == addr {
			// 访问服务器成功
			if flag {
				// 有效权重默认与权重相同，通讯异常 -1，正常 +1，不能超过weight大小
				if w.effectiveWeight < w.weight {
					w.effectiveWeight++
				}
			} else {
				// 访问服务器失败
				w.effectiveWeight--

				// 刷新错误记录，把过去超过 failTimeout 的错误记录删除
				refreshErrRecords(w)

				// 记录本次失败
				w.failTimes = append(w.failTimes, time.Now())
				// 当前节点剩余失败次数，可能小于0
				w.maxFails = maxFails - len(w.failTimes)
			}
			break
		}
	}
}

// refreshErrRecords 刷新错误记录，把过去超过 failTimeout 的错误记录删除
// 过滤掉数组中超期的错误记录
// 数组中错误记录按从小到大排序，越靠前超期可能性越大
func refreshErrRecords(w *node) {
	now := time.Now()
	var i = 0
	for ; i < len(w.failTimes); i++ {
		// 错误记录 + failTimeout 之和，应该在当前系统时间之后
		failTime := w.failTimes[i].Add(failTimeout)
		// 移除 failTimeout 之前的错误记录
		if failTime.After(now) || failTime.Equal(now) {
			break
		}
	}
	// 从 i 开始截取，直到数组末尾，包含 i
	// i 代表数组中第一个合法记录
	w.failTimes = w.failTimes[i:]
}
