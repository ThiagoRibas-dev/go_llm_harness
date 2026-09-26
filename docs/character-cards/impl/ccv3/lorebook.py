"""Scanning a lorebook, which is where most of the specification's behaviour lives.

Matching is against the chat log rather than against the character card. An entry matches
when one of its keys appears in the recent messages; how recent is bounded by ``scan_depth``.
Once an entry matches, its content is injected exactly once no matter how many times its keys
appear, and the injected entries are ordered by ``insertion_order`` before being fitted into
``token_budget``.

The rules this module implements, in the order they are applied to each entry:

1. A disabled entry never matches. This is absolute and is checked before anything else.
2. An entry with no content contributes nothing, so it is skipped even if it matched.
3. ``@@activate`` forces a match regardless of keys; ``@@dont_activate`` forbids a match
   unless ``@@activate`` is also present.
4. The remaining activation decorators gate on facts about the conversation: how many
   messages have been sent, which greeting is active, which user icon is in use, and whether
   the entry has matched before.
5. ``constant`` entries match without needing a key.
6. Otherwise keys are matched, widened by ``@@additional_keys`` and vetoed by
   ``@@exclude_keys``.
7. ``selective`` entries additionally require one of their ``secondary_keys``.

Two design points are worth stating because they are choices rather than transcriptions.

The scanner has no prompt, so decorators that place content in a prompt cannot be applied. It
returns them on the hit instead, in a ``directives`` mapping, so a caller building a prompt
can act on them. A scanner that silently discarded them would make cards that rely on
``@@position`` render wrongly with no way to find out why.

Token counting is pluggable. There is no tokeniser in the standard library and the choice of
tokeniser belongs to whoever is calling this, so the default is a documented approximation
and a caller with a real tokeniser passes it in.
"""

from __future__ import annotations

import re
from dataclasses import dataclass, field
from typing import Any, Callable, Iterable, Optional, Sequence

from . import cbs, decorators
from .model import Lorebook, LorebookEntry

#: Default scanner behaviour.
DEFAULT_SCAN_DEPTH = 4
DEFAULT_RECURSION_LIMIT = 3

#: Patterns longer than this are treated as invalid rather than attempted. The specification
#: recommends the re2 engine and explicitly allows an application to give up on a pattern it
#: judges too expensive; without re2, a length cap is the cheapest guard available.
MAX_REGEX_LENGTH = 512

#: How many characters of scanned text are handed to a regular expression. Python's engine
#: has no timeout, so bounding the input is the remaining option.
MAX_REGEX_SCAN_CHARS = 100_000


