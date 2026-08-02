import { defineConfig } from "vitest/config";
export default defineConfig({
  test: {
    root: ".",
    environment: "node",
    include: ["src/**/__tests__/**/*.test.ts"],
    exclude: ["node_modules", ".bun", ".antigravity", ".claude"],
    globals: true,
  },
});
