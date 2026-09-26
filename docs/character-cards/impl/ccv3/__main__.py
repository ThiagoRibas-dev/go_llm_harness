"""Command line entry point: inspect a card without writing any code.

    python3 -m ccv3 character.png
    python3 -m ccv3 character.charx --scan "tell me about the sword"
    python3 -m ccv3 character.json --v2

The output is meant to answer the questions that come up when a card behaves unexpectedly:
which container was it, which specification did it claim, what did the reader have to guess,
and what does its lorebook do.
"""

from __future__ import annotations

import argparse
import json
import sys
from typing import Any

from . import __version__, export, read_file, scan_lorebook
from .lorebook import ScanContext


def _rule(title: str) -> None:
    print()
    print(title)
    print("-" * len(title))


def describe(card: Any, *, show_warnings: bool = True) -> None:
    """Print a summary of a card."""
    data = card.data

    _rule("Card")
    print(f"  name             {data.name or '(none)'}")
    if data.nickname:
        print(f"  nickname         {data.nickname}")
    print(f"  container        {card.container}")
    print(f"  declared as      {card.origin_spec} (version {card.origin_version:g})")
    print(f"  read via         {card.container}", end="")
    if card.backfilled:
        print("  (from the legacy 'chara' chunk)")
    else:
        print()
    print(f"  normalised to    {card.spec} {card.spec_version}")

    creator = data.creator or "(not stated)"
    print(f"  creator          {creator}")
    print(f"  tags             {', '.join(data.tags) if data.tags else '(none)'}")

    _rule("Content")
    print(f"  description      {len(data.description):>6} characters")
    print(f"  first message    {len(data.first_mes):>6} characters")
    print(f"  example dialogue {len(data.mes_example):>6} characters")
    print(f"  greetings        {1 + len(data.alternate_greetings)} total")
    if data.source:
        print(f"  source           {', '.join(data.source)}")

    _rule("V3 additions")
    print(f"  assets           {len(data.assets)}")
    for asset in data.assets:
        marker = " (embedded)" if asset.is_embedded else ""
        print(f"      {asset.type:<11} {asset.name or '(unnamed)':<12} {asset.uri}{marker}")
    print(f"  languages        {', '.join(data.creator_notes_multilingual) or '(none)'}")
    print(f"  group greetings  {len(data.group_only_greetings)}")
    print(f"  created          {data.creation_date or '(not stated)'}")
    print(f"  modified         {data.modification_date or '(not stated)'}")

    if data.extra:
        _rule("Fields this reader does not recognise, preserved")
        for key in sorted(data.extra):
            print(f"      {key}")

    book = card.lorebook
    _rule("Lorebook")
    if book is None:
        print("  none")
    else:
        print(f"  entries          {len(book)}")
        print(f"  scan_depth       {book.scan_depth if book.scan_depth is not None else '(unset)'}")
        print(f"  token_budget     {book.token_budget if book.token_budget is not None else '(unset)'}")
        recursive = book.recursive_scanning
        print(f"  recursive        {'(unset)' if recursive is None else recursive}")
        enabled = sum(1 for e in book.entries if e.enabled)
        constants = sum(1 for e in book.entries if e.constant)
        print(f"  enabled          {enabled} of {len(book)}")
        print(f"  constant         {constants}")

    if show_warnings:
        _rule(f"Warnings ({len(card.warnings)})")
        if not card.warnings:
            print("  none")
        for message in card.warnings:
            print(f"  - {message}")


def do_scan(card: Any, text: str) -> None:
    """Scan the card's lorebook against a piece of text and print what matched."""
    result = scan_lorebook(card, ScanContext(messages=[text], char_name=card.display_name))

    _rule(f"Lorebook scan against: {text!r}")
    if not result.hits:
        print("  nothing matched")
    for hit in result.hits:
        order = hit.insertion_order
        keys = ", ".join(hit.matched_keys) or "(forced)"
        print(f"  [{order:>4}] keys: {keys}")
        for line in hit.content.splitlines() or [""]:
            print(f"           {line}")
        if hit.needs_prompt_placement:
            print(f"           placement requested: {hit.directives}")
        print()

    if result.dropped:
        print(f"  dropped for the token budget: {len(result.dropped)}")
    print(f"  injected tokens (approximate): {result.total_tokens}")
    if result.warnings:
        print()
        for message in result.warnings:
            print(f"  ! {message}")


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        prog="ccv3",
        description="Inspect a Character Card V1, V2, or V3 file.",
    )
    parser.add_argument("path", help="the card file: PNG, APNG, JSON, or CHARX")
    parser.add_argument("--scan", metavar="TEXT", help="scan the lorebook against this text")
    parser.add_argument(
        "--v2",
        action="store_true",
        help="also print the card as V2, to see what a V2-only reader would receive",
    )
    parser.add_argument("--json", action="store_true", help="print the normalised V3 object")
    parser.add_argument("--version", action="version", version=f"ccv3 {__version__}")

    args = parser.parse_args(argv)

    try:
        card = read_file(args.path)
    except Exception as exc:  # noqa: BLE001 - the CLI reports rather than traces
        print(f"could not read {args.path}: {exc}", file=sys.stderr)
        return 2

    describe(card)

    if args.scan:
        do_scan(card, args.scan)

    if args.v2:
        _rule("The same card as V2, with decorators stripped")
        print(json.dumps(export.to_v2_dict(card), indent=2, ensure_ascii=False))

    if args.json:
        _rule("The normalised V3 object")
        print(json.dumps(export.to_v3_dict(card), indent=2, ensure_ascii=False))

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
