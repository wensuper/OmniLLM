// OmniLLM console. Everything the server returns is shown with textContent;
// nothing from the server or from a device is ever inserted as HTML.
"use strict";

const T = {
  zh: {
    loading: "正在加载……",
    loginTitle: "需要重新打开控制台",
    loginBody: "为了安全，控制台只能用主机给的一次性链接打开。在这台电脑上运行：",
    step1: "接入模型", step2: "添加设备", step3: "试一下",
    w1Title: "先接入一个模型",
    w1Body: "用订阅的：在这台电脑上装好并登录一次官方程序（Claude Pro / Max 装 Claude Code，运行 claude 登录；ChatGPT Plus / Pro 装 Codex，运行 codex login），再刷新这个页面就自动接上，登录只留在这台电脑。也可以粘贴一个按用量付费的 API Key（只存在这台电脑的钥匙串里），或者装 Ollama 用免费的本地模型。",
    ollamaFound: "发现了这台电脑上的本地模型（Ollama）",
    ollamaUse: "直接使用本地模型",
    keyTitle: "粘贴一个 API Key", keyTitleOr: "或者粘贴一个 API Key",
    provider: "提供方", otherCompatible: "其他 OpenAI 兼容服务",
    cname: "名字（英文，如 groq）", curl: "接口地址（https://）",
    key: "API Key",
    keyHint: "会先用它做一次最小的请求来确认能用，成功才保存。",
    connect: "验证并接入",
    w2Title: "添加一台设备",
    skipPair: "先跳过，以后再加",
    w3Title: "试一下",
    model: "模型", message: "消息", send: "发送", finish: "完成",
    todaySpend: "今天的花费", estimate: "估算", devices: "设备", models: "模型", apps: "应用", usage: "用量",
    addDevice: "添加设备", addProvider: "接入模型", addApp: "添加应用",
    name: "名字", lastSeen: "最近使用", budget: "每天额度", where: "数据去向", humanOnly: "只给人用",
    providers: "提供方", keyLast4: "Key 末 4 位", subLogin: "订阅登录（留在本机）",
    noDevices: "还没有设备。点“添加设备”。",
    appsBody: "给不能直接嵌入 OmniLLM 的程序（脚本、其他 App）用。应用只能做后台调用，只能用 API Key 和本地模型。",
    usageBody: "按提供方公布的价格估算；没设价格的模型按偏高的价格估算。",
    who: "谁", spend: "花费", calls: "次数", tokens: "token（入 / 出）",
    remove: "移除", removeConfirm: "确定移除“{0}”？它会立即无法使用。",
    removed: "已移除", connected: "已接入，{0} 个模型可用", saved: "已保存",
    local: "只在本机", remote: "会发到云端", sub: "订阅",
    pairHow: "手机或平板：用相机扫这个二维码，不用装任何东西。电脑或服务器：运行 omnillm connect pair，粘贴下面这段配对文字（{0} 前有效，只能用一次）。不要发给别人，也不要贴到聊天里。",
    pairPhone: "手机或平板：用相机扫这个二维码，不用装任何东西。{0} 前有效，只能用一次。",
    pairComputer: "电脑或服务器（没有摄像头）",
    pairComputerNew: "还没装 OmniLLM 的，在那台机器上运行这一行（只装一个程序，不装后台服务），装完会提示粘贴配对文字：",
    pairComputerHow: "已经装好的，运行 omnillm connect pair，粘贴下面这段配对文字。不要发给别人，也不要贴到聊天里。",
    pairWait: "等待设备连接……",
    pairConfirmQ: "设备“{0}”请求加入。它屏幕上显示的数字是这个吗？",
    pairYes: "一致，添加", pairNo: "不一致，拒绝",
    pairDone: "已添加“{0}”", pairFailed: "配对没有完成：{0}",
    pairDenied: "已拒绝",
    noRelay: "还没有配置连接服务，其他设备暂时连不上。可以先跳过；配好连接服务后，在“设备”里添加。",
    relayWait: "正在连上连接服务……", relayDown: "连接服务现在连不上，请检查这台电脑的网络。连上之后这里会自动继续。",
    relayOn: "连接服务已连上", relayOff: "连接服务未连上", relayNone: "未配置连接服务",
    regionPaused: "主机所在地区变了，已暂停调用。", regionConfirm: "确认继续",
    appToken: "“{0}”的令牌（只显示这一次，请现在复制保存）：",
    appName: "应用的名字",
    tryText: "你好",
    never: "—", copy: "复制", copied: "已复制",
  },
  en: {
    loading: "Loading…",
    loginTitle: "Open the console again",
    loginBody: "For safety the console opens only through a one-time link from the hub. On this computer, run:",
    step1: "Connect a model", step2: "Add a device", step3: "Try it",
    w1Title: "First, connect a model",
    w1Body: "With a subscription: install the official program on this computer and log in once (Claude Pro / Max: Claude Code, run claude; ChatGPT Plus / Pro: Codex, run codex login), then reload this page and it connects by itself; the login stays on this computer. Or paste a pay-as-you-go API key (kept only in this computer's keychain), or install Ollama for free local models.",
    ollamaFound: "A local model (Ollama) is running on this computer",
    ollamaUse: "Use the local model",
    keyTitle: "Paste an API key", keyTitleOr: "Or paste an API key",
    provider: "Provider", otherCompatible: "Other OpenAI-compatible service",
    cname: "Name (e.g. groq)", curl: "API address (https://)",
    key: "API key",
    keyHint: "It is checked with one minimal request first and saved only if it works.",
    connect: "Check and connect",
    w2Title: "Add a device",
    skipPair: "Skip for now",
    w3Title: "Try it",
    model: "Model", message: "Message", send: "Send", finish: "Done",
    todaySpend: "Spent today", estimate: "estimate", devices: "Devices", models: "Models", apps: "Apps", usage: "Usage",
    addDevice: "Add device", addProvider: "Connect a model", addApp: "Add app",
    name: "Name", lastSeen: "Last used", budget: "Daily budget", where: "Data goes", humanOnly: "People only",
    providers: "Providers", keyLast4: "Key ends", subLogin: "subscription login (stays here)",
    noDevices: "No devices yet. Choose \"Add device\".",
    appsBody: "For programs that cannot embed OmniLLM (scripts, other apps). Apps make background calls only, with API keys and local models.",
    usageBody: "Estimated from the providers' published prices; models without a price are estimated high.",
    who: "Who", spend: "Spend", calls: "Calls", tokens: "tokens (in / out)",
    remove: "Remove", removeConfirm: "Remove \"{0}\"? It stops working at once.",
    removed: "Removed", connected: "Connected: {0} models available", saved: "Saved",
    local: "stays on this computer", remote: "goes to the cloud", sub: "subscription",
    pairHow: "Phone or tablet: scan the QR code with the camera; nothing to install. Computer or server: run omnillm connect pair and paste this pairing text (valid until {0}, single use). Don't send it to anyone or paste it into a chat.",
    pairPhone: "Phone or tablet: scan this QR code with the camera; nothing to install. Valid until {0}, single use.",
    pairComputer: "Computer or server (no camera)",
    pairComputerNew: "Not installed there yet? Run this line on that machine (one program, no background service); it then asks for the pairing text:",
    pairComputerHow: "Already installed: run omnillm connect pair and paste this pairing text. Don't send it to anyone or paste it into a chat.",
    pairWait: "Waiting for the device…",
    pairConfirmQ: "\"{0}\" wants to join. Does its screen show this number?",
    pairYes: "It matches, add it", pairNo: "It doesn't, refuse",
    pairDone: "Added \"{0}\"", pairFailed: "Pairing did not finish: {0}",
    pairDenied: "Refused",
    noRelay: "No connection service is set up yet, so other devices can't connect. Skip for now; add devices under Devices once it is set up.",
    relayWait: "Connecting to the connection service…", relayDown: "The connection service can't be reached right now; check this computer's network. This continues by itself once it connects.",
    relayOn: "Connection service online", relayOff: "Connection service offline", relayNone: "No connection service",
    regionPaused: "The host's region changed; calls are paused.", regionConfirm: "Confirm and resume",
    appToken: "Token for \"{0}\" (shown once — copy it now):",
    appName: "App name",
    tryText: "Hello",
    never: "—", copy: "Copy", copied: "Copied",
  },
};

