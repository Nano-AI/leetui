#!/usr/bin/env node
// Browser checks for the static site. Requires playwright and @axe-core/playwright.
// CHROME=/path/to/chrome NODE_PATH=/path/to/node_modules node site/tools/check_site.cjs
const assert = require('node:assert/strict');
const http = require('node:http');
const fs = require('node:fs/promises');
const path = require('node:path');
const { chromium } = require('playwright');
const AxeBuilder = require('@axe-core/playwright').default;
const root = path.resolve(__dirname, '..');

const server = http.createServer(async (req, res) => {
  try {
    const pathname = new URL(req.url, 'http://localhost').pathname;
    const file = path.resolve(root, '.' + (pathname === '/' ? '/index.html' : pathname));
    if (!file.startsWith(root + path.sep)) { res.writeHead(403).end(); return; }
    const types = { '.html': 'text/html', '.svg': 'image/svg+xml', '.json': 'application/json' };
    res.setHeader('Content-Type', types[path.extname(file)] || 'application/octet-stream');
    res.end(await fs.readFile(file));
  } catch { res.writeHead(404).end(); }
});

(async () => {
  let browser;
  try {
    await new Promise((resolve, reject) => {
      server.once('error', reject);
      server.listen(0, '127.0.0.1', resolve);
    });
    const url = `http://127.0.0.1:${server.address().port}/`;
    browser = await chromium.launch({ executablePath: process.env.CHROME || undefined });
    for (const width of [1440, 768, 390, 320]) {
      const context = await browser.newContext({ viewport: { width, height: 900 }, reducedMotion: 'reduce' });
      const page = await context.newPage();
      const errors = [], requests = [];
      page.on('pageerror', e => errors.push(e.message));
      page.on('request', r => requests.push(r.url()));
      await page.goto(url);
      assert(!requests.some(r => r.endsWith('demo.json')), 'Recording must load only on request');
      assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'Page overflows');
      await page.keyboard.press('Tab');
      assert.equal(await page.locator(':focus').textContent(), 'Skip to content');
      await page.keyboard.press('Enter');
      assert.equal(await page.locator(':focus').getAttribute('id'), 'main');
      const tab = page.locator('#tab-script');
      await tab.focus();
      assert.notEqual(await tab.evaluate(el => getComputedStyle(el).outlineStyle), 'none');
      await page.keyboard.press(width <= 640 ? 'ArrowRight' : 'ArrowDown');
      assert.equal(await page.locator(':focus').getAttribute('id'), 'tab-brew');
      assert(await page.locator('#panel-brew').isVisible());
      await page.keyboard.press('End');
      assert(await page.locator('#panel-binary a').isVisible());
      await page.keyboard.press('Home');
      // Clipboard denial must select the complete command and explain how to copy it.
      await page.evaluate(() => Object.defineProperty(navigator, 'clipboard', {
        value: { writeText: () => Promise.reject(new Error('denied')) }, configurable: true
      }));
      await page.locator('#panel-script .copy').click();
      assert.equal(await page.evaluate(() => getSelection().toString()), await page.locator('#cmd-script').textContent());
      assert.match(await page.locator('#status').textContent(), /Command selected/);
      assert(await page.locator('#status').isVisible());
      await page.waitForTimeout(2300);
      assert.match(await page.locator('#status').textContent(), /Command selected/, 'Recovery instructions must remain available');
      await page.evaluate(() => Object.defineProperty(navigator, 'clipboard', {
        value: { writeText: text => { window.copied = text; return Promise.resolve(); } }, configurable: true
      }));
      await page.locator('#panel-script .copy').click();
      assert.equal(await page.evaluate(() => window.copied), await page.locator('#cmd-script').textContent());
      await page.locator('#demo-toggle').click();
      await page.waitForSelector('.screen');
      assert(await page.locator('#demo-toggle').isVisible());
      await page.locator('#demo-toggle').click();
      assert.equal(await page.locator('#demo-toggle').textContent(), 'Play session');
      const still = await page.locator('.screen').innerHTML();
      await page.waitForTimeout(300);
      assert.equal(await page.locator('.screen').innerHTML(), still, 'Paused demo repainted');
      assert(await page.locator('.term-view img').getAttribute('alt'));
      assert.match(await page.locator('.term-view').getAttribute('aria-label'), /leetui board/);
      assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'Player causes page overflow');
      for (const link of await page.locator('a[href^="#"]').all()) {
        const href = await link.getAttribute('href');
        assert.equal(await page.locator(href).count(), 1, `Missing anchor ${href}`);
      }
      for (const image of await page.locator('img').all()) {
        if (await image.isVisible()) { await image.scrollIntoViewIfNeeded(); }
        await image.evaluate(el => el.decode());
      }
      // The recording reproduces the terminal's palette, including dim text.
      // Audit website controls separately; do not silently report the recording as AA.
      const fullAudit = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21aa']).analyze();
      console.log(`Recording-inclusive axe findings at ${width}px: ${fullAudit.violations.map(v => `${v.id} (${v.nodes.length} nodes)`).join(', ') || 'none'}`);
      const audit = await new AxeBuilder({ page }).exclude('.screen').withTags(['wcag2a', 'wcag2aa', 'wcag21aa']).analyze();
      assert.deepEqual(audit.violations.map(v => ({ id: v.id, nodes: v.nodes.map(n => n.target) })), []);
      assert.deepEqual(errors, []);
      if (process.env.SCREENSHOTS) {
        await page.evaluate(() => { getSelection().removeAllRanges(); window.scrollTo(0, 0); });
        await page.screenshot({ path: path.join(process.env.SCREENSHOTS, `leetui-${width}.png`), fullPage: true });
      }
      console.log(`PASS ${width}×900: overflow, keyboard, tabs, copy success/denial, demo pause, assets, anchors, axe, console`);
      await context.close();
    }
    const page = await browser.newPage({ javaScriptEnabled: false, viewport: { width: 390, height: 844 } });
    await page.goto(url);
    for (const id of ['script', 'brew', 'go', 'binary']) { assert(await page.locator('#panel-' + id).isVisible()); }
    assert.equal(await page.locator('.count-n').allTextContents().then(a => a.join(',')), '4013,989,17,0');
    assert(!(await page.locator('.copy').first().isVisible()));
    assert(await page.locator('.term-view img').isVisible());
    for (const region of await page.locator('.shot, .term-view, .cmd-line').all()) {
      assert.equal(await region.getAttribute('tabindex'), '0', 'Scrollable content must be keyboard reachable without JavaScript');
    }
    console.log('PASS JavaScript disabled: all install methods, counters, still image, no inert copy buttons');
    await page.close();
    // Test an actual preference transition, independent of the host OS setting.
    // Explicit playback is allowed when reduce is already enabled; setting the
    // same value again does not dispatch a MediaQueryList change event.
    const interactive = await browser.newPage({ viewport: { width: 1280, height: 800 }, reducedMotion: 'no-preference' });
    await interactive.goto(url + '?install=go#install');
    assert(await interactive.locator('#panel-go').isVisible(), 'Install deep link must restore method');
    await interactive.locator('#tab-brew').click();
    assert.equal(new URL(interactive.url()).searchParams.get('install'), 'brew');
    await interactive.goBack();
    assert(await interactive.locator('#panel-go').isVisible());
    await interactive.goForward();
    assert(await interactive.locator('#panel-brew').isVisible());
    await interactive.reload();
    assert(await interactive.locator('#panel-brew').isVisible());
    await interactive.goto(url + '?install=unknown');
    assert(await interactive.locator('#panel-script').isVisible(), 'Unknown method must fall back');
    await interactive.goto(url);
    await interactive.locator('#main').focus();
    await interactive.keyboard.press('e');
    assert.equal(new URL(interactive.url()).hash, '', 'Shortcuts must be opt-in');
    await interactive.locator('#shortcuts').check();
    await interactive.locator('#main').focus();
    await interactive.keyboard.press('e');
    assert.equal(new URL(interactive.url()).hash, '#solve');
    assert.equal(await interactive.locator(':focus').getAttribute('id'), 'solve');
    await interactive.goBack();
    assert.equal(new URL(interactive.url()).hash, '');
    await interactive.route('**/demo.json', route => route.fulfill({ status: 503, body: '' }));
    await interactive.locator('#demo-toggle').click();
    await interactive.getByRole('button', { name: 'Retry session' }).waitFor();
    assert.match(await interactive.locator('#demo-status').textContent(), /could not load/);
    assert(await interactive.locator('.term-view img').isVisible());
    await interactive.unroute('**/demo.json');
    await interactive.route('**/demo.json', route => route.fulfill({ contentType: 'application/json', body: '{"rows":34,"cols":120,"styles":[],"frames":[]}' }));
    await interactive.locator('#demo-toggle').click();
    await interactive.getByRole('button', { name: 'Retry session' }).waitFor();
    assert(await interactive.locator('.term-view img').isVisible(), 'Malformed recording must preserve still');
    assert.equal(await interactive.locator('.screen').count(), 0);
    await interactive.unroute('**/demo.json');
    await interactive.locator('#demo-toggle').click();
    await interactive.getByRole('button', { name: 'Pause session' }).waitFor();
    assert.equal(await interactive.evaluate(() => matchMedia('(prefers-reduced-motion: reduce)').matches), false,
      'Live reduced-motion test must start with no-preference');
    await interactive.emulateMedia({ reducedMotion: 'reduce' });
    await interactive.getByRole('button', { name: 'Play session' }).waitFor();
    const paused = await interactive.locator('.screen').innerHTML();
    await interactive.waitForTimeout(400);
    assert.equal(await interactive.locator('.screen').innerHTML(), paused);
    await interactive.setViewportSize({ width: 640, height: 400 });
    assert(await interactive.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
    console.log('PASS install deep links/back/forward/reload/fallback, shortcuts opt-in/focus/history, demo network and malformed-data failure/retry, live reduced-motion change, 200%-equivalent viewport');
    await interactive.close();
  } finally {
    // Cover launch failures too, and release the server even if browser cleanup
    // fails. Await closure so the process result includes cleanup errors.
    try {
      if (browser) { await browser.close(); }
    } finally {
      if (server.listening) {
        await new Promise((resolve, reject) => {
          server.close(error => error ? reject(error) : resolve());
          server.closeAllConnections();
        });
      }
    }
  }
})().then(() => {
  console.log('PASS browser/server cleanup; all site checks completed');
  process.exitCode = 0;
}).catch(error => {
  console.error(error);
  process.exitCode = 1;
});
