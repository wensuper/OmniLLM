// OmniLLM console. Everything the server returns is shown with textContent;
// nothing from the server or from a device is ever inserted as HTML.
"use strict";

// Words for people, the same in every OmniLLM screen: the hub is "your Mac"
// (here: this Mac), the relay is "remote access", a provider is a "model
// source". Text in `backticks` is shown in code style (see rich()).
const T = {
  zh: {
    loading: "正在载入……",
    loginTitle: "需要重新打开控制台",
    loginBody: "点屏幕顶部菜单栏里的 OmniLLM 图标，选“打开控制台”。",
    loginOr: "或者在终端运行：",
    loginSafe: "为了安全，控制台只能从这台 Mac 上打开。",
    hubDown: "OmniLLM 没有运行。点菜单栏里的 OmniLLM 图标，选“启动 OmniLLM”，或在终端运行 omnillm install。",
    ariaLang: "切换语言", ariaTheme: "切换深色或浅色", ariaSteps: "步骤", back: "返回",
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
    name: "名字", lastSeen: "最近使用", budget: "每天额度", todayBudget: "今天 / 每天额度", where: "对话发往", humanOnly: "只能手动对话",
    showOnDevices: "在设备上显示", subAlways: "订阅模型始终只能手动对话", rename: "改名",
    enabled: "启用", pausedChip: "已暂停", pausedToast: "已暂停“{0}”", resumedToast: "已开启“{0}”",
    online: "在线", todayAt: "今天 {0}", yesterdayAt: "昨天 {0}",
    editDevice: "编辑设备 · {0}", editApp: "编辑应用 · {0}", secBasics: "基本", secStatus: "状态", secModels: "可用模型",
    infoAdded: "添加于 {0}", infoSeen: "最近使用 {0}", infoSpent: "今天已用 {0}",
    pauseHint: "关掉就是暂停：这台设备马上不能用，配对和设置都保留。", pauseHintApp: "关掉就是暂停：这个应用马上不能用，令牌和设置都保留。",
    limitHint: "只受整台 Mac 每天总额度（{0}）限制。", allModels: "全部模型（以后接入的新模型也自动可用）",
    grpSub: "订阅", grpKey: "API Key", grpLocal: "这台 Mac", pickOne: "至少选一个模型。", appNoSub: "订阅模型只能手动对话，应用用不了。",
    removeDevice: "移除设备", removeApp: "移除应用",
    providers: "模型来源", keyLast4: "接入方式", subLogin: "订阅登录（只留在这台 Mac 上）",
    problemLogin: "登录已失效：在终端运行 claude 并 /login", problemLoginCodex: "登录已失效：在终端运行 codex login",
    problemLimit: "用量到上限，约 {0}后恢复", problemLimitNow: "用量到上限，稍后恢复", hours: "{0} 小时", hour: "1 小时", minutes: "{0} 分钟", minute: "1 分钟",
    noDevices: "还没有设备。点“添加设备”。",
    appsBody: "给不能直接嵌入 OmniLLM 的程序（脚本、其他 App）用。应用只能做后台调用，只能用 API Key 和这台 Mac 上的模型。",
    usageBody: "{0}（按这台 Mac 的时间）。花费按模型来源公布的价格估算，没有公布价格的按偏高的价格估；订阅和这台 Mac 上的模型不花钱。",
    rangeToday: "今天", rangeYesterday: "昨天", rangeWeek: "最近 7 天",
    headToday: "今天的用量", headYesterday: "昨天的用量", headWeek: "最近 7 天的用量",
    totalToday: "今天合计", totalYesterday: "昨天合计", totalWeek: "最近 7 天合计",
    emptyToday: "今天还没有用过。", emptyYesterday: "昨天没有用过。", emptyWeek: "最近 7 天没有用过。",
    byModel: "按模型", modelsGone: "已移除的模型（{0} 个）",
    who: "设备或应用", spend: "花费（估算）", calls: "次数", tokens: "文字量（发出 / 收到）", tokensTip: "按 token 计",
    usageHost: "这台 Mac", usageGone: "已移除的设备和应用（{0} 个）", usageGoneOne: "{0}（已移除）",
    capLine: "整台 Mac 每天总额度 {0}", capLabel: "整台 Mac 每天总额度", capNone: "整台 Mac 每天总额度：不设上限", capEdit: "修改", noApps: "还没有应用。",
    modelsBody: "“只能手动对话”的模型只给在手机、电脑上聊天的人用，脚本和应用调用不了。订阅模型一直是这样，免得账号被当成自动化调用。",
    claudeSub: "Claude 订阅", codexName: "ChatGPT（Codex）",
    remove: "移除", removeConfirm: "确定移除“{0}”？它会立即无法使用。",
    removeDeviceConfirm: "移除“{0}”？它会立刻用不了，那台设备上的对话也会在它下次打开时清除。",
    removed: "已移除", connected: "已接入，{0} 个模型可用", saved: "已保存",
    local: "只在这台 Mac", remote: "云端", localChip: "这台 Mac", sub: "订阅",
    todayOf: "今天 {0} / {1}", todayNoLimit: "今天 {0} · 不设上限",
    pairPhone: "手机或平板：用相机扫这个二维码，不用装任何东西。{0} 前有效，只能用一次。",
    pairComputer: "电脑或服务器（没有摄像头）",
    pairComputerNew: "还没装 OmniLLM 的，在那台电脑上运行这一行（只用来聊天，不会一直在后台运行），装完会提示粘贴配对文字：",
    pairComputerHow: "已经装好的，运行 `omnillm connect pair`，粘贴下面这段配对文字。不要发给别人，也不要贴到聊天里。",
    pairWait: "等待设备连接……",
    pairConfirmQ: "设备“{0}”请求加入。它屏幕上显示的数字是这个吗？",
    pairYes: "一致，添加", pairNo: "不一致，拒绝", pairLeft: "手机那边取消了，或者连接断了。",
    pairDone: "已添加“{0}”", pairFailed: "配对没有完成：{0}",
    pairDenied: "已拒绝。", pairExpired: "这个二维码已过期。", pairNew: "换一个二维码",
    pairStartOffline: "远程连接现在没有连上，生成不了二维码。", retry: "重试",
    close: "关闭", edit: "编辑", save: "保存", cancel: "取消", noLimit: "不设上限",
    nameLabel: "设备名字", budgetLabel: "每天额度（美元）",
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
    loginSafe: "For safety, the console opens only from this Mac.",
    hubDown: "OmniLLM isn't running. Click the OmniLLM icon in the menu bar and choose Start OmniLLM. Or run omnillm install in Terminal.",
    ariaLang: "Language", ariaTheme: "Light or dark", ariaSteps: "Steps", back: "Back",
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
    name: "Name", lastSeen: "Last used", budget: "Daily limit", todayBudget: "Today / daily limit", where: "Messages go to", humanOnly: "Chat only",
    showOnDevices: "Show on devices", subAlways: "Subscription models are always chat-only", rename: "Rename",
    enabled: "On", pausedChip: "Paused", pausedToast: "Paused “{0}”", resumedToast: "Turned on “{0}”",
    online: "Online", todayAt: "Today {0}", yesterdayAt: "Yesterday {0}",
    editDevice: "Edit device · {0}", editApp: "Edit app · {0}", secBasics: "Basics", secStatus: "Status", secModels: "Models it can use",
    infoAdded: "Added {0}", infoSeen: "Last used {0}", infoSpent: "Spent today {0}",
    pauseHint: "Off pauses it: the device stops working at once; its pairing and settings stay.", pauseHintApp: "Off pauses it: the app stops working at once; its token and settings stay.",
    limitHint: "Only the daily total for this Mac ({0}) applies.", allModels: "All models (including ones added later)",
    grpSub: "Subscriptions", grpKey: "API key", grpLocal: "This Mac", pickOne: "Choose at least one model.", appNoSub: "Subscription models are chat-only; apps can't use them.",
    removeDevice: "Remove device", removeApp: "Remove app",
    providers: "Model sources", keyLast4: "Connected with", subLogin: "Subscription login (stays on this Mac)",
    problemLogin: "Signed out: run claude and /login in Terminal", problemLoginCodex: "Signed out: run codex login in Terminal",
    problemLimit: "Usage limit reached; back in about {0}", problemLimitNow: "Usage limit reached; back soon", hours: "{0} hours", hour: "1 hour", minutes: "{0} minutes", minute: "1 minute",
    noDevices: "No devices yet. Choose “Add device”.",
    appsBody: "For programs that cannot embed OmniLLM (scripts, other apps). Apps make background calls only, with API keys and models on this Mac.",
    usageBody: "{0} (this Mac's time). Spend is estimated from the model sources' published prices, high where there is no price; subscriptions and models on this Mac cost nothing.",
    rangeToday: "Today", rangeYesterday: "Yesterday", rangeWeek: "Last 7 days",
    headToday: "Today's use", headYesterday: "Yesterday's use", headWeek: "Use over the last 7 days",
    totalToday: "Total today", totalYesterday: "Total yesterday", totalWeek: "Total, last 7 days",
    emptyToday: "Nothing used yet today.", emptyYesterday: "Nothing was used yesterday.", emptyWeek: "Nothing used in the last 7 days.",
    byModel: "By model", modelsGone: "Removed models ({0})",
    who: "Device or app", spend: "Spend (est.)", calls: "Calls", tokens: "Text (sent / received)", tokensTip: "tokens",
    usageHost: "This Mac", usageGone: "Removed devices and apps ({0})", usageGoneOne: "{0} (removed)",
    capLine: "Daily total for this Mac: {0}", capLabel: "Daily total for this Mac", capNone: "Daily total for this Mac: no limit", capEdit: "Change", noApps: "No apps yet.",
    modelsBody: "Chat-only models are for people chatting on a phone or computer; scripts and apps can't use them. Subscription models are always chat-only, so the account isn't flagged as automated.",
    claudeSub: "Claude subscription", codexName: "ChatGPT (Codex)",
    remove: "Remove", removeConfirm: "Remove “{0}”? It stops working at once.",
    removeDeviceConfirm: "Remove “{0}”? It stops working at once, and its chats are cleared next time it opens.",
    removed: "Removed", connected: "Connected: {0} models available", saved: "Saved",
    local: "This Mac only", remote: "Cloud", localChip: "This Mac", sub: "subscription",
    todayOf: "Today {0} / {1}", todayNoLimit: "Today {0} · no limit",
    pairPhone: "Phone or tablet: scan this QR code with the camera; nothing to install. Valid until {0}, single use.",
    pairComputer: "Computer or server (no camera)",
    pairComputerNew: "Not installed there yet? Run this line on that computer (just for chatting; nothing keeps running in the background); it then asks for the pairing text:",
    pairComputerHow: "Already installed: run `omnillm connect pair` and paste this pairing text. Don't send it to anyone or paste it into a chat.",
    pairWait: "Waiting for the device…",
    pairConfirmQ: "“{0}” wants to join. Does its screen show this number?",
    pairYes: "Matches, add", pairNo: "Doesn't match, refuse", pairLeft: "The device cancelled or lost its connection.",
    pairDone: "Added “{0}”", pairFailed: "Pairing did not finish: {0}",
    pairDenied: "Refused.", pairExpired: "This QR code has expired.", pairNew: "New QR code",
    pairStartOffline: "Remote access isn't connected right now, so no QR code can be made.", retry: "Retry",
    close: "Close", edit: "Edit", save: "Save", cancel: "Cancel", noLimit: "No limit",
    nameLabel: "Device name", budgetLabel: "Daily limit (USD)",
    noRelay: "Remote access isn't set up yet, so other devices can't connect. Skip for now; add devices under Devices once it is set up.",
    relayWait: "Turning on remote access…", relayDown: "Remote access can't connect right now; check this Mac's network. This continues by itself once it connects.",
    relayOn: "Devices can connect", relayOff: "Remote access is down; devices can't connect now", relayNone: "This Mac only (no remote access)",
    regionPaused: "This Mac's region changed; calls are paused.", regionConfirm: "Confirm and resume",
    appToken: "Token for “{0}” (shown once — copy it now):",
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
  document.querySelectorAll("[data-aria]").forEach((n) => { n.setAttribute("aria-label", t(n.dataset.aria)); });
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

let S = null;

// status asks for everything on the dashboard, with the Usage tab's range.
const statusPath = () => "status?range=" + usageRange;

async function refresh() {
  S = await api(statusPath());
  for (const k of ["providers", "models", "devices", "apps", "usage", "range_usage", "model_usage"]) S[k] = S[k] || [];
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
  return !!(curPair || pairTimer || document.querySelector("tr.editing, .cap-edit, .pair-panel, .dialog-scrim"));
}
setInterval(async () => {
  if (!S || signedOut || document.hidden || $("dash").hidden || busy()) return;
  let next;
  try { next = await api(statusPath()); } catch (_) { return; }
  if ($("dash").hidden || busy()) return;
  for (const k of ["providers", "models", "devices", "apps", "usage", "range_usage", "model_usage"]) next[k] = next[k] || [];
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

// fromDash: "Connect a model" was opened from the dashboard, so the
// wizard offers a way back to it.
let fromDash = false;

async function showWizard() {
  show("wizard");
  $("wiz-back-line").hidden = !fromDash;
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
    const o = el("option", modelName(m) + (m.locality === "local" ? " · " + t("local") : isSub(m) ? " · " + t("sub") : ""));
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
      // The owner may name the device while confirming.
      const nid = "pair-name-" + p.id.slice(0, 8);
      const nl = el("label", t("name")); nl.htmlFor = nid;
      const nameIn = el("input"); nameIn.id = nid; nameIn.maxLength = 64; nameIn.value = st.device_name || "";
      c.append(nl, nameIn);
      const actions = el("div", undefined, "actions");
      const yes = el("button", t("pairYes"), "primary");
      const no = el("button", t("pairNo"), "secondary");
      yes.type = no.type = "button";
      const decide = async (v) => { yes.disabled = no.disabled = true; try { await api("pair/decide", { id: p.id, yes: v, name: v ? nameIn.value.trim() : "" }); } catch (e) { toast(e.message, true); } };
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
      else if (st.reason === "left") ended(t("pairLeft"));
      else if (st.reason === "expired") ended(t("pairExpired"));
      else ended(t("pairFailed", st.message || ""), true);
    }
  }, 1000);
}

// ---------- dashboard ----------

function showDash() {
  show("dash");
  fromDash = false;
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
function isSub(m) { const k = kindOf(m.provider); return k === "claude-code" || k === "codex"; }
// kindOf: what kind of source a provider name is (its kind, as connected).
function kindOf(name) { const p = S.providers.find((x) => x.name === name); return p ? p.kind : name; }
function cell(row, text, cls) { const td = el("td", text, cls); row.append(td); return td; }

function removeButton(label, path, id, q) {
  const b = el("button", t("remove"), "danger");
  b.type = "button";
  b.addEventListener("click", async () => {
    if (!window.confirm(t(q || "removeConfirm", label))) return;
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

// timeWords: when something last happened, as people say it: today or
// yesterday with the time, else the date.
function timeWords(iso) {
  if (!iso || iso.startsWith("0001")) return t("never");
  const d = new Date(iso), now = new Date();
  const loc = lang === "zh" ? "zh-CN" : "en-US";
  const time = d.toLocaleTimeString(loc, lang === "zh" ? { hour: "2-digit", minute: "2-digit", hour12: false } : { hour: "numeric", minute: "2-digit" });
  const day = (x) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
  const diff = Math.round((day(now) - day(d)) / 86400000);
  if (diff === 0) return t("todayAt", time);
  if (diff === 1) return t("yesterdayAt", time);
  return dateWords(iso);
}
// dateWords: a date alone: "10月18日" / "Oct 18" (with the year when it
// is not this year's).
function dateWords(iso) {
  if (!iso || iso.startsWith("0001")) return t("never");
  const d = new Date(iso), other = d.getFullYear() !== new Date().getFullYear();
  if (lang === "zh") return (other ? d.getFullYear() + "年" : "") + (d.getMonth() + 1) + "月" + d.getDate() + "日";
  return d.toLocaleDateString("en-US", other ? { year: "numeric", month: "short", day: "numeric" } : { month: "short", day: "numeric" });
}
// seenText: "Online" while the device has a session open, else when it was
// last used.
function seenText(td, online, iso) {
  if (online) td.append(el("span", "●", "dot-ok"), document.createTextNode(" " + t("online")));
  else td.textContent = timeWords(iso);
}

// nameCell: a device's or app's name, with a "Paused" chip when it is.
function nameCell(tr, name, paused) {
  const td = cell(tr, name, "nm");
  if (paused) td.append(" ", el("span", t("pausedChip"), "tag paused-tag"));
  td.title = name;
  return td;
}

// pauseSwitch: on = in use; turning it off pauses that device or app at
// once (its next request is refused), and on again resumes it.
function pauseSwitch(tr, label, path, id, paused) {
  const td = cell(tr, "", "sw");
  const sw = el("input"); sw.type = "checkbox"; sw.className = "switch"; sw.checked = !paused;
  sw.setAttribute("aria-label", t("enabled") + " " + label);
  sw.addEventListener("change", async () => {
    sw.disabled = true;
    try {
      await api(path, { id, paused: !sw.checked });
      toast(t(sw.checked ? "resumedToast" : "pausedToast", label));
      await refresh();
    } catch (e) { sw.checked = !sw.checked; sw.disabled = false; toast(e.message, true); }
  });
  td.append(sw);
}

function renderDevices() {
  const tb = $("devices");
  tb.textContent = "";
  for (const d of S.devices) {
    const tr = el("tr", undefined, d.Paused ? "paused" : "");
    nameCell(tr, d.Name, d.Paused);
    seenText(cell(tr, "", "seen"), d.Online, d.LastSeen);
    cell(tr, spentText("device/" + d.ID, d.BudgetMicroUSD), "spent");
    pauseSwitch(tr, d.Name, "devices/update", d.ID, d.Paused);
    const ed = el("button", t("edit"), "secondary small-btn");
    ed.type = "button";
    ed.addEventListener("click", () => editDialog("device", d));
    cell(tr, "", "actions").append(ed);
    tb.append(tr);
  }
  $("devices-empty").hidden = S.devices.length > 0;
}

// moneyInput: a dollar amount without the browser's number spinners.
function moneyInput(value, label) {
  const wrap = el("span", undefined, "money-in");
  const amt = el("input"); amt.type = "text"; amt.inputMode = "decimal"; amt.autocomplete = "off"; amt.spellcheck = false;
  amt.value = value; amt.setAttribute("aria-label", label);
  wrap.append(el("span", "$"), amt);
  return [wrap, amt];
}
// dollars reads an amount; anything unreadable is sent as 0, which the
// hub refuses with its own sentence.
function dollars(v) { const n = Number(String(v).trim().replace(",", ".")); return Number.isFinite(n) ? n : 0; }

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
    const [wrap, amt] = moneyInput(v === NO_LIMIT ? "20" : String(v / 1e6), t("capLabel"));
    amt.disabled = v === NO_LIMIT;
    const sw = el("input"); sw.type = "checkbox"; sw.className = "switch"; sw.checked = v === NO_LIMIT;
    const lab = el("label", undefined, "toggle"); lab.append(sw, el("span", t("noLimit")));
    sw.addEventListener("change", () => { amt.disabled = sw.checked; });
    const save = el("button", t("save"), "primary small-btn"); save.type = "button";
    const no = el("button", t("cancel"), "secondary small-btn"); no.type = "button";
    const submit = async () => {
      save.disabled = true;
      try { await api("daily-total", sw.checked ? { no_limit: true } : { budget_usd: dollars(amt.value) }); toast(t("saved")); await refresh(); } catch (e) { toast(e.message, true); save.disabled = false; }
    };
    save.addEventListener("click", submit);
    no.addEventListener("click", renderCap);
    f.addEventListener("keydown", (e) => { if (e.key === "Enter") submit(); if (e.key === "Escape") renderCap(); });
    const fields = el("div", undefined, "cap-fields");
    fields.append(el("span", t("capLabel"), "cap-label"), wrap, lab);
    const btns = el("div", undefined, "btns");
    btns.append(save, no);
    f.append(fields, btns);
    box.append(f);
    amt.focus();
  });
  box.append(b);
}

// ---------- edit dialog for a device or an app ----------

// editDialog: everything about one device or app in one place: its name,
// on or paused, its daily limit, the models it may use, and removing it.
// Esc, the x or the backdrop close it; Save sends one update.
function editDialog(kind, x) {
  const isApp = kind === "app";
  const path = isApp ? "apps/update" : "devices/update";
  const who = (isApp ? "app/" : "device/") + x.ID;
  const opener = document.activeElement;
  const scrim = el("div", undefined, "dialog-scrim");
  const dlg = el("div", undefined, "dialog");
  dlg.setAttribute("role", "dialog"); dlg.setAttribute("aria-modal", "true");
  const close = () => { scrim.remove(); document.removeEventListener("keydown", onKey, true); if (opener && opener.isConnected) opener.focus(); };
  // Head.
  const head = el("div", undefined, "dialog-head");
  const title = el("h2", t(isApp ? "editApp" : "editDevice", x.Name)); title.id = "dlg-title";
  dlg.setAttribute("aria-labelledby", title.id);
  const xb = el("button", "", "close-x"); xb.type = "button"; xb.setAttribute("aria-label", t("close")); xb.title = t("close");
  xb.addEventListener("click", close);
  head.append(title, xb);
  const body = el("div", undefined, "dialog-body");
  const section = (label) => { const sec = el("section", undefined, "dlg-sec"); sec.append(el("h3", label)); body.append(sec); return sec; };
  // Basics.
  const basics = section(t("secBasics"));
  const nid = "dlg-name";
  const nl = el("label", t("name")); nl.htmlFor = nid;
  const name = el("input"); name.id = nid; name.maxLength = 64; name.value = x.Name;
  const u = S.usage.find((r) => r.who === who);
  const info = [t("infoAdded", dateWords(x.Created))];
  if (!isApp) info.push(t("infoSeen", x.Online ? t("online") : timeWords(x.LastSeen)));
  info.push(t("infoSpent", money(u ? u.micro_usd : 0)));
  basics.append(nl, name, el("p", info.join(" · "), "muted small dlg-info"));
  // Status: on, or paused.
  const status = section(t("secStatus"));
  const on = el("input"); on.type = "checkbox"; on.className = "switch"; on.checked = !x.Paused;
  const onl = el("label", undefined, "toggle"); onl.append(on, el("span", t("enabled")));
  status.append(onl, el("p", t(isApp ? "pauseHintApp" : "pauseHint"), "muted small"));
  // Daily limit.
  const lim = section(t("budget"));
  const unl = x.BudgetMicroUSD === NO_LIMIT;
  const [wrap, amt] = moneyInput(unl ? "5" : String(x.BudgetMicroUSD / 1e6), t("budgetLabel"));
  amt.disabled = unl;
  const nolim = el("input"); nolim.type = "checkbox"; nolim.className = "switch"; nolim.checked = unl;
  const nll = el("label", undefined, "toggle"); nll.append(nolim, el("span", t("noLimit")));
  const line = el("div", undefined, "budget-line"); line.append(wrap, nll);
  const lhint = el("p", t("limitHint", S.daily_total_micro_usd === NO_LIMIT ? t("noLimit") : money(S.daily_total_micro_usd)), "muted small");
  lim.append(line, lhint);
  // Models it can use: all (also later ones), or the ones switched on.
  const msec = section(t("secModels"));
  const allSw = el("input"); allSw.type = "checkbox"; allSw.className = "switch"; allSw.checked = !(x.Models || []).length;
  const all = el("label", undefined, "toggle wrap"); all.append(allSw, el("span", t("allModels")));
  const list = el("div", undefined, "model-pick");
  const usable = (m) => !(isApp && (m.human_only || m.group === "sub"));
  const picks = [];
  for (const [g, label] of [["sub", t("grpSub")], ["key", t("grpKey")], ["local", t("grpLocal")]]) {
    const ms = S.models.filter((m) => (m.group || "key") === g);
    if (!ms.length) continue;
    const grp = el("div", undefined, "pick-group");
    grp.append(el("div", label, "pick-label"));
    if (isApp && g === "sub") grp.append(el("p", t("appNoSub"), "muted small"));
    for (const m of ms) {
      const row = el("label", undefined, "pick-row");
      const nm = el("span", modelName(m), "pick-name");
      if (m.human_only || g === "sub") nm.append(" ", el("span", t("humanOnly"), "tag"));
      const sw = el("input"); sw.type = "checkbox"; sw.className = "switch";
      sw.disabled = !usable(m);
      sw.checked = usable(m) && (x.Models || []).includes(m.id);
      row.append(nm, sw);
      grp.append(row);
      picks.push([m.id, sw]);
    }
    list.append(grp);
  }
  const pickHint = el("p", t("pickOne"), "err-text small");
  msec.append(all, list, pickHint);
  // Foot: remove on the left, cancel and save on the right.
  const foot = el("div", undefined, "dialog-foot");
  const rm = el("button", t(isApp ? "removeApp" : "removeDevice"), "danger-text"); rm.type = "button";
  rm.addEventListener("click", async () => {
    if (!window.confirm(t(isApp ? "removeConfirm" : "removeDeviceConfirm", x.Name))) return;
    try { await api(isApp ? "apps/remove" : "devices/remove", { id: x.ID }); close(); toast(t("removed")); refresh(); } catch (e) { toast(e.message, true); }
  });
  const no = el("button", t("cancel"), "secondary"); no.type = "button"; no.addEventListener("click", close);
  const save = el("button", t("save"), "primary"); save.type = "button";
  const right = el("div", undefined, "btns"); right.append(no, save);
  foot.append(rm, right);
  // What the form says now, and whether that differs from what is saved.
  const chosen = () => picks.filter(([, sw]) => sw.checked && !sw.disabled).map(([id]) => id);
  const state = () => JSON.stringify([name.value.trim(), on.checked, nolim.checked, nolim.checked ? 0 : dollars(amt.value), allSw.checked ? [] : chosen()]);
  const start = state();
  const sync = () => {
    amt.disabled = nolim.checked;
    lhint.hidden = !nolim.checked;
    list.hidden = allSw.checked;
    const none = !allSw.checked && chosen().length === 0;
    pickHint.hidden = !none;
    save.disabled = none || state() === start;
  };
  allSw.addEventListener("change", () => {
    // Turning "all" off starts from everything it may use.
    if (!allSw.checked && chosen().length === 0) picks.forEach(([, sw]) => { if (!sw.disabled) sw.checked = true; });
    sync();
  });
  dlg.addEventListener("input", sync);
  dlg.addEventListener("change", sync);
  save.addEventListener("click", async () => {
    const b = { id: x.ID, name: name.value, paused: !on.checked, models: allSw.checked ? [] : chosen() };
    if (nolim.checked) b.no_limit = true; else b.budget_usd = dollars(amt.value);
    save.disabled = true;
    try { await api(path, b); close(); toast(t("saved")); await refresh(); } catch (e) { toast(e.message, true); sync(); }
  });
  // Keys: Esc closes; Tab stays inside the dialog.
  const onKey = (e) => {
    if (e.key === "Escape") { e.preventDefault(); close(); return; }
    if (e.key !== "Tab") return;
    const f = [...dlg.querySelectorAll("button, input, select, textarea, [tabindex]")].filter((n) => !n.disabled && n.offsetParent !== null);
    if (!f.length) return;
    const first = f[0], last = f[f.length - 1];
    if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus(); }
    else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus(); }
    else if (!dlg.contains(document.activeElement)) { e.preventDefault(); first.focus(); }
  };
  document.addEventListener("keydown", onKey, true);
  scrim.addEventListener("mousedown", (e) => { if (e.target === scrim) close(); });
  dlg.append(head, body, foot);
  scrim.append(dlg);
  document.body.append(scrim);
  sync();
  name.focus(); name.select();
}

