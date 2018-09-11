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
	"time"
)

var options struct {
	serverIP         string
	network          string
	snapshotLen      int
	printInterval int
	topN             int
	printAll         bool
	timeout          int
}

func argParse() {
	flag.StringVar(&options.serverIP, "server", "",
		"couchbase server to check")
	flag.StringVar(&options.network, "network", "eth0",
		"network used")
	flag.IntVar(&options.snapshotLen, "snapshotLen", 1024,
		"package will be cut if more than snapshotLen")
	flag.IntVar(&options.printInterval, "printInterval", 120,
		"the interval to pop the metrics")
	flag.IntVar(&options.topN, "topN", 10,
		"top n most time spent operation info to show")
	flag.BoolVar(&options.printAll, "printAll", true,
		"whether to print all the info")
	flag.IntVar(&options.timeout, "timeout", 10,
		"timeout setting, in milliseconds")
	flag.Usage = usage
	flag.Parse()
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage : %s [OPTIONS] \n", os.Args[0])
	flag.PrintDefaults()
}

var (
	promiscuous bool = false
	err         error
	timeout     time.Duration = 30 * time.Second
	handle      *pcap.Handle
	reqChan     = make(chan MCReqAndTime)
	respChan    = make(chan MCRespAndTime)
)

func init() {
	argParse()
	data = make(opeHeap, options.topN)
}

func main() {
	// Open device
	handle, err = pcap.OpenLive(options.network, int32(options.snapshotLen), promiscuous, timeout)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	// Set filter
	var filter string
	if options.serverIP != "" {
		filter = fmt.Sprintf("port 11210 and host %s", options.serverIP)
	} else {
		filter = fmt.Sprintf("port 11210")
	}
	err = handle.SetBPFFilter(filter)
	if err != nil {
		log.Fatal(err)
	}

	go analyse()

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		dispatch(packet)
	}
}

func dispatch(packet gopacket.Packet) {

	// Let's see if the packet is IP (even though the ether type told us)
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer != nil {
		ip, _ := ipLayer.(*layers.IPv4)

		// IP layer variables:
		// Version (Either 4 or 6)
		// IHL (IP Header Length in 32-bit words)
		// TOS, Length, Id, Flags, FragOffset, TTL, Protocol (TCP?),
		// Checksum, SrcIP, DstIP

		// When iterating through packet.Layers() above,
		// if it lists Payload layer then that is the same as
		// this applicationLayer. applicationLayer contains the payload
		applicationLayer := packet.ApplicationLayer()
		if applicationLayer != nil {
			payload := applicationLayer.Payload()
			r := bytes.NewReader(payload)
			switch payload[0] {
			case 128:
				rv := gomemcached.MCRequest{}
				_, err := rv.Receive(r, nil)
				if err != nil {
					fmt.Println("Error decoding some part of the packet:", err)
                } else {
                    reqChan <- MCReqAndTime{rv, packet.Metadata().CaptureInfo.Timestamp, ip.SrcIP, ip.DstIP}
                }
			case 129:
				rv := gomemcached.MCResponse{}
				_, err := rv.Receive(r, nil)
				if err != nil {
					fmt.Println("Error decoding some part of the packet:", err)
                } else {
                    respChan <- MCRespAndTime{rv, packet.Metadata().CaptureInfo.Timestamp, ip.SrcIP, ip.DstIP}
                }
			default:
				/* fmt.Printf("%s\n", payload) */
			}
		}
	}

}
