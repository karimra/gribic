package app

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func (a *App) handleErrs(errs []error) error {
	numErrors := len(errs)
	if numErrors > 0 {
		for _, e := range errs {
			a.Logger.Debug(e)
		}
		return fmt.Errorf("there was %d error(s)", numErrors)
	}
	return nil
}

func flagIsSet(cmd *cobra.Command, name string) bool {
	if cmd == nil {
		return false
	}
	var isSet bool
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Name == name && f.Changed {
			isSet = true
			return
		}
	})
	return isSet
}
