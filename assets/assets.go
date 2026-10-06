// Package assets bundles the files the app ships with: fonts and the logo.
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
