import { defineConfig } from "vitest/config";

import { EXCLUDED_FROM_CHECKS } from "./lib/untested-modules.ts";

export default defineConfig({
  root: import.meta.dirname,
  test: {
    environment: "node",
    include: ["**/*.test.ts"],
    // Several suites here are repository-wide gates — the marker baseline, the
    // boilerplate / sample removal targets, the 1:1 export gate — and each walks
    // every tracked file. Their runtime therefore tracks the size of the
    // repository rather than the size of the code under test, and it only grows
    // as the tree does. vitest's 5s default is sized for a unit test and has no
    // relation to that, so it is raised here rather than per suite: a gate added
    // later inherits the right budget instead of discovering the wrong one.
    testTimeout: 10_000,
    coverage: {
      provider: "v8",
      include: ["**/*.ts"],
      exclude: [...EXCLUDED_FROM_CHECKS],
      reporter: ["text", "lcovonly"],
      thresholds: { statements: 100, branches: 100, functions: 100, lines: 100 },
    },
  },
});
