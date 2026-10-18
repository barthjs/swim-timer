#!/bin/sh
set -e

systemctl stop swim-timer >/dev/null 2>&1 || true

exit 0
