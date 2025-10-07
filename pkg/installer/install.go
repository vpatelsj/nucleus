package installer

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Install sets up a Kubernetes master node with Cilium CNI
func Install() error {
	steps := []struct {
		name string
		fn   func() error
	}{
		{"Enabling systemd in WSL", enableSystemd},
		{"Installing prerequisites", installPrerequisites},
		{"Installing containerd", installContainerd},
		{"Installing Kubernetes tools", installKubernetes},
		{"Configuring containerd for Kubernetes", configureContainerd},
		{"Disabling swap", disableSwap},
		{"Initializing Kubernetes master", initKubeadm},
		{"Configuring kubectl", configureKubectl},
		{"Installing Cilium CNI", installCilium},
		{"Restarting container runtime", restartContainerRuntime},
		{"Untainting master node", untaintMaster},
	}

	for _, step := range steps {
		fmt.Printf("→ %s...\n", step.name)
		if err := step.fn(); err != nil {
			return fmt.Errorf("%s failed: %w", step.name, err)
		}
	}

	return nil
}

func installPrerequisites() error {
	commands := [][]string{
		{"apt-get", "update"},
		{"apt-get", "install", "-y", "apt-transport-https", "ca-certificates", "curl", "gnupg", "lsb-release"},
	}
	return runCommands(commands)
}

func enableSystemd() error {
	// Check if running in WSL
	cmd := exec.Command("grep", "-qi", "microsoft", "/proc/version")
	if err := cmd.Run(); err != nil {
		fmt.Println("  Not running in WSL, skipping systemd configuration")
		return nil
	}

	// Check if /etc/wsl.conf exists and has systemd enabled
	wslConfPath := "/etc/wsl.conf"
	content, err := os.ReadFile(wslConfPath)
	
	needsRestart := false
	if err != nil || !strings.Contains(string(content), "[boot]") || !strings.Contains(string(content), "systemd=true") {
		// Create or update /etc/wsl.conf
		wslConf := `[boot]
systemd=true
`
		cmd := exec.Command("bash", "-c", fmt.Sprintf("echo '%s' > %s", wslConf, wslConfPath))
		if err := runSudo(cmd); err != nil {
			return fmt.Errorf("failed to configure systemd: %w", err)
		}
		needsRestart = true
	}

	if needsRestart {
		fmt.Println("  ⚠ Systemd configuration updated. WSL needs to be restarted.")
		fmt.Println("  Please run 'wsl --shutdown' from Windows PowerShell, then restart WSL.")
		fmt.Println("  After restart, run 'make install' again.")
		return fmt.Errorf("WSL restart required - please shutdown WSL and restart")
	}

	fmt.Println("  Systemd is already enabled")
	return nil
}

func installContainerd() error {
	// Add Docker GPG key (needed for containerd.io package)
	cmd := exec.Command("bash", "-c", "curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg --yes")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := runSudo(cmd); err != nil {
		return err
	}

	// Add Docker repository (for containerd.io package)
	arch := getArch()
	lsbRelease := getLsbRelease()
	repoLine := fmt.Sprintf("deb [arch=%s signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu %s stable", arch, lsbRelease)
	
	cmd = exec.Command("bash", "-c", fmt.Sprintf("echo '%s' > /etc/apt/sources.list.d/docker.list", repoLine))
	if err := runSudo(cmd); err != nil {
		return err
	}

	// Install containerd
	commands := [][]string{
		{"apt-get", "update"},
		{"apt-get", "install", "-y", "containerd.io"},
	}
	return runCommands(commands)
}

