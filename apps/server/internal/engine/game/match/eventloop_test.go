package match

import (
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
		t.Fatal("an event is pending as soon as schedule returns, before its goroutine runs")
	}

	close(release)
	eventually(t, func() bool { return !el.hasPending() }, "the event stopped counting once it had run")

	if !ran.Load() {
		t.Fatal("the loop was reported idle before the event finished")
	}
}

func TestEventLoopStopsCountingDroppedEvents(t *testing.T) {
	el := startTestEventLoop(t)
	release := blockLoop(t, el)

	// While the loop is busy, one of these takes the single queue slot and the
	// other is dropped at once; the queued one is dropped when the loop drains
	// the queue after the blocking event.
	var ran atomic.Int32
	for range 2 {
		el.schedule(func() { ran.Add(1) }, SequentialEvent)
	}

	eventually(t, func() bool { return el.pending.Load() == 2 }, "the event that could not be queued stopped counting")

	close(release)
	eventually(t, func() bool { return !el.hasPending() }, "the drained event stopped counting")

	if ran.Load() != 0 {
		t.Fatalf("events scheduled while the loop was busy are dropped, but %d ran", ran.Load())
	}
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
