import type { Metadata } from 'next';
import Link from 'next/link';
import type { ReactNode } from 'react';
import { KeywordBadges } from '@/components/how-to-play/keyword-badges';
import {
  AnnotatedCard,
  BoardButton,
  BoardZone,
  CardBack,
  CardPic,
  Captioned,
  Flow,
  FlowArrow,
  Versus,
  type CardCallout,
  type ShownCard,
} from '@/components/how-to-play/visuals';
import { RankBadge } from '@/components/players/rank-badge';
import { Panel } from '@/components/ui/panel';
import { serverApi } from '@/lib/api/server';
import type { Civilization } from '@/lib/civilizations';
import type { Rank } from '@/lib/players';

export const metadata: Metadata = {
  title: 'How to play',
  description:
    'How to play Duel Masters, in pictures: reading a card, the board, a turn, mana, attacking, shields, keywords, decks and ranks.',
};

/** The cards the page shows, by catalogue number, all from the first sets. */
const CARD_IDS = {
  bolshackDragon: '0079',
  brawlerZyler: '0080',
  valdios: '0124',
  holyAwe: '0016',
  hanusa: '0001',
  aquaHulcus: '0033',
  deathliger: '0005',
  bronzeArmTribe: '0100',
  gigaberos: '0065',
  granGure: '0015',
  bombazar: '0679',
} as const;

type CardName = keyof typeof CARD_IDS;

/**
 * The parts of a card, marked on Bolshack Dragon, where every card has them. Each marker sits on
 * the card's edge level with its part, so it covers nothing.
 */
const CARD_PARTS: readonly CardCallout[] = [
  { x: -6, y: 6, label: 'Cost', detail: 'The mana it takes to play the card.' },
  { x: 55, y: -3, label: 'Name', detail: 'Up to 4 copies of a name in a deck.' },
  { x: 106, y: 11.5, label: 'Race', detail: 'Some cards care about it, such as evolutions.' },
  { x: -6, y: 66.5, label: 'Type', detail: 'Creature, evolution creature or spell.' },
  { x: 106, y: 77, label: 'Abilities', detail: 'What the card does. The game applies it for you.' },
  { x: -6, y: 94, label: 'Power', detail: 'Its strength in battle. A + means it can grow.' },
  {
    x: 51,
    y: 103,
    label: 'Civilization',
    detail: 'The symbol and the frame’s colour. Pay with mana of it.',
  },
];

const SECTIONS = [
  { id: 'goal', title: 'Win the duel', accent: 'fire' },
  { id: 'card', title: 'Reading a card', accent: 'light' },
  { id: 'kinds', title: 'Three kinds of cards', accent: 'water' },
  { id: 'civilizations', title: 'Five civilizations', accent: 'darkness' },
  { id: 'board', title: 'The board', accent: 'nature' },
  { id: 'turn', title: 'Your turn', accent: 'fire' },
  { id: 'mana', title: 'Paying with mana', accent: 'nature' },
  { id: 'attacking', title: 'Attacking', accent: 'fire' },
  { id: 'shields', title: 'Shield triggers', accent: 'light' },
  { id: 'controls', title: 'Controls', accent: 'water' },
  { id: 'keywords', title: 'Keywords', accent: 'darkness' },
  { id: 'decks', title: 'Decks', accent: 'nature' },
  { id: 'games', title: 'Games and ranks', accent: 'light' },
  { id: 'tips', title: 'Tips', accent: 'water' },
] as const satisfies readonly { id: string; title: string; accent: Civilization }[];

type SectionId = (typeof SECTIONS)[number]['id'];

const RANKS: readonly { readonly rank: Rank; readonly from: string }[] = [
  { rank: { tier: 'Bronze', label: 'Bronze', rating: 1000 }, from: 'Start' },
  { rank: { tier: 'Silver', label: 'Silver', rating: 1100 }, from: '1100' },
  { rank: { tier: 'Gold', label: 'Gold', rating: 1300 }, from: '1300' },
  { rank: { tier: 'Platinum', label: 'Platinum', rating: 1500 }, from: '1500' },
  { rank: { tier: 'Diamond', label: 'Diamond', rating: 1700 }, from: '1700' },
];

/**
 * How to play, in pictures (design/SPEC.md, section 10.2): the rules as this site's engine plays
 * them, shown with real cards and the board's own badges and buttons, with as few words as will do.
 */
