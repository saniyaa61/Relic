#!/usr/bin/env bash
# Compares every number the app shows with the prototype's own code, for an
# archive at several clock times (cloud sessions: needs the pre-installed
# Chromium). Prints the differences per time.
#   bash tools/check-numbers.sh [archive] [times...]
set -eu
archive=${1:-importer/testdata/scrubbed-archive.json}
shift || true
times=${*:-"2026-06-07T08:00:00Z 2026-06-30T23:30:00Z 2026-07-15T12:00:00Z 2026-09-15T23:59:00Z 2026-09-29T09:00:00Z 2026-10-31T10:00:00Z 2027-01-01T00:30:00Z 2027-06-01T12:00:00Z"}
mkdir -p build/numbers
for t in $times; do
  ARCHIVE=$archive NOW=$t node tools/protonumbers.js "$PWD" > build/numbers/proto-$t.json
  go run ./cmd/numbers -archive "$archive" -now "$t" > build/numbers/app-$t.json
  echo "== $t"
  python3 tools/compare-numbers.py build/numbers/proto-$t.json build/numbers/app-$t.json 2>&1
done
