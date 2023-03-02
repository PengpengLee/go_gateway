package loadbalance

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"strconv"
	"testing"
)

func TestRoundRobin(t *testing.T) {
	rb := &RoundRobinBalance{}
	rb.Add("127.0.0.1:8001") //0
	rb.Add("127.0.0.1:8002") //1
	rb.Add("127.0.0.1:8003") //2
	rb.Add("127.0.0.1:8004") //3
	rb.Add("127.0.0.1:8005") //4

	for i := 0; i < 10; i++ {
		fmt.Println(rb.Next())
	}
}

func TestRandom(t *testing.T) {
	rb := &RandomBalance{}
	rb.Add("127.0.0.1:8001") //0
	rb.Add("127.0.0.1:8002") //1
	rb.Add("127.0.0.1:8003") //2
	rb.Add("127.0.0.1:8004") //3
	rb.Add("127.0.0.1:8005") //4

	for i := 0; i < 100; i++ {
		fmt.Println(rb.Next())
	}
}

func TestWeightRoundRobinBalance(t *testing.T) {
	rb := &WeightRoundRobinBalance{}
	// 9, 4, 2
	rb.Add("127.0.0.1:8001", "6") //0
	rb.Add("127.0.0.1:8002", "3") //1
	rb.Add("127.0.0.1:8003", "1") //2

	print(rb, "")
	fmt.Println("--------- init over --------")
	for i := 0; i < 15; i++ {
		addr, err := rb.Next()
		assert.Nil(t, err)

		var r = rand.Intn(2) // 0, 1
		if r == 1 {          // 出现 1 的概率是 1/2
			fmt.Println("server " + addr + " has failed.")
			rb.Callback(addr, false)
		} else {
			fmt.Println("正常访问：" + addr)
			rb.Callback(addr, true)
		}
		print(rb, addr)
	}

}

func print(rb *WeightRoundRobinBalance, addr string) {
	fmt.Println("主机地址\t\t\t当前权重\t有效权重")
	// 打印所有服务器当前权重
	total := 0
	for j := 0; j < len(rb.servAddrs); j++ {
		w := rb.servAddrs[j]
		total += w.effectiveWeight
		cw := strconv.Itoa(w.currentWeight)
		ew := strconv.Itoa(w.effectiveWeight)
		if w.addr == addr {
			// 被选中的服务器，高亮显示
			// 0x1B定义颜色的开始和结束标记
			// 1代表高亮，0代表无背景色，31代表红色前景，0代表恢复默认颜色
			fmt.Printf("%c[1;0;31m%s%c[0m", 0x1B, addr, 0x1B)
		} else {
			fmt.Print(w.addr)
		}
		var str = "\t\t" + cw + "\t\t" + ew + "\t\t"
		fmt.Println(str)
	}
	fmt.Println("有效权重之和：\t\t\t\t" + strconv.Itoa(total))
}

func TestConsistentHashBalance(t *testing.T) {
	rb := NewConsistentHashBalance(2, nil)
	rb.Add("127.0.0.1:8003", "127.0.0.1:8004", "127.0.0.1:8005", "127.0.0.1:8006", "127.0.0.1:8007")
	//fmt.Println(rb.hashKeys)
	fmt.Println(rb.hashMap)

	funcName(rb)

	key := rb.hashKeys[9]
	rb.hashKeys = append(rb.hashKeys[:9], rb.hashKeys[10:]...)
	delete(rb.hashMap, key)
	funcName(rb)
}

func funcName(rb *ConsistentHashBalance) {
	fmt.Println("--------------")
	// URL
	fmt.Println(rb.Get("http://127.0.0.1:8002/demo/get"))
	fmt.Println(rb.Get("http://127.0.0.1:8003/demo/getDemo"))
	fmt.Println(rb.Get("http://127.0.0.1:8002/demo/get"))
	fmt.Println(rb.Get("http://127.0.0.1:8004/demo/getBalance"))

	fmt.Println("--------------")
	// IP
	fmt.Println(rb.Get("127.0.0.1:8005"))
	fmt.Println(rb.Get("192.168.1.1:8007"))
	fmt.Println(rb.Get("127.0.0.1:8005"))
}
