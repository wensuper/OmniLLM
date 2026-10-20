<p align="center"><img src="web/icon.svg" width="96" height="96" alt=""></p>

<h1 align="center">OmniLLM</h1>

<p align="center"><b>Your AI, Everywhere</b></p>

**在一台电脑上登录大模型，手机、平板、服务器都能用。**
*Log in to your LLM once, on one computer. Use it from every device you own.*

OmniLLM 把你的一台电脑（通常是常开的 Mac）变成**主机**：只有它保存账号、和大模型通信。你的手机、平板、别的电脑和服务器经过端到端加密的连接，用这台主机上的模型。手机和平板扫一下二维码就能用，什么都不用装。

```
  Claude / ChatGPT 订阅 · API Key · 本地模型
                  ▲
                  │  只有主机和模型通信，账号只在这里
        ┌─────────┴─────────┐
        │  主机（你的 Mac）  │
        └─────────┬─────────┘
                  │  端到端加密
     连接服务（只转发它解不开的密文）
                  │  端到端加密
   手机 · 平板（浏览器，免安装） · 电脑 · Linux 服务器
```

> 现在是早期预发布版（v0.2.0-alpha），主机支持 macOS 和 Linux。请先在自己的设备上试用。

## 为什么要有它

如果你已经付费订阅了 Claude 或 ChatGPT，或者有自己的 API Key、本地模型，想在每台设备上都用，常见的做法各有问题：

| 常见做法 | 问题 |
|---|---|
| 每台设备各登录一次 | 麻烦；服务器、公用电脑上不想留登录；同一个账号在多处、多个 IP 同时用，容易被判成账号共享而受限，甚至封号 |
| 把 API Key 复制到每台设备 | Key 散在各处，丢一台设备就得换 Key；脚本、配置文件、命令历史里都可能留下 Key |
| 自建一个公网 API 网关 | 要服务器、域名、证书；网关对公网开放，被扫到就会被盗刷；网关能看到你所有的对话 |
| 内网穿透或组网连回家里的电脑 | 要改路由器或装组网软件；每台设备都要装、要登录 |

OmniLLM 的做法：

- **账号只在主机上**：订阅的登录留在主机上的官方程序里，API Key 存在主机的系统钥匙串里（Linux 上是加密文件），都不会发到别的设备。设备丢了，在主机上移除它就行，不用换 Key、不用改密码。
- **在模型提供商看来，只有一个人在一台电脑上用**：所有请求都从主机这一个出口发出，统一限速、排队，遇到限流就退让；不轮换账号、不换 IP、不绕限制。
- **中间的服务器看不到内容**：主机和设备都主动连到一个连接服务，由它转发；内容端到端加密，连接服务只拿得到解不开的密文。主机不对外开任何端口，不用改路由器，也不用自己的服务器。
- **别的设备尽量不装东西**：手机、平板用浏览器；Linux 服务器只放一个程序，不装后台服务。要装的都装在主机上。

## 零投入，3 步上手

不买服务器、不注册新账号、不额外付费：用你已有的订阅（或免费的本地模型），加上项目提供的公共连接服务。

**① 在 Mac 上装主机。** 把这一句发给你正在用的 AI 编程助手（Claude Code、Codex、Cursor 等），它会自己装好：

> 请阅读并按照 https://raw.githubusercontent.com/wensuper/OmniLLM/main/INSTALL.md 在这台电脑上安装 OmniLLM。

也可以自己在终端里运行（脚本很短，建议先看一眼；不需要 sudo）：

```bash
curl -fsSL https://raw.githubusercontent.com/wensuper/OmniLLM/main/install.sh -o omnillm-install.sh && sh omnillm-install.sh
```

