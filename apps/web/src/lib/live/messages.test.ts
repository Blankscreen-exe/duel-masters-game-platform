import { describe, expect, it, vi } from 'vitest';
import { LiveListeners, parseLiveMessage, reconnectDelay, socketUrl } from './messages';

describe('parseLiveMessage', () => {
  it('reads a message with a type', () => {
    expect(parseLiveMessage('{"type":"game_started","gameId":"g1"}')).toEqual({
      type: 'game_started',
      gameId: 'g1',
    });
  });

  it('passes on types it does not know, which nobody listens to', () => {
    expect(parseLiveMessage('{"type":"something_new"}')).toEqual({ type: 'something_new' });
  });

  it('ignores anything else', () => {
    for (const data of ['not json', '42', 'null', '{"gameId":"g1"}', '{"type":7}', new Blob()]) {
      expect(parseLiveMessage(data)).toBeNull();
    }
  });
});

describe('LiveListeners', () => {
  it('passes each message to the listeners of its type', () => {
    const listeners = new LiveListeners();
    const started = vi.fn();
    const chat = vi.fn();
    listeners.on('game_started', started);
    listeners.on('chat_message', chat);

    listeners.emit({ type: 'game_started', gameId: 'g1' });

    expect(started).toHaveBeenCalledWith({ type: 'game_started', gameId: 'g1' });
    expect(chat).not.toHaveBeenCalled();
  });

  it('stops passing messages once a listener leaves', () => {
    const listeners = new LiveListeners();
    const started = vi.fn();
    const off = listeners.on('game_started', started);

    off();
    listeners.emit({ type: 'game_started', gameId: 'g1' });

    expect(started).not.toHaveBeenCalled();
  });

  it('lets a listener leave while a message is being passed on', () => {
    const listeners = new LiveListeners();
    const second = vi.fn();
    const off = listeners.on('game_started', () => {
      off();
    });
    listeners.on('game_started', second);

    listeners.emit({ type: 'game_started', gameId: 'g1' });

    expect(second).toHaveBeenCalledTimes(1);
  });
});

describe('reconnectDelay', () => {
  it('doubles from a second up to ten', () => {
    const middle = () => 0.5;
    expect([0, 1, 2, 3, 4, 10].map((attempt) => reconnectDelay(attempt, middle))).toEqual([
      1000, 2000, 4000, 8000, 10_000, 10_000,
    ]);
  });

  it('varies a little, so a lobby does not reconnect at once', () => {
    expect(reconnectDelay(0, () => 0)).toBe(800);
    expect(reconnectDelay(0, () => 1)).toBe(1200);
  });
});

describe('socketUrl', () => {
  it('uses a secure websocket on a secure page', () => {
    expect(socketUrl({ protocol: 'https:', host: 'duelmasters.example' }, '/api/realtime')).toBe(
      'wss://duelmasters.example/api/realtime',
    );
    expect(socketUrl({ protocol: 'http:', host: 'localhost:3000' }, '/api/games/g1/socket')).toBe(
      'ws://localhost:3000/api/games/g1/socket',
    );
  });
});
