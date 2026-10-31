#!/usr/bin/env bash
# Builds a copy of gogio that, with GOGIO_NOSIGN=1, skips signing an iOS
# .ipa (CI only). gogio insists on signing device builds with a
# provisioning profile from a Mac's keychain; an unsigned .ipa is what
# Windows sideloading tools (Sideloadly) take, signing it with the
# owner's free Apple ID as they install it.
#   bash tools/gogio-nosign.sh <output binary>
set -eu
out=$1
go install gioui.org/cmd/gogio@v0.10.0
src=$(go env GOMODCACHE)/gioui.org/cmd@v0.10.0
work=$(mktemp -d)
cp -R "$src"/. "$work"
chmod -R u+w "$work"
f="$work/gogio/iosbuild.go"
old='if err := signApple(bi.appID, tmpDir, embedded, appDir, provisions); err != nil {'
new='if err := signUnlessSkipped(bi.appID, tmpDir, embedded, appDir, provisions); err != nil {'
grep -qF "$old" "$f" || { echo "gogio changed: the signing call wasn't found"; exit 1; }
sed -i.bak "s|$old|$new|" "$f"
cat >> "$f" <<'GO'

func signUnlessSkipped(appID, tmpDir, embedded, app string, provisions []string) error {
	if os.Getenv("GOGIO_NOSIGN") != "" {
		return nil
	}
	return signApple(appID, tmpDir, embedded, app, provisions)
}
GO
(cd "$work" && go build -o "$out" ./gogio)
echo "built $out"
