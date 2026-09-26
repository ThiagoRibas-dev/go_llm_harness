# Implementing Character Card Reading

A language-neutral guide to reading, interpreting, and writing Character Card files, covering
specification versions 1, 2, and 3, with V2 as the compatibility baseline.

---

## 1. What this document is, and what it is not

This is a build guide. It describes an algorithm, step by step, that takes a file off disk and
produces a usable character object. Each step says what goes in, what comes out, what the
rules are, where implementations commonly go wrong, and how to test that your version works.

It is not a transcription of the specification. Where the specification is precise, this
document quotes it and moves on. Where the specification is silent, ambiguous, or contradicted
by what working implementations do, this document says so and recommends a position.

It is written to be language-neutral. The pseudocode is deliberately plain, and the data
structures are described in terms any language can express. A companion reference
implementation in Python lives in `impl/`, with 132 tests, and this guide points at it by
name wherever a choice needs to be shown rather than described. The directory layout of that
implementation follows the steps of this guide one for one, so the mapping is direct if you
want to port it.

The single most important claim in this document is made in section 7: **normalise upward to
the newest version, never downward.** Everything about V2 compatibility follows from that
decision, and section 7 explains why the alternative, which is what the most widely deployed
reader does, costs you more than it looks like it does.

## 2. The shape of a card, in one page

A V3 card is a JSON object with three top-level keys.

```
{
  "spec":         "chara_card_v3",     // a fixed string
  "spec_version": "3.0",               // compared as a float
  "data":         { ... }              // everything else lives here
}
```

A V2 card has the same three keys with `chara_card_v2` and `2.0`. A V1 card has no wrapper at
all: the character fields sit directly at the top level, and there is no version marker
anywhere in the file.

Inside `data`, V3 is a strict superset of V2, and V2 is a strict superset of V1 in everything
but the wrapper. Every field that existed in V2 keeps its name, its meaning, and its position
under `data`. The V3 additions are:

| Field | What it carries |
| --- | --- |
| `assets` | An array of media the card ships with, each with a type, a URI, a name, and an extension. |
| `nickname` | A short name that replaces `name` in text substitution. |
| `creator_notes_multilingual` | The creator notes in several languages, keyed by two-letter language code. |
| `source` | An append-only list of identifiers or URLs recording where the card came from. The application should not let the user edit this. |
| `group_only_greetings` | Greetings shown only in group conversations. Required, and legitimately empty. |
| `creation_date` | Unix seconds in UTC, or zero for unknown. The application should not let the user edit this. |
| `modification_date` | The same, for the last edit. |

Two other V3 changes are not fields at all and are easy to miss when you are reading the
field list. The lorebook is specified in full for the first time, and the macro syntax that
was previously a matter of convention is standardised. Sections 10 and 12 cover both.

The specification also states three compatibility rules that shape everything else. An
application should ignore fields it does not recognise rather than refusing the card. It may
preserve those fields so that a later export does not destroy them. And application-specific
data belongs in the `extensions` field, with applications forbidden from inventing new
top-level keys. That third rule is what makes the first two safe, because it bounds where
foreign data can appear.

---

# Part One: Reading

## 3. Step one — identify the container

**Input:** a byte array. **Output:** a container kind, or a failure.

A card travels in one of three containers, and the container is not the same thing as the
format of the card inside it.

| Container | How to recognise it | Where the card lives |
| --- | --- | --- |
| PNG or APNG | The file begins with the eight-byte PNG signature `89 50 4E 47 0D 0A 1A 0A`. | A `tEXt` chunk whose keyword is `ccv3`, falling back to one named `chara`. |
| JSON | The first non-whitespace byte is `{`. | The file *is* the card object. |
| CHARX | The file is a ZIP archive, so it contains the local file header signature `50 4B 03 04`. | `card.json` at the root of the archive. |

**Sniff by content, not by file extension.** Cards are routinely renamed: a `.png` holding a
JSON card, a `.charx` holding a plain ZIP, and a `.json` holding an archive all appear in the
wild. The extension is a hint for the error message, never the decision.

APNG is a PNG with an `acTL` chunk. It is worth distinguishing only so that you can report the
container accurately; nothing else about reading changes.

There is a fourth container worth knowing about even if you decide not to support it. Some
tools ship cards in an unrelated archive format of their own, and a reader that dispatches on
"what kind of file is this" will meet one eventually. Detect it, and fail with a message that
names the format, rather than failing with "unrecognised".

**The trap:** a self-extracting ZIP. Some writers prepend data to a valid archive, so the ZIP
signature is not at offset zero. If you want to handle these, search for the signature within
the first megabyte. Search carefully, because a ZIP contains a local file header for every
entry it holds, so a naive search from offset one will find a *nested* header a few hundred
bytes in and slice the archive open at the wrong place. Check offset zero first and return it
directly; only search when the file does not already begin with the signature.

**How to test it:** feed the recogniser a PNG, a JSON object, a ZIP, an empty file, and a
plain text file. Assert the four container kinds and two failures. Then feed it a valid ZIP
with a stub prepended, and assert it still resolves to CHARX at the right offset rather than
producing a corrupt slice.

## 4. Step two — extract the payload

**Input:** bytes and a container kind. **Output:** a decoded JSON object, plus provenance
about where it came from.

### 4.1 From a PNG

A PNG is a signature followed by a sequence of chunks. Each chunk is a four-byte big-endian
length, a four-byte type, that many bytes of payload, and a four-byte CRC over the type and
payload together.

```
for each chunk:
    length = read_uint32_be()
    kind   = read_4_bytes()
    data   = read(length)
    crc    = read_uint32_be()
```

The card is in a `tEXt` chunk whose payload is `keyword`, a null byte, then the value. For the
card the keyword is `ccv3`, and the value is the card's JSON, encoded as UTF-8 and then
base64.

Preference order matters and the specification is explicit about it. If a file carries both a
`ccv3` chunk and a `chara` chunk, read `ccv3`.

Do not assume the `chara` chunk holds a V2 card, though. The name suggests it and older tooling
guaranteed it, but the implementation that writes most of these files stamps whatever it has
stored into both chunks, so on real files the two are sometimes byte-identical V3 payloads. Read
the version the payload declares and let the chunk name be a container detail rather than
evidence. Section 17.2 has the measurements.

Three details make the difference between a reader that works on real files and one that works
on the files you generated yourself.

**Base64 may be unpadded or wrapped.** The specification says the value is base64 but does not
require padding, and several writers emit it unpadded or broken across lines. Strip
whitespace, then add `=` padding to bring the length to a multiple of four, then decode.

**A bad CRC should not stop you.** In practice a mismatched checksum usually means the image
was edited by a tool that did not rewrite the checksums, not that the card is corrupt. The
card sits inside the chunk verbatim either way. Report it and use it.

**Decode `iTXt` and `zTXt` too, if you can spare the lines.** The specification requires
`tEXt`. Other tools write the other two text chunk types, and a reader that only understands
`tEXt` will report "no card found" for a file that plainly has one.

