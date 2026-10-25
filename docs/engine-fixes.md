# Engine fix plan

Working list from a code analysis of the rules engine at upstream `duel-masters` commit
`e5443966`, before it was copied here. We work through it one item at a time. Paths are relative
to `apps/server/internal/engine/`. Line numbers are from that commit and drift as fixes land.

**Verified** means confirmed by reading the code. **Reported** means a review found it and it still
needs checking before we fix it.

Every fix follows [`apps/server/internal/engine/CONTRIBUTING.md`](../apps/server/internal/engine/CONTRIBUTING.md):
add a test that reproduces the bug first, then fix it and run, from `apps/server`:

```text
go test ./... -count=1 -timeout 120s
go test -race ./... -count=1 -timeout 180s   # for concurrency/engine items
```

---

## Phase 1: small, safe engine fixes

### [x] 1. Spectator lock taken twice in `BroadcastState` (Verified)
- **Status:** Fixed before the engine was copied here. Test: `tests/spectator_test.go`.
- **Where:** `game/match/match.go` (`broadcastState`)
- **Problem:** `m.spectators.RLock()` was followed by `m.spectators.Iter()`, which calls `RLock()` again. Go's `RWMutex` does not allow a goroutine to take a read lock it already holds. If `Add`/`Remove` (a spectator joining or leaving) asked for the write lock between the two calls, the event loop deadlocked. The lock was also held during `Socket.Send` (network I/O, up to a 10s write deadline).
- **Fix:** Removed the outer `RLock`/`RUnlock`. `Iter()` already returns a copy.

### [ ] 2. Global `seqID` shared by all matches (Verified)
- **Where:** `game/match/persistenteffect.go:3,13,22`
- **Problem:** A package-level `var seqID` is read and incremented without synchronization from every match's event loop goroutine. That is a data race.
- **Fix:** Make it a per-match counter field (for example `nextPersistentEffectID int` on `Match`). It is only touched from that match's event loop.
- **Test:** Run two scenario matches in parallel that each add persistent effects, under `-race`.

### [ ] 3. `BreakShields` can index an empty slice (Verified)
- **Where:** `game/match/match.go:310-314`
- **Problem:** `shields := event.Cards` and then `shields[0]`. Handlers may shrink `event.Cards`. Mecha del Sol and Initiate avoid leaving it empty by cancelling the event themselves, but the engine itself does not check.
- **Fix:** `if len(shields) == 0 { return }` after the cancel check.
- **Test:** A test handler or card setup that removes every shield from the event without cancelling it. Expect no panic and the turn to continue.

---

## Phase 2: concurrency in the socket/match layer

### [x] 4. `join_match` runs as a ParallelEvent and mutates match state (Verified)
- **Status:** Fixed. Joining stays a ParallelEvent, so a player coming back is seated even while the event loop waits on a prompt. The seats, the coin toss and the disconnect timers are now guarded by `setupMu` (`game/match/lifecycle.go`), which is never held while sending to or closing a socket. The toss begins once (`tossBegun`) and the game starts once (`turnChosen`). Tests: `TestPlayersJoiningAtTheSameMomentStartOneToss` and the other tests in `tests/lifecycle_test.go`, under `-race`.
- **Left:** a join's state broadcast still reads `m.Step` and the zones unsynchronized; see item 24.

### [x] 5. `OnSocketClose` reads player/socket fields without synchronization (Reported)
- **Status:** Fixed. `OnSocketClose` reads the seats under `setupMu` (`Match.leave`), and the match's started flag is atomic (`started`), so the ticker reads it safely.

### [x] 6. Reconnect can end the match (Reported)
- **Status:** Fixed. The new socket takes the seat before the old one is closed, and closing a socket that is no longer a player's changes nothing. Test: `TestComingBackWhileTheOpponentIsAwayKeepsTheMatch`.

### [ ] 7. Stale action answers get delivered to the next prompt (Verified)
- **Where:** `game/match/match.go:1779-1833`
- **Problem:** An `action` message sent while no prompt is open blocks forever on the unbuffered `p.Player.Action <- msg`. The next prompt, even an unrelated one, receives it, and a stale `Cancel: true` cancels it. When the match is disposed, that blocked send panics "send on closed channel", which `recoverEvent` logs as a real fault.
- **Fix idea:** Only deliver an action when the player has an open, unresolved prompt (`ActionState`). Otherwise reject it with a warning and do not block. Make sure the disposal path does not log a stack trace.
- **Test:** Send an action with no prompt open, then open a cancellable prompt. It must not be auto-cancelled. Dispose the match afterwards and check that no goroutine leaks.

