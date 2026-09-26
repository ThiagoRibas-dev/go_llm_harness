# ccv3 — a reference implementation

A reader, writer, and scanner for Character Card V1, V2, and V3 files, in pure Python with no
dependencies outside the standard library.

This exists to accompany `../IMPLEMENTATION_GUIDE.md`. Every module maps to one step of the
guide's pipeline, so the guide and the code can be read side by side.

## Running it

```
python3 -m unittest test_ccv3      # 178 tests, runs in well under a second
python3 demo.py                    # an end-to-end narrative, writes demo_out/
python3 batch_to_markdown.py <cards-dir-or-zip> <out-dir>
```

`batch_to_markdown.py` converts a whole collection of cards into Markdown documents, plus an
index and a run report. It accepts a directory or a ZIP.

## Inspecting a card

```
python3 -m ccv3 card.png
python3 -m ccv3 card.charx --scan "tell me about the sword"
python3 -m ccv3 card.png --v2      # what a V2-only reader would receive
python3 -m ccv3 card.png --json    # the normalised V3 object
```

## Using it as a library

```python
import ccv3

card = ccv3.read_file("character.png")

print(card.display_name)          # nickname if set, otherwise name
print(card.container)             # png, apng, json, or charx
print(card.origin_spec)           # what the file declared
print(card.backfilled)            # read from the legacy 'chara' chunk?
for message in card.warnings:
    print("warning:", message)

result = ccv3.scan_lorebook(
    card,
    ccv3.ScanContext(
        messages=["I think I saw the valley yesterday", "You did?"],
        char_name=card.display_name,
        user_name="Sam",
        assistant_message_count=2,
    ),
)
print(result.injection_text())
```

Writing:

```python
from ccv3 import export

png_bytes = export.write_png(card)                    # V3 plus a V2 fallback chunk
strict_png = export.write_png(card, also_write_chara=False)
v2_png = export.write_png(card, spec="chara_card_v2")  # decorators stripped
charx_bytes = export.write_charx(card, {"assets/icon/images/main.png": blob})

export.to_v3_dict(card)                # plain dictionaries, if you want to edit them
export.to_v2_dict(card)                # the V2 projection, with decorators stripped
export.to_v1_dict(card)                # the flat V1 shape
export.to_lorebook_v3(book)            # a standalone `lorebook_v3` document
```

The dictionary converters are the ones to reach for when you need to inspect or patch the output
before writing it. `to_v2_dict` strips the decorators from entry content by default, because a
V2 reader would render them as literal text; `to_v3_dict` keeps them.

Rendering a document:

```python
from ccv3 import markdown

text = markdown.to_markdown(card, source_name="card.png")
markdown.to_markdown_file(card, "out.md", user_name="Thiago")
```

The Markdown renderer translates `{{char}}` and `{{user}}`, shows random macros as their
alternatives rather than resolving them, converts the HTML that most real cards carry in their
prose fields, and protects message content inside code fences. Pass `render_macros=False` or
`convert_html=False` to turn either off.

A card's name can be empty or whitespace. `card.name` keeps whatever was stored, because
deciding a value is meaningless is not the reader's job, but a document titled with ten spaces
has no visible title. `markdown.display_label(card, source_name)` resolves that for display: the
stripped name if there is one, otherwise the source filename without its extension, otherwise
`Unnamed card`. The document title, the `{{char}}` substitution, and the batch index all use it.

## Lorebooks

A card's lorebook is read into `card.lorebook`, which is `None` when the card carried none.
`card.lorebook_or_empty()` returns an empty book instead, which is what a scanner wants.

```python
import ccv3

card = ccv3.read_file("character.png")
book = card.lorebook_or_empty()

print(len(book), book.scan_depth, book.token_budget)

result = ccv3.scan_lorebook(card, ccv3.ScanContext(
    messages=["I think I saw the valley yesterday", "You did?"],
    assistant_message_count=2,
    char_name=card.display_name,
    user_name="Sam",
))

for hit in result.hits:
    print(hit.entry_index, hit.position, hit.tokens)

print(result.injection_text())     # what this turn would add to the prompt
print(result.matched_indices)     # pass back in as `prior_matches` next turn
```

Almost all of the decorator behaviour needs a fact about the conversation, and `ScanContext` is
where those facts go. Anything left unset means the fact is unknowable, and the specification
says to ignore a decorator whose fact is missing, so a scan with an empty context is valid and
returns only the entries that need nothing.

