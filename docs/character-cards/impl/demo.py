"""A worked end-to-end demonstration of the reader, the writer, and the scanner.

Run it with ``python3 demo.py``. It writes its output files into a ``demo_out`` directory
next to itself so the artefacts can be opened in other tools.

The script is written as a narrative rather than as a test: each section prints what it is
about to do and why, then shows the result. It doubles as the worked example in
``IMPLEMENTATION_GUIDE.md``, so the assertions in it are deliberately loud.
"""

from __future__ import annotations

import json
import os

import ccv3
from ccv3 import ScanContext, export, lorebook, png


def heading(number: str, title: str) -> None:
    print()
    print("=" * 72)
    print(f"{number}. {title}")
    print("=" * 72)


def show(label: str, value: object) -> None:
    print(f"  {label:<28} {value}")


def build_a_v3_card() -> dict:
    """A card that exercises every feature the guide talks about.

    It carries a lorebook with decorators, an embedded asset, multilingual notes, a source
    list, and one field that no version of the specification defines, so that the
    forward-compatibility behaviour is visible in the output rather than only asserted.
    """
    return {
        "spec": "chara_card_v3",
        "spec_version": "3.0",
        "data": {
            "name": "Aria",
            "nickname": "Ari",
            "description": "A cartographer who maps places that do not exist yet.",
            "personality": "Precise, patient, quietly funny.",
            "scenario": "You have hired Aria to draw a map of a valley you cannot find again.",
            "first_mes": "Aria unrolls a blank sheet. {{comment:the map is blank on purpose}} \"Show me where you think it was.\"",
            "mes_example": "<START>\n{{user}}: Where does this road go?\n{{char}}: Nowhere yet.",
            "creator_notes": "Written as a reference card for the ccv3 reader.",
            "creator_notes_multilingual": {
                "en": "A cartographer.",
                "pt": "Uma cartógrafa.",
            },
            "system_prompt": "Stay in character.",
            "post_history_instructions": "Keep replies short.",
            "alternate_greetings": [
                "Aria looks up from a half-drawn coastline.",
                "@@keep_activate_after_match\nA third greeting that carries a decorator by mistake.",
            ],
            "group_only_greetings": ["The other cartographers compare notes."],
            "tags": ["cartography", "reference", "test"],
            "creator": "goharness",
            "character_version": "1.0",
            "source": ["https://example.invalid/aria"],
            "creation_date": 1700000000,
            "modification_date": 1700000100,
            "extensions": {},
            "assets": [
                {
                    "type": "icon",
                    "uri": "embeded://assets/icon/images/main.png",
                    "name": "main",
                    "ext": "png",
                },
                {
                    "type": "background",
                    "uri": "ccdefault:",
                    "name": "parchment",
                    "ext": "png",
                },
            ],
            "character_book": {
                "name": "Aria's atlas",
                "description": "Lore about the valley and the maps.",
                "scan_depth": 4,
                "token_budget": 400,
                "recursive_scanning": True,
                "extensions": {},
                "entries": [
                    {
                        "keys": ["valley"],
                        "content": "The valley moves. It is never in the same place twice.",
                        "enabled": True,
                        "insertion_order": 10,
                        "extensions": {},
                        "name": "The valley",
                        "priority": 10,
                    },
                    {
                        "keys": ["map", "plan"],
                        "content": "@@additional_keys atlas\n@@exclude_keys finished\nAn unfinished map is a promise, not a failure.",
                        "enabled": True,
                        "insertion_order": 20,
                        "extensions": {},
                        "priority": 5,
                    },
                    {
                        "keys": ["north"],
                        "content": "@@risu_only_decorator 5\n@@@agn_only_decorator 5\n@@@activate_only_after 4\n@@position after_desc\nNorth is a convention, not a fact. {{hidden_key:compass}}",
                        "enabled": True,
                        "insertion_order": 30,
                        "extensions": {},
                        "priority": 1,
                    },
                    {
                        "keys": ["secrets"],
                        "content": "This must never appear.",
                        "enabled": False,
                        "insertion_order": 40,
                        "extensions": {},
                    },
                ],
            },
            "x_from_a_future_spec_version": {"kept": "because unknown fields are preserved"},
        },
    }


