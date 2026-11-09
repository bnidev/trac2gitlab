// Slice 1: read-only dashboard. Vanilla fetch, no external dependencies.
async function getJSON(url) {
  const res = await fetch(url);
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
    return raw;
  }
}

function setVerdict(el, verdict) {
  const v = verdict || "";
  el.textContent = v ? "Conclusion: " + v : "No cached report yet";
  let cls = "verdict";
  if (/BLOCKED|NOT READY|FAIL/.test(v)) cls += " fail";
  else if (/WARN/.test(v)) cls += " warn";
  else if (/READY|PASS/.test(v)) cls += " ok";
  el.className = cls;
}

// ---- Step navigation ----
const STEPS = [
  {id: "connect", n: 1, title: "Connect", sub: "Pick a migration project. Config lives in migration.yaml, secrets in .env — never shown here."},
  {id: "preflight", n: 2, title: "Preflight", sub: "Read-only checks. No target mutation until READY."},
  {id: "identities", n: 3, title: "Identities", sub: "Map Trac authors to GitLab users. Unresolved get an explicit footer."},
  {id: "export", n: 4, title: "Export", sub: "Pull tickets, wiki and users from Trac into data/. Read-only vs Trac."},
  {id: "dryrun", n: 5, title: "Dry-run", sub: "Preview per-phase counts in state/migration.yaml. Nothing written to GitLab."},
  {id: "execute", n: 6, title: "Execute", sub: "Writes to GitLab. Requires READY preflight plus your explicit confirmation."},
  {id: "finish", n: 7, title: "Patch + Validate", sub: "Restore authors on the GitLab host, then compare export with target."},
];

function showStep(id) {
  window.currentStep = id;
  for (const s of STEPS) {
    document.getElementById("pane-" + s.id).classList.toggle("on", s.id === id);
  }
  const cur = STEPS.find(s => s.id === id);
  document.getElementById("headline").textContent = cur.title;
  document.getElementById("subhead").textContent = cur.sub;
  document.querySelectorAll("#steps .step").forEach(el => {
    el.classList.toggle("active", el.dataset.step === id);
  });
}

function renderSteps() {
  const nav = document.getElementById("steps");
  nav.innerHTML = "";
  for (const s of STEPS) {
    const d = document.createElement("div");
    d.className = "step" + (s.id === "execute" ? " locked" : "");
    d.dataset.step = s.id;
    d.innerHTML = '<span class="dot">' + s.n + '</span><span>' + s.title + '</span>';
    d.addEventListener("click", () => showStep(s.id));
    nav.appendChild(d);
  }
}

async function loadProjectInfo(name) {
  const list = await getJSON("/api/projects");
  const p = (list || []).find(x => x.name === name);
  const el = document.getElementById("projectInfo");
  if (!p) {
    el.textContent = "select a project…";
    return;
  }
  const rows = [
    ["config", p.hasConfig ? "migration.yaml present" : "missing — run init"],
    ["export data", p.hasState || p.hasConfig ? "see data/ after export" : "—"],
    ["state", p.hasState ? "state/migration.yaml present" : "no run yet"],
    ["preflight", p.hasPreflight ? "reports/preflight.md present" : "not run yet"],
    ["validation", p.hasValidation ? "reports/validation.md present" : "not run yet"],
  ];
  el.innerHTML = rows.map(r => '<div class="kv"><span class="mono">' + r[0] + '</span><span>' + r[1] + '</span></div>').join("");
}

async function renderProjectList() {
  const list = await getJSON("/api/projects");
  const box = document.getElementById("projectList");
  box.innerHTML = "";
  if (!list || list.length === 0) {
    box.innerHTML = '<span class="muted">No projects yet — create one above.</span>';
    return list;
  }
  for (const p of list) {
    const b = document.createElement("button");
    b.className = "proj-pick mono" + (p.name === window.currentProject ? " sel" : "");
    b.textContent = p.name;
    b.addEventListener("click", () => select(p.name));
    box.appendChild(b);
  }
  return list;
}

