# Changelog

English | [简体中文](docs/zh-CN/CHANGELOG.md)

All notable changes to OmniLLM are documented here. OmniLLM is in early pre-release (0.2.0-alpha); versions follow [Semantic Versioning](https://semver.org/).

## [0.2.0-alpha.89](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.89) - 2026-10-30

### Changed

- Web app: your own messages show exactly as you typed them, as in Claude: a `#`, `1.`, `**` or `<b>` you type or paste is no longer turned into a heading, list or bold text. Web addresses are still links.

### Fixed

- Web app: while a reply arrives, a link shows its words instead of half-written `[text](https://…` syntax, and a formula appears once it is complete instead of as raw TeX first. Long code in a reply that just finished no longer folds away while you are reading it; it folds when the chat is opened again later.

## [0.2.0-alpha.88](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.88) - 2026-10-30

### Changed

- Web app: a reply that is still arriving no longer redraws itself whole with every word. What is already finished stays as drawn, so long replies stay smooth, and you can click a link, copy a code block or select text in them before the reply ends.

### Fixed

- Web app: Markdown shows as it does in Claude in more cases. A numbered list with blank lines between its items, or with paragraphs and code blocks inside an item, stays one list; code shown inside ```` fences, ~~~ fences and languages such as `c#` display correctly, and ``` in the middle of a sentence stays text; `\*`, `\_`, `\$` and other escaped marks show without the backslash; ***bold italic*** and *italic* inside **bold** render; links with parentheses (as on Wikipedia) or with a title work.

## [0.2.0-alpha.87](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.87) - 2026-10-29

### Fixed

- Web app: some unusual replies could freeze the page for seconds or keep a chat from opening (very long lines, thousands of nested quotes, a long data URL in CSS code). They now show as plain text where formatting would be slow, and a reply that can’t be formatted shows as text instead of breaking the chat. Long replies with formulas and code redraw less while they arrive.
- Web app: a paragraph starting with an inline formula ($$a$$ …) is no longer read as one big formula; `[^1]:` inside code stays code; a table row with an extra cell keeps it; `Vec<u>` in text stays text; a lost connection no longer shows the same thinking twice; when Auto switches models before answering, the first model’s thinking is dropped.
- Thinking: a server that sends reasoning in an unexpected form, or under two names, no longer breaks or doubles it; the local API’s stream keeps the first piece of reasoning.

## [0.2.0-alpha.86](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.86) - 2026-10-29

### Added

- A thinking model’s reasoning now shows above its answer, as in Claude: “Thinking · 7s” with its latest lines while it thinks, then “Thought for 12s”, which you can open to read it all. It works with models that show their thinking: Ollama models such as Qwen3.5, DeepSeek, Claude Code and Codex. Your Mac needs this version too (`omnillm update`). Apps using OmniLLM’s local API get it as `reasoning_content`.

## [0.2.0-alpha.85](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.85) - 2026-10-29

### Added

- Web app: a diff shows added lines in green and removed lines in red; HTML, CSS, shell variables and the keys in JSON and YAML get their own colors. Code longer than 30 lines folds once the reply is done (“Show all N lines”). Tables have “Copy table”, which pastes into a spreadsheet as cells. A picture from another site shows as a link you can open. Footnotes ([^1]) are numbered and listed at the end of the reply; tap a number to jump to its note.

## [0.2.0-alpha.84](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.84) - 2026-10-29

### Added

- Web app: replies are laid out the way Claude shows them. Tables have a header and row lines and scroll sideways on a phone; headings come in sizes; quotes, rules, nested lists, checklists, struck text and links show as such; code is colored by language; math written in TeX ($…$, $$…$$) shows as formulas. A few HTML tags models use for layout (`<br>`, `<sub>`, `<sup>`, `<kbd>`, `<mark>`…) work too. Anything else a model writes in HTML stays plain text, so a reply can’t run code in the page.

## [0.2.0-alpha.83](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.83) - 2026-10-29

### Added

- Web app (phone and computer): point at a message, or tap it on a phone, to see its time to the second. A question shows when you sent it; a reply shows when it finished, how long its first words took and how long it took in all (“14:42:59 · waited 12s · 18s in all”). Point at the time for the whole date. A chat that runs over several days has a line where each day begins. Nothing new shows otherwise, and chats from before this version show no times.

## [0.2.0-alpha.82](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.82) - 2026-10-27

### Changed

- A model that can’t read images (such as a text-only local llama.cpp model) no longer fails a chat because of pictures in earlier turns: they are left out, the model is told a picture was there, and it answers. Your Mac remembers the model, so later turns skip the pictures at once. Only a picture in the message you are sending still gets “this model can’t read images”.

## [0.2.0-alpha.81](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.81) - 2026-10-27

### Fixed

- A text-only local llama.cpp model (llama-server without a vision projector) refuses a chat that holds a picture with “image input is not supported”. That showed “The model source had an error, try again later” and counted as a source failure. It now says the model can’t read images and this chat has one: pick a model that can, or start a new chat with text only.

## [0.2.0-alpha.80](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.80) - 2026-10-27

### Fixed

- A chat longer than a local llama.cpp model’s window (llama-server’s “exceeds the available context size”) showed “The model source had an error, try again later.” It now says the chat is too long for that model and to start a new chat, as other sources already did. LM Studio’s wording is recognised too.

## [0.2.0-alpha.79](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.79) - 2026-10-27

### Fixed

- Web app: when a subscription model hits its usage limit (or is signed out) during a reply, the model picker marks it right away: the dot turns to the warning colour and the model shows “At its limit, back at …”. Before, it kept a green dot until the page next reloaded the list, so it looked usable.

## [0.2.0-alpha.78](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.78) - 2026-10-27

### Fixed

- Gemini subscription on a Mac: the dialog “A keychain cannot be found to store “antigravity”” kept coming back. OmniLLM's Antigravity now has its own keychain inside OmniLLM's folder, so it never touches your own keychain and no longer asks. If you saw the dialog, choose *Cancel*, not *Reset To Defaults*, which resets your own keychain.

## [0.2.0-alpha.77](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.77) - 2026-10-27

### Changed

- ChatGPT sign-in on a machine without a browser names ChatGPT's real setting: turn on *Device code authorization* in Settings → Security.
- Docs quote the console's *Sign in* button in the same style as other buttons.

## [0.2.0-alpha.76](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.76) - 2026-10-27

### Changed

- Same words on every surface, last polish:
  - The QR line spacing in Chinese is fixed.
  - The computer welcome's “The web app is checkable” line now names the web app.
  - INSTALL, README and the quickstart's requirements list Gemini and “the web app” like the rest.

## [0.2.0-alpha.75](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.75) - 2026-10-27

### Changed

- Same words on every surface, fourth pass:
  - A machine with no screen is no longer offered Gemini (its sign-in needs a screen), and the quickstart says so.
  - The quickstart's subscription table and heading include Gemini.
  - Hints quote the console's full label, *No browser on that computer? Use the command line*.
  - The computer welcome's trust line reads like the console picture (“The link expires in 5 minutes · single use”).

## [0.2.0-alpha.74](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.74) - 2026-10-27

### Changed

- Same words on every surface, third pass:
  - In the console, a signed-out source's chip says “Sign in again” (the state). It used to repeat the button label.
  - The add-device steps and the `verify web` help say “the web app”.
  - The security docs quote the question the web app really asks (“Add this phone?”).
  - The chat-history notes name Gemini too.
  - The command line points free models at the console's “Install Ollama” line.

## [0.2.0-alpha.73](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.73) - 2026-10-27

### Changed

- Same words on every surface, second pass:
  - The command line, the errors the web app shows and the docs now say “the web app”, “Gemini subscription” and “sign in again” too.
  - Pairing texts and QR codes say they expire in 5 minutes and work once.
  - On a computer that joins with a pairing text, the number check reads “Matches, add” in the console or y in the terminal, as on the web.
  - The phone's welcome line also says what to run when the hub isn't a Mac (`omnillm open --add`).
  - English says “sign-in”, not “login”.

## [0.2.0-alpha.72](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.72) - 2026-10-27

### Changed

- The Mac (menu bar, console, command line), the phone page and the computer page now use the same words for the same things:
  - The web page is “the web app” everywhere, since computers use it too, not only phones. This covers the console's check warning, the menu bar line and `omnillm verify web`.
  - The Gemini subscription is called that in English too, not “Google AI subscription”. The console's first step offers “I have a Gemini subscription”, next to Claude and ChatGPT.
  - A signed-out subscription reads “Sign in again” on every surface.
  - The phone's welcome line gives the same path as the computer's step 1: the menu bar icon, then *Add device…*. The computer's step 1 adds what to run when the hub isn't a Mac (`omnillm open`).
  - The console picture on the computer's welcome screen says “Expires in 5 minutes · single use”, as the console does.
  - The number check says to confirm in the console *or* type y in the terminal.
  - Browsers that can't keep the pairing recommend Safari, Chrome or Edge everywhere.
  - Texts from a hub that isn't a Mac no longer say “this Mac” or “Terminal”, and an error no longer calls the hub “your server”.

### Fixed

- *Add device…* in the menu bar did nothing before any model was connected; it now says to connect a model first.

## [0.2.0-alpha.71](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.71) - 2026-10-26

### Changed

- Web page on a computer: clearer words on the welcome screen. The title is “Use your OmniLLM / in this browser”; the line under it says OmniLLM runs on one of your own computers, where your models and subscriptions are, and that pasting a link from it lets this browser use them (even on that same computer). Step 1 goes straight to the menu bar's *Add device…*.

## [0.2.0-alpha.70](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.70) - 2026-10-26

### Changed

- Web page on a computer: a redesigned welcome screen. A two-line title (“Bring your OmniLLM / to this computer”); the three steps written out above the paste box and numbered the same way on the console picture beside it (open the console, *Add device* → *Computer or server*, *Copy link*); “Works for 5 minutes · single use” as the console says it; *Don’t have OmniLLM yet?* right under the box; and three lines at the bottom on why it is safe (end-to-end encrypted, a link works once for 5 minutes, the page can be checked with `omnillm verify web`). Phones are unchanged.

## [0.2.0-alpha.69](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.69) - 2026-10-26

### Fixed

- Phone and web page: a page left open (a welcome screen on a computer, a phone tab kept for days) switches to a new version by itself within a few minutes, when nothing is in progress. Before, it checked every half hour, and a page opened for the first time never heard of the next version until it was reloaded.

## [0.2.0-alpha.68](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.68) - 2026-10-26

### Changed

- Spend: API-key models you haven't priced are now counted at the provider's published standard price, built in for common Anthropic, OpenAI, Gemini, DeepSeek and Groq models (read from their pricing pages on 2026-10-26), instead of $15 in / $75 out per million tokens for everything — which made daily limits stop cheap models up to ~100× too early. A price you set still wins. It only applies to the provider's own API address, by exact model ID (a dated snapshot counts as its model); a proxy, an unlisted model or one priced "contact sales" is still counted high. Long prompts pay the long-context price where the provider charges one, and announced price changes apply from their date. `omnillm hub models` marks each price as yours, "(published)" or "(est.)"; the console's spend notes say the same.

## [0.2.0-alpha.67](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.67) - 2026-10-26

### Changed

- The Mac console is redesigned. A sidebar holds Overview, Devices, Models, Apps and Usage, with this Mac's state at the bottom. The new Overview shows what needs you, online devices, models, today's spend and calls, and today's use by device and app. Warnings sit at the top of every page. Devices get type icons and add-device opens as a dialog. Model sources are cards with their state, sign-in, switch and remove. Apps always show how to connect. Usage rows get a bar by calls. First-run setup puts subscriptions and an API key side by side. A narrow window gets a top bar.
- Menu bar: when a model source needs signing in again or is at its limit, a line under the status says so; clicking it opens the console's Models page. A source you switched off is no longer counted.

## [0.2.0-alpha.66](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.66) - 2026-10-26

### Fixed

- Phone page, in Chinese: the Home Screen steps for Safari name its button as Safari does (“共享”, not “分享”).

## [0.2.0-alpha.65](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.65) - 2026-10-26

### Changed

- Phone page, from a real iPhone: Safari no longer has to add the page to the Home Screen first. You pair right there, and *Put it on the Home Screen, like an app?* under the welcome screen shows the steps if you want it. Opened inside WeChat or another app, *Use it in WeChat* is now the main button, with *Copy the address for your browser* second; the corner-menu hint only appears for apps it is right for. In a browser that can't keep the pairing, *Use it here* comes first. The wrong steps (“⋯ at the top right”, “Open in Safari”) no longer show in Safari.

### Fixed

- Auto isn't offered when you've taken every model out of it (it could only fail). A program asking for Auto where only chat-only models are left is told that, not that models are signed out. Auto names each model once: the model's own name no longer arrives as a second one mid-reply, which overwrote a name you gave it and read as a switch.
- Updates and the web page check no longer stop when GitHub limits this computer's network address (60 API checks an hour without a login, quickly reached by several computers on one network): they read GitHub's release feed instead. If GitHub refuses that too, `omnillm update` says it is GitHub's limit and to try again in a while, not that the network is down.
- Phone page: the "chats stay on this device" note names Google AI too; a browser that can't keep chats warns once, on the first chat only.
- Console: a connected subscription whose source is switched off reads "Source off", not "Connected".
- CLI: `connect pair --no-limit`; an already-paired device says so first, whoever runs `connect pair`; `hub provider add gemini` (google, antigravity, agy) points to `omnillm login gemini`, and `hub provider off gemini` works; `uninstall --purge` removes the Google sign-in too; on a device only an install-started hub may `hub run`; `status` on a removed device exits 1 and says to run `connect forget`; a wrong `connect chat --model` gives a runnable command.
- Console: a switched-off subscription no longer says "Ready to use" under "Source off".
- CLI: status and the `omnillm` menu offer the Google AI subscription (`omnillm login gemini`); on a device-only computer `omnillm connect` and `-h` show the device's commands; `hub region confirm` checks the hub is running first; `hub pair` refuses `--budget-usd` with `--no-limit`, and the command it hands an AI assistant keeps the options; `hub app add` says when the local API for apps isn't running; `hub provider off|on` takes `google` and `agy`; model names and IDs are cleaned of control characters in the terminal; the `login` help line no longer runs into its description.
- Phone page: a reply keeps its model's name on top while it streams; under Auto the chosen model (or the "Switched to" line) shows as soon as the Mac names it.
- CLI: "(Auto · model)" is said again when another model takes over; `omnillm hub region` gives its usage; `connect models` from a script marks Auto chat-only when everything it could pick is.
- CLI: `-h`/`--help` after any hub subcommand shows the help; `hub audit` refuses extra words; `status` with remote access off says it's OMNILLM_RELAY=off and how to add devices; `help --all` mentions "in Auto" for `hub model set`. The sqlite3 used to clean Gemini's records is checked like the subscription programs.
- Spend: the console no longer says API keys are counted "at their official prices". There is no price list: a model without a price you set is counted high ($15 in / $75 out per million tokens), and the console now says so and how to set one. `hub models` shows each model's price, marking estimates; `hub model`'s usage lists `--price-in/--price-out`.
- Phone page: if a pairing started right after another one succeeded is cancelled or fails, the chats open properly (not a blank chat); a new chat whose first message got no reply loses that message's name. CLI: a bad price says the range; empty `--json` lists are `[]`, never `null`; a hub built from source (version …-dev) doesn't update itself to a release.
- Phone page: a pairing link opened in the 1.2 s after another pairing succeeded is no longer covered by the chats. The hub's daily web-page check cleans its address like `verify web` (an OMNILLM_WEB_APP ending in index.html was never checked). A wrong local API or `--listen` address names the port range.
- Phone page: the same pairing link tapped twice is handled once; another link that waited during a pairing opens after the chats do (it could leave the pairing stuck). `verify web` accepts an address copied from the phone (drops `#pair=…`, a query, index.html); OMNILLM_API needs a port from 1 to 65535.
- `verify web`: deciding that cosign couldn't reach Sigstore reads only cosign's own words, not the certificate it quotes (a signer could name themselves "tuf" and turn a rejection into "couldn't check"); a 404 at the address says to check it.
- Phone page: a pairing link stays in the address until the saved pairing is read (a reload while loading keeps it); a link opened during a reply or a pairing waits for it to end.
- CLI and console: `install` stops on an OMNILLM_API that isn't 127.0.0.1:PORT or off; commands in the hub's messages are written as this terminal can run them; the hub log and the console's Apps tab say how to turn remote access or the local API back on when a setting turned them off.
- `verify web`: the signature is checked on the very checksums.txt the file list was checked against; with cosign installed, missing signature files are a bad signature (every release has them), while cosign failing to reach Sigstore is "couldn't check"; a release whose file list disowns its checksums is said as such. `hub app add` gives the command that turns the local API back on, and says when its address isn't on this computer.
- CLI: `verify web` warns about a signature only when cosign rejects it (a failed download is "couldn't check"), warns when the release's checksums disown the manifest, says a page with no build number isn't a release build, and asks for an http(s) address; `hub app add` says when the local API is off by setting, or names the real port.
- CLI: `verify web` says in your language whether the page, GitHub, or GitHub's rate limit stopped the check; with remote access off, status and pair give the command that turns it back on (`OMNILLM_RELAY=wss://relay.ixxzh.com omnillm install`, since a reinstall keeps "off"); Auto is marked chat-only for devices when everything in it is, so a script defaults to a model that answers it.
- Install and update: the pre-release lookup uses the version the hub found, then GitHub's API, then the release feed when the API limits this address, so unattended updates no longer fail there.
- CLI: Auto is no longer the default for a script or pipe when every model it could pick is chat-only, and the local API doesn't offer programs Auto alone; `connect chat` and `connect ask` say which model Auto picked; `omnillm connect -h` and `omnillm hub -h` show the help; `hub models` has an In Auto column and `hub model set` takes `--auto true|false`; with remote access off, `hub pair` says so before handing an AI assistant a command; `hub region confirm` says when there is nothing to confirm.

### Security

- Google AI subscription: of agy's own settings only `trustedWorkspaces` and `showTips` are kept; settings and hooks are written to a temporary file and renamed (agy never reads a half-written fence), and left alone when unchanged; deleting a conversation also deletes its title, lock and summary row, and leftovers of conversations deleted by earlier versions are swept.

## [0.2.0-alpha.64](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.64) - 2026-10-26

### Changed

- Phone page: a new welcome and pairing page, with a language switch, dark mode and one action per device (a computer pastes the link, which starts pairing; a phone taps to scan). iPhone and iPad go to the Home Screen first, and the Home Screen app pairs with the same code once; pages inside WeChat and other apps open in the browser first; a browser that can't keep the pairing says so before pairing, not after; one line shows the pairing (connecting, the six digits, connected, failed); a relay other than relay.ixxzh.com is shown as a warning.

### Fixed

- Phone page: each gate before pairing is passed on its own and stays passed across a language switch, so a private tab on iPhone still hears it can't keep the pairing, and a page in another app isn't told twice. A pairing link arriving while the saved pairing loads waits for it; a paired device opening a broken link stays in its chats; a dropped pairing on a computer says to choose "New link" and paste it. In a browser that can't keep the pairing, a second tab no longer leaves the first one stuck. (Found by the client review, rounds 12–18.)

## [0.2.0-alpha.63](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.63) - 2026-10-26

### Fixed

- *New chat* (and deleting the open chat) moves the model picker to Auto at once; before, the picker kept the last chat's model until the list reloaded, so a message sent right away went to that model.
- When Auto leaves the list (only one model left for a device), its chats switch without saying the previous model was removed.
- Auto signs a reply with the name you gave the model, if you renamed it.
- Two chats that begin with the same words (“hi”) no longer share Auto's memory of which model a chat uses: the start of the first reply tells them apart.

## [0.2.0-alpha.62](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.62) - 2026-10-26

### Fixed

- Auto no longer shows “Switched to Auto” on every new message: a reply counts as a switch only once your Mac has said which model it chose, and only when that model differs from the last one.

### Changed

- The thinking line says which model is thinking (“Claude Opus 5.5 is thinking”; “Auto is picking a model” for the moment before Auto has chosen). Auto names its choice as soon as it makes it, not with the first words, and names the new one if it has to pass a model over.

## [0.2.0-alpha.61](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.61) - 2026-10-26

### Added

- *Auto (picks the model)*, at the top of the model list, and new chats start on it. Your Mac picks the model for each message by rules, with no extra call and no extra wait: a fast model for short small talk, the Sonnet / GPT-6-Sol tier for most questions, the strongest for code, error logs and harder requests, and only image-reading models for a message with an image. Subscriptions come before API keys, and when several can answer, the one used least lately goes first. A chat stays with its model (so the subscription's session goes on) and moves up only when it gets harder; each reply says which model wrote it ("Auto · Claude Sonnet 5.5"), and a move is shown as a line. Signed-out, at-limit or switched-off models are never chosen, and a model that fails before it starts answering is passed over for the next one. *In Auto* under Models in the console keeps a model out of it. Picking a model yourself works as before.

## [0.2.0-alpha.60](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.60) - 2026-10-26

### Fixed

- Phone page: a hub that gives this device no models now says so, instead of an error shown as "can't connect".
- Phone page: "can't reach your Mac" reads the same everywhere; a signed-out subscription is named and fixed the same way in the note and the error; a usage limit that resets on another day shows the day; stopping before the first word no longer shows a red error; loading failures say one sentence.
- Console: the wizard's last step can always be finished; removing a model source says its models disappear everywhere and a saved API key is deleted; a connected subscription that is signed out shows Sign in on its card; a paused region or a mismatched phone page is spelled out on a narrow window too.
- Command line: `omnillm uninstall -h` and `install -h` show help instead of acting, and a mistyped option stops; hub commands say "OmniLLM isn't running" in your language when it isn't; a mistyped command shows its reason and where the full list is; `connect ask` warns when `--model` comes after the question; Ctrl-C before the first word just stops.
- Command line: `hub provider add` checks the kind (subscriptions go through `omnillm login`) and that OmniLLM runs before asking for the key; `hub app add` shows the address with the token; `hub models` marks switched-off sources; `status --json` exits 1 when OmniLLM isn't running; uninstall on a device points to `connect forget`.

- Console: a signed-out subscription's card offers Sign in without claiming it is connected; "Connect a model" from the dashboard returns to the dashboard and doesn't offer an Ollama that's already connected; switching language in the wizard keeps what you typed; a first load during an update retries; the menu bar no longer says "Online" while models are paused for a region change.
- Phone page: a reload (language, new version) returns to the chat you had open; it doesn't reload while you're typing or with images attached; pairing with an unreachable computer says to scan a new code; Android gets its own Add-to-Home-Screen steps.
- Command line: subscriptions can be named `claude` and `chatgpt`, and `hub providers` shows IDs; removing a source with a key says the key is deleted; `openai-compatible` asks for its name and address before the key; hub commands on a device or without OmniLLM installed say so; `/image` understands `~/` and dragged-in paths; a wrong `--model` is caught at once; `status` checks its options and reports a device's pairing.

- Command line: when the hub refuses a command (a wrong name, a key that doesn't work, an expired pairing), its own sentence is shown, not "something went wrong". `hub audit` stays JSON for scripts. A device that isn't paired says so from `status` (exit 1), hub commands and `connect forget`.
- Console: a signed-out subscription has a Sign in button right in the Models tab; messages from the console and the local API name "your Mac" only on a Mac.

- Console: after you sign in again, a signed-out subscription recovers within about half a minute instead of up to 5 minutes; the Sign in button works on hubs that aren't a Mac too.
- `omnillm install` and `uninstall` leave launchd/systemd alone when HOME isn't your own home folder (a test or sandbox), instead of acting on your real service.

- Phone page: a load that fails says why (the browser blocks site data, or there's no network) instead of "the program stopped".
- Command line: a device with no usable models says so and how to check its model list, rather than claiming the Mac has none; `hub device set` warns when the device is left with no usable model.

- `omnillm hub run` while the background service runs says OmniLLM is already running (and where its log is) instead of starting a second hub that fights the first.
- Phone page: a paired phone that opens a new pairing link and goes back returns to a working chat.
- Command line: a device removed on the hub is told to run `connect forget` then `connect pair`; `connect` commands on the hub itself say it is the hub.

- Without a service manager (WSL, containers), installing or updating stops the running hub before starting the new one, so the update really takes effect.
- `status` says when every model source is switched off; pairing a device that can't use any model doesn't say "just chat on it"; `hub provider add ollama` says when Ollama isn't running.
- Phone page: a question you send is saved at once, so it isn't lost if the phone closes the page in the background; a browser that can't keep the pairing no longer reloads itself (which lost the pairing).

- Console: opening another console tab (the menu bar opens one each time) no longer signs out the tab that's already open.
- Phone page: what you're typing is kept if the phone closes the page in the background; a page that couldn't load offline loads again by itself once online.
- Command line: an update replaces even a hub started by an older version without a service manager, and says so if it couldn't; `hub provider add ollama` accepts the address without `/v1`; a model switched off mid-chat is replaced by one that works; local openai-compatible servers are refused with the real reason before the key is asked.

- Command line: `hub provider add ollama --base-url http://localhost:11434` works (it tried only the first of localhost's addresses); a local openai-compatible server over http is refused before the key; an Ollama address that can't be used says why; `omnillm help` on a device lists the device's commands.
- Phone page: a new version no longer reloads the page in the middle of pairing; a computer that stops answering reads as "can't be reached", not "something went wrong".

- Command line: `omnillm help` lists `omnillm login gemini`.
- Command line: a source's name and address are checked before the key is asked for; an address where nothing answers says to check it (not "try again later"); pairing with remote access switched off says so; a computer paired with `connect pair` alone counts as a device.
- Phone page: with two tabs open, a reload no longer leaves the tab you use unable to save; switching back to a tab keeps what you typed there.
- Console: *Check again* with an Ollama that has no models at all says what to pull, and the note stays long enough to read.

### Security

- Model replies shown in a terminal have control characters removed, so a reply can't use escape sequences (to write the clipboard or fake a prompt).
- `uninstall --purge` on a device deletes its pairing; a control socket that belongs to another user gets its own warning.
- A device's message that starts with "/" after any kind of leading space (Unicode spaces, zero-width characters) is still only a message.

## [0.2.0-alpha.59](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.59) - 2026-10-26

### Added

- Google AI subscriptions (Gemini) through Antigravity CLI, Google's official program: run `omnillm login gemini`, or use the new *I have a Google AI subscription (Gemini)* card in the console. It installs Antigravity CLI with Homebrew if needed and signs it in once more, for OmniLLM only (the agy you use yourself isn't affected). Every model Antigravity offers your account is listed, in its order, under subscriptions. As with Claude and ChatGPT, it is for chatting only: all of Antigravity's tools are refused before they run, and your own rule files aren't read. It takes text only. Google has banned accounts that used third-party tools with an Antigravity sign-in; OmniLLM runs the official program itself, but Google hasn't said whether that is allowed, so a Gemini API key is the safer choice.

## [0.2.0-alpha.58](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.58) - 2026-10-26

### Fixed

- A model server on this Mac other than the Ollama app (for example llama.cpp’s llama-server) that was added under its own name showed as “Ollama (this Mac)”. It now shows as “Model on this Mac (its name)” in the console and on the command line. *Check again* no longer mentions Ollama when a model on this Mac is already connected.

## [0.2.0-alpha.57](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.57) - 2026-10-26

### Fixed

- Console: *Check again* showed a red error when Ollama had only a non-chat model (such as an embedding model), though nothing had failed. It is now a plain note: Ollama on this Mac has no chat model yet; to use it, run `ollama pull qwen3` in Terminal.

## [0.2.0-alpha.56](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.56) - 2026-10-25

### Fixed

- Web chat: the chat search box showed a second, inner frame around the text field when focused; only the outer ring is shown now.

## [0.2.0-alpha.55](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.55) - 2026-10-25

### Changed

- Web chat: the chat’s name now sits at the top left in its own title bar, with a small arrow. Click it for *Rename* and *Delete* (the same menu as *…* in the chat list); double-click it to rename right away. A thin line appears under the bar only once the chat scrolls beneath it. With the sidebar hidden, the name moves right to clear the two buttons. On phones, tapping the name in the top bar opens the same menu.

## [0.2.0-alpha.54](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.54) - 2026-10-25

### Added

- Chats can be renamed in the web chat. The chat’s name is shown at the top of the chat (in the bar on phones); click it to rename. In the chat list, the *…* button next to each chat has *Rename* and *Delete*, and on a computer a double-click renames it too. A name you give is kept; only an unnamed chat is named after its first message.

## [0.2.0-alpha.53](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.53) - 2026-10-25

### Added

- Every model Codex offers your ChatGPT subscription is now listed, not just its default: for example GPT-6-Astra, GPT-6-Sol, GPT-6-Luna and GPT-5.6-Sol, in Codex’s own order. Each runs as itself (named with `-m`). The list follows Codex’s catalog on your Mac: models it adds appear after the hub restarts, and models it drops go away. The existing “ChatGPT (Codex · …)” entry stays Codex’s first model, so devices and chats that use it are unchanged; devices set to “All models” get the new ones at once.

## [0.2.0-alpha.52](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.52) - 2026-10-25

### Fixed

- Codex’s model name now really appears after the hub starts: the background version lookup ran for Claude only, so alpha.50 and alpha.51 showed the name only after a first Codex chat.

## [0.2.0-alpha.51](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.51) - 2026-10-25

### Fixed

- Codex’s model name now appears right after the hub starts, even if its version was looked up earlier in the week: reading Codex’s catalog costs nothing, so it isn’t limited to once a week like the Claude probe.

## [0.2.0-alpha.50](https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.50) - 2026-10-25

### Changed

- ChatGPT (Codex) shows the model it actually runs, for example “ChatGPT (Codex · GPT-6-Astra)”, instead of “(latest)”. The hub reads Codex’s own model catalog on your Mac (no model is called and no usage is counted), takes the model Codex lists first, and runs Codex with that model explicitly, so the name shown is the one that answers. If the catalog can’t be read, it falls back to Codex’s own default and shows “latest” as before.

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
- `omnillm hub region confirm` is the person's step too (not run for an AI assistant). `omnillm open` on a machine without a screen lists what to run there instead of a link no browser can open.
- The program's own messages say "your computer" instead of "your Mac" when the hub isn't a Mac, in the terminal too (hub and device). A wrong API key at `omnillm hub provider add` says to check the key. Pairing codes say how many minutes they last instead of a clock time (a server's clock is often in UTC).
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
