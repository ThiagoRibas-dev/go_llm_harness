"""Lifting V1, V2, and V3 cards into one internal shape.

V3 is a superset of V2, and V2 is a superset of V1 in everything but the wrapper. That means
a card can always be normalised *upward*: read whatever the file declares, then populate the
V3 fields, leaving the V3-only ones empty for older cards. After this module runs, nothing
above it needs to know which specification the file used.

There are two consequences worth being explicit about.

The first is that normalising upward is what makes V2 retro-compatibility free rather than
special-cased. A V2 card becomes a V3 card that happens to have no assets, no nickname, and
no multilingual notes. Every consumer above this layer reads one shape. The alternative used
by at least one widely deployed application is to normalise downward to V2 and keep the
original file around as an opaque blob; that round-trips safely but leaves the V3 fields
unreachable without going back through the blob.

The second is that unknown fields have to be preserved deliberately. Lifting known fields out
of a dictionary and dropping the rest would silently delete every field the specification
adds after this reader was written, which is precisely the failure the specification warns
about when it asks applications to ignore unknown fields rather than reject them. So every
object built here carries an ``extra`` mapping holding everything that was not recognised,
and the writer in :mod:`ccv3.export` puts it back.

Where a field has the wrong type, the reader coerces when the intent is unmistakable and
records a warning. A card whose ``tags`` is a comma-separated string rather than an array is
read as an array of tags, because that is what the author meant; a card whose ``name`` is an
object is read as an empty name, because nothing about the intent is recoverable.
"""

from __future__ import annotations

from typing import Any, Optional

from .model import Asset, Card, CardData, Lorebook, LorebookEntry

# ---------------------------------------------------------------------------
# Field inventories
# ---------------------------------------------------------------------------

#: Every field V3 defines inside ``data``. Anything else found there goes to ``extra``.
KNOWN_DATA_FIELDS = frozenset(
    {
        # V2 and earlier
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
        "character_book",
        # V3 additions
        "assets",
        "nickname",
        "creator_notes_multilingual",
        "source",
        "group_only_greetings",
        "creation_date",
        "modification_date",
    }
)

#: Every top-level field a card object may carry. Anything else goes to the card's ``extra``.
KNOWN_CARD_FIELDS = frozenset({"spec", "spec_version", "data"})

#: Every field a lorebook may carry.
KNOWN_LOREBOOK_FIELDS = frozenset(
    {
        "name",
        "description",
        "scan_depth",
        "token_budget",
        "recursive_scanning",
        "extensions",
        "entries",
    }
)

#: Every field a lorebook entry may carry.
KNOWN_ENTRY_FIELDS = frozenset(
    {
        "keys",
        "content",
        "extensions",
        "enabled",
        "insertion_order",
        "use_regex",
        "constant",
        "name",
        "id",
        "comment",
        "priority",
        "case_sensitive",
        "selective",
        "secondary_keys",
        "position",
    }
)

#: Names used by older tooling for the same idea, mapped onto the field we store it in. These
#: are carried across rather than discarded so that a V1 card does not lose its lorebook.
LEGACY_ALIASES = {
    "character_book": ("character_book", "char_book"),
    "creator_notes": ("creator_notes", "creatorcomment", "creator_notes"),
}

#: SillyTavern's flat V1 extension fields, kept in ``extensions`` so they survive a round
#: trip without being promoted to specification fields that mean something slightly different.
FLAT_ST_EXTENSION_FIELDS = (
    "talkativeness",
    "fav",
    "depth_prompt_prompt",
    "depth_prompt_depth",
    "depth_prompt_role",
    "avatar",
    "chat",
)


# ---------------------------------------------------------------------------
# Tolerant coercion
# ---------------------------------------------------------------------------


def as_str(value: Any, *, default: str = "") -> str:
    """Coerce a value to a string, treating ``None`` as absent.

    Numbers are stringified because cards in the wild carry versions and identifiers as
    numbers where the specification says string. Containers are rejected, since there is no
    reading of a list as a name that the author could have intended.
    """
    if value is None:
        return default
    if isinstance(value, str):
        return value
    if isinstance(value, bool):
        return default
    if isinstance(value, (int, float)):
        return str(value)
    return default


def as_bool(value: Any, *, default: Optional[bool] = False) -> Optional[bool]:
    """Coerce a value to a boolean, accepting the string forms that appear in real cards."""
    if isinstance(value, bool):
        return value
    if isinstance(value, str):
        lowered = value.strip().lower()
        if lowered in ("true", "yes", "1", "on"):
            return True
        if lowered in ("false", "no", "0", "off", ""):
            return False
    if isinstance(value, (int, float)):
        return bool(value)
    return default


