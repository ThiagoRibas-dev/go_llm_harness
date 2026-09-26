# 📥 Knowledge Ingress & Retrieval System — Execution Plan

> **Status:** Execution plan
> **System:** Knowledge Ingress & Retrieval System
> **Primary roadmap rows:** `11.1`, `11.4`, `11.16`, `11.19`, `12.5`, `12.6`, `12.19`, `12.28.12`, `12.30.1`, `12.30.2`, `12.30.3`, `12.30.4`, `12.30.5`

This document defines the program of work for how knowledge gets into GoHarness, how it is stored and addressed, how it becomes visible to the model, and how it can be found again later.

The belief that shapes the design is this:

> Every source of knowledge is the same kind of thing. A web page, a file the user referenced, a pasted screenshot, an uploaded document, a large tool result that was spilled to disk, and a diagnostic from a language server are all evidence, and all of them need the same three things: provenance, an address, and an explicit budget for what reaches the model.

The surfaces that bring knowledge in are genuinely different from each other. The storage underneath them should not be.

The character-card series in phases J to N extends that belief to a source this plan did not originally consider. A character card is a document a user wrote or downloaded, and a lorebook is a set of entries with keys, positions, and a budget. Both are evidence by this plan's definition, and the second one arrives with an injection algorithm already specified by a public format. The design that puts them here is in [`CHARACTER_CARD_PERSONA_AND_MEMORY.md`](./CHARACTER_CARD_PERSONA_AND_MEMORY.md).

## Row coverage map

For each roadmap row this plan claims, here is the phase that owns it and how complete that coverage actually is.

| Roadmap row | Phase | Coverage |
|---|---|---|
| `11.1` Web search and fetch | Phase H | **core.** The tools, the fetch policy, and evidence staging are all specified. The choice of search backend is left as an open decision inside the phase. |
| `11.4` `@file` mentions in the composer | Phase E | **core** for the resolver and the typeahead contract. The dropdown interface itself belongs to Shell & Interaction. |
| `11.16` Walking parent directories for context, with overrides | Phase F | **core.** Walks up to a project-root marker, reads from the root down to the working directory, honours an override file, and respects a byte budget. |
| `11.19` Pasting or dragging images into the composer | Phase G | **core** for getting the image in and storing it. Whether the model can actually see it depends on the provider, and the plan states that honestly instead of hiding it. |
| `12.5` The LSP seam | Phase I | **core**, but staged. The seam and one read-only capability come first. This is the largest item in the system. |
| `12.6` Spilling large tool output to disk | Phase C | **core.** This phase closes out the partial status by making spilled output searchable and referenceable. |
| `12.19` Content-addressed attachments | Phase B | **core.** This generalizes the addressing scheme that `src/spill.go` already uses. |
| `12.28.12` Deliverables, references, and message feedback | Phase A for the record, Phase I for the projection | **partial.** This plan owns the structured record. Rendering belongs to Shell & Interaction, and feedback routing is operator-side under row `12.23`. |
| `12.30.1` Card repository and per-session imports | Phase J | **core.** The repository layout, the offline converter seam, the manifest, the imported file retained in `source/`, and the per-session import list. |
| `12.30.2` Character cards as a persona source | Phase K | **core.** The rendering, the block registration, the trust value, and the choice of one persona per session. The block model itself belongs to row `12.30.8` in the Runtime, Config & Operator Control Plane. |
| `12.30.3` Cards and lorebooks as evidence kinds | Phase L | **core.** Two values added to the `kind` enumeration, the derivation lineage, and registration on import. This phase is blocked until Phase A lands. |
| `12.30.4` Lorebook as a retrieval and injection layer | Phase M | **core.** The matcher, the scan window, the budget and its drop order, recursive scanning, and the diagnostics that make an injection explainable. |
| `12.30.5` Agent-maintained memory, files plus an index | Phase N | **core.** The four memory tools, the logging, the configuration switch for creating a book, and the two storage shapes. |

---

# 1. System scope

This system owns evidence records and their provenance, content addressing for anything that gets ingested (row `12.19`), the catalog that holds evidence and the surface that searches it, everything that brings knowledge in from the composer such as file mentions (row `11.4`), images (row `11.19`), and attachments, the discovery of instructions and context across directories (row `11.16`), network ingress through web search and fetch (row `11.1`), code intelligence through the LSP seam (row `12.5`), the integration of spilled tool output into the same substrate (row `12.6`), and the structured record behind deliverables and references (row `12.28.12`).

It owns the character-card series as well: the repository that holds converted cards, personas, and lorebooks (`12.30.1`), the persona as one prompt block (`12.30.2`), cards and lorebooks as evidence kinds (`12.30.3`), the lorebook as a retrieval and injection layer (`12.30.4`), and the read and write memory surface the agent curates (`12.30.5`). The ordering and enablement of the prompt blocks those items feed into belongs to the Runtime, Config & Operator Control Plane under row `12.30.8`, which is a separate system and a separate plan.

It does not own the shell surfaces that render evidence, which belong to Shell & Interaction under row `12.29.9`. It does not own the mechanics of approval and egress policy, which belong to the Policy & Hooks Engine under rows `12.16` and `14.2`. It does not own the long-term event log or the session query projections, which belong to Session Event & Memory under rows `12.2` and `12.18`. It does not own scheduling slow fetches as jobs, which belongs to Task & Background Work under row `11.7`.

One boundary matters more than the rest, and it is worth stating on its own:

> This system decides what knowledge is available and how it is addressed. It does not decide what the model is allowed to see. Injection is always explicit, always budgeted, and always labelled with where the content came from.

---

# 2. Where the design comes from

## 2.1 What the existing GoHarness source already provides

### `src/bm25.go` — the lexical engine that already exists

The engine is small and self-contained. `BM25Document` holds a path, a map of term frequencies, and a document length. `BM25Engine` holds the scoring parameters, the documents, the document frequencies, the total count, and the average length. `NewBM25Engine` sets `K1` to 1.5 and `B` to 0.75. `AddDocument` tokenizes and indexes. `Search` returns ranked results. `IndexDirectory` walks a directory and indexes what it finds.

This is the retrieval backbone, and it already has no external dependencies, which is worth protecting. Two properties matter for this plan.

First, `Search` returns a `SearchResult` containing only a path and a score. There is no provenance, no offset, and no snippet. Every caller then re-reads the file and truncates it by raw bytes, which is what `executeBM25Search` does when it takes the first four hundred characters. Evidence-aware retrieval needs structured hits instead.

Second, `IndexDirectory` walks and indexes in a single call, with no incremental index, no cache, and no check of modification times. Section 7.4 explains why that becomes expensive as a workspace grows, since today the entire workspace is re-walked on every search.

### `src/agent.go` — search and instruction loading

`executeBM25Search` builds a brand-new engine on every call. When the scope is `session`, it indexes that session's turn folder while skipping `backups` and the `compacted_summary_up_to_turn_` folders. Otherwise it indexes the workspace, or only the directories named in `TargetScanDirs` when that setting is populated, and it always also indexes the session's uploads folder. It returns ranked paths along with four hundred characters of excerpt.

That last behaviour is a clue worth noticing. Uploads and workspace files are already treated as one corpus by the search tool. The substrate simply does not say so explicitly yet, and this plan makes it explicit.

`LoadLocalInstructions` is the current implementation of row `11.16`, and it is deliberately shallow. It looks for four filenames: `AGENTS.md`, `SKILLS.md`, `INSTRUCTIONS.md`, and `CLAUDE.md`. For each one it checks the workspace root, and if that fails it checks next to the binary.

The consequences are worth spelling out. Only the workspace root and the global binary directory are ever checked, so a project nested three levels inside a larger repository never inherits the instructions of its parents. There is no upward walk, no override file, and no size limit. What was discovered is written into the session metadata through `updateSessionPinnedFiles` and shown to the interface through `/api/sessions/pinned`, which is a good pattern worth keeping.

