# dcpdump
dcpdump is used to analyse a couchbase dcp request and response from network's perspective, aka packet capture and parse the contents of the packet.

Build

  **go build**

Capture all dcp packet

  **./dcpdump -network=bond0.107 -snapshotLen=1024 -printAll=true -printInterval=100 -timeout=0 -mode=client**
  
Cpature dcp packet from or to one specific host

  **./dcpdump -network=bond0.107 -snapshotLen=1024 -printAll=true -printInterval=100 -timeout=0 -mode=server --server=127.0.0.1**
  
  # dcpdump
dcpdump 用于从网卡层面分析couchbase的dcp协议中的请求和响应的信息, 即抓包和解析数据包的内容。

Build

  **go build**

Capture all dcp packet

  **./dcpdump -network=bond0.107 -snapshotLen=1024 -printAll=true -printInterval=100 -timeout=0 -mode=client**
  
Cpature dcp packet from or to one specific host

  **./dcpdump -network=bond0.107 -snapshotLen=1024 -printAll=true -printInterval=100 -timeout=0 -mode=server --server=127.0.0.1**
