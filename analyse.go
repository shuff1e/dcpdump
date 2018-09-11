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
	srcIP   net.IP
	dstIP   net.IP
}

type reqAndTime struct {
	MCReqAndTime
	spentTime time.Duration
}

var (
	data        opeHeap
	initialTime time.Time
)

type counterAndHisto struct {
    all metrics.Counter
    timeout metrics.Counter
    histo metrics.Histogram
}
func analyse() {
	ticker := time.NewTicker(time.Duration(options.printInterval) * time.Second)
    serverMetrics := make(map[string]counterAndHisto)
	rawTicker := time.NewTicker(120 * time.Second)
	rawData := make(map[string]MCReqAndTime)
	for {
		select {
		case req := <-reqChan:
			if _, ok := rawData[req.srcIP.String() + req.dstIP.String() + fmt.Sprintf("%d",req.request.Opaque)]; !ok {
				rawData[req.srcIP.String() + req.dstIP.String() + fmt.Sprintf("%d",req.request.Opaque)] = req
			}
		case resp := <-respChan:
			if req, ok := rawData[resp.dstIP.String() + resp.srcIP.String() + fmt.Sprintf("%d",resp.response.Opaque)]; ok {
				if options.printAll {
					fmt.Printf("%s, key %s, sent from %15s to %15s at %s, received at %s\n", req.request.Opcode, string(req.request.Key), req.srcIP, req.dstIP, req.reqTime, resp.respTime)
				}
				spentTime := resp.respTime.Sub(req.reqTime)
				if ch, ok := serverMetrics[req.dstIP.String()]; ok {
                    ch.all.Inc(1)
					ch.histo.Update(spentTime.Nanoseconds() / 1000)
				} else {
                    c1 := metrics.NewCounter()
                    c2 := metrics.NewCounter()
                    c1.Inc(1)
					s := metrics.NewExpDecaySample(1024, 0.015)
					/* s := metrics.NewUniformSample(1028) */
					h := metrics.NewHistogram(s)
					h.Update(spentTime.Nanoseconds() / 1000)
					serverMetrics[req.dstIP.String()] = counterAndHisto{c1,c2,h}
				}
				if spentTime > time.Duration(time.Duration(options.timeout)*time.Millisecond) {
					serverMetrics[req.dstIP.String()].timeout.Inc(1)
				}
				delete(rawData, req.srcIP.String() + req.dstIP.String() + fmt.Sprintf("%d",req.request.Opaque))
				Push(data, reqAndTime{req, spentTime})
			}
		case <-ticker.C:
			fmt.Printf("\n\n------------------top %v request with the longest response time-----------------------------------\n", options.topN)
			sort.Sort(data)
			for _, x := range data {
				if x.reqTime != initialTime {
					fmt.Printf("%s, key %s, sent from %15s to %15s at %s, spent %s\n", x.request.Opcode, string(x.request.Key), x.srcIP, x.dstIP, x.reqTime, x.spentTime)
					/* data[i] = reqAndTime{} */
				}
			}
            Heapify(data)
			fmt.Printf("\n")
			for i, x := range serverMetrics {
				fmt.Printf("metrics of server %s\n", i)
				fmt.Printf("%v timeout in %v\n", x.timeout.Count(), x.all.Count())
				ps := x.histo.Percentiles([]float64{0.5, 0.75, 0.95, 0.99})
				fmt.Printf("min = %.4f ms\n", float64(x.histo.Min())/1000)
				fmt.Printf("max = %.4f ms\n", float64(x.histo.Max())/1000)
				fmt.Printf("mean = %.4f ms\n", x.histo.Mean()/1000)
				fmt.Printf("%%50 <= %.4f ms\n", ps[0]/1000)
				fmt.Printf("%%75 <= %.4f ms\n", ps[1]/1000)
				fmt.Printf("%%95 <= %.4f ms\n", ps[2]/1000)
				fmt.Printf("%%99 <= %v ms\n", ps[3]/1000)
				fmt.Println()
				/* h.Clear() */
			}
		case <-rawTicker.C:
			for k, v := range rawData {
				if time.Since(v.reqTime) > time.Duration(60*time.Second) {
					delete(rawData, k)
				}
			}
		}
	}
}