// modelName: Codex without a known model is "ChatGPT (Codex)" in every row.
function modelName(m) {
  if (kindOf(m.provider) === "codex" && /^(codex|chatgpt)\b/i.test(m.display_name) && !/\d/.test(m.display_name)) return t("codexName");
  return m.display_name;
}
// vendorChip: where a model's messages go: this Mac, or the company that
// runs it in the cloud.
function vendorChip(m) {
  if (m.locality === "local") return el("span", t("localChip"), "tag local");
  const k = kindOf(m.provider);
  const v = k === "claude-code" || k === "anthropic" ? "Anthropic" : k === "codex" || k === "openai" ? "OpenAI" : t("remote");
  return el("span", v, "tag");
}

// aboutText: a wait in words, rounded the way people say it.
function aboutText(sec) {
  const n = sec >= 3600 ? Math.round(sec / 3600) : Math.max(1, Math.round(sec / 60));
  if (sec >= 3600) return n === 1 ? t("hour") : t("hours", n);
  return n === 1 ? t("minute") : t("minutes", n);
}
// problemText: what is wrong with a model source right now, if anything:
// signed out, or at its usage limit until resets_at (unix seconds).
function problemText(p) {
  if (p.problem === "login") return p.kind === "codex" ? t("problemLoginCodex") : t("problemLogin");
  if (p.problem === "limit") {
    const sec = p.resets_at ? p.resets_at - Date.now() / 1000 : 0;
    return sec > 0 ? t("problemLimit", aboutText(sec)) : t("problemLimitNow");
  }
  return "";
}

