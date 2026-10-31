import type { components } from './api/schema';
import { firstParam, hrefWithQuery, wholeNumberParam, type SearchParams } from './search-params';

type Schemas = components['schemas'];
export type AdminUser = Schemas['AdminUser'];
export type AdminUserPage = Schemas['AdminUserPage'];
export type AdminUserStatus = Schemas['AdminUserStatus'];
export type AdminLogAction = Schemas['AdminLogAction'];
export type AdminLogEntry = Schemas['AdminLogEntry'];
export type AdminLogPage = Schemas['AdminLogPage'];
export type AdminSetting = Schemas['AdminSetting'];

/** The admin panel's sections, in its menu. Sections join it as they are built. */
export const ADMIN_SECTIONS = [
  {
    href: '/admin/users',
    label: 'Players',
    description: 'Find players, ban, mute, change roles and passwords.',
  },
  {
    href: '/admin/formats',
    label: 'Formats',
    description: 'Add, change, order and retire the formats games are played in.',
  },
  {
    href: '/admin/computer-opponents',
    label: 'Computer opponents',
    description: 'Add, change and retire the personalities players practise against.',
  },
  {
    href: '/admin/settings',
    label: 'Settings',
    description: 'Every server setting, which applies at once.',
  },
  {
    href: '/admin/mail',
    label: 'Email',
    description: 'The mail server that sends new players their codes.',
  },
  {
    href: '/admin/log',
    label: 'Admin log',
    description: 'Everything admins have done, the latest first.',
  },
] as const;

/** The settings by group, the groups and the settings in each in the order given. */
export function groupSettings(
  settings: readonly AdminSetting[],
): { group: string; settings: AdminSetting[] }[] {
  const groups: { group: string; settings: AdminSetting[] }[] = [];
  for (const setting of settings) {
    const found = groups.find(({ group }) => group === setting.group);
    if (found) found.settings.push(setting);
    else groups.push({ group: setting.group, settings: [setting] });
  }
  return groups;
}

/** What a setting allows, in a few words, or null for a yes-or-no setting. */
export function settingLimits(setting: AdminSetting): string | null {
  switch (setting.kind) {
    case 'number':
      return `From ${String(setting.min)} to ${String(setting.max)}`;
    case 'text':
      return `Up to ${String(setting.maxLength)} characters`;
    default:
      return null;
  }
}

/** The longest name search, as names are at most this long. */
export const ADMIN_SEARCH_MAX_LENGTH = 20;

/** The filters of the players list, with their labels. */
export const ADMIN_USER_STATUSES: readonly {
  readonly status?: AdminUserStatus;
  readonly label: string;
}[] = [
  { label: 'Everyone' },
  { status: 'banned', label: 'Banned' },
  { status: 'muted', label: 'Muted' },
  { status: 'admin', label: 'Admins' },
  { status: 'unverified', label: 'Not confirmed' },
];

const STATUSES = new Set<string>(ADMIN_USER_STATUSES.flatMap(({ status }) => status ?? []));

/** The players list as a page address asks for it. */
export interface AdminUsersQuery {
  readonly search?: string;
  readonly status?: AdminUserStatus;
  readonly page?: number;
}

/**
 * The players list asked for in a page address (/admin/users?search=…&status=banned&page=2).
 * Anything not valid is left out, so a hand-edited address shows everyone rather than an error.
 */
export function parseAdminUsersQuery(params: SearchParams): AdminUsersQuery {
  const search = firstParam(params['search'])?.trim().slice(0, ADMIN_SEARCH_MAX_LENGTH);
  const status = firstParam(params['status']);
  const page = wholeNumberParam(params['page'], 1);

  return {
    ...(search ? { search } : {}),
    ...(status !== undefined && STATUSES.has(status) ? { status: status as AdminUserStatus } : {}),
    ...(page === undefined ? {} : { page }),
  };
}

/** A page of the players list. */
export function adminUsersHref(query: AdminUsersQuery, page = query.page ?? 1): string {
  return hrefWithQuery('/admin/users', { search: query.search, status: query.status, page });
}

/** What each admin log action is called. */
export const ADMIN_LOG_ACTIONS: Readonly<Record<AdminLogAction, string>> = {
  user_banned: 'Banned',
  user_unbanned: 'Lifted a ban',
  user_muted: 'Muted',
  user_unmuted: 'Lifted a mute',
  role_changed: 'Changed a role',
  password_reset: 'Gave a new password',
  player_let_in: 'Let a player in',
  setting_changed: 'Changed a setting',
  setting_reset: 'Reset a setting',
  chat_message_deleted: 'Deleted a chat message',
  game_stopped: 'Stopped a game',
  computer_opponent_added: 'Added a computer opponent',
  computer_opponent_changed: 'Changed a computer opponent',
  computer_opponent_offered: 'Offered a computer opponent',
  computer_opponent_retired: 'Retired a computer opponent',
  format_added: 'Added a format',
  format_changed: 'Changed a format',
  format_retired: 'Retired a format',
  format_restored: 'Brought back a format',
  formats_reordered: 'Reordered the formats',
  mail_settings_changed: 'Changed the mail server',
  mail_settings_cleared: 'Forgot the mail server',
};

/** The admin log as a page address asks for it. */
export interface AdminLogQuery {
  readonly action?: AdminLogAction;
  readonly page?: number;
}

/** The admin log asked for in a page address (/admin/log?action=user_banned&page=2). */
export function parseAdminLogQuery(params: SearchParams): AdminLogQuery {
  const action = firstParam(params['action']);
  const page = wholeNumberParam(params['page'], 1);

  return {
    ...(action !== undefined && Object.hasOwn(ADMIN_LOG_ACTIONS, action)
      ? { action: action as AdminLogAction }
      : {}),
    ...(page === undefined ? {} : { page }),
  };
}

/** A page of the admin log. */
export function adminLogHref(query: AdminLogQuery, page = query.page ?? 1): string {
  return hrefWithQuery('/admin/log', { action: query.action, page });
}