### [x] 23. Scenario actions are dropped by the event loop's one-slot queue (Verified)
- **Status:** Fixed. Tests: `game/match/eventloop_test.go`; the card tests below.
- **Where:** `game/match/eventloop.go` (`schedule`, `start`), `tests/scenario/scenario.go`, `tests/cards/helpers_test.go`
- **Problem:** Sequential events go through a channel with one slot. `schedule` drops an event when the slot is taken, and the loop drains the slot after every event, dropping whatever arrived meanwhile. The test harness sent actions while the loop was still busy, in two ways:
  - `waitForIdleLoop` probed the loop with harmless `attack_player` messages, which took that slot too: a probe delivered late pushed out the test's next real action.
  - `answerInTurn` and `cancelInTurn` returned as soon as any message followed the answer, often a state update, while the event that had waited on the prompt was still running; the next action was then dropped. `ActionUseTapAbility` could even accept that state update as its own answer.
- **Fix:**
  - The event loop counts the sequential events scheduled and not yet over (queued, running, or waiting on a prompt), and stops counting an event when it is dropped; `Match.HasPendingEvents` reports it. Checking takes no place in the queue.
  - `waitForIdleLoop` waits on that count instead of probing.
  - `answerInTurn` and `cancelInTurn` use the new `TestScenario.Mark` and `WaitForAnswerToSettle`: they return once the loop is idle, any player has been sent a new prompt (`action`, `show_cards_non_dismissible`) or told to wait on one, or the answer was refused.
  - The engine still drops real players' actions that arrive while it is busy; whether it should is part of item 7.
- **Measured under `-race`:** `TestSpinalParasite` failed 8 times in 200 runs before, 0 in 200 after. `TestBodaciousGiant`, `TestForcedFrenzy`, `TestQuillspikeRumbler`, `TestCometEyeTheSpectralSpud`, `TestEmergencyTyphoon`, `TestLuckyBall` and `TestWilyCarpenter`: 0 failures in 100 runs each. The whole server suite: 5 runs without a failure or race.

---

## Phase 3: card rules bugs

### [ ] 8. Raging Dash-Horn removes double breaker granted by other sources (Verified)
- **Where:** `game/cards/dm03/horned_beast.go:33`
- **Problem:** `card.RemoveCondition(cnd.DoubleBreaker)` also removes the one granted by Blaze Cannon or Snake Attack (`game/cards/dm03/spells.go:241,266`).
- **Fix:** `RemoveConditionBySource`/`RemoveSpecificConditionBySource` with `card.ID`, plus `AddUniqueSourceCondition` for the add.
- **Test:** Dash-Horn with non-nature mana and Snake Attack cast. It still double-breaks.

### [ ] 9. Sol Galla gains power while not in the battle zone (Verified)
- **Where:** `game/cards/dm08/guardian.go:26`
- **Problem:** `fx.AnySpellResolved` has no zone guard, so the bonus builds up while the card is in hand or deck.
- **Fix:** Check `card.Zone == match.BATTLEZONE` (inside the scheduled callback too). Also re-check the printed text for the exact duration.
- **Test:** Cast a spell, then summon Sol Galla the same turn. Base power only. Cast a spell with it in play: +3000 until end of turn.

### [ ] 10. `eachPlayerDestroysMana` limits the opponent by the caster's mana count (Verified)
- **Where:** `game/fx/destroy_effects.go:17-44`
- **Problem:** The shared `quantity` is lowered inside the per-player loop.
- **Fix:** Use a per-player local `n := min(quantity, len(mana))`.
- **Affects:** Machine Eater, `game/cards/dm02/rock_beast.go:51`, `game/cards/dm07/dragonoid.go:18`.
- **Test:** Caster has 1 mana and opponent has 5, effect quantity 2. The opponent loses 2.

### [ ] 11. Kalute: wrong battle-zone scope and inverted chat message (Reported)
- **Where:** `game/cards/dm09/initiate.go:24-38`
- **Problem:** Only checks its own battle zone, while the text says "another Kalute in the battle zone". Reports to chat only when `err != nil`. Tries the move once per copy.

### [ ] 12. Lamiel draws before destruction is final (Reported)
- **Where:** `game/cards/dm11/berserker.go:21-36`
- **Problem:** Uses the pre-destruction `CreatureDestroyed` event, so it still draws when a replacement effect (Amnis, Kalute, Balloonshroom) prevents the destruction.
- **Fix:** Observe the battle-zone-to-graveyard `CardMoved` instead.

### [ ] 13. Brutal Charge: wrong expiry and wrong break count (Reported)
- **Where:** `game/cards/dm05/spells.go:164-205`, `game/fx/quality_of_life.go:873`
- **Problem:** Uses `EndOfMyTurn` instead of "end of this turn". When cast as a shield trigger it waits until the caster's next turn. `ShieldBroken` counts any broken shield, not just shields "your creatures broke". Miracle Quest has the same TODO.

### [ ] 14. Snip Striker Bullraizer and `fx/cant_attack.go` remove other sources' attack restrictions (Reported)
- **Where:** `game/cards/dm03/dragonoid.go:44-45`, `game/fx/cant_attack.go:47-48`
- **Fix:** Source-tagged add/remove.