function store(k, v) { try { if (v === undefined) return localStorage.getItem(k); localStorage.setItem(k, v); } catch (_) { return null; } }

let lang = store("omnillm.lang") || ((navigator.language || "en").toLowerCase().startsWith("zh") ? "zh" : "en");
const t = (k, ...a) => (T[lang][k] || k).replace(/\{(\d)\}/g, (_, i) => String(a[+i] ?? ""));
const $ = (id) => document.getElementById(id);
const el = (tag, text, cls) => { const e = document.createElement(tag); if (text !== undefined) e.textContent = text; if (cls) e.className = cls; return e; };

function applyLang() {
  document.documentElement.lang = lang === "zh" ? "zh-CN" : "en";
  document.querySelectorAll("[data-t]").forEach((n) => { n.textContent = t(n.dataset.t); });
  $("lang").textContent = lang === "zh" ? "English" : "中文";
  $("try-text").value = t("tryText");
}

function applyTheme(v) {
  if (v) document.documentElement.dataset.theme = v; else delete document.documentElement.dataset.theme;
}

// The session secret lives in this origin's sessionStorage (not shared
// with other localhost ports, unlike cookies) and goes with every call.
function sessionSecret(v) { try { if (v === undefined) return sessionStorage.getItem("omnillm.session") || ""; sessionStorage.setItem("omnillm.session", v); } catch (_) { return ""; } }