function switchCell(tr, checked, label, onChange) {
  const td = cell(tr, "", "sw");
  const box = el("input"); box.type = "checkbox"; box.className = "switch"; box.checked = checked;
  box.setAttribute("aria-label", label);
  box.addEventListener("change", () => onChange(box));
  td.append(box);
  return box;
}

function renderModels() {
  const tb = $("models");
  tb.textContent = "";
  for (const m of S.models) {
    const tr = el("tr");
    const name = modelName(m);
    const nc = cell(tr, name + (isSub(m) ? " · " + t("sub") : ""), "nm");
    // On a phone the chip sits under the name (its own column is hidden).
    nc.append(el("br", undefined, "narrow-only"), vendorChip(m)); nc.lastChild.classList.add("narrow-only");
    cell(tr, "").append(vendorChip(m));
    const ho = switchCell(tr, !!m.human_only || isSub(m), t("humanOnly") + " " + name, async (box) => {
      try { await api("models/set", { id: m.id, human_only: box.checked }); toast(t("saved")); } catch (e) { box.checked = !box.checked; toast(e.message, true); }
    });
    if (isSub(m)) { ho.disabled = true; ho.title = t("subAlways"); ho.parentNode.title = t("subAlways"); }
    switchCell(tr, !m.hidden, t("showOnDevices") + " " + name, async (box) => {
      try { await api("models/set", { id: m.id, shown: box.checked }); toast(t("saved")); m.hidden = !box.checked; } catch (e) { box.checked = !box.checked; toast(e.message, true); }
    });
    const act = cell(tr, "", "actions");
    const rn = el("button", t("rename"), "secondary small-btn"); rn.type = "button";
    rn.addEventListener("click", () => renameModel(tr, nc, act, m, name));
    act.append(rn);
    tb.append(tr);
  }
  const pb = $("providers");
  pb.textContent = "";
  for (const p of S.providers) {
    const tr = el("tr");
    const label = p.kind === "claude-code" ? t("claudeSub") : p.kind === "codex" ? t("codexName") : (p.display_name || p.name);
    const nc = cell(tr, label + (p.local ? " · " + t("local") : ""));
    const pt = problemText(p);
    if (pt) { nc.append(document.createElement("br"), el("span", pt, "tag warn-tag")); }
    cell(tr, p.subscription ? t("subLogin") : p.key_last4 ? "API Key …" + p.key_last4 : t("local"));
    cell(tr, "", "actions").append(removeButton(label, "providers/remove", p.name));
    pb.append(tr);
  }
}

