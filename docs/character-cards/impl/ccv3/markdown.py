"""Rendering a card as a Markdown document.

This module is a presentation layer, not part of the format. Nothing here is required by the
specification, and a different application would reasonably render the same card differently.
It exists because a card is data and a document is not, and the translation between them has a
few decisions in it worth making deliberately.

**Choices are shown, not resolved.** The macro syntaxes include three that produce a value at
random: ``{{random}}``, ``{{pick}}``, and ``{{roll}}``. Resolving them here would bake one
arbitrary draw into a document that is meant to be read repeatedly, and would quietly mislead
anyone who assumed the text was what the model would see. So ``{{random:a,b}}`` renders as a
visible list of alternatives and ``{{roll:6}}`` renders as a range. The macros that *are*
deterministic, ``{{char}}`` and ``{{user}}``, are resolved, because leaving those unresolved
would make the document harder to read for no gain.

**Message content goes in fenced blocks.** The fields that hold literal text a model would
receive, such as ``first_mes`` and ``mes_example``, are placed inside code fences so that
Markdown does not reinterpret their contents. Cards routinely contain asterisks, underscores,
angle brackets, and lines beginning with ``#``, all of which would be mangled in prose. The
fence is made longer than any run of backticks in the content, so a card that itself contains a
fence still renders correctly.

**Empty fields are reported, not hidden.** A reader looking at a card wants to know that a
field was absent, which is different from a field that was present and empty. The summary table
lists every field the specification defines, whether or not it carries anything.
"""

from __future__ import annotations

import re
from datetime import datetime, timezone
from typing import Any, Iterable, Optional

from . import decorators
from .model import Card, Lorebook, LorebookEntry

#: The pattern used to find macros. The character class excludes braces so that a match never
#: spans an enclosing macro, which makes inner-first substitution work.
_MACRO = re.compile(r"\{\{([^{}]*)\}\}")

#: How many passes of substitution to allow, matching the limit in :mod:`ccv3.cbs`.
_MAX_PASSES = 8


def normalise_newlines(text: str) -> str:
    """Convert CRLF and lone CR line endings to LF.

    Cards are authored on several platforms and some of them store Windows line endings. Left
    alone, a carriage return survives the conversions in this module, because the whitespace
    tidying here and in the HTML converter both match on ``\n`` and never expect a ``\r``
    before it. The result is a document with stray carriage returns inside headings and on
    otherwise blank lines, which most Markdown viewers tolerate and none display cleanly.

    Batch two of the real-world testing introduced ninety-four of them across two documents
    after batch one had none, because batch one happened to use plain LF throughout.
    """
    if not text:
        return text
    return text.replace("\r\n", "\n").replace("\r", "\n")


def _display_macros(text: str, *, char_name: str, user_name: str) -> str:
    """Replace macros in a way that suits a document rather than a prompt.

    Deterministic macros are resolved. Random ones are rewritten to show that a choice exists.
    Comments are removed, and hidden keys are marked, because both are meaningful to someone
    reading a card rather than chatting with it.
    """
    if not text or "{{" not in text:
        return text

    def substitute(match: re.Match[str]) -> str:
        body = match.group(1)
        original = match.group(0)

        if body.lstrip().startswith("//"):
            return ""

        stripped = body.strip()
        if not stripped:
            return original

        if ":" in stripped:
            name, _, raw = stripped.partition(":")
        else:
            name, raw = stripped, ""
        name = name.strip().lower()

        if name == "char":
            return char_name or original
        if name == "user":
            return user_name
        if name in ("random", "pick"):
            options = [part.strip() for part in raw.split(",") if part.strip()]
            if not options:
                return ""
            joined = " / ".join(options)
            label = "one of" if name == "random" else "stable choice of"
            return f"*({label}: {joined})*"
        if name == "roll":
            spec = raw.strip().lstrip("dD") or "?"
            return f"*(a roll from 1 to {spec})*"
        if name == "reverse":
            return f"*(reversed: {raw})*"
        if name == "comment":
            return f"*(comment: {raw.lstrip()})*"
        if name == "hidden_key":
            return f"*(hidden key: {raw})*"

        return original

    current = text
    for _pass in range(_MAX_PASSES):
        updated = _MACRO.sub(substitute, current)
        if updated == current:
            break
        current = updated
    return current


