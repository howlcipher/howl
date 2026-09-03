//go:build unix

package state

import "syscall"

// processAlive checks liveness via signal 0, which the kernel validates
// without actually delivering a signal.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	if err == nil {
		return true
	}
	if err == syscall.ESRCH {
		return false
	}
	// EPERM (or anything else): the process exists but we can't signal it,
	// so it's still alive as far as staleness detection is concerned.
	return true
}
