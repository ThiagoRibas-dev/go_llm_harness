# Character cards and lorebooks

A reading of the Character Card V3 format and a working implementation of it, kept together
because the guide and the module map are meant to be read side by side. The design work for
using any of this in GoHarness lives one directory up, in
[`../CHARACTER_CARD_PERSONA_AND_MEMORY.md`](../CHARACTER_CARD_PERSONA_AND_MEMORY.md).

| Path | What it is |
| --- | --- |
| `IMPLEMENTATION_GUIDE.md` | The guide. Implementation-agnostic, four parts, eighteen sections. |
| `impl/` | A reference implementation in Python. Pure standard library, no dependencies. |
| `sources/` | The captured specification text, and a manifest for the sources that are not ours to commit. |

## The guide

`IMPLEMENTATION_GUIDE.md` explains how to read a card, not how this particular reader works. The
spine is a seven-step pipeline: identify the container, extract the payload, determine the
specification version, validate, normalise into an internal model, resolve assets, and mount the
lorebook. Its central argument, in section 7, is that a reader should normalise **upward** to the
newest shape and never downward, because a downward normalisation loses fields that the next
version adds.

Two parts of it are the reason it exists rather than a shorter note. Part Two covers the lorebook
matching algorithm, all twenty decorators, and the curly-braced syntaxes, which is the part of the
specification with the most behaviour per line. Part Four is about practice: what to reject, what
to accept with a warning, how to test a reader, and what seventy real cards taught that the
specification does not say.

## The implementation

```sh
cd impl
python3 -m unittest test_ccv3              # the test suite
python3 demo.py                            # an end-to-end narrative
python3 -m ccv3 card.png                   # inspect a card
python3 batch_to_markdown.py cards/ out/   # convert a collection
```

It reads V1, V2, and V3 cards from PNG, APNG, JSON, and CHARX containers, including the legacy
`chara` chunk that older exports use, and it writes any of those shapes back out. `impl/README.md`
documents the API.

Nothing here is on GoHarness's runtime path. The intent, described in the design document above, is
that this runs offline as a converter and the harness consumes the JSON it produces.

## Sources and licensing

`ccv3_spec.md` and `ccv3_concepts.md` are the specification text, MIT licensed, redistributed here
unmodified with attribution to the authors of
[kwaroran/character-card-spec-v3](https://github.com/kwaroran/character-card-spec-v3). Note that
`concepts.md` states in its own header that it is outdated, so it is only ever used for decorator
semantics and never as the implementation reference.

The other sources that informed the guide are **not** committed. [SillyTavern](https://github.com/SillyTavern/SillyTavern)
and [lenML/char-card-reader](https://github.com/lenML/char-card-reader) are both licensed AGPL-3.0,
and copying their source into this MIT repository would be a licensing problem. They are recorded
in [`sources/MANIFEST.md`](sources/MANIFEST.md) by URL, revision, license, size, and content hash
instead, which keeps the evidence auditable without the files.