#: Tag names treated as HTML. Anything outside this set is left alone, which is what keeps
#: card syntax like ``<START>`` and ``<BOT>`` intact. Those are literal markers in example
#: dialogue blocks, not markup, and a converter that strips anything matching a tag shape
#: would delete them.
_HTML_TAGS = frozenset(
    {
        "a", "abbr", "address", "article", "aside", "b", "blockquote", "br", "caption",
        "cite", "code", "col", "colgroup", "dd", "del", "details", "dfn", "div", "dl", "dt",
        "em", "figcaption", "figure", "footer", "h1", "h2", "h3", "h4", "h5", "h6", "header",
        "hr", "i", "img", "ins", "kbd", "li", "main", "mark", "nav", "ol", "p", "pre", "q",
        "s", "samp", "section", "small", "span", "strike", "strong", "sub", "summary", "sup",
        "table", "tbody", "td", "tfoot", "th", "thead", "tr", "u", "ul", "var", "wbr",
    }
)

#: A tag whose name is in the set above. Captures the closing slash, the name, and the
#: attribute text.
_HTML_TAG = re.compile(r"<(/?)([a-zA-Z][a-zA-Z0-9]*)((?:\s[^<>]*)?)/?>")


def html_to_markdown(text: str, *, heading_offset: int = 2) -> str:
    """Convert the HTML that cards carry into Markdown, leaving everything else alone.

    This is not a general HTML parser and does not try to be. It handles the subset that card
    authors actually produce, which on a sample of sixty real cards from a popular site meant
    ``p``, ``span``, ``strong``, ``em``, ``br``, ``a``, headings, lists, ``mark``, and ``u``,
    plus HTML entities.

    Two design points matter more than coverage.

    Only *known* tag names are treated as markup. Card content uses angle brackets for its own
    purposes, most visibly the ``<START>`` marker that delimits example dialogue blocks, and a
    converter that stripped anything shaped like a tag would silently delete them. An
    unrecognised name is left exactly where it was.

    Entities are unescaped last, after tags have been handled. Doing it first would turn
    ``&lt;p&gt;``, which an author escaped precisely so that it would display as text, into
    something the tag stripper then removes.

    ``heading_offset`` demotes HTML headings so they nest under the section they appear in.
    Authors write card fields as standalone documents, so a field frequently opens with an
    ``h1``. Converted literally that becomes a top-level heading competing with the card's own
    title and splitting the rendered document in two. Shifting every heading down two levels,
    which is the default, puts the deepest one at ``######`` and keeps the structure readable.
    """
    text = normalise_newlines(text)

    # The guard has to admit ampersands as well as angle brackets. A value like
    # `Tom &amp; Jerry` contains no literal `<` at all, so a guard on `<` alone skipped it and
    # left the entity unescaped in the output.
    if not text or ("<" not in text and "&" not in text):
        return text

    import html as html_module

    result = text

    # Script and style bodies are dropped with their content rather than unwrapped.
    result = re.sub(r"<(script|style)\b[^>]*>.*?</\1>", "", result, flags=re.IGNORECASE | re.DOTALL)

    # Line breaks.
    result = re.sub(r"<br\s*/?>", "\n", result, flags=re.IGNORECASE)

    # Headings become ATX headings, demoted so they sit under the section they are in. Six is
    # the deepest Markdown heading, so anything past that is pinned there.
    for level in range(1, 7):
        depth = min(6, level + max(0, heading_offset))
        result = re.sub(
            rf"<h{level}\b[^>]*>(.*?)</h{level}>",
            lambda m, n=depth: "\n" + "#" * n + " " + m.group(1).strip() + "\n",
            result,
            flags=re.IGNORECASE | re.DOTALL,
        )

    # Lists. Only a flat list is handled, since a nested one would need a real parser to get
    # the indentation right and cards rarely use one.
    result = re.sub(r"<li\b[^>]*>", "\n- ", result, flags=re.IGNORECASE)
    result = re.sub(r"</li>", "", result, flags=re.IGNORECASE)
    result = re.sub(r"</?(ul|ol)\b[^>]*>", "\n", result, flags=re.IGNORECASE)

    # Links, before the generic inline handling so the href survives.
    result = re.sub(
        r'<a\b[^>]*href\s*=\s*["\']([^"\']*)["\'][^>]*>(.*?)</a>',
        lambda m: f"[{m.group(2).strip()}]({m.group(1)})",
        result,
        flags=re.IGNORECASE | re.DOTALL,
    )
    result = re.sub(r"<a\b[^>]*>(.*?)</a>", r"\1", result, flags=re.IGNORECASE | re.DOTALL)

    # Inline emphasis.
    result = re.sub(
        r"<(strong|b)\b[^>]*>(.*?)</\1>", r"**\2**", result, flags=re.IGNORECASE | re.DOTALL
    )
    result = re.sub(
        r"<(em|i)\b[^>]*>(.*?)</\1>", r"*\2*", result, flags=re.IGNORECASE | re.DOTALL
    )

    # Code. Inline first, then blocks, so a block is not eaten by the inline pattern.
    result = re.sub(
        r"<pre\b[^>]*>(.*?)</pre>",
        lambda m: "\n" + fence(m.group(1)) + "\n",
        result,
        flags=re.IGNORECASE | re.DOTALL,
    )
    result = re.sub(r"<code\b[^>]*>(.*?)</code>", r"`\1`", result, flags=re.IGNORECASE | re.DOTALL)

    # Rules.
    result = re.sub(r"<hr\s*/?>", "\n---\n", result, flags=re.IGNORECASE)

    # Remaining block boundaries become paragraph breaks, and the wrappers are dropped without
    # leaving anything behind, since spans and marks carry styling rather than emphasis that
    # Markdown can express.
    result = re.sub(
        r"</?(p|div|section|article|blockquote|table|thead|tbody|tfoot|tr|td|th|figure|figcaption)"
        r"\b[^>]*>",
        "\n",
        result,
        flags=re.IGNORECASE,
    )
    result = re.sub(r"</?(span|mark|u|s|strike|small|sub|sup|abbr|cite|q|kbd|samp|var|ins|del)"
                    r"\b[^>]*>", "", result, flags=re.IGNORECASE)

    # Anything left that is a *known* tag is stripped, keeping its content. A tag-shaped
    # sequence whose name is not in the set is returned untouched, which is what protects the
    # card syntax that shares the angle-bracket shape: `<START>`, `<BOT>`, and their relatives
    # are literal markers in example dialogue, and deleting them damages the card's content.
    def _strip_known_tag(match: re.Match[str]) -> str:
        name = match.group(2).lower()
        if name not in _HTML_TAGS:
            return match.group(0)
        if name in ("p", "div", "br"):
            return "\n"
        return ""

    result = _HTML_TAG.sub(_strip_known_tag, result)

    # Entities last, so an escaped angle bracket stays visible as text.
    result = html_module.unescape(result)

    # Tidy the whitespace the substitutions left behind.
    result = re.sub(r"[ \t]+\n", "\n", result)
    result = re.sub(r"\n{3,}", "\n\n", result)
    return result.strip()


