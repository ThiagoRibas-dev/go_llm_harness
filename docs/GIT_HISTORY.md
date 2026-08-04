# 🔬 GoHarness: Git Commit History & Development Ledger

This document acts as an audit-ready development ledger recording the complete chronological history of git commits, structural refactors, bug resolutions, and system implementations during the lifecycle of the **GoHarness** platform.

---

## 📈 Summary of Contributions

* **Total Commits:** 40
* **Repository Root:** `https://github.com/ThiagoRibas-dev/go_llm_harness`
* **Core Contributors:**
  * **ThiagoRibas-dev** (Strategic enhancements, hierarchical memory, multi-agent concurrency, and Windows sandboxing)
  * **GoHarness Developer** (Initial core platform implementations and research baselines)

---

## 📋 Chronological Commit Log

### 1. Upgraded Context Compaction, In-Place Branching, & Windows Sandbox Fallbacks
> **Commit Hash:** `ee8907f5276deeb998af1065cb183ec26dae4dd5`  
> **Author:** ThiagoRibas-dev <thiago.ribas@dev.com>  
> **Date:** Tue Aug 4 21:16:42 2026 +0000  
* Upgraded the rolling window compaction engine to group by actual **user turns** (conversational rounds) rather than raw JSON file counts, resolving early-compaction triggers.
* Implemented **True State Eviction** by physically archiving compacted JSON logs into soft-related, named sibling directories (`compacted_summary_up_to_turn_%03d/`), resetting the active turn counter.
* Unified metadata storage by moving the compaction boundary directly inside **`meta.json`** (`CompactionBoundary` field), eliminating `compaction_boundary.txt` entirely.
* Resolved a Windows-specific file-lock bug by normalising all paths using `filepath.Clean()`.
* Implemented `findMaxTurnNumber()` to scan both active and archived folders, ensuring **100% linear, logical, and sorted turn-number sequences** across session loads and resumes.
* Implemented **In-Place Conversational Branching (Edit & Fork)** on any past user or assistant message cards, executing parallel edited timelines recursively in sub-sessions.
* Implemented **One-Click Turn Regeneration (Reroll)** to cleanly delete the last assistant/tool turns, revert workspace file changes, and resubmit the user prompt.
* Implemented **Standard-Compliant Claude Code 9-Aspect Compaction Prompt** and interwoven non-code specific pointers (research synthesis, draft baselines, desired tones).
* Registered and wired `/api/mcp` configuration controllers enabling hot-reload registrations, process handshakes, and tool listings.
* Rebuilt and cross-compiled optimized static binaries for all major platforms (Linux, Windows, macOS Intel, and macOS Apple Silicon).

### 2. Multi-Agent Concurrency and Synchronization Specs
> **Commit Hash:** `c91636a37778d7837b306802149487dd0b313310`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Tue Aug 4 14:15:38 2026 +0000  
* Updated ROADMAP.md with complete Phase 10 Multi-Agent Concurrency and Wait Mode synchronization specifications.

### 3. Comparison Matrix Expansion: Workspace UX & File Pickers
> **Commit Hash:** `de85226ca7a5c89030fd94737e50bd3e5dc5fbf5`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Tue Aug 4 14:10:45 2026 +0000  
* Expanded COMPARISON_MATRIX.md with detailed Workspace UX, Native File Pickers, and directory walking vectors.

### 4. Comparison Matrix Expansion: Session Branching & Rollbacks
> **Commit Hash:** `8509082c9b8cd76ec5cd8a1a5ac6687c5a31bc86`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Tue Aug 4 14:00:15 2026 +0000  
* Expanded COMPARISON_MATRIX.md with detailed Workspace and Session/Branching comparison vectors.

### 5. Llama.cpp & SimpleChat Comparative Auditing
> **Commit Hash:** `aab6a3967e0b6b7122eea5255a58c1a880f62a95`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Tue Aug 4 13:32:26 2026 +0000  
* Updated COMPARISON_MATRIX.md with llama.cpp SimpleChat comparisons and extensive systems-level architectural details.

### 6. Technical Self-Audit Against SOTA Frameworks
> **Commit Hash:** `0240019666634c6c484bf35adf132bbc6c4da1e4`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Tue Aug 4 13:12:39 2026 +0000  
* Added COMPARISON_MATRIX.md conducting a deep, technical self-audit against industry-leading developer tools.

