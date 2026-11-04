import type { Metadata } from 'next';
import Link from 'next/link';
import type { ReactNode } from 'react';
import { KeywordBadges } from '@/components/how-to-play/keyword-badges';
import { Panel } from '@/components/ui/panel';
import type { Civilization } from '@/lib/civilizations';

export const metadata: Metadata = {
  title: 'How to play',
  description:
    'How to play Duel Masters on this site: the goal, the zones, a turn step by step, attacking and blocking, the duel board, keywords, deck building and ratings.',
};

/** The page's sections, in order, for the contents list and the links between sections. */
const SECTIONS = [
  { id: 'goal', title: 'The goal', accent: 'fire' },
  { id: 'quick-start', title: 'Your first game', accent: 'light' },
  { id: 'cards', title: 'The cards', accent: 'water' },
  { id: 'zones', title: 'The zones', accent: 'darkness' },
  { id: 'setup', title: 'Starting a duel', accent: 'nature' },
  { id: 'turn', title: 'A turn, step by step', accent: 'fire' },
  { id: 'mana', title: 'Mana and paying for cards', accent: 'nature' },
  { id: 'attacking', title: 'Attacking, battles and blocking', accent: 'fire' },
  { id: 'shields', title: 'Shields and shield triggers', accent: 'light' },
  { id: 'board', title: 'Using the duel board', accent: 'water' },
  { id: 'keywords', title: 'Keywords and badges', accent: 'darkness' },
  { id: 'decks', title: 'Building a deck', accent: 'nature' },
  { id: 'matches', title: 'Finding games, ratings and rematches', accent: 'light' },
  { id: 'tips', title: 'Tips for new duelists', accent: 'water' },
] as const satisfies readonly { id: string; title: string; accent: Civilization }[];

type SectionId = (typeof SECTIONS)[number]['id'];

/**
 * How to play, for visitors and players alike: the rules as this site's engine plays them, and how
 * to play them on the duel board (design/SPEC.md, section 10.2).
 */