// renameModel: the name cell becomes a field; an empty name goes back to
// the one the model source gives.
function renameModel(tr, nc, act, m, cur) {
  tr.classList.add("editing");
  nc.textContent = "";
  const inp = el("input"); inp.value = cur; inp.maxLength = 64; inp.setAttribute("aria-label", t("rename"));
  nc.append(inp);
  act.textContent = "";
  const save = el("button", t("save"), "primary small-btn"); save.type = "button";
  const no = el("button", t("cancel"), "secondary small-btn"); no.type = "button";
  act.append(save, no);
  const submit = async () => {
    save.disabled = true;
    try { await api("models/set", { id: m.id, name: inp.value.trim() }); toast(t("saved")); await refresh(); } catch (e) { toast(e.message, true); save.disabled = false; }
  };
  save.addEventListener("click", submit);
  no.addEventListener("click", renderModels);
  inp.addEventListener("keydown", (e) => { if (e.key === "Enter") submit(); if (e.key === "Escape") renderModels(); });
  inp.focus(); inp.select();
}

function renderApps() {
  const tb = $("apps");
  tb.textContent = "";
  for (const a of S.apps) {
    const tr = el("tr", undefined, a.Paused ? "paused" : "");
    nameCell(tr, a.Name, a.Paused);
    cell(tr, spentText("app/" + a.ID, a.BudgetMicroUSD), "spent");
    pauseSwitch(tr, a.Name, "apps/update", a.ID, a.Paused);
    const ed = el("button", t("edit"), "secondary small-btn");
    ed.type = "button";
    ed.addEventListener("click", () => editDialog("app", a));
    cell(tr, "", "actions").append(ed);
    tb.append(tr);
  }
  $("apps-empty").hidden = S.apps.length > 0;
}

