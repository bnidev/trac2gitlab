// Slice 1: read-only dashboard. Vanilla fetch, no external dependencies.
async function getJSON(url) {
  const res = await fetch(url);
  return res.json();
}

function setVerdict(el, verdict) {
  el.textContent = verdict ? "Conclusion: " + verdict : "No cached report yet";
  el.className = "verdict" + (/WARN/.test(verdict || "") ? " warn" : /READY|PASS/.test(verdict || "") ? " ok" : "");
}

async function select(name) {
  document.querySelectorAll(".proj-btn").forEach(b => b.classList.toggle("sel", b.dataset.name === name));
  document.getElementById("projLabel").textContent = "projects / " + name;
  const q = "?project=" + encodeURIComponent(name);
  const status = await getJSON("/api/status" + q);
  document.getElementById("status").textContent = JSON.stringify(status, null, 2);
  const pre = await getJSON("/api/preflight" + q);
  setVerdict(document.getElementById("verdict"), pre.verdict);
  document.getElementById("report").textContent = pre.exists ? pre.raw : "No reports/preflight.md yet — run: trac2gitlab preflight --project-dir projects/" + name;
  const val = await getJSON("/api/validation" + q);
  document.getElementById("validation").textContent = val.exists ? val.raw : "No reports/validation.md yet";
}

document.addEventListener("DOMContentLoaded", async () => {
  document.querySelectorAll(".proj-btn").forEach(b => b.addEventListener("click", () => select(b.dataset.name)));
  const first = document.querySelector(".proj-btn");
  if (first) await select(first.dataset.name);
});
