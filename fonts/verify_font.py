#!/usr/bin/env python3
"""Report whether a font can render every symbol powerline-go's prompt modes need.

Checks each font's Unicode cmaps against the symbols in defaults.go: the whole
"patched" mode block, plus the handful only "compatible" and "flat" spell differently.
Any codepoint reported MISSING will draw blank in a terminal that does no font
fallback, such as MobaXterm.

Decompiles every table rather than only the ones it reads, so a font whose outlines
are corrupt fails here instead of passing a cmap-only check. TTFont decompiles on
access by default (and lazy=False does not change that), so without this a destroyed
glyf table sails through.

Every symbol is expected to render; a single MISSING one exits non-zero, so this can
gate a regeneration or a CI step.

Usage:
    python3 verify_font.py                              # the installed Mono family
    python3 verify_font.py --dir ./patched
    python3 verify_font.py --file /path/to/one.ttf
"""

import argparse
import glob
import os
import sys

import fontTools
from fontTools.ttLib import TTFont

# ensureDecompiled landed in fontTools 4.31.0. Checked up front because the call
# sits inside a broad except that converts anything into CorruptFont, so on an
# older fontTools every font would be reported as damaged and the operator would
# conclude the shipped zip is broken rather than their toolchain too old.
if not hasattr(TTFont, "ensureDecompiled"):
    raise SystemExit(
        "fontTools >= 4.31.0 required for TTFont.ensureDecompiled; found %s"
        % fontTools.version
    )

# Every non-ASCII symbol defaults.go uses, plus the module that renders it.
# Codepoints in the private-use area ship with Nerd Fonts; the plain Unicode ones
# are what RobotoMono lacks upstream. Entries below the marker comment belong to
# the "compatible" and "flat" modes, which spell some symbols in plain Unicode
# where "patched" reaches for a Nerd Fonts icon.
#
# Everything here MUST render: there is no expected-gap allowlist, so only add a
# codepoint once the font can resolve it, either natively or via an ALIAS target
# in patch_font.py.
#
# Deliberately absent: U+1F433 SPOUTING WHALE, hardcoded in
# segment-docker_context.go. No mono Nerd Font carries an emoji for it, and
# aliasing it to a docker icon would be wrong anyway: the whale is East Asian
# Wide, so powerline-go's runewidth accounting reserves two cells while any Nerd
# Fonts icon draws one, misaligning the prompt. Override it in config.json
# instead. See README.md, "Known remaining gaps".
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
    # --- "compatible" and "flat" mode spellings from here down ---
    (0x25C6, "RvmIndicator", "rvm"),
    (0x25B6, "Separator", "cwd/host/user"),
    (0x25C0, "SeparatorReverse", "right prompt"),
    (0x2744, "NixShellIndicator", "nix-shell"),
    (0x276F, "SeparatorThin", "cwd"),
    (0x276E, "SeparatorReverseThin", "right prompt"),
    (0x03C0, "VenvIndicator", "venv"),
    # --- glyphs hardcoded in Go source rather than defaults.go ---
    (0x2026, "ellipsis", "cwd"),  # segment-cwd.go and powerline.go, truncation
    (0x00B5, "micro sign", "duration"),  # segment-duration.go, sub-ms
]

# The whole family, not just Regular: an install that silently skipped some styles,
# or left an older generation's Bold behind, has to show up as a failure here.
#
# Same glob and same directory order as patch_font.py's DEFAULT_SRC_GLOBS, so a bare
# patch_font.py run and the bare verify_font.py run the README pairs it with cover
# the same fonts. These glob files rather than directories, so on a multi-profile
# Windows box this unions every profile's copies where patch_font.py takes only the
# first directory that matches. Erring towards checking more is the safe direction
# for a verifier.
DISCOVER_GLOBS = [
    "/mnt/c/Users/*/AppData/Local/Microsoft/Windows/Fonts/RobotoMonoNerdFontMono-*.ttf",
    "/mnt/c/Windows/Fonts/RobotoMonoNerdFontMono-*.ttf",
    os.path.expanduser("~/.local/share/fonts/RobotoMonoNerdFontMono-*.ttf"),
    os.path.expanduser("~/.fonts/RobotoMonoNerdFontMono-*.ttf"),
]


