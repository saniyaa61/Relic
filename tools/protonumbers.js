// Every number the prototype shows, computed by the prototype's own code
// (reference/relic.html in Chromium), as JSON, for comparing with the app
// (cmd/numbers). Runs in UTC, the prototype's own bucketing for streaks.
//
//   ARCHIVE=importer/testdata/scrubbed-archive.json NOW=2026-09-29T09:00:00Z \
//     node tools/protonumbers.js $PWD > build/proto-numbers.json
const { chromium } = require('/opt/node-tools/node_modules/playwright');
const fs = require('fs');
const path = require('path');
const repo = process.argv[2];

(async () => {
  const browser = await chromium.launch({ executablePath: '/opt/pw-browsers/chromium-1194/chrome-linux/chrome' }).catch(() => chromium.launch());
  const ctx = await browser.newContext({ timezoneId: 'UTC' });
  const page = await ctx.newPage();
  await page.route(/fonts\.(googleapis|gstatic)\.com/, r => r.abort());
  const url = 'file://' + path.join(repo, 'reference/relic.html');
  await page.clock.setFixedTime(new Date(process.env.NOW));
  await page.goto(url);
  const data = JSON.parse(fs.readFileSync(process.env.ARCHIVE, 'utf8'));
  await page.evaluate(d => localStorage.setItem('relic_v3', JSON.stringify(d)), data);
  await page.goto(url);
  const out = await page.evaluate(() => {
    const r = x => Math.round(x * 1e6) / 1e6;
    const res = {};
    const minsMap = buildEntryMinsMap();
    res.entries = {};
    for (const e of entries) {
      const c = JSON.parse(JSON.stringify(e));
      recomputeProgress(c);
      res.entries[e.id] = { minutes: r(minsMap.get(e.id)), watched: c.watchedEpisodes ?? null, pages: c.pagesRead ?? null };
    }
    const total = [...minsMap.values()].reduce((a, v) => a + v, 0);
    const now = new Date();
    res.home = {
      total: entries.length,
      finished: entries.filter(e => e.status === 'finished').length,
      ongoing: entries.filter(e => e.status === 'ongoing' || e.status === 'watching' || e.status === 'reading').length,
      minutes: r(total),
      consumed: formatDuration(total) || '0 min',
      streak: computeStreak(),
    };
    const monthMins = entries.reduce((a, e) => a + computeMinutesInRange(e, new Date(now.getFullYear(), now.getMonth(), 1), new Date(now.getFullYear(), now.getMonth() + 1, 1)), 0);
    res.consumed = {
      thisMonth: r(monthMins),
      perspective: crPerspective(total),
      weekly: crWeeklyMinutes().map(r),
      categories: buildConsumedCatBlocks(minsMap).map(b => ({ name: b.sec.name, minutes: r(b.catMins), folders: b.cards.map(c => ({ name: c.name === UNCAT ? '' : c.name, minutes: r(c.mins) })) })),
    };
    const period = (start, end) => {
      const nw = entries.filter(e => { const d = new Date(e.createdAt || e.watchedDate || e.startDate || 0); return d >= start && d < end; });
      const fin = entries.filter(e => e.finishedAt && new Date(e.finishedAt) >= start && new Date(e.finishedAt) < end);
      const sess = sessionsInRange(start, end);
      const secCount = {};
      sess.forEach(({ e }) => { secCount[e.section] = (secCount[e.section] || 0) + 1; });
      nw.forEach(e => { secCount[e.section] = (secCount[e.section] || 0) + 1; });
      const topSec = Object.entries(secCount).sort((a, b) => b[1] - a[1])[0];
      const tagCount = {};
      nw.forEach(e => (e.tags || []).forEach(t => { tagCount[t] = (tagCount[t] || 0) + 1; }));
      const topTag = Object.entries(tagCount).sort((a, b) => b[1] - a[1])[0];
      const ids = new Set([...nw.map(e => e.id), ...sess.map(x => x.e.id)]);
      const rated = [...entries].filter(e => ids.has(e.id) && e.rating > 0).sort((a, b) => b.rating - a.rating);
      return {
        new: nw.length, finished: fin.length,
        minutes: r(entries.reduce((a, e) => a + computeMinutesInRange(e, start, end), 0)),
        pages: pagesReadInRange(start, end), sessions: sess.length,
        topCategory: topSec ? topSec[0] : '', topTag: topTag ? topTag[0] : '',
        topRated: rated.slice(0, 3).map(e => e.id),
      };
    };
    res.week = period(new Date(Date.now() - 7 * 864e5), now);
    const floor = getDigestFloorDate();
    res.months = {};
    for (let d = new Date(floor.getFullYear(), floor.getMonth(), 1); d <= now; d = new Date(d.getFullYear(), d.getMonth() + 1, 1)) {
      const key = d.getFullYear() + '-' + String(d.getMonth() + 1).padStart(2, '0');
      res.months[key] = period(d, new Date(d.getFullYear(), d.getMonth() + 1, 1));
    }
    res.years = {};
    for (let y = floor.getFullYear(); y <= now.getFullYear(); y++) {
      const yc = computeYearCardData(y);
      res.years[y] = Object.assign(period(new Date(y, 0, 1), new Date(y + 1, 0, 1)), {
        monthly: yearMonthlyMinutes(y).map(r),
        card: { minutes: r(yc.totalMins), categories: yc.cats.map(c => ({ name: c.name, minutes: r(c.mins) })), started: yc.started.length,
          finished: yc.finished.length, topRated: yc.topRated.map(e => e.id), topFeeling: yc.topFeeling || '' },
      });
    }
    const md = computeMoodData();
    res.moods = {
      months: md.months.map(m => m.start.getFullYear() + '-' + String(m.start.getMonth() + 1).padStart(2, '0')),
      tags: md.tags.map(t => ({ tag: t.tag, counts: t.counts, total: t.total })),
      insights: moodInsights(md).map(s => s.replace(/<\/?em>/g, '')),
    };
    return res;
  });
  console.log(JSON.stringify(out, null, 1));
  await browser.close();
})();
