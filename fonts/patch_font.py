#!/usr/bin/env python3
"""Add cmap aliases to RobotoMono Nerd Font so powerline-go's default symbols render.

RobotoMono Nerd Font carries the Nerd Fonts private-use icons (U+E0B0 separator,
U+E0A0 branch, ...) but not the plain-Unicode symbols powerline-go uses by default
(U+21E1 ahead, U+270E notStaged, ...). Terminals with font fallback (Windows
Terminal via DirectWrite) paper over the gap; MobaXterm does no fallback, so those
glyphs simply draw blank.

The fix needs no donor font and no outline scaling: for each missing codepoint,
point it at an icon the font already contains, by adding an entry to every Unicode
cmap subtable. Family/full/PostScript names are preserved so Windows treats the
result as an in-place update of the same font, not a new family.

Usage:
    python3 patch_font.py                       # discover installed fonts -> ./patched
    python3 patch_font.py --out /tmp/out
    python3 patch_font.py --src DIR --glob 'RobotoMonoNerdFont-*.ttf'   # non-Mono family
    python3 patch_font.py --dry-run

Requires fontTools (pip install fonttools).
"""

import argparse
import glob
import os
import sys

from fontTools.ttLib import TTFont

# Marker appended to the version / unique-id name records so a patched font is
# identifiable, and so re-running this script is a no-op instead of a corruption.
MARKER = "pwl-patch"

# Where fonts tend to live. Windows per-user installs come first: that is where
# "Install for me only" from Explorer puts them.
DEFAULT_SRC_GLOBS = [
    "/mnt/c/Users/*/AppData/Local/Microsoft/Windows/Fonts",
    "/mnt/c/Windows/Fonts",
    os.path.expanduser("~/.local/share/fonts"),
    os.path.expanduser("~/.fonts"),
]

# powerline-go default symbol  ->  an icon the font already has.
# Left column is from defaults.go (the "patched" mode block); right column is a
# Nerd Fonts / Font Awesome private-use codepoint present in RobotoMono NF.
ALIAS = {
    0x21E1: 0xF062,  # ⇡ RepoAhead      -> fa-arrow-up
    0x21E3: 0xF063,  # ⇣ RepoBehind     -> fa-arrow-down
    0x2713: 0xF00C,  # ✓ RepoStaged     -> fa-check
    0x270E: 0xF040,  # ✎ RepoNotStaged  -> fa-pencil
    0x273C: 0xF071,  # ✼ RepoConflicted -> fa-warning
    0x2691: 0xF024,  # ⚑ RepoStashed    -> fa-flag
    0x2693: 0xF13D,  # ⚓ RepoDetached   -> fa-anchor
    0x2235: 0xF15B,  # ∵ DotEnvIndicator-> fa-file
    0x260E: 0xF095,  # ☎ Network        -> fa-phone
    0x2388: 0xF085,  # ⎈ KubeIndicator  -> fa-gears
}


def unicode_cmap(font):
    """Merge every Unicode cmap subtable into one codepoint -> glyph-name dict."""
    merged = {}
    for table in font["cmap"].tables:
        if table.isUnicode():
            merged.update(table.cmap)
    return merged


def already_patched(font):
    for record in font["name"].names:
        if record.nameID in (3, 5):
            try:
                if MARKER in record.toUnicode():
                    return True
            except UnicodeDecodeError:
                continue
    return False


def stamp_names(font):
    """Mark the font as patched without touching family/full/PostScript names.

    nameID 3 is the unique identifier and 5 the version string; Windows keys its
    "is this an update?" decision off those plus head.fontRevision.
    """
    name = font["name"]
    for record in list(name.names):
        if record.nameID not in (3, 5):
            continue
        try:
            value = record.toUnicode()
        except UnicodeDecodeError:
            continue
        suffix = ";" + MARKER if record.nameID == 3 else " (" + MARKER + ")"
        name.setName(
            value + suffix,
            record.nameID,
            record.platformID,
            record.platEncID,
            record.langID,
        )
    font["head"].fontRevision += 0.001


def discover_source(name_glob):
    """First candidate directory that actually holds a matching font.

    Requiring a match matters: several candidates commonly exist (a Windows
    per-user font dir and the machine-wide one), and returning the first that
    merely exists would report "no files matching" while another candidate held
    the fonts all along.
    """
    for pattern in DEFAULT_SRC_GLOBS:
        for path in sorted(glob.glob(pattern)):
            if os.path.isdir(path) and glob.glob(os.path.join(path, name_glob)):
                return path
    return None


def patch_file(path, outdir, dry_run):
    """Alias missing codepoints in one font. Returns (status, alias_count)."""
    font = TTFont(path)
    basename = os.path.basename(path)

    if already_patched(font):
        return "skipped (already patched)", 0

    cmap = unicode_cmap(font)
    added = 0
    for source_cp, target_cp in ALIAS.items():
        if source_cp in cmap:
            continue  # font has a real glyph for it; leave well alone
        glyph_name = cmap.get(target_cp)
        if not glyph_name:
            continue  # no icon to borrow, nothing sensible to do
        for table in font["cmap"].tables:
            if table.isUnicode():
                table.cmap[source_cp] = glyph_name
        added += 1

    if added == 0:
        return "unchanged (nothing missing)", 0

    if dry_run:
        return "would patch", added

    stamp_names(font)
    font.save(os.path.join(outdir, basename))
    return "patched", added


def main():
    parser = argparse.ArgumentParser(
        description="Alias powerline-go's default symbols onto icons RobotoMono NF already has."
    )
    parser.add_argument("--src", help="directory holding the source .ttf files")
    parser.add_argument(
        "--glob",
        default="RobotoMonoNerdFontMono-*.ttf",
        help="filename pattern within --src (default: the Mono family)",
    )
    parser.add_argument(
        "--out",
        default=os.path.join(os.getcwd(), "patched"),
        help="output directory (default: ./patched)",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="report what would change without writing anything",
    )
    args = parser.parse_args()

    src = args.src or discover_source(args.glob)
    if not src or not os.path.isdir(src):
        sys.exit(
            "No directory containing %r found. Pass --src explicitly; searched:\n  "
            % args.glob
            + "\n  ".join(DEFAULT_SRC_GLOBS)
        )

    files = sorted(glob.glob(os.path.join(src, args.glob)))
    if not files:
        sys.exit("No files matching %r in %s" % (args.glob, src))

    if not args.dry_run:
        os.makedirs(args.out, exist_ok=True)

    print("source: %s" % src)
    print("output: %s%s\n" % (args.out, " (dry run)" if args.dry_run else ""))

    written = 0
    for path in files:
        status, added = patch_file(path, args.out, args.dry_run)
        if status in ("patched", "would patch"):
            written += 1
        print("  %-46s %+d aliases  %s" % (os.path.basename(path), added, status))

    print(
        "\n%d/%d file(s) %s."
        % (written, len(files), "would change" if args.dry_run else "written")
    )
    if written and not args.dry_run:
        print("Verify with: python3 verify_font.py --dir %s" % args.out)


if __name__ == "__main__":
    main()
