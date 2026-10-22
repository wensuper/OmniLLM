# Usage and FAQ

English | [简体中文](../zh-CN/usage.md)

If you haven’t installed OmniLLM yet, start with the [Quickstart](quickstart.md).

- [On your phone and tablet](#on-your-phone-and-tablet)
- [In the terminal](#in-the-terminal)
- [Managing devices and limits](#managing-devices-and-limits)
- [Keeping the hub available](#keeping-the-hub-available)
- [Messages and what to do](#messages-and-what-to-do)
- [Updating and uninstalling](#updating-and-uninstalling)
- [FAQ](#faq)

## On your phone and tablet

- **Choose a model**: use the drop-down next to the message box. Models marked *subscription* run on your Claude / ChatGPT subscription.
- **Send images**: tap the image button to take a photo or pick one from your library, or paste one directly. Up to 4 images per message. Photos are scaled down on the phone first (1600 pixels on the long side) before they’re sent, and the original never leaves the phone. Both Claude and Codex can read images; a model that can’t says so.
- **Voice input**: tap the microphone in the message box and speak; the button turns into a moving sound wave and your words appear in the box as you talk. Tap it again, press Esc, start typing, or pause for a few seconds to stop. The words only fill the box and are never sent by themselves, so you can edit them first. Your browser does the recognition (Chrome and Edge use Google, Safari uses Apple); browsers without it show no microphone. The first time, the browser asks for microphone access.
- **Stop**: tap *Stop* while an answer is coming in. It stops only that answer; the next message works as usual.
- **Chat history**: kept only in this phone’s browser, encrypted. The hub doesn’t keep chats. Open earlier chats or start a new one from the top left. In a long chat, only the two most recent messages with images resend their images; earlier images are noted in the text, so requests don’t keep growing.
- **No network**: when the phone goes offline, you’re told right away and what you just typed stays in the message box. It reconnects automatically once you’re back online.
- **Hub is off**: you see “Can’t reach your Mac”. It recovers automatically when the hub wakes up, with no need to pair again.
- **Remove this device**: *Remove this device* in the menu deletes the pairing and the chats on this phone and removes it from the hub as well. If the hub can’t be reached at that moment, the phone tells you to remove it under *Devices* on the hub later.

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

**Menu bar icon** (macOS): appears automatically at login once installed. A glance tells you whether your phone can connect right now (solid: yes; dashed orbit, hollow: no). Its menu has *Open the console…*, *Add device…*, and *Check for updates…*; when OmniLLM isn’t running, it has *Start OmniLLM*. If you remove it from the menu bar, it comes back at your next login. If you use a menu bar organizer (such as Ice or Bartender), the new icon may land in the hidden section; drag it out.


| Page | What you can do |
|---|---|
| Devices | *Add device* shows a QR code; × in the top right closes it (the code stops working at once). See when each device was last used. *Edit* changes the name and daily limit (*No limit* is an option). *Remove* takes effect immediately |
| Models | Connect or remove API keys and local models; each model source has an *On* switch: off, devices and apps can’t use its models for now, but its sign-in or key is kept and it works again as soon as it’s on (*Remove* deletes the key); see where each model’s messages go (this computer / cloud); choose which models are *Chat only* (scripts and apps can’t call them) |
| Apps | Issue tokens to scripts or programs on the hub; *Edit* changes the name and daily limit. Apps can use only API keys and local models |
| Usage | Today’s calls and spend for each device and app (estimated from the providers’ published prices). Removed ones are merged into one row, with a total at the end |

The top of the page shows today’s spend and the **daily total for the whole hub** (default $20). Click *Change* to change it, or set no limit.

The command line can do the same things; see `omnillm help`:

```bash
omnillm hub devices                 # list devices
omnillm hub revoke <name or ID>     # remove a device
omnillm hub provider off <source>   # switch a model source off (on: switch it on)
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
| Claude Code / Codex on your Mac is signed out | The subscription sign-in expired | In a terminal on the hub, run `claude` (or `codex login`) to sign in again |
| Claude Code / Codex not found on the hub | It isn’t installed on the hub, or was uninstalled | Install it on the hub and sign in |
| Subscriptions are used at one person’s chatting pace, so requests are spaced out | Too many requests within a minute | Wait the time shown; if your turn comes within 20 seconds, it waits automatically instead of failing |
| This device is using the subscription models faster than a person chats | Over 60 requests an hour or 300 a day | Try again later, or switch to a model that uses an API key |
| … has reached its usage limit | Claude’s or ChatGPT’s own usage limit | Wait for it to reset (the message says roughly when) |
| This model is chat-only | A script or program called a subscription model | Switch to an API key or a local model |
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
- **Web page on phones and tablets**: switches to the new version when you open it; unsent text is kept.
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

**Uninstall a device**: on a server, run `omnillm connect forget`, then delete `~/.local/bin/omnillm`; on a phone, use *Remove this device* in the menu. If the hub can’t be reached at that moment, remove the device under *Devices* on the hub later.

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
The web client runs the same encryption code as the command-line client (as WebAssembly), so end-to-end encryption happens right in the browser. The trade-off is that the first time you open the page, you have to trust whoever hosts it; see [Security](security.md#the-web-client-on-your-phone).

**Can the hub be a Linux server?**
Yes. Install it the same way. Without a browser, connect API keys in the terminal with `omnillm hub provider add`, and show the pairing QR code and confirm in the terminal too (`omnillm hub pair`). To use a subscription, install Claude Code or Codex on the server and sign in.

**What about Windows?**
Windows can’t be the hub yet. A Windows computer can open the web client in a browser and be used as a device.

**Can I use it in my own programs?**
Yes. Run `omnillm connect run` on a device to get an OpenAI-compatible local interface (API keys and local models only). Go programs can also embed OmniLLM as a library; see the [design document](../design.md#24-库优先一套代码两种用法) (in Chinese).
