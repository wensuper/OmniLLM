// OmniLLM web client: a phone or tablet becomes a device of the owner's hub.
// All crypto runs in omni.wasm (the same Go device code as `omnillm
// connect`); this file is the page around it. It never builds HTML from
// strings: every text goes through textContent.
"use strict";
// The build this script belongs to (stamped by scripts/build-web.sh, the
// same in index.html and sw.js).
const BUILD = "dev";
if (window.top !== window.self) { document.documentElement.textContent = ""; throw new Error("framed"); }

// ---------- language ----------
// Words people see (glossary): the hub is "your Mac", the relay is "remote
// access"; "host" and "hub" stay in developer text only.
const T = {
  zh: {
    loading: "正在载入……", bootDownload: "正在下载……", bootStart: "正在启动……", bootStorage: "正在打开……",
    bootSlow: "载入停了 20 秒没有进展，停在“{0}”。点下面的按钮清掉缓存重新载入；网络慢的话也可以换个网络再试。",
    bootCrashed: "程序意外停止了。点下面的按钮清掉缓存重新载入。", bootError: "载入失败：", bootRetry: "重新载入",
    welcomeTitle: "在这台设备上用你的模型",
    welcomeBody: "在你的 Mac 上打开 OmniLLM，点“添加设备”，再用这台设备扫屏幕上的二维码。",
    scan: "扫描二维码", pasteInstead: "扫不了码？粘贴配对文字", pastePh: "粘贴配对文字（以 omnillm3- 开头）", add: "添加", cancel: "取消", back: "返回", delete: "删除", remove: "移除", gotIt: "知道了",
    scanHint: "把 Mac 屏幕上的二维码放进框里", camDenied: "没有拿到摄像头权限。可以改用下面的“粘贴配对文字”。",
    pairTitle: "添加这台设备", pairConnecting: "正在连接你的 Mac……",
    pairAsk: "要把这台设备添加到你的 Mac 吗？只有你刚在 Mac 上点了“添加设备”才继续。", pairVia: "经由 {0}",
    pairReplace: "这会替换这台设备现在的配对（{0}），之前的 Mac 就不能再从这里用了。",
    pairCheck: "在 Mac 上看到同样的数字，就在 Mac 上点“一致，添加”。", pairWait: "等你在 Mac 上确认……", pairMismatch: "数字不一样？取消", pairDone: "已添加。",
    pairDropped: "连接断了。在 Mac 上点“换一个二维码”再扫。", connectedMac: "已连上你的 Mac",
    notCode: "这不是 OmniLLM 的配对二维码或配对文字。", needTLS: "你的 Mac 用的远程连接不支持手机网页（需要 wss:// 地址）。请在 Mac 上设置一个公开的远程连接。",
    chats: "对话", newChat: "新对话", placeholder: "输入消息", stop: "停止", send: "发送", menu: "菜单", model: "模型",
    addImage: "添加图片", removeImage: "移除图片", tooManyImages: "每条消息最多 4 张图片。", imageFailed: "这张图片打不开，换一张试试。", imageTooBig: "图片太大，换一张小一点的。", imageTitle: "图片",
    offline: "连不上你的 Mac：它可能睡眠了或者没联网。醒来后会自动恢复。", retry: "重试",
    noNetwork: "这台设备没有联网，联网后会自动连上。", kept: "刚才输入的还在。", loadingModels: "正在连接你的 Mac……",
    noModels: "这台设备现在没有能用的模型。在 Mac 的控制台里接入模型，或在“设备”里给它开放模型。", noModelsOpt: "没有可用的模型",
    modelGone: "原来的模型已从你的 Mac 上移除，换成了 {0}",
    grpSub: "订阅（不另花钱）", grpKey: "API Key", grpLocal: "你的 Mac 上", latest: "（最新版）", codex: "ChatGPT（Codex · 最新版）", codexV: "ChatGPT（Codex · {0}）",
    forget: "移除这台设备", forgetQ: "这里的对话会删掉。以后要用，在 Mac 上重新添加。", removing: "正在移除……",
    forgetFar: "这台设备已移除，对话已删除。你的 Mac 现在连不上，之后记得在 Mac 的“设备”里也移除它。",
    close: "关闭", deleteChat: "删除这个对话", deleteChatQ: "删除对话“{0}”？", chatsLocal: "对话只保存在这台设备上。", searchChats: "搜索对话", noMatch: "没有找到",
    addHome: "添加到主屏幕", addHomeTip: "点浏览器的“分享”按钮，选“添加到主屏幕”。第一次从主屏幕打开时，需要再扫一次码。",
    deviceLine: "这台设备：{0}", empty: "有什么可以帮你？", cut: "（回复太长被截断了。发“继续”让它接着写。）", stopped: "（已停止）",
    today: "今天", copy: "复制", copied: "已复制", again: "重新回答", againWith: "用 {0} 再答一次", thinking: "正在思考",
    earlier: "之前的回答（{0}）· 展开", earlierOpen: "之前的回答（{0}）· 收起",
    dropped: "（连接断了，回答没写完。连上后点“重试”。）",
    longChat: "对话已经很长，每次都会整段发出。开个新对话会更快，用 API Key 时也更省。", paused: "已暂停", pausedDot: "已在你的 Mac 上暂停", waited: "{0} 秒", slow: "比平常慢。可以点停止，换个模型再问。",
    switchedTo: "换成了 {0}", newContent: "↓ 新内容", waitReply: "等这条回答完成，或先点停止。",
    liveOn: "在线：你的 Mac 连得上，模型可以用", liveOff: "离线：现在连不上你的 Mac", liveWait: "正在连接你的 Mac……",
    volatile: "这个浏览器不能安全地保存配对，所以只在这个页面打开时有效；关掉后要在 Mac 上重新添加。换用 Safari 或 Chrome 可以一直保存。",
    removedHere: "这台设备已从你的 Mac 上移除，这里的对话也已清除。要再用，在 Mac 上点“添加设备”再扫一次。",
    otherTab: "OmniLLM 已在另一个标签页打开。", useHere: "在这里用",
  },
  en: {
    loading: "Loading…", bootDownload: "Downloading…", bootStart: "Starting…", bootStorage: "Opening…",
    bootSlow: "Loading has made no progress for 20 seconds, stuck at “{0}”. Tap below to clear the cache and load again; on a slow network, another network may help.",
    bootCrashed: "The program stopped unexpectedly. Tap below to clear the cache and load again.", bootError: "Loading failed: ", bootRetry: "Load again",
    welcomeTitle: "Use your models on this device",
    welcomeBody: "On your Mac, open OmniLLM, choose “Add device”, and scan the QR code on its screen with this device.",
    scan: "Scan QR code", pasteInstead: "Can’t scan? Paste the pairing text", pastePh: "Paste the pairing text (starts with omnillm3-)", add: "Add", cancel: "Cancel", back: "Back", delete: "Delete", remove: "Remove", gotIt: "Got it",
    scanHint: "Point at the QR code on your Mac’s screen", camDenied: "No camera access. Use “Paste the pairing text” below instead.",
    pairTitle: "Add this device", pairConnecting: "Connecting to your Mac…",
    pairAsk: "Add this device to your Mac? Continue only if you just chose “Add device” on your Mac.", pairVia: "Via {0}",
    pairReplace: "This replaces this device’s current pairing ({0}); the previous Mac can no longer be used from here.",
    pairCheck: "If your Mac shows the same number, choose “Matches, add” there.", pairWait: "Waiting for you to confirm on your Mac…", pairMismatch: "Numbers don’t match? Cancel", pairDone: "Added.",
    pairDropped: "The connection dropped. Choose “New QR code” on your Mac and scan again.", connectedMac: "Connected to your Mac",
    notCode: "This is not an OmniLLM pairing QR code or pairing text.", needTLS: "Your Mac’s remote access can’t be used from a phone browser (it needs a wss:// address). Set up public remote access on your Mac.",
    chats: "Chats", newChat: "New chat", placeholder: "Message", stop: "Stop", send: "Send", menu: "Menu", model: "Model",
    addImage: "Add image", removeImage: "Remove image", tooManyImages: "Up to 4 images per message.", imageFailed: "This image can’t be opened; try another one.", imageTooBig: "Image too large; try a smaller one.", imageTitle: "Image",
    offline: "Can’t reach your Mac: it may be asleep or offline. It reconnects when your Mac wakes.", retry: "Retry",
    noNetwork: "This device is offline; it reconnects when the network is back.", kept: "What you typed is kept.", loadingModels: "Connecting to your Mac…",
    noModels: "No models are available to this device. On your Mac, connect a model or allow this device more models under Devices.", noModelsOpt: "No models",
    modelGone: "The previous model was removed from your Mac; switched to {0}",
    grpSub: "Subscriptions (no extra cost)", grpKey: "API key", grpLocal: "On your Mac", latest: " (latest)", codex: "ChatGPT (Codex · latest)", codexV: "ChatGPT (Codex · {0})",
    forget: "Remove this device", forgetQ: "Chats here will be deleted. To use this device later, add it on your Mac again.", removing: "Removing…",
    forgetFar: "This device was removed and its chats deleted. Your Mac can’t be reached right now, so remove it under “Devices” on your Mac later too.",
    close: "Close", deleteChat: "Delete this chat", deleteChatQ: "Delete the chat “{0}”?", chatsLocal: "Chats are kept on this device only.", searchChats: "Search chats", noMatch: "No matching chats",
    addHome: "Add to Home Screen", addHomeTip: "Tap the browser’s “Share” button and choose “Add to Home Screen”. The first time it’s opened from the Home Screen, you’ll scan the QR code once more.",
    deviceLine: "This device: {0}", empty: "How can I help?", cut: "(The reply was cut off. Send “continue” to have it go on.)", stopped: "(stopped)",
    today: "Today", copy: "Copy", copied: "Copied", again: "Answer again", againWith: "Answer again with {0}", thinking: "Thinking",
    earlier: "Earlier answer ({0}) · Show", earlierOpen: "Earlier answer ({0}) · Hide",
    dropped: "(Connection lost; the reply is incomplete. Tap Retry once reconnected.)",
    longChat: "This chat is long and is sent in full each time. A new chat is faster, and cheaper with an API key.", paused: "Paused", pausedDot: "Paused on your Mac", waited: "{0}s", slow: "Slower than usual. You can tap Stop and try another model.",
    switchedTo: "Switched to {0}", newContent: "↓ New", waitReply: "Wait for this reply, or tap Stop first.",
    liveOn: "Online: your Mac answers; models are ready", liveOff: "Offline: your Mac can’t be reached now", liveWait: "Connecting to your Mac…",
    volatile: "This browser can’t keep the pairing safely, so it only lasts while this page is open; after closing it, add the device on your Mac again. Safari or Chrome keep it.",
    removedHere: "This device was removed from your Mac and its chats here were cleared. To use it again, choose “Add device” on your Mac and scan again.",
    otherTab: "OmniLLM is open in another tab.", useHere: "Use here",
  },
};
const lang = (navigator.language || "en").toLowerCase().startsWith("zh") ? "zh" : "en";
document.documentElement.lang = lang === "zh" ? "zh-CN" : "en";
function t(k, ...a) { let s = (T[lang][k] ?? T.en[k] ?? k); a.forEach((v, i) => { s = s.replace("{" + i + "}", v); }); return s; }
const $ = (id) => document.getElementById(id);
function el(tag, text, cls) { const e = document.createElement(tag); if (text !== undefined) e.textContent = text; if (cls) e.className = cls; return e; }
// quoted sets a text whose quoted UI names (“添加设备”) never break inside:
// each “…” goes into a nowrap span. Still text nodes only.
function quoted(e, s) {
  e.textContent = "";
  String(s).split(/(“[^”]*”)/).forEach((part, i) => { if (part) e.append(i % 2 ? el("span", part, "q") : part); });
  return e;
}
document.querySelectorAll("[data-t]").forEach((e) => { quoted(e, t(e.dataset.t)); });
document.querySelectorAll("[data-ta]").forEach((e) => { e.setAttribute("aria-label", t(e.dataset.ta)); });
function show(id) { document.querySelectorAll(".screen").forEach((s) => { s.hidden = s.id !== id; }); }

