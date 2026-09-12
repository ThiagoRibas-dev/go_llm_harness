#!/usr/bin/env node
/*
 * HTML linter for the embedded GoHarness web console.
 *
 * This is a dependency-free static checker (pure Node, no npm install) that
 * validates the single-file SPA at src/web/index.html. It exists to catch the
 * exact class of bug that bit us in the settings modal: mismatched/overlapping
 * tags, duplicate element IDs, and onclick/onchange handlers that reference
 * undefined JavaScript functions.
 *
 * It intentionally does NOT try to be a full HTML5 validator. It checks the
 * structural invariants that have actually caused regressions.
 *
 * Exit code is non-zero if any errors are found.
 */

"use strict";

process.exitCode = 0;

const fs = require("fs");
const path = require("path");

// Resolve the repo root by walking up from cwd until we find go.mod, falling
// back to the script's own location (so it works whether invoked as
// `node scripts/lint-html.js`, `npm run lint`, or from CI).
function findRepoRoot(start) {
  let dir = path.resolve(start);
  for (let i = 0; i < 8; i++) {
    if (fs.existsSync(path.join(dir, "go.mod"))) return dir;
    const parent = path.dirname(dir);
    if (parent === dir) break;
    dir = parent;
  }
  return path.resolve(__dirname, "..");
}
const REPO_ROOT = findRepoRoot(process.cwd());
const HTML_PATH = path.join(REPO_ROOT, "src", "web", "index.html");

if (!fs.existsSync(HTML_PATH)) {
  console.error("✖ Could not find src/web/index.html (repo root: " + REPO_ROOT + ")");
  process.exit(1);
}

function resolveHtmlIncludes(filePath, depth = 0, stack = new Set()) {
  if (depth > 32) {
    throw new Error("HTML include depth exceeded while resolving " + filePath);
  }
  const clean = path.resolve(filePath);
  if (stack.has(clean)) {
    throw new Error("Cyclic HTML include detected at " + clean);
  }
  stack.add(clean);
  let source = fs.readFileSync(clean, "utf8");
  source = source.replace(/<!--#include\s+file="([^"]+)"\s*-->/g, (_, includeRef) => {
    const includePath = path.resolve(path.dirname(clean), includeRef);
    return resolveHtmlIncludes(includePath, depth + 1, stack);
  });
  stack.delete(clean);
  return source;
}

function fail(msg) {
  console.error("✖ " + msg);
  process.exitCode = 1;
}
function ok(msg) {
  console.log("✓ " + msg);
}

let html = "";
try {
  html = resolveHtmlIncludes(HTML_PATH);
} catch (err) {
  console.error("✖ Failed to resolve HTML includes: " + err.message);
  process.exit(1);
}

// Strip comments and <script>/<style> contents so we don't false-positive on
// IDs/HTML that live inside JS template literals.
const noComments = html.replace(/<!--[\s\S]*?-->/g, "");
const noScripts = noComments
  .replace(/<script\b[^>]*>[\s\S]*?<\/script>/gi, "")
  .replace(/<style\b[^>]*>[\s\S]*?<\/style>/gi, "");

// The inline <script> bodies are extracted separately for syntax/handler checks.
const commentsStripped = noComments;

const scriptBodies = [];
for (const m of commentsStripped.matchAll(/<script\b([^>]*)>([\s\S]*?)<\/script>/gi)) {
  const attrs = m[1] || "";
  const body = m[2] || "";
  const srcMatch = attrs.match(/\bsrc=["']([^"']+)["']/i);
  const typeMatch = attrs.match(/\btype=["']([^"']+)["']/i);
  const kind = typeMatch && /module/i.test(typeMatch[1]) ? "module" : "script";
  if (srcMatch) {
    scriptBodies.push({ kind, src: srcMatch[1], body: "" });
    continue;
  }
  if (!body.trim()) continue;
  scriptBodies.push({ kind, src: null, body });
}

function walkJsFiles(dir, out) {
  if (!fs.existsSync(dir)) return;
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) walkJsFiles(full, out);
    else if (entry.isFile() && entry.name.endsWith(".js")) out.push(full);
  }
}

const externalJsFiles = [];
const jsRoot = path.join(REPO_ROOT, "src", "web", "js");
walkJsFiles(jsRoot, externalJsFiles);
for (const file of externalJsFiles) {
  scriptBodies.push({ kind: "module", src: file, body: fs.readFileSync(file, "utf8") });
}

// ---------------------------------------------------------------------------
// 2. Tag-balance check. Void elements don't need a closing tag.
// ---------------------------------------------------------------------------
const VOID_TAGS = new Set([
  "area", "base", "br", "col", "embed", "hr", "img", "input",
  "link", "meta", "param", "source", "track", "wbr",
]);

