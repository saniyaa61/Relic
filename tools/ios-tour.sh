#!/usr/bin/env bash
# Runs the tour app on a booted iOS simulator (CI) and screenshots each
# screen as the app logs it ("relic: tour <n>/<total> <screen> <theme> <mode>").
#   bash tools/ios-tour.sh <udid> <Relic.app> <app id> <out dir>
set -u
udid=$1 app=$2 id=$3 out=$4
mkdir -p "$out"
log="$out/console.txt"
xcrun simctl terminate "$udid" "$id" >/dev/null 2>&1 || true
xcrun simctl install "$udid" "$app"
xcrun simctl launch --console-pty "$udid" "$id" > "$log" 2>&1 &

line() { grep -ao "relic: tour $1/[0-9]* [a-z0-9-]* [a-z]* [a-z]*" "$log" | tail -1; }

# Wait (up to 4 minutes) for the first screen.
for _ in $(seq 120); do [ -n "$(line 1)" ] && break; sleep 2; done
first=$(line 1)
if [ -z "$first" ]; then echo "tour never started"; cat "$log"; exit 1; fi
total=$(echo "$first" | sed 's|.*tour 1/\([0-9]*\).*|\1|')
echo "tour: $total screens"
# The app holds each screen until told the screenshot is taken: it logs
# the file to write the step number to (cmd/relic/tour.go, tourWait).
next=$(grep -ao "relic: tour waits on .*" "$log" | tail -1 | sed 's|relic: tour waits on ||' | tr -d '\r')
if [ -z "$next" ]; then echo "the app didn't say where to signal"; cat "$log"; exit 1; fi

missing=0
for n in $(seq 1 "$total"); do
  l=""
  # Up to 2 minutes: the simulator can stall for half a minute (the first
  # keyboard, for one).
  for _ in $(seq 240); do l=$(line "$n"); [ -n "$l" ] && break; sleep 0.5; done
  if [ -z "$l" ]; then
    echo "screen $n never appeared"
    missing=$((missing + 1))
    # Three in a row means the tour has stalled: stop rather than wait out the rest.
    [ "$missing" -ge 3 ] && break
    continue
  fi
  missing=0
  sleep 1.5
  name=$(echo "$l" | awk '{print $4"-"$5"-"$6}')
  # The simulator can take a while to bring the app to the front at first
  # (screenshots once showed the home screen).
  [ "$n" -le 2 ] && sleep 8
  # The system share sheet takes a few seconds to slide up.
  case "$name" in *-sheet-*) sleep 6 ;; esac
  xcrun simctl io "$udid" screenshot "$out/$(printf %02d "$n")-$name.png" >/dev/null 2>&1
  echo "$l"
  echo "$n" > "$next"
done

for _ in $(seq 30); do grep -q "relic: tour done" "$log" && break; sleep 1; done
grep -q "relic: tour done" "$log" || { echo "tour didn't finish"; cat "$log"; exit 1; }
# A screen that couldn't be shown logs "relic: tour <screen>: <error>";
# the last step's Share button logs "relic: year card share: <error>".
if grep -aE "relic: tour [a-z0-9-]+: |relic: year card share: " "$log"; then exit 1; fi
xcrun simctl spawn "$udid" launchctl list | grep -q "$id" || { echo "the app crashed"; exit 1; }
xcrun simctl terminate "$udid" "$id" || true
