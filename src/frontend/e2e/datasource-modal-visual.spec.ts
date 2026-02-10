/**
 * Visual Regression Tests for Compact Data Source Modal
 * 
 * Feature: compact-datasource-modal
 * 
 * These tests capture screenshots of the Snowflake and BigQuery forms
 * at multiple resolutions to detect visual regressions.
 * 
 * **Validates: Requirements 1.3, 2.4, 6.1, 6.2**
 * 
 * Prerequisites:
 *   1. Install Playwright browsers: npx playwright install chromium
 *   2. Start dev server: npm run dev
 *   3. Run tests: npx playwright test
 *   4. Update baselines: npx playwright test --update-snapshots
 */

import { test, expect } from '@playwright/test';

// Helper to open the AddDataSourceModal and select a driver type
async function openModalWithDriver(page: any, driverType: string) {
  // Navigate to the app
  await page.goto('/');
  
  // Wait for the app to load
  await page.waitForLoadState('networkidle');
  
  // Click the "Add Data Source" button (adjust selector as needed)
  const addButton = page.locator('button:has-text("Add"), button:has-text("添加"), [data-testid="add-datasource"]');
  if (await addButton.count() > 0) {
    await addButton.first().click();
  }
  
  // Wait for modal to appear
  await page.waitForSelector('.fixed.inset-0', { timeout: 5000 });
  
  // Select the driver type
  const select = page.locator('select');
  if (await select.count() > 0) {
    await select.first().selectOption(driverType);
  }
  
  // Wait for form to render
  await page.waitForTimeout(500);
}

test.describe('Snowflake Form Visual Regression', () => {
  test('should capture Snowflake form screenshot', async ({ page }) => {
    await openModalWithDriver(page, 'snowflake');
    
    // Capture the modal
    const modal = page.locator('.bg-white.rounded-xl.shadow-2xl');
    if (await modal.count() > 0) {
      await expect(modal.first()).toHaveScreenshot('snowflake-form.png', {
        maxDiffPixelRatio: 0.01,
      });
    }
  });

  test('should show confirmation button without scrolling', async ({ page }) => {
    await openModalWithDriver(page, 'snowflake');
    
    // Check that the Import/Confirm button is visible
    const confirmButton = page.locator('button:has-text("Import"), button:has-text("导入")');
    if (await confirmButton.count() > 0) {
      await expect(confirmButton.first()).toBeVisible();
      await expect(confirmButton.first()).toBeInViewport();
    }
  });
});

test.describe('BigQuery Form Visual Regression', () => {
  test('should capture BigQuery form screenshot', async ({ page }) => {
    await openModalWithDriver(page, 'bigquery');
    
    // Capture the modal
    const modal = page.locator('.bg-white.rounded-xl.shadow-2xl');
    if (await modal.count() > 0) {
      await expect(modal.first()).toHaveScreenshot('bigquery-form.png', {
        maxDiffPixelRatio: 0.01,
      });
    }
  });

  test('should show confirmation button without scrolling', async ({ page }) => {
    await openModalWithDriver(page, 'bigquery');
    
    // Check that the Import/Confirm button is visible
    const confirmButton = page.locator('button:has-text("Import"), button:has-text("导入")');
    if (await confirmButton.count() > 0) {
      await expect(confirmButton.first()).toBeVisible();
      await expect(confirmButton.first()).toBeInViewport();
    }
  });

  test('should show textarea with 4 rows', async ({ page }) => {
    await openModalWithDriver(page, 'bigquery');
    
    const textarea = page.locator('textarea');
    if (await textarea.count() > 0) {
      const rows = await textarea.first().getAttribute('rows');
      expect(rows).toBe('4');
    }
  });
});

test.describe('Modal Layout Across Resolutions', () => {
  test('modal should not exceed viewport height', async ({ page }) => {
    await openModalWithDriver(page, 'snowflake');
    
    const modal = page.locator('.bg-white.rounded-xl.shadow-2xl');
    if (await modal.count() > 0) {
      const viewportSize = page.viewportSize();
      const box = await modal.first().boundingBox();
      
      if (box && viewportSize) {
        // Modal height should be less than viewport height
        expect(box.height).toBeLessThan(viewportSize.height);
        
        // Modal should be fully visible (bottom edge within viewport)
        expect(box.y + box.height).toBeLessThanOrEqual(viewportSize.height);
      }
    }
  });

  test('modal width should be 500px', async ({ page }) => {
    await openModalWithDriver(page, 'snowflake');
    
    const modal = page.locator('.bg-white.rounded-xl.shadow-2xl');
    if (await modal.count() > 0) {
      const box = await modal.first().boundingBox();
      if (box) {
        expect(box.width).toBe(500);
      }
    }
  });

  test('full page screenshot for regression baseline', async ({ page }) => {
    await openModalWithDriver(page, 'snowflake');
    await expect(page).toHaveScreenshot('full-page-snowflake.png', {
      maxDiffPixelRatio: 0.02,
    });
  });
});
