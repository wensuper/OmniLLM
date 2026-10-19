package provider

import (
	"os/exec"
	"syscall"
)

// ownGroup runs cmd in a process group of its own and makes a cancel kill
// the whole group: the subscription programs start helpers of their own.
func ownGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
