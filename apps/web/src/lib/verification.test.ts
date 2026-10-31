import { describe, expect, it } from 'vitest';
import type { Verification } from './api/client';
import { verificationStep, verifyEmailHref, waitText } from './verification';

const status = (overrides: Partial<Verification>): Verification => ({
  email: 'shobu@example.test',
  mailWorks: true,
  resendInSeconds: 0,
  ...overrides,
});

describe('verificationStep', () => {
  it('waits while the site cannot send email', () => {
    expect(verificationStep(status({ mailWorks: false }))).toBe('waiting');
  });

  it('offers a first code once it can', () => {
    expect(verificationStep(status({}))).toBe('send');
  });

  it('asks for the code once one is sent', () => {
    expect(verificationStep(status({ codeSentAt: '2026-11-01T00:00:00Z' }))).toBe('enter');
  });

  it('still asks for a code already sent if the mail server has since gone', () => {
    expect(verificationStep(status({ mailWorks: false, codeSentAt: '2026-11-01T00:00:00Z' }))).toBe(
      'enter',
    );
  });
});

describe('verifyEmailHref', () => {
  it('leaves out going home, which is where it goes anyway', () => {
    expect(verifyEmailHref('/')).toBe('/verify-email');
  });

  it('keeps where the player was going', () => {
    expect(verifyEmailHref('/decks/1')).toBe('/verify-email?next=%2Fdecks%2F1');
  });
});

describe('waitText', () => {
  it('reads as seconds, then minutes and seconds', () => {
    expect(waitText(45)).toBe('45 s');
    expect(waitText(60)).toBe('1 min');
    expect(waitText(125)).toBe('2 min 5 s');
  });
});
