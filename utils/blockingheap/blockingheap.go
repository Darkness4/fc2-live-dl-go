package blockingheap

import (
	"io"
	"sync"

	"github.com/Darkness4/fc2-live-dl-lite/utils/heap"
)

type BlockingHeap[T any] struct {
	heap heap.Interface[T]
	*sync.Mutex
	*sync.Cond // Condition variable to block and signal waiting goroutines
	isClosed   bool
}

func New[T any](h heap.Interface[T]) *BlockingHeap[T] {
	// heapify
	heap.Init(h)
	mu := &sync.Mutex{}
	return &BlockingHeap[T]{
		heap:  h,
		Mutex: mu,
		Cond:  sync.NewCond(mu),
	}
}

func (h *BlockingHeap[T]) Push(x T) error {
	h.Lock()
	defer h.Unlock()
	if h.isClosed {
		return io.EOF
	}
	n := h.heap.Len()
	heap.Push(h.heap, x)
	if n == 0 {
		h.Signal()
	}
	return nil
}

func (h *BlockingHeap[T]) Pop() (out T, _ error) {
	h.Lock()
	defer h.Unlock()
	for h.heap.Len() == 0 && !h.isClosed {
		h.Wait()
	}
	if h.isClosed {
		return out, io.EOF
	}
	return heap.Pop(h.heap), nil
}

func (h *BlockingHeap[T]) Close() {
	h.Lock()
	defer h.Unlock()
	h.isClosed = true
	h.Broadcast()
}
