// Package timerqueue provides a timer queue for a list of items that should be processed
// after a fixed duration of time from when they are added to the queue.
package timerqueue

import (
	"sync"
	"time"
)

// Queue holds a list of elements. When a new element is added to the
// queue, the queue callback will be called with the element after
// the set queue duration.
type Queue struct {
	first, last *element
	m           map[interface{}]*element
	mu          sync.Mutex
	duration    time.Duration
	cb          func(v interface{})
}

type element struct {
	v          interface{}
	next, prev *element
	time       time.Time
}

// New creates a new timer Queue
func New(callback func(interface{}), duration time.Duration) *Queue {
	return &Queue{
		m:        make(map[interface{}]*element),
		duration: duration,
		cb:       callback,
	}
}

// Add adds a new element to the timer Queue, starting a timer
// for the fixed duration for the queue. Once the duration
// has passed, the queue callback will be called.
func (q *Queue) Add(v interface{}) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Ensure it is not a duplicate
	if _, ok := q.m[v]; ok {
		panic("Value already in queue")
	}

	el := &element{v: v, time: time.Now().Add(q.duration)}

	q.m[v] = el
	q.push(el)
	if el == q.first {
		go q.timer(el, el.time)
	}
}

func (q *Queue) Len() int {

	c := 0
	el := q.first
	for el != nil {
		el = el.next
		c++
	}

	return c
}

// Clear removes all elements from the queue.
func (q *Queue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.first = nil
	q.last = nil
	q.m = make(map[interface{}]*element)
}

// Remove removes an element from the queue.
// If the element is not in the queue, the call is a no-op
func (q *Queue) Remove(v interface{}) {
	q.mu.Lock()
	defer q.mu.Unlock()

	el, ok := q.m[v]
	if !ok {
		return
	}

	first := q.first
	delete(q.m, v)
	q.remove(el)

	// If the element was first, we need to start a new timer
	if first == el && q.first != nil {
		go q.timer(q.first, q.first.time)
	}
}

// Reset sets the time of the element callback back to full duration.
func (q *Queue) Reset(v interface{}) {
	q.mu.Lock()
	defer q.mu.Unlock()

	el, ok := q.m[v]
	if !ok {
		return
	}

	el.time = time.Now().Add(q.duration)
	first := q.first
	q.remove(el)
	q.push(el)

	// If the element was first, we need to start a new timer
	if first == el {
		go q.timer(q.first, q.first.time)
	}
}

func (q *Queue) remove(el *element) {

	if q.first == el {
		q.first = el.next
	} else {
		el.prev.next = el.next
	}

	if q.last == el {
		q.last = el.prev
	} else {
		el.next.prev = el.prev
	}
}

func (q *Queue) push(el *element) {

	last := q.last
	if last != nil {
		last.next = el
		el.prev = last
	} else {
		q.first = el
		el.prev = nil
	}

	el.next = nil
	q.last = el
}

func (q *Queue) timer(el *element, t time.Time) {
	var v interface{}
	for {
		time.Sleep(t.Sub(time.Now()))

		q.mu.Lock()
		// Check if the first element has changed
		if el != q.first || t != el.time {
			q.mu.Unlock()
			break
		}

		v = el.v
		q.remove(el)
		delete(q.m, v)

		el = q.first
		if el == nil {
			q.mu.Unlock()
			q.cb(v)
			break
		}
		t = el.time
		q.mu.Unlock()

		q.cb(v)
	}
}
