/*
Copyright © 2022 Karim Radhouani <medkarimrdi@gmail.com>

*/
package cmd

import (
	"github.com/spf13/cobra"
)

func newServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "server",
		Aliases: []string{"serve", "s"},
		Short:   "start a gNMI server",
		PreRun: func(cmd *cobra.Command, args []string) {
			gApp.Config.SetLocalFlagsFromFile(cmd)
		},
		RunE:         gApp.RunEServer,
		SilenceUsage: true,
	}
	// init flags
	gApp.InitServerFlags(cmd)
	return cmd
}
