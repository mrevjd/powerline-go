# Patched RobotoMono Nerd Font Mono

`RobotoMono-NFM-patched.zip` holds the 12 styles of **RobotoMono Nerd Font Mono**
(Nerd Fonts 3.4.0) with 10 extra cmap aliases added so powerline-go's default
symbols render in terminals that do no font fallback, notably MobaXterm.

Generated 2026-07-27. Reproducible from an unpatched original with `patch_font.py`.

## The problem

powerline-go's git segment showed a branch and a count with nothing after it in
MobaXterm, while the same prompt rendered correctly in Windows Terminal on the
same machine.

Diagnostic, run in the affected terminal:

```sh
printf 'dashed:⇡⇣ basic:↑↓\n'
```

If `basic` renders but `dashed` is blank, this is the font gap, not a powerline-go
bug.

## Cause

powerline-go's default symbols (`defaults.go`, the `patched` mode block) mix two
different families of codepoint:

* Nerd Fonts private-use icons such as `U+E0B0` separator and `U+E0A0` branch,
  which RobotoMono Nerd Font does contain.
* Plain Unicode symbols such as `U+21E1` ahead and `U+270E` not-staged, which
  RobotoMono does **not** contain at all.

Windows Terminal hides the gap by substituting glyphs from another installed font
via DirectWrite fallback. MobaXterm does no fallback, so a missing glyph draws as
blank space. Changing MobaXterm's font only helps if the replacement font itself
carries those codepoints, and among the Nerd Fonts installed here only
CaskaydiaCove does.

Note that MobaXterm *does* fall back for very common codepoints (`U+2191`/`U+2193`
render even though RobotoMono lacks them) but not for rarer ones. That is why only
some symbols were blank rather than all of them.

## What the patch does

For each missing codepoint, it adds an entry to every Unicode cmap subtable
pointing at an icon the font already ships. No donor font, no outline copying, no
scaling, and no glyf changes at all.

| Codepoint | powerline-go field | Aliased to |
| --- | --- | --- |
| `U+21E1` ⇡ | RepoAhead | `U+F062` fa-arrow-up |
| `U+21E3` ⇣ | RepoBehind | `U+F063` fa-arrow-down |
| `U+2713` ✓ | RepoStaged | `U+F00C` fa-check |
| `U+270E` ✎ | RepoNotStaged | `U+F040` fa-pencil |
| `U+273C` ✼ | RepoConflicted | `U+F071` fa-warning |
| `U+2691` ⚑ | RepoStashed | `U+F024` fa-flag |
| `U+2693` ⚓ | RepoDetached | `U+F13D` fa-anchor |
| `U+2235` ∵ | DotEnvIndicator | `U+F15B` fa-file |
| `U+260E` ☎ | Network | `U+F095` fa-phone |
| `U+2388` ⎈ | KubeIndicator | `U+F085` fa-gears |

Family, full and PostScript names are preserved so Windows treats the result as an
update to the same font rather than a new family. Only the unique id (nameID 3),
version string (nameID 5) and `head.fontRevision` change, each gaining a
`pwl-patch` marker so a patched file is identifiable and re-running the script is a
no-op.

## Installing on Windows

1. Copy `RobotoMono-NFM-patched.zip` to the Windows side and extract it.
2. Select all 12 `.ttf` files, right-click, **Install** (or **Install for all
   users**). Same-name fonts are replaced in place.
3. Restart MobaXterm. Its terminal font stays `RobotoMono Nerd Font Mono`; no
   settings change is needed.
4. If a `~/.config/powerline-go/config.json` symbol override is still in place,
   retire it so powerline-go uses its defaults again:
   `mv ~/.config/powerline-go/config.json{,.bak}`

MobaXterm reads the session font family from `MobaXterm.ini` under `LastSession`.
That file is UTF-16, so read it with `iconv -f UTF-16LE` if you need to confirm
which family a session uses.

## Verifying

```sh
python3 verify_font.py                    # the currently installed Mono family
python3 verify_font.py --dir ./patched    # a directory of .ttf files
```

It reports every symbol in the `patched` mode block and whether the font can
render it, so it doubles as a check that an install actually took effect.

It exits non-zero if any symbol the patch is supposed to cover is missing, so it can
gate a regeneration or a CI step. The two known gaps below are reported but treated
as expected, and do not fail the run.

## Regenerating

Requires `fonttools`.

```sh
python3 patch_font.py --dry-run           # show what would change
python3 patch_font.py --out ./patched     # write patched copies
cd patched && zip -9 -X ../RobotoMono-NFM-patched.zip *.ttf
```

Both scripts are meant to run **from WSL against the Windows drive**, which is how
the fonts get patched and installed here, so default discovery looks in
`/mnt/c/Users/*/AppData/Local/Microsoft/Windows/Fonts` (per-user installs) and
`/mnt/c/Windows/Fonts`, then the Linux font directories, for the first one that
actually holds a match. Anywhere else, including native Windows Python where those
`/mnt/c` paths do not exist, pass `--src` explicitly.

Because the patched fonts are now the installed ones on this machine, a bare re-run
reports "already patched" and writes nothing; point `--src` at unpatched originals
from the [Nerd Fonts releases](https://github.com/ryanoasis/nerd-fonts/releases) to
regenerate from scratch.

The zip is not byte-reproducible, since zip records mtimes. The fonts inside are.

To patch the non-Mono family instead, which is what SSH sessions here use:

```sh
python3 patch_font.py --glob 'RobotoMonoNerdFont-*.ttf' --out ./patched-nonmono
```

## Known remaining gaps

Two symbols are still unmapped, because RobotoMono Nerd Font has no reasonable
stand-in icon for either:

| Codepoint | Field | Module |
| --- | --- | --- |
| `U+2B22` ⬢ | NodeIndicator | `node` |
| `U+E92B` | RvmIndicator | `rvm` |

Neither module is enabled in the prompt this was built for, so neither gap is
live. If you enable `node` or `rvm`, either add an entry to `ALIAS` in
`patch_font.py` and regenerate, or override the symbol in
`~/.config/powerline-go/config.json`.

## Licensing

Roboto Mono is Copyright 2015 The Roboto Mono Project Authors and licensed under
the Apache License, Version 2.0, as recorded in the fonts' own name table. A copy
is included as `LICENSE-Apache-2.0.txt`. The Nerd Fonts patcher that produced the
upstream files is MIT licensed.

**Notice of modification, per Apache 2.0 section 4:** the files in
`RobotoMono-NFM-patched.zip` are modified copies of the Nerd Fonts 3.4.0 build of
RobotoMono Nerd Font Mono. The modification is limited to the 10 cmap aliases
listed above plus the nameID 3, nameID 5 and `head.fontRevision` markers. No
glyph outlines were altered, added or removed.
