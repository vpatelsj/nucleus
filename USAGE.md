# Usage Instructions

## Prerequisites
- WSL2 (Ubuntu recommended)
- Go 1.21+ (installed automatically if not present)
- make
- sudo privileges

## Installation Steps

1. Clone this repository or copy the files to your WSL instance.
2. Run the installation:
   ```bash
   make install
   ```
3. If you are prompted to restart your shell (for Docker group changes), do so and re-run the command if needed.

## Manual Build and Run

If you prefer to build and run manually:

```bash
# Build the binary
make build

# Install Kubernetes
sudo ./bin/nucleus install

# Or cleanup
sudo ./bin/nucleus cleanup
```

## Notes
- The script disables swap, which is required for kubeadm.
- Cilium is installed as the CNI plugin.
- The master node is untainted to allow pod scheduling (single-node setup).

## Cleanup

To completely remove the Kubernetes installation:

```bash
make cleanup
```

**Note:** The cleanup process removes Kubernetes components but keeps Docker installed by default.

## Troubleshooting

### General Issues
- If you encounter issues, ensure Docker is running and you have network connectivity.
- For Cilium status, run:
  ```bash
  kubectl -n kube-system get pods | grep cilium
  ```
- For Kubernetes status:
  ```bash
  kubectl get nodes
  kubectl get pods -A
  ```

### WSL2-Specific Issues

**Cilium pods stuck in CreateContainerError:**
This is usually due to mount propagation issues in WSL2. Fix with:
```bash
sudo mount --make-shared /
sudo systemctl restart containerd
kubectl delete pod -n kube-system $(kubectl get pods -n kube-system | grep cilium | grep -v operator | awk '{print $1}')
```

**Node shows as NotReady:**
Wait for Cilium CNI to fully initialize (may take 1-2 minutes). Check with:
```bash
kubectl get nodes
kubectl -n kube-system get pods
```