If no usable chunk is found, the error should list the text chunk keywords that *were* present.
"Expected `ccv3` or `chara`, found `Comment`, `Software`" is a fixable bug report. "No card
found" is not.

### 4.2 From JSON

The file is the object. Parse it and check that it is an object rather than an array or a
scalar, because everything downstream assumes a mapping.

### 4.3 From CHARX

Unzip, read `card.json` from the root, parse it. Then read every entry under `assets/` into
memory, keyed by its full path inside the archive, because the card's `embeded://` URIs refer
to those paths and you will need them at step six.

Real archives deviate in ways worth tolerating. A card nested one directory deep, written by a
careless re-zip, is common enough to be worth finding when there is exactly one candidate. No
`card.json` at all should be a clear failure that lists the entries the archive did contain.

## 5. Step three — determine the specification version

**Input:** the decoded object. **Output:** a spec string, a numeric version, and a list of
things the reader had to guess.

The specification says applications compare `spec_version` as a float. It does not say how to
identify a card's version, and that is a real gap, because a bare V1 card has no version
marker at all.

Use this order.

1. If `spec` is present and is one of the three known strings, that is the answer. Take
   `spec_version` from the field if present, and from the spec string's implied version if not.
2. If `spec` is present but unrecognised, try to pull a version out of its numbering. A string
   ending `_v3` or `_v4` identifies its major version well enough to proceed, and the reader
   should record that it guessed.
3. If `spec_version` is present and numerical, use it.
4. If neither is present but `data` is, decide from the shape of `data`. If any V3-only field
   appears, it is V3. Otherwise, if the data carries several V2 fields, it is V2.
5. If there is no `data` object and the top level carries several V1 fields, it is V1.
6. Otherwise, fail, and say what you saw.

**Accept both a number and a string for `spec_version`.** Cards in the wild carry `2.0` and
`"2.0"`, and the dominant implementation itself writes the string. Reject nothing that parses.

**Warn, do not refuse, on a newer major version.** A card declaring version 4 should be
imported on a best-effort basis with a warning, not rejected. This is the specification's
position and it is the right one, because the compatibility rules mean a newer card is
*designed* to degrade gracefully in an older reader.

**Do not treat a boolean as a version.** In some languages `true` coerces to `1`, which will
silently read as V1. Guard against it explicitly.

## 6. Step four — validate

**Input:** the decoded object and the detected version. **Output:** a list of issues, each with
a severity and a path.

Validation comes before normalisation and produces a *report*, not an exception. This is a
design position and it is worth defending, because the opposite position is tempting.

Every real implementation in this ecosystem is permissive to a degree that surprises people
reading the specification for the first time. The dominant application's own V3 validator
checks only that `spec_version` parses into a numeric range and that `data` is a non-null
object. It checks no field inside `data` at all. Its V2 validator, by contrast, requires
fourteen named fields and type-checks three of them. The asymmetry is unintentional, but the
permissiveness is not.

The reason is that a card with a malformed field is usually a usable card, and a reader that
refuses it produces a worse outcome for the user than one that imports it and says what looked
wrong. So: check everything, report everything, refuse almost nothing.

What is worth checking, roughly in order of how often it catches a real problem:

- The required fields for the declared version are present. This catches truncated exports,
  which are the most common real corruption.
- Fields that must be arrays are arrays. A `tags` field holding a comma-separated string is
  common enough that section 7 recommends tolerating it rather than rejecting it.
- `position` on a lorebook entry is `before_char` or `after_char`.
- The `assets` array is well-formed, and no more than one `icon` is named `main`.
- Asset URIs begin with a scheme the specification defines.
- `creator_notes_multilingual` keys are two-letter language codes.
- Dates are numbers, and zero means unknown rather than the epoch.

Anything that is a *warning* should be attached to the card so the user sees it. Anything
that is an *error* should still be attached, and the caller should be given the option to
turn errors into failures if that suits their application.

The reference implementation puts this in `ccv3/validate.py` and returns a `ValidationReport`
with `errors` and `warnings` accessors.

---

## 7. Step five — normalise into an internal model

**This is the central decision of the whole design.**

**Input:** the decoded object and its version. **Output:** a single internal representation,
identical in structure regardless of what the file contained.

### 7.1 Normalise upward, not downward

V3 is a superset of V2, and V2 is a superset of V1 in everything but the wrapper. So a card can
always be lifted *upward*: read whatever version the file declares, then populate the V3
fields, leaving the V3-only ones at their empty defaults for older cards.

```
if version is V1:   data = the object itself
elif version is V2: data = object["data"]
else:               data = object["data"]          // V3, same path as V2

normalised = new CardData()
copy every V2 field from data into normalised
copy every V3 field from data into normalised     // absent for old cards, so defaults stand
normalised.extra = every field of data we did not copy
```

After this runs, nothing above the reader needs to know which specification the file used. A
V2 card becomes a V3 card that happens to have no assets, no nickname, and no multilingual
notes. Every consumer reads one shape. In the reference implementation, a V1 card and a V3
card both produce a `CardData` with the same twenty-three fields, which is asserted in
`test_all_three_versions_yield_the_same_field_set`.

### 7.2 Why the alternative costs more than it looks like it does

The dominant implementation does the opposite, and understanding why is instructive because it
is a reasonable design that has a hidden price.

It normalises *downward* to V2, and it keeps the original file's JSON intact so that nothing is
actually lost. Every import path funnels through a function that re-stamps the version fields:

```
result.spec         = "chara_card_v2"
result.spec_version = "2.0"
```

and then seeds the stored record from a complete copy of the incoming object, passing it in as
a key called `json_data`, removing only that one key so the blob cannot nest inside itself on
the next save. The field-copying path reads `data.name`, `data.description`, and so on, and
never checks the version at all, because V3's `data` is a superset and the same field names
work for both.

This works. It round-trips V3 cards without damaging them, the extra fields travelling
unharmed inside the preserved blob. It even produces a V3 file on export, by promoting the
stored V2 object and adding the version fields back.

But it leaves the reader unable to answer a question about a V3 field without reaching into an
opaque blob, it requires two guard clauses purely to stop the blob nesting inside itself, and
it means every piece of code running against a card has no idea which version it is looking
at. The blob is load-bearing, and load-bearing blobs are where bugs hide.

Normalising upward has none of those properties. The cost is that a field which was absent and
a field which was present-but-empty become indistinguishable, which is why every object in the
model needs an `extra` mapping and the top level needs to keep the untouched original.

### 7.3 Preserving what you do not understand

Lifting known fields out of a dictionary and dropping the rest would silently delete every
field the specification adds after your reader was written. This is exactly the failure the
specification warns about when it asks applications to ignore unknown fields rather than
reject them.

So every object in the model carries an `extra` mapping holding everything that was not
recognised, at three levels:

| Level | What goes in `extra` |
| --- | --- |
| The card | Top-level keys other than `spec`, `spec_version`, and `data`. |
| The `data` object | Keys inside `data` that the specification does not define. |
| Each lorebook entry | Keys inside an entry that the specification does not define. |

