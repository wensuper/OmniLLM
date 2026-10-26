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
    bootCrashed: "程序意外停止了。点下面的按钮清掉缓存重新载入。", bootStorageFailed: "这个浏览器不让网页保存数据（可能在设置里禁止了网站数据）。允许这个网站保存数据后再载入。", bootError: "载入失败：", bootRetry: "重新载入",
    welcomeTitle1: "你电脑上的 AI，", welcomeTitle2: "现在也在这台设备上",
    step1: "打开电脑上的 OmniLLM 控制台（Mac：菜单栏图标）", step2: "点“添加设备”", step3: "扫电脑屏幕上的二维码",
    e2eChip: "端到端加密", noHub: "还没有 OmniLLM？先装好",
    getHubTitle: "先在你的电脑上装好 OmniLLM", getHubBody: "打开终端（Mac：按 ⌘+空格，输入“终端”后回车），粘贴下面这一行再按回车。大约一分钟装好，然后回到这里扫码。", getHubLongPress: "请长按上面这一行复制。", getHubBodyPaste: "打开终端（Mac：按 ⌘+空格，输入“终端”后回车），粘贴下面这一行再按回车。大约一分钟装好，然后回到这里。",
    step3Paste: "在电脑的“添加设备”里选“电脑或服务器”，点“复制链接”在这里打开；或者点“用命令行”复制配对文字，粘贴到下面",
    getHubShare: "发到我的电脑", getHubCopy: "复制命令", getHubGuide: "查看完整的安装说明", getHubCopied: "已复制。把它发到电脑上，在终端里粘贴运行。",
    getHubHang: "如果卡住不动，多半是连不上 GitHub，换个网络再试。", pasteOpen: "粘贴配对文字",
    getHubSend: "在电脑上安装 OmniLLM：在终端里运行这一行",
    scan: "扫描二维码", pasteInstead: "扫不了码？粘贴配对文字", pastePh: "粘贴配对文字（以 omnillm3- 开头）", add: "添加", cancel: "取消", back: "返回", delete: "删除", remove: "移除", gotIt: "知道了",
    scanHint: "把电脑屏幕上的二维码放进框里", camDenied: "没有拿到摄像头权限。在浏览器设置里允许这个网站用摄像头后再试，或者用“粘贴配对文字”。",
    pairTitle: "添加这台设备", pairConnecting: "正在连接你的电脑……",
    pairAsk: "要把这台设备添加到你的电脑吗？只有你刚扫的二维码正显示在你自己的电脑上，才继续。", pairVia: "经由 {0}",
    pairReplace: "这会替换这台设备现在的配对（{0}），之前的 Mac 就不能再从这里用了。",
    pairCheck: "电脑上显示同样的数字，就在那边确认：控制台里点“一致，添加”，终端里输入 y。", pairWait: "等你在电脑上确认……", pairMismatch: "数字不一样？取消", pairDone: "已添加。",
    pairDropped: "连接断了。在电脑上换一个二维码再扫（控制台点“换一个二维码”，终端里重新运行 omnillm pair）。", connectedMac: "已连上你的 Mac",
    notCode: "这不是 OmniLLM 的配对二维码或配对文字。请从电脑上“添加设备”里重新复制。", needTLS: "你的 Mac 的远程连接只能在它自己的网络里用，手机网页连不上。在 Mac 的终端里用 wss:// 地址重新安装（OMNILLM_RELAY=wss://… omnillm install；不设就用默认的），再换一个新的二维码扫。",
    chats: "对话", newChat: "新对话", placeholder: "输入消息", stop: "停止", send: "发送", menu: "菜单", model: "模型",
    addImage: "添加图片", voice: "语音输入", voiceStop: "停止语音输入", listening: "正在听，说完再点一下……", voiceFirst: "语音由浏览器识别：Chrome 交给 Google，Safari 交给 Apple。识别出的文字只填进输入框，你确认后再发。", voiceDenied: "没有麦克风权限。在浏览器的网站设置里允许麦克风，再点一次。", voiceNoMic: "没找到麦克风。", voiceNoSpeech: "没听到声音，再说一次试试。", voiceNetwork: "语音识别要联网，现在连不上。", voiceFailed: "语音输入没能开始，再试一次。", removeImage: "移除图片", tooManyImages: "每条消息最多 4 张图片。", imageFailed: "这张图片打不开，换一张试试。", imageTooBig: "图片太大，换一张小一点的。", imageTitle: "图片",
    offline: "你的 Mac 现在连不上，可能在睡眠、关机或断网。恢复后会自动连上。", retry: "重试",
    noNetwork: "这台设备没有联网，联网后会自动连上。", kept: "刚才输入的还在。", loadingModels: "正在连接你的 Mac……",
    noModels: "这台设备现在没有能用的模型。在 Mac 的控制台里接入模型，或在“设备”里给它开放模型。", noModelsOpt: "没有可用的模型", probLogin: "要在 Mac 上登录", probLimit: "到上限，{0} 恢复", probLoginNote: "你的 Mac 上 {0}的登录已失效，现在发不出去。在 Mac 的终端里运行 omnillm login {1}，或在控制台里点“登录”，然后再试。", langPending: "先发出或移除已添加的图片，再切换语言。", langVolatile: "这个浏览器保存不了配对，换语言要重新载入，就得重新添加这台设备。换用 Safari 或 Chrome 再切换。", scanFailed: "扫码工具没能载入。用下面的“粘贴配对文字”，或刷新后再试。", addHomeTipAndroid: "点浏览器右上角的 ⋮，选“添加到主屏幕”或“安装应用”。", probLimitNote: "{0}的用量到上限了，{1} 恢复。可以先换一个模型。", probLimitNoTime: "到上限了", probLimitNoteNoTime: "{0}的用量到上限了。可以先换一个模型。", subClaude: "Claude 订阅", subChatGPT: "ChatGPT 订阅", subGemini: "Gemini 订阅",
    modelGone: "原来的模型已从你的 Mac 上移除，换成了 {0}",
    grpSub: "订阅（不另花钱）", grpKey: "API Key", grpLocal: "你的 Mac 上", latest: "（最新版）", codex: "ChatGPT（Codex · 最新版）", codexV: "ChatGPT（Codex · {0}）",
    forget: "移除这台设备", forgetQ: "这里的对话会删除。以后要用，在 Mac 上重新添加。", removing: "正在移除……",
    forgetFar: "这台设备已移除，对话已删除。你的 Mac 现在连不上，之后记得在 Mac 的“设备”里也移除它。",
    close: "关闭", deleteChat: "删除这个对话", rename: "重命名", renameChat: "重命名对话", chatMore: "更多操作", deleteChatQ: "删除对话“{0}”？", chatsLocal: "对话保存在这台设备上。用订阅模型（Claude、ChatGPT）时，你的 Mac 也会留一份，方便接着聊；一天没用，Mac 上的那份就会删除。", searchChats: "搜索对话", noMatch: "没有找到",
    addHome: "添加到主屏幕", addHomeTip: "点浏览器的“分享”按钮，选“添加到主屏幕”。第一次从主屏幕打开时，需要再扫一次码。",
    deviceLine: "这台设备：{0}", empty: "有什么可以帮你？", cut: "（回复太长被截断了。发“继续”让它接着写。）", stopped: "（已停止）",
    today: "今天", copy: "复制", copied: "已复制", again: "重新回答", againWith: "用 {0} 再答一次", thinking: "正在思考",
    earlier: "之前的回答（{0}）", earlierOpen: "之前的回答（{0}）",
    yourMac: "你的 Mac", macOn: "在线", macOff: "连不上 · 醒来后自动恢复", macWait: "正在连接……", macPaused: "已暂停这台设备", macNoNet: "这台设备没有联网",
    language: "语言", langMac: "跟随你的 Mac", shortcuts: "键盘快捷键", hideSide: "收起侧栏", showSide: "展开侧栏", dropHere: "松开就能添加图片",
    grpToday: "今天", grpYesterday: "昨天", grpWeek: "前 7 天", grpOlder: "更早", copyCode: "复制", modelMenu: "选择模型",
    kNew: "新对话", kSearch: "搜索对话", kSide: "收起或展开侧栏", kStop: "停止回答", kCopyLast: "复制最后一条回答", kFocus: "跳到输入框", kSend: "发送", kLine: "换行",
    dropped: "（连接断了，回答没写完。连上后点“重试”。）",
    longChat: "对话已经很长，每次都会整段发出。开个新对话会更快，用 API Key 时也更省。", longChatSub: "对话已经很长，开个新对话回答会更快。", paused: "已暂停", pausedDot: "已在你的 Mac 上暂停", waited: "{0} 秒", slow: "比平常慢。可以点停止，换个模型再问。",
    switchedTo: "换成了 {0}", newContent: "↓ 新内容", waitReply: "等这条回答完成，或先点停止。",
    liveOn: "在线：你的 Mac 连得上，模型可以用", liveOff: "离线：现在连不上你的 Mac", liveWait: "正在连接你的 Mac……",
    volatile: "这个浏览器不能安全地保存配对，所以只在这个页面打开时有效；关掉后要在 Mac 上重新添加。换用 Safari 或 Chrome 可以一直保存。",
    removedHere: "这台设备已从你的 Mac 上移除，这里的对话也已删除。要再用，在 Mac 上点“添加设备”再扫一次。",
  },
  en: {
    loading: "Loading…", bootDownload: "Downloading…", bootStart: "Starting…", bootStorage: "Opening…",
    bootSlow: "Loading has made no progress for 20 seconds, stuck at “{0}”. Tap below to clear the cache and load again; on a slow network, another network may help.",
    bootCrashed: "The program stopped unexpectedly. Tap below to clear the cache and load again.", bootStorageFailed: "This browser isn’t letting the page save data (site data may be blocked in its settings). Allow site data for this site, then load again.", bootError: "Loading failed: ", bootRetry: "Load again",
    welcomeTitle1: "The AI on your computer, ", welcomeTitle2: "now on this device",
    step1: "Open the OmniLLM console on your computer (on a Mac: the menu bar icon)", step2: "Choose “Add device”", step3: "Scan the QR code on your computer’s screen",
    e2eChip: "End-to-end encrypted", noHub: "No OmniLLM yet? Set it up first",
    getHubTitle: "First, set up OmniLLM on your computer", getHubBody: "Open a terminal (Mac: press ⌘-Space, type “Terminal”, press Return), paste this line and press Return. It takes about a minute; then come back here and scan.", getHubLongPress: "Touch and hold the line above to copy it.", getHubBodyPaste: "Open a terminal (Mac: press ⌘-Space, type “Terminal”, press Return), paste this line and press Return. It takes about a minute; then come back here.",
    step3Paste: "In “Add device” on your computer, pick “Computer or server” and click “Copy link” to open it here, or click “Use the command line” and paste the pairing text below",
    getHubShare: "Send this to my computer", getHubCopy: "Copy command", getHubGuide: "Full setup guide", getHubCopied: "Copied. Get it to your computer and paste it into a terminal there.",
    getHubHang: "If it hangs, GitHub is probably unreachable; try another network.", pasteOpen: "Paste the pairing text",
    getHubSend: "Install OmniLLM on your computer: run this line in a terminal",
    scan: "Scan QR code", pasteInstead: "Can’t scan? Paste the pairing text", pastePh: "Paste the pairing text (starts with omnillm3-)", add: "Add", cancel: "Cancel", back: "Back", delete: "Delete", remove: "Remove", gotIt: "Got it",
    scanHint: "Point at the QR code on your computer’s screen", camDenied: "No camera access. Allow the camera for this site in your browser settings and try again, or use “Paste the pairing text”.",
    pairTitle: "Add this device", pairConnecting: "Connecting to your computer…",
    pairAsk: "Add this device to your computer? Continue only if the QR code you just scanned is showing on your own computer.", pairVia: "Via {0}",
    pairReplace: "This replaces this device’s current pairing ({0}); the previous Mac can no longer be used from here.",
    pairCheck: "If your computer shows the same number, confirm there: “Matches, add” in the console, or type y in the terminal.", pairWait: "Waiting for you to confirm on your computer…", pairMismatch: "Numbers don’t match? Cancel", pairDone: "Added.",
    pairDropped: "The connection dropped. Get a new QR code on your computer (“New QR code” in the console, or run omnillm pair again) and scan it.", connectedMac: "Connected to your Mac",
    notCode: "This is not an OmniLLM pairing QR code or pairing text. Copy it again from “Add device” on your computer.", needTLS: "Your Mac’s remote access only works on its own network, so a phone browser can’t use it. On your Mac, install again with a wss:// address (OMNILLM_RELAY=wss://… omnillm install, or leave it unset for the default), then scan a new QR code.",
    chats: "Chats", newChat: "New chat", placeholder: "Message", stop: "Stop", send: "Send", menu: "Menu", model: "Model",
    addImage: "Add image", voice: "Voice input", voiceStop: "Stop voice input", listening: "Listening… tap again when you’re done", voiceFirst: "Your browser turns speech into text: Chrome uses Google, Safari uses Apple. The words only fill the box; you send them.", voiceDenied: "No microphone permission. Allow the microphone in the browser’s site settings, then try again.", voiceNoMic: "No microphone found.", voiceNoSpeech: "Didn’t hear anything. Try again.", voiceNetwork: "Voice input needs the internet, which isn’t reachable right now.", voiceFailed: "Voice input couldn’t start. Try again.", removeImage: "Remove image", tooManyImages: "Up to 4 images per message.", imageFailed: "This image can’t be opened; try another one.", imageTooBig: "Image too large; try a smaller one.", imageTitle: "Image",
    offline: "Your Mac can’t be reached right now. It may be asleep, off or offline, and will reconnect when it’s back.", retry: "Retry",
    noNetwork: "This device is offline; it reconnects when the network is back.", kept: "What you typed is kept.", loadingModels: "Connecting to your Mac…",
    noModels: "No models are available to this device. On your Mac, connect a model or allow this device more models under Devices.", noModelsOpt: "No models", probLogin: "Sign in on your Mac", probLimit: "At its limit until {0}", probLoginNote: "{0}’s sign-in has expired on your Mac, so messages to it won’t go through. In a terminal on your Mac, run omnillm login {1}, or choose “Sign in” in the console, then try again.", langPending: "Send or remove the added images first, then switch language.", langVolatile: "This browser can’t keep the pairing, and switching language reloads the page, so the device would have to be added again. Switch in Safari or Chrome instead.", scanFailed: "The scanner couldn’t load. Use “Paste the pairing text” below, or reload and try again.", addHomeTipAndroid: "Tap the browser’s ⋮ menu and choose “Add to Home screen” or “Install app”.", probLimitNote: "{0} has reached its usage limit until {1}. Pick another model for now.", probLimitNoTime: "At its limit", probLimitNoteNoTime: "{0} has reached its usage limit. Pick another model for now.", subClaude: "Your Claude subscription", subChatGPT: "Your ChatGPT subscription", subGemini: "Your Google AI subscription",
    modelGone: "The previous model was removed from your Mac; switched to {0}",
    grpSub: "Subscriptions (no extra cost)", grpKey: "API key", grpLocal: "On your Mac", latest: " (latest)", codex: "ChatGPT (Codex · latest)", codexV: "ChatGPT (Codex · {0})",
    forget: "Remove this device", forgetQ: "Chats here will be deleted. To use this device later, add it on your Mac again.", removing: "Removing…",
    forgetFar: "This device was removed and its chats deleted. Your Mac can’t be reached right now, so remove it under “Devices” on your Mac later too.",
    close: "Close", deleteChat: "Delete this chat", rename: "Rename", renameChat: "Rename chat", chatMore: "More", deleteChatQ: "Delete the chat “{0}”?", chatsLocal: "Chats are kept on this device. With subscription models (Claude, ChatGPT), your Mac also keeps a copy so it can continue the chat; it deletes that copy after a day without use.", searchChats: "Search chats", noMatch: "No matching chats",
    addHome: "Add to Home Screen", addHomeTip: "Tap the browser’s “Share” button and choose “Add to Home Screen”. The first time it’s opened from the Home Screen, you’ll scan the QR code once more.",
    deviceLine: "This device: {0}", empty: "How can I help?", cut: "(The reply was cut off. Send “continue” to have it go on.)", stopped: "(stopped)",
    today: "Today", copy: "Copy", copied: "Copied", again: "Answer again", againWith: "Answer again with {0}", thinking: "Thinking",
    earlier: "Earlier answer ({0})", earlierOpen: "Earlier answer ({0})",
    yourMac: "Your Mac", macOn: "Online", macOff: "Can’t reach it · back when it wakes", macWait: "Connecting…", macPaused: "Paused this device", macNoNet: "This device is offline",
    language: "Language", langMac: "Follow your Mac", shortcuts: "Keyboard shortcuts", hideSide: "Hide sidebar", showSide: "Show sidebar", dropHere: "Drop to add images",
    grpToday: "Today", grpYesterday: "Yesterday", grpWeek: "Previous 7 days", grpOlder: "Earlier", copyCode: "Copy", modelMenu: "Choose a model",
    kNew: "New chat", kSearch: "Search chats", kSide: "Hide or show the sidebar", kStop: "Stop the answer", kCopyLast: "Copy the last answer", kFocus: "Go to the message box", kSend: "Send", kLine: "New line",
    dropped: "(Connection lost; the reply is incomplete. Tap Retry once reconnected.)",
    longChat: "This chat is long and is sent in full each time. A new chat is faster, and cheaper with an API key.", longChatSub: "This chat is long; a new chat gets faster replies.", paused: "Paused", pausedDot: "Paused on your Mac", waited: "{0}s", slow: "Slower than usual. You can tap Stop and try another model.",
    switchedTo: "Switched to {0}", newContent: "↓ New", waitReply: "Wait for this reply, or tap Stop first.",
    liveOn: "Online: your Mac answers; models are ready", liveOff: "Offline: your Mac can’t be reached now", liveWait: "Connecting to your Mac…",
    volatile: "This browser can’t keep the pairing safely, so it only lasts while this page is open; after closing it, add the device on your Mac again. Safari or Chrome keep it.",
    removedHere: "This device was removed from your Mac and its chats here were deleted. To use it again, choose “Add device” on your Mac and scan again.",
  },
};
// The page's language: chosen here (the "Your Mac" menu), else the one the
// Mac speaks (it says so with the models; kept for the next start), else
// the browser's.
const oneLang = (v) => (v === "zh" || v === "en" ? v : "");
const browserLang = (navigator.language || "en").toLowerCase().startsWith("zh") ? "zh" : "en";
const lang = oneLang(localStorageGet("lang")) || oneLang(localStorageGet("maclang")) || browserLang;
document.documentElement.lang = lang === "zh" ? "zh-CN" : "en";
// The hub's system, from the models reply (kept for the next start): a
// hub that isn't a Mac is "your computer" in the words people see. The
// set-up steps for a computer without OmniLLM yet keep their Mac hints.
let hubOS = localStorageGet("hubos") || "";
const MAC_HINTS = new Set(["step1", "getHubBody", "getHubBodyPaste"]);
function computerWords(k, s) {
  if (hubOS === "darwin" || MAC_HINTS.has(k)) return s; // not known yet: "computer" too
  return lang === "zh" ? s.replace(/ ?Mac ?/g, "电脑") : s.replace(/\bMac\b/g, "computer");
}
function t(k, ...a) { let s = computerWords(k, T[lang][k] ?? T.en[k] ?? k); a.forEach((v, i) => { s = s.replace("{" + i + "}", v); }); return s; }
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
document.querySelectorAll("[data-ta]").forEach((e) => { e.setAttribute("aria-label", t(e.dataset.ta)); e.title = t(e.dataset.ta); });
// Shortcuts are written the Mac way (⌘, ⇧); elsewhere they read Ctrl+, Shift+.
const MAC = /Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent);
function keyName(k) { return MAC ? k : k.replace(/⌘/g, "Ctrl+").replace(/⇧/g, "Shift+"); }
document.querySelectorAll("[data-k]").forEach((e) => { e.textContent = keyName(e.dataset.k); });
// Every screen opens at its top (the welcome page kept the chat's or the
// paste box's scroll, cutting off a notice placed at the top).
function show(id) { document.querySelectorAll(".screen").forEach((s) => { s.hidden = s.id !== id; if (!s.hidden) s.scrollTop = 0; }); window.scrollTo(0, 0); }

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
    if (B.err) console.warn(B.err);
    bootFailed(t("bootSlow", t(B.stage)));
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
  // A computer: the browser and the system ("Chrome (Mac)"), so two can be told apart.
  const br = /Edg\//.test(ua) ? "Edge" : /Firefox\//.test(ua) ? "Firefox" : /Chrome\//.test(ua) ? "Chrome" : /Safari\//.test(ua) ? "Safari" : "";
  const os = /Mac OS X|Macintosh/.test(ua) ? "Mac" : /Windows/.test(ua) ? "Windows" : /Linux/.test(ua) ? "Linux" : "";
  if (br && os) return lang === "zh" ? br + "（" + os + "）" : br + " (" + os + ")";
  return lang === "zh" ? "网页浏览器" : "Web browser";
}

