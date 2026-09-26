"""Working out which specification a decoded object claims to follow.

This is the step that makes V2 support possible, and it is more delicate than it looks. The
specification does not say how to identify a card's version, only how to compare versions
once you have one. In practice a reader meets four shapes:

* A bare V1 card, with the character fields at the top level and no wrapper at all. V1
  predates the ``spec`` field, so there is nothing to match on and detection has to be by
  the presence of the fields themselves.
* A V2 card, wrapped as ``{"spec": "chara_card_v2", "spec_version": "2.0", "data": {...}}``.
* A V3 card, wrapped the same way with a V3 spec string.
* Something that looks like a card but lies about its version, which is common enough that
  refusing to handle it produces a reader that fails on real files.

The rule this module follows is: use the declared string when it is present and recognised,
use the numeric version when the string is missing or unrecognised, and only fall back to
shape detection when both are absent or useless. Every fallback records a warning rather than
failing, because a card that can be read is worth reading.
"""

from __future__ import annotations

from typing import Any, Optional

from .errors import MalformedCardError, UnknownSpecError

#: Spec strings defined by the specifications.
SPEC_V1 = "chara_card_v1"
SPEC_V2 = "chara_card_v2"
SPEC_V3 = "chara_card_v3"
SPEC_LOREBOOK_V3 = "lorebook_v3"

#: The version the current specification defines.
CURRENT_SPEC = SPEC_V3
CURRENT_MAJOR = 3.0

#: Fields that identify a bare V1 card. A card carrying several of these at the top level
#: with no ``data`` object is a V1 card. Two are required rather than one, because ``name``
#: alone appears in plenty of unrelated JSON.
V1_SIGNATURE_FIELDS = ("name", "description", "personality", "scenario", "first_mes", "mes_example")

#: Fields that identify a V2 ``data`` object when the spec string is missing.
V2_DATA_SIGNATURE_FIELDS = ("name", "description", "first_mes", "mes_example")

#: Fields that only exist in V3. Their presence in a ``data`` object is evidence for V3 when
#: no version is declared at all.
V3_ONLY_DATA_FIELDS = (
    "assets",
    "nickname",
    "creator_notes_multilingual",
    "source",
    "group_only_greetings",
    "creation_date",
    "modification_date",
)


def parse_version(value: Any) -> Optional[float]:
    """Coerce a declared ``spec_version`` into a float, or return ``None``.

    The specification says applications compare ``spec_version`` as a float. Cards in the
    wild carry it as a JSON number, as a string, and occasionally as something that is
    neither, so this function accepts the first two and rejects the rest rather than raising.
    A string is accepted because writing ``"2.0"`` rather than ``2.0`` is extremely common,
    and it happens to be what the dominant implementation itself emits.
    """
    if isinstance(value, bool):
        # bool is a subclass of int in Python; true is not version 1.0.
        return None
    if isinstance(value, (int, float)):
        return float(value)
    if isinstance(value, str):
        text = value.strip()
        if not text:
            return None
        try:
            return float(text)
        except ValueError:
            return None
    return None


def major_of(version: float) -> int:
    """The major component of a version, as an integer."""
    return int(version)


def is_newer_major(version: float) -> bool:
    """Whether a version's major component is ahead of the one this reader implements.

    The specification asks an application to warn but still import in this case, so callers
    use this to decide whether to warn, never whether to refuse.
    """
    return major_of(version) > major_of(CURRENT_MAJOR)


def is_older_major(version: float) -> bool:
    """Whether a version's major component is behind the one this reader implements."""
    return major_of(version) < major_of(CURRENT_MAJOR)


def version_from_spec_string(spec: str) -> Optional[float]:
    """Pull a version out of a ``chara_card_vN`` style string.

    This exists so that a card declaring ``chara_card_v4`` is understood as version 4 rather
    than being rejected as unknown. The suffix is read leniently: ``v3``, ``V3``, and ``v3.0``
    all yield 3.
    """
    if not isinstance(spec, str):
        return None
    lowered = spec.strip().lower()
    marker = "_v"
    index = lowered.rfind(marker)
    if index == -1:
        return None
    suffix = lowered[index + len(marker):]
    if not suffix:
        return None
    # Take the leading numeric run, so "3", "3.0", and "3beta" all read as 3.
    digits = []
    seen_dot = False
    for char in suffix:
        if char.isdigit():
            digits.append(char)
        elif char == "." and not seen_dot:
            digits.append(char)
            seen_dot = True
        else:
            break
    if not digits:
        return None
    try:
        return float("".join(digits))
    except ValueError:
        return None


def looks_like_v1(obj: dict[str, Any]) -> bool:
    """Whether an unwrapped object carries enough V1 fields to be a V1 card."""
    if "data" in obj and isinstance(obj.get("data"), dict):
        return False
    present = sum(1 for key in V1_SIGNATURE_FIELDS if key in obj)
    return present >= 2 and ("name" in obj or "first_mes" in obj)