// ---------- storage: IndexedDB, encrypted with a key the page cannot export ----------
const idb = (() => {
  let dbp;
  function db() {
    dbp ??= new Promise((res, rej) => {
      const r = indexedDB.open("omnillm", 1);
      r.onupgradeneeded = () => r.result.createObjectStore("kv");
      r.onsuccess = () => res(r.result);
      r.onerror = () => rej(r.error);
    });
    return dbp;
  }
  async function op(mode, f) {
    const d = await db();
    return new Promise((res, rej) => {
      const tx = d.transaction("kv", mode);
      const req = f(tx.objectStore("kv"));
      tx.oncomplete = () => res(req && req.result);
      tx.onerror = () => rej(tx.error);
    });
  }
  return {
    get: (k) => op("readonly", (s) => s.get(k)),
    put: (k, v) => op("readwrite", (s) => s.put(v, k)),
    del: (k) => op("readwrite", (s) => s.delete(k)),
  };
})();

let wrapKeyP = null;
function wrapKey() {
  wrapKeyP ??= (async () => {
    let k = await idb.get("wrap");
    if (!k) {
      k = await crypto.subtle.generateKey({ name: "AES-GCM", length: 256 }, false, ["encrypt", "decrypt"]);
      // Some browsers can't keep a non-exportable key in storage (it throws
      // "can not be cloned"). Then the key lives only while this page is
      // open: pairing and chatting work, and the page says that closing it
      // means pairing again. The key is never made exportable.
      try { await idb.put("wrap", k); } catch (e) { S.volatile = true; }
    }
    return k;
  })();
  return wrapKeyP;
}
// seal stores value encrypted. A write that started before unpaired()
// (S.gen moved on meanwhile) is dropped, so a late save can't bring back
// a pairing or chats that were just removed.
async function seal(name, value) {
  const gen = S.gen;
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const ct = await crypto.subtle.encrypt({ name: "AES-GCM", iv, additionalData: new TextEncoder().encode(name) }, await wrapKey(),
    new TextEncoder().encode(JSON.stringify(value)));
  if (gen !== S.gen) return;
  await idb.put(name, { iv, ct });
}
async function unseal(name) {
  const box = await idb.get(name);
  if (!box) return null;
  try {
    const pt = await crypto.subtle.decrypt({ name: "AES-GCM", iv: box.iv, additionalData: new TextEncoder().encode(name) }, await wrapKey(), box.ct);
    return JSON.parse(new TextDecoder().decode(pt));
  } catch { return null; }
}

// ---------- the Go side ----------
const ready = new Promise((res) => addEventListener("omnillm-ready", res, { once: true }));
// boot shows the loading step; the watchdog names it if loading stalls.
const B = { stage: "", started: Date.now(), err: "", timer: null };
function boot(stage, detail) {
  B.stage = stage;
  if (!$("boot").hidden) $("boot-text").textContent = t(stage) + (detail ? " " + detail : "");
}
function bootFailed(why) {
  clearTimeout(B.timer);
  $("boot-text").textContent = why;
  $("boot-retry").hidden = false;
}
// bootWatch fires when loading makes no progress for 20 s: a slow but
// moving download keeps it waiting (each chunk calls bootWatch again).
function bootWatch() {
  clearTimeout(B.timer);
  B.timer = setTimeout(() => {
    if (B.done) return;
    bootFailed(t("bootSlow", t(B.stage)) + (B.err ? " (" + B.err + ")" : ""));
  }, 20000);
}
// Errors while loading are shown, not swallowed.
addEventListener("error", (e) => { if (!B.done) B.err = String(e.message || e.error || "error").slice(0, 160); });
addEventListener("unhandledrejection", (e) => { if (!B.done) B.err = String(e.reason && e.reason.message || e.reason).slice(0, 160); });
async function cleanReload() {
  try {
    if (navigator.serviceWorker) await Promise.all((await navigator.serviceWorker.getRegistrations()).map((r) => r.unregister()));
    if (window.caches) await Promise.all((await caches.keys()).map((k) => caches.delete(k)));
  } catch {}
  location.reload();
}
async function startWasm() {
  boot("bootDownload");
  const res = await fetch("omni.wasm");
  if (!res.ok) throw new Error("omni.wasm HTTP " + res.status);
  // Read it ourselves to show progress (the size is the compressed one when
  // the server says so; then the percentage is only an estimate).
  const total = Number(res.headers.get("content-length")) || 0;
  const reader = res.body.getReader();
  const parts = []; let got = 0;
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    parts.push(value); got += value.length;
    bootWatch();
    boot("bootDownload", total && total >= got ? Math.floor(got * 100 / total) + "%" : (got / 1048576).toFixed(1) + " MB");
  }
  const bytes = new Uint8Array(got); let o = 0;
  for (const p of parts) { bytes.set(p, o); o += p.length; }
  boot("bootStart"); bootWatch();
  const go = new Go();
  const r = await WebAssembly.instantiate(bytes, go.importObject);
  go.run(r.instance).then(() => { if (!B.done) bootFailed(t("bootCrashed")); });
  await ready;
}

