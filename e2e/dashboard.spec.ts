import { test, expect } from '@playwright/test';

test.describe('Admin Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/dashboard');
  });

  test('should load dashboard', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('ProxyMesh');
  });

  test('should navigate to nodes tab', async ({ page }) => {
    await page.click("button:has-text('Nodes')");
    await expect(page.locator('#nodes')).toBeVisible();
  });

  test('should navigate to health tab', async ({ page }) => {
    await page.click("button:has-text('Health')");
    await expect(page.locator('#health')).toBeVisible();
  });

  test('should navigate to logs tab', async ({ page }) => {
    await page.click("button:has-text('Logs')");
    await expect(page.locator('#logs')).toBeVisible();
  });

  test('should navigate to config tab', async ({ page }) => {
    await page.click("button:has-text('Config')");
    await expect(page.locator('#config')).toBeVisible();
  });
});
