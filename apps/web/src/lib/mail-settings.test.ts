import { describe, expect, it } from 'vitest';
import { mailPassword, mailStatusText, type MailSettings } from './mail-settings';

const settings = (overrides: Partial<MailSettings>): MailSettings => ({
  configured: false,
  keyAvailable: true,
  unreadable: false,
  hasPassword: false,
  ...overrides,
});

describe('mailPassword', () => {
  it('sends a new password', () => {
    expect(mailPassword('secret', false)).toBe('secret');
  });

  it('leaves out an empty field, which keeps the stored one', () => {
    expect(mailPassword('', false)).toBeUndefined();
  });

  it('sends empty to remove it, whatever is typed', () => {
    expect(mailPassword('', true)).toBe('');
    expect(mailPassword('secret', true)).toBe('');
  });
});

describe('mailStatusText', () => {
  it('names a missing key first, as nothing else can be done', () => {
    expect(mailStatusText(settings({ keyAvailable: false }))).toContain('APP_SECRET_KEY');
  });

  it('says when the stored settings cannot be read', () => {
    expect(mailStatusText(settings({ unreadable: true }))).toContain('cannot be read');
  });

  it('says what new players do without a mail server', () => {
    expect(mailStatusText(settings({}))).toContain('let them in');
  });

  it('says it works once set up', () => {
    expect(mailStatusText(settings({ configured: true }))).toContain('stored encrypted');
  });
});
