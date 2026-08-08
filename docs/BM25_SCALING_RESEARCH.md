# 🔬 Research Report: A Scalability Study of RAG Paradigms (BM25 Wins at Scale)

This document records the summary, methodology, core metrics, and engineering takeaways of the seminal July 2026 paper: **"BM25 Wins at Scale: A Scaling Study of Retrieval-Augmented Generation Paradigms"** (Wang et al., USTC & Metastone, arXiv:2607.26497v3).

---

## 📈 Executive Summary

In modern enterprise applications, Retrieval-Augmented Generation (RAG) spans several divergent methodologies: simple **lexical retrieval (BM25)**, **dense retrieval (embeddings)**, **graph-based RAG (MS-GraphRAG, LightRAG)**, and **File-System Agentic Search (Claude Code, Codex, SWE-bench style)**. 

To resolve their scaling accuracy and cost boundaries, the authors conducted a strictly controlled study across a **28-tier nested corpus ladder** growing 450-fold (from 1,144 to 511,959 documents, or 1.7M to 601M tokens), holding evidence and questions fixed.

### 🔑 The Core Revelation: The 10M Token Crossover
The paper reveals a **scale-dependent crossover** rather than an unconditional winner at every size:
* **The Bedrock (Small Scale):** File-System Agent leads with **77.4%** point-estimate accuracy (vs. 74.7% for BM25) by navigating raw folder paths sequentially.
* **The Crossover (10M Tokens):** Around **10 million corpus tokens**, BM25 overtakes the raw-tree File-System Agent.
* **The Enterprise Scale (Full Scale):** At the full 601M-token corpus, **BM25 leads all native single-shot paradigms at 50.5%**, while the raw File-System Agent collapses to **30.7%** and DenseRAG falls to **29.9%**.
* **The Synergy (Agent + BM25):** The absolute best scalability is achieved by **integrating agency on top of lexical candidate ranking**. An Agent backed by BM25 achieves **69.4% accuracy** at full scale, using **1/9th of the query tokens** of raw-file search!

---

## 📊 Comparative Scaling Performance Matrix

The following matrix aggregates the official main scaling results across completed tiers on EnterpriseRAG-Bench:

| Retrieval Paradigm | Bedrock (1.7M Tokens) | 10M Token Tier | Mid-Scale (131k Docs) | Full Scale (601M Tokens) | Primary Bottleneck / Limit |
| :--- | :---: | :---: | :---: | :---: | :--- |
| **BM25 (Lexical)** | 74.7% | **~67.0%** (Leads) | **55.2%** (Leads) | **50.5%** (Leads) | Synthesis limits, not discovery |
| **File-System Agent** | **77.4%** (Leads) | ~66.0% (Passed) | 50.9% | 30.7% | Sequential path search latency & query token bloat |
| **DenseRAG (Embeddings)** | 58.1% | 51.0% | 36.0% | 29.9% | Vulnerable to factual adversarial "traps" |
| **HippoRAG 2 (Graph)** | 66.2% | 58.6% | 41.0% | — (Unbuilt) | Linear indexing cost (2.9B tokens projected full-scale) |
| **MS-GraphRAG (Graph)** | 45.9% | 38.4% | — (Unbuilt) | — (Unbuilt) | Indexing Wall (7.9B token construction / 50 d) |
| **LightRAG (Graph)** | 48.0% | — (Unbuilt) | — (Unbuilt) | — (Unbuilt) | Super-linear Indexing Wall (102B token / 4 yr) |
| **Agent + BM25 (Hybrid)** | **90.1%** | — | — | **69.4%** (Frontier) | 101K query tokens (1/9th of raw file-agent) |

---

## 🔍 Deep-Dive of RAG Family Behaviors

### 1. 📂 File-FileSystem Agentic Search
* **The Mechanism:** Operates without a prior index. Uses an LLM to browse local trees, search contents, and read files dynamically through iterative tool loops.
* **The Failure Mode at Scale:** As the search space grows, the path depth increases, requiring the agent to make increasingly long sequential tool chains. Under an 80-call budget, **budget exhaustion reaches 31% at full scale**.
* **Query Token Overhead:** Query tokens grow from 226K per question at bedrock to over 895K at scale (**60x to 150x the cost of BM25**), leading to extreme latency and high costs.

### 2. 🎛️ Lexical (BM25) vs. Dense (Embeddings)
* **The Lexical Advantage:** Enterprise knowledge bases contain highly precise lexical anchors (part numbers, ticket IDs, function names), whereas adversarial "traps" are semantically identical but factually outdated.
* **Why BM25 Leads:** BM25 uses exact lexical matching, which successfully filters out semantic traps. Dense retrieval (DPR) matches semantic neighborhoods, making it highly vulnerable to factually wrong but semantically similar distractors, resulting in a **20.6% accuracy deficit** behind BM25 at full scale.

### 3. 🕸️ Graph-Based RAG ("The Construction Wall")
* **The Bottleneck:** Building entity-relationship graphs requires passing every chunk through an LLM to extract nodes, communities, and generate reports.
* **The Scaling Ceiling:** 
  * **MS-GraphRAG** requires **7.9 Billion construction tokens** (projected 50 single-instance days) to build the full 601M token corpus.
  * **LightRAG** runs super-linearly ($b=1.36$), requiring an estimated **102 Billion construction tokens** (4 single-instance years) to compile the full index, making it practically undeployable for growing document bases.
  * **LinearRAG** (embedding-only graph construction without LLM extraction) completes within 1.8% of MS-GraphRAG accuracy, proving that heavy generative graph construction adds cost far faster than signal.

---

## 💡 Practical Engineering Guidance for GoHarness

The USTC/Metastone scaling study provides **exceptional, highly actionable architectural pointers** for developing local-first, lightweight agent harnesses like **GoHarness**:

1. **"Agency Works Best After Ranked Discovery, Not in Place of It"**
   * **The Lesson:** Do not ask the agent to search raw directory trees sequentially (`ls`, `find`, `grep`) when dealing with larger file bases. The exploration will fail, exhaust the loop context, and bloat query tokens.
   * **GoHarness Action:** For multi-file search and global codebase exploration, GoHarness should invoke a local, lightweight lexical indexer (like a fast BM25 or ripgrep wrapper) to identify the top-5 candidate file chunks first, and then present only these narrowed candidates inside the agent's context for recursive reasoning.
2. **Beware of Heavy Graph RAG Indexes**
   * Unless dealing with highly dense, relational-heavy questions, avoid heavy offline knowledge-graph extraction models. They introduce immense extraction noise, take days/weeks of CPU compile time, and are outperformed by robust lexical matchers at scale.
3. **The Power of Compact Hybrid RAG**
   * Integrating standard lexical ranking (for candidate discovery) with agentic reasoning (for information synthesis and multi-turn correction) represents the absolute **Pareto frontier** of accuracy, query speed, and computational cost.

---

*Ref: Pengyu Wang, Benfeng Xu, Shaohan Wang, Mingxuan Du, Xin Zeng, Huarui Wu, Lei Zhang, Licheng Zhang. "BM25 Wins at Scale: A Scaling Study of Retrieval-Augmented Generation Paradigms" (July 2026).*