The writer puts them all back, merging with `setdefault` semantics so that a value you computed
from the model wins over the preserved copy. That can only matter when a caller has edited the
card, and in that case the edit should win.

### 7.4 Coerce when the intent is unmistakable

Real cards have wrong types. Handle the ones where the author's intent is obvious, and record
a warning when you do.

| Situation | Recommendation |
| --- | --- |
| `tags` is a comma-separated string | Split it into an array and warn. One tag is worse than the three the author meant. |
| `enabled` is the string `"true"` | Coerce to a boolean. |
| `spec_version` is a number, not a string | Accept it silently. This is normal. |
| `creation_date` is `0` | Treat as absent. Zero means unknown by definition. |
| `creation_date` is milliseconds | Divide by a thousand. Sixteen digits is past any plausible second value. |
| `name` is an object | Read as empty and warn. Nothing about the intent is recoverable. |

### 7.5 Keep the original

Store the untouched object alongside the normalised model. When a field looks wrong, the first
question is always whether the reader mangled it or the author did, and the only way to answer
that quickly is to have both.

---

## 8. Step six — resolve assets

**Input:** the normalised card and, if the container was CHARX, the asset blobs. **Output:** a
way to get from an asset entry to usable bytes.

Leave the `uri` string exactly as it appeared. Do not normalise its case, do not resolve it at
parse time, and do not strip its prefix. Resolution needs to know which container the card came
from and whether the bytes are actually present, and both of those should be the caller's
choice rather than a side effect of reading.

There are four URI forms.

| Form | Meaning |
| --- | --- |
| `embeded://path/to/asset.png` | A path inside a CHARX archive. |
| `ccdefault:` | Use whatever default this application ships for the asset's type. |
| An HTTPS URL | Fetch it. Applications may refuse plain HTTP for security reasons. |
| A base64 data URL | Decode it. Applications may refuse formats they cannot handle. |

Two traps.

**The prefix really is spelled `embeded://`, with three e's.** The specification names the
alternative `embedded://` explicitly, in parentheses, so that implementers do not "fix" it.
Accepting the conventional spelling as well is what the dominant implementation does and is a
reasonable compatibility measure, but emit the specification's spelling and never silently
rewrite one into the other.

**Paths are case sensitive and slash separated.** Do not case-fold them, and do not run them
through a path library that will collapse separators.

The `ccdefault:` placeholder has one rule that reads oddly until you see why. If the card came
from a PNG, `ccdefault:` on an `icon` should resolve to that PNG file itself, because the image
the user downloaded *is* the portrait. But `ccdefault:` on a `user_icon` should *not* resolve to
that PNG, because the user's own portrait is not the character's.

Finally, the `user_icon` type has a consequence beyond presentation: if a card supplies one, an
application with a persona concept should disable it, since the card is asserting who the user
looks like.

---

## 9. Step seven — mount the lorebook

**Input:** the entries under `data.character_book`. **Output:** a component that can answer
"which entries match this conversation".

The lorebook is the largest thing V3 adds, and it is the first time the format specifies the
behaviour rather than just the field names. Build it as a component with a query interface, not
as a field you read and forget. Section 10 covers the matching rules.

A lorebook can also be exported on its own, outside any card, wrapped as
`{"spec": "lorebook_v3", "data": {...}}`. Give that its own read path rather than pretending
it is a card with every character field empty. A reader that receives one where it expected a
card should say "this is a lorebook, use the other function", which is a much better error than
"missing field: name".

---

# Part Two: Interpreting

## 10. The lorebook

### 10.1 Fields

The lorebook has six optional fields and two required ones.

| Field | Status | Meaning |
| --- | --- | --- |
| `name` | optional | A label. |
| `description` | optional | A longer description, for display rather than matching. |
| `scan_depth` | optional | How far back into the conversation matching looks. |
| `token_budget` | optional | The maximum this book may contribute. |
| `recursive_scanning` | optional | Whether injected content is itself scanned for further matches. |
| `extensions` | required | Application-specific data. |
| `entries` | required | The array of entries. |

Five fields are required on every entry.

| Field | Meaning |
| --- | --- |
| `keys` | The trigger words. |
| `content` | What gets injected. An empty string means inject nothing. |
| `extensions` | Application-specific data. |
| `enabled` | Whether the entry may ever match. |
| `insertion_order` | Position among injected entries. Lower is earlier. |

Two more are optional in V2 and required to be implemented in V3, which is a distinction worth
reading twice. It does not mean the field must be present in every card. It means a conforming
V3 application must honour the field when it appears.

| Field | Meaning |
| --- | --- |
| `use_regex` | Treat keys as regular expressions rather than literal strings. |
| `constant` | Include the entry whether or not its keys appear. |

The remaining optional fields are `name`, `id`, `comment`, `priority`, `case_sensitive`,
`selective` with `secondary_keys`, and `position`, which takes `before_char` or `after_char`.

The specification describes the entry fields twice and the two descriptions do not match. Its
`Lorebook` TypeScript block lists fifteen entry fields, including `extensions` and `position`.
The prose reference that follows describes thirteen and omits both. Real cards follow the
TypeScript block, so treat all fifteen as entry fields and preserve the ones you do not act on.

Here is every entry field in one list, with where the specification states it and how often a
real card used it. The card is the thirty-one entry lorebook described in section 17.3.

| Entry field | Required by the prose | In the TypeScript block | In the prose reference | Used by the real card |
| --- | --- | --- | --- | --- |
| `keys` | yes | yes | yes | 31 of 31 |
| `content` | yes | yes | yes | 31 of 31 |
| `enabled` | yes | yes | yes | 31 of 31 |
| `insertion_order` | yes | yes | yes | 31 of 31 |
| `use_regex` | yes | yes | yes | **0 of 31** |
| `secondary_keys` | yes | yes | yes | 31 of 31 |
| `selective` | yes | yes | yes | 31 of 31 |
| `case_sensitive` | yes, or undefined | yes | yes | 31 of 31 |
| `constant` | yes, or undefined | yes | yes | 31 of 31 |
| `name` | optional | yes | yes | 31 of 31 |
| `id` | optional | yes | yes | 31 of 31 |
| `comment` | optional | yes | yes | 31 of 31 |
| `priority` | optional | yes | yes | 31 of 31 |
| `extensions` | not stated | yes | **no** | 31 of 31 |
| `position` | not stated | yes | **no** | 31 of 31 |
| `probability` | no | **no** | **no** | 31 of 31 |
| `selectiveLogic` | no | **no** | **no** | 31 of 31 |

Seventeen fields in total, of which the specification's two lists agree on thirteen. The TypeScript
block adds `extensions` and `position`; the prose adds nothing that the TypeScript block lacks;
and the real card adds two more that neither mentions. An implementation that reads only the
prose will treat two intended fields as unknown, and one that reads only the required list will
miss four fields that every entry in a real card actually carries. The safe reading is to accept
all seventeen, act on the ones you have semantics for, and preserve the rest.

Note also what the last column shows about `use_regex`. It is required, it is documented in both
lists, and not one entry in a real lorebook has it. Section 17.3 records what a reader should do
about that.

