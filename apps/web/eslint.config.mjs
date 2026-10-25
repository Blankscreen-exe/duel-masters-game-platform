import { defineConfig, globalIgnores } from 'eslint/config';
import nextVitals from 'eslint-config-next/core-web-vitals';
import nextTs from 'eslint-config-next/typescript';
import tseslint from 'typescript-eslint';

export default defineConfig([
  ...nextVitals,
  ...nextTs,
  {
    files: ['**/*.{ts,tsx}'],
    extends: [tseslint.configs.strictTypeChecked, tseslint.configs.stylisticTypeChecked],
    languageOptions: {
      parserOptions: { projectService: true, tsconfigRootDir: import.meta.dirname },
    },
    rules: {
      '@typescript-eslint/consistent-type-imports': 'error',
      // Braces whenever a body does not fit on the same line as its condition.
      curly: ['error', 'multi-line'],
    },
  },
  // The API types are generated from api/openapi.yaml.
  globalIgnores(['.next/**', 'out/**', 'next-env.d.ts', 'src/lib/api/schema.d.ts']),
]);
