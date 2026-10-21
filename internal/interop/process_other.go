//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris && !windows

package interop

import "os/exec"
import "errors"

// Unsupported process-tree platforms fail closed before executing a descriptor.
func configureProcessTree(cmd *exec.Cmd) (func() error, func(), error) {
	return nil, nil, errors.New("bounded descriptor process trees are unsupported on this platform")
}
