async function fetchJSON(path) {
  const response = await fetch(path);
  if (!response.ok) {
    throw new Error(`request failed: ${path}`);
  }
  return response.json();
}

function setActionOutput(value) {
  document.getElementById("actionOutput").textContent =
    typeof value === "string" ? value : JSON.stringify(value, null, 2);
}

function renderCards(elementId, items, render) {
  const node = document.getElementById(elementId);
  node.innerHTML = "";

  if (!items.length) {
    node.innerHTML = `<div class="card"><small>No data yet.</small></div>`;
    return;
  }

  items.forEach((item) => {
    const div = document.createElement("div");
    div.className = "card";
    div.innerHTML = render(item);
    node.appendChild(div);
  });
}

function formatBytes(bytes) {
  if (!bytes) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[unit]}`;
}

async function load() {
  const [status, daemonStatus, adapters, artifacts, jobs] = await Promise.all([
    fetchJSON("/api/v1/status"),
    fetchJSON("/api/v1/daemon/status"),
    fetchJSON("/api/v1/adapters"),
    fetchJSON("/api/v1/artifacts"),
    fetchJSON("/api/v1/scheduler"),
  ]);

  document.getElementById("modeValue").textContent = daemonStatus.mode;
  document.getElementById("configValue").textContent = `${status.configSource || "defaults"} via ${daemonStatus.serviceName}`;
  document.getElementById("artifactCount").textContent = daemonStatus.artifacts.count;
  document.getElementById("artifactSummary").textContent =
    `${formatBytes(daemonStatus.artifacts.totalSizeBytes)} total, ${daemonStatus.artifacts.restoreTested} restore-tested`;
  document.getElementById("scopeCount").textContent = daemonStatus.protectedScopes;
  document.getElementById("warningCount").textContent = daemonStatus.artifacts.degradedArtifacts;
  document.getElementById("adapterCount").textContent = adapters.items.length;

  renderCards("adapters", adapters.items, (item) => `
    <strong>${item.name}</strong>
    <small>${item.description}</small>
    <small>Hints: ${item.imageHints.join(", ") || "none"}</small>
  `);

  renderCards("artifacts", artifacts.items, (item) => `
    <strong>${item.scope} / ${item.service}</strong>
    <small>${item.id}</small>
    <small>${formatBytes(item.sizeBytes)} ${item.degraded ? '<span class="warn">degraded</span>' : ''}</small>
  `);

  renderCards("jobs", jobs.jobs, (item) => `
    <strong>${item.name}</strong>
    <small>Cadence: ${item.cadence}</small>
    <small>Last success: ${item.lastSuccessAt || "never"}</small>
  `);
}

async function runDaemonAction(path) {
  const result = await fetchJSON(path);
  setActionOutput(result);
}

document.getElementById("refreshButton").addEventListener("click", () => {
  load().catch((error) => console.error(error));
});

document.getElementById("guardComposeButton").addEventListener("click", () => {
  const composePath = encodeURIComponent(document.getElementById("composePath").value);
  runDaemonAction(`/api/v1/daemon/guard/compose?path=${composePath}&operation=compose.down.volumes`).catch((error) =>
    setActionOutput(error.message),
  );
});

document.getElementById("protectComposeButton").addEventListener("click", () => {
  const composePath = encodeURIComponent(document.getElementById("composePath").value);
  runDaemonAction(`/api/v1/daemon/protect/compose?path=${composePath}`).catch((error) =>
    setActionOutput(error.message),
  );
});

document.getElementById("guardKubeButton").addEventListener("click", () => {
  runDaemonAction("/api/v1/daemon/guard/kube-delete?path=examples/kubernetes-beta/manifests.yaml").catch((error) =>
    setActionOutput(error.message),
  );
});

load().catch((error) => {
  console.error(error);
});
