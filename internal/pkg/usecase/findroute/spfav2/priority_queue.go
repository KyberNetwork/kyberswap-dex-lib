package spfav2

import (
	"container/heap"
)

type priorityQueue struct {
	cmpFunc func(a, b int) bool
	pathIds []int
}

func NewPriorityQueue(n int, cmpFunc func(a, b int) bool) *priorityQueue {
	pathIds := make([]int, n)
	for i := 0; i < n; i++ {
		pathIds[i] = i
	}
	pq := &priorityQueue{
		cmpFunc: cmpFunc,
		pathIds: pathIds,
	}
	heap.Init(pq)
	return pq
}

func (pq priorityQueue) Len() int { return len(pq.pathIds) }
func (pq priorityQueue) Less(i, j int) bool {
	return pq.cmpFunc(pq.pathIds[i], pq.pathIds[j])
}
func (pq priorityQueue) Swap(i, j int) { pq.pathIds[i], pq.pathIds[j] = pq.pathIds[j], pq.pathIds[i] }

func (pq *priorityQueue) Push(x interface{}) {
	pq.pathIds = append(pq.pathIds, x.(int))
}

func (pq *priorityQueue) Pop() interface{} {
	n := pq.Len()
	x := pq.pathIds[n-1]
	pq.pathIds = pq.pathIds[0 : n-1]
	return x
}

func (pq *priorityQueue) Top() interface{} {
	return pq.pathIds[0]
}
