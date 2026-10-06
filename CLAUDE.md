# Relic

Relic ("your stories, preserved") is a personal archive for films, dramas, books, podcasts and music, and the feelings the user had along the way. We are rebuilding a finished web prototype as a **native Go app for Android and iOS**, offline-first, with accounts and sync added later.

## Key files

- **`docs/STATUS.md` — read first.** Where the work stands, decisions the owner made in conversation, open questions, real-data notes, CI notes and how to start the next phase. Update it in the same commit as each finished piece of work.
- `docs/SPEC.md` is the source of truth for data model, rules and screens. Read the relevant section before working on a feature. If the spec and the prototype disagree, the spec wins.
- `docs/ROADMAP.md` lists the phases and their "Done when" checks. Work in phase order.
- `docs/design-tokens.json` holds all theme colours (6 themes × light/dark, plus custom-palette formulas) and fonts.
- `reference/relic.html` is the working prototype. It is **read-only**: use it to see how things look and behave, never edit it. Open it in a browser to compare screens side by side.

## Decisions

- Language: **Go**, for the app and later the backend.
- UI toolkit: **Gio** (gioui.org v0.10). The Phase 0 spike matched the prototype's chip and poster card with custom drawing and bundled fonts, and runs on Android.
- Local storage: **SQLite via github.com/ncruces/go-sqlite3** (SQLite translated to pure Go, no cgo), in `store/`. Uses Go's own file I/O, so it runs inside Android's syscall filter; builds for Windows, Android and iOS with no C toolchain. Schema changes are numbered migrations tracked in `PRAGMA user_version`. (modernc.org/sqlite was rejected: it crashed on Android x86_64.)
- Architecture: pure `core/` package for every rule in SPEC §4. UI and storage depend on core, never the other way round.
- Offline-first: the app works fully without a network. Sync comes in Phase 5.
- Storage (Phase 2): tables per record type, times as fixed-width UTC text, type-specific fields and lists as JSON. UI writes through `store.Store` (`Load`, `ReplaceAll`, `Update` for one transaction). Posters are JPEG files (long edge ≤ 600 px) in a `posters/` folder beside the database; entries hold the file name. Imported entries and categories keep their prototype ids.
- App shell (Phase 3 step 1): `ui.App` owns the window: tab, sub-page stack, dialog, toast. Screens implement `ui.Screen`; a page with its own search implements `ui.BackHandler`. Back order: dialog → page search → sub-page → Home → system (Android leaves the app); Escape does the same on Windows. Theme changes go through `App.SetTheme`, which saves the profile. Icons are the prototype's SVG markup, parsed by `ui.IconSVG`.
- Symbols (★ ☆ ♥ ✓ ✕ ✦ ✧) come from a bundled 3 KB subset of Noto Sans Symbols 2 (OFL), with its line spacing reduced so it never makes a line taller. Colour emoji (🔥, 🎨) aren't drawable in Gio yet; decide per screen (open question in STATUS).
- Gio blends colours in linear light, browsers in sRGB: translucent black (scrims, shadows) goes through `shade()` so it darkens like the prototype's `rgba(0,0,0,a)`.
- Core model (Phase 1): ids are random 128-bit hex. Entries reference categories by id, so renaming a category touches nothing else. A session's minutes are optional (`nil` = not logged, so books estimate 1.5 min/page; an explicit 0 counts as 0); the importer must map the prototype's `mins: 0` to "not logged". Partial rewatches store "episodes rewatched"; music counts album length (or tracks × 3.5 min). Totals (episodes, pages) of 0 mean "not given" and never count as reached. Editing an entry doesn't rewrite the start session's "logged as finished" flag.

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

- The owner has **no Android phone** and a **3.8 GB RAM** laptop, so the local Android emulator isn't usable. Day to day, check screens in the Windows build (phone-sized window) and `cmd/snapshot` PNGs. The Android check is the GitHub Actions run (`.github/workflows/build.yml`), which builds the APK, runs it on a cloud emulator and uploads a screenshot.
- The owner edits in **VS Code**, not Android Studio. Don't give instructions that need the Android Studio app; use command-line tools (`sdkmanager`, `avdmanager`, `emulator`, `adb`, `gogio`) and VS Code.
- All work is saved to a private GitHub repo (`origin` → github.com/saniyaa61/Relic). **Commit and push after each piece of finished work**, with `go vet ./...` and `go test ./...` passing first.
- When a step needs something installed, tell the owner what to install and how, step by step, before it's needed.
- The owner is the product designer and tests on real data. Describe changes in plain language, not code.
- Ask before changing anything visual that the spec doesn't define.
- Report honestly: say what was tested and how, and what wasn't.

## Setup

- **Windows (the owner's laptop):** install Go (the version in `go.mod`, or any newer Go, which fetches it automatically), Git and VS Code with the Go extension. `go run ./cmd/relic` then works with no further steps. Android tools (SDK, NDK, JDK 17, `gogio`) are only needed to build the APK locally; CI builds it otherwise.
- **Linux / cloud sessions:** the `ui` package needs Gio's system libraries, the same list as the "Install Gio's Linux libraries" step in `.github/workflows/build.yml` (keep the two in step):
  `apt-get install -y gcc pkg-config libwayland-dev libx11-dev libx11-xcb-dev libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev libxcursor-dev libvulkan-dev`
  The cloud environment's setup script installs these. Without them, only `go test ./core/... ./store/... ./importer/...` runs.
- **No display (cloud):** set `EGL_PLATFORM=surfaceless` so `cmd/snapshot` can render PNGs.
- No API keys or secrets are needed. Never commit `.env` files or the real archive (both are gitignored).

## Commands

- Checks: `go vet ./...` and `go test ./...`
- Run on Windows: `go run ./cmd/relic`
- Screen to PNG: `go run ./cmd/snapshot -screen home -o build/home.png` (add `-theme midnight -dark`; `-screen` takes home, library, new, favorites, digest, subpage, search, prompt, confirm, toast, toast-error)
- Import the prototype archive into a database and print counts, time and Top 5: `go run ./cmd/import -in ../relic-archive.json -db build/relic.db` (posters go to `build/posters/`).
- Prototype screenshots to compare with (cloud sessions; uses the pre-installed Chromium): `MODE=light node tools/protoshot.js $PWD build/proto relic_v3`
- Compare an import with the prototype's own formulas: `go test ./importer -run Prototype -v` (scrubbed copy); add `RELIC_ARCHIVE=C:/Users/saniy/Downloads/relic-archive.json` to also check the real archive.
- Refresh the scrubbed test copy after a new export: `go run ./cmd/scrub -in ../relic-archive.json -out importer/testdata/scrubbed-archive.json`. Never commit the real archive.
- Android APK locally (needs `ANDROID_HOME`, the NDK, and JDK 17's `bin` on PATH for `keytool`): `gogio -target android -arch arm64,amd64 -minsdk 24 -appid com.saniyaa61.relic -name Relic -o build/relic.apk ./cmd/relic`
- Android on an emulator: push to GitHub; the **Build** workflow uploads `relic-apk` and `emulator-screenshots` as run artifacts.
- iOS: needs a Mac (Phase 4).
