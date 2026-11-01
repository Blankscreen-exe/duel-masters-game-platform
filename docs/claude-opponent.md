# Playing against Claude

Claude can play you on your own computer, from [Claude Code](https://claude.com/claude-code). It
builds its own deck, plays through the same seat as a computer opponent and can talk to you while
it plays. It uses your Claude plan and needs no API key.

This is for a site running on your own computer. It is off by default, and the connection Claude
uses can only be reached from the computer the server runs on.

## Setting it up

1. Start the site as usual. In development the server also listens for Claude Code on
   `127.0.0.1:8081`. Set `LOCAL_PLAYER_ADDR` to use another port, and change `.mcp.json` to match.
   In production nothing listens unless `LOCAL_PLAYER_ADDR` is set.
2. As an admin, open **Admin → Settings → Computer opponents** and turn on
   `computer.claude_opponent`.
3. Open Claude Code in this repository and approve the `duel-masters` server when it asks.
   `.mcp.json` connects it to the site's server. If that server was not running when Claude Code
   started, run `/mcp` there to reconnect.
4. So it does not stop to ask at every move, allow the game's tools, either by choosing "don't ask
   again" for each tool the first time it uses it, or once with `/permissions` by adding the rule
   `mcp__duel-masters`.

## Playing

1. Tell Claude Code something like "let's duel". You can ask for a format or a kind of deck.
2. Claude picks a format, builds a deck and waits for a game.
3. On the **Play** page, open **Vs computer**, pick **Claude** and one of your decks legal in its
   format, then press play. Reload the page if Claude is not listed yet.

Claude is listed only while the setting is on, it has a playable deck, Claude Code has been in touch
in the last few minutes and it is not already in a game. Ask for a new deck between games whenever
you like.

## What to expect

- **It is slow.** Claude thinks about every decision, so a move can take from a few seconds to about
  half a minute, and a whole game much longer than against the other opponents.
- **Games are practice:** they are private, are shown in your history and never change ratings.
- **Rematches** work as for other computer opponents, as long as Claude Code is still connected.
- **If Claude Code closes during a game,** the game is given up after 15 minutes without contact, so
  you are never stuck. You can also resign at any time.

## Fair play

The game's tools only tell Claude what a player in its seat could know: not your hand, your shields
or the order of any deck. Claude Code can also read files on your computer, including your deck
list in the database. Its instructions tell it to use only the game's tools, but nothing enforces
that outside the tools, so this relies on Claude following them.

## Tools

| Tool              | What it does                                                       |
| ----------------- | ------------------------------------------------------------------ |
| `status`          | Whether a game is under way, and its current deck and format       |
| `list_formats`    | The formats and their rules                                        |
| `find_cards`      | Searches the cards the game can play                               |
| `save_deck`       | Saves its deck for the next game, or says what is wrong with it    |
| `wait_for_turn`   | Waits until it is its move, returning after about 40 seconds       |
| `coin_toss`       | Calls the toss, or chooses who plays first                         |
| `make_move`       | Charges mana, plays a card, attacks, uses a tap ability, ends turn |
| `answer_question` | Answers a question: paying mana, targets, blocking and so on       |
| `say`             | Says something in the game's chat                                  |
| `resign`          | Gives up the game                                                  |
