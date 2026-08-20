import { Locator, Page, expect } from '@playwright/test';

/**
 * The inventory screen. Knows two things the specs should not have to:
 *
 *  - the stock ledger is a modal on /inventory, not a section of a product
 *    page — InventoryLedgerModal, opened per row;
 *  - the row actions are icon buttons identified by their title attribute
 *    ("Add stock", "Remove stock", "Stock history"), because they have no text.
 *
 * It also absorbs a Headless UI detail. Modal renders
 * <Dialog as="div" className="relative z-50"> — that element carries
 * role="dialog", but every child is `fixed inset-0`, so the wrapper collapses to
 * a zero-size box and Playwright reports it hidden even while it is open. It is
 * fine to scope queries to it; it is never right to assert it visible. Waits
 * here target the title or a field instead, both of which have real layout.
 */
export class InventoryPage {
  constructor(private readonly page: Page) {}

  /** Search narrows the table to one product; the box is debounced 300ms. */
  async findProduct(nameOrSku: string): Promise<Locator> {
    await this.page.goto('/inventory');
    await this.page.getByPlaceholder('Search products by name or SKU...').fill(nameOrSku);

    const row = this.page.locator('tr', { hasText: nameOrSku }).first();
    await expect(row, `no inventory row for ${nameOrSku}`).toBeVisible({ timeout: 20_000 });
    return row;
  }

  /** Opens the stock-history modal for a product and returns it. */
  async openLedger(nameOrSku: string): Promise<Locator> {
    const row = await this.findProduct(nameOrSku);
    await row.locator('[title="Stock history"]').click();

    const modal = this.page.getByRole('dialog');
    await expect(
      modal.locator('[id^="headlessui-dialog-title"]'),
      'stock history modal did not open'
    ).toContainText('Stock history', { timeout: 20_000 });
    return modal;
  }

  async closeModal(): Promise<void> {
    await this.page.keyboard.press('Escape');
    // Transition unmounts the dialog, so absence is the signal — not hidden,
    // which the wrapper reports even when open.
    await expect(this.page.getByRole('dialog')).toHaveCount(0, { timeout: 10_000 });
  }

  /** Opens Add or Remove stock for a product and fills it in. */
  async adjustStock(
    nameOrSku: string,
    action: 'Add stock' | 'Remove stock',
    quantity: number,
    reason: string
  ): Promise<void> {
    const row = await this.findProduct(nameOrSku);
    await row.locator(`[title="${action}"]`).click();

    const modal = this.page.getByRole('dialog');
    const quantityField = modal.getByPlaceholder('Enter quantity');
    await expect(quantityField, `${action} modal did not open`).toBeVisible({ timeout: 20_000 });

    await quantityField.fill(String(quantity));
    await modal.locator('textarea').first().fill(reason);
    await modal.locator('button[type="submit"]').click();

    await expect(modal, `${action} modal did not close`).toHaveCount(0, { timeout: 30_000 });
  }
}
