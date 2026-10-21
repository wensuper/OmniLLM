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
const T = {
  zh: {
    loading: "正在载入……", bootDownload: "正在下载……", bootStart: "正在启动……", bootStorage: "正在读取这台设备上保存的内容……",
    bootSlow: "载入停了 20 秒没有进展，停在“{0}”。点下面的按钮清掉缓存重新载入；网络慢的话也可以换个网络再试。",
    bootCrashed: "程序意外停止了。点下面的按钮清掉缓存重新载入。", bootError: "载入失败：", bootRetry: "重新载入",
    welcomeTitle: "在这台设备上用你的模型",
    welcomeBody: "在你的 Mac（或装了 OmniLLM 的电脑）上打开 OmniLLM，点“添加设备”，再用这台设备扫屏幕上的二维码。",
    scan: "扫描二维码", pasteInstead: "扫不了码？粘贴配对文字", connect: "连接", cancel: "取消", back: "返回",
    scanHint: "把主机屏幕上的二维码放进框里", camDenied: "没有拿到摄像头权限。可以改用下面的“粘贴配对文字”。",
    pairTitle: "添加这台设备", pairConnecting: "正在连接你的主机……",
    pairAsk: "把这台设备连到经过下面这个连接服务的主机上？只在你自己刚在主机上点了“添加设备”时继续。",
    pairReplace: "这会替换这台设备现在的配对（{0}），之前的主机就不能再从这里用了。",
    pairCheck: "在主机上核对这个数字，一致就在主机上点确认：", pairDone: "已添加。",
    notCode: "这不是 OmniLLM 的配对二维码或配对文字。", needTLS: "这台主机的连接服务不支持手机网页（需要 wss:// 地址）。请在主机上设置一个公开的连接服务。",
    chats: "对话", newChat: "新对话", placeholder: "输入消息", stop: "停止",
    addImage: "添加图片", removeImage: "移除图片", tooManyImages: "每条消息最多 4 张图片。", imageFailed: "这张图片打不开，换一张试试。", imageTitle: "图片",
    offline: "连不上主机：它可能睡眠了或者没联网。醒来后会自动恢复。", retry: "重试",
    noNetwork: "这台设备没有联网。联网后会自动连上，刚才输入的内容还在。", loadingModels: "正在连接主机……",
    noModels: "主机上还没有可用的模型。", forget: "断开这台设备", forgetQ: "断开后，这台设备不能再用你 Mac 上的模型；对话记录也会从这台设备上删掉。Mac 的设备列表里它还在，记得在那里也移除。确定吗？", close: "关闭", deleteChat: "删除这个对话", deleteChatQ: "删除对话“{0}”？",
    addHome: "添加到主屏幕", addHomeTip: "点浏览器的“分享”按钮，选“添加到主屏幕”。从主屏幕打开时需要再扫一次码。",
    deviceLine: "这台设备：{0}", sub: "订阅", empty: "有什么可以帮你？", cut: "（回复被截断）", stopped: "（已停止）",
    today: "今天", copy: "复制", copied: "已复制",
    volatile: "这个浏览器不能安全地保存配对，所以只在这个页面打开时有效；关掉后要在主机上重新添加。换用 Safari 或 Chrome 可以一直保存。",
    removedHere: "这台设备已在主机上被移除，对话记录也已从这台设备上清除。需要的话，在主机上重新添加。",
  },
  en: {
    loading: "Loading…", bootDownload: "Downloading…", bootStart: "Starting…", bootStorage: "Reading what this device saved…",
    bootSlow: "Loading has made no progress for 20 seconds, stuck at \"{0}\". Tap below to clear the cache and load again; on a slow network, another network may help.",
    bootCrashed: "The program stopped unexpectedly. Tap below to clear the cache and load again.", bootError: "Loading failed: ", bootRetry: "Load again",
    welcomeTitle: "Use your models on this device",
    welcomeBody: "On your Mac (or the computer running OmniLLM), open OmniLLM, choose Add device, and scan the QR code on its screen with this device.",
    scan: "Scan QR code", pasteInstead: "Can't scan? Paste the pairing text", connect: "Connect", cancel: "Cancel", back: "Back",
    scanHint: "Point at the QR code on your hub's screen", camDenied: "No camera access. Use \"paste the pairing text\" below instead.",
    pairTitle: "Add this device", pairConnecting: "Connecting to your hub…",
    pairAsk: "Connect this device to the hub reached through the connection service below? Continue only if you just chose Add device on your own hub.",
    pairReplace: "This replaces this device's current pairing ({0}); the previous hub can no longer be used from here.",
    pairCheck: "Check this number on your hub and confirm there if it matches:", pairDone: "Added.",
    notCode: "This is not an OmniLLM pairing QR code or pairing text.", needTLS: "This hub's connection service can't be used from a phone browser (it needs a wss:// address). Set up a public connection service on the hub.",
    chats: "Chats", newChat: "New chat", placeholder: "Message", stop: "Stop",
    addImage: "Add image", removeImage: "Remove image", tooManyImages: "Up to 4 images per message.", imageFailed: "This image can't be opened; try another one.", imageTitle: "Image",
    offline: "Can't reach your hub: it may be asleep or offline. This recovers by itself when it wakes.", retry: "Retry",
    noNetwork: "This device is offline. It reconnects by itself when the network is back; what you typed is kept.", loadingModels: "Connecting to your hub…",
    noModels: "The hub has no models yet.", forget: "Disconnect this device", forgetQ: "This device will no longer use your Mac's models, and its chats are deleted from it. It still shows in the Mac's device list; remove it there too. Continue?", close: "Close", deleteChat: "Delete this chat", deleteChatQ: "Delete the chat \"{0}\"?",
    addHome: "Add to Home Screen", addHomeTip: "Tap the browser's Share button and choose Add to Home Screen. Opened from the Home Screen, it asks for the code once more.",
    deviceLine: "This device: {0}", sub: "subscription", empty: "How can I help?", cut: "(reply cut off)", stopped: "(stopped)",
    today: "Today", copy: "Copy", copied: "Copied",
    volatile: "This browser can't keep the pairing safely, so it only lasts while this page is open; after closing it, add the device on the hub again. Safari or Chrome keep it.",
    removedHere: "This device was removed on the hub, and its conversations were cleared from it. Add it again on the hub if you need it.",
  },
};
const lang = (navigator.language || "en").toLowerCase().startsWith("zh") ? "zh" : "en";
document.documentElement.lang = lang === "zh" ? "zh-CN" : "en";
function t(k, ...a) { let s = (T[lang][k] ?? T.en[k] ?? k); a.forEach((v, i) => { s = s.replace("{" + i + "}", v); }); return s; }
document.querySelectorAll("[data-t]").forEach((e) => { e.textContent = t(e.dataset.t); });
const $ = (id) => document.getElementById(id);
function el(tag, text, cls) { const e = document.createElement(tag); if (text !== undefined) e.textContent = text; if (cls) e.className = cls; return e; }
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
async function seal(name, value) {
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const ct = await crypto.subtle.encrypt({ name: "AES-GCM", iv, additionalData: new TextEncoder().encode(name) }, await wrapKey(),
    new TextEncoder().encode(JSON.stringify(value)));
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
const S = { device: null, models: [], chats: [], cur: null, busy: null, online: true };
const isStandalone = matchMedia("(display-mode: standalone)").matches || navigator.standalone === true;
function deviceName() {
  const ua = navigator.userAgent;
  if (/iPad/.test(ua) || (/Macintosh/.test(ua) && navigator.maxTouchPoints > 1)) return "iPad";
  if (/iPhone/.test(ua)) return "iPhone";
  if (/Android/.test(ua)) return /Mobile/.test(ua) ? "Android phone" : "Android tablet";
  return lang === "zh" ? "网页浏览器" : "Web browser";
}

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

// ---------- pairing ----------
$("paste-go").addEventListener("click", () => {
  const v = $("paste-text").value.trim();
  if (!v) return;
  pair(v);
});
$("pair-retry").addEventListener("click", () => { show(S.device ? "chat" : "welcome"); });

// pair connects this device. A link (fromLink) could have been sent by
// anyone, so the person first sees which connection service it leads to
// and taps Connect; a code they scanned or pasted here needs no extra tap.
async function pair(text, fromLink) {
  if (S.pairing) return;
  const m = text.match(/omnillm[123]-[A-Za-z0-9_-]+/);
  if (!m) { $("welcome-err").textContent = t("notCode"); show("welcome"); return; }
  text = m[0];
  S.pairing = true;
  try {
    show("pairing");
    $("pair-err").textContent = ""; $("pair-retry").hidden = true; $("sas").hidden = true; $("pair-status").textContent = "";
    try { await omnillm.check(text); } catch (e) { $("pair-err").textContent = errText(e, text); $("pair-retry").hidden = false; return; }
    if (fromLink || S.device) {
      $("pair-host").textContent = relayHost(text);
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
function relayHost(code) {
  try { return new URL(relayURL(code)).host; } catch { return "?"; }
}

async function connectPair(text) {
  $("pair-status").textContent = t("pairConnecting");
  try {
    await omnillm.check(text);
    const name = deviceName();
    const state = await omnillm.pair(text, name, (code) => {
      $("pair-status").textContent = t("pairCheck");
      $("sas").textContent = code.slice(0, 3) + " " + code.slice(3);
      $("sas").hidden = false;
    });
    if (S.device) { try { omnillm.close(); } catch {} }
    S.device = { state, name, added: Date.now() };
    await seal("device", S.device);
    $("pair-status").textContent = t("pairDone");
    $("sas").hidden = true;
    S.cur = null;
    setTimeout(openChat, 600);
  } catch (e) {
    $("sas").hidden = true;
    $("pair-status").textContent = "";
    $("pair-err").textContent = errText(e, text);
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
  } catch { $("welcome-err").textContent = t("camDenied"); $("welcome").querySelector("details").open = true; return; }
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
  renderList(); renderLog(); sendState();
  await connect();
}

// connect opens the device and loads the models; S.ready_ is that work,
// so a message sent meanwhile waits for it instead of going nowhere.
function connect() {
  S.connecting = (async () => {
    if (!navigator.onLine) { setOnline(false, t("noNetwork")); return; }
    const sel = $("model");
    if (!S.models.length) { sel.textContent = ""; const o = el("option", t("loadingModels")); o.value = ""; sel.append(o); }
    try {
      // Open once per pairing: reopening closes the old device, and with it
      // a reply still on its way (the network coming back fires "online"
      // mid-reply). An open device reconnects by itself.
      if (S.openedState !== S.device.state) { await omnillm.open(S.device.state); S.openedState = S.device.state; }
      await loadModels();
      setOnline(true);
    } catch (e) {
      if (e && e.msg === "device_revoked") return unpaired(true);
      setOnline(false, errText(e));
    }
  })();
  return S.connecting;
}

async function recheck() {
  if (!S.device || S.busy || document.hidden || !navigator.onLine) return;
  try { await loadModels(); setOnline(true); } catch (e) {
    if (e && e.msg === "device_revoked") return unpaired(true);
    if (e && e.code === "hub_offline") setOnline(false);
  }
}
document.addEventListener("visibilitychange", () => { if (!document.hidden) recheck(); });
setInterval(recheck, 120000);

// isSub: a Claude or ChatGPT subscription (the official programs on the
// hub), not merely a model kept for people (a local model can be that too).
function isSub(m) { return m.provider === "claude-code" || m.provider === "codex"; }
function modelName(id) { const m = S.models.find((x) => x.id === id); return m ? m.name : id; }

async function loadModels() {
  const ms = await omnillm.models();
  // Subscription models (people only) first: they are what the owner pays for already.
  ms.sort((a, b) => (isSub(b) - isSub(a)) || a.name.localeCompare(b.name));
  S.models = ms;
  const sel = $("model"); sel.textContent = "";
  for (const m of ms) { const o = el("option", isSub(m) ? m.name + " · " + t("sub") : m.name); o.value = m.id; sel.append(o); }
  const want = S.cur && S.cur.model || localStorageGet("model");
  // Without a choice yet: the Claude subscription's Sonnet, else the first.
  const pick = [want, "claude-code/sonnet"].find((id) => id && ms.some((m) => m.id === id));
  if (pick) sel.value = pick;
  if (!ms.length) { const o = el("option", t("noModels")); o.disabled = true; sel.append(o); }
}
$("model").addEventListener("change", () => { localStorageSet("model", $("model").value); if (S.cur) S.cur.model = $("model").value; });
function localStorageGet(k) { try { return localStorage.getItem("omnillm." + k); } catch { return null; } }
function localStorageSet(k, v) { try { localStorage.setItem("omnillm." + k, v); } catch {} }

function setOnline(ok, why) {
  S.online = ok;
  const b = $("banner");
  b.hidden = ok; b.textContent = "";
  if (!ok) {
    b.append(el("span", why || t("offline")));
    const r = el("button", t("retry"), "link"); r.type = "button"; r.addEventListener("click", connect); b.append(r);
  }
}
// The phone's own network: say so at once, reconnect when it is back
// (the old connection may be dead without anyone noticing).
addEventListener("offline", () => { if (S.device) setOnline(false, t("noNetwork")); });
addEventListener("online", () => { if (S.device) connect(); });
document.addEventListener("visibilitychange", () => { if (!document.hidden && !S.online && S.device) connect(); });

function newChat() {
  const c = { id: crypto.randomUUID(), title: "", model: $("model").value || localStorageGet("model") || "", messages: [], updated: Date.now() };
  return c;
}
$("new").addEventListener("click", () => { if (S.busy) return; S.cur = newChat(); renderLog(); renderList(); closeDrawer(); $("input").focus(); });

function renderLog() {
  const log = $("log"); log.textContent = "";
  if (!S.cur.messages.length) log.append(el("p", t("empty"), "empty"));
  for (const m of S.cur.messages) log.append(bubble(m));
  log.scrollTop = log.scrollHeight;
}

// bubble renders a message; fenced code becomes <pre>, the rest stays text.
function bubble(m) {
  const b = el("div", undefined, "msg " + m.role);
  if (m.images && m.images.length) {
    const pics = el("div", undefined, "pics");
    for (const im of m.images) { const i = el("img"); i.alt = t("imageTitle"); i.src = "data:" + im.mime + ";base64," + im.data; pics.append(i); }
    b.append(pics);
  }
  fill(b, m.content);
  if (m.note) b.append(el("p", m.note, "note"));
  // Which model wrote this reply, so a chat that switched models stays clear.
  if (m.role === "assistant" && m.by) b.append(el("span", m.by, "by"));
  if (m.role === "assistant" && m.content) {
    const cp = el("button", t("copy"), "copy"); cp.type = "button";
    cp.addEventListener("click", () => { navigator.clipboard.writeText(m.content).then(() => { cp.textContent = t("copied"); setTimeout(() => { cp.textContent = t("copy"); }, 1200); }); });
    b.append(cp);
  }
  return b;
}
function fill(b, text) {
  const parts = text.split(/```/);
  parts.forEach((p, i) => {
    if (i % 2) { const pre = el("pre"); pre.append(el("code", p.replace(/^[\w+-]*\n/, "").replace(/\n+$/, ""))); b.append(pre); }
    else if (p.trim()) b.append(prose(p.replace(/^\n+|\n+$/g, "")));
  });
}
// prose shows the light Markdown models write (headings, **bold**, `code`,
// list bullets) as formatting. Everything stays text nodes: nothing a model
// writes is ever parsed as HTML.
function prose(text) {
  const p = el("p");
  text.split("\n").forEach((line, i) => {
    if (i) p.append("\n");
    let strong = false;
    const h = line.match(/^#{1,6}\s+(.*)$/);
    if (h) { line = h[1]; strong = true; }
    line = line.replace(/^(\s*)[-*+]\s+/, "$1• ");
    inline(p, line, strong);
  });
  return p;
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

function renderList() {
  const ul = $("chat-list"); ul.textContent = "";
  const all = [S.cur, ...S.chats.filter((c) => c.id !== S.cur.id)].filter((c) => c.messages.length || c === S.cur);
  for (const c of all) {
    const li = el("li"); const a = el("button", c.title || t("newChat"), c.id === S.cur.id ? "cur" : ""); a.type = "button";
    a.addEventListener("click", () => { if (S.busy) return; S.cur = c; if (c.model) $("model").value = c.model; renderLog(); renderList(); closeDrawer(); });
    li.append(a);
    if (c.messages.length) {
      const x = el("button", "", "chat-del"); x.type = "button"; x.setAttribute("aria-label", t("deleteChat")); x.title = t("deleteChat");
      x.addEventListener("click", async () => {
        if (S.busy && c === S.cur) return;
        if (!confirm(t("deleteChatQ", c.title || t("newChat")))) return;
        S.chats = S.chats.filter((o) => o.id !== c.id);
        if (c === S.cur) { S.cur = S.chats[0] || newChat(); renderLog(); }
        await seal("chats", S.chats).catch(() => {});
        renderList();
      });
      li.append(x);
    }
    ul.append(li);
  }
}

async function saveChats() {
  S.cur.updated = Date.now();
  const rest = S.chats.filter((c) => c.id !== S.cur.id);
  S.chats = [S.cur, ...rest].filter((c) => c.messages.length).slice(0, 50);
  await seal("chats", S.chats).catch(() => {});
}

const input = $("input");
input.addEventListener("input", () => { grow(); sendState(); });
// The send button looks pressable only when there is something to send.
function sendState() { $("send").classList.toggle("idle", !S.busy && !input.value.trim() && !(S.pending && S.pending.length)); }
function grow() { input.style.height = "auto"; input.style.height = Math.min(input.scrollHeight, 180) + "px"; }
input.addEventListener("keydown", (e) => {
  // Enter sends on a computer keyboard; on phones the send button does.
  if (e.key === "Enter" && !e.shiftKey && !e.isComposing && matchMedia("(pointer: fine)").matches) { e.preventDefault(); $("composer").requestSubmit(); }
});
$("composer").addEventListener("submit", async (e) => {
  e.preventDefault();
  if (S.busy) { omnillm.cancel(S.busy); return; }
  const text = input.value.trim();
  const images = S.pending.map(({ mime, data }) => ({ mime, data }));
  if (!text && !images.length) return;
  if (!navigator.onLine) { setOnline(false, t("noNetwork")); return; } // keep the text
  if (!$("model").value && S.connecting) { // still loading the models
    if (S.waiting) return;
    S.waiting = true;
    try { await S.connecting; } finally { S.waiting = false; }
  }
  const model = $("model").value;
  if (!model) { if (S.online) setOnline(false); return; }
  input.value = ""; grow(); clearPending();
  const chat = S.cur; // this reply belongs to this conversation, whatever happens meanwhile
  chat.model = model;
  if (!chat.title) chat.title = text.slice(0, 40) || t("imageTitle");
  chat.messages.push(images.length ? { role: "user", content: text, images } : { role: "user", content: text });
  const reply = { role: "assistant", content: "", by: modelName(model) };
  chat.messages.push(reply);
  renderLog(); renderList();
  const log = $("log"), node = log.lastElementChild;
  const id = crypto.randomUUID(); S.busy = id; busyUI(true);
  // Images go with the latest two turns that have them; older ones are
  // named in the text, so a long chat with pictures keeps working.
  const withImages = new Set(chat.messages.slice(0, -1).filter((m) => m.images && m.images.length).slice(-2));
  const msgs = chat.messages.slice(0, -1).map((m) => {
    if (!m.images || !m.images.length) return { role: m.role, content: m.content };
    if (withImages.has(m)) return { role: m.role, content: m.content, images: m.images };
    return { role: m.role, content: (m.content ? m.content + "\n" : "") + "[" + m.images.length + " earlier image(s)]" };
  });
  let raf = 0;
  const flush = () => { raf = 0; node.textContent = ""; fill(node, reply.content); log.scrollTop = log.scrollHeight; };
  try {
    const r = await omnillm.chat(id, model, JSON.stringify(msgs), (d) => { reply.content += d; if (!raf) raf = requestAnimationFrame(flush); });
    if (r.finish === "length") reply.note = t("cut");
    setOnline(true);
  } catch (err) {
    if (err && err.code === "cancelled") reply.note = t("stopped");
    else if (err && err.msg === "device_revoked") return unpaired(true);
    else {
      reply.note = errText(err);
      if (err && err.code === "hub_offline") setOnline(false);
    }
    if (!reply.content) { chat.messages.pop(); if (err && err.code !== "cancelled") { chat.messages.pop(); input.value = text; grow(); S.pending = images; renderPending(); } }
  } finally {
    S.busy = null; busyUI(false);
    // Removed meanwhile (unpaired): nothing left to show or save.
    if (!S.device) return;
    renderLog();
    if (reply.note && !reply.content) { const n = el("p", reply.note, "note err"); $("log").append(n); }
    await saveChats();
  }
});
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
    try { S.pending.push(await shrink(f)); } catch { toastText(t("imageFailed")); }
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
function toastText(s) { setOnline(S.online); const n = el("p", s, "note err"); $("log").append(n); $("log").scrollTop = $("log").scrollHeight; setTimeout(() => n.remove(), 4000); }

function busyUI(on) { $("send").classList.toggle("stop", on); $("send").setAttribute("aria-label", on ? t("stop") : "Send"); sendState(); }

// ---------- drawer ----------
$("menu").addEventListener("click", () => { $("drawer").hidden = false; });
$("drawer").addEventListener("click", (e) => { if (e.target.id === "drawer") closeDrawer(); });
function closeDrawer() { $("drawer").hidden = true; }
$("drawer-close").setAttribute("aria-label", t("close")); $("drawer-close").title = t("close");
$("drawer-close").addEventListener("click", closeDrawer);
$("install-tip").addEventListener("click", () => { alert(t("addHomeTip")); });
$("forget").addEventListener("click", async () => {
  if (!confirm(t("forgetQ"))) return;
  try { omnillm.close(); } catch {}
  await unpaired();
});
async function unpaired(removed) {
  S.device = null; S.chats = []; S.cur = null;
  await idb.del("device"); await idb.del("chats");
  closeDrawer(); show("welcome");
  $("welcome-err").textContent = removed ? t("removedHere") : "";
}
