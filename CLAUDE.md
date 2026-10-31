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
- UI toolkit: **Gio** (gioui.org v0.10). The Phase 0 spike matched the prototype's chip and poster card with custom drawing and bundled fonts, and runs on Android. Relic builds against its own copy, `third_party/gio` (v0.10.3, via `replace` in go.mod), to carry small fixes listed in `third_party/gio/RELIC-PATCHES.md` (first: touch slop 3dp → 8dp, so taps in scrolling pages aren't lost).
- Local storage: **SQLite via github.com/ncruces/go-sqlite3** (SQLite translated to pure Go, no cgo), in `store/`. Uses Go's own file I/O, so it runs inside Android's syscall filter; builds for Windows, Android and iOS with no C toolchain. Schema changes are numbered migrations tracked in `PRAGMA user_version`. (modernc.org/sqlite was rejected: it crashed on Android x86_64.)
- Architecture: pure `core/` package for every rule in SPEC §4. UI and storage depend on core, never the other way round.
- Offline-first: the app works fully without a network. Sync comes in Phase 5.
- Storage (Phase 2): tables per record type, times as fixed-width UTC text, type-specific fields and lists as JSON. UI writes through `store.Store` (`Load`, `ReplaceAll`, `Update` for one transaction). Posters are JPEG files (long edge ≤ 600 px) in a `posters/` folder beside the database; entries hold the file name. Imported entries and categories keep their prototype ids.
- App shell (Phase 3 step 1): `ui.App` owns the window: tab, sub-page stack, dialog, toast. Screens implement `ui.Screen`; a page with its own search implements `ui.BackHandler`. Back order: dialog → page search → sub-page → Home → system (Android leaves the app); Escape does the same on Windows. Theme changes go through `App.SetTheme`, which saves the profile. Icons are the prototype's SVG markup, parsed by `ui.IconSVG`.
- Library (Phase 3 step 2): category and folder pages are sub-pages that keep the bottom bar (prototype behaviour, confirmed by the owner). Grids measure cells on scratch ops (`gridRow`) so nothing drawn while measuring reaches the screen. Pop-up menus use `op.Defer` to sit over neighbouring cards. Cards (`card`) subtract their padding from the minimum size, since `layout.Inset` doesn't.
- Entry form and page (Phase 3 step 3): controls the prototype got from the browser are drawn by us: dropdowns (`dropdown`, the ☰-menu look the owner chose), date fields with a month calendar dialog, the system file picker for posters (gioui.org/x/explorer, needs `explorer.ListenEvents` in the window loop). The bottom sheet (`App.ShowSheet`) sits between the page and dialogs; back closes dialog → sheet → page search → sub-page. Translucent overlays that must match the browser are pre-mixed in sRGB (`shade`, the hero backdrop).
- Symbols (★ ☆ ♥ ✓ ✕ ✦ ✧) come from a bundled 3 KB subset of Noto Sans Symbols 2 (OFL), with its line spacing reduced so it never makes a line taller. Colour emoji (🔥, 🎨) aren't drawable as text in Gio; `ui.Emoji` draws bundled Noto Color Emoji PNGs (`assets/emoji`, OFL). The streak pill uses the colour 🔥 image (owner's choice).
- Gio blends colours in linear light, browsers in sRGB: translucent black (scrims, shadows) goes through `shade()` so it darkens like the prototype's `rgba(0,0,0,a)`.
- Home (Phase 3 step 4): "see all" pages share `posterPage` (scoped search, 3 or 4 cards per row). Ongoing cards in a row share one height. The Consumed chip opens `consumedPage` (step 7).
- Favorites (Phase 3 step 5): tabs list every category. Favourite and Top 5 changes go through `toggleFavourite` / `addTopFive` / `removeTopFive` (save + toast), shared by the entry page and the Favorites tab. Where CSS margins collapse (e.g. the meta line above a divider), use the larger margin, not the sum.
- Digest (Phase 3 step 6): the narrative and "Still going" wording live in core (`DigestNarrative`, `ProgressSummary`). `card()` draws a 1dp border inside its size: for a CSS box with a border use inset = CSS padding; for one without, padding − 1. Single-line labels that must match CSS height set `LineHeight: normalLineHeight[face]`. CSS weight 600 on DM Sans or Lora means the browser's fake bold of Medium: use `Text{FakeBold: true}` with Medium, not a heavier font.
- Year in Review (Phase 3 step 8): the PNG is drawn by `yearcard/` in plain Go (no GPU), so it's identical on every platform and testable. Saving files goes through `App.CreateFile` (x/explorer). Gio has no share sheet, so `share/` has native code (Phase 4): Android (`share_android.java`, compiled into the committed `share_android.jar`, which gogio adds to the APK) stores the PNG in Pictures/Relic through the media store, the only way to hand a file to other apps without a manifest file provider, then opens the chooser (Android 10+); iOS (Objective-C in `share_ios.go`) opens `UIActivityViewController` on a temporary file. The UI sees it as `App.ShareImage` (nil on Windows/Linux, so no Share button there). Pass `share.ListenEvents` every window event, as for x/explorer.
- Posters on phones (Phase 4): `photos/` opens the photo library, not the file picker (owner's request, 2026-10-30): Android's photo picker on Android 13+ or the gallery (`ACTION_PICK`) before that (`photos_android.java` → committed `photos_android.jar`; formats Go can't read, such as HEIC, are re-encoded as JPEG in Java), and iOS's `PHPickerViewController` (`photos_ios.m`; redrawn upright as JPEG). The UI sees it as `App.PickPhoto`; without it (Windows) posters come through `App.ChooseFile`. `store.ResizePoster` turns camera photos upright from their EXIF orientation.
- App icon (owner's icon pack, 2026-10-30): `cmd/relic/appicon.png` (gogio's default; Android, where launchers crop to a circle or squircle, so it's the pack's padded adaptive foreground on the linen background) and `cmd/relic/appicon-ios.png` (the pack's full App Store icon; CI's iOS builds pass `-icon`).
- Text wraps only between words (`Paragraph` uses `text.WrapWords`), as browsers do; a box whose CSS gives a long word room to overflow into its padding gets that width in Gio, since Gio clips text to its width (category names).
- Bottom bar on phones: the prototype's 14px bottom padding becomes 4dp when the system has a bottom inset (owner: the bar sat too high above Android's buttons).
- Launch splash (prototype `#splash`, missing until 2026-10-31): `ui/splash.go`, the framed rose on black (`assets/splash.png`, PNG because Go's WebP decoder lifts pure black to grey), pulsing 2.2s, shown 1.6s then faded 0.8s; it swallows touches and turns the system bars black while up. `cmd/relic` calls `App.ShowSplash` (not in the tour). Android 12+ still shows its own system splash (the app icon) for a moment first; apps can't remove that.
- Phone sizing (owner's choice, 2026-10-31): `cmd/relic` scales dp and sp on Android and iOS so the layout fills the screen width as on a 390dp phone (`fitWidth`, clamped 0.85–1.2): wider phones zoom in, narrower ones out. Windows and snapshots keep 1dp = 1dp, so their numbers still compare with the prototype.
- iPhone header (owner, 2026-10-31): pages drop 13dp of the top inset on iOS (`App.TopTrim`), since iOS's safe area already leaves room under the status bar; Android keeps the full inset plus the top bar's 13dp (owner: "on point").
- Time zone on phones (2026-10-31): Go has no local zone on Android (UTC) and not always on iOS, so `tz.SetLocal` (called first in `cmd/relic`) asks the system (`java.util.TimeZone` / `NSTimeZone`) and sets `time.Local`; `time/tzdata` is built in. CI checks the log line `relic: time zone`. `App.Layout` also redraws at each new minute, so dates and "this week" stay current while the app sits open.
- Responsiveness (owner: taps "very slow and laggy", 2026-10-31): posters are decoded (as RGBA, for x/image's fast resize path) and resized on a background worker (`ui/images.go`): until ready a box shows a plain tile and the worker calls `Invalidate`, so no frame waits on image work. `posterSize` reads only the JPEG header. Snapshots, the tour and tests set `posters.sync`. SQLite runs `synchronous(NORMAL)` with WAL, so saves don't wait for the phone's storage to flush.
- Home's stat chips (owner, 2026-10-31): one row that exactly fills the width, no sideways scroll (`fitChips`): natural widths plus an equal share of spare room; on narrow screens the side padding shrinks (14 → 6dp), then the text (to 70%).
- Window insets: `cmd/relic` builds the frame's `layout.Context` for the whole window itself; `app.NewContext` would already offset and shrink by the insets, and `App.Layout` applies them (so the bars' space would count twice).
- Settings & archive (Phase 3 step 9): `archive/` is the export/import format (versioned JSON of the core library plus poster bytes; old prototype exports go through `importer/`). Import and Clear rebuild every page (`App.restart`). First launch shows onboarding (`App.StartOnboarding`), then Home with the welcome card (`ui/welcome.go`; owner's choice over the prototype's New category pop-up, 2026-10-31), kept by `Profile.ShowWelcome` (schema migration 3, not exported in the archive) until "Got it". Files come in through `App.ChooseFile` and go out through `App.CreateFile`.
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
- **No gradients, no busy hero sections.** Heroes stay slim and simple. (Agreed exceptions: the entry page's poster hero keeps the prototype's blurred backdrop and fade; the Year digest's month bars keep its gradient.)
- Match the prototype's spacing and card sizes. Don't resize poster cards or change layouts unasked.
- Bundle the fonts: Playfair Display, Lora and DM Sans (all OFL licensed).
- When unsure about a visual choice, show options before building one.

## Prototype quirks NOT to copy

- The prototype synthesised a missing "first session" on the fly and wrote to storage while rendering. In Go, the start session is a normal stored record, and repairing old data happens once in `importer/` (SPEC §9).
- It bucketed dates in UTC.
- It stored posters as full-size inline data.
- Its import didn't restore the theme.

## Working with the owner

- The owner has a **3.8 GB RAM** laptop, so the local Android emulator isn't usable. They have an **Android phone and an iPhone** for checking the UI by hand (since 2026-10-30), and a macOS virtual machine on their PC kept as a backup only. Day to day, check screens in the Windows build (phone-sized window) and `cmd/snapshot` PNGs. The Android and iOS checks are the GitHub Actions run (`.github/workflows/build.yml`), which builds the APK and the iOS simulator app, runs each on a cloud emulator/simulator and uploads screenshots of every screen (the tour). iOS work goes the GitHub route (owner's decision, 2026-10-30); no paid Apple Developer account is needed for the simulator.
- The owner edits in **VS Code**, not Android Studio. Don't give instructions that need the Android Studio app; use command-line tools (`sdkmanager`, `avdmanager`, `emulator`, `adb`, `gogio`) and VS Code.
- All work is saved to a GitHub repo (public since 2026-10-06) (`origin` → github.com/saniyaa61/Relic). **Commit and push after each piece of finished work**, with `go vet ./...` and `go test ./...` passing first. **Push straight to `main`; no pull requests** unless the owner asks for one (owner's decision, 2026-10-06).
- **After every finished step or phase, give the owner a Windows check:** what to pull (`git checkout main`, `git pull`), what to run (`go run ./cmd/relic`), and a short list of exactly what to try, with what should happen. Wait for their result before moving far ahead on anything it could change.
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
- Run on Windows: `go run ./cmd/relic`. Real data: Home → gear → Settings → Import archive, pick `relic-archive.json` (a prototype export or the app's own; replaces the app's data, keeps light/dark).
- Screen to PNG: `go run ./cmd/snapshot -screen home -o build/home.png` (add `-theme midnight -dark`; `-now 2026-09-29T09:00:00Z` fixes the clock; `-archive importer/testdata/scrubbed-archive.json` for data; `-screen` names are listed in `ui.SnapshotScreens`, e.g. home, home-empty, home-memory, swy, rf, ae, favorites, fav-series, fav-rank, digest, digest-week, digest-year, moods, consumed, consumed-folder, yearcard, settings, settings-custom, onboarding, library, lib-category, lib-folder, prompt, toast)
- The Year card PNG from the test archive: `YEARCARD_OUT=build/card.png go test ./yearcard -run TestRender`
- Import the prototype archive into a database and print counts, time and Top 5: `go run ./cmd/import -in ../relic-archive.json -db build/relic.db` (posters go to `build/posters/`).
- Prototype screenshots to compare with (cloud sessions; uses the pre-installed Chromium): `ARCHIVE=importer/testdata/scrubbed-archive.json node tools/protoshot.js $PWD build/proto library` (shot sets: shell, library, form, detail, home, favorites, digest, consumed, yearcard, settings; `NOW=2026-09-29T09:00:00Z` fixes the clock; `HEIGHT=1900` for long pages, `MODE=dark`, `THEME=midnight`)
- Check every number the app shows against the prototype's own code (cloud sessions; Chromium): `bash tools/check-numbers.sh [archive] [times...]` runs `tools/protonumbers.js` (the prototype in Chromium, UTC) and `go run ./cmd/numbers` (the app's core) at several clock times and prints the differences (`tools/compare-numbers.py`).
- Compare an import with the prototype's own formulas: `go test ./importer -run Prototype -v` (scrubbed copy); add `RELIC_ARCHIVE=C:/Users/saniy/Downloads/relic-archive.json` to also check the real archive.
- Refresh the scrubbed test copy after a new export: `go run ./cmd/scrub -in ../relic-archive.json -out importer/testdata/scrubbed-archive.json`. Never commit the real archive.
- Android APK locally (needs `ANDROID_HOME`, the NDK, and JDK 17's `bin` on PATH for `keytool`): `gogio -target android -arch arm64,amd64 -minsdk 24 -appid com.saniyaa61.relic -name Relic -o build/relic.apk ./cmd/relic`
- Android on a phone: download `relic-apk` from the latest green run's Artifacts, unzip, open `relic.apk` on the phone. Signed with the owner's fixed key (repo secrets, see STATUS CI notes), so updates install over the old version and keep the data.
- Android on an emulator: push to GitHub; the **Build** workflow uploads `relic-apk` and `emulator-screenshots` (first launch, plus `tour/` with every screen in Linen light and Midnight dark from the tour build) as run artifacts. The repo is public (free Actions minutes).
- iOS: needs a Mac (Phase 4).
