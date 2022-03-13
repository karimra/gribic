package app

import "fmt"

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
