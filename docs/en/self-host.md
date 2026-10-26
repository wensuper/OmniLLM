# Run your own connection service

English | [简体中文](../zh-CN/self-host.md)

**Most people don’t need this.** The public connection service `relay.ixxzh.com` only forwards end-to-end encrypted data; it can’t see the content or impersonate your hub (see [Security](security.md)). Running your own has just two benefits:

- The metadata the connection service can see (IP addresses, connection times, traffic volume, and which devices are online) stays only on your own server.
- You don’t depend on the public service being up.

You need a server with a public IP address and a domain name. The connection service is lightweight: it writes nothing to disk, records no content, and is a single program using a few dozen MB of memory.

## 1. Deploy the connection service

On the server, download the program (from the same release as the hub, the Linux build), then run:

```bash
omnillm relay --listen :443 --autocert relay.example.com --open
```

- `--autocert` gets and renews a certificate from Let’s Encrypt automatically. The domain must already resolve to this server, and ports 80 and 443 must be reachable from outside.
- `--open` lets any hub register, which suits personal use and small groups. The connection service limits, per IP address, how many hubs can register and how many password guesses can be made.
- If it’s only for your own hub, replace `--open` with `--hub-token-hash <hash>`; see step 3 below.
- If you already run Nginx or another reverse proxy, use `--listen 127.0.0.1:8470 --behind-tls-proxy` and have the proxy terminate TLS and forward WebSocket. Example configuration: [deploy/zz-relay.ixxzh.com.conf](../../deploy/zz-relay.ixxzh.com.conf). **Don’t** let the proxy log the `Sec-WebSocket-Protocol` request header: it carries the mailbox password of the web app.
- To run it long term, use systemd; see [deploy/omnillm-relay.service](../../deploy/omnillm-relay.service). It runs under a separate account with no login and a read-only file system.

Check it:

```bash
curl -fsS https://relay.example.com/healthz     # prints ok
```

## 2. Point your hub at it

On the hub, run the installer once more with an environment variable (your settings, keys, and paired devices are kept):

```bash
OMNILLM_RELAY=wss://relay.example.com ~/.local/bin/omnillm install
```

**Devices that are already paired must pair again once**: a device remembers the connection service it paired through. In the console, choose *Add device*, scan the new QR code, and then remove the old device entry.

The web app still uses `omni.ixxzh.com`. It connects to the connection service named in the QR code, which is now yours. Encryption and storage in the web page all happen on the phone.

## 3. (Optional) Allow only your own hub to register

The first time the hub connects to the connection service, it generates a registration token. The token is stored in the keychain, and its hash is written to the hub log:

```bash
grep -i "token hash\|令牌的哈希" ~/Library/Application\ Support/OmniLLM/hub/hub.log | tail -1   # Linux: ~/.config/OmniLLM/hub/hub.log
```

On the connection service, replace `--open` with `--hub-token-hash <this hash>` (you can pass it more than once), and restart the connection service. The hash can be made public; keep the token itself private.

## 4. (Optional) Host the web app yourself

If you don’t want the web app to depend on someone else’s hosting either (see the “first load” issue in [Security](security.md#the-web-app)):

1. Download `omnillm-web.tar.gz` from the same release and check its SHA-256 against `checksums.txt` (`checksums.txt` is signed with cosign, so you can verify it with `cosign verify-blob`). Unpack it on an HTTPS domain used for nothing else, such as `https://omni.example.com`. **Don’t** share the domain with other sites: pages on the same origin can read its storage.
2. Have the web server set these response headers (see [deploy/zz-omni.ixxzh.com.conf](../../deploy/zz-omni.ixxzh.com.conf)): the type `application/wasm` for `.wasm` files; a strict `Content-Security-Policy` (including `frame-ancestors 'none'`); `X-Content-Type-Options: nosniff`; and `Referrer-Policy: no-referrer`. Turn on gzip as well if you can: the wasm file gets much smaller when compressed, and the first load is more than twice as fast.
3. Allow this web origin on the connection service: add `--web-origin https://omni.example.com`.
4. Point the hub at this web app:

   ```bash
   OMNILLM_RELAY=wss://relay.example.com OMNILLM_WEB_APP=https://omni.example.com/ ~/.local/bin/omnillm install
   ```

When you upgrade, update the web app first, then the hub: QR codes from a newer hub may use a format the older web app doesn’t recognize.

## Using it on one computer only

If you don’t need other devices, you can do without the connection service entirely:

```bash
OMNILLM_RELAY=off ~/.local/bin/omnillm install
```

The hub then provides the console and the local API on this computer only.

For how the project’s own public services are deployed, see [relay-deploy.md](../relay-deploy.md) (in Chinese).