// ---------- state ----------
const S = { device: null, models: [], chats: [], cur: null, busy: null, online: true, gen: 0 };
const isStandalone = matchMedia("(display-mode: standalone)").matches || navigator.standalone === true;
function deviceName() {
  const ua = navigator.userAgent;
  if (/iPad/.test(ua) || (/Macintosh/.test(ua) && navigator.maxTouchPoints > 1)) return "iPad";
  if (/iPhone/.test(ua)) return "iPhone";
  if (/Android/.test(ua)) return lang === "zh" ? (/Mobile/.test(ua) ? "安卓手机" : "安卓平板") : (/Mobile/.test(ua) ? "Android phone" : "Android tablet");
  return lang === "zh" ? "网页浏览器" : "Web browser";
}

// ---------- one tab at a time ----------
// Two tabs would each save their own copy of the chats over the other's.
// A tab that opens says so; the others go quiet (they write nothing) until
// the person picks one with "Use here".
const tabs = "BroadcastChannel" in window ? new BroadcastChannel("omnillm") : null;
function claimTab() { if (tabs) tabs.postMessage("here"); }
if (tabs) tabs.onmessage = (e) => {
  if (e.data !== "here" || S.quiet) return;
  S.quiet = true;
  if (S.busy) { try { omnillm.cancel(S.busy); } catch {} }
  $("other-tab").hidden = false;
};
// Use here: reload, so this tab reads what the other one saved, and claim.
$("use-here").addEventListener("click", () => {
  try { sessionStorage.setItem("omnillm.draft", $("input").value); } catch {}
  location.reload();
});
claimTab();

// ---------- start ----------
// A page made of two versions (loaded while the server switched to a new
// one) is reloaded once from the network, the cache cleared.
(function sameBuild() {
  const page = (document.querySelector('meta[name="omnillm-build"]') || {}).content;
  if (page === BUILD) return;
  let tried = false;
  try { tried = sessionStorage.getItem("omnillm.rebuilt") === BUILD; sessionStorage.setItem("omnillm.rebuilt", BUILD); } catch {}
  if (tried) return;
  Promise.all([
    navigator.serviceWorker ? navigator.serviceWorker.getRegistrations().then((rs) => Promise.all(rs.map((r) => r.unregister()))) : null,
    window.caches ? caches.keys().then((ks) => Promise.all(ks.map((k) => caches.delete(k)))) : null,
  ]).finally(() => location.reload());
  throw new Error("mixed build");
})();

// New versions: the service worker caches the new files and takes over;
// an open page (phones keep tabs for weeks) then reloads itself as soon as
// no reply is on its way, keeping what was typed.
const UP = { ready: false };
function watchUpdates() {
  if (!navigator.serviceWorker) return;
  navigator.serviceWorker.addEventListener("controllerchange", () => { UP.ready = true; reloadIfIdle(); });
  const check = () => navigator.serviceWorker.getRegistration().then((r) => r && r.update()).catch(() => {});
  document.addEventListener("visibilitychange", () => { if (!document.hidden) check(); });
  setInterval(check, 30 * 60 * 1000);
}
function reloadIfIdle() {
  // A quiet tab stays quiet: reloading would take over from the tab in use.
  if (!UP.ready || S.quiet || S.busy || (S.pending && S.pending.length)) return;
  try { sessionStorage.setItem("omnillm.draft", $("input").value); } catch {}
  location.reload();
}

$("boot-retry").addEventListener("click", cleanReload);

(async function main() {
  $("boot-text").textContent = t("loading");
  // A pairing link carries the code in the fragment; take it and wipe it
  // from the address bar and history at once.
  let link = "";
  if (location.hash.startsWith("#pair=")) {
    link = location.hash.slice(6);
    history.replaceState(null, "", location.pathname + location.search);
  }
  if ("serviceWorker" in navigator) navigator.serviceWorker.register("sw.js").catch(() => {});
  watchUpdates();
  if (navigator.storage && navigator.storage.persist) navigator.storage.persist().catch(() => {});
  bootWatch();
  try { await startWasm(); } catch (e) { bootFailed(t("bootError") + String(e && e.message || e)); return; }
  S.ready = true;
  boot("bootStorage"); bootWatch();
  if (S.pendingLink) { link = S.pendingLink; S.pendingLink = ""; }
  // A link that arrived while loading may have changed the address without
  // a hashchange event (seen once in 10 rounds): read the address itself.
  if (!link && location.hash.startsWith("#pair=")) {
    link = location.hash.slice(6);
    history.replaceState(null, "", location.pathname + location.search);
  }
  try {
    S.device = await unseal("device");
    S.chats = (await unseal("chats")) || [];
    for (const c of S.chats) for (const m of c.messages) delete m.live; // a reply cut by a reload is no longer on its way
  } catch (e) { bootFailed(t("bootError") + String(e && e.message || e)); return; }
  B.done = true; clearTimeout(B.timer);
  if (S.device && !link) return openChat();
  if (link) return pair(link, true);
  show("welcome");
})();

// A pairing link opened in a tab where the page is already loaded only
// changes the fragment.
addEventListener("hashchange", () => {
  if (!location.hash.startsWith("#pair=")) return;
  const code = location.hash.slice(6);
  history.replaceState(null, "", location.pathname + location.search);
  // Still loading: main() picks it up once the device code is ready.
  if (!S.ready) { S.pendingLink = code; return; }
  if (!S.busy && !S.pairing) pair(code, true);
});

// ---------- welcome ----------
// With the paste box open, its Add button is the one to press.
$("paste-text").placeholder = t("pastePh");
$("paste").addEventListener("toggle", () => {
  const open = $("paste").open;
  $("paste-go").className = open ? "primary" : "secondary";
  $("scan").className = open ? "secondary" : "primary";
});
// welcomeNote: a calm notice at the top of the welcome screen (removed,
// or remove it on the Mac too); the paste box folded so it reads first.
// While it shows, the welcome body (how to add a device) steps aside.
function welcomeNote(key) {
  $("welcome-note").hidden = !key;
  quoted($("welcome-note"), key ? t(key) : "");
  $("welcome-body").hidden = !!key;
  if (key) $("paste").open = false;
}

// ---------- pairing ----------
$("paste-go").addEventListener("click", () => {
  const v = $("paste-text").value.trim();
  if (!v) return;
  pair(v);
});
$("pair-retry").addEventListener("click", () => { show(S.device ? "chat" : "welcome"); });
// Different numbers (someone else's pairing), or Cancel while connecting:
// stop it and go back. A phone that already left its old pairing has none
// to go back to.
function stopPairing() {
  S.pairStopped = true;
  try { omnillm.pairCancel(); } catch {}
  sasShown(false); $("pair-stop").hidden = true;
  show(S.device && !S.leftOld ? "chat" : "welcome");
}
$("pair-mismatch").addEventListener("click", stopPairing);
$("pair-stop").addEventListener("click", stopPairing);
function sasShown(on) { S.sasUp = on; $("sas").hidden = $("pair-wait").hidden = $("pair-mismatch").hidden = !on; }

// pair connects this device. A link (fromLink) could have been sent by
// anyone, so the person first sees which remote access it leads through
// and taps Add; a code they scanned or pasted here needs no extra tap.
async function pair(text, fromLink) {
  if (S.pairing) return;
  const m = text.match(/omnillm[123]-[A-Za-z0-9_-]+/);
  if (!m) { $("welcome-err").textContent = t("notCode"); show("welcome"); return; }
  text = m[0];
  S.pairing = true;
  welcomeNote("");
  try {
    show("pairing");
    $("pair-err").textContent = ""; $("pair-retry").hidden = true; sasShown(false); $("pair-status").textContent = "";
    try { await omnillm.check(text); } catch (e) { quoted($("pair-err"), errText(e, text)); $("pair-retry").hidden = false; return; }
    if (fromLink || S.device) {
      $("pair-host").textContent = t("pairVia", relayHost(text));
      $("pair-replace").hidden = !S.device;
      if (S.device) $("pair-replace").textContent = t("pairReplace", S.device.name);
      $("pair-ask").hidden = false;
      const go = await new Promise((res) => {
        $("pair-go").onclick = () => res(true);
        $("pair-cancel").onclick = () => res(false);
      });
      $("pair-ask").hidden = true;
      if (!go) { show(S.device ? "chat" : "welcome"); return; }
    }
    await connectPair(text);
  } finally { S.pairing = false; }
}

