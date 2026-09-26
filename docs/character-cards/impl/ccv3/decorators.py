"""Decorator parsing and resolution.

A decorator is a line at the very start of a lorebook entry's ``content`` that begins with
``@@``. It carries a name, optionally followed by a value, and a value may be a
comma-separated list. Once the reader has read them, the decorator lines are removed from the
content and the surrounding blank lines trimmed, because they are instructions to the
application rather than text for the model.

A fallback decorator begins with ``@@@`` and sits on the line immediately after the one it
stands in for. The reader walks the chain from the top and takes the first name it
recognises. This is how an author writes one entry that behaves sensibly in several different
applications at once.

There is a distinction this module takes seriously, because getting it wrong makes fallbacks
useless. A reader can *recognise* a decorator name without being able to *act* on it. This
library recognises all twenty names in the specification, but only the ones that affect
whether an entry matches and what it matches against are implemented here; the ones that
position content in a prompt are recorded and handed back to the caller, because this library
has no prompt to put them in. If a name were treated as unrecognised merely because it cannot
be acted on locally, an author's fallback chain would resolve to the wrong link and a valid
alternative would be skipped.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Optional

#: Every decorator name the specification defines. Recognition is by this set.
KNOWN_DECORATOR_NAMES = frozenset(
    {
        # Activation
        "activate",
        "dont_activate",
        "activate_only_after",
        "activate_only_every",
        "keep_activate_after_match",
        "dont_activate_after_match",
        "is_greeting",
        "is_user_icon",
        # Placement
        "depth",
        "instruct_depth",
        "reverse_depth",
        "reverse_instruct_depth",
        "position",
        "role",
        "disable_ui_prompt",
        # Matching
        "scan_depth",
        "instruct_scan_depth",
        "additional_keys",
        "exclude_keys",
        # Budget
        "ignore_on_max_context",
    }
)

#: Decorators that decide whether an entry is eligible at all.
ACTIVATION_DECORATORS = frozenset(
    {
        "activate",
        "dont_activate",
        "activate_only_after",
        "activate_only_every",
        "keep_activate_after_match",
        "dont_activate_after_match",
        "is_greeting",
        "is_user_icon",
    }
)

#: Decorators that change how keys are matched.
MATCHING_DECORATORS = frozenset(
    {"scan_depth", "instruct_scan_depth", "additional_keys", "exclude_keys"}
)

#: Decorators that say where content goes in a prompt. This library records these and hands
#: them to the caller rather than acting on them.
PLACEMENT_DECORATORS = frozenset(
    {"depth", "instruct_depth", "reverse_depth", "reverse_instruct_depth", "position", "role",
     "disable_ui_prompt"}
)

#: Decorators that affect trimming when the context window is full.
BUDGET_DECORATORS = frozenset({"ignore_on_max_context"})

#: Values the specification allows for ``@@position``. Applications may add more; an
#: unrecognised value means the decorator is ignored.
KNOWN_POSITIONS = frozenset({"after_desc", "before_desc", "personality", "scenario"})

#: Values the specification allows for ``@@role``.
KNOWN_ROLES = frozenset({"assistant", "system", "user"})

#: Values the specification allows for ``@@disable_ui_prompt``.
KNOWN_UI_PROMPTS = frozenset({"post_history_instructions", "system_prompt"})

#: Decorators allowed to appear more than once on one entry.
REPEATABLE_DECORATORS = frozenset({"additional_keys"})

#: The minimum fallback chain length an implementation is asked to support.
MIN_FALLBACK_CHAIN = 5

_DECORATOR_PREFIX = "@@"
_FALLBACK_PREFIX = "@@@"


@dataclass
class Decorator:
    """One decorator line, already split into a name and its values."""

    name: str
    values: list[str] = field(default_factory=list)
    fallback: bool = False
    raw: str = ""

    @property
    def value(self) -> str:
        """The first value, or an empty string, for the single-valued decorators."""
        return self.values[0] if self.values else ""

    def as_int(self) -> Optional[int]:
        """The first value as an integer, or ``None`` when it is not one."""
        if not self.values:
            return None
        try:
            return int(float(self.values[0].strip()))
        except ValueError:
            return None

    def as_bool(self) -> Optional[bool]:
        """The first value as a boolean, or ``None`` when it is not one."""
        if not self.values:
            return None
        lowered = self.values[0].strip().lower()
        if lowered in ("true", "yes", "1"):
            return True
        if lowered in ("false", "no", "0"):
            return False
        return None


@dataclass
class ParsedContent:
    """The result of reading decorators off an entry's content.

    ``cleaned`` is what should be injected into a prompt or matched against. ``directives``
    holds the resolved decorators, keyed by name.
    """

    cleaned: str
    directives: dict[str, list[Decorator]] = field(default_factory=dict)
    warnings: list[str] = field(default_factory=list)

    def get(self, name: str) -> Optional[Decorator]:
        """The first resolved decorator with this name, or ``None``."""
        found = self.directives.get(name)
        return found[0] if found else None

    def all(self, name: str) -> list[Decorator]:
        """Every resolved decorator with this name, which matters for the repeatable ones."""
        return list(self.directives.get(name, ()))

    def has(self, name: str) -> bool:
        return name in self.directives

    @property
    def is_placement_aware(self) -> bool:
        """Whether any decorator here asks for a specific prompt position."""
        return any(name in PLACEMENT_DECORATORS for name in self.directives)


def _split_line(line: str) -> Optional[tuple[str, list[str]]]:
    """Split a decorator line into a name and a list of values.

    The name is the first whitespace-delimited token after the prefix. Everything after it is
    one value string, split on commas.
    """
    body = line.strip()
    if body.startswith(_FALLBACK_PREFIX):
        body = body[len(_FALLBACK_PREFIX):]
    elif body.startswith(_DECORATOR_PREFIX):
        body = body[len(_DECORATOR_PREFIX):]
    else:
        return None

    body = body.strip()
    if not body:
        return None

    parts = body.split(None, 1)
    name = parts[0].strip().lower()
    if not name:
        return None

    values: list[str] = []
    if len(parts) > 1:
        for value in parts[1].split(","):
            stripped = value.strip()
            if stripped:
                values.append(stripped)
    return name, values


def parse(content: str) -> ParsedContent:
    """Read decorators off the front of ``content`` and return the cleaned remainder.

    Parsing stops at the first line that is not a decorator, which is what makes a decorator
    line in the middle of an entry plain text. Leading blank lines are skipped, since writers
    regularly leave one after the block.

    Fallback chains are resolved here: for each chain, the first name this module recognises
    wins and the rest are discarded. A chain with no recognised name contributes nothing, and
    the entry keeps its content.
    """
    result = ParsedContent(cleaned=content)

    if not content:
        return result

    lines = content.split("\n")
    index = 0

    # Skip blank lines so that a card with a stray newline before its decorators still works.
    while index < len(lines) and not lines[index].strip():
        index += 1

    consumed = index
    pending_primary: Optional[Decorator] = None
    chain: list[Decorator] = []
    chain_had_any = False

    def _flush_chain() -> None:
        """Resolve the chain accumulated so far and store the winner."""
        nonlocal chain, pending_primary
        if not chain:
            pending_primary = None
            return
        winner = next((item for item in chain if item.name in KNOWN_DECORATOR_NAMES), None)
        if winner is None:
            names = ", ".join(repr(item.name) for item in chain)
            result.warnings.append(
                f"ignored unrecognised decorator chain ({names}); the entry keeps its content"
            )
        else:
            if len(chain) > MIN_FALLBACK_CHAIN:
                result.warnings.append(
                    f"decorator fallback chain for {winner.name!r} is {len(chain)} long, "
                    f"past the {MIN_FALLBACK_CHAIN} implementations are asked to support"
                )
            existing = result.directives.setdefault(winner.name, [])
            if existing and winner.name not in REPEATABLE_DECORATORS:
                result.warnings.append(
                    f"decorator {winner.name!r} appears more than once on this entry; only "
                    f"the first is used"
                )
            else:
                existing.append(winner)
        chain = []
        pending_primary = None

    while index < len(lines):
        line = lines[index]
        stripped = line.strip()

        if not stripped:
            # A blank line inside the decorator block. Keep going only if we are mid-chain and
            # the next content line is still a decorator; otherwise the block has ended.
            lookahead = index + 1
            while lookahead < len(lines) and not lines[lookahead].strip():
                lookahead += 1
            if lookahead < len(lines) and lines[lookahead].strip().startswith(_DECORATOR_PREFIX):
                index += 1
                continue
            break

        if not stripped.startswith(_DECORATOR_PREFIX):
            break

        split = _split_line(stripped)
        if split is None:
            break

        name, values = split
        is_fallback = stripped.startswith(_FALLBACK_PREFIX)

        if is_fallback and not chain:
            # A fallback with nothing to fall back from. Drop it and say so.
            result.warnings.append(
                f"fallback decorator {name!r} has no preceding decorator; ignored"
            )
            index += 1
            consumed = index
            continue

        item = Decorator(name=name, values=values, fallback=is_fallback, raw=stripped)
        if not is_fallback:
            _flush_chain()
        chain.append(item)
        chain_had_any = True

        index += 1
        consumed = index

    _flush_chain()

    if chain_had_any:
        result.cleaned = "\n".join(lines[consumed:]).lstrip("\n")

    _validate_directives(result)
    return result


def _validate_directives(result: ParsedContent) -> None:
    """Check resolved decorator values and drop the ones that are unusable.

    The specification says an invalid value means the decorator is ignored, and the entry is
    kept. That is what happens here: the decorator is removed from the resolved set and a
    warning is recorded.
    """
    numeric_only = (
        "activate_only_after",
        "activate_only_every",
        "depth",
        "instruct_depth",
        "reverse_depth",
        "reverse_instruct_depth",
        "scan_depth",
        "instruct_scan_depth",
        "is_greeting",
    )

    for name in numeric_only:
        for item in result.all(name):
            if item.as_int() is None:
                result.warnings.append(
                    f"decorator @@{name} requires a number, got "
                    f"{item.value!r}; ignored"
                )
                result.directives[name].remove(item)

    for item in result.all("position"):
        if item.value not in KNOWN_POSITIONS:
            result.warnings.append(
                f"decorator @@position value {item.value!r} is not one of "
                f"{', '.join(sorted(KNOWN_POSITIONS))}; ignored"
            )
            result.directives["position"].remove(item)

    for item in result.all("role"):
        if item.value not in KNOWN_ROLES:
            result.warnings.append(
                f"decorator @@role value {item.value!r} is not one of "
                f"{', '.join(sorted(KNOWN_ROLES))}; ignored"
            )
            result.directives["role"].remove(item)

    for item in result.all("disable_ui_prompt"):
        if item.value not in KNOWN_UI_PROMPTS:
            result.warnings.append(
                f"decorator @@disable_ui_prompt value {item.value!r} is not one of "
                f"{', '.join(sorted(KNOWN_UI_PROMPTS))}; ignored"
            )
            result.directives["disable_ui_prompt"].remove(item)

    # Clean out any name whose list became empty, so callers can use truthiness.
    for name in [n for n, items in result.directives.items() if not items]:
        del result.directives[name]


def strip_all(content: str) -> str:
    """Remove every decorator line from content, without resolving anything.

    This is what the specification asks for when backfilling a V3 card to V2: decorators are
    meaningless to a V2 reader and would otherwise appear in the prompt as literal text.
    Unlike :func:`parse`, this keeps peeling decorator lines even when a chain resolves to
    nothing, and it does not stop at an unrecognised name.
    """
    if not content:
        return content

    lines = content.split("\n")
    index = 0
    while index < len(lines):
        stripped = lines[index].strip()
        if not stripped:
            index += 1
            continue
        if stripped.startswith(_DECORATOR_PREFIX):
            index += 1
            continue
        break

    return "\n".join(lines[index:]).lstrip("\n")


def summarise(parsed: ParsedContent) -> dict[str, Any]:
    """A compact, JSON-serialisable view of the resolved directives.

    Callers that are assembling a prompt want to know what the entry asked for without
    walking the decorator objects themselves.
    """
    summary: dict[str, Any] = {}

    for name, items in parsed.directives.items():
        if name in REPEATABLE_DECORATORS:
            summary[name] = [item.values for item in items]
        else:
            summary[name] = items[0].values if items[0].values else True

    if parsed.is_placement_aware:
        summary["_needs_prompt"] = True

    return summary
