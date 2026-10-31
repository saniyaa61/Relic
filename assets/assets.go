// Package assets bundles the files the app ships with: fonts, the logo and
// the launch picture.
package assets

import "embed"

// Fonts holds the bundled OFL fonts: Playfair Display, Lora and DM Sans,
// plus a few symbols from Noto Sans Symbols 2.
//
//go:embed fonts/*.ttf
var Fonts embed.FS

// Logo is the Relic mark (200×341 WebP), copied byte for byte from the
// prototype's embedded logo (window._RL in reference/relic.html).
//
//go:embed logo.webp
var Logo []byte

// Splash is the launch picture (400×600 PNG): the framed rose and RELIC on
// black, the prototype's #splash WebP decoded once by a browser-grade
// decoder (Go's WebP decoder lifts its pure blacks to grey).
//
//go:embed splash.png
var Splash []byte

// Emoji holds the few colour emoji the app shows (🔥, 🎨) as PNGs from
// Noto Color Emoji (OFL), since Gio can't draw colour emoji from a font.
//
//go:embed emoji/*.png
var Emoji embed.FS