// relayURL reads the relay address out of a pairing text (both formats:
// omnillm1- is base64url JSON, omnillm2- is binary with the URL after 116
// bytes, omnillm3- the same after a 4-byte checksum). Display only: the device code checks the text for real.
function relayURL(code) {
  try {
    const raw = atob(code.slice(9).replace(/-/g, "+").replace(/_/g, "/"));
    if (code.startsWith("omnillm2-")) return new TextDecoder().decode(Uint8Array.from(raw.slice(116), (c) => c.charCodeAt(0)));
    if (code.startsWith("omnillm3-")) return new TextDecoder().decode(Uint8Array.from(raw.slice(120), (c) => c.charCodeAt(0))); // 4-byte checksum first
    return JSON.parse(raw).r;
  } catch { return ""; }
}
// relayHost is the host alone (no scheme, no port), except on this
// computer itself, where the port is what tells relays apart.
function relayHost(code) {
  try {
    const u = new URL(relayURL(code));
    return /^(127\.|localhost$|\[::1\]$)/.test(u.hostname) ? u.host : u.hostname;
  } catch { return "?"; }
}

// A phone paired again first leaves its old pairing on that Mac, so the
// Mac doesn't keep a dead device under the same name. Its chats stay on
// the phone. S.leftOld says the old Mac really let it go: only then is the
// old pairing gone if the new one fails; otherwise (the old Mac away) it
// stays in use.
async function connectPair(text) {
  $("pair-status").textContent = t("pairConnecting");
  $("pair-stop").hidden = false;
  S.pairStopped = false; S.leftOld = false;
  const old = S.device;
  let closed = false;
  try {
    await omnillm.check(text);
    if (old && !S.pairStopped) {
      closed = true;
      try { omnillm.close(); } catch {}
      S.openedState = null;
      try { await omnillm.leave(old.state); S.leftOld = true; } catch {}
    }
    if (S.pairStopped) throw new Error("cancelled");
    const name = deviceName();
    const r = await omnillm.pair(text, name, (code) => {
      $("pair-stop").hidden = true;
      quoted($("pair-status"), t("pairCheck"));
      $("sas").textContent = code.slice(0, 3) + " " + code.slice(3);
      sasShown(true);
    });
    // The Mac names the device (another iPhone there: "iPhone (2)").
    S.device = { state: r.state, name: r.name || name, added: Date.now() };
    await seal("device", S.device);
    $("pair-status").textContent = t("pairDone");
    sasShown(false); $("pair-stop").hidden = true;
    // A fresh, empty chat says once that the Mac is there.
    S.cur = newChat(); S.hello = S.cur.id;
    setTimeout(openChat, 600);
  } catch (e) {
    const dropped = S.sasUp && e && e.code === "hub_offline";
    sasShown(false); $("pair-stop").hidden = true;
    $("pair-status").textContent = "";
    if (S.leftOld && S.device === old) {
      S.device = null; await idb.del("device").catch(() => {});
      if (S.pairStopped) show("welcome"); // Cancel came while it was leaving
    }
    // The old pairing is still good: open it again (it was closed to leave).
    else if (old && closed && S.device === old) connect();
    if (S.pairStopped) return; // cancelled here; already back where it started
    quoted($("pair-err"), dropped ? t("pairDropped") : errText(e, text));
    $("pair-retry").hidden = false;
  }
}

function errText(e, code) {
  if (e && e.code) {
    if (code && e.code === "hub_offline" && relayIsPlain(code)) return t("needTLS");
    return e.message;
  }
  return String(e && e.message || e);
}
function relayIsPlain(code) {
  const r = relayURL(code);
  return /^ws:\/\//.test(r) && !/^ws:\/\/(127\.0\.0\.1|localhost)[:/]/.test(r);
}

// ---------- QR scanning (camera in the page; used from the Home Screen) ----------
let camStream = null, camTimer = null;
$("scan").addEventListener("click", startScan);
$("scan-stop").addEventListener("click", stopScan);
let jsqrLoaded = null;
function loadJsQR() {
  jsqrLoaded ??= new Promise((res, rej) => { const s = document.createElement("script"); s.src = "vendor/jsqr.js"; s.onload = res; s.onerror = rej; document.body.append(s); });
  return jsqrLoaded;
}
async function startScan() {
  $("welcome-err").textContent = "";
  await loadJsQR().catch(() => {});
  try {
    camStream = await navigator.mediaDevices.getUserMedia({ video: { facingMode: "environment" }, audio: false });
  } catch { $("welcome-err").textContent = t("camDenied"); $("paste").open = true; return; }
  show("scanner");
  const v = $("cam"); v.srcObject = camStream; await v.play().catch(() => {});
  const c = $("cam-canvas"), g = c.getContext("2d", { willReadFrequently: true });
  camTimer = setInterval(() => {
    if (!v.videoWidth) return;
    const w = Math.min(v.videoWidth, 720), h = Math.round(v.videoHeight * w / v.videoWidth);
    c.width = w; c.height = h; g.drawImage(v, 0, 0, w, h);
    const r = jsQR(g.getImageData(0, 0, w, h).data, w, h, { inversionAttempts: "dontInvert" });
    if (r && r.data) {
      const m = r.data.match(/omnillm[123]-[A-Za-z0-9_-]+/);
      stopScan(!!m);
      if (m) pair(m[0]); else { $("welcome-err").textContent = t("notCode"); }
    }
  }, 250);
}
function stopScan(keep) {
  clearInterval(camTimer); camTimer = null;
  if (camStream) camStream.getTracks().forEach((tr) => tr.stop());
  camStream = null;
  if (keep !== true) show(S.device ? "chat" : "welcome");
}

// ---------- chat ----------
async function openChat() {
  if (S.volatile && !S.volatileSaid) { S.volatileSaid = true; setTimeout(() => { const n = el("p", t("volatile"), "note warn-note"); $("log").prepend(n); setTimeout(() => n.remove(), 15000); }, 500); }
  show("chat");
  $("input").placeholder = t("placeholder");
  $("device-line").textContent = t("deviceLine", S.device.name);
  $("install-tip").hidden = isStandalone;
  if (!S.cur) S.cur = S.chats[0] || newChat();
  try { const d = sessionStorage.getItem("omnillm.draft"); if (d) { $("input").value = d; sessionStorage.removeItem("omnillm.draft"); grow(); } } catch {}
  renderList(); renderLog(); sendState();
  await connect();
}

// connect opens the device and loads the models; S.connecting is that
// work, so a message sent meanwhile waits for it instead of going nowhere.
function connect() {
  S.connecting = (async () => {
    if (!navigator.onLine) { setOnline(false, t("noNetwork")); return; }
    const sel = $("model");
    if (!S.models.length) { sel.textContent = ""; const o = el("option", t("loadingModels")); o.value = ""; sel.append(o); }
    liveDot("wait");
    S.dialing = true;
    try {
      // Open once per pairing: reopening closes the old device, and with it
      // a reply still on its way (the network coming back fires "online"
      // mid-reply). An open device reconnects by itself.
      if (S.openedState !== S.device.state) { await omnillm.open(S.device.state); S.openedState = S.device.state; }
      await loadModels();
      setOnline(true);
    } catch (e) {
      if (e && e.msg === "device_revoked") return unpaired("removedHere");
      if (e && e.code === "device_paused") return setPaused(errText(e));
      setOnline(false, errText(e));
    } finally { S.dialing = false; }
  })();
  return S.connecting;
}

async function recheck() {
  if (!S.device || S.quiet || S.busy || document.hidden || !navigator.onLine) return;
  try { await loadModels(); setOnline(true); } catch (e) {
    if (e && e.msg === "device_revoked") return unpaired("removedHere");
    if (e && e.code === "device_paused") return setPaused(errText(e));
    if (e && e.code === "hub_offline") setOnline(false);
  }
}
// Back in front: the list again, so versions found meanwhile show. A page
// restored from the browser's back-forward cache counts too.
document.addEventListener("visibilitychange", () => { if (!document.hidden) recheck(); });
addEventListener("pageshow", (e) => { if (e.persisted) recheck(); });
setInterval(recheck, 120000);

// isSub: a Claude or ChatGPT subscription (the official programs on the
// hub), not merely a model kept for people (a local model can be that too).
function isSub(m) { const k = m.kind || m.provider; return k === "claude-code" || k === "codex"; }
// isLocal: runs on the Mac itself. The hub tells a device every model is
// remote (the data leaves the phone), so this goes by the provider: local
// models come through Ollama.
function isLocal(m) { return !!m.onHost; } // runs on the Mac itself (the hub says so)
function group(m) { return isSub(m) ? 0 : isLocal(m) ? 2 : 1; }
function modelName(id) { const m = S.models.find((x) => x.id === id); return m ? m.name : id; }
// label: the name in this page's language. The Mac sends the name without
// language-built parts and says what to add: "latest" (the version isn't
// known yet) and the kind (Codex). A name the owner chose stays as it is.
function label(m) {
  if (m.renamed) return m.name;
  if (m.kind === "codex") return m.version ? t("codexV", m.version) : t("codex");
  return m.latest ? m.name + t("latest") : m.name;
}
// family orders Claude's models Opus, Sonnet, Haiku (largest first).
function family(m) { const s = (m.id + " " + m.name).toLowerCase(), i = ["opus", "sonnet", "haiku"].findIndex((f) => s.includes(f)); return i < 0 ? 3 : i; }