const tagRe = /<\/?([a-zA-Z][a-zA-Z0-9-]*)\b([^>]*?)(\/?)>/g;
const stack = [];
let tagCount = 0;
let inScript = false, inStyle = false;
let m;
const tagRe2 = /<\/?([a-zA-Z][a-zA-Z0-9-]*)\b([^>]*?)(\/?)>/g;
while ((m = tagRe2.exec(noScripts)) !== null) {
  const full = m[0];
  const name = m[1].toLowerCase();
  const attrs = m[2];
  const selfClose = m[3] === "/";

  if (name === "script") {
    if (full.startsWith("</")) inScript = false;
    else inScript = true;
    continue;
  }
  if (name === "style") {
    if (full.startsWith("</")) inStyle = false;
    else inStyle = true;
    continue;
  }
  if (inScript || inStyle) continue;

  tagCount++;

  if (full.startsWith("</")) {
    // Closing tag
    if (stack.length === 0) {
      fail(`Unexpected closing </${name}> with no matching open tag (at char ${m.index})`);
      continue;
    }
    const top = stack[stack.length - 1];
    if (top.name !== name) {
      fail(
        `Mismatched tags: expected </${top.name}> but found </${name}> ` +
        `(open <${top.name}> at line ${top.line}).`
      );
      // Try to recover by popping until we find a match.
      while (stack.length && stack[stack.length - 1].name !== name) stack.pop();
      if (stack.length) stack.pop();
    } else {
      stack.pop();
    }
  } else if (!selfClose && !VOID_TAGS.has(name)) {
    // Opening tag
    const line = noScripts.slice(0, m.index).split("\n").length;
    stack.push({ name, line });
  }
}
if (stack.length > 0) {
  const open = stack.map(s => `<${s.name}> (line ${s.line})`).join(", ");
  fail("Unclosed tags: " + open);
} else if (process.exitCode === 0) {
  ok(`tag balance (${tagCount} tags)`);
}

// ---------------------------------------------------------------------------
// 3. Duplicate IDs
// ---------------------------------------------------------------------------
const ids = new Map();
const idRe = /\bid=["']([^"']+)["']/g;
while ((m = idRe.exec(noScripts)) !== null) {
  const id = m[1];
  if (ids.has(id)) {
    fail(`Duplicate id="${id}" (first seen, and again at char ${m.index})`);
  } else {
    ids.set(id, m.index);
  }
}
if (process.exitCode === 0) ok(`unique element IDs (${ids.size})`);

// ---------------------------------------------------------------------------
// 4. Inline event handlers reference functions that exist in the inline JS.
//    Catches typos like onclick="saveSetings()" or renamed handlers.
// ---------------------------------------------------------------------------
const definedFns = new Set();
for (const script of scriptBodies) {
  const body = script.body;
  // function foo(  /  async function foo( / export function foo(
  for (const fm of body.matchAll(/(?:export\s+)?(?:async\s+)?function\s+([A-Za-z_$][\w$]*)\s*\(/g)) {
    definedFns.add(fm[1]);
  }
  // const foo = () =>  /  const foo = function / export const foo =
  for (const fm of body.matchAll(/(?:export\s+)?(?:const|let|var)\s+([A-Za-z_$][\w$]*)\s*=\s*(?:async\s*)?(?:function|\()/g)) {
    definedFns.add(fm[1]);
  }
}

const handlerRe = /\b(on[a-z]+)\s*=\s*["']([^"']+)["']/gi;
let handlerCount = 0;
while ((m = handlerRe.exec(noScripts)) !== null) {
  const attr = m[1];
  const code = m[2].trim();
  // Only check the simple form: a direct function call "fn(...)" or "fn();".
  const callMatch = code.match(/^([A-Za-z_$][\w$]*)\s*\(/);
  if (!callMatch) continue;
  handlerCount++;
  const fn = callMatch[1];
  // Allow common globals / namespaced calls we know about.
  const builtins = new Set([
    "confirm", "alert", "prompt", "fetch", "console", "JSON", "document",
    "window", "location", "setTimeout", "setInterval", "event",
  ]);
  if (builtins.has(fn) || definedFns.has(fn)) continue;
  fail(`${attr} calls undefined function "${fn}()" in: ${code.slice(0, 60)}`);
}
if (process.exitCode === 0) ok(`inline event handlers resolve (${handlerCount})`);

// ---------------------------------------------------------------------------
// 5. Every href/src that points at a local anchor or asset is sane.
// ---------------------------------------------------------------------------
// (We don't validate external URLs; just flag obvious broken #no-such-id.)
const hrefRe = /\bhref=["']#([^"']+)["']/g;
while ((m = hrefRe.exec(noScripts)) !== null) {
  const target = m[1];
  if (!ids.has(target)) {
    fail(`href="#${target}" points at a missing element id`);
  }
}

// ---------------------------------------------------------------------------
// 6. Inline JS syntax check via `new Function` (parses without executing).
// ---------------------------------------------------------------------------
scriptBodies.forEach((script, i) => {
  try {
    let source = script.body;
    if (script.kind === "module") {
      source = source
        .replace(/^\s*import\s+[^;]+;\s*$/gm, "")
        .replace(/^\s*export\s+/gm, "");
    }
    // Wrapping in a function parses the body without running it. For modules
    // we strip import/export syntax first so this remains dependency-free.
    new Function(source);
  } catch (err) {
    const label = script.src ? path.relative(REPO_ROOT, script.src) : `inline <script> #${i + 1}`;
    fail(`${label} has a syntax error: ${err.message}`);
  }
});
if (process.exitCode === 0) ok(`script/module syntax (${scriptBodies.length} block${scriptBodies.length === 1 ? "" : "s"})`);

// ---------------------------------------------------------------------------
// Done
// ---------------------------------------------------------------------------
if (process.exitCode === 0) {
  console.log("\nAll HTML/JS checks passed.");
} else {
  console.error("\nHTML/JS lint found problems.");
  process.exit(1);
}
