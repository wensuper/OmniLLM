# Changelog

English | [简体中文](docs/zh-CN/CHANGELOG.md)

All notable changes to OmniLLM are documented here. OmniLLM is in early pre-release (0.2.0-alpha); versions follow [Semantic Versioning](https://semver.org/).

## [0.2.0-alpha.49](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.49) - 2026-10-25

### Changed

- On a server, `curl … | sh` now goes straight on to the phone's QR code too (it reads your answer from the terminal), and shows only what a phone needs: the QR code and how long it works. Device IDs are no longer shown after pairing.
- With the console just opened and no model yet, the next step points only at that page. Once any model is connected, the installer no longer insists on signing in the AI assistant you installed with.
- The console waits for the sign-in window the installer opened and goes on to the QR code as soon as you're signed in; the window's last line says to go back to the browser. After the first device is added, the wizard ends ("send your first message from it").
- Installing again doesn't reopen the sign-in window, and says it is already installed.
- The installer says when OmniLLM starts by itself, and warns on Linux when it would stop at logout or after a restart.
- The phone page no longer assumes your OmniLLM runs on a Mac: pairing says "your computer" (with both ways to confirm), and a hub that isn't a Mac is called "your computer" everywhere.
- Signed-out fixes (console and terminal) and the API-key line use the same commands everywhere (`omnillm login claude`, `omnillm login chatgpt`, `omnillm hub provider add anthropic`).
- A hub installed with `OMNILLM_HOME` is found by every later command without setting it again.
- On a server, "All set" no longer tells you to open a console you can't open there. After `omnillm login` or `omnillm hub provider add` on a computer with a screen, the next step (adding a phone) follows.
- A computer set up as a device, and pairing errors, say "your other computer" or "the console / omnillm pair", not "your Mac". The phone's confirmation reads "Continue only if the QR code you just scanned is showing on your own computer".
- An AI assistant setting up a device (`--device`) never gets the pairing prompt; it tells you to run `omnillm connect pair` in your own terminal. INSTALL.md says the pairing text never goes through a chat.
- `omnillm pair` and `omnillm connect pair` don't start a pairing for an AI assistant (or with input from /dev/null, as INSTALL.md runs the installer); they say it is your step in your own terminal. A device that isn't paired yet says so, and where the pairing text comes from.
- `omnillm login`, and `omnillm hub provider add` asking for a key, don't run for an AI assistant either: signing in and entering a key happen in your own terminal. Codex bundled in the ChatGPT desktop app is recognized as an assistant.
- The pairing-text prompt says the paste isn't shown, so it is pasted once.
- Messages from your computer reach the phone page as "your computer" for a hub that isn't a Mac.
- On Linux the service keeps the local API setting (`OMNILLM_API`) across updates, as on macOS.
- On a phone or a device, a signed-out subscription's error now says to run `omnillm login claude|chatgpt` on your computer. `omnillm pair` on a server tells a new computer to install with `--device` first. Reinstalling into a different data folder no longer claims your settings were kept.
- The installer script reports progress only ("Updating X → Y…"); the result is said once.

### Security

- A data folder set with `OMNILLM_HOME` must be your own (it is made private), and the hub service now keeps it, so an update with it no longer rolls itself back.
- The master key file, the control socket and the Claude Code / Codex programs are used only when they are your own (or, for programs, the system's) and others can't change them.
- Downloads follow redirects only while they stay on https. A relay or phone page other than the project's, set once, is named at every install.
- Claude Code and Codex are run only from places no other user can change (the link, the file and both folders), checked again before every run. `OMNILLM_HOME` can't be a link or sit under a folder someone else can swap; the hub checks it at start too. On Linux the hub service no longer uses a private /tmp (it hid who owns a file from these checks); programs installed by npm under the usual umask 002 are still found.
- When no browser opens, `omnillm open` prints its one-time console link only for a person at the terminal, never into an AI assistant's output.

## [0.2.0-alpha.48](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.48) - 2026-10-24

### Security

- A device’s message that starts with “/” can no longer run one of Claude Code’s own commands (including the owner’s custom ones); it reaches the model as plain text.
- An update runs the install script of the release it installs (from that release’s tag), not the newest script on the main branch.
- `omnillm verify web` checks the release’s signature when cosign is installed, and calls the release “signed” only when it did.
- The public relay keeps no error log either (an error line names the client address and the mailbox). The port-80 redirect no longer logs requests.
- The relay counts hubs per IPv6 /64 as one address, caps how much may wait for one slow device (1 MB, then it is disconnected), and keeps its wrong-secret table at a fixed size, so one attacker can’t fill it or exhaust its memory.
- The hub’s control socket folder is made private every time and must be a real folder; the relay service may only use network and local sockets.

### Incident

- **On 2026-10-24, browser devices couldn’t reach their hubs for about 32 minutes after alpha.48 went live (10:54–11:26 Beijing time), with three more short interruptions while it was tracked down.** Phones and desktop browsers couldn’t connect or pair; hubs and the command-line client were unaffected, and no data was exposed. Cause: the deploy turned TLS session tickets off on the relay’s site alone, while other sites on the same server port issue them, so browsers’ TLS handshakes with the relay failed. That line was removed; the rest of alpha.48 is live. Relay changes now go out one at a time and count as done only after a real browser pairs on the new configuration. Full timeline (in Chinese): [docs/relay-deploy.md](docs/relay-deploy.md#2026-10-24-alpha48浏览器设备全部连不上重大).

## [0.2.0-alpha.47](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.47) - 2026-10-24

### Added

- The connection service counts, per day, how many different hubs connected and how many of them were new installs. Counts only: a hub is told apart within the day by a keyed hash whose random key is discarded at midnight, no IP address or device is recorded, and a new hub says so itself on its first connection, so the relay keeps no list of hubs. One log line a day, and `/stats` on the relay’s own machine only.

## [0.2.0-alpha.46](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.46) - 2026-10-23

### Changed

- Typing just `omnillm` now says where things stand and the one thing to do next, then lists the few commands people look for, by what they want to do (add a phone, connect a subscription or an API key, see the status, update). Before, it printed a generic command list. On a machine set up as a device, it lists chatting and re-pairing instead. `omnillm help` still prints every command.
- After a device is added in the terminal (`omnillm pair`, or right after an install or sign-in on a server), a short closing screen says the device is ready to use and how to find your way later: `omnillm pair`, `omnillm status`, or just `omnillm`.
- While pairing, a computer added from the command line no longer assumes the other side is a Mac: it asks for the pairing text from your Mac or from `omnillm pair` on your server, and says to confirm there.

## [0.2.0-alpha.45](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.45) - 2026-10-23

### Added

- On a server without a screen, one command after the install adds your phone. Run `omnillm login claude` (or `chatgpt`), or paste an API key with `omnillm hub provider add`; once the model is connected, the same terminal shows a QR code. Scan it with your phone and type y if the 6 digits match. If a model is already connected, the installer goes straight to the QR code. `omnillm pair` is the new short command for adding a phone later.

### Fixed

- The installer could report “OmniLLM isn’t running” right after installing it, when the hub answered slowly.
- A Claude Code that runs on an API key or another provider’s endpoint is no longer sent to a subscription sign-in.
- The sign-in window for the assistant’s subscription opens in front of the browser, only on a first install. Once you sign in, it connects the subscription itself, even with the console closed.
- The console opens on the QR code only when there is a model to use; otherwise it opens on the step that is next.
- The next step in `omnillm status --json` uses the full path when `omnillm` isn’t on PATH, and the JSON says whether the console or a sign-in window was opened (`console_opened`, `login_opened`).
- A pairing that expires or is refused right after an install or a sign-in no longer makes the command fail.
- `omnillm status` no longer runs the Claude and Codex programs on every call when no model is connected.
- INSTALL.md: the assistant never runs sign-ins, key prompts or pairing for the person; it downloads into a fixed folder, gives the install at least 5 minutes, and recognizes a sandbox refusal.

## [0.2.0-alpha.44](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.44) - 2026-10-23

### Changed

- Installing through Claude Code or Codex now makes sure that assistant’s own subscription gets connected. If OmniLLM can’t use its sign-in yet (signed out, signed in only to the system keychain, or running on an API key), the installer opens the guided sign-in right away on a Mac (a Terminal window, then a confirmation in the browser). The next step the assistant passes on says so first. `omnillm status` says the same until it is connected, and INSTALL.md tells the assistant to leave the sign-in to the person.

## [0.2.0-alpha.43](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.43) - 2026-10-23

### Added

- `omnillm verify web [address]` checks the web page phones open against the signed release that built it, file by file. Each release now publishes `omnillm-web-manifest.json` (every file’s sha256, in the signed `checksums.txt`); older releases are checked from their signed web tarball. Your hub runs the same check for the page its devices use, a few minutes after it starts and then daily; if it differs, the console’s top bar, the menu bar, and `omnillm status` warn you. Nobody has to take the web host’s word for it.

### Security

- The public connection service keeps no access log (it used to record each device’s connection with its mailbox ID). The security guide has a new section on what an operator of the relay and web page can and can’t do, and what you can check yourself.

## [0.2.0-alpha.42](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.42) - 2026-10-22

### Changed

- The console’s top bar no longer says “Phones and other devices can connect” all the time. It shows a line only when something needs you: remote access is down or not set up, or the region changed.
- Under today’s spending, “Estimated from published prices” now says what it means: API keys are counted at their official prices, and subscriptions and local models cost nothing.

### Fixed

- A subscription signed out on your Mac no longer shows a green dot in the web chat. Each model's dot says whether it works right now: amber with the reason under its name when its subscription is signed out, or at its usage limit (with the time it comes back). While such a model is chosen, a notice above the message box says how to sign in (`omnillm login claude` / `chatgpt`, or the card in the console). The hub checks the programs’ sign-in by itself at start, every 5 minutes, and when a device asks for its models, without calling a model; signing in again clears it.
- Error messages in the web chat use the page’s language (which follows your Mac), not the browser’s.

## [0.2.0-alpha.41](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.41) - 2026-10-22

### Changed

- A message you send shows *Thinking* at once. Before, the reply first said “Connecting to your Mac…” until your Mac confirmed it had the message, even when the connection was long up.

## [0.2.0-alpha.40](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.40) - 2026-10-22

### Fixed

- An `OMNILLM_API` set when installing (another loopback address, or `off`) is now written into the background service and kept by updates, like `OMNILLM_RELAY` and `OMNILLM_WEB_APP`. Before, the service always used the default.
- `omnillm help` lists `omnillm open --add`.

### Changed

- The design document and the user guides in both languages are brought up to date with everything through alpha.39: the desktop web chat, voice input, the Add device panel, the console tables and edit dialog, model source switches, the Apps tab, sign-in, the menu bar, and updates.

## [0.2.0-alpha.39](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.39) - 2026-10-22

### Changed

- The web page no longer sits on “Connecting to your Mac…” each time it opens. It shows the models your Mac listed last time right away, so you can type, pick a model, and send at once; a message sent before the connection is up goes out as soon as it is. The dot pulses until your Mac answers. Opening the connection took about 2.5 seconds on a slow network: one round trip each for the connection itself, the encrypted session, and the model list.
- When connecting fails (often a single lost packet on a slow network), the page tries again after 1, 2, and 4 seconds, then every 10 seconds as before. Previously it always waited 10 seconds.

## [0.2.0-alpha.38](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.38) - 2026-10-22

### Changed

- The web page no longer shows “OmniLLM is open in another tab” with a *Use here* button. Only one tab saves your chats at a time, as before. A tab you come back to now takes over by itself, picking up what the other tab saved, and the tab you left goes quiet. A tab you aren’t looking at never takes over, so two windows side by side don’t pass it back and forth.

### Fixed

- A browser’s built-in translation no longer turns model names into gibberish (for example “Claude Sonnet 5.5” rendered in Chinese characters): the web page and the console are marked as not to be translated, since they already speak Chinese and English themselves.

## [0.2.0-alpha.37](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.37) - 2026-10-22

### Added

- Voice input in the web chat, on computers and phones. Tap the microphone in the message box and speak: the button turns into a moving sound wave and your words appear in the box as you talk, after anything you already typed. Tap again, press Esc, start typing, or pause to stop; the words are never sent by themselves. It uses the browser’s own speech recognition (Chrome, Edge, Safari), and the page says once that the audio goes to Google or Apple for that; browsers without it show no microphone.

### Changed

- In the web page, pasting a pairing link or pairing text and choosing *Add* goes straight to the 6 digits; the extra “Add this device to your Mac?” step is gone for what you paste yourself. A link opened from the address bar (a scan or a click) still asks once, since anyone could have sent it, and replacing an existing pairing still asks.

## [0.2.0-alpha.36](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.36) - 2026-10-22

### Changed

- The web chat has a real desktop layout. On a wide window the chat list is a sidebar (grouped by Today, Yesterday, Previous 7 days and Earlier, with search), and the talk and the message box share one centred column instead of stretching across the screen. The sidebar can be hidden; on phones it is still a drawer.
- Replies are no longer boxed in cards. The model's name heads the first reply and a line marks where the model changes; every other reply names its model in the row under it. Code blocks have a header with the language and *Copy*.
- The model picker moved into the message box and shows which models your Mac can reach right now. Pictures can also be pasted or dragged into the window.
- *Your Mac* at the bottom of the sidebar says whether it is online and opens a menu: this device's name, language, keyboard shortcuts, where chats are kept, and removing this device. Notices such as "can't reach your Mac" now sit right above the message box.
- Keyboard shortcuts on computers: ⌘⇧O new chat, ⌘K search, ⌘\ hide the sidebar, Esc stop the answer, ⌘⇧; copy the last answer, / go to the message box, ⌘/ list them (Ctrl on Windows and Linux).
- The page now speaks your Mac's language (the one chosen for the console and the menu bar) instead of the browser's; you can still pick one for this device in the *Your Mac* menu.
- The console’s *Apps* tab says what apps are for in one line. How a program connects (the address with a *Copy* button, the app’s token as the key, OpenAI- and Anthropic-compatible, programs on this Mac only) opens from the *?* next to it and stays open until you click elsewhere. If the local API is off, the page still says so directly.

## [0.2.0-alpha.35](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.35) - 2026-10-22

### Added

- Each model source in the console (*Models*) has an *On* switch, like devices and apps. Switched off, its models are offered to no device or app, but its sign-in or API key is kept, and it works again as soon as it’s switched on. *Remove* is now a quieter text button, since it deletes the key. On the command line: `omnillm hub provider off|on <source>`; `omnillm hub providers` shows “Off”.
- `omnillm open --add` opens the console straight on the QR code for adding a phone or another computer.

### Changed

- Installing through an AI assistant ends on the QR code. When there is no device yet, the console the installer opens goes straight to adding a phone, and the last sentence the assistant passes on says just that: scan the QR code that is open in the browser, and if the 6 digits match, choose *Matches, add* on the computer. INSTALL.md now says that adding a phone is the person’s step: the assistant never runs the pairing or confirms it for them.
- The *Models* tab’s note on chat-only models is shorter and clearer.

## [0.2.0-alpha.34](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.34) - 2026-10-22

### Changed

- “Add device” in the console is redesigned. A switch at the top picks *Phone or tablet* or *Computer or server*, so you see one way at a time. On the left, the QR code (or the link) turns into the 6 digits in place. On the right are three steps: the current one is highlighted and finished ones are checked off. A status line below says what the panel is waiting for, and a countdown ring shows how long the code has left. When it expires, or the device isn’t added, *New QR code* sits right on top of it.
- A computer now joins through its browser: *Copy link* gives the same link the QR code holds; open it in that computer’s browser and choose *Add*, just like a phone. The command line is folded into one line at the bottom (*No browser on that computer? Use the command line*). It shows one install command, which also works on a computer that already has OmniLLM, and the pairing text, each with Copy. The command keeps `--device` in sight, and the page says that nothing shows in the terminal while you paste the pairing text.

## [0.2.0-alpha.33](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.33) - 2026-10-22

### Changed

- Documentation is now published separately in English and Chinese: `README.md` and `README.zh-CN.md`, user guides in `docs/en/` and `docs/zh-CN/`, and this changelog in both languages. Installing through an AI assistant is listed first everywhere.
- Added a contributing guide, security policy, code of conduct, and issue and pull request templates.
- The “how to install” links in the CLI and on the phone’s welcome page open the install section of the README in your language.

## [0.2.0-alpha.32](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.32) - 2026-10-22

### Changed

- Before a model is set to “Chat only” under “Models” in the console, the console now asks first and lists the apps and programs-only devices that would lose it, noting any that would stop working entirely because it is their only model. Previously the change took effect on one click and apps using the model silently broke. Nothing is asked when no one is using the model.

### Fixed

- While the pointer rests on a device or model card, the page’s 10-second refresh waits until the card closes, so the card you are reading no longer disappears.

## [0.2.0-alpha.31](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.31) - 2026-10-21

### Added

- Hovering over a device or app name, or a device’s “Models” cell, shows a small card: the name card lists when it was added, when it was last seen and today’s usage; the “2 of 5” card lists which models it can use and which are off.

### Fixed

- “Apps” in the console now actually works. Installed hubs never opened the local API for programs, so issued tokens were unusable and the page didn’t say where to connect. The hub now opens it by default at `http://127.0.0.1:8765/v1` (reachable only from programs on this Mac, token required); the “Apps” page shows the address, and after you add an app, ready-to-copy settings and a test command appear next to the token.
- A mistyped `omnillm login` (for example `omnillm login gemini`) now prints a single usage line instead of the full command list.

## [0.2.0-alpha.30](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.30) - 2026-10-21

### Added

- New commands `omnillm login claude` / `omnillm login chatgpt` do the same setup from a terminal and connect right away. The end of installation and the next step in `omnillm status` now point to these commands.

### Changed

- Smoother setup with a Claude or ChatGPT subscription: the first step in the console detects what is missing (for example “Found the Claude desktop app, Claude Code not installed” or “installed but not signed in”). “Install and sign in” opens a Terminal window that installs with the official installer and signs you in in your browser; once you’re signed in the card turns to “Connected” by itself, with no need to choose “Check again”.

### Fixed

- When Codex stores its sign-in in the system keychain, OmniLLM used to just say “not signed in”; it now explains the situation and guides you to sign in once more in a form it can use.

## [0.2.0-alpha.29](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.29) - 2026-10-21

### Changed

- The name column under “Devices” and “Apps” in the console is wider (other columns shrink to fit), so long names are no longer cut to “lima-omnill…”; when a name still doesn’t fit, hovering over it shows the full name.

## [0.2.0-alpha.28](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.28) - 2026-10-21

### Added

- “Devices” and “Apps” in the console have a new “Models” column: “All 4” when every model is usable, “2 of 4” (in purple) when only some are allowed, and “—” when none are; hovering lists which ones. At phone width it appears under the name.

### Changed

- When editing a device or app, you can switch off a single model even while “All models” is on: “All models” turns off and the other models stay on; switching it back on returns to “All models”.

## [0.2.0-alpha.27](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.27) - 2026-10-21

### Changed

- When editing a device or app, “All models” is on whenever every usable model is on (previously it needed a separate click). While “All models” is on, the models below show as on and greyed out, so you can see what it covers; switching on the last model that was off turns on “All models” automatically.

## [0.2.0-alpha.26](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.26) - 2026-10-21

### Fixed

- In Safari, closing the active tab now lets another open tab take over within seconds (alpha.24 did this in Chrome only).

## [0.2.0-alpha.25](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.25) - 2026-10-21

### Added

- The menu bar can switch between 中文 and English: “Language · 语言” in the bottom group offers 中文, English or “Use system language”, with the current choice checked. It is the same setting as the language button at the top right of the console; change it in one place and the other follows within seconds.

## [0.2.0-alpha.24](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.24) - 2026-10-21

Found by testing after alpha.23 went live.

### Fixed

- After a second tab is opened on the same phone and then closed, the first tab takes over by itself, with no need to tap “Use here”.
- With a Chinese locale, the end of installation printed an extra “REPO: unbound variable” line and left out the line saying the signature was verified (installation itself succeeded). Every variable directly followed by Chinese text in the install script is now braced, with a test guarding it.
- In a wide window, “Devices” in the console repeated “Last seen” and today’s usage under each device name (a side effect of alpha.23’s narrow-screen change). They now appear under the name only at phone width.
- “No usable models” next to a device now explains why: a device for programs and scripts only can’t use models set to “Chat only”. Editing the device also tells you how to fix it.

## [0.2.0-alpha.23](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.23) - 2026-10-21

Installation and updates were redesigned so that you install once and never have to think about it again. The Mac menu bar menu was reorganized to match NEXA, subscription chats continue the previous session, and eight rounds of product review (newcomer, daily use, visual comparison with NEXA, errors and edge cases, Chinese and English copy; full screenshots each round) led to many smaller changes.

### Added

- OmniLLM updates itself: a few minutes after starting and every 6 hours after that, it checks for a new version and installs it once nobody has used it for 10 minutes; it restarts in seconds and devices reconnect on their own. Turn this off with `omnillm update --auto off`.
- New commands `omnillm update` (works on the hub and on servers used only as devices) and `omnillm update --check`. `omnillm status` says whether a new version is available.
- If a new version fails to start, OmniLLM switches back to the previous version and never installs that version automatically again.
- `omnillm connect` on a server mentions when a new version is available.
- Subscription chats continue the previous session: when chatting with a Claude or ChatGPT subscription, each new message no longer resends the whole conversation. Your Mac continues the previous Claude Code / Codex session, so replies come faster and long chats use less of your subscription. Editing an earlier message, answering again or switching models starts over automatically.
- These sessions stay on your Mac (in Claude Code’s / Codex’s own session files) and are deleted after a day without use, or when the device or subscription is removed.
- Devices and apps: an edit dialog (name, on or paused, daily limit or no limit, models it can use, remove), a pause switch, online status and today’s spend; the daily total for the whole Mac can be changed; duplicate names are numbered automatically (“iPhone (2)”), and the phone shows the name the Mac gave it.
- Models: names include the version (looked up in the background at most 3 times a week, so it is known even before first use); models can be renamed or hidden on devices; a removed subscription doesn’t come back on its own, and “Check again” can reconnect it; model sources show signed-out and usage-limit states with the time they recover.
- Usage: shown by name, for today / yesterday / the last 7 days, broken down by model, with totals.
- Command line: `hub device set` / `app set` / `model set` / `provider rm` / `daily-total`.

### Changed

- Installation is now a single line, `curl -fsSL …/install.sh | sh`, with no install script left behind; a download interrupted halfway never runs a partial script.
- Running the install command again on a server used only as a device updates only the program, and it stays a device.
- Reinstalling (including updating) keeps your own remote access and web address settings; reinstalling from a terminal used to lose them.
- The README was trimmed to a Quick Start; “Which models work” moved to the quick start guide and “Compared with other approaches” to the security notes.
- The phone web page before pairing was redesigned: grid and aurora background, an orbit diagram of your Mac and devices, three steps and a large headline; people who haven’t installed OmniLLM yet can go straight to the setup guide.
- Mac menu bar: the top line states the current status (such as “Online · phones and other devices can connect”), followed by “Devices  3 (2 online)” and “Model sources  3”, each opening the matching console page; then “Add device…” and “Open the console…”; finally the version and “Check for updates…”. This replaces three lines of grey text.
- Mac menu bar: choosing “Check for updates…” keeps the menu open and shows the result on that line; when a new version is out, choose “New version X · click to update”, which uses the same path as automatic updates.
- Mac menu bar: the icon always stays in the menu bar, so “Remove from menu bar” is gone; menu text follows the system language.
- Consistent wording: your Mac, remote access, model source, daily limit, API key. The console says “this Mac” or “this computer” depending on the operating system, and the command line answers in the terminal’s language.
- Phone: the add-device and number-check pages were rewritten; “Answer again” keeps the earlier answer; a real chat list; streaming replies no longer flash half-formatted text; chat search; notices for paused, disconnected and connecting; two tabs on one phone give way to each other; removing this device also removes it on the Mac; the web page updates itself.
- Console: browser-native confirmation dialogs were all replaced by in-page ones; narrow-screen tables no longer wrap mid-word; switches, segmented buttons and notices use NEXA’s styles.
- Command line: tables instead of raw JSON (`--json` still works); ambiguous names are refused with the matching IDs listed; a mistyped option says what is wrong first; `omnillm status` lists model sources that need attention.
- Error messages: wait times are in the reader’s language (“about 3 hours”), and every message says what to do next.

### Fixed

- Console: pressing Enter to pick words in a Chinese input method no longer triggers a click (it used to approve a device directly).
- Command line: Chinese text no longer appears as truncated garbage.

## [0.2.0-alpha.21](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.21) - 2026-10-21

### Added

- Each reply is labelled with the model that wrote it, including its version (such as Claude Opus 5.5), and the model list shows versions too; a divider marks where the model changed.
- While waiting for a reply, an animated “Thinking” indicator (rotating orbit, shimmering text, bouncing dots, and after a while the seconds waited) replaces the empty thin box; a blinking cursor trails the text as it arrives.
- A green / grey dot in the model picker shows whether your Mac can be reached right now.

### Changed

- Pages open on a phone switch to a new version automatically (keeping unsent text) instead of staying on the old one.

## [0.2.0-alpha.20](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.20) - 2026-10-21

### Added

- OmniLLM icon in the Mac menu bar: shows at a glance whether your phone can connect, opens the console and “Add device”, and can start the hub when it isn’t running.
- Console: “Add device” can be closed; devices and apps can be renamed and given a daily limit or no limit; the hub’s daily total can be changed at the top of the page; the browser tab has an icon.
- Phone: chats in the chat list can be closed and deleted.

### Changed

- Console: usage is shown by device name instead of an ID string; the “Models” page uses wording anyone can follow; switches and buttons match NEXA.
- Phone: the Send button is greyed out while there is no text; plainer wording.

## [0.2.0-alpha.19](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.19) - 2026-10-21

### Changed

- New logo and tagline, “Your AI, Everywhere”, in the same family as NEXA, applied to the phone web page’s welcome, loading and chat-list screens, the console header, the home-screen icon and the README.

## [0.2.0-alpha.18](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.18) - 2026-10-21

Problems found by full testing after alpha.17 went live.

### Fixed

- If the network dropped for a few seconds while sending an image (for example when a phone switches between Wi-Fi and cellular), an “internal error” appeared once it came back. The page reopened its connection on recovery, closing the request in flight. It no longer reopens, and if the connection drops before the first word of a reply arrives, the question is asked again automatically.
- Two devices with the same name (both “iPhone”) couldn’t be told apart in the console, so removing one could remove the wrong one. Newly paired devices with a duplicate name are numbered automatically, for example “iPhone 2”.
- Opening “Add device” in the console during the first seconds after the hub started kept saying the connection service wasn’t configured and never recovered. It now says it is connecting to the connection service and shows the QR code once connected; the status line also updates by itself.
- `omnillm status` said everything was fine even when the connection service couldn’t be reached. It now says the connection service is temporarily unreachable and phones can’t be used for now.
- With a Chinese locale, the install script ended with “tag: unbound variable” (Chinese punctuation directly after a variable was read as part of its name).

## [0.2.0-alpha.17](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.17) - 2026-10-21

### Fixed

- Some browsers refuse to store non-exportable keys in local storage (failing with “The object can not be cloned.”), so pairing got stuck on the pairing page. These browsers can now pair and chat: the key lasts only while the page is open (still non-exportable), and the page explains in one sentence that after closing it you need to add the device on the hub again, and that Safari or Chrome keep it. Safari and Chrome on iPhone are not affected.

## [0.2.0-alpha.16](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.16) - 2026-10-20

Fixes after zc tried OmniLLM on a real phone.

### Added

- Each reply in a chat is labelled with the model that answered, so switching models midway is clear.

### Changed

- New look shared with NEXA: the phone web page and console use NEXA’s Aurora palette (purple for what you can click; green / orange / red only for status; neutrals with a hint of purple), type sizes, corner radii, shadows and focus styles, with matching dark mode. The new icon is in the same family as NEXA’s (deep purple background, purple-to-cyan light, the hub at the centre and devices around it).

### Fixed

- If an image was sent just as the connection service restarted (during a release), the phone retried only once before saying it couldn’t reach the hub. Now, if the hub replied within the last minute, a “can’t reach” is retried a few times over about 10 seconds, at 1, 2, 3 and 4-second intervals (only requests the hub hasn’t acknowledged are resent, so nothing runs twice). When the hub has been gone for a long time (Mac asleep), you are still told quickly.
- The “subscription” label now applies only to Claude / ChatGPT subscriptions (Claude Code, Codex); local models are no longer labelled “subscription” (previously everything set to “Chat only” was). Fixed on both the phone and the console.

## [0.2.0-alpha.15](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.15) - 2026-10-20

### Fixed

- The loading watchdog now warns only after 20 seconds with no progress; a slow but ongoing download (showing a percentage) is waited out. At night, downloads from the server ran at only about 150 KB/s, and alpha.14’s fixed 20 seconds interrupted normal slow downloads.

## [0.2.0-alpha.14](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.14) - 2026-10-20

### Fixed

- After scanning on a real phone, the page stayed on “Loading…” forever (zc’s iPhone, opened from the camera). Loading now says which step it is on (downloading the program with progress, starting, reading what this device saved). If it hasn’t finished after 20 seconds, it says where it is stuck and offers a “Load again” button that clears the old offline cache and reloads. If loading fails or the program stops unexpectedly, the reason is shown directly instead of an endless spinner.

## [0.2.0-alpha.13](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.13) - 2026-10-20

### Changed

- Faster first load on phones: the web client’s WebAssembly shrank from 2.24 MB to 1.35 MB gzipped (8.2 MB → 4.7 MB raw). The browser build uses the browser’s own WebSocket to reach the mailbox and no longer carries HTTP, TLS and WebSocket libraries or image decoders; JSON uses the classic implementation. JPEG and WebP dimensions are now checked by the hub (which already checked them). Native programs and the embedding API are unchanged.
- The web client must be deployed before the hubs update; the connection service and hubs need no change.

## [0.2.0-alpha.12](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.12) - 2026-10-20

Security review of the changes in alpha.6–11: no high-severity findings; all 4 low-severity findings fixed. In addition, alpha.11 passed 5 full rounds of regression testing over the public internet, 294 checks with 0 failures. Only the hub is updated; the web client and connection service are unchanged.

### Security

- When installation writes PATH into the shell startup file, it writes only to a regular file and doesn’t follow symlinks elsewhere.
- `omnillm install` itself now refuses to run as root / with sudo (previously only `install.sh` did), so no root-owned files end up in your home directory.
- The install script sorts versions in a fixed character order when picking one, so a release always sorts after its pre-releases under any locale.
- The hub sends only human-readable messages back to the command line over the control channel; the cause of an unexpected error goes to the hub log.

## [0.2.0-alpha.11](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.11) - 2026-10-20

### Fixed

- Replies on the phone now render `*italics*` as formatting (multiplication signs and asterisks inside words are not treated as italics) instead of showing the asterisks.

## [0.2.0-alpha.10](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.10) - 2026-10-20

### Fixed

- The install script picked the wrong version: GitHub’s release list isn’t sorted by version (alpha.10 came after alpha.5), and the script took the first entry, installing alpha.9. It now compares version numbers (numbers as numbers) and takes the highest. The script is downloaded from the `main` branch, so the fix applies immediately.
- Asked “introduce yourself”, Claude via subscription described itself as an assistant “built on the Claude Agent SDK that can work with files” (Claude Code’s built-in wording), although it can’t read or write files here. Now, when a chat has no system prompt of its own, the hub tells the model it is Claude (an AI assistant on the Codex side) chatting with a person on a phone or computer, that it can’t read or write files, run commands or browse the web and should say so when asked, and that it should reply in the person’s language. Only the hub is updated; the web client and connection service are unchanged.

## [0.2.0-alpha.9](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.9) - 2026-10-20

### Changed

- “Connect a model” when the console first opens: subscription setup is explained once, naming the program to install and the sign-in command (`claude`, `codex login`), and Ollama is mentioned too; the duplicate collapsible explanation was removed. The three steps no longer wrap onto two lines on a 375 px screen.

## [0.2.0-alpha.8](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.8) - 2026-10-20

Fixes from a usability walkthrough from install to daily use (web page on a real phone, console, Linux server, public service).

### Changed

- Replies on the phone no longer show raw markup such as `**bold**`: headings, bold, inline code and lists are rendered as formatting (HTML is still not parsed).
- During terminal pairing, the 6-digit number is shown as “123 456”, and a successful pairing tells you the device’s name on the hub.

### Fixed

- On a new Mac, `~/.local/bin` isn’t on PATH, so following the hints to type `omnillm open` or `omnillm connect chat` after installing failed with “command not found”. Installation now adds it to the shell startup file (a single line), the commands shown by this installation use the full path, and new terminals can just type `omnillm`.
- For Chinese users running `omnillm hub pair` in a terminal, the hub’s replies (confirmation questions, errors) were in English; the background service now remembers the language used at install time. The install script’s messages are now available in Chinese too.
- After a phone is removed on the hub, that phone shows “removed” and clears its pairing and chats as soon as it returns to the foreground (or every two minutes while open), without waiting for the next message.

## [0.2.0-alpha.7](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.7) - 2026-10-20

### Fixed

- Human-readable messages on the command line (such as “this device is already paired” or “the master key file must be readable only by you”) used to all appear as “something went wrong internally”, with the real message in parentheses; when the connection service refused to open a mailbox during `omnillm hub pair`, it also just said “something went wrong internally”. The original message is now shown directly, connection service problems say the connection service is temporarily unavailable, and genuinely unexpected errors include the cause for easier reporting.
- `omnillm connect pair` used to treat “can’t read the saved pairing” (keychain locked, key file mismatch) as “not paired”, so pairing again would overwrite a working pairing; it now stops and explains. “Already paired” and “not paired yet” both name the data directory checked, so a changed user or data directory is obvious.

## [0.2.0-alpha.6](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.6) - 2026-10-20

### Added

- User-facing documentation rewritten ahead of open-sourcing: the README (why it exists, 3 steps to start at no cost, one-page security promises and limits, supported models), plus a new quick start, daily use and FAQ, security notes, self-hosting the connection service, `SECURITY.md` (GitHub private vulnerability reporting is enabled) and `CONTRIBUTING.md`.

### Changed

- The console’s “subscription only” note now explains how to use a subscription (install and sign in to Claude Code / Codex and it connects automatically) instead of saying it isn’t supported.

### Fixed

- Pasting damaged or wrong pairing text used to say only “something went wrong internally” on both the phone and the terminal (alpha.5’s check caught it, but the message never reached the person). It now says either that the pairing text is incomplete or altered and should be copied again in full on the hub, or that this isn’t an OmniLLM pairing text.
- The “request too large” message states the correct limit (5 MB of text plus images).

## [0.2.0-alpha.5](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.5) - 2026-10-20

### Added

- Pairing text now carries a checksum (`omnillm3-`): if a character is missing or changed when pasting, you are told right away that the pairing text is damaged and should be copied again, instead of being asked whether to connect to an unrecognized connection service. `omnillm1-` and `omnillm2-` still work. The web client must be upgraded before the hubs.

### Fixed

- A pairing link received while the web page is still loading (the address changed but the browser sent no `hashchange` event) is no longer lost; the page reads the address once loaded. This happened once in 5 rounds over the public internet.

## [0.2.0-alpha.4](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.4) - 2026-10-20

Includes all fixes from alpha.3.

### Changed

- Pairing QR codes are easier to scan: pairing text uses a compact format (`omnillm2-`, about half the length), so the QR code has larger, sparser dots; the old `omnillm1-` format still works. The web client must be upgraded before the hubs (release order: the server’s web client, then each person’s hub).
- “Add device” in the console: a larger QR code with a single sentence for phones; steps for computers and servers are folded under “Computer or server (no camera)”, which holds a one-line install command (installs the program only, no service, then asks you to paste the pairing text) and the pairing text, each with a copy button. The 6-digit confirmation number is shown as “123 456”, the same as on the phone.

### Security

- Log redaction explicitly covers both pairing text formats and pairing links.

## [0.2.0-alpha.3](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.3) - 2026-10-20

Fixes after the third security review and public-internet testing.

### Changed

- Long chats with many images no longer stall: only the two most recent messages with images include them; earlier images are noted in the text.
- When the phone loses its network, it immediately says the device is offline, keeps what you just typed and reconnects automatically once back online. A message sent before the hub connects waits for the model list to load before going out (over the public internet the list arrives slightly after the chat screen).
- Codex receives an attached image as a single argument, so the `-` that reads the prompt is no longer taken for an image.

### Fixed

- Web client: the page, scripts and Service Worker share one build number. If the page is opened just as the server switches versions and gets a mix of old and new files, it clears its cache and reloads once instead of caching the mixed version.

### Security

- Images: besides format and size, images are rejected by the dimensions in their header when a file of a few KB claims billions of pixels (≤ 8192 per side, ≤ about 40 million pixels in total), so Claude Code / Codex on the hub can’t run out of memory decoding them.
- Server deployment: the connection service runs under its own login-less account (systemd hardening); web files are owned by root and read-only, so a compromised connection service can’t change the web page phones load. Packaging stops if cosign is missing; the server re-verifies the signed checksums before installing; the web directory doesn’t follow symlinks; installation stops if any log format would record the request header carrying the browser’s mailbox secret.
- The connection service counts secret-guessing attempts per /64 for IPv6.

## [0.2.0-alpha.2](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.2) - 2026-10-20

### Added

- Text + images: on phones and tablets, tap the image button (camera or photo library) or paste; images are scaled down on the phone before sending. In a terminal, use `omnillm connect ask --image FILE`, or `/image FILE` in a chat. The local API accepts inline images in OpenAI / Anthropic format (it doesn’t download URLs for programs).
- Subscriptions can see images too: Claude Code via its JSON input, Codex via `-i` reading image files from a private temporary directory; tools remain fully disabled. Each image is at most 2 MB, at most 4 per message, and the hub checks the format from the file contents. Models that can’t see images say so directly.
- The hub log records how long each call took (first token and total), without content.

### Changed

- UI: “subscription” in model names follows the interface language; the image and Send buttons use clear icons; “Retry” in the offline notice no longer wraps; replies no longer have extra blank lines around paragraphs.
- The web client is served compressed (7.5 MB → about 2 MB), cutting a phone’s first pairing from about 31 seconds to about 11; the web client manifest’s content type is fixed (server configuration, see `deploy/`).

### Fixed

- After a device is removed on the hub, the phone notices in about 1 second and returns to the pairing page (previously 8 seconds). Subscription pacing allows bursts of 4 requests, 12 per minute.
- During upgrades the background service occasionally failed to re-register (registering before launchd had finished unloading); it now waits for the unload and retries.

## [0.2.0-alpha.1](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.1) - 2026-10-19

A Claude or ChatGPT subscription signed in on your Mac now works from phones, tablets and Linux servers, with as little as possible installed anywhere but the Mac.

### Added

- Subscriptions: the hub uses the signed-in Claude Code and Codex (the official programs) directly, as text-only models: no tools, no reading the owner’s settings or instruction files, no saved sessions. Codex runs in the hub’s own directory and only links the owner’s sign-in file. Subscriptions connect automatically on install, on start, on `omnillm status` or when the console opens.
- Phones and tablets: scan the QR code in the console to open the web client directly (no app to install), then pair and chat in the browser. Encryption runs in the page using the same code as the command line (WebAssembly); pairing state and chats are stored encrypted in the browser.
- Linux servers: `install.sh --device` installs a single program with no service; paste the pairing text and it’s ready. `omnillm connect chat` gives multi-turn chat in a terminal; scripts (no terminal) can’t use subscriptions.
- Public connection service `wss://relay.ixxzh.com` and web client `https://omni.ixxzh.com` (hosted on the ixxzh.com server; steps in `docs/relay-deploy.md`). Hubs connect to it by default, and the web client is signed with each release instead of using GitHub Pages.

### Changed

- Request pacing: when the subscription quota is temporarily used up, a request that can get its turn within 20 seconds waits and is sent then, instead of failing.

### Fixed

- A removed device is told clearly that it was removed, instead of showing “can’t reach the hub” forever.

### Security

- Ban protection: subscriptions are for people only (hard-coded; the owner can’t turn it off); one request at a time, 12 per minute, 60 per hour per device; when subscription usage hits its cap, use pauses until the recovery time the provider gives; every new version of the official programs must pass a “can it use tools” check before it is used.

## [0.1.0-alpha.2](https://github.com/wensuper/OmniLLM/releases/tag/v0.1.0-alpha.2) - 2026-10-18

Fixes after several full runs on two machines (a Linux server as hub and a Mac as hub).

### Added

- Releases include a Sigstore bundle.

### Fixed

- Abandoned pairings no longer take up slots: quitting the command line, reopening “Add device” in the console or closing the page, or the SDK’s ctx ending all cancel the pairing immediately. Previously, after 8 abandoned attempts no new device could be paired for 5 minutes.
- Pairing started just after the relay restarts waits for it to reconnect (up to 15 seconds), and a device retries connecting to its mailbox for up to 20 seconds. When the relay can’t be reached, you are told so instead of being wrongly told the pairing text expired.
- After the relay disconnects, the hub reconnects within seconds (backoff capped at 10 seconds); only connections that drop again right away keep backing off longer.
- If the console’s default port is taken by another program, the next free port is used and the sign-in link follows it.
- Ollama with only embedding models is not connected automatically.
- Machines without a display are prompted to enter keys in the terminal.
- The Linux service also writes `hub.log`.
- A wrong token on a device’s local API gets a clear message.

## 0.1.0-dev - 2026-10-18

### Added

- Libraries: `hub`, `device`, `relay` (connection service) and `localapi` (local OpenAI-compatible API).
- End-to-end encryption (Noise IK / IKpsk2); the connection service can’t see content.
- A single upstream exit: rate limiting, backoff, circuit breaking, people first; budgets are reserved for the worst case.
- Pairing: pairing text plus a 6-digit check code confirmed on the hub; the mailbox secret is rotated after pairing.
- `omnillm` command line for the hub, devices and the connection service.
- Tests: unit, fuzz, race, end-to-end (capturing connection service traffic), adversarial, regression from two independent reviews, static analysis, and smoke tests with the real programs.
- Local console and first-run wizard; QR-code pairing; `omnillm install / uninstall / status / open`; `INSTALL.md` written for AI agents and an `install.sh` that verifies downloads; signed releases with GoReleaser, and CI.
- Work not done yet is listed in `docs/design.md` §13.3.
