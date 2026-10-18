# OmniLLM 概要设计

> 状态：v0.1（2026-10-18），已评审通过
> 一句话：**一处接入大模型，所有设备都能用。** OmniLLM 是一个个人工具，不是平台。

## 1. 定位与范围

**是什么**：一个装在“中心节点”（Mac、PC 或服务器）上的单文件程序。中心节点保管模型凭据，其他设备（手机、平板、别的电脑或服务器）配对后，通过它来使用模型。

**解决的问题**：

- 每台设备都要配一遍 API Key，Key 散落各处，丢一台设备就得全部换掉。
- 本地模型（Ollama 等）只能在本机用，手机、平板用不上。
- 现有方案（LiteLLM Proxy、One API、Open WebUI 等）偏多用户平台，部署重，默认安全设置也不够收紧。

**设计原则**：

1. **凭据不出中心节点**：其他设备只拿到可单独吊销的设备令牌，永远拿不到 API Key。
2. **默认安全**：不配置就只监听本机；对外监听必须开 TLS，否则拒绝启动。
3. **够用就好**：一个用户、多台设备；单文件程序，不依赖外部数据库。
4. **兼容现有生态**：对外提供 OpenAI 兼容接口，现有 App 和 SDK 改一下 `base_url` 和 Key 就能用。

**不做（v1）**：多用户或团队计费；在设备之间共享网页版订阅的登录态（违反各家服务条款，见 §5.8）；自建公网中继。

## 2. 总体架构

```
 ┌─────────────────────── 中心节点 omnillm hub ───────────────────────┐
 │                                                                     │
 │  接入层     HTTPS (TLS 1.3) · 主机名/来源校验 · 限流                    │
 │     │                                                               │
 │  鉴权层     设备令牌校验 → 设备权限（可用模型、额度）                     │
 │     │                                                               │
 │  路由层     统一请求格式 → 选择 Provider → 流式转发 (SSE)                │
 │     │                                                               │
 │  Provider   OpenAI · Anthropic · Gemini · OpenAI 兼容 · Ollama/llama.cpp │
 │     │                                                               │
 │  凭据库     系统钥匙串 / 加密文件  ← 只有 Provider 层能读                 │
 │                                                                     │
 │  管理面     仅限本机 127.0.0.1 + Passkey：配对、吊销、额度、审计           │
 │  存储       SQLite：设备、额度、审计日志（默认不存对话内容）               │
 └─────────────────────────────────────────────────────────────────────┘
          ▲ 局域网 / Tailscale / 公网 HTTPS
          │
   ┌──────┴──────┬──────────────┬─────────────────┐
  手机/平板       其他电脑          服务器/脚本
  (内置 PWA)     (CLI / 任意       (OpenAI SDK)
                 OpenAI 兼容客户端)
```

### 2.1 组件

| 组件 | 说明 |
|---|---|
| `omnillm hub` | 中心节点守护进程，以上各层都在里面 |
| 内置 Web 应用 (PWA) | 编进程序里的聊天页面；手机、平板用浏览器打开后可“添加到主屏幕” |
| `omnillm` CLI | 管理中心节点（`pair`、`devices`、`revoke`、`status`），也可以作为客户端在终端里对话 |
| 第三方客户端 | 任何支持自定义 OpenAI `base_url` 的 App 或 SDK |

### 2.2 两种部署方式

| | 个人电脑做中心 | 服务器做中心 |
|---|---|---|
| 场景 | Mac 上已登录或配置好模型，手机出门也想用 | 一台常开的 VPS 或家里的 NAS |
| 远程访问 | **推荐 Tailscale**（内嵌 tsnet，零端口暴露） | 公网 HTTPS（自动申请 Let's Encrypt 证书）或 Tailscale |
| 局域网 | 自签证书 + 二维码里带证书指纹（证书固定） | 同左 |
| 凭据存放 | 系统钥匙串 | 加密文件，主密钥来自环境变量、systemd credential 或 KMS |

### 2.3 一次请求的流程

1. 客户端带着 `Authorization: Bearer omni_dev_…` 调用 `POST /v1/chat/completions`。
2. 接入层校验 TLS、Host 头和限流。
3. 鉴权层先对令牌做哈希，再到 SQLite 里查设备，并检查是否被吊销、能否用这个模型、额度还剩多少。
4. 路由层把请求转成目标 Provider 的格式，由 Provider 层从凭据库取 Key 并发出请求。
5. 结果以 SSE 流式返回给客户端，同时累计 token 用量、写一条审计记录（时间、设备、模型、token 数，**不含内容**）。

