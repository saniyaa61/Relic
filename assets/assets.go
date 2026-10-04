// Package assets bundles the files the app ships with: fonts and icons.
package assets

import "embed"

// Fonts holds the bundled OFL fonts: Playfair Display, Lora and DM Sans.
//
//go:embed fonts/*.ttf
var Fonts embed.FS
