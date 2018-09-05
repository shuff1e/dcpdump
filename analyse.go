package main

import (
	"fmt"
	"github.com/couchbase/gomemcached"
	"net"
	"sort"
	"time"
)

type MCReqAndTime struct {
	request gomemcached.MCRequest
	reqTime time.Time
	srcIP   net.IP
	dstIP   net.IP
}

type MCRespAndTime struct {
	response gomemcached.MCResponse
	respTime time.Time
}

type reqAndTime struct {
	MCReqAndTime
	spentTime time.Duration
}

var (
	data        opeHeap
	initialTime time.Time
)

func analyse() {
	ticker := time.NewTicker(time.Duration(options.analysisInterval) * time.Second)
	rawTicker := time.NewTicker(120 * time.Second)
	rawData := make(map[uint32]MCReqAndTime)
	serverData := make(map[string]uint32)
	var allNumber int
	var timeoutNumber int
	for {
		select {
		case req := <-reqChan:
			if _, ok := rawData[req.request.Opaque]; !ok {
				rawData[req.request.Opaque] = req
			}
		case resp := <-respChan:
			if req, ok := rawData[resp.response.Opaque]; ok {
				if options.printAll {
					fmt.Printf("Operation %s, key %s, sent from %15s to %15s at %s, received at %s\n", req.request.Opcode, string(req.request.Key), req.srcIP, req.dstIP, req.reqTime, resp.respTime)
				}
				allNumber++
				spentTime := resp.respTime.Sub(req.reqTime)
				if spentTime > time.Duration(time.Duration(options.timeout)*time.Millisecond) {
					timeoutNumber++
					serverData[req.dstIP.String()]++
				}
				delete(rawData, resp.response.Opaque)
				Push(data, reqAndTime{req, spentTime})
			}
		case <-ticker.C:
			fmt.Printf("\n\n------------------top %v request with the longest response time-----------------------------------\n", options.topN)
			sort.Sort(data)
			for i, x := range data {
				if x.reqTime != initialTime {
					fmt.Printf("Operation %s, key %s, sent from %15s to %15s at %s, spent %s\n", x.request.Opcode, string(x.request.Key), x.srcIP, x.dstIP, x.reqTime, x.spentTime)
					data[i] = reqAndTime{}
				}
			}
			fmt.Printf("\n")
			fmt.Printf("------------------%v timeout in %v, %.4f%% below %v ms-------------------------------\n", timeoutNumber, allNumber, float64(allNumber-timeoutNumber)/float64(allNumber)*100, options.timeout)
			for i, x := range serverData {
				fmt.Printf("%v timeout at server %s\n", x, i)
				delete(serverData, i)
			}
			allNumber = 0
			timeoutNumber = 0
		case <-rawTicker.C:
			for k, v := range rawData {
				if time.Since(v.reqTime) > time.Duration(60*time.Second) {
					delete(rawData, k)
				}
			}
		}
	}
}
