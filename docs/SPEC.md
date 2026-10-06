# Relic — Product & Behaviour Spec

**Relic: your stories, preserved.** A personal archive for everything you watch, read and listen to — films, dramas, books, podcasts, music — with the feelings you had along the way.

This document is the source of truth for rebuilding Relic as a native Go app for Android and iOS. It was written from the working prototype in `reference/relic.html`. Where this spec and the prototype disagree, **this spec wins** (some prototype behaviour was a workaround for old data and is deliberately not carried over — see §11).

Contents: 1 Glossary · 2 Data model · 3 Entry types & fields · 4 Core rules · 5 Screens · 6 Visual design · 7 Year in Review card · 8 Settings & archive · 9 Importing prototype data · 10 Out of scope for v1 · 11 Known gaps & open questions

---

## 1. Glossary

| Term | Meaning |
|---|---|
| **Category** | A user-created top-level shelf, e.g. "Dramas", "Movies", "Listening". There are **no built-in categories**. Each category has one **entry type** that decides which fields its entries have. (Called `section` in the prototype.) |
| **Folder** | A user-created sub-shelf inside a category, e.g. "Bollywood". Optional. |
| **Uncategorised** | Entries in a category that are not in any folder. Shown as its own group. |
| **Entry** | One thing the user is watching/reading/listening to. Belongs to exactly one category, optionally one folder. |
| **Session** | One sitting logged against an entry ("watched 2 episodes", "read pages 40–80"). |
| **Start session** | The session created automatically when the entry is first logged. It records how far the user already was. (`first:true` in the prototype.) |
| **Rewatch** | A later re-experience of a finished entry (reread / relisten for books / audio). |
| **Journey** | The timeline of an entry's sessions and rewatches. |

---

## 2. Data model

Clean model for the Go app. All timestamps are stored as **UTC instants**; all "which day / month / year does this belong to" questions are answered in the **user's local timezone** (see §4.11).

```
Profile
  name            string    (optional)
  firstUsedAt     time      set once, on first launch
  theme           string    linen | midnight | blush | forest | rose | slate | custom
  mode            string    light | dark (remembered across launches and imports)
  customBase      color     (only when theme = custom)
  customAccent    color     (only when theme = custom)

Category
  id, name        (unique per user, case-insensitive)
  type            film | series | book | podcast | short | music | other
  folders         ordered list of folder names (unique within category, case-insensitive)
  createdAt

Entry
  id
  categoryId
  folder          string or empty (empty = Uncategorised)
  type            copied from category at creation; changes if the entry moves category
  title           required
  poster          image reference (stored as a file, NOT inline — see §11)
  rating          0–5 in 0.5 steps; 0 = unrated
  tags            ordered list of strings ("How it felt")
  review          the words written at creation ("Your words" source)
  status          ongoing | finished
  createdAt       when first logged
  finishedAt      set when status becomes finished; cleared if it goes back to ongoing
  favorite        bool
  fields          type-specific values (see §3)
  sessions        []Session
  rewatches       []Rewatch

Session
  id, at (time), note
  isStart         bool — exactly one per entry at most; cannot be deleted
  startedFinished bool — only meaningful on the start session: entry was logged as already finished
  episodes        int    (series, podcast)
  fromPage,toPage int    (book)
  minutes         int    (book, other; optional for book)

Rewatch
  id, at, note
  rating          optional 0–5 (stored on the rewatch only; never changes the entry rating — kept as separate history)
  full            bool, default true (series/podcast: "Yes, all of it" vs "Partial rewatch")
  episodes        int  (partial rewatch only: "Episodes rewatched", asked when "Partial rewatch" is picked)
  startDate, endDate  optional (books)

Favourites
  topFive[categoryId]     ordered list of entry ids, max 5
  favouriteOrder[categoryId]  user's custom order of the remaining favourites
```

Derived, never stored independently in the new app: `watchedEpisodes` (= sum of session episodes, capped at total), `pagesRead` (= max `toPage` across sessions). The prototype stored these and they drifted out of sync; derive them.

---

## 3. Entry types & fields

| Type | Label | Fields | Progress tracked? |
|---|---|---|---|
| `film` | Film | director, language, cast, duration (mins), watchedDate | no |
| `series` | Series / Drama | totalEpisodes, watchedEpisodes (initial), episodeDuration (mins), platform, cast | episodes |
| `book` | Book | author, startDate, totalPages, pagesRead (initial), publisher/year | pages |
| `podcast` | Podcast | host, platform, totalEpisodes, watchedEpisodes (initial), episodeDuration ("Ep. length (mins)", as series), language, watchedDate | episodes |
| `short` | Shorts / YouTube | creator, platform, duration (mins), watchedDate, url | no |
| `music` | Music / Album | artist, genre, tracks, length (mins), watchedDate, platform | no |
| `other` | Other | creator, platform, duration (free text), watchedDate | no |