### 7. Core Research Vectors Formalization
> **Commit Hash:** `6e45c7969c9d52c833687fe1f8df11d77b9a19c9`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Tue Aug 4 13:08:17 2026 +0000  
* Formalized and wrote out the 9 Core Agentic Research Vectors inside RESEARCH.md.

### 8. IDE Copilots Integration Specs
> **Commit Hash:** `72726fc2f65a2b926521e8942d82c8996403bb2a`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Tue Aug 4 13:03:14 2026 +0000  
* Updated RESEARCH.md with specifications for Continue, Cline, Zoo Code, and OpenCode integrations.

### 9. Research Bibliography & Inspirations Ledger
> **Commit Hash:** `1d5164a44c317569980f0e88676c2beea546578f`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Tue Aug 4 13:01:59 2026 +0000  
* Wrote RESEARCH.md documenting the bibliography, open-source inspirations, and research lineages of the project.

### 10. Dual-Scoped Injections & Phase 9 Specs
> **Commit Hash:** `b81077a8270f6f2996bc5a915be53b3714564baf`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Tue Aug 4 03:08:17 2026 +0000  
* Updated README.md and ROADMAP.md to record complete dual-scoped injections and Phase 9 cognitive memory specs.

### 11. Hierarchical Cognitive Memory Specifications
> **Commit Hash:** `ad70a7317c72d7f83657c4929172f85189e7d61b`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Mon Aug 3 23:10:08 2026 +0000  
* Updated ROADMAP.md with detailed Phase 9 specifications for Hierarchical Cognitive Memory and a Visual Memory Map Timeline.

### 12. Completed Phase 8 Specifications
> **Commit Hash:** `0337b9028acd91dfade930801acd751cb6ffbac2`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Sat Aug 1 22:10:12 2026 +0000  
* Updated README.md and ROADMAP.md with final completed Phase 8 specifications.

### 13. Dual-Scoped Guideline Injections (`CLAUDE.md`)
> **Commit Hash:** `932aa0b35c1d77595a0e055e49919eca553fc019`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Sat Aug 1 22:07:12 2026 +0000  
* Implemented dual-scoped guideline injections scanning both active project workspaces and global system roots with CLAUDE.md support.

### 14. Launch Session Resumption Persistence
> **Commit Hash:** `b4ab100a27aeb69058d934cefa20213fb4a24492`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Sat Aug 1 22:02:17 2026 +0000  
* Implemented Phase 8.6 launch session-resumption persistence and directory checkers.

### 15. Turn Index Restoration fixes
> **Commit Hash:** `9f43cdef80124315ace57f4ec3fd460817bf54b4`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Sat Aug 1 21:37:26 2026 +0000  
* Resolved UI history-loading omission of turn_number parameters.

### 16. Config pathing and binary directories fixes
> **Commit Hash:** `40b8e83b0990f250eacfd9a4b6e8435256030ff7`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Sat Aug 1 21:29:24 2026 +0000  
* Resolved config.go pathing dependencies and complete executable path conversions.

### 17. High-verbosity Diagnostic logs
> **Commit Hash:** `57ea346277d592b98c911d650d9396dda3dd6021`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Sat Aug 1 21:26:01 2026 +0000  
* Added high-verbosity diagnostic console printouts to session file loader.

### 18. Execution-path directory sensitivity fixes
> **Commit Hash:** `f9385ee92013d525285f6fe8e8d588d34f08d978`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Sat Aug 1 21:16:34 2026 +0000  
* Resolved execution-path sensitivity by making system folders and config files resolve relative to the executable binary directory.

### 19. Session-restore TypeError fixes
> **Commit Hash:** `045b2b8110056a00d777bd360e98a32e4353de89`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Sat Aug 1 21:09:08 2026 +0000  
* Resolved session-restore TypeError on empty/clean history streams.

### 20. Token & Character Trackers with Dynamic Cost Calculator
> **Commit Hash:** `55a35070c5392b0e274ccf3f0687457854650cf7`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Sat Aug 1 20:48:25 2026 +0000  
* Implemented Phase 8.6 real-time total tokens and character usage trackers with dynamic USD pricing calculators.

### 21. Google thoughtSignature echo fixes
> **Commit Hash:** `2f8148a61168c1f11470fdc07eb82b1c58435bc9`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Sat Aug 1 20:37:37 2026 +0000  
* Resolved Gemini 400 error by preserving and echoing back native Google thoughtSignature fields.

