package match

import (
	"encoding/json"
	"slices"
	"sync"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"

	"github.com/sirupsen/logrus"
)

// prompt is the question a player is being asked, if any, and the answer they
// have given to it.
//
// The event loop opens a question, sends it, and waits for its answer; answers
// arrive from the player's connection, on another goroutine, whenever the
// player sends them. Each question a player is asked is numbered, and an answer
// names the number of the question it answers. An answer naming any other
// question, such as a second click on a question already settled or one that
// arrives after its question was closed, is refused, so it can never be taken
// for the answer to whatever the player is asked next. An answer may arrive
// before the event loop starts waiting for it; it is kept until it is taken.
//
// Giving an answer never waits: the answer is kept and the waiting side woken.
// The lock is only held to read or change these fields, never while sending,
// waiting or doing anything else.
type prompt struct {
	mu sync.Mutex

	// last is the number of the latest question, and open whether it still
	// waits for an answer.
	last int
	open bool

	// message is the open question as it was sent, for sending again to a
	// player who comes back.
	message server.Prompting

	// answer is the answer given to the open question and not yet taken.
	answer *PlayerAction

	// answered wakes the side waiting for an answer. It holds one wake-up, so
	// that giving an answer never blocks.
	answered chan struct{}

	// disposed is closed when the match is, which ends every wait.
	disposed    chan struct{}
	disposeOnce sync.Once
}

func newPrompt() *prompt {
	return &prompt{
		answered: make(chan struct{}, 1),
		disposed: make(chan struct{}),
	}
}

// ask opens a question, numbering message, which the caller then sends. A
// question still open is replaced, along with any answer given to it.
func (p *prompt) ask(message server.Prompting) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.last++
	message.Number(p.last)
	p.open = true
	p.message = message
	p.answer = nil
	p.clearWakeUp()
}

// close settles the open question. Answers given to it from now on are
// refused.
func (p *prompt) close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.open = false
	p.message = nil
	p.answer = nil
	p.clearWakeUp()
}

// clearWakeUp drops a wake-up left for an answer that is no longer kept. The
// caller holds the lock.
func (p *prompt) clearWakeUp() {
	select {
	case <-p.answered:
	default:
	}
}

// pending returns the open question, if there is one.
func (p *prompt) pending() (server.Prompting, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.message, p.open
}

// give keeps an answer to the open question for the event loop to take. It
// reports false, keeping nothing, for an answer naming a question that is not
// the open one, and for a second answer given before the first was taken: the
// first answer stands.
func (p *prompt) give(answer PlayerAction) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.open || answer.Prompt != p.last || p.answer != nil {
		return false
	}

	p.answer = &answer
	select {
	case p.answered <- struct{}{}:
	default:
	}

	return true
}

// take waits for an answer to the open question and returns it, leaving the
// question open, as a prompt that refuses an answer asks again. It reports
// false once the match has been disposed.
func (p *prompt) take() (PlayerAction, bool) {
	for {
		select {
		case <-p.disposed:
			return PlayerAction{}, false
		default:
		}

		p.mu.Lock()
		answer := p.answer
		p.answer = nil
		p.mu.Unlock()

		if answer != nil {
			return *answer, true
		}

		select {
		case <-p.answered:
		case <-p.disposed:
			return PlayerAction{}, false
		}
	}
}

// dispose ends every wait for an answer, for good.
func (p *prompt) dispose() {
	p.disposeOnce.Do(func() {
		close(p.disposed)
	})
}

// answer takes a player's answer from their connection and gives it to the
// question it names. An answer to a question no longer open is dropped
// quietly: the player has already been told the question is closed, or is
// about to be.
func (m *Match) answer(s *server.Socket, data []byte) {
	defer recoverEvent()

	p, err := m.PlayerForSocket(s)

	if err != nil {
		return
	}

	msg := PlayerAction{}

	if err := json.Unmarshal(data, &msg); err != nil {
		m.ActionWarning(p.Player, "Invalid selection")
		return
	}

	// Check to see if the client is trying something fishy with selecting the same card multiple times
	if len(slices.Compact(slices.Sorted(slices.Values(msg.Cards)))) != len(msg.Cards) {
		m.ActionWarning(p.Player, "You cannot select the same card multiple times")
		return
	}

	if !p.Player.prompt.give(msg) {
		logrus.Debugf("Dropped an answer from %s to question %d, which is not open", p.Username, msg.Prompt)
	}
}
