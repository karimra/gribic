/*
Copyright © 2022 Karim Radhouani <medkarimrdi@gmail.com>


*/
package cmd

import (
	"github.com/spf13/cobra"
)

func newModifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "modify",
		Aliases:      []string{"mod", "m"},
		Short:        "run gRIBI Modify RPC",
		PreRunE:      gApp.ModifyPreRunE,
		RunE:         gApp.ModifyRunE,
		SilenceUsage: true,
	}
	gApp.InitModifyFlags(cmd)
	// init flags
	return cmd
}
