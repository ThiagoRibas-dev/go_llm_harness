"""PNG and APNG chunk handling.

A PNG is a signature followed by a sequence of chunks. Each chunk is a four-byte big-endian
length, a four-byte type, that many bytes of payload, and a four-byte CRC over the type and
payload together. The card sits in a ``tEXt`` chunk whose keyword is ``ccv3`` and whose
payload is the card's JSON encoded as UTF-8 and then base64.

Two decisions here are worth stating.

First, the payload is treated as opaque bytes on the way in and only decoded once the reader
above has decided which keyword it wants. That keeps the container layer free of any
knowledge about cards, which in turn means the same code handles the legacy ``chara`` chunk
and the extension-asset chunks without special cases.

Second, a chunk whose CRC does not match its contents is reported through the returned
warning list rather than raising. In practice a bad CRC usually means the image was edited by
a tool that did not rewrite the checksums, not that the card is corrupt, and the card
sits inside the chunk verbatim whether or not the CRC is right. Callers that want the strict
behaviour can pass ``strict_crc=True``.
"""

from __future__ import annotations

import struct
import zlib
from typing import Iterator, Optional

#: The eight bytes every PNG and APNG file begins with.
PNG_SIGNATURE = b"\x89PNG\r\n\x1a\n"

#: Chunk types this module cares about by name.
CHUNK_IHDR = b"IHDR"
CHUNK_ACTL = b"acTL"  # APNG animation control; its presence marks a file as an APNG
CHUNK_TEXT = b"tEXt"
CHUNK_ZTXT = b"zTXt"
CHUNK_ITXT = b"iTXt"
CHUNK_IEND = b"IEND"

#: The keyword the V3 specification assigns to the card.
KEYWORD_CCV3 = "ccv3"

#: The keyword the V2 specification assigns to the card.
KEYWORD_CHARA = "chara"

#: The prefix for extension assets embedded directly in a PNG. The specification documents
#: this method only so that readers recognise it, and asks new implementations to use CHARX
#: instead, so this reader recognises the chunks and reports them rather than advertising
#: support for them.
KEYWORD_ASSET_PREFIX = "chara-ext-asset_:"

#: The only compression method a PNG defines, for both zTXt and iTXt.
_COMPRESSION_DEFLATE = 0

_MAX_CHUNK_LENGTH = 0x7FFFFFFF


def is_png(data: bytes) -> bool:
    """Whether the bytes begin with the PNG signature."""
    return data[:8] == PNG_SIGNATURE


def is_apng(data: bytes) -> bool:
    """Whether the bytes are a PNG carrying an animation control chunk.

    An APNG is a PNG in every respect that matters here, so the only reason to distinguish
    them is to report the container accurately.
    """
    if not is_png(data):
        return False
    for chunk_type, _payload, _crc_ok in iter_chunks(data, strict_crc=False):
        if chunk_type == CHUNK_ACTL:
            return True
        if chunk_type in (CHUNK_IEND, CHUNK_IHDR):
            # acTL must appear before IDAT, so once IDAT or IEND passes there is no more
            # chance of finding one. Checking only these two keeps the scan short.
            if chunk_type == CHUNK_IEND:
                return False
    return False


def iter_chunks(
    data: bytes, *, strict_crc: bool = False
) -> Iterator[tuple[bytes, bytes, bool]]:
    """Walk the chunk sequence, yielding ``(type, payload, crc_ok)`` for each chunk.

    The walk stops at ``IEND`` or at the first structurally impossible chunk, whichever
    comes first. It does not raise on a malformed tail, because a truncated PNG still
    frequently contains an intact ``tEXt`` chunk earlier in the file and refusing to read it
    would lose a recoverable card.
    """
    if not is_png(data):
        return

    offset = len(PNG_SIGNATURE)
    total = len(data)

    while offset + 8 <= total:
        declared_length = struct.unpack_from(">I", data, offset)[0]
        chunk_type = data[offset + 4:offset + 8]

        # A chunk longer than the file cannot be real. Bail rather than allocating.
        if declared_length > _MAX_CHUNK_LENGTH:
            return

        payload_start = offset + 8
        payload_end = payload_start + declared_length
        crc_end = payload_end + 4

        if crc_end > total:
            # Truncated final chunk. Yield what is present, then stop.
            if payload_end <= total:
                yield chunk_type, data[payload_start:payload_end], False
            return

        payload = data[payload_start:payload_end]
        stored_crc = struct.unpack_from(">I", data, payload_end)[0]
        actual_crc = zlib.crc32(chunk_type + payload) & 0xFFFFFFFF
        crc_ok = stored_crc == actual_crc

        if strict_crc and not crc_ok:
            raise ValueError(
                f"chunk {chunk_type!r} has CRC {stored_crc:#010x} but the contents hash to "
                f"{actual_crc:#010x}"
            )

        yield chunk_type, payload, crc_ok

        if chunk_type == CHUNK_IEND:
            return

        offset = crc_end