async function loadModels() {
  const ms = await omnillm.models();
  syncName();
  for (const m of ms) m.name = label(m);
  // Subscriptions first (the owner pays for them already), then API keys,
  // then models on the Mac; the picker shows names only, grouped.
  ms.sort((a, b) => (group(a) - group(b)) || (group(a) === 0 && family(a) - family(b)) || a.name.localeCompare(b.name));
  S.models = ms; S.listed = true;
  const sel = $("model"); sel.textContent = "";
  let og = null, g = -1;
  for (const m of ms) {
    if (group(m) !== g) { g = group(m); og = el("optgroup"); og.label = t(["grpSub", "grpKey", "grpLocal"][g]); sel.append(og); }
    const o = el("option", m.name); o.value = m.id; og.append(o);
  }
  if (!ms.length) { const o = el("option", t("noModelsOpt")); o.value = ""; o.disabled = true; sel.append(o); sel.value = ""; return; }
  if (pickModel()) renderLog(true);
  if (S.cur) renderList(); // the drawer names models by name now
}
// syncName keeps this device's name as the Mac has it (the owner may rename
// it there; the hub sends it with the models): the drawer line and the
// "replace this pairing" question use it.
function syncName() {
  const n = omnillm.name && omnillm.name();
  if (!n || !S.device || S.openedState !== S.device.state || n === S.device.name) return;
  S.device.name = n;
  seal("device", S.device).catch(() => {});
  $("device-line").textContent = t("deviceLine", n);
}
// pickModel selects the current chat's model, else the last one used, else
// the Claude subscription's Sonnet, else the first. A chat whose model is
// gone from the Mac moves to the selected one and says so (true).
function pickModel() {
  const ms = S.models, sel = $("model");
  if (!ms.length) return false;
  const want = S.cur && S.cur.model;
  sel.value = [want, localStorageGet("model"), "claude-code/sonnet"].find((id) => id && ms.some((m) => m.id === id)) || ms[0].id;
  if (!want || want === sel.value) return false;
  if (S.cur.messages.length) S.cur.gone = t("modelGone", modelName(sel.value));
  S.cur.model = sel.value;
  return true;
}
$("model").addEventListener("change", () => { localStorageSet("model", $("model").value); if (S.cur) S.cur.model = $("model").value; refreshActs(); });
function localStorageGet(k) { try { return localStorage.getItem("omnillm." + k); } catch { return null; } }
function localStorageSet(k, v) { try { localStorage.setItem("omnillm." + k, v); } catch {} }

// liveDot: green when the Mac answered (even with no models: it is
// there), amber while the Mac has paused this device, grey when it can't
// be reached, pulsing grey while connecting.
function liveDot(state) {
  const d = $("live-dot"); d.className = "live-dot " + state;
  const label = t({ on: "liveOn", off: "liveOff", paused: "pausedDot" }[state] || "liveWait");
  d.title = label; d.setAttribute("aria-label", label);
}
// Any answer from the Mac other than "paused" ends a pause.
function setOnline(ok, why) {
  S.online = ok;
  if (ok) { S.kept = false; S.paused = null; } else S.why = why || t("offline");
  paintBanner(); sendState(); refreshActs();
}
// setPaused: the Mac answered but has paused this device (its sentence
// says so). The typed text stays; Send rests until a check gets through.
function setPaused(why) {
  S.online = true; S.paused = why;
  // Paused before the list ever came: the picker says so, not "loading".
  if (!S.models.length) { const sel = $("model"); sel.textContent = ""; const o = el("option", t("paused")); o.value = ""; sel.append(o); }
  paintBanner(); sendState(); refreshActs();
}
// paintBanner: paused, offline (why, and that the text is kept if a send
// was blocked), or a Mac with no models; Retry only when this device has
// a network (without one it can't help; the page retries by itself).
function paintBanner() {
  liveDot(!S.online ? "off" : S.paused ? "paused" : "on");
  const b = $("banner"), none = S.online && !S.paused && S.listed && !S.models.length;
  b.textContent = ""; b.hidden = S.online && !none && !S.paused;
  if (S.paused && S.online) { b.append(quoted(el("span"), S.paused)); return; }
  if (none) { b.append(quoted(el("span"), t("noModels"))); return; }
  if (S.online) return;
  b.append(quoted(el("span"), S.why + (S.kept ? " " + t("kept") : "")));
  if (navigator.onLine) { const r = el("button", t("retry"), "link"); r.type = "button"; r.addEventListener("click", connect); b.append(r); }
}
// The phone's own network: say so at once, reconnect when it is back
// (the old connection may be dead without anyone noticing).
addEventListener("offline", () => { if (S.device) setOnline(false, t("noNetwork")); });
addEventListener("online", () => { if (S.device && !S.quiet) connect(); });
document.addEventListener("visibilitychange", () => { if (!document.hidden && !S.online && S.device && !S.quiet) connect(); });
// While offline (no network, or the Mac not answering), try again every
// 10 s as long as the page is in view.
// Paused: look again just as often (the models call); nothing is resent,
// the person taps Send again once it clears.
setInterval(() => {
  if (!S.device || S.quiet || S.busy || S.dialing || document.hidden || $("chat").hidden) return;
  if (!S.online) connect(); else if (S.paused) recheck();
}, 10000);

function newChat() {
  const c = { id: crypto.randomUUID(), title: "", model: $("model").value || localStorageGet("model") || "", messages: [], updated: Date.now() };
  return c;
}
$("new").addEventListener("click", () => {
  if (S.busy) return toastText(t("waitReply"));
  S.cur = newChat(); renderLog(); renderList(); closeDrawer(); $("input").focus();
});

// ---------- the log ----------
// It follows a reply only while the person is at the bottom (within 80 px);
// scrolled up to read, it stays put and a pill offers the new words.
function nearBottom() { const l = $("log"); return l.scrollHeight - l.scrollTop - l.clientHeight < 80; }
function toBottom() { const l = $("log"); l.scrollTop = l.scrollHeight; $("jump").hidden = true; }
$("jump").textContent = t("newContent");
$("jump").addEventListener("click", () => { $("log").scrollTo({ top: $("log").scrollHeight, behavior: "smooth" }); $("jump").hidden = true; });
$("log").addEventListener("scroll", () => { if (nearBottom()) $("jump").hidden = true; }, { passive: true });

// renderLog draws the chat; keep leaves the scroll where it is when the
// person had scrolled up.
function renderLog(keep) {
  const log = $("log"), stay = keep && !nearBottom(), top = log.scrollTop;
  log.textContent = "";
  const msgs = S.cur.messages;
  // Just paired: the first empty chat says once that the Mac is there.
  if (!msgs.length && S.hello === S.cur.id) log.append(switchLine(t("connectedMac")));
  if (!msgs.length) log.append(el("p", t("empty"), "empty"));
  // A line where the chat changes model, so who answered what stays clear.
  // Compared by model id (the name gains its version once known). An
  // earlier answer (kept by "answer again") folds above the new one and
  // doesn't count.
  let last = "", userEl = null;
  msgs.forEach((m, i) => {
    if (m.prev) { log.append(earlier(m)); return; }
    if (m.role === "assistant" && m.by) {
      const key = m.model || m.by;
      if (last && key !== last) {
        const line = switchLine(t("switchedTo", m.by));
        if (userEl && !(msgs[i - 1] && msgs[i - 1].prev)) userEl.before(line); else log.append(line);
      }
      last = key;
    }
    const b = bubble(m, i === msgs.length - 1);
    if (m.role === "user") userEl = b;
    log.append(b);
    if (S.cur.long === i + 1) log.append(switchLine(t("longChat")));
  });
  const row = actsRow();
  if (row) log.append(row);
  if (S.cur.gone) log.append(switchLine(S.cur.gone));
  if (stay) log.scrollTop = top; else toBottom();
}
// actsRow: under the last turn, once it is over: copy, and answer again
// (with the model now selected, so it doubles as "ask another model"); a
// reply that failed or was stopped, or a question left without one,
// offers Retry. Answering needs the Mac: offline or paused, it rests.
function actsRow() {
  const msgs = S.cur ? S.cur.messages : [], end = msgs[msgs.length - 1];
  if (!end || end.live || S.busy || !(end.role === "user" || end.content || end.note)) return null;
  const row = el("div", undefined, "acts");
  if (end.role === "assistant" && end.content) row.append(copyButton(end, "act copy-act"));
  const retry = end.role === "user" || end.failed, pick = $("model").value;
  const label = retry ? t("retry") : pick && end.model && pick !== end.model ? t("againWith", modelName(pick)) : t("again");
  const again = el("button", label, "act again-act"); again.type = "button";
  again.disabled = !S.online || !!S.paused;
  again.addEventListener("click", answerAgain);
  row.append(again);
  return row;
}
// refreshActs redraws only that row (the picker or the connection changed).
function refreshActs() {
  const old = $("log").querySelector(":scope > .acts");
  if (!old || !S.cur) return;
  const row = actsRow();
  if (row) old.replaceWith(row); else old.remove();
}
// earlier: a reply replaced by "answer again", folded to one line; it is
// kept for the person, never sent to the model again.
function earlier(m) {
  const box = el("div", undefined, "earlier");
  const open = S.opened && S.opened.has(m);
  const tg = el("button", t(open ? "earlierOpen" : "earlier", m.by || modelName(m.model)), "earlier-toggle"); tg.type = "button";
  tg.setAttribute("aria-expanded", String(!!open));
  tg.addEventListener("click", () => {
    S.opened ??= new WeakSet();
    if (S.opened.has(m)) S.opened.delete(m); else S.opened.add(m);
    box.replaceWith(earlier(m));
  });
  box.append(tg);
  if (open) box.append(bubble(m, false));
  return box;
}
function switchLine(text) { const line = el("div", undefined, "switch-line"); line.append(el("span", text)); return line; }
function copyButton(m, cls) {
  const cp = el("button", t("copy"), cls); cp.type = "button";
  cp.addEventListener("click", () => { navigator.clipboard.writeText(m.content).then(() => { cp.textContent = t("copied"); setTimeout(() => { cp.textContent = t("copy"); }, 1200); }); });
  return cp;
}

