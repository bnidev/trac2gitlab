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
  const first = document.querySelector(".proj-btn");
  if (first) await select(first.dataset.name);
});