### `src/spill.go` — content addressing already exists here

This file is the most valuable thing the project already has for row `12.19`. The threshold for spilling is 20 KB, set by `toolSpillThresholdChars`. The function `hashSpillID` returns the sha256 of the content as a hex string. `saveSpill` writes the payload to `.goharness/sessions/<session>/spill/<id>.txt` and the metadata to a matching `.json` file. `readSpill` lets the model page through the content by offset, with an optional search term, and `loadSpillMeta` reads the sidecar back.

The important point is that GoHarness already addresses large tool output by the hash of its content. It already stores data and metadata side by side. And it already gives the model a way to read the payload in pages, so a huge result never enters the context window uninvited.

Content addressing is therefore not a new idea for this project. Row `12.19` is the generalization of a pattern that is already in use, extended from spilled tool output to everything that gets ingested.

### `src/web.go` — uploads, which are still addressed by filename

The upload handler writes to `.goharness/sessions/<id>/uploads/`, and it builds the destination with `filepath.Join(uploadsDir, handler.Filename)`. Before writing, it checks whether the cleaned destination path contains `.goharness` and, if so, whether it also contains the session's uploads folder, and rejects the upload if not.

Four consequences follow, and this plan has to deal with all of them.

The filename is the identity, so uploading `report.pdf` a second time silently replaces the first. Two sessions referring to what the user thinks is the same file hold two independent copies. The traversal guard is a substring check on a cleaned path, which is the kind of check that works until it does not. And the size limit is a flat 10 MB, with no distinction between something too large to show the model and something too large to store at all. There is no hash, no deduplication, and no provenance beyond the folder the file landed in.

### `src/artifacts.go` — deliverables are inferred by matching regular expressions against prose

The file defines three patterns and one function. `writeHostPathRE` looks for a line beginning "Successfully wrote file to host disk at", `writeDockerPathRE` looks for the Docker equivalent, and `patchPathRE` looks for "Successfully patched file". `inferArtifactsFromToolResult` runs them over a tool result and collects the paths it finds.

The consequence is that row `12.28.12` currently depends on parsing English sentences out of tool output. It works, and it is fragile in a specific way: if any tool message is reworded, the deliverables list silently becomes empty. The structured fix is for tools to report the paths they produced as data. There is already a channel for that in `tool_message_meta.go`, and the evidence catalog should become the source of truth.

### The composer, where these pieces do not exist yet

`src/web/js/composer.js` has no `@` trigger, no paste handler, and no drop handler. Rows `11.4` and `11.19` are entirely new work on the client side. What does exist is the upload endpoint, and the workspace tree built by `GenerateWorkspaceEntries` in `src/workspace_entries.go`, which a file picker can be built from.

---

## 2.2 What we learned by reading reference implementations

For this plan we read the source of two projects: OpenAI's Codex CLI, in the `codex-rs` tree, and Aider. Both were read for design lessons rather than for code to copy.

### A. Codex CLI: hierarchical project documentation, in `core/src/agents_md.rs`

The module documentation states the algorithm precisely, and it is worth following closely because it solves the problem row `11.16` describes.

First, determine the project root by walking upwards from the working directory until a configured `project_root_markers` entry is found. When that setting is unset, a default marker list is used. An empty marker list disables parent traversal entirely, which is a useful escape hatch.

Second, collect every `AGENTS.md` found from the project root down to the working directory, inclusive, and concatenate them in that order.

Third, do not walk past the project root.

Several supporting details are worth copying as well. There is a default filename, `AGENTS.md`, and a preferred local override, `AGENTS.override.md`. User-level and project-level documents are joined with an explicit separator, `--- project-doc ---`, which keeps the assembled block greppable and testable. There is a byte budget called `project_doc_max_bytes`, and content that exceeds it is truncated with a warning that says so, rather than being silently cut. A fallback filename list is configurable. And ancestor probes run concurrently but bounded, with a limit of 256, rather than fanning out without limit.

The lesson for GoHarness is that row `11.16` should port these semantics faithfully: walk up to a marker, order from root to working directory, honour an override file, respect a budget, and never walk past the root. The existing list of four filenames becomes the equivalent of the fallback list. What it should not be is an ad hoc patch that also happens to check the parent folder.

### B. Codex CLI: `view_image`, in `core/src/tools/handlers/view_image_spec.rs`

The tool takes a local path to an image file and returns an image as a data URL in an `image_url` field, along with the `detail` value it actually used. It offers a detail setting with two values, `high` for the default resized rendering and `original` to preserve exact resolution. Its description explains that it is for images already available on disk when visual inspection is needed.

Three lessons follow.

Images are addressed by their path on the filesystem and delivered to the model as a rendering. The file on disk stays the source of truth.

There is an explicit control over image detail rather than an implicit resize, which means image tokens are treated as a budget in the same way text is.

And inspecting an image is a tool the model chooses to call, not something that happens automatically because an image appeared in the conversation. The model decides when vision is worth the tokens.

### C. Codex CLI: web search as typed actions, in `core/src/web_search.rs`

The interesting part of this file is the action type, `WebSearchAction`. It enumerates four cases: `Search`, which takes one query or several; `OpenPage`, which takes a URL; `FindInPage`, which takes a pattern and a URL; and `Other`. The rendering function converts each of those into a short human-readable string, such as a pattern and the URL it was found in.

The lesson is that search, opening, and finding are the three primitives, and that every representation in the interface is derived from a typed action record rather than from the raw text of a page. Results are attributed rather than dumped.

For GoHarness this means network knowledge should be modelled as typed actions over content-addressed snapshots of pages. The statement "opened this URL and found this" should remain reconstructible long after the page has changed or disappeared.

### D. Codex CLI: fuzzy file search, in `app-server/src/fuzzy_file_search.rs`

Matches carry a `score`, the character `indices` that matched, and a `match_type`. Results are sorted by score descending and then by path ascending, so ties are broken deterministically. A `MATCH_LIMIT` bounds the result set.

There is also an incremental session protocol: `start_fuzzy_file_search_session`, then `update_query`, then snapshots delivered through `on_update` and a final `on_complete`. Snapshots are ignored when they belong to a query the client has already moved past, which prevents a slow response for an earlier query from overwriting the results of a later one.

The lesson for the `@` trigger is that a typeahead needs a stream with a stale-query guard rather than a request per keystroke, and that highlighting needs character indices. The cheap half of that is the dropdown. The expensive half is what happens when the user commits a mention, which is what the staging model in section 5.3 exists to handle.

### E. Aider: ranked structure under a budget, in `aider/repomap.py`

The repository map is a PageRank computation over a graph of definitions and references, with personalization derived from the files and identifiers already in play in the conversation. The personalization value starts at `100 / len(fnames)` and is boosted when a path component matches an identifier that has been mentioned.

The budget comes from the model's context window, scaled by `map_mul_no_files` when the conversation is empty, with a padding reserve of 4096 tokens.

Tags are cached using modification-time keys, and the refresh mode `auto` re-indexes only what changed.

Its `render_tree` emits only the lines of interest plus surrounding context, cached by relative filename, that set of lines, and the modification time. It sends structure, not whole files.

And it degrades gracefully. On a `RecursionError` the map is disabled with a message that the repository may be too large, rather than hanging.

Four lessons apply to GoHarness. Ranking is the right way to bundle evidence under a budget, and relevance should be personalized by what is already in play rather than being a flat top-N. Caching by modification time avoids re-tokenizing unchanged files, which is exactly the problem described in section 2.1. Sending structure and lines of interest is better than sending file bodies when the goal is orientation. And a retrieval feature should degrade visibly rather than hang.

---

### F. Character Card V3: a public specification, a reference implementation, and three real corpora

