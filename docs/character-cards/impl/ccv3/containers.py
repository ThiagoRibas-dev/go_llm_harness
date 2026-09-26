"""Getting the card object out of whatever it arrived in.

This is step one and step two of the reading pipeline: work out which container the bytes
are, then extract the JSON payload from it.

The module returns a :class:`Payload`, which is deliberately dumb. It holds the decoded
object, the name of the container, and a small amount of provenance, and it makes no attempt
to interpret the card. Version detection, validation, and normalisation all happen above
this layer, which means the container code contains no `if version == ...` branches at all
and a new container can be added without touching any card logic.
"""

from __future__ import annotations

import base64
import binascii
import io
import json
import zipfile
from dataclasses import dataclass, field
from typing import Any, Optional

from . import png
from .errors import ContainerError, MalformedCardError

#: Container names this module produces.
CONTAINER_PNG = "png"
CONTAINER_APNG = "apng"
CONTAINER_JSON = "json"
CONTAINER_CHARX = "charx"

#: The name of the card file at the root of a CHARX archive.
CHARX_CARD_NAME = "card.json"

#: Where CHARX assets live inside the archive.
CHARX_ASSET_ROOT = "assets/"

#: The local file header signature of a ZIP entry. CHARX archives can be self-extracting,
#: meaning arbitrary bytes precede a perfectly valid archive, so the reader searches for this
#: signature rather than assuming the file begins with it.
_ZIP_LOCAL_HEADER = b"PK\x03\x04"

#: The message the specification asks an application to leave for the user when it presents
#: a V3 card through a V2-shaped view.
BACKFILL_WARNING = (
    "This character card is Character Card V3, but it is loaded as a Character Card V2. "
    "Please use a Character Card V3 compatible application to use this character card "
    "properly."
)


@dataclass
class Payload:
    """A decoded card object plus where it came from."""

    obj: dict[str, Any]
    container: str
    warnings: list[str] = field(default_factory=list)

    #: For PNG and APNG, which chunk the card came out of: ``ccv3`` or ``chara``. ``None``
    #: for every other container. The caller turns this into a backfill decision once it
    #: knows which version the card actually declares.
    from_chunk: Optional[str] = None

    #: Asset payloads by archive path. Only CHARX populates this.
    asset_blobs: dict[str, bytes] = field(default_factory=dict)

    #: Extension assets found in PNG text chunks, by path, still base64 encoded. Reported but
    #: not resolved, since the specification asks new implementations not to use this method.
    legacy_png_assets: dict[str, str] = field(default_factory=dict)


# ---------------------------------------------------------------------------
# Format sniffing
# ---------------------------------------------------------------------------


def sniff_container(data: bytes) -> str:
    """Identify the container from the leading bytes.

    Detection is by content rather than by file extension, because cards are routinely
    renamed: a ``.png`` that is really a JSON file, or a ``.charx`` with the wrong extension,
    both appear in the wild. Extensions are only consulted if this function returns
    ``unknown``.
    """
    if png.is_png(data):
        return CONTAINER_APNG if png.is_apng(data) else CONTAINER_PNG

    stripped = data.lstrip(b"\xef\xbb\xbf \t\r\n")
    if stripped.startswith(_ZIP_LOCAL_HEADER) or _find_zip_start(data) is not None:
        return CONTAINER_CHARX

    if stripped[:1] in (b"{", b"["):
        return CONTAINER_JSON

    return "unknown"


def _find_zip_start(data: bytes) -> Optional[int]:
    """Offset of the first ZIP local file header, or ``None``.

    The offset-zero case is checked first and returned directly. That check is load-bearing
    rather than an optimisation: a ZIP archive contains a local header for every entry it
    holds, so a search beginning at offset one finds a *nested* header a few hundred bytes in
    and would slice the file open at the wrong place. Treating offset zero as the ordinary
    case and searching only when it does not match keeps the self-extracting case working
    without breaking the normal one.

    The search is bounded to the first megabyte, which is far past any real self-extracting
    stub and keeps the scan cheap.
    """
    if data[:4] == _ZIP_LOCAL_HEADER:
        return 0
    limit = min(len(data), 1024 * 1024)
    index = data.find(_ZIP_LOCAL_HEADER, 0, limit)
    return index if index != -1 else None


