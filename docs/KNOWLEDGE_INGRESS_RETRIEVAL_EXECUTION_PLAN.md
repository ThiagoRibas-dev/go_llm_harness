# 📥 Knowledge Ingress & Retrieval System — Execution Plan

> **Status:** Execution plan
> **System:** Knowledge Ingress & Retrieval System
> **Primary roadmap rows:** `11.1`, `11.4`, `11.16`, `11.19`, `12.5`, `12.6`, `12.19`, `12.28.12`

**Row coverage map** — for each roadmap row this plan claims, which phase owns it and how deep that
coverage actually is. `gap` means the row is claimed but not yet written into the plan body.

| Roadmap row | Phase | Coverage |
|---|---|---|
| `11.1` Web search & fetch | Phase H | **core** — tools, fetch policy, evidence staging; search backend choice scoped in-phase |
| `11.4` @file mentions in composer | Phase E | **core** — resolver + typeahead contract; the dropdown UI itself is Shell & Interaction's |
| `11.16` Parent-directory context walking + overrides | Phase F | **core** — walks to project-root markers, root→cwd ordering, override file, byte budget |
| `11.19` Image paste/drag into composer | Phase G | **core** for ingress/storage; model *visibility* is provider-dependent and gated honestly |
| `12.5` LSP seam | Phase I | **core** but staged — seam + one read-only capability first; largest item in this system |
| `12.6` Tool-output spill to disk | Phase C | **core** — closes the partial status: spills become evidence, searchable and referenceable |
| `12.19` Content-addressed attachments | Phase B | **core** — generalizes the addressing scheme `spill.go` already ships |
| `12.28.12` Deliverables, references, message feedback | Phase A (registration), Phase I (projection) | **partial** — this plan owns the structured record; rendering and feedback routing belong to Shell & Interaction (`12.23` feedback is operator-side) |

This document defines the implementation program for GoHarness's **knowledge ingress and retrieval
substrate**: how external and local knowledge gets in, how it is addressed and stored, how it becomes
visible or invisible to the model, and how it is searched.

The core belief behind this system is:

> every source of knowledge — a web page, a file mention, an image, an upload, a spilled tool result, an
> LSP diagnostic — is the same kind of thing: **an evidence record with provenance, an address, and an
> explicit injection budget.** Ingress surfaces differ; the substrate must not.

---

# 1. System scope

This system owns:

- evidence records and their provenance,
- content addressing for ingested material (`12.19`),
- the evidence catalog and its retrieval surface,
- composer-side ingress: file mentions (`11.4`), images (`11.19`), attachments,
- instruction/context discovery across directories (`11.16`),
- network ingress: web search and fetch (`11.1`),
- code-intelligence ingress: the LSP seam (`12.5`),
- spill integration into the same substrate (`12.6`),
- the structured record behind deliverables and references (`12.28.12`).

It does **not** own:

- the shell surfaces that render evidence blocks — Shell & Interaction (`12.29.9` typed blocks),
- approval/egress policy mechanics — Policy & Hooks Engine (`12.16`, `14.2`),
- the long-term event log and session query projections — Session Event & Memory (`12.2`, `12.18`),
- job scheduling for slow fetches — Task & Background Work (`11.7`),
- transcript/turn persistence — Session Event & Memory.

The boundary rule that matters most:

> **This system decides what knowledge is available and how it is addressed. It does not decide what the
> model is allowed to see.** That is an injection decision, and injection is always explicit, budgeted,
> and provenance-labelled.

---

# 2. Source inputs and what they contribute

## 2.1 GoHarness source already relevant

### `src/bm25.go` — the existing lexical engine

```go
type BM25Document struct { Path string; TermFreqs map[string]int; DocLength int }
type BM25Engine struct { K1, B float64; Documents []BM25Document; DocFreqs map[string]int; TotalDocs int; AvgDocLen float64 }
func (e *BM25Engine) AddDocument(path, content string)
func (e *BM25Engine) Search(query string, limit int) []SearchResult
func (e *BM25Engine) IndexDirectory(rootPath string, ignoreSubDirs []string) error
```

This is the retrieval backbone and it is already dependency-free. Two properties matter for this plan:

1. `Search` returns `SearchResult{Path, Score}` — **path and score only, no provenance, no offsets, no
   snippet**. Every consumer then re-reads the file and truncates by byte (`executeBM25Search` takes
   `content[:400]`). Evidence-aware retrieval needs structured hits instead.
2. `IndexDirectory` walks and indexes in one call. There is **no incremental index, no cache, no
   mtime check** — see §7.4 for why that matters at workspace scale.

### `src/agent.go` — `executeBM25Search` and `LoadLocalInstructions`

`executeBM25Search(a, query, scope, limit)` builds a **fresh engine per call**:

- `scope == "session"` → indexes this session's turn folder (skipping `backups` and
  `compacted_summary_up_to_turn_`),
- otherwise → indexes the whole workspace, or only `TargetScanDirs` when configured, and *always* also
  indexes the session uploads folder,
- returns ranked paths plus 400-char excerpts.

That "always index uploads into the workspace scope" behaviour is the first hint that uploads and
workspace files are already treated as one corpus — the substrate just does not say so explicitly.

