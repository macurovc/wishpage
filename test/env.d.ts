/// <reference types="@cloudflare/vitest-plugin/types" />

// The vitest plugin injects the parsed migrations as a binding (see
// vitest.config.ts). It is only present in tests.
declare namespace Cloudflare {
  interface Env {
    TEST_MIGRATIONS: { name: string; queries: string[] }[];
  }
}
