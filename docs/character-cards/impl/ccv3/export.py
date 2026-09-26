"""Writing cards back out, including down to V2 and V1.

The reading side of this package normalises upward into the V3 shape, so the writing side has
to be able to project back down onto any of the three specifications. Three decisions shape
this module.

**Unknown fields are written back.** Every value that was preserved in an ``extra`` mapping is
merged into the output. Without this, a reader-and-writer pair would quietly delete every
field added to the specification after the version of this code, which is exactly the failure
the specification asks applications to avoid.

**Backfilling to V2 strips decorators.** The specification says so explicitly, and the reason
is practical: a V2 reader has no idea what ``@@activate_only_after 3`` means, so it would
treat the line as ordinary entry text and inject it into the prompt. Stripping them is the
one destructive transformation this module performs, and it is why the function that does it
is named for the specification clause rather than called something neutral like
"downgrade".

**Required fields are always written, optional ones only when they carry something.** A card
that omits ``nickname`` is unambiguous; a card that writes ``nickname: ""`` invites a reader
to wonder whether the field is meant to suppress the name. Writing only what is set keeps the
output close to what a human author would have typed.

The backfill notice is offered rather than imposed. The specification supplies a sentence to
leave in ``creator_notes`` when a V3 card is presented as V2, and asking the caller to opt in
keeps a deliberate V2 export from being decorated with a warning the author did not want.
"""

from __future__ import annotations

import io
import json
import zipfile
from typing import Any, Iterable, Optional

from . import containers, decorators, png
from .model import Asset, Card, CardData, Lorebook, LorebookEntry

#: The notice the specification asks an application to leave for a user when it presents a V3
#: card through a V2 view.
BACKFILL_NOTICE = (
    "This character card is Character Card V3, but it is loaded as a Character Card V2. "
    "Please use a Character Card V3 compatible application to use this character card "
    "properly."
)

#: The fourteen fields a V2 card must carry inside ``data``.
V2_REQUIRED_DATA_FIELDS = (
    "name",
    "description",
    "personality",
    "scenario",
    "first_mes",
    "mes_example",
    "creator_notes",
    "system_prompt",
    "post_history_instructions",
    "alternate_greetings",
    "tags",
    "creator",
    "character_version",
    "extensions",
)

#: The fields a V1 card carries flat at the top level.
V1_FIELDS = (
    "name",
    "description",
    "personality",
    "scenario",
    "first_mes",
    "mes_example",
)


def _merge_unknown(target: dict[str, Any], extra: dict[str, Any]) -> dict[str, Any]:
    """Merge preserved unknown fields into an output object without overwriting anything.

    ``setdefault`` rather than assignment, because a value the writer computed from the model
    is by definition more current than the copy preserved from the original file. This can
    only matter when a caller has edited the card, and in that case the edit should win.
    """
    for key, value in extra.items():
        target.setdefault(key, value)
    return target


# ---------------------------------------------------------------------------
# Assets
# ---------------------------------------------------------------------------


def asset_to_dict(asset: Asset) -> dict[str, Any]:
    """Serialise one asset. The ``ext`` key keeps its specification name."""
    out: dict[str, Any] = {
        "type": asset.type,
        "uri": asset.uri,
        "name": asset.name,
        "ext": asset.ext,
    }
    return _merge_unknown(out, asset.extra)


def assets_from_dicts(raw: Any) -> list[Asset]:
    """Build assets from decoded JSON, reusing the normalisation rules."""
    from .normalise import normalise_assets

    return normalise_assets(raw, warnings=[])


# ---------------------------------------------------------------------------
# Lorebook
# ---------------------------------------------------------------------------


