#!/usr/bin/env sh

set -e

CURRENT_GID=$(getent group "${APP_USER}" | cut -d: -f3)
if [ "$PGID" != "$CURRENT_GID" ]; then
    groupmod -o -g "$PGID" "${APP_USER}"
fi

CURRENT_UID=$(id -u "${APP_USER}")
if [ "$PUID" != "$CURRENT_UID" ]; then
    usermod -o -u "$PUID" "${APP_USER}"
fi

chown "${PUID}":"${PGID}" /app

mkdir -p "${PRESET_DIR}" && chown -R "${PUID}":"${PGID}" "${PRESET_DIR}"
mkdir -p "${RACE_DATA_DIR}" && chown -R "${PUID}":"${PGID}" "${RACE_DATA_DIR}"

exec su-exec "${PUID}":"${PGID}" "$@"
