"""Tests for the ccv3 reader, writer, and scanner.

Run with ``python3 -m unittest discover -s . -p 'test_*.py' -v`` from this directory, or
simply ``python3 test_ccv3.py``.

The tests are grouped by the pipeline step they exercise, in pipeline order, so a failure
names the stage that broke. The V2 retro-compatibility tests are the ones to read first, since
that is the property the whole design is arranged around.
"""

from __future__ import annotations

import json
import os
import unittest

import ccv3
from ccv3 import (
    cbs,
    containers,
    decorators,
    export,
    lorebook,
    markdown,
    normalise,
    png,
    validate,
    version,
)
from ccv3.model import Asset, Card, CardData, Lorebook, LorebookEntry


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


def v3_card_dict(*, name: str = "Aria", extra_data: dict | None = None) -> dict:
    """A small but complete V3 card, with every required field present."""
    data = {
        "name": name,
        "description": "A test character.",
        "personality": "Curious.",
        "scenario": "A test scenario.",
        "first_mes": "Hello there.",
        "mes_example": "<START>\n{{user}}: hi\n{{char}}: hello",
        "creator_notes": "Notes for the user.",
        "system_prompt": "Be helpful.",
        "post_history_instructions": "Stay in character.",
        "alternate_greetings": ["A second greeting."],
        "tags": ["test", "example"],
        "creator": "tester",
        "character_version": "1.0",
        "extensions": {},
        "group_only_greetings": [],
    }
    if extra_data:
        data.update(extra_data)
    return {"spec": "chara_card_v3", "spec_version": "3.0", "data": data}


def v2_card_dict(*, name: str = "Aria") -> dict:
    """The same card written against V2, so it carries no V3-only fields."""
    v3 = v3_card_dict(name=name)
    data = v3["data"]
    data.pop("group_only_greetings")
    return {"spec": "chara_card_v2", "spec_version": "2.0", "data": data}


def v1_card_dict(*, name: str = "Aria") -> dict:
    """A bare V1 card, with the character fields at the top level."""
    return {
        "name": name,
        "description": "A test character.",
        "personality": "Curious.",
        "scenario": "A test scenario.",
        "first_mes": "Hello there.",
        "mes_example": "<START>\n{{user}}: hi\n{{char}}: hello",
    }


def lorebook_dict(entries: list[dict]) -> dict:
    return {"extensions": {}, "entries": entries}


def entry(**kwargs) -> dict:
    base = {
        "keys": [],
        "content": "",
        "extensions": {},
        "enabled": True,
        "insertion_order": 0,
    }
    base.update(kwargs)
    return base


def png_with_card(card_dict: dict, keyword: str = "ccv3", *, also: dict | None = None) -> bytes:
    """Build a PNG carrying a card in the named chunk."""
    image = export.build_placeholder_png()
    payload = containers.encode_base64_text(json.dumps(card_dict).encode("utf-8"))
    result = png.set_text_chunk(image, keyword, payload)
    if also is not None:
        other = containers.encode_base64_text(json.dumps(also).encode("utf-8"))
        result = png.set_text_chunk(result, "chara" if keyword == "ccv3" else "ccv3", other)
    return result


# ---------------------------------------------------------------------------
# Step one and two: containers
# ---------------------------------------------------------------------------


class TestContainers(unittest.TestCase):
    def test_detects_png(self):
        self.assertEqual(containers.sniff_container(export.build_placeholder_png()), "png")

    def test_detects_json(self):
        self.assertEqual(containers.sniff_container(b'{"spec": "x"}'), "json")

    def test_detects_charx(self):
        blob = export.write_charx(ccv3.read_text(json.dumps(v3_card_dict())))
        self.assertEqual(containers.sniff_container(blob), "charx")

    def test_rejects_unknown(self):
        with self.assertRaises(ccv3.ContainerError):
            containers.load(b"this is not a card")

    def test_rejects_unknown_with_informative_message(self):
        try:
            containers.load(b"not a card at all")
        except ccv3.ContainerError as exc:
            self.assertIn("unrecognised container", str(exc))
        else:
            self.fail("expected ContainerError")

    def test_reads_json_container(self):
        card = ccv3.read_text(json.dumps(v3_card_dict()))
        self.assertEqual(card.container, "json")
        self.assertEqual(card.name, "Aria")

    def test_reads_png_container(self):
        card = ccv3.read(png_with_card(v3_card_dict()))
        self.assertEqual(card.container, "png")
        self.assertEqual(card.origin_spec, "chara_card_v3")

    def test_prefers_ccv3_over_chara(self):
        """When a PNG carries both chunks, the V3 one wins."""
        blob = png_with_card(v3_card_dict(name="FromV3"), "ccv3", also=v2_card_dict(name="FromV2"))
        card = ccv3.read(blob)
        self.assertEqual(card.name, "FromV3")
        self.assertFalse(card.backfilled)

    def test_falls_back_to_chara_and_flags_backfill(self):
        blob = png_with_card(v2_card_dict(name="Legacy"), "chara")
        card = ccv3.read(blob)
        self.assertEqual(card.name, "Legacy")
        self.assertTrue(card.backfilled)
        self.assertTrue(any("chara" in w for w in card.warnings))

    def test_apng_is_distinguished(self):
        """An APNG is a PNG plus an acTL chunk, and the reader should say so."""
        image = export.build_placeholder_png()
        acTL = png.build_chunk(b"acTL", b"\x00\x00\x00\x01\x00\x00\x00\x00")
        # Insert after IHDR, which is where the specification requires it.
        offset = 8 + 8 + 13 + 4
        apng = image[:offset] + acTL + image[offset:]
        self.assertEqual(containers.sniff_container(apng), "apng")
        card = ccv3.read(png_with_card(v3_card_dict()))
        self.assertEqual(card.container, "png", "an ordinary PNG is still just a PNG")

    def test_charx_round_trip(self):
        card = ccv3.read_text(json.dumps(v3_card_dict()))
        card.data.assets = [
            Asset(type="icon", uri="embeded://assets/icon/images/main.png", name="main", ext="png")
        ]
        blob = export.write_charx(
            card, {"assets/icon/images/main.png": b"\x89PNG pretend bytes"}
        )
        reread = ccv3.read(blob)
        self.assertEqual(reread.container, "charx")
        self.assertEqual(reread.data.assets[0].embedded_path, "assets/icon/images/main.png")
        self.assertIn("assets/icon/images/main.png", reread.asset_blobs)

    def test_charx_without_card_json_reports_clearly(self):
        import io
        import zipfile

        buffer = io.BytesIO()
        with zipfile.ZipFile(buffer, "w") as archive:
            archive.writestr("something.txt", b"hello")
        with self.assertRaises(ccv3.MalformedCardError) as ctx:
            ccv3.read(buffer.getvalue())
        self.assertIn("card.json", str(ctx.exception))

    def test_self_extracting_charx_is_found(self):
        """A CHARX with a stub in front of the ZIP should still open."""
        card = ccv3.read_text(json.dumps(v3_card_dict()))
        blob = b"MZ this is a fake executable stub" + export.write_charx(card)
        reread = ccv3.read(blob)
        self.assertEqual(reread.name, "Aria")


# ---------------------------------------------------------------------------
# Step three: version detection
# ---------------------------------------------------------------------------


