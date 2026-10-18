# OmniLLM

**一处接入大模型，所有设备都能用。** Log in once, use your LLM everywhere.

OmniLLM 让你在一台设备上接入大模型（登录账号、配置 API Key 或部署本地模型），然后在其他设备上直接使用，不用每台设备各配一遍。

## 两种用法

1. **个人电脑做中心**：在 Mac / PC 上登录或配置好模型，手机、平板、其他电脑通过 OmniLLM 直接使用。
2. **服务器做中心**：把 OmniLLM 装在服务器上，PC、平板、手机通过中转服务器连到它使用。

```
   模型提供商 ◀── 中心节点（Mac / PC / 服务器，唯一和模型交互的地方）
                        │ 端到端加密
                   中转服务器（只转发密文）
                        │ 端到端加密
          手机 · 平板 · 其他电脑 · 服务器
```

## 目标

- **多端复用**：一份模型配置，PC、手机、平板、服务器都能用。
- **不封号**：只有中心节点和大模型交互，限流、退避都在这一处，看起来就是一个人在一台机器上用自己的账号。
- **服务器看不到内容**：中心节点和设备只经中转服务器通信，端到端非对称加密。
- **凭据不出门**：API Key 和登录会话只留在中心节点。
- **5 分钟用上**：不需要自己的服务器。装好中心节点，向导带你接入模型、扫码配对手机、发出第一条消息。

需要至少一个官方 API Key 或本地模型（如 Ollama）；只有 ChatGPT Plus、Claude Pro 这类网页订阅的话，云端模型用不了。

必须保证的规则见 [docs/principles.md](docs/principles.md)。

## 状态

概要设计见 [docs/design.md](docs/design.md)。v0.1 的库、连接服务、命令行、本机控制台和一步安装已实现并测试；实现状态见 §13。

## 安装：交给你的 AI

把这一句发给你正在用的 AI（Claude Code、Codex、Cursor 等），它会自己装好，全部用默认值：

> 请阅读并按照 https://raw.githubusercontent.com/wensuper/OmniLLM/main/INSTALL.md 在这台电脑上安装 OmniLLM。

装好后浏览器会打开本机控制台。唯一要你做的一件事是在控制台里粘贴 API Key（不要贴到和 AI 的聊天里）；如果这台电脑上已经有 Ollama 本地模型，连这一步都不用。

安装说明是写给 AI 看的，人也可以照着做：[INSTALL.md](INSTALL.md)。支持 macOS 和 Linux。

## License

[Apache-2.0](LICENSE)
