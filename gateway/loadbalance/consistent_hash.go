package loadbalance

import (
	"errors"
	"hash/crc32"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// ConsistentHashBalance 一致性hash算法实现负载均衡。
// 用于解决简单哈希算法在增删节点时，重新映射带来的效率低下问题。
// 	一致性/单调性：以uint32范围作为哈希表，新增或者删减节点时，
//	  【不影响】系统正常运行，解决哈希表的动态伸缩问题
// 	分散性：数据应该分散地存放在（分布式集群中的）各个节点，不必每个节点都存储所有的数据
// 	平衡性：采用虚拟节点解决hash环偏斜问题。
//	  hash的结果应该平均分配到各个节点，从算法层面解决负载均衡问题
// 实现步骤：
// 	1.计算存储节点（服务器）哈希值，将其存储空间抽象成一个环（0 - 2^32 -1）
// 	2.对数据（URL、IP）进行哈希计算，按顺时针方向将其映射到距离最近的节点上
type ConsistentHashBalance struct {
	// hash 函数，支持用户自定义，默认使用 crc32.ChecksumIEEE
	// 1.运算效率；2.散列均匀
	hash Hash
	// 服务器节点 hash值列表，按照从小到大排序
	hashKeys UInt32Slice
	// 服务器节点 hash值与服务器真实地址的映射表
	// key - hashKeys[i]
	// value - addr
	hashMap map[uint32]string

	// 观察主体
	conf LoadBalanceConf

	// 虚拟节点倍数
	// 解决平衡性问题
	replicas int
	// 读写锁：扩容或宕机时，读写同步，确保并发安全
	mux sync.RWMutex
}

// Hash 函数，根据给定数据计算哈希值，返回一个32位无符号数
// 默认使用 crc32.ChecksumIEEE，支持用户自定义
type Hash func(data []byte) uint32

type UInt32Slice []uint32

func (s UInt32Slice) Len() int {
	return len(s)
}

func (s UInt32Slice) Less(i, j int) bool {
	return s[i] < s[j]
}

func (s UInt32Slice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func NewConsistentHashBalance(replicas int, fn Hash) *ConsistentHashBalance {
	ch := &ConsistentHashBalance{
		hash:     fn,
		replicas: replicas,
		hashMap:  make(map[uint32]string),
	}

	if ch.hash == nil {
		// crc: Cycle Redundancy Check 循环冗余检验
		// 返回一个32位无符号整数（0 - 2^32 -1）
		// 参数：data []byte
		ch.hash = crc32.ChecksumIEEE
	}
	return ch
}

// Add 添加服务器节点，参数为服务器地址 addr，比如使用URL, IP
// 允许传入 1 或 多个真实节点的名称，格式如：
//	"http://ip:port/demo", "IP:port", "IP:port"
//
// 对每一个真实节点 addr，对应创建 c.replicas 个虚拟节点
// 用 c.hash() 计算虚拟节点的哈希值，添加到环上（hashKeys）
// 最后，要对 c.hashKeys 进行排序
func (c *ConsistentHashBalance) Add(servers ...string) error {
	if len(servers) == 0 {
		return errors.New("servers length at least 1")
	}

	c.mux.Lock()
	defer c.mux.Unlock()
	for _, addr := range servers {
		for i := 0; i < c.replicas; i++ {
			hash := c.hash([]byte(strconv.Itoa(i) + addr))
			c.hashKeys = append(c.hashKeys, hash)
			c.hashMap[hash] = addr
		}
	}
	// 对所有节点的哈希值进行排序
	// c.hashKeys 数据类型必须实现 Interface 接口，完成方法重写
	sort.Sort(c.hashKeys)
	return nil
}

// Get 获取指定key最靠近它的那个服务器节点。
// 返回服务器节点的hash值 >= key的hash值（也可能穿过环起止点）
//
// 实现步骤：
// 	1.计算 key 的hash值
// 	2.通过二分查找最优服务器节点
// 	3.取出服务器地址，并返回
func (c *ConsistentHashBalance) Get(key string) (string, error) {
	l := len(c.hashKeys)
	if l == 0 {
		return "", errors.New("node list is empty")
	}
	// 	1.计算 key 的hash值
	hash := c.hash([]byte(key))
	//fmt.Print(strconv.FormatInt(int64(hash), 10) + ":")

	// 	2.通过二分查找最优服务器节点
	// 1 3 5 7 9
	index := sort.Search(l, func(i int) bool {
		return c.hashKeys[i] >= hash
	})
	// 环：查找结果大于服务器节点的最大索引
	// 说明此时该对象哈希值在最后一个节点之后，
	// 返回第一个节点
	if index == l {
		index = 0
	}

	// 	3.取出服务器地址，并返回
	// 读锁，允许多读，不允许写
	c.mux.RLock()
	defer c.mux.RUnlock()
	return c.hashMap[c.hashKeys[index]], nil
}

func (c *ConsistentHashBalance) SetConf(conf LoadBalanceConf) {
	c.conf = conf
}

func (c *ConsistentHashBalance) Update() {
	if conf, ok := c.conf.(*LoadBalanceZkConf); ok {
		//fmt.Println("Update get conf:", conf.GetConf())
		c.hashKeys = nil
		c.hashMap = nil
		for _, ip := range conf.GetConf() {
			c.Add(strings.Split(ip, ",")...)
		}
	}
	//if conf, ok := c.conf.(*LoadBalanceCheckConf); ok {
	//	//fmt.Println("Update get conf:", conf.GetConf())
	//	c.hashKeys = nil
	//	c.hashMap = map[uint32]string{}
	//	for _, ip := range conf.GetConf() {
	//		c.Add(strings.Split(ip, ",")...)
	//	}
	//}
}