# ---------------------------------------------------------------------------
# Base64 and JSON helpers
# ---------------------------------------------------------------------------


def decode_base64_text(text: str) -> bytes:
    """Decode base64 that may be missing its padding or contain line breaks.

    The specification says the chunk value is UTF-8 encoded to base64. It does not say the
    base64 is padded, and several writers emit it unpadded or wrapped across lines, so this
    function normalises before decoding rather than raising on input that every other reader
    in the ecosystem accepts.
    """
    compact = "".join(text.split())
    remainder = len(compact) % 4
    if remainder == 2:
        compact += "=="
    elif remainder == 3:
        compact += "="
    elif remainder == 1:
        # Not a length base64 can produce. Let the decoder raise a clear error below.
        pass
    try:
        return base64.b64decode(compact, validate=False)
    except (binascii.Error, ValueError) as exc:
        raise MalformedCardError(f"card payload is not valid base64: {exc}") from exc


def encode_base64_text(data: bytes) -> str:
    """Encode bytes as base64 text, the way a PNG chunk carries a card.

    Padding is kept. Unpadded base64 is tolerated on the way in because writers emit it, but
    there is no reason to produce it on the way out.
    """
    return base64.b64encode(data).decode("ascii")


def parse_json_object(raw: bytes, *, source: str) -> dict[str, Any]:
    """Decode bytes as a JSON object, with a useful error message when it fails.

    A JSON array or scalar is rejected here rather than passed upward, because every card
    shape is an object and a caller receiving a list would have to handle it separately
    anyway.
    """
    try:
        text = raw.decode("utf-8-sig")
    except UnicodeDecodeError as exc:
        raise MalformedCardError(f"{source} is not valid UTF-8: {exc}") from exc

    try:
        decoded = json.loads(text)
    except json.JSONDecodeError as exc:
        preview = text[:120].replace("\n", " ")
        raise MalformedCardError(
            f"{source} is not valid JSON: {exc.msg} at line {exc.lineno} column {exc.colno}. "
            f"The payload begins: {preview!r}"
        ) from exc

    if not isinstance(decoded, dict):
        raise MalformedCardError(
            f"{source} contains a JSON {type(decoded).__name__}, but a card is always a "
            f"JSON object"
        )
    return decoded


# ---------------------------------------------------------------------------
# Per-container loaders
# ---------------------------------------------------------------------------


def load_png(data: bytes, container: str) -> Payload:
    """Extract the card from a PNG or APNG.

    Preference order is the one the specification states: a ``ccv3`` chunk wins over a
    ``chara`` chunk. If the preferred chunk is present but cannot be decoded, the reader
    falls back to ``chara`` and records why, rather than failing the file. That behaviour is
    worth having because a partially written ``ccv3`` chunk is a real failure mode when an
    export is interrupted, and the older chunk is usually still intact.
    """
    if not png.is_png(data):
        raise ContainerError("not a PNG: the signature is missing")

    chunks, warnings = png.read_text_chunks(data)
    legacy_assets = png.read_embedded_assets(data)

    for candidate in (png.KEYWORD_CCV3, png.KEYWORD_CHARA):
        raw_text = chunks.get(candidate)
        if raw_text is None:
            continue

        try:
            decoded = decode_base64_text(raw_text)
            obj = parse_json_object(decoded, source=f"the PNG {candidate!r} chunk")
        except MalformedCardError as exc:
            # Remember the failure and try the next candidate. If none succeed, the caller
            # gets an error that names every chunk we tried.
            warnings.append(f"PNG {candidate!r} chunk could not be read: {exc}")
            continue

        if candidate == png.KEYWORD_CHARA and png.KEYWORD_CCV3 in chunks:
            warnings.append(
                "PNG carries both a 'ccv3' and a 'chara' chunk and the 'ccv3' chunk was "
                "unusable; fell back to the older 'chara' chunk"
            )

        if legacy_assets:
            warnings.append(
                f"PNG carries {len(legacy_assets)} extension asset chunk(s). The "
                f"specification asks new implementations to avoid this method in favour of "
                f"CHARX, so the assets are reported but not loaded."
            )

        return Payload(
            obj=obj,
            container=container,
            warnings=warnings,
            from_chunk=candidate,
            legacy_png_assets=legacy_assets,
        )

    tried = ", ".join(repr(k) for k in chunks) or "none"
    raise MalformedCardError(
        f"PNG contains no usable card. Text chunks found: {tried}. "
        f"Expected a {png.KEYWORD_CCV3!r} chunk, optionally falling back to "
        f"{png.KEYWORD_CHARA!r}."
    )