class TestVersionDetection(unittest.TestCase):
    def test_detects_v3(self):
        spec, ver, _ = version.detect_version(v3_card_dict())
        self.assertEqual(spec, ccv3.SPEC_V3)
        self.assertEqual(ver, 3.0)

    def test_detects_v2(self):
        spec, ver, _ = version.detect_version(v2_card_dict())
        self.assertEqual(spec, ccv3.SPEC_V2)

    def test_detects_v1(self):
        spec, ver, _ = version.detect_version(v1_card_dict())
        self.assertEqual(spec, ccv3.SPEC_V1)

    def test_detects_lorebook(self):
        spec, _ver, _ = version.detect_version({"spec": "lorebook_v3", "data": lorebook_dict([])})
        self.assertEqual(spec, ccv3.SPEC_LOREBOOK_V3)

    def test_numeric_spec_version_is_accepted(self):
        """The dominant implementation writes the number, not the string."""
        obj = v2_card_dict()
        obj["spec_version"] = 2.0
        spec, ver, _ = version.detect_version(obj)
        self.assertEqual(spec, ccv3.SPEC_V2)
        self.assertEqual(ver, 2.0)

    def test_missing_spec_version_is_inferred_from_shape(self):
        obj = v2_card_dict()
        del obj["spec_version"]
        del obj["spec"]
        spec, ver, warnings = version.detect_version(obj)
        self.assertEqual(spec, ccv3.SPEC_V2)
        self.assertTrue(warnings, "expected a warning about the guess")

    def test_v3_only_fields_identify_v3_without_a_version(self):
        obj = v2_card_dict()
        del obj["spec"]
        del obj["spec_version"]
        obj["data"]["assets"] = []
        spec, _ver, warnings = version.detect_version(obj)
        self.assertEqual(spec, ccv3.SPEC_V3)
        self.assertTrue(any("V3-only" in w for w in warnings))

    def test_unrecognised_spec_string_is_read_by_number(self):
        obj = v3_card_dict()
        obj["spec"] = "some_other_app_v3"
        spec, ver, warnings = version.detect_version(obj)
        self.assertEqual(spec, ccv3.SPEC_V3)
        self.assertTrue(any("unrecognised spec" in w for w in warnings))

    def test_newer_major_warns_but_imports(self):
        obj = v3_card_dict()
        obj["spec"] = "chara_card_v4"
        obj["spec_version"] = "4.0"
        spec, ver, warnings = version.detect_version(obj)
        self.assertEqual(ver, 4.0)
        self.assertTrue(any("newer" in w for w in warnings))

    def test_non_card_is_rejected(self):
        with self.assertRaises(ccv3.UnknownSpecError):
            version.detect_version({"hello": "world"})

    def test_parse_version_rejects_booleans(self):
        self.assertIsNone(version.parse_version(True))
        self.assertEqual(version.parse_version("3.0"), 3.0)
        self.assertEqual(version.parse_version(3), 3.0)


# ---------------------------------------------------------------------------
# Step five: normalisation, and V2 retro-compatibility
# ---------------------------------------------------------------------------


class TestNormalisation(unittest.TestCase):
    def test_v3_reads_with_origin_recorded(self):
        card = ccv3.read_text(json.dumps(v3_card_dict()))
        self.assertEqual(card.origin_spec, ccv3.SPEC_V3)
        self.assertEqual(card.origin_version, 3.0)
        self.assertEqual(card.spec, ccv3.SPEC_V3, "normalised cards are always V3-shaped")

    def test_v2_card_is_lifted_into_the_v3_shape(self):
        card = ccv3.read_text(json.dumps(v2_card_dict()))
        self.assertEqual(card.origin_spec, ccv3.SPEC_V2)
        # The V3-only fields exist and are empty rather than missing.
        self.assertEqual(card.data.assets, [])
        self.assertEqual(card.data.group_only_greetings, [])
        self.assertEqual(card.data.nickname, "")
        self.assertIsNone(card.data.creation_date)

    def test_v1_card_is_lifted_into_the_v3_shape(self):
        card = ccv3.read_text(json.dumps(v1_card_dict()))
        self.assertEqual(card.origin_spec, ccv3.SPEC_V1)
        self.assertEqual(card.data.name, "Aria")
        self.assertEqual(card.data.first_mes, "Hello there.")
        self.assertEqual(card.data.tags, [])

    def test_all_three_versions_yield_the_same_field_set(self):
        """The point of normalising upward: consumers never need to know the version."""
        fields = []
        for source in (v1_card_dict(), v2_card_dict(), v3_card_dict()):
            card = ccv3.read_text(json.dumps(source))
            fields.append(sorted(vars(card.data).keys()))
        self.assertEqual(fields[0], fields[1])
        self.assertEqual(fields[1], fields[2])

    def test_unknown_data_fields_are_preserved(self):
        card = ccv3.read_text(
            json.dumps(v3_card_dict(extra_data={"x_future_field": {"nested": [1, 2, 3]}}))
        )
        self.assertIn("x_future_field", card.data.extra)
        self.assertEqual(card.data.extra["x_future_field"], {"nested": [1, 2, 3]})

    def test_unknown_card_fields_are_preserved(self):
        obj = v3_card_dict()
        obj["x_top_level"] = "kept"
        card = ccv3.read_text(json.dumps(obj))
        self.assertEqual(card.extra["x_top_level"], "kept")

    def test_round_trip_keeps_unknown_fields(self):
        obj = v3_card_dict(extra_data={"x_future_field": "survives"})
        obj["x_top_level"] = "also survives"
        card = ccv3.read_text(json.dumps(obj))
        written = export.to_v3_dict(card)
        self.assertEqual(written["data"]["x_future_field"], "survives")
        self.assertEqual(written["x_top_level"], "also survives")

    def test_display_name_prefers_nickname(self):
        card = ccv3.read_text(json.dumps(v3_card_dict(extra_data={"nickname": "Ari"})))
        self.assertEqual(card.data.name, "Aria")
        self.assertEqual(card.display_name, "Ari")

    def test_empty_nickname_falls_back_to_name(self):
        card = ccv3.read_text(json.dumps(v3_card_dict(extra_data={"nickname": ""})))
        self.assertEqual(card.display_name, "Aria")

    def test_type_coercion_handles_a_comma_separated_tag_string(self):
        card = ccv3.read_text(
            json.dumps(v3_card_dict(extra_data={"tags": "one, two,three"}))
        )
        self.assertEqual(card.data.tags, ["one", "two", "three"])

    def test_timestamp_of_zero_means_absent(self):
        card = ccv3.read_text(json.dumps(v3_card_dict(extra_data={"creation_date": 0})))
        self.assertIsNone(card.data.creation_date)

    def test_millisecond_timestamp_is_converted(self):
        """Real cards from a common export tool carry thirteen-digit milliseconds."""
        card = ccv3.read_text(
            json.dumps(v3_card_dict(extra_data={"creation_date": 1790435629079}))
        )
        self.assertEqual(card.data.creation_date, 1790435629)

    def test_microsecond_timestamp_is_converted(self):
        card = ccv3.read_text(
            json.dumps(v3_card_dict(extra_data={"creation_date": 1790435629079000}))
        )
        self.assertEqual(card.data.creation_date, 1790435629)

    def test_a_plausible_second_timestamp_is_left_alone(self):
        card = ccv3.read_text(
            json.dumps(v3_card_dict(extra_data={"creation_date": 1700000000}))
        )
        self.assertEqual(card.data.creation_date, 1700000000)

    def test_iso_string_timestamp_is_parsed(self):
        card = ccv3.read_text(
            json.dumps(v3_card_dict(extra_data={"creation_date": "2024-01-02T03:04:05Z"}))
        )
        self.assertEqual(card.data.creation_date, 1704164645)

    def test_nested_metadata_dates_are_read(self):
        """The V1-era convention: dates under metadata, in milliseconds."""
        obj = v1_card_dict()
        obj["metadata"] = {"created": 1790435629079, "modified": 1790435639079, "version": 1}
        card = ccv3.read_text(json.dumps(obj))
        self.assertEqual(card.data.creation_date, 1790435629)
        self.assertEqual(card.data.modification_date, 1790435639)

    def test_application_style_metadata_dates_are_read(self):
        """The other convention: the same idea under different key names, as ISO strings."""
        obj = v2_card_dict()
        obj["metadata"] = {
            "create_date": "2024-01-02T03:04:05Z",
            "modify_date": "2024-02-03T04:05:06Z",
        }
        card = ccv3.read_text(json.dumps(obj))
        self.assertEqual(card.data.creation_date, 1704164645)
        self.assertEqual(card.data.modification_date, 1706933106)

    def test_top_level_create_date_is_read(self):
        obj = v2_card_dict()
        obj["create_date"] = "2024-01-02T03:04:05Z"
        card = ccv3.read_text(json.dumps(obj))
        self.assertEqual(card.data.creation_date, 1704164645)

    def test_the_specification_field_beats_the_metadata_convention(self):
        obj = v1_card_dict()
        obj["metadata"] = {"created": 1790435629079}
        obj["creation_date"] = 1700000000
        card = ccv3.read_text(json.dumps(obj))
        self.assertEqual(card.data.creation_date, 1700000000)

    def test_v1_flat_extension_fields_are_kept(self):
        obj = v1_card_dict()
        obj["talkativeness"] = 0.7
        card = ccv3.read_text(json.dumps(obj))
        self.assertEqual(card.data.extensions.get("talkativeness"), 0.7)

    def test_v1_lorebook_is_carried_across(self):
        obj = v1_card_dict()
        obj["character_book"] = lorebook_dict([entry(keys=["sword"], content="A blade.")])
        card = ccv3.read_text(json.dumps(obj))
        self.assertIsNotNone(card.lorebook)
        self.assertEqual(card.lorebook.entries[0].content, "A blade.")