export default async function HowToPlayPage() {
  const cards = await loadCards();
  const card = (name: CardName): ShownCard => cards[name];

  return (
    <div className="space-y-6">
      <header className="space-y-4">
        <h1 className="font-display text-4xl font-bold tracking-widest text-white uppercase">
          How to play
        </h1>
        <p className="max-w-2xl text-lg text-slate-300">
          Two players, one deck each. Grow your mana, summon creatures, and break through your
          opponent’s shields. The game keeps every rule for you.
        </p>
        <nav aria-label="On this page">
          <ol className="flex flex-wrap gap-2">
            {SECTIONS.map((section, index) => (
              <li key={section.id}>
                <a
                  href={`#${section.id}`}
                  className="inline-flex items-center gap-1.5 rounded-full bg-white/5 px-3 py-1 text-sm text-slate-300 ring-1 ring-slate-600/60 transition hover:bg-white/10 hover:text-white"
                >
                  <span className="text-slate-500 tabular-nums">{index + 1}</span>
                  {section.title}
                </a>
              </li>
            ))}
          </ol>
        </nav>
      </header>

      <Section id="goal">
        <Flow>
          <Captioned label="5 shields each" detail="Face-down cards from your deck">
            <div className="flex -space-x-6 sm:-space-x-10">
              {[1, 2, 3, 4, 5].map((n) => (
                <CardBack key={n} size="sm" />
              ))}
            </div>
          </Captioned>
          <FlowArrow label="Break them all" />
          <Captioned label="Attack once more" detail="With no shields left, and not blocked">
            <CardPic card={card('bolshackDragon')} size="sm" />
          </Captioned>
          <FlowArrow />
          <p className="font-display text-4xl font-bold tracking-widest text-accent uppercase">
            You win
          </p>
        </Flow>
        <Note>
          You also lose if your <B>deck runs out</B>, if you <B>forfeit</B>, or if you stay
          disconnected for <B>2 minutes</B>.
        </Note>
      </Section>

      <Section id="card">
        <AnnotatedCard card={card('bolshackDragon')} callouts={CARD_PARTS} />
        <Note>
          Hover over any card (or tap it, on a phone) to see it large. Browse them all in the{' '}
          <TextLink href="/cards">card list</TextLink>.
        </Note>
      </Section>

      <Section id="kinds">
        <div className="grid grid-cols-1 justify-items-center gap-8 sm:grid-cols-3">
          <Captioned label="Creature" detail="Goes to the battle zone to attack and defend.">
            <CardPic card={card('bolshackDragon')} size="md" />
          </Captioned>
          <Captioned
            label="Evolution creature"
            detail="Goes on top of one of your creatures (here, a Human). Attacks at once."
          >
            <div className="relative pt-4 pl-4">
              <div className="absolute top-0 left-0 opacity-80">
                <CardPic card={card('brawlerZyler')} size="md" />
              </div>
              <div className="relative">
                <CardPic card={card('valdios')} size="md" />
              </div>
            </div>
          </Captioned>
          <Captioned label="Spell" detail="Does what it says, then goes to the graveyard.">
            <CardPic card={card('holyAwe')} size="md" />
          </Captioned>
        </div>
      </Section>

      <Section id="civilizations">
        <div className="grid grid-cols-2 justify-items-center gap-6 sm:grid-cols-3 lg:grid-cols-5">
          <Captioned label={<Civ name="Light" colour="bg-civ-light" />} detail="Blockers, tapping">
            <CardPic card={card('hanusa')} size="md" />
          </Captioned>
          <Captioned
            label={<Civ name="Water" colour="bg-civ-water" />}
            detail="Drawing, bouncing, hard to block"
          >
            <CardPic card={card('aquaHulcus')} size="md" />
          </Captioned>
          <Captioned
            label={<Civ name="Darkness" colour="bg-civ-darkness" />}
            detail="Destroying, slayers"
          >
            <CardPic card={card('deathliger')} size="md" />
          </Captioned>
          <Captioned
            label={<Civ name="Fire" colour="bg-civ-fire" />}
            detail="Fast attackers, burning creatures"
          >
            <CardPic card={card('bolshackDragon')} size="md" />
          </Captioned>
          <Captioned
            label={<Civ name="Nature" colour="bg-civ-nature" />}
            detail="Extra mana, big power"
          >
            <CardPic card={card('bronzeArmTribe')} size="md" />
          </Captioned>
        </div>
      </Section>

      <Section id="board">
        <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_auto]">
          <div className="space-y-5">
            <BoardZone name="Battle zone" note="Your creatures. Sideways = tapped (used).">
              <CardPic card={card('aquaHulcus')} />
              <CardPic card={card('brawlerZyler')} tapped />
              <CardPic card={card('granGure')} />
            </BoardZone>
            <BoardZone name="Shields" note="Broken shields go to your hand.">
              {[1, 2, 3, 4, 5].map((n) => (
                <CardBack key={n} size="xs" />
              ))}
            </BoardZone>
            <BoardZone name="Mana zone" note="Tap these to pay for cards. They untap each turn.">
              <CardPic card={card('bronzeArmTribe')} size="xs" />
              <CardPic card={card('deathliger')} size="xs" />
              <CardPic card={card('hanusa')} size="xs" tapped />
              <CardPic card={card('gigaberos')} size="xs" tapped />
            </BoardZone>
            <BoardZone name="Hand" note="Only you see these.">
              <CardPic card={card('holyAwe')} />
              <CardPic card={card('valdios')} />
              <CardPic card={card('bombazar')} />
            </BoardZone>
          </div>
          <div className="grid grid-cols-2 gap-5 lg:grid-cols-1 lg:content-start">
            <BoardZone name="Deck" note="Draw from here. Empty = you lose.">
              <CardBack />
            </BoardZone>
            <BoardZone name="Graveyard" note="Destroyed creatures, used spells.">
              <CardPic card={card('bolshackDragon')} />
            </BoardZone>
          </div>
        </div>
        <Note>Your half of the board. Your opponent’s half mirrors it, above.</Note>
      </Section>

      <Section id="turn">
        <ol className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
          <Phase n={1} name="Draw" what="Untap everything, draw a card." next={<Auto />} />
          <Phase
            n={2}
            name="Mana"
            what="Put up to 1 card from your hand into mana."
            next={<BoardButton>Go to summon phase</BoardButton>}
          />
          <Phase
            n={3}
            name="Summon"
            what="Summon creatures, cast spells."
            next={<BoardButton>Battle phase</BoardButton>}
          />
          <Phase
            n={4}
            name="Battle"
            what="Attack, use tap abilities."
            next={<BoardButton tone="quiet">End turn</BoardButton>}
          />
          <Phase n={5} name="End" what="End-of-turn effects, then their turn." next={<Auto />} />
        </ol>
        <ul className="flex flex-wrap gap-x-6 gap-y-1 text-sm text-slate-400">
          <li>• Whoever goes first skips their first draw.</li>
          <li>• No going back to an earlier phase.</li>
          <li>
            • <BoardButton tone="quiet">End turn</BoardButton> works in any phase.
          </li>
        </ul>
      </Section>

      <Section id="mana">
        <Flow>
          <Captioned label="Costs 6, Fire" detail="Bolshack Dragon">
            <CardPic card={card('bolshackDragon')} size="md" />
          </Captioned>
          <Versus>=</Versus>
          <Captioned label="Tap 6 mana" detail="At least 1 of them Fire">
            <div className="flex flex-wrap justify-center gap-1.5">
              <div className="rounded-[6%] ring-2 ring-accent ring-offset-2 ring-offset-panel">
                <CardPic card={card('bolshackDragon')} size="xs" />
              </div>
              <CardPic card={card('bronzeArmTribe')} size="xs" />
              <CardPic card={card('aquaHulcus')} size="xs" />
              <CardPic card={card('deathliger')} size="xs" />
              <CardPic card={card('hanusa')} size="xs" />
              <CardPic card={card('granGure')} size="xs" />
            </div>
          </Captioned>
        </Flow>
        <Flow>
          <Captioned label="Multicolored" detail="Needs Fire and Nature mana to play.">
            <CardPic card={card('bombazar')} size="md" />
          </Captioned>
          <FlowArrow label="Into mana" />
          <Captioned label="Enters tapped" detail="Can’t pay for anything that turn.">
            <CardPic card={card('bombazar')} size="md" tapped />
          </Captioned>
        </Flow>
      </Section>

      <Section id="attacking">
        <div className="grid gap-4">
          <Example
            title="Attack the player"
            note="Breaks 1 shield, or more with breakers. You pick which."
          >
            <Flow row>
              <CardPic card={card('bolshackDragon')} />
              <FlowArrow label="Double breaker" row />
              <div className="flex -space-x-8">
                <CardBack />
                <CardBack />
              </div>
            </Flow>
          </Example>
          <Example
            title="Attack a creature"
            note="Only tapped ones. Higher power wins; a tie destroys both."
          >
            <Flow row>
              <Captioned label="2000" detail="Wins">
                <CardPic card={card('aquaHulcus')} />
              </Captioned>
              <Versus>vs</Versus>
              <Captioned label="1000" detail="Destroyed">
                <div className="opacity-50 grayscale">
                  <CardPic card={card('bronzeArmTribe')} tapped />
                </div>
              </Captioned>
            </Flow>
          </Example>
          <Example
            title="Block"
            note="A blocker taps to battle the attacker instead. No shields break."
          >
            <Captioned label="Blocker" detail="Gran Gure, 9000">
              <CardPic card={card('granGure')} tapped />
            </Captioned>
          </Example>
        </div>
        <Note>
          Creatures summoned this turn can’t attack yet (<B>summoning sickness</B>). Attacking taps
          a creature, so it can’t block on your opponent’s turn.
        </Note>
      </Section>

      <Section id="shields">
        <Flow>
          <Captioned label="A shield breaks" detail="It goes to your hand">
            <CardBack size="md" />
          </Captioned>
          <FlowArrow label="Shield trigger?" />
          <Captioned label="Play it free" detail="Right away, even on their turn">
            <CardPic card={card('holyAwe')} size="md" />
          </Captioned>
        </Flow>
        <Note>Holy Awe taps all your opponent’s creatures, stopping their next attacks.</Note>
      </Section>

      <Section id="controls">
        <div className="grid gap-4 sm:grid-cols-3">
          <Control title="Click or tap a card" icon="👆">
            Its menu: <B>Summon</B>, <B>Add to mana</B>, <B>Attack player</B>,{' '}
            <B>Attack creature</B>, <B>View</B>. Greyed out = not now (it says why).
          </Control>
          <Control title="Or drag it" icon="✋">
            Hand → mana zone or battle zone. Creature → their shields or their creatures.
          </Control>
          <Control title="Answer the board" icon="✔️">
            When there is a choice (mana to tap, a target, a block), pick cards and confirm.
          </Control>
        </div>
        <Note>
          The red arrow button <B>forfeits</B>. The game log and chat are beside the board.
          Disconnected? Open the game again to carry on.
        </Note>
      </Section>

      <Section id="keywords">
        <KeywordBadges />
        <Note>
          Badges show above creatures. A <B>yellow ring</B> means another card gave it; <B>+2</B>{' '}
          means 2 more, listed when you <B>View</B> the card.
        </Note>
      </Section>

      <Section id="decks">
        <div className="grid gap-3 sm:grid-cols-3">
          <Stat value="40" label="cards in a deck" />
          <Stat value="4" label="copies of a card, at most" />
          <Stat value="5" label="starter decks to play at once" />
        </div>
        <Note>
          Each game has a <B>format</B> that decides which cards are allowed (Open: all; DM-01 to
          DM-12: the first sets). Build and check decks in the{' '}
          <TextLink href="/decks">deck builder</TextLink>. Start with 1 or 2 civilizations and
          plenty of cheap cards.
        </Note>
      </Section>

      <Section id="games">
        <div className="grid gap-3 sm:grid-cols-3">
          <Stat value="Quick match" label="Paired with someone near your rating." small />
          <Stat value="Open a game" label="Public or invite-only. Others can watch." small />
          <Stat value="Vs computer" label="Practice. Never changes your rating." small />
        </div>
        <ol className="flex flex-wrap items-center justify-center gap-3">
          {RANKS.map(({ rank, from }, index) => (
            <li key={rank.tier} className="flex items-center gap-3">
              {index > 0 && (
                <span aria-hidden className="text-slate-600">
                  →
                </span>
              )}
              <span className="flex flex-col items-center gap-1">
                <RankBadge rank={rank} large />
                <span className="text-xs text-slate-400 tabular-nums">{from}</span>
              </span>
            </li>
          ))}
        </ol>
        <Note>
          Everyone starts at <B>1000</B>. Wins raise it, losses lower it. Games under a minute don’t
          count. After a game, either player can offer a <B>rematch</B>. See the{' '}
          <TextLink href="/rankings">rankings</TextLink>.
        </Note>
      </Section>

      <Section id="tips">
        <ul className="grid gap-3 sm:grid-cols-2">
          <Tip>Add mana every turn early on.</Tip>
          <Tip>Play everything you want before the battle phase.</Tip>
          <Tip>Attacking leaves a creature tapped and open to attack.</Tip>
          <Tip>Low on shields? Keep a blocker back.</Tip>
          <Tip>Count their shields: a few attacks in one turn can finish it.</Tip>
          <Tip>Learn a deck against the computer first.</Tip>
        </ul>
        <Link
          href="/play"
          className="inline-block font-display text-lg font-semibold tracking-wider text-accent uppercase hover:underline"
        >
          Ready? Find a game →
        </Link>
      </Section>
    </div>
  );
}

