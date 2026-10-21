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
    recheck: "重新检查", recheckNone: "还没发现已登录的订阅或本机模型。", connectedNames: "已接入 {0}",
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
    addDevice: "添加设备", addProvider: "接入模型", addApp: "添加应用", add: "添加",
    name: "名字", lastSeen: "最近使用", budget: "每天额度", todayBudget: "今天 / 每天额度", where: "对话发往", humanOnly: "只能手动对话",
    showOnDevices: "在设备上显示", subAlways: "订阅模型始终只能手动对话", rename: "改名", editModel: "编辑模型 · {0}",
    enabled: "开启", pausedChip: "已暂停", pausedToast: "已暂停“{0}”", resumedToast: "已开启“{0}”",
    online: "在线", todayAt: "今天 {0}", yesterdayAt: "昨天 {0}",
    editDevice: "编辑设备 · {0}", editApp: "编辑应用 · {0}", secStatus: "状态", secModels: "可用模型",
    infoAdded: "添加于 {0}", infoSeen: "最近使用 {0}", infoSpent: "今天已用 {0}",
    pauseHint: "关掉即暂停：这台设备马上停用，配对和设置都保留。", pauseHintApp: "关掉即暂停：这个应用马上停用，令牌和设置都保留。",
    limitHint: "只受整台 Mac 每天总额度（{0}）限制。", allModels: "全部模型（以后接入的新模型也自动可用）",
    grpSub: "订阅", grpKey: "API Key", grpLocal: "这台 Mac", pickOne: "至少选一个模型。", appNoSub: "订阅模型只能手动对话，应用用不了。",
    hiddenTag: "不在设备上显示", appNoModels: "没有能用的模型", appNoModelsBody: "现在接入的模型应用都用不了。先接入 API Key 或这台 Mac 上的模型。",
    removeDevice: "移除设备", removeApp: "移除应用",
    discardQ: "放弃没保存的修改？", discard: "放弃", keepEditing: "继续编辑",
    providers: "模型来源", keyLast4: "接入方式", subLogin: "订阅登录（只留在这台 Mac 上）",
    problemLogin: "登录已失效：在终端运行 claude 并 /login", problemLoginCodex: "登录已失效：在终端运行 codex login",
    problemLimit: "用量到上限，约 {0}后恢复", problemLimitNow: "用量到上限，稍后恢复", hours: "{0} 小时", hour: "1 小时", minutes: "{0} 分钟", minute: "1 分钟",
    noDevices: "还没有设备。点“添加设备”。",
    appsBody: "给不能直接嵌入 OmniLLM 的程序（脚本或其他程序）用。应用只能做后台调用，只能用 API Key 和这台 Mac 上的模型。",
    usageBody: "按这台 Mac 的时间计。花费按公布的价格估算，没有公布价格的按偏高估；订阅和这台 Mac 上的模型不花钱。",
    rangeToday: "今天", rangeYesterday: "昨天", rangeWeek: "最近 7 天",
    totalToday: "今天合计", totalYesterday: "昨天合计", totalWeek: "最近 7 天合计",
    emptyToday: "今天还没有用过。", emptyYesterday: "昨天没有用过。", emptyWeek: "最近 7 天没有用过。",
    byModel: "按模型", modelsGone: "已移除的模型（{0} 个）", byModelTotal: "合计", byModelEarlier: "更早的记录（没有按模型记）",
    who: "设备或应用", spend: "花费", calls: "次数", tokens: "Token（发出 / 收到）",
    usageHost: "这台 Mac", usageGone: "已移除的设备和应用（{0} 个）", usageGoneOne: "{0}（已移除）",
    capLine: "整台 Mac 每天总额度 {0}", capLabel: "整台 Mac 每天总额度", capNone: "整台 Mac 每天总额度：不设上限", capEdit: "修改", noApps: "还没有应用。",
    modelsBody: "“只能手动对话”的模型只给在手机、电脑上聊天的人用，脚本和应用调用不了。订阅模型一直是这样，免得账号被当成自动化调用。",
    claudeSub: "Claude 订阅", codexName: "ChatGPT（Codex）", latest: "（最新版）",
    remove: "移除", removeConfirm: "移除“{0}”？它会立刻用不了。",
    removed: "已移除", connected: "已接入，{0} 个模型可用", saved: "已保存",
    local: "只在这台 Mac", remote: "云端", localChip: "这台 Mac", sub: "订阅",
    todayOf: "{0} / {1}", todayNoLimit: "{0} · 不设上限",
    pairPhone: "手机或平板：用相机扫这个二维码，不用装任何东西。{0} 前有效，只能用一次。",
    pairComputer: "电脑或服务器（没有摄像头）",
    pairComputerNew: "那台电脑还没装 OmniLLM 的话，在它上面运行这一行，装完会提示粘贴配对文字（只用来聊天，不常驻后台）：",
    pairComputerHow: "已经装好的，运行 `omnillm connect pair`，粘贴下面这段配对文字。不要发给别人，也不要贴到聊天里。",
    pairWait: "等待设备连接……",
    pairConfirmQ: "设备“{0}”请求加入。它屏幕上显示的数字是这个吗？",
    pairYes: "一致，添加", pairNo: "不一致，拒绝", pairLeft: "手机那边取消了，或者连接断了。",
    pairDone: "已添加“{0}”", pairFailed: "配对没有完成：{0}",
    pairDenied: "已拒绝。", pairExpired: "这个二维码已过期。", pairNew: "换一个二维码",
    pairStartOffline: "远程连接现在没有连上，生成不了二维码。", retry: "重试",
    close: "关闭", edit: "编辑", save: "保存", cancel: "取消", noLimit: "不设上限",
    budgetLabel: "每天额度（美元）",
    noRelay: "还没有设置远程连接，其他设备暂时连不上。可以先跳过；设置好远程连接后，在“设备”里添加。",
    relayWait: "正在打开远程连接……", relayDown: "远程连接现在连不上，请检查这台 Mac 的网络。连上之后这里会自动继续。",
    relayOn: "设备可以连上", relayOff: "远程连接断开，设备暂时连不上", relayNone: "只在这台 Mac 上用（没有远程连接）",
    regionPaused: "这台 Mac 所在的地区变了，已暂停调用。", regionConfirm: "确认继续",
    appTokenFor: "“{0}”的令牌", appTokenOnce: "只显示这一次，现在复制。",
    appName: "应用的名字",
    tryText: "你好",
    never: "还没用过", copy: "复制", copied: "已复制",
  },
  en: {
    loading: "Loading…",
    loginTitle: "Open the console again",
    loginBody: "Click the OmniLLM icon in the menu bar and choose “Open the console”.",
    loginOr: "Or run in Terminal:",
    loginSafe: "For safety, the console opens only from this Mac.",
    hubDown: "OmniLLM isn’t running. Click the OmniLLM icon in the menu bar and choose “Start OmniLLM”, or run omnillm install in Terminal.",
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
    recheck: "Check again", recheckNone: "No signed-in subscription or local model found yet.", connectedNames: "Connected {0}",
    ollamaFound: "A local model (Ollama) is running on this Mac",
    ollamaUse: "Use the local model",
    source: "Source", otherCompatible: "Other OpenAI-compatible service",
    cname: "Name (e.g. groq)", curl: "API address (https://)",
    key: "API key",
    keyHint: "It is checked with one minimal request first and saved only if it works; it is kept only in this Mac’s keychain.",
    connect: "Check and connect",
    w2Title: "Add a device",
    skipPair: "Skip for now",
    w3Title: "Try it",
    model: "Model", message: "Message", send: "Send", finish: "Done",
    todaySpend: "Spent today", estimate: "Estimated from published prices", devices: "Devices", models: "Models", apps: "Apps", usage: "Usage",
    addDevice: "Add device", addProvider: "Connect a model", addApp: "Add app", add: "Add",
    name: "Name", lastSeen: "Last used", budget: "Daily limit", todayBudget: "Today / daily limit", where: "Messages go to", humanOnly: "Chat only",
    showOnDevices: "Show on devices", subAlways: "Subscription models are always chat-only", rename: "Rename", editModel: "Edit model · {0}",
    enabled: "On", pausedChip: "Paused", pausedToast: "Paused “{0}”", resumedToast: "Turned on “{0}”",
    online: "Online", todayAt: "Today {0}", yesterdayAt: "Yesterday {0}",
    editDevice: "Edit device · {0}", editApp: "Edit app · {0}", secStatus: "Status", secModels: "Models it can use",
    infoAdded: "Added {0}", infoSeen: "Last used {0}", infoSpent: "Spent today {0}",
    pauseHint: "Turn off to pause: the device stops at once; its pairing and settings are kept.", pauseHintApp: "Turn off to pause: the app stops at once; its token and settings are kept.",
    limitHint: "Only the daily total for this Mac ({0}) applies.", allModels: "All models (including ones added later)",
    grpSub: "Subscriptions", grpKey: "API key", grpLocal: "This Mac", pickOne: "Choose at least one model.", appNoSub: "Subscription models are chat-only; apps can’t use them.",
    hiddenTag: "Hidden on devices", appNoModels: "No usable models", appNoModelsBody: "None of the connected models work for apps. Connect an API key or a model on this Mac first.",
    removeDevice: "Remove device", removeApp: "Remove app",
    discardQ: "Discard unsaved changes?", discard: "Discard", keepEditing: "Keep editing",
    providers: "Model sources", keyLast4: "Connected with", subLogin: "Subscription login (stays on this Mac)",
    problemLogin: "Signed out: run claude and /login in Terminal", problemLoginCodex: "Signed out: run codex login in Terminal",
    problemLimit: "Usage limit reached; back in about {0}", problemLimitNow: "Usage limit reached; back soon", hours: "{0} hours", hour: "1 hour", minutes: "{0} minutes", minute: "1 minute",
    noDevices: "No devices yet. Choose “Add device”.",
    appsBody: "For programs that cannot embed OmniLLM (scripts, other apps). Apps make background calls only, with API keys and models on this Mac.",
    usageBody: "Days follow this Mac’s clock. Spend is estimated from published prices (on the high side where none is published); subscriptions and models on this Mac cost nothing.",
    rangeToday: "Today", rangeYesterday: "Yesterday", rangeWeek: "Last 7 days",
    totalToday: "Total today", totalYesterday: "Total yesterday", totalWeek: "Total, last 7 days",
    emptyToday: "Nothing used yet today.", emptyYesterday: "Nothing was used yesterday.", emptyWeek: "Nothing used in the last 7 days.",
    byModel: "By model", modelsGone: "Removed models ({0})", byModelTotal: "Total", byModelEarlier: "Earlier records (not by model)",
    who: "Device or app", spend: "Spend", calls: "Calls", tokens: "Tokens (in / out)",
    usageHost: "This Mac", usageGone: "Removed devices and apps ({0})", usageGoneOne: "{0} (removed)",
    capLine: "Daily total for this Mac: {0}", capLabel: "Daily total for this Mac", capNone: "Daily total for this Mac: no limit", capEdit: "Change", noApps: "No apps yet.",
    modelsBody: "Chat-only models are for people chatting on a phone or computer; scripts and apps can’t use them. Subscription models are always chat-only, so the account isn’t flagged as automated.",
    claudeSub: "Claude subscription", codexName: "ChatGPT (Codex)", latest: " (latest)",
    remove: "Remove", removeConfirm: "Remove “{0}”? It stops working at once.",
    removed: "Removed", connected: "Connected: {0} models available", saved: "Saved",
    local: "This Mac only", remote: "Cloud", localChip: "This Mac", sub: "subscription",
    todayOf: "{0} / {1}", todayNoLimit: "{0} · no limit",
    pairPhone: "Phone or tablet: scan this QR code with the camera; nothing to install. Valid until {0}, single use.",
    pairComputer: "Computer or server (no camera)",
    pairComputerNew: "Not installed there yet? Run this line on that computer; it asks for the pairing text when done (chat only; nothing stays running):",
    pairComputerHow: "Already installed: run `omnillm connect pair` and paste this pairing text. Don’t send it to anyone or paste it into a chat.",
    pairWait: "Waiting for the device…",
    pairConfirmQ: "“{0}” wants to join. Does its screen show this number?",
    pairYes: "Matches, add", pairNo: "Doesn’t match, refuse", pairLeft: "The device cancelled or lost its connection.",
    pairDone: "Added “{0}”", pairFailed: "Pairing did not finish: {0}",
    pairDenied: "Refused.", pairExpired: "This QR code has expired.", pairNew: "New QR code",
    pairStartOffline: "Remote access isn’t connected right now, so no QR code can be made.", retry: "Retry",
    close: "Close", edit: "Edit", save: "Save", cancel: "Cancel", noLimit: "No limit",
    budgetLabel: "Daily limit (USD)",
    noRelay: "Remote access isn’t set up yet, so other devices can’t connect. Skip for now; add devices under Devices once it is set up.",
    relayWait: "Turning on remote access…", relayDown: "Remote access can’t connect right now; check this Mac’s network. It picks up again once connected.",
    relayOn: "Devices can connect", relayOff: "Remote access is down; devices can’t connect now", relayNone: "This Mac only (no remote access)",
    regionPaused: "This Mac’s region changed; calls are paused.", regionConfirm: "Confirm and resume",
    appTokenFor: "Token for “{0}”", appTokenOnce: "Shown once — copy it now.",
    appName: "App name",
    tryText: "Hello",
    never: "Not yet", copy: "Copy", copied: "Copied",
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

// Changes the owner makes (every POST): how many were sent, and how many
// are still on their way. A status poll that overlaps one is dropped, so
// it can't put back what was just changed (a pause switch flipping back).
let changes = 0, changing = 0;

async function api(path, body) {
  const opts = { method: body === undefined ? "GET" : "POST", credentials: "same-origin",
    headers: { "X-OmniLLM-Console": "1", "X-OmniLLM-Lang": lang, "X-OmniLLM-Session": sessionSecret() } };
  if (body !== undefined) { opts.headers["Content-Type"] = "application/json"; opts.body = JSON.stringify(body); changes++; changing++; }
  let r;
  try { r = await fetch("/console/api/" + path, opts); } finally { if (body !== undefined) changing--; }
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

// load asks for everything on the dashboard, with the Usage tab's range.
// The answer is tagged with that range (S.range) and dropped (null) when
// another range was picked meanwhile, or, for a poll, when the owner
// changed something while it was on its way.
async function load(poll) {
  const range = usageRange, before = changes, busyBefore = changing;
  const next = await api("status?range=" + range);
  if (range !== usageRange || (poll && (busyBefore || changing || before !== changes))) return null;
  for (const k of ["providers", "models", "devices", "apps", "usage", "range_usage", "model_usage"]) next[k] = next[k] || [];
  next.range = range;
  sortModels(next.models);
  return next;
}

async function refresh() {
  const next = await load(false);
  if (!next) return; // a newer range was asked for; its answer draws
  S = next;
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
  return !!(curPair || pairTimer || document.querySelector("tr.editing, tr.asking, .cap-edit, .pair-panel, .dialog-scrim"));
}
setInterval(async () => {
  if (!S || signedOut || document.hidden || $("dash").hidden || busy()) return;
  let next;
  try { next = await load(true); } catch (_) { return; }
  if (!next || $("dash").hidden || busy()) return;
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
    startPair($("pair-area"), () => { stepTo(3); fillModels(); }, false);
  } else {
    stepTo(3);
    fillModels();
  }
}

function fillModels() {
  const sel = $("try-model");
  sel.textContent = "";
  for (const m of S.models) {
    const o = el("option", m.locality === "local" ? withDot(modelName(m), t("local")) : isSub(m) ? withDot(modelName(m), t("sub")) : modelName(m));
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

// "Check again": the hub connects every subscription signed in on this Mac
// (even one removed before: asking is adding it again) and a running
// Ollama, and says which.
$("recheck").addEventListener("click", async (ev) => {
  const b = ev.currentTarget;
  b.disabled = true;
  try {
    const r = await api("providers/detect", {});
    const names = r.connected || [];
    toast(names.length ? t("connectedNames", names.join(lang === "zh" ? "、" : ", ")) : t("recheckNone"));
    if (names.length) await refresh();
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
// panel with its x. onDone gets the new device's id.
async function startPair(area, onDone, closable) {
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
    again.addEventListener("click", () => startPair(area, onDone, closable));
    area.append(el("p", msg, isErr ? "err-text" : ""), again);
  };
  fresh();
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
      if (gen !== pairGen) return;
      S.relay_up = r.relay_up; S.relay_configured = r.relay_configured;
      if (r.relay_up || !r.relay_configured) { renderStatus(); startPair(area, onDone, closable); return; }
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
  const right = el("div");
  const state = el("p", t("pairWait"), "muted");
  right.append(state);
  const more = el("details", undefined, "pair-more");
  more.append(el("summary", t("pairComputer")));
  const install = "curl -fsSL https://raw.githubusercontent.com/wensuper/OmniLLM/main/install.sh | sh -s -- --device";
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
      // Only the question now: the QR code and the computer steps are done
      // with, and the owner's eyes go straight to the number.
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
      const decide = async (v) => {
        yes.disabled = no.disabled = true;
        try { await api("pair/decide", { id: p.id, yes: v, name: v ? nameIn.value.trim() : "" }); } catch (e) { yes.disabled = no.disabled = false; toast(e.message, true); }
      };
      yes.addEventListener("click", () => decide(true));
      no.addEventListener("click", () => decide(false));
      nameIn.addEventListener("keydown", (e) => { if (e.key === "Enter" && !yes.disabled) decide(true); });
      actions.append(no, yes);
      c.append(actions);
      box.hidden = true;
      area.append(c);
      c.scrollIntoView({ block: "center" });
      yes.focus({ preventScroll: true });
    }
    if (st.state === "done") {
      stopPair();
      curPair = null;
      area.textContent = "";
      area.append(el("p", t("pairDone", st.device_name)));
      toast(t("pairDone", st.device_name));
      onDone && onDone(st.device_id);
    }
    if (st.state === "failed") {
      curPair = null;
      if (st.reason === "denied") ended(t("pairDenied"));
      else if (st.reason === "left") ended(t("pairLeft"));
      else if (st.reason === "relay") ended(st.message || t("pairStartOffline"), true);
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

// removeButton: Remove in a table row; it asks in a line of its own right
// under the row (the question, then Cancel and Remove).
function removeButton(label, path, id, redraw) {
  const b = el("button", t("remove"), "danger");
  b.type = "button";
  b.addEventListener("click", () => {
    const tr = b.closest("tr");
    tr.classList.add("asking");
    b.hidden = true;
    const ask = el("tr", undefined, "asking ask-row");
    const td = el("td"); td.colSpan = tr.cells.length;
    const no = el("button", t("cancel"), "secondary small-btn"); no.type = "button";
    const yes = el("button", t("remove"), "danger"); yes.type = "button";
    no.addEventListener("click", redraw);
    yes.addEventListener("click", async () => {
      no.disabled = yes.disabled = true;
      try { await api(path, { id }); toast(t("removed")); refresh(); } catch (e) { toast(e.message, true); no.disabled = yes.disabled = false; }
    });
    const btns = el("div", undefined, "btns"); btns.append(no, yes);
    const line = el("div", undefined, "row-ask"); line.append(el("p", t("removeConfirm", label)), btns);
    td.append(line); ask.append(td);
    tr.after(ask);
    no.focus();
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
// On a phone-width window the row stacks: the name (after a dot while it
// is online), and under it what it spent today against its limit.
function nameCell(tr, name, paused, online, spent) {
  const td = cell(tr, "", "nm");
  const line = el("div", undefined, "nm-line");
  if (online) { const dot = el("span", "", "dot-on narrow-only"); dot.setAttribute("role", "img"); dot.setAttribute("aria-label", t("online")); line.append(dot); }
  line.append(el("span", name, "nm-text"));
  if (paused) line.append(el("span", t("pausedChip"), "tag paused-tag"));
  td.append(line);
  if (spent) td.append(el("div", spent, "nm-sub narrow-only"));
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
    const spent = spentText("device/" + d.ID, d.BudgetMicroUSD);
    nameCell(tr, d.Name, d.Paused, d.Online, spent);
    seenText(cell(tr, "", "seen"), d.Online, d.LastSeen);
    cell(tr, spent, "spent");
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
    btns.append(no, save);
    f.append(fields, btns);
    box.append(f);
    amt.focus();
  });
  box.append(b);
}

// ---------- edit dialogs ----------

// dialog: the frame every edit dialog shares: a head with its title and
// x, a body, and a foot with Cancel and then Save on the right (left: an
// optional button such as Remove). With nothing changed, Esc, the x and
// the backdrop close it; with unsaved changes the backdrop does nothing,
// and Esc or the x ask first, in the foot. Enter in a field saves when
// Save can. Tab stays inside. Questions are asked in the foot, never in a
// browser window.
function dialog(titleText) {
  const opener = document.activeElement;
  const scrim = el("div", undefined, "dialog-scrim");
  const dlg = el("div", undefined, "dialog");
  dlg.setAttribute("role", "dialog"); dlg.setAttribute("aria-modal", "true"); dlg.tabIndex = -1;
  const head = el("div", undefined, "dialog-head");
  const title = el("h2", titleText); title.id = "dlg-title";
  dlg.setAttribute("aria-labelledby", title.id);
  const xb = el("button", "", "close-x"); xb.type = "button"; xb.setAttribute("aria-label", t("close")); xb.title = t("close");
  head.append(title, xb);
  const body = el("div", undefined, "dialog-body");
  const foot = el("div", undefined, "dialog-foot");
  const no = el("button", t("cancel"), "secondary"); no.type = "button";
  const save = el("button", t("save"), "primary"); save.type = "button";
  const right = el("div", undefined, "btns"); right.append(no, save);
  const d = { dlg, body, save, left: null, asking: false, dirty: () => false };
  const onKey = (e) => {
    if (e.key === "Escape") { e.preventDefault(); if (d.asking) d.showFoot(); else leave(); return; }
    if (e.key === "Enter" && e.target.tagName === "INPUT" && e.target.type !== "checkbox" && dlg.contains(e.target)) {
      e.preventDefault();
      if (!save.disabled && !d.asking) save.click();
      return;
    }
    if (e.key !== "Tab") return;
    const f = [...dlg.querySelectorAll("button, input, select, textarea, [tabindex]")].filter((n) => !n.disabled && n.offsetParent !== null);
    if (!f.length) return;
    const first = f[0], last = f[f.length - 1];
    if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus(); }
    else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus(); }
    else if (!dlg.contains(document.activeElement) || document.activeElement === dlg) { e.preventDefault(); first.focus(); }
  };
  d.close = () => { scrim.remove(); document.removeEventListener("keydown", onKey, true); if (opener && opener.isConnected) opener.focus(); };
  // showFoot: the foot as it normally is.
  d.showFoot = () => {
    d.asking = false;
    foot.textContent = ""; foot.classList.remove("asking");
    if (d.left) foot.append(d.left);
    foot.append(right);
  };
  // ask: a question in the foot: Cancel (or back) goes back to the form,
  // the other button does it.
  d.ask = (q, yesLabel, onYes, backLabel) => {
    d.asking = true;
    foot.textContent = ""; foot.classList.add("asking");
    const back = el("button", backLabel || t("cancel"), "secondary"); back.type = "button";
    const yes = el("button", yesLabel, "danger"); yes.type = "button";
    back.addEventListener("click", () => { d.showFoot(); });
    yes.addEventListener("click", async () => { back.disabled = yes.disabled = true; if (await onYes() === false) back.disabled = yes.disabled = false; });
    const btns = el("div", undefined, "btns"); btns.append(back, yes);
    foot.append(el("p", q, "foot-q"), btns);
    back.focus();
  };
  const leave = () => { if (d.dirty()) d.ask(t("discardQ"), t("discard"), d.close, t("keepEditing")); else d.close(); };
  xb.addEventListener("click", leave);
  no.addEventListener("click", d.close);
  document.addEventListener("keydown", onKey, true);
  scrim.addEventListener("mousedown", (e) => { if (e.target === scrim && !d.dirty()) d.close(); });
  dlg.append(head, body, foot);
  scrim.append(dlg);
  // open: shows it. On a wide screen the first field takes the caret, at
  // the end of what is there; on a phone nothing is focused, so the
  // keyboard does not cover the dialog before anything is chosen.
  d.open = (field) => {
    d.showFoot();
    document.body.append(scrim);
    if (field && matchMedia("(min-width: 601px)").matches) { field.focus(); const n = field.value.length; field.setSelectionRange(n, n); }
    else dlg.focus({ preventScroll: true });
  };
  return d;
}

// section: a part of a dialog's body, with a heading when it has one.
function section(body, label) {
  const sec = el("section", undefined, "dlg-sec");
  if (label) sec.append(el("h3", label));
  body.append(sec);
  return sec;
}

// usableByApp: apps make background calls only: no chat-only models.
const usableByApp = (m) => !(m.human_only || m.group === "sub");
// appHasModels: an app can use at least one connected model.
function appHasModels(a) {
  return S.models.some((m) => usableByApp(m) && (!(a.Models || []).length || a.Models.includes(m.id)));
}

// editDialog: everything about one device or app in one place: its name,
// on or paused, its daily limit, the models it may use, and removing it.
function editDialog(kind, x) {
  const isApp = kind === "app";
  const path = isApp ? "apps/update" : "devices/update";
  const who = (isApp ? "app/" : "device/") + x.ID;
  const d = dialog(t(isApp ? "editApp" : "editDevice", x.Name));
  // The name, and what is known about it.
  const basics = section(d.body);
  const nid = "dlg-name";
  const nl = el("label", t("name")); nl.htmlFor = nid;
  const name = el("input"); name.id = nid; name.maxLength = 64; name.value = x.Name;
  const u = S.usage.find((r) => r.who === who);
  const info = [t("infoAdded", dateWords(x.Created))];
  if (!isApp) info.push(t("infoSeen", x.Online ? t("online") : timeWords(x.LastSeen)));
  info.push(t("infoSpent", money(u ? u.micro_usd : 0)));
  basics.append(nl, name, el("p", info.join(" · "), "muted small dlg-info"));
  // Status: on, or paused.
  const status = section(d.body, t("secStatus"));
  const on = el("input"); on.type = "checkbox"; on.className = "switch"; on.checked = !x.Paused;
  const onl = el("label", undefined, "toggle"); onl.append(on, el("span", t("enabled")));
  status.append(onl, el("p", t(isApp ? "pauseHintApp" : "pauseHint"), "muted small"));
  // Daily limit.
  const lim = section(d.body, t("budget"));
  const unl = x.BudgetMicroUSD === NO_LIMIT;
  const [wrap, amt] = moneyInput(unl ? "5" : String(x.BudgetMicroUSD / 1e6), t("budgetLabel"));
  amt.disabled = unl;
  const nolim = el("input"); nolim.type = "checkbox"; nolim.className = "switch"; nolim.checked = unl;
  const nll = el("label", undefined, "toggle"); nll.append(nolim, el("span", t("noLimit")));
  const line = el("div", undefined, "budget-line"); line.append(wrap, nll);
  const lhint = el("p", t("limitHint", S.daily_total_micro_usd === NO_LIMIT ? t("noLimit") : money(S.daily_total_micro_usd)), "muted small");
  lim.append(line, lhint);
  // Models it can use: all (also later ones), or the ones switched on. A
  // device never gets a model kept off devices; an app never a chat-only one.
  const msec = section(d.body, t("secModels"));
  const usable = (m) => isApp ? usableByApp(m) : !m.hidden;
  if (isApp && !S.models.some(usableByApp)) msec.append(el("p", t("appNoModelsBody"), "warn-note"));
  const allSw = el("input"); allSw.type = "checkbox"; allSw.className = "switch"; allSw.checked = !(x.Models || []).length;
  const all = el("label", undefined, "toggle wrap"); all.append(allSw, el("span", t("allModels")));
  const list = el("div", undefined, "model-pick");
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
      if (!isApp && m.hidden) nm.append(" ", el("span", t("hiddenTag"), "tag paused-tag"));
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
  // Foot: Remove on the left; it asks in the foot first.
  const rm = el("button", t(isApp ? "removeApp" : "removeDevice"), "danger-text"); rm.type = "button";
  rm.addEventListener("click", () => d.ask(t("removeConfirm", x.Name), t("remove"), async () => {
    try { await api(isApp ? "apps/remove" : "devices/remove", { id: x.ID }); d.close(); toast(t("removed")); refresh(); } catch (e) { toast(e.message, true); return false; }
  }));
  d.left = rm;
  // What the form says now, and whether that differs from what is saved.
  const chosen = () => picks.filter(([, sw]) => sw.checked && !sw.disabled).map(([id]) => id);
  const state = () => JSON.stringify([name.value.trim(), on.checked, nolim.checked, nolim.checked ? 0 : dollars(amt.value), allSw.checked ? [] : chosen()]);
  const start = state();
  d.dirty = () => state() !== start;
  const sync = () => {
    amt.disabled = nolim.checked;
    lhint.hidden = !nolim.checked;
    list.hidden = allSw.checked;
    const none = !allSw.checked && chosen().length === 0;
    pickHint.hidden = !none;
    d.save.disabled = none || !name.value.trim() || !d.dirty();
  };
  allSw.addEventListener("change", () => {
    // Turning "all" off starts from everything it may use.
    if (!allSw.checked && chosen().length === 0) picks.forEach(([, sw]) => { if (!sw.disabled) sw.checked = true; });
    sync();
  });
  d.dlg.addEventListener("input", sync);
  d.dlg.addEventListener("change", sync);
  d.save.addEventListener("click", async () => {
    const b = { id: x.ID, name: name.value, paused: !on.checked, models: allSw.checked ? [] : chosen() };
    if (nolim.checked) b.no_limit = true; else b.budget_usd = dollars(amt.value);
    d.save.disabled = true;
    try { await api(path, b); d.close(); toast(t("saved")); await refresh(); } catch (e) { toast(e.message, true); sync(); }
  });
  sync();
  d.open(name);
}

// modelDialog: one model's name and switches, for a phone-width window
// where the table keeps only the name and an Edit button.
function modelDialog(m) {
  const cur = modelName(m);
  const d = dialog(t("editModel", cur));
  const sec = section(d.body);
  const nl = el("label", t("name")); nl.htmlFor = "dlg-model-name";
  const name = el("input"); name.id = "dlg-model-name"; name.maxLength = 64; name.value = cur;
  sec.append(nl, name);
  const sw = (checked, label) => {
    const box = el("input"); box.type = "checkbox"; box.className = "switch"; box.checked = checked;
    const l = el("label", undefined, "toggle wrap"); l.append(box, el("span", label));
    return [l, box];
  };
  const [sl, shown] = sw(!m.hidden, t("showOnDevices"));
  const [hl, human] = sw(!!m.human_only || isSub(m), t("humanOnly"));
  human.disabled = isSub(m);
  const opts = section(d.body);
  opts.append(sl, hl);
  if (isSub(m)) opts.append(el("p", t("subAlways"), "muted small"));
  // Only what changed is sent; an unchanged name is not a rename.
  const diff = () => {
    const b = {};
    if (name.value.trim() !== cur) b.name = name.value.trim();
    if (shown.checked === !!m.hidden) b.shown = shown.checked;
    if (!isSub(m) && human.checked !== !!m.human_only) b.human_only = human.checked;
    return b;
  };
  d.dirty = () => Object.keys(diff()).length > 0;
  const sync = () => { d.save.disabled = !d.dirty(); };
  d.dlg.addEventListener("input", sync);
  d.dlg.addEventListener("change", sync);
  d.save.addEventListener("click", async () => {
    d.save.disabled = true;
    try { await api("models/set", { id: m.id, ...diff() }); d.close(); toast(t("saved")); await refresh(); } catch (e) { toast(e.message, true); sync(); }
  });
  sync();
  d.open(name);
}

// modelName: a model as people see it here: the owner's own name as it
// is; else Codex is "ChatGPT (Codex)", and a model that follows its
// family's newest version (not known yet) says so, in the page's language.
function modelName(m) {
  if (m.renamed) return m.display_name;
  if ((m.kind || kindOf(m.provider)) === "codex") return t("codexName");
  return m.display_name + (m.latest ? t("latest") : "");
}
// withDot: "a · b", with no space after a full-width bracket
// ("ChatGPT（Codex）· 订阅").
const withDot = (a, b) => a + (/[）」』】]$/.test(a) ? "· " : " · ") + b;
// sortModels: the models of one source stay together, in the order the
// sources come; Claude's go Opus, Sonnet, Haiku.
function sortModels(ms) {
  const src = [...new Set(ms.map((m) => m.provider))];
  const rank = (m) => { const id = m.id.toLowerCase(); return id.includes("opus") ? 0 : id.includes("sonnet") ? 1 : id.includes("haiku") ? 2 : 3; };
  ms.sort((a, b) => src.indexOf(a.provider) - src.indexOf(b.provider) || rank(a) - rank(b));
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
    const nc = cell(tr, isSub(m) ? withDot(name, t("sub")) : name, "nm");
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
    const rn = el("button", t("rename"), "secondary small-btn wide-only"); rn.type = "button";
    rn.addEventListener("click", () => renameModel(tr, nc, act, m, name));
    // A phone-width window keeps the name and this; the rest is in a dialog.
    const ed = el("button", t("edit"), "secondary small-btn narrow-only"); ed.type = "button";
    ed.addEventListener("click", () => modelDialog(m));
    act.append(rn, ed);
    tb.append(tr);
  }
  const pb = $("providers");
  pb.textContent = "";
  for (const p of S.providers) {
    const tr = el("tr");
    const label = p.kind === "claude-code" ? t("claudeSub") : p.kind === "codex" ? t("codexName") : (p.display_name || p.name);
    const nc = cell(tr, p.local ? withDot(label, t("local")) : label);
    const pt = problemText(p);
    if (pt) { nc.append(document.createElement("br"), el("span", pt, "tag warn-tag")); }
    cell(tr, p.subscription ? t("subLogin") : p.key_last4 ? "API Key …" + p.key_last4 : t("local"));
    cell(tr, "", "actions").append(removeButton(label, "providers/remove", p.name, renderModels));
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
  act.append(no, save);
  const submit = async () => {
    // The name as it was is not a rename: nothing to send.
    if (inp.value.trim() === cur) { renderModels(); return; }
    save.disabled = true;
    try { await api("models/set", { id: m.id, name: inp.value.trim() }); toast(t("saved")); await refresh(); } catch (e) { toast(e.message, true); save.disabled = false; }
  };
  save.addEventListener("click", submit);
  no.addEventListener("click", renderModels);
  inp.addEventListener("keydown", (e) => { if (e.key === "Enter") submit(); if (e.key === "Escape") renderModels(); });
  inp.focus(); inp.setSelectionRange(inp.value.length, inp.value.length);
}

function renderApps() {
  const tb = $("apps");
  tb.textContent = "";
  for (const a of S.apps) {
    const tr = el("tr", undefined, a.Paused ? "paused" : "");
    const spent = spentText("app/" + a.ID, a.BudgetMicroUSD);
    const nc = nameCell(tr, a.Name, a.Paused, false, spent);
    // Nothing it may use works for apps (only subscriptions or chat-only
    // models are connected, or its own list has none left).
    if (!appHasModels(a)) nc.firstChild.append(el("span", t("appNoModels"), "tag warn-tag"));
    cell(tr, spent, "spent");
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
// rangeWords: the total line and empty line of the range the data is for.
const rangeWords = (r) => r === "yesterday" ? [t("totalYesterday"), t("emptyYesterday")]
  : r === "week" ? [t("totalWeek"), t("emptyWeek")] : [t("totalToday"), t("emptyToday")];
const pressRange = () => document.querySelectorAll("#usage-range button").forEach((b) => b.setAttribute("aria-pressed", String(b.dataset.range === usageRange)));
// A click shows the new choice at once; the tables change when that
// range's numbers arrive (an older answer that comes later is dropped).
document.querySelectorAll("#usage-range button").forEach((b) => {
  b.addEventListener("click", async () => {
    if (b.dataset.range === usageRange) return;
    usageRange = b.dataset.range;
    store("omnillm.range", usageRange);
    pressRange();
    try { await refresh(); } catch (e) { toast(e.message, true); }
  });
});

// renderUsage: one line per device or app by its name (never an internal
// id), busiest first; removed devices folded into one line; the range's
// total at the bottom. Below it the same per model.
const num = (n) => Number(n || 0).toLocaleString(lang === "zh" ? "zh-CN" : "en-US");
function renderUsage() {
  pressRange();
  const [totalLabel, emptyText] = rangeWords(S.range);
  $("usage-body").textContent = t("usageBody");
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
  const mt = $("usage-models"), mfoot = $("usage-models-total");
  mt.textContent = ""; mfoot.textContent = "";
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
  // Calls from before use was kept per model are in the total above but in
  // no model's line: one line says so, so the two totals agree.
  const fields = ["calls", "micro_usd", "input_tokens", "output_tokens"];
  const sum = Object.fromEntries(fields.map((k) => [k, mrows.reduce((n, u) => n + (u[k] || 0), 0)]));
  if (total && mrows.length && (total.calls > sum.calls || total.micro_usd > sum.micro_usd)) {
    mrows.push({ ...Object.fromEntries(fields.map((k) => [k, Math.max(0, (total[k] || 0) - sum[k])])), label: t("byModelEarlier"), muted: true });
  }
  mrows.forEach((u) => line(u, mt));
  if (mrows.length) line({ ...(total || sum), label: t("byModelTotal") }, mfoot, "total");
  $("usage-models-box").hidden = mrows.length === 0;
}

document.querySelectorAll(".tabs button").forEach((b) => {
  b.addEventListener("click", () => {
    document.querySelectorAll(".tabs button").forEach((x) => x.setAttribute("aria-selected", String(x === b)));
    document.querySelectorAll(".tab").forEach((x) => { x.hidden = x.id !== "tab-" + b.dataset.tab; });
    store("omnillm.tab", b.dataset.tab);
  });
});

$("add-device").addEventListener("click", () => startPair($("pair-dash"), (id) => { closePair($("pair-dash")); afterPair(id); }, true));

// afterPair: the new device opens its session a moment after it is
// added; look again (every second, for up to 10 s) until it shows online.
function afterPair(id) {
  let n = 0;
  const look = async () => {
    try { await refresh(); } catch (_) { return; }
    const d = S.devices.find((x) => x.ID === id);
    if (d && !d.Online && ++n < 10 && !signedOut) setTimeout(look, 1000);
  };
  look();
}
$("add-provider").addEventListener("click", () => { store("omnillm.wizard", ""); fromDash = true; S.providers = []; showWizard(); });
$("wiz-back").addEventListener("click", async () => { fromDash = false; store("omnillm.wizard", "done"); try { await refresh(); } catch (e) { toast(e.message, true); } });
// "Add app": a name field in place, then the app's token, shown this once
// with its Copy button.
const appForm = $("app-add"), appName = $("app-name");
const closeAppForm = () => { appForm.hidden = true; $("add-app").hidden = false; appName.value = ""; };
$("add-app").addEventListener("click", () => {
  appForm.hidden = false; $("add-app").hidden = true; $("app-add-ok").disabled = true;
  appName.focus();
});
appName.addEventListener("input", () => { $("app-add-ok").disabled = !appName.value.trim(); });
appName.addEventListener("keydown", (e) => { if (e.key === "Escape") closeAppForm(); });
$("app-add-cancel").addEventListener("click", closeAppForm);
appForm.addEventListener("submit", async (ev) => {
  ev.preventDefault();
  const name = appName.value.trim();
  if (!name) return;
  const ok = $("app-add-ok");
  ok.disabled = true;
  try {
    const r = await api("apps/add", { name });
    closeAppForm();
    const box = $("app-token");
    box.textContent = "";
    box.append(el("p", t("appTokenFor", r.name), "token-for"), codeBox(r.token), el("p", t("appTokenOnce"), "muted small"));
    box.hidden = false;
    refresh();
  } catch (e) { toast(e.message, true); ok.disabled = false; }
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
