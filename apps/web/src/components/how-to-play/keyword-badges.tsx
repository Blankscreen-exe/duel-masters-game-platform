'use client';

import { StatusBadgeIcon, statusBadges } from '@duel-masters/duel-board';

/** A keyword the duel board shows as a badge above a creature, explained for the how to play page. */
interface BadgeKeyword {
  /** The status the board names the badge after. */
  readonly id: string;
  readonly name: string;
  readonly meaning: string;
}

/**
 * The board's badges, in the order the page lists them: attacking, defending, then the rest. Drawn
 * with the board's own badge, so the picture here is the one players see in a game.
 */
const BADGE_KEYWORDS: readonly BadgeKeyword[] = [
  {
    id: 'blocker',
    name: 'Blocker',
    meaning:
      'When an opponent’s creature attacks, you may tap this untapped creature to make the attacker battle it instead. A blocked attack breaks no shields.',
  },
  {
    id: 'speed_attacker',
    name: 'Speed attacker',
    meaning: 'Has no summoning sickness: it can attack the turn it is summoned.',
  },
  {
    id: 'power_attacker',
    name: 'Power attacker',
    meaning:
      'Gets extra power while it attacks, by the amount on the card (such as +2000). It has its normal power when it is attacked or blocks.',
  },
  {
    id: 'doublebreaker',
    name: 'Double breaker',
    meaning: 'Breaks 2 shields when it attacks a player, instead of 1.',
  },
  {
    id: 'triplebreaker',
    name: 'Triple breaker',
    meaning: 'Breaks 3 shields when it attacks a player.',
  },
  {
    id: 'slayer',
    name: 'Slayer',
    meaning:
      'When it loses a battle, the creature it battled is destroyed too, whatever that creature’s power.',
  },
  {
    id: 'cant_be_blocked',
    name: 'Can’t be blocked',
    meaning: 'Blockers can’t stop its attacks.',
  },
  {
    id: 'stealth',
    name: 'Stealth',
    meaning:
      'Can’t be blocked while the opponent has a card of a certain civilization in their mana zone (such as water stealth).',
  },
  {
    id: 'attack_untapped',
    name: 'Attacks untapped creatures',
    meaning: 'Can attack the opponent’s creatures even when they are untapped.',
  },
  {
    id: 'treated_as_tapped',
    name: 'Attacked as if tapped',
    meaning: 'Can be attacked even while it is untapped.',
  },
  {
    id: 'cant_be_attacked',
    name: 'Can’t be attacked',
    meaning: 'Opposing creatures can’t choose it as the target of an attack.',
  },
  {
    id: 'cant_attack_players',
    name: 'Can’t attack players',
    meaning: 'It may only attack creatures.',
  },
  {
    id: 'cant_attack_creatures',
    name: 'Can’t attack creatures',
    meaning: 'It may only attack players.',
  },
  {
    id: 'tap_ability',
    name: 'Tap ability',
    meaning:
      'In the battle phase, instead of attacking, it can tap to use the ability written on the card.',
  },
  {
    id: 'silent_skill',
    name: 'Silent skill',
    meaning:
      'At the start of your turn, a tapped creature with silent skill can stay tapped instead of untapping, to use its ability.',
  },
  {
    id: 'survivor',
    name: 'Survivor',
    meaning:
      'Each of your survivors has the survivor abilities of all your other survivors in the battle zone.',
  },
  {
    id: 'wave_striker',
    name: 'Wave striker',
    meaning:
      'Its wave striker ability only works while 2 or more other wave strikers are in the battle zone.',
  },
  {
    id: 'cant_be_selected_by_opp',
    name: 'Can’t be chosen',
    meaning: 'The opponent’s spells and abilities can’t choose it.',
  },
  {
    id: 'cant_be_tapped_by_opp',
    name: 'Can’t be tapped',
    meaning: 'The opponent’s spells and abilities can’t tap it.',
  },
  {
    id: 'doesnt_untap',
    name: 'Doesn’t untap',
    meaning: 'Stays tapped at the start of your turn.',
  },
  {
    id: 'destroy_after_battle',
    name: 'Destroyed after battle',
    meaning: 'Is destroyed after it battles, even when it wins.',
  },
  {
    id: 'summoning_sickness',
    name: 'Summoning sickness',
    meaning:
      'A creature summoned this turn can’t attack or use a tap ability until your next turn. The board also dims it.',
  },
];

/** The keywords shown as badges on the duel board, each with its badge. */
export function KeywordBadges() {
  return (
    <ul className="grid gap-x-8 gap-y-4 md:grid-cols-2">
      {BADGE_KEYWORDS.map((keyword) => {
        const [badge] = statusBadges([{ id: keyword.id, value: 0 }]);
        return (
          <li key={keyword.id} className="flex gap-3">
            <span className="mt-0.5 text-base">{badge && <StatusBadgeIcon badge={badge} />}</span>
            <span>
              <span className="font-semibold text-white">{keyword.name}.</span>{' '}
              <span className="text-slate-300">{keyword.meaning}</span>
            </span>
          </li>
        );
      })}
    </ul>
  );
}