/** The page's cards from the catalogue. One that fails to load shows its name instead. */
async function loadCards(): Promise<Record<CardName, ShownCard>> {
  const api = await serverApi();
  const entries = await Promise.all(
    (Object.entries(CARD_IDS) as [CardName, string][]).map(async ([name, cardId]) => {
      const { data } = await api.GET('/cards/{cardId}', { params: { path: { cardId } } });
      const shown: ShownCard = data
        ? { name: data.name, imageUrl: data.imageUrl }
        : { name: `Card ${cardId}` };
      return [name, shown] as const;
    }),
  );

  return Object.fromEntries(entries) as Record<CardName, ShownCard>;
}

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
      <div className="space-y-6 p-5 sm:p-6">{children}</div>
    </Panel>
  );
}

function Note({ children }: { readonly children: ReactNode }) {
  return <p className="text-center text-slate-300">{children}</p>;
}

function B({ children }: { readonly children: ReactNode }) {
  return <strong className="font-semibold text-white">{children}</strong>;
}

function TextLink({ href, children }: { readonly href: string; readonly children: ReactNode }) {
  return (
    <Link href={href} className="font-semibold text-accent hover:underline">
      {children}
    </Link>
  );
}

function Civ({ name, colour }: { readonly name: string; readonly colour: string }) {
  return (
    <span className="inline-flex items-center gap-2">
      <span aria-hidden className={`inline-block size-2.5 rotate-45 ${colour}`} />
      {name}
    </span>
  );
}

