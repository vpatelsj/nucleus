# Nucleus

Single-node Kubernetes cluster installer for WSL using kubeadm and Cilium CNI.

## Quick Start

```bash
# Install Kubernetes cluster
make install

# Remove cluster
make cleanup
```

## Requirements

- WSL2 (Ubuntu)
- Sudo privileges
- Systemd enabled in WSL (installer configures automatically)

## What It Does

- Installs Docker, Kubernetes (kubeadm, kubelet, kubectl)
- Initializes single-node Kubernetes cluster
- Installs Cilium CNI v1.18.2
- Configures kubectl for your user
- Untaints master node for workload scheduling

## Verification

```bash
kubectl get nodes
kubectl get pods -A
cilium status
```

For detailed documentation, see [USAGE.md](USAGE.md).