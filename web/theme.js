// Light or dark, decided before the page draws (no flash of the wrong one).
// The choice is kept on this device: "" follows the system, "light" and
// "dark" stay so, "time" goes by the local clock (dark from 19:00 to 7:00).
// app.js offers the choice and calls omniTheme.set.
(() => {
  const root = document.documentElement, dark = matchMedia("(prefers-color-scheme: dark)");
  const get = () => { try { return localStorage.getItem("omnillm.theme") || ""; } catch { return ""; } };
  const night = () => { const h = new Date().getHours(); return h >= 19 || h < 7; };
  const apply = () => {
    const c = get(), on = c === "dark" || (c === "time" ? night() : c !== "light" && dark.matches);
    root.dataset.theme = on ? "dark" : "light";
    // The browser's bar follows (one tag, without the media query).
    const metas = document.querySelectorAll('meta[name="theme-color"]');
    metas.forEach((m, i) => { if (i) m.remove(); else { m.removeAttribute("media"); m.content = on ? "#080912" : "#F6F6FB"; } });
    document.dispatchEvent(new Event("omnitheme"));
  };
  apply();
  // (Safari before 14 has only addListener.)
  if (dark.addEventListener) dark.addEventListener("change", apply); else dark.addListener(apply);
  // By the clock: checked each minute (the system's change too, should its
  // event not come), and when the page comes back.
  setInterval(apply, 60000);
  document.addEventListener("visibilitychange", () => { if (!document.hidden) apply(); });
  window.omniTheme = {
    get,
    // A choice made here fades across (unless motion is turned down).
    set(c) {
      try { localStorage.setItem("omnillm.theme", c); } catch {}
      if (!matchMedia("(prefers-reduced-motion: reduce)").matches) { root.classList.add("theming"); setTimeout(() => root.classList.remove("theming"), 350); }
      apply();
    },
  };
})();
