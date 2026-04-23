#!/usr/bin/env bash
set -euo pipefail

if [[ "$(id -u)" -ne 0 ]]; then
  echo "Please run as root." >&2
  exit 1
fi

if command -v k3s >/dev/null 2>&1; then
  echo "k3s is already installed."
  exit 0
fi

curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="server --disable traefik --disable servicelb --disable metrics-server --write-kubeconfig-mode 644" sh -

echo "k3s installed."
echo "kubectl context:"
k3s kubectl config current-context
