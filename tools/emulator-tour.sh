#!/usr/bin/env bash
# Runs the tour APK on the emulator (CI) and screenshots each screen as the
# app logs it ("relic: tour <n>/<total> <screen> <theme> <mode>").
#   bash tools/emulator-tour.sh <apk> <app id> <out dir>
set -u
apk=$1 id=$2 out=$3
mkdir -p "$out"
adb uninstall "$id" >/dev/null 2>&1 || true
adb install -r "$apk"
adb logcat -c
adb shell monkey -p "$id" -c android.intent.category.LAUNCHER 1 >/dev/null

line() { adb logcat -d | grep -o "relic: tour $1/[0-9]* [a-z0-9-]* [a-z]* [a-z]*" | tail -1; }

# Wait (up to 4 minutes) for the first screen.
for _ in $(seq 120); do [ -n "$(line 1)" ] && break; sleep 2; done
first=$(line 1)
if [ -z "$first" ]; then echo "tour never started"; adb logcat -d > "$out/logcat.txt"; exit 1; fi
total=$(echo "$first" | sed 's|.*tour 1/\([0-9]*\).*|\1|')
echo "tour: $total screens"

for n in $(seq 1 "$total"); do
  l=""
  for _ in $(seq 40); do l=$(line "$n"); [ -n "$l" ] && break; sleep 0.5; done
  [ -z "$l" ] && { echo "screen $n never appeared"; continue; }
  sleep 1.5
  name=$(echo "$l" | awk '{print $4"-"$5"-"$6}')
  adb exec-out screencap -p > "$out/$(printf %02d "$n")-$name.png"
  echo "$l"
done

for _ in $(seq 30); do adb logcat -d | grep -q "relic: tour done" && break; sleep 1; done
adb logcat -d > "$out/logcat.txt"
grep -q "relic: tour done" "$out/logcat.txt" || { echo "tour didn't finish"; exit 1; }
# A screen that couldn't be shown logs "relic: tour <screen>: <error>".
if grep -E "relic: tour [a-z0-9-]+: " "$out/logcat.txt"; then exit 1; fi
adb shell pidof "$id" >/dev/null || { echo "the app crashed"; exit 1; }
