# 🔬 Technical Specification: GoHarness v2.0 LLM-Assisted Workflow Creator & Staging Engine

This document defines the complete system design, prompt engineering guidelines, JSON schema constraints, and UI/UX lifecycles of the **LLM-Assisted Workflow Creator & Staging Engine** inside **GoHarness v2.0**. 

This subsystem allows developers to describe complex reasoning pipelines in plain natural language (e.g. *"First perform a BM25 search on the workspace, then run 3 parallel LLMs to analyze different aspects of the results, and finally aggregate them using Claude 3.5"*), automatically compiling them into verified, standard-compliant `workflows.json` schemas.

---

## 🎨 1. End-to-End Execution Lifecycle

To prevent syntax errors, infinite execution hangs, or unsafe file-system writes, GoHarness 2.0 adopts a **Staged-Commit Pattern** for LLM-generated configurations:

```
┌─────────────────┐      ┌─────────────────────────┐      ┌───────────────────────────┐
│  User Request   ├─────►│  LLM Generator Engine   ├─────►│  Staging & Verification   │
│  (Plain Text)   │      │  (Using Cookbook Spec)  │      │  (UI Canvas & JSON Panel) │
└─────────────────┘      └─────────────────────────┘      └─────────────┬─────────────┘
                                                                        │
                                                                        ▼ [User Edit / Confirm]
┌─────────────────┐      ┌─────────────────────────┐      ┌─────────────┴─────────────┐
│ Active Pipeline ◄──────┤ workflows.json on Disk  │◄─────┤   "Compile & Apply"       │
│  (Hot-Reloaded) │      │ (Unified Configuration) │      │   (Dynamic Validation)    │
└─────────────────┘      └─────────────────────────┘      └───────────────────────────┘
```

1. **Generation:** The user types a natural language description. GoHarness dispatches the request to the LLM using our highly structured **Cookbook System Prompt**.
2. **Staging:** The LLM returns a JSON payload containing the proposed workflow. GoHarness intercepts the JSON and loads it into the **Staging & Verification Panel** on the Web UI.
3. **Verification:** The Web UI compiles a live, non-editable **visual node graph preview** on the canvas, while presenting the raw JSON inside an editable code editor panel. The user inspects, tweaks system prompts, or corrects any port connections.
4. **Validation:** Clicking **Compile & Apply** triggers a strict in-process Go validation pass (checks for circular dependencies, unlinked inputs, and missing terminal anchors).
5. **Commit:** Upon validation success, the JSON is written to `workflows.json` and the active workflow is hot-swapped dynamically!

---

## 📖 2. The Compaction/Compilation Cookbook Prompt

To ensure the LLM generates a $100\%$ valid, syntactically correct, and parsable DAG, the generator is bound to this precise, highly-engineered system instruction:

```markdown
You are the GoHarness v2.0 AI Workflow Compiler. Your task is to translate a user's natural language request into a 100% syntactically valid JSON Directed Acyclic Graph (DAG) configuration. 

### 🚫 CRITICAL CONSTRAINTS:
1. Output ONLY a raw, un-enclosed JSON block. Do NOT wrap the output in markdown code blocks (such as ```json).
2. Every node in your graph must strictly map to one of our five standard GoHarness Node types.
3. The graph MUST start with a node of type "user_input" (id: "start") and end with a node of type "assistant_response" (id: "terminal").
4. Every input connection must map a valid upstream output port to a valid downstream input port.

### 🔌 PORT CONNECTIVITY RULES:
* "user_input" Node:
  - Outputs: ["prompt" (string), "uploaded_files" (array of strings)]
* "llm_query" / "llm_synthesis" Node:
  - Inputs: ["prompt" (string), "optional_system_override" (string)]
  - Outputs: ["response" (string)]
* "tool_execution" Node:
  - Inputs: ["arguments" (JSON string)]
  - Outputs: ["stdout" (string), "exit_code" (integer)]