# ---------------------------------------------------------------------------
# V2 retro-compatibility on the writing side
# ---------------------------------------------------------------------------


class TestV2Backfill(unittest.TestCase):
    def make_card_with_decorated_lorebook(self) -> Card:
        obj = v3_card_dict()
        obj["data"]["character_book"] = lorebook_dict(
            [
                entry(
                    keys=["magic"],
                    content="@@activate_only_after 2\n@@@activate_every\nReal lore content.",
                )
            ]
        )
        return ccv3.read_text(json.dumps(obj))

    def test_v2_export_strips_decorators(self):
        """The specification's instruction: on backfilling V2, remove all decorators."""
        card = self.make_card_with_decorated_lorebook()
        v2 = export.to_v2_dict(card)
        content = v2["data"]["character_book"]["entries"][0]["content"]
        self.assertNotIn("@@", content)
        self.assertEqual(content, "Real lore content.")

    def test_v3_export_keeps_decorators(self):
        card = self.make_card_with_decorated_lorebook()
        v3 = export.to_v3_dict(card)
        content = v3["data"]["character_book"]["entries"][0]["content"]
        self.assertIn("@@activate_only_after 2", content)

    def test_v2_export_keeps_v3_fields_by_default(self):
        obj = v3_card_dict(extra_data={"nickname": "Ari", "assets": []})
        card = ccv3.read_text(json.dumps(obj))
        v2 = export.to_v2_dict(card)
        self.assertEqual(v2["spec"], "chara_card_v2")
        self.assertEqual(v2["data"]["nickname"], "Ari")

    def test_v2_export_can_drop_v3_fields(self):
        obj = v3_card_dict(extra_data={"nickname": "Ari"})
        card = ccv3.read_text(json.dumps(obj))
        v2 = export.to_v2_dict(card, keep_v3_fields=False)
        self.assertNotIn("nickname", v2["data"])
        self.assertNotIn("group_only_greetings", v2["data"])

    def test_v2_export_has_all_fourteen_required_fields(self):
        card = ccv3.read_text(json.dumps(v3_card_dict()))
        v2 = export.to_v2_dict(card)
        for field in export.V2_REQUIRED_DATA_FIELDS:
            self.assertIn(field, v2["data"], f"missing {field}")

    def test_backfill_notice_can_be_added(self):
        card = ccv3.read_text(json.dumps(v3_card_dict()))
        v2 = export.to_v2_dict(card, add_backfill_notice=True)
        self.assertTrue(v2["data"]["creator_notes"].startswith(export.BACKFILL_NOTICE))

    def test_v2_export_validates_as_v2(self):
        """The strongest V2 test: what we write, our own V2 validator accepts."""
        card = ccv3.read_text(json.dumps(v3_card_dict()))
        v2 = export.to_v2_dict(card)
        report = validate.validate_card_dict(v2, spec=ccv3.SPEC_V2, version=2.0)
        self.assertTrue(report.ok, f"errors: {report.messages()}")

    def test_v3_survives_a_v2_round_trip(self):
        """A card exported as V2 and read back still knows it was V3 in substance."""
        obj = v3_card_dict(extra_data={"nickname": "Ari"})
        card = ccv3.read_text(json.dumps(obj))
        v2_blob = export.to_json_bytes(card, "chara_card_v2")
        reread = ccv3.read(v2_blob)
        self.assertEqual(reread.origin_spec, ccv3.SPEC_V2)
        self.assertEqual(reread.data.nickname, "Ari", "V3 fields ride along inside the V2 object")

    def test_png_writer_emits_both_chunks(self):
        card = ccv3.read_text(json.dumps(v3_card_dict()))
        blob = export.write_png(card)
        chunks, _warnings = png.read_text_chunks(blob)
        self.assertIn("ccv3", chunks)
        self.assertIn("chara", chunks)

    def test_png_round_trip_is_lossless_for_known_fields(self):
        card = ccv3.read_text(json.dumps(v3_card_dict()))
        original = card.data.name
        blob = export.write_png(card)
        reread = ccv3.read(blob)
        self.assertEqual(reread.data.name, original)

    def test_rewriting_a_png_does_not_leave_two_ccv3_chunks(self):
        """A second ccv3 chunk would shadow the first, so the writer must replace."""
        card = ccv3.read_text(json.dumps(v3_card_dict(name="First")))
        blob = export.write_png(card)
        card.data.name = "Second"
        blob = export.write_png(card, base_image=blob)
        reread = ccv3.read(blob)
        self.assertEqual(reread.data.name, "Second")
        chunks, _ = png.read_text_chunks(blob)
        declared = sum(
            1
            for chunk_type, payload, _ in png.iter_chunks(blob)
            if chunk_type == png.CHUNK_TEXT and payload.startswith(b"ccv3\x00")
        )
        self.assertEqual(declared, 1)


# ---------------------------------------------------------------------------
# Decorators
# ---------------------------------------------------------------------------


