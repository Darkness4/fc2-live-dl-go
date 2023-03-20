package queue_test

import (
	"io"
	"sync"
	"testing"
	"time"

	"github.com/Darkness4/fc2-live-dl-lite/utils/blockingheap"
	"github.com/Darkness4/fc2-live-dl-lite/utils/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueue(t *testing.T) {
	q := queue.NewPriorityQueue[int](100)
	bq := blockingheap.New[*queue.Item[int]](q)
	fixture := &queue.Item[int]{
		Value:    5,
		Priority: 0,
	}
	err := bq.Push(fixture)
	require.NoError(t, err)
	value, err := bq.Pop()
	assert.NoError(t, err)
	assert.EqualValues(t, fixture, value)
}

func TestPriorityQueue(t *testing.T) {
	q := queue.NewPriorityQueue[int](100)
	bq := blockingheap.New[*queue.Item[int]](q)
	fixture := &queue.Item[int]{
		Value:    5,
		Priority: 0,
	}
	fixture2 := &queue.Item[int]{
		Value:    5,
		Priority: 1,
	}

	err := bq.Push(fixture)
	require.NoError(t, err)
	err = bq.Push(fixture2)
	require.NoError(t, err)
	value, err := bq.Pop()
	assert.EqualValues(t, fixture, value)
	assert.NoError(t, err)
	value, err = bq.Pop()
	assert.EqualValues(t, fixture2, value)
	assert.NoError(t, err)
}

func TestBlockingQueue(t *testing.T) {
	q := queue.NewPriorityQueue[int](100)
	bq := blockingheap.New[*queue.Item[int]](q)
	fixture := &queue.Item[int]{
		Value:    5,
		Priority: 0,
	}

	go func() {
		time.Sleep(time.Second)
		err := bq.Push(fixture)
		require.NoError(t, err)
	}()

	value, err := bq.Pop()
	assert.NoError(t, err)
	assert.EqualValues(t, fixture, value)
}

func TestMultiListenerBlockingQueue(t *testing.T) {
	q := queue.NewPriorityQueue[int](100)
	bq := blockingheap.New[*queue.Item[int]](q)
	fixture := &queue.Item[int]{
		Value:    5,
		Priority: 0,
	}
	fixture2 := &queue.Item[int]{
		Value:    5,
		Priority: 1,
	}

	go func() {
		time.Sleep(time.Second)
		err := bq.Push(fixture)
		require.NoError(t, err)
		time.Sleep(time.Second)
		err = bq.Push(fixture2)
		require.NoError(t, err)
	}()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		value, err := bq.Pop()
		assert.NoError(t, err)
		assert.EqualValues(t, fixture2, value)
		wg.Done()
	}()
	go func() {
		value, err := bq.Pop()
		assert.NoError(t, err)
		assert.EqualValues(t, fixture, value)
		wg.Done()
	}()
	wg.Wait()
}

func TestMultiListenerBlockingQueueAbort(t *testing.T) {
	q := queue.NewPriorityQueue[int](100)
	bq := blockingheap.New[*queue.Item[int]](q)

	go func() {
		time.Sleep(time.Second)
		bq.Close()
	}()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		value, err := bq.Pop()
		assert.Equal(t, io.EOF, err)
		assert.Nil(t, value)
		wg.Done()
	}()
	go func() {
		value, err := bq.Pop()
		assert.Equal(t, io.EOF, err)
		assert.Nil(t, value)
		wg.Done()
	}()
	wg.Wait()
}
