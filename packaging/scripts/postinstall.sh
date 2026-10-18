#!/bin/sh
set -e

if ! getent passwd swim-timer >/dev/null; then
  useradd --system --user-group --home-dir /var/lib/swim-timer --no-create-home --shell /usr/sbin/nologin swim-timer
fi

mkdir -p /etc/swim-timer /var/lib/swim-timer
chown -R swim-timer:swim-timer /etc/swim-timer /var/lib/swim-timer

systemctl daemon-reload >/dev/null 2>&1 || true

if systemctl is-enabled --quiet swim-timer 2>/dev/null; then
  systemctl restart swim-timer >/dev/null 2>&1 || true
fi

cat <<'MSG'
Swim-Timer installed.

Configure it at /etc/swim-timer/config.yaml,
then enable and start it with:

  sudo systemctl enable --now swim-timer
MSG

exit 0
