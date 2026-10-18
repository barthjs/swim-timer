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

if [ -n "${LOG_PATH}" ]; then
    mkdir -p "${LOG_PATH}"
    chown -R "${PUID}":"${PGID}" "${LOG_PATH}"
fi

mkdir -p "${STORAGE_PATH}"
chown -R "${PUID}":"${PGID}" "${STORAGE_PATH}"

# The application saves the configuration through a temporary file in its directory,
# so the directory itself has to be writable, not only the file.
if [ -n "${CONFIG_FILE}" ]; then
    CONFIG_DIR=$(dirname "${CONFIG_FILE}")
    mkdir -p "${CONFIG_DIR}"
    chown "${PUID}":"${PGID}" "${CONFIG_DIR}"
fi

if ls /dev/ttyUSB* /dev/ttyACM* > /dev/null 2>&1; then
    chmod 666 /dev/ttyUSB* /dev/ttyACM* 2> /dev/null || true
fi

exec su-exec "${PUID}":"${PGID}" "$@"