async function api(path, body) {
  const opts = { method: body === undefined ? "GET" : "POST", credentials: "same-origin",
    headers: { "X-OmniLLM-Console": "1", "X-OmniLLM-Lang": lang, "X-OmniLLM-Session": sessionSecret() } };
  if (body !== undefined) { opts.headers["Content-Type"] = "application/json"; opts.body = JSON.stringify(body); }
  const r = await fetch("/console/api/" + path, opts);
  let data = null;
  try { data = await r.json(); } catch (_) { data = null; }
  if (!r.ok) {
    const e = new Error((data && data.error && data.error.message) || ("HTTP " + r.status));
    e.status = r.status;
    throw e;
  }
  return data;
}

let toastTimer;
function toast(msg, err) {
  const n = $("toast");
  n.textContent = msg;
  n.className = "toast" + (err ? " err" : "");
  n.hidden = false;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => { n.hidden = true; }, err ? 6000 : 2500);
}

const money = (micro) => "$" + (micro / 1e6).toFixed(2);
const when = (iso) => { if (!iso || iso.startsWith("0001")) return t("never"); const d = new Date(iso); return d.toLocaleString(lang === "zh" ? "zh-CN" : "en-US", { dateStyle: "short", timeStyle: "short" }); };

let S = null;

async function refresh() {
  S = await api("status");
  for (const k of ["providers", "models", "devices", "apps", "usage"]) S[k] = S[k] || [];
  renderStatus();
  if (!wizardDone() && (S.providers.length === 0 || (S.devices.length === 0 && !store("omnillm.skipPair")))) {
    showWizard();
  } else {
    showDash();
  }
}

// The hub reaches the relay a moment after it starts, and reconnects by
// itself after a drop: keep the status line current instead of showing
// what was true when the page opened.
setInterval(async () => {
  if (!S || document.hidden) return;
  let r;
  try { r = await api("relay"); } catch (_) { return; }
  if (r.relay_up !== S.relay_up || r.relay_configured !== S.relay_configured) {
    S.relay_up = r.relay_up; S.relay_configured = r.relay_configured;
    renderStatus();
  }
}, 3000);

function wizardDone() { return store("omnillm.wizard") === "done"; }

function renderStatus() {
  const s = $("status");
  s.textContent = "";
  const dot = el("span", "", "dot" + (S.relay_up ? " on" : ""));
  s.append(dot, document.createTextNode(S.relay_configured ? (S.relay_up ? t("relayOn") : t("relayOff")) : t("relayNone")));
  if (S.region && S.region.paused) {
    const b = el("button", t("regionConfirm"), "link");
    b.type = "button";
    b.addEventListener("click", async () => { await api("region/confirm", {}); refresh(); });
    s.append(document.createTextNode(" · " + t("regionPaused") + " "), b);
  }
}

function show(id) { ["loading", "login-needed", "wizard", "dash"].forEach((x) => { $(x).hidden = x !== id; }); }

// ---------- wizard ----------

function stepTo(n) {
  [1, 2, 3].forEach((i) => {
    $("w" + i).hidden = i !== n;
    $("st" + i).className = i < n ? "done" : i === n ? "now" : "";
  });
}