## 3. 配对与设备管理

配对是这个工具的安全核心。

1. 主人在中心节点运行 `omnillm pair --name "iPhone"`，可以加 `--models gpt-4o,claude-sonnet --budget 2usd/day`。
2. 中心节点生成**一次性配对码**：256 位随机数，**5 分钟过期，只能用一次**。它和地址、TLS 证书指纹一起编成二维码，在终端里显示。
3. 设备扫码打开 PWA（或在 CLI、App 里粘贴），PWA 校验证书指纹后，用配对码换取**设备令牌**。
4. 中心节点**只存令牌的哈希**（SHA-256，令牌本身是 256 位随机数，不需要慢哈希），明文令牌只在这一次返回给设备。
5. 设备把令牌存进本地安全存储：PWA 存在 IndexedDB 里，用不可导出的 WebCrypto 密钥包裹；CLI 存进系统钥匙串。

**设备权限**：每台设备单独配置可用模型、每日或每月额度（token 数或金额）和到期时间，随时可以 `omnillm revoke <设备>`，立即生效。

**令牌格式**：`omni_dev_<随机串>_<校验位>`，带固定前缀，方便 GitHub secret scanning 等工具识别泄露。

## 4. 核心技术

| 领域 | 选型 | 理由 |
|---|---|---|
| 语言 | **Go 1.23+** | 单文件、跨平台（macOS、Linux、Windows，amd64 和 arm64）、内存安全、标准库自带高质量的 HTTP 和 TLS |
| HTTP / 流式 | `net/http` + SSE | 不引入 Web 框架，依赖越少，攻击面越小 |
| TLS | `crypto/tls`（只用 TLS 1.3）；公网用 `x/crypto/acme/autocert`；局域网用自签 ECDSA P-256 证书 | 标准库实现，自动续期 |
| 远程组网 | **Tailscale `tsnet`**（可选，内嵌） | 中心节点直接加入你的 tailnet，不开端口、不暴露公网，NAT 穿透由 Tailscale 负责 |
| 存储 | SQLite（`modernc.org/sqlite`，纯 Go） | 单文件、免安装、免 CGO |
| 凭据存放 | `zalando/go-keyring`（macOS Keychain、Windows Credential Manager、Linux Secret Service）；无桌面环境时用 `filippo.io/age` 加密文件 | 用系统级保护，不自己造加密 |
| 管理员认证 | Passkey / WebAuthn（`go-webauthn/webauthn`） | 防钓鱼，没有密码可泄露 |
| 前端 PWA | TypeScript + Preact，构建产物用 `go:embed` 编进程序 | 体积小、不加载第三方 CDN，CSP 可以设得很严 |
| 二维码 | `skip2/go-qrcode`，终端显示 | 配对零输入 |
| Provider 协议 | OpenAI Chat Completions / Responses、Anthropic Messages、Gemini、Ollama | 自己写薄适配层，不依赖大而全的 SDK |
| 构建与发布 | GoReleaser + cosign 签名 + SBOM（syft）+ `govulncheck` | 发布物可校验，依赖漏洞进 CI |

## 5. 安全设计（重点）

### 5.1 威胁模型

| 资产 | 主要威胁 |
|---|---|
| 模型 API Key | 中心节点被入侵、磁盘或备份泄露、日志泄露、被客户端拿走 |
| 设备令牌 | 手机丢失、被抓包、被恶意网页窃取 |
| 中心节点本身 | 被暴露到公网后遭爆破或扫描、DNS 重绑定、SSRF、依赖投毒 |
| 对话内容 | 中间人攻击、日志或数据库里留存 |
| 额度和钱 | 令牌泄露后被盗刷 |

### 5.2 凭据保护

- API Key 只存在中心节点的系统钥匙串或 age 加密文件里，**不进 SQLite、不进配置文件、不进日志**。
- 只有 Provider 层能读凭据，读出来后不缓存到全局变量，用完即弃。
- 接口和管理页面**永远不回显** Key，只显示末 4 位。
- 日志中间件统一脱敏：`Authorization`、`x-api-key`、`omni_dev_*`、`sk-*` 一律替换。

### 5.3 网络暴露：默认最小

- 默认只监听 `127.0.0.1`。要监听局域网或公网，必须显式配置 `listen`，**而且必须启用 TLS，否则拒绝启动**。
- 管理接口和使用接口分开：管理面**只绑在 127.0.0.1**，远程管理必须走 Passkey，并且只能在 Tailscale 内使用。
- 校验 `Host` 头，只允许配置过的主机名，防 DNS 重绑定；PWA 接口再校验 `Origin`。
- 公网模式建议放在 Tailscale 后面。直接暴露时，启动时打印风险提示。

