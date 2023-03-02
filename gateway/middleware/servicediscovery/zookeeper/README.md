# 安装zookeeper

- 参考官方文档安装
  - 官方介绍
    `https://zookeeper.apache.org/doc/current/index.html`
  - 下载地址
  `https://zookeeper.apache.org/releases.html`
  - 3.8.0
  https://dlcdn.apache.org/zookeeper/zookeeper-3.8.0/apache-zookeeper-3.8.0-bin.tar.gz
  - 3.7.1
    https://dlcdn.apache.org/zookeeper/zookeeper-3.7.1/apache-zookeeper-3.7.1-bin.tar.gz
  - 3.6.3
  https://dlcdn.apache.org/zookeeper/zookeeper-3.6.3/apache-zookeeper-3.6.3-bin.tar.gz
- 解压缩
- 编辑 conf/zoo.cfg（把zoo_sample.cfg重命名即可）
```
tickTime=2000
dataDir=/var/lib/zookeeper
clientPort=2181
```
- 运行 `bin/zkServer.sh start`