async function showWizard() {
  show("wizard");
  if (S.providers.length === 0) {
    stepTo(1);
    try {
      const o = await api("ollama");
      const chat = (o.models || []).filter((m) => !/embed|bge|nomic|minilm|e5-/i.test(m));
      if (o.available && chat.length) {
        $("ollama-box").hidden = false;
        $("ollama-models").textContent = chat.join(", ");
        document.querySelector('#key-form [data-t^="keyTitle"]').dataset.t = "keyTitleOr";
        applyLang();
      } else if (o.available) {
        $("ollama-box").hidden = true;
      }
    } catch (_) { /* no local model */ }
  } else if (S.devices.length === 0 && !store("omnillm.skipPair")) {
    stepTo(2);
    startPair($("pair-area"), () => { stepTo(3); fillModels(); });
  } else {
    stepTo(3);
    fillModels();
  }
}

function fillModels() {
  const sel = $("try-model");
  sel.textContent = "";
  for (const m of S.models) {
    const o = el("option", m.display_name + (m.locality === "local" ? " · " + t("local") : isSub(m) ? " · " + t("sub") : ""));
    o.value = m.id;
    sel.append(o);
  }
}

$("kind").addEventListener("change", () => { $("compat").hidden = $("kind").value !== "openai-compatible"; });

$("key-form").addEventListener("submit", async (ev) => {
  ev.preventDefault();
  const btn = ev.submitter || $("key-form").querySelector("button");
  btn.disabled = true;
  const body = { kind: $("kind").value, key: $("key").value.trim() };
  if (body.kind === "openai-compatible") { body.name = $("cname").value.trim(); body.base_url = $("curl").value.trim(); }
  try {
    const r = await api("providers/add", body);
    $("key").value = "";
    toast(t("connected", r.models.length));
    await refresh();
  } catch (e) { toast(e.message, true); } finally { btn.disabled = false; }
});

$("ollama-use").addEventListener("click", async (ev) => {
  ev.target.disabled = true;
  try {
    const r = await api("providers/add", { kind: "ollama", key: "" });
    toast(t("connected", r.models.length));
    await refresh();
  } catch (e) { toast(e.message, true); } finally { ev.target.disabled = false; }
});

$("skip-pair").addEventListener("click", () => { store("omnillm.skipPair", "1"); stopPair(); stepTo(3); fillModels(); });

$("try-form").addEventListener("submit", async (ev) => {
  ev.preventDefault();
  const btn = ev.submitter || $("try-form").querySelector("button");
  btn.disabled = true;
  const out = $("try-reply");
  out.hidden = false;
  out.textContent = "…";
  try {
    const r = await api("test", { model: $("try-model").value, text: $("try-text").value });
    out.textContent = r.reply;
    $("finish").hidden = false;
  } catch (e) { out.textContent = e.message; } finally { btn.disabled = false; }
});

$("finish").addEventListener("click", () => { store("omnillm.wizard", "done"); refresh(); });

// ---------- pairing ----------

let pairTimer = null;
function stopPair() { if (pairTimer) { clearInterval(pairTimer); pairTimer = null; } }