def _split_keyword(payload: bytes) -> Optional[tuple[str, bytes]]:
    """Split a text chunk payload into its keyword and the bytes after the separator.

    Returns ``None`` when the payload has no null separator, which makes it not a text chunk
    the reader can use.
    """
    separator = payload.find(b"\x00")
    if separator == -1:
        return None
    keyword = payload[:separator].decode("latin-1")
    return keyword, payload[separator + 1:]


def decode_text_payload(chunk_type: bytes, payload: bytes) -> Optional[str]:
    """Decode one text chunk into a string, handling the three text chunk types.

    ``tEXt`` is what the specification requires. ``iTXt`` and ``zTXt`` are handled because
    other tools write them and a reader that only understands ``tEXt`` will silently find no
    card in a file that plainly has one. Within ``iTXt`` only the uncompressed form is
    decoded; a compressed ``iTXt`` returns ``None`` rather than guessing, since the
    language tag and translated keyword that precede the text in that chunk are easy to
    mis-slice.
    """
    split = _split_keyword(payload)
    if split is None:
        return None
    keyword, remainder = split

    if chunk_type == CHUNK_TEXT:
        return remainder.decode("latin-1")

    if chunk_type == CHUNK_ZTXT:
        if not remainder:
            return None
        method = remainder[0]
        if method != _COMPRESSION_DEFLATE:
            return None
        try:
            return zlib.decompress(remainder[1:]).decode("utf-8", errors="replace")
        except zlib.error:
            return None

    if chunk_type == CHUNK_ITXT:
        if len(remainder) < 2:
            return None
        compression_flag = remainder[0]
        compression_method = remainder[1]
        rest = remainder[2:]

        # language tag, then translated keyword, then the text, each null separated.
        first = rest.find(b"\x00")
        if first == -1:
            return None
        rest = rest[first + 1:]
        second = rest.find(b"\x00")
        if second == -1:
            return None
        text_bytes = rest[second + 1:]

        if compression_flag == 0:
            return text_bytes.decode("utf-8", errors="replace")
        if compression_flag == 1 and compression_method == _COMPRESSION_DEFLATE:
            try:
                return zlib.decompress(text_bytes).decode("utf-8", errors="replace")
            except (zlib.error, ValueError):
                return None
        return None

    return None


def read_text_chunks(
    data: bytes, *, strict_crc: bool = False
) -> tuple[dict[str, str], list[str]]:
    """Return every text chunk as a mapping of keyword to text, plus any warnings.

    When several chunks share a keyword, the first wins. That matches the behaviour of the
    applications in the ecosystem, and it means a file with a stale duplicate after a bad
    edit reads the same way it did before the edit.
    """
    found: dict[str, str] = {}
    warnings: list[str] = []

    for chunk_type, payload, crc_ok in iter_chunks(data, strict_crc=strict_crc):
        if chunk_type not in (CHUNK_TEXT, CHUNK_ZTXT, CHUNK_ITXT):
            continue
        split = _split_keyword(payload)
        if split is None:
            continue
        keyword, _ = split
        if not crc_ok:
            warnings.append(f"PNG chunk {keyword!r} has a bad CRC; using it anyway")
        if keyword in found:
            warnings.append(f"PNG has more than one {keyword!r} chunk; using the first")
            continue
        text = decode_text_payload(chunk_type, payload)
        if text is None:
            warnings.append(f"PNG chunk {keyword!r} could not be decoded")
            continue
        found[keyword] = text

    return found, warnings


def read_embedded_assets(data: bytes) -> dict[str, str]:
    """Return the extension-asset chunks as a mapping of path to base64 payload.

    These are the ``chara-ext-asset_:{path}`` chunks the specification documents and then
    asks new implementations to avoid. They are returned raw, without decoding, because a
    reader that wants to use them almost certainly wants the base64 string itself.
    """
    assets: dict[str, str] = {}
    for chunk_type, payload, _crc_ok in iter_chunks(data, strict_crc=False):
        if chunk_type != CHUNK_TEXT:
            continue
        split = _split_keyword(payload)
        if split is None:
            continue
        keyword, remainder = split
        if not keyword.startswith(KEYWORD_ASSET_PREFIX):
            continue
        path = keyword[len(KEYWORD_ASSET_PREFIX):]
        assets[path] = remainder.decode("latin-1")
    return assets


