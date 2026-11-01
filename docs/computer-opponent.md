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

### Starting set

Confirmed, and seeded at startup (`internal/opponents`): each plays at Normal strength, in the
Open format, with creatures, evolution creatures and spells, and four copies of each card listed.

| Name        | Theme and style                                           | Deck rules              | Deck                                                                                                                                                     |
| ----------- | --------------------------------------------------------- | ----------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Grimwhisper | Hand discard (`discard`): empties your hand first         | one civilization        | Ghost Touch, Horrid Worm, Cloned Nightmare, Windmill Mutant, Gigabalza, Locomotiver, Masked Horror, Jagila, Zero Nemesis, Gigavrand                       |
| Blazerush   | Rapid summon (`rush`): cheap creatures, attacks every turn | one civilization        | Deadly Fighter Braid Claw, Brawler Zyler, Immortal Baron Vorg, Mini Titan Gett, Crimson Hammer, Explosive Dude Joe, Pyrofighter Magnus, Rikabu, Fatal Attacker Horvath, Twin-Cannon Skyterror |
| Aegis       | Defensive wall (`wall`): blockers, attacks when safe      | one civilization        | La Ura Giga, Emerald Grass, Iocant, Senatine Jade Tree, Bex, Dia Nork, Engbelt, Szubs Kin, Syforce, Sasha                                                 |
| Stormbinder | Tapper (`tapper`): taps your creatures, then attacks      | up to two civilizations | Solar Ray, Aqua Trickster, Miele, Larba Geer, Mizoy, Moonlight Flash, Flohdani, Rom, Holy Awe, Craze Valkyrie                                             |
| Tidecaller  | Bounce (`bounce`): sends your creatures back to your hand | one civilization        | Spiral Gate, Stained Glass, Wave Lance, Emperor Maroll, Unicorn Fish, Cloned Spiral, Crystal Paladin, Saucer-Head Shark, Aqua Surfer, King Mazelan        |
| Titangrove  | Big creatures (`giant`): ramps mana, then giants          | up to two civilizations | Poisonous Mushroom, Faerie Life, Bronze-Arm Tribe, Ultimate Force, Bolshack Dragon, Carnival Totem, Roaring Great-Horn, Gatling Skyterror, Dawn Giant, Bodacious Giant |

Each plays under a computer account of its own (a `computer` flag on the account, which nobody
logs in to and admins cannot change), so its games are recorded like anyone's. A name an account
already has is left out of the seed rather than taken.

## Playing against Claude

Separate from the personalities, Claude can play live from Claude Code on the computer the server
runs on, with decks it builds itself. See [Playing against Claude](claude-opponent.md).

## Settings

Admin settings: how long the computer pauses before each move (`computer.pause_milliseconds`), and
the most games against the computer running at once (`computer.max_games`), so a busy evening
cannot overload the server.

## Details settled while building

- **Seats:** the player hosts; the computer takes the guest's seat at once, over the same path a
  browser's connection comes in by. It is not held to one game at a time: every personality can
  play any number of players at once, up to the limit.
- **No profile page:** a computer opponent has none, as it would show the games of everyone who
  practised against it. The history shows its name without a link.
- **Rematch:** the computer has always agreed. If its opponent is retired, or its deck stops being
  legal, while a player decides, the rematch ends as if it had declined.

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
3. **Personalities** (done). Stored in the database with their rules and decks, the starting set
   seeded, their play styles (`internal/opponents`, and the styles and strengths in
   `internal/computer`, tested by the six playing whole games in their styles). Games against them
   start through `opponents.Challenges` and the API's `/computer-opponents`, run by the games
   service like any other, and end in the match history, marked, without touching ratings
   (`match.Options.Practice`).
4. **The "Vs computer" tab** on the play page (done). Pick a personality and a deck legal in its
   format, in a row like the other tabs', so the page still fits the screen.
5. **Admin panel** (done). The computer opponents page: add, change, retire and bring back, with the
   deck built in the players' deck builder. An offered opponent must have a deck it can play; a
   retired one's may be unfinished, so a new one starts retired. Retired, not deleted, as its games
   stay in players' histories. The two settings are on the admin panel's settings page.
6. **Later:** the look-ahead and a Hard strength.
