import { expect, test } from "@playwright/test";

/**
 * Public-surface smoke (no backend session). The authenticated core loop
 * (login → connect → compose → schedule) runs against the Go backend +
 * simulator with seeded credentials - see FRONTEND_PLAN §10/12.7.
 */

test("unauthenticated users land on sign-in", async ({ page }) => {
  await page.goto("/");
  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();
  await expect(page.getByLabel("Email")).toBeVisible();
});

test("theme toggle flips the dark class and persists across reload", async ({ page }) => {
  await page.goto("/login");
  const toggle = page.getByRole("button", { name: "Toggle color theme" });
  await expect(toggle).toBeVisible();
  const before = await page.evaluate(() => document.documentElement.classList.contains("dark"));
  await toggle.click();
  await expect
    .poll(() => page.evaluate(() => document.documentElement.classList.contains("dark")))
    .not.toBe(before);
  await page.reload();
  await expect
    .poll(() => page.evaluate(() => document.documentElement.classList.contains("dark")))
    .not.toBe(before);
});

test("auth pages cross-link (sign in ↔ sign up ↔ reset)", async ({ page }) => {
  await page.goto("/login");
  await page.getByRole("link", { name: "Create an account" }).click();
  await expect(page).toHaveURL(/\/signup$/);
  await page.goBack();
  await page.getByRole("link", { name: "Forgot your password?" }).click();
  await expect(page).toHaveURL(/\/reset$/);
});