class TestDecorators(unittest.TestCase):
    def test_parses_a_simple_decorator(self):
        parsed = decorators.parse("@@activate_only_after 3\nReal content.")
        self.assertEqual(parsed.cleaned, "Real content.")
        self.assertEqual(parsed.get("activate_only_after").as_int(), 3)

    def test_parses_a_valueless_decorator(self):
        parsed = decorators.parse("@@activate\nContent.")
        self.assertTrue(parsed.has("activate"))
        self.assertEqual(parsed.cleaned, "Content.")

    def test_parses_multiple_values(self):
        parsed = decorators.parse("@@exclude_keys moon,sun\nContent.")
        self.assertEqual(parsed.get("exclude_keys").values, ["moon", "sun"])

    def test_takes_the_first_of_a_fallback_chain(self):
        content = "@@risu_only 4\n@@@agn_only 4\n@@@activate_only_after 4\nContent."
        parsed = decorators.parse(content)
        self.assertTrue(parsed.has("activate_only_after"))
        self.assertFalse(parsed.has("risu_only"))
        self.assertEqual(parsed.cleaned, "Content.")

    def test_a_wholly_unknown_chain_leaves_content_alone(self):
        parsed = decorators.parse("@@unknown_one 1\n@@@unknown_two 2\nContent.")
        self.assertEqual(parsed.cleaned, "Content.")
        self.assertEqual(parsed.directives, {})

    def test_duplicate_decorators_use_the_first(self):
        parsed = decorators.parse("@@scan_depth 2\n@@scan_depth 9\nContent.")
        self.assertEqual(parsed.get("scan_depth").as_int(), 2)
        self.assertTrue(any("more than once" in w for w in parsed.warnings))

    def test_additional_keys_may_repeat(self):
        parsed = decorators.parse("@@additional_keys a\n@@additional_keys b\nContent.")
        self.assertEqual(len(parsed.all("additional_keys")), 2)

    def test_invalid_number_is_ignored(self):
        parsed = decorators.parse("@@scan_depth notanumber\nContent.")
        self.assertFalse(parsed.has("scan_depth"))
        self.assertTrue(any("requires a number" in w for w in parsed.warnings))

    def test_invalid_position_value_is_ignored(self):
        parsed = decorators.parse("@@position sideways\nContent.")
        self.assertFalse(parsed.has("position"))

    def test_decorators_only_count_at_the_start(self):
        """A decorator after the first line of content is ordinary text, not a decorator."""
        parsed = decorators.parse("Line one.\n@@activate\nLine three.")
        self.assertEqual(parsed.cleaned, "Line one.\n@@activate\nLine three.")
        self.assertEqual(parsed.directives, {})

    def test_a_trailing_decorator_is_not_parsed(self):
        parsed = decorators.parse("Content first.\n@@activate")
        self.assertEqual(parsed.cleaned, "Content first.\n@@activate")
        self.assertFalse(parsed.has("activate"))

    def test_blank_lines_before_the_block_are_tolerated(self):
        parsed = decorators.parse("\n\n@@activate\nContent.")
        self.assertTrue(parsed.has("activate"))
        self.assertEqual(parsed.cleaned, "Content.")

    def test_strip_all_removes_everything(self):
        content = "@@unrecognised 1\n@@@also_unrecognised 2\n@@activate\nContent."
        self.assertEqual(decorators.strip_all(content), "Content.")

    def test_strip_all_leaves_mid_content_decorators(self):
        content = "Start.\n@@activate\nMore."
        self.assertEqual(decorators.strip_all(content), content)

    def test_all_twenty_names_are_recognised(self):
        self.assertEqual(len(decorators.KNOWN_DECORATOR_NAMES), 20)


# ---------------------------------------------------------------------------
# Curly braced syntaxes
# ---------------------------------------------------------------------------


class TestCbs(unittest.TestCase):
    def test_char_falls_back_to_name(self):
        self.assertEqual(cbs.render("{{char}}", char_name="Aria").text, "Aria")

    def test_user_is_substituted(self):
        self.assertEqual(cbs.render("{{user}}", user_name="Sam").text, "Sam")

    def test_random_picks_one_of_the_values(self):
        for _ in range(20):
            self.assertIn(cbs.render("{{random:alpha,beta,gamma}}").text, ("alpha", "beta", "gamma"))

    def test_random_respects_escaped_commas(self):
        result = cbs.render("{{random:a\\,b}}").text
        self.assertEqual(result, "a,b")

    def test_pick_is_stable_for_the_same_input(self):
        text = "{{pick:alpha,beta,gamma}}"
        first = cbs.render(text).text
        for _ in range(10):
            self.assertEqual(cbs.render(text).text, first)

    def test_roll_within_range(self):
        for _ in range(40):
            value = int(cbs.render("{{roll:6}}").text)
            self.assertGreaterEqual(value, 1)
            self.assertLessEqual(value, 6)

    def test_roll_accepts_the_d_notation(self):
        for _ in range(20):
            value = int(cbs.render("{{roll:d20}}").text)
            self.assertGreaterEqual(value, 1)
            self.assertLessEqual(value, 20)

    def test_double_slash_comment_is_removed(self):
        result = cbs.render("A {{// hidden note}} B")
        self.assertEqual(result.text, "A  B")
        self.assertEqual(result.hidden_keys, [])

    def test_hidden_key_is_removed_but_kept_for_matching(self):
        result = cbs.render("A {{hidden_key:sword}} B")
        self.assertEqual(result.text, "A  B")
        self.assertEqual(result.hidden_keys, ["sword"])
        self.assertIn("sword", result.matchable_text)

    def test_comment_is_removed_and_reported(self):
        result = cbs.render("A {{comment:author note}} B")
        self.assertEqual(result.text, "A  B")
        self.assertEqual(result.comments, ["author note"])
        self.assertNotIn("author note", result.matchable_text)

    def test_reverse(self):
        self.assertEqual(cbs.render("{{reverse:Hello}}").text, "olleH")

    def test_unknown_macro_is_left_alone(self):
        result = cbs.render("{{some_other_app_thing}}")
        self.assertEqual(result.text, "{{some_other_app_thing}}")
        self.assertIn("some_other_app_thing", result.unknown)

    def test_nested_macros_resolve_inner_first(self):
        """The outer macro must see the inner one already substituted.

        Both cases here are deterministic on purpose. A nested `{{random}}` with two options
        would be a coin toss and would prove nothing, so the choices are a single-item list
        and a reversal, where the only correct answer is the one that shows the inner macro
        was resolved before the outer one ran.
        """
        single = cbs.render("{{random:{{user}}}}", user_name="Sam")
        self.assertEqual(single.text, "Sam")

        reversed_result = cbs.render("{{reverse:{{user}}}}", user_name="Sam")
        self.assertEqual(reversed_result.text, "maS")

    def test_case_insensitive_detection(self):
        self.assertEqual(cbs.render("{{CHAR}}", char_name="Aria").text, "Aria")

    def test_text_without_macros_is_untouched(self):
        self.assertEqual(cbs.render("plain text").text, "plain text")


# ---------------------------------------------------------------------------
# The scanner
# ---------------------------------------------------------------------------


