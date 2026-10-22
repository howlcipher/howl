package cmd

import (
	"github.com/spf13/cobra"
)

// newCreativeCommand is a process adapter. HowlPlane owns every workflow decision.
func newCreativeCommand() *cobra.Command {
	return newHowlPlaneForward("creative [run|resume|inspect|status]", "Run a HowlPlane creative workflow",
		"creative")
}
