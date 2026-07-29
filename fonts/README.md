# Patched RobotoMono Nerd Font Mono

`RobotoMono-NFM-patched.zip` holds the 12 styles of **RobotoMono Nerd Font Mono**
(Nerd Fonts 3.4.0), plus the Apache 2.0 licence text, with 16 extra cmap aliases
added so powerline-go's default symbols render in terminals that do no font
fallback, notably MobaXterm.

Generated 2026-07-27, extended 2026-07-29 to cover every symbol all three prompt
modes use. Reproducible from an unpatched original with `patch_font.py`. The
shipped generation reports `Version 3.003`; `verify_font.py` prints the version of
whatever font it inspects, which is how you tell what is actually installed.

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

powerline-go's default symbols (`defaults.go`) mix two different families of
codepoint. The `patched` mode block draws on both; `compatible` and `flat` use only
the second, since they are meant for terminals with no patched font at all:

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
| `U+2B22` ⬢ | NodeIndicator | `U+E718` dev-nodejs_small |
| `U+E92B` | RvmIndicator (`patched`) | `U+E739` dev-ruby |
| `U+25C6` ◆ | RvmIndicator (`compatible`, `flat`) | `U+F070B` md-rhombus |
| `U+25B6` ▶ | Separator (`compatible`) | `U+E0B0` pl-left_hard_divider |
| `U+25C0` ◀ | SeparatorReverse (`compatible`) | `U+E0B2` pl-right_hard_divider |
| `U+2744` ❄ | NixShellIndicator (`compatible`, `flat`) | `U+F313` linux-nixos |

The last six are a later addition, and the last four cover the `compatible` and
`flat` modes. Those modes exist to avoid needing a patched font at all, so they
spell everything in plain Unicode, which RobotoMono lacks just as thoroughly as
the symbols `patched` uses, meaning switching `-mode` used to trade one set of
blanks for another.

Three of the choices are judgement calls worth knowing about, because a cmap alias
is font-global and applies to any text in the terminal, not just the prompt:

* `U+2B22` gets Node's own hexagon logo rather than a bare hexagon such as `U+E24F`
  fae-hexagon, since at terminal sizes the logo reads as "node" where a plain
  hexagon reads as nothing in particular.
* `U+25B6` and `U+25C0` get the powerline dividers, which are exactly the filled
  triangles those codepoints name but drawn full-bleed to tile as separators. That
  is right for their actual use and bolder than a literal ▶ elsewhere.
* `U+25C6` deliberately goes the other way, to `md-rhombus` rather than the ruby
  icon that `U+E92B` gets, since ◆ is common in ordinary text and a diamond is what
  upstream meant by it.

`OS/2.ulUnicodeRange` is deliberately left exactly as upstream shipped it, even
though the aliases add coverage in blocks whose bits are unset (Arrows, Dingbats,
Misc Symbols, Misc Technical). Glyph lookup goes through the cmap, so nothing that
renders depends on the field; it is advisory, upstream Nerd Fonts is already
inconsistent about it (bit 90 is unset despite native Supplementary-PUA glyphs),
and rewriting it would mean touching a table the patch otherwise never opens. The
cost is that a capability query such as GDI `EnumFontFamiliesEx` under-reports what
the font covers.

