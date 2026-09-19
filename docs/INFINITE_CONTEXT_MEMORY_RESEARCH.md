# 🔬 Research Report: `Lumi-node/infinite-context` and What It Means for GoHarness Memory

This document evaluates [`Lumi-node/infinite-context`](https://github.com/Lumi-node/infinite-context) as a reference for improving **GoHarness's memory model**.

The goal is **not** to copy that project wholesale.

The goal is to determine:

1. what the project is actually doing at the source-code level,
2. what role **beam search** really plays in its memory architecture,
3. what parts are useful for GoHarness,
4. what parts should **not** be adopted,
5. and how to translate the useful ideas into a **pure-Go, single-binary, local-first** design.

---

## 1. Executive summary

`infinite-context` is best understood as a **hierarchical retrieval memory layer** for local LLMs.

It is **not** true infinite context.

Its practical flow is:

- embed text chunks,
- place them in a fixed hierarchy,
- use **beam search** to route queries through that hierarchy,
- retrieve a small set of relevant leaf chunks,
- inject those chunks back into the model's normal context window.

That makes it relevant to GoHarness because GoHarness already has:

- session persistence,
- compaction,
- archived turn folders,
- pinned context,
- uploaded documents,
- and BM25-based lexical retrieval.

The useful lesson is **not** the marketing phrase “infinite context”.

The useful lesson is:

> **cold archived memory should be organized hierarchically and searched selectively under a strict prompt budget**.

For GoHarness, the best translation is likely:

- **BM25-first hierarchical archived-memory retrieval**,
- with optional embeddings later,
- integrated into session compaction, archived turn storage, and future event-log-backed memory projections.

---

## 2. Scope of evaluation

This report is based on both the project README and direct inspection of the repository source tree, especially:

- `src/adapters/index/hat.rs`
- `src/adapters/index/mod.rs`
- `src/adapters/index/learnable_routing.rs`
- `src/adapters/python.rs`
- `python/infinite_context/context.py`

The emphasis here is on the **implemented mechanics**, not headline claims.

---

## 3. What the project actually implements

## 3.1 Core structure: a fixed hierarchy, not a general memory graph

The project's central data structure is a **Hierarchical Attention Tree (HAT)**.

At the source level, the hierarchy is hardcoded as:

- `Global`
- `Session`
- `Document`
- `Chunk`

This is defined in `src/adapters/index/hat.rs` via `ContainerLevel`.

Each container stores:

- an ID,
- a hierarchy level,
- a centroid vector,
- a timestamp,
- child IDs,
- descendant count,
- optional accumulated sums,
- optional subspace metadata.

So the memory model is not “flat vector search”. It is:

> **a tree of centroids over explicitly grouped memory units**.

That matters because the retrieval algorithm is not comparing the query against every chunk first. It is comparing the query against **container centroids** and using that to decide where to descend.

---

## 3.2 Insertion model: the hierarchy is partly semantic, partly batching

Insertion happens in `HatIndex::add(...)`.

At a high level:

- a chunk is created,
- it is placed in the current active document,
- if a document exceeds `max_children`, a new document starts,
- if a session exceeds `max_children` documents, a new session starts.

Important source-level detail:

- `max_children` defaults to `50`
- `beam_width` defaults to `3`

This means the hierarchy is **not automatically discovering semantic clusters** in the strong sense.

Unless the caller explicitly creates boundaries with things like:

- `new_session()`
- `new_document()`

…the structure is largely formed by **size thresholds**.

So the quality of retrieval depends heavily on whether the caller creates meaningful boundaries.

This is an important caution for GoHarness:

> a hierarchy only helps if its boundaries mean something operationally useful.

For GoHarness, that argues strongly for using boundaries like:

- session,
- branch,
- compaction epoch,
- uploaded document,
- deliverable family,
- maybe inferred topic cluster,

rather than only arbitrary chunk-count rollover.

---

## 3.3 Query scoring: centroid distance with optional extras

Routing in HAT uses `combined_distance(...)`.

By default, the score is mostly based on:

- similarity between the query embedding and the container centroid.

Optional additions exist for:

- temporal weighting,
- subspace-aware routing,
- learnable routing weights.

But the important implementation fact is:

- those advanced features are **optional**,
- and several are **off by default**.

So the practical default algorithm is much simpler than the README's broader framing:

> **route down the tree using centroid similarity, optionally mixed with time distance**.

---

## 4. How beam search actually factors into the design

This is the key part.

## 4.1 Beam search is the branch-pruning algorithm

The core traversal is `search_tree(...)` in `src/adapters/index/hat.rs`.

What it does:

1. start at a container, usually the root,
2. inspect the current frontier of containers,
3. if a container is a leaf chunk, add it to candidate results,
4. otherwise score its children,
5. sort children by distance,
6. keep only the top `beam_width` children,
7. repeat for the next level,
8. sort collected leaves and return top `k`.

So beam search here is not decoder-style token beam search.

It is:

> **best-first pruning of a hierarchical retrieval tree**.

In practical terms:

- many candidate sessions may exist,
- only the best few survive,
- then many candidate documents may exist,
- only the best few survive,
- finally chunk leaves under those branches are considered.

This is what gives HAT its speed.

---

## 4.2 The beam width is configurable, but the defaults are narrow

At the Rust level:

- `HatConfig.beam_width` exists,
- default is `3`,
- the actual traversal uses `max(config.beam_width, k)`.

So if you ask for `k=10`, it will effectively use beam width `10`.

This is a sensible practical choice: if the final answer wants 10 chunks, the routing beam should not be narrower than 10.

Still, the algorithm remains approximate:

- if the correct chunk sits under a poorly scoring parent centroid,
- the beam may prune the right branch early,
- and the correct chunk is never even considered.

That is the classic beam-search tradeoff:

- more speed,
- less exhaustive search.

---

## 4.3 Beam search here does **not** make context infinite

This is the most important conceptual clarification.

Beam search does **not** enlarge the model's actual context window.

It only makes retrieval over a large external memory store tractable.

The actual model still sees:

- its normal prompt,
- plus a bounded set of retrieved chunks.

So beam search is part of the **memory selection layer**, not the model's native reasoning capacity.

That distinction is critical for GoHarness product honesty.

If we adopt ideas from this space, we should describe them as:

- **archived-memory retrieval**,
- **selective reinjection**,
- **hierarchical recall**,

not “infinite context”.

---

## 5. Source-level weaknesses and caveats

This is where direct code inspection matters.

## 5.1 The project itself labels HAT as approximate

In `src/adapters/index/mod.rs`, the project distinguishes:

- `FlatIndex` — brute force, exact
- `HatIndex` — hierarchical tree, approximate

That is the correct description.

So any headline claim implying exactness should be read cautiously.

Beam search plus centroid routing is, by nature, an approximation strategy.

---

## 5.2 High-level Python API appears to ignore `beam_width`

In `python/infinite_context/context.py`, the convenience API accepts:

- `beam_width: int = 10`

But in the Rust/Python bridge (`src/adapters/python.rs`), the high-level `PyInfiniteContext::new(...)` path simply creates:

- `RustHatIndex::cosine(dimensionality)`

with default config.

So the convenience-layer `beam_width` argument appears not to be wired through.

That is a concrete implementation mismatch between API surface and actual runtime behavior.

For GoHarness this is a cautionary lesson:

> never surface a memory/routing setting in UI or config unless the runtime truly honors it.

---

## 5.3 The high-level text retrieval mapping appears incorrect

In `src/adapters/python.rs`, `PyInfiniteContext.retrieve(...)`:

- calls `index.near(...)` and gets back `SearchResult { id, score }`
- but then maps those results back to text by **enumeration index** instead of by result ID.

So even if the tree retrieval finds the correct chunk ID, the returned text association can be wrong.

That is a serious correctness issue in the high-level product wrapper.

For GoHarness, this is another important lesson:

> retrieval IDs, memory chunks, and user-visible text/artifacts must be bound by a trustworthy storage contract, not positional coincidence.

---

## 5.4 Segmentation quality is the real hidden variable

Because the tree is fixed-depth and often built with rollover thresholds, the real quality driver is:

- how documents/chunks are grouped,
- how boundaries are chosen,
- how representative centroids are,
- and how query embeddings relate to those centroids.

So the beam-search story is only as good as the segmentation story underneath it.

That is highly relevant to GoHarness, because GoHarness has much richer structural signals than this project already:

- user/assistant/tool turn roles,
- session and branch identities,
- compaction boundaries,
- uploaded files,
- workflow traces,
- deliverable files,
- workspace path provenance.

We should exploit those signals instead of pretending one generic hierarchy solves everything.

---

## 6. What GoHarness should learn from this project

## 6.1 Adopt the retrieval model idea, not the product slogan

The good idea is:

- maintain large cold memory externally,
- retrieve selectively,
- inject under budget.

That is absolutely useful for GoHarness.

The bad idea is:

- talking about “unlimited memory” as if retrieval were equivalent to direct reasoning over all past context.

GoHarness should stay explicit:

- the model still has a bounded prompt,
- archived memory must compete for budget,
- retrieval can fail,
- summaries can omit things,
- and chunk boundaries matter.

---

## 6.2 Adopt hierarchy over archived memory

This is the strongest transferable idea.

GoHarness already has three rough memory temperature zones:

### Hot context
- current system prompt
- recent raw turns
- staged files
- pinned files
- immediate workflow outputs

### Warm memory
- compacted session summary
- recent artifacts/deliverables
- uploaded docs currently in play

### Cold memory
- archived raw turns
- old compacted summaries
- old branch histories
- maybe old workspaces/sessions later

What `infinite-context` suggests is:

> cold memory should not be a flat pile.

It should be queryable through a hierarchy like:

- workspace
- session / branch
- compaction epoch / topic block / artifact family
- leaf chunks

That is where beam-style or coarse-to-fine retrieval becomes useful.

---

## 6.3 Use GoHarness-native structure instead of generic session/document/chunk only

A GoHarness hierarchy should be richer than theirs.

Proposed first-pass hierarchy:

- **Workspace**
  - **Session / Branch**
    - **Epoch**
      - raw turn cluster
      - compacted summary
      - uploaded docs
      - deliverable group

Where an **Epoch** could mean:

- pre-compaction block,
- post-compaction segment,
- branch fork boundary,
- or explicit logical ranges in the future event log.

That gives retrieval branches that reflect actual agent work structure rather than only arbitrary chunk counts.

---

## 6.4 Prefer BM25-first before embeddings

This is where GoHarness should be opinionated.

The strongest near-term implementation path is **not** to import a sentence-transformer stack.

It is to do:

### v1
- BM25-first hierarchical retrieval over archived memory
- lexical filters + metadata filters
- recency weighting
- selective reinjection

### v2 optional
- embedding/rerank path behind an optional feature boundary

Why BM25-first is the right fit:

- aligns with GoHarness's pure-Go runtime values
- easier to debug
- deterministic and inspectable
- avoids model/download/runtime sprawl
- already consistent with our existing retrieval direction
- matches broader evidence that lexical retrieval remains very strong for developer/document corpora at scale

Embeddings may still become useful later, but they should not be required for a good GoHarness archived-memory v1.

---

## 7. Proposed GoHarness memory architecture influenced by this research

## 7.1 Storage model

### Keep current durable artifacts
Do not throw away:

- per-turn JSON files
- compacted summary files
- session metadata
- backups / branch structure

### Add a derived archived-memory index layer
Build a GoHarness-native index over archived memory units.

Suggested unit:

```text
MemoryUnit
- memory_id
- workspace_id
- session_id
- branch_id
- epoch_id
- source_kind: raw_turn | compacted_summary | upload | deliverable | note
- role_mix
- turn_start
- turn_end
- created_at
- path_refs[]
- lexical_text
- summary_text
- optional embedding
- provenance
```

This should be derived from canonical storage, not replace it.

That preserves transparency and rollback/debug friendliness.

---

## 7.2 Retrieval model

### Stage 1: coarse retrieval
Search/select likely relevant high-level groups:

- workspace
- session/branch
- epoch/topic block

Scoring inputs:

- lexical match
- recency
- role/source match
- path/artifact match
- user-staged hints

### Stage 2: leaf retrieval
Within selected groups, retrieve:

- raw turn chunks
- summary chunks
- artifact snippets
- upload snippets

### Stage 3: budgeted prompt assembly
Reinject only the best candidates under a strict token/char budget.

That is the actual reusable pattern from `infinite-context`.

---

## 7.3 Optional beam-search analogue for GoHarness

GoHarness does not need to literally implement their Rust HAT tree first.

A Go-native analogue could be:

- scored frontier expansion over memory groups,
- bounded branching factor,
- prune at each hierarchy level,
- then rank leaves.

That could be implemented over plain Go structs and our on-disk/session metadata.

In other words:

- adopt the **strategy**,
- not necessarily the exact data structure.

---

## 7.4 UI implications

If hierarchical archived-memory retrieval lands, the UI should expose it truthfully:

- what memory sources were searched
- what was retrieved
- what was excluded due to budget
- whether the result came from:
  - recent turns
  - compacted summary
  - archived raw turns
  - uploaded docs
  - deliverables

This should appear as:

- a retrieval/source strip in the composer or details panel,
- not as invisible “magic memory”.

---

## 8. Recommended implementation sequence for GoHarness

## Phase A — formalize the memory tiers
Write the storage and retrieval spec around:

- hot / warm / cold memory,
- archived-memory unit schema,
- provenance and reinjection policy.

## Phase B — archived chunk indexing
When compaction occurs:

- preserve the summary,
- also derive chunkable archived-memory units from the compacted/evicted turns,
- index them with BM25 + metadata.

## Phase C — cold-memory retrieval API
Add a retrieval path that can answer:

- retrieve top epochs
- retrieve top chunks within epochs
- return provenance and scores

## Phase D — prompt budgeter
Build a budgeted reinjection planner that decides how much space goes to:

- recent raw turns
- compaction summary
- archived retrieved chunks
- staged files
- uploads

## Phase E — optional embedding/rerank layer
Only after BM25-first cold memory works well, consider optional:

- local embedding support
- reranking
- hybrid lexical + embedding scoring

---

## 9. What GoHarness should explicitly avoid copying

1. **Do not adopt “infinite context” language.**
   It overstates what retrieval memory actually is.

2. **Do not depend on Python/Rust-heavy runtime layers for core memory.**
   That conflicts with GoHarness runtime principles.

3. **Do not surface fake knobs.**
   Their apparent `beam_width` high-level mismatch is a warning here.

4. **Do not rely on positional mappings between retrieval results and payload text.**
   Every memory unit needs durable IDs and provenance.

5. **Do not replace transparent archived state with a black-box memory blob.**
   GoHarness's inspectability is a strength.

---

## 10. Bottom-line recommendation

`Lumi-node/infinite-context` is a **worthwhile reference** for GoHarness memory design, but only in a constrained way.

### Worth borrowing
- hierarchical cold-memory organization
- coarse-to-fine retrieval
- branch pruning / beam-style frontier reduction
- budgeted reinjection into a normal context window
- explicit separation between memory search and answer generation

### Not worth borrowing directly
- the product framing
- the current runtime stack
- the current high-level wrapper correctness model
- any suggestion that retrieval recall equals answer accuracy

### Best GoHarness translation

> **Hierarchical archived-memory retrieval, BM25-first, optional embeddings later, with explicit provenance and prompt budgeting.**

That would materially improve GoHarness's memory model without violating its core constraints.

---

## 11. Concrete roadmap consequence

This research strengthens the case for three roadmap directions:

1. **Canonical event/session storage and projections**
   - because archived memory retrieval wants clean provenance and range semantics

2. **Hierarchical memory subsystem**
   - because summaries alone are too lossy as the only cold-memory layer

3. **Cross-session memory retrieval**
   - because the right abstraction is not “remember everything forever in prompt”, but “search prior structured memory under budget”

That means this project should influence GoHarness as a **design reference for future hierarchical cold-memory retrieval**, not as a direct implementation dependency.
