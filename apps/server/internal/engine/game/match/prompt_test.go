package match

import (
	"testing"
	"time"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
)

const promptTestTimeout = 2 * time.Second

// askedPrompt is a prompt with a question open, and the question's number.
func askedPrompt(t *testing.T) (*prompt, int) {
	t.Helper()

	p := newPrompt()
	question := &server.ActionMessage{Header: "action"}
	p.ask(question)

	return p, question.Prompt.ID
}

// takeSoon takes an answer, failing the test if none comes in time.
func takeSoon(t *testing.T, p *prompt) (PlayerAction, bool) {
	t.Helper()

	type taken struct {
		answer PlayerAction
		ok     bool
	}
	done := make(chan taken, 1)
	go func() {
		answer, ok := p.take()
		done <- taken{answer, ok}
	}()

	select {
	case result := <-done:
		return result.answer, result.ok
	case <-time.After(promptTestTimeout):
		// Ends the wait, so the goroutine does not outlive the test.
		p.dispose()
		t.Fatal("no answer was taken")
		return PlayerAction{}, false
	}
}

func TestPromptNumbersEachQuestion(t *testing.T) {
	p := newPrompt()

	first := &server.ActionMessage{}
	second := &server.ShowCardsMessage{}
	p.ask(first)
	p.ask(second)

	if first.Prompt.ID != 1 || second.Prompt.ID != 2 {
		t.Fatalf("questions are numbered in order, got %d and %d", first.Prompt.ID, second.Prompt.ID)
	}

	if question, open := p.pending(); !open || question != second {
		t.Fatal("the latest question is the open one")
	}
}

func TestPromptKeepsAnAnswerGivenBeforeTheWait(t *testing.T) {
	p, id := askedPrompt(t)

	if !p.give(PlayerAction{Prompt: id, Count: 3}) {
		t.Fatal("an answer to the open question is kept")
	}

	answer, ok := takeSoon(t, p)
	if !ok || answer.Count != 3 {
		t.Fatalf("the kept answer is taken, got %+v", answer)
	}
}

func TestPromptWakesTheWaitWhenAnswered(t *testing.T) {
	p, id := askedPrompt(t)

	go func() {
		time.Sleep(10 * time.Millisecond)
		p.give(PlayerAction{Prompt: id, Cancel: true})
	}()

	answer, ok := takeSoon(t, p)
	if !ok || !answer.Cancel {
		t.Fatalf("the answer reaches the wait, got %+v", answer)
	}
}

func TestPromptRefusesAnswersToAnyOtherQuestion(t *testing.T) {
	p, id := askedPrompt(t)

	for _, wrong := range []int{0, id - 1, id + 1} {
		if p.give(PlayerAction{Prompt: wrong}) {
			t.Fatalf("an answer naming question %d is refused while %d is open", wrong, id)
		}
	}

	p.close()
	if p.give(PlayerAction{Prompt: id}) {
		t.Fatal("an answer to a closed question is refused")
	}
	if _, open := p.pending(); open {
		t.Fatal("a closed question is not pending")
	}
}

func TestPromptKeepsTheFirstOfTwoAnswers(t *testing.T) {
	p, id := askedPrompt(t)

	p.give(PlayerAction{Prompt: id, Count: 1})
	if p.give(PlayerAction{Prompt: id, Cancel: true}) {
		t.Fatal("a second answer before the first is taken is refused")
	}

	answer, _ := takeSoon(t, p)
	if answer.Cancel || answer.Count != 1 {
		t.Fatalf("the first answer stands, got %+v", answer)
	}
}

func TestPromptAsksAgainAfterARefusedAnswer(t *testing.T) {
	p, id := askedPrompt(t)

	p.give(PlayerAction{Prompt: id, Count: 1})
	takeSoon(t, p)

	// The prompt warned the player and waits again on the same question.
	if !p.give(PlayerAction{Prompt: id, Count: 2}) {
		t.Fatal("the question stays open for another answer")
	}
	answer, _ := takeSoon(t, p)
	if answer.Count != 2 {
		t.Fatalf("the new answer is taken, got %+v", answer)
	}
}

func TestPromptDropsTheAnswerOfAReplacedQuestion(t *testing.T) {
	p, id := askedPrompt(t)
	p.give(PlayerAction{Prompt: id, Cancel: true})

	next := &server.ActionMessage{}
	p.ask(next)
	p.give(PlayerAction{Prompt: next.Prompt.ID, Count: 7})

	answer, _ := takeSoon(t, p)
	if answer.Cancel || answer.Count != 7 {
		t.Fatalf("only the new question's answer is taken, got %+v", answer)
	}
}

func TestPromptDisposalEndsTheWait(t *testing.T) {
	p, _ := askedPrompt(t)

	go func() {
		time.Sleep(10 * time.Millisecond)
		p.dispose()
	}()

	if _, ok := takeSoon(t, p); ok {
		t.Fatal("a disposed prompt has no answer")
	}

	// Disposing again, as both players' references may, is harmless.
	p.dispose()
}