class TestLorebookScan(unittest.TestCase):
    def book(self, entries: list[dict], **book_fields) -> Lorebook:
        raw = lorebook_dict(entries)
        raw.update(book_fields)
        warnings: list[str] = []
        return normalise.normalise_lorebook(raw, warnings=warnings, where="test")

    def test_a_matching_key_injects_content(self):
        book = self.book([entry(keys=["dragon"], content="Dragons are real.")])
        result = lorebook.scan(book, lorebook.ScanContext(messages=["I saw a dragon"]))
        self.assertEqual(len(result.hits), 1)
        self.assertEqual(result.injection_text(), "Dragons are real.")
        self.assertEqual(result.hits[0].matched_keys, ["dragon"])

    def test_a_missing_key_injects_nothing(self):
        book = self.book([entry(keys=["dragon"], content="Dragons are real.")])
        result = lorebook.scan(book, lorebook.ScanContext(messages=["nothing here"]))
        self.assertEqual(result.hits, [])

    def test_disabled_entries_never_match(self):
        book = self.book([entry(keys=["dragon"], content="Hidden.", enabled=False)])
        result = lorebook.scan(book, lorebook.ScanContext(messages=["dragon"]))
        self.assertEqual(result.hits, [])

    def test_content_is_injected_once_even_when_keys_repeat(self):
        book = self.book([entry(keys=["dragon"], content="Once only.")])
        context = lorebook.ScanContext(messages=["dragon", "dragon", "dragon"])
        result = lorebook.scan(book, context)
        self.assertEqual(len(result.hits), 1)
        self.assertEqual(result.injection_text().count("Once only."), 1)

    def test_constant_entries_need_no_key(self):
        book = self.book([entry(keys=[], content="Always here.", constant=True)])
        result = lorebook.scan(book, lorebook.ScanContext(messages=["unrelated"]))
        self.assertEqual(len(result.hits), 1)

    def test_empty_content_contributes_nothing(self):
        book = self.book([entry(keys=["dragon"], content="")])
        result = lorebook.scan(book, lorebook.ScanContext(messages=["dragon"]))
        self.assertEqual(result.hits, [])
        self.assertEqual(len(result.empty), 0, "an entry with no content is skipped before matching")

    def test_scan_depth_limits_how_far_back_matching_looks(self):
        book = self.book([entry(keys=["dragon"], content="Old news.")], scan_depth=1)
        result = lorebook.scan(
            book, lorebook.ScanContext(messages=["dragon", "later", "newest"])
        )
        self.assertEqual(result.hits, [], "the dragon is outside the scan depth")

    def test_entry_scan_depth_decorator_overrides_the_book(self):
        book = self.book(
            [entry(keys=["dragon"], content="@@scan_depth 3\nOld news.")], scan_depth=1
        )
        result = lorebook.scan(
            book, lorebook.ScanContext(messages=["dragon", "later", "newest"])
        )
        self.assertEqual(len(result.hits), 1)

    def test_case_insensitive_by_default(self):
        book = self.book([entry(keys=["dragon"], content="Match.")])
        result = lorebook.scan(book, lorebook.ScanContext(messages=["DRAGON"]))
        self.assertEqual(len(result.hits), 1)

    def test_case_sensitive_entries_respect_case(self):
        book = self.book([entry(keys=["dragon"], content="Match.", case_sensitive=True)])
        result = lorebook.scan(book, lorebook.ScanContext(messages=["DRAGON"]))
        self.assertEqual(result.hits, [])

    def test_use_regex_matches_a_pattern(self):
        book = self.book([entry(keys=[r"drag[oa]n"], content="Regex match.", use_regex=True)])
        result = lorebook.scan(book, lorebook.ScanContext(messages=["a dragon appeared"]))
        self.assertEqual(len(result.hits), 1)

    def test_invalid_regex_is_a_non_match_not_an_error(self):
        book = self.book([entry(keys=["[unclosed"], content="Never.", use_regex=True)])
        result = lorebook.scan(book, lorebook.ScanContext(messages=["[unclosed"]))
        self.assertEqual(result.hits, [])

    def test_selective_entries_need_the_secondary_key(self):
        book = self.book(
            [
                entry(
                    keys=["dragon"],
                    content="Both.",
                    selective=True,
                    secondary_keys=["mountain"],
                )
            ]
        )
        only_primary = lorebook.scan(book, lorebook.ScanContext(messages=["dragon"]))
        self.assertEqual(only_primary.hits, [])
        both = lorebook.scan(book, lorebook.ScanContext(messages=["dragon in a mountain"]))
        self.assertEqual(len(both.hits), 1)

    def test_exclude_keys_vetoes_a_match(self):
        book = self.book(
            [entry(keys=["dragon"], content="@@exclude_keys friendly\nScary dragon lore.")]
        )
        vetoed = lorebook.scan(book, lorebook.ScanContext(messages=["a friendly dragon"]))
        self.assertEqual(vetoed.hits, [])
        allowed = lorebook.scan(book, lorebook.ScanContext(messages=["a dragon"]))
        self.assertEqual(len(allowed.hits), 1)

    def test_exclude_keys_is_ignored_when_use_regex_is_on(self):
        book = self.book(
            [
                entry(
                    keys=["dragon"],
                    content="@@exclude_keys friendly\nLore.",
                    use_regex=True,
                )
            ]
        )
        result = lorebook.scan(book, lorebook.ScanContext(messages=["friendly dragon"]))
        self.assertEqual(len(result.hits), 1)

    def test_additional_keys_widen_the_match(self):
        book = self.book([entry(keys=["dragon"], content="@@additional_keys wyrm\nLore.")])
        result = lorebook.scan(book, lorebook.ScanContext(messages=["a wyrm"]))
        self.assertEqual(len(result.hits), 1)

    def test_activate_forces_a_match(self):
        book = self.book([entry(keys=["never"], content="@@activate\nForced.")])
        result = lorebook.scan(book, lorebook.ScanContext(messages=["nothing"]))
        self.assertEqual(len(result.hits), 1)

    def test_dont_activate_forbids_a_match(self):
        book = self.book([entry(keys=["dragon"], content="@@dont_activate\nNever.")])
        result = lorebook.scan(book, lorebook.ScanContext(messages=["dragon"]))
        self.assertEqual(result.hits, [])

    def test_activate_beats_dont_activate(self):
        book = self.book(
            [entry(keys=["never"], content="@@dont_activate\n@@activate\nForced anyway.")]
        )
        result = lorebook.scan(book, lorebook.ScanContext(messages=["nothing"]))
        self.assertEqual(len(result.hits), 1)

    def test_activate_only_after_gates_on_message_count(self):
        book = self.book(
            [entry(keys=["dragon"], content="@@activate_only_after 3\nLater.")]
        )
        early = lorebook.scan(
            book,
            lorebook.ScanContext(messages=["dragon"], assistant_message_count=1),
        )
        self.assertEqual(early.hits, [])
        late = lorebook.scan(
            book,
            lorebook.ScanContext(messages=["dragon"], assistant_message_count=3),
        )
        self.assertEqual(len(late.hits), 1)

    def test_activate_only_every_gates_on_divisibility(self):
        book = self.book(
            [entry(keys=["dragon"], content="@@activate_only_every 2\nEvery other.")]
        )
        odd = lorebook.scan(
            book, lorebook.ScanContext(messages=["dragon"], assistant_message_count=3)
        )
        self.assertEqual(odd.hits, [])
        even = lorebook.scan(
            book, lorebook.ScanContext(messages=["dragon"], assistant_message_count=4)
        )
        self.assertEqual(len(even.hits), 1)

    def test_is_greeting_gates_on_the_active_greeting(self):
        book = self.book([entry(keys=["dragon"], content="@@is_greeting 1\nAlt lore.")])
        default_greeting = lorebook.scan(
            book, lorebook.ScanContext(messages=["dragon"], greeting_index=0)
        )
        self.assertEqual(default_greeting.hits, [])
        alternate = lorebook.scan(
            book, lorebook.ScanContext(messages=["dragon"], greeting_index=1)
        )
        self.assertEqual(len(alternate.hits), 1)

    def test_keep_activate_after_match_persists(self):
        book = self.book(
            [entry(keys=["dragon"], content="@@keep_activate_after_match\nSticky.")]
        )
        result = lorebook.scan(
            book,
            lorebook.ScanContext(messages=["nothing relevant"], prior_matches=frozenset({0})),
        )
        self.assertEqual(len(result.hits), 1)

    def test_dont_activate_after_match_stops(self):
        book = self.book(
            [entry(keys=["dragon"], content="@@dont_activate_after_match\nOnce.")]
        )
        result = lorebook.scan(
            book,
            lorebook.ScanContext(messages=["dragon"], prior_matches=frozenset({0})),
        )
        self.assertEqual(result.hits, [])

    def test_insertion_order_decides_the_order(self):
        book = self.book(
            [
                entry(keys=["a"], content="Second.", insertion_order=20),
                entry(keys=["a"], content="First.", insertion_order=10),
            ]
        )
        result = lorebook.scan(book, lorebook.ScanContext(messages=["a"]))
        self.assertEqual(result.injection_text().split("\n\n"), ["First.", "Second."])

    def test_token_budget_drops_the_lowest_priority_first(self):
        book = self.book(
            [
                entry(keys=["a"], content="High priority content here.", priority=10),
                entry(keys=["a"], content="Low priority content here.", priority=1),
            ],
            token_budget=8,
        )
        result = lorebook.scan(book, lorebook.ScanContext(messages=["a"]))
        self.assertEqual(len(result.hits), 1)
        self.assertIn("High priority", result.hits[0].content)
        self.assertEqual(len(result.dropped), 1)
        self.assertIsNotNone(result.dropped[0].dropped_reason)

    def test_token_budget_falls_back_to_insertion_order(self):
        """With no priority set, the specification says drop the lowest insertion order."""
        book = self.book(
            [
                entry(keys=["a"], content="Later content.", insertion_order=5),
                entry(keys=["a"], content="Earlier content.", insertion_order=1),
            ],
            token_budget=6,
        )
        result = lorebook.scan(book, lorebook.ScanContext(messages=["a"]))
        self.assertEqual(len(result.hits), 1)
        self.assertIn("Later content", result.hits[0].content)

    def test_a_budget_that_fits_drops_nothing(self):
        book = self.book(
            [entry(keys=["a"], content="Short.")], token_budget=1000
        )
        result = lorebook.scan(book, lorebook.ScanContext(messages=["a"]))
        self.assertEqual(len(result.hits), 1)
        self.assertEqual(result.dropped, [])

    def test_custom_tokeniser_is_used(self):
        book = self.book([entry(keys=["a"], content="four tokens here")])
        context = lorebook.ScanContext(
            messages=["a"], tokenizer=lambda text: len(text.split())
        )
        result = lorebook.scan(book, context)
        self.assertEqual(result.hits[0].tokens, 3)

    def test_placement_decorators_are_reported_not_applied(self):
        book = self.book(
            [entry(keys=["a"], content="@@position after_desc\nPlaced content.")]
        )
        result = lorebook.scan(book, lorebook.ScanContext(messages=["a"]))
        self.assertEqual(len(result.hits), 1)
        self.assertTrue(result.hits[0].needs_prompt_placement)
        self.assertEqual(result.hits[0].directives["position"], ["after_desc"])

    def test_macros_are_rendered_in_injected_content(self):
        book = self.book([entry(keys=["a"], content="{{char}} speaks to {{user}}.")])
        result = lorebook.scan(
            book,
            lorebook.ScanContext(messages=["a"], char_name="Aria", user_name="Sam"),
        )
        self.assertEqual(result.hits[0].content, "Aria speaks to Sam.")

    def test_recursive_scanning_finds_a_second_entry(self):
        book = self.book(
            [
                entry(keys=["trigger"], content="The word is dragon.", insertion_order=1),
                entry(keys=["dragon"], content="Dragon lore reached recursively.", insertion_order=2),
            ],
            recursive_scanning=True,
        )
        result = lorebook.scan(book, lorebook.ScanContext(messages=["trigger"]))
        self.assertEqual(len(result.hits), 2)

    def test_recursion_off_by_default(self):
        book = self.book(
            [
                entry(keys=["trigger"], content="The word is dragon.", insertion_order=1),
                entry(keys=["dragon"], content="Should not appear.", insertion_order=2),
            ]
        )
        result = lorebook.scan(book, lorebook.ScanContext(messages=["trigger"]))
        self.assertEqual(len(result.hits), 1)

    def test_unknown_macro_in_content_is_reported(self):
        book = self.book([entry(keys=["a"], content="{{not_a_macro}}")])
        result = lorebook.scan(book, lorebook.ScanContext(messages=["a"]))
        self.assertTrue(any("not defined" in w for w in result.warnings))

    def test_empty_lorebook_scans_to_nothing(self):
        result = lorebook.scan(Lorebook(), lorebook.ScanContext(messages=["a"]))
        self.assertEqual(result.hits, [])
        self.assertEqual(result.total_tokens, 0)

    def test_matched_indices_round_trips_into_prior_matches(self):
        book = self.book([entry(keys=["a"], content="Once.")])
        first = lorebook.scan(book, lorebook.ScanContext(messages=["a"]))
        self.assertEqual(first.matched_indices, frozenset({0}))
        second = lorebook.scan(
            book, lorebook.ScanContext(messages=["a"], prior_matches=first.matched_indices)
        )
        self.assertEqual(len(second.hits), 1, "a plain entry matches again on a later turn")


