package main

import (
	"container/heap"
)

type opeHeap []reqAndTime

func (h opeHeap) Len() int           { return len(h) }
func (h opeHeap) Less(i, j int) bool { return h[i].spentTime < h[j].spentTime }
func (h opeHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h opeHeap) Push(x interface{}) {
}

func (h opeHeap) Pop() interface{} {
	return nil
}

func Push(h opeHeap, value reqAndTime) {
	if value.spentTime > h[0].spentTime {
		h[0] = value
	}
	heap.Fix(h, 0)
}

func Heapify(h opeHeap) {
    heap.Init(h)
}