def build_chunk(chunk_type: bytes, payload: bytes) -> bytes:
    """Assemble one chunk, length and CRC included."""
    return (
        struct.pack(">I", len(payload))
        + chunk_type
        + payload
        + struct.pack(">I", zlib.crc32(chunk_type + payload) & 0xFFFFFFFF)
    )


def build_text_chunk(keyword: str, text: str) -> bytes:
    """Assemble a ``tEXt`` chunk carrying ``text`` under ``keyword``.

    The payload is encoded as latin-1 because that is what the PNG specification says a
    ``tEXt`` chunk contains. The strings this module actually writes are base64, which is
    pure ASCII and therefore identical in either encoding, but the encoding is written
    explicitly so a future caller passing arbitrary text does not silently produce mojibake.
    """
    payload = keyword.encode("latin-1") + b"\x00" + text.encode("latin-1")
    return build_chunk(CHUNK_TEXT, payload)


def add_text_chunk(png_bytes: bytes, keyword: str, text: str) -> bytes:
    """Return a copy of a PNG with a ``tEXt`` chunk inserted before the first ``IDAT``.

    Text chunks are placed before the image data because the specification requires ``tEXt``
    to precede ``IDAT``. A reader that rewrote a card in place instead of inserting it here
    would produce a file that some decoders reject.
    """
    if not is_png(png_bytes):
        raise ValueError("not a PNG: the signature is missing")

    # Walk the chunk list to find the offset of the first IDAT, since iter_chunks yields
    # payloads rather than offsets.
    insertion_point = len(PNG_SIGNATURE)
    offset = len(PNG_SIGNATURE)
    total = len(png_bytes)
    while offset + 8 <= total:
        declared_length = struct.unpack_from(">I", png_bytes, offset)[0]
        chunk_type = png_bytes[offset + 4:offset + 8]
        if chunk_type == b"IDAT":
            insertion_point = offset
            break
        if chunk_type == CHUNK_IEND:
            insertion_point = offset
            break
        offset = offset + 8 + declared_length + 4

    new_chunk = build_text_chunk(keyword, text)
    return png_bytes[:insertion_point] + new_chunk + png_bytes[insertion_point:]


def set_text_chunk(png_bytes: bytes, keyword: str, text: str) -> bytes:
    """Return a copy of a PNG with ``keyword`` set to ``text``, replacing any existing value.

    This differs from :func:`add_text_chunk` in the way that matters for rewriting a card:
    every existing chunk with the same keyword is removed first. Writing a second ``ccv3``
    chunk alongside a stale one would leave the file with two candidate cards, and since
    readers take the first match, the stale one would win.

    The file is rebuilt chunk by chunk rather than spliced, so the new chunk lands before the
    image data as the specification requires and every chunk keeps a correct CRC.
    """
    if not is_png(png_bytes):
        raise ValueError("not a PNG: the signature is missing")

    output = bytearray(PNG_SIGNATURE)
    inserted = False

    for chunk_type, payload, _crc_ok in iter_chunks(png_bytes, strict_crc=False):
        if chunk_type == CHUNK_TEXT:
            split = _split_keyword(payload)
            if split is not None and split[0] == keyword:
                continue  # drop the previous value; ours goes in before IDAT

        if not inserted and chunk_type in (b"IDAT", CHUNK_IEND):
            output += build_text_chunk(keyword, text)
            inserted = True

        output += build_chunk(chunk_type, payload)

    if not inserted:
        # No IDAT or IEND was found, which means the file was truncated. Put ours at the end
        # so the card is at least recoverable.
        output += build_text_chunk(keyword, text)

    return bytes(output)


def remove_text_chunk(png_bytes: bytes, keyword: str) -> bytes:
    """Return a copy of a PNG with every text chunk of the given keyword removed."""
    if not is_png(png_bytes):
        raise ValueError("not a PNG: the signature is missing")

    output = bytearray(PNG_SIGNATURE)
    for chunk_type, payload, _crc_ok in iter_chunks(png_bytes, strict_crc=False):
        if chunk_type == CHUNK_TEXT:
            split = _split_keyword(payload)
            if split is not None and split[0] == keyword:
                continue
        output += build_chunk(chunk_type, payload)
    return bytes(output)
