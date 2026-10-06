// Screenshots of the prototype (reference/relic.html) at 400 px wide, 2x,
// for side-by-side checks against cmd/snapshot. Uses the bundled fonts
// (Google Fonts may be unreachable) and the pre-installed Chromium.
//
//   node tools/protoshot.js <repo> <out-dir> [shell|library]
//
// HEIGHT=1900 for a taller window (long forms). MODE=dark for dark mode, THEME=midnight etc. (default linen). ARCHIVE=<prototype export JSON> loads that data
// (e.g. importer/testdata/scrubbed-archive.json); otherwise one empty
// category. Add shot lists below for other screens.
const { chromium } = require('/opt/node-tools/node_modules/playwright');
const fs = require('fs');
const path = require('path');
const [,, repo, outDir, set = 'shell'] = process.argv;
fs.mkdirSync(outDir, { recursive: true });
const fonts = path.join(repo, 'assets/fonts');
const face = (fam, file, w, style) => `@font-face{font-family:'${fam}';src:url('file://${fonts}/${file}');font-weight:${w};font-style:${style}}`;
const css = [
  face('Playfair Display', 'PlayfairDisplay-Regular.ttf', 400, 'normal'),
  face('Playfair Display', 'PlayfairDisplay-Medium.ttf', 500, 'normal'),
  face('Playfair Display', 'PlayfairDisplay-SemiBold.ttf', 600, 'normal'),
  face('Playfair Display', 'PlayfairDisplay-Italic.ttf', 400, 'italic'),
  face('Playfair Display', 'PlayfairDisplay-MediumItalic.ttf', 500, 'italic'),
  face('Lora', 'Lora-Regular.ttf', 400, 'normal'),
  face('Lora', 'Lora-Medium.ttf', 500, 'normal'),
  face('Lora', 'Lora-Italic.ttf', 400, 'italic'),
  face('DM Sans', 'DMSans-Light.ttf', 300, 'normal'),
  face('DM Sans', 'DMSans-Regular.ttf', 400, 'normal'),
  face('DM Sans', 'DMSans-Medium.ttf', 500, 'normal'),
].join('');

const shots = {
  shell: [
    ['home', () => {}],
    ['library', () => nav('library')],
    ['prompt', () => openPromptModal({ title: 'New category', sub: 'Call it whatever makes sense to you — Films, Comfort Rewatches, Books for the train.', label: 'Category name', placeholder: 'e.g. Films', confirm: 'Create', onConfirm: () => {} })],
    ['confirm', () => { closeModal(); openConfirmModal({ title: 'Delete this entry?', sub: "This piece of your journey will be removed. This can't be undone.", confirm: 'Delete', onConfirm: () => {} }); }],
    ['toast', () => { closeModal(); nav('home'); showToast('Session logged ✦'); }],
    ['toast-error', () => showToast('You already have that category', true)],
  ],
  form: [
    ['form-new', () => startNewEntry()],
    ['form-series', () => { formState.section = sections[1].name; renderNew(); }],
    ['form-podcast', () => { formState.section = sections[3].name; renderNew(); }],
    ['form-edit', () => { showDetail(entries[0].id); openEditEntry(entries[0].id); }],
  ],
  detail: [
    ['detail', () => showDetail(entries[0].id)],
    ['detail-noposter', () => { const e = entries.find(x => !x.poster) || entries[0]; showDetail(e.id); }],
    ['detail-book', () => showDetail((entries.find(x => x.type === 'book') || entries[0]).id)],
    ['detail-film', () => showDetail((entries.find(x => x.type === 'film') || entries[0]).id)],
    ['log-session', () => { showDetail(entries[0].id); openLogSheet(entries[0].id, 'session'); }],
    ['log-book', () => { closeLogSheet(); const e = entries.find(x => x.type === 'book'); showDetail(e.id); openLogSheet(e.id, 'session'); }],
    ['log-rewatch', () => { closeLogSheet(); const e = entries.find(x => x.type === 'series' && x.status === 'finished'); showDetail(e.id); openLogSheet(e.id, 'rewatch'); }],
    ['log-edit', () => { closeLogSheet(); showDetail(entries[0].id); openLogSheet(entries[0].id, 'session', 0); }],
    ['reached-end', () => { closeLogSheet(); const e = entries[0]; openConfirmModal({ title: 'Looks like you reached the end!', sub: `You've caught up with every bit of ${e.title}. Mark it as finished?`, confirm: 'Mark as finished', cancel: 'Not yet', onConfirm: () => {} }); }],
  ],
  library: [
    ['lib-root', () => nav('library')],
    ['lib-new-category', () => goAddCategory()],
    ['lib-menu', () => { closeModal(); const b = document.querySelector('.cat-card .card-menu-btn'); b.click(); }],
    ['lib-rename', () => { closeCardMenus(); renameSection(sections[1].id); }],
    ['lib-delete', () => { closeModal(); deleteSection(sections[1].id); }],
    ['lib-search', () => { closeModal(); libSearchOpen = true; libSearchQuery = 'entry 1'; renderLibrary(); }],
    ['lib-category', () => { libSearchOpen = false; libSearchQuery = ''; openCategory(sections[1].id); }],
    ['lib-category-films', () => openCategory(sections[0].id)],
    ['lib-category-empty', () => openCategory(sections[3].id)],
    ['lib-new-folder', () => { openCategory(sections[1].id); goAddFolder(); }],
    ['lib-delete-folder', () => { closeModal(); deleteFolder(sections[1].id, 0); }],
    ['lib-folder', () => { closeModal(); openFolder(0); }],
    ['lib-folder-search', () => { libSearchOpen = true; libSearchQuery = 'x'; renderLibrary(); }],
    ['lib-folder-empty', () => { libSearchOpen = false; libSearchQuery = ''; openCategory(sections[1].id); openFolder(1); }],
  ],
};

(async () => {
  const browser = await chromium.launch({ executablePath: '/opt/pw-browsers/chromium-1194/chrome-linux/chrome' }).catch(() => chromium.launch());
  const page = await browser.newPage({ viewport: { width: 400, height: Number(process.env.HEIGHT || 800) }, deviceScaleFactor: 2 });
  await page.route(/fonts\.(googleapis|gstatic)\.com/, r => r.abort());
  const url = 'file://' + path.join(repo, 'reference/relic.html');
  await page.goto(url);
  const mode = process.env.MODE || 'light';
  let data = { userName: 'Saniya', theme: 'linen', sections: [{ id: 's1', name: 'Films', type: 'film', folders: [] }], entries: [] };
  if (process.env.ARCHIVE) data = JSON.parse(fs.readFileSync(process.env.ARCHIVE, 'utf8'));
  data.mode = mode;
  data.theme = process.env.THEME || 'linen'; // exports don't carry a usable custom palette
  await page.evaluate(d => localStorage.setItem('relic_v3', JSON.stringify(d)), data);
  await page.goto(url);
  await page.addStyleTag({ content: css });
  await page.evaluate(() => document.fonts.ready);
  await page.waitForTimeout(3500); // let the splash screen finish
  for (const [name, fn] of shots[set]) {
    await page.evaluate(fn);
    await page.waitForTimeout(700);
    await page.screenshot({ path: path.join(outDir, name + '.png') });
  }
  await browser.close();
})();