This plan's later phases rest on work that already exists in this repository, which is unusual enough
to state plainly. [`docs/character-cards/`](./character-cards/) holds an implementation-agnostic
guide to reading card files, a working reference implementation with 178 tests, and the
specification text with attribution. The guide was written from the specification and from reading
two implementations at source level, and it was then corrected against three collections of real
cards, sixty, eight, and two of them, downloaded from the sites people actually get cards from.

Three findings from that work shaped these phases.

**The parsing is hard and the runtime is easy.** Card files arrive as PNG text chunks, as bare JSON,
and inside a ZIP archive called CHARX, across three versions of the format, and real cards deviate
from the specification in predictable ways. All of that work happens once, at import, and can run
offline. What the harness needs at runtime is a reader for two JSON shapes, which is a small amount
of Go rather than a runtime dependency on another language.

**The lorebook format is a complete answer to a question this plan had left open.** The plan says
injection always respects a budget, and it does not say what happens when the budget is exceeded.
The format specifies a drop order: entries that opt out when the window is full go first, then the
lowest `priority`, then the lowest `insertion_order`. It also specifies positions relative to the
character, depths counted back from the newest message, and recursion that lets one entry's content
trigger another. Phase M adopts the format's answers rather than inventing its own.

**Metadata is where the specifications disagree, so the reader follows the ecosystem.** Three
collections produced examples of creation timestamps in two different places, in two different
units, and under three different key names, and all of them outside the specification. The reader
accepts all of them and reports what it saw, which is the same posture this plan takes toward
provider behaviour elsewhere.

# 3. Where GoHarness stands today

## 3.1 What already exists

| Capability | Where it lives | State |
|---|---|---|
| Lexical ranking with BM25 | `src/bm25.go` | shipped |
| Search over workspace and session for the agent | the `bm25_search` tool and `executeBM25Search` | shipped |
| Content-addressed spilling of large tool output, with paged reads | `src/spill.go` and `read_spill` | shipped |
| Session uploads with a traversal guard and a 10 MB limit | `POST /api/upload` in `src/web.go` | shipped |
| Instruction discovery at the workspace root and the binary directory, pinned per session | `LoadLocalInstructions` and `/api/sessions/pinned` | shipped |
| A workspace tree for interface pickers | `src/workspace_entries.go` | shipped |
| Artifact paths shown to the interface | `inferArtifactsFromToolResult` | shipped, but regex-based |
| Ingesting tool schemas from MCP servers | `src/mcp.go` | shipped |

## 3.2 What is missing

There is no evidence record. Nothing in the codebase represents the idea that this knowledge came from that place at that time and can be addressed again. Spills have metadata sidecars, but uploads have neither a hash nor provenance, and web pages and language server results do not exist at all.

There is no content addressing outside of spills, so uploads are identified by filename and cannot be deduplicated across sessions.

Spilled evidence cannot be retrieved. A spilled tool result can be read by its identifier, but `bm25_search` indexes only uploads and workspace files, so the spill is invisible to search. Since spills are the densest knowledge the agent already produces, this is the cheapest significant retrieval improvement available.

Deliverables come from regular expressions over prose rather than from structured records.

There is no network ingress at all. The full set of tools is `read_file`, `read_spill`, `write_file`, `patch_file`, `execute_command`, `spawn_sub_agent`, and `bm25_search`. There is no fetch, no search, and therefore no egress policy that would gate them.

Instructions are only discovered at the workspace root or globally, never by walking upwards.

The composer has no file mentions, no paste handling, and no drag and drop.

There is no code intelligence, so no diagnostics or symbol information enters the system.

Search re-walks and re-indexes the workspace on every call, so its cost grows with the size of the workspace rather than with the size of the query.

And nothing today distinguishes text the user wrote in their own repository from text a stranger published on the internet. That distinction is not a nicety. It becomes safety-critical the moment web fetch exists.

---

# 4. Program strategy

The work happens in three layers, in dependency order.

**Layer 1 is the evidence substrate.** One kind of record, one way of addressing it, one catalog. Everything that gets ingested becomes an evidence record, and no source gets special treatment.

**Layer 2 is the ingress surfaces.** These are the things that produce evidence: mentions and drops in the composer, pasted images, instructions found by walking directories, web pages, language server results, and spills registering themselves.

**Layer 3 is retrieval and projection.** This covers searching the catalog, extending the retrieval surface with provenance and caching, and producing the structured records that the shell renders as typed blocks and reference chips.

---

# 5. The core model

## 5.1 The evidence record

Every ingested thing is described by a record of this shape:

```json
{
  "evidence_id": "sha256:9f2c…",
  "kind": "web_page | file_mention | image | upload | spill | lsp_diagnostic | deliverable | instruction | character_card | lorebook",
  "origin": {
    "surface": "composer_drop | agent_tool | instruction_walk | session_upload",
    "uri": "https://example.com/spec",
    "path": "docs/SPEC.md",
    "session_id": "sess_20260919-2310",
    "workspace_id": "ws_go_llm_harness"
  },
  "trust": "user_local | workspace | untrusted_external",
  "media_type": "text/plain | text/markdown | image/png | application/pdf",
  "byte_count": 41233,
  "created_at": "2026-09-19T23:10:00Z",
  "derived_from": ["sha256:4b81…"],
  "notes": "truncated at 200 KB for injection; full blob intact"
}
```

Three fields carry most of the weight.

The `trust` field records where the content came from in terms of how much it should be believed. Anything fetched from the network is marked `untrusted_external`. Untrusted evidence is never eligible to occupy an instruction position, and when it is injected it must be delimited and labelled as data. This is the boundary against prompt injection, and making it a property of the record rather than a property of a code path means every consumer gets it by construction.

The `evidence_id` is the sha256 of the content, which matches the scheme `hashSpillID` already uses. Identical bytes are one record no matter how many times they are referenced. The `derived_from` field keeps the lineage, so a page snapshot, the text extracted from it, and a summary of that text remain connected.

The `origin` and `trust` fields together are what make recall honest later. The archived memory units in the Session Event & Memory System's row `9.3` are derived from these records, and that plan's requirement that recalled memory carries provenance can only be met if the provenance was captured when the content came in.

Two kinds are added by the card series, and they behave like any other evidence once they are
registered. `character_card` covers the converted document, which is searchable through its
Markdown rendering, and `lorebook` covers a book, which may be a specification-shaped document or a
files-plus-index pair. Both are `user_local`, both carry a `derived_from` pointing at the file the
user imported, and neither is treated as instruction material even when it contains imperative
sentences. A card's description is written in the second person, so it reads like an instruction,
and it is a document about a character rather than a directive to the harness.

One distinction belongs here rather than in a later section. A card that a session has imported as
its persona is prompt material, and it is still evidence. The two relationships are separate: the
persona block reads the document, and the evidence record makes it searchable and attributable.
Nothing about the persona requires the substrate, which is why phase K can land before phase L.

## 5.2 Where things are stored

```
.goharness/
  evidence/                        # workspace-scoped store of content-addressed blobs
    <sha256>.blob                  # the raw bytes, immutable
    <sha256>.meta.json             # the evidence record
  sessions/<sid>/
    uploads/<sha256>.<ext>         # uploaded files, named by hash under row 12.19
    spill/<sha256>.txt             # existing spill payloads, unchanged paths
    spill/<sha256>.json            # existing spill metadata, unchanged
    evidence-index.jsonl           # append-only list of what this session ingested
```

The card repository sits outside both of those stores, because it is user-owned material rather
than runtime state. The path is configurable, and it defaults to `~/.goharness/characters/`:

```
~/.goharness/characters/             # the repository, shared by every workspace
  <name>/
    source/                          # the imported file, byte for byte
    card.json                        # the normalised card
    character.md                     # the readable rendering, which is what search indexes
    lorebook.json                    # a standalone book, when the card carried one
    conversion.json                  # converter name and version, source hash, time, files written
  <name>/memory/                     # optional: files the agent maintains, with an index over them
    index.json                       # keys, position, priority, tags, and a path per entry
    <entry>.md                       # the content, in files the tools already read and write
```

