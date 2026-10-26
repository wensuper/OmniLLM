# 快速上手

[English](../en/quickstart.md) | 简体中文

目标：5 分钟内在手机上用到你 Mac 上的 Claude 或 ChatGPT，全程不花额外的钱、不用自己的服务器。

- [开始前：你需要什么](#开始前你需要什么)
- [第 1 步：在主机上安装](#第-1-步在主机上安装)
- [第 2 步：接上模型](#第-2-步接上模型)
- [第 3 步：加手机或平板](#第-3-步加手机或平板)
- [第 4 步：加 Linux 服务器或另一台电脑](#第-4-步加-linux-服务器或另一台电脑)
- [第 5 步（可选）：让程序也能用](#第-5-步可选让程序也能用)
- [检查一下](#检查一下)

## 开始前：你需要什么

| 需要 | 说明 |
|---|---|
| 一台做主机的电脑 | macOS 或 Linux（x86-64、ARM64）。设备只有在它开着、联网时才能用，所以最好是常开的那台。Windows 暂不支持做主机 |
| 一个模型来源，三选一 | **订阅**：主机上装好并登录 Claude Code（Claude Pro / Max）或 Codex（ChatGPT Plus / Pro）。**API Key**：Anthropic、OpenAI 或兼容 OpenAI 接口的服务。**本地模型**：主机上装 [Ollama](https://ollama.com)，免费，数据不出这台电脑 |
| 手机或平板 | 一个现代浏览器：iPhone / iPad 用 Safari，安卓用 Chrome。不用装 App |

**不需要**：服务器、域名、改路由器、注册任何新账号、sudo 权限。主机和设备都连项目提供的公共连接服务（`relay.ixxzh.com`），免费，它只转发加密数据。想用自己的，见[自己部署连接服务](self-host.md)。

## 第 1 步：在主机上安装

**方法一：交给你的 AI 助手（推荐）。** 把这一句发给 Claude Code、Codex、Cursor 等：

> 请阅读并按照 https://raw.githubusercontent.com/wensuper/OmniLLM/main/INSTALL.md 在这台电脑上安装 OmniLLM。

它会先读一遍安装脚本，再下载、核对、安装。这台电脑上已经登录的 Claude Code 或 Codex 会自动接上；OmniLLM 还用不上它的登录时（没登录、或用的是 API Key），登录窗口或控制台先带你接上一个模型。接着浏览器里显示添加手机的二维码：用手机相机扫一下，两边的 6 位数一样，就在电脑上点「一致，添加」。扫码和核对数字由你自己完成，AI 不会替你确认；它也不会、不应该让你把任何 Key 贴进聊天。

**方法二：自己运行。** 在终端里（普通用户，不要 sudo）：

```bash
curl -fsSL https://raw.githubusercontent.com/wensuper/OmniLLM/main/install.sh | sh
```

安装脚本会做这些事，不会改动别的东西：

1. 从本仓库的 GitHub 发布页下载对应系统的程序，核对 SHA-256 校验和；装了 [cosign](https://github.com/sigstore/cosign) 的话，还会核对校验和是本仓库的发布流程签的。任何一步对不上就停下。
2. 把程序放到 `~/.local/bin/omnillm`。
3. 注册一个只属于当前用户的后台服务（macOS 用 launchd，Linux 用 systemd 用户服务），开机登录后自动运行。
4. 打开本机控制台（只在这台电脑的浏览器里打开，地址是 `127.0.0.1`）：停在第一步“接入模型”；已经接上模型的话，直接是添加手机的二维码。

以后有新版本，主机会在没人用的时候自己更新，不用再运行这条命令（见[日常使用 · 更新](usage.md#更新和卸载)）。想先看一眼脚本再运行：把 `| sh` 换成 `-o omnillm-install.sh`，看完再 `sh omnillm-install.sh`。

控制台关掉了，随时用这条命令再打开：

```bash
~/.local/bin/omnillm open          # 加 --add 直接打开添加设备的二维码
```

## 第 2 步：接上模型

能用的模型来源：

| 来源 | 主机上需要 | 手机、平板、终端对话 | 脚本、后台程序 |
|---|---|---|---|
| Claude 订阅（Pro / Max） | 装好并登录 [Claude Code](https://claude.com/claude-code) | ✅ 文字 + 图片 | ❌ 订阅只给人用 |
| ChatGPT 订阅（Plus / Pro） | 装好并登录 [Codex](https://github.com/openai/codex) | ✅ 文字 + 图片 | ❌ 订阅只给人用 |
| Anthropic / OpenAI API Key | 在控制台粘贴 Key | ✅ | ✅ |
| 兼容 OpenAI 接口的服务 | 在控制台填地址和 Key | ✅ | ✅ |
| 本地模型（Ollama） | 装好 Ollama，自动接上 | ✅ | ✅ |

只有网页版订阅、又不想在主机上装 Claude Code 或 Codex 的话，订阅用不了：OmniLLM 不抓取网页的登录信息，那样违反服务条款，也最容易被封号。

### 用订阅（Claude 或 ChatGPT）

订阅通过官方的命令行程序用：Claude 订阅（Pro / Max）用 [Claude Code](https://claude.com/claude-code)，ChatGPT 订阅（Plus / Pro 等付费订阅）用 [Codex](https://github.com/openai/codex)。只装了 Claude 或 ChatGPT 桌面版也没关系，同一个账号再登录一次命令行版就行，OmniLLM 会带着你做：

- **控制台**：第一步里「我有 Claude 订阅」（或「我有 ChatGPT 订阅」）的卡片会说清现在差哪一步（“发现 Claude 桌面版”“已经装好，还没登录”），点「安装并登录」或「登录」（之前移除过的是「重新接入」）。会打开一个终端窗口，里面每条命令都写着：没装的先用官方安装程序装上（不用 sudo，约 1 分钟），然后浏览器打开，你用订阅账号确认。完成后卡片自己变成“已接入”，不用再点别的。
- **终端**（或没有浏览器的服务器）：

  ```bash
  omnillm login claude     # Claude Pro / Max
  omnillm login chatgpt    # ChatGPT Plus / Pro
  omnillm login gemini     # Gemini 订阅（用 Antigravity CLI）
  ```

已经登录过的，主机会自己发现并接上。Claude 的 Sonnet、Opus、Haiku 和 Codex 的默认模型会出现在模型列表里，不用填任何东西。

Gemini 订阅用 Google 官方的 Antigravity CLI（`agy`）。它要给 OmniLLM 单独登录一次，登录放在 OmniLLM 自己的文件夹里，你自己平时用的 agy 不受影响。注意：Google 封过用第三方工具调用 Antigravity 登录的账号，OmniLLM 运行的是官方程序本身，但 Google 没有明说这样是否允许；不想冒险就用 Gemini 的 API Key。

Codex 有时把登录存在系统钥匙串里，OmniLLM 用不了；控制台会提示“再登一次”，这次的登录会存成 OmniLLM 能用的文件（`~/.codex/auth.json`）。

没装好或没登录成功：终端窗口里会说下一步；控制台的卡片上有「重新打开」，再点一次就行。免费版 Claude 用不了 Claude Code，要 Pro 或 Max。

登录始终留在官方程序自己那里，OmniLLM 只问它“登录了吗”，不读取、不复制登录信息。主机调用它时关掉了它所有的工具（运行命令、读写文件、联网搜索、插件），只当纯文本模型用，所以设备上的人没法让它动你 Mac 上的文件。

### 用 API Key

在控制台的「模型」里选提供商，粘贴 Key，点「验证并接入」。主机会先发一个最小请求验证，通过了才保存。Key 存进系统钥匙串（Linux 上是只有你能读的加密文件）。

**只在控制台里粘贴 Key**，不要贴进和 AI 的聊天，也不要写在命令行上：聊天会发给 AI 服务商，命令行会留在 shell 历史里。

没有浏览器的 Linux 主机，在它的终端里运行下面这条，然后按提示粘贴 Key（Key 从输入读取，不出现在命令行上）：

```bash
~/.local/bin/omnillm hub provider add anthropic   # 或 openai
```

### 用本地模型

装好 [Ollama](https://ollama.com) 并下载一个模型（如 `ollama pull qwen3`），主机会自动发现并接上。没有自动接上的话，运行 `~/.local/bin/omnillm hub provider add ollama`。

## 第 3 步：加手机或平板

1. 在控制台里点「设备」→「添加设备」（Mac 上也可以点菜单栏图标 →「添加设备…」，或运行 `omnillm open --add`）。顶上的开关选在「手机或平板」：左边是二维码，右边是三个步骤，下面一行说现在在等什么。
2. 用手机**相机**扫码，打开链接。网页会问“添加这台手机？”，并说明经由哪个连接服务；点「添加」。iPhone、iPad 上也是直接添加。
3. 二维码会变成一个 6 位数，手机上也显示一个。**核对一致**，再在电脑上点「一致，添加」（不一样就点「不一致，不添加」）。
4. 手机上出现对话界面，发一句试试。

主机没有屏幕（服务器）时，改为在它的终端里运行 `omnillm pair`：二维码显示在那里，两边 6 位数一样就输入 y。它还会给一段配对文字，用来以 `omnillm connect pair` 添加电脑（第 4 步）。

添加手机是你自己的一步：通过 AI 助手安装时，它不会替你配对或确认。

几点说明：

- 二维码 5 分钟后过期（倒计时外面的圆环显示还剩多久），只能用一次。过期了，或者这次没添加上，直接点上面的「换一个二维码」。
- 手机扫不了码的话，在网页上点「粘贴链接或配对文字」，粘贴从控制台（「电脑或服务器」）复制的配对链接或配对文字，直接到核对 6 位数。另一台电脑上打开网页，一开始就是这个粘贴框。
- 想像 App 一样打开：安卓上点 Chrome 菜单里的「添加到主屏幕」（配对跟着走）。iPhone、iPad 上欢迎页下方有「想像 App 一样放到主屏幕？」，里面写着步骤；主屏幕上的 OmniLLM 要单独添加一次。在微信等 App 里打开时，可以「就在微信里用」，也可以复制网址到浏览器里打开（浏览器里能一直保存）。
- 每台设备默认能用所有已接入的模型，每天额度 5 美元（估算；订阅模型不计钱，按次数限）。在控制台的「设备」里可以改。
- 平板和手机步骤一样，每台设备各配对一次。

## 第 4 步：加 Linux 服务器或另一台电脑

有浏览器的电脑，什么都不用装：

1. 在主机控制台点「添加设备」，切到「电脑或服务器」，点「复制链接」（和二维码里是同一个链接）。
2. 在那台电脑的浏览器里打开这个链接（链接由你自己带过去，相当于钥匙，不要发给别人），点「添加」。
3. 和手机一样，核对两边的 6 位数，在主机上点「一致，添加」。

如果主机本身没有屏幕（服务器），不用它的控制台，改为在它的终端里运行 `omnillm pair`：它会给出配对文字，两边数字一样就在那里输入 y。

没有浏览器的服务器用命令行。服务器上只放一个程序，不装后台服务，不需要 sudo。

1. 在主机控制台点「添加设备」，切到「电脑或服务器」，展开最下面的「那台电脑没有浏览器？用命令行」。
2. 在那台机器上运行页面上给出的这一行（已经装过也运行这条）：

   ```bash
   curl -fsSL https://raw.githubusercontent.com/wensuper/OmniLLM/main/install.sh | sh -s -- --device
   ```

   装完它会提示你粘贴配对文字（通过 AI 助手或脚本安装时不会问，之后在你自己的终端里运行 `omnillm connect pair`，在那里粘贴）。
3. 从控制台复制配对文字（以 `omnillm3-` 开头；每一行都有自己的「复制」），粘贴过去，回车。粘贴时终端里看不到字，是正常的。配对文字由你自己搬运，比如通过 SSH 粘贴；不要发给别人，也不要贴进聊天。
4. 和手机一样，核对两边的 6 位数，在主机上点「一致，添加」。

服务器上没有后台服务，不会自己更新；有新版本时，`omnillm connect` 会提一句，运行 `omnillm update` 就行。

配好以后，在那台机器的终端里：

```bash
omnillm connect chat                                 # 多轮对话；Ctrl-C 只停这一次回答
omnillm connect ask "用一句话解释什么是端到端加密"      # 问一句就退出
omnillm connect ask --image 截图.png "这张图里有什么？" # 带图片
omnillm connect models                               # 看能用哪些模型
```

不指定模型时，默认用 Claude Sonnet（有 Claude 订阅时）。

安装程序会把 `~/.local/bin` 加到 PATH（写进 `~/.bashrc`、`~/.zshrc` 等），新开的终端里可以直接输入 `omnillm`；刚装完的那个终端里，用安装结束时打印的完整路径 `~/.local/bin/omnillm`。

## 第 5 步（可选）：让程序也能用

**主机这台 Mac 上的程序**：主机在 `http://127.0.0.1:8765/v1` 提供本机接口，默认就是开着的。在控制台的「应用」里点「添加应用」，起个名字，就会拿到一枚只给它用的令牌（只显示一次），旁边有可以直接复制的设置和一条试用命令。令牌就是 API Key。「应用」页顶上那句说明旁边的「?」点开是接入方式：接口地址（带「复制」）、Key 用应用的令牌、兼容 OpenAI 也支持 Anthropic、只有这台 Mac 上的程序连得上。每个应用有自己的每天额度和能用的模型，随时可以暂停或移除。每次请求都要带令牌，网页发来的请求一律拒绝。

页面上说本机接口没有打开的话，再运行一次安装命令。想换一个回环地址或关掉它，运行安装命令时设 `OMNILLM_API`（例如 `OMNILLM_API=127.0.0.1:9000` 或 `OMNILLM_API=off`），以后更新会一直保留。

大多数支持 OpenAI 接口的程序和 SDK 读这两个环境变量：

```bash
export OPENAI_BASE_URL=http://127.0.0.1:8765/v1
export OPENAI_API_KEY=<这个应用的令牌>
```

**其他设备上的程序**（脚本、编辑器插件、其他 AI 应用）可以通过设备上的一个兼容 OpenAI 的本机接口使用主机的模型：

```bash
omnillm connect run          # 在 http://127.0.0.1:8766/v1 提供接口，令牌文件的位置会打印出来
```

把程序的 API 地址填成 `http://127.0.0.1:8766/v1`，Key 填令牌文件里的内容。也支持 Anthropic 格式的 `/v1/messages`。

**程序只能用 API Key 和本地模型，看不到订阅模型。** 订阅只给人手动对话用：用订阅跑脚本、批量任务，是最容易被封号的用法。这条规则写在代码里，关不掉。

## 检查一下

在主机上运行：

```bash
~/.local/bin/omnillm status
```

它会说主机是否在运行、接了几个模型、配了几台设备，还有哪一步没做。显示“一切正常”（英文是 All set）就全部好了。

下一步：[日常使用和常见问题](usage.md) · [安全说明](security.md)
