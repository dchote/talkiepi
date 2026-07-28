#!/bin/sh
set -e

DATA_DIR="/var/lib/talkiepi"
CONFIG_DIR="/etc/talkiepi"
LOG_DIR="/var/log/talkiepi"

mkdir -p "${DATA_DIR}" "${CONFIG_DIR}" "${LOG_DIR}"

if ! getent group talkiepi > /dev/null 2>&1; then
    groupadd --system talkiepi
fi

if ! getent passwd talkiepi > /dev/null 2>&1; then
    useradd --system --gid talkiepi --home-dir "${DATA_DIR}" --shell /usr/sbin/nologin talkiepi
fi

# Ensure talkiepi can access audio devices and GPIO.
if getent group audio > /dev/null 2>&1; then
    usermod -aG audio talkiepi || true
fi
if getent group gpio > /dev/null 2>&1; then
    usermod -aG gpio talkiepi || true
fi

chown talkiepi:talkiepi "${DATA_DIR}" "${LOG_DIR}"
chmod 750 "${DATA_DIR}" "${LOG_DIR}"
chmod 750 "${CONFIG_DIR}"

if [ ! -f "${CONFIG_DIR}/talkiepi.yaml" ]; then
    if [ -f "/usr/share/talkiepi/talkiepi.yaml" ]; then
        cp "/usr/share/talkiepi/talkiepi.yaml" "${CONFIG_DIR}/talkiepi.yaml"
        chown root:talkiepi "${CONFIG_DIR}/talkiepi.yaml"
        chmod 640 "${CONFIG_DIR}/talkiepi.yaml"
    fi
fi

if command -v systemctl > /dev/null 2>&1; then
    systemctl daemon-reload
    systemctl enable talkiepi.service || true
fi
