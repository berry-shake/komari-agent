package server

import (
	"os/exec"
	"strconv"
)

func containTask(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		// Kill the process tree so children cannot outlive a timed-out task.
		_ = exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
		return cmd.Process.Kill()
	}
}
