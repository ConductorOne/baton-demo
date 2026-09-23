"use strict";

const $ = (id) => document.getElementById(id);
const state = { snapshot: null, type: "users", id: null, changes: [], busy: false };
const relationships = {
  groups: { admins: ["users", "Admin"], members: ["users", "Member"] },
  roles: { direct_assignments: ["users", "Direct assignment"], group_assignments: ["groups", "Assigned group"] },
  projects: { owner: ["users", "Owner"], group_assignments: ["groups", "Assigned group"] },
  scoped_roles: { project_id: ["projects", "Project"], role_id: ["roles", "Role"], user_assignments: ["users", "Direct assignment"] },
  secrets: { identity_id: ["users", "Owning identity"] },
  agents: { identity_id: ["users", "Authenticating identity"] },
};
const words = (s) => s.replaceAll("_", " ").replace(/^./, (c) => c.toUpperCase());
const ids = (value) => value ? String(value).split(",").filter(Boolean) : [];
const records = (type) => state.snapshot?.data[type] || [];
const find = (type, id) => {
  const canonical = type === "users" ? state.snapshot?.user_aliases?.[id] || id : id;
  return records(type).find((r) => r.id === canonical);
};
function name(type, row) {
  if (row.name) return row.name;
  if (type === "scoped_roles") return `${find("roles", row.role_id)?.name || row.role_id} / ${find("projects", row.project_id)?.name || row.project_id}`;
  return row.id;
}
function el(tag, text, className) {
  const node = document.createElement(tag);
  if (text !== undefined) node.textContent = text;
  if (className) node.className = className;
  return node;
}
function route(type, id) { return `#${encodeURIComponent(type)}${id ? "/" + encodeURIComponent(id) : ""}`; }
function link(type, id) {
  const row = find(type, id);
  if (!row) return el("span", `${id} (record unavailable)`, "muted");
  const node = el("a", undefined, "link related-object");
  node.href = route(type, row.id);
  const title = name(type, row);
  const initials = title.split(/\s+/).slice(0, 2).map((s) => s[0]).join("").toUpperCase();
  const text = el("span", undefined, "related-object-text");
  text.append(el("strong", title));
  const subtitle = row.email || row.nhi_type || row.credential_type || row.id;
  text.append(el("small", subtitle));
  node.append(el("span", initials, "avatar"), text);
  return node;
}
function parseRoute() {
  const parts = location.hash.slice(1).split("/");
  const previous = state.type;
  try { state.type = decodeURIComponent(parts[0] || "users"); state.id = parts[1] ? decodeURIComponent(parts[1]) : null; }
  catch { state.type = "users"; state.id = null; }
  if (state.snapshot && !state.snapshot.tables.some((t) => t.name === state.type)) state.type = "users";
  if (previous !== state.type) $("search").value = "";
  render();
}
function render() {
  if (!state.snapshot) return;
  if (!state.snapshot.tables.some((t) => t.name === state.type)) { state.type = "users"; state.id = null; }
  const current = state.snapshot.tables.find((t) => t.name === state.type);
  $("title").textContent = current.label;
  $("subtitle").textContent = "Live records and direct relationships from your demo application.";
  $("navigation").replaceChildren(...state.snapshot.tables.map((t) => {
    const a = el("a", t.label, t.name === state.type ? "active" : "");
    a.href = route(t.name);
    if (t.name === state.type) a.setAttribute("aria-current", "page");
    a.append(el("span", state.snapshot.missing.includes(t.name) ? "—" : records(t.name).length));
    return a;
  }));
  renderList(); renderDetail();
}
function renderList() {
  const query = $("search").value.toLowerCase();
  const all = records(state.type);
  const filtered = all.filter((r) => (name(state.type, r) + " " + JSON.stringify(r)).toLowerCase().includes(query));
  $("count").textContent = `${filtered.length} / ${all.length}`;
  if (!filtered.length) {
    $("list").replaceChildren(el("div", state.snapshot.missing.includes(state.type) ? "This table is absent in this database version." : "No matching records.", "placeholder"));
    return;
  }
  $("list").replaceChildren(...filtered.map((r) => {
    const button = el("button", undefined, "record" + (r.id === state.id ? " selected" : ""));
    button.setAttribute("aria-pressed", String(r.id === state.id));
    button.addEventListener("click", () => { location.hash = route(state.type, r.id); });
    const title = name(state.type, r);
    const initials = title.split(/\s+/).slice(0, 2).map((s) => s[0]).join("").toUpperCase();
    const text = el("span", undefined, "record-text");
    text.append(el("strong", title), el("small", r.email || r.nhi_type || r.credential_type || r.id));
    button.append(el("span", initials, "avatar"), text);
    if (state.type === "users") button.append(el("span", r.enabled ? "Enabled" : "Disabled", "badge" + (r.enabled ? "" : " disabled")));
    if (state.type === "groups") {
      const count = ids(r.members).length;
      button.append(el("span", `${count} ${count === 1 ? "member" : "members"}`, "badge"));
    }
    return button;
  }));
}
function relation(type, id, label) {
  const row = el("div", undefined, "relation");
  const object = find(type, id);
  row.append(link(type, id));
  if (type === "users" && object && object.enabled != null) {
    row.append(el("span", object.enabled ? "Enabled" : "Disabled", "badge" + (object.enabled ? "" : " disabled")));
  }
  row.append(el("span", label, "relation-label"));
  if (object && object.id !== id) {
    row.append(el("small", "Matched by seeded email ID", "alias-note"));
  }
  return row;
}
function renderDetail() {
  const panel = $("detail");
  if (!state.id) { panel.replaceChildren(el("div", "Select a record to see its details and assignments.", "placeholder")); return; }
  const r = find(state.type, state.id);
  if (!r) { panel.replaceChildren(el("div", "This record is missing or has been deleted: " + state.id, "placeholder")); return; }
  const content = document.createDocumentFragment();
  content.append(el("h2", name(state.type, r)), el("div", r.id, "identifier"));
  const fields = el("dl", undefined, "fields");
  for (const [key, value] of Object.entries(r)) {
    if (["id", "name"].includes(key) || relationships[state.type]?.[key]) continue;
    let formatted = value ?? "—";
    if (key === "enabled") formatted = value ? "Enabled" : "Disabled";
    if (["attrs", "profile", "nhi_detail", "credential_detail"].includes(key) && value) {
      try { formatted = JSON.stringify(JSON.parse(value), null, 2); } catch { /* Display non-JSON metadata as text. */ }
    }
    fields.append(el("dt", words(key)), el("dd", String(formatted)));
  }
  content.append(fields);
  for (const [key, [type, label]] of Object.entries(relationships[state.type] || {})) {
    content.append(el("h3", words(key), "section-title"));
    const values = ids(r[key]);
    if (!values.length) content.append(el("p", "None", "muted"));
    values.forEach((id) => content.append(relation(type, id, label)));
  }
  const incoming = [];
  for (const [type, mapping] of Object.entries(relationships)) {
    for (const other of records(type)) {
      for (const [key, [target, label]] of Object.entries(mapping)) {
        if (target === state.type && ids(other[key]).some((id) => find(target, id)?.id === r.id)) incoming.push(relation(type, other.id, label));
      }
    }
  }
  if (incoming.length) content.append(el("h3", "Related records · direct relationships", "section-title"), ...incoming);
  const raw = el("details"); raw.append(el("summary", "View record JSON"), el("pre", JSON.stringify(r, null, 2))); content.append(raw);
  panel.replaceChildren(content);
}
function observe(before, after) {
  if (!before) return;
  for (const table of after.tables) {
    const oldRows = new Map((before.data[table.name] || []).map((r) => [r.id, r]));
    for (const row of after.data[table.name]) {
      const old = oldRows.get(row.id);
      if (!old) addChange(`Added ${table.label.toLowerCase()}: ${name(table.name, row)}`);
      else if (JSON.stringify(old) !== JSON.stringify(row)) {
        const changes = [];
        for (const [key, [target, label]] of Object.entries(relationships[table.name] || {})) {
          const previous = ids(old[key]), next = ids(row[key]);
          for (const id of next.filter((v) => !previous.includes(v))) changes.push(`added ${label.toLowerCase()} ${find(target, id)?.name || id}`);
          for (const id of previous.filter((v) => !next.includes(v))) changes.push(`removed ${label.toLowerCase()} ${find(target, id)?.name || id}`);
        }
        addChange(`${name(table.name, row)}: ${changes.length ? changes.join("; ") : "record updated"}`);
      }
      oldRows.delete(row.id);
    }
    for (const row of oldRows.values()) addChange(`Removed ${table.label.toLowerCase()}: ${row.name || row.id}`);
  }
}
function addChange(text) {
  state.changes.unshift({ text, time: new Date().toLocaleTimeString() });
  state.changes = state.changes.slice(0, 40);
  $("changes").replaceChildren(...state.changes.map((change, i) => {
    const li = el("li", undefined, i === 0 ? "changed" : "");
    li.append(el("time", change.time), document.createTextNode(change.text)); return li;
  }));
}
async function refresh() {
  if (state.busy) return;
  state.busy = true; $("refresh").disabled = true;
  try {
    const response = await fetch("/api/snapshot", { cache: "no-store", signal: AbortSignal.timeout(8000) });
    if (!response.ok) throw new Error(await response.text());
    const next = await response.json();
    const previous = state.snapshot; state.snapshot = next;
    observe(previous, next);
    $("path").textContent = next.path;
    $("status").textContent = `Updated ${new Date(next.read_at).toLocaleTimeString()} · ${$("auto").checked ? "refreshes every 2 seconds" : "automatic refresh paused"}`;
    $("error").hidden = !next.missing.length;
    $("error").textContent = next.missing.length ? "Older database: missing tables " + next.missing.join(", ") + ". The viewer has left the schema unchanged." : "";
    // Unchanged refreshes leave focus, scroll position and expanded JSON intact.
    if (!previous || JSON.stringify(previous.data) !== JSON.stringify(next.data)) render();
  } catch (error) {
    $("error").hidden = false;
    $("error").textContent = "Unable to refresh. " + error.message;
    $("status").textContent = state.snapshot ? `Stale · last read ${new Date(state.snapshot.read_at).toLocaleTimeString()}` : "Database unavailable";
  } finally { state.busy = false; $("refresh").disabled = false; }
}
$("search").addEventListener("input", renderList);
$("refresh").addEventListener("click", refresh);
$("auto").addEventListener("change", () => {
  if ($("auto").checked) refresh();
  else $("status").textContent = "Automatic refresh paused · use Refresh now";
});
window.addEventListener("hashchange", parseRoute);
document.addEventListener("visibilitychange", () => { if (!document.hidden && $("auto").checked) refresh(); });
setInterval(() => { if ($("auto").checked && !document.hidden) refresh(); }, 2000);
parseRoute(); refresh();
