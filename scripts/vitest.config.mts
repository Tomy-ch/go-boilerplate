import { defineConfig } from "vitest/config";

import { EXCLUDED_FROM_CHECKS } from "./lib/untested-modules.ts";

export default defineConfig({
  root: import.meta.dirname,
  test: {
    environment: "node",
    include: ["**/*.test.ts"],
    coverage: {
      provider: "v8",
      include: ["**/*.ts"],
      exclude: [...EXCLUDED_FROM_CHECKS],
      reporter: ["text", "lcovonly"],
      thresholds: { statements: 100, branches: 100, functions: 100, lines: 100 },
    },
  },
});
