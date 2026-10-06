# Relic — Project status and handoff

Read this after CLAUDE.md at the start of every session. It records where
the work stands and what the owner has already decided in conversation, so
nothing needs re-explaining. **Update it at the end of each piece of work**
(same commit), keeping it short and current; delete what's no longer true.

Last updated: 2026-10-06, after Phase 2 and the readiness audit (Setup section, `.env` ignored).

## Where we are

| Phase | State |
|---|---|
| 0 Decisions and spike | Done. Gio + SQLite (ncruces) chosen; spike runs on Android and iOS simulator in CI. |
| 1 Core library | Done. Every SPEC §4 rule in `core/`, table-driven tests, ~93% coverage. |
| 2 Storage and import | Done. "Done when" verified: the real archive, imported and read back from SQLite, gives the same counts, time totals and Top 5 as the prototype's own formulas. |
| 3 Screens | **Next.** Start with step 1 (app shell). Nothing of the real UI exists yet; `ui/` only has the Phase 0 spike (stat chip + poster card + theme tokens). |

## What exists

- `core/` — model and all rules. `core/doc.go` maps each SPEC §4 subsection to its file. Entry points the UI will use: `Library` (categories, folders, entries, favourites, scopes such as `StillWithYou`, `RecentlyFinished`, `InCategory`, `InFolder`), `Entry` (`LogSession`, `EditSession`, `LogRewatch`, `SetStatus`, `Journey`, `YourWords`, `Progress`, `ShouldPromptFinished`), `NewTimeIndex` (build once per render; totals, monthly, weekly, `ConsumedByCategory`, `ShareBar`), `Library.Digest` / `DigestBounds` / `MoodTrends` / `OneYearAgo` / `Search`, formatting (`FormatDuration`, `Perspective`, `FormatDate`, `Greeting`, `ResultsHeader`). Day/month bucketing functions take a `*time.Location`.
- `store/` — SQLite (migration 2 = library schema). `store.Store`: `Load`, `ReplaceAll`, `Update(func(Writer) error)` for single-transaction saves (`SaveProfile`, `SaveCategories`, `SaveEntry`, `DeleteEntry`, `SaveFavourites`). Posters are files in `posters/` beside the database (`WritePosters`, `RemoveUnusedPosters`); `Entry.Poster` is just the file name.
- `importer/` — `importer.Read` turns `relic-archive.json` into a `core.Library` plus resized JPEG posters and a `Report` of repairs (SPEC §9). `importer.Summarize` prints counts/time/Top 5.
- `cmd/import` (archive → database + summary), `cmd/scrub` (makes a de-personalised test copy), `cmd/snapshot` (screen → PNG), `cmd/relic` (the app; currently the spike plus a store launch counter that CI checks).
- `importer/testdata/scrubbed-archive.json` — scrubbed copy of the owner's real archive (same structure, dates, numbers, ids; words and posters replaced). Use it as the realistic test fixture.

## Decisions the owner made in conversation

These are already reflected in SPEC.md / CLAUDE.md; listed here so they aren't re-asked.