# ---------------------------------------------------------------------------
# Validation
# ---------------------------------------------------------------------------


class TestOptionalFieldStates(unittest.TestCase):
    """The fields whose absence and whose explicit-`false` value mean different things.

    A real card carried `case_sensitive: false` on all thirty-one of its entries and
    `recursive_scanning: false` on its book. The specification gives each of those values a
    meaning distinct from leaving the field out, so a reader that collapses the two loses the
    author's intent on the way back out.
    """

    def book(self, entries: list[dict], **book_fields) -> Lorebook:
        raw = lorebook_dict(entries)
        raw.update(book_fields)
        warnings: list[str] = []
        return normalise.normalise_lorebook(raw, warnings=warnings, where="test")

    def test_an_explicit_false_is_not_confused_with_absence(self):
        entries = [entry(keys=["a"], case_sensitive=False), entry(keys=["b"])]
        book = self.book(entries)
        self.assertIs(book.entries[0].case_sensitive, False)
        self.assertIsNone(book.entries[1].case_sensitive)

    def test_an_explicit_false_survives_export(self):
        book = self.book([entry(keys=["a"], case_sensitive=False)])
        out = export.to_lorebook_v3(book)["data"]["entries"][0]
        self.assertIn("case_sensitive", out)
        self.assertIs(out["case_sensitive"], False)

    def test_an_absent_field_is_left_out_of_the_export(self):
        book = self.book([entry(keys=["a"])])
        out = export.to_lorebook_v3(book)["data"]["entries"][0]
        self.assertNotIn("case_sensitive", out)
        self.assertNotIn("selective", out)
        self.assertNotIn("position", out)

    def test_position_survives_in_both_directions(self):
        book = self.book([entry(keys=["a"], position="before_char")])
        self.assertEqual(book.entries[0].position, "before_char")
        out = export.to_lorebook_v3(book)["data"]["entries"][0]
        self.assertEqual(out["position"], "before_char")

    def test_an_unrecognised_position_is_dropped_with_a_warning(self):
        warnings: list[str] = []
        raw = lorebook_dict([entry(keys=["a"], position="somewhere_else")])
        book = normalise.normalise_lorebook(raw, warnings=warnings, where="test")
        self.assertIsNone(book.entries[0].position)
        self.assertTrue(any("position" in w for w in warnings))

    def test_recursive_scanning_keeps_its_three_states(self):
        self.assertIs(self.book([], recursive_scanning=False).recursive_scanning, False)
        self.assertIs(self.book([], recursive_scanning=True).recursive_scanning, True)
        self.assertIsNone(self.book([]).recursive_scanning)

    def test_an_explicit_recursive_false_survives_export(self):
        book = self.book([entry(keys=["a"])], recursive_scanning=False)
        out = export.to_lorebook_v3(book)["data"]
        self.assertIn("recursive_scanning", out)
        self.assertIs(out["recursive_scanning"], False)

    def test_an_absent_case_sensitive_still_matches_case_insensitively(self):
        """The application's own choice for an unspecified field, stated as a test."""
        # The content is not decoration. An entry with empty content is skipped before
        # matching, per the specification, so a test without it would pass or fail for a
        # reason unrelated to case sensitivity.
        book = self.book([entry(keys=["Dragon"], content="Dragons hoard.")])
        result = lorebook.scan(book, lorebook.ScanContext(messages=["a dragon appeared"]))
        self.assertEqual(len(result.hits), 1)

    def test_an_absent_recursive_scanning_does_not_recurse(self):
        """The conservative reading: the specification only permits recursion when told to."""
        book = self.book([
            entry(keys=["dragon"], content="Dragons hoard treasure."),
            entry(keys=["treasure"], content="Treasure is valuable."),
        ])
        result = lorebook.scan(book, lorebook.ScanContext(messages=["dragon"]))
        self.assertEqual([hit.entry_index for hit in result.hits], [0])


