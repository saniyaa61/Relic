# Relic

Relic ("your stories, preserved") is a personal archive for films, dramas, books, podcasts and music, and the feelings the user had along the way. We are rebuilding a finished web prototype as a **native Go app for Android and iOS**, offline-first, with accounts and sync added later.

## Key files

- `docs/SPEC.md` is the source of truth for data model, rules and screens. Read the relevant section before working on a feature. If the spec and the prototype disagree, the spec wins.
- `docs/ROADMAP.md` lists the phases and their "Done when" checks. Work in phase order.
- `docs/design-tokens.json` holds all theme colours (6 themes × light/dark, plus custom-palette formulas) and fonts.
- `reference/relic.html` is the working prototype. It is **read-only**: use it to see how things look and behave, never edit it. Open it in a browser to compare screens side by side.

## Decisions

- Language: **Go**, for the app and later the backend.
- UI toolkit: _pending Phase 0 spike_ (recommended: Gio; alternative: Fyne).
- Local storage: _pending Phase 0 spike_ (must build for Android and iOS).
- Architecture: pure `core/` package for every rule in SPEC §4. UI and storage depend on core, never the other way round.
- Offline-first: the app works fully without a network. Sync comes in Phase 5.

Update this section when a decision is made.

## Conventions

- **Rules live in `core/`, with tests.** Any logic about time, progress, status, dates, search, digest or moods goes in `core/`, covered by table-driven tests, before the UI uses it.
- **Derive, don't store.** `watchedEpisodes` and `pagesRead` are computed from sessions. Don't keep parallel copies that can drift.
- **Zero is a value.** Never treat 0 as "missing" (page 0, 0 episodes, 0 minutes).
- **Store UTC, bucket locally.** Day/week/month/year grouping uses the user's local timezone.
- **One status function** sets and clears `finishedAt` (SPEC §4.2).
- **One search implementation**; pages only pass the entry set, scope label and layout.
- **Compute time events once per render** and reuse them across totals and charts.
- Keep the copy voice warm and literary, using the same wording as the prototype unless the spec says otherwise.
- Small, reviewable commits. Run `go vet ./...` and `go test ./...` before committing.

## Design guardrails (from the owner's feedback)

- Flat, quiet design: thin borders, rounded cards, serif headings, small letter-spaced eyebrows.
- **No gradients, no busy hero sections.** Heroes stay slim and simple.
- Match the prototype's spacing and card sizes. Don't resize poster cards or change layouts unasked.
- Bundle the fonts: Playfair Display, Lora and DM Sans (all OFL licensed).
- When unsure about a visual choice, show options before building one.

## Prototype quirks NOT to copy

- The prototype synthesised a missing "first session" on the fly and wrote to storage while rendering. In Go, the start session is a normal stored record, and repairing old data happens once in `importer/` (SPEC §9).
- It bucketed dates in UTC.
- It stored posters as full-size inline data.
- Its import didn't restore the theme.

## Working with the owner

- The owner is the product designer and tests on real data. Describe changes in plain language, not code.
- Ask before changing anything visual that the spec doesn't define.
- Report honestly: say what was tested and how, and what wasn't.

## Commands

_Fill in after Phase 0, e.g._ `go test ./...` · building the Android APK · running on the emulator · building for iOS.