func configureContainerd() error {
	fmt.Println("  Generating default containerd configuration...")
	
	// Create containerd config directory
	cmd := exec.Command("mkdir", "-p", "/etc/containerd")
	if err := runSudo(cmd); err != nil {
		return fmt.Errorf("failed to create containerd config directory: %w", err)
	}

	// Generate default config
	cmd = exec.Command("bash", "-c", "containerd config default > /etc/containerd/config.toml")
	if err := runSudo(cmd); err != nil {
		return fmt.Errorf("failed to generate containerd config: %w", err)
	}

	// Enable SystemdCgroup
	cmd = exec.Command("sed", "-i", "s/SystemdCgroup = false/SystemdCgroup = true/g", "/etc/containerd/config.toml")
	if err := runSudo(cmd); err != nil {
		return fmt.Errorf("failed to enable SystemdCgroup: %w", err)
	}

	// Restart containerd to apply configuration
	commands := [][]string{
		{"systemctl", "daemon-reload"},
		{"systemctl", "enable", "containerd"},
		{"systemctl", "restart", "containerd"},
	}
	if err := runCommands(commands); err != nil {
		return fmt.Errorf("failed to restart containerd: %w", err)
	}

	fmt.Println("  Waiting for containerd to initialize...")
	time.Sleep(5 * time.Second)
	
	return nil
}

func installKubernetes() error {
	// Add Kubernetes GPG key (new repository)
	cmd := exec.Command("bash", "-c", "curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.31/deb/Release.key | gpg --dearmor -o /usr/share/keyrings/kubernetes-apt-keyring.gpg --yes")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := runSudo(cmd); err != nil {
		return err
	}

	// Add Kubernetes repository (new repository)
	repoLine := "deb [signed-by=/usr/share/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.31/deb/ /"
	cmd = exec.Command("bash", "-c", fmt.Sprintf("echo '%s' > /etc/apt/sources.list.d/kubernetes.list", repoLine))
	if err := runSudo(cmd); err != nil {
		return err
	}

	// Install Kubernetes tools
	commands := [][]string{
		{"apt-get", "update"},
		{"apt-get", "install", "-y", "kubelet", "kubeadm", "kubectl"},
		{"apt-mark", "hold", "kubelet", "kubeadm", "kubectl"},
	}
	return runCommands(commands)
}

func disableSwap() error {
	commands := [][]string{
		{"swapoff", "-a"},
		{"sed", "-i", "/ swap / s/^/#/", "/etc/fstab"},
		// Fix mount propagation for WSL2 (required for Cilium)
		{"mount", "--make-shared", "/"},
	}
	return runCommands(commands)
}

func initKubeadm() error {
	cmd := exec.Command("kubeadm", "init", 
		"--pod-network-cidr=10.0.0.0/16", 
		"--cri-socket=unix:///run/containerd/containerd.sock")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return runSudo(cmd)
}

func configureKubectl() error {
	// Get the actual user (not root when running with sudo)
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser == "" {
		sudoUser = os.Getenv("USER")
	}
	
	// Get user's home directory
	var homeDir string
	if sudoUser != "" && sudoUser != "root" {
		homeDir = "/home/" + sudoUser
	} else {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return err
		}
	}

	kubeDir := homeDir + "/.kube"
	
	// Create .kube directory as the actual user
	cmd := exec.Command("mkdir", "-p", kubeDir)
	if sudoUser != "" && sudoUser != "root" {
		cmd = exec.Command("sudo", "-u", sudoUser, "mkdir", "-p", kubeDir)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create .kube directory: %w", err)
	}

	// Copy admin.conf without interactive prompt
	cmd = exec.Command("cp", "-f", "/etc/kubernetes/admin.conf", kubeDir+"/config")
	if err := runSudo(cmd); err != nil {
		return err
	}

	// Get the actual user's UID and GID
	var uid, gid string
	if sudoUser != "" && sudoUser != "root" {
		cmd = exec.Command("id", "-u", sudoUser)
		uidBytes, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to get user UID: %w", err)
		}
		uid = strings.TrimSpace(string(uidBytes))

		cmd = exec.Command("id", "-g", sudoUser)
		gidBytes, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to get user GID: %w", err)
		}
		gid = strings.TrimSpace(string(gidBytes))
	} else {
		uid = fmt.Sprintf("%d", os.Getuid())
		gid = fmt.Sprintf("%d", os.Getgid())
	}

	cmd = exec.Command("chown", fmt.Sprintf("%s:%s", uid, gid), kubeDir+"/config")
	if err := runSudo(cmd); err != nil {
		return err
	}

	fmt.Printf("  Configured kubectl for user %s at %s/.kube/config\n", sudoUser, homeDir)
	return nil
}

