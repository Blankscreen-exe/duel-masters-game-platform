import { describe, expect, it } from 'vitest';
import { allParams, firstParam, hrefWithQuery, pageCount, wholeNumberParam } from './search-params';

describe('firstParam and allParams', () => {
  it('read a parameter given once, several times or not at all', () => {
    expect(firstParam('a')).toBe('a');
    expect(firstParam(['a', 'b'])).toBe('a');
    expect(firstParam(undefined)).toBeUndefined();
    expect(allParams('a')).toEqual(['a']);
    expect(allParams(['a', 'b'])).toEqual(['a', 'b']);
    expect(allParams(undefined)).toEqual([]);
  });
});

describe('wholeNumberParam', () => {
  it('keeps whole numbers from the minimum up, and nothing else', () => {
    expect(wholeNumberParam('3', 1)).toBe(3);
    expect(wholeNumberParam(['2', '9'], 1)).toBe(2);
    expect(wholeNumberParam('0', 1)).toBeUndefined();
    expect(wholeNumberParam('0', 0)).toBe(0);
    for (const odd of ['-1', '1.5', 'two', '', '1234567', undefined]) {
      expect(wholeNumberParam(odd, 0)).toBeUndefined();
    }
  });
});

describe('pageCount', () => {
  it('rounds up, and has a page even when the list is empty', () => {
    expect(pageCount(0, 60)).toBe(1);
    expect(pageCount(60, 60)).toBe(1);
    expect(pageCount(61, 60)).toBe(2);
  });
});

describe('hrefWithQuery', () => {
  it('leaves out empty values and the first page', () => {
    expect(hrefWithQuery('/rankings', {})).toBe('/rankings');
    expect(hrefWithQuery('/rankings', { search: '', page: 1 })).toBe('/rankings');
    expect(hrefWithQuery('/rankings', { search: 'shobu & co', page: 2 })).toBe(
      '/rankings?search=shobu+%26+co&page=2',
    );
    expect(hrefWithQuery('/rankings', { search: undefined, page: 3 })).toBe('/rankings?page=3');
  });
});