def as_int(value: Any, *, default: Optional[int] = None) -> Optional[int]:
    """Coerce a value to an integer, or return the default when it cannot be read as one."""
    if isinstance(value, bool):
        return default
    if isinstance(value, int):
        return value
    if isinstance(value, float):
        return int(value)
    if isinstance(value, str):
        try:
            return int(float(value.strip()))
        except ValueError:
            return default
    return default


def as_str_list(value: Any) -> list[str]:
    """Coerce a value to a list of strings.

    A bare string is split on commas. That is not in the specification, but a comma-separated
    string is a common authoring mistake and reading it as one tag is worse than reading it
    as three. Non-string members are stringified, and members that cannot be stringified are
    dropped.
    """
    if value is None:
        return []
    if isinstance(value, str):
        return [part.strip() for part in value.split(",") if part.strip()]
    if isinstance(value, (list, tuple)):
        result: list[str] = []
        for item in value:
            if isinstance(item, str):
                result.append(item)
            elif isinstance(item, (int, float)) and not isinstance(item, bool):
                result.append(str(item))
        return result
    return []


def as_dict(value: Any) -> dict[str, Any]:
    """Coerce a value to a dictionary, returning an empty one when it is not a mapping."""
    return dict(value) if isinstance(value, dict) else {}


def as_dict_of_str(value: Any) -> dict[str, str]:
    """Coerce a value to a mapping of string to string, dropping unusable members."""
    if not isinstance(value, dict):
        return {}
    return {str(k): str(v) for k, v in value.items() if isinstance(v, (str, int, float)) and not isinstance(v, bool)}


#: Values above this are treated as milliseconds rather than seconds. One hundred billion
#: seconds is the year 5138, so nothing legitimate approaches it, while the current date in
#: milliseconds is around 1.8e12. This threshold was originally set three orders of magnitude
#: higher, which silently passed twelve- and thirteen-digit millisecond timestamps through as
#: if they were seconds, putting every card's creation date roughly fifty thousand years in
#: the future. Real cards from a widely used export tool use exactly that format.
_MILLISECOND_THRESHOLD = 10**11


def as_timestamp(value: Any) -> Optional[int]:
    """Coerce a value to a unix timestamp in seconds.

    The specification says ``creation_date`` and ``modification_date`` are unix seconds in
    UTC, and that ``0`` means unknown. A zero is therefore stored as ``None``, so that
    callers testing for presence get the answer the specification intends rather than a
    truthy-looking number.

    Cards in the wild carry three other things in these fields. Some write an ISO 8601 string,
    which is the convention in one widely used application. Some write milliseconds. Some
    write microseconds. The unit is inferred from the magnitude and normalised down, rather
    than being trusted.
    """
    if isinstance(value, str):
        parsed_from_string = _timestamp_from_iso(value)
        if parsed_from_string is not None:
            return parsed_from_string

    parsed = as_int(value)
    if parsed is None or parsed == 0:
        return None

    # Normalise down through milliseconds to seconds. The loop rather than a single division
    # because a microsecond timestamp needs two passes.
    while parsed > _MILLISECOND_THRESHOLD:
        parsed //= 1000

    return parsed


def _timestamp_from_iso(text: str) -> Optional[int]:
    """Parse an ISO 8601 timestamp into unix seconds, or return ``None``.

    The trailing ``Z`` that means UTC is accepted, which older Python versions reject. The
    result is treated as UTC when the string carries no offset, since a card's creation date
    is not a moment in the reader's local time.
    """
    from datetime import datetime, timezone

    candidate = text.strip()
    if not candidate:
        return None
    if candidate.endswith(("Z", "z")):
        candidate = candidate[:-1] + "+00:00"
    try:
        moment = datetime.fromisoformat(candidate)
    except ValueError:
        return None
    if moment.tzinfo is None:
        moment = moment.replace(tzinfo=timezone.utc)
    return int(moment.timestamp())


def extract_extras(source: dict[str, Any], known: frozenset[str]) -> dict[str, Any]:
    """Every field of ``source`` that is not in ``known``.

    This is the whole of the forward-compatibility story. Fields we do not understand are
    kept, not dropped, so a card written against a later version of the specification
    survives a round trip through this reader.
    """
    return {key: value for key, value in source.items() if key not in known}


# ---------------------------------------------------------------------------
# Lorebook
# ---------------------------------------------------------------------------


