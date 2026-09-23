package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/howlcipher/howl/internal/platform"
	"github.com/spf13/cobra"
)

// newOrchestrateCommand is a process adapter. HowlPlane owns every workflow decision.
func newOrchestrateCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "orchestrate [goal|resume|inspect|discard]",
		Short:              "Run a HowlPlane orchestration session",
		DisableFlagParsing: true,
		Args:               cobra.ArbitraryArgs,
		RunE: func(command *cobra.Command, args []string) error {
			binary, err := exec.LookPath("howlplane")
			if err != nil {
				paths, pathErr := platform.DefaultPaths()
				if pathErr != nil {
					return pathErr
				}
				candidate := paths.ComponentBinLink("howlplane")
				if info, statErr := os.Stat(candidate); statErr == nil && info.Mode().IsRegular() && info.Mode()&0111 != 0 {
					binary = candidate
				} else {
					return fmt.Errorf("HowlPlane is not installed; install it with `howl install howlplane`")
				}
			}
			binary, err = filepath.Abs(binary)
			if err != nil {
				return err
			}
			child := exec.CommandContext(command.Context(), binary, append([]string{"orchestrate"}, args...)...)
			child.Stdin = command.InOrStdin()
			child.Stdout = command.OutOrStdout()
			child.Stderr = command.ErrOrStderr()
			if err := child.Run(); err != nil {
				if exit, ok := err.(*exec.ExitError); ok {
					return &forwardedExit{code: exit.ExitCode()}
				}
				return fmt.Errorf("launch HowlPlane: %w", err)
			}
			return nil
		},
	}
}

type forwardedExit struct{ code int }

func (e *forwardedExit) Error() string { return "" }
func (e *forwardedExit) ExitCode() int { return e.code }
