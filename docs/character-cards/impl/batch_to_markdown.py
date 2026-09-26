"""Convert a directory or archive of character cards into Markdown documents.

    python3 batch_to_markdown.py <input-dir-or-zip> <output-dir>

The input may be a directory of card files or a ZIP containing them. Cards are read with the
same reader the rest of this package uses, so anything the reader reports as a warning is
reproduced faithfully in the document and in the run summary at the end.

The script always exits zero unless nothing could be read at all. A partial conversion is a
result, not a failure, and refusing to write the documents that did work because one card was
malformed would be unhelpful.

Two files are written into the output directory beyond the per-card documents:

* ``_index.md`` lists every card with its source file and the fields most useful for scanning
  a large collection.
* ``_report.txt`` records what happened, including every warning and every failure, so that a
  conversion of a few hundred cards can be audited afterwards without re-running it.
"""

from __future__ import annotations

import argparse
import collections
import os
import re
import sys
import zipfile
from typing import Iterable, Optional

import ccv3
from ccv3 import markdown

#: Extensions worth attempting to read.
CARD_EXTENSIONS = (".png", ".apng", ".json", ".charx", ".webp", ".jpg", ".jpeg")


def safe_name(name: str, *, taken: set[str]) -> str:
    """Turn a card's filename into a Markdown filename that is safe and unique.

    Three things happen here, and each was prompted by a real name in a batch of sixty cards.

    The ``.card`` marker is removed wherever it appears, not only when it directly precedes a
    known extension. Cards are exported as ``Character.card.png`` most of the time, but also
    as ``Character.card v2.png``, where a suffix check on ``.card.png`` matches nothing and
    leaves the marker stranded in the middle of the name.

    Characters that are not letters, digits, or a small set of punctuation become an
    underscore, and runs of underscores are then collapsed and the ends trimmed. Without that
    last step a card whose name begins with an emoji produces a filename beginning with a bare
    underscore, which is awkward to type and sorts away from the card it belongs to.

    A numeric suffix is added when the cleaned name has already been used, because two cards
    can share a name, or differ only in punctuation that the cleaning erased.
    """
    stem = os.path.basename(name)

    without_marker = re.sub(r"\.card(\s+v\d+)?(\.[A-Za-z0-9]+)?$", lambda m: m.group(1) or "", stem)
    stem = without_marker if without_marker != stem else os.path.splitext(stem)[0]

    cleaned = re.sub(r"[^\w \-.,'()\[\]&]", "_", stem, flags=re.UNICODE)
    cleaned = re.sub(r"_{2,}", "_", cleaned)
    cleaned = re.sub(r"\s{2,}", " ", cleaned).strip(" _")
    cleaned = cleaned or "card"

    candidate = f"{cleaned}.md"
    counter = 2
    while candidate.lower() in taken:
        candidate = f"{cleaned} ({counter}).md"
        counter += 1
    taken.add(candidate.lower())
    return candidate


def collect_inputs(source: str) -> list[tuple[str, bytes]]:
    """Gather cards from a directory or a ZIP, returning pairs of name and bytes."""
    if os.path.isdir(source):
        collected: list[tuple[str, bytes]] = []
        for entry in sorted(os.listdir(source)):
            path = os.path.join(source, entry)
            if not os.path.isfile(path):
                continue
            if not entry.lower().endswith(CARD_EXTENSIONS):
                continue
            with open(path, "rb") as handle:
                collected.append((entry, handle.read()))
        return collected

    if zipfile.is_zipfile(source):
        from ccv3.containers import _find_zip_start  # the same offset logic the reader uses

        offset = _find_zip_start(open(source, "rb").read()) or 0
        collected = []
        with zipfile.ZipFile(source) as archive:
            for entry in sorted(archive.namelist()):
                if entry.endswith("/"):
                    continue
                if not entry.lower().endswith(CARD_EXTENSIONS):
                    continue
                collected.append((os.path.basename(entry), archive.read(entry)))
        return collected

    raise SystemExit(f"{source} is neither a directory nor a readable ZIP archive")


