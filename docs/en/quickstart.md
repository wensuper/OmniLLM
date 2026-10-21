# Quickstart

English | [简体中文](../zh-CN/quickstart.md)

Goal: within 5 minutes, use the Claude or ChatGPT on your Mac from your phone, with no extra cost and no server of your own.

- [Before you start: what you need](#before-you-start-what-you-need)
- [Step 1: Install on the hub](#step-1-install-on-the-hub)
- [Step 2: Connect a model](#step-2-connect-a-model)
- [Step 3: Add a phone or tablet](#step-3-add-a-phone-or-tablet)
- [Step 4: Add a Linux server or another computer](#step-4-add-a-linux-server-or-another-computer)
- [Step 5 (optional): Let programs use it](#step-5-optional-let-programs-use-it)
- [Check that it works](#check-that-it-works)

## Before you start: what you need

| You need | Details |
|---|---|
| A computer to act as the hub | macOS or Linux (x86-64, ARM64). Your devices work only while the hub is on and online, so pick the one that stays on. Windows can’t be the hub yet |
| One model source, any of three | **Subscription**: Claude Code (Claude Pro / Max) or Codex (ChatGPT Plus / Pro) installed and signed in on the hub. **API key**: Anthropic, OpenAI, or a service with an OpenAI-compatible API. **Local model**: [Ollama](https://ollama.com) installed on the hub; free, and your data never leaves that computer |
| A phone or tablet | A modern browser: Safari on iPhone / iPad, Chrome on Android. No app to install |

**You don’t need** a server, a domain, router changes, a new account anywhere, or sudo. The hub and your devices all use the project’s public connection service (`relay.ixxzh.com`). It’s free and only forwards encrypted data. To run your own, see [Self-host the connection service](self-host.md).

## Step 1: Install on the hub

**Option 1 — Ask your AI assistant (recommended).** Send this sentence to Claude Code, Codex, Cursor, or a similar assistant:

> Read https://raw.githubusercontent.com/wensuper/OmniLLM/main/INSTALL.md and install OmniLLM on this computer.

It reads the install script first, then downloads, verifies, and installs, and tells you what to do next. It won’t, and shouldn’t, ask you to paste any key into the chat.

**Option 2 — Run the installer yourself.** In a terminal (as a regular user, not with sudo):

```bash
curl -fsSL https://raw.githubusercontent.com/wensuper/OmniLLM/main/install.sh | sh
```

The install script does the following and changes nothing else:

1. Downloads the program for your system from this repository’s GitHub releases page and checks its SHA-256 checksum. If [cosign](https://github.com/sigstore/cosign) is installed, it also checks that the checksums were signed by this repository’s release workflow. If any check fails, it stops.
2. Puts the program at `~/.local/bin/omnillm`.
3. Registers a background service for the current user only (launchd on macOS, a systemd user service on Linux) that starts automatically when you log in.
4. Opens the local console (only in this computer’s browser, at a `127.0.0.1` address).

When a new version comes out, the hub updates itself while nobody is using it, so you don’t run this command again (see [Usage · Updating](usage.md#updating-and-uninstalling)). To look at the script before running it, replace `| sh` with `-o omnillm-install.sh`, read it, then run `sh omnillm-install.sh`.

If you close the console, reopen it at any time with:

```bash
~/.local/bin/omnillm open
```

## Step 2: Connect a model

Available model sources:

| Source | Needed on the hub | Phone, tablet, terminal chat | Scripts, background programs |
|---|---|---|---|
| Claude subscription (Pro / Max) | [Claude Code](https://claude.com/claude-code) installed and signed in | ✅ Text + images | ❌ Subscriptions are for people only |
| ChatGPT subscription (Plus / Pro) | [Codex](https://github.com/openai/codex) installed and signed in | ✅ Text + images | ❌ Subscriptions are for people only |
| Anthropic / OpenAI API key | Paste the key in the console | ✅ | ✅ |
| OpenAI-compatible service | Enter the address and key in the console | ✅ | ✅ |
| Local model (Ollama) | Install Ollama; it connects automatically | ✅ | ✅ |

If you only use your subscription on the web and don’t want to install Claude Code or Codex on the hub, you can’t use the subscription. OmniLLM doesn’t scrape web login sessions: that breaks the terms of service and is the quickest way to get an account banned.

### Use a subscription (Claude or ChatGPT)

Subscriptions work through the official command-line programs: [Claude Code](https://claude.com/claude-code) for a Claude subscription (Pro / Max), and [Codex](https://github.com/openai/codex) for a ChatGPT subscription (Plus, Pro, or another paid plan). If you only have the Claude or ChatGPT desktop app, that’s fine: sign in to the command-line program with the same account, and OmniLLM walks you through it:

- **Console**: the card in the first step says exactly what’s missing (“Found the Claude desktop app”, “Claude Code is installed but not signed in”). Click *Install and sign in* or *Sign in*. A Terminal window opens with every command written out: if the program isn’t installed, the official installer installs it first (no sudo, about a minute), then your browser opens and you confirm with your subscription account. When it’s done, the card changes to *Connected* by itself; there’s nothing else to click.
- **Terminal** (or a server with no browser):

  ```bash
  omnillm login claude     # Claude Pro / Max
  omnillm login chatgpt    # ChatGPT Plus / Pro
  ```

If you’re already signed in, the hub finds it and connects. Claude’s Sonnet, Opus, and Haiku, and Codex’s default model, appear in the model list with nothing to fill in.

Codex sometimes stores its sign-in in the system keychain, which OmniLLM can’t use. The console then asks you to sign in once more, and this sign-in is saved to a file OmniLLM can use (`~/.codex/auth.json`).

If installing or signing in didn’t work, the Terminal window tells you the next step, and the console card has *Open again*; click it once more. The free Claude plan can’t use Claude Code; you need Pro or Max.

The sign-in always stays with the official program. OmniLLM only asks it “are you signed in?” and never reads or copies the sign-in. When the hub calls the program, it turns off all of the program’s tools (running commands, reading and writing files, web search, plugins) and uses it as a text-only model, so nobody on a device can make it touch the files on your Mac.

### Use an API key

In the console, under *Models*, choose the provider, paste the key, and click *Check and connect*. The hub first verifies the key with one minimal request and saves it only if that succeeds. The key is stored in the system keychain (on Linux, an encrypted file only you can read).

**Paste keys only into the console.** Don’t paste them into a chat with an AI or type them on the command line: chats are sent to the AI provider, and command lines stay in your shell history.

On a Linux hub with no browser, run the following in its terminal and paste the key when prompted (the key is read from input and never appears on the command line):

```bash
~/.local/bin/omnillm hub provider add anthropic   # or openai
```

### Use a local model

Install [Ollama](https://ollama.com) and download a model (for example `ollama pull qwen3`); the hub finds it and connects automatically. If it doesn’t, run `~/.local/bin/omnillm hub provider add ollama`.

## Step 3: Add a phone or tablet

1. In the console, click *Devices → Add device*. A large QR code appears on the screen.
2. Scan the code with your phone’s **camera** and open the link. The web page shows which connection service it will connect through; tap *Add*.
3. Your computer and your phone show the same 6-digit number. **Check that they match**, then click *Matches, add* on the computer.
4. A chat screen appears on the phone. Send a message to try it.

A few notes:

- The QR code is valid for 5 minutes and works once. If it expires, click *Add device* to make a new one.
- To open it like an app: on iPhone, tap Safari’s *Share → Add to Home Screen*; on Android, tap *Add to Home Screen* in Chrome’s menu. A page opened from the Home Screen has its own storage, so the first time you need to pair once more inside it (you can scan the code with the camera right on the page, or paste the pairing text).
- By default each device can use every connected model, with a daily limit of $5 (an estimate; subscription models aren’t counted in money but limited by number of requests). You can change this under *Devices* in the console.
- Tablets work the same as phones. Pair each device once.

## Step 4: Add a Linux server or another computer

Machines without a camera use **pairing text** instead of a QR code. The server gets just the program: no background service, and no sudo.

1. In the hub’s console, click *Add device* and expand *Computer or server (no camera)*.
2. On that machine, run the line shown on the page:

   ```bash
   curl -fsSL https://raw.githubusercontent.com/wensuper/OmniLLM/main/install.sh | sh -s -- --device
   ```

   When it finishes, it asks you to paste the pairing text. (When an AI assistant or a script runs the install, there’s no prompt; run `omnillm connect pair` afterward and paste it then.)
3. Copy the pairing text from the console (it starts with `omnillm3-`) and paste it there. You carry the pairing text over yourself, for example by pasting it over SSH. Don’t send it to anyone or paste it into a chat.
4. As with a phone, check that the 6-digit numbers on both sides match, then click *Matches, add* on the hub.

On a machine that already has the program, just run `~/.local/bin/omnillm connect pair` and paste the pairing text.

The server has no background service and doesn’t update itself. When a new version is out, `omnillm connect` mentions it; run `omnillm update`.

Once it’s set up, in that machine’s terminal:

```bash
omnillm connect chat                                                 # multi-turn chat; Ctrl-C stops only the current answer
omnillm connect ask "Explain end-to-end encryption in one sentence"  # ask one question and exit
omnillm connect ask --image screenshot.png "What is in this image?"  # with an image
omnillm connect models                                               # see which models you can use
```

If you don’t choose a model, it uses Claude Sonnet by default (when there’s a Claude subscription).

The installer adds `~/.local/bin` to your PATH (in `~/.bashrc`, `~/.zshrc`, and so on), so in a new terminal you can type `omnillm` directly. In the terminal where you just installed, use the full path printed at the end of the install, `~/.local/bin/omnillm`.

## Step 5 (optional): Let programs use it

**Programs on the hub Mac**: in the console, under *Apps*, click *Add app* and give it a name. You get a token for that app alone, shown next to the address to enter (`http://127.0.0.1:8765/v1`) and settings you can copy as is. Each app has its own limit and can be paused or removed at any time. Only programs on this Mac can reach this interface, and every request must carry the token.

**Programs on other devices** (scripts, editor plugins, other AI apps) can use the hub’s models through an OpenAI-compatible local interface on the device:

```bash
omnillm connect run          # serves the interface at http://127.0.0.1:8766/v1 and prints where the token file is
```

Set the program’s API address to `http://127.0.0.1:8766/v1` and its key to the contents of the token file. The Anthropic-format `/v1/messages` works too.

**Programs can use only API keys and local models; they can’t see subscription models.** Subscriptions are only for people chatting by hand: running scripts and batch jobs on a subscription is the quickest way to get an account banned. This rule is in the code and can’t be turned off.

## Check that it works

On the hub, run:

```bash
~/.local/bin/omnillm status
```

It tells you whether the hub is running, how many models are connected, how many devices are paired, and which step is still left. When it says “All set”, you’re done.

Next: [Usage and FAQ](usage.md) · [Security](security.md)
