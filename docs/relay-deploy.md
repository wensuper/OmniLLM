# 公共连接服务和网页客户端上线（relay.ixxzh.com、omni.ixxzh.com）

zc 2026-10-19 决定：OmniLLM 的公共连接服务（relay）和手机网页客户端都放在 ixxzh.com 的服务器 139.224.110.2 上，按服务器上线流程（pgserver `docs/production.md`）操作，由「服务器上线操作方案」会话执行。

| 名字 | 是什么 | 机器上 |
|---|---|---|
| `wss://relay.ixxzh.com` | 连接服务：只转发端到端加密的数据，看不到内容 | `omnillm-relay.service`，只听 `127.0.0.1:8470`，Tengine `zz-relay.ixxzh.com.conf` 反代 |
| `https://omni.ixxzh.com` | 网页客户端：手机扫码打开，设备密钥在这里生成和保存 | 静态文件 `/home/yunwei/www/omni/current`，Tengine `zz-omni.ixxzh.com.conf` |

两个都只来自打了标签、CI 签过名的 GitHub 发布（`omnillm_linux_amd64.tar.gz`、`omnillm-web.tar.gz`，校验和 `checksums.txt` 由 cosign 无密钥签名）。

## 第一次上线

1. **DNS（zc 在阿里云 DNS 加）**：`relay.ixxzh.com` 和 `omni.ixxzh.com` 两条 A 记录指向 `139.224.110.2`。`dig +short relay.ixxzh.com` 有结果再往下做。
2. **证书**（acme.sh 在 root crontab，和官网一样走 80 端口的 webroot）：
   ```
   sudo cp deploy/zz-omni-acme.conf /usr/local/nginx/conf/vhost/
   sudo /usr/local/nginx/sbin/nginx -t && sudo /usr/local/nginx/sbin/nginx -s reload
   for d in relay.ixxzh.com omni.ixxzh.com; do
     sudo /root/.acme.sh/acme.sh --issue -d $d -w /home/yunwei/www/_acme
     sudo mkdir -p /etc/https/acme/$d
     sudo /root/.acme.sh/acme.sh --install-cert -d $d --key-file /etc/https/acme/$d/key.pem \
       --fullchain-file /etc/https/acme/$d/fullchain.pem --reloadcmd "/usr/local/nginx/sbin/nginx -s reload"
   done
   ```
   acme.sh 的实际路径以服务器上为准（官网证书用的同一个）。
3. **程序和网页**：在 Mac 上 `deploy/package.sh <标签>`（下载、核对签名和校验和），把输出的目录 `scp` 到服务器，在服务器上运行里面的 `install-server.sh`：装程序、装网页、装并启动 `omnillm-relay`，最后自检 `healthz`。
4. **Tengine**：备份后放入 `zz-relay.ixxzh.com.conf`、`zz-omni.ixxzh.com.conf`，删掉第 2 步的 `zz-omni-acme.conf`，`nginx -t` 通过再 `-s reload`。
   - 文件名以 `zz-` 开头（production.md §7：不能抢默认站点）。
   - 不要加 `http2`，不要改别的站点。
   - 这两个站点不要记录请求头（浏览器设备的信箱口令在 `Sec-WebSocket-Protocol` 里；默认日志格式不记）。
5. **外网核对**：
   ```
   curl -fsS https://relay.ixxzh.com/healthz          # ok
   curl -sI https://omni.ixxzh.com/ | grep -i content-security-policy
   curl -sI https://omni.ixxzh.com/omni.wasm | grep -i content-type   # application/wasm
   ```
   然后在 Mac 上装好的 OmniLLM 里加一台设备，用真手机扫码，发一条消息。

## 证书（2026-10-20 起）

两个名字都用 zc 在阿里云申请的 DigiCert 证书（比 Let's Encrypt 兼容更多老手机），放在 `/etc/https/aliyun/<名字>/`，和 Parley 一样**不会自动续期**，2027-01-16 到期，巡检在剩 14 天时告警；续期步骤同 production.md §12。上线时签的 Let's Encrypt 证书留在 `/etc/https/acme/<名字>/`，acme.sh 照常续期，阿里云证书来不及续时把 vhost 的两行路径指回去即可。

## 账号和权限（v0.2.0-alpha.3 起）

relay 以无登录的 `omnillm-relay` 账号运行（`install-server.sh` 第一次会建），systemd 加固（只读文件系统、看不到 /home、没有特权）；程序和网页文件归 root、对别人只读。这样 relay 即使被攻破，也改不了手机加载的网页。`package.sh` 没有 cosign 就停下；`install-server.sh` 在服务器上按签名过的校验和再核对一遍，网页里不许有符号链接，Tengine 配置里出现记录 `Sec-WebSocket-Protocol` 的日志格式就停下。

## 以后每次发版

