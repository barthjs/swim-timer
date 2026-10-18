#!/bin/sh
set -e

if [ "$1" = "remove" ] || [ "$1" = "purge" ]; then
  systemctl disable swim-timer >/dev/null 2>&1 || true
fi

systemctl daemon-reload >/dev/null 2>&1 || true

exit 0
