package main

import (
	"fmt"
	"github.com/couchbase/gomemcached"
	"github.com/rcrowley/go-metrics"
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
	serverHisto := make(map[string]metrics.Histogram)
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
				if histo, ok := serverHisto[req.dstIP.String()]; ok {
					histo.Update(spentTime.Nanoseconds() / 1000)
				} else {
					s := metrics.NewExpDecaySample(1024, 0.015)
                    h := metrics.NewHistogram(s)
                    h.Update(spentTime.Nanoseconds() / 1000)
					serverHisto[req.dstIP.String()] = h
				}
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
			fmt.Printf("------------------%v timeout in %v, %.4f%% below %v ms-------------------------------\n\n", timeoutNumber, allNumber, float64(allNumber-timeoutNumber)/float64(allNumber)*100, options.timeout)
			for i, x := range serverData {
				fmt.Printf("%v timeout at server %s\n", x, i)
				h := serverHisto[i]
				ps := h.Percentiles([]float64{0.5, 0.75, 0.95, 0.99})
				fmt.Printf("min = %.4f ms\n", float64(h.Min())/1000)
				fmt.Printf("max = %.4f ms\n", float64(h.Max())/1000)
				fmt.Printf("mean = %.4f ms\n", h.Mean()/1000)
				fmt.Printf("%%50 <= %.4f ms\n", ps[0]/1000)
				fmt.Printf("%%75 <= %.4f ms\n", ps[1]/1000)
				fmt.Printf("%%95 <= %.4f ms\n", ps[2]/1000)
				fmt.Printf("%%99 <= %v ms\n", ps[3]/1000)
				fmt.Println()
				/* delete(serverData, i) */
				/* h.Clear() */
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
