# 0014. Players confirm an email address, through a mail server an admin sets up

- Status: Accepted
- Date: 2026-11-01

## Context

Sign-up asked only for a name and a password. New players must now give an email address and
confirm it with a one-time code, sent through an SMTP server the admin sets up on the site. Its
password must not sit in the database in the clear. The first account signs up before any mail
server exists, because it is the one that sets it up.

## Decision

- **Every new player gives an address, unique regardless of case, and enters a 6-digit code.** Until
  then the session is good for confirming and nothing else: everything else answers as if they were
  not logged in, sockets included. A code works for 15 minutes and 5 guesses, is stored only as a
  hash, and another can be asked for after 60 seconds (admin settings, but for the guesses).
- **The first account needs no code**, and neither do accounts from before this change.
- **While no mail server is set up, other new players wait.** They can sign up and correct their
  address; an admin can let them in, or they ask for a code once one is set up.
- **An account is deleted 24 hours after its first code** if still not confirmed (an admin setting),
  freeing its name and address. A waiting account has been sent no code, so nothing counts down.
- **The mail settings are sealed together** with AES-256-GCM into one row, which holds nothing
  readable: not only the password but the server, login and addresses. The key is `APP_SECRET_KEY`,
  from the environment and never in the database, required in production. Losing it loses only the
  mail settings: the admin enters them again.
- The settings have their own table and page rather than the key-value settings, whose values the
  generic page shows and the admin log records.
- Codes are sent during the request, within 15 seconds. A failure does not stop the sign-up: the
  player asks again from the page they land on.

## Consequences

- A stolen database or backup gives away neither the mail password nor usable codes.
- Signing up answers more slowly by the time the mail server takes.
- Saying an address is taken tells anyone that it has an account, as saying a name is taken does.
- A verified player cannot yet change their address, and a forgotten password is still reset by an
  admin: password reset by email would be built on this.
- Nothing limits sign-ups per visitor yet, so someone could sign up with many addresses to send a
  code to each. That limit is needed before going public.
