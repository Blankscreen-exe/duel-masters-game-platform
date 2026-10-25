# 0013. The card catalogue is copied into the database

- Status: Accepted
- Date: 2026-10-25

## Context

The card catalogue comes from the linked card database (0006): a JSON file of 2,647 printings,
their pictures and set icons. The server needs to search it, show cards, check decks against it
and link it to the 895 cards the rules engine plays. 18 cards were reprinted in several sets.

## Decision

- At startup the server copies the card database into its own database: sets, cards, printings
  and each card's civilizations, races, keywords, categories and rules lines. It copies only
  when a fingerprint of the card data, the engine's cards and the copy format has changed, and
  always in one transaction.
- **A reprinted card is one card** with every printing. Its details and picture are its first
  printing's, the picture the engine shows in games; copy limits count the card.
- Cards are known by their first printing's id. Cards that leave the card database keep their
  row, marked gone and hidden, so nothing referring to them, such as a deck, breaks.
- Name order and a search text folded to lowercase without accents are worked out once while
  copying, so SQLite and PostgreSQL sort and search alike.
- Pictures and set icons stay in the files and are served from them by printing id or engine id.

## Alternatives considered

- **Keep the catalogue in memory**, read from the files at every start: no copy to keep in step,
  but searching, filtering and deck checks would be written by hand instead of as queries, and
  other tables (decks, formats) could not refer to cards.
- **List every printing separately**: players could choose the art, but the copy limit would
  still have to count printings together, and the engine shows one picture per card anyway.

## Consequences

- Searches and later deck and format checks are database queries, and other tables can refer to
  cards by id.
- A start after the card database or the engine changes takes about half a second longer
  (SQLite) to a second and a half (PostgreSQL) while the catalogue is copied.
