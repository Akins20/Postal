import { defineConfig, devices } from "@playwright/test";

/**
 * E2E against the running app (FRONTEND_PLAN §9.2/§12). The smoke specs need
 * only the web app; the authenticated core-loop spec needs the full local
 * stack (docker deps + `postal sim`/`serve`/`worker`) — see
 * scripts/curl/web-e2e.sh for the recipe.
 *
 * Browsers: uses the system Chrome (`channel: "chrome"`) because Playwright's
 * downloaded binaries aren't published for this OS. In CI, `npx playwright
 * install chromium` runs and no channel is forced.
 */
const channel = process.env.CI ? undefined : "chrome";

export default defineConfig({
  testDir: "./src/test/e2e",
  timeout: 30_000,
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  reporter: process.env.CI ? "github" : "list",
  use: {
    baseURL: process.env.POSTAL_WEB_URL ?? "http://localhost:3000",
    trace: "retain-on-failure",
  },
  webServer: {
    command: "npm run dev",
    url: "http://localhost:3000",
    reuseExistingServer: true,
    timeout: 120_000,
  },
  projects: [
    { name: "desktop", use: { ...devices["Desktop Chrome"], channel } },
    { name: "mobile", use: { ...devices["Pixel 7"], channel } },
  ],
});