Two of them carry position information and on real cards they disagree. The entry-level
`position` accepts only `before_char` or `after_char`. SillyTavern writes a second, numeric
position inside the entry's `extensions`, where `0` means before the character, `1` means after
it, and `4` means inside the chat history at a depth. A writer projecting the numeric form onto
the two-value field has nowhere to put `4` and collapses it to `after_char`. Section 17.3
measures how often that happens. Read the extension when it is present and treat the `position`
field as a lossy summary of it.

### 10.2 The matching algorithm

Apply these rules to each entry, in this order.

```
1. If the entry is not enabled, skip it. This is absolute and outranks everything else.

2. If the entry's content is empty, it contributes nothing. Skip it.

3. Apply the activation decorators (section 11.3). Two of them compel a match, and for
   those, steps 4 and 5 do not run at all.

4. If the entry is `constant`, it matches without any key.

5. Otherwise, match its keys against the scan window.
      - Widen the key set with `@@additional_keys` and veto with `@@exclude_keys`.
      - If `selective` and the entry has `secondary_keys`, a secondary match is also
        required. If `selective` is true but `secondary_keys` is empty, treat the entry as
        non-selective. The specification does not define that combination, and treating it as
        a requirement that can never be met would silently disable the entry.
      - If `use_regex`, compile the keys as patterns.

6. Sort what matched by `insertion_order`, ascending. Ties keep their order in the book.

7. Trim to `token_budget`, dropping the lowest `priority` first, or the lowest
   `insertion_order` when no priority is set.
```

Six rules within step five are easy to get wrong.

**Matching is against the conversation, not the character card.** The specification says the
chat log. A reader that matches against the character description will appear to work on
single-turn tests and fail in use.

**`scan_depth` bounds how much of the conversation is searched.** A depth of zero or an unset
depth conventionally means the whole log rather than nothing. Decide which convention you use
and document it, because implementations differ.

**Content is injected once, no matter how many times its keys match.** An entry whose trigger
appears in every message is still added a single time.

**An invalid regular expression is a non-match, not an error.** The specification is explicit.
One bad pattern must not take down the whole book. There is a second half to this rule: the
specification anticipates catastrophic backtracking, recommends the `re2` engine on the server
side, and explicitly permits an application to abandon a match it judges too expensive. In
languages without `re2`, a pattern-length cap and a cap on the scanned-text length are the
cheap guards; Python's engine, for example, has no timeout.

**`enabled: false` means the entry never matches in any case.** Not even a decorator overrides
it.

**`case_sensitive` defaults to false.** Match case-insensitively unless the entry says
otherwise.

### 10.3 Ordering and budget

After matching, sort the hits by `insertion_order` ascending. When two entries tie, the
specification leaves it to the application; keeping the order they appeared in the book is the
stable choice.

When the total exceeds `token_budget`, drop entries. The priority order is specific:

1. Entries marked `@@ignore_on_max_context` go first, ahead of ordinary entries regardless of
   their priority. This is the decorator's whole purpose.
2. Among the rest, the lowest `priority` goes first.
3. When priority is absent, the lowest `insertion_order` goes first.

An entry with no priority sorts below every entry that has one. When everything has to be
dropped, produce an empty injection rather than an error: an over-tight budget is a
configuration problem, not a broken card.

**Token counting is your problem, and it belongs to the caller.** There is no tokeniser in the
specification, and the right one depends on the model being targeted. Take a tokeniser
function as a parameter and supply a documented approximation as the default. The reference
implementation uses four characters per token, which is close enough for budgeting English
prose and wrong in predictable ways.

### 10.4 Recursive scanning

When `recursive_scanning` is on, content that was injected becomes part of what the next pass
searches. That lets one entry's text trigger another.

Cap the number of passes. A book whose entries reference each other's keys never converges, and
a reader that loops forever on a malformed card is worse than one that stops early and warns.

---

## 11. Decorators

### 11.1 The mechanism

A decorator is a line at the very start of a `content` string that begins with `@@`. It carries
a name, optionally followed by a value, and a value may be a comma-separated list. The reader
applies each rule it recognises, then removes the decorator lines and trims the surrounding
blank lines before injecting anything. Decorators are instructions to the application, not text
for the model.

```
@@decorator_with_a_value something
@@decorator_with_a_number 4
@@decorator_with_a_list a,b,c
@@decorator_without_a_value

The actual lore content, which is what gets injected.
```

Five rules keep the mechanism predictable.

**Parsing stops at the first line that is not a decorator.** A line beginning with `@@` in the
middle of an entry is ordinary text.

**An unrecognised name, or an invalid value, means that decorator is ignored and the entry is
kept.** This is deliberate: an unknown decorator must not cost the author their content.

**If the same decorator appears twice, only the first counts.** The exceptions are
`@@additional_keys`, which the specification allows to repeat, and application-defined
decorators.

**Applications may define their own.** The name should use latin characters and underscores in
snake_case.

**On backfill to V2, remove all decorators.** A V2 reader has no idea what they mean and would
inject the lines as literal text. This is the one destructive transformation the specification
asks for, and section 13 covers it.

### 11.2 Fallbacks

A fallback decorator uses three `@` characters and sits on the line immediately after the one
it stands in for. The reader walks the chain from the top and takes the first name it
*recognises*.

```
@@risu_only_decorator 4
@@@agn_only_decorator 4
@@@activate_only_after 4
```

**The word "recognises" is doing real work, and this is the subtlest trap in the whole
specification.** A reader can recognise a decorator name without being able to act on it. A
library that has no prompt cannot *act* on `@@depth`, but it must still recognise the name, or
an author's fallback chain will resolve to the wrong link and a valid alternative will be
skipped.

Keep two separate notions: the set of names you know, and the set you can act on. This guide's
reference implementation knows all twenty and can act on about half, and it reports the rest to
the caller rather than discarding them.

Implementations are asked to support chains of at least five.

### 11.3 The twenty decorators

**Activation.** These decide whether an entry is eligible at all.

| Decorator | Value | Effect |
| --- | --- | --- |
| `@@activate` | none | The entry matches in any case. It outranks every other activation decorator, including `@@dont_activate`. |
| `@@dont_activate` | none | The entry never matches, unless `@@activate` is also present, in which case this one is ignored. |
| `@@activate_only_after` | number | Cannot match until the assistant has sent that many messages. |
| `@@activate_only_every` | number | Matches only when the assistant's message count is divisible by the value. |
| `@@keep_activate_after_match` | none | Once matched, stays matched. Matches in any case thereafter. |
| `@@dont_activate_after_match` | none | Once matched, stops matching. |
| `@@is_greeting` | number | Matches only while a specific greeting is active. |
| `@@is_user_icon` | string | Matches only while the named user icon is active. |

**Placement.** These decide where content goes. They need a prompt, so a data-layer library
records them and hands them to the caller.

