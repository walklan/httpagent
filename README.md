采集机http代理
====

采集机http代理服务端, 接收http的get或post请求, 实现snmp采集和批量ping测试, 返回json格式数据

启动方式
----

```
./service-start.sh
```

使用方法
----

* snmp采集get方式
```
http://127.0.0.1:1216/snmpagent?seq=1111&ip=127.0.0.1&version=v2c&community=public&oids=get:.1.3.6.1.2.1.1.2.0!table:.1.3.6.1.2.1.31.1.1.1.1,.1.3.6.1.2.1.31.1.1.1.10
```

* ping测试get方式, 最大同时ping100个地址

```
http://127.0.0.1:1216/pingagent?seq=1111&ip=192.168.1.1,192.168.1.2,192.168.1.3
```

* 返回结果示例

    * snmp
    ```
    {
      "Data": [
        {
          "Index": ".1.3.6.1.2.1.31.1.1.1.10.1",
          "Value": "2441856676",
          "Error": ""
        }
      ],
      "Starttime": "20170727102151.481",
      "Endtime": "20170727102151.494",
      "Error": ""
    }
    ```
    
    * ping
    ```
    {
      "Data": [
        {
          "IP": "192.168.6.2",
          "Status": 0,
          "Lag": "-1"
        },
        {
          "IP": "192.168.6.1",
          "Status": 1,
          "Lag": "1.820279ms"
        }
      ],
      "Starttime": "20170727102426.109",
      "Endtime": "20170727102427.114",
      "Error": ""
    }
    ```
    
配置文件httpagent.yml
----

代理服务端配置参数详解

```
snmp:
    - asyncnum: 10           # snmp并发数
    - timeout: 2             # 系统级超时, 支持请求级超时
    - retry: 1               # 系统级重试此时, 支持请求级重试
    - maxsesspool: 1000      # 最大支持session数
    - maxlifetime: 30        # session缓存最大保存时间
http:
    - port: 1216             # http服务监听端口
log:
    - debug: false           # debug
    - logarchsize: 10485760  # 日志归档大小, 单位byte
```
