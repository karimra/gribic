/*
Copyright © 2022 Karim Radhouani <medkarimrdi@gmail.com>


*/
package cmd

import (
	"github.com/spf13/cobra"
)

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get",
		Aliases: []string{"g"},
		Short:   "run gRIBI Get RPC",

		RunE:         gApp.GetRunE,
		SilenceUsage: true,
	}
	// init flags
	gApp.InitGetFlags(cmd)
	return cmd
}