| Decorator | Value | Effect |
| --- | --- | --- |
| `@@depth` | number | Insert at that message depth, counting back from the newest message. |
| `@@instruct_depth` | number | The same, counted in tokens. |
| `@@reverse_depth` | number | Count forward from the oldest message. |
| `@@reverse_instruct_depth` | number | Count forward from the beginning of the text. |
| `@@position` | string | Insert relative to a part of the card: `after_desc`, `before_desc`, `personality`, or `scenario`. |
| `@@role` | `assistant`, `system`, or `user` | Send the content as a message with that role. |
| `@@disable_ui_prompt` | string | Suppress `post_history_instructions` or `system_prompt`. |

**Matching.** These adjust how keys are found.

| Decorator | Value | Effect |
| --- | --- | --- |
| `@@scan_depth` | number | Override the book's scan depth for this entry, in messages. |
| `@@instruct_scan_depth` | number | The same, in tokens. |
| `@@additional_keys` | list | Add keys that must also be present. May appear more than once. |
| `@@exclude_keys` | list | If any appears in the scanned text, the entry does not match. Ignored when `use_regex` is on. |

**Budget.** One member: `@@ignore_on_max_context`, with no value, marks an entry to be dropped
first when the context window fills.

Several decorators cancel each other. If `@@position` is present, both `@@depth` and
`@@reverse_depth` are ignored. If `@@depth` is present, `@@reverse_depth` is ignored, and the
reverse pairing holds too.

### 11.4 The `is_greeting` numbering

`@@is_greeting` numbers the default greeting, the `first_mes` field, as zero, and the elements
of `alternate_greetings` from one. Build the greeting list in that order and do not reorder it,
or every `@@is_greeting` value in the card shifts by one.

---

## 12. Curly braced syntaxes

The specification calls these Curly Braced Syntaxes. Most of the ecosystem calls them macros.
The application must substitute the nine the specification defines, may add more provided they
open with `{{` and close with `}}`, and should detect them case insensitively.

| Syntax | Result |
| --- | --- |
| `{{char}}` | The card's `nickname` if set, otherwise its `name`. |
| `{{user}}` | The client's display name for the user, or the active persona name. |
| `{{random:A,B}}` | A random choice. A comma can be escaped as `\,`. |
| `{{pick:A,B}}` | A choice that is stable for the same prompt. |
| `{{roll:N}}` | A random integer from 1 to N. `{{roll:d6}}` is equivalent. |
| `{{// A}}` | Nothing. Removed, not shown, and not matched against. |
| `{{hidden_key:A}}` | Nothing to the user, but it *is* matched against when the scan is recursive. |
| `{{comment:A}}` | Nothing when sent as a prompt, shown to the user as an inline comment. Not matched against. |
| `{{reverse:A}}` | The reversed text. |

Three implementation notes.

**Handle nesting by substitution order, not with a recursive parser.** Use a pattern whose
captured group cannot contain a brace, so an inner macro is always matched before the outer one
containing it, and repeat until nothing changes. A single greedy pass over
`{{random:{{user}},nobody}}` mangles it.

**`{{pick}}` must be deterministic for the same input.** The specification asks for a stable
choice across the pieces of one prompt. Seed the choice from the surrounding text, the macro's
own text, and which occurrence it is, hashed rather than drawn from a seeded random object, so
that the same input gives the same answer in a different process and on a different machine.
A pick that changes on every render makes a card's output non-deterministic in a way that is
very hard to debug from outside.

**Leave unknown macros exactly as written.** Another layer may understand them, and removing
them destroys information. Record them so the caller knows.

The distinction between `{{// A}}` and `{{hidden_key:A}}` is what lets an author put trigger
words in a card that the model never reads. Both are removed from the prompt; only the second
participates in a recursive scan.

---

# Part Three: Writing

## 13. Writing a card back out

**Input:** the normalised model. **Output:** bytes in one of the three specifications.

### 13.1 Which version to write

Three options, and the right answer depends on the audience.

**Write V3 and a V2 fallback into the same PNG.** The specification supports this by telling
readers to prefer `ccv3`, so one file then serves both a V3-aware reader and an old one. For
most authors this is what "export" should mean.

Note that the dominant implementation appears to do this and, on inspection, does something
slightly different: it stamps the *same* stored payload into both chunks rather than building a
genuine V2 fallback. When the stored card is V2 the result matches the description above. When
the stored card is V3, both chunks carry V3 and the second is a duplicate rather than a
fallback. If you implement this deliberately, write an actual V2 projection into the `chara`
chunk, which is what section 13.2 describes.

**Write strictly V3.** One `ccv3` chunk, nothing else. Correct when the point of the export is
to prove the card does not depend on the legacy path.

**Write strictly V2.** One `chara` chunk and no `ccv3` chunk. **Do not write a `ccv3` chunk
containing a V2 object.** A file that advertises V3 while carrying a V2 card is a lie that a
conforming reader will act on.

### 13.2 The V2 backfill rules

The specification asks for three things when backfilling to V2.

**Remove all decorators from lorebook content.** A V2 reader would treat `@@exclude_keys
finished` as ordinary text and paste it into the prompt. This is the one place the
specification asks you to destroy something. Name the function accordingly, so that a reader of
your code sees that it is not a neutral conversion.

**Optionally warn the user.** The specification supplies a sentence to prepend to
`creator_notes`, explaining that the card is V3 and is being shown reduced. Make this opt-in
rather than automatic, so that a deliberate V2 export is not decorated with a warning its
author did not want.

**Guarantee the required fields are present.** V2 requires fourteen named fields inside `data`.
Fill in empty defaults for any that are missing rather than emitting a card that fails
validation.

### 13.3 Let the V3 fields ride along

The compatibility rules mean a V2 reader should ignore fields it does not understand. So when
you write a V2 card from a V3 model, leaving the V3 fields in place is safe, and it makes the
export lossless: reading the result back recovers the nickname, the assets, and the multilingual
notes.

Offer a flag to strip them anyway. The reference implementation calls it `keep_v3_fields`, and
it defaults to true, with `false` producing a card that is exactly V2 for cases where that is
the point.

### 13.4 Rewriting a PNG in place

If you write a card into a PNG that already has one, **replace** the existing chunk rather than
adding a second. A file with two `ccv3` chunks has two candidate cards, and since readers take
the first match, the stale one wins. Rebuild the file chunk by chunk, dropping the old value
and inserting the new one before the first `IDAT`, which is where the specification requires
text chunks to go.

---

# Part Four: Practice

### 13.5 Absence and refusal are different answers

Some fields can say yes, say no, or say nothing, and the specification gives those three states
different meanings.

| Field | `true` | `false` | absent |
| --- | --- | --- | --- |
| `case_sensitive` | Keys are case sensitive. | Keys are case insensitive. | The application may choose. |
| `recursive_scanning` | Injected content may be scanned again. | Injected content must not be scanned again. | The application may decide. |
| `selective` | A secondary key is also required. | No secondary key is required. | No secondary key is required. |

For the first two, a reader must not invent a default. A model that stores a plain `False` for
both cannot distinguish an explicit refusal from silence, and an exporter built on that model
has no way to write back what it read. Store them as optional values, where the absence is
itself a value, and write the field whenever the card said anything at all, including when it
said `False`.

