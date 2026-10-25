# Security

English | [简体中文](../zh-CN/security.md)

This document is for people who use OmniLLM: what it protects, how it protects it, and what it does **not** protect. For implementation details, see sections 3 and 5 of the [design document](../design.md) (in Chinese); for the rules that must never be broken, see the [principles](../principles.md) (in Chinese).

## The short version

- Your **accounts** (subscription sign-ins and API keys) stay on the hub. They’re never sent to any device and never pass through the connection service.
- Your **chats** are end-to-end encrypted between your devices and the hub. No connection service, including the public one the project runs, can see their content or impersonate your hub.
- Your **subscriptions** are used only for people chatting by hand, at the pace of one person, to make it as unlikely as possible that the model provider sees the account as shared or abused by automation.
- Protection is limited if the hub is compromised, if someone takes your phone while it’s unlocked, or if the web host acts maliciously. Each case is covered below.

## Who can see what

| Who | Can see | Can’t see or do |
|---|---|---|
| **Model providers** (Anthropic, OpenAI, and others) | What you send to the model, the same as when you use them directly | How many devices you have or where they are; every request comes from the hub |
| **The connection service** (public or your own) | The IP addresses of the hub and devices, connection times, the rough size and timing of each message, and which devices are online | Chat content, images, model names, usage, device names, or any credentials. It also can’t replace the hub’s public key, so it can’t act as a man in the middle |
| **Others on the network** (public Wi-Fi, your carrier) | That you’re connecting to `relay.ixxzh.com` (TLS-encrypted) | Everything else, as for the connection service, with an extra layer of TLS |
| **Someone who only photographed the QR code** | Nothing | Pairing: the QR code expires after 5 minutes and works only once, and in the end you still have to check a 6-digit number and confirm at the hub |
| **Someone who has your phone** | Whatever the phone can open, including chat history stored on it | Your accounts and keys; they also can’t go over that device’s limits. When you choose *Remove* for the device on the hub, it stops working at once; the next time it opens, it’s told it was removed and its pairing is cleared |
| **Other programs on the hub, or a malicious web page you open** | Nothing | The local API listens only on `127.0.0.1`, every request needs a token, any request that carries a browser origin is refused, and so is any request with the wrong `Host` header (to block DNS rebinding); the console needs a one-time sign-in link |
| **People using OmniLLM on a device** | The model’s answers | Making the hub run commands, read or write files, or go online: every tool is turned off when a subscription model is called |
| **Someone who controls the hub** (the hub is compromised) | Everything | This is what OmniLLM can’t protect against: your accounts are on the hub |

## Encryption and pairing

