#!/usr/bin/env bash
# Runs the tour APK on the emulator (CI) and screenshots each screen as the
# app logs it ("relic: tour <n>/<total> <screen> <theme> <mode>").
#   bash tools/emulator-tour.sh <apk> <app id> <out dir>
set -u
apk=$1 id=$2 out=$3
mkdir -p "$out"

# Install over the launch-check app. CI signs both APKs with the owner's
# key, so no uninstall is needed: uninstalling made Android run a late
# "package removed" cleanup that now and then closed the freshly started
# tour app. Without the key (the APKs' keys differ) uninstall, and give
# that cleanup time to finish.
if ! adb install -r "$apk"; then
  adb uninstall "$id" >/dev/null 2>&1 || true
  for _ in $(seq 30); do adb shell pm list packages | grep -q "$id" || break; sleep 1; done
  sleep 10
  adb install -r "$apk"
fi
sleep 5

line() { adb logcat -d | grep -o "relic: tour $1/[0-9]* [a-z0-9-]* [a-z]* [a-z]*" | tail -1; }

# tour launches the app and screenshots every screen. It returns 2 if the
# tour stalled (never started, or three screens in a row never appeared),
# so it can be tried once more.
tour() {
  adb shell am force-stop "$id"
  adb logcat -c
  adb shell monkey -p "$id" -c android.intent.category.LAUNCHER 1 >/dev/null

  # Wait (up to 4 minutes) for the first screen.
  for _ in $(seq 480); do [ -n "$(line 1)" ] && break; sleep 0.5; done
  first=$(line 1)
  if [ -z "$first" ]; then echo "tour never started"; return 2; fi
  total=$(echo "$first" | sed 's|.*tour 1/\([0-9]*\).*|\1|')
  echo "tour: $total screens"

  missing=0 late=0
  for n in $(seq 1 "$total"); do
    l=""
    for _ in $(seq 40); do l=$(line "$n"); [ -n "$l" ] && break; sleep 0.5; done
    if [ -z "$l" ]; then
      echo "screen $n never appeared"
      missing=$((missing + 1))
      [ "$missing" -ge 3 ] && return 2
      continue
    fi
    missing=0
    sleep 1.5
    # The screen must still be up when the screenshot starts (screencap
    # itself takes well under a second; each screen stays up 4 s).
    if [ "$n" -lt "$total" ] && [ -n "$(line $((n + 1)))" ]; then
      echo "screenshot $n started too late: screen $((n + 1)) was already up"
      late=1
    fi
    name=$(echo "$l" | awk '{print $4"-"$5"-"$6}')
    # The system share sheet takes a few seconds to slide up.
    case "$name" in *-sheet-*) sleep 6 ;; esac
    adb exec-out screencap -p > "$out/$(printf %02d "$n")-$name.png"
    echo "$l"
  done
  return 0
}

tour
status=$?
if [ "$status" = 2 ]; then
  adb logcat -d > "$out/logcat-stalled.txt"
  echo "the tour stalled; trying once more (log in logcat-stalled.txt)"
  rm -f "$out"/[0-9]*.png
  tour
  status=$?
fi
[ "$status" = 2 ] && { adb logcat -d > "$out/logcat.txt"; exit 1; }

for _ in $(seq 30); do adb logcat -d | grep -q "relic: tour done" && break; sleep 1; done
adb logcat -d > "$out/logcat.txt"
grep -q "relic: tour done" "$out/logcat.txt" || { echo "tour didn't finish"; exit 1; }
# A screen that couldn't be shown logs "relic: tour <screen>: <error>";
# the last step's Share button logs "relic: year card share: <error>".
if grep -E "relic: tour [a-z0-9-]+: |relic: year card share: " "$out/logcat.txt"; then exit 1; fi
adb shell pidof "$id" >/dev/null || { echo "the app crashed"; exit 1; }
[ "$late" = 0 ] || exit 1