def normalise_entry(raw: Any, *, index: int, warnings: list[str]) -> LorebookEntry:
    """Build one lorebook entry from decoded JSON."""
    if not isinstance(raw, dict):
        warnings.append(f"lorebook entry {index} is a {type(raw).__name__}, not an object; skipped")
        return LorebookEntry()

    # Absent stays absent. The specification lists no default for this field, so inventing
    # one on the way in would both misreport what the card said and make the exporter unable
    # to round-trip an entry that genuinely said `before_char`.
    position = as_str(raw.get("position")) or None
    if position is not None and position not in ("before_char", "after_char"):
        warnings.append(
            f"lorebook entry {index} has position {position!r}, which is neither "
            f"'before_char' nor 'after_char'; ignoring it"
        )
        position = None

    entry_id = raw.get("id")
    if isinstance(entry_id, (dict, list)):
        entry_id = None

    return LorebookEntry(
        keys=as_str_list(raw.get("keys")),
        content=as_str(raw.get("content")),
        enabled=as_bool(raw.get("enabled"), default=True),
        insertion_order=as_int(raw.get("insertion_order"), default=0) or 0,
        extensions=as_dict(raw.get("extensions")),
        use_regex=as_bool(raw.get("use_regex"), default=False),
        constant=as_bool(raw.get("constant"), default=False),
        name=as_str(raw.get("name")),
        id=entry_id,
        comment=as_str(raw.get("comment")),
        priority=as_int(raw.get("priority")),
        case_sensitive=as_bool(raw.get("case_sensitive"), default=None),
        selective=as_bool(raw.get("selective"), default=None),
        secondary_keys=as_str_list(raw.get("secondary_keys")),
        position=position,
        extra=extract_extras(raw, KNOWN_ENTRY_FIELDS),
    )


def normalise_lorebook(raw: Any, *, warnings: list[str], where: str) -> Optional[Lorebook]:
    """Build a lorebook from decoded JSON, or return ``None`` when there is nothing there.

    A lorebook with no ``entries`` array is treated as absent rather than as an empty book.
    The two are indistinguishable to a scanner, and ``None`` lets the caller tell whether the
    card carried one at all.
    """
    if raw is None:
        return None
    if not isinstance(raw, dict):
        warnings.append(f"{where} is a {type(raw).__name__}, not an object; ignored")
        return None
    if "entries" not in raw:
        warnings.append(f"{where} has no 'entries' array; ignored")
        return None

    raw_entries = raw.get("entries")
    if not isinstance(raw_entries, list):
        warnings.append(f"{where} has an 'entries' field that is not an array; ignored")
        return None

    entries = [
        normalise_entry(item, index=index, warnings=warnings)
        for index, item in enumerate(raw_entries)
    ]
    entries = [entry for entry in entries if entry.content or entry.keys]

    scan_depth = as_int(raw.get("scan_depth"))
    if scan_depth is not None and scan_depth < 0:
        warnings.append(f"{where} declares a negative scan_depth; treating it as absent")
        scan_depth = None

    token_budget = as_int(raw.get("token_budget"))
    if token_budget is not None and token_budget < 0:
        warnings.append(f"{where} declares a negative token_budget; treating it as absent")
        token_budget = None

    return Lorebook(
        entries=entries,
        name=as_str(raw.get("name")),
        description=as_str(raw.get("description")),
        scan_depth=scan_depth,
        token_budget=token_budget,
        recursive_scanning=as_bool(raw.get("recursive_scanning"), default=None),
        extensions=as_dict(raw.get("extensions")),
        extra=extract_extras(raw, KNOWN_LOREBOOK_FIELDS),
    )


# ---------------------------------------------------------------------------
# Assets
# ---------------------------------------------------------------------------


def normalise_assets(raw: Any, *, warnings: list[str]) -> list[Asset]:
    """Build the asset list, keeping the raw URI and type strings unmodified."""
    if raw is None:
        return []
    if not isinstance(raw, list):
        warnings.append("data.assets is not an array; ignored")
        return []

    assets: list[Asset] = []
    seen_main_icons = 0

    for index, item in enumerate(raw):
        if not isinstance(item, dict):
            warnings.append(f"asset {index} is a {type(item).__name__}, not an object; skipped")
            continue

        asset_type = as_str(item.get("type"))
        uri = as_str(item.get("uri"))
        if not asset_type or not uri:
            warnings.append(f"asset {index} has no 'type' or no 'uri'; skipped")
            continue

        name = as_str(item.get("name"))
        ext = as_str(item.get("ext"))

        if asset_type == "icon" and name == "main":
            seen_main_icons += 1

        assets.append(
            Asset(
                type=asset_type,
                uri=uri,
                name=name,
                ext=ext,
                extra=extract_extras(item, frozenset({"type", "uri", "name", "ext"})),
            )
        )

    if seen_main_icons > 1:
        warnings.append(
            f"card has {seen_main_icons} assets of type 'icon' named 'main', but the name is "
            f"required to be unique; the first is used"
        )

    # A custom asset type must begin with x_. An unrecognised prefix is a warning rather than
    # an error, because refusing the card over it would be out of proportion.
    for asset in assets:
        if asset.type not in ("icon", "background", "emotion", "user_icon", "other"):
            if not asset.is_custom_type:
                warnings.append(
                    f"asset type {asset.type!r} is not one of the specification's types and "
                    f"does not begin with 'x_'"
                )

    return assets


