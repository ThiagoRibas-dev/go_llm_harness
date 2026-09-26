# Character Cards and Lorebooks in GoHarness

A design proposal. This document works out how the Character Card V3 work could be used by the
harness. It is a proposal rather than a plan, so it records options and open decisions instead of
phases and slices.

## 1. The short version

Three separate ideas were raised, and they attach to three different parts of the harness.

1. A character card as the agent's persona.
2. A directory of cards as an information base the agent can read.
3. A lorebook as a retrieval layer, and as a memory the agent can maintain itself.

The third is the strongest, and it is the one worth building first. A lorebook is a written-down,
portable format for exactly the problem the harness has: deciding which pieces of context to
inject, at what depth, in what order, and what to drop when the window is full. It solves a
problem the harness would otherwise solve ad hoc, and it does so in a format a human can open and
edit in tools that already exist.

The proposal deliberately avoids a bespoke roleplay mode. Nothing here assumes conversation for
its own sake. The persona is one prompt element, the cards are documents, and the lorebook is a
retrieval index.

## 2. What already exists

The work in `docs/character-cards/` is a 1,215 line implementation guide and a reference
implementation of 178 passing tests. Between them they cover reading V1, V2, and V3 cards from
PNG, APNG, JSON, and CHARX containers, the lorebook matching algorithm, the decorator layer, and
the curly-braced syntaxes. They were exercised against seventy real cards across three corpora,
which is where most of the guide's practical advice comes from.

Two properties of that work matter for what follows.

**It is complete enough to be authoritative.** The matching algorithm, the budget rules, and the
decorator semantics are implemented and tested, not described from memory. The guide's section 10
is the specification the harness would otherwise have to derive.

**It is in Python, and the harness is in Go with no Python dependency today.** That is a real
obstacle to any design that calls the reader at runtime. Section 3 resolves it, and the resolution
is the same thing that was asked for in the first place.

## 3. The shape that avoids the language problem

A card is an input. It is read once, when it is added, and it does not change while the agent
runs. That means the parsing does not have to happen at runtime at all.

So the design is an offline conversion step, and the harness consumes its output:

```
character.png  --(converter, offline)-->  characters/aria/
                                            card.json       the normalised V3 card
                                            character.md    the readable rendering
                                            lorebook.json   a standalone lorebook_v3 document
```

The converter is the existing reference implementation. The harness never opens a PNG, never
decodes base64, and never needs Python. What it needs is a small reader for the `lorebook_v3`
JSON shape, which is a plain object with seventeen known fields per entry and an algorithm the
guide already states.

This is not a compromise. It is the more useful arrangement, for three reasons.

**The converted directory is inspectable.** A person can read `character.md`, edit
`lorebook.json` in any text editor, and put the directory under version control. Memory that a
human can review is memory a human can correct.

**The conversion happens once instead of every run.** Decoding a PNG chunk per session would be
work repeated for no benefit.

**It matches what was proposed.** The request was to save cards as JSON or Markdown into a
directory and have the agent use that as a base. This is that, with the reader as the thing that
produces the directory.

## 4. The four pieces, one at a time

### 4.1 A card as the agent's persona

Today the system prompt is assembled in `src/agent_run.go` around line 44 from four parts: a
hardcoded sentence describing the agent, `buildEnvironmentSystemPrompt()`, the local instructions
from `LoadLocalInstructions()`, and the workspace tree. There is no persona concept and no
configuration for one.

A persona would be a fifth part, resolved from configuration and placed after the hardcoded base
and before the environment block. The card fields map onto it directly:

| Card field | Where it would go | Notes |
| --- | --- | --- |
| `description`, `personality` | The persona block | The core of the role. |
| `scenario` | The persona block | Useful as standing context. |
| `system_prompt` | Merged into the persona block | The card's own instructions. |
| `post_history_instructions` | After the conversation, if used at all | The harness has no equivalent slot yet. |
| `mes_example` | Optional worked-example block | The one field with a genuine non-roleplay reading. |
| `creator_notes` | Nowhere | The specification says it is for the user. It must not be injected. |
| `alternate_greetings`, `first_mes` | Nowhere in an agent context | Read, preserved, unused. |
| `character_book` | The retrieval layer, section 4.3 | Not part of the persona text. |

This is a prompt-composition change, not a mode. No other behaviour depends on a persona being
present. An agent with no persona configured behaves exactly as it does today.

