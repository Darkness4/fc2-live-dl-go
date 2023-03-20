package queue

import (
	"sync"
)

type Item[T any] struct {
	Value    T
	Priority int
	index    int
}

type PriorityQueue[T any] struct {
	items  []*Item[T]
	qMutex sync.Mutex // Mutex to protect access to the queue
}

func NewPriorityQueue[T any](size int) *PriorityQueue[T] {
	return &PriorityQueue[T]{
		items: make([]*Item[T], 0, size),
	}
}

func (pq *PriorityQueue[T]) Len() int {
	pq.qMutex.Lock()
	defer pq.qMutex.Unlock()
	return len(pq.items)
}

func (pq *PriorityQueue[T]) Less(i, j int) bool {
	pq.qMutex.Lock()
	defer pq.qMutex.Unlock()
	// Gives the lowest
	return pq.items[i].Priority < pq.items[j].Priority
}

func (pq *PriorityQueue[T]) Swap(i, j int) {
	pq.qMutex.Lock()
	defer pq.qMutex.Unlock()
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
	pq.items[i].index = i
	pq.items[j].index = j
}

func (pq *PriorityQueue[T]) Push(x *Item[T]) {
	pq.qMutex.Lock()
	defer pq.qMutex.Unlock()
	n := len(pq.items)
	x.index = n
	pq.items = append(pq.items, x)
}

func (pq *PriorityQueue[T]) Pop() *Item[T] {
	pq.qMutex.Lock()
	defer pq.qMutex.Unlock()
	old := pq.items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	pq.items = old[0 : n-1]
	return item
}