async function startPair(area, onDone) {
  stopPair();
  area.textContent = "";
  if (!S.relay_configured) {
    area.append(el("p", t("noRelay"), "muted"));
    return;
  }
  if (!S.relay_up) {
    // Configured but not connected yet (just started, or the network is
    // down): wait for it here and carry on by itself.
    const w = el("p", t("relayWait"), "muted");
    area.append(w);
    const since = Date.now();
    pairTimer = setInterval(async () => {
      let r;
      try { r = await api("relay"); } catch (_) { return; }
      S.relay_up = r.relay_up; S.relay_configured = r.relay_configured;
      if (r.relay_up || !r.relay_configured) { renderStatus(); startPair(area, onDone); return; }
      if (Date.now() - since > 15000) w.textContent = t("relayDown");
    }, 1000);
    return;
  }
  let p;
  try { p = await api("pair/start", {}); } catch (e) { area.append(el("p", e.message, "muted")); return; }
  // Phones first: a big QR code and one sentence. Computers and servers
  // (no camera) open the section below it.
  const box = el("div", undefined, "pair");
  const exp = new Date(p.expires).toLocaleTimeString(lang === "zh" ? "zh-CN" : "en-US", { timeStyle: "short" });
  const left = el("div", undefined, "pair-qr");
  const img = el("img");
  img.alt = "QR";
  img.src = p.qr;
  left.append(img, el("p", t("pairPhone", exp), "muted small"));
  const right = el("div");
  const state = el("p", t("pairWait"), "muted");
  right.append(state);
  const more = el("details", undefined, "pair-more");
  more.append(el("summary", t("pairComputer")));
  const copyBtn = (text) => {
    const b = el("button", t("copy"), "secondary small-btn");
    b.type = "button";
    b.addEventListener("click", async () => { try { await navigator.clipboard.writeText(text); b.textContent = t("copied"); } catch (_) { /* select instead */ } });
    return b;
  };
  const install = "curl -fsSL https://raw.githubusercontent.com/wensuper/OmniLLM/main/install.sh -o omnillm-install.sh && sh omnillm-install.sh --device";
  more.append(el("p", t("pairComputerNew")), el("div", install, "code"), copyBtn(install),
    el("p", t("pairComputerHow")), el("div", p.code, "code"), copyBtn(p.code));
  right.append(more);
  box.append(left, right);
  area.append(box);
  let asked = false;
  pairTimer = setInterval(async () => {
    let st;
    try { st = await api("pair/state?id=" + encodeURIComponent(p.id)); } catch (e) { stopPair(); state.textContent = e.message; return; }
    if (st.state === "confirm" && !asked) {
      asked = true;
      const c = el("div", undefined, "confirm");
      c.append(el("p", t("pairConfirmQ", st.device_name)), el("div", String(st.code).replace(/^(\d{3})(\d{3})$/, "$1 $2"), "sas"));
      const actions = el("div", undefined, "actions");
      const yes = el("button", t("pairYes"), "primary");
      const no = el("button", t("pairNo"), "secondary");
      yes.type = no.type = "button";
      const decide = async (v) => { yes.disabled = no.disabled = true; try { await api("pair/decide", { id: p.id, yes: v }); } catch (e) { toast(e.message, true); } };
      yes.addEventListener("click", () => decide(true));
      no.addEventListener("click", () => decide(false));
      actions.append(yes, no);
      c.append(actions);
      state.replaceWith(c);
    }
    if (st.state === "done") {
      stopPair();
      area.textContent = "";
      area.append(el("p", t("pairDone", st.device_name)));
      toast(t("pairDone", st.device_name));
      onDone && onDone();
    }
    if (st.state === "failed") {
      stopPair();
      area.textContent = "";
      area.append(el("p", t("pairFailed", st.message || "")));
    }
  }, 1000);
}

// ---------- dashboard ----------

function showDash() {
  show("dash");
  const total = (S.usage || []).find((u) => u.who === "_total");
  $("spend-today").textContent = money(total ? total.micro_usd : 0);
  $("n-devices").textContent = String(S.devices.length);
  $("n-models").textContent = String(S.models.length);
  renderDevices();
  renderModels();
  renderApps();
  renderUsage();
}

// isSub: a Claude or ChatGPT subscription (the official programs on this
// computer), not merely a model kept for people.
function isSub(m) { return m.provider === "claude-code" || m.provider === "codex"; }
function cell(row, text, cls) { const td = el("td", text, cls); row.append(td); return td; }

function removeButton(label, path, id) {
  const b = el("button", t("remove"), "danger");
  b.type = "button";
  b.addEventListener("click", async () => {
    if (!window.confirm(t("removeConfirm", label))) return;
    try { await api(path, { id }); toast(t("removed")); refresh(); } catch (e) { toast(e.message, true); }
  });
  return b;
}

function renderDevices() {
  const tb = $("devices");
  tb.textContent = "";
  for (const d of S.devices) {
    const tr = el("tr");
    cell(tr, d.Name);
    cell(tr, when(d.LastSeen));
    cell(tr, money(d.BudgetMicroUSD), "num");
    cell(tr, "").append(removeButton(d.Name, "devices/remove", d.ID));
    tb.append(tr);
  }
  $("devices-empty").hidden = S.devices.length > 0;
}