Two consequences are worth naming. A session imports characters rather than owning them, so a card
converted once serves every workspace, and a session that imports nothing is indistinguishable from
the harness as it exists today. And because the repository is outside the workspace, the traversal
guard from phase B does not apply to it, which means reads from the repository are a deliberate
exception in the sandbox rules rather than an accident.

Blobs are scoped to the workspace so that two sessions referring to the same file share one copy rather than storing it twice. References remain local to a session, so the question "what did this session ingest" stays answerable without scanning the whole workspace.

Spill files keep the paths they have today. This plan registers spills in the catalog rather than moving them, which means `read_spill` and every existing spill identifier continue to work unchanged.

## 5.3 Staging is not injection

The distinction the system turns on is between two states a piece of evidence can be in.

**Staged** means the evidence has been recorded, can be addressed, can be searched, and is visible in the interface. It costs the model nothing.

**Injected** means the evidence is in the prompt. It costs tokens, and it keeps costing tokens until compaction.

Four rules follow.

Injection is always explicit. It happens because a mention was committed, a tool returned something, or a retrieval block was assembled under a budget. It never happens as a side effect of a file merely being uploaded.

Injected evidence is delimited and attributed, with a marker identifying the evidence and its origin at the start and a closing marker at the end.

Untrusted evidence is injected only as data, inside those delimiters, with a standing instruction that content inside an evidence block is not an instruction to be followed.

Injection always respects a budget, and the resulting block reports what was left out because of it. This is the same honesty rule the memory plan applies to recall.

Uploads currently blur this line. A file dropped into the composer is written to disk and quietly joins the search corpus. The staged and injected split makes that relationship visible instead of incidental.

## 5.4 What a retrieval hit contains

```
EvidenceHit {
    EvidenceID
    Kind
    URI          // a URL, or a path relative to the workspace
    Score
    Snippet      // bounded, with offsets into the blob
    Offsets
    Trust
    SessionID
}
```

The current `SearchResult`, which contains only a path and a score, is the gap. It has no provenance, no offsets, no snippet, and no notion of trust. The engine keeps its scoring mathematics. The catalog layer adds provenance by resolving identifiers to records.

---

# 6. Execution phases

# Phase A — The evidence substrate

## Goal

One record type, one addressing scheme, and one catalog, used by everything that follows.

## Deliverables

1. **The `EvidenceRecord`** as described in section 5.1, including a schema version field.

2. **A content-addressed blob store** at `.goharness/evidence/`, with immutable blobs and `<id>.meta.json` sidecars. This deliberately mirrors the shape of `spill.go`, so the codebase has one addressing idiom rather than two.

3. **A session evidence index** in `evidence-index.jsonl`, append-only, recording references with timestamps.

4. **A catalog API** covering `Register`, `Resolve`, and `List` by session or workspace, plus `Derive` for creating a new record from an existing one.

5. **Structured reporting of produced paths.** Tools should report the files they wrote through the channel that already exists in `tool_message_meta.go`. The regular expressions in `src/artifacts.go` are demoted to a fallback that logs when it fires, which keeps any un-migrated tool visible instead of silently carrying the whole feature.

## Proposed new files

`src/evidence.go` for the record type, identifiers, and the trust values. `src/evidence_store.go` for the blob store and metadata. `src/evidence_catalog.go` for the session and workspace indexes and registration.

## Files likely to change

`src/artifacts.go`, which becomes a fallback. `src/tool_message_meta.go`. And `src/spill.go`, which registers a record when it saves.

## Why this comes first

Every later phase is a producer for this catalog. Building the ingress surfaces first would leave us with five incompatible ideas about what a thing the agent looked at actually is.

---

# Phase B — Content-addressed attachments

## Goal

Generalize the addressing scheme that spills already use so that it covers everything uploaded or attached.

## Deliverables

1. **Uploads stored by hash**, at `uploads/<sha256>.<ext>`. The original filename is kept in the evidence record rather than in the path, so the hash provides identity while the record preserves what the user called it.

2. **Deduplication.** Uploading the same bytes again returns the existing evidence identifier instead of writing a second copy.

3. **Reference semantics.** Several sessions in one workspace share a blob. Removing one session's reference does not delete a blob that another session still refers to.

4. **A split size policy.** Two settings replace the single 10 MB limit: `ingest_max_inject_bytes` governs the largest thing that may be placed into a prompt, and `store_max_bytes` governs the largest thing that may be stored at all. A 50 MB PDF becomes storable and referenceable while only extracted or selected portions are ever injectable.

5. **A traversal guard that does not rely on substring matching.** The correct check resolves the target path and confirms containment, using something like `filepath.Rel` over cleaned absolute paths, rather than asking whether the string contains `.goharness`.

## Why this comes before the other ingress work

Deduplication and hash identity are cheapest to introduce before more producers exist. Doing this after web fetch and image paste land would mean migrating three formats at once.

---

# Phase C — Integrating spills

## Goal

Close out the partial status on row `12.6` by making spilled output a first-class part of the evidence system.

## Deliverables

1. **Registration on spill.** When `saveSpill` writes a payload, it also writes an evidence record with the kind `spill`, the origin surface `agent_tool`, and notes recording which tool produced it and how large it was.

2. **Searchable spills.** The `bm25_search` tool gains a scope for evidence, covering the session's spills, uploads, and fetched page snapshots. A spilled test log becomes findable by keyword rather than only by its identifier.

3. **Referenceable spills.** `read_spill` keeps working exactly as it does today, with no identifier churn, and the evidence record links back to it.

4. **A retention rule.** Version one deletes nothing automatically, but the catalog reports how much space each session occupies, so the operator surface under rows `13.10` and `14.10` can offer cleanup. Evidence the agent might cite later should never be evicted silently.

## Why now

Spills are the densest knowledge source GoHarness already produces, and they are currently the only one invisible to search. Making them retrievable is the largest improvement available for the least work.

---

# Phase D — Retrieval over the catalog

## Goal

Make retrieval aware of provenance and cheap enough to call often.

## Deliverables

1. **`EvidenceHit` results** as described in section 5.4, replacing `SearchResult` for the agent-facing tool so that every hit carries an identifier, a kind, a URI, offsets, a snippet, a trust value, and a session.

2. **Trust-aware ranking.** Results marked `untrusted_external` are returned labelled, never silently merged with results from the workspace. A single result list that mixes the two without distinguishing them is a defect rather than a feature.

3. **An index cache invalidated by modification time**, following the lesson from Aider described in section 7.4. Per-file term frequencies are persisted alongside the file's modification time, and only changed files are re-indexed. Today every call re-walks the workspace.

4. **A vocabulary for scopes**: `workspace`, `session`, `evidence`, and `all`. The `session` scope keeps the turn-folder behaviour it has today.

5. **Budgeted evidence blocks** for prompt assembly. Retrieval ranks first, then emits a bounded block that names what was omitted because of the budget.

6. **A hook for personalization.** Documents whose paths or identifiers overlap with what is already in play in the session should rank higher. This is Aider's idea applied without the graph machinery, since GoHarness has no definition and reference graph yet.

## Why now

Every ingress surface added later benefits immediately, and the caching work is independent of those surfaces.

---

# Phase E — File mentions

## Goal

Make `@` in the composer resolve to workspace files and stage them deliberately.

## Deliverables

1. **A resolver endpoint on the server** that turns a query into ranked paths with scores, matched character indices, and match types. Ties break deterministically by path, and the result set is bounded.

2. **A typeahead contract with incremental queries** and a stale-query guard, so a slow response for one query cannot overwrite the results of a later one. This is the lesson from section 7.5.

