package ui

import (
	"github.com/charmbracelet/huh/spinner"
)

// RunWithSpinner runs fn while showing a spinner with the given label.
// In JSON mode, the spinner is suppressed and fn runs directly.
func RunWithSpinner(label string, jsonMode bool, fn func() error) error {
	if jsonMode {
		return fn()
	}
	var fnErr error
	if err := spinner.New().
		Title(label).
		Action(func() { fnErr = fn() }).
		Run(); err != nil {
		return err
	}
	return fnErr
}
