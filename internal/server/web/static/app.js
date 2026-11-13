// trac2gitlab wizard. Vanilla JS, no external dependencies.
"use strict";

// ---- Icons (Lucide, ISC licence) ----
const ICONS = {
  "arrow-right-left": '<path d="m16 3 4 4-4 4"/><path d="M20 7H4"/><path d="m8 21-4-4 4-4"/><path d="M4 17h16"/>',
  "arrow-right": '<path d="M5 12h14"/><path d="m12 5 7 7-7 7"/>',
  "arrow-left": '<path d="m12 19-7-7 7-7"/><path d="M19 12H5"/>',
  "check": '<path d="M20 6 9 17l-5-5"/>',
  "lock": '<rect width="18" height="11" x="3" y="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>',
  "folder-git": '<path d="M4 20h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.93a2 2 0 0 1-1.66-.9l-.82-1.2A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13c0 1.1.9 2 2 2Z"/><circle cx="12" cy="13" r="2"/><path d="M14 13h3"/><path d="M7 13h3"/>',
  "chevrons-up-down": '<path d="m7 15 5 5 5-5"/><path d="m7 9 5-5 5 5"/>',
  "chevron-right": '<path d="m9 18 6-6-6-6"/>',
  "shield-check": '<path d="M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67-.01C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1z"/><path d="m9 12 2 2 4-4"/>',
  "shield-alert": '<path d="M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67-.01C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1z"/><path d="M12 8v4"/><path d="M12 16h.01"/>',
  "plus": '<path d="M5 12h14"/><path d="M12 5v14"/>',
  "database": '<ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5V19A9 3 0 0 0 21 19V5"/><path d="M3 12A9 3 0 0 0 21 12"/>',
  "git-merge": '<circle cx="18" cy="18" r="3"/><circle cx="6" cy="6" r="3"/><path d="M6 21V9a9 9 0 0 0 9 9"/>',
  "save": '<path d="M15.2 3a2 2 0 0 1 1.4.6l3.8 3.8a2 2 0 0 1 .6 1.4V19a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2z"/><path d="M17 21v-7a1 1 0 0 0-1-1H8a1 1 0 0 0-1 1v7"/><path d="M7 3v4a1 1 0 0 0 1 1h7"/>',
  "plug": '<path d="M12 22v-5"/><path d="M9 8V2"/><path d="M15 8V2"/><path d="M18 8v5a4 4 0 0 1-4 4h-4a4 4 0 0 1-4-4V8Z"/>',
  "download": '<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><path d="m7 10 5 5 5-5"/><path d="M12 15V3"/>',
  "info": '<circle cx="12" cy="12" r="10"/><path d="M12 16v-4"/><path d="M12 8h.01"/>',
  "search": '<circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/>',
  "user-check": '<path d="m16 11 2 2 4-4"/><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/>',
  "refresh": '<path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8"/><path d="M21 3v5h-5"/><path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16"/><path d="M8 16H3v5"/>',
  "play": '<polygon points="6 3 20 12 6 21 6 3"/>',
  "copy": '<rect width="14" height="14" x="8" y="8" rx="2"/><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/>',
  "circle-check": '<circle cx="12" cy="12" r="10"/><path d="m9 12 2 2 4-4"/>',
  "circle-x": '<circle cx="12" cy="12" r="10"/><path d="m15 9-6 6"/><path d="m9 9 6 6"/>',
  "triangle-alert": '<path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3"/><path d="M12 9v4"/><path d="M12 17h.01"/>',
  "loader": '<path d="M21 12a9 9 0 1 1-6.22-8.56"/>',
};

function icon(name, size) {
  const cls = "icon" + (size ? " " + size : "") + (name === "loader" ? " spin" : "");
  return '<svg class="' + cls + '" viewBox="0 0 24 24" aria-hidden="true">' + (ICONS[name] || "") + "</svg>";
}

function hydrateIcons(root) {
  root.querySelectorAll("[data-icon]").forEach(el => {
    el.insertAdjacentHTML("afterbegin", icon(el.dataset.icon, el.dataset.size));
    el.removeAttribute("data-icon");
  });
}