// The Usage tab's range, kept for next time: today, yesterday or week.
let usageRange = ["today", "yesterday", "week"].includes(store("omnillm.range")) ? store("omnillm.range") : "today";
// rangeWords: the range's heading, total line and empty line.
const rangeWords = () => usageRange === "yesterday" ? [t("headYesterday"), t("totalYesterday"), t("emptyYesterday")]
  : usageRange === "week" ? [t("headWeek"), t("totalWeek"), t("emptyWeek")] : [t("headToday"), t("totalToday"), t("emptyToday")];
document.querySelectorAll("#usage-range button").forEach((b) => {
  b.addEventListener("click", async () => {
    if (b.dataset.range === usageRange) return;
    usageRange = b.dataset.range;
    store("omnillm.range", usageRange);
    renderUsage();
    try { await refresh(); } catch (e) { toast(e.message, true); }
  });
});

// renderUsage: one line per device or app by its name (never an internal
// id), busiest first; removed devices folded into one line; the range's
// total at the bottom. Below it the same per model.
const num = (n) => Number(n || 0).toLocaleString(lang === "zh" ? "zh-CN" : "en-US");
function renderUsage() {
  document.querySelectorAll("#usage-range button").forEach((b) => b.setAttribute("aria-pressed", String(b.dataset.range === usageRange)));
  const [head, totalLabel, emptyText] = rangeWords();
  $("usage-body").textContent = t("usageBody", head);
  const tb = $("usage"), foot = $("usage-total");
  tb.textContent = ""; foot.textContent = "";
  const paused = new Set([...S.devices.filter((d) => d.Paused).map((d) => "device/" + d.ID), ...S.apps.filter((a) => a.Paused).map((a) => "app/" + a.ID)]);
  const rows = [], gone = { calls: 0, micro_usd: 0, input_tokens: 0, output_tokens: 0, n: 0, name: "" };
  let total = null;
  for (const u of S.range_usage || []) {
    if (u.who === "_total") { total = u; continue; }
    if (u.removed) {
      gone.n++; gone.name = u.name || "";
      for (const k of ["calls", "micro_usd", "input_tokens", "output_tokens"]) gone[k] += u[k] || 0;
      continue;
    }
    rows.push({ ...u, label: u.who === "host" ? t("usageHost") : (u.name || u.who), paused: paused.has(u.who) });
  }
  rows.sort((a, b) => b.calls - a.calls);
  if (gone.n) rows.push({ ...gone, label: gone.n === 1 && gone.name ? t("usageGoneOne", gone.name) : t("usageGone", gone.n), muted: true });
  const line = (u, into, cls) => {
    const tr = el("tr", undefined, cls);
    const c = cell(tr, u.label, u.muted ? "muted nm" : "nm");
    if (u.paused) c.append(" ", el("span", t("pausedChip"), "tag paused-tag"));
    cell(tr, num(u.calls), "num");
    cell(tr, money(u.micro_usd), "num");
    cell(tr, num(u.input_tokens) + " / " + num(u.output_tokens), "num narrow-hide");
    into.append(tr);
  };
  rows.forEach((u) => line(u, tb));
  if (total && rows.length) line({ ...total, label: totalLabel }, foot, "total");
  $("usage-empty").textContent = emptyText;
  $("usage-empty").hidden = rows.length > 0;
  // By model: the same range, one line per model by its name.
  const mt = $("usage-models");
  mt.textContent = "";
  const mrows = [], mgone = { calls: 0, micro_usd: 0, input_tokens: 0, output_tokens: 0, n: 0 };
  for (const u of S.model_usage || []) {
    if (u.removed) {
      mgone.n++;
      for (const k of ["calls", "micro_usd", "input_tokens", "output_tokens"]) mgone[k] += u[k] || 0;
      continue;
    }
    const m = S.models.find((x) => x.id === u.who);
    mrows.push({ ...u, label: m ? modelName(m) : (u.name || u.who) });
  }
  mrows.sort((a, b) => b.calls - a.calls);
  if (mgone.n) mrows.push({ ...mgone, label: t("modelsGone", mgone.n), muted: true });
  mrows.forEach((u) => line(u, mt));
  $("usage-models-box").hidden = mrows.length === 0;
}

document.querySelectorAll(".tabs button").forEach((b) => {
  b.addEventListener("click", () => {
    document.querySelectorAll(".tabs button").forEach((x) => x.setAttribute("aria-selected", String(x === b)));
    document.querySelectorAll(".tab").forEach((x) => { x.hidden = x.id !== "tab-" + b.dataset.tab; });
    store("omnillm.tab", b.dataset.tab);
  });
});

$("add-device").addEventListener("click", () => startPair($("pair-dash"), () => { closePair($("pair-dash")); refresh(); }, true));
$("add-provider").addEventListener("click", () => { store("omnillm.wizard", ""); fromDash = true; S.providers = []; showWizard(); });
$("wiz-back").addEventListener("click", async () => { fromDash = false; store("omnillm.wizard", "done"); try { await refresh(); } catch (e) { toast(e.message, true); } });
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