def entry_to_dict(entry: LorebookEntry, *, strip_decorators: bool = False) -> dict[str, Any]:
    """Serialise one lorebook entry.

    All eight of the entry's specification fields that carry meaning are written
    unconditionally for the required ones, and the optional ones only when set. ``use_regex``
    and ``constant`` are written even when false, because V3 requires an implementation to
    honour them and writing them makes the entry's behaviour explicit rather than implied.
    """
    content = entry.content
    if strip_decorators:
        content = decorators.strip_all(content)

    out: dict[str, Any] = {
        "keys": list(entry.keys),
        "content": content,
        "extensions": dict(entry.extensions),
        "enabled": entry.enabled,
        "insertion_order": entry.insertion_order,
        "use_regex": entry.use_regex,
        "constant": entry.constant,
    }

    if entry.name:
        out["name"] = entry.name
    if entry.id is not None:
        out["id"] = entry.id
    if entry.comment:
        out["comment"] = entry.comment
    if entry.priority is not None:
        out["priority"] = entry.priority
    # Written whenever the card said anything, including when it said `False`. The
    # specification gives `false` and `undefined` different meanings, so dropping an
    # explicit `false` would change what the exported card tells a reader.
    if entry.case_sensitive is not None:
        out["case_sensitive"] = bool(entry.case_sensitive)
    if entry.selective is not None:
        out["selective"] = bool(entry.selective)
    if entry.secondary_keys:
        out["secondary_keys"] = list(entry.secondary_keys)
    if entry.position:
        out["position"] = entry.position

    return _merge_unknown(out, entry.extra)


def lorebook_to_dict(book: Lorebook, *, strip_decorators: bool = False) -> dict[str, Any]:
    """Serialise a lorebook."""
    out: dict[str, Any] = {
        "extensions": dict(book.extensions),
        "entries": [
            entry_to_dict(entry, strip_decorators=strip_decorators) for entry in book.entries
        ],
    }

    if book.name:
        out["name"] = book.name
    if book.description:
        out["description"] = book.description
    if book.scan_depth is not None:
        out["scan_depth"] = book.scan_depth
    if book.token_budget is not None:
        out["token_budget"] = book.token_budget
    if book.recursive_scanning is not None:
        out["recursive_scanning"] = bool(book.recursive_scanning)

    return _merge_unknown(out, book.extra)


def to_lorebook_v3(card_or_book: Any) -> dict[str, Any]:
    """Export a lorebook on its own, in the standalone shape the specification defines."""
    book = card_or_book if isinstance(card_or_book, Lorebook) else card_or_book.lorebook_or_empty()
    return {
        "spec": "lorebook_v3",
        "data": lorebook_to_dict(book),
    }


# ---------------------------------------------------------------------------
# Cards
# ---------------------------------------------------------------------------


def _base_data(card: Card) -> dict[str, Any]:
    data = card.data
    return {
        "name": data.name,
        "description": data.description,
        "personality": data.personality,
        "scenario": data.scenario,
        "first_mes": data.first_mes,
        "mes_example": data.mes_example,
        "creator_notes": data.creator_notes,
        "system_prompt": data.system_prompt,
        "post_history_instructions": data.post_history_instructions,
        "alternate_greetings": list(data.alternate_greetings),
        "tags": list(data.tags),
        "creator": data.creator,
        "character_version": data.character_version,
        "extensions": dict(data.extensions),
    }


def to_v3_dict(card: Card) -> dict[str, Any]:
    """Serialise a card as Character Card V3.

    Every V3-only field is written when it carries something, and the unknown fields are
    merged back in. The result is a card that a V2 reader will read correctly and a V3 reader
    will read completely.
    """
    data = card.data
    out = _base_data(card)

    # Required in V3, and may legitimately be empty, so it is always written.
    out["group_only_greetings"] = list(data.group_only_greetings)

    if data.character_book is not None:
        out["character_book"] = lorebook_to_dict(data.character_book)
    if data.assets:
        out["assets"] = [asset_to_dict(asset) for asset in data.assets]
    if data.nickname:
        out["nickname"] = data.nickname
    if data.creator_notes_multilingual:
        out["creator_notes_multilingual"] = dict(data.creator_notes_multilingual)
    if data.source:
        out["source"] = list(data.source)
    if data.creation_date is not None:
        out["creation_date"] = data.creation_date
    if data.modification_date is not None:
        out["modification_date"] = data.modification_date

    _merge_unknown(out, data.extra)

    card_out: dict[str, Any] = {
        "spec": "chara_card_v3",
        "spec_version": "3.0",
        "data": out,
    }
    return _merge_unknown(card_out, card.extra)


