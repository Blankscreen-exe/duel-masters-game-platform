import { describe, expect, it } from 'vitest';
import { textField } from './form-data';

describe('textField', () => {
  it('reads a text box, and nothing for one that is missing or is a file', () => {
    const form = new FormData();
    form.set('name', 'Shobu');
    form.set('picture', new Blob(['x']), 'x.png');

    expect(textField(form, 'name')).toBe('Shobu');
    expect(textField(form, 'missing')).toBe('');
    expect(textField(form, 'picture')).toBe('');
  });
});
