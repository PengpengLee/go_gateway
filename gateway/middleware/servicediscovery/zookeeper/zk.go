package zookeeper

import (
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"time"
)

// ZkManager zookeeper管理器
// 	提供如下功能：
// 	1.连接的建立与关闭
// 	2.加载与更新配置
// 	3.创建临时节点
// 	4.监听与通知
type ZkManager struct {
	hosts      []string // zk主机列表，支持同时维护多个zk服务器
	conn       *zk.Conn // zookeeper连接，底层是 net.Conn
	pathPrefix string   // 路径前缀，默认为：/gateway_servers_
}

// NewZkManager 新建 zookeeper管理器
// 	封装指定主机列表
func NewZkManager(hosts []string) *ZkManager {
	return &ZkManager{hosts: hosts, pathPrefix: "/gateway_servers_"}
}

// GetConnect 连接zk服务器
func (z *ZkManager) GetConnect() error {
	conn, _, err := zk.Connect(z.hosts, 5*time.Second)
	if err != nil {
		return err
	}
	z.conn = conn
	return nil
}

// Close 关闭服务
func (z *ZkManager) Close() {
	z.conn.Close()
	return
}

// GetPathData 获取配置
func (z *ZkManager) GetPathData(nodePath string) ([]byte, *zk.Stat, error) {
	return z.conn.Get(nodePath)
}

// SetPathData 更新配置
func (z *ZkManager) SetPathData(nodePath string, config []byte, version int32) (err error) {
	ex, _, _ := z.conn.Exists(nodePath)
	if !ex {
		z.conn.Create(nodePath, config, 0, zk.WorldACL(zk.PermAll))
		return nil
	}
	_, dStat, err := z.GetPathData(nodePath)
	if err != nil {
		return
	}
	_, err = z.conn.Set(nodePath, config, dStat.Version)
	if err != nil {
		fmt.Println("Update node error", err)
		return err
	}
	fmt.Println("SetData ok")
	return
}

// RegisterServerPath 创建临时节点
// 节点注册流程：
// 	1.创建主节点：持久节点
// 	2.创建子节点：临时节点
func (z *ZkManager) RegisterServerPath(nodePath, host string) (err error) {
	// 先检查指定路径的节点是否存在
	ex, _, err := z.conn.Exists(nodePath)
	if err != nil {
		fmt.Println("Exists error", nodePath)
		return err
	}
	// 不存在
	if !ex {
		// 创建主节点，即服务节点：持久化节点，否则断开连接所有下级节点被清理
		//	zk.FlagEphemeral = 0:永久，除非手动删除
		//	zk.FlagEphemeral = 1:短暂，session断开则改节点也被删除，session维持时间为zk.Connect的第二个参数
		//	zk.FlagSequence  = 2:会自动在节点后面添加序号
		//	zk.FlagEphemeral = 3:Ephemeral和Sequence，即，短暂且自动添加序号
		_, err = z.conn.Create(nodePath, nil, 0, zk.WorldACL(zk.PermAll))
		if err != nil {
			fmt.Println("Create error", nodePath)
			return err
		}
	}
	// 子节点：临时节点
	subNodePath := nodePath + "/" + host
	ex, _, err = z.conn.Exists(subNodePath)
	if err != nil {
		fmt.Println("Exists error", subNodePath)
		return err
	}
	if !ex {
		_, err = z.conn.Create(subNodePath, nil, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
		if err != nil {
			fmt.Println("Create error", subNodePath)
			return err
		}
	}
	return
}

// GetServerListByPath 获取服务列表
func (z *ZkManager) GetServerListByPath(path string) (list []string, err error) {
	list, _, err = z.conn.Children(path)
	return
}

// WatchServerListByPath watch机制，监听子节点变化
// 服务器有断开或者重连，收到消息
func (z *ZkManager) WatchServerListByPath(path string) (chan []string, chan error) {
	// zookeeper连接
	conn := z.conn
	// 快照，节点变化收到通知
	snapshots := make(chan []string)
	// 错误信息
	errors := make(chan error)
	go func() {
		for {
			// snapshot：子节点内容
			// events: 绑定到path的事件
			snapshot, _, events, err := conn.ChildrenW(path)
			if err != nil {
				errors <- err
			}
			snapshots <- snapshot
			// 监听错误信息
			select {
			case evt := <-events:
				if evt.Err != nil {
					errors <- evt.Err
				}
				fmt.Printf("ChildrenW Event Path:%v, Type:%v\n", evt.Path, evt.Type)
			}
		}
	}()

	return snapshots, errors
}

// WatchPathData watch机制，监听节点值变化
func (z *ZkManager) WatchPathData(nodePath string) (chan []byte, chan error) {
	conn := z.conn
	snapshots := make(chan []byte)
	errors := make(chan error)

	go func() {
		for {
			dataBuf, _, events, err := conn.GetW(nodePath)
			if err != nil {
				errors <- err
				return
			}
			snapshots <- dataBuf
			select {
			case evt := <-events:
				if evt.Err != nil {
					errors <- evt.Err
					return
				}
				fmt.Printf("GetW Event Path:%v, Type:%v\n", evt.Path, evt.Type)
			}
		}
	}()
	return snapshots, errors
}
