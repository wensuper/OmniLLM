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

## 以后每次发版

只做第 3 步（`package.sh` → `scp` → `install-server.sh`）和第 5 步的核对。旧版本留在 `releases/` 下：回滚就是把 `/opt/omnillm/omnillm` 和 `/home/yunwei/www/omni/current` 两个链接指回上一个标签，再 `sudo systemctl restart omnillm-relay`。

## 资源和安全

- relay 本身很轻：每个连接一个 WebSocket，内存里只有信箱表，不落盘、不记内容。开放模式（任何主机用 32 字节以上的令牌登记），按 IP 限主机数和猜口令次数，全局最多 10000 台主机。
- 机器上不保存任何人的密钥或对话。网页客户端只是静态文件；它一旦被换掉，就能拿到之后打开它的设备的密钥，所以只从签名发布安装，服务器上不手改。