只做第 3 步（`package.sh` → `scp` → `install-server.sh`）和第 5 步的核对。旧版本留在 `releases/` 下：回滚就是把 `/opt/omnillm/omnillm` 和 `/home/yunwei/www/omni/current` 两个链接指回上一个标签，再 `sudo systemctl restart omnillm-relay`。

上线后必须用真实浏览器验一次（原则 9），健康检查通过、主机连上都不算：
1. 程序、服务单元（`omnillm-relay.service`）、Tengine 站点配置**分开上**，一次只换一样。
2. 每换一样，先等主机重新连上（`omnillm status` 里 `relay_up` 为 true，Tengine reload 会断开所有长连接），再做浏览器测试：testkit 的 `conntime.mjs` 用无界面 Chromium 打开 `https://omni.ixxzh.com` 配对一台临时设备、刷新三次、测完移除，三次都要在几秒内连上你的 Mac。
3. 不通过就只退回刚换的那一样，再查原因。
4. 最后 `omnillm verify web`，确认线上网页就是签名发布构建的那一份。

## 事故记录

### 2026-10-24 alpha.48：浏览器设备全部连不上（重大）

**影响**：所有浏览器设备（手机网页、电脑网页）连不上主机，也配不上对：页面一直显示“正在连接你的 Mac……”或“Your Mac can’t be reached”，配对报“二维码已过期或已用过”。主机自己一直连着中转服务器，控制台和菜单栏都显示正常；命令行设备和主机都是 Go 程序，不受影响。没有任何数据泄露：故障发生在 TLS 握手，连接根本没建立起来。

**时间线（北京时间）**
- 10:54 alpha.48 上线：中转程序、服务单元（加 `RestrictAddressFamilies`）、站点配置（关日志，加 `ssl_session_tickets off`）三样一起换。只验了健康检查和主机重连。
- 约 11:15 zc 发现手机网页 30 秒以上连不上，来报。
- 11:17–11:24 排查：主机 `relay_up` 为 true、在线设备 0；用无界面浏览器配对一台临时设备，失败。
- 11:26 三样全部退回 alpha.47，浏览器恢复（配对 5.8 秒，刷新 2–3 秒连上）。**浏览器中断约 32 分钟。**
- 11:32 只换程序：正常。11:33 换站点配置：坏，11:36 退回。之后两次只改一行的测试都失败，误以为日志那几行也有问题（见下面“为什么绕了弯路”）。
- 15:15 换上整份 alpha.48 站点配置，在浏览器里抓到 `net::ERR_SSL_PROTOCOL_ERROR`，确认是 `ssl_session_tickets off`。15:17 去掉这一行，日志那几行保留：正常。
- 15:18 换 alpha.48 服务单元：正常。15:19 网页换 alpha.48。15:20 最后一次浏览器配对和 `verify web` 都通过，alpha.48 完整上线。
- 排查期间浏览器又短暂中断了三次（11:33–11:36、11:49–11:52、15:15–15:17），每次几分钟。

**原因**：139 上几个站点共用同一个 443 端口。别的站点都发 TLS 会话票据，只有 relay 站点关掉了。Chromium 连 relay 时的 TLS 握手因此失败；Go 的 TLS 客户端处理方式不同，照常连上。所以出现了“主机全正常、浏览器全不通”的局面。

**为什么排查绕了弯路**
- 每次 Tengine reload 都会断开所有经过它的长连接，主机要过几秒到一分钟才能连回来。有几次测试在主机还没连回来时就开始了，结果不能用，其中一次被误读成“日志那几行也有问题”。
- 这台 Mac 走 Cloudflare WARP，偶尔会解析失败或连接被重置，也会干扰测试结果。
- 真正起决定作用的证据，是在浏览器里直接抓到的 TLS 报错。以后先看浏览器自己的报错（WebSocket 的错误事件、控制台），再看配对结果。

**改了什么**
- 仓库里的站点配置去掉了 `ssl_session_tickets off`，并在旁边写明原因（a096678）。
- 原则 9：中转相关改动必须用真实浏览器验，共用 443 的 TLS 设置不单独改一个站点，服务器改动一次只上一样。
- 上面“以后每次发版”加上了浏览器验证的步骤。

**还没做的**：没有自动报警。这次是 zc 自己发现的。以后可以每隔几分钟用浏览器自动配对测一次，失败就报警（运维的 ops-digest 现在只能看日志，跑不了浏览器）。

## 资源和安全

- relay 本身很轻：每个连接一个 WebSocket，内存里只有信箱表，不落盘、不记内容。开放模式（任何主机用 32 字节以上的令牌登记），按 IP 限主机数和猜口令次数，全局最多 10000 台主机。
- 机器上不保存任何人的密钥或对话。网页客户端只是静态文件；它一旦被换掉，就能拿到之后打开它的设备的密钥，所以只从签名发布安装，服务器上不手改。
