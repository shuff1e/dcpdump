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
	localIP       string
	remoteIP	  string
	snapshotLen   int
	printInterval int
	topN          int
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
	flag.IntVar(&options.printInterval, "printInterval", 120,
		"the interval to pop the metrics")
	flag.IntVar(&options.topN, "topN", 10,
		"top n most time spent operation info to show")
	flag.BoolVar(&options.printAll, "printAll", true,
		"whether to print all the info")
	flag.IntVar(&options.timeout, "timeout", 10,
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

var (
	promiscuous bool = false
	timeout     time.Duration = 30 * time.Second
	reqChan     = make(chan MCReqAndTime)
	respChan    = make(chan MCRespAndTime)
	httpChan    = make(chan string, 10)
)

func init() {
	argParse()
}

func main() {
	// Open device
	network,err := FindInterface(options.localIP)
	if err != nil {
		panic(err)
	}
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
	/* go httpPrint() */

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		dispatch(packet)
	}
}

func dispatch(packet gopacket.Packet) {

	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer != nil {
		ip, _ := ipLayer.(*layers.IPv4)

		tcpLayer := packet.Layer(layers.LayerTypeTCP)
		if tcpLayer != nil {
			tcp, _ := tcpLayer.(*layers.TCP)

			applicationLayer := packet.ApplicationLayer()
			if applicationLayer != nil {
				payload := applicationLayer.Payload()
				r := bytes.NewReader(payload)
				switch payload[0] {
				case 128:
					rv := gomemcached.MCRequest{}
					_, err := func() (n int,err error) {
						defer func() {
							if tmperr := recover(); tmperr != nil {
								n = -1
								err = fmt.Errorf(fmt.Sprintf("%#v",tmperr))
							}
						}()
						n,err = rv.Receive(r, nil)
						return
					}()
					if err != nil {
						//fmt.Println("Error decoding some part of the packet:", err)
					} else {
						reqChan <- MCReqAndTime{rv, packet.Metadata().CaptureInfo.Timestamp, ip.SrcIP, tcp.SrcPort, ip.DstIP, tcp.DstPort}
					}
				case 129:
					rv := gomemcached.MCResponse{}
					_, err := func() (n int ,err error) {
						defer func() {
							if tmperr := recover(); tmperr != nil {
								n = -1
								err = fmt.Errorf(fmt.Sprintf("%#v",tmperr))
							}
						}()
						n,err = rv.Receive(r, nil)
						return
					}()
					if err != nil {
						//fmt.Println("Error decoding some part of the packet:", err)
					} else {
						respChan <- MCRespAndTime{rv, packet.Metadata().CaptureInfo.Timestamp, ip.SrcIP, tcp.SrcPort, ip.DstIP, tcp.DstPort}
					}
				default:
					/* fmt.Printf("%s\n", payload) */
				}
			}
		}
	}

}
