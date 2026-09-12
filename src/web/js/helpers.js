export function escapeHtml(str) {
  const div = document.createElement("div");
  div.textContent = str == null ? "" : String(str);
  return div.innerHTML;
}

export function toolCallTitle(tc) {
  const name = (tc && tc.function && tc.function.name) || "tool";
  let summary = "";
  try {
    const args = tc.function.arguments ? JSON.parse(tc.function.arguments) : {};
    if (args.path) summary = args.path;
    else if (args.command) summary = args.command;
    else if (args.query) summary = args.query;
    else {
      for (const k of Object.keys(args)) {
        if (typeof args[k] === "string") {
          summary = args[k];
          break;
        }
      }
    }
  } catch {
    summary = (tc.function.arguments || "").trim();
  }
  summary = (summary || "").replace(/\s+/g, " ").trim();
  if (summary.length > 60) summary = summary.slice(0, 57) + "…";
  return summary ? `${name} · ${summary}` : name;
}

export function toolResultTitle(name, content) {
  const spillMatch = (content || "").match(/spill_id:\s*([a-f0-9]{16,64})/i);
  if (spillMatch) {
    return `result · ${name || "tool"} · spill ${spillMatch[1].slice(0, 12)}…`;
  }
  const len = (content || "").length;
  let firstLine = (content || "").split(/\r?\n/).find((l) => l.trim().length > 0) || "";
  firstLine = firstLine.replace(/\s+/g, " ").trim();
  if (firstLine.length > 50) firstLine = firstLine.slice(0, 47) + "…";
  const sizeLabel = len < 1024 ? `${len} ch` : `${(len / 1024).toFixed(1)} KB`;
  const base = `result · ${name || "tool"} · ${sizeLabel}`;
  return firstLine ? `${base} — ${firstLine}` : base;
}

export function knownModelsForProvider(provider) {
  switch (provider) {
    case "anthropic":
      return ["claude-3-5-sonnet-latest", "claude-3-5-haiku", "claude-3-7-sonnet-latest"];
    case "gemini":
    case "vertex":
      return ["gemini-1.5-flash", "gemini-1.5-pro", "gemini-3.1-flash-lite"];
    default:
      return ["gpt-4o", "gpt-4o-mini", "o3-mini"];
  }
}

export function populateKnownModelList(listId, provider) {
  const list = document.getElementById(listId);
  if (!list) return;
  list.innerHTML = knownModelsForProvider(provider).map((model) => `<option value="${model}"></option>`).join("");
}
