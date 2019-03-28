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
	Delay  time.Duration
	Action TestAction
	Item   ItemID
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
			q.Remove(ev.Item)
		case Reset:
			q.Reset(ev.Item)
		case Clear:
			q.Clear()
		case Flush:
			q.Flush()
		case Len:
			l := q.Len()
			if l != int(ev.Item) {
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
			{0, Add, 1},
		},
		Expected: []ItemID{1},
	})
}

func TestAddMultipleItemsWithoutDelay(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1},
			{0, Add, 2},
			{0, Add, 3},
		},
		Expected: []ItemID{1, 2, 3},
	})
}

func TestAddMultipleItemsWithDelay(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1},
			{5, Add, 2},
			{5, Add, 3},
		},
		Expected: []ItemID{1, 2, 3},
	})
}

func TestAddMultipleItemsWithLongDelay(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1},
			{15, Add, 2},
			{15, Add, 3},
		},
		Expected: []ItemID{1, 2, 3},
	})
}

func TestResetOnSingleItem(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1},
			{5, Reset, 1},
		},
		Expected: []ItemID{1},
	})
}

func TestResetOnFirstItem(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1},
			{1, Add, 2},
			{1, Add, 3},
			{1, Reset, 1},
		},
		Expected: []ItemID{2, 3, 1},
	})
}

func TestResetOnMiddleItem(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1},
			{1, Add, 2},
			{1, Add, 3},
			{1, Reset, 2},
		},
		Expected: []ItemID{1, 3, 2},
	})
}

func TestResetOnLastItem(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1},
			{1, Add, 2},
			{1, Add, 3},
			{1, Reset, 3},
		},
		Expected: []ItemID{1, 2, 3},
	})
}

func TestRemoveItem(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1},
			{1, Remove, 1},
		},
		Expected: []ItemID{},
	})
}

func TestRemoveOnFirstItem(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1},
			{1, Add, 2},
			{1, Add, 3},
			{1, Remove, 1},
		},
		Expected: []ItemID{2, 3},
	})
}

func TestRemoveOnMiddleItem(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1},
			{1, Add, 2},
			{1, Add, 3},
			{1, Remove, 2},
		},
		Expected: []ItemID{1, 3},
	})
}

func TestRemoveOnLastItem(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1},
			{1, Add, 2},
			{1, Add, 3},
			{1, Remove, 3},
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
			{0, Add, 1},
			{1, Add, 1},
		},
		Expected: []ItemID{1},
	})
}

func TestClearOnEmpty(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Clear, 0},
		},
		Expected: []ItemID{},
	})
}

func TestClearOnList(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1},
			{15, Add, 2},
			{1, Clear, 0},
			{1, Add, 3},
		},
		Expected: []ItemID{1, 3},
	})
}

func TestFlushOnEmpty(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Flush, 0},
		},
		Expected: []ItemID{},
	})
}

func TestFlushOnList(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1},
			{1, Add, 2},
			{1, Flush, 0},
			{1, Len, 1},
		},
		Expected: []ItemID{1, 2},
	})
}

func TestLen(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Len, 0},
			{1, Add, 1},
			{0, Len, 1},
			{1, Add, 2},
			{0, Len, 2},
			{1, Add, 3},
			{1, Len, 3},
		},
		Expected: []ItemID{1, 2, 3},
	})
}

func TestRemoveOnNonExisting(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1},
			{1, Add, 2},
			{1, Add, 3},
			{15, Remove, 1},
		},
		Expected: []ItemID{1, 2, 3},
	})
}

func TestResetOnNonExisting(t *testing.T) {
	runTestSet(t, TestSet{
		Events: []TestEvent{
			{0, Add, 1},
			{1, Add, 2},
			{1, Add, 3},
			{15, Reset, 1},
		},
		Expected: []ItemID{1, 2, 3},
	})
}
