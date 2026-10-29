package match

type PersistentHandlerFunc func(ctx2 *Context, exit func())

type PersistentEffect struct {
	exit   func()
	effect PersistentHandlerFunc
}

// ApplyPersistentEffect adds an effect that sees every event of the match
// until it calls exit. Each effect is keyed by a number from the match's own
// counter, so matches running at the same time never share one.
func (match *Match) ApplyPersistentEffect(f PersistentHandlerFunc) {
	match.lastPersistentEffectID++
	key := match.lastPersistentEffectID

	fx := PersistentEffect{
		exit:   func() { match.RemovePersistentEffect(key) },
		effect: f,
	}

	match.persistentEffects[key] = fx
}

func (match *Match) RemovePersistentEffect(id int) {
	_, ok := match.persistentEffects[id]
	if ok {
		delete(match.persistentEffects, id)
	}
}
