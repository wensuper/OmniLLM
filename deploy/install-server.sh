#!/bin/sh
# Run on the server (139.224.110.2) as yunwei with sudo, from the folder
# deploy/package.sh made (copied over). It re-checks every file against the
# release checksums, then installs side by side and switches links, so the
# previous version stays for rollback:
#   /opt/omnillm/releases/TAG/omnillm   -> /opt/omnillm/omnillm
#   /home/yunwei/www/omni/releases/TAG  -> /home/yunwei/www/omni/current
# Rollback: point both links at the previous TAG and restart the service.
# Program and web files are owned by root and read-only to everyone else;
# the relay runs as its own account (omnillm-relay) that can write nothing.
set -eu
src=$(cd "$(dirname "$0")" && pwd)
tag=$(cat "$src/TAG")
case "$tag" in
  v[0-9]*) ;;
  *) echo "bad tag" >&2; exit 1 ;;
esac
case "$tag" in *[!0-9A-Za-z._-]*) echo "bad tag" >&2; exit 1 ;; esac

# 1. The archives must match the signed checksums package.sh verified, and
#    the unpacked files must match the archives.
cd "$src"
sha256sum -c want.txt
rm -rf "$src/check" && mkdir -p "$src/check/web"
tar -xzf omnillm_linux_amd64.tar.gz -C "$src/check" omnillm
tar -xzf omnillm-web.tar.gz -C "$src/check/web"
cmp -s "$src/check/omnillm" "$src/bin/omnillm" || { echo "bin/omnillm differs from the signed archive" >&2; exit 1; }
diff -r "$src/check/web" "$src/web" >/dev/null || { echo "web/ differs from the signed archive" >&2; exit 1; }
if [ -n "$(find "$src/web" -type l)" ]; then echo "web/ contains symbolic links" >&2; exit 1; fi
rm -rf "$src/check"

# 2. No log format may record the header that carries browsers' mailbox secret.
if sudo grep -rqi 'sec_websocket_protocol' /usr/local/nginx/conf/; then
  echo "a Tengine log format records Sec-WebSocket-Protocol (mailbox secrets); remove it first" >&2; exit 1
fi

# 3. The relay's own account.
id omnillm-relay >/dev/null 2>&1 || sudo useradd --system --no-create-home --home-dir /var/empty --shell /sbin/nologin omnillm-relay

# 4. Install side by side, root-owned, then switch the links.
sudo mkdir -p "/opt/omnillm/releases/$tag"
sudo install -o root -g root -m 755 "$src/bin/omnillm" "/opt/omnillm/releases/$tag/omnillm"
sudo ln -sfn "/opt/omnillm/releases/$tag/omnillm" /opt/omnillm/omnillm.new
sudo mv -T /opt/omnillm/omnillm.new /opt/omnillm/omnillm
sudo mkdir -p /home/yunwei/www/omni/releases
sudo rm -rf "/home/yunwei/www/omni/releases/$tag"
sudo cp -R "$src/web" "/home/yunwei/www/omni/releases/$tag"
sudo chown -R root:root "/home/yunwei/www/omni/releases/$tag"
sudo chmod -R u=rwX,go=rX "/home/yunwei/www/omni/releases/$tag"
sudo ln -sfn "/home/yunwei/www/omni/releases/$tag" /home/yunwei/www/omni/current.new
sudo mv -T /home/yunwei/www/omni/current.new /home/yunwei/www/omni/current

# 5. The service.
if ! cmp -s "$src/omnillm-relay.service" /etc/systemd/system/omnillm-relay.service; then
  sudo install -m 644 "$src/omnillm-relay.service" /etc/systemd/system/omnillm-relay.service
  sudo systemctl daemon-reload
  sudo systemctl enable omnillm-relay
fi
sudo systemctl restart omnillm-relay
sleep 2
systemctl is-active omnillm-relay
ps -o user= -p "$(systemctl show -p MainPID omnillm-relay | cut -d= -f2)"
curl -fsS http://127.0.0.1:8470/healthz
/opt/omnillm/omnillm version