/** A phase of the turn, with the board's button that moves on from it. */
function Phase({
  n,
  name,
  what,
  next,
}: {
  readonly n: number;
  readonly name: string;
  readonly what: string;
  readonly next: ReactNode;
}) {
  return (
    <li className="flex flex-col gap-2 rounded-lg bg-white/[0.03] p-4 ring-1 ring-slate-600/50">
      <span className="font-display text-sm font-bold tracking-widest text-slate-500">
        PHASE {n}
      </span>
      <span className="font-display text-2xl font-bold tracking-wide text-white uppercase">
        {name}
      </span>
      <span className="flex-1 text-sm text-slate-300">{what}</span>
      <span>{next}</span>
    </li>
  );
}

function Auto() {
  return <span className="text-xs tracking-wider text-slate-500 uppercase">Happens by itself</span>;
}

function Example({
  title,
  note,
  children,
}: {
  readonly title: string;
  readonly note: string;
  readonly children: ReactNode;
}) {
  return (
    <div className="grid items-center gap-4 rounded-lg bg-white/[0.03] p-4 ring-1 ring-slate-600/50 md:grid-cols-[16rem_minmax(0,1fr)]">
      <div className="text-center md:text-left">
        <h3 className="font-display text-xl font-bold tracking-wide text-white uppercase">
          {title}
        </h3>
        <p className="mt-1 text-sm text-slate-400">{note}</p>
      </div>
      <div className="flex justify-center">{children}</div>
    </div>
  );
}

