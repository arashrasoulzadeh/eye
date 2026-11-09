import { test, expect } from "@playwright/test";

const ADMIN_KEY = "dev-admin-key-1234567890";

test("login, then create and delete a tenant", async ({ page }) => {
  await page.goto("/login");
  await page.locator('input[type="password"]').fill(ADMIN_KEY);
  await page.getByRole("button", { name: "Sign in" }).click();

  await expect(page).toHaveURL(/\/dashboard/, { timeout: 10_000 });

  await page.goto("/tenants");
  await expect(page.getByRole("heading", { name: "Tenants", exact: true })).toBeVisible();

  const tenantName = `e2e-${Date.now()}`;
  await page.getByPlaceholder("acme-corp").fill(tenantName);
  await page.getByRole("button", { name: "Create tenant" }).click();

  const row = page.getByRole("link", { name: tenantName });
  await expect(row).toBeVisible({ timeout: 10_000 });

  const tableRow = page.locator("tr", { has: row });
  await tableRow.getByRole("button", { name: "Delete" }).click();
  await page.getByRole("button", { name: "Delete tenant" }).click();

  await expect(row).not.toBeVisible({ timeout: 10_000 });
});
