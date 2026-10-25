package match

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/internal"
	"runtime/debug"
	"sync"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

type EventExecutionStrategy int

const (
	SequentialEvent EventExecutionStrategy = iota
	ParallelEvent
)

// queueLimit is how many sequential events may wait for the loop at once. A
// player sends a handful at most, one per click; past the limit, events are
// dropped, so a flood of messages cannot grow the queue without end.
const queueLimit = 64

// EventLoop runs a match's sequential events one at a time, in the order they
// were scheduled, on its own goroutine. Parallel events run at once, beside
// it.
type EventLoop struct {
	stopped sync.Once
	running atomic.Bool
	exit    chan struct{}

	// queue holds the sequential events waiting to run, oldest first, and mu
	// guards it. wake tells the loop the queue has something; it holds one
	// wake-up, so scheduling never blocks.
	mu    sync.Mutex
	queue []func()
	wake  chan struct{}

	// pending counts the sequential events scheduled and not yet over: queued
	// or running. An event that is dropped is never counted.
	pending atomic.Int64
}

func NewEventLoop() *EventLoop {
	el := &EventLoop{
		exit: make(chan struct{}),
		wake: make(chan struct{}, 1),
	}

	// Set here rather than in start so that the loop does not look stopped in
	// the window between constructing it and its goroutine being scheduled.
	el.running.Store(true)

	return el
}

// isRunning reports whether the loop's goroutine is still alive. A stopped match
// whose loop is still running is stuck in, or spinning on, an event.
func (el *EventLoop) isRunning() bool {
	return el.running.Load()
}

// hasPending reports whether the loop still has sequential events to run. A
// loop that has exited has none: whatever it left queued never runs.
func (el *EventLoop) hasPending() bool {
	return el.running.Load() && el.pending.Load() > 0
}

func (el *EventLoop) start() {
	defer internal.Recover()
	defer el.running.Store(false)
	defer logrus.Debug("Stopped event loop")

	for {
		select {
		case <-el.exit:
			return
		case <-el.wake:
		}

		for event := el.next(); event != nil; event = el.next() {
			el.process(event)
			el.pending.Add(-1)

			// Stopping leaves the rest of the queue unrun.
			select {
			case <-el.exit:
				return
			default:
			}
		}
	}
}

// next takes the oldest queued event, or returns nil when there is none.
func (el *EventLoop) next() func() {
	el.mu.Lock()
	defer el.mu.Unlock()

	if len(el.queue) == 0 {
		return nil
	}

	event := el.queue[0]
	el.queue[0] = nil
	el.queue = el.queue[1:]

	return event
}

func (el *EventLoop) stop() {

	// Closing rather than sending means stopping never blocks the caller. An
	// event may currently be processing and waiting for a player action, in
	// which case the loop only reaches its select once the caller has disposed
	// the players and the abandoned prompt has unwound. Closing also makes
	// repeated stops safe, and makes stopping a loop that has already exited a
	// no-op instead of a goroutine that blocks on the send forever.
	el.stopped.Do(func() {
		close(el.exit)
	})

}

// schedule runs an event. A sequential event is queued behind the ones
// scheduled before it, and runs once they have, however long they take: while
// an event waits for a player to answer a prompt, the events scheduled after
// it wait too. Scheduling never blocks.
func (el *EventLoop) schedule(event func(), strategy EventExecutionStrategy) {

	if strategy == ParallelEvent {
		go func() {
			// A parallel event should never open a prompt, but it runs on its
			// own goroutine, so an abort raised by one has no other boundary
			// to reach.
			defer recoverEvent()

			event()
		}()
		return
	}

	el.mu.Lock()
	if len(el.queue) >= queueLimit {
		el.mu.Unlock()
		logrus.Warn("Dropped an incoming event: too many are waiting")
		return
	}
	el.queue = append(el.queue, event)
	// Counted before schedule returns, so that from then on the event shows
	// as pending until it has run.
	el.pending.Add(1)
	el.mu.Unlock()

	select {
	case el.wake <- struct{}{}:
	default:
	}

}

func (el *EventLoop) process(event func()) {

	defer recoverEvent()

	event()

}

// recoverEvent ends the event in progress. An event that was waiting for a
// player when the match was disposed aborts through NextAction, which is an
// ordinary shutdown rather than a fault, so it is not reported as one. Anything
// else is a real panic and keeps its warning and stack trace.
func recoverEvent() {

	r := recover()

	if r == nil {
		return
	}

	if IsMatchDisposed(r) {
		logrus.Debug("Abandoned an event because the match was disposed")
		return
	}

	logrus.Warnf("%v", r)
	debug.PrintStack()

}
