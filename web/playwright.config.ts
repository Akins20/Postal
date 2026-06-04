import { defineConfig, devices } from "@playwright/test";

/**
 * E2E against the running app (FRONTEND_PLAN §9.2/§12). Foundation smoke tests
 * don't hit the backend; feature sub-phases run against the real Go API + the X
 * simulator. Requires browsers: `npx playwright install chromium`.
 */
export default defineConfig({
  testDir: "./src/test/e2e",
  timeout: 30_000,
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  reporter: process.env.CI ? "github" : "list",
  use: {
    baseURL: "http://localhost:3100",
    trace: "retain-on-failure",
  },
  webServer: {
    command: "npm run dev -- --port 3100",
    url: "http://localhost:3100",
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
  projects: [
    { name: "desktop", use: { ...devices["Desktop Chrome"] } },
    { name: "mobile", use: { ...devices["Pixel 7"] } },
  ],
});
