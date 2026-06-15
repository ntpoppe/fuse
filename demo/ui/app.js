const API_BASE = window.FUSE_API_BASE ?? "";

const tableQueries = {
  shopUsers: "SELECT u.id, u.name, u.email, u.active FROM shop.users u LIMIT 100",
  warehouseOrders: "SELECT o.id, o.user_id, o.total, o.status FROM warehouse.orders o LIMIT 100",
};

const examples = [
  {
    label: "Federated join: shipped orders",
    sql: "SELECT u.id, u.name, o.total, o.status FROM shop.users u JOIN warehouse.orders o ON u.id = o.user_id WHERE u.active = 1 AND o.status = 'shipped' LIMIT 100",
    federated: true,
  },
  {
    label: "Pending order (Alice)",
    sql: "SELECT u.name, o.total FROM shop.users u JOIN warehouse.orders o ON u.id = o.user_id WHERE o.status = 'pending' LIMIT 10",
    federated: true,
  },
  {
    label: "Active users only",
    sql: "SELECT u.id, u.name FROM shop.users u WHERE u.active = 1 LIMIT 10",
    federated: true,
  },
  {
    label: "Single: active users (shop)",
    sql: "SELECT id, name, email FROM users WHERE active = 1 ORDER BY id",
    connection: "shop",
  },
  {
    label: "Single: shipped orders (warehouse)",
    sql: "SELECT id, user_id, total, status FROM orders WHERE status = 'shipped'",
    connection: "warehouse",
  },
  {
    label: "Orders >= 45.00",
    sql: "SELECT u.name, o.total FROM shop.users u INNER JOIN warehouse.orders o ON u.id = o.user_id WHERE o.total >= 45.00 LIMIT 10",
    federated: true,
  },
];

const connectionsEl = document.getElementById("connections");
const connectionSelect = document.getElementById("connection");
const sqlEditorEl = document.getElementById("sql-editor");
const errorEl = document.getElementById("error");
const resultsEl = document.getElementById("results");
const resultsMetaEl = document.getElementById("results-meta");
const examplesEl = document.getElementById("examples");

const sqlEditor = CodeMirror(sqlEditorEl, {
  value: examples[0].sql,
  mode: "text/x-sql",
  theme: "dracula",
  lineNumbers: true,
  lineWrapping: true,
  indentWithTabs: false,
  indentUnit: 2,
  tabSize: 2,
  viewportMargin: Infinity,
});

function getSQL() {
  return sqlEditor.getValue().trim();
}

function setSQL(sql) {
  sqlEditor.setValue(sql);
  sqlEditor.focus();
}

function showError(message) {
  if (!message) {
    errorEl.textContent = "";
    errorEl.classList.add("hidden");
    return;
  }
  errorEl.textContent = message;
  errorEl.classList.remove("hidden");
}

async function api(path, options) {
  const res = await fetch(`${API_BASE}${path}`, options);
  const text = await res.text();
  let body;
  try {
    body = text ? JSON.parse(text) : null;
  } catch {
    body = text;
  }
  if (!res.ok) {
    const msg = body && body.error ? body.error : `HTTP ${res.status}`;
    throw new Error(msg);
  }
  return body;
}

function renderConnections(connections) {
  connectionsEl.innerHTML = "";
  connectionSelect.innerHTML = '<option value="">federated</option>';

  if (!connections.length) {
    connectionsEl.innerHTML = '<li class="muted">No connections</li>';
    return;
  }

  for (const conn of connections) {
    const li = document.createElement("li");
    li.innerHTML =
      '<span class="conn-id">' +
      escapeHtml(conn.id) +
      '</span> <span class="conn-driver">' +
      escapeHtml(conn.driver) +
      "</span>";
    connectionsEl.appendChild(li);

    const opt = document.createElement("option");
    opt.value = conn.id;
    opt.textContent = conn.id + " (" + conn.driver + ")";
    connectionSelect.appendChild(opt);
  }
}

function escapeHtml(s) {
  return String(s)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function renderResults(rows) {
  resultsEl.innerHTML = "";

  if (!Array.isArray(rows) || rows.length === 0) {
    resultsMetaEl.textContent = "0 rows";
    resultsEl.innerHTML = '<p class="muted">No rows returned.</p>';
    return;
  }

  resultsMetaEl.textContent = rows.length + (rows.length === 1 ? " row" : " rows");

  const columns = Object.keys(rows[0]);
  const table = document.createElement("table");
  const thead = document.createElement("thead");
  const headerRow = document.createElement("tr");

  for (const col of columns) {
    const th = document.createElement("th");
    th.textContent = col;
    headerRow.appendChild(th);
  }
  thead.appendChild(headerRow);
  table.appendChild(thead);

  const tbody = document.createElement("tbody");
  for (const row of rows) {
    const tr = document.createElement("tr");
    for (const col of columns) {
      const td = document.createElement("td");
      const val = row[col];
      td.textContent = val === null || val === undefined ? "" : String(val);
      tr.appendChild(td);
    }
    tbody.appendChild(tr);
  }
  table.appendChild(tbody);
  resultsEl.appendChild(table);
}

async function runSingle() {
  const id = connectionSelect.value;
  const sql = getSQL();
  if (!id) {
    showError("Select a connection for a single-database query, or use Run federated.");
    return;
  }
  if (!sql) {
    showError("Enter a SQL statement.");
    return;
  }
  showError("");
  const rows = await api("/api/query", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ id, sql }),
  });
  renderResults(rows);
}

async function runFederated() {
  const sql = getSQL();
  if (!sql) {
    showError("Enter a SQL statement.");
    return;
  }
  showError("");
  const rows = await api("/api/federated-query", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ sql }),
  });
  renderResults(rows);
}

function handleRunError(err) {
  showError(err.message);
  resultsEl.innerHTML = "";
  resultsMetaEl.textContent = "";
}

function runSelectedQuery() {
  if (connectionSelect.value) {
    return runSingle();
  }
  return runFederated();
}

function loadExample(example) {
  setSQL(example.sql);
  if (example.connection) {
    connectionSelect.value = example.connection;
  } else {
    connectionSelect.value = "";
  }
  showError("");
}

function loadTableQuery(sql) {
  setSQL(sql);
  connectionSelect.value = "";
  showError("");
}

function initExamples() {
  for (const ex of examples) {
    const li = document.createElement("li");
    const btn = document.createElement("button");
    btn.type = "button";
    btn.textContent = ex.label;
    btn.addEventListener("click", () => loadExample(ex));
    li.appendChild(btn);
    examplesEl.appendChild(li);
  }
}

document.getElementById("run-single").addEventListener("click", () => {
  runSingle().catch(handleRunError);
});

document.getElementById("run-federated").addEventListener("click", () => {
  runFederated().catch(handleRunError);
});

sqlEditor.setOption("extraKeys", {
  "Ctrl-Enter": () => runSelectedQuery().catch(handleRunError),
  "Cmd-Enter": () => runSelectedQuery().catch(handleRunError),
});

document.getElementById("select-shop-users").addEventListener("click", () => {
  loadTableQuery(tableQueries.shopUsers);
});

document.getElementById("select-warehouse-orders").addEventListener("click", () => {
  loadTableQuery(tableQueries.warehouseOrders);
});

initExamples();

api("/api/connections")
  .then(renderConnections)
  .catch((err) => {
    connectionsEl.innerHTML = '<li class="muted">Failed to load</li>';
    showError(err.message);
  });
