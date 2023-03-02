# 安装熔断器监控面板：hystrix-dashboard

- 一、安装docker环境
    ```
    （略）
    ```
    
- 二、安装hystrix-dashboard

    下载地址：
    ```
    https://github.com/mlabouardy/hystrix-dashboard-docker
    ```
  
- 执行下载操作：
    ```
    $git clone git@github.com:mlabouardy/hystrix-dashboard-docker.git
    ```

- 切换到下载目录：
    ```
    $cd hystrix-dashboard-docker
    ```

- 执行服务启动
    ```
    $docker run -d -p 8080:9002 --name hystrix-dashboard mlabouardy/hystrix-dashboard:latest
    ```

- 在浏览器访问监控面板hystrix-dashboard：
    ```
    http://{docker主机ip}:8080/hystrix
    ```

# 启动测试用例 

- 切换以下目录
    ```
    $cd gateway/middleware/circuitbreaker
    ```
- 在该地址下运行测试用例，你会启动一个Stream server
    ```
    $go test
    ```
- Stream server 地址： 
    ```
    http://{测试用例主机ip}:8070/
    ```
将此地址复制到docker监控页面的输入框，点击”Monitor Stream“即可。