const $ = id => document.getElementById(id);
const esc = s => String(s ?? "").replace(/[&<>"']/g, c => ({"&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;"}[c]));
const q = () => "?project=" + encodeURIComponent(app.project || "");

async function getJSON(url) {
  const res = await fetch(url);
  if (!res.ok) throw new Error(await errText(res));
  return res.json();
}

// errText reads an error response that may be JSON or plain text
// (http.Error writes plain text).
async function errText(res) {
  const raw = await res.text();
  try {
    const body = JSON.parse(raw);
    return body.error || body.message || raw;
  } catch {
    return raw.trim();
  }
}

function netMessage(e) {
  return e instanceof TypeError && e.message === "Failed to fetch"
    ? "No response. Is trac2gitlab serve still running?"
    : e.message;
}

async function busy(btn, label, fn) {
  const html = btn.innerHTML;
  btn.disabled = true;
  btn.setAttribute("aria-busy", "true");
  btn.innerHTML = icon("loader") + esc(label);
  try {
    return await fn();
  } finally {
    btn.innerHTML = html;
    btn.disabled = false;
    btn.removeAttribute("aria-busy");
  }
}

// ---- Status vocabulary ----
function kindOf(verdict) {
  const v = (verdict || "").toUpperCase();
  if (/BLOCKED|NOT READY|FAIL/.test(v)) return "fail";
  if (/WARN/.test(v)) return "warn";
  if (/READY|PASS/.test(v)) return "ok";
  return "";
}

const KIND_ICON = {ok: "circle-check", warn: "triangle-alert", fail: "circle-x", run: "loader", "": "info"};

function pill(text, kind) {
  const ic = {ok: "check", warn: "triangle-alert", fail: "circle-x", run: "loader"}[kind];
  return '<span class="pill ' + (kind || "") + '">' + (ic ? icon(ic) : "") + esc(text) + "</span>";
}

function setBanner(el, kind, title, detail) {
  el.className = "banner " + (kind || "");
  el.innerHTML = icon(KIND_ICON[kind || ""], "lg") + "<div><strong>" + esc(title) + "</strong>" +
    (detail ? "<span>" + esc(detail) + "</span>" : "") + "</div>";
}

function setNote(el, kind, text) {
  el.hidden = false;
  el.className = "note " + (kind || "");
  el.innerHTML = icon(KIND_ICON[kind || ""]) + "<span>" + esc(text) + "</span>";
}

function setPill(el, text, kind) {
  el.outerHTML = pill(text, kind).replace("<span ", '<span id="' + el.id + '" ');
}

function titleCase(v) {
  const s = (v || "").toLowerCase();
  return s.charAt(0).toUpperCase() + s.slice(1);
}

// ---- Steps ----
const STEPS = [
  {id: "connect", title: "Connect", risk: "Local config", next: "Export reads tickets, wiki, milestones and attachments from Trac into data/. Trac is never modified."},
  {id: "export", title: "Export", risk: "Reads Trac", next: "Discover authors from the export, then map them to GitLab users in mappings/users.yaml."},
  {id: "identities", title: "Identities", risk: "Local mapping", next: "Preflight runs read-only checks against Trac and GitLab. Nothing is written."},
  {id: "preflight", title: "Preflight", risk: "Read-only checks", next: "The dry-run plans every phase and shows counts. Nothing is written to GitLab."},
  {id: "dryrun", title: "Dry-run", risk: "No GitLab writes", next: "Execute writes to GitLab after you type the project path to confirm."},
  {id: "execute", title: "Execute", risk: "Writes to GitLab", writes: true, next: "Restore historical authors with the patch script, then validate source against target."},
  {id: "finish", title: "Patch & validate", risk: "Runs on GitLab host", next: "Review reports/validation.md and the remaining warnings. Trac can then be archived."},
];

const app = {
  project: null,
  step: "connect",
  projects: [],
  conn: {},
  preflight: null,
  users: null,
  status: null,
  job: null,
  tested: {trac: null, gitlab: null},
  patchApply: false,
  userFilter: "all",
};

function stepState() {
  const pre = kindOf(app.preflight?.verdict);
  const phases = app.status?.Phases || {};
  const ran = Object.keys(phases).length > 0;
  const jobDone = kind => app.job && app.job.kind === kind && app.job.status === "completed";
  const executed = ran && app.status && !app.status.DryRun && app.status.Status === "completed";
  const p = app.projects.find(x => x.name === app.project) || {};
  return {
    connect: !!(app.conn.tracBaseUrl && app.conn.gitlabUrl && app.conn.gitlabProject),
    export: !!p.hasExport || jobDone("export"),
    identities: !!app.users?.exists && !!app.users?.users?.length && app.users.users.every(u => u.resolved),
    preflight: pre === "ok" || pre === "warn",
    dryrun: (ran && app.status?.DryRun && app.status.Status === "completed") || jobDone("dry-run") || executed,
    execute: executed || jobDone("execute"),
    finish: !!p.hasValidation,
    preflightOk: pre === "ok" || pre === "warn",
    preflightKind: app.preflight?.exists ? pre : "",
    unresolved: (app.users?.users || []).filter(u => !u.resolved).length,
    connFail: (app.tested.trac ?? preflightService("trac")) === false || (app.tested.gitlab ?? preflightService("gitlab")) === false,
  };
}

function renderSteps() {
  const st = stepState();
  const nav = $("steps");
  nav.innerHTML = "";
  STEPS.forEach((s, i) => {
    const locked = (s.id === "execute" && !st.preflightOk) || (s.id === "finish" && !st.execute && !st.finish);
    let state = st[s.id] ? "done" : "";
    let risk = s.risk;
    if (s.id === "preflight" && st.preflightKind === "fail") { state = "fail"; risk = "NOT READY"; }
    if (s.id === "preflight" && st.preflightKind === "warn") { state = "warn"; risk = "READY WITH WARNINGS"; }
    if (s.id === "connect" && st.connFail) { state = "fail"; risk = "Connection failing"; }
    if (s.id === "identities" && st.unresolved && app.users?.exists) { state = "warn"; risk = st.unresolved + " unresolved"; }
    const isCur = s.id === app.step;
    const b = document.createElement("button");
    b.type = "button";
    b.className = "step" + (s.writes ? " writes" : "") + (isCur ? " current" : "") + (state ? " " + state : "") + (locked && !state ? " locked" : "");
    if (isCur) b.setAttribute("aria-current", "step");
    const badge = state === "done" ? icon("check", "sm") : state === "fail" ? icon("circle-x", "sm") : state === "warn" ? icon("triangle-alert", "sm") : locked ? icon("lock", "sm") : String(i + 1);
    b.innerHTML = '<span class="badge">' + badge + '</span><span><span class="label">' + esc(s.title) +
      '</span><span class="risk">' + esc(risk) + "</span></span>";
    b.addEventListener("click", () => showStep(s.id));
    nav.appendChild(b);
  });
  applyPrimaries(st);
  const cur = nav.querySelector(".current");
  if (cur && nav.scrollWidth > nav.clientWidth) cur.scrollIntoView({inline: "center", block: "nearest"});
}

// One primary action per screen: the run button leads until its step passes, then Continue leads.
function applyPrimaries(st) {
  const p = st || stepState();
  const passed = {export: p.export, identities: p.identities, preflight: p.preflightOk, dryrun: p.dryrun};
  const runs = {export: "runExport", identities: "validateUsers", preflight: "runPreflight", dryrun: "runDryRun"};
  for (const [pane, ok] of Object.entries(passed)) {
    $(runs[pane]).classList.toggle("primary", !ok);
    const cont = document.querySelector("#pane-" + pane + " .actions [data-goto]:last-child");
    if (cont) cont.classList.toggle("primary", !!ok);
  }
}

function showStep(id) {
  app.step = id;
  for (const s of STEPS) $("pane-" + s.id).hidden = s.id !== id;
  $("sNext").textContent = STEPS.find(s => s.id === id).next;
  renderSteps();
  if (id === "execute") renderGate();
  try { localStorage.setItem("t2g.step." + app.project, id); } catch { /* storage unavailable */ }
  history.replaceState(null, "", "#/" + encodeURIComponent(app.project) + "/" + id);
  window.scrollTo({top: 0});
}

// ---- Views ----
function showHome() {
  app.project = null;
  $("viewHome").hidden = false;
  $("viewWizard").hidden = true;
  $("projectSelect").value = "";
  history.replaceState(null, "", location.pathname);
  setChips();
  renderHome();
}

function showWizard() {
  $("viewHome").hidden = true;
  $("viewWizard").hidden = false;
}

// ---- Projects ----
async function loadProjects() {
  app.projects = (await getJSON("/api/projects")) || [];
  const sel = $("projectSelect");
  sel.innerHTML = '<option value="">All migrations</option>' +
    app.projects.map(p => '<option value="' + esc(p.name) + '">' + esc(p.name) + "</option>").join("");
  sel.value = app.project || "";
}

async function projectOverview(p) {
  const pq = "?project=" + encodeURIComponent(p.name);
  const [conn, pre, job] = await Promise.all([
    getJSON("/api/project/connection" + pq).catch(() => ({})),
    getJSON("/api/preflight" + pq).catch(() => ({})),
    getJSON("/api/job" + pq).catch(() => ({})),
  ]);
  let step = 0, status = "Not configured", kind = "";
  if (conn.tracBaseUrl && conn.gitlabUrl) { step = 1; status = "Connected"; }
  if (p.hasPreflight) { step = 3; status = pre.verdict || "Preflight run"; kind = kindOf(pre.verdict); }
  if (p.hasState) { step = Math.max(step, 4); }
  if (p.hasValidation) { step = 6; status = "Validated"; kind = "ok"; }
  if (job && job.status === "running") { status = titleCase(job.kind) + " running"; kind = "run"; }
  else if (job && job.status === "failed") { status = titleCase(job.kind) + " failed"; kind = "fail"; }
  return {p, conn, step, status, kind};
}

async function renderHome() {
  const body = $("projectRows");
  if (!app.projects.length) {
    body.innerHTML = '<tr><td colspan="5" class="empty">No migrations yet. Press <b>New migration</b>, or run <code>trac2gitlab init --project-dir projects/&lt;name&gt;</code>.</td></tr>';
    $("homeStats").innerHTML = "";
    return;
  }
  const rows = await Promise.all(app.projects.map(projectOverview));
  const count = k => rows.filter(r => r.kind === k).length;
  $("homeStats").innerHTML = [[rows.length, rows.length === 1 ? "project" : "projects"], [count("warn") + count("fail"), "need attention"], [count("run"), "running"], [rows.filter(r => r.step === 6).length, "validated"]]
    .map(([n, l]) => "<div><b>" + n + "</b>" + l + "</div>").join("");
  body.innerHTML = rows.map(r => {
    const segs = STEPS.map((s, i) => '<i class="' + (i < r.step ? "done" : i === r.step ? (r.kind || "") : "") + '"></i>').join("");
    const target = r.conn.gitlabUrl ? r.conn.gitlabUrl.replace(/^https?:\/\//, "") + (r.conn.gitlabProject ? " · " + r.conn.gitlabProject : "") : "not set";
    return '<tr class="clickable ' + (r.kind === "warn" || r.kind === "fail" ? r.kind : "") + '" data-name="' + esc(r.p.name) + '">' +
      '<td><span class="proj-name">' + icon("folder-git") + esc(r.p.name) + "</span></td>" +
      '<td class="detail mono">' + esc(target) + "</td>" +
      '<td><div class="segments">' + segs + '</div><span class="muted small">' + (r.step + 1) + " of 7 · " + esc(STEPS[r.step].title) + "</span></td>" +
      "<td>" + pill(r.status, r.kind) + "</td>" +
      '<td class="act"><button class="btn link" type="button">Open' + icon("chevron-right") + "</button></td></tr>";
  }).join("");
  body.querySelectorAll("tr[data-name]").forEach(tr => tr.addEventListener("click", () => select(tr.dataset.name)));
}

async function createProject(ev) {
  ev.preventDefault();
  const input = $("newProject");
  const name = input.value.trim();
  if (!name) return;
  $("createError").hidden = true;
  try {
    const res = await fetch("/api/projects", {method: "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify({name})});
    if (!res.ok) throw new Error(await errText(res));
    input.value = "";
    $("createForm").hidden = true;
    await loadProjects();
    await select(name, "connect");
  } catch (e) {
    setNote($("createError"), "fail", "Could not create project: " + netMessage(e));
  }
}

// ---- Connection ----
function preflightService(k) {
  const checks = app.preflight?.checks || [];
  const mine = checks.filter(c => new RegExp(k === "trac" ? "trac" : "gitlab", "i").test(c.name));
  if (!mine.length) return null;
  return mine.every(c => c.state !== "fail");
}

function setChips() {
  for (const k of ["trac", "gitlab"]) {
    const t = app.tested[k] ?? preflightService(k);
    const el = $(k === "trac" ? "chipTrac" : "chipGitlab");
    const name = k === "trac" ? "Trac" : "GitLab";
    el.className = "chip" + (t === true ? " ok" : t === false ? " fail" : "");
    const src = app.tested[k] === null && t !== null ? " (preflight)" : "";
    el.textContent = name + (t === true ? " reachable" : t === false ? " failing" : " not tested") + src;
    el.hidden = !app.project;
    const p = $(k === "trac" ? "tracPill" : "gitlabPill");
    setPill(p, t === true ? "Connected" : t === false ? "Failing" : "Not tested", t === true ? "ok" : t === false ? "fail" : "");
  }
  renderConnBanner();
  if (app.project) renderSteps();
}

function renderConnBanner() {
  const el = $("connVerdict");
  if (!el || !app.project) return;
  const t = app.tested.trac ?? preflightService("trac"), g = app.tested.gitlab ?? preflightService("gitlab");
  const configured = !!(app.conn.tracBaseUrl && app.conn.gitlabUrl && app.conn.gitlabProject);
  if (!configured) setBanner(el, "", "Connection incomplete", "Fill in the Trac URL, GitLab URL and GitLab project, then save and test each.");
  else if (t === false || g === false) setBanner(el, "fail", (t === false && g === false ? "Trac and GitLab are" : t === false ? "Trac is" : "GitLab is") + " failing", "Check the URL and credentials below, save, then test again.");
  else if (t && g) setBanner(el, "ok", "Both connections work", "Continue to the export.");
  else setBanner(el, "", "Saved, not tested yet", "Test Trac and GitLab to confirm the saved settings.");
}

const STORED = "Stored. Leave blank to keep.";

async function loadConnection() {
  const body = await getJSON("/api/project/connection" + q());
  app.conn = body;
  $("fTracUrl").value = body.tracBaseUrl || "";
  $("fTracRpc").value = body.tracRpcPath || "";
  $("fGitlabUrl").value = body.gitlabUrl || "";
  $("fGitlabProj").value = body.gitlabProject || "";
  $("fTracUser").value = body.tracUser || "";
  $("fTracHtUser").value = body.tracGateUser || "";
  $("fGitlabHtUser").value = body.gitlabGateUser || "";
  const tracGate = !!body.tracGateUser || body.hasTracGatePassword;
  const gitlabGate = !!body.gitlabGateUser || body.hasGitlabGatePassword;
  $("tracHtaccess").checked = tracGate;
  $("tracHtaccessFields").hidden = !tracGate;
  $("gitlabHtaccess").checked = gitlabGate;
  $("gitlabHtaccessFields").hidden = !gitlabGate;
  for (const [id, has] of [["fTracPass", body.hasTracPassword], ["fTracHtPass", body.hasTracGatePassword], ["fGitlabHtPass", body.hasGitlabGatePassword], ["fGitlabToken", body.hasGitlabToken]]) {
    $(id).value = "";
    $(id).placeholder = has ? STORED : "Only sent, never shown";
  }
  $("hTracPass").textContent = body.hasTracPassword ? "Stored in .env" : "Saved to .env";
  const src = (body.tracBaseUrl || "").replace(/^https?:\/\//, "");
  $("sSource").textContent = src || "not set";
  $("sSource").title = body.tracBaseUrl || "";
  const tgt = body.gitlabUrl ? body.gitlabUrl.replace(/^https?:\/\//, "") + (body.gitlabProject ? " · " + body.gitlabProject : "") : "not set";
  $("sTarget").textContent = tgt;
  $("sTarget").title = tgt;
  $("confirmWord").textContent = body.gitlabProject || "the project path";
  renderConnBanner();
}

async function testConnection(kind) {
  if (!app.project) return;
  const el = $(kind === "trac" ? "tracTest" : "gitlabTest");
  const btn = $(kind === "trac" ? "testTrac" : "testGitlab");
  await busy(btn, "Testing…", async () => {
    try {
      const res = await fetch("/api/test/" + kind + q(), {method: "POST"});
      if (!res.ok) throw new Error(await errText(res));
      const body = await res.json();
      app.tested[kind] = !!body.ok;
      setNote(el, body.ok ? "ok" : "fail", body.message);
    } catch (e) {
      app.tested[kind] = false;
      setNote(el, "fail", "Test failed: " + netMessage(e));
    }
    setChips();
  });
}

async function saveService(kind) {
  const el = $(kind === "trac" ? "tracTest" : "gitlabTest");
  if (!app.project) return;
  const v = id => $(id).value;
  const payload = kind === "trac" ? {
    tracBaseUrl: v("fTracUrl"), tracRpcPath: v("fTracRpc"), tracUser: v("fTracUser"),
    tracPassword: v("fTracPass"), tracGateUser: v("fTracHtUser"), tracGatePass: v("fTracHtPass"),
  } : {
    gitlabUrl: v("fGitlabUrl"), gitlabToken: v("fGitlabToken"), gitlabProject: v("fGitlabProj"),
    gitlabGateUser: v("fGitlabHtUser"), gitlabGatePass: v("fGitlabHtPass"),
  };
  await busy($(kind === "trac" ? "saveTrac" : "saveGitlab"), "Saving…", async () => {
    try {
      const res = await fetch("/api/project/connection" + q(), {
        method: "PUT", headers: {"Content-Type": "application/json"}, body: JSON.stringify(payload),
      });
      if (!res.ok) throw new Error(await errText(res));
      await loadConnection();
      app.tested[kind] = null;
      setChips();
      setNote(el, "ok", (kind === "trac" ? "Trac" : "GitLab") + " connection saved to migration.yaml and .env. Test it next.");
      renderSteps();
    } catch (e) {
      setNote(el, "fail", "Save failed: " + netMessage(e));
    }
  });
}

// ---- Preflight ----
function renderPreflight() {
  const pre = app.preflight || {};
  const kind = kindOf(pre.verdict);
  $("sPreflight").innerHTML = pre.exists ? pill(pre.verdict || "UNKNOWN", kind) : '<span class="muted">Not run</span>';
  if (!pre.exists) {
    setBanner($("verdict"), "", "Not run yet", "Preflight gates every GitLab write. It needs mappings/users.yaml.");
  } else {
    const detail = {
      ok: "You can continue to the dry-run.",
      warn: "You can continue to the dry-run. Review the warnings first; they will not block execution.",
      fail: "Execute stays locked. Fix " + ((pre.checks || []).filter(c => c.state === "fail").map(c => c.name).join(", ") || "the failing checks") +
        " using the result in each row (usually the Connect settings or the token's role), then run preflight again.",
    }[kind] || "";
    setBanner($("verdict"), kind, pre.verdict || "UNKNOWN VERDICT", detail);
  }
  const checks = pre.checks || [];
  $("checks").innerHTML = checks.length ? checks.map(c => {
    const k = c.state === "pass" ? "ok" : c.state;
    return '<tr class="' + (k === "ok" ? "" : k) + '"><td><b style="font-weight:500">' + esc(c.name) + '</b></td><td class="detail">' + esc(c.detail) +
      "</td><td>" + pill(c.state === "pass" ? "Pass" : c.state === "warn" ? "Warning" : "Fail", k) + "</td></tr>";
  }).join("") : '<tr><td colspan="3" class="empty">' + (pre.exists ? "No structured checks cached. Run preflight to refresh them." : "No checks yet.") + "</td></tr>";
  $("report").textContent = pre.exists ? pre.raw : "No reports/preflight.md yet. Press Run preflight, or run: trac2gitlab preflight --project-dir projects/" + app.project;
}

async function loadPreflight() {
  app.preflight = await getJSON("/api/preflight" + q());
  renderPreflight();
  setChips();
}

async function runPreflight() {
  if (!app.project) return;
  await busy($("runPreflight"), "Running…", async () => {
    try {
      const res = await fetch("/api/preflight/run" + q(), {method: "POST"});
      if (!res.ok) throw new Error(await errText(res));
      await res.json();
      await loadPreflight();
      await loadProjects();
    } catch (e) {
      setBanner($("verdict"), "fail", "Preflight could not run", netMessage(e));
    }
    renderSteps();
  });
}

// ---- Identities ----
function renderUsers() {
  const body = app.users;
  const tbody = $("users");
  const users = body?.users || [];
  const unresolved = users.filter(u => !u.resolved);
  $("cAll").textContent = users.length;
  $("cUnres").textContent = unresolved.length;
  $("cMapped").textContent = users.length - unresolved.length;
  $("sUsers").textContent = body?.exists ? (users.length - unresolved.length) + " of " + users.length + " mapped" : "No mapping yet";
  if (!body?.exists) {
    tbody.innerHTML = '<tr><td colspan="4" class="empty">No mappings/users.yaml yet. Run the export, then press <b>Discover authors</b>.</td></tr>';
    return;
  }
  const list = users.filter(u => app.userFilter === "all" || (app.userFilter === "mapped") === !!u.resolved);
  tbody.innerHTML = list.length ? list.map(u => {
    const target = u.resolved ? '<span class="mono">@' + esc(u.gitlab) + "</span>" : '<span class="muted">' + (u.action && u.action !== "unresolved" ? esc(u.action) : "“Originally by” footer") + "</span>";
    return '<tr class="' + (u.resolved ? "" : "warn") + '"><td class="mono">' + esc(u.key) + '</td><td class="detail mono">' + esc((u.aliases || []).join(" · ") || "—") +
      "</td><td>" + target + "</td><td>" + pill(u.resolved ? "Mapped" : "Unresolved", u.resolved ? "ok" : "warn") + "</td></tr>";
  }).join("") : '<tr><td colspan="4" class="empty">Nothing in this filter.</td></tr>';
}

async function loadUsers() {
  app.users = await getJSON("/api/users" + q());
  if (app.users.error) setBanner($("usersVerdict"), "fail", "mappings/users.yaml does not parse", app.users.error);
  else if (!app.users.exists) setBanner($("usersVerdict"), "", "No mapping yet", "Run the export, then Discover authors.");
  else {
    const un = app.users.users.filter(u => !u.resolved).length;
    if (un) setBanner($("usersVerdict"), "warn", un + " of " + app.users.users.length + " authors unresolved", "Fill in their GitLab usernames in mappings/users.yaml and press Validate, or accept the “Originally by” footer.");
    else setBanner($("usersVerdict"), "ok", "All " + app.users.users.length + " authors mapped", "Press Validate to confirm they exist in GitLab.");
  }
  renderUsers();
}

async function runDiscover() {
  if (!app.project) return;
  await busy($("runDiscover"), "Discovering…", async () => {
    try {
      const res = await fetch("/api/discover/run" + q(), {method: "POST"});
      if (!res.ok) throw new Error(await errText(res));
      const body = await res.json();
      await loadUsers();
      setBanner($("usersVerdict"), "warn", "Wrote mappings/users.yaml",
        body.authors.length + " authors from " + body.tickets + " tickets and " + body.wiki_pages + " wiki pages. Fill in the GitLab usernames, then Validate.");
    } catch (e) {
      setBanner($("usersVerdict"), "fail", "Discover failed", netMessage(e));
    }
    renderSteps();
  });
}

async function validateUsers() {
  if (!app.project) return;
  await busy($("validateUsers"), "Validating…", async () => {
    try {
      const res = await fetch("/api/users/validate" + q(), {method: "POST"});
      if (!res.ok) throw new Error(await errText(res));
      const body = await res.json();
      if (body.users) app.users = {exists: true, users: body.users};
      renderUsers();
      if (body.error) setBanner($("usersVerdict"), "fail", "Validation failed", body.error);
      else if (body.ok) setBanner($("usersVerdict"), "ok", "All " + body.total + " identities resolved", "Every author keeps their GitLab identity.");
      else setBanner($("usersVerdict"), "warn", body.unresolved.length + " of " + body.total + " unresolved",
        body.unresolved.join(", ") + " will get an “Originally by …” footer and are skipped by the patch.");
    } catch (e) {
      setBanner($("usersVerdict"), "fail", "Validation failed", netMessage(e));
    }
    renderSteps();
  });
}

// ---- Jobs, state, plan ----
const PHASE_ORDER = ["milestones", "labels", "issues", "notes", "comments", "attachments", "wiki"];

function renderPlan() {
  const phases = app.status?.Phases || {};
  const names = Object.keys(phases).sort((a, b) => {
    const ia = PHASE_ORDER.indexOf(a), ib = PHASE_ORDER.indexOf(b);
    return (ia < 0 ? 99 : ia) - (ib < 0 ? 99 : ib) || a.localeCompare(b);
  });
  const mode = app.status?.DryRun ? "dry-run" : "execute";
  const html = names.length ? '<table class="table"><thead><tr><th>Phase</th><th class="num">Total</th><th class="num">Done</th><th class="num">Skipped</th><th class="num">Failed</th><th class="w-status">Status</th></tr></thead><tbody>' +
    names.map(n => {
      const p = phases[n];
      const k = p.Failed > 0 || p.Status === "failed" ? "fail" : p.Status === "completed" ? "ok" : p.Status === "running" ? "run" : "";
      const n0 = v => '<td class="num' + (v ? "" : " dim") + '">' + (v || 0).toLocaleString() + "</td>";
      return '<tr class="' + (k === "fail" ? "fail" : "") + '"><td><b style="font-weight:500">' + esc(titleCase(n)) + "</b>" +
        (p.Errors && p.Errors.length ? '<div class="muted small">' + esc(p.Errors.slice(0, 2).join(" · ")) + (p.Errors.length > 2 ? " · +" + (p.Errors.length - 2) + " more" : "") + "</div>" : "") +
        "</td>" + n0(p.Total) + n0(p.Completed) + n0(p.Skipped) + n0(p.Failed) + "<td>" + pill(titleCase(p.Status || "pending"), k) + "</td></tr>";
    }).join("") + '</tbody></table><div class="muted small" style="padding:10px 20px;border-top:1px solid var(--line)">From the last ' + mode + " in state/migration.yaml</div>"
    : '<div class="empty">No plan yet. <button class="btn link" type="button" data-goto-dry>Go to dry-run' + icon("arrow-right") + "</button></div>";
  document.querySelectorAll("[data-plan]").forEach(el => { el.innerHTML = html; });
  document.querySelectorAll("[data-goto-dry]").forEach(b => b.addEventListener("click", () => showStep("dryrun")));
  $("sCounts").innerHTML = names.length ? names.map(n => '<div class="stat"><span>' + esc(titleCase(n)) + '</span><b>' + (phases[n].Total || 0).toLocaleString() + "</b></div>").join("")
    : '<span class="muted">Counts appear after the dry-run.</span>';
}

async function refreshStatus() {
  app.status = await getJSON("/api/status" + q());
  renderPlan();
}

let pollTimer = null;

function ago(t) {
  const m = Math.round((Date.now() - new Date(t)) / 60000);
  if (!isFinite(m)) return "";
  return m < 1 ? "just now" : m < 60 ? m + " min ago" : m < 1440 ? Math.round(m / 60) + " h ago" : new Date(t).toLocaleDateString();
}

function renderJob() {
  const job = app.job;
  const logs = document.querySelectorAll("[data-log]");
  if (!job || job.exists === false) {
    $("sJob").textContent = "None in this session";
    logs.forEach(l => { l.textContent = "No job yet."; });
    return;
  }
  const kind = job.status === "completed" ? "ok" : job.status === "failed" ? "fail" : "run";
  $("sJob").textContent = titleCase(job.kind) + " " + job.status + (job.startedAt ? " · " + ago(job.startedAt) : "");
  logs.forEach(l => { l.textContent = (job.log || []).join("\n") || "…"; l.scrollTop = l.scrollHeight; });
  const title = titleCase(job.kind) + " " + job.status;
  const detail = job.error || (job.status === "running" ? "Job " + job.id + " is running. This page updates every 2 seconds." : "Job " + job.id);
  if (job.kind === "export") setBanner($("exportVerdict"), kind, title, detail);
  if (job.kind === "dry-run") setBanner($("jobStatus"), kind, title, detail);
  if (job.kind === "validate") setNote($("valVerdict"), kind, title + ". " + detail);
  if (job.kind === "execute") setNote($("executeVerdict"), kind, title + ". " + detail);
}

async function pollJob() {
  app.job = await getJSON("/api/job" + q());
  renderJob();
  if (!app.job || app.job.exists === false || app.job.status !== "running") {
    await refreshStatus();
    renderSteps();
    return true;
  }
  return false;
}

function startPolling() {
  if (pollTimer) clearInterval(pollTimer);
  const project = app.project;
  pollTimer = setInterval(async () => {
    if (app.project !== project) return clearInterval(pollTimer);
    try {
      if (await pollJob()) {
        clearInterval(pollTimer);
        if (app.job?.kind === "validate") await loadValidation();
        await loadProjects();
      }
    } catch { /* keep polling */ }
  }, 2000);
}

async function startRun(path, btn, onError) {
  if (!app.project) return;
  await busy(btn, "Starting…", async () => {
    try {
      const res = await fetch(path + q(), {method: "POST"});
      if (!res.ok) throw new Error(await errText(res));
      app.job = await res.json();
      renderJob();
      startPolling();
    } catch (e) {
      onError(netMessage(e));
    }
  });
}

// ---- Execute gate ----
function renderGate() {
  const st = stepState();
  const word = app.conn.gitlabProject || "";
  const target = (app.conn.gitlabUrl || "").replace(/^https?:\/\//, "");
  $("gateTitle").textContent = "Write to " + (target ? target + " / " : "") + (word || "GitLab");
  const total = Object.values(app.status?.Phases || {}).reduce((n, p) => n + (p.Total || 0), 0);
  const items = [
    [st.preflightOk, "Preflight: " + (app.preflight?.exists ? app.preflight.verdict : "not run")],
    [st.dryrun ? true : "warn", st.dryrun ? "Dry-run finished" + (total ? ": " + total.toLocaleString() + " objects planned" : "") : "No completed dry-run yet. Recommended before executing, not required."],
    [!!word, word ? "Target project is " + word : "No GitLab project set on the Connect step."],
  ];
  $("gateChecks").innerHTML = items.map(([ok, t]) => {
    const c = ok === "warn" ? "warn" : ok ? "ok" : "no";
    return '<li class="' + c + '">' + icon(c === "warn" ? "triangle-alert" : ok ? "circle-check" : "circle-x") + esc(t) + "</li>";
  }).join("");
  if (!st.preflightOk) setBanner($("executeBanner"), "fail", "Locked: preflight is " + (app.preflight?.exists ? app.preflight.verdict : "not run"), "Execute unlocks at READY or READY WITH WARNINGS.");
  else if (st.execute) setBanner($("executeBanner"), "ok", "Executed", "Continue to Patch & validate.");
  else setBanner($("executeBanner"), "warn", "Ready to execute", "Review the plan, then type the project path to confirm. This writes to GitLab.");
  updateExecuteButton();
}

function updateExecuteButton() {
  const st = stepState();
  const word = app.conn.gitlabProject || "";
  $("confirmExecute").disabled = !st.preflightOk;
  const typed = $("confirmExecute").value.trim();
  const running = app.job && app.job.status === "running";
  const can = st.preflightOk && word && typed === word && !running;
  $("runExecute").disabled = !can;
  $("execReason").textContent = running ? "A job is running." : !st.preflightOk ? "" : !word ? "Set the GitLab project first." : typed !== word ? "Type " + word + " to unlock." : "";
}

async function runExecute() {
  const word = app.conn.gitlabProject || "";
  if (!app.project || !word || $("confirmExecute").value.trim() !== word) return;
  const btn = $("runExecute");
  btn.disabled = true;
  try {
    const res = await fetch("/api/migrate/execute" + q(), {
      method: "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify({confirm: true}),
    });
    if (!res.ok) throw new Error(await errText(res));
    app.job = await res.json();
    $("confirmExecute").value = "";
    renderJob();
    startPolling();
  } catch (e) {
    setNote($("executeVerdict"), "fail", "Execute refused: " + netMessage(e));
  } finally {
    updateExecuteButton();
  }
}

// ---- Patch + validate ----
function setPatchMode(apply) {
  app.patchApply = apply;
  $("modeDry").setAttribute("aria-pressed", String(!apply));
  $("modeApply").setAttribute("aria-pressed", String(apply));
  $("modeApply").disabled = !$("patchReviewed").checked;
}

async function loadPatch() {
  if (!app.project) return;
  const apply = app.patchApply;
  await busy($("loadPatch"), "Generating…", async () => {
    try {
      const res = await fetch("/api/patch" + q() + "&apply=" + (apply ? "true" : "false"));
      if (!res.ok) throw new Error(await errText(res));
      const body = await res.json();
      if (body.shell) {
        $("patchShell").textContent = body.shell;
        window.patchShell = body.shell;
        $("copyPatch").disabled = false;
      }
      const mode = apply ? "Apply script (DRY_RUN=false): this changes authors when you run it." : "Dry-run script: run it, review the output, then switch to Apply.";
      const un = body.unresolved || [];
      setNote($("patchVerdict"), apply ? "fail" : un.length ? "warn" : "ok",
        body.rows + " rows. " + mode + (un.length ? " " + un.length + " unresolved skipped: " + un.join(", ") + "." : ""));
    } catch (e) {
      $("patchShell").textContent = "No script generated. See the error below.";
      window.patchShell = null;
      $("copyPatch").disabled = true;
      setNote($("patchVerdict"), "fail", "Patch unavailable: " + netMessage(e));
    }
  });
}

async function loadValidation() {
  const val = await getJSON("/api/validation" + q());
  $("validation").textContent = val.exists ? val.raw : "No reports/validation.md yet.";
  const k = val.exists ? kindOf(val.verdict) : "";
  setBanner($("valBanner"), k, val.exists ? "Validation: " + (val.verdict || "report ready") : "Validation not run",
    val.exists ? {ok: "Source and target match.", warn: "Target matches with warnings. Review them in reports/validation.md.", fail: "Source and target differ. See reports/validation.md."}[k] || "See reports/validation.md." : "Run the patch, then validate to compare the export with GitLab.");
}

// ---- Select project ----
async function select(name, step) {
  app.project = name;
  app.tested = {trac: null, gitlab: null};
  app.job = null;
  if (pollTimer) clearInterval(pollTimer);
  showWizard();
  $("projectSelect").value = name;
  $("sProject").textContent = "projects/" + name;
  document.querySelectorAll(".pdir").forEach(el => { el.textContent = "projects/" + name + "/"; });
  for (const id of ["tracTest", "gitlabTest", "executeVerdict", "patchVerdict", "valVerdict"]) $(id).hidden = true;
  setBanner($("exportVerdict"), "", "No export job in this session", "Run the export after the Trac connection tests green.");
  setBanner($("jobStatus"), "", "No dry-run job in this session", "Run it after preflight to see what Execute will create.");
  $("patchShell").textContent = "Press “Generate script” to build the patch for this project.";
  $("copyPatch").disabled = true;
  $("patchReviewed").checked = false;
  setPatchMode(false);
  setChips();
  try {
    await Promise.all([loadConnection(), loadPreflight(), loadUsers(), refreshStatus(), loadValidation()]);
    const running = !(await pollJob());
    if (running) startPolling();
    if (stepState().export && !(app.job && app.job.kind === "export")) {
      setBanner($("exportVerdict"), "ok", "Exported to projects/" + name + "/data/",
        "Authors were discovered from it into mappings/users.yaml. Run the export again only to refresh the data.");
      applyPrimaries();
    }
  } catch (e) {
    setBanner($("verdict"), "fail", "Could not load project", netMessage(e));
  }
  let saved = null;
  try { saved = localStorage.getItem("t2g.step." + name); } catch { /* storage unavailable */ }
  showStep(step || (STEPS.some(s => s.id === saved) ? saved : firstOpenStep()));
}

function firstOpenStep() {
  const st = stepState();
  return (STEPS.find(s => !st[s.id]) || STEPS[STEPS.length - 1]).id;
}

// ---- Wire up ----
document.addEventListener("DOMContentLoaded", async () => {
  hydrateIcons(document);
  $("homeLink").addEventListener("click", showHome);
  $("projectSelect").addEventListener("change", e => (e.target.value ? select(e.target.value) : showHome()));
  $("showCreate").addEventListener("click", () => { $("createForm").hidden = false; $("newProject").focus(); });
  $("cancelCreate").addEventListener("click", () => { $("createForm").hidden = true; $("createError").hidden = true; });
  $("createForm").addEventListener("submit", createProject);
  $("saveTrac").addEventListener("click", () => saveService("trac"));
  $("saveGitlab").addEventListener("click", () => saveService("gitlab"));
  $("testTrac").addEventListener("click", () => testConnection("trac"));
  $("testGitlab").addEventListener("click", () => testConnection("gitlab"));
  for (const [box, fields] of [["tracHtaccess", "tracHtaccessFields"], ["gitlabHtaccess", "gitlabHtaccessFields"]]) {
    $(box).addEventListener("change", e => { $(fields).hidden = !e.target.checked; });
  }
  document.querySelectorAll("[data-goto]").forEach(b => b.addEventListener("click", () => showStep(b.dataset.goto)));
  $("runPreflight").addEventListener("click", runPreflight);
  $("runDiscover").addEventListener("click", runDiscover);
  $("validateUsers").addEventListener("click", validateUsers);
  document.querySelectorAll("#userTabs .tab").forEach(t => t.addEventListener("click", () => {
    app.userFilter = t.dataset.filter;
    document.querySelectorAll("#userTabs .tab").forEach(x => x.setAttribute("aria-selected", String(x === t)));
    renderUsers();
  }));
  $("runExport").addEventListener("click", () => startRun("/api/export/run", $("runExport"),
    m => setBanner($("exportVerdict"), "fail", "Export failed to start", m)));
  $("runDryRun").addEventListener("click", () => startRun("/api/migrate/dry-run", $("runDryRun"),
    m => setBanner($("jobStatus"), "fail", "Dry-run failed to start", m)));
  $("runValidate").addEventListener("click", () => startRun("/api/validate/run", $("runValidate"),
    m => setNote($("valVerdict"), "fail", "Validate failed to start. " + m)));
  $("confirmExecute").addEventListener("input", updateExecuteButton);
  $("runExecute").addEventListener("click", runExecute);
  $("modeDry").addEventListener("click", () => setPatchMode(false));
  $("modeApply").addEventListener("click", () => setPatchMode(true));
  $("patchReviewed").addEventListener("change", e => { if (!e.target.checked) setPatchMode(false); else $("modeApply").disabled = false; });
  $("loadPatch").addEventListener("click", loadPatch);
  $("copyPatch").addEventListener("click", async () => {
    if (!window.patchShell) return;
    try {
      await navigator.clipboard.writeText(window.patchShell);
      setNote($("patchVerdict"), "ok", "Script copied. Run it on the GitLab host.");
    } catch {
      setNote($("patchVerdict"), "warn", "Clipboard unavailable. Select the script text and copy it manually.");
    }
  });
  try {
    await loadProjects();
  } catch (e) {
    $("projectRows").innerHTML = '<tr><td colspan="5" class="empty">Could not load projects: ' + esc(netMessage(e)) + "</td></tr>";
  }
  const [, hashProject, hashStep] = decodeURIComponent(location.hash).match(/^#\/([^/]+)\/?([a-z]*)/) || [];
  if (hashProject && app.projects.some(p => p.name === hashProject)) {
    await select(hashProject, STEPS.some(s => s.id === hashStep) ? hashStep : undefined);
  } else {
    showHome();
  }
});