# ---------------------------------------------------------------------------
# Card data
# ---------------------------------------------------------------------------


#: The two key spellings a nested metadata object uses for the creation date, in the order
#: they are checked. The first is the V1-era convention, the second is what one widely used
#: application writes.
_METADATA_CREATED_KEYS = ("created", "create_date", "creation_date")
_METADATA_MODIFIED_KEYS = ("modified", "modify_date", "modification_date", "last_modified")


def _dates_from_metadata(
    raw: dict[str, Any], *, top_level: Optional[dict[str, Any]] = None
) -> tuple[Optional[int], Optional[int]]:
    """Find creation and modification dates in the places cards actually keep them.

    Neither spelling here is in the specification, and the location is not fixed either. A V1
    card has no ``data`` wrapper, so its ``metadata`` object is the same dictionary the
    character fields came from. A V2 card from the same tool keeps ``metadata`` *beside*
    ``data`` rather than inside it. So both containers are searched, in order, rather than
    assuming whichever one the first card you happened to test used.

    The direct top-level keys are checked as a last resort, because at least one application
    writes the creation date of a V2 card outside ``data`` while leaving the specification's
    own field inside it unset.
    """
    created: Optional[int] = None
    modified: Optional[int] = None

    containers: list[dict[str, Any]] = [raw]
    if top_level is not None and top_level is not raw:
        containers.append(top_level)

    for container in containers:
        metadata = container.get("metadata")

        if isinstance(metadata, dict):
            if created is None:
                for key in _METADATA_CREATED_KEYS:
                    if key in metadata:
                        created = as_timestamp(metadata[key])
                        if created is not None:
                            break
            if modified is None:
                for key in _METADATA_MODIFIED_KEYS:
                    if key in metadata:
                        modified = as_timestamp(metadata[key])
                        if modified is not None:
                            break

        if created is None:
            for key in ("create_date", "created", "creation_date"):
                if key in container:
                    created = as_timestamp(container[key])
                    if created is not None:
                        break
        if modified is None:
            for key in ("modify_date", "modified", "modification_date", "last_modified"):
                if key in container:
                    modified = as_timestamp(container[key])
                    if modified is not None:
                        break

        if created is not None and modified is not None:
            break

    return created, modified


