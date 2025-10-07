package installer

import (
	"fmt"
	"os"
	"os/exec"
)

// Cleanup removes Kubernetes installation
func Cleanup() error {
	steps := []struct {
		name string
		fn   func() error
	}{
		{"Resetting kubeadm", resetKubeadm},
		{"Removing kubectl configuration", removeKubectlConfig},
		{"Stopping Docker containers", stopDockerContainers},
		{"Removing Kubernetes packages", removeKubernetesPackages},
		{"Cleaning up directories", cleanupDirectories},
		{"Removing repository sources", removeRepoSources},
		{"Re-enabling swap", enableSwap},
	}

	for _, step := range steps {
		fmt.Printf("→ %s...\n", step.name)
		if err := step.fn(); err != nil {
			// Log error but continue cleanup
			fmt.Printf("  Warning: %s: %v\n", step.name, err)
		}
	}

	return nil
}

func resetKubeadm() error {
	cmd := exec.Command("kubeadm", "reset", "-f")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return runSudo(cmd)
}

func removeKubectlConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return os.RemoveAll(homeDir + "/.kube")
}

func stopDockerContainers() error {
	// Stop all containers
	cmd := exec.Command("bash", "-c", "docker stop $(docker ps -aq) 2>/dev/null || true")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()

	// Remove all containers
	cmd = exec.Command("bash", "-c", "docker rm $(docker ps -aq) 2>/dev/null || true")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()

	return nil
}

func removeKubernetesPackages() error {
	commands := [][]string{
		{"apt-mark", "unhold", "kubelet", "kubeadm", "kubectl"},
		{"apt-get", "purge", "-y", "kubelet", "kubeadm", "kubectl"},
		{"apt-get", "autoremove", "-y"},
	}
	
	for _, cmdArgs := range commands {
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		runSudo(cmd)
	}
	return nil
}

func cleanupDirectories() error {
	dirs := []string{
		"/etc/kubernetes",
		"/var/lib/kubelet",
		"/var/lib/etcd",
		"/etc/cni/net.d",
		"/opt/cni/bin",
	}

	for _, dir := range dirs {
		cmd := exec.Command("rm", "-rf", dir)
		runSudo(cmd)
	}
	return nil
}

func removeRepoSources() error {
	files := []string{
		"/etc/apt/sources.list.d/kubernetes.list",
		"/etc/apt/sources.list.d/docker.list",
		"/usr/share/keyrings/kubernetes-archive-keyring.gpg",
		"/usr/share/keyrings/kubernetes-apt-keyring.gpg",
		"/usr/share/keyrings/docker-archive-keyring.gpg",
	}

	for _, file := range files {
		cmd := exec.Command("rm", "-f", file)
		runSudo(cmd)
	}

	cmd := exec.Command("apt-get", "update")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	runSudo(cmd)
	
	return nil
}

func enableSwap() error {
	commands := [][]string{
		{"sed", "-i", "/^#.*swap/s/^#//", "/etc/fstab"},
		{"bash", "-c", "swapon -a 2>/dev/null || true"},
	}

	for _, cmdArgs := range commands {
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		runSudo(cmd)
	}
	return nil
}
