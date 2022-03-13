/*
Copyright © 2022 Karim Radhouani <medkarimrdi@gmail.com>

*/
package cmd

import (
	"github.com/spf13/cobra"
)

func newFlushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "flush",
		Aliases: []string{"f"},
		Short:   "run gRIBI Flush RPC",

		RunE:         gApp.FlushRunE,
		SilenceUsage: true,
	}
	// init flags
	gApp.InitFlushFlags(cmd)
	return cmd
}