- **Partial rewatch counts time**: when "Partial rewatch" is picked (series/podcast), the sheet asks **"Episodes rewatched"**; time = episodes × episode length. Full rewatch = total episodes × episode length. (SPEC §2, §4.5)
- **Music counts time**: new **"Length (mins)"** field on music entries, counted like a film (once when logged, again per relisten). Blank length → tracks × 3.5 min. (SPEC §3, §4.5, §11 #1)
- **"Other" duration stays free text** and counts no time; only session minutes count. (SPEC §11 #2)
- Accepted from the Phase 1 report without objection: a logged "0 minutes" on a book means zero (blank = estimate 1.5 min/page; the importer maps the prototype's `mins: 0` to blank); totals of 0 mean "not given"; editing an entry doesn't change the start session's "logged as finished" flag; the edit form's "Watched so far" / "Pages read so far" edit only the **start session's** amount (`Entry.StartEpisodes` / `StartPages`), not the running total.

The two new form fields (Episodes rewatched, Length (mins)) are not in the prototype. The owner approved them from these sketches, so build them like this, styled like neighbouring fields:

```
Rewatched in full?
[ Yes, all of it ] [•Partial rewatch ]
Episodes rewatched            <- only shown when Partial is picked
[ 4            ]

Artist      [ ...... ]
Tracks      [ 11 ]  Length (mins) [ 44 ]
```

## Open questions to raise with the owner (not yet asked)

1. **Podcasts count no time.** The prototype's podcast form has no episode-length field, so podcast time is always 0 (the real archive's podcast shows 0 min). Suggest adding "Ep. length (mins)" to podcasts as series have; the core already uses `EpisodeDuration` for podcasts.
2. SPEC §11 #8: should a rewatch rating ever update the entry rating? (Currently never.)
3. The prototype export has no light/dark mode; imports default to light. Fine, or remember last mode?

## The owner's real data

- The real archive is **not in the repo** (gitignored; never commit it). It lives on the owner's laptop at `C:\Users\saniy\Downloads\relic-archive.json`. Cloud sessions can't see it: use the scrubbed fixture, and ask the owner to run real-data checks locally, e.g.
  `RELIC_ARCHIVE=C:/Users/saniy/Downloads/relic-archive.json go test ./importer -run Prototype -v`
  or `go run ./cmd/import -in ../relic-archive.json -db build/relic.db`.
- Shape of the real data (also true of the scrubbed copy): 8 categories (2 have no stored type and are guessed from their names; one is empty), 15 entries, 23 sessions, 2 rewatches, 15 posters, 9 favourites, one Top 5 entry. Totals: 15 entries, 11 finished, 4 ongoing, 4,681 min (3.3 days). Importing produces no repair warnings.
- The owner tests on this real data and compares against the prototype (`reference/relic.html`, opened in a browser with the archive imported).

## CI (GitHub Actions, `.github/workflows/build.yml`)

- Jobs: vet+test (Linux, installs Gio's X11/Wayland libs), Android APK, Android emulator run (screenshot + checks the log for `relic: store ok, launch 1/2`), iOS simulator run on an Intel Mac (same check).
- The latest run (Phase 2, commit 6e240a9) passed every job. The two runs before it failed **only** in the emulator/simulator launch steps, from slow cloud machines: the iOS app started after the fixed `sleep`s had passed, and one Android emulator boot was cancelled at about 15 min. The code built and its tests passed in those runs. **Suggested fix, not yet done:** replace the fixed sleeps with polling loops that wait for the "store ok" log line (up to ~2 min), and give the emulator step a longer boot timeout. Do this if failures recur.
- Download results from the run's Artifacts section: `relic-apk`, `emulator-screenshots`, `ios-screenshots`.

## Working in a cloud session

- The cloud session is Linux. The owner works on Windows 11 in VS Code with a 3.8 GB RAM laptop and no Android phone (so no local emulator). Write instructions for the owner as Windows + VS Code + command-line steps.
- `go test ./...` on Linux needs Gio's system libraries for the `ui` package (the apt list is in the workflow's "Install Gio's Linux libraries" step). If they can't be installed, run `go test ./core/... ./store/... ./importer/...` and say so.
- There's no display in the cloud, so `go run ./cmd/relic` won't open a window. Check screens with `cmd/snapshot` PNGs (needs `EGL_PLATFORM=surfaceless`; see CLAUDE.md "Setup" for the cloud setup script and variables) and the CI emulator screenshots, and ask the owner to try the Windows build (`go run ./cmd/relic`).
- Commit and push after each finished piece (CLAUDE.md). Commit messages end with the Co-Authored-By line given by the harness.

## Phase 3 — how to start

1. Read SPEC §5 and §6, `docs/design-tokens.json`, and the matching parts of `reference/relic.html` (its CSS is at the top; the render functions are named per screen, e.g. `renderHome`, `renderLibrary`, `renderDetail`).
2. Build in ROADMAP order, starting with the app shell: theme tokens (6 themes × light/dark + custom-palette formulas), bundled fonts, bottom navigation, back behaviour (back closes an open search first, SPEC §4.9), dialogs, toasts.
3. The app should open the SQLite store at startup, `Load` the library, build one `core.TimeIndex` per render, and save through `Store.Update`. The first-launch flow is onboarding (SPEC §5); importing the archive belongs to Settings (step 9). For testing earlier, it's fine to add a temporary dev-only import path that uses `importer.Read`.
4. Before any visual choice the spec doesn't define, show the owner options first (CLAUDE.md design guardrails). No gradients, slim heroes, keep the prototype's card sizes.