`position` has the same problem in a different shape. It takes one of two strings and the
specification names no default, so a reader that assumes `before_char` both misreports the card
and leaves the exporter unable to write an entry that genuinely chose `before_char`. Section
17.3 describes the real card that exposed this.

## 14. Failure handling: what to reject and what to accept

The general principle: reject structural failures, accept everything else with a report.

| Situation | Do this |
| --- | --- |
| Not a PNG, ZIP, or JSON object | Reject. Name the bytes you saw. |
| A recognised container with no card in it | Reject. List what the container did contain. |
| The payload is not valid JSON, or not a JSON object | Reject. Include a snippet and the parse position. |
| The card declares an unknown spec | Accept, infer from the version string or number, warn. |
| The card declares a newer major version | Accept, warn, ignore what you do not understand. |
| A required field is missing | Accept, warn, use the default. |
| An unrecognised field appears, at the top level or inside `data` | Accept, preserve it verbatim, do not warn. |
| A field has the wrong type but the intent is clear | Accept, coerce, warn. |
| A field has the wrong type and the intent is unrecoverable | Accept, use the default, warn. |
| A PNG chunk has a bad CRC | Accept, use it, warn. |
| A lorebook key is an invalid regular expression | Treat as a non-match. Do not warn per key; summarise. |
| A CHARX asset named in the card is missing from the archive | Accept, leave the URI unresolved, warn once. |
| Both a `ccv3` and a `chara` chunk are present | Read `ccv3`. Do not warn; this is the normal, expected case. |

## 15. Testing an implementation

The following are the tests that earn their keep, roughly in order. The reference
implementation's suite is in `impl/test_ccv3.py`, grouped by pipeline step.

**Round-trip tests.** Read a card, write it, read it again, and compare. Run this for each
container and each specification version. This single family of tests catches more real bugs
than any other, because it exercises reading and writing against each other.

**Cross-version tests, which are the V2 compatibility tests.** Read a V1, a V2, and a V3 card
that describe the same character, and assert that the resulting objects have the same field set.
Then read a V3 card, write it as V2, read *that* back, and assert that the V3-only fields are
still present. That second test is the one that proves the compatibility claim rather than
merely exercising it.

**Unknown-field preservation.** Put a field in a card that no version defines, round-trip it,
and assert it survives. Then check it survives a trip through V2 as well.

**Decorator tests.** Assert that a decorator is stripped from the injected content. Assert that
a fallback chain resolves to the first *recognised* name, including the case where a fallback
comes out ahead because the primary is unknown to you. Assert that an invalid value drops the
decorator but keeps the entry.

**Scanner tests.** For each rule in section 10.2, one test. In particular: content injected
once when keys repeat, `enabled: false` never matching, invalid regex as a non-match, budget
dropping in the specified order, and `@@activate` beating a missing key.

**Container-edge tests.** A ZIP with a prepended stub. Base64 with the padding stripped. A
truncated PNG. A duplicate chunk. Two chunks where the preferred one is corrupt.

Two of these are worth calling out because they caught real bugs while this guide was written.
The ZIP signature search initially found a *nested* entry header inside a perfectly valid
archive and sliced it open at the wrong offset. And the `@@activate` decorator initially
permitted a match without compelling one, so an entry that said "match in any case" was still
rejected for having no matching keys. Neither would have been found by a test that only used
well-formed input.

## 16. The reference implementation

`impl/` contains a working implementation in Python with no dependencies outside the standard
library. The zero-dependency constraint is deliberate: metadata extraction is the one thing
usually delegated to a library, and doing it by hand keeps the code portable and auditable.

| File | Pipeline step | What it holds |
| --- | --- | --- |
| `ccv3/containers.py` | 1 and 2 | Sniffing the container and extracting the payload. |
| `ccv3/png.py` | 2 | Chunk reading and writing, including `tEXt`, `iTXt`, and `zTXt`. |
| `ccv3/version.py` | 3 | Working out which specification a card follows. |
| `ccv3/validate.py` | 4 | The issue report. |
| `ccv3/normalise.py` | 5 | Lifting V1 and V2 into the V3 shape, preserving unknowns. |
| `ccv3/model.py` | 5 | The data structures and the reasoning behind their shape. |
| `ccv3/decorators.py` | 9 and 11 | Parsing decorators, fallback chains, and stripping. |
| `ccv3/cbs.py` | 12 | The curly braced syntaxes. |
| `ccv3/lorebook.py` | 9 and 10 | The scanner. |
| `ccv3/export.py` | 13 | Writing any of the three specifications, and the V2 backfill rules. |
| `ccv3/__init__.py` | all | `read()`, `read_lorebook()`, `scan_lorebook()`, `render_macros()`. |
| `ccv3/markdown.py` | — | A presentation layer: rendering a card as a Markdown document. |
| `ccv3/__main__.py` | — | A command line inspector. |
| `batch_to_markdown.py` | — | Converts a directory or archive of cards into documents. |
| `demo.py` | — | An end-to-end narrative with assertions. |
| `test_ccv3.py` | — | 178 tests, grouped by pipeline step. |

```
python3 -m unittest test_ccv3          # the test suite
python3 demo.py                        # the worked example
python3 -m ccv3 demo_out/aria.png      # inspect a card
python3 -m ccv3 card.charx --scan "tell me about the valley"
python3 -m ccv3 card.png --v2          # what a V2-only reader would receive
python3 batch_to_markdown.py cards/ out/   # convert a whole collection
```

The implementation is best read in pipeline order, starting with `__init__.read()`, which is
about thirty lines and calls each of the seven steps in turn.

## 17. What real cards showed

The implementation in `impl/` has been run against two collections downloaded from public
archives. The first held sixty cards, all written by one tool and exported by a large roleplay
site. The second held eight cards from a different site. Between the two, every specification
turns up at least once, which matters because a reader can be entirely correct about a version
it has never met a real file for.

Several of the results contradict what you would infer from the specification alone, so they
are recorded here rather than left in the test suite.

### 17.1 The first collection: sixty cards, all V1

**All sixty were V1.** Not one carried a `spec` field, a `spec_version`, or a `data` object.
Every field sat flat at the top level, and every card arrived in a single legacy `chara` chunk
with no `ccv3` chunk anywhere. V2 and V3 support is not optional politeness: for this
collection it is the difference between reading the cards and reading nothing. A reader that
handled only V3 would have returned zero results from these sixty files.

**Dates are not where either specification says they are.** These cards carried creation and
modification times under a nested `metadata` object, as thirteen-digit millisecond timestamps
under the keys `created` and `modified`. Neither the key names nor the location nor the unit
comes from the specification. At least one widely used application writes the same information
under `create_date` and `modify_date` as ISO 8601 strings, sometimes at the card's top level
rather than inside `data`. A reader that checks only `creation_date` in the specification's
format will show "unknown" for both conventions. Section 7.4 recommends coercing where the
intent is unmistakable; this is that recommendation meeting real data.

