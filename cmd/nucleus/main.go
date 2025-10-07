package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/vapa/nucleus/pkg/installer"
)

func main() {
	installCmd := flag.NewFlagSet("install", flag.ExitOnError)
	cleanupCmd := flag.NewFlagSet("cleanup", flag.ExitOnError)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "install":
		installCmd.Parse(os.Args[2:])
		if err := installer.Install(); err != nil {
			fmt.Fprintf(os.Stderr, "Installation failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Kubernetes master node with Cilium CNI is ready!")

	case "cleanup":
		cleanupCmd.Parse(os.Args[2:])
		if err := installer.Cleanup(); err != nil {
			fmt.Fprintf(os.Stderr, "Cleanup failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Cleanup complete! Kubernetes has been removed from the system.")

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Nucleus - Kubernetes master node installer with Cilium CNI")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  nucleus install    Install Kubernetes master node")
	fmt.Println("  nucleus cleanup    Remove Kubernetes installation")
}