def to_v2_dict(
    card: Card,
    *,
    keep_v3_fields: bool = True,
    strip_lorebook_decorators: bool = True,
    add_backfill_notice: bool = False,
) -> dict[str, Any]:
    """Serialise a card as Character Card V2, for readers that cannot handle V3.

    The three keyword arguments are the three decisions a caller has to make.

    ``keep_v3_fields`` decides whether the V3 additions ride along inside the V2 object. The
    specification's forward-compatibility rule says a V2 reader should ignore fields it does
    not understand, so leaving them in is safe and makes the export lossless. Setting it to
    false produces a card that is exactly V2, which is what you want when the point of the
    export is to prove the card works without V3.

    ``strip_lorebook_decorators`` implements the specification's instruction to remove all
    decorators when backfilling. It is on by default because leaving them in produces a
    prompt containing literal ``@@`` lines.

    ``add_backfill_notice`` prepends the specification's sentence to ``creator_notes`` so the
    user of a V2-only application learns why the card looks reduced.
    """
    data = card.data
    out = _base_data(card)

    if add_backfill_notice and card.origin_spec == "chara_card_v3":
        existing = out.get("creator_notes", "")
        out["creator_notes"] = f"{BACKFILL_NOTICE}\n\n{existing}".strip()

    if data.character_book is not None:
        out["character_book"] = lorebook_to_dict(
            data.character_book, strip_decorators=strip_lorebook_decorators
        )

    if keep_v3_fields:
        if data.assets:
            out["assets"] = [asset_to_dict(asset) for asset in data.assets]
        if data.nickname:
            out["nickname"] = data.nickname
        if data.creator_notes_multilingual:
            out["creator_notes_multilingual"] = dict(data.creator_notes_multilingual)
        if data.source:
            out["source"] = list(data.source)
        if data.group_only_greetings:
            out["group_only_greetings"] = list(data.group_only_greetings)
        if data.creation_date is not None:
            out["creation_date"] = data.creation_date
        if data.modification_date is not None:
            out["modification_date"] = data.modification_date
        _merge_unknown(out, data.extra)

    # Guarantee the fourteen required fields are present even if the card was sparse.
    for field_name in V2_REQUIRED_DATA_FIELDS:
        out.setdefault(field_name, "" if field_name not in ("alternate_greetings", "tags") else [])
    out["extensions"] = out.get("extensions") or {}

    card_out: dict[str, Any] = {
        "spec": "chara_card_v2",
        "spec_version": "2.0",
        "data": out,
    }

    if keep_v3_fields:
        _merge_unknown(card_out, card.extra)

    return card_out


def to_v1_dict(card: Card) -> dict[str, Any]:
    """Serialise a card as a bare V1 object, with the fields at the top level.

    V1 has no wrapper and no version marker, so this is the one export that loses the ability
    to say what it is. It exists for completeness and for talking to very old tooling.
    """
    data = card.data
    out: dict[str, Any] = {field_name: getattr(data, field_name) for field_name in V1_FIELDS}
    if data.character_book is not None:
        out["character_book"] = lorebook_to_dict(
            data.character_book, strip_decorators=True
        )
    return out


def to_dict(card: Card, spec: str = "chara_card_v3", **kwargs: Any) -> dict[str, Any]:
    """Serialise to a named specification."""
    normalised = spec.strip().lower()
    if normalised in ("chara_card_v3", "v3", "3", "3.0"):
        return to_v3_dict(card)
    if normalised in ("chara_card_v2", "v2", "2", "2.0"):
        return to_v2_dict(card, **kwargs)
    if normalised in ("chara_card_v1", "v1", "1", "1.0"):
        return to_v1_dict(card)
    raise ValueError(f"unknown target spec {spec!r}")


def to_json_bytes(card: Card, spec: str = "chara_card_v3", **kwargs: Any) -> bytes:
    """Serialise to UTF-8 JSON, the way a JSON container file holds a card."""
    return json.dumps(
        to_dict(card, spec, **kwargs), ensure_ascii=False, indent=2
    ).encode("utf-8")


# ---------------------------------------------------------------------------
# Containers
# ---------------------------------------------------------------------------