### 4.2 A directory of cards as an information base

This is the least novel piece, because plan #2 already provides the mechanism. That plan ingests
every source into one kind of record, an `EvidenceRecord`, with a content hash, an origin, and a
trust value. Its `kind` enumeration currently reads:

```
web_page | file_mention | image | upload | spill | lsp_diagnostic | deliverable | instruction
```

Two values would be added, `character_card` and `lorebook`. Nothing else about the plan changes.
A directory of cards becomes a directory of evidence, searchable by the existing retrieval path,
with provenance pointing back at the source file. The Markdown rendering is what the agent reads,
which means the HTML conversion, macro handling, and code fencing that the renderer already does
are doing real work inside the harness rather than only in a batch tool.

### 4.3 A lorebook as a second retrieval layer

The harness has one retrieval engine today, the BM25 index in `src/bm25.go`, which answers "find
the documents most similar to this query". There is no semantic search and no vector store
anywhere in `src/`. (`src/embed.go` is unrelated despite the name: it is a dormant helper for
unpacking a portable Python runtime, and nothing calls it.)

A lorebook answers a different question: "given the last few turns, which specific facts should be
in the prompt right now". It is curated rather than searched, keyed rather than ranked, and it
carries its own budget. The two are complementary, and the mapping onto the harness is close to
one-to-one:

| Lorebook concept | Harness equivalent |
| --- | --- |
| `messages`, the scan window | The recent turns, or the recent tool calls |
| `scan_depth` | How many turns back to look for keys |
| `keys` | Trigger words matched against that window |
| `secondary_keys` with `selective` | Two-part triggers, such as a topic plus a specific module |
| `constant` | Entries that are always present, such as project conventions |
| `priority`, `insertion_order` | Ordering, and the order to drop entries in |
| `token_budget` | How much of the window memory may occupy |
| `recursive_scanning` | Multi-hop expansion: an injected fact triggering further lookups |
| `enabled` | Retiring a fact without deleting it |
| `@@depth`, `@@position` | System block against recent history |
| `@@ignore_on_max_context` | Already means "drop this when the window is full" |
| `extensions` | Where harness-specific metadata belongs, which the specification reserves for it |

Four of those rows are worth calling out, because they are things the harness does not have today
and would otherwise have to invent.

**`token_budget` with `priority` and `insertion_order` is a complete answer to a question the
harness has not answered yet.** When the context fills, something must be dropped. The lorebook
format says what to drop first, and in what order to put back what survives. That is a rule the
harness can adopt rather than design.

**`recursive_scanning` is multi-hop retrieval, expressed in one boolean.** An injected entry whose
content matches another entry's keys can pull that entry in too, bounded by a recursion limit. The
harness would call this query expansion and build it from scratch.

**`constant` is the "always in context" case.** Project conventions, build commands, and house
style are currently either in the instructions file or not present. The lorebook format makes
them entries with a priority and a budget, which means they can be dropped under pressure like
anything else.

**An entry is small.** Content is a paragraph, not a document. That is the right granularity for
facts, and it is the opposite of the document-grained evidence records the knowledge plan
describes. The two levels coexist: evidence holds documents, a lorebook holds the facts extracted
from them.

### 4.4 A lorebook as memory the agent maintains

This is the idea with the most reach, and the one where the existing plan's vocabulary is most
useful.

Three things are already shipped and worth knowing about. `src/spill.go` writes large tool
results to disk under a content-addressed name, which is the precedent the knowledge plan
generalises rather than replaces. Compaction summaries are written per session. And
`LoadLocalInstructions` in `src/agent.go` already reads `AGENTS.md`, `SKILLS.md`,
`INSTRUCTIONS.md`, and `CLAUDE.md` from the workspace, including session-pinned files, so there
is an established place where standing text enters the prompt.

What the harness does not have yet is the evidence substrate itself. Plan #2 specifies it; it is
not built. And nothing today is a writable, cross-session store of small facts that the agent
curates. A lorebook is exactly that shape, which is why it is worth considering before the
substrate is built rather than after.

The agent-facing surface would be small, and it follows the existing tool pattern rather than
inventing one. It is a read and write surface, and every call is recorded.

| Tool | Effect |
| --- | --- |
| `memory_write` | Add or replace an entry: keys, content, priority, and optional metadata. |
| `memory_read` | Return one entry in full, by identifier. |
| `memory_list` | List entries with identifiers, keys, and status, without content. |
| `memory_disable` | Set `enabled` false, which retires a fact without deleting it. |