### 5.4 认证与访问控制

- 每台设备一个令牌，分别设权限、额度和到期时间，可以单独吊销。
- 令牌哈希比对用常数时间比较。
- 认证失败按 IP 和设备指数退避，连续失败锁定并写审计。
- 配对码一次性、5 分钟过期，配对接口单独限流。
- 服务器模式可选 mTLS，让设备证书和令牌双因子。

### 5.5 防盗刷

- 每台设备有额度上限（token 数或金额，按日和按月），超了直接拒绝。
- 全局有总额度上限，即使所有设备都没超，总量也封顶。
- 出现异常用量（比如一小时内超过日均的 5 倍）就自动暂停这台设备，并通知主人（系统通知或终端提示）。

### 5.6 内容与隐私

- **默认不保存对话内容**；审计只记元数据。
- 开启“保存历史”后，内容用中心节点主密钥加密后再写库。
- 不接入任何遥测或第三方统计。

### 5.7 Web 应用安全

- 严格 CSP：`default-src 'self'`，不允许内联脚本，不加载第三方资源。
- 令牌不放进 URL、不放进 localStorage 明文；Cookie 方案用 `HttpOnly; Secure; SameSite=Strict`。
- 所有写操作做 CSRF 防护；返回 `X-Content-Type-Options`、`Referrer-Policy: no-referrer`、`frame-ancestors 'none'`。

### 5.8 Provider 与合规

- Provider 的 `base_url` 只允许内置清单和用户显式添加的地址；禁止指向内网、元数据地址（如 `169.254.169.254`），防 SSRF。
- **只支持官方 API Key 和本地模型。** 不做“网页版订阅账号多端共享”，这违反 OpenAI、Anthropic 等的服务条款，也需要保存高权限的会话 Cookie，风险过高。
- 对外请求一律校验证书，不提供“跳过证书验证”的选项。

### 5.9 供应链与发布

- 依赖尽量只用标准库和 `golang.org/x`；新增依赖要过评审，CI 跑 `govulncheck` 和 `gosec`。
- 发布物用 cosign 签名并附 SBOM；安装脚本先校验签名再安装。
- 自动更新只接受签名有效的版本，不允许降级。
- 仓库开启 Dependabot、secret scanning 和分支保护。

### 5.10 安全默认值一览

| 项 | 默认值 |
|---|---|
| 监听地址 | `127.0.0.1` |
| 非本机监听不带 TLS | 拒绝启动 |
| 管理面 | 只在本机 |
| 配对码 | 5 分钟、一次性 |
| 设备令牌 | 90 天到期，可续 |
| 对话内容存储 | 关闭 |
| 单设备日额度 | 必须设置（无默认无限额） |

## 6. 对外接口（草案）

| 接口 | 说明 |
|---|---|
| `GET  /v1/models` | 当前设备可用的模型 |
| `POST /v1/chat/completions` | OpenAI 兼容，支持 `stream` |
| `POST /v1/messages` | Anthropic 兼容（透传格式） |
| `POST /pair/claim` | 用配对码换设备令牌 |
| `GET  /` | 内置 PWA |
| 管理面（仅限本机）`/admin/*` | 设备、额度、Provider、审计 |

模型名用 `provider/model` 形式（如 `anthropic/claude-sonnet-5-5`、`ollama/qwen3`），也支持别名。

## 7. 里程碑

| 版本 | 内容 |
|---|---|
| v0.1 | Hub + OpenAI 兼容接口 + OpenAI / Anthropic / Ollama 三个 Provider + 配对 + CLI + 审计 + 额度 |
| v0.2 | 内置 PWA（手机、平板可用）+ 局域网自签证书固定 |
| v0.3 | Tailscale tsnet 内嵌 + 公网 ACME 模式 + Passkey 管理 |
| v0.4 | 签名发布、自动更新、Homebrew / apt 安装 |
| 以后 | 原生 iOS / Android App；端到端加密中继（给不想用 Tailscale 的人） |

## 8. 已定决策（2026-10-18）

1. **许可证**：Apache-2.0（带专利授权，对企业用户更友好）。
2. **远程访问**：Tailscale 是推荐方案，不是必需；公网 ACME 模式保留。
3. **客户端优先级**：先做 PWA，覆盖所有平台；原生 App 等需求明确后再做。
