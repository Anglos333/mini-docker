package cmd

import (
	"mini-docker/pkg/container"
	"mini-docker/pkg/network"
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/cobra"
)

var childCmd = &cobra.Command{
	Use:   "child",
	Short: "Child container process",
	Run: func(cmd *cobra.Command, args []string) {
		containerIP, vethChild, gatewayIP, mergedDir := args[0], args[1], args[2], args[3]

		readPipe := os.NewFile(3, "pipe")
		msg := make([]byte, 2)
		readPipe.Read(msg)
		readPipe.Close()

		syscall.Sethostname([]byte("mini-docker-container"))
		network.ConfigContainerNetwork(vethChild, containerIP, gatewayIP)
		container.PivotRoot(mergedDir)

		shell := exec.Command("/bin/sh", "-i")
		shell.Stdin, shell.Stdout, shell.Stderr = os.Stdin, os.Stdout, os.Stderr
		shell.Run()
	},
}