3. **Staging semantics.** Committing a mention stages the file: an evidence record is registered with the kind `file_mention` and the trust value `workspace`. Injection happens as a delimited evidence block whose size is governed by the budget, not by dumping the file.

4. **Honest labelling in the interface.** The difference between staged and sent to the model has to be visible in the composer. The roadmap's rule about not showing controls that do nothing applies directly here: a mention that silently injects forty kilobytes of text would be exactly that kind of lie.

## Files likely to change

`src/web.go` for the resolver route, `src/web/js/composer.js` for the trigger and dropdown, and `src/workspace_entries.go`, so that the picker respects the same ignore and collapse rules as the visible tree.

---

# Phase F — Walking parent directories for context

## Goal

Port the instruction-discovery behaviour from Codex faithfully, adapted to the filenames GoHarness already looks for.

## Deliverables

1. **An upward walk to a project root marker.** The marker list is configurable, with `.git`, `go.mod`, and `.hg` as sensible defaults, and an empty list that disables parent traversal entirely, exactly as the reference implementation allows.

2. **Concatenation from root to working directory**, in that order, with an explicit separator between entries so the assembled block is greppable and testable.

3. **An override filename.** A file such as `AGENTS.override.md` takes precedence over the base file in the same directory. It overrides its own directory's file, not the files of its ancestors.

4. **A byte budget with visible truncation.** Content beyond the budget is cut, and the assembly notes that it happened. Silent truncation is not acceptable.

5. **No walking past the project root**, and nothing read outside the workspace subtree in a way that would surprise the user. The global and binary-directory instructions remain a separate and explicit layer.

6. **Session pinning stays authoritative.** The `/api/sessions/pinned` endpoint keeps working, and auto-discovered entries are recorded as evidence with the kind `instruction`, so the interface can show what was injected and where it came from.

## Why now

It is self-contained, it matters most in repositories where projects are nested, and it changes one function in one place. It is a good small honest win to take alongside the larger ingress work.

---

# Phase G — Images

## Goal

Let images be pasted and dropped, store them as evidence, and be honest about when the model can actually see them.

## Deliverables

1. **Capture from paste and drop** in the composer, including screenshots taken from the clipboard. Files route through the content-addressed store built in Phase B.

2. **An evidence record** with the kind `image`, a media type determined by sniffing the bytes rather than trusting the filename extension, and recorded dimensions and size.

3. **Model access as a tool**, following the design in section 7.2. The tool takes a path or identifier, returns a data URL, and lets the model decide when vision is worth the tokens. An image sitting in the composer is staged rather than automatically injected.

4. **A detail control**, with a bounded resize by default and an explicit path to the original resolution, so image tokens are a budget rather than a surprise.

5. **An honest gate on provider support.** Image transport differs across the providers wired in `src/llm.go`. OpenAI-style models take image URLs, while Anthropic and Gemini use inline image blocks. The capability must be detected per profile and surfaced, and when a profile cannot carry images the tool must say so. Failing obscurely, or silently dropping the image, are both worse than an honest refusal.

## Why now

The storage half depends on Phase B. The tool half is independent and small, so this phase can proceed as soon as content addressing exists.

---

# Phase H — Web search and fetch

## Goal

Bring knowledge in from the network with typed actions, content-addressed snapshots, and an egress policy that is designed rather than bolted on.

## Deliverables

1. **`web_fetch`.** Fetch a URL, apply limits on redirects, timeouts, and size, extract readable text, and store the raw response as an evidence blob with the kind `web_page` and the trust value `untrusted_external`. The extracted text becomes a derived record, so the lineage from raw bytes to usable text is preserved.

2. **`web_search`.** Return ranked results with a title, a URL, and a snippet, recorded as typed action records. The choice of backend is an open decision described in section 10. Results are not fetched automatically. A page becomes evidence only when it is actually opened.

3. **Typed actions.** Search, open page, and find in page are recorded, so the transcript can say that a URL was opened or that a word was found in it and remain accurate after the page changes. This is the lesson from section 7.3.

4. **An egress policy seam.** An allow or deny rule set is consulted before any request is issued. It should be implemented through the capability seam the policy engine will provide under rows `12.1` and `12.16`. For version one the default is either to allow with logging or to allow only an explicit allowlist, and this is the user's choice. Whichever is active must be visible in the interface.

5. **Untrusted delimiters.** Fetched content is injected only inside evidence blocks carrying an explicit notice that the content is not an instruction. This is the most important safety property of the phase. A web page must not be able to change the agent's behaviour by containing a phrase like "ignore previous instructions", no matter how it is formatted.

6. **Slow fetches as jobs.** A fetch that exceeds an interactive time budget should be runnable as a Task-system job under row `11.7`, so a host that never responds cannot stall a turn. The seam is defined here and implemented there.

## Why last, despite being Wave 1

This is the row with the fewest dependencies, which normally argues for doing it first. It is last in this plan for two reasons. It introduces the most risk, because the content is untrusted and because it reaches outwards from the machine, and it benefits substantially from the evidence and policy shapes being settled before it is written. Sequencing it last is a deliberate choice about risk, not about difficulty.

---

# Phase I — The LSP seam

## Goal

Make code intelligence available as a knowledge source, staged behind a capability seam.

## Deliverables

1. **Transport.** A JSON-RPC client over stdio, talking to a language server process per workspace, with a managed lifecycle covering spawn, initialize, shutdown, and restart with backoff after a crash.

2. **Document synchronization.** `didOpen` and `didChange` notifications driven by the existing read and write paths, so the server's view tracks what the agent actually touched rather than preloading the whole project. The lifecycle methods `initialize` and shutdown are handled alongside these.

3. **Capability gating.** Only the capabilities a server advertises are exposed. Something the server does not offer is not a surface the user can see.

4. **A first version of the surface.** `diagnostics` and `document_symbols` become evidence records with the kind `lsp_diagnostic` and the trust value `workspace`, each carrying a file path, a range, a severity, and the server that produced it.

5. **Timeouts everywhere.** A language server is a third-party process and can hang. Every request is bounded, and a failure is a normal reported outcome rather than a stall.

6. **Server configuration as data.** Which server runs for which language is configured in the workspace configuration and visible in settings, not hardcoded.

## Why this is staged and last

This is the largest build in the system and the one with the most operational surface, because it involves external processes and per-language configuration. It is genuinely useful, but nothing else in the plan waits on it. Specifying the seam now means that later work, including the capability seams under row `12.1` and the validation loops in the Execution & Review System, can target a stable interface instead of inventing a second definition of what a diagnostic is.

---

# Phase J — The card repository and the converter seam

## Goal

A person converts a card once, inspecting the result, and a session then chooses which characters it
is using.

## Deliverables

1. **The repository layout** described in section 5.2, at a configurable path that defaults to
   `~/.goharness/characters/`. One directory per character, with `source/` holding the imported file
   byte for byte, alongside the files converted from it.

2. **A converter seam rather than a library dependency.** The harness never parses a card. It runs a
   configured command with a documented contract: an input path, an output directory, and an exit
   code. The reference implementation in `docs/character-cards/impl/` satisfies that contract
   already, and anything else that writes the documented files works too. This keeps the parsing
   offline, and it keeps a Python implementation out of the runtime of a Go program.

3. **A manifest per character**, written as `conversion.json`, recording the converter and its
   version, the source hash, the time of conversion, and the files produced. The manifest is what
   lets the harness report that a directory was converted by an older version, and it is what makes
   a re-conversion safe to offer.

4. **Reading the converted shapes in Go.** A small reader for `card.json` and for the `lorebook_v3`
   shape, covering the seventeen entry fields the guide tabulates. No PNG, no base64, no archive
   handling at runtime, because none of that is in the converted output.

5. **Per-session imports.** A record of which characters a session is using, with operations to add,
   remove, and list them. A session with nothing imported behaves exactly as the harness does today,
   which is the property that makes this safe to land early.

