import createClient from 'openapi-fetch';
import { describe, expect, it } from 'vitest';
import { jsonChanges } from './json-changes';
import type { paths } from './schema';

/** A client whose requests are caught rather than sent, to look at what it would send. */
function recordingClient() {
  const sent: Request[] = [];
  const client = createClient<paths>({
    baseUrl: 'http://site.invalid/api',
    fetch: (request: Request) => {
      sent.push(request);
      return Promise.resolve(new Response(null, { status: 204 }));
    },
  });
  client.use(jsonChanges);
  return { client, sent };
}

describe('jsonChanges', () => {
  it('marks changes without a body as JSON', async () => {
    const { client, sent } = recordingClient();

    await client.DELETE('/decks/{deckId}', { params: { path: { deckId: 'd1' } } });
    await client.POST('/auth/logout');

    expect(sent.map((request) => request.headers.get('Content-Type'))).toEqual([
      'application/json',
      'application/json',
    ]);
  });

  it('keeps changes with a body as they are', async () => {
    const { client, sent } = recordingClient();

    await client.POST('/decks/text', { body: { text: '4 Bolshack Dragon' } });

    expect(sent[0]?.headers.get('Content-Type')).toBe('application/json');
    expect(await sent[0]?.text()).toBe('{"text":"4 Bolshack Dragon"}');
  });

  it('leaves reads alone', async () => {
    const { client, sent } = recordingClient();

    await client.GET('/formats');

    expect(sent[0]?.headers.has('Content-Type')).toBe(false);
  });
});
