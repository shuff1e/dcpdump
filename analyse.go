package main

import (
	"github.com/google/gopacket/layers"
	"github.com/couchbase/gomemcached"
	"github.com/rcrowley/go-metrics"
	"net"
	/* "sort" */
	"time"
    "strconv"
)

type MCReqAndTime struct {
	request gomemcached.MCRequest
	reqTime time.Time
	srcIP   net.IP
    srcPort layers.TCPPort
	dstIP   net.IP
    dstPort layers.TCPPort
}

type MCRespAndTime struct {
	response gomemcached.MCResponse
	respTime time.Time
	srcIP    net.IP
    srcPort layers.TCPPort
	dstIP    net.IP
    dstPort layers.TCPPort
}

func (req MCReqAndTime) Key() string {
	return req.srcIP.String()+ req.srcPort.String() + req.dstIP.String() + req.dstPort.String() + strconv.Itoa(int(req.request.Opaque))
}

func (req MCReqAndTime) modeServer(mode string) string {
    if mode == "client" {
        return req.dstIP.String()
    } else {
        return req.srcIP.String()
    }
}

func (resp MCRespAndTime) Key() string {
	return resp.dstIP.String() + resp.dstPort.String() + resp.srcIP.String()+ resp.srcPort.String() +  strconv.Itoa(int(resp.response.Opaque))
}

type reqAndTime struct {
	MCReqAndTime
	spentTime time.Duration
}

var (
	/* data        opeHeap */
	initialTime time.Time
)

type counterAndHisto struct {
	all     metrics.Counter
	timeout metrics.Counter
	histo   metrics.Histogram
}

func initMetrics() counterAndHisto {
    c1 := metrics.NewCounter()
    c2 := metrics.NewCounter()
    s := metrics.NewExpDecaySample(1024, 0.015)
    /* s := metrics.NewUniformSample(1028) */
    h := metrics.NewHistogram(s)
    return counterAndHisto{c1,c2,h}
}

func analyse() {
	ticker := time.NewTicker(time.Duration(options.printInterval) * time.Second)
	serverMetrics := make(map[string]counterAndHisto)
	rawTicker := time.NewTicker(120 * time.Second)
	rawData := make(map[string]MCReqAndTime)
	for {
		select {
		case req := <-reqChan:
			if _, ok := rawData[req.Key()]; !ok {
				rawData[req.Key()] = req
			}
		case resp := <-respChan:
			if req, ok := rawData[resp.Key()]; ok {
				spentTime := resp.respTime.Sub(req.reqTime)
				if _, ok := serverMetrics[req.modeServer(options.mode)]; !ok {
					serverMetrics[req.modeServer(options.mode)] = initMetrics()
                }
                ch := serverMetrics[req.modeServer(options.mode)]
                ch.all.Inc(1)
                ch.histo.Update(spentTime.Nanoseconds() / 1000)
				if spentTime > time.Duration(time.Duration(options.timeout)*time.Millisecond) {
					ch.timeout.Inc(1)
                    if options.printAll {
                        Printf("%s, %s, %21s => %21s ,resp received at %s, spent %s\n", req.request.Opcode, string(req.request.Key), req.srcIP.String() + ":" +req.srcPort.String(), req.dstIP.String() + ":" + req.dstPort.String(), resp.respTime, spentTime)
                    }
                }
				delete(rawData, req.Key())
				/* Push(data, reqAndTime{req, spentTime}) */
			}
		case <-ticker.C:
			Printf("\n\n--------------------------------------\n")
			/* Printf("\n\n------------------top %v request with the longest response time-----------------------------------\n", options.topN) */
			/* sort.Sort(data) */
			/* for _, x := range data { */
			/* 	if x.reqTime != initialTime { */
			/* 		Printf("%s, key %s, sent from %21s to %21s at %s, spent %s\n", x.request.Opcode, string(x.request.Key), x.srcIP.String() + ":" +x.srcPort.String(), x.dstIP.String() + ":" + x.dstPort.String(), x.reqTime, x.spentTime) */
			/* 		/1* data[i] = reqAndTime{} *1/ */
			/* 	} */
			/* } */
			/* Heapify(data) */
			Printf("\n")
			for i, x := range serverMetrics {
				Printf("metrics of server %s\n", i)
				Printf("%v timeout in %v\n", x.timeout.Count(), x.all.Count())
				ps := x.histo.Percentiles([]float64{0.5, 0.75, 0.95, 0.99})
				Printf("min = %.4f ms\n", float64(x.histo.Min())/1000)
				Printf("max = %.4f ms\n", float64(x.histo.Max())/1000)
				Printf("mean = %.4f ms\n", x.histo.Mean()/1000)
				Printf("%%50 <= %.4f ms\n", ps[0]/1000)
				Printf("%%75 <= %.4f ms\n", ps[1]/1000)
				Printf("%%95 <= %.4f ms\n", ps[2]/1000)
				Printf("%%99 <= %v ms\n", ps[3]/1000)
				Println()
				/* h.Clear() */
			}
			Printf("--------------------------------------\n\n")
		case <-rawTicker.C:
			for k, v := range rawData {
				if time.Since(v.reqTime) > time.Duration(60*time.Second) {
					delete(rawData, k)
				}
			}
		}
	}
}
