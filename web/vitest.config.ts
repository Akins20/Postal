import { fileURLToPath } from "node:url";

import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) },
  },
  test: {
    environment: "jsdom",
    globals: true,
    // Absolute origin so the API client's requests are valid in node fetch + MSW.
    env: { NEXT_PUBLIC_API_BASE: "http://localhost" },
    setupFiles: ["./src/test/setup.ts"],
    // Vitest owns *.test.* (unit/component); Playwright owns *.spec.* (e2e).
    include: ["src/**/*.test.{ts,tsx}"],
    exclude: ["node_modules/**", ".next/**", "src/test/e2e/**"],
    css: false,
  },
});
