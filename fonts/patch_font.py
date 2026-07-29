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
    python3 patch_font.py --repatch             # re-alias an already-patched font
    python3 patch_font.py --min-revision 3.003  # regenerating from pristine originals
    python3 patch_font.py --dry-run

Requires fontTools (pip install fonttools).
"""

import argparse
import glob
import os
import re
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
# Left column is from defaults.go; right column is a Nerd Fonts / Font Awesome
# private-use codepoint present in RobotoMono NF. Covers every non-ASCII symbol
# all three modes use, so switching -mode never lands on a blank cell.
ALIAS = {
    # The "patched" mode block.
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
    0x2B22: 0xE718,  # ⬢ NodeIndicator  -> dev-nodejs_small (Node's hexagon logo)
    0xE92B: 0xE739,  #   RvmIndicator   -> dev-ruby
    # Symbols only the "compatible" and "flat" modes use. Those modes exist to
    # avoid needing a patched font at all, so they spell everything in plain
    # Unicode -- which RobotoMono lacks just as thoroughly as the rest.
    0x25C6: 0xF070B,  # ◆ RvmIndicator      -> md-rhombus (a faithful diamond)
    0x25B6: 0xE0B0,  # ▶ Separator         -> pl-left_hard_divider
    0x25C0: 0xE0B2,  # ◀ SeparatorReverse  -> pl-right_hard_divider
    0x2744: 0xF313,  # ❄ NixShellIndicator -> linux-nixos (as "patched" uses)
}

# The number in the nameID 5 version string, e.g. the "3.000" of
# "Version 3.000;Nerd Fonts 3.4.0". Deliberately anchored on the "Version "
# prefix so the Nerd Fonts version later in the same string is left alone.
VERSION_NUMBER = re.compile(r"(?<=Version )\d+\.\d+")

# head.fontRevision compiles to a signed 16.16 fixed-point field.
FONT_REVISION_MAX = 32767.0


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


def next_revision(font, basename, min_revision=0.0):
    """The head.fontRevision this patch would write, validated against the field.

    Separate from stamp_names so --dry-run can run the same arithmetic and the
    same ceiling check without touching the font, rather than promising a patch
    the real run would refuse.
    """
    # Round both: fontRevision round-trips through a 16.16 fixed-point field, so a
    # stored 3.001 reads back as 3.00100708 and would drift on every re-patch. The
    # floor needs it too, or a fractional --min-revision desynchronises the version
    # string from head (3.0035 -> head 3.005, string "3.004").
    bumped = max(round(font["head"].fontRevision, 3), round(min_revision, 3)) + 0.001
    # --min-revision is bounds-checked in main(), but the source font's own
    # revision is not, and it is the sum that has to fit the field. Fail here with
    # something readable rather than a struct error out of the bottom of save().
    if bumped >= FONT_REVISION_MAX:
        raise ValueError(
            "%s: would need fontRevision %.3f, at or above the usable 16.16 "
            "maximum of %g" % (basename, bumped, FONT_REVISION_MAX)
        )
    return bumped


def stamp_names(font, basename, bumped):
    """Mark the font as patched without touching family/full/PostScript names.

    Family, full and PostScript names (nameIDs 1, 4 and 6) are what Windows
    matches on, so preserving them is what makes a same-named install replace the
    font in place instead of registering a new family.

    nameID 3, the unique identifier, gains the marker on the first patch only;
    re-appending would corrupt it, and Windows does not consult it. What advances
    on every patch is head.fontRevision, with the nameID 5 version number kept in
    step, so "which generation of the patch is installed?" is answerable from the
    version string alone and a version-comparing installer sees an increase.
    """
    head = font["head"]
    head.fontRevision = bumped
    revision = "%.3f" % head.fontRevision

    name = font["name"]
    for record in list(name.names):
        if record.nameID not in (3, 5):
            continue
        try:
            value = record.toUnicode()
        except UnicodeDecodeError:
            continue
        if record.nameID == 5:
            value, substituted = VERSION_NUMBER.subn(revision, value, count=1)
            if not substituted:
                # The "version string tracks head.fontRevision" invariant just
                # broke: this record does not open with the spec's mandated
                # "Version <n>.<n>". Say so rather than silently diverging.
                # Flush stdout first, or the block-buffered progress lines land
                # after this and the warning appears attached to nothing.
                sys.stdout.flush()
                print(
                    "  warning: %s nameID 5 (platformID %d, platEncID %d, "
                    'langID 0x%x) has no "Version <n>.<n>" to update, so it will '
                    "not track fontRevision %s: %r"
                    % (
                        basename,
                        record.platformID,
                        record.platEncID,
                        record.langID,
                        revision,
                        value,
                    ),
                    file=sys.stderr,
                )
        if MARKER not in value:
            value += ";" + MARKER if record.nameID == 3 else " (" + MARKER + ")"
        name.setName(
            value,
            record.nameID,
            record.platformID,
            record.platEncID,
            record.langID,
        )


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


def patch_file(path, outdir, dry_run, repatch=False, min_revision=0.0):
    """Alias missing codepoints in one font. Returns (status, alias_count)."""
    font = TTFont(path)
    basename = os.path.basename(path)

    if already_patched(font) and not repatch:
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

    # Before the dry-run return, so a dry run surfaces a ceiling overflow instead
    # of reporting "would patch" for a font the real run would refuse.
    bumped = next_revision(font, basename, min_revision)

    if dry_run:
        return "would patch", added

    stamp_names(font, basename, bumped)
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
    parser.add_argument(
        "--repatch",
        action="store_true",
        help="also process fonts already carrying the marker, for when ALIAS has "
        "grown since they were built and no unpatched originals are to hand. "
        "Safe: cmap entries are additive and no outline is touched.",
    )
    parser.add_argument(
        "--min-revision",
        type=float,
        default=0.0,
        metavar="N.NNN",
        help="floor for head.fontRevision before the bump. Pass the revision "
        "already shipped/installed when regenerating from pristine originals, "
        "whose revision restarts lower and would otherwise emit a version "
        "Windows reads as older and declines to install over.",
    )
    args = parser.parse_args()

    # head.fontRevision is a signed 16.16 fixed-point field, so anything at or
    # above 32767 fails deep inside font.save() with a raw struct error, after
    # earlier files in the batch have already been written.
    if not 0.0 <= args.min_revision < FONT_REVISION_MAX:
        parser.error(
            "--min-revision must be in [0, %g): head.fontRevision is a 16.16 "
            "fixed-point field and cannot hold more" % FONT_REVISION_MAX
        )

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
    failed = 0
    for path in files:
        try:
            status, added = patch_file(
                path, args.out, args.dry_run, args.repatch, args.min_revision
            )
        except Exception as err:
            # One bad source font, or one unwritable output path, should cost you
            # that file and not the other eleven. The guard spans read, alias,
            # stamp and save, so keep the reporting neutral about which failed.
            failed += 1
            status, added = "FAILED (%s: %s)" % (type(err).__name__, err), 0
        if status in ("patched", "would patch"):
            written += 1
        print("  %-46s %+3d aliases  %s" % (os.path.basename(path), added, status))

    print(
        "\n%d/%d file(s) %s."
        % (written, len(files), "would change" if args.dry_run else "written")
    )
    # Before the failure branch: a partial batch is exactly when the operator most
    # needs pointing at the verifier, since the output directory is now of
    # uncertain completeness.
    if written and not args.dry_run:
        # Both paths absolute, so the hint is copy-pasteable from any working
        # directory. args.out is often relative, straight from the README recipe.
        verifier = os.path.join(
            os.path.dirname(os.path.abspath(__file__)), "verify_font.py"
        )
        print(
            "Verify with: %s %s --dir %s"
            % (sys.executable, verifier, os.path.abspath(args.out))
        )
    if failed:
        print("%d file(s) failed; see FAILED above." % failed)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