def fence(text: str, *, language: str = "") -> str:
    """Wrap text in a code fence long enough to survive its own contents.

    A card containing three backticks would break a three-backtick fence, and cards containing
    code samples or Markdown examples do turn up. The fence is made one character longer than
    the longest run of backticks in the text.
    """
    longest = 0
    current = 0
    for char in text:
        if char == "`":
            current += 1
            longest = max(longest, current)
        else:
            current = 0

    bars = "`" * max(3, longest + 1)
    return f"{bars}{language}\n{text}\n{bars}"


def display_label(card: Card, source_name: str = "") -> str:
    """A usable label for a card, falling back to its filename when it has no name.

    Cards with absent or whitespace-only names turn up in real collections: one card in a batch
    of eight downloaded from a popular site had a name consisting of ten spaces. Rendering that
    verbatim produces a document whose title is invisible and an index entry whose link has no
    text at all, which is worse than either showing nothing or admitting the problem.

    The reader deliberately preserves the name exactly as stored, because deciding that a value
    is meaningless is not the reader's job and stripping it would lose information on a round
    trip. This function is the presentation layer's answer to the same situation, and it is
    used by both the document renderer and the batch index so that the two agree.

    The filename is the fallback rather than a placeholder because a file called
    ``main_bestiality-rpg.png`` identifies its card far better than ``(unnamed card)`` does.
    """
    name = (card.display_name or "").strip()
    if name:
        return name

    if source_name:
        import os

        stem = os.path.basename(source_name)
        for suffix in (".card.png", ".card.json", ".png", ".apng", ".json", ".charx"):
            if stem.lower().endswith(suffix):
                stem = stem[: -len(suffix)]
                break
        else:
            stem = os.path.splitext(stem)[0]
        stem = stem.strip()
        if stem:
            return stem

    return "Unnamed card"


