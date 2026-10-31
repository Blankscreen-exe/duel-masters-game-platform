import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';
import { parse } from 'yaml';
import type { Card } from './api/client';
import {
  changedLimit,
  FORMAT_DESCRIPTION_MAX_LENGTH,
  FORMAT_NAME_MAX_LENGTH,
  formatFormProblem,
  formatRulesText,
  formatsForOpponent,
  formOfFormat,
  LIMIT_CHOICES,
  movedFormat,
  NEW_FORMAT,
  sameFormatForm,
  toFormatInput,
  withLimit,
  withoutLimit,
  type AdminFormat,
  type FormatForm,
} from './formats';

function card(id: string, name: string): Card {
  return { id, name } as unknown as Card;
}

const saved: AdminFormat = {
  id: 'f1',
  name: 'Legends',
  description: 'Old cards.',
  allSets: false,
  sets: ['DM-01', 'DM-02'],
  limits: [
    { cardId: 'a', maxCopies: 0, card: card('a', 'Aqua') },
    { cardId: 'b', maxCopies: 1 },
  ],
  retired: false,
  opponents: [],
};

const form: FormatForm = formOfFormat(saved);

describe('the limits', () => {
  it('are the API contract’s', () => {
    const contract = parse(
      readFileSync(fileURLToPath(new URL('../../../../api/openapi.yaml', import.meta.url)), 'utf8'),
    ) as {
      components: {
        schemas: {
          FormatInput: {
            properties: { name: { maxLength: number }; description: { maxLength: number } };
          };
          FormatCardLimit: { properties: { maxCopies: { minimum: number; maximum: number } } };
        };
      };
    };
    const { FormatInput, FormatCardLimit } = contract.components.schemas;

    expect(FORMAT_NAME_MAX_LENGTH).toBe(FormatInput.properties.name.maxLength);
    expect(FORMAT_DESCRIPTION_MAX_LENGTH).toBe(FormatInput.properties.description.maxLength);
    expect(LIMIT_CHOICES[0].maxCopies).toBe(FormatCardLimit.properties.maxCopies.minimum);
    expect(LIMIT_CHOICES.at(-1)?.maxCopies).toBe(FormatCardLimit.properties.maxCopies.maximum);
  });
});

describe('formOfFormat and toFormatInput', () => {
  it('goes from what the API shows to what it saves', () => {
    expect(toFormatInput(form)).toEqual({
      name: 'Legends',
      description: 'Old cards.',
      allSets: false,
      sets: ['DM-01', 'DM-02'],
      limits: [
        { cardId: 'a', maxCopies: 0 },
        { cardId: 'b', maxCopies: 1 },
      ],
    });
  });

  it('saves no sets while every set is allowed', () => {
    expect(toFormatInput({ ...form, allSets: true }).sets).toEqual([]);
  });
});

describe('sameFormatForm', () => {
  it('ignores spaces around the words and the order of sets and limits', () => {
    expect(sameFormatForm({ ...form, name: ' Legends ', description: ' Old cards. ' }, form)).toBe(
      true,
    );
    expect(sameFormatForm({ ...form, sets: ['DM-02', 'DM-01'] }, form)).toBe(true);
    expect(sameFormatForm({ ...form, limits: [...form.limits].reverse() }, form)).toBe(true);
  });

  it('ignores sets kept aside while every set is allowed', () => {
    const every = { ...form, allSets: true };

    expect(sameFormatForm({ ...every, sets: [] }, every)).toBe(true);
  });

  it('sees a change', () => {
    expect(sameFormatForm({ ...form, name: 'Other' }, form)).toBe(false);
    expect(sameFormatForm({ ...form, allSets: true }, form)).toBe(false);
    expect(sameFormatForm({ ...form, sets: ['DM-01'] }, form)).toBe(false);
    expect(sameFormatForm(changedLimit(form, 'a', 2), form)).toBe(false);
  });
});

describe('formatFormProblem', () => {
  it('finds nothing wrong with a whole form', () => {
    expect(formatFormProblem(form)).toBeNull();
    expect(formatFormProblem({ ...form, allSets: true, sets: [] })).toBeNull();
  });

  it('says what is missing', () => {
    expect(formatFormProblem(NEW_FORMAT)).toBe('The format needs a name.');
    expect(formatFormProblem({ ...form, description: ' ' })).toBe(
      'The format needs a description.',
    );
    expect(formatFormProblem({ ...form, sets: [] })).toBe('Choose at least one set.');
  });
});

describe('the limits on a form', () => {
  const zebra = card('z', 'Zebra');
  const empty: FormatForm = { ...form, limits: [] };

  it('adds a card, in order of name', () => {
    const limits = withLimit(withLimit(empty, zebra, 0), card('m', 'Mole'), 1).limits;

    expect(limits.map(({ cardId, maxCopies }) => [cardId, maxCopies])).toEqual([
      ['m', 1],
      ['z', 0],
    ]);
  });

  it('changes a card already limited, rather than listing it twice', () => {
    const limits = withLimit(withLimit(empty, zebra, 0), zebra, 2).limits;

    expect(limits.map(({ maxCopies }) => maxCopies)).toEqual([2]);
  });

  it('changes and removes one by id, whether or not it has details', () => {
    expect(changedLimit(form, 'b', 2).limits[1]).toEqual({ cardId: 'b', maxCopies: 2 });
    expect(withoutLimit(form, 'b').limits.map(({ cardId }) => cardId)).toEqual(['a']);
  });
});

describe('formatRulesText', () => {
  it('says what the format allows', () => {
    expect(formatRulesText(saved)).toBe('2 sets · 1 banned · 1 limited');
    expect(formatRulesText({ ...saved, allSets: true, sets: [], limits: [] })).toBe('Every set');
    expect(formatRulesText({ ...saved, sets: ['DM-01'], limits: [] })).toBe('Set DM-01');
  });
});

describe('movedFormat', () => {
  const ids = ['a', 'b', 'c'];

  it('moves one a place', () => {
    expect(movedFormat(ids, 'b', 'up')).toEqual(['b', 'a', 'c']);
    expect(movedFormat(ids, 'b', 'down')).toEqual(['a', 'c', 'b']);
  });

  it('cannot move past the ends, or one it does not have', () => {
    expect(movedFormat(ids, 'a', 'up')).toBeNull();
    expect(movedFormat(ids, 'c', 'down')).toBeNull();
    expect(movedFormat(ids, 'x', 'up')).toBeNull();
  });
});

describe('formatsForOpponent', () => {
  const other = { ...saved, id: 'f2', name: 'Old', retired: true };

  it('leaves out retired formats', () => {
    expect(formatsForOpponent([saved, other], 'f1').map(({ id }) => id)).toEqual(['f1']);
  });

  it('keeps the one the opponent has, marked', () => {
    expect(formatsForOpponent([saved, other], 'f2').map(({ name }) => name)).toEqual([
      'Legends',
      'Old (retired)',
    ]);
  });
});