def write_png(
    card: Card,
    base_image: Optional[bytes] = None,
    *,
    spec: str = "chara_card_v3",
    also_write_chara: bool = True,
    v2_kwargs: Optional[dict[str, Any]] = None,
) -> bytes:
    """Write a card into a PNG image.

    ``base_image`` is the image to carry the card. Passing the PNG a card was read from is the
    common case and keeps the artwork. When no image is supplied a minimal transparent PNG is
    generated, so this function is usable without an image library.

    ``spec`` chooses what the file says it is.

    * Writing V3 with ``also_write_chara`` true, which is the default, puts a ``ccv3`` chunk
      and a V2 ``chara`` chunk in the same file. That is what the most widely deployed
      application does, and the specification supports it by telling readers to prefer
      ``ccv3``. One file then serves a V3-aware reader and an old one, which is the export
      most authors actually want.
    * Writing V3 with ``also_write_chara`` false produces a strictly V3 file, which is what
      you want when the point of the export is to prove the card does not depend on the
      legacy chunk.
    * Writing V2 or V1 puts a single ``chara`` chunk in the file and no ``ccv3`` chunk,
      because a file that advertises V3 while containing a V2 object is a lie that a
      conforming reader will act on.

    Writing a V2 file is the one case where the card's V3 fields may or may not travel with
    it, and that is decided by ``v2_kwargs``, which is passed through to :func:`to_v2_dict`.
    """
    image = base_image if base_image is not None else build_placeholder_png()

    if not png.is_png(image):
        raise ValueError("base_image is not a PNG")

    target = spec.strip().lower()
    is_v3 = target in ("chara_card_v3", "v3", "3", "3.0")

    if is_v3:
        v3_text = containers.encode_base64_text(
            json.dumps(to_v3_dict(card), ensure_ascii=False).encode("utf-8")
        )
        result = png.set_text_chunk(image, png.KEYWORD_CCV3, v3_text)

        if also_write_chara:
            v2_text = containers.encode_base64_text(
                json.dumps(to_v2_dict(card, **(v2_kwargs or {})), ensure_ascii=False).encode(
                    "utf-8"
                )
            )
            result = png.set_text_chunk(result, png.KEYWORD_CHARA, v2_text)
        return result

    # Any non-V3 target goes into the legacy chunk, and the V3 chunk, if the base image
    # carried one, is removed so the file does not advertise a version it is not.
    payload = containers.encode_base64_text(
        json.dumps(to_dict(card, spec), ensure_ascii=False).encode("utf-8")
    )
    result = png.set_text_chunk(image, png.KEYWORD_CHARA, payload)
    return png.remove_text_chunk(result, png.KEYWORD_CCV3)


def write_charx(
    card: Card,
    assets: Optional[dict[str, bytes]] = None,
    *,
    spec: str = "chara_card_v3",
) -> bytes:
    """Write a card and its assets into a CHARX archive.

    Asset keys are archive paths such as ``assets/icon/images/main.png``. Entries named in the
    card's ``embeded://`` URIs that are not supplied are reported by the caller's own
    validation rather than silently omitted; this function writes what it is given.
    """
    buffer = io.BytesIO()
    payload = json.dumps(to_dict(card, spec), ensure_ascii=False, indent=2).encode("utf-8")

    with zipfile.ZipFile(buffer, mode="w", compression=zipfile.ZIP_DEFLATED) as archive:
        archive.writestr(containers.CHARX_CARD_NAME, payload)
        for path, blob in (assets or card.asset_blobs).items():
            archive.writestr(path, blob)

    return buffer.getvalue()


def build_placeholder_png(width: int = 1, height: int = 1) -> bytes:
    """Build a minimal valid PNG with no image library.

    A one-pixel transparent image is enough to carry a card, and generating it here keeps this
    package free of dependencies. The chunks are assembled by hand because that is about
    twenty lines of standard library work, and pulling in an imaging library to produce a
    one-pixel image would be a poor trade.
    """
    import struct
    import zlib

    def chunk(chunk_type: bytes, payload: bytes) -> bytes:
        return (
            struct.pack(">I", len(payload))
            + chunk_type
            + payload
            + struct.pack(">I", zlib.crc32(chunk_type + payload) & 0xFFFFFFFF)
        )

    # Colour type 6 is RGBA. Bit depth 8, no interlacing, filter method 0.
    ihdr = struct.pack(">IIBBBBB", width, height, 8, 6, 0, 0, 0)

    raw = b""
    for _row in range(height):
        raw += b"\x00"  # filter type 0, meaning no filtering
        raw += b"\x00" * (width * 4)  # transparent RGBA pixels

    return (
        png.PNG_SIGNATURE
        + chunk(b"IHDR", ihdr)
        + chunk(b"IDAT", zlib.compress(raw, 9))
        + chunk(b"IEND", b"")
    )