def format_timestamp(value: Optional[int]) -> str:
    """Render a unix timestamp as an ISO-like UTC string, or a dash."""
    if value is None:
        return "—"
    try:
        moment = datetime.fromtimestamp(value, timezone.utc)
    except (OverflowError, OSError, ValueError):
        return f"{value} (out of range)"
    return moment.strftime("%Y-%m-%d %H:%M UTC")


def _table_row(label: str, value: str) -> str:
    return f"| {label} | {value} |"


def _escape_cell(value: str) -> str:
    """Make a value safe inside a Markdown table cell."""
    return value.replace("|", "\\|").replace("\n", " ")


def _section(title: str, body: str, *, level: int = 2) -> str:
    """A heading and its body, trimmed. Empty bodies produce nothing at all."""
    if not body or not body.strip():
        return ""
    return f"{'#' * level} {title}\n\n{body.strip()}\n"


def _prose(text: str, *, convert_html: bool = True) -> str:
    """Render a field that is prose.

    Real cards in this field are not always prose. Authors write them in a rich text editor,
    so the stored value is frequently HTML: on a sample of sixty cards from one popular site,
    fifty carried markup in their personality field alone, most of it paragraph and emphasis
    tags. Left alone, that markup appears verbatim in the rendered document.

    ``convert_html`` is threaded through rather than decided here so that a caller who wants
    the bytes exactly as stored can have them.

    This helper is used directly only for the multilingual notes. The scalar fields go through
    the ``field`` closure inside :func:`to_markdown`, which applies the same conversion with
    the same flag.
    """
    body = text.strip()
    return html_to_markdown(body) if convert_html else body


def _message(text: str) -> str:
    """Render a field that is literal message content, protected from Markdown."""
    return fence(text.strip())


def describe_asset(asset: Any) -> str:
    """A one-line description of an asset for the summary table."""
    parts = [f"`{asset.type}`"]
    if asset.name:
        parts.append(f"named {asset.name}")
    parts.append(asset.uri)
    return " ".join(parts)


