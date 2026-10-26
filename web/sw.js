// Keeps the web client working offline and pins the files it was installed
// with: everything is served from this cache, and a new version is cached
// as a whole before it replaces the old one (no half-updated page).
const VERSION = "dev";
const FILES = ["./", "index.html", "app.js", "app.css", "wasm_exec.js", "omni.wasm", "manifest.webmanifest", "icon.svg", "icon-180.png", "icon-512.png", "vendor/jsqr.js"];
self.addEventListener("install", (e) => {
  // Cache the whole set, then check it is one build (a deploy can switch
  // mid-download); a mixed set is thrown away and tried on the next visit.
  e.waitUntil(caches.open("omnillm-" + VERSION).then(async (c) => {
    await c.addAll(FILES.map((f) => new Request(f, { cache: "reload" })));
    for (const f of ["index.html", "app.js"]) {
      const r = await c.match(f);
      if (!r || !(await r.text()).includes(VERSION)) { await caches.delete("omnillm-" + VERSION); throw new Error("mixed build"); }
    }
  }).then(() => self.skipWaiting()));
});
// Taking over open pages too (the first visit's included), so a page left
// open hears of the next version and reloads itself when idle.
self.addEventListener("activate", (e) => {
  e.waitUntil(caches.keys().then((ks) => Promise.all(ks.filter((k) => k.startsWith("omnillm-") && k !== "omnillm-" + VERSION).map((k) => caches.delete(k)))).then(() => self.clients.claim()));
});
self.addEventListener("fetch", (e) => {
  const u = new URL(e.request.url);
  if (e.request.method !== "GET" || u.origin !== location.origin) return;
  e.respondWith(caches.open("omnillm-" + VERSION).then((c) => c.match(e.request, { ignoreSearch: true }).then((r) => r || fetch(e.request))));
});
