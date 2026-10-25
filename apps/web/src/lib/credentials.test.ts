import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';
import { parse } from 'yaml';
import {
  PASSWORD_MAX_LENGTH,
  PASSWORD_MIN_LENGTH,
  USERNAME_MAX_LENGTH,
  USERNAME_MIN_LENGTH,
  USERNAME_PATTERN,
} from './credentials';

interface StringSchema {
  minLength: number;
  maxLength: number;
  pattern?: string;
}

const contract = parse(
  readFileSync(fileURLToPath(new URL('../../../../api/openapi.yaml', import.meta.url)), 'utf8'),
) as {
  components: {
    schemas: {
      Credentials: { properties: { username: StringSchema; password: StringSchema } };
      PasswordChange: { properties: { newPassword: StringSchema } };
    };
  };
};

describe('the credential rules', () => {
  const { username, password } = contract.components.schemas.Credentials.properties;
  const { newPassword } = contract.components.schemas.PasswordChange.properties;

  it('match the API contract', () => {
    expect([USERNAME_MIN_LENGTH, USERNAME_MAX_LENGTH]).toEqual([
      username.minLength,
      username.maxLength,
    ]);
    expect([PASSWORD_MIN_LENGTH, PASSWORD_MAX_LENGTH]).toEqual([
      password.minLength,
      password.maxLength,
    ]);
    expect([PASSWORD_MIN_LENGTH, PASSWORD_MAX_LENGTH]).toEqual([
      newPassword.minLength,
      newPassword.maxLength,
    ]);
  });

  it('accept and refuse the same names as the contract', () => {
    const ours = new RegExp(`^(?:${USERNAME_PATTERN})$`, 'v');
    const theirs = new RegExp(username.pattern ?? '');

    for (const name of ['Shobu', 'blank_screen', 'Hakuoh-99', 'two words', 'émile', 'name!', '']) {
      expect(ours.test(name), name).toBe(theirs.test(name));
    }
  });
});