function renderModels() {
  const tb = $("models");
  tb.textContent = "";
  for (const m of S.models) {
    const tr = el("tr");
    cell(tr, m.display_name + (isSub(m) ? " · " + t("sub") : ""));
    cell(tr, "").append(el("span", m.locality === "local" ? t("local") : t("remote"), "tag" + (m.locality === "local" ? " local" : "")));
    const td = cell(tr, "");
    const box = el("input");
    box.type = "checkbox";
    box.checked = !!m.human_only;
    box.setAttribute("aria-label", t("humanOnly") + " " + m.display_name);
    box.addEventListener("change", async () => {
      try { await api("models/set", { id: m.id, human_only: box.checked }); toast(t("saved")); } catch (e) { box.checked = !box.checked; toast(e.message, true); }
    });
    td.append(box);
    tb.append(tr);
  }
  const pb = $("providers");
  pb.textContent = "";
  for (const p of S.providers) {
    const tr = el("tr");
    cell(tr, (p.display_name || p.name) + (p.local ? " · " + t("local") : ""));
    cell(tr, p.subscription ? t("subLogin") : p.key_last4 ? "…" + p.key_last4 : t("never"), "num");
    cell(tr, "").append(removeButton(p.name, "providers/remove", p.name));
    pb.append(tr);
  }
}

function renderApps() {
  const tb = $("apps");
  tb.textContent = "";
  for (const a of S.apps) {
    const tr = el("tr");
    cell(tr, a.Name);
    cell(tr, money(a.BudgetMicroUSD), "num");
    cell(tr, "").append(removeButton(a.Name, "apps/remove", a.ID));
    tb.append(tr);
  }
}

function renderUsage() {
  const tb = $("usage");
  tb.textContent = "";
  const names = {};
  for (const d of S.devices) names["device/" + d.ID] = d.Name;
  for (const a of S.apps) names["app/" + a.ID] = a.Name;
  for (const u of (S.usage || []).filter((x) => x.who !== "_total")) {
    const tr = el("tr");
    cell(tr, names[u.who] || (u.who === "host" ? "OmniLLM" : u.who));
    cell(tr, money(u.micro_usd), "num");
    cell(tr, String(u.calls), "num");
    cell(tr, u.input_tokens + " / " + u.output_tokens, "num");
    tb.append(tr);
  }
}

document.querySelectorAll(".tabs button").forEach((b) => {
  b.addEventListener("click", () => {
    document.querySelectorAll(".tabs button").forEach((x) => x.setAttribute("aria-selected", String(x === b)));
    document.querySelectorAll(".tab").forEach((x) => { x.hidden = x.id !== "tab-" + b.dataset.tab; });
    store("omnillm.tab", b.dataset.tab);
  });
});

$("add-device").addEventListener("click", () => startPair($("pair-dash"), refresh));
$("add-provider").addEventListener("click", () => { store("omnillm.wizard", ""); S.providers = []; showWizard(); });
$("add-app").addEventListener("click", async () => {
  const name = window.prompt(t("appName"));
  if (!name) return;
  try {
    const r = await api("apps/add", { name });
    const box = $("app-token");
    box.hidden = false;
    box.textContent = t("appToken", r.name) + "\n" + r.token;
    refresh();
  } catch (e) { toast(e.message, true); }
});

$("lang").addEventListener("click", () => { lang = lang === "zh" ? "en" : "zh"; store("omnillm.lang", lang); applyLang(); if (S) { renderStatus(); if (!$("dash").hidden) showDash(); } });
$("theme").addEventListener("click", () => {
  const cur = document.documentElement.dataset.theme || (matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light");
  const next = cur === "dark" ? "light" : "dark";
  store("omnillm.theme", next);
  applyTheme(next);
});

async function boot() {
  applyLang();
  applyTheme(store("omnillm.theme"));
  const tab = store("omnillm.tab") || "devices";
  const tb = document.querySelector('.tabs button[data-tab="' + tab + '"]') || document.querySelector(".tabs button");
  tb.click();
  const m = /(?:^|[#&])login=([0-9a-f]{64})/.exec(location.hash);
  if (m) {
    history.replaceState(null, "", "/console/");
    try { const r = await api("login", { token: m[1] }); sessionSecret(r.session); } catch (e) { show("login-needed"); return; }
  }
  try { await refresh(); } catch (e) { if (e.status === 401) show("login-needed"); else toast(e.message, true); }
}
boot();
