package match

import (
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const eventLoopTestTimeout = 2 * time.Second

func startTestEventLoop(t *testing.T) *EventLoop {
	t.Helper()

	el := NewEventLoop()
	go el.start()
	t.Cleanup(el.stop)

	return el
}

// eventually fails the test unless condition becomes true in time.
func eventually(t *testing.T, condition func() bool, message string) {
	t.Helper()

	deadline := time.Now().Add(eventLoopTestTimeout)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal(message)
		}
		time.Sleep(time.Millisecond)
	}
}

// blockLoop schedules an event that holds the loop until release is closed,
// and waits for it to start.
func blockLoop(t *testing.T, el *EventLoop) (release chan struct{}) {
	t.Helper()

	started := make(chan struct{})
	release = make(chan struct{})
	el.schedule(func() {
		close(started)
		<-release
	}, SequentialEvent)

	select {
	case <-started:
	case <-time.After(eventLoopTestTimeout):
		t.Fatal("the blocking event never started")
	}

	return release
}

func TestEventLoopIsIdleBeforeAnyEvent(t *testing.T) {
	el := startTestEventLoop(t)

	if el.hasPending() {
		t.Fatal("a new loop has nothing pending")
	}
}

func TestEventLoopCountsAnEventFromSchedulingUntilItHasRun(t *testing.T) {
	el := startTestEventLoop(t)
	release := make(chan struct{})
	var ran atomic.Bool

	el.schedule(func() {
		<-release
		ran.Store(true)
	}, SequentialEvent)

	if !el.hasPending() {
		t.Fatal("an event is pending as soon as schedule returns, before it runs")
	}

	close(release)
	eventually(t, func() bool { return !el.hasPending() }, "the event stopped counting once it had run")

	if !ran.Load() {
		t.Fatal("the loop was reported idle before the event finished")
	}
}

func TestEventLoopRunsEventsScheduledWhileBusyInOrder(t *testing.T) {
	el := startTestEventLoop(t)
	release := blockLoop(t, el)

	// While the loop is busy, as when it waits for a player to answer, more
	// events arrive. They wait their turn rather than being dropped.
	var mu sync.Mutex
	var ran []int
	for i := range 5 {
		el.schedule(func() {
			mu.Lock()
			defer mu.Unlock()
			ran = append(ran, i)
		}, SequentialEvent)
	}

	if got := el.pending.Load(); got != 6 {
		t.Fatalf("the busy event and the five waiting are pending, got %d", got)
	}

	close(release)
	eventually(t, func() bool { return !el.hasPending() }, "every event ran")

	mu.Lock()
	defer mu.Unlock()
	if !slices.Equal(ran, []int{0, 1, 2, 3, 4}) {
		t.Fatalf("events run in the order they were scheduled, got %v", ran)
	}
}

func TestEventLoopDropsEventsPastTheQueueLimit(t *testing.T) {
	el := startTestEventLoop(t)
	release := blockLoop(t, el)

	var ran atomic.Int32
	for range queueLimit + 10 {
		el.schedule(func() { ran.Add(1) }, SequentialEvent)
	}

	if got := el.pending.Load(); got != queueLimit+1 {
		t.Fatalf("the busy event and a full queue are pending, got %d", got)
	}

	close(release)
	eventually(t, func() bool { return !el.hasPending() }, "the queued events ran")

	if ran.Load() != queueLimit {
		t.Fatalf("the events that fit in the queue run and the rest are dropped, but %d ran", ran.Load())
	}

	// With room again, events are taken as before.
	el.schedule(func() { ran.Add(1) }, SequentialEvent)
	eventually(t, func() bool { return ran.Load() == queueLimit+1 }, "an event scheduled later runs")
}

func TestEventLoopStopsCountingAnEventThatPanics(t *testing.T) {
	el := startTestEventLoop(t)

	el.schedule(func() { panic("broken event") }, SequentialEvent)

	eventually(t, func() bool { return !el.hasPending() }, "a panicking event stopped counting")
}

func TestEventLoopDoesNotCountParallelEvents(t *testing.T) {
	el := startTestEventLoop(t)
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })

	el.schedule(func() { <-release }, ParallelEvent)

	if el.hasPending() {
		t.Fatal("a parallel event runs beside the loop and is not pending on it")
	}
}

func TestStoppedEventLoopHasNothingPending(t *testing.T) {
	el := NewEventLoop()
	go el.start()

	release := blockLoop(t, el)
	el.schedule(func() {}, SequentialEvent)

	el.stop()
	close(release)
	eventually(t, func() bool { return !el.isRunning() }, "the loop exited")

	if el.hasPending() {
		t.Fatal("an exited loop never runs what it left queued, so nothing is pending")
	}
}
