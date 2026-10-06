# Relic — Project status and handoff

Read this after CLAUDE.md at the start of every session. It records where
the work stands and what the owner has already decided in conversation, so
nothing needs re-explaining. **Update it at the end of each piece of work**
(same commit), keeping it short and current; delete what's no longer true.

Last updated: 2026-10-06, after Phase 3 step 3 (entry form, entry page, logging).

## Where we are

| Phase | State |
|---|---|
| 0 Decisions and spike | Done. Gio + SQLite (ncruces) chosen; spike runs on Android and iOS simulator in CI. |
| 1 Core library | Done. Every SPEC §4 rule in `core/`, table-driven tests, ~93% coverage. |
| 2 Storage and import | Done. "Done when" verified: the real archive, imported and read back from SQLite, gives the same counts, time totals and Top 5 as the prototype's own formulas. |
| 3 Screens | **In progress.** Step 1 (app shell) done. Steps 1–2 done. Step 3 (New/Edit entry form, entry page, logging) built and pushed; waiting for the owner's Windows check and three answers (below). Step 4 (Home) is next. |

## What exists

- `core/` — model and all rules. `core/doc.go` maps each SPEC §4 subsection to its file. Entry points the UI will use: `Library` (categories, folders, entries, favourites, scopes such as `StillWithYou`, `RecentlyFinished`, `InCategory`, `InFolder`), `Entry` (`LogSession`, `EditSession`, `LogRewatch`, `SetStatus`, `Journey`, `YourWords`, `Progress`, `ShouldPromptFinished`), `NewTimeIndex` (build once per render; totals, monthly, weekly, `ConsumedByCategory`, `ShareBar`), `Library.Digest` / `DigestBounds` / `MoodTrends` / `OneYearAgo` / `Search`, formatting (`FormatDuration`, `Perspective`, `FormatDate`, `Greeting`, `ResultsHeader`). Day/month bucketing functions take a `*time.Location`.
- `store/` — SQLite (migration 2 = library schema). `store.Store`: `Load`, `ReplaceAll`, `Update(func(Writer) error)` for single-transaction saves (`SaveProfile`, `SaveCategories`, `SaveEntry`, `DeleteEntry`, `SaveFavourites`). Posters are files in `posters/` beside the database (`WritePosters`, `RemoveUnusedPosters`); `Entry.Poster` is just the file name.
- `importer/` — `importer.Read` turns `relic-archive.json` into a `core.Library` plus resized JPEG posters and a `Report` of repairs (SPEC §9). `importer.Summarize` prints counts/time/Top 5.
- `cmd/import` (archive → database + summary), `cmd/scrub` (makes a de-personalised test copy), `cmd/snapshot` (any shell state → PNG, in any theme), `cmd/relic` (the app: loads the library, runs the shell, logs a store launch counter that CI checks).
- `ui/` app shell (Phase 3 step 1): `App` (tabs, sub-page stack, back order, dialog, toast, `SetTheme` saving the profile), all 6 themes × light/dark + custom palette (`PaletteFor`, `CustomPalette`, tested against the tokens and the prototype's own JS), bottom nav, logo top bar, sub-page bar with search (`Search`), `PromptDialog` / `ConfirmDialog`, toasts, `IconSVG` (prototype SVG icons), `Paragraph` (CSS line-height), `shade()` (browser-matching translucent black). Tabs other than Home are placeholders; Home is a temporary "shell preview" with buttons for dialogs, toasts and a theme picker sub-page. `preview.go` and `snapshot.go` hold the temporary bits.
- `ui/` Library (step 2), `library.go`: categories grid with the fanned poster stack, ☰ menus (Rename / Delete), dashed "+ New category" tile, category page (folder grid, "+ New folder", Uncategorised rows), folder page, scoped search on all three, card drop-in animation. Dialogs use the prototype's wording; changes save through `App.save` (reloads from the store if a save fails). Entry rows open a placeholder until step 3. Shared pieces: `entryrow.go` (entry rows, stars, tag chips, empty state, `searchResults`), `libparts.go` (`addTile`, `cardMenu`, `cardDrop`, dashed borders), `typepicker.go` ("What lives here"), `images.go` (posters read from `App.PosterDir`, resized once and cached). `core` gained `SetCategoryType` and `CoverEntries`.
- `ui/` step 3: `entryform.go` (New / Edit form; `formfields.go` holds each type's fields, incl. podcast "Ep. length" and music "Length"), `entrydetail.go` (hero, badges + Top 5 pill, meta chips with total time, progress, Your words, tags, cast, journey timeline, favourite, two-tap delete), `logsheet.go` + `sheet.go` (bottom sheet to log / edit sessions and rewatches, partial rewatch "Episodes rewatched", the reached-the-end prompt, deleting journey items). Controls: `dropdown` (category, folder, type), `dateField` + month calendar dialog, multi-line `Input`, `starInput` (half stars), `tagPicker`, `uploadBox` (system file picker via gioui.org/x/explorer, `App.ChooseImage`; pictures resized by `store.ResizePoster`). `App.timeIndex()` builds time events once per frame.
- `cmd/relic -import <archive>`: temporary desktop-only way to load the prototype archive (replaces all data, keeps light/dark mode) until Settings has Import (step 9). Remove it then.
- `tools/protoshot.js` screenshots the prototype in the pre-installed Chromium for side-by-side checks (see CLAUDE.md Commands).
- `importer/testdata/scrubbed-archive.json` — scrubbed copy of the owner's real archive (same structure, dates, numbers, ids; words and posters replaced). Use it as the realistic test fixture.

## Decisions the owner made in conversation

These are already reflected in SPEC.md / CLAUDE.md; listed here so they aren't re-asked.

- **Partial rewatch counts time**: when "Partial rewatch" is picked (series/podcast), the sheet asks **"Episodes rewatched"**; time = episodes × episode length. Full rewatch = total episodes × episode length. (SPEC §2, §4.5)
- **Music counts time**: new **"Length (mins)"** field on music entries, counted like a film (once when logged, again per relisten). Blank length → tracks × 3.5 min. (SPEC §3, §4.5, §11 #1)
- **"Other" duration stays free text** and counts no time; only session minutes count. (SPEC §11 #2)
- Accepted from the Phase 1 report without objection: a logged "0 minutes" on a book means zero (blank = estimate 1.5 min/page; the importer maps the prototype's `mins: 0` to blank); totals of 0 mean "not given"; editing an entry doesn't change the start session's "logged as finished" flag; the edit form's "Watched so far" / "Pages read so far" edit only the **start session's** amount (`Entry.StartEpisodes` / `StartPages`), not the running total.

Decided 2026-10-06 (first cloud session):

- **Podcasts get "Ep. length (mins)"**, same as series. The core already counts podcast time from `EpisodeDuration`; the field goes on the podcast form in step 3. (SPEC §3)
- **Rewatch ratings never change the entry rating**; they are kept as separate history. (SPEC §2, §11 #8)
- **The app remembers the last light/dark mode, including after import.** The importer leaves the mode empty and `importer.Result.KeepMode` copies the app's current mode in; `cmd/import` does this, and the Settings import (step 9) must too. (SPEC §2, §8)
- **Middle tab says "New"** (SPEC §5), not the prototype's "Add".
- **Symbols from Noto Sans Symbols 2 approved** (★ ♥ ✓ ✕ ✦ as bundled).
- **"What lives here" is the dropdown (option A)**, as built.
- **Category and folder pages keep the bottom bar**, as in the prototype (overrides SPEC §5's "sub-pages hide the bottom bar" for these two pages).

The two new form fields (Episodes rewatched, Length (mins)) are not in the prototype. The owner approved them from these sketches, so build them like this, styled like neighbouring fields:

```
Rewatched in full?
[ Yes, all of it ] [•Partial rewatch ]
Episodes rewatched            <- only shown when Partial is picked
[ 4            ]

Artist      [ ...... ]
Tracks      [ 11 ]  Length (mins) [ 44 ]
```

## Open questions to raise with the owner

1. **(Asked 2026-10-06, awaiting answer)** Entry page hero: A the prototype's (blurred poster behind, fade into the page) or B flat (built; keeps "no gradients"). Switch: `detailHero` in `ui/entrydetail.go`.
2. **(Asked 2026-10-06, awaiting answer)** "Mark as finished" button: red as the prototype (built) or the accent colour. Switch: `markFinishedRed` in `ui/logsheet.go`.
3. **(Asked 2026-10-06, awaiting answer)** Date fields open a month calendar in a dialog (the prototype used the browser's date picker). Fine as is?
4. **Colour emoji.** Gio can't draw colour emoji, and the prototype uses 🔥 in the Home streak pill ("🔥 4 days in a row") and 🎨 in Settings. Show options when building Home (step 4): e.g. a small flame line icon in the accent colour, or bundled emoji images.

## The owner's real data

- The real archive is **not in the repo** (gitignored; never commit it). It lives on the owner's laptop at `C:\Users\saniy\Downloads\relic-archive.json`. Cloud sessions can't see it: use the scrubbed fixture, and ask the owner to run real-data checks locally, e.g.
  `RELIC_ARCHIVE=C:/Users/saniy/Downloads/relic-archive.json go test ./importer -run Prototype -v`
  or `go run ./cmd/import -in ../relic-archive.json -db build/relic.db`.
- Shape of the real data (also true of the scrubbed copy): 8 categories (2 have no stored type and are guessed from their names; one is empty), 15 entries, 23 sessions, 2 rewatches, 15 posters, 9 favourites, one Top 5 entry. Totals: 15 entries, 11 finished, 4 ongoing, 4,681 min (3.3 days). Importing produces no repair warnings.
- The owner tests on this real data and compares against the prototype (`reference/relic.html`, opened in a browser with the archive imported).

## CI (GitHub Actions, `.github/workflows/build.yml`)

- Jobs: vet+test (Linux, installs Gio's X11/Wayland libs), Android APK, Android emulator run (screenshot + checks the log for `relic: store ok, launch 1/2`), iOS simulator run on an Intel Mac (same check).
- Emulator and simulator launches used fixed pauses and failed now and then on slow cloud machines (most recently the iOS step on 2026-10-06: the simulator took ~3 min to boot). They now wait up to 3 minutes for the app's "store ok" log line instead. If a launch step still fails, read its log before calling it a flake.
- Download results from the run's Artifacts section: `relic-apk`, `emulator-screenshots`, `ios-screenshots`.

## How the owner checks each step (Windows)

Work goes straight to `main` (no pull requests). After each finished step, tell the owner: in VS Code's terminal, `git checkout main`, `git pull`, `go run ./cmd/relic`, then a short list of what to try and what should happen. Esc is the back button on Windows. The app's data lives in `C:\Users\saniy\AppData\Roaming\Relic\relic.db` (delete it for a fresh start).

## Working in a cloud session

- The cloud session is Linux. The owner works on Windows 11 in VS Code with a 3.8 GB RAM laptop and no Android phone (so no local emulator). Write instructions for the owner as Windows + VS Code + command-line steps.
- `go test ./...` on Linux needs Gio's system libraries for the `ui` package (the apt list is in the workflow's "Install Gio's Linux libraries" step). If they can't be installed, run `go test ./core/... ./store/... ./importer/...` and say so.
- There's no display in the cloud, so `go run ./cmd/relic` won't open a window. Check screens with `cmd/snapshot` PNGs (needs `EGL_PLATFORM=surfaceless`; see CLAUDE.md "Setup" for the cloud setup script and variables) and the CI emulator screenshots, and ask the owner to try the Windows build (`go run ./cmd/relic`).
- Commit and push after each finished piece (CLAUDE.md). Commit messages end with the Co-Authored-By line given by the harness.

## Phase 3 — how to continue

1. Read SPEC §5 and §6 and the matching parts of `reference/relic.html` (its CSS is at the top; render functions are named per screen, e.g. `renderLibrary`, `renderDetail`). Screenshot the prototype with `tools/protoshot.js` and compare with `cmd/snapshot` PNGs.
2. Next is step 4: Home (stat chips, Still with you, Recently finished, memory card) and the three "see all" pages. Replace `homePreview` in `preview.go`; `ui/postercard.go` and `ui/chip.go` are the Phase 0 spike's poster card and stat chip, to be brought up to date. Entry pages open with `newEntryDetail(id)`; time comes from `App.timeIndex()`. Settle the colour-emoji question (🔥 streak pill) with options first. Sub-pages go through `App.Push`, dialogs `App.ShowDialog`, sheets `App.ShowSheet`, saves `App.save`.
3. The app opens the store at startup and loads the library; the first-launch flow is onboarding (step 9). For testing with real data earlier, a temporary dev-only import path using `importer.Read` + `KeepMode` is fine.
4. Before any visual choice the spec doesn't define, show the owner options first (CLAUDE.md design guardrails). No gradients, slim heroes, keep the prototype's card sizes.
