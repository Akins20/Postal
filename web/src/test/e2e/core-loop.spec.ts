import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

/**
 * Authenticated core loop in a real browser, against the full local stack
 * (docker deps + `postal sim` + `postal serve` + `postal worker` + web dev).
 * Covers what jsdom can't: RSC serialization, the dock/shell chrome, the real
 * OAuth redirect round trip, and dialog flows.
 */

const PW = "e2e-browser-password";

async function signupAndLogin(page: Page, request: APIRequestContext, email: string) {
  const res = await request.post("/api/v1/auth/signup", { data: { email, password: PW } });
  expect(res.status()).toBe(201);
  await page.goto("/login");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(PW);
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page).toHaveURL(/\/$/, { timeout: 20_000 });
}

test("core loop: login → dashboard → connect X → compose → schedule → calendar", async ({
  page,
  request,
}) => {
  test.setTimeout(120_000);
  const email = `pw-${Date.now()}-${Math.floor(Math.random() * 1e5)}@example.com`;
  await signupAndLogin(page, request, email);

  // Dashboard chrome renders: dock + widgets (catches RSC serialization bugs).
  const dock = page.getByRole("navigation", { name: "Primary" });
  await expect(dock).toBeVisible();
  await expect(page.getByText("No accounts connected yet.")).toBeVisible();

  // Into Channels via the dock.
  await dock.getByRole("link", { name: "Channels" }).click();
  await expect(page.getByRole("heading", { name: "Channels" })).toBeVisible();

  // Connect X: real redirect to the simulator's consent page and back.
  await page.getByRole("button", { name: "Connect", exact: true }).click();
  await expect(page).toHaveURL(/\/channels$/, { timeout: 20_000 });
  await expect(page.getByText("@simuser").first()).toBeVisible();
  await expect(page.getByText("Active")).toBeVisible();

  // Compose a draft to the connected channel; server validates it.
  await page.goto("/compose");
  // The checkbox is sr-only inside the chip label - click the chip like a user.
  await page.locator("label", { hasText: "@simuser" }).click();
  await expect(page.getByRole("checkbox", { name: "@simuser" })).toBeChecked();
  await page.getByLabel("Post text").fill(`Browser e2e ${Date.now()}`);
  await page.getByRole("button", { name: "Save draft" }).click();
  await expect(page.getByText(/draft saved/i)).toBeVisible({ timeout: 15_000 });
  await expect(page.getByText("Ready")).toBeVisible();

  // X is pay-per-use: an empty wallet must block scheduling with a way out.
  await page.getByRole("button", { name: "Schedule" }).first().click();
  const gateDialog = page.getByRole("dialog", { name: "Schedule post" });
  await gateDialog.getByRole("button", { name: "Schedule", exact: true }).click();
  await expect(gateDialog.getByText(/not enough wallet credits/i)).toBeVisible({
    timeout: 15_000,
  });
  await gateDialog.getByRole("link", { name: "Open Wallet" }).click();
  await expect(page).toHaveURL(/\/wallet/);

  // Top up with the development provider (instant credits) and come back.
  await expect(page.getByText(/credits per X post/i)).toBeVisible({ timeout: 15_000 });
  await page.getByRole("button", { name: /dev top-up/i }).click();
  await expect(page.getByText(/payment received/i)).toBeVisible({ timeout: 20_000 });

  // Schedule it for a specific future time via the dialog.
  await page.goto("/compose");
  await page.getByRole("button", { name: "Schedule" }).first().click();
  const dialog = page.getByRole("dialog", { name: "Schedule post" });
  await expect(dialog).toBeVisible();
  await dialog.getByRole("radio", { name: /specific time/i }).check();
  const runAt = new Date(Date.now() + 24 * 60 * 60 * 1000);
  const pad = (n: number) => String(n).padStart(2, "0");
  await dialog
    .getByLabel("Publish at")
    .fill(
      `${runAt.getFullYear()}-${pad(runAt.getMonth() + 1)}-${pad(runAt.getDate())}T${pad(runAt.getHours())}:${pad(runAt.getMinutes())}`,
    );
  await dialog.getByRole("button", { name: "Schedule", exact: true }).click();
  await expect(dialog.getByText(/1 job created/i)).toBeVisible({ timeout: 15_000 });
  await dialog.getByRole("button", { name: "Done" }).click();

  // The job shows up on the calendar.
  await page.goto("/calendar");
  await expect(page.getByRole("heading", { name: "Calendar" })).toBeVisible();
  await expect(page.getByText("@simuser").first()).toBeVisible({ timeout: 15_000 });
});

test("feature routes render their shells", async ({ page, request }) => {
  test.setTimeout(60_000);
  const email = `pw-${Date.now()}-${Math.floor(Math.random() * 1e5)}@example.com`;
  await signupAndLogin(page, request, email);

  for (const [path, heading] of [
    ["/media", "Media"],
    ["/analytics", "Analytics"],
    ["/settings", "Settings"],
  ] as const) {
    await page.goto(path);
    await expect(page.getByRole("heading", { name: heading, exact: true })).toBeVisible({
      timeout: 15_000,
    });
  }
});