6. **Retention of the imported file**, read back for two purposes: re-deriving the output when the
   converter changes, and using the portrait inside an image card as an ordinary image asset.

## Proposed new files

`src/character_repository.go` for the layout, the manifest, and listing. `src/card.go` for the
converted shapes and the reader. `src/session_characters.go` for the import list.

## Files likely to change

`config.example.json` gains the repository path and the converter command. The command surface gains
`characters convert`, `characters list`, and `characters import`.

## Why this comes first

It is the only phase in the series with no dependency on the evidence substrate, so it can be built
while phase A is still in progress. Everything after it needs a converted directory to work with,
which means the persona has nothing to render and the lorebook layer has nothing to match without
it. It is also where the offline-parsing decision is either kept or quietly abandoned, and keeping it
is a property of the seam rather than a promise.

---

# Phase K — The persona block

## Goal

One imported card becomes standing text in the system prompt, in a position a person can change.

## Deliverables

1. **The rendering rule.** Card fields become one labelled block. `description`, `personality`,
   `scenario`, and `system_prompt` contribute. `creator_notes` never contributes, because the
   specification says it is for the human. `alternate_greetings` and `first_mes` are read and
   preserved but never injected, because the harness has no opening message to greet with.
   `mes_example` contributes only when a person enables it.

2. **Registration as a block.** The persona joins the prompt assembly as a block whose default
   position follows the hardcoded base and precedes the environment description. When row `12.30.8`
   has landed, the block is editable and budgeted like any other. When it has not, the position is
   fixed and documented, so this phase does not depend on that one.

3. **A labelled prompt element.** The injected text identifies its source, so a reader of the prompt
   can tell which part came from a card and which came from the workspace. The label carries the
   character's name as the card gives it, falling back to the filename stem when the name is blank,
   because real cards do carry blank and whitespace-only names.

4. **One persona per session at most**, chosen from the imported characters. A second card is
   evidence rather than identity, which keeps the merge question from arising.

5. **A stated exception, written down rather than implied.** The persona is text a user chose, so it
   is the one place where content that arrived from outside the workspace may read like an
   instruction. That exception is recorded in the plan and in the code, together with the reasoning:
   the document is local, the import is explicit, and the block is labelled.

6. **Tests against real cards.** One V1 card, one V2 card, and one V3 card from the three collections
   in `docs/character-cards/` each render to a block containing the description and none of the
   excluded fields.

## Files likely to change

`src/agent_run.go`, at the point the system prompt is assembled. `src/agent.go` for instruction
loading, since the persona is a source of standing text of the same kind.

## Why this comes second

It is the smallest end-to-end demonstration that the repository works: a card goes in, and what the
agent receives changes. It is also the item a user is most likely to ask for by name.

---

# Phase L — Cards and lorebooks as evidence

## Goal

Every converted character, and every book that came with one, is in the catalog and searchable
alongside everything else that has been ingested.

## Deliverables

1. **Two `kind` values**, `character_card` and `lorebook`, added to the enumeration in section 5.1.
   Nothing else about the record changes.

2. **The Markdown rendering is the indexed text**, and the converted JSON is the blob. A search hit
   resolves to the document the user imported rather than to a summary of it, and the offsets point
   into the rendering that a person can open and read.

3. **Lineage that reads in one direction.** The source file, the converted card, and the lorebook
   derived from it are three records connected by `derived_from`, so the question "where did this
   entry come from" has an answer that terminates at the bytes the user imported.

4. **Registration on import rather than on every run.** Importing a character registers its records
   once, and re-importing compares the manifest so that unchanged characters do not produce new
   records. This is the same modification-time discipline that phase D applies to the search index.

5. **A trust value of `user_local`**, with the origin recording the repository path and the character
   name. A card downloaded from a card site is still local material as far as this system is
   concerned, because the user chose it, and that is a different statement from saying it is safe.

## Why this phase is short, and why it waits

The substrate does the work, so this phase is a set of producers rather than a new mechanism. It
waits on phase A for the obvious reason that there is nothing to register into until the catalog
exists, which is why the roadmap row is `Blocked` rather than merely ordered.

---

# Phase M — The lorebook as a retrieval and injection layer

## Goal

A book's entries reach the prompt under the format's own rules, with the budget and the drop order it
specifies, and with an account of what was left out.

## Deliverables

1. **The matcher.** A scan window of recent messages is matched against each entry's `keys`, with
   `secondary_keys` under `selective`, and with `case_sensitive`, `use_regex`, `constant`, and
   `enabled` honoured. An absent `use_regex` reads as false, because the alternative turns literal
   keys into patterns. The guide's entry table is the field reference.

2. **The budget and the drop order.** `token_budget` caps the injected total. When it is exceeded,
   entries carrying `@@ignore_on_max_context` go first, then the lowest `priority`, then the lowest
   `insertion_order`. Every dropped entry is reported, because a silent drop is the failure mode that
   makes a memory system untrustworthy.

3. **Positions and depths.** `before_char` and `after_char`, with the `depth` family counted back
   from the newest message, applied as an ordered injection list rather than as one block. This is
   the part that does not fit the block model of row `12.30.8`, and the plan states that rather than
   pretending otherwise.

4. **Recursive scanning**, with the limits the reference implementation already settled on: four
   passes of depth and a recursion limit of three, so a book whose entries reference each other
   cannot loop forever.

5. **Entries that point at evidence.** The format reserves `extensions` on every entry, so an entry
   can carry the evidence identifier it came from, and a hit can be traced back to the document.

6. **Two content shapes, one matcher.** An entry may carry its content inline, as the specification
   does, or reference a file on disk through a path in the index. The matcher works on keys,
   position, priority, and content, and it does not care where the content came from, which is what
   makes the files-plus-index shape cheap rather than a second implementation.

7. **A diagnostic command.** Given a book and a piece of text, print what would be injected, in
   order, with token counts, and what was dropped and why. This is the artefact that makes the layer
   reviewable, and it doubles as the acceptance test.

## Proposed new files

`src/lorebook.go` for the entry model and the reader. `src/lorebook_match.go` for keys, scan windows,
and recursion. `src/lorebook_budget.go` for the budget and the drop order.

## Why this is the centre of the series

It answers the question the plan had left open: what is dropped when the window fills. It also turns
the format from a file to read into an algorithm to reuse, which is the part of the card ecosystem
that is worth taking.

---

# Phase N — Memory the agent maintains

## Goal

The agent can add to, read, retire, and list what it knows, and every one of those acts is
attributable.

## Deliverables

1. **Four tools**, matching the surface the design settles: `memory_write`, `memory_read`,
   `memory_list`, and `memory_disable`. They belong to a mode-scoped tool set rather than to every
   turn, because a seven-tool prompt becomes an eleven-tool prompt otherwise.

2. **A log entry for every call**, recording the session, the entry, the operation, and the trust
   value `agent_written`. An agent-written fact is a different kind of claim from a fact a person
   wrote, and the record should say which it is.

3. **The files-plus-index storage shape**, where the index carries keys, position, priority, and tags,
   and the content lives in files the tools already read and write. The specification shape remains
   supported for books that arrived that way, through the same matcher.

4. **A configuration switch for creating a new book**, defaulting to allowed. New books are new
   injection surfaces with their own budgets, so the switch exists, and it is configuration rather
   than a mode because a mode is focus rather than a permission boundary.

5. **A budget for agent-written content**, so that memory cannot grow without limit and crowd out the
   task. What happens when it is reached is one of the open decisions below.

6. **Concurrency that is decided rather than assumed.** Two sessions may edit the same book. The
   plan chooses locking with a re-read over silent last-writer-wins, because the alternative loses
   work without saying so.

## Why last, and why after the retrieval layer

Writing is only safe once reading is honest about order and budget. An agent that adds entries
without a way to see what they displace is guessing, and the retrieval layer is what turns that
guess into a measurement.

