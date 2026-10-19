// Keeps the web client working offline and pins the files it was installed
// with: everything is served from this cache, and a new version is cached
// as a whole before it replaces the old one (no half-updated page).
const VERSION = "dev";
const FILES = ["./", "index.html", "app.js", "app.css", "wasm_exec.js", "omni.wasm", "manifest.webmanifest", "icon.svg", "icon-180.png", "icon-512.png", "vendor/jsqr.js"];
self.addEventListener("install", (e) => {
  e.waitUntil(caches.open("omnillm-" + VERSION).then((c) => c.addAll(FILES.map((f) => new Request(f, { cache: "reload" })))).then(() => self.skipWaiting()));
});
self.addEventListener("activate", (e) => {
  e.waitUntil(caches.keys().then((ks) => Promise.all(ks.filter((k) => k.startsWith("omnillm-") && k !== "omnillm-" + VERSION).map((k) => caches.delete(k)))));
});
self.addEventListener("fetch", (e) => {
  const u = new URL(e.request.url);
  if (e.request.method !== "GET" || u.origin !== location.origin) return;
  e.respondWith(caches.open("omnillm-" + VERSION).then((c) => c.match(e.request, { ignoreSearch: true }).then((r) => r || fetch(e.request))));
});
