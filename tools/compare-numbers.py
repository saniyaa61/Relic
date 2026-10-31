"""Compares the prototype's numbers (tools/protonumbers.js) with the app's
(cmd/numbers): prints every difference, or "all match".

    python3 tools/compare-numbers.py build/proto-numbers.json build/app-numbers.json
"""
import json
import sys

proto = json.load(open(sys.argv[1]))
app = json.load(open(sys.argv[2]))
diffs = []


def same(a, b):
    if isinstance(a, (int, float)) and isinstance(b, (int, float)) and not isinstance(a, bool):
        return abs(a - b) < 1e-6
    return a == b


def walk(p, a, path):
    if isinstance(p, dict):
        for k in p:
            if k not in a:
                diffs.append(f"{path}.{k}: prototype has it, the app doesn't")
                continue
            walk(p[k], a[k], f"{path}.{k}")
        for k in a:
            if k not in p and path.startswith((".months", ".years", ".entries")) and path.count(".") <= 1:
                diffs.append(f"{path}.{k}: the app has it, the prototype doesn't")
    elif isinstance(p, list):
        if len(p) != len(a or []):
            diffs.append(f"{path}: prototype {p}, app {a}")
            return
        for i, (x, y) in enumerate(zip(p, a)):
            walk(x, y, f"{path}[{i}]")
    elif p is None:
        return  # the prototype doesn't track it for this entry type
    elif not same(p, a):
        diffs.append(f"{path}: prototype {p!r}, app {a!r}")


walk(proto, app, "")
print("\n".join(diffs) if diffs else "all match")
print(f"({len(diffs)} differences)", file=sys.stderr)