export default function HowToPlayPage() {
  return (
    <div className="space-y-6">
      <header className="max-w-3xl space-y-3">
        <h1 className="font-display text-4xl font-bold tracking-widest text-white uppercase">
          How to play
        </h1>
        <p className="text-lg text-slate-300">
          Duel Masters is a card game for two players. You build a deck of creatures and spells,
          grow your mana a card at a time, and send your creatures to break through your opponent’s
          shields. Everything here is how this site plays it: the server keeps every rule and card
          effect, so you only make the choices.
        </p>
      </header>

      <div className="grid gap-6 lg:grid-cols-[15rem_minmax(0,1fr)]">
        <nav aria-label="On this page" className="lg:sticky lg:top-20 lg:self-start">
          <Panel title="On this page">
            <ol className="space-y-0.5 p-2 text-sm">
              {SECTIONS.map((section, index) => (
                <li key={section.id}>
                  <a
                    href={`#${section.id}`}
                    className="flex gap-2 rounded px-2 py-1 text-slate-300 transition hover:bg-white/5 hover:text-white"
                  >
                    <span className="w-5 text-right text-slate-500 tabular-nums">{index + 1}.</span>
                    {section.title}
                  </a>
                </li>
              ))}
            </ol>
          </Panel>
        </nav>

        <div className="min-w-0 space-y-6">
          <Section id="goal">
            <P>
              Each player starts with <B>5 shields</B>: face-down cards from the top of their deck.
              Attacking your opponent with a creature breaks their shields, one at a time (or more,
              with the right creatures). Once they have no shields left, land one more attack on
              them that they don’t block, and <B>you win</B>.
            </P>
            <P>A duel can also end in these ways:</P>
            <List>
              <li>
                <B>Deck out:</B> a player whose deck runs out of cards loses straight away, so keep
                an eye on how many cards your deck has left.
              </li>
              <li>
                <B>Forfeit:</B> a player who forfeits loses.
              </li>
              <li>
                <B>Disconnect:</B> a player who loses their connection has 2 minutes to come back to
                the game (unless the site’s admins changed it), or they lose.
              </li>
              <li>A few cards win or lose the game by their own effect, as their text says.</li>
            </List>
          </Section>

          <Section id="quick-start">
            <Steps>
              <Step title="Make an account">
                <Link href="/signup" className={LINK}>
                  Sign up
                </Link>{' '}
                with a name, an email address and a password. You may be asked for a code the site
                emails you, to confirm the address.
              </Step>
              <Step title="Pick a deck">
                You don’t have to build one first. Every player can play the <B>starter decks</B>,
                one for each civilization (Fire: Bolshack Rush is chosen for you). Later, copy one
                into{' '}
                <Link href="/decks" className={LINK}>
                  your decks
                </Link>{' '}
                to change it, or build your own (see{' '}
                <SectionLink id="decks">Building a deck</SectionLink>).
              </Step>
              <Step title="Find an opponent">
                Go to{' '}
                <Link href="/play" className={LINK}>
                  Play
                </Link>{' '}
                and choose <B>Quick match</B> to be paired with a player near your rating,{' '}
                <B>Open a game</B> to wait for someone to join, or <B>Vs computer</B> to practise
                against a computer opponent. Games against the computer never change your rating, so
                it is the place to learn.
              </Step>
              <Step title="Play">
                The duel starts with a coin toss, then the turns begin. The board tells you what you
                can do at every moment, and this page explains the rest.
              </Step>
            </Steps>
          </Section>

          <Section id="cards">
            <P>
              Every card belongs to one or more of the five <B>civilizations</B>, shown by its
              colour. Each has its own way of playing:
            </P>
            <List>
              <li>
                <Civ name="Light" colour="bg-civ-light" />
                Blockers that guard your shields, and spells that tap the opponent’s creatures.
              </li>
              <li>
                <Civ name="Water" colour="bg-civ-water" />
                Drawing cards, sending creatures back to their owner’s hand, and attackers that are
                hard to block.
              </li>
              <li>
                <Civ name="Darkness" colour="bg-civ-darkness" />
                Destroying creatures, slayers that take their killer with them, and using the
                graveyard.
              </li>
              <li>
                <Civ name="Fire" colour="bg-civ-fire" />
                Fast, hard-hitting attackers, and spells that destroy creatures and blockers.
              </li>
              <li>
                <Civ name="Nature" colour="bg-civ-nature" />
                Extra mana, powerful creatures and boosts to power.
              </li>
            </List>
            <P>There are three kinds of cards:</P>
            <List>
              <li>
                <B>Creatures</B> are summoned to the battle zone, where they attack and defend. Each
                has a <B>cost</B> (the number in its top-left corner), a <B>power</B> (how strong it
                is in a battle) and a <B>race</B>, such as Armored Dragon. Their abilities are
                written in their text box.
              </li>
              <li>
                <B>Evolution creatures</B> are put on top of one of your creatures already in the
                battle zone: the card says what it must evolve from, such as a creature of a certain
                race. They have no summoning sickness, so they can attack at once. The cards
                underneath stay with them, and leave the battle zone with them.
              </li>
              <li>
                <B>Spells</B> do what their text says the moment you cast them, then go to your
                graveyard.
              </li>
            </List>
            <P>
              To read any card, hover over it (on a computer) or tap it (on a phone). The{' '}
              <Link href="/cards" className={LINK}>
                card list
              </Link>{' '}
              has every card the game can play, with filters to search them.
            </P>
          </Section>

          <Section id="zones">
            <P>
              Each player has the same zones. Yours are on the lower half of the board, your
              opponent’s on the upper half.
            </P>
            <dl className="grid gap-3 sm:grid-cols-2">
              <Zone name="Deck">
                Your 40 cards, face down and shuffled. You draw from the top. Nobody can look
                through it, but both players see how many cards are left.
              </Zone>
              <Zone name="Hand">
                The cards you hold. Only you see them; your opponent sees how many you have.
              </Zone>
              <Zone name="Mana zone">
                Cards you put here pay for your other cards. Each turn you may add one card from
                your hand. Mana cards are tapped (turned sideways) when you use them, and untap at
                the start of your turn.
              </Zone>
              <Zone name="Battle zone">
                Where your creatures are. Untapped creatures stand upright; a creature taps when it
                attacks, blocks or uses a tap ability.
              </Zone>
              <Zone name="Shield zone">
                Your shields, face down. When one is broken, it goes to your hand, and it may be a
                shield trigger (see{' '}
                <SectionLink id="shields">Shields and shield triggers</SectionLink>).
              </Zone>
              <Zone name="Graveyard">
                Destroyed creatures and cast spells, face up. Some cards bring cards back from here.
              </Zone>
            </dl>
          </Section>

          <Section id="setup">
            <List ordered>
              <li>Both decks are shuffled.</li>
              <li>Each player puts the top 5 cards of their deck face down as their shields.</li>
              <li>Each player draws 5 cards.</li>
              <li>
                A <B>coin toss</B> decides who starts. One player calls heads or tails (chosen at
                random, or in a rematch the player who lost the last game). Both players watch the
                coin, and its winner chooses to go first or second.
              </li>
              <li>
                The player who goes first <B>doesn’t draw</B> on their first turn, to make up for
                starting first.
              </li>
            </List>
          </Section>

          <Section id="turn">
            <P>
              A turn has five phases, shown in the phase bar on the board. The draw and end phases
              pass by themselves; in the mana, summon and battle phases, the turn waits for you. The
              turn only moves forwards: once you leave a phase, you can’t go back to it this turn.
            </P>
            <Steps>
              <Step title="Draw phase">
                Your tapped creatures and mana cards untap, then you draw a card.
              </Step>
              <Step title="Mana phase">
                You may put <B>one card</B> from your hand into your mana zone, or none. Any card
                will do: it gives you one mana of its civilization from now on. When you’re done,
                press <B>Go to summon phase</B>.
              </Step>
              <Step title="Summon phase">
                Summon creatures and cast spells, as many as your mana can pay for (see{' '}
                <SectionLink id="mana">Mana and paying for cards</SectionLink>). Then press{' '}
                <B>Battle phase</B>. If you could still play a card or add mana, the board asks you
                first, because neither is possible after that.
              </Step>
              <Step title="Battle phase">
                Attack with your creatures and use their tap abilities, one creature at a time (see{' '}
                <SectionLink id="attacking">Attacking, battles and blocking</SectionLink>). Nothing
                can be summoned, cast or put into mana any more this turn.
              </Step>
              <Step title="End phase">
                When you press <B>End turn</B>, effects that happen at the end of a turn take place,
                and it is your opponent’s turn.
              </Step>
            </Steps>
            <P>
              <B>End turn</B> is there in every phase, so you can end your turn early, skipping
              whatever phases are left, such as when you have nothing to attack with.
            </P>
          </Section>

          <Section id="mana">
            <P>
              Every card has a cost. To play it, you tap that many <B>untapped</B> cards in your
              mana zone, and among them must be:
            </P>
            <List>
              <li>
                at least one card of the card’s <B>civilization</B>. A 5-cost Fire creature needs 5
                mana, at least one of them Fire.
              </li>
              <li>
                for a <B>multicolored</B> card (a card of two or more civilizations), at least one
                card of <B>each</B> of its civilizations.
              </li>
            </List>
            <P>
              A multicolored card in your mana zone counts as only one of its civilizations each
              time you pay, never several at once. Multicolored cards also go into the mana zone{' '}
              <B>tapped</B>, so they can’t pay for anything on the turn you put them there.
            </P>
            <P>
              When you play a card, the board asks you to choose the mana cards to tap. It only lets
              you summon or cast a card you can pay for, and a greyed-out <B>Summon</B> or{' '}
              <B>Cast</B> button says why you can’t yet.
            </P>
          </Section>

          <Section id="attacking">
            <P>
              In the battle phase, any of your untapped creatures can attack, except those summoned
              this turn: they have <B>summoning sickness</B> and must wait until your next turn
              (unless they are speed attackers or evolution creatures). Attacking taps the creature,
              and each creature attacks at most once a turn. You can attack:
            </P>
            <List>
              <li>
                <B>Your opponent</B>, to break their shields. You choose which shields to break. If
                they have no shields left and the attack isn’t blocked, you win.
              </li>
              <li>
                <B>One of their tapped creatures</B>, to start a battle with it. Untapped creatures
                can’t be attacked, unless the attacker has an ability that allows it.
              </li>
            </List>

            <H3>Battles</H3>
            <P>
              When two creatures battle, the one with the <B>higher power</B> wins and the other is
              destroyed (put into its owner’s graveyard). If their power is the same, both are
              destroyed. Before an attack, the board shows both creatures’ power and how the battle
              would end, and asks you to confirm.
            </P>

            <H3>Blocking</H3>
            <P>
              When your opponent attacks you or one of your creatures, and you have an untapped
              creature with <B>blocker</B>, the board asks whether you want to block. If you block,
              your blocker taps and battles the attacker instead, and the attack breaks no shields.
              You don’t have to block.
            </P>
            <P>
              This is why untapped creatures matter: a creature that attacked stays tapped through
              your opponent’s next turn, where it can be attacked, and can’t block.
            </P>
          </Section>

          <Section id="shields">
            <P>
              A broken shield isn’t lost: the card goes into its owner’s <B>hand</B>, so taking hits
              gives the defender more cards to play.
            </P>
            <P>
              Some cards are <B>shield triggers</B>. When a shield trigger is broken out of your
              shield zone, you may play it at once, <B>for free</B>, even during your opponent’s
              turn: summon the creature or cast the spell without paying its cost. When several are
              broken together, you play them one at a time. A well-timed shield trigger can turn a
              game around, so it is worth having a few in a deck.
            </P>
            <P>
              Creatures with <B>double breaker</B> or <B>triple breaker</B> break 2 or 3 shields in
              one attack.
            </P>
          </Section>

          <Section id="board">
            <H3>Taking actions</H3>
            <P>
              <B>Click or tap a card</B> to see what you can do with it. Actions you can’t take
              right now are greyed out, with the reason:
            </P>
            <List>
              <li>
                A card in your hand: <B>Summon</B> or <B>Cast</B>, and <B>Add to mana</B>.
              </li>
              <li>
                A creature in your battle zone: <B>Attack player</B>, <B>Attack creature</B>, and{' '}
                <B>Use tap ability</B> for creatures that have one.
              </li>
              <li>
                Any face-up card: <B>View</B>, to read it, with all its keywords spelled out.
              </li>
            </List>
            <P>
              Or <B>drag cards</B> where they should go: from your hand to your mana zone to add
              mana, or to your battle zone to summon or cast. Drag a creature onto your opponent’s
              shields to attack them, or onto their battle zone to attack one of their creatures.
            </P>

            <H3>Choices</H3>
            <P>
              When the game needs a decision, such as which mana to tap, which creature a spell
              targets, which shields to break, whether to block or which shield trigger to play, a
              panel explains what to choose. Pick the cards, then confirm. Some choices can be
              cancelled, such as summoning a creature after all.
            </P>

            <H3>The rest of the board</H3>
            <List>
              <li>
                The <B>phase bar</B> shows the turn’s phases, with the current one lit, and how many
                turns have been played. Hover over a phase for what happens in it. Beside it are the
                button for the next phase and <B>End turn</B>, and a short hint tells you what you
                can do now.
              </li>
              <li>
                Each creature shows its <B>power</B>, and <B>badges</B> above it for its most
                important keywords (see <SectionLink id="keywords">Keywords and badges</SectionLink>
                ).
              </li>
              <li>
                The <B>game log</B> lists everything that happens; click a card name in it to see
                the card. The <B>chat</B> is beside it.
              </li>
              <li>
                The red <B>forfeit</B> button (an arrow heading out) gives up the game, after asking
                you to confirm.
              </li>
            </List>
            <P>
              If your connection drops, open the game again to carry on where you left off: your
              opponent is told while you are away.
            </P>
          </Section>

          <Section id="keywords">
            <P>
              Many creatures have <B>keywords</B>: short names for abilities. The duel board shows
              the most important ones as badges above each creature, so you can read the board at a
              glance. A badge with a <B>yellow ring</B> was given by another card’s effect rather
              than printed on the card, and <B>+2</B> beside the badges means 2 more, which{' '}
              <B>View</B> lists.
            </P>
            <KeywordBadges />

            <H3>Keywords without a badge</H3>
            <List>
              <li>
                <B>Shield trigger:</B> can be played for free when it is broken out of your shields
                (see <SectionLink id="shields">Shields and shield triggers</SectionLink>
                ).
              </li>
              <li>
                <B>Evolution:</B> put on top of one of your creatures, as its card says (see{' '}
                <SectionLink id="cards">The cards</SectionLink>).
              </li>
              <li>
                <B>Charger:</B> after you cast this spell, it goes into your mana zone instead of
                your graveyard, so it adds mana as well.
              </li>
              <li>
                <B>Multicolored:</B> a card of two or more civilizations. It needs mana of each to
                play, and goes into the mana zone tapped.
              </li>
            </List>
          </Section>

          <Section id="decks">
            <P>
              A deck has exactly <B>40 cards</B>, with at most <B>4 copies</B> of any one card. You
              can save a deck before it has 40 cards and finish it later, but only a full deck can
              be played.
            </P>
            <P>
              Every game is played in a <B>format</B>, which decides the cards a deck may use:{' '}
              <B>Open</B> allows every card the game can play, and <B>DM-01 to DM-12</B> only the
              first twelve sets. A format can also ban cards, limit some to 1 or 2 copies, or set
              how many civilizations a deck may have. The deck builder shows, for each format,
              whether your deck is legal there and what breaks the rules.
            </P>
            <P>
              In the{' '}
              <Link href="/decks" className={LINK}>
                deck builder
              </Link>
              , the trunk on the side lists every card. Search it, sort it, or open <B>Filters</B>{' '}
              to find cards by civilization, cost, type, race, set and more. Drag cards into your
              deck, or use the + and − buttons. The stats show your mana curve and civilizations.
              Decks can also be exported and imported as text, one line per card, such as{' '}
              <code className="text-slate-200">4 Bolshack Dragon</code>.
            </P>
            <H3>A first deck</H3>
            <List>
              <li>
                Start with <B>one or two civilizations</B>. More makes it harder to have the right
                mana.
              </li>
              <li>
                Keep it <B>cheap</B>: plenty of cards costing 2 to 4, so you have something to play
                every turn, and only a few big ones at 6 or more.
              </li>
              <li>
                Add some <B>shield triggers</B> and <B>blockers</B> to survive attacks, and a few
                ways to destroy or remove the opponent’s creatures.
              </li>
              <li>Use 4 copies of your best cards, so you draw them more often.</li>
            </List>
          </Section>

          <Section id="matches">
            <List>
              <li>
                <B>Quick match:</B> choose a format and a deck, and you are paired with the waiting
                player whose rating is closest to yours. The longer you wait, the wider the search,
                so a match is found sooner.
              </li>
              <li>
                <B>Open a game:</B> name your game, choose its format and your deck, and whether it
                is listed in the lobby or private (only your invite link leads to it), and whether
                others may watch. Then wait for someone to join.
              </li>
              <li>
                <B>Vs computer:</B> play one of the site’s computer opponents, whenever you like.
              </li>
              <li>
                <B>Watch:</B> logged-in players can watch the lobby’s games that allow it.
              </li>
            </List>
            <H3>Ratings and ranks</H3>
            <P>
              Every player starts with a rating of <B>1000</B>. Winning raises it and losing lowers
              it, more so against a stronger opponent: beating a higher-rated player gains more than
              beating a lower-rated one. Games against the computer, and very short games (under a
              minute), don’t count. Your rating gives your rank:
            </P>
            <div className="overflow-x-auto">
              <table className="w-full max-w-md text-left text-sm">
                <thead className="text-[11px] tracking-wider text-slate-400 uppercase">
                  <tr className="border-b border-slate-700/60">
                    <th scope="col" className="py-2 pr-4 font-semibold">
                      Rank
                    </th>
                    <th scope="col" className="py-2 font-semibold">
                      Rating
                    </th>
                  </tr>
                </thead>
                <tbody className="text-slate-300">
                  {RANKS.map(([rank, rating]) => (
                    <tr key={rank} className="border-b border-slate-800/80 last:border-0">
                      <td className="py-2 pr-4 font-semibold text-slate-100">{rank}</td>
                      <td className="py-2 tabular-nums">{rating}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <P>
              Each rank but Diamond is split into III, II and I, from low to high. The{' '}
              <Link href="/rankings" className={LINK}>
                rankings
              </Link>{' '}
              list every player who has finished a game that counted.
            </P>
            <H3>Rematches</H3>
            <P>
              When a game ends, either player can offer a <B>rematch</B> from the result screen, and
              the other can accept or decline. Before accepting, each player may switch to another
              of their decks. The offer lasts 2 minutes, and a rematch starts with a coin toss of
              its own.
            </P>
          </Section>

          <Section id="tips">
            <List>
              <li>
                <B>Add mana every turn early on.</B> Missing a mana drop leaves you a turn behind.
                Put in cards you can spare, such as a second copy of an expensive card.
              </li>
              <li>
                <B>Plan your turn before the battle phase.</B> Once you move on, you can’t summon or
                cast until your next turn.
              </li>
              <li>
                <B>Think before attacking.</B> An attacking creature stays tapped, where your
                opponent can attack it, and can’t block. Breaking shields also hands your opponent
                cards, and maybe a shield trigger.
              </li>
              <li>
                <B>Count the shields.</B> When your opponent is down to one or two, a few unblocked
                attacks in one turn can end the game.
              </li>
              <li>
                <B>Keep a blocker back</B> when you are low on shields.
              </li>
              <li>
                <B>Practise against the computer</B> to learn a deck before taking it to a rated
                game.
              </li>
            </List>
            <div className="pt-2">
              <Link
                href="/play"
                className="inline-block font-display text-lg font-semibold tracking-wider text-accent uppercase hover:underline"
              >
                Ready? Find a game →
              </Link>
            </div>
          </Section>
        </div>
      </div>
    </div>
  );
}

const RANKS = [
  ['Bronze', 'below 1100'],
  ['Silver', '1100 to 1299'],
  ['Gold', '1300 to 1499'],
  ['Platinum', '1500 to 1699'],
  ['Diamond', '1700 and above'],
] as const;

const LINK = 'font-semibold text-accent hover:underline';

function Section({ id, children }: { readonly id: SectionId; readonly children: ReactNode }) {
  const index = SECTIONS.findIndex((section) => section.id === id);
  const section = SECTIONS[index];
  if (!section) throw new Error(`No section ${id}`);

  return (
    <Panel
      id={id}
      title={`${String(index + 1)}. ${section.title}`}
      accent={section.accent}
      className="scroll-mt-20"
    >
      <div className="space-y-4 p-5 leading-relaxed">{children}</div>
    </Panel>
  );
}

function SectionLink({ id, children }: { readonly id: SectionId; readonly children: ReactNode }) {
  return (
    <a href={`#${id}`} className={LINK}>
      {children}
    </a>
  );
}

function H3({ children }: { readonly children: ReactNode }) {
  return (
    <h3 className="pt-2 font-display text-lg font-bold tracking-wider text-white uppercase">
      {children}
    </h3>
  );
}

function P({ children }: { readonly children: ReactNode }) {
  return <p className="text-slate-300">{children}</p>;
}

function B({ children }: { readonly children: ReactNode }) {
  return <strong className="font-semibold text-white">{children}</strong>;
}

function List({
  ordered = false,
  children,
}: {
  readonly ordered?: boolean;
  readonly children: ReactNode;
}) {
  const className = `space-y-2 pl-5 text-slate-300 marker:text-slate-500 ${ordered ? 'list-decimal' : 'list-disc'}`;
  return ordered ? (
    <ol className={className}>{children}</ol>
  ) : (
    <ul className={className}>{children}</ul>
  );
}

function Steps({ children }: { readonly children: ReactNode }) {
  return <ol className="space-y-4 [counter-reset:step]">{children}</ol>;
}

/** A numbered step, its number in a diamond like the site's logo. */
function Step({ title, children }: { readonly title: string; readonly children: ReactNode }) {
  return (
    <li className="flex gap-4 [counter-increment:step]">
      <span
        aria-hidden
        className="mt-1 flex size-7 shrink-0 rotate-45 items-center justify-center bg-slate-700 ring-1 ring-slate-500/60"
      >
        <span className="-rotate-45 font-display text-sm font-bold text-white before:content-[counter(step)]" />
      </span>
      <div>
        <h3 className="font-display text-lg font-bold tracking-wide text-white">{title}</h3>
        <p className="text-slate-300">{children}</p>
      </div>
    </li>
  );
}

function Civ({ name, colour }: { readonly name: string; readonly colour: string }) {
  return (
    <span className="mr-1.5 inline-flex items-center gap-1.5 font-semibold text-white">
      <span aria-hidden className={`inline-block size-2.5 rotate-45 ${colour}`} />
      {name}:
    </span>
  );
}

function Zone({ name, children }: { readonly name: string; readonly children: ReactNode }) {
  return (
    <div className="rounded border border-slate-700/60 bg-white/[0.02] p-3">
      <dt className="font-display text-base font-bold tracking-wider text-white uppercase">
        {name}
      </dt>
      <dd className="mt-1 text-sm text-slate-300">{children}</dd>
    </div>
  );
}