def normalise_data(
    raw: dict[str, Any],
    *,
    spec: str,
    warnings: list[str],
    top_level: Optional[dict[str, Any]] = None,
) -> CardData:
    """Build :class:`CardData` from a decoded ``data`` object, or from a bare V1 card.

    A V1 card has no ``data`` object, so its fields sit at the top level alongside a handful
    of application extension fields. Both cases arrive here as a plain dictionary and are
    treated identically; the difference is that a V1 dictionary contains extra keys, which
    end up in ``extensions`` or in ``extra`` rather than being mistaken for specification
    fields.
    """
    # --- fields shared across every version ------------------------------------------
    creator_notes = ""
    for alias in LEGACY_ALIASES["creator_notes"]:
        if alias in raw:
            creator_notes = as_str(raw.get(alias))
            break

    character_book_raw = None
    for alias in LEGACY_ALIASES["character_book"]:
        if alias in raw:
            character_book_raw = raw.get(alias)
            break

    extensions = as_dict(raw.get("extensions"))

    # SillyTavern's flat V1 fields are folded into extensions so they survive a round trip
    # without being promoted to specification fields with slightly different meanings.
    for field_name in FLAT_ST_EXTENSION_FIELDS:
        if field_name in raw and field_name not in extensions:
            value = raw[field_name]
            if value not in (None, ""):
                extensions[field_name] = value

    # Dates are the one field where three conventions coexist and none of them is the one the
    # specification defines. The specification's own field wins when present; otherwise the
    # nested `metadata` object is consulted, which is where V1-era tools and one widely used
    # application both put them, under two different spellings.
    creation_date = as_timestamp(raw.get("creation_date"))
    modification_date = as_timestamp(raw.get("modification_date"))
    metadata_created, metadata_modified = _dates_from_metadata(
        raw, top_level=top_level
    )
    if creation_date is None:
        creation_date = metadata_created
    if modification_date is None:
        modification_date = metadata_modified

    data = CardData(
        name=as_str(raw.get("name")),
        description=as_str(raw.get("description")),
        personality=as_str(raw.get("personality")),
        scenario=as_str(raw.get("scenario")),
        first_mes=as_str(raw.get("first_mes")),
        mes_example=as_str(raw.get("mes_example")),
        creator_notes=creator_notes,
        system_prompt=as_str(raw.get("system_prompt")),
        post_history_instructions=as_str(raw.get("post_history_instructions")),
        alternate_greetings=as_str_list(raw.get("alternate_greetings")),
        tags=as_str_list(raw.get("tags")),
        creator=as_str(raw.get("creator")),
        character_version=as_str(raw.get("character_version")),
        extensions=extensions,
        character_book=normalise_lorebook(
            character_book_raw, warnings=warnings, where="data.character_book"
        ),
        assets=normalise_assets(raw.get("assets"), warnings=warnings),
        nickname=as_str(raw.get("nickname")),
        creator_notes_multilingual=as_dict_of_str(raw.get("creator_notes_multilingual")),
        source=as_str_list(raw.get("source")),
        group_only_greetings=as_str_list(raw.get("group_only_greetings")),
        creation_date=creation_date,
        modification_date=modification_date,
    )

    # Unknown fields are preserved. For a V2 card this is where nothing accumulates, since
    # every V2 field is known. For a V1 card it is where the application's own fields and
    # anything unrecognised end up.
    known_for_extras = set(KNOWN_DATA_FIELDS)
    if spec == "chara_card_v1":
        known_for_extras |= set(FLAT_ST_EXTENSION_FIELDS)
        known_for_extras |= set(LEGACY_ALIASES["creator_notes"])
        known_for_extras |= set(LEGACY_ALIASES["character_book"])
    data.extra = extract_extras(raw, frozenset(known_for_extras))

    # --- language coverage check ------------------------------------------------------
    # The specification keys creator_notes_multilingual by ISO 639-1 code. A two-letter key
    # is the signal we can check cheaply; anything else is reported so the author can see it.
    for key in data.creator_notes_multilingual:
        if len(key) != 2 or not key.isalpha():
            warnings.append(
                f"creator_notes_multilingual key {key!r} is not a two-letter ISO 639-1 "
                f"language code"
            )

    return data


def normalise_card(
    obj: dict[str, Any],
    *,
    spec: str,
    version: float,
    container: str,
    container_warnings: Optional[list[str]] = None,
    from_chunk: Optional[str] = None,
    asset_blobs: Optional[dict[str, bytes]] = None,
) -> Card:
    """Build a fully populated :class:`Card` from a decoded object and its detected version.

    The warnings list passed in from version detection and from the container layer is carried
    onto the card here, so that a caller has one place to look for everything the reader
    wanted to mention.
    """
    warnings: list[str] = list(container_warnings or [])

    if spec == "chara_card_v1":
        # A V1 card is its own data object.
        raw_data = obj
    else:
        raw_data = as_dict(obj.get("data"))
        if not raw_data and spec == "chara_card_v2":
            warnings.append("card declares V2 but has an empty or missing data object")

    # The card's top level is passed in as a second place to look for dates, because a V2 card
    # from at least one application carries create_date outside `data` while leaving the
    # specification's own field inside it unset.
    data = normalise_data(raw_data, spec=spec, warnings=warnings, top_level=obj)

    if not data.name:
        warnings.append(
            "card has no name. The specification requires one; the card is still readable "
            "but will be hard to identify."
        )

    card = Card(
        data=data,
        spec="chara_card_v3",
        spec_version="3.0",
        origin_spec=spec,
        origin_version=version,
        container=container,
        raw=obj,
        asset_blobs=dict(asset_blobs or {}),
        extra=extract_extras(obj, KNOWN_CARD_FIELDS) if spec != "chara_card_v1" else {},
    )

    # Reading a card out of the legacy chunk is only a backfill if the card is not V3 already,
    # since a V3 card read from a `chara` chunk has lost nothing.
    if from_chunk == "chara" and spec != "chara_card_v3":
        card.backfilled = True
        warnings.append(
            "card was read from the legacy 'chara' PNG chunk because no 'ccv3' chunk was "
            "present. If this file was written by a V3-aware tool, V3 data may be missing."
        )

    for message in warnings:
        card.warn(message)

    return card
