import type { components } from './api/schema';

type Schemas = components['schemas'];
export type MailSettings = Schemas['MailSettings'];
export type MailSettingsInput = Schemas['MailSettingsInput'];
export type MailSecurity = Schemas['MailSecurity'];

/** How the connection can be protected. */
export const MAIL_SECURITIES: readonly {
  readonly value: MailSecurity;
  readonly label: string;
}[] = [
  { value: 'starttls', label: 'STARTTLS (usually port 587)' },
  { value: 'tls', label: 'TLS from the start (usually port 465)' },
  { value: 'none', label: 'None, for a server on this machine' },
];

/** What a new mail server starts as: the most common way of connecting. */
export const DEFAULT_MAIL_SECURITY: MailSecurity = 'starttls';
export const DEFAULT_MAIL_PORT = 587;

/** What the password field and its checkbox mean: a new one, none, or the one stored. */
export function mailPassword(typed: string, remove: boolean): string | undefined {
  if (remove) return '';
  return typed === '' ? undefined : typed;
}

/** What the admin is told about the mail server as it stands. */
export function mailStatusText(settings: MailSettings): string {
  if (!settings.keyAvailable) {
    return 'The server has no APP_SECRET_KEY, so mail settings cannot be stored. Set it and restart the server.';
  }
  if (settings.unreadable) {
    return 'The stored settings cannot be read with this APP_SECRET_KEY, which must have changed. Enter them again.';
  }
  if (!settings.configured) {
    return 'No mail server is set up. New players wait until one is, or until you let them in from the Players page.';
  }
  return 'New players are sent their code through this server. Everything here is stored encrypted.';
}