Injection is not a tool. Matching runs during prompt assembly, the same way the instructions file
is not fetched by a tool. The four tools above exist so the agent can curate what gets injected,
and they belong to a mode rather than to every turn, for the reason section 5 sets out.

The important design constraint is that a lorebook entry must not become a second store. The
knowledge plan already has a content-addressed store at `.goharness/evidence/`, and the lorebook
should be a view over it rather than a parallel copy. The specification reserves the `extensions`
object on every entry for exactly this, and real cards use it for exactly this, so each entry can
carry a pointer to the evidence it came from:

```json
{
  "keys": ["deadline", "Q3 release"],
  "content": "The Q3 release deadline is 14 October.",
  "extensions": {
    "goharness_evidence_id": "sha256:...",
    "goharness_written_by": "session/2026-09-26-4a1f",
    "goharness_trust": "agent_written"
  }
}
```

That keeps the property the knowledge plan asks for, that there is one place where ingested things
live, while still letting the harness use a format that other tools understand.

## 5. Mode-scoped tool sets

The memory tools are where the tool list would start to strain. The linear chat node already
exposes six built-in tools plus MCP, and four memory tools on every turn makes eleven, none of
which the model needs while it is editing a file. The answer is not to hide them but to group
them, and the grouping turns out to be useful well beyond this feature.

### 5.1 What already exists

Tool scoping is already per node. An `llm` node in `workflows.json` carries `tools_enabled`,
`allowed_tools`, and `mcp_tools`, and `src/tools.go` turns those into the schema list the model
sees:

```go
func selectTools(allowed []string, includeMCP bool) []Tool
```

So the configuration surface this needs already exists in the workflow format. What is missing is
that the choice is fixed for the life of the node. Nothing can change it while the node runs, and
`12.12` ("Per-agent tool scoping & personas") is `Ready` precisely because the static half is
already there.

### 5.2 The proposal: named modes

A node declares a default tool set and any number of named alternatives. The agent can ask to
enter one, and the set it is offered changes on the next turn.

```json
"properties": {
  "tools_enabled": true,
  "allowed_tools": ["read_file", "write_file", "patch_file", "execute_command", "spawn_sub_agent", "bm25_search"],
  "modes": {
    "lorebook": ["memory_write", "memory_read", "memory_list", "memory_disable"],
    "research": ["web_fetch", "bm25_search", "read_file"]
  }
}
```

Three rules keep it predictable.

**The agent may only enter a mode the node declared.** A name that is not in the map is a failed
tool call, so the reachable tool sets come from the workflow author rather than from the model.

**Switching is announced and recorded.** The active mode is stated in the system prompt each
turn, so the model always knows what it currently has, and the change is written to the session
event log so a transcript shows when it happened.

**Every mode keeps the switching tools.** `enter_mode` and `exit_mode` are available in all of
them, and `exit_mode` restores the node's default set. A mode is somewhere the agent can leave.

### 5.3 The mode belongs in the prompt, not only in the tool list

The point of a mode is not only to shorten the list. It is also a statement of what the agent is
doing. "You are in lorebook management mode" tells the model which task it is on, in the same way
that naming a sub-agent role does. That is worth more than the tokens it saves.

### 5.4 What it generalises to

The lorebook case is the motivation, not the boundary. The same mechanism covers a research mode
with network tools, a deployment mode with the destructive tools behind an approval, an MCP server
subset for one task, and a review mode that reads but does not write. It is a harness-wide
capability that happens to be motivated by the memory work.

### 5.5 What could go wrong

An agent that can change its own tool set can get it wrong. Three failure modes, and the cheap
mitigation for each.

- **Thrashing.** The agent enters and leaves a mode repeatedly. Switching is one tool call and it
  appears in the log, so the behaviour is visible rather than silent.
- **Getting stuck.** The agent enters a mode and can no longer do the task it was actually on.
  `exit_mode` is always available, and the node's default set is one call away.
- **Gating by omission.** A mode that removes a tool is not a permission boundary if the agent can
  leave the mode. A mode exists here to focus the agent, not to withhold capability from it, and
  the roadmap should not treat modes as policy. Where capability genuinely needs withholding, that
  is `12.16` (approval policy and sandbox as services), which is a separate mechanism.

