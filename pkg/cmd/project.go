package cmd

import (
	planeCli "github.com/howlcipher/howlplane/pkg/cli"
	"github.com/spf13/cobra"
)

func newProjectCommand() *cobra.Command {
	planeCmd := planeCli.NewPlaneCommand()
	for _, sub := range planeCmd.Commands() {
		if sub.Name() == "project" {
			return sub
		}
	}

	return &cobra.Command{
		Use:   "project",
		Short: "Manage and inspect project integrations",
	}
}
