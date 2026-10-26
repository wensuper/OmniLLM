# 自己部署连接服务

[English](../en/self-host.md) | 简体中文

**大多数人不需要这一步。** 公共连接服务 `relay.ixxzh.com` 只转发端到端加密的数据，看不到内容，也冒充不了你的主机（见[安全说明](security.md)）。自己部署的好处只有两个：

- 连接服务能看到的元数据（IP、连接时间、流量大小、哪些设备在线）只留在你自己的服务器上；
- 不依赖公共服务是否在线。

自己部署需要一台有公网 IP 的服务器和一个域名。连接服务很轻：不落盘、不记内容，一个程序，几十 MB 内存。

## 1. 部署连接服务

在服务器上下载程序（和主机同一个发布包，Linux 版），然后运行：

```bash
omnillm relay --listen :443 --autocert relay.example.com --open
```

- `--autocert` 自动从 Let's Encrypt 申请和续期证书，需要域名已经解析到这台服务器，并且 80、443 端口能从外面访问。
- `--open` 允许任何主机登记，适合自己用和小范围用。连接服务按 IP 限制主机数量和猜口令的次数。
- 只给自己的主机用的话，把 `--open` 换成 `--hub-token-hash <哈希>`，见下面第 3 步。
- 已经有 Nginx 或其他反向代理的话，用 `--listen 127.0.0.1:8470 --behind-tls-proxy`，由代理终结 TLS，并转发 WebSocket。参考配置：[deploy/zz-relay.ixxzh.com.conf](../../deploy/zz-relay.ixxzh.com.conf)。**不要**让代理的日志记录 `Sec-WebSocket-Protocol` 请求头：手机网页的信箱口令在这个头里。
- 长期运行用 systemd，参考 [deploy/omnillm-relay.service](../../deploy/omnillm-relay.service)：用无登录的独立账号运行，文件系统只读。

检查：

```bash
curl -fsS https://relay.example.com/healthz     # 输出 ok
```

## 2. 让主机用它

在主机上用环境变量重新运行一次安装（设置、Key、已配对的设备都保留）：

```bash
OMNILLM_RELAY=wss://relay.example.com ~/.local/bin/omnillm install
```

**已经配对的设备要重新配对一次**：设备记住的是它配对时的连接服务。在控制台点「添加设备」，扫新的二维码，再把旧的设备记录移除。

网页版仍然用 `omni.ixxzh.com`，它会连二维码里写的连接服务，也就是你的。网页里的加密和存储都在手机上完成。

## 3.（可选）只允许自己的主机登记

主机第一次连连接服务时会生成一个登记令牌，令牌存在钥匙串里，它的哈希写在主机日志里：

```bash
grep -i "token hash\|令牌的哈希" ~/Library/Application\ Support/OmniLLM/hub/hub.log | tail -1   # Linux：~/.config/OmniLLM/hub/hub.log
```

把连接服务的 `--open` 换成 `--hub-token-hash <这个哈希>`（可以写多个），重启连接服务。哈希可以公开，令牌本身不要外传。

## 4.（可选）自己部署网页版

想连网页也不依赖别人托管（见[安全说明](security.md#网页版)的“第一次加载”问题）：

1. 从同一个发布里下载 `omnillm-web.tar.gz`，核对它在 `checksums.txt` 里的 SHA-256（`checksums.txt` 由 cosign 签名，可以用 `cosign verify-blob` 核对），解压到一个只给它用的 HTTPS 域名下，比如 `https://omni.example.com`。**不要**和别的网站共用域名：同源的网页能读它的存储。
2. 网页服务器要设置这些响应头，参考 [deploy/zz-omni.ixxzh.com.conf](../../deploy/zz-omni.ixxzh.com.conf)：`.wasm` 的类型是 `application/wasm`；严格的 `Content-Security-Policy`（含 `frame-ancestors 'none'`）；`X-Content-Type-Options: nosniff`；`Referrer-Policy: no-referrer`。最好再开 gzip，wasm 压缩后小很多，首次打开快一倍以上。
3. 连接服务允许这个网页来源：加上 `--web-origin https://omni.example.com`。
4. 主机用这个网页：

   ```bash
   OMNILLM_RELAY=wss://relay.example.com OMNILLM_WEB_APP=https://omni.example.com/ ~/.local/bin/omnillm install
   ```

升级时，先换网页，再升级主机：新主机生成的二维码可能用了旧网页不认识的格式。

## 只在一台电脑上用

不需要别的设备时，可以完全不用连接服务：

```bash
OMNILLM_RELAY=off ~/.local/bin/omnillm install
```

这时主机只在本机提供控制台和本机接口。

项目自己的公共服务是怎么部署的，见 [relay-deploy.md](../relay-deploy.md)。
