package cmd

// Exit codes distinguish why a command failed, so scripts calling howl can
// branch on outcome rather than parsing text output.
const (
	ExitSuccess             = 0
	ExitGeneric             = 1
	ExitCancelled           = 2
	ExitValidationFailure   = 3
	ExitMissingDependency   = 4
	ExitIntegrityFailure    = 5
	ExitInstallationFailure = 6
	ExitVerificationFailure = 7
	ExitRollbackFailure     = 8
	ExitLocked              = 9
)

// ExitCoder is implemented by errors that carry a specific exit code.
// Errors that don't implement it fall back to ExitGeneric.
type ExitCoder interface {
	ExitCode() int
}

type codedError struct {
	code int
	err  error
}

func (e *codedError) Error() string { return e.err.Error() }
func (e *codedError) Unwrap() error { return e.err }
func (e *codedError) ExitCode() int { return e.code }

// exitErr wraps err so Execute() reports the given exit code for it.
func exitErr(code int, err error) error {
	if err == nil {
		return nil
	}
	return &codedError{code: code, err: err}
}