| `ScanContext` field | What it feeds |
| --- | --- |
| `messages` | The chat, oldest first. Only the tail is examined, per `scan_depth`. |
| `scan_depth` | Overrides the book's own value. |
| `assistant_message_count` | `@@activate_only_after`, `@@activate_only_every`. |
| `greeting_index` | `@@is_greeting`. The default greeting is zero. |
| `user_icon` | `@@is_user_icon`. |
| `max_context_reached` | `@@ignore_on_max_context`. |
| `prior_matches` | `@@keep_activate_after_match`, `@@dont_activate_after_match`. |
| `char_name`, `user_name` | Substituted into `{{char}}` and `{{user}}` inside entry content. |
| `pick_seed` | Makes `{{pick}}` deterministic instead of random. |
| `tokenizer` | Replaces the character-count approximation for the token budget. |

`ScanResult` reports four groups: `hits` in injection order, `dropped` for entries removed to
fit `token_budget`, `empty` for entries that matched but had no content to contribute, and
`warnings` for non-fatal problems. Both `dropped` and `empty` are separate from `hits` because
an entry matching and an entry contributing are different events, and only the second one costs
tokens.

## Layout

| Module | Pipeline step | Responsibility |
| --- | --- | --- |
| `ccv3/containers.py` | 1, 2 | Sniff the container and extract the payload. |
| `ccv3/png.py` | 2 | PNG chunk reading and writing. |
| `ccv3/version.py` | 3 | Determine the specification version. |
| `ccv3/validate.py` | 4 | Report issues rather than raising. |
| `ccv3/normalise.py` | 5 | Lift any version into the V3 shape. |
| `ccv3/model.py` | 5 | The data structures. |
| `ccv3/decorators.py` | 9, 11 | Decorator parsing and fallback chains. |
| `ccv3/cbs.py` | 12 | The curly braced syntaxes. |
| `ccv3/lorebook.py` | 9, 10 | The scanner. |
| `ccv3/export.py` | 13 | Writing, including the V2 backfill rules. |
| `ccv3/markdown.py` | — | Rendering a card as a Markdown document. |
| `ccv3/__init__.py` | all | The public surface. |
| `ccv3/__main__.py` | — | The command line inspector. |
| `batch_to_markdown.py` | — | Batch conversion over a directory or archive. |

## The four design decisions worth knowing

**Cards normalise upward.** A V1 or V2 file produces the same object as a V3 file, with the
V3-only fields at their empty defaults. Nothing above the reader branches on the version. The
guide's section 7 explains why the alternative, normalising down to V2 with a passthrough blob,
costs more than it appears to.

**Unknown fields are preserved.** At the card level, inside `data`, and inside every lorebook
entry. Without this, a read-and-write pair deletes every field added to the specification after
the code was written.

**Validation reports rather than raising.** A malformed card is usually still a usable card.
Issues land in `card.warnings`, and `ccv3.validate.validate_card_dict` gives the structured
report if the caller wants to act on it.

**Absence is a value.** `case_sensitive`, `selective`, `position`, and `recursive_scanning` are
stored as optional, so `None` means the card did not say and `False` means the card said no. The
specification gives those two answers different meanings: a book declaring `recursive_scanning:
false` forbids recursion outright, while one that says nothing leaves the choice to the
application. An implementation that stores a plain `False` cannot tell those apart, and so cannot
write back what it read. A real card exposed this by losing `case_sensitive` from all thirty-one
of its entries and `recursive_scanning` from its book on a round trip.

Where the field is absent, this library reads it as the application's own choice: keys are
matched case-insensitively, injected content is not scanned again, and a secondary key is
required only when `selective` is set and there is a secondary key to require. Absent `use_regex`
is read as `false`, which is the only reading that does not turn literal keys into patterns.

## Known limitations

- `approximate_tokens` divides the character count by four. It is a documented approximation,
  not a tokeniser. Pass a real one through `ScanContext.tokenizer` when the budget matters.
- Regex matching has no timeout, because Python's engine does not offer one. Pattern length and
  scanned-text length are capped instead, which is the cheap guard the specification permits.
- CHARX assets are read into memory wholesale. Fine for cards, wrong for very large archives.
- Whether a card needs `@@depth` or other placement decorators is reported to the caller, not
  acted on, because this library has no prompt to place content into.