- **End-to-end encryption**: The hub and each device each have a long-term key pair. Every connection uses an IK-pattern handshake from the [Noise Protocol Framework](https://noiseprotocol.org) (X25519 key exchange, ChaCha20-Poly1305 encryption, BLAKE2s hashing; the same set WireGuard uses). Session keys are negotiated afresh on every connection, giving forward secrecy: if today’s keys leak, they can’t decrypt traffic recorded earlier. Sessions rekey automatically every hour or every 1 million messages.
- **Protection against replay and tampering**: The first handshake message carries no request, so recording and replaying it can’t trigger a call. Every message has a strictly increasing number; a message that is altered, repeated, or out of order is dropped and the connection is closed.
- **Message lengths are padded**: Messages are padded to one of several sizes (1, 4, 16, or 64 KB), so from the size the connection service can tell only the rough order of magnitude.
- **Public keys are exchanged only in person**: The hub’s public key is in the QR code or pairing text, which you carry from the hub’s screen to the device; it never passes through the connection service. The device checks this public key on every later connection.
- **Pairing needs your confirmation**: The QR code also holds a one-time pairing key (a 256-bit random number that expires after 5 minutes). After the handshake, both screens show the same 6-digit number, derived from the handshake. Pairing succeeds only after you check that they match and confirm on the hub. Confirmation can be done only on the hub.
- **Pairing text has a checksum**: If a character is missing or wrong after copying, you’re told at once “The pairing text is incomplete or altered,” and the device doesn’t connect anywhere else.
- **Pairing links ask before connecting**: A pairing link opened from the address bar (a scan or a click) could have come from anyone, so the web page first asks “Add this device to your Mac?”, shows which connection service it leads to, and continues only after you choose *Add*. A pairing link or text you paste into the page yourself goes straight to the 6 digits. Replacing an existing pairing always asks first.

## Accounts and credentials

- **Subscriptions**: Claude and ChatGPT subscriptions are used through the official Claude Code and Codex programs on the hub. You sign in to the official program yourself, and the sign-in stays there (in the system keychain or the program’s own files). OmniLLM only asks it “Are you signed in?”; it doesn’t read, copy, or refresh the sign-in. OmniLLM doesn’t use cookies from the subscription websites and doesn’t scrape website sign-ins.
- **API keys**: Kept only in the hub’s system keychain (on Linux without a keyring, in an encrypted file only you can read, with the key file stored separately). The console shows only the last 4 characters. You can enter a key only in the console or in a terminal on the hub, not as a command-line argument, because arguments stay in your shell history.
- **Nothing leaks out**: Only the text of the model’s answer is sent to devices. Upstream response headers and raw errors are never forwarded; errors are rewritten as fixed explanations. Logs automatically redact `Authorization`, `x-api-key`, `sk-…`, JWT-shaped strings, and pairing text.
- **Guarded by tests**: Tests run pairing and chats with a set of “canary” credentials and plaintext, and assert that none of them appears in any byte the connection service receives, in the logs, or in files on disk.

## How subscription models are confined

Claude Code and Codex are coding assistants that can run commands and read and write files. When the hub calls them, it uses them only as plain text models:

- **All tools are turned off**: Claude Code gets no tools, no MCP connections, and doesn’t read your settings. Codex runs in a read-only sandbox with shell, plugins, browser, web search, and every other feature turned off, and doesn’t read your configuration.
- **Your personal instructions aren’t read**: Codex runs in OmniLLM’s own directory, so it can’t read the personal instructions you wrote for it in `~/.codex/AGENTS.md`.
- **A message is only ever a message**: a device’s text that starts with “/” is handed over so that it can’t run one of the program’s own commands (yours included); it reaches the model as plain text.
- **A clean environment**: They run in an empty temporary directory with only these environment variables: HOME, PATH, language, and the temporary directory. The hub’s own keys, connection service token, and proxy settings are never passed on.
- **New versions are checked first**: Whether the tools can be turned off depends on whether the official program honors these options. So for every new version it sees, the hub first has it try, in the background, to run a command and read a temporary file. If it manages either one, that version isn’t used, and devices show “Update OmniLLM.”
- **Images are checked before use**: Only PNG, JPEG, GIF, and WebP are accepted; the format is checked against the file’s leading bytes; images that are a few KB but claim billions of pixels are blocked (no more than 8192 pixels on a side and about 40 million pixels in total); each image is at most 2 MB. The local API accepts only images attached directly; it doesn’t download URLs for programs, so the hub can’t be used as a stepping stone to reach arbitrary URLs.

## Avoiding account bans

Providers usually ban accounts that look shared, abused by automation, or used to get around limits. OmniLLM’s rules are written into the code, and the owner can’t loosen them:

| Risk | Rule |
|---|---|
| Looks shared by several people | Every request comes from one computer, the hub; a hub serves only one owner and that owner’s own devices, and there’s no “share with others” feature |
| Looks automated | Subscriptions are for people chatting by hand only: in the web client a person is typing; in a terminal, `connect chat` and `ask` count as a person only in a real terminal, while pipes, cron, and scripts all count as background programs. Background programs, app tokens, and the local API on devices can’t see subscription models |
| Requests too frequent | A subscription model runs one request at a time, at most 12 a minute, with at most 4 waiting; each device gets at most 60 an hour and 300 a day |
| Keeps sending after being rate-limited | Honors `Retry-After` and the subscription’s usage-limit notices, and pauses the whole provider until the recovery time the provider gives, instead of hammering it with retries |
| Getting around limits | No rotating accounts, rotating keys, changing IPs, or faking client identifiers |

These rules reduce the risk but **can’t guarantee** your account won’t be banned: that’s decided by the provider’s terms and judgment. Follow the terms of use of each provider you use.

## The web client on your phone

- **Encryption happens on the phone**: The web page runs the same code as the command-line client (compiled to WebAssembly), and all encryption and decryption happen on the phone.
- **Encrypted storage**: Pairing information (such as the device’s private key) and chat history are stored in the browser’s IndexedDB, encrypted with a **non-exportable** AES-GCM key generated by the browser.
- **Protections for the page itself**: A strict Content Security Policy (it runs only this site’s scripts, loads no third-party scripts, styles, or resources, and allows no inline scripts); it can’t be embedded in other pages; it sends no Referer.
- **A separate domain**: The web client is on `omni.ixxzh.com`, which is used for nothing else, so no other site on the same origin can read its storage.
- **Pairing text stays out of server logs**: In the QR code link, the pairing text comes after the `#`, and browsers don’t send that part to the web server; once the page opens, it immediately erases the pairing text from the address bar and history.
- **Voice input doesn’t go through OmniLLM**: the microphone button uses the browser’s own speech recognition, so the audio is processed by the browser’s maker (Chrome and Edge send it to Google; Safari to Apple, or recognises it on the device when it can). Browsers without speech recognition, such as Firefox, show no microphone. It doesn’t pass through the connection service or the hub; the recognised words are treated like typed text and are end-to-end encrypted when you send them. If you’d rather not give your voice to the browser’s maker, don’t use the microphone; typing works as before. The page says this once, the first time you use it.

**A limit to be aware of**: Browsers don’t verify the web page code they load the first time. If the server hosting the page is compromised or acts maliciously, it can serve malicious code to devices that open the page afterward. How we handle this:

- The web page is deployed only from signed release packages on GitHub and is never edited by hand on the server; its files are owned by root and read-only, and the connection service runs as a separate unprivileged account, so even if the connection service is compromised, it can’t change the page.
- Once cached, the page is pinned as a whole set, and a new version replaces the whole set.
- Anyone can check that the page served is, file by file, what a signed release built: `omnillm verify web` (see the next section). Every hub also checks the page its own devices open, a few minutes after it starts and then daily, and warns loudly if it differs.
- Not done yet: the browser itself refusing a changed page. A check finds a change; it can’t stop the first load.

If this matters to you, use the command-line client `omnillm connect` (a signed program), or [host the web client yourself](self-host.md).

## If someone else runs the connection service and the web page

Most people use the public connection service (`relay.ixxzh.com`) and web page (`omni.ixxzh.com`), run by the project’s author. You don’t have to take the operator’s word for any of this; each point below can be checked.

**What the operator can’t do**
- **Read your chats or pictures.** They are end-to-end encrypted between your device and your own hub (Noise, ChaCha20-Poly1305). The connection service only ever sees ciphertext.
- **Pretend to be your hub, or sit in the middle.** Your hub’s public key and the pairing secret reach a device only through the QR code or pairing text shown on your own screen, and you confirm the 6 digits on both sides. The connection service never sees either.
- **Add a device to your hub.** Pairing needs the code from your screen and your confirmation on the hub.
- **Get your accounts or keys.** They never leave your hub; devices don’t have them either.
- **Change the hub program.** It comes from GitHub releases signed by the release workflow; the installer checks them, and the operator’s servers aren’t involved.

**What the operator can see**
- That a hub or device is connected, from which IP address and when, and the rough size and rhythm of messages (padded to 1, 4, 16, or 64 KB). This is what any relay sees.
- The public service keeps no access log of connections and no error log (an error line would name the address and the mailbox), the connection service writes nothing to disk, and what it holds in memory is gone when a connection ends.
- It limits abuse without logging: hubs per address (an IPv6 /64 counts as one address), wrong secrets per address, and how much may wait for one slow device (1 MB; past that the device is disconnected).
- Two numbers a day, for the operator: how many different hubs connected, and how many of them were new installs. During the day a hub is remembered only as a keyed hash under a key drawn at random that morning and discarded at midnight, so the count can’t be traced to a hub or linked across days; a new hub says so itself on its first connection, so the relay keeps no list of hubs. The numbers are one log line per day and are readable only on the relay’s own machine.

**What you have to check rather than trust: the web page.** The page phones open is served by the operator, and a changed page could read what you type on that phone. So it is made checkable:
- Every release publishes the exact files of its web page (`omnillm-web-manifest.json`, covered by the signed `checksums.txt`).
- Anyone can run `omnillm verify web` (or `omnillm verify web https://your-page/`). It reads the build of the page being served, finds the release that built it, checks that release’s checksums and, when cosign is installed, its signature, and compares every file. The result is either “exactly what release … built” (it says “signed” only when the signature was checked) or the list of files that differ. A host could still serve one page to this check and another to phones; the check proves what the host serves to it.
- Your own hub does this by itself, a few minutes after it starts and then daily, for the page your devices use. If it differs, the console’s top bar, the menu bar, and `omnillm status` say so.
- A limit: an operator who served a changed page to one person only, and the real one to everyone else, could avoid other people’s checks. For that level of assurance, [run your own connection service and web page](self-host.md) (one command each), or use the signed command-line client `omnillm connect`.

## The hub and the local machine

- The hub **doesn’t listen on any external port**, so you don’t need to change your router or firewall. It only makes outgoing connections: to model providers and to the connection service.
- The local console listens only on `127.0.0.1`. To sign in, you use a one-time link generated by `omnillm open` (valid for 2 minutes, single use), which is exchanged for a session that works only on this computer; every request is also checked for its `Host`, origin, and a custom request header.
- The background service belongs only to the current user; it doesn’t need or accept sudo. The data directory has permissions 0700 and files 0600.
- The hub doesn’t store chat content; audit records hold only metadata (who, when, which model, and how much was used). The exception: when you chat with a Claude or ChatGPT subscription, Claude Code and Codex keep the conversation on the hub in their own session files, so the whole conversation doesn’t have to be resent with each new message (like your history when you use them yourself, readable only by your user account). These are deleted after a day without use, when the device is removed, or when you remove the subscription in the console; they never pass through the connection service and never reach other devices.

## Releases, installation, and updates

- Release packages are built by GitHub Actions from tagged code and come with a software bill of materials (SBOM); the checksum file is signed keylessly with [cosign](https://github.com/sigstore/cosign), and the signature is bound to this repository’s release workflow.
- Downloads follow redirects only while they stay on https, in the install script and in updates alike.
- OmniLLM trusts only what your own user made: a data folder set with `OMNILLM_HOME` must be yours (it is made private), the master key file must be yours and readable only by you, a control socket that belongs to someone else is never used, and Claude Code or Codex is used only when the program, any link to it and the folders holding them are owned by you or the system and others can't change them (checked again before every run). A data folder set with `OMNILLM_HOME` can't be a link or sit under a folder someone else could swap.
- The download address is hard-coded in the install script and can’t be changed with an environment variable; the script checks the SHA-256 first and, if cosign is installed, the signature too, and stops if either doesn’t match.
- Updates use the same install script: whether the hub updates itself, you run `omnillm update`, or you click update in the menu bar, it downloads the install script as of the release it installs (from that release’s tag, not the branch tip), checks it by the same rules, and reinstalls; there’s no other download path. Without cosign, an update is vouched for by GitHub’s TLS and the release’s checksums; with cosign, by the release workflow’s signature too. Versions you build yourself during development only notify you and never replace themselves.
- If a newly installed version fails to start, the installer switches back to the previous program file (`~/.local/bin/omnillm.prev`), and automatic updates skip that version from then on.
- CI runs the tests (including race detection), `go vet`, `staticcheck`, and `govulncheck`.

## If something goes wrong

| Situation | What to do |
|---|---|
| You lost your phone or computer | In the hub’s console, under *Devices*, choose *Remove*; it takes effect at once |
| You suspect an API key has leaked | Revoke it in the provider’s dashboard, then replace it with a new one in the console. Deleting a key in OmniLLM only removes it from this computer; the key itself stays valid with the provider |
| The hub is lost or compromised | Revoke your API keys in each provider’s dashboard, and sign out of your subscriptions in the official programs (or sign out of all sessions on the provider’s website); then reinstall on a new computer and pair your devices again |
| You received a pairing link from an unknown source | Don’t choose *Add*. Normally, pairing links come only from the QR code on your own hub’s screen |

## Safer ways to use it

- **Run your own connection service** (see [Run your own connection service](self-host.md)): the metadata the connection service can see, such as IP addresses and connection times, then stays only on your own server.
- **Use the command-line client** instead of the web client, to avoid having to trust the host when the web page loads for the first time.
- In the console, give each device only the models it needs, and lower its daily limit.
- Install cosign so the install script also checks the signature.

## Compared with other approaches

If you want to use your own subscription or key on every device, the common approaches each have problems:

| Common approach | Problem |
|---|---|
| Sign in on each device | Tedious; you may not want to leave a sign-in on servers or shared computers; using one account in many places and from many IPs at once can get it flagged as shared, and limited or even banned |
| Copy the API key to each device | Keys end up scattered; losing one device means replacing the key; keys can be left behind in scripts, config files, and command history |
| Run your own public API gateway | Needs a server, a domain, and a certificate; the gateway is open to the internet, so once scanned it can be abused for free calls on your bill; the gateway can see all your chats |
| Tunnel or mesh-network back to a computer at home | Needs router changes or mesh-network software; every device has to install it and sign in |

OmniLLM keeps your accounts only on the hub, sends every request from that one exit point under one set of rate limits, opens no external ports on the hub, and the connection service in between only forwards ciphertext.

## Reporting a vulnerability

Please don’t report security issues in public issues. See [SECURITY.md](../../SECURITY.md) for how to report them.