* "bm25_search" Node:
  - Inputs: ["query" (string)]
  - Outputs: ["search_results" (string)]
* "assistant_response" Node:
  - Inputs: ["final_output" (string)]
  - Outputs: [] (Terminal Anchor)

### 📊 EXAMPLE SCHEMAS (Cookbook Baseline):
If the user requests a "Parallel Search and Summary":
{
  "name": "Parallel Search and Summary",
  "description": "Searches workspace files using BM25 and condenses them using an LLM.",
  "nodes": [
    {
      "id": "start",
      "type": "user_input",
      "properties": {},
      "inputs": []
    },
    {
      "id": "searcher",
      "type": "bm25_search",
      "properties": { "scope": "workspace", "limit": 5 },
      "inputs": [
        { "source_node": "start", "source_output": "prompt", "target_input": "query" }
      ]
    },
    {
      "id": "aggregator",
      "type": "llm_synthesis",
      "properties": {
        "provider": "openai",
        "model": "gpt-4o-mini",
        "temperature": 0.2,
        "system_prompt": "Synthesize the search results against the original user prompt."
      },
      "inputs": [
        { "source_node": "searcher", "source_output": "search_results", "target_input": "prompt" }
      ]
    },
    {
      "id": "terminal",
      "type": "assistant_response",
      "properties": {},
      "inputs": [
        { "source_node": "aggregator", "source_output": "response", "target_input": "final_output" }
      ]
    }
  ]
}
```

---

## 🎨 3. Staging & Verification Screen Specification

The Staging screen is built directly into the Web Console Settings modal as a distinct sub-tab: **AI Workflow Lab**.

```
┌────────────────────────────────────────────────────────┐
│ 🧪 AI WORKFLOW LAB (STAGING)                           │
├────────────────────────────────────────────────────────┤
│  Describe your pipeline in plain text:                 │
│  [ I want to search files and run parallel critiques ]  │
│  [✨ Generate Workflow Draft]                          │
├────────────────────────────────────────────────────────┤
│  ┌─────────────────────────┐  ┌──────────────────────┐ │
│  │ 🌲 Visual Staged Graph  │  │ 💻 Editable JSON     │ │
│  │                         │  │ {                    │ │
│  │   [Start] ──► [Search]  │  │   "nodes": [...]     │ │
│  │                         │  │ }                    │ │
│  └─────────────────────────┘  └──────────────────────┘ │
├────────────────────────────────────────────────────────┤
│  [⚠️ Validation: Ready]         [🚀 Compile & Apply Now]│
└────────────────────────────────────────────────────────┘
```

### 3.1 The Verification Steps:
1. **Interactive Generation:** The user types their request in the text input and clicks **Generate Workflow Draft**.
2. **Visual Overlay Compilation:** The frontend unmarshals the returned JSON payload and draws a live, interactive topological node preview on the left side of the panel. 
   * *The benefit:* The user can visually trace if the connections are correct (e.g., verifying that the output of the searcher is actually wired to the input of the critique model).
3. **JSON Code Panel:** The raw JSON is presented in an interactive, syntax-highlighted editor on the right side. The user can directly edit prompts, model names, or ports.
4. **Local Port Validation:** The UI dynamically monitors the JSON for structural soundness:
   * **Circular Check:** Ensures there are no infinite feedback loops (cycles).
   * **Port Match Check:** Warns the user in bright orange text if an output port type does not match its connected input port.
   * **Anchor Check:** Verifies that both `start` and `terminal` nodes exist.
5. **The Safe Commit:** Once verified, the user clicks **Compile & Apply**. GoHarness performs a final Go-level safety check, writes the verified code to `workflows.json`, and swaps your active run-time pipeline seamlessly.

---

This specification provides **GoHarness v2.0** with a highly advanced, user-friendly, and secure **natural-language compiler pipeline**, making workflow orchestration accessible to every developer!
