package cmd

import (
	"os/exec"
)

func runCommand(command string) {
	cmd := exec.Command("sh", "-c", command)
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}