**Most cards contain HTML.** Fifty-two of the sixty carried markup, fifty of them in the
`personality` field alone: eight hundred and eighty paragraph tags, then spans, strong tags,
emphasis, breaks, links, headings, lists, and marks. Authors write these fields in rich text
editors, and what gets stored is the editor's HTML. A reader that treats the field as plain
prose is not wrong about the format, but any application that displays it has to decide what to
do with the markup, and Markdown output has to convert it or show the tags.

**Angle brackets are used for card syntax as well as for markup.** The same collection used
`<START>` to delimit example dialogue blocks. A converter that strips anything shaped like a tag
deletes them, and the result is a card whose dialogue markers have silently vanished. Only
names that appear in a known list of HTML tags should be treated as markup.

**None of the sixty had a lorebook.** This is worth stating because it is easy to build the
lorebook machinery first, test it against synthetic cards, and never discover that a corpus
does not exercise it at all. Lorebook support is specified in detail and barely used in
practice by cards from at least one large site, where world information lives on the platform
rather than inside the exported file.

**Every card used the same shape.** All sixty had exactly the same seven top-level keys. Real
corpora are frequently more uniform than the specification's range of possibilities suggests,
which is good news for a reader but means synthetic tests built to the specification will
exercise paths that real files never reach, and vice versa.

### 17.2 The second collection: eight cards, V2 and V3

A second collection of eight cards, from a different platform, was the mirror image of the
first. It held **three V3 cards and five V2 cards, and no V1 cards at all**, so between the two
batches every specification has now been exercised against real files rather than against cards
written for the purpose.

**The two chunks can be identical.** The three V3 cards carried a `ccv3` chunk and a `chara`
chunk holding byte-identical V3 payloads. Batch one produced V2-in-both-chunks files from the
same writer. The conclusion is that the chunk name does not determine the version, which is the
correction now recorded in sections 4.1 and 13.1.

**A card with no name.** One card's `name` field contained ten spaces. Rendered literally that
produces a document whose title is invisible and an index entry whose link carries no text. The
reader preserves the name exactly as stored, because deciding that a value is meaningless is not
its job and stripping it would lose information on a round trip. The presentation layer resolves
it by falling back to the filename, which identifies the card far better than a placeholder.

**Windows line endings.** The first batch used plain line feeds throughout. This one had
carriage returns in descriptions, in first messages, and in all eight of one card's alternate
greetings: ninety-four of them across two documents once converted. Whitespace rules in a
renderer tend to match on a newline and never expect a carriage return before it, so they
survive, ending up inside headings and on otherwise blank lines.

**No lorebooks, again.** Neither collection, sixty-eight cards in total, contained a single
lorebook entry. That is now a pattern rather than an observation about one platform, and it
argues for building the lorebook machinery last rather than first.

**A field can be present and empty, and the empty one was the field you would have expected to
matter.** All eight cards had a `personality` field, and all eight were empty strings. The prose
sat in `description`, which held between ninety-one and three thousand nine hundred characters
depending on the card. The first collection was the reverse: fifty of its sixty cards carried
HTML in `personality`. Presence therefore tells you nothing about whether a field holds
anything, and a renderer should test for emptiness before it prints a heading.

**Unrecognised fields turn up inside `data`, not only around it.** All five V2 cards carried an
`avatar` field inside their `data` object. Neither V2 nor V3 defines `avatar`, and the value
reads as a website's own database identifier that leaked into the export. The advice in section
7.3 to preserve what you do not understand therefore applies at every level of the object, not
only to the wrapper. A reader that keeps unknown top-level keys but rebuilds `data` from a fixed
field list will drop this field and any like it, quietly.

**A real card can be missing a field its own specification requires.** One of the three V3 cards
had no `extensions` object and no `group_only_greetings`, and V3 requires both. Every reader
tested against it read the card without complaint. The row in the table in section 14 that says
to accept, warn, and use the default when a required field is missing is therefore not
hypothetical: a strict reader would have rejected a card that the dominant implementation reads
without difficulty.

**The specification's own date fields can carry the wrong unit too.** Section 17.1 records
non-standard date keys holding thirteen-digit millisecond timestamps. Two of these three V3
cards did the same thing in `creation_date` and `modification_date`, which are the
specification's own fields and which it defines as second-precision Unix timestamps. So the unit
coercion described in section 7.4 has to cover the specified fields, not only the vendor keys
that stand in for them.

### 17.3 The third collection: two cards, and the first real lorebook

The third collection held two V2 cards from the same site as the second. One of them carries a
lorebook of thirty-one entries, which is the first the implementation has ever read from a real
file. The two earlier collections had none at all, sixty-eight cards between them, so this is
the point at which a chapter of the specification that had only ever met synthetic fixtures met
actual data.

The book declares `scan_depth` 50, `token_budget` 500, and `recursive_scanning` false. All
thirty-one entries are enabled, three are constants, and every one of them sets
`selective` true. Scanning it works. A message containing one entry's key injects that entry and
the three constants and stays inside the budget; a message containing a key from every entry
injects sixteen and drops six.

Five things about it are worth recording.

**The entries carry four fields the specification does not define.** All thirty-one have
`position`, `extensions`, `probability`, and `selectiveLogic`. Neither of the specification's
two entry field lists mentions the last two, and only the TypeScript block mentions the first
two. The implementation preserves them as unknown fields, which is what section 7.3 asks for,
and none of them is lost on a round trip.

**The entries are missing a field V3 requires.** Not one of the thirty-one has `use_regex`,
which V3 requires and which decides whether keys are literal strings or regular expressions.
The implementation reads an absent `use_regex` as `false`, which is the only reading that does
not turn thirty-one literal keys into patterns.

**`selective` is true on every entry, and only nine have any secondary keys.** The specification
says `selective` means a secondary match is also required, and then does not say what to do when
there is no secondary key to match. Read literally, twenty-two of the thirty-one entries could
never fire. The implementation treats an entry with `selective` set and no secondary keys as
non-selective, which is the reading that keeps them alive. Confirmed against the scanner: all
twenty-two match on their primary key alone, and all nine of the others correctly refuse until
their secondary key also appears. Section 10.2 states the rule.

**The two position fields disagree, and the extension is the accurate one.** Every entry carries
a `position` of `before_char` or `after_char`, and a numeric `position` inside its `extensions`.
Thirty of the thirty-one are `after_char` and their extension says `1`, which agrees. Fourteen
of those also carry a depth and an extension position of `4`, which in SillyTavern's scheme
means inside the chat history rather than beside the character card. The two-value field has
nowhere to put that, so the writer collapsed it. A reader trusting `position` alone would place
fourteen of thirty-one entries in the wrong part of the prompt.

**One of the three-state fields was being flattened, and the round trip showed it.** The reader
was substituting `False` for an absent `case_sensitive` and `before_char` for an absent
`position`, so the exporter could not write back what it had read: an explicit `false` and
silence produced identical output. On this card that silently dropped `case_sensitive` from all
thirty-one entries and `recursive_scanning` from the book. The model now stores those fields as
optional, the scanner treats an absent value as the application's choice, and section 13.5
states the rule. None of the sixty-eight earlier cards could have exposed this, because none of
them had a lorebook to flatten.

