import { describe, expect, it } from 'vitest';
import { goServerTarget, goServerUrl } from './go-server-url';

describe('goServerUrl', () => {
  it('defaults to the server on this machine', () => {
    expect(goServerUrl({})).toBe('http://127.0.0.1:8080');
  });

  it('reads GO_SERVER_URL without a trailing slash', () => {
    expect(goServerUrl({ GO_SERVER_URL: 'http://server:8080/' })).toBe('http://server:8080');
  });

  it('refuses anything but a web address', () => {
    for (const value of ['', 'not a url', 'server:8080', 'ftp://server']) {
      expect(() => goServerUrl({ GO_SERVER_URL: value }), value).toThrow(/GO_SERVER_URL/);
    }
  });
});

describe('goServerTarget', () => {
  it('keeps the path and query', () => {
    expect(
      goServerTarget('/api/cards', '?q=bolshack&civilization=Fire', 'http://server:8080').href,
    ).toBe('http://server:8080/api/cards?q=bolshack&civilization=Fire');
  });

  it('stays on the Go server whatever the path', () => {
    for (const path of ['/api//evil.example/x', '/api/../../etc', '/api/%2F%2Fevil.example']) {
      expect(goServerTarget(path, '', 'http://server:8080').host, path).toBe('server:8080');
    }
  });
});