## 6. Why not invent a format instead

The honest counter-argument is that a JSON file of keyed facts is not hard to design, and adopting
a format from a roleplay community for a coding harness is odd. Four reasons it is the better
trade.

**The algorithm is specified and tested.** Matching rules, case sensitivity, budget behaviour,
and the interaction between constants and selective entries are all written down, and the
reference implementation has tests for the corner cases. A new format would start with none of
that, and the interesting parts of retrieval are all in the corner cases.

**It is portable in a way a bespoke format is not.** The file can be opened, inspected, and edited
in tools that already exist and that users already have. Agent memory that a person can read in a
real editor, and hand-edit without fear, is worth more than a format with a slightly better field
name.

**The interoperability is real, not theoretical.** A lorebook the harness writes can be loaded by
other software, and a lorebook a user already owns can be loaded by the harness. Neither direction
needs a converter.

**The three-state fields and the preservation rules are already solved.** The guide records which
fields distinguish "no" from "not stated", and the implementation preserves unknown fields rather
than dropping them. Those are the details that make a format survive contact with real files
written by other people.

## 7. The security question, stated plainly

This is the part that needs a decision rather than a design.

Plan #2 states the rule for the harness: content fetched from the internet is marked
`untrusted_external` and **can never take the position of an instruction**. A character card is
third-party text from the internet. Using one as the agent's persona puts that text in the most
privileged position in the prompt, which is a deliberate exception to the rule.

It is a defensible exception, but only if it stays explicit:

- **The persona is chosen by the user in configuration, never by the agent.** A card does not
  become a persona because the agent read it. There is no tool that promotes content into the
  persona slot.
- **The persona block is labelled as a role definition**, distinct from system rules, so a
  reader of a transcript can see where the text came from.
- **The choice is recorded in telemetry**, so a session can be audited afterwards.
- **Lorebook entries never reach the persona slot**, whatever their priority.

The writable-memory idea has its own version of the same problem. An agent that can write memory
which is later injected into every future turn can be persuaded to write something durable. One
successful prompt injection becomes permanent influence rather than a single bad turn. Four
mitigations follow, and the last two are the ones that matter:

- Every entry records the session that wrote it.
- Writes are appended to the session event log like any other action.
- Injected entries are always delimited and described as reference material, never as
  instructions, so the framing survives into the prompt.
- Entries written by the agent are visible to the user, and removable, through the same directory
  listing that shows everything else.

## 8. Where this attaches to the roadmap

Three findings from reading the roadmap and the written plans.

**Row `12.12` already exists and is named "Per-agent tool scoping & personas".** It is owned by
Policy, its status is `Ready`, and its note says it is "most truthful once roles and task contexts
are unified". That row is about a different sense of the word persona, the capabilities an agent
has rather than the character it plays. The collision is worth taking seriously rather than
working around, because a card is a concrete, portable, user-authored role document. It could be
the thing that unifies roles and task contexts, which is what the row is waiting for.

**Plan #2 absorbs the ingest half with two new enumeration values.** Its central claim is that
every source becomes the same kind of record, so adding kinds is the intended extension. The
lorebook retrieval layer is a new retrieval strategy in the same system, beside the lexical
search that exists today.

**The whole feature is owned by plan #2, including the persona.** That is a decision that has
been taken rather than a recommendation: a persona is treated as one more ingestion target rather
than as a separate configuration concern, which keeps the whole feature inside one system and one
plan. The cost is that plan #2, already written, needs amending rather than extending. The
benefit is that the persona shares the evidence record, the trust vocabulary, and the staging
rules with everything else that gets ingested, instead of growing a parallel lifecycle in the
operator surface.

## 9. Proposed roadmap rows

Six rows, in the shape the roadmap uses. **The identifiers are proposed rather than assigned.**
New groups in this roadmap have been added as sub-series before, which is what `12.28.x` and
`12.29.x` are, so `12.30.x` follows that precedent. The band is the roadmap author's call and the
proposal is easy to renumber.

