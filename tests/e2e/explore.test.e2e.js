// @ts-check
// document is a global in evaluate, so it's safe to ignore here
/* eslint no-undef: 0 */
import {test, expect} from '@playwright/test';

test('Explore view taborder', async ({page}) => {
  await page.goto('/explore/repos');

  const l1 = page.locator('[href="https://forgejo.org"]');
  const l2 = page.locator('[href="/assets/licenses.txt"]');
  const l3 = page.locator('[href*="/stars"]').first();
  const l4 = page.locator('[href*="/forks"]').first();
  let res = 0;
  const exp = 15; // 0b1111 = four passing tests

  for (let i = 0; i < 150; i++) {
    await page.keyboard.press('Tab');
    if (await l1.evaluate((node) => document.activeElement === node)) {
      res |= 1;
      continue;
    }
    if (await l2.evaluate((node) => document.activeElement === node)) {
      res |= 1 << 1;
      continue;
    }
    if (await l3.evaluate((node) => document.activeElement === node)) {
      res |= 1 << 2;
      continue;
    }
    if (await l4.evaluate((node) => document.activeElement === node)) {
      res |= 1 << 3;
      continue;
    }
    if (res === exp) {
      break;
    }
  }
  await expect(res).toBe(exp);
});
