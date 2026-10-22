//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package interop

import (
	"os/exec"
	"syscall"
)

// Descriptor discovery owns this process group, including ordinary descendants.
// This is a lifecycle bound, not a sandbox against a process escaping its group.
func configureProcessTree(cmd *exec.Cmd) (func() error, func(), error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	kill := func() {
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
	}
	cmd.Cancel = func() error { kill(); return nil }
	return func() error { return nil }, kill, nil
}