`LoadLocalInstructions()` is the current `11.16` implementation, and it is deliberately shallow:

```go
targets := []string{"AGENTS.md", "SKILLS.md", "INSTRUCTIONS.md", "CLAUDE.md"}
// checks <workspace>/<target>, else <binary-dir>/<target>
```

- only the **workspace root** and the **global binary directory** are checked,
- there is **no upward walk**, so a monorepo subdirectory never inherits its parents' instructions,
- discovery is persisted into session meta (`updateSessionPinnedFiles`) and surfaced to the UI through
  `/api/sessions/pinned`,
- precedence is "project root, else global", with no override file and no size budget.

### `src/spill.go` — content addressing already exists here

```go
const toolSpillThresholdChars = 20 * 1024
func hashSpillID(content string) string  // sha256 hex of the content
func spillDataPath(sessionID, id string) string // .goharness/sessions/<sid>/spill/<id>.txt
func saveSpill / loadSpillMeta / readSpill(offset, limit, findText)
```

This is the single most important existing asset for `12.19`: **GoHarness already content-addresses
large tool output by sha256 of the content**, stores data and metadata side by side (`<id>.txt` +
`<id>.json`), and gives the model a paged read path (`read_spill`) with explicit offsets so the full
payload never enters context uninvited.

Content addressing is already the house style. `12.19` is not a new idea — it is the *generalization* of
this pattern from spilled tool output to every ingested artifact.

### `src/web.go` — uploads are filename-addressed

```go
uploadsDir := GetSystemPath(filepath.Join(".goharness", "sessions", activeSessionID, "uploads"))
destPath := filepath.Clean(filepath.Join(uploadsDir, handler.Filename))
if strings.Contains(destPath, ".goharness") && !strings.Contains(destPath, filepath.Join("sessions", activeSessionID, "uploads")) {
    http.Error(w, "Security Exception: Invalid upload destination path", http.StatusForbidden)
}
```

Consequences that `12.19` must fix:

- **the filename is the identity**, so uploading `report.pdf` twice silently overwrites the first, and two
  sessions referencing "the same file" hold two independent copies,
- the traversal guard is a **substring check** on the cleaned path, which is the kind of check that
  survives by luck rather than by construction,
- size is capped at 10 MB with no notion of "too large to ingest" versus "store but do not inject",
- no dedupe, no hash, no provenance record beyond the session folder it landed in.

### `src/artifacts.go` — deliverables are inferred by regex over prose

