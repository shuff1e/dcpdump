# dcpdump
dcpdump is used to analyse a couchbase dcp request and response from network's perspective, aka packet capture and parse the contents of the packet.

>Build

  **go build**

>Capture all dcp packet

  **./dcpdump -network=eth0 -snapshotLen=1024 -analysisInterval=120 -topN=10 -printAll=false -timeout=10**
  
>Cpature dcp packet from or to one specific host

  **./dcpdump -network=eth0 -snapshotLen=1024 -analysisInterval=120 -topN=10 -printAll=false -timeout=10 -server=127.0.0.1**
