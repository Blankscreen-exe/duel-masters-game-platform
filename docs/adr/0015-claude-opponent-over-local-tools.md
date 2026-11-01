# 15. Claude plays through local tools

Date: 2026-11-02

## Status

Accepted

## Context

The owner wants to play against Claude on their own computer, with Claude building its own decks.
It is not meant for a deployed site, but the code lives in the public repository, so it must be
safely off unless asked for.

Claude could play in two ways: the server could call the Claude API for each decision, which needs
an API key and costs money per game, or Claude Code could play through tools the server offers,
which uses the owner's existing plan and lets Claude talk to them while it plays.

## Decision

Claude Code plays, through tools the server offers over the Model Context Protocol, using the
official Go library.

- **The same seat.** `computer.Live` sits in a game over the connection the computer player uses,
  so it is sent only what that seat could know, and the engine judges its moves like anyone's. Moves
  come from `computer.Desk` instead of the rules of thumb.
- **An ordinary opponent underneath.** Claude is a row in `computer_opponents` with the style
  `live`, so offering it, starting a game, rematches and history share the existing code. Its tools
  replace its deck and format. It is left out of the admin panel's list, whose editor would treat
  its style as a mistake.
- **Off unless asked for.** It takes an admin setting (`computer.claude_opponent`, off by default) to
  offer it, and Claude Code must have been in touch recently.
- **This computer only.** The tools have a listener of their own, which must be on a loopback
  address, and is on by default only in development. The library's protection refuses requests made
  through other sites' pages or naming another host.
- **One game at a time,** given up after 15 minutes without contact, so a player is never stuck.

## Consequences

- There is no extra bill, and a game takes as long as Claude thinks, within the limits of the
  owner's plan.
- The server gains one dependency, and its binary contains the tools even where they are never
  served. A build tag would have left them out, but could not be switched from the admin panel.
- Claude Code can also read local files, so fair play partly relies on it following its
  instructions. The tools themselves reveal nothing hidden.
- Calling the API later would need only a new driver for `computer.Desk`.
