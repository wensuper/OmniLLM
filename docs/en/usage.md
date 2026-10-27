# Usage and FAQ

English | [简体中文](../zh-CN/usage.md)

If you haven’t installed OmniLLM yet, start with the [Quickstart](quickstart.md).

- [In the browser: phone, tablet, or computer](#in-the-browser-phone-tablet-or-computer)
- [In the terminal](#in-the-terminal)
- [Managing devices and limits](#managing-devices-and-limits)
- [Keeping the hub available](#keeping-the-hub-available)
- [Messages and what to do](#messages-and-what-to-do)
- [Updating and uninstalling](#updating-and-uninstalling)
- [FAQ](#faq)

## In the browser: phone, tablet, or computer

The web app works the same on a phone and on a computer; only the layout differs. On a phone the chat list is a drawer behind the menu button at the top left. In a wide window it is a sidebar with *Search chats* and your chats grouped by *Today*, *Yesterday*, *Previous 7 days*, and *Earlier*; `⌘\` hides or shows it.

A chat is named after its first message. Its name is shown at the top left of the chat (in the bar on a phone). Click it for *Rename* and *Delete*; double-click it to rename right away, then press Enter (Esc keeps the old name). In the chat list, the *…* button next to a chat has *Rename* and *Delete*, and on a computer a double-click on a chat renames it too.

- **Opening**: the page opens at once on the models your Mac listed last time, so you can type, pick a model, and send right away; a message sent before the connection is up goes out as soon as it is. The dot next to the model pulses until your Mac answers.
- **Auto**: new chats start on *Auto (picks the model)*, at the top of the model list. For each message your Mac picks the model from how hard the message is: a fast one for short small talk, the Sonnet / GPT-6-Sol tier for most questions, the strongest for code, error logs and words like “refactor”, “derive” or “step by step”, and only image-reading models for a message with an image. While it thinks, it says which model is thinking (“Claude Opus 5.5 is thinking”). A chat stays with the model that answered it last and moves up only when it gets harder; a move is shown as a line (“Switched to Auto · …”), and each reply says which model wrote it. Signed-out, at-limit or switched-off models are never chosen, and a model that fails before it starts answering is passed over for the next one. When several subscriptions can answer, the one used least lately goes first. To stay on one model, pick it in the list; *In Auto* under Models in the console keeps a model out of Auto.
- **Choose a model**: the model picker is in the message box. Models are grouped under *Subscriptions (no extra cost)*, *API key*, and *On your Mac*, and a dot shows whether each model works right now: green when it does, grey when your Mac can’t be reached, amber when the model needs something first, with the reason under its name (its subscription is signed out on your Mac, or has reached its usage limit until a given time). While an amber model is chosen, a notice above the message box says what to do.
- **Send images**: tap the image button to take a photo or pick one, or paste one; on a computer you can also drag pictures into the window. Up to 4 images per message. Photos are scaled down first (1600 pixels on the long side), and the original never leaves the device. Both Claude and Codex can read images; a model that can’t says so.
- **Voice input**: tap the microphone in the message box and speak; the button turns into a moving sound wave and your words appear in the box as you talk, after anything you already typed. Tap it again, press Esc, start typing, or pause to stop. The words only fill the box and are never sent by themselves. Your browser does the recognition: Chrome and Edge send the audio to Google, Safari to Apple. Browsers without speech recognition, such as Firefox, show no microphone. The first time, the browser asks for microphone access.
- **Stop**: tap *Stop* (or press Esc) while an answer is coming in. It stops only that answer; the next message works as usual.
- **Chat history**: kept in this browser, encrypted. With subscription models (Claude, ChatGPT, Gemini), your Mac also keeps a copy so it can continue the chat, and deletes it after a day without use. In a long chat, only the two most recent messages with images resend their images, so requests don’t keep growing.
- **The *Your Mac* menu**: *Your Mac* at the bottom of the chat list says whether your Mac is online. Its menu shows this device’s name, *Language* (*Follow your Mac*, 中文, or English), *Keyboard shortcuts*, where chats are kept, and *Remove this device*. The page speaks your Mac’s language (the one chosen in the console or the menu bar) unless you pick one here.
- **Several tabs**: only one tab saves your chats at a time. The tab you come back to takes over by itself, with what the other tab saved.
- **No network**: when the device goes offline, you’re told right away and what you just typed stays in the message box. It reconnects automatically once you’re back online.
- **Your Mac is off or asleep**: you see “Can’t reach your Mac”. It recovers automatically when your Mac wakes, with no need to pair again.
- **Remove this device**: *Remove this device* in the *Your Mac* menu deletes the pairing and the chats in this browser and removes the device from your Mac as well. If your Mac can’t be reached at that moment, the page tells you to remove it under *Devices* on your Mac later.

Keyboard shortcuts on a computer (Ctrl instead of ⌘ on Windows and Linux):

| Keys | Does |
|---|---|
| `⌘⇧O` | New chat |
| `⌘K` | Search chats |
| `⌘\` | Hide or show the sidebar |
| `Esc` | Stop the answer |
| `⌘⇧;` | Copy the last answer |
| `/` | Go to the message box |
| `Enter` / `⇧Enter` | Send / new line |
| `⌘/` | Keyboard shortcuts |

## In the terminal

On a computer or server installed with `install.sh --device` and paired:

| Command | What it does |
|---|---|
| `omnillm connect chat [--model ID]` | Multi-turn chat. `/image FILE` attaches an image, Ctrl-C stops the current answer, Ctrl-D exits |
| `omnillm connect ask [--model ID] [--image FILE] "question"` | Ask one question and exit |
| `omnillm connect models` | List the models this device can use |
| `omnillm connect run` | Serve an OpenAI / Anthropic-compatible local interface at `127.0.0.1:8766` for programs on this machine |
| `omnillm connect pair` | Paste new pairing text (to switch hubs or pair again) |
| `omnillm connect forget` | Remove this device (from the hub too) |

Subscription models work only when you type by hand in a real terminal. Calls through a pipe, cron, a script, or the local interface all count as background programs and can use only API keys and local models. For why, see [Security](security.md#avoiding-account-bans).

## Managing devices and limits

Open the console on the hub: click the OmniLLM icon in the Mac menu bar → *Open the console…*, or run `~/.local/bin/omnillm open` (add `--add` to open straight on the QR code for adding a device).

**Menu bar icon** (macOS): appears automatically at login once installed, and always stays there. A glance tells you whether your phone can connect right now (solid: yes; dashed orbit, hollow: no). The menu’s top line says the same in words (such as “Online · phones and other devices can connect”). Below it, *Devices* and *Model sources* show how many you have and open those pages of the console; then come *Add device…* and *Open the console…*, the version with *Check for updates…*, and *Language · 语言*, which is the same setting as the language button in the console. When OmniLLM isn’t running, the menu has *Start OmniLLM*. If the icon is missing, run `omnillm tray`. If you use a menu bar organizer (such as Ice or Bartender), the new icon may land in the hidden section; drag it out.

| Page | What you can do |
|---|---|
| Devices | *Add device* (see [Quickstart](quickstart.md#step-3-add-a-phone-or-tablet)). Each device has an *On* switch: off pauses it at once and keeps its pairing and settings. The *Models* column says *All 4* when the device can use every model and *2 of 4* when only some; hovering over it, or over a device’s name, shows a card with the models, or with when it was added, last seen, and today’s use. *Edit* changes the name, the daily limit (*No limit* is an option), and the models: with *All models* on, models connected later are included too; switching off a single model turns *All models* off and leaves the rest on. *Remove* takes effect immediately |
| Models | Connect API keys and local models. Under *Model sources*, each source has an *On* switch: off, devices and apps can’t use its models for now, but its sign-in or key is kept and it works again as soon as it’s on; *Remove* deletes the key. See where each model’s messages go (*This Mac only* or *Cloud*), rename a model or hide it on devices, and choose which models are *Chat only* (scripts and apps can’t call them; always on for subscriptions). If apps or programs-only devices use a model, turning on *Chat only* first lists them and asks |
| Apps | Tokens for scripts and programs on the hub (see [Quickstart](quickstart.md#step-5-optional-let-programs-use-it)); the same *On* switch, *Models* column, and *Edit*. Apps can use only API keys and local models |
| Usage | Calls and spend for each device and app today, yesterday, or over the last 7 days, also broken down by model (estimated at the price set for each model, else the provider’s published price, and high where neither is known: see `omnillm hub models`). Removed ones are merged into one row, with a total at the end |

The top of the page shows today’s spend and the **daily total for the whole hub** (default $20). Click *Change* to change it, or set no limit.

The command line can do the same things; see `omnillm help`:

```bash
omnillm hub devices                 # list devices
omnillm hub revoke <name or ID>     # remove a device
omnillm hub device set <name or ID> --paused true   # pause a device
omnillm hub provider off <source>   # switch a model source off (on: switch it on)
omnillm verify web                  # check the web app file by file against the signed release
omnillm hub models                  # connected models
omnillm hub usage                   # usage
```

**Limits**: each device gets $5 a day by default (estimated). You can change it, or choose *No limit*, in which case only the hub’s daily total applies; the daily total can also have no limit. Subscription models aren’t counted in money (the subscription is already paid for) but by number of requests: 60 an hour and 300 a day per device, and 12 a minute for the whole hub, one at a time.

## Keeping the hub available

Devices work only while the hub is on and online.

- **When your Mac’s lid is closed or it’s asleep, no device can use it.** Everything recovers when it wakes. To keep it available, turn on *Prevent automatic sleeping on power adapter when the display is off* under *System Settings → Battery (or Energy)*, or use an always-on Mac mini or Linux server as the hub.
- After the hub restarts or updates, the background service starts by itself and devices reconnect automatically, with no need to pair again.
- If the hub changes networks (say you take your laptop out), that’s fine: devices follow it and reconnect automatically.

## Messages and what to do

| You see | Cause | What to do |
|---|---|---|
| Can’t reach your Mac: it may be asleep or offline | The hub is asleep, off, or offline | Wake the hub; it recovers automatically |
| This device is offline | The phone itself has no network | It recovers once you’re online; what you typed is kept |
| Your Claude subscription (or ChatGPT, Gemini subscription) on your Mac needs you to sign in again | The subscription sign-in expired | On the hub, run `omnillm login claude` (or `chatgpt`, `gemini`), or choose *Sign in* in the console |
| To use …, install Claude Code / Codex on your Mac and sign in first | It isn’t installed on the hub, or was uninstalled | On the hub, run `omnillm login claude` (or `chatgpt`), or use the card in the console |
| Subscriptions are used at one person’s chatting pace, so requests are spaced out | Too many requests within a minute | Wait the time shown; if your turn comes within 20 seconds, it waits automatically instead of failing |
| This device is using the subscription models faster than a person chats | Over 60 requests an hour or 300 a day | Try again later, or switch to a model that uses an API key |
| … has reached its usage limit | Claude’s or ChatGPT’s own usage limit | Wait for it to reset (the message says roughly when) |
| This model is chat-only | A script or program called a subscription model | Switch to an API key or a local model |
| This model’s source is switched off on your Mac | Its model source is switched off under *Models* | Switch it on in the console |
| This device is paused on your Mac | The device’s *On* switch is off | Switch it on under *Devices* in the console |
| This version of … on your Mac couldn’t be kept from using tools | The official program updated and failed the safety check | Update OmniLLM; until then, that subscription is unavailable |
| This device’s daily limit is used up | This device reached its daily limit | It resets tomorrow, or raise it in the hub’s console |
| The … API key (ending …) doesn’t work | The key was revoked or is out of credit | Replace the key in the console |
| That image can’t be used | Wrong format or too large | PNG, JPEG, GIF, or WebP, up to 2 MB each and 4 per message |
| This model can’t read images | The chosen model doesn’t support images | Switch to Claude or Codex, or send text only |
| This device was removed from your Mac | It was removed on the hub | If you need it, add it again on the hub |
| This QR code or pairing text expired or was already used | The QR code is over 5 minutes old, or was already used | Click *Add device* on the hub again |
| The pairing text is incomplete or altered | Characters were lost or changed when copying | Copy it again, in full, on the hub |
| This isn’t an OmniLLM pairing text | You copied the wrong thing | Copy it from *Add device* in the hub’s console |
| Remote access is unavailable right now | The public connection service is under maintenance or has network trouble | It reconnects automatically; if it keeps failing, check the [project page](https://github.com/wensuper/OmniLLM) for announcements |

If it still doesn’t work, the hub’s log is at:

- macOS: `~/Library/Application Support/OmniLLM/hub/hub.log`
- Linux: `~/.config/OmniLLM/hub/hub.log`

The log contains no chat content and no keys. When you open an issue, you can attach the last few dozen lines (look them over yourself before sending).

## Updating and uninstalling

**Updates take care of themselves**:

- **Hub**: checks for a new version a few minutes after startup and every 6 hours after that. When there is one, it installs it once nobody has used it for 10 minutes. The restart takes only a few seconds, and your phone and other devices reconnect by themselves. If the new version fails to start, the hub rolls back to the previous version and won’t install that version automatically again.
- **The web app** (phones, tablets, computers): switches to the new version when you open it; unsent text is kept.
- **Linux servers (used only as devices)**: have no background service and don’t update themselves; when there’s a new version, `omnillm connect` mentions it.

**0.2.0-alpha.22 and earlier** don’t have automatic updates or `omnillm update` yet: run the install command once more (your settings are kept), and you won’t need to do anything after that.

**To update right away**, run this on the hub or server (the hub restarts for a few seconds, and any answer coming in on your phone is interrupted):

```bash
omnillm update            # update if there's a new version; says so if already up to date
omnillm update --check    # only check whether there's a new version
```

On a Mac you can also click the OmniLLM icon in the menu bar → *Check for updates…*; when a new version is available, that item changes to *New version … · click to update*. `omnillm status` tells you whether there’s a new version and whether it’s waiting for an idle moment to update automatically.

**Turn off automatic updates**: `omnillm update --auto off` (to turn them back on: `--auto on`). After that, new versions are only announced, not installed.

An update is simply a fresh install of the latest release (all releases are still pre-releases): the same install script, with the same checksum check (and signature check if cosign is installed), keeping your settings, keys, and paired devices. Running the install command again has the same effect; on a server used only as a device, it updates it as a device and doesn’t turn the server into a hub.

**Uninstall the hub**:

```bash
~/.local/bin/omnillm uninstall          # stop and remove the background service; keep data and keys
~/.local/bin/omnillm uninstall --purge  # also delete data and saved keys
rm -f ~/.local/bin/omnillm
```

`--purge` deletes only the keys saved on this computer. The keys are still valid with the provider; if you no longer use them, revoke them in the provider’s dashboard. The subscription sign-in belongs to Claude Code / Codex, and uninstalling OmniLLM doesn’t touch it.

**Uninstall a device**: on a server, run `omnillm connect forget`, then delete `~/.local/bin/omnillm`; in a browser, use *Remove this device* in the *Your Mac* menu. If the hub can’t be reached at that moment, remove the device under *Devices* on the hub later.

## FAQ

**Does it cost anything?**
OmniLLM is free and open source (Apache-2.0), and the public connection service is free too. You pay for models the way you already do: subscriptions as usual, APIs by usage, and local models are free.

**Will my account get banned?**
OmniLLM tries to make your usage look like what it is, you on your own computer: requests are sent only from the hub, subscriptions are only for chatting by hand, requests are rate-limited, and it backs off when throttled. But whether you can use a service, and whether you get limited, is ultimately up to the provider, so follow their terms. See [Security](security.md#avoiding-account-bans) for details.

**Can I share it with family or coworkers?**
No, and that’s by design. A hub serves only one owner and their own devices: sharing a subscription with others already breaks most providers’ terms and is the quickest way to get banned.

**Who runs the public connection service? Can it see my chats?**
The project’s author runs it at `relay.ixxzh.com`. It only forwards encrypted data and can’t see chat content; it can see metadata such as IP addresses, connection times, and traffic volume. If that concerns you, you can [host your own](self-host.md): one program and one command.

**Why is there no app to install on the phone?**
The web app runs the same encryption code as the command-line client (as WebAssembly), so end-to-end encryption happens right in the browser. The trade-off is that the first time you open the page, you have to trust whoever hosts it; see [Security](security.md#the-web-app).

**Can the hub be a Linux server?**
Yes. Install it the same way, then in the server’s terminal run one command to connect a model: `omnillm login claude` (or `omnillm login chatgpt`) for a subscription, which gives an address and a code to open on your phone, or `omnillm hub provider add anthropic` (or `openai`) to paste an API key. Once it’s connected, the same terminal shows a QR code: scan it with your phone, and if the 6 digits match, type y. To add another phone later: `omnillm pair`. If you forget how it works, type just `omnillm`: it says where things stand, the one thing to do next, and the few commands people use most.

**What about Windows?**
Windows can’t be the hub yet. A Windows computer can open the web app in a browser and be used as a device.

**Can I use it in my own programs?**
Yes. Run `omnillm connect run` on a device to get an OpenAI-compatible local interface (API keys and local models only). Go programs can also embed OmniLLM as a library; see the [design document](../design.md#24-库优先一套代码两种用法) (in Chinese).
