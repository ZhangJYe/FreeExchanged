# K3s Lite Deployment

The K3s lite overlay entry point is `deploy/kustomization.yaml`.

It lives at `deploy/` so `kubectl kustomize` can read both the shared `deploy/k8s` manifests and the K3s-specific `deploy/k3s/lite` patches without disabling kustomize load restrictions.

## Build Local Images

```bash
sudo bash deploy/k3s/build-images.sh
```

The script builds backend images with the `k3s` tag and builds the web image with `VITE_APP_BASE=/free/`.

## Render Or Apply

Create the `freeexchanged-secrets` secret first, then render or apply:

```bash
kubectl kustomize deploy
kubectl apply -k deploy
```

The lite overlay exposes:

- web: `NodePort 30080`
- gateway: `NodePort 30888`

`Caddyfile.free.snippet` can proxy `/free`, `/free/v1`, and `/free/ws` to those NodePorts.
