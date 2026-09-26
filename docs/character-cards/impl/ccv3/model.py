"""The in-memory model, and the reasoning behind its shape.

The single most important design decision in this reader is that **the internal model is the
V3 shape, and every card is normalised upward into it.** V1 and V2 cards are lifted into V3
fields, where the additions start empty. V3 cards are used as they are.

The alternative, and the one taken by at least one widely deployed application, is to
normalise *downward* to V2 and keep the original file's JSON around as an opaque blob so that
nothing is lost. That works, and it is why such applications round-trip V3 cards without
damaging them. But it leaves the reader unable to answer a question about a V3 field without
first reaching into the blob, and it means the code that runs against a card has no idea
which version it is looking at.

Normalising upward avoids both problems. The cost is that a field which was absent and a
field which was present-but-empty become indistinguishable unless something records the
difference, so every object here carries an ``extra`` mapping for fields the specification
does not define, and the top-level :class:`Card` keeps the untouched original under ``raw``.

Field names follow the specification exactly, including the ones that are awkward. The
lorebook entry field ``insertion_order`` is not renamed to ``order``, and the asset field
``ext`` is not renamed to ``extension``, because the value of a reference model is that a
reader can move between it and the specification without a translation table in their head.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Optional

# ---------------------------------------------------------------------------
# Assets
# ---------------------------------------------------------------------------

#: Asset types defined by the specification. Anything else must begin with ``x_``.
KNOWN_ASSET_TYPES = ("icon", "background", "emotion", "user_icon", "other")

#: The URI scheme that refers to an asset inside a CHARX archive. The spelling with three
#: e's is deliberate and comes from the specification, which calls the alternative out by
#: name so that implementers do not "fix" it.
EMBEDDED_URI_PREFIX = "embeded://"

#: The placeholder meaning "use whatever default this application has for this asset type".
CCDEFAULT_URI_PREFIX = "ccdefault:"


@dataclass
class Asset:
    """One entry of the card's ``assets`` array.

    ``uri`` is kept as the raw string exactly as it appeared in the card. It is not resolved
    at parse time, because resolution needs to know which container the card came out of and
    whether the bytes are actually present, and those are questions the caller should be able
    to defer or skip.
    """

    type: str
    uri: str
    name: str = ""
    ext: str = ""
    extra: dict[str, Any] = field(default_factory=dict)

    @property
    def is_embedded(self) -> bool:
        """Whether the URI points inside a CHARX archive."""
        return self.uri.startswith(EMBEDDED_URI_PREFIX)

    @property
    def is_default(self) -> bool:
        """Whether the URI is the ``ccdefault:`` placeholder."""
        return self.uri.startswith(CCDEFAULT_URI_PREFIX)

    @property
    def embedded_path(self) -> Optional[str]:
        """The path inside the archive, or ``None`` if this asset is not embedded.

        The path is returned unmodified and unquoted, because it is case sensitive and
        slash separated by definition and any normalisation here would break the lookup.
        """
        if not self.is_embedded:
            return None
        return self.uri[len(EMBEDDED_URI_PREFIX):]

    @property
    def is_custom_type(self) -> bool:
        """Whether the type is an application-defined extension type."""
        return self.type.startswith("x_")


# ---------------------------------------------------------------------------
# Lorebook
# ---------------------------------------------------------------------------


@dataclass
class LorebookEntry:
    """One entry in a lorebook.

    Required fields come first and are given defaults that make an entry harmless rather than
    wrong: ``enabled`` defaults to true because a card that omits the field almost always
    meant "on", and ``content`` and ``keys`` default to empty because an entry with neither
    simply never matches.
    """

    # Required by the specification.
    keys: list[str] = field(default_factory=list)
    content: str = ""
    enabled: bool = True
    insertion_order: int = 0
    extensions: dict[str, Any] = field(default_factory=dict)

    # Optional in V2, required to be implemented in V3. The distinction means an
    # implementation must honour these when present, not that they must be present.
    use_regex: bool = False
    constant: bool = False

    # Genuinely optional.
    name: str = ""
    id: Optional[Any] = None
    comment: str = ""
    priority: Optional[int] = None
    selective: Optional[bool] = None
    secondary_keys: list[str] = field(default_factory=list)
    position: Optional[str] = None

    # The three-state fields. ``None`` means the card did not say, which the specification
    # treats differently from saying no. For ``case_sensitive`` it says true is case
    # sensitive, false is case insensitive, and undefined leaves the choice to the
    # application. For ``recursive_scanning`` it says false is *must not* recurse and
    # undefined is *may decide*. A plain ``False`` default would collapse each pair into one
    # state and make an explicit answer indistinguishable from silence, which is exactly the
    # reduction this model exists to avoid.
    case_sensitive: Optional[bool] = None

    # Anything the specification does not define. Preserved so that a card we read and write
    # back out is not quietly reduced.
    extra: dict[str, Any] = field(default_factory=dict)

    def __post_init__(self) -> None:
        if not isinstance(self.keys, list):
            self.keys = []
        if not isinstance(self.secondary_keys, list):
            self.secondary_keys = []


@dataclass
class Lorebook:
    """A lorebook, either embedded in a card or exported on its own."""

    entries: list[LorebookEntry] = field(default_factory=list)

    name: str = ""
    description: str = ""
    scan_depth: Optional[int] = None
    token_budget: Optional[int] = None
    #: ``None`` leaves the choice to the application, as the specification permits.
    recursive_scanning: Optional[bool] = None
    extensions: dict[str, Any] = field(default_factory=dict)
    extra: dict[str, Any] = field(default_factory=dict)

    def __post_init__(self) -> None:
        if not isinstance(self.entries, list):
            self.entries = []

    def __len__(self) -> int:
        return len(self.entries)


# ---------------------------------------------------------------------------
# Card data
# ---------------------------------------------------------------------------


@dataclass
class CardData:
    """The contents of a card's ``data`` object.

    The V2 fields are listed first and the V3 additions after them, in that order, because it
    makes the superset relationship visible when reading the class top to bottom.
    """

    # --- Fields carried over from TavernCardV2 -----------------------------------------
    name: str = ""
    description: str = ""
    personality: str = ""
    scenario: str = ""
    first_mes: str = ""
    mes_example: str = ""
    creator_notes: str = ""
    system_prompt: str = ""
    post_history_instructions: str = ""
    alternate_greetings: list[str] = field(default_factory=list)
    tags: list[str] = field(default_factory=list)
    creator: str = ""
    character_version: str = ""
    extensions: dict[str, Any] = field(default_factory=dict)
    character_book: Optional[Lorebook] = None

    # --- Additions in Character Card V3 -----------------------------------------------
    assets: list[Asset] = field(default_factory=list)
    nickname: str = ""
    creator_notes_multilingual: dict[str, str] = field(default_factory=dict)
    source: list[str] = field(default_factory=list)
    group_only_greetings: list[str] = field(default_factory=list)
    creation_date: Optional[int] = None
    modification_date: Optional[int] = None

    # --- Fields the specification does not define -------------------------------------
    extra: dict[str, Any] = field(default_factory=dict)

    def __post_init__(self) -> None:
        for listy in ("alternate_greetings", "tags", "assets", "source", "group_only_greetings"):
            if not isinstance(getattr(self, listy), list):
                setattr(self, listy, [])

    @property
    def display_name(self) -> str:
        """The name to show the user.

        The specification says ``nickname`` replaces ``name`` when it is set, and falls back
        to ``name`` when it is missing or empty. Both conditions are checked because a card
        may carry ``nickname: ""`` to mean "no nickname", and an empty string is falsy.
        """
        return self.nickname or self.name

    def greetings(self) -> list[str]:
        """Every greeting, with the default first.

        The order matters because ``@@is_greeting`` numbers the default greeting as zero and
        the first element of ``alternate_greetings`` as one. Do not reorder this list.
        """
        result = [self.first_mes] if self.first_mes else []
        result.extend(self.alternate_greetings)
        return result

    def assets_of_type(self, asset_type: str) -> list[Asset]:
        """Every asset of the given type, in card order."""
        return [a for a in self.assets if a.type == asset_type]

    @property
    def main_icon(self) -> Optional[Asset]:
        """The icon the card nominates as primary.

        The specification reserves the name ``main`` for exactly this purpose and requires it
        to be unique among icons. If no icon carries the name, the first icon is returned,
        which is the behaviour a reader wants when the card is informal about it.
        """
        icons = self.assets_of_type("icon")
        for asset in icons:
            if asset.name == "main":
                return asset
        return icons[0] if icons else None


# ---------------------------------------------------------------------------
# Card
# ---------------------------------------------------------------------------


@dataclass
class Card:
    """A card that has been read, with its provenance recorded.

    ``origin`` and ``raw`` exist so that the caller can always answer "what did the file
    actually say", which is the question that matters when a field looks wrong and you need
    to know whether the reader mangled it or the author did.
    """

    data: CardData = field(default_factory=CardData)

    #: The specification string as it appears in this object. This is always V3 after
    #: normalisation, since normalisation happens upward.
    spec: str = "chara_card_v3"
    spec_version: str = "3.0"

    #: What the card claimed on disk, before normalisation. One of ``chara_card_v1``,
    #: ``chara_card_v2``, or ``chara_card_v3``.
    origin_spec: str = "chara_card_v3"

    #: The numeric version the origin declared, or 3.0 when the origin declared nothing.
    origin_version: float = 3.0

    #: Which container the card was read from: ``png``, ``apng``, ``json``, ``charx``, or
    #: ``lorebook`` for a standalone lorebook file.
    container: str = "json"

    #: True when a V3 card was read from a legacy ``chara`` chunk rather than from a
    #: ``ccv3`` chunk. The user should be told, since they are seeing a reduced view.
    backfilled: bool = False

    #: Non-fatal problems found while reading. Always worth surfacing; never worth refusing
    #: the card over.
    warnings: list[str] = field(default_factory=list)

    #: The untouched object exactly as it was decoded from the container.
    raw: dict[str, Any] = field(default_factory=dict)

    #: Asset payloads by archive path, for CHARX. Empty for other containers.
    asset_blobs: dict[str, bytes] = field(default_factory=dict)

    #: Card-level fields the specification does not define.
    extra: dict[str, Any] = field(default_factory=dict)

    def __post_init__(self) -> None:
        if not isinstance(self.warnings, list):
            self.warnings = []

    @property
    def name(self) -> str:
        return self.data.name

    @property
    def display_name(self) -> str:
        return self.data.display_name

    @property
    def lorebook(self) -> Optional[Lorebook]:
        return self.data.character_book

    def lorebook_or_empty(self) -> Lorebook:
        """The lorebook, or an empty one, so callers need no None check to scan."""
        return self.data.character_book or Lorebook()

    def warn(self, message: str) -> None:
        """Record a non-fatal problem. Duplicates are dropped, since a card with fifty
        malformed entries should not produce fifty identical warnings."""
        if message not in self.warnings:
            self.warnings.append(message)