def approximate_tokens(text: str) -> int:
    """A rough token count, for callers that do not supply a tokeniser.

    Four characters per token is close enough for budgeting on English prose and wrong in
    predictable ways: it undercounts code and overcounts text with many short words. It
    exists so that ``token_budget`` has some effect by default, not because it is accurate.
    Pass a real tokeniser into :func:`scan` when the budget has to be honoured precisely.
    """
    if not text:
        return 0
    return max(1, len(text) // 4)


@dataclass
class ScanContext:
    """Everything about the conversation that the scan needs to know.

    Every field is optional, because a caller reading a card before any conversation exists
    should be able to scan and get sensible results. Where a decorator depends on a fact that
    is missing, the specification says to ignore that decorator, and that is what happens.
    """

    #: The chat messages, oldest first. Only the tail is examined, per ``scan_depth``.
    messages: Sequence[str] = ()

    #: Overrides the lorebook's own ``scan_depth`` when set.
    scan_depth: Optional[int] = None

    #: How many messages the assistant has sent. Used by ``@@activate_only_after`` and
    #: ``@@activate_only_every``. ``None`` means the count is unknowable, so those decorators
    #: are ignored.
    assistant_message_count: Optional[int] = None

    #: The index of the active greeting. The default greeting is index zero and
    #: ``alternate_greetings`` are numbered from one.
    greeting_index: Optional[int] = None

    #: The name of the active user icon asset, for ``@@is_user_icon``.
    user_icon: Optional[str] = None

    #: Whether the context window is full, for ``@@ignore_on_max_context``.
    max_context_reached: bool = False

    #: Entries that have matched earlier in this conversation, by index into the lorebook.
    #: Needed by ``@@keep_activate_after_match`` and ``@@dont_activate_after_match``.
    prior_matches: frozenset[int] = frozenset()

    #: The character's display name, for ``{{char}}``.
    char_name: str = ""

    #: The user's display name, for ``{{user}}``.
    user_name: str = ""

    #: A seed that keeps ``{{pick}}`` stable across the pieces of one prompt.
    pick_seed: Optional[str] = None

    #: A tokeniser, defaulting to :func:`approximate_tokens`.
    tokenizer: Optional[Callable[[str], int]] = None

    def token_count(self, text: str) -> int:
        counter = self.tokenizer or approximate_tokens
        return counter(text)


@dataclass
class ScanHit:
    """One entry that matched, with everything a prompt builder needs about it."""

    entry: LorebookEntry

    #: Which of the entry's keys actually matched. Empty for a constant entry.
    matched_keys: list[str] = field(default_factory=list)

    #: The content to inject: decorators removed, macros substituted.
    content: str = ""

    #: The text that was matched against, after macro substitution. Useful when a match looks
    #: wrong and the caller needs to see what the scanner actually searched.
    haystack_excerpt: str = ""

    #: Resolved decorators. Placement decorators appear here for the caller to act on.
    directives: dict[str, Any] = field(default_factory=dict)

    #: Position from the entry's ``position`` field, which is ``before_char`` or
    #: ``after_char``. ``@@position`` takes precedence when present, and is reported through
    #: ``directives`` instead.
    position: str = "before_char"

    #: Order in the lorebook, from ``insertion_order``. Lower goes earlier.
    insertion_order: int = 0

    #: How many tokens this hit contributes, by the active tokeniser.
    tokens: int = 0

    #: Why this hit was dropped, when it was. ``None`` for a hit that made it in.
    dropped_reason: Optional[str] = None

    #: The entry's index within the lorebook, offset by whatever the caller passed as
    #: ``index_offset``. Declared as a field rather than attached dynamically so that the
    #: dataclass stays introspectable.
    entry_index: int = 0

    @property
    def priority(self) -> Optional[int]:
        return self.entry.priority

    @property
    def needs_prompt_placement(self) -> bool:
        """Whether this entry asked for a specific place in a prompt."""
        return bool(self.directives.get("_needs_prompt"))


@dataclass
class ScanResult:
    """The outcome of a whole scan."""

    #: Entries that matched and survived the budget, in injection order.
    hits: list[ScanHit] = field(default_factory=list)

    #: Entries that matched but were removed to fit ``token_budget``.
    dropped: list[ScanHit] = field(default_factory=list)

    #: Entries that matched and contributed nothing because their content was empty.
    empty: list[ScanHit] = field(default_factory=list)

    #: Non-fatal problems found while scanning.
    warnings: list[str] = field(default_factory=list)

    #: How many passes the recursive scan made.
    recursion_passes: int = 0

    @property
    def total_tokens(self) -> int:
        return sum(hit.tokens for hit in self.hits)

    @property
    def matched_indices(self) -> frozenset[int]:
        """Which entries matched, for passing back in as ``prior_matches`` next time."""
        indices = {hit.entry_index for hit in self.hits}
        indices.update(hit.entry_index for hit in self.empty)
        indices.update(hit.entry_index for hit in self.dropped)
        return frozenset(indices)

    def injection_text(self) -> str:
        """The matched content joined in injection order, separated by blank lines.

        This is what a caller that only wants the text can use. A caller that needs to place
        each piece separately should read ``hits`` and consult ``needs_prompt_placement``.
        """
        return "\n\n".join(hit.content for hit in self.hits if hit.content)


def _compile_key(key: str, *, use_regex: bool, case_sensitive: bool) -> Optional[re.Pattern[str]]:
    """Compile one key into a pattern, or return ``None`` when it cannot be used.

    The specification is explicit that an invalid regular expression means the key is not a
    match rather than an error for the whole book, so a bad pattern returns ``None`` and the
    caller ignores that key.
    """
    if not key:
        return None
    flags = 0 if case_sensitive else re.IGNORECASE
    if not use_regex:
        return re.compile(re.escape(key), flags)
    if len(key) > MAX_REGEX_LENGTH:
        return None
    try:
        return re.compile(key, flags)
    except re.error:
        return None


def _matches_any(
    patterns: Iterable[re.Pattern[str]], haystack: str
) -> Optional[str]:
    """The source pattern of the first pattern that matches, or ``None``.

    Returning the pattern rather than a boolean lets the caller report which key matched,
    which is the single most useful thing to know when a lorebook behaves unexpectedly.
    """
    bounded = haystack[:MAX_REGEX_SCAN_CHARS]
    for pattern in patterns:
        if pattern.search(bounded):
            return pattern.pattern
    return None


def _build_haystack(messages: Sequence[str], depth: int) -> str:
    """Join the most recent ``depth`` messages into one searchable string.

    A depth of zero means the whole log, matching the convention that an unset or zero depth
    disables the limit rather than searching nothing.
    """
    if depth <= 0:
        selected = list(messages)
    else:
        selected = list(messages)[-depth:]
    return "\n".join(selected)


def _structural_keys_are_satisfied(
    entry: LorebookEntry,
    parsed: decorators.ParsedContent,
    haystack: str,
) -> tuple[bool, list[str]]:
    """Apply the key-matching rules to one entry.

    Returns whether the entry matches on its keys, and which keys matched.
    """
    matched: list[str] = []

    if entry.constant:
        # A constant entry is included without any key match. It still has to pass the
        # exclusion decorator, which is handled below.
        pass
    else:
        all_keys = list(entry.keys)
        for item in parsed.all("additional_keys"):
            all_keys.extend(item.values)

        patterns = [
            pattern
            for pattern in (
                _compile_key(key, use_regex=entry.use_regex, case_sensitive=entry.case_sensitive)
                for key in all_keys
            )
            if pattern is not None
        ]

        if not patterns:
            return False, []

        found = _matches_any(patterns, haystack)
        if found is None:
            return False, []
        matched.append(found)

        # `selective` entries need a second, independent match.
        if entry.selective and entry.secondary_keys:
            secondary = [
                pattern
                for pattern in (
                    _compile_key(
                        key, use_regex=entry.use_regex, case_sensitive=entry.case_sensitive
                    )
                    for key in entry.secondary_keys
                )
                if pattern is not None
            ]
            secondary_found = _matches_any(secondary, haystack)
            if secondary_found is None:
                return False, []
            matched.append(secondary_found)

    # Exclusion is checked last, because it vetoes everything above, including constants.
    # The specification says to ignore it entirely when the entry uses regular expressions.
    if not entry.use_regex:
        for item in parsed.all("exclude_keys"):
            excluded = [
                pattern
                for pattern in (
                    _compile_key(key, use_regex=False, case_sensitive=entry.case_sensitive)
                    for key in item.values
                )
                if pattern is not None
            ]
            if _matches_any(excluded, haystack) is not None:
                return False, []

    return True, matched


def _activation_decision(
    entry: LorebookEntry,
    parsed: decorators.ParsedContent,
    context: ScanContext,
    entry_index: int,
    warnings: list[str],
) -> tuple[bool, bool]:
    """Apply the activation decorators.

    Returns a pair of booleans: whether the entry may match at all, and whether it matches
    regardless of its keys. The second value exists because two of these decorators do not
    merely permit a match, they compel one. ``@@activate`` says the entry is a match in any
    case, and ``@@keep_activate_after_match`` says the same once the entry has matched before.
    For those, running the key comparison afterwards and rejecting the entry because its keys
    are absent would contradict the decorator the author wrote.
    """
    forced_on = parsed.has("activate")

    # `@@activate` outranks everything, including `@@dont_activate`, which the specification
    # says to ignore when `@@activate` is present.
    if forced_on:
        return True, True

    if parsed.has("dont_activate"):
        return False, False

    if parsed.has("dont_activate_after_match"):
        if entry_index in context.prior_matches:
            return False, False

    if parsed.has("keep_activate_after_match"):
        if entry_index in context.prior_matches:
            return True, True

    after = parsed.get("activate_only_after")
    if after is not None and context.assistant_message_count is not None:
        threshold = after.as_int()
        if threshold is not None and context.assistant_message_count < threshold:
            return False, False

    every = parsed.get("activate_only_every")
    if every is not None and context.assistant_message_count is not None:
        divisor = every.as_int()
        if divisor is not None and divisor > 0:
            if context.assistant_message_count % divisor != 0:
                return False, False

    greeting = parsed.get("is_greeting")
    if greeting is not None and context.greeting_index is not None:
        wanted = greeting.as_int()
        if wanted is not None and context.greeting_index != wanted:
            return False, False

    icon = parsed.get("is_user_icon")
    if icon is not None and context.user_icon is not None:
        if icon.value != context.user_icon:
            return False, False

    return True, False


def _entry_scan_depth(
    entry: LorebookEntry, parsed: decorators.ParsedContent, book: Lorebook, context: ScanContext
) -> int:
    """The depth to use for this entry, resolving decorator, book, and context in order."""
    override = parsed.get("scan_depth")
    if override is not None:
        value = override.as_int()
        if value is not None:
            return value
    if context.scan_depth is not None:
        return context.scan_depth
    if book.scan_depth is not None:
        return book.scan_depth
    return DEFAULT_SCAN_DEPTH


def _drop_for_budget(
    hits: list[ScanHit], budget: Optional[int], warnings: list[str]
) -> tuple[list[ScanHit], list[ScanHit]]:
    """Trim hits to fit a token budget, returning what was kept and what was dropped.

    The specification's rule has two parts. Entries marked ``@@ignore_on_max_context`` are the
    first to go when the context is full, ahead of ordinary entries regardless of their
    priority. Among ordinary entries, the lowest ``priority`` goes first, and when priority is
    absent the lowest ``insertion_order`` goes first.

    When every entry has to be dropped the result is an empty injection rather than an error,
    since an over-budget budget is a configuration problem, not a broken card.
    """
    if budget is None or budget <= 0:
        return hits, []

    total = sum(hit.tokens for hit in hits)
    if total <= budget:
        return hits, []

    def drop_key(hit: ScanHit) -> tuple[int, int, int]:
        """Sort key for which hit to drop first. Lower sorts earlier, so it is dropped sooner."""
        deferrable = 1 if hit.directives.get("ignore_on_max_context") else 0
        priority = hit.entry.priority
        # An absent priority sorts below every present one, per the specification's fallback.
        priority_rank = -1 if priority is None else priority
        return (deferrable, priority_rank, hit.insertion_order)

    ordered = sorted(hits, key=drop_key)
    dropped: list[ScanHit] = []
    kept = list(hits)
    remaining = total

    for candidate in ordered:
        if remaining <= budget:
            break
        candidate.dropped_reason = (
            "dropped to fit token_budget"
            + (
                " (marked @@ignore_on_max_context)"
                if candidate.directives.get("ignore_on_max_context")
                else f" (priority {candidate.entry.priority})"
                if candidate.entry.priority is not None
                else " (no priority set)"
            )
        )
        kept.remove(candidate)
        dropped.append(candidate)
        remaining -= candidate.tokens

    if dropped:
        warnings.append(
            f"{len(dropped)} lorebook entr{'y' if len(dropped) == 1 else 'ies'} dropped to fit "
            f"the {budget} token budget"
        )

    return kept, dropped


def scan(
    book: Lorebook,
    context: Optional[ScanContext] = None,
    *,
    index_offset: int = 0,
) -> ScanResult:
    """Scan a lorebook against a conversation and return what should be injected.

    ``index_offset`` lets a caller that has merged several lorebooks keep ``prior_matches``
    and entry indices distinct across them.
    """
    context = context or ScanContext()
    result = ScanResult()

    if len(book) == 0:
        return result

    # Parse decorators once per entry. The cleaned content is what goes into a prompt, and the
    # directives decide everything else.
    parsed_entries: list[decorators.ParsedContent] = []
    for index, entry in enumerate(book.entries):
        parsed = decorators.parse(entry.content)
        parsed_entries.append(parsed)
        for message in parsed.warnings:
            result.warnings.append(f"entry {index}: {message}")

    # A recursive scan feeds injected content back into the haystack and scans again. The
    # limit exists because a book whose entries reference each other's keys never converges.
    haystack_extra: list[str] = []
    passes = 0
    matched_this_round: set[int] = set()

    while True:
        passes += 1
        candidates = []
        for index, entry in enumerate(book.entries):
            if index in matched_this_round:
                continue

            parsed = parsed_entries[index]

            if not entry.enabled:
                continue

            if not entry.content.strip():
                # An entry with no content injects nothing. It is reported so a caller can
                # tell "matched but empty" apart from "never matched".
                continue

            allowed, forced = _activation_decision(
                entry, parsed, context, index + index_offset, result.warnings
            )
            if not allowed:
                continue

            depth = _entry_scan_depth(entry, parsed, book, context)
            depth_extra = "\n".join(haystack_extra) if haystack_extra else ""
            haystack = _build_haystack(context.messages, depth)
            if depth_extra:
                haystack = haystack + "\n" + depth_extra

            if forced:
                # The decorator compels a match, so the keys are not consulted.
                matched, keys = True, []
            else:
                matched, keys = _structural_keys_are_satisfied(entry, parsed, haystack)
            if not matched:
                continue

            candidates.append((index, entry, parsed, keys, haystack))

        if not candidates:
            break

        new_content_this_pass: list[str] = []

        for index, entry, parsed, keys, haystack in candidates:
            matched_this_round.add(index)

            rendered = cbs.render(
                parsed.cleaned,
                char_name=context.char_name,
                user_name=context.user_name,
                pick_seed=context.pick_seed,
            )
            for message in rendered.unknown:
                result.warnings.append(
                    f"entry {index}: macro {{{{{message}}}}} is not defined by this "
                    f"implementation and was left in the text"
                )

            hit = ScanHit(
                entry=entry,
                matched_keys=keys,
                content=rendered.text,
                haystack_excerpt=haystack[-400:],
                directives=decorators.summarise(parsed),
                position=entry.position,
                insertion_order=entry.insertion_order,
                tokens=context.token_count(rendered.text),
            )
            hit.entry_index = index + index_offset

            if not hit.content.strip():
                result.empty.append(hit)
                continue

            result.hits.append(hit)

            # If recursion is on, this content becomes part of the haystack for the next pass.
            if book.recursive_scanning and rendered.matchable_text:
                new_content_this_pass.append(rendered.matchable_text)

        if not book.recursive_scanning or not new_content_this_pass:
            break

        if passes >= DEFAULT_RECURSION_LIMIT:
            result.warnings.append(
                f"recursive scanning stopped after {passes} passes; the lorebook may contain "
                f"entries that reference each other"
            )
            break

        haystack_extra.extend(new_content_this_pass)

    result.recursion_passes = passes

    # Injection order. Lower `insertion_order` goes earlier, and ties keep their order in the
    # book, which is the stable result a reader can reason about.
    result.hits.sort(key=lambda hit: hit.insertion_order)

    kept, dropped = _drop_for_budget(result.hits, book.token_budget, result.warnings)
    result.hits = kept
    result.dropped = dropped

    return result
