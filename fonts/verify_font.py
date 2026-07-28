#!/usr/bin/env python3
"""Report whether a font can render every symbol powerline-go's "patched" mode uses.

Loads each font fully (which also validates its structure, since a font that survives
a non-lazy TTFont load is not corrupt) and checks its Unicode cmaps against the symbol
set in defaults.go. Any codepoint reported MISSING will draw blank in a terminal
that does no font fallback, such as MobaXterm.

Usage:
    python3 verify_font.py                              # the installed Mono family
    python3 verify_font.py --dir ./patched
    python3 verify_font.py --file /path/to/one.ttf
"""

import argparse
import glob
import os
import sys

from fontTools.ttLib import TTFont

# Every symbol in the "patched" mode block of defaults.go, plus the module that
# renders it. Codepoints in the private-use area ship with Nerd Fonts; the plain
# Unicode ones are what RobotoMono lacks upstream.
SYMBOLS = [
    (0xE0B0, "Separator", "cwd/host/user"),
    (0xE0B1, "SeparatorThin", "cwd"),
    (0xE0B2, "SeparatorReverse", "right prompt"),
    (0xE0B3, "SeparatorReverseThin", "right prompt"),
    (0xE0A0, "RepoBranch", "git"),
    (0xE0A2, "Lock", "cwd/root"),
    (0x2693, "RepoDetached", "git"),
    (0x21E1, "RepoAhead", "git"),
    (0x21E3, "RepoBehind", "git"),
    (0x2713, "RepoStaged", "git"),
    (0x270E, "RepoNotStaged", "git"),
    (0x273C, "RepoConflicted", "git"),
    (0x2691, "RepoStashed", "git"),
    (0x2235, "DotEnvIndicator", "dotenv"),
    (0x260E, "Network", "host"),
    (0xE235, "VenvIndicator", "venv"),
    (0x2388, "KubeIndicator", "kube"),
    (0xF313, "NixShellIndicator", "nix-shell"),
    (0x2B22, "NodeIndicator", "node"),
    (0xE92B, "RvmIndicator", "rvm"),
]

# Symbols RobotoMono Nerd Font has no sensible stand-in icon for, so patch_font.py
# deliberately leaves them unmapped. They are reported but do not fail the run;
# anything else missing is a real failure. See README.md.
KNOWN_GAPS = {0x2B22, 0xE92B}

DISCOVER_GLOBS = [
    "/mnt/c/Users/*/AppData/Local/Microsoft/Windows/Fonts/RobotoMonoNerdFontMono-Regular.ttf",
    "/mnt/c/Windows/Fonts/RobotoMonoNerdFontMono-Regular.ttf",
]


def unicode_cmap(font):
    merged = {}
    for table in font["cmap"].tables:
        if table.isUnicode():
            merged.update(table.cmap)
    return merged


def check(path):
    """Print a coverage report for one font. Returns the list of missing codepoints."""
    font = TTFont(path)
    cmap = unicode_cmap(font)
    name = font["name"]

    print("=" * 78)
    print(os.path.basename(path))
    print("  family  %s" % name.getDebugName(1))
    print("  version %s" % name.getDebugName(5))
    print()

    missing = []
    for codepoint, field, module in SYMBOLS:
        glyph = cmap.get(codepoint)
        if glyph:
            print(
                "  U+%04X  %-22s %-14s renders (%s)" % (codepoint, field, module, glyph)
            )
        else:
            missing.append((codepoint, field, module))
            note = "MISSING (known gap)" if codepoint in KNOWN_GAPS else "MISSING"
            print("  U+%04X  %-22s %-14s %s" % (codepoint, field, module, note))
    return missing


def main():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--dir", help="check every .ttf in this directory")
    parser.add_argument("--file", help="check a single .ttf")
    args = parser.parse_args()

    if args.file:
        paths = [args.file]
    elif args.dir:
        paths = sorted(glob.glob(os.path.join(args.dir, "*.ttf")))
        if not paths:
            sys.exit("No .ttf files in %s" % args.dir)
    else:
        paths = []
        for pattern in DISCOVER_GLOBS:
            paths = sorted(glob.glob(pattern))
            if paths:
                break
        if not paths:
            sys.exit(
                "Could not find an installed RobotoMono NFM; pass --dir or --file."
            )

    all_missing = set()
    for path in paths:
        for codepoint, field, module in check(path):
            all_missing.add((codepoint, field, module))

    print("=" * 78)
    gaps = sorted(m for m in all_missing if m[0] in KNOWN_GAPS)
    unexpected = sorted(m for m in all_missing if m[0] not in KNOWN_GAPS)

    if gaps:
        print("%d known gap(s), documented and deliberately not patched:" % len(gaps))
        for codepoint, field, module in gaps:
            print("  U+%04X %s (module: %s)" % (codepoint, field, module))
        print()

    if not unexpected:
        print(
            "All %d patchable symbol(s) render in %d file(s)."
            % (len(SYMBOLS) - len(gaps), len(paths))
        )
        return

    print(
        "%d symbol(s) unexpectedly missing across %d file(s):"
        % (len(unexpected), len(paths))
    )
    for codepoint, field, module in unexpected:
        print("  U+%04X %s (module: %s)" % (codepoint, field, module))
    print("\nThese draw blank in terminals without font fallback. See README.md.")
    raise SystemExit(1)


if __name__ == "__main__":
    main()
