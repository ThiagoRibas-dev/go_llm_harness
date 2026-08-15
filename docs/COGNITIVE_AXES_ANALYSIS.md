# 🔬 Theoretical Analysis: The Cognitive Axes of Large Language Models

This document formalizes the multidimensional space of LLM reasoning, examining how model scale (parameter width, network depth, and attention capacity) dictates a model's ability to handle high "cognitive load" without structural collapse. 

---

## 🧬 Introduction: The Geometry of LLM "Internal Space"

In transformer-based architectures, an LLM does not merely store facts; it projects concepts into a high-dimensional vector space ($d_{\text{model}}$). 
* **The Width ($d_{\text{model}}$ & Parameters):** Controls the *representational capacity*—the number of independent concepts, nuances, and features the model can represent simultaneously.
* **The Depth (Layers):** Controls the *computational capacity*—the number of sequential "reasoning steps" or logical operations the model can perform during a single forward pass.
* **The Attention Heads:** Control the *relational capacity*—the number of concurrent data points and historical context tokens the model can keep in active focus.

When a model is small, its "internal space" is geometrically restricted. When forced to handle high cognitive load (e.g., maintaining chronological plot continuity, obeying strict physical rules, representing character sub-intentions, and polishing literary style simultaneously), its internal representational dimensions collide, leading to **cognitive collapse** (hallucinations, loss of factual continuity, or logical contradictions).

To understand this boundary, we isolate **five distinct classes of cognitive axis**.

---

## 📊 The 5 Core Classes of Cognitive Axis

```
                    ▲ [Logical / Causal]
                    │   (Rules, Constraints, State Progression)
                    │
[Chronological] ◄───┼───► [Semantic / World]
(Timelines, Memory) │   (Ontology, Physical Reality)
                    │
                    ▼ [Behavioral / Psychological]
                        (Theory of Mind, Agentic Intent)
```

### 1. ⏳ The Chronological Axis (Temporal Continuity & State Tracking)
* **Definition:** The capacity to maintain absolute consistency across a sequential progression of events, times, or conversational turns.
* **Cognitive Manifestation (Creative):** Ensuring a character who lost a sword in Chapter 2 does not suddenly draw it in Chapter 4; preserving the timeline's age progression, dates, and geographic locations.
* **Cognitive Manifestation (Technical):** Keeping track of execution traces, mutable variable states, and system file changes across dozens of tool-calling loops.
* **Why Small Models Fail:** Attention head saturation. As the context window fills, a small model's attention weights become diffuse. It "forgets" or misallocates older state parameters because it lacks the relational capacity to hold distant tokens in tight mathematical focus.

### 2. ⛓️ The Causal-Logical Axis (Causal Chains & Strict Constraints)
* **Definition:** The capacity to model cause-and-effect relationships and obey strict, unyielding system/logical rules.
* **Cognitive Manifestation (Creative):** Obeying the physical laws of the fictional world (e.g. gravity, magic systems) or ensuring that if a door is locked, a character cannot enter unless they find the key.
* **Cognitive Manifestation (Technical):** Obeying compiler rules, system privileges, type declarations, or legal/operational instructions.
* **Why Small Models Fail:** Insufficient network depth. Deep feed-forward (MLP) blocks act as sequential logical circuits. A smaller model (fewer layers) cannot perform the nested multi-step inferences (e.g., *"If A is true, then B must occur, which blocks C, which forces D"*) in a single forward pass, collapsing instead into shallow, plausible-sounding correlation patterns.

### 3. 🗺️ The Semantic-World Axis (Ontology & High-Dimensional Reality)
* **Definition:** The internal, spatial-ontological map representing the physical reality of objects, their properties, and their relationships.
* **Cognitive Manifestation (Creative):** Spatial reasoning (e.g., understanding that a key is inside a drawer, the drawer is inside a desk, and the desk is in the study).
* **Cognitive Manifestation (Technical):** Understanding file-system hierarchies, directory scopes, and structural class architectures.
* **Why Small Models Fail:** Restrictive hidden dimension ($d_{\text{model}}$). Smaller models project concepts into a narrow high-dimensional space where concepts overlap too heavily. It loses spatial/conceptual resolution, resulting in physical impossibilities (e.g., *"He reached into his pocket and pulled out the hook the coat was hanging on"*).

### 4. 🧠 The Behavioral-Psychological Axis (Theory of Mind & Persona)
* **Definition:** The capacity to represent, track, and simulate multiple, nested "minds" (character motivations, secret intents, or sub-agent objectives) simultaneously.
* **Cognitive Manifestation (Creative):** Nested Theory of Mind (e.g., *Character A has a secret that Character B does not know, but Character A believes Character C has leaked it*).
* **Cognitive Manifestation (Technical):** Managing parallel agent execution, delegating specialized search tasks, and tracking parent-to-sub-agent progress boundaries.
* **Why Small Models Fail:** Sub-space collapse. Maintaining separate, nested "mental states" requires the model to allocate independent, high-dimensional representational vectors. Smaller models cannot isolate these vectors, resulting in "bleed-through"—where all character personalities or agent objectives homogenize into a flat, uniform voice.

### 5. 🎭 The Stylistic-Prose Axis (Form, Aesthetics, & Syntax)
* **Definition:** The surface-level presentation layer (grammar, vocabulary, tone, prose rhythm, formatting rules, or coding syntax guidelines).
* **Cognitive Manifestation (Creative):** Emulating 19th-century Gothic prose or matching specific character dialects.
* **Cognitive Manifestation (Technical):** Obeying strict indentation rules (YAML/Python) or writing compact, idiomatic, and clean code.
* **Why Small Models Fail:** Middle-layer overload. Syntactic and lexical structures are primarily processed in the earlier and middle layers of the transformer. If a small model must invest its entire computational capacity into maintaining stylistic form, it runs out of "internal space" to calculate plot progression or logical constraints, resulting in beautiful prose that is factually or logically completely broken.

---

## 🎛️ How GoHarness Optimizes for the Cognitive Axis

When executing autonomous workflows, **GoHarness's architecture specifically relieves the cognitive load of smaller local LLMs** across these axes by offloading systems work to robust, platform-native tools:

| Cognitive Axis | LLM Failure Mode | GoHarness Architectural Mitigation |
| :--- | :--- | :--- |
| **⏳ Chronological** | Attention saturation & history bloat | **Dynamic Memory Eviction & Compaction:** Evicts raw logs into `compacted_summary_` sibling folders, keeping the active context window clean and bounded. |
| **⛓️ Causal-Logical** | Multi-step inference failure | **Explicit Tool Schemas (`patch_file`, `read_file`):** Offloads complex file manipulation and search from raw shell scripts to precise, structurally validated tools. |
| **🗺️ Semantic-World** | Spatial & ontological confusion | **Auto-LS Tree Walk & `target_scan_dirs`:** Provides a clean, token-safe directory map directly in the context, eliminating the need for the model to guess file scopes. |
| **🧠 Behavioral** | Sub-space / objective collapse | **Recursive Sub-Agent Delegation (`spawn_sub_agent`):** Fully isolates parallel research or code analysis tasks into independent subprocesses with their own sessions. |