// bubble renders a message; fenced code becomes <pre>, the rest stays text.
// A reply says first which model is writing it; until its first words
// arrive it shows "connecting to your Mac", then that model thinking (an
// orbiting core, as in the logo). The last reply's Copy sits in the row
// under it instead.
function bubble(m, isLast) {
  const b = el("div", undefined, "msg " + m.role + (isLast ? " last" : ""));
  if (m.role === "assistant" && m.by) {
    const who = el("div", undefined, "who");
    who.append(el("span", "", "who-dot"), el("span", m.by, "by"));
    b.append(who);
  }
  if (m.images && m.images.length) {
    const pics = el("div", undefined, "pics");
    for (const im of m.images) { const i = el("img"); i.alt = t("imageTitle"); i.src = "data:" + im.mime + ";base64," + im.data; pics.append(i); }
    b.append(pics);
  }
  const body = el("div", undefined, "body");
  b.append(body);
  if (m.live && !m.content) body.append(thinking(m));
  else fill(body, m.live ? holdBack(m.content) : m.content);
  if (m.live && m.content) {
    let last = body.lastElementChild;
    if (last && (last.tagName === "UL" || last.tagName === "OL")) last = last.lastElementChild;
    (last && (last.tagName === "P" || last.tagName === "LI") ? last : body).append(el("span", "", "caret"));
  }
  if (m.note) b.append(quoted(el("p", undefined, "note"), m.note));
  if (m.role === "assistant" && m.content && !m.live && !isLast) b.append(copyButton(m, "copy"));
  return b;
}
// holdBack: while words arrive, a line that so far is only a list marker
// ("- ", "* ", "1. ") waits for its next character, so no bare "-" flashes
// before it becomes a bullet. Inside an open code block nothing waits.
// The same goes for a mark that opens **bold**, *italic*, __bold__ or
// `code` on the last line before its closing half arrives: it waits, so
// "**Set" shows as "Set", not as stars.
const PAIRED = /\*\*[^*]+\*\*|__[^_]+__|`[^`]+`|\*[^*\s][^*]*?\*|_[^_\s][^_]*?_/g;
function holdBack(text) {
  if ((text.match(/```/g) || []).length % 2) return text;
  text = text.replace(/(^|\n)[ \t]*([-*+]|\d+[.)]?)[ \t]*$/, "$1");
  const at = text.lastIndexOf("\n") + 1;
  // Marks already closed are blanked out; what is left last is open.
  const line = text.slice(at).replace(PAIRED, (s) => " ".repeat(s.length));
  const m = line.match(/(\*+|_+|`+)[^*_`]*$/);
  if (!m) return text;
  const i = m.index, mark = m[1], next = line[i + mark.length];
  // An opener is followed by a word (or nothing yet); a lone * or _ also
  // starts a word ("2*3", "snake_case" and "a * b" stay as they are).
  const opens = next === undefined || !/\s/.test(next);
  const starts = mark[0] === "`" || mark.length > 1 || i === 0 || !/\w/.test(line[i - 1]);
  return opens && starts ? text.slice(0, at + i) + text.slice(at + i + mark.length) : text;
}
function thinking(m) {
  const box = el("div", undefined, "thinking-box");
  const w = el("div", undefined, "thinking");
  const orb = el("span", "", "orb"); orb.setAttribute("aria-hidden", "true");
  const secs = el("span", "", "waited");
  // Until the Mac has the request, say that; then the model thinks.
  w.append(orb, el("span", t(m.acked ? "thinking" : "pairConnecting"), "think-text"));
  if (m.acked) {
    const dots = el("span", undefined, "dots"); dots.setAttribute("aria-hidden", "true");
    dots.append(el("i"), el("i"), el("i"));
    w.append(dots);
  }
  w.append(secs);
  w.setAttribute("role", "status");
  // Past a minute, say what can be done about it.
  const slow = el("p", t("slow"), "slow"); slow.hidden = true;
  box.append(w, slow);
  const t0 = S.sentAt || Date.now();
  const tick = () => {
    if (!box.isConnected) return;
    const s = Math.floor((Date.now() - t0) / 1000);
    if (s >= 5) secs.textContent = t("waited", s);
    if (s >= 60) slow.hidden = false;
    setTimeout(tick, 1000);
  };
  setTimeout(tick, 1000);
  return box;
}
function fill(b, text) {
  const parts = text.split(/```/);
  parts.forEach((p, i) => {
    if (i % 2) { const pre = el("pre"); pre.append(el("code", p.replace(/^[\w+-]*\n/, "").replace(/\n+$/, ""))); b.append(pre); }
    else if (p.trim()) prose(b, p.replace(/^\n+|\n+$/g, ""));
  });
}
// prose shows the light Markdown models write (headings, **bold**, `code`,
// lists) as formatting: paragraphs split at blank lines, list lines become
// real <ul>/<ol>. Everything stays text nodes: nothing a model writes is
// ever parsed as HTML.
const LIST_ITEM = /^(\s*)([-*+]|\d+[.)])\s+(.*)$/;
function prose(b, text) {
  let p = null, list = null;
  for (const line of text.split("\n")) {
    if (!line.trim()) { p = null; list = null; continue; }
    const h = line.match(/^#{1,6}\s+(.*)$/);
    if (h) { p = null; list = null; const hp = el("p"); inline(hp, h[1], true); b.append(hp); continue; }
    const li = line.match(LIST_ITEM);
    if (li) {
      const tag = /\d/.test(li[2]) ? "ol" : "ul";
      if (!list || list.tagName.toLowerCase() !== tag) {
        list = el(tag);
        const n = parseInt(li[2], 10);
        if (tag === "ol" && n > 1) list.start = n;
        b.append(list);
      }
      p = null;
      const item = el("li"); inline(item, li[3], false); list.append(item);
      continue;
    }
    // An indented line under a list item continues that item.
    if (list && /^\s/.test(line)) { const item = list.lastElementChild; item.append("\n"); inline(item, line.trim(), false); continue; }
    list = null;
    if (!p) { p = el("p"); b.append(p); } else p.append("\n");
    inline(p, line, false);
  }
}
function inline(p, line, strong) {
  // **bold**, __bold__, `code`, and *italic* that isn't part of a word
  // (no lookbehind: older iPhones' Safari lacks it).
  const re = /\*\*([^*]+)\*\*|__([^_]+)__|`([^`]+)`|(^|[^*\w])\*([^*\s][^*]*?)\*(?![*\w])/g;
  let last = 0, m;
  const put = (s, tag) => { if (!s) return; if (tag) p.append(el(tag, s)); else if (strong) p.append(el("strong", s)); else p.append(s); };
  while ((m = re.exec(line))) {
    const lead = m[4] || "";
    put(line.slice(last, m.index) + lead);
    if (m[3] !== undefined) put(m[3], "code"); else if (m[5] !== undefined) put(m[5], "em"); else put(m[1] || m[2], "strong");
    last = re.lastIndex;
  }
  put(line.slice(last));
}

// ---------- drawer list ----------
// Each chat: its title, then who answered last and when.
function when(ts) {
  const d = new Date(ts), now = new Date(), loc = lang === "zh" ? "zh-CN" : "en-US";
  if (d.toDateString() === now.toDateString()) return t("today") + " " + d.toLocaleTimeString(loc, { hour: "numeric", minute: "2-digit" });
  return d.toLocaleDateString(loc, d.getFullYear() === now.getFullYear() ? { month: "short", day: "numeric" } : { year: "numeric", month: "short", day: "numeric" });
}
function chatMeta(c) {
  const by = [...c.messages].reverse().find((m) => m.by);
  return [by ? by.by : c.model ? modelName(c.model) : "", when(c.updated)].filter(Boolean).join(" · ");
}
// The search field filters by title and message text, here on the device.
function matches(c, q) {
  return !q || (c.title || "").toLowerCase().includes(q) || c.messages.some((m) => (m.content || "").toLowerCase().includes(q));
}
function renderList() {
  const ul = $("chat-list"); ul.textContent = "";
  const q = $("chat-search").value.trim().toLowerCase();
  const all = [S.cur, ...S.chats.filter((c) => c.id !== S.cur.id)].filter((c) => (c.messages.length || c === S.cur) && matches(c, q));
  if (!all.length) ul.append(el("li", t("noMatch"), "muted small no-match"));
  for (const c of all) {
    const li = el("li"); const a = el("button", undefined, c.id === S.cur.id ? "cur" : ""); a.type = "button";
    a.append(el("span", c.title || t("newChat"), "title"));
    if (c.messages.length) a.append(el("span", chatMeta(c), "meta"));
    a.addEventListener("click", () => {
      if (S.busy && c !== S.cur) { closeDrawer(); return toastText(t("waitReply")); }
      S.cur = c; if (S.models.length) pickModel(); renderLog(); renderList(); closeDrawer();
    });
    li.append(a);
    if (c.messages.length) {
      const x = el("button", "", "chat-del"); x.type = "button"; x.setAttribute("aria-label", t("deleteChat")); x.title = t("deleteChat");
      x.addEventListener("click", async () => {
        if (S.busy && c === S.cur) return toastText(t("waitReply"));
        if (!await ask(t("deleteChatQ", c.title || t("newChat")), t("delete"), true)) return;
        S.chats = S.chats.filter((o) => o.id !== c.id);
        if (c === S.cur) { S.cur = S.chats[0] || newChat(); renderLog(); }
        if (!S.quiet) await seal("chats", S.chats).catch(() => {});
        renderList();
      });
      li.append(x);
    }
    ul.append(li);
  }
}

async function saveChats() {
  if (S.quiet) return; // another tab is in use and saves its own
  S.cur.updated = Date.now();
  const rest = S.chats.filter((c) => c.id !== S.cur.id);
  S.chats = [S.cur, ...rest].filter((c) => c.messages.length).slice(0, 200);
  await seal("chats", S.chats).catch(() => {});
}

// ---------- sending ----------
const input = $("input");
input.addEventListener("input", () => { grow(); sendState(); });
// The send button looks pressable only when there is something to send;
// while the Mac has this device paused it rests (Stop still works).
function sendState() {
  $("send").classList.toggle("idle", !S.busy && (!!S.paused || (!input.value.trim() && !(S.pending && S.pending.length))));
  $("send").disabled = !S.busy && !!S.paused;
}
function grow() { input.style.height = "auto"; input.style.height = Math.min(input.scrollHeight, 180) + "px"; }
input.addEventListener("keydown", (e) => {
  // Enter sends on a computer keyboard; on phones the send button does.
  // While a reply streams, Enter only says so: Stop is the one way to cancel.
  // An IME's Enter picks a word, not sends (Safari says so only by keyCode 229).
  if (e.key === "Enter" && !e.shiftKey && !e.isComposing && e.keyCode !== 229 && matchMedia("(pointer: fine)").matches) {
    e.preventDefault();
    if (S.busy) return toastText(t("waitReply"));
    $("composer").requestSubmit();
  }
});
// chosenModel waits for the models if they are still loading; "" when
// there is none (the banner says why).
async function chosenModel() {
  if (!$("model").value && S.connecting) { // still loading the models
    if (S.waiting) return "";
    S.waiting = true;
    try { await S.connecting; } finally { S.waiting = false; }
  }
  if (!$("model").value) paintBanner();
  return $("model").value;
}
$("composer").addEventListener("submit", async (e) => {
  e.preventDefault();
  if (S.busy) { omnillm.cancel(S.busy); return; }
  const text = input.value.trim();
  const images = S.pending.map(({ mime, data }) => ({ mime, data }));
  if (!text && !images.length) return;
  if (S.paused) return; // the banner says why; the text stays
  if (!navigator.onLine) { S.kept = true; setOnline(false, t("noNetwork")); return; } // keep the text
  const model = await chosenModel();
  if (!model || S.busy) return;
  input.value = ""; grow(); clearPending();
  const chat = S.cur; // this reply belongs to this conversation, whatever happens meanwhile
  if (!chat.title) chat.title = text.slice(0, 40) || t("imageTitle");
  chat.messages.push(images.length ? { role: "user", content: text, images } : { role: "user", content: text });
  await run(chat, model, { text, images });
});
// answerAgain asks the same question again with the model selected now.
// A finished reply is kept, folded above the new one (and not sent to the
// model again); a failed or stopped one is dropped (Retry).
async function answerAgain() {
  if (S.busy) return toastText(t("waitReply"));
  if (!S.online || S.paused) return;
  if (!navigator.onLine) { setOnline(false, t("noNetwork")); return; }
  const model = await chosenModel();
  const chat = S.cur, ms = chat.messages;
  if (!model || S.busy) return;
  let old = null;
  const end = ms[ms.length - 1];
  if (end && end.role === "assistant") {
    if (end.failed || !end.content) ms.pop(); else { end.prev = true; old = end; }
  }
  const asked = [...ms].reverse().find((m) => !m.prev);
  if (!asked || asked.role !== "user") { if (old) delete old.prev; return; }
  await run(chat, model, null, old);
}

// Past this a chat says once that a new one is faster (the whole chat goes
// out with every message).
const LONG_TURNS = 40, LONG_CHARS = 80000;
// run asks for a reply to the chat as it stands. fresh is what was just
// typed: if nothing came back it goes back into the box, joined with what
// is there now. old is the reply "answer again" folded away: if the new
// one brings nothing, it comes back.
async function run(chat, model, fresh, old) {
  chat.model = model; delete chat.gone;
  const sent = chat.messages.filter((m) => !m.prev);
  if (chat.long == null && (sent.length > LONG_TURNS || sent.reduce((n, m) => n + (m.content || "").length, 0) > LONG_CHARS)) {
    chat.long = chat.messages.length + 1; // the line goes under this reply
  }
  if (S.hello === chat.id) S.hello = null;
  const reply = { role: "assistant", content: "", by: modelName(model), model, live: true };
  S.sentAt = Date.now();
  chat.messages.push(reply);
  renderLog(); renderList();
  const log = $("log"); let node = log.querySelector(":scope > .msg.last");
  const id = crypto.randomUUID(); S.busy = id; busyUI(true);
  // Images go with the latest two turns that have them; older ones are
  // named in the text, so a long chat with pictures keeps working. Earlier
  // answers stay on the phone.
  const ask = chat.messages.slice(0, -1).filter((m) => !m.prev);
  const withImages = new Set(ask.filter((m) => m.images && m.images.length).slice(-2));
  const msgs = ask.map((m) => {
    if (!m.images || !m.images.length) return { role: m.role, content: m.content };
    if (withImages.has(m)) return { role: m.role, content: m.content, images: m.images };
    return { role: m.role, content: (m.content ? m.content + "\n" : "") + "[" + m.images.length + " earlier image(s)]" };
  });
  let raf = 0;
  const flush = () => {
    raf = 0;
    const stick = nearBottom();
    const nb = bubble(reply, true); node.replaceWith(nb); node = nb;
    if (stick) toBottom(); else if (reply.content) $("jump").hidden = false;
  };
  // quiet: the error is said elsewhere (the banner, or the "model
  // removed" line), not again as a red note.
  let quiet = false;
  try {
    const r = await omnillm.chat(id, model, JSON.stringify(msgs), (d, exact, acked) => {
      if (acked) reply.acked = true; // the Mac has it: thinking from here
      if (exact) { reply.by = exact; S.versionSeen = true; } // "Claude Opus 5.5", as the hub reports it
      reply.content += d || ""; if (!raf) raf = requestAnimationFrame(flush);
    });
    if (r.finish === "length") reply.note = t("cut");
    setOnline(true);
  } catch (err) {
    if (err && err.msg === "device_revoked") return unpaired("removedHere");
    const code = err && err.code;
    reply.failed = true;
    if (code === "cancelled") reply.note = t("stopped");
    else if (code === "hub_offline") {
      // Nothing came: the banner says it. Cut off midway: the bubble says so.
      if (reply.content) reply.note = t("dropped"); else quiet = true;
      S.kept = !!fresh && !reply.content; setOnline(false);
    } else if (code === "device_paused") {
      quiet = !reply.content; reply.note = errText(err);
      setPaused(errText(err));
    } else {
      reply.note = errText(err);
      // The model left the Mac (or this device may no longer use it): load
      // the list again, which switches the chat and says so.
      if (err && (err.msg === "unknown_model" || err.msg === "model_not_allowed" || code === "model_not_allowed")) {
        await loadModels().catch(() => {});
        if (chat.gone) quiet = true;
      }
    }
    if (!reply.content) {
      chat.messages.pop();
      if (old) delete old.prev;
      if (fresh && code !== "cancelled") {
        chat.messages.pop();
        // Back into the box, before whatever was typed meanwhile.
        const now = input.value;
        input.value = now.trim() ? fresh.text + "\n" + now : fresh.text; grow();
        S.pending.push(...fresh.images); renderPending();
      }
    }
  } finally {
    delete reply.live;
    S.busy = null; busyUI(false);
    setTimeout(reloadIfIdle, 1500); // a new version waited for this reply
    // The picker learns the version too (the hub now knows it).
    if (S.versionSeen) { S.versionSeen = false; loadModels().catch(() => {}); }
    // Removed meanwhile (unpaired): nothing left to show or save.
    if (S.device) {
      const near = nearBottom();
      renderLog(true);
      if (reply.note && !reply.content && !quiet) { $("log").append(quoted(el("p", undefined, "note err"), reply.note)); if (near) toBottom(); }
      await saveChats();
    }
  }
}
// ---------- images ----------
S.pending = [];
$("attach").setAttribute("aria-label", t("addImage"));
$("attach").addEventListener("click", () => $("file").click());
$("file").addEventListener("change", async (e) => { await addImages([...e.target.files]); e.target.value = ""; });
input.addEventListener("paste", (e) => {
  const files = [...(e.clipboardData && e.clipboardData.files || [])].filter((f) => f.type.startsWith("image/"));
  if (files.length) { e.preventDefault(); addImages(files); }
});

async function addImages(files) {
  for (const f of files) {
    if (S.pending.length >= 4) { toastText(t("tooManyImages")); break; }
    try { S.pending.push(await shrink(f)); } catch (e) { toastText(t(e && e.message === "too big" ? "imageTooBig" : "imageFailed")); }
  }
  renderPending();
}

// shrink makes a photo small enough to send quickly: the long side at most
// 1600 px, JPEG, under 2 MB. A small GIF stays as it is (it may move).
async function shrink(file) {
  if (file.type === "image/gif" && file.size <= 1.5e6) return { mime: "image/gif", data: await b64(file) };
  const url = URL.createObjectURL(file);
  try {
    const img = await new Promise((res, rej) => { const i = new Image(); i.onload = () => res(i); i.onerror = rej; i.src = url; });
    const scale = Math.min(1, 1600 / Math.max(img.naturalWidth, img.naturalHeight));
    const c = document.createElement("canvas");
    c.width = Math.max(1, Math.round(img.naturalWidth * scale)); c.height = Math.max(1, Math.round(img.naturalHeight * scale));
    const g = c.getContext("2d"); g.fillStyle = "#fff"; g.fillRect(0, 0, c.width, c.height); g.drawImage(img, 0, 0, c.width, c.height);
    for (const q of [0.85, 0.7, 0.55]) {
      const blob = await new Promise((res) => c.toBlob(res, "image/jpeg", q));
      if (blob && blob.size <= 1.9e6) return { mime: "image/jpeg", data: await b64(blob) };
    }
    throw new Error("too big");
  } finally { URL.revokeObjectURL(url); }
}
function b64(blob) {
  return new Promise((res, rej) => { const r = new FileReader(); r.onload = () => res(String(r.result).split(",")[1]); r.onerror = rej; r.readAsDataURL(blob); });
}
function renderPending() {
  sendState();
  const box = $("pending"); box.textContent = "";
  box.hidden = !S.pending.length;
  S.pending.forEach((p, i) => {
    const f = el("figure"); const img = el("img"); img.alt = t("imageTitle"); img.src = "data:" + p.mime + ";base64," + p.data;
    const x = el("button", "×"); x.type = "button"; x.setAttribute("aria-label", t("removeImage"));
    x.addEventListener("click", () => { S.pending.splice(i, 1); renderPending(); });
    f.append(img, x); box.append(f);
  });
}
function clearPending() { S.pending = []; renderPending(); }
// toastText: a short message over the page for a few seconds; it never
// touches the banner (an offline notice stays as it is).
let toastTimer = 0;
function toastText(s) {
  const n = $("toast"); n.textContent = s; n.hidden = false;
  clearTimeout(toastTimer); toastTimer = setTimeout(() => { n.hidden = true; }, 4000);
}

function busyUI(on) { $("send").classList.toggle("stop", on); $("send").setAttribute("aria-label", t(on ? "stop" : "send")); sendState(); }

// ---------- drawer ----------
$("menu").addEventListener("click", () => { $("drawer").hidden = false; });
$("drawer").addEventListener("click", (e) => { if (e.target.id === "drawer") closeDrawer(); });
function closeDrawer() {
  $("drawer").hidden = true;
  if ($("chat-search").value) { $("chat-search").value = ""; if (S.cur) renderList(); }
}
$("chat-search").placeholder = t("searchChats"); $("chat-search").setAttribute("aria-label", t("searchChats"));
$("chat-search").addEventListener("input", () => { if (S.cur) renderList(); });
$("drawer-close").setAttribute("aria-label", t("close")); $("drawer-close").title = t("close");
$("drawer-close").addEventListener("click", closeDrawer);
$("install-tip").addEventListener("click", () => { ask(t("addHomeTip"), t("gotIt")); });
// Remove this device: on the Mac too (so it leaves the device list), then
// here. If the Mac can't be reached it is still removed here, and the
// welcome screen says to remove it on the Mac as well.
$("forget").addEventListener("click", async () => {
  if (!await ask(t("forgetQ"), t("remove"), true) || !S.device) return; // removed on the Mac meanwhile
  const btn = $("forget"); btn.disabled = true; btn.textContent = t("removing");
  if (S.busy) { try { omnillm.cancel(S.busy); } catch {} }
  let far = !navigator.onLine;
  if (!far) { try { await omnillm.leave(); } catch { far = true; } }
  btn.disabled = false; btn.textContent = t("forget");
  if (!S.device) return; // the Mac's removal got here first: unpaired() has run
  try { omnillm.close(); } catch {}
  S.openedState = null;
  await unpaired(far ? "forgetFar" : "");
});
// ask shows a question in the page (no browser dialog): yes is the action
// (danger: red), Cancel beside it; with no cancel it is a note with one
// button. Esc or a tap outside counts as Cancel. Resolves true for yes.
let askDone = null; // the open question's answer, so unpaired() can cancel it
function ask(text, yes, danger, cancel = !!danger) {
  const d = $("ask"), ok = $("ask-yes"), no = $("ask-no");
  quoted($("ask-text"), text);
  ok.textContent = yes; ok.className = danger ? "primary bad" : "primary";
  no.textContent = t("cancel"); no.hidden = !cancel;
  return new Promise((res) => {
    if (askDone) askDone(false); // a newer question replaces an open one
    const done = askDone = (v) => { askDone = null; ok.onclick = no.onclick = d.onclick = d.oncancel = null; if (d.open) d.close(); res(v); };
    ok.onclick = () => done(true);
    no.onclick = () => done(false);
    d.oncancel = (e) => { e.preventDefault(); done(false); };
    d.onclick = (e) => { // the backdrop (the card's own padding is the dialog too)
      const r = d.getBoundingClientRect();
      if (e.target === d && (e.clientX < r.left || e.clientX > r.right || e.clientY < r.top || e.clientY > r.bottom)) done(false);
    };
    d.showModal();
    (cancel ? no : ok).focus();
  });
}
async function unpaired(note) {
  if (S.quiet) return; // the tab in use handles it
  S.gen++; // saves already on their way are dropped (seal)
  if (askDone) askDone(false); // an open question is about a pairing that is gone
  S.device = null; S.chats = []; S.cur = null; S.models = []; S.listed = false; S.paused = null; S.hello = null;
  await idb.del("device"); await idb.del("chats");
  closeDrawer(); show("welcome");
  $("welcome-err").textContent = "";
  welcomeNote(note);
}