Preset feeling tags: Emotional, Haunting, Rewatch-worthy, Quiet, Profound, Funny, Disturbing, Comforting, Life-changing, Beautiful, Slow-burn, Bittersweet, Gothic, Romantic, Devastating. Users can also add custom tags.

Statuses offered in the UI: **Ongoing** and **Finished** only.

---

## 4. Core rules

These belong in the Go **core** package, fully unit-tested, with no UI or storage dependencies.

### 4.1 Creating an entry
- Title required; category required. New entries go to the top (newest first).
- If created as **finished**: `finishedAt = createdAt`.
- A **start session** is created when the entry has a review *or* its type tracks progress (series, podcast, book). It holds:
  - `at = createdAt`, `note = review`, `isStart = true`, `startedFinished = (status == finished)`
  - series/podcast: `episodes = initial watchedEpisodes`
  - book: `fromPage = 0`, `toPage = initial pagesRead`

### 4.2 Status & finished date
- One function owns status changes. Transition to finished → set `finishedAt = now` (or the given date). Transition away from finished → clear `finishedAt`. Same status → no change.
- Applies everywhere: the "Mark as finished" prompt, the Still With You badge, and the edit form.

### 4.3 Sessions
- Log a session: series/podcast ask *episodes this session* (default 1); books ask *from page / to page / minutes*; other types ask *minutes*. All ask *your thoughts*.
- **Zero is a valid value** (e.g. "from page 0"). Never treat 0 as "empty".
- Every session (including the start session) can be **edited**. All except the start session can be **deleted**. Deleting/editing recomputes progress.
- Editing the start session's note also updates the entry's `review`, and vice-versa (edit form).

### 4.4 Progress & the finished prompt
- `watchedEpisodes = min(sum(session.episodes), totalEpisodes)` when total is set.
- `pagesRead = max(session.toPage)`.
- **Reached the end** = progress ≥ total (types with progress only).
- After saving a session that reaches the end on an *ongoing* entry → show "Looks like you reached the end! Mark as finished?" (Not yet / Mark as finished).
- On Still With You cards, an entry that has reached the end but is still ongoing shows a **"✓ Finished?"** badge that opens the same confirm.
- Never show the prompt before the end.

### 4.5 Time spent
Every entry produces a list of **time events** `(at, minutes)`. All totals — all-time, per category, per folder, per period, charts — are sums over these events.

| Type | Events |
|---|---|
| series, podcast | each session: `episodes × episodeDuration`; each rewatch with `full = true`: `totalEpisodes × episodeDuration`; each partial rewatch: `episodes rewatched × episodeDuration` |
| book | each session: `minutes` if logged, otherwise `(toPage − fromPage) × 1.5` |
| other | each session: `minutes` (if any) |
| film, short | one event at `createdAt` of `duration`; plus `duration` per rewatch |
| music | one event at `createdAt` of the album length; plus the length per relisten. If length is blank: `tracks × 3.5` min |

Compute each entry's events **once per render** and reuse them (the prototype was slow when it re-derived them per chart bucket).

### 4.6 Formatting durations
- `< 60 min` → "45 min"
- `≤ 72 hours` → hours, one decimal unless whole: "5.4 hrs", "1 hr", "72 hrs"
- `> 72 hours` → days, one decimal: "3.1 days"
- Zero → nothing shown (or "0 min" where a value is required).

"In perspective" line (Consumed hero, Year card), joined with ", ":
- if 24 h ≤ total ≤ 72 h: "about 2.8 days nonstop"
- if `round(total / 120) ≥ 1`: "like 33 films back to back"

### 4.7 "Your words" (entry detail)
Shows the **single most powerful sentence** the user has written about the entry, across the review and every session note.
- Split all text into sentences (on `.`, `!`, `?`).
- Score = length in characters; +20 if it contains `!`; +15 per ALL-CAPS word of 3+ letters.
- Highest score wins; it is re-evaluated whenever notes change. Label it with its date ("Started · 16 June 2026" if from the start session).