The second card of the pair has no lorebook, and does carry the unrecognised `avatar` field
inside its `data`, which is the third collection in a row to produce it.

## 18. Two things the specification does not tell you

**It never defines a prompt.** Several decorators name positions in one, and they name four
values while saying applications may add more. So two conforming applications can place the
same content in different parts of the prompt, and a card relying on `@@position` will not
render identically everywhere. If you are building a reader rather than an application, record
these decorators and hand them to your caller. If you are building an application, define your
own named positions and map the standard ones onto them.

**It never defines a lifecycle.** There is no initialisation hook, no registration step, and
no callback interface. A card is data. Anything that behaves like a plugin belongs in
`extensions` and in your own code, not in the card.

---

## Appendix A: What real implementations do

This appendix exists for calibration. Where the specification and the ecosystem disagree, the
ecosystem usually wins, and knowing which way it leans tells you how strictly to enforce a
rule.

Two implementations were read closely while writing this guide. A large chat application, which
is the dominant implementation and defines what a card becomes after import. And a standalone
reading library, which treats a card as data rather than as something to run.

| Question | Specification | Large application | Standalone library |
| --- | --- | --- | --- |
| Preferred PNG chunk | `ccv3`, falling back to `chara` | `ccv3`, falling back to `chara` | PNG, JPEG, and WebP |
| Spec after import | V3 stays V3 | Re-stamped as V2, with the V3 fields preserved in a passthrough blob | All three preserved, with converters |
| V3 validation | Field level | Version range and an object check only | Type level |
| V3 required fields checked | Yes | None | Yes, by type |
| Decorators | Twenty, with fallback chains | Not implemented | Not implemented |
| Assets | Read from `embeded://` inside the archive | Copied into a per-character folder on disk | Held as typed references |
| Unknown fields | Ignore, and may preserve | Preserved wholesale | Ignored by the type layer |

Three observations follow from that table, and each of them should inform your choices.

**Decorators are implemented almost nowhere.** This is the single most useful finding for
anyone relying on them. The mechanism is behaviour inside a `content` string, so it has no
representation in a type declaration, and a library built around types reads every field
correctly while ignoring the entire decorator layer. An author writing decorators for
portability should verify support per application rather than assuming it.

**The dominant implementation's validator is looser than its own documentation.** Its V3 path
checks a numeric version range and that `data` is an object, and nothing else, while its V2
path requires fourteen named fields. It also does not re-emit V3 as V3 on every path, so a
card's fields and its declared version can become independent of each other. Do not treat that
behaviour as a statement about what V3 requires.

**The dominant implementation keeps a second, richer position scheme outside the
specification.** Its lorebook entries carry a numeric `position` inside their `extensions`
alongside the two-value `position` field that V3 describes. In the real card measured in section
17.3 the numeric values used were `0` for before the character card, `1` for after it, and `4`
for injected into the chat history at a depth, with the depth written alongside. The mapping is
not documented in the card specification at all, because it belongs to that application rather
than to the format.

Two consequences are worth carrying into an implementation. The two-value field cannot express
the numeric `4`, so when the application writes a card it projects the numeric form down and the
information is lost; fourteen of the thirty-one entries in that card are in exactly that state,
declared `after_char` while their extension asks for a position inside the conversation. And the
numeric form is written for every entry regardless of whether it applies, so a present depth does
not mean the entry is at depth. Sixteen of the thirty-one carry a depth value that means nothing
because their numeric position is `1`. Read the numeric position first when it is there, and read
the depth only when the numeric position says it is meaningful.

**Widening is more common than narrowing.** Implementations accept `embedded://` alongside the
specification's `embeded://`, accept the widely used `expression` sprite type alongside
`emotion`, and locate ZIPs that have been prepended with data. Where the ecosystem has settled
on a convention the specification omits, implementations follow the ecosystem. Accepting the
supersets costs little and is what makes a reader work on real files.

## Appendix B: Specification errata

Worth knowing before you transcribe anything.

**The `{{pick}}` example contains a doubled colon.** Its prose gives `{{pick:A,B,C}}`, and its
example writes `{{pick::Hello,Hi,Hey}}`. The prose form is the consistent one.

**A `{{/// A}}` form does not exist.** It circulates in discussion of V3 macros, but the
specification defines only `{{// A}}` for a fully stripped comment and `{{hidden_key:A}}` for
one that still participates in a recursive scan.

**The `ext` field is described as "a `uri` ext"** in the asset prose, which reads as a
copy-paste error from the adjacent `uri` field. The type declarations use `ext`.

**Several passages carry typos** that do not change the meaning but defeat text search. The
CHARX path description says "sepearated", the asset directory list says "programing", and the
opening line reads "shorted as CCv3".

**The entry field list is given twice and the two versions disagree.** The `Lorebook`
TypeScript block lists fifteen entry fields, including `extensions` and `position`. The prose
reference that follows lists thirteen and omits both. Nothing reconciles them. The TypeScript
block matches what real files contain, as section 17.3 measures.

**The `@@dont_activate` wording is a double negative worth reading twice.** An entry with that
decorator should not be considered a match in any case *unless* `@@activate` is present, in
which case `@@dont_activate` is ignored.

## Appendix C: Field reference

The fields a card's `data` object carries, with the version each arrived in.

| Field | Version | Type | Notes |
| --- | --- | --- | --- |
| `name` | V1 | string | Required. |
| `description` | V1 | string | Required. Where the prose usually lives. |
| `personality` | V1 | string | Required. Frequently present but empty in practice. |
| `scenario` | V1 | string | Required. |
| `first_mes` | V1 | string | Required. The default greeting, and greeting index zero. |
| `mes_example` | V1 | string | Required. |
| `creator_notes` | V2 | string | Required in V2 onward. The backfill notice goes here. |
| `system_prompt` | V2 | string | Required in V2 onward. |
| `post_history_instructions` | V2 | string | Required in V2 onward. |
| `alternate_greetings` | V2 | array of strings | Required in V2 onward. Numbered from one by `@@is_greeting`. |
| `tags` | V2 | array of strings | Required in V2 onward. |
| `creator` | V2 | string | Required in V2 onward. |
| `character_version` | V2 | string | Required in V2 onward. |
| `extensions` | V2 | object | Required, though real V3 cards omit it. The only place application-specific data belongs. |
| `character_book` | V2 | object | Optional. The lorebook. Fully specified only in V3. |
| `assets` | V3 | array of objects | Optional. Types are `icon`, `background`, `emotion`, `user_icon`, `other`, or `x_`-prefixed. |
| `nickname` | V3 | string | Optional. Replaces `name` in `{{char}}`. |
| `creator_notes_multilingual` | V3 | object | Optional. Keyed by ISO 639-1 code. |
| `source` | V3 | array of strings | Optional. Append-only; the application should not let the user edit it. |
| `group_only_greetings` | V3 | array of strings | Required in V3. Legitimately empty. |
| `creation_date` | V3 | integer | Optional. Unix seconds UTC; zero means unknown. Occasionally written in milliseconds. |
| `modification_date` | V3 | integer | Optional. The same. |