### 22. Vertex AI 401 unauthenticated fixes
> **Commit Hash:** `0fa72a038b6b0d852b1c93bea91e6630a82f3b78`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Sat Aug 1 20:18:46 2026 +0000  
* Resolved Vertex AI 401 unauthenticated error by decoupling Authorization headers from API key queries.

### 23. GCP Vertex dynamic configurations
> **Commit Hash:** `d45ca0e1638ec179accce0587a9f9a3e6b59a695`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Sat Aug 1 20:02:31 2026 +0000  
* Implemented Phase 8.6 Google Vertex dynamic URL builders, sampling tuning, and reactive settings modal form fields.

### 24. Untracked Created File Tracking
> **Commit Hash:** `f4a7e9ce71e7d5624ba53563626351d587a3bd47`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Sat Aug 1 19:54:59 2026 +0000  
* Implemented Phase 8.5 Untracked Created File Tracking and perfect rollback restorations.

### 25. Web Server configs & Session creations
> **Commit Hash:** `a28378947960364b7d51b641a59eef104e006847`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Sat Aug 1 19:51:40 2026 +0000  
* Implemented Phase 8.4 Web Server configurations, beautiful console boot logs, and session creation endpoints.

### 26. Complete Phase 7 & 8 specs inside README
> **Commit Hash:** `bc895a5dd1edb28b7f1f29f1e0ab910887733212`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Sat Aug 1 19:11:49 2026 +0000  
* Updated README.md with complete Phase 7 and 8 specifications and full config parameter tables.

### 27. Production GUI Boot Logging
> **Commit Hash:** `2e75e95f9c7ca8c78a759b01b526a8b7b87b60bf`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Sat Aug 1 19:04:16 2026 +0000  
* Implemented Phase 8.4 Web Server configurations and beautiful console boot logs.

### 28. .gitignore optimizations
> **Commit Hash:** `c7c14903d93f473aed75c712889d0baf51a92020`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Sat Aug 1 18:57:53 2026 +0000  
* Ignore uploads directory in `.gitignore`.

### 29. Completed Phase 8 specifications
> **Commit Hash:** `9e86a0181b69a4767b8ed26d65896ba6c9ac989d`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Sat Aug 1 18:43:40 2026 +0000  
* Updated ROADMAP.md to mark Phase 8 as complete.

### 30. OpenAI-Compatible API Gateway
> **Commit Hash:** `e0625ee15441a2f93e3f60035132a681441a7a40`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Sat Aug 1 18:43:15 2026 +0000  
* Implemented Phase 8 OpenAI-Compatible API Gateway with List Models, Completions, Embeddings, and Tokenizer/Detokenizer proxies.

### 31. Tokenizer & Detokenizer Gateway Specs
> **Commit Hash:** `b42e74d32e945e072073b08aecc6ffd22e6b36c8`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Sat Aug 1 18:38:28 2026 +0000  
* Added detailed Phase 8 specifications to ROADMAP.md including Tokenizer and Detokenizer proxies.

### 32. Multi-Provider API Connectors
> **Commit Hash:** `5149c038510ac4bba0e5ab9baef50584c4ef4376`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Sat Aug 1 18:07:02 2026 +0000  
* Implemented Phase 7 Multi-Provider AI API Connectors (OpenAI, Anthropic, Gemini, Vertex AI).

### 33. Architectural Diagram update
> **Commit Hash:** `01e444803f60e585e4bb3ae5981eff9f215b34c8`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Sat Aug 1 18:00:24 2026 +0000  
* Updated `harness_architecture.svg` with complete Phase 3-6 streaming, MCP, and bare-metal sandbox details.

### 34. Non-Destructive Parallel Branching
> **Commit Hash:** `39e3f376a3e79c9be25f9c6c075473ac4a87076b`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Sat Aug 1 17:55:46 2026 +0000  
* Implemented Workspace Management, Conversations indexer, and Non-Destructive timeline branching.

### 35. Core structural refactor
> **Commit Hash:** `c9d6e4685f1aaeb641760a431aeaea9034d672e9`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Sat Aug 1 17:21:53 2026 +0000  
* Moved all Go source and web assets inside `src/` folder, and updated `README.md` compilation instructions.

### 36. Initial Commit
> **Commit Hash:** `d7a044a07d174f82c3f77be5a45a36ee07b619d7`  
> **Author:** GoHarness Developer <dev@goharness.io>  
> **Date:** Sat Aug 1 17:11:30 2026 +0000  
* Initial commit: Complete GoHarness local secure agent & web console platform.

---

*End of Development Ledger.*