// ---------- one tab at a time ----------
// Two tabs would each save their own copy of the chats over the other's.
// A tab that opens says so; the others go quiet (they write nothing). A
// quiet tab takes over by itself the moment the person comes back to it
// (it gains focus or comes to the front), with no question: only the
// person's own switch moves it, so two windows side by side never pull it
// back and forth by themselves.
const tabs = "BroadcastChannel" in window ? new BroadcastChannel("omnillm") : null;
function claimTab() { if (tabs) tabs.postMessage("here"); }
function takeOver() {
  if (S.volatile) return; // a reload would lose the pairing here
  saveForReload();
  location.reload();
}
if (tabs) tabs.onmessage = (e) => {
  // The tab in use went away: a quiet tab takes over by itself (reload reads
  // what that tab saved, then claims).
  if (e.data === "gone" && S.quiet) { takeOver(); return; }
  if (e.data === "who") { if (!S.quiet) tabs.postMessage("alive"); return; }
  if (e.data === "alive") { S.alive = true; return; }
  if (e.data !== "here" || S.quiet) return;
  S.quiet = true;
  if (S.busy) { try { omnillm.cancel(S.busy); } catch {} }
};
// Back to a quiet tab: it becomes the one in use (reload reads what the
// other tab saved, then claims). Once only: focus and visibility both fire.
let takingOver = false;
function backHere() {
  if (!S.quiet || takingOver || document.hidden) return;
  takingOver = true;
  takeOver();
}
addEventListener("focus", backHere);
document.addEventListener("visibilitychange", backHere);
// "gone" can be lost when a tab closes (WebKit drops messages sent while the
// page unloads), so a quiet tab in front also asks every few seconds and
// whenever it comes back to the front; no tab in use answering → take over.
function checkTab() {
  if (!tabs || !S.quiet || document.hidden) return;
  S.alive = false;
  tabs.postMessage("who");
  setTimeout(() => { if (S.quiet && !S.alive && !document.hidden) takeOver(); }, 600);
}
document.addEventListener("visibilitychange", checkTab);
setInterval(checkTab, 3000);
claimTab();
addEventListener("pagehide", () => { if (tabs && !S.quiet) tabs.postMessage("gone"); });

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
// S.relang: the Mac now speaks another language; the page reloads in it the
// same way.
// saveForReload keeps what a reload would lose: the draft and which chat
// was open.
function saveForReload() {
  try { sessionStorage.setItem("omnillm.draft", $("input").value); sessionStorage.setItem("omnillm.cur", S.cur ? S.cur.id : ""); } catch {}
}
function reloadIfIdle() {
  // A quiet tab stays quiet: reloading would take over from the tab in use.
  // Nor under someone's fingers: the reload after the next reply catches up.
  if (!(UP.ready || S.relang) || S.quiet || S.busy || (S.pending && S.pending.length)) return;
  if (S.volatile) return; // a reload would lose the pairing here
  if (document.activeElement === $("input") && $("input").value.trim()) return;
  saveForReload();
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
  try { await startWasm(); } catch (e) { console.warn(e); bootFailed(t(!navigator.onLine || e instanceof TypeError ? "noNetwork" : "bootCrashed")); return; }
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
    // A reply cut by a reload is no longer on its way: an empty one goes
    // (Retry shows for the question), a partial one says it was cut off.
    for (const c of S.chats) {
      const last = c.messages[c.messages.length - 1];
      if (last && last.live && last.role === "assistant") {
        if (!last.content) c.messages.pop();
        else { last.note = t("dropped"); last.failed = true; }
      }
      for (const m of c.messages) delete m.live;
    }
  } catch (e) { console.warn(e); bootFailed(t("bootStorageFailed")); return; } // saved pairing and chats can't be opened
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
// With the paste box open, its Add button is the one to press. It waits
// until the box holds pairing text (it starts with omnillm3-).
$("paste-text").placeholder = t("pastePh");
$("paste").addEventListener("toggle", () => {
  const open = $("paste").open;
  $("paste-go").className = open ? "primary" : "secondary";
  $("scan").classList.toggle("primary", !open);
  $("scan").classList.toggle("secondary", open);
});
// A computer (mouse, no camera pointed at a screen) pastes the pairing
// text; scanning stays for phones and tablets.
const MOUSE_ONLY = matchMedia("(pointer: fine)").matches && !matchMedia("(any-pointer: coarse)").matches;
if (MOUSE_ONLY) {
  $("paste").open = true;
  $("scan").hidden = true;
  quoted(document.querySelector("#paste summary"), t("pasteOpen"));
  quoted(document.querySelector('[data-t="getHubBody"]'), t("getHubBodyPaste"));
}
// Step 3 says what to do with the box that is open.
const step3 = () => quoted(document.querySelector('[data-t="step3"]'), t($("paste").open ? "step3Paste" : "step3"));
step3();
// Only a box the person opened takes the focus: opened for them (no
// camera), the keyboard would cover the message saying why.
let pasteByHand = false;
document.querySelector("#paste summary").addEventListener("click", () => { pasteByHand = true; });
$("paste").addEventListener("toggle", () => {
  step3();
  const byHand = pasteByHand;
  pasteByHand = false;
  // Opened by a tap: bring the box into view and type into it.
  if ($("paste").open && byHand && !MOUSE_ONLY) {
    $("paste-text").scrollIntoView({ block: "center" });
    $("paste-text").focus({ preventScroll: true });
  }
});
// No hub yet: how to set one up, without leaving the page. The phone
// can't run the command, so the first choice sends it to the computer.
const INSTALL_CMD = "curl -fsSL https://raw.githubusercontent.com/wensuper/OmniLLM/main/install.sh | sh";
// The install section of the README in the page's language.
const GUIDE_URL = lang === "zh" ? "https://github.com/wensuper/OmniLLM/blob/main/README.zh-CN.md#安装" : "https://github.com/wensuper/OmniLLM#installation";
document.querySelector(".wl-guide").href = GUIDE_URL;
$("no-hub").addEventListener("click", () => {
  const open = $("get-hub").hidden;
  $("get-hub").hidden = !open;
  $("no-hub").setAttribute("aria-expanded", String(open));
  // Instant: a smooth scroll is skipped by some browsers, and then the tap looks dead.
  if (open) $("get-hub").scrollIntoView({ block: "nearest" });
});
// Where the phone can share, sending it to the computer is the way.
$("get-share").hidden = !navigator.share;
if (navigator.share) $("get-share").className = "primary";
$("get-share").addEventListener("click", () => {
  navigator.share({ title: "OmniLLM", text: t("getHubSend") + "\n" + INSTALL_CMD, url: GUIDE_URL }).catch(() => {});
});
$("get-copy").addEventListener("click", () => {
  const fail = () => { getSelection().selectAllChildren(document.querySelector(".wl-cmd")); toastText(t("getHubLongPress")); };
  if (!navigator.clipboard) return fail();
  navigator.clipboard.writeText(INSTALL_CMD).then(() => toastText(t("getHubCopied")), fail);
});
// welcomeNote: a calm notice at the top of the welcome screen (removed,
// or remove it on the Mac too); the paste box folded so it reads first.
// While it shows, the welcome body (how to add a device) steps aside.
// The three steps stay (they are how to add it again); the notice sits
// right under the top bar, and "No OmniLLM yet?" makes no sense then.
function welcomeNote(key) {
  $("welcome-note").hidden = !key;
  quoted($("welcome-note"), key ? t(key) : "");
  $("no-hub").hidden = !!key;
  if (key) { $("get-hub").hidden = true; $("no-hub").setAttribute("aria-expanded", "false"); }
  if (key && !MOUSE_ONLY) $("paste").open = false;
  if (key) { $("welcome").scrollTop = 0; window.scrollTo(0, 0); $("welcome-note").tabIndex = -1; $("welcome-note").focus({ preventScroll: true }); }
}

