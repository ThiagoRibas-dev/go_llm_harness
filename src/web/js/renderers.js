import { escapeHtml } from "./helpers.js";

export function deliverablePathsFromTool(turn) {
  if (!turn || turn.role !== "tool") return [];
  const metaArtifacts = turn.meta && Array.isArray(turn.meta.artifacts) ? turn.meta.artifacts : null;
  if (metaArtifacts && metaArtifacts.length > 0) {
    return metaArtifacts.filter(Boolean);
  }
  const content = String(turn.content || "");
  const out = [];
  const matches = [
    content.match(/Successfully wrote file to host disk at\s+(.+)$/m),
    content.match(/Successfully wrote file inside Docker at\s+(.+)$/m),
    content.match(/Successfully patched file '([^']+)'/m),
  ].filter(Boolean);
  matches.forEach((m) => {
    const path = (m[1] || "").trim();
    if (path && !out.includes(path)) out.push(path);
  });
  return out;
}

export function renderDeliverableChips(paths) {
  if (!paths || paths.length === 0) return "";
  return `<div class="deliverable-chip-row px-3 py-2 border-t border-slate-800/70">` + paths.map((path) => `
      <button type="button" class="deliverable-chip" onclick="openWorkspaceFile(${JSON.stringify(path)})">Open ${escapeHtml(path)}</button>
      <button type="button" class="deliverable-chip" onclick="stageWorkspaceFile(${JSON.stringify(path)})">Stage ${escapeHtml(path)}</button>
    `).join("") + `</div>`;
}

export function renderTerminalBlock(content) {
  const lines = String(content || "").split(/\r?\n/);
  const rendered = lines.map((line) => {
    let cls = "term-line";
    if (/^Exit Code:/i.test(line)) cls += " term-exit";
    else if (/^(Stdout:|Stderr:)/.test(line)) cls += " term-section";
    else if (/^(Sandbox Execution Error:|Security Exception:|Docker write failed:|Patch Error:|Failed to)/.test(line)) cls += " term-error";
    else if (/^⚠️|^Warning/i.test(line)) cls += " term-warning";
    return `<span class="${cls}">${escapeHtml(line || " ")}</span>`;
  }).join("");
  return `<div class="typed-terminal-lines">${rendered}</div>`;
}

export function renderReadBlock(content) {
  const lines = String(content || "").split(/\r?\n/);
  const rows = [];
  const extras = [];
  lines.forEach((line) => {
    const m = line.match(/^(\d+) \| ?(.*)$/);
    if (m) {
      rows.push(`<div class="typed-read-row"><span class="typed-read-line">${escapeHtml(m[1])}</span><span class="typed-read-text">${escapeHtml(m[2])}</span></div>`);
    } else if (line.trim()) {
      extras.push(line);
    }
  });
  const meta = extras.length ? `<div class="typed-block-meta">${escapeHtml(extras.join(" "))}</div>` : "";
  if (rows.length === 0) return renderTerminalBlock(content);
  return `<div class="typed-read-block">${rows.join("")}</div>${meta}`;
}

export function renderSearchBlock(content) {
  const headerMatch = String(content || "").match(/^===\s*(.*?)\s*===/m);
  const body = [];
  const re = /\[(\d+)\]\s+File:\s+(.+?)\s+\(BM25 Relevance Score:\s+([^)]+)\)\nExcerpt:\n([\s\S]*?)(?=\n--------------------------------------------------\n|$)/g;
  let m;
  while ((m = re.exec(String(content || ""))) !== null) {
    const path = m[2].trim();
    const score = m[3].trim();
    const excerpt = m[4].trim();
    body.push(`<div class="typed-search-item"><div class="search-path">${escapeHtml(path)}</div><div class="search-score">score ${escapeHtml(score)}</div><div class="search-excerpt">${escapeHtml(excerpt)}</div><div class="typed-search-actions"><button type="button" class="typed-action-btn" onclick="openWorkspaceFile(${JSON.stringify(path)})">Open file</button><button type="button" class="typed-action-btn" onclick="stageWorkspaceFile(${JSON.stringify(path)})">Stage file</button></div></div>`);
  }
  if (body.length === 0) return renderTerminalBlock(content);
  const header = headerMatch ? `<div class="typed-block-meta">${escapeHtml(headerMatch[1])}</div>` : "";
  return `${header}<div class="typed-search-list">${body.join("")}</div>`;
}

export function renderWebBlock(content) {
  return `<div class="typed-web-block">${escapeHtml(content || "")}</div>`;
}

export function renderPatchArgumentDiff(rawArgs) {
  try {
    const args = JSON.parse(rawArgs || "{}");
    const path = args.path || "patch";
    const searchLines = String(args.search || "").split(/\r?\n/);
    const replaceLines = String(args.replace || "").split(/\r?\n/);
    const previewLimit = 24;
    const del = searchLines.slice(0, previewLimit).map((line) => `<span class="typed-diff-line typed-diff-del">- ${escapeHtml(line)}</span>`).join("");
    const add = replaceLines.slice(0, previewLimit).map((line) => `<span class="typed-diff-line typed-diff-add">+ ${escapeHtml(line)}</span>`).join("");
    const more = (searchLines.length > previewLimit || replaceLines.length > previewLimit)
      ? `<div class="typed-block-meta">Diff preview truncated for readability.</div>` : "";
    return `<div class="typed-diff-block"><div class="typed-diff-header">Patch preview · ${escapeHtml(path)}</div>${del}${add}${more}</div>`;
  } catch {
    return `<pre class="overflow-x-auto whitespace-pre-wrap text-slate-300 font-mono text-[11px]">${escapeHtml(rawArgs || "")}</pre>`;
  }
}

export function renderToolCallArgumentsBody(tc) {
  const name = tc && tc.function ? tc.function.name : "tool";
  const rawArgs = tc && tc.function ? tc.function.arguments || "" : "";
  if (name === "patch_file") {
    return renderPatchArgumentDiff(rawArgs);
  }
  if (name === "write_file") {
    try {
      const args = JSON.parse(rawArgs || "{}");
      const content = String(args.content || "");
      const rows = content.split(/\r?\n/).slice(0, 24).map((line, idx) => `<div class="typed-read-row"><span class="typed-read-line">${idx + 1}</span><span class="typed-read-text">${escapeHtml(line)}</span></div>`).join("");
      const more = content.split(/\r?\n/).length > 24 ? `<div class="typed-block-meta">Write preview truncated for readability.</div>` : "";
      return `<div class="typed-read-block">${rows}</div>${more}`;
    } catch {
      return `<pre class="overflow-x-auto whitespace-pre-wrap text-slate-300 font-mono text-[11px]">${escapeHtml(rawArgs)}</pre>`;
    }
  }
  return `<pre class="overflow-x-auto whitespace-pre-wrap text-slate-300 font-mono text-[11px]">${escapeHtml(rawArgs)}</pre>`;
}

export function renderToolBody(name, content) {
  const tool = String(name || "");
  const text = String(content || "");
  if (tool === "execute_command") return renderTerminalBlock(text);
  if (tool === "read_file" || tool === "read_spill") return renderReadBlock(text);
  if (tool === "bm25_search") return renderSearchBlock(text);
  if (tool === "web_search" || tool === "web_fetch") return renderWebBlock(text);
  return renderTerminalBlock(text);
}