**② 接上模型。** 这台 Mac 上已经装好并登录了 Claude Code 或 Codex 的，会被自动接上，什么都不用做。没有的话，在自动打开的本机控制台里粘贴一个 API Key，或者装 [Ollama](https://ollama.com) 用免费的本地模型。

**③ 加手机。** 在控制台里点「添加设备」，用手机相机扫二维码，点「连接」。电脑和手机上会显示同一个 6 位数，一致就在电脑上点「确认」，然后就可以在手机上对话了，文字、图片都行。

整个过程约 5 分钟。Linux 服务器、另一台电脑怎么加，以及每一步的细节，见 **[快速上手](docs/quickstart.md)**。

## 安全，一页看完

| 我们保证 | 怎么做到 |
|---|---|
| 连接服务看不到内容 | 主机和设备之间用 Noise 协议端到端加密（X25519 + ChaCha20-Poly1305，和 WireGuard 同一类），连接服务只转发密文 |
| 连接服务冒充不了你的主机 | 主机的公钥只通过你面前的二维码（或你自己复制过去的配对文字）交给设备，从不经过服务器；设备每次连接都核对 |
| 偷拍到二维码也配不上 | 二维码 5 分钟过期、只能用一次；最后还要你在主机上核对 6 位数、点确认 |
| 账号不出主机 | 订阅登录留在官方程序里，OmniLLM 不读取、不复制；API Key 在钥匙串里，界面、日志、发往设备的数据里都不会出现 |
| 尽量不被封号 | 订阅只给人手动对话用，脚本和后台程序用不了，这一条写死在代码里，主人也关不掉；一次只跑一个请求、每分钟最多 12 次、每台设备每小时最多 60 次；被限流就按要求暂停 |
| 设备丢了损失有限 | 每台设备有自己的额度（默认每天 5 美元，估算）；在主机上点「移除」立即失效 |
| 主机不暴露 | 主机不监听任何对外端口；本机接口只开在 `127.0.0.1`，要令牌，拒绝来自网页的请求 |
| 安装包可以核对 | 发布包由 GitHub Actions 构建，校验和用 cosign 签名；安装脚本先核对再安装，对不上就停 |

**也如实说明它的限制**：

- 连接服务看不到内容，但能看到双方的 IP、连接时间、流量大小和哪些设备在线。
- 手机上的网页客户端第一次打开时，要信任托管网页的服务器（浏览器不会校验第一次加载的页面）。要求更高时，用命令行客户端，或自己部署网页。
- 主机本身被入侵，什么都保护不了：账号就保存在它上面。

完整说明见 **[安全说明](docs/security.md)**。发现漏洞请按 [SECURITY.md](SECURITY.md) 私下报告。

## 能用哪些模型

| 来源 | 主机上需要 | 手机、平板、终端对话 | 脚本、后台程序 |
|---|---|---|---|
| Claude 订阅（Pro / Max） | 装好并登录 [Claude Code](https://claude.com/claude-code) | ✅ 文字 + 图片 | ❌ 订阅只给人用 |
| ChatGPT 订阅（Plus / Pro） | 装好并登录 [Codex](https://github.com/openai/codex) | ✅ 文字 + 图片 | ❌ 订阅只给人用 |
| Anthropic / OpenAI API Key | 在控制台粘贴 Key | ✅ | ✅ |
| 兼容 OpenAI 接口的服务 | 在控制台填地址和 Key | ✅ | ✅ |
| 本地模型（Ollama） | 装好 Ollama，自动接上 | ✅ | ✅ |

只有网页版订阅、又不想在主机上装 Claude Code 或 Codex 的话，订阅用不了。OmniLLM 不抓取网页的登录信息：那样违反服务条款，也最容易被封号。

## 文档

| 文档 | 给谁看 |
|---|---|
| [快速上手](docs/quickstart.md) | 第一次用：装主机、接模型、加手机、加服务器 |
| [日常使用和常见问题](docs/usage.md) | 用起来以后：设备管理、图片、额度、断网、出错怎么办、卸载 |
| [安全说明](docs/security.md) | 想知道 OmniLLM 保护什么、不保护什么 |
| [自己部署连接服务](docs/self-host.md) | 不想用公共连接服务的人 |
| [INSTALL.md](INSTALL.md) | 写给 AI 助手的安装步骤 |
| [设计文档](docs/design.md) · [原则](docs/principles.md) · [更新记录](docs/changelog.md) | 开发者，以及想深入了解的人 |

## 参与

欢迎提 Issue 和 PR，见 [CONTRIBUTING.md](CONTRIBUTING.md)。任何改动都不能违背[原则](docs/principles.md)里的两条：只有主机和模型通信；主机和设备之间只经连接服务、端到端加密。

## English summary

OmniLLM turns one computer (the **hub**) into the only place that holds your LLM credentials and talks to providers: Claude or ChatGPT subscriptions through the official Claude Code / Codex CLIs, API keys, or local models (Ollama). Your phone, tablet, other computers and servers use those models over an end-to-end encrypted connection (Noise IK) through a relay that only forwards ciphertext. Phones and tablets install nothing: scan a QR code and chat in the browser. Subscriptions are usable only by a person typing, paced like a single user, to avoid account bans. Quick start: give your AI coding assistant the sentence above (see [INSTALL.md](INSTALL.md)), then choose *Add device* in the local console and scan. [docs/security.md](docs/security.md) covers the threat model and its honest limits.

## License

[Apache-2.0](LICENSE)
