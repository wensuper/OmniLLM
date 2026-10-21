// OmniLLM console. Everything the server returns is shown with textContent;
// nothing from the server or from a device is ever inserted as HTML.
"use strict";

// Words for people, the same in every OmniLLM screen: the hub is "your Mac"
// (here: this Mac), the relay is "remote access", a provider is a "model
// source". Text in `backticks` is shown in code style (see rich()).
const T = {
  zh: {
    loading: "正在加载……",
    loginTitle: "需要重新打开控制台",
    loginBody: "点屏幕顶部菜单栏里的 OmniLLM 图标，选“打开控制台”。",
    loginOr: "或者在终端运行：",
    hubDown: "OmniLLM 没有运行。点菜单栏里的 OmniLLM 图标，选“启动 OmniLLM”。",
    step1: "接入模型", step2: "添加设备", step3: "试一下",
    w1Title: "先接入一个模型",
    w1Claude: "我有 Claude 订阅",
    w1Claude1: "在这台 Mac 上安装 Claude Code。",
    w1Claude2: "在终端运行 `claude` 并登录。",
    w1Codex: "我有 ChatGPT 订阅",
    w1Codex1: "在这台 Mac 上安装 Codex。",
    w1Codex2: "在终端运行 `codex login`。",
    w1Key: "我有 API Key",
    w1Free: "想免费：安装 Ollama，用这台 Mac 上的模型。",
    recheck: "重新检查", recheckNone: "还没发现已登录的订阅或本机模型。",
    ollamaFound: "发现了这台 Mac 上的本地模型（Ollama）",
    ollamaUse: "直接使用本地模型",
    source: "来源", otherCompatible: "其他 OpenAI 兼容服务",
    cname: "名字（英文，如 groq）", curl: "接口地址（https://）",
    key: "API Key",
    keyHint: "会先用它做一次最小的请求来确认能用，成功才保存；只存在这台 Mac 的钥匙串里。",
    connect: "验证并接入",
    w2Title: "添加一台设备",
    skipPair: "先跳过，以后再加",
    w3Title: "试一下",
    model: "模型", message: "消息", send: "发送", finish: "完成",
    todaySpend: "今天的花费", estimate: "按公布的价格估算", devices: "设备", models: "模型", apps: "应用", usage: "用量",
    addDevice: "添加设备", addProvider: "接入模型", addApp: "添加应用",
    name: "名字", lastSeen: "最近使用", budget: "每天额度", where: "对话发往", humanOnly: "只能手动对话",
    providers: "模型来源", keyLast4: "接入方式", subLogin: "订阅登录，只留在这台 Mac",
    noDevices: "还没有设备。点“添加设备”。",
    appsBody: "给不能直接嵌入 OmniLLM 的程序（脚本、其他 App）用。应用只能做后台调用，只能用 API Key 和这台 Mac 上的模型。",
    usageBody: "今天的用量（按这台 Mac 的时间）。花费按模型来源公布的价格估算，没有公布价格的按偏高的价格估；订阅和这台 Mac 上的模型不花钱。",
    who: "设备或应用", spend: "花费（估算）", calls: "次数", tokens: "文字量（发出 / 收到）", tokensTip: "按 token 计",
    usageHost: "这台 Mac", usageGone: "已移除的设备和应用（{0} 个）", usageGoneOne: "{0}（已移除）", usageTotal: "今天合计",
    capLine: "整台 Mac 每天总额度 {0}", capNone: "整台 Mac 每天总额度：不设上限", capEdit: "修改", noApps: "还没有应用。",
    modelsBody: "“只能手动对话”的模型只给在手机、电脑上聊天的人用，脚本和应用调用不了。订阅模型一直是这样，免得账号被当成自动化调用。",
    claudeSub: "Claude 订阅", codexSub: "ChatGPT 订阅（Codex）", usageEmpty: "今天还没有用过。",
    remove: "移除", removeConfirm: "确定移除“{0}”？它会立即无法使用。",
    removed: "已移除", connected: "已接入，{0} 个模型可用", saved: "已保存",
    local: "只在这台 Mac", remote: "云端", sub: "订阅",
    todayOf: "今天 {0} / {1}", todayNoLimit: "今天 {0} · 不设上限",
    pairPhone: "手机或平板：用相机扫这个二维码，不用装任何东西。{0} 前有效，只能用一次。",
    pairComputer: "电脑或服务器（没有摄像头）",
    pairComputerNew: "还没装 OmniLLM 的，在那台电脑上运行这一行（只用来聊天，不会一直在后台运行），装完会提示粘贴配对文字：",
    pairComputerHow: "已经装好的，运行 `omnillm connect pair`，粘贴下面这段配对文字。不要发给别人，也不要贴到聊天里。",
    pairWait: "等待设备连接……",
    pairConfirmQ: "设备“{0}”请求加入。它屏幕上显示的数字是这个吗？",
    pairYes: "一致，添加", pairNo: "不一致，拒绝",
    pairDone: "已添加“{0}”", pairFailed: "配对没有完成：{0}",
    pairDenied: "已拒绝。", pairExpired: "这个二维码已过期。", pairNew: "换一个二维码",
    pairStartOffline: "远程连接现在没有连上，生成不了二维码。", retry: "重试",
    close: "关闭", edit: "编辑", save: "保存", cancel: "取消", noLimit: "不设上限",
    noLimitHint: "不设上限时只受整台 Mac 每天总额度（{0}）限制；订阅模型不按钱计。", nameLabel: "设备名字", budgetLabel: "每天额度（美元）",
    noRelay: "还没有设置远程连接，其他设备暂时连不上。可以先跳过；设置好远程连接后，在“设备”里添加。",
    relayWait: "正在打开远程连接……", relayDown: "远程连接现在连不上，请检查这台 Mac 的网络。连上之后这里会自动继续。",
    relayOn: "设备可以连上", relayOff: "远程连接断开，设备暂时连不上", relayNone: "只在这台 Mac 上用（没有远程连接）",
    regionPaused: "这台 Mac 所在的地区变了，已暂停调用。", regionConfirm: "确认继续",
    appToken: "“{0}”的令牌（只显示这一次，请现在复制保存）：",
    appName: "应用的名字",
    tryText: "你好",
    never: "—", copy: "复制", copied: "已复制",
  },
  en: {
    loading: "Loading…",
    loginTitle: "Open the console again",
    loginBody: "Click the OmniLLM icon in the menu bar and choose Open the console.",
    loginOr: "Or run in Terminal:",
    hubDown: "OmniLLM isn't running. Click the OmniLLM icon in the menu bar and choose Start OmniLLM.",
    step1: "Connect a model", step2: "Add a device", step3: "Try it",
    w1Title: "First, connect a model",
    w1Claude: "I have a Claude subscription",
    w1Claude1: "Install Claude Code on this Mac.",
    w1Claude2: "In Terminal, run `claude` and log in.",
    w1Codex: "I have a ChatGPT subscription",
    w1Codex1: "Install Codex on this Mac.",
    w1Codex2: "In Terminal, run `codex login`.",
    w1Key: "I have an API key",
    w1Free: "Want it free? Install Ollama to use models on this Mac.",
    recheck: "Check again", recheckNone: "No signed-in subscription or local model found yet.",
    ollamaFound: "A local model (Ollama) is running on this Mac",
    ollamaUse: "Use the local model",
    source: "Source", otherCompatible: "Other OpenAI-compatible service",
    cname: "Name (e.g. groq)", curl: "API address (https://)",
    key: "API key",
    keyHint: "It is checked with one minimal request first and saved only if it works; it is kept only in this Mac's keychain.",
    connect: "Check and connect",
    w2Title: "Add a device",
    skipPair: "Skip for now",
    w3Title: "Try it",
    model: "Model", message: "Message", send: "Send", finish: "Done",
    todaySpend: "Spent today", estimate: "estimated from published prices", devices: "Devices", models: "Models", apps: "Apps", usage: "Usage",
    addDevice: "Add device", addProvider: "Connect a model", addApp: "Add app",
    name: "Name", lastSeen: "Last used", budget: "Daily limit", where: "Messages go to", humanOnly: "Chat only",
    providers: "Model sources", keyLast4: "Connected with", subLogin: "subscription login, stays on this Mac",
    noDevices: "No devices yet. Choose \"Add device\".",
    appsBody: "For programs that cannot embed OmniLLM (scripts, other apps). Apps make background calls only, with API keys and models on this Mac.",
    usageBody: "Today's use (this Mac's time). Spend is estimated from the model sources' published prices, high where there is no price; subscriptions and models on this Mac cost nothing.",
    who: "Device or app", spend: "Spend (est.)", calls: "Calls", tokens: "Text (sent / received)", tokensTip: "tokens",
    usageHost: "This Mac", usageGone: "Removed devices and apps ({0})", usageGoneOne: "{0} (removed)", usageTotal: "Total today",
    capLine: "Daily total for this Mac: {0}", capNone: "Daily total for this Mac: no limit", capEdit: "Change", noApps: "No apps yet.",
    modelsBody: "Chat-only models are for people chatting on a phone or computer; scripts and apps can't use them. Subscription models are always chat-only, so the account isn't flagged as automated.",
    claudeSub: "Claude subscription", codexSub: "ChatGPT subscription (Codex)", usageEmpty: "Nothing used yet today.",
    remove: "Remove", removeConfirm: "Remove \"{0}\"? It stops working at once.",
    removed: "Removed", connected: "Connected: {0} models available", saved: "Saved",
    local: "this Mac only", remote: "the cloud", sub: "subscription",
    todayOf: "Today {0} / {1}", todayNoLimit: "Today {0} · no limit",
    pairPhone: "Phone or tablet: scan this QR code with the camera; nothing to install. Valid until {0}, single use.",
    pairComputer: "Computer or server (no camera)",
    pairComputerNew: "Not installed there yet? Run this line on that computer (just for chatting; nothing keeps running in the background); it then asks for the pairing text:",
    pairComputerHow: "Already installed: run `omnillm connect pair` and paste this pairing text. Don't send it to anyone or paste it into a chat.",
    pairWait: "Waiting for the device…",
    pairConfirmQ: "\"{0}\" wants to join. Does its screen show this number?",
    pairYes: "It matches, add it", pairNo: "It doesn't, refuse",
    pairDone: "Added \"{0}\"", pairFailed: "Pairing did not finish: {0}",
    pairDenied: "Refused.", pairExpired: "This QR code has expired.", pairNew: "New QR code",
    pairStartOffline: "Remote access isn't connected right now, so no QR code can be made.", retry: "Retry",
    close: "Close", edit: "Edit", save: "Save", cancel: "Cancel", noLimit: "No limit",
    noLimitHint: "With no limit, only the daily total for this Mac ({0}) applies; subscription models aren't counted in money.", nameLabel: "Device name", budgetLabel: "Daily limit (USD)",
    noRelay: "Remote access isn't set up yet, so other devices can't connect. Skip for now; add devices under Devices once it is set up.",
    relayWait: "Turning on remote access…", relayDown: "Remote access can't connect right now; check this Mac's network. This continues by itself once it connects.",
    relayOn: "Devices can connect", relayOff: "Remote access is down; devices can't connect now", relayNone: "This Mac only (no remote access)",
    regionPaused: "This Mac's region changed; calls are paused.", regionConfirm: "Confirm and resume",
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

// rich sets text where `backticked` parts are commands, shown in code style.
// Still text only: the parts become text nodes and <code> elements.
function rich(node, text) {
  node.textContent = "";
  text.split("`").forEach((part, i) => { if (part) node.append(i % 2 ? el("code", part) : document.createTextNode(part)); });
  return node;
}

// copyBtn copies text and says so on the button.
function copyBtn(text, b) {
  b = b || el("button", t("copy"), "secondary small-btn copy");
  b.type = "button";
  b.addEventListener("click", async () => { try { await navigator.clipboard.writeText(text); b.textContent = t("copied"); } catch (_) { /* the block is user-select: all */ } });
  return b;
}

// codeBox: a command or pairing text with its Copy button in the top-right
// corner, so the button never sits between two blocks.
function codeBox(text) {
  const box = el("div", undefined, "codebox");
  box.append(el("div", text, "code"), copyBtn(text));
  return box;
}

function applyLang() {
  document.documentElement.lang = lang === "zh" ? "zh-CN" : "en";
  document.querySelectorAll("[data-t]").forEach((n) => { if (n.hasAttribute("data-rich")) rich(n, t(n.dataset.t)); else n.textContent = t(n.dataset.t); });
  document.querySelectorAll("[data-tip]").forEach((n) => { n.title = t(n.dataset.tip); });
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
    e.code = data && data.error && data.error.code;
    // Any call that finds the session gone (the hub restarted, or the
    // session timed out) leads to the way back in, not to an error toast.
    if (r.status === 401 && path !== "login") loggedOut();
    throw e;
  }
  return data;
}

// loggedOut: the session is gone; stop everything that polls and show how
// to open the console again.
let signedOut = false;
function loggedOut() {
  signedOut = true;
  stopPair();
  $("hub-down").hidden = true;
  show("login-needed");
}

let toastTimer;
function toast(msg, err) {
  if (signedOut) return; // the login panel already says what to do
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
// what was true when the page opened. A poll that gets no answer at all
// means the hub itself stopped: say so, and how to start it again.
setInterval(async () => {
  if (!S || signedOut || document.hidden) return;
  let r;
  try { r = await api("relay"); } catch (e) {
    if (e.status === undefined && $("hub-down").hidden) { $("hub-down").hidden = false; S.relay_up = false; renderStatus(); }
    return;
  }
  $("hub-down").hidden = true;
  if (r.relay_up !== S.relay_up || r.relay_configured !== S.relay_configured) {
    S.relay_up = r.relay_up; S.relay_configured = r.relay_configured;
    renderStatus();
  }
}, 3000);

// Devices, usage and models change while the page is open (a phone pairs,
// someone chats): redraw the dashboard every 10 s, but never while
// something is being edited or a device is being added, so nothing typed
// or on screen is thrown away.
function busy() {
  return !!(curPair || pairTimer || document.querySelector("tr.editing, .cap-edit, .pair-panel"));
}
setInterval(async () => {
  if (!S || signedOut || document.hidden || $("dash").hidden || busy()) return;
  let next;
  try { next = await api("status"); } catch (_) { return; }
  if ($("dash").hidden || busy()) return;
  for (const k of ["providers", "models", "devices", "apps", "usage"]) next[k] = next[k] || [];
  S = next;
  renderStatus();
  showDash();
}, 10000);

function wizardDone() { return store("omnillm.wizard") === "done"; }

function renderStatus() {
  const s = $("status");
  s.textContent = "";
  let text = S.relay_configured ? (S.relay_up ? t("relayOn") : t("relayOff")) : t("relayNone");
  if (S.region && S.region.paused) text += " · " + t("regionPaused");
  // On a phone-width window only the dot shows; it carries the sentence
  // for screen readers and as a tooltip.
  const dot = el("span", "", "dot" + (S.relay_up ? " on" : ""));
  dot.setAttribute("role", "img");
  dot.setAttribute("aria-label", text);
  s.title = text;
  const words = el("span", text, "status-text");
  words.setAttribute("aria-hidden", "true");
  s.append(dot, words);
  if (S.region && S.region.paused) {
    const b = el("button", t("regionConfirm"), "link");
    b.type = "button";
    b.addEventListener("click", async () => { try { await api("region/confirm", {}); await refresh(); } catch (e) { toast(e.message, true); } });
    s.append(document.createTextNode(" "), b);
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

// The wizard's "Skip for now" line; startPair moves it under the QR code.
const skipLine = $("skip-line");

async function showWizard() {
  show("wizard");
  if (S.providers.length === 0) {
    stepTo(1);
    try {
      const o = await api("ollama");
      const chat = (o.models || []).filter((m) => !/embed|bge|nomic|minilm|e5-/i.test(m));
      $("ollama-box").hidden = !(o.available && chat.length);
      $("ollama-models").textContent = chat.join(", ");
    } catch (_) { /* no local model */ }
  } else if (S.devices.length === 0 && !store("omnillm.skipPair")) {
    stepTo(2);
    startPair($("pair-area"), () => { stepTo(3); fillModels(); }, false, skipLine);
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

// "Check again": a subscription logged in since the page opened is found
// by status itself (and Ollama by showWizard), no reload needed.
$("recheck").addEventListener("click", async (ev) => {
  const b = ev.currentTarget;
  b.disabled = true;
  try {
    await refresh();
    if (S.providers.length === 0) toast(t("recheckNone"));
  } catch (e) { toast(e.message, true); } finally { b.disabled = false; }
});

$("skip-pair").addEventListener("click", () => { store("omnillm.skipPair", "1"); closePair($("pair-area")); stepTo(3); fillModels(); });

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

// The pairing on screen, so closing the panel can end it at once. pairGen
// counts starts and closes, so a pairing whose start answers after the
// panel was closed (or started again) is dropped instead of shown.
let curPair = null, pairGen = 0;
function cancelPair() { if (curPair) { api("pair/cancel", { id: curPair }).catch(() => {}); curPair = null; } }
function closePair(area) {
  pairGen++;
  stopPair();
  cancelPair();
  $("add-device").hidden = false;
  area.textContent = "";
  area.classList.remove("pair-panel");
}

// startPair shows "add device" in area: a QR code for phones, pairing text
// for computers, then the owner's yes or no. closable: the dashboard's
// panel with its x. under: kept right under the QR caption, or under any
// message shown instead of the code (the wizard's "Skip for now").
async function startPair(area, onDone, closable, under) {
  const gen = ++pairGen;
  stopPair();
  cancelPair();
  if (closable) {
    $("add-device").hidden = true;
    area.classList.add("pair-panel");
  }
  // fresh empties the area but keeps the panel's x.
  const fresh = () => {
    area.textContent = "";
    if (!closable) return;
    const x = el("button", "", "close-x"); x.type = "button"; x.setAttribute("aria-label", t("close")); x.title = t("close");
    x.addEventListener("click", () => closePair(area));
    area.append(x);
  };
  // ended: no device this time (the code expired, was refused, or could
  // not be made). Say so in place and offer a new code right there, instead
  // of sending the owner back to "Add device".
  const ended = (msg, isErr, label) => {
    stopPair();
    cancelPair();
    fresh();
    const again = el("button", label || t("pairNew"), "primary");
    again.type = "button";
    again.addEventListener("click", () => startPair(area, onDone, closable, under));
    area.append(el("p", msg, isErr ? "err-text" : ""), again);
    if (under) area.append(under);
  };
  fresh();
  if (!S.relay_configured) {
    area.append(el("p", t("noRelay"), "muted"));
    if (under) area.append(under);
    return;
  }
  if (!S.relay_up) {
    // Configured but not connected yet (just started, or the network is
    // down): wait for it here and carry on by itself.
    const w = el("p", t("relayWait"), "muted");
    area.append(w);
    if (under) area.append(under);
    const since = Date.now();
    pairTimer = setInterval(async () => {
      let r;
      try { r = await api("relay"); } catch (_) { return; }
      if (gen !== pairGen) return;
      S.relay_up = r.relay_up; S.relay_configured = r.relay_configured;
      if (r.relay_up || !r.relay_configured) { renderStatus(); startPair(area, onDone, closable, under); return; }
      if (Date.now() - since > 15000) w.textContent = t("relayDown");
    }, 1000);
    return;
  }
  let p;
  try { p = await api("pair/start", {}); } catch (e) {
    if (gen !== pairGen || e.status === 401) return;
    // The hub's own sentence for a missing relay promises a reconnect that
    // this button is the way to; say plainly why there is no code.
    ended(e.status === undefined ? t("hubDown") : e.code === "hub_offline" ? t("pairStartOffline") : e.message, true, t("retry"));
    return;
  }
  if (gen !== pairGen) { api("pair/cancel", { id: p.id }).catch(() => {}); return; }
  curPair = p.id;
  const expires = new Date(p.expires).getTime();
  // Phones first: a big QR code and one sentence. Computers and servers
  // (no camera) open the section below it.
  const box = el("div", undefined, "pair");
  const exp = new Date(p.expires).toLocaleTimeString(lang === "zh" ? "zh-CN" : "en-US", { timeStyle: "short" });
  const left = el("div", undefined, "pair-qr");
  const img = el("img");
  img.alt = "QR";
  img.src = p.qr;
  left.append(img, el("p", t("pairPhone", exp), "muted small"));
  if (under) left.append(under);
  const right = el("div");
  const state = el("p", t("pairWait"), "muted");
  right.append(state);
  const more = el("details", undefined, "pair-more");
  more.append(el("summary", t("pairComputer")));
  const install = "curl -fsSL https://raw.githubusercontent.com/wensuper/OmniLLM/main/install.sh -o omnillm-install.sh && sh omnillm-install.sh --device";
  more.append(rich(el("p"), t("pairComputerNew")), codeBox(install),
    rich(el("p"), t("pairComputerHow")), codeBox(p.code));
  right.append(more);
  box.append(left, right);
  area.append(box);
  let asked = false;
  pairTimer = setInterval(async () => {
    let st;
    try { st = await api("pair/state?id=" + encodeURIComponent(p.id)); } catch (e) {
      if (gen !== pairGen) return;
      if (e.status === 404) { ended(t("pairExpired")); return; } // gone on the hub: expired
      if (e.status === undefined) return; // the hub isn't answering: the banner says so; keep asking
      if (e.status !== 401) ended(e.message, true, t("retry"));
      return;
    }
    if (gen !== pairGen) return;
    if (st.state === "waiting" && Date.now() > expires) { ended(t("pairExpired")); return; }
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
      curPair = null;
      area.textContent = "";
      area.append(el("p", t("pairDone", st.device_name)));
      toast(t("pairDone", st.device_name));
      onDone && onDone();
    }
    if (st.state === "failed") {
      curPair = null;
      if (st.reason === "denied") ended(t("pairDenied"));
      else if (st.reason === "expired") ended(t("pairExpired"));
      else ended(t("pairFailed", st.message || ""), true);
    }
  }, 1000);
}

// ---------- dashboard ----------

function showDash() {
  show("dash");
  const total = (S.usage || []).find((u) => u.who === "_total");
  $("spend-today").textContent = money(total ? total.micro_usd : 0);
  renderCap();
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

const NO_LIMIT = -1;
// spentText: what a device or app spent today against its own daily limit,
// so a limit is never read without how close it is.
function spentText(who, limit) {
  const u = S.usage.find((x) => x.who === who);
  const today = money(u ? u.micro_usd : 0);
  return limit === NO_LIMIT ? t("todayNoLimit", today) : t("todayOf", today, money(limit));
}

function renderDevices() {
  const tb = $("devices");
  tb.textContent = "";
  for (const d of S.devices) {
    const tr = el("tr");
    cell(tr, d.Name);
    cell(tr, when(d.LastSeen));
    cell(tr, spentText("device/" + d.ID, d.BudgetMicroUSD), "spent");
    const act = cell(tr, "", "actions");
    const ed = el("button", t("edit"), "secondary small-btn");
    ed.type = "button";
    ed.addEventListener("click", () => editRow(tr, { id: d.ID, name: d.Name, budget: d.BudgetMicroUSD, seen: d.LastSeen }, "devices/update", renderDevices));
    act.append(ed, removeButton(d.Name, "devices/remove", d.ID));
    tb.append(tr);
  }
  $("devices-empty").hidden = S.devices.length > 0;
}

// renderCap: the whole hub's daily total next to today's spend, changed in
// place (an amount, or no limit).
function renderCap() {
  const box = $("cap"); box.textContent = "";
  const v = S.daily_total_micro_usd;
  box.append(el("span", v === NO_LIMIT ? t("capNone") : t("capLine", money(v)), "muted"));
  const b = el("button", t("capEdit"), "link"); b.type = "button";
  b.addEventListener("click", () => {
    box.textContent = "";
    const f = el("div", undefined, "cap-edit");
    const wrap = el("span", undefined, "money-in");
    const amt = el("input"); amt.type = "number"; amt.min = "1"; amt.step = "1"; amt.inputMode = "decimal";
    amt.value = v === NO_LIMIT ? "20" : String(v / 1e6); amt.disabled = v === NO_LIMIT; amt.setAttribute("aria-label", t("capLine", ""));
    wrap.append(el("span", "$"), amt);
    const sw = el("input"); sw.type = "checkbox"; sw.className = "switch"; sw.checked = v === NO_LIMIT;
    const lab = el("label", undefined, "toggle"); lab.append(sw, el("span", t("noLimit")));
    sw.addEventListener("change", () => { amt.disabled = sw.checked; });
    const save = el("button", t("save"), "primary small-btn"); save.type = "button";
    const no = el("button", t("cancel"), "secondary small-btn"); no.type = "button";
    save.addEventListener("click", async () => {
      save.disabled = true;
      try { await api("daily-total", sw.checked ? { no_limit: true } : { budget_usd: Number(amt.value) }); toast(t("saved")); await refresh(); } catch (e) { toast(e.message, true); save.disabled = false; }
    });
    no.addEventListener("click", renderCap);
    f.append(wrap, lab, save, no);
    box.append(f);
    amt.focus();
  });
  box.append(b);
}

// editRow turns a device's or app's row into a small form: its name, and
// its daily limit or no limit. Enter saves, Escape puts the row back.
function editRow(tr, d, path, back) {
  tr.textContent = "";
  tr.className = "editing";
  const name = el("input"); name.value = d.name; name.maxLength = 64; name.setAttribute("aria-label", t("nameLabel"));
  cell(tr, "").append(name);
  if (d.seen !== undefined) cell(tr, when(d.seen));
  const bc = el("div", undefined, "budget-edit"); cell(tr, "").append(bc);
  const unl = d.budget === NO_LIMIT;
  const wrap = el("span", undefined, "money-in");
  const amt = el("input"); amt.type = "number"; amt.min = "0.5"; amt.step = "0.5"; amt.inputMode = "decimal";
  amt.value = unl ? "5" : String(d.budget / 1e6); amt.disabled = unl; amt.setAttribute("aria-label", t("budgetLabel"));
  wrap.append(el("span", "$"), amt);
  const sw = el("input"); sw.type = "checkbox"; sw.className = "switch"; sw.checked = unl;
  const lab = el("label", undefined, "toggle"); lab.append(sw, el("span", t("noLimit")));
  const hint = el("p", t("noLimitHint", S.daily_total_micro_usd === NO_LIMIT ? t("noLimit") : money(S.daily_total_micro_usd)), "muted small"); hint.hidden = !unl;
  sw.addEventListener("change", () => { amt.disabled = sw.checked; hint.hidden = !sw.checked; });
  bc.append(wrap, lab, hint);
  const act = cell(tr, "", "actions");
  const save = el("button", t("save"), "primary small-btn"); save.type = "button";
  const no = el("button", t("cancel"), "secondary small-btn"); no.type = "button";
  act.append(save, no);
  const submit = async () => {
    const body = { id: d.id, name: name.value };
    if (sw.checked) body.no_limit = true; else body.budget_usd = Number(amt.value);
    save.disabled = true;
    try { await api(path, body); toast(t("saved")); await refresh(); } catch (e) { toast(e.message, true); save.disabled = false; }
  };
  const done = () => back();
  save.addEventListener("click", submit);
  no.addEventListener("click", done);
  tr.addEventListener("keydown", (e) => { if (e.key === "Enter") submit(); if (e.key === "Escape") done(); });
  name.focus(); name.select();
}

function renderModels() {
  const tb = $("models");
  tb.textContent = "";
  for (const m of S.models) {
    const tr = el("tr");
    cell(tr, m.display_name + (isSub(m) ? " · " + t("sub") : ""));
    cell(tr, "").append(el("span", m.locality === "local" ? t("local") : t("remote"), "tag" + (m.locality === "local" ? " local" : "")));
    const td = cell(tr, "", "actions");
    const box = el("input");
    box.type = "checkbox";
    box.className = "switch";
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
    const label = p.name === "claude-code" ? t("claudeSub") : p.name === "codex" ? t("codexSub") : (p.display_name || p.name);
    cell(tr, label + (p.local ? " · " + t("local") : ""));
    cell(tr, p.subscription ? t("subLogin") : p.key_last4 ? "API Key …" + p.key_last4 : t("local"));
    cell(tr, "", "actions").append(removeButton(label, "providers/remove", p.name));
    pb.append(tr);
  }
}

function renderApps() {
  const tb = $("apps");
  tb.textContent = "";
  for (const a of S.apps) {
    const tr = el("tr");
    cell(tr, a.Name);
    cell(tr, spentText("app/" + a.ID, a.BudgetMicroUSD), "spent");
    const act = cell(tr, "", "actions");
    const ed = el("button", t("edit"), "secondary small-btn");
    ed.type = "button";
    ed.addEventListener("click", () => editRow(tr, { id: a.ID, name: a.Name, budget: a.BudgetMicroUSD }, "apps/update", renderApps));
    act.append(ed, removeButton(a.Name, "apps/remove", a.ID));
    tb.append(tr);
  }
  $("apps-empty").hidden = S.apps.length > 0;
}

// renderUsage: one line per device or app by its name (never an internal
// id), busiest first; removed devices folded into one line; today's total
// against the hub's daily limit at the bottom.
const num = (n) => Number(n || 0).toLocaleString(lang === "zh" ? "zh-CN" : "en-US");
function renderUsage() {
  const tb = $("usage"), foot = $("usage-total");
  tb.textContent = ""; foot.textContent = "";
  const rows = [], gone = { calls: 0, micro_usd: 0, input_tokens: 0, output_tokens: 0, n: 0, name: "" };
  let total = null;
  for (const u of S.usage || []) {
    if (u.who === "_total") { total = u; continue; }
    if (u.removed) {
      gone.n++; gone.name = u.name || "";
      for (const k of ["calls", "micro_usd", "input_tokens", "output_tokens"]) gone[k] += u[k] || 0;
      continue;
    }
    rows.push({ ...u, label: u.who === "host" ? t("usageHost") : (u.name || u.who) });
  }
  rows.sort((a, b) => b.calls - a.calls);
  if (gone.n) rows.push({ ...gone, label: gone.n === 1 && gone.name ? t("usageGoneOne", gone.name) : t("usageGone", gone.n), muted: true });
  const line = (u, into, cls) => {
    const tr = el("tr", undefined, cls);
    cell(tr, u.label, u.muted ? "muted" : "");
    cell(tr, num(u.calls), "num");
    cell(tr, money(u.micro_usd), "num");
    cell(tr, num(u.input_tokens) + " / " + num(u.output_tokens), "num narrow-hide");
    into.append(tr);
  };
  rows.forEach((u) => line(u, tb));
  if (total && rows.length) line({ ...total, label: t("usageTotal") }, foot, "total");
  $("usage-empty").hidden = rows.length > 0;
}

document.querySelectorAll(".tabs button").forEach((b) => {
  b.addEventListener("click", () => {
    document.querySelectorAll(".tabs button").forEach((x) => x.setAttribute("aria-selected", String(x === b)));
    document.querySelectorAll(".tab").forEach((x) => { x.hidden = x.id !== "tab-" + b.dataset.tab; });
    store("omnillm.tab", b.dataset.tab);
  });
});

$("add-device").addEventListener("click", () => startPair($("pair-dash"), () => { closePair($("pair-dash")); refresh(); }, true));
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

document.querySelectorAll("button[data-copy]").forEach((b) => copyBtn(b.dataset.copy, b));

async function boot() {
  applyLang();
  applyTheme(store("omnillm.theme"));
  const tab = store("omnillm.tab") || "devices";
  const tb = document.querySelector('.tabs button[data-tab="' + tab + '"]') || document.querySelector(".tabs button");
  tb.click();
  const m = /(?:^|[#&])login=([0-9a-f]{64})/.exec(location.hash);
  // "Add a phone or computer…" in the menu bar opens the console with &add.
  const wantAdd = /[#&]add(?:&|$)/.test(location.hash);
  if (m) {
    history.replaceState(null, "", "/console/");
    try { const r = await api("login", { token: m[1] }); sessionSecret(r.session); } catch (e) { show("login-needed"); return; }
  }
  try { await refresh(); } catch (e) { if (e.status === 401) show("login-needed"); else toast(e.message, true); return; }
  if (wantAdd && !$("dash").hidden) {
    document.querySelector('.tabs button[data-tab="devices"]').click();
    $("add-device").click();
  }
}
boot();
