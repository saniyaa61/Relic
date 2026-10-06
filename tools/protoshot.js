// Screenshots of the prototype (reference/relic.html) at 400 px wide, 2x,
// for side-by-side checks against cmd/snapshot. Uses the bundled
// fonts (Google Fonts may be unreachable). Cloud sessions:
//   MODE=light node tools/protoshot.js $PWD build/proto relic_v3
// Edit the shot list at the bottom for other screens.
const { chromium } = require('/opt/node-tools/node_modules/playwright');
const path = require('path');
const [,, repo, outDir, dbKey] = process.argv;
require('fs').mkdirSync(outDir, { recursive: true });
const fonts = path.join(repo, 'assets/fonts');
const face = (fam, file, w, style) => `@font-face{font-family:'${fam}';src:url('file://${fonts}/${file}');font-weight:${w};font-style:${style}}`;
const css = [
  face('Playfair Display','PlayfairDisplay-Regular.ttf',400,'normal'),
  face('Playfair Display','PlayfairDisplay-Medium.ttf',500,'normal'),
  face('Playfair Display','PlayfairDisplay-SemiBold.ttf',600,'normal'),
  face('Playfair Display','PlayfairDisplay-Italic.ttf',400,'italic'),
  face('Playfair Display','PlayfairDisplay-MediumItalic.ttf',500,'italic'),
  face('Lora','Lora-Regular.ttf',400,'normal'),
  face('Lora','Lora-Medium.ttf',500,'normal'),
  face('Lora','Lora-Italic.ttf',400,'italic'),
  face('DM Sans','DMSans-Light.ttf',300,'normal'),
  face('DM Sans','DMSans-Regular.ttf',400,'normal'),
  face('DM Sans','DMSans-Medium.ttf',500,'normal'),
].join('');
(async () => {
  const browser = await chromium.launch({ executablePath: '/opt/pw-browsers/chromium-1194/chrome-linux/chrome' }).catch(() => chromium.launch());
  const page = await browser.newPage({ viewport: { width: 400, height: 800 }, deviceScaleFactor: 2 });
  await page.route(/fonts\.(googleapis|gstatic)\.com/, r => r.abort());
  const url = 'file://' + path.join(repo, 'reference/relic.html');
  await page.goto(url);
  const mode = process.env.MODE || 'light';
  await page.evaluate(([k, mode]) => localStorage.setItem(k, JSON.stringify({ userName: 'Saniya', theme: 'linen', mode, sections: [{ id: 's1', name: 'Films', type: 'film', folders: [] }], entries: [] })), [dbKey, mode]);
  await page.goto(url);
  await page.addStyleTag({ content: css });
  await page.evaluate(() => document.fonts.ready);
  await page.waitForTimeout(500);
  const shot = async (name) => { await page.waitForTimeout(600); await page.screenshot({ path: path.join(outDir, name + '.png') }); };
  await shot('home');
  await page.evaluate(() => nav('library')); await shot('library');
  await page.evaluate(() => openPromptModal({ title: 'New category', sub: 'Call it whatever makes sense to you — Films, Comfort Rewatches, Books for the train.', label: 'Category name', placeholder: 'e.g. Films', confirm: 'Create', onConfirm: () => {} }));
  await shot('prompt');
  await page.evaluate(() => closeModal()); await page.waitForTimeout(300);
  await page.evaluate(() => openConfirmModal({ title: 'Delete this entry?', sub: "This piece of your journey will be removed. This can't be undone.", confirm: 'Delete', onConfirm: () => {} }));
  await shot('confirm');
  await page.evaluate(() => closeModal()); await page.waitForTimeout(300);
  await page.evaluate(() => { nav('home'); showToast('Session logged ✦'); }); await shot('toast');
  await page.evaluate(() => showToast('You already have that category', true)); await shot('toast-error');
  await page.evaluate(() => { nav('home'); renderStillWithYou && (document.querySelectorAll('.vw').forEach(e=>e.classList.remove('active')), document.getElementById('v-stillwithyou').classList.add('active'), document.getElementById('bnav').style.display='none', swySearchOpen=true, swySearchQuery='dune', renderStillWithYou()); });
  await shot('search');
  await browser.close();
})();