### 4.8 Journey labels
- Start session label, unless `startedFinished`: verb + progress, e.g. "Started watching · Ep 3", "Started reading · p. 42".
- Verb by type: book → "Started reading"; podcast, music → "Started listening"; film, series, short → "Started watching".
- For `other`, infer from category + folder name (case-insensitive): listen/audio/music/podcast/song/radio → listening; read/book/novel/article/write/comic/manga → reading; watch/movie/film/show/series/video/drama/anime → watching; play/game → "Started playing"; else just "Started".
- Later sessions: "Watched 2 episodes" / "Listened to 2 episodes", "Pages 40–80 · 35 min", or "35 min".
- Timeline is newest first; rewatches carry a "Rewatch / Reread / Relisten" pill.

### 4.9 Search
- An entry matches if the query (case-insensitive substring) appears in: title, review, any tag, category name, folder name, or any type-specific field.
- Search is **scoped to the page**: Library root = everything; inside a category = that category; inside a folder = that folder; Still With You = ongoing only; Recently Finished = finished only; All Entries / Consumed = everything; Consumed folder = that folder.
- Result header: "3 results for "dune" in Movies" (or "across your library").
- **Back button while search is open closes the search first**, staying on the page. A second back navigates away.
- One shared search implementation; pages only supply the entry set, scope label, and result layout.

### 4.10 Streak
- Active days = local dates with any entry created, session, or rewatch.
- Current streak counts consecutive days back from the most recent active day, but only if that day is today or yesterday. Also track the longest streak.

### 4.11 Dates & timezones
- Store UTC; bucket by **local** date (day / week / month / year). The prototype bucketed by UTC and mis-attributed late-night logs — fix this.

### 4.12 Digest periods
- **Week** = the last 7 days (rolling), no navigation.
- **Month / Year** = calendar periods with ← → navigation.
- Navigation floor = the month/year of `min(firstUsedAt, earliest entry createdAt)`. Ceiling = current month/year. Arrows disable at both ends; the step functions also refuse to go past them.
- Period membership: "new" = `createdAt` in period; "finished" = `finishedAt` in period; minutes = time events in period; pages = `sum(toPage − fromPage)` for book sessions in period; sessions = sessions + rewatches in period.

### 4.13 Mood trends
- A feeling counts in the month of the entry's `createdAt` (when it was tagged).
- Window: the last 12 months ending this month, but never before the Digest floor month.
- Show the top 8 tags by count. Per tag: one bar per month, height relative to the largest single month count across shown tags; that tag's peak month highlighted; total on the right.
- Insights:
  - always: "Your most frequent feeling: *X* (n entries)."
  - only with ≥ 4 months in the window: compare average per month in the last `min(3, floor(months/2))` months vs the earlier months. Largest increase ≥ 0.3/month → "Lately you're leaning toward *X*." Largest decrease ≤ −0.3/month (and a different tag) → "*Y* has faded from your recent entries."
- Fewer than 2 months: show "Trends take shape after a couple of months of logging."

### 4.14 Favourites & Top 5
- Any entry can be favourited. Favourites are grouped by category.
- Each category has an **All-Time Top 5**: ordered, max 5; reorder left/right; remove. Adding a 6th shows "Your Top 5 is full — remove one to add this." An entry in the Top 5 is not repeated in the favourites list below.
- Entry detail shows "#1 All-Time ✕" (tap removes) or "+ All-Time Top 5" for favourited entries.

### 4.15 "One year ago today"
- Home shows a memory card for a **finished** entry whose `createdAt` is within ±3 days of today one year ago (first match), with its review excerpt (≤160 chars).

### 4.16 Deleting
- Delete category → confirm ("Deleting it removes them too"), deletes its entries and their Top 5 / order data.
- Delete folder → confirm; its entries move to Uncategorised (nothing lost).
- Delete entry → confirm.
- Rename category/folder → updates all its entries and favourites keys.

---

## 5. Screens

Bottom navigation: **Home · Library · New · Favorites · Digest**. Sub-pages hide the bottom bar and have a back arrow (top-left) and, where relevant, search (top-right).

**Onboarding (first launch)** — "What should we call you?" (optional) → Home, then immediately opens *New category* if there are none.

**Home**
- Greeting by time of day ("Good morning/afternoon/evening, Name") and today's date; streak pill ("🔥 4 days in a row") when > 0.
- Four stat chips, each tappable: **Total** (count → All Entries), **Consumed** (all-time time → Consumed), **Finished** (count → Recently Finished), **Ongoing** (count → Still With You). Consumed label reads "all time".
- **Still with you** — horizontal row of ongoing cards (poster, progress ring %, title, category, "Ep 4/16" / "p.80/321"; "✓ Finished?" badge at the end). "See all".
- **Recently finished** — horizontal row of poster cards (most recently finished first). "All entries".
- **One year ago today** memory card (§4.15).
- Empty state with "Add your first entry".

