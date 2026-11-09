// Slice 1: read-only dashboard. Vanilla fetch, no external dependencies.
async function getJSON(url) {
  const res = await fetch(url);
  return res.json();
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

async function select(name) {
  document.querySelectorAll(".proj-btn").forEach(b => b.classList.toggle("sel", b.dataset.name === name));
  document.getElementById("projLabel").textContent = "projects / " + name;
  window.currentProject = name;
  const q = "?project=" + encodeURIComponent(name);
  const status = await getJSON("/api/status" + q);
  document.getElementById("status").textContent = JSON.stringify(status, null, 2);
  await loadCached(q, name);
  await loadUsers(q);
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
    const body = await res.json();
    if (!res.ok) throw new Error(body || res.statusText);
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
    const body = await res.json();
    if (!res.ok) throw new Error(body || res.statusText);
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
  document.querySelectorAll(".proj-btn").forEach(b => b.addEventListener("click", () => select(b.dataset.name)));
  document.getElementById("runPreflight").addEventListener("click", runPreflight);
  document.getElementById("validateUsers").addEventListener("click", validateUsers);
  const first = document.querySelector(".proj-btn");
  if (first) await select(first.dataset.name);
});
