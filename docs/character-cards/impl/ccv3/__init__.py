"""ccv3: a reader and writer for Character Card V1, V2, and V3.

The public surface is deliberately small. :func:`read` takes bytes and gives back a
:class:`~ccv3.model.Card` that is always in the V3 shape, whatever the file actually
contained. :func:`~ccv3.lorebook.scan` takes that card's lorebook and a conversation and
returns what should be injected. :mod:`ccv3.export` writes cards back out in any of the three
specifications.

Reading runs seven steps, and each one lives in its own module so it can be tested and
replaced on its own:

1. **Sniff the container** from the leading bytes: PNG, APNG, JSON, or CHARX.
2. **Extract the payload**, which means base64 and a ``tEXt`` chunk for a PNG, the archive
   root for a CHARX, and nothing at all for JSON.
3. **Detect the specification version**, falling back to shape detection when the card does
   not say, because a bare V1 card has no version marker at all.
4. **Validate**, producing a report rather than an exception.
5. **Normalise upward** into the V3 shape, preserving every field the specification does not
   define.
6. **Resolve assets**, which is a lookup for CHARX and a no-op for the other containers.
7. **Mount the lorebook**, so that scanning is a call away rather than a re-parse.

A typical read looks like this::

    import ccv3

    card = ccv3.read_file("character.png")
    print(card.display_name, card.container, card.origin_spec)
    for message in card.warnings:
        print("warning:", message)

    result = ccv3.scan_lorebook(card, ccv3.ScanContext(messages=["hello world"]))
    print(result.injection_text())

And writing a card so that both a V3 reader and an old one can use it::

    png_bytes = ccv3.export.write_png(card, base_image=original_bytes)
"""

from __future__ import annotations

from typing import Any, Optional

from . import cbs, containers, decorators, export, lorebook, normalise, png, validate, version
from .errors import (
    CCV3Error,
    ContainerError,
    MalformedCardError,
    UnknownSpecError,
    UnsupportedVersionError,
)
from .lorebook import ScanContext, ScanHit, ScanResult, approximate_tokens, scan
from .model import Asset, Card, CardData, Lorebook, LorebookEntry
from .version import (
    CURRENT_MAJOR,
    SPEC_LOREBOOK_V3,
    SPEC_V1,
    SPEC_V2,
    SPEC_V3,
    detect_version,
    parse_version,
)

__version__ = "1.0.0"

__all__ = [
    # Reading
    "read",
    "read_file",
    "read_text",
    "read_lorebook",
    # Writing
    "export",
    # Model
    "Asset",
    "Card",
    "CardData",
    "Lorebook",
    "LorebookEntry",
    # Scanning
    "ScanContext",
    "ScanHit",
    "ScanResult",
    "scan",
    "scan_lorebook",
    "approximate_tokens",
    # Macros
    "cbs",
    "render_macros",
    # Validation
    "validate",
    # Version helpers
    "SPEC_V1",
    "SPEC_V2",
    "SPEC_V3",
    "SPEC_LOREBOOK_V3",
    "CURRENT_MAJOR",
    "detect_version",
    "parse_version",
    # Errors
    "CCV3Error",
    "ContainerError",
    "MalformedCardError",
    "UnknownSpecError",
    "UnsupportedVersionError",
    # Modules worth reaching into
    "containers",
    "decorators",
    "lorebook",
    "normalise",
    "png",
]


def read(
    data: bytes,
    *,
    run_validation: bool = True,
    strict_crc: bool = False,
) -> Card:
    """Read a card from bytes of any supported container.

    The returned card is always in the V3 shape, and its ``origin_spec`` records what the file
    actually declared. Every non-fatal problem encountered along the way, from a bad PNG
    checksum to a missing required field, ends up in ``card.warnings``.

    Raises :class:`~ccv3.errors.ContainerError` when the bytes are not a container this reader
    knows, and :class:`~ccv3.errors.MalformedCardError` when the container is fine but the
    card inside it is unusable.
    """
    payload = containers.load(data)

    spec, spec_version, version_warnings = detect_version(payload.obj)

    if spec == SPEC_LOREBOOK_V3:
        raise MalformedCardError(
            "this file contains a standalone lorebook, not a card. Use read_lorebook() for it."
        )

    card = normalise.normalise_card(
        payload.obj,
        spec=spec,
        version=spec_version,
        container=payload.container,
        container_warnings=list(payload.warnings) + list(version_warnings),
        from_chunk=payload.from_chunk,
        asset_blobs=payload.asset_blobs,
    )

    if payload.legacy_png_assets:
        card.asset_blobs.update(
            {
                f"__asset:{path}": _decode_legacy_asset(blob)
                for path, blob in payload.legacy_png_assets.items()
            }
        )

    if run_validation:
        report = validate.validate_card_dict(payload.obj, spec=spec, version=spec_version)
        for message in report.messages():
            card.warn(message)

    return card


def _decode_legacy_asset(base64_text: str) -> bytes:
    """Decode a legacy PNG extension asset, returning empty bytes when it will not decode.

    The specification asks new implementations not to use this embedding method, so the goal
    here is only to avoid losing bytes a card refers to, not to support the method properly.
    """
    try:
        return containers.decode_base64_text(base64_text)
    except MalformedCardError:
        return b""


def read_file(path: str, **kwargs: Any) -> Card:
    """Read a card from a file on disk. The extension is not consulted."""
    with open(path, "rb") as handle:
        return read(handle.read(), **kwargs)


def read_text(json_text: str, **kwargs: Any) -> Card:
    """Read a card from a JSON string, for callers that already have the text."""
    return read(json_text.encode("utf-8"), **kwargs)


def read_lorebook(data: bytes) -> Lorebook:
    """Read a standalone lorebook, the ``{"spec": "lorebook_v3", "data": {...}}`` shape.

    A standalone lorebook is not a card, so it gets its own entry point rather than being
    squeezed into a :class:`~ccv3.model.Card` with every character field empty.
    """
    payload = containers.load(data)
    spec, _version, _warnings = detect_version(payload.obj)

    warnings: list[str] = list(payload.warnings)

    if spec == SPEC_LOREBOOK_V3:
        book = normalise.normalise_lorebook(payload.obj.get("data"), warnings=warnings, where="data")
    else:
        # A card was handed to this function. Reading its embedded book is the useful thing to
        # do rather than refusing, since that is almost certainly what the caller wanted.
        card = read(data)
        book = card.lorebook

    if book is None:
        raise MalformedCardError("no lorebook was found in the file")
    return book


def scan_lorebook(card: Card, context: Optional[ScanContext] = None, **kwargs: Any) -> ScanResult:
    """Scan a card's lorebook. A convenience wrapper over :func:`~ccv3.lorebook.scan`."""
    return scan(card.lorebook_or_empty(), context, **kwargs)


def render_macros(text: str, card: Optional[Card] = None, user_name: str = "", **kwargs: Any) -> str:
    """Substitute the curly braced syntaxes in a piece of text.

    When a card is supplied, its nickname and name feed ``{{char}}`` with the specification's
    fallback order applied.
    """
    char_name = card.display_name if card is not None else ""
    return cbs.render(text, char_name=char_name, user_name=user_name, **kwargs).text
