'use client';

import { StatusBadgeIcon, statusBadges } from '@duel-masters/duel-board';

/**
 * The keywords the duel board shows as badges above creatures, attacking first, then defending,
 * then the rest, as `[status, example value]`. Each is drawn with the board's own badge and
 * described in the board's own words, so this list says what a player sees in a game.
 */
const BADGE_KEYWORDS: readonly (readonly [string, number?])[] = [
  ['speed_attacker'],
  ['power_attacker', 2000],
  ['doublebreaker'],
  ['triplebreaker'],
  ['slayer'],
  ['cant_be_blocked'],
  ['stealth'],
  ['attack_untapped'],
  ['blocker'],
  ['cant_be_attacked'],
  ['cant_be_selected_by_opp'],
  ['cant_be_tapped_by_opp'],
  ['tap_ability'],
  ['silent_skill'],
  ['survivor'],
  ['wave_striker'],
  ['treated_as_tapped'],
  ['cant_attack_players'],
  ['cant_attack_creatures'],
  ['doesnt_untap'],
  ['destroy_after_battle'],
  ['summoning_sickness'],
];

/** The board's keyword badges, each with what it means. */
export function KeywordBadges() {
  const badges = statusBadges(BADGE_KEYWORDS.map(([id, value]) => ({ id, value })));

  return (
    <ul className="grid gap-x-6 gap-y-2.5 sm:grid-cols-2 xl:grid-cols-3">
      {badges.map((badge) => {
        const [name, meaning] = splitLabel(badge.label);
        return (
          <li key={badge.id} className="flex items-center gap-3 text-sm">
            <span className="text-base">
              <StatusBadgeIcon badge={badge} />
            </span>
            <span>
              <span className="font-semibold text-white">{name}</span>
              {meaning && <span className="text-slate-400">: {meaning}</span>}
            </span>
          </li>
        );
      })}
    </ul>
  );
}

/** "Blocker: can stop an attack" as its name and meaning; a label without a colon is all name. */
function splitLabel(label: string): readonly [string, string | undefined] {
  const at = label.indexOf(': ');
  return at === -1 ? [label, undefined] : [label.slice(0, at), label.slice(at + 2)];
}