```go
writeHostPathRE   = regexp.MustCompile(`(?m)^Successfully wrote file to host disk at\s+(.+?)\s*$`)
writeDockerPathRE = regexp.MustCompile(`(?m)^Successfully wrote file inside Docker at\s+(.+?)\s*$`)
patchPathRE       = regexp.MustCompile(`(?m)^Successfully patched file '([^']+)'`)
func inferArtifactsFromToolResult(toolName, result string) []string
```

`12.28.12` currently rests on parsing **English sentences out of tool results**. It works, and it is
fragile: any wording change in a tool message silently empties the deliverables list. The structured fix
is for tools to return artifact paths as data (the tool-result metadata channel already exists via
`tool_message_meta.go`), and for the evidence catalog to be the source of truth.

### Composer side — the ingress surfaces that do not exist yet

`src/web/js/composer.js` currently has **no** `@` trigger, **no** paste handler, and **no** drop handler.
`11.4` and `11.19` are greenfield on the client. What exists is the upload endpoint plus the workspace
tree (`src/workspace_entries.go`, `GenerateWorkspaceEntries`) that a file picker can be built from.

---

## 2.2 Reference source inspection and lessons

> Reference sources inspected for this plan: **OpenAI Codex CLI** (`codex-rs`) — `core/src/agents_md.rs`,
> `core/src/tools/handlers/view_image_spec.rs`, `core/src/web_search.rs`, `app-server/src/fuzzy_file_search.rs`
> — and **Aider** (`aider/repomap.py`). Both are read for design lessons, not for code to copy.

### A. Codex CLI: hierarchical project docs (`agents_md.rs`) → `11.16`

The module doc comment states the algorithm precisely:

1. Determine the project root by **walking upwards** from the working directory until a configured
   `project_root_markers` entry is found (default marker list when unset; **an empty marker list disables
   parent traversal entirely**).
2. Collect every `AGENTS.md` found **from the project root down to the working directory**, inclusive,
   and **concatenate in that order**.
3. **Do not walk past the project root.**

Supporting details worth copying:

- `DEFAULT_AGENTS_MD_FILENAME = "AGENTS.md"` plus `LOCAL_AGENTS_MD_FILENAME = "AGENTS.override.md"` as a
  preferred local override,
- user-level and project-level docs are concatenated with an explicit, greppable separator
  (`"\n\n--- project-doc ---\n\n"`),
- a **byte budget** (`project_doc_max_bytes`) with `data.truncate(remaining)` and a warning
  ("project doc exceeds remaining budget; truncating"),
- a fallback filename list (`project_doc_fallback_filenames`),
- bounded concurrent ancestor probes (`MAX_CONCURRENT_ANCESTOR_PROBES = 256`) rather than unbounded fan-out.

**Lesson for GoHarness:** `11.16` should be a faithful port of these semantics — upward walk to a marker,
root→cwd ordering, override filename, explicit budget, never past the root — not an ad-hoc "also check the
parent folder" patch. GoHarness's existing `AGENTS.md`/`SKILLS.md`/`INSTRUCTIONS.md`/`CLAUDE.md` target
list becomes the equivalent of the fallback filename list.

### B. Codex CLI: `view_image` → `11.19`

```rust
name: VIEW_IMAGE_TOOL_NAME,
description: "View a local image file from the filesystem when visual inspection is needed. Use this for images already available on disk.",
// parameters: { path: string, [detail: "high" | "original"], [environment_id] }
// output: { image_url: <data URL>, detail: <"high" | "original"> }
```

Three lessons:

1. Images are **addressed by local filesystem path** and delivered to the model as a **data URL** — the
   file stays the source of truth on disk, the model gets a rendering of it.
2. There is an explicit **detail** control (`high` = default resized, `original` = exact resolution) rather
   than an implicit resize — image tokens are treated as a budget, exactly like text.
3. Image inspection is a **model-invoked tool**, not an automatic transcript attachment. The model decides
   when vision is worth the tokens.

### C. Codex CLI: web search as typed actions (`web_search.rs`) → `11.1`

```rust
enum WebSearchAction {
    Search { query: Option<String>, queries: Option<Vec<String>> },
    OpenPage { url: Option<String> },
    FindInPage { url: Option<String>, pattern: Option<String> },
    Other,
}
```

Search, open-page, and find-in-page are the three primitives, and every UI/transcript representation is
derived from a **typed action record** (`"'{pattern}' in {url}"`), not from raw page text. Search results
are attributed, not dumped.

**Lesson:** GoHarness should model network knowledge as typed actions over content-addressed page
snapshots. "Opened URL X and found Y" must be reconstructible from the record, long after the page
changed or vanished.

### D. Codex CLI: fuzzy file search with match indices (`fuzzy_file_search.rs`) → `11.4`

- matches carry `score`, `indices`, and a `match_type`,
- results sort by **score desc, then path asc** (deterministic tie-breaking),
- a `MATCH_LIMIT` bounds the result set,
- there is an **incremental session protocol**: `start_fuzzy_file_search_session` → `update_query` →
  `on_update(snapshot)` → `on_complete`, and snapshots are ignored when `snapshot.query != query`
  (stale-response guard).

**Lesson for the `@` trigger:** a typeahead needs a *stream with a stale-query guard*, not a request per
keystroke, and highlighting needs character `indices`. That is the cheap part; the expensive part is what
happens when the user commits a mention (see the staging model in §5).

### E. Aider: ranked structure under a budget (`repomap.py`) → retrieval discipline

- The repo map is a **PageRank over a def/ref graph**, with **personalization** derived from files and
  identifiers already in the chat (`personalize = 100 / len(fnames)`, boosted when a path component
  matches a mentioned identifier).
- Budget comes from the model's **context window**, scaled by `map_mul_no_files` when the chat is empty,
  with a padding reserve (`padding = 4096`).
- Tags are **cached with mtime keys**; `refresh="auto"` re-indexes only what changed.
- `render_tree` emits only **lines of interest** plus surrounding context, cached by
  `(rel_fname, lois, mtime)` — structure, not whole files.
- Graceful degradation: on `RecursionError` it disables the map rather than hanging ("git repo too large?").

**Lessons for GoHarness:**

1. Bundling evidence under a budget is a *ranking* problem, and relevance should be **personalized by what
   is already in play** (mentioned files, recent reads) rather than being a flat BM25 top-N,
2. **cache by mtime** — GoHarness re-walks and re-tokenizes the whole workspace on every `bm25_search`
   call today,
3. send **structure and lines of interest**, not file bodies, when the goal is orientation.

---

# 3. Current baseline and gap analysis

## 3.1 What already exists

| Capability | Where | State |
|---|---|---|
| Lexical ranking (BM25, K1=1.5, B=0.75) | `src/bm25.go` | shipped |
| Agent-facing lexical search over workspace/session | `bm25_search` tool, `executeBM25Search` | shipped |
| Content-addressed large tool output with paged reads | `src/spill.go`, `read_spill` | shipped |
| Session uploads with a traversal guard and 10 MB cap | `POST /api/upload` (`src/web.go`) | shipped |
| Instruction discovery (root + global) with session pinning | `LoadLocalInstructions`, `/api/sessions/pinned` | shipped |
| Workspace tree for UI pickers | `src/workspace_entries.go` | shipped |
| Artifact paths surfaced to the UI | `inferArtifactsFromToolResult` | shipped but regex-based |
| MCP tool schema ingestion | `src/mcp.go` | shipped |

## 3.2 What is missing

1. **No evidence record.** Nothing in the codebase represents "this knowledge came from there, at this
   time, and here is how to address it again." Spills have meta sidecars; uploads have neither hash nor
   provenance; web pages and LSP results do not exist at all.
2. **No content addressing outside spills** (`12.19`): uploads are filename-identity, so re-uploading
   overwrites and cross-session references cannot dedupe.
3. **No retrieval into spilled evidence** (`12.6`): a spilled tool result is readable by id but invisible
   to `bm25_search`, which indexes uploads and workspace files only.
4. **No structured artifacts** (`12.28.12`): deliverables come from regex over prose.
5. **No network ingress** (`11.1`): there are no web tools of any kind — no fetch, no search, and no
   egress policy to gate them.
6. **No parent-directory context** (`11.16`): instructions are root-or-global only.
7. **No composer ingress** (`11.4`, `11.19`): no `@`, no paste, no drop.
8. **No code-intelligence ingress** (`12.5`): no LSP client, no diagnostics/symbol source.
9. **No index caching:** every `bm25_search` call re-walks and re-indexes from scratch, so search cost
   scales with workspace size on every invocation.
10. **No untrusted-content discipline:** nothing today distinguishes "text from the user's own workspace"
    from "text a stranger wrote on the internet". That distinction becomes safety-critical the moment
    `11.1` lands.

---

# 4. Program strategy

Three layers, in dependency order:

**Layer 1 — Evidence substrate.** One record type, one address scheme, one catalog. Everything that is
ingested becomes an evidence record; nothing is special-cased per source. Includes the `12.19`
generalization of the spill addressing scheme.

**Layer 2 — Ingress surfaces.** The things that *produce* evidence: composer mentions and drops, image
paste, instruction walking, web fetch/search, LSP results, spill registration.

**Layer 3 — Retrieval and projection.** Search over the catalog (extending the existing BM25 surface with
provenance and caching), plus the structured records the shell renders as typed blocks and reference
chips.

---

# 5. Core model

## 5.1 Evidence record

```json
{
  "evidence_id": "sha256:9f2c…",
  "kind": "web_page | file_mention | image | upload | spill | lsp_diagnostic | deliverable | instruction",
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

Three fields carry the weight:

- **`trust`** — `untrusted_external` marks anything fetched from the network. Untrusted evidence is
  **never** eligible for instruction-position injection and must be delimited when staged (§5.3). This is
  the prompt-injection boundary, and it is a property of the record rather than of a code path.
- **`evidence_id`** — sha256 of content, matching `hashSpillID`'s existing scheme. Identical bytes are one
  record regardless of how many times they are mentioned; `derived_from` keeps lineage (page snapshot →
  extracted text → summary).
- **`origin` + `trust` together** are what make recall honest later: the Session Event & Memory System's
  `9.3` archived-memory units derive from these records, and "recalled memory is provenance-aware" (that
  plan's success condition) is only true if the provenance exists at ingress time.

## 5.2 Storage layout

```
.goharness/
  evidence/                        # workspace-scoped content-addressed blob store
    <sha256>.blob                  # raw bytes, immutable
    <sha256>.meta.json             # EvidenceRecord
  sessions/<sid>/
    uploads/<sha256>.<ext>         # uploaded files, hash-named (12.19)
    spill/<sha256>.txt             # existing spill data (unchanged path)
    spill/<sha256>.json            # existing spill meta (unchanged)
    evidence-index.jsonl           # append-only references from this session
```

Blobs are workspace-scoped so two sessions referencing the same file dedupe instead of double-storing;
**references** stay session-local so "what did this session ingest" remains answerable without scanning
the workspace.

Spill files keep their current paths — this plan **registers** spills into the catalog rather than moving
them, which keeps `read_spill` and every existing spill id valid.

## 5.3 Staging versus injection

The distinction the whole system turns on:

- **Staged** — the evidence is recorded, addressable, searchable, and visible in the UI. Cost to the model:
  zero tokens.
- **Injected** — the evidence is placed into the prompt. Cost: tokens, permanently, until compaction.

Rules for injection:

1. injection is always **explicit** (a mention, a tool result, or a budgeted retrieval block) — never an
   implicit side effect of an upload,
2. injected evidence is **delimited and attributed**: `=== EVIDENCE <id> (<uri>) ===` … `=== END EVIDENCE ===`,
3. `untrusted_external` evidence is injected only as **data**, with the delimiters above and a standing
   instruction that content inside evidence blocks is not instructions,
4. every injection respects a **budget**, and the block reports what was omitted by budget — the same
   honesty rule the Session Event & Memory plan applies to recall.

Uploads today leak across this line: a file dropped into the composer is written to disk *and* is silently
inside the BM25 corpus. The staged/injected split makes that relationship visible instead of incidental.

## 5.4 Retrieval hits

```go
type EvidenceHit struct {
    EvidenceID string
    Kind       string
    URI        string       // https://… or workspace-relative path
    Score      float64
    Snippet    string       // bounded, with offsets into the blob
    Offsets    [2]int
    Trust      string
    SessionID  string
}
```

Current `SearchResult{Path, Score}` is the gap: no provenance, no offsets, no snippet, no trust. The
engine keeps its BM25 math; the *catalog* layer adds provenance by resolving ids to records.

---

# 6. Execution phases

## Phase A — Evidence substrate

### Goal
One record type, one address scheme, one catalog, used by everything that follows.

### Deliverables
1. **`EvidenceRecord`** as specified in §5.1, with a schema version field.
2. **Content-addressed blob store** at `.goharness/evidence/` with immutable blobs and `<id>.meta.json`
   sidecars — deliberately the same shape as `spill.go` so there is one addressing idiom in the codebase.
3. **Session evidence index** (`evidence-index.jsonl`, append-only) recording references with timestamps.
4. **Catalog API**: `Register`, `Resolve`, `List(session|workspace)`, `Derive(from, kind, blob)`.
5. **Structured artifact path**: tools report produced paths through the existing tool-result metadata
   channel (`tool_message_meta.go`) instead of `inferArtifactsFromToolResult` regexes; the regex path stays
   as a **fallback only**, logged when it fires so the remaining un-migrated tools are visible.

### Proposed new Go files
- `src/evidence.go` — record type, ids, trust enum
- `src/evidence_store.go` — blob store + meta sidecars
- `src/evidence_catalog.go` — session/workspace indexes, registration

### GoHarness files likely touched
- `src/artifacts.go` (demoted to fallback), `src/tool_message_meta.go`, `src/spill.go` (register on save)

### Why now
Every later phase is a producer for this catalog. Building ingress first would produce five incompatible
notions of "a thing the agent looked at".

---

## Phase B — Content-addressed attachments (`12.19`)

### Goal
Generalize the spill addressing scheme to all uploaded/attached material.

### Deliverables
1. **Hash-named upload storage**: `uploads/<sha256>.<ext>`, with the original filename kept in the
   evidence record (`origin.path` keeps the display name; the hash keeps the identity).
2. **Dedupe**: re-uploading identical bytes returns the existing evidence id instead of writing again.
3. **Reference semantics**: multiple sessions in a workspace referencing the same blob share it; deleting a
   session reference does not delete a blob still referenced elsewhere.
4. **Size policy split**: an explicit `ingest_max_inject_bytes` (what may be staged into a prompt) separate
   from `store_max_bytes` (what may be stored at all) — a 50 MB PDF is storable and referenceable while
   only extracted/selected portions are ever injectable.
5. **Correct traversal guard**: validate by resolving the target and checking containment
   (`filepath.Rel` on cleaned absolute paths), not by substring matching on `.goharness`.

### Why now
Dedupe and hash identity are cheapest before more producers exist. Doing this after web fetch and image
paste would mean migrating three formats at once.

---

## Phase C — Spill integration (`12.6`)

### Goal
Close the "Partial" status on `12.6`: spilled output becomes first-class evidence.

### Deliverables
1. **Register on spill**: `saveSpill` also writes an evidence record with `kind: "spill"`,
   `origin.surface: "agent_tool"`, and `notes` carrying the originating tool name and the char counts.
2. **Searchable spills**: `bm25_search` scope gains `evidence` (session spills, uploads, and fetched page
   snapshots) so a spilled `go test ./...` log is retrievable by keyword instead of only by id.
3. **Referenceable spills**: `read_spill` keeps working exactly as today (no id churn), and the evidence
   record links back to it.
4. **Retention rule**: blobs have no automatic deletion in v1, but the catalog reports total size per
   session so the operator surface (`13.10`/`14.10`) can offer cleanup — never silent eviction of
   evidence the agent might cite later.

### Why now
Spills are the densest knowledge source GoHarness already produces and the only one that is currently
invisible to search. This is the cheapest large retrieval win in the system.

---

## Phase D — Retrieval over the catalog

### Goal
Make retrieval provenance-aware and cheap enough to call often.

### Deliverables
1. **`EvidenceHit`** results (§5.4) replacing `SearchResult` for the agent-facing tool: every hit carries
   id, kind, uri, offsets, snippet, trust, session.
2. **Trust-aware ranking**: `untrusted_external` hits are returned **labelled**, never silently merged with
   workspace hits; a single result list that mixes them without labels is a defect.
3. **Index cache with mtime invalidation** (Aider lesson §7.4): persist per-file term frequencies plus
   mtime; re-index only changed files. Today every call re-walks the workspace.
4. **Scope vocabulary**: `workspace` | `session` | `evidence` | `all`, with `session` retaining today's
   turn-folder behaviour.
5. **Budgeted evidence blocks** for prompt assembly: rank, then emit a bounded block that names what was
   omitted by budget.
6. **Personalization hook**: boost documents whose paths or identifiers overlap what is already in play in
   the session (recently read/written files, current mentions) — the Aider personalization idea without
   the graph machinery, since GoHarness has no def/ref graph yet.

### Why now
Every ingress surface added later benefits immediately, and the caching work is orthogonal to those
surfaces.

---

## Phase E — File mentions (`11.4`)

### Goal
`@` in the composer resolves to workspace files and stages them deliberately.

### Deliverables
1. **Server-side resolver endpoint**: query → ranked paths (`score`, matched `indices`, `match_type`),
   deterministic tie-break by path, bounded by a match limit.
2. **Typeahead contract**: incremental query sessions with a stale-query guard, so a slow response for
   `@doc` cannot overwrite results for `@docs` (Codex lesson §7.5).
3. **Staging semantics**: committing a mention **stages** the file (evidence record registered,
   `kind: "file_mention"`, trust `workspace`); injection happens as a delimited evidence block whose size
   is governed by the budget, not by dumping the file.
4. **Honest UI copy**: "staged" versus "sent to the model" must be distinguishable in the composer —
   per the roadmap's "do not surface fake controls" rule, a mention that silently injects 40 KB of text
   would be exactly that kind of lie.

### GoHarness files likely touched
- `src/web.go` (resolver route), `src/web/js/composer.js` (trigger + dropdown), `src/workspace_entries.go`
  (reuse the ignore/collapse rules so the picker matches the visible tree)

---

## Phase F — Parent-directory context walking and overrides (`11.16`)

### Goal
Faithful port of the Codex `agents_md` semantics, adapted to GoHarness's existing target list.

### Deliverables
1. **Upward walk to a project root marker** (configurable list; default `.git`, `go.mod`, `.hg`, and an
   explicit "no parent traversal" empty-list option) — an empty marker list must disable walking, exactly
   as the reference does.
2. **Root→cwd concatenation** of discovered files, in that order, with an explicit separator between
   entries so the assembled block is greppable and testable.
3. **Override filename** (`AGENTS.override.md`-equivalent, e.g. `<name>.override.md`) taking precedence
   over the base file **in the same directory** — not overriding ancestors.
4. **Byte budget** with truncation and a visible note (`project doc exceeds remaining budget; truncating`)
   — never silent truncation.
5. **Do not walk past the project root**, and never read outside the workspace subtree in a way that
   surprises the user (global/binary-dir instructions remain a separate, explicit layer).
6. **Session pinning stays authoritative**: `/api/sessions/pinned` keeps working; auto-discovered entries
   are recorded as evidence (`kind: "instruction"`) so the UI can show what was injected and from where.

### Why now
It is self-contained, high-value in monorepos, and touches `LoadLocalInstructions` in one place — a good
"small honest win" alongside the larger ingress work.

---

## Phase G — Image ingress (`11.19`)

### Goal
Paste/drag/store images as evidence; make model visibility explicit and provider-honest.

### Deliverables
1. **Paste and drop capture** in the composer, including screenshots from the clipboard; files route through
   the Phase B content-addressed store.
2. **Evidence record** with `kind: "image"`, `media_type` from sniffing (not from the filename extension),
   dimensions and byte size recorded.
3. **Model access as a tool**, following the `view_image` reference (§7.2): address by path/id, return a
   data URL, and let the model decide when vision is worth the tokens. An image sitting in the composer is
   **staged**, not auto-injected.
4. **Detail control**: a bounded-resize default plus an explicit original-resolution path, so image tokens
   are a budget rather than a surprise.
5. **Provider honesty gate**: image transport support differs across the providers wired in `src/llm.go`
   (OpenAI-style `image_url` data URLs vs Anthropic/Gemini inline image blocks). The capability must be
   **detected and surfaced per profile**; when a profile cannot carry images, the tool must say so instead
   of failing obscurely or silently dropping the image. No provider gets fake image support.

### Why now
The storage half depends on Phase B; the tool half is independent and small.

---

## Phase H — Web search and fetch (`11.1`)

### Goal
Network ingress with typed actions, content-addressed snapshots, and an egress policy that is not an
afterthought.

### Deliverables
1. **`web_fetch`** — fetch a URL, apply redirect/timeout/size limits, extract readable text, store the raw
   response as an evidence blob (`kind: "web_page"`, `trust: "untrusted_external"`) plus a derived
   extracted-text record (`derived_from` the raw bytes).
2. **`web_search`** — return ranked results (title, URL, snippet) as typed action records; backend choice is
   an in-phase decision (see §10, open decisions). Results are **not** auto-fetched; a page becomes evidence
   only when opened.
3. **Typed actions** (`Search` / `OpenPage` / `FindInPage`) recorded so the transcript can say *"opened
   https://…"* or *"'quota' in https://…"* and remain reconstructible after the page changes (§7.3).
4. **Egress policy seam**: an allow/deny rule set consulted *before* the request, implemented through the
   policy engine's future capability seam (`12.1`, `12.16`) but with a v1 default that is deny-nothing-but-log
   or an explicit allowlist — the choice is the user's, and the UI must show which one is active.
5. **Untrusted delimiters**: fetched content is injected only inside evidence blocks with an explicit
   non-instruction notice (§5.3). This is the single most important safety property of this phase:
   a web page may not be able to rewrite the agent's instructions by containing the phrase "ignore previous
   instructions" in bold.
6. **Slow fetches as jobs**: fetches exceeding an interactive budget should be runnable as Task-system jobs
   (`11.7`) so a hung host does not stall a turn — the seam is defined here, implemented there.

### Why now
Wave 1 per the roadmap, and it is the row with the fewest dependencies. It is also last in this plan's own
ordering because it introduces the most risk surface (untrusted content, egress) and benefits from the
evidence and policy shapes landing first.

---

## Phase I — LSP seam (`12.5`)

### Goal
Code intelligence as a knowledge source, staged behind a capability seam.

### Deliverables
1. **Client transport**: JSON-RPC over stdio to a per-workspace language server process, with lifecycle
   (spawn, `initialize`, shutdown, crash-restart with backoff).
2. **Document sync**: `didOpen`/`didChange` driven by the existing read/write tool paths, so the server's
   view tracks what the agent actually touched rather than a full-project preload.
3. **Capability gating**: only capabilities the server advertises are exposed; absent capabilities are
   absent surfaces (no fake controls).
4. **v1 surface**: `diagnostics` and `document_symbols` as evidence (`kind: "lsp_diagnostic"`,
   trust `workspace`), each with file path, range, severity, and source server.
5. **Timeouts everywhere**: LSP servers are third-party processes that can hang. Every request is bounded
   and its failure is a normal, reported outcome.
6. **Server configuration as data**: which server runs for which language, configured in the workspace
   config and surfaced in settings — not hardcoded.

### Why now (and not earlier)
This is the largest build in the system and the one with the most operational surface (external processes,
per-language configuration). It is genuinely useful but nothing else in this plan waits on it. The seam is
specified now so that later work (`12.1` capability seams, Execution & Review's validation loops) can
target a stable interface instead of inventing a second definition of "diagnostics".

---

# 7. Reference-project implementation lessons to apply directly

## 7.1 From Codex `agents_md` — instructions are hierarchical, ordered, and budgeted
Walk up to a marker, concatenate root→cwd, never past the root, honour an override filename, truncate
against an explicit budget with a visible warning. Apply in Phase F.

## 7.2 From Codex `view_image` — address by path, deliver a rendering, budget the detail
Images live on disk and reach the model as data URLs with an explicit detail level, invoked by the model
rather than auto-attached. Apply in Phase G.

## 7.3 From Codex typed `WebSearchAction` — record actions, not page text
Search / open / find are the primitives, and the transcript is derived from typed records. Apply in
Phase H so claims about web sources stay auditable after the web moves on.

## 7.4 From Aider `repomap` — cache by mtime, rank by what is in play, render structure
The repo map is cached incrementally, personalized by files/identifiers already in the chat, and rendered
as lines of interest rather than whole files. GoHarness applies the caching and personalization parts to
its BM25 index in Phase D; the def/ref graph itself is out of scope (it is a much larger project, and
`docs/BM25_SCALING_RESEARCH.md` argues lexical-first for the scale GoHarness targets).

## 7.5 From Codex fuzzy file search — stream matches with a stale-query guard
Typeahead is a session protocol with snapshot invalidation, scored matches, and match indices for
highlighting. Apply in Phase E.

## 7.6 From GoHarness's own `spill.go` — content addressing is already the house style
`12.19` generalizes a pattern the codebase already proved, which also means the migration story is
"extend the existing idiom", not "introduce a new storage era".

---

# 8. Concrete GoHarness implementation slices

## Slice 1 — Evidence substrate and content-addressed uploads
### Add
- `EvidenceRecord` + blob store + session index
- hash-named uploads with dedupe
- correct containment-based path validation
- structured artifact reporting from write/patch tools, regex demoted to logged fallback
### Success criterion
Uploading the same file twice in two sessions produces **one** blob, two references, and the deliverables
list no longer depends on tool-result wording.

## Slice 2 — Spill evidence and catalog retrieval
### Add
- spill registration on save
- `evidence` search scope
- `EvidenceHit` results with provenance, trust labels and offsets
- mtime-invalidated index cache
### Success criterion
A keyword buried in a 300 KB spilled test log is findable by `bm25_search`, returns the spill's evidence id
and offsets, and a second identical search does not re-walk the workspace.

## Slice 3 — File mentions
### Add
- resolver endpoint with scored matches + indices
- composer `@` trigger with stale-query-guarded typeahead
- staged-vs-sent distinction in the composer
### Success criterion
Typing `@` lists files matching the visible workspace tree's ignore rules; committing a mention stages it
and the composer shows it as staged until it is actually injected.

## Slice 4 — Instruction walking
### Add
- marker-based upward walk, root→cwd concatenation, override file, byte budget
- discovered instructions registered as evidence
### Success criterion
In a workspace nested three levels inside a repo root, instructions from every level are injected in
root→cwd order, the override file wins in its own directory, and an over-budget file is truncated with a
visible note.

## Slice 5 — Image ingress
### Add
- paste/drop capture, content-addressed storage, sniffed media type
- model-invoked image tool with detail control
- per-profile capability detection with an honest unsupported message
### Success criterion
A pasted screenshot is stored once, visible as staged evidence, and readable by the model on profiles that
support images — and clearly refused, not silently dropped, on profiles that do not.

## Slice 6 — Web fetch and search
### Add
- fetch with limits, raw snapshot + extracted text as linked evidence
- search returning typed results
- egress rule check before request, with the active mode visible to the user
- untrusted evidence delimiters in injection
### Success criterion
A fetched page is retrievable later by keyword, still attributed to its URL, and cannot be injected without
its `untrusted_external` delimiting and notice.

## Slice 7 — LSP seam
### Add
- stdio JSON-RPC client with lifecycle and restart
- diagnostics + symbols as evidence
- per-language server configuration
### Success criterion
Opening a file with a real error yields a diagnostic evidence record with file, range and severity — and a
hung server degrades to a reported timeout instead of stalling the turn.

## Slice 8 — Deliverables and references projection
### Add
- deliverables derived from the evidence catalog rather than prose
- reference chips pointing at evidence ids
- feedback routing kept out of model context (operator-side, per `12.23`)
### Success criterion
The deliverables list for a turn is derived from structured records, and every entry resolves to an
evidence id whose origin can be shown to the user.

---

# 9. Testing plan

## 9.1 Substrate tests
- identical bytes registered twice produce one blob and two references
- blob immutability: a second write with the same id does not alter the first
- containment validation rejects `../` traversal and absolute paths outside the session root
- evidence ids survive session restart (resolve by id after reload)

## 9.2 Retrieval tests
- hits carry provenance: id, kind, uri, offsets, snippet, trust
- `untrusted_external` hits are labelled in the result payload
- search results are stable across repeated calls (deterministic tie-break)
- index cache invalidation: a modified file is re-indexed, an untouched file is not re-tokenized
- budgeted evidence blocks report omitted-by-budget counts

## 9.3 Ingress tests
- file mention resolver: ranking, indices, stale-query rejection
- instruction walking: marker stop, root→cwd order, override precedence, budget truncation, empty marker
  list disables parent traversal
- image sniffing rejects a `.png` that is not an image, and records true dimensions
- upload dedupe across sessions in one workspace

## 9.4 Network tests (no live network in unit tests)
- fetch limits: redirect cap, timeout, size cap, non-HTML content types
- egress rule check runs **before** any request is issued (assert no dial on deny)
- fetched evidence is `untrusted_external` and is injected only inside evidence delimiters
- search results are recorded as typed actions and reconstructible from the record alone

## 9.5 LSP tests
- server crash mid-request produces a reported failure, not a hang
- capabilities are gated on server advertisement
- diagnostics map to evidence records with correct ranges

---

# 10. Risks, anti-goals, and open decisions

## Risks
1. **Evidence sprawl** — every source registering records could bloat `.goharness/evidence/` without bound.
   Mitigation: per-session size reporting in the catalog and operator cleanup tooling; never silent eviction.
2. **Prompt injection via fetched content** — mitigated structurally by the `trust` field and delimiters,
   but only if every injection path respects them. This needs a test suite, not good intentions.
3. **Fake capability surfaces** — LSP servers, image transports, and search backends vary. A control that
   is not truly honoured is worse than a missing control; provider/server capability must be detected.
4. **Index churn** — mtime-based caching can thrash on a workspace touched by builds. Needs a bounded
   re-index budget per call.
5. **Two retrieval worlds** — Session Event & Memory's `12.18` session query and this system's evidence
   search could diverge into two engines. They should share `BM25Engine` and the provenance vocabulary.

## Anti-goals
1. Do not auto-inject uploaded or fetched content. Ingress is not injection.
2. Do not introduce a second storage era: extend the `spill.go` addressing idiom.
3. Do not ship a web tool before the egress check and untrusted delimiters exist.
4. Do not claim image support for a provider profile that cannot receive images.
5. Do not build a def/ref graph or embeddings in v1 — `docs/BM25_SCALING_RESEARCH.md` argues lexical-first
   at the scale GoHarness targets, and Aider's graph is a much larger project.
6. Do not put UI rendering decisions in this system; the shell renders evidence, it does not define it.

## Open decisions
1. **Web search backend** for `11.1` — provider-hosted search tool versus an API-key search service versus
   a self-hosted option. The evidence and staging model is backend-independent; the choice is a config
   surface and should be made when Phase H starts, not now.
2. **Extraction quality for PDFs and office documents** — `11.19` covers images, but uploads already accept
   PDFs. Whether v1 extracts text from PDFs or stores them as opaque blobs is unresolved; the honest answer
   may be "store, render page images, and let the model read images where supported".
3. **Where instruction evidence is shown** — whether injected instructions are surfaced per turn in the
   shell or only in settings. This is a Shell & Interaction decision that consumes this system's records.

---

# 11. Recommended first implementation order

1. **Slice 1** — evidence substrate + content-addressed uploads
2. **Slice 2** — spill evidence + catalog retrieval
3. **Slice 4** — instruction walking (small, self-contained, immediately useful)
4. **Slice 3** — file mentions
5. **Slice 6** — web fetch and search
6. **Slice 5** — image ingress
7. **Slice 7** — LSP seam
8. **Slice 8** — deliverables/references projection

Instruction walking is sequenced before file mentions because it is self-contained and closes a
long-standing context gap without depending on the composer work.

---

# 12. Success definition

This system is succeeding when GoHarness can truthfully say:

- every piece of external or local knowledge has **one record with provenance and an address**,
- identical bytes are stored once and referenced many times,
- retrieval returns **provenance, offsets and trust**, not just a path and a score,
- internet content is **never** an implicit instruction,
- an upload or a mention is **staged** until something explicitly injects it,
- images, diagnostics, spills and pages are all searchable through one surface,
- and no capability is presented to the user unless the underlying transport actually honours it.

That is a knowledge system rather than a pile of ingestion features — and it is the foundation the
Session Event & Memory System's archived-memory units (`9.3`) and the Shell's typed evidence blocks
(`12.29.9`) are both waiting on.
