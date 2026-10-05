// Package store persists the core model behind an interface the UI uses.
// It uses SQLite through modernc.org/sqlite, a pure-Go build of SQLite, so
// the app compiles for Android and iOS without a C toolchain.
package store
