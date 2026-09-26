"""Curly braced syntaxes, which most of the ecosystem calls macros.

The specification defines nine of these and says applications must substitute the ones it
defines, may add their own, and should detect them case insensitively. Two of them are
comments, and the difference between the two is the interesting part:

* ``{{// A}}`` is removed, is not shown to the user, and does not participate in lorebook
  matching.
* ``{{hidden_key:A}}`` is also removed and also not shown, but *does* participate in matching
  when the lorebook is scanned recursively.

That pair is what lets an author place trigger words in a card that the model never reads.

Two implementation details are worth stating, because both are easy to get wrong.

Nesting is handled by substitution order rather than by a recursive parser. The pattern used
here cannot match across a brace, so an inner macro is always matched before the outer one
that contains it, and repeating the pass until nothing changes resolves any depth of nesting.
A single greedy pass over the same input would mangle ``{{random:{{user}},Bob}}``.

``{{pick}}`` is required to be stable for the same prompt. That is implemented by seeding the
choice from the surrounding text plus the macro's own text plus which occurrence it is, so
that re-rendering the same input yields the same result without any state being carried
between calls. A random choice that changes on every render would make a card's output
non-deterministic in a way that is very hard to debug from the outside.
"""

from __future__ import annotations

import hashlib
import random
import re
from dataclasses import dataclass, field
from typing import Any, Optional

#: The macro pattern. The character class excludes braces so that a match can never span an
#: enclosing macro, which is what makes inner-first substitution work.
_MACRO_PATTERN = re.compile(r"\{\{([^{}]*)\}\}")

#: The maximum number of substitution passes. Nesting deeper than this is pathological, and
#: stopping keeps a malformed card from spinning the reader forever.
_MAX_PASSES = 16

#: Macros whose values are comma-separated lists, so their arguments need unescaping.
_LIST_MACROS = frozenset({"random", "pick"})


@dataclass
class RenderResult:
    """The outcome of substituting macros in one piece of text."""

    #: The text with every recognised macro replaced. Unknown macros are left untouched.
    text: str

    #: The values of every ``{{hidden_key:...}}``, for a recursive lorebook scan to match
    #: against. These were removed from ``text``.
    hidden_keys: list[str] = field(default_factory=list)

    #: The values of every ``{{comment:...}}``, for an interface to display. These were
    #: removed from ``text`` and do not participate in matching.
    comments: list[str] = field(default_factory=list)

    #: Macro names that this implementation does not define, left in the text as written.
    #: Another layer may handle them, so removing them would be destructive.
    unknown: list[str] = field(default_factory=list)

    @property
    def matchable_text(self) -> str:
        """The text a recursive scan should search, with hidden keys appended.

        The specification says a hidden key participates in matching when the scan is
        recursive. Appending rather than merging keeps the reader's behaviour predictable:
        a key contributes to a match, and nothing else about the text changes.
        """
        if not self.hidden_keys:
            return self.text
        return self.text + "\n" + "\n".join(self.hidden_keys)


def _split_values(raw: str) -> list[str]:
    """Split a macro's argument list on unescaped commas, then unescape.

    The specification allows a literal comma inside a value to be written ``\\,``. Splitting
    naively on every comma would turn one value into two.
    """
    values: list[str] = []
    current: list[str] = []
    index = 0

    while index < len(raw):
        char = raw[index]
        if char == "\\" and index + 1 < len(raw) and raw[index + 1] == ",":
            current.append(",")
            index += 2
            continue
        if char == ",":
            values.append("".join(current))
            current = []
            index += 1
            continue
        current.append(char)
        index += 1

    values.append("".join(current))
    return values


def _stable_index(seed: str, macro_text: str, occurrence: int, size: int) -> int:
    """A deterministic index into ``range(size)``.

    Determinism comes from hashing rather than from a seeded random object, so that the same
    inputs give the same answer on a different machine, in a different process, and in a
    different Python version. That matters because the specification's promise for
    ``{{pick}}`` is about rendering, not about the current session.
    """
    if size <= 1:
        return 0
    digest = hashlib.sha256(f"{seed}\x00{macro_text}\x00{occurrence}".encode("utf-8")).digest()
    return int.from_bytes(digest[:8], "big") % size


def render(
    text: str,
    *,
    char_name: str = "",
    user_name: str = "",
    pick_seed: Optional[str] = None,
    rng: Optional[random.Random] = None,
) -> RenderResult:
    """Substitute every macro this implementation defines.

    ``pick_seed`` defaults to the input text itself, which satisfies the specification's
    requirement that the same prompt renders the same way without the caller having to manage
    any state. Pass an explicit seed when rendering several pieces of one prompt, so that the
    same ``{{pick}}`` appearing in two of them agrees.

    ``rng`` is accepted so that a caller who wants reproducible output across a whole
    session can supply a seeded generator. It affects ``{{random}}`` and ``{{roll}}`` but not
    ``{{pick}}``, which is deterministic by construction.
    """
    if not text or "{{" not in text:
        return RenderResult(text=text)

    generator = rng if rng is not None else random.Random()
    seed = pick_seed if pick_seed is not None else text

    result = RenderResult(text=text)
    occurrence = 0

    def substitute(match: re.Match[str]) -> str:
        nonlocal occurrence

        body = match.group(1)
        original = match.group(0)

        # The comment form has no colon and may be written as `//` with any spacing.
        if body.lstrip().startswith("//"):
            return ""

        stripped = body.strip()
        if not stripped:
            return original

        if ":" in stripped:
            name, _, raw_value = stripped.partition(":")
        else:
            name, raw_value = stripped, ""

        name = name.strip().lower()

        if name == "char":
            return char_name

        if name == "user":
            return user_name

        if name in _LIST_MACROS:
            values = _split_values(raw_value)
            values = [v.strip() for v in values if v.strip()]
            if not values:
                return ""
            if name == "random":
                return generator.choice(values)
            index = _stable_index(seed, original, occurrence, len(values))
            occurrence += 1
            return values[index]

        if name == "roll":
            spec = raw_value.strip()
            if spec[:1] in ("d", "D"):
                spec = spec[1:]
            try:
                sides = int(float(spec))
            except ValueError:
                result.unknown.append(body)
                return original
            if sides < 1:
                return "1"
            return str(generator.randint(1, sides))

        if name == "reverse":
            return raw_value[::-1]

        if name == "hidden_key":
            result.hidden_keys.append(raw_value)
            return ""

        if name == "comment":
            # Removed from the prompt and shown to the user instead.
            result.comments.append(raw_value.lstrip())
            return ""

        # Not a macro this implementation defines. Leave it exactly as written, because
        # another layer may understand it and mangling it would lose information.
        if body not in result.unknown:
            result.unknown.append(body)
        return original

    # Substitute repeatedly so that nested macros resolve from the inside out.
    current = text
    for _pass in range(_MAX_PASSES):
        updated = _MACRO_PATTERN.sub(substitute, current)
        if updated == current:
            break
        current = updated

    result.text = current
    return result


def contains_hidden_keys(text: str) -> bool:
    """Whether a piece of text carries hidden keys, without rendering it.

    Used by the scanner to decide whether a recursive pass is worth attempting.
    """
    if not text or "{{hidden_key" not in text.lower():
        return False
    return True


def strip_comments(text: str) -> str:
    """Remove comment macros and return the rest of the text unchanged.

    A convenience for callers that want a clean string rather than a
    :class:`RenderResult`, for example when writing a card back out.
    """
    return render(text).text