// ---------- pairing ----------
// Add waits for something to add.
const pasteReady = () => { $("paste-go").disabled = !$("paste-text").value.trim(); $("welcome-err").textContent = ""; }; // pair() explains anything that isn’t pairing text
$("paste-text").addEventListener("input", pasteReady);
pasteReady();
$("paste-go").addEventListener("click", () => {
  const v = $("paste-text").value.trim();
  if (!v) return;
  // Pasted here by the person: a link or the bare text goes straight to
  // the 6 digits, as a scan does. Only replacing a pairing still asks.
  pair(v, false);
});
// Back from pairing: the chat (opened now when this start went straight
// to a pairing link), else the welcome.
function leavePairing() {
  if (!S.device) return show("welcome");
  if (S.cur) return show("chat");
  openChat();
}
$("pair-retry").addEventListener("click", leavePairing);
// Different numbers (someone else's pairing), or Cancel while connecting:
// stop it and go back. A phone that already left its old pairing has none
// to go back to.
function stopPairing() {
  S.pairStopped = true;
  try { omnillm.pairCancel(); } catch {}
  sasShown(false); $("pair-stop").hidden = true;
  if (S.leftOld) show("welcome"); else leavePairing();
}
$("pair-mismatch").addEventListener("click", stopPairing);
$("pair-stop").addEventListener("click", stopPairing);
function sasShown(on) { S.sasUp = on; $("sas").hidden = $("pair-wait").hidden = $("pair-mismatch").hidden = !on; }