def main(argv: Optional[list[str]] = None) -> int:
    parser = argparse.ArgumentParser(
        description="Convert character cards into Markdown documents."
    )
    parser.add_argument("source", help="a directory of cards, or a ZIP containing them")
    parser.add_argument("output", help="the directory to write Markdown into")
    parser.add_argument(
        "--user-name",
        default="you",
        help="what {{user}} becomes in the rendered text (default: you)",
    )
    parser.add_argument(
        "--no-lorebook",
        action="store_true",
        help="omit lorebook entries from the documents",
    )
    parser.add_argument(
        "--include-empty",
        action="store_true",
        help="emit a section for every field, even the ones the card leaves unset",
    )
    parser.add_argument(
        "--raw-macros",
        action="store_true",
        help="leave {{char}} and {{user}} as written, instead of translating them for reading",
    )
    parser.add_argument(
        "--raw-html",
        action="store_true",
        help="leave markup in the prose fields as stored, instead of converting it to Markdown",
    )
    args = parser.parse_args(argv)

    cards = collect_inputs(args.source)
    if not cards:
        print(f"no card files found in {args.source}", file=sys.stderr)
        return 2

    os.makedirs(args.output, exist_ok=True)

    converted: list[tuple[str, str, ccv3.Card]] = []
    failures: list[tuple[str, str]] = []
    taken: set[str] = {"_index.md", "_report.txt"}
    warning_kinds: collections.Counter = collections.Counter()
    specs: collections.Counter = collections.Counter()

    for name, blob in cards:
        try:
            card = ccv3.read(blob)
        except Exception as exc:  # noqa: BLE001 - reported, not raised
            failures.append((name, f"{type(exc).__name__}: {exc}"))
            continue

        document = markdown.to_markdown(
            card,
            source_name=name,
            user_name=args.user_name,
            include_lorebook=not args.no_lorebook,
            include_empty=args.include_empty,
            render_macros=not args.raw_macros,
            convert_html=not args.raw_html,
        )
        out_name = safe_name(name, taken=taken)
        with open(os.path.join(args.output, out_name), "w", encoding="utf-8") as handle:
            handle.write(document)

        converted.append((name, out_name, card))
        specs[card.origin_spec] += 1
        for message in card.warnings:
            warning_kinds[message[:200]] += 1

    # --- index -------------------------------------------------------------------------
    lines = [
        "# Converted character cards",
        "",
        f"Converted from `{os.path.basename(args.source)}`.",
        "",
        f"{len(converted)} of {len(cards)} cards converted"
        + (f", {len(failures)} failed." if failures else "."),
        "",
        "| Card | Name | Version | Description | Lorebook |",
        "| --- | --- | --- | --- | --- |",
    ]
    for source_name, out_name, card in sorted(
        converted,
        key=lambda row: markdown.display_label(row[2], row[0]).lower(),
    ):
        data = card.data
        # The index describes the same cards the documents do, so it goes through the same
        # rendering rather than showing the author's raw markup and macro placeholders.
        summary = data.description or data.personality or data.scenario or ""
        if not args.raw_html:
            summary = markdown.html_to_markdown(summary)
        if not args.raw_macros:
            summary = markdown._display_macros(
                summary, char_name=card.display_name, user_name=args.user_name
            )
        description = " ".join(summary.split())
        description = description.replace("|", "\\|")
        if len(description) > 110:
            description = description[:107].rstrip() + "…"
        book = card.lorebook
        book_cell = f"{len(book)} entries" if book is not None else "—"
        label = markdown.display_label(card, source_name).replace("|", "\\|")
        version = card.origin_spec.replace("chara_card_", "").upper()
        lines.append(
            f"| [{label}]({out_name}) | {data.name.strip().replace('|', chr(92) + '|') or '—'} | {version} "
            f"| {description} | {book_cell} |"
        )
    lines.append("")

    with open(os.path.join(args.output, "_index.md"), "w", encoding="utf-8") as handle:
        handle.write("\n".join(lines))

    # --- report ------------------------------------------------------------------------
    report = [
        f"Source            : {args.source}",
        f"Output            : {args.output}",
        f"Card files found  : {len(cards)}",
        f"Converted         : {len(converted)}",
        f"Failed            : {len(failures)}",
        "",
        "Specifications read:",
    ]
    for spec, count in specs.most_common():
        report.append(f"  {count:>4}  {spec}")

    total_entries = sum(len(card.lorebook) for _s, _o, card in converted if card.lorebook)
    with_book = sum(1 for _s, _o, card in converted if card.lorebook is not None)
    report += [
        "",
        f"Cards with a lorebook : {with_book}",
        f"Lorebook entries total: {total_entries}",
        "",
        f"Distinct warning kinds: {len(warning_kinds)}",
    ]
    for message, count in warning_kinds.most_common():
        report.append(f"  {count:>4}  {message}")

    if failures:
        report += ["", f"Failures ({len(failures)}):"]
        for name, reason in failures:
            report.append(f"  {name}")
            report.append(f"      {reason}")

    report_text = "\n".join(report) + "\n"
    with open(os.path.join(args.output, "_report.txt"), "w", encoding="utf-8") as handle:
        handle.write(report_text)

    print(report_text)
    return 0 if converted else 1


if __name__ == "__main__":
    raise SystemExit(main())