func installCilium() error {
	// Install Cilium CLI first
	fmt.Println("  Installing Cilium CLI...")
	
	// Determine architecture
	arch := "amd64"
	cmd := exec.Command("uname", "-m")
	output, _ := cmd.Output()
	if strings.Contains(string(output), "aarch64") {
		arch = "arm64"
	}
	
	// Get latest stable version
	cmd = exec.Command("bash", "-c", "curl -s https://raw.githubusercontent.com/cilium/cilium-cli/main/stable.txt")
	versionOutput, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get Cilium CLI version: %w", err)
	}
	version := strings.TrimSpace(string(versionOutput))
	
	// Download Cilium CLI
	downloadURL := fmt.Sprintf("https://github.com/cilium/cilium-cli/releases/download/%s/cilium-linux-%s.tar.gz", version, arch)
	cmd = exec.Command("bash", "-c", fmt.Sprintf("curl -L --fail --remote-name-all %s{,.sha256sum}", downloadURL))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to download Cilium CLI: %w", err)
	}

	// Verify checksum
	tarFile := fmt.Sprintf("cilium-linux-%s.tar.gz", arch)
	cmd = exec.Command("sha256sum", "--check", tarFile+".sha256sum")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run() // Ignore error, continue anyway

	// Extract to /usr/local/bin
	cmd = exec.Command("tar", "xzvfC", tarFile, "/usr/local/bin")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := runSudo(cmd); err != nil {
		return fmt.Errorf("failed to extract Cilium CLI: %w", err)
	}

	// Cleanup downloaded files
	cmd = exec.Command("rm", "-f", tarFile, tarFile+".sha256sum")
	cmd.Run()

	// Get kubeconfig path
	kubeconfigPath := "/etc/kubernetes/admin.conf"

	// Install Cilium using the CLI with version 1.18.2 (latest stable)
	fmt.Println("  Installing Cilium CNI...")
	cmd = exec.Command("cilium", "install", "--version", "1.18.2")
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfigPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func untaintMaster() error {
	kubeconfigPath := "/etc/kubernetes/admin.conf"
	cmd := exec.Command("kubectl", "taint", "nodes", "--all", "node-role.kubernetes.io/control-plane-")
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfigPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Ignore error if taint doesn't exist
	cmd.Run()
	return nil
}

func restartContainerRuntime() error {
	fmt.Println("  Restarting containerd...")
	commands := [][]string{
		{"systemctl", "restart", "containerd"},
	}
	if err := runCommands(commands); err != nil {
		return err
	}
	
	fmt.Println("  Restarting kubelet...")
	commands = [][]string{
		{"systemctl", "restart", "kubelet"},
	}
	if err := runCommands(commands); err != nil {
		return err
	}
	
	// Wait for kubelet to settle
	fmt.Println("  Waiting for kubelet to initialize CNI...")
	time.Sleep(15 * time.Second)
	
	return nil
}

// Helper functions

func runCommands(commands [][]string) error {
	for _, cmdArgs := range commands {
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := runSudo(cmd); err != nil {
			return err
		}
	}
	return nil
}

func runSudo(cmd *exec.Cmd) error {
	// Prepend sudo if not running as root
	if os.Geteuid() != 0 {
		args := append([]string{cmd.Path}, cmd.Args[1:]...)
		cmd = exec.Command("sudo", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}

func getArch() string {
	cmd := exec.Command("dpkg", "--print-architecture")
	output, _ := cmd.Output()
	return strings.TrimSpace(string(output))
}

func getLsbRelease() string {
	cmd := exec.Command("lsb_release", "-cs")
	output, _ := cmd.Output()
	return strings.TrimSpace(string(output))
}
