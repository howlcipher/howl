//go:build !unix

package state

// processAlive is conservative on platforms without a cheap liveness
// check: never report a lock as stale automatically, so doctor --fix
// never removes a lock it can't actually verify.
func processAlive(pid int) bool {
	return true
}
