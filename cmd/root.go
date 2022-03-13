package cmd

import (
	"fmt"
	"os"

	"github.com/karimra/gribic/app"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var gApp = app.New()

func newRootCmd() *cobra.Command {
	gApp.RootCmd = &cobra.Command{
		Use:   "gribic",
		Short: "run gRIBI RPCs from the terminal",
		PreRun: func(cmd *cobra.Command, args []string) {
			gApp.Config.SetPersistantFlagsFromFile(cmd)
		},
		PersistentPreRunE: gApp.PreRun,
	}
	gApp.InitGlobalFlags()
	//
	gApp.RootCmd.AddCommand(
		newGetCmd(),
		newModifyCmd(),
		newFlushCmd(),
		newServerCmd(),
	)
	return gApp.RootCmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := newRootCmd().Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	err := gApp.Config.Load(gApp.Context())
	if err == nil {
		return
	}
	if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
		fmt.Fprintf(os.Stderr, "failed loading config file: %v\n", err)
	}
}