**Still With You / Recently Finished / All Entries** — full pages. Rows of poster cards that scroll horizontally: Still With You 3 per row; the other two 4 per row; rows stacked vertically. Same card style as Home. Scoped search. Cards show **no time** here.

**Library** (root) — "Categories" header with counts. Two-column grid of category boxes: icon chip, name, ⋯ menu (Rename / Delete). If the category has posters, show the 3 most recent as a **fanned stack** (front flat, the next two tilted 6° and 12° and shifted down-right). Dashed "+ New category" tile spans the grid width. Search covers everything.
- New category dialog: name + "What lives here" (entry type), with note "This only decides which fields appear when you log an entry here."

**Category page** — title, type, counts. Folder grid (⋯ menu per folder: Rename / Delete), dashed "+ New folder" tile, then Uncategorised entries as a list. Search scoped to the category.

**Folder page** — entries in that folder as a list. Search scoped to the folder.

**New entry / Edit entry** (same form) — category & folder pickers, title, poster upload, status (Ongoing / Finished), rating (half stars), type-specific fields, "Your words" textarea (word count), "How it felt" tags (+ custom). New: "Preserve this memory →". Edit: "Save changes →", returns to the entry detail (back from detail then goes to wherever the user came from — not back into the form). Opening *New* always starts from an empty form.

**Entry detail** — poster hero; type badge + Top 5 pill (same size, vertically aligned); title; meta chips (rating, key fields, total time); progress bar for progress types; **Your words** (§4.7); How it felt tags; other fields (cast etc.); **Your journey** with "Log a new session" and "Log a rewatch"; each journey item has edit / delete icons; Edit and Delete entry; favourite toggle.

**Favorites** — horizontally scrolling category tabs; tapping one updates the content in place (no full-page flash) and smoothly scrolls the selected tab into view. Sections: "All-Time Top 5 · Category" and "Favourites · Category", poster cards 4 per row.

**Digest** — Week / Month / Year toggle; month/year arrows (§4.12).
- Hero: eyebrow (Weekly digest / Monthly digest / Year in review), period label, a narrative sentence built from the period's time, top category and top tag (Year also says how many were finished).
- Stat grid: new, time, pages, streak or sessions (Year: new, time, pages, finished).
- Best-rated entry of the period (poster, review excerpt, stars).
- Year only: month-by-month bar chart (peak month highlighted), "Top rated this year" (top 3), **Share your year** card (§7).
- Month & Year: **Mood trends** card → Mood Trends page.
- Week & Month: "Still going" (most-progressed ongoing entry), sessions count card.
- Closing note card: "You kept something that mattered."

**Consumed** (from the Consumed chip)
- Hero: "All time, all worlds", the total (number large, unit smaller italic), "+X this month" pill, the perspective line, an 8-week activity sparkline ("Last 8 weeks"), and a category share bar (top 3 + Other) with legend.
- Then one section per category (sorted by time, with its total), and under it each folder (and Uncategorised) as a small heading with its time and "See all", followed by one row of up to 4 poster cards that **do** show each entry's time.
- Folder page ("See all"): all entries as 4-per-row rows with time. Scoped search.
- Search on the root page searches all entries (standard results layout).

**Mood Trends** — §4.13. Header "How you've felt"; "What stands out" insights; "Feelings over time" rows with month-initial axis; "Month by month" list with each month's top 3 feelings as chips.

**Settings** — theme picker (6 themes + Custom with base & accent pickers), light/dark, your name, archive started date, export / import archive, danger zone (clear all data, double-confirmed).

---

## 6. Visual design

- **Fonts** (all SIL Open Font License — free to bundle): Playfair Display (titles, numbers), Lora (italic voice text, quotes), DM Sans (UI labels, buttons). Weights in `design-tokens.json`.
- **Colours**: every theme × mode is in `docs/design-tokens.json`, including the formulas that derive a custom palette from two colours. Tokens: bg, surface, card, text, muted, accent, accent2, border, tag, btn-text, timeline.
- **Style**: flat and quiet. Thin borders, rounded cards (≈14–20 px radius), lots of whitespace, serif headings, small uppercase letter-spaced eyebrows. The user has repeatedly rejected gradients and busy hero sections — keep heroes slim.
- **Signature details**: poster cards with a soft shadow that lifts on press; the fanned poster stack on category boxes; cards falling into place one after another (≈0.5 s, 55 ms stagger, capped) when a grid appears; journey timeline with dots and connecting line.
- **Copy voice**: warm, literary, second person ("Your words", "Still with you", "You kept something that mattered.").
- On touch devices, hover effects become press feedback.

