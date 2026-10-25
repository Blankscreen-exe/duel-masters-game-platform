/**
 * The username and password rules, for the forms to check as the player types. The Go server
 * checks them for real; credentials.test.ts keeps these numbers equal to the API contract's.
 */
export const USERNAME_MIN_LENGTH = 3;
export const USERNAME_MAX_LENGTH = 20;
/**
 * Letters, digits, underscores and hyphens. Used as an HTML pattern attribute, which browsers
 * compile with the v flag, so the hyphen is escaped.
 */
export const USERNAME_PATTERN = String.raw`[A-Za-z0-9_\-]+`;

export const PASSWORD_MIN_LENGTH = 8;
export const PASSWORD_MAX_LENGTH = 128;