class CorruptFont(Exception):
    """A font file that will not read, so its coverage cannot be judged."""


def unicode_cmap(font):
    merged = {}
    for table in font["cmap"].tables:
        if table.isUnicode():
            merged.update(table.cmap)
    return merged


def check(path):
    """Print a coverage report for one font.

    Returns a list of (codepoint, field, module) tuples for the symbols that will
    not render. Raises CorruptFont if the file will not read.
    """
    try:
        # Construction is inside the guard too: a truncated sfnt header raises
        # here, before ensureDecompiled gets a chance, and that must be reported
        # as one bad file rather than abandoning the rest of the directory.
        font = TTFont(path)
        # Not just the tables read below: TTFont decompiles on access, so a
        # corrupt glyf or loca would otherwise pass a cmap-only check unnoticed.
        font.ensureDecompiled()
        cmap = unicode_cmap(font)
        name = font["name"]
    except Exception as err:
        # Deliberately broad: fontTools raises whatever the malformed table's
        # decompiler happens to raise (IndexError, struct.error, KeyError, its
        # own TTLibError, ...), and any of them means the same thing here.
        raise CorruptFont(
            "%s: will not read (%s: %s)"
            % (os.path.basename(path), type(err).__name__, err)
        ) from err

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
            print("  U+%04X  %-22s %-14s MISSING" % (codepoint, field, module))
    return missing


def main():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    # Mutually exclusive: --file used to silently win over --dir, quietly checking
    # one font when the operator asked for twelve.
    target = parser.add_mutually_exclusive_group()
    target.add_argument("--dir", help="check every .ttf in this directory")
    target.add_argument("--file", help="check a single .ttf")
    args = parser.parse_args()

    if args.file:
        # Checked up front so a mistyped path reads as a missing file rather than
        # a corrupt font: FileNotFoundError would otherwise become CorruptFont.
        if not os.path.isfile(args.file):
            sys.exit("No such file: %s" % args.file)
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
    files_with_gaps = set()
    corrupt = []
    for path in paths:
        try:
            missing = check(path)
            if missing:
                files_with_gaps.add(path)
            all_missing.update(missing)
        except CorruptFont as err:
            # Keep going: reporting every bad file in one run beats making the
            # operator rediscover them one at a time.
            corrupt.append(str(err))
            print("=" * 78)
            print("%s\n  CORRUPT, coverage not checked" % os.path.basename(path))

    print("=" * 78)

    if corrupt:
        print("%d file(s) would not read:" % len(corrupt))
        for message in corrupt:
            print("  %s" % message)
        print()

    readable = len(paths) - len(corrupt)

    if not all_missing and not corrupt:
        print("All %d symbol(s) render in %d file(s)." % (len(SYMBOLS), len(paths)))
        return

    if not all_missing:
        # Corrupt files but full coverage everywhere readable. Say so, rather than
        # exiting 1 with the corrupt list as the only output and leaving the
        # operator to scroll back through every per-symbol report.
        if readable:
            print(
                "All %d symbol(s) render in the %d readable file(s)."
                % (len(SYMBOLS), readable)
            )
        raise SystemExit(1)

    print(
        "%d symbol(s) missing in %d of %d readable file(s):"
        % (len(all_missing), len(files_with_gaps), readable)
    )
    for codepoint, field, module in sorted(all_missing):
        print("  U+%04X %s (module: %s)" % (codepoint, field, module))
    print("\nThese draw blank in terminals without font fallback. See README.md.")
    raise SystemExit(1)


if __name__ == "__main__":
    main()
