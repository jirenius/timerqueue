package timerqueue

import (
	"sync"
	"testing"
	"time"
)

var (
	timeUnits     = time.Millisecond * 20
	queueDuration = timeUnits * 10
)

type ItemID int

type TestAction int

const (
	Add TestAction = iota
	Remove
	Reset
	Clear
	Flush
	Len
)

type TestSet struct {
	Events   []TestEvent
	Expected []ItemID
}

type TestEvent struct {
	Delay    time.Duration
	Action   TestAction
	Item     ItemID
	Expected interface{}
}

func runTestSet(t *testing.T, set TestSet) {
	var itemIDs []ItemID
	var wg sync.WaitGroup

	wg.Add(len(set.Expected))

	q := New(callback(t, &wg, &itemIDs), queueDuration)

	for _, ev := range set.Events {
		time.Sleep(timeUnits * ev.Delay)

		switch ev.Action {
		case Add:
			q.Add(ev.Item)
		case Remove:
			r := q.Remove(ev.Item)
			if r != ev.Expected.(bool) {
				t.Errorf("Queue.Remove(): expected %+v, actual %+v", ev.Expected, r)
			}
		case Reset:
			r := q.Reset(ev.Item)
			if r != ev.Expected.(bool) {
				t.Errorf("Queue.Reset(): expected %+v, actual %+v", ev.Expected, r)
			}
		case Clear:
			elems := q.Clear()
			ex := ev.Expected.([]interface{})
			if len(elems) != len(ex) {
				t.Errorf("Queue.Clear() len: expected %d, actual %d", len(ex), len(elems))
			}
			for i, v := range elems {
				if v != ex[i] {
					t.Errorf("Queue.Clear() index %d: expected %d, actual %d", i, ex[i], v)
				}
			}
		case Flush:
			q.Flush()
		case Len:
			l := q.Len()
			if l != ev.Expected.(int) {
				t.Errorf("Queue.Len(): expected %d, actual %d", ev.Item, l)
			}
		}
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(queueDuration * 4):
		t.Errorf("Number of callback calls: expected %d, actual %d", len(set.Expected), len(itemIDs))
		return
	}

	for i, v := range set.Expected {
		if v != itemIDs[i] {
			t.Errorf("Queue callback %d: expected item %d, actual %d", i, v, itemIDs[i])
		}
	}
}

func callback(t *testing.T, wg *sync.WaitGroup, itemIDs *[]ItemID) func(interface{}) {
	var mu sync.Mutex
	return func(v interface{}) {
		mu.Lock()
		defer mu.Unlock()

		itemID := v.(ItemID)

		*itemIDs = append(*itemIDs, itemID)
		wg.Done()
	}
}

func TestAddSingleItem(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1, nil},
		},
		Expected: []ItemID{1},
	})
}

func TestAddMultipleItemsWithoutDelay(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1, nil},
			{0, Add, 2, nil},
			{0, Add, 3, nil},
		},
		Expected: []ItemID{1, 2, 3},
	})
}

func TestAddMultipleItemsWithDelay(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1, nil},
			{5, Add, 2, nil},
			{5, Add, 3, nil},
		},
		Expected: []ItemID{1, 2, 3},
	})
}

func TestAddMultipleItemsWithLongDelay(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1, nil},
			{15, Add, 2, nil},
			{15, Add, 3, nil},
		},
		Expected: []ItemID{1, 2, 3},
	})
}

func TestResetOnSingleItem(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1, nil},
			{5, Reset, 1, true},
		},
		Expected: []ItemID{1},
	})
}

func TestResetOnFirstItem(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1, nil},
			{1, Add, 2, nil},
			{1, Add, 3, nil},
			{1, Reset, 1, true},
		},
		Expected: []ItemID{2, 3, 1},
	})
}

func TestResetOnMiddleItem(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1, nil},
			{1, Add, 2, nil},
			{1, Add, 3, nil},
			{1, Reset, 2, true},
		},
		Expected: []ItemID{1, 3, 2},
	})
}

func TestResetOnLastItem(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1, nil},
			{1, Add, 2, nil},
			{1, Add, 3, nil},
			{1, Reset, 3, true},
		},
		Expected: []ItemID{1, 2, 3},
	})
}

func TestRemoveItem(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1, nil},
			{1, Remove, 1, true},
		},
		Expected: []ItemID{},
	})
}

func TestRemoveOnFirstItem(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1, nil},
			{1, Add, 2, nil},
			{1, Add, 3, nil},
			{1, Remove, 1, true},
		},
		Expected: []ItemID{2, 3},
	})
}

func TestRemoveOnMiddleItem(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1, nil},
			{1, Add, 2, nil},
			{1, Add, 3, nil},
			{1, Remove, 2, true},
		},
		Expected: []ItemID{1, 3},
	})
}

func TestRemoveOnLastItem(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1, nil},
			{1, Add, 2, nil},
			{1, Add, 3, nil},
			{1, Remove, 3, true},
		},
		Expected: []ItemID{1, 2},
	})
}

func TestPanicOnDuplicate(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1, nil},
			{1, Add, 1, nil},
		},
		Expected: []ItemID{1},
	})
}

func TestClearOnEmpty(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Clear, 0, []interface{}{}},
		},
		Expected: []ItemID{},
	})
}

func TestClearOnList(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1, nil},
			{15, Add, 2, nil},
			{1, Clear, 1, []interface{}{ItemID(2)}},
			{1, Add, 3, nil},
		},
		Expected: []ItemID{1, 3},
	})
}

func TestFlushOnEmpty(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Flush, 0, nil},
		},
		Expected: []ItemID{},
	})
}

func TestFlushOnList(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1, nil},
			{1, Add, 2, nil},
			{1, Flush, 0, nil},
			{1, Len, 0, 0},
		},
		Expected: []ItemID{1, 2},
	})
}

func TestLen(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Len, 0, 0},
			{1, Add, 1, nil},
			{0, Len, 0, 1},
			{1, Add, 2, nil},
			{0, Len, 0, 2},
			{1, Add, 3, nil},
			{1, Len, 0, 3},
		},
		Expected: []ItemID{1, 2, 3},
	})
}

func TestRemoveOnNonExisting(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1, nil},
			{1, Add, 2, nil},
			{1, Add, 3, nil},
			{15, Remove, 1, false},
		},
		Expected: []ItemID{1, 2, 3},
	})
}

func TestResetOnNonExisting(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1, nil},
			{1, Add, 2, nil},
			{1, Add, 3, nil},
			{15, Reset, 1, false},
		},
		Expected: []ItemID{1, 2, 3},
	})
}