| ID | Item | Role | Status | System | Notes |
| --- | --- | --- | --- | --- | --- |
| 12.30.1 | Character cards as a persona source | Integration | **Ready** | Knowledge | Amends plan #2. The persona is an ingestion target, not a separate config surface. |
| 12.30.2 | Cards and lorebooks as evidence kinds | Substrate | **Blocked** | Knowledge | Two values added to the existing kind enumeration. Waits on the evidence substrate in plan #2. |
| 12.30.3 | Lorebook as a retrieval and injection layer | Retrieval | **Ready** | Knowledge | The matching algorithm, the scan window, and the token budget. The first thing worth building. |
| 12.30.4 | Agent-maintained lorebook memory | Retrieval | **Ready** | Knowledge | Read and write tools, every call logged. Depends on 12.30.3. |
| 12.30.5 | Mode-scoped tool sets | Policy | **Ready** | Policy | Extends `12.12`. The mechanism, not the lorebook feature. Useful whether or not the memory work proceeds. |
| 12.30.6 | Showcase workflow: persona, modes, lorebooks | Integration | **Ready** | Shell | A third workflow beside `linear_chat` and `enhanced_cognition`. |

Three notes on those.

**`12.30.5` is the one with the widest reach.** It can land first and pay for itself before any
card is ever loaded, because it applies to every existing tool. Nothing in it depends on the
lorebook work.

**`12.30.6` adds a workflow rather than changing one.** `linear_chat` and `enhanced_cognition`
stay as they are. The new workflow is where the persona node, a mode switch, and a lorebook
round-trip are demonstrated end to end, so the feature has one place where it is known to work
rather than being spread across the existing graphs.

**Nothing here needs a new plan document.** The Knowledge rows amend plan #2. `12.30.5` is a
`12.12` extension and belongs to whatever plan covers that row. `12.30.6` is a workflow, which is
configuration rather than a system.

## 10. Decisions taken, and what remains open

### 10.1 Decided

**The canonized material lives in `docs/character-cards/`.** The guide, the reference
implementation, and the sources manifest are together, so the guide and the module map can be read
side by side, and the harness `docs/` tree keeps holding documents.

**The specification text is committed; the implementation sources are not.** `ccv3_spec.md` and
`ccv3_concepts.md` are MIT licensed and are redistributed unmodified with attribution. SillyTavern
and lenML/char-card-reader are AGPL-3.0, so their files are recorded in `sources/MANIFEST.md` by
URL, revision, license, size, and content hash instead. The evidence stays auditable without
committing code this repository cannot license.

**Memory is readable and writable, tool based, with logging.** The agent gets a read and write
surface over lorebook entries and every call is recorded. An append-only log, rather than a
read-only design, is what makes the writes attributable, and attribution is what makes the surface
survivable.

**Tools are grouped into modes rather than presented all at once**, with the mode set configurable
per node, as section 5 describes.

**The whole feature is owned by plan #2 (Knowledge Ingress & Retrieval)**, including the persona.

**Nothing is implemented yet.** This document is the design; the work it describes becomes
roadmap rows and then plan amendments.

### 10.2 Still open

**Whether the persona is one card or several.** Recommendation: one card as the persona, and any
number as cards for evidence. A persona assembled from several cards has no obvious merge rule and
the fields do not compose.

**How the converted directory is addressed.** Recommendation: a workspace-level `characters/`
directory rather than `.goharness/`, so cards stay visible, versionable, and editable, and the
runtime state directory keeps containing only state.

**Whether the agent can create a whole new lorebook**, as opposed to entries within one. The
distinction matters because a new book is a new injection surface with its own budget, and the
first version may not need it.

**Where the active mode is recorded.** The session event log is the obvious place, and the exact
event shape depends on the schema work already scheduled in plan #1.

## 11. What this deliberately is not

**Not a roleplay mode.** No conversational assumptions, no greeting flow, no character voice
enforced at generation time. The card contributes text to one prompt element. A card also happens
to work for roleplay, which is what the format was designed for, but nothing in the harness
branches on that.

**Not a second memory store.** The evidence substrate stays the one place where ingested content
lives. The lorebook is a format for a view over it, and its entries point back at evidence
identifiers rather than duplicating blobs.

**Not a reason to add a runtime dependency.** The converter runs offline. The harness gains a
small JSON reader and a matching function, and both are testable in Go against the reference
implementation's behaviour. The dormant `ExtractPortablePythonRuntime` helper in `src/embed.go`
stays dormant, which is where it belongs until something actually needs a Python runtime at
execution time.

**Not a rewrite of the retrieval plan.** Plans #2 and #8 are amended, not replaced. The two new
evidence kinds and one new retrieval strategy are the whole of the change to plan #2.
