import { expect, test } from "@playwright/test";

test("dashboard renders with the dock and a working theme toggle", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Welcome to Postal" })).toBeVisible();
  await expect(page.getByRole("navigation", { name: "Primary" })).toBeVisible();

  const toggle = page.getByRole("button", { name: "Toggle color theme" });
  await expect(toggle).toBeVisible();
  const before = await page.evaluate(() => document.documentElement.classList.contains("dark"));
  await toggle.click();
  await expect
    .poll(() => page.evaluate(() => document.documentElement.classList.contains("dark")))
    .not.toBe(before);
});

test("navigating into a feature route shows the sidebar shell", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("link", { name: "Compose" }).first().click();
  await expect(page).toHaveURL(/\/compose$/);
  await expect(page.getByRole("heading", { name: "Compose" })).toBeVisible();
});