---

# 7. Lessons from reference projects, applied directly

## 7.1 Instructions are hierarchical, ordered, and budgeted

From Codex's `agents_md`: walk up to a marker, concatenate from the root down to the working directory, never walk past the root, honour an override filename, and truncate against an explicit budget with a visible warning. Applied in Phase F.

## 7.2 Images are addressed by path and delivered as renderings

From Codex's `view_image`: address the image on disk, deliver a data URL, offer an explicit detail level, and let the model decide when to look. Applied in Phase G.

## 7.3 Record actions, not page text

From Codex's typed search actions: searching, opening, and finding are the primitives, and the transcript is derived from those records rather than from whatever was on the page. Applied in Phase H, so that claims about web sources stay auditable after the web moves on.

## 7.4 Cache by modification time, rank by what is in play, send structure

From Aider's repository map: the map is cached incrementally, personalized by the files and identifiers already in the conversation, and rendered as lines of interest rather than whole files. GoHarness applies the caching and personalization parts to its own index in Phase D. The definition and reference graph itself is out of scope, because it is a much larger project and because `docs/BM25_SCALING_RESEARCH.md` argues that lexical retrieval is the right first choice at the scale GoHarness targets.

## 7.5 Stream matches with a stale-query guard

From Codex's fuzzy file search: typeahead is a session protocol with snapshot invalidation, scored matches, and character indices for highlighting. Applied in Phase E.

## 7.6 Content addressing is already the house style

From GoHarness's own `src/spill.go`: the project already addresses large output by content hash and already stores payloads beside metadata. Row `12.19` extends a pattern that is proven in this codebase, which means the migration story is "extend what exists" rather than "introduce a new storage era".

---

# 8. Concrete slices of work

## Slice 1 — The evidence substrate and content-addressed uploads

**What it adds.** The evidence record, the blob store, and the session index. Uploads stored by hash with deduplication. Containment-based path validation replacing the substring check. Structured artifact reporting from the write and patch tools, with the regular expressions demoted to a logged fallback.

**How we know it worked.** Uploading the same file in two sessions produces one blob and two references, and the list of deliverables no longer depends on the wording of a tool message.

---

## Slice 2 — Spill evidence and retrieval over the catalog

**What it adds.** Spills register themselves. Search gains an evidence scope. Hits carry provenance, a trust label, and offsets. The index is cached and invalidated by modification time.

**How we know it worked.** A word buried inside a 300 KB spilled test log can be found by search, the result identifies the spill and points at offsets within it, and running the same search twice does not walk the workspace a second time.

---

## Slice 3 — File mentions

**What it adds.** A resolver endpoint returning scored matches with indices. An `@` trigger in the composer with a typeahead guarded against stale queries. A visible distinction between staged and sent.

**How we know it worked.** Typing `@` lists files respecting the same ignore rules as the visible workspace tree, and a committed mention shows as staged until something actually injects it.

---

## Slice 4 — Walking instructions

**What it adds.** The marker-based upward walk, concatenation from root to working directory, the override file, and the byte budget. Discovered instructions register as evidence.

**How we know it worked.** In a workspace nested three levels inside a repository root, instructions from every level are injected in root-to-working-directory order, the override file wins within its own directory, and a file that exceeds the budget is truncated with a visible note.

---

## Slice 5 — Images

**What it adds.** Paste and drop capture, content-addressed storage, sniffed media types, a model-invoked image tool with a detail control, and per-profile capability detection with an honest message when images are not supported.

**How we know it worked.** A pasted screenshot is stored once, appears as staged evidence, and can be read by the model on profiles that support images, while being clearly refused rather than silently dropped on profiles that do not.

---

## Slice 6 — Web fetch and search

**What it adds.** Fetch with limits, storing both the raw snapshot and the extracted text as linked evidence. Search returning typed results. An egress check that runs before the request, with the active mode visible to the user. Untrusted delimiters on anything injected.

**How we know it worked.** A fetched page can be found later by keyword and is still attributed to its URL, and it cannot be injected without its trust marking and notice.

---

## Slice 7 — The LSP seam

**What it adds.** A stdio JSON-RPC client with lifecycle management and restart, diagnostics and symbols as evidence, and per-language server configuration.

**How we know it worked.** Opening a file containing a real error produces a diagnostic record with the file, range, and severity, and a server that hangs degrades into a reported timeout rather than stalling the turn.

---

## Slice 8 — Projecting deliverables and references

**What it adds.** Deliverables derived from the evidence catalog instead of from prose, reference chips that point at evidence identifiers, and feedback routing kept out of the model's context, which is operator-side under row `12.23`.

**How we know it worked.** The list of deliverables for a turn comes from structured records, and every entry resolves to an evidence record whose origin can be shown to the user.

---

## Slice 9 — The repository, the converter seam, and the persona

**What it adds.** Phases J and K together. The repository layout and its manifest, the converter
invocation with `characters convert`, `list`, and `import`, the reader for the converted shapes in
Go, the per-session import list, and the persona rendered as a labelled block in the system prompt.
The imported file is kept in `source/` from the beginning rather than retrofitted, because the
conversion is the thing most likely to be redone.

**How we know it worked.** A card image is converted once, the original stays byte-identical beside
the output, importing it into a session adds one labelled block to the system prompt, removing it
restores the prompt the harness sends today, and a second session in another workspace can import
the same character without converting it again.

---

## Slice 10 — Cards and lorebooks in the catalog

**What it adds.** Phase L. The two new evidence kinds, the Markdown rendering as the indexed text,
the lineage that connects source, card, and book, and registration that follows the manifest rather
than the clock.

**How we know it worked.** A word that appears only inside a card's description, and a key that
appears only inside its book, are both findable through search, each hit names the character it came
from, and re-importing an unchanged character produces no new records.

---

## Slice 11 — The lorebook injection layer

**What it adds.** Phase M. The matcher, the scan window, the budget with its specified drop order,
positions and depths, recursion with limits, and the diagnostic command that prints the decided
injection.

**How we know it worked.** Run against the thirty-one-entry book from the third collection, the
diagnostic prints the injected entries in order with token counts and the dropped entries with
reasons, and the totals match the reference implementation's recorded figures for the same input.

---

## Slice 12 — Memory the agent maintains

**What it adds.** Phase N. The four tools in a mode-scoped set, the log entries, the files-plus-index
storage, the configuration switch for creating a book, the content budget, and locking.

**How we know it worked.** In one session the agent writes a fact, the next session's prompt contains
it without anything being pasted, the log shows who wrote it and when, disabling it removes it from
the prompt while keeping the file, and two concurrent writers leave both entries present rather than
one of them lost.

---

# 9. Testing plan

## 9.1 The substrate

Tests check that identical bytes registered twice produce one blob and two references, that a second write with the same identifier does not alter the first, that containment validation rejects `../` traversal and absolute paths outside the session root, and that evidence can still be resolved by identifier after the session restarts.

## 9.2 Retrieval

Tests check that hits carry provenance including the identifier, kind, URI, offsets, snippet, and trust. They check that untrusted hits are labelled in the payload rather than merged silently. They check that repeated searches return the same order, which requires deterministic tie-breaking. They check that a modified file is re-indexed while an untouched file is not re-tokenized. And they check that a budgeted block reports what it omitted.

## 9.3 Ingress

Tests check the ranking, indices, and stale-query rejection of the file mention resolver. They check the instruction walk against each of its rules: stopping at the marker, ordering from root to working directory, precedence of the override file, truncation at the budget, and the empty marker list disabling parent traversal. They check that media sniffing rejects a file named `.png` that is not actually an image and records the true dimensions. They check that uploads deduplicate across sessions in one workspace.

## 9.4 The network