class TestValidation(unittest.TestCase):
    def test_a_complete_v3_card_validates(self):
        report = validate.validate_card_dict(v3_card_dict())
        self.assertTrue(report.ok, f"unexpected errors: {report.messages()}")

    def test_a_v3_card_missing_data_is_an_error(self):
        obj = v3_card_dict()
        del obj["data"]
        report = validate.validate_card_dict(obj)
        self.assertFalse(report.ok)

    def test_a_v3_card_missing_a_required_field_is_an_error(self):
        obj = v3_card_dict()
        del obj["data"]["first_mes"]
        report = validate.validate_card_dict(obj)
        self.assertFalse(report.ok)
        self.assertTrue(any("first_mes" in str(i) for i in report.errors))

    def test_a_missing_v3_only_field_is_only_a_warning(self):
        obj = v3_card_dict()
        del obj["data"]["group_only_greetings"]
        report = validate.validate_card_dict(obj)
        self.assertTrue(report.ok, "a missing V3 addition should not be an error")
        self.assertTrue(report.warnings)

    def test_bad_tag_type_is_an_error(self):
        obj = v3_card_dict()
        obj["data"]["tags"] = "not an array"
        report = validate.validate_card_dict(obj)
        self.assertFalse(report.ok)

    def test_duplicate_main_icons_are_an_error(self):
        obj = v3_card_dict()
        obj["data"]["assets"] = [
            {"type": "icon", "uri": "a.png", "name": "main", "ext": "png"},
            {"type": "icon", "uri": "b.png", "name": "main", "ext": "png"},
        ]
        report = validate.validate_card_dict(obj)
        self.assertFalse(report.ok)
        self.assertTrue(any("main" in str(i) for i in report.errors))

    def test_lorebook_entry_missing_keys_is_an_error(self):
        obj = v3_card_dict()
        obj["data"]["character_book"] = lorebook_dict([{"content": "x"}])
        report = validate.validate_card_dict(obj)
        self.assertFalse(report.ok)

    def test_bad_position_value_is_an_error(self):
        obj = v3_card_dict()
        obj["data"]["character_book"] = lorebook_dict(
            [entry(keys=["a"], content="x", position="middle")]
        )
        report = validate.validate_card_dict(obj)
        self.assertFalse(report.ok)

    def test_validation_warnings_reach_the_card(self):
        obj = v3_card_dict()
        del obj["data"]["group_only_greetings"]
        card = ccv3.read_text(json.dumps(obj))
        self.assertTrue(any("group_only_greetings" in w for w in card.warnings))

    def test_a_malformed_card_still_reads(self):
        """The library's position: report, do not refuse."""
        obj = v3_card_dict()
        obj["data"]["tags"] = "not an array"
        card = ccv3.read_text(json.dumps(obj))
        self.assertEqual(card.data.name, "Aria")


# ---------------------------------------------------------------------------
# Standalone lorebooks
# ---------------------------------------------------------------------------


class TestStandaloneLorebook(unittest.TestCase):
    def test_reads_a_standalone_lorebook(self):
        blob = json.dumps(
            {"spec": "lorebook_v3", "data": lorebook_dict([entry(keys=["a"], content="Lore.")])}
        ).encode()
        book = ccv3.read_lorebook(blob)
        self.assertEqual(len(book), 1)

    def test_exports_a_standalone_lorebook(self):
        card = ccv3.read_text(json.dumps(v3_card_dict()))
        card.data.character_book = Lorebook(entries=[LorebookEntry(keys=["a"], content="Lore.")])
        out = export.to_lorebook_v3(card)
        self.assertEqual(out["spec"], "lorebook_v3")
        self.assertEqual(len(out["data"]["entries"]), 1)

    def test_reading_a_lorebook_file_as_a_card_explains_itself(self):
        blob = json.dumps({"spec": "lorebook_v3", "data": lorebook_dict([])}).encode()
        with self.assertRaises(ccv3.MalformedCardError) as ctx:
            ccv3.read(blob)
        self.assertIn("read_lorebook", str(ctx.exception))


# ---------------------------------------------------------------------------
# Robustness
# ---------------------------------------------------------------------------


class TestRobustness(unittest.TestCase):
    def test_truncated_png_does_not_crash(self):
        blob = png_with_card(v3_card_dict())
        with self.assertRaises(ccv3.CCV3Error):
            ccv3.read(blob[:40])

    def test_base64_without_padding_is_read(self):
        payload = json.dumps(v3_card_dict()).encode("utf-8")
        text = containers.encode_base64_text(payload).rstrip("=")
        image = png.set_text_chunk(export.build_placeholder_png(), "ccv3", text)
        card = ccv3.read(image)
        self.assertEqual(card.name, "Aria")

    def test_empty_input_is_rejected(self):
        with self.assertRaises(ccv3.ContainerError):
            ccv3.read(b"")

    def test_non_object_json_is_rejected(self):
        with self.assertRaises(ccv3.CCV3Error):
            ccv3.read(b"[1, 2, 3]")

    def test_duplicate_ccv3_chunks_warn_and_use_the_first(self):
        """A stale duplicate must not win over the chunk that was written first."""
        first = containers.encode_base64_text(json.dumps(v3_card_dict(name="First")).encode())
        second = containers.encode_base64_text(json.dumps(v3_card_dict(name="Second")).encode())
        image = png.set_text_chunk(export.build_placeholder_png(), "ccv3", first)
        image = png.add_text_chunk(image, "ccv3", second)
        card = ccv3.read(image)
        self.assertEqual(card.name, "First", "the first chunk wins")
        self.assertTrue(any("more than one" in w for w in card.warnings))


if __name__ == "__main__":
    unittest.main(verbosity=2)

# ---------------------------------------------------------------------------
# Markdown rendering
# ---------------------------------------------------------------------------


