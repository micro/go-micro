#!/usr/bin/env python3
"""Scan legacy Jekyll site and Hugo site, match by title, report findings.

Usage:
    python3 scripts/add-aliases.py [--apply]

Without --apply: read-only scan, prints match table.
With --apply: adds aliases frontmatter to matched Hugo files.
"""

import argparse
import re
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
SOURCE_DIR = REPO_ROOT / "assets" / "website"
TARGET_DIR = REPO_ROOT / "content" / "en"

FRONTMATTER_RE = re.compile(r"^---\s*\n(.*?)\n---\s*\n", re.DOTALL)


def parse_frontmatter(text: str) -> dict[str, str]:
    """Extract YAML frontmatter as a simple key-value dict.

    Handles quoted and unquoted values without requiring a YAML library.
    """
    m = FRONTMATTER_RE.match(text)
    if not m:
        return {}
    fm: dict[str, str] = {}
    for line in m.group(1).splitlines():
        if ":" not in line:
            continue
        key, _, val = line.partition(":")
        key = key.strip()
        val = val.strip().strip('"').strip("'")
        if key and val:
            fm[key] = val
    return fm


def scan_sources() -> dict[str, str]:
    """Return {title: permalink} for source files that have both fields."""
    results: dict[str, str] = {}
    for md in sorted(SOURCE_DIR.rglob("*.md")):
        if md.name == "README.md":
            continue
        text = md.read_text(encoding="utf-8", errors="replace")
        fm = parse_frontmatter(text)
        title = fm.get("title")
        permalink = fm.get("permalink")
        if title and permalink:
            results[title] = permalink
    return results


def scan_targets() -> dict[str, list[Path]]:
    """Return {title: [file_paths]} for target files that have a title."""
    results: dict[str, list[Path]] = {}
    for md in sorted(TARGET_DIR.rglob("*.md")):
        text = md.read_text(encoding="utf-8", errors="replace")
        fm = parse_frontmatter(text)
        title = fm.get("title")
        if title:
            results.setdefault(title, []).append(md)
    return results


def pick_target(title: str, permalink: str, candidates: list[Path]) -> Path:
    """Pick the best target file from candidates based on source section.

    Blog sources (permalink starts with /blog/) prefer blog/ targets.
    Doc sources prefer docs/ targets. Falls back to first candidate.
    """
    if len(candidates) == 1:
        return candidates[0]

    is_blog_source = permalink.startswith("/blog/")
    for c in candidates:
        rel = str(c.relative_to(TARGET_DIR))
        if is_blog_source and "blog/" in rel:
            return c
        if not is_blog_source and "docs/" in rel:
            return c
    return candidates[0]


def has_alias(text: str, permalink: str) -> bool:
    """Check if the file's frontmatter already contains this alias."""
    m = FRONTMATTER_RE.match(text)
    if not m:
        return False
    fm_block = m.group(1)
    # Look for the permalink in an aliases list
    return permalink in fm_block


def add_alias(text: str, permalink: str) -> str:
    """Add aliases: [<permalink>] to the frontmatter, preserving formatting."""
    m = FRONTMATTER_RE.match(text)
    if not m:
        # No frontmatter — prepend one
        return f"---\naliases:\n  - {permalink}\n---\n{text}"

    fm_block = m.group(1)
    rest = text[m.end():]

    # Check if aliases field already exists
    aliases_re = re.compile(r"(^aliases:\s*\n(?:\s+-\s+.+\n)*)", re.MULTILINE)
    am = aliases_re.search(fm_block)
    if am:
        # Append to existing aliases list
        new_fm_block = fm_block[:am.end()] + f"  - {permalink}\n" + fm_block[am.end():]
    else:
        # Add new aliases field at end of frontmatter
        new_fm_block = fm_block + f"\naliases:\n  - {permalink}"

    return f"---\n{new_fm_block}\n---\n{rest}"


def main() -> None:
    parser = argparse.ArgumentParser(description="Add aliases from legacy Jekyll permalinks")
    parser.add_argument("--apply", action="store_true", help="Write aliases to Hugo files")
    args = parser.parse_args()

    sources = scan_sources()
    targets = scan_targets()

    matched: list[tuple[str, str, Path]] = []
    no_match: list[tuple[str, str]] = []
    skipped_no_title = 0
    skipped_no_permalink = 0

    # Count source files without title or permalink
    for md in sorted(SOURCE_DIR.rglob("*.md")):
        if md.name == "README.md":
            continue
        text = md.read_text(encoding="utf-8", errors="replace")
        fm = parse_frontmatter(text)
        if not fm.get("title"):
            skipped_no_title += 1
        elif not fm.get("permalink"):
            skipped_no_permalink += 1

    # Find matches
    for title, permalink in sources.items():
        if title in targets:
            target = pick_target(title, permalink, targets[title])
            matched.append((title, permalink, target))
        else:
            no_match.append((title, permalink))

    # Report
    print(f"Source files with title+permalink: {len(sources)}")
    print(f"Source files without title:        {skipped_no_title}")
    print(f"Source files without permalink:    {skipped_no_permalink}")
    print(f"Target files with title:           {len(targets)}")
    print(f"Exact title matches:               {len(matched)}")
    print(f"No match:                          {len(no_match)}")
    print()

    if matched:
        print("=== MATCHES ===")
        for title, permalink, target in matched:
            rel_target = target.relative_to(REPO_ROOT)
            print(f"  {permalink:30s} -> {rel_target}")
            print(f"    title: {title}")
        print()

    if no_match:
        print("=== NO MATCH ===")
        for title, permalink in no_match:
            print(f"  {permalink:30s}  title: {title}")
        print()

    # Warn about duplicate titles
    duplicates = {t: ps for t, ps in targets.items() if len(ps) > 1}
    if duplicates:
        print("=== DUPLICATE TITLES (picked best match) ===")
        for title, paths in sorted(duplicates.items()):
            rels = [str(p.relative_to(REPO_ROOT)) for p in paths]
            print(f"  {title}: {', '.join(rels)}")
        print()

    if args.apply:
        modified = 0
        already_done = 0
        for title, permalink, target in matched:
            text = target.read_text(encoding="utf-8")
            if has_alias(text, permalink):
                already_done += 1
                continue
            target.write_text(add_alias(text, permalink), encoding="utf-8")
            modified += 1
            rel = target.relative_to(REPO_ROOT)
            print(f"  modified: {rel}")

        print(f"\nModified: {modified}")
        print(f"Already had alias: {already_done}")
    else:
        print("Dry run — no files modified. Use --apply to write aliases.")


if __name__ == "__main__":
    main()
