// Package store persists the core model behind an interface the UI uses.
// It uses SQLite through github.com/ncruces/go-sqlite3, a build of SQLite
// translated to pure Go. It needs no C toolchain and does its file I/O
// through Go's os package, so it runs inside Android's syscall filter.
// (modernc.org/sqlite was tried first: it crashed on Android x86_64 because
// its libc issues lstat, which Android's seccomp policy forbids.)
package store