These tests run without a live network. They check the redirect cap, the timeout, the size cap, and non-HTML content types. They check that the egress rule is consulted before any request is issued, by asserting that no connection is made when the rule denies. They check that fetched evidence carries the untrusted marking and is injected only inside the delimiters. And they check that search results are recorded as typed actions and can be reconstructed from the record alone.

## 9.5 The LSP seam

Tests check that a server crashing mid-request produces a reported failure rather than a hang, that capabilities are gated on what the server advertises, and that diagnostics map to evidence records with the correct ranges.

---

## 9.6 Cards, personas, and lorebooks

Three kinds of test carry most of the weight here, and all three are cheap because the material
already exists.

**The converter seam is tested against all three card versions.** A V1 flat card, a V2 card, and a V3
card from the collections described in `docs/character-cards/` are converted into a temporary
directory. The assertions are that the layout matches, that `source/` is byte-identical to the input,
and that the manifest records the hash it read rather than the hash of what it wrote.

**The reader is tested against the guide rather than against intuition.** A V1 card and a V3 card
each yield the same twenty-three `data` fields, the fields the guide tabulates as TypeScript-only
are read when present and absent without error when not, and the optional fields distinguish an
absent value from an explicit false. That last case is a real bug found while writing the reference
implementation, and it is the kind of bug that only a round trip catches.

**The matcher is tested with recorded numbers from a real book.** The thirty-one-entry book from the
third collection is the fixture, and the expected result for one recorded input is sixteen entries
injected, six dropped, and a total of four hundred tokens. A test that asserts a number recorded from
a real object is worth more than a test that asserts a shape, because it fails when behaviour drifts
rather than only when code breaks.

Two smaller families round this out. The persona rendering is tested against three real cards, one
per version, asserting that the description text is present and that `creator_notes`,
`alternate_greetings`, and `first_mes` are absent. The memory tools are tested for the properties the
logging is meant to guarantee, which are that every call produces a record and that a disabled entry
survives on disk while leaving the prompt.

# 10. Risks, anti-goals, and open decisions

## Risks

1. **Evidence accumulating without limit.** Every source registering records could grow the store indefinitely. The mitigation is reporting per-session size in the catalog and offering cleanup through the operator surface, never evicting anything silently.

2. **Prompt injection through fetched content.** This is mitigated structurally by the trust field and the delimiters, but only if every injection path respects them. That needs a test suite rather than good intentions.

3. **Capabilities that only appear to exist.** Language servers differ, image transports differ, and search backends differ. A control that is not genuinely honoured is worse than a missing control, so each capability has to be detected rather than assumed.

4. **Index churn.** Caching by modification time can thrash on a workspace that builds produce output into. There needs to be a bounded amount of re-indexing per call.

5. **A persona that carries instructions rather than a description.** A card is written in the
   second person and reads like a directive, and a downloaded card is text a stranger wrote. The
   mitigation is that importing is explicit, the persona is a labelled block, the document is local
   and reviewable, and the exception is recorded rather than assumed. The residual risk is real and
   belongs in the documentation a user reads before importing somebody else's card.

6. **Agent-written memory outliving its truth.** A fact the agent wrote in one session is injected
   into the next, and it may be wrong or superseded. The mitigation is that the log records who
   wrote each entry and when, the trust value distinguishes an agent from a person, disabling is
   cheap, and the content lives in files a person can read and correct.

7. **Converter drift.** The repository is produced by a program that is not part of the harness, so
   a directory can be older than the code that reads it, and a card converted badly stays bad. The
   mitigation is the manifest, a doctor check that reports stale directories, and the retained
   source file that makes re-conversion possible at any time.

8. **Scope.** Five phases is a large addition to a plan that already carried nine. The mitigation is
   that each phase is useful on its own, that phase J has no dependency on the substrate, and that
   slice 9 alone delivers the persona, which is the item most likely to be wanted first.

9. **Two retrieval systems drifting apart.** The session query projection in the memory plan's row `12.18` and the evidence search here could become separate engines. They should share both `BM25Engine` and the vocabulary of provenance.

## Anti-goals

1. Do not inject uploaded or fetched content automatically. Bringing something in is not the same as putting it in front of the model.

2. Do not introduce a second way of storing things. Extend the addressing idiom that `spill.go` already uses.

3. Do not ship a web tool before the egress check and the untrusted delimiters exist.

4. Do not claim image support for a provider profile that cannot receive images.

5. Do not build a definition and reference graph, or embeddings, in version one. The research in `docs/BM25_SCALING_RESEARCH.md` argues for lexical retrieval first at this scale, and Aider's graph is a substantially larger project.

6. Do not put rendering decisions in this system. The shell renders evidence. It does not define it.

## Open decisions

1. **Which web search backend to use for row `11.1`.** The options are a provider-hosted search tool, a third-party search API with a key, or something self-hosted. The evidence and staging model is independent of the choice, which is why it can be deferred to the start of Phase H.

2. **How much to extract from PDFs and office documents.** Row `11.19` covers images, but uploads already accept PDFs. Whether version one extracts their text or stores them as opaque blobs is unresolved. An honest answer may be to store the file, render page images, and let the model read those where its provider supports images.

3. **The default repository path, and whether a session records imports in session state or in a
   file.** The design proposes `~/.goharness/characters/` and leaves both questions to this plan,
   which settles them in phase J. They are small decisions with a wide blast radius, because moving
   the repository later means moving every converted card with it.

4. **What the memory content budget is, and what happens when it is reached.** A ceiling is agreed.
   Whether the agent is refused, or told to retire something, or simply cannot write, is a decision
   for the start of phase N once the retrieval layer can report what an entry costs.

5. **Where injected instructions are shown.** Whether injected instructions appear per turn in the shell or only in settings is a decision for Shell & Interaction. This system provides the records either way.

---

# 11. Recommended order of implementation

1. **Slice 1** — the evidence substrate and content-addressed uploads
2. **Slice 2** — spill evidence and retrieval over the catalog
3. **Slice 4** — walking instructions
4. **Slice 3** — file mentions
5. **Slice 6** — web fetch and search
6. **Slice 5** — images
7. **Slice 7** — the LSP seam
8. **Slice 8** — deliverables and references

The card series runs last, as a second group, because none of it is needed for the ingestion work
that already has a plan and a queue.

9. **Slice 9** — the repository, the converter seam, and the persona
10. **Slice 10** — cards and lorebooks in the catalog
11. **Slice 11** — the lorebook injection layer
12. **Slice 12** — memory the agent maintains

Within that group the order is a dependency order rather than a value order. The repository comes
first because nothing else can be tested without converted input, and it has no dependency on the
evidence substrate, so it can be built in parallel with phase A. The persona follows because it is
the smallest thing that demonstrates the repository working end to end. Registration in the catalog
waits for the substrate. The injection layer follows, and the memory tools come last because writing
is only safe once reading reports what it costs.

Instruction walking is sequenced before file mentions because it is self-contained and closes a long-standing gap in how context is gathered, without depending on any of the composer work.

---

# 12. What success looks like

This system is working when GoHarness can honestly say the following.

Every piece of knowledge, whether it came from the internet or from the user's own machine, has one record carrying where it came from and how to address it again. Identical bytes are stored once and referenced many times. Retrieval returns provenance, offsets, and a trust value rather than just a path and a score. Internet content is never treated as an instruction. An upload or a mention is staged until something explicitly injects it. Images, diagnostics, spills, and pages are all searchable through one surface. And no capability is presented to the user unless the transport behind it genuinely honours it.

It can also say that a person chooses what the agent's prompt contains and in what order, that a
card imported once serves every workspace, that the original file is kept beside what was derived
from it, and that the agent's own memory is a set of files a person can read, correct, and retire.

That adds up to a knowledge system rather than a collection of ingestion features. It is also the foundation that the archived memory units in the Session Event & Memory System and the typed evidence blocks in the Shell & Interaction System are both waiting on.
