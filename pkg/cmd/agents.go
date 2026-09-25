package cmd

import "github.com/spf13/cobra"

// newAgentsCommand exposes HowlPlane's agent readiness check. Only the fixed
// `agents doctor` path is forwarded; HowlPlane owns every readiness decision.
func newAgentsCommand() *cobra.Command {
	agents := &cobra.Command{
		Use:   "agents",
		Short: "Inspect agent CLI readiness through HowlPlane",
	}
	agents.AddCommand(newHowlPlaneForward("doctor [--repo PATH] [--live] [--json] [--workspace-trust strict|prepare|bypass]",
		"Verify agent CLIs and, with --repo, workspace trust (howlplane agents doctor)", "agents", "doctor"))
	return agents
}

// newFactoryCommand exposes HowlPlane's workspace preparation for Factory.
// Only the fixed `factory prepare` path is forwarded.
func newFactoryCommand() *cobra.Command {
	factory := &cobra.Command{
		Use:   "factory",
		Short: "Prepare repositories for unattended HowlPlane Factory work",
	}
	factory.AddCommand(newHowlPlaneForward("prepare [--repo PATH] [--yes] [--revoke] [--live] [--json] [--workspace-trust strict|prepare|bypass]",
		"Authorize a repository and prepare agent workspace trust (howlplane factory prepare)", "factory", "prepare"))
	return factory
}
