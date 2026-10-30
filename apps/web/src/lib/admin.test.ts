import { describe, expect, it } from 'vitest';
import {
  ADMIN_LOG_ACTIONS,
  groupSettings,
  settingLimits,
  type AdminSetting,
  adminLogHref,
  adminUsersHref,
  parseAdminLogQuery,
  parseAdminUsersQuery,
} from './admin';

describe('parseAdminUsersQuery and adminUsersHref', () => {
  it('reads a search, a status and a page, and leaves out what is not valid', () => {
    expect(parseAdminUsersQuery({ search: ' koku ', status: 'banned', page: '2' })).toEqual({
      search: 'koku',
      status: 'banned',
      page: 2,
    });
    expect(parseAdminUsersQuery({ search: '  ', status: 'sleeping', page: '0' })).toEqual({});
    expect(parseAdminUsersQuery({ search: 'x'.repeat(30) }).search).toHaveLength(20);
  });

  it('writes them back', () => {
    expect(adminUsersHref({})).toBe('/admin/users');
    expect(adminUsersHref({ search: 'koku', status: 'muted', page: 3 })).toBe(
      '/admin/users?search=koku&status=muted&page=3',
    );
    expect(adminUsersHref({ status: 'admin', page: 3 }, 1)).toBe('/admin/users?status=admin');
  });
});

describe('parseAdminLogQuery and adminLogHref', () => {
  it('reads an action and a page, and leaves out what is not valid', () => {
    expect(parseAdminLogQuery({ action: 'user_banned', page: '2' })).toEqual({
      action: 'user_banned',
      page: 2,
    });
    expect(parseAdminLogQuery({ action: 'teleported' })).toEqual({});
    expect(parseAdminLogQuery({ action: 'toString' })).toEqual({});
  });

  it('writes them back', () => {
    expect(adminLogHref({})).toBe('/admin/log');
    expect(adminLogHref({ action: 'game_stopped', page: 2 })).toBe(
      '/admin/log?action=game_stopped&page=2',
    );
  });

  it('names every action', () => {
    expect(ADMIN_LOG_ACTIONS.user_banned).toBe('Banned');
    expect(Object.keys(ADMIN_LOG_ACTIONS)).toHaveLength(14);
  });
});

describe('settings', () => {
  const setting = (overrides: Partial<AdminSetting>): AdminSetting => ({
    key: 'a.b',
    group: 'Games',
    description: 'A setting.',
    kind: 'boolean',
    default: 'true',
    value: 'true',
    isDefault: true,
    ...overrides,
  });

  it('groups them, keeping the order given', () => {
    const groups = groupSettings([
      setting({ key: '1', group: 'Games' }),
      setting({ key: '2', group: 'Lobby' }),
      setting({ key: '3', group: 'Games' }),
    ]);

    expect(groups.map(({ group }) => group)).toEqual(['Games', 'Lobby']);
    expect(groups[0]?.settings.map(({ key }) => key)).toEqual(['1', '3']);
  });

  it('says what a setting allows', () => {
    expect(settingLimits(setting({ kind: 'number', min: 0, max: 5000 }))).toBe('From 0 to 5000');
    expect(settingLimits(setting({ kind: 'text', maxLength: 20 }))).toBe('Up to 20 characters');
    expect(settingLimits(setting({}))).toBeNull();
  });
});