// pair connects this device. A link opened in the browser (fromLink) could
// have been sent by anyone, so the person first sees which remote access
// it leads through and taps Add; text they pasted here themselves (a link
// or the bare pairing text) needs no extra tap.
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
      if (!go) { leavePairing(); return; }
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
    const dropped = e && e.code === "hub_offline" && !relayIsPlain(text);
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
    return computerWords("", e.message); // the program's words: "your Mac" only on a Mac
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
  if (!(await loadJsQR().then(() => true, () => false))) { $("welcome-err").textContent = t("scanFailed"); $("paste").open = true; return; }
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
  $("install-tip").hidden = isStandalone || MOUSE_ONLY;
  setSide(localStorageGet("side") === "off");
  if (!S.cur) {
    let cid = null;
    try { cid = sessionStorage.getItem("omnillm.cur"); sessionStorage.removeItem("omnillm.cur"); } catch {}
    S.cur = S.chats.find((c) => c.id === cid) || (cid ? newChat() : S.chats[0]) || newChat(); // a new, unsaved chat stays new
  }
  try { const d = sessionStorage.getItem("omnillm.draft"); if (d) { $("input").value = d; sessionStorage.removeItem("omnillm.draft"); grow(); } } catch {}
  renderList(); renderLog(); sendState();
  await keptModels();
  await connect();
}

