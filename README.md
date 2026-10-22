<p align="center"><img src="web/icon.svg" width="96" height="96" alt="OmniLLM logo"></p>

<h1 align="center">OmniLLM</h1>

<p align="center"><b>Your AI, Everywhere</b><br>Sign in to your AI models once, on one computer. Use them from every device you own.</p>

<p align="center">
  <a href="https://github.com/wensuper/OmniLLM/releases"><img src="https://img.shields.io/github/v/release/wensuper/OmniLLM?include_prereleases&label=release" alt="Latest release"></a>
  <a href="https://github.com/wensuper/OmniLLM/actions/workflows/ci.yml"><img src="https://github.com/wensuper/OmniLLM/actions/workflows/ci.yml/badge.svg?branch=main" alt="CI"></a>
  <a href="go.mod"><img src="https://img.shields.io/github/go-mod/go-version/wensuper/OmniLLM" alt="Go version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/wensuper/OmniLLM" alt="License"></a>
</p>

<p align="center">English | <a href="README.zh-CN.md">简体中文</a></p>

---

OmniLLM turns one computer — the **hub** — into the only place that holds your AI accounts: Claude and ChatGPT subscriptions (through the official Claude Code and Codex), API keys, and local models. Your phone, tablet, and servers use those models through an end-to-end encrypted connection. Phones install nothing: scan a QR code and chat in the browser.

## Features

- **Accounts stay on the hub.** Subscriptions, API keys, and local models never leave the computer they are on. Devices never see a credential.
- **End-to-end encrypted.** Devices reach the hub through a connection service that only forwards encrypted data; it can’t read your chats or pretend to be your hub.
- **No app to install.** Scan a QR code with your phone’s camera and chat in the browser, with text and pictures.
- **You decide who uses what.** Name each device, give it a daily limit, choose which models it may use, or pause it with one switch.
- **Kind to your subscriptions.** Subscriptions are for people chatting, at one person’s pace; scripts and programs use API keys or local models.
- **Takes care of itself.** Runs in the background, updates itself while nobody is using it, and rolls back an update that doesn’t start.

## How it works

```mermaid
flowchart LR
  P["Phone / tablet<br>(browser)"] <-- end-to-end encrypted --> R["Connection service<br>(forwards only)"]
  S["Server / other computer<br>(omnillm connect)"] <-- end-to-end encrypted --> R
  R <-- end-to-end encrypted --> H["Hub<br>(your Mac or Linux computer)"]
  H --> M["Claude Code · Codex<br>API keys · Ollama"]
```

Only the hub talks to model providers. The hub opens no ports to the network: it connects out to the connection service, the same way your devices do. See [Security](docs/en/security.md) for what is and isn’t protected.

## Installation

The hub runs on **macOS** and **Linux** (x86-64 and ARM64). Installing needs no `sudo`, takes about a minute, and asks no questions.

### Option 1 — Ask your AI assistant (recommended)

If you use an AI coding assistant on that computer (Claude Code, Codex, Cursor, …), send it this one sentence:

```text
Read https://raw.githubusercontent.com/wensuper/OmniLLM/main/INSTALL.md and install OmniLLM on this computer.
```

It reads the installer before running it, verifies the download, and installs OmniLLM; your browser then opens on the QR code for adding your phone, and you scan it and compare the digits yourself. It never needs your API keys: those go into the OmniLLM console, never into a chat.

### Option 2 — Run the installer yourself

In a terminal, as your normal user:

```bash
curl -fsSL https://raw.githubusercontent.com/wensuper/OmniLLM/main/install.sh | sh
```

The installer downloads the release for your system, checks its SHA-256 checksum (and its signature, if [cosign](https://github.com/sigstore/cosign) is installed), installs `~/.local/bin/omnillm`, starts the hub as a background service, and opens the console in your browser. In the terminal you installed from, use `~/.local/bin/omnillm` until you open a new window.

## Quick start

1. **Connect a model.** With a Claude or ChatGPT subscription (even if you only have the desktop app), choose the button on the Claude or ChatGPT card in the console, or run `omnillm login claude` / `omnillm login chatgpt`. It installs Claude Code or Codex with the official installer if needed, signs you in in the browser, and connects. Without a subscription, paste an API key in the console, or install [Ollama](https://ollama.com) for free local models.
2. **Add your phone.** In the console, choose *Devices → Add device* (on a Mac, also the OmniLLM menu bar icon → *Add device…*) and scan the QR code with your phone’s camera. When the 6 digits on both screens match, choose *Matches, add* on the computer.
3. **Chat.** That’s it — about five minutes in all.

The [quick start guide](docs/en/quickstart.md) covers every step in detail, including servers and scripts.

## Usage

```bash
omnillm status            # what’s running, and what to do next
omnillm open              # open the console (--add: straight to the QR code for adding a device)
omnillm login claude      # connect a Claude subscription (chatgpt: a ChatGPT subscription)
omnillm update            # update now (updates also install by themselves)
omnillm uninstall         # stop and remove the background service (--purge also deletes data and keys)
```

Run `omnillm help --all` for every command.

**Updates** install by themselves: the hub checks every few hours and updates while nobody is using it. To turn that off, run `omnillm update --auto off`. Hubs on 0.2.0-alpha.22 or earlier can’t update themselves yet: run the install command once more (your settings are kept).

## Documentation

| Guide | What’s in it |
|---|---|
| [Quick start](docs/en/quickstart.md) | Every setup step, which models work, adding Linux servers, letting scripts use the hub |
| [Everyday use and FAQ](docs/en/usage.md) | Devices and daily limits, messages and what to do about them, updating and uninstalling |
| [Security](docs/en/security.md) | What OmniLLM protects and what it doesn’t, compared with other approaches |
| [Self-hosting the connection service](docs/en/self-host.md) | Running your own connection service instead of the public one |
| [INSTALL.md](INSTALL.md) | Install steps written for AI assistants |
| [Changelog](CHANGELOG.md) | What changed in each release |

## Project status

OmniLLM is in early pre-release (0.2.0-alpha). Expect rough edges; the hub updates itself, so fixes reach you quickly.

| Platform | Hub | Device |
|---|---|---|
| macOS (Apple silicon and Intel) | ✅ with a menu bar icon | ✅ browser or `omnillm connect` |
| Linux (x86-64 and ARM64) | ✅ | ✅ browser or `omnillm connect` |
| iPhone, iPad, Android | — | ✅ browser, no app |
| Windows | Not yet | ✅ browser |

## Contributing

Issues and pull requests are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) first, and follow our [Code of Conduct](CODE_OF_CONDUCT.md).

## Security

Please report vulnerabilities privately, as described in [SECURITY.md](SECURITY.md). Don’t open a public issue.

## License

OmniLLM is licensed under the [Apache License 2.0](LICENSE).
