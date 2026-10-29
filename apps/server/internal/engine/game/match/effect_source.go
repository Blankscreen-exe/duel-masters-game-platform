package match

// The cards whose effects are running, innermost last, so a question can
// tell which card asks (Match.promptAbout). Effects run on the match's event
// loop, one inside another as they set off further events, so a stack kept
// on the match follows them.

// effectSource is the card whose effect is running now, or nil when none is,
// such as while the rules themselves ask a player something.
func (m *Match) effectSource() *Card {
	if m == nil || len(m.effectSources) == 0 {
		return nil
	}
	return m.effectSources[len(m.effectSources)-1]
}

// asEffectOf returns f to run as an effect of source: a callback scheduled
// by a card, or a persistent effect a card added, still runs as that card's.
func (m *Match) asEffectOf(source *Card, f func()) func() {
	if m == nil {
		return f
	}
	return func() {
		depth := m.enterEffect(source)
		defer m.leaveEffect(depth)
		f()
	}
}

// enterEffect marks source's effect as running and returns the depth to go
// back to with leaveEffect once it ends.
func (m *Match) enterEffect(source *Card) int {
	depth := len(m.effectSources)
	m.effectSources = append(m.effectSources, source)
	return depth
}

// leaveEffect ends the effects entered since depth. It also undoes those an
// abandoned effect, unwound by a panic, left behind.
func (m *Match) leaveEffect(depth int) {
	clear(m.effectSources[depth:])
	m.effectSources = m.effectSources[:depth]
}