// connect opens the device and loads the models; S.connecting is that
// work, so a message sent meanwhile waits for it instead of going nowhere.
function connect() {
  S.connecting = (async () => {
    if (!navigator.onLine) { setOnline(false, t("noNetwork")); return; }
    const sel = $("model");
    if (!S.models.length) { sel.textContent = ""; const o = el("option", t("loadingModels")); o.value = ""; sel.append(o); paintPick(); }
    liveDot("wait");
    S.dialing = true;
    try {
      // Open once per pairing: reopening closes the old device, and with it
      // a reply still on its way (the network coming back fires "online"
      // mid-reply). An open device reconnects by itself.
      if (S.openedState !== S.device.state) { await omnillm.open(S.device.state); S.openedState = S.device.state; }
      await loadModels();
      S.retry = 0;
      setOnline(true);
    } catch (e) {
      if (e && e.msg === "device_revoked") return unpaired("removedHere");
      if (e && e.code === "device_paused") return setPaused(errText(e));
      setOnline(false, errText(e));
      // A failed try is often one lost packet on a slow network: try again
      // soon (1, 2, 4 s), then every 10 s as before.
      const n = S.retry || 0;
      if (n < 3) {
        S.retry = n + 1;
        setTimeout(() => { if (S.device && !S.quiet && !S.online && !S.dialing && !document.hidden) connect(); }, 1000 << n);
      }
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

// isSub: a Claude, ChatGPT or Google AI subscription (the official programs
// on the hub), not merely a model kept for people (a local model can be that too).
function isSub(m) { const k = m.kind || m.provider; return k === "claude-code" || k === "codex" || k === "antigravity"; }
// isLocal: runs on the Mac itself. The hub tells a device every model is
// remote (the data leaves the phone), so this goes by the provider: local
// models come through Ollama.
function isLocal(m) { return !!m.onHost; } // runs on the Mac itself (the hub says so)
function group(m) { return isSub(m) ? 0 : isLocal(m) ? 2 : 1; }
function modelName(id) { const m = S.models.find((x) => x.id === id); return m ? m.name : id; }
// What keeps a model from working right now, as the Mac last said: signed
// out of its program, or at its subscription's limit (until when). Shown
// before anything is sent: a dot in the picker, a line under the name, and
// a notice above the box while it is the chosen one.
function problemOf(m) {
  if (!m || !m.problem) return null;
  const sub = { codex: ["subChatGPT", "chatgpt", "ChatGPT"], antigravity: ["subGemini", "gemini", "Gemini"] }[m.kind] || ["subClaude", "claude", "Claude"], src = t(sub[0]);
  if (m.problem === "login") return { short: t("probLogin"), long: t("probLoginNote", src, sub[1], sub[2]) };
  if (m.problem === "limit") {
    const at = m.resetsAt ? new Date(m.resetsAt * 1000) : null;
    if (at && at <= new Date()) return null;
    if (!at) return { short: t("probLimitNoTime"), long: t("probLimitNoteNoTime", src) };
    const today = at.toDateString() === new Date().toDateString();
    const when = at.toLocaleString(document.documentElement.lang, today ? { hour: "2-digit", minute: "2-digit" } : { weekday: "short", hour: "2-digit", minute: "2-digit" });
    return { short: t("probLimit", when), long: t("probLimitNote", src, when) };
  }
  return null;
}
const chosenProblem = () => problemOf(S.models.find((m) => m.id === $("model").value));
// label: the name in this page's language. The Mac sends the name without
// language-built parts and says what to add: "latest" (the version isn't
// known yet) and the kind (Codex). A name the owner chose stays as it is.
function label(m) {
  if (m.renamed) return m.name;
  if (m.kind === "codex") return m.version ? t("codexV", m.version) : t("codex");
  return m.latest ? m.name + t("latest") : m.name;
}
// family orders Claude's models Opus, Sonnet, Haiku (largest first).
// Subscriptions stay together: Claude, then ChatGPT, then Google AI
// (Antigravity also offers a Claude model; it stays with Antigravity's).
function family(m) {
  const k = m.kind || m.provider;
  if (k === "codex") return 10;
  if (k === "antigravity") return 20;
  const s = (m.id + " " + m.name).toLowerCase(), i = ["opus", "sonnet", "haiku"].findIndex((f) => s.includes(f));
  return i < 0 ? 3 : i;
}

async function loadModels() {
  const ms = await omnillm.models();
  syncName();
  for (const m of ms) m.name = label(m);
  // Subscriptions first (the owner pays for them already), then API keys,
  // then models on the Mac; the picker shows names only, grouped.
  // Codex's own models in Codex's own order (rank, from its catalog).
  ms.sort((a, b) => (group(a) - group(b)) || (group(a) === 0 && family(a) - family(b)) || ((a.rank || 0) - (b.rank || 0)) || a.name.localeCompare(b.name));
  S.models = ms; S.listed = true;
  // Kept for the next start: the picker shows them at once while the
  // connection to the Mac opens (seconds on a slow network).
  if (S.device) seal("models", { added: S.device.added, ms }).catch(() => {});
  fillPicker(ms);
  macLang();
  const os = omnillm.hubOS ? omnillm.hubOS() : "";
  if (os && os !== hubOS) {
    hubOS = os; localStorageSet("hubos", os);
    // Labels drawn while the hub's system wasn't known yet say it right now.
    document.querySelectorAll("#chat [data-t]").forEach((e) => { quoted(e, t(e.dataset.t)); });
    document.querySelectorAll("#chat [data-ta]").forEach((e) => { e.setAttribute("aria-label", t(e.dataset.ta)); e.title = t(e.dataset.ta); });
    if (S.cur && !S.cur.messages.length) renderLog(true);
  }
  if (!ms.length) { const sel = $("model"), o = el("option", t("noModelsOpt")); o.value = ""; o.disabled = true; sel.append(o); sel.value = ""; paintPick(); return; }
  if (pickModel()) renderLog(true);
  if (S.cur) renderList(); // the drawer names models by name now
}
// fillPicker lists the models, grouped (subscriptions, keys, the Mac's own).
function fillPicker(ms) {
  const sel = $("model"); sel.textContent = "";
  let og = null, g = -1;
  for (const m of ms) {
    if (group(m) !== g) { g = group(m); og = el("optgroup"); og.label = t(["grpSub", "grpKey", "grpLocal"][g]); sel.append(og); }
    const o = el("option", m.name); o.value = m.id; og.append(o);
  }
}
// keptModels shows the models the Mac listed last time, before the
// connection is up: the person can type, pick and send at once (a message
// waits for the connection), and the dot pulses until the Mac answers.
async function keptModels() {
  if (S.models.length || !S.device) return;
  const k = await unseal("models").catch(() => null);
  if (!k || k.added !== S.device.added || !Array.isArray(k.ms) || !k.ms.length || S.models.length) return;
  S.models = k.ms;
  fillPicker(k.ms);
  const want = [S.cur && S.cur.model, localStorageGet("model"), "claude-code/sonnet"].find((id) => id && k.ms.some((m) => m.id === id));
  $("model").value = want || k.ms[0].id;
  paintPick();
}
// macLang: the language the Mac speaks (the owner's choice there, else the
// Mac's own) is kept for the next start; when it changes and nothing was
// chosen here, the page reloads in it once idle. A browser that can't keep
// it (no storage) never reloads for it, so it can't loop.
function macLang() {
  const ml = oneLang(omnillm.lang ? omnillm.lang() : "");
  if (ml === oneLang(localStorageGet("maclang"))) return;
  localStorageSet("maclang", ml);
  if (oneLang(localStorageGet("maclang")) !== ml || oneLang(localStorageGet("lang"))) return;
  if ((ml || browserLang) !== lang) { S.relang = true; reloadIfIdle(); }
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
  paintPick();
  if (!want || want === sel.value) return false;
  if (S.cur.messages.length) S.cur.gone = t("modelGone", modelName(sel.value));
  S.cur.model = sel.value;
  return true;
}
$("model").addEventListener("change", () => { localStorageSet("model", $("model").value); if (S.cur) S.cur.model = $("model").value; paintPick(); refreshActs(); paintBanner(); });
function localStorageGet(k) { try { return localStorage.getItem("omnillm." + k); } catch { return null; } }
function localStorageSet(k, v) { try { localStorage.setItem("omnillm." + k, v); } catch {} }

// liveDot: green when the Mac answered (even with no models: it is
// there), amber while the Mac has paused this device, grey when it can't
// be reached, pulsing grey while connecting.
function liveDot(state) {
  const label = t({ on: "liveOn", off: "liveOff", paused: "pausedDot" }[state] || "liveWait");
  for (const id of ["live-dot", "pick-dot"]) { const d = $(id); d.className = "live-dot " + state; d.title = label; }
  const pr = state === "on" && chosenProblem();
  if (pr) { $("pick-dot").className = "live-dot prob"; $("pick-dot").title = pr.short; }
  $("live-dot").setAttribute("aria-label", label);
  const said = t({ on: "macOn", off: navigator.onLine ? "macOff" : "macNoNet", paused: "macPaused" }[state] || "macWait");
  $("mac-state").textContent = said;
  $("mac-head").textContent = t("yourMac") + " · " + said;
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
  if (!S.models.length) { const sel = $("model"); sel.textContent = ""; const o = el("option", t("paused")); o.value = ""; sel.append(o); paintPick(); }
  paintBanner(); sendState(); refreshActs();
}
// paintBanner: paused, offline (why, and that the text is kept if a send
// was blocked), or a Mac with no models; Retry only when this device has
// a network (without one it can't help; the page retries by itself).
function paintBanner() {
  // The notice sits above the composer: the talk keeps its bottom in view.
  const stick = !$("chat").hidden && nearBottom();
  try { paintNotice(); } finally { if (stick) toBottom(); }
}
function paintNotice() {
  liveDot(!S.online ? "off" : S.paused ? "paused" : "on");
  const b = $("banner"), none = S.online && !S.paused && S.listed && !S.models.length;
  b.textContent = ""; b.hidden = S.online && !none && !S.paused;
  if (S.paused && S.online) { b.append(quoted(el("span"), S.paused)); return; }
  if (none) { b.append(quoted(el("span"), t("noModels"))); return; }
  const pr = S.online && !S.paused && chosenProblem();
  if (pr) { b.hidden = false; b.append(quoted(el("span"), pr.long)); return; }
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
function startNew() {
  if (S.busy) return toastText(t("waitReply"));
  S.cur = newChat(); renderLog(); renderList(); closeDrawer(); $("input").focus();
}
for (const id of ["new", "new-side", "new-float"]) $(id).addEventListener("click", startNew);

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
  $("chat").classList.toggle("fresh", !msgs.length);
  paintTitle();
  if (!msgs.length) log.append(hero());
  // Who answered is said once, then again where the model changes: the
  // first reply is headed with its model's name, a change gets a line
  // ("Switched to X"), and every other reply names its model in the row
  // under it. Compared by model id (the name gains its version once
  // known). An earlier answer (kept by "answer again") folds above the new
  // one and doesn't count.
  let last = "", userEl = null;
  S.lastWho = false;
  msgs.forEach((m, i) => {
    if (m.prev) { log.append(earlier(m)); return; }
    let who = false;
    if (m.role === "assistant" && m.by) {
      const key = m.model || m.by;
      if (last && key !== last) {
        const line = switchLine(t("switchedTo", m.by));
        if (userEl && !(msgs[i - 1] && msgs[i - 1].prev)) userEl.before(line); else log.append(line);
      }
      who = !last;
      last = key;
    }
    const isLast = i === msgs.length - 1;
    if (isLast) S.lastWho = who;
    const b = bubble(m, isLast, who);
    if (m.role === "user") userEl = b;
    log.append(b);
    if (S.cur.long === i + 1) log.append(switchLine(t(S.cur.longSub ? "longChatSub" : "longChat")));
  });
  const row = actsRow();
  if (row) log.append(row);
  if (S.cur.gone) log.append(switchLine(S.cur.gone));
  if (stay) log.scrollTop = top; else toBottom();
}
// hero: a new chat says what it is for (and, just paired, that the Mac is there).
function hero() {
  const h = el("div", undefined, "hero"), img = el("img");
  img.src = "icon.svg"; img.alt = "";
  h.append(img, el("h1", t("empty")));
  if (S.hello === S.cur.id) h.append(el("p", t("connectedMac"), "hello"));
  return h;
}
// paintTitle: the chat's title in the bar (phones), at the top of the talk
// (wide screens, once the chat has begun) and in the browser tab. Both
// titles rename the chat when clicked.
function paintTitle() {
  const title = S.cur && S.cur.title;
  const begun = !!(S.cur && S.cur.messages.length);
  titleButton($("bar-title"), title || t("newChat"), begun);
  $("chat-head").hidden = !begun;
  if (begun) titleButton($("head-title"), title || t("newChat"), true);
  $("chat-head").classList.toggle("lifted", $("log").scrollTop > 4);
  document.title = title ? title + " · OmniLLM" : "OmniLLM";
}
// titleButton: the name, and a small chevron when it opens the chat's menu
// (Rename, Delete). Left as it is while the name is being edited.
function titleButton(b, text, menu) {
  if (b.querySelector("input")) return;
  b.textContent = "";
  b.append(el("span", text, "head-name"));
  if (menu) b.append(icon("i-chev s14 head-chev"));
  b.disabled = !menu && b.id === "head-title";
  b.title = menu ? t("chatMore") : "";
}
// renameIn turns a title (the button itself, or a list row's button) into
// a text field: Enter or leaving it keeps the new name, Esc goes back. An
// empty name keeps the old one.
function renameIn(btn, chat) {
  if (!chat || !chat.messages.length || btn.querySelector("input")) return;
  const old = chat.title || "";
  const f = el("input", undefined, "rename-in");
  f.type = "text"; f.value = old; f.maxLength = 80; f.setAttribute("aria-label", t("renameChat"));
  btn.textContent = ""; btn.append(f);
  f.focus(); f.select();
  let done = false;
  const finish = async (keep) => {
    if (done) return; done = true;
    const v = f.value.trim();
    if (keep && v && v !== old) {
      chat.title = v;
      if (!S.quiet) await seal("chats", S.chats.some((c) => c.id === chat.id) ? S.chats : [chat, ...S.chats]).catch(() => {});
    }
    f.remove();
    paintTitle(); renderList();
  };
  f.addEventListener("keydown", (e) => {
    if (e.isComposing || e.keyCode === 229) return;
    if (e.key === "Enter") { e.preventDefault(); finish(true); }
    else if (e.key === "Escape") { e.preventDefault(); e.stopPropagation(); finish(false); }
  });
  f.addEventListener("blur", () => finish(true));
  // The field sits inside the title's button: its clicks must not reach it.
  f.addEventListener("click", (e) => e.stopPropagation());
}
// The title at the top opens the chat's menu; a double click renames it at once.
for (const id of ["head-title", "bar-title"]) {
  const b = $(id);
  b.addEventListener("click", (e) => {
    if (!S.cur || !S.cur.messages.length || b.querySelector("input")) return;
    e.stopPropagation();
    rowMenu(b.parentElement, b, b, S.cur);
  });
  b.addEventListener("dblclick", (e) => { e.preventDefault(); closeRowMenu(); renameIn(b, S.cur); });
}
// The bar under the title gets a hairline once the talk scrolls under it.
$("log").addEventListener("scroll", () => $("chat-head").classList.toggle("lifted", $("log").scrollTop > 4), { passive: true });
function icon(cls) { const i = el("span", "", "ic " + cls); i.setAttribute("aria-hidden", "true"); return i; }
function iconButton(cls, label) {
  const b = el("button", undefined, "ibtn"); b.type = "button";
  b.setAttribute("aria-label", label); b.title = label; b.append(icon(cls));
  return b;
}
// actsRow: under the last turn, once it is over: copy, and answer again
// (with the model now selected, so it doubles as "ask another model"); a
// reply that failed or was stopped, or a question left without one,
// offers Retry. Answering needs the Mac: offline or paused, it rests.
function actsRow() {
  const msgs = S.cur ? S.cur.messages : [], end = msgs[msgs.length - 1];
  if (!end || end.live || S.busy || !(end.role === "user" || end.content || end.note)) return null;
  const row = el("div", undefined, "acts");
  if (end.role === "assistant" && end.content) row.append(copyButton(end.content));
  const retry = end.role === "user" || end.failed, pick = $("model").value;
  const label = retry ? t("retry") : pick && end.model && pick !== end.model ? t("againWith", modelName(pick)) : t("again");
  let again;
  if (retry) { again = el("button", undefined, "retry"); again.type = "button"; again.append(icon("i-again s14"), label); }
  else again = iconButton("i-again s16", label);
  again.disabled = !S.online || !!S.paused;
  again.addEventListener("click", answerAgain);
  row.append(again);
  if (end.role === "assistant" && end.by && !S.lastWho) row.append(el("span", end.by, "by"));
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
  const tg = el("button", undefined, "earlier-toggle"); tg.type = "button";
  tg.append(icon("i-chev s14"), t(open ? "earlierOpen" : "earlier", m.by || modelName(m.model)));
  tg.setAttribute("aria-expanded", String(!!open));
  tg.addEventListener("click", () => {
    S.opened ??= new WeakSet();
    if (S.opened.has(m)) S.opened.delete(m); else S.opened.add(m);
    box.replaceWith(earlier(m));
  });
  box.append(tg);
  if (open) box.append(bubble(m, false, true));
  return box;
}
function switchLine(text) { const line = el("div", undefined, "switch-line"); line.append(el("span", text)); return line; }
// copyButton copies text; the icon turns into a tick for a moment.
function copyButton(text) {
  const cp = iconButton("i-copy s16", t("copy"));
  cp.addEventListener("click", () => copyText(text, () => {
    cp.classList.add("done"); cp.firstChild.className = "ic i-check s16"; cp.title = t("copied");
    setTimeout(() => { cp.classList.remove("done"); cp.firstChild.className = "ic i-copy s16"; cp.title = t("copy"); }, 1200);
  }));
  return cp;
}
function copyText(text, done) {
  if (!navigator.clipboard) return;
  navigator.clipboard.writeText(text).then(done, () => {});
}

// bubble renders a message; fenced code becomes a code block, the rest
// stays text. who heads a reply with its model's name. Until a reply's
// first words arrive it shows "connecting to your Mac", then that model
// thinking (an orbiting core, as in the logo). Earlier replies carry their
// own row (Copy and who answered), shown on hover or a tap; the last
// reply's row sits under it in the log instead.
function bubble(m, isLast, who) {
  const b = el("div", undefined, "msg " + m.role + (isLast ? " last" : ""));
  if (m.role === "assistant" && m.by && who) {
    const head = el("div", undefined, "who");
    const orb = el("span", "", "orb"); orb.setAttribute("aria-hidden", "true");
    head.append(orb, el("span", m.by, "by"));
    b.append(head);
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
  if (m.note) {
    // Stopped or cut short is a quiet line; anything else went wrong (red).
    const quietNote = [T.zh.stopped, T.en.stopped, T.zh.cut, T.en.cut].includes(m.note);
    b.append(quoted(el("p", undefined, "note" + (quietNote ? " quiet" : "")), m.note.replace(/^[（(]|[）)]$/g, "")));
  }
  if (m.role === "assistant" && m.content && !m.live && !isLast) {
    const row = el("div", undefined, "acts");
    row.append(copyButton(m.content));
    if (m.by && !who) row.append(el("span", m.by, "by"));
    b.append(row);
    // A touch screen has no hover: a tap on the reply shows its row.
    b.addEventListener("click", (e) => { if (!e.target.closest("button, a, pre")) b.classList.toggle("tapped"); });
  }
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
  // Thinking from the moment it is sent: the Mac's receipt comes a round
  // trip later and is no step of the person's (a lost connection says so
  // in the banner and the reply's error, not here).
  w.append(orb, el("span", t("thinking"), "think-text"));
  const dots = el("span", undefined, "dots"); dots.setAttribute("aria-hidden", "true");
  dots.append(el("i"), el("i"), el("i"));
  w.append(dots);
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
    if (i % 2) b.append(codeBlock(p));
    else if (p.trim()) prose(b, p.replace(/^\n+|\n+$/g, ""));
  });
}
// codeBlock: the code with a head naming its language and a Copy button.
function codeBlock(p) {
  const lang = (p.match(/^([\w+-]*)\n/) || [])[1] || "";
  const code = p.replace(/^[\w+-]*\n/, "").replace(/\n+$/, "");
  const box = el("div", undefined, "code"), head = el("div", undefined, "code-h"), pre = el("pre");
  const cp = el("button", undefined); cp.type = "button";
  const word = el("span", t("copyCode"));
  cp.append(icon("i-copy s14"), word);
  cp.addEventListener("click", () => copyText(code, () => { word.textContent = t("copied"); setTimeout(() => { word.textContent = t("copyCode"); }, 1200); }));
  head.append(el("span", lang), cp);
  pre.append(el("code", code));
  box.append(head, pre);
  return box;
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

// ---------- the chat list ----------
// Newest first, under Today, Yesterday, Previous 7 days and Earlier.
function dayGroup(ts) {
  const day = (d) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
  const n = Math.round((day(new Date()) - day(new Date(ts))) / 864e5);
  return n <= 0 ? 0 : n === 1 ? 1 : n < 7 ? 2 : 3;
}
// The search field filters by title and message text, here on the device.
function matches(c, q) {
  return !q || (c.title || "").toLowerCase().includes(q) || c.messages.some((m) => (m.content || "").toLowerCase().includes(q));
}
function renderList() {
  const ul = $("chat-list"); ul.textContent = "";
  const q = $("chat-search").value.trim().toLowerCase();
  const all = [S.cur, ...S.chats.filter((c) => c.id !== S.cur.id)].filter((c) => (c.messages.length || c === S.cur) && matches(c, q));
  // A chat not yet begun counts as now.
  const at = (c) => (c.messages.length ? c.updated : Infinity);
  all.sort((a, b) => at(b) - at(a));
  if (!all.length) ul.append(el("li", t("noMatch"), "muted small no-match"));
  let g = -1;
  for (const c of all) {
    const cg = c.messages.length ? dayGroup(c.updated) : 0;
    if (cg !== g) { g = cg; ul.append(el("li", t(["grpToday", "grpYesterday", "grpWeek", "grpOlder"][g]), "grp")); }
    const li = el("li", undefined, "row" + (c.id === S.cur.id ? " cur" : ""));
    const a = el("button", c.title || t("newChat")); a.type = "button"; a.title = c.title || "";
    if (c.id === S.cur.id) a.setAttribute("aria-current", "true");
    a.addEventListener("click", () => {
      // Two clicks on the same chat within 400 ms rename it. The first one
      // redraws the list (it opens the chat), so the browser's own dblclick
      // would land on a button that is gone.
      const now = Date.now(), last = rowClick;
      rowClick = { id: c.id, at: now };
      if (last.id === c.id && now - last.at < 400 && c.messages.length) { rowClick = { id: "", at: 0 }; return renameIn(a, c); }
      if (c === S.cur) { closeDrawer(); return; }
      if (S.busy && c !== S.cur) { closeDrawer(); return toastText(t("waitReply")); }
      S.cur = c; if (S.models.length) pickModel(); renderLog(); renderList(); closeDrawer();
    });
    li.append(a);
    if (c.messages.length) {
      // ⋯: Rename and Delete, as in other chat apps.
      const x = iconButton("i-more s16", t("chatMore")); x.classList.add("chat-del");
      x.setAttribute("aria-haspopup", "menu"); x.setAttribute("aria-expanded", "false");
      x.addEventListener("click", (e) => { e.stopPropagation(); rowMenu(li, x, a, c); });
      li.append(x);
    }
    ul.append(li);
  }
}

let rowClick = { id: "", at: 0 }; // the last click on a chat row (double click renames)
// rowMenu opens a chat's small menu (Rename, Delete) under the button that
// asked for it: a list row's ⋯, or the title at the top.
function rowMenu(li, btn, title, c) {
  const open = document.querySelector(".row-menu");
  if (open) { const same = open.parentElement === li; closeRowMenu(); if (same) return; }
  const m = el("div", undefined, "pop row-menu"); m.setAttribute("role", "menu");
  const item = (ic, label, run, bad) => {
    const b = el("button", undefined, "pop-item" + (bad ? " danger" : "")); b.type = "button"; b.setAttribute("role", "menuitem");
    b.append(icon(ic + " s16"), el("span", label, "nm"));
    b.addEventListener("click", (e) => { e.stopPropagation(); closeRowMenu(); run(); });
    return b;
  };
  m.append(item("i-pen", t("rename"), () => renameIn(title, c)), item("i-trash", t("delete"), async () => {
    if (S.busy && c === S.cur) return toastText(t("waitReply"));
    if (!await ask(t("deleteChatQ", c.title || t("newChat")), t("delete"), true)) return;
    S.chats = S.chats.filter((o) => o.id !== c.id);
    if (c === S.cur) { S.cur = S.chats[0] || newChat(); renderLog(); }
    if (!S.quiet) await seal("chats", S.chats).catch(() => {});
    renderList();
  }, true));
  li.append(m); btn.setAttribute("aria-expanded", "true"); li.classList.add("menu-open");
  m.querySelector("button").focus();
}
function closeRowMenu() {
  const m = document.querySelector(".row-menu");
  if (!m) return;
  const li = m.parentElement; m.remove(); li.classList.remove("menu-open");
  li.querySelectorAll("[aria-expanded]").forEach((b) => b.setAttribute("aria-expanded", "false"));
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
input.addEventListener("input", () => { grow(); sendState(); if (V.rec) stopVoice(); });

// ---------- voice input ----------
// The browser's own speech recognition (Chrome, Edge, Safari). Words show
// in the box as they are heard, after what was already typed, and are
// never sent by themselves: the person reads, edits and sends. Tap again,
// Esc, typing, sending or a long pause ends it. No recogniser, no mic.
const Speech = window.SpeechRecognition || window.webkitSpeechRecognition;
const V = { rec: null, base: "", quiet: 0 };
const VOICE_PAUSE = 6000; // this long with nothing new heard ends it
function voiceUI(on) {
  const b = $("mic");
  b.classList.toggle("on", on);
  b.setAttribute("aria-pressed", String(on));
  b.setAttribute("aria-label", t(on ? "voiceStop" : "voice"));
  b.title = t(on ? "voiceStop" : "voice");
  $("composer").classList.toggle("listening", on);
  input.placeholder = t(on ? "listening" : "placeholder");
}
function startVoice() {
  if (V.rec || S.quiet) return;
  if (!localStorageGet("voiceSeen")) { localStorageSet("voiceSeen", "1"); toastText(t("voiceFirst")); }
  const r = new Speech();
  r.lang = lang === "zh" ? "zh-CN" : (/^en/i.test(navigator.language) ? navigator.language : "en-US");
  r.interimResults = true;
  r.continuous = true;
  // What was typed stays; the speech goes after it.
  const typed = input.value.replace(/\s+$/, "");
  V.base = typed ? typed + (lang === "zh" ? "" : " ") : "";
  const pause = () => { clearTimeout(V.quiet); V.quiet = setTimeout(stopVoice, VOICE_PAUSE); };
  // Rebuilt from every result each time (finished and still changing), so
  // a browser that reports a result twice can't double the words.
  r.onresult = (e) => {
    let heard = "";
    for (const res of e.results) heard += res[0].transcript;
    input.value = V.base + heard;
    grow(); sendState();
    input.scrollTop = input.scrollHeight;
    pause();
  };
  r.onerror = (e) => {
    const k = { "not-allowed": "voiceDenied", "service-not-allowed": "voiceDenied", "audio-capture": "voiceNoMic", "no-speech": "voiceNoSpeech", "network": "voiceNetwork", "aborted": "" }[e.error];
    if (k !== "") toastText(t(k || "voiceFailed"));
  };
  r.onend = () => {
    clearTimeout(V.quiet);
    if (V.rec !== r) return;
    V.rec = null;
    voiceUI(false);
    // Leave the caret at the end, ready to edit or send (computers only:
    // on a phone the keyboard would jump up over the words).
    if (MOUSE_ONLY) { input.focus(); input.setSelectionRange(input.value.length, input.value.length); }
  };
  try { r.start(); } catch { toastText(t("voiceFailed")); return; }
  V.rec = r;
  voiceUI(true);
  pause();
}
function stopVoice() { if (V.rec) { try { V.rec.stop(); } catch {} } }
if (Speech) {
  $("mic").hidden = false;
  voiceUI(false);
  $("mic").addEventListener("click", () => (V.rec ? stopVoice() : startVoice()));
  // A tab put away stops listening.
  document.addEventListener("visibilitychange", () => { if (document.hidden) stopVoice(); });
}
// The send button looks pressable only when there is something to send;
// while the Mac has this device paused it rests (Stop still works).
function sendState() {
  $("send").classList.toggle("idle", !S.busy && (!!S.paused || !S.online || (!input.value.trim() && !(S.pending && S.pending.length))));
  $("send").disabled = !S.busy && !!S.paused;
}
// The box grows with what is typed, up to 40% of the window, then scrolls.
function grow() { input.style.height = "auto"; input.style.height = Math.min(input.scrollHeight, Math.round(innerHeight * 0.4)) + "px"; }
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
  if (V.rec) { const r = V.rec; V.rec = null; try { r.abort(); } catch {} voiceUI(false); }
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
    // A subscription continues the program's own session on the Mac
    // (nothing is resent, nothing is billed): only speed is worth saying.
    const lm = S.models.find((x) => x.id === model);
    if (lm && isSub(lm)) chat.longSub = true;
  }
  if (S.hello === chat.id) S.hello = null;
  const reply = { role: "assistant", content: "", by: modelName(model), model, live: true };
  S.sentAt = Date.now();
  chat.messages.push(reply);
  // Saved now: the page may be thrown away (a phone in the background)
  // before the reply is done; the question must not go with it.
  saveChats().catch(() => {});
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
    const r = await omnillm.chat(id, model, JSON.stringify(msgs), (d, exact) => {
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
      if (reply.note && !reply.content && !quiet && reply.note !== t("stopped")) { $("log").append(quoted(el("p", undefined, "note err"), reply.note)); if (near) toBottom(); }
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

function busyUI(on) {
  $("send").classList.toggle("stop", on); $("send").setAttribute("aria-label", t(on ? "stop" : "send"));
  $("send").title = MOUSE_ONLY ? t(on ? "stop" : "send") + (on ? " (Esc)" : " (Enter)") : t(on ? "stop" : "send");
  sendState(); paintPick();
}

// ---------- the chat list: a drawer on phones, a sidebar on wide screens ----------
const WIDE = matchMedia("(min-width: 1024px)");
const overlay = () => !WIDE.matches;
$("menu").addEventListener("click", () => { $("drawer").hidden = false; });
$("drawer").addEventListener("click", (e) => { if (e.target.id === "drawer") closeDrawer(); });
// closeDrawer: a drawer closes (and forgets its search); a sidebar stays.
function closeDrawer() {
  macMenu(false);
  if (!overlay()) return;
  $("drawer").hidden = true;
  if ($("chat-search").value) { $("chat-search").value = ""; if (S.cur) renderList(); }
}
// setSide hides or shows the sidebar on a wide screen (remembered here).
function setSide(off) { $("chat").classList.toggle("side-off", off); localStorageSet("side", off ? "off" : ""); }
$("side-hide").addEventListener("click", () => setSide(true));
$("side-show").addEventListener("click", () => setSide(false));

// "Your Mac": the state, and a menu with this device, the language,
// shortcuts, where chats are kept, and removing this device.
function macMenu(open) {
  $("mac-menu").hidden = !open; $("mac").setAttribute("aria-expanded", String(open));
  $("lang-list").hidden = true; $("lang-btn").setAttribute("aria-expanded", "false");
  if (open) paintLang();
}
$("mac").addEventListener("click", () => macMenu($("mac-menu").hidden));
$("lang-btn").addEventListener("click", () => {
  const open = $("lang-list").hidden;
  $("lang-list").hidden = !open; $("lang-btn").setAttribute("aria-expanded", String(open));
});
function paintLang() {
  const chosen = oneLang(localStorageGet("lang"));
  $("lang-now").textContent = chosen ? (chosen === "zh" ? "中文" : "English") : t("langMac");
  document.querySelectorAll("#lang-list [data-lang]").forEach((b) => b.setAttribute("aria-checked", String(b.dataset.lang === chosen)));
}
// A language chosen here is kept on this device only; the page reloads in
// it (what was typed stays).
document.querySelectorAll("#lang-list [data-lang]").forEach((b) => b.addEventListener("click", () => {
  if (S.busy || (S.pending && S.pending.length)) return toastText(t(S.busy ? "waitReply" : "langPending"));
  if (S.volatile) return toastText(t("langVolatile"));
  localStorageSet("lang", b.dataset.lang);
  saveForReload();
  location.reload();
}));
$("keys-btn").addEventListener("click", () => { macMenu(false); showKeys(); });
document.addEventListener("click", (e) => {
  if (!e.target.closest(".row-menu")) closeRowMenu();
  if (!$("mac-menu").hidden && !e.target.closest(".drawer-foot")) macMenu(false);
  if (!$("mmenu").hidden && !e.target.closest(".mwrap")) pickMenu(false);
});

// ---------- the model picker ----------
// The chosen model's id stays in the hidden <select> (it holds the list,
// grouped); the button shows it and opens a list that can show who is online.
function paintPick() {
  const sel = $("model"), o = sel.options[sel.selectedIndex];
  const name = o ? o.textContent : t("noModelsOpt");
  $("pick-name").textContent = name;
  $("mpick").title = name;
  $("mpick").setAttribute("aria-label", t("model") + " · " + name);
  $("mpick").disabled = !!S.busy;
  if (S.busy && !$("mmenu").hidden) pickMenu(false);
}
function pickMenu(open) {
  const m = $("mmenu"), b = $("mpick");
  if (open && b.disabled) return;
  m.hidden = !open; b.setAttribute("aria-expanded", String(open));
  if (!open) return;
  m.textContent = "";
  const sel = $("model"), live = S.online && !S.paused;
  let n = 0;
  for (const g of sel.querySelectorAll("optgroup")) {
    if (n) m.append(el("hr"));
    m.append(el("div", g.label, "mh"));
    for (const o of g.querySelectorAll("option")) {
      const it = el("div", undefined, "mi"); it.id = "mi-" + n++; it.dataset.v = o.value;
      it.setAttribute("role", "option"); it.setAttribute("aria-selected", String(o.value === sel.value));
      const pr = problemOf(S.models.find((m) => m.id === o.value));
      const dot = el("span", "", "live-dot " + (!live ? "off" : pr ? "prob" : "on")); dot.setAttribute("aria-hidden", "true");
      const nm = el("span", undefined, "nm");
      nm.append(el("span", o.textContent, "nmx"));
      if (pr) { nm.append(el("span", pr.short, "mprob")); it.classList.add("has-prob"); }
      it.append(dot, nm);
      if (o.value === sel.value) it.append(icon("i-check s16"));
      it.addEventListener("click", () => choosePick(o.value));
      m.append(it);
    }
  }
  if (!n) m.append(el("div", sel.options[0] ? sel.options[0].textContent : t("noModelsOpt"), "none"));
  pickMove(m.querySelector('[aria-selected="true"]') || m.querySelector(".mi"));
  m.focus({ preventScroll: true });
}
function pickMove(it) {
  const m = $("mmenu");
  m.querySelectorAll(".mi.act").forEach((x) => x.classList.remove("act"));
  if (!it) { m.removeAttribute("aria-activedescendant"); return; }
  it.classList.add("act"); m.setAttribute("aria-activedescendant", it.id);
  // Scroll the list only (scrollIntoView would move the chat behind it too).
  if (it.offsetTop < m.scrollTop) m.scrollTop = it.offsetTop - 6;
  else if (it.offsetTop + it.offsetHeight > m.scrollTop + m.clientHeight) m.scrollTop = it.offsetTop + it.offsetHeight - m.clientHeight + 6;
}
function choosePick(v) {
  const sel = $("model");
  if (v && sel.value !== v) { sel.value = v; sel.dispatchEvent(new Event("change")); }
  pickMenu(false); paintPick();
  if (MOUSE_ONLY) input.focus();
}
$("mpick").addEventListener("click", () => pickMenu($("mmenu").hidden));
$("mpick").addEventListener("keydown", (e) => { if (e.key === "ArrowUp" || e.key === "ArrowDown") { e.preventDefault(); pickMenu(true); } });
$("mmenu").addEventListener("keydown", (e) => {
  const items = [...$("mmenu").querySelectorAll(".mi")], cur = $("mmenu").querySelector(".mi.act"), i = items.indexOf(cur);
  if (e.key === "ArrowDown" || e.key === "ArrowUp") { e.preventDefault(); if (items.length) pickMove(items[(i + (e.key === "ArrowDown" ? 1 : items.length - 1)) % items.length]); }
  else if (e.key === "Home" || e.key === "End") { e.preventDefault(); pickMove(e.key === "Home" ? items[0] : items[items.length - 1]); }
  else if (e.key === "Enter" || e.key === " ") { e.preventDefault(); if (cur) choosePick(cur.dataset.v); }
  else if (e.key === "Tab") pickMenu(false);
});

// ---------- keyboard (computers) ----------
// ⌘ on a Mac, Ctrl elsewhere. Keys the browser keeps (⌘N, ⌘T, ⌘W, ⌘⇧C)
// are left alone.
const KEYS = [["kNew", "⌘⇧O"], ["kSearch", "⌘K"], ["kSide", "⌘\\"], ["kStop", "Esc"], ["kCopyLast", "⌘⇧;"], ["kFocus", "/"], ["kSend", "Enter"], ["kLine", "⇧Enter"], ["shortcuts", "⌘/"]];
function showKeys() {
  const tb = $("keys-table"); tb.textContent = "";
  for (const [k, key] of KEYS) { const tr = el("tr"); const td = el("td"); td.append(el("kbd", keyName(key))); tr.append(el("td", t(k)), td); tb.append(tr); }
  $("keys").showModal(); $("keys-ok").focus();
}
$("keys-ok").addEventListener("click", () => $("keys").close());
function copyLast() {
  const m = S.cur && [...S.cur.messages].reverse().find((x) => x.role === "assistant" && x.content && !x.live && !x.prev);
  if (m) copyText(m.content, () => toastText(t("copied")));
}
const typing = (n) => n && n.closest && n.closest("input, textarea, select, [contenteditable]");
document.addEventListener("keydown", (e) => {
  if ($("chat").hidden || S.quiet || e.isComposing || e.keyCode === 229) return;
  const mod = MAC ? e.metaKey : e.ctrlKey;
  if (e.key === "Escape") {
    if (document.querySelector(".row-menu")) {
      e.preventDefault();
      const m = document.querySelector(".row-menu"), opener = m && m.parentElement.querySelector('[aria-expanded="true"]');
      closeRowMenu();
      if (opener) opener.focus();
      return;
    }
    if (V.rec) { e.preventDefault(); stopVoice(); return; }
    if (!$("mmenu").hidden) { e.preventDefault(); pickMenu(false); $("mpick").focus(); return; }
    if (!$("mac-menu").hidden) { e.preventDefault(); macMenu(false); $("mac").focus(); return; }
    if (overlay() && !$("drawer").hidden) { closeDrawer(); return; }
    if (S.busy && !document.querySelector("dialog[open]") && e.target.id !== "chat-search") { e.preventDefault(); omnillm.cancel(S.busy); }
    return;
  }
  if (document.querySelector("dialog[open]")) return;
  const k = e.key.toLowerCase();
  if (mod && e.shiftKey && k === "o") { e.preventDefault(); startNew(); }
  else if (mod && !e.shiftKey && k === "k") { e.preventDefault(); if (overlay()) $("drawer").hidden = false; else setSide(false); $("chat-search").focus(); $("chat-search").select(); }
  else if (mod && !e.shiftKey && e.key === "\\") { e.preventDefault(); if (!overlay()) setSide(!$("chat").classList.contains("side-off")); }
  else if (mod && e.shiftKey && e.code === "Semicolon") { e.preventDefault(); copyLast(); }
  else if (mod && !e.shiftKey && e.key === "/") { e.preventDefault(); showKeys(); }
  else if (e.key === "/" && !mod && !e.altKey && !typing(e.target)) { e.preventDefault(); input.focus(); }
});

// ---------- pictures dropped onto the page ----------
let dragDepth = 0;
const hasFiles = (e) => e.dataTransfer && [...e.dataTransfer.types].includes("Files");
$("chat").addEventListener("dragenter", (e) => { if (!hasFiles(e) || S.quiet) return; e.preventDefault(); dragDepth++; $("drop").hidden = false; });
$("chat").addEventListener("dragover", (e) => { if (hasFiles(e)) e.preventDefault(); });
$("chat").addEventListener("dragleave", () => { if (--dragDepth <= 0) { dragDepth = 0; $("drop").hidden = true; } });
$("chat").addEventListener("drop", (e) => {
  if (!hasFiles(e)) return;
  e.preventDefault(); dragDepth = 0; $("drop").hidden = true;
  const files = [...e.dataTransfer.files].filter((f) => f.type.startsWith("image/"));
  if (files.length) addImages(files);
});
// A file dropped elsewhere must not open in place of the page.
addEventListener("dragover", (e) => { if (hasFiles(e)) e.preventDefault(); });
addEventListener("drop", (e) => { if (hasFiles(e)) e.preventDefault(); });

$("chat-search").placeholder = t("searchChats"); $("chat-search").setAttribute("aria-label", t("searchChats"));
$("chat-search").addEventListener("input", () => { if (S.cur) renderList(); });
$("drawer-close").setAttribute("aria-label", t("close")); $("drawer-close").title = t("close");
$("drawer-close").addEventListener("click", closeDrawer);
$("install-tip").addEventListener("click", () => { macMenu(false); ask(t(/Android/.test(navigator.userAgent) ? "addHomeTipAndroid" : "addHomeTip"), t("gotIt")); });
// Remove this device: on the Mac too (so it leaves the device list), then
// here. If the Mac can't be reached it is still removed here, and the
// welcome screen says to remove it on the Mac as well.
$("forget").addEventListener("click", async () => {
  if (!await ask(t("forgetQ"), t("remove"), true) || !S.device) return; // removed on the Mac meanwhile
  const btn = $("forget"), word = btn.querySelector(".nm"); btn.disabled = true; word.textContent = t("removing");
  if (S.busy) { try { omnillm.cancel(S.busy); } catch {} }
  let far = !navigator.onLine;
  if (!far) { try { await omnillm.leave(); } catch { far = true; } }
  btn.disabled = false; word.textContent = t("forget");
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
  await idb.del("device"); await idb.del("chats"); await idb.del("models"); localStorageSet("hubos", "");
  closeDrawer(); $("chat-search").value = ""; show("welcome");
  $("welcome-err").textContent = "";
  welcomeNote(note);
}

// Hidden while a reply comes in (a phone switching apps may throw the page
// away): keep what has arrived so far.
document.addEventListener("visibilitychange", () => { if (document.hidden && S.busy) saveChats().catch(() => {}); });
