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

概要设计见 [docs/design.md](docs/design.md)。v0.1 的库、连接服务和命令行已实现并测试，实现状态见 §13。

## 试用（v0.1，命令行）

```bash
go install github.com/wensuper/OmniLLM/cmd/omnillm@latest
omnillm relay token                     # 自己部署连接服务时用；公共连接服务上线后可省略
omnillm hub run --relay wss://<连接服务>  # 主机，前台运行
omnillm hub provider add anthropic      # 粘贴 API Key（不会显示）
omnillm hub pair --name "我的服务器"      # 显示配对文字，在主机上确认 6 位数字
omnillm connect pair                    # 在另一台设备上粘贴配对文字
omnillm connect ask --model anthropic/<模型> "你好"
```

现在只有命令行；网页控制台、手机网页客户端、一行安装还在做，见 [docs/design.md §13](docs/design.md)。

## License

[Apache-2.0](LICENSE)
