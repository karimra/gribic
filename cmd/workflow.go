/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

func newWorkflowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workflow",
		Aliases: []string{"wf", "w"},
		Short:   "run a workflow",
		PreRun: func(cmd *cobra.Command, _ []string) {
			gApp.Config.SetLocalFlagsFromFile(cmd)
		},
		PreRunE:      gApp.WorkflowPreRunE,
		RunE:         gApp.WorkflowRunE,
		SilenceUsage: true,
	}
	// init flags
	gApp.InitWorkflowFlags(cmd)
	return cmd
}
