package queuemod

import (
	"fmt"
	"sync/atomic"
	"time"
)

// --- queue module ---

type Queue struct{}

func (*Queue) New(args ...interface{}) interface{} {
	capacity := 1024
	if len(args) > 0 {
		capacity = rugo_to_int(args[0])
		if capacity < 0 {
			panic("queue.new: capacity must be non-negative")
		}
	}
	return interface{}(&rugoQueue{
		ch: make(chan interface{}, capacity),
	})
}

// rugoQueue is the runtime representation of a Rugo queue.
type rugoQueue struct {
	ch     chan interface{}
	closed int32 // atomic: 0 = open, 1 = closed
}

// DotGet implements property access on queue objects.
func (q *rugoQueue) DotGet(field string) (interface{}, bool) {
	switch field {
	case "size":
		return len(q.ch), true
	case "closed":
		return atomic.LoadInt32(&q.closed) == 1, true
	}
	return nil, false
}

// DotCall implements method calls on queue objects.
func (q *rugoQueue) DotCall(method string, args ...interface{}) (interface{}, bool) {
	switch method {
	case "push":
		if len(args) < 1 {
			panic("queue.push requires a value")
		}
		if atomic.LoadInt32(&q.closed) == 1 {
			panic("cannot push to a closed queue")
		}
		q.ch <- args[0]
		return nil, true
	case "pop":
		if len(args) > 0 {
			return q.popWithTimeout(args[0]), true
		}
		item, ok := <-q.ch
		if !ok {
			panic("cannot pop from a closed and empty queue")
		}
		return item, true
	case "size":
		return len(q.ch), true
	case "close":
		if !atomic.CompareAndSwapInt32(&q.closed, 0, 1) {
			panic("queue is already closed")
		}
		close(q.ch)
		return nil, true
	case "each":
		if len(args) < 1 {
			panic("queue.each requires a function argument")
		}
		fn, ok := args[0].(func(...interface{}) interface{})
		if !ok {
			panic(fmt.Sprintf("queue.each expects a function, got %T", args[0]))
		}
		for item := range q.ch {
			fn(item)
		}
		return nil, true
	}
	return nil, false
}

func (q *rugoQueue) popWithTimeout(timeout interface{}) interface{} {
	secs := rugo_to_int(timeout)
	select {
	case item, ok := <-q.ch:
		if !ok {
			panic("cannot pop from a closed and empty queue")
		}
		return item
	case <-time.After(time.Duration(secs) * time.Second):
		panic(fmt.Sprintf("queue.pop timed out after %d seconds", secs))
	}
}