Family, full and PostScript names are preserved so Windows treats the result as an
update to the same font rather than a new family. A first patch changes only the
unique id (nameID 3), the version string (nameID 5) and `head.fontRevision`; the
first two gain a `pwl-patch` marker so a patched file is identifiable. Each later
generation advances `head.fontRevision` and keeps the nameID 5 version number in
step with it, while nameID 3 keeps the bare marker, so the version string alone
answers "which generation of the patch is installed?".

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
python3 verify_font.py                    # every installed style of the Mono family
python3 verify_font.py --dir ./patched    # a directory of .ttf files
```

It reports every non-ASCII symbol `defaults.go` uses, across all three modes, plus
the glyphs hardcoded in Go source (`U+2026` in `segment-cwd.go` and `powerline.go`,
`U+00B5` in `segment-duration.go`), and whether the font can render each. One more,
`U+1F433` in `segment-docker_context.go`, is deliberately excluded; see "Known
remaining gaps". So it doubles as a check that an install actually took effect,
and reading the reported version line is how you confirm the *current* generation
is installed and not an earlier one.

It also decompiles every table rather than only the cmap and name tables it reads,
so a font whose outlines are corrupt fails here instead of passing a coverage-only
check. Corrupt files are reported by name and do not stop the other files in a
directory from being checked.

Every symbol is expected to render, so a single missing one exits non-zero, as
does a single corrupt file, and the run can gate a regeneration or a CI step.

## Regenerating

Requires `fonttools` >= 4.31.0 (`verify_font.py` uses `TTFont.ensureDecompiled`,
added in that release, and says so rather than reporting every font as corrupt).

```sh
python3 patch_font.py --dry-run           # show what would change
python3 patch_font.py --out ./patched     # write patched copies
rm -f RobotoMono-NFM-patched.zip          # fresh archive, not an in-place update
( cd patched && zip -9 -X ../RobotoMono-NFM-patched.zip *.ttf )
zip -9 -Xj RobotoMono-NFM-patched.zip LICENSE-Apache-2.0.txt  # Apache 2.0 section 4
```

Both scripts are meant to run **from WSL against the Windows drive**, which is how
the fonts get patched and installed here, so default discovery looks in
`/mnt/c/Users/*/AppData/Local/Microsoft/Windows/Fonts` (per-user installs) and
`/mnt/c/Windows/Fonts`, then the Linux font directories, for the first one that
actually holds a match. Anywhere else, including native Windows Python where those
`/mnt/c` paths do not exist, name the location explicitly: `--src` for
`patch_font.py`, `--dir` or `--file` for `verify_font.py`.

Because the patched fonts are now the installed ones on this machine, a bare re-run
reports "already patched" and writes nothing. Two ways forward:

* `--repatch` processes marked fonts anyway, which is how the `node`/`rvm` and
  `compatible`/`flat` aliases were added without unpatched originals to hand.
  Aliasing is purely additive to the cmap and touches no outline, so layering a
  generation on top of an earlier one is safe. It still writes nothing when the font
  already has every alias, so it is not a way to force a version bump.
* Point `--src` at unpatched originals from the
  [Nerd Fonts releases](https://github.com/ryanoasis/nerd-fonts/releases) to
  regenerate from scratch. Do this if the upstream Nerd Fonts version moves.

  **Pass `--min-revision` when you do.** A pristine original's
  `head.fontRevision` restarts at the upstream value (3.000 for Nerd Fonts 3.4.0),
  so the `+0.001` bump would emit 3.001, *lower* than the 3.003 already installed,
  which Windows reads as older and declines to install over. Floor it at the
  shipped revision:

  ```sh
  python3 patch_font.py --src ./nerd-fonts-originals --min-revision 3.003 --out ./patched
  ```

  That yields `Version 3.004`, one generation past the shipped 3.003, which is the
  point: Windows needs an increase to install over what is already there. Update
  the `Version 3.003` reference at the top of this file when you ship the result.
  To reproduce the shipped 3.003 exactly for comparison, pass
  `--min-revision 3.002`.

Neither the zip nor the fonts are byte-reproducible: zip records mtimes,
`fontTools` stamps `head.modified` with the save time, and each run advances
`fontRevision` by design. The **cmap tables** are reproducible, and they are the
whole substance of the patch. `verify_font.py` is what confirms a rebuild matches.

To patch the non-Mono family instead, which is what SSH sessions here use:

```sh
python3 patch_font.py --glob 'RobotoMonoNerdFont-*.ttf' --out ./patched-nonmono
```

## Known remaining gaps

One:

| Codepoint | Where | Module |
| --- | --- | --- |
| `U+1F433` 🐳 | hardcoded in `segment-docker_context.go` | `docker-context` |

No mono Nerd Font carries a whale, and none of the docker icons it does carry
(`seti-docker`, `dev-docker`, `fa-docker`, `linux-docker`, `md-docker`) reads as the
same thing at prompt size. Override it in `~/.config/powerline-go/config.json`
instead, pointing it at one of those, or leave `docker-context` disabled.

Aliasing it would not be a clean win regardless: 🐳 is East Asian Wide, so
powerline-go's `runewidth` accounting reserves **two** cells while every Nerd Fonts
icon has a one-cell advance, leaving a stray cell. Note this is not unique to the
whale — `U+2693` ⚓ RepoDetached is Wide too (`go-runewidth` returns 2 for both) and
*is* aliased, so detached HEAD already carries that one-cell overhang. That trade is
accepted there because a visible anchor with a stray cell beats a blank glyph with
two. The whale differs only in having no stand-in worth making the trade for.

Everything else resolves. Every non-ASCII codepoint the prompt can actually *print*
is in `SYMBOLS`: all three `defaults.go` mode blocks, plus `U+2026` in
`segment-cwd.go` and `powerline.go` and `U+00B5` in `segment-duration.go`. So "one
gap" is a claim `verify_font.py` can falsify rather than one resting on a narrower
list. Re-derive the inventory with:

```sh
( cd .. && find . -name '*.go' -print0 |
    xargs -0 grep -Poh '\\u[0-9a-fA-F]{4}|[^\x00-\x7F]' | sort -u )
```

The `cd ..` matters: every other command here runs from `fonts/`, but this one needs
the repository root, and from `fonts/` it would match nothing and print nothing while
appearing to pass.

That prints 33 rows for 32 distinct codepoints. Three are expected outside `SYMBOLS`:
`🐳` above, `’` (prose in comments only, never printed), and `\u001b` (the zsh
`ColorTemplate` escape prefix, not a glyph). The 33rd row is `…` written literally in
`powerline.go`, the same codepoint as the `\u2026` row from `segment-cwd.go`, and it is
in `SYMBOLS`.

If a future powerline-go symbol is missing from the font, add it to `SYMBOLS` in
`verify_font.py` only once the font can resolve it, natively or via a new `ALIAS`
target. That keeps `SYMBOLS` meaning "must render", which is what lets the verifier
be a clean pass/fail gate. If no reasonable stand-in exists, do what the whale entry
does: override the symbol in `config.json`, leave it out of `SYMBOLS`, and record why
both here and in a comment there. There is deliberately no expected-gap allowlist to
hide it in.

## Licensing

Roboto Mono is Copyright 2015 The Roboto Mono Project Authors and licensed under
the Apache License, Version 2.0, as recorded in the fonts' own name table. A copy is
included in the zip, and beside it here, as `LICENSE-Apache-2.0.txt`. The Nerd Fonts
patcher that produced the upstream files is MIT licensed.

**Notice of modification, per Apache 2.0 section 4:** the files in
`RobotoMono-NFM-patched.zip` are modified copies of the Nerd Fonts 3.4.0 build of
RobotoMono Nerd Font Mono. The modification is limited to the 16 cmap aliases
listed above plus the nameID 3, nameID 5 and `head.fontRevision` markers. No
glyph outlines were altered, added or removed.
