package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/couchbase/gomemcached"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"log"
	"os"
	"runtime/debug"
	"time"
)

var options struct {
	localIP       string
	remoteIP	  string
	snapshotLen   int
	printInterval int
	printAll      bool
	timeout       int
	mode          string
}

func argParse() {
	flag.StringVar(&options.localIP, "localIP", "",
		"the ip of the machine which dcpdump is running")
	flag.StringVar(&options.remoteIP, "remoteIP", "",
		"the ip which is interacting with this machine")
	flag.IntVar(&options.snapshotLen, "snapshotLen", 1024,
		"package will be cut if more than snapshotLen")
	flag.IntVar(&options.printInterval, "printInterval", 60,
		"the interval to pop the metrics")
	flag.BoolVar(&options.printAll, "printAll", true,
		"whether to print all the info")
	flag.IntVar(&options.timeout, "timeout", 0,
		"timeout setting, in milliseconds")
	flag.StringVar(&options.mode, "mode", "client",
		"run at server or client")
	flag.Usage = usage
	flag.Parse()
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage : %s [OPTIONS] \n", os.Args[0])
	flag.PrintDefaults()
}

func init() {
	argParse()
}

var (
	promiscuous bool = false
	timeout     time.Duration = 30 * time.Second
	reqChan     = make(chan MCReqAndTime)
	respChan    = make(chan MCRespAndTime)
)

func main() {
	// Find device
	network,err := FindInterface(options.localIP)
	if err != nil {
		panic(err)
	}
	// Open device
	handle, err := pcap.OpenLive(network, int32(options.snapshotLen), promiscuous, timeout)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	// Set filter
	var filter string
	if options.remoteIP != "" {
		filter = fmt.Sprintf("port 11210 and host %s", options.remoteIP)
	} else {
		filter = fmt.Sprintf("port 11210")
	}
	err = handle.SetBPFFilter(filter)
	if err != nil {
		log.Fatal(err)
	}
	// analyse the couchbase dcp packet
	go analyse()

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		dispatch(packet)
	}
}

func dispatch(packet gopacket.Packet) {
	defer func() {
		if err := recover(); err != nil {
			debug.PrintStack()
		}
	}()
	// ip
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		return
	}
	ip, _ := ipLayer.(*layers.IPv4)

	// tcp
	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	if tcpLayer == nil {
		return
	}
	tcp, _ := tcpLayer.(*layers.TCP)

	// application
	applicationLayer := packet.ApplicationLayer()
	if applicationLayer == nil {
		return
	}
	payload := applicationLayer.Payload()

	r := bytes.NewReader(payload)
	switch payload[0] {
	case 128:
		rv := gomemcached.MCRequest{}
		_, err := rv.Receive(r, nil)
		if err == nil {
			reqChan <- MCReqAndTime{rv, packet.Metadata().CaptureInfo.Timestamp, ip.SrcIP, tcp.SrcPort, ip.DstIP, tcp.DstPort}
		} else {
			//fmt.Println("Error decoding some part of the packet:", err)
		}
	case 129:
		rv := gomemcached.MCResponse{}
		_, err := rv.Receive(r, nil)
		if err == nil {
			respChan <- MCRespAndTime{rv, packet.Metadata().CaptureInfo.Timestamp, ip.SrcIP, tcp.SrcPort, ip.DstIP, tcp.DstPort}
		} else {
			//fmt.Println("Error decoding some part of the packet:", err)
		}
	default:
		/* fmt.Printf("%s\n", payload) */
	}
}