### [ ] 15. Miracle Portal adds a condition on every event, with a `*Card` source (Reported)
- **Where:** `game/cards/dm07/spells.go:208`
- **Fix:** Add it once with `AddUniqueSourceCondition` and source `card.ID`, and remove it on expiry.

### [ ] 16. Jack Viper and Dava Torey trigger more narrowly or broadly than printed (Reported)
- **Where:** `game/cards/dm03/ghost.go:26`, `game/cards/dm06/mecha_thunder.go:33`
- **Problem:** Jack Viper only handles destruction, but the text covers any "put into graveyard from the battle zone". Dava Torey triggers on any hand-to-graveyard move instead of only on discards.

### [ ] 17. Cancelled `MoveCard` results are ignored (Reported)
- **Where:** `game/match/match.go:271` (`Match.MoveCard` reports to chat even when the move was cancelled), `game/fx/return_to.go:11,49,69`, `game/cards/dm05/balloon_mushroom.go:32-35`, and about 160 of 222 call sites.
- **Fix:** Fix the shared helpers first, then the cards, checking `card.Zone` after the move.

### [ ] 18. `fx/creature.go` returns without `CloseAction` (Reported)
- **Where:** `game/fx/creature.go:~268` (`GetCard` failure path)

---

## Phase 4: engine hardening

### [x] 19. Admin/cheat commands open to everyone in local dev (Verified)
- **Status:** Fixed. Chat commands run only when the match system is created with `SystemOptions.DevCommands`, which is off by default, rather than whenever the `local_dev` environment variable is set. The server never turns it on; it is for local testing tools.

### [ ] 20. 512-byte websocket read limit (Verified)
- **Where:** `server/socket.go:18`
- **Problem:** A large multi-card `action` message (about 35+ IDs) closes the socket.
- **Fix:** Raise the limit to a bound derived from the maximum selection size.

### [ ] 21. No per-socket rate limiting (Reported)
- **Where:** `Match.Parse` / `eventloop.schedule` spawns a goroutine per message. Chat is unthrottled.
- **Fix:** Throttle per connection in the new server's `realtime/` layer, before messages reach the engine.

### [ ] 22. Game creation trusts its input (Reported)
- **Where:** `game/match/player.go:216-223` (`CreateDeck`)
- **Problem:** Deck size and card IDs are not checked.
- **Fix:** The new server checks decks with its single legality check (`decks/`) before creating a game; the engine still rejects unknown card IDs instead of building a partial deck.

### [ ] 24. A player's socket is swapped while the game reads it (Verified)
- **Where:** `game/match/lifecycle.go` (`join` sets `PlayerReference.Socket` under `setupMu`); readers across the engine, card effects included, use `PlayerRef(...).Socket` without the lock.
- **Problem:** when a player comes back during a game, their new socket is written while the event loop may be sending to, or reading the user from, the old one. `Broadcast` now reads the seats under the lock, but the many direct readers do not. The tests do not show it, since they reconnect while the event loop is idle, but it is a real race in play.
- **Fix idea:** keep the socket behind a method on `PlayerReference` that reads it under a lock (or an `atomic.Pointer`), and move every reader to it. It touches many card files, so it is its own change.
- **Test:** reconnect a player while the event loop broadcasts state in a loop, under `-race`.

---

## Requirements for the new server

These came from the original project's standalone web server (`api/`, `cmd/`, `moderation/`),
which was not copied here. They are not engine fixes, but the new server must not repeat them:

- **HTTP timeouts:** set `ReadHeaderTimeout`, `ReadTimeout`, `IdleTimeout` and `MaxHeaderBytes`.
- **Websocket origins:** check the `Origin` header against an allowed list from configuration
  (the original accepted any origin). This matters more with cookie sessions, which a foreign page
  could otherwise ride.
- **Request size limits:** wrap request bodies in `http.MaxBytesReader`.
- **CORS:** never combine `*` with credentials; with Next.js forwarding to Go on one origin, no
  CORS is needed at all.
- **Upgrade errors:** do not write a second response after a failed websocket upgrade.
- **Shutdown:** shut down gracefully. Running games are ended and recorded, never killed by
  `os.Exit`.
- **Shared data:** a function documented as returning a copy must return a copy; JSON and other
  errors are never ignored.

---

## Not a fix, but tracked

- **Running race tests on Windows without a C compiler:** run them in Docker with the host module cache (mount `go env GOMODCACHE` to `/go/pkg/mod`, `GOPROXY=off`, `GOFLAGS=-mod=mod`). Check `git status` afterwards: `go.mod` and `go.sum` must not change.
- **Card test coverage gaps:** DM07 0/60, DM06 9/117, DM09 12/60, DM02–05 and DM08 have only a handful each.
- **`HandleFx` cost:** runs every handler of every card in every zone, decks included, for every event, including the `GetPowerEvent` fired by each `GetPower` call. Profile before changing anything.
- **Possible fx helpers:** "until end of turn" persistent effects (about 22 copies), the "would be destroyed → move elsewhere instead" replacement, and the copied `Wait`/`EndWait` block in the select helpers.
