import { Page, expect } from '@playwright/test';

/**
 * The admin SPA keeps its JWT in an HttpOnly cookie set by the real auth
 * Lambda, so there is no token to inject — the suite logs in the way a person
 * does. Selectors are anchored on the placeholders in LoginPage.tsx.
 */
export async function loginToAdmin(
  page: Page,
  role: 'admin' | 'operator'
): Promise<void> {
  const prefix = role === 'admin' ? 'E2E_ADMIN' : 'E2E_OPERATOR';
  const email = process.env[`${prefix}_EMAIL`];
  const password = process.env[`${prefix}_PASSWORD`];
  if (!email || !password) {
    throw new Error(`${prefix}_EMAIL and ${prefix}_PASSWORD must be set`);
  }

  await page.goto('/login');
  await page.getByPlaceholder('admin@handloom.com').fill(email);
  await page.getByPlaceholder('Enter your password').fill(password);
  await page.getByRole('button', { name: /sign in|log ?in/i }).click();

  await expect(
    page,
    `${role} login did not leave /login — check the credentials in dev`
  ).not.toHaveURL(/\/login/);
}
