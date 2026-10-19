#!/bin/sh
# Run on the server (139.224.110.2) as yunwei with sudo, from the folder
# deploy/package.sh made (copied over). Installs side by side and switches
# links, so the previous version stays for rollback:
#   /opt/omnillm/releases/TAG/omnillm   -> /opt/omnillm/omnillm
#   /home/yunwei/www/omni/releases/TAG  -> /home/yunwei/www/omni/current
# Rollback: point both links at the previous TAG and restart the service.
set -eu
src=$(cd "$(dirname "$0")" && pwd)
tag=$(cat "$src/TAG")
case "$tag" in v[0-9]*) ;; *) echo "bad tag" >&2; exit 1 ;; esac
sudo mkdir -p "/opt/omnillm/releases/$tag"
sudo install -m 755 "$src/bin/omnillm" "/opt/omnillm/releases/$tag/omnillm"
sudo ln -sfn "/opt/omnillm/releases/$tag/omnillm" /opt/omnillm/omnillm.new
sudo mv -T /opt/omnillm/omnillm.new /opt/omnillm/omnillm
mkdir -p "/home/yunwei/www/omni/releases/$tag"
cp -R "$src/web/." "/home/yunwei/www/omni/releases/$tag/"
chmod -R a+rX "/home/yunwei/www/omni/releases/$tag"
ln -sfn "/home/yunwei/www/omni/releases/$tag" /home/yunwei/www/omni/current.new
mv -T /home/yunwei/www/omni/current.new /home/yunwei/www/omni/current
if [ -f "$src/omnillm-relay.service" ] && ! cmp -s "$src/omnillm-relay.service" /etc/systemd/system/omnillm-relay.service; then
  sudo install -m 644 "$src/omnillm-relay.service" /etc/systemd/system/omnillm-relay.service
  sudo systemctl daemon-reload
  sudo systemctl enable omnillm-relay
fi
sudo systemctl restart omnillm-relay
sleep 2
systemctl is-active omnillm-relay
curl -fsS http://127.0.0.1:8470/healthz
/opt/omnillm/omnillm version