def load_json(data: bytes) -> Payload:
    """Extract the card from a JSON file, where the file *is* the card object."""
    obj = parse_json_object(data, source="the JSON file")
    return Payload(obj=obj, container=CONTAINER_JSON, warnings=[])


def load_charx(data: bytes) -> Payload:
    """Extract the card and its assets from a CHARX archive.

    The archive is located by signature search rather than read from offset zero, so a CHARX
    file with a self-extracting stub in front of it still opens. Every entry under
    ``assets/`` is read into memory as raw bytes. That is a deliberate simplification: the
    card's own size is small and the assets are worth having at hand, but a caller dealing
    with a very large archive should stream them instead.
    """
    start = _find_zip_start(data)
    archive_bytes = data[start:] if start else data

    warnings: list[str] = []
    try:
        archive = zipfile.ZipFile(io.BytesIO(archive_bytes))
    except zipfile.BadZipFile as exc:
        raise ContainerError(f"CHARX file is not a readable ZIP archive: {exc}") from exc

    with archive:
        names = archive.namelist()

        if CHARX_CARD_NAME not in names:
            # A CHARX with the card nested in a folder is non-conforming but common enough
            # after a careless re-zip that finding it is worth the few lines.
            nested = [n for n in names if n.endswith("/" + CHARX_CARD_NAME)]
            if len(nested) == 1:
                warnings.append(
                    f"CHARX has no {CHARX_CARD_NAME!r} at the root, but exactly one nested "
                    f"copy at {nested[0]!r}; using it"
                )
                card_name = nested[0]
            elif len(nested) > 1:
                raise MalformedCardError(
                    f"CHARX has no {CHARX_CARD_NAME!r} at the root and {len(nested)} nested "
                    f"copies, so the intended card is ambiguous"
                )
            else:
                raise MalformedCardError(
                    f"CHARX archive has no {CHARX_CARD_NAME!r}. Entries present: "
                    f"{', '.join(names[:10])}{'...' if len(names) > 10 else ''}"
                )
        else:
            card_name = CHARX_CARD_NAME

        card_bytes = archive.read(card_name)
        obj = parse_json_object(card_bytes, source=f"CHARX {card_name!r}")

        asset_blobs: dict[str, bytes] = {}
        for name in names:
            if name.endswith("/"):
                continue
            if not name.startswith(CHARX_ASSET_ROOT):
                continue
            try:
                asset_blobs[name] = archive.read(name)
            except (KeyError, zipfile.BadZipFile) as exc:
                warnings.append(f"CHARX asset {name!r} could not be read: {exc}")

    if not asset_blobs:
        warnings.append(
            f"CHARX archive contains no entries under {CHARX_ASSET_ROOT!r}; any embedded "
            f"asset URI in the card will not resolve"
        )

    return Payload(
        obj=obj,
        container=CONTAINER_CHARX,
        warnings=warnings,
        asset_blobs=asset_blobs,
    )


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------


def load(data: bytes) -> Payload:
    """Extract a card payload from bytes of unknown container.

    The container is sniffed from the content. If that fails, the error message names the
    bytes it actually saw, because "unrecognised format" is not actionable and "the file
    starts with 0x50 0x4b but is not a readable ZIP" is.
    """
    container = sniff_container(data)

    if container in (CONTAINER_PNG, CONTAINER_APNG):
        return load_png(data, container)
    if container == CONTAINER_CHARX:
        return load_charx(data)
    if container == CONTAINER_JSON:
        return load_json(data)

    head = data[:16]
    raise ContainerError(
        f"unrecognised container. The file begins with {head!r}. This reader understands "
        f"PNG and APNG files, JSON files, and CHARX archives. A PNG must begin with the "
        f"PNG signature, a CHARX archive must be a ZIP, and a JSON card must begin with an "
        f"object."
    )
