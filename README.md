官方客户端已经有tracing功能。

参考：[Response Time Observability with the Java SDK](https://blog.couchbase.com/response-time-observability-with-the-java-sdk/)

代码：[ConnectionBase.java](https://github.com/shuff1e/devguide-examples/blob/master/java/src/main/java/com/couchbase/devguide/ConnectionBase.java)

# dcpdump
# couchbase 抓包工具

dcpdump is used to analyse a couchbase dcp request and response from network's perspective, aka packet capture and parse the contents of the packet.

Build

  **go build**

Capture all dcp packet

  **./dcpdump -localIP=10.153.194.168 -snapshotLen=1024 -printAll=true -printInterval=100 -timeout=0 -mode=client**
  
Cpature dcp packet from or to one specific host

  **./dcpdump -localIp=10.153.194.168 -snapshotLen=1024 -printAll=true -printInterval=100 -timeout=0 -mode=server -remoteIP=10.153.194.188**

Options

  **localIP为运行dcpdump的机器的IP**
  
  **snapshotLen指从抓取的MAC帧中截取的长度(具体信息可以man tcpdump)**

  **printAll为是否输出超时响应的具体信息**

  **printInterval为隔一定时间(printInterval秒)，输出到目前为止响应时间的metrics信息**

  **timeout为超时时间**

  **mode为运行模式，在客户端机器上运行时，mode为client，服务端节点上运行时，mode为server**

  **remoteIP为与本机交互的其他机器的IP**