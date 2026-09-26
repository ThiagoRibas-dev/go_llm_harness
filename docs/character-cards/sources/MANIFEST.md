# Captured sources

These are the reading notes behind `../IMPLEMENTATION_GUIDE.md`. They are recorded as a
manifest rather than as files, because most of them are not ours to redistribute.

The specification text itself is MIT licensed and is committed alongside this file in
`ccv3_spec.md` and `ccv3_concepts.md`, with attribution to its authors. Everything else in
the table below is referenced by URL, revision, and content hash so the evidence stays
auditable without copying the files. SillyTavern and `char-card-reader` are both licensed
AGPL-3.0, and copying their source into this MIT repository would be a licensing problem
rather than a judgement call.

The hashes are of the copies that were actually read while writing the guide. A hash that
no longer matches upstream means the file changed after the reading, not that the reading
was wrong.

| File | Upstream | Path | License | Revision | Size | SHA-256 |
| --- | --- | --- | --- | --- | --- | --- |
| `blog_tavernsprite.txt` | tavernsprite.com | `blog/sillytavern-character-card-v2-vs-v3`, 2026-09-04 | all rights reserved | n/a | 15,251 | `4ec550385815ae25…` |
| `ccr_CharacterBook.ts` | lenML/char-card-reader | `src/CharacterBook.ts` | AGPL-3.0 | main | 3,246 | `5fd451692c4eadf2…` |
| `ccr_README.md` | lenML/char-card-reader | `README.md` | AGPL-3.0 | main | 4,183 | `b0ca5228019edfe6…` |
| `ccr_spec_v3.ts` | lenML/char-card-reader | `src/spec_types/spec_v3.ts` | AGPL-3.0 | main | 1,869 | `c3f0c5d12b56c882…` |
| `ccr_utils.ts` | lenML/char-card-reader | `src/utils.ts` | AGPL-3.0 | main | 4,253 | `f6f2734896910b48…` |
| `ccv3_concepts.md` | kwaroran/character-card-spec-v3 | `concepts.md` | MIT | main | 9,851 | `1eee6af98f4ec4c8…` |
| `ccv3_spec.md` | kwaroran/character-card-spec-v3 | `SPEC_V3.md` | MIT | main | 48,247 | `3c472a16eeda5d01…` |
| `st_byaf.js` | SillyTavern/SillyTavern | `src/byaf.js` | AGPL-3.0 | release | 18,547 | `dc056d1b7f9d616e…` |
| `st_characters.js` | SillyTavern/SillyTavern | `src/endpoints/characters.js` | AGPL-3.0 | release | 68,637 | `3235555529e9ff17…` |
| `st_src_character-card-parser.js` | SillyTavern/SillyTavern | `src/character-card-parser.js` | AGPL-3.0 | release | 3,284 | `b74541cd54bb3fe3…` |
| `st_src_charx.js` | SillyTavern/SillyTavern | `src/charx.js` | AGPL-3.0 | release | 14,377 | `92e0045ff6969105…` |
| `st_src_png_encode.js` | SillyTavern/SillyTavern | `src/png/encode.js` | AGPL-3.0 | release | 1,808 | `bde45d94ca4750fb…` |
| `st_src_validator_TavernCardValidator.js` | SillyTavern/SillyTavern | `src/validator/TavernCardValidator.js` | AGPL-3.0 | release | 4,561 | `a0eba17805b004f5…` |

## What each file was read for

**`blog_tavernsprite.txt`** — Ecosystem posture in 2026: V3 adoption, CHARX support, and where sprites come from.

**`ccr_CharacterBook.ts`** — A second implementation of lorebook scanning, with no decorator handling.

**`ccr_README.md`** — What the library claims to support, including JPEG and WebP metadata.

**`ccr_spec_v3.ts`** — A second, independent type declaration for V3.

**`ccr_utils.ts`** — Version detection and type coercion helpers.

**`ccv3_concepts.md`** — Decorator semantics only. Its own header says it is outdated, so it is never used as the implementation reference.

**`ccv3_spec.md`** — The normative text. Every MUST and SHOULD in the guide traces to this file.

**`st_byaf.js`** — The backup-file format that wraps a card in a sidecar.

**`st_characters.js`** — The import and normalisation path, including convertToV2 and readFromV2.

**`st_src_character-card-parser.js`** — PNG chunk reading and writing; the write path emits both chunks.

**`st_src_charx.js`** — CHARX archive handling and the widened asset URI prefixes.

**`st_src_png_encode.js`** — How the image bytes are rebuilt around the text chunk.

**`st_src_validator_TavernCardValidator.js`** — The V2 and V3 validation paths, which differ in strictness.

