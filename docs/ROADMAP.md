# Relic — Build Roadmap

Native Go app for Android and iOS, built offline-first. Each phase ends with a **Done when** check; don't start the next phase until it passes.

## Before you start

- [x] Export your data from the prototype (Settings → Export) and keep `relic-archive.json` safe. It's your only backup and the import test data for Phase 4.
- [x] Install Go, Git, Claude Code.
- [ ] Android: SDK + NDK command-line tools and JDK 17. With no phone, or too little RAM for an emulator, the GitHub Actions workflow runs the app on a cloud emulator.
- [ ] iOS (later phases): a Mac with Xcode. iOS builds are not possible without one.

## Phase 0 — Decisions and spike (short) — ✅ done

Decide by building, not debating:

1. **UI toolkit.** Recommended: **Gio** (gioui.org) — full control over drawing, which Relic's custom look needs. Alternative: **Fyne** — quicker, but its standard widgets fight a custom design. Spike: render one Home stat chip and one poster card with the bundled Playfair/DM Sans fonts and the Linen theme, and run it on the Android emulator. Pick the toolkit whose result is closer to the prototype for less code.
2. **Local storage.** Candidates: SQLite (cgo driver), a pure-Go SQLite, or an embedded key-value store. Requirement: must build for **both** Android and iOS — verify with a tiny build for each target before committing.
3. **Repo layout.** Proposed:
   ```
   core/        pure Go: model + all rules from SPEC §4 (no UI, no storage imports)
   store/       persistence behind an interface
   importer/    prototype archive → model (SPEC §9)
   ui/          screens and components
   assets/      fonts, icons
   cmd/relic/   app entry point
   docs/        SPEC.md, ROADMAP.md, design-tokens.json
   reference/   relic.html (the prototype — read-only)
   ```

**Done when:** toolkit and storage are chosen and written into CLAUDE.md "Decisions"; the spike runs on the Android emulator (locally or in the GitHub Actions run).

## Phase 1 — Core library — ✅ done

Port every rule in SPEC §4 into `core/` with table-driven tests. Suggested order: model & ids → status/finishedAt → sessions & start session → progress & reached-end → time events & totals → duration formatting & perspective → Your words → journey labels & start verbs → search & scopes → streak → local-date bucketing → digest periods & floor → mood trends → favourites/Top 5 → one-year-ago.

Include tests for the cases that broke in the prototype:
- 2-episode drama logged with 1 watched; one more session → reached end → prompt.
- 8 episodes, 5 watched at creation, remaining 3 across two sessions → prompt only at 8.
- Book session "from page 0" saves as 0.
- Finished at creation → no "Started …" label.
- Full rewatch doubles series time; partial adds nothing.
- Book with pages but no minutes → estimated time; logged minutes override.
- Category totals = sum of folder totals = all-time total.
- Digest month navigation can't go before first use or past today.
- Late-night log near midnight lands on the local date.

**Done when:** `go test ./core/...` passes; coverage of every SPEC §4 subsection.

## Phase 2 — Storage and import — ✅ done

- `store/` implements save/load for the full model behind an interface the UI uses.
- `importer/` converts `relic-archive.json` (SPEC §9), resizing posters to files.
- Test the importer on a scrubbed copy of the real archive.

**Done when:** importing the real archive and reading it back reproduces the same counts, time totals and Top 5 as the prototype shows.

## Phase 3 — Screens (Windows build day to day, Android emulator via GitHub Actions) — ⏭ next

Build in this order, checking each against the prototype side by side:

1. App shell: theme tokens, fonts, bottom navigation, back behaviour, dialogs, toasts. (Built.)
2. Library: categories → category → folder, create / rename / delete, poster fan, search. (Built.)
3. New / Edit entry form; Entry detail with journey, logging, editing, finished prompt. (Built.)
4. Home with stat chips, Still with you, Recently finished, memory card; the three "see all" pages. (Built.)
5. Favorites with Top 5. (Built.)
6. Digest (week / month / year), Mood Trends. (Built.)
7. Consumed and its folder pages. (Built.)
8. Year in Review image + share sheet. (Built: image, preview, Save image. Share waits on the owner's choice.)
9. Settings, export / import, onboarding.

**Done when:** every screen in SPEC §5 works on Android with imported real data, in at least Linen light and one dark theme.

## Phase 4 — iOS

Build and run on iOS (Mac + Xcode), fix platform differences (share sheet, photo picker, safe areas, fonts).

**Done when:** the same checklist as Phase 3 passes on an iPhone or the iOS simulator.

## Phase 5 — Accounts and sync (backend)

Go API server, database, Google and Apple sign-in (Apple generally requires offering Sign in with Apple when Google sign-in is offered), sync of the local store, conflict handling. Reuse `core/` on the server.

## Phase 6 — Release

Developer accounts (Apple, Google), signing, icons, store listings, privacy policy, review.
