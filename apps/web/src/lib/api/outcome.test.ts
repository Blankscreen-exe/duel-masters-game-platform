import { describe, expect, it } from 'vitest';
import { UNREACHABLE } from './client';
import { attempt, attemptEmpty, attemptFound } from './outcome';

const ok = new Response(null, { status: 200 });
const refused = new Response(null, { status: 409 });
const refusal = { code: 'busy', message: 'You are already in a game.' };

describe('attempt', () => {
  it('gives the answer', async () => {
    expect(await attempt(() => Promise.resolve({ data: { id: 'g1' }, response: ok }))).toEqual({
      ok: true,
      value: { id: 'g1' },
    });
  });

  it('gives the server’s reason for a refusal', async () => {
    expect(await attempt(() => Promise.resolve({ error: refusal, response: refused }))).toEqual({
      ok: false,
      message: 'You are already in a game.',
    });
  });

  it('says when the server cannot be reached', async () => {
    expect(await attempt(() => Promise.reject(new TypeError('Failed to fetch')))).toEqual({
      ok: false,
      message: UNREACHABLE,
    });
  });
});

describe('attemptEmpty', () => {
  it('succeeds without an answer', async () => {
    expect(await attemptEmpty(() => Promise.resolve({ response: ok }))).toEqual({
      ok: true,
      value: null,
    });
  });

  it('gives the server’s reason for a refusal', async () => {
    expect(
      await attemptEmpty(() => Promise.resolve({ error: refusal, response: refused })),
    ).toEqual({ ok: false, message: 'You are already in a game.' });
  });

  it('says when the server cannot be reached', async () => {
    expect(await attemptEmpty(() => Promise.reject(new TypeError('Failed to fetch')))).toEqual({
      ok: false,
      message: UNREACHABLE,
    });
  });
});

describe('attemptFound', () => {
  const missing = new Response(null, { status: 404 });

  it('gives the answer, or null for something not found', async () => {
    expect(await attemptFound(() => Promise.resolve({ data: { id: 'g1' }, response: ok }))).toEqual(
      { ok: true, value: { id: 'g1' } },
    );
    expect(await attemptFound(() => Promise.resolve({ response: missing }))).toEqual({
      ok: true,
      value: null,
    });
  });

  it('gives the server’s reason for a refusal', async () => {
    expect(
      await attemptFound(() => Promise.resolve({ error: refusal, response: refused })),
    ).toEqual({ ok: false, message: 'You are already in a game.' });
  });

  it('says when the server cannot be reached', async () => {
    expect(await attemptFound(() => Promise.reject(new TypeError('Failed to fetch')))).toEqual({
      ok: false,
      message: UNREACHABLE,
    });
  });
});
