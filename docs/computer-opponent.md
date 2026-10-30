# Computer opponent plan

Players can practise against the computer, which plays with ready-made themed decks. This is the
plan; it is built in the slices at the end, each usable on its own.

## Decided

- **How it thinks:** rules of thumb plus scoring every legal move first. A limited look-ahead (a
  few possible futures, within a time limit) comes later, for stronger personalities.
- **Personalities:** each has its own play style and its own deck, so a player can try their deck
  against different kinds of opponent. Each personality is built around a theme, such as making
  you discard, summoning fast, defending, or tapping your creatures.
- **Strength:** each personality plays at one strength; players do not choose a difficulty.
- **Deck rules:** each personality has rules its deck must follow, such as one civilization or two,
  or whether castles are allowed.
- **Managed by:** a starting set is seeded, then admins add, edit, retire and rebuild them in the
  admin panel.
- **Records:** games against the computer appear in the player's match history, marked as against
  the computer, and never change ratings.
- **Who and where:** signed-in players only, from a third "Vs computer" tab beside Quick match and
  Open a game on the play page.
- **Private:** these games are not listed in the lobby's games list and cannot be watched.
- **Built after** engine fixes 2, 3, 21 and 24 (done).

## How the computer plays

The computer joins a game as a player, inside the Go server, over a connection like a browser's. It
is sent exactly what a player in its seat is sent and answers with the same moves, so the engine
treats it like anyone else and it never sees hidden cards: your hand, your shields or the order of
any deck. What a card does is public, so it looks up each card it sees in the card database: cost,
civilization, power and abilities.

The coin toss goes as in every game, animated; the computer calls or chooses at once. It pauses
briefly before each move so the player can follow what it does.

On its turn it works through the same steps as a player:

1. **Mana:** charges the card it least needs, keeping the civilizations its hand needs.
2. **Summon and cast:** plays the best cards it can afford, paying with mana that leaves the most
   options open.
3. **Attack:** attacks a player or a creature when the attack is worth it, weighing blockers and
   the chance of shield triggers.
4. **End the turn.**

On your turn it answers what it is asked: whether to block, which shield triggers to use, and the
choices card effects ask of it.

**Scoring:** every legal move is given a score from the board it would leave: creatures and their
power, shields on both sides, cards in hand, mana. It plays the best. A personality's style changes
the weights: a rush personality values breaking shields over keeping creatures; a defensive one
values blockers and its own shields.

**Questions from card effects:** the engine now asks every question in words for people. It will
also say what kind of question it is (block, shield trigger, pay mana, choose a target, discard...)
and whose cards are offered, so the computer can answer any card's question without special code.
A personality can also be given advice for particular cards where the general answer is not good
enough.

## Personalities

Each has:

- **a name and a short description**, shown when choosing one;
- **a theme and play style:** what it values (see Scoring);
- **a strength:** Easy, which now and then makes a deliberate small mistake, or Normal, which always
  plays its best move. Hard comes with the look-ahead;
- **deck rules:**
  - a **format** (its allowed sets and ban list, as for players' games);
  - **civilizations:** exactly one, up to two, or any;
  - **card types allowed:** creatures, evolution creatures and spells, and castles and Cross Gear,
    which become usable once the engine plays them;
- **its deck**, which must follow its rules, built with the deck builder in the admin panel.

The player picks a personality and one of their own decks that is legal in the personality's format.
Rematches work as for other games, with deck switching; the computer always accepts.

### Starting set (proposal, to confirm)

| Personality theme | Plays like                                                        | Deck rules            |
| ----------------- | ----------------------------------------------------------------- | --------------------- |
| Hand discard      | makes you discard, then grinds you down                           | one civilization      |
| Rapid summon      | many cheap creatures, attacks every turn                          | one civilization      |
| Defensive wall    | blockers first, attacks only when it is safe                      | one civilization      |
| Tapper            | taps your creatures so they cannot block, then attacks            | up to two             |
| Bounce            | sends your creatures back to your hand to slow you down           | one civilization      |
| Big creatures     | charges mana fast and plays the largest creatures it can          | up to two             |

Names, descriptions and deck lists come with the deck building, using cards that are implemented
and tested, so the computer plays them correctly.

## Settings

Admin settings: how long the computer pauses before each move, and the most games against the
computer running at once, so a busy evening cannot overload the server.

## Build order

1. **The engine says what each question is about** (done). Every question tells its kind, the
   card whose effect asks when the player may know it, and which cards offered are the
   opponent's (`server.PromptAbout`).
2. **The computer plays a whole game** (done: `internal/computer`). Rules of thumb and scoring, in
   Go, tested by having it play complete games against itself with each starter deck: every game
   must finish, won by the game's rules, with no stalled questions, under the race detector. It
   knows cards by their keywords and categories in the card database. A move or answer the engine
   refuses is not repeated: it cancels, tries other choices, attacks with whatever can when the
   engine will not end its turn, and as a last resort gives up the game rather than leave its
   opponent waiting.
3. **Personalities.** Stored in the database with their rules and decks, the starting set seeded,
   their play styles, and games against them started, recorded in the match history and kept out
   of ratings.
4. **The "Vs computer" tab** on the play page.
5. **Admin panel:** the computer opponents page and the two settings.
6. **Later:** the look-ahead and a Hard strength.
