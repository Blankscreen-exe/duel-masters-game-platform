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

type EventLoop struct {
	stopped sync.Once
	running atomic.Bool
	events  chan func()
	exit    chan struct{}

	// pending counts the sequential events scheduled and not yet over: on their
	// way into the queue, queued, or running. An event the loop drops instead
	// of running stops counting when it is dropped.
	pending atomic.Int64
}

func NewEventLoop() *EventLoop {
	el := &EventLoop{
		events: make(chan func(), 1),
		exit:   make(chan struct{}),
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
			// events is deliberately left open. Senders schedule onto it from
			// their own goroutines, so closing it here would turn a scheduled
			// event into a send on a closed channel.
			return
		case event := <-el.events:
			el.process(event)

			// drain the channel
			for len(el.events) > 0 {
				<-el.events
				// The dropped event never runs, so it stops counting here.
				el.pending.Add(-1)
			}
		}
	}
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

func (el *EventLoop) schedule(event func(), strategy EventExecutionStrategy) {

	// Counted before the goroutine starts, so that by the time schedule
	// returns, the event shows as pending until it has run or been dropped.
	if strategy == SequentialEvent {
		el.pending.Add(1)
	}

	go func() {
		// A parallel event should never open a prompt, but it runs on its own
		// goroutine, so an abort raised by one has no other boundary to reach.
		defer recoverEvent()

		switch strategy {
		case ParallelEvent:
			event()
			return

		case SequentialEvent:
			queued := func() {
				// Deferred, so that an event ending in a panic, such as one
				// abandoned when the match is disposed, also stops counting.
				defer el.pending.Add(-1)

				event()
			}

			select {
			case el.events <- queued:
			default:
				el.pending.Add(-1)
				logrus.Debug("Skipped an incoming event")
			}
		}
	}()

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