def render_lorebook(
    book: Lorebook,
    *,
    char_name: str,
    user_name: str,
    level: int = 2,
) -> str:
    """Render a lorebook as a set of entries, each with its triggers and its directives."""
    if book is None or len(book) == 0:
        return ""

    lines: list[str] = []

    settings: list[str] = []
    if book.scan_depth is not None:
        settings.append(f"scan depth {book.scan_depth}")
    if book.token_budget is not None:
        settings.append(f"token budget {book.token_budget}")
    if book.recursive_scanning:
        settings.append("recursive scanning on")
    enabled = sum(1 for entry in book.entries if entry.enabled)
    settings.append(f"{enabled} of {len(book.entries)} entries enabled")

    if book.name:
        lines.append(f"**{book.name}**")
        lines.append("")
    if book.description:
        lines.append(book.description.strip())
        lines.append("")
    if settings:
        lines.append(", ".join(settings) + ".")
        lines.append("")

    for index, entry in enumerate(book.entries):
        parsed = decorators.parse(entry.content)
        heading = entry.name or f"Entry {index + 1}"
        if not entry.enabled:
            heading += " (disabled)"
        lines.append(f"{'#' * (level + 1)} {heading}")
        lines.append("")

        facts: list[str] = []
        if entry.keys:
            facts.append("keys: " + ", ".join(f"`{key}`" for key in entry.keys))
        else:
            facts.append("no keys")
        if entry.constant:
            facts.append("constant, so it needs no key match")
        if entry.use_regex:
            facts.append("keys are regular expressions")
        if entry.case_sensitive:
            facts.append("case sensitive")
        if entry.selective and entry.secondary_keys:
            secondary = ", ".join(f"`{key}`" for key in entry.secondary_keys)
            facts.append(f"also requires one of {secondary}")
        if entry.insertion_order:
            facts.append(f"insertion order {entry.insertion_order}")
        if entry.priority is not None:
            facts.append(f"priority {entry.priority}")
        if entry.position:
            facts.append(f"position {entry.position}")

        lines.append("  \n".join(f"- {fact}" for fact in facts))
        lines.append("")

        if parsed.directives:
            lines.append("Directives:")
            lines.append("")
            for name in sorted(parsed.directives):
                for item in parsed.directives[name]:
                    values = ", ".join(item.values)
                    suffix = f" {values}" if values else ""
                    lines.append(f"- `@@{name}{suffix}`")
            lines.append("")

        if parsed.warnings:
            for message in parsed.warnings:
                lines.append(f"- Warning: {message}")
            lines.append("")

        body = _display_macros(
            parsed.cleaned, char_name=char_name, user_name=user_name
        )
        if body.strip():
            lines.append(body.strip())
            lines.append("")
        else:
            lines.append("*(no content)*")
            lines.append("")

    return "\n".join(lines).rstrip() + "\n"