def looks_like_v2_data(data: dict[str, Any]) -> bool:
    """Whether a ``data`` object carries enough V2 fields to be treated as V2 content."""
    present = sum(1 for key in V2_DATA_SIGNATURE_FIELDS if key in data)
    return present >= 2


def v3_only_fields_present(data: dict[str, Any]) -> list[str]:
    """The V3-only fields present in a ``data`` object, for use as evidence of version."""
    return [key for key in V3_ONLY_DATA_FIELDS if key in data]


def detect_version(obj: Any) -> tuple[str, float, list[str]]:
    """Identify a decoded object's specification.

    Returns a three-tuple of the spec string, the numeric version, and a list of warnings
    raised during detection. The warnings are returned rather than logged so that the caller
    can attach them to the card it is building, where the user will actually see them.

    Raises :class:`UnknownSpecError` when the object is not a card at all, and
    :class:`MalformedCardError` when it looks like a card but is shaped impossibly.
    """
    warnings: list[str] = []

    if not isinstance(obj, dict):
        raise UnknownSpecError(
            f"expected a JSON object describing a card, found {type(obj).__name__}"
        )

    declared_spec = obj.get("spec")
    declared_version = parse_version(obj.get("spec_version"))
    has_data = isinstance(obj.get("data"), dict)

    # --- Case one: a recognised spec string ------------------------------------------
    # This is the only path that needs no guessing, and it covers every conforming card.
    if isinstance(declared_spec, str):
        normalised_spec = declared_spec.strip()

        if normalised_spec == SPEC_LOREBOOK_V3:
            return SPEC_LOREBOOK_V3, 3.0, warnings

        if normalised_spec in (SPEC_V1, SPEC_V2, SPEC_V3):
            version = declared_version or {"chara_card_v1": 1.0, "chara_card_v2": 2.0, "chara_card_v3": 3.0}[normalised_spec]
            _warn_on_version(version, normalised_spec, warnings)
            return normalised_spec, version, warnings

        # An unrecognised spec string. Pull a version out of it if it follows the naming
        # convention, otherwise fall through to the numeric field.
        inferred = version_from_spec_string(normalised_spec)
        if inferred is not None:
            warnings.append(
                f"card declares an unrecognised spec {normalised_spec!r}; "
                f"treating it as version {inferred:g} by its numbering"
            )
            _warn_on_version(inferred, normalised_spec, warnings)
            return _spec_string_for_version(inferred), inferred, warnings

        if declared_version is not None:
            warnings.append(
                f"card declares an unrecognised spec {normalised_spec!r}; "
                f"falling back to spec_version {declared_version:g}"
            )
            _warn_on_version(declared_version, normalised_spec, warnings)
            return _spec_string_for_version(declared_version), declared_version, warnings

        raise UnknownSpecError(
            f"card declares spec {normalised_spec!r}, which is not recognised and does not "
            f"carry a usable spec_version"
        )

    # --- Case two: no spec string, but a version number and a data object -------------
    if has_data and declared_version is not None:
        _warn_on_version(declared_version, "", warnings)
        return _spec_string_for_version(declared_version), declared_version, warnings

    # --- Case three: no version at all, decide from the shape ------------------------
    if has_data:
        data = obj["data"]  # type: ignore[index]
        v3_evidence = v3_only_fields_present(data)
        if v3_evidence:
            warnings.append(
                "card declares no spec_version but carries V3-only fields "
                f"({', '.join(sorted(v3_evidence))}); treating it as V3"
            )
            return SPEC_V3, 3.0, warnings
        if looks_like_v2_data(data):
            warnings.append(
                "card declares no spec or spec_version but has a V2-shaped data object; "
                "treating it as V2"
            )
            return SPEC_V2, 2.0, warnings
        raise UnknownSpecError(
            "card has a data object but no spec, no spec_version, and no fields that "
            "identify its version"
        )

    # --- Case four: a bare V1 card ---------------------------------------------------
    if looks_like_v1(obj):
        return SPEC_V1, 1.0, warnings

    if "spec" in obj or "spec_version" in obj:
        raise UnknownSpecError(
            "card declares a specification but has no data object and no V1 fields"
        )

    raise UnknownSpecError(
        "object is not a character card: no spec, no spec_version, no data object, and "
        "not enough V1 fields"
    )


def _spec_string_for_version(version: float) -> str:
    """The canonical spec string for a numeric version, for versions we handle by number."""
    major = major_of(version)
    if major <= 1:
        return SPEC_V1
    if major == 2:
        return SPEC_V2
    return SPEC_V3


def _warn_on_version(version: float, spec_string: str, warnings: list[str]) -> None:
    """Append a warning when the declared version is ahead of what this reader implements."""
    if is_newer_major(version):
        label = f" {spec_string!r}" if spec_string else ""
        warnings.append(
            f"card declares{label} version {version:g}, which is newer than the {CURRENT_MAJOR:g} "
            f"this reader implements; importing on a best-effort basis and ignoring "
            f"anything unrecognised"
        )