async function createProject() {
  const input = document.getElementById("newProject");
  const name = input.value.trim();
  const el = document.getElementById("connVerdict");
  el.hidden = false;
  if (!name) return;
  try {
    const res = await fetch("/api/projects", {method: "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify({name})});
    if (!res.ok) throw new Error(await errText(res));
    input.value = "";
    el.textContent = "Project " + name + " created — now set its connection below.";
    el.className = "verdict ok";
    await renderProjectList();
    await select(name);
  } catch (e) {
    el.textContent = "Create failed: " + e.message;
    el.className = "verdict fail";
  }
}

async function loadConnection(q) {
  const body = await getJSON("/api/project/connection" + q);
  document.getElementById("fTracUrl").value = body.tracBaseUrl || "";
  document.getElementById("fTracRpc").value = body.tracRpcPath || "";
  document.getElementById("fGitlabUrl").value = body.gitlabUrl || "";
  document.getElementById("fGitlabProj").value = body.gitlabProject || "";
  document.getElementById("fTracUser").value = body.tracUser || "";
  document.getElementById("fTracPass").value = "";
  document.getElementById("fTracPass").placeholder = body.hasTracPassword ? "stored ✓ — leave blank to keep" : "only sent, never shown";
  document.getElementById("fGitlabToken").value = "";
  document.getElementById("fGitlabToken").placeholder = body.hasGitlabToken ? "stored ✓ — leave blank to keep" : "only sent, never shown";
}

async function testConnection(kind) {
  const name = window.currentProject;
  const el = document.getElementById(kind === "trac" ? "tracTest" : "gitlabTest");
  if (!name) return;
  el.hidden = false;
  el.textContent = "Testing… (uses saved connection — save first)";
  el.className = "verdict";
  try {
    const res = await fetch("/api/test/" + kind + "?project=" + encodeURIComponent(name), {method: "POST"});
    if (!res.ok) throw new Error(await errText(res));
    const body = await res.json();
    el.textContent = body.message;
    el.className = "verdict " + (body.ok ? "ok" : "fail");
  } catch (e) {
    el.textContent = "Test failed: " + e.message;
    el.className = "verdict fail";
  }
}

async function saveConnection() {
  const name = window.currentProject;
  const el = document.getElementById("connVerdict");
  el.hidden = false;
  if (!name) {
    el.textContent = "Select or create a project first.";
    el.className = "verdict";
    return;
  }
  const payload = {
    tracBaseUrl: document.getElementById("fTracUrl").value,
    tracRpcPath: document.getElementById("fTracRpc").value,
    tracUser: document.getElementById("fTracUser").value,
    tracPassword: document.getElementById("fTracPass").value,
    gitlabUrl: document.getElementById("fGitlabUrl").value,
    gitlabToken: document.getElementById("fGitlabToken").value,
    gitlabProject: document.getElementById("fGitlabProj").value,
  };
  try {
    const res = await fetch("/api/project/connection?project=" + encodeURIComponent(name), {
      method: "PUT", headers: {"Content-Type": "application/json"}, body: JSON.stringify(payload),
    });
    if (!res.ok) throw new Error(await errText(res));
    document.getElementById("fTracPass").value = "";
    document.getElementById("fGitlabToken").value = "";
    el.textContent = "Saved migration.yaml + .env locally. Secrets stay server-side.";
    el.className = "verdict ok";
  } catch (e) {
    el.textContent = "Save failed: " + e.message;
    el.className = "verdict fail";
  }
}

async function select(name) {
  window.currentProject = name;
  document.getElementById("projLabel").textContent = "projects / " + name;
  document.querySelectorAll(".proj-pick").forEach(b => b.classList.toggle("sel", b.textContent === name));
  await loadProjectInfo(name);
  const q = "?project=" + encodeURIComponent(name);
  await loadConnection(q);
  const status = await getJSON("/api/status" + q);
  document.getElementById("status").textContent = JSON.stringify(status, null, 2);
  await loadCached(q, name);
  await loadUsers(q);
  await pollJob(q);
  const val = await getJSON("/api/validation" + q);
  document.getElementById("validation").textContent = val.exists ? val.raw : "No reports/validation.md yet";
}

async function loadCached(q, name) {
  const pre = await getJSON("/api/preflight" + q);
  setVerdict(document.getElementById("verdict"), pre.verdict);
  if (pre.checks) renderChecks(pre.checks);
  document.getElementById("report").textContent = pre.exists ? pre.raw : "No reports/preflight.md yet — press Run preflight (or run: trac2gitlab preflight --project-dir projects/" + name + ")";
}

function renderChecks(checks) {
  const box = document.getElementById("checks");
  box.innerHTML = "";
  for (const c of checks) {
    const row = document.createElement("div");
    row.className = "check";
    const icon = c.state === "pass" ? "✓" : c.state === "warn" ? "!" : "✗";
    row.innerHTML = '<span class="badge ' + c.state + '">' + icon + '</span>' +
      '<span><span class="n">' + c.name + '</span><br><span class="d mono">' + c.detail + '</span></span>' +
      '<span class="pill mono">' + c.state.toUpperCase() + '</span>';
    box.appendChild(row);
  }
}

function renderUsers(users) {
  const box = document.getElementById("users");
  box.innerHTML = "";
  if (!users) return;
  for (const u of users) {
    const row = document.createElement("div");
    row.className = "check";
    const icon = u.resolved ? "✓" : "!";
    const state = u.resolved ? "pass" : "warn";
    const target = u.resolved ? "@" + u.gitlab : "footer + patch skip";
    row.innerHTML = '<span class="badge ' + state + '">' + icon + '</span>' +
      '<span><span class="n mono">' + u.key + '</span><br><span class="d mono">' + u.aliases.join(" · ") + '</span></span>' +
      '<span class="pill mono">' + target.toUpperCase() + '</span>';
    box.appendChild(row);
  }
}

async function loadUsers(q) {
  const body = await getJSON("/api/users" + q);
  if (!body.exists) {
    document.getElementById("users").innerHTML = "";
    document.getElementById("usersVerdict").textContent = "No mappings/users.yaml — run: trac2gitlab discover --write-users mappings/users.yaml";
    document.getElementById("usersVerdict").className = "verdict";
    return;
  }
  if (body.error) {
    document.getElementById("usersVerdict").textContent = "Parse error: " + body.error;
    document.getElementById("usersVerdict").className = "verdict fail";
    return;
  }
  renderUsers(body.users);
  document.getElementById("usersVerdict").textContent = body.users.length + " entries loaded — press Validate";
  document.getElementById("usersVerdict").className = "verdict";
}

async function validateUsers() {
  const btn = document.getElementById("validateUsers");
  const name = window.currentProject;
  if (!name) return;
  btn.disabled = true;
  btn.textContent = "Validating…";
  try {
    const res = await fetch("/api/users/validate?project=" + encodeURIComponent(name), {method: "POST"});
    if (!res.ok) throw new Error(await errText(res));
    const body = await res.json();
    renderUsers(body.users);
    const el = document.getElementById("usersVerdict");
    if (body.error) {
      el.textContent = body.error;
      el.className = "verdict fail";
    } else if (body.ok) {
      el.textContent = "All " + body.total + " identities resolved";
      el.className = "verdict ok";
    } else {
      el.textContent = body.unresolved.length + " unresolved (" + body.unresolved.join(", ") + ") → “Originally by …” footer, skipped in patch";
      el.className = "verdict warn";
    }
  } catch (e) {
    document.getElementById("usersVerdict").textContent = "Validate failed: " + e.message;
  } finally {
    btn.disabled = false;
    btn.textContent = "Validate";
  }
}

let pollTimer = null;

async function refreshStatus(q) {
  const status = await getJSON("/api/status" + q);
  document.getElementById("status").textContent = JSON.stringify(status, null, 2);
}

async function pollJob(q) {
  const job = await getJSON("/api/job" + q);
  const st = document.getElementById("jobStatus");
  const log = document.getElementById("jobLog");
  if (!job || job.exists === false) {
    st.textContent = "no job yet";
    st.className = "verdict";
    return true;
  }
  st.textContent = job.kind + " " + job.id + " · " + job.status + (job.error ? " — " + job.error : "");
  st.className = "verdict" + (job.status === "completed" ? " ok" : job.status === "failed" ? " fail" : "");
  log.textContent = (job.log || []).join("\n") || "…";
  if (job.status !== "running") {
    await refreshStatus(q);
    return true;
  }
  return false;
}

function startPolling(q) {
  if (pollTimer) clearInterval(pollTimer);
  pollTimer = setInterval(async () => {
    if (await pollJob(q)) clearInterval(pollTimer);
  }, 2000);
}

async function startRun(path, verdictId, q) {
  const res = await fetch(path + q, {method: "POST"});
  if (!res.ok) throw new Error(await errText(res));
  const body = await res.json();
  document.getElementById(verdictId).textContent = body.kind + " " + body.id + " started…";
  document.getElementById(verdictId).className = "verdict";
  startPolling(q);
}

async function runExport() {
  const btn = document.getElementById("runExport");
  const name = window.currentProject;
  if (!name) return;
  btn.disabled = true;
  try {
    await startRun("/api/export/run", "exportVerdict", "?project=" + encodeURIComponent(name));
  } catch (e) {
    const el = document.getElementById("exportVerdict");
    el.textContent = "Export failed to start: " + e.message;
    el.className = "verdict fail";
  } finally {
    btn.disabled = false;
  }
}

async function runDryRun() {
  const btn = document.getElementById("runDryRun");
  const name = window.currentProject;
  if (!name) return;
  btn.disabled = true;
  try {
    await startRun("/api/migrate/dry-run", "jobStatus", "?project=" + encodeURIComponent(name));
  } catch (e) {
    const el = document.getElementById("jobStatus");
    el.textContent = "Dry-run failed to start: " + e.message;
    el.className = "verdict fail";
  } finally {
    btn.disabled = false;
  }
}
async function runExecute() {
  const btn = document.getElementById("runExecute");
  const box = document.getElementById("confirmExecute");
  const name = window.currentProject;
  if (!name || !box.checked) return;
  btn.disabled = true;
  try {
    const res = await fetch("/api/migrate/execute?project=" + encodeURIComponent(name), {
      method: "POST",
      headers: {"Content-Type": "application/json"},
      body: JSON.stringify({confirm: true}),
    });
    if (!res.ok) throw new Error(await errText(res));
    const body = await res.json();
    const el = document.getElementById("executeVerdict");
    el.textContent = "execute " + body.id + " started…";
    el.className = "verdict";
    startPolling("?project=" + encodeURIComponent(name));
  } catch (e) {
    const el = document.getElementById("executeVerdict");
    el.textContent = "Execute refused: " + e.message;
    el.className = "verdict fail";
  } finally {
    btn.disabled = !document.getElementById("confirmExecute").checked;
  }
}

async function loadPatch(q, apply) {
  const el = document.getElementById("patchVerdict");
  const shell = document.getElementById("patchShell");
  el.textContent = "Loading patch…";
  el.className = "verdict";
  try {
    const res = await fetch("/api/patch" + q + "&apply=" + (apply ? "true" : "false"));
    if (!res.ok) throw new Error(await errText(res));
    const body = await res.json();
    if (body.shell) {
      shell.textContent = body.shell;
      window.patchShell = body.shell;
    }
    if (body.unresolved && body.unresolved.length > 0) {
      el.textContent = body.rows + " rows · " + body.unresolved.length + " unresolved skipped (" + body.unresolved.join(", ") + ")";
      el.className = "verdict warn";
    } else {
      el.textContent = body.rows + " rows · " + (apply ? "APPLYING (DRY_RUN=false)" : "dry run — review, then tick apply");
      el.className = "verdict " + (apply ? "fail" : "ok");
    }
  } catch (e) {
    el.textContent = "Patch unavailable: " + e.message;
    el.className = "verdict fail";
  }
}

async function runValidate() {
  const btn = document.getElementById("runValidate");
  const name = window.currentProject;
  if (!name) return;
  btn.disabled = true;
  try {
    await startRun("/api/validate/run", "jobStatus", "?project=" + encodeURIComponent(name));
  } catch (e) {
    const el = document.getElementById("jobStatus");
    el.textContent = "Validate failed to start: " + e.message;
    el.className = "verdict fail";
  } finally {
    btn.disabled = false;
  }
}

async function runPreflight() {
  const btn = document.getElementById("runPreflight");
  const name = window.currentProject;
  if (!name) {
    document.getElementById("verdict").textContent = "No project selected — create one with: trac2gitlab init --project-dir projects/demo";
    return;
  }
  btn.disabled = true;
  btn.textContent = "Running…";
  try {
    const res = await fetch("/api/preflight/run?project=" + encodeURIComponent(name), {method: "POST"});
    if (!res.ok) throw new Error(await errText(res));
    const body = await res.json();
    renderChecks(body.checks);
    setVerdict(document.getElementById("verdict"), body.verdict);
    await loadCached("?project=" + encodeURIComponent(name), name);
  } catch (e) {
    document.getElementById("verdict").textContent = "Run failed: " + e.message;
  } finally {
    btn.disabled = false;
    btn.textContent = "Run preflight";
  }
}

document.addEventListener("DOMContentLoaded", async () => {
  renderSteps();
  showStep("connect");
  document.getElementById("createProject").addEventListener("click", createProject);
  document.getElementById("saveConnection").addEventListener("click", saveConnection);
  document.getElementById("testTrac").addEventListener("click", () => testConnection("trac"));
  document.getElementById("testGitlab").addEventListener("click", () => testConnection("gitlab"));
  document.querySelectorAll("[data-goto]").forEach(b => b.addEventListener("click", () => showStep(b.dataset.goto)));
  document.querySelectorAll(".proj-btn").forEach(b => b.addEventListener("click", () => select(b.dataset.name)));
  document.getElementById("runPreflight").addEventListener("click", runPreflight);
  document.getElementById("validateUsers").addEventListener("click", validateUsers);
  document.getElementById("runExport").addEventListener("click", runExport);
  document.getElementById("runDryRun").addEventListener("click", runDryRun);
  document.getElementById("runExecute").addEventListener("click", runExecute);
  document.getElementById("runValidate").addEventListener("click", runValidate);
  document.getElementById("loadPatch").addEventListener("click", () => {
    const q = "?project=" + encodeURIComponent(window.currentProject || "");
    loadPatch(q, document.getElementById("patchApply").checked);
  });
  document.getElementById("copyPatch").addEventListener("click", async () => {
    if (window.patchShell) await navigator.clipboard.writeText(window.patchShell);
  });
  document.getElementById("confirmExecute").addEventListener("change", (e) => {
    document.getElementById("runExecute").disabled = !e.target.checked;
  });
  const list = await renderProjectList();
  if (list && list[0]) await select(list[0].name);
});