function Control({
  title,
  icon,
  children,
}: {
  readonly title: string;
  readonly icon: string;
  readonly children: ReactNode;
}) {
  return (
    <div className="rounded-lg bg-white/[0.03] p-4 ring-1 ring-slate-600/50">
      <p className="flex items-center gap-2 font-display text-lg font-bold tracking-wide text-white uppercase">
        <span aria-hidden>{icon}</span>
        {title}
      </p>
      <p className="mt-2 text-sm text-slate-300">{children}</p>
    </div>
  );
}

function Stat({
  value,
  label,
  small = false,
}: {
  readonly value: string;
  readonly label: string;
  readonly small?: boolean;
}) {
  return (
    <div className="rounded-lg bg-white/[0.03] p-4 text-center ring-1 ring-slate-600/50">
      <p
        className={`font-display font-bold tracking-wide text-accent uppercase ${small ? 'text-xl' : 'text-5xl'}`}
      >
        {value}
      </p>
      <p className="mt-1 text-sm text-slate-300">{label}</p>
    </div>
  );
}

function Tip({ children }: { readonly children: ReactNode }) {
  return (
    <li className="flex items-start gap-3 rounded-lg bg-white/[0.03] px-4 py-3 text-slate-200 ring-1 ring-slate-600/50">
      <span aria-hidden className="mt-1.5 size-2 shrink-0 rotate-45 bg-accent" />
      {children}
    </li>
  );
}
