package zookeeper

import (
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

// zookeeper提供了 分布式 发布/订阅 功能，
// 定义了一对多的订阅关系：一个主体对象对应多个订阅者
// 订阅者监听该主题对象，若其状态变化则通知所有订阅者
//
// 客户端向服务器注册一个 Watcher监听，当服务端一些事件（节点创建、
// 删除、子节点改变等）触发了这个 Watcher，就会向指定客户端发送
// 一个通知，客户端回调 Watcher得到触发事件情况。
//
// 一次性触发：一个 Watcher Event被发送到设置监听的客户端，
// 这种效果是一次性的，再次发生该事件不会再次触发。
// 先注册再触发：Watch机制，必须客户端先去服务端注册监听，这样
// 事件发生后才会触发监听，通知给客户端。

// 测试 zookeeper增删改查 API
func TestZkCRUD(t *testing.T) {
	// zookeeper服务地址，默认端口号2181
	var host = []string{"192.168.154.132:2181"}
	conn, _, err := zk.Connect(host, 5*time.Second)
	if err != nil {
		panic(err)
	}

	// 增
	// 	create [-s][-e] path data acl
	//	-s[Sequence==1]是否序列化
	// 	-e[Ephemeral==2]是否临时节点
	//	data数据
	//	acl节点权限
	if _, err := conn.Create("/test_tree2", []byte("tree_content"),
		0, zk.WorldACL(zk.PermAll)); err != nil {
		fmt.Println("create err", err)
	}

	// 查
	// 	ls：查询指定节点的一级子节点
	//	get：获取指定目录的数据和属性
	//	ls2：不能获取数据，可以获取子目录和属性
	nodeValue, dStat, err := conn.Get("/test_tree2")
	if err != nil {
		fmt.Println("get err", err)
		return
	}
	fmt.Println("nodeValue", string(nodeValue))

	// 改
	// set path data [version]
	//	data是数据，version的数据版本号，必须是当前版本号
	// 对节点增加限制：
	// setquota -n|-b val path
	//	n：子节点最大个数
	// 	b：数据值最大长度
	//	val：子节点最大个数或数据值最大长度
	// 	path：节点路径
	// 查看限制：
	// 	listquota path 列出指定节点的quota
	// 删除限制：
	// 	delquota [-n|-b] path 删除quota
	if _, err := conn.Set("/test_tree2", []byte("new_content"),
		dStat.Version); err != nil {
		fmt.Println("update err", err)
	}

	// 删除
	// delete path [version]
	// 	若删除时存在子节点，则无法删除。必须先删除子节点再删父节点
	// deleteall path 递归删除节点
	_, dStat, _ = conn.Get("/test_tree2")
	if err := conn.Delete("/test_tree2", dStat.Version); err != nil {
		fmt.Println("Delete err", err)
		//return
	}

	// 验证存在
	hasNode, _, err := conn.Exists("/test_tree2")
	if err != nil {
		fmt.Println("Exists err", err)
		//return
	}
	fmt.Println("node Exist", hasNode)

	// 增加
	if _, err := conn.Create("/test_tree2", []byte("tree_content"),
		0, zk.WorldACL(zk.PermAll)); err != nil {
		fmt.Println("create err", err)
	}

	// 设置子节点
	if _, err := conn.Create("/test_tree2/subnode", []byte("node_content"),
		0, zk.WorldACL(zk.PermAll)); err != nil {
		fmt.Println("create err", err)
	}

	// 获取子节点列表
	childNodes, _, err := conn.Children("/test_tree2")
	if err != nil {
		fmt.Println("Children err", err)
	}
	fmt.Println("childNodes", childNodes)
}

func TestRegister(t *testing.T) {
	zkManager := NewZkManager([]string{"192.168.154.132:2181"})
	zkManager.GetConnect()
	defer zkManager.Close()
	i := 0
	for {
		// 注册的内容交付给zk服务器
		zkManager.RegisterServerPath("/realserver", fmt.Sprint(i))
		fmt.Println("Register", i)
		time.Sleep(5 * time.Second)
		i++
	}
}

// 监听节点变化
//
// zookeeper 的 watcher 机制，可以分为四个过程：
//	1.客户端注册 watcher
//	2.服务端处理 watcher
//	3.服务端触发 watcher 事件
//		前提：可能是错误事件；可能是其它客户端的修改事件
//	4.客户端回调 watcher
func TestWatch(t *testing.T) {
	// 获取zk节点列表：192.168.154.132是zk服务器地址
	zkManager := NewZkManager([]string{"192.168.154.132:2181"})
	zkManager.GetConnect()
	defer zkManager.Close()

	// 获取服务器列表
	zList, err := zkManager.GetServerListByPath("/realserver")
	fmt.Println("server node:")
	fmt.Println(zList)
	if err != nil {
		log.Println(err)
	}

	// 动态监听节点变化：这两个 channel的数据由zk服务器提供
	chanList, chanErr := zkManager.WatchServerListByPath("/realserver")
	go func() {
		for {
			select {
			case changeErr := <-chanErr:
				fmt.Println("changeErr")
				fmt.Println(changeErr)
			case changedList := <-chanList:
				fmt.Println("watch node changed")
				fmt.Println(changedList)
			}
		}
	}()

	// 关闭信号监听
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}

func TestWrite(t *testing.T) {
	zkManager := NewZkManager([]string{"192.168.154.132:2181"})
	zkManager.GetConnect()
	defer zkManager.Close()
	i := 0

	for {
		conf := fmt.Sprintf("{name:" + fmt.Sprint(i) + "}")
		zkManager.SetPathData("/rs_server_conf", []byte(conf), int32(i))
		time.Sleep(5 * time.Second)
		i++
	}
}