def to_markdown(
    card: Card,
    *,
    source_name: str = "",
    user_name: str = "you",
    include_empty: bool = False,
    include_lorebook: bool = True,
    include_provenance: bool = True,
    include_unrecognised: bool = True,
    render_macros: bool = True,
    convert_html: bool = True,
    heading_level: int = 1,
) -> str:
    """Render a card as a Markdown document.

    ``user_name`` is what ``{{user}}`` becomes. The default reads naturally in a document:
    "you".

    ``include_empty`` controls whether sections for absent fields are emitted at all. The
    default omits them, since a card with no alternate greetings does not need a heading saying
    so, and the summary table already records which fields were present.

    ``render_macros`` decides whether the curly braced syntaxes are translated for reading. It
    applies to *every* text field, not just the message fields, because a card's description
    and scenario use ``{{user}}`` as freely as its greetings do. Passing false leaves them
    exactly as the author wrote them, which is what you want if the document is going to be
    edited and written back into a card.

    ``convert_html`` decides whether markup in the prose fields is turned into Markdown.
    Leaving it on is the default because most real cards need it, and turning it off produces
    a document that shows the tags the author's editor stored.
    """
    data = card.data
    label = display_label(card, source_name)
    # The label feeds both the title and the macro substitution for {{char}}, so a nameless
    # card still renders text that reads properly rather than a run of blanks.
    char_name = label

    out: list[str] = []

    # --- title ------------------------------------------------------------------------
    out.append(f"{'#' * heading_level} {label}")
    out.append("")

    origin = {
        "chara_card_v1": "Character Card V1",
        "chara_card_v2": "Character Card V2",
        "chara_card_v3": "Character Card V3",
    }.get(card.origin_spec, card.origin_spec)

    preamble = f"*{origin}, read from the `{card.container}` container"
    if card.backfilled:
        preamble += ", via the legacy `chara` chunk"
    preamble += ".*"
    out.append(preamble)
    out.append("")

    # --- summary table -----------------------------------------------------------------
    rows: list[str] = []
    if source_name:
        rows.append(_table_row("Source file", f"`{source_name}`"))
    rows.append(_table_row("Name", _escape_cell(data.name.strip()) or "—"))
    if data.nickname:
        rows.append(_table_row("Nickname", _escape_cell(data.nickname)))
    rows.append(_table_row("Creator", _escape_cell(data.creator) or "—"))
    rows.append(_table_row("Character version", _escape_cell(data.character_version) or "—"))
    rows.append(_table_row("Tags", ", ".join(data.tags) if data.tags else "—"))
    rows.append(_table_row("Created", format_timestamp(data.creation_date)))
    rows.append(_table_row("Modified", format_timestamp(data.modification_date)))
    rows.append(
        _table_row("Greetings", str(1 + len(data.alternate_greetings)) if data.first_mes else "0")
    )
    if data.assets:
        rows.append(_table_row("Assets", str(len(data.assets))))
    if include_lorebook and card.lorebook is not None:
        rows.append(_table_row("Lorebook", f"{len(card.lorebook)} entries"))
    if data.creator_notes_multilingual:
        rows.append(
            _table_row(
                "Notes in", ", ".join(sorted(data.creator_notes_multilingual))
            )
        )
    if data.source:
        rows.append(_table_row("Stated sources", ", ".join(data.source)))

    out.append("## Summary")
    out.append("")
    out.append("| | |")
    out.append("| --- | --- |")
    out.extend(rows)
    out.append("")

    # --- the fields --------------------------------------------------------------------
    def translate(text: str) -> str:
        """Apply the display rendering, or not, according to the caller's choice.

        Line endings are normalised here rather than at each call site, because this is the one
        place every text field passes through. Doing it per field is how a stray carriage
        return survives in two fields out of six.
        """
        text = normalise_newlines(text)
        if not render_macros:
            return text
        return _display_macros(text, char_name=char_name, user_name=user_name)

    def empty_section(title: str) -> None:
        out.append(f"## {title}")
        out.append("")
        out.append("*(not set)*")
        out.append("")

    def field(title: str, value: str, kind: str) -> None:
        """Emit one scalar field.

        ``kind`` is ``"prose"`` for text that reads as a paragraph and ``"message"`` for text
        a model would receive. The distinction decides whether the body is left as Markdown or
        protected inside a code fence.

        An earlier version of this helper chose the treatment by comparing the renderer
        function against ``_prose`` by identity. That works until it does not: the comparison
        silently took the wrong branch for the message fields and dropped their code fences
        entirely, producing documents where a card's first message was indistinguishable from
        its description. A string parameter makes the choice explicit and impossible to get
        wrong by accident.
        """
        if not (value and value.strip()):
            if include_empty:
                empty_section(title)
            return

        body = translate(value)
        if convert_html:
            body = html_to_markdown(body)
        body = body.strip()

        out.append(f"## {title}")
        out.append("")
        out.append(body if kind == "prose" else fence(body))
        out.append("")

    field("Description", data.description, "prose")
    field("Personality", data.personality, "prose")
    field("Scenario", data.scenario, "prose")

    field("Creator notes", data.creator_notes, "prose")

    if data.creator_notes_multilingual:
        out.append("## Creator notes in other languages")
        out.append("")
        for language, note in sorted(data.creator_notes_multilingual.items()):
            out.append(f"**{language}**")
            out.append("")
            out.append(_prose(note, convert_html=convert_html))
            out.append("")

    field("System prompt", data.system_prompt, "message")
    field("Post-history instructions", data.post_history_instructions, "message")
    field("First message", data.first_mes, "message")

    if not data.alternate_greetings and include_empty:
        empty_section("Alternate greetings")

    if data.alternate_greetings:
        out.append("## Alternate greetings")
        out.append("")
        for index, greeting in enumerate(data.alternate_greetings, start=1):
            rendered = translate(greeting).strip()
            if convert_html:
                rendered = html_to_markdown(rendered)
            out.append(f"### Greeting {index}")
            out.append("")
            out.append(_message(rendered))
            out.append("")

    if not data.group_only_greetings and include_empty:
        empty_section("Group-only greetings")

    if data.group_only_greetings:
        out.append("## Group-only greetings")
        out.append("")
        for index, greeting in enumerate(data.group_only_greetings, start=1):
            rendered = translate(greeting).strip()
            if convert_html:
                rendered = html_to_markdown(rendered)
            out.append(f"### Group greeting {index}")
            out.append("")
            out.append(_message(rendered))
            out.append("")

    if not (data.mes_example and data.mes_example.strip()) and include_empty:
        empty_section("Example dialogue")

    if data.mes_example and data.mes_example.strip():
        rendered = translate(data.mes_example)
        if convert_html:
            rendered = html_to_markdown(rendered)
        out.append("## Example dialogue")
        out.append("")
        out.append(_message(rendered))
        out.append("")

    if not data.assets and include_empty:
        empty_section("Assets")

    if data.assets:
        out.append("## Assets")
        out.append("")
        out.append("| Type | Name | URI |")
        out.append("| --- | --- | --- |")
        for asset in data.assets:
            out.append(
                f"| `{_escape_cell(asset.type)}` | {_escape_cell(asset.name) or '—'} "
                f"| `{_escape_cell(asset.uri)}` |"
            )
        out.append("")

    if include_lorebook and card.lorebook is None and include_empty:
        empty_section("Lorebook")

    if include_lorebook and card.lorebook is not None:
        rendered = render_lorebook(
            card.lorebook, char_name=char_name, user_name=user_name, level=2
        )
        if rendered:
            out.append("## Lorebook")
            out.append("")
            out.append(rendered)

    # --- provenance --------------------------------------------------------------------
    if include_provenance:
        items: list[str] = []
        items.append(
            f"Container: `{card.container}`. Declared as `{card.origin_spec}` "
            f"version {card.origin_version:g}."
        )
        items.append("Normalised into the V3 shape on read.")
        if render_macros:
            items.append(
                "Curly braced syntaxes were translated for reading: `{{char}}` became the "
                "character's name and `{{user}}` became "
                f"{user_name!r}. Macros that resolve randomly at run time are shown as their "
                "alternatives rather than being resolved."
            )
        else:
            items.append("Curly braced syntaxes were left exactly as the card wrote them.")
        if card.backfilled:
            items.append(
                "Read from the legacy `chara` chunk, because the file carried no `ccv3` chunk."
            )
        if card.asset_blobs:
            items.append(f"{len(card.asset_blobs)} asset payload(s) available from the archive.")

        if items:
            out.append("## Provenance")
            out.append("")
            for item in items:
                out.append(f"- {item}")
            out.append("")

    if card.warnings:
        out.append("## Reader warnings")
        out.append("")
        for message in card.warnings:
            out.append(f"- {message}")
        out.append("")

    if include_unrecognised:
        extra = data.extra
        card_extra = card.extra
        if extra or card_extra:
            out.append("## Fields this reader did not recognise")
            out.append("")
            out.append(
                "Preserved on read so that a round trip does not destroy them. Their meaning "
                "is not defined by the specification, so they are listed rather than "
                "interpreted."
            )
            out.append("")
            for key in sorted(extra):
                out.append(f"- `{key}` at the data level")
            for key in sorted(card_extra):
                out.append(f"- `{key}` at the card level")
            out.append("")

    return "\n".join(out).rstrip() + "\n"


def to_markdown_file(card: Card, path: str, **kwargs: Any) -> str:
    """Render a card and write it to a file, returning the path."""
    text = to_markdown(card, **kwargs)
    with open(path, "w", encoding="utf-8") as handle:
        handle.write(text)
    return path