---

## 7. Year in Review card

Opened from "Share your year" on the Year digest (only when that year has data). Rendered as a **1080 × 1350 PNG** in the current theme's colours:

1. "RELIC · YEAR IN REVIEW" eyebrow; the year, very large.
2. "TIME SPENT IN OTHER WORLDS"; total (big number + italic unit); perspective line.
3. Three stats: Started · Finished · Top feeling.
4. "WHERE IT WENT": share bar of top 3 categories + Other (Other drawn as dimmed muted so it stays visible on dark themes), with legend.
5. "MONTH BY MONTH": 12 bars, peak month in accent; empty months as thin flat lines.
6. "TOP RATED": up to 3 entries active that year — poster (or a tile with the title's first letter), title wrapped to 3 lines with ellipsis, stars.
7. Footer: "your stories, preserved".

Preview in a dialog with **Save image** and **Share** (native share sheet). A cancelled share must not trigger a save. Text that doesn't fit shrinks to a minimum size, then truncates.

---

## 8. Settings & archive

- **Export** produces a single JSON archive of everything (profile, categories, entries, favourites) including posters. Name: `relic-archive.json`.
- **Import** replaces all data after confirmation and restores the theme too (the prototype forgot the theme — §11). It keeps the app's current light/dark mode: the app remembers the last mode, including after an import.
- Clear all data requires confirmation.

---

## 9. Importing prototype data

The user's real data exists today as a prototype export (`relic-archive.json`). Shape:

```
{ userName, theme, customBase, customAccent,
  sections: [{ id, name, type, folders: [string] }],
  entries:  [{ id, type, title, poster (data URL), rating, tags, review,
               section (category NAME, not id), category (same), folder, status,
               createdAt, finishedAt?, favorite?,
               <type fields>, sessions: [...], rewatches: [...] }],
  topFavorites:  { [sectionName]: [entryId] },
  favoriteOrder: { [sectionName]: [entryId] },
  createdAt }
```

Rules for the importer (do this once, carefully, with tests built from real-looking fixtures):
- Entries reference categories **by name**; map to category ids.
- Sessions: `first:true` → `isStart`; `doneAtStart` → `startedFinished`; `eps` → episodes; `from`/`to` → pages; `mins` → minutes; `date` → at.
- If an entry has **no start session** but has a review or a progress type: create one at `createdAt` with the review as note and `episodes = max(0, watchedEpisodes − sum(other sessions' episodes))` (books: `toPage = pagesRead`). This mirrors how the prototype repaired old data.
- Legacy statuses `watching` / `reading` → `ongoing`.
- Finished entries without `finishedAt` → use `createdAt` (best available).
- Posters: decode data URLs, **resize** (e.g. long edge 600 px), store as files.
- `firstUsedAt` isn't in the export → use the earliest `createdAt`.
- Rewatch `full` missing → true.

---

## 10. Out of scope for v1 (planned next)

Accounts (Google & Apple sign-in), cloud storage and sync across devices, store release. The core and local storage should be designed so sync can be added without changing the core rules (stable ids, `updatedAt` on records, soft deletes are worth considering).

---

## 11. Known gaps & open questions

| # | Item | Notes |
|---|---|---|
| 1 | ~~Music counts zero time~~ | **Decided:** album length (mins) field, counted like a film; blank length estimates tracks × 3.5 min. |
| 2 | **"Other" duration is free text and unused** | Only session minutes count. **Decided:** keep it free text for now. |
| 3 | **Timezone** | Prototype used UTC days; new app must use local days (§4.11). |
| 4 | **Posters stored full-size inline** | Prototype risked filling browser storage. Resize and store as files. |
| 5 | **Prototype import didn't restore theme** | Fixed by §8. |
| 6 | **Mood attribution** | Feelings count in the month the entry was created; edited tags have no history. Acceptable for v1. |
| 7 | **Consumed page with very large libraries** | Folder previews are the slow part; render them lazily. |
| 8 | ~~Rewatch rating~~ | **Decided:** stored on the rewatch only and never changes the entry rating; rewatch ratings are separate history. |