def main() -> None:
    out_dir = os.path.join(os.path.dirname(os.path.abspath(__file__)), "demo_out")
    os.makedirs(out_dir, exist_ok=True)

    heading("1", "Write a V3 card into a PNG, with both chunks")
    source = build_a_v3_card()
    card = ccv3.read_text(json.dumps(source))

    png_path = os.path.join(out_dir, "aria.png")
    blob = export.write_png(card, also_write_chara=True)
    with open(png_path, "wb") as handle:
        handle.write(blob)

    chunks, _warnings = png.read_text_chunks(blob)
    show("file written", png_path)
    show("size", f"{len(blob)} bytes")
    show("text chunks present", ", ".join(sorted(chunks)))
    print()
    print("  Both chunks are written on purpose. A V3 reader takes 'ccv3'; an older one takes")
    print("  'chara' and loses nothing it could have used, because V3 is a superset of V2.")

    heading("2", "Read the PNG back")
    reread = ccv3.read(blob)
    show("container", reread.container)
    show("declared as", reread.origin_spec)
    show("normalised to", f"{reread.spec} {reread.spec_version}")
    show("name / display name", f"{reread.data.name} / {reread.display_name}")
    show("read from the legacy chunk", reread.backfilled)
    show("warnings", len(reread.warnings))
    for message in reread.warnings:
        print(f"      - {message}")

    heading("3", "Check that unknown fields survived the round trip")
    written_back = export.to_v3_dict(reread)
    future = written_back["data"].get("x_from_a_future_spec_version")
    show("x_from_a_future_spec_version", future)
    assert future == {"kept": "because unknown fields are preserved"}, "unknown field was lost"
    print()
    print("  A reader that dropped unrecognised fields would delete this. The specification")
    print("  asks applications to ignore unknown fields, and preserving them is what makes a")
    print("  round trip safe.")

    heading("4", "Scan the lorebook")
    for text in (
        "tell me about the valley",
        "do you have a map of the place?",
        "which way is north?",
        "what are your secrets?",
    ):
        print()
        print(f"  scan: {text!r}")
        result = ccv3.scan_lorebook(
            reread,
            ScanContext(
                messages=[text],
                char_name=reread.display_name,
                assistant_message_count=6,
            ),
        )
        if not result.hits:
            print("      nothing matched")
        for hit in result.hits:
            keys = ", ".join(hit.matched_keys) or "(forced by a decorator)"
            print(f"      matched on {keys}")
            print(f"      -> {hit.content!r}")
            if hit.needs_prompt_placement:
                print(f"      placement: {hit.directives}")

    print()
    print("  Note the third entry. Its content begins with a fallback chain:")
    print()
    print("      @@risu_only_decorator 5      <- unknown to this reader")
    print("      @@@agn_only_decorator 5      <- also unknown")
    print("      @@@activate_only_after 4     <- recognised, so this one applies")
    print()
    print("  A chain is walked from the top and the first recognised name wins. The two")
    print("  unknown names cost the author nothing, which is the point of the mechanism:")
    print("  @@risu_only_decorator is not an error, it is a fallback that did not need to fire.")
    print("  None of the decorator lines appears in the injected text, and the hidden key was")
    print("  lifted out of the text into the matchable text instead.")

    heading("5", "Export as V2 and watch the decorators come off")
    v2 = export.to_v2_dict(reread)
    show("spec", v2["spec"])
    show("spec_version", v2["spec_version"])
    original_content = reread.lorebook.entries[1].content
    exported_content = v2["data"]["character_book"]["entries"][1]["content"]
    print()
    print("  V3 entry content:")
    print(f"      {original_content!r}")
    print("  The same entry written as V2:")
    print(f"      {exported_content!r}")
    assert "@@" not in exported_content, "decorators should be stripped on V2 backfill"
    print()
    print("  The specification asks for this explicitly. A V2 reader has no idea what")
    print("  '@@exclude_keys finished' means, so it would paste the line into the prompt.")

    heading("6", "Write a V2-only file and read it back")
    v2_path = os.path.join(out_dir, "aria_v2.png")
    v2_blob = export.write_png(reread, base_image=blob, spec="chara_card_v2")
    with open(v2_path, "wb") as handle:
        handle.write(v2_blob)

    show("file", v2_path)
    show("text chunks", ", ".join(sorted(png.read_text_chunks(v2_blob)[0])))
    v2_card = ccv3.read(v2_blob)
    show("declared as", v2_card.origin_spec)
    show("normalised to", v2_card.spec)
    show("nickname survived", repr(v2_card.data.nickname))
    show("assets survived", len(v2_card.data.assets))
    show("multilingual notes survived", list(v2_card.data.creator_notes_multilingual))
    print()
    print("  The file says V2 and carries no 'ccv3' chunk, so a V2-only reader handles it")
    print("  correctly. The V3 fields are still in there, as unknown fields that the V2")
    print("  specification tells readers to ignore and that this reader picks back up.")

    heading("7", "Read a PNG that has only the legacy chunk")
    legacy = png.remove_text_chunk(blob, "ccv3")
    legacy_path = os.path.join(out_dir, "aria_legacy.png")
    with open(legacy_path, "wb") as handle:
        handle.write(legacy)

    legacy_card = ccv3.read(legacy)
    show("file", legacy_path)
    show("text chunks", ", ".join(sorted(png.read_text_chunks(legacy)[0])))
    show("read from the legacy chunk", legacy_card.backfilled)
    show("name still readable", legacy_card.data.name)
    print()
    for message in legacy_card.warnings:
        print(f"      - {message}")

    heading("8", "What the normalised card looks like")
    print(f"  Data fields on a V3 card:   {len(vars(reread.data))}")
    v1_card = ccv3.read_text(json.dumps({
        "name": "Old", "description": "d", "personality": "p",
        "scenario": "s", "first_mes": "f", "mes_example": "m",
    }))
    print(f"  Data fields on a V1 card:   {len(vars(v1_card.data))}")
    print("  Same count, same names, same defaults. That is what normalising upward buys: no")
    print("  code above the reader ever needs to branch on the version.")

    print()
    print("=" * 72)
    print("Artefacts written to demo_out/:")
    for name in sorted(os.listdir(out_dir)):
        size = os.path.getsize(os.path.join(out_dir, name))
        print(f"  {name:<20} {size:>7} bytes")
    print("=" * 72)


if __name__ == "__main__":
    main()
