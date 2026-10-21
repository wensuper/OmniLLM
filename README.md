<p align="center"><img src="web/icon.svg" width="96" height="96" alt=""></p>

<h1 align="center">OmniLLM</h1>

<p align="center"><b>Your AI, Everywhere</b></p>

<p align="center">在一台电脑上登录大模型，手机、平板、服务器都能用。<br><i>Log in to your LLM once, on one computer. Use it from every device you own.</i></p>

- **账号只在你的电脑上**：Claude / ChatGPT 订阅、API Key、本地模型都留在这台电脑（主机）上，不发到别的设备。
- **端到端加密**：设备和主机之间的内容，中间的服务器只能转发、看不到。
- **手机不用装 App**：扫一下二维码就能对话，文字、图片都行。

> 早期预发布版（v0.2.0-alpha），主机支持 macOS 和 Linux。

## Quick Start

**① 安装**：在 Mac（或 Linux）的终端里运行，不需要 sudo：

```bash
curl -fsSL https://raw.githubusercontent.com/wensuper/OmniLLM/main/install.sh | sh
```

> 也可以把这一句发给你的 AI 编程助手（Claude Code、Codex、Cursor 等），让它来装：
> `请阅读并按照 https://raw.githubusercontent.com/wensuper/OmniLLM/main/INSTALL.md 在这台电脑上安装 OmniLLM。`

**② 接上模型**：这台电脑上已经登录了 Claude Code 或 Codex 的，会自动接上，跳过这步。没有的话，在自动打开的控制台里粘贴一个 API Key，或者装 [Ollama](https://ollama.com) 用免费的本地模型。

**③ 加手机**：在控制台的「设备」里点「添加设备」（Mac 上也可以点菜单栏的 OmniLLM 图标 →「添加设备…」），用手机相机扫二维码；两边显示的 6 位数一致，就在电脑上点「一致，添加」。

完成，大约 5 分钟。

## 更新

**不用管**：主机每隔几小时看一次新版本，在没人用的时候自己更新；手机上的网页也会自己换成新版。想马上更新：

```bash
omnillm update
```

Mac 上也可以点菜单栏的 OmniLLM 图标 →「检查更新…」。不想自动更新，运行 `omnillm update --auto off`。

> 装的是 0.2.0-alpha.21 或更早的版本？那时还不会自己更新：再运行一次上面的安装命令（设置都保留），之后就不用管了。

## 常用命令

刚装完的那个终端窗口还不认识 `omnillm`：新开一个终端窗口，或者在命令前加上 `~/.local/bin/`。

```bash
omnillm status      # 现在怎么样、下一步做什么
omnillm open        # 打开控制台
omnillm update      # 更新到最新版本
omnillm uninstall   # 停止并移除（加 --purge 连数据和 Key 一起删）
```

## 更多

| 文档 | 内容 |
|---|---|
| [快速上手（详细版）](docs/quickstart.md) | 每一步的细节、能用哪些模型、加 Linux 服务器、让脚本也能用 |
| [日常使用和常见问题](docs/usage.md) | 设备和额度、出错怎么办、更新和卸载 |
| [安全说明](docs/security.md) | 保护什么、不保护什么，和别的做法比 |
| [自己部署连接服务](docs/self-host.md) | 不想用公共连接服务时 |
| [INSTALL.md](INSTALL.md) | 写给 AI 助手的安装步骤 |
| [设计](docs/design.md) · [原则](docs/principles.md) · [更新记录](docs/changelog.md) | 开发者 |

## English

OmniLLM makes one computer (the **hub**) the only place that holds your LLM accounts — Claude or ChatGPT subscriptions via the official Claude Code / Codex, API keys, or local Ollama models — and lets your phone, tablet and servers use them over an end-to-end encrypted connection. Phones install nothing: scan a QR code and chat in the browser.

1. Install on your Mac or Linux box (no sudo):
   ```bash
   curl -fsSL https://raw.githubusercontent.com/wensuper/OmniLLM/main/install.sh | sh
   ```
2. Connect a model: a logged-in Claude Code or Codex connects by itself; otherwise paste an API key in the console that opens, or install Ollama.
3. Add your phone: *Devices → Add device* in the console (or the OmniLLM menu bar icon → *Add device…*), scan the QR code with the phone’s camera, check that the 6 digits match, then choose *Matches, add* on the computer.

Updates install by themselves while nobody is using the hub. Run `omnillm update` to update now, or `omnillm update --auto off` to turn automatic updates off. On 0.2.0-alpha.21 or earlier, run the install command once more (settings are kept); after that it updates itself.

Contributions are welcome: see [CONTRIBUTING.md](CONTRIBUTING.md). Report vulnerabilities privately per [SECURITY.md](SECURITY.md).

## License

[Apache-2.0](LICENSE)