class TestMarkdown(unittest.TestCase):
    def card(self, **extra_data):
        return ccv3.read_text(json.dumps(v3_card_dict(extra_data=extra_data or None)))

    def test_title_uses_the_display_name(self):
        card = self.card(nickname="Ari")
        self.assertTrue(markdown.to_markdown(card).startswith("# Ari\n"))

    def test_macros_are_translated_in_every_text_field(self):
        """Regression: macros used to render in mes_example but not in first_mes or scenario.

        Both fields are checked here because the bug was an inconsistency between them, and a
        test that only covered one would have passed.
        """
        card = self.card(
            description="{{char}} is described to {{user}}.",
            scenario="{{char}} meets {{user}}.",
            first_mes="Hello {{user}}, I am {{char}}.",
            mes_example="{{char}}: hi {{user}}",
        )
        document = markdown.to_markdown(card)
        # The provenance note mentions the macro names in prose, so only the card's own
        # content is checked for leftovers.
        body = document.split("## Provenance")[0]
        self.assertNotIn("{{char}}", body)
        self.assertNotIn("{{user}}", body)
        self.assertIn("Aria is described to you.", document)
        self.assertIn("Aria meets you.", document)
        self.assertIn("Hello you, I am Aria.", document)

    def test_raw_macros_flag_leaves_them_alone(self):
        card = self.card(first_mes="Hello {{user}}.")
        document = markdown.to_markdown(card, render_macros=False)
        self.assertIn("{{user}}", document)

    def test_user_name_is_configurable(self):
        card = self.card(first_mes="Hello {{user}}.")
        document = markdown.to_markdown(card, user_name="Thiago")
        self.assertIn("Hello Thiago.", document)

    def test_random_macros_are_shown_not_resolved(self):
        """Resolving a random macro would bake one draw into a document meant to be re-read."""
        card = self.card(first_mes="{{random:alpha,beta}}")
        document = markdown.to_markdown(card)
        self.assertIn("alpha", document)
        self.assertIn("beta", document)
        self.assertNotIn("{{random", document)

    def test_roll_is_shown_as_a_range(self):
        card = self.card(first_mes="{{roll:20}}")
        document = markdown.to_markdown(card)
        self.assertIn("1 to 20", document)

    def test_comment_macros_become_visible_notes(self):
        card = self.card(first_mes="A line. {{comment:a note}}")
        document = markdown.to_markdown(card)
        self.assertIn("a note", document)

    def test_hidden_keys_are_marked(self):
        card = self.card(first_mes="A line.{{hidden_key:sword}}")
        document = markdown.to_markdown(card)
        self.assertIn("hidden key: sword", document)

    def test_code_fence_grows_for_content_that_contains_a_fence(self):
        card = self.card(first_mes="here is a fence\n```\ninside it\n```")
        document = markdown.to_markdown(card)
        # The three-backtick sequence must open a fence that is longer than the inner run.
        self.assertIn("````", document)

    def test_crlf_line_endings_are_normalised(self):
        """Regression: a real card used Windows line endings and left stray CR bytes.

        The check reads the file in binary, because opening it in text mode translates newlines
        and would have reported zero carriage returns for a file full of them.
        """
        card = self.card(
            description="Line one.\r\nLine two.",
            first_mes="Hello.\r\nWelcome.",
        )
        document = markdown.to_markdown(card)
        self.assertNotIn("\r", document)

    def test_a_nameless_card_falls_back_to_its_filename(self):
        """A real card had a name of ten spaces, which produced a blank document title."""
        obj = v3_card_dict()
        obj["data"]["name"] = "          "
        card = ccv3.read_text(json.dumps(obj))
        document = markdown.to_markdown(card, source_name="main_bestiality-rpg.png")
        self.assertTrue(document.startswith("# main_bestiality-rpg\n"))
        # The macro substitution uses the same label, so {{char}} is not a run of blanks.
        self.assertIn("main_bestiality-rpg", document)

    def test_a_nameless_card_without_a_filename_says_so(self):
        obj = v3_card_dict()
        obj["data"]["name"] = "   "
        card = ccv3.read_text(json.dumps(obj))
        self.assertTrue(markdown.to_markdown(card).startswith("# Unnamed card\n"))

    def test_display_label_strips_the_extension_chain(self):
        card = self.card()
        obj = v3_card_dict()
        obj["data"]["name"] = ""
        blank = ccv3.read_text(json.dumps(obj))
        self.assertEqual(markdown.display_label(blank, "Lila.card.png"), "Lila")

    def test_html_is_converted_to_markdown(self):
        card = self.card(
            personality="<p>Brave <strong>and</strong> bold.</p><p>Second.</p>",
        )
        document = markdown.to_markdown(card)
        self.assertIn("Brave **and** bold.", document)
        self.assertNotIn("<p>", document)
        self.assertNotIn("<strong>", document)

    def test_html_lists_become_markdown_lists(self):
        card = self.card(personality="<ul><li>one</li><li>two</li></ul>")
        document = markdown.to_markdown(card)
        self.assertIn("- one", document)
        self.assertIn("- two", document)

    def test_html_entities_are_unescaped(self):
        card = self.card(personality="Tom &amp; Jerry said &quot;hi&quot;")
        document = markdown.to_markdown(card)
        self.assertIn('Tom & Jerry said "hi"', document)

    def test_escaped_angle_brackets_stay_visible_as_text(self):
        """An author who wrote &lt;p&gt; wanted the reader to see the literal text."""
        card = self.card(personality="Use &lt;p&gt; for a paragraph.")
        document = markdown.to_markdown(card)
        self.assertIn("Use <p> for a paragraph.", document)

    def test_card_syntax_is_not_mistaken_for_html(self):
        """Regression: <START> is a dialogue marker, not a tag, and was being deleted.

        The converter documents that only known HTML tag names are treated as markup. The
        implementation initially stripped anything tag-shaped, which silently removed the
        markers that delimit example dialogue blocks.
        """
        card = self.card(mes_example="<START>\n{{char}}: hello\n<START>\nmore")
        document = markdown.to_markdown(card)
        self.assertIn("<START>", document)
        self.assertEqual(document.count("<START>"), 2)

    def test_html_headings_are_demoted(self):
        """An h1 inside a field must not become a top-level heading in the document."""
        card = self.card(personality="<h1>Big</h1><p>Body.</p>")
        document = markdown.to_markdown(card)
        self.assertNotIn("\n# Big", document)
        self.assertIn("\n### Big", document)
        # The card's own title is still the only level-one heading.
        self.assertEqual(sum(1 for line in document.splitlines() if line.startswith("# ")), 1)

    def test_html_can_be_left_alone(self):
        card = self.card(personality="<p>Brave.</p>")
        document = markdown.to_markdown(card, convert_html=False)
        self.assertIn("<p>Brave.</p>", document)

    def test_empty_fields_are_omitted_by_default(self):
        card = self.card(alternate_greetings=[], scenario="")
        document = markdown.to_markdown(card)
        self.assertNotIn("## Alternate greetings", document)
        self.assertNotIn("## Scenario", document)

    def test_empty_fields_can_be_listed(self):
        card = self.card(alternate_greetings=[], scenario="")
        document = markdown.to_markdown(card, include_empty=True)
        self.assertIn("## Alternate greetings", document)
        self.assertIn("## Scenario", document)
        self.assertIn("*(not set)*", document)

    def test_summary_lists_the_dates(self):
        card = self.card(creation_date=1700000000)
        document = markdown.to_markdown(card)
        self.assertIn("2023-11-14", document)

    def test_lorebook_entries_are_rendered_with_their_directives(self):
        obj = v3_card_dict()
        obj["data"]["character_book"] = lorebook_dict(
            [
                entry(
                    keys=["dragon"],
                    content="@@activate_only_after 2\n@@position after_desc\nDragon lore.",
                    name="Dragons",
                    insertion_order=5,
                    priority=3,
                )
            ]
        )
        card = ccv3.read_text(json.dumps(obj))
        document = markdown.to_markdown(card)
        self.assertIn("Dragon lore.", document)
        self.assertIn("`@@activate_only_after 2`", document)
        self.assertIn("`@@position after_desc`", document)
        self.assertIn("keys: `dragon`", document)
        self.assertIn("insertion order 5", document)
        self.assertIn("priority 3", document)

    def test_disabled_entries_are_marked(self):
        obj = v3_card_dict()
        obj["data"]["character_book"] = lorebook_dict(
            [entry(keys=["x"], content="Hidden.", enabled=False)]
        )
        card = ccv3.read_text(json.dumps(obj))
        self.assertIn("(disabled)", markdown.to_markdown(card))

    def test_lorebook_can_be_omitted(self):
        obj = v3_card_dict()
        obj["data"]["character_book"] = lorebook_dict([entry(keys=["x"], content="Body.")])
        card = ccv3.read_text(json.dumps(obj))
        self.assertNotIn("Body.", markdown.to_markdown(card, include_lorebook=False))

    def test_provenance_records_the_container_and_version(self):
        card = self.card()
        document = markdown.to_markdown(card)
        self.assertIn("Character Card V3", document)
        self.assertIn("`json`", document)

    def test_pipes_in_content_do_not_break_the_table(self):
        card = self.card(name="A|B")
        document = markdown.to_markdown(card)
        summary = document.split("## Summary")[1].split("##")[0]
        for line in summary.splitlines():
            if line.startswith("| Name"):
                # A table row has three real delimiters. Any further pipe must be escaped,
                # so the count of unescaped ones is what has to stay at three.
                unescaped = sum(
                    1
                    for index, char in enumerate(line)
                    if char == "|" and (index == 0 or line[index - 1] != "\\")
                )
                self.assertEqual(unescaped, 3, f"unescaped pipes in: {line}")
                self.assertIn("A\\|B", line)

    def test_markdown_is_written_to_a_file(self):
        import tempfile

        card = self.card()
        with tempfile.TemporaryDirectory() as directory:
            path = os.path.join(directory, "out.md")
            markdown.to_markdown_file(card, path, source_name="x.png")
            with open(path, encoding="utf-8") as handle:
                self.assertIn("# Aria", handle.read())


if __name__ == "__main__":
    unittest.main(verbosity=2)
